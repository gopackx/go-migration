# go-migration Makefile

BINARY_NAME := go-migration
MAIN_PKG    := ./cmd/migrator
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE  := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS     := -s -w -X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE)

.PHONY: build install clean test test-integration

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(MAIN_PKG)

install:
	go install -ldflags "$(LDFLAGS)" $(MAIN_PKG)

clean:
	rm -f $(BINARY_NAME)

test:
	go test ./...

# Integration tests against real databases. SQLite always runs; PostgreSQL and
# MySQL run only when GOMIGRATE_TEST_POSTGRES_DSN / GOMIGRATE_TEST_MYSQL_DSN
# are set. Those databases are wiped by the tests.
test-integration:
	go test -count=1 -tags integration ./tests/integration/...
