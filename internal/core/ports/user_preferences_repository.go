package ports

import (
	"context"
	"hirely-api/internal/core/domain"
)

type UserPreferencesRepository interface {
	FindByUserID(context.Context, string) (*domain.UserPreferences, error)
	Update(context.Context, string, domain.UpdateUserPreferences) (*domain.UserPreferences, error)
}
