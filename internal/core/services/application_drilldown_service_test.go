package services

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"hirely-api/internal/core/domain"
	"testing"
	"time"
)

type drilldownRepo struct {
	*pageRepo
	countInput domain.ApplicationQuery
	countError error
}

func (r *drilldownRepo) CountQuery(_ context.Context, _ string, input domain.ApplicationQuery) (int64, error) {
	r.countInput = input
	return 12, r.countError
}

func TestDrilldownCountAndCursorAreBoundToCohort(t *testing.T) {
	rows := make([]*domain.Application, 3)
	for i := range rows {
		rows[i] = domain.NewApplication(uuid.NewString(), "owner", "Acme", "Engineer", domain.StatusRejected)
	}
	repo := &drilldownRepo{pageRepo: &pageRepo{mockAppRepo: newMockAppRepo(), results: rows}}
	service := NewApplicationService(repo, nil, nil)
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(1, 0, 0)
	query := domain.ApplicationQuery{ArchiveState: "all", Limit: 2, Metric: "interviews", StartDate: &start, EndDate: &end}
	page, err := service.ListApplicationsPage(context.Background(), "owner", query, "")
	if err != nil || page.TotalCount != 12 || len(page.Applications) != 2 || page.NextPageToken == "" {
		t.Fatalf("page: %+v %v", page, err)
	}
	if repo.countInput.Cursor != nil || repo.countInput.Metric != "interviews" || repo.countInput.EvaluatedAt.IsZero() {
		t.Fatalf("count lost cohort: %+v", repo.countInput)
	}
	if _, err := service.ListApplicationsPage(context.Background(), "owner", query, page.NextPageToken); err != nil {
		t.Fatalf("same cohort cursor: %v", err)
	}
	if repo.last.Cursor == nil || repo.countInput.Cursor != nil || !repo.last.EvaluatedAt.Equal(repo.countInput.EvaluatedAt) {
		t.Fatal("count/page did not share evaluation time or cursor leaked into total")
	}
	changed := query
	changed.Metric = "rejections"
	if _, err := service.ListApplicationsPage(context.Background(), "owner", changed, page.NextPageToken); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal("cursor accepted different metric")
	}
	changed = query
	newStart := start.AddDate(0, 0, 1)
	changed.StartDate = &newStart
	if _, err := service.ListApplicationsPage(context.Background(), "owner", changed, page.NextPageToken); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal("cursor accepted different dates")
	}
	repo.countError = errors.New("count failed")
	if _, err := service.ListApplicationsPage(context.Background(), "owner", query, ""); !errors.Is(err, repo.countError) {
		t.Fatal("count failure hidden")
	}
}

func TestDrilldownRejectsInvalidMetricAndDateOrder(t *testing.T) {
	now := time.Now()
	before := now.Add(-time.Hour)
	for _, query := range []domain.ApplicationQuery{{Metric: "invalid"}, {StartDate: &now, EndDate: &before}} {
		if _, err := normalizeApplicationQuery(query); !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("invalid filter accepted: %+v", query)
		}
	}
}
