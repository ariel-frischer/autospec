# Research: GitHub Issue Templates

**Feature**: 006-github-issue-templates
**Date**: 2025-10-23

## Overview

Research findings for implementing GitHub issue templates. All technical context was clear from requirements; this document captures format decisions and best practices research.

## Technical Decisions

### Decision 1: Markdown Templates vs YAML Forms

**Decision**: Use markdown templates with YAML frontmatter

**Rationale**:
- Simpler to implement and maintain
- More flexible - contributors can add freeform content
- Backward compatible with older GitHub versions
- Easier to read and edit in text editors
- Meets all functional requirements (FR-001 through FR-011)

**Alternatives Considered**:
- YAML form schema templates: More structured with validation and required fields, but:
  - More complex to create and maintain
  - Less flexible for contributors to provide context
  - Overkill for this project's needs
  - Not requested in original requirements

### Decision 2: YAML Frontmatter Format

**Decision**: Use standard GitHub issue template frontmatter format

**Format**:
```yaml
---
name: Template Name
about: Brief description
title: '[PREFIX] '
labels: label1, label2
assignees: username
---
```

**Rationale**:
- Official GitHub format documented at https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests
- Widely adopted across major open source projects
- Allows automatic labeling and title prefixing
- No custom parsing required

**Best Practices Identified**:
- Keep frontmatter minimal - only essential fields
- Use descriptive names that appear in template chooser
- Add helpful "about" text to guide template selection
- Use title prefixes to categorize issues (e.g., `[BUG]`, `[FEATURE]`)
- Labels should align with repository's existing label scheme

### Decision 3: Template Section Structure

**Decision**: Use structured sections with descriptive headings and guidance text

**Bug Report Structure**:
1. Bug Description
2. Steps to Reproduce
3. Expected Behavior
4. Actual Behavior
5. Environment/System Information
6. Additional Context

**Feature Request Structure**:
1. Problem Statement (what need/pain point)
2. Use Case Description (who benefits, how used)
3. Proposed Solution (one possible approach)
4. Alternatives Considered (other options)
5. Additional Context

**Rationale**:
- Aligns with industry best practices (Django, React, Vue.js projects)
- Separates "what" from "how" in feature requests
- Encourages complete bug reports with reproduction steps
- Provides structure while remaining flexible

**Best Practices Identified**:
- Include guidance text under each heading (HTML comments or italic text)
- Use markdown formatting for clarity (bold, code blocks, lists)
- Make sections deletable - don't enforce required fields
- Keep total template length under 50 lines for readability

### Decision 4: Configuration Options

**Decision**: Create minimal config.yml with blank_issues_enabled and contact_links

**Configuration**:
```yaml
blank_issues_enabled: false
contact_links:
  - name: Community Forum
    url: https://example.com/forum
    about: For questions and discussions
```

**Rationale**:
- Disabling blank issues encourages use of structured templates
- Contact links redirect support questions away from issue tracker
- Minimal configuration reduces maintenance burden
- Can expand later if needed

**Alternatives Considered**:
- Allow blank issues: Simpler but leads to incomplete bug reports
- No contact links: Missed opportunity to direct users to appropriate channels

### Decision 5: Testing Strategy

**Decision**: Automated YAML validation + manual GitHub UI verification

**Test Approach**:
1. **Automated Tests**:
   - YAML syntax validation using `yq` or similar tool
   - File existence checks (all 3 required files present)
   - Section presence validation (grep for required headings)
   - Frontmatter completeness (name, about fields present)

2. **Manual Tests**:
   - Create test issues using each template on GitHub
   - Verify template chooser displays correctly
   - Confirm labels and title prefixes apply automatically
   - Test that blank issues are disabled

**Rationale**:
- Automated tests catch syntax errors and missing files
- Manual testing required for GitHub-specific rendering behavior
- No GitHub API endpoint for template validation
- Fast feedback loop: automated tests run in CI, manual tests during PR review

**Best Practices Identified**:
- Use bats framework for bash-based test scripts (matches project conventions)
- Validate YAML with tools like `yq` or `yamllint`
- Include tests in CI/CD pipeline
- Document manual test steps in PR template or tasks.md

## Technology Research

### YAML Validation Tools

**Options Evaluated**:
- `yq`: YAML processor, good for parsing and validation
- `yamllint`: Focused YAML linter, strict validation
- `python -c "import yaml; yaml.safe_load(open('file'))"`: Python-based validation

**Recommendation**: Use `yq` if available, fallback to Python

**Rationale**:
- `yq` is lightweight and widely available
- Python yaml module is standard library (no extra dependencies)
- Both provide sufficient validation for our needs

### GitHub Template Examples

**Projects Reviewed**:
- [microsoft/vscode](https://github.com/microsoft/vscode/tree/main/.github/ISSUE_TEMPLATE)
- [facebook/react](https://github.com/facebook/react/tree/main/.github/ISSUE_TEMPLATE)
- [vuejs/vue](https://github.com/vuejs/vue/tree/dev/.github/ISSUE_TEMPLATE)

**Common Patterns Identified**:
- Most projects use 2-4 templates (bug, feature, question, blank)
- Bug templates consistently request reproduction steps and environment
- Feature templates focus on "why" before "what"
- Config files commonly disable blank issues
- Templates are concise (20-40 lines typically)

## Implementation Notes

### File Locations (Confirmed)

All files must be in `.github/ISSUE_TEMPLATE/` directory per GitHub requirements:
- `.github/ISSUE_TEMPLATE/bug_report.md`
- `.github/ISSUE_TEMPLATE/feature_request.md`
- `.github/ISSUE_TEMPLATE/config.yml`

### Environment Information to Request

For this Go CLI project, relevant environment details:
- Operating System (Linux, macOS, Windows)
- Go version (`go version`)
- autospec version (`./autospec version`)
- Installation method (binary, source)
- Shell environment (bash, zsh, etc.)

### Labels to Use

Suggested labels for auto-application:
- Bug reports: `bug`, `needs-triage`
- Feature requests: `enhancement`, `needs-discussion`

(Actual labels should match existing repository label scheme)

## Open Questions (Resolved)

1. **Q**: Should we use YAML forms instead of markdown templates?
   **A**: No - markdown templates are simpler and meet all requirements

2. **Q**: What environment information should bug reports request?
   **A**: OS, Go version, autospec version, installation method, shell

3. **Q**: Should blank issues be allowed?
   **A**: No - disable to encourage structured reporting

4. **Q**: How to test template rendering without manual GitHub verification?
   **A**: Automated syntax validation + manual verification during PR review

5. **Q**: What labels should be auto-applied?
   **A**: bug/needs-triage for bugs, enhancement/needs-discussion for features

## References

- [GitHub Docs: Issue Templates](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/configuring-issue-templates-for-your-repository)
- [GitHub Docs: Template Syntax](https://docs.github.com/en/communities/using-templates-to-encourage-useful-issues-and-pull-requests/syntax-for-issue-forms)
- [Example: microsoft/vscode templates](https://github.com/microsoft/vscode/tree/main/.github/ISSUE_TEMPLATE)
- [Example: facebook/react templates](https://github.com/facebook/react/tree/main/.github/ISSUE_TEMPLATE)
