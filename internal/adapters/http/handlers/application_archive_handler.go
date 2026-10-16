package handlers

import (
	"errors"
	"github.com/gin-gonic/gin"
	"hirely-api/internal/core/domain"
	"log/slog"
	"net/http"
)

func (h *ApplicationHandler) Archive(c *gin.Context) { h.setArchived(c, true) }
func (h *ApplicationHandler) Restore(c *gin.Context) { h.setArchived(c, false) }

func (h *ApplicationHandler) setArchived(c *gin.Context, archived bool) {
	userID, ok := h.checkIsolation(c)
	if !ok {
		return
	}
	app, err := h.appService.SetArchived(c.Request.Context(), userID, c.Param("application_id"), archived)
	if err != nil {
		applicationError(c, err)
		return
	}
	c.JSON(http.StatusOK, app)
}

func applicationError(c *gin.Context, err error) {
	code, status, message := http.StatusInternalServerError, "INTERNAL", "Could not complete this action. Try again."
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		code, status, message = 400, "INVALID_ARGUMENT", "Invalid application filters or pagination token."
	case errors.Is(err, domain.ErrEventNotFound):
		code, status, message = 404, "NOT_FOUND", "Event not found."
	case errors.Is(err, domain.ErrApplicationNotFound):
		code, status, message = 404, "NOT_FOUND", "Application not found."
	case errors.Is(err, domain.ErrForbidden):
		code, status, message = 403, "PERMISSION_DENIED", "Permission denied."
	case errors.Is(err, domain.ErrApplicationArchived):
		code, status, message = 409, "FAILED_PRECONDITION", "Restore this application before editing it."
	default:
		slog.ErrorContext(c.Request.Context(), "Application action failed", "error", err)
	}
	c.JSON(code, gin.H{"error": gin.H{"message": message, "status": status}})
}

func (h *ApplicationHandler) DeleteEvent(c *gin.Context) {
	userID, ok := h.checkIsolation(c)
	if !ok {
		return
	}
	if err := h.appService.DeleteEvent(c.Request.Context(), userID, c.Param("application_id"), c.Param("event_id")); err != nil {
		applicationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
