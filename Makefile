.PHONY: up dev down run test cover lint sqlc

up:        ## Sobe Postgres e API com Docker
	docker compose up --build -d

dev:       ## Sobe Postgres e API com hot reload (air)
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up --build

down:
	docker compose down

run:       ## Roda a API local (precisa do Postgres: docker compose up -d db)
	go run ./cmd/api

test:
	go test -race ./...

cover:
	go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1

lint:
	go vet ./...
	test -z "$$(gofmt -l .)"

sqlc:      ## Regenera internal/store a partir das queries
	sqlc generate
