package config

import (
	"log"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Port        string `env:"PORT" envDefault:"8080"`
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
