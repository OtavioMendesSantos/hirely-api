package dto

import (
	"hirely-api/internal/core/domain"
	"time"
)

type CreateApplicationRequest struct {
	CompanyName        string                   `json:"company_name" binding:"required,max=255"`
	JobTitle           string                   `json:"job_title" binding:"required,max=255"`
	JobURL             string                   `json:"job_url" binding:"omitempty,url"`
	SalaryRange        string                   `json:"salary_range" binding:"omitempty,max=100"`
	Status             domain.ApplicationStatus `json:"status" binding:"required,oneof=TO_APPLY APPLIED INTERVIEW OFFER ACCEPTED REJECTED OTHER"`
	ContractType       *string                  `json:"contract_type,omitempty" binding:"omitempty,oneof=CLT PJ INTERNSHIP OTHER"`
	Location           string                   `json:"location" binding:"omitempty,max=255"`
	SubmittedDocuments []string                 `json:"submitted_documents" binding:"omitempty,dive,max=255"`
	JobDescription     string                   `json:"job_description" binding:"omitempty"`
	Notes              string                   `json:"notes" binding:"omitempty"`
	AppliedAt          *time.Time               `json:"applied_at" binding:"omitempty"`
	TagIDs             []string                 `json:"tag_ids,omitempty" binding:"omitempty,dive,uuid"`
}

type UpdateApplicationRequest struct {
	CompanyName        *string                   `json:"company_name" binding:"omitempty,max=255"`
	JobTitle           *string                   `json:"job_title" binding:"omitempty,max=255"`
	JobURL             *string                   `json:"job_url" binding:"omitempty,url"`
	SalaryRange        *string                   `json:"salary_range" binding:"omitempty,max=100"`
	Status             *domain.ApplicationStatus `json:"status" binding:"omitempty,oneof=TO_APPLY APPLIED INTERVIEW OFFER ACCEPTED REJECTED OTHER"`
	ContractType       *string                   `json:"contract_type,omitempty" binding:"omitempty,oneof=CLT PJ INTERNSHIP OTHER"`
	Location           *string                   `json:"location" binding:"omitempty,max=255"`
	SubmittedDocuments []string                  `json:"submitted_documents" binding:"omitempty,dive,max=255"`
	JobDescription     *string                   `json:"job_description" binding:"omitempty"`
	Notes              *string                   `json:"notes" binding:"omitempty"`
	AppliedAt          *time.Time                `json:"applied_at" binding:"omitempty"`
	TagIDs             []string                  `json:"tag_ids,omitempty" binding:"omitempty,dive,uuid"`
}

type CreateManualEventRequest struct {
	Description string `json:"description" binding:"required"`
}

type ListApplicationsResponse struct {
	Applications  []*domain.Application `json:"applications"`
	NextPageToken string                `json:"next_page_token"`
}

type GroupedApplicationsResponse struct {
	GroupedApplications map[domain.ApplicationStatus][]*domain.Application `json:"grouped_applications"`
}

type KPIMetric struct {
	Count int     `json:"count"`
	Rate  float64 `json:"rate"`
}

type KPIs struct {
	Interviews KPIMetric `json:"interviews"`
	Rejections KPIMetric `json:"rejections"`
	Ghosting   KPIMetric `json:"ghosting"`
}

type ApplicationStatsResponse struct {
	TotalApplications int                    `json:"total_applications"`
	FunnelByStatus    map[string]int         `json:"funnel_by_status"`
	KPIs              KPIs                   `json:"kpis"`
	TopTags           []domain.TagCountStats `json:"top_tags"`
	TopJobTitles      []domain.JobTitleStats `json:"top_job_titles"`
}
