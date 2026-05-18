package spec

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const activeFeatureStateFilename = "active-feature.yaml"

var errPersistedFeatureDirectoryNotExist = errors.New("persisted active feature directory does not exist")

// ActiveFeatureState is the project-local persisted feature selection.
type ActiveFeatureState struct {
	FeatureDirectory string `yaml:"feature_directory"`
	Source           string `yaml:"source"`
	UpdatedAt        string `yaml:"updated_at"`
}

// SaveActiveFeatureState writes project-local active feature state.
func SaveActiveFeatureState(stateDir, featureDirectory, source string) (*ActiveFeatureState, error) {
	state := &ActiveFeatureState{
		FeatureDirectory: featureDirectory,
		Source:           source,
		UpdatedAt:        time.Now().UTC().Format(time.RFC3339),
	}
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating active feature state directory: %w", err)
	}
	data, err := yaml.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("serializing active feature state: %w", err)
	}
	if err := os.WriteFile(activeFeatureStatePath(stateDir), data, 0o644); err != nil {
		return nil, fmt.Errorf("writing active feature state: %w", err)
	}
	return state, nil
}

// LoadActiveFeatureState loads project-local active feature state if present.
func LoadActiveFeatureState(stateDir string) (*ActiveFeatureState, bool, error) {
	data, err := os.ReadFile(activeFeatureStatePath(stateDir))
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("reading active feature state: %w", err)
	}
	var state ActiveFeatureState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, false, fmt.Errorf("parsing active feature state: %w", err)
	}
	return &state, true, nil
}

// ClearActiveFeatureState removes project-local active feature state.
func ClearActiveFeatureState(stateDir string) error {
	if err := os.Remove(activeFeatureStatePath(stateDir)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clearing active feature state: %w", err)
	}
	return nil
}

// ValidateActiveFeatureState resolves and validates persisted active feature state.
func ValidateActiveFeatureState(specsDir string, state *ActiveFeatureState) (*Metadata, error) {
	if state == nil {
		return nil, fmt.Errorf("active feature state is nil")
	}
	directory, err := resolvePersistedFeatureDirectory(specsDir, state.FeatureDirectory)
	if err != nil {
		return nil, err
	}
	if err := validateFeatureDirectory(directory); err != nil {
		return nil, err
	}
	return metadataFromDirectory(directory)
}

func activeFeatureStatePath(stateDir string) string {
	return filepath.Join(stateDir, activeFeatureStateFilename)
}

func resolvePersistedFeatureDirectory(specsDir, value string) (string, error) {
	clean := filepath.Clean(value)
	if clean == "." || clean == "" {
		return "", fmt.Errorf("persisted active feature directory is empty")
	}
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("persisted active feature directory %q escapes specs directory", value)
	}
	candidate := filepath.Join(specsDir, clean)
	if strings.HasPrefix(clean, filepath.Base(specsDir)+string(filepath.Separator)) {
		candidate = filepath.Join(filepath.Dir(specsDir), clean)
	}
	if err := ensureWithinSpecsDir(specsDir, candidate, value); err != nil {
		return "", err
	}
	return candidate, nil
}

func ensureWithinSpecsDir(specsDir, candidate, value string) error {
	absSpecs, err := filepath.Abs(specsDir)
	if err != nil {
		return fmt.Errorf("resolving specs directory: %w", err)
	}
	absCandidate, err := filepath.Abs(candidate)
	if err != nil {
		return fmt.Errorf("resolving persisted active feature directory: %w", err)
	}
	rel, err := filepath.Rel(absSpecs, absCandidate)
	if err != nil {
		return fmt.Errorf("checking persisted active feature directory: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("persisted active feature directory %q escapes specs directory", value)
	}
	return nil
}

func validateFeatureDirectory(directory string) error {
	info, err := os.Stat(directory)
	if os.IsNotExist(err) {
		return fmt.Errorf("%w: %s: %w", errPersistedFeatureDirectoryNotExist, directory, err)
	}
	if err != nil {
		return fmt.Errorf("checking persisted active feature directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("persisted active feature path is not a directory: %s", directory)
	}
	if _, err := os.Stat(filepath.Join(directory, "spec.yaml")); err != nil {
		return fmt.Errorf("persisted active feature directory missing spec.yaml: %w", err)
	}
	return nil
}

func metadataFromDirectory(directory string) (*Metadata, error) {
	baseName := filepath.Base(directory)
	match := specDirPattern.FindStringSubmatch(baseName)
	if match == nil {
		return nil, fmt.Errorf("could not parse persisted active feature directory name: %s", baseName)
	}
	return &Metadata{
		Number:    match[1],
		Name:      match[2],
		Directory: directory,
		Detection: DetectionFallbackRecent,
	}, nil
}
