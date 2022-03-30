package server

import (
	"bytes"
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
		instanceRaw, err := s.InstanceService.FindInstance(ctx, &api.InstanceFind{ID: &databaseCreate.InstanceID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find instance").SetInternal(err)
		}
		if instanceRaw == nil {
			err := fmt.Errorf("Instance ID not found %v", databaseCreate.InstanceID)
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		databaseCreate.EnvironmentID = instanceRaw.EnvironmentID
		project, err := s.composeProjectByID(ctx, databaseCreate.ProjectID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find project").SetInternal(err)
		}
		if project == nil {
			err := fmt.Errorf("Project ID not found %v", databaseCreate.ProjectID)
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		if project.TenantMode == api.TenantModeTenant && !s.feature(api.FeatureMultiTenancy) {
			return echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
		}
		// Pre-validate database labels.
		if databaseCreate.Labels != nil && *databaseCreate.Labels != "" {
			// TODO(dragonly): compose Instance
			if err := s.setDatabaseLabels(ctx, *databaseCreate.Labels, &api.Database{Name: databaseCreate.Name, Instance: instanceRaw.ToInstance()} /* dummy database */, project, databaseCreate.CreatorID, true /* validateOnly */); err != nil {
				return err
			}
		}

		dbRaw, err := s.DatabaseService.CreateDatabase(ctx, databaseCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Database name already exists: %s", databaseCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create database").SetInternal(err)
		}
		db, err := s.composeDatabaseRelationship(ctx, dbRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch created database relationship").SetInternal(err)
		}

		// Set database labels, except bb.environment is immutable and must match instance environment.
		// This needs to be after we compose database relationship.
		if databaseCreate.Labels != nil && *databaseCreate.Labels != "" {
			if err := s.setDatabaseLabels(ctx, *databaseCreate.Labels, db, project, databaseCreate.CreatorID, false /* validateOnly */); err != nil {
				return err
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, db); err != nil {
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
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}
		// Wildcard(*) database is used to connect all database at instance level.
		// Do not return it via `get database by id` API.
		if database.Name == api.AllDatabaseName {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database %d is a wildcard *", id))
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

		dbPatch := &api.DatabasePatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, dbPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch database request").SetInternal(err)
		}

		dbRaw, err := s.composeDatabaseByFind(ctx, &api.DatabaseFind{
			ID: &id,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch database ID: %v", id)).SetInternal(err)
		}
		if dbRaw == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}

		targetProject := dbRaw.Project
		if dbPatch.ProjectID != nil && *dbPatch.ProjectID != dbRaw.ProjectID {
			// Before updating the database projectID, we first need to check if there are still bound sheets.
			sheetList, err := s.SheetService.FindSheetList(ctx, &api.SheetFind{
				DatabaseID: &dbRaw.ID,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find sheets by database ID: %d", dbRaw.ID)).SetInternal(err)
			}
			if len(sheetList) > 0 {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("The transferring database has %d bound sheets, unbind them first", len(sheetList)))
			}

			toProject, err := s.composeProjectByID(ctx, *dbPatch.ProjectID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find project ID: %d", *dbPatch.ProjectID)).SetInternal(err)
			}
			if toProject == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", *dbPatch.ProjectID))
			}
			targetProject = toProject

			if toProject.TenantMode == api.TenantModeTenant {
				if !s.feature(api.FeatureMultiTenancy) {
					return echo.NewHTTPError(http.StatusForbidden, api.FeatureMultiTenancy.AccessErrorMessage())
				}

				labels := dbRaw.Labels
				if dbPatch.Labels != nil {
					labels = *dbPatch.Labels
				}
				// For database being transferred to a tenant mode project, its schema version and schema has to match a peer tenant database.
				// When a peer tenant database doesn't exist, we will return an error if there are databases in the project with the same name.
				baseDatabaseName, err := api.GetBaseDatabaseName(dbRaw.Name, toProject.DBNameTemplate, labels)
				if err != nil {
					return fmt.Errorf("api.GetBaseDatabaseName(%q, %q, %q) failed, error: %v", dbRaw.Name, toProject.DBNameTemplate, labels, err)
				}
				peerSchemaVersion, peerSchema, err := s.getSchemaFromPeerTenantDatabase(ctx, dbRaw.Instance, toProject, *dbPatch.ProjectID, baseDatabaseName)
				if err != nil {
					return err
				}

				// Tenant database exists when peerSchemaVersion or peerSchema are not empty.
				if peerSchemaVersion != "" || peerSchema != "" {
					driver, err := getAdminDatabaseDriver(ctx, dbRaw.Instance, dbRaw.Name, s.l)
					if err != nil {
						return err
					}
					defer driver.Close(ctx)
					schemaVersion, err := getLatestSchemaVersion(ctx, driver, dbRaw.Name)
					if err != nil {
						return fmt.Errorf("failed to get migration history for database %q: %w", dbRaw.Name, err)
					}
					if peerSchemaVersion != schemaVersion {
						return fmt.Errorf("the schema version %q does not match the peer database schema version %q in the target tenant mode project %q", schemaVersion, peerSchemaVersion, toProject.Name)
					}

					var schemaBuf bytes.Buffer
					if err := driver.Dump(ctx, dbRaw.Name, &schemaBuf, true /* schemaOnly */); err != nil {
						return fmt.Errorf("failed to get database schema for database %q: %w", dbRaw.Name, err)
					}
					if peerSchema != schemaBuf.String() {
						return fmt.Errorf("the schema for database %q does not match the peer database schema in the target tenant mode project %q", dbRaw.Name, toProject.Name)
					}
				}
			}
		}

		// Patch database labels
		// We will completely replace the old labels with the new ones, except bb.environment is immutable and
		// must match instance environment.
		if dbPatch.Labels != nil {
			if err := s.setDatabaseLabels(ctx, *dbPatch.Labels, dbRaw, targetProject, dbPatch.UpdaterID, false /* validateOnly */); err != nil {
				return err
			}
		}

		// If we are transferring the database to a different project, then we create a project activity in both
		// the old project and new project.
		var dbRawExisting *api.DatabaseRaw
		if dbPatch.ProjectID != nil {
			dbRawExisting, err = s.DatabaseService.FindDatabase(ctx, &api.DatabaseFind{
				ID: &dbPatch.ID,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch database ID: %v", id)).SetInternal(err)
			}
			if dbRawExisting == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
			}
		}
		dbExisting, err := s.composeDatabaseRelationship(ctx, dbRawExisting)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}

		dbRawPatched, err := s.DatabaseService.PatchDatabase(ctx, dbPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch database ID: %v", id)).SetInternal(err)
		}

		dbPatched, err := s.composeDatabaseRelationship(ctx, dbRawPatched)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated database relationship: %v", dbRawPatched.Name)).SetInternal(err)
		}

		// Create transferring database project activity.
		if dbPatch.ProjectID != nil {
			bytes, err := json.Marshal(api.ActivityProjectDatabaseTransferPayload{
				DatabaseID:   dbPatched.ID,
				DatabaseName: dbPatched.Name,
			})
			if err == nil {
				dbExisting.Project, err = s.composeProjectByID(ctx, dbExisting.ProjectID)
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
					s.l.Warn("Failed to create project activity after transferring database",
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
						s.l.Warn("Failed to create project activity after transferring database",
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
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}

		tableFind := &api.TableFind{
			DatabaseID: &id,
		}
		tableRawList, err := s.TableService.FindTableList(ctx, tableFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch table list for database id: %d", id)).SetInternal(err)
		}
		var tableList []*api.Table
		for _, tableRaw := range tableRawList {
			table, err := s.composeTableRelationship(ctx, tableRaw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to compose table with ID %d and name %s", id, tableRaw.Name)).SetInternal(err)
			}
			tableList = append(tableList, table)
		}

		for _, table := range tableList {
			table.Database = database
			columnFind := &api.ColumnFind{
				DatabaseID: &id,
				TableID:    &table.ID,
			}
			columnList, err := s.ColumnService.FindColumnList(ctx, columnFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch colmun list for database id: %d, table name: %s", id, table.Name)).SetInternal(err)
			}
			table.ColumnList = columnList

			indexFind := &api.IndexFind{
				DatabaseID: &id,
				TableID:    &table.ID,
			}
			indexList, err := s.IndexService.FindIndexList(ctx, indexFind)
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
		tableRaw, err := s.TableService.FindTable(ctx, tableFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch table for database id: %d, table name: %s", id, tableName)).SetInternal(err)
		}
		if tableRaw == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("table %q not found from database %v", tableName, id)).SetInternal(err)
		}
		table, err := s.composeTableRelationship(ctx, tableRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to compose table with ID %d and name %s", id, tableName)).SetInternal(err)
		}

		table.Database = database

		columnFind := &api.ColumnFind{
			DatabaseID: &id,
			TableID:    &table.ID,
		}
		columnList, err := s.ColumnService.FindColumnList(ctx, columnFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch colmun list for database id: %d, table name: %s", id, tableName)).SetInternal(err)
		}
		table.ColumnList = columnList

		indexFind := &api.IndexFind{
			DatabaseID: &id,
			TableID:    &table.ID,
		}
		indexList, err := s.IndexService.FindIndexList(ctx, indexFind)
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
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}

		viewFind := &api.ViewFind{
			DatabaseID: &id,
		}
		viewRawList, err := s.ViewService.FindViewList(ctx, viewFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch view list for database ID: %d", id)).SetInternal(err)
		}
		var viewList []*api.View
		for _, raw := range viewRawList {
			view, err := s.composeViewRelationship(ctx, raw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to compose view relationship with ID: %d", id)).SetInternal(err)
			}
			viewList = append(viewList, view)
		}
		// TODO(dragonly): should we do this in composeViewRelationship?
		for _, view := range viewList {
			view.Database = database
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
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}

		backupCreate.Path, err = getAndCreateBackupPath(s.dataDir, database, backupCreate.Name)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create backup directory for database ID: %v", id)).SetInternal(err)
		}

		driver, err := getAdminDatabaseDriver(ctx, database.Instance, database.Name, s.l)
		if err != nil {
			return err
		}
		defer driver.Close(ctx)

		version, err := getLatestSchemaVersion(ctx, driver, database.Name)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get migration history for database %q", database.Name)).SetInternal(err)
		}
		backupCreate.MigrationHistoryVersion = version

		backupRaw, err := s.BackupService.CreateBackup(ctx, backupCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Backup name already exists: %s", backupCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create backup").SetInternal(err)
		}
		backup, err := s.composeBackupRelationship(ctx, backupRaw)
		if err != nil {
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
		database, err := s.composeDatabaseByFind(ctx, databaseFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}

		backupFind := &api.BackupFind{
			DatabaseID: &id,
		}
		backupRawList, err := s.BackupService.FindBackupList(ctx, backupFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to backup list for database id: %d", id)).SetInternal(err)
		}

		var backupList []*api.Backup
		for _, backupRaw := range backupRawList {
			backup, err := s.composeBackupRelationship(ctx, backupRaw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to compose backup relationship").SetInternal(err)
			}
			backupList = append(backupList, backup)
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
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}
		if db == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
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
		database, err := s.composeDatabaseByFind(ctx, databaseFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", id))
		}

		backupSettingFind := &api.BackupSettingFind{
			DatabaseID: &id,
		}
		backupSettingRaw, err := s.BackupService.FindBackupSetting(ctx, backupSettingFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get backup setting for database id: %d", id)).SetInternal(err)
		}
		if backupSettingRaw == nil {
			// Returns the backup setting with UNKNOWN_ID to indicate the database has no backup
			backupSettingRaw = &api.BackupSettingRaw{
				ID: api.UnknownID,
			}
		}
		// TODO(dragonly): compose this
		backupSetting := backupSettingRaw.ToBackupSetting()

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, backupSetting); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal get backup setting response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:id/datasource/:dataSourceID", func(c echo.Context) error {
		ctx := context.Background()
		databaseID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		dataSourceID, err := strconv.Atoi(c.Param("dataSourceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Data source ID is not a number: %s", c.Param("dataSourceID"))).SetInternal(err)
		}

		databaseFind := &api.DatabaseFind{
			ID: &databaseID,
		}
		database, err := s.composeDatabaseByFind(ctx, databaseFind)
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
		if dataSource.DatabaseID != databaseID {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("data source not found by ID %d and database ID %d", dataSourceID, databaseID))
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, dataSource); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal find data source response").SetInternal(err)
		}
		return nil
	})

	g.POST("/database/:id/datasource", func(c echo.Context) error {
		ctx := context.Background()
		databaseID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		databaseFind := &api.DatabaseFind{
			ID: &databaseID,
		}
		database, err := s.composeDatabaseByFind(ctx, databaseFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", databaseID)).SetInternal(err)
		}
		if database == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database not found with ID %d", databaseID))
		}

		dataSourceCreate := &api.DataSourceCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, dataSourceCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create data source request").SetInternal(err)
		}

		dataSourceCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)
		dataSourceCreate.DatabaseID = databaseID

		dataSource, err := s.store.CreateDataSource(ctx, dataSourceCreate)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create data source").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, dataSource); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create data source response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/database/:id/datasource/:dataSourceID", func(c echo.Context) error {
		ctx := context.Background()
		databaseID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		dataSourceID, err := strconv.Atoi(c.Param("dataSourceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Data source ID is not a number: %s", c.Param("dataSourceID"))).SetInternal(err)
		}

		// Because data source could use a wildcard database "*" as its database,
		// so we need to include wildcard databases when check if relevant database exists.
		databaseFind := &api.DatabaseFind{
			ID:                 &databaseID,
			IncludeAllDatabase: true,
		}
		database, err := s.composeDatabaseByFind(ctx, databaseFind)
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
		dataSourceRaw, err := s.store.GetDataSource(ctx, dataSourceFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find data source").SetInternal(err)
		}
		if dataSourceRaw == nil || dataSourceRaw.DatabaseID != databaseID {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("data source not found by ID %d and database ID %d", dataSourceID, databaseID))
		}

		dataSourcePatch := &api.DataSourcePatch{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, dataSourcePatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch data source request").SetInternal(err)
		}

		dataSourcePatch.ID = dataSourceID
		dataSourcePatch.UpdaterID = c.Get(getPrincipalIDContextKey()).(int)

		if dataSourcePatch.UseEmptyPassword != nil && *dataSourcePatch.UseEmptyPassword {
			password := ""
			dataSourcePatch.Password = &password
		}

		dataSource, err := s.store.PatchDataSource(ctx, dataSourcePatch)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to update data source with ID %d", dataSourceID)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, dataSource); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal patch data source response").SetInternal(err)
		}
		return nil
	})
}

