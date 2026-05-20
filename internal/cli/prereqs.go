package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/prereqs"
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
	IsGitRepo       bool     `json:"IS_GIT_REPO"`
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
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	specsDir, err := cmd.Flags().GetString("specs-dir")
	if err != nil || specsDir == "" {
		specsDir = cfg.SpecsDir
	}

	opts := prereqs.Options{
		SpecsDir:     specsDir,
		StateDir:     cfg.StateDir,
		RequireSpec:  prereqsRequireSpec,
		RequirePlan:  prereqsRequirePlan,
		RequireTasks: prereqsRequireTasks,
		IncludeTasks: prereqsIncludeTasks,
		PathsOnly:    prereqsPathsOnly,
	}

	ctx, err := prereqs.ComputeContext(opts)
	if err != nil {
		return err
	}

	output := PrereqsOutput{
		FeatureDir:      ctx.FeatureDir,
		FeatureSpec:     ctx.FeatureSpec,
		ImplPlan:        ctx.ImplPlan,
		Tasks:           ctx.TasksFile,
		AvailableDocs:   ctx.AvailableDocs,
		AutospecVersion: ctx.AutospecVersion,
		CreatedDate:     ctx.CreatedDate,
		IsGitRepo:       ctx.IsGitRepo,
	}

	if prereqsJSON {
		enc := json.NewEncoder(os.Stdout)
		return enc.Encode(output)
	}

	fmt.Printf("FEATURE_DIR:%s\n", output.FeatureDir)
	fmt.Printf("FEATURE_SPEC:%s\n", output.FeatureSpec)
	fmt.Printf("IMPL_PLAN:%s\n", output.ImplPlan)
	fmt.Printf("TASKS:%s\n", output.Tasks)
	fmt.Println("AVAILABLE_DOCS:")
	for _, doc := range output.AvailableDocs {
		fmt.Printf("  ✓ %s\n", doc)
	}

	return nil
}

// PrintSpecInfo prints the detected spec info to stdout.
// This should be called after successfully detecting a spec to provide
// visibility into which spec was selected and how it was detected.
func PrintSpecInfo(metadata *spec.Metadata) {
	if metadata != nil {
		fmt.Println(metadata.FormatInfo())
	}
}
