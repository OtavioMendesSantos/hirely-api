package handlers

import (
	"errors"
	"hirely-api/internal/adapters/http/dto"
	"hirely-api/internal/adapters/logger"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ApplicationHandler struct {
	appService *services.ApplicationService
}

func NewApplicationHandler(appService *services.ApplicationService) *ApplicationHandler {
	return &ApplicationHandler{
		appService: appService,
	}
}

func (h *ApplicationHandler) checkIsolation(c *gin.Context) (string, bool) {
	authUserID := c.GetString("userID")
	if authUserID == "" {
		slog.Warn("Unauthorized application request: missing userID in context",
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

func (h *ApplicationHandler) Create(c *gin.Context) {
	userID, ok := h.checkIsolation(c)
	if !ok {
		return
	}

	var req dto.CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("Failed to decode application create request",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "CreateApplication"),
			slog.String("error", err.Error()),
		)
		dto.HandleValidationError(c, err)
		return
	}

	var cType *domain.ContractType
	if req.ContractType != nil {
		val := domain.ContractType(*req.ContractType)
		cType = &val
	}

	input := services.CreateApplicationInput{
		CompanyName:        req.CompanyName,
		JobTitle:           req.JobTitle,
		JobURL:             req.JobURL,
		SalaryRange:        req.SalaryRange,
		Status:             req.Status,
		ContractType:       cType,
		Location:           req.Location,
		SubmittedDocuments: req.SubmittedDocuments,
		JobDescription:     req.JobDescription,
		Notes:              req.Notes,
		AppliedAt:          req.AppliedAt,
		TagIDs:             req.TagIDs,
	}

	app, err := h.appService.CreateApplication(c.Request.Context(), userID, input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid application data: company_name and job_title are required", "status": "INVALID_ARGUMENT"}})
			return
		}
		slog.Error("Failed to create application",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "CreateApplication"),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Internal server error", "status": "INTERNAL"}})
		return
	}

	c.JSON(http.StatusCreated, app)
}

func (h *ApplicationHandler) List(c *gin.Context) {
	userID, ok := h.checkIsolation(c)
	if !ok {
		return
	}

	var statuses []string
	statusQuery := c.Query("status")
	if statusQuery != "" {
		for _, s := range strings.Split(statusQuery, ",") {
			st := strings.TrimSpace(s)
			if st != "" {
				statuses = append(statuses, st)
			}
		}
	}

	orderBy := c.Query("order_by")
	orderDir := c.Query("order")
	search := c.Query("search")

	apps, err := h.appService.ListApplications(c.Request.Context(), userID, search, statuses, orderBy, orderDir)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid input parameters", "status": "INVALID_ARGUMENT"}})
			return
		}
		slog.Error("Failed to list applications",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "ListApplications"),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Internal server error", "status": "INTERNAL"}})
		return
	}

	resp := dto.ListApplicationsResponse{
		Applications:  apps,
		NextPageToken: "",
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ApplicationHandler) GroupedByStatus(c *gin.Context) {
	userID, ok := h.checkIsolation(c)
	if !ok {
		return
	}

	var statuses []string
	statusQuery := c.Query("status")
	if statusQuery != "" {
		for _, s := range strings.Split(statusQuery, ",") {
			st := strings.TrimSpace(s)
			if st != "" {
				statuses = append(statuses, st)
			}
		}
	}

	orderBy := c.Query("order_by")
	orderDir := c.Query("order")
	search := c.Query("search")

	grouped, err := h.appService.ListApplicationsGroupedByStatus(c.Request.Context(), userID, search, statuses, orderBy, orderDir)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid input parameters", "status": "INVALID_ARGUMENT"}})
			return
		}
		slog.Error("Failed to list applications grouped by status",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "GroupedByStatus"),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Internal server error", "status": "INTERNAL"}})
		return
	}

	resp := dto.GroupedApplicationsResponse{
		GroupedApplications: grouped,
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ApplicationHandler) GetByID(c *gin.Context) {
	userID, ok := h.checkIsolation(c)
	if !ok {
		return
	}

	appID := c.Param("application_id")
	app, err := h.appService.GetApplicationByID(c.Request.Context(), userID, appID)
	if err != nil {
		if errors.Is(err, domain.ErrApplicationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "Application not found", "status": "NOT_FOUND"}})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"message": "Permission denied", "status": "PERMISSION_DENIED"}})
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid application ID", "status": "INVALID_ARGUMENT"}})
			return
		}
		slog.Error("Failed to get application by ID",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "GetApplicationByID"),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Internal server error", "status": "INTERNAL"}})
		return
	}

	c.JSON(http.StatusOK, app)
}

