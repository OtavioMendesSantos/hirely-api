package postgres

import (
	"hirely-api/internal/core/domain"
	"time"
)

type TagModel struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	UserID    string    `gorm:"type:uuid;not null;uniqueIndex:idx_user_tag_name"`
	User      UserModel `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Name      string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_user_tag_name"`
	ColorHex  string    `gorm:"type:varchar(7);not null"`
	CreatedAt time.Time `gorm:"type:timestamp;not null"`
}

func (TagModel) TableName() string {
	return "tags"
}

func (m *TagModel) ToDomain() *domain.Tag {
	return &domain.Tag{
		ID:        m.ID,
		UserID:    m.UserID,
		Name:      m.Name,
		ColorHex:  m.ColorHex,
		CreatedAt: m.CreatedAt,
	}
}

func TagFromDomain(t *domain.Tag) *TagModel {
	return &TagModel{
		ID:        t.ID,
		UserID:    t.UserID,
		Name:      t.Name,
		ColorHex:  t.ColorHex,
		CreatedAt: t.CreatedAt,
	}
}
