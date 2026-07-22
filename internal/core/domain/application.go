package domain

import (
	"time"
)

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
	Status             ApplicationStatus `json:"status"`
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
