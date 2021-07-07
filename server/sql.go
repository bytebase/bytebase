package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
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

		db, err := db.Open(connectionInfo.DBType, db.DriverConfig{Logger: s.l}, db.ConnectionConfig{
			Username: connectionInfo.Username,
			Password: connectionInfo.Password,
			Host:     connectionInfo.Host,
			Port:     connectionInfo.Port,
		})

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
			resultSet.Error = fmt.Errorf("failed to connect '%s' for user '%s' (using password: %s), %w", hostPort, connectionInfo.Username, usePassword, err).Error()
		} else {
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
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", sync.InstanceId))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", sync.InstanceId)).SetInternal(err)
		}

		driver, err := db.Open(db.Mysql, db.DriverConfig{Logger: s.l}, db.ConnectionConfig{
			Username: instance.Username,
			Password: instance.Password,
			Host:     instance.Host,
			Port:     instance.Port,
		})
		if err != nil {
			usePassword := "YES"
			if instance.Password == "" {
				usePassword = "NO"
			}
			hostPort := instance.Host
			if instance.Port != "" {
				hostPort += ":" + instance.Port
			}
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to connect '%s' for user '%s' (using password: %s)", hostPort, instance.Username, usePassword)).SetInternal(err)
		}

		resultSet := &api.SqlResultSet{}
		schemaList, err := driver.SyncSchema(context.Background())
		if err != nil {
			resultSet.Error = err.Error()
		} else {
			var createTable = func(database *api.Database, tableCreate *api.TableCreate) error {
				_, err := s.TableService.CreateTable(context.Background(), tableCreate)
				if err != nil {
					if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
						return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Failed to sync table for instance: %s, database: %s. Table name already exists: %s", instance.Name, database.Name, tableCreate.Name))
					}
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to sync database for instance: %s, database: %s. Failed to import new table: %s", instance.Name, database.Name, tableCreate.Name)).SetInternal(err)
				}
				return nil
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
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to sync database for instance: %s. Failed to find database list", instance.Name)).SetInternal(err)
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
						if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
							return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Failed to sync database for instance: %s. Database not found: %s", instance.Name, database.Name))
						}
						return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to sync database for instance: %s. Failed to update database: %s", instance.Name, database.Name)).SetInternal(err)
					}

					for _, table := range schema.TableList {
						tableFind := &api.TableFind{
							DatabaseId: &database.ID,
							Name:       &table.Name,
						}
						storedTable, err := s.TableService.FindTable(context.Background(), tableFind)
						if err != nil {
							if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
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
								if err := createTable(database, tableCreate); err != nil {
									return err
								}
							} else {
								return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to sync table for instance: %s, database: %s", instance.Name, database.Name)).SetInternal(err)
							}
						} else {
							tablePatch := &api.TablePatch{
								ID:                   storedTable.ID,
								UpdaterId:            api.SYSTEM_BOT_ID,
								SyncStatus:           &syncStatus,
								LastSuccessfulSyncTs: &ts,
							}
							_, err := s.TableService.PatchTable(context.Background(), tablePatch)
							if err != nil {
								if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
									return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Failed to sync table for instance: %s, database: %s. Table not found: %s", instance.Name, database.Name, storedTable.Name))
								}
								return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to sync table for instance: %s, database: %s. Failed to update table: %s", instance.Name, database.Name, storedTable.Name)).SetInternal(err)
							}
						}
					}
				} else {
					// Case 2
					databaseCreate := &api.DatabaseCreate{
						CreatorId:    api.SYSTEM_BOT_ID,
						ProjectId:    api.DEFAULT_PROJECT_ID,
						InstanceId:   instance.ID,
						Name:         schema.Name,
						CharacterSet: schema.CharacterSet,
						Collation:    schema.Collation,
					}
					database, err := s.DatabaseService.CreateDatabase(context.Background(), databaseCreate)
					if err != nil {
						if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
							return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Failed to sync database for instance: %s. Database name already exists: %s", instance.Name, databaseCreate.Name))
						}
						return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to sync database for instance: %s. Failed to import new database: %s", instance.Name, databaseCreate.Name)).SetInternal(err)
					}

					for _, table := range schema.TableList {
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
						if err := createTable(database, tableCreate); err != nil {
							return err
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
						if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
							return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Failed to sync database for instance: %s. Database not found: %s", instance.Name, database.Name))
						}
						return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to sync database for instance: %s. Failed to update database: %s", instance.Name, database.Name)).SetInternal(err)
					}
				}
			}
		}

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
		driver, err := db.Open(db.Mysql, db.DriverConfig{Logger: s.l}, db.ConnectionConfig{
			Username: instance.Username,
			Password: instance.Password,
			Host:     instance.Host,
			Port:     instance.Port,
		})
		if err != nil {
			return fmt.Errorf("failed to connect instance: %v with user: %v. Error %w", instance.Name, instance.Username, err)
		}

		schemaList, err := driver.SyncSchema(context.Background())
		if err != nil {
			resultSet.Error = err.Error()
		} else {
			var createTable = func(database *api.Database, tableCreate *api.TableCreate) error {
				_, err := s.TableService.CreateTable(context.Background(), tableCreate)
				if err != nil {
					if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
						return fmt.Errorf("failed to sync table for instance: %s, database: %s. Table name already exists: %s", instance.Name, database.Name, tableCreate.Name)
					}
					return fmt.Errorf("failed to sync database for instance: %s, database: %s. Failed to import new table: %s. Error %w", instance.Name, database.Name, tableCreate.Name, err)
				}
				return nil
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
						if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
							return fmt.Errorf("failed to sync database for instance: %s. Database not found: %s", instance.Name, database.Name)
						}
						return fmt.Errorf("failed to sync database for instance: %s. Failed to update database: %s. Error %w", instance.Name, database.Name, err)
					}

					for _, table := range schema.TableList {
						tableFind := &api.TableFind{
							DatabaseId: &database.ID,
							Name:       &table.Name,
						}
						storedTable, err := s.TableService.FindTable(context.Background(), tableFind)
						if err != nil {
							if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
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
								if err := createTable(database, tableCreate); err != nil {
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
							_, err := s.TableService.PatchTable(context.Background(), tablePatch)
							if err != nil {
								if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
									return fmt.Errorf("failed to sync table for instance: %s, database: %s. Table not found: %s", instance.Name, database.Name, storedTable.Name)
								}
								return fmt.Errorf("failed to sync table for instance: %s, database: %s. Failed to update table: %s. Error %w", instance.Name, database.Name, storedTable.Name, err)
							}
						}
					}
				} else {
					// Case 2
					databaseCreate := &api.DatabaseCreate{
						CreatorId:    api.SYSTEM_BOT_ID,
						ProjectId:    api.DEFAULT_PROJECT_ID,
						InstanceId:   instance.ID,
						Name:         schema.Name,
						CharacterSet: schema.CharacterSet,
						Collation:    schema.Collation,
					}
					database, err := s.DatabaseService.CreateDatabase(context.Background(), databaseCreate)
					if err != nil {
						if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
							return fmt.Errorf("failed to sync database for instance: %s. Database name already exists: %s", instance.Name, databaseCreate.Name)
						}
						return fmt.Errorf("failed to sync database for instance: %s. Failed to import new database: %s. Error %w", instance.Name, databaseCreate.Name, err)
					}

					for _, table := range schema.TableList {
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
						if err := createTable(database, tableCreate); err != nil {
							return err
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
						if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
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
