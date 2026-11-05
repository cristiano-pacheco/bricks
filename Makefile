# Check to see if we can use ash, in Alpine images, or default to BASH.
SHELL_PATH = /bin/ash
SHELL = $(if $(wildcard $(SHELL_PATH)),/bin/ash,/bin/bash)

# Load environment variables from .env file if it exists
ifneq (,$(wildcard ./.env))
    include .env
    export
endif

.PHONY: all
all: install-libs lint test cover

# ==============================================================================
# Install dependencies

.PHONY: install-libs
install-libs:
	./scripts/install-tools.sh

# ==============================================================================
# Running tests within the local computer

GO_BIN := $(shell go env GOBIN)
ifeq ($(GO_BIN),)
GO_BIN := $(shell go env GOPATH)/bin
endif

.PHONY: static
static: lint vuln-check nilaway

.PHONY: lint
lint:
	./scripts/install-tools.sh golangci-lint
	$(GO_BIN)/golangci-lint run ./... --allow-parallel-runners

.PHONY: vuln-check
vuln-check:
	./scripts/install-tools.sh govulncheck
	$(GO_BIN)/govulncheck -show verbose ./...

.PHONY: nilaway
nilaway:
	./scripts/install-tools.sh nilaway
	$(GO_BIN)/nilaway --include-pkgs="github.com/cristiano-pacheco/bricks" ./...

.PHONY: test
test:
	CGO_ENABLED=1 go test ./...

.PHONY: check-bricks-local-source
check-bricks-local-source:
	python3 -m unittest discover -s scripts -p 'test_bricks_local_source.py'

.PHONY: unit-test-coverage
unit-test-coverage:
	CGO_ENABLED=1 go test ./... -coverprofile=coverage.txt

# ==============================================================================
# Integration Tests
# All integration test commands are delegated to scripts/integration-test.sh
# which handles Docker socket detection and testcontainers configuration.
#
# The script auto-detects Docker (Colima or Docker Desktop) and sets up
# all required environment variables for testcontainers.
#
# Override Docker configuration by setting environment variables:
#   DOCKER_HOST=unix:///path/to/docker.sock make test-integration
#   TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/path/to/docker.sock make test-integration
#
# For more information, run: ./scripts/integration-test.sh help
# ==============================================================================

.PHONY: test-integration
test-integration:
	@./scripts/integration-test.sh test

.PHONY: test-integration-race
test-integration-race:
	@./scripts/integration-test.sh test-race

.PHONY: test-integration-cover
test-integration-cover:
	@./scripts/integration-test.sh test-cover

.PHONY: cover
cover:
	mkdir -p reports
	go test -race -coverprofile=reports/cover.out -coverpkg=./... ./... && \
	go tool cover -html=reports/cover.out -o reports/cover.html

.PHONY: update-mocks
update-mocks:
	mockery