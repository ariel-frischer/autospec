// Package validation_test tests automatic artifact fixing and format normalization.
// Related: internal/validation/autofix.go
// Tags: validation, autofix, artifact, yaml, meta, formatting, repair
package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixArtifact_AddsMetaSection(t *testing.T) {
	// Create a temporary copy of the test file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "spec.yaml")

	// Read the test fixture
	data, err := os.ReadFile("testdata/spec/missing_meta.yaml")
	if err != nil {
		t.Fatalf("failed to read test fixture: %v", err)
	}

	// Write to temp file
	if err := os.WriteFile(tempFile, data, 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Run auto-fix
	result, err := FixArtifact(tempFile, ArtifactTypeSpec)
	if err != nil {
		t.Fatalf("FixArtifact failed: %v", err)
	}

	// Check that at least one fix was applied (meta section)
	if len(result.FixesApplied) == 0 {
		t.Error("expected at least 1 fix applied, got 0")
	}

	// Verify _meta fix was applied
	foundMetaFix := false
	for _, fix := range result.FixesApplied {
		if fix.Type == "add_optional_field" && fix.Path == "_meta" {
			foundMetaFix = true
			break
		}
	}
	if !foundMetaFix {
		t.Error("expected add_optional_field fix for _meta")
	}

	// Verify file was modified
	if !result.Modified {
		t.Error("expected file to be modified")
	}

	// Read the modified file and verify _meta exists
	modifiedData, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("failed to read modified file: %v", err)
	}

	if !strings.Contains(string(modifiedData), "_meta:") {
		t.Error("modified file does not contain _meta section")
	}
	if !strings.Contains(string(modifiedData), "artifact_type: spec") {
		t.Error("modified file does not contain correct artifact_type")
	}
}

