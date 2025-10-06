package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	metoffice "github.com/rm-hull/metoffice-uk-weather-overlays/internal/models/met_office"
)

type DataHubClient interface {
	GetLatest(orderId string, params QueryParams) (*metoffice.Response, error)
	GetLatestDataFile(orderId, fileId string, params QueryParams) (io.ReadCloser, error)
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

func (mgr *DataHubManager) GetLatest(orderId string, params QueryParams) (*metoffice.Response, error) {
	url := fmt.Sprintf("%s/orders/%s/latest%s", mgr.baseUrl, orderId, params.toString())
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

func (mgr *DataHubManager) GetLatestDataFile(orderId, fileId string, params QueryParams) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/orders/%s/latest/%s/data%s", mgr.baseUrl, orderId, fileId, params.toString())
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
		_ = res.Body.Close()
		return nil, fmt.Errorf("http status response from %s: %s", url, res.Status)
	}
	return res.Body, nil
}

type QueryParams map[string]string

func NewQueryParams(keypairs ...string) QueryParams {
	if len(keypairs)%2 != 0 {
		panic("NewQueryParams requires an even number of arguments")
	}
	params := make(QueryParams)
	for i := 0; i < len(keypairs); i += 2 {
		params[keypairs[i]] = keypairs[i+1]
	}
	return params
}

func (q *QueryParams) Add(key string, value string) {
	(*q)[key] = value
}

func (q *QueryParams) toString() string {
	if q == nil || len(*q) == 0 {
		return ""
	}

	values := make(url.Values)
	for k, v := range *q {
		values.Set(k, v)
	}

	// The Encode method handles URL encoding and joins with '&'.
	// It also sorts keys for deterministic output.
	return "?" + values.Encode()

}
