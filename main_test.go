package main

import (
	"os"
	"strconv"

	"github.com/kintone-labs/go-kintone"
)

func newApp() *kintone.App {
	appID, _ := strconv.ParseUint(os.Getenv("KINTONE_APP_ID"), 10, 64)

	return &kintone.App{
		Domain:   os.Getenv("KINTONE_DOMAIN"),
		User:     os.Getenv("KINTONE_USERNAME"),
		Password: os.Getenv("KINTONE_PASSWORD"),
		AppId:    appID,
	}
}
