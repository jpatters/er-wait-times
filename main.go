package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/jpatters/er-wait-times/migrate"
	_ "github.com/lib/pq"
)

var endpoint = "https://wdf.princeedwardisland.ca/api/workflow"
var locations = []string{
	"QEH",
	"PCH",
}

var db *sqlx.DB
var (
	dbHost     = os.Getenv("DB_HOST")
	dbUser     = os.Getenv("DB_USER")
	dbPassword = os.Getenv("DB_PASSWORD")
	dbName     = os.Getenv("DB_NAME")
	dbOptions  = os.Getenv("DB_OPTIONS")
)

func main() {
	var err error
	db, err = sqlx.Connect("postgres", fmt.Sprintf("host=%s password=%s user=%s dbname=%s sslmode=require options=%s", dbHost, dbPassword, dbUser, dbName, dbOptions))
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	err = migrate.Migrate(db)
	if err != nil {
		log.Fatalln(err)
	}

	file, err := os.OpenFile("waittimes.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Unable to open log file: %v", err)
	}
	defer file.Close()
	log.SetOutput(file)
	for _, loc := range locations {
		result, err := fetch(loc)
		if err != nil {
			log.Println(err)
		}

		_, err = db.NamedExec(`
			INSERT INTO waittime (
				location,
				patients_in_waiting_room_count,
				most_urgent_count,
				most_urgent_waittime,
				most_urgent_waittime_max,
				urgent_count,
				urgent_waittime,
				urgent_waittime_max,
				less_urgent_count,
				less_urgent_waittime,
				less_urgent_waittime_max,
				patients_being_treated_count,
				total_patients_count,
				patients_waiting_transfer_count,
				created_at
			) VALUES (
				:location,
				:patients_in_waiting_room_count,
				:most_urgent_count,
				:most_urgent_waittime,
				:most_urgent_waittime_max,
				:urgent_count,
				:urgent_waittime,
				:urgent_waittime_max,
				:less_urgent_count,
				:less_urgent_waittime,
				:less_urgent_waittime_max,
				:patients_being_treated_count,
				:total_patients_count,
				:patients_waiting_transfer_count,
				:created_at
			)`, result)

		if err != nil {
			log.Println(err)
		}
	}
}

func fetch(location string) (*Response, error) {
	reqBody := buildRequest(location)

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(reqBody)
	if err != nil {
		return nil, err
	}

	res, err := doPost(endpoint, buf)
	if err != nil {
		return nil, err
	}

	result, err := parseResponse(res)
	if err != nil {
		return nil, err
	}

	result.Location = location

	return result, nil
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
	Location                string `db:"location"`
	PatientsInWaitingRoom   int    `db:"patients_in_waiting_room_count"`
	MostUrgentCount         int    `db:"most_urgent_count"`
	MostUrgentTime          string `db:"most_urgent_waittime"`
	MostUrgentTimeMax       int    `db:"most_urgent_waittime_max"`
	UrgentCount             int    `db:"urgent_count"`
	UrgentTime              string `db:"urgent_waittime"`
	UrgentTimeMax           int    `db:"urgent_waittime_max"`
	LessThanUrgentCount     int    `db:"less_urgent_count"`
	LessThanUrgentTime      string `db:"less_urgent_waittime"`
	LessThanUrgentTimeMax   int    `db:"less_urgent_waittime_max"`
	PatientsBeingTreated    int    `db:"patients_being_treated_count"`
	TotalPatients           int    `db:"total_patients_count"`
	PatientsWaitingTransfer int    `db:"patients_waiting_transfer_count"`
	Time                    string `db:"created_at"`
}

func parseResponse(res *http.Response) (*Response, error) {
	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	rawResponse := &RawResponse{}
	err = json.Unmarshal(body, &rawResponse)
	if err != nil {
		return nil, err
	}

	loc, _ := time.LoadLocation("America/Halifax")
	response := &Response{}
	for _, obj := range rawResponse.Data {
		if obj.Type == "TableV2" {
			response.PatientsInWaitingRoom = unsafeStringtoInt(obj.Children[1].Children[1].Data.Text)
			response.MostUrgentCount = unsafeStringtoInt(obj.Children[2].Children[1].Data.Text)
			response.MostUrgentTime = obj.Children[2].Children[2].Data.Text
			response.MostUrgentTimeMax = stringToInt(obj.Children[2].Children[2].Data.Text)
			response.UrgentCount = unsafeStringtoInt(obj.Children[3].Children[1].Data.Text)
			response.UrgentTime = obj.Children[3].Children[2].Data.Text
			response.UrgentTimeMax = stringToInt(obj.Children[3].Children[2].Data.Text)
			response.LessThanUrgentCount = unsafeStringtoInt(obj.Children[4].Children[1].Data.Text)
			response.LessThanUrgentTime = obj.Children[4].Children[2].Data.Text
			response.LessThanUrgentTimeMax = stringToInt(obj.Children[4].Children[2].Data.Text)
			response.PatientsBeingTreated = unsafeStringtoInt(obj.Children[5].Children[1].Data.Text)
			response.TotalPatients = unsafeStringtoInt(obj.Children[7].Children[1].Data.Text)
			response.PatientsWaitingTransfer = unsafeStringtoInt(obj.Children[7].Children[1].Data.Text) - (unsafeStringtoInt(obj.Children[1].Children[1].Data.Text) + unsafeStringtoInt(obj.Children[5].Children[1].Data.Text))
			response.Time = time.Now().In(loc).Format("1/2/2006 15:04:05")
		}
	}

	return response, nil
}

func stringToInt(str string) int {
	switch str {
	case "< 1 hour":
		return 1
	case "1-2 hours":
		return 2
	case "2-3 hours":
		return 3
	case "3-4 hours":
		return 4
	case "4-5 hours":
		return 5
	case "5-6 hours":
		return 6
	case "6-7 hours":
		return 7
	case "7-8 hours":
		return 8
	case "8-9 hours":
		return 9
	case "9-10 hours":
		return 10
	case "> 10 hours":
		return 11
	default:
		return 0
	}
}

func unsafeStringtoInt(str string) int {
	i, _ := strconv.Atoi(str)
	return i
}
