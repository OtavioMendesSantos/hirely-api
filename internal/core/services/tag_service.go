package services

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/ports"
)

type TagService struct {
	tagRepo ports.TagRepository
}

func NewTagService(tagRepo ports.TagRepository) *TagService {
	return &TagService{tagRepo: tagRepo}
}

func (s *TagService) CreateTag(ctx context.Context, userID, name, colorHex string) (*domain.Tag, error) {
	userID = strings.TrimSpace(userID)
	name = strings.TrimSpace(name)
	colorHex = strings.TrimSpace(colorHex)

	if userID == "" || name == "" || colorHex == "" {
		return nil, domain.ErrInvalidInput
	}

	tag := domain.NewTag(uuid.NewString(), userID, name, colorHex)
	
	if err := s.tagRepo.Create(ctx, tag); err != nil {
		return nil, err
	}

	return tag, nil
}

func (s *TagService) ListTags(ctx context.Context, userID string) ([]*domain.Tag, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, domain.ErrInvalidInput
	}

	return s.tagRepo.ListByUserID(ctx, userID)
}

func (s *TagService) DeleteTag(ctx context.Context, userID, tagID string) error {
	userID = strings.TrimSpace(userID)
	tagID = strings.TrimSpace(tagID)

	if userID == "" || tagID == "" {
		return domain.ErrInvalidInput
	}

	tag, err := s.tagRepo.FindByID(ctx, tagID)
	if err != nil {
		return err
	}
	if tag == nil {
		return domain.ErrTagNotFound
	}

	if tag.UserID != userID {
		return domain.ErrForbidden
	}

	return s.tagRepo.Delete(ctx, tagID)
}
