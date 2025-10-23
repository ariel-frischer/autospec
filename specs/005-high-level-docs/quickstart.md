# Quickstart Design: High-Level Documentation

**Feature**: 005-high-level-docs
**Date**: 2025-10-23

## Purpose

This document describes the design and structure of the `docs/quickstart.md` file that will be created during implementation. It serves as a blueprint for the actual documentation file.

## Target Audience

- **Primary**: New users who want to use Auto Claude SpecKit for the first time
- **Secondary**: Developers evaluating the tool before adoption

## Design Goals

1. **10-minute completion**: User can complete their first workflow within 10 minutes
2. **Zero assumptions**: Assume no prior knowledge of SpecKit or Claude
3. **Success-focused**: Guide user to successful completion, anticipate failure points
4. **Progressive disclosure**: Show basics first, link to advanced topics

## File Structure

### Header (H1)
- Title: "Quick Start Guide"
- Brief one-sentence description of document purpose

### Prerequisites (H2)
**Content**:
- Claude CLI installed and authenticated (link to Claude docs)
- Git repository initialized (not strictly required, but recommended)
- Basic command line familiarity

**Validation**:
- Provide `claude --version` command to verify installation
- Link to troubleshooting.md for installation issues

### Installation (H2)

**Content**:
```bash
# Option 1: Build from source (recommended for contributors)
git clone <repo-url>
cd auto-claude-speckit
make build
make install

# Option 2: Download binary (recommended for users)
# Link to releases page with instructions
```

**Validation**:
- Provide `autospec version` command to verify installation
- Mention expected output: version number
- Link to troubleshooting.md if command not found

### Your First Workflow (H2)

**Content**: Step-by-step guide to complete specify → plan → tasks workflow

**Steps**:

1. **Initialize configuration**
   ```bash
   autospec init
   ```
   - Explain what this creates (~/.autospec/config.json)
   - Show sample config with comments

2. **Verify setup**
   ```bash
   autospec doctor
   ```
   - Explain health check output
   - What to do if checks fail (link to troubleshooting.md)

3. **Create your first feature spec**
   ```bash
   autospec specify "Add dark mode toggle to settings page"
   ```
   - Show expected output
   - Explain what file is created (specs/NNN-feature-name/spec.md)
   - Time estimate: ~2 minutes

4. **Generate implementation plan**
   ```bash
   autospec plan
   ```
   - Explain auto-detection of current feature
   - Show expected output (plan.md created)
   - Time estimate: ~3 minutes

5. **Generate tasks**
   ```bash
   autospec tasks
   ```
   - Show expected output (tasks.md created)
   - Time estimate: ~2 minutes

6. **Review generated artifacts**
   ```bash
   ls specs/001-dark-mode-toggle/
   # Expected: spec.md, plan.md, tasks.md
   ```

**Success Criteria**:
- All three files exist
- No error messages during execution
- User understands workflow progression

### Common Commands (H2)

**Content**: Quick reference for frequently used commands

**Format**: Table with command, description, example

| Command | Description | Example |
|---------|-------------|---------|
| `autospec full "..."` | Complete workflow (specify → plan → tasks → implement) | `autospec full "Add user auth"` |
| `autospec workflow "..."` | Partial workflow (specify → plan → tasks) | `autospec workflow "Add user auth"` |
| `autospec implement` | Execute implementation phase | `autospec implement` |
| `autospec status` | Check current feature status | `autospec status` |
| `autospec --help` | Show all commands | `autospec --help` |

**Link**: For complete command reference, see reference.md

### Understanding the Workflow (H2)

**Content**: Brief explanation of workflow phases

**Diagram**: Simple Mermaid flowchart showing:
```mermaid
graph LR
    A[specify] --> B[plan]
    B --> C[tasks]
    C --> D[implement]

    A:::phase
    B:::phase
    C:::phase
    D:::phase

    classDef phase fill:#e1f5ff
```

**Phase Descriptions**:
- **specify**: Create feature specification with requirements and acceptance criteria
- **plan**: Generate implementation plan with architecture and design
- **tasks**: Break down plan into actionable tasks
- **implement**: Execute tasks with Claude's assistance

**Link**: For detailed architecture, see architecture.md

### Configuration Basics (H2)

**Content**: Essential configuration options

**Format**: Code block with JSON and inline comments

```json
{
  // Claude CLI command (default: "claude")
  "claude_cmd": "claude",

  // Maximum retry attempts (default: 3)
  "max_retries": 3,

  // Specs directory (default: "./specs")
  "specs_dir": "./specs",

  // Command timeout in seconds (0 = no timeout)
  "timeout": 0
}
```

**Link**: For complete configuration reference, see reference.md

### Troubleshooting (H2)

**Content**: Quick solutions for common first-time issues

**Format**: Problem → Solution pairs

1. **"claude: command not found"**
   - Solution: Install Claude CLI (link to installation guide)

2. **"autospec: command not found"**
   - Solution: Run `make install` or add binary to PATH

3. **"Workflow validation failed"**
   - Solution: Check retry count, see troubleshooting.md for details

4. **"Spec not detected"**
   - Solution: Ensure you're on a feature branch (NNN-feature-name format)

**Link**: For comprehensive troubleshooting, see troubleshooting.md

### Next Steps (H2)

**Content**: Where to go after completing first workflow

**Links**:
- **Advanced usage**: Reference.md for all commands and options
- **Understanding the system**: Architecture.md for design details
- **Customization**: Configuration reference in reference.md
- **Contributing**: CLAUDE.md for development guidelines
- **Getting help**: Link to GitHub issues

## Content Guidelines

### Tone
- Friendly and encouraging
- Clear and direct (no jargon)
- Action-oriented (imperative commands)

### Formatting
- Use code blocks for all commands
- Use inline code for file names and paths
- Use bold for emphasis (sparingly)
- Use links for cross-references

### Examples
- Use realistic feature names (not "foo" or "test")
- Show actual expected output
- Provide context for why each step matters

## Validation Checklist

Before considering quickstart.md complete:

- [ ] File is under 500 lines
- [ ] All commands are accurate and tested
- [ ] All file paths are correct
- [ ] All links resolve to existing files or sections
- [ ] Can be followed by someone unfamiliar with project
- [ ] Success criteria are clear at each step
- [ ] Time estimates are realistic
- [ ] Troubleshooting covers most common issues

## Line Count Estimate

Based on structure above:

- Header: ~5 lines
- Prerequisites: ~15 lines
- Installation: ~20 lines
- First Workflow: ~80 lines
- Common Commands: ~30 lines
- Understanding Workflow: ~40 lines
- Configuration Basics: ~30 lines
- Troubleshooting: ~40 lines
- Next Steps: ~20 lines

**Total Estimate**: ~280 lines (well under 500-line limit)

## Implementation Notes

- Test all commands before writing documentation
- Use actual output from running commands (not hypothetical)
- Verify all links before committing
- Consider adding screenshots or asciinema recordings (future enhancement)
- Keep language at 8th-grade reading level for accessibility
