package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

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
