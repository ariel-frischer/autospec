package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

const (
	// GitHubAPIURL is the endpoint for fetching the latest release.
	GitHubAPIURL = "https://api.github.com/repos/ariel-frischer/autospec/releases/latest"

	// DefaultHTTPTimeout is the default timeout for HTTP requests.
	DefaultHTTPTimeout = 5 * time.Second
)

// ReleaseInfo represents a GitHub release.
type ReleaseInfo struct {
	TagName     string    `json:"tag_name"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

// Asset represents a single release asset.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// UpdateCheck contains the result of an update check.
type UpdateCheck struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	DownloadURL     string
	ChecksumURL     string
	AssetName       string
}

// Checker provides update checking functionality.
type Checker struct {
	httpClient *http.Client
	apiURL     string
}

// NewChecker creates a new update checker with the given timeout.
func NewChecker(timeout time.Duration) *Checker {
	if timeout == 0 {
		timeout = DefaultHTTPTimeout
	}
	return &Checker{
		httpClient: &http.Client{Timeout: timeout},
		apiURL:     GitHubAPIURL,
	}
}

// CheckForUpdate checks GitHub for a newer version of autospec.
func (c *Checker) CheckForUpdate(ctx context.Context, currentVersion string) (*UpdateCheck, error) {
	current, err := ParseVersion(currentVersion)
	if err != nil {
		return nil, fmt.Errorf("parsing current version: %w", err)
	}

	if current.IsDev() {
		return &UpdateCheck{
			CurrentVersion:  currentVersion,
			UpdateAvailable: false,
		}, nil
	}

	release, err := c.fetchLatestRelease(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching latest release: %w", err)
	}

	latest, err := ParseVersion(release.TagName)
	if err != nil {
		return nil, fmt.Errorf("parsing latest version: %w", err)
	}

	result := &UpdateCheck{
		CurrentVersion:  currentVersion,
		LatestVersion:   release.TagName,
		UpdateAvailable: latest.IsNewerThan(current),
	}

	if result.UpdateAvailable {
		if err := c.populateDownloadURLs(result, release); err != nil {
			return nil, fmt.Errorf("finding download URLs: %w", err)
		}
	}

	return result, nil
}

// fetchLatestRelease fetches the latest release from GitHub API.
func (c *Checker) fetchLatestRelease(ctx context.Context) (*ReleaseInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "autospec-update-checker")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("rate limit exceeded")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no releases found")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &release, nil
}

// populateDownloadURLs finds and sets the appropriate download URLs for the current platform.
func (c *Checker) populateDownloadURLs(check *UpdateCheck, release *ReleaseInfo) error {
	assetName := buildAssetName(check.LatestVersion)
	checksumName := "checksums.txt"

	for _, asset := range release.Assets {
		switch asset.Name {
		case assetName:
			check.DownloadURL = asset.BrowserDownloadURL
			check.AssetName = asset.Name
		case checksumName:
			check.ChecksumURL = asset.BrowserDownloadURL
		}
	}

	if check.DownloadURL == "" {
		return fmt.Errorf("no asset found for platform %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	return nil
}

// buildAssetName constructs the asset name for the current platform.
func buildAssetName(version string) string {
	// Map Go's GOOS/GOARCH to goreleaser naming conventions
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "Darwin"
	} else if osName == "linux" {
		osName = "Linux"
	}

	archName := runtime.GOARCH
	if archName == "amd64" {
		archName = "x86_64"
	}

	// Strip 'v' prefix for version in filename
	ver := version
	if len(ver) > 0 && ver[0] == 'v' {
		ver = ver[1:]
	}

	return fmt.Sprintf("autospec_%s_%s_%s.tar.gz", ver, osName, archName)
}
