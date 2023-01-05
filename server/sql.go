package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	advisorDB "github.com/bytebase/bytebase/plugin/advisor/db"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/ast"
	"github.com/bytebase/bytebase/server/component/activity"
	"github.com/bytebase/bytebase/store"
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
			adminPassword, err := s.store.GetInstanceAdminPasswordByID(ctx, *connectionInfo.InstanceID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve admin password for instance: %d", connectionInfo.InstanceID)).SetInternal(err)
			}
			password = adminPassword
		}

		var tlsConfig db.TLSConfig
		supportTLS := (connectionInfo.Engine == db.ClickHouse || connectionInfo.Engine == db.MySQL || connectionInfo.Engine == db.TiDB)
		if supportTLS {
			if connectionInfo.SslCa == nil && connectionInfo.SslCert == nil && connectionInfo.SslKey == nil && connectionInfo.InstanceID != nil {
				// Frontend will not pass ssl related field if user don't modify ssl suite, we need get ssl suite from database for this case.
				tc, err := s.store.GetInstanceSslSuiteByID(ctx, *connectionInfo.InstanceID)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve ssl suite for instance: %d", *connectionInfo.InstanceID)).SetInternal(err)
				}
				tlsConfig = tc
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
				TLSConfig:              tlsConfig,
				SRV:                    connectionInfo.SRV,
				AuthenticationDatabase: connectionInfo.AuthenticationDatabase,
				Database:               connectionInfo.Database,
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
			instance, err := s.store.GetInstanceByID(ctx, *sync.InstanceID)
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
			database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: sync.DatabaseID})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to database instance ID: %d", *sync.DatabaseID)).SetInternal(err)
			}
			if database == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", *sync.DatabaseID))
			}
			if err := s.SchemaSyncer.SyncDatabaseSchema(ctx, database.Instance, database.Name, true /* force */); err != nil {
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
		if !validateSQLSelectStatement(exec.Statement) {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed sql execute request, only support SELECT sql statement")
		}

		instance, err := s.store.GetInstanceByID(ctx, exec.InstanceID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", exec.InstanceID)).SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", exec.InstanceID))
		}
		principalID := c.Get(getPrincipalIDContextKey()).(int)
		role := c.Get(getRoleContextKey()).(api.Role)
		var database *api.Database
		if exec.DatabaseName != "" {
			database, err = s.getDatabase(ctx, instance.ID, exec.DatabaseName)
			if err != nil {
				return err
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
		if instance.Engine == db.MySQL || instance.Engine == db.TiDB {
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
					accessDatabase, err := s.getDatabase(ctx, instance.ID, databaseName)
					if err != nil {
						if httpErr, ok := err.(*echo.HTTPError); ok && httpErr.Code == echo.ErrNotFound.Code {
							// If database not found, skip.
							continue
						}
						return err
					}

					hasAccessRights, err := s.hasDatabaseAccessRights(ctx, principalID, role, accessDatabase)
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to check access control for database: %q", accessDatabase.Name)).SetInternal(err)
					}
					if !hasAccessRights {
						return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Malformed sql execute request, no permission to access database %q", accessDatabase.Name))
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

			catalog, err := s.store.NewCatalog(ctx, database.ID, instance.Engine)
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
				database.CharacterSet,
				database.Collation,
				instance.EnvironmentID,
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
					InstanceID:             instance.ID,
					DeprecatedInstanceName: instance.Name,
					DatabaseID:             database.ID,
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
		if instance.Engine == db.MySQL || instance.Engine == db.TiDB {
			databaseList, err := parser.ExtractDatabaseList(parser.MySQL, exec.Statement)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get database list: %s", exec.Statement)).SetInternal(err)
			}

			sensitiveSchemaInfo, err = s.getSensitiveSchemaInfo(ctx, instance.Engine, instance.ID, databaseList, exec.DatabaseName)
			if err != nil {
				return err
			}
		}

		start := time.Now().UnixNano()

		bytes, queryErr := func() ([]byte, error) {
			driver, err := s.dbFactory.GetReadOnlyDatabaseDriver(ctx, instance, exec.DatabaseName)
			if err != nil {
				return nil, err
			}
			defer driver.Close(ctx)

			rowSet, err := driver.Query(ctx, exec.Statement, &db.QueryContext{
				Limit:           exec.Limit,
				ReadOnly:        true,
				CurrentDatabase: exec.DatabaseName,
				// TODO(rebelice): we cannot deal with multi-SensitiveDataMaskType now. Fix it.
				SensitiveDataMaskType: db.SensitiveDataMaskTypeDefault,
				SensitiveSchemaInfo:   sensitiveSchemaInfo,
			})
			if err != nil {
				return nil, err
			}

			return json.Marshal(rowSet)
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
				indexAdvice := checkPostgreSQLIndexHit(exec.Statement, string(bytes))
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
			databaseID = database.ID
		}
		if err := s.createSQLEditorQueryActivity(ctx, c, level, exec.InstanceID, api.ActivitySQLEditorQueryPayload{
			Statement:              exec.Statement,
			DurationNs:             time.Now().UnixNano() - start,
			InstanceID:             instance.ID,
			DeprecatedInstanceName: instance.Name,
			DatabaseID:             databaseID,
			DatabaseName:           exec.DatabaseName,
			Error:                  errMessage,
			AdviceList:             adviceList,
		}); err != nil {
			return err
		}

		resultSet := &api.SQLResultSet{AdviceList: adviceList}
		if queryErr == nil {
			resultSet.Data = string(bytes)
			log.Debug("Query result advice",
				zap.String("statement", exec.Statement),
				zap.Array("advice", advisor.ZapAdviceArray(resultSet.AdviceList)),
			)
		} else {
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
		instance, err := s.store.GetInstanceByID(ctx, exec.InstanceID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", exec.InstanceID)).SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", exec.InstanceID))
		}
		var database *api.Database
		if exec.DatabaseName != "" {
			databaseFind := &api.DatabaseFind{
				InstanceID: &instance.ID,
				Name:       &exec.DatabaseName,
			}
			dbList, err := s.store.FindDatabase(ctx, databaseFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database `%s` for instance ID: %d", exec.DatabaseName, instance.ID)).SetInternal(err)
			}
			if len(dbList) == 0 {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database `%s` for instance ID: %d not found", exec.DatabaseName, instance.ID))
			}
			if len(dbList) > 1 {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("There are multiple database `%s` for instance ID: %d", exec.DatabaseName, instance.ID))
			}
			database = dbList[0]
		}

		// Admin API always executes with read-only off.
		exec.Readonly = true
		start := time.Now().UnixNano()

		bytes, queryErr := func() ([]byte, error) {
			driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, exec.DatabaseName)
			if err != nil {
				return nil, err
			}
			defer driver.Close(ctx)

			rowSet, err := driver.Query(ctx, exec.Statement, &db.QueryContext{
				Limit:               exec.Limit,
				ReadOnly:            false,
				CurrentDatabase:     exec.DatabaseName,
				SensitiveSchemaInfo: nil,
			})
			if err != nil {
				return nil, err
			}

			return json.Marshal(rowSet)
		}()

		level := api.ActivityInfo
		errMessage := ""
		if queryErr != nil {
			level = api.ActivityError
			errMessage = queryErr.Error()
		}
		var databaseID int
		if database != nil {
			databaseID = database.ID
		}
		if err := s.createSQLEditorQueryActivity(ctx, c, level, exec.InstanceID, api.ActivitySQLEditorQueryPayload{
			Statement:              exec.Statement,
			DurationNs:             time.Now().UnixNano() - start,
			InstanceID:             instance.ID,
			DeprecatedInstanceName: instance.Name,
			DatabaseID:             databaseID,
			DatabaseName:           exec.DatabaseName,
			Error:                  errMessage,
		}); err != nil {
			return err
		}

		resultSet := &api.SQLResultSet{
			AdviceList: []advisor.Advice{},
		}
		if queryErr == nil {
			resultSet.Data = string(bytes)
			log.Debug("Query result advice",
				zap.String("statement", exec.Statement),
			)
		} else {
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

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, resultSet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal sql result set response").SetInternal(err)
		}
		return nil
	})
}

