package main

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

type RawResponse struct {
	Data []TableElement `json:"data"`
}

type TableElement struct {
	Type     string         `json:"type"`
	Data     TableData      `json:"data"`
	Children []TableElement `json:"children"`
}

type TableData struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Text  string `json:"text"`
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

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return Response{}, err
	}

	rawResponse := RawResponse{}
	err = json.Unmarshal(body, &rawResponse)
	if err != nil {
		return Response{}, err
	}

	response := Response{}
	for _, obj := range rawResponse.Data {
		if obj.Type == "TableV2" {
			response.PatientsInWaitingRoom = unsafeStringtoInt(obj.Children[1].Children[1].Data.Text)
			response.MostUrgentCount = unsafeStringtoInt(obj.Children[2].Children[1].Data.Text)
			response.MostUrgentTime = obj.Children[2].Children[2].Data.Text
			response.UrgentCount = unsafeStringtoInt(obj.Children[3].Children[1].Data.Text)
			response.UrgentTime = obj.Children[3].Children[2].Data.Text
			response.LessThanUrgentCount = unsafeStringtoInt(obj.Children[4].Children[1].Data.Text)
			response.LessThanUrgentTime = obj.Children[4].Children[2].Data.Text
			response.PatientsBeingTreated = unsafeStringtoInt(obj.Children[5].Children[1].Data.Text)
			response.TotalPatients = unsafeStringtoInt(obj.Children[7].Children[1].Data.Text)
			response.PatientsWaitingTransfer = unsafeStringtoInt(obj.Children[7].Children[1].Data.Text) - (unsafeStringtoInt(obj.Children[1].Children[1].Data.Text) + unsafeStringtoInt(obj.Children[5].Children[1].Data.Text))
		}
	}

	return response, nil
}

func unsafeStringtoInt(str string) int {
	i, _ := strconv.Atoi(str)
	return i
}
