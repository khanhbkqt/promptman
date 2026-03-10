.PHONY: build test lint coverage fmt vet clean

# Build all binaries
build:
	go build ./...

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
