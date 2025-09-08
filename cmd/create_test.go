package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/takl/takl/internal/prompt"
)

// validatePriority mimics the validation logic for testing
func validatePriority(input string) (string, error) {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return "medium", nil // Default priority
	}

	validPriorities := []string{"low", "medium", "high", "critical"}
	for _, valid := range validPriorities {
		if input == valid {
			return input, nil
		}
	}

	return "", fmt.Errorf("invalid priority '%s'", input)
}

func TestValidatePriority(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"empty default", "", "medium", false},
		{"low priority", "low", "low", false},
		{"medium priority", "medium", "medium", false},
		{"high priority", "high", "high", false},
		{"critical priority", "critical", "critical", false},
		{"case insensitive", "HIGH", "high", false},
		{"with whitespace", " low ", "low", false},
		{"invalid priority", "invalid", "", true},
		{"numeric input", "1", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := validatePriority(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePriority() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("validatePriority() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPromptSelectFromOptions(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		options       []string
		defaultOption string
		want          string
		wantErr       bool
	}{
		{
			name:          "default selection",
			input:         "\n",
			options:       []string{"low", "medium", "high"},
			defaultOption: "medium",
			want:          "medium",
			wantErr:       false,
		},
		{
			name:          "numeric selection",
			input:         "1\n",
			options:       []string{"low", "medium", "high"},
			defaultOption: "medium",
			want:          "low",
			wantErr:       false,
		},
		{
			name:          "text selection",
			input:         "high\n",
			options:       []string{"low", "medium", "high"},
			defaultOption: "medium",
			want:          "high",
			wantErr:       false,
		},
		{
			name:          "case insensitive text",
			input:         "HIGH\n",
			options:       []string{"low", "medium", "high"},
			defaultOption: "medium",
			want:          "high",
			wantErr:       false,
		},
		{
			name:          "invalid selection",
			input:         "invalid\n",
			options:       []string{"low", "medium", "high"},
			defaultOption: "medium",
			want:          "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader := strings.NewReader(tt.input)
			writer := &bytes.Buffer{}
			prompter := prompt.NewTestPrompter(reader, writer)

			got, err := prompter.SelectFromOptions("Select option:", tt.options, tt.defaultOption)
			if (err != nil) != tt.wantErr {
				t.Errorf("SelectFromOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SelectFromOptions() = %v, want %v", got, tt.want)
			}

			// Check that prompt was displayed
			output := writer.String()
			if !strings.Contains(output, "Select option:") {
				t.Errorf("Expected prompt to be displayed in output: %s", output)
			}
		})
	}
}

func TestPromptWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		prompt       string
		defaultValue string
		want         string
	}{
		{
			name:         "use default",
			input:        "\n",
			prompt:       "Enter value",
			defaultValue: "default",
			want:         "default",
		},
		{
			name:         "custom input",
			input:        "custom\n",
			prompt:       "Enter value",
			defaultValue: "default",
			want:         "custom",
		},
		{
			name:         "empty default",
			input:        "value\n",
			prompt:       "Enter value",
			defaultValue: "",
			want:         "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reader := strings.NewReader(tt.input)
			writer := &bytes.Buffer{}
			prompter := prompt.NewTestPrompter(reader, writer)

			got := prompter.PromptWithDefault(tt.prompt, tt.defaultValue)
			if got != tt.want {
				t.Errorf("PromptWithDefault() = %v, want %v", got, tt.want)
			}

			// Check that prompt was displayed
			output := writer.String()
			if !strings.Contains(output, tt.prompt) {
				t.Errorf("Expected prompt to be displayed in output: %s", output)
			}
		})
	}
}

func TestMockPrompter(t *testing.T) {
	t.Parallel()

	mock := prompt.NewMockPrompter("high", "Test assignee", "bug,urgent")

	// Test SelectFromOptions
	priority, err := mock.SelectFromOptions("Priority:", []string{"low", "medium", "high"}, "medium")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if priority != "high" {
		t.Errorf("Expected 'high', got '%s'", priority)
	}

	// Test PromptWithDefault
	assignee := mock.PromptWithDefault("Assignee:", "")
	if assignee != "Test assignee" {
		t.Errorf("Expected 'Test assignee', got '%s'", assignee)
	}

	labels := mock.PromptWithDefault("Labels:", "")
	if labels != "bug,urgent" {
		t.Errorf("Expected 'bug,urgent', got '%s'", labels)
	}

	// Test default when no more responses
	extra := mock.PromptWithDefault("Extra:", "default")
	if extra != "default" {
		t.Errorf("Expected 'default', got '%s'", extra)
	}
}
