package postgres

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/ports"
)

type UserPreferencesRepository struct{ db *gorm.DB }

func NewUserPreferencesRepository(db *gorm.DB) *UserPreferencesRepository {
	return &UserPreferencesRepository{db: db}
}

func (r *UserPreferencesRepository) FindByUserID(ctx context.Context, userID string) (*domain.UserPreferences, error) {
	var preferences domain.UserPreferences
	err := r.db.WithContext(ctx).Table("user_preferences").Where("user_id = ?", userID).Take(&preferences).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &preferences, nil
}

func (r *UserPreferencesRepository) Update(ctx context.Context, userID string, patch domain.UpdateUserPreferences) (*domain.UserPreferences, error) {
	var preferences domain.UserPreferences
	// One statement preserves omitted fields even when devices save concurrently.
	err := r.db.WithContext(ctx).Raw(`
 INSERT INTO user_preferences (user_id, theme, confetti_enabled, created_at, updated_at)
 VALUES (?, COALESCE(?, 'system'), COALESCE(?, TRUE), NOW(), NOW())
 ON CONFLICT (user_id) DO UPDATE SET
  theme = COALESCE(?, user_preferences.theme),
  confetti_enabled = COALESCE(?, user_preferences.confetti_enabled),
  updated_at = NOW()
 RETURNING user_id, theme, confetti_enabled, created_at, updated_at`,
		userID, patch.Theme, patch.ConfettiEnabled, patch.Theme, patch.ConfettiEnabled).Scan(&preferences).Error
	if err != nil {
		return nil, err
	}
	return &preferences, nil
}

var _ ports.UserPreferencesRepository = (*UserPreferencesRepository)(nil)
