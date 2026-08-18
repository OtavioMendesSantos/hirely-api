package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"hirely-api/internal/adapters/storage/postgres"
	"hirely-api/internal/config"
	"hirely-api/internal/core/domain"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// 1. Load config and connect to database
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Error loading config:", err)
	}

	if cfg.ENV == "production" || cfg.ENV == "prod" {
		log.Fatal("You cannot run the seed in a production environment!")
	}

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
		log.Fatal("Error connecting to database:", err)
	}

	fmt.Println("Starting Database Seed...")

	// 2. Create a test user
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("12345678"), bcrypt.DefaultCost)

	user := postgres.UserModel{}

	// Use FirstOrCreate with Attrs to prevent duplication if run multiple times
	err = db.Where(postgres.UserModel{Email: "teste@hirely.com"}).
		Attrs(postgres.UserModel{
			ID:           uuid.New().String(),
			Name:         "Test User",
			PasswordHash: string(hashedPassword),
			CreatedAt:    time.Now(),
		}).
		FirstOrCreate(&user).Error

	if err != nil {
		log.Fatal("Error creating user:", err)
	}

	userID := user.ID // Get the real ID (new or existing)

	// 3. Create test tags
	tagData := []postgres.TagModel{
		{Name: "Remote", ColorHex: "#3b82f6"},
		{Name: "Go", ColorHex: "#10b981"},
		{Name: "Senior", ColorHex: "#f59e0b"},
		{Name: "On-site", ColorHex: "#ef4444"},
	}

	tags := make([]postgres.TagModel, len(tagData))
	for i, t := range tagData {
		db.Where(postgres.TagModel{UserID: userID, Name: t.Name}).
			Attrs(postgres.TagModel{
				ID:        uuid.New().String(),
				ColorHex:  t.ColorHex,
				CreatedAt: time.Now(),
			}).
			FirstOrCreate(&tags[i])
	}

	// 4. Create test applications with slight variations to see the similarity query working
	jobTitles := []string{
		"Software Engineer",
		"software engineer",
		"Software Enginer",
		"Backend Developer",
		"Backend Developer",
		"back-end developer",
		"Fullstack",
		"Full Stack Developer",
	}

	companies := []string{"Google", "Amazon", "Microsoft", "Startup X", "Fintech Y", "Consulting Z"}

	statuses := []domain.ApplicationStatus{
		domain.StatusToApply,
		domain.StatusApplied,
		domain.StatusInterview,
		domain.StatusOffer,
		domain.StatusAccepted,
		domain.StatusRejected,
		domain.StatusOther,
	}

	fmt.Println("Generating applications...")
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < 30; i++ {
		title := jobTitles[rand.Intn(len(jobTitles))]
		company := companies[rand.Intn(len(companies))]
		status := statuses[rand.Intn(len(statuses))]

		daysAgo := rand.Intn(60)
		createdAt := time.Now().Add(-time.Duration(daysAgo) * 24 * time.Hour)

		var appliedAt *time.Time
		if status != domain.StatusToApply {
			t := createdAt.Add(time.Hour * 2)
			appliedAt = &t
		}

		app := postgres.ApplicationModel{
			ID:          uuid.New().String(),
			UserID:      userID,
			CompanyName: company,
			JobTitle:    title,
			Status:      status,
			CreatedAt:   createdAt,
			UpdatedAt:   createdAt,
			AppliedAt:   appliedAt,
		}

		if err := db.Create(&app).Error; err != nil {
			log.Printf("Error creating app %d: %v", i, err)
			continue
		}

		// Associate 1 to 2 random tags
		numTags := rand.Intn(2) + 1
		for j := 0; j < numTags; j++ {
			tag := tags[rand.Intn(len(tags))]
			db.Model(&app).Association("Tags").Append(&tag)
		}
	}

	fmt.Println("Seed finished successfully! 🎉")
	fmt.Printf("Test user login:\nEmail: %s\nPassword: 12345678\n", user.Email)
}
