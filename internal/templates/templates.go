package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// Template represents an issue template
type Template struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description"`
	Type        string              `yaml:"type"`       // bug, feature, task, epic
	Priority    string              `yaml:"priority"`   // low, medium, high, critical
	Labels      []string            `yaml:"labels"`     // default labels to apply
	Assignee    string              `yaml:"assignee"`   // default assignee
	Content     string              `yaml:"content"`    // template content with placeholders
	Fields      []TemplateField     `yaml:"fields"`     // custom fields to prompt for
	Validation  *TemplateValidation `yaml:"validation"` // validation rules
}

// TemplateField represents a field that should be prompted for when using template
type TemplateField struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Type        string   `yaml:"type"` // text, textarea, select, checkbox
	Required    bool     `yaml:"required"`
	Default     string   `yaml:"default"`
	Options     []string `yaml:"options"`    // for select fields
	Validation  string   `yaml:"validation"` // regex pattern for validation
}

// TemplateValidation represents validation rules for the template
type TemplateValidation struct {
	RequiredLabels []string `yaml:"required_labels"` // labels that must be present
	ForbiddenWords []string `yaml:"forbidden_words"` // words not allowed in content
	MinLength      int      `yaml:"min_length"`      // minimum content length
	MaxLength      int      `yaml:"max_length"`      // maximum content length
}

// Manager manages issue templates
type Manager struct {
	templateDir string
	templates   map[string]*Template
}

// NewManager creates a new template manager
func NewManager(projectPath string) (*Manager, error) {
	templateDir := filepath.Join(projectPath, ".takl", "templates")

	// Create templates directory if it doesn't exist
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create templates directory: %w", err)
	}

	manager := &Manager{
		templateDir: templateDir,
		templates:   make(map[string]*Template),
	}

	// Load existing templates
	if err := manager.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Create default templates if none exist
	if len(manager.templates) == 0 {
		if err := manager.createDefaultTemplates(); err != nil {
			return nil, fmt.Errorf("failed to create default templates: %w", err)
		}
	}

	return manager, nil
}

// loadTemplates loads all template files from the templates directory
func (m *Manager) loadTemplates() error {
	entries, err := os.ReadDir(m.templateDir)
	if err != nil {
		return nil // Directory might not exist yet
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			templatePath := filepath.Join(m.templateDir, entry.Name())
			template, err := m.loadTemplate(templatePath)
			if err != nil {
				// Skip invalid templates but log the error
				fmt.Printf("Warning: Failed to load template %s: %v\n", entry.Name(), err)
				continue
			}

			templateName := strings.TrimSuffix(entry.Name(), ".yaml")
			m.templates[templateName] = template
		}
	}

	return nil
}

// loadTemplate loads a single template file
func (m *Manager) loadTemplate(path string) (*Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var template Template
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}

	return &template, nil
}

// createDefaultTemplates creates a set of default templates
func (m *Manager) createDefaultTemplates() error {
	defaultTemplates := m.getDefaultTemplates()

	for name, template := range defaultTemplates {
		templatePath := filepath.Join(m.templateDir, name+".yaml")

		data, err := yaml.Marshal(template)
		if err != nil {
			return fmt.Errorf("failed to marshal template %s: %w", name, err)
		}

		if err := os.WriteFile(templatePath, data, 0644); err != nil {
			return fmt.Errorf("failed to write template %s: %w", name, err)
		}

		m.templates[name] = template
	}

	return nil
}

