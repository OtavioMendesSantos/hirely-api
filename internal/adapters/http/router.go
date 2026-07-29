package http

import (
	"hirely-api/internal/adapters/http/handlers"
	"hirely-api/internal/adapters/http/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(
	authHandler *handlers.AuthHandler,
	oauthHandler *handlers.OAuthHandler,
	userHandler *handlers.UserHandler,
	appHandler *handlers.ApplicationHandler,
	tagHandler *handlers.TagHandler,
	healthHandler *handlers.HealthHandler,
	jwtSecret string,
) *gin.Engine {
	r := gin.New()

	r.Use(middleware.Trace())
	r.Use(middleware.CORS())

	v1 := r.Group("/v1")
	{
		r.GET("/ping", healthHandler.Ping)
		v1.GET("/health", healthHandler.Check)
		v1.POST("/users", authHandler.Register)
		v1.POST("/auth/login", authHandler.Login)

		v1.GET("/auth/google/url", oauthHandler.GoogleAuthURL)
		v1.POST("/auth/google/login", oauthHandler.GoogleLogin)

		auth := v1.Group("/")
		auth.Use(middleware.Auth(jwtSecret))
		{
			auth.GET("/users/me", userHandler.GetMe)

			auth.POST("/users/:user_id/applications", appHandler.Create)
			auth.GET("/users/:user_id/applications", appHandler.List)
			auth.GET("/users/:user_id/applications/grouped-by-status", appHandler.GroupedByStatus)
			auth.GET("/users/:user_id/applications/:application_id", appHandler.GetByID)
			auth.PATCH("/users/:user_id/applications/:application_id", appHandler.Update)
			auth.DELETE("/users/:user_id/applications/:application_id", appHandler.Delete)
			auth.POST("/users/:user_id/applications/:application_id/events", appHandler.AddEvent)
			auth.GET("/users/:user_id/applications/stats", appHandler.GetStats)

			auth.POST("/users/:user_id/tags", tagHandler.Create)
			auth.GET("/users/:user_id/tags", tagHandler.List)
			auth.DELETE("/users/:user_id/tags/:tag_id", tagHandler.Delete)
		}
	}

	return r
}
