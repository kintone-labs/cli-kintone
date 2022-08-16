package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/kintone-labs/go-kintone"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"

	flags "github.com/jessevdk/go-flags"
)

// NAME of this package
const NAME = "cli-kintone"

// VERSION of this package
const VERSION = "0.14.0"

// IMPORT_ROW_LIMIT The maximum row will be import
const IMPORT_ROW_LIMIT = 100

// EXPORT_ROW_LIMIT The maximum row will be export
const EXPORT_ROW_LIMIT = 500

// Configure of this package
type Configure struct {
	IsImport          bool     `long:"import" description:"Import data from stdin. If \"-f\" is also specified, data is imported from the file instead"`
	IsExport          bool     `long:"export" description:"Export kintone data to stdout"`
	Domain            string   `short:"d" default:"" description:"Domain name (specify the FQDN)"`
	AppID             uint64   `short:"a" default:"0" description:"App ID"`
	Login             string   `short:"u" default:"" description:"User's log in name"`
	Password          string   `short:"p" default:"" description:"User's password"`
	APIToken          string   `short:"t" default:"" description:"API token"`
	GuestSpaceID      uint64   `short:"g" default:"0" description:"Guest Space ID"`
	Format            string   `short:"o" default:"csv" description:"Output format. Specify either 'json' or 'csv'"`
	Encoding          string   `short:"e" default:"utf-8" description:"Character encoding (default: utf-8).\n Only support the encoding below both field code and data itself: \n 'utf-8', 'utf-16', 'utf-16be-with-signature', 'utf-16le-with-signature', 'sjis' or 'euc-jp', 'gbk' or 'big5'"`
	BasicAuthUser     string   `short:"U" default:"" description:"Basic authentication user name"`
	BasicAuthPassword string   `short:"P" default:"" description:"Basic authentication password"`
	Query             string   `short:"q" default:"" description:"Query string"`
	Fields            []string `short:"c" description:"Fields to export (comma separated). Specify the field code name"`
	FilePath          string   `short:"f" default:"" description:"Input file path"`
	FileDir           string   `short:"b" default:"" description:"Attachment file directory"`
	DeleteAll         bool     `short:"D" description:"Delete records before insert. You can specify the deleting record condition by option \"-q\""`
	Line              uint64   `short:"l" default:"1" description:"Position index of data in the input file"`
	Version           bool     `short:"v" long:"version" description:"Version of cli-kintone"`
}

var config Configure

// Column config
// Column config is deprecated, replace using Cell config
type Column struct {
	Code       string
	Type       string
	IsSubField bool
	Table      string
}

// Columns config
// Columns config is deprecated, replace using Row config
type Columns []*Column

// Cell config
type Cell struct {
	Code       string
	Type       string
	IsSubField bool
	Table      string
	Index      int
}

// Row config
type Row []*Cell

func getFields(app *kintone.App) (map[string]*kintone.FieldInfo, error) {
	fields, err := app.Fields()
	ignoreFields := [3]string{"Status", "Assignee", "Categories"}
	for i := 0; i < len(ignoreFields); i++ {
		delete(fields, ignoreFields[i])
	}
	if err != nil {
		return nil, err
	}
	return fields, nil
}

// set column information from fieldinfo
// This function is deprecated, replace using function getCell
func getColumn(code string, fields map[string]*kintone.FieldInfo) *Column {
	// initialize values
	column := Column{Code: code, IsSubField: false, Table: ""}

	if code == "$id" {
		column.Type = kintone.FT_ID
		return &column
	} else if code == "$revision" {
		column.Type = kintone.FT_REVISION
		return &column
	} else {
		// is this code the one of sub field?
		for _, val := range fields {
			if val.Code == code {
				column.Type = val.Type
				return &column
			}
			if val.Type == kintone.FT_SUBTABLE {
				for _, subField := range val.Fields {
					if subField.Code == code {
						column.IsSubField = true
						column.Type = subField.Type
						column.Table = val.Code
						return &column
					}
				}
			}
		}
	}

	// the code is not found
	column.Type = "UNKNOWN"
	return &column
}

func containtString(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

// set Cell information from fieldinfo
// function replace getColumn so getColumn is invalid name
func getCell(code string, fields map[string]*kintone.FieldInfo) *Cell {
	// initialize values
	cell := Cell{Code: code, IsSubField: false, Table: ""}

	if code == "$id" {
		cell.Type = kintone.FT_ID
		return &cell
	} else if code == "$revision" {
		cell.Type = kintone.FT_REVISION
		return &cell
	} else {
		// is this code the one of sub field?
		for _, val := range fields {
			if val.Code == code {
				cell.Type = val.Type
				return &cell
			}
			if val.Type == kintone.FT_SUBTABLE {
				for _, subField := range val.Fields {
					if subField.Code == code {
						cell.IsSubField = true
						cell.Type = subField.Type
						cell.Table = val.Code
						return &cell
					}
				}
			}
		}
	}

	// the code is not found
	cell.Type = "UNKNOWN"
	return &cell
}

func getEncoding() encoding.Encoding {
	switch config.Encoding {
	case "utf-16":
		return unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM)
	case "utf-16be-with-signature":
		return unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM)
	case "utf-16le-with-signature":
		return unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM)
	case "euc-jp":
		return japanese.EUCJP
	case "sjis":
		return japanese.ShiftJIS
	case "gbk":
		return simplifiedchinese.GBK
	case "big5":
		return traditionalchinese.Big5
	default:
		return nil
	}
}

