// Package cli_test tests the artifact command for validating spec, plan, and tasks YAML files with schema display.
// Related: internal/cli/artifact.go
// Tags: cli, artifact, validation, schema, spec, plan, tasks
package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/validation"
)

func TestArtifactCommand_InvalidType(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runArtifactCommand([]string{"unknown"}, "", &stdout, &stderr)

	if err == nil {
		t.Error("expected error for invalid artifact type")
	}

	if code := ExitCode(err); code != ExitInvalidArguments {
		t.Errorf("exit code = %d, want %d", code, ExitInvalidArguments)
	}

	if !strings.Contains(stderr.String(), "invalid artifact type") {
		t.Errorf("stderr should contain 'invalid artifact type', got: %s", stderr.String())
	}
}

func TestArtifactCommand_MissingFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runArtifactCommand([]string{"spec", "nonexistent.yaml"}, "", &stdout, &stderr)

	if err == nil {
		t.Error("expected error for missing file")
	}

	if code := ExitCode(err); code != ExitInvalidArguments {
		t.Errorf("exit code = %d, want %d", code, ExitInvalidArguments)
	}

	if !strings.Contains(stderr.String(), "not found") {
		t.Errorf("stderr should contain 'not found', got: %s", stderr.String())
	}
}

func TestArtifactCommand_ValidSpec(t *testing.T) {
	var stdout, stderr bytes.Buffer
	testFile := filepath.Join("..", "validation", "testdata", "spec", "valid.yaml")
	err := runArtifactCommand([]string{"spec", testFile}, "", &stdout, &stderr)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		t.Logf("stderr: %s", stderr.String())
	}

	if !strings.Contains(stdout.String(), "is valid") {
		t.Errorf("stdout should contain 'is valid', got: %s", stdout.String())
	}

	if !strings.Contains(stdout.String(), "user stories") {
		t.Errorf("stdout should contain summary with 'user stories', got: %s", stdout.String())
	}
}

func TestArtifactCommand_InvalidSpec(t *testing.T) {
	var stdout, stderr bytes.Buffer
	testFile := filepath.Join("..", "validation", "testdata", "spec", "missing_feature.yaml")
	err := runArtifactCommand([]string{"spec", testFile}, "", &stdout, &stderr)

	if err == nil {
		t.Error("expected error for invalid spec")
	}

	if code := ExitCode(err); code != ExitValidationFailed {
		t.Errorf("exit code = %d, want %d", code, ExitValidationFailed)
	}

	if !strings.Contains(stderr.String(), "has") && !strings.Contains(stderr.String(), "error") {
		t.Errorf("stderr should indicate errors, got: %s", stderr.String())
	}
}

func TestArtifactCommand_ValidPlan(t *testing.T) {
	var stdout, stderr bytes.Buffer
	testFile := filepath.Join("..", "validation", "testdata", "plan", "valid.yaml")
	err := runArtifactCommand([]string{"plan", testFile}, "", &stdout, &stderr)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		t.Logf("stderr: %s", stderr.String())
	}

	if !strings.Contains(stdout.String(), "is valid") {
		t.Errorf("stdout should contain 'is valid', got: %s", stdout.String())
	}
}

func TestArtifactCommand_ValidTasks(t *testing.T) {
	var stdout, stderr bytes.Buffer
	testFile := filepath.Join("..", "validation", "testdata", "tasks", "valid.yaml")
	err := runArtifactCommand([]string{"tasks", testFile}, "", &stdout, &stderr)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		t.Logf("stderr: %s", stderr.String())
	}

	if !strings.Contains(stdout.String(), "is valid") {
		t.Errorf("stdout should contain 'is valid', got: %s", stdout.String())
	}
}

func TestArtifactCommand_SchemaSpec(t *testing.T) {
	// Create temp specs directory for config loading
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	specDir := filepath.Join(specsDir, "001-test")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}
	// Create spec.yaml so detection works
	if err := os.WriteFile(filepath.Join(specDir, "spec.yaml"), []byte("feature:\n  branch: test\n"), 0644); err != nil {
		t.Fatalf("failed to create spec.yaml: %v", err)
	}

	// Create config file pointing to our specs dir
	configFile := filepath.Join(tmpDir, "config.yml")
	configContent := fmt.Sprintf("specs_dir: %s\n", specsDir)
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	// Set schema flag
	oldSchemaFlag := artifactSchemaFlag
	artifactSchemaFlag = true
	defer func() { artifactSchemaFlag = oldSchemaFlag }()

	var stdout, stderr bytes.Buffer
	err := runArtifactCommand([]string{"spec"}, configFile, &stdout, &stderr)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		t.Logf("stderr: %s", stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Schema for spec") {
		t.Errorf("output should contain 'Schema for spec', got: %s", output)
	}

	if !strings.Contains(output, "feature") {
		t.Errorf("output should contain 'feature' field, got: %s", output)
	}

	if !strings.Contains(output, "user_stories") {
		t.Errorf("output should contain 'user_stories' field, got: %s", output)
	}
}

