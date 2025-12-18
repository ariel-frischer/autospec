// Package spec provides feature specification detection and metadata management.
// It automatically detects the current spec from git branch names (e.g., "002-feature-name")
// or falls back to the most recently modified directory in the specs folder.
package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/ariel-frischer/autospec/internal/git"
	"gopkg.in/yaml.v3"
)

var (
	// specBranchPattern matches branch names like "002-go-binary-migration"
	specBranchPattern = regexp.MustCompile(`^(\d{3})-(.+)$`)
	// specDirPattern matches directory names like "002-go-binary-migration"
	specDirPattern = regexp.MustCompile(`^(\d{3})-(.+)$`)
)

// DetectionMethod indicates how the spec was detected
type DetectionMethod string

const (
	// DetectionGitBranch indicates spec was detected from git branch name
	DetectionGitBranch DetectionMethod = "git_branch"
	// DetectionFallbackRecent indicates spec was detected as most recently modified
	DetectionFallbackRecent DetectionMethod = "fallback"
	// DetectionEnvVar indicates spec was detected from SPECIFY_FEATURE env var
	DetectionEnvVar DetectionMethod = "env_var"
	// DetectionExplicit indicates spec was explicitly specified by user
	DetectionExplicit DetectionMethod = "explicit"
)

// Metadata represents information about a feature specification
type Metadata struct {
	Name      string          // Feature name (e.g., "go-binary-migration")
	Number    string          // Spec number (e.g., "002")
	Directory string          // Full path to spec directory
	Branch    string          // Git branch name (if in git repo)
	Detection DetectionMethod // How the spec was detected
}

// FormatInfo returns a formatted string showing the detected spec with detection method.
// Example: "✓ Using spec: specs/002-feature (via git branch)"
func (m *Metadata) FormatInfo() string {
	methodDesc := ""
	switch m.Detection {
	case DetectionGitBranch:
		methodDesc = "via git branch"
	case DetectionFallbackRecent:
		methodDesc = "fallback - most recent"
	case DetectionEnvVar:
		methodDesc = "via SPECIFY_FEATURE env"
	case DetectionExplicit:
		methodDesc = "explicitly specified"
	default:
		methodDesc = "auto-detected"
	}
	return fmt.Sprintf("✓ Using spec: %s (%s)", m.Directory, methodDesc)
}

// DetectCurrentSpec attempts to detect the current spec from git branch or directory.
//
// Two-strategy detection:
//  1. Git branch: Parse branch name matching "NNN-name" pattern, verify directory exists
//  2. Fallback: Glob all spec directories, sort by modification time, return most recent
//
// Strategy 1 provides branch-based workflow; Strategy 2 handles detached HEAD or non-git.
// Returns Detection field indicating which strategy succeeded.
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
						Detection: DetectionGitBranch,
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
			Detection: DetectionFallbackRecent,
		}, nil
	}

	return nil, fmt.Errorf("could not parse spec directory name: %s", baseName)
}

// GetSpecDirectory returns the full path to a spec directory given its number or name.
//
// Three-level matching (tries in order, returns first match):
//  1. Exact: "002-migration" → specs/002-migration
//  2. Number: "002" → specs/002-* (glob, must be unique)
//  3. Name: "migration" → specs/*-migration (glob, must be unique)
//
// Returns error if multiple matches found (ambiguous) or no matches.
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

// UpdateResult contains the result of a spec status update
type UpdateResult struct {
	Updated        bool   // Whether the status was updated
	PreviousStatus string // The previous status value
	NewStatus      string // The new status value
	CompletedAt    string // The completion timestamp (if set)
}

// MarkSpecCompleted updates the spec.yaml status to "Completed" and sets the completed_at timestamp.
// Returns an UpdateResult indicating what changed, or an error if the update failed.
// This operation is idempotent - if already completed, it returns with Updated=false.
func MarkSpecCompleted(specDir string) (*UpdateResult, error) {
	return UpdateSpecStatus(specDir, "Completed", time.Now().UTC())
}

