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

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/db/util"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func (s *Server) registerSQLRoutes(g *echo.Group) {
	g.POST("/sql/ping", func(c echo.Context) error {
		ctx := context.Background()
		connectionInfo := &api.ConnectionInfo{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, connectionInfo); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted sql ping request").SetInternal(err)
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
			adminPassword, err := s.findInstanceAdminPasswordByID(ctx, *connectionInfo.InstanceID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve admin password for instance: %d", connectionInfo.InstanceID)).SetInternal(err)
			}
			password = adminPassword
		}

		db, err := db.Open(
			ctx,
			connectionInfo.Engine,
			db.DriverConfig{Logger: s.l},
			db.ConnectionConfig{
				Username: connectionInfo.Username,
				Password: password,
				Host:     connectionInfo.Host,
				Port:     connectionInfo.Port,
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
		ctx := context.Background()
		sync := &api.SQLSyncSchema{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, sync); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted sql sync schema request").SetInternal(err)
		}

		instance, err := s.store.GetInstanceByID(ctx, sync.InstanceID)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", sync.InstanceID))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", sync.InstanceID)).SetInternal(err)
		}

		resultSet := s.syncEngineVersionAndSchema(ctx, instance)

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, resultSet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal sql result set response").SetInternal(err)
		}
		return nil
	})

	g.POST("/sql/execute", func(c echo.Context) error {
		ctx := context.Background()
		exec := &api.SQLExecute{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, exec); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted sql execute request").SetInternal(err)
		}

		if exec.InstanceID == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted sql execute request, missing instanceId")
		}
		if len(exec.Statement) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted sql execute request, missing sql statement")
		}
		if !exec.Readonly {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted sql execute request, only support readonly sql statement")
		}
		if !validateSQLSelectStatement(exec.Statement) {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted sql execute request, only support SELECT sql statement")
		}

		instance, err := s.store.GetInstanceByID(ctx, exec.InstanceID)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", exec.InstanceID))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", exec.InstanceID)).SetInternal(err)
		}

		start := time.Now().UnixNano()

		bytes, err := func() ([]byte, error) {
			driver, err := tryGetReadOnlyDatabaseDriver(ctx, instance, exec.DatabaseName, s.l)
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

		{
			errMessage := ""
			activityLevel := api.ActivityInfo
			if err != nil {
				errMessage = err.Error()
				activityLevel = api.ActivityError
			}

			activityBytes, err := json.Marshal(api.ActivitySQLEditorQueryPayload{
				Statement:    exec.Statement,
				DurationNs:   time.Now().UnixNano() - start,
				InstanceName: instance.Name,
				DatabaseName: exec.DatabaseName,
				Error:        errMessage,
			})

			if err != nil {
				s.l.Warn("Failed to marshal activity after executing sql statement",
					zap.String("database_name", exec.DatabaseName),
					zap.String("instance_name", instance.Name),
					zap.String("statement", exec.Statement),
					zap.Error(err))
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
			}

			activityCreate := &api.ActivityCreate{
				CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
				Type:        api.ActivitySQLEditorQuery,
				ContainerID: exec.InstanceID,
				Level:       activityLevel,
				Comment: fmt.Sprintf("Executed `%q` in database %q of instance %q.",
					exec.Statement, exec.DatabaseName, instance.Name),
				Payload: string(activityBytes),
			}

			_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})

			if err != nil {
				s.l.Warn("Failed to create activity after executing sql statement",
					zap.String("database_name", exec.DatabaseName),
					zap.String("instance_name", instance.Name),
					zap.String("statement", exec.Statement),
					zap.Error(err))
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create activity").SetInternal(err)
			}
		}

		resultSet := &api.SQLResultSet{}
		if err == nil {
			resultSet.Data = string(bytes)
			s.l.Debug("Query result",
				zap.String("statement", exec.Statement),
				zap.String("data", resultSet.Data),
			)
		} else {
			resultSet.Error = err.Error()
			if s.mode == common.ReleaseModeDev {
				s.l.Error("Failed to execute query",
					zap.Error(err),
					zap.String("statement", exec.Statement),
				)
			} else {
				s.l.Debug("Failed to execute query",
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

func (s *Server) syncEngineVersionAndSchema(ctx context.Context, instance *api.Instance) (rs *api.SQLResultSet) {
	resultSet := &api.SQLResultSet{}
	err := func() error {
		driver, err := tryGetReadOnlyDatabaseDriver(ctx, instance, "", s.l)
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
			fmt.Printf("sync schema error: %v\n", err)
			resultSet.Error = err.Error()
		} else {
			var createTable = func(database *api.Database, tableCreate *api.TableCreate) (*api.Table, error) {
				createTableRaw, err := s.TableService.CreateTable(ctx, tableCreate)
				if err != nil {
					if common.ErrorCode(err) == common.Conflict {
						return nil, fmt.Errorf("failed to sync table for instance: %s, database: %s. Table name already exists: %s", instance.Name, database.Name, tableCreate.Name)
					}
					return nil, fmt.Errorf("failed to sync table for instance: %s, database: %s. Failed to import new table: %s. Error %w", instance.Name, database.Name, tableCreate.Name, err)
				}
				createTable, err := s.composeTableRelationship(ctx, createTableRaw)
				if err != nil {
					return nil, fmt.Errorf("failed to compose table with ID %d, error: %v", createTable.ID, err)
				}
				return createTable, nil
			}

			var createView = func(database *api.Database, viewCreate *api.ViewCreate) (*api.View, error) {
				createViewRaw, err := s.ViewService.CreateView(ctx, viewCreate)
				if err != nil {
					if common.ErrorCode(err) == common.Conflict {
						return nil, fmt.Errorf("failed to sync view for instance: %s, database: %s. View name already exists: %s", instance.Name, database.Name, viewCreate.Name)
					}
					return nil, fmt.Errorf("failed to sync view for instance: %s, database: %s. Failed to import new view: %s. Error %w", instance.Name, database.Name, viewCreate.Name, err)
				}
				createView, err := s.composeViewRelationship(ctx, createViewRaw)
				if err != nil {
					return nil, fmt.Errorf("failed to compose view relationship for instance: %s, database: %s. Failed to import new view: %s. Error %w", instance.Name, database.Name, viewCreate.Name, err)
				}
				return createView, nil
			}

			var createColumn = func(database *api.Database, table *api.Table, columnCreate *api.ColumnCreate) error {
				_, err := s.ColumnService.CreateColumn(ctx, columnCreate)
				if err != nil {
					if common.ErrorCode(err) == common.Conflict {
						return fmt.Errorf("failed to sync column for instance: %s, database: %s, table: %s. Column name already exists: %s", instance.Name, database.Name, table.Name, columnCreate.Name)
					}
					return fmt.Errorf("failed to sync column for instance: %s, database: %s, table: %s. Failed to import new column: %s. Error %w", instance.Name, database.Name, table.Name, columnCreate.Name, err)
				}
				return nil
			}

			var createIndex = func(database *api.Database, table *api.Table, indexCreate *api.IndexCreate) error {
				_, err := s.IndexService.CreateIndex(ctx, indexCreate)
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
					col, err := s.ColumnService.FindColumn(ctx, columnFind)
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
					idx, err := s.IndexService.FindIndex(ctx, indexFind)
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

			instanceUserFind := &api.InstanceUserFind{
				InstanceID: instance.ID,
			}
			instanceUserList, err := s.InstanceUserService.FindInstanceUserList(ctx, instanceUserFind)
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
				_, err := s.InstanceUserService.UpsertInstanceUser(ctx, userUpsert)
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
					err := s.InstanceUserService.DeleteInstanceUser(ctx, userDelete)
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
			dbRawList, err := s.DatabaseService.FindDatabaseList(ctx, databaseFind)
			if err != nil {
				return fmt.Errorf("Failed to sync database for instance: %s. Failed to find database list. Error %w", instance.Name, err)
			}
			var dbList []*api.Database
			for _, dbRaw := range dbRawList {
				db, err := s.composeDatabaseRelationship(ctx, dbRaw)
				if err != nil {
					return fmt.Errorf("Failed to compose database relationship with ID %v, error: %v", dbRaw.ID, err)
				}
				dbList = append(dbList, db)
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
					dbRawPatched, err := s.DatabaseService.PatchDatabase(ctx, databasePatch)
					if err != nil {
						if common.ErrorCode(err) == common.NotFound {
							return fmt.Errorf("failed to sync database for instance: %s. Database not found: %v", instance.Name, matchedDb.Name)
						}
						return fmt.Errorf("failed to sync database for instance: %s. Failed to update database: %s. Error %w", instance.Name, matchedDb.Name, err)
					}
					dbPatched, err := s.composeDatabaseRelationship(ctx, dbRawPatched)
					if err != nil {
						return fmt.Errorf("Failed to compose database relationship with ID %v, error: %v", dbRawPatched.ID, err)
					}

					tableDelete := &api.TableDelete{
						DatabaseID: dbPatched.ID,
					}
					if err := s.TableService.DeleteTable(ctx, tableDelete); err != nil {
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
					if err := s.ViewService.DeleteView(ctx, viewDelete); err != nil {
						return fmt.Errorf("failed to sync database for instance: %s. Failed to reset view info for database: %s. Error %w", instance.Name, dbPatched.Name, err)
					}

					for _, view := range schema.ViewList {
						err = recreateViewSchema(dbPatched, view)
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
					dbRaw, err := s.DatabaseService.CreateDatabase(ctx, databaseCreate)
					if err != nil {
						if common.ErrorCode(err) == common.Conflict {
							return fmt.Errorf("failed to sync database for instance: %s. Database name already exists: %s", instance.Name, databaseCreate.Name)
						}
						return fmt.Errorf("failed to sync database for instance: %s. Failed to import new database: %s. Error %w", instance.Name, databaseCreate.Name, err)
					}
					db, err := s.composeDatabaseRelationship(ctx, dbRaw)
					if err != nil {
						return fmt.Errorf("Failed to compose database relationship with ID %v, error: %v", dbRaw.ID, err)
					}

					for _, table := range schema.TableList {
						err = recreateTableSchema(db, table)
						if err != nil {
							return err
						}
					}

					for _, view := range schema.ViewList {
						err = recreateViewSchema(db, view)
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
					database, err := s.DatabaseService.PatchDatabase(ctx, databasePatch)
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
	if err := util.ApplyMultiStatements(sc, func(stmt string) error {
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
