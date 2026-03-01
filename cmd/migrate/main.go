package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type config struct {
	dsn string
	dir string
}

type migration struct {
	version uint64
	name    string
}

func main() {
	cfg := config{}
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	fs.StringVar(&cfg.dsn, "dsn", "postgres://leetcode:leetcode@localhost:5433/leetcode?sslmode=disable", "PostgreSQL DSN")
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
	m, err := newMigrator(cfg)
	if err != nil {
		return err
	}
	defer closeMigrator(m)

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("No pending migrations")
			return nil
		}
		return fmt.Errorf("migrate up: %w", err)
	}

	fmt.Println("Migrations applied")
	return nil
}

func migrateDown(cfg config) error {
	m, err := newMigrator(cfg)
	if err != nil {
		return err
	}
	defer closeMigrator(m)

	_, _, err = m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			fmt.Println("No applied migrations")
			return nil
		}
		return fmt.Errorf("read current migration version: %w", err)
	}

	if err := m.Steps(-1); err != nil {
		if errors.Is(err, migrate.ErrNoChange) || errors.Is(err, migrate.ErrNilVersion) {
			fmt.Println("No applied migrations")
			return nil
		}
		return fmt.Errorf("migrate down: %w", err)
	}

	fmt.Println("Rolled back last migration")
	return nil
}

func migrateStatus(cfg config) error {
	migrations, err := collectMigrations(cfg.dir)
	if err != nil {
		return err
	}
	if len(migrations) == 0 {
		fmt.Println("No migrations found")
		return nil
	}

	m, err := newMigrator(cfg)
	if err != nil {
		return err
	}
	defer closeMigrator(m)

	current, dirty, err := m.Version()
	var currentVersion uint64
	hasCurrent := false
	if err != nil {
		if !errors.Is(err, migrate.ErrNilVersion) {
			return fmt.Errorf("read migration status: %w", err)
		}
		fmt.Println("Current version: none")
	} else {
		currentVersion = uint64(current)
		hasCurrent = true
		fmt.Printf("Current version: %d", current)
		if dirty {
			fmt.Print(" (dirty)")
		}
		fmt.Println()
	}

	for _, mig := range migrations {
		state := "pending"
		switch {
		case hasCurrent && mig.version < currentVersion:
			state = "applied"
		case hasCurrent && mig.version == currentVersion && dirty:
			state = "dirty"
		case hasCurrent && mig.version == currentVersion:
			state = "applied"
		}
		fmt.Printf("%d_%s %s\n", mig.version, mig.name, state)
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
		version uint64
		name    string
		hasUp   bool
		hasDown bool
	}

	partials := map[uint64]partial{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		var base string
		var isUp, isDown bool
		switch {
		case strings.HasSuffix(name, ".up.sql"):
			base = strings.TrimSuffix(name, ".up.sql")
			isUp = true
		case strings.HasSuffix(name, ".down.sql"):
			base = strings.TrimSuffix(name, ".down.sql")
			isDown = true
		default:
			continue
		}

		version, migName, err := parseMigrationBase(base)
		if err != nil {
			return nil, err
		}

		p := partials[version]
		if p.version == 0 {
			p.version = version
			p.name = migName
		}
		if isUp {
			p.hasUp = true
		}
		if isDown {
			p.hasDown = true
		}
		partials[version] = p
	}

	versions := make([]uint64, 0, len(partials))
	for v := range partials {
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[i] < versions[j]
	})

	out := make([]migration, 0, len(versions))
	for _, v := range versions {
		p := partials[v]
		if !p.hasUp {
			return nil, fmt.Errorf("missing up migration for version %d", v)
		}
		if !p.hasDown {
			return nil, fmt.Errorf("missing down migration for version %d", v)
		}

		out = append(out, migration{version: p.version, name: p.name})
	}

	return out, nil
}

func parseMigrationBase(base string) (uint64, string, error) {
	parts := strings.SplitN(base, "_", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid migration file name %q: expected <version>_<name>.(up|down).sql", base)
	}

	versionRaw := strings.TrimSpace(parts[0])
	name := strings.TrimSpace(parts[1])
	if versionRaw == "" || name == "" {
		return 0, "", fmt.Errorf("invalid migration file name %q: empty version or name", base)
	}

	version, err := strconv.ParseUint(versionRaw, 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("invalid migration file name %q: invalid version %q", base, versionRaw)
	}

	return version, name, nil
}

func newMigrator(cfg config) (*migrate.Migrate, error) {
	if strings.TrimSpace(cfg.dsn) == "" {
		return nil, errors.New("dsn is empty")
	}

	absDir, err := filepath.Abs(cfg.dir)
	if err != nil {
		return nil, fmt.Errorf("resolve migrations dir: %w", err)
	}

	sourceURL := "file://" + filepath.ToSlash(absDir)
	m, err := migrate.New(sourceURL, cfg.dsn)
	if err != nil {
		return nil, fmt.Errorf("create migrator: %w", err)
	}

	return m, nil
}

func closeMigrator(m *migrate.Migrate) {
	sourceErr, dbErr := m.Close()
	if sourceErr != nil || dbErr != nil {
		fmt.Fprintf(os.Stderr, "close migrator: %v\n", errors.Join(sourceErr, dbErr))
	}
}
