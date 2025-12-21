package update

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a parsed semantic version.
type Version struct {
	Major int
	Minor int
	Patch int
	Raw   string
}

// ParseVersion parses a version string in the format "v0.6.1" or "0.6.1".
// Returns an error if the version string is invalid.
// The "dev" version is a special case that returns a zero version.
func ParseVersion(v string) (*Version, error) {
	raw := v
	v = strings.TrimPrefix(v, "v")

	if v == "dev" || v == "" {
		return &Version{Raw: raw}, nil
	}

	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("parsing version %q: expected format MAJOR.MINOR.PATCH", raw)
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("parsing major version %q: %w", parts[0], err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("parsing minor version %q: %w", parts[1], err)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("parsing patch version %q: %w", parts[2], err)
	}

	return &Version{Major: major, Minor: minor, Patch: patch, Raw: raw}, nil
}

// IsDev returns true if this is a development build (not a proper release version).
func (v *Version) IsDev() bool {
	return v.Raw == "dev" || v.Raw == ""
}

// String returns the version string in "vMAJOR.MINOR.PATCH" format.
func (v *Version) String() string {
	if v.IsDev() {
		return "dev"
	}
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare compares two versions and returns:
//   - -1 if v < other
//   - 0 if v == other
//   - 1 if v > other
//
// Dev versions are always considered less than any proper version.
func (v *Version) Compare(other *Version) int {
	if v.IsDev() && other.IsDev() {
		return 0
	}
	if v.IsDev() {
		return -1
	}
	if other.IsDev() {
		return 1
	}

	if v.Major != other.Major {
		return compareInts(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return compareInts(v.Minor, other.Minor)
	}
	return compareInts(v.Patch, other.Patch)
}

// IsNewerThan returns true if v is newer than other.
func (v *Version) IsNewerThan(other *Version) bool {
	return v.Compare(other) > 0
}

// compareInts compares two integers and returns -1, 0, or 1.
func compareInts(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
