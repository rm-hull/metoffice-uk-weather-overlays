package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	metoffice "github.com/rm-hull/metoffice-uk-weather-overlays/models/met_office"
)

type DataHubClient interface {
	GetLatest(orderId string) (*metoffice.Response, error)
	GetLatestDataFile(orderId, fileId string) (io.ReadCloser, error)
}

type DataHubManager struct {
	baseUrl string
	apiKey  string
}

func NewDataHubClient(apiKey string) DataHubClient {
	return &DataHubManager{
		baseUrl: "https://data.hub.api.metoffice.gov.uk/map-images/1.0.0",
		apiKey:  apiKey,
	}
}

func (mgr *DataHubManager) GetLatest(orderId string) (*metoffice.Response, error) {
	url := fmt.Sprintf("%s/orders/%s/latest", mgr.baseUrl, orderId)
	log.Printf("Retrieving: %s", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("apikey", mgr.apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from %s: %w", url, err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Printf("failed to close request body: %v", err)
		}
	}()

	if res.StatusCode > 299 {
		return nil, fmt.Errorf("http status response from %s: %s", url, res.Status)
	}

	var resp metoffice.Response
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

func (mgr *DataHubManager) GetLatestDataFile(orderId, fileId string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/orders/%s/latest/%s/data", mgr.baseUrl, orderId, fileId)
	log.Printf("Retrieving: %s", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("apikey", mgr.apiKey)
	req.Header.Set("Accept", "image/png")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from %s: %w", url, err)
	}

	if res.StatusCode > 299 {
		return nil, fmt.Errorf("http status response from %s: %s", url, res.Status)
	}

	return res.Body, nil
}
