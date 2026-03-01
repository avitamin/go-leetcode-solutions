package seeders

import (
	"context"
	"database/sql"
	"fmt"
)

// SeedSecondHighestSalary fills test data for the LeetCode "Second Highest Salary" problem.
func SeedSecondHighestSalary(ctx context.Context, db *sql.DB) error {
	const rowsCount = 100000

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.ExecContext(ctx, `SET search_path TO "second-highest-salary", public;`); err != nil {
		return fmt.Errorf("set search_path: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `TRUNCATE TABLE "Employee";`); err != nil {
		return fmt.Errorf("truncate Employee: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`
		INSERT INTO "Employee" (id, salary)
		SELECT gs, (random() * 999999 + 1)::int
		FROM generate_series(1, $1) AS gs;
		`,
		rowsCount,
	); err != nil {
		return fmt.Errorf("insert random employees: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
