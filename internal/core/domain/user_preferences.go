package domain

import "time"

type UserPreferences struct {
	UserID          string    `json:"-"`
	Theme           string    `json:"theme"`
	AutoArchiveDays int       `json:"auto_archive_days"`
	ConfettiEnabled bool      `json:"confetti_enabled"`
	CreatedAt       time.Time `json:"-"`
	UpdatedAt       time.Time `json:"-"`
}

type UpdateUserPreferences struct {
	Theme           *string
	AutoArchiveDays *int
	ConfettiEnabled *bool
}

func DefaultUserPreferences(userID string) *UserPreferences {
	return &UserPreferences{UserID: userID, Theme: "system", ConfettiEnabled: true, AutoArchiveDays: 60}
}
