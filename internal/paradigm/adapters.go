package paradigm

import (
	"context"
	"io"

	"github.com/spf13/cobra"
)

// CLI integration types
type CLIFlag struct {
	Name      string
	Shorthand string
	Value     any
	Usage     string
	Required  bool
}

type CLICommand struct {
	Use   string
	Short string
	Long  string
	RunE  func(ctx context.Context, args []string, out io.Writer) error
	Flags []CLIFlag
}

// ToCobra converts a paradigm CLICommand to a Cobra command
func ToCobra(cmd *CLICommand) *cobra.Command {
	cobraCmd := &cobra.Command{
		Use:   cmd.Use,
		Short: cmd.Short,
		Long:  cmd.Long,
		RunE: func(c *cobra.Command, args []string) error {
			return cmd.RunE(c.Context(), args, c.OutOrStdout())
		},
	}

	// Add flags
	for _, flag := range cmd.Flags {
		switch v := flag.Value.(type) {
		case string:
			cobraCmd.Flags().StringP(flag.Name, flag.Shorthand, v, flag.Usage)
		case int:
			cobraCmd.Flags().IntP(flag.Name, flag.Shorthand, v, flag.Usage)
		case bool:
			cobraCmd.Flags().BoolP(flag.Name, flag.Shorthand, v, flag.Usage)
		}

		if flag.Required {
			_ = cobraCmd.MarkFlagRequired(flag.Name)
		}
	}

	return cobraCmd
}

// Board configuration for UI integration
type BoardConfig struct {
	Lanes   []BoardLane `json:"lanes"`
	Actions []string    `json:"actions"`
	Layout  string      `json:"layout"` // "swimlanes" | "columns"
}

type BoardLane struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	StateKey string `json:"state_key"`
	WIPLimit int    `json:"wip_limit,omitempty"`
	Color    string `json:"color,omitempty"`
}

// HTTP/API integration helpers
type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Paradigm extension interface for optional features
type CLIProvider interface {
	GetCLICommands() []*CLICommand
}

type BoardProvider interface {
	GetBoardConfiguration() *BoardConfig
}

// Helper to check if paradigm supports optional interfaces
func SupportsCLI(p Paradigm) bool {
	_, ok := p.(CLIProvider)
	return ok
}

func SupportsBoard(p Paradigm) bool {
	_, ok := p.(BoardProvider)
	return ok
}
