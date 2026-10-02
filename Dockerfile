FROM golang:1.27-alpine AS dev
RUN go install github.com/air-verse/air@latest
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
# Compila fora de /src para não deixar arquivos de root na pasta do projeto.
CMD ["air", "-c", ".air.toml", "--tmp_dir", "/tmp/air", "--build.cmd", "go build -o /tmp/air/api ./cmd/api", "--build.bin", "/tmp/air/api"]

FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ ./cmd/api ./cmd/worker

FROM gcr.io/distroless/static-debian12:nonroot AS prod
# A mesma imagem roda a API (padrão) e o worker de notificações (entrypoint /worker).
COPY --from=build /out/api /out/worker /
EXPOSE 8080
ENTRYPOINT ["/api"]
