.PHONY: build run test migrate-up migrate-down dev

build:
	go build -o bin/server ./cmd/server

run: build
	./bin/server

test:
	go test ./... -v -race

migrate-up:
	@echo "TODO: migrate-up"

migrate-down:
	@echo "TODO: migrate-down"

dev:
	air
