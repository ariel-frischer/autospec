package workflow

import "github.com/ariel-frischer/autospec/internal/config"

// OpenCodeModelForStage returns the OpenCode model configured for a stage.
func OpenCodeModelForStage(cfg config.OpenCodeConfig, stage Stage) string {
	if cfg.ModelOverride != "" {
		return cfg.ModelOverride
	}

	model := openCodeStageModel(cfg.Models, stage)
	if model != "" {
		return model
	}
	return cfg.Model
}

func openCodeStageModel(models config.OpenCodeStageModels, stage Stage) string {
	switch stage {
	case StageSpecify:
		return models.Specify
	case StagePlan:
		return models.Plan
	case StageTasks:
		return models.Tasks
	case StageImplement:
		return models.Implement
	case StageConstitution:
		return models.Constitution
	case StageClarify:
		return models.Clarify
	case StageChecklist:
		return models.Checklist
	case StageAnalyze:
		return models.Analyze
	default:
		return ""
	}
}
