// Package cli_test tests the prep command which runs the planning stages (specify, plan, tasks) without implementation.
// Related: internal/cli/prep.go
// Tags: cli, prep, planning, specify, plan, tasks, workflow
package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepCmdRegistration(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "prep <feature-description>" {
			found = true
			break
		}
	}
	assert.True(t, found, "prep command should be registered")
}

func TestPrepCmdRequiresExactlyOneArg(t *testing.T) {
	// Should require exactly 1 arg
	err := prepCmd.Args(prepCmd, []string{})
	assert.Error(t, err)

	err = prepCmd.Args(prepCmd, []string{"feature description"})
	assert.NoError(t, err)

	err = prepCmd.Args(prepCmd, []string{"arg1", "arg2"})
	assert.Error(t, err)
}

func TestPrepCmdFlags(t *testing.T) {
	// max-retries flag should exist
	f := prepCmd.Flags().Lookup("max-retries")
	require.NotNil(t, f)
	assert.Equal(t, "r", f.Shorthand)
	assert.Equal(t, "0", f.DefValue)

	specFlag := prepCmd.Flags().Lookup("spec")
	require.NotNil(t, specFlag)
	assert.Contains(t, specFlag.Usage, "overrides branch detection")
}

func TestPrepCmdExamples(t *testing.T) {
	examples := []string{
		"autospec prep",
		"Add user authentication",
		"Refactor database",
	}

	for _, example := range examples {
		assert.Contains(t, prepCmd.Example, example)
	}
}

func TestPrepCmdLongDescription(t *testing.T) {
	keywords := []string{
		"pre-flight",
		"specify",
		"plan",
		"tasks",
		"validate",
		"retry",
	}

	for _, keyword := range keywords {
		assert.Contains(t, prepCmd.Long, keyword)
	}
}

func TestPrepCmd_ExcludesImplement(t *testing.T) {
	// The prep command description should NOT mention implement in the steps
	// because prep excludes the implement phase (but it can mention "implementation" in context)
	assert.NotContains(t, prepCmd.Long, "Execute /autospec.implement")
	assert.Contains(t, prepCmd.Short, "specify")
	assert.Contains(t, prepCmd.Short, "plan")
	assert.Contains(t, prepCmd.Short, "tasks")
}

func TestPrepCmd_InheritedFlags(t *testing.T) {
	// Should inherit skip-preflight from root
	f := rootCmd.PersistentFlags().Lookup("skip-preflight")
	require.NotNil(t, f)

	// Should inherit config from root
	f = rootCmd.PersistentFlags().Lookup("config")
	require.NotNil(t, f)
}

func TestPrepCmd_MaxRetriesDefault(t *testing.T) {
	// Default should be 0 (use config)
	f := prepCmd.Flags().Lookup("max-retries")
	require.NotNil(t, f)
	assert.Equal(t, "0", f.DefValue)
}

func TestResolvePrepFeatureCompatibility(t *testing.T) {
	tests := map[string]struct {
		branch       string
		specs        []string
		persisted    string
		explicit     string
		wantSpec     string
		wantSource   spec.SelectionSource
		wantErr      string
		setupGitRepo bool
	}{
		"branch prefix fallback without persisted state": {
			branch:       "128-branch-feature",
			specs:        []string{"128-branch-feature"},
			wantSpec:     "128-branch-feature",
			wantSource:   spec.SelectionSourceGitBranch,
			setupGitRepo: true,
		},
		"explicit spec overrides persisted active feature": {
			specs:      []string{"128-persisted-feature", "129-explicit-feature"},
			persisted:  "128-persisted-feature",
			explicit:   "129-explicit-feature",
			wantSpec:   "129-explicit-feature",
			wantSource: spec.SelectionSourceExplicit,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newActiveFeatureCLIFixture(t)
			for _, specName := range tt.specs {
				fixture.createSpec(t, specName)
			}
			if tt.persisted != "" {
				fixture.persistActiveFeature(t, tt.persisted)
			}
			if tt.setupGitRepo {
				initPrepGitRepo(t, fixture.Root, tt.branch)
				fixture.chdir(t)
			}

			cfg := &config.Configuration{
				SpecsDir: fixture.SpecsDir,
				StateDir: fixture.StateDir,
			}
			got, err := resolvePrepFeature(cfg, tt.explicit)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantSource, got.Source)
			require.Equal(t, filepath.Join(fixture.SpecsDir, tt.wantSpec), got.Metadata.Directory)
		})
	}
}

func initPrepGitRepo(t *testing.T, dir, branch string) {
	t.Helper()

	runPrepGit(t, dir, "init")
	runPrepGit(t, dir, "config", "user.email", "test@example.com")
	runPrepGit(t, dir, "config", "user.name", "Test User")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("test\n"), 0o644))
	runPrepGit(t, dir, "add", "README.md")
	runPrepGit(t, dir, "commit", "-m", "initial")
	runPrepGit(t, dir, "checkout", "-b", branch)
}

func runPrepGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}
