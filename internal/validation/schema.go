package validation

import "fmt"

// ArtifactType represents the type of artifact to validate.
type ArtifactType string

const (
	// ArtifactTypeSpec represents spec.yaml artifacts.
	ArtifactTypeSpec ArtifactType = "spec"
	// ArtifactTypePlan represents plan.yaml artifacts.
	ArtifactTypePlan ArtifactType = "plan"
	// ArtifactTypeTasks represents tasks.yaml artifacts.
	ArtifactTypeTasks ArtifactType = "tasks"
)

// FieldType represents the expected type of a schema field.
type FieldType string

const (
	FieldTypeString FieldType = "string"
	FieldTypeInt    FieldType = "int"
	FieldTypeBool   FieldType = "bool"
	FieldTypeArray  FieldType = "array"
	FieldTypeObject FieldType = "object"
)

// SchemaField defines a field in an artifact schema.
type SchemaField struct {
	Name        string    // Field name in YAML
	Type        FieldType // Expected type
	Required    bool      // Whether field must be present
	Pattern     string    // Regex pattern for string validation (optional)
	Enum        []string  // Valid values for enum fields (optional)
	Description string    // Human-readable description
	Children    []SchemaField // Nested fields for object/array types
}

// Schema represents the complete schema for an artifact type.
type Schema struct {
	Type        ArtifactType
	Description string
	Fields      []SchemaField
}

// SpecSchema defines the schema for spec.yaml artifacts.
var SpecSchema = Schema{
	Type:        ArtifactTypeSpec,
	Description: "Feature specification artifact containing user stories, requirements, and acceptance criteria",
	Fields: []SchemaField{
		{
			Name:        "feature",
			Type:        FieldTypeObject,
			Required:    true,
			Description: "Feature metadata including branch name, creation date, and status",
			Children: []SchemaField{
				{Name: "branch", Type: FieldTypeString, Required: true, Description: "Git branch name for the feature"},
				{Name: "created", Type: FieldTypeString, Required: true, Description: "Creation date (YYYY-MM-DD)"},
				{Name: "status", Type: FieldTypeString, Required: false, Enum: []string{"Draft", "Review", "Approved", "Implemented"}, Description: "Feature status"},
				{Name: "input", Type: FieldTypeString, Required: false, Description: "Original input description"},
			},
		},
		{
			Name:        "user_stories",
			Type:        FieldTypeArray,
			Required:    true,
			Description: "List of user stories defining feature requirements",
			Children: []SchemaField{
				{Name: "id", Type: FieldTypeString, Required: true, Pattern: `^US-\d+$`, Description: "Story ID (US-NNN format)"},
				{Name: "title", Type: FieldTypeString, Required: true, Description: "Short story title"},
				{Name: "priority", Type: FieldTypeString, Required: true, Enum: []string{"P0", "P1", "P2", "P3"}, Description: "Story priority"},
				{Name: "as_a", Type: FieldTypeString, Required: true, Description: "User role"},
				{Name: "i_want", Type: FieldTypeString, Required: true, Description: "Desired functionality"},
				{Name: "so_that", Type: FieldTypeString, Required: true, Description: "Business value"},
				{Name: "acceptance_scenarios", Type: FieldTypeArray, Required: true, Description: "Given/When/Then scenarios"},
			},
		},
		{
			Name:        "requirements",
			Type:        FieldTypeObject,
			Required:    true,
			Description: "Functional and non-functional requirements",
			Children: []SchemaField{
				{Name: "functional", Type: FieldTypeArray, Required: true, Description: "List of functional requirements"},
				{Name: "non_functional", Type: FieldTypeArray, Required: false, Description: "List of non-functional requirements"},
			},
		},
		{
			Name:        "key_entities",
			Type:        FieldTypeArray,
			Required:    false,
			Description: "Domain entities with their attributes",
		},
		{
			Name:        "success_criteria",
			Type:        FieldTypeArray,
			Required:    false,
			Description: "Measurable success criteria",
		},
		{
			Name:        "edge_cases",
			Type:        FieldTypeArray,
			Required:    false,
			Description: "Edge cases and expected behaviors",
		},
		{
			Name:        "assumptions",
			Type:        FieldTypeArray,
			Required:    false,
			Description: "Assumptions made during specification",
		},
		{
			Name:        "constraints",
			Type:        FieldTypeArray,
			Required:    false,
			Description: "Technical or business constraints",
		},
		{
			Name:        "out_of_scope",
			Type:        FieldTypeArray,
			Required:    false,
			Description: "Items explicitly excluded from scope",
		},
		{
			Name:        "_meta",
			Type:        FieldTypeObject,
			Required:    false,
			Description: "Artifact metadata",
			Children: []SchemaField{
				{Name: "version", Type: FieldTypeString, Required: false, Description: "Schema version"},
				{Name: "generator", Type: FieldTypeString, Required: false, Description: "Generator tool name"},
				{Name: "generator_version", Type: FieldTypeString, Required: false, Description: "Generator version"},
				{Name: "created", Type: FieldTypeString, Required: false, Description: "Creation timestamp"},
				{Name: "artifact_type", Type: FieldTypeString, Required: false, Enum: []string{"spec"}, Description: "Artifact type"},
			},
		},
	},
}

