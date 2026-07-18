package postgres

import (
	"hirely-api/internal/core/domain"
	"time"
)

type ApplicationModel struct {
	ID                 string                   `gorm:"type:uuid;primaryKey"`
	UserID             string                   `gorm:"type:uuid;not null;index"`
	User               UserModel                `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CompanyName        string                   `gorm:"type:varchar(255);not null"`
	JobTitle           string                   `gorm:"type:varchar(255);not null"`
	JobURL             string                   `gorm:"type:text"`
	Status             domain.ApplicationStatus `gorm:"type:varchar(50);not null;index"`
	AppliedAt          *time.Time               `gorm:"type:timestamp"`
	Location           string                   `gorm:"type:varchar(255)"`
	SubmittedDocuments []string                 `gorm:"type:jsonb;serializer:json"`
	JobDescription     string                   `gorm:"type:text"`
	Notes              string                   `gorm:"type:text"`
	CreatedAt          time.Time                `gorm:"type:timestamp;not null"`
	UpdatedAt          time.Time                `gorm:"type:timestamp;not null"`
}

func (ApplicationModel) TableName() string {
	return "applications"
}

func (m *ApplicationModel) ToDomain() *domain.Application {
	return &domain.Application{
		ID:                 m.ID,
		UserID:             m.UserID,
		CompanyName:        m.CompanyName,
		JobTitle:           m.JobTitle,
		JobURL:             m.JobURL,
		Status:             m.Status,
		AppliedAt:          m.AppliedAt,
		Location:           m.Location,
		SubmittedDocuments: m.SubmittedDocuments,
		JobDescription:     m.JobDescription,
		Notes:              m.Notes,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

func ApplicationFromDomain(a *domain.Application) *ApplicationModel {
	return &ApplicationModel{
		ID:                 a.ID,
		UserID:             a.UserID,
		CompanyName:        a.CompanyName,
		JobTitle:           a.JobTitle,
		JobURL:             a.JobURL,
		Status:             a.Status,
		AppliedAt:          a.AppliedAt,
		Location:           a.Location,
		SubmittedDocuments: a.SubmittedDocuments,
		JobDescription:     a.JobDescription,
		Notes:              a.Notes,
		CreatedAt:          a.CreatedAt,
		UpdatedAt:          a.UpdatedAt,
	}
}
