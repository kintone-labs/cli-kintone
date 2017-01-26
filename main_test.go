
package main

import (
	"os"
  "strconv"
  "github.com/kintone/go-kintone"
)

func newApp() *kintone.App {
  appId, _ := strconv.ParseUint(os.Getenv("KINTONE_APP_ID"), 10, 64)

  return &kintone.App{
		Domain:   os.Getenv("KINTONE_DOMAIN"),
		ApiToken: os.Getenv("KINTONE_API_TOKEN"),
		AppId:    appId,
	}
}
