package handlers

import (
	"encoding/json"
	"hirely-api/internal/core/services"
	"log/slog"
	"net/http"
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

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("Invalid payload on register", slog.String("error", err.Error()))
		http.Error(w, `{"error": "invalid payload"}`, http.StatusBadRequest)
		return
	}

	user, err := h.authService.RegisterUser(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		if err.Error() == "Email already registered" {
			http.Error(w, `{"error": "Email already registered"}`, http.StatusConflict)
			return
		}

		slog.Error("Error registering user",
			slog.String("error", err.Error()),
			slog.String("operation", "RegisterUser"),
		)
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
	})
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("Payload inválido no login", slog.String("error", err.Error()))
		http.Error(w, `{"error": "payload inválido"}`, http.StatusBadRequest)
		return
	}

	token, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if err.Error() == "credenciais inválidas" {
			http.Error(w, `{"error": "e-mail ou senha incorretos"}`, http.StatusUnauthorized)
			return
		}

		slog.Error("Erro interno no login", slog.String("error", err.Error()))
		http.Error(w, `{"error": "erro interno do servidor"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}
