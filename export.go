package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/kintone/go-kintone"
	"golang.org/x/text/transform"
)

const (
	SUBTABLE_ROW_PREFIX = "*"
	RECORD_NOT_FOUND    = "No record found."
)

func getAllRecordsByCursor(app *kintone.App, id string) (*kintone.GetRecordsCursorResponse, error) {
	recordsCursor, err := app.GetRecordsByCursor(id)
	checkNoRecord(recordsCursor.Records)
	if err != nil {
		return nil, err
	}
	return recordsCursor, nil
}

func writeRecordsBySeekMethodForCsv(app *kintone.App, id uint64, writer io.Writer, columns Columns, hasTable bool) error {
	query := fmt.Sprintf(" order by $id desc limit %v", EXPORT_ROW_LIMIT)
	if id > 0 {
		query = "$id < " + fmt.Sprintf("%v", id) + query
	}

	records, err := app.GetRecords(nil, query)
	checkNoRecord(records)
	if err != nil {
		return err
	}

	err = writeCsv(app, writer, records, columns, hasTable)

	if len(records) == EXPORT_ROW_LIMIT {
		return writeRecordsBySeekMethodForCsv(app, records[len(records)-1].Id(), writer, columns, hasTable)
	}
	return nil
}

func writeRecordsBySeekMethodForJson(app *kintone.App, id uint64, writer io.Writer, columns Columns, hasTable bool) error {
	defaultQuery := fmt.Sprintf(" order by $id asc limit %v", EXPORT_ROW_LIMIT)
	query := "$id > " + fmt.Sprintf("%v", id) + defaultQuery

	records, err := app.GetRecords(nil, query)
	checkNoRecord(records)
	if err != nil {
		return err
	}
	err = writeJSON(app, writer, records)

	if len(records) == EXPORT_ROW_LIMIT {
		return writeRecordsBySeekMethodForJson(app, records[len(records)-1].Id(), writer, columns, hasTable)
	}
	return nil
}

func writeRecordsBySeekMethod(app *kintone.App, id uint64, writer io.Writer, columns Columns, hasTable bool) error {
	if config.Format == "json" {
		fmt.Fprint(writer, "{\"records\": [\n")
		err := writeRecordsBySeekMethodForJson(app, id, writer, columns, hasTable)
		fmt.Fprint(writer, "\n]}")
		return err
	}
	writeHeaderCsv(writer, hasTable, columns)
	return writeRecordsBySeekMethodForCsv(app, id, writer, columns, hasTable)
}

func exportRecordsBySeekMethod(app *kintone.App, writer io.Writer) error {
	columns, hasTable, err := createRow(app)
	if err != nil {
		return err
	}

	return writeRecordsBySeekMethod(app, 0, writer, columns, hasTable)
}

func exportRecords(app *kintone.App, fields []string, writer io.Writer) error {
	records, err := app.GetRecords(fields, config.Query)
	checkNoRecord(records)
	if err != nil {
		return err
	}

	if config.Format == "json" {
		fmt.Fprint(writer, "{\"records\": [\n")
		err = writeJSON(app, writer, records)
		fmt.Fprint(writer, "\n]}")
	} else {
		columns, hasTable, err := createRow(app)
		if err != nil {
			return err
		}

		writeHeaderCsv(writer, hasTable, columns)
		err = writeCsv(app, writer, records, columns, hasTable)
	}

	if err != nil {
		return err
	}

	return nil
}

func exportRecordsByCursorForJSON(app *kintone.App, fields []string, writer io.Writer) error {
	cursor, err := app.CreateCursor(fields, config.Query, EXPORT_ROW_LIMIT)
	if err != nil {
		return err
	}
	fmt.Fprint(writer, "{\"records\": [\n")
	for {
		recordsCursor, err := getAllRecordsByCursor(app, cursor.Id)
		if err != nil {
			return err
		}

		err = writeJSON(app, writer, recordsCursor.Records)
		if err != nil {
			return err
		}

		if !recordsCursor.Next {
			break
		}
	}
	fmt.Fprint(writer, "\n]}")
	return nil
}

