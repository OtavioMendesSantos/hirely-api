package domain

import "errors"

var (
	ErrEmailAlreadyExists      = errors.New("email already registered")
	ErrInvalidCredentials      = errors.New("invalid email or password")
	ErrInvalidInput            = errors.New("invalid input parameters")
	ErrUserNotFound            = errors.New("user not found")
	ErrApplicationArchived     = errors.New("restore this application before editing it")
	ErrEventNotFound           = errors.New("event not found")
	ErrApplicationNotFound     = errors.New("application not found")
	ErrTagNotFound             = errors.New("tag not found")
	ErrTagAlreadyExists        = errors.New("tag with this name already exists")
	ErrForbidden               = errors.New("permission denied")
	ErrAPIKeyNotFound          = errors.New("api key not found")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
)
