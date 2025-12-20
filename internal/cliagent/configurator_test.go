package cliagent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// mockConfigurableAgent is a test agent that implements Configurator.
type mockConfigurableAgent struct {
	BaseAgent
	configResult ConfigResult
	configErr    error
	callCount    int
}

func (m *mockConfigurableAgent) ConfigureProject(projectDir, specsDir string) (ConfigResult, error) {
	m.callCount++
	if m.configErr != nil {
		return ConfigResult{}, m.configErr
	}
	return m.configResult, nil
}

// mockNonConfigurableAgent is a test agent that does NOT implement Configurator.
type mockNonConfigurableAgent struct {
	BaseAgent
}

func TestConfigure(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agent       Agent
		projectDir  string
		specsDir    string
		wantResult  *ConfigResult
		wantErr     bool
		wantErrMsg  string
		wantNilCall bool // true if Configure should return (nil, nil)
	}{
		"non-configurator agent returns nil": {
			agent:       &mockNonConfigurableAgent{},
			projectDir:  "/project",
			specsDir:    "specs",
			wantResult:  nil,
			wantNilCall: true,
		},
		"configurator with permissions added": {
			agent: &mockConfigurableAgent{
				configResult: ConfigResult{
					PermissionsAdded: []string{"Write(.autospec/**)", "Edit(.autospec/**)"},
				},
			},
			projectDir: "/project",
			specsDir:   "specs",
			wantResult: &ConfigResult{
				PermissionsAdded: []string{"Write(.autospec/**)", "Edit(.autospec/**)"},
			},
		},
		"configurator already configured": {
			agent: &mockConfigurableAgent{
				configResult: ConfigResult{
					AlreadyConfigured: true,
				},
			},
			projectDir: "/project",
			specsDir:   "specs",
			wantResult: &ConfigResult{
				AlreadyConfigured: true,
			},
		},
		"configurator with warning": {
			agent: &mockConfigurableAgent{
				configResult: ConfigResult{
					PermissionsAdded: []string{"Write(specs/**)"},
					Warning:          "Some permissions could not be added due to deny list",
				},
			},
			projectDir: "/project",
			specsDir:   "specs",
			wantResult: &ConfigResult{
				PermissionsAdded: []string{"Write(specs/**)"},
				Warning:          "Some permissions could not be added due to deny list",
			},
		},
		"configurator returns error": {
			agent: &mockConfigurableAgent{
				configErr: errors.New("permission denied"),
			},
			projectDir: "/project",
			specsDir:   "specs",
			wantErr:    true,
			wantErrMsg: "permission denied",
		},
		"custom specs directory": {
			agent: &mockConfigurableAgent{
				configResult: ConfigResult{
					PermissionsAdded: []string{"Write(features/**)", "Edit(features/**)"},
				},
			},
			projectDir: "/my/project",
			specsDir:   "features",
			wantResult: &ConfigResult{
				PermissionsAdded: []string{"Write(features/**)", "Edit(features/**)"},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := Configure(tt.agent, tt.projectDir, tt.specsDir)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("Configure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.wantErrMsg != "" && err != nil {
				if err.Error() != tt.wantErrMsg {
					t.Errorf("Configure() error = %q, want %q", err.Error(), tt.wantErrMsg)
				}
				return
			}

			// Check nil call case
			if tt.wantNilCall {
				if result != nil {
					t.Errorf("Configure() result = %v, want nil for non-configurator", result)
				}
				return
			}

			// Check result
			if result == nil {
				t.Fatal("Configure() result = nil, want non-nil")
			}

			if len(result.PermissionsAdded) != len(tt.wantResult.PermissionsAdded) {
				t.Errorf("PermissionsAdded len = %d, want %d",
					len(result.PermissionsAdded), len(tt.wantResult.PermissionsAdded))
			}
			for i, perm := range result.PermissionsAdded {
				if i < len(tt.wantResult.PermissionsAdded) && perm != tt.wantResult.PermissionsAdded[i] {
					t.Errorf("PermissionsAdded[%d] = %q, want %q",
						i, perm, tt.wantResult.PermissionsAdded[i])
				}
			}
			if result.AlreadyConfigured != tt.wantResult.AlreadyConfigured {
				t.Errorf("AlreadyConfigured = %v, want %v",
					result.AlreadyConfigured, tt.wantResult.AlreadyConfigured)
			}
			if result.Warning != tt.wantResult.Warning {
				t.Errorf("Warning = %q, want %q", result.Warning, tt.wantResult.Warning)
			}
		})
	}
}

