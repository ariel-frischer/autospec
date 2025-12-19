package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ariel-frischer/autospec/internal/cli/util"
	"github.com/ariel-frischer/autospec/internal/git"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/spf13/cobra"
)

var (
	prereqsJSON         bool
	prereqsRequireSpec  bool
	prereqsRequirePlan  bool
	prereqsRequireTasks bool
	prereqsIncludeTasks bool
	prereqsPathsOnly    bool
)

// PrereqsOutput is the JSON output structure for the prereqs command
type PrereqsOutput struct {
	FeatureDir      string   `json:"FEATURE_DIR"`
	FeatureSpec     string   `json:"FEATURE_SPEC"`
	ImplPlan        string   `json:"IMPL_PLAN"`
	Tasks           string   `json:"TASKS"`
	AvailableDocs   []string `json:"AVAILABLE_DOCS"`
	AutospecVersion string   `json:"AUTOSPEC_VERSION"`
	CreatedDate     string   `json:"CREATED_DATE"`
}

var prereqsCmd = &cobra.Command{
	Use:   "prereqs",
	Short: "Check prerequisites for workflow stages",
	Long: `Check that required artifacts exist before running a workflow stage.

This command validates that the necessary files are present in the current feature
directory and outputs the paths to those files. It's used by Claude slash commands
to ensure prerequisites are met before executing workflow stages.

By default, the plan file is required. Use --require-* flags to specify which
files must exist.`,
	Example: `  # Check spec prerequisites (spec.yaml required)
  autospec prereqs --json --require-spec

  # Check plan prerequisites (plan.yaml required - default)
  autospec prereqs --json

  # Check implementation prerequisites (plan.yaml + tasks.yaml required)
  autospec prereqs --json --require-tasks --include-tasks

  # Get feature paths only (no validation)
  autospec prereqs --paths-only`,
	RunE: runPrereqs,
}

func init() {
	prereqsCmd.GroupID = GroupInternal
	prereqsCmd.Flags().BoolVar(&prereqsJSON, "json", false, "Output in JSON format")
	prereqsCmd.Flags().BoolVar(&prereqsRequireSpec, "require-spec", false, "Require spec.yaml to exist")
	prereqsCmd.Flags().BoolVar(&prereqsRequirePlan, "require-plan", false, "Require plan.yaml to exist (default behavior)")
	prereqsCmd.Flags().BoolVar(&prereqsRequireTasks, "require-tasks", false, "Require tasks.yaml to exist")
	prereqsCmd.Flags().BoolVar(&prereqsIncludeTasks, "include-tasks", false, "Include tasks.yaml in AVAILABLE_DOCS list")
	prereqsCmd.Flags().BoolVar(&prereqsPathsOnly, "paths-only", false, "Only output path variables (no validation)")
	rootCmd.AddCommand(prereqsCmd)
}

