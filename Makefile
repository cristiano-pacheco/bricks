# Check to see if we can use ash, in Alpine images, or default to BASH.
SHELL_PATH = /bin/ash

.PHONY: all
all: install-libs lint test cover

# ==============================================================================
# Install dependencies

.PHONY: install-libs
install-libs:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/vektra/mockery/v2@latest
	go install github.com/swaggo/swag/cmd/swag@latest
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

.PHONY: cover
cover:
	mkdir -p reports
	go test -race -coverprofile=reports/cover.out -coverpkg=./... ./... && \
	go tool cover -html=reports/cover.out -o reports/cover.html

.PHONY: update-mocks
update-mocks:
	mockery