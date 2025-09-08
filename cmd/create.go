package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/takl/takl/internal/prompt"
	"github.com/takl/takl/internal/store"
	"github.com/takl/takl/sdk"
)

var createCmd = &cobra.Command{
	Use:   "create [type]",
	Short: "Create a new issue",
	Long: `Create a new issue with the specified type.

Types: bug, feature, task, epic

Usage modes:
1. Git-style with -m flag:
   takl create bug -m "Button not working"
   takl create feature -m "Add dark mode" --assignee=john@example.com
   
2. Interactive mode (no -m flag):
   takl create bug
   (prompts for title, description, assignee, etc.)
   
3. Legacy positional arguments (still supported):
   takl create bug "Button not working"`,
	Args: cobra.RangeArgs(0, 2),
	RunE: createIssue,
}

var (
	message  string // -m flag for title (git-style)
	assignee string
	priority string
	labels   []string
	content  string
)

func init() {
	rootCmd.AddCommand(createCmd)

	createCmd.Flags().StringVarP(&message, "message", "m", "",
		"issue title/message (git-style)")
	createCmd.Flags().StringVar(&assignee, "assignee", "",
		"assign issue to user")
	createCmd.Flags().StringVar(&priority, "priority", "medium",
		"issue priority (low, medium, high, critical)")
	createCmd.Flags().StringSliceVar(&labels, "labels", []string{},
		"issue labels")
	createCmd.Flags().StringVar(&content, "content", "",
		"issue content/description")
}

func createIssue(cmd *cobra.Command, args []string) error {
	var issueType string
	var title string

	// If no arguments, start full interactive mode
	if len(args) == 0 {
		return createIssueFullyInteractive()
	}

	issueType = args[0]

	// Determine the mode and get title
	if message != "" {
		// Git-style mode: -m flag provided
		title = message
	} else if len(args) == 2 {
		// Legacy mode: positional argument provided
		title = args[1]
	} else {
		// Interactive mode: prompt for all details
		return createIssueInteractive(issueType, prompt.NewStdPrompter())
	}

	// Create issue using provided flags
	return createIssueWithOptions(issueType, title)
}

func createIssueFullyInteractive() error {
	prompter := prompt.NewStdPrompter()
	// Select issue type
	issueType, err := prompter.SelectFromOptions("Select issue type:", store.ValidIssueTypes, store.DefaultIssueType)
	if err != nil {
		return err
	}

	return createIssueInteractive(issueType, prompter)
}

