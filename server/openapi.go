package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	advisorDB "github.com/bytebase/bytebase/plugin/advisor/db"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/differ"
)

var (
	_ catalog.Catalog = (*catalogService)(nil)
)

type projectSQLCheckRequestBody struct {
	Statement string `json:"statement"`
	FilePath  string `json:"filePath"`
}

// catalogService is the catalog service for sql check api.
type catalogService struct {
	finder *catalog.Finder
}

func newCatalogService(dbType advisorDB.Type) *catalogService {
	return &catalogService{
		finder: catalog.NewEmptyFinder(&catalog.FinderContext{CheckIntegrity: false}, dbType),
	}
}

// GetFinder is the API message in catalog.
func (c *catalogService) GetFinder() *catalog.Finder {
	return c.finder
}

func (s *Server) registerOpenAPIRoutes(g *echo.Group) {
	g.POST("/sql/advise", s.sqlCheckController)
	g.POST("/sql/schema/diff", s.schemaDiff)
	g.POST("/project/:projectID/sql-review", func(c echo.Context) error {
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}
		log.Debug("SQL review request received for VCS project", zap.Int("project", projectID))

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read the request").SetInternal(err)
		}

		request := &projectSQLCheckRequestBody{}
		if err := json.Unmarshal(body, request); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Cannot format request body").SetInternal(err)
		}

		fileEscaped := common.EscapeForLogging(request.FilePath)
		log.Debug("Processing file",
			zap.String("file", fileEscaped),
		)

		ctx := c.Request().Context()
		repos, err := s.store.FindRepository(ctx, &api.RepositoryFind{
			ProjectID: &projectID,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find repository").SetInternal(err)
		}
		if len(repos) == 0 {
			return echo.NewHTTPError(http.StatusNotFound, "Cannot found repository in project")
		}

		repo := repos[0]

		if !strings.HasPrefix(fileEscaped, repo.BaseDirectory) {
			log.Debug("Ignored file outside the base directory",
				zap.String("file", fileEscaped),
				zap.String("base_directory", repo.BaseDirectory),
			)
			return c.JSON(http.StatusOK, []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.Unsupported,
					Title:   "Ignored file outside the base directory",
					Line:    1,
					Content: fmt.Sprintf("This SQL file is outside the base directory \"%s\" configured in the VCS workflow. Skip the SQL check.", repo.BaseDirectory),
				},
			})
		}

		migrationInfo, err := db.ParseMigrationInfo(fileEscaped, path.Join(repo.BaseDirectory, repo.FilePathTemplate))
		if err != nil {
			log.Debug("Failed to parse migration info",
				zap.Int("project", projectID),
				zap.String("file", fileEscaped),
				zap.Error(err),
			)
			return echo.NewHTTPError(http.StatusOK, []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.Unsupported,
					Title:   "Failed to parse migration info",
					Line:    1,
					Content: fmt.Sprintf("Failed to parse migration info for this SQL file. Error: %v", err),
				},
			})
		}
		log.Debug(
			"Parse the migration info",
			zap.String("file", request.FilePath),
			zap.String("database", migrationInfo.Database),
			zap.String("environment", migrationInfo.Environment),
		)
		if migrationInfo.Database == "" {
			return c.JSON(http.StatusOK, []advisor.Advice{})
		}

		databases, err := s.findProjectDatabases(ctx, repo.ProjectID, repo.Project.TenantMode, migrationInfo.Database, migrationInfo.Environment)
		if err != nil {
			log.Debug(
				"Failed to list databse migration info",
				zap.Int("project", repo.ProjectID),
				zap.String("database", migrationInfo.Database),
				zap.String("environment", migrationInfo.Environment),
				zap.Error(err),
			)
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to list databse migration info").SetInternal(err)
		}

		sort.Slice(databases, func(i, j int) bool {
			return databases[i].Instance.Environment.Order < databases[j].Instance.Environment.Order
		})

		for _, database := range databases {
			policy, err := s.store.GetNormalSQLReviewPolicy(ctx, &api.PolicyFind{EnvironmentID: &database.Instance.EnvironmentID})
			if err != nil {
				if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
					log.Debug("Cannot found SQL review policy in environment", zap.Int("Environment", database.Instance.EnvironmentID), zap.Error(err))
					continue
				}

				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get SQL review policy in environment: %v", database.Instance.EnvironmentID)).SetInternal(err)
			}

			dbType, err := advisorDB.ConvertToAdvisorDBType(string(database.Instance.Engine))
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to convert database engine type %v to advisor db type", database.Instance.Engine)).SetInternal(err)
			}

			catalog, err := s.store.NewCatalog(ctx, database.ID, database.Instance.Engine)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get catalog for database %v", database.ID)).SetInternal(err)
			}

			adviceList, err := advisor.SQLReviewCheck(request.Statement, policy.RuleList, advisor.SQLReviewCheckContext{
				Charset:   database.CharacterSet,
				Collation: database.Collation,
				DbType:    dbType,
				Catalog:   catalog,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to exec the SQL check for database %v", database.ID)).SetInternal(err)
			}

			return c.JSON(http.StatusOK, adviceList)
		}

		return c.JSON(http.StatusOK, []advisor.Advice{})
	})
}

type sqlCheckRequestBody struct {
	Statement       string `json:"statement"`
	DatabaseType    string `json:"databaseType"`
	DatabaseName    string `json:"databaseName"`
	EnvironmentName string `json:"environmentName"`
	Host            string `json:"host"`
	Port            string `json:"port"`
}

