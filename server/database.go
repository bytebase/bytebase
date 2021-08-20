package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func (s *Server) registerDatabaseRoutes(g *echo.Group) {
	g.POST("/database", func(c echo.Context) error {
		databaseCreate := &api.DatabaseCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, databaseCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create database request").SetInternal(err)
		}

		z, offset := time.Now().Zone()
		databaseCreate.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)
		databaseCreate.TimezoneName = z
		databaseCreate.TimezoneOffset = offset

		database, err := s.DatabaseService.CreateDatabase(context.Background(), databaseCreate)
		if err != nil {
			if common.ErrorCode(err) == common.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Database name already exists: %s", databaseCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create database").SetInternal(err)
		}

		if err := s.ComposeDatabaseRelationship(context.Background(), database); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch created database relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, database); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create database response").SetInternal(err)
		}
		return nil
	})

	g.GET("/database", func(c echo.Context) error {
		databaseFind := &api.DatabaseFind{}
		if instanceIdStr := c.QueryParam("instance"); instanceIdStr != "" {
			instanceId, err := strconv.Atoi(instanceIdStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Query parameter instance is not a number: %s", instanceIdStr)).SetInternal(err)
			}
			databaseFind.InstanceId = &instanceId
		}
		projectIdStr := c.QueryParams().Get("project")
		if projectIdStr != "" {
			projectId, err := strconv.Atoi(projectIdStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("project query parameter is not a number: %s", projectIdStr)).SetInternal(err)
			}
			databaseFind.ProjectId = &projectId
		}
		list, err := s.ComposeDatabaseListByFind(context.Background(), databaseFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch database list").SetInternal(err)
		}

		filteredList := []*api.Database{}
		role := c.Get(GetRoleContextKey()).(api.Role)
		// If caller is NOT requesting for a paritcular instance or the caller is a Developer,
		// then we will only return databases belonging to the project where the caller is a member of.
		// Looking from the UI perspective:
		// - The database list left sidebar will only return databases related to the caller regardless of the caller's role.
		// - The database list on the instance page will return all databases if the caller is Owner or DBA, but will only return
		//   related databases if the caller is Developer.
		if databaseFind.InstanceId == nil || role == api.Developer {
			principalId := c.Get(GetPrincipalIdContextKey()).(int)
			for _, database := range list {
				for _, projectMember := range database.Project.ProjectMemberList {
					if projectMember.PrincipalId == principalId {
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
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		databaseFind := &api.DatabaseFind{
			ID: &id,
		}
		database, err := s.ComposeDatabaseByFind(context.Background(), databaseFind)
		if err != nil {
			if common.ErrorCode(err) == common.ENOTFOUND {
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
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Database ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		databasePatch := &api.DatabasePatch{
			ID:        id,
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, databasePatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch database request").SetInternal(err)
		}

		database, err := s.DatabaseService.PatchDatabase(context.Background(), databasePatch)
		if err != nil {
			if common.ErrorCode(err) == common.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch database ID: %v", id)).SetInternal(err)
		}

		if err := s.ComposeDatabaseRelationship(context.Background(), database); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch updated database relationship: %v", database.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, database); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal database ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:id/table", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		databaseFind := &api.DatabaseFind{
			ID: &id,
		}
		database, err := s.ComposeDatabaseByFind(context.Background(), databaseFind)
		if err != nil {
			if common.ErrorCode(err) == common.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}

		tableFind := &api.TableFind{
			DatabaseId: &id,
		}
		tableList, err := s.TableService.FindTableList(context.Background(), tableFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch table list for database id: %d", id)).SetInternal(err)
		}

		for _, table := range tableList {
			table.Database = database
			columnFind := &api.ColumnFind{
				DatabaseId: &id,
				TableId:    &table.ID,
			}
			table.ColumnList, err = s.ColumnService.FindColumnList(context.Background(), columnFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch colmun list for database id: %d, table name: %s", id, table.Name)).SetInternal(err)
			}

			indexFind := &api.IndexFind{
				DatabaseId: &id,
				TableId:    &table.ID,
			}
			table.IndexList, err = s.IndexService.FindIndexList(context.Background(), indexFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch index list for database id: %d, table name: %s", id, table.Name)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, tableList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal fetch table list response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/database/:id/table/:tableName", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		databaseFind := &api.DatabaseFind{
			ID: &id,
		}
		database, err := s.ComposeDatabaseByFind(context.Background(), databaseFind)
		if err != nil {
			if common.ErrorCode(err) == common.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}

		tableName := c.Param("tableName")
		tableFind := &api.TableFind{
			DatabaseId: &id,
			Name:       &tableName,
		}
		table, err := s.TableService.FindTable(context.Background(), tableFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch table for database id: %d, table name: %s", id, tableName)).SetInternal(err)
		}

		table.Database = database

		columnFind := &api.ColumnFind{
			DatabaseId: &id,
			TableId:    &table.ID,
		}
		table.ColumnList, err = s.ColumnService.FindColumnList(context.Background(), columnFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch colmun list for database id: %d, table name: %s", id, tableName)).SetInternal(err)
		}

		indexFind := &api.IndexFind{
			DatabaseId: &id,
			TableId:    &table.ID,
		}
		table.IndexList, err = s.IndexService.FindIndexList(context.Background(), indexFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch index list for database id: %d, table name: %s", id, table.Name)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, table); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal fetch table response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.POST("/database/:id/backup", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		backupCreate := &api.BackupCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, backupCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create backup request").SetInternal(err)
		}
		backupCreate.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)

		databaseFind := &api.DatabaseFind{
			ID: &id,
		}
		database, err := s.ComposeDatabaseByFind(context.Background(), databaseFind)
		if err != nil {
			if common.ErrorCode(err) == common.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}

		backupCreate.Path, err = getAndCreateBackupPath(s.dataDir, database, backupCreate.Name)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create backup directory for database ID: %v", id)).SetInternal(err)
		}

		version, err := getMigrationVersion(database, s.l)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get migration history for database %q", database.Name)).SetInternal(err)
		}
		backupCreate.MigrationHistoryVersion = version

		backup, err := s.BackupService.CreateBackup(context.Background(), backupCreate)
		if err != nil {
			if common.ErrorCode(err) == common.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Backup name already exists: %s", backupCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create backup").SetInternal(err)
		}
		if err := s.ComposeBackupRelationship(context.Background(), backup); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to compose backup relationship").SetInternal(err)
		}

		payload := api.TaskDatabaseBackupPayload{
			BackupId: backup.ID,
		}
		bytes, err := json.Marshal(payload)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create backup task payload").SetInternal(err)
		}

		createdPipeline, err := s.PipelineService.CreatePipeline(context.Background(), &api.PipelineCreate{
			Name:      fmt.Sprintf("backup-pipeline-%s", backup.Name),
			CreatorId: backupCreate.CreatorId,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create backup pipeline").SetInternal(err)
		}

		createdStage, err := s.StageService.CreateStage(context.Background(), &api.StageCreate{
			Name:          fmt.Sprintf("backup-stage-%s", backup.Name),
			EnvironmentId: database.Instance.EnvironmentId,
			PipelineId:    createdPipeline.ID,
			CreatorId:     backupCreate.CreatorId,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create backup stage").SetInternal(err)
		}

		_, err = s.TaskService.CreateTask(context.Background(), &api.TaskCreate{
			Name:       fmt.Sprintf("backup-task-%s", backup.Name),
			PipelineId: createdPipeline.ID,
			StageId:    createdStage.ID,
			InstanceId: database.InstanceId,
			DatabaseId: &database.ID,
			Status:     api.TaskPending,
			Type:       api.TaskDatabaseBackup,
			Payload:    string(bytes),
			CreatorId:  backupCreate.CreatorId,
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
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		databaseFind := &api.DatabaseFind{
			ID: &id,
		}
		_, err = s.ComposeDatabaseByFind(context.Background(), databaseFind)
		if err != nil {
			if common.ErrorCode(err) == common.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}

		backupFind := &api.BackupFind{
			DatabaseId: &id,
		}
		backupList, err := s.BackupService.FindBackupList(context.Background(), backupFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to backup list for database id: %d", id)).SetInternal(err)
		}

		for _, backup := range backupList {
			if err := s.ComposeBackupRelationship(context.Background(), backup); err != nil {
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
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		backupSettingUpsert := &api.BackupSettingUpsert{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, backupSettingUpsert); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted set backup setting request").SetInternal(err)
		}
		backupSettingUpsert.UpdaterId = c.Get(GetPrincipalIdContextKey()).(int)

		databaseFind := &api.DatabaseFind{
			ID: &id,
		}
		_, err = s.ComposeDatabaseByFind(context.Background(), databaseFind)
		if err != nil {
			if common.ErrorCode(err) == common.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}

		backupSetting, err := s.BackupService.UpsertBackupSetting(context.Background(), backupSettingUpsert)
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
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		databaseFind := &api.DatabaseFind{
			ID: &id,
		}
		_, err = s.ComposeDatabaseByFind(context.Background(), databaseFind)
		if err != nil {
			if common.ErrorCode(err) == common.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Database ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch database ID: %v", id)).SetInternal(err)
		}

		backupSettingFind := &api.BackupSettingFind{
			DatabaseId: &id,
		}
		backupSetting, err := s.BackupService.FindBackupSetting(context.Background(), backupSettingFind)
		if err != nil {
			if common.ErrorCode(err) == common.ENOTFOUND {
				// Returns the backup setting with UNKNOWN_ID to indicate the database has no backup
				backupSetting = &api.BackupSetting{
					ID: api.UNKNOWN_ID,
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

func (s *Server) ComposeDatabaseByFind(ctx context.Context, find *api.DatabaseFind) (*api.Database, error) {
	database, err := s.DatabaseService.FindDatabase(context.Background(), find)
	if err != nil {
		return nil, err
	}

	if err := s.ComposeDatabaseRelationship(ctx, database); err != nil {
		return nil, err
	}

	return database, nil
}

func (s *Server) ComposeDatabaseListByFind(ctx context.Context, find *api.DatabaseFind) ([]*api.Database, error) {
	list, err := s.DatabaseService.FindDatabaseList(context.Background(), find)
	if err != nil {
		return nil, err
	}

	for _, database := range list {
		if err := s.ComposeDatabaseRelationship(context.Background(), database); err != nil {
			return nil, err
		}
	}

	return list, nil
}

func (s *Server) ComposeDatabaseRelationship(ctx context.Context, database *api.Database) error {
	var err error

	database.Creator, err = s.ComposePrincipalById(context.Background(), database.CreatorId)
	if err != nil {
		return err
	}

	database.Updater, err = s.ComposePrincipalById(context.Background(), database.UpdaterId)
	if err != nil {
		return err
	}

	database.Project, err = s.ComposeProjectlById(context.Background(), database.ProjectId)
	if err != nil {
		return err
	}

	database.Instance, err = s.ComposeInstanceById(context.Background(), database.InstanceId)
	if err != nil {
		return err
	}

	if database.SourceBackupId != 0 {
		database.SourceBackup, err = s.ComposeBackupByID(context.Background(), database.SourceBackupId)
		if err != nil {
			return err
		}
	}

	database.DataSourceList = []*api.DataSource{}

	return nil
}

// ComposeBackupByID will compose the backup by backup ID.
func (s *Server) ComposeBackupByID(ctx context.Context, id int) (*api.Backup, error) {
	backupFind := &api.BackupFind{
		ID: &id,
	}
	backup, err := s.BackupService.FindBackup(ctx, backupFind)
	if err != nil {
		return nil, err
	}

	if err := s.ComposeBackupRelationship(ctx, backup); err != nil {
		return nil, err
	}

	return backup, nil
}

// ComposeBackupRelationship will compose the relationship of a backup.
func (s *Server) ComposeBackupRelationship(ctx context.Context, backup *api.Backup) error {
	var err error
	backup.Creator, err = s.ComposePrincipalById(ctx, backup.CreatorId)
	if err != nil {
		return err
	}
	backup.Updater, err = s.ComposePrincipalById(ctx, backup.UpdaterId)
	if err != nil {
		return err
	}
	return nil
}

// Retrieve db.Driver connection.
// Upon successful return, caller MUST call driver.Close, otherwise, it will leak the database connection.
func GetDatabaseDriver(instance *api.Instance, databaseName string, logger *zap.Logger) (db.Driver, error) {
	driver, err := db.Open(
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
		return nil, fmt.Errorf("failed to connect database %s/%s at %q:%q with user %q: %w", instance.Name, databaseName, instance.Host, instance.Port, instance.Username, err)
	}
	return driver, nil
}
