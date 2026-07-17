package services

import (
	"context"
	"errors"
	"hirely-api/internal/core/domain"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepositoryForAuthTest struct {
	users         map[string]*domain.User
	createErr     error
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

func (m *mockUserRepositoryForAuthTest) FindByID(ctx context.Context, id string) (*domain.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

func TestAuthService_RegisterUser_Success(t *testing.T) {
	repo := newMockUserRepositoryForAuthTest()
	service := NewAuthService(repo, "test-secret-key", 24*time.Hour)

	user, token, err := service.RegisterUser(context.Background(), "Otavio Mendes", "otavio@hirely.app", "password123")
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
	if token == "" {
		t.Error("expected non-empty JWT token")
	}

	// Verify password hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("password123")); err != nil {
		t.Errorf("saved password hash does not match plain password: %v", err)
	}

	// Verify JWT token claims
	parsedToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte("test-secret-key"), nil
	})
	if err != nil || !parsedToken.Valid {
		t.Fatalf("failed to verify generated JWT token: %v", err)
	}
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || claims["sub"] != user.ID || claims["email"] != user.Email {
		t.Errorf("unexpected JWT claims: %+v", claims)
	}
}

func TestAuthService_RegisterUser_InvalidInputs(t *testing.T) {
	repo := newMockUserRepositoryForAuthTest()
	service := NewAuthService(repo, "secret", time.Hour)

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
			user, token, err := service.RegisterUser(context.Background(), tc.userName, tc.email, tc.password)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got %v", err)
			}
			if user != nil || token != "" {
				t.Errorf("expected nil user and empty token when input is invalid")
			}
		})
	}
}

func TestAuthService_RegisterUser_EmailAlreadyExists(t *testing.T) {
	repo := newMockUserRepositoryForAuthTest()
	service := NewAuthService(repo, "secret", time.Hour)

	_, _, err := service.RegisterUser(context.Background(), "Otavio Mendes", "otavio@hirely.app", "password123")
	if err != nil {
		t.Fatalf("expected first registration to succeed, got %v", err)
	}

	user, token, err := service.RegisterUser(context.Background(), "Another Name", "otavio@hirely.app", "differentpass")
	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Errorf("expected ErrEmailAlreadyExists, got %v", err)
	}
	if user != nil || token != "" {
		t.Errorf("expected nil user and empty token on duplicate registration")
	}
}

func TestAuthService_RegisterUser_RepoError(t *testing.T) {
	repo := newMockUserRepositoryForAuthTest()
	service := NewAuthService(repo, "secret", time.Hour)

	repo.findByEmailErr = errors.New("database connection failed")
	_, _, err := service.RegisterUser(context.Background(), "Otavio Mendes", "otavio@hirely.app", "password123")
	if err == nil || err.Error() != "database connection failed" {
		t.Errorf("expected database connection failed error, got %v", err)
	}

	repo.findByEmailErr = nil
	repo.createErr = errors.New("insert failed")
	_, _, err = service.RegisterUser(context.Background(), "Otavio Mendes", "otavio@hirely.app", "password123")
	if err == nil || err.Error() != "insert failed" {
		t.Errorf("expected insert failed error, got %v", err)
	}
}

func TestAuthService_Login_Success(t *testing.T) {
	repo := newMockUserRepositoryForAuthTest()
	service := NewAuthService(repo, "secret", time.Hour)

	registeredUser, _, err := service.RegisterUser(context.Background(), "Otavio Mendes", "otavio@hirely.app", "password123")
	if err != nil {
		t.Fatalf("failed to setup test user: %v", err)
	}

	user, token, err := service.Login(context.Background(), "otavio@hirely.app", "password123")
	if err != nil {
		t.Fatalf("expected successful login, got %v", err)
	}
	if user == nil || user.ID != registeredUser.ID {
		t.Errorf("expected user %+v, got %+v", registeredUser, user)
	}
	if token == "" {
		t.Error("expected non-empty token on successful login")
	}
}

func TestAuthService_Login_InvalidInputs(t *testing.T) {
	repo := newMockUserRepositoryForAuthTest()
	service := NewAuthService(repo, "secret", time.Hour)

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
			_, _, err := service.Login(context.Background(), tc.email, tc.password)
			if !errors.Is(err, domain.ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	repo := newMockUserRepositoryForAuthTest()
	service := NewAuthService(repo, "secret", time.Hour)

	_, _, err := service.Login(context.Background(), "nonexistent@hirely.app", "password123")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	repo := newMockUserRepositoryForAuthTest()
	service := NewAuthService(repo, "secret", time.Hour)

	_, _, _ = service.RegisterUser(context.Background(), "Otavio Mendes", "otavio@hirely.app", "password123")

	_, _, err := service.Login(context.Background(), "otavio@hirely.app", "wrongpassword")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for wrong password, got %v", err)
	}
}

func TestAuthService_Login_RepoError(t *testing.T) {
	repo := newMockUserRepositoryForAuthTest()
	service := NewAuthService(repo, "secret", time.Hour)

	repo.findByEmailErr = errors.New("db error on login")
	_, _, err := service.Login(context.Background(), "otavio@hirely.app", "password123")
	if err == nil || err.Error() != "db error on login" {
		t.Errorf("expected db error on login, got %v", err)
	}
}
