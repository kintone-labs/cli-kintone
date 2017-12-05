package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"reflect"
	"time"

	"github.com/kintone/go-kintone"
)

const (
	// ConstBulkRequestLimitRecordOption set option: record per bulkRequest
	ConstBulkRequestLimitRecordOption = 100

	// ConstBulkRequestLimitRequest kintone limited: The request count per bulkRequest
	ConstBulkRequestLimitRequest = 20

	// ConstRecordsLimitPerRequest kintone limited: The records count per request
	ConstRecordsLimitPerRequest = 100
)

// BulkRequestItem item in the bulkRequest array
type BulkRequestItem struct {
	Method  string      `json:"method"`
	API     string      `json:"api"`
	Payload interface{} `json:"payload,string"`
}

// BulkRequests BulkRequests structure
type BulkRequests struct {
	Requests []*BulkRequestItem `json:"requests,string"`
}

// BulkRequestsError structure
type BulkRequestsError struct {
	HTTPStatus     string      `json:"-"`
	HTTPStatusCode int         `json:"-"`
	Message        string      `json:"message"` // Human readable message.
	ID             string      `json:"id"`      // A unique error ID.
	Code           string      `json:"code"`    // For machines.
	Errors         interface{} `json:"errors"`
}

// BulkRequestsErrors structure
type BulkRequestsErrors struct {
	HTTPStatus     string               `json:"-"`
	HTTPStatusCode int                  `json:"-"`
	Results        []*BulkRequestsError `json:"results"`
}

// DataResponseBulkPOST structure
type DataResponseBulkPOST struct {
	Results []interface{} `json:"results,string"`
}

// DataRequestRecordsPOST structure
type DataRequestRecordsPOST struct {
	App     uint64            `json:"app,string"`
	Records []*kintone.Record `json:"records"`
}

//DataRequestRecordPUT structure
type DataRequestRecordPUT struct {
	ID     uint64          `json:"id,string"`
	Record *kintone.Record `json:"record,string"`
}

// DataRequestRecordPUTByKey structure
type DataRequestRecordPUTByKey struct {
	UpdateKey *kintone.UpdateKey `json:"updateKey,string"`
	Record    *kintone.Record    `json:"record,string"`
}

// DataRequestRecordsPUT - Data which will be update in the kintone app
// DataRequestRecordsPUT.Records - array include DataRequestRecordPUTByKey and DataRequestRecordPUT
type DataRequestRecordsPUT struct {
	App     uint64        `json:"app,string"`
	Records []interface{} `json:"records"`
}

// SetRecord set data record for PUT method
func (recordsPut *DataRequestRecordsPUT) SetRecord(record *kintone.Record) {
	recordPut := &DataRequestRecordPUT{ID: record.Id(), Record: record}
	recordsPut.Records = append(recordsPut.Records, recordPut)

}

// SetRecordWithKey set data record for PUT method
func (recordsPut *DataRequestRecordsPUT) SetRecordWithKey(record *kintone.Record, keyCode string) {
	updateKey := &kintone.UpdateKey{FieldCode: keyCode, Field: record.Fields[keyCode].(kintone.UpdateKeyField)}
	delete(record.Fields, keyCode)
	recordPut := &DataRequestRecordPUTByKey{UpdateKey: updateKey, Record: record}
	recordsPut.Records = append(recordsPut.Records, recordPut)

}

// Request bulkRequest with multi method which included only one request
func (bulk *BulkRequests) Request(app *kintone.App) (*DataResponseBulkPOST, interface{}) {

	data, _ := json.Marshal(bulk)
	req, err := newRequest(app, "POST", "bulkRequest", bytes.NewReader(data))

	if err != nil {
		return nil, err
	}
	resp, err := Do(app, req)
	if err != nil {
		return nil, err
	}
	body, err22 := parseResponse(resp)
	if err22 != nil {
		return nil, err22
	}
	resp1, err := bulk.Decode(body)
	if err != nil {
		return nil, kintone.ErrInvalidResponse
	}
	return resp1, nil
}

// Decode for BulkRequests
func (bulk *BulkRequests) Decode(b []byte) (*DataResponseBulkPOST, error) {
	var rsp *DataResponseBulkPOST
	err := json.Unmarshal(b, &rsp)
	if err != nil {
		return nil, errors.New("Invalid JSON format: " + err.Error())
	}
	return rsp, nil
}