// getDefaultTemplates returns a set of default templates
func (m *Manager) getDefaultTemplates() map[string]*Template {
	return map[string]*Template{
		"bug-report": {
			Name:        "Bug Report",
			Description: "Report a bug or issue",
			Type:        "bug",
			Priority:    "medium",
			Labels:      []string{"bug"},
			Content: `## Bug Description
{{.bug_description}}

## Steps to Reproduce
{{.steps_to_reproduce}}

## Expected Behavior
{{.expected_behavior}}

## Actual Behavior
{{.actual_behavior}}

## Environment
- OS: {{.operating_system}}
- Browser/Version: {{.browser_version}}
- Additional context: {{.additional_context}}

## Screenshots
{{.screenshots}}`,
			Fields: []TemplateField{
				{
					Name:        "bug_description",
					Description: "Brief description of the bug",
					Type:        "text",
					Required:    true,
				},
				{
					Name:        "steps_to_reproduce",
					Description: "Steps to reproduce the issue",
					Type:        "textarea",
					Required:    true,
				},
				{
					Name:        "expected_behavior",
					Description: "What should happen?",
					Type:        "textarea",
					Required:    true,
				},
				{
					Name:        "actual_behavior",
					Description: "What actually happens?",
					Type:        "textarea",
					Required:    true,
				},
				{
					Name:        "operating_system",
					Description: "Operating system",
					Type:        "select",
					Required:    false,
					Options:     []string{"Windows", "macOS", "Linux", "iOS", "Android", "Other"},
					Default:     "Linux",
				},
				{
					Name:        "browser_version",
					Description: "Browser and version (if applicable)",
					Type:        "text",
					Required:    false,
				},
				{
					Name:        "additional_context",
					Description: "Any additional context",
					Type:        "textarea",
					Required:    false,
				},
				{
					Name:        "screenshots",
					Description: "Screenshots or error messages",
					Type:        "textarea",
					Required:    false,
					Default:     "N/A",
				},
			},
			Validation: &TemplateValidation{
				RequiredLabels: []string{"bug"},
				MinLength:      50,
			},
		},
		"feature-request": {
			Name:        "Feature Request",
			Description: "Request a new feature or enhancement",
			Type:        "feature",
			Priority:    "medium",
			Labels:      []string{"enhancement"},
			Content: `## Feature Description
{{.feature_description}}

## Problem Statement
{{.problem_statement}}

## Proposed Solution
{{.proposed_solution}}

## Alternatives Considered
{{.alternatives_considered}}

## Success Criteria
{{.success_criteria}}

## Priority Justification
{{.priority_justification}}

## Additional Context
{{.additional_context}}`,
			Fields: []TemplateField{
				{
					Name:        "feature_description",
					Description: "What feature would you like to see?",
					Type:        "text",
					Required:    true,
				},
				{
					Name:        "problem_statement",
					Description: "What problem does this solve?",
					Type:        "textarea",
					Required:    true,
				},
				{
					Name:        "proposed_solution",
					Description: "How should this be implemented?",
					Type:        "textarea",
					Required:    true,
				},
				{
					Name:        "alternatives_considered",
					Description: "What other approaches did you consider?",
					Type:        "textarea",
					Required:    false,
				},
				{
					Name:        "success_criteria",
					Description: "How will we know this is successful?",
					Type:        "textarea",
					Required:    false,
				},
				{
					Name:        "priority_justification",
					Description: "Why should this be prioritized?",
					Type:        "textarea",
					Required:    false,
				},
				{
					Name:        "additional_context",
					Description: "Any additional context or screenshots",
					Type:        "textarea",
					Required:    false,
				},
			},
			Validation: &TemplateValidation{
				RequiredLabels: []string{"enhancement"},
				MinLength:      100,
			},
		},
		"task": {
			Name:        "Task",
			Description: "General work task",
			Type:        "task",
			Priority:    "medium",
			Labels:      []string{"task"},
			Content: `## Task Description
{{.task_description}}

## Acceptance Criteria
{{.acceptance_criteria}}

## Implementation Notes
{{.implementation_notes}}

## Dependencies
{{.dependencies}}

## Estimated Effort
{{.estimated_effort}}`,
			Fields: []TemplateField{
				{
					Name:        "task_description",
					Description: "What needs to be done?",
					Type:        "textarea",
					Required:    true,
				},
				{
					Name:        "acceptance_criteria",
					Description: "How will we know this is done?",
					Type:        "textarea",
					Required:    true,
				},
				{
					Name:        "implementation_notes",
					Description: "Any implementation details or constraints",
					Type:        "textarea",
					Required:    false,
				},
				{
					Name:        "dependencies",
					Description: "What does this depend on?",
					Type:        "textarea",
					Required:    false,
					Default:     "None",
				},
				{
					Name:        "estimated_effort",
					Description: "Estimated effort",
					Type:        "select",
					Required:    false,
					Options:     []string{"Small (< 1 day)", "Medium (1-3 days)", "Large (3-5 days)", "Extra Large (> 5 days)"},
					Default:     "Medium (1-3 days)",
				},
			},
			Validation: &TemplateValidation{
				RequiredLabels: []string{"task"},
				MinLength:      30,
			},
		},
	}
}

// ListTemplates returns all available templates
func (m *Manager) ListTemplates() map[string]*Template {
	return m.templates
}

// GetTemplate returns a template by name
func (m *Manager) GetTemplate(name string) (*Template, bool) {
	template, exists := m.templates[name]
	return template, exists
}

// RenderTemplate renders a template with the provided values
func (m *Manager) RenderTemplate(templateName string, values map[string]string) (string, string, error) {
	tpl, exists := m.templates[templateName]
	if !exists {
		return "", "", fmt.Errorf("template %s not found", templateName)
	}

	// Create template instance
	tmpl, err := template.New("issue").Parse(tpl.Content)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Render template
	var buf strings.Builder
	if err := tmpl.Execute(&buf, values); err != nil {
		return "", "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Generate title from first field or use template name
	title := tpl.Name
	if len(tpl.Fields) > 0 {
		if firstFieldValue, ok := values[tpl.Fields[0].Name]; ok && firstFieldValue != "" {
			title = firstFieldValue
		}
	}

	return title, buf.String(), nil
}

// ValidateTemplate validates template data against template rules
func (m *Manager) ValidateTemplate(template *Template, values map[string]string, content string) error {
	if template.Validation == nil {
		return nil
	}

	validation := template.Validation

	// Check content length
	if validation.MinLength > 0 && len(content) < validation.MinLength {
		return fmt.Errorf("content too short (minimum %d characters)", validation.MinLength)
	}

	if validation.MaxLength > 0 && len(content) > validation.MaxLength {
		return fmt.Errorf("content too long (maximum %d characters)", validation.MaxLength)
	}

	// Check for forbidden words
	contentLower := strings.ToLower(content)
	for _, word := range validation.ForbiddenWords {
		if strings.Contains(contentLower, strings.ToLower(word)) {
			return fmt.Errorf("content contains forbidden word: %s", word)
		}
	}

	return nil
}

// CreateTemplate creates a new custom template
func (m *Manager) CreateTemplate(name string, template *Template) error {
	templatePath := filepath.Join(m.templateDir, name+".yaml")

	data, err := yaml.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	if err := os.WriteFile(templatePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write template file: %w", err)
	}

	m.templates[name] = template
	return nil
}

// DeleteTemplate removes a template
func (m *Manager) DeleteTemplate(name string) error {
	templatePath := filepath.Join(m.templateDir, name+".yaml")

	if err := os.Remove(templatePath); err != nil {
		return fmt.Errorf("failed to delete template file: %w", err)
	}

	delete(m.templates, name)
	return nil
}
