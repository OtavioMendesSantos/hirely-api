package services

import (
	"context"
	"errors"
	"hirely-api/internal/core/domain"
	"testing"
)

type preferencesRepoStub struct {
	rows   map[string]*domain.UserPreferences
	err    error
	writes int
}

func (r *preferencesRepoStub) FindByUserID(_ context.Context, id string) (*domain.UserPreferences, error) {
	return r.rows[id], r.err
}
func (r *preferencesRepoStub) Update(_ context.Context, id string, patch domain.UpdateUserPreferences) (*domain.UserPreferences, error) {
	r.writes++
	if r.err != nil {
		return nil, r.err
	}
	value := domain.DefaultUserPreferences(id)
	if existing := r.rows[id]; existing != nil {
		*value = *existing
	}
	if patch.Theme != nil {
		value.Theme = *patch.Theme
	}
	if patch.ConfettiEnabled != nil {
		value.ConfettiEnabled = *patch.ConfettiEnabled
	}
	r.rows[id] = value
	return value, nil
}

func TestPreferencesDefaultsAndPartialUpdates(t *testing.T) {
	repo := &preferencesRepoStub{rows: map[string]*domain.UserPreferences{}}
	service := NewUserPreferencesService(repo)
	ctx := context.Background()
	value, err := service.Get(ctx, "user-a")
	if err != nil || value.Theme != "system" || !value.ConfettiEnabled || repo.writes != 0 {
		t.Fatalf("unexpected defaults: %+v, %v", value, err)
	}
	dark, disabled := "dark", false
	value, err = service.Update(ctx, "user-a", domain.UpdateUserPreferences{ConfettiEnabled: &disabled})
	if err != nil || value.ConfettiEnabled || value.Theme != "system" {
		t.Fatalf("false was not saved: %+v, %v", value, err)
	}
	value, err = service.Update(ctx, "user-a", domain.UpdateUserPreferences{Theme: &dark})
	if err != nil || value.Theme != "dark" || value.ConfettiEnabled {
		t.Fatalf("partial update lost an omitted field: %+v, %v", value, err)
	}
	other, err := service.Get(ctx, "user-b")
	if err != nil || other.UserID != "user-b" || other.Theme != "system" || !other.ConfettiEnabled {
		t.Fatalf("preferences leaked across users: %+v, %v", other, err)
	}
	_, err = service.Update(ctx, "user-a", domain.UpdateUserPreferences{})
	if err != nil || repo.writes != 2 {
		t.Fatal("empty patch should only read")
	}
}

func TestPreferencesRejectInvalidThemeAndMissingUser(t *testing.T) {
	repo := &preferencesRepoStub{rows: map[string]*domain.UserPreferences{}}
	service := NewUserPreferencesService(repo)
	invalid := "sepia"
	_, err := service.Update(context.Background(), "user-a", domain.UpdateUserPreferences{Theme: &invalid})
	if !errors.Is(err, domain.ErrInvalidInput) || repo.writes != 0 {
		t.Fatal("invalid theme was accepted")
	}
	_, err = service.Get(context.Background(), "")
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal("missing user was accepted")
	}
	_, err = service.Update(context.Background(), "", domain.UpdateUserPreferences{})
	if !errors.Is(err, domain.ErrInvalidInput) {
		t.Fatal("missing user update was accepted")
	}
}

func TestPreferencesPropagateRepositoryErrors(t *testing.T) {
	repo := &preferencesRepoStub{err: errors.New("database unavailable")}
	service := NewUserPreferencesService(repo)
	if _, err := service.Get(context.Background(), "user-a"); !errors.Is(err, repo.err) {
		t.Fatal("read error swallowed")
	}
	dark := "dark"
	if _, err := service.Update(context.Background(), "user-a", domain.UpdateUserPreferences{Theme: &dark}); !errors.Is(err, repo.err) {
		t.Fatal("write error swallowed")
	}
}
