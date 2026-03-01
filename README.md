# go-leetcode-solutions

Go solutions for LeetCode problems.

## Structure

- `problems/<problem_name>/solution.go` - solution implementation
- `problems/<problem_name>/solution_test.go` - tests for the solution

## Run tests

```bash
go test ./...
```

## PostgreSQL (Docker Compose)

Start PostgreSQL:

```bash
make up
```

If `5432` is already in use, this project maps PostgreSQL to `5433` by default.
You can override:

```bash
make up DB_PORT=55432
```

Stop containers:

```bash
make down
```

Open `psql` shell:

```bash
make db-shell
```

## Run SQL solutions

Run one SQL file:

```bash
make sql-run FILE=sql/second_highest_salary/solution.sql
```

Run all SQL files from a directory (sorted by filename):

```bash
make sql-run-dir DIR=sql/second_highest_salary
```

Direct command:

```bash
go run ./cmd/sql-runner --file sql/second_highest_salary/solution.sql
```

Recommended structure:

```text
models/
  <problem_name>/
    *.go
seeders/
  *.go
sql/
  <problem_name>/
    solution.sql
```

Run one Go seeder:

```bash
make seed PROBLEM=second_highest_salary
```

Run all Go seeders:

```bash
make seed-all
```

## Migrations

Migration files live in `migrations/` and use this format:

- `<version>_<name>.up.sql`
- `<version>_<name>.down.sql`

Commands:

```bash
make migrate-up
make migrate-down
make migrate-status
make migrate-create NAME=add_users_table
```

Direct command examples:

```bash
go run ./cmd/migrate up
go run ./cmd/migrate create --name add_indexes
```
