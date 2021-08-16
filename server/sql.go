package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/db"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerSqlRoutes(g *echo.Group) {
	g.POST("/sql/ping", func(c echo.Context) error {
		connectionInfo := &api.ConnectionInfo{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, connectionInfo); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted sql ping request").SetInternal(err)
		}

		password := connectionInfo.Password
		// Instance detail page has a Test Connection button, if user doesn't input new password, we
		// want the connection to use the existing password to test the connection, however, we do
		// not transfer the password back to client, thus the client will pass the instanceId to
		// let server retrieve the password.
		if password == "" && connectionInfo.InstanceId != nil {
			adminPassword, err := s.FindInstanceAdminPasswordById(context.Background(), *connectionInfo.InstanceId)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to retrieve admin password for instance: %d", connectionInfo.InstanceId)).SetInternal(err)
			}
			password = adminPassword
		}

		db, err := db.Open(
			connectionInfo.DBType,
			db.DriverConfig{Logger: s.l},
			db.ConnectionConfig{
				Username: connectionInfo.Username,
				Password: password,
				Host:     connectionInfo.Host,
				Port:     connectionInfo.Port,
			},
			db.ConnectionContext{},
		)

		resultSet := &api.SqlResultSet{}
		if err != nil {
			usePassword := "YES"
			if connectionInfo.Password == "" {
				usePassword = "NO"
			}
			hostPort := connectionInfo.Host
			if connectionInfo.Port != "" {
				hostPort += ":" + connectionInfo.Port
			}
			resultSet.Error = fmt.Errorf("failed to connect %q for user %q (using password: %s), %w", hostPort, connectionInfo.Username, usePassword, err).Error()
		} else {
			defer db.Close(context.Background())
			if err := db.Ping(context.Background()); err != nil {
				resultSet.Error = err.Error()
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, resultSet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal sql result set response").SetInternal(err)
		}
		return nil
	})

	g.POST("/sql/syncschema", func(c echo.Context) error {
		sync := &api.SqlSyncSchema{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, sync); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted sql sync schema request").SetInternal(err)
		}

		instance, err := s.ComposeInstanceById(context.Background(), sync.InstanceId)
		if err != nil {
			if common.ErrorCode(err) == common.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", sync.InstanceId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", sync.InstanceId)).SetInternal(err)
		}

		resultSet := s.SyncSchema(instance)

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, resultSet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal sql result set response").SetInternal(err)
		}
		return nil
	})
}

