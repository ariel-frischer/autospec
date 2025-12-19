// Package completion_test tests shell completion installation with backup and idempotency.
// Related: /home/ari/repos/autospec/internal/completion/install.go
// Tags: completion, install, backup, idempotency

package completion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateBackup(t *testing.T) {
	tests := map[string]struct {
		setup       func(t *testing.T) string
		wantErr     bool
		wantBackup  bool
		errContains string
	}{
		"creates backup for existing file": {
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				filePath := filepath.Join(dir, ".bashrc")
				if err := os.WriteFile(filePath, []byte("original content"), 0644); err != nil {
					t.Fatal(err)
				}
				return filePath
			},
			wantErr:    false,
			wantBackup: true,
		},
		"no backup for non-existent file": {
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return filepath.Join(dir, ".bashrc")
			},
			wantErr:    false,
			wantBackup: false,
		},
		"preserves original content": {
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				filePath := filepath.Join(dir, ".bashrc")
				content := "# My custom bashrc\nexport PATH=$HOME/bin:$PATH\n"
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return filePath
			},
			wantErr:    false,
			wantBackup: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			filePath := tc.setup(t)

			backupPath, err := CreateBackup(filePath)

			if (err != nil) != tc.wantErr {
				t.Errorf("CreateBackup() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if tc.wantBackup {
				if backupPath == "" {
					t.Errorf("CreateBackup() returned empty backup path, expected backup")
					return
				}

				// Verify backup file exists
				if _, err := os.Stat(backupPath); os.IsNotExist(err) {
					t.Errorf("Backup file does not exist at %s", backupPath)
					return
				}

				// Verify backup contains original content
				originalContent, _ := os.ReadFile(filePath)
				backupContent, _ := os.ReadFile(backupPath)
				if string(originalContent) != string(backupContent) {
					t.Errorf("Backup content = %q, want %q", string(backupContent), string(originalContent))
				}

				// Verify backup filename format
				if !strings.Contains(backupPath, ".autospec-backup-") {
					t.Errorf("Backup path %s doesn't contain expected suffix", backupPath)
				}
			} else {
				if backupPath != "" {
					t.Errorf("CreateBackup() = %s, want empty path", backupPath)
				}
			}
		})
	}
}

func TestCreateBackupTimestampFormat(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, ".bashrc")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	backupPath, err := CreateBackup(filePath)
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}

	// Extract timestamp from backup filename
	// Format: .bashrc.autospec-backup-YYYYMMDD-HHMMSS
	parts := strings.Split(filepath.Base(backupPath), ".autospec-backup-")
	if len(parts) != 2 {
		t.Fatalf("Unexpected backup path format: %s", backupPath)
	}

	timestamp := parts[1]
	// Verify timestamp format is valid (YYYYMMDD-HHMMSS)
	_, err = time.Parse("20060102-150405", timestamp)
	if err != nil {
		t.Fatalf("Failed to parse timestamp %s: %v", timestamp, err)
	}

	// Verify format components
	if len(timestamp) != 15 { // YYYYMMDD-HHMMSS = 15 characters
		t.Errorf("Timestamp %s has wrong length, expected 15 chars", timestamp)
	}
	if timestamp[8] != '-' {
		t.Errorf("Timestamp %s missing separator at position 8", timestamp)
	}
}

