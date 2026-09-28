//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestE2E_ArtifactSchemaWithoutSetup verifies `artifact <type> --schema` works
// in an uninitialized directory for every artifact type and creates nothing.
func TestE2E_ArtifactSchemaWithoutSetup(t *testing.T) {
	tests := map[string]struct {
		rootField string
	}{
		"spec":         {rootField: "feature: object (required)"},
		"plan":         {rootField: "plan: object (required)"},
		"tasks":        {rootField: "tasks: object (required)"},
		"analysis":     {rootField: "analysis: object (required)"},
		"checklist":    {rootField: "checklist: object (required)"},
		"constitution": {rootField: "constitution: object (required)"},
	}

	for artifactType, tt := range tests {
		t.Run(artifactType, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			result := env.Run("artifact", artifactType, "--schema")

			require.Equal(t, 0, result.ExitCode, "stdout=%s\nstderr=%s", result.Stdout, result.Stderr)
			require.Contains(t, result.Stdout, "Schema for "+artifactType+" artifacts")
			require.Contains(t, result.Stdout, tt.rootField)
			entries, err := os.ReadDir(env.SpecsDir())
			require.NoError(t, err)
			require.Empty(t, entries, "--schema must not create spec entries")
			require.NoDirExists(t, filepath.Join(env.TempDir(), ".autospec"))
		})
	}
}
