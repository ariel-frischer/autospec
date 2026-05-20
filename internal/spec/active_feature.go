package spec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// SelectionSource identifies why a feature directory was chosen.
type SelectionSource string

const (
	// SelectionSourceExplicit indicates a command argument or --spec flag selected the feature.
	SelectionSourceExplicit SelectionSource = "explicit"
	// SelectionSourcePersisted indicates project-local active feature state selected the feature.
	SelectionSourcePersisted SelectionSource = "persisted"
	// SelectionSourceGitBranch indicates the current git branch selected the feature.
	SelectionSourceGitBranch SelectionSource = "git_branch"
	// SelectionSourceFallback indicates autospec used the most recent feature directory.
	SelectionSourceFallback SelectionSource = "fallback"
)

// ActiveFeatureRequest contains the inputs needed to resolve an active feature.
type ActiveFeatureRequest struct {
	SpecsDir           string
	ExplicitIdentifier string
	StateDir           string
	RequiredArtifact   string
	AllowMissingSpec   bool
}

// ActiveFeatureResult reports the resolved feature and the source that selected it.
type ActiveFeatureResult struct {
	Metadata *Metadata
	Source   SelectionSource
}

// ResolveActiveFeature resolves a feature directory using explicit, persisted,
// then current branch or fallback selection.
func ResolveActiveFeature(req ActiveFeatureRequest) (*ActiveFeatureResult, error) {
	if req.SpecsDir == "" {
		return nil, fmt.Errorf("resolving active feature: specs directory is required")
	}
	if req.ExplicitIdentifier != "" {
		return resolveExplicitFeature(req)
	}
	if req.StateDir != "" {
		if result, found, err := resolvePersistedFeature(req); err != nil || found {
			return result, err
		}
	}
	return resolveCurrentFeature(req)
}

func resolveExplicitFeature(req ActiveFeatureRequest) (*ActiveFeatureResult, error) {
	metadata, err := GetSpecMetadata(req.SpecsDir, req.ExplicitIdentifier)
	if err != nil {
		return nil, fmt.Errorf("resolving active feature from explicit selection %q: %w", req.ExplicitIdentifier, err)
	}
	metadata.Detection = DetectionExplicit
	if err := requireArtifact(metadata.Directory, req.RequiredArtifact); err != nil {
		return nil, fmt.Errorf("resolving active feature from explicit selection %q: %w", req.ExplicitIdentifier, err)
	}
	return &ActiveFeatureResult{Metadata: metadata, Source: SelectionSourceExplicit}, nil
}

func resolvePersistedFeature(req ActiveFeatureRequest) (*ActiveFeatureResult, bool, error) {
	state, found, err := LoadActiveFeatureState(req.StateDir)
	if err != nil {
		return nil, false, fmt.Errorf("resolving active feature from persisted state: %w", err)
	}
	if !found {
		return nil, false, nil
	}
	metadata, err := validateActiveFeatureState(req.SpecsDir, state, !req.AllowMissingSpec)
	if err != nil {
		if errors.Is(err, errPersistedFeatureDirectoryNotExist) {
			return nil, false, nil
		}
		return nil, true, fmt.Errorf("resolving active feature from persisted state: %w", err)
	}
	if err := requireArtifact(metadata.Directory, req.RequiredArtifact); err != nil {
		return nil, true, fmt.Errorf("resolving active feature from persisted state: %w", err)
	}
	return &ActiveFeatureResult{Metadata: metadata, Source: SelectionSourcePersisted}, true, nil
}

func resolveCurrentFeature(req ActiveFeatureRequest) (*ActiveFeatureResult, error) {
	metadata, err := DetectCurrentSpec(req.SpecsDir)
	if err != nil {
		return nil, fmt.Errorf("resolving active feature from git branch or fallback lookup: %w", err)
	}
	if err := requireArtifact(metadata.Directory, req.RequiredArtifact); err != nil {
		return nil, fmt.Errorf("resolving active feature from %s lookup: %w", sourceFromDetection(metadata.Detection), err)
	}
	return &ActiveFeatureResult{Metadata: metadata, Source: sourceFromDetection(metadata.Detection)}, nil
}

func sourceFromDetection(detection DetectionMethod) SelectionSource {
	if detection == DetectionGitBranch {
		return SelectionSourceGitBranch
	}
	return SelectionSourceFallback
}

func requireArtifact(featureDir, artifact string) error {
	if artifact == "" {
		return nil
	}
	artifactPath := filepath.Join(featureDir, artifact)
	info, err := os.Stat(artifactPath)
	if err != nil {
		return fmt.Errorf("missing required artifact %s in %s: %w", artifact, featureDir, err)
	}
	if info.IsDir() {
		return fmt.Errorf("missing required artifact %s in %s: path is a directory", artifact, featureDir)
	}
	return nil
}
