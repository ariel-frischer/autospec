package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup       func(t *testing.T, dir string)
		wantErr     bool
		wantErrMsg  string
		checkResult func(t *testing.T, s *Settings)
	}{
		"missing file returns empty settings": {
			setup: func(t *testing.T, dir string) {
				// No setup - file doesn't exist
			},
			checkResult: func(t *testing.T, s *Settings) {
				assert.NotNil(t, s)
				assert.False(t, s.Exists())
				assert.Empty(t, s.getAllowList())
			},
		},
		"empty file returns empty settings": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, "")
			},
			checkResult: func(t *testing.T, s *Settings) {
				assert.NotNil(t, s)
				assert.True(t, s.Exists())
				assert.Empty(t, s.getAllowList())
			},
		},
		"valid JSON with permissions": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, `{"permissions": {"allow": ["Bash(foo:*)"]}}`)
			},
			checkResult: func(t *testing.T, s *Settings) {
				assert.NotNil(t, s)
				assert.True(t, s.Exists())
				assert.Equal(t, []string{"Bash(foo:*)"}, s.getAllowList())
			},
		},
		"malformed JSON returns error": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, `{invalid json}`)
			},
			wantErr:    true,
			wantErrMsg: "parsing settings file",
		},
		"preserves extra fields": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, `{
					"permissions": {"allow": ["Bash(foo:*)"]},
					"sandbox": {"enabled": true},
					"custom_field": "value"
				}`)
			},
			checkResult: func(t *testing.T, s *Settings) {
				assert.NotNil(t, s)
				assert.Contains(t, s.data, "sandbox")
				assert.Contains(t, s.data, "custom_field")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, dir)
			}

			s, err := Load(dir)

			if tt.wantErr {
				require.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
				return
			}

			require.NoError(t, err)
			if tt.checkResult != nil {
				tt.checkResult(t, s)
			}
		})
	}
}

func TestHasPermission(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		allowList []string
		perm      string
		want      bool
	}{
		"empty allow list": {
			allowList: nil,
			perm:      "Bash(autospec:*)",
			want:      false,
		},
		"permission not in list": {
			allowList: []string{"Bash(foo:*)", "Bash(bar:*)"},
			perm:      "Bash(autospec:*)",
			want:      false,
		},
		"permission in list": {
			allowList: []string{"Bash(foo:*)", "Bash(autospec:*)", "Bash(bar:*)"},
			perm:      "Bash(autospec:*)",
			want:      true,
		},
		"exact match required": {
			allowList: []string{"Bash(autospec:run)"},
			perm:      "Bash(autospec:*)",
			want:      false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := &Settings{data: make(map[string]interface{})}
			if tt.allowList != nil {
				setAllowList(s, tt.allowList)
			}

			got := s.HasPermission(tt.perm)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCheckDenyList(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		denyList []string
		perm     string
		want     bool
	}{
		"empty deny list": {
			denyList: nil,
			perm:     "Bash(autospec:*)",
			want:     false,
		},
		"permission not denied": {
			denyList: []string{"Bash(rm:*)", "Bash(sudo:*)"},
			perm:     "Bash(autospec:*)",
			want:     false,
		},
		"permission denied": {
			denyList: []string{"Bash(rm:*)", "Bash(autospec:*)", "Bash(sudo:*)"},
			perm:     "Bash(autospec:*)",
			want:     true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := &Settings{data: make(map[string]interface{})}
			if tt.denyList != nil {
				setDenyList(s, tt.denyList)
			}

			got := s.CheckDenyList(tt.perm)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCheck(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup           func(t *testing.T, dir string)
		wantStatus      SettingsStatus
		wantMsgContains string
	}{
		"missing file": {
			setup: func(t *testing.T, dir string) {
				// No setup - file doesn't exist
			},
			wantStatus:      StatusMissing,
			wantMsgContains: "not found",
		},
		"permission in deny list": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, `{
					"permissions": {"deny": ["Bash(autospec:*)"]}
				}`)
			},
			wantStatus:      StatusDenied,
			wantMsgContains: "explicitly denied",
		},
		"permission missing from allow list": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, `{
					"permissions": {"allow": ["Bash(foo:*)"]}
				}`)
			},
			wantStatus:      StatusNeedsPermission,
			wantMsgContains: "missing",
		},
		"permission configured": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, `{
					"permissions": {"allow": ["Bash(autospec:*)"]}
				}`)
			},
			wantStatus:      StatusConfigured,
			wantMsgContains: "configured",
		},
		"empty allow list": {
			setup: func(t *testing.T, dir string) {
				createSettingsFile(t, dir, `{
					"permissions": {"allow": []}
				}`)
			},
			wantStatus:      StatusNeedsPermission,
			wantMsgContains: "missing",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.setup != nil {
				tt.setup(t, dir)
			}

			s, err := Load(dir)
			require.NoError(t, err)

			result := s.Check()

			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Contains(t, result.Message, tt.wantMsgContains)
		})
	}
}

func TestAddPermission(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		initialAllow []string
		permToAdd    string
		wantAllow    []string
	}{
		"add to empty list": {
			initialAllow: nil,
			permToAdd:    "Bash(autospec:*)",
			wantAllow:    []string{"Bash(autospec:*)"},
		},
		"add to existing list": {
			initialAllow: []string{"Bash(foo:*)"},
			permToAdd:    "Bash(autospec:*)",
			wantAllow:    []string{"Bash(foo:*)", "Bash(autospec:*)"},
		},
		"no duplicate when already present": {
			initialAllow: []string{"Bash(foo:*)", "Bash(autospec:*)"},
			permToAdd:    "Bash(autospec:*)",
			wantAllow:    []string{"Bash(foo:*)", "Bash(autospec:*)"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := &Settings{data: make(map[string]interface{})}
			if tt.initialAllow != nil {
				setAllowList(s, tt.initialAllow)
			}

			s.AddPermission(tt.permToAdd)

			assert.Equal(t, tt.wantAllow, s.getAllowList())
		})
	}
}

