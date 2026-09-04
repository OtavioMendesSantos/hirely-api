package ports

import (
	"context"
	"hirely-api/internal/core/domain"
	"time"
)

type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	FindByHash(ctx context.Context, hash string) (*domain.Session, error)
	RevokeByHash(ctx context.Context, hash string) error
	RevokeAllByUserID(ctx context.Context, userID string) error
	UpdateExpiresAt(ctx context.Context, hash string, expiresAt time.Time) error
}
