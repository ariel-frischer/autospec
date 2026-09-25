// Package git provides Git repository utilities for autospec including branch detection,
// repository validation, and branch management. It uses go-git library for core operations
// (branch detection, repo validation, fetch) while falling back to git CLI only for
// operations not supported by go-git (worktree management).
package git

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// debugLogger is a function that logs debug messages when debug mode is enabled.
// By default, it's a no-op. Set it via SetDebugLogger to enable debug output.
var debugLogger func(format string, args ...any)

// SetDebugLogger configures the debug logger for git operations.
// Pass nil to disable debug logging. The logger function should format
// and output the message (similar to log.Printf signature).
func SetDebugLogger(logger func(format string, args ...any)) {
	debugLogger = logger
}

// logDebug logs a debug message if the debug logger is set.
func logDebug(format string, args ...any) {
	if debugLogger != nil {
		debugLogger(format, args...)
	}
}

// openRepo opens a git repository at the specified path or current working directory.
// It uses go-git's PlainOpenWithOptions with DetectDotGit enabled to traverse
// up the directory tree to find the repository root.
// If path is empty, the current working directory is used.
func openRepo(path string) (*git.Repository, error) {
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current directory: %w", err)
		}
	}

	logDebug("[git] opening repository at %s", path)

	repo, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{
		DetectDotGit:          true,
		EnableDotGitCommonDir: true,
	})
	if err != nil {
		return nil, fmt.Errorf("opening repository at %s: %w", path, err)
	}

	logDebug("[git] repository opened successfully")
	return repo, nil
}

// GetCurrentBranch returns the name of the current git branch.
// Returns empty string if in detached HEAD state.
func GetCurrentBranch() (string, error) {
	repo, err := openRepo("")
	if err != nil {
		return "", err
	}

	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("getting HEAD reference: %w", err)
	}

	// Check if in detached HEAD state
	if !head.Name().IsBranch() {
		logDebug("[git] GetCurrentBranch: detached HEAD state")
		return "", nil
	}

	branch := head.Name().Short()
	logDebug("[git] GetCurrentBranch: %s", branch)
	return branch, nil
}

// GetRepositoryRoot returns the absolute path to the repository root.
func GetRepositoryRoot() (string, error) {
	repo, err := openRepo("")
	if err != nil {
		return "", err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("getting worktree: %w", err)
	}

	root := worktree.Filesystem.Root()
	logDebug("[git] GetRepositoryRoot: %s", root)
	return root, nil
}

// IsGitRepository checks if the current directory is within a git repository.
func IsGitRepository() bool {
	_, err := openRepo("")
	result := err == nil
	logDebug("[git] IsGitRepository: %v", result)
	return result
}

// BranchInfo contains metadata about a git branch
type BranchInfo struct {
	Name     string
	IsRemote bool
	Remote   string // Remote name (e.g., "origin") if IsRemote is true
}

// GetAllBranches returns a list of all local and remote branches.
// Filters out HEAD pointers and deduplicates (local preferred over remote).
func GetAllBranches() ([]BranchInfo, error) {
	repo, err := openRepo("")
	if err != nil {
		// Not a git repository - return nil like original
		return nil, nil
	}

	seen := make(map[string]bool)
	var branches []BranchInfo

	// Collect local branches
	branches, err = collectLocalBranches(repo, branches, seen)
	if err != nil {
		return nil, err
	}

	// Collect remote branches
	branches, err = collectRemoteBranches(repo, branches, seen)
	if err != nil {
		return nil, err
	}

	sort.Slice(branches, func(i, j int) bool {
		return branches[i].Name < branches[j].Name
	})

	logDebug("[git] GetAllBranches: found %d branches", len(branches))
	return branches, nil
}

// collectLocalBranches iterates local branches and adds them to the list.
func collectLocalBranches(repo *git.Repository, branches []BranchInfo, seen map[string]bool) ([]BranchInfo, error) {
	branchIter, err := repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("listing local branches: %w", err)
	}

	err = branchIter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		if name == "HEAD" || strings.Contains(name, "HEAD") {
			return nil
		}
		info := BranchInfo{Name: name, IsRemote: false}
		branches = addBranchWithDedup(branches, info, seen)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating local branches: %w", err)
	}

	return branches, nil
}

