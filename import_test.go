
package main

import (
	"bytes"
	//"io"
  //"fmt"
	"testing"
  "github.com/kintone/go-kintone"
)

func TestImport1(t *testing.T) {
  data := "\"single_line_text\",\"multi_line_text\",\"number\",\"date_and_time\"\n\"single line2\",\"multi line2\nmulti line\",\"12345\",\"2016-09-12T10:13:00Z\"\n\"single line1\",\"multi line1\nmulti line\",\"12345\",\"2016-09-12T10:13:00Z\""

	app := newApp()

	config.deleteAll = true
  err := readCsv(app, bytes.NewBufferString(data))
	if err != nil {
		t.Error(err)
	}

	recs, err := app.GetRecords(nil, "order by record_number desc")
	if err != nil {
		t.Error(err)
	}
	if len(recs) != 2 {
		t.Error("Invalid record count")
	}

	fields := recs[0].Fields
	if _, ok := fields["single_line_text"].(kintone.SingleLineTextField); !ok {
		t.Error("Not a SingleLineTextField")
	}
	if fields["single_line_text"] != kintone.SingleLineTextField("single line1") {
		t.Error("single_line_text mismatch")
	}
	if _, ok := fields["multi_line_text"].(kintone.MultiLineTextField); !ok {
		t.Error("Not a MultiLineTextField")
	}
	if fields["multi_line_text"] != kintone.MultiLineTextField("multi line1\nmulti line") {
		t.Error("multi_line_text mismatch")
	}
	num, ok := fields["number"].(kintone.DecimalField)
	if !ok {
		t.Error("Not a DecimalField")
	}
	if num != kintone.DecimalField("12345") {
		t.Error("number mismatch")
	}
}

func TestImport2(t *testing.T) {
  data := "\"*\",\"single_line_text\",\"multi_line_text\",\"number\",\"date_and_time\",\"table_single_line_text\",\"table_multi_line_text\"\n\"*\",\"single line2\",\"multi line2\nmulti line\",\"12345\",\"2016-09-12T10:13:00Z\",\"single1\",\"multi1\"\n\"\",\"single line2\",\"multi line2\nmulti line\",\"12345\",\"2016-09-12T10:13:00Z\",\"single2\",\"multi2\"\n\"*\",\"single line1\",\"multi line1\nmulti line\",\"12345\",\"2016-09-12T10:13:00Z\",\"\",\"\""

	app := newApp()

	config.deleteAll = true
  err := readCsv(app, bytes.NewBufferString(data))
	if err != nil {
		t.Error(err)
	}

	recs, err := app.GetRecords(nil, "order by record_number asc")
	if err != nil {
		t.Error(err)
	}
	if len(recs) != 2 {
		t.Error("Invalid record count")
	}

	fields := recs[0].Fields
	if _, ok := fields["single_line_text"].(kintone.SingleLineTextField); !ok {
		t.Error("Not a SingleLineTextField")
	}
	if fields["single_line_text"] != kintone.SingleLineTextField("single line2") {
		t.Error("single_line_text mismatch")
	}
	if _, ok := fields["multi_line_text"].(kintone.MultiLineTextField); !ok {
		t.Error("Not a MultiLineTextField")
	}
	if fields["multi_line_text"] != kintone.MultiLineTextField("multi line2\nmulti line") {
		t.Error("multi_line_text mismatch")
	}
	num, ok := fields["number"].(kintone.DecimalField)
	if !ok {
		t.Error("Not a DecimalField")
	}
	if num != kintone.DecimalField("12345") {
		t.Error("number mismatch")
	}
	table, ok := fields["table"].(kintone.SubTableField)
	if !ok {
		t.Error("Not a SubTableField")
	}
	if len(table) != 2 {
		t.Error("Invalid sub record count")
	}
	sub := table[0].Fields
	if _, ok := sub["table_single_line_text"].(kintone.SingleLineTextField); !ok {
		t.Error("Not a SingleLineTextField")
	}
	if sub["table_single_line_text"] != kintone.SingleLineTextField("single1") {
		t.Error("single_line_text mismatch")
	}
}
