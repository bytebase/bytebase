// Package cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// NewRootCmd creates the root command.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "bb",
		Short: "A database management tool provided by bytebase.com",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Usage()
		},
	}

	rootCmd.AddCommand(newDumpCmd(), newRestoreCmd(), newVersionCmd(), newMigrateCmd())

	return rootCmd
}

// Execute is the execute command for root command.
func Execute() error {
	logConfig := zap.NewProductionConfig()
	// Always set encoding to "console" for now since we do not redirect to file.
	logConfig.Encoding = "console"
	// "console" encoding needs to use the corresponding development encoder config.
	logConfig.EncoderConfig = zap.NewDevelopmentEncoderConfig()
	logConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	myLogger, err := logConfig.Build()
	if err != nil {
		panic(fmt.Errorf("failed to create logger. %w", err))
	}
	logger = myLogger

	return NewRootCmd().Execute()
}
