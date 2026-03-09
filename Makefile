dev:
	docker compose up --build

down:
	docker compose down

run:
	go run cmd/server/main.go


# -----------------------
# Database
# -----------------------

migrate-up:
	docker compose run --rm migrate sh -c 'migrate -path /app/migrations -database "$$DATABASE_URL" up'

migrate-down:
	docker compose run --rm migrate sh -c 'migrate -path /app/migrations -database "$$DATABASE_URL" down'


# -----------------------
# SQLC
# -----------------------

sqlc:
	sqlc generate


# -----------------------
# Docker utilities
# -----------------------

docker-build:
	docker compose build

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f api

docker-restart:
	docker compose restart api


# -----------------------
# Production build
# -----------------------

docker-prod:
	docker build --target production -t unibuzz-api:prod .