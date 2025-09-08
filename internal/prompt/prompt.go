package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Interface for prompt operations (for testing)
type Prompter interface {
	SelectFromOptions(prompt string, options []string, defaultOption string) (string, error)
	PromptWithDefault(prompt, defaultValue string) string
}

// StdPrompter implements Prompter using stdin/stdout
type StdPrompter struct {
	reader io.Reader
	writer io.Writer
}

// NewStdPrompter creates a prompter using standard input/output
func NewStdPrompter() *StdPrompter {
	return &StdPrompter{
		reader: os.Stdin,
		writer: os.Stdout,
	}
}

// NewTestPrompter creates a prompter for testing with custom reader/writer
func NewTestPrompter(reader io.Reader, writer io.Writer) *StdPrompter {
	return &StdPrompter{
		reader: reader,
		writer: writer,
	}
}

// SelectFromOptions provides a numbered menu selection
func (p *StdPrompter) SelectFromOptions(prompt string, options []string, defaultOption string) (string, error) {
	reader := bufio.NewReader(p.reader)

	fmt.Fprintf(p.writer, "\n%s\n", prompt)
	for i, option := range options {
		marker := " "
		if option == defaultOption {
			marker = "*"
		}
		fmt.Fprintf(p.writer, "%s %d. %s\n", marker, i+1, option)
	}
	fmt.Fprintf(p.writer, "Enter number [%s]: ", defaultOption)

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	// If empty, return default
	if input == "" {
		return defaultOption, nil
	}

	// Try to parse as number
	if len(input) == 1 && input[0] >= '1' && input[0] <= '9' {
		index := int(input[0] - '1')
		if index >= 0 && index < len(options) {
			return options[index], nil
		}
	}

	// Check if input matches an option directly
	for _, option := range options {
		if strings.EqualFold(input, option) {
			return option, nil
		}
	}

	return "", fmt.Errorf("invalid selection. Please choose 1-%d or enter the option name", len(options))
}

// PromptWithDefault prompts for input with a default value
func (p *StdPrompter) PromptWithDefault(prompt, defaultValue string) string {
	reader := bufio.NewReader(p.reader)
	if defaultValue != "" {
		fmt.Fprintf(p.writer, "%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Fprintf(p.writer, "%s: ", prompt)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultValue
	}
	return input
}

// MockPrompter is a simple mock for testing
type MockPrompter struct {
	Responses []string
	index     int
}

// NewMockPrompter creates a mock prompter with predefined responses
func NewMockPrompter(responses ...string) *MockPrompter {
	return &MockPrompter{Responses: responses}
}

// SelectFromOptions returns the next predefined response
func (m *MockPrompter) SelectFromOptions(prompt string, options []string, defaultOption string) (string, error) {
	if m.index >= len(m.Responses) {
		return defaultOption, nil
	}
	response := m.Responses[m.index]
	m.index++

	// If response is empty, return default
	if response == "" {
		return defaultOption, nil
	}

	return response, nil
}

// PromptWithDefault returns the next predefined response
func (m *MockPrompter) PromptWithDefault(prompt, defaultValue string) string {
	if m.index >= len(m.Responses) {
		return defaultValue
	}
	response := m.Responses[m.index]
	m.index++

	// If response is empty, return default
	if response == "" {
		return defaultValue
	}

	return response
}
