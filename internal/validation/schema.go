package validation

import (
	"fmt"
	"path/filepath"
)

// ArtifactType represents the type of artifact to validate.
type ArtifactType string

const (
	// ArtifactTypeSpec represents spec.yaml artifacts.
	ArtifactTypeSpec ArtifactType = "spec"
	// ArtifactTypePlan represents plan.yaml artifacts.
	ArtifactTypePlan ArtifactType = "plan"
	// ArtifactTypeTasks represents tasks.yaml artifacts.
	ArtifactTypeTasks ArtifactType = "tasks"
	// ArtifactTypeAnalysis represents analysis.yaml artifacts.
	ArtifactTypeAnalysis ArtifactType = "analysis"
	// ArtifactTypeChecklist represents checklist.yaml artifacts.
	ArtifactTypeChecklist ArtifactType = "checklist"
	// ArtifactTypeConstitution represents constitution.yaml artifacts.
	ArtifactTypeConstitution ArtifactType = "constitution"
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
	Name        string        // Field name in YAML
	Type        FieldType     // Expected type
	Required    bool          // Whether field must be present
	Pattern     string        // Regex pattern for string validation (optional)
	Enum        []string      // Valid values for enum fields (optional)
	Description string        // Human-readable description
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
				{Name: "status", Type: FieldTypeString, Required: false, Enum: []string{"Draft", "Review", "Approved", "Completed"}, Description: "Feature status"},
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
			Children: []SchemaField{
				{Name: "id", Type: FieldTypeString, Required: false, Pattern: `^RISK-\d{3}$`, Description: "Unique risk identifier (optional for backward compatibility)"},
				{Name: "risk", Type: FieldTypeString, Required: true, Description: "Brief description of the risk"},
				{Name: "likelihood", Type: FieldTypeString, Required: true, Enum: []string{"low", "medium", "high"}, Description: "Probability of occurrence"},
				{Name: "impact", Type: FieldTypeString, Required: true, Enum: []string{"low", "medium", "high"}, Description: "Severity if risk materializes"},
				{Name: "mitigation", Type: FieldTypeString, Required: false, Description: "Strategy to address the risk"},
			},
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

// AnalysisSchema defines the schema for analysis.yaml artifacts.
var AnalysisSchema = Schema{
	Type:        ArtifactTypeAnalysis,
	Description: "Cross-artifact analysis results containing findings, coverage, and recommendations",
	Fields: []SchemaField{
		{
			Name:        "analysis",
			Type:        FieldTypeObject,
			Required:    true,
			Description: "Analysis metadata including branch and file references",
			Children: []SchemaField{
				{Name: "branch", Type: FieldTypeString, Required: true, Description: "Git branch name"},
				{Name: "timestamp", Type: FieldTypeString, Required: true, Description: "Analysis timestamp (ISO 8601)"},
				{Name: "spec_path", Type: FieldTypeString, Required: false, Description: "Path to spec file"},
				{Name: "plan_path", Type: FieldTypeString, Required: false, Description: "Path to plan file"},
				{Name: "tasks_path", Type: FieldTypeString, Required: false, Description: "Path to tasks file"},
				{Name: "constitution_path", Type: FieldTypeString, Required: false, Description: "Path to constitution file"},
			},
		},
		{
			Name:        "findings",
			Type:        FieldTypeArray,
			Required:    true,
			Description: "List of analysis findings",
			Children: []SchemaField{
				{Name: "id", Type: FieldTypeString, Required: true, Description: "Finding ID (e.g., DUP-001, AMB-001)"},
				{Name: "category", Type: FieldTypeString, Required: true, Enum: []string{"duplication", "ambiguity", "coverage", "constitution", "inconsistency", "underspecification"}, Description: "Finding category"},
				{Name: "severity", Type: FieldTypeString, Required: true, Enum: []string{"CRITICAL", "HIGH", "MEDIUM", "LOW"}, Description: "Finding severity"},
				{Name: "location", Type: FieldTypeString, Required: true, Description: "Location of the finding"},
				{Name: "summary", Type: FieldTypeString, Required: true, Description: "Brief summary of the finding"},
				{Name: "details", Type: FieldTypeString, Required: false, Description: "Detailed explanation"},
				{Name: "recommendation", Type: FieldTypeString, Required: false, Description: "Suggested fix"},
			},
		},
		{
			Name:        "coverage",
			Type:        FieldTypeObject,
			Required:    false,
			Description: "Coverage analysis for requirements and stories",
			Children: []SchemaField{
				{Name: "requirements", Type: FieldTypeArray, Required: false, Description: "Requirement coverage details"},
				{Name: "user_stories", Type: FieldTypeArray, Required: false, Description: "User story coverage details"},
			},
		},
		{
			Name:        "constitution_alignment",
			Type:        FieldTypeObject,
			Required:    false,
			Description: "Constitution compliance check results",
			Children: []SchemaField{
				{Name: "status", Type: FieldTypeString, Required: false, Enum: []string{"PASS", "FAIL"}, Description: "Overall alignment status"},
				{Name: "violations", Type: FieldTypeArray, Required: false, Description: "List of constitution violations"},
			},
		},
		{
			Name:        "unmapped_tasks",
			Type:        FieldTypeArray,
			Required:    false,
			Description: "Tasks without corresponding requirements",
		},
		{
			Name:        "metrics",
			Type:        FieldTypeObject,
			Required:    false,
			Description: "Analysis metrics and counts",
			Children: []SchemaField{
				{Name: "total_requirements", Type: FieldTypeInt, Required: false, Description: "Total number of requirements"},
				{Name: "total_tasks", Type: FieldTypeInt, Required: false, Description: "Total number of tasks"},
				{Name: "coverage_percentage", Type: FieldTypeInt, Required: false, Description: "Percentage of requirements with tasks"},
				{Name: "critical_issues", Type: FieldTypeInt, Required: false, Description: "Number of critical issues"},
				{Name: "high_issues", Type: FieldTypeInt, Required: false, Description: "Number of high issues"},
				{Name: "medium_issues", Type: FieldTypeInt, Required: false, Description: "Number of medium issues"},
				{Name: "low_issues", Type: FieldTypeInt, Required: false, Description: "Number of low issues"},
			},
		},
		{
			Name:        "summary",
			Type:        FieldTypeObject,
			Required:    true,
			Description: "Analysis summary",
			Children: []SchemaField{
				{Name: "overall_status", Type: FieldTypeString, Required: true, Enum: []string{"PASS", "WARN", "FAIL"}, Description: "Overall analysis status"},
				{Name: "blocking_issues", Type: FieldTypeInt, Required: false, Description: "Number of blocking issues"},
				{Name: "actionable_improvements", Type: FieldTypeInt, Required: false, Description: "Number of actionable improvements"},
				{Name: "ready_for_implementation", Type: FieldTypeBool, Required: false, Description: "Whether ready for implementation"},
			},
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
				{Name: "artifact_type", Type: FieldTypeString, Required: false, Enum: []string{"analysis"}, Description: "Artifact type"},
			},
		},
	},
}

