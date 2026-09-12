.PHONY: help dev build test test-api test-web lint clean docker-up docker-down migrate-up

help:
	@echo "ForgeHub Development Tasks"
	@echo "  dev          - Run API and Web concurrently"
	@echo "  test         - Run all automated tests (Go and TypeScript)"
	@echo "  test-api     - Run Go backend tests"
	@echo "  test-web     - Run Web frontend type check and tests"
	@echo "  build        - Build both backend binary and frontend static bundle"
	@echo "  docker-up    - Start services with Docker Compose"
	@echo "  docker-down  - Stop Docker Compose services"

dev:
	@echo "Starting ForgeHub development services..."
	powershell -ExecutionPolicy Bypass -File scripts/dev.ps1

test-api:
	@echo "Testing backend..."
	cd apps/api && go test -v ./...

test-web:
	@echo "Testing frontend..."
	cd apps/web && npm run build

test: test-api test-web

build:
	@echo "Building API..."
	cd apps/api && go build -o bin/server cmd/server/main.go
	@echo "Building Web..."
	cd apps/web && npm run build

docker-up:
	docker compose up -d

docker-down:
	docker compose down
