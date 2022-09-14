package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	advisorDB "github.com/bytebase/bytebase/plugin/advisor/db"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/plugin/parser"
)

var (
	_ catalog.Catalog = (*catalogService)(nil)
)

// catalogService is the catalog service for sql check api.
type catalogService struct{}

// GetDatabase is the API message in catalog.
// We will not connect to the user's database in the early version of sql check api.
func (*catalogService) GetDatabase() *catalog.Database {
	return &catalog.Database{}
}

// GetFinder is the API message in catalog.
// We will not connect to the user's database in the early version of sql check api.
func (*catalogService) GetFinder() *catalog.Finder {
	return catalog.NewEmptyFinder(&catalog.FinderContext{CheckIntegrity: false})
}

func (s *Server) registerOpenAPIRoutes(g *echo.Group) {
	g.POST("/sql/advise", s.sqlCheckController)
	g.POST("/sql/schema/diff", schemaDiff)
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
	var catalog catalog.Catalog = &catalogService{}

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
// @Success  200  {array}   the target diff string of schemas
// @Failure  400  {object}  echo.HTTPError
// @Failure  500  {object}  echo.HTTPError
// @Router  /sql/schema/diff  [post].
func schemaDiff(c echo.Context) error {
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

	sourceSchemaNodes, err := parser.Parse(engine, parser.ParseContext{}, request.SourceSchema)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to parse source schema to AST nodes: %s", request.SourceSchema)).SetInternal(err)
	}

	targetSchemaNodes, err := parser.Parse(engine, parser.ParseContext{}, request.TargetSchema)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to parse target schema to AST nodes: %s", request.TargetSchema)).SetInternal(err)
	}

	diff, err := parser.SchemaDiff(sourceSchemaNodes, targetSchemaNodes)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to compute diff between source and target schemas").SetInternal(err)
	}

	return c.JSON(http.StatusOK, diff)
}
