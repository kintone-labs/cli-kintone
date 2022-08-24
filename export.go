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

	"github.com/kintone-labs/go-kintone"
	"golang.org/x/text/transform"
)

const (
	SUBTABLE_ROW_PREFIX = "*"
	RECORD_NOT_FOUND    = "No record found. \nPlease check your query or permission settings."
)

func checkNoRecord(records []*kintone.Record) {
	if len(records) < 1 {
		fmt.Println(RECORD_NOT_FOUND)
		os.Exit(1)
	}
}

func getRecordsForSeekMethod(app *kintone.App, id uint64, fields []string, isRecordFound bool) ([]*kintone.Record, error) {
	query := fmt.Sprintf(" order by $id desc limit %v", EXPORT_ROW_LIMIT)
	if id > 0 {
		query = "$id < " + fmt.Sprintf("%v", id) + query
	}
	records, err := app.GetRecords(fields, query)
	if err != nil {
		return nil, err
	}
	if isRecordFound {
		checkNoRecord(records)
	}
	return records, nil
}

func getRow(app *kintone.App) (Row, error) {
	var row Row
	// retrieve field list
	fields, err := getSupportedFields(app)
	if err != nil {
		return row, err
	}

	if config.Fields == nil {
		row = makeRow(fields)
	} else {
		row = makePartialRow(fields, config.Fields)
	}
	fixOrderCell(row)
	return row, err
}

