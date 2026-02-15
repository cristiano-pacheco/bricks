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
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/vektra/mockery/v2@latest
	go install go.uber.org/nilaway/cmd/nilaway@latest

# ==============================================================================
# Running tests within the local computer

.PHONY: static
static: lint vuln-check nilaway

.PHONY: lint
lint:
	golangci-lint run ./... --allow-parallel-runners

.PHONY: vuln-check
vuln-check:
	govulncheck -show verbose ./... 

.PHONY: nilaway
nilaway:
	nilaway --include-pkgs="github.com/cristiano-pacheco/bricks" ./...

.PHONY: test
test:
	CGO_ENABLED=0 go test ./...

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