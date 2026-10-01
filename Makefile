.PHONY: up prod down run test cover lint sqlc

up:        ## Sobe Postgres e API com hot reload (air)
	docker compose up --build

prod:      ## Sobe Postgres e a imagem de produção da API, sem air
	docker compose -f docker-compose.yml up --build -d

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
