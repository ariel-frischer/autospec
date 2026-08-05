package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"gopkg.in/yaml.v3"
)

const profileDirectoryName = "profiles"

var profileNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// ValidateProfileName validates a profile name used as a filesystem basename.
func ValidateProfileName(name string) error {
	if !profileNamePattern.MatchString(name) {
		return fmt.Errorf("invalid profile name %q: use 1-64 letters, numbers, hyphens, or underscores", name)
	}
	return nil
}

// ProfilesDir returns the profile directory below an autospec user config directory.
func ProfilesDir(userConfigDir string) string {
	return filepath.Join(userConfigDir, profileDirectoryName)
}

// ProfilePath returns the path for a named profile below an autospec user config directory.
func ProfilePath(userConfigDir, name string) string {
	return filepath.Join(ProfilesDir(userConfigDir), name+".yml")
}

// ProjectProfilesDir returns the project profile directory.
func ProjectProfilesDir() string {
	return filepath.Join(ProjectConfigDir(), profileDirectoryName)
}

// ProjectProfilePath returns the path for a named project profile.
func ProjectProfilePath(name string) string {
	return filepath.Join(ProjectProfilesDir(), name+".yml")
}

// ProfileLocations returns the user and project files searched for a profile.
func ProfileLocations(name string) (userPath, projectPath string, err error) {
	if err := ValidateProfileName(name); err != nil {
		return "", "", err
	}
	userDir, err := UserConfigDir()
	if err != nil {
		return "", "", fmt.Errorf("getting user config directory: %w", err)
	}
	return ProfilePath(userDir, name), ProjectProfilePath(name), nil
}

// ProfileExists reports whether a named profile exists in user or project storage.
func ProfileExists(name string) (bool, error) {
	userPath, projectPath, err := ProfileLocations(name)
	if err != nil {
		return false, err
	}
	for _, path := range []string{userPath, projectPath} {
		if _, err := os.Stat(path); err == nil {
			return true, nil
		} else if !os.IsNotExist(err) {
			return false, fmt.Errorf("checking profile %q: %w", name, err)
		}
	}
	return false, nil
}

// ListProfilesFrom lists profile names stored below userConfigDir.
func ListProfilesFrom(userConfigDir string) ([]string, error) {
	entries, err := os.ReadDir(ProfilesDir(userConfigDir))
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("reading profiles directory: %w", err)
	}

	profiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yml" {
			continue
		}
		name := entry.Name()[:len(entry.Name())-len(filepath.Ext(entry.Name()))]
		if ValidateProfileName(name) == nil {
			profiles = append(profiles, name)
		}
	}
	sort.Strings(profiles)
	return profiles, nil
}

// LoadProfileFrom loads a named profile as a generic configuration map.
func LoadProfileFrom(userConfigDir, name string) (map[string]interface{}, error) {
	if err := ValidateProfileName(name); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(ProfilePath(userConfigDir, name))
	if err != nil {
		return nil, fmt.Errorf("reading profile %q: %w", name, err)
	}
	values := make(map[string]interface{})
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("parsing profile %q: %w", name, err)
	}
	return values, nil
}

// SaveProfileTo writes a named profile, refusing to overwrite unless force is true.
func SaveProfileTo(userConfigDir, name string, values map[string]interface{}, force bool) error {
	if err := ValidateProfileName(name); err != nil {
		return err
	}
	path := ProfilePath(userConfigDir, name)
	if !force && fileExists(path) {
		return fmt.Errorf("profile %q already exists; use --force to overwrite", name)
	}
	data, err := yaml.Marshal(values)
	if err != nil {
		return fmt.Errorf("marshaling profile %q: %w", name, err)
	}
	if err := writeAtomically(path, data); err != nil {
		return fmt.Errorf("saving profile %q: %w", name, err)
	}
	return nil
}

// UserProfilesDir returns the user's autospec profile directory.
func UserProfilesDir() (string, error) {
	dir, err := UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting user config directory: %w", err)
	}
	return ProfilesDir(dir), nil
}

// LoadProfile loads a named profile from the user's autospec config directory.
func LoadProfile(name string) (map[string]interface{}, error) {
	dir, err := UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("getting user config directory: %w", err)
	}
	return LoadProfileFrom(dir, name)
}

// SaveProfile saves a named profile to the user's autospec config directory.
func SaveProfile(name string, values map[string]interface{}, force bool) error {
	dir, err := UserConfigDir()
	if err != nil {
		return fmt.Errorf("getting user config directory: %w", err)
	}
	return SaveProfileTo(dir, name, values, force)
}

// ListProfiles lists profiles in the user's autospec config directory.
func ListProfiles() ([]string, error) {
	dir, err := UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("getting user config directory: %w", err)
	}
	return ListProfilesFrom(dir)
}

// ListAllProfiles returns the union of user and project profile names.
func ListAllProfiles() ([]string, error) {
	user, err := ListProfiles()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(ProjectProfilesDir())
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading project profiles directory: %w", err)
	}
	seen := make(map[string]bool, len(user))
	for _, name := range user {
		seen[name] = true
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yml" {
			continue
		}
		name := entry.Name()[:len(entry.Name())-len(filepath.Ext(entry.Name()))]
		if ValidateProfileName(name) == nil && !seen[name] {
			user = append(user, name)
			seen[name] = true
		}
	}
	sort.Strings(user)
	return user, nil
}