// sqlCheckController godoc
// @Summary  Check the SQL statement.
// @Description  Parse and check the SQL statement according to the SQL review policy.
// @Accept  */*
// @Tags  SQL review
// @Produce  json
// @Param  environmentName  body  string  true   "The environment name. Case sensitive."
// @Param  statement        body  string  true   "The SQL statement."
// @Param  databaseType     body  string  false  "The database type. Required if the port, host and database name is not specified."  Enums(MYSQL, POSTGRES, TIDB)
// @Param  host             body  string  false  "The instance host."
// @Param  port             body  string  false  "The instance port."
// @Param  databaseName     body  string  false  "The database name in the instance."
// @Success  200  {array}   advisor.Advice
// @Failure  400  {object}  echo.HTTPError
// @Failure  500  {object}  echo.HTTPError
// @Router  /sql/advise  [post].
func (s *Server) sqlCheckController(c echo.Context) error {
	request := &sqlCheckRequestBody{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body").SetInternal(err)
	}
	if err := json.Unmarshal(body, request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Cannot format request body").SetInternal(err)
	}

	if request.EnvironmentName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required environment name")
	}

	if request.Statement == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required SQL statement")
	}

	ctx := c.Request().Context()
	var databaseType string
	var catalog catalog.Catalog

	if request.DatabaseName != "" && request.Host != "" && request.Port != "" {
		database, err := s.findDatabase(ctx, request.Host, request.Port, request.DatabaseName)
		if err != nil {
			return err
		}
		dbType := database.Instance.Engine
		databaseType = string(dbType)
		catalog, err = s.store.NewCatalog(ctx, database.ID, dbType)
		if err != nil {
			return err
		}
	} else {
		databaseType = request.DatabaseType
		if databaseType == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Missing required database type")
		}
	}

	advisorDBType, err := advisorDB.ConvertToAdvisorDBType(databaseType)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database %s is not support", databaseType))
	}
	if catalog == nil {
		catalog = newCatalogService(advisorDBType)
	}

	envList, err := s.store.FindEnvironment(ctx, &api.EnvironmentFind{
		Name: &request.EnvironmentName,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to find environment %s", request.EnvironmentName)).SetInternal(err)
	}
	if len(envList) != 1 {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid environment %s", request.EnvironmentName))
	}

	_, adviceList, err := s.sqlCheck(
		ctx,
		advisorDBType,
		"utf8mb4",
		"utf8mb4_general_ci",
		envList[0].ID,
		request.Statement,
		catalog,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to run sql check").SetInternal(err)
	}

	if s.MetricReporter != nil {
		s.MetricReporter.report(&metric.Metric{
			Name:  metricAPI.SQLAdviseAPIMetricName,
			Value: 1,
			Labels: map[string]string{
				"database_type": databaseType,
				"environment":   request.EnvironmentName,
			},
		})
	}

	return c.JSON(http.StatusOK, adviceList)
}

func (s *Server) findDatabase(ctx context.Context, host string, port string, databaseName string) (*api.Database, error) {
	instanceList, err := s.store.FindInstance(ctx, &api.InstanceFind{
		Host: &host,
		Port: &port,
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find instance by host: %s, port: %s", host, port)).SetInternal(err)
	}
	if len(instanceList) == 0 {
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Cannot find instance with host: %s, port: %s", host, port))
	}

	for _, instance := range instanceList {
		databaseList, err := s.store.FindDatabase(ctx, &api.DatabaseFind{
			InstanceID: &instance.ID,
			Name:       &databaseName,
		})
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find database by name %s in instance %d", databaseName, instance.ID)).SetInternal(err)
		}
		if len(databaseList) != 0 {
			return databaseList[0], nil
		}
	}

	return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Cannot find database %s in instance %s:%s", databaseName, host, port))
}

type schemaDiffRequestBody struct {
	EngineType   parser.EngineType `json:"engineType"`
	SourceSchema string            `json:"sourceSchema"`
	TargetSchema string            `json:"targetSchema"`
}

// schemaDiff godoc
// @Summary  Get the diff statement between source and target schema.
// @Description  Parse and diff the schema statements.
// @Accept  */*
// @Tags  SQL schema diff
// @Produce  json
// @Param  engineType       body  string  true   "The database engine type."
// @Param  sourceSchema     body  string  true   "The source schema statement."
// @Param  targetSchema     body  string  false  "The target schema statement."
// @Success  200  {string}  the target diff string of schemas
// @Failure  400  {object}  echo.HTTPError
// @Failure  500  {object}  echo.HTTPError
// @Router  /sql/schema/diff  [post].
func (s *Server) schemaDiff(c echo.Context) error {
	if !s.feature(api.FeatureSyncSchema) {
		return echo.NewHTTPError(http.StatusForbidden, api.FeatureSyncSchema.AccessErrorMessage())
	}

	request := &schemaDiffRequestBody{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body").SetInternal(err)
	}
	if err := json.Unmarshal(body, request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Cannot format request body").SetInternal(err)
	}

	var engine parser.EngineType
	switch request.EngineType {
	case parser.EngineType(db.Postgres):
		engine = parser.Postgres
	case parser.EngineType(db.MySQL):
		engine = parser.MySQL
	default:
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid database engine %s", request.EngineType))
	}

	diff, err := differ.SchemaDiff(engine, request.SourceSchema, request.TargetSchema)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to compute diff between source and target schemas").SetInternal(err)
	}

	return c.JSON(http.StatusOK, diff)
}