// PlanSchema defines the schema for plan.yaml artifacts.
var PlanSchema = Schema{
	Type:        ArtifactTypePlan,
	Description: "Implementation plan artifact containing technical context, phases, and deliverables",
	Fields: []SchemaField{
		{
			Name:        "plan",
			Type:        FieldTypeObject,
			Required:    true,
			Description: "Plan metadata including branch name and spec reference",
			Children: []SchemaField{
				{Name: "branch", Type: FieldTypeString, Required: true, Description: "Git branch name"},
				{Name: "created", Type: FieldTypeString, Required: false, Description: "Creation date"},
				{Name: "spec_path", Type: FieldTypeString, Required: true, Description: "Path to related spec file"},
			},
		},
		{
			Name:        "summary",
			Type:        FieldTypeString,
			Required:    true,
			Description: "Executive summary of the implementation plan",
		},
		{
			Name:        "technical_context",
			Type:        FieldTypeObject,
			Required:    true,
			Description: "Technical context including language, framework, and dependencies",
			Children: []SchemaField{
				{Name: "language", Type: FieldTypeString, Required: false, Description: "Primary programming language"},
				{Name: "framework", Type: FieldTypeString, Required: false, Description: "Main framework"},
				{Name: "primary_dependencies", Type: FieldTypeArray, Required: false, Description: "Key dependencies"},
				{Name: "storage", Type: FieldTypeString, Required: false, Description: "Data storage solution"},
				{Name: "testing", Type: FieldTypeObject, Required: false, Description: "Testing approach"},
				{Name: "target_platform", Type: FieldTypeString, Required: false, Description: "Target platforms"},
				{Name: "project_type", Type: FieldTypeString, Required: false, Description: "Project type"},
				{Name: "performance_goals", Type: FieldTypeString, Required: false, Description: "Performance targets"},
				{Name: "constraints", Type: FieldTypeArray, Required: false, Description: "Technical constraints"},
				{Name: "scale_scope", Type: FieldTypeString, Required: false, Description: "Scale expectations"},
			},
		},
		{
			Name:        "implementation_phases",
			Type:        FieldTypeArray,
			Required:    false,
			Description: "Ordered list of implementation phases",
			Children: []SchemaField{
				{Name: "phase", Type: FieldTypeInt, Required: true, Description: "Phase number"},
				{Name: "name", Type: FieldTypeString, Required: true, Description: "Phase name"},
				{Name: "goal", Type: FieldTypeString, Required: false, Description: "Phase goal"},
				{Name: "deliverables", Type: FieldTypeArray, Required: false, Description: "Phase deliverables"},
				{Name: "dependencies", Type: FieldTypeArray, Required: false, Description: "Dependencies on other phases"},
			},
		},
		{
			Name:        "constitution_check",
			Type:        FieldTypeObject,
			Required:    false,
			Description: "Constitution compliance check results",
		},
		{
			Name:        "research_findings",
			Type:        FieldTypeObject,
			Required:    false,
			Description: "Research findings and technical decisions",
		},
		{
			Name:        "data_model",
			Type:        FieldTypeObject,
			Required:    false,
			Description: "Data model entities and relationships",
		},
		{
			Name:        "api_contracts",
			Type:        FieldTypeObject,
			Required:    false,
			Description: "API endpoint contracts",
		},
		{
			Name:        "project_structure",
			Type:        FieldTypeObject,
			Required:    false,
			Description: "Project file structure",
		},
		{
			Name:        "risks",
			Type:        FieldTypeArray,
			Required:    false,
			Description: "Identified risks and mitigations",
		},
		{
			Name:        "open_questions",
			Type:        FieldTypeArray,
			Required:    false,
			Description: "Open questions requiring resolution",
		},
		{
			Name:        "_meta",
			Type:        FieldTypeObject,
			Required:    false,
			Description: "Artifact metadata",
			Children: []SchemaField{
				{Name: "version", Type: FieldTypeString, Required: false, Description: "Schema version"},
				{Name: "generator", Type: FieldTypeString, Required: false, Description: "Generator tool name"},
				{Name: "generator_version", Type: FieldTypeString, Required: false, Description: "Generator version"},
				{Name: "created", Type: FieldTypeString, Required: false, Description: "Creation timestamp"},
				{Name: "artifact_type", Type: FieldTypeString, Required: false, Enum: []string{"plan"}, Description: "Artifact type"},
			},
		},
	},
}

