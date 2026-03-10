.PHONY: build build-cli test lint coverage fmt vet clean

# Version injection variables
VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/khanhnguyen/promptman/internal/cli.Version=$(VERSION) \
           -X github.com/khanhnguyen/promptman/internal/cli.Commit=$(COMMIT) \
           -X github.com/khanhnguyen/promptman/internal/cli.Date=$(DATE)

# Build all binaries
build:
	go build ./...

# Build CLI binary with version info
build-cli:
	go build -ldflags "$(LDFLAGS)" -o bin/promptman ./cmd/cli

# Build daemon binary
build-daemon:
	go build -o bin/promptman-daemon ./cmd/daemon

# Build all binaries
build-all: build-cli build-daemon


# Run all tests
test:
	go test ./... -v

# Run tests with race detector
test-race:
	go test ./... -race -v

# Run linter
lint:
	golangci-lint run ./...

# Run tests with coverage
coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out
	@echo ""
	@echo "To view HTML report: go tool cover -html=coverage.out"

# Format code
fmt:
	gofmt -s -w .
	goimports -w .

# Run go vet
vet:
	go vet ./...

# Clean build artifacts
clean:
	rm -f coverage.out
	go clean ./...

# Run all checks (build + vet + lint + test)
check: build vet lint test
