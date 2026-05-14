package changelog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromReader_ValidYAML(t *testing.T) {
	tests := map[string]struct {
		yaml     string
		expected *Changelog
	}{
		"minimal valid changelog": {
			yaml: `
project: myproject
versions:
  - version: "1.0.0"
    date: "2024-01-15"
    changes:
      added:
        - "Initial release"
`,
			expected: &Changelog{
				Project: "myproject",
				Versions: []Version{
					{
						Version: "1.0.0",
						Date:    "2024-01-15",
						Changes: Changes{Added: []string{"Initial release"}},
					},
				},
			},
		},
		"changelog with unreleased": {
			yaml: `
project: myproject
versions:
  - version: unreleased
    changes:
      added:
        - "New feature"
  - version: "1.0.0"
    date: "2024-01-15"
    changes:
      fixed:
        - "Bug fix"
`,
			expected: &Changelog{
				Project: "myproject",
				Versions: []Version{
					{
						Version: "unreleased",
						Changes: Changes{Added: []string{"New feature"}},
					},
					{
						Version: "1.0.0",
						Date:    "2024-01-15",
						Changes: Changes{Fixed: []string{"Bug fix"}},
					},
				},
			},
		},
		"changelog with all categories": {
			yaml: `
project: myproject
versions:
  - version: "2.0.0"
    date: "2024-02-20"
    changes:
      added:
        - "New feature A"
      changed:
        - "Modified behavior"
      deprecated:
        - "Old API"
      removed:
        - "Legacy function"
      fixed:
        - "Critical bug"
      security:
        - "CVE-2024-1234"
`,
			expected: &Changelog{
				Project: "myproject",
				Versions: []Version{
					{
						Version: "2.0.0",
						Date:    "2024-02-20",
						Changes: Changes{
							Added:      []string{"New feature A"},
							Changed:    []string{"Modified behavior"},
							Deprecated: []string{"Old API"},
							Removed:    []string{"Legacy function"},
							Fixed:      []string{"Critical bug"},
							Security:   []string{"CVE-2024-1234"},
						},
					},
				},
			},
		},
		"semver with prerelease": {
			yaml: `
project: myproject
versions:
  - version: "1.0.0-beta.1"
    date: "2024-01-10"
    changes:
      added:
        - "Beta feature"
`,
			expected: &Changelog{
				Project: "myproject",
				Versions: []Version{
					{
						Version: "1.0.0-beta.1",
						Date:    "2024-01-10",
						Changes: Changes{Added: []string{"Beta feature"}},
					},
				},
			},
		},
		"chlog map format": {
			yaml: `
project: myproject
versions:
  unreleased:
    added:
      - "New feature"
  1.0.0:
    date: "2024-01-15"
    fixed:
      - "Bug fix"
`,
			expected: &Changelog{
				Project: "myproject",
				Versions: []Version{
					{
						Version: "unreleased",
						Changes: Changes{Added: []string{"New feature"}},
					},
					{
						Version: "1.0.0",
						Date:    "2024-01-15",
						Changes: Changes{Fixed: []string{"Bug fix"}},
					},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := LoadFromReader(strings.NewReader(tt.yaml))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadFromReader_InvalidYAML(t *testing.T) {
	tests := map[string]struct {
		yaml        string
		errContains string
	}{
		"malformed yaml syntax": {
			yaml: `
project: myproject
versions:
  - version: "1.0.0"
    date: [invalid
`,
			errContains: "parsing changelog YAML",
		},
		"invalid yaml structure": {
			yaml:        `not: a: valid: yaml`,
			errContains: "parsing changelog YAML",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := LoadFromReader(strings.NewReader(tt.yaml))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}

func TestValidate_MissingProject(t *testing.T) {
	changelog := &Changelog{
		Project: "",
		Versions: []Version{
			{
				Version: "1.0.0",
				Date:    "2024-01-15",
				Changes: Changes{Added: []string{"Feature"}},
			},
		},
	}

	err := Validate(changelog)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "project")
	assert.Contains(t, err.Error(), "required field is empty")
}

func TestValidate_VersionErrors(t *testing.T) {
	tests := map[string]struct {
		changelog   *Changelog
		errContains string
	}{
		"empty version string": {
			changelog: &Changelog{
				Project: "myproject",
				Versions: []Version{
					{Version: "", Date: "2024-01-15", Changes: Changes{Added: []string{"Feature"}}},
				},
			},
			errContains: "versions[0].version: required field is empty",
		},
		"missing date for released version": {
			changelog: &Changelog{
				Project: "myproject",
				Versions: []Version{
					{Version: "1.0.0", Date: "", Changes: Changes{Added: []string{"Feature"}}},
				},
			},
			errContains: "date is required for released versions",
		},
		"invalid semver format": {
			changelog: &Changelog{
				Project: "myproject",
				Versions: []Version{
					{Version: "1.0", Date: "2024-01-15", Changes: Changes{Added: []string{"Feature"}}},
				},
			},
			errContains: "invalid semver format",
		},
		"invalid date format": {
			changelog: &Changelog{
				Project: "myproject",
				Versions: []Version{
					{Version: "1.0.0", Date: "2024/01/15", Changes: Changes{Added: []string{"Feature"}}},
				},
			},
			errContains: "invalid date format",
		},
		"empty changes": {
			changelog: &Changelog{
				Project: "myproject",
				Versions: []Version{
					{Version: "1.0.0", Date: "2024-01-15", Changes: Changes{}},
				},
			},
			errContains: "at least one change entry is required",
		},
		"empty change entry": {
			changelog: &Changelog{
				Project: "myproject",
				Versions: []Version{
					{Version: "1.0.0", Date: "2024-01-15", Changes: Changes{Added: []string{"", "Valid"}}},
				},
			},
			errContains: "change entry cannot be empty",
		},
		"whitespace-only change entry": {
			changelog: &Changelog{
				Project: "myproject",
				Versions: []Version{
					{Version: "1.0.0", Date: "2024-01-15", Changes: Changes{Fixed: []string{"   "}}},
				},
			},
			errContains: "change entry cannot be empty",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := Validate(tt.changelog)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
			assert.True(t, IsValidationError(err))
		})
	}
}

func TestValidate_MultipleUnreleased(t *testing.T) {
	changelog := &Changelog{
		Project: "myproject",
		Versions: []Version{
			{Version: "unreleased", Changes: Changes{Added: []string{"Feature 1"}}},
			{Version: "unreleased", Changes: Changes{Added: []string{"Feature 2"}}},
		},
	}

	err := Validate(changelog)
	require.Error(t, err)
	// Duplicate version check catches this before unreleased count check
	assert.Contains(t, err.Error(), "duplicate version")
}

func TestValidate_DuplicateVersions(t *testing.T) {
	changelog := &Changelog{
		Project: "myproject",
		Versions: []Version{
			{Version: "1.0.0", Date: "2024-01-15", Changes: Changes{Added: []string{"A"}}},
			{Version: "1.0.0", Date: "2024-01-10", Changes: Changes{Added: []string{"B"}}},
		},
	}

	err := Validate(changelog)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate version")
}

func TestValidate_UnreleasedNoDateRequired(t *testing.T) {
	changelog := &Changelog{
		Project: "myproject",
		Versions: []Version{
			{Version: "unreleased", Changes: Changes{Added: []string{"New feature"}}},
		},
	}

	err := Validate(changelog)
	assert.NoError(t, err)
}

func TestValidate_EmptyUnreleasedAllowed(t *testing.T) {
	changelog := &Changelog{
		Project: "myproject",
		Versions: []Version{
			{Version: "unreleased", Changes: Changes{}},
			{Version: "1.0.0", Date: "2024-01-15", Changes: Changes{Added: []string{"Initial release"}}},
		},
	}

	err := Validate(changelog)
	assert.NoError(t, err)
}

func TestValidate_UnreleasedWithDateAllowed(t *testing.T) {
	changelog := &Changelog{
		Project: "myproject",
		Versions: []Version{
			{
				Version: "unreleased",
				Date:    "2024-01-20",
				Changes: Changes{Added: []string{"New feature"}},
			},
		},
	}

	err := Validate(changelog)
	assert.NoError(t, err)
}

func TestNormalizeVersion(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"no prefix":          {input: "1.0.0", expected: "1.0.0"},
		"lowercase v prefix": {input: "v1.0.0", expected: "1.0.0"},
		"uppercase V prefix": {input: "V1.0.0", expected: "1.0.0"},
		"unreleased":         {input: "unreleased", expected: "unreleased"},
		"prerelease":         {input: "v1.0.0-beta.1", expected: "1.0.0-beta.1"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := NormalizeVersion(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestChanges_IsEmpty(t *testing.T) {
	tests := map[string]struct {
		changes  Changes
		expected bool
	}{
		"all empty":      {changes: Changes{}, expected: true},
		"has added":      {changes: Changes{Added: []string{"A"}}, expected: false},
		"has changed":    {changes: Changes{Changed: []string{"A"}}, expected: false},
		"has deprecated": {changes: Changes{Deprecated: []string{"A"}}, expected: false},
		"has removed":    {changes: Changes{Removed: []string{"A"}}, expected: false},
		"has fixed":      {changes: Changes{Fixed: []string{"A"}}, expected: false},
		"has security":   {changes: Changes{Security: []string{"A"}}, expected: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.changes.IsEmpty())
		})
	}
}

func TestChanges_Count(t *testing.T) {
	tests := map[string]struct {
		changes  Changes
		expected int
	}{
		"empty": {changes: Changes{}, expected: 0},
		"one category": {
			changes:  Changes{Added: []string{"A", "B"}},
			expected: 2,
		},
		"multiple categories": {
			changes: Changes{
				Added:   []string{"A"},
				Changed: []string{"B", "C"},
				Fixed:   []string{"D"},
			},
			expected: 4,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.changes.Count())
		})
	}
}

func TestVersion_IsUnreleased(t *testing.T) {
	tests := map[string]struct {
		version  Version
		expected bool
	}{
		"unreleased":       {version: Version{Version: "unreleased"}, expected: true},
		"released version": {version: Version{Version: "1.0.0"}, expected: false},
		"Unreleased caps":  {version: Version{Version: "Unreleased"}, expected: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.version.IsUnreleased())
		})
	}
}

func TestVersion_Entries(t *testing.T) {
	version := Version{
		Version: "1.0.0",
		Changes: Changes{
			Added:   []string{"Feature A", "Feature B"},
			Fixed:   []string{"Bug fix"},
			Changed: []string{"Behavior change"},
		},
	}

	entries := version.Entries()
	assert.Len(t, entries, 4)

	// Verify all entries have correct version
	for _, e := range entries {
		assert.Equal(t, "1.0.0", e.Version)
	}

	// Verify categories
	categories := make(map[string]int)
	for _, e := range entries {
		categories[e.Category]++
	}
	assert.Equal(t, 2, categories["added"])
	assert.Equal(t, 1, categories["fixed"])
	assert.Equal(t, 1, categories["changed"])
}

func TestValidCategories(t *testing.T) {
	categories := ValidCategories()
	assert.Len(t, categories, 6)
	assert.Contains(t, categories, "added")
	assert.Contains(t, categories, "changed")
	assert.Contains(t, categories, "deprecated")
	assert.Contains(t, categories, "removed")
	assert.Contains(t, categories, "fixed")
	assert.Contains(t, categories, "security")
}

func TestIsValidationError(t *testing.T) {
	tests := map[string]struct {
		err      error
		expected bool
	}{
		"validation error": {
			err:      &ValidationError{Field: "test", Message: "error"},
			expected: true,
		},
		"other error": {
			err:      assert.AnError,
			expected: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := IsValidationError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := map[string]struct {
		err      *ValidationError
		expected string
	}{
		"with field": {
			err:      &ValidationError{Field: "project", Message: "required"},
			expected: "project: required",
		},
		"without field": {
			err:      &ValidationError{Message: "general error"},
			expected: "general error",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}
