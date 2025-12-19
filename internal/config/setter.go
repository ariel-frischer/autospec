package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrEmptyKeyPath is returned when an empty key path is provided.
var ErrEmptyKeyPath = errors.New("empty key path")

// ParseKeyPath splits a dotted key path into its component parts.
// For example, "notifications.enabled" becomes ["notifications", "enabled"].
func ParseKeyPath(path string) ([]string, error) {
	if path == "" {
		return nil, ErrEmptyKeyPath
	}
	parts := strings.Split(path, ".")
	return parts, nil
}

// SetNestedValue sets a value in a YAML node tree at the specified key path.
// Creates parent nodes if they don't exist.
func SetNestedValue(root *yaml.Node, keyPath []string, value interface{}) error {
	if root.Kind == 0 {
		root.Kind = yaml.DocumentNode
		root.Content = append(root.Content, &yaml.Node{Kind: yaml.MappingNode})
	}
	var mapNode *yaml.Node
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		mapNode = root.Content[0]
	} else if root.Kind == yaml.MappingNode {
		mapNode = root
	} else {
		return fmt.Errorf("root node must be document or mapping, got %v", root.Kind)
	}
	return setValueInMap(mapNode, keyPath, value)
}

// setValueInMap recursively navigates/creates the map structure and sets the value.
func setValueInMap(node *yaml.Node, keyPath []string, value interface{}) error {
	if len(keyPath) == 0 {
		return nil
	}
	key := keyPath[0]
	remaining := keyPath[1:]

	// Find or create the key
	keyIndex := findKeyIndex(node, key)
	if keyIndex == -1 {
		// Key doesn't exist, create it
		return createNestedKey(node, key, remaining, value)
	}

	valueIndex := keyIndex + 1
	if len(remaining) == 0 {
		// This is the final key, set the value
		return setScalarValue(node.Content[valueIndex], value)
	}

	// Navigate deeper
	childNode := node.Content[valueIndex]
	if childNode.Kind != yaml.MappingNode {
		// Convert to mapping node for nested keys
		childNode.Kind = yaml.MappingNode
		childNode.Content = nil
	}
	return setValueInMap(childNode, remaining, value)
}

// findKeyIndex finds the index of a key in a mapping node's content.
// Returns -1 if the key is not found.
func findKeyIndex(node *yaml.Node, key string) int {
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return i
		}
	}
	return -1
}

// createNestedKey creates a new key and its nested structure.
func createNestedKey(node *yaml.Node, key string, remaining []string, value interface{}) error {
	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: key}
	var valueNode *yaml.Node

	if len(remaining) == 0 {
		valueNode = &yaml.Node{Kind: yaml.ScalarNode}
		if err := setScalarValue(valueNode, value); err != nil {
			return err
		}
	} else {
		valueNode = &yaml.Node{Kind: yaml.MappingNode}
		if err := setValueInMap(valueNode, remaining, value); err != nil {
			return err
		}
	}
	node.Content = append(node.Content, keyNode, valueNode)
	return nil
}

// setScalarValue sets the value and tag of a scalar node.
func setScalarValue(node *yaml.Node, value interface{}) error {
	node.Kind = yaml.ScalarNode
	switch v := value.(type) {
	case bool:
		node.Tag = "!!bool"
		node.Value = fmt.Sprintf("%t", v)
	case int:
		node.Tag = "!!int"
		node.Value = fmt.Sprintf("%d", v)
	case string:
		node.Tag = "!!str"
		node.Value = v
	default:
		node.Value = fmt.Sprintf("%v", v)
	}
	return nil
}

// GetNestedValue retrieves a value from a YAML node tree at the specified key path.
// Returns nil if the key path doesn't exist.
func GetNestedValue(root *yaml.Node, keyPath []string) *yaml.Node {
	if root == nil || len(keyPath) == 0 {
		return nil
	}
	var mapNode *yaml.Node
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		mapNode = root.Content[0]
	} else if root.Kind == yaml.MappingNode {
		mapNode = root
	} else {
		return nil
	}
	return getValueFromMap(mapNode, keyPath)
}

// getValueFromMap recursively navigates the map structure to find a value.
func getValueFromMap(node *yaml.Node, keyPath []string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode || len(keyPath) == 0 {
		return nil
	}
	key := keyPath[0]
	remaining := keyPath[1:]
	keyIndex := findKeyIndex(node, key)
	if keyIndex == -1 {
		return nil
	}
	valueNode := node.Content[keyIndex+1]
	if len(remaining) == 0 {
		return valueNode
	}
	return getValueFromMap(valueNode, remaining)
}

// writeAtomically writes content to a file atomically using a temporary file and rename.
// Creates parent directories if they don't exist.
func writeAtomically(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}
	tmpFile, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		// Clean up temp file on error
		if tmpPath != "" {
			os.Remove(tmpPath)
		}
	}()
	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("writing to temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("renaming temp file: %w", err)
	}
	tmpPath = "" // Prevent cleanup since rename succeeded
	return nil
}

// SetConfigValue sets a configuration value in a YAML file.
// Validates the key and value against the schema before writing.
// Creates the file if it doesn't exist.
func SetConfigValue(filePath, key, value string) error {
	// Validate key and value
	parsed, err := ValidateValue(key, value)
	if err != nil {
		return fmt.Errorf("validating value: %w", err)
	}
	// Load existing YAML or create new
	root, err := loadOrCreateYAML(filePath)
	if err != nil {
		return err
	}
	// Parse key path
	keyPath, err := ParseKeyPath(key)
	if err != nil {
		return fmt.Errorf("parsing key path: %w", err)
	}
	// Set the value
	if err := SetNestedValue(root, keyPath, parsed.Parsed); err != nil {
		return fmt.Errorf("setting nested value: %w", err)
	}
	// Marshal and write
	content, err := yaml.Marshal(root)
	if err != nil {
		return fmt.Errorf("marshaling YAML: %w", err)
	}
	if err := writeAtomically(filePath, content); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}
	return nil
}

// loadOrCreateYAML loads a YAML file or creates an empty document node.
func loadOrCreateYAML(filePath string) (*yaml.Node, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create empty document with mapping
			return &yaml.Node{
				Kind:    yaml.DocumentNode,
				Content: []*yaml.Node{{Kind: yaml.MappingNode}},
			}, nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}
	return &root, nil
}