func (s *Server) setDatabaseLabels(ctx context.Context, labelsJSON string, database *api.Database, project *api.Project, updaterID int, validateOnly bool) error {
	// NOTE: this is a partially filled DatabaseLabel
	// TODO(dragonly): should we make it cleaner?
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
	labelKeyRawList, err := s.LabelService.FindLabelKeyList(ctx, &api.LabelKeyFind{RowStatus: &rowStatus})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to find label key list").SetInternal(err)
	}
	// TODO(dragonly): implement composeLabelKeyRelationship
	var labelKeyList []*api.LabelKey
	for _, raw := range labelKeyRawList {
		labelKeyList = append(labelKeyList, raw.ToLabelKey())
	}

	if err = validateDatabaseLabelList(labels, labelKeyList, database.Instance.Environment.Name); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to validate database labels").SetInternal(err)
	}

	// Validate labels can match database name template on the project.
	if project.DBNameTemplate != "" {
		tokens := make(map[string]string)
		for _, label := range labels {
			tokens[label.Key] = tokens[label.Value]
		}
		baseDatabaseName, err := api.GetBaseDatabaseName(database.Name, project.DBNameTemplate, labelsJSON)
		if err != nil {
			return fmt.Errorf("api.GetBaseDatabaseName(%q, %q, %q) failed, error: %v", database.Name, project.DBNameTemplate, labelsJSON, err)
		}
		if _, err := formatDatabaseName(baseDatabaseName, project.DBNameTemplate, tokens); err != nil {
			err := fmt.Errorf("database labels don't match with database name template %q", project.DBNameTemplate)
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
	}

	if !validateOnly {
		if _, err = s.LabelService.SetDatabaseLabelList(ctx, labels, database.ID, updaterID); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to set database labels, database ID: %v", database.ID)).SetInternal(err)
		}
	}
	return nil
}

