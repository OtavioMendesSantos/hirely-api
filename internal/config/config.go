package config

import (
	"log"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port         string        `env:"PORT" envDefault:"8080"`
	ENV          string        `env:"ENV" envDefault:"development"`
	ReadTimeout  time.Duration `env:"SERVER_READ_TIMEOUT" envDefault:"10s"`
	WriteTimeout time.Duration `env:"SERVER_WRITE_TIMEOUT" envDefault:"20s"`
	IdleTimeout  time.Duration `env:"SERVER_IDLE_TIMEOUT" envDefault:"120s"`
	JWTSecret    string        `env:"JWT_SECRET"`
	JWTExpiresIn time.Duration `env:"JWT_EXPIRES_IN" envDefault:"3d"`

	// Database
	DB_HOST     string `env:"DB_HOST,required"`
	DB_PORT     string `env:"DB_PORT,required"`
	DB_USER     string `env:"DB_USER,required"`
	DB_PASSWORD string `env:"DB_PASSWORD,required"`
	DB_NAME     string `env:"DB_NAME,required"`
	DB_SSLMODE  string `env:"DB_SSLMODE,required"`
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println(".Env file not found")
	}
}

func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
