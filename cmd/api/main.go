package main

import (
	"context"
	"fmt"
	adapterHttp "hirely-api/internal/adapters/http"
	"hirely-api/internal/adapters/http/handlers"
	"hirely-api/internal/adapters/logger"
	"hirely-api/internal/adapters/storage/postgres"
	"hirely-api/internal/config"
	"hirely-api/internal/core/services"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gorm.io/gorm"
)

func init() {
	time.Local = time.UTC
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	logger.Setup(cfg.ENV, "hirely-api")
	slog.Info("Starting API...", slog.String("operation", "SystemBoot"))

	db := initDB(cfg)

	userRepo := postgres.NewUserRepository(db)
	authService := services.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpiresIn)
	authHandler := handlers.NewAuthHandler(authService)

	mux := adapterHttp.SetupRoutes(authHandler)

	startHTTPServer(cfg, mux)
}

func initDB(cfg *config.Config) *gorm.DB {
	dbConfig := postgres.DBConfig{
		Host:     cfg.DB_HOST,
		Port:     cfg.DB_PORT,
		User:     cfg.DB_USER,
		Password: cfg.DB_PASSWORD,
		DBName:   cfg.DB_NAME,
		SSLMode:  cfg.DB_SSLMODE,
	}

	db, err := postgres.NewConnection(dbConfig)
	if err != nil {
		slog.Error("Error connecting to database",
			slog.String("error", err.Error()),
			slog.String("operation", "InitDB"),
		)
		os.Exit(1)
	}

	slog.Info("Database connected and migrated successfully", slog.String("operation", "InitDB"))
	return db
}

func startHTTPServer(cfg *config.Config, handler http.Handler) {
	addr := fmt.Sprintf(":%s", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	go func() {
		slog.Info("Server HTTP is running",
			slog.String("port", cfg.Port),
			slog.String("operation", "HttpServer"),
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Error starting server",
				slog.String("error", err.Error()),
				slog.String("operation", "HttpServer"),
			)
			os.Exit(1)
		}
	}()

	// Capture signals for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	slog.Info("Shutdown signal received...", slog.String("operation", "GracefulShutdown"))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Error forcing shutdown",
			slog.String("error", err.Error()),
			slog.String("operation", "GracefulShutdown"),
		)
		os.Exit(1)
	}
	slog.Info("Server shutdown gracefully", slog.String("operation", "GracefulShutdown"))
}
