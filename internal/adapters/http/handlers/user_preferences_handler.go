package handlers

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"hirely-api/internal/adapters/http/dto"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"
	"io"
	"log/slog"
	"net/http"
)

type UserPreferencesHandler struct {
	service *services.UserPreferencesService
}

func NewUserPreferencesHandler(service *services.UserPreferencesService) *UserPreferencesHandler {
	return &UserPreferencesHandler{service: service}
}

func (h *UserPreferencesHandler) Get(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		preferencesError(c, http.StatusUnauthorized, "Authentication required")
		return
	}
	preferences, err := h.service.Get(c.Request.Context(), userID)
	h.respond(c, preferences, err)
}

func (h *UserPreferencesHandler) Update(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		preferencesError(c, http.StatusUnauthorized, "Authentication required")
		return
	}
	var request dto.UpdateUserPreferencesRequest
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		preferencesError(c, http.StatusBadRequest, "Invalid preferences")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		preferencesError(c, http.StatusBadRequest, "Invalid preferences")
		return
	}
	preferences, err := h.service.Update(c.Request.Context(), userID, domain.UpdateUserPreferences{
		Theme: request.Theme, ConfettiEnabled: request.ConfettiEnabled, AutoArchiveDays: request.AutoArchiveDays,
	})
	h.respond(c, preferences, err)
}

func (h *UserPreferencesHandler) respond(c *gin.Context, preferences *domain.UserPreferences, err error) {
	if errors.Is(err, domain.ErrInvalidInput) {
		preferencesError(c, http.StatusBadRequest, "Invalid preferences")
		return
	}
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "User preferences operation failed", "error", err)
		preferencesError(c, http.StatusInternalServerError, "Could not save or load preferences")
		return
	}
	c.JSON(http.StatusOK, preferences)
}

func preferencesError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": gin.H{"message": message}})
}
