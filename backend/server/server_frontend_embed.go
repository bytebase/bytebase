//go:build embed_frontend

package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log/slog"
	"sync"

	"github.com/labstack/echo/v5"
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

func getFileSystem(path string) fs.FS {
	subFS, err := fs.Sub(embeddedFiles, path)
	if err != nil {
		panic(err)
	}

	return subFS
}

func embedFrontend(e *echo.Echo) {
	registerFrontendRoutes(e, getFileSystem("dist"))
}
