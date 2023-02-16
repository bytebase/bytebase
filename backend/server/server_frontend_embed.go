//go:build embed_frontend
// +build embed_frontend

package server

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/bytebase/bytebase/backend/common"
)

//go:embed dist/assets/*
//go:embed dist
var embeddedFiles embed.FS

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
		Skipper:    DefaultAPIRequestSkipper,
		HTML5:      true,
		Filesystem: getFileSystem("dist"),
	}))

	g := e.Group("assets")
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set(echo.HeaderCacheControl, "max-age=31536000, immutable")
			return next(c)
		}
	})
	g.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Skipper:    DefaultAPIRequestSkipper,
		HTML5:      true,
		Filesystem: getFileSystem("dist/assets"),
	}))
}

func oauthRedirectURL(externalURL string) string {
	return externalURL
}

func oauthErrorMessage(externalURL string) string {
	if externalURL == common.ExternalURLDocsLink {
		return fmt.Sprintf("Failed to exchange OAuth token. You have not configured --external-url, please follow %s", common.ExternalURLDocsLink)
	}
	return fmt.Sprintf("Failed to exchange OAuth token. Make sure --external-url %s matches your browser host. Note that if you are not using port 80 or 443, you should also specify the port such as --external-url=http://host:port", externalURL)
}