// ChecklistSchema defines the schema for checklist.yaml artifacts.
var ChecklistSchema = Schema{
	Type:        ArtifactTypeChecklist,
	Description: "Feature quality validation checklist for requirements completeness and clarity",
	Fields: []SchemaField{
		{
			Name:        "checklist",
			Type:        FieldTypeObject,
			Required:    true,
			Description: "Checklist metadata",
			Children: []SchemaField{
				{Name: "feature", Type: FieldTypeString, Required: true, Description: "Feature name"},
				{Name: "branch", Type: FieldTypeString, Required: true, Description: "Git branch name"},
				{Name: "spec_path", Type: FieldTypeString, Required: false, Description: "Path to spec file"},
				{Name: "domain", Type: FieldTypeString, Required: true, Description: "Checklist domain (ux, api, security, performance, etc.)"},
				{Name: "audience", Type: FieldTypeString, Required: false, Enum: []string{"author", "reviewer", "qa", "release"}, Description: "Target audience"},
				{Name: "depth", Type: FieldTypeString, Required: false, Enum: []string{"lightweight", "standard", "comprehensive"}, Description: "Checklist depth"},
			},
		},
		{
			Name:        "categories",
			Type:        FieldTypeArray,
			Required:    true,
			Description: "Checklist categories with items",
			Children: []SchemaField{
				{Name: "name", Type: FieldTypeString, Required: true, Description: "Category name"},
				{Name: "description", Type: FieldTypeString, Required: false, Description: "Category description"},
				{Name: "items", Type: FieldTypeArray, Required: true, Description: "Checklist items in this category"},
			},
		},
		{
			Name:        "summary",
			Type:        FieldTypeObject,
			Required:    false,
			Description: "Checklist summary statistics",
			Children: []SchemaField{
				{Name: "total_items", Type: FieldTypeInt, Required: false, Description: "Total number of items"},
				{Name: "passed", Type: FieldTypeInt, Required: false, Description: "Number of passed items"},
				{Name: "failed", Type: FieldTypeInt, Required: false, Description: "Number of failed items"},
				{Name: "pending", Type: FieldTypeInt, Required: false, Description: "Number of pending items"},
				{Name: "pass_rate", Type: FieldTypeString, Required: false, Description: "Pass rate percentage"},
			},
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
				{Name: "artifact_type", Type: FieldTypeString, Required: false, Enum: []string{"checklist"}, Description: "Artifact type"},
			},
		},
	},
}

