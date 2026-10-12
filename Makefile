.PHONY: run build test lint generate install-tools migrate migrate-up migrate-down

BINARY := bin/trip-service
OPENAPI_SPEC := contracts/openapi/trip-service.openapi.yaml

run:
	go run ./cmd/trip-service

build:
	mkdir -p bin
	go build -o $(BINARY) ./cmd/trip-service

test:
	go test ./...

lint:
	golangci-lint run

install-tools:
	go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen

generate:
	go tool oapi-codegen \
		-generate types,chi-server \
		-package api \
		-o api/api.gen.go \
		$(OPENAPI_SPEC)

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate: migrate-up

migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down
