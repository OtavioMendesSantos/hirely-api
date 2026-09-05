package services

import (
	"context"
	"errors"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/security"
	"testing"
	"time"
)

type mockUserRepositoryForAuthTest struct {
	users          map[string]*domain.User
	createErr      error
	findByEmailErr error
}

func newMockUserRepositoryForAuthTest() *mockUserRepositoryForAuthTest {
	return &mockUserRepositoryForAuthTest{
		users: make(map[string]*domain.User),
	}
}

func (m *mockUserRepositoryForAuthTest) Create(ctx context.Context, user *domain.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepositoryForAuthTest) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.findByEmailErr != nil {
		return nil, m.findByEmailErr
	}
	user, exists := m.users[email]
	if !exists {
		return nil, nil
	}
	return user, nil
}

func (m *mockUserRepositoryForAuthTest) Update(ctx context.Context, user *domain.User) error {
	m.users[user.Email] = user
	return nil
}

func (m *mockUserRepositoryForAuthTest) FindByID(ctx context.Context, id string) (*domain.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

type mockSessionRepository struct {
	sessions map[string]*domain.Session
}

func newMockSessionRepository() *mockSessionRepository {
	return &mockSessionRepository{
		sessions: make(map[string]*domain.Session),
	}
}
func (m *mockSessionRepository) Create(ctx context.Context, session *domain.Session) error {
	m.sessions[session.Hash] = session
	return nil
}
func (m *mockSessionRepository) FindByHash(ctx context.Context, hash string) (*domain.Session, error) {
	s, ok := m.sessions[hash]
	if !ok {
		return nil, nil
	}
	return s, nil
}
func (m *mockSessionRepository) UpdateExpiresAt(ctx context.Context, hash string, expiresAt time.Time) error {
	if s, ok := m.sessions[hash]; ok {
		s.ExpiresAt = expiresAt
	}
	return nil
}
func (m *mockSessionRepository) RevokeByHash(ctx context.Context, hash string) error {
	if s, ok := m.sessions[hash]; ok {
		s.Revoked = true
	}
	return nil
}

func TestAuthService_RegisterUser_Success(t *testing.T) {
	userRepo := newMockUserRepositoryForAuthTest()
	sessionRepo := newMockSessionRepository()
	service := NewAuthService(userRepo, sessionRepo, "test-pepper")

	user, err := service.RegisterUser(context.Background(), "Otavio Mendes", "otavio@hirely.app", "password123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.ID == "" {
		t.Error("expected non-empty user ID")
	}
	if user.Name != "Otavio Mendes" || user.Email != "otavio@hirely.app" {
		t.Errorf("unexpected user values: %+v", user)
	}

	if !security.VerifyPassword(user.PasswordHash, "password123", "test-pepper") {
		t.Errorf("saved password hash does not match plain password with pepper")
	}
}

func TestAuthService_RegisterUser_InvalidInputs(t *testing.T) {
	userRepo := newMockUserRepositoryForAuthTest()
	sessionRepo := newMockSessionRepository()
	service := NewAuthService(userRepo, sessionRepo, "test-pepper")

	testCases := []struct {
		name     string
		userName string
		email    string
		password string
	}{
		{"ShortName", "A", "otavio@hirely.app", "password123"},
		{"InvalidEmail", "Otavio Mendes", "not-an-email", "password123"},
		{"ShortPassword", "Otavio Mendes", "otavio@hirely.app", "123"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			user, err := service.RegisterUser(context.Background(), tc.userName, tc.email, tc.password)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got %v", err)
			}
			if user != nil {
				t.Errorf("expected nil user when input is invalid")
			}
		})
	}
}

func TestAuthService_RegisterUser_EmailAlreadyExists(t *testing.T) {
	userRepo := newMockUserRepositoryForAuthTest()
	sessionRepo := newMockSessionRepository()
	service := NewAuthService(userRepo, sessionRepo, "test-pepper")

	_, err := service.RegisterUser(context.Background(), "Otavio Mendes", "otavio@hirely.app", "password123")
	if err != nil {
		t.Fatalf("expected first registration to succeed, got %v", err)
	}

	user, err := service.RegisterUser(context.Background(), "Another Name", "otavio@hirely.app", "differentpass")
	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Errorf("expected ErrEmailAlreadyExists, got %v", err)
	}
	if user != nil {
		t.Errorf("expected nil user on duplicate registration")
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	userRepo := newMockUserRepositoryForAuthTest()
	sessionRepo := newMockSessionRepository()
	service := NewAuthService(userRepo, sessionRepo, "test-pepper")

	registeredUser, err := service.RegisterUser(context.Background(), "Otavio Mendes", "otavio@hirely.app", "password123")
	if err != nil {
		t.Fatalf("failed to setup test user: %v", err)
	}

	user, err := service.Login(context.Background(), "otavio@hirely.app", "password123")
	if err != nil {
		t.Fatalf("expected successful login, got %v", err)
	}
	if user == nil || user.ID != registeredUser.ID {
		t.Errorf("expected user %+v, got %+v", registeredUser, user)
	}
}

func TestAuthService_Login_InvalidInputs(t *testing.T) {
	userRepo := newMockUserRepositoryForAuthTest()
	sessionRepo := newMockSessionRepository()
	service := NewAuthService(userRepo, sessionRepo, "test-pepper")

	testCases := []struct {
		name     string
		email    string
		password string
	}{
		{"EmptyEmail", "", "password123"},
		{"EmptyPassword", "otavio@hirely.app", ""},
		{"BothEmpty", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.Login(context.Background(), tc.email, tc.password)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	userRepo := newMockUserRepositoryForAuthTest()
	sessionRepo := newMockSessionRepository()
	service := NewAuthService(userRepo, sessionRepo, "test-pepper")

	_, err := service.Login(context.Background(), "nonexistent@hirely.app", "password123")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	userRepo := newMockUserRepositoryForAuthTest()
	sessionRepo := newMockSessionRepository()
	service := NewAuthService(userRepo, sessionRepo, "test-pepper")

	_, _ = service.RegisterUser(context.Background(), "Otavio Mendes", "otavio@hirely.app", "password123")

	_, err := service.Login(context.Background(), "otavio@hirely.app", "wrongpassword")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for wrong password, got %v", err)
	}
}
func (m *mockSessionRepository) RevokeAllByUserID(ctx context.Context, userID string) error {
	for _, s := range m.sessions {
		if s.UserID == userID {
			s.Revoked = true
		}
	}
	return nil
}
