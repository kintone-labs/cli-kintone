package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kintone-labs/go-kintone"
	"golang.org/x/text/transform"
)

// SubRecord structure
type SubRecord struct {
	Id     uint64
	Fields map[string]interface{}
}

func getReader(reader io.Reader) io.Reader {
	readerWithoutBOM := removeBOMCharacter(reader)

	encoding := getEncoding()
	if encoding == nil {
		return readerWithoutBOM
	}
	return transform.NewReader(readerWithoutBOM, encoding.NewDecoder())
}

// delete specific records
func deleteRecords(app *kintone.App, query string) error {
	var lastID uint64
	for {
		ids := make([]uint64, 0, IMPORT_ROW_LIMIT)

		r := regexp.MustCompile(`limit\s+\d+`)
		var _query string
		if r.MatchString(query) {
			_query = query
		} else {
			_query = query + fmt.Sprintf(" limit %v", IMPORT_ROW_LIMIT)
		}
		records, err := app.GetRecords([]string{"$id"}, _query)
		if err != nil {
			return err
		}

		if len(records) == 0 {
			break
		}

		for _, record := range records {
			id := record.Id()
			ids = append(ids, id)
		}

		err = app.DeleteRecords(ids)
		if err != nil {
			return err
		}

		if len(records) < IMPORT_ROW_LIMIT {
			break
		}
		if lastID == ids[0] {
			// prevent an inifinite loop
			return fmt.Errorf("Unexpected error occured during deleting")
		}
		lastID = ids[0]
	}

	return nil
}
func getSubRecord(tableName string, tables map[string]*SubRecord) *SubRecord {
	table := tables[tableName]
	if table == nil {
		fields := make(map[string]interface{})
		table = &SubRecord{0, fields}
		tables[tableName] = table
	}

	return table
}
func addSubField(app *kintone.App, column *Column, col string, table *SubRecord) error {
	if len(col) == 0 {
		return nil
	}

	if column.Type == kintone.FT_FILE {
		field, err := uploadFiles(app, col)
		if err != nil {
			return err
		}
		if field != nil {
			table.Fields[column.Code] = field
		}
	} else {
		field := getField(column.Type, col)
		if field != nil {
			table.Fields[column.Code] = field
		}
	}
	return nil
}
func importFromCSV(app *kintone.App, _reader io.Reader) error {

	reader := csv.NewReader(getReader(_reader))

	head := true
	var columns Columns

	var nextRowImport uint64
	nextRowImport = config.Line
	bulkRequests := &BulkRequests{}
	// retrieve field list
	fields, err := getFields(app)
	if err != nil {
		return err
	}

	if config.DeleteAll {
		err = deleteRecords(app, config.Query)
		if err != nil {
			return err
		}
		config.DeleteAll = false
	}

	keyField := ""
	hasTable := false
	var peeked *[]string
	var rowNumber uint64
	for rowNumber = 1; ; rowNumber++ {
		var err error
		var row []string
		if peeked == nil {
			row, err = reader.Read()
			if err == io.EOF {
				rowNumber--
				break
			} else if err != nil {
				return err
			}
		} else {
			row = *peeked
			peeked = nil
		}
		if head && columns == nil {
			columns = make([]*Column, 0)
			for _, col := range row {
				re := regexp.MustCompile("^(.*)\\[(.*)\\]$")
				match := re.FindStringSubmatch(col)
				if match != nil {
					// for backward compatible
					column := &Column{Code: match[1], Type: match[2]}
					columns = append(columns, column)
					col = column.Code
				} else {
					if len(col) > 0 && col[0] == '*' {
						col = col[1:]
						keyField = col
					}
					column := getColumn(col, fields)
					if column.IsSubField {
						if row[0] == "" || row[0] == "*" {
							hasTable = true
						}
					}
					columns = append(columns, column)
				}
			}
			head = false
		} else {
			if rowNumber < config.Line {
				continue
			}
			var id uint64
			var err error
			record := make(map[string]interface{})

			for {
				tables := make(map[string]*SubRecord)
				for i, col := range row {
					column := columns[i]
					if column.IsSubField {
						table := getSubRecord(column.Table, tables)
						err := addSubField(app, column, col, table)
						if err != nil {
							return err
						}
					} else if column.Type == kintone.FT_SUBTABLE {
						if col != "" {
							subID, _ := strconv.ParseUint(col, 10, 64)
							table := getSubRecord(column.Code, tables)
							table.Id = subID
						}
					} else {
						if hasTable && row[0] != "*" {
							continue
						}
						if column.Code == "$id" {
							if col != "" {
								id, _ = strconv.ParseUint(col, 10, 64)
							}
						} else if column.Code == "$revision" {

						} else if column.Type == kintone.FT_FILE {
							field, err := uploadFiles(app, col)
							if err != nil {
								return err
							}
							if field != nil {
								record[column.Code] = field
							}
						} else {
							if column.Code == keyField && col == "" {
							} else {
								field := getField(column.Type, col)
								if field != nil {
									record[column.Code] = field
								}
							}
						}
					}
				}
				for key, table := range tables {
					if record[key] == nil {
						record[key] = getField(kintone.FT_SUBTABLE, "")
					}

					stf := record[key].(kintone.SubTableField)
					stf = append(stf, kintone.NewRecordWithId(table.Id, table.Fields))
					record[key] = stf
				}

				if !hasTable {
					break
				}
				row, err = reader.Read()
				if err == io.EOF {
					break
				} else if err != nil {
					return err
				}
				if len(row) > 0 && row[0] == "*" {
					peeked = &row
					break
				}
			}

			_, hasKeyField := record[keyField]
			if id != 0 || (keyField != "" && hasKeyField) {
				setRecordUpdatable(record, columns)
				err = bulkRequests.ImportDataUpdate(app, kintone.NewRecordWithId(id, record), keyField)
				if err != nil {
					log.Fatalln(err)
				}
			} else {
				err = bulkRequests.ImportDataInsert(app, kintone.NewRecord(record))
				if err != nil {
					log.Fatalln(err)
				}
			}
			if (rowNumber-nextRowImport+1)%(ConstBulkRequestLimitRecordOption) == 0 {
				showTimeLog()
				fmt.Printf("Start from lines: %d - %d", nextRowImport, rowNumber)

				resp, err := bulkRequests.Request(app)
				bulkRequests.HandelResponse(resp, err, nextRowImport, rowNumber)

				bulkRequests.Requests = bulkRequests.Requests[:0]
				nextRowImport = rowNumber + 1

			}
		}
	}
	if len(bulkRequests.Requests) > 0 {
		showTimeLog()
		fmt.Printf("Start from lines: %d - %d", nextRowImport, rowNumber)
		resp, err := bulkRequests.Request(app)
		bulkRequests.HandelResponse(resp, err, nextRowImport, rowNumber)
	}
	showTimeLog()
	fmt.Printf("DONE\n")

	return nil
}
func setRecordUpdatable(record map[string]interface{}, columns Columns) {
	for _, col := range columns {
		switch col.Type {
		case
			kintone.FT_CREATOR,
			kintone.FT_MODIFIER,
			kintone.FT_CTIME,
			kintone.FT_MTIME:
			delete(record, col.Code)
		}
	}
}
func uploadFiles(app *kintone.App, value string) (kintone.FileField, error) {
	if config.FileDir == "" {
		return nil, nil
	}

	var ret kintone.FileField = []kintone.File{}
	value = strings.TrimSpace(value)
	if value == "" {
		return ret, nil
	}

	files := strings.Split(value, "\n")
	for _, file := range files {
		var path string
		if filepath.IsAbs(file) {
			path = file
		} else {
			path = filepath.Join(config.FileDir, file)
		}
		fileKey, err := uploadFile(app, path)
		if err != nil {
			return nil, err
		}
		ret = append(ret, kintone.File{FileKey: fileKey})
	}
	return ret, nil
}

