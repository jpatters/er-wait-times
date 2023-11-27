package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func appendToSheet(location string, data Response) error {
	ctx := context.Background()
	b, err := ioutil.ReadFile("creds.json")
	if err != nil {
		return fmt.Errorf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/drive", "https://www.googleapis.com/auth/drive.file", "https://www.googleapis.com/auth/spreadsheets")
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
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
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
