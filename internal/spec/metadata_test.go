package spec

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMetadataBackwardCompatibility(t *testing.T) {
	tests := map[string]struct {
		setup       func(t *testing.T) (string, string)
		resolve     func(specsDir, identifier string) (*Metadata, error)
		identifier  string
		wantDir     string
		wantSource  DetectionMethod
		wantNumber  string
		wantName    string
		wantErrText string
	}{
		"branch prefix lookup works without persisted state": {
			setup: func(t *testing.T) (string, string) {
				root := newGitFixture(t, "128-persist-feature-directory")
				specsDir := filepath.Join(root, "specs")
				createMetadataSpec(t, specsDir, "128-persist-feature-directory")
				return specsDir, "128-persist-feature-directory"
			},
			resolve: func(specsDir, _ string) (*Metadata, error) {
				return DetectCurrentSpec(specsDir)
			},
			wantDir:    "128-persist-feature-directory",
			wantSource: DetectionGitBranch,
			wantNumber: "128",
			wantName:   "persist-feature-directory",
		},
		"fallback remains deterministic with shared numeric prefix": {
			setup: func(t *testing.T) (string, string) {
				root := newGitFixture(t, "128-unmatched-branch")
				specsDir := filepath.Join(root, "specs")
				createMetadataSpec(t, specsDir, "128-first-feature")
				time.Sleep(10 * time.Millisecond)
				createMetadataSpec(t, specsDir, "128-second-feature")
				return specsDir, "128-second-feature"
			},
			resolve: func(specsDir, _ string) (*Metadata, error) {
				return DetectCurrentSpec(specsDir)
			},
			wantDir:    "128-second-feature",
			wantSource: DetectionFallbackRecent,
			wantNumber: "128",
			wantName:   "second-feature",
		},
		"explicit exact identifier still resolves through metadata helper": {
			setup: func(t *testing.T) (string, string) {
				specsDir := filepath.Join(t.TempDir(), "specs")
				createMetadataSpec(t, specsDir, "129-explicit-feature")
				return specsDir, "129-explicit-feature"
			},
			resolve: func(specsDir, identifier string) (*Metadata, error) {
				return GetSpecMetadata(specsDir, identifier)
			},
			identifier: "129-explicit-feature",
			wantDir:    "129-explicit-feature",
			wantNumber: "129",
			wantName:   "explicit-feature",
		},
		"explicit number identifier remains ambiguous with shared prefix": {
			setup: func(t *testing.T) (string, string) {
				specsDir := filepath.Join(t.TempDir(), "specs")
				createMetadataSpec(t, specsDir, "130-first-feature")
				createMetadataSpec(t, specsDir, "130-second-feature")
				return specsDir, ""
			},
			resolve: func(specsDir, identifier string) (*Metadata, error) {
				return GetSpecMetadata(specsDir, identifier)
			},
			identifier:  "130",
			wantErrText: "multiple specs found for number 130",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			specsDir, wantDir := tt.setup(t)

			got, err := tt.resolve(specsDir, tt.identifier)
			if tt.wantErrText != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErrText)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.wantNumber, got.Number)
			require.Equal(t, tt.wantName, got.Name)
			require.Equal(t, filepath.Join(specsDir, wantDir), got.Directory)
			if tt.wantSource != "" {
				require.Equal(t, tt.wantSource, got.Detection)
			}
		})
	}
}

func newGitFixture(t *testing.T, branch string) string {
	t.Helper()

	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	require.NoError(t, os.WriteFile(filepath.Join(root, "README.md"), []byte("test\n"), 0o644))
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-m", "initial")
	runGit(t, root, "checkout", "-b", branch)
	chdir(t, root)
	return root
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}

func chdir(t *testing.T, dir string) {
	t.Helper()

	previous, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(previous))
	})
}

func createMetadataSpec(t *testing.T, specsDir, name string) {
	t.Helper()

	dir := filepath.Join(specsDir, name)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "spec.yaml"), []byte("feature:\n  branch: "+name+"\n"), 0o644))
}
