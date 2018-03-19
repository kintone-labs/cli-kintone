package main

import (
	"bytes"
	"testing"

	"github.com/kintone/go-kintone"
)

func TestImport1(t *testing.T) {
	data := "Text,Text_Area,Rich_text\n11,22,<div>aaaaaa</div>\n111,22,<div>dddddqqddss</div>\n211,22,<div>aaaaaa</div>"

	app := newApp()

	config.DeleteAll = true
	err := importFromCSV(app, bytes.NewBufferString(data))
	if err != nil {
		t.Fatal(err)
	}

	recs, err := app.GetRecords(nil, "order by $id desc")
	if err != nil {
		t.Error(err)
	}
	if len(recs) != 3 {
		t.Error("Invalid record count")
	}

	fields := recs[0].Fields
	if _, ok := fields["Text"].(kintone.SingleLineTextField); !ok {
		t.Error("Not a SingleLineTextField")
	}
	if fields["Text"] != kintone.SingleLineTextField("211") {
		t.Error("Text mismatch")
	}
	if _, ok := fields["Text_Area"].(kintone.MultiLineTextField); !ok {
		t.Error("Not a MultiLineTextField")
	}
	if fields["Text_Area"] != kintone.MultiLineTextField("22") {
		t.Error("Text_Area mismatch")
	}
	if _, ok := fields["Rich_text"].(kintone.RichTextField); !ok {
		t.Error("Not a RichTextField")
	}
	if fields["Rich_text"] != kintone.RichTextField("<div>aaaaaa</div>") {
		t.Error("Rich_text mismatch")
	}
}
