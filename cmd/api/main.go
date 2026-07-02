package main

import (
	"fmt"
	adapterHttp "hirely-api/internal/adapters/http"
	"hirely-api/internal/config"
	"log"
	"net/http"
	"time"
)

func init() {
	time.Local = time.UTC
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	mux := adapterHttp.SetupRoutes()
	
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Server HTTP is running on port %s", cfg.Port)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
