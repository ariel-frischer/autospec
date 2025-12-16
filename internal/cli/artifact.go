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
	Use:   "artifact <type> [path]",
	Short: "Validate YAML artifacts against their schemas",
	Long: `Validate YAML artifacts (spec, plan, tasks) against their schemas.

Types:
  spec   - Feature specification (spec.yaml)
  plan   - Implementation plan (plan.yaml)
  tasks  - Task breakdown (tasks.yaml)

Validates:
  - Valid YAML syntax
  - Required fields present for artifact type
  - Field types correct (strings, lists, enums)
  - Cross-references valid (e.g. task dependencies exist)

Output:
  - Success message with artifact summary on valid artifacts
  - Detailed errors with line numbers and hints on invalid artifacts

Exit Codes:
  0 - Success (artifact is valid)
  1 - Validation failed (artifact has errors)
  3 - Invalid arguments (unknown type or missing file)`,
	Example: `  # Validate a spec artifact
  autospec artifact spec specs/001-feature/spec.yaml

  # Validate a plan artifact
  autospec artifact plan specs/001-feature/plan.yaml

  # Validate tasks with dependency checking
  autospec artifact tasks specs/001-feature/tasks.yaml

  # Show schema for an artifact type
  autospec artifact spec --schema
  autospec artifact plan --schema
  autospec artifact tasks --schema

  # Auto-fix common issues
  autospec artifact spec specs/001-feature/spec.yaml --fix`,
	Args:          cobra.RangeArgs(1, 2),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runArtifactCommand(args, cmd.OutOrStdout(), cmd.ErrOrStderr())
	},
}

func init() {
	rootCmd.AddCommand(artifactCmd)
	artifactCmd.Flags().BoolVar(&artifactSchemaFlag, "schema", false, "Print the expected schema for the artifact type")
	artifactCmd.Flags().BoolVar(&artifactFixFlag, "fix", false, "Auto-fix common issues (missing optional fields, formatting)")
}

// runArtifactCommand executes the artifact validation command.
func runArtifactCommand(args []string, out, errOut io.Writer) error {
	artifactType := args[0]

	// Parse artifact type
	artType, err := validation.ParseArtifactType(artifactType)
	if err != nil {
		fmt.Fprintf(errOut, "Error: %v\n", err)
		fmt.Fprintf(errOut, "Valid types: %s\n", strings.Join(validation.ValidArtifactTypes(), ", "))
		return &exitError{code: ExitInvalidArguments}
	}

	// Handle --schema flag
	if artifactSchemaFlag {
		return printSchema(artType, out)
	}

	// Require file path for validation
	if len(args) < 2 {
		fmt.Fprintf(errOut, "Error: file path required for validation\n")
		fmt.Fprintf(errOut, "Usage: autospec artifact %s <path>\n", artifactType)
		fmt.Fprintf(errOut, "       autospec artifact %s --schema  (to view schema)\n", artifactType)
		return &exitError{code: ExitInvalidArguments}
	}

	filePath := args[1]

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Fprintf(errOut, "Error: file not found: %s\n", filePath)
		return &exitError{code: ExitInvalidArguments}
	}

	// Check if path is a directory
	if info, _ := os.Stat(filePath); info != nil && info.IsDir() {
		fmt.Fprintf(errOut, "Error: path is a directory, not a file: %s\n", filePath)
		fmt.Fprintf(errOut, "Hint: Specify the full path to the %s.yaml file\n", artifactType)
		return &exitError{code: ExitInvalidArguments}
	}

	// Handle --fix flag
	if artifactFixFlag {
		return runAutoFix(filePath, artType, out, errOut)
	}

	// Create validator
	validator, err := validation.NewArtifactValidator(artType)
	if err != nil {
		fmt.Fprintf(errOut, "Error: %v\n", err)
		return &exitError{code: ExitInvalidArguments}
	}

	// Run validation
	result := validator.Validate(filePath)

	// Format and display results
	return formatValidationResult(result, filePath, artType, out, errOut)
}

// printSchema prints the schema for an artifact type.
func printSchema(artType validation.ArtifactType, out io.Writer) error {
	schema, err := validation.GetSchema(artType)
	if err != nil {
		return err
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

	return &exitError{code: ExitValidationFailed}
}

// exitError is a custom error type that carries an exit code.
type exitError struct {
	code int
}

func (e *exitError) Error() string {
	return fmt.Sprintf("exit code %d", e.code)
}

// ExitCode returns the exit code from an error.
func ExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	if e, ok := err.(*exitError); ok {
		return e.code
	}
	return ExitValidationFailed
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
		return &exitError{code: ExitValidationFailed}
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
		return &exitError{code: ExitValidationFailed}
	}

	// All issues fixed
	if result.Modified {
		fmt.Fprintf(out, "%s File fixed and saved: %s\n", green("✓"), filePath)
	} else {
		fmt.Fprintf(out, "%s File is valid (no changes needed)\n", green("✓"))
	}

	return nil
}
