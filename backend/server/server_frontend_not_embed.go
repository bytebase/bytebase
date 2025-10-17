//go:build !embed_frontend

package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

// loadCSPHashes loads CSP hashes from the filesystem.
// Falls back to hardcoded hashes if the file doesn't exist.
func loadCSPHashes() []string {
	cspHashesCacheLock.RLock()
	if cspHashesCache != nil {
		defer cspHashesCacheLock.RUnlock()
		return cspHashesCache.ScriptHashes
	}
	cspHashesCacheLock.RUnlock()

	cspHashesCacheLock.Lock()
	defer cspHashesCacheLock.Unlock()

	// Double-check after acquiring write lock
	if cspHashesCache != nil {
		return cspHashesCache.ScriptHashes
	}

	// Try to read from filesystem (development mode)
	hashFilePath := "dist/csp-hashes.json"
	data, err := os.ReadFile(hashFilePath)

	if err == nil {
		var hashes CSPHashes
		if err := json.Unmarshal(data, &hashes); err == nil && len(hashes.ScriptHashes) > 0 {
			cspHashesCache = &hashes
			slog.Info("Loaded CSP hashes from filesystem",
				"path", hashFilePath,
				"plugin_version", hashes.PluginVersion,
				"hash_count", len(hashes.ScriptHashes))
			return hashes.ScriptHashes
		}
	}

	// Fallback to hardcoded hashes for @vitejs/plugin-legacy@7.2.1
	// These are stable as long as the plugin version is pinned with ~ in package.json
	fallbackHashes := []string{
		"'sha256-MS6/3FCg4WjP9gwgaBGwLpRCY6fZBgwmhVCdrPrNf3E='", // Safari 10 nomodule fix
		"'sha256-tQjf8gvb2ROOMapIxFvFAYBeUJ0v1HCbOcSmDNXGtDo='", // SystemJS inline code
		"'sha256-ZxAi3a7m9Mzbc+Z1LGuCCK5Xee6reDkEPRas66H9KSo='", // Modern browser detection
		"'sha256-+5XkZFazzJo8n0iOP4ti/cLCMUudTf//Mzkb7xNPXIc='", // Dynamic fallback
	}

	cspHashesCache = &CSPHashes{
		ScriptHashes:  fallbackHashes,
		PluginVersion: "7.2.1",
		GeneratedAt:   "fallback",
	}

	slog.Warn("CSP hash file not found on filesystem, using fallback hashes",
		"path", hashFilePath,
		"error", err)

	return fallbackHashes
}

func embedFrontend(e *echo.Echo) {
	slog.Info("Skip embedding frontend, build with 'embed_frontend' tag if you want embedded frontend.")

	e.GET("/*", func(c echo.Context) error {
		return c.HTML(http.StatusOK, "This Bytebase build does not bundle frontend and backend together.")
	})
}
