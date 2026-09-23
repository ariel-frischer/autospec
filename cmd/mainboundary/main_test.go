package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.invalid")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}

func writeTestFile(t *testing.T, dir, path, content string) {
	t.Helper()
	fullPath := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestMainBoundary(t *testing.T) {
	tests := map[string]struct {
		setup   func(*testing.T, string)
		wantErr bool
	}{
		"clean candidate": {},
		"video in HEAD": {setup: func(t *testing.T, dir string) {
			writeTestFile(t, dir, "video/source.go", "package video\n")
			gitTest(t, dir, "add", "video/source.go")
			gitTest(t, dir, "commit", "-qm", "add video")
		}, wantErr: true},
		"video removed but present in ancestry": {setup: func(t *testing.T, dir string) {
			writeTestFile(t, dir, "video/source.go", "package video\n")
			gitTest(t, dir, "add", "video/source.go")
			gitTest(t, dir, "commit", "-qm", "add video")
			gitTest(t, dir, "rm", "video/source.go")
			gitTest(t, dir, "commit", "-qm", "remove video")
		}, wantErr: true},
		"video added and removed on merged branch": {setup: func(t *testing.T, dir string) {
			gitTest(t, dir, "checkout", "-qb", "topic")
			writeTestFile(t, dir, "video/source.go", "package video\n")
			gitTest(t, dir, "add", "video/source.go")
			gitTest(t, dir, "commit", "-qm", "add video on topic")
			gitTest(t, dir, "rm", "video/source.go")
			gitTest(t, dir, "commit", "-qm", "remove video on topic")
			gitTest(t, dir, "checkout", "-q", "-")
			gitTest(t, dir, "merge", "--no-ff", "-qm", "merge topic", "topic")
		}, wantErr: true},
		"ignored and untracked media": {setup: func(t *testing.T, dir string) {
			writeTestFile(t, dir, ".gitignore", "video/*.mp4\n")
			writeTestFile(t, dir, "video/ignored.mp4", "media")
			writeTestFile(t, dir, "video/untracked.jpg", "media")
			gitTest(t, dir, "add", ".gitignore")
			gitTest(t, dir, "commit", "-qm", "ignore media")
		}},
		"invalid repository": {setup: func(t *testing.T, dir string) {
			if err := os.RemoveAll(filepath.Join(dir, ".git")); err != nil {
				t.Fatal(err)
			}
		}, wantErr: true},
		"shallow repository": {setup: func(t *testing.T, dir string) {
			cmd := exec.Command("git", "rev-parse", "HEAD")
			cmd.Dir = dir
			commit, err := cmd.Output()
			if err != nil {
				t.Fatalf("finding shallow boundary: %v", err)
			}
			writeTestFile(t, dir, ".git/shallow", string(commit))
		}, wantErr: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			gitTest(t, dir, "init", "-q")
			writeTestFile(t, dir, "README.md", "clean\n")
			gitTest(t, dir, "add", "README.md")
			gitTest(t, dir, "commit", "-qm", "initial")
			if tt.setup != nil {
				tt.setup(t, dir)
			}
			err := runGuard(dir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("runGuard() error = %v, wantErr %v", err, tt.wantErr)
			}
			if strings.HasPrefix(name, "video ") && err != nil && !strings.Contains(err.Error(), "video/") {
				t.Errorf("error should name prohibited path: %v", err)
			}
		})
	}
}

func TestMainBoundaryDeepCleanHistory(t *testing.T) {
	dir := t.TempDir()
	gitTest(t, dir, "init", "-q")
	writeTestFile(t, dir, "README.md", "clean\n")
	gitTest(t, dir, "add", "README.md")
	gitTest(t, dir, "commit", "-qm", "initial")
	for range 50 {
		gitTest(t, dir, "commit", "--allow-empty", "-qm", "clean")
	}
	trace := filepath.Join(t.TempDir(), "git.trace")
	t.Setenv("GIT_TRACE", trace)
	if err := runGuard(dir); err != nil {
		t.Fatalf("deep clean ancestry rejected: %v", err)
	}
	contents, err := os.ReadFile(trace)
	if err != nil {
		t.Fatalf("reading Git invocation trace: %v", err)
	}
	if calls := strings.Count(string(contents), "git ls-tree"); calls > 1 {
		t.Fatalf("checked %d trees using separate Git processes; want at most HEAD", calls)
	}
}

func TestMainBoundaryCLI(t *testing.T) {
	tests := map[string]struct {
		addVideo bool
		wantErr  bool
	}{
		"clean candidate exits successfully": {},
		"video candidate exits nonzero":      {addVideo: true, wantErr: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			gitTest(t, dir, "init", "-q")
			path := "README.md"
			if tt.addVideo {
				path = "video/source.go"
			}
			writeTestFile(t, dir, path, "candidate\n")
			gitTest(t, dir, "add", path)
			gitTest(t, dir, "commit", "-qm", "candidate")
			source, err := filepath.Abs("main.go")
			if err != nil {
				t.Fatal(err)
			}
			command := exec.Command("go", "run", source)
			command.Dir = dir
			output, err := command.CombinedOutput()
			if (err != nil) != tt.wantErr {
				t.Fatalf("command error = %v, wantErr %v; output: %s", err, tt.wantErr, output)
			}
		})
	}
}
