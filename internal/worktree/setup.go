package worktree

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// SetupResult contains the result of running a setup script.
type SetupResult struct {
	// Executed indicates whether the script was actually run.
	Executed bool
	// Output contains the combined stdout/stderr from the script.
	Output string
	// Error contains any error that occurred during execution.
	Error error
}

// RunSetupScript executes a setup script with the given parameters.
// The script receives:
//   - Arguments: worktreePath, worktreeName, branchName
//   - Environment: WORKTREE_PATH, WORKTREE_NAME, WORKTREE_BRANCH, SOURCE_REPO
//
// Returns nil (with Executed=false) if the script doesn't exist.
func RunSetupScript(scriptPath, worktreePath, worktreeName, branchName, sourceRepo string, stdout io.Writer) *SetupResult {
	result := &SetupResult{Executed: false}

	if scriptPath == "" {
		return result
	}

	// Make script path absolute if relative
	if !filepath.IsAbs(scriptPath) {
		scriptPath = filepath.Join(sourceRepo, scriptPath)
	}

	// Check if script exists
	info, err := os.Stat(scriptPath)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		result.Error = fmt.Errorf("checking setup script: %w", err)
		return result
	}

	// Check if script is executable
	if info.Mode()&0111 == 0 {
		result.Error = fmt.Errorf("setup script is not executable: %s", scriptPath)
		return result
	}

	result.Executed = true

	cmd := exec.Command(scriptPath, worktreePath, worktreeName, branchName)
	cmd.Dir = worktreePath
	cmd.Env = buildSetupEnv(worktreePath, worktreeName, branchName, sourceRepo)

	var outputBuf bytes.Buffer
	if stdout != nil {
		cmd.Stdout = io.MultiWriter(&outputBuf, stdout)
		cmd.Stderr = io.MultiWriter(&outputBuf, stdout)
	} else {
		cmd.Stdout = &outputBuf
		cmd.Stderr = &outputBuf
	}

	if err := cmd.Run(); err != nil {
		result.Output = outputBuf.String()
		result.Error = fmt.Errorf("running setup script: %w", err)
		return result
	}

	result.Output = outputBuf.String()
	return result
}

// buildSetupEnv creates the environment for the setup script.
func buildSetupEnv(worktreePath, worktreeName, branchName, sourceRepo string) []string {
	env := os.Environ()
	env = append(env,
		"WORKTREE_PATH="+worktreePath,
		"WORKTREE_NAME="+worktreeName,
		"WORKTREE_BRANCH="+branchName,
		"SOURCE_REPO="+sourceRepo,
	)
	return env
}

// CreateDefaultSetupScript creates a template setup script at the given path.
func CreateDefaultSetupScript(path string) error {
	script := `#!/bin/bash
# Worktree setup script
# Arguments: $1 = worktree path, $2 = worktree name, $3 = branch name
# Environment: WORKTREE_PATH, WORKTREE_NAME, WORKTREE_BRANCH, SOURCE_REPO

set -e

WORKTREE_PATH="${1:-$WORKTREE_PATH}"
WORKTREE_NAME="${2:-$WORKTREE_NAME}"
WORKTREE_BRANCH="${3:-$WORKTREE_BRANCH}"

echo "Setting up worktree: $WORKTREE_NAME"
echo "Path: $WORKTREE_PATH"
echo "Branch: $WORKTREE_BRANCH"

cd "$WORKTREE_PATH"

# Add your project-specific setup commands below:
# Examples:
#   npm install
#   go mod download
#   pip install -r requirements.txt

echo "Setup complete!"
`

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating script directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		return fmt.Errorf("writing setup script: %w", err)
	}

	return nil
}