func (s *Server) composeDatabaseByFind(ctx context.Context, find *api.DatabaseFind) (*api.Database, error) {
	databaseRaw, err := s.DatabaseService.FindDatabase(ctx, find)
	if err != nil {
		return nil, err
	}
	if databaseRaw == nil {
		return nil, nil
	}

	database, err := s.composeDatabaseRelationship(ctx, databaseRaw)
	if err != nil {
		return nil, err
	}

	return database, nil
}

func (s *Server) composeDatabaseListByFind(ctx context.Context, find *api.DatabaseFind) ([]*api.Database, error) {
	dbRawList, err := s.DatabaseService.FindDatabaseList(ctx, find)
	if err != nil {
		return nil, err
	}

	var dbList []*api.Database
	for _, dbRaw := range dbRawList {
		db, err := s.composeDatabaseRelationship(ctx, dbRaw)
		if err != nil {
			return nil, err
		}
		dbList = append(dbList, db)
	}

	return dbList, nil
}

func (s *Server) composeDatabaseRelationship(ctx context.Context, raw *api.DatabaseRaw) (*api.Database, error) {
	db := raw.ToDatabase()

	creator, err := s.store.GetPrincipalByID(ctx, db.CreatorID)
	if err != nil {
		return nil, err
	}
	db.Creator = creator

	updater, err := s.store.GetPrincipalByID(ctx, db.UpdaterID)
	if err != nil {
		return nil, err
	}
	db.Updater = updater

	project, err := s.composeProjectByID(ctx, db.ProjectID)
	if err != nil {
		return nil, err
	}
	db.Project = project

	instance, err := s.composeInstanceByID(ctx, db.InstanceID)
	if err != nil {
		return nil, err
	}
	db.Instance = instance

	if db.SourceBackupID != 0 {
		sourceBackup, err := s.composeBackupByID(ctx, db.SourceBackupID)
		if err != nil {
			return nil, err
		}
		db.SourceBackup = sourceBackup
	}

	// For now, only wildcard(*) database has data sources and we disallow it to be returned to the client.
	// So we set this value to an empty array until we need to develop a data source for a non-wildcard database.
	db.DataSourceList = []*api.DataSource{}

	rowStatus := api.Normal
	anomalyListRaw, err := s.AnomalyService.FindAnomalyList(ctx, &api.AnomalyFind{
		RowStatus:  &rowStatus,
		DatabaseID: &db.ID,
	})
	if err != nil {
		return nil, err
	}
	var anomalyList []*api.Anomaly
	for _, anomalyRaw := range anomalyListRaw {
		anomalyList = append(anomalyList, anomalyRaw.ToAnomaly())
	}
	// TODO(dragonly): implement composeAnomalyRelationship
	db.AnomalyList = anomalyList
	for _, anomaly := range db.AnomalyList {
		anomaly.Creator, err = s.store.GetPrincipalByID(ctx, anomaly.CreatorID)
		if err != nil {
			return nil, err
		}
		anomaly.Updater, err = s.store.GetPrincipalByID(ctx, anomaly.UpdaterID)
		if err != nil {
			return nil, err
		}
	}

	rowStatus = api.Normal
	labelRawList, err := s.LabelService.FindDatabaseLabelList(ctx, &api.DatabaseLabelFind{
		DatabaseID: &db.ID,
		RowStatus:  &rowStatus,
	})
	if err != nil {
		return nil, err
	}
	// TODO(dragonly): seems like we do not need to composed this.
	// need redesign, e.g., extract the kv part which is only in memory, and the relations which are in the database.
	var labelList []*api.DatabaseLabel
	for _, raw := range labelRawList {
		labelList = append(labelList, raw.ToDatabaseLabel())
	}

	// Since tenants are identified by labels in deployment config, we need an environment
	// label to identify tenants from different environment in a schema update deployment.
	// If we expose the environment label concept in the deployment config, it should look consistent in the label API.

	// Each database instance is created under a particular environment.
	// The value of bb.environment is identical to the name of the environment.

	labelList = append(labelList, &api.DatabaseLabel{
		Key:   api.EnvironmentKeyName,
		Value: db.Instance.Environment.Name,
	})

	labels, err := json.Marshal(labelList)
	if err != nil {
		return nil, err
	}
	db.Labels = string(labels)

	return db, nil
}

