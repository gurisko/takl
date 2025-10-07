//go:build unix

package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/gurisko/takl/internal/apiclient"
	"github.com/spf13/cobra"
)

var rmJSON bool
var rmYes bool

var projectsRemoveCmd = &cobra.Command{
	Use:   "remove <project-id>",
	Short: "Remove a project by id",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := apiclient.New()
		id := strings.TrimSpace(args[0])

		// refuse to prompt on non-tty unless -y
		if !rmYes && !rmJSON {
			if fi, _ := os.Stdin.Stat(); (fi.Mode() & os.ModeCharDevice) == 0 {
				return errors.New("refusing to prompt on non-interactive stdin; use -y to confirm")
			}
			fmt.Printf("Remove project %s? [y/N]: ", id)
			reader := bufio.NewReader(os.Stdin)
			ans, _ := reader.ReadString('\n')
			ans = strings.ToLower(strings.TrimSpace(ans))
			if ans != "y" && ans != "yes" {
				fmt.Println("aborted")
				return nil
			}
		}

		if err := c.Delete(cmd.Context(), "/api/projects/"+url.PathEscape(id)); err != nil {
			return err
		}

		if rmJSON {
			// API returns 204; supply a tiny confirmation object for scripting
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]any{"removed": true, "id": id})
		}
		fmt.Println("Removed", id)
		return nil
	},
}

func init() {
	projectsCmd.AddCommand(projectsRemoveCmd)
	projectsRemoveCmd.Flags().BoolVarP(&rmYes, "yes", "y", false, "assume yes")
	projectsRemoveCmd.Flags().BoolVar(&rmJSON, "json", false, "print JSON")
}
