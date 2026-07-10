package main

import (
	"fmt"
	adapterHttp "hirely-api/internal/adapters/http"
	"hirely-api/internal/adapters/storage/postgres"
	"hirely-api/internal/config"
	"log"
	"net/http"
	"time"

	"gorm.io/gorm"
)

func init() {
	time.Local = time.UTC
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
		log.Fatalf("Error connecting to database: %v", err)
	}
	log.Println("Database connected and migrated successfully!")
	return db
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	initDB(cfg)

	mux := adapterHttp.SetupRoutes()
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server HTTP is running on port %s", cfg.Port)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
