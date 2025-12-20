package cliagent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsTokenValid(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		expiresAtMs int64
		want        bool
	}{
		"zero value": {
			expiresAtMs: 0,
			want:        false,
		},
		"expired yesterday": {
			expiresAtMs: time.Now().Add(-24 * time.Hour).UnixMilli(),
			want:        false,
		},
		"expires in 1 minute (within buffer)": {
			expiresAtMs: time.Now().Add(1 * time.Minute).UnixMilli(),
			want:        false,
		},
		"expires in 10 minutes (outside buffer)": {
			expiresAtMs: time.Now().Add(10 * time.Minute).UnixMilli(),
			want:        true,
		},
		"expires in 1 year": {
			expiresAtMs: time.Now().Add(365 * 24 * time.Hour).UnixMilli(),
			want:        true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := isTokenValid(tt.expiresAtMs)
			if got != tt.want {
				t.Errorf("isTokenValid(%d) = %v, want %v", tt.expiresAtMs, got, tt.want)
			}
		})
	}
}

func TestClaudeAuthStatus_IsAuthenticated(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status ClaudeAuthStatus
		want   bool
	}{
		"no auth": {
			status: ClaudeAuthStatus{AuthType: AuthTypeNone},
			want:   false,
		},
		"oauth valid": {
			status: ClaudeAuthStatus{AuthType: AuthTypeOAuth, Valid: true},
			want:   true,
		},
		"oauth invalid": {
			status: ClaudeAuthStatus{AuthType: AuthTypeOAuth, Valid: false},
			want:   false,
		},
		"api valid": {
			status: ClaudeAuthStatus{AuthType: AuthTypeAPI, Valid: true},
			want:   true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := tt.status.IsAuthenticated(); got != tt.want {
				t.Errorf("IsAuthenticated() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClaudeAuthStatus_RecommendedSetup(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status      ClaudeAuthStatus
		wantContain string
	}{
		"not installed": {
			status:      ClaudeAuthStatus{Installed: false},
			wantContain: "not installed",
		},
		"oauth max valid": {
			status: ClaudeAuthStatus{
				Installed:        true,
				AuthType:         AuthTypeOAuth,
				SubscriptionType: "max",
				Valid:            true,
			},
			wantContain: "max subscription",
		},
		"oauth expired": {
			status: ClaudeAuthStatus{
				Installed: true,
				AuthType:  AuthTypeOAuth,
				Valid:     false,
			},
			wantContain: "expired",
		},
		"api key": {
			status: ClaudeAuthStatus{
				Installed: true,
				AuthType:  AuthTypeAPI,
				APIKeySet: true,
			},
			wantContain: "API key",
		},
		"no auth": {
			status: ClaudeAuthStatus{
				Installed: true,
				AuthType:  AuthTypeNone,
			},
			wantContain: "authenticate",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := tt.status.RecommendedSetup()
			if !contains(got, tt.wantContain) {
				t.Errorf("RecommendedSetup() = %q, want to contain %q", got, tt.wantContain)
			}
		})
	}
}

func TestClaudeAuthStatus_ExpiresIn(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		status   ClaudeAuthStatus
		wantZero bool
	}{
		"not oauth": {
			status:   ClaudeAuthStatus{AuthType: AuthTypeAPI},
			wantZero: true,
		},
		"oauth no expiry": {
			status:   ClaudeAuthStatus{AuthType: AuthTypeOAuth},
			wantZero: true,
		},
		"oauth expired": {
			status: ClaudeAuthStatus{
				AuthType:  AuthTypeOAuth,
				ExpiresAt: time.Now().Add(-1 * time.Hour),
			},
			wantZero: true,
		},
		"oauth future": {
			status: ClaudeAuthStatus{
				AuthType:  AuthTypeOAuth,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			},
			wantZero: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := tt.status.ExpiresIn()
			if tt.wantZero && got != 0 {
				t.Errorf("ExpiresIn() = %v, want 0", got)
			}
			if !tt.wantZero && got == 0 {
				t.Errorf("ExpiresIn() = 0, want non-zero")
			}
		})
	}
}

func TestReadOAuthCredentials(t *testing.T) {
	t.Parallel()

	t.Run("valid credentials file", func(t *testing.T) {
		t.Parallel()

		// Create temp dir to simulate ~/.claude/
		tmpDir := t.TempDir()
		credDir := filepath.Join(tmpDir, ".claude")
		if err := os.MkdirAll(credDir, 0700); err != nil {
			t.Fatal(err)
		}

		creds := claudeCredentials{
			ClaudeAIOAuth: &claudeOAuthData{
				AccessToken:      "test-token",
				ExpiresAt:        time.Now().Add(24 * time.Hour).UnixMilli(),
				SubscriptionType: "max",
			},
		}

		data, _ := json.Marshal(creds)
		credPath := filepath.Join(credDir, ".credentials.json")
		if err := os.WriteFile(credPath, data, 0600); err != nil {
			t.Fatal(err)
		}

		// Can't easily test readOAuthCredentials directly since it uses os.UserHomeDir,
		// but we can test the JSON parsing logic
		var parsed claudeCredentials
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Failed to parse credentials: %v", err)
		}

		if parsed.ClaudeAIOAuth == nil {
			t.Fatal("Expected ClaudeAIOAuth to be non-nil")
		}
		if parsed.ClaudeAIOAuth.SubscriptionType != "max" {
			t.Errorf("SubscriptionType = %q, want %q", parsed.ClaudeAIOAuth.SubscriptionType, "max")
		}
	})

	t.Run("missing oauth data", func(t *testing.T) {
		t.Parallel()

		// Empty credentials
		data := []byte(`{}`)
		var parsed claudeCredentials
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		if parsed.ClaudeAIOAuth != nil {
			t.Error("Expected ClaudeAIOAuth to be nil for empty credentials")
		}
	})

	t.Run("invalid json degrades gracefully", func(t *testing.T) {
		t.Parallel()

		data := []byte(`{invalid json}`)
		var parsed claudeCredentials
		err := json.Unmarshal(data, &parsed)
		if err == nil {
			t.Error("Expected error for invalid JSON")
		}
		// This is the expected behavior - caller should check for nil
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