// TasksSchema defines the schema for tasks.yaml artifacts.
var TasksSchema = Schema{
	Type:        ArtifactTypeTasks,
	Description: "Task breakdown artifact containing phases, tasks, and dependencies",
	Fields: []SchemaField{
		{
			Name:        "tasks",
			Type:        FieldTypeObject,
			Required:    true,
			Description: "Tasks metadata including branch name and file references",
			Children: []SchemaField{
				{Name: "branch", Type: FieldTypeString, Required: true, Description: "Git branch name"},
				{Name: "created", Type: FieldTypeString, Required: false, Description: "Creation date"},
				{Name: "spec_path", Type: FieldTypeString, Required: false, Description: "Path to related spec file"},
				{Name: "plan_path", Type: FieldTypeString, Required: false, Description: "Path to related plan file"},
			},
		},
		{
			Name:        "summary",
			Type:        FieldTypeObject,
			Required:    true,
			Description: "Summary statistics for the task breakdown",
			Children: []SchemaField{
				{Name: "total_tasks", Type: FieldTypeInt, Required: false, Description: "Total number of tasks"},
				{Name: "total_phases", Type: FieldTypeInt, Required: false, Description: "Total number of phases"},
				{Name: "parallel_opportunities", Type: FieldTypeInt, Required: false, Description: "Number of parallel execution opportunities"},
				{Name: "estimated_complexity", Type: FieldTypeString, Required: false, Description: "Overall complexity estimate"},
			},
		},
		{
			Name:        "phases",
			Type:        FieldTypeArray,
			Required:    true,
			Description: "Ordered list of implementation phases with tasks",
			Children: []SchemaField{
				{Name: "number", Type: FieldTypeInt, Required: true, Description: "Phase number"},
				{Name: "title", Type: FieldTypeString, Required: true, Description: "Phase title"},
				{Name: "purpose", Type: FieldTypeString, Required: false, Description: "Phase purpose"},
				{Name: "story_reference", Type: FieldTypeString, Required: false, Description: "Related user story ID"},
				{Name: "independent_test", Type: FieldTypeString, Required: false, Description: "Independent test description"},
				{Name: "tasks", Type: FieldTypeArray, Required: true, Description: "List of tasks in this phase"},
			},
		},
		{
			Name:        "dependencies",
			Type:        FieldTypeObject,
			Required:    false,
			Description: "Dependency relationships between stories and phases",
		},
		{
			Name:        "parallel_execution",
			Type:        FieldTypeArray,
			Required:    false,
			Description: "Parallel execution groups",
		},
		{
			Name:        "implementation_strategy",
			Type:        FieldTypeObject,
			Required:    false,
			Description: "Implementation strategy including MVP scope",
		},
		{
			Name:        "_meta",
			Type:        FieldTypeObject,
			Required:    false,
			Description: "Artifact metadata",
			Children: []SchemaField{
				{Name: "version", Type: FieldTypeString, Required: false, Description: "Schema version"},
				{Name: "generator", Type: FieldTypeString, Required: false, Description: "Generator tool name"},
				{Name: "generator_version", Type: FieldTypeString, Required: false, Description: "Generator version"},
				{Name: "created", Type: FieldTypeString, Required: false, Description: "Creation timestamp"},
				{Name: "artifact_type", Type: FieldTypeString, Required: false, Enum: []string{"tasks"}, Description: "Artifact type"},
			},
		},
	},
}

// TaskFieldSchema defines the schema for individual task fields.
var TaskFieldSchema = []SchemaField{
	{Name: "id", Type: FieldTypeString, Required: true, Pattern: `^T\d+$`, Description: "Task ID (TNNN format)"},
	{Name: "title", Type: FieldTypeString, Required: true, Description: "Task title"},
	{Name: "status", Type: FieldTypeString, Required: true, Enum: []string{"Pending", "InProgress", "Completed", "Blocked"}, Description: "Task status"},
	{Name: "type", Type: FieldTypeString, Required: true, Enum: []string{"setup", "implementation", "test", "documentation", "refactor"}, Description: "Task type"},
	{Name: "parallel", Type: FieldTypeBool, Required: false, Description: "Whether task can run in parallel"},
	{Name: "story_id", Type: FieldTypeString, Required: false, Description: "Related user story ID"},
	{Name: "file_path", Type: FieldTypeString, Required: false, Description: "Primary file path for this task"},
	{Name: "dependencies", Type: FieldTypeArray, Required: false, Description: "List of task IDs this task depends on"},
	{Name: "acceptance_criteria", Type: FieldTypeArray, Required: false, Description: "Acceptance criteria for the task"},
}

// GetSchema returns the schema for the given artifact type.
func GetSchema(artifactType ArtifactType) (*Schema, error) {
	switch artifactType {
	case ArtifactTypeSpec:
		return &SpecSchema, nil
	case ArtifactTypePlan:
		return &PlanSchema, nil
	case ArtifactTypeTasks:
		return &TasksSchema, nil
	default:
		return nil, fmt.Errorf("unknown artifact type: %s", artifactType)
	}
}

// ParseArtifactType parses a string into an ArtifactType.
func ParseArtifactType(s string) (ArtifactType, error) {
	switch s {
	case "spec":
		return ArtifactTypeSpec, nil
	case "plan":
		return ArtifactTypePlan, nil
	case "tasks":
		return ArtifactTypeTasks, nil
	default:
		return "", fmt.Errorf("invalid artifact type: %s (valid types: spec, plan, tasks)", s)
	}
}

// ValidArtifactTypes returns a list of valid artifact type strings.
func ValidArtifactTypes() []string {
	return []string{"spec", "plan", "tasks"}
}
