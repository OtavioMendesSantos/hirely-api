package handlers

import (
	"encoding/json"
	"hirely-api/internal/adapters/http/dto"
	"hirely-api/internal/core/services"
	"log/slog"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type OAuthHandler struct {
	authService *services.AuthService
	oauthConfig *oauth2.Config
}

func NewOAuthHandler(authService *services.AuthService, clientID, clientSecret, redirectURL string) *OAuthHandler {
	return &OAuthHandler{
		authService: authService,
		oauthConfig: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		},
	}
}

func (h *OAuthHandler) GoogleAuthURL(w http.ResponseWriter, r *http.Request) {
	url := h.oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}

type OAuthLoginRequest struct {
	Code string `json:"code"`
}

func (h *OAuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	var req OAuthLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		dto.WriteError(w, http.StatusBadRequest, "Invalid JSON payload", "INVALID_ARGUMENT")
		return
	}

	if req.Code == "" {
		dto.WriteError(w, http.StatusBadRequest, "Code is required", "INVALID_ARGUMENT")
		return
	}

	token, err := h.oauthConfig.Exchange(r.Context(), req.Code)
	if err != nil {
		slog.Error("Failed to exchange token", slog.String("error", err.Error()))
		dto.WriteError(w, http.StatusUnauthorized, "Failed to exchange token", "UNAUTHENTICATED")
		return
	}

	client := h.oauthConfig.Client(r.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		slog.Error("Failed to get user info", slog.String("error", err.Error()))
		dto.WriteError(w, http.StatusInternalServerError, "Failed to get user info", "INTERNAL")
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		slog.Error("Failed to decode user info", slog.String("error", err.Error()))
		dto.WriteError(w, http.StatusInternalServerError, "Failed to decode user info", "INTERNAL")
		return
	}

	user, jwtToken, err := h.authService.GoogleLogin(r.Context(), userInfo.Email, userInfo.Name, userInfo.ID)
	if err != nil {
		slog.Error("Failed to login with Google", slog.String("error", err.Error()))
		dto.WriteError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dto.AuthResponse{
		Token: jwtToken,
		User:  user,
	})
}
