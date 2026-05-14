// Package prereqs provides pre-computed context for slash commands.
// It extracts feature paths and metadata that can be injected into
// command templates before agent execution, eliminating the need for
// agents to run bash commands to obtain this information.
package prereqs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ariel-frischer/autospec/internal/constitution"
	"github.com/ariel-frischer/autospec/internal/git"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/version"
)

// Context contains pre-computed prereqs data for template rendering.
// These fields map to template variables in command markdown files:
// {{.FeatureDir}}, {{.FeatureSpec}}, {{.ImplPlan}}, etc.
type Context struct {
	FeatureDir       string   // Path to the feature directory (e.g., specs/103-feature-name)
	FeatureSpec      string   // Path to spec.yaml file
	ImplPlan         string   // Path to plan.yaml file
	TasksFile        string   // Path to tasks.yaml file
	ConstitutionFile string   // Path to project constitution file, when present
	AutospecVersion  string   // Current autospec version string (e.g., "autospec 0.9.0")
	CreatedDate      string   // ISO 8601 timestamp for artifact creation
	IsGitRepo        bool     // Whether current directory is in a git repository
	AvailableDocs    []string // List of available artifact files in feature directory
}

// Options configures the behavior of ComputeContext.
type Options struct {
	SpecsDir     string // Directory containing spec folders (default: "./specs")
	RequireSpec  bool   // Require spec.yaml to exist
	RequirePlan  bool   // Require plan.yaml to exist
	RequireTasks bool   // Require tasks.yaml to exist
	IncludeTasks bool   // Include tasks.yaml in AvailableDocs
	PathsOnly    bool   // Only compute paths, skip validation
}

// ComputeContext computes prereqs context from the current environment.
// It detects the current feature from git branch or spec directories and
// validates that required files exist based on the provided options.
func ComputeContext(opts Options) (*Context, error) {
	specsDir := opts.SpecsDir
	if specsDir == "" {
		specsDir = "./specs"
	}

	hasGit := git.IsGitRepository()

	specMeta, err := detectFeature(specsDir, hasGit)
	if err != nil && !opts.PathsOnly {
		return nil, fmt.Errorf("detecting current feature: %w", err)
	}

	ctx := buildContextFromMetadata(specMeta, hasGit)

	if opts.PathsOnly {
		return ctx, nil
	}

	if err := validateContextPaths(ctx, opts); err != nil {
		return nil, err
	}

	ctx.AvailableDocs = computeAvailableDocs(ctx, opts.IncludeTasks)

	return ctx, nil
}

// buildContextFromMetadata creates a Context from spec metadata.
func buildContextFromMetadata(meta *spec.Metadata, hasGit bool) *Context {
	ctx := &Context{
		AutospecVersion: fmt.Sprintf("autospec %s", version.Version),
		CreatedDate:     time.Now().UTC().Format(time.RFC3339),
		IsGitRepo:       hasGit,
		AvailableDocs:   []string{},
	}

	if meta != nil {
		ctx.FeatureDir = meta.Directory
		ctx.FeatureSpec = filepath.Join(meta.Directory, "spec.yaml")
		ctx.ImplPlan = filepath.Join(meta.Directory, "plan.yaml")
		ctx.TasksFile = filepath.Join(meta.Directory, "tasks.yaml")
	}
	if path, ok := constitution.Find(); ok {
		ctx.ConstitutionFile = path
	}

	return ctx
}

// validateContextPaths validates that required files exist.
func validateContextPaths(ctx *Context, opts Options) error {
	if ctx.FeatureDir == "" {
		return fmt.Errorf("feature directory not detected")
	}

	if _, err := os.Stat(ctx.FeatureDir); os.IsNotExist(err) {
		return fmt.Errorf("feature directory not found: %s\nRun /autospec.specify first to create the feature structure", ctx.FeatureDir)
	}

	requirePlan := opts.RequirePlan || (!opts.RequireSpec && !opts.RequireTasks)

	if opts.RequireSpec {
		if _, err := os.Stat(ctx.FeatureSpec); os.IsNotExist(err) {
			return fmt.Errorf("no spec.yaml found in %s\nRun /autospec.specify first to create the spec", ctx.FeatureDir)
		}
	}

	if requirePlan {
		if _, err := os.Stat(ctx.ImplPlan); os.IsNotExist(err) {
			return fmt.Errorf("no plan.yaml found in %s\nRun /autospec.plan first to create the plan", ctx.FeatureDir)
		}
	}

	if opts.RequireTasks {
		if _, err := os.Stat(ctx.TasksFile); os.IsNotExist(err) {
			return fmt.Errorf("no tasks.yaml found in %s\nRun /autospec.tasks first to create tasks", ctx.FeatureDir)
		}
	}

	return nil
}

// computeAvailableDocs builds the list of available documents.
func computeAvailableDocs(ctx *Context, includeTasks bool) []string {
	var docs []string

	if _, err := os.Stat(ctx.FeatureSpec); err == nil {
		docs = append(docs, "spec.yaml")
	}

	if _, err := os.Stat(ctx.ImplPlan); err == nil {
		docs = append(docs, "plan.yaml")
	}

	if includeTasks {
		if _, err := os.Stat(ctx.TasksFile); err == nil {
			docs = append(docs, "tasks.yaml")
		}
	}

	checklistsDir := filepath.Join(ctx.FeatureDir, "checklists")
	if info, err := os.Stat(checklistsDir); err == nil && info.IsDir() {
		entries, err := os.ReadDir(checklistsDir)
		if err == nil && len(entries) > 0 {
			docs = append(docs, "checklists/")
		}
	}

	return docs
}

// detectFeature attempts to detect the current feature from environment, git, or spec directories.
func detectFeature(specsDir string, hasGit bool) (*spec.Metadata, error) {
	if envFeature := os.Getenv("SPECIFY_FEATURE"); envFeature != "" {
		// Use GetSpecMetadata to properly parse number and name from spec identifier
		meta, err := spec.GetSpecMetadata(specsDir, envFeature)
		if err == nil {
			meta.Detection = spec.DetectionEnvVar
			return meta, nil
		}
		// Fall through to other detection methods if env var spec not found
	}

	meta, err := spec.DetectCurrentSpec(specsDir)
	if err != nil {
		if hasGit {
			branch, _ := git.GetCurrentBranch()
			if branch != "" {
				return nil, fmt.Errorf("not on a feature branch. Current branch: %s\nFeature branches should be named like: 001-feature-name", branch)
			}
		}
		return nil, fmt.Errorf("could not detect current feature: %w", err)
	}

	return meta, nil
}
