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
	assert.Equal(t, string(JcodeRunnerExec), jcodeDefaults["runner"])
	assert.Equal(t, "", jcodeDefaults["socket_path"])
	assert.Equal(t, "", jcodeDefaults["binary"])
	assert.Equal(t, "", jcodeDefaults["home"])
	assert.False(t, jcodeDefaults["inherit_logins"].(bool))
	assert.Equal(t, (30 * time.Second).String(), jcodeDefaults["startup_timeout"])
	assert.Equal(t, (30 * time.Second).String(), jcodeDefaults["cleanup_timeout"])

	cfg := JcodeConfig{
		Mode:           JcodeModePrivate,
		SocketPath:     "/Users/alice/.jcode/run/jcode-api.sock",
		Binary:         "/Users/alice/bin/jcode",
		Home:           "/Users/alice/.jcode",
		InheritLogins:  true,
		StartupTimeout: 10 * time.Second,
		CleanupTimeout: 20 * time.Second,
	}
	redacted := cfg.Redacted()
	assert.Equal(t, "[redacted]", redacted.SocketPath)
	assert.Equal(t, "[redacted]", redacted.Binary)
	assert.Equal(t, "[redacted]", redacted.Home)
	assert.Equal(t, cfg.Mode, redacted.Mode)
	assert.Equal(t, cfg.InheritLogins, redacted.InheritLogins)
}

func TestJcodeRunnerValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		runner JcodeRunner
		valid  bool
	}{
		"exec":    {runner: JcodeRunnerExec, valid: true},
		"sdk":     {runner: JcodeRunnerSDK, valid: true},
		"custom":  {runner: JcodeRunnerCustom, valid: true},
		"unset":   {runner: "", valid: true},
		"invalid": {runner: "shell", valid: false},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := validateJcodeConfig(JcodeConfig{Runner: tt.runner}, "config")
			if tt.valid {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), "jcode.runner")
		})
	}
}

func TestEffectiveJcodeRunner(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config JcodeConfig
		want   JcodeRunner
	}{
		"unset defaults to exec": {
			config: JcodeConfig{Mode: JcodeModePrivate, Binary: "/custom/jcode"},
			want:   JcodeRunnerExec,
		},
		"explicit exec": {
			config: JcodeConfig{Runner: JcodeRunnerExec},
			want:   JcodeRunnerExec,
		},
		"explicit sdk": {
			config: JcodeConfig{Runner: JcodeRunnerSDK},
			want:   JcodeRunnerSDK,
		},
		"explicit custom": {
			config: JcodeConfig{Runner: JcodeRunnerCustom},
			want:   JcodeRunnerCustom,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.config.EffectiveRunner())
		})
	}
}

func TestJcodeExplicitRunnerFixtures(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		fixture string
		want    JcodeRunner
	}{
		"custom binary remains explicit exec": {
			fixture: "custom-binary.yaml",
			want:    JcodeRunnerExec,
		},
		"SDK requires explicit opt in": {
			fixture: "sdk-opt-in.yaml",
			want:    JcodeRunnerSDK,
		},
		"missing binary still resolves as exec": {
			fixture: "missing-binary.yaml",
			want:    JcodeRunnerExec,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assertJcodeRunner(t, tt.fixture, tt.want)
		})
	}
}

func TestJcodeExplicitRunnerValidationErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		fixture string
		want    string
	}{
		"unsupported runner": {
			fixture: "unsupported-runner.yaml",
			want:    "jcode.runner",
		},
		"unsafe custom binary": {
			fixture: "unsafe-binary.yaml",
			want:    "jcode.binary",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := requireJcodeFixtureError(t, tt.fixture)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
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
