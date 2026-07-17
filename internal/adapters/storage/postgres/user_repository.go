package postgres

import (
	"context"
	"errors"
	"hirely-api/internal/core/domain"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user in the database.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	model := FromDomain(user)

	result := r.db.WithContext(ctx).Create(model)
	return result.Error
}

// FindByEmail finds a user by email.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var model UserModel

	result := r.db.WithContext(ctx).Where("email = ?", email).First(&model)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return model.ToDomain(), nil
}

// FindByID finds a user by their unique ID.
func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var model UserModel

	result := r.db.WithContext(ctx).Where("id = ?", id).First(&model)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return model.ToDomain(), nil
}
