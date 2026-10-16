package postgres

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/ports"
	"time"
)

func (r *ApplicationRepository) ArchiveDueBatch(ctx context.Context, now time.Time, limit int) (int, error) {
	if limit < 1 || limit > 100 {
		return 0, domain.ErrInvalidInput
	}
	processed := 0
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var candidates []struct {
			ApplicationModel
			ArchiveDays int
		}
		err := tx.Table("applications AS a").Select("a.*, COALESCE(p.auto_archive_days, 60) AS archive_days").
			Joins("LEFT JOIN user_preferences p ON p.user_id = a.user_id").
			Where("a.archived_at IS NULL AND COALESCE(p.auto_archive_days, 60) > 0").
			Where("a.updated_at <= (?::timestamptz AT TIME ZONE 'UTC') - COALESCE(p.auto_archive_days, 60) * INTERVAL '1 day'", now).
			Order("a.updated_at ASC, a.id ASC").Limit(limit).
			Clauses(clause.Locking{Strength: "UPDATE", Table: clause.Table{Name: "a"}, Options: "SKIP LOCKED"}).Find(&candidates).Error
		if err != nil {
			return err
		}
		for i := range candidates {
			candidate := &candidates[i]
			if err := changeArchiveState(tx, &candidate.ApplicationModel, true, now, fmt.Sprintf("Automatically archived after %d days without activity.", candidate.ArchiveDays)); err != nil {
				return err
			}
			processed++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return processed, nil
}

var _ ports.ArchiveProcessor = (*ApplicationRepository)(nil)
