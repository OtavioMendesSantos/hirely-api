package postgres

import (
	"hirely-api/internal/core/domain"
	"time"
)

type APIKeyModel struct {
	ID            string     `gorm:"type:uuid;primaryKey"`
	UserID        string     `gorm:"type:uuid;not null;index"`
	User          UserModel  `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Name          string     `gorm:"type:varchar(100);not null"`
	KeyHash       string     `gorm:"type:varchar(255);not null;uniqueIndex"`
	UsageCount    int        `gorm:"default:0"`
	LastIP        *string    `gorm:"type:varchar(45)"`
	LastUserAgent *string    `gorm:"type:text"`
	LastUsedAt    *time.Time `gorm:"type:timestamp"`
	Revoked       bool       `gorm:"default:false"`
	CreatedAt     time.Time  `gorm:"type:timestamp;not null"`
}

func (APIKeyModel) TableName() string {
	return "api_keys"
}

func (m *APIKeyModel) ToDomain() *domain.APIKey {
	return &domain.APIKey{
		ID:            m.ID,
		UserID:        m.UserID,
		Name:          m.Name,
		KeyHash:       m.KeyHash,
		UsageCount:    m.UsageCount,
		LastIP:        m.LastIP,
		LastUserAgent: m.LastUserAgent,
		LastUsedAt:    m.LastUsedAt,
		Revoked:       m.Revoked,
		CreatedAt:     m.CreatedAt,
	}
}

func APIKeyFromDomain(k *domain.APIKey) *APIKeyModel {
	return &APIKeyModel{
		ID:            k.ID,
		UserID:        k.UserID,
		Name:          k.Name,
		KeyHash:       k.KeyHash,
		UsageCount:    k.UsageCount,
		LastIP:        k.LastIP,
		LastUserAgent: k.LastUserAgent,
		LastUsedAt:    k.LastUsedAt,
		Revoked:       k.Revoked,
		CreatedAt:     k.CreatedAt,
	}
}