func exportRecordsByCursor(app *kintone.App, fields []string, writer io.Writer) error {
	if config.Format == "json" {
		return exportRecordsByCursorForJSON(app, fields, writer)
	}
	return exportRecordsByCursorForCsv(app, fields, writer)
}

func exportRecordsByCursorForCsv(app *kintone.App, fields []string, writer io.Writer) error {
	cursor, err := app.CreateCursor(fields, config.Query, EXPORT_ROW_LIMIT)
	if err != nil {
		return err
	}

	columns, hasTable, err := createRow(app)
	if err != nil {
		return err
	}

	writeHeaderCsv(writer, hasTable, columns)

	for {
		recordsCursor, err := getAllRecordsByCursor(app, cursor.Id)
		checkNoRecord(recordsCursor.Records)
		if err != nil {
			return err
		}

		err = writeCsv(app, writer, recordsCursor.Records, columns, hasTable)
		if err != nil {
			return err
		}

		if !recordsCursor.Next {
			break
		}
	}
	return nil
}

func checkNoRecord(records []*kintone.Record) {
	if len(records) < 1 {
		fmt.Println(RECORD_NOT_FOUND)
		fmt.Println("Please check your query or permission settings.")
		os.Exit(1)
	}
}

func exportRecordsWithQuery(app *kintone.App, fields []string, writer io.Writer) error {
	containLimit := regexp.MustCompile(`limit\s+\d+`)
	containOffset := regexp.MustCompile(`offset\s+\d+`)

	hasLimit := containLimit.MatchString(config.Query)
	hasOffset := containOffset.MatchString(config.Query)

	if hasLimit || hasOffset {
		return exportRecords(app, fields, writer)
	}
	return exportRecordsByCursor(app, fields, writer)
}

func getWriter(writer io.Writer) io.Writer {
	encoding := getEncoding()
	if encoding == nil {
		return writer
	}
	return transform.NewWriter(writer, encoding.NewEncoder())
}

func writeJSON(app *kintone.App, writer io.Writer, records []*kintone.Record) error {
	i := 0
	for _, record := range records {
		if i > 0 {
			fmt.Fprint(writer, ",\n")
		}
		// Download file to local folder that is the value of param -b
		for fieldCode, fieldInfo := range record.Fields {
			fieldType := reflect.TypeOf(fieldInfo).String()
			if fieldType == "kintone.FileField" {
				dir := fmt.Sprintf("%s-%d", fieldCode, record.Id())
				err := downloadFile(app, fieldInfo, dir)
				if err != nil {
					return err

				}
			} else if fieldType == "kintone.SubTableField" {
				subTable := fieldInfo.(kintone.SubTableField)
				for subTableIndex, subTableValue := range subTable {
					for fieldCodeInSubTable, fieldValueInSubTable := range subTableValue.Fields {
						if reflect.TypeOf(fieldValueInSubTable).String() == "kintone.FileField" {
							dir := fmt.Sprintf("%s-%d-%d", fieldCodeInSubTable, record.Id(), subTableIndex)
							err := downloadFile(app, fieldValueInSubTable, dir)
							if err != nil {
								return err

							}
						}
					}
				}
			}
		}
		jsonArray, _ := record.MarshalJSON()
		json := string(jsonArray)
		_, err := fmt.Fprint(writer, json)
		if err != nil {
			return err
		}
		i++
	}
	return nil
}

func makeColumns(fields map[string]*kintone.FieldInfo) Columns {
	columns := make([]*Column, 0)

	var column *Column

	column = &Column{Code: "$id", Type: kintone.FT_ID}
	columns = append(columns, column)
	column = &Column{Code: "$revision", Type: kintone.FT_REVISION}
	columns = append(columns, column)

	for _, val := range fields {
		if val.Code == "" {
			continue
		}
		if val.Type == kintone.FT_SUBTABLE {
			// record id for subtable
			column := &Column{Code: val.Code, Type: val.Type}
			columns = append(columns, column)

			for _, subField := range val.Fields {
				column := &Column{Code: subField.Code, Type: subField.Type, IsSubField: true, Table: val.Code}
				columns = append(columns, column)
			}
		} else {
			column := &Column{Code: val.Code, Type: val.Type}
			columns = append(columns, column)
		}
	}

	return columns
}

