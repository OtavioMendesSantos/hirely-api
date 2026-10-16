package services

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/google/uuid"
	"hirely-api/internal/core/domain"
	"sort"
	"strings"
	"time"
)

type applicationPageToken struct {
	Version     int                      `json:"v"`
	Fingerprint string                   `json:"query"`
	Cursor      domain.ApplicationCursor `json:"cursor"`
}

func (s *ApplicationService) SetArchived(ctx context.Context, userID, applicationID string, archived bool) (*domain.Application, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(applicationID) == "" {
		return nil, domain.ErrInvalidInput
	}
	return s.appRepo.SetArchived(ctx, userID, applicationID, archived, time.Now().UTC())
}

func normalizeApplicationQuery(query domain.ApplicationQuery) (domain.ApplicationQuery, error) {
	if query.StartDate != nil && query.EndDate != nil && query.StartDate.After(*query.EndDate) {
		return query, domain.ErrInvalidInput
	}
	switch query.Metric {
	case "", "interviews", "rejections", "direct_rejections", "advanced_rejections", "ghosting":
	default:
		return query, domain.ErrInvalidInput
	}
	query.Search = strings.TrimSpace(query.Search)
	if query.ArchiveState == "" {
		query.ArchiveState = "active"
	}
	switch query.ArchiveState {
	case "active", "archived", "all":
	default:
		return query, domain.ErrInvalidInput
	}
	if query.OrderBy == "" && query.ArchiveState == "archived" {
		query.OrderBy = "archived_at"
	}
	var valid bool
	query.OrderBy, query.Order, valid = normalizeOrderOptions(query.OrderBy, query.Order)
	if !valid || query.Limit < 0 || query.Limit > 100 {
		return query, domain.ErrInvalidInput
	}
	if query.Unpaged {
		query.Limit = 0
	} else if query.Limit == 0 && query.ArchiveState == "archived" {
		query.Limit = 20
	}
	query.Statuses = canonicalFilters(query.Statuses)
	for _, status := range query.Statuses {
		if !isValidStatus(domain.ApplicationStatus(status)) {
			return query, domain.ErrInvalidInput
		}
	}
	query.TagIDs = canonicalFilters(query.TagIDs)
	return query, nil
}

func canonicalFilters(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	sort.Strings(result)
	unique := result[:0]
	for _, value := range result {
		if len(unique) == 0 || unique[len(unique)-1] != value {
			unique = append(unique, value)
		}
	}
	return unique
}

func queryFingerprint(userID string, query domain.ApplicationQuery) string {
	query.Cursor = nil
	raw, _ := json.Marshal(struct {
		UserID string
		Query  domain.ApplicationQuery
	}{userID, query})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s *ApplicationService) ListApplicationsPage(ctx context.Context, userID string, query domain.ApplicationQuery, token string) (*domain.ApplicationPage, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, domain.ErrInvalidInput
	}
	// A token also opts legacy active/all reads into pagination.
	if token != "" && query.Limit == 0 && !query.Unpaged {
		query.Limit = 20
	}
	query, err := normalizeApplicationQuery(query)
	if err != nil {
		return nil, err
	}
	fingerprint := queryFingerprint(userID, query)
	if token != "" {
		if len(token) > 4096 || query.Unpaged {
			return nil, domain.ErrInvalidInput
		}
		raw, err := base64.RawURLEncoding.DecodeString(token)
		if err != nil {
			return nil, domain.ErrInvalidInput
		}
		var decoded applicationPageToken
		if err := json.Unmarshal(raw, &decoded); err != nil || decoded.Version != 1 || decoded.Fingerprint != fingerprint {
			return nil, domain.ErrInvalidInput
		}
		if _, err := uuid.Parse(decoded.Cursor.ID); err != nil {
			return nil, domain.ErrInvalidInput
		}
		if decoded.Cursor.Value == nil {
			if query.OrderBy != "applied_at" && query.OrderBy != "archived_at" {
				return nil, domain.ErrInvalidInput
			}
		} else if query.OrderBy != "job_title" {
			if _, err := time.Parse(time.RFC3339Nano, *decoded.Cursor.Value); err != nil {
				return nil, domain.ErrInvalidInput
			}
		}
		query.Cursor = &decoded.Cursor
	}
	query.EvaluatedAt = time.Now().UTC()
	apps, err := s.appRepo.Query(ctx, userID, query)
	if err != nil {
		return nil, err
	}
	page := &domain.ApplicationPage{Applications: apps, TotalCount: int64(len(apps))}
	if query.Limit > 0 {
		countQuery := query
		countQuery.Cursor = nil
		page.TotalCount, err = s.appRepo.CountQuery(ctx, userID, countQuery)
		if err != nil {
			return nil, err
		}
	}
	if query.Limit > 0 && len(apps) > query.Limit {
		page.Applications = apps[:query.Limit]
		cursor := applicationCursor(page.Applications[len(page.Applications)-1], query.OrderBy)
		raw, _ := json.Marshal(applicationPageToken{Version: 1, Fingerprint: fingerprint, Cursor: cursor})
		page.NextPageToken = base64.RawURLEncoding.EncodeToString(raw)
	}
	return page, nil
}

func applicationCursor(app *domain.Application, column string) domain.ApplicationCursor {
	cursor := domain.ApplicationCursor{ID: app.ID}
	var value string
	switch column {
	case "job_title":
		value = app.JobTitle
	case "created_at":
		value = app.CreatedAt.UTC().Format(time.RFC3339Nano)
	case "updated_at":
		value = app.UpdatedAt.UTC().Format(time.RFC3339Nano)
	case "archived_at":
		if app.ArchivedAt == nil {
			return cursor
		}
		value = app.ArchivedAt.UTC().Format(time.RFC3339Nano)
	case "applied_at":
		if app.AppliedAt == nil {
			return cursor
		}
		value = app.AppliedAt.UTC().Format(time.RFC3339Nano)
	}
	cursor.Value = &value
	return cursor
}

func (s *ApplicationService) DeleteEvent(ctx context.Context, userID, applicationID, eventID string) error {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(applicationID) == "" || strings.TrimSpace(eventID) == "" {
		return domain.ErrInvalidInput
	}
	return s.appRepo.DeleteEvent(ctx, userID, applicationID, eventID, time.Now().UTC())
}
