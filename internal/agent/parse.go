package agent

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// planYAML represents the structure of a plan.yaml file.
// We only parse the fields we need for agent context updates.
type planYAML struct {
	Plan struct {
		Branch string `yaml:"branch"`
	} `yaml:"plan"`
	TechnicalContext struct {
		Language            string `yaml:"language"`
		Storage             string `yaml:"storage"`
		ProjectType         string `yaml:"project_type"`
		PrimaryDependencies []struct {
			Name    string `yaml:"name"`
			Version string `yaml:"version"`
		} `yaml:"primary_dependencies"`
	} `yaml:"technical_context"`
}

// ParsePlanData reads a plan.yaml file and extracts technology information.
// It returns the extracted PlanData or an error if the file cannot be read or parsed.
func ParsePlanData(planPath string) (*PlanData, error) {
	// Check if file exists
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plan.yaml not found at %s: run 'autospec plan' first to generate it", planPath)
	}

	// Read the file
	data, err := os.ReadFile(planPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan.yaml: %w", err)
	}

	// Parse YAML
	var plan planYAML
	if err := yaml.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan.yaml: %w", err)
	}

	// Extract framework from first primary dependency
	var framework string
	if len(plan.TechnicalContext.PrimaryDependencies) > 0 {
		dep := plan.TechnicalContext.PrimaryDependencies[0]
		if dep.Version != "" {
			framework = fmt.Sprintf("%s %s", dep.Name, dep.Version)
		} else {
			framework = dep.Name
		}
	}

	return &PlanData{
		Language:    plan.TechnicalContext.Language,
		Framework:   framework,
		Database:    plan.TechnicalContext.Storage,
		ProjectType: plan.TechnicalContext.ProjectType,
		Branch:      plan.Plan.Branch,
	}, nil
}

// GetTechnologies returns a slice of non-empty technology strings from PlanData.
// This is useful for adding to the Active Technologies section.
func (p *PlanData) GetTechnologies() []string {
	var techs []string

	if p.Language != "" {
		techs = append(techs, p.Language)
	}
	if p.Framework != "" {
		techs = append(techs, p.Framework)
	}
	if p.Database != "" && p.Database != "None" {
		techs = append(techs, p.Database)
	}
	if p.ProjectType != "" {
		techs = append(techs, fmt.Sprintf("Project Type: %s", p.ProjectType))
	}

	return techs
}

// GetChangeEntry returns a formatted change entry for the Recent Changes section.
func (p *PlanData) GetChangeEntry() string {
	if p.Branch == "" {
		return ""
	}
	return fmt.Sprintf("%s: Added from plan.yaml", p.Branch)
}
