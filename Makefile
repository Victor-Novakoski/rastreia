.PHONY: up down run test cover lint sqlc

up:        ## Sobe Postgres e API com Docker (com hot reload)
	docker compose up --build -d

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
