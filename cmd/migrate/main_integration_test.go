//go:build integration

package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func TestIntegrationMigrateUpStatusDown(t *testing.T) {
	dsn := integrationDSN(t)
	requirePostgres(t, dsn)

	tableName := fmt.Sprintf("schema_migrations_it_%d", time.Now().UnixNano())
	cfg := config{
		dsn: withMigrationsTable(t, dsn, tableName),
		dir: integrationMigrationsDir(t),
	}

	statusBefore, err := captureStdout(func() error {
		return migrateStatus(cfg)
	})
	if err != nil {
		t.Fatalf("migrateStatus before up: %v", err)
	}
	if !strings.Contains(statusBefore, "Current version: none") {
		t.Fatalf("status before up = %q, want Current version: none", statusBefore)
	}

	if _, err := captureStdout(func() error {
		return migrateUp(cfg)
	}); err != nil {
		t.Fatalf("migrateUp: %v", err)
	}

	version, dirty := currentVersion(t, cfg)
	if version != 2 || dirty {
		t.Fatalf("version after up = (%d, dirty=%v), want (2, false)", version, dirty)
	}

	count := queryInt(t, dsn, "SELECT COUNT(*) FROM integration_migrate_users;")
	if count != 1 {
		t.Fatalf("row count after up = %d, want 1", count)
	}

	statusAfterUp, err := captureStdout(func() error {
		return migrateStatus(cfg)
	})
	if err != nil {
		t.Fatalf("migrateStatus after up: %v", err)
	}
	if !strings.Contains(statusAfterUp, "Current version: 2") {
		t.Fatalf("status after up = %q, want Current version: 2", statusAfterUp)
	}
	if !strings.Contains(statusAfterUp, "1_create_integration_migrate_users applied") {
		t.Fatalf("status after up missing migration 1 applied: %q", statusAfterUp)
	}
	if !strings.Contains(statusAfterUp, "2_seed_integration_migrate_users applied") {
		t.Fatalf("status after up missing migration 2 applied: %q", statusAfterUp)
	}

	if _, err := captureStdout(func() error {
		return migrateDown(cfg)
	}); err != nil {
		t.Fatalf("first migrateDown: %v", err)
	}

	version, dirty = currentVersion(t, cfg)
	if version != 1 || dirty {
		t.Fatalf("version after first down = (%d, dirty=%v), want (1, false)", version, dirty)
	}

	count = queryInt(t, dsn, "SELECT COUNT(*) FROM integration_migrate_users;")
	if count != 0 {
		t.Fatalf("row count after first down = %d, want 0", count)
	}

	if _, err := captureStdout(func() error {
		return migrateDown(cfg)
	}); err != nil {
		t.Fatalf("second migrateDown: %v", err)
	}

	statusAfterAllDown, err := captureStdout(func() error {
		return migrateStatus(cfg)
	})
	if err != nil {
		t.Fatalf("migrateStatus after all down: %v", err)
	}
	if !strings.Contains(statusAfterAllDown, "Current version: none") {
		t.Fatalf("status after all down = %q, want Current version: none", statusAfterAllDown)
	}

	if _, err := captureStdout(func() error {
		return migrateDown(cfg)
	}); err != nil {
		t.Fatalf("third migrateDown (no migrations): %v", err)
	}
}

func integrationDSN(t *testing.T) string {
	t.Helper()

	dsn := os.Getenv("MIGRATE_TEST_DSN")
	if strings.TrimSpace(dsn) == "" {
		dsn = "postgres://leetcode:leetcode@localhost:5433/leetcode?sslmode=disable"
	}
	return dsn
}

func requirePostgres(t *testing.T, dsn string) {
	t.Helper()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("skip integration test: open postgres failed: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Skipf("skip integration test: postgres is unavailable (%v)", err)
	}
}

func withMigrationsTable(t *testing.T, dsn, table string) string {
	t.Helper()

	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse dsn %q: %v", dsn, err)
	}

	q := u.Query()
	q.Set("x-migrations-table", table)
	u.RawQuery = q.Encode()
	return u.String()
}

func integrationMigrationsDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "1_create_integration_migrate_users.up.sql"), `
CREATE TABLE IF NOT EXISTS integration_migrate_users (
	id SERIAL PRIMARY KEY,
	name TEXT NOT NULL
);
`)
	mustWriteFile(t, filepath.Join(dir, "1_create_integration_migrate_users.down.sql"), `
DROP TABLE IF EXISTS integration_migrate_users;
`)
	mustWriteFile(t, filepath.Join(dir, "2_seed_integration_migrate_users.up.sql"), `
INSERT INTO integration_migrate_users(name) VALUES ('alice');
`)
	mustWriteFile(t, filepath.Join(dir, "2_seed_integration_migrate_users.down.sql"), `
DELETE FROM integration_migrate_users WHERE name = 'alice';
`)
	return dir
}

func currentVersion(t *testing.T, cfg config) (uint, bool) {
	t.Helper()

	m, err := newMigrator(cfg)
	if err != nil {
		t.Fatalf("newMigrator: %v", err)
	}
	defer closeMigrator(m)

	version, dirty, err := m.Version()
	if err != nil {
		t.Fatalf("m.Version: %v", err)
	}
	return version, dirty
}

func queryInt(t *testing.T, dsn, query string) int {
	t.Helper()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	var n int
	if err := db.QueryRow(query).Scan(&n); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return n
}

func captureStdout(fn func() error) (string, error) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}

	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	runErr := fn()
	_ = w.Close()
	out, readErr := io.ReadAll(r)
	_ = r.Close()
	if readErr != nil {
		return "", readErr
	}
	return string(out), runErr
}
