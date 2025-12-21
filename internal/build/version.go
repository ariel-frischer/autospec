// Package build provides version and build information for autospec.
// This package intentionally has no dependencies on other internal packages
// to avoid import cycles.
package build

var (
	// Version information - set via ldflags during build
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// IsDevBuild returns true if running a development build (not a release).
// Used to gate experimental features that aren't ready for production.
func IsDevBuild() bool {
	return Version == "dev"
}

// MultiAgentEnabled returns true if multi-agent selection is enabled.
// Currently tied to dev builds. When ready for production, change to return true.
//
// TODO: Re-enable when multi-agent feature development begins.
// The agent selection prompt in `autospec init` is confusing without the
// full feature implemented. Keeping the code but hiding the UI until ready.
// To re-enable: return IsDevBuild()
func MultiAgentEnabled() bool {
	return false
}
