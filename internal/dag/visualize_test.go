package dag

import (
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependencyGraph_RenderASCII(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		tasks       []validation.TaskItem
		wantContain []string
		wantExclude []string
	}{
		"single wave": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
				{ID: "T002", Dependencies: []string{}},
			},
			wantContain: []string{
				"Wave 1",
				"[T001]",
				"[T002]",
				"2 tasks",
			},
			wantExclude: []string{"Wave 2"},
		},
		"multiple waves": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
				{ID: "T002", Dependencies: []string{"T001"}},
				{ID: "T003", Dependencies: []string{"T002"}},
			},
			wantContain: []string{
				"Wave 1",
				"Wave 2",
				"Wave 3",
				"[T001]",
				"[T002]",
				"[T003]",
				"v", // Wave connector (downward arrow)
			},
		},
		"summary stats": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
				{ID: "T002", Dependencies: []string{"T001"}},
			},
			wantContain: []string{
				"Summary:",
				"Total Waves: 2",
				"Total Tasks: 2",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g, err := BuildFromTasks(tt.tasks)
			require.NoError(t, err)
			_, err = g.ComputeWaves()
			require.NoError(t, err)

			output := g.RenderASCII()

			for _, want := range tt.wantContain {
				assert.Contains(t, output, want, "output should contain %q", want)
			}

			for _, exclude := range tt.wantExclude {
				assert.NotContains(t, output, exclude, "output should not contain %q", exclude)
			}
		})
	}
}

func TestDependencyGraph_RenderASCII_NoWaves(t *testing.T) {
	t.Parallel()

	g := NewDependencyGraph()
	output := g.RenderASCII()
	assert.Contains(t, output, "No waves computed")
}

func TestDependencyGraph_RenderCompact(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		tasks  []validation.TaskItem
		expect string
	}{
		"single wave two tasks": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
				{ID: "T002", Dependencies: []string{}},
			},
			expect: "Wave 1: [T001, T002]",
		},
		"three waves linear": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
				{ID: "T002", Dependencies: []string{"T001"}},
				{ID: "T003", Dependencies: []string{"T002"}},
			},
			expect: "Wave 1: [T001] -> Wave 2: [T002] -> Wave 3: [T003]",
		},
		"diamond pattern": {
			tasks: []validation.TaskItem{
				{ID: "T001", Dependencies: []string{}},
				{ID: "T002", Dependencies: []string{"T001"}},
				{ID: "T003", Dependencies: []string{"T001"}},
				{ID: "T004", Dependencies: []string{"T002", "T003"}},
			},
			expect: "Wave 1: [T001] -> Wave 2: [T002, T003] -> Wave 3: [T004]",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			g, err := BuildFromTasks(tt.tasks)
			require.NoError(t, err)
			_, err = g.ComputeWaves()
			require.NoError(t, err)

			output := g.RenderCompact()
			assert.Equal(t, tt.expect, output)
		})
	}
}

func TestDependencyGraph_RenderCompact_NoWaves(t *testing.T) {
	t.Parallel()

	g := NewDependencyGraph()
	output := g.RenderCompact()
	assert.Equal(t, "No waves computed", output)
}

func TestDependencyGraph_RenderDetailed(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{"T001"}},
		{ID: "T003", Dependencies: []string{"T001"}},
		{ID: "T004", Dependencies: []string{"T002", "T003"}},
	}

	g, err := BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	output := g.RenderDetailed()

	// Check structure
	assert.Contains(t, output, "Detailed Task Execution Plan")
	assert.Contains(t, output, "Wave 1:")
	assert.Contains(t, output, "Wave 2:")
	assert.Contains(t, output, "Wave 3:")

	// Check dependency info
	assert.Contains(t, output, "Depends on:")
	assert.Contains(t, output, "Blocks:")
}

func TestDependencyGraph_RenderProgress(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{}},
		{ID: "T003", Dependencies: []string{}},
	}

	g, err := BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	// All pending
	output := g.RenderProgress(1)
	assert.Contains(t, output, "Wave 1:")
	assert.Contains(t, output, "o") // Pending symbol

	// Set some statuses
	_ = g.SetNodeStatus("T001", StatusRunning)
	_ = g.SetNodeStatus("T002", StatusCompleted)
	_ = g.SetNodeStatus("T003", StatusFailed)

	output = g.RenderProgress(1)
	assert.Contains(t, output, "T001 *") // Running
	assert.Contains(t, output, "T002 +") // Completed
	assert.Contains(t, output, "T003 x") // Failed
}

func TestDependencyGraph_RenderProgress_InvalidWave(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
	}

	g, err := BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	assert.Empty(t, g.RenderProgress(0))
	assert.Empty(t, g.RenderProgress(99))
}

func TestASCII_OnlyCharacters(t *testing.T) {
	t.Parallel()

	tasks := []validation.TaskItem{
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{"T001"}},
		{ID: "T003", Dependencies: []string{"T001"}},
		{ID: "T004", Dependencies: []string{"T002", "T003"}},
	}

	g, err := BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	outputs := []string{
		g.RenderASCII(),
		g.RenderCompact(),
		g.RenderDetailed(),
		g.RenderProgress(1),
	}

	for _, output := range outputs {
		// Check that all characters are ASCII (< 128)
		for _, char := range output {
			assert.Less(t, char, rune(128), "non-ASCII character found: %c (code %d)", char, char)
		}
	}
}

func TestGetStatusSymbol(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status TaskStatus
		want   string
	}{
		"pending":   {status: StatusPending, want: "o"},
		"running":   {status: StatusRunning, want: "*"},
		"completed": {status: StatusCompleted, want: "+"},
		"failed":    {status: StatusFailed, want: "x"},
		"skipped":   {status: StatusSkipped, want: "-"},
		"unknown":   {status: TaskStatus(99), want: "?"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, getStatusSymbol(tt.status))
		})
	}
}

func TestRenderASCII_EmptyWave(t *testing.T) {
	t.Parallel()

	g := NewDependencyGraph()
	g.waves = []ExecutionWave{
		{Number: 1, TaskIDs: []string{}},
	}

	output := g.RenderASCII()
	assert.Contains(t, output, "(empty)")
}

func TestVisualization_ConsistentOrdering(t *testing.T) {
	t.Parallel()

	// Task IDs intentionally out of order
	tasks := []validation.TaskItem{
		{ID: "T003", Dependencies: []string{}},
		{ID: "T001", Dependencies: []string{}},
		{ID: "T002", Dependencies: []string{}},
	}

	g, err := BuildFromTasks(tasks)
	require.NoError(t, err)
	_, err = g.ComputeWaves()
	require.NoError(t, err)

	// Multiple renders should produce the same output
	output1 := g.RenderASCII()
	output2 := g.RenderASCII()
	assert.Equal(t, output1, output2)

	// Task IDs should be sorted in output
	assert.True(t, strings.Index(output1, "T001") < strings.Index(output1, "T002"))
	assert.True(t, strings.Index(output1, "T002") < strings.Index(output1, "T003"))
}