func TestIsConfigurator(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		agent Agent
		want  bool
	}{
		"agent implements Configurator": {
			agent: &mockConfigurableAgent{},
			want:  true,
		},
		"agent does not implement Configurator": {
			agent: &mockNonConfigurableAgent{},
			want:  false,
		},
		"base agent does not implement Configurator": {
			agent: &BaseAgent{AgentName: "test"},
			want:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := IsConfigurator(tt.agent); got != tt.want {
				t.Errorf("IsConfigurator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigResult_ZeroValue(t *testing.T) {
	t.Parallel()

	// Verify zero value is sensible
	var result ConfigResult

	if result.AlreadyConfigured {
		t.Error("zero value AlreadyConfigured should be false")
	}
	if len(result.PermissionsAdded) != 0 {
		t.Error("zero value PermissionsAdded should be empty")
	}
	if result.Warning != "" {
		t.Error("zero value Warning should be empty string")
	}
}

func TestConfigure_IdempotencyCheck(t *testing.T) {
	t.Parallel()

	agent := &mockConfigurableAgent{
		configResult: ConfigResult{
			PermissionsAdded: []string{"Write(.autospec/**)"},
		},
	}

	// Call Configure multiple times
	for i := 0; i < 3; i++ {
		_, err := Configure(agent, "/project", "specs")
		if err != nil {
			t.Fatalf("Configure() call %d error = %v", i+1, err)
		}
	}

	// Verify ConfigureProject was called each time
	if agent.callCount != 3 {
		t.Errorf("ConfigureProject call count = %d, want 3", agent.callCount)
	}
}

// Ensure mockNonConfigurableAgent satisfies Agent interface
var _ Agent = (*mockNonConfigurableAgent)(nil)

// Ensure mockConfigurableAgent satisfies both Agent and Configurator interfaces
var _ Agent = (*mockConfigurableAgent)(nil)
var _ Configurator = (*mockConfigurableAgent)(nil)

// mockNonConfigurableAgent needs minimal Agent interface implementation
func (m *mockNonConfigurableAgent) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Result, error) {
	return &Result{}, nil
}

// mockConfigurableAgent needs minimal Agent interface implementation
func (m *mockConfigurableAgent) Execute(ctx context.Context, prompt string, opts ExecOptions) (*Result, error) {
	return &Result{}, nil
}

func TestBuildClaudePermissions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		specsDir string
		want     []string
	}{
		"default specs directory": {
			specsDir: "specs",
			want: []string{
				"Bash(autospec:*)",
				"Write(.autospec/**)",
				"Edit(.autospec/**)",
				"Write(specs/**)",
				"Edit(specs/**)",
			},
		},
		"custom specs directory": {
			specsDir: "features",
			want: []string{
				"Bash(autospec:*)",
				"Write(.autospec/**)",
				"Edit(.autospec/**)",
				"Write(features/**)",
				"Edit(features/**)",
			},
		},
		"nested specs directory": {
			specsDir: "my/custom/specs",
			want: []string{
				"Bash(autospec:*)",
				"Write(.autospec/**)",
				"Edit(.autospec/**)",
				"Write(my/custom/specs/**)",
				"Edit(my/custom/specs/**)",
			},
		},
		"specs directory with special chars": {
			specsDir: "spec-files",
			want: []string{
				"Bash(autospec:*)",
				"Write(.autospec/**)",
				"Edit(.autospec/**)",
				"Write(spec-files/**)",
				"Edit(spec-files/**)",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := buildClaudePermissions(tt.specsDir)

			if len(got) != len(tt.want) {
				t.Fatalf("buildClaudePermissions() returned %d permissions, want %d", len(got), len(tt.want))
			}

			for i, perm := range got {
				if perm != tt.want[i] {
					t.Errorf("buildClaudePermissions()[%d] = %q, want %q", i, perm, tt.want[i])
				}
			}
		})
	}
}

