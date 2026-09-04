package handlers

import (
	"errors"
	"hirely-api/internal/adapters/http/dto"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const SessionCookieName = "__Secure-sid"

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=255"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		dto.HandleValidationError(c, err)
		return
	}

	user, err := h.authService.RegisterUser(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			dto.WriteError(c, http.StatusConflict, "Email already registered", "ALREADY_EXISTS")
			return
		}
		dto.WriteError(c, http.StatusInternalServerError, "Internal server error", "INTERNAL")
		return
	}

	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")
	sessionToken, expiresAt, err := h.authService.CreateSession(c.Request.Context(), user.ID, &ip, &ua, false)
	if err != nil {
		dto.WriteError(c, http.StatusInternalServerError, "Failed to create session", "INTERNAL")
		return
	}

	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 86400
	}
	c.SetCookie(SessionCookieName, sessionToken, maxAge, "/", "", true, true)

	c.JSON(http.StatusCreated, gin.H{"user": user})
}

type LoginRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required"`
	RememberMe bool   `json:"rememberMe" binding:"omitempty"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.HandleValidationError(c, err)
		return
	}

	user, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			dto.WriteError(c, http.StatusUnauthorized, "Invalid email or password", "UNAUTHENTICATED")
			return
		}
		dto.WriteError(c, http.StatusInternalServerError, "Internal server error", "INTERNAL")
		return
	}

	ip := c.ClientIP()
	ua := c.GetHeader("User-Agent")
	sessionToken, expiresAt, err := h.authService.CreateSession(c.Request.Context(), user.ID, &ip, &ua, req.RememberMe)
	if err != nil {
		dto.WriteError(c, http.StatusInternalServerError, "Failed to create session", "INTERNAL")
		return
	}

	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 86400
	}
	c.SetCookie(SessionCookieName, sessionToken, maxAge, "/", "", true, true)
	c.JSON(http.StatusOK, gin.H{"user": user})
}
