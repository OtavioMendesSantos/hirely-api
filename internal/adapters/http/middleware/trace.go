package middleware

import (
	"hirely-api/internal/adapters/logger"
	"net/http"

	"github.com/google/uuid"
)

func Trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Request-ID")
		if traceID == "" {
			traceID = r.Header.Get("traceparent")
		}
		if traceID == "" {
			traceID = uuid.New().String()
		}

		ctx := logger.WithTraceID(r.Context(), traceID)
		w.Header().Set("X-Trace-ID", traceID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
