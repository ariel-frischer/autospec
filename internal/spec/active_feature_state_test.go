package spec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestActiveFeatureStatePersistence(t *testing.T) {
	tests := map[string]struct {
		setup     func(t *testing.T, specsDir, stateDir string)
		wantFound bool
		wantErr   string
		validate  bool
	}{
		"save and load state": {
			setup: func(t *testing.T, specsDir, stateDir string) {
				createActiveFeatureSpec(t, specsDir, "001-saved")
				_, err := SaveActiveFeatureState(stateDir, "001-saved", "test")
				require.NoError(t, err)
			},
			wantFound: true,
		},
		"missing state is not found": {
			setup:     func(t *testing.T, specsDir, stateDir string) {},
			wantFound: false,
		},
		"deleted persisted directory is invalid": {
			setup: func(t *testing.T, specsDir, stateDir string) {
				createActiveFeatureSpec(t, specsDir, "001-deleted")
				_, err := SaveActiveFeatureState(stateDir, "001-deleted", "test")
				require.NoError(t, err)
				require.NoError(t, os.RemoveAll(filepath.Join(specsDir, "001-deleted")))
			},
			wantFound: true,
			validate:  true,
			wantErr:   "persisted active feature directory does not exist",
		},
		"missing spec yaml is invalid": {
			setup: func(t *testing.T, specsDir, stateDir string) {
				createActiveFeatureSpec(t, specsDir, "001-missing-spec")
				require.NoError(t, os.Remove(filepath.Join(specsDir, "001-missing-spec", "spec.yaml")))
				_, err := SaveActiveFeatureState(stateDir, "001-missing-spec", "test")
				require.NoError(t, err)
			},
			wantFound: true,
			validate:  true,
			wantErr:   "missing spec.yaml",
		},
		"path traversal is rejected": {
			setup: func(t *testing.T, specsDir, stateDir string) {
				require.NoError(t, os.MkdirAll(stateDir, 0o755))
				writeActiveFeatureState(t, stateDir, "../outside", "test")
			},
			wantFound: true,
			validate:  true,
			wantErr:   "escapes specs directory",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			specsDir := filepath.Join(root, "specs")
			stateDir := filepath.Join(root, ".autospec", "state")
			tt.setup(t, specsDir, stateDir)

			state, found, err := LoadActiveFeatureState(stateDir)
			require.NoError(t, err)
			require.Equal(t, tt.wantFound, found)
			if !found {
				return
			}
			require.NotEmpty(t, state.FeatureDirectory)
			require.NotEmpty(t, state.Source)
			require.NotEmpty(t, state.UpdatedAt)

			if tt.validate {
				_, err = ValidateActiveFeatureState(specsDir, state)
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestActiveFeatureStateYAMLFields(t *testing.T) {
	root := t.TempDir()
	stateDir := filepath.Join(root, ".autospec", "state")

	state, err := SaveActiveFeatureState(stateDir, "001-feature", "specify")
	require.NoError(t, err)
	require.Equal(t, "001-feature", state.FeatureDirectory)

	data, err := os.ReadFile(filepath.Join(stateDir, "active-feature.yaml"))
	require.NoError(t, err)

	var fields map[string]string
	require.NoError(t, yaml.Unmarshal(data, &fields))
	require.NotEmpty(t, fields["feature_directory"])
	require.NotEmpty(t, fields["source"])
	require.NotEmpty(t, fields["updated_at"])
}

func TestClearActiveFeatureState(t *testing.T) {
	stateDir := filepath.Join(t.TempDir(), ".autospec", "state")
	_, err := SaveActiveFeatureState(stateDir, "001-feature", "test")
	require.NoError(t, err)

	require.NoError(t, ClearActiveFeatureState(stateDir))
	_, found, err := LoadActiveFeatureState(stateDir)
	require.NoError(t, err)
	require.False(t, found)
}

func writeActiveFeatureState(t *testing.T, stateDir, featureDirectory, source string) {
	t.Helper()

	data := []byte("feature_directory: " + featureDirectory + "\nsource: " + source + "\nupdated_at: 2026-05-14T00:00:00Z\n")
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "active-feature.yaml"), data, 0o644))
}
