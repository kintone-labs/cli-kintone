package main

import (
	"fmt"
	"testing"

	"github.com/kintone/go-kintone"
)

func newAppTest(id uint64) *kintone.App {
	return &kintone.App{
		Domain:   os.Getenv("KINTONE_DOMAIN"),
		User:     os.Getenv("KINTONE_USERNAME"),
		Password: os.Getenv("KINTONE_PASSWORD"),
		AppId:    id,
	}
}

func TestRequest(t *testing.T) {

	bulkReq := &BulkRequests{}
	app := newAppTest(16)
	bulkReq.Requests = make([]*BulkRequestItem, 0)

	/// INSERT
	records := make([]*kintone.Record, 0)
	record1 := kintone.NewRecord(map[string]interface{}{
		"Text": kintone.SingleLineTextField("test 11!"),
		"_2":   kintone.SingleLineTextField("test 21!"),
	})
	records = append(records, record1)
	record2 := kintone.NewRecord(map[string]interface{}{
		"Text": kintone.SingleLineTextField("test 22!"),
		"_2":   kintone.SingleLineTextField("test 22!"),
	})
	records = append(records, record2)
	dataPOST := &DataRequestRecordsPOST{app.AppId, records}
	postRecords := &BulkRequestItem{"POST", "/k/v1/records.json", dataPOST}

	bulkReq.Requests = append(bulkReq.Requests, postRecords)

	/// UPDATE
	recordsUpdate := make([]interface{}, 0)
	recordsUpdate1 := kintone.NewRecordWithId(4902, map[string]interface{}{
		"Text": kintone.SingleLineTextField("test NNN!"),
		"_2":   kintone.SingleLineTextField("test MMM!"),
	})
	fmt.Println(recordsUpdate1)
	recordsUpdate = append(recordsUpdate, &DataRequestRecordPUT{ID: recordsUpdate1.Id(),
		Record: recordsUpdate1})

	recordsUpdate2 := kintone.NewRecordWithId(4903, map[string]interface{}{
		"Text": kintone.SingleLineTextField("test 123!"),
		"_2":   kintone.SingleLineTextField("test 234!"),
	})
	recordsUpdate = append(recordsUpdate, &DataRequestRecordPUT{
		ID: recordsUpdate2.Id(), Record: recordsUpdate2})

	dataPUT := &DataRequestRecordsPUT{app.AppId, recordsUpdate}
	putRecords := &BulkRequestItem{"PUT", "/k/v1/records.json", dataPUT}

	bulkReq.Requests = append(bulkReq.Requests, putRecords)

	rs, err := bulkReq.Request(app)

	if err != nil {
		t.Error(" failed", err)
	} else {
		t.Log(rs)
	}
}
