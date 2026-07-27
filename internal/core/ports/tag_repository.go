package ports

import (
	"context"
	"hirely-api/internal/core/domain"
)

type TagRepository interface {
	Create(ctx context.Context, tag *domain.Tag) error
	FindByID(ctx context.Context, id string) (*domain.Tag, error)
	FindByIDs(ctx context.Context, ids []string) ([]*domain.Tag, error)
	ListByUserID(ctx context.Context, userID string) ([]*domain.Tag, error)
	Delete(ctx context.Context, id string) error
}
