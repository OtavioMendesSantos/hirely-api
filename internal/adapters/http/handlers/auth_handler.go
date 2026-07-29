package handlers

import (
	"errors"
	"hirely-api/internal/adapters/http/dto"
	"hirely-api/internal/adapters/logger"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

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
		slog.Warn("Invalid payload on register",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "RegisterUser"),
			slog.String("error", err.Error()),
		)
		dto.HandleValidationError(c, err)
		return
	}

	user, tokenString, err := h.authService.RegisterUser(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			dto.WriteError(c, http.StatusConflict, "Email already registered", "ALREADY_EXISTS")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			dto.WriteError(c, http.StatusBadRequest, "Invalid input parameters", "INVALID_ARGUMENT")
			return
		}

		slog.Error("Error registering user",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "RegisterUser"),
			slog.String("error", err.Error()),
			slog.Any("context", map[string]string{
				"email": req.Email,
			}),
		)
		dto.WriteError(c, http.StatusInternalServerError, "Internal server error", "INTERNAL")
		return
	}

	c.JSON(http.StatusCreated, dto.AuthResponse{
		Token: tokenString,
		User:  user,
	})
}

type LoginRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required"`
	RememberMe bool   `json:"rememberMe" binding:"omitempty"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("Invalid payload on login",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "Login"),
			slog.String("error", err.Error()),
		)
		dto.HandleValidationError(c, err)
		return
	}

	user, tokenString, err := h.authService.Login(c.Request.Context(), req.Email, req.Password, req.RememberMe)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			dto.WriteError(c, http.StatusUnauthorized, "Invalid email or password", "UNAUTHENTICATED")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			dto.WriteError(c, http.StatusBadRequest, "Invalid input parameters", "INVALID_ARGUMENT")
			return
		}

		slog.Error("Internal error on login",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "Login"),
			slog.String("error", err.Error()),
			slog.Any("context", map[string]string{
				"email": req.Email,
			}),
		)
		dto.WriteError(c, http.StatusInternalServerError, "Internal server error", "INTERNAL")
		return
	}

	c.JSON(http.StatusOK, dto.AuthResponse{
		Token: tokenString,
		User:  user,
	})
}
