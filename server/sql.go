package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/advisor"
	"github.com/bytebase/bytebase/plugin/advisor/catalog"
	advisorDB "github.com/bytebase/bytebase/plugin/advisor/db"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
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
		if connectionInfo.Engine == db.ClickHouse {
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
				Username:  connectionInfo.Username,
				Password:  password,
				Host:      connectionInfo.Host,
				Port:      connectionInfo.Port,
				TLSConfig: tlsConfig,
			},
			db.ConnectionContext{},
		)

		resultSet := &api.SQLResultSet{}
		if err != nil {
			hostPort := connectionInfo.Host
			if connectionInfo.Port != "" {
				hostPort += ":" + connectionInfo.Port
			}
			resultSet.Error = fmt.Errorf("failed to connect %q for user %q, %w", hostPort, connectionInfo.Username, err).Error()
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
			if err := s.syncEngineVersionAndSchema(ctx, instance); err != nil {
				resultSet.Error = err.Error()
			}
		}
		if sync.DatabaseID != nil {
			database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: sync.DatabaseID})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to database instance ID: %d", *sync.DatabaseID)).SetInternal(err)
			}
			if database == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", *sync.DatabaseID))
			}
			if err := s.syncDatabaseSchema(ctx, database.Instance, database.Name); err != nil {
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

		adviceLevel := advisor.Success
		adviceList := []advisor.Advice{}

		if s.feature(api.FeatureSQLReviewPolicy) && api.IsSQLReviewSupported(instance.Engine, s.profile.Mode) {
			dbType, err := advisorDB.ConvertToAdvisorDBType(string(instance.Engine))
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to convert db type %v into advisor db type", instance.Engine))
			}

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
			db := dbList[0]

			adviceLevel, adviceList, err = s.sqlCheck(
				ctx,
				dbType,
				db.CharacterSet,
				db.Collation,
				instance.EnvironmentID,
				exec.Statement,
				store.NewCatalog(&db.ID, s.store, instance.Engine),
			)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to check SQL review policy").SetInternal(err)
			}

			if adviceLevel == advisor.Error {
				if err := s.createSQLEditorQueryActivity(ctx, c, api.ActivityError, exec.InstanceID, api.ActivitySQLEditorQueryPayload{
					Statement:    exec.Statement,
					DurationNs:   0,
					InstanceName: instance.Name,
					DatabaseName: exec.DatabaseName,
					Error:        "",
					AdviceList:   adviceList,
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

		start := time.Now().UnixNano()

		bytes, queryErr := func() ([]byte, error) {
			driver, err := tryGetReadOnlyDatabaseDriver(ctx, instance, exec.DatabaseName)
			if err != nil {
				return nil, err
			}
			defer driver.Close(ctx)

			rowSet, err := driver.Query(ctx, exec.Statement, exec.Limit)
			if err != nil {
				return nil, err
			}

			return json.Marshal(rowSet)
		}()

		level := api.ActivityInfo
		errMessage := ""
		if adviceLevel == advisor.Warn {
			level = api.ActivityWarn
		}
		if queryErr != nil {
			level = api.ActivityError
			errMessage = queryErr.Error()
		}
		if err := s.createSQLEditorQueryActivity(ctx, c, level, exec.InstanceID, api.ActivitySQLEditorQueryPayload{
			Statement:    exec.Statement,
			DurationNs:   time.Now().UnixNano() - start,
			InstanceName: instance.Name,
			DatabaseName: exec.DatabaseName,
			Error:        errMessage,
			AdviceList:   adviceList,
		}); err != nil {
			return err
		}

		resultSet := &api.SQLResultSet{AdviceList: adviceList}
		if queryErr == nil {
			resultSet.Data = string(bytes)
			log.Debug("Query result",
				zap.String("statement", exec.Statement),
				zap.String("data", resultSet.Data),
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
}

func (s *Server) syncEngineVersionAndSchema(ctx context.Context, instance *api.Instance) error {
	driver, err := tryGetReadOnlyDatabaseDriver(ctx, instance, "")
	if err != nil {
		return err
	}
	defer driver.Close(ctx)

	databaseList, err := s.syncInstanceSchema(ctx, instance, driver)
	if err != nil {
		return err
	}

	var errorList []string
	for _, databaseName := range databaseList {
		// If we fail to sync a particular database due to permission issue, we will continue to sync the rest of the databases.
		if err := s.syncDatabaseSchema(ctx, instance, databaseName); err != nil {
			errorList = append(errorList, err.Error())
		}
	}
	if len(errorList) > 0 {
		return fmt.Errorf("sync database schema errors, %s", strings.Join(errorList, ", "))
	}

	return nil
}

// syncInstanceSchema syncs the instance and all database metadata first without diving into the deep structure of each database.
func (s *Server) syncInstanceSchema(ctx context.Context, instance *api.Instance, driver db.Driver) ([]string, error) {
	// Sync instance metadata.
	instanceMeta, err := driver.SyncInstance(ctx)
	if err != nil {
		return nil, err
	}

	// Underlying version may change due to upgrade, however it's a rare event, so we only update if it actually differs
	// to avoid changing the updated_ts.
	if instanceMeta.Version != instance.EngineVersion {
		_, err := s.store.PatchInstance(ctx, &api.InstancePatch{
			ID:            instance.ID,
			UpdaterID:     api.SystemBotID,
			EngineVersion: &instanceMeta.Version,
		})
		if err != nil {
			return nil, err
		}
		instance.EngineVersion = instanceMeta.Version
	}

	instanceUserList, err := s.store.FindInstanceUserByInstanceID(ctx, instance.ID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch user list for instance: %v", instance.ID)).SetInternal(err)
	}

	// Upsert user found in the instance
	for _, user := range instanceMeta.UserList {
		userUpsert := &api.InstanceUserUpsert{
			CreatorID:  api.SystemBotID,
			InstanceID: instance.ID,
			Name:       user.Name,
			Grant:      user.Grant,
		}
		_, err := s.store.UpsertInstanceUser(ctx, userUpsert)
		if err != nil {
			return nil, fmt.Errorf("failed to sync user for instance: %s. Failed to upsert user. Error %w", instance.Name, err)
		}
	}

	// Delete user no longer found in the instance
	for _, user := range instanceUserList {
		found := false
		for _, dbUser := range instanceMeta.UserList {
			if user.Name == dbUser.Name {
				found = true
				break
			}
		}

		if !found {
			userDelete := &api.InstanceUserDelete{
				ID: user.ID,
			}
			err := s.store.DeleteInstanceUser(ctx, userDelete)
			if err != nil {
				return nil, fmt.Errorf("failed to sync user for instance: %s. Failed to delete user: %s. Error %w", instance.Name, user.Name, err)
			}
		}
	}

	// Compare the stored db info with the just synced db schema.
	// Case 1: If item appears in both stored db info and the synced db metadata, then it's a no-op. We rely on syncDatabaseSchema() later to sync its details.
	// Case 2: If item only appears in the synced schema and not in the stored db, then we CREATE the database record in the stored db.
	// Case 3: Conversely, if item only appears in the stored db, but not in the synced schema, then we MARK the record as NOT_FOUND.
	//   	   We don't delete the entry because:
	//   	   1. This entry has already been associated with other entities, we can't simply delete it.
	//   	   2. The deletion in the schema might be a mistake, so it's better to surface as NOT_FOUND to let user review it.
	databaseFind := &api.DatabaseFind{
		InstanceID: &instance.ID,
	}
	dbList, err := s.store.FindDatabase(ctx, databaseFind)
	if err != nil {
		return nil, fmt.Errorf("failed to sync database for instance: %s. Failed to find database list. Error %w", instance.Name, err)
	}
	for _, databaseMetadata := range instanceMeta.DatabaseList {
		databaseName := databaseMetadata.Name

		var matchedDb *api.Database
		for _, db := range dbList {
			if db.Name == databaseName {
				matchedDb = db
				break
			}
		}
		if matchedDb != nil {
			// Case 1, appear in both the Bytebase metadata and the synced database metadata.
			// We rely on syncDatabaseSchema() to sync the database details.
		} else {
			// Case 2, only appear in the synced db schema.
			databaseCreate := &api.DatabaseCreate{
				CreatorID:     api.SystemBotID,
				ProjectID:     api.DefaultProjectID,
				InstanceID:    instance.ID,
				EnvironmentID: instance.EnvironmentID,
				Name:          databaseName,
				CharacterSet:  databaseMetadata.CharacterSet,
				Collation:     databaseMetadata.Collation,
			}
			if _, err := s.store.CreateDatabase(ctx, databaseCreate); err != nil {
				if common.ErrorCode(err) == common.Conflict {
					return nil, fmt.Errorf("failed to sync database for instance: %s. Database name already exists: %s", instance.Name, databaseCreate.Name)
				}
				return nil, fmt.Errorf("failed to sync database for instance: %s. Failed to import new database: %s. Error %w", instance.Name, databaseCreate.Name, err)
			}
		}
	}

	// Case 3, only appear in the Bytebase metadata
	for _, db := range dbList {
		found := false
		for _, databaseMetadata := range instanceMeta.DatabaseList {
			if db.Name == databaseMetadata.Name {
				found = true
				break
			}
		}
		if !found {
			syncStatus := api.NotFound
			ts := time.Now().Unix()
			databasePatch := &api.DatabasePatch{
				ID:                   db.ID,
				UpdaterID:            api.SystemBotID,
				SyncStatus:           &syncStatus,
				LastSuccessfulSyncTs: &ts,
				// SchemaVersion will not be over-written.
			}
			database, err := s.store.PatchDatabase(ctx, databasePatch)
			if err != nil {
				if common.ErrorCode(err) == common.NotFound {
					return nil, fmt.Errorf("failed to sync database for instance: %s. Database not found: %s", instance.Name, database.Name)
				}
				return nil, fmt.Errorf("failed to sync database for instance: %s. Failed to update database: %s. Error: %w", instance.Name, database.Name, err)
			}
		}
	}

	var databaseList []string
	for _, database := range instanceMeta.DatabaseList {
		databaseList = append(databaseList, database.Name)
	}

	return databaseList, nil
}

func (s *Server) syncDatabaseSchema(ctx context.Context, instance *api.Instance, databaseName string) error {
	driver, err := tryGetReadOnlyDatabaseDriver(ctx, instance, "")
	if err != nil {
		return err
	}
	defer driver.Close(ctx)

	databaseFind := &api.DatabaseFind{
		InstanceID: &instance.ID,
		Name:       &databaseName,
	}
	matchedDb, err := s.store.GetDatabase(ctx, databaseFind)
	if err != nil {
		return fmt.Errorf("failed to sync database for instance: %s. Failed to find database list. Error %w", instance.Name, err)
	}

	// Sync database schema
	schema, err := driver.SyncDBSchema(ctx, databaseName)
	if err != nil {
		return err
	}

	// When there are too many databases, this might have performance issue and will
	// cause frontend timeout since we set a 30s limit (INSTANCE_OPERATION_TIMEOUT).
	schemaVersion, err := getLatestSchemaVersion(ctx, driver, schema.Name)
	if err != nil {
		return err
	}

	var database *api.Database
	if matchedDb != nil {
		syncStatus := api.OK
		ts := time.Now().Unix()
		databasePatch := &api.DatabasePatch{
			ID:                   matchedDb.ID,
			UpdaterID:            api.SystemBotID,
			SyncStatus:           &syncStatus,
			LastSuccessfulSyncTs: &ts,
			SchemaVersion:        &schemaVersion,
		}
		dbPatched, err := s.store.PatchDatabase(ctx, databasePatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return fmt.Errorf("failed to sync database for instance: %s. Database not found: %v", instance.Name, matchedDb.Name)
			}
			return fmt.Errorf("failed to sync database for instance: %s. Failed to update database: %s. Error %w", instance.Name, matchedDb.Name, err)
		}
		database = dbPatched
	} else {
		databaseCreate := &api.DatabaseCreate{
			CreatorID:     api.SystemBotID,
			ProjectID:     api.DefaultProjectID,
			InstanceID:    instance.ID,
			EnvironmentID: instance.EnvironmentID,
			Name:          schema.Name,
			CharacterSet:  schema.CharacterSet,
			Collation:     schema.Collation,
			SchemaVersion: schemaVersion,
		}
		createdDatabase, err := s.store.CreateDatabase(ctx, databaseCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return fmt.Errorf("failed to sync database for instance: %s. Database name already exists: %s", instance.Name, databaseCreate.Name)
			}
			return fmt.Errorf("failed to sync database for instance: %s. Failed to import new database: %s. Error %w", instance.Name, databaseCreate.Name, err)
		}
		database = createdDatabase
	}
	if err := syncTableSchema(ctx, s.store, database, schema); err != nil {
		return err
	}
	if err := syncViewSchema(ctx, s.store, database, schema); err != nil {
		return err
	}
	if err := syncDBExtensionSchema(ctx, s.store, database, schema); err != nil {
		return err
	}
	return nil
}

func syncTableSchema(ctx context.Context, store *store.Store, database *api.Database, schema *db.Schema) error {
	return store.SetTableList(ctx, schema, database.ID)
}

func syncViewSchema(ctx context.Context, store *store.Store, database *api.Database, schema *db.Schema) error {
	return store.SetViewList(ctx, schema, database.ID)
}

func syncDBExtensionSchema(ctx context.Context, store *store.Store, database *api.Database, schema *db.Schema) error {
	return store.SetDBExtensionList(ctx, schema, database.ID)
}

func getLatestSchemaVersion(ctx context.Context, driver db.Driver, databaseName string) (string, error) {
	// TODO(d): support semantic versioning.
	limit := 1
	history, err := driver.FindMigrationHistoryList(ctx, &db.MigrationHistoryFind{
		Database: &databaseName,
		Limit:    &limit,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get migration history for database %q, error %v", databaseName, err)
	}
	var schemaVersion string
	if len(history) == 1 {
		schemaVersion = history[0].Version
	}
	return schemaVersion, nil
}

func validateSQLSelectStatement(sqlStatement string) bool {
	// Check if the query has only one statement.
	count := 0
	sc := bufio.NewScanner(strings.NewReader(sqlStatement))
	if err := util.ApplyMultiStatements(sc, func(_ string) error {
		count++
		return nil
	}); err != nil {
		return false
	}
	if count != 1 {
		return false
	}

	// Allow SELECT and EXPLAIN queries only.
	whiteListRegs := []string{`^SELECT\s+?`, `^EXPLAIN\s+?`}
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
			zap.String("instance_name", payload.InstanceName),
			zap.String("statement", payload.Statement),
			zap.Error(err))
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
	}

	activityCreate := &api.ActivityCreate{
		CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
		Type:        api.ActivitySQLEditorQuery,
		ContainerID: containerID,
		Level:       level,
		Comment: fmt.Sprintf("Executed `%q` in database %q of instance %q.",
			payload.Statement, payload.DatabaseName, payload.InstanceName),
		Payload: string(activityBytes),
	}

	if _, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{}); err != nil {
		log.Warn("Failed to create activity after executing sql statement",
			zap.String("database_name", payload.DatabaseName),
			zap.String("instance_name", payload.InstanceName),
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
) (advisor.Status, []advisor.Advice, error) {
	var adviceList []advisor.Advice
	policy, err := s.store.GetNormalSQLReviewPolicy(ctx, &api.PolicyFind{EnvironmentID: &environmentID})
	if err != nil {
		if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
			adviceList = []advisor.Advice{
				{
					Status:  advisor.Warn,
					Code:    advisor.NotFound,
					Title:   "SQL review policy is not configured or disabled",
					Content: "",
				},
			}
			return advisor.Warn, adviceList, nil
		}
		return advisor.Error, nil, err
	}

	res, err := advisor.SQLReviewCheck(statement, policy.RuleList, advisor.SQLReviewCheckContext{
		Charset:   dbCharacterSet,
		Collation: dbCollation,
		DbType:    dbType,
		Catalog:   catalog,
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

	if len(adviceList) == 0 {
		adviceList = append(adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}
	return adviceLevel, adviceList, nil
}
