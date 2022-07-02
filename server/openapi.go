package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bytebase/bytebase/api"
	metricAPI "github.com/bytebase/bytebase/metric"
	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/metric"
	"github.com/labstack/echo/v4"
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
// @Param  environment   query  string  true   "The environment name. Case sensitive"
// @Param  statement     query  string  true   "The SQL statement"
// @Param  databaseType  query  string  true   "The database type"  Enums(MySQL, PostgreSQL, TiDB)
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

	dbType := c.QueryParams().Get("databaseType")
	if dbType == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required database type")
	}
	advisorDBType, err := api.ConvertToAdvisorDBType(db.Type(strings.ToUpper(dbType)))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database %s is not support", dbType))
	}

	ctx := c.Request().Context()
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
		&catalogService{},
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to run sql check").SetInternal(err)
	}

	if s.MetricReporter != nil {
		s.MetricReporter.report(&metric.Metric{
			Name:  metricAPI.SQLAdviseAPIMetricName,
			Value: 1,
			Labels: map[string]string{
				"database_type": dbType,
				"environment":   envName,
			},
		})
	}

	return c.JSON(http.StatusOK, adviceList)
}
