package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORS_PreflightOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.Any("/v1/users:login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("OPTIONS", "/v1/users:login", nil)
	req.Header.Set("Origin", "http://localhost:4200")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 No Content for OPTIONS preflight, got %d", rec.Code)
	}

	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:4200" {
		t.Errorf("expected origin http://localhost:4200, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Errorf("expected Access-Control-Allow-Methods header to be set")
	}
}

func TestCORS_PostRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.POST("/v1/users:login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/v1/users:login", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 from wrapped handler, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected fallback origin *, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}
