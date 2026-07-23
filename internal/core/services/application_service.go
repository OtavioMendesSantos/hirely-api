package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/ports"
)

type CreateApplicationInput struct {
	CompanyName        string
	JobTitle           string
	JobURL             string
	SalaryRange        string
	Status             domain.ApplicationStatus
	ContractType       *domain.ContractType
	Location           string
	SubmittedDocuments []string
	JobDescription     string
	Notes              string
	AppliedAt          *time.Time
}

type UpdateApplicationInput struct {
	CompanyName        *string
	JobTitle           *string
	JobURL             *string
	SalaryRange        *string
	Status             *domain.ApplicationStatus
	ContractType       *domain.ContractType
	Location           *string
	SubmittedDocuments []string
	JobDescription     *string
	Notes              *string
	AppliedAt          *time.Time
}

type ApplicationService struct {
	appRepo   ports.ApplicationRepository
	eventRepo ports.EventRepository
}

func NewApplicationService(appRepo ports.ApplicationRepository, eventRepo ports.EventRepository) *ApplicationService {
	return &ApplicationService{
		appRepo:   appRepo,
		eventRepo: eventRepo,
	}
}

func isValidStatus(status domain.ApplicationStatus) bool {
	switch status {
	case domain.StatusToApply, domain.StatusApplied, domain.StatusInterview,
		domain.StatusOffer, domain.StatusAccepted, domain.StatusRejected, domain.StatusOther:
		return true
	default:
		return false
	}
}

func (s *ApplicationService) CreateApplication(ctx context.Context, userID string, input CreateApplicationInput) (*domain.Application, error) {
	userID = strings.TrimSpace(userID)
	companyName := strings.TrimSpace(input.CompanyName)
	jobTitle := strings.TrimSpace(input.JobTitle)

	if userID == "" || companyName == "" || jobTitle == "" {
		return nil, domain.ErrInvalidInput
	}

	status := input.Status
	if status == "" {
		status = domain.StatusToApply
	}
	if !isValidStatus(status) {
		return nil, domain.ErrInvalidInput
	}

	app := domain.NewApplication(uuid.NewString(), userID, companyName, jobTitle, status)
	
	if input.ContractType != nil && !input.ContractType.IsValid() {
		return nil, domain.ErrInvalidInput
	}
	app.ContractType = input.ContractType
	
	app.JobURL = strings.TrimSpace(input.JobURL)
	app.SalaryRange = strings.TrimSpace(input.SalaryRange)
	app.Location = strings.TrimSpace(input.Location)
	app.JobDescription = input.JobDescription
	app.Notes = input.Notes
	app.AppliedAt = input.AppliedAt
	if input.SubmittedDocuments != nil {
		app.SubmittedDocuments = input.SubmittedDocuments
	}

	event := domain.NewAutomaticEvent(
		uuid.NewString(),
		app.ID,
		fmt.Sprintf("Application created with status %s", app.Status),
		"",
		string(app.Status),
	)

	if err := s.appRepo.UpdateStatus(ctx, app, event); err != nil {
		return nil, err
	}

	app.Events = []domain.Event{*event}
	return app, nil
}

func (s *ApplicationService) GetApplicationByID(ctx context.Context, userID string, appID string) (*domain.Application, error) {
	userID = strings.TrimSpace(userID)
	appID = strings.TrimSpace(appID)

	if userID == "" || appID == "" {
		return nil, domain.ErrInvalidInput
	}

	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, domain.ErrApplicationNotFound
	}
	if app.UserID != userID {
		return nil, domain.ErrForbidden
	}

	return app, nil
}

func normalizeOrderOptions(orderBy, orderDir string) (string, string, bool) {
	orderBy = strings.ToLower(strings.TrimSpace(orderBy))
	orderDir = strings.ToLower(strings.TrimSpace(orderDir))

	if strings.Contains(orderBy, " ") {
		parts := strings.Fields(orderBy)
		if len(parts) >= 2 {
			orderBy = parts[0]
			if orderDir == "" {
				orderDir = parts[1]
			}
		}
	}

	switch orderBy {
	case "", "created_at", "createdat", "create_time":
		orderBy = "created_at"
	case "job_title", "jobtitle", "title":
		orderBy = "job_title"
	case "updated_at", "updatedat", "update_time":
		orderBy = "updated_at"
	case "applied_at", "appliedat", "apply_time":
		orderBy = "applied_at"
	default:
		return "", "", false
	}

	switch orderDir {
	case "", "asc", "ascending":
		if orderDir == "" {
			if orderBy == "job_title" {
				orderDir = "asc"
			} else {
				orderDir = "desc"
			}
		} else {
			orderDir = "asc"
		}
	case "desc", "descending":
		orderDir = "desc"
	default:
		return "", "", false
	}

	return orderBy, orderDir, true
}

