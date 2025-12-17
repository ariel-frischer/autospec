package validation

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// TasksValidator validates tasks.yaml artifacts.
type TasksValidator struct {
	baseValidator
}

// Type returns the artifact type.
func (v *TasksValidator) Type() ArtifactType {
	return ArtifactTypeTasks
}

// Validate validates a tasks.yaml file at the given path.
func (v *TasksValidator) Validate(path string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Parse the YAML file
	root, err := parseYAMLFile(path)
	if err != nil {
		result.AddError(&ValidationError{
			Path:    path,
			Message: fmt.Sprintf("failed to parse YAML: %v", err),
			Hint:    "Check the YAML syntax for errors",
		})
		return result
	}

	rootMapping := getRootMapping(root)
	if rootMapping == nil {
		result.AddError(&ValidationError{
			Path:    path,
			Message: "expected a YAML mapping at document root",
			Hint:    "The tasks.yaml file should start with key-value pairs, not a list or scalar",
		})
		return result
	}

	// Validate required fields
	tasksNode := validateRequiredField(rootMapping, "tasks", result)
	summaryNode := validateRequiredField(rootMapping, "summary", result)
	phasesNode := validateRequiredField(rootMapping, "phases", result)

	// Validate tasks section
	if tasksNode != nil {
		v.validateTasksSection(tasksNode, result)
	}

	// Validate summary section
	if summaryNode != nil {
		v.validateSummarySection(summaryNode, result)
	}

	// Collect all task IDs for dependency validation
	taskIDs := make(map[string]int) // task ID -> line number
	taskLines := make(map[string]int)

	// Validate phases section and collect task IDs
	if phasesNode != nil {
		v.validatePhases(phasesNode, result, taskIDs, taskLines)
	}

	// Validate dependencies after collecting all task IDs
	if phasesNode != nil && phasesNode.Kind == yaml.SequenceNode {
		v.validateAllDependencies(phasesNode, taskIDs, taskLines, result)
	}

	// Build summary if valid
	if result.Valid {
		result.Summary = v.buildSummary(rootMapping, taskIDs)
	}

	return result
}

// validateTasksSection validates the tasks header section.
func (v *TasksValidator) validateTasksSection(node *yaml.Node, result *ValidationResult) {
	if !validateFieldType(node, "tasks", yaml.MappingNode, "object", result) {
		return
	}

	// Required field: branch
	validateRequiredField(node, "branch", result)
}

// validateSummarySection validates the summary section.
func (v *TasksValidator) validateSummarySection(node *yaml.Node, result *ValidationResult) {
	if !validateFieldType(node, "summary", yaml.MappingNode, "object", result) {
		return
	}
	// Summary fields are optional, just validate types if present
}

// validatePhases validates the phases array and collects task IDs.
func (v *TasksValidator) validatePhases(node *yaml.Node, result *ValidationResult, taskIDs map[string]int, taskLines map[string]int) {
	if !validateFieldType(node, "phases", yaml.SequenceNode, "array", result) {
		return
	}

	for i, phaseNode := range node.Content {
		path := fmt.Sprintf("phases[%d]", i)
		v.validatePhase(phaseNode, path, result, taskIDs, taskLines)
	}
}

// validatePhase validates a single phase and its tasks.
func (v *TasksValidator) validatePhase(node *yaml.Node, path string, result *ValidationResult, taskIDs map[string]int, taskLines map[string]int) {
	if node.Kind != yaml.MappingNode {
		result.AddError(&ValidationError{
			Path:     path,
			Line:     getNodeLine(node),
			Message:  fmt.Sprintf("wrong type for '%s'", path),
			Expected: "object",
			Actual:   nodeKindToString(node.Kind),
		})
		return
	}

	// Required fields: number, title, tasks
	validateRequiredField(node, "number", result)
	validateRequiredField(node, "title", result)

	// Validate tasks array
	tasksNode := findNode(node, "tasks")
	if tasksNode == nil {
		result.AddError(&ValidationError{
			Path:    path + ".tasks",
			Line:    getNodeLine(node),
			Message: "missing required field: tasks",
			Hint:    "Add a 'tasks' field with a list of tasks",
		})
		return
	}

	if !validateFieldType(tasksNode, path+".tasks", yaml.SequenceNode, "array", result) {
		return
	}

	for j, taskNode := range tasksNode.Content {
		taskPath := fmt.Sprintf("%s.tasks[%d]", path, j)
		v.validateTask(taskNode, taskPath, result, taskIDs, taskLines)
	}
}

