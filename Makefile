# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint

# Binary name
BINARY_NAME=trazr-gen

# Main package path
MAIN_PACKAGE=./cmd/trazr-gen

# Build directory
BUILD_DIR=./build

# Source files
SRC=$(shell find . -name "*.go")

# Test coverage output
COVERAGE_OUTPUT=coverage.out

.PHONY: all build clean test coverage lint deps tidy run help integration-coverage

all: build

build: ## Build the binary
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)

clean: ## Remove build artifacts
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f $(COVERAGE_OUTPUT)

test: ## Run tests
	$(GOTEST) -v ./...

coverage: ## Run tests with coverage
	$(GOTEST) -v -coverprofile=$(COVERAGE_OUTPUT) ./...
	$(GOCMD) tool cover -html=$(COVERAGE_OUTPUT)

lint: ## Run linter
	$(GOLINT) run

deps: ## Download dependencies
	$(GOGET) -v -t -d ./...

tidy: ## Tidy and verify dependencies
	$(GOMOD) tidy
	$(GOMOD) verify

run: build ## Run the application
	$(BUILD_DIR)/$(BINARY_NAME)

docker-build: ## Build Docker image
	docker build -t $(BINARY_NAME) .

docker-run: ## Run Docker container
	docker run --rm $(BINARY_NAME)

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

integration-coverage: ## Run all tests (including integration) with coverage and print total coverage
	$(GOTEST) -coverprofile=$(COVERAGE_OUTPUT) ./...
	@echo "\nTotal coverage:" && $(GOCMD) tool cover -func=$(COVERAGE_OUTPUT) | grep total:

.DEFAULT_GOAL := help
