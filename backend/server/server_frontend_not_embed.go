//go:build !embed_frontend

package server

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

// getCSPHashesFromEmbedded returns an error in non-embed builds.
// The loadCSPHashes function will fall back to reading from filesystem.
func getCSPHashesFromEmbedded(_ string) ([]byte, error) {
	return nil, errors.New("frontend not embedded, use filesystem")
}

func embedFrontend(e *echo.Echo) {
	slog.Info("Skip embedding frontend, build with 'embed_frontend' tag if you want embedded frontend.")

	e.GET("/*", func(c echo.Context) error {
		return c.HTML(http.StatusOK, "This Bytebase build does not bundle frontend and backend together.")
	})
}
