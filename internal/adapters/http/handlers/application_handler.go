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
	"strings"
)

type ApplicationHandler struct {
	appService *services.ApplicationService
}

func NewApplicationHandler(appService *services.ApplicationService) *ApplicationHandler {
	return &ApplicationHandler{
		appService: appService,
	}
}

func (h *ApplicationHandler) checkIsolation(w http.ResponseWriter, r *http.Request) (string, bool) {
	authUserID := middleware.GetUserID(r.Context())
	if authUserID == "" {
		slog.Warn("Unauthorized application request: missing userID in context",
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

func (h *ApplicationHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.checkIsolation(w, r)
	if !ok {
		return
	}

	var req dto.CreateApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("Failed to decode application create request",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "CreateApplication"),
			slog.String("error", err.Error()),
		)
		dto.WriteError(w, http.StatusBadRequest, "Invalid JSON payload", "INVALID_ARGUMENT")
		return
	}

	input := services.CreateApplicationInput{
		CompanyName:        req.CompanyName,
		JobTitle:           req.JobTitle,
		JobURL:             req.JobURL,
		Status:             req.Status,
		Location:           req.Location,
		SubmittedDocuments: req.SubmittedDocuments,
		JobDescription:     req.JobDescription,
		Notes:              req.Notes,
		AppliedAt:          req.AppliedAt,
	}

	app, err := h.appService.CreateApplication(r.Context(), userID, input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			dto.WriteError(w, http.StatusBadRequest, "Invalid application data: company_name and job_title are required", "INVALID_ARGUMENT")
			return
		}
		slog.Error("Failed to create application",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "CreateApplication"),
			slog.String("error", err.Error()),
		)
		dto.WriteError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(app)
}

func (h *ApplicationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.checkIsolation(w, r)
	if !ok {
		return
	}

	var statuses []string
	statusQuery := r.URL.Query().Get("status")
	if statusQuery != "" {
		for _, s := range strings.Split(statusQuery, ",") {
			st := strings.TrimSpace(s)
			if st != "" {
				statuses = append(statuses, st)
			}
		}
	}

	apps, err := h.appService.ListApplications(r.Context(), userID, statuses)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			dto.WriteError(w, http.StatusBadRequest, "Invalid input parameters", "INVALID_ARGUMENT")
			return
		}
		slog.Error("Failed to list applications",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "ListApplications"),
			slog.String("error", err.Error()),
		)
		dto.WriteError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL")
		return
	}

	resp := dto.ListApplicationsResponse{
		Applications:  apps,
		NextPageToken: "",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *ApplicationHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.checkIsolation(w, r)
	if !ok {
		return
	}

	appID := r.PathValue("application_id")
	app, err := h.appService.GetApplicationByID(r.Context(), userID, appID)
	if err != nil {
		if errors.Is(err, domain.ErrApplicationNotFound) {
			dto.WriteError(w, http.StatusNotFound, "Application not found", "NOT_FOUND")
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			dto.WriteError(w, http.StatusForbidden, "Permission denied", "PERMISSION_DENIED")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			dto.WriteError(w, http.StatusBadRequest, "Invalid application ID", "INVALID_ARGUMENT")
			return
		}
		slog.Error("Failed to get application by ID",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "GetApplicationByID"),
			slog.String("error", err.Error()),
		)
		dto.WriteError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(app)
}

func (h *ApplicationHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.checkIsolation(w, r)
	if !ok {
		return
	}

	appID := r.PathValue("application_id")
	var req dto.UpdateApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("Failed to decode application update request",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "UpdateApplication"),
			slog.String("error", err.Error()),
		)
		dto.WriteError(w, http.StatusBadRequest, "Invalid JSON payload", "INVALID_ARGUMENT")
		return
	}

	input := services.UpdateApplicationInput{
		CompanyName:        req.CompanyName,
		JobTitle:           req.JobTitle,
		JobURL:             req.JobURL,
		Status:             req.Status,
		Location:           req.Location,
		SubmittedDocuments: req.SubmittedDocuments,
		JobDescription:     req.JobDescription,
		Notes:              req.Notes,
		AppliedAt:          req.AppliedAt,
	}

	app, err := h.appService.UpdateApplication(r.Context(), userID, appID, input)
	if err != nil {
		if errors.Is(err, domain.ErrApplicationNotFound) {
			dto.WriteError(w, http.StatusNotFound, "Application not found", "NOT_FOUND")
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			dto.WriteError(w, http.StatusForbidden, "Permission denied", "PERMISSION_DENIED")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) || errors.Is(err, domain.ErrInvalidStatusTransition) {
			dto.WriteError(w, http.StatusBadRequest, "Invalid update parameters", "INVALID_ARGUMENT")
			return
		}
		slog.Error("Failed to update application",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "UpdateApplication"),
			slog.String("error", err.Error()),
		)
		dto.WriteError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(app)
}

func (h *ApplicationHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.checkIsolation(w, r)
	if !ok {
		return
	}

	appID := r.PathValue("application_id")
	err := h.appService.DeleteApplication(r.Context(), userID, appID)
	if err != nil {
		if errors.Is(err, domain.ErrApplicationNotFound) {
			dto.WriteError(w, http.StatusNotFound, "Application not found", "NOT_FOUND")
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			dto.WriteError(w, http.StatusForbidden, "Permission denied", "PERMISSION_DENIED")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			dto.WriteError(w, http.StatusBadRequest, "Invalid application ID", "INVALID_ARGUMENT")
			return
		}
		slog.Error("Failed to delete application",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "DeleteApplication"),
			slog.String("error", err.Error()),
		)
		dto.WriteError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ApplicationHandler) AddEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.checkIsolation(w, r)
	if !ok {
		return
	}

	appID := r.PathValue("application_id")
	var req dto.CreateManualEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Warn("Failed to decode create manual event request",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "AddEvent"),
			slog.String("error", err.Error()),
		)
		dto.WriteError(w, http.StatusBadRequest, "Invalid JSON payload", "INVALID_ARGUMENT")
		return
	}

	event, err := h.appService.AddManualEvent(r.Context(), userID, appID, req.Description)
	if err != nil {
		if errors.Is(err, domain.ErrApplicationNotFound) {
			dto.WriteError(w, http.StatusNotFound, "Application not found", "NOT_FOUND")
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			dto.WriteError(w, http.StatusForbidden, "Permission denied", "PERMISSION_DENIED")
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			dto.WriteError(w, http.StatusBadRequest, "Description is required", "INVALID_ARGUMENT")
			return
		}
		slog.Error("Failed to add manual event",
			slog.String("traceId", logger.GetTraceID(r.Context())),
			slog.String("operation", "AddEvent"),
			slog.String("error", err.Error()),
		)
		dto.WriteError(w, http.StatusInternalServerError, "Internal server error", "INTERNAL")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}
