.DEFAULT_GOAL := help
.PHONY: help up down logs run test lint

help:
	@echo "make up      start the bot in docker (rebuilds)"
	@echo "make down    stop and remove the container"
	@echo "make logs    follow the container logs"
	@echo "make run     run locally without docker"
	@echo "make test    run the unit tests"
	@echo "make lint    format and vet the code"

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f

run:
	go run ./cmd/goat-bot

test:
	go test ./...

lint:
	go fmt ./...
	go vet ./...
