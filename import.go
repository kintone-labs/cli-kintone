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

// set column information from fieldinfo
func setColumn(code string, column *Column, fields map[string]*kintone.FieldInfo) {
	// initialize values
	column.Code = code
	column.IsSubField = false
	column.Table = ""

	if code == "$id" {
		column.Type = "__ID__"
		return
	} else {
		// is this code the one of sub field?
		for _, val := range fields {
			if val.Code == code {
				column.Type = val.Type
				return
			}
			if val.Type == "SUBTABLE" {
				for _, subField := range val.Fields {
					if subField.Code == code {
						column.IsSubField = true
						column.Type = subField.Type
						column.Table = val.Code
						return
					}
				}
			}
		}
	}

	// the code is not found
	column.Type = "UNKNOWN"
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
	var columns []Column

	// retrieve field list
	fields, err := getFields(app)
	if err != nil {
		return err
	}

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		//fmt.Printf("%#v", row)
		if head && columns == nil {
			columns =make([]Column, len(row))
			for i, col := range row {
				re := regexp.MustCompile("^(.*)\\[(.*)\\]$")
				match := re.FindStringSubmatch(col)
				if match != nil {
					// for backward compatible
					columns[i].Code = match[1]
					columns[i].Type = match[2]
					col = columns[i].Code
				} else {
					setColumn(col, &columns[i], fields)
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

			for i, col := range row {
				fieldName := columns[i].Code
				if fieldName == "$id" {
					id, err = strconv.ParseUint(col, 10, 64)
					if err != nil {
						return fmt.Errorf("Invalid record ID: %v", col)
					}
				} else {
					field := getField(columns[i].Type, col, updating)
					if field != nil {
						record[fieldName] = field
					}
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
		return nil
	case "UNKNOWN":
		return nil
	}

	return kintone.SingleLineTextField(value)

}
