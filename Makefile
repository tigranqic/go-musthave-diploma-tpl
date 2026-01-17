SERVER_DIR=cmd/gophermart

SERVER_BIN=$(SERVER_DIR)/gophermart

MIGRATIONS_DIR=./migrations
GOOSE_BIN=goose
PG_DSN=${DATABASE_DSN}

.PHONY: all build test clean fmt vet lint help

all: build test

build-server:
	go build -o $(SERVER_BIN) $(SERVER_DIR)/*.go

build: build-server

test:

clean:
	rm -f $(SERVER_BIN) $(AGENT_BIN)

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run

help:
	@echo "Makefile commands:"
	@echo "  make build           - Build server"
	@echo "  make test            - Run autotests"
	@echo "  make clean           - Remove binaries"
	@echo "  make fmt             - Format code"
	@echo "  make vet             - Run 'go vet'"
	@echo "  make lint            - Run golangci-lint (optional)"

run-server:
	$(SERVER_BIN) -a=localhost:8081

migrate-new:
	@echo "Creating new migration: $(name)"
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) create $(name) sql
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) fix

migrate-up:
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) postgres "$(PG_DSN)" up

migrate-down:
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) postgres "$(PG_DSN)" down

migrate-reset:
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) postgres "$(PG_DSN)" reset

migrate-fix:
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) fix

migrate-status:
	$(GOOSE_BIN) -dir $(MIGRATIONS_DIR) postgres "$(PG_DSN)" status
