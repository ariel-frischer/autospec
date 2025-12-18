// Package workflow provides workflow orchestration for autospec.
// This file contains schema validation wrapper functions that invoke
// existing artifact validators and return errors compatible with ExecuteStage.
package workflow

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/validation"
)

// ValidateSpecSchema validates a spec.yaml file against its full schema.
// It wraps the existing SpecValidator and returns an error suitable for
// ExecuteStage's validation callback.
//
// Performance contract: <10ms (delegated to existing validator)
func ValidateSpecSchema(specDir string) error {
	specPath := filepath.Join(specDir, "spec.yaml")
	validator := &validation.SpecValidator{}
	result := validator.Validate(specPath)

	if result.Valid {
		return nil
	}

	return formatValidationErrors("spec.yaml", result.Errors)
}

// ValidatePlanSchema validates a plan.yaml file against its full schema.
// It wraps the existing PlanValidator and returns an error suitable for
// ExecuteStage's validation callback.
//
// Performance contract: <10ms (delegated to existing validator)
func ValidatePlanSchema(specDir string) error {
	planPath := filepath.Join(specDir, "plan.yaml")
	validator := &validation.PlanValidator{}
	result := validator.Validate(planPath)

	if result.Valid {
		return nil
	}

	return formatValidationErrors("plan.yaml", result.Errors)
}

// ValidateTasksSchema validates a tasks.yaml file against its full schema.
// It wraps the existing TasksValidator and returns an error suitable for
// ExecuteStage's validation callback.
//
// Performance contract: <10ms (delegated to existing validator)
func ValidateTasksSchema(specDir string) error {
	tasksPath := filepath.Join(specDir, "tasks.yaml")
	validator := &validation.TasksValidator{}
	result := validator.Validate(tasksPath)

	if result.Valid {
		return nil
	}

	return formatValidationErrors("tasks.yaml", result.Errors)
}

// MakeSpecSchemaValidatorWithDetection creates a validation function that first
// detects the current spec directory, then validates spec.yaml against its schema.
// This is necessary for the specify stage where the spec name is not known until
// after Claude creates it.
//
// The returned function ignores the specDir parameter passed by ExecuteStage
// (which is empty for specify) and instead detects the spec directory dynamically.
func MakeSpecSchemaValidatorWithDetection(specsDir string) func(string) error {
	return func(_ string) error {
		// Detect the newly created spec directory
		metadata, err := spec.DetectCurrentSpec(specsDir)
		if err != nil {
			return fmt.Errorf("detecting spec for validation: %w", err)
		}

		// Validate the detected spec directory
		return ValidateSpecSchema(metadata.Directory)
	}
}

// formatValidationErrors formats a list of validation errors into a single error.
// The error message is formatted for inclusion in retry context.
func formatValidationErrors(artifactName string, validationErrs []*validation.ValidationError) error {
	if len(validationErrs) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("schema validation failed for %s:\n", artifactName))

	for _, err := range validationErrs {
		sb.WriteString(fmt.Sprintf("- %s\n", err.Error()))
	}

	return errors.New(sb.String())
}
