package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func (s *Server) registerDatabaseRoutes(g *echo.Group) {
	g.POST("/database", func(c echo.Context) error {
		ctx := context.Background()
		databaseCreate := &api.DatabaseCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, databaseCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create database request").SetInternal(err)
		}

		databaseCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)
		instance, err := s.InstanceService.FindInstance(ctx, &api.InstanceFind{ID: &databaseCreate.InstanceID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find instance").SetInternal(err)
		}
		databaseCreate.EnvironmentID = instance.EnvironmentID

		database, err := s.DatabaseService.CreateDatabase(ctx, databaseCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Database name already exists: %s", databaseCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create database").SetInternal(err)
		}
		if err := s.composeDatabaseRelationship(ctx, database); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch created database relationship").SetInternal(err)
		}

		// Set database labels, except bb.environment is immutable and must match instance environment.
		// This needs to be after we compose database relationship.
		if databaseCreate.Labels != nil && *databaseCreate.Labels != "" {
			if err := s.setDatabaseLabels(ctx, *databaseCreate.Labels, database, databaseCreate.CreatorID, false /* validateOnly */); err != nil {
				return err
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, database); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create database response").SetInternal(err)
		}
		return nil
	})

	g.GET("/database", func(c echo.Context) error {
		ctx := context.Background()
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
		projectIDStr := c.QueryParams().Get("project")
		if projectIDStr != "" {
			projectID, err := strconv.Atoi(projectIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("project query parameter is not a number: %s", projectIDStr)).SetInternal(err)
			}
			databaseFind.ProjectID = &projectID
		}
		list, err := s.composeDatabaseListByFind(ctx, databaseFind)
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
			for _, database := range list {
				for _, projectMember := range database.Project.ProjectMemberList {
					if projectMember.PrincipalID == principalID {
						filteredList = append(filteredList, database)
						break
					}
				}
			}
		} else {
			filteredList = list
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, filteredList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal database list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:id", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		databaseFind := &api.DatabaseFind{
			ID: &id,
		}
		database, err := s.composeDatabaseByFind(ctx, databaseFind)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, database); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/database/:id", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		databasePatch := &api.DatabasePatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, databasePatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch database request").SetInternal(err)
		}

		database, err := s.composeDatabaseByFind(ctx, &api.DatabaseFind{
			ID: &id,
		})
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch database ID: %v", id)).SetInternal(err)
		}

		// Patch database labels
		// We will completely replace the old labels with the new ones, except bb.environment is immutable and
		// must match instance environment.
		if databasePatch.Labels != nil {
			if err := s.setDatabaseLabels(ctx, *databasePatch.Labels, database, databasePatch.UpdaterID, false /* validateOnly */); err != nil {
				return err
			}
		}

		// If we are transferring the database to a different project, then we create a project activity in both
		// the old project and new project.
		var existingDatabase *api.Database
		if databasePatch.ProjectID != nil {
			existingDatabase, err = s.DatabaseService.FindDatabase(ctx, &api.DatabaseFind{
				ID: &databasePatch.ID,
			})
			if err != nil {
				if common.ErrorCode(err) == common.NotFound {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch database ID: %v", id)).SetInternal(err)
			}
		}

		database, err = s.DatabaseService.PatchDatabase(ctx, databasePatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch database ID: %v", id)).SetInternal(err)
		}

		if err := s.composeDatabaseRelationship(ctx, database); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated database relationship: %v", database.Name)).SetInternal(err)
		}

		// Create transferring database project activity.
		if databasePatch.ProjectID != nil {
			bytes, err := json.Marshal(api.ActivityProjectDatabaseTransferPayload{
				DatabaseID:   database.ID,
				DatabaseName: database.Name,
			})
			if err == nil {
				existingDatabase.Project, err = s.composeProjectByID(ctx, existingDatabase.ProjectID)
				if err == nil {
					activityCreate := &api.ActivityCreate{
						CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
						ContainerID: existingDatabase.ProjectID,
						Type:        api.ActivityProjectDatabaseTransfer,
						Level:       api.ActivityInfo,
						Comment: fmt.Sprintf("Transferred out database %q to project %q.",
							database.Name, database.Project.Name),
						Payload: string(bytes),
					}
					_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
				}

				if err != nil {
					s.l.Warn("Failed to create project activity after transferring database",
						zap.Int("database_id", database.ID),
						zap.String("database_name", database.Name),
						zap.Int("old_project_id", existingDatabase.ProjectID),
						zap.Int("new_project_id", database.ProjectID),
						zap.Error(err))
				}

				{
					activityCreate := &api.ActivityCreate{
						CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
						ContainerID: database.ProjectID,
						Type:        api.ActivityProjectDatabaseTransfer,
						Level:       api.ActivityInfo,
						Comment: fmt.Sprintf("Transferred in database %q from project %q.",
							existingDatabase.Name, existingDatabase.Project.Name),
						Payload: string(bytes),
					}
					_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
					if err != nil {
						s.l.Warn("Failed to create project activity after transferring database",
							zap.Int("database_id", database.ID),
							zap.String("database_name", database.Name),
							zap.Int("old_project_id", existingDatabase.ProjectID),
							zap.Int("new_project_id", database.ProjectID),
							zap.Error(err))
					}
				}
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, database); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal database ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:id/table", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		databaseFind := &api.DatabaseFind{
			ID: &id,
		}
		database, err := s.composeDatabaseByFind(ctx, databaseFind)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}

		tableFind := &api.TableFind{
			DatabaseID: &id,
		}
		tableList, err := s.TableService.FindTableList(ctx, tableFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch table list for database id: %d", id)).SetInternal(err)
		}

		for _, table := range tableList {
			table.Database = database
			columnFind := &api.ColumnFind{
				DatabaseID: &id,
				TableID:    &table.ID,
			}
			table.ColumnList, err = s.ColumnService.FindColumnList(ctx, columnFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch colmun list for database id: %d, table name: %s", id, table.Name)).SetInternal(err)
			}

			indexFind := &api.IndexFind{
				DatabaseID: &id,
				TableID:    &table.ID,
			}
			table.IndexList, err = s.IndexService.FindIndexList(ctx, indexFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch index list for database id: %d, table name: %s", id, table.Name)).SetInternal(err)
			}

			if err := s.composeTableRelationship(ctx, table); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to compose table relationship").SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, tableList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal fetch table list response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:id/table/:tableName", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		databaseFind := &api.DatabaseFind{
			ID: &id,
		}
		database, err := s.composeDatabaseByFind(ctx, databaseFind)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}

		tableName := c.Param("tableName")
		tableFind := &api.TableFind{
			DatabaseID: &id,
			Name:       &tableName,
		}
		table, err := s.TableService.FindTable(ctx, tableFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch table for database id: %d, table name: %s", id, tableName)).SetInternal(err)
		}

		table.Database = database

		columnFind := &api.ColumnFind{
			DatabaseID: &id,
			TableID:    &table.ID,
		}
		table.ColumnList, err = s.ColumnService.FindColumnList(ctx, columnFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch colmun list for database id: %d, table name: %s", id, tableName)).SetInternal(err)
		}

		indexFind := &api.IndexFind{
			DatabaseID: &id,
			TableID:    &table.ID,
		}
		table.IndexList, err = s.IndexService.FindIndexList(ctx, indexFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch index list for database id: %d, table name: %s", id, table.Name)).SetInternal(err)
		}

		if err := s.composeTableRelationship(ctx, table); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to compose table relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, table); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal fetch table response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:id/view", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		databaseFind := &api.DatabaseFind{
			ID: &id,
		}
		database, err := s.composeDatabaseByFind(ctx, databaseFind)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}

		viewFind := &api.ViewFind{
			DatabaseID: &id,
		}
		viewList, err := s.ViewService.FindViewList(ctx, viewFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch view list for database id: %d", id)).SetInternal(err)
		}

		for _, view := range viewList {
			view.Database = database

			if err := s.composeViewRelationship(ctx, view); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to compose view relationship").SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, viewList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal fetch view list response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.POST("/database/:id/backup", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		backupCreate := &api.BackupCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, backupCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create backup request").SetInternal(err)
		}
		backupCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)

		databaseFind := &api.DatabaseFind{
			ID: &id,
		}
		database, err := s.composeDatabaseByFind(ctx, databaseFind)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}

		backupCreate.Path, err = getAndCreateBackupPath(s.dataDir, database, backupCreate.Name)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create backup directory for database ID: %v", id)).SetInternal(err)
		}

		driver, err := getDatabaseDriver(ctx, database.Instance, database.Name, s.l)
		if err != nil {
			return err
		}
		defer driver.Close(ctx)

		version, err := getLatestSchemaVersion(ctx, driver, database.Name)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get migration history for database %q", database.Name)).SetInternal(err)
		}
		backupCreate.MigrationHistoryVersion = version

		backup, err := s.BackupService.CreateBackup(ctx, backupCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Backup name already exists: %s", backupCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create backup").SetInternal(err)
		}
		if err := s.composeBackupRelationship(ctx, backup); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to compose backup relationship").SetInternal(err)
		}

		payload := api.TaskDatabaseBackupPayload{
			BackupID: backup.ID,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create backup task payload").SetInternal(err)
		}

		createdPipeline, err := s.PipelineService.CreatePipeline(ctx, &api.PipelineCreate{
			Name:      fmt.Sprintf("backup-pipeline-%s", backup.Name),
			CreatorID: backupCreate.CreatorID,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create backup pipeline").SetInternal(err)
		}

		createdStage, err := s.StageService.CreateStage(ctx, &api.StageCreate{
			Name:          fmt.Sprintf("backup-stage-%s", backup.Name),
			EnvironmentID: database.Instance.EnvironmentID,
			PipelineID:    createdPipeline.ID,
			CreatorID:     backupCreate.CreatorID,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create backup stage").SetInternal(err)
		}

		_, err = s.TaskService.CreateTask(ctx, &api.TaskCreate{
			Name:       fmt.Sprintf("backup-task-%s", backup.Name),
			PipelineID: createdPipeline.ID,
			StageID:    createdStage.ID,
			InstanceID: database.InstanceID,
			DatabaseID: &database.ID,
			Status:     api.TaskPending,
			Type:       api.TaskDatabaseBackup,
			Payload:    string(bytes),
			CreatorID:  backupCreate.CreatorID,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create backup task").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, backup); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create backup response").SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:id/backup", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		databaseFind := &api.DatabaseFind{
			ID: &id,
		}
		_, err = s.composeDatabaseByFind(ctx, databaseFind)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}

		backupFind := &api.BackupFind{
			DatabaseID: &id,
		}
		backupList, err := s.BackupService.FindBackupList(ctx, backupFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to backup list for database id: %d", id)).SetInternal(err)
		}

		for _, backup := range backupList {
			if err := s.composeBackupRelationship(ctx, backup); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to compose backup relationship").SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, backupList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal fetch backup list response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/database/:id/backupsetting", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		backupSettingUpsert := &api.BackupSettingUpsert{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, backupSettingUpsert); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted set backup setting request").SetInternal(err)
		}
		backupSettingUpsert.UpdaterID = c.Get(getPrincipalIDContextKey()).(int)

		databaseFind := &api.DatabaseFind{
			ID: &id,
		}
		db, err := s.composeDatabaseByFind(ctx, databaseFind)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}
		backupSettingUpsert.EnvironmentID = db.Instance.Environment.ID

		backupSetting, err := s.BackupService.UpsertBackupSetting(ctx, backupSettingUpsert)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to set backup setting").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, backupSetting); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create set backup setting response").SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:id/backupsetting", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		databaseFind := &api.DatabaseFind{
			ID: &id,
		}
		_, err = s.composeDatabaseByFind(ctx, databaseFind)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}

		backupSettingFind := &api.BackupSettingFind{
			DatabaseID: &id,
		}
		backupSetting, err := s.BackupService.FindBackupSetting(ctx, backupSettingFind)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				// Returns the backup setting with UNKNOWN_ID to indicate the database has no backup
				backupSetting = &api.BackupSetting{
					ID: api.UnknownID,
				}
			} else {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get backup setting for database id: %d", id)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, backupSetting); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal get backup setting response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) setDatabaseLabels(ctx context.Context, labelsJSON string, database *api.Database, updaterID int, validateOnly bool) error {
	var labels []*api.DatabaseLabel
	if err := json.Unmarshal([]byte(labelsJSON), &labels); err != nil {
		return err
	}

	// For scalability, each database can have up to four labels for now.
	if len(labels) > api.DatabaseLabelSizeMax {
		err := fmt.Errorf("database labels are up to a maximum of %d", api.DatabaseLabelSizeMax)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}

	rowStatus := api.Normal
	labelKeyList, err := s.LabelService.FindLabelKeyList(ctx, &api.LabelKeyFind{RowStatus: &rowStatus})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find label key list").SetInternal(err)
	}

	if err = validateDatabaseLabelList(labels, labelKeyList, database.Instance.Environment.Name); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to validate database labels").SetInternal(err)
	}

	if !validateOnly {
		if _, err = s.LabelService.SetDatabaseLabelList(ctx, labels, database.ID, updaterID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to set database labels, database ID: %v", database.ID)).SetInternal(err)
		}
	}
	return nil
}

