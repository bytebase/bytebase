//go:build embed_frontend
// +build embed_frontend

package server

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

//go:embed dist
var embeddedFiles embed.FS

func getFileSystem() http.FileSystem {
	fs, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		panic(err)
	}

	return http.FS(fs)
}

func embedFrontend(e *echo.Echo) {
	// Use echo static middleware to serve the built dist folder
	// refer: https://github.com/labstack/echo/blob/master/middleware/static.go
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		HTML5:      true,
		Filesystem: getFileSystem(),
	}))
}
