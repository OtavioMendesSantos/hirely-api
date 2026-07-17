package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler_Check(t *testing.T) {
	handler := NewHealthHandler()

	req := httptest.NewRequest("GET", "/v1/health", nil)
	rec := httptest.NewRecorder()

	handler.Check(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var resp HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal JSON response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %s", resp.Status)
	}
	if resp.Service != "hirely-api" {
		t.Errorf("expected service hirely-api, got %s", resp.Service)
	}
	if resp.Timestamp == "" {
		t.Errorf("expected timestamp to not be empty")
	}
}