func (s *Server) composeDatabaseByFind(ctx context.Context, find *api.DatabaseFind) (*api.Database, error) {
	database, err := s.DatabaseService.FindDatabase(ctx, find)
	if err != nil {
		return nil, err
	}

	if err := s.composeDatabaseRelationship(ctx, database); err != nil {
		return nil, err
	}

	return database, nil
}

func (s *Server) composeDatabaseListByFind(ctx context.Context, find *api.DatabaseFind) ([]*api.Database, error) {
	list, err := s.DatabaseService.FindDatabaseList(ctx, find)
	if err != nil {
		return nil, err
	}

	for _, database := range list {
		if err := s.composeDatabaseRelationship(ctx, database); err != nil {
			return nil, err
		}
	}

	return list, nil
}

func (s *Server) composeDatabaseRelationship(ctx context.Context, database *api.Database) error {
	var err error

	database.Creator, err = s.composePrincipalByID(ctx, database.CreatorID)
	if err != nil {
		return err
	}

	database.Updater, err = s.composePrincipalByID(ctx, database.UpdaterID)
	if err != nil {
		return err
	}

	database.Project, err = s.composeProjectByID(ctx, database.ProjectID)
	if err != nil {
		return err
	}

	database.Instance, err = s.composeInstanceByID(ctx, database.InstanceID)
	if err != nil {
		return err
	}

	if database.SourceBackupID != 0 {
		database.SourceBackup, err = s.composeBackupByID(ctx, database.SourceBackupID)
		if err != nil {
			return err
		}
	}

	database.DataSourceList = []*api.DataSource{}

	rowStatus := api.Normal
	database.AnomalyList, err = s.AnomalyService.FindAnomalyList(ctx, &api.AnomalyFind{
		RowStatus:  &rowStatus,
		DatabaseID: &database.ID,
	})
	if err != nil {
		return err
	}
	for _, anomaly := range database.AnomalyList {
		anomaly.Creator, err = s.composePrincipalByID(ctx, anomaly.CreatorID)
		if err != nil {
			return err
		}
		anomaly.Updater, err = s.composePrincipalByID(ctx, anomaly.UpdaterID)
		if err != nil {
			return err
		}
	}

	rowStatus = api.Normal
	labelList, err := s.LabelService.FindDatabaseLabelList(ctx, &api.DatabaseLabelFind{
		DatabaseID: &database.ID,
		RowStatus:  &rowStatus,
	})
	if err != nil {
		return err
	}

	// Since tenants are identified by labels in deployment config, we need an environment
	// label to identify tenants from different environment in a schema update deployment.
	// If we expose the environment label concept in the deployment config, it should look consistent in the label API.

	// Each database instance is created under a particular environment.
	// The value of bb.environment is identical to the name of the environment.

	labelList = append(labelList, &api.DatabaseLabel{
		Key:   api.EnvironmentKeyName,
		Value: database.Instance.Environment.Name,
	})

	labels, err := json.Marshal(labelList)
	if err != nil {
		return err
	}
	database.Labels = string(labels)

	return nil
}

