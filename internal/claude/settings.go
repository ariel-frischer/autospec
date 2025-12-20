package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SettingsStatus represents the state of Claude settings configuration.
type SettingsStatus int

const (
	// StatusConfigured indicates the settings file exists with the required permission.
	StatusConfigured SettingsStatus = iota
	// StatusMissing indicates the settings file does not exist.
	StatusMissing
	// StatusNeedsPermission indicates the settings file exists but lacks the required permission.
	StatusNeedsPermission
	// StatusDenied indicates the required permission is explicitly denied.
	StatusDenied
)

// String returns a human-readable representation of the status.
func (s SettingsStatus) String() string {
	switch s {
	case StatusConfigured:
		return "Configured"
	case StatusMissing:
		return "Missing"
	case StatusNeedsPermission:
		return "NeedsPermission"
	case StatusDenied:
		return "Denied"
	default:
		return "Unknown"
	}
}

// SettingsCheckResult contains the result of validating Claude settings.
type SettingsCheckResult struct {
	Status   SettingsStatus
	Message  string
	FilePath string
}

// RequiredPermission is the permission autospec needs in Claude settings.
const RequiredPermission = "Bash(autospec:*)"

// SettingsFileName is the name of the Claude settings file.
const SettingsFileName = "settings.local.json"

// SettingsDir is the directory containing Claude settings.
const SettingsDir = ".claude"

// Settings represents a Claude settings file with flexible JSON structure.
// Uses map[string]interface{} to preserve unknown fields during modification.
type Settings struct {
	data     map[string]interface{}
	filePath string
}

// NewSettings creates a new empty Settings instance for the given project directory.
func NewSettings(projectDir string) *Settings {
	return &Settings{
		data:     make(map[string]interface{}),
		filePath: filepath.Join(projectDir, SettingsDir, SettingsFileName),
	}
}

// Load reads and parses Claude settings from the project directory.
// Returns a Settings instance even if the file doesn't exist (with empty data).
// Returns an error only for actual failures like permission errors or malformed JSON.
func Load(projectDir string) (*Settings, error) {
	settingsPath := filepath.Join(projectDir, SettingsDir, SettingsFileName)
	return loadFromPath(settingsPath)
}

// loadFromPath loads settings from a specific file path.
func loadFromPath(settingsPath string) (*Settings, error) {
	s := &Settings{
		data:     make(map[string]interface{}),
		filePath: settingsPath,
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("reading settings file %s: %w", settingsPath, err)
	}

	if len(data) == 0 {
		return s, nil
	}

	if err := json.Unmarshal(data, &s.data); err != nil {
		return nil, fmt.Errorf("parsing settings file %s: %w", settingsPath, err)
	}

	return s, nil
}

// FilePath returns the path to the settings file.
func (s *Settings) FilePath() string {
	return s.filePath
}

// Exists returns true if the settings file exists on disk.
func (s *Settings) Exists() bool {
	_, err := os.Stat(s.filePath)
	return err == nil
}

// getPermissions returns the permissions object, creating it if necessary.
func (s *Settings) getPermissions() map[string]interface{} {
	perms, ok := s.data["permissions"].(map[string]interface{})
	if !ok {
		perms = make(map[string]interface{})
		s.data["permissions"] = perms
	}
	return perms
}

// getAllowList returns the allow list as a slice of strings.
func (s *Settings) getAllowList() []string {
	perms := s.getPermissions()
	allowRaw, ok := perms["allow"]
	if !ok {
		return nil
	}

	return interfaceSliceToStrings(allowRaw)
}

// getDenyList returns the deny list as a slice of strings.
func (s *Settings) getDenyList() []string {
	perms := s.getPermissions()
	denyRaw, ok := perms["deny"]
	if !ok {
		return nil
	}

	return interfaceSliceToStrings(denyRaw)
}

