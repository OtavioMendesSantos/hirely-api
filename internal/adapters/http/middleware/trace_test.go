package middleware

import (
	"hirely-api/internal/adapters/logger"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTrace_InjectsOrGeneratesTraceID(t *testing.T) {
	var ctxTraceID string
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxTraceID = logger.GetTraceID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := Trace(dummyHandler)

	// Case 1: Custom X-Request-ID provided
	req := httptest.NewRequest("POST", "/v1/users", nil)
	req.Header.Set("X-Request-ID", "test-uuid-1234")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Trace-ID") != "test-uuid-1234" {
		t.Errorf("expected X-Trace-ID response header test-uuid-1234, got %s", rec.Header().Get("X-Trace-ID"))
	}
	if ctxTraceID != "test-uuid-1234" {
		t.Errorf("expected context trace ID test-uuid-1234, got %s", ctxTraceID)
	}

	// Case 2: No header provided -> generates UUID
	req2 := httptest.NewRequest("POST", "/v1/users", nil)
	rec2 := httptest.NewRecorder()

	handler.ServeHTTP(rec2, req2)

	if rec2.Header().Get("X-Trace-ID") == "" {
		t.Errorf("expected generated X-Trace-ID response header, got empty")
	}
	if ctxTraceID == "" {
		t.Errorf("expected generated trace ID in context, got empty")
	}
}
