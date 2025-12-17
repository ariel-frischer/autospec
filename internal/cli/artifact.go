package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/validation"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	artifactSchemaFlag bool
	artifactFixFlag    bool
)

var artifactCmd = &cobra.Command{
	Use:   "artifact <type|path> [path]",
	Short: "Validate YAML artifacts against their schemas",
	Long: `Validate YAML artifacts against their schemas.

Smart Detection:
  - Type only: autospec artifact plan → auto-detects spec from git branch
  - Path only: autospec artifact specs/001/plan.yaml → infers type from filename
  - Explicit: autospec artifact plan specs/001/plan.yaml → backward compatible

Types:
  spec         - Feature specification (spec.yaml)
  plan         - Implementation plan (plan.yaml)
  tasks        - Task breakdown (tasks.yaml)
  analysis     - Cross-artifact analysis (analysis.yaml)
  checklist    - Feature quality checklist (checklists/*.yaml)
  constitution - Project constitution (.autospec/memory/constitution.yaml)

Validates:
  - Valid YAML syntax
  - Required fields present for artifact type
  - Field types correct (strings, lists, enums)
  - Cross-references valid (e.g. task dependencies exist)

Output:
  - Shows which spec is being used (with fallback indicator if applicable)
  - Success message with artifact summary on valid artifacts
  - Detailed errors with line numbers and hints on invalid artifacts

Exit Codes:
  0 - Success (artifact is valid)
  1 - Validation failed (artifact has errors)
  3 - Invalid arguments (unknown type or missing file)`,
	Example: `  # Path only - preferred format (infers type from filename)
  autospec artifact specs/001-feature/spec.yaml
  autospec artifact specs/001-feature/plan.yaml
  autospec artifact specs/001-feature/tasks.yaml
  autospec artifact specs/001-feature/analysis.yaml
  autospec artifact .autospec/memory/constitution.yaml

  # Checklist requires explicit type (filename varies by domain)
  autospec artifact checklist specs/001-feature/checklists/ux.yaml

  # Type only (auto-detects spec from git branch)
  autospec artifact plan
  autospec artifact tasks

  # Show schema for an artifact type
  autospec artifact spec --schema
  autospec artifact constitution --schema

  # Auto-fix common issues
  autospec artifact specs/001-feature/plan.yaml --fix`,
	Args:          cobra.RangeArgs(1, 2),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")
		return runArtifactCommand(args, configPath, cmd.OutOrStdout(), cmd.ErrOrStderr())
	},
}

func init() {
	artifactCmd.GroupID = GroupInternal
	rootCmd.AddCommand(artifactCmd)
	artifactCmd.Flags().BoolVar(&artifactSchemaFlag, "schema", false, "Print the expected schema for the artifact type")
	artifactCmd.Flags().BoolVar(&artifactFixFlag, "fix", false, "Auto-fix common issues (missing optional fields, formatting)")
}

// artifactArgs represents parsed artifact command arguments.
type artifactArgs struct {
	artType      validation.ArtifactType
	filePath     string
	specMetadata *spec.Metadata // Detected spec metadata (for display)
	isPathArg    bool           // Whether first arg was a path (type inferred)
	isTypeOnly   bool           // Whether only type was provided (path auto-detected)
}

// parseArtifactArgs parses the command arguments and determines the artifact type and path.
// It supports three invocation patterns:
//   - Type only: autospec artifact plan → auto-detects path from spec directory
//   - Path only: autospec artifact specs/001/plan.yaml → infers type from filename
//   - Explicit: autospec artifact plan specs/001/plan.yaml → backward compatible
func parseArtifactArgs(args []string, specsDir string) (*artifactArgs, error) {
	result := &artifactArgs{}

	if len(args) == 0 {
		return nil, fmt.Errorf("no arguments provided")
	}

	firstArg := args[0]

	// Check if first arg is a path (contains .yaml or .yml extension)
	isPath := strings.HasSuffix(firstArg, ".yaml") || strings.HasSuffix(firstArg, ".yml")

	if isPath {
		// Path-only invocation: infer type from filename
		artType, err := validation.InferArtifactTypeFromFilename(firstArg)
		if err != nil {
			return nil, fmt.Errorf("%w\nValid artifact filenames: %s",
				err, strings.Join(validation.ValidArtifactFilenames(), ", "))
		}
		result.artType = artType
		result.filePath = firstArg
		result.isPathArg = true
		return result, nil
	}

	// First arg is a type
	artType, err := validation.ParseArtifactType(firstArg)
	if err != nil {
		return nil, err
	}
	result.artType = artType

	if len(args) == 2 {
		// Explicit type + path: backward compatible
		result.filePath = args[1]
		return result, nil
	}

	// Type-only invocation: auto-detect path from spec directory
	result.isTypeOnly = true

	resolvedPath, specMeta, err := resolveArtifactPath(artType, specsDir)
	if err != nil {
		return nil, err
	}

	result.filePath = resolvedPath
	result.specMetadata = specMeta

	return result, nil
}