// ChecklistItemSchema defines the schema for individual checklist items.
var ChecklistItemSchema = []SchemaField{
	{Name: "id", Type: FieldTypeString, Required: true, Description: "Checklist item ID (CHKnnn format)"},
	{Name: "description", Type: FieldTypeString, Required: true, Description: "Item description (question format)"},
	{Name: "quality_dimension", Type: FieldTypeString, Required: false, Enum: []string{"completeness", "clarity", "consistency", "measurability", "coverage", "edge_cases"}, Description: "Quality dimension being checked"},
	{Name: "spec_reference", Type: FieldTypeString, Required: false, Description: "Reference to spec requirement"},
	{Name: "status", Type: FieldTypeString, Required: true, Enum: []string{"pending", "pass", "fail"}, Description: "Item status"},
	{Name: "notes", Type: FieldTypeString, Required: false, Description: "Additional notes"},
}

// ConstitutionSchema defines the schema for constitution.yaml artifacts.
var ConstitutionSchema = Schema{
	Type:        ArtifactTypeConstitution,
	Description: "Project constitution defining principles, governance, and standards",
	Fields: []SchemaField{
		{
			Name:        "constitution",
			Type:        FieldTypeObject,
			Required:    true,
			Description: "Constitution metadata",
			Children: []SchemaField{
				{Name: "project_name", Type: FieldTypeString, Required: true, Description: "Project name"},
				{Name: "version", Type: FieldTypeString, Required: true, Description: "Constitution version"},
				{Name: "ratified", Type: FieldTypeString, Required: false, Description: "Ratification date"},
				{Name: "last_amended", Type: FieldTypeString, Required: false, Description: "Last amendment date"},
			},
		},
		{
			Name:        "preamble",
			Type:        FieldTypeString,
			Required:    false,
			Description: "Constitution preamble describing purpose",
		},
		{
			Name:        "principles",
			Type:        FieldTypeArray,
			Required:    true,
			Description: "List of project principles",
			Children: []SchemaField{
				{Name: "name", Type: FieldTypeString, Required: true, Description: "Principle name"},
				{Name: "id", Type: FieldTypeString, Required: true, Description: "Principle ID (PRIN-nnn format)"},
				{Name: "category", Type: FieldTypeString, Required: false, Enum: []string{"quality", "architecture", "process", "security"}, Description: "Principle category"},
				{Name: "priority", Type: FieldTypeString, Required: true, Enum: []string{"NON-NEGOTIABLE", "MUST", "SHOULD", "MAY"}, Description: "Principle priority"},
				{Name: "description", Type: FieldTypeString, Required: true, Description: "Principle description"},
				{Name: "rationale", Type: FieldTypeString, Required: false, Description: "Rationale for the principle"},
				{Name: "enforcement", Type: FieldTypeArray, Required: false, Description: "Enforcement mechanisms"},
				{Name: "exceptions", Type: FieldTypeArray, Required: false, Description: "Allowed exceptions"},
			},
		},
		{
			Name:        "sections",
			Type:        FieldTypeArray,
			Required:    false,
			Description: "Additional constitution sections",
			Children: []SchemaField{
				{Name: "name", Type: FieldTypeString, Required: true, Description: "Section name"},
				{Name: "content", Type: FieldTypeString, Required: true, Description: "Section content"},
			},
		},
		{
			Name:        "governance",
			Type:        FieldTypeObject,
			Required:    false,
			Description: "Governance rules and processes",
			Children: []SchemaField{
				{Name: "amendment_process", Type: FieldTypeArray, Required: false, Description: "Amendment process steps"},
				{Name: "versioning_policy", Type: FieldTypeString, Required: false, Description: "Version policy description"},
				{Name: "compliance_review", Type: FieldTypeObject, Required: false, Description: "Compliance review settings"},
				{Name: "rules", Type: FieldTypeArray, Required: false, Description: "Governance rules"},
			},
		},
		{
			Name:        "sync_impact",
			Type:        FieldTypeObject,
			Required:    false,
			Description: "Impact of recent changes",
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
				{Name: "artifact_type", Type: FieldTypeString, Required: false, Enum: []string{"constitution"}, Description: "Artifact type"},
			},
		},
	},
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
	case ArtifactTypeAnalysis:
		return &AnalysisSchema, nil
	case ArtifactTypeChecklist:
		return &ChecklistSchema, nil
	case ArtifactTypeConstitution:
		return &ConstitutionSchema, nil
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
	case "analysis":
		return ArtifactTypeAnalysis, nil
	case "checklist":
		return ArtifactTypeChecklist, nil
	case "constitution":
		return ArtifactTypeConstitution, nil
	default:
		return "", fmt.Errorf("invalid artifact type: %s (valid types: spec, plan, tasks, analysis, checklist, constitution)", s)
	}
}

