package server

import (
	"context"
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

func (s *Server) registerInstanceRoutes(g *echo.Group) {
	// Besides adding the instance to Bytebase, it will also try to create a "bytebase" db in the newly added instance.
	g.POST("/instance", func(c echo.Context) error {
		ctx := context.Background()
		if err := s.instanceCountGuard(ctx); err != nil {
			return err
		}

		instanceCreate := &api.InstanceCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, instanceCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create instance request").SetInternal(err)
		}

		instanceCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)

		instance, err := s.InstanceService.CreateInstance(ctx, instanceCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Instance name already exists: %s", instanceCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create instance").SetInternal(err)
		}

		if err := s.composeInstanceRelationship(ctx, instance); err != nil {
			return err
		}

		// Try creating the "bytebase" db in the added instance if needed.
		// Since we allow user to add new instance upfront even providing the incorrect username/password,
		// thus it's OK if it fails. Frontend will surface relevant info suggesting the "bytebase" db hasn't created yet.
		db, err := getAdminDatabaseDriver(ctx, instance, "", s.l)
		if err == nil {
			defer db.Close(ctx)
			if err := db.SetupMigrationIfNeeded(ctx); err != nil {
				s.l.Warn("Failed to setup migration schema on instance creation",
					zap.String("instance_name", instance.Name),
					zap.String("engine", string(instance.Engine)),
					zap.Error(err))
			}
			// Try immediately sync the engine version and schema after instance creation.
			s.syncEngineVersionAndSchema(ctx, instance)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, instance); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create instance response").SetInternal(err)
		}
		return nil
	})

	g.GET("/instance", func(c echo.Context) error {
		ctx := context.Background()
		instanceFind := &api.InstanceFind{}
		if rowStatusStr := c.QueryParam("rowstatus"); rowStatusStr != "" {
			rowStatus := api.RowStatus(rowStatusStr)
			instanceFind.RowStatus = &rowStatus
		}
		list, err := s.InstanceService.FindInstanceList(ctx, instanceFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch instance list").SetInternal(err)
		}

		for _, instance := range list {
			if err := s.composeInstanceRelationship(ctx, instance); err != nil {
				return err
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal instance list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceID", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		instance, err := s.composeInstanceByID(ctx, id)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, instance); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal instance ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/instance/:instanceID", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		instancePatch := &api.InstancePatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, instancePatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch instance request").SetInternal(err)
		}

		var instance *api.Instance
		if instancePatch.RowStatus != nil || instancePatch.Name != nil || instancePatch.ExternalLink != nil || instancePatch.Host != nil || instancePatch.Port != nil {
			// Users can switch instance status from ARCHIVED to NORMAL.
			// So we need to check the current instance count with NORMAL status for quota limitation.
			if instancePatch.RowStatus != nil && *instancePatch.RowStatus == api.Normal.String() {
				if err := s.instanceCountGuard(ctx); err != nil {
					return err
				}
			}
			instance, err = s.InstanceService.PatchInstance(ctx, instancePatch)
			if err != nil {
				if common.ErrorCode(err) == common.NotFound {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch instance ID: %v", id)).SetInternal(err)
			}
		}

		// TODO(d): remove this once UI fully switched to data source API.
		if instancePatch.Username != nil || instancePatch.Password != nil || instancePatch.UseEmptyPassword {
			instanceFind := &api.InstanceFind{
				ID: &id,
			}
			instance, err = s.InstanceService.FindInstance(ctx, instanceFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
			}
			if instance == nil {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
			}

			dataSourceType := api.Admin
			dataSourceFind := &api.DataSourceFind{
				InstanceID: &instance.ID,
				Type:       &dataSourceType,
			}
			adminDataSource, err := s.DataSourceService.FindDataSource(ctx, dataSourceFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch data source for instance: %v", instance.Name)).SetInternal(err)
			}
			if adminDataSource == nil {
				err := fmt.Errorf("data source not found for instance ID %v, name %q and type %q", instance.ID, instance.Name, dataSourceType)
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error()).SetInternal(err)
			}

			dataSourcePatch := &api.DataSourcePatch{
				ID:        adminDataSource.ID,
				UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
				Username:  instancePatch.Username,
			}
			if instancePatch.Password != nil {
				dataSourcePatch.Password = instancePatch.Password
			} else if instancePatch.UseEmptyPassword {
				password := ""
				dataSourcePatch.Password = &password
			}
			_, err = s.DataSourceService.PatchDataSource(ctx, dataSourcePatch)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch data source for instance: %v", instance.Name)).SetInternal(err)
			}
		}

		if err := s.composeInstanceRelationship(ctx, instance); err != nil {
			return err
		}

		// Try immediately setup the migration schema, sync the engine version and schema after updating any connection related info.
		if instancePatch.Host != nil || instancePatch.Port != nil {
			db, err := getAdminDatabaseDriver(ctx, instance, "", s.l)
			if err == nil {
				defer db.Close(ctx)
				if err := db.SetupMigrationIfNeeded(ctx); err != nil {
					s.l.Warn("Failed to setup migration schema on instance update",
						zap.String("instance_name", instance.Name),
						zap.String("engine", string(instance.Engine)),
						zap.Error(err))
				}
				s.syncEngineVersionAndSchema(ctx, instance)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, instance); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal instance ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceID/user", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		instanceUserFind := &api.InstanceUserFind{
			InstanceID: id,
		}
		list, err := s.InstanceUserService.FindInstanceUserList(ctx, instanceUserFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch user list for instance: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal instance user list response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.POST("/instance/:instanceID/migration", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		instance, err := s.composeInstanceByID(ctx, id)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}

		resultSet := &api.SQLResultSet{}
		db, err := getAdminDatabaseDriver(ctx, instance, "", s.l)
		if err != nil {
			resultSet.Error = err.Error()
		} else {
			defer db.Close(ctx)
			if err := db.SetupMigrationIfNeeded(ctx); err != nil {
				resultSet.Error = err.Error()
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, resultSet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration setup status response for host:port: %v:%v", instance.Host, instance.Port)).SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceID/migration/status", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		instance, err := s.composeInstanceByID(ctx, id)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}

		instanceMigration := &api.InstanceMigration{}
		db, err := getAdminDatabaseDriver(ctx, instance, "", s.l)
		if err != nil {
			instanceMigration.Status = api.InstanceMigrationSchemaUnknown
			instanceMigration.Error = err.Error()
		} else {
			defer db.Close(ctx)
			setup, err := db.NeedsSetupMigration(ctx)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to check migration setup status for host:port: %v:%v", instance.Host, instance.Port)).SetInternal(err)
			}
			if setup {
				instanceMigration.Status = api.InstanceMigrationSchemaNotExist
			} else {
				instanceMigration.Status = api.InstanceMigrationSchemaOK
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, instanceMigration); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration setup status response for host:port: %v:%v", instance.Host, instance.Port)).SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceID/migration/history/:historyID", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Instance ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		historyID, err := strconv.Atoi(c.Param("historyID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("History ID is not a number: %s", c.Param("historyID"))).SetInternal(err)
		}

		instance, err := s.composeInstanceByID(ctx, id)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}

		find := &db.MigrationHistoryFind{ID: &historyID}
		driver, err := getAdminDatabaseDriver(ctx, instance, "", s.l)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch migration history ID %d for instance %q", id, instance.Name)).SetInternal(err)
		}
		defer driver.Close(ctx)
		list, err := driver.FindMigrationHistoryList(ctx, find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch migration history list").SetInternal(err)
		}
		if len(list) == 0 {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Migration history ID %d not found for instance %q", historyID, instance.Name))
		}
		entry := list[0]

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, &api.MigrationHistory{
			ID:                  entry.ID,
			Creator:             entry.Creator,
			CreatedTs:           entry.CreatedTs,
			Updater:             entry.Updater,
			UpdatedTs:           entry.UpdatedTs,
			ReleaseVersion:      entry.ReleaseVersion,
			Database:            entry.Namespace,
			Source:              entry.Source,
			Type:                entry.Type,
			Status:              entry.Status,
			Version:             entry.Version,
			Description:         entry.Description,
			Statement:           entry.Statement,
			Schema:              entry.Schema,
			SchemaPrev:          entry.SchemaPrev,
			ExecutionDurationNs: entry.ExecutionDurationNs,
			IssueID:             entry.IssueID,
			Payload:             entry.Payload,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration history response for instance: %v", instance.Name)).SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceID/migration/history", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		instance, err := s.composeInstanceByID(ctx, id)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}

		find := &db.MigrationHistoryFind{}
		databaseStr := c.QueryParams().Get("database")
		if databaseStr != "" {
			find.Database = &databaseStr
		}
		versionStr := c.QueryParams().Get("version")
		if versionStr != "" {
			find.Version = &versionStr
		}
		if limitStr := c.QueryParam("limit"); limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("limit query parameter is not a number: %s", limitStr)).SetInternal(err)
			}
			find.Limit = &limit
		}

		historyList := []*api.MigrationHistory{}
		driver, err := getAdminDatabaseDriver(ctx, instance, "", s.l)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch migration history for instance %q", instance.Name)).SetInternal(err)
		}
		defer driver.Close(ctx)
		list, err := driver.FindMigrationHistoryList(ctx, find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch migration history list").SetInternal(err)
		}

		for _, entry := range list {
			historyList = append(historyList, &api.MigrationHistory{
				ID:                  entry.ID,
				Creator:             entry.Creator,
				CreatedTs:           entry.CreatedTs,
				Updater:             entry.Updater,
				UpdatedTs:           entry.UpdatedTs,
				ReleaseVersion:      entry.ReleaseVersion,
				Database:            entry.Namespace,
				Source:              entry.Source,
				Type:                entry.Type,
				Status:              entry.Status,
				Version:             entry.Version,
				Description:         entry.Description,
				Statement:           entry.Statement,
				Schema:              entry.Schema,
				SchemaPrev:          entry.SchemaPrev,
				ExecutionDurationNs: entry.ExecutionDurationNs,
				IssueID:             entry.IssueID,
				Payload:             entry.Payload,
			})
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, historyList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration history response for instance: %v", instance.Name)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) composeInstanceByID(ctx context.Context, id int) (*api.Instance, error) {
	instanceFind := &api.InstanceFind{
		ID: &id,
	}
	instance, err := s.InstanceService.FindInstance(ctx, instanceFind)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("instance ID not found %v", id)}
	}

	if err := s.composeInstanceRelationship(ctx, instance); err != nil {
		return nil, err
	}

	return instance, nil
}

