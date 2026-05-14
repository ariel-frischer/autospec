package config

import "time"

// GetDefaultConfigTemplate returns a fully commented config template
// that helps users understand all available options
func GetDefaultConfigTemplate() string {
	return `# Autospec Configuration
# See 'autospec config -h' for commands, 'autospec config keys' for all options

# Agent settings
agent_preset: ""                      # Built-in agent: claude | codex | opencode
use_subscription: true                # Force subscription mode (no API charges); set false to use API key

# Workflow settings
max_retries: 0                        # Max retry attempts per stage (0-10)
specs_dir: ./specs                    # Directory for feature specs
state_dir: ~/.autospec/state          # Directory for state files
skip_preflight: false                 # Skip preflight checks
timeout: 2400                         # Timeout in seconds (40 min default, 0 = no timeout)
skip_confirmations: false             # Skip confirmation prompts
implement_method: phases              # Default: phases | tasks | single-session
auto_commit: false                    # Auto-create git commit after workflow (disabled by default)

# History settings
max_history_entries: 500              # Max command history entries to retain

# View dashboard settings
view_limit: 5                         # Number of recent specs to display

# Agent initialization settings
default_agents: []                    # Agents to pre-select in 'autospec init' prompt

# Worktree management settings
worktree:
  base_dir: ""                        # Parent dir for worktrees (default: parent of repo)
  prefix: ""                          # Directory name prefix
  setup_script: ""                    # Path to setup script relative to repo
  auto_setup: true                    # Run setup automatically on create
  track_status: true                  # Persist worktree state
  copy_dirs:                          # Non-tracked dirs to copy
    - .autospec
    - .agents
    - .claude
  setup_timeout: 5m                   # Max setup script duration (e.g., '5m', '30s')

# Notifications (all platforms)
notifications:
  enabled: false                      # Enable notifications (opt-in)
  type: both                          # sound | visual | both
  sound_file: ""                      # Custom sound file path (empty = system default)
  on_command_complete: true           # Notify when command finishes
  on_stage_complete: false            # Notify on each stage completion
  on_error: true                      # Notify on failures
  on_long_running: false              # Enable duration-based notifications
  long_running_threshold: 2m          # Threshold for long-running notification

# Cclean (claude-clean) output formatting
cclean:
  verbose: false                      # Verbose output with usage stats and tool IDs (-V)
  line_numbers: false                 # Show line numbers in formatted output (-n)
  style: default                      # Output style: default | compact | minimal | plain (-s)

# Codex automated output formatting
codex_output:
  mode: compact                       # compact | full
  max_lines_per_message: 40           # Max displayed lines per compact output block
  color: true                         # Colorize compact Codex output

# Autonomous execution
skip_permissions: true                # Enable autonomous mode for supported agents

# Verification settings
verification:
  level: basic                        # Verification depth: basic | enhanced | full
  # adversarial_review: false         # Override: enable adversarial review
  # contracts: false                  # Override: enable contracts verification
  # property_tests: false             # Override: enable property-based tests
  # metamorphic_tests: false          # Override: enable metamorphic tests
  mutation_threshold: 0.8             # Minimum mutation score (0.0-1.0)
  coverage_threshold: 0.85            # Minimum code coverage (0.0-1.0)
  complexity_max: 10                  # Maximum cyclomatic complexity
`
}

