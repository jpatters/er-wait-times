package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var endpoint = "https://wdf.princeedwardisland.ca/api/workflow"
var locations = []string{
	"QEH",
	"PCH",
}

func main() {
	file, err := os.OpenFile("waittimes.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Unable to open log file: %v", err)
	}
	defer file.Close()
	log.SetOutput(file)
	for _, loc := range locations {
		err := fetch(loc)
		if err != nil {
			log.Println(err)
		}
	}
}

func fetch(location string) error {
	reqBody := buildRequest(location)

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(reqBody)
	if err != nil {
		return err
	}

	res, err := doPost(endpoint, buf)
	if err != nil {
		return err
	}

	parsedResponse, err := parseResponse(res)
	if err != nil {
		return err
	}

	return appendToSheet(location, parsedResponse)
}

type Request struct {
	AppName     string `json:"appName"`
	FeatureName string `json:"featureName"`
	MetaVars    Meta   `json:"metaVars"`
	QueryVars   Query  `json:"queryVars"`
	QueryName   string `json:"queryName"`
}

type Meta struct {
	ServiceID    *string `json:"service_id"`
	SaveLocation *string `json:"save_location"`
}

type Query struct {
	Service  string `json:"service"`
	Activity string `json:"activity"`
}

func buildRequest(location string) Request {
	location = fmt.Sprintf("ERWaitTimes_%s", location)
	return Request{
		AppName:     location,
		FeatureName: location,
		MetaVars:    Meta{nil, nil},
		QueryVars:   Query{location, location},
		QueryName:   location,
	}
}

func doPost(endpoint string, data io.Reader) (*http.Response, error) {
	client := http.Client{
		Timeout: time.Second * 10, // Timeout after 2 seconds
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, data)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	return client.Do(req)
}

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

	loc, _ := time.LoadLocation("America/Halifax")
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
			response.Time = time.Now().In(loc).Format("1/2/2006 15:04:05")
		}
	}

	return response, nil
}

func unsafeStringtoInt(str string) int {
	i, _ := strconv.Atoi(str)
	return i
}

func appendToSheet(location string, data Response) error {
	ctx := context.Background()
	creds, err := os.ReadFile("/ermon/config/creds.json")
	if err != nil {
		return fmt.Errorf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON([]byte(creds), "https://www.googleapis.com/auth/drive", "https://www.googleapis.com/auth/drive.file", "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	sheetsService, err := sheets.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return err
	}

	// The ID of the spreadsheet to update.
	spreadsheetId := "1Cv33IZFeCOqnGeb9xb6nC8HQbfgkmB7Mo86QetyDtEQ" // TODO: Update placeholder value.

	// The A1 notation of a range to search for a logical table of data.
	// Values will be appended after the last row of the table.
	range2 := fmt.Sprintf("%s!B1:C1", location) // TODO: Update placeholder value.

	// How the input data should be interpreted.
	valueInputOption := "USER_ENTERED" // TODO: Update placeholder value.

	// How the input data should be inserted.
	insertDataOption := "INSERT_ROWS" // TODO: Update placeholder value.

	rb := &sheets.ValueRange{
		MajorDimension: "ROWS",
		Range:          fmt.Sprintf("%s!B1:C1", location),
		Values:         transposeResults(data),
	}

	_, err = sheetsService.Spreadsheets.Values.Append(spreadsheetId, range2, rb).ValueInputOption(valueInputOption).InsertDataOption(insertDataOption).Context(ctx).Do()
	return err
}

func transposeResults(data Response) [][]interface{} {
	v := reflect.ValueOf(data)

	values := make([]interface{}, v.NumField())

	for i := 0; i < v.NumField(); i++ {
		values[i] = v.Field(i).Interface()
	}
	return [][]interface{}{values}
}

func getClient(config *oauth2.Config) *http.Client {
	// The file token.json stores the user's access and refresh tokens, and is
	// created automatically when the authorization flow completes for the first
	// time.
	t, _ := os.ReadFile("/ermon/config/token.json")
	tok := &oauth2.Token{}
	json.Unmarshal([]byte(t), tok)
	return config.Client(context.Background(), tok)
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
