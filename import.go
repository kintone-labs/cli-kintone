package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"strconv"
	"time"

	"github.com/ryokdy/go-kintone"
	"golang.org/x/text/transform"
)

func getReader(file *os.File) io.Reader {
	encoding := getEncoding()
	if (encoding == nil) {
		return file
	}
	return transform.NewReader(file, encoding.NewDecoder())
}

func addSubField(column *Column, col string, tables map[string]map[string]interface{}) {
	if len(col) == 0 {
		return
	}

	table := tables[column.Table]
	if table == nil {
		table = make(map[string]interface{})
		tables[column.Table] = table
	}

	field := getField(column.Type, col, true)
	if field != nil {
		table[column.Code] = field
	}
}

func readCsv(app *kintone.App, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(getReader(file))

	head := true
	updating := false
	records := make([]*kintone.Record, 0, ROW_LIMIT)
	var columns Columns

	// retrieve field list
	fields, err := getFields(app)
	if err != nil {
		return err
	}

	hasTable := false
	var peeked *[]string
	for {
		var err error
		var row []string
		if peeked == nil {
			row, err = reader.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}
		} else {
			row = *peeked
			peeked = nil
		}
		//fmt.Printf("%#v", row)
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
					column := getColumn(col, fields)
					if column.IsSubField {
						hasTable = true
					}
					columns = append(columns, column)
				}
				if col == "$id" {
					updating = true
				}
			}
			head = false
		} else {
			var id uint64 = 0
			var err error
			record := make(map[string]interface{})

			for {
				tables := make(map[string]map[string]interface{})
				for i, col := range row {
					column := columns[i]
					if column.IsSubField {
						addSubField(column, col, tables)
					} else {
						if hasTable && row[0] != "*" {
							continue
						}
						if column.Code == "$id" {
							id, err = strconv.ParseUint(col, 10, 64)
							if err != nil {
								return fmt.Errorf("Invalid record ID: %v", col)
							}
						} else if column.Code == "$revision" {
						} else {
							field := getField(column.Type, col, updating)
							if field != nil {
								record[column.Code] = field
							}
						}
					}
				}
				for key, table := range tables {
					if record[key] == nil {
						record[key] = getField(kintone.FT_SUBTABLE, "", false)
					}

					stf := record[key].(kintone.SubTableField)
					stf = append(stf, kintone.NewRecord(table))
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

			if updating {
				records = append(records, kintone.NewRecordWithId(id, record))
			} else {
				records = append(records, kintone.NewRecord(record))
			}
			if len(records) >= ROW_LIMIT {
				upsert(app, records[:], updating)
				records = make([]*kintone.Record, 0, ROW_LIMIT)
			}
		}
	}
	if len(records) > 0 {
		return upsert(app, records[:], updating)
	}

	return nil
}

func upsert(app *kintone.App, recs []*kintone.Record, updating bool)  error {
	var err error
	if updating {
		err = app.UpdateRecords(recs, true)
	} else {
		if config.deleteAll {
			deleteRecords(app)
			config.deleteAll = false
		}
		_, err = app.AddRecords(recs)
	}

	return err
}

// delete all records
func deleteRecords(app *kintone.App) error {
	var lastId uint64 = 0
	for {
		ids := make([]uint64, 0, ROW_LIMIT)
		records, err := getRecords(app, []string{"$id"}, 0)
		if err != nil {
			return err
		}
		for _, record := range records {
			id := record.Id()
			ids = append(ids, id)
		}

		err = app.DeleteRecords(ids)
		if err != nil {
			return err
		}

		if len(records) < ROW_LIMIT {
			break
		}
		if lastId == ids[0] {
			// prevent an inifinite loop
			return fmt.Errorf("Unexpected error occured during deleting")
		}
		lastId = ids[0]
	}

	return nil
}

func getField(fieldType string, value string, updating bool) interface{} {
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
		} else {
			return kintone.CheckBoxField(strings.Split(value, "\n"))
		}
	case kintone.FT_RADIO:
		return kintone.RadioButtonField(value)
	case kintone.FT_SINGLE_SELECT:
		if len(value) == 0 {
			return kintone.SingleSelectField{Valid: false}
		} else {
			return kintone.SingleSelectField{value, true}
		}
	case kintone.FT_MULTI_SELECT:
		if len(value) == 0 {
			return kintone.MultiSelectField([]string{})
		} else {
			return kintone.MultiSelectField(strings.Split(value, "\n"))
		}
	case kintone.FT_FILE:
		return nil
	case kintone.FT_LINK:
		return kintone.LinkField(value)
	case kintone.FT_DATE:
		if value == "" {
			return kintone.DateField{Valid: false}
		} else {
			dt, err := time.Parse("2006-01-02", value)
			if err == nil {
				return kintone.DateField{dt, true}
			}
			dt, err = time.Parse("2006/1/2", value)
			if err == nil {
				return kintone.DateField{dt, true}
			}
		}
	case kintone.FT_TIME:
		if value == "" {
			return kintone.TimeField{Valid: false}
		} else {
			dt, err := time.Parse("15:04:05", value)
			if err == nil {
				return kintone.TimeField{dt, true}
			}
		}
	case kintone.FT_DATETIME:
		if value == "" {
			return kintone.DateTimeField{Valid: false}
		} else {
			dt, err := time.Parse(time.RFC3339, value)
			if err == nil {
				return kintone.DateTimeField{dt, true}
			}
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
	case kintone.FT_CATEGORY:
		return nil
	case kintone.FT_STATUS:
		return nil
	case kintone.FT_RECNUM:
		return nil
	case kintone.FT_ASSIGNEE:
		return nil
	case kintone.FT_CREATOR:
		if updating {
			return nil
		} else {
			return kintone.CreatorField{Code: value}
		}
	case kintone.FT_MODIFIER:
		if updating {
			return nil
		} else {
			return kintone.ModifierField{Code: value}
		}
	case kintone.FT_CTIME:
		if updating {
			return nil
		} else {
			dt, err := time.Parse(time.RFC3339, value)
			if err == nil {
				return kintone.CreationTimeField(dt)
			}
		}
	case kintone.FT_MTIME:
		if updating {
			return nil
		} else {
			dt, err := time.Parse(time.RFC3339, value)
			if err == nil {
				return kintone.ModificationTimeField(dt)
			}
		}
	case kintone.FT_SUBTABLE:
		sr := make([]*kintone.Record, 0)
		return kintone.SubTableField(sr)
	case "UNKNOWN":
		return nil
	}

	return kintone.SingleLineTextField(value)

}
