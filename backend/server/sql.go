package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	metricAPI "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"

	"github.com/bytebase/bytebase/backend/store"
)

func (s *Server) registerSQLRoutes(g *echo.Group) {
	g.POST("/sql/execute/admin", func(c echo.Context) error {
		ctx := c.Request().Context()
		exec := &api.SQLExecute{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, exec); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed sql execute request").SetInternal(err)
		}

		if exec.InstanceID == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed sql execute request, missing instanceId")
		}
		if len(exec.Statement) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed sql execute request, missing sql statement")
		}
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &exec.InstanceID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", exec.InstanceID)).SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", exec.InstanceID))
		}
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &exec.DatabaseName})
		if err != nil {
			return err
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database %q not found", exec.DatabaseName))
		}
		// Admin API always executes with read-only off.
		exec.Readonly = false
		start := time.Now().UnixNano()

		singleSQLResults, queryErr := func() ([]api.SingleSQLResult, error) {
			driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, database)
			if err != nil {
				return nil, err
			}
			defer driver.Close(ctx)

			// TODO(p0ny): refactor
			if instance.Engine == db.MongoDB || instance.Engine == db.Spanner || instance.Engine == db.Redis {
				data, err := driver.QueryConn(ctx, nil, exec.Statement, &db.QueryContext{
					Limit:               exec.Limit,
					ReadOnly:            false,
					CurrentDatabase:     exec.DatabaseName,
					SensitiveSchemaInfo: nil,
				})
				if err != nil {
					return nil, err
				}

				dataJSON, err := json.Marshal(data)
				if err != nil {
					return nil, err
				}
				return []api.SingleSQLResult{
					{
						Data: string(dataJSON),
					},
				}, nil
			}

			sqlDB := driver.GetDB()
			conn, err := sqlDB.Conn(ctx)
			if err != nil {
				return nil, err
			}
			defer conn.Close()

			var singleSQLResults []api.SingleSQLResult
			// We split the query into multiple statements and execute them one by one for MySQL and PostgreSQL.
			if instance.Engine == db.MySQL || instance.Engine == db.TiDB || instance.Engine == db.MariaDB || instance.Engine == db.Postgres || instance.Engine == db.Oracle || instance.Engine == db.Redshift || instance.Engine == db.OceanBase {
				singleSQLs, err := parser.SplitMultiSQL(parser.EngineType(instance.Engine), exec.Statement)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to split statements")
				}
				for _, singleSQL := range singleSQLs {
					rowSet, err := driver.QueryConn(ctx, conn, singleSQL.Text, &db.QueryContext{
						Limit:               exec.Limit,
						ReadOnly:            false,
						CurrentDatabase:     exec.DatabaseName,
						SensitiveSchemaInfo: nil,
					})
					if err != nil {
						singleSQLResults = append(singleSQLResults, api.SingleSQLResult{
							Error: err.Error(),
						})
						continue
					}
					data, err := json.Marshal(rowSet)
					if err != nil {
						singleSQLResults = append(singleSQLResults, api.SingleSQLResult{
							Error: err.Error(),
						})
						continue
					}
					singleSQLResults = append(singleSQLResults, api.SingleSQLResult{
						Data: string(data),
					})
				}
			} else {
				if err := util.ApplyMultiStatements(strings.NewReader(exec.Statement), func(statement string) error {
					rowSet, err := driver.QueryConn(ctx, conn, statement, &db.QueryContext{
						Limit:               exec.Limit,
						ReadOnly:            false,
						CurrentDatabase:     exec.DatabaseName,
						SensitiveSchemaInfo: nil,
					})
					if err != nil {
						singleSQLResults = append(singleSQLResults, api.SingleSQLResult{
							Error: err.Error(),
						})
						//nolint
						return nil
					}
					data, err := json.Marshal(rowSet)
					if err != nil {
						singleSQLResults = append(singleSQLResults, api.SingleSQLResult{
							Error: err.Error(),
						})
						//nolint
						return nil
					}
					singleSQLResults = append(singleSQLResults, api.SingleSQLResult{
						Data: string(data),
					})
					return nil
				}); err != nil {
					// It should never happen.
					return nil, err
				}
			}
			return singleSQLResults, nil
		}()

		level := api.ActivityInfo
		errMessage := ""
		if err != nil {
			level = api.ActivityError
			errMessage += err.Error()
		}
		for idx, singleSQLResult := range singleSQLResults {
			level = api.ActivityError
			if singleSQLResult.Error != "" {
				errMessage += fmt.Sprintf("\nFor query statement #%d: %s", idx+1, singleSQLResult.Error)
			}
		}
		var databaseID int
		if database != nil {
			databaseID = database.UID
		}
		if err := s.createSQLEditorQueryActivity(ctx, c, level, exec.InstanceID, api.ActivitySQLEditorQueryPayload{
			Statement:              exec.Statement,
			DurationNs:             time.Now().UnixNano() - start,
			InstanceID:             instance.UID,
			DeprecatedInstanceName: instance.Title,
			DatabaseID:             databaseID,
			DatabaseName:           exec.DatabaseName,
			Error:                  errMessage,
		}); err != nil {
			return err
		}

		resultSet := &api.SQLResultSet{
			AdviceList:          []advisor.Advice{},
			SingleSQLResultList: singleSQLResults,
		}

		if queryErr != nil {
			resultSet.Error = queryErr.Error()
			if s.profile.Mode == common.ReleaseModeDev {
				log.Error("Failed to execute query",
					zap.Error(err),
					zap.String("statement", exec.Statement),
				)
			} else {
				log.Debug("Failed to execute query",
					zap.Error(err),
					zap.String("statement", exec.Statement),
				)
			}
		}

		s.MetricReporter.Report(ctx, &metric.Metric{
			Name:  metricAPI.SQLEditorExecutionMetricName,
			Value: 1,
			Labels: map[string]any{
				"engine":     instance.Engine,
				"readonly":   exec.Readonly,
				"admin_mode": true,
			},
		})

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, resultSet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal sql result set response").SetInternal(err)
		}
		return nil
	})
}

func (s *Server) createSQLEditorQueryActivity(ctx context.Context, c echo.Context, level api.ActivityLevel, containerID int, payload api.ActivitySQLEditorQueryPayload) error {
	activityBytes, err := json.Marshal(payload)
	if err != nil {
		log.Warn("Failed to marshal activity after executing sql statement",
			zap.String("database_name", payload.DatabaseName),
			zap.Int("instance_id", payload.InstanceID),
			zap.String("statement", payload.Statement),
			zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
	}

	activityCreate := &api.ActivityCreate{
		CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
		Type:        api.ActivitySQLEditorQuery,
		ContainerID: containerID,
		Level:       level,
		Comment: fmt.Sprintf("Executed `%q` in database %q of instance %d.",
			payload.Statement, payload.DatabaseName, payload.InstanceID),
		Payload: string(activityBytes),
	}

	if _, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{}); err != nil {
		log.Warn("Failed to create activity after executing sql statement",
			zap.String("database_name", payload.DatabaseName),
			zap.Int("instance_id", payload.InstanceID),
			zap.String("statement", payload.Statement),
			zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create activity").SetInternal(err)
	}
	return nil
}
