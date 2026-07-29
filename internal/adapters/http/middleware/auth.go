package middleware

import (
	"context"
	"fmt"
	"hirely-api/internal/adapters/logger"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type userIDKey struct{}

// WithUserID injects the authenticated UserID into the context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// GetUserID retrieves the authenticated UserID from the context.
func GetUserID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if userID, ok := ctx.Value(userIDKey{}).(string); ok {
		return userID
	}
	return ""
}

// Auth creates an HTTP middleware that validates the Bearer JWT token in the Authorization header.
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			slog.Warn("Missing Authorization header",
				slog.String("traceId", logger.GetTraceID(c.Request.Context())),
				slog.String("operation", "AuthMiddleware"),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header", "code": "UNAUTHENTICATED"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			slog.Warn("Invalid Authorization header format",
				slog.String("traceId", logger.GetTraceID(c.Request.Context())),
				slog.String("operation", "AuthMiddleware"),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format", "code": "UNAUTHENTICATED"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			slog.Warn("Invalid or expired JWT token",
				slog.String("traceId", logger.GetTraceID(c.Request.Context())),
				slog.String("operation", "AuthMiddleware"),
				slog.String("error", err.Error()),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token", "code": "UNAUTHENTICATED"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			slog.Warn("Invalid JWT claims format",
				slog.String("traceId", logger.GetTraceID(c.Request.Context())),
				slog.String("operation", "AuthMiddleware"),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims", "code": "UNAUTHENTICATED"})
			c.Abort()
			return
		}

		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			slog.Warn("Missing sub claim in JWT token",
				slog.String("traceId", logger.GetTraceID(c.Request.Context())),
				slog.String("operation", "AuthMiddleware"),
			)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token subject", "code": "UNAUTHENTICATED"})
			c.Abort()
			return
		}

		// Save the userID in both contexts to ensure compatibility while transitioning
		ctx := WithUserID(c.Request.Context(), userID)
		c.Request = c.Request.WithContext(ctx)
		c.Set("userID", userID)
		c.Next()
	}
}
