package postgres

import (
	"context"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/ports"
	"time"

	"gorm.io/gorm"
)

type sessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) ports.SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, session *domain.Session) error {
	model := SessionFromDomain(session)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *sessionRepository) FindByHash(ctx context.Context, hash string) (*domain.Session, error) {
	var model SessionModel
	if err := r.db.WithContext(ctx).Where("hash = ?", hash).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *sessionRepository) RevokeByHash(ctx context.Context, hash string) error {
	return r.db.WithContext(ctx).Model(&SessionModel{}).Where("hash = ?", hash).Update("revoked", true).Error
}

func (r *sessionRepository) RevokeAllByUserID(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&SessionModel{}).Where("user_id = ?", userID).Update("revoked", true).Error
}

func (r *sessionRepository) UpdateExpiresAt(ctx context.Context, hash string, expiresAt time.Time) error {
	return r.db.WithContext(ctx).Model(&SessionModel{}).Where("hash = ?", hash).Update("expires_at", expiresAt).Error
}
