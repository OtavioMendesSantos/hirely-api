package ports

import (
	"context"
	"hirely-api/internal/core/domain"
)

type ApplicationRepository interface {
	Create(ctx context.Context, app *domain.Application) error
	FindByID(ctx context.Context, id string) (*domain.Application, error)
	ListByUserID(ctx context.Context, userID string, search string, orderBy string, orderDir string) ([]*domain.Application, error)
	ListByUserIDWithFilters(ctx context.Context, userID string, search string, statuses []string, orderBy string, orderDir string) ([]*domain.Application, error)
	Update(ctx context.Context, app *domain.Application) error
	Delete(ctx context.Context, id string) error
	// UpdateStatus encapsula a atualização do status e a inserção do evento em uma transaction DB
	UpdateStatus(ctx context.Context, app *domain.Application, event *domain.Event) error
}
