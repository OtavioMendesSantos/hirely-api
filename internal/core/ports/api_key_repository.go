package ports

import (
	"context"
	"hirely-api/internal/core/domain"
	"time"
)

type APIKeyRepository interface {
	Create(ctx context.Context, key *domain.APIKey) error
	FindByHash(ctx context.Context, hash string) (*domain.APIKey, error)
	FindByUserID(ctx context.Context, userID string) ([]*domain.APIKey, error)
	FindByIDAndUserID(ctx context.Context, id, userID string) (*domain.APIKey, error)
	Revoke(ctx context.Context, id string) error
	RecordUsage(ctx context.Context, id string, ip, ua string, usedAt time.Time) error
}
