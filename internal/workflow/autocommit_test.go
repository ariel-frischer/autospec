package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAutoCommitInstructions(t *testing.T) {
	t.Parallel()

	instructions := BuildAutoCommitInstructions()

	// Verify instructions are non-empty
	assert.NotEmpty(t, instructions, "instructions should not be empty")

	t.Run("contains gitignore guidance", func(t *testing.T) {
		t.Parallel()

		// Should contain .gitignore section
		assert.Contains(t, instructions, ".gitignore",
			"instructions should mention .gitignore")
		assert.Contains(t, instructions, "Update .gitignore",
			"instructions should include .gitignore update section")

		// Should include common ignorable patterns
		patterns := []string{
			"node_modules",
			"__pycache__",
			".venv",
			"dist/",
			"build/",
			".DS_Store",
			".env",
		}
		for _, pattern := range patterns {
			assert.Contains(t, instructions, pattern,
				"instructions should include common pattern: %s", pattern)
		}
	})

	t.Run("contains staging rules", func(t *testing.T) {
		t.Parallel()

		// Should contain staging guidance
		assert.Contains(t, instructions, "Stage",
			"instructions should mention staging")
		assert.Contains(t, instructions, "git add",
			"instructions should include git add command")
		assert.Contains(t, instructions, "git status",
			"instructions should include git status for verification")

		// Should mention what NOT to stage
		assert.Contains(t, instructions, "Do NOT stage",
			"instructions should mention files not to stage")
	})

	t.Run("contains conventional commit format", func(t *testing.T) {
		t.Parallel()

		// Should contain conventional commit section
		assert.Contains(t, instructions, "Conventional Commit",
			"instructions should mention conventional commit")
		assert.Contains(t, instructions, "type(scope): description",
			"instructions should include commit format template")

		// Should include common commit types
		commitTypes := []string{
			"feat:",
			"fix:",
			"docs:",
			"style:",
			"refactor:",
			"test:",
			"chore:",
		}
		for _, ct := range commitTypes {
			assert.Contains(t, instructions, ct,
				"instructions should include commit type: %s", ct)
		}

		// Should include scope guidance
		assert.Contains(t, instructions, "scope",
			"instructions should mention scope")
		assert.Contains(t, instructions, "git commit",
			"instructions should include git commit command")
	})

	t.Run("is agent-agnostic", func(t *testing.T) {
		t.Parallel()

		// Should NOT contain agent-specific references
		agentSpecificTerms := []string{
			"Claude",
			"Gemini",
			"GPT",
			"OpenAI",
			"Anthropic",
			"Google AI",
			"Copilot",
		}
		instructionsLower := strings.ToLower(instructions)
		for _, term := range agentSpecificTerms {
			assert.NotContains(t, instructionsLower, strings.ToLower(term),
				"instructions should not contain agent-specific term: %s", term)
		}
	})

	t.Run("handles edge cases", func(t *testing.T) {
		t.Parallel()

		// Should mention what to do when there are no changes
		assert.Contains(t, instructions, "no changes",
			"instructions should handle no-changes case")

		// Should mention detached HEAD state
		assert.Contains(t, instructions, "detached HEAD",
			"instructions should handle detached HEAD case")
	})
}

func TestBuildAutoCommitInstructionsIdempotent(t *testing.T) {
	t.Parallel()

	// Calling the function multiple times should return the same result
	first := BuildAutoCommitInstructions()
	second := BuildAutoCommitInstructions()

	assert.Equal(t, first, second,
		"BuildAutoCommitInstructions should be idempotent")
}

func TestAutoCommitInstructionsStructure(t *testing.T) {
	t.Parallel()

	instructions := BuildAutoCommitInstructions()

	tests := map[string]struct {
		section     string
		description string
	}{
		"has step 1 header": {
			section:     "Step 1",
			description: "gitignore update step",
		},
		"has step 2 header": {
			section:     "Step 2",
			description: "staging step",
		},
		"has step 3 header": {
			section:     "Step 3",
			description: "commit creation step",
		},
		"has important notes": {
			section:     "Important Notes",
			description: "edge case handling notes",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Contains(t, instructions, tc.section,
				"instructions should contain %s for %s", tc.section, tc.description)
		})
	}
}
