package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"os"
	"github.com/kintone/go-kintone"
	"github.com/howeyc/gopass"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/unicode"
)

type Configure struct {
	login string
	password string
	basicAuthUser string
	basicAuthPassword string
	apiToken string
	domain string
	basic string
	format string
	query string
	appId uint64
	fields []string
	filePath string
	deleteAll bool
	encoding string
	guestSpaceId uint64
	fileDir string
}

var config Configure

const IMPORT_ROW_LIMIT = 100
const EXPORT_ROW_LIMIT = 500

type Column struct {
	Code        string
	Type        string
	IsSubField  bool
	Table       string
}

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
	switch config.encoding {
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
	var colNames string

	flag.StringVar(&config.login, "u", "", "Login name")
	flag.StringVar(&config.password, "p", "", "Password")
	flag.StringVar(&config.basicAuthUser, "U", "", "Basic authentication user name")
	flag.StringVar(&config.basicAuthPassword, "P", "", "Basic authentication password")
	flag.StringVar(&config.domain, "d", "", "Domain name")
	flag.StringVar(&config.apiToken, "t", "", "API token")
	flag.Uint64Var(&config.appId, "a", 0, "App ID")
	flag.Uint64Var(&config.guestSpaceId, "g", 0, "Guest Space ID")
	flag.StringVar(&config.format, "o", "csv", "Output format: 'json' or 'csv'(default)")
	flag.StringVar(&config.query, "q", "", "Query string")
	flag.StringVar(&colNames, "c", "", "Field names (comma separated)")
	flag.StringVar(&config.filePath, "f", "", "Input file path")
	flag.BoolVar(&config.deleteAll, "D", false, "Delete all records before insert")
	flag.StringVar(&config.encoding, "e", "utf-8", "Character encoding: 'utf-8'(default), 'utf-16', 'utf-16be-with-signature', 'utf-16le-with-signature', 'sjis' or 'euc-jp'")
	flag.StringVar(&config.fileDir, "b", "", "Attachment file directory")

	flag.Parse()

	if config.appId == 0 || (config.apiToken == "" && (config.domain == "" || config.login == "")) {
		flag.PrintDefaults()
		return
	}

	if !strings.Contains(config.domain, ".") {
		config.domain += ".cybozu.com"
	}

	if colNames != "" {
		config.fields = strings.Split(colNames, ",")
		for i, field := range config.fields {
			config.fields[i] = strings.TrimSpace(field)
		}
	}


	var app *kintone.App

	if config.basicAuthUser != "" && config.basicAuthPassword == "" {
		fmt.Printf("Basic authentication password: ")
		pass, _ := gopass.GetPasswd()
		config.basicAuthPassword = string(pass)
	}

	if config.apiToken == "" {
		if config.password == "" {
			fmt.Printf("Password: ")
			pass, _ := gopass.GetPasswd()
			config.password = string(pass)
		}

		app = &kintone.App{
			Domain:       config.domain,
			User:         config.login,
			Password:     config.password,
			AppId:        config.appId,
			GuestSpaceId: config.guestSpaceId,
		}
	} else {
		app = &kintone.App{
			Domain:       config.domain,
			ApiToken:     config.apiToken,
			AppId:        config.appId,
			GuestSpaceId: config.guestSpaceId,
		}
	}

	if config.basicAuthUser != "" {
		app.SetBasicAuth(config.basicAuthUser, config.basicAuthPassword)
	}

	var err error
	if config.filePath == "" {
		if config.format == "json" {
			err = writeJson(app, os.Stdout)
		} else {
			err = writeCsv(app, os.Stdout)
		}
	} else {
		var file *os.File
		file, err = os.Open(config.filePath)
		if err == nil {
			defer file.Close()
			err = readCsv(app, file)
		}
	}
	if err != nil {
		log.Fatal(err)
	}
}
