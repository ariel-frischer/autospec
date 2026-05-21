package workflow

import "github.com/ariel-frischer/autospec/internal/config"

const (
	ModelSourceCLI     ModelSelectionSource = "cli_model"
	ModelSourceConfig  ModelSelectionSource = "config_model"
	ModelSourceDefault ModelSelectionSource = "default"
)

// ModelSelectionSource identifies the setting that supplied a workflow model.
type ModelSelectionSource string

// WorkflowModelSelection is the resolved model value for one workflow stage.
type WorkflowModelSelection struct {
	Value  string
	Source ModelSelectionSource
	Stage  Stage
	Agent  string
}

// ModelSelectionInput groups model-selection inputs for a workflow stage.
type ModelSelectionInput struct {
	Agent string
	Stage Stage
}

// ResolveWorkflowModelSelection returns the model selected for one workflow stage.
func ResolveWorkflowModelSelection(cfg config.Configuration, input ModelSelectionInput) WorkflowModelSelection {
	selection := defaultModelSelection(input)
	if !agentSupportsWorkflowModel(input.Agent) {
		return selection
	}
	if cfg.ModelOverride != "" {
		return selection.with(cfg.ModelOverride, ModelSourceCLI)
	}
	if cfg.Model != "" {
		return selection.with(cfg.Model, ModelSourceConfig)
	}
	return selection
}

func defaultModelSelection(input ModelSelectionInput) WorkflowModelSelection {
	return WorkflowModelSelection{
		Source: ModelSourceDefault,
		Stage:  input.Stage,
		Agent:  input.Agent,
	}
}

func (s WorkflowModelSelection) with(value string, source ModelSelectionSource) WorkflowModelSelection {
	s.Value = value
	s.Source = source
	return s
}

func agentSupportsWorkflowModel(agent string) bool {
	switch agent {
	case "claude", "codex", "opencode":
		return true
	default:
		return false
	}
}
