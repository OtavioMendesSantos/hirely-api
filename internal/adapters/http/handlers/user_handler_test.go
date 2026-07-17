package handlers

import (
	"context"
	"encoding/json"
	"hirely-api/internal/adapters/http/dto"
	"hirely-api/internal/adapters/http/middleware"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockUserRepoForHandlerTest struct {
	users map[string]*domain.User
}

func newMockUserRepoForHandlerTest() *mockUserRepoForHandlerTest {
	return &mockUserRepoForHandlerTest{
		users: make(map[string]*domain.User),
	}
}

func (m *mockUserRepoForHandlerTest) Create(ctx context.Context, user *domain.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepoForHandlerTest) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepoForHandlerTest) FindByID(ctx context.Context, id string) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func TestUserHandler_GetMe_Success(t *testing.T) {
	repo := newMockUserRepoForHandlerTest()
	userService := services.NewUserService(repo)
	handler := NewUserHandler(userService)

	user := domain.NewUser("user-777", "Otavio Mendes", "otavio@hirely.app", "hash")
	_ = repo.Create(context.Background(), user)

	req := httptest.NewRequest("GET", "/v1/users/me", nil)
	ctx := middleware.WithUserID(req.Context(), "user-777")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.GetMe(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var respUser domain.User
	if err := json.Unmarshal(rec.Body.Bytes(), &respUser); err != nil {
		t.Fatalf("failed to unmarshal user response: %v", err)
	}

	if respUser.ID != "user-777" || respUser.Email != "otavio@hirely.app" {
		t.Errorf("expected otavio@hirely.app with id user-777, got %v", respUser)
	}
	if respUser.PasswordHash != "" {
		t.Errorf("password hash must not be returned in JSON response")
	}
}

func TestUserHandler_GetMe_NotFound(t *testing.T) {
	repo := newMockUserRepoForHandlerTest()
	userService := services.NewUserService(repo)
	handler := NewUserHandler(userService)

	req := httptest.NewRequest("GET", "/v1/users/me", nil)
	ctx := middleware.WithUserID(req.Context(), "non-existent")
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	handler.GetMe(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}

	var errResp dto.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Status != "NOT_FOUND" {
		t.Errorf("expected status NOT_FOUND, got %s", errResp.Error.Status)
	}
}

func TestUserHandler_GetMe_UnauthorizedWithoutContext(t *testing.T) {
	repo := newMockUserRepoForHandlerTest()
	userService := services.NewUserService(repo)
	handler := NewUserHandler(userService)

	req := httptest.NewRequest("GET", "/v1/users/me", nil)
	rec := httptest.NewRecorder()

	handler.GetMe(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}
