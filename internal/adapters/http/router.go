package http

import (
	"hirely-api/internal/adapters/http/handlers"
	"net/http"
)

func SetupRoutes(authHandler *handlers.AuthHandler) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/health", HealthCheck)
	mux.HandleFunc("POST /v1/auth/register", authHandler.Register)
	mux.HandleFunc("POST /v1/auth/login", authHandler.Login)

	return mux
}
