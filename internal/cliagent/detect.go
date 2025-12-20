// Package cliagent provides abstractions for CLI AI coding agents.
package cliagent

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// AuthType represents the type of Claude authentication.
type AuthType string

const (
	// AuthTypeOAuth indicates OAuth-based authentication (Max/Pro subscription).
	AuthTypeOAuth AuthType = "oauth"
	// AuthTypeAPI indicates API key-based authentication.
	AuthTypeAPI AuthType = "api"
	// AuthTypeNone indicates no authentication detected.
	AuthTypeNone AuthType = "none"
)

// ClaudeAuthStatus contains Claude Code authentication detection results.
type ClaudeAuthStatus struct {
	// Installed indicates if Claude Code CLI is in PATH.
	Installed bool
	// Version is the installed CLI version (empty if not installed).
	Version string
	// AuthType indicates the authentication method detected.
	AuthType AuthType
	// SubscriptionType is the plan type for OAuth auth (e.g., "max", "pro").
	SubscriptionType string
	// ExpiresAt is the token expiration time for OAuth auth.
	ExpiresAt time.Time
	// Valid indicates if the detected auth is currently valid.
	Valid bool
	// APIKeySet indicates if ANTHROPIC_API_KEY env var is set.
	APIKeySet bool
}

// claudeCredentials represents the structure of ~/.claude/.credentials.json.
// Only fields we need are included for forward compatibility.
type claudeCredentials struct {
	ClaudeAIOAuth *claudeOAuthData `json:"claudeAiOauth,omitempty"`
}

type claudeOAuthData struct {
	AccessToken      string `json:"accessToken,omitempty"`
	ExpiresAt        int64  `json:"expiresAt,omitempty"` // Unix timestamp in milliseconds
	SubscriptionType string `json:"subscriptionType,omitempty"`
}

// expiryBuffer is the time before actual expiry when we consider token expired.
const expiryBuffer = 5 * time.Minute

// DetectClaudeAuth detects Claude Code installation and authentication status.
// This reads ~/.claude/.credentials.json (internal Claude Code file) and checks
// environment variables. The detection is read-only with no side effects.
//
// Note: This reads an undocumented internal file that may change in future
// Claude Code versions. The function degrades gracefully if the file format changes.
func DetectClaudeAuth() ClaudeAuthStatus {
	status := ClaudeAuthStatus{
		AuthType: AuthTypeNone,
	}

	// Check if Claude is installed
	status.Installed, status.Version = detectClaudeInstalled()

	// Check for API key in environment
	status.APIKeySet = isAPIKeySet()

	// Try to read OAuth credentials
	if oauthData := readOAuthCredentials(); oauthData != nil {
		status.AuthType = AuthTypeOAuth
		status.SubscriptionType = oauthData.SubscriptionType
		status.ExpiresAt = time.UnixMilli(oauthData.ExpiresAt)
		status.Valid = isTokenValid(oauthData.ExpiresAt)
	} else if status.APIKeySet {
		// Fall back to API key if no OAuth
		status.AuthType = AuthTypeAPI
		status.Valid = true // Can't verify API key without making a request
	}

	return status
}

// detectClaudeInstalled checks if Claude CLI is installed and returns version.
func detectClaudeInstalled() (installed bool, version string) {
	path, err := exec.LookPath("claude")
	if err != nil || path == "" {
		return false, ""
	}

	// Try to get version (with timeout)
	cmd := exec.Command("claude", "--version")
	output, err := cmd.Output()
	if err != nil {
		return true, "unknown"
	}

	version = strings.TrimSpace(string(output))
	return true, version
}

// isAPIKeySet checks if ANTHROPIC_API_KEY environment variable is set and non-empty.
func isAPIKeySet() bool {
	key := os.Getenv("ANTHROPIC_API_KEY")
	return key != ""
}

// readOAuthCredentials attempts to read Claude OAuth credentials from the credentials file.
// Returns nil if file doesn't exist, is unreadable, or has unexpected format.
func readOAuthCredentials() *claudeOAuthData {
	credPath := getCredentialsPath()
	if credPath == "" {
		return nil
	}

	data, err := os.ReadFile(credPath)
	if err != nil {
		return nil // File doesn't exist or unreadable - not an error
	}

	var creds claudeCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil // Invalid JSON - degrade gracefully
	}

	if creds.ClaudeAIOAuth == nil || creds.ClaudeAIOAuth.AccessToken == "" {
		return nil // No OAuth data
	}

	return creds.ClaudeAIOAuth
}

// getCredentialsPath returns the path to Claude credentials file.
// Returns empty string if home directory cannot be determined.
func getCredentialsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude", ".credentials.json")
}

// isTokenValid checks if the OAuth token is still valid with a buffer.
func isTokenValid(expiresAtMs int64) bool {
	if expiresAtMs == 0 {
		return false
	}
	expiresAt := time.UnixMilli(expiresAtMs)
	return time.Now().Add(expiryBuffer).Before(expiresAt)
}

// IsAuthenticated returns true if any form of valid authentication is detected.
func (s ClaudeAuthStatus) IsAuthenticated() bool {
	return s.AuthType != AuthTypeNone && s.Valid
}

// RecommendedSetup returns a human-readable recommendation based on auth status.
func (s ClaudeAuthStatus) RecommendedSetup() string {
	if !s.Installed {
		return "Claude Code not installed. Install from: https://claude.ai/download"
	}

	if s.AuthType == AuthTypeOAuth && s.Valid {
		return "claude-code preset (using " + s.SubscriptionType + " subscription)"
	}

	if s.AuthType == AuthTypeOAuth && !s.Valid {
		return "OAuth token expired. Run 'claude' to refresh login."
	}

	if s.AuthType == AuthTypeAPI && s.APIKeySet {
		return "claude-code preset (using API key). Consider OAuth for better rate limits."
	}

	return "Run 'claude' to authenticate, or set ANTHROPIC_API_KEY."
}

// ExpiresIn returns the duration until token expiry, or 0 if not applicable.
func (s ClaudeAuthStatus) ExpiresIn() time.Duration {
	if s.AuthType != AuthTypeOAuth || s.ExpiresAt.IsZero() {
		return 0
	}
	d := time.Until(s.ExpiresAt)
	if d < 0 {
		return 0
	}
	return d
}
