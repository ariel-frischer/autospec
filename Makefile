.PHONY: help build build-all install clean test test-go test-bash test-all lint lint-go lint-bash fmt vet run dev validate-workflow validate-implement deps snapshot release patch minor major version h b i c t l f r d s p v

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

# Version management (for autobump)
CURRENT_VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
MAJOR := $(shell echo $(CURRENT_VERSION) | sed 's/v//' | cut -d. -f1)
MINOR := $(shell echo $(CURRENT_VERSION) | sed 's/v//' | cut -d. -f2)
PATCH := $(shell echo $(CURRENT_VERSION) | sed 's/v//' | cut -d. -f3)

# Platform detection (override with PLATFORM=github or PLATFORM=gitlab)
REMOTE_URL := $(shell git remote get-url origin 2>/dev/null)
DETECTED_PLATFORM := $(shell echo $(REMOTE_URL) | grep -q github && echo github || (echo $(REMOTE_URL) | grep -q gitlab && echo gitlab || echo unknown))
PLATFORM ?= $(DETECTED_PLATFORM)

# Default target
.DEFAULT_GOAL := help

##@ General

help: ## Display this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

version: ## Show current version and release info
	@echo "Current version: $(CURRENT_VERSION)"
	@echo "Next patch:      v$(MAJOR).$(MINOR).$$(( $(PATCH) + 1 ))"
	@echo "Next minor:      v$(MAJOR).$$(( $(MINOR) + 1 )).0"
	@echo "Next major:      v$$(( $(MAJOR) + 1 )).0.0"
	@echo ""
	@echo "Platform:        $(PLATFORM)"
	@echo "Commit:          $(COMMIT)"
	@echo ""
	@echo "Recent tags:"
	@git tag --sort=-version:refname | head -5 || echo "  (no tags)"

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

snapshot: ## Build snapshot release locally (no publish)
	goreleaser release --snapshot --clean

release: ## Create a release (make release VERSION=v1.0.0)
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make release VERSION=v1.0.0"; \
		echo "  or use: make patch | make minor | make major"; \
		echo "  override platform: PLATFORM=github or PLATFORM=gitlab"; \
		exit 1; \
	fi
	@echo "Releasing $(VERSION) to $(PLATFORM)..."
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
ifeq ($(PLATFORM),github)
	unset GITLAB_TOKEN && GITHUB_TOKEN=$$(gh auth token) goreleaser release --clean
else ifeq ($(PLATFORM),gitlab)
	unset GITHUB_TOKEN && goreleaser release --clean
else
	@echo "Error: Unknown platform '$(PLATFORM)'. Use PLATFORM=github or PLATFORM=gitlab"
	@exit 1
endif

patch: ## Bump patch version (v0.0.X)
	@$(MAKE) release VERSION=v$(MAJOR).$(MINOR).$(shell echo $$(($(PATCH)+1)))

minor: ## Bump minor version (v0.X.0)
	@$(MAKE) release VERSION=v$(MAJOR).$(shell echo $$(($(MINOR)+1))).0

major: ## Bump major version (vX.0.0)
	@$(MAKE) release VERSION=v$(shell echo $$(($(MAJOR)+1))).0.0

release-build: test-all lint build-all ## Run tests, linting, and build all platforms (no publish)
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
s: snapshot ## snapshot
p: patch    ## patch release
v: version  ## version info
