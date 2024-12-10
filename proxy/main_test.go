package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddressSearch(t *testing.T) {
	reqBody := `{"query": "Москва"}`
	req := httptest.NewRequest(http.MethodPost, "/api/address/search", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()
	AddressSearch(w, req)
	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected statuc 200, got %v", res.StatusCode)
	}
}

func TestAddressGeocode(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    GeocodeRequest
		expectedStatus int
	}{
		{
			name: "Valid request",
			requestBody: GeocodeRequest{
				Lat: "55.7558",
				Lng: "37.6173",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "Invalid request - missing lat",
			requestBody: GeocodeRequest{
				Lat: "",
				Lng: "37.6173",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Invalid request",
			requestBody: GeocodeRequest{
				Lat: "55.7558",
				Lng: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatalf("failed to marshal request body: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/address/geocode", bytes.NewBuffer(reqBody))
			w := httptest.NewRecorder()

			AddressGeocode(w, req)

			res := w.Result()
			if res.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %v, got %v", tt.expectedStatus, res.StatusCode)
			}
		})
	}
}
