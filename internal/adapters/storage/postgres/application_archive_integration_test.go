package postgres

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"hirely-api/internal/core/domain"
	"hirely-api/internal/core/services"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// Integration tests use an explicitly supplied disposable database, never the app's .env.
func archiveTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("HIRELY_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set HIRELY_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	admin := stdlib.OpenDB(*config)
	schema := "archive_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	config.RuntimeParams["search_path"] = schema
	pool := stdlib.OpenDB(*config)
	db, err := gorm.Open(pgdriver.New(pgdriver.Config{Conn: pool}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Error(err)
		}
		admin.Close()
	})
	if err := db.AutoMigrate(&UserModel{}, &UserPreferencesModel{}, &TagModel{}, &ApplicationModel{}, &EventModel{}); err != nil {
		t.Fatal(err)
	}
	if err := migrateApplicationArchiving(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func archiveFixture(t *testing.T, db *gorm.DB) (*ApplicationRepository, *domain.Application) {
	t.Helper()
	user := UserModel{ID: uuid.NewString(), Name: "Test", Email: uuid.NewString() + "@example.com", GoogleID: uuid.NewString(), CreatedAt: time.Now().UTC()}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	app := domain.NewApplication(uuid.NewString(), user.ID, "Acme", "Engineer", domain.StatusRejected)
	app.CreatedAt = time.Now().UTC().Truncate(time.Microsecond).Add(-100 * 24 * time.Hour)
	app.UpdatedAt = app.CreatedAt
	repo := NewApplicationRepository(db)
	if err := repo.Create(context.Background(), app); err != nil {
		t.Fatal(err)
	}
	return repo, app
}

func TestPostgresArchiveAtomicityAndConcurrency(t *testing.T) {
	db := archiveTestDB(t)
	repo, app := archiveFixture(t, db)
	now := time.Now().UTC().Truncate(time.Microsecond)
	var group sync.WaitGroup
	for range 4 {
		group.Go(func() {
			if _, err := repo.SetArchived(context.Background(), app.UserID, app.ID, true, now); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	archived, err := repo.FindByID(context.Background(), app.ID)
	if err != nil || archived.ArchivedAt == nil || len(archived.Events) != 1 || !archived.UpdatedAt.Equal(app.UpdatedAt) || archived.Status != app.Status {
		t.Fatalf("archive: %+v %v", archived, err)
	}
	if err := repo.Update(context.Background(), app); err != domain.ErrApplicationArchived {
		t.Fatalf("stale edit overwrote archive: %v", err)
	}
	restored, err := repo.SetArchived(context.Background(), app.UserID, app.ID, false, now)
	if err != nil || restored.ArchivedAt != nil || len(restored.Events) != 2 || restored.Status != app.Status || !restored.UpdatedAt.Equal(now) {
		t.Fatalf("restore: %+v %v", restored, err)
	}
	if err := db.Exec("ALTER TABLE events ADD CONSTRAINT reject_archive_test CHECK (action <> 'ARCHIVED')").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SetArchived(context.Background(), app.UserID, app.ID, true, now); err == nil {
		t.Fatal("event insert should fail")
	}
	unchanged, _ := repo.FindByID(context.Background(), app.ID)
	if unchanged.ArchivedAt != nil || len(unchanged.Events) != 2 {
		t.Fatal("archive update did not roll back with the event")
	}
}

func TestPostgresArchiveFilteringAndCursor(t *testing.T) {
	db := archiveTestDB(t)
	repo, app := archiveFixture(t, db)
	now := time.Now().UTC().Truncate(time.Microsecond)
	ids := []string{app.ID}
	for range 4 {
		next := *app
		next.ID = uuid.NewString()
		if err := repo.Create(context.Background(), &next); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, next.ID)
	}
	for _, id := range ids {
		if _, err := repo.SetArchived(context.Background(), app.UserID, id, true, now); err != nil {
			t.Fatal(err)
		}
	}
	_, other := archiveFixture(t, db)
	if _, err := repo.SetArchived(context.Background(), other.UserID, other.ID, true, now); err != nil {
		t.Fatal(err)
	}
	service := services.NewApplicationService(repo, nil, nil)
	query := domain.ApplicationQuery{ArchiveState: "archived", Search: "engineer", Statuses: []string{"REJECTED"}, Limit: 2}
	var received []string
	token := ""
	for {
		page, err := service.ListApplicationsPage(context.Background(), app.UserID, query, token)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range page.Applications {
			received = append(received, row.ID)
		}
		if token == "" {
			if _, err := repo.SetArchived(context.Background(), app.UserID, page.Applications[0].ID, false, now); err != nil {
				t.Fatal(err)
			}
		}
		token = page.NextPageToken
		if token == "" {
			break
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ids)))
	if strings.Join(ids, ",") != strings.Join(received, ",") {
		t.Fatalf("missing, duplicate, or cross-account rows: want %v got %v", ids, received)
	}
	stats, err := repo.GetStatsByUserID(context.Background(), app.UserID, nil, nil)
	if err != nil || stats.TotalApplications != 5 {
		t.Fatalf("archiving removed historical metrics: %+v %v", stats, err)
	}
}

func TestPostgresAutomaticArchiveThresholdsAndNever(t *testing.T) {
	db := archiveTestDB(t)
	now := time.Now().UTC().Truncate(time.Microsecond)
	for _, test := range []struct {
		name   string
		days   *int
		age    time.Duration
		status domain.ApplicationStatus
		want   bool
	}{
		{"missing preference defaults to sixty", nil, 60 * 24 * time.Hour, domain.StatusInterview, true},
		{"never", archiveDays(0), 100 * 24 * time.Hour, domain.StatusRejected, false},
		{"thirty exact", archiveDays(30), 30 * 24 * time.Hour, domain.StatusApplied, true},
		{"thirty recent", archiveDays(30), 30*24*time.Hour - time.Second, domain.StatusToApply, false},
		{"sixty exact", archiveDays(60), 60 * 24 * time.Hour, domain.StatusAccepted, true},
		{"ninety recent", archiveDays(90), 89 * 24 * time.Hour, domain.StatusOffer, false},
		{"ninety exact", archiveDays(90), 90 * 24 * time.Hour, domain.StatusOther, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			repo, app := archiveFixture(t, db)
			if err := db.Model(&ApplicationModel{}).Where("id = ?", app.ID).UpdateColumns(map[string]any{"updated_at": now.Add(-test.age), "status": test.status}).Error; err != nil {
				t.Fatal(err)
			}
			if test.days != nil {
				if _, err := NewUserPreferencesRepository(db).Update(context.Background(), app.UserID, domain.UpdateUserPreferences{AutoArchiveDays: test.days}); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := repo.ArchiveDueBatch(context.Background(), now, 100); err != nil {
				t.Fatal(err)
			}
			loaded, err := repo.FindByID(context.Background(), app.ID)
			if err != nil || (loaded.ArchivedAt != nil) != test.want {
				t.Fatalf("archive due: %+v %v", loaded, err)
			}
			if loaded.Status != test.status {
				t.Fatal("archiver changed status")
			}
			if test.want {
				if len(loaded.Events) != 1 || loaded.Events[0].Action != "ARCHIVED" {
					t.Fatal("missing automatic archive event")
				}
				restored, err := repo.SetArchived(context.Background(), app.UserID, app.ID, false, now)
				if err != nil || !restored.UpdatedAt.Equal(now) {
					t.Fatalf("restore did not reset activity: %+v %v", restored, err)
				}
				if _, err := repo.ArchiveDueBatch(context.Background(), now, 100); err != nil {
					t.Fatal(err)
				}
				fresh, _ := repo.FindByID(context.Background(), app.ID)
				if fresh.ArchivedAt != nil || len(fresh.Events) != 2 {
					t.Fatal("restored application was immediately archived again")
				}
			}
			// Isolate each subcase from subsequent global batches.
			if err := db.Where("id = ?", app.UserID).Delete(&UserModel{}).Error; err != nil {
				t.Fatal(err)
			}
		})
	}
}

func archiveDays(days int) *int { return &days }

func TestPostgresPreferenceUpsertPreservesNeverAndOtherFields(t *testing.T) {
	db := archiveTestDB(t)
	_, app := archiveFixture(t, db)
	repo := NewUserPreferencesRepository(db)
	disabled := false
	preferences, err := repo.Update(context.Background(), app.UserID, domain.UpdateUserPreferences{AutoArchiveDays: archiveDays(0), ConfettiEnabled: &disabled})
	if err != nil || preferences.AutoArchiveDays != 0 || preferences.ConfettiEnabled {
		t.Fatalf("explicit zero/false lost: %+v %v", preferences, err)
	}
	dark := "dark"
	preferences, err = repo.Update(context.Background(), app.UserID, domain.UpdateUserPreferences{Theme: &dark})
	if err != nil || preferences.AutoArchiveDays != 0 || preferences.ConfettiEnabled || preferences.Theme != "dark" {
		t.Fatalf("omitted fields overwritten: %+v %v", preferences, err)
	}
}
