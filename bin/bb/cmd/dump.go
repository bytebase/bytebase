// cmd is the command surface of Bytebase bb tool provided by bytebase.com.
package cmd

import (
	"github.com/bytebase/bb/dump"
	"github.com/spf13/cobra"
)

func init() {
	dumpCmd.Flags().StringVar(&databaseType, "database-type", "mysql", "Database type such as `mysql`.")
	dumpCmd.Flags().StringVar(&username, "username", "root", "User name to login database.")
	dumpCmd.Flags().StringVar(&password, "password", "", "Password to login database.")
	dumpCmd.Flags().StringVar(&hostname, "hostname", "localhost", "Hostname of database.")
	dumpCmd.Flags().StringVar(&port, "port", "3306", "port of database.")
	dumpCmd.Flags().StringVar(&directory, "directory", "", "directory to dump baselines; output to stdout if unspecified.")

	rootCmd.AddCommand(dumpCmd)
}

var (
	databaseType string
	username     string
	password     string
	hostname     string
	port         string
	directory    string

	dumpCmd = &cobra.Command{
		Use:   "dump",
		Short: "Exports the schema of a database instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			return dump.Dump(databaseType, username, password, hostname, port, directory)
		},
	}
)
