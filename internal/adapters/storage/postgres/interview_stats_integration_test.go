package postgres

import (
	"context"
	"github.com/google/uuid"
	"hirely-api/internal/core/domain"
	"math"
	"testing"
	"time"
)

func TestPostgresInterviewRateIncludesHistoricalProgressOnce(t *testing.T) {
	db := archiveTestDB(t)
	repo, base := archiveFixture(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Microsecond)
	base.CreatedAt, base.UpdatedAt = now, now
	// Use fresh rows so no creation event implies an interview that never happened.
	if err := repo.Delete(ctx, base.ID); err != nil {
		t.Fatal(err)
	}
	add := func(status domain.ApplicationStatus, transitions [][2]string, archived bool) *domain.Application {
		t.Helper()
		app := *base
		app.ID, app.Status = uuid.NewString(), status
		if archived {
			app.ArchivedAt = &now
		}
		if err := repo.Create(ctx, &app); err != nil {
			t.Fatal(err)
		}
		for _, transition := range transitions {
			event := domain.NewAutomaticEvent(uuid.NewString(), app.ID, "Status changed", transition[0], transition[1])
			if err := db.Create(EventFromDomain(event)).Error; err != nil {
				t.Fatal(err)
			}
		}
		return &app
	}
	rejected := add(domain.StatusRejected, [][2]string{{"APPLIED", "INTERVIEW"}, {"INTERVIEW", "OFFER"}, {"OFFER", "INTERVIEW"}, {"INTERVIEW", "REJECTED"}}, false)
	add(domain.StatusOther, [][2]string{{"INTERVIEW", "OTHER"}}, false)
	add(domain.StatusInterview, nil, false)
	add(domain.StatusOffer, nil, false)
	add(domain.StatusAccepted, nil, false)
	applied := add(domain.StatusApplied, nil, false)
	// A note mentioning interviews is not evidence of a status transition.
	if err := db.Create(EventFromDomain(domain.NewManualEvent(uuid.NewString(), applied.ID, "I want an interview"))).Error; err != nil {
		t.Fatal(err)
	}
	add(domain.StatusRejected, nil, false)
	add(domain.StatusToApply, nil, false)
	add(domain.StatusToApply, [][2]string{{"APPLIED", "INTERVIEW"}, {"INTERVIEW", "TO_APPLY"}}, false)
	add(domain.StatusToApply, [][2]string{{"APPLIED", "TO_APPLY"}}, false)
	add(domain.StatusRejected, [][2]string{{"INTERVIEW", "REJECTED"}}, true)
	_, other := archiveFixture(t, db)
	if err := db.Model(&ApplicationModel{}).Where("id = ?", other.ID).UpdateColumn("status", domain.StatusInterview).Error; err != nil {
		t.Fatal(err)
	}
	stats, err := repo.GetStatsByUserID(ctx, base.UserID, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Seven distinct interview participants out of ten ever-applied applications.
	if stats.KPIs.Interviews.Count != 7 || math.Abs(stats.KPIs.Interviews.Rate-0.7) > 0.000001 {
		t.Fatalf("historical conversion: %+v", stats.KPIs.Interviews)
	}
	if stats.TotalApplications != 11 || stats.FunnelByStatus["INTERVIEW"] != 1 {
		t.Fatalf("current-status funnel changed: %+v", stats)
	}
	// Removing all evidence changes the historical metric without changing status.
	if err := db.Where("application_id = ?", rejected.ID).Delete(&EventModel{}).Error; err != nil {
		t.Fatal(err)
	}
	updated, err := repo.GetStatsByUserID(ctx, base.UserID, nil, nil)
	if err != nil || updated.KPIs.Interviews.Count != 6 || math.Abs(updated.KPIs.Interviews.Rate-0.6) > 0.000001 {
		t.Fatalf("deletion not reflected: %+v %v", updated, err)
	}
}

func TestPostgresInterviewRateKeepsCreationDateCohort(t *testing.T) {
	db := archiveTestDB(t)
	repo, app := archiveFixture(t, db)
	now := time.Now().UTC().Truncate(time.Microsecond)
	ctx := context.Background()
	event := domain.NewAutomaticEvent(uuid.NewString(), app.ID, "Status changed", "INTERVIEW", "REJECTED")
	event.CreatedAt = now
	if err := db.Create(EventFromDomain(event)).Error; err != nil {
		t.Fatal(err)
	}
	start, end := now.Add(-24*time.Hour), now.Add(24*time.Hour)
	excluded, err := repo.GetStatsByUserID(ctx, app.UserID, &start, &end)
	if err != nil || excluded.KPIs.Interviews.Count != 0 || excluded.KPIs.Interviews.Rate != 0 {
		t.Fatalf("event date changed cohort: %+v %v", excluded, err)
	}
	start = app.CreatedAt.Add(-time.Second)
	end = app.CreatedAt.Add(time.Second)
	included, err := repo.GetStatsByUserID(ctx, app.UserID, &start, &end)
	if err != nil || included.KPIs.Interviews.Count != 1 || included.KPIs.Interviews.Rate != 1 {
		t.Fatalf("application cohort not respected: %+v %v", included, err)
	}
	if err := db.Where("application_id = ?", app.ID).Delete(&EventModel{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&ApplicationModel{}).Where("id = ?", app.ID).UpdateColumn("status", domain.StatusToApply).Error; err != nil {
		t.Fatal(err)
	}
	empty, err := repo.GetStatsByUserID(ctx, app.UserID, nil, nil)
	if err != nil || empty.KPIs.Interviews.Rate != 0 {
		t.Fatalf("empty denominator: %+v %v", empty, err)
	}
}
