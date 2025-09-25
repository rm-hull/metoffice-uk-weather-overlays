package internal

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/orders/test-order/latest?dataSpec=1.1.0", r.URL.Path+"?"+r.URL.RawQuery)
			assert.Equal(t, "test-key", r.Header.Get("apikey"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockResponseJSON))
		}))
		defer server.Close()

		mgr := &DataHubManager{
			baseUrl: server.URL,
			apiKey:  "test-key",
			client:  server.Client(),
		}

		resp, err := mgr.GetLatest("test-order")
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "test-order", resp.OrderDetails.Order.OrderId)
		assert.Len(t, resp.OrderDetails.Files, 1)
		assert.Equal(t, "file1", resp.OrderDetails.Files[0].FileId)
	})

	t.Run("API error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/orders/test-order/latest?dataSpec=1.1.0", r.URL.Path+"?"+r.URL.RawQuery)
			assert.Equal(t, "test-key", r.Header.Get("apikey"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal Server Error"))
		}))
		defer server.Close()

		mgr := &DataHubManager{
			baseUrl: server.URL,
			apiKey:  "test-key",
			client:  server.Client(),
		}

		resp, err := mgr.GetLatest("test-order")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Equal(t, fmt.Sprintf("http status response from %s/orders/test-order/latest?dataSpec=1.1.0: 500 Internal Server Error", server.URL), err.Error())
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/orders/test-order/latest?dataSpec=1.1.0", r.URL.Path+"?"+r.URL.RawQuery)
			assert.Equal(t, "test-key", r.Header.Get("apikey"))
			assert.Equal(t, "application/json", r.Header.Get("Accept"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"invalid json"`))
		}))
		defer server.Close()

		mgr := &DataHubManager{
			baseUrl: server.URL,
			apiKey:  "test-key",
			client:  server.Client(),
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
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/orders/test-order/latest/test-file/data?dataSpec=1.1.0", r.URL.Path+"?"+r.URL.RawQuery)
			assert.Equal(t, "test-key", r.Header.Get("apikey"))
			assert.Equal(t, "image/png", r.Header.Get("Accept"))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockFileData))
		}))
		defer server.Close()

		mgr := &DataHubManager{
			baseUrl: server.URL,
			apiKey:  "test-key",
			client:  server.Client(),
		}

        reader, err := mgr.GetLatestDataFile("test-order", "test-file")
        require.NoError(t, err)
        require.NotNil(t, reader)
        defer reader.Close()

        data, err := io.ReadAll(reader)
        require.NoError(t, err)
        assert.Equal(t, mockFileData, string(data))
	})

	t.Run("API error response for data file", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/orders/test-order/latest/test-file/data?dataSpec=1.1.0", r.URL.Path+"?"+r.URL.RawQuery)
			assert.Equal(t, "test-key", r.Header.Get("apikey"))
			assert.Equal(t, "image/png", r.Header.Get("Accept"))
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Not Found"))
		}))
		defer server.Close()

		mgr := &DataHubManager{
			baseUrl: server.URL,
			apiKey:  "test-key",
			client:  server.Client(),
		}

		reader, err := mgr.GetLatestDataFile("test-order", "test-file")
		assert.Error(t, err)
		assert.Nil(t, reader)
		assert.Equal(t, fmt.Sprintf("http status response from %s/orders/test-order/latest/test-file/data?dataSpec=1.1.0: 404 Not Found", server.URL), err.Error())
	})
}