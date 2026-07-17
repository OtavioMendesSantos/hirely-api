package http

import (
	"hirely-api/internal/adapters/http/handlers"
	"hirely-api/internal/adapters/http/middleware"
	"net/http"
)

func SetupRoutes(authHandler *handlers.AuthHandler, userHandler *handlers.UserHandler, healthHandler *handlers.HealthHandler, jwtSecret string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/health", healthHandler.Check)
	mux.HandleFunc("POST /v1/users", authHandler.Register)
	mux.HandleFunc("POST /v1/users:login", authHandler.Login)

	authGuard := middleware.Auth(jwtSecret)
	mux.Handle("GET /v1/users/me", authGuard(http.HandlerFunc(userHandler.GetMe)))

	return middleware.Trace(middleware.CORS(mux))
}
