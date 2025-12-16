package validation

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// TasksYAML represents the complete tasks.yaml structure
type TasksYAML struct {
	Meta    TasksMeta    `yaml:"_meta"`
	Tasks   TasksInfo    `yaml:"tasks"`
	Summary TasksSummary `yaml:"summary"`
	Phases  []TaskPhase  `yaml:"phases"`
}

// TasksMeta contains metadata about the tasks file
type TasksMeta struct {
	Version          string `yaml:"version"`
	Generator        string `yaml:"generator"`
	GeneratorVersion string `yaml:"generator_version"`
	Created          string `yaml:"created"`
	ArtifactType     string `yaml:"artifact_type"`
}

// TasksInfo contains basic task info
type TasksInfo struct {
	Branch   string `yaml:"branch"`
	Created  string `yaml:"created"`
	SpecPath string `yaml:"spec_path"`
	PlanPath string `yaml:"plan_path"`
}

// TasksSummary contains summary statistics from the tasks file
type TasksSummary struct {
	TotalTasks            int    `yaml:"total_tasks"`
	TotalPhases           int    `yaml:"total_phases"`
	ParallelOpportunities int    `yaml:"parallel_opportunities"`
	EstimatedComplexity   string `yaml:"estimated_complexity"`
}

// TaskPhase represents a phase in the tasks file
type TaskPhase struct {
	Number         int        `yaml:"number"`
	Title          string     `yaml:"title"`
	Purpose        string     `yaml:"purpose"`
	StoryReference string     `yaml:"story_reference,omitempty"`
	Tasks          []TaskItem `yaml:"tasks"`
}

// TaskItem represents an individual task
type TaskItem struct {
	ID                 string   `yaml:"id"`
	Title              string   `yaml:"title"`
	Status             string   `yaml:"status"`
	Type               string   `yaml:"type"`
	Parallel           bool     `yaml:"parallel"`
	StoryID            string   `yaml:"story_id,omitempty"`
	FilePath           string   `yaml:"file_path,omitempty"`
	Dependencies       []string `yaml:"dependencies"`
	AcceptanceCriteria []string `yaml:"acceptance_criteria"`
}

// TaskStats contains computed statistics about task completion
type TaskStats struct {
	TotalTasks      int
	CompletedTasks  int
	InProgressTasks int
	PendingTasks    int
	BlockedTasks    int
	TotalPhases     int
	CompletedPhases int
	PhaseStats      []PhaseStats
}

// PhaseStats contains statistics for a single phase
type PhaseStats struct {
	Number         int
	Title          string
	TotalTasks     int
	CompletedTasks int
	IsComplete     bool
}

// CompletionPercentage returns the completion percentage
func (s *TaskStats) CompletionPercentage() float64 {
	if s.TotalTasks == 0 {
		return 100.0
	}
	return float64(s.CompletedTasks) / float64(s.TotalTasks) * 100.0
}

// IsComplete returns true if all tasks are completed
func (s *TaskStats) IsComplete() bool {
	return s.TotalTasks > 0 && s.CompletedTasks == s.TotalTasks
}

// ParseTasksYAML parses a tasks.yaml file and returns the structure
func ParseTasksYAML(tasksPath string) (*TasksYAML, error) {
	data, err := os.ReadFile(tasksPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks file: %w", err)
	}

	var tasks TasksYAML
	if err := yaml.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse tasks YAML: %w", err)
	}

	return &tasks, nil
}

// GetTaskStats computes statistics from a tasks.yaml file
func GetTaskStats(tasksPath string) (*TaskStats, error) {
	// Check if it's a YAML file
	if !strings.HasSuffix(tasksPath, ".yaml") && !strings.HasSuffix(tasksPath, ".yml") {
		// Fall back to markdown parsing for .md files
		return getTaskStatsFromMarkdown(tasksPath)
	}

	tasks, err := ParseTasksYAML(tasksPath)
	if err != nil {
		return nil, err
	}

	stats := &TaskStats{
		TotalPhases: len(tasks.Phases),
		PhaseStats:  make([]PhaseStats, 0, len(tasks.Phases)),
	}

	for _, phase := range tasks.Phases {
		phaseStat := PhaseStats{
			Number:     phase.Number,
			Title:      phase.Title,
			TotalTasks: len(phase.Tasks),
		}

		for _, task := range phase.Tasks {
			stats.TotalTasks++

			switch strings.ToLower(task.Status) {
			case "completed", "done", "complete":
				stats.CompletedTasks++
				phaseStat.CompletedTasks++
			case "in_progress", "inprogress", "in-progress", "wip":
				stats.InProgressTasks++
			case "blocked":
				stats.BlockedTasks++
			default:
				// Pending or unknown status
				stats.PendingTasks++
			}
		}

		phaseStat.IsComplete = phaseStat.TotalTasks > 0 && phaseStat.CompletedTasks == phaseStat.TotalTasks
		if phaseStat.IsComplete {
			stats.CompletedPhases++
		}

		stats.PhaseStats = append(stats.PhaseStats, phaseStat)
	}

	return stats, nil
}

