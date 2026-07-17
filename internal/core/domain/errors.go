package domain

import "errors"

var (
	ErrEmailAlreadyExists = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidInput       = errors.New("invalid input parameters")
	ErrUserNotFound       = errors.New("user not found")
)