// resolveArtifactPath resolves the artifact path from the current spec directory.
// It uses DetectCurrentSpec to find the spec directory and constructs the artifact path.
// Returns the path, spec metadata, and any error.
func resolveArtifactPath(artType validation.ArtifactType, specsDir string) (string, *spec.Metadata, error) {
	metadata, err := spec.DetectCurrentSpec(specsDir)
	if err != nil {
		return "", nil, fmt.Errorf("failed to detect spec: %w\nHint: Run from a spec branch or specify the path explicitly", err)
	}

	// Construct path to artifact
	artifactFilename := string(artType) + ".yaml"
	artifactPath := filepath.Join(metadata.Directory, artifactFilename)

	return artifactPath, metadata, nil
}

// runArtifactCommand executes the artifact validation command.
func runArtifactCommand(args []string, configPath string, out, errOut io.Writer) error {
	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Fprintf(errOut, "Error loading config: %v\n", err)
		return NewExitError(ExitInvalidArguments)
	}

	// Parse arguments
	parsed, err := parseArtifactArgs(args, cfg.SpecsDir)
	if err != nil {
		fmt.Fprintf(errOut, "Error: %v\n", err)
		if strings.Contains(err.Error(), "invalid artifact type") {
			fmt.Fprintf(errOut, "Valid types: %s\n", strings.Join(validation.ValidArtifactTypes(), ", "))
		}
		return NewExitError(ExitInvalidArguments)
	}

	// Handle --schema flag
	if artifactSchemaFlag {
		return printSchema(parsed.artType, out)
	}

	// Check if file exists
	if _, err := os.Stat(parsed.filePath); os.IsNotExist(err) {
		fmt.Fprintf(errOut, "Error: file not found: %s\n", parsed.filePath)
		if parsed.isTypeOnly {
			fmt.Fprintf(errOut, "Hint: The %s.yaml file does not exist in the detected spec directory\n", parsed.artType)
		}
		return NewExitError(ExitInvalidArguments)
	}

	// Check if path is a directory
	if info, _ := os.Stat(parsed.filePath); info != nil && info.IsDir() {
		fmt.Fprintf(errOut, "Error: path is a directory, not a file: %s\n", parsed.filePath)
		fmt.Fprintf(errOut, "Hint: Specify the full path to the %s.yaml file\n", parsed.artType)
		return NewExitError(ExitInvalidArguments)
	}

	// Print spec identification for auto-detected paths
	printSpecIdentification(parsed, out)

	// Handle --fix flag
	if artifactFixFlag {
		return runAutoFix(parsed.filePath, parsed.artType, out, errOut)
	}

	// Create validator
	validator, err := validation.NewArtifactValidator(parsed.artType)
	if err != nil {
		fmt.Fprintf(errOut, "Error: %v\n", err)
		return NewExitError(ExitInvalidArguments)
	}

	// Run validation
	result := validator.Validate(parsed.filePath)

	// Format and display results
	return formatValidationResult(result, parsed.filePath, parsed.artType, out, errOut)
}

// printSpecIdentification prints the spec identification message when using auto-detection.
func printSpecIdentification(parsed *artifactArgs, out io.Writer) {
	if parsed.specMetadata == nil {
		return
	}

	fmt.Fprintln(out, parsed.specMetadata.FormatInfo())
}

// printSchema prints the schema for an artifact type.
func printSchema(artType validation.ArtifactType, out io.Writer) error {
	schema, err := validation.GetSchema(artType)
	if err != nil {
		return fmt.Errorf("getting schema for %s: %w", artType, err)
	}

	fmt.Fprintf(out, "Schema for %s artifacts\n", artType)
	fmt.Fprintf(out, "%s\n\n", strings.Repeat("=", 40))
	fmt.Fprintf(out, "%s\n\n", schema.Description)

	fmt.Fprintf(out, "Fields:\n")
	fmt.Fprintf(out, "%s\n", strings.Repeat("-", 40))

	for _, field := range schema.Fields {
		printSchemaField(field, "", out)
	}

	return nil
}

// printSchemaField prints a single schema field with indentation.
func printSchemaField(field validation.SchemaField, indent string, out io.Writer) {
	required := ""
	if field.Required {
		required = " (required)"
	}

	typeStr := string(field.Type)
	if len(field.Enum) > 0 {
		typeStr = fmt.Sprintf("enum[%s]", strings.Join(field.Enum, ", "))
	}

	fmt.Fprintf(out, "%s%s: %s%s\n", indent, field.Name, typeStr, required)

	if field.Description != "" {
		fmt.Fprintf(out, "%s  # %s\n", indent, field.Description)
	}

	// Print children for nested fields
	for _, child := range field.Children {
		printSchemaField(child, indent+"  ", out)
	}
}

