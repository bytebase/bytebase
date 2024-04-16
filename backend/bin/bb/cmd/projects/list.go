package projects

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bytebase/bytebase/backend/bin/bb/util"
)

// ListCmd encapsulates the command for listing backups for a branch.
func listCmd(s *util.Setting) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all projects",
		Aliases: []string{"ls"},
		RunE: func(_ *cobra.Command, _ []string) error {
			fmt.Printf("All projects: token: %s url: %s\n", s.Config.Token, s.Config.URL)
			return nil
		},
	}

	return cmd
}
