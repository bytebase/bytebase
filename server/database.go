package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/edit"
)

func (s *Server) registerDatabaseRoutes(g *echo.Group) {
	g.GET("/database", func(c echo.Context) error {
		ctx := c.Request().Context()
		databaseFind := new(api.DatabaseFind)
		if instanceIDStr := c.QueryParam("instance"); instanceIDStr != "" {
			instanceID, err := strconv.Atoi(instanceIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter instance is not a number: %s", instanceIDStr)).SetInternal(err)
			}
			databaseFind.InstanceID = &instanceID
		}
		if name := c.QueryParam("name"); name != "" {
			databaseFind.Name = &name
		}
		if syncStatusStr := c.QueryParam("syncStatus"); syncStatusStr != "" {
			syncStatus := api.SyncStatus(syncStatusStr)
			databaseFind.SyncStatus = &syncStatus
		}
		projectIDStr := c.QueryParams().Get("project")
		if projectIDStr != "" {
			projectID, err := strconv.Atoi(projectIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("project query parameter is not a number: %s", projectIDStr)).SetInternal(err)
			}
			databaseFind.ProjectID = &projectID
		}
		dbList, err := s.store.FindDatabase(ctx, databaseFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch database list").SetInternal(err)
		}

		var filteredList []*api.Database
		role := c.Get(getRoleContextKey()).(api.Role)
		// If caller is NOT requesting for a particular project and is NOT requesting for a particular
		// instance or the caller is a Developer, then we will only return databases belonging to the
		// project where the caller is a member of.
		// Looking from the UI perspective:
		// - The database list left sidebar will only return databases related to the caller regardless of the caller's role.
		// - The database list on the instance page will return all databases if the caller is Owner or DBA, but will only return
		//   related databases if the caller is Developer.
		if projectIDStr == "" && (databaseFind.InstanceID == nil || role == api.Developer) {
			principalID := c.Get(getPrincipalIDContextKey()).(int)
			for _, database := range dbList {
				for _, projectMember := range database.Project.ProjectMemberList {
					if projectMember.PrincipalID == principalID {
						filteredList = append(filteredList, database)
						break
					}
				}
			}
		} else {
			filteredList = dbList
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, filteredList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal database list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:databaseID", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database with ID %d", id)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}
		// Wildcard(*) database is used to connect all database at instance level.
		// Do not return it via `get database by id` API.
		if database.Name == api.AllDatabaseName {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database with ID %d is a wildcard *", id))
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, database); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/database/:databaseID", func(c echo.Context) error {
		ctx := c.Request().Context()
		currentPrincipalID := c.Get(getPrincipalIDContextKey()).(int)
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		dbPatch := &api.DatabasePatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, dbPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch database request").SetInternal(err)
		}

		database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch database with ID %d", id)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}

		targetProject := database.Project
		if dbPatch.ProjectID != nil && *dbPatch.ProjectID != database.ProjectID {
			// Before updating database's projectID, we need to check if there are still bound sheets.
			sheetList, err := s.store.FindSheet(ctx, &api.SheetFind{DatabaseID: &database.ID}, currentPrincipalID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find sheets by database ID: %d", database.ID)).SetInternal(err)
			}
			if len(sheetList) > 0 {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("The transferring database has %d bound sheets, please go to SQL editor to unbind them first", len(sheetList)))
			}

			toProject, err := s.store.GetProjectByID(ctx, *dbPatch.ProjectID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find project with ID %d", *dbPatch.ProjectID)).SetInternal(err)
			}
			if toProject == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", *dbPatch.ProjectID))
			}
			targetProject = toProject
		}

		// Patch database labels
		// We will completely replace the old labels with the new ones, except bb.environment is immutable and
		// must match instance environment.
		if dbPatch.Labels != nil {
			if err := s.setDatabaseLabels(ctx, *dbPatch.Labels, database, targetProject, dbPatch.UpdaterID, false /* validateOnly */); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to set database labels").SetInternal(err)
			}
		}

		// If we are transferring the database to a different project, then we create a project activity in both
		// the old project and new project.
		var dbExisting *api.Database
		if dbPatch.ProjectID != nil {
			dbExisting, err = s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &dbPatch.ID})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch database with ID %d", id)).SetInternal(err)
			}
			if dbExisting == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
			}
		}

		dbPatched, err := s.store.PatchDatabase(ctx, dbPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch database ID: %v", id)).SetInternal(err)
		}

		// Create transferring database project activity.
		if dbPatch.ProjectID != nil {
			bytes, err := json.Marshal(api.ActivityProjectDatabaseTransferPayload{
				DatabaseID:   dbPatched.ID,
				DatabaseName: dbPatched.Name,
			})
			if err == nil {
				dbExisting.Project, err = s.store.GetProjectByID(ctx, dbExisting.ProjectID)
				if err == nil {
					activityCreate := &api.ActivityCreate{
						CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
						ContainerID: dbExisting.ProjectID,
						Type:        api.ActivityProjectDatabaseTransfer,
						Level:       api.ActivityInfo,
						Comment:     fmt.Sprintf("Transferred out database %q to project %q.", dbPatched.Name, dbPatched.Project.Name),
						Payload:     string(bytes),
					}
					_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
				}

				if err != nil {
					log.Warn("Failed to create project activity after transferring database",
						zap.Int("database_id", dbPatched.ID),
						zap.String("database_name", dbPatched.Name),
						zap.Int("old_project_id", dbExisting.ProjectID),
						zap.Int("new_project_id", dbPatched.ProjectID),
						zap.Error(err))
				}

				{
					activityCreate := &api.ActivityCreate{
						CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
						ContainerID: dbPatched.ProjectID,
						Type:        api.ActivityProjectDatabaseTransfer,
						Level:       api.ActivityInfo,
						Comment:     fmt.Sprintf("Transferred in database %q from project %q.", dbExisting.Name, dbExisting.Project.Name),
						Payload:     string(bytes),
					}
					_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
					if err != nil {
						log.Warn("Failed to create project activity after transferring database",
							zap.Int("database_id", dbPatched.ID),
							zap.String("database_name", dbPatched.Name),
							zap.Int("old_project_id", dbExisting.ProjectID),
							zap.Int("new_project_id", dbPatched.ProjectID),
							zap.Error(err))
					}
				}
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, dbPatched); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal database ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:databaseID/table", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		tableFind := &api.TableFind{
			DatabaseID: &id,
		}
		tableList, err := s.store.FindTable(ctx, tableFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch table list for database id: %d", id)).SetInternal(err)
		}

		for _, table := range tableList {
			columnFind := &api.ColumnFind{
				DatabaseID: &id,
				TableID:    &table.ID,
			}
			columnList, err := s.store.FindColumn(ctx, columnFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch column list for database id: %d, table name: %s", id, table.Name)).SetInternal(err)
			}
			table.ColumnList = columnList

			indexFind := &api.IndexFind{
				DatabaseID: &id,
				TableID:    &table.ID,
			}
			indexList, err := s.store.FindIndex(ctx, indexFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch index list for database id: %d, table name: %s", id, table.Name)).SetInternal(err)
			}
			table.IndexList = indexList
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, tableList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal fetch table list response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:databaseID/table/:tableName", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}

		tableName := c.Param("tableName")
		tableFind := &api.TableFind{
			DatabaseID: &id,
			Name:       &tableName,
		}
		table, err := s.store.GetTable(ctx, tableFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch table for database id: %d, table name: %s", id, tableName)).SetInternal(err)
		}
		if table == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("table %q not found from database %v", tableName, id)).SetInternal(err)
		}
		table.Database = database

		columnFind := &api.ColumnFind{
			DatabaseID: &id,
			TableID:    &table.ID,
		}
		columnList, err := s.store.FindColumn(ctx, columnFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch column list for database id: %d, table name: %s", id, tableName)).SetInternal(err)
		}
		table.ColumnList = columnList

		indexFind := &api.IndexFind{
			DatabaseID: &id,
			TableID:    &table.ID,
		}
		indexList, err := s.store.FindIndex(ctx, indexFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch index list for database id: %d, table name: %s", id, table.Name)).SetInternal(err)
		}
		table.IndexList = indexList

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, table); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal fetch table response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:databaseID/view", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		viewFind := &api.ViewFind{
			DatabaseID: &id,
		}
		viewList, err := s.store.FindView(ctx, viewFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch view list for database ID: %d", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, viewList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal fetch view list response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:databaseID/extension", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		dbExtensionFind := &api.DBExtensionFind{
			DatabaseID: &id,
		}
		dbExtensionList, err := s.store.FindDBExtension(ctx, dbExtensionFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch dbExtension list for database ID: %d", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, dbExtensionList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal fetch dbExtension list response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:databaseID/schema", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}

		driver, err := getAdminDatabaseDriver(ctx, database.Instance, database.Name, s.pgInstance.BaseDir, s.profile.DataDir)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get database driver").SetInternal(err)
		}
		defer driver.Close(ctx)
		var schemaBuf bytes.Buffer
		if _, err := driver.Dump(ctx, database.Name, &schemaBuf, true /*schemaOnly*/); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to dump database schema").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextPlainCharsetUTF8)
		if _, err := c.Response().Write(schemaBuf.Bytes()); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to write schema response for database %v", id)).SetInternal(err)
		}
		return nil
	})

	g.POST("/database/:databaseID/backup", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		backupCreate := &api.BackupCreate{
			CreatorID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, backupCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create backup request").SetInternal(err)
		}

		database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}

		storeBackupList, err := s.store.FindBackup(ctx, &api.BackupFind{
			DatabaseID: &id,
			Name:       &backupCreate.Name,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch backup with name %q", backupCreate.Name)).SetInternal(err)
		}
		if len(storeBackupList) > 0 {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Backup %q already exists", backupCreate.Name))
		}

		backup, err := s.scheduleBackupTask(ctx, database, backupCreate.Name, backupCreate.Type, c.Get(getPrincipalIDContextKey()).(int))
		if err != nil {
			if common.ErrorCode(err) == common.DbConnectionFailure {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to connect to instance %q", database.Instance.Name)).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to schedule task for backup %q", backupCreate.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, backup); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create backup response").SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:databaseID/backup", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}

		backupFind := &api.BackupFind{
			DatabaseID: &id,
		}
		backupList, err := s.store.FindBackup(ctx, backupFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to backup list for database id: %d", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, backupList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal fetch backup list response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/database/:databaseID/backup-setting", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		backupSettingUpsert := &api.BackupSettingUpsert{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, backupSettingUpsert); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed set backup setting request").SetInternal(err)
		}
		backupSettingUpsert.UpdaterID = c.Get(getPrincipalIDContextKey()).(int)

		db, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}
		if db == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}
		backupSettingUpsert.EnvironmentID = db.Instance.Environment.ID

		backupSetting, err := s.store.UpsertBackupSetting(ctx, backupSettingUpsert)
		if err != nil {
			if common.ErrorCode(err) == common.Invalid {
				return echo.NewHTTPError(http.StatusBadRequest, "Invalid backup setting").SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to set backup setting").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, backupSetting); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create set backup setting response").SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:databaseID/backup-setting", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}

		backupSetting, err := s.store.GetBackupSettingByDatabaseID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get backup setting for database id: %d", id)).SetInternal(err)
		}
		if backupSetting == nil {
			// Returns the backup setting with UNKNOWN_ID to indicate the database has no backup
			backupSetting = &api.BackupSetting{
				ID: api.UnknownID,
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, backupSetting); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal get backup setting response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:databaseID/data-source/:dataSourceID", func(c echo.Context) error {
		ctx := c.Request().Context()
		databaseID, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		dataSourceID, err := strconv.Atoi(c.Param("dataSourceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Data source ID is not a number: %s", c.Param("dataSourceID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &databaseID, IncludeAllDatabase: true})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", databaseID)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", databaseID))
		}

		dataSourceFind := &api.DataSourceFind{
			ID: &dataSourceID,
		}
		dataSource, err := s.store.GetDataSource(ctx, dataSourceFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch data source by ID %d", dataSourceID)).SetInternal(err)
		}
		if dataSource == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Data source not found with ID %d", dataSourceID))
		}
		if dataSource.DatabaseID != databaseID {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("data source not found by ID %d and database ID %d", dataSourceID, databaseID))
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, dataSource); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal find data source response").SetInternal(err)
		}
		return nil
	})

	g.POST("/database/:databaseID/data-source", func(c echo.Context) error {
		ctx := c.Request().Context()
		databaseID, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &databaseID, IncludeAllDatabase: true})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", databaseID)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", databaseID))
		}

		dataSourceCreate := &api.DataSourceCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, dataSourceCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create data source request").SetInternal(err)
		}
		if !s.feature(api.FeatureReadReplicaConnection) {
			if dataSourceCreate.HostOverride != "" || dataSourceCreate.PortOverride != "" {
				return echo.NewHTTPError(http.StatusForbidden, api.FeatureReadReplicaConnection.AccessErrorMessage())
			}
		}

		if dataSourceCreate.Type == api.Admin && (dataSourceCreate.HostOverride != "" || dataSourceCreate.PortOverride != "") {
			return echo.NewHTTPError(http.StatusBadRequest, "Host and port override cannot be set for admin type of data sources.")
		}

		dataSourceCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)
		dataSourceCreate.DatabaseID = databaseID

		dataSource, err := s.store.CreateDataSource(ctx, dataSourceCreate)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create data source").SetInternal(err)
		}

		// Refetch the instance to get the updated data source.
		updatedInstance, err := s.store.GetInstanceByID(ctx, database.InstanceID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated instance with ID %d", database.InstanceID)).SetInternal(err)
		}
		if _, err := s.syncInstance(ctx, updatedInstance); err != nil {
			log.Warn("Failed to sync instance",
				zap.Int("instance_id", updatedInstance.ID),
				zap.Error(err))
		}
		// Sync all databases in the instance asynchronously.
		instanceDatabaseSyncChan <- updatedInstance

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, dataSource); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create data source response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/database/:databaseID/data-source/:dataSourceID", func(c echo.Context) error {
		ctx := c.Request().Context()
		databaseID, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		dataSourceID, err := strconv.Atoi(c.Param("dataSourceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Data source ID is not a number: %s", c.Param("dataSourceID"))).SetInternal(err)
		}

		// Because data source could use a wildcard database "*" as its database,
		// so we need to include wildcard databases when check if relevant database exists.
		database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &databaseID, IncludeAllDatabase: true})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", databaseID)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", databaseID))
		}

		dataSourceFind := &api.DataSourceFind{
			ID:         &dataSourceID,
			DatabaseID: &databaseID,
		}
		dataSourceOld, err := s.store.GetDataSource(ctx, dataSourceFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find data source").SetInternal(err)
		}
		if dataSourceOld == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("data source not found by ID %d and database ID %d", dataSourceID, databaseID))
		}

		dataSourcePatch := &api.DataSourcePatch{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, dataSourcePatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch data source request").SetInternal(err)
		}
		if !s.feature(api.FeatureReadReplicaConnection) {
			// In the non-enterprise version, we should allow users to set HostOverride or PortOverride to the empty string.
			if (dataSourcePatch.HostOverride != nil && *dataSourcePatch.HostOverride != "") || (dataSourcePatch.PortOverride != nil && *dataSourcePatch.PortOverride != "") {
				return echo.NewHTTPError(http.StatusForbidden, api.FeatureReadReplicaConnection.AccessErrorMessage())
			}
		}

		dataSourcePatch.ID = dataSourceID
		dataSourcePatch.UpdaterID = c.Get(getPrincipalIDContextKey()).(int)

		if dataSourcePatch.UseEmptyPassword != nil && *dataSourcePatch.UseEmptyPassword {
			password := ""
			dataSourcePatch.Password = &password
		}
		if dataSourceOld.Type == api.Admin && (dataSourcePatch.HostOverride != nil || dataSourcePatch.PortOverride != nil) {
			return echo.NewHTTPError(http.StatusBadRequest, "Host and port override cannot be set for admin type of data sources.")
		}

		dataSourceNew, err := s.store.PatchDataSource(ctx, dataSourcePatch)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update data source with ID %d", dataSourceID)).SetInternal(err)
		}

		// Refetch the instance to get the updated data source.
		updatedInstance, err := s.store.GetInstanceByID(ctx, database.InstanceID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated instance with ID %d", database.InstanceID)).SetInternal(err)
		}
		if _, err := s.syncInstance(ctx, updatedInstance); err != nil {
			log.Warn("Failed to sync instance",
				zap.Int("instance_id", updatedInstance.ID),
				zap.Error(err))
		}
		// Sync all databases in the instance asynchronously.
		instanceDatabaseSyncChan <- updatedInstance

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, dataSourceNew); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal patch data source response").SetInternal(err)
		}
		return nil
	})

	g.DELETE("/database/:databaseID/data-source/:dataSourceID", func(c echo.Context) error {
		ctx := c.Request().Context()
		databaseID, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		dataSourceID, err := strconv.Atoi(c.Param("dataSourceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Data source ID is not a number: %s", c.Param("dataSourceID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{ID: &databaseID, IncludeAllDatabase: true})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", databaseID)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", databaseID))
		}

		dataSourceFind := &api.DataSourceFind{
			ID:         &dataSourceID,
			DatabaseID: &databaseID,
		}
		dataSource, err := s.store.GetDataSource(ctx, dataSourceFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find data source").SetInternal(err)
		}
		if dataSource == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Data source not found by ID %d and database ID %d", dataSourceID, databaseID))
		}
		// Only allow to delete ReadOnly data source at present.
		if dataSource.Type != api.RO {
			return echo.NewHTTPError(http.StatusForbidden, "Data source type is not read only")
		}

		if err := s.store.DeleteDataSource(ctx, &api.DataSourceDelete{
			ID:         dataSource.ID,
			InstanceID: database.InstanceID,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete data source").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	g.POST("/database/:databaseID/edit", func(c echo.Context) error {
		ctx := c.Request().Context()
		databaseID, err := strconv.Atoi(c.Param("databaseID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database ID is not a number: %s", c.Param("databaseID"))).SetInternal(err)
		}

		database, err := s.store.GetDatabase(ctx, &api.DatabaseFind{
			ID: &databaseID,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find database").SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", databaseID))
		}

		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body").SetInternal(err)
		}
		databaseEdit := &api.DatabaseEdit{}
		if err := json.Unmarshal(body, databaseEdit); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed post database edit request").SetInternal(err)
		}

		engineType := parser.EngineType(database.Instance.Engine)
		statement, err := edit.DeparseDatabaseEdit(engineType, databaseEdit)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to deparse DatabaseEdit").SetInternal(err)
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextPlainCharsetUTF8)
		if _, err := c.Response().Write([]byte(statement)); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to write DDL statement response for database %v", databaseID)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) setDatabaseLabels(ctx context.Context, labelsJSON string, database *api.Database, project *api.Project, updaterID int, validateOnly bool) error {
	// NOTE: this is a partially filled DatabaseLabel
	var labels []*api.DatabaseLabel
	if err := json.Unmarshal([]byte(labelsJSON), &labels); err != nil {
		return err
	}

	// For scalability, each database can have up to four labels for now.
	if len(labels) > api.DatabaseLabelSizeMax {
		err := errors.Errorf("database labels are up to a maximum of %d", api.DatabaseLabelSizeMax)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}

	rowStatus := api.Normal
	labelKeyList, err := s.store.FindLabelKey(ctx, &api.LabelKeyFind{RowStatus: &rowStatus})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find label key list").SetInternal(err)
	}

	if err = validateDatabaseLabelList(labels, labelKeyList, database.Instance.Environment.Name); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to validate database labels").SetInternal(err)
	}

	// Validate labels can match database name template on the project if the
	// template is not a wildcard.
	if project.DBNameTemplate != "" {
		tokens := make(map[string]string)
		for _, label := range labels {
			tokens[label.Key] = tokens[label.Value]
		}
		baseDatabaseName, err := api.GetBaseDatabaseName(database.Name, project.DBNameTemplate, labelsJSON)
		if err != nil {
			return errors.Wrapf(err, "api.GetBaseDatabaseName(%q, %q, %q) failed", database.Name, project.DBNameTemplate, labelsJSON)
		}
		if _, err := formatDatabaseName(baseDatabaseName, project.DBNameTemplate, tokens); err != nil {
			err := errors.Errorf("database labels don't match with database name template %q", project.DBNameTemplate)
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
	}

	if !validateOnly {
		if _, err = s.store.SetDatabaseLabelList(ctx, labels, database.ID, updaterID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to set database labels, database ID: %v", database.ID)).SetInternal(err)
		}
	}
	return nil
}

