package middleware

import (
	"encoding/json"
	"hirely-api/internal/adapters/http/dto"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestAuth_MissingHeader(t *testing.T) {
	guard := Auth("secret")
	handler := guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach inner handler when token is missing")
	}))

	req := httptest.NewRequest("GET", "/v1/users/me", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}

	var errResp dto.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Status != "UNAUTHENTICATED" {
		t.Errorf("expected status UNAUTHENTICATED, got %s", errResp.Error.Status)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	guard := Auth("secret")
	handler := guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not reach inner handler when token is invalid")
	}))

	req := httptest.NewRequest("GET", "/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.string")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestAuth_Success(t *testing.T) {
	jwtSecret := "secret"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "user-999",
		"email": "test@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	tokenStr, _ := token.SignedString([]byte(jwtSecret))

	var extractedID string
	guard := Auth(jwtSecret)
	handler := guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		extractedID = GetUserID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/v1/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if extractedID != "user-999" {
		t.Errorf("expected GetUserID to return user-999, got %s", extractedID)
	}
}
