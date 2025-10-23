package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/anthropics/auto-claude-speckit/internal/git"
)

var (
	// specBranchPattern matches branch names like "002-go-binary-migration"
	specBranchPattern = regexp.MustCompile(`^(\d{3})-(.+)$`)
	// specDirPattern matches directory names like "002-go-binary-migration"
	specDirPattern = regexp.MustCompile(`^(\d{3})-(.+)$`)
)

// Metadata represents information about a feature specification
type Metadata struct {
	Name      string // Feature name (e.g., "go-binary-migration")
	Number    string // Spec number (e.g., "002")
	Directory string // Full path to spec directory
	Branch    string // Git branch name (if in git repo)
}

// DetectCurrentSpec attempts to detect the current spec from git branch or directory
func DetectCurrentSpec(specsDir string) (*Metadata, error) {
	// Strategy 1: Try git branch name
	if git.IsGitRepository() {
		branch, err := git.GetCurrentBranch()
		if err == nil {
			if match := specBranchPattern.FindStringSubmatch(branch); match != nil {
				number := match[1]
				name := match[2]
				directory := filepath.Join(specsDir, fmt.Sprintf("%s-%s", number, name))

				// Verify the directory exists
				if _, err := os.Stat(directory); err == nil {
					return &Metadata{
						Number:    number,
						Name:      name,
						Directory: directory,
						Branch:    branch,
					}, nil
				}
			}
		}
	}

	// Strategy 2: Find most recently modified spec directory
	pattern := filepath.Join(specsDir, "*-*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob spec directories: %w", err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no spec directories found in %s", specsDir)
	}

	// Sort by modification time (most recent first)
	type dirInfo struct {
		path    string
		modTime time.Time
	}

	var dirs []dirInfo
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || !info.IsDir() {
			continue
		}
		dirs = append(dirs, dirInfo{path: match, modTime: info.ModTime()})
	}

	if len(dirs) == 0 {
		return nil, fmt.Errorf("no valid spec directories found in %s", specsDir)
	}

	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].modTime.After(dirs[j].modTime)
	})

	// Parse the most recent directory
	mostRecent := dirs[0].path
	baseName := filepath.Base(mostRecent)
	if match := specDirPattern.FindStringSubmatch(baseName); match != nil {
		return &Metadata{
			Number:    match[1],
			Name:      match[2],
			Directory: mostRecent,
			Branch:    "",
		}, nil
	}

	return nil, fmt.Errorf("could not parse spec directory name: %s", baseName)
}

// GetSpecDirectory returns the full path to a spec directory given its number or name
func GetSpecDirectory(specsDir, specIdentifier string) (string, error) {
	// Try exact match first (e.g., "002-go-binary-migration")
	exactPath := filepath.Join(specsDir, specIdentifier)
	if info, err := os.Stat(exactPath); err == nil && info.IsDir() {
		return exactPath, nil
	}

	// Try number match (e.g., "002" -> "002-*")
	if regexp.MustCompile(`^\d{3}$`).MatchString(specIdentifier) {
		pattern := filepath.Join(specsDir, specIdentifier+"-*")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return "", fmt.Errorf("failed to glob spec directory: %w", err)
		}
		if len(matches) == 1 {
			return matches[0], nil
		}
		if len(matches) > 1 {
			return "", fmt.Errorf("multiple specs found for number %s: %v", specIdentifier, matches)
		}
	}

	// Try name match (e.g., "go-binary-migration" -> "*-go-binary-migration")
	pattern := filepath.Join(specsDir, "*-"+specIdentifier)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to glob spec directory: %w", err)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple specs found for name %s: %v", specIdentifier, matches)
	}

	return "", fmt.Errorf("spec directory not found for identifier: %s", specIdentifier)
}

// GetSpecMetadata returns metadata for a given spec identifier
func GetSpecMetadata(specsDir, specIdentifier string) (*Metadata, error) {
	directory, err := GetSpecDirectory(specsDir, specIdentifier)
	if err != nil {
		return nil, err
	}

	// Parse directory name to extract number and name
	baseName := filepath.Base(directory)
	if match := specDirPattern.FindStringSubmatch(baseName); match != nil {
		metadata := &Metadata{
			Number:    match[1],
			Name:      match[2],
			Directory: directory,
		}

		// Try to get branch if in git repo
		if git.IsGitRepository() {
			if branch, err := git.GetCurrentBranch(); err == nil {
				metadata.Branch = branch
			}
		}

		return metadata, nil
	}

	return nil, fmt.Errorf("could not parse spec directory name: %s", baseName)
}
