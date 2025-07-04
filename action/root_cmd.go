package main

import (
	"encoding/json"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/bytebase/bytebase/action/world"
)

func NewRootCommand(w *world.World) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "bytebase-action",
		Short:              "Bytebase action",
		PersistentPreRunE:  validateSharedFlagsWithWorld(w),
		PersistentPostRunE: writeOutputJSON(w),
	}
	// bytebase-action flags
	cmd.PersistentFlags().StringVar(&w.Output, "output", "", "Output file location. The output file is a JSON file with the created resource names")
	cmd.PersistentFlags().StringVar(&w.URL, "url", "https://demo.bytebase.com", "Bytebase URL")
	cmd.PersistentFlags().StringVar(&w.ServiceAccount, "service-account", "api@service.bytebase.com", "Bytebase Service account")
	cmd.PersistentFlags().StringVar(&w.ServiceAccountSecret, "service-account-secret", os.Getenv("BYTEBASE_SERVICE_ACCOUNT_SECRET"), "Bytebase Service account secret")
	cmd.PersistentFlags().StringVar(&w.Project, "project", "projects/hr", "Bytebase project")
	cmd.PersistentFlags().StringSliceVar(&w.Targets, "targets", []string{"instances/test-sample-instance/databases/hr_test", "instances/prod-sample-instance/databases/hr_prod"}, "Bytebase targets")
	cmd.PersistentFlags().StringVar(&w.FilePattern, "file-pattern", "", "File pattern to glob migration files")

	cmd.AddCommand(NewCheckCommand(w))
	cmd.AddCommand(NewRolloutCommand(w))
	return cmd
}

func validateSharedFlagsWithWorld(w *world.World) func(cmd *cobra.Command, args []string) error {
	return func(*cobra.Command, []string) error {
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
		for _, target := range w.Targets {
			if !strings.HasPrefix(target, "instances/") || !strings.Contains(target, "/databases/") {
				return errors.Errorf("invalid target format, must be instances/{instance}/databases/{database}: %s", target)
			}
		}

		return nil
	}
}

func writeOutputJSON(w *world.World) func(cmd *cobra.Command, args []string) error {
	return func(*cobra.Command, []string) error {
		if w.Output == "" {
			return nil
		}

		slog.Info("writing output to file", "file", w.Output)

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
	}
}
