package main

import (
	"github.com/cybozu/go-kintone"
	"github.com/howeyc/gopass"
	"sort"
	"flag"
	"log"
	"fmt"
	"strings"
	"time"
)

var login string
var password string
var domain string
var basic string
var format string
var query string
var appId uint64
var encoding string
var fields []string

const ROW_LIMIT = 100

func main() {
	var colNames string
	
	flag.StringVar(&login, "u", "", "Login name")
	flag.StringVar(&password, "p", "", "Password")
	flag.StringVar(&domain, "d", "", "Domain name")
	flag.Uint64Var(&appId, "a", 0, "App ID")
	flag.StringVar(&format, "f", "csv", "Output format: 'json' or 'csv'(default)")
	flag.StringVar(&query, "q", "", "Query string")
	flag.StringVar(&colNames, "c", "", "Field names (comma separated)")
    flag.Parse()

	if (domain == "" || login == "" || appId == 0) {
		flag.PrintDefaults()
		return
	}

	if password == "" {
		fmt.Printf("Password: ")
		password = string(gopass.GetPasswd())
	}

	if !strings.Contains(domain, ".") {
		domain = domain + ".cybozu.com"
	}

	if colNames != "" {
		fields = strings.Split(colNames, ",")
	}

	app := &kintone.App{
		Domain:   domain,
		User:     login,
		Password: password,
		AppId:    appId,
	}

	if format == "json" {
		writeJson(app)
	} else {
		writeCsv(app)
	}
}

func getRecords(app *kintone.App, offset int64) []*kintone.Record{

	newQuery := query + fmt.Sprintf(" limit %v offset %v", ROW_LIMIT, offset)
    records, err := app.GetRecords(fields, newQuery)
    if err != nil {
        log.Fatal(err)
    }
	return records
}

func writeJson(app *kintone.App) {
	i := 0
	offset := int64(0)
	fmt.Print("{\"records\": [\n")
	for ;;offset += ROW_LIMIT {
		records := getRecords(app, offset)
		for _, record := range records {
			if i > 0 {
				fmt.Print(",\n")
			}			
			jsonArray, _ := record.MarshalJSON()
			json := string(jsonArray)
			fmt.Print(json)
			i += 1
		}
		if len(records) < ROW_LIMIT {
			break
		}
	}
	fmt.Print("\n]}")
}

func writeCsv(app *kintone.App) {
	i := 0
	offset := int64(0)

	for ;;offset += ROW_LIMIT {
		records := getRecords(app, offset)
		
		for _, record := range records {
			if i == 0 {
				if fields == nil {
					fields = make([]string, 0, len(record.Fields))
					for key, _ := range record.Fields {
						fields = append(fields, key)
					}
					sort.Strings(fields)
				}
				j := 0
				for _, f := range fields {
					if j > 0 {
						fmt.Print(",");
					}
					fmt.Print("\"" + f + "\"")
					j++;            
				}
				fmt.Print("\n");
			}
			j := 0
			for _, f := range fields {
				field := record.Fields[f]
				if j > 0 {
					fmt.Print(",");
				}
				fmt.Print("\"" + escapeCol(toString(field, "\n")) + "\"")
				j++;            
			}
			fmt.Print("\n");
			i++;
		}
		if len(records) < ROW_LIMIT {
			break
		}
	}
}

func escapeCol(s string) string {
	return strings.Replace(s, "\"", "\"\"", -1)
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
		return time.Time(dateTimeField).Format("RFC3339")
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
		return time.Time(creationTimeField).Format("RFC3339")
	case kintone.ModificationTimeField:
		modificationTimeField := f.(kintone.ModificationTimeField)
		return time.Time(modificationTimeField).Format("RFC3339")
	case kintone.SubTableField:
		return "" // unsupported
	}
	return ""
}
	
