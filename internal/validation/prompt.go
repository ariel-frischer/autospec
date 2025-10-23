package validation

import (
	"fmt"
	"strings"
)

// ListIncompletePhasesWithTasks returns a formatted string of incomplete phases and their tasks
func ListIncompletePhasesWithTasks(phases []Phase) string {
	var builder strings.Builder

	for _, phase := range phases {
		if !phase.IsComplete() {
			builder.WriteString(fmt.Sprintf("\n## %s (%d/%d tasks complete)\n",
				phase.Name, phase.CheckedTasks, phase.TotalTasks))

			// List first 5 unchecked tasks
			uncheckedCount := 0
			for _, task := range phase.Tasks {
				if !task.Checked {
					builder.WriteString(fmt.Sprintf("- [ ] %s\n", task.Description))
					uncheckedCount++
					if uncheckedCount >= 5 {
						remaining := phase.UncheckedTasks() - uncheckedCount
						if remaining > 0 {
							builder.WriteString(fmt.Sprintf("... and %d more unchecked tasks\n", remaining))
						}
						break
					}
				}
			}
		}
	}

	return builder.String()
}

// GenerateContinuationPrompt creates a context-aware prompt for incomplete work
func GenerateContinuationPrompt(specDir string, phase string, phases []Phase) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("The %s phase is incomplete. ", phase))

	totalUnchecked := 0
	for _, p := range phases {
		totalUnchecked += p.UncheckedTasks()
	}

	builder.WriteString(fmt.Sprintf("%d task(s) remain unchecked.\n", totalUnchecked))
	builder.WriteString(ListIncompletePhasesWithTasks(phases))

	builder.WriteString(fmt.Sprintf("\nPlease continue working on the implementation for %s.\n", specDir))
	builder.WriteString("Review the tasks.md file and complete the remaining tasks.\n")

	return builder.String()
}