func TestArtifactCommand_SchemaPlan(t *testing.T) {
	// Create temp specs directory for config loading
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	specDir := filepath.Join(specsDir, "001-test")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "plan.yaml"), []byte("plan:\n  branch: test\n"), 0644); err != nil {
		t.Fatalf("failed to create plan.yaml: %v", err)
	}

	configFile := filepath.Join(tmpDir, "config.yml")
	configContent := fmt.Sprintf("specs_dir: %s\n", specsDir)
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	oldSchemaFlag := artifactSchemaFlag
	artifactSchemaFlag = true
	defer func() { artifactSchemaFlag = oldSchemaFlag }()

	var stdout, stderr bytes.Buffer
	err := runArtifactCommand([]string{"plan"}, configFile, &stdout, &stderr)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		t.Logf("stderr: %s", stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Schema for plan") {
		t.Errorf("output should contain 'Schema for plan', got: %s", output)
	}

	if !strings.Contains(output, "technical_context") {
		t.Errorf("output should contain 'technical_context' field, got: %s", output)
	}
}

func TestArtifactCommand_SchemaTasks(t *testing.T) {
	// Create temp specs directory for config loading
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	specDir := filepath.Join(specsDir, "001-test")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(specDir, "tasks.yaml"), []byte("tasks:\n  branch: test\n"), 0644); err != nil {
		t.Fatalf("failed to create tasks.yaml: %v", err)
	}

	configFile := filepath.Join(tmpDir, "config.yml")
	configContent := fmt.Sprintf("specs_dir: %s\n", specsDir)
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	oldSchemaFlag := artifactSchemaFlag
	artifactSchemaFlag = true
	defer func() { artifactSchemaFlag = oldSchemaFlag }()

	var stdout, stderr bytes.Buffer
	err := runArtifactCommand([]string{"tasks"}, configFile, &stdout, &stderr)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		t.Logf("stderr: %s", stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "Schema for tasks") {
		t.Errorf("output should contain 'Schema for tasks', got: %s", output)
	}

	if !strings.Contains(output, "phases") {
		t.Errorf("output should contain 'phases' field, got: %s", output)
	}
}

func TestArtifactCommand_CircularDependency(t *testing.T) {
	var stdout, stderr bytes.Buffer
	testFile := filepath.Join("..", "validation", "testdata", "tasks", "invalid_dep_circular.yaml")
	err := runArtifactCommand([]string{"tasks", testFile}, "", &stdout, &stderr)

	if err == nil {
		t.Error("expected error for circular dependency")
	}

	if !strings.Contains(stderr.String(), "circular dependency") {
		t.Errorf("stderr should contain 'circular dependency', got: %s", stderr.String())
	}
}

func TestExitCode(t *testing.T) {
	tests := map[string]struct {
		err      error
		expected int
	}{
		"nil error":     {err: nil, expected: ExitSuccess},
		"exit error 1":  {err: NewExitError(1), expected: 1},
		"exit error 3":  {err: NewExitError(3), expected: 3},
		"generic error": {err: fmt.Errorf("some error"), expected: ExitValidationFailed},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := ExitCode(tt.err); got != tt.expected {
				t.Errorf("ExitCode() = %d, want %d", got, tt.expected)
			}
		})
	}
}

