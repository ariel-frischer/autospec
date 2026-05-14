package workflow

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
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
			want:     []string{"Command: go test ./...", "Status: completed", "stdout:", "line 3", "stderr:", "warning", "truncated 2 lines"},
			deny:     []string{"line 4", "line 5"},
		},
		"file change summarized": {
			maxLines: 40,
			line:     `{"type":"item.completed","item":{"type":"file_change","path":"internal/workflow/codex_formatter.go","action":"modified","diff":"large diff omitted"}}`,
			want:     []string{"File change: modified internal/workflow/codex_formatter.go"},
			deny:     []string{"large diff omitted"},
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
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q:\n%s", want, got)
				}
			}
			for _, deny := range tt.deny {
				if strings.Contains(got, deny) {
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

func numberedLines(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i+1)
	}
	return strings.Join(lines, "\n")
}
