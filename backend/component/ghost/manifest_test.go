package ghost

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// frontendFlagManifestPath is the gh-ost flag manifest the "Configure" UI builds
// its parameter list from (frontend/src/react/components/ghost/constants.ts
// imports it). The path is relative to this package directory, where `go test`
// runs.
const frontendFlagManifestPath = "../../../frontend/src/react/components/ghost/flags.json"

// TestFrontendFlagManifestInSync fails if the frontend gh-ost flag manifest
// drifts from the backend allowlist/defaults, so a flag added, removed, or
// re-defaulted on one side cannot silently diverge from the other.
func TestFrontendFlagManifestInSync(t *testing.T) {
	data, err := os.ReadFile(frontendFlagManifestPath)
	require.NoErrorf(t, err, "read frontend flag manifest %s", frontendFlagManifestPath)

	var manifest []struct {
		Key     string `json:"key"`
		Type    string `json:"type"`
		Default string `json:"default"`
	}
	require.NoErrorf(t, json.Unmarshal(data, &manifest), "parse %s", frontendFlagManifestPath)

	// 1. The manifest's key set must match the backend allowlist exactly.
	manifestKeys := make(map[string]bool, len(manifest))
	for _, f := range manifest {
		require.Falsef(t, manifestKeys[f.Key], "duplicate flag %q in manifest", f.Key)
		manifestKeys[f.Key] = true
	}
	for key := range knownKeys {
		require.Truef(t, manifestKeys[key], "backend flag %q is missing from the frontend manifest — add it so the Configure UI exposes it", key)
	}
	for key := range manifestKeys {
		require.Truef(t, knownKeys[key], "frontend manifest flag %q is not in the backend allowlist (knownKeys) — the backend would reject it", key)
	}

	// 2. Defaults the backend actually applies must match what the UI shows.
	// Derived from defaultConfig so a backend default change is caught here.
	// Flags whose default is gh-ost's own zero value (e.g. max-load, the *-rbr
	// toggles) are not held in defaultConfig and are left unchecked.
	backendDefaults := map[string]string{
		"attempt-instant-ddl":              strconv.FormatBool(defaultConfig.attemptInstantDDL),
		"allow-on-master":                  strconv.FormatBool(defaultConfig.allowedRunningOnMaster),
		"heartbeat-interval-millis":        strconv.FormatInt(defaultConfig.heartbeatIntervalMilliseconds, 10),
		"nice-ratio":                       strconv.FormatFloat(defaultConfig.niceRatio, 'g', -1, 64),
		"chunk-size":                       strconv.FormatInt(defaultConfig.chunkSize, 10),
		"dml-batch-size":                   strconv.FormatInt(defaultConfig.dmlBatchSize, 10),
		"max-lag-millis":                   strconv.FormatInt(defaultConfig.maxLagMillisecondsThrottleThreshold, 10),
		"default-retries":                  strconv.FormatInt(defaultConfig.defaultNumRetries, 10),
		"cut-over-lock-timeout-seconds":    strconv.FormatInt(defaultConfig.cutoverLockTimeoutSeconds, 10),
		"exponential-backoff-max-interval": strconv.FormatInt(defaultConfig.exponentialBackoffMaxInterval, 10),
	}
	for _, f := range manifest {
		want, ok := backendDefaults[f.Key]
		if !ok {
			continue
		}
		require.Equalf(t, want, f.Default, "default for flag %q drifted from backend defaultConfig", f.Key)
	}
}
