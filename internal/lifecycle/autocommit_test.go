package lifecycle

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/git"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOpener implements git.Opener for testing
type mockOpener struct {
	repo     git.Repository
	openErr  error
	openPath string
}

func (m *mockOpener) Open(path string) (git.Repository, error) {
	m.openPath = path
	if m.openErr != nil {
		return nil, m.openErr
	}
	return m.repo, nil
}

// mockRepository implements git.Repository for testing
type mockRepository struct {
	headRef *plumbing.Reference
	headErr error
}

func (m *mockRepository) Head() (*plumbing.Reference, error) {
	if m.headErr != nil {
		return nil, m.headErr
	}
	return m.headRef, nil
}

// createMockBranchRef creates a mock branch reference for testing
func createMockBranchRef(branchName, commitSHA string) *plumbing.Reference {
	hash := plumbing.NewHash(commitSHA)
	name := plumbing.NewBranchReferenceName(branchName)
	return plumbing.NewHashReference(name, hash)
}

func TestNewAutoCommitHandler(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		repoPath string
		enabled  bool
	}{
		"enabled handler": {
			repoPath: "/test/repo",
			enabled:  true,
		},
		"disabled handler": {
			repoPath: "/another/repo",
			enabled:  false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			handler := NewAutoCommitHandler(tc.repoPath, tc.enabled)

			assert.Equal(t, tc.enabled, handler.Enabled)
			assert.Equal(t, tc.repoPath, handler.RepoPath)
			assert.NotNil(t, handler.Opener, "opener should be set to default")
			assert.Nil(t, handler.InitialState, "initial state should be nil before capture")
		})
	}
}

func TestAutoCommitHandler_CaptureInitialState(t *testing.T) {
	// Not parallel: subtests modify global os.Stderr

	tests := map[string]struct {
		enabled     bool
		openErr     error
		headRef     *plumbing.Reference
		wantCapture bool
	}{
		"disabled - no capture": {
			enabled:     false,
			wantCapture: false,
		},
		"enabled - successful capture": {
			enabled:     true,
			headRef:     createMockBranchRef("main", "abc123def456789012345678901234567890abcd"),
			wantCapture: true,
		},
		"enabled - open error": {
			enabled:     true,
			openErr:     errors.New("not a git repository"),
			wantCapture: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Not parallel: modifies global os.Stderr

			handler := &AutoCommitHandler{
				Enabled:  tc.enabled,
				RepoPath: "/test/repo",
				Opener: &mockOpener{
					repo:    &mockRepository{headRef: tc.headRef},
					openErr: tc.openErr,
				},
			}

			// Capture stderr to check for warnings
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			handler.CaptureInitialState()

			w.Close()
			os.Stderr = oldStderr
			var stderr bytes.Buffer
			io.Copy(&stderr, r)

			if tc.wantCapture {
				assert.NotNil(t, handler.InitialState)
				assert.NotEmpty(t, handler.InitialState.CommitSHA)
				assert.NotEmpty(t, handler.InitialState.BranchName)
			} else {
				if tc.enabled && tc.openErr != nil {
					// Should have logged a warning
					assert.Contains(t, stderr.String(), "Warning")
				}
			}
		})
	}
}

func TestAutoCommitHandler_CompareAndLogWarnings(t *testing.T) {
	// Not parallel: subtests modify global os.Stderr

	initialCommit := "abc123def456789012345678901234567890abcd"
	newCommit := "def456789012345678901234567890abcd123456"

	tests := map[string]struct {
		enabled        bool
		initialState   *git.GitState
		finalHeadRef   *plumbing.Reference
		wantWarning    string
		wantNoWarnings bool
	}{
		"disabled - no warnings": {
			enabled:        false,
			wantNoWarnings: true,
		},
		"no initial state - no warnings": {
			enabled:        true,
			initialState:   nil,
			wantNoWarnings: true,
		},
		"same state - no commit warning": {
			enabled: true,
			initialState: &git.GitState{
				CommitSHA:  initialCommit,
				BranchName: "main",
				CapturedAt: time.Now(),
			},
			finalHeadRef: createMockBranchRef("main", initialCommit),
			wantWarning:  "no new commit was created",
		},
		"branch changed - serious warning": {
			enabled: true,
			initialState: &git.GitState{
				CommitSHA:  initialCommit,
				BranchName: "main",
				CapturedAt: time.Now(),
			},
			finalHeadRef: createMockBranchRef("feature", newCommit),
			wantWarning:  "branch changed",
		},
		"new commit created - no warnings": {
			enabled: true,
			initialState: &git.GitState{
				CommitSHA:  initialCommit,
				BranchName: "main",
				CapturedAt: time.Now(),
			},
			finalHeadRef:   createMockBranchRef("main", newCommit),
			wantNoWarnings: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Not parallel: modifies global os.Stderr

			handler := &AutoCommitHandler{
				Enabled:      tc.enabled,
				RepoPath:     "/test/repo",
				InitialState: tc.initialState,
				Opener: &mockOpener{
					repo: &mockRepository{headRef: tc.finalHeadRef},
				},
			}

			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			handler.CompareAndLogWarnings()

			w.Close()
			os.Stderr = oldStderr
			var stderr bytes.Buffer
			io.Copy(&stderr, r)
			stderrStr := stderr.String()

			if tc.wantNoWarnings {
				assert.Empty(t, stderrStr, "should not log any warnings")
			} else if tc.wantWarning != "" {
				assert.Contains(t, stderrStr, tc.wantWarning,
					"should log warning containing: %s", tc.wantWarning)
			}
		})
	}
}

