package validation

import (
	"testing"
)

func TestGetSchema_Spec(t *testing.T) {
	schema, err := GetSchema(ArtifactTypeSpec)
	if err != nil {
		t.Fatalf("GetSchema(spec) returned error: %v", err)
	}
	if schema == nil {
		t.Fatal("GetSchema(spec) returned nil schema")
	}
	if schema.Type != ArtifactTypeSpec {
		t.Errorf("schema.Type = %q, want %q", schema.Type, ArtifactTypeSpec)
	}

	// Verify required top-level fields
	requiredFields := []string{"feature", "user_stories", "requirements"}
	for _, fieldName := range requiredFields {
		found := false
		for _, f := range schema.Fields {
			if f.Name == fieldName && f.Required {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("spec schema missing required field: %s", fieldName)
		}
	}
}

func TestGetSchema_Plan(t *testing.T) {
	schema, err := GetSchema(ArtifactTypePlan)
	if err != nil {
		t.Fatalf("GetSchema(plan) returned error: %v", err)
	}
	if schema == nil {
		t.Fatal("GetSchema(plan) returned nil schema")
	}
	if schema.Type != ArtifactTypePlan {
		t.Errorf("schema.Type = %q, want %q", schema.Type, ArtifactTypePlan)
	}

	// Verify required top-level fields
	requiredFields := []string{"plan", "summary", "technical_context"}
	for _, fieldName := range requiredFields {
		found := false
		for _, f := range schema.Fields {
			if f.Name == fieldName && f.Required {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("plan schema missing required field: %s", fieldName)
		}
	}
}

func TestGetSchema_Tasks(t *testing.T) {
	schema, err := GetSchema(ArtifactTypeTasks)
	if err != nil {
		t.Fatalf("GetSchema(tasks) returned error: %v", err)
	}
	if schema == nil {
		t.Fatal("GetSchema(tasks) returned nil schema")
	}
	if schema.Type != ArtifactTypeTasks {
		t.Errorf("schema.Type = %q, want %q", schema.Type, ArtifactTypeTasks)
	}

	// Verify required top-level fields
	requiredFields := []string{"tasks", "summary", "phases"}
	for _, fieldName := range requiredFields {
		found := false
		for _, f := range schema.Fields {
			if f.Name == fieldName && f.Required {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tasks schema missing required field: %s", fieldName)
		}
	}
}

func TestGetSchema_UnknownType(t *testing.T) {
	_, err := GetSchema(ArtifactType("unknown"))
	if err == nil {
		t.Error("GetSchema(unknown) should return error")
	}
}

func TestParseArtifactType(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected ArtifactType
		wantErr  bool
	}{
		"spec":                {input: "spec", expected: ArtifactTypeSpec, wantErr: false},
		"plan":                {input: "plan", expected: ArtifactTypePlan, wantErr: false},
		"tasks":               {input: "tasks", expected: ArtifactTypeTasks, wantErr: false},
		"unknown":             {input: "unknown", expected: "", wantErr: true},
		"SPEC case-sensitive": {input: "SPEC", expected: "", wantErr: true},
		"empty string":        {input: "", expected: "", wantErr: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ParseArtifactType(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseArtifactType(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("ParseArtifactType(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestValidArtifactTypes(t *testing.T) {
	types := ValidArtifactTypes()
	if len(types) != 6 {
		t.Errorf("ValidArtifactTypes() returned %d types, want 6", len(types))
	}

	expected := map[string]bool{
		"spec":         true,
		"plan":         true,
		"tasks":        true,
		"analysis":     true,
		"checklist":    true,
		"constitution": true,
	}
	for _, typ := range types {
		if !expected[typ] {
			t.Errorf("unexpected artifact type: %s", typ)
		}
	}
}

func TestSpecSchema_UserStoryEnums(t *testing.T) {
	schema, _ := GetSchema(ArtifactTypeSpec)

	// Find user_stories field
	var userStoriesField *SchemaField
	for i, f := range schema.Fields {
		if f.Name == "user_stories" {
			userStoriesField = &schema.Fields[i]
			break
		}
	}

	if userStoriesField == nil {
		t.Fatal("user_stories field not found in spec schema")
	}

	// Find priority field in children
	var priorityField *SchemaField
	for i, f := range userStoriesField.Children {
		if f.Name == "priority" {
			priorityField = &userStoriesField.Children[i]
			break
		}
	}

	if priorityField == nil {
		t.Fatal("priority field not found in user_stories schema")
	}

	// Verify enum values
	if len(priorityField.Enum) == 0 {
		t.Error("priority field should have enum values")
	}

	expectedEnums := []string{"P0", "P1", "P2", "P3"}
	for _, e := range expectedEnums {
		found := false
		for _, v := range priorityField.Enum {
			if v == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("priority enum missing value: %s", e)
		}
	}
}

func TestTasksSchema_StatusEnums(t *testing.T) {
	// Verify task status enum values
	var statusField *SchemaField
	for i, f := range TaskFieldSchema {
		if f.Name == "status" {
			statusField = &TaskFieldSchema[i]
			break
		}
	}

	if statusField == nil {
		t.Fatal("status field not found in task schema")
	}

	expectedEnums := []string{"Pending", "InProgress", "Completed", "Blocked"}
	if len(statusField.Enum) != len(expectedEnums) {
		t.Errorf("status enum has %d values, want %d", len(statusField.Enum), len(expectedEnums))
	}

	for _, e := range expectedEnums {
		found := false
		for _, v := range statusField.Enum {
			if v == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("status enum missing value: %s", e)
		}
	}
}

func TestTasksSchema_TypeEnums(t *testing.T) {
	// Verify task type enum values
	var typeField *SchemaField
	for i, f := range TaskFieldSchema {
		if f.Name == "type" {
			typeField = &TaskFieldSchema[i]
			break
		}
	}

	if typeField == nil {
		t.Fatal("type field not found in task schema")
	}

	expectedEnums := []string{"setup", "implementation", "test", "documentation", "refactor"}
	if len(typeField.Enum) != len(expectedEnums) {
		t.Errorf("type enum has %d values, want %d", len(typeField.Enum), len(expectedEnums))
	}

	for _, e := range expectedEnums {
		found := false
		for _, v := range typeField.Enum {
			if v == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("type enum missing value: %s", e)
		}
	}
}

func TestSchemaField_HasDescription(t *testing.T) {
	schemas := []*Schema{&SpecSchema, &PlanSchema, &TasksSchema}

	for _, schema := range schemas {
		if schema.Description == "" {
			t.Errorf("%s schema missing description", schema.Type)
		}

		for _, field := range schema.Fields {
			if field.Description == "" {
				t.Errorf("%s.%s field missing description", schema.Type, field.Name)
			}
		}
	}
}

func TestInferArtifactTypeFromFilename(t *testing.T) {
	tests := map[string]struct {
		filename string
		want     ArtifactType
		wantErr  bool
	}{
		// Valid .yaml filenames
		"spec.yaml":  {filename: "spec.yaml", want: ArtifactTypeSpec, wantErr: false},
		"plan.yaml":  {filename: "plan.yaml", want: ArtifactTypePlan, wantErr: false},
		"tasks.yaml": {filename: "tasks.yaml", want: ArtifactTypeTasks, wantErr: false},

		// Valid .yml filenames
		"spec.yml":  {filename: "spec.yml", want: ArtifactTypeSpec, wantErr: false},
		"plan.yml":  {filename: "plan.yml", want: ArtifactTypePlan, wantErr: false},
		"tasks.yml": {filename: "tasks.yml", want: ArtifactTypeTasks, wantErr: false},

		// Full paths with .yaml
		"path with spec.yaml":  {filename: "specs/016-feature/spec.yaml", want: ArtifactTypeSpec, wantErr: false},
		"path with plan.yaml":  {filename: "specs/016-feature/plan.yaml", want: ArtifactTypePlan, wantErr: false},
		"path with tasks.yaml": {filename: "specs/016-feature/tasks.yaml", want: ArtifactTypeTasks, wantErr: false},

		// Full paths with .yml
		"path with spec.yml": {filename: "/absolute/path/spec.yml", want: ArtifactTypeSpec, wantErr: false},
		"path with plan.yml": {filename: "relative/plan.yml", want: ArtifactTypePlan, wantErr: false},

		// Unrecognized filenames
		"config.yaml":              {filename: "config.yaml", want: "", wantErr: true},
		"random.yaml":              {filename: "random.yaml", want: "", wantErr: true},
		"myspec.yaml":              {filename: "myspec.yaml", want: "", wantErr: true},
		"spec.json":                {filename: "spec.json", want: "", wantErr: true},
		"SPEC.yaml case-sensitive": {filename: "SPEC.yaml", want: "", wantErr: true},
		"Plan.yaml case-sensitive": {filename: "Plan.yaml", want: "", wantErr: true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := InferArtifactTypeFromFilename(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("InferArtifactTypeFromFilename(%q) error = %v, wantErr %v", tt.filename, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("InferArtifactTypeFromFilename(%q) = %q, want %q", tt.filename, got, tt.want)
			}
		})
	}
}

func TestValidArtifactFilenames(t *testing.T) {
	filenames := ValidArtifactFilenames()
	if len(filenames) != 5 {
		t.Errorf("ValidArtifactFilenames() returned %d filenames, want 5", len(filenames))
	}

	expected := map[string]bool{
		"spec.yaml":         true,
		"plan.yaml":         true,
		"tasks.yaml":        true,
		"analysis.yaml":     true,
		"constitution.yaml": true,
	}
	for _, filename := range filenames {
		if !expected[filename] {
			t.Errorf("unexpected artifact filename: %s", filename)
		}
	}
}