func (s *Server) SyncSchema(instance *api.Instance) (rs *api.SqlResultSet) {
	resultSet := &api.SqlResultSet{}
	err := func() error {
		driver, err := db.Open(
			db.Mysql,
			db.DriverConfig{Logger: s.l},
			db.ConnectionConfig{
				Username: instance.Username,
				Password: instance.Password,
				Host:     instance.Host,
				Port:     instance.Port,
			},
			db.ConnectionContext{
				EnvironmentName: instance.Environment.Name,
				InstanceName:    instance.Name,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to connect instance: %v with user: %v. Error %w", instance.Name, instance.Username, err)
		}

		defer driver.Close(context.Background())

		userList, schemaList, err := driver.SyncSchema(context.Background())
		if err != nil {
			resultSet.Error = err.Error()
		} else {
			var createTable = func(database *api.Database, tableCreate *api.TableCreate) (*api.Table, error) {
				createTable, err := s.TableService.CreateTable(context.Background(), tableCreate)
				if err != nil {
					if common.ErrorCode(err) == common.ECONFLICT {
						return nil, fmt.Errorf("failed to sync table for instance: %s, database: %s. Table name already exists: %s", instance.Name, database.Name, tableCreate.Name)
					}
					return nil, fmt.Errorf("failed to sync table for instance: %s, database: %s. Failed to import new table: %s. Error %w", instance.Name, database.Name, tableCreate.Name, err)
				}
				return createTable, nil
			}

			var createColumn = func(database *api.Database, table *api.Table, columnCreate *api.ColumnCreate) error {
				_, err := s.ColumnService.CreateColumn(context.Background(), columnCreate)
				if err != nil {
					if common.ErrorCode(err) == common.ECONFLICT {
						return fmt.Errorf("failed to sync column for instance: %s, database: %s, table: %s. Column name already exists: %s", instance.Name, database.Name, table.Name, columnCreate.Name)
					}
					return fmt.Errorf("failed to sync column for instance: %s, database: %s, table: %s. Failed to import new column: %s. Error %w", instance.Name, database.Name, table.Name, columnCreate.Name, err)
				}
				return nil
			}

			var createIndex = func(database *api.Database, table *api.Table, indexCreate *api.IndexCreate) error {
				_, err := s.IndexService.CreateIndex(context.Background(), indexCreate)
				if err != nil {
					if common.ErrorCode(err) == common.ECONFLICT {
						return fmt.Errorf("failed to sync index for instance: %s, database: %s, table: %s. index and expression already exists: %s(%s)", instance.Name, database.Name, table.Name, indexCreate.Name, indexCreate.Expression)
					}
					return fmt.Errorf("failed to sync index for instance: %s, database: %s, table: %s. Failed to import new index and expression: %s(%s). Error %w", instance.Name, database.Name, table.Name, indexCreate.Name, indexCreate.Expression, err)
				}
				return nil
			}

			instanceUserFind := &api.InstanceUserFind{
				InstanceId: instance.ID,
			}
			instanceUserList, err := s.InstanceUserService.FindInstanceUserList(context.Background(), instanceUserFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch user list for instance: %v", instance.ID)).SetInternal(err)
			}

			// Upsert user found in the instance
			for _, user := range userList {
				userUpsert := &api.InstanceUserUpsert{
					CreatorId:  api.SYSTEM_BOT_ID,
					InstanceId: instance.ID,
					Name:       user.Name,
					Grant:      user.Grant,
				}
				_, err := s.InstanceUserService.UpsertInstanceUser(context.Background(), userUpsert)
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
					err := s.InstanceUserService.DeleteInstanceUser(context.Background(), userDelete)
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
			databaseFind := &api.DatabaseFind{
				InstanceId: &instance.ID,
			}
			dbList, err := s.DatabaseService.FindDatabaseList(context.Background(), databaseFind)
			if err != nil {
				return fmt.Errorf("failed to sync database for instance: %s. Failed to find database list. Error %w", instance.Name, err)
			}

			for _, schema := range schemaList {
				var matchedDb *api.Database
				for _, db := range dbList {
					if db.Name == schema.Name {
						matchedDb = db
						break
					}
				}
				if matchedDb != nil {
					// Case 1
					syncStatus := api.OK
					ts := time.Now().Unix()
					databasePatch := &api.DatabasePatch{
						ID:                   matchedDb.ID,
						UpdaterId:            api.SYSTEM_BOT_ID,
						SyncStatus:           &syncStatus,
						LastSuccessfulSyncTs: &ts,
					}
					database, err := s.DatabaseService.PatchDatabase(context.Background(), databasePatch)
					if err != nil {
						if common.ErrorCode(err) == common.ENOTFOUND {
							return fmt.Errorf("failed to sync database for instance: %s. Database not found: %s", instance.Name, database.Name)
						}
						return fmt.Errorf("failed to sync database for instance: %s. Failed to update database: %s. Error %w", instance.Name, database.Name, err)
					}

					for _, table := range schema.TableList {
						// Table
						tableFind := &api.TableFind{
							DatabaseId: &database.ID,
							Name:       &table.Name,
						}
						storedTable, err := s.TableService.FindTable(context.Background(), tableFind)
						var upsertedTable *api.Table
						if err != nil {
							if common.ErrorCode(err) == common.ENOTFOUND {
								tableCreate := &api.TableCreate{
									CreatorId:     api.SYSTEM_BOT_ID,
									DatabaseId:    database.ID,
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
								upsertedTable, err = createTable(database, tableCreate)
								if err != nil {
									return err
								}
							} else {
								return fmt.Errorf("failed to sync table for instance: %s, database: %s. Error %w", instance.Name, database.Name, err)
							}
						} else {
							tablePatch := &api.TablePatch{
								ID:                   storedTable.ID,
								UpdaterId:            api.SYSTEM_BOT_ID,
								SyncStatus:           &syncStatus,
								LastSuccessfulSyncTs: &ts,
							}
							upsertedTable, err = s.TableService.PatchTable(context.Background(), tablePatch)
							if err != nil {
								if common.ErrorCode(err) == common.ENOTFOUND {
									return fmt.Errorf("failed to sync table for instance: %s, database: %s. Table not found: %s", instance.Name, database.Name, storedTable.Name)
								}
								return fmt.Errorf("failed to sync table for instance: %s, database: %s. Failed to update table: %s. Error %w", instance.Name, database.Name, storedTable.Name, err)
							}
						}

						// Column
						for _, column := range table.ColumnList {
							columnFind := &api.ColumnFind{
								DatabaseId: &database.ID,
								TableId:    &upsertedTable.ID,
								Name:       &column.Name,
							}
							storedColumn, err := s.ColumnService.FindColumn(context.Background(), columnFind)
							if err != nil {
								if common.ErrorCode(err) == common.ENOTFOUND {
									columnCreate := &api.ColumnCreate{
										CreatorId:    api.SYSTEM_BOT_ID,
										DatabaseId:   database.ID,
										TableId:      upsertedTable.ID,
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
								} else {
									return fmt.Errorf("failed to sync column for instance: %s, database: %s, table: %s. Error %w", instance.Name, database.Name, upsertedTable.Name, err)
								}
							} else {
								columnPatch := &api.ColumnPatch{
									ID:                   storedColumn.ID,
									UpdaterId:            api.SYSTEM_BOT_ID,
									SyncStatus:           &syncStatus,
									LastSuccessfulSyncTs: &ts,
								}
								_, err := s.ColumnService.PatchColumn(context.Background(), columnPatch)
								if err != nil {
									if common.ErrorCode(err) == common.ENOTFOUND {
										return fmt.Errorf("failed to sync column for instance: %s, database: %s, table: %s. Column not found: %s", instance.Name, database.Name, upsertedTable.Name, storedColumn.Name)
									}
									return fmt.Errorf("failed to sync column for instance: %s, database: %s, table: %s. Failed to update column: %s. Error %w", instance.Name, database.Name, upsertedTable.Name, storedColumn.Name, err)
								}
							}
						}

						// Index
						for _, index := range table.IndexList {
							indexFind := &api.IndexFind{
								DatabaseId: &database.ID,
								TableId:    &upsertedTable.ID,
								Name:       &index.Name,
								Expression: &index.Expression,
							}
							storedIndex, err := s.IndexService.FindIndex(context.Background(), indexFind)
							if err != nil {
								if common.ErrorCode(err) == common.ENOTFOUND {
									indexCreate := &api.IndexCreate{
										CreatorId:  api.SYSTEM_BOT_ID,
										DatabaseId: database.ID,
										TableId:    upsertedTable.ID,
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
								} else {
									return fmt.Errorf("failed to sync index for instance: %s, database: %s, table: %s. Error %w", instance.Name, database.Name, upsertedTable.Name, err)
								}
							} else {
								indexPatch := &api.IndexPatch{
									ID:                   storedIndex.ID,
									UpdaterId:            api.SYSTEM_BOT_ID,
									SyncStatus:           &syncStatus,
									LastSuccessfulSyncTs: &ts,
								}
								_, err := s.IndexService.PatchIndex(context.Background(), indexPatch)
								if err != nil {
									if common.ErrorCode(err) == common.ENOTFOUND {
										return fmt.Errorf("failed to sync index for instance: %s, database: %s, table: %s. Index not found: %s", instance.Name, database.Name, upsertedTable.Name, storedIndex.Name)
									}
									return fmt.Errorf("failed to sync index for instance: %s, database: %s, table: %s. Failed to update index: %s. Error %w", instance.Name, database.Name, upsertedTable.Name, storedIndex.Name, err)
								}
							}
						}
					}
				} else {
					// Case 2
					z, offset := time.Now().Zone()
					databaseCreate := &api.DatabaseCreate{
						CreatorId:      api.SYSTEM_BOT_ID,
						ProjectId:      api.DEFAULT_PROJECT_ID,
						InstanceId:     instance.ID,
						Name:           schema.Name,
						CharacterSet:   schema.CharacterSet,
						Collation:      schema.Collation,
						TimezoneName:   z,
						TimezoneOffset: offset,
					}
					database, err := s.DatabaseService.CreateDatabase(context.Background(), databaseCreate)
					if err != nil {
						if common.ErrorCode(err) == common.ECONFLICT {
							return fmt.Errorf("failed to sync database for instance: %s. Database name already exists: %s", instance.Name, databaseCreate.Name)
						}
						return fmt.Errorf("failed to sync database for instance: %s. Failed to import new database: %s. Error %w", instance.Name, databaseCreate.Name, err)
					}

					for _, table := range schema.TableList {
						// Table
						tableCreate := &api.TableCreate{
							CreatorId:     api.SYSTEM_BOT_ID,
							DatabaseId:    database.ID,
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
								DatabaseId: &database.ID,
								TableId:    &upsertedTable.ID,
								Name:       &column.Name,
							}
							_, err := s.ColumnService.FindColumn(context.Background(), columnFind)
							if err != nil {
								if common.ErrorCode(err) == common.ENOTFOUND {
									columnCreate := &api.ColumnCreate{
										CreatorId:    api.SYSTEM_BOT_ID,
										DatabaseId:   database.ID,
										TableId:      upsertedTable.ID,
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
								} else {
									return fmt.Errorf("failed to sync column for instance: %s, database: %s, table: %s. Error %w", instance.Name, database.Name, upsertedTable.Name, err)
								}
							}
						}

						// Index
						for _, index := range table.IndexList {
							indexFind := &api.IndexFind{
								DatabaseId: &database.ID,
								TableId:    &upsertedTable.ID,
								Name:       &index.Name,
								Expression: &index.Expression,
							}
							_, err := s.IndexService.FindIndex(context.Background(), indexFind)
							if err != nil {
								if common.ErrorCode(err) == common.ENOTFOUND {
									indexCreate := &api.IndexCreate{
										CreatorId:  api.SYSTEM_BOT_ID,
										DatabaseId: database.ID,
										TableId:    upsertedTable.ID,
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
								} else {
									return fmt.Errorf("failed to sync index for instance: %s, database: %s, table: %s. Error %w", instance.Name, database.Name, upsertedTable.Name, err)
								}
							}
						}
					}
				}
			}

			// Case 3
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
						UpdaterId:            api.SYSTEM_BOT_ID,
						SyncStatus:           &syncStatus,
						LastSuccessfulSyncTs: &ts,
					}
					database, err := s.DatabaseService.PatchDatabase(context.Background(), databasePatch)
					if err != nil {
						if common.ErrorCode(err) == common.ENOTFOUND {
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
