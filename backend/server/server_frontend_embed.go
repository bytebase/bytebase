//go:build embed_frontend

package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/bytebase/bytebase/backend/common"
)

//go:embed dist/assets/*
//go:embed dist
var embeddedFiles embed.FS

// CSPHashes contains the Content Security Policy hashes exported by the frontend build.
type CSPHashes struct {
	ScriptHashes  []string `json:"scriptHashes"`
	GeneratedAt   string   `json:"generatedAt"`
	PluginVersion string   `json:"pluginVersion"`
}

var (
	cspHashesCache     *CSPHashes
	cspHashesCacheLock sync.RWMutex
)

// loadCSPHashes loads CSP hashes from the embedded frontend build output.
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

	// Try to load from embedded filesystem
	hashFilePath := "dist/csp-hashes.json"
	data, err := embeddedFiles.ReadFile(hashFilePath)

	if err == nil {
		var hashes CSPHashes
		if err := json.Unmarshal(data, &hashes); err == nil && len(hashes.ScriptHashes) > 0 {
			cspHashesCache = &hashes
			slog.Info("Loaded CSP hashes from embedded frontend build",
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

	slog.Warn("CSP hash file not found in embedded assets, using fallback hashes",
		"path", hashFilePath,
		"error", err)

	return fallbackHashes
}

func getFileSystem(path string) http.FileSystem {
	fs, err := fs.Sub(embeddedFiles, path)
	if err != nil {
		panic(err)
	}

	return http.FS(fs)
}

func embedFrontend(e *echo.Echo) {
	// Use echo static middleware to serve the built dist folder
	// refer: https://github.com/labstack/echo/blob/master/middleware/static.go
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Skipper:    defaultAPIRequestSkipper,
		HTML5:      true,
		Filesystem: getFileSystem("dist"),
	}))

	g := e.Group("assets")
	// Use echo gzip middleware to compress the response.
	// refer: https://echo.labstack.com/docs/middleware/gzip
	g.Use(middleware.GzipWithConfig(middleware.GzipConfig{
		Skipper: defaultAPIRequestSkipper,
		Level:   5,
	}))

	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set(echo.HeaderCacheControl, "max-age=31536000, immutable")
			return next(c)
		}
	})
	g.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Skipper:    defaultAPIRequestSkipper,
		HTML5:      true,
		Filesystem: getFileSystem("dist/assets"),
	}))
}

// defaultAPIRequestSkipper is echo skipper for api requests.
func defaultAPIRequestSkipper(c echo.Context) bool {
	path := c.Path()
	return common.HasPrefixes(path, "/api", "/v1", webhookAPIPrefix)
}