func makePartialColumns(fields map[string]*kintone.FieldInfo, partialFields []string) Columns {
	columns := make([]*Column, 0)

	for _, val := range partialFields {
		column := getColumn(val, fields)

		if column.Type == "UNKNOWN" || column.IsSubField {
			continue
		}
		if column.Type == kintone.FT_SUBTABLE {
			// record id for subtable
			column := &Column{Code: column.Code, Type: column.Type}
			columns = append(columns, column)

			// append all sub fields
			field := fields[val]

			for _, subField := range field.Fields {
				column := &Column{Code: subField.Code, Type: subField.Type, IsSubField: true, Table: val}
				columns = append(columns, column)
			}
		} else {
			columns = append(columns, column)
		}
	}
	return columns
}

func getSubTableRowCount(record *kintone.Record, columns []*Column) int {
	var ret = 1
	for _, c := range columns {
		if c.IsSubField {
			subTable := record.Fields[c.Table].(kintone.SubTableField)

			count := len(subTable)
			if count > ret {
				ret = count
			}
		}
	}

	return ret
}

func hasSubTable(columns []*Column) bool {
	for _, c := range columns {
		if c.IsSubField {
			return true
		}
	}
	return false
}

func writeHeaderCsv(writer io.Writer, hasTable bool, columns Columns) {
	i := 0
	if hasTable {
		fmt.Fprint(writer, SUBTABLE_ROW_PREFIX)
		i++
	}
	for _, f := range columns {
		if i > 0 {
			fmt.Fprint(writer, ",")
		}
		fmt.Fprint(writer, "\""+f.Code+"\"")
		i++
	}
	fmt.Fprint(writer, "\r\n")
}

func createRow(app *kintone.App) (Columns, bool, error) {
	var columns Columns
	hasTable := false

	// retrieve field list
	fields, err := getFields(app)
	if err != nil {
		return columns, hasTable, err
	}

	if config.Fields == nil {
		columns = makeColumns(fields)
	} else {
		columns = makePartialColumns(fields, config.Fields)
	}

	hasTable = hasSubTable(columns)
	return columns, hasTable, err

}

func writeCsv(app *kintone.App, writer io.Writer, records []*kintone.Record, columns Columns, hasTable bool) error {
	i := uint64(0)
	for _, record := range records {
		rowID := record.Id()
		if rowID == 0 {
			rowID = i
		}

		// determine subtable's row count
		rowNum := getSubTableRowCount(record, columns)
		for j := 0; j < rowNum; j++ {
			k := 0
			if hasTable {
				if j == 0 {
					fmt.Fprint(writer, "*")
				}
				k++
			}

			for _, f := range columns {
				if k > 0 {
					fmt.Fprint(writer, ",")
				}

				if f.Code == "$id" {
					fmt.Fprintf(writer, "\"%d\"", record.Id())
				} else if f.Code == "$revision" {
					fmt.Fprintf(writer, "\"%d\"", record.Revision())
				} else if f.Type == kintone.FT_SUBTABLE {
					table := record.Fields[f.Code].(kintone.SubTableField)
					if j < len(table) {
						fmt.Fprintf(writer, "\"%d\"", table[j].Id())
					}
				} else if f.IsSubField {
					table := record.Fields[f.Table].(kintone.SubTableField)
					if j < len(table) {
						subField := table[j].Fields[f.Code]
						if f.Type == kintone.FT_FILE {
							dir := fmt.Sprintf("%s-%d-%d", f.Code, rowID, j)
							err := downloadFile(app, subField, dir)
							if err != nil {
								return err
							}
						}
						fmt.Fprint(writer, "\"")
						_, err := fmt.Fprint(writer, escapeCol(toString(subField, "\n")))
						if err != nil {
							return err
						}
						fmt.Fprint(writer, "\"")
					}
				} else {
					field := record.Fields[f.Code]
					if field != nil {
						if j == 0 && f.Type == kintone.FT_FILE {
							dir := fmt.Sprintf("%s-%d", f.Code, rowID)
							err := downloadFile(app, field, dir)
							if err != nil {
								return err
							}
						}
						fmt.Fprint(writer, "\"")
						_, err := fmt.Fprint(writer, escapeCol(toString(field, "\n")))
						if err != nil {
							return err
						}
						fmt.Fprint(writer, "\"")
					}
				}
				k++
			}
			fmt.Fprint(writer, "\r\n")
		}
		i++
	}

	return nil
}