func validateSQLSelectStatement(sqlStatement string) bool {
	// Check if the query has only one statement.
	count := 0
	if err := util.ApplyMultiStatements(strings.NewReader(sqlStatement), func(_ string) error {
		count++
		return nil
	}); err != nil {
		return false
	}
	if count != 1 {
		return false
	}

	// Allow SELECT and EXPLAIN queries only.
	whiteListRegs := []string{`^SELECT\s+?`, `^EXPLAIN\s+?`, `^WITH\s+?`}
	formattedStr := strings.ToUpper(strings.TrimSpace(sqlStatement))
	for _, reg := range whiteListRegs {
		matchResult, _ := regexp.MatchString(reg, formattedStr)
		if matchResult {
			return true
		}
	}
	return false
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
	environmentResourceType := api.PolicyResourceTypeEnvironment
	policy, err := s.store.GetNormalSQLReviewPolicy(ctx, &api.PolicyFind{ResourceType: &environmentResourceType, ResourceID: &environmentID})
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

func (s *Server) getDatabase(ctx context.Context, instanceID int, databaseName string) (*api.Database, error) {
	databaseFind := &api.DatabaseFind{
		InstanceID: &instanceID,
		Name:       &databaseName,
	}
	dbList, err := s.store.FindDatabase(ctx, databaseFind)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database `%s` for instance ID: %d", databaseName, instanceID)).SetInternal(err)
	}
	if len(dbList) == 0 {
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database `%s` for instance ID: %d not found", databaseName, instanceID))
	}
	if len(dbList) > 1 {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("There are multiple database `%s` for instance ID: %d", databaseName, instanceID))
	}
	return dbList[0], nil
}

