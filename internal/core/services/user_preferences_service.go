package services

import (
	"context"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/ports"
)

type UserPreferencesService struct {
	repo ports.UserPreferencesRepository
}

func NewUserPreferencesService(repo ports.UserPreferencesRepository) *UserPreferencesService {
	return &UserPreferencesService{repo: repo}
}

func (s *UserPreferencesService) Get(ctx context.Context, userID string) (*domain.UserPreferences, error) {
	if userID == "" {
		return nil, domain.ErrInvalidInput
	}
	preferences, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if preferences == nil {
		return domain.DefaultUserPreferences(userID), nil
	}
	return preferences, nil
}

func (s *UserPreferencesService) Update(ctx context.Context, userID string, patch domain.UpdateUserPreferences) (*domain.UserPreferences, error) {
	if userID == "" {
		return nil, domain.ErrInvalidInput
	}
	if patch.Theme != nil && *patch.Theme != "system" && *patch.Theme != "dark" && *patch.Theme != "light" {
		return nil, domain.ErrInvalidInput
	}
	if patch.AutoArchiveDays != nil {
		switch *patch.AutoArchiveDays {
		case 0, 30, 60, 90:
		default:
			return nil, domain.ErrInvalidInput
		}
	}
	if patch.Theme == nil && patch.ConfettiEnabled == nil && patch.AutoArchiveDays == nil {
		return s.Get(ctx, userID)
	}
	return s.repo.Update(ctx, userID, patch)
}
