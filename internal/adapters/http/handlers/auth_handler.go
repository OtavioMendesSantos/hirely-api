package handlers

import (
	"encoding/json"
	"errors"
	"hirely-api/internal/adapters/http/dto"
	"hirely-api/internal/adapters/logger"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
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
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (req *RegisterRequest) Validate() error {
	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(req.Email)

	if len(name) < 2 {
		return errors.New("name must be at least 2 characters long")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return errors.New("invalid email address format")
	}
	if len(req.Password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	return nil
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("Invalid payload on register",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "RegisterUser"),
			slog.String("error", err.Error()),
		)
		dto.WriteError(w, http.StatusBadRequest, "Invalid JSON payload", "INVALID_ARGUMENT")
		return
	}

	if err := req.Validate(); err != nil {
		slog.Warn("Validation error on register",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "RegisterUser"),
			slog.String("error", err.Error()),
			slog.Any("context", map[string]string{
				"email": req.Email,
			}),
		)
		dto.WriteError(w, http.StatusBadRequest, err.Error(), "INVALID_ARGUMENT")
		return
	}

	user, tokenString, err := h.authService.RegisterUser(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			dto.WriteError(w, http.StatusConflict, "Email already registered", "ALREADY_EXISTS")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			dto.WriteError(w, http.StatusBadRequest, "Invalid input parameters", "INVALID_ARGUMENT")
			return
		}

		slog.Error("Error registering user",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "RegisterUser"),
			slog.String("error", err.Error()),
			slog.Any("context", map[string]string{
				"email": req.Email,
			}),
		)
		dto.WriteError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dto.AuthResponse{
		Token: tokenString,
		User:  user,
	})
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (req *LoginRequest) Validate() error {
	email := strings.TrimSpace(req.Email)
	if _, err := mail.ParseAddress(email); err != nil {
		return errors.New("invalid email address format")
	}
	if strings.TrimSpace(req.Password) == "" {
		return errors.New("password is required")
	}
	return nil
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("Invalid payload on login",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "Login"),
			slog.String("error", err.Error()),
		)
		dto.WriteError(w, http.StatusBadRequest, "Invalid JSON payload", "INVALID_ARGUMENT")
		return
	}

	if err := req.Validate(); err != nil {
		slog.Warn("Validation error on login",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "Login"),
			slog.String("error", err.Error()),
			slog.Any("context", map[string]string{
				"email": req.Email,
			}),
		)
		dto.WriteError(w, http.StatusBadRequest, err.Error(), "INVALID_ARGUMENT")
		return
	}

	user, tokenString, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			dto.WriteError(w, http.StatusUnauthorized, "Invalid email or password", "UNAUTHENTICATED")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			dto.WriteError(w, http.StatusBadRequest, "Invalid input parameters", "INVALID_ARGUMENT")
			return
		}

		slog.Error("Internal error on login",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "Login"),
			slog.String("error", err.Error()),
			slog.Any("context", map[string]string{
				"email": req.Email,
			}),
		)
		dto.WriteError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dto.AuthResponse{
		Token: tokenString,
		User:  user,
	})
}
