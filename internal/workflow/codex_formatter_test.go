package workflow

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestCodexFormatterProcessLine(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		maxLines int
		line     string
		want     []string
		deny     []string
	}{
		"agent message under limit": {
			maxLines: 40,
			line:     `{"type":"item.completed","item":{"type":"agent_message","text":"hello\nworld"}}`,
			want:     []string{"Agent message", "hello", "world"},
		},
		"agent message truncated": {
			maxLines: 40,
			line:     fmt.Sprintf(`{"type":"item.completed","item":{"type":"agent_message","text":%q}}`, numberedLines(42)),
			want:     []string{"line 40", "... truncated 2 lines; set codex_output.mode: full for complete Codex output"},
			deny:     []string{"line 41", "line 42"},
		},
		"command execution summarized": {
			maxLines: 3,
			line:     fmt.Sprintf(`{"type":"item.completed","item":{"type":"command_execution","command":"go test ./...","status":"completed","stdout":%q,"stderr":%q}}`, numberedLines(5), "warning\n"),
			want:     []string{"Command: go test ./...", "stdout:", "line 3", "stderr:", "warning", "truncated 2 lines"},
			deny:     []string{"Status: completed", "line 4", "line 5"},
		},
		"command execution shows non-success status": {
			maxLines: 3,
			line:     `{"type":"item.completed","item":{"type":"command_execution","command":"go test ./...","status":"failed"}}`,
			want:     []string{"Command: go test ./...", "Status: failed"},
		},
		"command execution without output stays one line": {
			maxLines: 3,
			line:     `{"type":"item.completed","item":{"type":"command_execution","command":"git branch --show-current","status":"completed"}}`,
			want:     []string{"Command: git branch --show-current"},
			deny:     []string{"line 4", "line 5"},
		},
		"file change summarized": {
			maxLines: 40,
			line:     `{"type":"item.completed","item":{"type":"file_change","path":"internal/workflow/codex_formatter.go","action":"modified","diff":"large diff omitted"}}`,
			want:     []string{"File change: modified internal/workflow/codex_formatter.go"},
			deny:     []string{"large diff omitted"},
		},
		"nested file change summarized": {
			maxLines: 40,
			line:     `{"type":"item.completed","item":{"type":"file_change","change":{"path":"specs/128-persist-feature-directory/plan.yaml","operation":"created"}}}`,
			want:     []string{"File change: created specs/128-persist-feature-directory/plan.yaml"},
		},
		"file change array summarized": {
			maxLines: 40,
			line:     `{"type":"item.completed","item":{"type":"file_change","changes":[{"path":"specs/128-persist-feature-directory/plan.yaml","action":"created"},{"file":"internal/validation/artifact_plan.go","action":"modified"}]}}`,
			want: []string{
				"File change: created specs/128-persist-feature-directory/plan.yaml",
				"File change: modified internal/validation/artifact_plan.go",
			},
		},
		"real codex file change array uses kind": {
			maxLines: 40,
			line:     `{"type":"item.completed","item":{"id":"item_3","type":"file_change","changes":[{"path":"/tmp/codex-output-check.txt","kind":"update"}],"status":"completed"}}`,
			want:     []string{"File change: update /tmp/codex-output-check.txt"},
		},
		"file change string array uses item action": {
			maxLines: 40,
			line:     `{"type":"item.completed","item":{"type":"file_change","action":"changed","files":["specs/128-persist-feature-directory/plan.yaml"]}}`,
			want:     []string{"File change: changed specs/128-persist-feature-directory/plan.yaml"},
		},
		"malformed JSON passes through": {
			maxLines: 40,
			line:     `{not json`,
			want:     []string{"{not json"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			formatter := NewCodexFormatter(tt.maxLines, &out)
			formatter.ProcessLine(tt.line)

			got := out.String()
			gotPlain := stripANSI(got)
			for _, want := range tt.want {
				if !strings.Contains(gotPlain, want) {
					t.Errorf("output missing %q:\n%s", want, got)
				}
			}
			for _, deny := range tt.deny {
				if strings.Contains(gotPlain, deny) {
					t.Errorf("output contains %q:\n%s", deny, got)
				}
			}
		})
	}
}

func TestCodexFormatterWriterFlushesPartialLine(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	writer := NewCodexFormatterWriter(40, &out)
	if _, err := writer.Write([]byte(`{"type":"item.completed","item":{"type":"reasoning","summary":"checked config"}}`)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	writer.Flush()

	got := out.String()
	if !strings.Contains(got, "Reasoning:") || !strings.Contains(got, "checked config") {
		t.Fatalf("output = %q, want reasoning summary", got)
	}
}

func TestCodexFormatterUsesColorForParsedBlocks(t *testing.T) {
	t.Cleanup(func() { color.NoColor = false })
	color.NoColor = false

	tests := map[string]struct {
		line     string
		wantANSI bool
	}{
		"agent message is colored": {
			line:     `{"type":"item.completed","item":{"type":"agent_message","text":"hello"}}`,
			wantANSI: true,
		},
		"command execution is colored": {
			line:     `{"type":"item.completed","item":{"type":"command_execution","command":"go test ./...","status":"completed"}}`,
			wantANSI: true,
		},
		"malformed JSON remains raw": {
			line:     `{not json`,
			wantANSI: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var out bytes.Buffer
			formatter := NewCodexFormatter(40, &out)
			formatter.ProcessLine(tt.line)

			gotANSI := strings.Contains(out.String(), "\x1b[")
			if gotANSI != tt.wantANSI {
				t.Fatalf("ANSI presence = %v, want %v; output = %q", gotANSI, tt.wantANSI, out.String())
			}
		})
	}
}

func TestCodexFormatterCanDisableColor(t *testing.T) {
	t.Cleanup(func() { color.NoColor = false })
	color.NoColor = false

	var out bytes.Buffer
	formatter := NewCodexFormatterWithOptions(CodexFormatterOptions{
		MaxLines:     40,
		ColorEnabled: false,
		Writer:       &out,
	})
	formatter.ProcessLine(`{"type":"item.completed","item":{"type":"agent_message","text":"hello"}}`)

	if strings.Contains(out.String(), "\x1b[") {
		t.Fatalf("output should not contain ANSI when color is disabled: %q", out.String())
	}
	if !strings.Contains(out.String(), "Agent message") {
		t.Fatalf("output = %q, want agent message label", out.String())
	}
}

func numberedLines(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i+1)
	}
	return strings.Join(lines, "\n")
}

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}
