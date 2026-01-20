//go:build !embed_frontend

package server

import (
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"
)

// loadCSPHashes returns empty hashes in dev mode since frontend is served separately.
func loadCSPHashes() []string {
	return []string{}
}

func embedFrontend(e *echo.Echo) {
	slog.Info("Skip embedding frontend, build with 'embed_frontend' tag if you want embedded frontend.")

	e.GET("/*", func(c *echo.Context) error {
		return c.HTML(http.StatusOK, "This Bytebase build does not bundle frontend and backend together.")
	})
}
