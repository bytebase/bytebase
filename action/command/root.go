package command

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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

		// Special handling for Bytebase cloud URLs (*.us-central1.bytebase.com)
		cloudURLPattern := regexp.MustCompile(`^[a-z0-9]+\.us-central1\.bytebase\.com$`)
		if cloudURLPattern.MatchString(u.Host) {
			if err := ensureWorkspaceAwake(w, u.Host); err != nil {
				return errors.Wrapf(err, "failed to wake up workspace")
			}
		}

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

// ensureWorkspaceAwake checks if a Bytebase cloud workspace is healthy and wakes it up if needed.
func ensureWorkspaceAwake(w *world.World, host string) error {
	healthzURL := fmt.Sprintf("https://%s/healthz", host)

	// Check if the workspace is already healthy
	if isHealthy(healthzURL) {
		w.Logger.Info("Workspace is already healthy", "host", host)
		return nil
	}

	// Wake up the workspace
	w.Logger.Info("Workspace needs to be awakened", "host", host)
	if err := wakeUpWorkspace(w.Logger, host); err != nil {
		return errors.Wrapf(err, "failed to wake up workspace")
	}

	// Wait 15 seconds before checking healthz.
	time.Sleep(15 * time.Second)

	// Wait for the workspace to become healthy (3 consecutive successful health checks)
	w.Logger.Info("Waiting for workspace to become healthy...")
	consecutiveSuccess := 0
	maxAttempts := 60 // Maximum 5 minutes (60 * 5 seconds)

	for attempt := range maxAttempts {
		if isHealthy(healthzURL) {
			consecutiveSuccess++
			w.Logger.Info("Health check succeeded", "consecutive", consecutiveSuccess)
			if consecutiveSuccess >= 3 {
				w.Logger.Info("Workspace is now healthy", "host", host)
				return nil
			}
		} else {
			consecutiveSuccess = 0
			w.Logger.Info("Health check failed, retrying...", "attempt", attempt+1)
			time.Sleep(5 * time.Second)
		}
	}

	return errors.Errorf("workspace did not become healthy after %d attempts", maxAttempts)
}

// isHealthy checks if the workspace health endpoint returns OK.
func isHealthy(healthzURL string) bool {
	// Add cache-busting parameter to ensure fresh response
	urlWithCacheBust := fmt.Sprintf("%s?_=%d", healthzURL, time.Now().UnixNano())

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", urlWithCacheBust, nil)
	if err != nil {
		return false
	}

	// Add no-cache headers
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// wakeUpWorkspace calls the API to wake up a Bytebase cloud workspace.
func wakeUpWorkspace(logger *slog.Logger, domain string) error {
	wakeUpURL := "https://hub.bytebase.com/v1/workspaces:wakeUp"

	payload := map[string]string{
		"domain": domain,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal wake up request")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", wakeUpURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return errors.Wrapf(err, "failed to create wake up request")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to send wake up request")
	}
	defer resp.Body.Close()

	// Read response body for logging
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyString := string(bodyBytes)

	// Log response status and body
	logger.Info("Wake up workspace response", "status", resp.StatusCode, "body", bodyString)

	return nil
}
