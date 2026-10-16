package handlers

import (
	"context"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type drilldownHandlerRepo struct {
	*mockAppRepoForHandlerTest
	last domain.ApplicationQuery
}

func (r *drilldownHandlerRepo) Query(_ context.Context, _ string, input domain.ApplicationQuery) ([]*domain.Application, error) {
	r.last = input
	return []*domain.Application{}, nil
}
func (r *drilldownHandlerRepo) CountQuery(_ context.Context, _ string, _ domain.ApplicationQuery) (int64, error) {
	return 42, nil
}

func TestListDrilldownDatesAndValidation(t *testing.T) {
	repo := &drilldownHandlerRepo{mockAppRepoForHandlerTest: newMockAppRepoForHandlerTest()}
	router := setupAppRouter(services.NewApplicationService(repo, nil, nil), "owner")
	for _, query := range []string{"start_date=bad", "end_date=2026-02-30", "start_date=2026-10-02&end_date=2026-10-01", "metric=unknown"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/users/owner/applications?"+query, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s: %d %s", query, response.Code, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/users/owner/applications?archive_state=all&page_size=20&metric=interviews&status=REJECTED&start_date=2026-10-01&end_date=2026-10-16", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("%d %s", response.Code, response.Body.String())
	}
	if repo.last.Metric != "interviews" || repo.last.ArchiveState != "all" || len(repo.last.Statuses) != 1 {
		t.Fatalf("filters lost: %+v", repo.last)
	}
	if repo.last.StartDate == nil || repo.last.EndDate == nil || repo.last.StartDate.Format(time.RFC3339) != "2026-10-01T00:00:00Z" || repo.last.EndDate.Format(time.RFC3339Nano) != "2026-10-16T23:59:59.999999999Z" {
		t.Fatalf("incorrect UTC date range: %+v", repo.last)
	}
}
