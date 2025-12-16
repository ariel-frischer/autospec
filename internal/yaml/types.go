// Package yaml provides YAML validation and parsing for autospec artifacts.
package yaml

// Meta represents the _meta section common to all YAML artifacts.
type Meta struct {
	Version          string `yaml:"version"`
	Generator        string `yaml:"generator"`
	GeneratorVersion string `yaml:"generator_version"`
	Created          string `yaml:"created"`
	ArtifactType     string `yaml:"artifact_type"`
}

// Feature represents the feature metadata in a spec artifact.
type Feature struct {
	Branch      string `yaml:"branch"`
	Created     string `yaml:"created"`
	Status      string `yaml:"status"`
	CompletedAt string `yaml:"completed_at,omitempty"`
	Input       string `yaml:"input,omitempty"`
}

// UserStory represents a user story in a spec artifact.
type UserStory struct {
	ID                  string               `yaml:"id"`
	Title               string               `yaml:"title"`
	Priority            string               `yaml:"priority"`
	AsA                 string               `yaml:"as_a"`
	IWant               string               `yaml:"i_want"`
	SoThat              string               `yaml:"so_that"`
	WhyPriority         string               `yaml:"why_priority,omitempty"`
	TestDescription     string               `yaml:"test_description,omitempty"`
	AcceptanceScenarios []AcceptanceScenario `yaml:"acceptance_scenarios"`
}

// AcceptanceScenario represents a Given/When/Then scenario.
type AcceptanceScenario struct {
	Given string `yaml:"given"`
	When  string `yaml:"when"`
	Then  string `yaml:"then"`
}

// EdgeCase represents a question/answer edge case pair.
type EdgeCase struct {
	Question string `yaml:"question"`
	Answer   string `yaml:"answer"`
}

// Requirement represents a functional or non-functional requirement.
type Requirement struct {
	ID          string `yaml:"id"`
	Description string `yaml:"description"`
}

// Requirements groups functional and non-functional requirements.
type Requirements struct {
	Functional    []Requirement `yaml:"functional"`
	NonFunctional []Requirement `yaml:"non_functional,omitempty"`
}

// Entity represents a key entity in the domain.
type Entity struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Criterion represents a success criterion.
type Criterion struct {
	ID          string `yaml:"id"`
	Description string `yaml:"description"`
}

// TextItem represents a simple text item for assumptions, constraints, etc.
type TextItem struct {
	Text string `yaml:"text"`
}

// SpecArtifact represents the complete spec.yaml structure.
type SpecArtifact struct {
	Meta            Meta         `yaml:"_meta"`
	Feature         Feature      `yaml:"feature"`
	UserStories     []UserStory  `yaml:"user_stories"`
	EdgeCases       []EdgeCase   `yaml:"edge_cases,omitempty"`
	Requirements    Requirements `yaml:"requirements"`
	KeyEntities     []Entity     `yaml:"key_entities,omitempty"`
	SuccessCriteria []Criterion  `yaml:"success_criteria,omitempty"`
	Assumptions     []TextItem   `yaml:"assumptions,omitempty"`
	Constraints     []TextItem   `yaml:"constraints,omitempty"`
	OutOfScope      []TextItem   `yaml:"out_of_scope,omitempty"`
}

// PlanInfo represents the plan header information.
type PlanInfo struct {
	Branch   string `yaml:"branch"`
	Date     string `yaml:"date"`
	SpecPath string `yaml:"spec_path"`
}

// TechContext represents the technical context of a plan.
type TechContext struct {
	Language            string   `yaml:"language,omitempty"`
	PrimaryDependencies []string `yaml:"primary_dependencies,omitempty"`
	Storage             string   `yaml:"storage,omitempty"`
	Testing             string   `yaml:"testing,omitempty"`
	TargetPlatform      string   `yaml:"target_platform,omitempty"`
	ProjectType         string   `yaml:"project_type,omitempty"`
	PerformanceGoals    string   `yaml:"performance_goals,omitempty"`
	Constraints         string   `yaml:"constraints,omitempty"`
	ScaleScope          string   `yaml:"scale_scope,omitempty"`
}

// Gate represents a constitution gate check result.
type Gate struct {
	Name   string `yaml:"name"`
	Status string `yaml:"status"`
	Notes  string `yaml:"notes,omitempty"`
}

// ConstitutionCheck represents the constitution check section.
type ConstitutionCheck struct {
	Gates []Gate `yaml:"gates,omitempty"`
}

// PathDescription represents a path with description.
type PathDescription struct {
	Path        string `yaml:"path"`
	Description string `yaml:"description"`
}

// ProjectStructure represents the project structure documentation.
type ProjectStructure struct {
	Documentation []PathDescription `yaml:"documentation,omitempty"`
	SourceCode    []PathDescription `yaml:"source_code,omitempty"`
}

// ComplexityItem represents a complexity tracking entry.
type ComplexityItem struct {
	Violation           string `yaml:"violation"`
	WhyNeeded           string `yaml:"why_needed"`
	AlternativeRejected string `yaml:"alternative_rejected"`
}

