package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"runtime"

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
const VERSION = "0.10.1"

// IMPORT_ROW_LIMIT The maximum row will be import
const IMPORT_ROW_LIMIT = 100

// EXPORT_ROW_LIMIT The maximum row will be export
const EXPORT_ROW_LIMIT = 500

// Configure of this package
type Configure struct {
	Domain            string   `short:"d" default:"" description:"Domain name (specify the FQDN)"`
	AppID             uint64   `short:"a" default:"0" description:"App ID"`
	Login             string   `short:"u" default:"" description:"User's log in name"`
	Password          string   `short:"p" default:"" description:"User's password"`
	APIToken          string   `short:"t" default:"" description:"API token"`
	GuestSpaceID      uint64   `short:"g" default:"0" description:"Guest Space ID"`
	Format            string   `short:"o" default:"csv" description:"Output format. Specify either 'json' or 'csv'"`
	Encoding          string   `short:"e" default:"utf-8" description:"Character encoding. Specify one of the following -> 'utf-8'(default), 'utf-16', 'utf-16be-with-signature', 'utf-16le-with-signature', 'sjis' or 'euc-jp'"`
	BasicAuthUser     string   `short:"U" default:"" description:"Basic authentication user name"`
	BasicAuthPassword string   `short:"P" default:"" description:"Basic authentication password"`
	Query             string   `short:"q" default:"" description:"Query string"`
	Fields            []string `short:"c" description:"Fields to export (comma separated). Specify the field code name"`
	FilePath          string   `short:"f" default:"" description:"Input file path"`
	FileDir           string   `short:"b" default:"" description:"Attachment file directory"`
	DeleteAll         bool     `short:"D" description:"Delete records before insert. You can specify the deleting record condition by option \"-q\""`
	Line              uint64   `short:"l" default:"1" description:"Position index of data in the input file"`
	IsImport          bool     `long:"import" description:"Import data from stdin. If \"-f\" is also specified, data is imported from the file instead"`
	IsExport          bool     `long:"export" description:"Export kintone data to stdout"`
	Version           bool     `short:"v" long:"version" description:"Version of cli-kintone"`
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
