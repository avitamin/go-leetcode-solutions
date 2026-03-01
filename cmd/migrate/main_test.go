package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestSanitizeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "simple", in: "add_users", want: "add_users"},
		{name: "spaces and dashes", in: " Add Users-Table ", want: "add_users_table"},
		{name: "special chars", in: "a@b#c$", want: "abc"},
		{name: "empty fallback", in: "---", want: "migration"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := sanitizeName(tc.in)
			if got != tc.want {
				t.Fatalf("sanitizeName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseMigrationBase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		wantVer uint64
		wantNam string
		wantErr string
	}{
		{name: "valid", in: "20260301191000_second_highest_salary_schema", wantVer: 20260301191000, wantNam: "second_highest_salary_schema"},
		{name: "missing underscore", in: "20260301191000", wantErr: "expected <version>_<name>"},
		{name: "invalid version", in: "abc_name", wantErr: "invalid version"},
		{name: "empty name", in: "123_", wantErr: "empty version or name"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ver, name, err := parseMigrationBase(tc.in)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("parseMigrationBase(%q) expected error %q, got nil", tc.in, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("parseMigrationBase(%q) error = %q, want contains %q", tc.in, err.Error(), tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseMigrationBase(%q) unexpected error: %v", tc.in, err)
			}
			if ver != tc.wantVer || name != tc.wantNam {
				t.Fatalf("parseMigrationBase(%q) = (%d, %q), want (%d, %q)", tc.in, ver, name, tc.wantVer, tc.wantNam)
			}
		})
	}
}

func TestCollectMigrationsNotExists(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "missing")
	got, err := collectMigrations(dir)
	if err != nil {
		t.Fatalf("collectMigrations(%q) unexpected error: %v", dir, err)
	}
	if len(got) != 0 {
		t.Fatalf("collectMigrations(%q) len = %d, want 0", dir, len(got))
	}
}

func TestCollectMigrationsSortedAndComplete(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "2_second.up.sql"), "-- up")
	mustWriteFile(t, filepath.Join(dir, "2_second.down.sql"), "-- down")
	mustWriteFile(t, filepath.Join(dir, "1_first.up.sql"), "-- up")
	mustWriteFile(t, filepath.Join(dir, "1_first.down.sql"), "-- down")
	mustWriteFile(t, filepath.Join(dir, "README.txt"), "ignore")

	got, err := collectMigrations(dir)
	if err != nil {
		t.Fatalf("collectMigrations(%q) unexpected error: %v", dir, err)
	}

	if len(got) != 2 {
		t.Fatalf("collectMigrations(%q) len = %d, want 2", dir, len(got))
	}

	if got[0].version != 1 || got[0].name != "first" {
		t.Fatalf("first migration = %+v, want version=1 name=first", got[0])
	}
	if got[1].version != 2 || got[1].name != "second" {
		t.Fatalf("second migration = %+v, want version=2 name=second", got[1])
	}
}

func TestCollectMigrationsMissingPair(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "1_first.up.sql"), "-- up")

	_, err := collectMigrations(dir)
	if err == nil {
		t.Fatalf("collectMigrations(%q) expected error, got nil", dir)
	}
	if !strings.Contains(err.Error(), "missing down migration for version 1") {
		t.Fatalf("collectMigrations(%q) error = %q, want missing down", dir, err.Error())
	}
}

func TestCollectMigrationsInvalidFileName(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "invalid.up.sql"), "-- up")

	_, err := collectMigrations(dir)
	if err == nil {
		t.Fatalf("collectMigrations(%q) expected error, got nil", dir)
	}
	if !strings.Contains(err.Error(), "expected <version>_<name>") {
		t.Fatalf("collectMigrations(%q) error = %q, want parse error", dir, err.Error())
	}
}

func TestMigrateCreateRequiresName(t *testing.T) {
	t.Parallel()

	err := migrateCreate([]string{}, config{dir: t.TempDir()})
	if err == nil {
		t.Fatal("migrateCreate expected error, got nil")
	}
	if !strings.Contains(err.Error(), "create requires --name") {
		t.Fatalf("migrateCreate error = %q, want create requires --name", err.Error())
	}
}

func TestMigrateCreateCreatesFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	err := migrateCreate([]string{"--name", "Add Users-Table!"}, config{dir: dir})
	if err != nil {
		t.Fatalf("migrateCreate unexpected error: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q) unexpected error: %v", dir, err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries len = %d, want 2", len(entries))
	}

	pattern := regexp.MustCompile(`^\d{14}_add_users_table\.(up|down)\.sql$`)
	seenUp := false
	seenDown := false
	for _, e := range entries {
		if !pattern.MatchString(e.Name()) {
			t.Fatalf("unexpected migration file name: %s", e.Name())
		}

		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("ReadFile(%q) unexpected error: %v", e.Name(), err)
		}

		s := string(content)
		if strings.HasSuffix(e.Name(), ".up.sql") {
			seenUp = true
			if s != "-- Write migration SQL here\n" {
				t.Fatalf("up file content = %q, want template", s)
			}
		}
		if strings.HasSuffix(e.Name(), ".down.sql") {
			seenDown = true
			if s != "-- Write rollback SQL here\n" {
				t.Fatalf("down file content = %q, want template", s)
			}
		}
	}

	if !seenUp || !seenDown {
		t.Fatalf("seenUp=%v seenDown=%v, want both true", seenUp, seenDown)
	}
}

func TestWriteNewFileDoesNotOverwrite(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "file.sql")
	mustWriteFile(t, path, "first")

	err := writeNewFile(path, "second")
	if err == nil {
		t.Fatalf("writeNewFile(%q) expected error, got nil", path)
	}
	if !strings.Contains(err.Error(), "create") {
		t.Fatalf("writeNewFile(%q) error = %q, want create error", path, err.Error())
	}
}

func TestNewMigratorRequiresDSN(t *testing.T) {
	t.Parallel()

	_, err := newMigrator(config{dsn: "", dir: t.TempDir()})
	if err == nil {
		t.Fatal("newMigrator expected error, got nil")
	}
	if !strings.Contains(err.Error(), "dsn is empty") {
		t.Fatalf("newMigrator error = %q, want dsn is empty", err.Error())
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) unexpected error: %v", path, err)
	}
}
