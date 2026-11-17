ifeq ($(OS),Windows_NT)
BASH := C:/Program Files/Git/bin/bash.exe
else
BASH := bash
endif

.DEFAULT_GOAL := help
.PHONY: help up down logs backup run test lint

help:
	@echo "make up      start the bot in docker (rebuilds)"
	@echo "make down    stop and remove the container"
	@echo "make logs    follow the container logs"
	@echo "make backup  copy data/state.json into backups/"
	@echo "make run     run locally without docker"
	@echo "make test    run the unit tests"
	@echo "make lint    format and vet the code"

up:
	docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f

backup:
	"$(BASH)" ops/backup.sh

run:
	go run ./cmd/goat-bot

test:
	go test ./...

lint:
	go fmt ./...
	go vet ./...
