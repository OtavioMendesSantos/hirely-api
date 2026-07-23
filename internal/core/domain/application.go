package domain

import (
	"time"
)

type ContractType string

const (
	ContractTypeCLT        ContractType = "CLT"
	ContractTypePJ         ContractType = "PJ"
	ContractTypeInternship ContractType = "INTERNSHIP"
	ContractTypeOther      ContractType = "OTHER"
)

func (c ContractType) IsValid() bool {
	switch c {
	case ContractTypeCLT, ContractTypePJ, ContractTypeInternship, ContractTypeOther:
		return true
	}
	return false
}

type ApplicationStatus string

const (
	StatusToApply   ApplicationStatus = "TO_APPLY"
	StatusApplied   ApplicationStatus = "APPLIED"
	StatusInterview ApplicationStatus = "INTERVIEW"
	StatusOffer     ApplicationStatus = "OFFER"
	StatusAccepted  ApplicationStatus = "ACCEPTED"
	StatusRejected  ApplicationStatus = "REJECTED"
	StatusOther     ApplicationStatus = "OTHER"
)

type Application struct {
	ID                 string            `json:"id"`
	UserID             string            `json:"userId"`
	CompanyName        string            `json:"companyName"`
	JobTitle           string            `json:"jobTitle"`
	JobURL             string            `json:"jobUrl"`
	SalaryRange        string            `json:"salaryRange,omitempty"`
	Status             ApplicationStatus `json:"status"`
	ContractType       *ContractType     `json:"contractType,omitempty"`
	AppliedAt          *time.Time        `json:"appliedAt"`
	Location           string            `json:"location"`
	SubmittedDocuments []string          `json:"submittedDocuments"`
	JobDescription     string            `json:"jobDescription"`
	Notes              string            `json:"notes"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
	Events             []Event           `json:"events,omitempty"`
}

func NewApplication(id, userID, companyName, jobTitle string, status ApplicationStatus) *Application {
	now := time.Now().UTC()
	return &Application{
		ID:                 id,
		UserID:             userID,
		CompanyName:        companyName,
		JobTitle:           jobTitle,
		Status:             status,
		SubmittedDocuments: make([]string, 0),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func AllStatuses() []ApplicationStatus {
	return []ApplicationStatus{
		StatusToApply,
		StatusApplied,
		StatusInterview,
		StatusOffer,
		StatusAccepted,
		StatusRejected,
		StatusOther,
	}
}
