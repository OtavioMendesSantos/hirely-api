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

type TagHandler struct {
	tagService *services.TagService
}

func NewTagHandler(tagService *services.TagService) *TagHandler {
	return &TagHandler{
		tagService: tagService,
	}
}

func (h *TagHandler) checkIsolation(w http.ResponseWriter, r *http.Request) (string, bool) {
	authUserID := middleware.GetUserID(r.Context())
	if authUserID == "" {
		slog.Warn("Unauthorized tag request: missing userID in context",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", r.Method+" "+r.URL.Path),
		)
		dto.WriteError(w, http.StatusUnauthorized, "Authentication required", "UNAUTHENTICATED")
		return "", false
	}

	targetUserID := r.PathValue("user_id")
	if targetUserID == "" || targetUserID != authUserID {
		slog.Warn("Permission denied: target user_id does not match authenticated user",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", r.Method+" "+r.URL.Path),
			slog.String("authUserId", authUserID),
			slog.String("targetUserId", targetUserID),
		)
		dto.WriteError(w, http.StatusForbidden, "Permission denied: user_id mismatch", "PERMISSION_DENIED")
		return "", false
	}

	return authUserID, true
}

func (h *TagHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.checkIsolation(w, r)
	if !ok {
		return
	}

	var req dto.CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("Failed to decode tag create request",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "CreateTag"),
			slog.String("error", err.Error()),
		)
		dto.WriteError(w, http.StatusBadRequest, "Invalid JSON payload", "INVALID_ARGUMENT")
		return
	}

	tag, err := h.tagService.CreateTag(r.Context(), userID, req.Name, req.ColorHex)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			dto.WriteError(w, http.StatusBadRequest, "Invalid tag data: name and color_hex are required", "INVALID_ARGUMENT")
			return
		}
		slog.Error("Failed to create tag",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "CreateTag"),
			slog.String("error", err.Error()),
		)
		dto.WriteError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tag)
}

func (h *TagHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.checkIsolation(w, r)
	if !ok {
		return
	}

	tags, err := h.tagService.ListTags(r.Context(), userID)
	if err != nil {
		slog.Error("Failed to list tags",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "ListTags"),
			slog.String("error", err.Error()),
		)
		dto.WriteError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL")
		return
	}

	resp := dto.ListTagsResponse{
		Tags: tags,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *TagHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.checkIsolation(w, r)
	if !ok {
		return
	}

	tagID := r.PathValue("tag_id")
	err := h.tagService.DeleteTag(r.Context(), userID, tagID)
	if err != nil {
		if errors.Is(err, domain.ErrApplicationNotFound) {
			dto.WriteError(w, http.StatusNotFound, "Tag not found", "NOT_FOUND")
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			dto.WriteError(w, http.StatusForbidden, "Permission denied", "PERMISSION_DENIED")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			dto.WriteError(w, http.StatusBadRequest, "Invalid tag ID", "INVALID_ARGUMENT")
			return
		}
		slog.Error("Failed to delete tag",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "DeleteTag"),
			slog.String("error", err.Error()),
		)
		dto.WriteError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
