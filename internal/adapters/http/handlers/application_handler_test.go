package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"hirely-api/internal/adapters/http/dto"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"

	"github.com/gin-gonic/gin"
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

func (m *mockAppRepoForHandlerTest) ListByUserID(ctx context.Context, userID string, search string, tagIDs []string, orderBy string, orderDir string) ([]*domain.Application, error) {
	var list []*domain.Application
	for _, app := range m.apps {
		if app.UserID == userID {
			list = append(list, app)
		}
	}
	return list, nil
}

func (m *mockAppRepoForHandlerTest) ListByUserIDWithFilters(ctx context.Context, userID string, search string, statuses []string, tagIDs []string, orderBy string, orderDir string) ([]*domain.Application, error) {
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

func (m *mockAppRepoForHandlerTest) GetStatsByUserID(ctx context.Context, userID string, startDate, endDate *time.Time) (*domain.ApplicationStats, error) {
	return &domain.ApplicationStats{
		TotalApplications: 10,
		FunnelByStatus: map[string]int{
			"applied":   10,
			"interview": 5,
		},
		KPIs: domain.KPIs{
			Interviews: domain.KPIMetric{Count: 5, Rate: 0.5},
			Rejections: domain.KPIMetric{Count: 2, Rate: 0.2},
			Ghosting:   domain.KPIMetric{Count: 1, Rate: 0.1},
		},
		TopTags: []domain.TagCountStats{
			{TagName: "Backend", Count: 3},
		},
	}, nil
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

func setupAppRouter(appService *services.ApplicationService, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := NewApplicationHandler(appService)
	r := gin.New()

	r.Use(func(c *gin.Context) {
		c.Set("userID", userID)
	})

	r.POST("/v1/users/:user_id/applications", handler.Create)
	r.GET("/v1/users/:user_id/applications", handler.List)
	r.GET("/v1/users/:user_id/applications/grouped-by-status", handler.GroupedByStatus)
	r.GET("/v1/users/:user_id/applications/:application_id", handler.GetByID)
	r.PATCH("/v1/users/:user_id/applications/:application_id", handler.Update)
	r.DELETE("/v1/users/:user_id/applications/:application_id", handler.Delete)
	r.POST("/v1/users/:user_id/applications/:application_id/events", handler.AddEvent)
	r.GET("/v1/users/:user_id/applications/stats", handler.GetStats)

	return r
}

func TestApplicationHandler_CreateAndList_Success(t *testing.T) {
	appRepo := newMockAppRepoForHandlerTest()
	eventRepo := newMockEventRepoForHandlerTest()
	appService := services.NewApplicationService(appRepo, eventRepo, newMockTagRepo())
	r := setupAppRouter(appService, "user-123")

	payload := dto.CreateApplicationRequest{
		CompanyName: "Hirely Corp",
		JobTitle:    "Senior Backend Engineer",
		JobURL:      "https://linkedin.com/jobs/123",
		Status:      domain.StatusApplied,
		Location:    "Remote",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/users/user-123/applications", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

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
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)

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
	appService := services.NewApplicationService(appRepo, eventRepo, newMockTagRepo())
	r := setupAppRouter(appService, "user-999") // Authenticated as user-999

	req := httptest.NewRequest("GET", "/v1/users/user-123/applications", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403 Forbidden, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestApplicationHandler_GetUpdateDeleteAndEvent_Success(t *testing.T) {
	appRepo := newMockAppRepoForHandlerTest()
	eventRepo := newMockEventRepoForHandlerTest()
	appService := services.NewApplicationService(appRepo, eventRepo, newMockTagRepo())
	r := setupAppRouter(appService, "user-123")

	ctx := context.Background()
	app, _ := appService.CreateApplication(ctx, "user-123", services.CreateApplicationInput{
		CompanyName: "Amazon",
		JobTitle:    "Cloud Dev",
		Status:      domain.StatusToApply,
	})

	// GetByID
	getReq := httptest.NewRequest("GET", "/v1/users/user-123/applications/"+app.ID, nil)
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", getRec.Code)
	}

	// Update Status
	newStatus := domain.StatusInterview
	updatePayload := dto.UpdateApplicationRequest{Status: &newStatus}
	updateBody, _ := json.Marshal(updatePayload)

	updateReq := httptest.NewRequest("PATCH", "/v1/users/user-123/applications/"+app.ID, bytes.NewReader(updateBody))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	r.ServeHTTP(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on patch, got %d: %s", updateRec.Code, updateRec.Body.String())
	}

	// Add Manual Event
	eventPayload := dto.CreateManualEventRequest{Description: "Scheduled technical interview"}
	eventBody, _ := json.Marshal(eventPayload)

	eventReq := httptest.NewRequest("POST", "/v1/users/user-123/applications/"+app.ID+"/events", bytes.NewReader(eventBody))
	eventReq.Header.Set("Content-Type", "application/json")
	eventRec := httptest.NewRecorder()
	r.ServeHTTP(eventRec, eventReq)

	if eventRec.Code != http.StatusCreated {
		t.Fatalf("expected status 201 on add event, got %d: %s", eventRec.Code, eventRec.Body.String())
	}

	// Delete
	deleteReq := httptest.NewRequest("DELETE", "/v1/users/user-123/applications/"+app.ID, nil)
	deleteRec := httptest.NewRecorder()
	r.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 on delete, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
}

func TestApplicationHandler_GroupedByStatus_Success(t *testing.T) {
	appRepo := newMockAppRepoForHandlerTest()
	eventRepo := newMockEventRepoForHandlerTest()
	appService := services.NewApplicationService(appRepo, eventRepo, newMockTagRepo())
	r := setupAppRouter(appService, "user-123")

	ctx := context.Background()
	appService.CreateApplication(ctx, "user-123", services.CreateApplicationInput{
		CompanyName: "Netflix",
		JobTitle:    "DevOps",
		Status:      domain.StatusToApply,
	})
	appService.CreateApplication(ctx, "user-123", services.CreateApplicationInput{
		CompanyName: "Spotify",
		JobTitle:    "SRE",
		Status:      domain.StatusApplied,
	})

	req := httptest.NewRequest("GET", "/v1/users/user-123/applications/grouped-by-status", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp dto.GroupedApplicationsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal grouped response: %v", err)
	}

	if len(resp.GroupedApplications) != 7 {
		t.Errorf("expected 7 status entries, got %d", len(resp.GroupedApplications))
	}
	if len(resp.GroupedApplications[domain.StatusToApply]) != 1 {
		t.Errorf("expected 1 TO_APPLY app, got %d", len(resp.GroupedApplications[domain.StatusToApply]))
	}
	if len(resp.GroupedApplications[domain.StatusApplied]) != 1 {
		t.Errorf("expected 1 APPLIED app, got %d", len(resp.GroupedApplications[domain.StatusApplied]))
	}
}

func TestApplicationHandler_Ordering_Success(t *testing.T) {
	appRepo := newMockAppRepoForHandlerTest()
	eventRepo := newMockEventRepoForHandlerTest()
	appService := services.NewApplicationService(appRepo, eventRepo, newMockTagRepo())
	r := setupAppRouter(appService, "user-123")

	ctx := context.Background()
	appService.CreateApplication(ctx, "user-123", services.CreateApplicationInput{
		CompanyName: "Netflix",
		JobTitle:    "Backend",
		Status:      domain.StatusApplied,
	})

	// Test List with order_by
	req := httptest.NewRequest("GET", "/v1/users/user-123/applications?order_by=job_title&order=asc", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Test GroupedByStatus with order_by
	reqGrouped := httptest.NewRequest("GET", "/v1/users/user-123/applications/grouped-by-status?order_by=applied_at&order=desc", nil)
	recGrouped := httptest.NewRecorder()
	r.ServeHTTP(recGrouped, reqGrouped)

	if recGrouped.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recGrouped.Code, recGrouped.Body.String())
	}
}

func TestApplicationHandler_GetStats_Success(t *testing.T) {
	appRepo := newMockAppRepoForHandlerTest()
	eventRepo := newMockEventRepoForHandlerTest()
	appService := services.NewApplicationService(appRepo, eventRepo, newMockTagRepo())
	r := setupAppRouter(appService, "user-123")

	req := httptest.NewRequest("GET", "/v1/users/user-123/applications/stats?start_date=2024-01-01&end_date=2024-12-31", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp dto.ApplicationStatsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.TotalApplications != 10 {
		t.Errorf("expected 10 total applications, got %d", resp.TotalApplications)
	}
	if resp.KPIs.Interviews.Rate != 0.5 {
		t.Errorf("expected 0.5 conversion rate, got %f", resp.KPIs.Interviews.Rate)
	}
	if len(resp.TopTags) != 1 || resp.TopTags[0].TagName != "Backend" {
		t.Errorf("unexpected top tags: %+v", resp.TopTags)
	}
}
