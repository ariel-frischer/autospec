package git

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// GetCurrentBranch returns the name of the current git branch
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetRepositoryRoot returns the absolute path to the repository root
func GetRepositoryRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// IsGitRepository checks if the current directory is within a git repository
func IsGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// BranchInfo contains metadata about a git branch
type BranchInfo struct {
	Name     string
	IsRemote bool
	Remote   string // Remote name (e.g., "origin") if IsRemote is true
}

// GetAllBranches returns a list of all local and remote branches
// It filters out HEAD pointers and duplicates
func GetAllBranches() ([]BranchInfo, error) {
	if !IsGitRepository() {
		return nil, nil // Return empty list for non-git repos
	}

	cmd := exec.Command("git", "branch", "-a", "--format=%(refname:short)")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	seen := make(map[string]bool)
	var branches []BranchInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip HEAD pointers
		if strings.Contains(line, "HEAD") {
			continue
		}

		var info BranchInfo

		// Check if it's a remote branch
		if strings.HasPrefix(line, "remotes/") {
			line = strings.TrimPrefix(line, "remotes/")
			parts := strings.SplitN(line, "/", 2)
			if len(parts) == 2 {
				info.Remote = parts[0]
				info.Name = parts[1]
				info.IsRemote = true
			} else {
				continue
			}
		} else if strings.Contains(line, "/") {
			// Handle format like "origin/branch-name" without remotes/ prefix
			parts := strings.SplitN(line, "/", 2)
			if len(parts) == 2 {
				info.Remote = parts[0]
				info.Name = parts[1]
				info.IsRemote = true
			} else {
				info.Name = line
			}
		} else {
			info.Name = line
			info.IsRemote = false
		}

		// Skip duplicates (prefer local over remote)
		key := info.Name
		if seen[key] && !info.IsRemote {
			// Replace remote with local
			for i, b := range branches {
				if b.Name == info.Name && b.IsRemote {
					branches[i] = info
					break
				}
			}
			continue
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		branches = append(branches, info)
	}

	// Sort branches by name for consistent output
	sort.Slice(branches, func(i, j int) bool {
		return branches[i].Name < branches[j].Name
	})

	return branches, nil
}

// GetBranchNames returns just the names of all branches (local and remote, deduplicated)
func GetBranchNames() ([]string, error) {
	branches, err := GetAllBranches()
	if err != nil {
		return nil, err
	}

	names := make([]string, len(branches))
	for i, b := range branches {
		names[i] = b.Name
	}
	return names, nil
}

// CreateBranch creates a new git branch and checks it out
// Returns an error if the branch already exists or if not in a git repository
func CreateBranch(name string) error {
	if !IsGitRepository() {
		return fmt.Errorf("not a git repository")
	}

	// Check if branch already exists
	branches, err := GetBranchNames()
	if err != nil {
		return fmt.Errorf("failed to check existing branches: %w", err)
	}

	for _, b := range branches {
		if b == name {
			return fmt.Errorf("branch '%s' already exists", name)
		}
	}

	// Create and checkout the branch
	cmd := exec.Command("git", "checkout", "-b", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch '%s': %w", name, err)
	}

	return nil
}

// FetchAllRemotes fetches from all configured remotes
// It continues on failure and returns true if all fetches succeeded
// Network failures are handled gracefully (returns false but no error for transient failures)
func FetchAllRemotes() (bool, error) {
	if !IsGitRepository() {
		return false, nil
	}

	// Get list of remotes
	cmd := exec.Command("git", "remote")
	output, err := cmd.Output()
	if err != nil {
		// No remotes configured is not an error
		return true, nil
	}

	remotes := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(remotes) == 0 || (len(remotes) == 1 && remotes[0] == "") {
		return true, nil
	}

	allSucceeded := true
	for _, remote := range remotes {
		remote = strings.TrimSpace(remote)
		if remote == "" {
			continue
		}

		cmd := exec.Command("git", "fetch", "--prune", remote)
		if err := cmd.Run(); err != nil {
			// Log warning to stderr but continue
			fmt.Fprintf(os.Stderr, "[git] Warning: failed to fetch from remote '%s': %v\n", remote, err)
			allSucceeded = false
		}
	}

	return allSucceeded, nil
}
