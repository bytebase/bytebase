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
)

func (s *Server) registerInstanceRoutes(g *echo.Group) {
	// Besides adding the instance to Bytebase, it will also try to create a "bytebase" db in the newly added instance.
	g.POST("/instance", func(c echo.Context) error {
		instanceCreate := &api.InstanceCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, instanceCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create instance request").SetInternal(err)
		}

		instanceCreate.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)

		instance, err := s.InstanceService.CreateInstance(context.Background(), instanceCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Instance name already exists: %s", instanceCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create instance").SetInternal(err)
		}

		if err := s.ComposeInstanceRelationship(context.Background(), instance); err != nil {
			return err
		}

		// Try creating the "bytebase" db in the added instance if needed.
		// Since we allow user to add new instance upfront even providing the incorrect username/password,
		// thus it's OK if it fails. Frontend will surface relavant info suggesting the "bytebase" db hasn't created yet.
		db, err := GetDatabaseDriver(instance, "", s.l)
		if err == nil {
			defer db.Close(context.Background())
			db.SetupMigrationIfNeeded(context.Background())
		}

		// Try immediately sync the schema after instance creation.
		s.SyncSchema(instance)

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, instance); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create instance response").SetInternal(err)
		}
		return nil
	})

	g.GET("/instance", func(c echo.Context) error {
		instanceFind := &api.InstanceFind{}
		if rowStatusStr := c.QueryParam("rowstatus"); rowStatusStr != "" {
			rowStatus := api.RowStatus(rowStatusStr)
			instanceFind.RowStatus = &rowStatus
		}
		list, err := s.InstanceService.FindInstanceList(context.Background(), instanceFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch instance list").SetInternal(err)
		}

		for _, instance := range list {
			if err := s.ComposeInstanceRelationship(context.Background(), instance); err != nil {
				return err
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal instance list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("instanceId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceId"))).SetInternal(err)
		}

		instance, err := s.ComposeInstanceById(context.Background(), id)
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

	g.PATCH("/instance/:instanceId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("instanceId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceId"))).SetInternal(err)
		}

		instancePatch := &api.InstancePatch{
			ID:        id,
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, instancePatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted patch instance request").SetInternal(err)
		}

		var instance *api.Instance
		if instancePatch.RowStatus != nil || instancePatch.Name != nil || instancePatch.ExternalLink != nil || instancePatch.Host != nil || instancePatch.Port != nil {
			instance, err = s.InstanceService.PatchInstance(context.Background(), instancePatch)
			if err != nil {
				if common.ErrorCode(err) == common.NotFound {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch instance ID: %v", id)).SetInternal(err)
			}
		}

		if instancePatch.Username != nil || instancePatch.Password != nil {
			instanceFind := &api.InstanceFind{
				ID: &id,
			}
			instance, err = s.InstanceService.FindInstance(context.Background(), instanceFind)
			if err != nil {
				if common.ErrorCode(err) == common.NotFound {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
			}

			dataSourceType := api.Admin
			dataSourceFind := &api.DataSourceFind{
				InstanceId: &instance.ID,
				Type:       &dataSourceType,
			}
			adminDataSource, err := s.DataSourceService.FindDataSource(context.Background(), dataSourceFind)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch data source for instance: %v", instance.Name)).SetInternal(err)
			}

			dataSourcePatch := &api.DataSourcePatch{
				ID:        adminDataSource.ID,
				UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
				Username:  instancePatch.Username,
				Password:  instancePatch.Password,
			}
			_, err = s.DataSourceService.PatchDataSource(context.Background(), dataSourcePatch)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch data source for instance: %v", instance.Name)).SetInternal(err)
			}
		}

		if err := s.ComposeInstanceRelationship(context.Background(), instance); err != nil {
			return err
		}

		if instancePatch.Host != nil || instancePatch.Port != nil || instancePatch.Username != nil || instancePatch.Password != nil {
			// Try immediately sync the schema after updating any connection related info.
			s.SyncSchema(instance)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, instance); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal instance ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceId/user", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("instanceId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceId"))).SetInternal(err)
		}

		instanceUserFind := &api.InstanceUserFind{
			InstanceId: id,
		}
		list, err := s.InstanceUserService.FindInstanceUserList(context.Background(), instanceUserFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch user list for instance: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal instance user list response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.POST("/instance/:instanceId/migration", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("instanceId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceId"))).SetInternal(err)
		}

		instance, err := s.ComposeInstanceById(context.Background(), id)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}

		resultSet := &api.SqlResultSet{}
		db, err := db.Open(
			instance.Engine,
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
			resultSet.Error = err.Error()
		} else {
			defer db.Close(context.Background())
			if err := db.SetupMigrationIfNeeded(context.Background()); err != nil {
				resultSet.Error = err.Error()
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, resultSet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration setup status response for host:port: %v:%v", instance.Host, instance.Port)).SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceId/migration/status", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("instanceId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceId"))).SetInternal(err)
		}

		instance, err := s.ComposeInstanceById(context.Background(), id)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}

		instanceMigration := &api.InstanceMigration{}
		db, err := GetDatabaseDriver(instance, "", s.l)
		if err != nil {
			instanceMigration.Status = api.InstanceMigrationSchemaUnknown
			instanceMigration.Error = err.Error()
		} else {
			defer db.Close(context.Background())
			setup, err := db.NeedsSetupMigration(context.Background())
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

	g.GET("/instance/:instanceId/migration/history/:historyId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("instanceId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Intance ID is not a number: %s", c.Param("instanceId"))).SetInternal(err)
		}

		historyId, err := strconv.Atoi(c.Param("historyId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("History ID is not a number: %s", c.Param("historyId"))).SetInternal(err)
		}

		instance, err := s.ComposeInstanceById(context.Background(), id)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}

		find := &db.MigrationHistoryFind{ID: &historyId}
		driver, err := GetDatabaseDriver(instance, "", s.l)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch migration history ID %d for instance %q", id, instance.Name)).SetInternal(err)
		}
		defer driver.Close(context.Background())
		list, err := driver.FindMigrationHistoryList(context.Background(), find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch migration history list").SetInternal(err)
		}
		if len(list) == 0 {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Migration history ID %d not found for instance %q", historyId, instance.Name))
		}
		entry := list[0]

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, &api.MigrationHistory{
			ID:                entry.ID,
			Creator:           entry.Creator,
			CreatedTs:         entry.CreatedTs,
			Updater:           entry.Updater,
			UpdatedTs:         entry.UpdatedTs,
			ReleaseVersion:    entry.ReleaseVersion,
			Database:          entry.Namespace,
			Engine:            entry.Engine,
			Type:              entry.Type,
			Status:            entry.Status,
			Version:           entry.Version,
			Description:       entry.Description,
			Statement:         entry.Statement,
			Schema:            entry.Schema,
			SchemaPrev:        entry.SchemaPrev,
			ExecutionDuration: entry.ExecutionDuration,
			IssueId:           entry.IssueId,
			Payload:           entry.Payload,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration history response for instance: %v", instance.Name)).SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceId/migration/history", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("instanceId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceId"))).SetInternal(err)
		}

		instance, err := s.ComposeInstanceById(context.Background(), id)
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
		driver, err := GetDatabaseDriver(instance, "", s.l)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch migration history for instance %q", instance.Name)).SetInternal(err)
		}
		defer driver.Close(context.Background())
		list, err := driver.FindMigrationHistoryList(context.Background(), find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch migration history list").SetInternal(err)
		}

		for _, entry := range list {
			historyList = append(historyList, &api.MigrationHistory{
				ID:                entry.ID,
				Creator:           entry.Creator,
				CreatedTs:         entry.CreatedTs,
				Updater:           entry.Updater,
				UpdatedTs:         entry.UpdatedTs,
				ReleaseVersion:    entry.ReleaseVersion,
				Database:          entry.Namespace,
				Engine:            entry.Engine,
				Type:              entry.Type,
				Status:            entry.Status,
				Version:           entry.Version,
				Description:       entry.Description,
				Statement:         entry.Statement,
				Schema:            entry.Schema,
				SchemaPrev:        entry.SchemaPrev,
				ExecutionDuration: entry.ExecutionDuration,
				IssueId:           entry.IssueId,
				Payload:           entry.Payload,
			})
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, historyList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration history response for instance: %v", instance.Name)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) ComposeInstanceById(ctx context.Context, id int) (*api.Instance, error) {
	instanceFind := &api.InstanceFind{
		ID: &id,
	}
	instance, err := s.InstanceService.FindInstance(context.Background(), instanceFind)
	if err != nil {
		return nil, err
	}

	if err := s.ComposeInstanceRelationship(ctx, instance); err != nil {
		return nil, err
	}

	return instance, nil
}

