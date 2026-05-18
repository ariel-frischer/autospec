package cli

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/spec"
)

func resolveCLIActiveFeature(cfg *config.Configuration, requiredArtifact string) (*spec.ActiveFeatureResult, error) {
	result, err := spec.ResolveActiveFeature(spec.ActiveFeatureRequest{
		SpecsDir:         cfg.SpecsDir,
		StateDir:         cfg.StateDir,
		RequiredArtifact: requiredArtifact,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to resolve active feature: %w", err)
	}
	return result, nil
}

func printCLIActiveFeature(result *spec.ActiveFeatureResult) {
	if result == nil || result.Metadata == nil {
		return
	}
	PrintSpecInfo(result.Metadata)
	fmt.Printf("  active feature: %s (source: %s)\n", result.Metadata.Directory, result.Source)
}