// getTaskStatsFromMarkdown parses markdown tasks.md and returns stats
func getTaskStatsFromMarkdown(tasksPath string) (*TaskStats, error) {
	phases, err := ParseTasksByPhase(tasksPath)
	if err != nil {
		return nil, err
	}

	stats := &TaskStats{
		TotalPhases: len(phases),
		PhaseStats:  make([]PhaseStats, 0, len(phases)),
	}

	for i, phase := range phases {
		stats.TotalTasks += phase.TotalTasks
		stats.CompletedTasks += phase.CheckedTasks
		stats.PendingTasks += phase.UncheckedTasks()

		phaseStat := PhaseStats{
			Number:         i + 1,
			Title:          phase.Name,
			TotalTasks:     phase.TotalTasks,
			CompletedTasks: phase.CheckedTasks,
			IsComplete:     phase.IsComplete(),
		}

		if phaseStat.IsComplete {
			stats.CompletedPhases++
		}

		stats.PhaseStats = append(stats.PhaseStats, phaseStat)
	}

	return stats, nil
}

// PhaseInfo contains detailed information about a phase's status for execution decisions
type PhaseInfo struct {
	Number          int    // Phase number (1-based)
	Title           string // Phase title from tasks.yaml
	TotalTasks      int    // Total tasks in this phase
	CompletedTasks  int    // Tasks with Completed status
	BlockedTasks    int    // Tasks with Blocked status
	ActionableTasks int    // Tasks with Pending or InProgress status
}

// IsComplete returns true when all tasks are Completed or Blocked (no actionable tasks remain)
func (p *PhaseInfo) IsComplete() bool {
	return p.ActionableTasks == 0
}

// GetPhaseInfo extracts phase information from tasks.yaml
// Returns a slice of PhaseInfo containing status counts for each phase
func GetPhaseInfo(tasksPath string) ([]PhaseInfo, error) {
	tasks, err := ParseTasksYAML(tasksPath)
	if err != nil {
		return nil, err
	}

	phases := make([]PhaseInfo, 0, len(tasks.Phases))

	for _, phase := range tasks.Phases {
		info := PhaseInfo{
			Number:     phase.Number,
			Title:      phase.Title,
			TotalTasks: len(phase.Tasks),
		}

		for _, task := range phase.Tasks {
			switch strings.ToLower(task.Status) {
			case "completed", "done", "complete":
				info.CompletedTasks++
			case "blocked":
				info.BlockedTasks++
			default:
				// Pending, InProgress, or unknown = actionable
				info.ActionableTasks++
			}
		}

		phases = append(phases, info)
	}

	return phases, nil
}

// IsPhaseComplete checks if a specific phase is complete (all tasks Completed or Blocked)
// Returns true when all tasks are Completed or Blocked, false otherwise
// Returns true for empty phases
func IsPhaseComplete(tasksPath string, phaseNumber int) (bool, error) {
	phases, err := GetPhaseInfo(tasksPath)
	if err != nil {
		return false, err
	}

	for _, phase := range phases {
		if phase.Number == phaseNumber {
			return phase.IsComplete(), nil
		}
	}

	// Phase not found - could be out of range
	return false, fmt.Errorf("phase %d not found in tasks.yaml", phaseNumber)
}

// GetActionablePhases returns phases that have Pending or InProgress tasks
// Filters out phases where all tasks are Completed or Blocked
// Returns phases in original order
func GetActionablePhases(tasksPath string) ([]PhaseInfo, error) {
	phases, err := GetPhaseInfo(tasksPath)
	if err != nil {
		return nil, err
	}

	actionable := make([]PhaseInfo, 0)
	for _, phase := range phases {
		if phase.ActionableTasks > 0 {
			actionable = append(actionable, phase)
		}
	}

	return actionable, nil
}

// GetFirstIncompletePhase returns the lowest phase number with incomplete tasks
// Returns the phase number and its info
// Returns 0 and nil if all phases are complete
func GetFirstIncompletePhase(tasksPath string) (int, *PhaseInfo, error) {
	phases, err := GetPhaseInfo(tasksPath)
	if err != nil {
		return 0, nil, err
	}

	for _, phase := range phases {
		if !phase.IsComplete() {
			return phase.Number, &phase, nil
		}
	}

	// All phases complete
	return 0, nil, nil
}

