package postgres

import (
	"context"
	"gorm.io/gorm"
	"hirely-api/internal/core/domain"
	"time"
)

func (r *ApplicationRepository) DeleteEvent(ctx context.Context, userID, applicationID, eventID string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Archived applications also allow removal of history. The parent row lock
		// prevents archiving/restoration or a note write from racing this activity.
		app, err := lockOwnedApplication(tx, userID, applicationID)
		if err != nil {
			return err
		}
		result := tx.Where("id = ? AND application_id = ?", eventID, applicationID).Delete(&EventModel{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrEventNotFound
		}
		return tx.Model(app).UpdateColumn("updated_at", gorm.Expr("GREATEST(updated_at, ?)", now.UTC())).Error
	})
}
