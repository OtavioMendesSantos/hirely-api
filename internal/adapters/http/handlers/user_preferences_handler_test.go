package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type handlerPreferencesRepo struct {
	userID string
	patch  domain.UpdateUserPreferences
	err    error
}

func (r *handlerPreferencesRepo) FindByUserID(_ context.Context, id string) (*domain.UserPreferences, error) {
	r.userID = id
	return nil, r.err
}
func (r *handlerPreferencesRepo) Update(_ context.Context, id string, patch domain.UpdateUserPreferences) (*domain.UserPreferences, error) {
	r.userID, r.patch = id, patch
	value := domain.DefaultUserPreferences(id)
	if patch.Theme != nil {
		value.Theme = *patch.Theme
	}
	if patch.ConfettiEnabled != nil {
		value.ConfettiEnabled = *patch.ConfettiEnabled
	}
	return value, r.err
}

func TestUserPreferencesHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name, method, body, userID string
		status                     int
	}{
		{"defaults", "GET", "", "account-a", 200},
		{"theme", "PATCH", `{"theme":"dark"}`, "account-a", 200},
		{"disable confetti", "PATCH", `{"confetti_enabled":false}`, "account-a", 200},
		{"invalid theme", "PATCH", `{"theme":"sepia"}`, "account-a", 400},
		{"invalid bool", "PATCH", `{"confetti_enabled":"false"}`, "account-a", 400},
		{"unknown fields", "PATCH", `{"user_id":"another-account"}`, "account-a", 400},
		{"malformed", "PATCH", `{`, "account-a", 400},
		{"extra payload", "PATCH", `{} {}`, "account-a", 400},
		{"anonymous read", "GET", "", "", 401},
		{"anonymous write", "PATCH", `{"theme":"dark"}`, "", 401},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &handlerPreferencesRepo{}
			handler := NewUserPreferencesHandler(services.NewUserPreferencesService(repo))
			response := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(response)
			c.Set("userID", test.userID)
			c.Request = httptest.NewRequest(test.method, "/v1/users/me/preferences", strings.NewReader(test.body))
			c.Request.Header.Set("Content-Type", "application/json")
			if test.method == "GET" {
				handler.Get(c)
			} else {
				handler.Update(c)
			}
			if response.Code != test.status {
				t.Fatalf("got %d: %s", response.Code, response.Body.String())
			}
			if test.status == 200 {
				if repo.userID != test.userID {
					t.Fatal("did not use authenticated user")
				}
				var preferences domain.UserPreferences
				if err := json.Unmarshal(response.Body.Bytes(), &preferences); err != nil {
					t.Fatal(err)
				}
				if test.name == "disable confetti" && (preferences.ConfettiEnabled || repo.patch.ConfettiEnabled == nil) {
					t.Fatal("explicit false was lost")
				}
			}
		})
	}
}

func TestUserPreferencesHandlerDatabaseFailure(t *testing.T) {
	repo := &handlerPreferencesRepo{err: errors.New("database down")}
	handler := NewUserPreferencesHandler(services.NewUserPreferencesService(repo))
	response := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(response)
	c.Set("userID", "account-a")
	c.Request = httptest.NewRequest("GET", "/v1/users/me/preferences", nil)
	handler.Get(c)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("got %d", response.Code)
	}
	if strings.Contains(response.Body.String(), "database down") {
		t.Fatal("internal error exposed")
	}
}
