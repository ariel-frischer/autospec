package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/stretchr/testify/require"
)

type activeFeatureCLIFixture struct {
	Root     string
	SpecsDir string
	StateDir string
	Config   string
}

func newActiveFeatureCLIFixture(t *testing.T) activeFeatureCLIFixture {
	t.Helper()

	root := t.TempDir()
	specsDir := filepath.Join(root, "specs")
	stateDir := filepath.Join(root, ".autospec", "state")
	configPath := filepath.Join(root, "config.yml")
	config := fmt.Sprintf("specs_dir: %s\nstate_dir: %s\n", specsDir, stateDir)
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o644))
	return activeFeatureCLIFixture{
		Root:     root,
		SpecsDir: specsDir,
		StateDir: stateDir,
		Config:   configPath,
	}
}

func (f activeFeatureCLIFixture) createSpec(t *testing.T, name string) string {
	t.Helper()

	dir := filepath.Join(f.SpecsDir, name)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "spec.yaml"), []byte("feature:\n  branch: "+name+"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plan.yaml"), []byte("plan:\n  branch: "+name+"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tasks.yaml"), []byte("tasks: []\n"), 0o644))
	return dir
}

func (f activeFeatureCLIFixture) persistActiveFeature(t *testing.T, name string) {
	t.Helper()

	_, err := spec.SaveActiveFeatureState(f.StateDir, name, "test")
	require.NoError(t, err)
}

func (f activeFeatureCLIFixture) chdir(t *testing.T) {
	t.Helper()

	previous, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(f.Root))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(previous))
	})
}
