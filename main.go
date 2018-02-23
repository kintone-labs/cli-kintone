package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/howeyc/gopass"
	"github.com/kintone/go-kintone"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/unicode"

	flags "github.com/jessevdk/go-flags"
)

// NAME of this package
const NAME = "cli-kintone"

// VERSION of this package
const VERSION = "0.9.0"

// IMPORT_ROW_LIMIT The maximum row will be import
const IMPORT_ROW_LIMIT = 100

// EXPORT_ROW_LIMIT The maximum row will be export
const EXPORT_ROW_LIMIT = 500

// Configure of this package
type Configure struct {
	Domain            string   `short:"d" long:"domain" default:"" description:"Domain name"`
	Login             string   `short:"u" long:"username" default:"" description:"Login name"`
	Password          string   `short:"p" long:"password"  default:"" description:"Password"`
	BasicAuthUser     string   `short:"U" long:"basic-username" default:"" description:"Basic authentication user name"`
	BasicAuthPassword string   `short:"P" long:"basic-password" default:"" description:"Basic authentication password"`
	APIToken          string   `short:"t" long:"api-token" default:"" description:"API token"`
	Format            string   `short:"o" long:"output-format" default:"csv" description:"Output format: 'json' or 'csv'"`
	Query             string   `short:"q" long:"query" default:"" description:"Query string"`
	AppID             uint64   `short:"a" long:"app-id" default:"0" description:"App ID"`
	Fields            []string `short:"c" long:"columns" description:"Field names (comma separated)"`
	FilePath          string   `short:"f" default:"" long:"input-file" description:"Input file path"`
	DeleteAll         bool     `short:"D" long:"delete-all" description:"Delete all records before inserting"`
	Encoding          string   `short:"e" long:"encoding" default:"utf-8" description:"Character encoding: 'utf-8', 'utf-16', 'utf-16be-with-signature', 'utf-16le-with-signature', 'sjis' or 'euc-jp'"`
	GuestSpaceID      uint64   `short:"g" long:"guest-space-id" default:"0" description:"Guest Space ID"`
	FileDir           string   `short:"b" default:"" long:"attachment-dir" description:"Attachment file directory"`
	Line              uint64   `short:"l" long:"line" default:"1" description:"The position index of data in the input file"`
	IsImport          bool     `long:"import" description:"Force import"`
	IsExport          bool     `long:"export" description:"Force export"`
}

var config Configure

// Column config
type Column struct {
	Code       string
	Type       string
	IsSubField bool
	Table      string
}

// Columns config
type Columns []*Column

func (p Columns) Len() int {
	return len(p)
}

func (p Columns) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p Columns) Less(i, j int) bool {
	p1 := p[i]
	code1 := p1.Code
	if p1.IsSubField {
		code1 = p1.Table
	}
	p2 := p[j]
	code2 := p2.Code
	if p2.IsSubField {
		code2 = p2.Table
	}
	if code1 == code2 {
		return p[i].Code < p[j].Code
	}
	return code1 < code2
}

func getFields(app *kintone.App) (map[string]*kintone.FieldInfo, error) {
	fields, err := app.Fields()
	if err != nil {
		return nil, err
	}
	return fields, nil
}

// set column information from fieldinfo
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
	default:
		return nil
	}
}

func main() {
	var err error

	_, err = flags.ParseArgs(&config, os.Args[1:])
	if err != nil {
		os.Exit(1)
	}
	if config.AppID == 0 || (config.APIToken == "" && (config.Domain == "" || config.Login == "")) {
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

	// Old logic without force import/export
	if config.IsImport == false && config.IsExport == false {
		if config.FilePath == "" {
			if config.Format == "json" {
				err = writeJSON(app, os.Stdout)
			} else {
				err = writeCsv(app, os.Stdout)
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
		if config.Format == "json" {
			err = writeJSON(app, os.Stdout)
		} else {
			err = writeCsv(app, os.Stdout)
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