// validateTask validates a single task.
func (v *TasksValidator) validateTask(node *yaml.Node, path string, result *ValidationResult, taskIDs map[string]int, taskLines map[string]int) {
	if node.Kind != yaml.MappingNode {
		result.AddError(&ValidationError{
			Path:     path,
			Line:     getNodeLine(node),
			Message:  fmt.Sprintf("wrong type for '%s'", path),
			Expected: "object",
			Actual:   nodeKindToString(node.Kind),
		})
		return
	}

	// Required fields
	idNode := findNode(node, "id")
	if idNode == nil {
		result.AddError(&ValidationError{
			Path:    path + ".id",
			Line:    getNodeLine(node),
			Message: "missing required field: id",
			Hint:    "Add an 'id' field with format 'TNNN' (e.g., 'T001')",
		})
	} else {
		taskID := idNode.Value
		// Check for duplicate task IDs
		if existingLine, exists := taskIDs[taskID]; exists {
			result.AddError(&ValidationError{
				Path:    path + ".id",
				Line:    getNodeLine(idNode),
				Message: fmt.Sprintf("duplicate task ID: %s (first defined at line %d)", taskID, existingLine),
				Hint:    "Each task must have a unique ID",
			})
		} else {
			taskIDs[taskID] = getNodeLine(idNode)
			taskLines[taskID] = getNodeLine(node)
		}
	}

	validateRequiredField(node, "title", result)

	// Validate status enum
	statusNode := findNode(node, "status")
	if statusNode == nil {
		result.AddError(&ValidationError{
			Path:    path + ".status",
			Line:    getNodeLine(node),
			Message: "missing required field: status",
			Hint:    "Add a 'status' field with one of: Pending, InProgress, Completed, Blocked",
		})
	} else {
		validateEnumValue(statusNode, path+".status", []string{"Pending", "InProgress", "Completed", "Blocked"}, result)
	}

	// Validate type enum
	typeNode := findNode(node, "type")
	if typeNode == nil {
		result.AddError(&ValidationError{
			Path:    path + ".type",
			Line:    getNodeLine(node),
			Message: "missing required field: type",
			Hint:    "Add a 'type' field with one of: setup, implementation, test, documentation, refactor",
		})
	} else {
		validateEnumValue(typeNode, path+".type", []string{"setup", "implementation", "test", "documentation", "refactor"}, result)
	}

	// dependencies should be an array if present
	depsNode := findNode(node, "dependencies")
	if depsNode != nil {
		validateFieldType(depsNode, path+".dependencies", yaml.SequenceNode, "array", result)
	}

	// acceptance_criteria should be an array if present
	criteriaNode := findNode(node, "acceptance_criteria")
	if criteriaNode != nil {
		validateFieldType(criteriaNode, path+".acceptance_criteria", yaml.SequenceNode, "array", result)
	}

	// Validate blocked_reason for blocked tasks
	v.validateBlockedReason(node, path, statusNode, result)
}

// validateBlockedReason checks that blocked tasks have a reason.
func (v *TasksValidator) validateBlockedReason(node *yaml.Node, path string, statusNode *yaml.Node, result *ValidationResult) {
	if statusNode == nil || statusNode.Value != "Blocked" {
		return
	}

	blockedReasonNode := findNode(node, "blocked_reason")
	if blockedReasonNode == nil || blockedReasonNode.Value == "" {
		result.AddWarning(&ValidationWarning{
			Path:    path + ".blocked_reason",
			Line:    getNodeLine(statusNode),
			Message: "blocked task is missing a blocked_reason",
			Hint:    "Add a blocked_reason field to document why this task is blocked",
		})
	}
}

// validateAllDependencies validates all task dependencies after collecting task IDs.
func (v *TasksValidator) validateAllDependencies(phasesNode *yaml.Node, taskIDs map[string]int, taskLines map[string]int, result *ValidationResult) {
	// Build dependency graph for circular dependency detection
	deps := make(map[string][]string) // task ID -> list of dependency IDs

	for i, phaseNode := range phasesNode.Content {
		if phaseNode.Kind != yaml.MappingNode {
			continue
		}

		tasksNode := findNode(phaseNode, "tasks")
		if tasksNode == nil || tasksNode.Kind != yaml.SequenceNode {
			continue
		}

		for j, taskNode := range tasksNode.Content {
			if taskNode.Kind != yaml.MappingNode {
				continue
			}

			idNode := findNode(taskNode, "id")
			if idNode == nil {
				continue
			}
			taskID := idNode.Value
			taskPath := fmt.Sprintf("phases[%d].tasks[%d]", i, j)

			depsNode := findNode(taskNode, "dependencies")
			if depsNode == nil || depsNode.Kind != yaml.SequenceNode {
				deps[taskID] = []string{}
				continue
			}

			taskDeps := []string{}
			for k, depNode := range depsNode.Content {
				if depNode.Kind != yaml.ScalarNode {
					continue
				}
				depID := depNode.Value
				taskDeps = append(taskDeps, depID)

				// Check for self-reference
				if depID == taskID {
					result.AddError(&ValidationError{
						Path:    fmt.Sprintf("%s.dependencies[%d]", taskPath, k),
						Line:    getNodeLine(depNode),
						Message: fmt.Sprintf("task '%s' cannot depend on itself", taskID),
						Hint:    "Remove the self-reference from the dependencies list",
					})
					continue
				}

				// Check if dependency exists
				if _, exists := taskIDs[depID]; !exists {
					result.AddError(&ValidationError{
						Path:    fmt.Sprintf("%s.dependencies[%d]", taskPath, k),
						Line:    getNodeLine(depNode),
						Message: fmt.Sprintf("invalid dependency: task '%s' depends on '%s' which does not exist", taskID, depID),
						Hint:    fmt.Sprintf("Either create a task with ID '%s' or remove this dependency", depID),
					})
				}
			}
			deps[taskID] = taskDeps
		}
	}

	// Detect circular dependencies
	v.detectCircularDependencies(deps, taskLines, result)
}

