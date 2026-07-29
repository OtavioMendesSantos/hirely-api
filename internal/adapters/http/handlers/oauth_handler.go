package handlers

import (
	"encoding/json"
	"hirely-api/internal/adapters/http/dto"
	"hirely-api/internal/core/services"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
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

func (h *OAuthHandler) GoogleAuthURL(c *gin.Context) {
	url := h.oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	c.JSON(http.StatusOK, gin.H{"url": url})
}

type OAuthLoginRequest struct {
	Code string `json:"code" binding:"required"`
}

func (h *OAuthHandler) GoogleLogin(c *gin.Context) {
	var req OAuthLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.HandleValidationError(c, err)
		return
	}

	token, err := h.oauthConfig.Exchange(c.Request.Context(), req.Code)
	if err != nil {
		slog.Error("Failed to exchange token", slog.String("error", err.Error()))
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"message": "Failed to exchange token", "status": "UNAUTHENTICATED"}})
		return
	}

	client := h.oauthConfig.Client(c.Request.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		slog.Error("Failed to get user info", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Failed to get user info", "status": "INTERNAL"}})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Failed to decode user info", "status": "INTERNAL"}})
		return
	}

	user, jwtToken, err := h.authService.GoogleLogin(c.Request.Context(), userInfo.Email, userInfo.Name, userInfo.ID)
	if err != nil {
		slog.Error("Failed to login with Google", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Internal server error", "status": "INTERNAL"}})
		return
	}

	c.JSON(http.StatusOK, dto.AuthResponse{
		Token: jwtToken,
		User:  user,
	})
}
