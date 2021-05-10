package server

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	scas "github.com/qiangmzsx/string-adapter/v2"
)

type Server struct {
	l *bytebase.Logger

	PrincipalService   api.PrincipalService
	MemberService      api.MemberService
	EnvironmentService api.EnvironmentService

	e *echo.Echo
}

//go:embed dist
var embededFiles embed.FS

//go:embed dist/index.html
var indexContent string

//go:embed acl_casbin_model.conf
var casbinModel string

//go:embed acl_casbin_policy.csv
var casbinPolicy string

func getFileSystem() http.FileSystem {
	fsys, err := fs.Sub(embededFiles, "dist")
	if err != nil {
		panic(err)
	}

	return http.FS(fsys)
}

func NewServer(logger *bytebase.Logger) *Server {
	e := echo.New()

	// Middleware
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: func(c echo.Context) bool {
			return !strings.HasPrefix(c.Path(), "/api")
		},
		Format: `{"time":"${time_rfc3339}",` +
			`"method":"${method}","uri":"${uri}",` +
			`"status":${status},"error":"${error}"}` + "\n",
	}))
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return RecoverMiddleware(logger, next)
	})

	// Catch-all route to return index.html, this is to prevent 404 when accessing non-root url.
	// See https://stackoverflow.com/questions/27928372/react-router-urls-dont-work-when-refreshing-or-writing-manually
	e.GET("/*", func(c echo.Context) error {
		return c.HTML(http.StatusOK, indexContent)
	})

	assetHandler := http.FileServer(getFileSystem())
	e.GET("/assets/*", echo.WrapHandler(assetHandler))

	s := &Server{
		l: logger,
		e: e,
	}

	g := e.Group("/api")

	g.Use(middleware.JWTWithConfig(middleware.JWTConfig{
		Skipper: func(c echo.Context) bool {
			return strings.HasPrefix(c.Path(), "/api/auth")
		},
		Claims:        &Claims{},
		SigningMethod: middleware.AlgorithmHS256,
		SigningKey:    []byte(GetJWTSecret()),
		ContextKey:    GetTokenContextKey(),
		TokenLookup:   "cookie:access-token", // "<source>:<name>"
		ErrorHandlerWithContext: func(err error, c echo.Context) error {
			return JWTErrorChecker(logger, err, c)
		},
	}))

	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return TokenMiddleware(logger, next)
	})

	m, err := model.NewModelFromString(casbinModel)
	if err != nil {
		e.Logger.Fatal(err)
	}

	sa := scas.NewAdapter(casbinPolicy)
	ce, err := casbin.NewEnforcer(m, sa)
	if err != nil {
		e.Logger.Fatal(err)
	}
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return AclMiddleware(logger, s.MemberService, ce, next)
	})

	s.registerDebugRoutes(g)

	s.registerAuthRoutes(g)

	s.registerPrincipalRoutes(g)

	s.registerMemberRoutes(g)

	s.registerEnvironmentRoutes(g)

	return s
}

func (server *Server) Run() error {
	return server.e.Start(":8080")
}

func (server *Server) Shutdown(ctx context.Context) {
	if err := server.e.Shutdown(ctx); err != nil {
		server.e.Logger.Fatal(err)
	}
}
