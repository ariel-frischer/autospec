package workflow

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunPreflightChecks tests the pre-flight validation logic
func TestRunPreflightChecks(t *testing.T) {
	tests := map[string]struct {
		setupFunc   func() func()
		wantPassed  bool
		wantMissing int
		wantFailed  int
	}{
		"all checks pass": {
			setupFunc: func() func() {
				// Create temporary directories
				os.MkdirAll(".claude/commands", 0755)
				os.MkdirAll(".autospec", 0755)
				return func() {
					os.RemoveAll(".claude")
					os.RemoveAll(".autospec")
				}
			},
			wantPassed:  true,
			wantMissing: 0,
			wantFailed:  0,
		},
		"missing .claude/commands directory": {
			setupFunc: func() func() {
				os.MkdirAll(".autospec", 0755)
				return func() {
					os.RemoveAll(".autospec")
				}
			},
			wantPassed:  false,
			wantMissing: 1,
		},
		"missing .autospec directory": {
			setupFunc: func() func() {
				os.MkdirAll(".claude/commands", 0755)
				return func() {
					os.RemoveAll(".claude")
				}
			},
			wantPassed:  false,
			wantMissing: 1,
		},
		"missing both directories": {
			setupFunc: func() func() {
				return func() {
					// No cleanup needed
				}
			},
			wantPassed:  false,
			wantMissing: 2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Setup test environment
			cleanup := tc.setupFunc()
			defer cleanup()

			// Run pre-flight checks
			result, err := RunPreflightChecks()
			require.NoError(t, err)

			// Verify results
			assert.Equal(t, tc.wantPassed, result.Passed,
				"Passed status should match")
			if tc.wantMissing > 0 {
				assert.Len(t, result.MissingDirs, tc.wantMissing,
					"Should detect missing directories")
			}
		})
	}
}

// TestCheckCommandExists tests command existence checking
func TestCheckCommandExists(t *testing.T) {
	tests := map[string]struct {
		command string
		wantErr bool
	}{
		"git exists": {
			command: "git",
			wantErr: false,
		},
		"nonexistent command": {
			command: "this-command-does-not-exist-12345",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := checkCommandExists(tc.command)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGenerateMissingDirsWarning tests warning message generation
func TestGenerateMissingDirsWarning(t *testing.T) {
	tests := map[string]struct {
		missingDirs  []string
		gitRoot      string
		wantContains []string
	}{
		"with git root": {
			missingDirs: []string{".claude/commands/", ".autospec/"},
			gitRoot:     "/home/user/project",
			wantContains: []string{
				"WARNING",
				".claude/commands/",
				".autospec/",
				"/home/user/project",
				"autospec init",
			},
		},
		"without git root": {
			missingDirs: []string{".claude/commands/"},
			gitRoot:     "",
			wantContains: []string{
				"WARNING",
				".claude/commands/",
				"autospec init",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			warning := generateMissingDirsWarning(tc.missingDirs, tc.gitRoot)

			for _, want := range tc.wantContains {
				assert.Contains(t, warning, want,
					"Warning should contain: %s", want)
			}
		})
	}
}

// TestShouldRunPreflightChecks tests pre-flight check skipping logic
func TestShouldRunPreflightChecks(t *testing.T) {
	tests := map[string]struct {
		skipPreflight bool
		ciEnvVar      string
		ciValue       string
		wantRun       bool
	}{
		"run normally": {
			skipPreflight: false,
			ciEnvVar:      "",
			ciValue:       "",
			wantRun:       true,
		},
		"skip via flag": {
			skipPreflight: true,
			ciEnvVar:      "",
			ciValue:       "",
			wantRun:       false,
		},
		"skip in GitHub Actions": {
			skipPreflight: false,
			ciEnvVar:      "GITHUB_ACTIONS",
			ciValue:       "true",
			wantRun:       false,
		},
		"skip in GitLab CI": {
			skipPreflight: false,
			ciEnvVar:      "GITLAB_CI",
			ciValue:       "true",
			wantRun:       false,
		},
		"skip in CircleCI": {
			skipPreflight: false,
			ciEnvVar:      "CIRCLECI",
			ciValue:       "true",
			wantRun:       false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Set environment variable if specified
			if tc.ciEnvVar != "" {
				t.Setenv(tc.ciEnvVar, tc.ciValue)
			}

			result := ShouldRunPreflightChecks(tc.skipPreflight)
			assert.Equal(t, tc.wantRun, result,
				"ShouldRunPreflightChecks should return %v", tc.wantRun)
		})
	}
}

// TestCheckDependencies tests dependency checking
func TestCheckDependencies(t *testing.T) {
	// This test will check for git (which should exist)
	// and potentially fail for claude/specify if not installed
	err := CheckDependencies()

	// We can't assert success/failure because it depends on the system
	// But we can verify the error message format if it fails
	if err != nil {
		assert.Contains(t, err.Error(), "missing required dependencies",
			"Error should mention missing dependencies")
	}
}

// TestCheckProjectStructure tests project structure validation
func TestCheckProjectStructure(t *testing.T) {
	// Create temporary directories
	os.MkdirAll(".claude/commands", 0755)
	os.MkdirAll(".autospec", 0755)
	defer func() {
		os.RemoveAll(".claude")
		os.RemoveAll(".autospec")
	}()

	err := CheckProjectStructure()
	assert.NoError(t, err, "Should pass with all directories present")

	// Remove one directory and test again
	os.RemoveAll(".claude")
	err = CheckProjectStructure()
	assert.Error(t, err, "Should fail with missing directory")
	assert.Contains(t, err.Error(), "missing required directories")
}

// BenchmarkRunPreflightChecks benchmarks pre-flight checks performance
// Target: <100ms
func BenchmarkRunPreflightChecks(b *testing.B) {
	// Setup test directories
	os.MkdirAll(".claude/commands", 0755)
	os.MkdirAll(".autospec", 0755)
	defer func() {
		os.RemoveAll(".claude")
		os.RemoveAll(".autospec")
	}()

	// Reset timer after setup
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = RunPreflightChecks()
	}
}

// BenchmarkCheckCommandExists benchmarks command existence checking
func BenchmarkCheckCommandExists(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = checkCommandExists("git")
	}
}

