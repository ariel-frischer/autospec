package util

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/dag"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/validation"
	"github.com/spf13/cobra"
)

var dagCmd = &cobra.Command{
	Use:          "dag [spec-name]",
	Short:        "Visualize task dependency graph and execution waves",
	Long:         `Display an ASCII visualization of the task dependency graph showing which tasks can run in parallel and the execution order.`,
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE:         runDagCmd,
}

func init() {
	dagCmd.Flags().Bool("compact", false, "Show compact single-line output")
	dagCmd.Flags().Bool("detailed", false, "Show detailed task information")
	dagCmd.Flags().Bool("stats", false, "Show only wave statistics")
}

func runDagCmd(cmd *cobra.Command, args []string) error {
	compact, _ := cmd.Flags().GetBool("compact")
	detailed, _ := cmd.Flags().GetBool("detailed")
	stats, _ := cmd.Flags().GetBool("stats")
	configPath, _ := cmd.Flags().GetString("config")

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		cliErr := clierrors.ConfigParseError(configPath, err)
		clierrors.PrintError(cliErr)
		return cliErr
	}

	// Detect or get spec
	metadata, err := detectSpec(cfg.SpecsDir, args)
	if err != nil {
		return err
	}
	shared.PrintSpecInfo(metadata)

	// Load tasks
	tasksPath := validation.GetTasksFilePath(metadata.Directory)
	tasks, err := validation.GetAllTasks(tasksPath)
	if err != nil {
		return fmt.Errorf("loading tasks: %w", err)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found in tasks.yaml")
		return nil
	}

	// Build dependency graph
	graph, err := dag.BuildFromTasks(tasks)
	if err != nil {
		return fmt.Errorf("building dependency graph: %w", err)
	}

	// Compute waves
	_, err = graph.ComputeWaves()
	if err != nil {
		return fmt.Errorf("computing execution waves: %w", err)
	}

	// Render output based on flags
	return renderOutput(graph, compact, detailed, stats)
}

// detectSpec determines which spec to use from args or auto-detection.
func detectSpec(specsDir string, args []string) (*spec.Metadata, error) {
	var metadata *spec.Metadata
	var err error

	if len(args) > 0 {
		metadata, err = spec.GetSpecMetadata(specsDir, args[0])
		if err == nil {
			metadata.Detection = spec.DetectionExplicit
		}
	} else {
		metadata, err = spec.DetectCurrentSpec(specsDir)
	}

	if err != nil {
		return nil, fmt.Errorf("detecting spec: %w", err)
	}
	return metadata, nil
}

// renderOutput renders the graph visualization based on flags.
func renderOutput(graph *dag.DependencyGraph, compact, detailed, stats bool) error {
	if stats {
		waveStats := graph.GetWaveStats()
		printStats(waveStats)
		return nil
	}

	if compact {
		fmt.Println(graph.RenderCompact())
		return nil
	}

	if detailed {
		fmt.Print(graph.RenderDetailed())
		return nil
	}

	// Default: ASCII rendering
	fmt.Print(graph.RenderASCII())
	return nil
}

// printStats outputs wave statistics.
func printStats(stats dag.WaveStats) {
	fmt.Println("Wave Statistics:")
	fmt.Printf("  Total Waves: %d\n", stats.TotalWaves)
	fmt.Printf("  Total Tasks: %d\n", stats.TotalTasks)
	fmt.Printf("  Max Wave Size: %d\n", stats.MaxWaveSize)
	fmt.Printf("  Min Wave Size: %d\n", stats.MinWaveSize)

	if stats.TotalWaves > 0 {
		avgSize := float64(stats.TotalTasks) / float64(stats.TotalWaves)
		fmt.Printf("  Avg Wave Size: %.1f\n", avgSize)
	}
}
