package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type config struct {
	composeCmd string
	service    string
	user       string
	dbName     string
}

func main() {
	var (
		filePath = flag.String("file", "", "Path to a single .sql file")
		dirPath  = flag.String("dir", "", "Path to a directory with .sql files")
	)

	cfg := config{}
	flag.StringVar(&cfg.composeCmd, "compose-cmd", "docker compose", "Compose command to use")
	flag.StringVar(&cfg.service, "service", "postgres", "Compose service name")
	flag.StringVar(&cfg.user, "user", "leetcode", "PostgreSQL user")
	flag.StringVar(&cfg.dbName, "db", "leetcode", "PostgreSQL database name")
	flag.Parse()

	if err := run(*filePath, *dirPath, cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(filePath, dirPath string, cfg config) error {
	if filePath == "" && dirPath == "" {
		return errors.New("provide --file or --dir")
	}
	if filePath != "" && dirPath != "" {
		return errors.New("use only one option: --file or --dir")
	}

	if filePath != "" {
		return executeSQLFile(filePath, cfg)
	}

	files, err := collectSQLFiles(dirPath)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no .sql files found in %q", dirPath)
	}

	for _, file := range files {
		if err := executeSQLFile(file, cfg); err != nil {
			return err
		}
	}

	return nil
}

func collectSQLFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".sql") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan sql directory %q: %w", dir, err)
	}

	sort.Strings(files)
	return files, nil
}

func executeSQLFile(filePath string, cfg config) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read %q: %w", filePath, err)
	}

	parts := strings.Fields(cfg.composeCmd)
	if len(parts) == 0 {
		return errors.New("compose command is empty")
	}

	args := append(parts[1:],
		"exec", "-T", cfg.service,
		"psql",
		"-q",
		"-v", "ON_ERROR_STOP=1",
		"-U", cfg.user,
		"-d", cfg.dbName,
		"-f", "-",
	)

	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = bytes.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("Running %s\n", filePath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("execute %q: %w", filePath, err)
	}

	return nil
}
