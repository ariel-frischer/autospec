package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/commands"
	"github.com/ariel-frischer/autospec/internal/prereqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetOptionsForCommand(t *testing.T) {
	tests := map[string]struct {
		commandName      string
		wantRequireSpec  bool
		wantRequirePlan  bool
		wantRequireTasks bool
		wantPathsOnly    bool
	}{
		"plan command requires spec": {
			commandName:     "autospec.plan",
			wantRequireSpec: true,
		},
		"tasks command requires plan": {
			commandName:     "autospec.tasks",
			wantRequireSpec: true,
			wantRequirePlan: true,
		},
		"implement command requires tasks": {
			commandName:      "autospec.implement",
			wantRequireTasks: true,
		},
		"constitution command has minimal requirements": {
			commandName:   "autospec.constitution",
			wantPathsOnly: false,
		},
		"specify command uses paths only": {
			commandName:   "autospec.specify",
			wantPathsOnly: true,
		},
		"unknown command uses paths only": {
			commandName:   "unknown.command",
			wantPathsOnly: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			opts := getOptionsForCommand(tt.commandName, "./specs")

			assert.Equal(t, tt.wantRequireSpec, opts.RequireSpec)
			assert.Equal(t, tt.wantRequirePlan, opts.RequirePlan)
			assert.Equal(t, tt.wantRequireTasks, opts.RequireTasks)
			assert.Equal(t, tt.wantPathsOnly, opts.PathsOnly)
			assert.Equal(t, "./specs", opts.SpecsDir)
		})
	}
}

