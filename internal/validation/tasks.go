package validation

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// taskPattern matches task lines: "- [ ]" or "- [x]" or "* [ ]" or "* [X]"
	taskPattern = regexp.MustCompile(`^(\s*)[-*]\s+\[([ xX])\]\s+(.+)$`)
	// phasePattern matches phase headings: "## Phase Name"
	phasePattern = regexp.MustCompile(`^##\s+(.+)$`)
)

// Task represents an individual task in tasks.md
type Task struct {
	Description string
	Checked     bool
	LineNumber  int
	PhaseName   string
	IndentLevel int
}

// Phase represents a section in tasks.md (identified by ## heading)
type Phase struct {
	Name         string
	Tasks        []Task
	LineNumber   int
	TotalTasks   int
	CheckedTasks int
}

// UncheckedTasks returns the number of unchecked tasks in this phase
func (p *Phase) UncheckedTasks() int {
	return p.TotalTasks - p.CheckedTasks
}

// IsComplete returns true if all tasks in this phase are checked
func (p *Phase) IsComplete() bool {
	return p.UncheckedTasks() == 0
}

// Progress returns the completion percentage (0.0 to 1.0)
func (p *Phase) Progress() float64 {
	if p.TotalTasks == 0 {
		return 1.0
	}
	return float64(p.CheckedTasks) / float64(p.TotalTasks)
}

// CountUncheckedTasks counts the number of unchecked tasks in a tasks.md file
// Performance contract: <50ms
func CountUncheckedTasks(tasksPath string) (int, error) {
	file, err := os.Open(tasksPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open tasks file: %w", err)
	}
	defer file.Close()

	unchecked := 0
	scanner := bufio.Scanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if match := taskPattern.FindStringSubmatch(line); match != nil {
			checkbox := match[2]
			if strings.TrimSpace(checkbox) == "" {
				unchecked++
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading tasks file: %w", err)
	}

	return unchecked, nil
}

// ValidateTasksComplete checks if all tasks in tasks.md are checked
func ValidateTasksComplete(tasksPath string) (bool, error) {
	count, err := CountUncheckedTasks(tasksPath)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

// ParseTasksByPhase parses tasks.md and groups tasks by phase
// Performance contract: <100ms
func ParseTasksByPhase(tasksPath string) ([]Phase, error) {
	file, err := os.Open(tasksPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open tasks file: %w", err)
	}
	defer file.Close()

	var phases []Phase
	var currentPhase *Phase
	lineNum := 0

	scanner := bufio.Scanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Check for phase heading
		if match := phasePattern.FindStringSubmatch(line); match != nil {
			// Save previous phase if exists
			if currentPhase != nil {
				phases = append(phases, *currentPhase)
			}

			// Start new phase
			currentPhase = &Phase{
				Name:       match[1],
				LineNumber: lineNum,
				Tasks:      []Task{},
			}
			continue
		}

		// Check for task
		if match := taskPattern.FindStringSubmatch(line); match != nil {
			if currentPhase == nil {
				// Task without phase - skip or create default phase
				currentPhase = &Phase{
					Name:       "Uncategorized",
					LineNumber: 0,
					Tasks:      []Task{},
				}
			}

			indent := len(match[1])
			checkbox := match[2]
			desc := match[3]
			checked := strings.ToLower(strings.TrimSpace(checkbox)) == "x"

			task := Task{
				Description: desc,
				Checked:     checked,
				LineNumber:  lineNum,
				PhaseName:   currentPhase.Name,
				IndentLevel: indent,
			}

			currentPhase.Tasks = append(currentPhase.Tasks, task)
			currentPhase.TotalTasks++
			if checked {
				currentPhase.CheckedTasks++
			}
		}
	}

	// Add last phase
	if currentPhase != nil {
		phases = append(phases, *currentPhase)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading tasks file: %w", err)
	}

	return phases, nil
}

// GetTasksFilePath returns the path to tasks.md for a given spec directory
func GetTasksFilePath(specDir string) string {
	return filepath.Join(specDir, "tasks.md")
}
