package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bytebase/bytebase/api"
	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	advisorDB "github.com/bytebase/bytebase/plugin/advisor/db"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/store"
	"github.com/labstack/echo/v4"
)

var (
	_ catalog.Catalog = (*catalogService)(nil)
)

// catalogService is the catalog service for sql check api.
type catalogService struct{}

// GetDatabase is the API message in catalog.
// We will not connect to the user's database in the early version of sql check api.
func (*catalogService) GetDatabase(_ context.Context) (*catalog.Database, error) {
	return nil, nil
}

func (s *Server) registerOpenAPIRoutes(g *echo.Group) {
	g.GET("/sql/advise", s.sqlCheckController)
}

// sqlCheckController godoc
// @Summary  Check the SQL statement.
// @Description  Parse and check the SQL statement according to the SQL review policy.
// @Accept  */*
// @Tags  SQL review
// @Produce  json
// @Param  environment   query  string  true   "The environment name. Case sensitive."
// @Param  statement     query  string  true   "The SQL statement."
// @Param  databaseType  query  string  false  "The database type. Required if the port, host and database name is not specified."  Enums(MYSQL, POSTGRES, TIDB)
// @Param  host          query  string  false  "The instance host."
// @Param  port          query  string  false  "The instance port."
// @Param  databaseName  query  string  false  "The database name in the instance."
// @Success  200  {array}   advisor.Advice
// @Failure  400  {object}  echo.HTTPError
// @Failure  500  {object}  echo.HTTPError
// @Router  /sql/advise  [get].
func (s *Server) sqlCheckController(c echo.Context) error {
	envName := c.QueryParams().Get("environment")
	if envName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required environment name")
	}

	statement := c.QueryParams().Get("statement")
	if statement == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required SQL statement")
	}

	ctx := c.Request().Context()
	var databaseType string
	var catalog catalog.Catalog = &catalogService{}

	databaseName := c.QueryParams().Get("databaseName")
	host := c.QueryParams().Get("host")
	port := c.QueryParams().Get("port")

	if databaseName != "" && host != "" && port != "" {
		database, err := s.findDatabase(ctx, host, port, databaseName)
		if err != nil {
			return err
		}
		dbType := database.Instance.Engine
		databaseType = string(dbType)
		catalog = store.NewCatalog(&database.ID, s.store, dbType)
	} else {
		databaseType = c.QueryParams().Get("databaseType")
		if databaseType == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Missing required database type")
		}
	}

	advisorDBType, err := advisorDB.ConvertToAdvisorDBType(databaseType)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database %s is not support", databaseType))
	}

	envList, err := s.store.FindEnvironment(ctx, &api.EnvironmentFind{
		Name: &envName,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to find environment %s", envName)).SetInternal(err)
	}
	if len(envList) != 1 {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid environment %s", envName))
	}

	_, adviceList, err := s.sqlCheck(
		ctx,
		advisorDBType,
		"utf8mb4",
		"utf8mb4_general_ci",
		envList[0].ID,
		statement,
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
				"environment":   envName,
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
