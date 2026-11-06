.PHONY: run build test test-integration lint generate install-tools migrate migrate-up migrate-down

BINARY := bin/trip-service
OPENAPI_SPEC := contracts/openapi/trip-service.openapi.yaml
COVERPROFILE := coverage.out

run:
	go run ./cmd/trip-service

build:
	mkdir -p bin
	go build -o $(BINARY) ./cmd/trip-service

test:
	go test ./... -race
	go test ./internal/usecase/... ./internal/domain/... -race -coverprofile=$(COVERPROFILE) -covermode=atomic
	go tool cover -func=$(COVERPROFILE) | tail -1

test-integration:
	go test ./... -tags=integration -race -count=1

lint:
	golangci-lint run

install-tools:
	go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen
	go get -tool go.uber.org/mock/mockgen

generate:
	go tool oapi-codegen \
		-generate types,chi-server \
		-package api \
		-o internal/generated/api.gen.go \
		$(OPENAPI_SPEC)
	go generate ./...

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate: migrate-up

migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down
