package postgres

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

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

	var db *gorm.DB
	var err error

	maxRetries := 5
	for i := range maxRetries {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err == nil {
			var sqlDB *sql.DB
			sqlDB, err = db.DB()
			if err == nil {
				err = sqlDB.Ping()
				if err == nil {
					break
				}
			}
		}

		slog.Warn("Failed to connect to database, retrying...", slog.Int("attempt", i+1), slog.String("error", err.Error()))
		time.Sleep(time.Second * time.Duration(1<<i))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetConnMaxLifetime(time.Minute * 5)
	sqlDB.SetMaxIdleConns(0)
	sqlDB.SetMaxOpenConns(10)

	slog.Info("Executing database migrations", slog.String("operation", "DatabaseMigration"))
	db.Exec("CREATE EXTENSION IF NOT EXISTS fuzzystrmatch")

	err = db.AutoMigrate(&UserModel{}, &TagModel{}, &ApplicationModel{}, &EventModel{})
	if err != nil {
		return nil, fmt.Errorf("database migration failed: %w", err)
	}

	return db, nil
}
