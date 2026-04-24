package server

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"

	"github.com/bytebase/bytebase/backend/common"
)

// registerFrontendRoutes wires up static file serving for the embedded frontend
// against the given filesystem (typically the embedded dist tree). It is extracted
// from embedFrontend so the behavior can be exercised in tests with an in-memory fs.FS.
//
// Layout:
//   - Global static middleware serves the SPA shell (index.html) with HTML5 fallback
//     for client-side routes, but skips /assets/* so the dedicated route below wins.
//   - /assets/* is served by a real route that returns a true 404 for missing files
//     (no HTML5 fallback) and attaches long-cache + gzip middleware.
func registerFrontendRoutes(e *echo.Echo, distFS fs.FS) {
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Skipper: func(c *echo.Context) bool {
			return defaultAPIRequestSkipper(c) || strings.HasPrefix(c.Request().URL.Path, "/assets/")
		},
		HTML5:      true,
		Filesystem: distFS,
	}))

	assetsFS, err := fs.Sub(distFS, "assets")
	if err != nil {
		panic(err)
	}
	cacheImmutable := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Response().Header().Set(echo.HeaderCacheControl, "public, max-age=31536000, immutable")
			return next(c)
		}
	}
	// Register GET + HEAD so the dedicated asset route matches HEAD requests too
	// (curl -I, CDN health checks, cache revalidation). Before this fix, the global
	// StaticWithConfig middleware served any method; a GET-only route would 405 on HEAD.
	e.Match([]string{http.MethodGet, http.MethodHead}, "/assets/*",
		echo.StaticDirectoryHandler(assetsFS, false),
		middleware.GzipWithConfig(middleware.GzipConfig{Level: 5}),
		cacheImmutable,
	)
}

// defaultAPIRequestSkipper is echo skipper for api requests.
func defaultAPIRequestSkipper(c *echo.Context) bool {
	path := c.Request().URL.Path
	return common.HasPrefixes(path, "/api", "/v1", "/.well-known", webhookAPIPrefix)
}
