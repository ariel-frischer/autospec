package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const activeProfileStateFileName = "active-profile.yaml"

type ActiveProfileState struct {
	ProfileName string    `yaml:"profile_name"`
	SetAt       time.Time `yaml:"set_at"`
}

func ActiveProfileStatePath(userConfigDir string) string {
	return filepath.Join(userConfigDir, "state", activeProfileStateFileName)
}

func LoadActiveProfileFrom(userConfigDir string) (string, error) {
	data, err := os.ReadFile(ActiveProfileStatePath(userConfigDir))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("reading active profile state: %w", err)
	}
	var state ActiveProfileState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return "", fmt.Errorf("parsing active profile state: %w", err)
	}
	if state.ProfileName == "" {
		return "", nil
	}
	if err := ValidateProfileName(state.ProfileName); err != nil {
		return "", fmt.Errorf("invalid active profile state: %w", err)
	}
	return state.ProfileName, nil
}

func SetActiveProfileFrom(userConfigDir, name string) error {
	if err := ValidateProfileName(name); err != nil {
		return err
	}
	if !fileExists(ProfilePath(userConfigDir, name)) {
		return fmt.Errorf("profile %q does not exist", name)
	}
	data, err := yaml.Marshal(&ActiveProfileState{ProfileName: name, SetAt: time.Now().UTC()})
	if err != nil {
		return fmt.Errorf("marshaling active profile state: %w", err)
	}
	if err := writeAtomically(ActiveProfileStatePath(userConfigDir), data); err != nil {
		return fmt.Errorf("saving active profile state: %w", err)
	}
	return nil
}

func ActiveProfile() (string, error) {
	dir, err := UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting user config directory: %w", err)
	}
	return LoadActiveProfileFrom(dir)
}

func SetActiveProfile(name string) error {
	dir, err := UserConfigDir()
	if err != nil {
		return fmt.Errorf("getting user config directory: %w", err)
	}
	return SetActiveProfileFrom(dir, name)
}