// formatValidationResult formats and displays the validation result.
func formatValidationResult(result *validation.ValidationResult, filePath string, artType validation.ArtifactType, out, errOut io.Writer) error {
	if result.Valid {
		// Success output
		green := color.New(color.FgGreen).SprintFunc()
		fmt.Fprintf(out, "%s %s is valid\n", green("✓"), filePath)

		if result.Summary != nil {
			fmt.Fprintf(out, "\nSummary:\n")
			for key, value := range result.Summary.Counts {
				displayKey := strings.ReplaceAll(key, "_", " ")
				fmt.Fprintf(out, "  %s: %d\n", displayKey, value)
			}
		}

		// Display warnings if any
		displayWarnings(result, out)

		return nil
	}

	// Error output
	red := color.New(color.FgRed).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	fmt.Fprintf(errOut, "%s %s has %d error(s)\n\n", red("✗"), filePath, len(result.Errors))

	for i, err := range result.Errors {
		fmt.Fprintf(errOut, "Error %d:\n", i+1)

		// Location
		if err.Line > 0 {
			fmt.Fprintf(errOut, "  Location: line %d", err.Line)
			if err.Column > 0 {
				fmt.Fprintf(errOut, ", column %d", err.Column)
			}
			fmt.Fprintf(errOut, "\n")
		}

		// Path
		if err.Path != "" {
			fmt.Fprintf(errOut, "  Path: %s\n", err.Path)
		}

		// Message
		fmt.Fprintf(errOut, "  Message: %s\n", err.Message)

		// Expected/Actual
		if err.Expected != "" {
			fmt.Fprintf(errOut, "  Expected: %s\n", err.Expected)
		}
		if err.Actual != "" {
			fmt.Fprintf(errOut, "  Got: %s\n", err.Actual)
		}

		// Hint
		if err.Hint != "" {
			fmt.Fprintf(errOut, "  %s %s\n", yellow("Hint:"), err.Hint)
		}

		fmt.Fprintf(errOut, "\n")
	}

	return NewExitError(ExitValidationFailed)
}

// runAutoFix runs the auto-fix operation on an artifact file.
func runAutoFix(filePath string, artType validation.ArtifactType, out, errOut io.Writer) error {
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	fmt.Fprintf(out, "Auto-fixing %s...\n\n", filePath)

	result, err := validation.FixArtifact(filePath, artType)
	if err != nil {
		fmt.Fprintf(errOut, "Error: %v\n", err)
		return NewExitError(ExitValidationFailed)
	}

	// Show fixes applied
	if len(result.FixesApplied) > 0 {
		fmt.Fprintf(out, "%s Applied %d fix(es):\n", green("✓"), len(result.FixesApplied))
		for _, fix := range result.FixesApplied {
			fmt.Fprintf(out, "  • [%s] %s: %s\n", fix.Type, fix.Path, fix.After)
		}
		fmt.Fprintf(out, "\n")
	} else {
		fmt.Fprintf(out, "%s No fixable issues found\n\n", yellow("•"))
	}

	// Show remaining errors
	if len(result.RemainingErrors) > 0 {
		fmt.Fprintf(errOut, "%s %d unfixable error(s) remain:\n\n", red("✗"), len(result.RemainingErrors))
		for i, err := range result.RemainingErrors {
			fmt.Fprintf(errOut, "Error %d:\n", i+1)
			if err.Line > 0 {
				fmt.Fprintf(errOut, "  Location: line %d\n", err.Line)
			}
			if err.Path != "" {
				fmt.Fprintf(errOut, "  Path: %s\n", err.Path)
			}
			fmt.Fprintf(errOut, "  Message: %s\n", err.Message)
			if err.Hint != "" {
				fmt.Fprintf(errOut, "  Hint: %s\n", err.Hint)
			}
			fmt.Fprintf(errOut, "\n")
		}
		return NewExitError(ExitValidationFailed)
	}

	// All issues fixed
	if result.Modified {
		fmt.Fprintf(out, "%s File fixed and saved: %s\n", green("✓"), filePath)
	} else {
		fmt.Fprintf(out, "%s File is valid (no changes needed)\n", green("✓"))
	}

	return nil
}

// displayWarnings displays validation warnings if any exist.
func displayWarnings(result *validation.ValidationResult, out io.Writer) {
	if !result.HasWarnings() {
		return
	}

	yellow := color.New(color.FgYellow).SprintFunc()
	fmt.Fprintf(out, "\n%s %d warning(s):\n", yellow("⚠"), len(result.Warnings))

	for _, warning := range result.Warnings {
		fmt.Fprintf(out, "  • ")
		if warning.Line > 0 {
			fmt.Fprintf(out, "line %d: ", warning.Line)
		}
		fmt.Fprintf(out, "%s\n", warning.Message)
		if warning.Hint != "" {
			fmt.Fprintf(out, "    %s %s\n", yellow("Hint:"), warning.Hint)
		}
	}
}
