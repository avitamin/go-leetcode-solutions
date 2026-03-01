COMPOSE := docker compose
SERVICE := postgres
DB_PORT ?= 5433
DB_DSN ?= postgres://leetcode:leetcode@localhost:$(DB_PORT)/leetcode?sslmode=disable

.PHONY: help up down restart logs ps db-shell db-reset pgsql sql-run sql-run-dir seed seed-all migrate-up migrate-down migrate-status migrate-create test-migrate-integration

help:
	@echo "Available commands:"
	@echo "  make up       - start PostgreSQL in background (DB_PORT=$(DB_PORT))"
	@echo "  make down     - stop and remove containers"
	@echo "  make restart  - restart PostgreSQL"
	@echo "  make logs     - show PostgreSQL logs"
	@echo "  make ps       - show compose services status"
	@echo "  make db-shell - open psql shell in container"
	@echo "  make pgsql    - alias for db-shell"
	@echo "  make db-reset - remove containers with volumes"
	@echo "  make sql-run FILE=path.sql         - run one SQL file"
	@echo "  make sql-run-dir DIR=sql           - run all SQL files in directory"
	@echo "  make seed PROBLEM=second_highest_salary - run Go seeder for problem"
	@echo "  make seed-all                           - run all Go seeders"
	@echo "  make migrate-up                    - apply all pending migrations"
	@echo "  make migrate-down                  - rollback last migration"
	@echo "  make migrate-status                - show migration status"
	@echo "  make migrate-create NAME=add_users - create migration files"
	@echo "  make test-migrate-integration      - run integration tests for cmd/migrate"

up:
	POSTGRES_PORT=$(DB_PORT) $(COMPOSE) up -d

down:
	$(COMPOSE) down

restart: down up

logs:
	$(COMPOSE) logs -f $(SERVICE)

ps:
	$(COMPOSE) ps

db-shell:
	$(COMPOSE) exec $(SERVICE) psql -U leetcode -d leetcode

pgsql: db-shell

db-reset:
	$(COMPOSE) down -v

sql-run:
	@test -n "$(FILE)" || (echo "Usage: make sql-run FILE=path/to/file.sql" && exit 1)
	go run ./cmd/sql-runner --file "$(FILE)"

sql-run-dir:
	@dir="$${DIR:-sql}"; go run ./cmd/sql-runner --dir "$$dir"

seed:
	@test -n "$(PROBLEM)" || (echo "Usage: make seed PROBLEM=second_highest_salary" && exit 1)
	go run ./cmd/seeder --problem "$(PROBLEM)" --dsn "$(DB_DSN)"

seed-all:
	go run ./cmd/seeder --problem second_highest_salary --dsn "$(DB_DSN)"

migrate-up:
	go run ./cmd/migrate --dsn "$(DB_DSN)" up

migrate-down:
	go run ./cmd/migrate --dsn "$(DB_DSN)" down

migrate-status:
	go run ./cmd/migrate --dsn "$(DB_DSN)" status

migrate-create:
	@test -n "$(NAME)" || (echo "Usage: make migrate-create NAME=add_users_table" && exit 1)
	go run ./cmd/migrate create --name "$(NAME)"

test-migrate-integration:
	MIGRATE_TEST_DSN="$(DB_DSN)" go test -tags=integration -run Integration ./cmd/migrate -v
