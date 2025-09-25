package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	metoffice "github.com/rm-hull/metoffice-uk-weather-overlays/internal/models/met_office"
)

type DataHubClient interface {
	GetLatest(orderId string) (*metoffice.Response, error)
	GetLatestDataFile(orderId, fileId string) (io.ReadCloser, error)
}

type DataHubManager struct {
	baseUrl string
	apiKey  string
	client  *http.Client
}

func NewDataHubClient(apiKey string) DataHubClient {
	return &DataHubManager{
		baseUrl: "https://data.hub.api.metoffice.gov.uk/map-images/1.0.0",
		apiKey:  apiKey,
		client:  &http.Client{},
	}
}

func (mgr *DataHubManager) GetLatest(orderId string) (*metoffice.Response, error) {
	url := fmt.Sprintf("%s/orders/%s/latest?dataSpec=1.1.0", mgr.baseUrl, orderId)
	body, err := mgr.get(url, "application/json")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := body.Close(); err != nil {
			log.Printf("failed to close body: %v", err)
		}
	}()

	var resp metoffice.Response
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

func (mgr *DataHubManager) GetLatestDataFile(orderId, fileId string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/orders/%s/latest/%s/data?dataSpec=1.1.0", mgr.baseUrl, orderId, fileId)
	return mgr.get(url, "image/png")
}

func (mgr *DataHubManager) get(url string, acceptHeader string) (io.ReadCloser, error) {
	log.Printf("Retrieving: %s", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("apikey", mgr.apiKey)
	req.Header.Set("Accept", acceptHeader)

	res, err := mgr.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from %s: %w", url, err)
	}

	if res.StatusCode > 299 {
		if err := res.Body.Close(); err != nil {
			log.Printf("failed to close body: %v", err)
		}
		return nil, fmt.Errorf("http status response from %s: %s", url, res.Status)
	}

	return res.Body, nil
}
