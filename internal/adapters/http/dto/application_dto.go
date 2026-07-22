package dto

import (
	"hirely-api/internal/core/domain"
	"time"
)

type CreateApplicationRequest struct {
	CompanyName        string                   `json:"company_name"`
	JobTitle           string                   `json:"job_title"`
	JobURL             string                   `json:"job_url"`
	Status             domain.ApplicationStatus `json:"status"`
	Location           string                   `json:"location"`
	SubmittedDocuments []string                 `json:"submitted_documents"`
	JobDescription     string                   `json:"job_description"`
	Notes              string                   `json:"notes"`
	AppliedAt          *time.Time               `json:"applied_at"`
}

type UpdateApplicationRequest struct {
	CompanyName        *string                   `json:"company_name"`
	JobTitle           *string                   `json:"job_title"`
	JobURL             *string                   `json:"job_url"`
	Status             *domain.ApplicationStatus `json:"status"`
	Location           *string                   `json:"location"`
	SubmittedDocuments []string                  `json:"submitted_documents"`
	JobDescription     *string                   `json:"job_description"`
	Notes              *string                   `json:"notes"`
	AppliedAt          *time.Time                `json:"applied_at"`
}

type CreateManualEventRequest struct {
	Description string `json:"description"`
}

type ListApplicationsResponse struct {
	Applications  []*domain.Application `json:"applications"`
	NextPageToken string                `json:"next_page_token"`
}

type GroupedApplicationsResponse struct {
	GroupedApplications map[domain.ApplicationStatus][]*domain.Application `json:"grouped_applications"`
}