func (s *Server) ComposeInstanceRelationship(ctx context.Context, instance *api.Instance) error {
	var err error

	instance.Creator, err = s.ComposePrincipalById(context.Background(), instance.CreatorId)
	if err != nil {
		return err
	}

	instance.Updater, err = s.ComposePrincipalById(context.Background(), instance.UpdaterId)
	if err != nil {
		return err
	}

	instance.Environment, err = s.ComposeEnvironmentById(context.Background(), instance.EnvironmentId)
	if err != nil {
		return err
	}

	rowStatus := api.Normal
	instance.AnomalyList, err = s.AnomalyService.FindAnomalyList(context.Background(), &api.AnomalyFind{
		RowStatus:    &rowStatus,
		InstanceId:   &instance.ID,
		InstanceOnly: true,
	})
	if err != nil {
		return err
	}
	for _, anomaly := range instance.AnomalyList {
		anomaly.Creator, err = s.ComposePrincipalById(context.Background(), anomaly.CreatorId)
		if err != nil {
			return err
		}
		anomaly.Updater, err = s.ComposePrincipalById(context.Background(), anomaly.UpdaterId)
		if err != nil {
			return err
		}
	}

	return s.ComposeInstanceAdminDataSource(context.Background(), instance)
}

func (s *Server) ComposeInstanceAdminDataSource(ctx context.Context, instance *api.Instance) error {
	dataSourceFind := &api.DataSourceFind{
		InstanceId: &instance.ID,
	}
	dataSourceList, err := s.DataSourceService.FindDataSourceList(context.Background(), dataSourceFind)
	if err != nil {
		return err
	}
	for _, dataSource := range dataSourceList {
		if dataSource.Type == api.Admin {
			instance.Username = dataSource.Username
			instance.Password = dataSource.Password
			break
		}
	}
	return nil
}

func (s *Server) FindInstanceAdminPasswordById(ctx context.Context, instanceId int) (string, error) {
	dataSourceFind := &api.DataSourceFind{
		InstanceId: &instanceId,
	}
	dataSourceList, err := s.DataSourceService.FindDataSourceList(context.Background(), dataSourceFind)
	if err != nil {
		return "", err
	}
	for _, dataSource := range dataSourceList {
		if dataSource.Type == api.Admin {
			return dataSource.Password, nil
		}
	}
	return "", &common.Error{Code: common.NotFound, Err: fmt.Errorf("missing admin password for instance: %d", instanceId)}
}
