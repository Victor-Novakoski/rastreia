.PHONY: up down run test test-short cover lint sqlc

up:        ## Sobe Postgres e API com Docker (com hot reload)
	docker compose up --build -d

down:
	docker compose down

run:       ## Roda a API local (precisa do Postgres: docker compose up -d db)
	go run ./cmd/api

test:      ## Unitários + integração (precisa do Docker)
	go test -race ./...

test-short: ## Só os unitários
	go test -race -short ./...

cover:
	go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1

lint:
	go vet ./...
	test -z "$$(gofmt -l .)"

sqlc:      ## Regenera internal/store a partir das queries (usa a imagem oficial, sem instalar nada)
	docker run --rm -u $$(id -u):$$(id -g) -v "$$PWD":/src -w /src sqlc/sqlc:1.31.1 generate