func (s *ApplicationService) ListApplications(ctx context.Context, userID string, search string, statusFilters []string, orderBy string, orderDir string) ([]*domain.Application, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, domain.ErrInvalidInput
	}

	normOrderBy, normOrderDir, ok := normalizeOrderOptions(orderBy, orderDir)
	if !ok {
		return nil, domain.ErrInvalidInput
	}

	var validFilters []string
	for _, st := range statusFilters {
		stTrim := strings.TrimSpace(st)
		if stTrim != "" && isValidStatus(domain.ApplicationStatus(stTrim)) {
			validFilters = append(validFilters, stTrim)
		}
	}

	if len(validFilters) > 0 {
		return s.appRepo.ListByUserIDWithFilters(ctx, userID, search, validFilters, normOrderBy, normOrderDir)
	}
	return s.appRepo.ListByUserID(ctx, userID, search, normOrderBy, normOrderDir)
}

func (s *ApplicationService) ListApplicationsGroupedByStatus(ctx context.Context, userID string, search string, statusFilters []string, orderBy string, orderDir string) (map[domain.ApplicationStatus][]*domain.Application, error) {
	apps, err := s.ListApplications(ctx, userID, search, statusFilters, orderBy, orderDir)
	if err != nil {
		return nil, err
	}

	grouped := make(map[domain.ApplicationStatus][]*domain.Application)

	var statusesToInit []domain.ApplicationStatus
	if len(statusFilters) > 0 {
		for _, st := range statusFilters {
			stTrim := strings.TrimSpace(st)
			if stTrim != "" && isValidStatus(domain.ApplicationStatus(stTrim)) {
				statusesToInit = append(statusesToInit, domain.ApplicationStatus(stTrim))
			}
		}
	}
	if len(statusesToInit) == 0 {
		statusesToInit = domain.AllStatuses()
	}

	for _, st := range statusesToInit {
		grouped[st] = make([]*domain.Application, 0)
	}

	for _, app := range apps {
		if _, ok := grouped[app.Status]; ok {
			grouped[app.Status] = append(grouped[app.Status], app)
		} else {
			grouped[app.Status] = []*domain.Application{app}
		}
	}

	return grouped, nil
}

func (s *ApplicationService) UpdateApplication(ctx context.Context, userID string, appID string, input UpdateApplicationInput) (*domain.Application, error) {
	app, err := s.GetApplicationByID(ctx, userID, appID)
	if err != nil {
		return nil, err
	}

	if input.CompanyName != nil {
		trimName := strings.TrimSpace(*input.CompanyName)
		if trimName == "" {
			return nil, domain.ErrInvalidInput
		}
		app.CompanyName = trimName
	}
	if input.JobTitle != nil {
		trimTitle := strings.TrimSpace(*input.JobTitle)
		if trimTitle == "" {
			return nil, domain.ErrInvalidInput
		}
		app.JobTitle = trimTitle
	}
	if input.JobURL != nil {
		app.JobURL = strings.TrimSpace(*input.JobURL)
	}
	if input.SalaryRange != nil {
		app.SalaryRange = strings.TrimSpace(*input.SalaryRange)
	}
	if input.ContractType != nil {
		if !input.ContractType.IsValid() {
			return nil, domain.ErrInvalidInput
		}
		app.ContractType = input.ContractType
	}
	if input.Location != nil {
		app.Location = strings.TrimSpace(*input.Location)
	}
	if input.JobDescription != nil {
		app.JobDescription = *input.JobDescription
	}
	if input.Notes != nil {
		app.Notes = *input.Notes
	}
	if input.AppliedAt != nil {
		app.AppliedAt = input.AppliedAt
	}
	if input.SubmittedDocuments != nil {
		app.SubmittedDocuments = input.SubmittedDocuments
	}
	app.UpdatedAt = time.Now().UTC()

	if input.Status != nil && *input.Status != app.Status {
		if !isValidStatus(*input.Status) {
			return nil, domain.ErrInvalidInput
		}
		oldStatus := app.Status
		app.Status = *input.Status

		event := domain.NewAutomaticEvent(
			uuid.NewString(),
			app.ID,
			fmt.Sprintf("Status changed from %s to %s", oldStatus, app.Status),
			string(oldStatus),
			string(app.Status),
		)

		if err := s.appRepo.UpdateStatus(ctx, app, event); err != nil {
			return nil, err
		}
		app.Events = append(app.Events, *event)
		return app, nil
	}

	if err := s.appRepo.Update(ctx, app); err != nil {
		return nil, err
	}
	return app, nil
}

func (s *ApplicationService) DeleteApplication(ctx context.Context, userID string, appID string) error {
	app, err := s.GetApplicationByID(ctx, userID, appID)
	if err != nil {
		return err
	}
	return s.appRepo.Delete(ctx, app.ID)
}

func (s *ApplicationService) AddManualEvent(ctx context.Context, userID string, appID string, description string) (*domain.Event, error) {
	description = strings.TrimSpace(description)
	if description == "" {
		return nil, domain.ErrInvalidInput
	}

	app, err := s.GetApplicationByID(ctx, userID, appID)
	if err != nil {
		return nil, err
	}

	event := domain.NewManualEvent(uuid.NewString(), app.ID, description)
	if err := s.eventRepo.Create(ctx, event); err != nil {
		return nil, err
	}
	return event, nil
}
