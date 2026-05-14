package commands

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/ariel-frischer/autospec/internal/prereqs"
)

// RequiredVars defines which prereqs context fields are required by each command.
// Commands not listed here require no specific prereqs context.
var RequiredVars = map[string][]string{
	"autospec.specify":      {},                                                                          // No prereqs required
	"autospec.plan":         {"FeatureDir", "FeatureSpec", "AutospecVersion", "CreatedDate"},             // Needs spec
	"autospec.tasks":        {"FeatureDir", "FeatureSpec", "ImplPlan", "AutospecVersion", "CreatedDate"}, // Needs plan
	"autospec.implement":    {"FeatureDir", "TasksFile"},                                                 // Needs tasks; constitution is optional
	"autospec.checklist":    {"FeatureDir", "FeatureSpec"},                                               // Needs spec
	"autospec.clarify":      {"FeatureDir", "FeatureSpec"},                                               // Needs spec
	"autospec.analyze":      {"FeatureDir", "FeatureSpec"},                                               // Needs spec
	"autospec.constitution": {"AutospecVersion", "CreatedDate"},                                          // Minimal context
}

// RenderTemplate renders a command template using the provided prereqs context.
// The template uses Go text/template syntax with {{.FieldName}} placeholders.
// After rendering, YAML frontmatter is stripped from the output since it's only
// metadata (description, version) and would cause issues if passed to CLI tools
// that interpret "---" as a flag delimiter.
func RenderTemplate(content []byte, ctx *prereqs.Context) ([]byte, error) {
	if ctx == nil {
		return nil, fmt.Errorf("prereqs context is nil")
	}

	tmpl, err := template.New("command").Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	// Strip YAML frontmatter after rendering - it's metadata only and would cause
	// issues with CLI tools that interpret "---" as a flag delimiter.
	return StripFrontmatter(buf.Bytes()), nil
}

// GetRequiredVars returns the list of required context fields for a command.
// Returns an empty slice if the command has no specific requirements.
func GetRequiredVars(commandName string) []string {
	if vars, ok := RequiredVars[commandName]; ok {
		return vars
	}
	return []string{}
}

// ValidateRequirements checks that the prereqs context contains all required
// fields for the given command. Returns an error listing missing fields if any.
func ValidateRequirements(commandName string, ctx *prereqs.Context) error {
	if ctx == nil {
		return fmt.Errorf("prereqs context is nil")
	}

	required := GetRequiredVars(commandName)
	if len(required) == 0 {
		return nil
	}

	var missing []string
	for _, field := range required {
		if !hasContextField(ctx, field) {
			missing = append(missing, field)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required context for %s: %s\nRun the prerequisite command first or ensure you're in a feature directory",
			commandName, strings.Join(missing, ", "))
	}

	return nil
}

// hasContextField checks if a context field is populated.
func hasContextField(ctx *prereqs.Context, field string) bool {
	switch field {
	case "FeatureDir":
		return ctx.FeatureDir != ""
	case "FeatureSpec":
		return ctx.FeatureSpec != ""
	case "ImplPlan":
		return ctx.ImplPlan != ""
	case "TasksFile":
		return ctx.TasksFile != ""
	case "ConstitutionFile":
		return ctx.ConstitutionFile != ""
	case "AutospecVersion":
		return ctx.AutospecVersion != ""
	case "CreatedDate":
		return ctx.CreatedDate != ""
	case "IsGitRepo":
		return true // bool is always "set"
	case "AvailableDocs":
		return true // slice is always "set"
	default:
		return false
	}
}

// RenderAndValidate renders a command template after validating all requirements.
// This is the primary entry point for command template rendering.
func RenderAndValidate(commandName string, content []byte, ctx *prereqs.Context) ([]byte, error) {
	if err := ValidateRequirements(commandName, ctx); err != nil {
		return nil, err
	}
	return RenderTemplate(content, ctx)
}