func (h *ApplicationHandler) Update(c *gin.Context) {
	userID, ok := h.checkIsolation(c)
	if !ok {
		return
	}

	appID := c.Param("application_id")
	var req dto.UpdateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("Failed to decode application update request",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "UpdateApplication"),
			slog.String("error", err.Error()),
		)
		dto.HandleValidationError(c, err)
		return
	}

	var cType *domain.ContractType
	if req.ContractType != nil {
		val := domain.ContractType(*req.ContractType)
		cType = &val
	}

	input := services.UpdateApplicationInput{
		CompanyName:        req.CompanyName,
		JobTitle:           req.JobTitle,
		JobURL:             req.JobURL,
		SalaryRange:        req.SalaryRange,
		Status:             req.Status,
		ContractType:       cType,
		Location:           req.Location,
		SubmittedDocuments: req.SubmittedDocuments,
		JobDescription:     req.JobDescription,
		Notes:              req.Notes,
		AppliedAt:          req.AppliedAt,
		TagIDs:             req.TagIDs,
	}

	app, err := h.appService.UpdateApplication(c.Request.Context(), userID, appID, input)
	if err != nil {
		if errors.Is(err, domain.ErrApplicationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "Application not found", "status": "NOT_FOUND"}})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"message": "Permission denied", "status": "PERMISSION_DENIED"}})
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) || errors.Is(err, domain.ErrInvalidStatusTransition) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid update parameters", "status": "INVALID_ARGUMENT"}})
			return
		}
		slog.Error("Failed to update application",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "UpdateApplication"),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Internal server error", "status": "INTERNAL"}})
		return
	}

	c.JSON(http.StatusOK, app)
}

func (h *ApplicationHandler) Delete(c *gin.Context) {
	userID, ok := h.checkIsolation(c)
	if !ok {
		return
	}

	appID := c.Param("application_id")
	err := h.appService.DeleteApplication(c.Request.Context(), userID, appID)
	if err != nil {
		if errors.Is(err, domain.ErrApplicationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "Application not found", "status": "NOT_FOUND"}})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"message": "Permission denied", "status": "PERMISSION_DENIED"}})
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Invalid application ID", "status": "INVALID_ARGUMENT"}})
			return
		}
		slog.Error("Failed to delete application",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "DeleteApplication"),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Internal server error", "status": "INTERNAL"}})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ApplicationHandler) AddEvent(c *gin.Context) {
	userID, ok := h.checkIsolation(c)
	if !ok {
		return
	}

	appID := c.Param("application_id")
	var req dto.CreateManualEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Warn("Failed to decode create manual event request",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "AddEvent"),
			slog.String("error", err.Error()),
		)
		dto.HandleValidationError(c, err)
		return
	}

	event, err := h.appService.AddManualEvent(c.Request.Context(), userID, appID, req.Description)
	if err != nil {
		if errors.Is(err, domain.ErrApplicationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"message": "Application not found", "status": "NOT_FOUND"}})
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"message": "Permission denied", "status": "PERMISSION_DENIED"}})
			return
		}
		if errors.Is(err, domain.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"message": "Description is required", "status": "INVALID_ARGUMENT"}})
			return
		}
		slog.Error("Failed to add manual event",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "AddEvent"),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Internal server error", "status": "INTERNAL"}})
		return
	}

	c.JSON(http.StatusCreated, event)
}

func (h *ApplicationHandler) GetStats(c *gin.Context) {
	userID, ok := h.checkIsolation(c)
	if !ok {
		return
	}

	var startDate, endDate *time.Time

	startStr := c.Query("start_date")
	if startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startDate = &t
		} else if t, err := time.Parse("2006-01-02", startStr); err == nil {
			startDate = &t
		}
	}

	endStr := c.Query("end_date")
	if endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endDate = &t
		} else if t, err := time.Parse("2006-01-02", endStr); err == nil {
			// Include the entire end date by setting it to the end of the day
			t = t.Add(24 * time.Hour).Add(-time.Nanosecond)
			endDate = &t
		}
	}

	stats, err := h.appService.GetApplicationStats(c.Request.Context(), userID, startDate, endDate)
	if err != nil {
		slog.Error("Failed to get application stats",
			slog.String("traceId", logger.GetTraceID(c.Request.Context())),
			slog.String("operation", "GetStats"),
			slog.String("error", err.Error()),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": "Internal server error", "status": "INTERNAL"}})
		return
	}

	resp := dto.ApplicationStatsResponse{
		TotalApplications:       stats.TotalApplications,
		FunnelByStatus:          stats.FunnelByStatus,
		ConversionRateInterview: stats.ConversionRateInterview,
		TopTags:                 stats.TopTags,
	}

	c.JSON(http.StatusOK, resp)
}
