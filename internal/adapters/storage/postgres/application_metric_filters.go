package postgres

import (
	"gorm.io/gorm"
	"time"
)

// Shared predicates keep KPI totals and drill-down lists in the same cohort.
// EXISTS prevents repeated status changes from duplicating applications.
const reachedInterviewPredicate = `(a.status IN ('INTERVIEW', 'OFFER', 'ACCEPTED') OR EXISTS (
 SELECT 1 FROM events e WHERE e.application_id = a.id AND (
 e.previous_status IN ('INTERVIEW', 'OFFER', 'ACCEPTED') OR
 e.new_status IN ('INTERVIEW', 'OFFER', 'ACCEPTED'))))`

const appliedHistoryPredicate = `(a.status <> 'TO_APPLY' OR EXISTS (
 SELECT 1 FROM events e WHERE e.application_id = a.id AND (
 e.previous_status IN ('APPLIED', 'INTERVIEW', 'OFFER', 'ACCEPTED', 'REJECTED', 'OTHER') OR
 e.new_status IN ('APPLIED', 'INTERVIEW', 'OFFER', 'ACCEPTED', 'REJECTED', 'OTHER'))))`

const advancedRejectionEvidencePredicate = `EXISTS (
 SELECT 1 FROM events e WHERE e.application_id = a.id AND (
 e.previous_status IN ('INTERVIEW', 'OFFER') OR e.new_status IN ('INTERVIEW', 'OFFER')))`

const ghostingPredicate = `a.status = 'APPLIED' AND a.updated_at <= ?`

func applyApplicationMetric(query *gorm.DB, metric string, now time.Time) *gorm.DB {
	switch metric {
	case "":
		return query
	case "interviews":
		return query.Where(reachedInterviewPredicate)
	case "rejections":
		return query.Where("a.status = 'REJECTED'")
	case "advanced_rejections":
		return query.Where("a.status = 'REJECTED'").Where(advancedRejectionEvidencePredicate)
	case "direct_rejections":
		return query.Where("a.status = 'REJECTED'").Where("NOT " + advancedRejectionEvidencePredicate)
	case "ghosting":
		if now.IsZero() {
			now = time.Now().UTC()
		}
		return query.Where(ghostingPredicate, now.Add(-30*24*time.Hour))
	default:
		return query.Where("FALSE")
	}
}
