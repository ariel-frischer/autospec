package changelog

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Changelog represents the root structure of a CHANGELOG.yaml file.
// It contains the project identifier and an ordered list of versions,
// with the newest versions appearing first.
type Changelog struct {
	Project  string    `yaml:"project"`
	Versions []Version `yaml:"versions"`
}

// UnmarshalYAML accepts the legacy autospec list format and chlog's map format.
func (c *Changelog) UnmarshalYAML(value *yaml.Node) error {
	type rawChangelog struct {
		Project  string    `yaml:"project"`
		Versions yaml.Node `yaml:"versions"`
	}

	var raw rawChangelog
	if err := value.Decode(&raw); err != nil {
		return err
	}

	versions, err := parseVersionsNode(&raw.Versions)
	if err != nil {
		return err
	}

	c.Project = raw.Project
	c.Versions = versions
	return nil
}

func parseVersionsNode(node *yaml.Node) ([]Version, error) {
	switch node.Kind {
	case yaml.SequenceNode:
		var versions []Version
		if err := node.Decode(&versions); err != nil {
			return nil, err
		}
		return versions, nil
	case yaml.MappingNode:
		return parseMappedVersions(node)
	default:
		return nil, fmt.Errorf("versions: expected sequence or mapping, got %d", node.Kind)
	}
}

func parseMappedVersions(node *yaml.Node) ([]Version, error) {
	versions := make([]Version, 0, len(node.Content)/2)
	for i := 0; i < len(node.Content)-1; i += 2 {
		version, err := parseMappedVersion(node.Content[i].Value, node.Content[i+1])
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, nil
}

func parseMappedVersion(version string, node *yaml.Node) (Version, error) {
	var mapped struct {
		Date    string  `yaml:"date"`
		Changes Changes `yaml:",inline"`
	}
	if err := node.Decode(&mapped); err != nil {
		return Version{}, fmt.Errorf("versions.%s: %w", version, err)
	}
	return Version{Version: version, Date: mapped.Date, Changes: mapped.Changes}, nil
}

// Version represents a single version entry in the changelog.
// The Version field should be a bare semantic version (e.g., "0.6.0") or
// the special identifier "unreleased". The CLI normalizes "v" prefixes on input.
// The Date field is required for released versions (format: YYYY-MM-DD)
// but should be empty for unreleased.
type Version struct {
	Version string  `yaml:"version"`
	Date    string  `yaml:"date,omitempty"`
	Changes Changes `yaml:"changes"`
}

// Changes groups change entries by Keep a Changelog category.
// All fields are optional; empty categories are omitted when rendering.
// Categories follow the Keep a Changelog specification:
// https://keepachangelog.com/en/1.1.0/
type Changes struct {
	Added      []string `yaml:"added,omitempty"`
	Changed    []string `yaml:"changed,omitempty"`
	Deprecated []string `yaml:"deprecated,omitempty"`
	Removed    []string `yaml:"removed,omitempty"`
	Fixed      []string `yaml:"fixed,omitempty"`
	Security   []string `yaml:"security,omitempty"`
}

// Entry represents a flattened view of a single changelog entry.
// This is used for querying and displaying individual entries,
// where the version and category context is needed alongside the text.
type Entry struct {
	Text     string `yaml:"text"`
	Category string `yaml:"category"`
	Version  string `yaml:"version"`
}

// IsEmpty returns true if the Changes struct has no entries in any category.
func (c Changes) IsEmpty() bool {
	return len(c.Added) == 0 &&
		len(c.Changed) == 0 &&
		len(c.Deprecated) == 0 &&
		len(c.Removed) == 0 &&
		len(c.Fixed) == 0 &&
		len(c.Security) == 0
}

// Count returns the total number of entries across all categories.
func (c Changes) Count() int {
	return len(c.Added) +
		len(c.Changed) +
		len(c.Deprecated) +
		len(c.Removed) +
		len(c.Fixed) +
		len(c.Security)
}

// IsUnreleased returns true if this version represents unreleased changes.
func (v Version) IsUnreleased() bool {
	return v.Version == "unreleased"
}

// Entries returns a flattened list of all entries in this version.
// Each entry includes the text, category, and version identifier.
func (v Version) Entries() []Entry {
	entries := make([]Entry, 0, v.Changes.Count())

	for _, text := range v.Changes.Added {
		entries = append(entries, Entry{Text: text, Category: "added", Version: v.Version})
	}
	for _, text := range v.Changes.Changed {
		entries = append(entries, Entry{Text: text, Category: "changed", Version: v.Version})
	}
	for _, text := range v.Changes.Deprecated {
		entries = append(entries, Entry{Text: text, Category: "deprecated", Version: v.Version})
	}
	for _, text := range v.Changes.Removed {
		entries = append(entries, Entry{Text: text, Category: "removed", Version: v.Version})
	}
	for _, text := range v.Changes.Fixed {
		entries = append(entries, Entry{Text: text, Category: "fixed", Version: v.Version})
	}
	for _, text := range v.Changes.Security {
		entries = append(entries, Entry{Text: text, Category: "security", Version: v.Version})
	}

	return entries
}

// ValidCategories returns the list of valid Keep a Changelog categories
// in their standard rendering order.
func ValidCategories() []string {
	return []string{"added", "changed", "deprecated", "removed", "fixed", "security"}
}
