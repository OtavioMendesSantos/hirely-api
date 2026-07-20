package postgres

import (
	"context"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/ports"

	"gorm.io/gorm"
)

type EventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(ctx context.Context, event *domain.Event) error {
	model := EventFromDomain(event)
	result := r.db.WithContext(ctx).Create(model)
	return result.Error
}

func (r *EventRepository) GetByApplicationID(ctx context.Context, applicationID string) ([]*domain.Event, error) {
	var models []EventModel
	result := r.db.WithContext(ctx).Where("application_id = ?", applicationID).Order("created_at asc").Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}
	events := make([]*domain.Event, len(models))
	for i, m := range models {
		events[i] = m.ToDomain()
	}
	return events, nil
}

var _ ports.EventRepository = (*EventRepository)(nil)
