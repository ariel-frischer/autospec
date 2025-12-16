package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/config"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/validation"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:          "status [spec-name]",
	Aliases:      []string{"st"},
	Short:        "Show implementation progress for current feature (st)",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		verbose, _ := cmd.Flags().GetBool("verbose")

		// Load configuration
		cfg, err := config.Load(configPath)
		if err != nil {
			cliErr := clierrors.ConfigParseError(configPath, err)
			clierrors.PrintError(cliErr)
			return cliErr
		}

		// Detect or get spec
		var metadata *spec.Metadata
		if len(args) > 0 {
			metadata, err = spec.GetSpecMetadata(cfg.SpecsDir, args[0])
		} else {
			metadata, err = spec.DetectCurrentSpec(cfg.SpecsDir)
		}
		if err != nil {
			return fmt.Errorf("failed to detect spec: %w", err)
		}

		// Check which artifact files exist
		artifacts := []string{"spec.yaml", "plan.yaml", "tasks.yaml"}
		var existing []string
		for _, artifact := range artifacts {
			path := filepath.Join(metadata.Directory, artifact)
			if _, err := os.Stat(path); err == nil {
				existing = append(existing, artifact)
			}
		}

		// Concise output
		fmt.Printf("%s-%s\n", metadata.Number, metadata.Name)

		// Show artifacts
		if len(existing) > 0 {
			fmt.Printf("  artifacts: %v\n", existing)
		} else {
			fmt.Println("  artifacts: none")
		}

		// Get tasks file path (prefers .yaml over .md)
		tasksPath := validation.GetTasksFilePath(metadata.Directory)

		// Get task stats (only if tasks file exists)
		stats, err := validation.GetTaskStats(tasksPath)
		if err == nil {
			fmt.Print(validation.FormatTaskSummary(stats))
		}

		// Show phase details in verbose mode
		if verbose && stats != nil {
			fmt.Println()
			for _, phase := range stats.PhaseStats {
				status := "[ ]"
				if phase.IsComplete {
					status = "[✓]"
				} else if phase.CompletedTasks > 0 {
					status = "[~]"
				}
				fmt.Printf("  %s Phase %d: %s (%d/%d)\n",
					status, phase.Number, phase.Title, phase.CompletedTasks, phase.TotalTasks)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolP("verbose", "v", false, "Show all tasks, not just unchecked")
}
