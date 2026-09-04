package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"hirely-api/internal/core/domain"
)

// Minimal mock repo for auth test
type mockSessionRepo struct{}

func (m *mockSessionRepo) Create(ctx context.Context, session *domain.Session) error { return nil }
func (m *mockSessionRepo) FindByHash(ctx context.Context, hash string) (*domain.Session, error) {
	return nil, nil
}
func (m *mockSessionRepo) UpdateExpiresAt(ctx context.Context, hash string, expiresAt time.Time) error {
	return nil
}
func (m *mockSessionRepo) RevokeByHash(ctx context.Context, hash string) error { return nil }

type mockAPIKeyRepo struct{}

func (m *mockAPIKeyRepo) Create(ctx context.Context, apiKey *domain.APIKey) error { return nil }
func (m *mockAPIKeyRepo) FindByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	return nil, nil
}
func (m *mockAPIKeyRepo) RecordUsage(ctx context.Context, id, ip, userAgent string, usedAt time.Time) error {
	return nil
}
func (m *mockAPIKeyRepo) Revoke(ctx context.Context, id string) error { return nil }

func TestHybridAuth_MissingCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	sRepo := &mockSessionRepo{}
	aRepo := &mockAPIKeyRepo{}

	r.Use(HybridAuth(sRepo, aRepo))
	r.GET("/v1/users/me", func(c *gin.Context) {
		t.Fatal("should not reach inner handler")
	})

	req := httptest.NewRequest("GET", "/v1/users/me", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}
func (m *mockSessionRepo) RevokeAllByUserID(ctx context.Context, userID string) error { return nil }
func (m *mockAPIKeyRepo) FindByUserID(ctx context.Context, userID string) ([]*domain.APIKey, error) { return nil, nil }
func (m *mockAPIKeyRepo) FindByIDAndUserID(ctx context.Context, id, userID string) (*domain.APIKey, error) { return nil, nil }
