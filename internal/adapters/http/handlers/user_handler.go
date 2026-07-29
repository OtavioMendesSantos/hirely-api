package handlers

import (
	"errors"
	"hirely-api/internal/adapters/logger"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *services.UserService
}

func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

func (h *UserHandler) GetMe(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		slog.Warn("Unauthorized GetMe attempt: missing userID in context",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "GetMe"),
		)
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"message": "Authentication required", "status": "UNAUTHENTICATED"}})
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "User not found", "status": "NOT_FOUND"}})
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid user ID", "status": "INVALID_ARGUMENT"}})
			return
		}

		slog.Error("Error retrieving user profile",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "GetMe"),
			slog.String("error", err.Error()),
			slog.Any("context", map[string]string{
				"userId": userID,
			}),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Internal server error", "status": "INTERNAL"}})
		return
	}

	c.JSON(http.StatusOK, user)
}
