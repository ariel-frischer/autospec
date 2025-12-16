package validation

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidationError represents a single validation error with location and context.
type ValidationError struct {
	Path     string // JSON-path style field location (e.g., "user_stories[0].id")
	Line     int    // 1-based line number in source file
	Column   int    // 1-based column number in source file
	Message  string // Human-readable error description
	Expected string // What was expected (type, value, format)
	Actual   string // What was found
	Hint     string // Suggestion for fixing the error
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	var sb strings.Builder
	if e.Line > 0 {
		sb.WriteString(fmt.Sprintf("line %d", e.Line))
		if e.Column > 0 {
			sb.WriteString(fmt.Sprintf(":%d", e.Column))
		}
		sb.WriteString(": ")
	}
	if e.Path != "" {
		sb.WriteString(fmt.Sprintf("%s: ", e.Path))
	}
	sb.WriteString(e.Message)
	return sb.String()
}

// FormatFull returns a detailed formatted error message.
func (e *ValidationError) FormatFull() string {
	var sb strings.Builder

	// Location
	if e.Line > 0 {
		sb.WriteString(fmt.Sprintf("  Line %d", e.Line))
		if e.Column > 0 {
			sb.WriteString(fmt.Sprintf(", Column %d", e.Column))
		}
		sb.WriteString("\n")
	}

	// Path
	if e.Path != "" {
		sb.WriteString(fmt.Sprintf("  Path: %s\n", e.Path))
	}

	// Message
	sb.WriteString(fmt.Sprintf("  Error: %s\n", e.Message))

	// Expected/Actual
	if e.Expected != "" {
		sb.WriteString(fmt.Sprintf("  Expected: %s\n", e.Expected))
	}
	if e.Actual != "" {
		sb.WriteString(fmt.Sprintf("  Got: %s\n", e.Actual))
	}

	// Hint
	if e.Hint != "" {
		sb.WriteString(fmt.Sprintf("  Hint: %s\n", e.Hint))
	}

	return sb.String()
}

// ArtifactSummary contains summary statistics about a validated artifact.
type ArtifactSummary struct {
	Type   ArtifactType   // Type of artifact validated
	Counts map[string]int // Key counts (stories, tasks, phases, etc.)
}

// ValidationResult represents the complete validation outcome for an artifact.
type ValidationResult struct {
	Valid   bool               // True if artifact passed all validation
	Errors  []*ValidationError // List of validation errors found
	Summary *ArtifactSummary   // Summary statistics (populated on valid artifacts)
}

// HasErrors returns true if there are any validation errors.
func (r *ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// AddError adds a validation error to the result.
func (r *ValidationResult) AddError(err *ValidationError) {
	r.Errors = append(r.Errors, err)
	r.Valid = false
}

// ArtifactValidator defines the interface for artifact validation.
type ArtifactValidator interface {
	// Validate validates the artifact at the given path.
	Validate(path string) *ValidationResult
	// Type returns the artifact type this validator handles.
	Type() ArtifactType
}

// NewArtifactValidator creates a validator for the given artifact type.
func NewArtifactValidator(artifactType ArtifactType) (ArtifactValidator, error) {
	switch artifactType {
	case ArtifactTypeSpec:
		return &SpecValidator{}, nil
	case ArtifactTypePlan:
		return &PlanValidator{}, nil
	case ArtifactTypeTasks:
		return &TasksValidator{}, nil
	default:
		return nil, fmt.Errorf("unknown artifact type: %s", artifactType)
	}
}

// baseValidator provides common validation functionality.
type baseValidator struct {
	artifactType ArtifactType
}

// Type returns the artifact type.
func (v *baseValidator) Type() ArtifactType {
	return v.artifactType
}

// parseYAMLFile parses a YAML file and returns the root node.
func parseYAMLFile(path string) (*yaml.Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	return parseYAMLReader(f)
}

// parseYAMLReader parses YAML from a reader and returns the root node.
func parseYAMLReader(r io.Reader) (*yaml.Node, error) {
	var node yaml.Node
	dec := yaml.NewDecoder(r)
	if err := dec.Decode(&node); err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("file is empty or contains only comments")
		}
		return nil, err
	}
	return &node, nil
}

// findNode finds a node by key in a mapping node.
func findNode(root *yaml.Node, key string) *yaml.Node {
	if root == nil {
		return nil
	}

	// Handle document node
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		return findNode(root.Content[0], key)
	}

	// Must be a mapping node
	if root.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			return root.Content[i+1]
		}
	}
	return nil
}

// getNodeLine returns the line number of a node (1-based).
func getNodeLine(node *yaml.Node) int {
	if node == nil {
		return 0
	}
	return node.Line
}

// getNodeColumn returns the column number of a node (1-based).
func getNodeColumn(node *yaml.Node) int {
	if node == nil {
		return 0
	}
	return node.Column
}

// validateRequiredField checks if a required field exists in a mapping node.
func validateRequiredField(root *yaml.Node, fieldName string, result *ValidationResult) *yaml.Node {
	node := findNode(root, fieldName)
	if node == nil {
		result.AddError(&ValidationError{
			Path:    fieldName,
			Line:    getNodeLine(root),
			Message: fmt.Sprintf("missing required field: %s", fieldName),
			Hint:    fmt.Sprintf("Add the '%s' field to your YAML file", fieldName),
		})
		return nil
	}
	return node
}

// validateFieldType checks if a field has the expected YAML node kind.
func validateFieldType(node *yaml.Node, path string, expectedKind yaml.Kind, expectedType string, result *ValidationResult) bool {
	if node == nil {
		return false
	}

	if node.Kind != expectedKind {
		actualType := nodeKindToString(node.Kind)
		result.AddError(&ValidationError{
			Path:     path,
			Line:     getNodeLine(node),
			Column:   getNodeColumn(node),
			Message:  fmt.Sprintf("wrong type for field '%s'", path),
			Expected: expectedType,
			Actual:   actualType,
			Hint:     fmt.Sprintf("Change '%s' to be a %s", path, expectedType),
		})
		return false
	}
	return true
}

// validateEnumValue checks if a string value is one of the allowed enum values.
func validateEnumValue(node *yaml.Node, path string, allowedValues []string, result *ValidationResult) bool {
	if node == nil {
		return false
	}

	value := node.Value
	for _, allowed := range allowedValues {
		if value == allowed {
			return true
		}
	}

	result.AddError(&ValidationError{
		Path:     path,
		Line:     getNodeLine(node),
		Column:   getNodeColumn(node),
		Message:  fmt.Sprintf("invalid value for field '%s'", path),
		Expected: fmt.Sprintf("one of: %s", strings.Join(allowedValues, ", ")),
		Actual:   fmt.Sprintf("'%s'", value),
		Hint:     fmt.Sprintf("Use one of the valid values: %s", strings.Join(allowedValues, ", ")),
	})
	return false
}

// nodeKindToString converts a yaml.Kind to a human-readable string.
func nodeKindToString(kind yaml.Kind) string {
	switch kind {
	case yaml.DocumentNode:
		return "document"
	case yaml.SequenceNode:
		return "array"
	case yaml.MappingNode:
		return "object"
	case yaml.ScalarNode:
		return "scalar"
	case yaml.AliasNode:
		return "alias"
	default:
		return "unknown"
	}
}

// getRootMapping returns the root mapping node from a document.
func getRootMapping(root *yaml.Node) *yaml.Node {
	if root == nil {
		return nil
	}
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		return root.Content[0]
	}
	if root.Kind == yaml.MappingNode {
		return root
	}
	return nil
}
