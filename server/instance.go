package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/db"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerInstanceRoutes(g *echo.Group) {
	g.POST("/instance", func(c echo.Context) error {
		instanceCreate := &api.InstanceCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, instanceCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create instance request").SetInternal(err)
		}

		instanceCreate.CreatorId = c.Get(GetPrincipalIdContextKey()).(int)

		instance, err := s.InstanceService.CreateInstance(context.Background(), instanceCreate)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Instance name already exists: %s", instanceCreate.Name))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create instance").SetInternal(err)
		}

		if err := s.ComposeInstanceRelationship(context.Background(), instance); err != nil {
			return err
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
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
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
				if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
				}
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch instance ID: %v", id)).SetInternal(err)
			}
		} else if instancePatch.Username != nil || instancePatch.Password != nil {
			instanceFind := &api.InstanceFind{
				ID: &id,
			}
			instance, err = s.InstanceService.FindInstance(context.Background(), instanceFind)
			if err != nil {
				if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
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
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}

		resultSet := &api.SqlResultSet{}
		db, err := db.Open(instance.Engine, db.DriverConfig{Logger: s.l}, db.ConnectionConfig{
			Username: instance.Username,
			Password: instance.Password,
			Host:     instance.Host,
			Port:     instance.Port,
		})
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
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}

		instanceMigration := &api.InstanceMigration{}
		db, err := db.Open(instance.Engine, db.DriverConfig{Logger: s.l}, db.ConnectionConfig{
			Username: instance.Username,
			Password: instance.Password,
			Host:     instance.Host,
			Port:     instance.Port,
		})
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

	g.GET("/instance/:instanceId/migration/history", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("instanceId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceId"))).SetInternal(err)
		}

		instance, err := s.ComposeInstanceById(context.Background(), id)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}

		find := &db.MigrationHistoryFind{}
		databaseStr := c.QueryParams().Get("database")
		if databaseStr != "" {
			find.Database = &databaseStr
		}
		if limitStr := c.QueryParam("limit"); limitStr != "" {
			limit, err := strconv.Atoi(limitStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("limit query parameter is not a number: %s", limitStr)).SetInternal(err)
			}
			find.Limit = &limit
		}

		historyList := []*api.MigrationHistory{}
		driver, err := db.Open(instance.Engine, db.DriverConfig{Logger: s.l}, db.ConnectionConfig{
			Username: instance.Username,
			Password: instance.Password,
			Host:     instance.Host,
			Port:     instance.Port,
		})
		if err == nil {
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
					Database:          entry.Namespace,
					Engine:            entry.Engine,
					Type:              entry.Type,
					Version:           entry.Version,
					Description:       entry.Description,
					Statement:         entry.Statement,
					ExecutionDuration: entry.ExecutionDuration,
					IssueId:           entry.IssueId,
					Payload:           entry.Payload,
				})
			}
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
	return "", &bytebase.Error{Code: bytebase.ENOTFOUND, Message: fmt.Sprintf("missing admin password for instance: %d", instanceId)}
}
