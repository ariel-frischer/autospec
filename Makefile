.PHONY: help build build-all install clean test test-go test-bash test-all lint lint-go lint-bash fmt vet run dev validate-workflow validate-implement deps h b i c t l f r d

# Variables
BINARY_NAME=autospec
CMD_PATH=./cmd/autospec
DIST_DIR=dist
VERSION?=dev
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
MODULE_PATH=github.com/anthropics/auto-claude-speckit

# Build flags
LDFLAGS=-ldflags="-X ${MODULE_PATH}/internal/cli.Version=${VERSION} \
                   -X ${MODULE_PATH}/internal/cli.Commit=${COMMIT} \
                   -X ${MODULE_PATH}/internal/cli.BuildDate=${BUILD_DATE} \
                   -s -w"

# Default target
.DEFAULT_GOAL := help

##@ General

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

build: ## Build the binary for current platform
	@echo "Building ${BINARY_NAME} ${VERSION} (commit: ${COMMIT})"
	@go build ${LDFLAGS} -o ${BINARY_NAME} ${CMD_PATH}
	@echo "Binary built: ${BINARY_NAME}"

build-all: ## Build binaries for all platforms
	@./scripts/build-all.sh ${VERSION}

install: build ## Install binary to ~/.local/bin
	@mkdir -p ~/.local/bin
	@cp ${BINARY_NAME} ~/.local/bin/
	@echo "Installed ${BINARY_NAME} to ~/.local/bin/"
	@echo "Ensure ~/.local/bin is in your PATH"

##@ Development

run: build ## Build and run the binary
	@./${BINARY_NAME}

dev: ## Quick build and run (alias for run)
	@$(MAKE) run

fmt: ## Format Go code
	@echo "Formatting Go code..."
	@go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

deps: ## Download and verify dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod verify
	@echo "Dependencies verified."

vendor: ## Vendor dependencies
	@echo "Vendoring dependencies..."
	@go mod vendor
	@echo "Vendored to ./vendor/"

tidy: ## Tidy go.mod and go.sum
	@echo "Tidying go.mod..."
	@go mod tidy

##@ Testing

test-go: ## Run Go tests
	@echo "Running Go tests..."
	@go test -v -race -cover ./...

test-bash: ## Run bats tests
	@echo "Running bats tests..."
	@./tests/run-all-tests.sh

test-all: test-go test-bash ## Run all tests (Go + bats)

test: test-all ## Alias for test-all

##@ Linting

lint-go: fmt vet ## Lint Go code (fmt + vet)
	@echo "Go linting complete."

lint-bash: ## Lint bash scripts with shellcheck
	@echo "Linting bash scripts..."
	@find scripts -name "*.sh" -exec shellcheck {} \;
	@echo "Bash linting complete."

lint: lint-go lint-bash ## Run all linters

##@ Validation

validate-workflow: ## Run workflow validation script
	@./scripts/speckit-workflow-validate.sh $(FEATURE)

validate-implement: ## Run implementation validation script
	@./scripts/speckit-implement-validate.sh $(FEATURE)

##@ Cleanup

clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	@rm -f ${BINARY_NAME}
	@rm -rf ${DIST_DIR}
	@rm -rf vendor
	@echo "Clean complete."

clean-all: clean ## Clean everything including test artifacts
	@echo "Cleaning test artifacts..."
	@find . -name "*.test" -delete
	@rm -rf /tmp/speckit-retry-*
	@echo "All artifacts cleaned."

##@ Release

release: test-all lint build-all ## Run tests, linting, and build all platforms
	@echo "Release build complete. Binaries in ${DIST_DIR}/"

##@ Abbreviations

h: help     ## help
b: build    ## build
i: install  ## install
c: clean    ## clean
t: test     ## test
l: lint     ## lint
f: fmt      ## fmt
r: run      ## run
d: dev      ## dev
