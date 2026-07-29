package middleware

import (
	"hirely-api/internal/adapters/logger"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTrace_InjectsOrGeneratesTraceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var ctxTraceID string

	r := gin.New()
	r.Use(Trace())
	r.POST("/v1/users", func(c *gin.Context) {
		ctxTraceID = logger.GetTraceID(c.Request.Context())
		c.Status(http.StatusOK)
	})

	// Case 1: Custom X-Request-ID provided
	req := httptest.NewRequest("POST", "/v1/users", nil)
	req.Header.Set("X-Request-ID", "test-uuid-1234")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Header().Get("X-Trace-ID") != "test-uuid-1234" {
		t.Errorf("expected X-Trace-ID response header test-uuid-1234, got %s", rec.Header().Get("X-Trace-ID"))
	}
	if ctxTraceID != "test-uuid-1234" {
		t.Errorf("expected context trace ID test-uuid-1234, got %s", ctxTraceID)
	}

	// Case 2: No header provided -> generates UUID
	req2 := httptest.NewRequest("POST", "/v1/users", nil)
	rec2 := httptest.NewRecorder()

	r.ServeHTTP(rec2, req2)

	if rec2.Header().Get("X-Trace-ID") == "" {
		t.Errorf("expected generated X-Trace-ID response header, got empty")
	}
	if ctxTraceID == "" {
		t.Errorf("expected generated trace ID in context, got empty")
	}
}
