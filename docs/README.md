# Autospec Documentation

User-facing documentation for the autospec CLI tool.

## Documentation Index

### User Guides

- **[Timeout Configuration](./timeout.md)** - Complete guide to configuring and using command timeouts
  - Quick start
  - Configuration options
  - Usage examples
  - Best practices
  - Troubleshooting

- **[Troubleshooting Guide](./troubleshooting.md)** - Solutions to common problems
  - Timeout issues
  - Configuration problems
  - Workflow execution errors
  - Performance issues
  - Debugging techniques

### Developer Documentation

- **[CLAUDE.md](../CLAUDE.md)** - Development documentation for working with this codebase
  - Architecture overview
  - Development patterns
  - Testing guidelines
  - Contributing guide

## Quick Links

### Getting Started

```bash
# Install autospec
make install

# Check dependencies
autospec doctor

# Run workflow
autospec workflow "Add user authentication feature"
```

### Common Tasks

```bash
# Set timeout (10 minutes)
export AUTOSPEC_TIMEOUT=600

# Run individual phases
autospec specify "feature description"
autospec plan
autospec tasks
autospec implement

# Check status
autospec status
autospec config show
```

### Configuration

**Local config** (`.autospec/config.json`):
```json
{
  "timeout": 600,
  "max_retries": 3,
  "claude_cmd": "claude",
  "specify_cmd": "specify"
}
```

**Environment variables**:
```bash
export AUTOSPEC_TIMEOUT=600
export AUTOSPEC_MAX_RETRIES=5
export AUTOSPEC_CLAUDE_CMD=claude
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Validation failed (retryable) |
| 2 | Retry limit exhausted |
| 3 | Invalid arguments |
| 4 | Missing dependencies |
| 5 | Command timeout |

## Support

- **Bug Reports**: Create an issue in the repository
- **Questions**: Check documentation or create a discussion
- **Feature Requests**: Create an issue with enhancement label

## See Also

- [Project README](../README.md) - Project overview and installation
- [CLAUDE.md](../CLAUDE.md) - Developer documentation
- [Specs Directory](../specs/) - Feature specifications and examples
