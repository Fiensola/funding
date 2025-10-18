.PHONY: build run test migrate-up migrate-down docker-up docker-down

build:
	go build -o bin/server cmd/server/main.go

run:
	go run cmd/server/main.go

test:
	go test -v ./...

migrate-up:
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/funding?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/funding?sslmode=disable" down

docker-up:
	docker compose up -d

docker-down:
	docker compose down

frontend-install:
	cd web && npm install

frontend-build:
	cd web && npm run build

frontend-dev:
	cd web && npm start

all: build frontend-build

dev:
	@echo "Starting PostgreSQL..."
	@docker compose up -d
	@echo "Waiting for database..."
	@sleep 1
	@echo "Running migrations..."
	@make migrate-up || true
	@echo "Starting backend in background..."
	@go run cmd/server/main.go &
	@echo "Starting frontend..."
	@cd web && npm run dev