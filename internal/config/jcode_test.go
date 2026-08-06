package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJcodeConfigDefaultsAndRedaction(t *testing.T) {
	t.Parallel()

	defaults := GetDefaults()
	jcodeDefaults, ok := defaults["jcode"].(map[string]interface{})
	require.True(t, ok, "jcode defaults should be a nested map")

	assert.Equal(t, string(JcodeModeConnect), jcodeDefaults["mode"])
	assert.Equal(t, "", jcodeDefaults["socket_path"])
	assert.Equal(t, "", jcodeDefaults["binary"])
	assert.Equal(t, "", jcodeDefaults["home"])
	assert.False(t, jcodeDefaults["inherit_logins"].(bool))
	assert.Equal(t, (30 * time.Second).String(), jcodeDefaults["startup_timeout"])
	assert.Equal(t, (30 * time.Second).String(), jcodeDefaults["cleanup_timeout"])
	assert.Equal(t, "", jcodeDefaults["startup_command"])
	assert.Equal(t, 2, jcodeDefaults["reconnect_attempts"])
	assert.Equal(t, 1, jcodeDefaults["restart_attempts"])
	assert.Equal(t, (250 * time.Millisecond).String(), jcodeDefaults["retry_delay"])

	cfg := JcodeConfig{
		Mode:              JcodeModePrivate,
		SocketPath:        "/Users/alice/.jcode/run/jcode-api.sock",
		Binary:            "/Users/alice/bin/jcode",
		Home:              "/Users/alice/.jcode",
		InheritLogins:     true,
		StartupTimeout:    10 * time.Second,
		CleanupTimeout:    20 * time.Second,
		StartupCommand:    "jcode serve --token=secret",
		ReconnectAttempts: 2,
		RestartAttempts:   1,
		RetryDelay:        250 * time.Millisecond,
	}
	redacted := cfg.Redacted()
	assert.Equal(t, "[redacted]", redacted.SocketPath)
	assert.Equal(t, "[redacted]", redacted.Binary)
	assert.Equal(t, "[redacted]", redacted.Home)
	assert.Equal(t, cfg.Mode, redacted.Mode)
	assert.Equal(t, cfg.InheritLogins, redacted.InheritLogins)
	assert.Equal(t, redactedJcodeValue, redacted.StartupCommand)
}

func TestValidateJcodeConfig(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config       JcodeConfig
		wantErrField string
	}{
		"connect mode": {
			config: JcodeConfig{Mode: JcodeModeConnect},
		},
		"private mode": {
			config: JcodeConfig{
				Mode:           JcodeModePrivate,
				Binary:         "/opt/jcode/bin/jcode",
				Home:           "/tmp/jcode-home",
				InheritLogins:  true,
				StartupTimeout: 10 * time.Second,
				CleanupTimeout: 20 * time.Second,
			},
		},
		"auto mode": {
			config: JcodeConfig{Mode: JcodeModeAuto, StartupCommand: "jcode serve", ReconnectAttempts: 2, RestartAttempts: 1, RetryDelay: 250 * time.Millisecond},
		},
		"invalid mode": {
			config:       JcodeConfig{Mode: "launch"},
			wantErrField: "jcode.mode",
		},
		"invalid startup timeout": {
			config:       JcodeConfig{Mode: JcodeModeConnect, StartupTimeout: -time.Second},
			wantErrField: "jcode.startup_timeout",
		},
		"invalid cleanup timeout": {
			config:       JcodeConfig{Mode: JcodeModeConnect, CleanupTimeout: -time.Second},
			wantErrField: "jcode.cleanup_timeout",
		},
		"startup command in connect mode": {
			config:       JcodeConfig{Mode: JcodeModeConnect, StartupCommand: "jcode serve"},
			wantErrField: "jcode.startup_command",
		},
		"blank startup command": {
			config:       JcodeConfig{Mode: JcodeModePrivate, StartupCommand: " \t"},
			wantErrField: "jcode.startup_command",
		},
		"negative reconnect attempts": {
			config:       JcodeConfig{Mode: JcodeModePrivate, ReconnectAttempts: -1},
			wantErrField: "jcode.reconnect_attempts",
		},
		"too many restart attempts": {
			config:       JcodeConfig{Mode: JcodeModePrivate, RestartAttempts: 11},
			wantErrField: "jcode.restart_attempts",
		},
		"unbounded retry delay": {
			config:       JcodeConfig{Mode: JcodeModePrivate, RetryDelay: 6 * time.Minute},
			wantErrField: "jcode.retry_delay",
		},
		"unsafe socket path": {
			config:       JcodeConfig{Mode: JcodeModeConnect, SocketPath: "/tmp/$(secret).sock"},
			wantErrField: "jcode.socket_path",
		},
		"unsafe binary path": {
			config:       JcodeConfig{Mode: JcodeModePrivate, Binary: "/tmp/jcode;echo secret"},
			wantErrField: "jcode.binary",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			cfg := Configuration{SpecsDir: "./specs", StateDir: "./state", Jcode: tt.config}
			err := ValidateConfigValues(&cfg, "config")
			if tt.wantErrField == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrField)
			assert.NotContains(t, err.Error(), "secret")
		})
	}
}

