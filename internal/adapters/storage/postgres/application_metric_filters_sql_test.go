package postgres

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"hirely-api/internal/core/domain"
)

// Run the actual GORM filter SQL against PostgreSQL even in environments where
// opening database sockets is prohibited. No application credentials are used.
func TestApplicationMetricListsMatchCounts(t *testing.T) {
	initdb, err := exec.LookPath("initdb")
	if err != nil {
		t.Skip("initdb unavailable")
	}
	postgres, err := exec.LookPath("postgres")
	if err != nil {
		t.Skip("postgres unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	directory := filepath.Join(t.TempDir(), "postgres")
	if output, err := exec.CommandContext(ctx, initdb, "-D", directory, "-A", "trust", "--no-locale", "--encoding=UTF8").CombinedOutput(); err != nil {
		t.Skipf("initdb: %v: %s", err, output)
	}
	db, err := gorm.Open(pgdriver.Open("host=localhost user=test dbname=test sslmode=disable"), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	defer pool.Close()
	const owner = "00000000-0000-0000-0000-000000000001"
	const another = "00000000-0000-0000-0000-000000000002"
	id := func(index int) string { return fmt.Sprintf("10000000-0000-0000-0000-%012d", index) }
	commands := []string{
		"CREATE TABLE applications (id UUID PRIMARY KEY,user_id UUID,status TEXT,company_name TEXT,job_title TEXT,created_at TIMESTAMPTZ,updated_at TIMESTAMPTZ,archived_at TIMESTAMPTZ);",
		"CREATE TABLE events (application_id UUID,previous_status TEXT,new_status TEXT);",
	}
	for i, status := range []string{"REJECTED", "REJECTED", "APPLIED", "APPLIED", "OFFER", "ACCEPTED", "INTERVIEW", "TO_APPLY", "OTHER", "REJECTED", "INTERVIEW", "REJECTED", "INTERVIEW"} {
		user, created, updated, archived := owner, "2026-10-01", "2026-10-01", "NULL"
		if i == 2 {
			updated = "2026-09-01"
		}
		if i == 3 {
			created = "2026-10-16 23:59:59.999"
		}
		if i == 9 {
			archived = "'2026-10-12'"
		}
		if i == 10 {
			user = another
		}
		if i == 11 {
			created = "2025-10-01"
		}
		if i == 12 {
			created = "2026-10-17"
		}
		commands = append(commands, fmt.Sprintf("INSERT INTO applications VALUES('%s','%s','%s','Acme','Engineer','%s','%s',%s);", id(i+1), user, status, created, updated, archived))
	}
	for _, e := range []struct {
		index          int
		previous, next string
	}{{1, "INTERVIEW", "REJECTED"}, {1, "OFFER", "REJECTED"}, {8, "INTERVIEW", "TO_APPLY"}, {9, "OFFER", "OTHER"}, {10, "OFFER", "REJECTED"}} {
		commands = append(commands, fmt.Sprintf("INSERT INTO events VALUES('%s','%s','%s');", id(e.index), e.previous, e.next))
	}
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	base := domain.ApplicationQuery{ArchiveState: "all", StartDate: &start, EndDate: &end, EvaluatedAt: now}
	cases := []struct {
		name, metric, archive string
		statuses              []string
		want                  int
	}{
		{"all", "", "all", nil, 10},
		{"interviews", "interviews", "all", nil, 7},
		{"rejections", "rejections", "all", nil, 3},
		{"advanced", "advanced_rejections", "all", nil, 2},
		{"direct", "direct_rejections", "all", nil, 1},
		{"ghosting", "ghosting", "all", nil, 1},
		{"current", "", "all", []string{"INTERVIEW"}, 1},
		{"rejected_interviews", "interviews", "all", []string{"REJECTED"}, 2},
		{"active_interviews", "interviews", "active", nil, 6},
		{"archived_interviews", "interviews", "archived", nil, 1},
	}
	for _, tc := range cases {
		query := base
		query.Metric = tc.metric
		query.ArchiveState = tc.archive
		query.Statuses = tc.statuses
		countSQL := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
			var count int64
			r := NewApplicationRepository(tx)
			return r.filteredApplications(ctx, owner, query).Count(&count)
		})
		selectSQL := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
			return NewApplicationRepository(tx).filteredApplications(ctx, owner, query).Select("a.id").Find(&[]ApplicationModel{})
		})
		commands = append(commands, fmt.Sprintf("CREATE TEMP TABLE %s_count AS %s;", tc.name, countSQL), fmt.Sprintf("CREATE TEMP TABLE %s_list AS %s;", tc.name, selectSQL), fmt.Sprintf("DO $$ BEGIN IF (SELECT count FROM %s_count)<>%d OR (SELECT count(*) FROM %s_list)<>%d OR (SELECT count(DISTINCT id) FROM %s_list)<>%d THEN RAISE EXCEPTION 'Wrong %s drilldown'; END IF; END $$;", tc.name, tc.want, tc.name, tc.want, tc.name, tc.want, tc.name))
	}
	for i, command := range commands {
		commands[i] = strings.Join(strings.Fields(command), " ")
	}
	run := exec.CommandContext(ctx, postgres, "--single", "-D", directory, "postgres")
	run.Stdin = strings.NewReader(strings.Join(commands, "\n") + "\n")
	output, err := run.CombinedOutput()
	if err != nil || strings.Contains(string(output), "ERROR:") || strings.Contains(string(output), "FATAL:") {
		t.Fatalf("PostgreSQL drilldown regression: %v\n%s", err, output)
	}
}
