package server

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
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
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	advisorDB "github.com/bytebase/bytebase/backend/plugin/advisor/db"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *Server) registerSQLRoutes(g *echo.Group) {
	g.POST("/sql/sync-schema", func(c echo.Context) error {
		ctx := c.Request().Context()
		sync := &api.SQLSyncSchema{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, sync); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed sql sync schema request").SetInternal(err)
		}
		if (sync.InstanceID == nil) == (sync.DatabaseID == nil) {
			return echo.NewHTTPError(http.StatusBadRequest, "Either InstanceID or DatabaseID should be set.")
		}

		var resultSet api.SQLResultSet
		if sync.InstanceID != nil {
			instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: sync.InstanceID})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %d", *sync.InstanceID)).SetInternal(err)
			}
			if instance == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", *sync.InstanceID))
			}
			if _, err := s.SchemaSyncer.SyncInstance(ctx, instance); err != nil {
				resultSet.Error = err.Error()
			}
			// Sync all databases in the instance asynchronously.
			s.stateCfg.InstanceDatabaseSyncChan <- instance
		}
		if sync.DatabaseID != nil {
			database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{UID: sync.DatabaseID})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", *sync.DatabaseID)).SetInternal(err)
			}
			if database == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", *sync.DatabaseID))
			}
			if err := s.SchemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
				resultSet.Error = err.Error()
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, &resultSet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal sql result set response").SetInternal(err)
		}
		return nil
	})

	g.POST("/sql/execute", func(c echo.Context) error {
		ctx := c.Request().Context()
		principalID := c.Get(getPrincipalIDContextKey()).(int)
		role := c.Get(getRoleContextKey()).(api.Role)
		user, err := s.store.GetUserByID(ctx, principalID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Principal not found").SetInternal(err)
		}

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
		if !exec.Readonly {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed sql execute request, only support readonly sql statement")
		}

		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &exec.InstanceID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", exec.InstanceID)).SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", exec.InstanceID))
		}

		switch instance.Engine {
		case db.Postgres:
			if _, err := parser.Parse(parser.Postgres, parser.ParseContext{}, exec.Statement); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid sql statement: %v", err))
			}
		case db.Oracle:
			if _, err := parser.ParsePLSQL(exec.Statement); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid sql statement: %v", err))
			}
		case db.TiDB:
			if _, err := parser.ParseTiDB(exec.Statement, "", ""); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid sql statement: %v", err))
			}
		case db.MySQL, db.OceanBase:
			if _, err := parser.ParseMySQL(exec.Statement, "", ""); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid sql statement: %v", err))
			}
		}

		if !parser.ValidateSQLForEditor(convertToParserEngine(instance.Engine), exec.Statement) {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed sql execute request, only support SELECT sql statement")
		}

		environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &instance.EnvironmentID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch environment ID: %s", instance.EnvironmentID)).SetInternal(err)
		}
		if environment == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Environment ID not found: %s", instance.EnvironmentID))
		}

		databaseNames, err := getDatabasesFromQuery(instance.Engine, exec.DatabaseName, exec.Statement)
		if err != nil {
			return err
		}
		// Check database access rights.
		isExport := exec.ExportFormat == "CSV" || exec.ExportFormat == "JSON"
		if role != api.Owner && role != api.DBA {
			var project *store.ProjectMessage
			var databases []*store.DatabaseMessage
			for _, databaseName := range databaseNames {
				database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &databaseName})
				if err != nil {
					if httpErr, ok := err.(*echo.HTTPError); ok && httpErr.Code == echo.ErrNotFound.Code {
						// If database not found, skip.
						continue
					}
					return err
				}
				databases = append(databases, database)
				p, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
				if err != nil {
					return err
				}
				if project == nil {
					project = p
				}
				if project.UID != p.UID {
					return echo.NewHTTPError(http.StatusBadRequest, "allow querying databases within the same project only")
				}
			}
			if project == nil {
				return echo.NewHTTPError(http.StatusBadRequest, "project not found")
			}
			projectPolicy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &project.ResourceID})
			if err != nil {
				return err
			}
			// TODO(d): perfect matching condition expression.
			var usedExpression string
			for _, database := range databases {
				databaseResourceURL := fmt.Sprintf("instances/%s/databases/%s", instance.ResourceID, database.DatabaseName)
				attributes := map[string]any{
					"request.time":          time.Now(),
					"resource.database":     databaseResourceURL,
					"request.statement":     base64.StdEncoding.EncodeToString([]byte(exec.Statement)),
					"request.row_limit":     exec.Limit,
					"request.export_format": exec.ExportFormat,
				}

				ok, ue, err := s.hasDatabaseAccessRights(ctx, principalID, projectPolicy, database, environment, attributes, isExport)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to check access control for database: %q", database.DatabaseName)).SetInternal(err)
				}
				if !ok {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Permission denied to access database %q", database.DatabaseName))
				}
				usedExpression = ue
			}
			if isExport {
				newPolicy := removeExportBinding(principalID, usedExpression, projectPolicy)
				if _, err := s.store.SetProjectIAMPolicy(ctx, newPolicy, api.SystemBotID, project.UID); err != nil {
					return err
				}
				// Post project IAM policy update activity.
				if _, err := s.ActivityManager.CreateActivity(ctx, &api.ActivityCreate{
					CreatorID:   api.SystemBotID,
					ContainerID: project.UID,
					Type:        api.ActivityProjectMemberCreate,
					Level:       api.ActivityInfo,
					Comment:     fmt.Sprintf("Granted %s to %s (%s).", user.Name, user.Email, api.Role(common.ProjectExporter)),
				}, &activity.Metadata{}); err != nil {
					log.Warn("Failed to create project activity", zap.Error(err))
				}
			}
		}

		var database *store.DatabaseMessage
		if exec.DatabaseName != "" {
			database, err = s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &exec.DatabaseName})
			if err != nil {
				return err
			}
		}

		adviceLevel := advisor.Success
		adviceList := []advisor.Advice{}

		if api.IsSQLReviewSupported(instance.Engine) && exec.DatabaseName != "" && !isExport {
			// Skip SQL review for exporting data.
			dbType, err := advisorDB.ConvertToAdvisorDBType(string(instance.Engine))
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to convert db type %v into advisor db type", instance.Engine))
			}
			dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
			if err != nil {
				return err
			}
			// The schema isn't loaded yet, let's load it on-demand.
			if dbSchema == nil {
				if err := s.SchemaSyncer.SyncDatabaseSchema(ctx, database, true /* force */); err != nil {
					return err
				}
			}
			dbSchema, err = s.store.GetDBSchema(ctx, database.UID)
			if err != nil {
				return err
			}
			if dbSchema == nil {
				return errors.Errorf("database schema %v not found", database.UID)
			}

			catalog, err := s.store.NewCatalog(ctx, database.UID, instance.Engine, advisor.SyntaxModeNormal)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create a catalog")
			}

			driver, err := s.dbFactory.GetReadOnlyDatabaseDriver(ctx, instance, exec.DatabaseName)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get database driver").SetInternal(err)
			}
			defer driver.Close(ctx)
			connection := driver.GetDB()
			adviceLevel, adviceList, err = s.sqlCheck(
				ctx,
				dbType,
				dbSchema.Metadata.CharacterSet,
				dbSchema.Metadata.Collation,
				environment.UID,
				exec.Statement,
				catalog,
				connection,
			)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check SQL review policy").SetInternal(err)
			}

			if adviceLevel == advisor.Error {
				if err := s.createSQLEditorQueryActivity(ctx, c, api.ActivityError, exec.InstanceID, api.ActivitySQLEditorQueryPayload{
					Statement:              exec.Statement,
					DurationNs:             0,
					InstanceID:             instance.UID,
					DeprecatedInstanceName: instance.Title,
					DatabaseID:             database.UID,
					DatabaseName:           exec.DatabaseName,
					Error:                  "",
					AdviceList:             adviceList,
				}); err != nil {
					return err
				}

				resultSet := &api.SQLResultSet{
					AdviceList: adviceList,
				}

				c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
				if err := jsonapi.MarshalPayload(c.Response().Writer, resultSet); err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal sql result set response").SetInternal(err)
				}
				return nil
			}
		}

		var sensitiveSchemaInfo *db.SensitiveSchemaInfo
		switch instance.Engine {
		case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
			databaseList, err := parser.ExtractDatabaseList(parser.MySQL, exec.Statement)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get database list: %s", exec.Statement)).SetInternal(err)
			}

			sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance, databaseList, exec.DatabaseName)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get sensitive schema info: %s", exec.Statement)).SetInternal(err)
			}
		case db.Postgres:
			sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance, []string{exec.DatabaseName}, exec.DatabaseName)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get sensitive schema info: %s", exec.Statement)).SetInternal(err)
			}
		}

		start := time.Now().UnixNano()

		singleSQLResults, queryErr := func() ([]api.SingleSQLResult, error) {
			driver, err := s.dbFactory.GetReadOnlyDatabaseDriver(ctx, instance, exec.DatabaseName)
			if err != nil {
				return nil, err
			}
			defer driver.Close(ctx)

			// TODO(p0ny): refactor
			if instance.Engine == db.MongoDB || instance.Engine == db.Spanner || instance.Engine == db.Redis {
				data, err := driver.QueryConn(ctx, nil, exec.Statement, &db.QueryContext{
					Limit:                 exec.Limit,
					ReadOnly:              true,
					CurrentDatabase:       exec.DatabaseName,
					SensitiveDataMaskType: db.SensitiveDataMaskTypeDefault,
					SensitiveSchemaInfo:   sensitiveSchemaInfo,
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
			if err != nil {
				return nil, err
			}
			conn, err := sqlDB.Conn(ctx)
			if err != nil {
				return nil, err
			}
			defer conn.Close()

			var singleSQLResults []api.SingleSQLResult

			rowSet, err := driver.QueryConn(ctx, conn, exec.Statement, &db.QueryContext{
				Limit:           exec.Limit,
				ReadOnly:        true,
				CurrentDatabase: exec.DatabaseName,
				// TODO(rebelice): we cannot deal with multi-SensitiveDataMaskType now. Fix it.
				SensitiveDataMaskType: db.SensitiveDataMaskTypeDefault,
				SensitiveSchemaInfo:   sensitiveSchemaInfo,
			})
			if err != nil {
				singleSQLResults = append(singleSQLResults, api.SingleSQLResult{
					Error: err.Error(),
				})
				//nolint
				return singleSQLResults, nil
			}
			data, err := json.Marshal(rowSet)
			if err != nil {
				singleSQLResults = append(singleSQLResults, api.SingleSQLResult{
					Error: err.Error(),
				})
				//nolint
				return singleSQLResults, nil
			}
			singleSQLResults = append(singleSQLResults, api.SingleSQLResult{
				Data: string(data),
			})
			return singleSQLResults, nil
		}()

		if instance.Engine == db.Postgres {
			stmts, err := parser.Parse(parser.Postgres, parser.ParseContext{}, exec.Statement)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to parse: %s", exec.Statement)).SetInternal(err)
			}
			if len(stmts) != 1 {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Expected one statement, but found %d, statement: %s", len(stmts), exec.Statement))
			}

			if _, ok := stmts[0].(*ast.ExplainStmt); ok {
				if len(singleSQLResults) != 1 {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Expected one result, but found %d, statement: %s, consider syntax error", len(singleSQLResults), exec.Statement))
				}
				indexAdvice := checkPostgreSQLIndexHit(exec.Statement, singleSQLResults[0].Data)
				if len(indexAdvice) > 0 {
					adviceLevel = advisor.Error
					adviceList = append(adviceList, indexAdvice...)
				}
			}
		}

		if len(adviceList) == 0 {
			adviceList = append(adviceList, advisor.Advice{
				Status:  advisor.Success,
				Code:    advisor.Ok,
				Title:   "OK",
				Content: "",
			})
		}

		level := api.ActivityInfo
		errMessage := ""
		switch adviceLevel {
		case advisor.Warn:
			level = api.ActivityWarn
		case advisor.Error:
			level = api.ActivityError
		}
		if queryErr != nil {
			level = api.ActivityError
			errMessage = queryErr.Error()
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
			AdviceList:             adviceList,
		}); err != nil {
			return err
		}

		resultSet := &api.SQLResultSet{
			AdviceList:          adviceList,
			SingleSQLResultList: singleSQLResults,
		}

		if queryErr != nil {
			resultSet.Error = queryErr.Error()
			if s.profile.Mode == common.ReleaseModeDev {
				log.Error("Failed to execute query",
					zap.Error(err),
					zap.String("statement", exec.Statement),
					zap.Array("advice", advisor.ZapAdviceArray(resultSet.AdviceList)),
				)
			} else {
				log.Debug("Failed to execute query",
					zap.Error(err),
					zap.String("statement", exec.Statement),
					zap.Array("advice", advisor.ZapAdviceArray(resultSet.AdviceList)),
				)
			}
		}

		s.MetricReporter.Report(ctx, &metric.Metric{
			Name:  metricAPI.SQLEditorExecutionMetricName,
			Value: 1,
			Labels: map[string]any{
				"engine":     instance.Engine,
				"readonly":   exec.Readonly,
				"admin_mode": false,
			},
		})

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, resultSet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal sql result set response").SetInternal(err)
		}
		return nil
	})

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
			driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, exec.DatabaseName)
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

