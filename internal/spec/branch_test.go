package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateBranchName(t *testing.T) {
	tests := map[string]struct {
		description string
		expected    string
	}{
		"simple feature": {
			description: "Add user authentication",
			expected:    "user-authentication",
		},
		"filters stop words": {
			description: "I want to add a feature for the users",
			expected:    "feature-users",
		},
		"keeps first 3 words": {
			description: "Implement OAuth2 integration for API access",
			expected:    "implement-oauth2-integration",
		},
		"keeps 4 words when exactly 4": {
			description: "Implement OAuth2 API access",
			expected:    "implement-oauth2-api-access",
		},
		"handles uppercase": {
			description: "Add API Support",
			expected:    "api-support",
		},
		"keeps two-letter words in meaningful context": {
			description: "Add CI CD pipeline",
			expected:    "ci-cd-pipeline",
		},
		"removes special characters": {
			description: "Add user-auth feature (v2)",
			expected:    "user-auth-feature",
		},
		"handles numbers": {
			description: "Version 2 upgrade",
			expected:    "version-upgrade",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := GenerateBranchName(tt.description)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanBranchName(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"lowercase conversion": {
			input:    "MyFeature",
			expected: "myfeature",
		},
		"replaces spaces": {
			input:    "my feature",
			expected: "my-feature",
		},
		"replaces special chars": {
			input:    "my_feature@v2",
			expected: "my-feature-v2",
		},
		"removes consecutive hyphens": {
			input:    "my--feature",
			expected: "my-feature",
		},
		"removes leading hyphen": {
			input:    "-my-feature",
			expected: "my-feature",
		},
		"removes trailing hyphen": {
			input:    "my-feature-",
			expected: "my-feature",
		},
		"handles mixed special chars": {
			input:    "  My Feature (v2.0)  ",
			expected: "my-feature-v2-0",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := CleanBranchName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateBranchName(t *testing.T) {
	tests := map[string]struct {
		branchName       string
		expectTruncation bool
	}{
		"short branch name unchanged": {
			branchName:       "001-my-feature",
			expectTruncation: false,
		},
		"exactly at limit unchanged": {
			branchName:       "001-" + strings.Repeat("a", 240),
			expectTruncation: false,
		},
		"over limit truncated": {
			branchName:       "001-" + strings.Repeat("a", 250),
			expectTruncation: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := TruncateBranchName(tt.branchName)
			assert.LessOrEqual(t, len(result), MaxBranchLength)

			if tt.expectTruncation {
				assert.Less(t, len(result), len(tt.branchName))
			} else {
				assert.Equal(t, tt.branchName, result)
			}
		})
	}
}

func TestGetNextBranchNumber(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "autospec-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	specsDir := filepath.Join(tmpDir, "specs")
	err = os.MkdirAll(specsDir, 0755)
	require.NoError(t, err)

	// Note: These tests run in the real git repo, so they will also pick up
	// existing branches. The tests verify relative behavior.

	t.Run("returns valid number format", func(t *testing.T) {
		num, err := GetNextBranchNumber(specsDir)
		require.NoError(t, err)
		// Should be a 3-digit zero-padded number
		assert.Len(t, num, 3)
		assert.Regexp(t, `^\d{3}$`, num)
	})

	t.Run("with existing specs increases number", func(t *testing.T) {
		// Get baseline
		baseNum, err := GetNextBranchNumber(specsDir)
		require.NoError(t, err)

		// Create spec directories with higher numbers
		err = os.MkdirAll(filepath.Join(specsDir, "100-first-feature"), 0755)
		require.NoError(t, err)
		err = os.MkdirAll(filepath.Join(specsDir, "101-second-feature"), 0755)
		require.NoError(t, err)

		num, err := GetNextBranchNumber(specsDir)
		require.NoError(t, err)

		// Should be at least 102 (or higher if git branches exist)
		numInt := 0
		fmt.Sscanf(num, "%d", &numInt)
		baseInt := 0
		fmt.Sscanf(baseNum, "%d", &baseInt)

		// Result should be >= 102 since we added 100 and 101
		assert.GreaterOrEqual(t, numInt, 102)
	})

	t.Run("non-existent directory returns valid number", func(t *testing.T) {
		num, err := GetNextBranchNumber("/nonexistent/path")
		require.NoError(t, err)
		// Should still return a valid format (from git branches if available)
		assert.Regexp(t, `^\d{3}$`, num)
	})
}

func TestFormatBranchName(t *testing.T) {
	tests := map[string]struct {
		number   string
		suffix   string
		expected string
	}{
		"001-my-feature":      {number: "001", suffix: "my-feature", expected: "001-my-feature"},
		"042-another-feature": {number: "042", suffix: "another-feature", expected: "042-another-feature"},
		"123-test":            {number: "123", suffix: "test", expected: "123-test"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := FormatBranchName(tt.number, tt.suffix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFeatureDirectory(t *testing.T) {
	result := GetFeatureDirectory("/home/user/project/specs", "001-my-feature")
	assert.Equal(t, "/home/user/project/specs/001-my-feature", result)
}

func TestStopWords(t *testing.T) {
	// Verify key stop words are in the map
	expectedStopWords := []string{"the", "a", "to", "for", "is", "are", "add", "get", "set"}
	for _, word := range expectedStopWords {
		assert.True(t, StopWords[word], "expected '%s' to be a stop word", word)
	}

	// Verify some non-stop words are not in the map
	nonStopWords := []string{"feature", "user", "api", "implement"}
	for _, word := range nonStopWords {
		assert.False(t, StopWords[word], "expected '%s' to NOT be a stop word", word)
	}
}
