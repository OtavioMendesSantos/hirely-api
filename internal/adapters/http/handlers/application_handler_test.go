package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"hirely-api/internal/adapters/http/dto"
	"hirely-api/internal/adapters/http/middleware"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"
)

type mockAppRepoForHandlerTest struct {
	apps map[string]*domain.Application
}

func newMockAppRepoForHandlerTest() *mockAppRepoForHandlerTest {
	return &mockAppRepoForHandlerTest{apps: make(map[string]*domain.Application)}
}

func (m *mockAppRepoForHandlerTest) Create(ctx context.Context, app *domain.Application) error {
	m.apps[app.ID] = app
	return nil
}

func (m *mockAppRepoForHandlerTest) FindByID(ctx context.Context, id string) (*domain.Application, error) {
	app, ok := m.apps[id]
	if !ok {
		return nil, nil
	}
	return app, nil
}

func (m *mockAppRepoForHandlerTest) ListByUserID(ctx context.Context, userID string) ([]*domain.Application, error) {
	var list []*domain.Application
	for _, app := range m.apps {
		if app.UserID == userID {
			list = append(list, app)
		}
	}
	return list, nil
}

func (m *mockAppRepoForHandlerTest) ListByUserIDWithFilters(ctx context.Context, userID string, statuses []string) ([]*domain.Application, error) {
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

func (m *mockAppRepoForHandlerTest) Update(ctx context.Context, app *domain.Application) error {
	m.apps[app.ID] = app
	return nil
}

func (m *mockAppRepoForHandlerTest) Delete(ctx context.Context, id string) error {
	delete(m.apps, id)
	return nil
}

func (m *mockAppRepoForHandlerTest) UpdateStatus(ctx context.Context, app *domain.Application, event *domain.Event) error {
	m.apps[app.ID] = app
	return nil
}

type mockEventRepoForHandlerTest struct {
	events map[string]*domain.Event
}

func newMockEventRepoForHandlerTest() *mockEventRepoForHandlerTest {
	return &mockEventRepoForHandlerTest{events: make(map[string]*domain.Event)}
}

func (m *mockEventRepoForHandlerTest) Create(ctx context.Context, event *domain.Event) error {
	m.events[event.ID] = event
	return nil
}

func (m *mockEventRepoForHandlerTest) GetByApplicationID(ctx context.Context, applicationID string) ([]*domain.Event, error) {
	var list []*domain.Event
	for _, e := range m.events {
		if e.ApplicationID == applicationID {
			list = append(list, e)
		}
	}
	return list, nil
}

func TestApplicationHandler_CreateAndList_Success(t *testing.T) {
	appRepo := newMockAppRepoForHandlerTest()
	eventRepo := newMockEventRepoForHandlerTest()
	appService := services.NewApplicationService(appRepo, eventRepo)
	handler := NewApplicationHandler(appService)

	payload := dto.CreateApplicationRequest{
		CompanyName: "Hirely Corp",
		JobTitle:    "Senior Backend Engineer",
		JobURL:      "https://linkedin.com/jobs/123",
		Status:      domain.StatusApplied,
		Location:    "Remote",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/users/user-123/applications", bytes.NewReader(body))
	req = req.WithContext(middleware.WithUserID(req.Context(), "user-123"))
	req.SetPathValue("user_id", "user-123")

	rec := httptest.NewRecorder()
	handler.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var createdApp domain.Application
	if err := json.Unmarshal(rec.Body.Bytes(), &createdApp); err != nil {
		t.Fatalf("failed to unmarshal created app: %v", err)
	}
	if createdApp.CompanyName != "Hirely Corp" || createdApp.UserID != "user-123" {
		t.Errorf("unexpected created app: %+v", createdApp)
	}

	// Test List
	listReq := httptest.NewRequest("GET", "/v1/users/user-123/applications", nil)
	listReq = listReq.WithContext(middleware.WithUserID(listReq.Context(), "user-123"))
	listReq.SetPathValue("user_id", "user-123")

	listRec := httptest.NewRecorder()
	handler.List(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on list, got %d", listRec.Code)
	}

	var listResp dto.ListApplicationsResponse
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to unmarshal list resp: %v", err)
	}
	if len(listResp.Applications) != 1 {
		t.Errorf("expected 1 application in list, got %d", len(listResp.Applications))
	}
}

func TestApplicationHandler_UserIsolation_Forbidden(t *testing.T) {
	appRepo := newMockAppRepoForHandlerTest()
	eventRepo := newMockEventRepoForHandlerTest()
	appService := services.NewApplicationService(appRepo, eventRepo)
	handler := NewApplicationHandler(appService)

	req := httptest.NewRequest("GET", "/v1/users/user-123/applications", nil)
	// Authenticated as user-999 trying to access user-123 path
	req = req.WithContext(middleware.WithUserID(req.Context(), "user-999"))
	req.SetPathValue("user_id", "user-123")

	rec := httptest.NewRecorder()
	handler.List(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestApplicationHandler_GetUpdateDeleteAndEvent_Success(t *testing.T) {
	appRepo := newMockAppRepoForHandlerTest()
	eventRepo := newMockEventRepoForHandlerTest()
	appService := services.NewApplicationService(appRepo, eventRepo)
	handler := NewApplicationHandler(appService)

	ctx := context.Background()
	app, _ := appService.CreateApplication(ctx, "user-123", services.CreateApplicationInput{
		CompanyName: "Amazon",
		JobTitle:    "Cloud Dev",
		Status:      domain.StatusToApply,
	})

	// GetByID
	getReq := httptest.NewRequest("GET", "/v1/users/user-123/applications/"+app.ID, nil)
	getReq = getReq.WithContext(middleware.WithUserID(getReq.Context(), "user-123"))
	getReq.SetPathValue("user_id", "user-123")
	getReq.SetPathValue("application_id", app.ID)

	getRec := httptest.NewRecorder()
	handler.GetByID(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", getRec.Code)
	}

	// Update Status
	newStatus := domain.StatusInterview
	updatePayload := dto.UpdateApplicationRequest{Status: &newStatus}
	updateBody, _ := json.Marshal(updatePayload)

	updateReq := httptest.NewRequest("PATCH", "/v1/users/user-123/applications/"+app.ID, bytes.NewReader(updateBody))
	updateReq = updateReq.WithContext(middleware.WithUserID(updateReq.Context(), "user-123"))
	updateReq.SetPathValue("user_id", "user-123")
	updateReq.SetPathValue("application_id", app.ID)

	updateRec := httptest.NewRecorder()
	handler.Update(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on patch, got %d: %s", updateRec.Code, updateRec.Body.String())
	}

	// Add Manual Event
	eventPayload := dto.CreateManualEventRequest{Description: "Scheduled technical interview"}
	eventBody, _ := json.Marshal(eventPayload)

	eventReq := httptest.NewRequest("POST", "/v1/users/user-123/applications/"+app.ID+"/events", bytes.NewReader(eventBody))
	eventReq = eventReq.WithContext(middleware.WithUserID(eventReq.Context(), "user-123"))
	eventReq.SetPathValue("user_id", "user-123")
	eventReq.SetPathValue("application_id", app.ID)

	eventRec := httptest.NewRecorder()
	handler.AddEvent(eventRec, eventReq)

	if eventRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 on add event, got %d: %s", eventRec.Code, eventRec.Body.String())
	}

	// Delete
	deleteReq := httptest.NewRequest("DELETE", "/v1/users/user-123/applications/"+app.ID, nil)
	deleteReq = deleteReq.WithContext(middleware.WithUserID(deleteReq.Context(), "user-123"))
	deleteReq.SetPathValue("user_id", "user-123")
	deleteReq.SetPathValue("application_id", app.ID)

	deleteRec := httptest.NewRecorder()
	handler.Delete(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 on delete, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
}