func convertToParserEngine(engine db.Type) parser.EngineType {
	// convert to parser engine
	switch engine {
	case db.Postgres:
		return parser.Postgres
	case db.MySQL:
		return parser.MySQL
	case db.TiDB:
		return parser.TiDB
	case db.MariaDB:
		return parser.MariaDB
	case db.Oracle:
		return parser.Oracle
	case db.MSSQL:
		return parser.MSSQL
	case db.OceanBase:
		return parser.OceanBase
	}
	return parser.Standard
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

func checkPostgreSQLIndexHit(statement string, plan string) []advisor.Advice {
	plans := strings.Split(plan, "\n")
	haveSeqScan := false
	haveIndexScan := false
	for _, row := range plans {
		if strings.Contains(row, "Seq Scan") {
			haveSeqScan = true
			continue
		}
		if strings.Contains(row, "Index Scan") {
			haveIndexScan = true
			continue
		}
	}
	if haveSeqScan && !haveIndexScan {
		return []advisor.Advice{
			{
				Status:  advisor.Error,
				Code:    advisor.NotUseIndex,
				Title:   "Query does not use index",
				Content: fmt.Sprintf("statement %q does not use any index", statement),
			},
		}
	}
	return nil
}

func (s *Server) getSensitiveSchemaInfo(ctx context.Context, instance *store.InstanceMessage, databaseList []string, currentDatabase string) (*db.SensitiveSchemaInfo, error) {
	type sensitiveDataMap map[api.SensitiveData]api.SensitiveDataMaskType
	isEmpty := true
	result := &db.SensitiveSchemaInfo{
		DatabaseList: []db.DatabaseSchema{},
	}
	for _, name := range databaseList {
		databaseName := name
		if name == "" {
			if currentDatabase == "" {
				continue
			}
			databaseName = currentDatabase
		}

		if isExcludeDatabase(instance.Engine, databaseName) {
			continue
		}

		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID, DatabaseName: &databaseName})
		if err != nil {
			return nil, err
		}
		if database == nil {
			return nil, errors.Errorf("database %q not found", databaseName)
		}

		columnMap := make(sensitiveDataMap)
		policy, err := s.store.GetSensitiveDataPolicy(ctx, database.UID)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find sensitive data policy for database %q in instance %q", databaseName, instance.Title))
		}
		if len(policy.SensitiveDataList) == 0 {
			// If there is no sensitive data policy, return nil to skip mask sensitive data.
			return nil, nil
		}

		for _, data := range policy.SensitiveDataList {
			columnMap[api.SensitiveData{
				Schema: data.Schema,
				Table:  data.Table,
				Column: data.Column,
			}] = data.Type
		}

		dbSchema, err := s.store.GetDBSchema(ctx, database.UID)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find table list for database %q", databaseName))
		}

		databaseSchema := db.DatabaseSchema{
			Name:      databaseName,
			TableList: []db.TableSchema{},
		}
		for _, schema := range dbSchema.Metadata.Schemas {
			for _, table := range schema.Tables {
				tableSchema := db.TableSchema{
					Name:       table.Name,
					ColumnList: []db.ColumnInfo{},
				}
				if instance.Engine == db.Postgres {
					tableSchema.Name = fmt.Sprintf("%s.%s", schema.Name, table.Name)
				}
				for _, column := range table.Columns {
					_, sensitive := columnMap[api.SensitiveData{
						Schema: schema.Name,
						Table:  table.Name,
						Column: column.Name,
					}]
					tableSchema.ColumnList = append(tableSchema.ColumnList, db.ColumnInfo{
						Name:      column.Name,
						Sensitive: sensitive,
					})
				}
				databaseSchema.TableList = append(databaseSchema.TableList, tableSchema)
			}
		}
		if len(databaseSchema.TableList) > 0 {
			isEmpty = false
		}
		result.DatabaseList = append(result.DatabaseList, databaseSchema)
	}

	if isEmpty {
		// If there is no tables, this query may access system databases, such as INFORMATION_SCHEMA.
		// Skip to extract sensitive column for this query.
		result = nil
	}
	return result, nil
}

