package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ariel-frischer/autospec/internal/agent"
	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/git"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/spf13/cobra"
)

var (
	updateAgentContextAgentFlag string
	updateAgentContextJSONFlag  bool
)

var updateAgentContextCmd = &cobra.Command{
	Use:   "update-agent-context",
	Short: "Update AI agent context files with technology information from plan.yaml",
	Long: `Updates AI agent context files (CLAUDE.md, GEMINI.md, etc.) with technology
information extracted from the current feature's plan.yaml file.

The command reads the technical_context section from plan.yaml and updates
the Active Technologies and Recent Changes sections in agent context files.

By default, it updates all existing agent context files. If no agent files
exist, it creates CLAUDE.md from a template.

Supported agents: claude, gemini, copilot, cursor, qwen, opencode, codex,
windsurf, kilocode, auggie, roo, codebuddy, qoder, amp, shai, q, bob`,
	Example: `  # Update all existing agent context files
  autospec update-agent-context

  # Update only a specific agent's context file
  autospec update-agent-context --agent claude

  # Create a specific agent file if it doesn't exist
  autospec update-agent-context --agent cursor

  # Get JSON output for integration with other tools
  autospec update-agent-context --json`,
	RunE: runUpdateAgentContext,
}

func init() {
	updateAgentContextCmd.GroupID = GroupInternal
	rootCmd.AddCommand(updateAgentContextCmd)

	updateAgentContextCmd.Flags().StringVar(&updateAgentContextAgentFlag, "agent", "",
		"Update only the specified agent's context file (e.g., claude, gemini, copilot)")
	updateAgentContextCmd.Flags().BoolVar(&updateAgentContextJSONFlag, "json", false,
		"Output results as JSON for programmatic consumption")
}

func runUpdateAgentContext(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")

	cfg, err := loadAgentContextConfig(configPath)
	if err != nil {
		return err
	}

	repoRoot, err := getGitRepoRoot()
	if err != nil {
		return err
	}

	metadata, err := detectSpecForAgentContext(cfg.SpecsDir)
	if err != nil {
		return err
	}

	specName := fmt.Sprintf("%s-%s", metadata.Number, metadata.Name)
	planPath := filepath.Join(metadata.Directory, "plan.yaml")

	planData, err := parseAgentPlanData(planPath)
	if err != nil {
		return err
	}

	if err := validateAgentFlag(); err != nil {
		return err
	}

	results, updateErr := executeAgentUpdates(repoRoot, planData)
	output := buildCommandOutput(specName, planPath, planData, results, updateErr)

	if updateAgentContextJSONFlag {
		return outputJSON(output)
	}
	return outputText(output)
}

// loadAgentContextConfig loads config for agent context command
func loadAgentContextConfig(configPath string) (*config.Configuration, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		cliErr := clierrors.ConfigParseError(configPath, err)
		if !updateAgentContextJSONFlag {
			clierrors.PrintError(cliErr)
		}
		return nil, outputError(cliErr, updateAgentContextJSONFlag)
	}
	return cfg, nil
}

// getGitRepoRoot gets the git repository root
func getGitRepoRoot() (string, error) {
	repoRoot, err := git.GetRepositoryRoot()
	if err != nil {
		cliErr := fmt.Errorf("not in a git repository: %w. Run this command from within a git repository", err)
		if !updateAgentContextJSONFlag {
			fmt.Fprintf(os.Stderr, "Error: %v\n", cliErr)
		}
		return "", outputError(cliErr, updateAgentContextJSONFlag)
	}
	return repoRoot, nil
}

// detectSpecForAgentContext detects the current spec
func detectSpecForAgentContext(specsDir string) (*spec.Metadata, error) {
	metadata, err := spec.DetectCurrentSpec(specsDir)
	if err != nil {
		cliErr := fmt.Errorf("failed to detect spec: %w\nEnsure you're on a feature branch or have spec directories in %s", err, specsDir)
		if !updateAgentContextJSONFlag {
			fmt.Fprintf(os.Stderr, "Error: %v\n", cliErr)
		}
		return nil, outputError(cliErr, updateAgentContextJSONFlag)
	}
	return metadata, nil
}

