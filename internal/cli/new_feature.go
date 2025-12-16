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
	newFeatureCmd.GroupID = GroupInternal
	newFeatureCmd.Flags().BoolVar(&newFeatureJSON, "json", false, "Output in JSON format")
	newFeatureCmd.Flags().StringVar(&newFeatureShortName, "short-name", "", "Custom short name for the branch (2-4 words)")
	newFeatureCmd.Flags().StringVar(&newFeatureNumber, "number", "", "Specify branch number manually (overrides auto-detection)")
	rootCmd.AddCommand(newFeatureCmd)
}

func runNewFeature(cmd *cobra.Command, args []string) error {
	featureDescription := args[0]

	specsDir, err := resolveSpecsDir(cmd)
	if err != nil {
		return err
	}

	hasGit := initGitForNewFeature()

	branchNumber, err := determineBranchNumber(specsDir)
	if err != nil {
		return err
	}

	branchName := generateBranchName(featureDescription, branchNumber)

	if err := createGitBranch(branchName, hasGit); err != nil {
		return err
	}

	specFile, err := setupFeatureDirectory(specsDir, branchName)
	if err != nil {
		return err
	}

	return outputNewFeatureResult(branchName, specFile, branchNumber)
}

// resolveSpecsDir gets and resolves the specs directory to an absolute path
func resolveSpecsDir(cmd *cobra.Command) (string, error) {
	specsDir, err := cmd.Flags().GetString("specs-dir")
	if err != nil || specsDir == "" {
		specsDir = "./specs"
	}

	if !filepath.IsAbs(specsDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		specsDir = filepath.Join(cwd, specsDir)
	}

	return specsDir, nil
}

// initGitForNewFeature checks for git and fetches remotes
func initGitForNewFeature() bool {
	hasGit := git.IsGitRepository()
	if hasGit {
		git.FetchAllRemotes() // Ignore errors, just try to get latest
	}
	return hasGit
}

// determineBranchNumber determines the branch number from flag or auto-detection
func determineBranchNumber(specsDir string) (string, error) {
	if newFeatureNumber != "" {
		num, err := strconv.Atoi(newFeatureNumber)
		if err != nil || num < 0 {
			return "", fmt.Errorf("invalid --number value: must be a positive integer")
		}
		return fmt.Sprintf("%03d", num), nil
	}

	branchNumber, err := spec.GetNextBranchNumber(specsDir)
	if err != nil {
		return "", fmt.Errorf("failed to determine next branch number: %w", err)
	}
	return branchNumber, nil
}

// generateBranchName creates the full branch name from description and number
func generateBranchName(featureDescription, branchNumber string) string {
	var branchSuffix string
	if newFeatureShortName != "" {
		branchSuffix = spec.CleanBranchName(newFeatureShortName)
	} else {
		branchSuffix = spec.GenerateBranchName(featureDescription)
	}

	branchName := spec.FormatBranchName(branchNumber, branchSuffix)
	return spec.TruncateBranchName(branchName)
}

// createGitBranch creates the git branch if in a git repository
func createGitBranch(branchName string, hasGit bool) error {
	if hasGit {
		if err := git.CreateBranch(branchName); err != nil {
			fmt.Fprintf(os.Stderr, "[specify] Warning: %v\n", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "[specify] Warning: Git repository not detected; skipped branch creation for %s\n", branchName)
	}
	return nil
}

// setupFeatureDirectory creates the feature directory and returns spec file path
func setupFeatureDirectory(specsDir, branchName string) (string, error) {
	featureDir := spec.GetFeatureDirectory(specsDir, branchName)
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create feature directory: %w", err)
	}

	specFile := filepath.Join(featureDir, "spec.yaml")
	os.Setenv("SPECIFY_FEATURE", branchName)

	return specFile, nil
}

// outputNewFeatureResult formats and outputs the result
func outputNewFeatureResult(branchName, specFile, branchNumber string) error {
	output := NewFeatureOutput{
		BranchName:      branchName,
		SpecFile:        specFile,
		FeatureNum:      branchNumber,
		AutospecVersion: fmt.Sprintf("autospec %s", Version),
		CreatedDate:     time.Now().UTC().Format(time.RFC3339),
	}

	if newFeatureJSON {
		enc := json.NewEncoder(os.Stdout)
		if err := enc.Encode(output); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
	} else {
		printNewFeatureText(output, branchName)
	}

	return nil
}

// printNewFeatureText prints the output in text format
func printNewFeatureText(output NewFeatureOutput, branchName string) {
	fmt.Printf("BRANCH_NAME: %s\n", output.BranchName)
	fmt.Printf("SPEC_FILE: %s\n", output.SpecFile)
	fmt.Printf("FEATURE_NUM: %s\n", output.FeatureNum)
	fmt.Printf("AUTOSPEC_VERSION: %s\n", output.AutospecVersion)
	fmt.Printf("CREATED_DATE: %s\n", output.CreatedDate)
	fmt.Printf("SPECIFY_FEATURE environment variable set to: %s\n", branchName)
}
