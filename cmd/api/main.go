package main

import (
	adapterHttp "hirely-api/internal/adapters/http"
	"log"
	"net/http"
	"time"
)

func init() {
	time.Local = time.UTC
}

func main() {
	port := ":8080"
	log.Printf("Server HTTP is running on port %s", port)

	if err := http.ListenAndServe(port, adapterHttp.SetupRoutes()); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
