package main

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
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
