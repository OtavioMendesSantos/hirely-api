package postgres

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func NewConnection(cfg DBConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("falha ao conectar ao banco de dados: %w", err)
	}

	// Executa a migração automática das tabelas registradas
	log.Println("Executando migrações do banco de dados...")
	err = db.AutoMigrate(&UserModel{})
	if err != nil {
		return nil, fmt.Errorf("falha ao executar AutoMigrate: %w", err)
	}

	return db, nil
}
