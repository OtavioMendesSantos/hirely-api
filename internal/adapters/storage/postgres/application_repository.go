package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model := ApplicationFromDomain(app)
		if err := tx.Omit("Tags").Create(model).Error; err != nil {
			return err
		}
		if len(model.Tags) > 0 {
			return tx.Model(model).Association("Tags").Append(model.Tags)
		}
		return nil
	})
}

func (r *ApplicationRepository) FindByID(ctx context.Context, id string) (*domain.Application, error) {
	var model ApplicationModel
	result := r.db.WithContext(ctx).
		Preload("Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at asc")
		}).
		Preload("Tags").
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

func (r *ApplicationRepository) ListByUserID(ctx context.Context, userID string, search string, tagIDs []string, orderBy string, orderDir string) ([]*domain.Application, error) {
	var models []ApplicationModel
	query := r.db.WithContext(ctx).
		Preload("Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at asc")
		}).
		Preload("Tags").
		Where("user_id = ?", userID)

	if search != "" {
		query = query.Where("company_name ILIKE ? OR job_title ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if len(tagIDs) > 0 {
		query = query.Where("id IN (SELECT application_id FROM application_tags WHERE tag_id IN ?)", tagIDs)
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

func (r *ApplicationRepository) ListByUserIDWithFilters(ctx context.Context, userID string, search string, statuses []string, tagIDs []string, orderBy string, orderDir string) ([]*domain.Application, error) {
	var models []ApplicationModel
	query := r.db.WithContext(ctx).
		Preload("Events", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at asc")
		}).
		Preload("Tags").
		Where("user_id = ?", userID)

	if search != "" {
		query = query.Where("company_name ILIKE ? OR job_title ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}

	if len(tagIDs) > 0 {
		query = query.Where("id IN (SELECT application_id FROM application_tags WHERE tag_id IN ?)", tagIDs)
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
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model := ApplicationFromDomain(app)
		if err := tx.Omit("Events", "User", "Tags").Save(model).Error; err != nil {
			return err
		}
		return tx.Model(model).Association("Tags").Replace(model.Tags)
	})
}

func (r *ApplicationRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&ApplicationModel{})
	return result.Error
}

func (r *ApplicationRepository) UpdateStatus(ctx context.Context, app *domain.Application, event *domain.Event) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		appModel := ApplicationFromDomain(app)
		if err := tx.Omit("Events", "User", "Tags").Save(appModel).Error; err != nil {
			return err
		}
		if err := tx.Model(appModel).Association("Tags").Replace(appModel.Tags); err != nil {
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

func (r *ApplicationRepository) GetStatsByUserID(ctx context.Context, userID string, startDate, endDate *time.Time) (*domain.ApplicationStats, error) {
	stats := &domain.ApplicationStats{
		FunnelByStatus: map[string]int{
			string(domain.StatusToApply):   0,
			string(domain.StatusApplied):   0,
			string(domain.StatusInterview): 0,
			string(domain.StatusOffer):     0,
			string(domain.StatusAccepted):  0,
			string(domain.StatusRejected):  0,
			string(domain.StatusOther):     0,
		},
		TopTags: make([]domain.TagCountStats, 0),
	}

	query1 := r.db.WithContext(ctx).Model(&ApplicationModel{}).Where("user_id = ?", userID)
	if startDate != nil {
		query1 = query1.Where("created_at >= ?", startDate)
	}
	if endDate != nil {
		query1 = query1.Where("created_at <= ?", endDate)
	}

	type statusCount struct {
		Status string
		Count  int
	}
	var funnel []statusCount
	if err := query1.Select("status, count(id) as count").Group("status").Scan(&funnel).Error; err != nil {
		return nil, err
	}

	total := 0
	interviewOrBeyond := 0
	for _, f := range funnel {
		stats.FunnelByStatus[f.Status] = f.Count
		total += f.Count
		if f.Status == string(domain.StatusInterview) || f.Status == string(domain.StatusOffer) || f.Status == string(domain.StatusAccepted) {
			interviewOrBeyond += f.Count
		}
	}
	stats.TotalApplications = total
	stats.KPIs.Interviews.Count = interviewOrBeyond

	stats.KPIs.Rejections.Count = stats.FunnelByStatus[string(domain.StatusRejected)]

	var advancedRejectionsCount int64
	queryAdv := r.db.WithContext(ctx).Table("applications a").
		Joins("JOIN events e ON a.id = e.application_id").
		Where("a.user_id = ?", userID).
		Where("a.status = ?", domain.StatusRejected).
		Where("e.new_status IN (?) OR e.previous_status IN (?)", []string{string(domain.StatusInterview), string(domain.StatusOffer)}, []string{string(domain.StatusInterview), string(domain.StatusOffer)})

	if startDate != nil {
		queryAdv = queryAdv.Where("a.created_at >= ?", startDate)
	}
	if endDate != nil {
		queryAdv = queryAdv.Where("a.created_at <= ?", endDate)
	}
	if err := queryAdv.Distinct("a.id").Count(&advancedRejectionsCount).Error; err != nil {
		return nil, err
	}

	stats.KPIs.AdvancedRejections.Count = int(advancedRejectionsCount)
	stats.KPIs.DirectRejections.Count = stats.KPIs.Rejections.Count - stats.KPIs.AdvancedRejections.Count

	appliedOrBeyond := total - stats.FunnelByStatus[string(domain.StatusToApply)]
	if appliedOrBeyond > 0 {
		stats.KPIs.Interviews.Rate = float64(interviewOrBeyond) / float64(appliedOrBeyond)
		stats.KPIs.Rejections.Rate = float64(stats.KPIs.Rejections.Count) / float64(appliedOrBeyond)
		stats.KPIs.AdvancedRejections.Rate = float64(stats.KPIs.AdvancedRejections.Count) / float64(appliedOrBeyond)
		stats.KPIs.DirectRejections.Rate = float64(stats.KPIs.DirectRejections.Count) / float64(appliedOrBeyond)
	}

	var ghostedCount int64
	queryGhosted := r.db.WithContext(ctx).Model(&ApplicationModel{}).
		Where("user_id = ?", userID).
		Where("status = ?", domain.StatusApplied).
		Where("updated_at <= ?", time.Now().Add(-30*24*time.Hour))

	if startDate != nil {
		queryGhosted = queryGhosted.Where("created_at >= ?", startDate)
	}
	if endDate != nil {
		queryGhosted = queryGhosted.Where("created_at <= ?", endDate)
	}
	if err := queryGhosted.Count(&ghostedCount).Error; err != nil {
		return nil, err
	}
	stats.KPIs.Ghosting.Count = int(ghostedCount)
	if appliedOrBeyond > 0 {
		stats.KPIs.Ghosting.Rate = float64(stats.KPIs.Ghosting.Count) / float64(appliedOrBeyond)
	}

	query2 := r.db.WithContext(ctx).Table("tags t").
		Select("t.name as tag_name, count(at.application_id) as count").
		Joins("JOIN application_tags at ON t.id = at.tag_id").
		Joins("JOIN applications a ON a.id = at.application_id").
		Where("a.user_id = ?", userID)

	if startDate != nil {
		query2 = query2.Where("a.created_at >= ?", startDate)
	}
	if endDate != nil {
		query2 = query2.Where("a.created_at <= ?", endDate)
	}

	query2 = query2.Group("t.id, t.name").Order("count DESC").Limit(5)

	if err := query2.Scan(&stats.TopTags).Error; err != nil {
		return nil, err
	}

	var topJobTitles []domain.JobTitleStats
	rawQuery := `
		WITH cargos_normalizados AS (
			SELECT TRIM(LOWER(job_title)) AS cargo, COUNT(*) AS frequencia
			FROM applications
			WHERE user_id = ?
	`
	args := []any{userID}

	if startDate != nil {
		rawQuery += ` AND created_at >= ?`
		args = append(args, startDate)
	}
	if endDate != nil {
		rawQuery += ` AND created_at <= ?`
		args = append(args, endDate)
	}

	rawQuery += `
			GROUP BY TRIM(LOWER(job_title))
		),
		cargos_canonicos AS (
			SELECT 
				c1.cargo AS cargo_original,
				(
					SELECT c2.cargo 
					FROM cargos_normalizados c2 
					WHERE levenshtein(c1.cargo, c2.cargo) <= 2 
					ORDER BY c2.frequencia DESC 
					LIMIT 1
				) AS cargo_canonico,
				c1.frequencia
			FROM cargos_normalizados c1
		)
		SELECT 
			cargo_canonico AS job_title, 
			SUM(frequencia) AS count
		FROM cargos_canonicos
		GROUP BY cargo_canonico
		ORDER BY count DESC
		LIMIT 5;
	`

	if err := r.db.WithContext(ctx).Raw(rawQuery, args...).Scan(&topJobTitles).Error; err != nil {
		return nil, err
	}
	stats.TopJobTitles = topJobTitles

	return stats, nil
}

var _ ports.ApplicationRepository = (*ApplicationRepository)(nil)
