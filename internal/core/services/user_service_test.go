package services

import (
	"context"
	"errors"
	"hirely-api/internal/core/domain"
	"testing"
)

type mockUserRepoForUserTest struct {
	users map[string]*domain.User
}

func newMockUserRepoForUserTest() *mockUserRepoForUserTest {
	return &mockUserRepoForUserTest{
		users: make(map[string]*domain.User),
	}
}

func (m *mockUserRepoForUserTest) Create(ctx context.Context, user *domain.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepoForUserTest) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepoForUserTest) FindByID(ctx context.Context, id string) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func TestUserService_GetUserByID_Success(t *testing.T) {
	repo := newMockUserRepoForUserTest()
	service := NewUserService(repo)

	user := domain.NewUser("user-123", "John Doe", "john@example.com", "hash")
	_ = repo.Create(context.Background(), user)

	found, err := service.GetUserByID(context.Background(), "user-123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if found == nil || found.ID != "user-123" {
		t.Errorf("expected user-123, got %v", found)
	}
}

func TestUserService_GetUserByID_NotFound(t *testing.T) {
	repo := newMockUserRepoForUserTest()
	service := NewUserService(repo)

	_, err := service.GetUserByID(context.Background(), "non-existent")
	if !errors.Is(err, domain.ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestUserService_GetUserByID_InvalidInput(t *testing.T) {
	repo := newMockUserRepoForUserTest()
	service := NewUserService(repo)

	_, err := service.GetUserByID(context.Background(), "  ")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}
