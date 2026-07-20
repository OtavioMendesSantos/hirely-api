package postgres

import (
	"hirely-api/internal/core/domain"
	"time"
)

type EventModel struct {
	ID             string    `gorm:"type:uuid;primaryKey"`
	ApplicationID  string    `gorm:"type:uuid;not null;index"`
	Type           string    `gorm:"type:varchar(20);not null"`
	Description    string    `gorm:"type:text;not null"`
	PreviousStatus string    `gorm:"type:varchar(50)"`
	NewStatus      string    `gorm:"type:varchar(50)"`
	CreatedAt      time.Time `gorm:"type:timestamp;not null"`
}

func (EventModel) TableName() string {
	return "events"
}

func (m *EventModel) ToDomain() *domain.Event {
	return &domain.Event{
		ID:             m.ID,
		ApplicationID:  m.ApplicationID,
		Type:           domain.EventType(m.Type),
		Description:    m.Description,
		PreviousStatus: m.PreviousStatus,
		NewStatus:      m.NewStatus,
		CreatedAt:      m.CreatedAt,
	}
}

func EventFromDomain(e *domain.Event) *EventModel {
	return &EventModel{
		ID:             e.ID,
		ApplicationID:  e.ApplicationID,
		Type:           string(e.Type),
		Description:    e.Description,
		PreviousStatus: e.PreviousStatus,
		NewStatus:      e.NewStatus,
		CreatedAt:      e.CreatedAt,
	}
}
