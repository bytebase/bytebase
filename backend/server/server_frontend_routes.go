package server

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
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
			// Mark an asset immutable only once the response is known to be a
			// success. In an HA rolling deploy an older replica can 404 a new
			// hashed asset briefly, and the static handler rejects a path with a
			// dot or empty segment outright, so a year of cached "does not exist"
			// has to be impossible. Reading the status rather than re-deriving the
			// file from the path also leaves echo's path rules to echo: deciding
			// from a cleaned path is what put the header on those 404s.
			//
			// 304 keeps the directive so a revalidating CDN does not shorten its
			// TTL; WriteHeader sets Status before it runs these hooks.
			res, err := echo.UnwrapResponse(c.Response())
			if err != nil {
				// Without echo's response the status is not observable, and guessing
				// from the path is exactly what put immutable on those 404s. Serve
				// the asset uncached instead.
				return next(c)
			}
			res.Before(func() {
				if res.Status == http.StatusOK || res.Status == http.StatusNotModified {
					res.Header().Set(echo.HeaderCacheControl, "public, max-age=31536000, immutable")
				}
			})
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

// defaultAPIRequestSkipper is echo skipper for api requests. /bytebase.v1 covers
// Connect-go gRPC handlers so unknown gRPC paths return a structured 404 instead
// of falling through to the SPA's HTML5 fallback (which would return index.html).
func defaultAPIRequestSkipper(c *echo.Context) bool {
	path := c.Request().URL.Path
	return hasPrefixes(path, "/api", "/v1", "/bytebase.v1", "/.well-known", webhookAPIPrefix)
}

func hasPrefixes(src string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(src, prefix) {
			return true
		}
	}
	return false
}
