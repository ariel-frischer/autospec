// autospec - Automated SpecKit workflow validation for Claude Code
//
// Dependency Size Summary (production binary):
//   Runtime dependencies:  ~500 KB (koanf, cobra, spinner, color)
//   Indirect dependencies: ~9 MB (mostly golang.org/x/sys)
//   Total binary size:     ~7.2 MB
//
// Note: testify (400K) is test-only and NOT included in the production binary.
// Go automatically excludes dependencies only used in *_test.go files.

module github.com/ariel-frischer/autospec

go 1.25.1

require (
	// Pure Go git implementation for repository operations without CLI dependency
	// Enables autospec to perform git operations (branch detection, repo validation, fetch)
	// without requiring git CLI installation for core operations
	github.com/go-git/go-git/v5 v5.16.4

	// Koanf configuration management library (224K total for all koanf packages)
	// Provides flexible config loading from multiple sources with priority ordering
	github.com/knadh/koanf/parsers/json v1.0.0 // JSON parser for config files
	github.com/knadh/koanf/providers/env v1.1.0 // Environment variable provider
	github.com/knadh/koanf/providers/file v1.2.0 // File-based config provider
	github.com/knadh/koanf/v2 v2.3.0 // Core koanf library

	// CLI framework for building command-line applications (292K)
	// Powers autospec's command structure (init, config, workflow, etc.)
	github.com/spf13/cobra v1.10.1

	// ============================================================================
	// TEST-ONLY DEPENDENCIES (NOT included in binary - only in *_test.go files)
	// ============================================================================

	// Testing toolkit with assertions and mocking (400K source, 0 KB in binary)
	// Used in unit and integration tests - Go excludes this automatically
	github.com/stretchr/testify v1.11.1

	// YAML parser and validator (364K)
	// Used for YAML artifact syntax validation and parsing
	gopkg.in/yaml.v3 v3.0.1
)

require (
	// ============================================================================
	// TEST-ONLY INDIRECT DEPENDENCIES (NOT in binary - only used by testify)
	// ============================================================================
	github.com/davecgh/go-spew v1.1.1 // indirect; indirect - Deep pretty printer (100K source, 0 KB in binary)

	// ============================================================================
	// PRODUCTION INDIRECT DEPENDENCIES (included in binary)
	// ============================================================================

	// Configuration and file system utilities
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect; indirect - Decode generic maps into structs (152K)

	// CLI framework dependencies
	github.com/inconshreveable/mousetrap v1.1.0 // indirect; indirect - Windows console handling (28K)
	github.com/knadh/koanf/maps v0.1.2 // indirect; indirect - Map manipulation utilities (included in koanf)

	// Reflection and struct utilities
	github.com/mitchellh/copystructure v1.2.0 // indirect; indirect - Deep copying of Go structures (32K)
	github.com/mitchellh/reflectwalk v1.0.2 // indirect; indirect - Reflection-based struct walking (36K)
	github.com/pmezard/go-difflib v1.0.0 // indirect; indirect - Diff library (36K source, 0 KB in binary)
	github.com/spf13/pflag v1.0.9 // indirect; indirect - POSIX/GNU-style flags (312K)
	golang.org/x/sys v0.39.0 // indirect - Low-level OS primitives (9.0M) ⚠️ LARGEST DEPENDENCY
)

require (
	github.com/briandowns/spinner v1.23.2
	golang.org/x/term v0.37.0
)

require (
	// Cross-platform file system notifications for log tailing (232K)
	// Used by dag logs command for efficient file change detection
	github.com/fsnotify/fsnotify v1.9.0
	github.com/knadh/koanf/parsers/yaml v0.1.0
	golang.org/x/sync v0.19.0
)

require github.com/ariel-frischer/claude-clean v0.2.0

require (
	dario.cat/mergo v1.0.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProtonMail/go-crypto v1.1.6 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/cyphar/filepath-securejoin v0.4.1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.6.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/pjbgf/sha1cd v0.3.2 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/skeema/knownhosts v1.3.1 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	golang.org/x/crypto v0.45.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
)

require (
	// Terminal colors with auto-detection (used in errors package)
	github.com/fatih/color v1.18.0
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
)
