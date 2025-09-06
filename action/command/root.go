package command

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/bytebase/bytebase/action/command/cloud"
	"github.com/bytebase/bytebase/action/command/validation"
	"github.com/bytebase/bytebase/action/world"
)

func NewRootCommand(w *world.World) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "bytebase-action",
		Short:             "Bytebase action",
		PersistentPreRunE: rootPreRun(w),
		// XXX: PersistentPostRunE is not called when the command fails
		// So we call it manually in the commands
	}
	// bytebase-action flags
	cmd.PersistentFlags().StringVar(&w.Output, "output", "", "Output file location. The output file is a JSON file with the created resource names")
	cmd.PersistentFlags().StringVar(&w.URL, "url", "https://demo.bytebase.com", "Bytebase URL")
	cmd.PersistentFlags().StringVar(&w.ServiceAccount, "service-account", "api@service.bytebase.com", "Bytebase Service account")
	cmd.PersistentFlags().StringVar(&w.ServiceAccountSecret, "service-account-secret", "", "Bytebase Service account secret")
	cmd.PersistentFlags().StringVar(&w.Project, "project", "projects/hr", "Bytebase project")
	cmd.PersistentFlags().StringSliceVar(&w.Targets, "targets", []string{"instances/test-sample-instance/databases/hr_test", "instances/prod-sample-instance/databases/hr_prod"}, "Bytebase targets. Either one or more databases or a single databaseGroup")
	cmd.PersistentFlags().StringVar(&w.FilePattern, "file-pattern", "", "File pattern to glob migration files")
	cmd.PersistentFlags().BoolVar(&w.Declarative, "declarative", false, "Whether to use declarative mode. (experimental)")

	cmd.AddCommand(NewCheckCommand(w))
	cmd.AddCommand(NewRolloutCommand(w))
	return cmd
}

func rootPreRun(w *world.World) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		w.Logger = slog.New(slog.NewTextHandler(cmd.ErrOrStderr(), nil))

		// Validate all flags and environment variables
		if err := validation.ValidateFlags(w); err != nil {
			return errors.Wrapf(err, "failed to validate flags")
		}

		// Special handling for Bytebase cloud URLs (*.us-central1.bytebase.com)
		if err := cloud.EnsureWorkspaceAwake(w); err != nil {
			return errors.Wrapf(err, "failed to ensure workspace awake")
		}

		return nil
	}
}

func CheckVersionCompatibility(w *world.World, client *Client, cliVersion string) {
	if cliVersion == "unknown" {
		w.Logger.Warn("CLI version unknown, unable to check compatibility")
		return
	}

	actuatorInfo, err := client.GetActuatorInfo(context.Background())
	if err != nil {
		w.Logger.Warn("Unable to get server version for compatibility check", "error", err)
		return
	}

	serverVersion := actuatorInfo.Version
	if serverVersion == "" {
		w.Logger.Warn("Server version is empty, unable to check compatibility")
		return
	}

	if cliVersion == "latest" {
		w.Logger.Warn("Using 'latest' CLI version. It is recommended to use a specific version like bytebase-action:" + serverVersion + " to match your Bytebase server version " + serverVersion)
		return
	}

	if cliVersion != serverVersion {
		w.Logger.Warn("CLI version mismatch", "cliVersion", cliVersion, "serverVersion", serverVersion, "recommendation", "use bytebase-action:"+serverVersion+" to match your Bytebase server")
	} else {
		w.Logger.Info("CLI version matches server version", "version", cliVersion)
	}
}
