package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"hirely-api/internal/adapters/http/dto"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type mockUserRepo struct {
	users map[string]*domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users: make(map[string]*domain.User),
	}
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) error {
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (m *mockUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

func TestAuthHandler_Register_Success(t *testing.T) {
	repo := newMockUserRepo()
	authService := services.NewAuthService(repo, "secret", time.Hour)
	handler := NewAuthHandler(authService)

	payload := map[string]string{
		"name":     "John Doe",
		"email":    "john@example.com",
		"password": "password123",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/users", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp dto.AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Token == "" {
		t.Errorf("expected token in register response, got empty")
	}
	if resp.User == nil || resp.User.Email != "john@example.com" {
		t.Errorf("expected email john@example.com, got %v", resp.User)
	}
}

func TestAuthHandler_Register_ValidationFailure(t *testing.T) {
	repo := newMockUserRepo()
	authService := services.NewAuthService(repo, "secret", time.Hour)
	handler := NewAuthHandler(authService)

	payload := map[string]string{
		"name":     "J",
		"email":    "invalid-email",
		"password": "123",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/users", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	handler.Register(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}

	var errResp dto.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Status != "INVALID_ARGUMENT" {
		t.Errorf("expected status INVALID_ARGUMENT, got %s", errResp.Error.Status)
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	repo := newMockUserRepo()
	authService := services.NewAuthService(repo, "secret", time.Hour)
	handler := NewAuthHandler(authService)

	payload := map[string]string{
		"email":    "nonexistent@example.com",
		"password": "wrongpassword123",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/users:login", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", rec.Code, rec.Body.String())
	}

	var errResp dto.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if errResp.Error.Status != "UNAUTHENTICATED" {
		t.Errorf("expected status UNAUTHENTICATED, got %s", errResp.Error.Status)
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	repo := newMockUserRepo()
	authService := services.NewAuthService(repo, "secret", time.Hour)
	handler := NewAuthHandler(authService)

	_, _, err := authService.RegisterUser(context.Background(), "Alice Smith", "alice@example.com", "password123")
	if err != nil {
		t.Fatalf("failed to register user: %v", err)
	}

	payload := map[string]string{
		"email":    "alice@example.com",
		"password": "password123",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/v1/users:login", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp dto.AuthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Token == "" {
		t.Errorf("expected token in login response, got empty")
	}
	if resp.User == nil || resp.User.Email != "alice@example.com" {
		t.Errorf("expected user in login response with email alice@example.com, got %v", resp.User)
	}
}