// Tests for parseArtifactArgs function
func TestParseArtifactArgs(t *testing.T) {
	// Create temp specs directory with spec subdirectory
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	specDir := filepath.Join(specsDir, "001-test-feature")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("failed to create spec dir: %v", err)
	}

	// Create artifact files
	for _, filename := range []string{"spec.yaml", "plan.yaml", "tasks.yaml"} {
		f, err := os.Create(filepath.Join(specDir, filename))
		if err != nil {
			t.Fatalf("failed to create %s: %v", filename, err)
		}
		f.Close()
	}

	tests := map[string]struct {
		args        []string
		specsDir    string
		wantType    validation.ArtifactType
		wantPath    string
		wantErr     bool
		errContains string
	}{
		"explicit type and path": {
			args:     []string{"plan", filepath.Join(specDir, "plan.yaml")},
			specsDir: specsDir,
			wantType: validation.ArtifactTypePlan,
			wantPath: filepath.Join(specDir, "plan.yaml"),
			wantErr:  false,
		},
		"path only - spec.yaml": {
			args:     []string{filepath.Join(specDir, "spec.yaml")},
			specsDir: specsDir,
			wantType: validation.ArtifactTypeSpec,
			wantPath: filepath.Join(specDir, "spec.yaml"),
			wantErr:  false,
		},
		"path only - plan.yaml": {
			args:     []string{filepath.Join(specDir, "plan.yaml")},
			specsDir: specsDir,
			wantType: validation.ArtifactTypePlan,
			wantPath: filepath.Join(specDir, "plan.yaml"),
			wantErr:  false,
		},
		"path only - tasks.yaml": {
			args:     []string{filepath.Join(specDir, "tasks.yaml")},
			specsDir: specsDir,
			wantType: validation.ArtifactTypeTasks,
			wantPath: filepath.Join(specDir, "tasks.yaml"),
			wantErr:  false,
		},
		"invalid type": {
			args:        []string{"unknown"},
			specsDir:    specsDir,
			wantErr:     true,
			errContains: "invalid artifact type",
		},
		"unrecognized filename": {
			args:        []string{"config.yaml"},
			specsDir:    specsDir,
			wantErr:     true,
			errContains: "unrecognized artifact filename",
		},
		"no arguments": {
			args:        []string{},
			specsDir:    specsDir,
			wantErr:     true,
			errContains: "no arguments provided",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := parseArtifactArgs(tt.args, tt.specsDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseArtifactArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error should contain %q, got: %v", tt.errContains, err)
				}
				return
			}

			if got.artType != tt.wantType {
				t.Errorf("artType = %v, want %v", got.artType, tt.wantType)
			}

			if got.filePath != tt.wantPath {
				t.Errorf("filePath = %v, want %v", got.filePath, tt.wantPath)
			}
		})
	}
}

// Test path-only invocation with type inference
func TestArtifactCommand_PathOnlyInvocation(t *testing.T) {
	// Create a temp file with a recognized filename (plan.yaml)
	tmpDir := t.TempDir()
	planFile := filepath.Join(tmpDir, "plan.yaml")

	// Write valid plan content
	content := `plan:
  branch: "test"
  spec_path: "specs/001/spec.yaml"
summary: "Test plan"
technical_context:
  language: "Go"
`
	if err := os.WriteFile(planFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err := runArtifactCommand([]string{planFile}, "", &stdout, &stderr)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		t.Logf("stderr: %s", stderr.String())
	}

	if !strings.Contains(stdout.String(), "is valid") {
		t.Errorf("stdout should contain 'is valid', got: %s", stdout.String())
	}
}

// Test unrecognized filename error
func TestArtifactCommand_UnrecognizedFilename(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runArtifactCommand([]string{"config.yaml"}, "", &stdout, &stderr)

	if err == nil {
		t.Error("expected error for unrecognized filename")
	}

	if code := ExitCode(err); code != ExitInvalidArguments {
		t.Errorf("exit code = %d, want %d", code, ExitInvalidArguments)
	}

	stderrStr := stderr.String()
	if !strings.Contains(stderrStr, "unrecognized artifact filename") {
		t.Errorf("stderr should contain 'unrecognized artifact filename', got: %s", stderrStr)
	}

	if !strings.Contains(stderrStr, "spec.yaml") || !strings.Contains(stderrStr, "plan.yaml") {
		t.Errorf("stderr should list valid filenames, got: %s", stderrStr)
	}
}

// Test .yml extension support
func TestArtifactCommand_YmlExtension(t *testing.T) {
	// Create a temp file with .yml extension
	tmpDir := t.TempDir()
	ymlFile := filepath.Join(tmpDir, "plan.yml")

	// Write valid plan content
	content := `plan:
  branch: "test"
  spec_path: "specs/001/spec.yaml"
summary: "Test plan"
technical_context:
  language: "Go"
`
	if err := os.WriteFile(ymlFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err := runArtifactCommand([]string{ymlFile}, "", &stdout, &stderr)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		t.Logf("stderr: %s", stderr.String())
	}

	if !strings.Contains(stdout.String(), "is valid") {
		t.Errorf("stdout should contain 'is valid', got: %s", stdout.String())
	}
}
