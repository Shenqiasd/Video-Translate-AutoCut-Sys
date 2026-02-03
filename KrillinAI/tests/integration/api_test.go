package integration

import (
	"encoding/json"
	"net/http"
	"testing"
)

const baseURL = "http://127.0.0.1:8888"

func TestHealthCheck(t *testing.T) {
	resp, err := http.Get(baseURL + "/")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}
}

func TestHistoryAPI(t *testing.T) {
	resp, err := http.Get(baseURL + "/api/capability/history")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Assuming the API returns a generic response structure
	if _, ok := result["code"]; !ok {
		t.Logf("Response might not have 'code' field: %v", result)
	}
}
