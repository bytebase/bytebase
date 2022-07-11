package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bytebase/bytebase/api"
	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/bytebase/bytebase/store"
	"github.com/labstack/echo/v4"
)

var (
	_ catalog.Catalog = (*catalogService)(nil)
)

// catalogService is the catalog service for sql check api.
type catalogService struct{}

// FindIndex is the API message for find index in catalog.
// We will not connect to the user's database in the early version of sql check api
func (c *catalogService) FindIndex(ctx context.Context, find *catalog.IndexFind) (*catalog.Index, error) {
	return nil, nil
}

func (s *Server) registerOpenAPIRoutes(g *echo.Group) {
	g.GET("/sql/advise", s.sqlCheckController)
}

// sqlCheckController godoc
// @Summary  Check the SQL statement.
// @Description  Parse and check the SQL statement according to the schema review policy.
// @Accept  */*
// @Tags  Schema Review
// @Produce  json
// @Param  environment   query  string  true   "The environment name. Case sensitive."
// @Param  statement     query  string  true   "The SQL statement."
// @Param  databaseType  query  string  false  "The database type. Required if not provide the database id."  Enums(MySQL, PostgreSQL, TiDB)
// @Param  databaseID    query  number  false  "The database id in your instance."
// @Success  200  {array}   advisor.Advice
// @Failure  400  {object}  echo.HTTPError
// @Failure  500  {object}  echo.HTTPError
// @Router  /sql/advise  [get]
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
	databaseID := c.QueryParams().Get("databaseID")

	var dbType db.Type
	var catalog catalog.Catalog = &catalogService{}

	if databaseID != "" {
		id, err := strconv.Atoi(databaseID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid database ID: %v", databaseID)).SetInternal(err)
		}

		database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{
			ID: &id,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find database by ID: %v", databaseID)).SetInternal(err)
		}

		dbType = database.Instance.Engine
		catalog = store.NewCatalog(&id, s.store)
	} else {
		databaseType := c.QueryParams().Get("databaseType")
		if databaseType == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Missing required database type")
		}
		dbType = db.Type(strings.ToUpper(databaseType))
	}

	advisorDBType, err := api.ConvertToAdvisorDBType(dbType)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database %s is not support", dbType))
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
				"database_type": string(dbType),
				"environment":   envName,
			},
		})
	}

	return c.JSON(http.StatusOK, adviceList)
}
