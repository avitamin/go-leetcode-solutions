package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/avitamin/go-leetcode-solutions/internal/seeders"
	_ "github.com/lib/pq"
)

const defaultDSN = "postgres://leetcode:leetcode@localhost:5432/leetcode?sslmode=disable"

func main() {
	problem := flag.String("problem", "", "Problem key, e.g. second_highest_salary")
	dsn := flag.String("dsn", defaultDSN, "PostgreSQL DSN")
	flag.Parse()

	if err := run(*problem, *dsn); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(problem, dsn string) error {
	if problem == "" {
		return errors.New("provide --problem")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping db: %w", err)
	}

	switch problem {
	case "second_highest_salary":
		if err := seeders.SeedSecondHighestSalary(ctx, db); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown problem %q", problem)
	}

	fmt.Printf("Seeder completed for %s\n", problem)
	return nil
}
