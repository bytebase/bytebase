package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/parser"
	"github.com/bytebase/bytebase/plugin/parser/edit"
	"github.com/bytebase/bytebase/server/component/activity"
	"github.com/bytebase/bytebase/server/utils"
)

func (s *Server) registerDatabaseRoutes(g *echo.Group) {
	g.GET("/database", func(c echo.Context) error {
		ctx := c.Request().Context()
		databaseFind := new(api.DatabaseFind)
		if rowStatusStr := c.QueryParam("rowstatus"); rowStatusStr != "" {
			rowStatus := api.RowStatus(rowStatusStr)
			databaseFind.RowStatus = &rowStatus
		}
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
		// If the caller is a developer, we will only return databases belonging to the
		// project where the caller is a member of.
		if role == api.Developer {
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
		}

		// Patch database labels
		// We will completely replace the old labels with the new ones, except bb.environment is immutable and
		// must match instance environment.
		if dbPatch.Labels != nil {
			if err := utils.SetDatabaseLabels(ctx, s.store, *dbPatch.Labels, database, dbPatch.UpdaterID, false /* validateOnly */); err != nil {
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

		// Verify before archiving the database:
		// 1. the database has no related open-status issues.
		// 2. the database has no related saved sheets.
		if v := dbPatch.RowStatus; v != nil && *v == string(api.Archived) {
			exists, err := s.hasRelatedOpenStatusIssue(ctx, database.ProjectID, database.ID)
			if err != nil {
				return err
			}
			if exists {
				return echo.NewHTTPError(http.StatusBadRequest, "Please cancel all open status issues related to the database before archiving the database.")
			}

			normalStatus := api.Normal
			sheetFind := &api.SheetFind{
				RowStatus:  &normalStatus,
				DatabaseID: &database.ID,
			}
			sheetList, err := s.store.FindSheet(ctx, sheetFind, currentPrincipalID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find sheets related to the database with ID %d", database.ID))
			}
			if len(sheetList) > 0 {
				return echo.NewHTTPError(http.StatusBadRequest, "Please remove all saved sheets related to the database before archiving the database.")
			}
		}
		dbPatched, err := s.store.PatchDatabase(ctx, dbPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch database ID: %v", id)).SetInternal(err)
		}

		// Create archiving database activity.
		if v := dbPatch.RowStatus; v != nil && *v == string(api.Archived) {
			bytes, err := json.Marshal(api.ActivityDatabaseArchivePayload{
				InstanceID: database.InstanceID,
				DatabaseID: database.ID,
			})
			if err != nil {
				log.Warn("Failed to construct archiving database activity payload",
					zap.Error(err),
				)
			} else {
				activityCreate := &api.ActivityCreate{
					CreatorID:   currentPrincipalID,
					ContainerID: database.ProjectID,
					Type:        api.ActivityDatabaseArchive,
					Level:       api.ActivityInfo,
					Comment:     fmt.Sprintf("Archive database %q in instance %q.", database.Name, database.Instance.Name),
					Payload:     string(bytes),
				}
				if _, err := s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{}); err != nil {
					log.Warn("Failed to create project activity after archiving database",
						zap.Int("database_id", dbPatched.ID),
						zap.String("database_name", dbPatched.Name),
						zap.Int("instance_id", database.InstanceID),
						zap.Error(err),
					)
				}
			}
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
					_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{})
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
					_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{})
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

	// When query metadata is present, we will return the schema metadata. Otherwise, we will return the raw dump.
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

		dbSchema, err := s.store.GetDBSchema(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get dbSchema for database ID %v", id)).SetInternal(err)
		}
		if dbSchema == nil {
			// TODO(d): make SyncDatabaseSchema return the updated database schema.
			if err := s.SchemaSyncer.SyncDatabaseSchema(ctx, database.Instance, database.Name, true /* force */); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to sync database schema for database ID %v", id)).SetInternal(err)
			}
			newDBSchema, err := s.store.GetDBSchema(ctx, id)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get dbSchema for database ID %v", id)).SetInternal(err)
			}
			if newDBSchema == nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("New dbSchema not found for database ID %v", id)).SetInternal(err)
			}
			dbSchema = newDBSchema
		}

		isQueryRawDump := c.QueryParam("metadata") == ""
		if isQueryRawDump {
			c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextPlainCharsetUTF8)
			if _, err := c.Response().Write([]byte(dbSchema.RawDump)); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to write schema response for database %v", id)).SetInternal(err)
			}
		} else {
			c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
			metadataBytes, err := protojson.Marshal(dbSchema.Metadata)
			if err != nil {
				return err
			}
			if _, err := c.Response().Write(metadataBytes); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to write schema response for database %v", id)).SetInternal(err)
			}
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

		backup, err := s.BackupRunner.ScheduleBackupTask(ctx, database, backupCreate.Name, backupCreate.Type, c.Get(getPrincipalIDContextKey()).(int))
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
		if !s.licenseService.IsFeatureEnabled(api.FeatureReadReplicaConnection) {
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
		if _, err := s.SchemaSyncer.SyncInstance(ctx, updatedInstance); err != nil {
			log.Warn("Failed to sync instance",
				zap.Int("instance_id", updatedInstance.ID),
				zap.Error(err))
		}
		// Sync all databases in the instance asynchronously.
		s.stateCfg.InstanceDatabaseSyncChan <- updatedInstance

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
		if !s.licenseService.IsFeatureEnabled(api.FeatureReadReplicaConnection) {
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
		if _, err := s.SchemaSyncer.SyncInstance(ctx, updatedInstance); err != nil {
			log.Warn("Failed to sync instance",
				zap.Int("instance_id", updatedInstance.ID),
				zap.Error(err))
		}
		// Sync all databases in the instance asynchronously.
		s.stateCfg.InstanceDatabaseSyncChan <- updatedInstance

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
		validateResultList, err := edit.ValidateDatabaseEdit(engineType, databaseEdit)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to validate DatabaseEdit").SetInternal(err)
		}

		databaseEditResult := &api.DatabaseEditResult{
			Statement:          "",
			ValidateResultList: validateResultList,
		}
		if len(validateResultList) == 0 {
			statement, err := edit.DeparseDatabaseEdit(engineType, databaseEdit)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to deparse DatabaseEdit").SetInternal(err)
			}
			databaseEditResult.Statement = statement
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, databaseEditResult); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal database edit result response").SetInternal(err)
		}
		return nil
	})
}

func (s *Server) hasRelatedOpenStatusIssue(ctx context.Context, projectID int, databaseID int) (bool, error) {
	issueFind := &api.IssueFind{
		ProjectID:  &projectID,
		StatusList: []api.IssueStatus{api.IssueOpen},
	}
	issueList, err := s.store.FindIssue(ctx, issueFind)
	if err != nil {
		return false, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find issues related to the database with ID: %d", databaseID)).SetInternal(err)
	}
	for _, issue := range issueList {
		for _, stage := range issue.Pipeline.StageList {
			for _, task := range stage.TaskList {
				if task.DatabaseID != nil && *task.DatabaseID == databaseID {
					return true, nil
				}
			}
		}
	}

	return false, nil
}