// GetTotalPhases returns the total number of phases in tasks.yaml
func GetTotalPhases(tasksPath string) (int, error) {
	tasks, err := ParseTasksYAML(tasksPath)
	if err != nil {
		return 0, err
	}
	return len(tasks.Phases), nil
}

// GetAllTasks returns a flat list of all tasks from all phases
func GetAllTasks(tasksPath string) ([]TaskItem, error) {
	tasks, err := ParseTasksYAML(tasksPath)
	if err != nil {
		return nil, err
	}

	var allTasks []TaskItem
	for _, phase := range tasks.Phases {
		allTasks = append(allTasks, phase.Tasks...)
	}
	return allTasks, nil
}

// GetTaskByID finds a task by its ID from a list of tasks
// Returns a pointer to the task if found, or an error if not found
// Task ID matching is case-sensitive
func GetTaskByID(tasks []TaskItem, id string) (*TaskItem, error) {
	for i := range tasks {
		if tasks[i].ID == id {
			return &tasks[i], nil
		}
	}
	return nil, fmt.Errorf("task %s not found", id)
}

// GetTasksInDependencyOrder returns tasks sorted by dependency order (topological sort)
// Tasks with no dependencies come first, followed by tasks whose dependencies are satisfied
// Returns an error if a circular dependency is detected
func GetTasksInDependencyOrder(tasks []TaskItem) ([]TaskItem, error) {
	// Build a map of task ID to task for quick lookup
	taskMap := make(map[string]*TaskItem)
	for i := range tasks {
		taskMap[tasks[i].ID] = &tasks[i]
	}

	// Track visited and currently-in-stack states for cycle detection
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	var result []TaskItem

	// DFS function for topological sort with cycle detection
	var visit func(id string) error
	visit = func(id string) error {
		if inStack[id] {
			return fmt.Errorf("circular dependency detected involving task %s", id)
		}
		if visited[id] {
			return nil
		}

		task := taskMap[id]
		if task == nil {
			// Referenced task doesn't exist - skip silently
			// (validation should have caught this earlier)
			return nil
		}

		inStack[id] = true

		// Visit all dependencies first
		for _, depID := range task.Dependencies {
			if err := visit(depID); err != nil {
				return err
			}
		}

		inStack[id] = false
		visited[id] = true
		result = append(result, *task)
		return nil
	}

	// Visit all tasks
	for _, task := range tasks {
		if err := visit(task.ID); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// ValidateTaskDependenciesMet checks if all dependencies of a task are completed
// Returns true if all dependencies have Completed status, false otherwise
// Also returns a list of unmet dependency IDs for logging/error messages
func ValidateTaskDependenciesMet(task TaskItem, tasks []TaskItem) (bool, []string) {
	if len(task.Dependencies) == 0 {
		return true, nil
	}

	// Build a map of task ID to status for quick lookup
	statusMap := make(map[string]string)
	for _, t := range tasks {
		statusMap[t.ID] = t.Status
	}

	var unmetDeps []string
	for _, depID := range task.Dependencies {
		status, exists := statusMap[depID]
		if !exists {
			// Dependency task doesn't exist - consider it unmet
			unmetDeps = append(unmetDeps, depID+" (not found)")
			continue
		}

		// Check if dependency is completed (case-insensitive)
		statusLower := strings.ToLower(status)
		if statusLower != "completed" && statusLower != "done" && statusLower != "complete" {
			unmetDeps = append(unmetDeps, depID)
		}
	}

	return len(unmetDeps) == 0, unmetDeps
}

// FormatTaskSummary formats the task stats as a human-readable summary
func FormatTaskSummary(stats *TaskStats) string {
	var sb strings.Builder

	// Task completion line
	sb.WriteString(fmt.Sprintf("  %d/%d tasks completed", stats.CompletedTasks, stats.TotalTasks))
	if stats.TotalTasks > 0 {
		sb.WriteString(fmt.Sprintf(" (%.0f%%)", stats.CompletionPercentage()))
	}
	sb.WriteString("\n")

	// Phase completion line
	sb.WriteString(fmt.Sprintf("  %d/%d task phases completed\n", stats.CompletedPhases, stats.TotalPhases))

	// Show in-progress/blocked if any
	if stats.InProgressTasks > 0 || stats.BlockedTasks > 0 {
		parts := []string{}
		if stats.InProgressTasks > 0 {
			parts = append(parts, fmt.Sprintf("%d in progress", stats.InProgressTasks))
		}
		if stats.BlockedTasks > 0 {
			parts = append(parts, fmt.Sprintf("%d blocked", stats.BlockedTasks))
		}
		sb.WriteString(fmt.Sprintf("  (%s)\n", strings.Join(parts, ", ")))
	}

	return sb.String()
}
