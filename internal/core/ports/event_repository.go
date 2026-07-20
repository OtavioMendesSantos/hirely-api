package ports

import (
	"context"
	"hirely-api/internal/core/domain"
)

type EventRepository interface {
	Create(ctx context.Context, event *domain.Event) error
	GetByApplicationID(ctx context.Context, applicationID string) ([]*domain.Event, error)
}
