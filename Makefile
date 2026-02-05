SERVER_DIR=cmd/gophermart
ACCRUAL_DIR=cmd/accrual
SERVER_BIN=$(SERVER_DIR)/gophermart
ACCRUAL_BIN=$(ACCRUAL_DIR)/accrual_darwin_arm64
MIGRATIONS_DIR=./migrations
GOOSE_BIN=goose
PG_DSN=${DATABASE_DSN}
ACCRUAL_PG_DSN=${ACCRUAL_DB_DSN}

GOPHERMARTTEST_BIN=./gophermarttest-darwin-arm64
GOPHERMART_HOST=localhost
GOPHERMART_PORT=8081
ACCRUAL_HOST=localhost
ACCRUAL_PORT=8080

STATICTEST_BIN=./statictest-darwin-arm64

.PHONY: all build test clean fmt vet lint help download-statictest autotest

download-statictest:
	@if [ ! -f $(STATICTEST_BIN) ]; then \
		echo "statictest binary not found. Downloading..."; \
		@mkdir -p .tools; \
		curl -sSL https://github.com/Yandex-Practicum/go-autotests/releases/latest/download/statictest-darwin-arm64 -o $(STATICTEST_BIN); \
		chmod +x $(STATICTEST_BIN); \
	else \
		echo "Using local statictest binary..."; \
	fi

vet: download-statictest
	@echo "Running go vet with statictest..."
	go vet -vettool=$(STATICTEST_BIN) ./...

all: build test

build-server:
	go build -o $(SERVER_BIN) $(SERVER_DIR)/*.go

build: build-server

autotest: build
	@echo "Running local autotests..."
	$(GOPHERMARTTEST_BIN) \
		-test.v \
		-test.run=^TestGophermart$$ \
		-gophermart-binary-path=$(SERVER_BIN) \
		-gophermart-host=$(GOPHERMART_HOST) \
		-gophermart-port=$(GOPHERMART_PORT) \
		-gophermart-database-uri="$(PG_DSN)" \
		-accrual-binary-path=$(ACCRUAL_BIN) \
		-accrual-host=$(ACCRUAL_HOST) \
		-accrual-port=$(ACCRUAL_PORT) \
		-accrual-database-uri="$(ACCRUAL_PG_DSN)"

test: autotest

clean:
	rm -f $(SERVER_BIN)

fmt:
	go fmt ./...

lint:
	golangci-lint run

help:
	@echo "Makefile commands:"
	@echo "  make build           - Build server"
	@echo "  make test            - Run autotests"
	@echo "  make clean           - Remove binaries"
	@echo "  make fmt             - Format code"
	@echo "  make vet             - Run 'go vet' with statictest"
	@echo "  make lint            - Run golangci-lint (optional)"

run-server:
	$(SERVER_BIN) -a=localhost:8081 -r=http://localhost:8080

run-accrual:
	$(ACCRUAL_BIN)

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