func TestClaudeConfigureProject(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		specsDir              string
		existingPermissions   []string
		denyList              []string
		wantPermissionsAdded  int
		wantAlreadyConfigured bool
		wantWarning           bool
	}{
		"fresh project with default specs dir": {
			specsDir:             "specs",
			existingPermissions:  nil,
			wantPermissionsAdded: 5,
		},
		"fresh project with custom specs dir": {
			specsDir:             "features",
			existingPermissions:  nil,
			wantPermissionsAdded: 5,
		},
		"project with all permissions already configured": {
			specsDir: "specs",
			existingPermissions: []string{
				"Bash(autospec:*)",
				"Write(.autospec/**)",
				"Edit(.autospec/**)",
				"Write(specs/**)",
				"Edit(specs/**)",
			},
			wantPermissionsAdded:  0,
			wantAlreadyConfigured: true,
		},
		"project with partial permissions": {
			specsDir: "specs",
			existingPermissions: []string{
				"Bash(autospec:*)",
				"Write(.autospec/**)",
			},
			wantPermissionsAdded: 3, // Edit(.autospec/**), Write(specs/**), Edit(specs/**)
		},
		"project with deny list conflict": {
			specsDir:            "specs",
			existingPermissions: nil,
			denyList: []string{
				"Bash(autospec:*)",
			},
			wantPermissionsAdded: 5, // All permissions are still added, just with warning
			wantWarning:          true,
		},
		"project with multiple deny list conflicts": {
			specsDir:            "specs",
			existingPermissions: nil,
			denyList: []string{
				"Bash(autospec:*)",
				"Write(.autospec/**)",
			},
			wantPermissionsAdded: 5,
			wantWarning:          true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create temp directory for test
			tempDir := t.TempDir()

			// Set up existing settings if needed
			if len(tt.existingPermissions) > 0 || len(tt.denyList) > 0 {
				settingsDir := tempDir + "/.claude"
				if err := createTestSettingsDir(settingsDir); err != nil {
					t.Fatalf("failed to create settings dir: %v", err)
				}

				settingsContent := buildTestSettingsJSON(tt.existingPermissions, tt.denyList)
				if err := writeTestSettings(settingsDir+"/settings.local.json", settingsContent); err != nil {
					t.Fatalf("failed to write settings: %v", err)
				}
			}

			// Create Claude agent and call ConfigureProject
			claude := NewClaude()
			result, err := claude.ConfigureProject(tempDir, tt.specsDir)

			if err != nil {
				t.Fatalf("ConfigureProject() error = %v", err)
			}

			// Check permissions added count
			if len(result.PermissionsAdded) != tt.wantPermissionsAdded {
				t.Errorf("PermissionsAdded count = %d, want %d\nGot: %v",
					len(result.PermissionsAdded), tt.wantPermissionsAdded, result.PermissionsAdded)
			}

			// Check already configured flag
			if result.AlreadyConfigured != tt.wantAlreadyConfigured {
				t.Errorf("AlreadyConfigured = %v, want %v", result.AlreadyConfigured, tt.wantAlreadyConfigured)
			}

			// Check warning presence
			hasWarning := result.Warning != ""
			if hasWarning != tt.wantWarning {
				t.Errorf("Warning presence = %v (warning: %q), want %v", hasWarning, result.Warning, tt.wantWarning)
			}
		})
	}
}

func TestClaudeConfigureProject_Idempotency(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	claude := NewClaude()

	// First call should add all permissions
	result1, err := claude.ConfigureProject(tempDir, "specs")
	if err != nil {
		t.Fatalf("first ConfigureProject() error = %v", err)
	}
	if len(result1.PermissionsAdded) != 5 {
		t.Errorf("first call PermissionsAdded = %d, want 5", len(result1.PermissionsAdded))
	}
	if result1.AlreadyConfigured {
		t.Error("first call should not be AlreadyConfigured")
	}

	// Second call should report already configured
	result2, err := claude.ConfigureProject(tempDir, "specs")
	if err != nil {
		t.Fatalf("second ConfigureProject() error = %v", err)
	}
	if len(result2.PermissionsAdded) != 0 {
		t.Errorf("second call PermissionsAdded = %d, want 0", len(result2.PermissionsAdded))
	}
	if !result2.AlreadyConfigured {
		t.Error("second call should be AlreadyConfigured")
	}

	// Third call should also report already configured
	result3, err := claude.ConfigureProject(tempDir, "specs")
	if err != nil {
		t.Fatalf("third ConfigureProject() error = %v", err)
	}
	if !result3.AlreadyConfigured {
		t.Error("third call should be AlreadyConfigured")
	}
}

func TestClaudeConfigureProject_NoDuplicates(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Set up settings with some existing permissions
	settingsDir := tempDir + "/.claude"
	if err := createTestSettingsDir(settingsDir); err != nil {
		t.Fatalf("failed to create settings dir: %v", err)
	}

	existingPerms := []string{"Bash(autospec:*)", "Write(.autospec/**)"}
	settingsContent := buildTestSettingsJSON(existingPerms, nil)
	if err := writeTestSettings(settingsDir+"/settings.local.json", settingsContent); err != nil {
		t.Fatalf("failed to write settings: %v", err)
	}

	claude := NewClaude()
	result, err := claude.ConfigureProject(tempDir, "specs")
	if err != nil {
		t.Fatalf("ConfigureProject() error = %v", err)
	}

	// Should only add the 3 missing permissions
	expectedAdded := []string{
		"Edit(.autospec/**)",
		"Write(specs/**)",
		"Edit(specs/**)",
	}

	if len(result.PermissionsAdded) != len(expectedAdded) {
		t.Fatalf("PermissionsAdded = %v, want %v", result.PermissionsAdded, expectedAdded)
	}

	// Verify no duplicates by checking exact matches
	for i, perm := range result.PermissionsAdded {
		if perm != expectedAdded[i] {
			t.Errorf("PermissionsAdded[%d] = %q, want %q", i, perm, expectedAdded[i])
		}
	}
}