func TestRenderCommandIntegration(t *testing.T) {
	// Create a temporary feature directory structure
	tmpDir, err := os.MkdirTemp("", "render-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	specsDir := filepath.Join(tmpDir, "specs")
	featureDir := filepath.Join(specsDir, "001-test-feature")
	require.NoError(t, os.MkdirAll(featureDir, 0o755))

	// Create required files
	specContent := "spec: test\n"
	planContent := "plan: test\n"
	tasksContent := "tasks: test\n"

	require.NoError(t, os.WriteFile(filepath.Join(featureDir, "spec.yaml"), []byte(specContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(featureDir, "plan.yaml"), []byte(planContent), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(featureDir, "tasks.yaml"), []byte(tasksContent), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(tmpDir, ".autospec"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".autospec", "constitution.yaml"),
		[]byte("constitution:\n  name: Test Governance\n"), 0o644))
	t.Chdir(tmpDir)

	// Set environment variable for feature detection
	oldEnv := os.Getenv("SPECIFY_FEATURE")
	os.Setenv("SPECIFY_FEATURE", "001-test-feature")
	defer os.Setenv("SPECIFY_FEATURE", oldEnv)

	tests := map[string]struct {
		commandName    string
		wantContains   []string
		wantNotContain []string
	}{
		"plan command renders feature paths": {
			commandName: "autospec.plan",
			wantContains: []string{
				"{{.FeatureDir}}",  // Should NOT be in output (should be rendered)
				"001-test-feature", // Should be in output (rendered value)
			},
		},
		"constitution renders version info": {
			commandName: "autospec.constitution",
			wantContains: []string{
				"autospec", // Version string
			},
		},
		"implement command renders governance context": {
			commandName: "autospec.implement",
			wantContains: []string{
				"CONSTITUTION_FILE",
				".autospec/constitution.yaml",
				"REQUIRED GOVERNANCE",
				"governance:",
				"skip_reads",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Get template content
			content, err := commands.GetTemplate(tt.commandName)
			require.NoError(t, err)

			// Compute context
			opts := getOptionsForCommand(tt.commandName, specsDir)
			ctx, err := prereqs.ComputeContext(opts)
			require.NoError(t, err)

			// Render template
			rendered, err := commands.RenderAndValidate(tt.commandName, content, ctx)
			require.NoError(t, err)

			renderedStr := string(rendered)

			// Check that template placeholders are NOT in output (they should be rendered)
			assert.NotContains(t, renderedStr, "{{.FeatureDir}}")
			assert.NotContains(t, renderedStr, "{{.FeatureSpec}}")
			assert.NotContains(t, renderedStr, "{{.ImplPlan}}")
			assert.NotContains(t, renderedStr, "{{.TasksFile}}")
			assert.NotContains(t, renderedStr, "{{.ConstitutionFile}}")
			assert.NotContains(t, renderedStr, "{{.AutospecVersion}}")
			assert.NotContains(t, renderedStr, "{{.CreatedDate}}")

			// Check expected content
			for _, want := range tt.wantContains {
				if !strings.HasPrefix(want, "{{") {
					assert.Contains(t, renderedStr, want, "should contain: %s", want)
				}
			}

			for _, notWant := range tt.wantNotContain {
				assert.NotContains(t, renderedStr, notWant, "should NOT contain: %s", notWant)
			}
		})
	}
}

func TestAllCommandsRenderWithoutError(t *testing.T) {
	// Create a temporary feature directory with all required files
	tmpDir, err := os.MkdirTemp("", "render-all-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	specsDir := filepath.Join(tmpDir, "specs")
	featureDir := filepath.Join(specsDir, "001-test-feature")
	require.NoError(t, os.MkdirAll(featureDir, 0o755))

	// Create all required files
	require.NoError(t, os.WriteFile(filepath.Join(featureDir, "spec.yaml"), []byte("spec: test"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(featureDir, "plan.yaml"), []byte("plan: test"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(featureDir, "tasks.yaml"), []byte("tasks: test"), 0o644))

	// Set environment variable
	oldEnv := os.Getenv("SPECIFY_FEATURE")
	os.Setenv("SPECIFY_FEATURE", "001-test-feature")
	defer os.Setenv("SPECIFY_FEATURE", oldEnv)

	// Get all autospec command names
	commandNames := commands.GetAutospecCommandNames()
	require.NotEmpty(t, commandNames, "should have autospec commands")

	for _, cmdName := range commandNames {
		t.Run(cmdName, func(t *testing.T) {
			content, err := commands.GetTemplate(cmdName)
			require.NoError(t, err, "should load template")

			opts := getOptionsForCommand(cmdName, specsDir)
			ctx, err := prereqs.ComputeContext(opts)
			require.NoError(t, err, "should compute context")

			rendered, err := commands.RenderAndValidate(cmdName, content, ctx)
			require.NoError(t, err, "should render without error")
			assert.NotEmpty(t, rendered, "rendered output should not be empty")

			// Verify no unrendered template variables remain
			renderedStr := string(rendered)
			assert.NotContains(t, renderedStr, "{{.FeatureDir}}")
			assert.NotContains(t, renderedStr, "{{.FeatureSpec}}")
			assert.NotContains(t, renderedStr, "{{.ImplPlan}}")
			assert.NotContains(t, renderedStr, "{{.TasksFile}}")
		})
	}
}

func TestRenderPreservesMarkdownStructure(t *testing.T) {
	// Create a temporary feature directory
	tmpDir, err := os.MkdirTemp("", "markdown-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	specsDir := filepath.Join(tmpDir, "specs")
	featureDir := filepath.Join(specsDir, "001-test-feature")
	require.NoError(t, os.MkdirAll(featureDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(featureDir, "spec.yaml"), []byte("spec: test"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(featureDir, "plan.yaml"), []byte("plan: test"), 0o644))

	oldEnv := os.Getenv("SPECIFY_FEATURE")
	os.Setenv("SPECIFY_FEATURE", "001-test-feature")
	defer os.Setenv("SPECIFY_FEATURE", oldEnv)

	content, err := commands.GetTemplate("autospec.plan")
	require.NoError(t, err)

	opts := getOptionsForCommand("autospec.plan", specsDir)
	ctx, err := prereqs.ComputeContext(opts)
	require.NoError(t, err)

	rendered, err := commands.RenderAndValidate("autospec.plan", content, ctx)
	require.NoError(t, err)

	renderedStr := string(rendered)

	// Frontmatter is stripped (starts with "---") because CLI tools misinterpret
	// it as a flag delimiter. Content should start with "## " (first heading).
	assert.True(t, strings.HasPrefix(renderedStr, "## "), "should start with heading (frontmatter stripped)")
	assert.NotContains(t, renderedStr, "description: Generate", "should not contain frontmatter content")

	// Check markdown structure is preserved
	assert.Contains(t, renderedStr, "## ", "should have headings")
	assert.Contains(t, renderedStr, "```", "should have code blocks")
	assert.Contains(t, renderedStr, "- **", "should have bold list items")
}
