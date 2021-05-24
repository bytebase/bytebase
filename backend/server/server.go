package server

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	scas "github.com/qiangmzsx/string-adapter/v2"
)

type Server struct {
	l             *bytebase.Logger
	TaskScheduler *TaskScheduler

	PrincipalService     api.PrincipalService
	MemberService        api.MemberService
	ProjectService       api.ProjectService
	ProjectMemberService api.ProjectMemberService
	EnvironmentService   api.EnvironmentService
	InstanceService      api.InstanceService
	DatabaseService      api.DatabaseService
	DataSourceService    api.DataSourceService
	IssueService         api.IssueService
	PipelineService      api.PipelineService
	StageService         api.StageService
	TaskService          api.TaskService
	TaskRunService       api.TaskRunService
	ActivityService      api.ActivityService
	BookmarkService      api.BookmarkService

	e *echo.Echo
}

//go:embed dist
var embededFiles embed.FS

//go:embed dist/index.html
var indexContent string

//go:embed acl_casbin_model.conf
var casbinModel string

//go:embed acl_casbin_policy_owner.csv
var casbinOwnerPolicy string

//go:embed acl_casbin_policy_dba.csv
var casbinDBAPolicy string

//go:embed acl_casbin_policy_developer.csv
var casbinDeveloperPolicy string

func getFileSystem() http.FileSystem {
	fsys, err := fs.Sub(embededFiles, "dist")
	if err != nil {
		panic(err)
	}

	return http.FS(fsys)
}

func NewServer(logger *bytebase.Logger) *Server {
	e := echo.New()

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

	scheduler := NewTaskScheduler(log.Default(), s)
	approveExecutor := NewApproveTaskExecutor(log.Default())
	sqlExecutor := NewSqlTaskExecutor(log.Default())
	scheduler.Register(string(api.TaskApprove), approveExecutor)
	scheduler.Register(string(api.TaskDatabaseSchemaUpdate), sqlExecutor)
	s.TaskScheduler = scheduler

	g := e.Group("/api")

	// Middleware
	g.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: func(c echo.Context) bool {
			return !strings.HasPrefix(c.Path(), "/api")
		},
		Format: `{"time":"${time_rfc3339}",` +
			`"method":"${method}","uri":"${uri}",` +
			`"status":${status},"error":"${error}"}` + "\n",
	}))
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return RecoverMiddleware(logger, next)
	})

	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return JWTMiddleware(logger, s.PrincipalService, next)
	})

	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return RequestMiddleware(logger, next)
	})

	m, err := model.NewModelFromString(casbinModel)
	if err != nil {
		e.Logger.Fatal(err)
	}
	sa := scas.NewAdapter(strings.Join([]string{casbinOwnerPolicy, casbinDBAPolicy, casbinDeveloperPolicy}, "\n"))
	ce, err := casbin.NewEnforcer(m, sa)
	if err != nil {
		e.Logger.Fatal(err)
	}
	g.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return ACLMiddleware(logger, s, ce, next)
	})

	s.registerDebugRoutes(g)
	s.registerAuthRoutes(g)
	s.registerPrincipalRoutes(g)
	s.registerMemberRoutes(g)
	s.registerProjectRoutes(g)
	s.registerProjectMemberRoutes(g)
	s.registerEnvironmentRoutes(g)
	s.registerInstanceRoutes(g)
	s.registerDatabaseRoutes(g)
	s.registerIssueRoutes(g)
	s.registerTaskRoutes(g)
	s.registerActivityRoutes(g)
	s.registerBookmarkRoutes(g)
	s.registerSqlRoutes(g)

	return s
}

func (server *Server) Run() error {
	if err := server.TaskScheduler.Run(); err != nil {
		return err
	}

	const port int = 8080
	// Sleep for 1 sec to make sure port is released between runs.
	time.Sleep(time.Duration(1) * time.Second)
	return server.e.Start(fmt.Sprintf(":%d", port))
}

func (server *Server) Shutdown(ctx context.Context) {
	if err := server.e.Shutdown(ctx); err != nil {
		server.e.Logger.Fatal(err)
	}
}