// Try to get database driver using the instance's admin data source.
// Upon successful return, caller MUST call driver.Close, otherwise, it will leak the database connection.
func getAdminDatabaseDriver(ctx context.Context, instance *api.Instance, databaseName, pgInstanceDir, dataDir string) (db.Driver, error) {
	connCfg, err := getConnectionConfig(instance, databaseName)
	if err != nil {
		return nil, err
	}

	driver, err := getDatabaseDriver(
		ctx,
		instance.Engine,
		db.DriverConfig{
			PgInstanceDir: pgInstanceDir,
			ResourceDir:   common.GetResourceDir(dataDir),
			BinlogDir:     getBinlogAbsDir(dataDir, instance.ID),
		},
		connCfg,
		db.ConnectionContext{
			EnvironmentName: instance.Environment.Name,
			InstanceName:    instance.Name,
		},
	)
	if err != nil {
		return nil, err
	}

	return driver, nil
}

// getConnectionConfig returns the connection config of the `databaseName` on `instance`.
func getConnectionConfig(instance *api.Instance, databaseName string) (db.ConnectionConfig, error) {
	adminDataSource := api.DataSourceFromInstanceWithType(instance, api.Admin)
	if adminDataSource == nil {
		return db.ConnectionConfig{}, common.Errorf(common.Internal, "admin data source not found for instance %d", instance.ID)
	}

	return db.ConnectionConfig{
		Username: adminDataSource.Username,
		Password: adminDataSource.Password,
		TLSConfig: db.TLSConfig{
			SslCA:   adminDataSource.SslCa,
			SslCert: adminDataSource.SslCert,
			SslKey:  adminDataSource.SslKey,
		},
		Host:     instance.Host,
		Port:     instance.Port,
		Database: databaseName,
	}, nil
}

