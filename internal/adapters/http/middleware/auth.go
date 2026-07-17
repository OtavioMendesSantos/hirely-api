package middleware

import (
	"context"
	"fmt"
	"hirely-api/internal/adapters/http/dto"
	"hirely-api/internal/adapters/logger"
	"log/slog"
	"net/http"
	"strings"

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
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				slog.Warn("Missing Authorization header",
					slog.String("traceId", logger.GetTraceID(r.Context())),
					slog.String("operation", "AuthMiddleware"),
				)
				dto.WriteError(w, http.StatusUnauthorized, "Missing authorization header", "UNAUTHENTICATED")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				slog.Warn("Invalid Authorization header format",
					slog.String("traceId", logger.GetTraceID(r.Context())),
					slog.String("operation", "AuthMiddleware"),
				)
				dto.WriteError(w, http.StatusUnauthorized, "Invalid authorization header format", "UNAUTHENTICATED")
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
					slog.String("traceId", logger.GetTraceID(r.Context())),
					slog.String("operation", "AuthMiddleware"),
					slog.String("error", err.Error()),
				)
				dto.WriteError(w, http.StatusUnauthorized, "Invalid or expired token", "UNAUTHENTICATED")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				slog.Warn("Invalid JWT claims format",
					slog.String("traceId", logger.GetTraceID(r.Context())),
					slog.String("operation", "AuthMiddleware"),
				)
				dto.WriteError(w, http.StatusUnauthorized, "Invalid token claims", "UNAUTHENTICATED")
				return
			}

			userID, ok := claims["sub"].(string)
			if !ok || userID == "" {
				slog.Warn("Missing sub claim in JWT token",
					slog.String("traceId", logger.GetTraceID(r.Context())),
					slog.String("operation", "AuthMiddleware"),
				)
				dto.WriteError(w, http.StatusUnauthorized, "Invalid token subject", "UNAUTHENTICATED")
				return
			}

			ctx := WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