// GetDefaults returns the default configuration values
func GetDefaults() map[string]interface{} {
	return map[string]interface{}{
		// Agent configuration
		"agent_preset":       "",
		"use_subscription":   true, // Protect users from accidental API charges
		"max_retries":        0,
		"specs_dir":          "./specs",
		"state_dir":          "~/.autospec/state",
		"skip_preflight":     false,
		"timeout":            2400,  // 40 minutes default
		"skip_confirmations": false, // Confirmation prompts enabled by default
		// implement_method: Default to "phases" for cost-efficient execution with context isolation.
		// This changes the legacy behavior (single-session) to run each phase in a separate agent session.
		// Valid values: "single-session", "phases", "tasks"
		"implement_method": "phases",
		// notifications: Notification settings for command and stage completion.
		// Disabled by default (opt-in). When enabled, defaults to both sound and visual notifications.
		"notifications": map[string]interface{}{
			"enabled":                false,                      // Disabled by default (opt-in)
			"type":                   "both",                     // Both sound and visual when enabled
			"sound_file":             "",                         // Use system default sound
			"on_command_complete":    true,                       // Notify when command finishes (default when enabled)
			"on_stage_complete":      false,                      // Don't notify on each stage by default
			"on_error":               true,                       // Notify on failures (default when enabled)
			"on_long_running":        false,                      // Don't use duration threshold by default
			"long_running_threshold": (2 * time.Minute).String(), // 2 minutes threshold
		},
		// max_history_entries: Maximum number of command history entries to retain.
		// Oldest entries are pruned when this limit is exceeded.
		"max_history_entries": 500,
		// view_limit: Number of recent specs to display in the view command.
		// Default: 5. Can be overridden with --limit flag.
		"view_limit": 5,
		// default_agents: List of agent names to pre-select in 'autospec init' prompts.
		// Saved from previous init selections. Empty by default.
		"default_agents": []string{},
		// worktree: Configuration for git worktree management.
		// Used by 'autospec worktree' command for creating and managing worktrees.
		"worktree": map[string]interface{}{
			"base_dir":      "",                                          // Parent directory for new worktrees
			"prefix":        "",                                          // Directory name prefix
			"setup_script":  "",                                          // Path to setup script relative to repo
			"auto_setup":    true,                                        // Run setup automatically on create
			"track_status":  true,                                        // Persist worktree state
			"copy_dirs":     []string{".autospec", ".agents", ".claude"}, // Non-tracked dirs to copy
			"setup_timeout": (5 * time.Minute).String(),                  // Max setup script duration (5m default)
		},
		// auto_commit: Enable automatic git commit creation after workflow completion.
		// When true, instructions are injected to update .gitignore, stage files, and create commits.
		// Default: false (disabled due to inconsistent behavior).
		"auto_commit": false,
		// enable_risk_assessment: Controls whether risk assessment instructions are injected
		// into the plan stage prompt. When enabled, generated plan.yaml includes a risks section.
		// Default: false (opt-in feature to reduce cognitive overhead for simple features).
		"enable_risk_assessment": false,
		// skip_permissions_notice_shown: Tracks whether the security notice about
		// --dangerously-skip-permissions has been shown. Set to true after first display.
		// User-level config only (not shown in project config).
		"skip_permissions_notice_shown": false,
		// cclean: Configuration options for cclean (claude-clean) output formatting.
		// Controls verbose mode, line numbers, and output style for stream-json display.
		// Environment variable support via AUTOSPEC_CCLEAN_* prefix.
		"cclean": map[string]interface{}{
			"verbose":      false,     // Verbose output with usage stats and tool IDs (-V flag)
			"line_numbers": false,     // Show line numbers in formatted output (-n flag)
			"style":        "default", // Output style: default, compact, minimal, plain (-s flag)
		},
		// codex_output: Output handling for automated Codex CLI runs.
		// compact uses `codex exec --json` and summarizes JSONL events; full
		// preserves Codex's native transcript.
		"codex_output": map[string]interface{}{
			"mode":                  CodexOutputModeCompact,
			"max_lines_per_message": 40,
			"color":                 true,
		},
		// skip_permissions: Enable autonomous mode for supported agents.
		// Claude uses --dangerously-skip-permissions; Codex uses
		// --dangerously-bypass-approvals-and-sandbox.
		// Default: true for unattended autospec runs. Can be set via AUTOSPEC_SKIP_PERMISSIONS env var.
		"skip_permissions": true,
		// verification: Configuration for verification depth and feature toggles.
		// Controls verification level (basic, enhanced, full), individual feature toggles,
		// and quality thresholds. Environment variable support via AUTOSPEC_VERIFICATION_* prefix.
		"verification": map[string]interface{}{
			"level":              "basic", // Default to basic for backwards compatibility
			"mutation_threshold": 0.8,     // 80% mutation score threshold
			"coverage_threshold": 0.85,    // 85% code coverage threshold
			"complexity_max":     10,      // Max cyclomatic complexity
		},
	}
}