// TestCheckConstitutionExists tests the constitution file validation
func TestCheckConstitutionExists(t *testing.T) {
	tests := map[string]struct {
		setupFunc    func() func()
		wantExists   bool
		wantPath     string
		wantErrEmpty bool
	}{
		"autospec constitution exists": {
			setupFunc: func() func() {
				os.MkdirAll(".autospec/memory", 0755)
				os.WriteFile(".autospec/memory/constitution.md", []byte("# Constitution"), 0644)
				return func() {
					os.RemoveAll(".autospec")
				}
			},
			wantExists:   true,
			wantPath:     ".autospec/memory/constitution.md",
			wantErrEmpty: true,
		},
		"legacy specify constitution exists": {
			setupFunc: func() func() {
				os.MkdirAll(".specify/memory", 0755)
				os.WriteFile(".specify/memory/constitution.md", []byte("# Constitution"), 0644)
				return func() {
					os.RemoveAll(".specify")
				}
			},
			wantExists:   true,
			wantPath:     ".specify/memory/constitution.md",
			wantErrEmpty: true,
		},
		"both constitutions exist - autospec takes precedence": {
			setupFunc: func() func() {
				os.MkdirAll(".autospec/memory", 0755)
				os.WriteFile(".autospec/memory/constitution.md", []byte("# Autospec Constitution"), 0644)
				os.MkdirAll(".specify/memory", 0755)
				os.WriteFile(".specify/memory/constitution.md", []byte("# Specify Constitution"), 0644)
				return func() {
					os.RemoveAll(".autospec")
					os.RemoveAll(".specify")
				}
			},
			wantExists:   true,
			wantPath:     ".autospec/memory/constitution.md",
			wantErrEmpty: true,
		},
		"no constitution exists": {
			setupFunc: func() func() {
				// Ensure neither directory exists
				os.RemoveAll(".autospec")
				os.RemoveAll(".specify")
				return func() {}
			},
			wantExists:   false,
			wantPath:     "",
			wantErrEmpty: false,
		},
		"directories exist but no constitution file": {
			setupFunc: func() func() {
				os.MkdirAll(".autospec/memory", 0755)
				os.MkdirAll(".specify/memory", 0755)
				return func() {
					os.RemoveAll(".autospec")
					os.RemoveAll(".specify")
				}
			},
			wantExists:   false,
			wantPath:     "",
			wantErrEmpty: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cleanup := tc.setupFunc()
			defer cleanup()

			result := CheckConstitutionExists()

			assert.Equal(t, tc.wantExists, result.Exists,
				"Exists should match expected")
			assert.Equal(t, tc.wantPath, result.Path,
				"Path should match expected")
			if tc.wantErrEmpty {
				assert.Empty(t, result.ErrorMessage,
					"ErrorMessage should be empty when constitution exists")
			} else {
				assert.NotEmpty(t, result.ErrorMessage,
					"ErrorMessage should not be empty when constitution missing")
				assert.Contains(t, result.ErrorMessage, "autospec constitution",
					"ErrorMessage should mention how to create constitution")
			}
		})
	}
}

// TestGenerateConstitutionMissingError tests the error message generation
func TestGenerateConstitutionMissingError(t *testing.T) {
	errMsg := generateConstitutionMissingError()

	assert.Contains(t, errMsg, "Error:")
	assert.Contains(t, errMsg, "constitution not found")
	assert.Contains(t, errMsg, "autospec constitution")
	assert.Contains(t, errMsg, ".specify/memory/constitution.md")
	assert.Contains(t, errMsg, "autospec init")
}

// BenchmarkCheckConstitutionExists benchmarks constitution check performance
// Target: <10ms
func BenchmarkCheckConstitutionExists(b *testing.B) {
	// Setup with constitution file
	os.MkdirAll(".autospec/memory", 0755)
	os.WriteFile(".autospec/memory/constitution.md", []byte("# Constitution"), 0644)
	defer os.RemoveAll(".autospec")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = CheckConstitutionExists()
	}
}
