package handlers

import (
	"encoding/json"
	"errors"
	"hirely-api/internal/adapters/http/dto"
	"hirely-api/internal/adapters/http/middleware"
	"hirely-api/internal/adapters/logger"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"
	"log/slog"
	"net/http"
)

type UserHandler struct {
	userService *services.UserService
}

func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		slog.Warn("Unauthorized GetMe attempt: missing userID in context",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "GetMe"),
		)
		dto.WriteError(w, http.StatusUnauthorized, "Authentication required", "UNAUTHENTICATED")
		return
	}

	user, err := h.userService.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			dto.WriteError(w, http.StatusNotFound, "User not found", "NOT_FOUND")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			dto.WriteError(w, http.StatusBadRequest, "Invalid user ID", "INVALID_ARGUMENT")
			return
		}

		slog.Error("Error retrieving user profile",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "GetMe"),
			slog.String("error", err.Error()),
			slog.Any("context", map[string]string{
				"userId": userID,
			}),
		)
		dto.WriteError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}
