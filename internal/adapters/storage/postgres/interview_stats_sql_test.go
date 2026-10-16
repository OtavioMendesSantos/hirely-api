package postgres

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Exercise the production aggregate against PostgreSQL without opening sockets.
// The ordinary repository integration tests additionally cover the GORM path.
func TestInterviewCountsSQLOnPostgres(t *testing.T) {
	initdb, err := exec.LookPath("initdb")
	if err != nil {
		t.Skip("PostgreSQL initdb is not installed")
	}
	postgres, err := exec.LookPath("postgres")
	if err != nil {
		t.Skip("PostgreSQL server binary is not installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	directory := filepath.Join(t.TempDir(), "postgres")
	init := exec.CommandContext(ctx, initdb, "-D", directory, "-A", "trust", "--no-locale", "--encoding=UTF8")
	if output, err := init.CombinedOutput(); err != nil {
		t.Skipf("Cannot initialize isolated PostgreSQL cluster: %v: %s", err, output)
	}
	const owner = "00000000-0000-0000-0000-000000000001"
	const another = "00000000-0000-0000-0000-000000000002"
	applicationID := func(index int) string { return fmt.Sprintf("10000000-0000-0000-0000-%012d", index) }
	var commands []string
	commands = append(commands,
		"CREATE TABLE applications (id UUID PRIMARY KEY,user_id UUID,status TEXT,created_at TIMESTAMP,archived_at TIMESTAMPTZ);",
		"CREATE TABLE events (application_id UUID, previous_status TEXT, new_status TEXT, description TEXT);",
	)
	for i, status := range []string{"REJECTED", "OTHER", "INTERVIEW", "OFFER", "ACCEPTED", "APPLIED", "REJECTED", "TO_APPLY", "TO_APPLY", "TO_APPLY", "REJECTED"} {
		archived := "NULL"
		if i == 10 {
			archived = "'2026-10-16'"
		}
		commands = append(commands, fmt.Sprintf("INSERT INTO applications VALUES ('%s','%s','%s','2026-10-01',%s);", applicationID(i+1), owner, status, archived))
	}
	commands = append(commands, fmt.Sprintf("INSERT INTO applications VALUES ('%s','%s','INTERVIEW','2026-10-01',NULL);", applicationID(12), another))
	addEvent := func(index int, previous, next string) {
		commands = append(commands, fmt.Sprintf("INSERT INTO events VALUES ('%s','%s','%s','Status changed');", applicationID(index), previous, next))
	}
	addEvent(1, "APPLIED", "INTERVIEW")
	addEvent(1, "INTERVIEW", "OFFER")
	addEvent(1, "OFFER", "INTERVIEW")
	addEvent(1, "INTERVIEW", "REJECTED")
	addEvent(2, "INTERVIEW", "OTHER")
	addEvent(9, "APPLIED", "INTERVIEW")
	addEvent(9, "INTERVIEW", "TO_APPLY")
	addEvent(10, "APPLIED", "TO_APPLY")
	addEvent(11, "INTERVIEW", "REJECTED")
	commands = append(commands, fmt.Sprintf("INSERT INTO events VALUES ('%s','','','Interview mentioned in a note');", applicationID(6)))
	aggregate := strings.Replace(interviewCountsSQL, "?", "'"+owner+"'", 1)
	commands = append(commands, "CREATE TEMP TABLE initial AS "+aggregate+";",
		"DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM initial WHERE interview_count=7 AND applied_count=10) THEN RAISE EXCEPTION 'Wrong cumulative count or denominator'; END IF; END $$;",
		"CREATE TEMP TABLE excluded AS "+aggregate+" AND a.created_at > '2026-10-02';",
		"DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM excluded WHERE interview_count=0 AND applied_count=0) THEN RAISE EXCEPTION 'Creation date cohort leaked'; END IF; END $$;",
		fmt.Sprintf("DELETE FROM events WHERE application_id='%s';", applicationID(1)),
		"CREATE TEMP TABLE after_deletion AS "+aggregate+";",
		"DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM after_deletion WHERE interview_count=6 AND applied_count=10) THEN RAISE EXCEPTION 'Deleted history still counted'; END IF; END $$;",
		fmt.Sprintf("DELETE FROM applications WHERE user_id='%s' AND id<>'%s';", owner, applicationID(8)),
		"CREATE TEMP TABLE never_applied AS "+aggregate+";",
		"DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM never_applied WHERE interview_count=0 AND applied_count=0) THEN RAISE EXCEPTION 'Never-applied application was counted'; END IF; END $$;",
	)
	// Single-user mode treats a newline as the end of an input command.
	for i, command := range commands {
		commands[i] = strings.Join(strings.Fields(command), " ")
	}
	run := exec.CommandContext(ctx, postgres, "--single", "-D", directory, "postgres")
	run.Stdin = strings.NewReader(strings.Join(commands, "\n") + "\n")
	output, err := run.CombinedOutput()
	// PostgreSQL single-user mode can continue after SQL errors and exit zero.
	if err != nil || strings.Contains(string(output), "ERROR:") || strings.Contains(string(output), "FATAL:") {
		t.Fatalf("PostgreSQL SQL regression failed: %v\n%s", err, output)
	}
}
