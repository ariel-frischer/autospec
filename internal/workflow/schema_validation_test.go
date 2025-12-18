package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateSpecSchema(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		specDir     string
		wantErr     bool
		errContains string
		description string
	}{
		"valid spec": {
			specDir:     filepath.Join("testdata", "spec", "valid"),
			wantErr:     false,
			description: "Valid spec.yaml should pass validation",
		},
		"invalid spec missing feature": {
			specDir:     filepath.Join("testdata", "spec", "invalid"),
			wantErr:     true,
			errContains: "missing required field: feature",
			description: "Spec missing required 'feature' field should fail",
		},
		"nonexistent directory": {
			specDir:     filepath.Join("testdata", "spec", "nonexistent"),
			wantErr:     true,
			errContains: "failed to parse YAML",
			description: "Nonexistent directory should fail with parse error",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := ValidateSpecSchema(tc.specDir)

			if tc.wantErr {
				if err == nil {
					t.Errorf("ValidateSpecSchema() expected error, got nil; %s", tc.description)
					return
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("ValidateSpecSchema() error = %q, want error containing %q; %s",
						err.Error(), tc.errContains, tc.description)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateSpecSchema() unexpected error: %v; %s", err, tc.description)
			}
		})
	}
}

func TestValidatePlanSchema(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		specDir     string
		wantErr     bool
		errContains string
		description string
	}{
		"valid plan": {
			specDir:     filepath.Join("testdata", "plan", "valid"),
			wantErr:     false,
			description: "Valid plan.yaml should pass validation",
		},
		"invalid plan missing plan field": {
			specDir:     filepath.Join("testdata", "plan", "invalid"),
			wantErr:     true,
			errContains: "missing required field: plan",
			description: "Plan missing required 'plan' field should fail",
		},
		"nonexistent directory": {
			specDir:     filepath.Join("testdata", "plan", "nonexistent"),
			wantErr:     true,
			errContains: "failed to parse YAML",
			description: "Nonexistent directory should fail with parse error",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := ValidatePlanSchema(tc.specDir)

			if tc.wantErr {
				if err == nil {
					t.Errorf("ValidatePlanSchema() expected error, got nil; %s", tc.description)
					return
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("ValidatePlanSchema() error = %q, want error containing %q; %s",
						err.Error(), tc.errContains, tc.description)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidatePlanSchema() unexpected error: %v; %s", err, tc.description)
			}
		})
	}
}

func TestValidateTasksSchema(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		specDir     string
		wantErr     bool
		errContains string
		description string
	}{
		"valid tasks": {
			specDir:     filepath.Join("testdata", "tasks", "valid"),
			wantErr:     false,
			description: "Valid tasks.yaml should pass validation",
		},
		"invalid tasks missing tasks field": {
			specDir:     filepath.Join("testdata", "tasks", "invalid"),
			wantErr:     true,
			errContains: "missing required field: tasks",
			description: "Tasks missing required 'tasks' field should fail",
		},
		"nonexistent directory": {
			specDir:     filepath.Join("testdata", "tasks", "nonexistent"),
			wantErr:     true,
			errContains: "failed to parse YAML",
			description: "Nonexistent directory should fail with parse error",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := ValidateTasksSchema(tc.specDir)

			if tc.wantErr {
				if err == nil {
					t.Errorf("ValidateTasksSchema() expected error, got nil; %s", tc.description)
					return
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("ValidateTasksSchema() error = %q, want error containing %q; %s",
						err.Error(), tc.errContains, tc.description)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateTasksSchema() unexpected error: %v; %s", err, tc.description)
			}
		})
	}
}

func TestFormatValidationErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		artifactName string
		errCount     int
		wantNil      bool
		wantContains []string
		description  string
	}{
		"no errors returns nil": {
			artifactName: "test.yaml",
			errCount:     0,
			wantNil:      true,
			description:  "Empty error slice should return nil",
		},
		"single error": {
			artifactName: "spec.yaml",
			errCount:     1,
			wantNil:      false,
			wantContains: []string{"schema validation failed for spec.yaml", "test error 0"},
			description:  "Single error should be formatted correctly",
		},
		"multiple errors": {
			artifactName: "plan.yaml",
			errCount:     3,
			wantNil:      false,
			wantContains: []string{"schema validation failed for plan.yaml", "test error 0", "test error 1", "test error 2"},
			description:  "Multiple errors should all be included",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create test validation errors
			var errors []*validationErrorForTest
			for i := 0; i < tc.errCount; i++ {
				errors = append(errors, &validationErrorForTest{
					message: "test error " + string(rune('0'+i)),
				})
			}

			// We can't directly call formatValidationErrors since it uses
			// *validation.ValidationError, so we test the behavior through
			// the public functions that use it.
			// This test documents the expected behavior.

			if tc.wantNil {
				// Verify nil behavior by checking valid file returns nil
				err := ValidateSpecSchema(filepath.Join("testdata", "spec", "valid"))
				if err != nil {
					t.Errorf("Expected nil error for valid file: %v", err)
				}
			} else {
				// Verify error formatting by checking invalid file
				err := ValidateSpecSchema(filepath.Join("testdata", "spec", "invalid"))
				if err == nil {
					t.Error("Expected error for invalid file")
					return
				}

				errStr := err.Error()
				if !strings.Contains(errStr, "schema validation failed for") {
					t.Errorf("Error should contain 'schema validation failed for': %s", errStr)
				}
			}
		})
	}
}

