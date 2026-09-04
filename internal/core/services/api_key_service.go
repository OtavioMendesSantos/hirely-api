package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/ports"
	"time"

	"github.com/google/uuid"
)

type APIKeyService struct {
	repo ports.APIKeyRepository
}

func NewAPIKeyService(repo ports.APIKeyRepository) *APIKeyService {
	return &APIKeyService{
		repo: repo,
	}
}

func (s *APIKeyService) CreateAPIKey(ctx context.Context, userID, name string) (*domain.APIKey, string, error) {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	rawKey := "hirely_sk_" + base64.RawURLEncoding.EncodeToString(bytes)

	hash := sha256.Sum256([]byte(rawKey))
	hashStr := hex.EncodeToString(hash[:])

	apiKey := &domain.APIKey{
		ID:         uuid.New().String(),
		UserID:     userID,
		Name:       name,
		KeyHash:    hashStr,
		UsageCount: 0,
		Revoked:    false,
		CreatedAt:  time.Now(),
	}

	if err := s.repo.Create(ctx, apiKey); err != nil {
		return nil, "", err
	}

	return apiKey, rawKey, nil
}

func (s *APIKeyService) ListUserAPIKeys(ctx context.Context, userID string) ([]*domain.APIKey, error) {
	return s.repo.FindByUserID(ctx, userID)
}

func (s *APIKeyService) RevokeAPIKey(ctx context.Context, keyID, userID string) error {
	key, err := s.repo.FindByIDAndUserID(ctx, keyID, userID)
	if err != nil {
		return err
	}
	if key == nil {
		return domain.ErrAPIKeyNotFound
	}

	return s.repo.Revoke(ctx, keyID)
}
