package http

import (
	"hirely-api/internal/adapters/http/handlers"
	"hirely-api/internal/adapters/http/middleware"
	"net/http"
)

func SetupRoutes(authHandler *handlers.AuthHandler, userHandler *handlers.UserHandler, appHandler *handlers.ApplicationHandler, healthHandler *handlers.HealthHandler, jwtSecret string) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/health", healthHandler.Check)
	mux.HandleFunc("POST /v1/users", authHandler.Register)
	mux.HandleFunc("POST /v1/users:login", authHandler.Login)

	authGuard := middleware.Auth(jwtSecret)
	mux.Handle("GET /v1/users/me", authGuard(http.HandlerFunc(userHandler.GetMe)))
	mux.Handle("POST /v1/users/{user_id}/applications", authGuard(http.HandlerFunc(appHandler.Create)))
	mux.Handle("GET /v1/users/{user_id}/applications", authGuard(http.HandlerFunc(appHandler.List)))
	mux.Handle("GET /v1/users/{user_id}/applications/grouped-by-status", authGuard(http.HandlerFunc(appHandler.GroupedByStatus)))
	mux.Handle("GET /v1/users/{user_id}/applications/{application_id}", authGuard(http.HandlerFunc(appHandler.GetByID)))
	mux.Handle("PATCH /v1/users/{user_id}/applications/{application_id}", authGuard(http.HandlerFunc(appHandler.Update)))
	mux.Handle("DELETE /v1/users/{user_id}/applications/{application_id}", authGuard(http.HandlerFunc(appHandler.Delete)))
	mux.Handle("POST /v1/users/{user_id}/applications/{application_id}/events", authGuard(http.HandlerFunc(appHandler.AddEvent)))

	return middleware.Trace(middleware.CORS(mux))
}
