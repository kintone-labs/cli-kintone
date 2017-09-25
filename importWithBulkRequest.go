package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"time"

	"github.com/kintone/go-kintone"
)

const (
	// ConstBulkRequestLimitRecordOption set option: record per bulkRequest
	ConstBulkRequestLimitRecordOption = 100

	// ConstBulkRequestLimitRequest kintone limited: The request count per bulkRequest
	ConstBulkRequestLimitRequest = 20

	// ConstRecordsLimitPerRequest kintone limited: The records count per request
	ConstRecordsLimitPerRequest = 100
)

func importFromCSV(app *kintone.App, _reader io.Reader) error {

	reader := csv.NewReader(getReader(_reader))

	head := true
	// recordsInsert := make([]*kintone.Record, 0, IMPORT_ROW_LIMIT)
	// recordsUpdate := make([]*kintone.Record, 0, IMPORT_ROW_LIMIT)

	var columns Columns
	var lastRowImport uint64
	lastRowImport = config.line
	bulkRequests := &BulkRequests{}
	// retrieve field list
	fields, err := getFields(app)
	if err != nil {
		return err
	}

	if config.deleteAll {
		err = deleteRecords(app, config.query)
		if err != nil {
			return err
		}
		config.deleteAll = false
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
			if rowNumber < config.line {
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
							subId, _ := strconv.ParseUint(col, 10, 64)
							table := getSubRecord(column.Code, tables)
							table.Id = subId
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
				err = bulkRequests.ImportDataUpdate(app, kintone.NewRecordWithId(id, record))
				if err != nil {
					log.Fatalln(err)
				}
			} else {
				err = bulkRequests.ImportDataInsert(app, kintone.NewRecord(record))
				if err != nil {
					log.Fatalln(err)
				}
			}
			if rowNumber%(ConstBulkRequestLimitRecordOption) == 0 {
				fmt.Printf("%v: ", time.Now().Format("[2006-01-02 15:04:05]"))
				fmt.Printf("Start from: %d - %d", lastRowImport, rowNumber)

				resp, err := bulkRequests.Request(app)
				bulkRequests.HandelResponse(resp, err, lastRowImport, rowNumber)

				bulkRequests.Requests = bulkRequests.Requests[:0]
				lastRowImport = rowNumber + 1

			}
		}
	}
	if len(bulkRequests.Requests) > 0 {
		fmt.Printf("%v: ", time.Now().Format("[2006-01-02 15:04:05]"))
		fmt.Printf("Start from: %d - %d", lastRowImport, rowNumber)
		resp, err := bulkRequests.Request(app)
		bulkRequests.HandelResponse(resp, err, lastRowImport, rowNumber)
	}
	fmt.Printf("%v: ", time.Now().Format("[2006-01-02 15:04:05]"))
	fmt.Printf("DONE\n")

	return nil
}
