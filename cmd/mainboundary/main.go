// Command mainboundary rejects a main candidate if video/ is present in its
// HEAD tree or in any commit reachable from HEAD.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w", args[0], err)
	}
	return string(output), nil
}

func runGuard(dir string) error {
	root, err := gitOutput(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("finding candidate repository: %w", err)
	}
	absoluteDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving candidate directory: %w", err)
	}
	if strings.TrimSpace(root) != absoluteDir {
		return fmt.Errorf("candidate directory is not a Git repository root")
	}
	shallow, err := gitOutput(dir, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return fmt.Errorf("checking repository history: %w", err)
	}
	if strings.TrimSpace(shallow) != "false" {
		return fmt.Errorf("full Git ancestry required (shallow or unknown repository state)")
	}

	path, err := gitOutput(dir, "ls-tree", "-d", "--name-only", "HEAD", "--", "video")
	if err != nil {
		return fmt.Errorf("inspecting candidate HEAD tree: %w", err)
	}
	if strings.TrimSpace(path) != "" {
		return fmt.Errorf("video/ exists in HEAD tree")
	}
	commit, err := gitOutput(dir, "rev-list", "--full-history", "--max-count=1", "HEAD", "--", "video/")
	if err != nil {
		return fmt.Errorf("checking HEAD path history: %w", err)
	}
	if strings.TrimSpace(commit) != "" {
		return fmt.Errorf("video/ exists in HEAD ancestry at commit %s", strings.TrimSpace(commit))
	}
	return nil
}

func main() {
	if err := runGuard("."); err != nil {
		fmt.Fprintln(os.Stderr, "main boundary:", err)
		os.Exit(1)
	}
	fmt.Println("main boundary: no video/ in HEAD ancestry")
}