func (s *Server) composeTableRelationship(ctx context.Context, raw *api.TableRaw) (*api.Table, error) {
	table := raw.ToTable()

	creator, err := s.store.GetPrincipalByID(ctx, table.CreatorID)
	if err != nil {
		return nil, err
	}
	table.Creator = creator

	updater, err := s.store.GetPrincipalByID(ctx, table.UpdaterID)
	if err != nil {
		return nil, err
	}
	table.Updater = updater

	return table, nil
}

func (s *Server) composeViewRelationship(ctx context.Context, raw *api.ViewRaw) (*api.View, error) {
	view := raw.ToView()

	creator, err := s.store.GetPrincipalByID(ctx, view.CreatorID)
	if err != nil {
		return nil, err
	}
	view.Creator = creator

	updater, err := s.store.GetPrincipalByID(ctx, view.UpdaterID)
	if err != nil {
		return nil, err
	}
	view.Updater = updater
	return view, nil
}

// composeBackupByID will compose the backup by backup ID.
func (s *Server) composeBackupByID(ctx context.Context, id int) (*api.Backup, error) {
	backupFind := &api.BackupFind{
		ID: &id,
	}
	backupRaw, err := s.BackupService.FindBackup(ctx, backupFind)
	if err != nil {
		return nil, err
	}
	if backupRaw == nil {
		return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("backup not found with ID %d", id)}
	}

	backup, err := s.composeBackupRelationship(ctx, backupRaw)
	if err != nil {
		return nil, err
	}

	return backup, nil
}