func createIssueInteractive(issueType string, prompter prompt.Prompter) error {
	ctx, err := RequireProjectContext()
	if err != nil {
		return err
	}

	// Show project context
	fmt.Printf("Creating a new %s issue in: %s\n", issueType, ctx.GetProjectInfo())

	// Title prompting (NO validation - API validates)
	title := prompter.PromptWithDefault("Title", "")
	if title != "" {
		fmt.Printf("✅ Title: %s\n", title)
	}

	// Enhanced description prompting
	fmt.Println("\n📝 Description (enter multiple lines, press Enter twice when done):")
	var descriptionLines []string
	for {
		line := prompter.PromptWithDefault("", "")
		if line == "" {
			break
		}
		descriptionLines = append(descriptionLines, line)
	}
	description := strings.Join(descriptionLines, "\n")

	// Show content preview if not empty
	if description != "" {
		fmt.Printf("📄 Description preview:\n%s\n\n", description)
	}

	// Select priority with enhanced display
	fmt.Printf("🎯 Priority levels:\n")
	fmt.Printf("   low      - Minor issues, nice-to-have features\n")
	fmt.Printf("   medium   - Normal development work (default)\n")
	fmt.Printf("   high     - Important features, significant bugs\n")
	fmt.Printf("   critical - Urgent fixes, blockers\n")
	priorityInput, err := prompter.SelectFromOptions("Select priority:", store.ValidPriorities, store.DefaultPriority)
	if err != nil {
		return err
	}
	fmt.Printf("✅ Priority: %s\n", priorityInput)

	// Enhanced assignee prompting with email validation
	var assigneeInput string
	for {
		assigneeInput = prompter.PromptWithDefault("👤 Assignee email (optional)", "")
		if assigneeInput == "" {
			break // Optional field
		}
		if isValidEmail(assigneeInput) {
			fmt.Printf("✅ Assignee: %s\n", assigneeInput)
			break
		} else {
			fmt.Println("❌ Please enter a valid email address")
		}
	}

	// Enhanced labels prompting with suggestions
	fmt.Printf("🏷️  Common labels: backend, frontend, ui, api, bug, enhancement, documentation\n")
	labelsInput := prompter.PromptWithDefault("Labels (comma-separated, optional)", "")
	var labelsList []string
	if labelsInput != "" {
		labelsList = strings.Split(labelsInput, ",")
		for i, label := range labelsList {
			labelsList[i] = strings.TrimSpace(label)
		}
		// Remove empty labels
		filteredLabels := make([]string, 0, len(labelsList))
		for _, label := range labelsList {
			if label != "" {
				filteredLabels = append(filteredLabels, label)
			}
		}
		labelsList = filteredLabels
		if len(labelsList) > 0 {
			fmt.Printf("✅ Labels: %s\n", strings.Join(labelsList, ", "))
		}
	}

	// Show summary and confirm creation
	fmt.Printf("\n📋 Issue Summary:\n")
	fmt.Printf("   Type: %s\n", issueType)
	fmt.Printf("   Title: %s\n", title)
	if description != "" {
		preview := description
		if len(preview) > 60 {
			preview = preview[:60] + "..."
		}
		fmt.Printf("   Description: %s\n", preview)
	}
	fmt.Printf("   Priority: %s\n", priorityInput)
	if assigneeInput != "" {
		fmt.Printf("   Assignee: %s\n", assigneeInput)
	}
	if len(labelsList) > 0 {
		fmt.Printf("   Labels: %s\n", strings.Join(labelsList, ", "))
	}

	// Confirmation prompt
	confirmCreate := prompter.PromptWithDefault("\n✅ Create this issue? (Y/n)", "Y")
	if !strings.HasPrefix(strings.ToLower(confirmCreate), "y") && confirmCreate != "" {
		fmt.Println("❌ Issue creation cancelled")
		return nil
	}

	// Create SDK client
	client := sdkClient()

	// Create issue via SDK
	req := sdk.CreateIssueRequest{
		Type:     issueType,
		Title:    title,
		Content:  description,
		Priority: priorityInput,
		Assignee: assigneeInput,
		Labels:   labelsList,
	}

	fmt.Println("⏳ Creating issue...")
	issue, err := client.CreateIssue(ctx.GetProjectID(), req)
	if err != nil {
		return fmt.Errorf("❌ Failed to create issue: %w", err)
	}

	fmt.Printf("\n🎉 Successfully created issue %s: %s\n", issue.ID, issue.Title)
	fmt.Printf("📁 File: %s\n", issue.FilePath)
	if verbose {
		fmt.Printf("🔧 Type: %s\n", issue.Type)
		fmt.Printf("⭐ Priority: %s\n", issue.Priority)
		if issue.Assignee != "" {
			fmt.Printf("👤 Assignee: %s\n", issue.Assignee)
		}
		if len(issue.Labels) > 0 {
			fmt.Printf("🏷️  Labels: %s\n", strings.Join(issue.Labels, ", "))
		}
	}

	return nil
}

func createIssueWithOptions(issueType, title string) error {
	ctx, err := RequireProjectContext()
	if err != nil {
		return err
	}

	// Create SDK client
	client := sdkClient()

	// Create issue via SDK
	req := sdk.CreateIssueRequest{
		Type:     issueType,
		Title:    title,
		Content:  content,
		Priority: priority,
		Assignee: assignee,
		Labels:   labels,
	}

	issue, err := client.CreateIssue(ctx.GetProjectID(), req)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	fmt.Printf("Created issue %s: %s in %s\n", issue.ID, issue.Title, ctx.GetProjectInfo())
	if verbose {
		fmt.Printf("File: %s\n", issue.FilePath)
		fmt.Printf("Type: %s\n", issue.Type)
		fmt.Printf("Priority: %s\n", issue.Priority)
		if issue.Assignee != "" {
			fmt.Printf("Assignee: %s\n", issue.Assignee)
		}
	}

	return nil
}

// emailRx is a pragmatic email validation pattern that allows most valid addresses
// while rejecting obvious invalids. This is best-effort validation - final delivery
// guarantees rely on SMTP acceptance.
var emailRx = regexp.MustCompile(`^[A-Za-z0-9._%+\-']+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)

// isValidEmail validates an email address using a pragmatic regex pattern.
// Note: Email validation is best-effort; final delivery guarantees rely on SMTP acceptance.
func isValidEmail(email string) bool {
	return emailRx.MatchString(email)
}