func runPrereqs(cmd *cobra.Command, args []string) error {
	// Get specs directory
	specsDir, err := cmd.Flags().GetString("specs-dir")
	if err != nil || specsDir == "" {
		specsDir = "./specs"
	}

	// Check if we have git
	hasGit := git.IsGitRepository()

	// Detect current spec
	specMeta, err := detectCurrentFeature(specsDir, hasGit)
	if err != nil && !prereqsPathsOnly {
		return fmt.Errorf("detecting current feature: %w", err)
	}

	// If specMeta is nil in paths-only mode, we still need to provide some paths
	var featureDir, featureSpec, implPlan, tasks string

	if specMeta != nil {
		featureDir = specMeta.Directory
		featureSpec = filepath.Join(featureDir, "spec.yaml")
		implPlan = filepath.Join(featureDir, "plan.yaml")
		tasks = filepath.Join(featureDir, "tasks.yaml")
	}

	// If paths-only mode, output paths and exit
	if prereqsPathsOnly {
		autospecVersion := fmt.Sprintf("autospec %s", util.Version)
		createdDate := time.Now().UTC().Format(time.RFC3339)

		if prereqsJSON {
			output := PrereqsOutput{
				FeatureDir:      featureDir,
				FeatureSpec:     featureSpec,
				ImplPlan:        implPlan,
				Tasks:           tasks,
				AvailableDocs:   []string{},
				AutospecVersion: autospecVersion,
				CreatedDate:     createdDate,
			}
			enc := json.NewEncoder(os.Stdout)
			return enc.Encode(output)
		}
		fmt.Printf("REPO_ROOT: %s\n", filepath.Dir(specsDir))
		fmt.Printf("BRANCH: %s\n", specMeta.Branch)
		fmt.Printf("FEATURE_DIR: %s\n", featureDir)
		fmt.Printf("FEATURE_SPEC: %s\n", featureSpec)
		fmt.Printf("IMPL_PLAN: %s\n", implPlan)
		fmt.Printf("TASKS: %s\n", tasks)
		return nil
	}

	// Validate feature directory exists
	if _, err := os.Stat(featureDir); os.IsNotExist(err) {
		return fmt.Errorf("feature directory not found: %s\nRun /autospec.specify first to create the feature structure", featureDir)
	}

	// Determine what to require (default is require-plan if nothing specified)
	requirePlan := prereqsRequirePlan || (!prereqsRequireSpec && !prereqsRequireTasks)

	// Validate required files
	if prereqsRequireSpec {
		if _, err := os.Stat(featureSpec); os.IsNotExist(err) {
			return fmt.Errorf("no spec.yaml found in %s\nRun /autospec.specify first to create the spec", featureDir)
		}
	}

	if requirePlan {
		if _, err := os.Stat(implPlan); os.IsNotExist(err) {
			return fmt.Errorf("no plan.yaml found in %s\nRun /autospec.plan first to create the plan", featureDir)
		}
	}

	if prereqsRequireTasks {
		if _, err := os.Stat(tasks); os.IsNotExist(err) {
			return fmt.Errorf("no tasks.yaml found in %s\nRun /autospec.tasks first to create tasks", featureDir)
		}
	}

	// Build list of available documents
	var docs []string

	if _, err := os.Stat(featureSpec); err == nil {
		docs = append(docs, "spec.yaml")
	}

	if _, err := os.Stat(implPlan); err == nil {
		docs = append(docs, "plan.yaml")
	}

	if prereqsIncludeTasks {
		if _, err := os.Stat(tasks); err == nil {
			docs = append(docs, "tasks.yaml")
		}
	}

	// Check for checklists directory
	checklistsDir := filepath.Join(featureDir, "checklists")
	if info, err := os.Stat(checklistsDir); err == nil && info.IsDir() {
		entries, err := os.ReadDir(checklistsDir)
		if err == nil && len(entries) > 0 {
			docs = append(docs, "checklists/")
		}
	}

	// Get version and timestamp
	autospecVersion := fmt.Sprintf("autospec %s", util.Version)
	createdDate := time.Now().UTC().Format(time.RFC3339)

	// Output results
	output := PrereqsOutput{
		FeatureDir:      featureDir,
		FeatureSpec:     featureSpec,
		ImplPlan:        implPlan,
		Tasks:           tasks,
		AvailableDocs:   docs,
		AutospecVersion: autospecVersion,
		CreatedDate:     createdDate,
	}

	if prereqsJSON {
		enc := json.NewEncoder(os.Stdout)
		return enc.Encode(output)
	}

	// Text output
	fmt.Printf("FEATURE_DIR:%s\n", output.FeatureDir)
	fmt.Printf("FEATURE_SPEC:%s\n", output.FeatureSpec)
	fmt.Printf("IMPL_PLAN:%s\n", output.ImplPlan)
	fmt.Printf("TASKS:%s\n", output.Tasks)
	fmt.Println("AVAILABLE_DOCS:")
	for _, doc := range docs {
		fmt.Printf("  ✓ %s\n", doc)
	}

	return nil
}

// detectCurrentFeature attempts to detect the current feature from environment, git, or spec directories
func detectCurrentFeature(specsDir string, hasGit bool) (*spec.Metadata, error) {
	// First check SPECIFY_FEATURE environment variable
	if envFeature := os.Getenv("SPECIFY_FEATURE"); envFeature != "" {
		// Try to find this feature in specs directory
		featureDir := filepath.Join(specsDir, envFeature)
		if info, err := os.Stat(featureDir); err == nil && info.IsDir() {
			return &spec.Metadata{
				Name:      envFeature,
				Directory: featureDir,
				Detection: spec.DetectionEnvVar,
			}, nil
		}
	}

	// Try to detect from git branch or specs directory
	meta, err := spec.DetectCurrentSpec(specsDir)
	if err != nil {
		// Provide helpful error message
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

// PrintSpecInfo prints the detected spec info to stdout.
// This should be called after successfully detecting a spec to provide
// visibility into which spec was selected and how it was detected.
func PrintSpecInfo(metadata *spec.Metadata) {
	if metadata != nil {
		fmt.Println(metadata.FormatInfo())
	}
}
