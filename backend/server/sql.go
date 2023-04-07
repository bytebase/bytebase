package server

import (
	"context"
	"database/sql"
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
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	advisorDB "github.com/bytebase/bytebase/backend/plugin/advisor/db"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/plugin/parser"
	"github.com/bytebase/bytebase/backend/plugin/parser/ast"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *Server) registerSQLRoutes(g *echo.Group) {
	g.POST("/sql/ping", func(c echo.Context) error {
		ctx := c.Request().Context()
		connectionInfo := &api.ConnectionInfo{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, connectionInfo); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed sql ping request").SetInternal(err)
		}
		if err := s.disallowBytebaseStore(connectionInfo.Engine, connectionInfo.Host, connectionInfo.Port); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}

		password := connectionInfo.Password
		// Instance detail page has a Test Connection button, if user doesn't input new password and doesn't specify
		// to use empty password, we want the connection to use the existing password to test the connection, however,
		// we do not transfer the password back to client, thus the client will pass the instanceID to let server
		// retrieve the password.
		if password == "" && !connectionInfo.UseEmptyPassword && connectionInfo.InstanceID != nil {
			instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: connectionInfo.InstanceID})
			if err != nil {
				return err
			}
			if instance == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("instance %d not found", *connectionInfo.InstanceID))
			}
			for _, ds := range instance.DataSources {
				if ds.Type == api.Admin {
					password, err = common.Unobfuscate(ds.ObfuscatedPassword, s.secret)
					if err != nil {
						return err
					}
					break
				}
			}
		}

		var tlsConfig db.TLSConfig
		supportTLS := (connectionInfo.Engine == db.ClickHouse || connectionInfo.Engine == db.MySQL || connectionInfo.Engine == db.TiDB || connectionInfo.Engine == db.MariaDB)
		if supportTLS {
			if connectionInfo.SslCa == nil && connectionInfo.SslCert == nil && connectionInfo.SslKey == nil && connectionInfo.InstanceID != nil {
				// Frontend will not pass ssl related field if user don't modify ssl suite, we need get ssl suite from database for this case.
				instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: connectionInfo.InstanceID})
				if err != nil {
					return err
				}
				if instance == nil {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("instance %d not found", *connectionInfo.InstanceID))
				}
				for _, ds := range instance.DataSources {
					if ds.ObfuscatedSslCa != "" || ds.ObfuscatedSslCert != "" || ds.ObfuscatedSslKey != "" {
						sslCa, err := common.Unobfuscate(ds.ObfuscatedSslCa, s.secret)
						if err != nil {
							return err
						}
						sslKey, err := common.Unobfuscate(ds.ObfuscatedSslKey, s.secret)
						if err != nil {
							return err
						}
						sslCert, err := common.Unobfuscate(ds.ObfuscatedSslCert, s.secret)
						if err != nil {
							return err
						}
						tlsConfig = db.TLSConfig{
							SslCA:   sslCa,
							SslKey:  sslKey,
							SslCert: sslCert,
						}
						break
					}
				}
			} else if connectionInfo.SslCa != nil && connectionInfo.SslCert != nil && connectionInfo.SslKey != nil {
				// Users may add instance and click test connection button now, we need get ssl suite from request for this case.
				tlsConfig = db.TLSConfig{
					SslCA:   *connectionInfo.SslCa,
					SslCert: *connectionInfo.SslCert,
					SslKey:  *connectionInfo.SslKey,
				}
			} else {
				// Unexpected case
				return echo.NewHTTPError(http.StatusBadRequest, "TLS/SSL suite must all be set or not be set")
			}
		}

		db, err := db.Open(
			ctx,
			connectionInfo.Engine,
			db.DriverConfig{},
			db.ConnectionConfig{
				Username:               connectionInfo.Username,
				Password:               password,
				Host:                   connectionInfo.Host,
				Port:                   connectionInfo.Port,
				Database:               connectionInfo.Database,
				TLSConfig:              tlsConfig,
				SRV:                    connectionInfo.SRV,
				AuthenticationDatabase: connectionInfo.AuthenticationDatabase,
				SID:                    connectionInfo.SID,
				ServiceName:            connectionInfo.ServiceName,
			},
			db.ConnectionContext{},
		)

		resultSet := &api.SQLResultSet{}
		if err != nil {
			hostPort := connectionInfo.Host
			if connectionInfo.Port != "" {
				hostPort += ":" + connectionInfo.Port
			}
			resultSet.Error = errors.Wrapf(err, "failed to connect %q for user %q", hostPort, connectionInfo.Username).Error()
		} else {
			defer db.Close(ctx)
			if err := db.Ping(ctx); err != nil {
				resultSet.Error = err.Error()
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, resultSet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal sql result set response").SetInternal(err)
		}
		return nil
	})

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
			composedInstance, err := s.store.GetInstanceByID(ctx, *sync.InstanceID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %d", *sync.InstanceID)).SetInternal(err)
			}
			if _, err := s.SchemaSyncer.SyncInstance(ctx, instance); err != nil {
				resultSet.Error = err.Error()
			}
			// Sync all databases in the instance asynchronously.
			s.stateCfg.InstanceDatabaseSyncChan <- composedInstance
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
		principalID := c.Get(getPrincipalIDContextKey()).(int)
		role := c.Get(getRoleContextKey()).(api.Role)
		var database *store.DatabaseMessage
		if exec.DatabaseName != "" {
			database, err = s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{EnvironmentID: &instance.EnvironmentID, InstanceID: &instance.ResourceID, DatabaseName: &exec.DatabaseName})
			if err != nil {
				return err
			}
			if database == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database %q not found", exec.DatabaseName))
			}
			// Database Access Control
			hasAccessRights, err := s.hasDatabaseAccessRights(ctx, principalID, role, database)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to check access control for database: %q", exec.DatabaseName)).SetInternal(err)
			}
			if !hasAccessRights {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformed sql execute request, no permission to access database %q", exec.DatabaseName))
			}
		}

		// Database Access Control for MySQL dialect.
		// MySQL dialect can query cross the database.
		// We need special check.
		if instance.Engine == db.MySQL || instance.Engine == db.TiDB || instance.Engine == db.MariaDB {
			databaseList, err := parser.ExtractDatabaseList(parser.MySQL, exec.Statement)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to extract database list: %q", exec.Statement)).SetInternal(err)
			}

			if exec.DatabaseName != "" {
				// Disallow cross-database query if specify database.
				for _, databaseName := range databaseList {
					upperDatabaseName := strings.ToUpper(databaseName)
					// We allow querying information schema.
					if upperDatabaseName == "" || upperDatabaseName == "INFORMATION_SCHEMA" {
						continue
					}
					if databaseName != exec.DatabaseName {
						return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformed sql execute request, specify database %q but access database %q", exec.DatabaseName, databaseName))
					}
				}
			} else {
				// Check database access rights.
				for _, databaseName := range databaseList {
					if databaseName == "" {
						// We have already checked the current database access rights.
						continue
					}
					accessDatabase, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{EnvironmentID: &instance.EnvironmentID, InstanceID: &instance.ResourceID, DatabaseName: &databaseName})
					if err != nil {
						if httpErr, ok := err.(*echo.HTTPError); ok && httpErr.Code == echo.ErrNotFound.Code {
							// If database not found, skip.
							continue
						}
						return err
					}

					hasAccessRights, err := s.hasDatabaseAccessRights(ctx, principalID, role, accessDatabase)
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to check access control for database: %q", accessDatabase.DatabaseName)).SetInternal(err)
					}
					if !hasAccessRights {
						return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformed sql execute request, no permission to access database %q", accessDatabase.DatabaseName))
					}
				}
			}
		}

		adviceLevel := advisor.Success
		adviceList := []advisor.Advice{}

		if api.IsSQLReviewSupported(instance.Engine) && exec.DatabaseName != "" {
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
			connection, err := driver.GetDBConnection(ctx, exec.DatabaseName)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get database connection").SetInternal(err)
			}

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
		case db.MySQL, db.TiDB, db.MariaDB:
			databaseList, err := parser.ExtractDatabaseList(parser.MySQL, exec.Statement)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get database list: %s", exec.Statement)).SetInternal(err)
			}

			sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance, databaseList, exec.DatabaseName)
			if err != nil {
				return err
			}
		case db.Postgres:
			sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance, []string{exec.DatabaseName}, exec.DatabaseName)
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

			sqlDB, err := driver.GetDBConnection(ctx, exec.DatabaseName)
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
			Labels: map[string]interface{}{
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
		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{EnvironmentID: &instance.EnvironmentID, InstanceID: &instance.ResourceID, DatabaseName: &exec.DatabaseName})
		if err != nil {
			return err
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database %q not found", exec.DatabaseName))
		}
		// Admin API always executes with read-only off.
		exec.Readonly = true
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

			sqlDB, err := driver.GetDBConnection(ctx, exec.DatabaseName)
			if err != nil {
				return nil, err
			}
			conn, err := sqlDB.Conn(ctx)
			if err != nil {
				return nil, err
			}
			defer conn.Close()

			var singleSQLResults []api.SingleSQLResult
			// We split the query into multiple statements and execute them one by one for MySQL and PostgreSQL.
			if instance.Engine == db.MySQL || instance.Engine == db.TiDB || instance.Engine == db.MariaDB || instance.Engine == db.Postgres || instance.Engine == db.Oracle {
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
					rowSet, err := driver.QueryConn(ctx, conn, exec.Statement, &db.QueryContext{
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
			Labels: map[string]interface{}{
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

		database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{EnvironmentID: &instance.EnvironmentID, InstanceID: &instance.ResourceID, DatabaseName: &databaseName})
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

func (s *Server) hasDatabaseAccessRights(ctx context.Context, principalID int, role api.Role, database *store.DatabaseMessage) (bool, error) {
	// Workspace Owners and DBAs always have database access rights.
	if role == api.Owner || role == api.DBA {
		return true, nil
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{ResourceID: &database.ProjectID})
	if err != nil {
		return false, err
	}
	environment, err := s.store.GetEnvironmentV2(ctx, &store.FindEnvironmentMessage{ResourceID: &database.EnvironmentID})
	if err != nil {
		return false, err
	}

	// Only project member can access database.
	projectPolicy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return false, err
	}
	if !hasActiveProjectMembership(principalID, projectPolicy) {
		return false, nil
	}

	// calculate the effective policy.
	databasePolicy, inheritFromEnvironment, err := s.store.GetAccessControlPolicy(ctx, api.PolicyResourceTypeDatabase, database.UID)
	if err != nil {
		return false, err
	}

	environmentPolicy, _, err := s.store.GetAccessControlPolicy(ctx, api.PolicyResourceTypeEnvironment, environment.UID)
	if err != nil {
		return false, err
	}

	if !inheritFromEnvironment {
		// Use database policy.
		return databasePolicy != nil && len(databasePolicy.DisallowRuleList) == 0, nil
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
	return hasAccessRights, nil
}