func (s *Server) composeTableRelationship(ctx context.Context, table *api.Table) error {
	var err error

	table.Creator, err = s.composePrincipalByID(ctx, table.CreatorID)
	if err != nil {
		return err
	}

	table.Updater, err = s.composePrincipalByID(ctx, table.UpdaterID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) composeViewRelationship(ctx context.Context, view *api.View) error {
	var err error

	view.Creator, err = s.composePrincipalByID(ctx, view.CreatorID)
	if err != nil {
		return err
	}

	view.Updater, err = s.composePrincipalByID(ctx, view.UpdaterID)
	if err != nil {
		return err
	}
	return nil
}

// composeBackupByID will compose the backup by backup ID.
func (s *Server) composeBackupByID(ctx context.Context, id int) (*api.Backup, error) {
	backupFind := &api.BackupFind{
		ID: &id,
	}
	backup, err := s.BackupService.FindBackup(ctx, backupFind)
	if err != nil {
		return nil, err
	}

	if err := s.composeBackupRelationship(ctx, backup); err != nil {
		return nil, err
	}

	return backup, nil
}

// composeBackupRelationship will compose the relationship of a backup.
func (s *Server) composeBackupRelationship(ctx context.Context, backup *api.Backup) error {
	var err error
	backup.Creator, err = s.composePrincipalByID(ctx, backup.CreatorID)
	if err != nil {
		return err
	}
	backup.Updater, err = s.composePrincipalByID(ctx, backup.UpdaterID)
	if err != nil {
		return err
	}
	return nil
}

// Retrieve db.Driver connection.
// Upon successful return, caller MUST call driver.Close, otherwise, it will leak the database connection.
func getDatabaseDriver(ctx context.Context, instance *api.Instance, databaseName string, logger *zap.Logger) (db.Driver, error) {
	driver, err := db.Open(
		ctx,
		instance.Engine,
		db.DriverConfig{Logger: logger},
		db.ConnectionConfig{
			Username: instance.Username,
			Password: instance.Password,
			Host:     instance.Host,
			Port:     instance.Port,
			Database: databaseName,
		},
		db.ConnectionContext{
			EnvironmentName: instance.Environment.Name,
			InstanceName:    instance.Name,
		},
	)
	if err != nil {
		return nil, common.Errorf(common.DbConnectionFailure, fmt.Errorf("failed to connect database at %s:%s with user %q: %w", instance.Host, instance.Port, instance.Username, err))
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
		labelKey, ok := keyValueList[label.Key]
		if !ok {
			return common.Errorf(common.Invalid, fmt.Errorf("invalid database label key: %v", label.Key))
		}
		_, ok = labelKey[label.Value]
		if !ok {
			return common.Errorf(common.Invalid, fmt.Errorf("invalid database label value %v for key %v", label.Value, label.Key))
		}
	}

	// Environment label must exist and is immutable.
	if environmentValue == nil {
		return common.Errorf(common.NotFound, fmt.Errorf("database label key %v not found", api.EnvironmentKeyName))
	}
	if environmentName != *environmentValue {
		return common.Errorf(common.Invalid, fmt.Errorf("cannot mutate database label key %v from %v to %v", api.EnvironmentKeyName, environmentName, *environmentValue))
	}

	return nil
}
