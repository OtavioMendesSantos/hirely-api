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

type TagHandler struct {
	tagService *services.TagService
}

func NewTagHandler(tagService *services.TagService) *TagHandler {
	return &TagHandler{
		tagService: tagService,
	}
}

func (h *TagHandler) checkIsolation(c *gin.Context) (string, bool) {
	authUserID := c.GetString("userID")
	if authUserID == "" {
		slog.Warn("Unauthorized tag request: missing userID in context",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", c.Request.Method+" "+c.Request.URL.Path),
		)
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"message": "Authentication required", "status": "UNAUTHENTICATED"}})
		return "", false
	}

	targetUserID := c.Param("user_id")
	if targetUserID == "" || targetUserID != authUserID {
		slog.Warn("Permission denied: target user_id does not match authenticated user",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", c.Request.Method+" "+c.Request.URL.Path),
			slog.String("authUserId", authUserID),
			slog.String("targetUserId", targetUserID),
		)
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"message": "Permission denied: user_id mismatch", "status": "PERMISSION_DENIED"}})
		return "", false
	}

	return authUserID, true
}

func (h *TagHandler) Create(c *gin.Context) {
	userID, ok := h.checkIsolation(c)
	if !ok {
		return
	}

	var req dto.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("Failed to decode tag create request",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "CreateTag"),
			slog.String("error", err.Error()),
		)
		dto.HandleValidationError(c, err)
		return
	}

	tag, err := h.tagService.CreateTag(c.Request.Context(), userID, req.Name, req.ColorHex)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid tag data: name and color_hex are required", "status": "INVALID_ARGUMENT"}})
			return
		}
		slog.Error("Failed to create tag",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "CreateTag"),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Internal server error", "status": "INTERNAL"}})
		return
	}

	c.JSON(http.StatusCreated, tag)
}

func (h *TagHandler) List(c *gin.Context) {
	userID, ok := h.checkIsolation(c)
	if !ok {
		return
	}

	tags, err := h.tagService.ListTags(c.Request.Context(), userID)
	if err != nil {
		slog.Error("Failed to list tags",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "ListTags"),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Internal server error", "status": "INTERNAL"}})
		return
	}

	resp := dto.ListTagsResponse{
		Tags: tags,
	}

	c.JSON(http.StatusOK, resp)
}

func (h *TagHandler) Delete(c *gin.Context) {
	userID, ok := h.checkIsolation(c)
	if !ok {
		return
	}

	tagID := c.Param("tag_id")
	err := h.tagService.DeleteTag(c.Request.Context(), userID, tagID)
	if err != nil {
		if errors.Is(err, domain.ErrTagNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "Tag not found", "status": "NOT_FOUND"}})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"message": "Permission denied", "status": "PERMISSION_DENIED"}})
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid tag ID", "status": "INVALID_ARGUMENT"}})
			return
		}
		slog.Error("Failed to delete tag",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "DeleteTag"),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Internal server error", "status": "INTERNAL"}})
		return
	}

	c.Status(http.StatusNoContent)
}
