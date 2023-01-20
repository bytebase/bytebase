//go:build !embed_frontend
// +build !embed_frontend

package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/backend/common/log"
)

func embedFrontend(e *echo.Echo) {
	log.Info("Skip embedding frontend, build with 'embed_frontend' tag if you want embedded frontend.")

	e.GET("/*", func(c echo.Context) error {
		return c.HTML(http.StatusOK, "This Bytebase build does not bundle frontend and backend together.")
	})
}

// In non embedded mode, the redirect URL is the frontend URL which is different
// from the external URL. By default, this frontend URL is http://localhost:3000
func oauthRedirectURL(_ string) string {
	url := os.Getenv("BB_REDIRECT_URL")
	if url != "" {
		return url
	}
	return "http://localhost:3000"
}

func oauthErrorMessage(externalURL string) string {
	return fmt.Sprintf("Failed to exchange OAuth token. Make sure BB_REDIRECT_URL: %s matches your browser host. Note that if you are not using port 80 or 443, you should also specify the port such as --external-url=http://host:port", externalURL)
}
