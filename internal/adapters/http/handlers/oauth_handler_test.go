package handlers

import (
	"bytes"
	"encoding/json"
	"hirely-api/internal/adapters/http/dto"
	"hirely-api/internal/core/services"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOAuthHandler_GoogleAuthURL(t *testing.T) {
	repo := newMockUserRepo()
	authService := services.NewAuthService(repo, "secret", time.Hour)
	handler := NewOAuthHandler(authService, "client-id", "client-secret", "http://localhost/callback")

	req := httptest.NewRequest("GET", "/v1/auth/google/url", nil)
	rec := httptest.NewRecorder()

	handler.GoogleAuthURL(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	url, ok := resp["url"]
	if !ok || url == "" {
		t.Fatalf("expected url in response")
	}

	if !strings.Contains(url, "client-id") {
		t.Errorf("url should contain client ID, got: %s", url)
	}
}

func TestOAuthHandler_GoogleLogin_InvalidJSON(t *testing.T) {
	repo := newMockUserRepo()
	authService := services.NewAuthService(repo, "secret", time.Hour)
	handler := NewOAuthHandler(authService, "client-id", "client-secret", "http://localhost/callback")

	req := httptest.NewRequest("POST", "/v1/auth/google/login", bytes.NewBuffer([]byte("{invalid-json")))
	rec := httptest.NewRecorder()

	handler.GoogleLogin(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestOAuthHandler_GoogleLogin_MissingCode(t *testing.T) {
	repo := newMockUserRepo()
	authService := services.NewAuthService(repo, "secret", time.Hour)
	handler := NewOAuthHandler(authService, "client-id", "client-secret", "http://localhost/callback")

	payload := map[string]string{
		"code": "",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/auth/google/login", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	handler.GoogleLogin(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
    
    var errResp dto.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Status != "INVALID_ARGUMENT" {
		t.Errorf("expected status INVALID_ARGUMENT, got %s", errResp.Error.Status)
	}
}
