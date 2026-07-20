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
	Status             domain.ApplicationStatus
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
	Status             *domain.ApplicationStatus
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
	app.JobURL = strings.TrimSpace(input.JobURL)
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
		fmt.Sprintf("Candidatura criada com status %s", app.Status),
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

func (s *ApplicationService) ListApplications(ctx context.Context, userID string, statusFilters []string) ([]*domain.Application, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
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
		return s.appRepo.ListByUserIDWithFilters(ctx, userID, validFilters)
	}
	return s.appRepo.ListByUserID(ctx, userID)
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
			fmt.Sprintf("Status alterado de %s para %s", oldStatus, app.Status),
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
