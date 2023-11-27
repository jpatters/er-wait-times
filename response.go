package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type RawResponse struct {
	Data []Unknown `json:"data"`
}

type Unknown struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type TableData struct {
	ColumnHeaders []string `json:"columnHeaders"`
	RowData       []Row    `json:"rowData"`
}

type Row struct {
	CellData []Cell `json:"cellData"`
}

type Cell struct {
	Value string
}

type Response struct {
	PatientsInWaitingRoom   int
	MostUrgentCount         int
	MostUrgentTime          string
	UrgentCount             int
	UrgentTime              string
	LessThanUrgentCount     int
	LessThanUrgentTime      string
	PatientsBeingTreated    int
	TotalPatients           int
	PatientsWaitingTransfer int
	Time                    string
}

func parseResponse(res *http.Response) (Response, error) {
	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return Response{}, err
	}

	rawResponse := RawResponse{}
	err = json.Unmarshal(body, &rawResponse)
	if err != nil {
		return Response{}, err
	}

	rows := []Row{}
	for _, obj := range rawResponse.Data {
		if obj.Type == "Table" {
			table := TableData{}
			err = json.Unmarshal(obj.Data, &table)
			if err != nil {
				return Response{}, err
			}
			rows = append(rows, table.RowData...)
			break
		}
	}

	return parseRows(rows), nil
}

func parseRows(rows []Row) Response {
	loc, _ := time.LoadLocation("America/Halifax")
	return Response{
		PatientsInWaitingRoom:   unsafeStringtoInt(rows[0].CellData[1].Value),
		MostUrgentCount:         unsafeStringtoInt(rows[1].CellData[1].Value),
		MostUrgentTime:          rows[1].CellData[2].Value,
		UrgentCount:             unsafeStringtoInt(rows[2].CellData[1].Value),
		UrgentTime:              rows[2].CellData[2].Value,
		LessThanUrgentCount:     unsafeStringtoInt(rows[3].CellData[1].Value),
		LessThanUrgentTime:      rows[3].CellData[2].Value,
		PatientsBeingTreated:    unsafeStringtoInt(rows[4].CellData[1].Value),
		TotalPatients:           unsafeStringtoInt(rows[6].CellData[1].Value),
		PatientsWaitingTransfer: unsafeStringtoInt(rows[6].CellData[1].Value) - (unsafeStringtoInt(rows[0].CellData[1].Value) + unsafeStringtoInt(rows[4].CellData[1].Value)),
		Time:                    time.Now().In(loc).Format("1/2/2006 15:04:05"),
	}
}

func unsafeStringtoInt(str string) int {
	i, _ := strconv.Atoi(str)
	return i
}