// PlanArtifact represents the complete plan.yaml structure.
type PlanArtifact struct {
	Meta               Meta              `yaml:"_meta"`
	Plan               PlanInfo          `yaml:"plan"`
	Summary            string            `yaml:"summary"`
	TechnicalContext   TechContext       `yaml:"technical_context"`
	ConstitutionCheck  ConstitutionCheck `yaml:"constitution_check,omitempty"`
	ProjectStructure   ProjectStructure  `yaml:"project_structure,omitempty"`
	ComplexityTracking []ComplexityItem  `yaml:"complexity_tracking,omitempty"`
}

// TasksInfo represents the tasks header information.
type TasksInfo struct {
	Branch   string `yaml:"branch"`
	SpecPath string `yaml:"spec_path"`
	PlanPath string `yaml:"plan_path"`
}

// Task represents a single task.
type Task struct {
	ID                  string   `yaml:"id"`
	Title               string   `yaml:"title"`
	Status              string   `yaml:"status"`
	Type                string   `yaml:"type"`
	Dependencies        []string `yaml:"dependencies,omitempty"`
	AcceptanceCriteria  []string `yaml:"acceptance_criteria"`
	ImplementationNotes string   `yaml:"implementation_notes,omitempty"`
}

// Phase represents a task phase.
type Phase struct {
	Number      int    `yaml:"number"`
	Title       string `yaml:"title"`
	Description string `yaml:"description,omitempty"`
	Tasks       []Task `yaml:"tasks"`
}

// TasksArtifact represents the complete tasks.yaml structure.
type TasksArtifact struct {
	Meta   Meta      `yaml:"_meta"`
	Tasks  TasksInfo `yaml:"tasks"`
	Phases []Phase   `yaml:"phases"`
}

// ChecklistInfo represents the checklist header information.
type ChecklistInfo struct {
	Feature  string `yaml:"feature"`
	SpecPath string `yaml:"spec_path"`
}

// ChecklistItem represents a single checklist item.
type ChecklistItem struct {
	ID          string `yaml:"id"`
	Description string `yaml:"description"`
	Checked     bool   `yaml:"checked"`
	Notes       string `yaml:"notes,omitempty"`
}

// Category represents a checklist category.
type Category struct {
	Name  string          `yaml:"name"`
	Items []ChecklistItem `yaml:"items"`
}

// ChecklistArtifact represents the complete checklist.yaml structure.
type ChecklistArtifact struct {
	Meta       Meta          `yaml:"_meta"`
	Checklist  ChecklistInfo `yaml:"checklist"`
	Categories []Category    `yaml:"categories"`
}

// AnalysisInfo represents the analysis header information.
type AnalysisInfo struct {
	SpecPath  string `yaml:"spec_path"`
	PlanPath  string `yaml:"plan_path"`
	TasksPath string `yaml:"tasks_path"`
	Timestamp string `yaml:"timestamp"`
}

// Finding represents an analysis finding.
type Finding struct {
	Category       string `yaml:"category"`
	Severity       string `yaml:"severity"`
	Description    string `yaml:"description"`
	Location       string `yaml:"location"`
	Recommendation string `yaml:"recommendation,omitempty"`
}

// AnalysisSummary represents the analysis summary.
type AnalysisSummary struct {
	TotalIssues int `yaml:"total_issues"`
	Errors      int `yaml:"errors"`
	Warnings    int `yaml:"warnings"`
	Info        int `yaml:"info"`
}

// AnalysisArtifact represents the complete analysis.yaml structure.
type AnalysisArtifact struct {
	Meta     Meta            `yaml:"_meta"`
	Analysis AnalysisInfo    `yaml:"analysis"`
	Findings []Finding       `yaml:"findings"`
	Summary  AnalysisSummary `yaml:"summary"`
}

// ConstitutionInfo represents the constitution header information.
type ConstitutionInfo struct {
	ProjectName string `yaml:"project_name"`
	Version     string `yaml:"version"`
	Ratified    string `yaml:"ratified"`
	LastAmended string `yaml:"last_amended,omitempty"`
}

// Principle represents a constitution principle.
type Principle struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Enforcement string `yaml:"enforcement,omitempty"`
}

// Section represents a constitution section.
type Section struct {
	Name    string `yaml:"name"`
	Content string `yaml:"content"`
}

// GovernanceRules represents governance rules.
type GovernanceRules struct {
	Rules []string `yaml:"rules,omitempty"`
}

// ConstitutionArtifact represents the complete constitution.yaml structure.
type ConstitutionArtifact struct {
	Meta         Meta             `yaml:"_meta"`
	Constitution ConstitutionInfo `yaml:"constitution"`
	Principles   []Principle      `yaml:"principles"`
	Sections     []Section        `yaml:"sections,omitempty"`
	Governance   GovernanceRules  `yaml:"governance,omitempty"`
}
