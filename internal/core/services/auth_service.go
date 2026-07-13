package services

import (
	"context"
	"errors"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/ports"
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

func (s *AuthService) RegisterUser(ctx context.Context, name, email, plainPassword string) (*domain.User, error) {
	existingUser, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("Email already registered")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:           uuid.New().String(),
		Name:         name,
		Email:        email,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now().UTC(),
	}

	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, plainPassword string) (string, error) {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return "", err
	}

	invalidCredsErr := errors.New("Email or password invalid")
	if user == nil {
		return "", invalidCredsErr
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(plainPassword))
	if err != nil {
		return "", invalidCredsErr
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"exp":   time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
