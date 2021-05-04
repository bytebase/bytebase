package server

import (
	"context"
	"embed"
	"io/fs"
	"net/http"

	"github.com/labstack/echo/v4"
)

type Environment struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Order uint   `json:"order"`
}

type EnvironmentService interface {
}

type Server struct {
	EnvironmentService EnvironmentService

	e *echo.Echo
}

//go:embed dist
var embededFiles embed.FS

func getFileSystem() http.FileSystem {
	fsys, err := fs.Sub(embededFiles, "dist")
	if err != nil {
		panic(err)
	}

	return http.FS(fsys)
}

func NewServer() *Server {
	e := echo.New()

	assetHandler := http.FileServer(getFileSystem())
	e.GET("/", echo.WrapHandler(assetHandler))
	e.GET("/assets/*", echo.WrapHandler(assetHandler))

	s := &Server{
		e: e,
	}

	s.registerDebugRoutes(e)

	s.registerEnvironmentRoutes(e)

	return s
}

func (s *Server) Run() error {
	return s.e.Start(":8080")
}

func (s *Server) Close(ctx context.Context) {
	if err := s.e.Shutdown(ctx); err != nil {
		s.e.Logger.Fatal(err)
	}
}
