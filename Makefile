.PHONY: help build build-all install clean test test-go lint lint-go lint-bash fmt vet run dev dev-setup deps snapshot release patch minor major version worktree worktree-list worktree-remove h w b i c t l f r d s p v

# Variables
BINARY_NAME=autospec
BINARY_PATH=bin/$(BINARY_NAME)
CMD_PATH=./cmd/autospec
DIST_DIR=dist
VERSION?=dev
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
MODULE_PATH=github.com/ariel-frischer/autospec

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
	@mkdir -p bin
	@go build ${LDFLAGS} -o ${BINARY_PATH} ${CMD_PATH}
	@echo "Binary built: ${BINARY_PATH}"

build-all: ## Build binaries for all platforms
	@./scripts/build-all.sh ${VERSION}

install: build ## Install binary to ~/.local/bin
	@mkdir -p ~/.local/bin
	@cp ${BINARY_PATH} ~/.local/bin/
	@echo "Installed ${BINARY_NAME} to ~/.local/bin/"
	@if echo "$$PATH" | tr ':' '\n' | grep -qx "$$HOME/.local/bin"; then \
		echo "✓ ~/.local/bin is already in your PATH"; \
		echo ""; \
		echo "Verify installation:"; \
		echo "  which ${BINARY_NAME}"; \
		echo "  ${BINARY_NAME} version"; \
	else \
		echo ""; \
		echo "⚠ ~/.local/bin is NOT in your PATH"; \
		echo ""; \
		echo "Add it to your shell config:"; \
		echo ""; \
		echo "  # Bash (~/.bashrc or ~/.bash_profile)"; \
		echo "  export PATH=\"\$$HOME/.local/bin:\$$PATH\""; \
		echo ""; \
		echo "  # Zsh (~/.zshrc)"; \
		echo "  export PATH=\"\$$HOME/.local/bin:\$$PATH\""; \
		echo ""; \
		echo "  # Fish (~/.config/fish/config.fish)"; \
		echo "  fish_add_path ~/.local/bin"; \
		echo ""; \
		echo "Then reload your shell:"; \
		echo "  source ~/.bashrc   # Bash"; \
		echo "  source ~/.zshrc    # Zsh"; \
		echo "  exec fish          # Fish"; \
		echo ""; \
		echo "Or start a new terminal session."; \
	fi

##@ Development

run: build ## Build and run the binary
	@./${BINARY_PATH}

serve: ## Serve documentation site locally at http://127.0.0.1:4000/autospec/
	@cd site && bundle exec jekyll serve --livereload

dev: ## Quick build and run (alias for run)
	@$(MAKE) run

dev-setup: ## Install git hooks for development
	@./scripts/setup-hooks.sh

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

test-go: ## Run Go tests (quiet, shows failures only)
	@go test -race -cover ./...

test-v: ## Run Go tests (verbose, for debugging)
	@go test -v -race -cover ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -v -race -tags=integration ./tests/integration/...

test-all: test-go test-integration ## Run all tests including integration

test: test-go ## Run all tests

test-summary: ## Show test statistics summary
	@echo "📊 Test Summary"
	@echo "═══════════════════════════════════════════════════════════════"
	@echo ""
	@echo "Running tests and collecting stats..."
	@echo ""
	@OUTPUT=$$(go test ./... -v -cover 2>&1); \
	TOTAL=$$(echo "$$OUTPUT" | grep -c "^=== RUN" || echo 0); \
	PASSED=$$(echo "$$OUTPUT" | grep -c "^--- PASS" || echo 0); \
	FAILED=$$(echo "$$OUTPUT" | grep -c "^--- FAIL" || echo 0); \
	SKIPPED=$$(echo "$$OUTPUT" | grep -c "^--- SKIP" || echo 0); \
	PKGS=$$(go list ./... | wc -l); \
	TOP_LEVEL=$$(echo "$$OUTPUT" | grep "^=== RUN" | grep -cv "/" || echo 0); \
	SUBTESTS=$$((TOTAL - TOP_LEVEL)); \
	echo "  Total test runs:     $$TOTAL"; \
	echo "  ├─ Top-level tests:  $$TOP_LEVEL"; \
	echo "  └─ Subtests:         $$SUBTESTS"; \
	echo ""; \
	echo "  ✅ Passed:           $$PASSED"; \
	echo "  ❌ Failed:           $$FAILED"; \
	echo "  ⏭️  Skipped:          $$SKIPPED"; \
	echo ""; \
	echo "  📦 Packages:         $$PKGS"; \
	echo ""; \
	echo "Coverage by package:"; \
	echo "$$OUTPUT" | grep -E "^ok.*coverage" | head -15; \
	echo ""; \
	echo "(Run 'make test-cover' for detailed coverage report)"

