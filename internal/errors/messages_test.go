package errors

import (
	"strings"
	"testing"
)

func TestMissingFeatureDescription(t *testing.T) {
	err := MissingFeatureDescription()

	if err.Category != Argument {
		t.Errorf("Expected Argument category, got %v", err.Category)
	}
	if err.Usage == "" {
		t.Error("Expected non-empty usage")
	}
	if len(err.Remediation) == 0 {
		t.Error("Expected remediation steps")
	}
}

func TestMissingSpecFile(t *testing.T) {
	err := MissingSpecFile("/path/to/spec")

	if err.Category != Prerequisite {
		t.Errorf("Expected Prerequisite category, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "/path/to/spec") {
		t.Error("Expected message to contain path")
	}
}

func TestMissingPlanFile(t *testing.T) {
	err := MissingPlanFile("/path/to/plan")

	if err.Category != Prerequisite {
		t.Errorf("Expected Prerequisite category, got %v", err.Category)
	}
}

func TestMissingTasksFile(t *testing.T) {
	err := MissingTasksFile("/path/to/tasks")

	if err.Category != Prerequisite {
		t.Errorf("Expected Prerequisite category, got %v", err.Category)
	}
}

func TestSpecNotDetected(t *testing.T) {
	err := SpecNotDetected()

	if err.Category != Prerequisite {
		t.Errorf("Expected Prerequisite category, got %v", err.Category)
	}
	if len(err.Remediation) == 0 {
		t.Error("Expected remediation steps")
	}
}

func TestInvalidSpecNameFormat(t *testing.T) {
	err := InvalidSpecNameFormat("bad-name")

	if err.Category != Argument {
		t.Errorf("Expected Argument category, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "bad-name") {
		t.Error("Expected message to contain provided name")
	}
}

func TestClaudeCliNotFound(t *testing.T) {
	err := ClaudeCliNotFound()

	if err.Category != Prerequisite {
		t.Errorf("Expected Prerequisite category, got %v", err.Category)
	}
	if len(err.Remediation) == 0 {
		t.Error("Expected remediation steps")
	}
}

func TestClaudeCliError(t *testing.T) {
	original := &testError{}
	err := ClaudeCliError(original)

	if err.Category != Runtime {
		t.Errorf("Expected Runtime category, got %v", err.Category)
	}
}

func TestConfigFileNotFound(t *testing.T) {
	err := ConfigFileNotFound("/path/to/config")

	if err.Category != Configuration {
		t.Errorf("Expected Configuration category, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "/path/to/config") {
		t.Error("Expected message to contain path")
	}
}

func TestConfigParseError(t *testing.T) {
	original := &testError{}
	err := ConfigParseError("/path/to/config", original)

	if err.Category != Configuration {
		t.Errorf("Expected Configuration category, got %v", err.Category)
	}
	if len(err.Remediation) == 0 {
		t.Error("Expected remediation steps")
	}
}

func TestInvalidFlagCombination(t *testing.T) {
	err := InvalidFlagCombination("-a -s", "redundant flags")

	if err.Category != Argument {
		t.Errorf("Expected Argument category, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "-a -s") {
		t.Error("Expected message to contain flags")
	}
}

func TestTimeoutError(t *testing.T) {
	err := TimeoutError("5m", "claude /autospec.plan")

	if err.Category != Runtime {
		t.Errorf("Expected Runtime category, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "5m") {
		t.Error("Expected message to contain duration")
	}
}

func TestDirectoryNotFound(t *testing.T) {
	err := DirectoryNotFound("/path/to/dir")

	if err.Category != Prerequisite {
		t.Errorf("Expected Prerequisite category, got %v", err.Category)
	}
}

func TestFileNotWritable(t *testing.T) {
	err := FileNotWritable("/path/to/file")

	if err.Category != Runtime {
		t.Errorf("Expected Runtime category, got %v", err.Category)
	}
}

func TestNoTasksPending(t *testing.T) {
	err := NoTasksPending()

	if err.Category != Prerequisite {
		t.Errorf("Expected Prerequisite category, got %v", err.Category)
	}
}

func TestTaskNotFound(t *testing.T) {
	err := TaskNotFound("T999")

	if err.Category != Argument {
		t.Errorf("Expected Argument category, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "T999") {
		t.Error("Expected message to contain task ID")
	}
}

func TestInvalidTaskStatus(t *testing.T) {
	err := InvalidTaskStatus("BadStatus")

	if err.Category != Argument {
		t.Errorf("Expected Argument category, got %v", err.Category)
	}
	if !strings.Contains(err.Message, "BadStatus") {
		t.Error("Expected message to contain status")
	}
}

func TestGitNotRepository(t *testing.T) {
	err := GitNotRepository()

	if err.Category != Prerequisite {
		t.Errorf("Expected Prerequisite category, got %v", err.Category)
	}
}
