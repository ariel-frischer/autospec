package stages

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/spec"
)

func resolveStageFeature(cfg *config.Configuration, explicit, requiredArtifact string) (*spec.ActiveFeatureResult, error) {
	result, err := spec.ResolveActiveFeature(spec.ActiveFeatureRequest{
		SpecsDir:           cfg.SpecsDir,
		StateDir:           cfg.StateDir,
		ExplicitIdentifier: explicit,
		RequiredArtifact:   requiredArtifact,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve active feature: %w", err)
	}
	return result, nil
}

func printActiveFeature(result *spec.ActiveFeatureResult) {
	if result == nil || result.Metadata == nil {
		return
	}
	fmt.Println(result.Metadata.FormatInfo())
	fmt.Printf("  active feature: %s (source: %s)\n", result.Metadata.Directory, result.Source)
}
