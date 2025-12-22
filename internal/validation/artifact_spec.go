package validation

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// SpecValidator validates spec.yaml artifacts.
type SpecValidator struct {
	baseValidator
}

// Type returns the artifact type.
func (v *SpecValidator) Type() ArtifactType {
	return ArtifactTypeSpec
}

// Validate validates a spec.yaml file at the given path.
func (v *SpecValidator) Validate(path string) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Parse the YAML file
	root, err := parseYAMLFile(path)
	if err != nil {
		result.AddError(&ValidationError{
			Path:    path,
			Message: fmt.Sprintf("failed to parse YAML: %v", err),
			Hint:    "Check the YAML syntax for errors",
		})
		return result
	}

	rootMapping := getRootMapping(root)
	if rootMapping == nil {
		result.AddError(&ValidationError{
			Path:    path,
			Message: "expected a YAML mapping at document root",
			Hint:    "The spec.yaml file should start with key-value pairs, not a list or scalar",
		})
		return result
	}

	// Validate required fields
	featureNode := validateRequiredField(rootMapping, "feature", result)
	userStoriesNode := validateRequiredField(rootMapping, "user_stories", result)
	requirementsNode := validateRequiredField(rootMapping, "requirements", result)

	// Validate feature section
	if featureNode != nil {
		v.validateFeature(featureNode, result)
	}

	// Validate user_stories section
	if userStoriesNode != nil {
		v.validateUserStories(userStoriesNode, result)
	}

	// Validate requirements section
	if requirementsNode != nil {
		v.validateRequirements(requirementsNode, result)
	}

	// Build summary if valid
	if result.Valid {
		result.Summary = v.buildSummary(rootMapping)
	}

	return result
}

// validateFeature validates the feature section.
func (v *SpecValidator) validateFeature(node *yaml.Node, result *ValidationResult) {
	if !validateFieldType(node, "feature", yaml.MappingNode, "object", result) {
		return
	}

	// Required fields in feature
	validateRequiredField(node, "branch", result)
	validateRequiredField(node, "created", result)

	// Validate status enum if present
	statusNode := findNode(node, "status")
	if statusNode != nil {
		validateEnumValue(statusNode, "feature.status", []string{"Draft", "Review", "Approved", "Completed"}, result)
	}
}

// validateUserStories validates the user_stories section.
func (v *SpecValidator) validateUserStories(node *yaml.Node, result *ValidationResult) {
	if !validateFieldType(node, "user_stories", yaml.SequenceNode, "array", result) {
		return
	}

	for i, storyNode := range node.Content {
		path := fmt.Sprintf("user_stories[%d]", i)
		v.validateUserStory(storyNode, path, result)
	}
}

// validateUserStory validates a single user story.
func (v *SpecValidator) validateUserStory(node *yaml.Node, path string, result *ValidationResult) {
	if node.Kind != yaml.MappingNode {
		result.AddError(&ValidationError{
			Path:     path,
			Line:     getNodeLine(node),
			Message:  fmt.Sprintf("wrong type for '%s'", path),
			Expected: "object",
			Actual:   nodeKindToString(node.Kind),
		})
		return
	}

	// Required fields
	requiredFields := []string{"id", "title", "priority", "as_a", "i_want", "so_that"}
	for _, field := range requiredFields {
		fieldNode := findNode(node, field)
		if fieldNode == nil {
			result.AddError(&ValidationError{
				Path:    fmt.Sprintf("%s.%s", path, field),
				Line:    getNodeLine(node),
				Message: fmt.Sprintf("missing required field: %s", field),
				Hint:    fmt.Sprintf("Add the '%s' field to this user story", field),
			})
		}
	}

	// Validate priority enum
	priorityNode := findNode(node, "priority")
	if priorityNode != nil {
		validateEnumValue(priorityNode, path+".priority", []string{"P0", "P1", "P2", "P3"}, result)
	}

	// Validate acceptance_scenarios is an array if present
	scenariosNode := findNode(node, "acceptance_scenarios")
	if scenariosNode != nil {
		validateFieldType(scenariosNode, path+".acceptance_scenarios", yaml.SequenceNode, "array", result)
	}
}

// validateRequirements validates the requirements section.
func (v *SpecValidator) validateRequirements(node *yaml.Node, result *ValidationResult) {
	if !validateFieldType(node, "requirements", yaml.MappingNode, "object", result) {
		return
	}

	// functional is required
	functionalNode := findNode(node, "functional")
	if functionalNode == nil {
		result.AddError(&ValidationError{
			Path:    "requirements.functional",
			Line:    getNodeLine(node),
			Message: "missing required field: functional",
			Hint:    "Add a 'functional' field with a list of functional requirements",
		})
	} else {
		validateFieldType(functionalNode, "requirements.functional", yaml.SequenceNode, "array", result)
	}

	// non_functional is optional but should be an array if present
	nonFunctionalNode := findNode(node, "non_functional")
	if nonFunctionalNode != nil {
		validateFieldType(nonFunctionalNode, "requirements.non_functional", yaml.SequenceNode, "array", result)
	}
}

// buildSummary builds the summary for a valid spec artifact.
func (v *SpecValidator) buildSummary(root *yaml.Node) *ArtifactSummary {
	summary := &ArtifactSummary{
		Type:   ArtifactTypeSpec,
		Counts: make(map[string]int),
	}

	// Count user stories
	userStoriesNode := findNode(root, "user_stories")
	if userStoriesNode != nil && userStoriesNode.Kind == yaml.SequenceNode {
		summary.Counts["user_stories"] = len(userStoriesNode.Content)
	}

	// Count functional requirements
	requirementsNode := findNode(root, "requirements")
	if requirementsNode != nil {
		functionalNode := findNode(requirementsNode, "functional")
		if functionalNode != nil && functionalNode.Kind == yaml.SequenceNode {
			summary.Counts["functional_requirements"] = len(functionalNode.Content)
		}

		nonFunctionalNode := findNode(requirementsNode, "non_functional")
		if nonFunctionalNode != nil && nonFunctionalNode.Kind == yaml.SequenceNode {
			summary.Counts["non_functional_requirements"] = len(nonFunctionalNode.Content)
		}
	}

	// Count key entities
	keyEntitiesNode := findNode(root, "key_entities")
	if keyEntitiesNode != nil && keyEntitiesNode.Kind == yaml.SequenceNode {
		summary.Counts["key_entities"] = len(keyEntitiesNode.Content)
	}

	// Count edge cases
	edgeCasesNode := findNode(root, "edge_cases")
	if edgeCasesNode != nil && edgeCasesNode.Kind == yaml.SequenceNode {
		summary.Counts["edge_cases"] = len(edgeCasesNode.Content)
	}

	return summary
}
