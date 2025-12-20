package workflow

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/config"
)

func TestShowSecurityNoticeOnce(t *testing.T) {
	tests := map[string]struct {
		cfg         *config.Configuration
		envVar      string
		wantShown   bool
		wantContain string
	}{
		"shows notice when not previously shown": {
			cfg: &config.Configuration{
				SkipPermissionsNoticeShown: false,
			},
			wantShown:   true,
			wantContain: "Security Notice",
		},
		"skips notice when previously shown": {
			cfg: &config.Configuration{
				SkipPermissionsNoticeShown: true,
			},
			wantShown: false,
		},
		"skips notice when env var set": {
			cfg: &config.Configuration{
				SkipPermissionsNoticeShown: false,
			},
			envVar:    "1",
			wantShown: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Set/unset environment variable
			if tt.envVar != "" {
				os.Setenv("AUTOSPEC_SKIP_PERMISSIONS_NOTICE", tt.envVar)
				defer os.Unsetenv("AUTOSPEC_SKIP_PERMISSIONS_NOTICE")
			} else {
				os.Unsetenv("AUTOSPEC_SKIP_PERMISSIONS_NOTICE")
			}

			var buf bytes.Buffer
			shown := ShowSecurityNoticeOnce(&buf, tt.cfg)

			if shown != tt.wantShown {
				t.Errorf("ShowSecurityNoticeOnce() = %v, want %v", shown, tt.wantShown)
			}

			if tt.wantContain != "" && !strings.Contains(buf.String(), tt.wantContain) {
				t.Errorf("Output does not contain %q, got: %s", tt.wantContain, buf.String())
			}

			if !tt.wantShown && buf.Len() > 0 {
				t.Errorf("Expected no output when notice should be skipped, got: %s", buf.String())
			}
		})
	}
}

func TestShowSecurityNotice_SandboxStatus(t *testing.T) {
	tests := map[string]struct {
		sandboxEnabled bool
		wantContain    string
		wantNotContain string
	}{
		"sandbox enabled shows checkmark": {
			sandboxEnabled: true,
			wantContain:    "enabled",
		},
		"sandbox disabled shows warning": {
			sandboxEnabled: false,
			wantContain:    "disabled",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			showSecurityNotice(&buf, tt.sandboxEnabled)

			output := buf.String()
			if !strings.Contains(output, tt.wantContain) {
				t.Errorf("Output does not contain %q, got: %s", tt.wantContain, output)
			}
		})
	}
}
