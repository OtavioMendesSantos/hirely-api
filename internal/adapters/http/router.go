package http

import (
	"hirely-api/internal/adapters/http/handlers"
	"hirely-api/internal/adapters/http/middleware"
	"hirely-api/internal/core/ports"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(
	authHandler *handlers.AuthHandler,
	oauthHandler *handlers.OAuthHandler,
	userHandler *handlers.UserHandler,
	appHandler *handlers.ApplicationHandler,
	tagHandler *handlers.TagHandler,
	mcpHandler *handlers.MCPHandler,
	apiKeyHandler *handlers.APIKeyHandler,
	healthHandler *handlers.HealthHandler,
	sessionRepo ports.SessionRepository,
	apiKeyRepo ports.APIKeyRepository,
	env string,
) *gin.Engine {
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
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
		auth.Use(middleware.HybridAuth(sessionRepo, apiKeyRepo))
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

			// Rotas de API Keys
			auth.GET("/users/me/api-keys", apiKeyHandler.List)
			auth.POST("/users/me/api-keys", apiKeyHandler.Create)
			auth.DELETE("/users/me/api-keys/:key_id", apiKeyHandler.Revoke)

			// Rotas MCP
			auth.GET("/mcp/sse", mcpHandler.HandleSSE())
			auth.POST("/mcp/messages", mcpHandler.HandleMessage())
		}
	}

	return r
}
