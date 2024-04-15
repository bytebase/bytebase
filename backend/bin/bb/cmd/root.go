// Package cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/bytebase/bytebase/backend/bin/bb/config"

	"github.com/bytebase/bytebase/backend/bin/bb/cmd/projects"
	"github.com/bytebase/bytebase/backend/bin/bb/cmd/version"
	"github.com/bytebase/bytebase/backend/bin/bb/util"
)

func runCmd(ctx context.Context) error {
	rootCmd := &cobra.Command{
		Use:   "bb",
		Short: "CLI for Bytebase",
		Long:  "bb is the CLI interacting with https://api.bytebase.com.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Usage()
		},
	}

	cfg, err := config.New()
	if err != nil {
		return err
	}

	rootCmd.PersistentFlags().StringVar(&cfg.URL,
		"url", "", "The URL for the Bytebase instance.")
	rootCmd.PersistentFlags().StringVar(&cfg.Token,
		"token", "", "The API token to interact with the Bytebase API.")

	s := &util.Setting{
		Config: cfg,
	}

	rootCmd.AddCommand(version.Cmd(), projects.Cmd(s))

	return rootCmd.ExecuteContext(ctx)
}

// Execute is the execute command for root command.
func Execute(ctx context.Context) int {
	err := runCmd(ctx)
	if err == nil {
		return 0
	}
	return 1
}
