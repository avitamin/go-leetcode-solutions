package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const migrationTable = "schema_migrations"

type config struct {
	composeCmd string
	service    string
	user       string
	dbName     string
	dir        string
}

type migration struct {
	version  string
	name     string
	upPath   string
	downPath string
}

func main() {
	cfg := config{}
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	fs.StringVar(&cfg.composeCmd, "compose-cmd", "docker compose", "Compose command to use")
	fs.StringVar(&cfg.service, "service", "postgres", "Compose service name")
	fs.StringVar(&cfg.user, "user", "leetcode", "PostgreSQL user")
	fs.StringVar(&cfg.dbName, "db", "leetcode", "PostgreSQL database name")
	fs.StringVar(&cfg.dir, "dir", "migrations", "Migrations directory")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <command>\n\n", filepath.Base(os.Args[0]))
		fmt.Fprintln(os.Stderr, "Commands:")
		fmt.Fprintln(os.Stderr, "  up                apply all pending migrations")
		fmt.Fprintln(os.Stderr, "  down              rollback last applied migration")
		fmt.Fprintln(os.Stderr, "  status            list migrations and their state")
		fmt.Fprintln(os.Stderr, "  create --name X   create pair of migration files")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	args := fs.Args()
	if len(args) == 0 {
		fs.Usage()
		os.Exit(2)
	}

	var err error
	switch args[0] {
	case "up":
		err = migrateUp(cfg)
	case "down":
		err = migrateDown(cfg)
	case "status":
		err = migrateStatus(cfg)
	case "create":
		err = migrateCreate(args[1:], cfg)
	default:
		err = fmt.Errorf("unknown command %q", args[0])
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func migrateUp(cfg config) error {
	migrations, err := collectMigrations(cfg.dir)
	if err != nil {
		return err
	}
	if len(migrations) == 0 {
		fmt.Println("No migrations found")
		return nil
	}

	if err := ensureMigrationTable(cfg); err != nil {
		return err
	}

	applied, err := getAppliedSet(cfg)
	if err != nil {
		return err
	}

	appliedCount := 0
	for _, m := range migrations {
		if applied[m.version] {
			continue
		}
		if m.upPath == "" {
			return fmt.Errorf("missing up migration for version %s", m.version)
		}
		if err := applyUpMigration(m, cfg); err != nil {
			return err
		}
		appliedCount++
	}

	fmt.Printf("Applied %d migration(s)\n", appliedCount)
	return nil
}

func migrateDown(cfg config) error {
	if err := ensureMigrationTable(cfg); err != nil {
		return err
	}

	lastVersion, err := getLastAppliedVersion(cfg)
	if err != nil {
		return err
	}
	if lastVersion == "" {
		fmt.Println("No applied migrations")
		return nil
	}

	migrations, err := collectMigrations(cfg.dir)
	if err != nil {
		return err
	}

	byVersion := make(map[string]migration, len(migrations))
	for _, m := range migrations {
		byVersion[m.version] = m
	}

	m, ok := byVersion[lastVersion]
	if !ok {
		return fmt.Errorf("applied migration %s not found in %q", lastVersion, cfg.dir)
	}
	if m.downPath == "" {
		return fmt.Errorf("missing down migration for version %s", m.version)
	}

	if err := applyDownMigration(m, cfg); err != nil {
		return err
	}

	fmt.Printf("Rolled back %s_%s\n", m.version, m.name)
	return nil
}

func migrateStatus(cfg config) error {
	migrations, err := collectMigrations(cfg.dir)
	if err != nil {
		return err
	}

	if err := ensureMigrationTable(cfg); err != nil {
		return err
	}

	applied, err := getAppliedSet(cfg)
	if err != nil {
		return err
	}

	if len(migrations) == 0 {
		fmt.Println("No migrations found")
		return nil
	}

	for _, m := range migrations {
		state := "pending"
		if applied[m.version] {
			state = "applied"
		}
		fmt.Printf("%s_%s %s\n", m.version, m.name, state)
	}

	return nil
}

func migrateCreate(args []string, cfg config) error {
	createFS := flag.NewFlagSet("create", flag.ExitOnError)
	name := createFS.String("name", "", "Migration name")
	if err := createFS.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(*name) == "" {
		return errors.New("create requires --name")
	}

	if err := os.MkdirAll(cfg.dir, 0o755); err != nil {
		return fmt.Errorf("create migrations dir: %w", err)
	}

	base := fmt.Sprintf("%s_%s", time.Now().UTC().Format("20060102150405"), sanitizeName(*name))
	upPath := filepath.Join(cfg.dir, base+".up.sql")
	downPath := filepath.Join(cfg.dir, base+".down.sql")

	upTemplate := "-- Write migration SQL here\n"
	downTemplate := "-- Write rollback SQL here\n"

	if err := writeNewFile(upPath, upTemplate); err != nil {
		return err
	}
	if err := writeNewFile(downPath, downTemplate); err != nil {
		return err
	}

	fmt.Println(upPath)
	fmt.Println(downPath)
	return nil
}

func writeNewFile(path, content string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create %q: %w", path, err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("write %q: %w", path, err)
	}
	return nil
}

func sanitizeName(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	n = strings.ReplaceAll(n, "-", "_")
	n = strings.ReplaceAll(n, " ", "_")

	re := regexp.MustCompile(`[^a-z0-9_]+`)
	n = re.ReplaceAllString(n, "")

	n = strings.Trim(n, "_")
	if n == "" {
		return "migration"
	}
	return n
}

func collectMigrations(dir string) ([]migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read migrations dir %q: %w", dir, err)
	}

	type partial struct {
		version  string
		name     string
		upPath   string
		downPath string
	}

	partials := map[string]partial{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		path := filepath.Join(dir, name)

		switch {
		case strings.HasSuffix(name, ".up.sql"):
			base := strings.TrimSuffix(name, ".up.sql")
			version, migName, err := parseMigrationBase(base)
			if err != nil {
				return nil, err
			}
			p := partials[version]
			if p.version == "" {
				p.version = version
				p.name = migName
			}
			p.upPath = path
			partials[version] = p
		case strings.HasSuffix(name, ".down.sql"):
			base := strings.TrimSuffix(name, ".down.sql")
			version, migName, err := parseMigrationBase(base)
			if err != nil {
				return nil, err
			}
			p := partials[version]
			if p.version == "" {
				p.version = version
				p.name = migName
			}
			p.downPath = path
			partials[version] = p
		}
	}

	versions := make([]string, 0, len(partials))
	for v := range partials {
		versions = append(versions, v)
	}
	sort.Strings(versions)

	out := make([]migration, 0, len(versions))
	for _, v := range versions {
		p := partials[v]
		out = append(out, migration{
			version:  p.version,
			name:     p.name,
			upPath:   p.upPath,
			downPath: p.downPath,
		})
	}
	return out, nil
}

