package cliagent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

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
		"oauth": {
			status: ClaudeAuthStatus{AuthType: AuthTypeOAuth},
			want:   true,
		},
		"api": {
			status: ClaudeAuthStatus{AuthType: AuthTypeAPI},
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
		"oauth max": {
			status: ClaudeAuthStatus{
				Installed:        true,
				AuthType:         AuthTypeOAuth,
				SubscriptionType: "max",
			},
			wantContain: "max subscription",
		},
		"oauth pro": {
			status: ClaudeAuthStatus{
				Installed:        true,
				AuthType:         AuthTypeOAuth,
				SubscriptionType: "pro",
			},
			wantContain: "pro subscription",
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
			if !stringContains(got, tt.wantContain) {
				t.Errorf("RecommendedSetup() = %q, want to contain %q", got, tt.wantContain)
			}
		})
	}
}

func TestCredentialsJSONParsing(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		json             string
		wantNil          bool
		wantSubscription string
	}{
		"valid oauth credentials": {
			json: `{
				"claudeAiOauth": {
					"accessToken": "sk-ant-oat01-xxx",
					"subscriptionType": "max"
				}
			}`,
			wantNil:          false,
			wantSubscription: "max",
		},
		"pro subscription": {
			json: `{
				"claudeAiOauth": {
					"accessToken": "sk-ant-oat01-yyy",
					"subscriptionType": "pro"
				}
			}`,
			wantNil:          false,
			wantSubscription: "pro",
		},
		"empty oauth object": {
			json:    `{"claudeAiOauth": {}}`,
			wantNil: true,
		},
		"missing access token": {
			json:    `{"claudeAiOauth": {"subscriptionType": "max"}}`,
			wantNil: true,
		},
		"empty json": {
			json:    `{}`,
			wantNil: true,
		},
		"null claudeAiOauth": {
			json:    `{"claudeAiOauth": null}`,
			wantNil: true,
		},
		"extra fields ignored": {
			json: `{
				"claudeAiOauth": {
					"accessToken": "token",
					"refreshToken": "refresh",
					"expiresAt": 9999999999999,
					"subscriptionType": "max",
					"rateLimitTier": "default_claude_max_20x"
				},
				"someOtherField": "ignored"
			}`,
			wantNil:          false,
			wantSubscription: "max",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var creds claudeCredentials
			if err := json.Unmarshal([]byte(tt.json), &creds); err != nil {
				t.Fatalf("Failed to parse JSON: %v", err)
			}

			// Simulate the check in readOAuthCredentials
			var result *claudeOAuthData
			if creds.ClaudeAIOAuth != nil && creds.ClaudeAIOAuth.AccessToken != "" {
				result = creds.ClaudeAIOAuth
			}

			if tt.wantNil && result != nil {
				t.Error("Expected nil result, got non-nil")
			}
			if !tt.wantNil && result == nil {
				t.Error("Expected non-nil result, got nil")
			}
			if !tt.wantNil && result != nil && result.SubscriptionType != tt.wantSubscription {
				t.Errorf("SubscriptionType = %q, want %q", result.SubscriptionType, tt.wantSubscription)
			}
		})
	}
}

func TestInvalidJSON(t *testing.T) {
	t.Parallel()

	invalidJSONs := []string{
		`{invalid}`,
		`not json at all`,
		`{"unclosed": `,
		``,
	}

	for _, input := range invalidJSONs {
		var creds claudeCredentials
		err := json.Unmarshal([]byte(input), &creds)
		if err == nil && input != "" {
			t.Errorf("Expected error for invalid JSON: %q", input)
		}
	}
}

func TestReadOAuthCredentials_WithMockFile(t *testing.T) {
	// Create temp credentials file
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "credentials.json")

	tests := map[string]struct {
		content          string
		wantNil          bool
		wantSubscription string
	}{
		"valid max subscription": {
			content: `{
				"claudeAiOauth": {
					"accessToken": "test-token-12345",
					"subscriptionType": "max"
				}
			}`,
			wantNil:          false,
			wantSubscription: "max",
		},
		"valid pro subscription": {
			content: `{
				"claudeAiOauth": {
					"accessToken": "test-token-67890",
					"subscriptionType": "pro"
				}
			}`,
			wantNil:          false,
			wantSubscription: "pro",
		},
		"no access token": {
			content: `{"claudeAiOauth": {"subscriptionType": "max"}}`,
			wantNil: true,
		},
		"empty file": {
			content: `{}`,
			wantNil: true,
		},
		"invalid json": {
			content: `{not valid json`,
			wantNil: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Write mock credentials file
			if err := os.WriteFile(credPath, []byte(tt.content), 0600); err != nil {
				t.Fatalf("Failed to write mock credentials: %v", err)
			}

			// Override the credentials path
			oldPath := credentialsPathOverride
			credentialsPathOverride = credPath
			defer func() { credentialsPathOverride = oldPath }()

			result := readOAuthCredentials()

			if tt.wantNil && result != nil {
				t.Errorf("Expected nil, got %+v", result)
			}
			if !tt.wantNil && result == nil {
				t.Error("Expected non-nil result, got nil")
			}
			if !tt.wantNil && result != nil && result.SubscriptionType != tt.wantSubscription {
				t.Errorf("SubscriptionType = %q, want %q", result.SubscriptionType, tt.wantSubscription)
			}
		})
	}
}

func TestReadOAuthCredentials_FileNotExists(t *testing.T) {
	// Override to non-existent path
	oldPath := credentialsPathOverride
	credentialsPathOverride = "/nonexistent/path/credentials.json"
	defer func() { credentialsPathOverride = oldPath }()

	result := readOAuthCredentials()
	if result != nil {
		t.Errorf("Expected nil for non-existent file, got %+v", result)
	}
}

func TestGetCredentialsPath_Override(t *testing.T) {
	// Test that override works
	oldPath := credentialsPathOverride
	credentialsPathOverride = "/custom/path/creds.json"
	defer func() { credentialsPathOverride = oldPath }()

	got := getCredentialsPath()
	if got != "/custom/path/creds.json" {
		t.Errorf("getCredentialsPath() = %q, want %q", got, "/custom/path/creds.json")
	}
}

func TestGetCredentialsPath_Default(t *testing.T) {
	// Ensure override is empty
	oldPath := credentialsPathOverride
	credentialsPathOverride = ""
	defer func() { credentialsPathOverride = oldPath }()

	path := getCredentialsPath()
	if path == "" {
		t.Skip("Could not determine home directory")
	}

	// Should contain the expected path components
	if !stringContains(path, ".claude") || !stringContains(path, ".credentials.json") {
		t.Errorf("getCredentialsPath() = %q, expected to contain .claude/.credentials.json", path)
	}
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
