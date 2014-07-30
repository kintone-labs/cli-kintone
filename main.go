package main

import (
	"github.com/ryokdy/go-kintone"
	"github.com/howeyc/gopass"
	"flag"
	"log"
	"fmt"
	"strings"
)

var login string
var password string
var apiToken string
var domain string
var basic string
var format string
var query string
var appId uint64
var encoding string
var fields []string
var filePath string
var deleteAll bool

const ROW_LIMIT = 100

func main() {
	var colNames string
	
	flag.StringVar(&login, "u", "", "Login name")
	flag.StringVar(&password, "p", "", "Password")
	flag.StringVar(&domain, "d", "", "Domain name")
	flag.StringVar(&apiToken, "t", "", "API token")
	flag.Uint64Var(&appId, "a", 0, "App ID")
	flag.StringVar(&format, "o", "csv", "Output format: 'json' or 'csv'(default)")
	flag.StringVar(&query, "q", "", "Query string")
	flag.StringVar(&colNames, "c", "", "Field names (comma separated)")
	flag.StringVar(&filePath, "f", "", "Input file path")
	flag.BoolVar(&deleteAll, "D", false, "Delete all records before insert")
	flag.StringVar(&encoding, "e", "utf8", "Character encoding: 'utf8'(default), 'sjis' or 'euc'")
	
    flag.Parse()

	if appId == 0 || (apiToken == "" && (domain == "" || login == "")) {
		flag.PrintDefaults()
		return
	}
	
	if !strings.Contains(domain, ".") {
		domain = domain + ".cybozu.com"
	}

	if colNames != "" {
		fields = strings.Split(colNames, ",")
	}

	var app *kintone.App
	
	if apiToken == "" {
		if password == "" {
			fmt.Printf("Password: ")
			password = string(gopass.GetPasswd())
		}

		app = &kintone.App{
			Domain:   domain,
			User:     login,
			Password: password,
			AppId:    appId,
		}
	} else {
		app = &kintone.App{
			Domain:   domain,
			ApiToken: apiToken,
			AppId:    appId,
		}
	}

	var err error
	if filePath == "" {
		if format == "json" {
			err = writeJson(app)
		} else {
			err = writeCsv(app)
		}
	} else {
		err = readCsv(app, filePath)
	}
	if err != nil {
		log.Fatal(err)
	}
}

