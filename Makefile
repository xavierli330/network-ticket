.PHONY: dev-backend dev-frontend build docker-up docker-down

dev-backend:
	cd backend && make dev

dev-frontend:
	@echo "TODO: dev-frontend"

build:
	cd backend && make build
	@echo "TODO: build frontend"

docker-up:
	docker compose up -d

docker-down:
	docker compose down
