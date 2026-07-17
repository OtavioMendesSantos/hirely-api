package services

import (
	"context"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/ports"
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo     ports.UserRepository
	jwtSecret    string
	jwtExpiresIn time.Duration
}

func NewAuthService(userRepo ports.UserRepository, jwtSecret string, jwtExpiresIn time.Duration) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		jwtSecret:    jwtSecret,
		jwtExpiresIn: jwtExpiresIn,
	}
}

func (s *AuthService) RegisterUser(ctx context.Context, name, email, plainPassword string) (*domain.User, string, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)

	if len(name) < 2 {
		return nil, "", domain.ErrInvalidInput
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, "", domain.ErrInvalidInput
	}
	if len(plainPassword) < 8 {
		return nil, "", domain.ErrInvalidInput
	}

	existingUser, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}
	if existingUser != nil {
		return nil, "", domain.ErrEmailAlreadyExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	user := domain.NewUser(
		uuid.New().String(),
		name,
		email,
		string(hashedPassword),
	)

	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, "", err
	}

	tokenString, err := s.generateToken(user, s.jwtExpiresIn)
	if err != nil {
		return nil, "", err
	}

	return user, tokenString, nil
}

func (s *AuthService) Login(ctx context.Context, email, plainPassword string, rememberMe bool) (*domain.User, string, error) {
	email = strings.TrimSpace(email)
	if email == "" || plainPassword == "" {
		return nil, "", domain.ErrInvalidInput
	}

	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}

	if user == nil {
		return nil, "", domain.ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(plainPassword))
	if err != nil {
		return nil, "", domain.ErrInvalidCredentials
	}

	expiresIn := s.jwtExpiresIn
	if rememberMe {
		expiresIn = 30 * 24 * time.Hour
	}

	tokenString, err := s.generateToken(user, expiresIn)
	if err != nil {
		return nil, "", err
	}

	return user, tokenString, nil
}

func (s *AuthService) generateToken(user *domain.User, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"exp":   time.Now().Add(expiresIn).Unix(),
	})
	return token.SignedString([]byte(s.jwtSecret))
}