func TestSave(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup       func(t *testing.T, s *Settings)
		checkResult func(t *testing.T, dir string, data []byte)
	}{
		"creates directory if missing": {
			setup: func(t *testing.T, s *Settings) {
				s.AddPermission("Bash(autospec:*)")
			},
			checkResult: func(t *testing.T, dir string, data []byte) {
				assert.FileExists(t, filepath.Join(dir, SettingsDir, SettingsFileName))
			},
		},
		"writes pretty-printed JSON": {
			setup: func(t *testing.T, s *Settings) {
				s.AddPermission("Bash(autospec:*)")
			},
			checkResult: func(t *testing.T, dir string, data []byte) {
				assert.Contains(t, string(data), "  ") // Indentation
				assert.True(t, json.Valid(data))
			},
		},
		"preserves existing fields": {
			setup: func(t *testing.T, s *Settings) {
				s.data["sandbox"] = map[string]interface{}{"enabled": true}
				s.data["custom"] = "value"
				s.AddPermission("Bash(autospec:*)")
			},
			checkResult: func(t *testing.T, dir string, data []byte) {
				assert.Contains(t, string(data), "sandbox")
				assert.Contains(t, string(data), "custom")
				assert.Contains(t, string(data), "Bash(autospec:*)")
			},
		},
		"ends with newline": {
			setup: func(t *testing.T, s *Settings) {
				s.AddPermission("Bash(autospec:*)")
			},
			checkResult: func(t *testing.T, dir string, data []byte) {
				assert.True(t, len(data) > 0)
				assert.Equal(t, byte('\n'), data[len(data)-1])
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			s := NewSettings(dir)

			if tt.setup != nil {
				tt.setup(t, s)
			}

			err := s.Save()
			require.NoError(t, err)

			data, err := os.ReadFile(s.FilePath())
			require.NoError(t, err)

			if tt.checkResult != nil {
				tt.checkResult(t, dir, data)
			}
		})
	}
}

func TestSaveAtomicWrite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	s := NewSettings(dir)
	s.AddPermission("Bash(autospec:*)")

	// Save the file
	err := s.Save()
	require.NoError(t, err)

	// Verify no temp files left behind
	entries, err := os.ReadDir(filepath.Join(dir, SettingsDir))
	require.NoError(t, err)

	for _, entry := range entries {
		assert.False(t,
			filepath.Ext(entry.Name()) == ".tmp",
			"temp file should not remain: %s", entry.Name())
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	// Test that loading, modifying, and saving preserves all fields
	dir := t.TempDir()

	original := `{
  "permissions": {
    "allow": ["Bash(existing:*)"],
    "ask": ["Write(*)"],
    "deny": ["Bash(rm:*)"]
  },
  "sandbox": {
    "enabled": true,
    "allow_paths": ["/tmp"]
  }
}`
	createSettingsFile(t, dir, original)

	// Load
	s, err := Load(dir)
	require.NoError(t, err)

	// Modify
	s.AddPermission("Bash(autospec:*)")

	// Save
	err = s.Save()
	require.NoError(t, err)

	// Reload and verify
	s2, err := Load(dir)
	require.NoError(t, err)

	// Check all permissions preserved
	assert.True(t, s2.HasPermission("Bash(existing:*)"))
	assert.True(t, s2.HasPermission("Bash(autospec:*)"))
	assert.True(t, s2.CheckDenyList("Bash(rm:*)"))

	// Check ask list preserved
	perms := s2.getPermissions()
	askRaw := perms["ask"]
	ask := interfaceSliceToStrings(askRaw)
	assert.Contains(t, ask, "Write(*)")

	// Check sandbox preserved
	assert.Contains(t, s2.data, "sandbox")
	sandbox := s2.data["sandbox"].(map[string]interface{})
	assert.Equal(t, true, sandbox["enabled"])
}

func TestSettingsStatus_String(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status SettingsStatus
		want   string
	}{
		"Configured":      {status: StatusConfigured, want: "Configured"},
		"Missing":         {status: StatusMissing, want: "Missing"},
		"NeedsPermission": {status: StatusNeedsPermission, want: "NeedsPermission"},
		"Denied":          {status: StatusDenied, want: "Denied"},
		"Unknown":         {status: SettingsStatus(99), want: "Unknown"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

func TestCheckInDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	createSettingsFile(t, dir, `{"permissions": {"allow": ["Bash(autospec:*)"]}}`)

	result, err := CheckInDir(dir)

	require.NoError(t, err)
	assert.Equal(t, StatusConfigured, result.Status)
}

// Helper functions

func createSettingsFile(t *testing.T, dir, content string) {
	t.Helper()
	settingsDir := filepath.Join(dir, SettingsDir)
	err := os.MkdirAll(settingsDir, 0755)
	require.NoError(t, err)

	settingsPath := filepath.Join(settingsDir, SettingsFileName)
	err = os.WriteFile(settingsPath, []byte(content), 0644)
	require.NoError(t, err)
}

func setAllowList(s *Settings, perms []string) {
	permData := s.getPermissions()
	allow := make([]interface{}, len(perms))
	for i, p := range perms {
		allow[i] = p
	}
	permData["allow"] = allow
}

func setDenyList(s *Settings, perms []string) {
	permData := s.getPermissions()
	deny := make([]interface{}, len(perms))
	for i, p := range perms {
		deny[i] = p
	}
	permData["deny"] = deny
}
