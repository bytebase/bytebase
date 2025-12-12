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

var (
	cspHashesCache []string
	cspHashesOnce  sync.Once
)

// loadCSPHashes loads CSP hashes from the embedded frontend build output.
// Uses sync.Once to ensure hashes are loaded only once.
func loadCSPHashes() []string {
	cspHashesOnce.Do(func() {
		hashFilePath := "dist/csp-hashes.json"
		data, err := embeddedFiles.ReadFile(hashFilePath)
		if err != nil {
			slog.Error("Failed to read CSP hashes from embedded assets",
				"path", hashFilePath,
				"error", err)
			panic("CSP hashes file not found in embedded build - this is a build error")
		}

		var hashes struct {
			ScriptHashes  []string `json:"scriptHashes"`
			GeneratedAt   string   `json:"generatedAt"`
			PluginVersion string   `json:"pluginVersion"`
		}
		if err := json.Unmarshal(data, &hashes); err != nil {
			slog.Error("Failed to unmarshal CSP hashes",
				"path", hashFilePath,
				"error", err)
			panic("Invalid CSP hashes file format - this is a build error")
		}

		if len(hashes.ScriptHashes) == 0 {
			slog.Error("CSP hashes file contains no hashes", "path", hashFilePath)
			panic("Empty CSP hashes - this is a build error")
		}

		cspHashesCache = hashes.ScriptHashes
		slog.Info("Loaded CSP hashes from embedded frontend build",
			"path", hashFilePath,
			"plugin_version", hashes.PluginVersion,
			"hash_count", len(hashes.ScriptHashes))
	})

	return cspHashesCache
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
	path := c.Request().URL.Path
	return common.HasPrefixes(path, "/api", "/v1", "/.well-known", webhookAPIPrefix)
}
