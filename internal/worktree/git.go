package worktree

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitWorktreeEntry represents a single entry from 'git worktree list'.
type GitWorktreeEntry struct {
	Path   string
	Commit string
	Branch string
}

// GitWorktreeAdd creates a new git worktree using 'git worktree add'.
// If the branch doesn't exist, git will create it from HEAD.
func GitWorktreeAdd(repoPath, worktreePath, branch string) error {
	cmd := exec.Command("git", "worktree", "add", "-b", branch, worktreePath)
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		// If branch already exists, try without -b flag
		if strings.Contains(string(output), "already exists") {
			return gitWorktreeAddExisting(repoPath, worktreePath, branch)
		}
		return fmt.Errorf("git worktree add: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

// gitWorktreeAddExisting adds a worktree for an existing branch.
func gitWorktreeAddExisting(repoPath, worktreePath, branch string) error {
	cmd := exec.Command("git", "worktree", "add", worktreePath, branch)
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree add (existing branch): %w: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

// GitWorktreeRemove removes a git worktree using 'git worktree remove'.
// If force is true, uses --force flag to remove even with uncommitted changes.
func GitWorktreeRemove(repoPath, worktreePath string, force bool) error {
	args := []string{"worktree", "remove", worktreePath}
	if force {
		args = append(args, "--force")
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git worktree remove: %w: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

// GitWorktreeList parses the output of 'git worktree list' and returns entries.
func GitWorktreeList(repoPath string) ([]GitWorktreeEntry, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}

	return parseWorktreeList(output)
}

// parseWorktreeList parses the porcelain output of 'git worktree list'.
func parseWorktreeList(output []byte) ([]GitWorktreeEntry, error) {
	var entries []GitWorktreeEntry
	var current GitWorktreeEntry
	var isBare bool

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "worktree "):
			// Add previous entry if not bare
			if current.Path != "" && !isBare {
				entries = append(entries, current)
			}
			current = GitWorktreeEntry{Path: strings.TrimPrefix(line, "worktree ")}
			isBare = false
		case strings.HasPrefix(line, "HEAD "):
			current.Commit = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		case line == "bare":
			isBare = true
		case line == "detached":
			current.Branch = "(detached)"
		}
	}

	// Add the last entry if not bare
	if current.Path != "" && !isBare {
		entries = append(entries, current)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parsing worktree list: %w", err)
	}

	return entries, nil
}

// HasUncommittedChanges checks if the worktree has uncommitted changes.
func HasUncommittedChanges(worktreePath string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = worktreePath

	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}

	return len(bytes.TrimSpace(output)) > 0, nil
}

// HasUnpushedCommits checks if the worktree has unpushed commits.
func HasUnpushedCommits(worktreePath string) (bool, error) {
	// Get current branch
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = worktreePath

	branchOutput, err := branchCmd.Output()
	if err != nil {
		return false, fmt.Errorf("getting current branch: %w", err)
	}

	branch := strings.TrimSpace(string(branchOutput))
	if branch == "HEAD" {
		// Detached HEAD state - no remote tracking
		return false, nil
	}

	// Check for upstream
	upstreamCmd := exec.Command("git", "rev-parse", "--abbrev-ref", branch+"@{upstream}")
	upstreamCmd.Dir = worktreePath

	if _, err := upstreamCmd.Output(); err != nil {
		// No upstream set - consider as having "unpushed" commits
		return true, nil
	}

	// Count commits ahead of upstream
	logCmd := exec.Command("git", "rev-list", "--count", branch+"@{upstream}..HEAD")
	logCmd.Dir = worktreePath

	logOutput, err := logCmd.Output()
	if err != nil {
		return false, fmt.Errorf("checking unpushed commits: %w", err)
	}

	count := strings.TrimSpace(string(logOutput))
	return count != "0", nil
}

// GetRepoRoot returns the root directory of the git repository.
func GetRepoRoot(path string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting repo root: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// IsWorktree checks if a path is inside a git worktree.
func IsWorktree(path string) (bool, error) {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	cmd.Dir = path

	output, err := cmd.Output()
	if err != nil {
		return false, nil // Not inside a git repo/worktree
	}

	return strings.TrimSpace(string(output)) == "true", nil
}

// GetMainWorktreePath returns the path to the main worktree.
func GetMainWorktreePath(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting git common dir: %w", err)
	}

	gitDir := strings.TrimSpace(string(output))

	// If it ends with .git, the parent is the main worktree
	if filepath.Base(gitDir) == ".git" {
		return filepath.Dir(gitDir), nil
	}

	// For worktrees, the common dir is .git inside the main worktree
	return filepath.Dir(gitDir), nil
}