func TestRunWithAutoCommit(t *testing.T) {
	// Not parallel: subtests modify global os.Stderr

	tests := map[string]struct {
		enabled     bool
		fnErr       error
		wantErr     bool
		wantErrMsg  string
		description string
	}{
		"disabled - function succeeds": {
			enabled:     false,
			fnErr:       nil,
			wantErr:     false,
			description: "should return nil when disabled and fn succeeds",
		},
		"disabled - function fails": {
			enabled:     false,
			fnErr:       errors.New("workflow failed"),
			wantErr:     true,
			wantErrMsg:  "workflow failed",
			description: "should return fn error when disabled",
		},
		"enabled - function succeeds": {
			enabled:     true,
			fnErr:       nil,
			wantErr:     false,
			description: "should return nil when enabled and fn succeeds",
		},
		"enabled - function fails": {
			enabled:     true,
			fnErr:       errors.New("execution error"),
			wantErr:     true,
			wantErrMsg:  "execution error",
			description: "should return fn error when enabled",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Not parallel: modifies global os.Stderr

			fnCalled := false
			fn := func() error {
				fnCalled = true
				return tc.fnErr
			}

			// Use temp directory as repo path (may not be a git repo, but that's ok)
			repoPath := t.TempDir()

			// Capture stderr (warnings from non-git directory are expected)
			oldStderr := os.Stderr
			_, w, _ := os.Pipe()
			os.Stderr = w

			err := RunWithAutoCommit(tc.enabled, repoPath, fn)

			w.Close()
			os.Stderr = oldStderr

			assert.True(t, fnCalled, "function should be called")

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRunWithAutoCommitHandler(t *testing.T) {
	// Parallel OK: subtests don't modify global os.Stderr
	t.Parallel()

	tests := map[string]struct {
		handler     *AutoCommitHandler
		fnErr       error
		wantErr     bool
		description string
	}{
		"nil handler - function succeeds": {
			handler:     nil,
			fnErr:       nil,
			wantErr:     false,
			description: "should work with nil handler",
		},
		"nil handler - function fails": {
			handler:     nil,
			fnErr:       errors.New("fn error"),
			wantErr:     true,
			description: "should return fn error with nil handler",
		},
		"disabled handler": {
			handler: &AutoCommitHandler{
				Enabled:  false,
				RepoPath: "/test",
				Opener:   &mockOpener{},
			},
			fnErr:       nil,
			wantErr:     false,
			description: "should work with disabled handler",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fnCalled := false
			fn := func() error {
				fnCalled = true
				return tc.fnErr
			}

			err := RunWithAutoCommitHandler(tc.handler, fn)

			assert.True(t, fnCalled, "function should be called")

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLogStateWarning(t *testing.T) {
	// Not parallel: subtests modify global os.Stderr

	tests := map[string]struct {
		warning     git.StateWarning
		wantPrefix  string
		wantMessage string
	}{
		"serious warning": {
			warning: git.StateWarning{
				Level:   "serious",
				Message: "branch changed during workflow",
			},
			wantPrefix:  "WARNING:",
			wantMessage: "branch changed",
		},
		"regular warning": {
			warning: git.StateWarning{
				Level:   "warning",
				Message: "no new commit was created",
			},
			wantPrefix:  "Note:",
			wantMessage: "no new commit",
		},
		"unknown level": {
			warning: git.StateWarning{
				Level:   "info",
				Message: "some info message",
			},
			wantPrefix:  "info:",
			wantMessage: "some info message",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Not parallel: modifies global os.Stderr

			// Capture stderr
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			logStateWarning(tc.warning)

			w.Close()
			os.Stderr = oldStderr
			var stderr bytes.Buffer
			io.Copy(&stderr, r)
			output := stderr.String()

			assert.True(t, strings.HasPrefix(output, tc.wantPrefix),
				"output should start with %s, got: %s", tc.wantPrefix, output)
			assert.Contains(t, output, tc.wantMessage)
		})
	}
}

// TestAutoCommitHandler_FR012_NoWarningsWhenDisabled verifies FR-012:
// No auto-commit warnings appear when auto_commit=false
func TestAutoCommitHandler_FR012_NoWarningsWhenDisabled(t *testing.T) {
	// Not parallel: modifies global os.Stderr

	handler := &AutoCommitHandler{
		Enabled:  false,
		RepoPath: "/test/repo",
		InitialState: &git.GitState{
			CommitSHA:  "abc123",
			BranchName: "main",
			CapturedAt: time.Now(),
		},
		Opener: &mockOpener{
			repo: &mockRepository{
				headRef: createMockBranchRef("main", "abc123"),
			},
		},
	}

	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Both operations should be no-ops when disabled
	handler.CaptureInitialState()
	handler.CompareAndLogWarnings()

	w.Close()
	os.Stderr = oldStderr
	var stderr bytes.Buffer
	io.Copy(&stderr, r)

	assert.Empty(t, stderr.String(),
		"no warnings should be logged when auto-commit is disabled (FR-012)")
}

// TestRunWithAutoCommit_FR013_SuccessOnWarnings verifies FR-013:
// Workflow returns exit 0 even if comparison shows issues
func TestRunWithAutoCommit_FR013_SuccessOnWarnings(t *testing.T) {
	// Not parallel: modifies global os.Stderr

	// Even if git state comparison would show warnings,
	// the function should return the original fn result
	fnCalled := false
	fn := func() error {
		fnCalled = true
		return nil // Function succeeds
	}

	// Use a path that's not a git repo to trigger capture failure
	// This tests that warnings don't cause function failure
	repoPath := t.TempDir()

	// Capture stderr
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	err := RunWithAutoCommit(true, repoPath, fn)

	w.Close()
	os.Stderr = oldStderr

	assert.True(t, fnCalled)
	assert.NoError(t, err, "function should return nil even with git warnings (FR-013)")
}
