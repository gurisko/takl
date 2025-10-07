//go:build unix

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/gurisko/takl/internal/apiclient"
	"github.com/spf13/cobra"
)

type listResp struct {
	Projects []struct {
		ID           string    `json:"id"`
		Name         string    `json:"name"`
		Path         string    `json:"path"`
		RegisteredAt time.Time `json:"registered_at"`
	} `json:"projects"`
}

var listJSON bool

var projectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registered projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := apiclient.New()
		var out listResp
		if err := c.GetJSON(cmd.Context(), "/api/projects", &out); err != nil {
			return err
		}
		if listJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}
		if len(out.Projects) == 0 {
			fmt.Println("No projects registered")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tPATH\tREGISTERED")
		for _, p := range out.Projects {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.ID, p.Name, p.Path, p.RegisteredAt.Format(time.RFC3339))
		}
		return w.Flush()
	},
}

func init() {
	projectsCmd.AddCommand(projectsListCmd)
	projectsListCmd.Flags().BoolVar(&listJSON, "json", false, "print JSON")
}
