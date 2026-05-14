package spec

// SelectionSource identifies why a feature directory was chosen.
type SelectionSource string

const (
	// SelectionSourceExplicit indicates a command argument or --spec flag selected the feature.
	SelectionSourceExplicit SelectionSource = "explicit"
	// SelectionSourcePersisted indicates project-local active feature state selected the feature.
	SelectionSourcePersisted SelectionSource = "persisted"
	// SelectionSourceGitBranch indicates the current git branch selected the feature.
	SelectionSourceGitBranch SelectionSource = "git_branch"
	// SelectionSourceFallback indicates autospec used the most recent feature directory.
	SelectionSourceFallback SelectionSource = "fallback"
)

// ActiveFeatureRequest contains the inputs needed to resolve an active feature.
type ActiveFeatureRequest struct {
	SpecsDir           string
	ExplicitIdentifier string
	StateDir           string
	RequiredArtifact   string
}

// ActiveFeatureResult reports the resolved feature and the source that selected it.
type ActiveFeatureResult struct {
	Metadata *Metadata
	Source   SelectionSource
}

// Active feature resolution should reuse existing lookup entry points:
// GetSpecMetadata handles explicit --spec and positional identifiers,
// DetectCurrentSpec handles git-branch and recent-directory fallback, and
// GetSpecDirectory provides exact, number, and slug matching.
//
// Current command-level lookup callers to centralize in later phases include:
// run --spec, status [spec-name], implement [spec-name-or-prompt],
// plan/tasks/implement stage auto-detection, and artifact type-only lookup.
