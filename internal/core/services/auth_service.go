package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/ports"
	"hirely-api/internal/core/security"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AuthService struct {
	userRepo    ports.UserRepository
	sessionRepo ports.SessionRepository
	pepper      string
}

func NewAuthService(userRepo ports.UserRepository, sessionRepo ports.SessionRepository, pepper string) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		pepper:      pepper,
	}
}

// GenerateSecureToken returns a raw token and its sha256 hash.
func (s *AuthService) GenerateSecureToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	rawToken := base64.RawURLEncoding.EncodeToString(bytes)

	hash := sha256.Sum256([]byte(rawToken))
	hashStr := hex.EncodeToString(hash[:])

	return rawToken, hashStr, nil
}

func (s *AuthService) CreateSession(ctx context.Context, userID string, ip *string, ua *string, rememberMe bool) (string, time.Time, error) {
	rawToken, hashStr, err := s.GenerateSecureToken()
	if err != nil {
		return "", time.Time{}, err
	}

	expiresIn := 24 * time.Hour // 1 dia por padrão
	if rememberMe {
		expiresIn = 15 * 24 * time.Hour // 15 dias
	}
	expiresAt := time.Now().Add(expiresIn)

	session := &domain.Session{
		ID:        uuid.New().String(),
		UserID:    userID,
		Hash:      hashStr,
		IP:        ip,
		UserAgent: ua,
		ExpiresAt: expiresAt,
		Revoked:   false,
		CreatedAt: time.Now(),
	}

	err = s.sessionRepo.Create(ctx, session)
	if err != nil {
		return "", time.Time{}, err
	}

	return rawToken, expiresAt, nil
}

func (s *AuthService) RegisterUser(ctx context.Context, name, email, plainPassword string) (*domain.User, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)

	if len(name) < 2 {
		return nil, domain.ErrInvalidInput
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, domain.ErrInvalidInput
	}
	if len(plainPassword) < 8 {
		return nil, domain.ErrInvalidInput
	}

	existingUser, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, domain.ErrEmailAlreadyExists
	}

	hashedPassword, err := security.HashPassword(plainPassword, s.pepper)
	if err != nil {
		return nil, err
	}

	user := domain.NewUser(
		uuid.New().String(),
		name,
		email,
		string(hashedPassword),
	)

	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, plainPassword string) (*domain.User, error) {
	email = strings.TrimSpace(email)
	if email == "" || plainPassword == "" {
		return nil, domain.ErrInvalidInput
	}

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, domain.ErrInvalidCredentials
	}

	if !security.VerifyPassword(user.PasswordHash, plainPassword, s.pepper) {
		return nil, domain.ErrInvalidCredentials
	}

	return user, nil
}

func (s *AuthService) GoogleLogin(ctx context.Context, email, name, googleID string) (*domain.User, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	if user == nil {
		user = domain.NewUser(
			uuid.New().String(),
			name,
			email,
			"",
		)
		user.GoogleID = googleID
		err = s.userRepo.Create(ctx, user)
		if err != nil {
			return nil, err
		}
	} else if user.GoogleID == "" {
		user.GoogleID = googleID
		err = s.userRepo.Update(ctx, user)
		if err != nil {
			return nil, err
		}
	}

	return user, nil
}