func uploadFile(app *kintone.App, filePath string) (string, error) {
	fi, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer fi.Close()

	fileinfo, err := fi.Stat()

	if err != nil {
		return "", err
	}

	if fileinfo.Size() > 10*1024*1024 {
		return "", fmt.Errorf("%s file must be less than 10 MB", filePath)
	}

	fileKey, err := app.Upload(path.Base(filePath), "application/octet-stream", fi)
	return fileKey, err
}

func getField(fieldType string, value string) interface{} {
	switch fieldType {
	case kintone.FT_SINGLE_LINE_TEXT:
		return kintone.SingleLineTextField(value)
	case kintone.FT_MULTI_LINE_TEXT:
		return kintone.MultiLineTextField(value)
	case kintone.FT_RICH_TEXT:
		return kintone.RichTextField(value)
	case kintone.FT_DECIMAL:
		return kintone.DecimalField(value)
	case kintone.FT_CALC:
		return nil
	case kintone.FT_CHECK_BOX:
		if len(value) == 0 {
			return kintone.CheckBoxField([]string{})
		}
		return kintone.CheckBoxField(strings.Split(value, "\n"))
	case kintone.FT_RADIO:
		return kintone.RadioButtonField(value)
	case kintone.FT_SINGLE_SELECT:
		if len(value) == 0 {
			return kintone.SingleSelectField{Valid: false}
		}
		return kintone.SingleSelectField{String: value, Valid: true}

	case kintone.FT_MULTI_SELECT:
		if len(value) == 0 {
			return kintone.MultiSelectField([]string{})
		}
		return kintone.MultiSelectField(strings.Split(value, "\n"))

	case kintone.FT_FILE:
		return nil
	case kintone.FT_LINK:
		return kintone.LinkField(value)
	case kintone.FT_DATE:
		if value == "" {
			return kintone.DateField{Valid: false}
		}
		dt, err := time.Parse("2006-01-02", value)
		if err == nil {
			return kintone.DateField{Date: dt, Valid: true}
		}
		dt, err = time.Parse("2006/1/2", value)
		if err == nil {
			return kintone.DateField{Date: dt, Valid: true}
		}

	case kintone.FT_TIME:
		if value == "" {
			return kintone.TimeField{Valid: false}
		}
		dt, err := time.Parse("15:04:05", value)
		if err == nil {
			return kintone.TimeField{Time: dt, Valid: true}
		}

	case kintone.FT_DATETIME:
		if value == "" {
			return kintone.DateTimeField{Valid: false}
		}
		dt, err := time.Parse(time.RFC3339, value)
		if err == nil {
			return kintone.DateTimeField{Time: dt, Valid: true}
		}
	case kintone.FT_USER:
		users := strings.Split(value, "\n")
		var ret kintone.UserField = []kintone.User{}
		for _, user := range users {
			if len(strings.TrimSpace(user)) > 0 {
				ret = append(ret, kintone.User{Code: user})
			}
		}
		return ret
	case kintone.FT_ORGANIZATION:
		organizations := strings.Split(value, "\n")
		var ret kintone.OrganizationField = []kintone.Organization{}
		for _, organization := range organizations {
			if len(strings.TrimSpace(organization)) > 0 {
				ret = append(ret, kintone.Organization{Code: organization})
			}
		}
		return ret
	case kintone.FT_GROUP:
		groups := strings.Split(value, "\n")
		var ret kintone.GroupField = []kintone.Group{}
		for _, group := range groups {
			if len(strings.TrimSpace(group)) > 0 {
				ret = append(ret, kintone.Group{Code: group})
			}
		}
		return ret
	case kintone.FT_CATEGORY:
		return nil
	case kintone.FT_STATUS:
		return nil
	case kintone.FT_RECNUM:
		return nil
	case kintone.FT_ASSIGNEE:
		return nil
	case kintone.FT_CREATOR:
		return kintone.CreatorField{Code: value}
	case kintone.FT_MODIFIER:
		return kintone.ModifierField{Code: value}
	case kintone.FT_CTIME:
		dt, err := time.Parse(time.RFC3339, value)
		if err == nil {
			return kintone.CreationTimeField(dt)
		}
	case kintone.FT_MTIME:
		dt, err := time.Parse(time.RFC3339, value)
		if err == nil {
			return kintone.ModificationTimeField(dt)
		}
	case kintone.FT_SUBTABLE:
		sr := make([]*kintone.Record, 0)
		return kintone.SubTableField(sr)
	case "UNKNOWN":
		return nil
	}

	return kintone.SingleLineTextField(value)

}
