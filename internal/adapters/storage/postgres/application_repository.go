package postgres

import (
	"context"
	"errors"
	"fmt"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/ports"

	"gorm.io/gorm"
)

type ApplicationRepository struct {
	db *gorm.DB
}

func NewApplicationRepository(db *gorm.DB) *ApplicationRepository {
	return &ApplicationRepository{db: db}
}

func getOrderClause(orderBy string, orderDir string) string {
	if orderBy == "" {
		orderBy = "created_at"
	}
	if orderDir == "" {
		orderDir = "desc"
	}
	clause := fmt.Sprintf("%s %s", orderBy, orderDir)
	if orderBy == "applied_at" {
		if orderDir == "desc" {
			clause += " nulls last"
		} else {
			clause += " nulls first"
		}
	}
	if orderBy != "created_at" {
		clause += ", created_at desc"
	}
	return clause
}

func (r *ApplicationRepository) Create(ctx context.Context, app *domain.Application) error {
	model := ApplicationFromDomain(app)
	result := r.db.WithContext(ctx).Create(model)
	return result.Error
}

func (r *ApplicationRepository) FindByID(ctx context.Context, id string) (*domain.Application, error) {
	var model ApplicationModel
	result := r.db.WithContext(ctx).
		Preload("Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at asc")
		}).
		Where("id = ?", id).
		First(&model)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return model.ToDomain(), nil
}

func (r *ApplicationRepository) ListByUserID(ctx context.Context, userID string, search string, orderBy string, orderDir string) ([]*domain.Application, error) {
	var models []ApplicationModel
	query := r.db.WithContext(ctx).
		Preload("Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at asc")
		}).
		Where("user_id = ?", userID)

	if search != "" {
		query = query.Where("company_name ILIKE ? OR job_title ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	result := query.Order(getOrderClause(orderBy, orderDir)).
		Find(&models)

	if result.Error != nil {
		return nil, result.Error
	}

	apps := make([]*domain.Application, len(models))
	for i, m := range models {
		apps[i] = m.ToDomain()
	}

	return apps, nil
}

func (r *ApplicationRepository) ListByUserIDWithFilters(ctx context.Context, userID string, search string, statuses []string, orderBy string, orderDir string) ([]*domain.Application, error) {
	var models []ApplicationModel
	query := r.db.WithContext(ctx).
		Preload("Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at asc")
		}).
		Where("user_id = ?", userID)

	if search != "" {
		query = query.Where("company_name ILIKE ? OR job_title ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}

	result := query.Order(getOrderClause(orderBy, orderDir)).Find(&models)
	if result.Error != nil {
		return nil, result.Error
	}

	apps := make([]*domain.Application, len(models))
	for i, m := range models {
		apps[i] = m.ToDomain()
	}

	return apps, nil
}

func (r *ApplicationRepository) Update(ctx context.Context, app *domain.Application) error {
	model := ApplicationFromDomain(app)
	result := r.db.WithContext(ctx).Omit("Events", "User").Save(model)
	return result.Error
}

func (r *ApplicationRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&ApplicationModel{})
	return result.Error
}

func (r *ApplicationRepository) UpdateStatus(ctx context.Context, app *domain.Application, event *domain.Event) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		appModel := ApplicationFromDomain(app)
		if err := tx.Omit("Events", "User").Save(appModel).Error; err != nil {
			return err
		}
		if event != nil {
			eventModel := EventFromDomain(event)
			if err := tx.Create(eventModel).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

var _ ports.ApplicationRepository = (*ApplicationRepository)(nil)