func (s *Server) composeInstanceRelationship(ctx context.Context, instance *api.Instance) error {
	var err error

	instance.Creator, err = s.composePrincipalByID(ctx, instance.CreatorID)
	if err != nil {
		return err
	}

	instance.Updater, err = s.composePrincipalByID(ctx, instance.UpdaterID)
	if err != nil {
		return err
	}

	instance.Environment, err = s.composeEnvironmentByID(ctx, instance.EnvironmentID)
	if err != nil {
		return err
	}

	rowStatus := api.Normal
	instance.AnomalyList, err = s.AnomalyService.FindAnomalyList(ctx, &api.AnomalyFind{
		RowStatus:    &rowStatus,
		InstanceID:   &instance.ID,
		InstanceOnly: true,
	})
	if err != nil {
		return err
	}
	for _, anomaly := range instance.AnomalyList {
		if anomaly.Creator, err = s.composePrincipalByID(ctx, anomaly.CreatorID); err != nil {
			return err
		}
		if anomaly.Updater, err = s.composePrincipalByID(ctx, anomaly.UpdaterID); err != nil {
			return err
		}
	}

	instance.DataSourceList, err = s.DataSourceService.FindDataSourceList(ctx, &api.DataSourceFind{
		InstanceID: &instance.ID,
	})
	if err != nil {
		return err
	}
	for _, dataSource := range instance.DataSourceList {
		if dataSource.Creator, err = s.composePrincipalByID(ctx, dataSource.CreatorID); err != nil {
			return err
		}
		if dataSource.Updater, err = s.composePrincipalByID(ctx, dataSource.UpdaterID); err != nil {
			return err
		}

		// TODO(d): remove this when UI is fully switched to data source API.
		if dataSource.Type == api.Admin {
			instance.Username = dataSource.Username
			instance.Password = dataSource.Password
		}
	}

	return nil
}

func (s *Server) findInstanceAdminPasswordByID(ctx context.Context, instanceID int) (string, error) {
	dataSourceFind := &api.DataSourceFind{
		InstanceID: &instanceID,
	}
	dataSourceList, err := s.DataSourceService.FindDataSourceList(ctx, dataSourceFind)
	if err != nil {
		return "", err
	}
	for _, dataSource := range dataSourceList {
		if dataSource.Type == api.Admin {
			return dataSource.Password, nil
		}
	}
	return "", &common.Error{Code: common.NotFound, Err: fmt.Errorf("missing admin password for instance: %d", instanceID)}
}

// instanceCountGuard is a feature guard for instance count.
// We only count instances with NORMAL status since users cannot make any operations for ARCHIVED one.
func (s *Server) instanceCountGuard(ctx context.Context) error {
	status := api.Normal
	count, err := s.InstanceService.CountInstance(ctx, &api.InstanceFind{
		RowStatus: &status,
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count instance").SetInternal(err)
	}
	subscription := s.loadSubscription()
	if count >= subscription.InstanceCount {
		return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("You have reached the maximum instance count %d.", subscription.InstanceCount))
	}

	return nil
}
