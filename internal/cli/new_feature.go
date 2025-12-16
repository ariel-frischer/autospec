package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ariel-frischer/autospec/internal/git"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/spf13/cobra"
)

var (
	newFeatureJSON      bool
	newFeatureShortName string
	newFeatureNumber    string
)

// NewFeatureOutput is the JSON output structure for the new-feature command
type NewFeatureOutput struct {
	BranchName      string `json:"BRANCH_NAME"`
	SpecFile        string `json:"SPEC_FILE"`
	FeatureNum      string `json:"FEATURE_NUM"`
	AutospecVersion string `json:"AUTOSPEC_VERSION"`
	CreatedDate     string `json:"CREATED_DATE"`
}

var newFeatureCmd = &cobra.Command{
	Use:   "new-feature <feature_description>",
	Short: "Create a new feature branch and directory",
	Long: `Create a new feature branch and directory for a new specification.

This command:
1. Generates a branch name from the feature description (or uses --short-name)
2. Determines the next available feature number (or uses --number)
3. Creates a git branch (if in a git repository)
4. Creates the feature directory under specs/

The command outputs the created branch name, spec file path, and metadata.`,
	Example: `  # Create a new feature from description
  autospec new-feature "Add user authentication"

  # Create with a custom short name
  autospec new-feature --short-name "user-auth" "Add user authentication system"

  # Create with a specific number
  autospec new-feature --number 5 "OAuth2 integration"

  # JSON output for scripting
  autospec new-feature --json "Add dark mode support"`,
	Args: cobra.ExactArgs(1),
	RunE: runNewFeature,
}

func init() {
	newFeatureCmd.Flags().BoolVar(&newFeatureJSON, "json", false, "Output in JSON format")
	newFeatureCmd.Flags().StringVar(&newFeatureShortName, "short-name", "", "Custom short name for the branch (2-4 words)")
	newFeatureCmd.Flags().StringVar(&newFeatureNumber, "number", "", "Specify branch number manually (overrides auto-detection)")
	rootCmd.AddCommand(newFeatureCmd)
}

func runNewFeature(cmd *cobra.Command, args []string) error {
	featureDescription := args[0]

	// Get specs directory
	specsDir, err := cmd.Flags().GetString("specs-dir")
	if err != nil || specsDir == "" {
		specsDir = "./specs"
	}

	// Make specs directory absolute
	if !filepath.IsAbs(specsDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		specsDir = filepath.Join(cwd, specsDir)
	}

	// Check if we have git
	hasGit := git.IsGitRepository()

	// Fetch all remotes if in git repo (to get latest branch info)
	if hasGit {
		git.FetchAllRemotes() // Ignore errors, just try to get latest
	}

	// Determine branch number
	var branchNumber string
	if newFeatureNumber != "" {
		// Validate and use provided number
		num, err := strconv.Atoi(newFeatureNumber)
		if err != nil || num < 0 {
			return fmt.Errorf("invalid --number value: must be a positive integer")
		}
		branchNumber = fmt.Sprintf("%03d", num)
	} else {
		// Auto-detect next number
		branchNumber, err = spec.GetNextBranchNumber(specsDir)
		if err != nil {
			return fmt.Errorf("failed to determine next branch number: %w", err)
		}
	}

	// Generate branch suffix
	var branchSuffix string
	if newFeatureShortName != "" {
		branchSuffix = spec.CleanBranchName(newFeatureShortName)
	} else {
		branchSuffix = spec.GenerateBranchName(featureDescription)
	}

	// Create full branch name
	branchName := spec.FormatBranchName(branchNumber, branchSuffix)

	// Truncate if necessary
	branchName = spec.TruncateBranchName(branchName)

	// Create git branch if in git repo
	if hasGit {
		if err := git.CreateBranch(branchName); err != nil {
			// If branch already exists, warn but continue
			fmt.Fprintf(os.Stderr, "[specify] Warning: %v\n", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "[specify] Warning: Git repository not detected; skipped branch creation for %s\n", branchName)
	}

	// Create feature directory
	featureDir := spec.GetFeatureDirectory(specsDir, branchName)
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		return fmt.Errorf("failed to create feature directory: %w", err)
	}

	// Spec file path (created by autospec specify command, not this one)
	specFile := filepath.Join(featureDir, "spec.yaml")

	// Set SPECIFY_FEATURE environment variable
	os.Setenv("SPECIFY_FEATURE", branchName)

	// Get version and timestamp
	autospecVersion := fmt.Sprintf("autospec %s", Version)
	createdDate := time.Now().UTC().Format(time.RFC3339)

	// Output
	output := NewFeatureOutput{
		BranchName:      branchName,
		SpecFile:        specFile,
		FeatureNum:      branchNumber,
		AutospecVersion: autospecVersion,
		CreatedDate:     createdDate,
	}

	if newFeatureJSON {
		enc := json.NewEncoder(os.Stdout)
		if err := enc.Encode(output); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
	} else {
		fmt.Printf("BRANCH_NAME: %s\n", output.BranchName)
		fmt.Printf("SPEC_FILE: %s\n", output.SpecFile)
		fmt.Printf("FEATURE_NUM: %s\n", output.FeatureNum)
		fmt.Printf("AUTOSPEC_VERSION: %s\n", output.AutospecVersion)
		fmt.Printf("CREATED_DATE: %s\n", output.CreatedDate)
		fmt.Printf("SPECIFY_FEATURE environment variable set to: %s\n", branchName)
	}

	return nil
}