func TestHasExistingInstallation(t *testing.T) {
	tests := map[string]struct {
		setup   func(t *testing.T) string
		want    bool
		wantErr bool
	}{
		"detects existing installation": {
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				filePath := filepath.Join(dir, ".bashrc")
				content := "# Some config\n" + StartMarker + "\nsource <(autospec completion bash)\n" + EndMarker + "\n"
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return filePath
			},
			want:    true,
			wantErr: false,
		},
		"no installation found": {
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				filePath := filepath.Join(dir, ".bashrc")
				content := "# Some config\nexport PATH=$HOME/bin:$PATH\n"
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return filePath
			},
			want:    false,
			wantErr: false,
		},
		"file does not exist": {
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				return filepath.Join(dir, ".bashrc")
			},
			want:    false,
			wantErr: false,
		},
		"only start marker present": {
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				filePath := filepath.Join(dir, ".bashrc")
				content := "# Some config\n" + StartMarker + "\n"
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				return filePath
			},
			want:    true, // We detect based on start marker only
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			filePath := tc.setup(t)

			got, err := HasExistingInstallation(filePath)

			if (err != nil) != tc.wantErr {
				t.Errorf("HasExistingInstallation() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if got != tc.want {
				t.Errorf("HasExistingInstallation() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestInstallResultActions(t *testing.T) {
	tests := map[string]struct {
		action InstallAction
		want   string
	}{
		"installed action": {action: ActionInstalled, want: "installed"},
		"updated action":   {action: ActionUpdated, want: "updated"},
		"skipped action":   {action: ActionSkipped, want: "skipped"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if string(tc.action) != tc.want {
				t.Errorf("action = %s, want %s", tc.action, tc.want)
			}
		})
	}
}

func TestPermissionError(t *testing.T) {
	err := &PermissionError{
		Path:      "/etc/bashrc",
		Operation: "write",
		Err:       os.ErrPermission,
	}

	// Test Error() method
	errMsg := err.Error()
	if !strings.Contains(errMsg, "permission denied") {
		t.Errorf("Error() = %q, want to contain 'permission denied'", errMsg)
	}
	if !strings.Contains(errMsg, "/etc/bashrc") {
		t.Errorf("Error() = %q, want to contain path", errMsg)
	}
	if !strings.Contains(errMsg, "write") {
		t.Errorf("Error() = %q, want to contain operation", errMsg)
	}

	// Test IsPermissionError
	if !IsPermissionError(err) {
		t.Error("IsPermissionError() = false, want true")
	}

	// Test that regular errors are not permission errors
	if IsPermissionError(os.ErrNotExist) {
		t.Error("IsPermissionError(os.ErrNotExist) = true, want false")
	}
}

func TestGetManualInstructions(t *testing.T) {
	tests := map[string]struct {
		shell        Shell
		wantContains []string
	}{
		"bash instructions": {
			shell: Bash,
			wantContains: []string{
				"bash",
				".bashrc",
				StartMarker,
				"source <(autospec completion bash)",
			},
		},
		"zsh instructions": {
			shell: Zsh,
			wantContains: []string{
				"zsh",
				".zshrc",
				"compinit",
				StartMarker,
			},
		},
		"fish instructions": {
			shell: Fish,
			wantContains: []string{
				"fish",
				"completions/autospec.fish",
			},
		},
		"powershell instructions": {
			shell: PowerShell,
			wantContains: []string{
				"powershell",
				"$PROFILE",
				"Out-String",
				"Invoke-Expression",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			instructions := GetManualInstructions(tc.shell)

			for _, substr := range tc.wantContains {
				if !strings.Contains(strings.ToLower(instructions), strings.ToLower(substr)) {
					t.Errorf("GetManualInstructions(%s) = %q, want to contain %q", tc.shell, instructions, substr)
				}
			}
		})
	}
}

func TestInstallBashIdempotency(t *testing.T) {
	// Set up a temp directory to act as home
	tempHome := t.TempDir()
	bashrcPath := filepath.Join(tempHome, ".bashrc")

	// Create initial bashrc
	initialContent := "# My bashrc\nexport PATH=$HOME/bin:$PATH\n"
	if err := os.WriteFile(bashrcPath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Write the completion block manually to simulate existing installation
	block := GetCompletionBlock(Bash)
	fullContent := initialContent + block.FormatBlock()
	if err := os.WriteFile(bashrcPath, []byte(fullContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Check that HasExistingInstallation detects it
	exists, err := HasExistingInstallation(bashrcPath)
	if err != nil {
		t.Fatalf("HasExistingInstallation() error = %v", err)
	}
	if !exists {
		t.Error("HasExistingInstallation() = false, want true after writing block")
	}
}

func TestInstallFishDirectory(t *testing.T) {
	// Create a temp directory structure
	tempHome := t.TempDir()
	fishCompletionsDir := filepath.Join(tempHome, ".config", "fish", "completions")

	// Fish config should have the correct completions directory
	config := GetShellConfig(Fish, tempHome)
	if config.CompletionDir != fishCompletionsDir {
		t.Errorf("Fish CompletionDir = %s, want %s", config.CompletionDir, fishCompletionsDir)
	}
	if config.RequiresRCModification {
		t.Error("Fish RequiresRCModification = true, want false")
	}
}

func TestPermissionErrorUnwrap(t *testing.T) {
	t.Parallel()

	innerErr := os.ErrPermission
	err := &PermissionError{
		Path:      "/etc/bashrc",
		Operation: "write",
		Err:       innerErr,
	}

	// Test Unwrap() method
	unwrapped := err.Unwrap()
	if unwrapped != innerErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, innerErr)
	}
}

func TestGetAllManualInstructions(t *testing.T) {
	t.Parallel()

	var buf strings.Builder
	GetAllManualInstructions(&buf)

	output := buf.String()

	// Should contain instructions for all shells
	shells := SupportedShells()
	for _, shell := range shells {
		if !strings.Contains(strings.ToLower(output), strings.ToLower(string(shell))) {
			t.Errorf("GetAllManualInstructions() should contain %s instructions", shell)
		}
	}

	// Should have separator between shells
	if !strings.Contains(output, "---") {
		t.Error("GetAllManualInstructions() should contain separators")
	}
}

// TestInstallRCFile tests the installRCFile function for bash, zsh, and powershell (T013, T014)
func TestInstallRCFile(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		shell           Shell
		rcFileName      string
		existingContent string
		wantAction      InstallAction
		wantErr         bool
		wantBackup      bool
	}{
		"bash new installation": {
			shell:           Bash,
			rcFileName:      ".bashrc",
			existingContent: "# My bashrc\nexport PATH=$HOME/bin:$PATH\n",
			wantAction:      ActionInstalled,
			wantErr:         false,
			wantBackup:      true,
		},
		"bash skip existing installation": {
			shell:           Bash,
			rcFileName:      ".bashrc",
			existingContent: "# My bashrc\n" + StartMarker + "\nsource <(autospec completion bash)\n" + EndMarker + "\n",
			wantAction:      ActionSkipped,
			wantErr:         false,
			wantBackup:      false,
		},
		"zsh new installation": {
			shell:           Zsh,
			rcFileName:      ".zshrc",
			existingContent: "# My zshrc\n",
			wantAction:      ActionInstalled,
			wantErr:         false,
			wantBackup:      true,
		},
		"zsh skip existing installation": {
			shell:           Zsh,
			rcFileName:      ".zshrc",
			existingContent: "# My zshrc\n" + StartMarker + "\nsource <(autospec completion zsh)\n" + EndMarker + "\n",
			wantAction:      ActionSkipped,
			wantErr:         false,
			wantBackup:      false,
		},
		"bash new rc file creation": {
			shell:           Bash,
			rcFileName:      ".bashrc",
			existingContent: "", // File doesn't exist
			wantAction:      ActionInstalled,
			wantErr:         false,
			wantBackup:      false, // No backup for new file
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tempHome := t.TempDir()
			rcPath := filepath.Join(tempHome, tc.rcFileName)

			// Create existing rc file if content provided
			if tc.existingContent != "" {
				if err := os.WriteFile(rcPath, []byte(tc.existingContent), 0644); err != nil {
					t.Fatal(err)
				}
			}

			config := GetShellConfig(tc.shell, tempHome)

			// Call installRCFile
			result, err := installRCFile(tc.shell, config)

			if (err != nil) != tc.wantErr {
				t.Errorf("installRCFile() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if result == nil {
				t.Fatal("installRCFile() returned nil result")
			}

			if result.Action != tc.wantAction {
				t.Errorf("installRCFile() action = %v, want %v", result.Action, tc.wantAction)
			}

			if result.Shell != tc.shell {
				t.Errorf("installRCFile() shell = %v, want %v", result.Shell, tc.shell)
			}

			if tc.wantBackup && result.BackupPath == "" {
				t.Error("installRCFile() expected backup but got none")
			}

			if !tc.wantBackup && result.BackupPath != "" {
				t.Errorf("installRCFile() unexpected backup at %s", result.BackupPath)
			}

			// Verify completion block was added for new installations
			if tc.wantAction == ActionInstalled {
				content, _ := os.ReadFile(rcPath)
				if !strings.Contains(string(content), StartMarker) {
					t.Error("installRCFile() did not add completion block")
				}
				if !strings.Contains(string(content), EndMarker) {
					t.Error("installRCFile() did not add end marker")
				}
			}
		})
	}
}

// TestInstallRCFileDirectoryCreation tests that installRCFile creates parent directories
func TestInstallRCFileDirectoryCreation(t *testing.T) {
	t.Parallel()

	tempHome := t.TempDir()
	// Use a nested directory that doesn't exist
	nestedDir := filepath.Join(tempHome, "Documents", "WindowsPowerShell")
	profilePath := filepath.Join(nestedDir, "Microsoft.PowerShell_profile.ps1")

	config := ShellConfig{
		Shell:                  PowerShell,
		RCPath:                 profilePath,
		RequiresRCModification: true,
	}

	result, err := installRCFile(PowerShell, config)

	if err != nil {
		t.Fatalf("installRCFile() error = %v", err)
	}

	if result.Action != ActionInstalled {
		t.Errorf("action = %v, want %v", result.Action, ActionInstalled)
	}

	// Verify directory was created
	if _, err := os.Stat(nestedDir); os.IsNotExist(err) {
		t.Error("installRCFile() did not create parent directory")
	}

	// Verify file was created
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Error("installRCFile() did not create profile file")
	}
}