test-cover: ## Show detailed coverage by package
	@go test ./... -cover -coverprofile=coverage.out 2>&1
	@echo ""
	@echo "📈 Coverage Summary:"
	@go tool cover -func=coverage.out | tail -1
	@echo ""
	@echo "Top 10 packages by coverage:"
	@go tool cover -func=coverage.out | grep -v "total:" | sort -t: -k3 -rn | head -10
	@rm -f coverage.out

##@ Linting

lint-go: fmt vet ## Lint Go code (fmt + vet)
	@echo "Go linting complete."

lint-bash: ## Lint bash scripts with shellcheck
	@echo "Linting bash scripts..."
	@find . -name '*.sh' -type f -not -path './.specify/*' -not -path '*/.autospec/*' -not -name 'quickstart-demo.sh' | xargs shellcheck -x --severity=warning
	@echo "Bash linting complete."

lint: lint-go lint-bash ## Run all linters

##@ Cleanup

clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin
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

release-build: test lint build-all ## Run tests, linting, and build all platforms (no publish)
	@echo "Release build complete. Binaries in ${DIST_DIR}/"

##@ Git Worktree

worktree: ## Create a new worktree (make worktree BRANCH=feature-name)
	@if [ -z "$(BRANCH)" ]; then \
		echo "Usage: make worktree BRANCH=feature-name"; \
		echo ""; \
		echo "This creates a new worktree at ../autospec-<branch>"; \
		exit 1; \
	fi
	@WORKTREE_PATH="../autospec-$(BRANCH)"; \
	ORIG_DIR="$$(pwd)"; \
	if [ -d "$$WORKTREE_PATH" ]; then \
		echo "Worktree already exists at $$WORKTREE_PATH"; \
		echo ""; \
		echo "To enter it, run:"; \
		echo "  cd $$WORKTREE_PATH"; \
	else \
		echo "Creating worktree for branch '$(BRANCH)' at $$WORKTREE_PATH..."; \
		git worktree add "$$WORKTREE_PATH" -b "$(BRANCH)" 2>/dev/null || \
		git worktree add "$$WORKTREE_PATH" "$(BRANCH)"; \
		echo ""; \
		echo "✓ Worktree created at $$WORKTREE_PATH"; \
		echo ""; \
		echo "Initializing autospec in worktree..."; \
		cd "$$WORKTREE_PATH" && autospec init; \
		echo ""; \
		echo "Copying .autospec/* to worktree (excluding context/)..."; \
		mkdir -p "$$WORKTREE_PATH/.autospec"; \
		find "$$ORIG_DIR/.autospec" -mindepth 1 -maxdepth 1 -not -name "context" -exec cp -rf {} "$$WORKTREE_PATH/.autospec/" \;; \
		echo "✓ Copied .autospec/ contents (excluding context/)"; \
		echo ""; \
		if [ -f "$$ORIG_DIR/.claude/settings.local.json" ]; then \
			echo "Copying .claude/settings.local.json to worktree..."; \
			mkdir -p "$$WORKTREE_PATH/.claude"; \
			cp "$$ORIG_DIR/.claude/settings.local.json" "$$WORKTREE_PATH/.claude/"; \
			echo "✓ Copied .claude/settings.local.json"; \
			echo ""; \
		fi; \
		echo "To enter it, run:"; \
		echo "  cd $$WORKTREE_PATH"; \
	fi

worktree-list: ## List all worktrees
	@git worktree list

worktree-remove: ## Remove a worktree (make worktree-remove BRANCH=feature-name)
	@if [ -z "$(BRANCH)" ]; then \
		echo "Usage: make worktree-remove BRANCH=feature-name"; \
		exit 1; \
	fi
	@WORKTREE_PATH="../autospec-$(BRANCH)"; \
	git worktree remove --force "$$WORKTREE_PATH" && \
	echo "✓ Removed worktree at $$WORKTREE_PATH"

##@ Abbreviations

h: help     ## help
w: worktree ## worktree
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
