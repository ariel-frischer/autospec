package history

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateID(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		validate func(t *testing.T, id string)
	}{
		"returns non-empty ID": {
			validate: func(t *testing.T, id string) {
				assert.NotEmpty(t, id)
			},
		},
		"matches expected format adjective_noun_YYYYMMDD_HHMMSS": {
			validate: func(t *testing.T, id string) {
				pattern := `^[a-z]+_[a-z]+_\d{8}_\d{6}$`
				matched, err := regexp.MatchString(pattern, id)
				require.NoError(t, err)
				assert.True(t, matched, "ID '%s' should match pattern %s", id, pattern)
			},
		},
		"contains valid adjective from word list": {
			validate: func(t *testing.T, id string) {
				pattern := regexp.MustCompile(`^([a-z]+)_`)
				matches := pattern.FindStringSubmatch(id)
				require.Len(t, matches, 2, "should extract adjective from ID")
				adjective := matches[1]
				assert.Contains(t, adjectives, adjective, "adjective '%s' should be in word list", adjective)
			},
		},
		"contains valid noun from word list": {
			validate: func(t *testing.T, id string) {
				pattern := regexp.MustCompile(`^[a-z]+_([a-z]+)_`)
				matches := pattern.FindStringSubmatch(id)
				require.Len(t, matches, 2, "should extract noun from ID")
				noun := matches[1]
				assert.Contains(t, nouns, noun, "noun '%s' should be in word list", noun)
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			id, err := GenerateID()
			require.NoError(t, err)
			tc.validate(t, id)
		})
	}
}

func TestGenerateID_Uniqueness(t *testing.T) {
	t.Parallel()

	// Generate multiple IDs rapidly. With 50x50=2500 combinations per second,
	// we expect good uniqueness but allow for rare same-second collisions.
	// Typical usage is one command at a time, so this tests beyond normal use.
	const numIDs = 20
	ids := make(map[string]bool, numIDs)

	for i := 0; i < numIDs; i++ {
		id, err := GenerateID()
		require.NoError(t, err)

		// Track all generated IDs
		ids[id] = true
	}

	// With 20 IDs from 2500 combinations, expect very high uniqueness
	// Allow at most 1 collision (birthday paradox: ~7% chance of any collision)
	minUnique := numIDs - 1
	assert.GreaterOrEqual(t, len(ids), minUnique, "should have at least %d unique IDs out of %d", minUnique, numIDs)
}

func TestWordLists(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		words    []string
		listName string
	}{
		"adjectives list has expected size": {
			words:    adjectives,
			listName: "adjectives",
		},
		"nouns list has expected size": {
			words:    nouns,
			listName: "nouns",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Should have at least 50 words
			assert.GreaterOrEqual(t, len(tc.words), 50, "%s list should have at least 50 words", tc.listName)

			// All words should be lowercase alphanumeric
			pattern := regexp.MustCompile(`^[a-z]+$`)
			for _, word := range tc.words {
				assert.True(t, pattern.MatchString(word), "word '%s' in %s should be lowercase letters only", word, tc.listName)
			}

			// Check for duplicates
			seen := make(map[string]bool)
			for _, word := range tc.words {
				assert.False(t, seen[word], "word '%s' is duplicated in %s", word, tc.listName)
				seen[word] = true
			}
		})
	}
}

func TestRandomWord(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		words   []string
		wantErr bool
	}{
		"selects from non-empty list": {
			words:   []string{"apple", "banana", "cherry"},
			wantErr: false,
		},
		"returns error for empty list": {
			words:   []string{},
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			word, err := randomWord(tc.words)

			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Contains(t, tc.words, word, "selected word should be from the input list")
		})
	}
}

func TestRandomWord_Distribution(t *testing.T) {
	t.Parallel()

	// Verify that randomWord produces varied results
	words := []string{"a", "b", "c", "d", "e"}
	counts := make(map[string]int)

	const iterations = 500
	for i := 0; i < iterations; i++ {
		word, err := randomWord(words)
		require.NoError(t, err)
		counts[word]++
	}

	// Each word should appear at least once (very high probability with 500 iterations)
	for _, w := range words {
		assert.Greater(t, counts[w], 0, "word '%s' should have been selected at least once", w)
	}
}
