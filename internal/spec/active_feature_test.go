package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveActiveFeaturePrecedence(t *testing.T) {
	tests := map[string]struct {
		explicit       string
		persisted      string
		wantDirectory  string
		wantSource     SelectionSource
		wantErr        string
		omitFallback   bool
		omitPersisted  bool
		omitExplicit   bool
		requiredFile   string
		removeRequired bool
		omitAllSpecs   bool
	}{
		"explicit selection wins over persisted and fallback": {
			explicit:      "002-explicit",
			persisted:     "001-persisted",
			wantDirectory: "002-explicit",
			wantSource:    SelectionSourceExplicit,
		},
		"persisted selection wins over fallback": {
			persisted:     "001-persisted",
			omitExplicit:  true,
			wantDirectory: "001-persisted",
			wantSource:    SelectionSourcePersisted,
		},
		"fallback selects existing current spec": {
			omitExplicit:  true,
			omitPersisted: true,
			wantDirectory: "003-fallback",
			wantSource:    SelectionSourceFallback,
		},
		"no match returns contextual error": {
			omitExplicit: true,
			omitAllSpecs: true,
			wantErr:      "resolving active feature",
		},
		"required artifact is validated": {
			explicit:       "002-explicit",
			persisted:      "001-persisted",
			requiredFile:   "plan.yaml",
			removeRequired: true,
			wantErr:        "missing required artifact plan.yaml",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			specsDir, stateDir := setupActiveFeatureDirs(t, tt.omitFallback, tt.omitAllSpecs)
			if !tt.omitPersisted && tt.persisted != "" {
				_, err := SaveActiveFeatureState(stateDir, tt.persisted, "test")
				require.NoError(t, err)
			}
			if tt.removeRequired {
				require.NoError(t, os.Remove(filepath.Join(specsDir, tt.explicit, tt.requiredFile)))
			}

			req := ActiveFeatureRequest{
				SpecsDir:           specsDir,
				StateDir:           stateDir,
				ExplicitIdentifier: tt.explicit,
				RequiredArtifact:   tt.requiredFile,
			}
			if tt.omitExplicit {
				req.ExplicitIdentifier = ""
			}

			got, err := ResolveActiveFeature(req)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantSource, got.Source)
			require.Equal(t, filepath.Join(specsDir, tt.wantDirectory), got.Metadata.Directory)
		})
	}
}

func setupActiveFeatureDirs(t *testing.T, omitFallback, omitAllSpecs bool) (string, string) {
	t.Helper()

	root := t.TempDir()
	specsDir := filepath.Join(root, "specs")
	stateDir := filepath.Join(root, ".autospec", "state")
	if omitAllSpecs {
		return specsDir, stateDir
	}
	for _, dir := range []string{"001-persisted", "002-explicit"} {
		createActiveFeatureSpec(t, specsDir, dir)
	}
	if !omitFallback {
		createActiveFeatureSpec(t, specsDir, "003-fallback")
	}
	return specsDir, stateDir
}

func createActiveFeatureSpec(t *testing.T, specsDir, name string) {
	t.Helper()

	dir := filepath.Join(specsDir, name)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "spec.yaml"), []byte("feature:\n  branch: "+name+"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "plan.yaml"), []byte("plan:\n  branch: "+name+"\n"), 0o644))
}

func TestResolveActiveFeatureReportsSelectionSource(t *testing.T) {
	specsDir, stateDir := setupActiveFeatureDirs(t, false, false)

	got, err := ResolveActiveFeature(ActiveFeatureRequest{
		SpecsDir:           specsDir,
		StateDir:           stateDir,
		ExplicitIdentifier: "002-explicit",
	})

	require.NoError(t, err)
	require.NotEmpty(t, got.Metadata.Directory)
	require.True(t, strings.HasPrefix(string(got.Source), "explicit"))
}
