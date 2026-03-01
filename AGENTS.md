# Repository Guidelines

## Project Structure & Module Organization
This repository combines Go problem code and SQL-backed LeetCode workflows.

- `cmd/`: CLI entry points (`migrate`, `seeder`, `sql-runner`).
- `internal/`: application internals (seeders and typed models used by CLIs).
- `sql/<problem_name>/`: SQL solutions and setup/test data files.
- `migrations/`: schema migration files named `<version>_<name>.up.sql` and `.down.sql`.
- `problems/<problem_name>/`: Go problem solutions and tests (`solution.go`, `solution_test.go`).
- `models/` and `seeders/`: per-problem artifacts mirroring SQL problems.

Prefer adding new problem assets under matching `<problem_name>` directories for consistency.

## Build, Test, and Development Commands
- `go test ./...`: run all Go tests.
- `make up` / `make down`: start/stop PostgreSQL via Docker Compose (`DB_PORT` defaults to `5433`).
- `make db-shell`: open `psql` inside the running container.
- `make sql-run FILE=sql/second_highest_salary/solution.sql`: execute one SQL file.
- `make sql-run-dir DIR=sql/second_highest_salary`: execute all SQL files in a directory.
- `make seed PROBLEM=second_highest_salary`: run one Go seeder.
- `make migrate-up`, `make migrate-down`, `make migrate-status`: manage schema migrations.
- `make migrate-create NAME=add_indexes`: scaffold migration files.

## Coding Style & Naming Conventions
Use standard Go style and keep code `gofmt`-formatted before opening a PR (`gofmt -w ./...`).

- Package/file names: lowercase, short, descriptive.
- Problem folders: snake_case (example: `second_highest_salary`).
- Keep CLI flags and Make variables explicit (`PROBLEM=...`, `FILE=...`, `NAME=...`).
- SQL files should be deterministic and runnable in order when used with `sql-run-dir`.

## Testing Guidelines
Use Go’s `testing` package with table-driven tests where practical.

- Test files must end with `_test.go`.
- For Go problems, colocate tests with solutions under `problems/<problem_name>/`.
- Validate SQL changes by running migrations, seeding, and executing the target SQL solution.

## Commit & Pull Request Guidelines
Recent commits use concise, imperative summaries (for example, `Initialize Go project...`).

- Keep commit subjects short and action-oriented; expand rationale in the body when needed.
- PRs should include: purpose, scope, commands run (`go test ./...`, relevant `make` targets), and migration impact.
- Link related issues/tasks and include sample query output when SQL behavior changes.
