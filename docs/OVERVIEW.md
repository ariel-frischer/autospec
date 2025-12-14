# Auto Claude SpecKit

**Automate your feature development workflow with AI-powered specification, planning, and implementation.**

## What is Auto Claude SpecKit?

Auto Claude SpecKit is a command-line tool that orchestrates the complete software development lifecycle by integrating with Claude AI. It transforms natural language feature descriptions into fully-specified, planned, and implemented features through a structured workflow.

The tool automates the SpecKit methodology—a systematic approach to feature development that ensures thorough planning, clear task breakdown, and validated implementation before writing code.

## Key Features

- **Automated Workflow Orchestration**: Execute the complete specify → plan → tasks → implement workflow with a single command
- **Intelligent Retry Management**: Automatic retry with persistent state tracking when validation fails
- **Multi-Phase Execution**: Run individual phases (specify, plan, tasks, implement) or complete workflows
- **Smart Spec Detection**: Automatically detects current feature from git branch or directory structure
- **Flexible Configuration**: Hierarchical configuration system supporting global, local, and environment-based settings
- **Health Checks**: Built-in dependency verification and system health diagnostics
- **Cross-Platform**: Runs on Linux, macOS, and Windows
- **Performance-Optimized**: Sub-second validation checks with <10ms validation functions
- **Progress Indicators**: Real-time feedback during long-running operations

## Target Audience

### Primary Users

- **Solo Developers**: Streamline feature development with automated planning and task breakdown
- **Small Teams**: Standardize development workflows and maintain consistent feature specifications
- **Technical Leaders**: Enforce structured development practices and improve team coordination

### Secondary Users

- **Open Source Maintainers**: Improve contribution quality with standardized feature specifications
- **Technical Writers**: Generate comprehensive documentation from structured specifications
- **Project Managers**: Track feature progress and understand implementation status

## Use Cases

### 1. Feature Development
Transform a feature idea into a complete specification, technical plan, and task breakdown:
```bash
autospec workflow "Add user authentication with OAuth support"
```

### 2. Full Implementation
Execute the entire workflow from specification to completed implementation:
```bash
autospec full "Add dark mode toggle to settings"
```

### 3. Iterative Development
Run individual phases for fine-grained control:
```bash
autospec specify "Add export functionality"
autospec plan "Focus on security and performance"
autospec tasks "Break into small incremental steps"
autospec implement
```

### 4. Team Standardization
Ensure all team members follow the same structured approach:
- Initialize shared configuration
- Use consistent workflow commands
- Validate specifications before implementation
- Track progress with status command

### 5. Documentation Generation
Create comprehensive feature documentation automatically:
- Specifications capture requirements and acceptance criteria
- Plans document architecture and technical decisions
- Tasks provide clear implementation roadmap

## How It Works

Auto Claude SpecKit orchestrates interactions between you, Claude AI, and your codebase:

1. **Specify**: Describe your feature in natural language → Claude generates a detailed specification
2. **Plan**: Specification → Claude creates technical plan with architecture and design decisions
3. **Tasks**: Plan → Claude breaks down into actionable, ordered tasks
4. **Implement**: Tasks → Claude executes implementation with you, validating progress

Each phase includes:
- **Validation**: Ensures output artifacts meet quality standards
- **Retry Logic**: Automatic retry on failure with configurable limits
- **State Persistence**: Tracks progress across multiple execution attempts
- **Progress Feedback**: Real-time status updates during execution

## Getting Started

Ready to streamline your development workflow?

- **[Quick Start Guide](./quickstart.md)**: Install and run your first workflow in 10 minutes
- **[Architecture Overview](./architecture.md)**: Understand system design and components
- **[Command Reference](./reference.md)**: Complete command and configuration documentation
- **[Troubleshooting](./troubleshooting.md)**: Solve common issues and debug problems

## Project Status

Auto Claude SpecKit is actively developed and maintained. The project recently transitioned from bash scripts to a cross-platform Go binary, providing:

- Faster execution and better error handling
- Cross-platform support (Linux, macOS, Windows)
- Improved retry logic and state management
- Enhanced progress indicators and user feedback

For contributors, see [CLAUDE.md](../CLAUDE.md) for detailed development guidelines and architectural documentation.

## Links

- **GitHub Repository**: [anthropics/auto-claude-speckit](https://github.com/anthropics/auto-claude-speckit)
- **Issue Tracker**: [Report bugs or request features](https://github.com/anthropics/auto-claude-speckit/issues)
- **Claude AI**: [Learn more about Claude](https://www.anthropic.com/claude)
- **SpecKit Methodology**: Documentation coming soon

## License

Auto Claude SpecKit is open source software. See LICENSE file for details.
