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
	SalaryRange        string                   `gorm:"type:varchar(255)"`
	Status             domain.ApplicationStatus `gorm:"type:varchar(50);not null;index"`
	ContractType       *string                  `gorm:"type:varchar(20);check:contract_type IN ('CLT', 'PJ', 'INTERNSHIP', 'OTHER')"`
	AppliedAt          *time.Time               `gorm:"type:timestamp"`
	Location           string                   `gorm:"type:varchar(255)"`
	SubmittedDocuments []string                 `gorm:"type:jsonb;serializer:json"`
	JobDescription     string                   `gorm:"type:text"`
	Notes              string                   `gorm:"type:text"`
	CreatedAt          time.Time                `gorm:"type:timestamp;not null"`
	UpdatedAt          time.Time                `gorm:"type:timestamp;not null"`
	Events             []EventModel             `gorm:"foreignKey:ApplicationID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (ApplicationModel) TableName() string {
	return "applications"
}

func (m *ApplicationModel) ToDomain() *domain.Application {
	var events []domain.Event
	if len(m.Events) > 0 {
		events = make([]domain.Event, len(m.Events))
		for i, e := range m.Events {
			events[i] = *e.ToDomain()
		}
	}

	return &domain.Application{
		ID:                 m.ID,
		UserID:             m.UserID,
		CompanyName:        m.CompanyName,
		JobTitle:           m.JobTitle,
		JobURL:             m.JobURL,
		SalaryRange:        m.SalaryRange,
		Status:             m.Status,
		ContractType:       (*domain.ContractType)(m.ContractType),
		AppliedAt:          m.AppliedAt,
		Location:           m.Location,
		SubmittedDocuments: m.SubmittedDocuments,
		JobDescription:     m.JobDescription,
		Notes:              m.Notes,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
		Events:             events,
	}
}

func ApplicationFromDomain(a *domain.Application) *ApplicationModel {
	var events []EventModel
	if len(a.Events) > 0 {
		events = make([]EventModel, len(a.Events))
		for i, e := range a.Events {
			events[i] = *EventFromDomain(&e)
		}
	}

	return &ApplicationModel{
		ID:                 a.ID,
		UserID:             a.UserID,
		CompanyName:        a.CompanyName,
		JobTitle:           a.JobTitle,
		JobURL:             a.JobURL,
		SalaryRange:        a.SalaryRange,
		Status:             a.Status,
		ContractType:       (*string)(a.ContractType),
		AppliedAt:          a.AppliedAt,
		Location:           a.Location,
		SubmittedDocuments: a.SubmittedDocuments,
		JobDescription:     a.JobDescription,
		Notes:              a.Notes,
		CreatedAt:          a.CreatedAt,
		UpdatedAt:          a.UpdatedAt,
		Events:             events,
	}
}