func parseMigrationBase(base string) (string, string, error) {
	parts := strings.SplitN(base, "_", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid migration file name %q: expected <version>_<name>.(up|down).sql", base)
	}
	version := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])
	if version == "" || name == "" {
		return "", "", fmt.Errorf("invalid migration file name %q: empty version or name", base)
	}
	return version, name, nil
}

func ensureMigrationTable(cfg config) error {
	sql := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	version TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`, migrationTable)
	return execSQL(sql, cfg)
}

func getAppliedSet(cfg config) (map[string]bool, error) {
	out, err := querySQL(fmt.Sprintf("SELECT version FROM %s ORDER BY version;", migrationTable), cfg)
	if err != nil {
		return nil, err
	}

	set := make(map[string]bool)
	for _, line := range strings.Split(out, "\n") {
		v := strings.TrimSpace(line)
		if v != "" {
			set[v] = true
		}
	}
	return set, nil
}

func getLastAppliedVersion(cfg config) (string, error) {
	out, err := querySQL(fmt.Sprintf("SELECT version FROM %s ORDER BY applied_at DESC, version DESC LIMIT 1;", migrationTable), cfg)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func applyUpMigration(m migration, cfg config) error {
	content, err := os.ReadFile(m.upPath)
	if err != nil {
		return fmt.Errorf("read %q: %w", m.upPath, err)
	}

	sql := fmt.Sprintf(
		"BEGIN;\n%s\nINSERT INTO %s(version, name) VALUES ('%s', '%s');\nCOMMIT;\n",
		string(content),
		migrationTable,
		escapeSQL(m.version),
		escapeSQL(m.name),
	)

	fmt.Printf("Applying %s_%s\n", m.version, m.name)
	if err := execSQL(sql, cfg); err != nil {
		return fmt.Errorf("apply %s_%s: %w", m.version, m.name, err)
	}
	return nil
}

func applyDownMigration(m migration, cfg config) error {
	content, err := os.ReadFile(m.downPath)
	if err != nil {
		return fmt.Errorf("read %q: %w", m.downPath, err)
	}

	sql := fmt.Sprintf(
		"BEGIN;\n%s\nDELETE FROM %s WHERE version = '%s';\nCOMMIT;\n",
		string(content),
		migrationTable,
		escapeSQL(m.version),
	)

	fmt.Printf("Rolling back %s_%s\n", m.version, m.name)
	if err := execSQL(sql, cfg); err != nil {
		return fmt.Errorf("rollback %s_%s: %w", m.version, m.name, err)
	}
	return nil
}

func escapeSQL(v string) string {
	return strings.ReplaceAll(v, "'", "''")
}

func execSQL(sql string, cfg config) error {
	cmd, err := buildPsqlCommand(cfg, false)
	if err != nil {
		return err
	}
	cmd.Stdin = strings.NewReader(sql)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func querySQL(sql string, cfg config) (string, error) {
	cmd, err := buildPsqlCommand(cfg, true)
	if err != nil {
		return "", err
	}
	cmd.Stdin = strings.NewReader(sql)
	cmd.Stderr = os.Stderr

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}

func buildPsqlCommand(cfg config, quiet bool) (*exec.Cmd, error) {
	parts := strings.Fields(cfg.composeCmd)
	if len(parts) == 0 {
		return nil, errors.New("compose command is empty")
	}

	args := append(parts[1:],
		"exec", "-T", cfg.service,
		"psql",
		"-v", "ON_ERROR_STOP=1",
		"-U", cfg.user,
		"-d", cfg.dbName,
	)

	if quiet {
		args = append(args, "-q", "-t", "-A")
	}
	args = append(args, "-f", "-")

	return exec.Command(parts[0], args...), nil
}
