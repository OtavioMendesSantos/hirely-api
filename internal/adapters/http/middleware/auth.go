package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"hirely-api/internal/adapters/logger"
	"hirely-api/internal/core/ports"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type userIDKey struct{}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

func GetUserID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if userID, ok := ctx.Value(userIDKey{}).(string); ok {
		return userID
	}
	return ""
}

// HybridAuth cria o middleware que suporta API Keys no Header e Session Cookies.
func HybridAuth(sessionRepo ports.SessionRepository, apiKeyRepo ports.APIKeyRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Verifica se há uma API Key no header Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				tokenString := parts[1]
				if strings.HasPrefix(tokenString, "hirely_sk_") {
					handleAPIKeyAuth(c, tokenString, apiKeyRepo)
					return
				}
			}
		}

		// 2. Fallback para Sessões em Cookies (Web)
		cookieSid, err := c.Cookie("__Secure-sid")
		if err == nil && cookieSid != "" {
			handleSessionAuth(c, cookieSid, sessionRepo)
			return
		}

		// 3. Nem API Key nem Cookie válido
		slog.Warn("Missing or invalid authentication credentials",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
		)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Não autorizado", "code": "UNAUTHENTICATED"})
		c.Abort()
	}
}

func handleAPIKeyAuth(c *gin.Context, rawKey string, repo ports.APIKeyRepository) {
	hash := sha256.Sum256([]byte(rawKey))
	hashStr := hex.EncodeToString(hash[:])

	apiKey, err := repo.FindByHash(c.Request.Context(), hashStr)
	if err != nil || apiKey == nil || apiKey.Revoked {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "API Key inválida ou revogada"})
		c.Abort()
		return
	}

	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")

	// Fire-and-forget stat update
	go func(id, clientIP, userAgent string) {
		// Criar um novo contexto background para a goroutine
		repo.RecordUsage(context.Background(), id, clientIP, userAgent, time.Now())
	}(apiKey.ID, ip, ua)

	injectUserContext(c, apiKey.UserID)
}

func handleSessionAuth(c *gin.Context, sid string, repo ports.SessionRepository) {
	hash := sha256.Sum256([]byte(sid))
	hashStr := hex.EncodeToString(hash[:])

	session, err := repo.FindByHash(c.Request.Context(), hashStr)
	if err != nil || session == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sessão inválida"})
		c.Abort()
		return
	}

	if session.Revoked {
		c.SetCookie("__Secure-sid", "", -1, "/", "", true, true)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sessão revogada"})
		c.Abort()
		return
	}

	if time.Now().After(session.ExpiresAt) {
		c.SetCookie("__Secure-sid", "", -1, "/", "", true, true)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sessão expirada"})
		c.Abort()
		return
	}

	// Sliding window - update expiresAt if less than 5 days
	if time.Until(session.ExpiresAt) < 5*24*time.Hour {
		newExpires := time.Now().Add(15 * 24 * time.Hour)
		go func(hash string, exp time.Time) {
			repo.UpdateExpiresAt(context.Background(), hash, exp)
		}(hashStr, newExpires)
		c.SetCookie("__Secure-sid", sid, int(15*24*3600), "/", "", true, true)
	}

	injectUserContext(c, session.UserID)
}

func injectUserContext(c *gin.Context, userID string) {
	ctx := WithUserID(c.Request.Context(), userID)
	c.Request = c.Request.WithContext(ctx)
	c.Set("userID", userID)
	c.Next()
}