func main() {
	var err error

	_, err = flags.ParseArgs(&config, os.Args[1:])
	if err != nil {
		if os.Args[1] != "-h" && os.Args[1] != "--help" {
			fileExecute := os.Args[0]
			fmt.Printf("\nTry '%s --help' for more information.\n", fileExecute)
		}
		os.Exit(1)
	}

	if len(os.Args) > 0 && config.Version {
		fmt.Println(VERSION)
		os.Exit(0)
	}

	if len(os.Args) == 0 || config.AppID == 0 || (config.APIToken == "" && (config.Domain == "" || config.Login == "")) {
		helpArg := []string{"-h"}
		flags.ParseArgs(&config, helpArg)
		os.Exit(1)
	}

	if !strings.Contains(config.Domain, ".") {
		config.Domain += ".cybozu.com"
	}

	// Support set columm with comma separated (",") in arg
	var cols []string
	if len(config.Fields) > 0 {
		for _, field := range config.Fields {
			curField := strings.Split(field, ",")
			cols = append(cols, curField...)
		}
		config.Fields = nil
		for _, col := range cols {
			curFieldString := strings.TrimSpace(col)
			if curFieldString != "" {
				config.Fields = append(config.Fields, curFieldString)
			}
		}
	}

	var app *kintone.App
	if config.BasicAuthUser != "" && config.BasicAuthPassword == "" {
		fmt.Printf("Basic authentication password: ")
		pass, _ := gopass.GetPasswd()
		config.BasicAuthPassword = string(pass)
	}

	if config.APIToken == "" {
		if config.Password == "" {
			fmt.Printf("Password: ")
			pass, _ := gopass.GetPasswd()
			config.Password = string(pass)
		}

		app = &kintone.App{
			Domain:       config.Domain,
			User:         config.Login,
			Password:     config.Password,
			AppId:        config.AppID,
			GuestSpaceId: config.GuestSpaceID,
		}
	} else {
		app = &kintone.App{
			Domain:       config.Domain,
			ApiToken:     config.APIToken,
			AppId:        config.AppID,
			GuestSpaceId: config.GuestSpaceID,
		}
	}

	if config.BasicAuthUser != "" {
		app.SetBasicAuth(config.BasicAuthUser, config.BasicAuthPassword)
	}

	app.SetUserAgentHeader(NAME + "/" + VERSION + " (" + runtime.GOOS + " " + runtime.GOARCH + ")")

	// Old logic without force import/export
	if config.IsImport == false && config.IsExport == false {
		if config.FilePath == "" {
			writer := getWriter(os.Stdout)
			if config.Query != "" {
				err = exportRecordsWithQuery(app, config.Fields, writer)
			} else {
				fields := config.Fields
				isAppendIdCustome := false
				if len(config.Fields) > 0 && !containtString(config.Fields, "$id") {
					fields = append(fields, "$id")
					isAppendIdCustome = true
				}

				err = exportRecordsBySeekMethod(app, writer, fields, isAppendIdCustome)
			}
		} else {
			err = importDataFromFile(app)
		}
	}
	if config.IsImport && config.IsExport {
		log.Fatal("The options --import and --export cannot be specified together!")
	}

	if config.IsImport {
		if config.FilePath == "" {
			err = importFromCSV(app, os.Stdin)
		} else {

			err = importDataFromFile(app)
		}
	}

	if config.IsExport {
		if config.FilePath != "" {
			log.Fatal("The -f option is not supported with the --export option.")
		}
		writer := getWriter(os.Stdout)
		if config.Query != "" {
			err = exportRecordsWithQuery(app, config.Fields, writer)
		} else {
			fields := config.Fields
			isAppendIdCustome := false
			if len(config.Fields) > 0 && !containtString(config.Fields, "$id") {
				fields = append(fields, "$id")
				isAppendIdCustome = true
			}
			err = exportRecordsBySeekMethod(app, writer, fields, isAppendIdCustome)
		}
	}
	if err != nil {
		log.Fatal(err)
	}
}

func importDataFromFile(app *kintone.App) error {
	var file *os.File
	var err error
	file, err = os.Open(config.FilePath)
	if err == nil {
		defer file.Close()
		err = importFromCSV(app, file)
	}
	return err
}
