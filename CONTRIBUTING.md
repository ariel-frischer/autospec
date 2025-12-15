# Contributing to autospec

Thank you for your interest in contributing to autospec! This document provides guidelines for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [How to Submit Issues](#how-to-submit-issues)
- [How to Submit Pull Requests](#how-to-submit-pull-requests)
- [Development Setup](#development-setup)
- [Code Style Guidelines](#code-style-guidelines)
- [Commit Message Conventions](#commit-message-conventions)
- [Testing Requirements](#testing-requirements)

## Code of Conduct

This project adheres to the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## How to Submit Issues

### Bug Reports

Before submitting a bug report:

1. Search [existing issues](https://github.com/ariel-frischer/autospec/issues) to avoid duplicates
2. Update to the latest version to see if the issue persists
3. Collect relevant information:
   - Version (`autospec version`)
   - Operating system and version
   - Configuration (`autospec config show`)
   - Steps to reproduce
   - Expected vs actual behavior
   - Error messages and logs

Use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md) when creating your issue.

### Feature Requests

For feature requests:

1. Check if the feature has already been requested
2. Describe the problem the feature would solve
3. Propose a solution if you have one
4. Consider if this aligns with the project's goals

Use the [feature request template](.github/ISSUE_TEMPLATE/feature_request.md) when creating your request.

## How to Submit Pull Requests

### Before Starting

1. Check for existing PRs addressing the same issue
2. For significant changes, open an issue first to discuss the approach
3. Fork the repository and create a branch from `main`

### PR Process

1. **Create a feature branch:**
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/your-bug-fix
   ```

2. **Make your changes following our guidelines:**
   - Follow the [code style guidelines](#code-style-guidelines)
   - Write tests for new functionality
   - Update documentation as needed

3. **Ensure all checks pass:**
   ```bash
   make lint    # Run linters
   make test    # Run all tests
   make build   # Verify build
   ```

4. **Submit your PR:**
   - Use a clear, descriptive title
   - Reference related issues using `Fixes #123` or `Relates to #123`
   - Fill out the PR template completely
   - Request review from maintainers

### PR Review

- Maintainers will review your PR within a reasonable timeframe
- Address feedback promptly
- Keep PRs focused on a single concern
- Large changes may need to be split into smaller PRs

## Development Setup

### Prerequisites

- Go 1.21 or later
- Git
- Make
- Claude Code CLI (for integration testing)
- SpecKit CLI (for integration testing)

### Getting Started

```bash
# Clone the repository
git clone https://github.com/ariel-frischer/autospec.git
cd autospec

# Install dependencies
go mod download

# Install git hooks (important for dev branch workflow)
make dev-setup

# Build the binary
make build

# Run tests
make test

# Run linters
make lint

# Install locally for testing
make install
```

### Branch Workflow

- **`main`** - Stable release branch (no `.dev/` files)
- **`dev`** - Development branch (has `.dev/` files)

| Action | Allowed |
|--------|---------|
| Merge `dev` -> `main` | Yes |
| Rebase `dev` from `main` | Yes (preferred) |
| Merge `main` -> `dev` | No (use rebase) |

The `dev` branch contains `.dev/` files (docs, scripts, specs) that shouldn't exist on `main`. Using rebase instead of merge keeps history clean and avoids conflicts with these files.

**Syncing dev with main after a release:**
```bash
git checkout dev
git rebase main
git push origin dev --force-with-lease
```

### Git Hooks

Install hooks after cloning:
```bash
make dev-setup
# or: ./scripts/setup-hooks.sh
```

**pre-merge-commit** - Prevents accidentally merging `main` into `dev` branches. Warns that merging will lose `.dev/` files (since they get deleted on main) and suggests using `git rebase main` instead to preserve them. To bypass: `git merge --no-verify main`

**post-merge** - Auto-cleans `.dev/` directory when merging to `main`. Runs automatically after `git merge dev` on main.

**pre-rebase** - Backs up `.dev/` directory before rebasing on the `dev` branch. This ensures `.dev/` files aren't lost during rebase operations.

**post-rewrite** - Restores `.dev/` directory after rebasing on the `dev` branch. Works together with `pre-rebase` to preserve development files.

### Project Structure

```
internal/
├── cli/          # Cobra commands
├── workflow/     # Workflow orchestration
├── config/       # Configuration management
├── validation/   # Validation functions
├── retry/        # Retry state management
├── spec/         # Spec detection
├── git/          # Git helpers
└── errors/       # Error handling
```

For detailed architecture information, see [CLAUDE.md](CLAUDE.md) and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Code Style Guidelines

### Go Code

- Follow standard Go conventions and idioms
- Run `go fmt` on all code
- Run `go vet` to catch common issues
- Use meaningful variable and function names
- Keep functions focused and reasonably sized
- Add comments for exported functions and complex logic

```bash
# Format code
go fmt ./...

# Vet code
go vet ./...

# Or use make
make lint-go
```

### Shell Scripts

- Use `shellcheck` for linting
- Follow POSIX shell conventions where possible
- Quote variables to prevent word splitting
- Use meaningful exit codes

```bash
# Lint shell scripts
make lint-bash
```

### Documentation

- Keep README.md and CLAUDE.md up to date
- Document new CLI commands and flags
- Include examples in help text
- Update CHANGELOG.md for notable changes

## Commit Message Conventions

We follow [Conventional Commits](https://www.conventionalcommits.org/) specification.

### Format

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, no logic change)
- `refactor`: Code refactoring (no feature or fix)
- `perf`: Performance improvements
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

### Scope (optional)

Common scopes: `cli`, `workflow`, `config`, `validation`, `retry`, `docs`, `tests`

### Examples

```bash
# Feature
feat(cli): add --dry-run flag to preview command execution

# Bug fix
fix(validation): handle empty tasks.yaml gracefully

# Documentation
docs(readme): update installation instructions

# Refactoring
refactor(workflow): extract phase execution into separate functions

# Tests
test(validation): add table-driven tests for spec validation
```

### Guidelines

- Use imperative mood ("add feature" not "added feature")
- Keep the first line under 72 characters
- Reference issues in the footer: `Fixes #123`
- Breaking changes should include `BREAKING CHANGE:` in the footer

## Testing Requirements

### Test Coverage

- All new features must have tests
- Bug fixes should include regression tests
- Maintain or improve code coverage

### Test Types

**Unit Tests:**
```bash
# Run all Go tests
make test-go

# Run specific package tests
go test -v ./internal/validation/

# Run specific test
go test -v -run TestValidateSpecFile ./internal/validation/
```

**Benchmark Tests:**
```bash
# Run benchmarks
go test -bench=. ./internal/validation/
```

**Integration Tests:**
```bash
# Run all tests including integration
make test
```

### Writing Tests

Use table-driven tests for validation logic:

```go
func TestValidateSpecFile(t *testing.T) {
    tests := []struct {
        name    string
        specDir string
        wantErr bool
    }{
        {"valid spec", "testdata/valid", false},
        {"missing spec", "testdata/missing", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateSpecFile(tt.specDir)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateSpecFile() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### Performance Contracts

Validation functions must complete in <10ms:

```go
func BenchmarkValidateSpecFile(b *testing.B) {
    for i := 0; i < b.N; i++ {
        ValidateSpecFile("testdata/valid")
    }
}
```

## Questions?

- Open a [discussion](https://github.com/ariel-frischer/autospec/discussions) for general questions
- Check existing [issues](https://github.com/ariel-frischer/autospec/issues) for known problems
- Review [documentation](docs/) for usage information

Thank you for contributing!
