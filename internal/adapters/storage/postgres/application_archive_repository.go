package postgres

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"hirely-api/internal/core/domain"
	"time"
)

func (r *ApplicationRepository) Query(ctx context.Context, userID string, input domain.ApplicationQuery) ([]*domain.Application, error) {
	query := r.db.WithContext(ctx).Model(&ApplicationModel{}).Where("user_id = ?", userID)
	switch input.ArchiveState {
	case "archived":
		query = query.Where("archived_at IS NOT NULL")
	case "all":
	default:
		query = query.Where("archived_at IS NULL")
	}
	if input.Search != "" {
		query = query.Where("(company_name ILIKE ? OR job_title ILIKE ?)", "%"+input.Search+"%", "%"+input.Search+"%")
	}
	if len(input.Statuses) > 0 {
		query = query.Where("status IN ?", input.Statuses)
	}
	if len(input.TagIDs) > 0 {
		query = query.Where("id IN (SELECT application_id FROM application_tags WHERE tag_id IN ?)", input.TagIDs)
	}
	// Column/direction are normalized again at the storage boundary before interpolation.
	column := input.OrderBy
	switch column {
	case "created_at", "updated_at", "applied_at", "job_title", "archived_at":
	default:
		return nil, domain.ErrInvalidInput
	}
	direction, comparison := "DESC", "<"
	if input.Order == "asc" {
		direction, comparison = "ASC", ">"
	} else if input.Order != "desc" {
		return nil, domain.ErrInvalidInput
	}
	nullable := column == "applied_at" || column == "archived_at"
	nullsFirst := column == "applied_at" && input.Order == "asc"
	if cursor := input.Cursor; cursor != nil {
		if cursor.Value == nil {
			if !nullable {
				return nil, domain.ErrInvalidInput
			}
			predicate := "(" + column + " IS NULL AND id " + comparison + " ?)"
			if nullsFirst {
				predicate = "(" + predicate + " OR " + column + " IS NOT NULL)"
			}
			query = query.Where(predicate, cursor.ID)
		} else {
			predicate := "(" + column + " " + comparison + " ? OR (" + column + " = ? AND id " + comparison + " ?))"
			if nullable && !nullsFirst {
				predicate = "(" + predicate + " OR " + column + " IS NULL)"
			}
			query = query.Where(predicate, *cursor.Value, *cursor.Value, cursor.ID)
		}
	}
	order := column + " " + direction
	if nullable {
		if nullsFirst {
			order += " NULLS FIRST"
		} else {
			order += " NULLS LAST"
		}
	}
	query = query.Order(order).Order("id " + direction)
	if input.Limit > 0 {
		query = query.Limit(input.Limit + 1)
	}
	var models []ApplicationModel
	if err := query.Preload("Events", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC, id ASC") }).Preload("Tags").Find(&models).Error; err != nil {
		return nil, err
	}
	applications := make([]*domain.Application, len(models))
	for i := range models {
		applications[i] = models[i].ToDomain()
	}
	return applications, nil
}

func (r *ApplicationRepository) SetArchived(ctx context.Context, userID, applicationID string, archived bool, now time.Time) (*domain.Application, error) {
	var result *domain.Application
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model, err := lockOwnedApplication(tx, userID, applicationID)
		if err != nil {
			return err
		}
		if (model.ArchivedAt != nil) != archived {
			if err := changeArchiveState(tx, model, archived, now, ""); err != nil {
				return err
			}
		}
		var loaded ApplicationModel
		if err := tx.Preload("Events", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC, id ASC") }).Preload("Tags").First(&loaded, "id = ?", model.ID).Error; err != nil {
			return err
		}
		result = loaded.ToDomain()
		return nil
	})
	return result, err
}

func changeArchiveState(tx *gorm.DB, model *ApplicationModel, archived bool, now time.Time, reason string) error {
	action, description := "ARCHIVED", "Application archived."
	changes := map[string]any{"archived_at": now}
	if !archived {
		action, description = "UNARCHIVED", "Application restored to "+string(model.Status)+"."
		changes["archived_at"] = nil
		changes["updated_at"] = now
	}
	if reason != "" {
		description = reason
	}
	// UpdateColumns intentionally keeps the previous activity date when archiving.
	if err := tx.Model(model).UpdateColumns(changes).Error; err != nil {
		return err
	}
	event := domain.NewAutomaticEvent(uuid.NewString(), model.ID, description, "", "")
	event.Action, event.CreatedAt = action, now
	if !archived {
		event.NewStatus = string(model.Status)
	}
	return tx.Create(EventFromDomain(event)).Error
}

func (r *ApplicationRepository) AddEvent(ctx context.Context, userID, applicationID string, event *domain.Event) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model, err := lockActiveApplication(tx, userID, applicationID)
		if err != nil {
			return err
		}
		if event.ApplicationID != applicationID {
			return domain.ErrInvalidInput
		}
		if err := tx.Create(EventFromDomain(event)).Error; err != nil {
			return err
		}
		return tx.Model(model).UpdateColumn("updated_at", event.CreatedAt).Error
	})
}
