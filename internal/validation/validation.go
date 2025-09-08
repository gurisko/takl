package validation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/takl/takl/internal/domain"
	"github.com/takl/takl/internal/workflow"
)

// Validator provides centralized validation for all issue operations
// This is the SINGLE source of truth for all business rules
type Validator struct {
	workflow *workflow.Workflow
}

// NewValidator creates a new centralized validator
func NewValidator(w *workflow.Workflow) *Validator {
	return &Validator{
		workflow: w,
	}
}

// ValidateIssue performs complete validation on an issue
// This is called by ALL entry points: CLI, Web UI, File Watcher
func (v *Validator) ValidateIssue(issue *domain.Issue) error {
	// Basic field validation
	if err := v.validateRequiredFields(issue); err != nil {
		return err
	}

	// Type validation
	if err := v.validateType(issue.Type); err != nil {
		return err
	}

	// Priority validation
	if err := v.validatePriority(issue.Priority); err != nil {
		return err
	}

	// Status validation
	if err := v.validateStatus(issue.Status); err != nil {
		return err
	}

	// Title validation
	if err := v.validateTitle(issue.Title); err != nil {
		return err
	}

	// Assignee validation (if present)
	if issue.Assignee != "" {
		if err := v.validateAssignee(issue.Assignee); err != nil {
			return err
		}
	}

	// Labels validation
	if err := v.validateLabels(issue.Labels); err != nil {
		return err
	}

	return nil
}

// ValidateTransition validates a state transition
func (v *Validator) ValidateTransition(issue *domain.Issue, from, to string) error {
	if v.workflow == nil {
		// No workflow means any transition is valid
		return nil
	}

	// Use the workflow's validation
	return v.workflow.ValidateTransition(context.TODO(), issue, from, to)
}

// ValidateCreate validates an issue for creation
func (v *Validator) ValidateCreate(issue *domain.Issue) error {
	// Set defaults for creation
	if issue.Status == "" {
		issue.Status = "open"
	}
	if issue.Priority == "" {
		issue.Priority = "medium"
	}
	if issue.Created.IsZero() {
		issue.Created = time.Now()
	}
	if issue.Updated.IsZero() {
		issue.Updated = issue.Created
	}

	return v.ValidateIssue(issue)
}

// ValidateUpdate validates an issue for update
func (v *Validator) ValidateUpdate(oldIssue, newIssue *domain.Issue) error {
	// Validate the new state
	if err := v.ValidateIssue(newIssue); err != nil {
		return err
	}

	// If status changed, validate the transition
	if oldIssue.Status != newIssue.Status {
		if err := v.ValidateTransition(newIssue, oldIssue.Status, newIssue.Status); err != nil {
			return err
		}
	}

	// Update timestamp
	newIssue.Updated = time.Now()

	return nil
}

// Private validation methods

func (v *Validator) validateRequiredFields(issue *domain.Issue) error {
	if issue.ID == "" {
		return fmt.Errorf("issue ID is required")
	}
	if issue.Title == "" {
		return fmt.Errorf("issue title is required")
	}
	if issue.Type == "" {
		return fmt.Errorf("issue type is required")
	}
	return nil
}