func TestLoadJcodeConfigFromYAMLAndEnvironment(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmpDir, "home"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	projectPath := filepath.Join(tmpDir, "project.yml")
	projectConfig := `jcode:
  mode: private
  socket_path: /tmp/jcode-api.sock
  binary: /opt/jcode/bin/jcode
  home: /tmp/jcode-home
  inherit_logins: true
  startup_timeout: 10s
  cleanup_timeout: 20s
`
	require.NoError(t, os.WriteFile(projectPath, []byte(projectConfig), 0o644))
	t.Setenv("AUTOSPEC_JCODE_MODE", "connect")
	t.Setenv("AUTOSPEC_JCODE_STARTUP_TIMEOUT", "45s")

	cfg, err := LoadWithOptions(LoadOptions{ProjectConfigPath: projectPath, SkipWarnings: true})
	require.NoError(t, err)
	assert.Equal(t, JcodeModeConnect, cfg.Jcode.Mode)
	assert.Equal(t, "/tmp/jcode-api.sock", cfg.Jcode.SocketPath)
	assert.Equal(t, "/opt/jcode/bin/jcode", cfg.Jcode.Binary)
	assert.Equal(t, "/tmp/jcode-home", cfg.Jcode.Home)
	assert.True(t, cfg.Jcode.InheritLogins)
	assert.Equal(t, 45*time.Second, cfg.Jcode.StartupTimeout)
	assert.Equal(t, 20*time.Second, cfg.Jcode.CleanupTimeout)
}

func TestLoadJcodeLifecycleConfigPrecedenceAndProfile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), ".config"))
	projectDir := t.TempDir()
	original, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(original)
	require.NoError(t, os.Chdir(projectDir))

	userPath, err := UserConfigPath()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Dir(userPath), 0o755))
	require.NoError(t, os.WriteFile(userPath, []byte("jcode:\n  mode: private\n  reconnect_attempts: 1\n"), 0o644))
	projectPath := filepath.Join(projectDir, ".autospec", "config.yml")
	require.NoError(t, os.MkdirAll(filepath.Dir(projectPath), 0o755))
	require.NoError(t, os.WriteFile(projectPath, []byte("jcode:\n  mode: auto\n  restart_attempts: 2\n"), 0o644))
	profilePath := ProjectProfilePath("cheap")
	require.NoError(t, os.MkdirAll(filepath.Dir(profilePath), 0o755))
	require.NoError(t, os.WriteFile(profilePath, []byte("jcode:\n  startup_command: jcode serve\n  retry_delay: 1s\n"), 0o644))
	t.Setenv("AUTOSPEC_JCODE_MODE", "auto")

	cfg, err := LoadWithOptions(LoadOptions{ProjectConfigPath: projectPath, Profile: "cheap", SkipWarnings: true})
	require.NoError(t, err)
	assert.Equal(t, JcodeModeAuto, cfg.Jcode.Mode)
	assert.Equal(t, 1, cfg.Jcode.ReconnectAttempts)
	assert.Equal(t, 2, cfg.Jcode.RestartAttempts)
	assert.Equal(t, "jcode serve", cfg.Jcode.StartupCommand)
	assert.Equal(t, time.Second, cfg.Jcode.RetryDelay)
}

func TestEnvTransformJcodeKeys(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  string
	}{
		"mode":            {input: "AUTOSPEC_JCODE_MODE", want: "jcode.mode"},
		"socket path":     {input: "AUTOSPEC_JCODE_SOCKET_PATH", want: "jcode.socket_path"},
		"binary":          {input: "AUTOSPEC_JCODE_BINARY", want: "jcode.binary"},
		"home":            {input: "AUTOSPEC_JCODE_HOME", want: "jcode.home"},
		"inherit logins":  {input: "AUTOSPEC_JCODE_INHERIT_LOGINS", want: "jcode.inherit_logins"},
		"startup timeout": {input: "AUTOSPEC_JCODE_STARTUP_TIMEOUT", want: "jcode.startup_timeout"},
		"cleanup timeout": {input: "AUTOSPEC_JCODE_CLEANUP_TIMEOUT", want: "jcode.cleanup_timeout"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, envTransform(tt.input))
		})
	}
}

func TestJcodeValidationRedactsUnsafeValues(t *testing.T) {
	t.Parallel()

	unsafePath := strings.Join([]string{"/tmp", "jcode", "$(TOKEN)"}, "/")
	cfg := Configuration{
		SpecsDir: "./specs",
		StateDir: "./state",
		Jcode:    JcodeConfig{Mode: JcodeModeConnect, SocketPath: unsafePath},
	}

	err := ValidateConfigValues(&cfg, "config")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jcode.socket_path")
	assert.NotContains(t, err.Error(), unsafePath)
	assert.NotContains(t, err.Error(), "TOKEN")
}