// collectRemoteBranches iterates remote-tracking branches and adds them to the list.
func collectRemoteBranches(repo *git.Repository, branches []BranchInfo, seen map[string]bool) ([]BranchInfo, error) {
	refIter, err := repo.References()
	if err != nil {
		return nil, fmt.Errorf("listing references: %w", err)
	}

	err = refIter.ForEach(func(ref *plumbing.Reference) error {
		if !ref.Name().IsRemote() {
			return nil
		}

		fullName := ref.Name().Short() // e.g., "origin/main"
		if strings.Contains(fullName, "HEAD") {
			return nil
		}

		parts := strings.SplitN(fullName, "/", 2)
		if len(parts) != 2 {
			return nil
		}

		info := BranchInfo{
			Name:     parts[1],
			IsRemote: true,
			Remote:   parts[0],
		}
		branches = addBranchWithDedup(branches, info, seen)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating remote branches: %w", err)
	}

	return branches, nil
}

// collectBranches parses branch lines and deduplicates them.
// Uses seen map for O(1) duplicate detection. When duplicate found,
// prefers local branch over remote (replaces remote with local in-place).
func collectBranches(lines []string) []BranchInfo {
	seen := make(map[string]bool)
	var branches []BranchInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "HEAD") {
			continue
		}

		info := parseBranchLine(line)
		if info == nil {
			continue
		}

		branches = addBranchWithDedup(branches, *info, seen)
	}

	return branches
}