func TestClaudeImplementsConfigurator(t *testing.T) {
	t.Parallel()

	claude := NewClaude()

	if !IsConfigurator(claude) {
		t.Error("Claude should implement Configurator interface")
	}

	// Verify we can use Configure helper with Claude
	tempDir := t.TempDir()
	result, err := Configure(claude, tempDir, "specs")
	if err != nil {
		t.Fatalf("Configure(claude) error = %v", err)
	}
	if result == nil {
		t.Error("Configure(claude) should return non-nil result")
	}
}

// Helper functions for tests

func createTestSettingsDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func writeTestSettings(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}

func buildTestSettingsJSON(allowList, denyList []string) []byte {
	settings := map[string]interface{}{}

	perms := map[string]interface{}{}
	if len(allowList) > 0 {
		allow := make([]interface{}, len(allowList))
		for i, p := range allowList {
			allow[i] = p
		}
		perms["allow"] = allow
	}
	if len(denyList) > 0 {
		deny := make([]interface{}, len(denyList))
		for i, p := range denyList {
			deny[i] = p
		}
		perms["deny"] = deny
	}
	if len(perms) > 0 {
		settings["permissions"] = perms
	}

	data, _ := json.MarshalIndent(settings, "", "  ")
	return data
}

// TestBuildClaudePermissions_SpecialCharacters verifies that specs_dir with
// various special characters is correctly handled in permission strings.
// Edge case from spec: "Special characters in specs_dir: correctly escaped in permissions"
func TestBuildClaudePermissions_SpecialCharacters(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		specsDir string
		want     []string
	}{
		"specs dir with spaces": {
			specsDir: "my specs",
			want: []string{
				"Bash(autospec:*)",
				"Write(.autospec/**)",
				"Edit(.autospec/**)",
				"Write(my specs/**)",
				"Edit(my specs/**)",
			},
		},
		"specs dir with parentheses": {
			specsDir: "specs(v2)",
			want: []string{
				"Bash(autospec:*)",
				"Write(.autospec/**)",
				"Edit(.autospec/**)",
				"Write(specs(v2)/**)",
				"Edit(specs(v2)/**)",
			},
		},
		"specs dir with underscores and numbers": {
			specsDir: "spec_files_2024",
			want: []string{
				"Bash(autospec:*)",
				"Write(.autospec/**)",
				"Edit(.autospec/**)",
				"Write(spec_files_2024/**)",
				"Edit(spec_files_2024/**)",
			},
		},
		"specs dir with dots": {
			specsDir: "specs.d",
			want: []string{
				"Bash(autospec:*)",
				"Write(.autospec/**)",
				"Edit(.autospec/**)",
				"Write(specs.d/**)",
				"Edit(specs.d/**)",
			},
		},
		"specs dir with unicode": {
			specsDir: "スペック",
			want: []string{
				"Bash(autospec:*)",
				"Write(.autospec/**)",
				"Edit(.autospec/**)",
				"Write(スペック/**)",
				"Edit(スペック/**)",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := buildClaudePermissions(tt.specsDir)

			if len(got) != len(tt.want) {
				t.Fatalf("buildClaudePermissions() returned %d permissions, want %d", len(got), len(tt.want))
			}

			for i, perm := range got {
				if perm != tt.want[i] {
					t.Errorf("buildClaudePermissions()[%d] = %q, want %q", i, perm, tt.want[i])
				}
			}
		})
	}
}

// TestClaudeConfigureProject_SpecsDirWithSpaces tests that specs_dir containing
// spaces works correctly through the full configuration flow.
func TestClaudeConfigureProject_SpecsDirWithSpaces(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	claude := NewClaude()

	result, err := claude.ConfigureProject(tempDir, "my specs")
	if err != nil {
		t.Fatalf("ConfigureProject() error = %v", err)
	}

	// Verify all 5 permissions were added
	if len(result.PermissionsAdded) != 5 {
		t.Errorf("PermissionsAdded count = %d, want 5", len(result.PermissionsAdded))
	}

	// Verify the specs permissions contain the space correctly
	foundSpecsWrite := false
	foundSpecsEdit := false
	for _, perm := range result.PermissionsAdded {
		if perm == "Write(my specs/**)" {
			foundSpecsWrite = true
		}
		if perm == "Edit(my specs/**)" {
			foundSpecsEdit = true
		}
	}

	if !foundSpecsWrite {
		t.Error("expected Write(my specs/**) in permissions")
	}
	if !foundSpecsEdit {
		t.Error("expected Edit(my specs/**) in permissions")
	}
}
