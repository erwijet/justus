.PHONY: run build migration migrate

run:
	go run ./cmd/justus

build:
	go build -o bin/justus ./cmd/justus

migration:
	atlas migrate diff $(name) --env gorm

migrate:
	atlas migrate apply --env gorm --url "$(DATABASE_URL)"
