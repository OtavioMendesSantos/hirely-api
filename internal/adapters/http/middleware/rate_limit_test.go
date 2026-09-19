package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestFixedWindowLimiter_Allow(t *testing.T) {
	l := NewFixedWindowLimiter(2, time.Minute)
	defer l.Stop()

	if !l.Allow("1.2.3.4") {
		t.Fatal("first request should be allowed")
	}
	if !l.Allow("1.2.3.4") {
		t.Fatal("second request should be allowed")
	}
	if l.Allow("1.2.3.4") {
		t.Fatal("third request should be rejected")
	}
	// Outra chave tem quota independente.
	if !l.Allow("5.6.7.8") {
		t.Fatal("request from another key should be allowed")
	}
}

func TestFixedWindowLimiter_WindowReset(t *testing.T) {
	l := NewFixedWindowLimiter(1, 20*time.Millisecond)
	defer l.Stop()

	if !l.Allow("key") {
		t.Fatal("first request should be allowed")
	}
	if l.Allow("key") {
		t.Fatal("second request within window should be rejected")
	}
	time.Sleep(40 * time.Millisecond)
	if !l.Allow("key") {
		t.Fatal("request after window reset should be allowed")
	}
}

func TestRateLimit_Returns429(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := NewFixedWindowLimiter(1, time.Minute)
	defer limiter.Stop()
	r := gin.New()
	r.Use(RateLimit(limiter))
	r.POST("/v1/mcp", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest("POST", "/v1/mcp", nil)
	req.RemoteAddr = "9.9.9.9:1234"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 on first request, got %d", rec.Code)
	}

	req2 := httptest.NewRequest("POST", "/v1/mcp", nil)
	req2.RemoteAddr = "9.9.9.9:5678"
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on second request, got %d", rec2.Code)
	}
}
