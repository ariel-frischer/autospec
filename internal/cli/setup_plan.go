package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/git"
	"github.com/ariel-frischer/autospec/internal/prereqs"
	"github.com/spf13/cobra"
)

var setupPlanJSON bool

// SetupPlanOutput is the JSON output structure for the setup-plan command
type SetupPlanOutput struct {
	FeatureSpec string `json:"FEATURE_SPEC"`
	ImplPlan    string `json:"IMPL_PLAN"`
	SpecsDir    string `json:"SPECS_DIR"`
	Branch      string `json:"BRANCH"`
	HasGit      string `json:"HAS_GIT"`
}

var setupPlanCmd = &cobra.Command{
	Use:   "setup-plan",
	Short: "Initialize plan file from template",
	Long: `Initialize a plan file in the current feature directory.

This command:
1. Detects the current feature from git branch or SPECIFY_FEATURE environment variable
2. Copies the plan template to the feature directory as plan.yaml
3. Creates the feature directory if it doesn't exist

The template is searched for in the following order:
1. .autospec/templates/plan-template.yaml
2. .autospec/templates/plan-template.md
3. If no template exists, an empty plan.yaml is created with a warning`,
	Example: `  # Initialize plan from template
  autospec setup-plan

  # JSON output for scripting
  autospec setup-plan --json`,
	RunE: runSetupPlan,
}

func init() {
	setupPlanCmd.GroupID = GroupInternal
	setupPlanCmd.Flags().BoolVar(&setupPlanJSON, "json", false, "Output in JSON format")
	rootCmd.AddCommand(setupPlanCmd)
}

func runSetupPlan(cmd *cobra.Command, args []string) error {
	// Get specs directory
	specsDir, err := cmd.Flags().GetString("specs-dir")
	if err != nil || specsDir == "" {
		specsDir = "./specs"
	}

	// Check if we have git
	hasGit := git.IsGitRepository()
	hasGitStr := "false"
	if hasGit {
		hasGitStr = "true"
	}

	// Detect current spec using prereqs package with PathsOnly to skip file validation
	opts := prereqs.Options{
		SpecsDir:  specsDir,
		PathsOnly: true,
	}
	ctx, err := prereqs.ComputeContext(opts)
	if err != nil {
		return fmt.Errorf("detecting current feature: %w", err)
	}

	featureDir := ctx.FeatureDir
	branch := ""
	if hasGit {
		branch, _ = git.GetCurrentBranch()
	}

	// Ensure feature directory exists
	if err := os.MkdirAll(featureDir, 0o755); err != nil {
		return fmt.Errorf("failed to create feature directory: %w", err)
	}

	// Construct paths
	featureSpec := filepath.Join(featureDir, "spec.yaml")
	implPlan := filepath.Join(featureDir, "plan.yaml")

	// Get repository root
	repoRoot := filepath.Dir(specsDir)
	if git.IsGitRepository() {
		if root, err := git.GetRepositoryRoot(); err == nil {
			repoRoot = root
		}
	}

	// Look for plan template
	templatePaths := setupPlanTemplatePaths(repoRoot)

	var templateFound bool
	for _, templatePath := range templatePaths {
		if _, err := os.Stat(templatePath); err == nil {
			// Copy template to plan file
			if err := copyFile(templatePath, implPlan); err != nil {
				return fmt.Errorf("failed to copy plan template: %w", err)
			}
			templateFound = true
			if !setupPlanJSON {
				fmt.Printf("Copied plan template to %s\n", implPlan)
			}
			break
		}
	}

	if !templateFound {
		// Create empty plan file
		if err := os.WriteFile(implPlan, []byte{}, 0o644); err != nil {
			return fmt.Errorf("failed to create plan file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Warning: Plan template not found at %s\n", templatePaths[0])
	}

	// Output
	output := SetupPlanOutput{
		FeatureSpec: featureSpec,
		ImplPlan:    implPlan,
		SpecsDir:    featureDir,
		Branch:      branch,
		HasGit:      hasGitStr,
	}

	if setupPlanJSON {
		enc := json.NewEncoder(os.Stdout)
		return enc.Encode(output)
	}

	fmt.Printf("FEATURE_SPEC: %s\n", output.FeatureSpec)
	fmt.Printf("IMPL_PLAN: %s\n", output.ImplPlan)
	fmt.Printf("SPECS_DIR: %s\n", output.SpecsDir)
	fmt.Printf("BRANCH: %s\n", output.Branch)
	fmt.Printf("HAS_GIT: %s\n", output.HasGit)

	return nil
}

func setupPlanTemplatePaths(repoRoot string) []string {
	return []string{
		filepath.Join(repoRoot, ".autospec", "templates", "plan-template.yaml"),
		filepath.Join(repoRoot, ".autospec", "templates", "plan-template.md"),
	}
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("copying file contents: %w", err)
	}
	return nil
}
