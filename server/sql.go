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

		db, err := db.Open(
			ctx,
			connectionInfo.Engine,
			db.DriverConfig{},
			db.ConnectionConfig{
				Username: connectionInfo.Username,
				Password: password,
				Host:     connectionInfo.Host,
				Port:     connectionInfo.Port,
				TLSConfig: db.TLSConfig{
					SslCA:   connectionInfo.SslCa,
					SslCert: connectionInfo.SslCert,
					SslKey:  connectionInfo.SslKey,
				},
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

		instance, err := s.store.GetInstanceByID(ctx, sync.InstanceID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", sync.InstanceID)).SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", sync.InstanceID))
		}

		resultSet := s.syncEngineVersionAndSchema(ctx, instance)

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, resultSet); err != nil {
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

		if s.feature(api.FeatureSchemaReviewPolicy) &&
			// For now we only support MySQL dialect schema review check.
			(instance.Engine == db.MySQL || instance.Engine == db.TiDB) {

			adviceLevel, adviceList, err = s.sqlCheck(ctx, instance, exec)
			if err != nil {
				return err
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
			errMessage = err.Error()
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

func (s *Server) syncEngineVersionAndSchema(ctx context.Context, instance *api.Instance) (rs *api.SQLResultSet) {
	resultSet := &api.SQLResultSet{}
	err := func() error {
		driver, err := tryGetReadOnlyDatabaseDriver(ctx, instance, "")
		if err != nil {
			return err
		}
		defer driver.Close(ctx)

		// Sync engine version
		version, err := driver.GetVersion(ctx)
		if err != nil {
			return err
		}
		// Underlying version may change due to upgrade, however it's a rare event, so we only update if it actually differs
		// to avoid changing the updated_ts
		if version != instance.EngineVersion {
			_, err := s.store.PatchInstance(ctx, &api.InstancePatch{
				ID:            instance.ID,
				UpdaterID:     api.SystemBotID,
				EngineVersion: &version,
			})
			if err != nil {
				return err
			}
			instance.EngineVersion = version
		}

		// Sync schema
		userList, schemaList, err := driver.SyncSchema(ctx)
		if err != nil {
			resultSet.Error = err.Error()
		} else {
			var createTable = func(database *api.Database, tableCreate *api.TableCreate) (*api.Table, error) {
				table, err := s.store.CreateTable(ctx, tableCreate)
				if err != nil {
					if common.ErrorCode(err) == common.Conflict {
						return nil, fmt.Errorf("failed to sync table for instance: %s, database: %s. Table name already exists: %s", instance.Name, database.Name, tableCreate.Name)
					}
					return nil, fmt.Errorf("failed to sync table for instance: %s, database: %s. Failed to import new table: %s. Error %w", instance.Name, database.Name, tableCreate.Name, err)
				}
				return table, nil
			}

			var createView = func(database *api.Database, viewCreate *api.ViewCreate) (*api.View, error) {
				createView, err := s.store.CreateView(ctx, viewCreate)
				if err != nil {
					if common.ErrorCode(err) == common.Conflict {
						return nil, fmt.Errorf("failed to sync view for instance: %s, database: %s. View name already exists: %s", instance.Name, database.Name, viewCreate.Name)
					}
					return nil, fmt.Errorf("failed to sync view for instance: %s, database: %s. Failed to import new view: %s. Error %w", instance.Name, database.Name, viewCreate.Name, err)
				}
				return createView, nil
			}

			var createDBExtension = func(database *api.Database, dbExtensionCreate *api.DBExtensionCreate) (*api.DBExtension, error) {
				createDBExtension, err := s.store.CreateDBExtension(ctx, dbExtensionCreate)
				if err != nil {
					if common.ErrorCode(err) == common.Conflict {
						return nil, fmt.Errorf("failed to sync dbExtension for instance: %s, database: %s. dbExtension name and schema already exists: %s", instance.Name, database.Name, dbExtensionCreate.Name)
					}
					return nil, fmt.Errorf("failed to sync view for instance: %s, database: %s. Failed to import new view: %s. Error %w", instance.Name, database.Name, dbExtensionCreate.Name, err)
				}
				return createDBExtension, nil
			}

			var createColumn = func(database *api.Database, table *api.Table, columnCreate *api.ColumnCreate) error {
				_, err := s.store.CreateColumn(ctx, columnCreate)
				if err != nil {
					if common.ErrorCode(err) == common.Conflict {
						return fmt.Errorf("failed to sync column for instance: %s, database: %s, table: %s. Column name already exists: %s", instance.Name, database.Name, table.Name, columnCreate.Name)
					}
					return fmt.Errorf("failed to sync column for instance: %s, database: %s, table: %s. Failed to import new column: %s. Error %w", instance.Name, database.Name, table.Name, columnCreate.Name, err)
				}
				return nil
			}

			var createIndex = func(database *api.Database, table *api.Table, indexCreate *api.IndexCreate) error {
				_, err := s.store.CreateIndex(ctx, indexCreate)
				if err != nil {
					if common.ErrorCode(err) == common.Conflict {
						return fmt.Errorf("failed to sync index for instance: %s, database: %s, table: %s. index and expression already exists: %s(%s)", instance.Name, database.Name, table.Name, indexCreate.Name, indexCreate.Expression)
					}
					return fmt.Errorf("failed to sync index for instance: %s, database: %s, table: %s. Failed to import new index and expression: %s(%s). Error %w", instance.Name, database.Name, table.Name, indexCreate.Name, indexCreate.Expression, err)
				}
				return nil
			}

			var recreateTableSchema = func(database *api.Database, table db.Table) error {
				// Table
				tableCreate := &api.TableCreate{
					CreatorID:     api.SystemBotID,
					CreatedTs:     table.CreatedTs,
					UpdatedTs:     table.UpdatedTs,
					DatabaseID:    database.ID,
					Name:          table.Name,
					Type:          table.Type,
					Engine:        table.Engine,
					Collation:     table.Collation,
					RowCount:      table.RowCount,
					DataSize:      table.DataSize,
					IndexSize:     table.IndexSize,
					DataFree:      table.DataFree,
					CreateOptions: table.CreateOptions,
					Comment:       table.Comment,
				}
				upsertedTable, err := createTable(database, tableCreate)
				if err != nil {
					return err
				}

				// Column
				for _, column := range table.ColumnList {
					columnFind := &api.ColumnFind{
						DatabaseID: &database.ID,
						TableID:    &upsertedTable.ID,
						Name:       &column.Name,
					}
					col, err := s.store.GetColumn(ctx, columnFind)
					if err != nil {
						return fmt.Errorf("failed to sync column for instance: %s, database: %s, table: %s. Error %w", instance.Name, database.Name, upsertedTable.Name, err)
					}
					// Create column if not exists yet.
					if col == nil {
						columnCreate := &api.ColumnCreate{
							CreatorID:    api.SystemBotID,
							DatabaseID:   database.ID,
							TableID:      upsertedTable.ID,
							Name:         column.Name,
							Position:     column.Position,
							Default:      column.Default,
							Nullable:     column.Nullable,
							Type:         column.Type,
							CharacterSet: column.CharacterSet,
							Collation:    column.Collation,
							Comment:      column.Comment,
						}
						if err := createColumn(database, upsertedTable, columnCreate); err != nil {
							return err
						}
					}
				}

				// Index
				for _, index := range table.IndexList {
					indexFind := &api.IndexFind{
						DatabaseID: &database.ID,
						TableID:    &upsertedTable.ID,
						Name:       &index.Name,
						Expression: &index.Expression,
					}
					idx, err := s.store.GetIndex(ctx, indexFind)
					if err != nil {
						return fmt.Errorf("failed to sync index for instance: %s, database: %s, table: %s. Error %w", instance.Name, database.Name, upsertedTable.Name, err)
					}
					if idx == nil {
						// Create index if not exists.
						indexCreate := &api.IndexCreate{
							CreatorID:  api.SystemBotID,
							DatabaseID: database.ID,
							TableID:    upsertedTable.ID,
							Name:       index.Name,
							Expression: index.Expression,
							Position:   index.Position,
							Type:       index.Type,
							Unique:     index.Unique,
							Visible:    index.Visible,
							Comment:    index.Comment,
						}
						if err := createIndex(database, upsertedTable, indexCreate); err != nil {
							return err
						}
					}
				}
				return nil
			}

			var recreateViewSchema = func(database *api.Database, view db.View) error {
				// View
				viewCreate := &api.ViewCreate{
					CreatorID:  api.SystemBotID,
					CreatedTs:  view.CreatedTs,
					UpdatedTs:  view.UpdatedTs,
					DatabaseID: database.ID,
					Name:       view.Name,
					Definition: view.Definition,
					Comment:    view.Comment,
				}
				_, err := createView(database, viewCreate)
				if err != nil {
					return err
				}
				return nil
			}

			var recreateDBExtensionSchema = func(database *api.Database, dbExtension db.Extension) error {
				// dbExtension
				dbExtensionCreate := &api.DBExtensionCreate{
					CreatorID:   api.SystemBotID,
					DatabaseID:  database.ID,
					Name:        dbExtension.Name,
					Version:     dbExtension.Version,
					Schema:      dbExtension.Schema,
					Description: dbExtension.Description,
				}
				_, err := createDBExtension(database, dbExtensionCreate)
				if err != nil {
					return err
				}
				return nil
			}

			instanceUserList, err := s.store.FindInstanceUserByInstanceID(ctx, instance.ID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch user list for instance: %v", instance.ID)).SetInternal(err)
			}

			// Upsert user found in the instance
			for _, user := range userList {
				userUpsert := &api.InstanceUserUpsert{
					CreatorID:  api.SystemBotID,
					InstanceID: instance.ID,
					Name:       user.Name,
					Grant:      user.Grant,
				}
				_, err := s.store.UpsertInstanceUser(ctx, userUpsert)
				if err != nil {
					return fmt.Errorf("failed to sync user for instance: %s. Failed to upsert user. Error %w", instance.Name, err)
				}
			}

			// Delete user no longer found in the instance
			for _, user := range instanceUserList {
				found := false
				for _, dbUser := range userList {
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
						return fmt.Errorf("failed to sync user for instance: %s. Failed to delete user: %s. Error %w", instance.Name, user.Name, err)
					}
				}
			}

			// Compare the stored db info with the just synced db schema.
			// Case 1: If item appears both in the stored db info and the synced db schema, then we UPDATE the corresponding record in the stored db.
			// Case 2: If item only appears in the synced schema and not in the stored db, then we CREATE the record in the stored db.
			// Case 3: Conversely, if item only appears in the stored db, but not in the synced schema, then we MARK the record as NOT_FOUND.
			//   	   We don't delete the entry because:
			//   	   1. This entry has already been associated with other entities, we can't simply delete it.
			//   	   2. The deletion in the schema might be a mistake, so it's better to surface as NOT_FOUND to let user review it.
			//
			// If we successfully synced a particular db schema, we just recreate its table, index, column info. We do this because
			// we don't reference those objects and they are for information purpose.

			databaseFind := &api.DatabaseFind{
				InstanceID: &instance.ID,
			}
			dbList, err := s.store.FindDatabase(ctx, databaseFind)
			if err != nil {
				return fmt.Errorf("failed to sync database for instance: %s. Failed to find database list. Error %w", instance.Name, err)
			}

			for _, schema := range schemaList {
				// When there are too many databases, this might have performance issue and will
				// cause frontend timeout since we set a 30s limit (INSTANCE_OPERATION_TIMEOUT).
				schemaVersion, err := getLatestSchemaVersion(ctx, driver, schema.Name)
				if err != nil {
					return err
				}

				var matchedDb *api.Database
				for _, db := range dbList {
					if db.Name == schema.Name {
						matchedDb = db
						break
					}
				}
				if matchedDb != nil {
					// Case 1, appear in both the bytebase metadata and the synced db schema
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

					tableDelete := &api.TableDelete{
						DatabaseID: dbPatched.ID,
					}
					if err := s.store.DeleteTable(ctx, tableDelete); err != nil {
						return fmt.Errorf("failed to sync database for instance: %s. Failed to reset table info for database: %s. Error %w", instance.Name, dbPatched.Name, err)
					}

					for _, table := range schema.TableList {
						err = recreateTableSchema(dbPatched, table)
						if err != nil {
							return err
						}
					}

					viewDelete := &api.ViewDelete{
						DatabaseID: dbPatched.ID,
					}
					if err := s.store.DeleteView(ctx, viewDelete); err != nil {
						return fmt.Errorf("failed to sync database for instance: %s. Failed to reset view info for database: %s. Error %w", instance.Name, dbPatched.Name, err)
					}

					for _, view := range schema.ViewList {
						err = recreateViewSchema(dbPatched, view)
						if err != nil {
							return err
						}
					}

					dbExtensionDelete := &api.DBExtensionDelete{
						DatabaseID: dbPatched.ID,
					}
					if err := s.store.DeleteDBExtension(ctx, dbExtensionDelete); err != nil {
						return fmt.Errorf("failed to sync database for instance: %s. Failed to reset dbExtension info for database: %s. Error %w", instance.Name, dbPatched.Name, err)
					}

					for _, dbExtension := range schema.ExtensionList {
						err = recreateDBExtensionSchema(dbPatched, dbExtension)
						if err != nil {
							return err
						}
					}
				} else {
					// Case 2, only appear in the synced db schema
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
					database, err := s.store.CreateDatabase(ctx, databaseCreate)
					if err != nil {
						if common.ErrorCode(err) == common.Conflict {
							return fmt.Errorf("failed to sync database for instance: %s. Database name already exists: %s", instance.Name, databaseCreate.Name)
						}
						return fmt.Errorf("failed to sync database for instance: %s. Failed to import new database: %s. Error %w", instance.Name, databaseCreate.Name, err)
					}

					for _, table := range schema.TableList {
						err = recreateTableSchema(database, table)
						if err != nil {
							return err
						}
					}

					for _, view := range schema.ViewList {
						err = recreateViewSchema(database, view)
						if err != nil {
							return err
						}
					}

					for _, dbExtension := range schema.ExtensionList {
						err = recreateDBExtensionSchema(database, dbExtension)
						if err != nil {
							return err
						}
					}
				}
			}

			// Case 3, only appear in the bytebase metadata
			for _, db := range dbList {
				found := false
				for _, schema := range schemaList {
					if db.Name == schema.Name {
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
							return fmt.Errorf("failed to sync database for instance: %s. Database not found: %s", instance.Name, database.Name)
						}
						return fmt.Errorf("failed to sync database for instance: %s. Failed to update database: %s. Error: %w", instance.Name, database.Name, err)
					}
				}
			}
		}
		return nil
	}()

	if err != nil {
		resultSet.Error = err.Error()
	}

	return resultSet
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
	formatedStr := strings.ToUpper(strings.TrimSpace(sqlStatement))
	for _, reg := range whiteListRegs {
		matchResult, _ := regexp.MatchString(reg, formatedStr)
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

func (s *Server) sqlCheck(ctx context.Context, instance *api.Instance, exec *api.SQLExecute) (advisor.Status, []advisor.Advice, error) {
	adviceLevel := advisor.Success
	var adviceList []advisor.Advice
	policy, err := s.store.GetNormalSchemaReviewPolicy(ctx, &api.PolicyFind{EnvironmentID: &instance.EnvironmentID})
	if err != nil {
		if e, ok := err.(*common.Error); ok && e.Code == common.NotFound {
			adviceLevel = advisor.Warn
			adviceList = append(adviceList, advisor.Advice{
				Status:  advisor.Warn,
				Code:    common.TaskCheckEmptySchemaReviewPolicy,
				Title:   "Empty schema review policy or disabled",
				Content: "",
			})
		} else {
			return advisor.Error, nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch schema review policy by environment ID: %d", instance.EnvironmentID)).SetInternal(err)
		}
	}
	if adviceLevel == advisor.Success {
		databaseFind := &api.DatabaseFind{
			InstanceID: &instance.ID,
			Name:       &exec.DatabaseName,
		}
		db, err := s.store.FindDatabase(ctx, databaseFind)
		if err != nil {
			return advisor.Error, nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database `%s` for instance ID: %d", exec.DatabaseName, instance.ID)).SetInternal(err)
		}
		if len(db) == 0 {
			return advisor.Error, nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database `%s` for instance ID: %d not found", exec.DatabaseName, instance.ID))
		}
		if len(db) > 1 {
			return advisor.Error, nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("There are multiple database `%s` for instance ID: %d", exec.DatabaseName, instance.ID))
		}

		res, err := advisor.SchemaReviewCheck(ctx, exec.Statement, policy, advisor.SchemaReviewCheckContext{
			Charset:   db[0].CharacterSet,
			Collation: db[0].Collation,
			DbType:    instance.Engine,
			Catalog:   store.NewCatalog(&db[0].ID, s.store),
		})
		if err != nil {
			return advisor.Error, nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to run schema review check").SetInternal(err)
		}

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
				Code:    common.Ok,
				Title:   "OK",
				Content: "",
			})
		}
	}
	return adviceLevel, adviceList, nil
}