func isExcludeDatabase(dbType db.Type, database string) bool {
	switch dbType {
	case db.MySQL, db.MariaDB:
		return isMySQLExcludeDatabase(database)
	case db.TiDB:
		if isMySQLExcludeDatabase(database) {
			return true
		}
		return database == "metrics_schema"
	default:
		return false
	}
}

func isMySQLExcludeDatabase(database string) bool {
	if strings.ToLower(database) == "information_schema" {
		return true
	}

	switch database {
	case "mysql":
	case "sys":
	case "performance_schema":
	default:
		return false
	}
	return true
}

func (s *Server) hasDatabaseAccessRights(ctx context.Context, principalID int, projectPolicy *store.IAMPolicyMessage, database *store.DatabaseMessage, environment *store.EnvironmentMessage, attributes map[string]any, isExport bool) (bool, string, error) {
	// Project IAM policy evaluation.
	pass := false
	var usedExpression string
	for _, binding := range projectPolicy.Bindings {
		if !((isExport && binding.Role == api.Role(common.ProjectExporter)) || (!isExport && binding.Role == api.Role(common.ProjectQuerier))) {
			continue
		}
		for _, member := range binding.Members {
			if member.ID != principalID {
				continue
			}
			ok, err := evaluateCondition(binding.Condition.Expression, attributes)
			if err != nil {
				log.Error("failed to evaluate condition", zap.Error(err), zap.String("condition", binding.Condition.Expression))
				break
			}
			if ok {
				pass = true
				usedExpression = binding.Condition.Expression
				break
			}
		}
		if pass {
			break
		}
	}
	if !pass {
		return false, "", nil
	}
	// calculate the effective policy.
	databasePolicy, inheritFromEnvironment, err := s.store.GetAccessControlPolicy(ctx, api.PolicyResourceTypeDatabase, database.UID)
	if err != nil {
		return false, "", err
	}

	environmentPolicy, _, err := s.store.GetAccessControlPolicy(ctx, api.PolicyResourceTypeEnvironment, environment.UID)
	if err != nil {
		return false, "", err
	}

	if !inheritFromEnvironment {
		// Use database policy.
		return databasePolicy != nil && len(databasePolicy.DisallowRuleList) == 0, "", nil
	}
	// Use both database policy and environment policy.
	hasAccessRights := true
	if environmentPolicy != nil {
		// Disallow by environment access policy.
		for _, rule := range environmentPolicy.DisallowRuleList {
			if rule.FullDatabase {
				hasAccessRights = false
				break
			}
		}
	}
	if databasePolicy != nil {
		// Allow by database access policy.
		hasAccessRights = true
	}
	return hasAccessRights, usedExpression, nil
}