// ValidArtifactTypes returns a list of valid artifact type strings.
func ValidArtifactTypes() []string {
	return []string{"spec", "plan", "tasks", "analysis", "checklist", "constitution"}
}

// artifactFilenames maps canonical filenames to artifact types.
var artifactFilenames = map[string]ArtifactType{
	"spec.yaml":         ArtifactTypeSpec,
	"spec.yml":          ArtifactTypeSpec,
	"plan.yaml":         ArtifactTypePlan,
	"plan.yml":          ArtifactTypePlan,
	"tasks.yaml":        ArtifactTypeTasks,
	"tasks.yml":         ArtifactTypeTasks,
	"analysis.yaml":     ArtifactTypeAnalysis,
	"analysis.yml":      ArtifactTypeAnalysis,
	"constitution.yaml": ArtifactTypeConstitution,
	"constitution.yml":  ArtifactTypeConstitution,
}

// InferArtifactTypeFromFilename infers the artifact type from a filename.
// It accepts both .yaml and .yml extensions.
// Returns the artifact type if recognized, or an error for unrecognized filenames.
func InferArtifactTypeFromFilename(filename string) (ArtifactType, error) {
	baseName := filepath.Base(filename)

	if artType, ok := artifactFilenames[baseName]; ok {
		return artType, nil
	}

	return "", fmt.Errorf("unrecognized artifact filename: %s", baseName)
}

// ValidArtifactFilenames returns a list of recognized artifact filenames.
func ValidArtifactFilenames() []string {
	return []string{"spec.yaml", "plan.yaml", "tasks.yaml", "analysis.yaml", "constitution.yaml"}
}
