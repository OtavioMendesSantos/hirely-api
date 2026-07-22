package services

import (
	"context"
	"errors"
	"testing"

	"hirely-api/internal/core/domain"
)

type mockAppRepo struct {
	apps map[string]*domain.Application
}

func newMockAppRepo() *mockAppRepo {
	return &mockAppRepo{apps: make(map[string]*domain.Application)}
}

func (m *mockAppRepo) Create(ctx context.Context, app *domain.Application) error {
	m.apps[app.ID] = app
	return nil
}

func (m *mockAppRepo) FindByID(ctx context.Context, id string) (*domain.Application, error) {
	app, ok := m.apps[id]
	if !ok {
		return nil, nil
	}
	return app, nil
}

func (m *mockAppRepo) ListByUserID(ctx context.Context, userID string, orderBy string, orderDir string) ([]*domain.Application, error) {
	var list []*domain.Application
	for _, app := range m.apps {
		if app.UserID == userID {
			list = append(list, app)
		}
	}
	return list, nil
}

func (m *mockAppRepo) ListByUserIDWithFilters(ctx context.Context, userID string, statuses []string, orderBy string, orderDir string) ([]*domain.Application, error) {
	statusMap := make(map[string]bool)
	for _, st := range statuses {
		statusMap[st] = true
	}
	var list []*domain.Application
	for _, app := range m.apps {
		if app.UserID == userID && statusMap[string(app.Status)] {
			list = append(list, app)
		}
	}
	return list, nil
}

func (m *mockAppRepo) Update(ctx context.Context, app *domain.Application) error {
	m.apps[app.ID] = app
	return nil
}

func (m *mockAppRepo) Delete(ctx context.Context, id string) error {
	delete(m.apps, id)
	return nil
}

func (m *mockAppRepo) UpdateStatus(ctx context.Context, app *domain.Application, event *domain.Event) error {
	m.apps[app.ID] = app
	return nil
}

type mockEventRepo struct {
	events map[string]*domain.Event
}

func newMockEventRepo() *mockEventRepo {
	return &mockEventRepo{events: make(map[string]*domain.Event)}
}

func (m *mockEventRepo) Create(ctx context.Context, event *domain.Event) error {
	m.events[event.ID] = event
	return nil
}

func (m *mockEventRepo) GetByApplicationID(ctx context.Context, applicationID string) ([]*domain.Event, error) {
	var list []*domain.Event
	for _, e := range m.events {
		if e.ApplicationID == applicationID {
			list = append(list, e)
		}
	}
	return list, nil
}