// detectCircularDependencies detects circular dependencies in the task graph.
func (v *TasksValidator) detectCircularDependencies(deps map[string][]string, taskLines map[string]int, result *ValidationResult) {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(taskID string, path []string) []string
	dfs = func(taskID string, path []string) []string {
		visited[taskID] = true
		recStack[taskID] = true
		path = append(path, taskID)

		for _, depID := range deps[taskID] {
			if !visited[depID] {
				if cycle := dfs(depID, path); cycle != nil {
					return cycle
				}
			} else if recStack[depID] {
				// Found a cycle - build the cycle path
				cycleStart := -1
				for i, id := range path {
					if id == depID {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle := append(path[cycleStart:], depID)
					return cycle
				}
				return append(path, depID)
			}
		}

		recStack[taskID] = false
		return nil
	}

	for taskID := range deps {
		if !visited[taskID] {
			if cycle := dfs(taskID, []string{}); cycle != nil {
				// Format cycle path
				cyclePath := ""
				for i, id := range cycle {
					if i > 0 {
						cyclePath += " -> "
					}
					cyclePath += id
				}

				result.AddError(&ValidationError{
					Path:    "phases",
					Line:    taskLines[cycle[0]],
					Message: fmt.Sprintf("circular dependency detected: %s", cyclePath),
					Hint:    "Remove one of the dependencies to break the cycle",
				})
				return // Only report the first cycle found
			}
		}
	}
}

// buildSummary builds the summary for a valid tasks artifact.
func (v *TasksValidator) buildSummary(root *yaml.Node, taskIDs map[string]int) *ArtifactSummary {
	summary := &ArtifactSummary{
		Type:   ArtifactTypeTasks,
		Counts: make(map[string]int),
	}

	// Count phases
	phasesNode := findNode(root, "phases")
	if phasesNode != nil && phasesNode.Kind == yaml.SequenceNode {
		summary.Counts["phases"] = len(phasesNode.Content)
	}

	// Count tasks
	summary.Counts["total_tasks"] = len(taskIDs)

	// Count tasks by status
	pending := 0
	inProgress := 0
	completed := 0
	blocked := 0

	if phasesNode != nil && phasesNode.Kind == yaml.SequenceNode {
		for _, phaseNode := range phasesNode.Content {
			tasksNode := findNode(phaseNode, "tasks")
			if tasksNode == nil || tasksNode.Kind != yaml.SequenceNode {
				continue
			}
			for _, taskNode := range tasksNode.Content {
				statusNode := findNode(taskNode, "status")
				if statusNode == nil {
					continue
				}
				switch statusNode.Value {
				case "Pending":
					pending++
				case "InProgress":
					inProgress++
				case "Completed":
					completed++
				case "Blocked":
					blocked++
				}
			}
		}
	}

	summary.Counts["pending"] = pending
	summary.Counts["in_progress"] = inProgress
	summary.Counts["completed"] = completed
	summary.Counts["blocked"] = blocked
	summary.Counts["blocked_without_reason"] = countBlockedWithoutReason(phasesNode)

	return summary
}

// countBlockedWithoutReason counts blocked tasks that are missing a blocked_reason.
func countBlockedWithoutReason(phasesNode *yaml.Node) int {
	if phasesNode == nil || phasesNode.Kind != yaml.SequenceNode {
		return 0
	}

	count := 0
	for _, phaseNode := range phasesNode.Content {
		tasksNode := findNode(phaseNode, "tasks")
		if tasksNode == nil || tasksNode.Kind != yaml.SequenceNode {
			continue
		}
		for _, taskNode := range tasksNode.Content {
			statusNode := findNode(taskNode, "status")
			if statusNode == nil || statusNode.Value != "Blocked" {
				continue
			}
			blockedReasonNode := findNode(taskNode, "blocked_reason")
			if blockedReasonNode == nil || blockedReasonNode.Value == "" {
				count++
			}
		}
	}
	return count
}
