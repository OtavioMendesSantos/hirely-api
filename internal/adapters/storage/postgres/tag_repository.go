package postgres

import (
	"context"
	"errors"
	"strings"

	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/ports"

	"gorm.io/gorm"
)

type TagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) *TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) Create(ctx context.Context, tag *domain.Tag) error {
	model := TagFromDomain(tag)
	result := r.db.WithContext(ctx).Create(model)
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "duplicate key value") || strings.Contains(result.Error.Error(), "SQLSTATE 23505") {
			return domain.ErrTagAlreadyExists
		}
		return result.Error
	}
	return nil
}

func (r *TagRepository) FindByID(ctx context.Context, id string) (*domain.Tag, error) {
	var model TagModel
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&model)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return model.ToDomain(), nil
}

func (r *TagRepository) FindByIDs(ctx context.Context, ids []string) ([]*domain.Tag, error) {
	if len(ids) == 0 {
		return []*domain.Tag{}, nil
	}

	var models []TagModel
	result := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&models)

	if result.Error != nil {
		return nil, result.Error
	}

	tags := make([]*domain.Tag, len(models))
	for i, m := range models {
		tags[i] = m.ToDomain()
	}

	return tags, nil
}

func (r *TagRepository) ListByUserID(ctx context.Context, userID string) ([]*domain.Tag, error) {
	var models []TagModel
	result := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&models)

	if result.Error != nil {
		return nil, result.Error
	}

	tags := make([]*domain.Tag, len(models))
	for i, m := range models {
		tags[i] = m.ToDomain()
	}

	return tags, nil
}

func (r *TagRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM application_tags WHERE tag_id = ?", id).Error; err != nil {
			return err
		}
		result := tx.Where("id = ?", id).Delete(&TagModel{})
		return result.Error
	})
}

var _ ports.TagRepository = (*TagRepository)(nil)