// UpdateSpecStatus updates the feature.status field in spec.yaml and optionally sets completed_at.
// If completedAt is not zero, it will be set to the ISO 8601 formatted timestamp.
// This preserves the existing YAML structure and comments using yaml.Node parsing.
func UpdateSpecStatus(specDir string, newStatus string, completedAt time.Time) (*UpdateResult, error) {
	specPath := filepath.Join(specDir, "spec.yaml")

	// Read the file
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec.yaml: %w", err)
	}

	// Parse YAML preserving structure
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("failed to parse spec.yaml: %w", err)
	}

	// Find and update the feature section
	result, err := updateFeatureStatus(&root, newStatus, completedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update feature status: %w", err)
	}

	// If no update needed (already at target status), return early
	if !result.Updated {
		return result, nil
	}

	// Write back the updated YAML
	output, err := yaml.Marshal(&root)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize spec.yaml: %w", err)
	}

	if err := os.WriteFile(specPath, output, 0644); err != nil {
		return nil, fmt.Errorf("failed to write spec.yaml: %w", err)
	}

	return result, nil
}

// updateFeatureStatus traverses the YAML node tree to find and update the feature.status field.
func updateFeatureStatus(node *yaml.Node, newStatus string, completedAt time.Time) (*UpdateResult, error) {
	if node == nil {
		return nil, fmt.Errorf("nil node")
	}

	// For document nodes, recurse into content
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return nil, fmt.Errorf("empty document")
		}
		return updateFeatureStatus(node.Content[0], newStatus, completedAt)
	}

	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("expected mapping node, got %v", node.Kind)
	}

	// Find the "feature" key
	var featureNode *yaml.Node
	for i := 0; i < len(node.Content)-1; i += 2 {
		key := node.Content[i]
		value := node.Content[i+1]
		if key.Value == "feature" {
			featureNode = value
			break
		}
	}

	if featureNode == nil {
		return nil, fmt.Errorf("feature section not found in spec.yaml")
	}

	if featureNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("feature section is not a mapping")
	}

	// Find status and completed_at fields within feature
	var statusNode *yaml.Node
	var completedAtNode *yaml.Node
	var completedAtKeyIdx int = -1

	for i := 0; i < len(featureNode.Content)-1; i += 2 {
		key := featureNode.Content[i]
		value := featureNode.Content[i+1]

		if key.Value == "status" {
			statusNode = value
		}
		if key.Value == "completed_at" {
			completedAtNode = value
			completedAtKeyIdx = i
		}
	}

	if statusNode == nil {
		return nil, fmt.Errorf("status field not found in feature section")
	}

	previousStatus := statusNode.Value

	// If already at target status, no update needed
	if previousStatus == newStatus {
		return &UpdateResult{
			Updated:        false,
			PreviousStatus: previousStatus,
			NewStatus:      newStatus,
		}, nil
	}

	// Update status
	statusNode.Value = newStatus

	result := &UpdateResult{
		Updated:        true,
		PreviousStatus: previousStatus,
		NewStatus:      newStatus,
	}

	// Set completed_at if completing and timestamp provided
	if newStatus == "Completed" && !completedAt.IsZero() {
		timestamp := completedAt.Format(time.RFC3339)
		result.CompletedAt = timestamp

		if completedAtNode != nil {
			// Update existing completed_at
			completedAtNode.Value = timestamp
		} else {
			// Add new completed_at field after status
			// Find the index of status field to insert after it
			statusIdx := -1
			for i := 0; i < len(featureNode.Content)-1; i += 2 {
				if featureNode.Content[i].Value == "status" {
					statusIdx = i
					break
				}
			}

			if statusIdx >= 0 {
				// Insert completed_at after status
				keyNode := &yaml.Node{
					Kind:  yaml.ScalarNode,
					Value: "completed_at",
				}
				valueNode := &yaml.Node{
					Kind:  yaml.ScalarNode,
					Value: timestamp,
				}

				// Insert at position statusIdx+2 (after status key-value pair)
				insertIdx := statusIdx + 2
				newContent := make([]*yaml.Node, 0, len(featureNode.Content)+2)
				newContent = append(newContent, featureNode.Content[:insertIdx]...)
				newContent = append(newContent, keyNode, valueNode)
				newContent = append(newContent, featureNode.Content[insertIdx:]...)
				featureNode.Content = newContent
			}
		}
	} else if newStatus != "Completed" && completedAtKeyIdx >= 0 {
		// Remove completed_at if changing away from Completed status
		newContent := make([]*yaml.Node, 0, len(featureNode.Content)-2)
		newContent = append(newContent, featureNode.Content[:completedAtKeyIdx]...)
		newContent = append(newContent, featureNode.Content[completedAtKeyIdx+2:]...)
		featureNode.Content = newContent
	}

	return result, nil
}
