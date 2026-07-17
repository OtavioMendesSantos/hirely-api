package dto

import "hirely-api/internal/core/domain"

type AuthResponse struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}
