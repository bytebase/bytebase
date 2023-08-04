package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/backend/common"
	metricAPI "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	advisorDB "github.com/bytebase/bytebase/backend/plugin/advisor/db"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/differ"
	"github.com/bytebase/bytebase/backend/store"
)

var (
	_ catalog.Catalog = (*catalogService)(nil)
)

// catalogService is the catalog service for sql check api.
type catalogService struct {
	finder *catalog.Finder
}

func newCatalogService(dbType advisorDB.Type) *catalogService {
	return &catalogService{
		finder: catalog.NewEmptyFinder(&catalog.FinderContext{CheckIntegrity: false, EngineType: dbType}),
	}
}

// GetFinder is the API message in catalog.
func (c *catalogService) GetFinder() *catalog.Finder {
	return c.finder
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
// @Param  databaseType     body  string  false  "The database type. Required if the port, host and database name is not specified."  Enums(MYSQL, POSTGRES, TIDB, OCEANBASE)
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

	// EnvironmentName is the environment resource ID.
	if request.EnvironmentName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required environment name")
	}

	if request.Statement == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required SQL statement")
	}

	ctx := c.Request().Context()
	var databaseType string
	var catalog catalog.Catalog
	var driver db.Driver
	var connection *sql.DB

	if request.DatabaseName != "" && request.Host != "" && request.Port != "" {
		instances, err := s.store.ListInstancesV2(ctx, &store.FindInstanceMessage{})
		if err != nil {
			return err
		}
		var instance *store.InstanceMessage
		for _, v := range instances {
			for _, d := range v.DataSources {
				if d.Host == request.Host && d.Port == request.Port {
					instance = v
					break
				}
			}
			if instance != nil {
				break
			}
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, "instance not found with host and port")
		}
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
			InstanceID:          &instance.ResourceID,
			DatabaseName:        &request.DatabaseName,
			IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		})
		if err != nil {
			return err
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, "database not found")
		}

		dbType := instance.Engine
		databaseType = string(dbType)
		// TODO(rebelice): support SDL mode for open api.
		catalog, err = s.store.NewCatalog(ctx, database.UID, dbType, advisor.SyntaxModeNormal)
		if err != nil {
			return err
		}
		driver, err = s.dbFactory.GetReadOnlyDatabaseDriver(ctx, instance, database)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get database driver").SetInternal(err)
		}
		defer driver.Close(ctx)
		connection = driver.GetDB()
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

	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &request.EnvironmentName})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to find environment %s", request.EnvironmentName)).SetInternal(err)
	}
	if environment == nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid environment %s", request.EnvironmentName))
	}

	_, adviceList, err := s.sqlCheck(
		ctx,
		advisorDBType,
		"utf8mb4",
		"utf8mb4_general_ci",
		environment.UID,
		request.Statement,
		catalog,
		connection,
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to run sql check").SetInternal(err)
	}

	s.MetricReporter.Report(ctx, &metric.Metric{
		Name:  metricAPI.SQLAdviseAPIMetricName,
		Value: 1,
		Labels: map[string]any{
			"database_type": databaseType,
			"environment":   request.EnvironmentName,
		},
	})

	return c.JSON(http.StatusOK, adviceList)
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
	case parser.EngineType(db.MySQL), parser.EngineType(db.MariaDB), parser.EngineType(db.OceanBase):
		engine = parser.MySQL
	case parser.EngineType(db.TiDB):
		engine = parser.TiDB
	default:
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid database engine %s", request.EngineType))
	}

	diff, err := differ.SchemaDiff(engine, request.SourceSchema, request.TargetSchema, false /* ignoreCaseSensitive */)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to compute diff between source and target schemas").SetInternal(err)
	}

	return c.JSON(http.StatusOK, diff)
}

func (s *Server) sqlCheck(
	ctx context.Context,
	dbType advisorDB.Type,
	dbCharacterSet string,
	dbCollation string,
	environmentID int,
	statement string,
	catalog catalog.Catalog,
	driver *sql.DB,
) (advisor.Status, []advisor.Advice, error) {
	var adviceList []advisor.Advice
	policy, err := s.store.GetSQLReviewPolicy(ctx, environmentID)
	if err != nil {
		if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
			return advisor.Success, nil, nil
		}
		return advisor.Error, nil, err
	}

	res, err := advisor.SQLReviewCheck(statement, policy.RuleList, advisor.SQLReviewCheckContext{
		Charset:   dbCharacterSet,
		Collation: dbCollation,
		DbType:    dbType,
		Catalog:   catalog,
		Driver:    driver,
		Context:   ctx,
	})
	if err != nil {
		return advisor.Error, nil, err
	}

	adviceLevel := advisor.Success
	for _, advice := range res {
		switch advice.Status {
		case advisor.Warn:
			if adviceLevel != advisor.Error {
				adviceLevel = advisor.Warn
			}
		case advisor.Error:
			adviceLevel = advisor.Error
		case advisor.Success:
			continue
		}

		adviceList = append(adviceList, advice)
	}

	return adviceLevel, adviceList, nil
}