func fixOrderCell(row Row) {
	for x := range row {
		hasIdOrRevision := x == 0 || x == 1
		if hasIdOrRevision {
			continue
		}
		y := x + 1
		for y = range row {
			if row[x].Index < row[y].Index {
				temp := row[x]
				row[x] = row[y]
				row[y] = temp
			}
		}
	}
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

func escapeCol(s string) string {
	return strings.Replace(s, "\"", "\"\"", -1)
}

func exportRecordsBySeekMethod(app *kintone.App, writer io.Writer, fields []string, isAppendIdCustome bool) error {
	row, err := getRow(app)
	hasTable := hasSubTable(row)
	if err != nil {
		return err
	}

	if config.Format == "json" {
		err := writeRecordsBySeekMethodForJson(app, 0, writer, 0, fields, true, isAppendIdCustome)
		return err
	}
	return writeRecordsBySeekMethodForCsv(app, 0, writer, row, hasTable, 0, fields, true, isAppendIdCustome)
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

func exportRecords(app *kintone.App, fields []string, writer io.Writer) error {
	records, err := app.GetRecords(fields, config.Query)
	if err != nil {
		return err
	}
	checkNoRecord(records)
	if config.Format == "json" {
		fmt.Fprint(writer, "{\"records\": [\n")
		_, err = writeRecordsJSON(app, writer, records, 0, false)
		if err != nil {
			return err
		}
		fmt.Fprint(writer, "\n]}")
	} else {
		row, err := getRow(app)
		hasTable := hasSubTable(row)

		if err != nil {
			return err
		}
		_, err = writeRecordsCsv(app, writer, records, row, hasTable, 0, false)
		if err != nil {
			return err
		}
	}

	if err != nil {
		return err
	}

	return nil
}

func exportRecordsByCursor(app *kintone.App, fields []string, writer io.Writer) error {
	if config.Format == "json" {
		return exportRecordsByCursorForJSON(app, fields, writer)
	}
	return exportRecordsByCursorForCsv(app, fields, writer)
}

func exportRecordsByCursorForJSON(app *kintone.App, fields []string, writer io.Writer) error {
	cursor, err := app.CreateCursor(fields, config.Query, EXPORT_ROW_LIMIT)
	if err != nil {
		return err
	}
	index := uint64(0)
	for {
		recordsCursor, err := getAllRecordsByCursor(app, cursor.Id)
		if err != nil {
			return err
		}
		if index == 0 {
			fmt.Fprint(writer, "{\"records\": [\n")
		}
		index, err = writeRecordsJSON(app, writer, recordsCursor.Records, index, false)
		if err != nil {
			return err
		}

		if !recordsCursor.Next {
			fmt.Fprint(writer, "\n]}")
			break
		}
	}

	return nil
}

func exportRecordsByCursorForCsv(app *kintone.App, fields []string, writer io.Writer) error {
	cursor, err := app.CreateCursor(fields, config.Query, EXPORT_ROW_LIMIT)
	if err != nil {
		return err
	}

	row, err := getRow(app)
	if err != nil {
		return err
	}
	hasTable := hasSubTable(row)
	index := uint64(0)
	for {
		recordsCursor, err := getAllRecordsByCursor(app, cursor.Id)
		if err != nil {
			return err
		}
		index, err = writeRecordsCsv(app, writer, recordsCursor.Records, row, hasTable, index, false)
		if err != nil {
			return err
		}

		if !recordsCursor.Next {
			break
		}
	}
	return nil
}

func getAllRecordsByCursor(app *kintone.App, id string) (*kintone.GetRecordsCursorResponse, error) {
	recordsCursor, err := app.GetRecordsByCursor(id)
	if err != nil {
		return nil, err
	}
	checkNoRecord(recordsCursor.Records)
	return recordsCursor, nil
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

func getSubTableRowCount(record *kintone.Record, row []*Cell) int {
	var ret = 1
	for _, cell := range row {
		if cell.IsSubField {
			subTable := record.Fields[cell.Table].(kintone.SubTableField)

			count := len(subTable)
			if count > ret {
				ret = count
			}
		}
	}

	return ret
}

func getWriter(writer io.Writer) io.Writer {
	encoding := getEncoding()
	if encoding == nil {
		return writer
	}
	return transform.NewWriter(writer, encoding.NewEncoder())
}

func hasSubTable(row []*Cell) bool {
	for _, cell := range row {
		if cell.IsSubField {
			return true
		}
	}
	return false
}

func isExistFile(fileFullPath string) bool {
	_, fileNotExist := os.Stat(fileFullPath)
	return !os.IsNotExist(fileNotExist)
}

func makeRow(fields map[string]*kintone.FieldInfo) Row {
	row := make([]*Cell, 0)

	var cell *Cell

	cell = &Cell{Code: "$id", Type: kintone.FT_ID}
	row = append(row, cell)
	cell = &Cell{Code: "$revision", Type: kintone.FT_REVISION}
	row = append(row, cell)

	for _, val := range fields {
		if val.Code == "" {
			continue
		}
		if val.Type == kintone.FT_SUBTABLE {
			// record id for subtable
			cell := &Cell{Code: val.Code, Type: val.Type, Index: val.Index}
			row = append(row, cell)

			for _, subField := range val.Fields {
				cell := &Cell{Code: subField.Code, Type: subField.Type, IsSubField: true, Table: val.Code, Index: subField.Index}
				row = append(row, cell)
			}
		} else {
			cell := &Cell{Code: val.Code, Type: val.Type, Index: val.Index}
			row = append(row, cell)
		}
	}

	return row
}

func makePartialRow(fields map[string]*kintone.FieldInfo, partialFields []string) Row {
	row := make([]*Cell, 0)

	maxFieldIdx := 0
	for index, val := range partialFields {
		cell := getCell(val, fields)
		if cell.Type == "UNKNOWN" || cell.IsSubField {
			continue
		}
		currentFieldIdx := index + maxFieldIdx
		if cell.Type == kintone.FT_SUBTABLE {
			// record id for subtable
			cell := &Cell{Code: cell.Code, Type: cell.Type, Index: currentFieldIdx}
			row = append(row, cell)

			// append all sub fields
			field := fields[val]
			maxSubFieldIdx := 0
			for _, subField := range field.Fields {
				currentSubFieldIdx := subField.Index + maxFieldIdx
				cell := &Cell{Code: subField.Code, Type: subField.Type, IsSubField: true, Table: val, Index: currentSubFieldIdx}
				row = append(row, cell)
				if currentSubFieldIdx > maxSubFieldIdx {
					maxSubFieldIdx = currentSubFieldIdx
				}
			}
			if maxSubFieldIdx > maxFieldIdx {
				maxFieldIdx = maxSubFieldIdx
			}
		} else {
			cell := &Cell{Code: cell.Code, Type: cell.Type, Index: currentFieldIdx}
			maxFieldIdx = currentFieldIdx
			row = append(row, cell)
		}
	}
	return row
}

func writeHeaderCsv(writer io.Writer, hasTable bool, row Row) {
	i := 0
	if hasTable {
		fmt.Fprint(writer, SUBTABLE_ROW_PREFIX)
		i++
	}
	for _, cell := range row {
		if i > 0 {
			fmt.Fprint(writer, ",")
		}
		fmt.Fprint(writer, "\""+cell.Code+"\"")
		i++
	}
	fmt.Fprint(writer, "\r\n")
}

func writeRecordsJSON(app *kintone.App, writer io.Writer, records []*kintone.Record, i uint64, isAppendIdCustome bool) (uint64, error) {
	for _, record := range records {
		if i > 0 {
			fmt.Fprint(writer, ",\n")
		}
		rowID := record.Id()
		if rowID == 0 || isAppendIdCustome {
			rowID = i
		}
		// Download file to local folder that is the value of param -b
		for fieldCode, fieldInfo := range record.Fields {
			fieldType := reflect.TypeOf(fieldInfo).String()
			if fieldType == "kintone.FileField" {
				dir := fmt.Sprintf("%s-%d", fieldCode, rowID)
				err := downloadFile(app, fieldInfo, dir)
				if err != nil {
					return 0, err

				}
			} else if fieldType == "kintone.SubTableField" {
				subTable := fieldInfo.(kintone.SubTableField)
				for subTableIndex, subTableValue := range subTable {
					for fieldCodeInSubTable, fieldValueInSubTable := range subTableValue.Fields {
						if reflect.TypeOf(fieldValueInSubTable).String() == "kintone.FileField" {
							dir := fmt.Sprintf("%s-%d-%d", fieldCodeInSubTable, rowID, subTableIndex)
							err := downloadFile(app, fieldValueInSubTable, dir)
							if err != nil {
								return 0, err

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
			return 0, err
		}
		i++
	}
	return i, nil
}

func writeRecordsCsv(app *kintone.App, writer io.Writer, records []*kintone.Record, row Row, hasTable bool, i uint64, isAppendIdCustome bool) (uint64, error) {
	if i == 0 {
		writeHeaderCsv(writer, hasTable, row)
	}
	for _, record := range records {
		rowID := record.Id()
		if rowID == 0 || isAppendIdCustome {
			rowID = i
		}

		// determine subtable's row count
		rowNum := getSubTableRowCount(record, row)
		for j := 0; j < rowNum; j++ {
			k := 0
			if hasTable {
				if j == 0 {
					fmt.Fprint(writer, "*")
				}
				k++
			}

			for _, f := range row {
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
								return 0, err
							}
						}
						fmt.Fprint(writer, "\"")
						_, err := fmt.Fprint(writer, escapeCol(toString(subField, "\n")))
						if err != nil {
							return 0, err
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
								return 0, err
							}
						}
						fmt.Fprint(writer, "\"")
						_, err := fmt.Fprint(writer, escapeCol(toString(field, "\n")))
						if err != nil {
							return 0, err
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

	return i, nil
}

func writeRecordsBySeekMethodForCsv(app *kintone.App, id uint64, writer io.Writer, row Row, hasTable bool, index uint64, fields []string, isRecordFound bool, isAppendIdCustome bool) error {
	records, err := getRecordsForSeekMethod(app, id, fields, isRecordFound)
	if err != nil {
		return err
	}
	index, err = writeRecordsCsv(app, writer, records, row, hasTable, index, isAppendIdCustome)
	if err != nil {
		return err
	}
	if len(records) == EXPORT_ROW_LIMIT {
		isRecordFound = false
		return writeRecordsBySeekMethodForCsv(app, records[len(records)-1].Id(), writer, row, hasTable, index, fields, isRecordFound, isAppendIdCustome)
	}
	return nil
}

func writeRecordsBySeekMethodForJson(app *kintone.App, id uint64, writer io.Writer, index uint64, fields []string, isRecordsNotFound bool, isAppendIdCustome bool) error {
	records, err := getRecordsForSeekMethod(app, id, fields, isRecordsNotFound)
	if err != nil {
		return err
	}
	if index == 0 {
		fmt.Fprint(writer, "{\"records\": [\n")
	}
	index, err = writeRecordsJSON(app, writer, records, index, isAppendIdCustome)
	if err != nil {
		return err
	}
	if len(records) == EXPORT_ROW_LIMIT {
		isRecordsNotFound = false
		return writeRecordsBySeekMethodForJson(app, records[len(records)-1].Id(), writer, index, fields, isRecordsNotFound, isAppendIdCustome)
	}
	fmt.Fprint(writer, "\n]}")
	return nil
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