// ImportDataUpdate import data with update
func (bulk *BulkRequests) ImportDataUpdate(app *kintone.App, recordData *kintone.Record, keyField string) error {
	bulkReqLength := len(bulk.Requests)

	if bulkReqLength > ConstBulkRequestLimitRequest {
		return errors.New("The length of bulk request was too large, maximun is " + string(ConstBulkRequestLimitRequest) + " per request")
	}
	var dataPUT *DataRequestRecordsPUT
	if bulkReqLength > 0 {
		for i, bulkReqItem := range bulk.Requests {
			if bulkReqItem.Method != "PUT" {
				continue
			}
			// TODO: Check limit 100 record - kintone limit
			dataPUT = bulkReqItem.Payload.(*DataRequestRecordsPUT)
			if len(dataPUT.Records) == ConstRecordsLimitPerRequest {
				continue
			}
			if keyField != "" {
				dataPUT.SetRecordWithKey(recordData, keyField)
			} else {
				dataPUT.SetRecord(recordData)
			}
			bulk.Requests[i].Payload = dataPUT
			return nil
		}
	}

	recordsUpdate := make([]interface{}, 0)
	var recordPUT interface{}
	if keyField != "" {
		updateKey := &kintone.UpdateKey{FieldCode: keyField, Field: recordData.Fields[keyField].(kintone.UpdateKeyField)}
		delete(recordData.Fields, keyField)
		recordPUT = &DataRequestRecordPUTByKey{UpdateKey: updateKey, Record: recordData}
	} else {
		recordPUT = &DataRequestRecordPUT{ID: recordData.Id(), Record: recordData}
	}
	recordsUpdate = append(recordsUpdate, recordPUT)
	dataPUT = &DataRequestRecordsPUT{App: app.AppId, Records: recordsUpdate}
	requestPUTRecords := &BulkRequestItem{"PUT", "/k/v1/records.json", dataPUT}
	bulk.Requests = append(bulk.Requests, requestPUTRecords)

	return nil

}

// ImportDataInsert import data with insert only
func (bulk *BulkRequests) ImportDataInsert(app *kintone.App, recordData *kintone.Record) error {
	bulkReqLength := len(bulk.Requests)

	if bulkReqLength > ConstBulkRequestLimitRequest {
		return errors.New("The length of bulk request was too large, maximun is " + string(ConstBulkRequestLimitRequest) + " per request")
	}
	var dataPOST *DataRequestRecordsPOST
	if bulkReqLength > 0 {
		for i, bulkReqItem := range bulk.Requests {
			if bulkReqItem.Method != "POST" {
				continue
			}
			dataPOST = bulkReqItem.Payload.(*DataRequestRecordsPOST)
			if len(dataPOST.Records) == ConstRecordsLimitPerRequest {
				continue
			}
			// TODO: Check limit 100 record - kintone limit
			dataPOST.Records = append(dataPOST.Records, recordData)
			bulk.Requests[i].Payload = dataPOST
			return nil
		}
	}
	recordsInsert := make([]*kintone.Record, 0)
	recordsInsert = append(recordsInsert, recordData)
	dataPOST = &DataRequestRecordsPOST{app.AppId, recordsInsert}
	requestPostRecords := &BulkRequestItem{"POST", "/k/v1/records.json", dataPOST}
	bulk.Requests = append(bulk.Requests, requestPostRecords)

	return nil

}

func newRequest(app *kintone.App, method, api string, body io.Reader) (*http.Request, error) {
	var path string
	if app.GuestSpaceId == 0 {
		path = fmt.Sprintf("/k/v1/%s.json", api)
	} else {
		path = fmt.Sprintf("/k/guest/%d/v1/%s.json", app.GuestSpaceId, api)
	}

	u := url.URL{
		Scheme: "https",
		Host:   app.Domain,
		Path:   path,
	}
	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if len(app.ApiToken) == 0 {
		req.Header.Set("X-Cybozu-Authorization", base64.StdEncoding.EncodeToString([]byte(app.User+":"+app.Password)))
	} else {
		req.Header.Set("X-Cybozu-API-Token", app.ApiToken)
	}

	if len(app.GetUserAgentHeader()) != 0 {
		req.Header.Set("User-Agent", app.GetUserAgentHeader())
	}

	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// Do Request to webservice
func Do(app *kintone.App, req *http.Request) (*http.Response, error) {
	if app.Client == nil {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return nil, err
		}
		app.Client = &http.Client{Jar: jar}
	}
	if app.Timeout == time.Duration(0) {
		app.Timeout = kintone.DEFAULT_TIMEOUT
	}

	type result struct {
		resp *http.Response
		err  error
	}
	done := make(chan result, 1)
	go func() {
		resp, err := app.Client.Do(req)
		done <- result{resp, err}
	}()

	type requestCanceler interface {
		CancelRequest(*http.Request)
	}

	select {
	case r := <-done:
		return r.resp, r.err
	case <-time.After(app.Timeout):
		if canceller, ok := app.Client.Transport.(requestCanceler); ok {
			canceller.CancelRequest(req)
		} else {
			go func() {
				r := <-done
				if r.err == nil {
					r.resp.Body.Close()
				}
			}()
		}
		return nil, kintone.ErrTimeout
	}
}
func isJSON(contentType string) bool {
	mediatype, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return mediatype == "application/json"
}