// composeBackupRelationship will compose the relationship of a backup.
func (s *Server) composeBackupRelationship(ctx context.Context, raw *api.BackupRaw) (*api.Backup, error) {
	backup := raw.ToBackup()
	creator, err := s.store.GetPrincipalByID(ctx, backup.CreatorID)
	if err != nil {
		return nil, err
	}
	backup.Creator = creator

	updater, err := s.store.GetPrincipalByID(ctx, backup.UpdaterID)
	if err != nil {
		return nil, err
	}
	backup.Updater = updater

	return backup, nil
}

// Try to get database driver using the instance's admin data source.
// Upon successful return, caller MUST call driver.Close, otherwise, it will leak the database connection.
func getAdminDatabaseDriver(ctx context.Context, instance *api.Instance, databaseName string, logger *zap.Logger) (db.Driver, error) {
	adminDataSource := api.DataSourceFromInstanceWithType(instance, api.Admin)
	if adminDataSource == nil {
		return nil, common.Errorf(common.Internal, fmt.Errorf("admin data source not found for instance %d", instance.ID))
	}

	driver, err := getDatabaseDriver(
		ctx,
		instance.Engine,
		db.DriverConfig{Logger: logger},
		db.ConnectionConfig{
			Username: adminDataSource.Username,
			Password: adminDataSource.Password,
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
		return nil, err
	}

	return driver, nil
}

// We'd like to use read-only data source whenever possible, but fallback to admin data source if there's no read-only data source.
// Upon successful return, caller MUST call driver.Close, otherwise, it will leak the database connection.
func tryGetReadOnlyDatabaseDriver(ctx context.Context, instance *api.Instance, databaseName string, logger *zap.Logger) (db.Driver, error) {
	dataSource := api.DataSourceFromInstanceWithType(instance, api.RO)
	// If there are no read-only data source, fall back to admin data source.
	if dataSource == nil {
		dataSource = api.DataSourceFromInstanceWithType(instance, api.Admin)
	}
	if dataSource == nil {
		return nil, common.Errorf(common.Internal, fmt.Errorf("data source not found for instance %d", instance.ID))
	}

	driver, err := getDatabaseDriver(
		ctx,
		instance.Engine,
		db.DriverConfig{Logger: logger},
		db.ConnectionConfig{
			Username: dataSource.Username,
			Password: dataSource.Password,
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
		return nil, common.Errorf(common.DbConnectionFailure, fmt.Errorf("failed to connect database at %s:%s with user %q: %w", connectionConfig.Host, connectionConfig.Port, connectionConfig.Username, err))
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
