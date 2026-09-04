package postgres

import (
	"context"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/ports"
	"time"

	"gorm.io/gorm"
)

type apiKeyRepository struct {
	db *gorm.DB
}

func NewAPIKeyRepository(db *gorm.DB) ports.APIKeyRepository {
	return &apiKeyRepository{db: db}
}

func (r *apiKeyRepository) Create(ctx context.Context, key *domain.APIKey) error {
	model := APIKeyFromDomain(key)
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *apiKeyRepository) FindByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	var model APIKeyModel
	if err := r.db.WithContext(ctx).Where("key_hash = ?", hash).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}

func (r *apiKeyRepository) Revoke(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&APIKeyModel{}).Where("id = ?", id).Update("revoked", true).Error
}

func (r *apiKeyRepository) RecordUsage(ctx context.Context, id string, ip, ua string, usedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&APIKeyModel{}).Where("id = ?", id).Updates(map[string]interface{}{
		"usage_count":     gorm.Expr("usage_count + 1"),
		"last_ip":         ip,
		"last_user_agent": ua,
		"last_used_at":    usedAt,
	}).Error
}
func (r *apiKeyRepository) FindByUserID(ctx context.Context, userID string) ([]*domain.APIKey, error) {
	var models []APIKeyModel
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&models).Error; err != nil {
		return nil, err
	}
	var keys []*domain.APIKey
	for _, m := range models {
		keys = append(keys, m.ToDomain())
	}
	return keys, nil
}

func (r *apiKeyRepository) FindByIDAndUserID(ctx context.Context, id, userID string) (*domain.APIKey, error) {
	var model APIKeyModel
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return model.ToDomain(), nil
}
