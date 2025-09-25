package internal

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// MockHTTPClient is a mock implementation of http.Client for testing
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func TestDataHubManager_GetLatest(t *testing.T) {
	// Mock successful response
	mockResponseJSON := `{
		"orderDetails": {
			"order": {
				"orderId": "test-order",
				"name": "Test Order",
				"modelId": "uk_model",
				"format": "png",
				"productType": "weather_overlay"
			},
			"files": [
				{
					"fileId": "file1",
					"runDateTime": "2025-01-01T00:00:00Z",
					"run": "00"
				}
			]
		}
	}`

	t.Run("successful response", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockResponseJSON)),
					Header:     make(http.Header),
				}, nil
			},
		}

		mgr := &DataHubManager{
			baseUrl: "http://test-url",
			apiKey:  "test-key",
			client:  mockClient,
		}

		resp, err := mgr.GetLatest("test-order")
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "test-order", resp.OrderDetails.Order.OrderId)
		assert.Len(t, resp.OrderDetails.Files, 1)
		assert.Equal(t, "file1", resp.OrderDetails.Files[0].FileId)
	})

	t.Run("API error response", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Status:     "500 Internal Server Error",
					Body:       io.NopCloser(bytes.NewBufferString("Internal Server Error")),
					Header:     make(http.Header),
				}, nil
			},
		}

		mgr := &DataHubManager{
			baseUrl: "http://test-url",
			apiKey:  "test-key",
			client:  mockClient,
		}

		resp, err := mgr.GetLatest("test-order")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, "http status response from http://test-url/orders/test-order/latest?dataSpec=1.1.0: 500 Internal Server Error", err.Error())
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"invalid json"`)),
					Header:     make(http.Header),
				}, nil
			},
		}

		mgr := &DataHubManager{
			baseUrl: "http://test-url",
			apiKey:  "test-key",
			client:  mockClient,
		}

		resp, err := mgr.GetLatest("test-order")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "failed to unmarshal response")
	})
}

func TestDataHubManager_GetLatestDataFile(t *testing.T) {
	mockFileData := "this is mock image data"

	t.Run("successful data file retrieval", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(mockFileData)),
					Header:     make(http.Header),
				}, nil
			},
		}

		mgr := &DataHubManager{
			baseUrl: "http://test-url",
			apiKey:  "test-key",
			client:  mockClient,
		}

		reader, err := mgr.GetLatestDataFile("test-order", "test-file")
		assert.NoError(t, err)
		assert.NotNil(t, reader)

		data, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, mockFileData, string(data))
		assert.NoError(t, reader.Close())
	})

	t.Run("API error response for data file", func(t *testing.T) {
		mockClient := &MockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Status:     "404 Not Found",
					Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
					Header:     make(http.Header),
				}, nil
			},
		}

		mgr := &DataHubManager{
			baseUrl: "http://test-url",
			apiKey:  "test-key",
			client:  mockClient,
		}

		reader, err := mgr.GetLatestDataFile("test-order", "test-file")
		assert.Error(t, err)
		assert.Nil(t, reader)
		assert.Equal(t, "http status response from http://test-url/orders/test-order/latest/test-file/data?dataSpec=1.1.0: 404 Not Found", err.Error())
	})
}