func (s *Server) getSensitiveSchemaInfo(ctx context.Context, engineType db.Type, instanceID int, databaseList []string, currentDatabase string) (*db.SensitiveSchemaInfo, error) {
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

		if isExcludeDatabase(engineType, databaseName) {
			continue
		}

		database, err := s.getDatabase(ctx, instanceID, databaseName)
		if err != nil {
			return nil, err
		}

		columnMap := make(sensitiveDataMap)

		policy, err := s.store.GetSensitiveDataPolicy(ctx, database.ID)
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find sensitive data policy for database `%s` in instance ID: %d", databaseName, instanceID))
		}
		for _, data := range policy.SensitiveDataList {
			columnMap[api.SensitiveData{
				Table:  data.Table,
				Column: data.Column,
			}] = data.Type
		}

		dbSchema, err := s.store.GetDBSchema(ctx, database.ID)
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
				for _, column := range table.Columns {
					_, sensitive := columnMap[api.SensitiveData{
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
	case db.MySQL:
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

func (s *Server) hasDatabaseAccessRights(ctx context.Context, principalID int, role api.Role, database *api.Database) (bool, error) {
	// Workspace Owners and DBAs always have database access rights.
	if role == api.Owner || role == api.DBA {
		return true, nil
	}

	// Only project member can access database.
	projectPolicy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &database.Project.ResourceID})
	if err != nil {
		return false, err
	}
	if !hasActiveProjectMembership(principalID, projectPolicy) {
		return false, nil
	}

	// calculate the effective policy.
	databasePolicy, inheritFromEnvironment, err := s.store.GetNormalAccessControlPolicy(ctx, api.PolicyResourceTypeDatabase, database.ID)
	if err != nil {
		return false, err
	}

	environmentPolicy, _, err := s.store.GetNormalAccessControlPolicy(ctx, api.PolicyResourceTypeEnvironment, database.Instance.EnvironmentID)
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