// We'd like to use read-only data source whenever possible, but fallback to admin data source if there's no read-only data source.
// Upon successful return, caller MUST call driver.Close, otherwise, it will leak the database connection.
func tryGetReadOnlyDatabaseDriver(ctx context.Context, instance *api.Instance, databaseName string) (db.Driver, error) {
	dataSource := api.DataSourceFromInstanceWithType(instance, api.RO)
	// If there are no read-only data source, fall back to admin data source.
	if dataSource == nil {
		dataSource = api.DataSourceFromInstanceWithType(instance, api.Admin)
	}
	if dataSource == nil {
		return nil, common.Errorf(common.Internal, "data source not found for instance %d", instance.ID)
	}

	host, port := instance.Host, instance.Port
	if dataSource.HostOverride != "" || dataSource.PortOverride != "" {
		host, port = dataSource.HostOverride, dataSource.PortOverride
	}
	driver, err := getDatabaseDriver(
		ctx,
		instance.Engine,
		// We don't need postgres installation for query.
		db.DriverConfig{},
		db.ConnectionConfig{
			Username: dataSource.Username,
			Password: dataSource.Password,
			Host:     host,
			Port:     port,
			Database: databaseName,
			TLSConfig: db.TLSConfig{
				SslCA:   dataSource.SslCa,
				SslCert: dataSource.SslCert,
				SslKey:  dataSource.SslKey,
			},
			ReadOnly: true,
		},
		db.ConnectionContext{
			EnvironmentName: instance.Environment.Name,
			InstanceName:    instance.Name,
		},
	)
	if err != nil {
		return nil, err
	}

	return driver, nil
}