// interfaceSliceToStrings converts an interface{} that should be []interface{}
// containing strings to a []string. Returns nil if conversion fails.
func interfaceSliceToStrings(v interface{}) []string {
	slice, ok := v.([]interface{})
	if !ok {
		return nil
	}

	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

// HasPermission checks if the given permission exists in the allow list.
func (s *Settings) HasPermission(perm string) bool {
	for _, p := range s.getAllowList() {
		if p == perm {
			return true
		}
	}
	return false
}

// CheckDenyList checks if the given permission is explicitly denied.
func (s *Settings) CheckDenyList(perm string) bool {
	for _, p := range s.getDenyList() {
		if p == perm {
			return true
		}
	}
	return false
}

// Check validates Claude settings for the required autospec permission.
// Returns a SettingsCheckResult with appropriate status and message.
func (s *Settings) Check() SettingsCheckResult {
	if !s.Exists() {
		return SettingsCheckResult{
			Status:   StatusMissing,
			Message:  fmt.Sprintf(".claude/settings.local.json not found (run 'autospec init' to configure)"),
			FilePath: "",
		}
	}

	if s.CheckDenyList(RequiredPermission) {
		return SettingsCheckResult{
			Status:   StatusDenied,
			Message:  fmt.Sprintf("%s is explicitly denied. Remove from permissions.deny in %s to allow autospec commands.", RequiredPermission, s.filePath),
			FilePath: s.filePath,
		}
	}

	if !s.HasPermission(RequiredPermission) {
		return SettingsCheckResult{
			Status:   StatusNeedsPermission,
			Message:  fmt.Sprintf("missing %s permission (run 'autospec init' to fix)", RequiredPermission),
			FilePath: s.filePath,
		}
	}

	return SettingsCheckResult{
		Status:   StatusConfigured,
		Message:  fmt.Sprintf("%s permission configured", RequiredPermission),
		FilePath: s.filePath,
	}
}

// CheckInDir performs a settings check for the given project directory.
// This is a convenience function that loads settings and checks them in one call.
func CheckInDir(projectDir string) (SettingsCheckResult, error) {
	settings, err := Load(projectDir)
	if err != nil {
		return SettingsCheckResult{}, fmt.Errorf("loading claude settings: %w", err)
	}
	return settings.Check(), nil
}

// AddPermission adds a permission to the allow list if not already present.
// Does not check the deny list - caller should check that first.
func (s *Settings) AddPermission(perm string) {
	if s.HasPermission(perm) {
		return
	}

	perms := s.getPermissions()
	allowList := s.getAllowList()

	// Convert to []interface{} for JSON compatibility
	newAllow := make([]interface{}, len(allowList)+1)
	for i, p := range allowList {
		newAllow[i] = p
	}
	newAllow[len(allowList)] = perm

	perms["allow"] = newAllow
}

// AddPermissions adds multiple permissions to the allow list, skipping duplicates.
// Returns the list of permissions that were actually added (not already present).
// This method is idempotent - calling multiple times with the same permissions
// has the same effect as calling once.
// Does not check the deny list - caller should check that first for each permission.
func (s *Settings) AddPermissions(permissions []string) []string {
	var added []string

	for _, perm := range permissions {
		if !s.HasPermission(perm) {
			s.AddPermission(perm)
			added = append(added, perm)
		}
	}

	return added
}

// Save writes the settings to disk using atomic write (temp file + rename).
// Creates the .claude directory if it doesn't exist.
// Written JSON is pretty-printed with indentation for human readability.
func (s *Settings) Save() error {
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing settings: %w", err)
	}

	// Add trailing newline for POSIX compliance
	data = append(data, '\n')

	return atomicWrite(s.filePath, data)
}

// atomicWrite writes data to a file atomically using temp file + rename.
func atomicWrite(filePath string, data []byte) error {
	dir := filepath.Dir(filePath)
	tmpFile, err := os.CreateTemp(dir, ".settings-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on any error
	defer func() {
		if tmpPath != "" {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("renaming temp file to %s: %w", filePath, err)
	}

	// Clear tmpPath so defer doesn't try to remove the final file
	tmpPath = ""
	return nil
}
