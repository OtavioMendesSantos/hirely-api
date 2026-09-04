package postgres

import (
	"hirely-api/internal/core/domain"
	"time"
)

type SessionModel struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	UserID    string    `gorm:"type:uuid;not null;index"`
	User      UserModel `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Hash      string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	IP        *string   `gorm:"type:varchar(45)"`
	UserAgent *string   `gorm:"type:text"`
	ExpiresAt time.Time `gorm:"type:timestamp;not null"`
	Revoked   bool      `gorm:"default:false"`
	CreatedAt time.Time `gorm:"type:timestamp;not null"`
}

func (SessionModel) TableName() string {
	return "sessions"
}

func (m *SessionModel) ToDomain() *domain.Session {
	return &domain.Session{
		ID:        m.ID,
		UserID:    m.UserID,
		Hash:      m.Hash,
		IP:        m.IP,
		UserAgent: m.UserAgent,
		ExpiresAt: m.ExpiresAt,
		Revoked:   m.Revoked,
		CreatedAt: m.CreatedAt,
	}
}

func SessionFromDomain(s *domain.Session) *SessionModel {
	return &SessionModel{
		ID:        s.ID,
		UserID:    s.UserID,
		Hash:      s.Hash,
		IP:        s.IP,
		UserAgent: s.UserAgent,
		ExpiresAt: s.ExpiresAt,
		Revoked:   s.Revoked,
		CreatedAt: s.CreatedAt,
	}
}