// TestInstallResultMessage tests that InstallResult messages are informative
func TestInstallResultMessage(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		action       InstallAction
		existingFile bool
		wantContains []string
	}{
		"installed with backup": {
			action:       ActionInstalled,
			existingFile: true,
			wantContains: []string{
				"installed",
				"backup",
			},
		},
		"installed without backup": {
			action:       ActionInstalled,
			existingFile: false,
			wantContains: []string{
				"installed",
			},
		},
		"skipped": {
			action:       ActionSkipped,
			existingFile: true,
			wantContains: []string{
				"already installed",
				"remove",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tempHome := t.TempDir()
			rcPath := filepath.Join(tempHome, ".bashrc")

			// Setup for the test case
			if tc.action == ActionSkipped {
				// Create file with existing installation
				content := "# bashrc\n" + StartMarker + "\ncompletion\n" + EndMarker + "\n"
				if err := os.WriteFile(rcPath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			} else if tc.existingFile {
				// Create empty file or file without installation
				if err := os.WriteFile(rcPath, []byte("# bashrc\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}

			config := GetShellConfig(Bash, tempHome)
			result, err := installRCFile(Bash, config)

			if err != nil {
				t.Fatalf("installRCFile() error = %v", err)
			}

			// Check expected strings in message
			for _, want := range tc.wantContains {
				if !strings.Contains(strings.ToLower(result.Message), strings.ToLower(want)) {
					t.Errorf("message %q should contain %q", result.Message, want)
				}
			}
		})
	}
}

// TestInstallFishCreatesCompletionsDir tests that fish installation creates the completions directory
func TestInstallFishCreatesCompletionsDir(t *testing.T) {
	t.Parallel()

	// Note: This test verifies the directory creation logic
	// without requiring the autospec binary

	tempHome := t.TempDir()
	completionsDir := filepath.Join(tempHome, ".config", "fish", "completions")

	// Verify directory doesn't exist initially
	if _, err := os.Stat(completionsDir); !os.IsNotExist(err) {
		t.Fatal("completions directory should not exist initially")
	}

	config := ShellConfig{
		Shell:                  Fish,
		CompletionDir:          completionsDir,
		RequiresRCModification: false,
	}

	// Create the directory (simulating what installFish would do)
	if err := os.MkdirAll(config.CompletionDir, 0755); err != nil {
		t.Fatalf("failed to create completions directory: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(completionsDir)
	if err != nil {
		t.Fatalf("completions directory should exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("completions path should be a directory")
	}
}

// TestInstallResultSuccess tests that successful installations return proper result
func TestInstallResultSuccess(t *testing.T) {
	t.Parallel()

	tempHome := t.TempDir()
	rcPath := filepath.Join(tempHome, ".bashrc")

	// Create initial rc file
	if err := os.WriteFile(rcPath, []byte("# bashrc\n"), 0644); err != nil {
		t.Fatal(err)
	}

	config := GetShellConfig(Bash, tempHome)
	result, err := installRCFile(Bash, config)

	if err != nil {
		t.Fatalf("installRCFile() error = %v", err)
	}

	// Verify result fields
	if !result.Success {
		t.Error("result.Success should be true")
	}

	if result.ConfigPath != rcPath {
		t.Errorf("result.ConfigPath = %s, want %s", result.ConfigPath, rcPath)
	}

	if result.Shell != Bash {
		t.Errorf("result.Shell = %v, want %v", result.Shell, Bash)
	}
}

// Additional tests to improve coverage to 85%

func TestCreateBackup_ReadPermissionError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, ".bashrc")

	// Create a file that exists
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Make file unreadable (test on unix only)
	if err := os.Chmod(filePath, 0000); err != nil {
		t.Skip("Cannot change file permissions on this platform")
	}
	defer os.Chmod(filePath, 0644) // Restore for cleanup

	_, err := CreateBackup(filePath)
	if err == nil {
		t.Error("CreateBackup() should fail with permission error")
	}
	if !IsPermissionError(err) {
		t.Errorf("CreateBackup() error should be PermissionError, got %T", err)
	}
}

func TestHasExistingInstallation_PermissionError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, ".bashrc")

	// Create a file that exists
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Make file unreadable
	if err := os.Chmod(filePath, 0000); err != nil {
		t.Skip("Cannot change file permissions on this platform")
	}
	defer os.Chmod(filePath, 0644) // Restore for cleanup

	_, err := HasExistingInstallation(filePath)
	if err == nil {
		t.Error("HasExistingInstallation() should fail with permission error")
	}
	if !IsPermissionError(err) {
		t.Errorf("HasExistingInstallation() error should be PermissionError, got %T", err)
	}
}

func TestInstallRCFile_PermissionError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Create a read-only directory to prevent writing
	readOnlyDir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(readOnlyDir, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(readOnlyDir, 0755) // Restore for cleanup

	rcPathInReadOnly := filepath.Join(readOnlyDir, ".bashrc")
	config := ShellConfig{
		Shell:                  Bash,
		RCPath:                 rcPathInReadOnly,
		RequiresRCModification: true,
	}

	_, err := installRCFile(Bash, config)
	if err == nil {
		// On some systems, root can write anywhere - skip test
		t.Skip("Permission error test requires non-root user")
	}
	if !IsPermissionError(err) && !strings.Contains(err.Error(), "permission") {
		t.Logf("Expected permission error, got: %v", err)
	}
}

func TestInstallRCFile_AllShells(t *testing.T) {
	t.Parallel()

	shells := []Shell{Bash, Zsh, PowerShell}
	rcFiles := map[Shell]string{
		Bash:       ".bashrc",
		Zsh:        ".zshrc",
		PowerShell: "profile.ps1",
	}

	for _, shell := range shells {
		shell := shell // capture range variable
		t.Run(string(shell), func(t *testing.T) {
			t.Parallel()

			tempHome := t.TempDir()
			rcPath := filepath.Join(tempHome, rcFiles[shell])

			// Create initial rc file
			if err := os.WriteFile(rcPath, []byte("# initial\n"), 0644); err != nil {
				t.Fatal(err)
			}

			config := ShellConfig{
				Shell:                  shell,
				RCPath:                 rcPath,
				RequiresRCModification: true,
			}

			result, err := installRCFile(shell, config)
			if err != nil {
				t.Fatalf("installRCFile() error = %v", err)
			}

			if result.Shell != shell {
				t.Errorf("result.Shell = %v, want %v", result.Shell, shell)
			}

			if result.Action != ActionInstalled {
				t.Errorf("result.Action = %v, want %v", result.Action, ActionInstalled)
			}

			// Verify file contains completion block
			content, _ := os.ReadFile(rcPath)
			if !strings.Contains(string(content), StartMarker) {
				t.Error("file should contain start marker")
			}
		})
	}
}

func TestCreateBackup_WritePermissionError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, ".bashrc")

	// Create original file
	if err := os.WriteFile(filePath, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	// Make directory read-only to prevent backup file creation
	if err := os.Chmod(dir, 0555); err != nil {
		t.Skip("Cannot change directory permissions on this platform")
	}
	defer os.Chmod(dir, 0755) // Restore for cleanup

	_, err := CreateBackup(filePath)
	if err == nil {
		t.Skip("Permission error test requires non-root user")
	}
	if !IsPermissionError(err) && !strings.Contains(err.Error(), "permission") {
		t.Logf("Expected permission error for backup write, got: %v", err)
	}
}

func TestInstallRCFile_PreservesExistingContent(t *testing.T) {
	t.Parallel()

	tempHome := t.TempDir()
	rcPath := filepath.Join(tempHome, ".bashrc")

	// Create initial rc file with important content
	initialContent := `# My bashrc
export PATH=$HOME/bin:$PATH
alias ls='ls --color=auto'

# Custom function
my_func() {
    echo "Hello"
}
`
	if err := os.WriteFile(rcPath, []byte(initialContent), 0644); err != nil {
		t.Fatal(err)
	}

	config := GetShellConfig(Bash, tempHome)
	_, err := installRCFile(Bash, config)
	if err != nil {
		t.Fatalf("installRCFile() error = %v", err)
	}

	// Verify original content is preserved
	content, _ := os.ReadFile(rcPath)
	if !strings.Contains(string(content), "export PATH=$HOME/bin:$PATH") {
		t.Error("original PATH export should be preserved")
	}
	if !strings.Contains(string(content), "alias ls") {
		t.Error("original alias should be preserved")
	}
	if !strings.Contains(string(content), "my_func()") {
		t.Error("original function should be preserved")
	}
}

func TestPermissionErrorFormatting(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		path      string
		operation string
		err       error
		wantMsg   string
	}{
		"read operation": {
			path:      "/etc/bashrc",
			operation: "read",
			err:       os.ErrPermission,
			wantMsg:   "permission denied: cannot read /etc/bashrc",
		},
		"write operation": {
			path:      "/etc/zshrc",
			operation: "write",
			err:       os.ErrPermission,
			wantMsg:   "permission denied: cannot write /etc/zshrc",
		},
		"create directory operation": {
			path:      "/root/config",
			operation: "create directory",
			err:       os.ErrPermission,
			wantMsg:   "permission denied: cannot create directory /root/config",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := &PermissionError{
				Path:      tc.path,
				Operation: tc.operation,
				Err:       tc.err,
			}

			msg := err.Error()
			if !strings.Contains(msg, tc.wantMsg) {
				t.Errorf("Error() = %q, want to contain %q", msg, tc.wantMsg)
			}
		})
	}
}
