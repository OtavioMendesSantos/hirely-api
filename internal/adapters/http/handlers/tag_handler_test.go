package handlers

import (
	"bytes"
	"encoding/json"
	"hirely-api/internal/adapters/http/dto"
	"hirely-api/internal/adapters/http/middleware"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"
	"net/http"
	"net/http/httptest"
	"testing"
	"context"
)

type mockTagRepo struct {
	tags map[string]*domain.Tag
}

func newMockTagRepo() *mockTagRepo {
	return &mockTagRepo{
		tags: make(map[string]*domain.Tag),
	}
}

func (m *mockTagRepo) Create(ctx context.Context, tag *domain.Tag) error {
	m.tags[tag.ID] = tag
	return nil
}

func (m *mockTagRepo) FindByID(ctx context.Context, id string) (*domain.Tag, error) {
	if t, ok := m.tags[id]; ok {
		return t, nil
	}
	return nil, nil
}

func (m *mockTagRepo) FindByIDs(ctx context.Context, ids []string) ([]*domain.Tag, error) {
	var res []*domain.Tag
	for _, id := range ids {
		if t, ok := m.tags[id]; ok {
			res = append(res, t)
		}
	}
	return res, nil
}

func (m *mockTagRepo) ListByUserID(ctx context.Context, userID string) ([]*domain.Tag, error) {
	var res []*domain.Tag
	for _, t := range m.tags {
		if t.UserID == userID {
			res = append(res, t)
		}
	}
	return res, nil
}

func (m *mockTagRepo) Delete(ctx context.Context, id string) error {
	delete(m.tags, id)
	return nil
}

func setupTagRouter(repo *mockTagRepo) *http.ServeMux {
	svc := services.NewTagService(repo)
	handler := NewTagHandler(svc)
	mux := http.NewServeMux()
	
	mux.HandleFunc("POST /users/{user_id}/tags", handler.Create)
	mux.HandleFunc("GET /users/{user_id}/tags", handler.List)
	mux.HandleFunc("DELETE /users/{user_id}/tags/{tag_id}", handler.Delete)
	
	return mux
}

func TestTagHandler_Create_Success(t *testing.T) {
	repo := newMockTagRepo()
	mux := setupTagRouter(repo)

	payload := map[string]string{
		"name":      "Backend",
		"color_hex": "#FFFFFF",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/users/user-123/tags", bytes.NewBuffer(body))
	req = req.WithContext(middleware.WithUserID(req.Context(), "user-123"))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	var tag domain.Tag
	if err := json.Unmarshal(rec.Body.Bytes(), &tag); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if tag.Name != "Backend" || tag.UserID != "user-123" {
		t.Errorf("unexpected tag data: %+v", tag)
	}
}

func TestTagHandler_Create_MismatchUser(t *testing.T) {
	repo := newMockTagRepo()
	mux := setupTagRouter(repo)

	req := httptest.NewRequest("POST", "/users/user-123/tags", bytes.NewBuffer([]byte(`{}`)))
	req = req.WithContext(middleware.WithUserID(req.Context(), "other-user"))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}
}

func TestTagHandler_List_Success(t *testing.T) {
	repo := newMockTagRepo()
	repo.tags["tag-1"] = domain.NewTag("tag-1", "user-123", "Tag1", "#111111")
	repo.tags["tag-2"] = domain.NewTag("tag-2", "user-123", "Tag2", "#222222")
	repo.tags["tag-3"] = domain.NewTag("tag-3", "other-user", "Tag3", "#333333")
	
	mux := setupTagRouter(repo)

	req := httptest.NewRequest("GET", "/users/user-123/tags", nil)
	req = req.WithContext(middleware.WithUserID(req.Context(), "user-123"))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp dto.ListTagsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(resp.Tags))
	}
}

func TestTagHandler_Delete_Success(t *testing.T) {
	repo := newMockTagRepo()
	repo.tags["tag-1"] = domain.NewTag("tag-1", "user-123", "Tag1", "#111111")
	
	mux := setupTagRouter(repo)

	req := httptest.NewRequest("DELETE", "/users/user-123/tags/tag-1", nil)
	req = req.WithContext(middleware.WithUserID(req.Context(), "user-123"))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}

	if _, exists := repo.tags["tag-1"]; exists {
		t.Errorf("expected tag to be deleted")
	}
}

func TestTagHandler_Delete_Forbidden(t *testing.T) {
	repo := newMockTagRepo()
	// Tag belongs to other-user
	tag := domain.NewTag("tag-1", "other-user", "Tag1", "#111111")
	tag.ID = "tag-1"
	repo.tags["tag-1"] = tag
	
	mux := setupTagRouter(repo)

	req := httptest.NewRequest("DELETE", "/users/user-123/tags/tag-1", nil)
	req = req.WithContext(middleware.WithUserID(req.Context(), "user-123"))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}
}
