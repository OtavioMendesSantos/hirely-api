package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// Ensure required env vars are set so Load() doesn't fail
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "root")
	os.Setenv("DB_NAME", "hirely")
	os.Setenv("DB_SSLMODE", "disable")
	os.Setenv("GOOGLE_CLIENT_ID", "testclientid")
	os.Setenv("GOOGLE_SECRET_ID", "testsecretid")
	os.Setenv("FRONT_END_URL", "http://localhost:3000")
	os.Setenv("BACK_END_URL", "http://localhost:8080/")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load to succeed, got %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("expected Port to be 8080, got %v", cfg.Port)
	}
}
