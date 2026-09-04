package domain

import "time"

type Session struct {
	ID        string
	UserID    string
	Hash      string
	IP        *string
	UserAgent *string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}
