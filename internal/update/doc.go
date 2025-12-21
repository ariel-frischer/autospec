// Package update provides version checking and self-update functionality for autospec.
//
// The package includes:
//   - Semantic version parsing and comparison (version.go)
//   - GitHub API client for fetching release info (check.go)
//   - Binary download with progress display (download.go)
//   - Binary installation with backup and rollback (install.go)
//
// The update check is designed to be non-blocking when used with the version command,
// using goroutines to fetch release info without delaying the display of version
// information. The update command handles the complete download, verification, and
// installation flow with checksum verification and atomic file operations.
package update
