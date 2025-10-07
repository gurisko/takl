//go:build unix

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gurisko/takl/internal/apiclient"
	"github.com/spf13/cobra"
)

type registerReq struct {
	Name string `json:"name"`
	Path string `json:"path"`
}
type registerResp struct {
	Project struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Path string `json:"path"`
	} `json:"project"`
}

var regName, regPath string
var regJSON bool

var projectsRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a project directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		regName = strings.TrimSpace(regName)
		regPath = strings.TrimSpace(regPath)
		if regName == "" || regPath == "" {
			return errors.New("--name and --path are required")
		}
		// client-side friendliness: expand ~ and make absolute (daemon also validates)
		if strings.HasPrefix(regPath, "~") {
			if home, _ := os.UserHomeDir(); home != "" {
				regPath = filepath.Join(home, strings.TrimPrefix(regPath, "~"))
			}
		}
		if abs, err := filepath.Abs(regPath); err == nil {
			regPath = abs
		}

		c := apiclient.New()
		var out registerResp
		if err := c.PostJSON(cmd.Context(), "/api/projects", registerReq{Name: regName, Path: regPath}, &out); err != nil {
			return err
		}
		if regJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}
		fmt.Printf("Registered %q at %s (id=%s)\n", out.Project.Name, out.Project.Path, out.Project.ID)
		return nil
	},
}

func init() {
	projectsCmd.AddCommand(projectsRegisterCmd)
	projectsRegisterCmd.Flags().StringVarP(&regName, "name", "n", "", "project name (required)")
	projectsRegisterCmd.Flags().StringVarP(&regPath, "path", "p", "", "project path (required)")
	projectsRegisterCmd.Flags().BoolVar(&regJSON, "json", false, "print JSON")
	_ = projectsRegisterCmd.MarkFlagRequired("name")
	_ = projectsRegisterCmd.MarkFlagRequired("path")
}
