package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePlanData(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		yamlContent string
		wantData    *PlanData
		wantErr     bool
		errContains string
	}{
		{
			name: "complete plan.yaml with all fields",
			yamlContent: `plan:
  branch: "017-update-agent-context-go"
  created: "2025-12-16"
  spec_path: "specs/017-update-agent-context-go/spec.yaml"

technical_context:
  language: "Go 1.25.1"
  storage: "PostgreSQL"
  project_type: "cli"
  primary_dependencies:
    - name: "github.com/spf13/cobra"
      version: "v1.10.1"
    - name: "gopkg.in/yaml.v3"
      version: "v3.0.1"
`,
			wantData: &PlanData{
				Language:    "Go 1.25.1",
				Framework:   "github.com/spf13/cobra v1.10.1",
				Database:    "PostgreSQL",
				ProjectType: "cli",
				Branch:      "017-update-agent-context-go",
			},
			wantErr: false,
		},
		{
			name: "plan.yaml with no storage",
			yamlContent: `plan:
  branch: "002-feature"

technical_context:
  language: "Python 3.11"
  storage: "None"
  project_type: "web"
  primary_dependencies:
    - name: "fastapi"
      version: "0.100.0"
`,
			wantData: &PlanData{
				Language:    "Python 3.11",
				Framework:   "fastapi 0.100.0",
				Database:    "None",
				ProjectType: "web",
				Branch:      "002-feature",
			},
			wantErr: false,
		},
		{
			name: "plan.yaml with empty primary_dependencies",
			yamlContent: `plan:
  branch: "003-simple"

technical_context:
  language: "JavaScript"
  storage: ""
  project_type: "library"
  primary_dependencies: []
`,
			wantData: &PlanData{
				Language:    "JavaScript",
				Framework:   "",
				Database:    "",
				ProjectType: "library",
				Branch:      "003-simple",
			},
			wantErr: false,
		},
		{
			name: "plan.yaml with dependency without version",
			yamlContent: `plan:
  branch: "004-noversion"

technical_context:
  language: "Rust"
  storage: "SQLite"
  project_type: "service"
  primary_dependencies:
    - name: "tokio"
`,
			wantData: &PlanData{
				Language:    "Rust",
				Framework:   "tokio",
				Database:    "SQLite",
				ProjectType: "service",
				Branch:      "004-noversion",
			},
			wantErr: false,
		},
		{
			name: "plan.yaml with missing technical_context fields",
			yamlContent: `plan:
  branch: "005-minimal"

technical_context:
  language: "Go"
`,
			wantData: &PlanData{
				Language:    "Go",
				Framework:   "",
				Database:    "",
				ProjectType: "",
				Branch:      "005-minimal",
			},
			wantErr: false,
		},
		{
			name:        "invalid YAML syntax",
			yamlContent: `plan:\n  branch: [invalid`,
			wantErr:     true,
			errContains: "failed to parse plan.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			planPath := filepath.Join(tmpDir, tt.name+"-plan.yaml")
			if err := os.WriteFile(planPath, []byte(tt.yamlContent), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			got, err := ParsePlanData(planPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePlanData() expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ParsePlanData() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ParsePlanData() unexpected error: %v", err)
				return
			}

			if got.Language != tt.wantData.Language {
				t.Errorf("Language = %q, want %q", got.Language, tt.wantData.Language)
			}
			if got.Framework != tt.wantData.Framework {
				t.Errorf("Framework = %q, want %q", got.Framework, tt.wantData.Framework)
			}
			if got.Database != tt.wantData.Database {
				t.Errorf("Database = %q, want %q", got.Database, tt.wantData.Database)
			}
			if got.ProjectType != tt.wantData.ProjectType {
				t.Errorf("ProjectType = %q, want %q", got.ProjectType, tt.wantData.ProjectType)
			}
			if got.Branch != tt.wantData.Branch {
				t.Errorf("Branch = %q, want %q", got.Branch, tt.wantData.Branch)
			}
		})
	}
}

func TestParsePlanData_FileNotFound(t *testing.T) {
	_, err := ParsePlanData("/nonexistent/path/plan.yaml")
	if err == nil {
		t.Error("ParsePlanData() expected error for non-existent file, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("ParsePlanData() error = %v, want error containing 'not found'", err)
	}
	if !strings.Contains(err.Error(), "autospec plan") {
		t.Errorf("ParsePlanData() error should suggest running 'autospec plan', got: %v", err)
	}
}

func TestPlanData_GetTechnologies(t *testing.T) {
	tests := []struct {
		name     string
		planData PlanData
		want     []string
	}{
		{
			name: "all fields populated",
			planData: PlanData{
				Language:    "Go 1.25.1",
				Framework:   "Cobra CLI v1.10.1",
				Database:    "PostgreSQL",
				ProjectType: "cli",
			},
			want: []string{"Go 1.25.1", "Cobra CLI v1.10.1", "PostgreSQL", "Project Type: cli"},
		},
		{
			name: "storage is None",
			planData: PlanData{
				Language:    "Python 3.11",
				Framework:   "FastAPI",
				Database:    "None",
				ProjectType: "web",
			},
			want: []string{"Python 3.11", "FastAPI", "Project Type: web"},
		},
		{
			name: "empty framework",
			planData: PlanData{
				Language:    "JavaScript",
				Framework:   "",
				Database:    "MongoDB",
				ProjectType: "service",
			},
			want: []string{"JavaScript", "MongoDB", "Project Type: service"},
		},
		{
			name: "all fields empty",
			planData: PlanData{
				Language:    "",
				Framework:   "",
				Database:    "",
				ProjectType: "",
			},
			want: []string{},
		},
		{
			name: "only language",
			planData: PlanData{
				Language: "Rust",
			},
			want: []string{"Rust"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.planData.GetTechnologies()

			if len(got) != len(tt.want) {
				t.Errorf("GetTechnologies() returned %d items, want %d", len(got), len(tt.want))
				t.Errorf("got: %v", got)
				t.Errorf("want: %v", tt.want)
				return
			}

			for i, tech := range got {
				if tech != tt.want[i] {
					t.Errorf("GetTechnologies()[%d] = %q, want %q", i, tech, tt.want[i])
				}
			}
		})
	}
}

func TestPlanData_GetChangeEntry(t *testing.T) {
	tests := []struct {
		name     string
		planData PlanData
		want     string
	}{
		{
			name:     "with branch",
			planData: PlanData{Branch: "017-update-agent-context"},
			want:     "017-update-agent-context: Added from plan.yaml",
		},
		{
			name:     "empty branch",
			planData: PlanData{Branch: ""},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.planData.GetChangeEntry()
			if got != tt.want {
				t.Errorf("GetChangeEntry() = %q, want %q", got, tt.want)
			}
		})
	}
}