// parseAgentPlanData parses plan.yaml for agent context
func parseAgentPlanData(planPath string) (*agent.PlanData, error) {
	planData, err := agent.ParsePlanData(planPath)
	if err != nil {
		cliErr := fmt.Errorf("failed to parse plan.yaml: %w", err)
		if !updateAgentContextJSONFlag {
			fmt.Fprintf(os.Stderr, "Error: %v\n", cliErr)
		}
		return nil, outputError(cliErr, updateAgentContextJSONFlag)
	}
	return planData, nil
}

// validateAgentFlag validates the agent flag if provided
func validateAgentFlag() error {
	if updateAgentContextAgentFlag == "" {
		return nil
	}
	if _, err := agent.GetAgentByID(updateAgentContextAgentFlag); err != nil {
		validAgents := strings.Join(agent.GetAllAgentIDs(), ", ")
		cliErr := fmt.Errorf("invalid agent: %q\nValid agents: %s", updateAgentContextAgentFlag, validAgents)
		if !updateAgentContextJSONFlag {
			fmt.Fprintf(os.Stderr, "Error: %v\n", cliErr)
		}
		return outputError(cliErr, updateAgentContextJSONFlag)
	}
	return nil
}

// executeAgentUpdates performs the agent file updates
func executeAgentUpdates(repoRoot string, planData *agent.PlanData) ([]agent.UpdateResult, error) {
	if updateAgentContextAgentFlag != "" {
		result, err := agent.UpdateSingleAgent(updateAgentContextAgentFlag, repoRoot, planData)
		if result != nil {
			return []agent.UpdateResult{*result}, err
		}
		return nil, err
	}
	return agent.UpdateAllAgents(repoRoot, planData)
}

func buildCommandOutput(specName, planPath string, planData *agent.PlanData, results []agent.UpdateResult, updateErr error) agent.CommandOutput {
	output := agent.CommandOutput{
		Success:      true,
		SpecName:     specName,
		PlanPath:     planPath,
		Technologies: planData,
		UpdatedFiles: results,
		Errors:       []string{},
	}

	// Check for errors in results
	for _, result := range results {
		if result.Error != nil {
			output.Success = false
			output.Errors = append(output.Errors, result.Error.Error())
		}
	}

	// Add update error if present
	if updateErr != nil {
		output.Success = false
		output.Errors = append(output.Errors, updateErr.Error())
	}

	return output
}

func outputJSON(output agent.CommandOutput) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode JSON output: %w", err)
	}

	if !output.Success {
		return fmt.Errorf("update failed")
	}
	return nil
}

func outputText(output agent.CommandOutput) error {
	// Print header
	fmt.Printf("Updating agent context files for: %s\n", output.SpecName)
	fmt.Printf("Plan: %s\n\n", output.PlanPath)

	// Print technologies found
	if output.Technologies != nil {
		techs := output.Technologies.GetTechnologies()
		if len(techs) > 0 {
			fmt.Println("Technologies detected:")
			for _, tech := range techs {
				fmt.Printf("  - %s\n", tech)
			}
			fmt.Println()
		}
	}

	// Print results
	if len(output.UpdatedFiles) > 0 {
		fmt.Println("Updated files:")
		for _, result := range output.UpdatedFiles {
			status := "updated"
			if result.Created {
				status = "created"
			}
			if result.Error != nil {
				status = "failed"
			}

			fmt.Printf("  ✓ %s (%s)\n", result.FilePath, status)

			if len(result.TechnologiesAdded) > 0 {
				for _, tech := range result.TechnologiesAdded {
					fmt.Printf("    + %s\n", tech)
				}
			}
		}
		fmt.Println()
	}

	// Print errors
	if len(output.Errors) > 0 {
		fmt.Println("Errors:")
		for _, err := range output.Errors {
			fmt.Printf("  ✗ %s\n", err)
		}
		return fmt.Errorf("update completed with errors")
	}

	fmt.Println("✓ Agent context files updated successfully")
	return nil
}

func outputError(err error, jsonOutput bool) error {
	if jsonOutput {
		output := agent.CommandOutput{
			Success: false,
			Errors:  []string{err.Error()},
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		encoder.Encode(output)
	}
	return err
}
