package cloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/action/command/validation"
	"github.com/bytebase/bytebase/action/world"
)

// EnsureWorkspaceAwake checks if a Bytebase cloud workspace is healthy and wakes it up if needed.
// Special handling for Bytebase cloud URLs (*.us-central1.bytebase.com)
func EnsureWorkspaceAwake(w *world.World) error {
	u, _ := url.Parse(w.URL) // Already validated in ValidateFlags
	if !validation.IsCloudURL(u) {
		return nil
	}
	host := u.Host
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
