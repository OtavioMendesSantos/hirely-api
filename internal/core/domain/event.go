package domain

import (
	"time"
)

type EventType string

const (
	EventTypeAutomatic EventType = "AUTOMATIC"
	EventTypeManual    EventType = "MANUAL"
)

type Event struct {
	ID             string    `json:"id"`
	ApplicationID  string    `json:"applicationId"`
	Type           EventType `json:"type"`
	Description    string    `json:"description"`
	PreviousStatus string    `json:"previousStatus,omitempty"`
	NewStatus      string    `json:"newStatus,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

func NewAutomaticEvent(id, applicationID, desc, prevStatus, newStatus string) *Event {
	return &Event{
		ID:             id,
		ApplicationID:  applicationID,
		Type:           EventTypeAutomatic,
		Description:    desc,
		PreviousStatus: prevStatus,
		NewStatus:      newStatus,
		CreatedAt:      time.Now().UTC(),
	}
}

func NewManualEvent(id, applicationID, desc string) *Event {
	return &Event{
		ID:             id,
		ApplicationID:  applicationID,
		Type:           EventTypeManual,
		Description:    desc,
		CreatedAt:      time.Now().UTC(),
	}
}
