package domain

import "time"

type APIKey struct {
	ID            string
	UserID        string
	Name          string
	KeyHash       string
	UsageCount    int
	LastIP        *string
	LastUserAgent *string
	LastUsedAt    *time.Time
	Revoked       bool
	CreatedAt     time.Time
}
