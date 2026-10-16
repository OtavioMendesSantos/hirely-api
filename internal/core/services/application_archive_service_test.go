package services

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"hirely-api/internal/core/domain"
	"testing"
	"time"
)

type pageRepo struct {
	*mockAppRepo
	results []*domain.Application
	last    domain.ApplicationQuery
}

func (r *pageRepo) Query(_ context.Context, _ string, input domain.ApplicationQuery) ([]*domain.Application, error) {
	r.last = input
	return r.results, nil
}

func TestArchiveLifecycleAndReadOnly(t *testing.T) {
	repo := newMockAppRepo()
	service := NewApplicationService(repo, newMockEventRepo(), nil)
	app := domain.NewApplication(uuid.NewString(), "owner", "Acme", "Engineer", domain.StatusAccepted)
	app.UpdatedAt = time.Now().Add(-90 * 24 * time.Hour)
	originalActivity := app.UpdatedAt
	repo.apps[app.ID] = app
	ctx := context.Background()
	archived, err := service.SetArchived(ctx, "owner", app.ID, true)
	if err != nil || archived.ArchivedAt == nil || archived.Status != domain.StatusAccepted || !archived.UpdatedAt.Equal(originalActivity) {
		t.Fatalf("archive: %+v %v", archived, err)
	}
	_, _ = service.SetArchived(ctx, "owner", app.ID, true)
	if len(app.Events) != 1 {
		t.Fatal("duplicate archive event")
	}
	if _, err := service.UpdateApplication(ctx, "owner", app.ID, UpdateApplicationInput{}); !errors.Is(err, domain.ErrApplicationArchived) {
		t.Fatalf("edit allowed: %v", err)
	}
	if _, err := service.AddManualEvent(ctx, "owner", app.ID, "note"); !errors.Is(err, domain.ErrApplicationArchived) {
		t.Fatalf("note allowed: %v", err)
	}
	if _, err := service.SetArchived(ctx, "someone-else", app.ID, false); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("cross-account restore: %v", err)
	}
	restored, err := service.SetArchived(ctx, "owner", app.ID, false)
	if err != nil || restored.ArchivedAt != nil || restored.Status != domain.StatusAccepted || !restored.UpdatedAt.After(originalActivity) {
		t.Fatalf("restore: %+v %v", restored, err)
	}
	_, _ = service.SetArchived(ctx, "owner", app.ID, false)
	if len(app.Events) != 2 || app.Events[1].Action != "UNARCHIVED" || app.Events[1].NewStatus != "ACCEPTED" {
		t.Fatal("invalid restore event")
	}
}

func TestArchivePageTokenAndQueryBinding(t *testing.T) {
	now := time.Now().UTC()
	rows := make([]*domain.Application, 3)
	for i := range rows {
		rows[i] = domain.NewApplication(uuid.NewString(), "owner", "Acme", "Engineer", domain.StatusRejected)
		rows[i].ArchivedAt = &now
	}
	repo := &pageRepo{mockAppRepo: newMockAppRepo(), results: rows}
	service := NewApplicationService(repo, nil, nil)
	query := domain.ApplicationQuery{ArchiveState: "archived", Limit: 2, Search: " Engineer ", Statuses: []string{"REJECTED", "REJECTED"}}
	page, err := service.ListApplicationsPage(context.Background(), "owner", query, "")
	if err != nil || len(page.Applications) != 2 || page.NextPageToken == "" {
		t.Fatalf("page: %+v %v", page, err)
	}
	if repo.last.OrderBy != "archived_at" || repo.last.Order != "desc" || len(repo.last.Statuses) != 1 {
		t.Fatalf("normalization: %+v", repo.last)
	}
	_, err = service.ListApplicationsPage(context.Background(), "owner", query, page.NextPageToken)
	if err != nil || repo.last.Cursor == nil || repo.last.Cursor.ID != rows[1].ID {
		t.Fatalf("next page: %+v %v", repo.last, err)
	}
	for _, change := range []struct {
		user  string
		query domain.ApplicationQuery
	}{
		{"other", query}, {"owner", domain.ApplicationQuery{ArchiveState: "archived", Limit: 2, Search: "Different"}},
		{"owner", domain.ApplicationQuery{ArchiveState: "archived", Limit: 2, Order: "asc"}},
	} {
		if _, err := service.ListApplicationsPage(context.Background(), change.user, change.query, page.NextPageToken); !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("accepted incompatible token: %v", err)
		}
	}
	if _, err := service.ListApplicationsPage(context.Background(), "owner", query, "not-a-token"); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal("malformed token accepted")
	}
}

func TestArchiveQueryDefaultsAndValidation(t *testing.T) {
	for _, invalid := range []domain.ApplicationQuery{{ArchiveState: "deleted"}, {Limit: 101}, {Limit: -1}, {OrderBy: "archived_at; DROP TABLE applications"}, {Order: "sideways"}, {Statuses: []string{"ARCHIVED"}}} {
		if _, err := normalizeApplicationQuery(invalid); !errors.Is(err, domain.ErrInvalidInput) {
			t.Fatalf("invalid query accepted: %+v", invalid)
		}
	}
	archived, _ := normalizeApplicationQuery(domain.ApplicationQuery{ArchiveState: "archived"})
	if archived.Limit != 20 || archived.OrderBy != "archived_at" {
		t.Fatalf("wrong defaults: %+v", archived)
	}
	legacy, _ := normalizeApplicationQuery(domain.ApplicationQuery{})
	if legacy.ArchiveState != "active" || legacy.Limit != 0 {
		t.Fatalf("legacy compatibility: %+v", legacy)
	}
}

func TestDeleteEventDoesNotReplayHistory(t *testing.T) {
	repo := newMockAppRepo()
	service := NewApplicationService(repo, nil, nil)
	app := domain.NewApplication(uuid.NewString(), "owner", "Acme", "Engineer", domain.StatusRejected)
	app.UpdatedAt = time.Now().Add(-100 * 24 * time.Hour)
	archived := time.Now().UTC()
	app.ArchivedAt = &archived
	app.Events = []domain.Event{*domain.NewAutomaticEvent("event-a", app.ID, "Status changed", "INTERVIEW", "REJECTED")}
	repo.apps[app.ID] = app
	if err := service.DeleteEvent(context.Background(), "other", app.ID, "event-a"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatal("other user could delete history")
	}
	if err := service.DeleteEvent(context.Background(), "owner", app.ID, "event-a"); err != nil {
		t.Fatal(err)
	}
	if app.Status != domain.StatusRejected || app.ArchivedAt == nil || len(app.Events) != 0 || !app.UpdatedAt.After(app.CreatedAt.Add(-time.Second)) {
		t.Fatal("deletion changed status/archive or failed to refresh activity")
	}
	if err := service.DeleteEvent(context.Background(), "owner", app.ID, "event-a"); !errors.Is(err, domain.ErrEventNotFound) {
		t.Fatalf("missing event: %v", err)
	}
	if err := service.DeleteEvent(context.Background(), "owner", app.ID, ""); !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal("missing ID accepted")
	}
}
