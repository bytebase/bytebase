package command

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/bytebase/bytebase/action/common"
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
		if w.ServiceAccountSecret == "" {
			w.ServiceAccountSecret = os.Getenv("BYTEBASE_SERVICE_ACCOUNT_SECRET")
		}

		if w.Platform == world.UnspecifiedPlatform {
			w.Platform = world.GetJobPlatform()
		}

		if w.ServiceAccount == "" {
			return errors.Errorf("service-account is required and cannot be empty")
		}
		if w.ServiceAccountSecret == "" {
			return errors.Errorf("service-account-secret is required and cannot be empty")
		}

		// Validate URL format
		u, err := url.Parse(w.URL)
		if err != nil {
			return errors.Wrapf(err, "invalid URL format: %s", w.URL)
		}
		w.URL = strings.TrimSuffix(u.String(), "/") // update the URL to the canonical form

		// Validate project format
		if !strings.HasPrefix(w.Project, "projects/") {
			return errors.Errorf("invalid project format, must be projects/{project}")
		}

		// Validate targets format
		var databaseTarget, databaseGroupTarget int
		for _, target := range w.Targets {
			if _, _, err := common.GetInstanceDatabaseID(target); err == nil {
				databaseTarget++
			} else if _, _, err := common.GetProjectIDDatabaseGroupID(target); err == nil {
				databaseGroupTarget++
			} else {
				return errors.Errorf("invalid target format, must be instances/{instance}/databases/{database} or projects/{project}/databaseGroups/{databaseGroup}")
			}
		}
		if databaseTarget > 0 && databaseGroupTarget > 0 {
			return errors.Errorf("targets must be either databases or databaseGroups")
		}
		if databaseGroupTarget > 1 {
			return errors.Errorf("targets must be a single databaseGroup")
		}

		return nil
	}
}

func writeOutputJSON(w *world.World) {
	if err := func() error {
		if w.Output == "" {
			return nil
		}

		w.Logger.Info("writing output to file", "file", w.Output)

		// Create parent directory if not exists
		if dir := filepath.Dir(w.Output); dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return errors.Wrapf(err, "failed to create output directory: %s", dir)
			}
		}

		f, err := os.Create(w.Output)
		if err != nil {
			return errors.Wrapf(err, "failed to create output file: %s", w.Output)
		}
		defer f.Close()

		j, err := json.Marshal(w.OutputMap)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal output map")
		}

		if _, err := f.Write(j); err != nil {
			return errors.Wrapf(err, "failed to write output file: %s", w.Output)
		}
		return nil
	}(); err != nil {
		w.Logger.Error("failed to write output JSON", "error", err)
	}
}

func checkVersionCompatibility(w *world.World, client *Client, cliVersion string) {
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
