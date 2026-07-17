package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_JWTExpiresInDefault(t *testing.T) {
	// Ensure required env vars are set so Load() doesn't fail
	os.Setenv("JWT_SECRET", "testsecret")
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "postgres")
	os.Setenv("DB_PASSWORD", "root")
	os.Setenv("DB_NAME", "hirely")
	os.Setenv("DB_SSLMODE", "disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load to succeed, got %v", err)
	}

	if cfg.JWTExpiresIn != 24*time.Hour {
		t.Errorf("expected JWTExpiresIn to be 24h, got %v", cfg.JWTExpiresIn)
	}
}
