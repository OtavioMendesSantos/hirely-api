package postgres

import (
	"context"
	"time"
)

// Current interview/offer/accepted statuses remain evidence even if their events
// were deleted. EXISTS counts historical progress once per application, including
// archived, rejected, and reverted applications, without duplicating repeat visits.
const interviewCountsSQL = `
 SELECT
  COUNT(*) FILTER (WHERE ` + reachedInterviewPredicate + `) AS interview_count,
  COUNT(*) FILTER (WHERE ` + appliedHistoryPredicate + `) AS applied_count
 FROM applications a
 WHERE a.user_id = ?`

type interviewCounts struct {
	InterviewCount int64
	AppliedCount   int64
}

func (r *ApplicationRepository) interviewCohortCounts(ctx context.Context, userID string, startDate, endDate *time.Time) (interviewCounts, error) {
	query := interviewCountsSQL
	args := []any{userID}
	// Keep the existing metrics cohort: application creation date, not event date.
	if startDate != nil {
		query += " AND a.created_at >= ?"
		args = append(args, *startDate)
	}
	if endDate != nil {
		query += " AND a.created_at <= ?"
		args = append(args, *endDate)
	}
	var counts interviewCounts
	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&counts).Error
	return counts, err
}
