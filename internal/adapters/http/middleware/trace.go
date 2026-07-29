package middleware

import (
	"hirely-api/internal/adapters/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Request-ID")
		if traceID == "" {
			traceID = c.GetHeader("traceparent")
		}
		if traceID == "" {
			traceID = uuid.New().String()
		}

		ctx := logger.WithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)
		c.Header("X-Trace-ID", traceID)

		c.Next()
	}
}