func TestApplicationService_CreateAndGet_UserIsolation(t *testing.T) {
	appRepo := newMockAppRepo()
	eventRepo := newMockEventRepo()
	service := NewApplicationService(appRepo, eventRepo)

	ctx := context.Background()

	created, err := service.CreateApplication(ctx, "user-1", CreateApplicationInput{
		CompanyName: "Google",
		JobTitle:    "Go Developer",
		Status:      domain.StatusApplied,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if created.UserID != "user-1" || created.Status != domain.StatusApplied {
		t.Errorf("unexpected created app: %+v", created)
	}

	// Success get by owner
	found, err := service.GetApplicationByID(ctx, "user-1", created.ID)
	if err != nil || found.ID != created.ID {
		t.Errorf("expected found app, got err=%v", err)
	}

	// Unauthorized get by different user
	_, err = service.GetApplicationByID(ctx, "user-2", created.ID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden when accessing another user app, got %v", err)
	}
}

func TestApplicationService_UpdateStatus_TriggersEvent(t *testing.T) {
	appRepo := newMockAppRepo()
	eventRepo := newMockEventRepo()
	service := NewApplicationService(appRepo, eventRepo)

	ctx := context.Background()
	created, _ := service.CreateApplication(ctx, "user-1", CreateApplicationInput{
		CompanyName: "Google",
		JobTitle:    "Backend Dev",
		Status:      domain.StatusToApply,
	})

	newStatus := domain.StatusInterview
	updated, err := service.UpdateApplication(ctx, "user-1", created.ID, UpdateApplicationInput{
		Status: &newStatus,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updated.Status != domain.StatusInterview {
		t.Errorf("expected status INTERVIEW, got %v", updated.Status)
	}
	if len(updated.Events) == 0 {
		t.Errorf("expected events to be added upon status transition")
	}
}

func TestApplicationService_DeleteAndManualEvent(t *testing.T) {
	appRepo := newMockAppRepo()
	eventRepo := newMockEventRepo()
	service := NewApplicationService(appRepo, eventRepo)

	ctx := context.Background()
	app, _ := service.CreateApplication(ctx, "user-1", CreateApplicationInput{
		CompanyName: "Apple",
		JobTitle:    "iOS Engineer",
	})

	event, err := service.AddManualEvent(ctx, "user-1", app.ID, "Had first screening call")
	if err != nil || event.Description != "Had first screening call" {
		t.Errorf("expected manual event created, got err=%v", err)
	}

	// Try adding event as another user
	_, err = service.AddManualEvent(ctx, "user-2", app.ID, "Attempted unauthorized note")
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden for manual event, got %v", err)
	}

	// Delete as another user
	err = service.DeleteApplication(ctx, "user-2", app.ID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("expected ErrForbidden on delete, got %v", err)
	}

	// Delete as owner
	err = service.DeleteApplication(ctx, "user-1", app.ID)
	if err != nil {
		t.Errorf("expected no error on valid delete, got %v", err)
	}
}

func TestApplicationService_ListApplicationsGroupedByStatus(t *testing.T) {
	appRepo := newMockAppRepo()
	eventRepo := newMockEventRepo()
	service := NewApplicationService(appRepo, eventRepo)

	ctx := context.Background()
	service.CreateApplication(ctx, "user-1", CreateApplicationInput{
		CompanyName: "Google",
		JobTitle:    "Dev",
		Status:      domain.StatusToApply,
	})
	service.CreateApplication(ctx, "user-1", CreateApplicationInput{
		CompanyName: "Meta",
		JobTitle:    "Dev",
		Status:      domain.StatusToApply,
	})
	service.CreateApplication(ctx, "user-1", CreateApplicationInput{
		CompanyName: "Apple",
		JobTitle:    "Dev",
		Status:      domain.StatusInterview,
	})

	grouped, err := service.ListApplicationsGroupedByStatus(ctx, "user-1", nil, "", "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(grouped) != 7 {
		t.Errorf("expected 7 status groups, got %d", len(grouped))
	}
	if len(grouped[domain.StatusToApply]) != 2 {
		t.Errorf("expected 2 TO_APPLY applications, got %d", len(grouped[domain.StatusToApply]))
	}
	if len(grouped[domain.StatusInterview]) != 1 {
		t.Errorf("expected 1 INTERVIEW application, got %d", len(grouped[domain.StatusInterview]))
	}
	if len(grouped[domain.StatusApplied]) != 0 {
		t.Errorf("expected 0 APPLIED applications, got %d", len(grouped[domain.StatusApplied]))
	}

	// With status filter
	filteredGrouped, err := service.ListApplicationsGroupedByStatus(ctx, "user-1", []string{"INTERVIEW"}, "job_title", "asc")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(filteredGrouped) != 1 {
		t.Errorf("expected 1 status group when filtered, got %d", len(filteredGrouped))
	}
	if len(filteredGrouped[domain.StatusInterview]) != 1 {
		t.Errorf("expected 1 INTERVIEW application in filter, got %d", len(filteredGrouped[domain.StatusInterview]))
	}

	// Test invalid sort option returns error
	_, err = service.ListApplications(ctx, "user-1", nil, "invalid_column", "asc")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput when ordering by invalid column, got %v", err)
	}
}