func downloadFile(app *kintone.App, field interface{}, dir string) error {
	if config.FileDir == "" {
		return nil
	}

	v, ok := field.(kintone.FileField)
	if !ok {
		return nil
	}

	if len(v) == 0 {
		return nil
	}

	fileDir := fmt.Sprintf("%s%c%s", config.FileDir, os.PathSeparator, dir)
	if err := os.MkdirAll(fileDir, 0777); err != nil {
		return err
	}
	for idx, file := range v {
		fileName := getUniqueFileName(file.Name, fileDir)
		path := fmt.Sprintf("%s%c%s", fileDir, os.PathSeparator, fileName)
		data, err := app.Download(file.FileKey)
		if err != nil {
			return err
		}

		fo, err := os.Create(path)
		if err != nil {
			return err
		}
		defer fo.Close()

		// make a buffer to keep chunks that are read
		buf := make([]byte, 256*1024)
		for {
			// read a chunk
			n, err := data.Reader.Read(buf)
			if err != nil && err != io.EOF {
				return err
			}
			if n == 0 {
				break
			}

			// write a chunk
			if _, err := fo.Write(buf[:n]); err != nil {
				return err
			}
		}

		v[idx].Name = fmt.Sprintf("%s%c%s", dir, os.PathSeparator, fileName)
	}

	return nil
}

func getUniqueFileName(filename, dir string) string {
	filenameOuput := filename
	fileExt := filepath.Ext(filename)
	fileBaseName := filename[0 : len(filename)-len(fileExt)]
	index := 0
	parentDir := fmt.Sprintf("%s%c", dir, os.PathSeparator)
	if dir == "" {
		parentDir = ""
	}
	for {
		fileFullPath := fmt.Sprintf("%s%s", parentDir, filenameOuput)
		if !isExistFile(fileFullPath) {
			break
		}
		index++
		filenameOuput = fmt.Sprintf("%s (%d)%s", fileBaseName, index, fileExt)
	}
	return filenameOuput
}
func isExistFile(fileFullPath string) bool {
	_, fileNotExist := os.Stat(fileFullPath)
	return !os.IsNotExist(fileNotExist)
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
	case kintone.OrganizationField:
		return kintone.FT_ORGANIZATION
	case kintone.GroupField:
		return kintone.FT_GROUP
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
		}
		return ""
	case kintone.TimeField:
		timeField := f.(kintone.TimeField)
		if timeField.Valid {
			return timeField.Time.Format("15:04:05")
		}
		return ""
	case kintone.DateTimeField:
		dateTimeField := f.(kintone.DateTimeField)
		if dateTimeField.Valid {
			return dateTimeField.Time.Format(time.RFC3339)
		}
		return ""
	case kintone.UserField:
		userField := f.(kintone.UserField)
		users := make([]string, 0, len(userField))
		for _, user := range userField {
			users = append(users, user.Code)
		}
		return strings.Join(users, delimiter)
	case kintone.OrganizationField:
		organizationField := f.(kintone.OrganizationField)
		organizations := make([]string, 0, len(organizationField))
		for _, organization := range organizationField {
			organizations = append(organizations, organization.Code)
		}
		return strings.Join(organizations, delimiter)
	case kintone.GroupField:
		groupField := f.(kintone.GroupField)
		groups := make([]string, 0, len(groupField))
		for _, group := range groupField {
			groups = append(groups, group.Code)
		}
		return strings.Join(groups, delimiter)
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
