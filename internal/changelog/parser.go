package changelog

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidationError represents a changelog validation error with context.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// Load reads and validates a CHANGELOG.yaml file from the given path.
// Returns the parsed Changelog struct or an error with context.
func Load(path string) (*Changelog, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening changelog file: %w", err)
	}
	defer f.Close()

	return LoadFromReader(f)
}

// LoadFromReader reads and validates a CHANGELOG.yaml from an io.Reader.
// This is useful for testing and for loading from embedded content.
func LoadFromReader(r io.Reader) (*Changelog, error) {
	var changelog Changelog

	dec := yaml.NewDecoder(r)
	if err := dec.Decode(&changelog); err != nil {
		return nil, fmt.Errorf("parsing changelog YAML: %w", err)
	}

	if err := Validate(&changelog); err != nil {
		return nil, err
	}

	return &changelog, nil
}

// Validate checks that a Changelog struct satisfies all schema constraints.
// Returns nil if valid, or a ValidationError with details if invalid.
func Validate(c *Changelog) error {
	if c.Project == "" {
		return &ValidationError{Field: "project", Message: "required field is empty"}
	}

	unreleasedCount := 0
	seenVersions := make(map[string]bool)

	for i, v := range c.Versions {
		if err := validateVersion(&v, i); err != nil {
			return err
		}

		normalizedVersion := NormalizeVersion(v.Version)
		if seenVersions[normalizedVersion] {
			return &ValidationError{
				Field:   fmt.Sprintf("versions[%d].version", i),
				Message: fmt.Sprintf("duplicate version %q", v.Version),
			}
		}
		seenVersions[normalizedVersion] = true

		if v.IsUnreleased() {
			unreleasedCount++
		}
	}

	if unreleasedCount > 1 {
		return &ValidationError{
			Field:   "versions",
			Message: "only one 'unreleased' version is allowed",
		}
	}

	return nil
}

// validateVersion checks constraints for a single version entry.
func validateVersion(v *Version, index int) error {
	if v.Version == "" {
		return &ValidationError{
			Field:   fmt.Sprintf("versions[%d].version", index),
			Message: "required field is empty",
		}
	}

	if !v.IsUnreleased() {
		if err := validateVersionFormat(v.Version, index); err != nil {
			return err
		}
		if err := validateDateRequired(v, index); err != nil {
			return err
		}
	}

	if v.Date != "" {
		if err := validateDateFormat(v.Date, index); err != nil {
			return err
		}
	}

	if v.Changes.IsEmpty() && !v.IsUnreleased() {
		return &ValidationError{
			Field:   fmt.Sprintf("versions[%d].changes", index),
			Message: "at least one change entry is required",
		}
	}

	if err := validateChangeEntries(&v.Changes, index); err != nil {
		return err
	}

	return nil
}

// validateVersionFormat checks that the version string is valid semver.
func validateVersionFormat(version string, index int) error {
	semverPattern := regexp.MustCompile(`^\d+\.\d+\.\d+(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$`)
	if !semverPattern.MatchString(version) {
		return &ValidationError{
			Field:   fmt.Sprintf("versions[%d].version", index),
			Message: fmt.Sprintf("invalid semver format %q (expected: X.Y.Z)", version),
		}
	}
	return nil
}

// validateDateRequired ensures released versions have a date.
func validateDateRequired(v *Version, index int) error {
	if v.Date == "" {
		return &ValidationError{
			Field:   fmt.Sprintf("versions[%d].date", index),
			Message: "date is required for released versions",
		}
	}
	return nil
}

// validateDateFormat checks that the date is in YYYY-MM-DD format.
func validateDateFormat(date string, index int) error {
	datePattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	if !datePattern.MatchString(date) {
		return &ValidationError{
			Field:   fmt.Sprintf("versions[%d].date", index),
			Message: fmt.Sprintf("invalid date format %q (expected: YYYY-MM-DD)", date),
		}
	}
	return nil
}

// validateChangeEntries checks that all change entries are non-empty strings.
func validateChangeEntries(c *Changes, versionIndex int) error {
	categories := map[string][]string{
		"added":      c.Added,
		"changed":    c.Changed,
		"deprecated": c.Deprecated,
		"removed":    c.Removed,
		"fixed":      c.Fixed,
		"security":   c.Security,
	}

	for category, entries := range categories {
		for i, entry := range entries {
			if strings.TrimSpace(entry) == "" {
				return &ValidationError{
					Field:   fmt.Sprintf("versions[%d].changes.%s[%d]", versionIndex, category, i),
					Message: "change entry cannot be empty",
				}
			}
		}
	}

	return nil
}

// NormalizeVersion normalizes a version string by removing the "v" prefix.
// This allows accepting both "v0.6.0" and "0.6.0" as input.
func NormalizeVersion(version string) string {
	return strings.TrimPrefix(strings.ToLower(version), "v")
}

// IsValidationError returns true if the error is a ValidationError.
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}
