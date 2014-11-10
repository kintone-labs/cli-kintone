package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/cybozu/go-kintone"
	"golang.org/x/text/transform"
)


func getRecords(app *kintone.App, fields []string, offset int64) ([]*kintone.Record, error) {

	newQuery := config.query + fmt.Sprintf(" limit %v offset %v", ROW_LIMIT, offset)
	records, err := app.GetRecords(config.fields, newQuery)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func getWriter() io.Writer {
	encoding := getEncoding()
	if (encoding == nil) {
		return os.Stdout
	}
	return transform.NewWriter(os.Stdout, encoding.NewEncoder())
}

func writeJson(app *kintone.App) error {
	i := 0
	offset := int64(0)
	writer := getWriter()
	
	fmt.Fprint(writer, "{\"records\": [\n")
	for ;;offset += ROW_LIMIT {
		records, err := getRecords(app, config.fields, offset)
		if err != nil {
			return err
		}
		for _, record := range records {
			if i > 0 {
				fmt.Fprint(writer, ",\n")
			}			
			jsonArray, _ := record.MarshalJSON()
			json := string(jsonArray)
			fmt.Fprint(writer, json)
			i += 1
		}
		if len(records) < ROW_LIMIT {
			break
		}
	}
	fmt.Fprint(writer, "\n]}")

	return nil
}

func writeCsv(app *kintone.App) error {
	i := 0
	offset := int64(0)
	writer := getWriter()
	var fields []string

	for ;;offset += ROW_LIMIT {
		records, err := getRecords(app, config.fields, offset)
		if err != nil {
			return err
		}
		
		for _, record := range records {
			if i == 0 {
				if config.fields == nil {
					tmpFields := make([]string, 0, len(record.Fields))
					for key, _ := range record.Fields {
						tmpFields = append(tmpFields, key)
					}
					sort.Strings(tmpFields)
					fields = make([]string, 0, len(record.Fields) + 1);
					fields = append(fields, "$id")
					fields = append(fields, tmpFields...);
				} else {
					fields = config.fields
				}
				j := 0
				for _, f := range fields {
					if j > 0 {
						fmt.Fprint(writer, ",");
					}
					var col string
					if f == "$id" {
						col = kintone.FT_ID
					} else if f == "$revision" {
						col =kintone.FT_REVISION
					} else {
						col = getType(record.Fields[f])
					}
					fmt.Fprint(writer, "\"" + f + "[" + col + "]\"")
					j++;			
				}
				fmt.Fprint(writer, "\r\n");
			}
			j := 0
			for _, f := range fields {
				field := record.Fields[f]
				if j > 0 {
					fmt.Fprint(writer, ",");
				}
				if f == "$id" {
					fmt.Fprintf(writer, "\"%d\"",  record.Id())
				} else if f == "$revision" {
					fmt.Fprintf(writer, "\"%d\"",  record.Revision())
				} else {
					fmt.Fprint(writer, "\"" + escapeCol(toString(field, "\n")) + "\"")
				}
				j++;			
			}
			fmt.Fprint(writer, "\r\n");
			i++;
		}
		if len(records) < ROW_LIMIT {
			break
		}
	}

	return nil
}

func escapeCol(s string) string {
	return strings.Replace(s, "\"", "\"\"", -1)
}

func getType(f interface{}) string {
	switch f.(type) {
	case kintone.SingleLineTextField:
		return kintone.FT_SINGLE_LINE_TEXT
	case kintone.MultiLineTextField:
		return kintone.FT_MULTI_LINE_TEXT
	case kintone.RichTextField:
		return kintone.FT_RICH_TEXT
	case kintone.DecimalField:
		return kintone.FT_DECIMAL
	case kintone.CalcField:
		return kintone.FT_CALC
	case kintone.CheckBoxField:
		return kintone.FT_CHECK_BOX
	case kintone.RadioButtonField:
		return kintone.FT_RADIO
	case kintone.SingleSelectField:
		return kintone.FT_SINGLE_SELECT
	case kintone.MultiSelectField:
		return kintone.FT_MULTI_SELECT
	case kintone.FileField:
		return kintone.FT_FILE
	case kintone.LinkField:
		return kintone.FT_LINK
	case kintone.DateField:
		return kintone.FT_DATE
	case kintone.TimeField:
		return kintone.FT_TIME
	case kintone.DateTimeField:
		return kintone.FT_DATETIME
	case kintone.UserField:
		return kintone.FT_USER
	case kintone.CategoryField:
		return kintone.FT_CATEGORY
	case kintone.StatusField:
		return kintone.FT_STATUS
	case kintone.RecordNumberField:
		return kintone.FT_RECNUM
	case kintone.AssigneeField:
		return kintone.FT_ASSIGNEE
	case kintone.CreatorField:
		return kintone.FT_CREATOR
	case kintone.ModifierField:
		return kintone.FT_MODIFIER
	case kintone.CreationTimeField:
		return kintone.FT_CTIME
	case kintone.ModificationTimeField:
		return kintone.FT_MTIME
	case kintone.SubTableField:
		return kintone.FT_SUBTABLE
	}
	return ""
}

func toString(f interface{}, delimiter string) string {

	if delimiter == "" {
		delimiter = ","
	}
	switch f.(type) {
	case kintone.SingleLineTextField:
		singleLineTextField := f.(kintone.SingleLineTextField)
		return string(singleLineTextField)
	case kintone.MultiLineTextField:
		multiLineTextField := f.(kintone.MultiLineTextField)
		return string(multiLineTextField)
	case kintone.RichTextField:
		richTextField := f.(kintone.RichTextField)
		return string(richTextField)
	case kintone.DecimalField:
		decimalField := f.(kintone.DecimalField)
		return string(decimalField)
	case kintone.CalcField:
		calcField := f.(kintone.CalcField)
		return string(calcField)
	case kintone.RadioButtonField:
		radioButtonField := f.(kintone.RadioButtonField)
		return string(radioButtonField)
	case kintone.LinkField:
		linkField := f.(kintone.LinkField)
		return string(linkField)
	case kintone.StatusField:
		statusField := f.(kintone.StatusField)
		return string(statusField)
	case kintone.RecordNumberField:
		recordNumberField := f.(kintone.RecordNumberField)
		return string(recordNumberField)
	case kintone.CheckBoxField:
		checkBoxField := f.(kintone.CheckBoxField)
		return strings.Join(checkBoxField, delimiter)
	case kintone.MultiSelectField:
		multiSelectField := f.(kintone.MultiSelectField)
		return strings.Join(multiSelectField, delimiter)
	case kintone.CategoryField:
		categoryField := f.(kintone.CategoryField)
		return strings.Join(categoryField, delimiter)
	case kintone.SingleSelectField:
		singleSelect := f.(kintone.SingleSelectField)
		return singleSelect.String
	case kintone.FileField:
		fileField := f.(kintone.FileField)
		files := make([]string, 0, len(fileField))
		for _, file := range fileField {
			files = append(files, file.Name)
		}
		return strings.Join(files, delimiter)
	case kintone.DateField:
		dateField := f.(kintone.DateField)
		if dateField.Valid {
			return dateField.Date.Format("2006-01-02")
		} else {
			return ""
		}
	case kintone.TimeField:
		timeField := f.(kintone.TimeField)
		if timeField.Valid {
			return timeField.Time.Format("15:04:05")
		} else {
			return ""
		}
	case kintone.DateTimeField:
		dateTimeField := f.(kintone.DateTimeField)
		if dateTimeField.Valid {
			return dateTimeField.Time.Format(time.RFC3339)
		} else {
			return ""
		}
	case kintone.UserField:
		userField := f.(kintone.UserField)
		users := make([]string, 0, len(userField))
		for _, user := range userField {
			users = append(users, user.Code)
		}
		return strings.Join(users, delimiter)
	case kintone.AssigneeField:
		assigneeField := f.(kintone.AssigneeField)
		users := make([]string, 0, len(assigneeField))
		for _, user := range assigneeField {
			users = append(users, user.Code)
		}
		return strings.Join(users, delimiter)
	case kintone.CreatorField:
		creatorField := f.(kintone.CreatorField)
		return creatorField.Code
	case kintone.ModifierField:
		modifierField := f.(kintone.ModifierField)
		return modifierField.Code
	case kintone.CreationTimeField:
		creationTimeField := f.(kintone.CreationTimeField)
		return time.Time(creationTimeField).Format(time.RFC3339)
	case kintone.ModificationTimeField:
		modificationTimeField := f.(kintone.ModificationTimeField)
		return time.Time(modificationTimeField).Format(time.RFC3339)
	case kintone.SubTableField:
		return "" // unsupported
	}
	return ""
}
	