func parseResponse(resp *http.Response) ([]byte, interface{}) {
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		if !isJSON(resp.Header.Get("Content-Type")) {
			return nil, &kintone.AppError{
				HttpStatus:     resp.Status,
				HttpStatusCode: resp.StatusCode,
			}
		}

		var appErrorBulkRequest BulkRequestsErrors
		appErrorBulkRequest.HTTPStatus = resp.Status
		appErrorBulkRequest.HTTPStatusCode = resp.StatusCode
		json.Unmarshal(body, &appErrorBulkRequest)

		if len(appErrorBulkRequest.Results) == 0 {
			var appErrorRequest BulkRequestsError
			appErrorRequest.HTTPStatus = resp.Status
			appErrorRequest.HTTPStatusCode = resp.StatusCode
			json.Unmarshal(body, &appErrorRequest)

			return nil, &appErrorRequest
		}
		return nil, &appErrorBulkRequest
	}
	return body, nil
}

// ErrorResponse show error detail when the bulkRequest has error(s)
type ErrorResponse struct {
	ID      string
	Code    string
	Status  string
	Message string
	Errors  interface{}
}

func (err *ErrorResponse) show(prefix string) {
	fmt.Println("ID: ", err.ID)
	fmt.Println("Code: ", err.Code)
	if err.Status != "" {
		fmt.Println("Status: ", err.Status)
	}
	fmt.Println("Message: ", err.Message)
	fmt.Printf(prefix + "Errors detail: ")
	if err.Errors != nil {
		fmt.Printf("\n")
		for indx, val := range err.Errors.(map[string]interface{}) {
			fieldMessage := val.(map[string]interface{})
			detailMessage := fieldMessage["messages"].([]interface{})
			fmt.Printf("%v  '%v': ", prefix, indx)
			for i, mess := range detailMessage {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf(mess.(string))
			}
			fmt.Printf("\n")
		}
		fmt.Printf("\n")
	} else {
		fmt.Printf("(none)\n\n")
	}

}

// HandelResponse for bulkRequest
func (bulk *BulkRequests) HandelResponse(rep *DataResponseBulkPOST, err interface{}, lastRowImport, rowNumber uint64) {

	if err != nil {
		CLIMessage := ""
		method := map[string]string{"POST": "INSERT", "PUT": "UPDATE"}
		methodOccuredError := ""
		if reflect.TypeOf(err).String() == "*main.BulkRequestsError" {
			errorResp := &ErrorResponse{}
			errorResp.Status = fmt.Sprintf("%v - %v", err.(*BulkRequestsError).HTTPStatus, err.(*BulkRequestsError).Message)
			errorResp.Message = "Please check your params config"
			errorResp.Errors = err.(*BulkRequestsError).Errors
			errorResp.ID = err.(*BulkRequestsError).ID
			errorResp.Code = err.(*BulkRequestsError).Code
		} else {

			errorsResp := err.(*BulkRequestsErrors)
			CLIMessage = fmt.Sprintf("ERROR.\nFor error details, please read the details above.\n")
			CLIMessage += fmt.Sprintf("Lines %d to %d of the imported file contain errors. Please fix the errors on the file, and re-import it with the flag \"-l %d\"\n", lastRowImport, rowNumber, lastRowImport)
			fmt.Printf(" => ERROR OCCURRED\n")
			fmt.Println("Status: ", errorsResp.HTTPStatus)
			for idx, errorItem := range errorsResp.Results {
				if errorItem.Code == "" {
					continue
				}
				errorResp := &ErrorResponse{}
				errorResp.ID = errorItem.ID
				errorResp.Code = errorItem.Code
				errorResp.Message = errorItem.Message
				errorResp.Errors = errorItem.Errors

				errorResp.show("")
				methodOccuredError = method[bulk.Requests[idx].Method]
			}
		}
		fmt.Printf("%v: ", time.Now().Format("[2006-01-02 15:04:05]"))
		fmt.Printf("PROCESS STOPPED!\n\n")
		if CLIMessage != "" {
			fmt.Println(methodOccuredError, CLIMessage)
		}
		os.Exit(502)
	}
	fmt.Println(" => SUCCESS")
}
