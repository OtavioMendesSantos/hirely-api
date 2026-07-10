package postgres

import (
	"hirely-api/internal/core/domain"
	"time"
)

type UserModel struct {
	ID           string    `gorm:"type:uuid;primaryKey"`
	Name         string    `gorm:"type:varchar(255);not null"`
	Email        string    `gorm:"type:varchar(255);uniqueIndex; not null"`
	PasswordHash string    `gorm:"type:varchar(255);not null"`
	CreatedAt    time.Time `gorm:"type:timestamp;not null"`
}

func (UserModel) TableName() string {
	return "users"
}

func (m *UserModel) ToDomain() *domain.User {
	return &domain.User{
		ID:           m.ID,
		Name:         m.Name,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		CreatedAt:    m.CreatedAt,
	}
}

func FromDomain(u *domain.User) *UserModel {
	return &UserModel{
		ID:           u.ID,
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		CreatedAt:    u.CreatedAt,
	}
}
