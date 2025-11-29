.PHONY: help build install clean test test-coverage test-unit lint fmt vet run build-all docker-build

# Variables
BINARY_NAME=azure-pr-cli
GO=go
GOTEST=$(GO) test
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean
GOGET=$(GO) get
GOMOD=$(GO) mod
GOFMT=$(GO) fmt
GOVET=$(GO) vet

# Version information
VERSION?=dev
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS=-ldflags "-X github.com/yourusername/azure-pr-cli/cmd.Version=$(VERSION) \
                  -X github.com/yourusername/azure-pr-cli/cmd.GitCommit=$(GIT_COMMIT) \
                  -X github.com/yourusername/azure-pr-cli/cmd.BuildDate=$(BUILD_DATE)"

# Colors for output
COLOR_RESET=\033[0m
COLOR_BOLD=\033[1m
COLOR_GREEN=\033[32m
COLOR_YELLOW=\033[33m
COLOR_BLUE=\033[34m

help: ## Display this help screen
	@echo "$(COLOR_BOLD)Azure DevOps PR CLI - Makefile Commands$(COLOR_RESET)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(COLOR_GREEN)%-20s$(COLOR_RESET) %s\n", $$1, $$2}'

deps: ## Download dependencies
	@echo "$(COLOR_BLUE)Downloading dependencies...$(COLOR_RESET)"
	$(GOMOD) download
	$(GOMOD) tidy

build: deps ## Build the binary
	@echo "$(COLOR_BLUE)Building $(BINARY_NAME)...$(COLOR_RESET)"
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v
	@echo "$(COLOR_GREEN)✓ Build complete: ./$(BINARY_NAME)$(COLOR_RESET)"

install: ## Install the binary to $GOPATH/bin
	@echo "$(COLOR_BLUE)Installing $(BINARY_NAME)...$(COLOR_RESET)"
	$(GO) install $(LDFLAGS)
	@echo "$(COLOR_GREEN)✓ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)$(COLOR_RESET)"

clean: ## Remove build artifacts and test cache
	@echo "$(COLOR_YELLOW)Cleaning...$(COLOR_RESET)"
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out coverage.html
	@echo "$(COLOR_GREEN)✓ Clean complete$(COLOR_RESET)"

test: ## Run all tests
	@echo "$(COLOR_BLUE)Running tests...$(COLOR_RESET)"
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "$(COLOR_GREEN)✓ Tests complete$(COLOR_RESET)"

test-unit: ## Run unit tests only
	@echo "$(COLOR_BLUE)Running unit tests...$(COLOR_RESET)"
	$(GOTEST) -v -short -race -coverprofile=coverage.out ./...
	@echo "$(COLOR_GREEN)✓ Unit tests complete$(COLOR_RESET)"

test-coverage: test ## Generate test coverage report
	@echo "$(COLOR_BLUE)Generating coverage report...$(COLOR_RESET)"
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(COLOR_GREEN)✓ Coverage report: coverage.html$(COLOR_RESET)"

test-coverage-text: test ## Show test coverage in terminal
	@$(GO) tool cover -func=coverage.out

lint: ## Run golangci-lint
	@echo "$(COLOR_BLUE)Running linter...$(COLOR_RESET)"
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin" && exit 1)
	golangci-lint run ./...
	@echo "$(COLOR_GREEN)✓ Linting complete$(COLOR_RESET)"

fmt: ## Format code
	@echo "$(COLOR_BLUE)Formatting code...$(COLOR_RESET)"
	$(GOFMT) ./...
	@echo "$(COLOR_GREEN)✓ Formatting complete$(COLOR_RESET)"

vet: ## Run go vet
	@echo "$(COLOR_BLUE)Running go vet...$(COLOR_RESET)"
	$(GOVET) ./...
	@echo "$(COLOR_GREEN)✓ Vet complete$(COLOR_RESET)"

run: build ## Build and run (set ORG, PROJECT, REPO, PAT env vars)
	@if [ -z "$(REPO)" ]; then \
		echo "$(COLOR_YELLOW)Usage: make run REPO=myrepo [ORG=myorg] [PROJECT=myproject]$(COLOR_RESET)"; \
		exit 1; \
	fi
	./$(BINARY_NAME) list -o $(ORG) -p $(PROJECT) -r $(REPO)

# Cross-compilation targets
build-linux-amd64: deps ## Build for Linux AMD64
	@echo "$(COLOR_BLUE)Building for Linux AMD64...$(COLOR_RESET)"
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64

build-linux-arm64: deps ## Build for Linux ARM64
	@echo "$(COLOR_BLUE)Building for Linux ARM64...$(COLOR_RESET)"
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64

build-darwin-amd64: deps ## Build for macOS AMD64
	@echo "$(COLOR_BLUE)Building for macOS AMD64...$(COLOR_RESET)"
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64

build-darwin-arm64: deps ## Build for macOS ARM64 (M1/M2)
	@echo "$(COLOR_BLUE)Building for macOS ARM64...$(COLOR_RESET)"
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64

build-windows-amd64: deps ## Build for Windows AMD64
	@echo "$(COLOR_BLUE)Building for Windows AMD64...$(COLOR_RESET)"
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe

build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64 ## Build for all platforms
	@echo "$(COLOR_GREEN)✓ All platform builds complete$(COLOR_RESET)"

check: fmt vet lint test ## Run all checks (format, vet, lint, test)

ci: deps check ## Run CI pipeline
	@echo "$(COLOR_GREEN)✓ CI pipeline complete$(COLOR_RESET)"

release: clean build-all ## Build release binaries
	@echo "$(COLOR_BLUE)Creating release archive...$(COLOR_RESET)"
	@mkdir -p dist
	@mv $(BINARY_NAME)-* dist/
	@echo "$(COLOR_GREEN)✓ Release binaries in dist/$(COLOR_RESET)"

docker-build: ## Build Docker image
	@echo "$(COLOR_BLUE)Building Docker image...$(COLOR_RESET)"
	docker build -t $(BINARY_NAME):$(VERSION) .
	@echo "$(COLOR_GREEN)✓ Docker image built: $(BINARY_NAME):$(VERSION)$(COLOR_RESET)"

dev: ## Run in development mode with live reload (requires air)
	@which air > /dev/null || (echo "air not installed. Run: go install github.com/cosmtrek/air@latest" && exit 1)
	air

.DEFAULT_GOAL := help