func (v *Validator) validateType(issueType string) error {
	validTypes := []string{"bug", "feature", "task", "epic"}
	issueType = strings.ToLower(strings.TrimSpace(issueType))

	for _, valid := range validTypes {
		if issueType == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid issue type '%s'. Valid types are: %v", issueType, validTypes)
}

func (v *Validator) validatePriority(priority string) error {
	validPriorities := []string{"low", "medium", "high", "critical"}
	priority = strings.ToLower(strings.TrimSpace(priority))

	for _, valid := range validPriorities {
		if priority == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid priority '%s'. Valid priorities are: %v", priority, validPriorities)
}

func (v *Validator) validateStatus(status string) error {
	if v.workflow != nil {
		// Validate against workflow states
		states := v.workflow.GetStates()
		status = strings.ToLower(strings.TrimSpace(status))

		for _, state := range states {
			if status == state.Key {
				return nil
			}
		}

		validStates := make([]string, len(states))
		for i, s := range states {
			validStates[i] = s.Key
		}
		return fmt.Errorf("invalid status '%s'. Valid states are: %v", status, validStates)
	}

	// Default validation if no workflow
	validStatuses := []string{"open", "in_progress", "done", "closed"}
	status = strings.ToLower(strings.TrimSpace(status))

	for _, valid := range validStatuses {
		if status == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid status '%s'. Valid statuses are: %v", status, validStatuses)
}

func (v *Validator) validateTitle(title string) error {
	title = strings.TrimSpace(title)

	if len(title) < 5 {
		return fmt.Errorf("title must be at least 5 characters long")
	}

	if len(title) > 200 {
		return fmt.Errorf("title must be less than 200 characters")
	}

	return nil
}

func (v *Validator) validateAssignee(assignee string) error {
	assignee = strings.TrimSpace(assignee)

	if assignee == "" {
		return nil // Empty assignee is valid
	}

	// If it contains @, validate as email
	if strings.Contains(assignee, "@") {
		// Basic email validation - must have @ and at least one dot after @
		parts := strings.Split(assignee, "@")
		if len(parts) != 2 {
			return fmt.Errorf("invalid email format: multiple @ symbols")
		}
		if parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid email format: missing local or domain part")
		}
		// Domain should have at least one dot
		if !strings.Contains(parts[1], ".") {
			return fmt.Errorf("invalid email format: domain must contain a dot")
		}
	} else {
		// If not an email, check if it's a valid username
		for _, r := range assignee {
			if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') || r == '-' || r == '_') {
				return fmt.Errorf("assignee must be an email or valid username (alphanumeric, dash, underscore only)")
			}
		}
	}

	return nil
}

func (v *Validator) validateLabels(labels []string) error {
	for _, label := range labels {
		label = strings.TrimSpace(label)

		if label == "" {
			return fmt.Errorf("empty labels are not allowed")
		}

		if len(label) > 50 {
			return fmt.Errorf("label '%s' exceeds maximum length of 50 characters", label)
		}

		// Check for invalid characters
		for _, r := range label {
			if r == ' ' || r == '\t' || r == '\n' {
				return fmt.Errorf("label '%s' contains whitespace", label)
			}
		}
	}

	return nil
}

// NormalizeIssue normalizes issue fields to canonical form
// This is also centralized to ensure consistency
func (v *Validator) NormalizeIssue(issue *domain.Issue) {
	// Normalize type
	issue.Type = strings.ToLower(strings.TrimSpace(issue.Type))

	// Normalize status
	issue.Status = strings.ToLower(strings.TrimSpace(issue.Status))
	issue.Status = normalizeStatus(issue.Status)

	// Normalize priority
	issue.Priority = strings.ToLower(strings.TrimSpace(issue.Priority))

	// Normalize title
	issue.Title = strings.TrimSpace(issue.Title)

	// Normalize assignee
	issue.Assignee = strings.TrimSpace(issue.Assignee)

	// Normalize labels
	for i, label := range issue.Labels {
		issue.Labels[i] = strings.TrimSpace(label)
	}

	// Ensure ID format
	if issue.ID != "" && !strings.HasPrefix(strings.ToUpper(issue.ID), "ISS-") {
		issue.ID = fmt.Sprintf("ISS-%s", strings.ToUpper(issue.ID))
	} else if issue.ID != "" {
		issue.ID = strings.ToUpper(issue.ID)
	}
}

// normalizeStatus handles common status aliases
var statusAliases = map[string]string{
	"in-progress": "in_progress",
	"inprogress":  "in_progress",
	"doing":       "in_progress",
	"wip":         "in_progress",
	"completed":   "done",
	"closed":      "done",
	"resolved":    "done",
	"finished":    "done",
	"todo":        "open",
	"new":         "open",
	"created":     "open",
	"backlog":     "open",
}

func normalizeStatus(status string) string {
	if normalized, ok := statusAliases[status]; ok {
		return normalized
	}
	return status
}