func removeExportBinding(principalID int, usedExpression string, projectPolicy *store.IAMPolicyMessage) *store.IAMPolicyMessage {
	var newPolicy store.IAMPolicyMessage
	for _, binding := range projectPolicy.Bindings {
		if binding.Role != api.Role(common.ProjectExporter) || binding.Condition.Expression != usedExpression {
			newPolicy.Bindings = append(newPolicy.Bindings, binding)
			continue
		}

		var newMembers []*store.UserMessage
		for _, member := range binding.Members {
			if member.ID != principalID {
				newMembers = append(newMembers, member)
			}
		}
		if len(newMembers) == 0 {
			continue
		}
		newBinding := *binding
		newBinding.Members = newMembers
		newPolicy.Bindings = append(newPolicy.Bindings, &newBinding)
	}
	return &newPolicy
}

func getDatabasesFromQuery(engine db.Type, databaseName, statement string) ([]string, error) {
	if engine == db.MySQL || engine == db.TiDB || engine == db.MariaDB || engine == db.OceanBase {
		databases, err := parser.ExtractDatabaseList(parser.MySQL, statement)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to extract database list: %q", statement)).SetInternal(err)
		}

		if databaseName != "" {
			// Disallow cross-database query if specify database.
			for _, name := range databases {
				upperDatabaseName := strings.ToUpper(name)
				// We allow querying information schema.
				if upperDatabaseName == "" || upperDatabaseName == "INFORMATION_SCHEMA" {
					continue
				}
				if databaseName != name {
					return nil, echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformed sql execute request, specify database %q but access database %q", databaseName, name))
				}
			}
			return []string{databaseName}, nil
		}
		return databases, nil
	}
	if databaseName == "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "databaseName is required")
	}
	return []string{databaseName}, nil
}

var queryAttributes = []cel.EnvOption{
	cel.Variable("request.time", cel.TimestampType),
	cel.Variable("resource.database", cel.StringType),
	cel.Variable("request.statement", cel.StringType),
	cel.Variable("request.row_limit", cel.IntType),
	cel.Variable("request.export_format", cel.StringType),
}

func evaluateCondition(expression string, attributes map[string]any) (bool, error) {
	if expression == "" {
		return true, nil
	}
	env, err := cel.NewEnv(queryAttributes...)
	if err != nil {
		return false, err
	}
	ast, issues := env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, issues.Err()
	}
	prg, err := env.Program(ast)
	if err != nil {
		return false, err
	}

	out, _, err := prg.Eval(attributes)
	if err != nil {
		return false, err
	}
	val, err := out.ConvertToNative(reflect.TypeOf(false))
	if err != nil {
		return false, errors.Wrap(err, "expect bool result")
	}
	boolVal, ok := val.(bool)
	if !ok {
		return false, errors.Wrap(err, "failed to convert to bool")
	}
	return boolVal, nil
}