// parseBranchLine parses a single branch line into BranchInfo.
// Handles three formats:
//   - "remotes/origin/main" → Remote="origin", Name="main", IsRemote=true
//   - "origin/main" → Remote="origin", Name="main", IsRemote=true
//   - "main" → Name="main", IsRemote=false
func parseBranchLine(line string) *BranchInfo {
	var info BranchInfo

	if strings.HasPrefix(line, "remotes/") {
		line = strings.TrimPrefix(line, "remotes/")
		parts := strings.SplitN(line, "/", 2)
		if len(parts) != 2 {
			return nil
		}
		info.Remote = parts[0]
		info.Name = parts[1]
		info.IsRemote = true
	} else if strings.Contains(line, "/") {
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

	return &info
}

// addBranchWithDedup adds a branch, handling duplicates (prefer local over remote).
// If branch name already seen and new branch is local, replaces the existing
// remote branch in-place via linear scan. Otherwise appends if not seen.
func addBranchWithDedup(branches []BranchInfo, info BranchInfo, seen map[string]bool) []BranchInfo {
	key := info.Name

	if seen[key] && !info.IsRemote {
		// Replace remote with local
		for i, b := range branches {
			if b.Name == info.Name && b.IsRemote {
				branches[i] = info
				break
			}
		}
		return branches
	}

	if seen[key] {
		return branches
	}

	seen[key] = true
	return append(branches, info)
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

// CreateBranch creates a new git branch and checks it out.
// Returns an error if the branch already exists or if not in a git repository.
func CreateBranch(name string) error {
	repo, err := openRepo("")
	if err != nil {
		return fmt.Errorf("not a git repository")
	}

	// Check if branch already exists
	if err := checkBranchExists(repo, name); err != nil {
		return err
	}

	// Get HEAD reference to use as the starting point
	head, err := repo.Head()
	if err != nil {
		return fmt.Errorf("getting HEAD: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}

	// Create and checkout the branch
	// Keep: true preserves untracked files/directories (like .autospec/)
	// Without Keep, go-git deletes untracked content during checkout
	branchRef := plumbing.NewBranchReferenceName(name)
	err = worktree.Checkout(&git.CheckoutOptions{
		Hash:   head.Hash(),
		Branch: branchRef,
		Create: true,
		Keep:   true,
	})
	if err != nil {
		return fmt.Errorf("creating branch '%s': %w", name, err)
	}

	logDebug("[git] CreateBranch: created and checked out %s", name)
	return nil
}

// checkBranchExists returns an error if the branch already exists.
func checkBranchExists(repo *git.Repository, name string) error {
	branchRef := plumbing.NewBranchReferenceName(name)
	_, err := repo.Reference(branchRef, false)
	if err == nil {
		return fmt.Errorf("branch '%s' already exists", name)
	}
	if err != plumbing.ErrReferenceNotFound {
		return fmt.Errorf("checking branch existence: %w", err)
	}
	return nil
}

// FetchAllRemotes fetches from all configured remotes with default timeout.
// It continues on failure and returns true if all fetches succeeded.
// Network failures are handled gracefully (returns false but no error for transient failures).
// Uses SSH agent auth for SSH remotes and environment credentials for HTTPS remotes.
// Uses DefaultFetchTimeout (60s) to prevent indefinite hangs (FR-008, NFR-001).
func FetchAllRemotes() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultFetchTimeout)
	defer cancel()
	return FetchAllRemotesWithContext(ctx)
}

// getAuthForURL returns the appropriate authentication method for a remote URL.
// SSH URLs use SSH agent auth, HTTPS URLs use environment credentials.
func getAuthForURL(url string) transport.AuthMethod {
	if isSSHURL(url) {
		auth, err := ssh.NewSSHAgentAuth("git")
		if err != nil {
			logDebug("[git] SSH agent auth failed: %v", err)
			return nil
		}
		return auth
	}

	// For HTTPS, try environment credentials
	username := os.Getenv("GIT_USERNAME")
	password := os.Getenv("GIT_PASSWORD")
	if username == "" {
		username = os.Getenv("GITHUB_TOKEN")
		if username != "" {
			password = "" // GitHub token can be used as username with empty password
		}
	}

	if username != "" {
		return &http.BasicAuth{
			Username: username,
			Password: password,
		}
	}

	return nil
}

// isSSHURL checks if a URL is an SSH URL.
// Detects git@ (SCP-style), ssh://, and git+ssh:// schemes.
func isSSHURL(url string) bool {
	return strings.HasPrefix(url, "git@") ||
		strings.HasPrefix(url, "ssh://") ||
		strings.HasPrefix(url, "git+ssh://")
}

// isSSHAgentAvailable checks if an SSH agent is available.
// Returns true only if SSH_AUTH_SOCK is set and non-empty.
func isSSHAgentAvailable() bool {
	sock := strings.TrimSpace(os.Getenv("SSH_AUTH_SOCK"))
	return sock != ""
}

// DefaultFetchTimeout is the default timeout for fetch operations (FR-008).
const DefaultFetchTimeout = 60 * time.Second

// FetchAllRemotesWithContext fetches from all configured remotes with context support.
// The context can be used for timeout/cancellation to prevent indefinite hangs (NFR-001).
// Returns true if all fetches succeeded, false if any failed.
// Timeout errors are handled gracefully with a warning log (not an error).
func FetchAllRemotesWithContext(ctx context.Context) (bool, error) {
	// Check for context cancellation before starting
	if err := ctx.Err(); err != nil {
		logDebug("[git] FetchAllRemotesWithContext: context already cancelled")
		return true, nil
	}

	repo, err := openRepo("")
	if err != nil {
		return false, nil
	}

	remotes, err := repo.Remotes()
	if err != nil {
		logDebug("[git] FetchAllRemotesWithContext: no remotes: %v", err)
		return true, nil
	}

	if len(remotes) == 0 {
		logDebug("[git] FetchAllRemotesWithContext: no remotes configured")
		return true, nil
	}

	allSucceeded := true
	for _, remote := range remotes {
		if err := ctx.Err(); err != nil {
			logDebug("[git] FetchAllRemotesWithContext: context cancelled, stopping fetch")
			return allSucceeded, nil
		}
		if err := fetchRemoteWithContext(ctx, repo, remote); err != nil {
			fmt.Fprintf(os.Stderr, "[git] Warning: failed to fetch from remote '%s': %v\n", remote.Config().Name, err)
			allSucceeded = false
		}
	}

	logDebug("[git] FetchAllRemotesWithContext: completed, all succeeded: %v", allSucceeded)
	return allSucceeded, nil
}

// fetchRemoteWithContext fetches from a single remote with context and authentication.
// Skips SSH remotes when no SSH agent is available. Handles timeout gracefully.
func fetchRemoteWithContext(ctx context.Context, repo *git.Repository, remote *git.Remote) error {
	remoteConfig := remote.Config()
	if len(remoteConfig.URLs) == 0 {
		return nil
	}

	url := remoteConfig.URLs[0]

	// Skip SSH URLs when SSH agent is not available (FR-001, FR-004, FR-005)
	if isSSHURL(url) && !isSSHAgentAvailable() {
		logDebug("[git] skipping fetch from remote '%s': SSH URL without SSH agent available", remoteConfig.Name)
		return nil
	}

	auth := getAuthForURL(url)
	logDebug("[git] fetching from remote '%s' (%s)", remoteConfig.Name, url)

	err := repo.FetchContext(ctx, &git.FetchOptions{
		RemoteName: remoteConfig.Name,
		Auth:       auth,
		Prune:      true,
		RefSpecs:   []config.RefSpec{config.RefSpec("+refs/heads/*:refs/remotes/" + remoteConfig.Name + "/*")},
	})

	// Handle context cancellation/timeout gracefully (FR-008)
	if ctx.Err() != nil {
		logDebug("[git] fetch from remote '%s' timed out or cancelled", remoteConfig.Name)
		return nil
	}

	// "already up-to-date" is not an error
	if err == git.NoErrAlreadyUpToDate {
		return nil
	}

	return err
}