// Retrieve db.Driver connection with standard parameters for all type data source.
func getDatabaseDriver(ctx context.Context, engine db.Type, driverConfig db.DriverConfig, connectionConfig db.ConnectionConfig, connCtx db.ConnectionContext) (db.Driver, error) {
	driver, err := db.Open(
		ctx,
		engine,
		driverConfig,
		connectionConfig,
		connCtx,
	)
	if err != nil {
		return nil, common.Wrapf(err, common.DbConnectionFailure, "failed to connect database at %s:%s with user %q", connectionConfig.Host, connectionConfig.Port, connectionConfig.Username)
	}
	return driver, nil
}

func validateDatabaseLabelList(labelList []*api.DatabaseLabel, labelKeyList []*api.LabelKey, environmentName string) error {
	keyValueList := make(map[string]map[string]bool)
	for _, labelKey := range labelKeyList {
		keyValueList[labelKey.Key] = map[string]bool{}
		for _, value := range labelKey.ValueList {
			keyValueList[labelKey.Key][value] = true
		}
	}

	var environmentValue *string

	// check label key & value availability
	for _, label := range labelList {
		if label.Key == api.EnvironmentKeyName {
			environmentValue = &label.Value
			continue
		}
		if _, ok := keyValueList[label.Key]; !ok {
			return common.Errorf(common.Invalid, "invalid database label key: %v", label.Key)
		}
	}

	// Environment label must exist and is immutable.
	if environmentValue == nil {
		return common.Errorf(common.NotFound, "database label key %v not found", api.EnvironmentKeyName)
	}
	if environmentName != *environmentValue {
		return common.Errorf(common.Invalid, "cannot mutate database label key %v from %v to %v", api.EnvironmentKeyName, environmentName, *environmentValue)
	}

	return nil
}
