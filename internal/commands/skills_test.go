package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandSkillContent(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		tpl      CommandTemplate
		specsDir string
		want     []string
		wantNot  []string
	}{
		"converts template to agent skill": {
			tpl: CommandTemplate{
				Name:        "autospec.clarify",
				Description: "Detect and reduce ambiguity",
				Content: []byte(`---
description: Detect and reduce ambiguity
version: "1.0.0"
---

Run /autospec.plan after clarification.
`),
			},
			specsDir: "specs",
			want: []string{
				"name: autospec-clarify",
				"description: \"Detect and reduce ambiguity\"",
				`"$autospec-clarify"`,
				`"/autospec.clarify"`,
				"$autospec-plan",
				"Project specs directory: specs",
			},
			wantNot: []string{
				"/autospec.plan",
				"version:",
			},
		},
		"uses fallback description": {
			tpl: CommandTemplate{
				Name:    "autospec.specify",
				Content: []byte("Create a spec."),
			},
			specsDir: "features",
			want: []string{
				"name: autospec-specify",
				"description: \"Run the autospec.specify autospec workflow prompt.\"",
				"Project specs directory: features",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := CommandSkillContent(tt.tpl, tt.specsDir)

			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("CommandSkillContent() missing %q:\n%s", want, got)
				}
			}
			for _, wantNot := range tt.wantNot {
				if strings.Contains(got, wantNot) {
					t.Errorf("CommandSkillContent() should not contain %q:\n%s", wantNot, got)
				}
			}
		})
	}
}

func TestInstallAgentSkills(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()

	skillNames, allExisted, err := InstallAgentSkills(projectDir, "specs")
	if err != nil {
		t.Fatalf("InstallAgentSkills() error = %v", err)
	}
	if allExisted {
		t.Fatal("first install allExisted = true, want false")
	}
	if len(skillNames) == 0 {
		t.Fatal("InstallAgentSkills() returned no skill names")
	}

	for _, skillName := range skillNames {
		skillPath := filepath.Join(projectDir, ".agents", "skills", skillName, "SKILL.md")
		content, err := os.ReadFile(skillPath)
		if err != nil {
			t.Fatalf("expected skill %s to exist: %v", skillName, err)
		}
		if !strings.Contains(string(content), "name: "+skillName) {
			t.Errorf("skill %s missing frontmatter name:\n%s", skillName, string(content))
		}
	}

	specifyPath := filepath.Join(projectDir, ".agents", "skills", "autospec-specify", "SKILL.md")
	if err := os.WriteFile(specifyPath, []byte("stale skill\n"), 0o644); err != nil {
		t.Fatalf("writing stale skill: %v", err)
	}

	_, allExisted, err = InstallAgentSkills(projectDir, "specs")
	if err != nil {
		t.Fatalf("second InstallAgentSkills() error = %v", err)
	}
	if !allExisted {
		t.Fatal("second install allExisted = false, want true")
	}

	refreshed, err := os.ReadFile(specifyPath)
	if err != nil {
		t.Fatalf("reading refreshed skill: %v", err)
	}
	if !strings.Contains(string(refreshed), "name: autospec-specify") {
		t.Fatalf("skill was not refreshed:\n%s", string(refreshed))
	}
}

func TestInstallClaudeSkills(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()

	skillNames, allExisted, err := InstallClaudeSkills(projectDir, "specs")
	if err != nil {
		t.Fatalf("InstallClaudeSkills() error = %v", err)
	}
	if allExisted {
		t.Fatal("first install allExisted = true, want false")
	}
	if len(skillNames) == 0 {
		t.Fatal("InstallClaudeSkills() returned no skill names")
	}

	skillPath := filepath.Join(projectDir, ".claude", "skills", "autospec.specify", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	if err != nil {
		t.Fatalf("expected Claude specify skill to exist: %v", err)
	}
	for _, want := range []string{
		"name: autospec-specify",
		"disable-model-invocation: true",
		`"/autospec.specify"`,
		"Project specs directory: specs",
	} {
		if !strings.Contains(string(content), want) {
			t.Errorf("Claude skill missing %q:\n%s", want, string(content))
		}
	}

	_, allExisted, err = InstallClaudeSkills(projectDir, "specs")
	if err != nil {
		t.Fatalf("second InstallClaudeSkills() error = %v", err)
	}
	if !allExisted {
		t.Fatal("second install allExisted = false, want true")
	}
}