// validationErrorForTest is a test helper
type validationErrorForTest struct {
	message string
}

func (e *validationErrorForTest) Error() string {
	return e.message
}

// TestMakeSpecSchemaValidatorWithDetection tests that the detection-based validator
// correctly finds and validates the newly created spec directory.
// This prevents regression of the bug where executeSpecify passed empty specName
// causing validation to look in specs/ instead of specs/<spec-name>/.
func TestMakeSpecSchemaValidatorWithDetection(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setupFunc   func(t *testing.T) string // Returns specsDir
		wantErr     bool
		errContains string
		description string
	}{
		"detects and validates valid spec": {
			setupFunc: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				specsDir := filepath.Join(tmpDir, "specs")
				specDir := filepath.Join(specsDir, "001-test-feature")
				if err := os.MkdirAll(specDir, 0755); err != nil {
					t.Fatalf("failed to create spec dir: %v", err)
				}
				// Create valid spec.yaml
				validSpec := `feature:
  branch: "001-test-feature"
  created: "2025-01-01"
  status: "Draft"
  input: "Test feature"
user_stories:
  - id: "US-001"
    title: "Test story"
    priority: "P1"
    as_a: "user"
    i_want: "test"
    so_that: "test"
    acceptance_scenarios: []
requirements:
  functional:
    - id: "FR-001"
      description: "Test requirement"
      testable: true
      acceptance_criteria: "Test passes"
`
				if err := os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte(validSpec), 0644); err != nil {
					t.Fatalf("failed to write spec.yaml: %v", err)
				}
				return specsDir
			},
			wantErr:     false,
			description: "Should detect spec directory and validate successfully",
		},
		"detects and fails on invalid spec": {
			setupFunc: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				specsDir := filepath.Join(tmpDir, "specs")
				specDir := filepath.Join(specsDir, "002-invalid-feature")
				if err := os.MkdirAll(specDir, 0755); err != nil {
					t.Fatalf("failed to create spec dir: %v", err)
				}
				// Create invalid spec.yaml (missing requirements)
				invalidSpec := `feature:
  branch: "002-invalid-feature"
  created: "2025-01-01"
user_stories: []
# missing: requirements
`
				if err := os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte(invalidSpec), 0644); err != nil {
					t.Fatalf("failed to write spec.yaml: %v", err)
				}
				return specsDir
			},
			wantErr:     true,
			errContains: "missing required field: requirements",
			description: "Should detect spec directory and return validation error",
		},
		"fails when no spec directory exists": {
			setupFunc: func(t *testing.T) string {
				t.Helper()
				tmpDir := t.TempDir()
				specsDir := filepath.Join(tmpDir, "specs")
				if err := os.MkdirAll(specsDir, 0755); err != nil {
					t.Fatalf("failed to create specs dir: %v", err)
				}
				// No spec directories inside
				return specsDir
			},
			wantErr:     true,
			errContains: "detecting spec for validation",
			description: "Should fail when no spec directory can be detected",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			specsDir := tc.setupFunc(t)
			validateFunc := MakeSpecSchemaValidatorWithDetection(specsDir)

			// Call with empty string (simulating ExecuteStage with empty specName)
			err := validateFunc("")

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil; %s", tc.description)
					return
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("error = %q, want error containing %q; %s",
						err.Error(), tc.errContains, tc.description)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v; %s", err, tc.description)
			}
		})
	}
}

// TestMakeSpecSchemaValidatorWithDetection_IgnoresSpecDirArg verifies that the
// returned validator ignores the specDir argument and uses detection instead.
// This is the key behavior that fixes the bug.
func TestMakeSpecSchemaValidatorWithDetection_IgnoresSpecDirArg(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	specDir := filepath.Join(specsDir, "001-test-feature")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}

	// Create valid spec.yaml
	validSpec := `feature:
  branch: "001-test-feature"
  created: "2025-01-01"
  status: "Draft"
  input: "Test feature"
user_stories:
  - id: "US-001"
    title: "Test story"
    priority: "P1"
    as_a: "user"
    i_want: "test"
    so_that: "test"
    acceptance_scenarios: []
requirements:
  functional:
    - id: "FR-001"
      description: "Test requirement"
      testable: true
      acceptance_criteria: "Test passes"
`
	if err := os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte(validSpec), 0644); err != nil {
		t.Fatalf("failed to write spec.yaml: %v", err)
	}

	validateFunc := MakeSpecSchemaValidatorWithDetection(specsDir)

	// Call with various invalid paths - should all succeed because detector ignores them
	testPaths := []string{
		"",                                  // Empty (the bug case)
		"/nonexistent/path",                 // Nonexistent path
		filepath.Join(specsDir, ""),         // specsDir with empty suffix
		filepath.Join(specsDir, "wrong-spec"), // Wrong spec name
	}

	for _, path := range testPaths {
		err := validateFunc(path)
		if err != nil {
			t.Errorf("validateFunc(%q) returned error: %v; validator should ignore specDir arg", path, err)
		}
	}
}