func TestFixArtifact_NoFixNeeded(t *testing.T) {
	// Create a temporary copy of the valid test file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "spec.yaml")

	// Read the valid test fixture (already has _meta)
	data, err := os.ReadFile("testdata/spec/valid.yaml")
	if err != nil {
		t.Fatalf("failed to read test fixture: %v", err)
	}

	// Write to temp file
	if err := os.WriteFile(tempFile, data, 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Run auto-fix
	result, err := FixArtifact(tempFile, ArtifactTypeSpec)
	if err != nil {
		t.Fatalf("FixArtifact failed: %v", err)
	}

	// Check that no _meta fix was applied (already exists)
	for _, fix := range result.FixesApplied {
		if fix.Type == "add_optional_field" && fix.Path == "_meta" {
			t.Error("expected no _meta fix as it already exists")
		}
	}

	// Check no remaining errors
	if len(result.RemainingErrors) != 0 {
		t.Errorf("expected 0 remaining errors, got %d", len(result.RemainingErrors))
	}

	// Note: normalize_format fix may be applied if formatting differs from yaml.Marshal output
	// This is expected behavior - valid files may still get formatting normalized
}

func TestFixArtifact_CannotFixMissingRequired(t *testing.T) {
	// Create a temporary copy of the missing_feature test file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "spec.yaml")

	// Read the test fixture (missing required 'feature' field)
	data, err := os.ReadFile("testdata/spec/missing_feature.yaml")
	if err != nil {
		t.Fatalf("failed to read test fixture: %v", err)
	}

	// Write to temp file
	if err := os.WriteFile(tempFile, data, 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Run auto-fix
	result, err := FixArtifact(tempFile, ArtifactTypeSpec)
	if err != nil {
		t.Fatalf("FixArtifact failed: %v", err)
	}

	// Check that remaining errors exist (missing required field can't be fixed)
	if len(result.RemainingErrors) == 0 {
		t.Error("expected remaining errors for missing required field")
	}

	// Verify at least one error is about missing 'feature' field
	found := false
	for _, e := range result.RemainingErrors {
		if strings.Contains(e.Message, "feature") || e.Path == "feature" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected an error about missing 'feature' field")
	}
}

func TestFixArtifact_MalformedYAML(t *testing.T) {
	// Create a temporary file with malformed YAML
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "malformed.yaml")

	malformedContent := `invalid:
  - item1
    bad_indent: value`

	if err := os.WriteFile(tempFile, []byte(malformedContent), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Run auto-fix
	result, err := FixArtifact(tempFile, ArtifactTypeSpec)
	if err != nil {
		t.Fatalf("FixArtifact failed: %v", err)
	}

	// Check that no fixes were applied
	if len(result.FixesApplied) != 0 {
		t.Errorf("expected 0 fixes applied for malformed YAML, got %d", len(result.FixesApplied))
	}

	// Check that remaining errors exist
	if len(result.RemainingErrors) == 0 {
		t.Error("expected remaining errors for malformed YAML")
	}

	// File should not be modified
	if result.Modified {
		t.Error("malformed file should not be modified")
	}
}

func TestFixArtifact_NonExistentFile(t *testing.T) {
	_, err := FixArtifact("/nonexistent/path/to/file.yaml", ArtifactTypeSpec)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestFormatFixes(t *testing.T) {
	tests := map[string]struct {
		fixes    []*AutoFix
		contains []string
	}{
		"no fixes": {
			fixes:    []*AutoFix{},
			contains: []string{"No fixes applied"},
		},
		"one fix": {
			fixes: []*AutoFix{
				{Type: "add_optional_field", Path: "_meta", After: "(added)"},
			},
			contains: []string{"Applied 1 fix(es)", "add_optional_field", "_meta", "(added)"},
		},
		"multiple fixes": {
			fixes: []*AutoFix{
				{Type: "add_optional_field", Path: "_meta", After: "(added)"},
				{Type: "normalize_format", Path: "status", After: "Draft"},
			},
			contains: []string{"Applied 2 fix(es)", "add_optional_field", "normalize_format"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := FormatFixes(tc.fixes)
			for _, s := range tc.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, got %q", s, result)
				}
			}
		})
	}
}

func TestFixArtifact_NormalizesFormatting(t *testing.T) {
	// Create a temporary file with inconsistent formatting (extra blank lines, trailing spaces)
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "spec.yaml")

	// Use extra blank lines and trailing spaces - these will be normalized
	content := `feature:
  branch: "001-test"
  created: "2025-01-15"
  status: "Draft"
  input: "Test feature"


user_stories:
  - id: "US-001"
    title: "Test story"
    priority: "P1"
    as_a: "user"
    i_want: "to test"
    so_that: "I can verify"
    acceptance_scenarios:
      - given: "a test"
        when: "I run it"
        then: "it passes"


requirements:
  functional:
    - id: "FR-001"
      description: "MUST work"


_meta:
  version: "1.0.0"
  generator: "test"

`

	if err := os.WriteFile(tempFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Run auto-fix
	result, err := FixArtifact(tempFile, ArtifactTypeSpec)
	if err != nil {
		t.Fatalf("FixArtifact failed: %v", err)
	}

	// Check for normalize_format fix (extra blank lines will be normalized)
	foundFormatFix := false
	for _, fix := range result.FixesApplied {
		if fix.Type == "normalize_format" {
			foundFormatFix = true
			break
		}
	}

	if !foundFormatFix {
		t.Error("expected normalize_format fix for inconsistent formatting")
	}

	// Verify file was modified
	if !result.Modified {
		t.Error("expected file to be modified")
	}
}

func TestFixArtifact_AllTypes(t *testing.T) {
	types := map[string]struct {
		artifactType ArtifactType
		fixture      string
	}{
		"spec":  {artifactType: ArtifactTypeSpec, fixture: "testdata/spec/valid.yaml"},
		"plan":  {artifactType: ArtifactTypePlan, fixture: "testdata/plan/valid.yaml"},
		"tasks": {artifactType: ArtifactTypeTasks, fixture: "testdata/tasks/valid.yaml"},
	}

	for name, tc := range types {
		t.Run(name, func(t *testing.T) {
			tempDir := t.TempDir()
			tempFile := filepath.Join(tempDir, string(tc.artifactType)+".yaml")

			data, err := os.ReadFile(tc.fixture)
			if err != nil {
				t.Fatalf("failed to read fixture: %v", err)
			}

			if err := os.WriteFile(tempFile, data, 0o644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			result, err := FixArtifact(tempFile, tc.artifactType)
			if err != nil {
				t.Fatalf("FixArtifact failed: %v", err)
			}

			// Valid fixtures should have no fixes and no errors
			if len(result.RemainingErrors) != 0 {
				t.Errorf("expected no remaining errors for valid %s fixture, got %d", tc.artifactType, len(result.RemainingErrors))
			}
		})
	}
}
