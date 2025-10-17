package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"

	connectcors "connectrpc.com/cors"

	directorysync "github.com/bytebase/bytebase/backend/api/directory-sync"
	"github.com/bytebase/bytebase/backend/api/lsp"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
)

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

func configureEchoRouters(
	e *echo.Echo,
	lspServer *lsp.Server,
	directorySyncServer *directorysync.Service,
	profile *config.Profile,
) {
	e.Use(recoverMiddleware)
	e.Use(securityHeadersMiddleware)

	if profile.Mode == common.ReleaseModeDev {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOriginFunc: func(string) (bool, error) {
				return true, nil
			},
			AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodOptions},
			AllowHeaders:     connectcors.AllowedHeaders(),
			ExposeHeaders:    connectcors.ExposedHeaders(),
			AllowCredentials: true,
		}))
	}

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogMethod: true,
		LogStatus: true,
		LogError:  true,
		LogValuesFunc: func(_ echo.Context, values middleware.RequestLoggerValues) error {
			if values.Error != nil {
				slog.Error("echo request logger", "method", values.Method, "uri", values.URI, "status", values.Status, log.BBError(values.Error))
			}
			return nil
		},
	}))

	// Embed frontend.
	embedFrontend(e)

	e.HideBanner = true
	e.HidePort = true

	registerPprof(e, &profile.RuntimeDebug)

	p := prometheus.NewPrometheus("api", nil)
	p.RequestCounterURLLabelMappingFunc = func(c echo.Context) string {
		return c.Request().URL.Path
	}
	p.Use(e)

	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	// LSP server.
	e.GET(lspAPI, lspServer.Router)

	hookGroup := e.Group(webhookAPIPrefix)
	scimGroup := hookGroup.Group(scimAPIPrefix)
	directorySyncServer.RegisterDirectorySyncRoutes(scimGroup)
}

func recoverMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					err = errors.Errorf("%v", r)
				}
				slog.Error("Middleware PANIC RECOVER", log.BBError(err), log.BBStack("panic-stack"))

				c.Error(err)
			}
		}()
		return next(c)
	}
}

// loadCSPHashes loads CSP hashes from the embedded frontend build output or filesystem.
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

	// Try to load from frontend build output
	hashFilePath := filepath.Join("dist", "csp-hashes.json")
	data, err := os.ReadFile(hashFilePath)
	if err == nil {
		var hashes CSPHashes
		if err := json.Unmarshal(data, &hashes); err == nil && len(hashes.ScriptHashes) > 0 {
			cspHashesCache = &hashes
			slog.Info("Loaded CSP hashes from frontend build",
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

	slog.Warn("CSP hash file not found, using fallback hashes",
		"path", hashFilePath,
		"error", err)

	return fallbackHashes
}

func securityHeadersMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Allow popups to maintain window.opener for OAuth flows
		c.Response().Header().Set("Cross-Origin-Opener-Policy", "same-origin-allow-popups")
		// Prevent being embedded in iframes from different origins
		c.Response().Header().Set("X-Frame-Options", "SAMEORIGIN")
		// Prevent MIME-type sniffing
		c.Response().Header().Set("X-Content-Type-Options", "nosniff")
		// Force HTTPS in production (only if request is already HTTPS)
		if c.Request().TLS != nil || c.Request().Header.Get("X-Forwarded-Proto") == "https" {
			// max-age=31536000 (1 year), includeSubDomains for all subdomains
			c.Response().Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Content Security Policy
		// Note: style-src allows 'unsafe-inline' temporarily due to inline styles in Vue components
		// TODO: Migrate inline styles to CSS classes and remove 'unsafe-inline'
		// Note: script-src uses SHA-256 hashes for @vitejs/plugin-legacy inline scripts
		//       Hashes are loaded from dist/csp-hashes.json (generated by frontend build)
		//       or fall back to hardcoded hashes for @vitejs/plugin-legacy@7.2.1
		// Note: script-src allows 'wasm-unsafe-eval' for Monaco Editor WebAssembly modules
		// Note: connect-src allows 'data:' for Monaco Editor language definitions
		scriptHashes := loadCSPHashes()
		csp := "default-src 'self'; " +
			"script-src 'self' " + strings.Join(scriptHashes, " ") + " 'wasm-unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: blob:; " +
			"connect-src 'self' data: ws: wss:; " +
			"font-src 'self'; " +
			"object-src 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'; " +
			"frame-ancestors 'self'"
		c.Response().Header().Set("Content-Security-Policy", csp)
		return next(c)
	}
}
