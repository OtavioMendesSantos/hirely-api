// Populate the local demo account. Safe to rerun.
package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"hirely-api/internal/adapters/storage/postgres"
	"hirely-api/internal/config"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/security"

	"github.com/google/uuid"
	pg "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

const email = "portfolio@hirely.example"
const password = "HirelyDemo2026!"

func id(name string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("hirely-portfolio/"+name)).String()
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if strings.TrimSpace(cfg.ENV) != "development" || (cfg.DB_HOST != "localhost" && cfg.DB_HOST != "127.0.0.1" && cfg.DB_HOST != "::1") {
		log.Fatal("Seed requires ENV=development and a loopback database host")
	}
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC", cfg.DB_HOST, cfg.DB_PORT, cfg.DB_USER, cfg.DB_PASSWORD, cfg.DB_NAME, cfg.DB_SSLMODE)
	db, err := gorm.Open(pg.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		log.Fatal(err)
	}
	hash, err := security.HashPassword(password, cfg.AuthPepper)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		uid := id("user")
		var existing postgres.UserModel
		if err := tx.Where("email = ?", email).First(&existing).Error; err != nil && err != gorm.ErrRecordNotFound {
			return err
		}
		if existing.ID != "" && existing.ID != uid {
			return fmt.Errorf("demo email already belongs to a different account")
		}
		now := time.Now().UTC().Truncate(24 * time.Hour).Add(12 * time.Hour)
		user := postgres.UserModel{ID: uid, Name: "Alex Morgan", Email: email, PasswordHash: string(hash), CreatedAt: now.AddDate(0, -4, 0)}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoUpdates: clause.AssignmentColumns([]string{"name", "password_hash"})}).Omit("GoogleID").Create(&user).Error; err != nil {
			return err
		}
		// Reset only records owned by the fixed demo user, inside this transaction.
		for _, query := range []string{
			"DELETE FROM application_tags WHERE application_id IN (SELECT id FROM applications WHERE user_id = ?)",
			"DELETE FROM events WHERE application_id IN (SELECT id FROM applications WHERE user_id = ?)",
			"DELETE FROM applications WHERE user_id = ?",
			"DELETE FROM tags WHERE user_id = ?",
		} {
			if err := tx.Exec(query, uid).Error; err != nil {
				return err
			}
		}
		prefs := postgres.UserPreferencesModel{UserID: uid, Theme: "dark", AutoArchiveDays: 0, ConfettiEnabled: false, CreatedAt: now, UpdatedAt: now}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoUpdates: clause.Assignments(map[string]any{"theme": "dark", "auto_archive_days": 0, "confetti_enabled": false})}).Create(&prefs).Error; err != nil {
			return err
		}
		if err := tx.Model(&prefs).Updates(map[string]any{"auto_archive_days": 0, "confetti_enabled": false}).Error; err != nil {
			return err
		}
		// Display-only integration examples. These non-hex sentinels cannot match
		// the SHA-256 hex hashes used by API authentication; no usable token exists.
		lastUsed, loopback := now.Add(-2*time.Hour), "127.0.0.1"
		keys := []postgres.APIKeyModel{
			{ID: id("key/claude"), UserID: uid, Name: "Claude Desktop · Personal workspace", KeyHash: "portfolio-demo:no-token:claude", UsageCount: 42, LastUsedAt: &lastUsed, LastIP: &loopback, CreatedAt: now.AddDate(0, 0, -12)},
			{ID: id("key/cursor"), UserID: uid, Name: "Cursor · Previous setup", KeyHash: "portfolio-demo:no-token:cursor", UsageCount: 18, Revoked: true, CreatedAt: now.AddDate(0, 0, -24)},
		}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, UpdateAll: true}).Create(&keys).Error; err != nil {
			return err
		}
		tags := []postgres.TagModel{
			{ID: id("tag/go"), UserID: uid, Name: "Go", ColorHex: "#38BDF8", CreatedAt: now},
			{ID: id("tag/typescript"), UserID: uid, Name: "TypeScript", ColorHex: "#818CF8", CreatedAt: now},
			{ID: id("tag/cloud"), UserID: uid, Name: "Cloud & DevOps", ColorHex: "#A78BFA", CreatedAt: now},
			{ID: id("tag/priority"), UserID: uid, Name: "Top choice", ColorHex: "#FBBF24", CreatedAt: now},
			{ID: id("tag/product"), UserID: uid, Name: "Product team", ColorHex: "#34D399", CreatedAt: now},
		}
		if err := tx.Create(&tags).Error; err != nil {
			return err
		}
		companies := []string{"Northstar Labs", "Atlas Cloud", "Orbit Studio", "Meridian Tech", "Evergreen Digital", "Nova Finance", "Lumina Health", "Harbor Systems", "Vertex Data", "Bloom Software", "Summit Platform", "Aurora Commerce", "Cedar Analytics", "Prism Network", "Horizon Labs", "Pulse Digital", "Mosaic Cloud", "Solstice Tech", "Acorn Studio", "Beacon Systems", "Lumen Works", "Meadow Data", "Ripple Platform", "Cobalt Digital", "Arc Technologies", "Fern Studio", "Ember Cloud", "Tidal Software"}
		roles := []string{"Backend Engineer", "Full Stack Developer", "Platform Engineer", "Software Engineer", "Frontend Engineer", "Cloud Engineer", "Go Developer"}
		statuses := []domain.ApplicationStatus{domain.StatusToApply, domain.StatusToApply, domain.StatusToApply, domain.StatusApplied, domain.StatusApplied, domain.StatusApplied, domain.StatusApplied, domain.StatusApplied, domain.StatusInterview, domain.StatusInterview, domain.StatusInterview, domain.StatusInterview, domain.StatusOffer, domain.StatusOffer, domain.StatusAccepted, domain.StatusAccepted, domain.StatusRejected, domain.StatusRejected, domain.StatusRejected, domain.StatusRejected, domain.StatusRejected, domain.StatusRejected, domain.StatusOther, domain.StatusOther, domain.StatusApplied, domain.StatusInterview, domain.StatusRejected, domain.StatusApplied}
		for i, company := range companies {
			created := now.AddDate(0, 0, -(i*3 + 3))
			updated := now.AddDate(0, 0, -(i % 9))
			contract, modality := "CLT", "REMOTE"
			if i%4 == 1 {
				contract = "PJ"
				modality = "HYBRID"
			}
			app := postgres.ApplicationModel{ID: id("application/" + company), UserID: uid, CompanyName: company, JobTitle: roles[i%len(roles)], JobURL: "https://example.com/careers", SalaryRange: []string{"R$ 10,000 – R$ 14,000", "R$ 12,000 – R$ 16,000", "R$ 14,000 – R$ 18,000"}[i%3], Status: statuses[i], ContractType: &contract, WorkModality: &modality, CreatedAt: created, UpdatedAt: updated, SubmittedDocuments: []string{"Resume", "Portfolio"}, JobDescription: "## Build products that matter\nJoin a product team building reliable APIs and thoughtful developer experiences.\n\n**Stack:** Go, TypeScript, PostgreSQL and cloud infrastructure.\n\n**Focus:** API design, observability and continuous delivery.", Notes: "## Interview preparation\n- Share the API performance case study\n- Discuss team autonomy and the product roadmap"}
			if app.Status != domain.StatusToApply {
				applied := created.Add(2 * time.Hour)
				app.AppliedAt = &applied
			}
			if i == 23 || i == 26 {
				archived := now.AddDate(0, 0, -2)
				app.ArchivedAt = &archived
			}
			if err := tx.Create(&app).Error; err != nil {
				return err
			}
			if err := tx.Model(&app).Association("Tags").Append(&tags[i%3], &tags[3+i%2]); err != nil {
				return err
			}
			// Association writes touch UpdatedAt; retain the curated activity dates.
			if err := tx.Model(&app).UpdateColumn("updated_at", updated).Error; err != nil {
				return err
			}
			events := []postgres.EventModel{{ID: id(company + "/created"), ApplicationID: app.ID, Type: "AUTOMATIC", Description: "Application created", NewStatus: "TO_APPLY", CreatedAt: created}}
			if app.AppliedAt != nil {
				events = append(events, postgres.EventModel{ID: id(company + "/applied"), ApplicationID: app.ID, Type: "AUTOMATIC", Description: "Application submitted", PreviousStatus: "TO_APPLY", NewStatus: "APPLIED", CreatedAt: *app.AppliedAt})
			}
			if app.Status == domain.StatusInterview || app.Status == domain.StatusOffer || app.Status == domain.StatusAccepted {
				events = append(events, postgres.EventModel{ID: id(company + "/interview"), ApplicationID: app.ID, Type: "AUTOMATIC", Description: "Moved to Interview", PreviousStatus: "APPLIED", NewStatus: "INTERVIEW", CreatedAt: updated.Add(-48 * time.Hour)}, postgres.EventModel{ID: id(company + "/note"), ApplicationID: app.ID, Type: "MANUAL", Description: "Great conversation with the engineering team. Discussed API architecture, developer experience and the product roadmap. Next step: technical discussion with the team lead.", CreatedAt: updated.Add(-24 * time.Hour)})
			}
			if app.Status != domain.StatusToApply && app.Status != domain.StatusApplied && app.Status != domain.StatusInterview {
				events = append(events, postgres.EventModel{ID: id(company + "/status"), ApplicationID: app.ID, Type: "AUTOMATIC", Description: "Application status updated", PreviousStatus: "INTERVIEW", NewStatus: string(app.Status), CreatedAt: updated})
			}
			if err := tx.Create(&events).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Demo account ready: %s\nLocal demo password: %s\n28 applications, 5 tags, 2 display-only API key examples; other accounts unchanged.\n", email, password)
}
