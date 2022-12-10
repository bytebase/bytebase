package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/common/log"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/resources/postgres"
	"github.com/bytebase/bytebase/store"
)

// pgConnectionInfo represents the embedded postgres instance connection info.
type pgConnectionInfo struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
}

const (
	// instanceNamePattern is the regex to check the instance name.
	// it allows lowercase and number, a single dash ("-") can be used as the word separator.
	// e.g. foo, foo-bar.
	instanceNamePattern = "^([0-9a-z]+-?)+[0-9a-z]+$"

	// instanceNameMinLength is the minimum length for the instance name.
	instanceNameMinLength = 2

	// instanceNameMinLength is the maximum length for the instance name.
	instanceNameMaxLength = 20
)

func (s *Server) registerInstanceRoutes(g *echo.Group) {
	// Besides adding the instance to Bytebase, it will also try to create a "bytebase" db in the newly added instance.
	g.POST("/instance", func(c echo.Context) error {
		ctx := c.Request().Context()

		instanceCreate := &api.InstanceCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, instanceCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create instance request").SetInternal(err)
		}

		instance, err := s.createInstance(ctx, &store.InstanceCreate{
			CreatorID:     c.Get(getPrincipalIDContextKey()).(int),
			EnvironmentID: instanceCreate.EnvironmentID,
			DataSourceList: []*api.DataSourceCreate{
				{
					Name:     api.AdminDataSourceName,
					Type:     api.Admin,
					Username: instanceCreate.Username,
					Password: instanceCreate.Password,
					SslCa:    instanceCreate.SslCa,
					SslCert:  instanceCreate.SslCert,
					SslKey:   instanceCreate.SslKey,
				},
			},
			Name:         instanceCreate.Name,
			Engine:       instanceCreate.Engine,
			ExternalLink: instanceCreate.ExternalLink,
			Host:         instanceCreate.Host,
			Port:         instanceCreate.Port,
			Database:     instanceCreate.Database,
		})
		if err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, instance); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create instance response").SetInternal(err)
		}
		return nil
	})

	g.GET("/instance", func(c echo.Context) error {
		ctx := c.Request().Context()
		instanceFind := &api.InstanceFind{}
		if rowStatusStr := c.QueryParam("rowstatus"); rowStatusStr != "" {
			rowStatus := api.RowStatus(rowStatusStr)
			instanceFind.RowStatus = &rowStatus
		}
		instanceList, err := s.store.FindInstance(ctx, instanceFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch instance list").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, instanceList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal instance list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceID", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		instance, err := s.store.GetInstanceByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, instance); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal instance ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/instance/:instanceID", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		instancePatch := &api.InstancePatch{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, instancePatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch instance request").SetInternal(err)
		}

		instancePatched, err := s.updateInstance(ctx, &store.InstancePatch{
			ID:            id,
			RowStatus:     instancePatch.RowStatus,
			UpdaterID:     c.Get(getPrincipalIDContextKey()).(int),
			Name:          instancePatch.Name,
			EngineVersion: instancePatch.EngineVersion,
			ExternalLink:  instancePatch.ExternalLink,
			Host:          instancePatch.Host,
			Port:          instancePatch.Port,
			Database:      instancePatch.Database,
		})
		if err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, instancePatched); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal instance ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceID/user", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		instanceUserList, err := s.store.FindInstanceUserByInstanceID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch user list for instance: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, instanceUserList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal instance user list response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceID/user/:userID", func(c echo.Context) error {
		ctx := c.Request().Context()
		instanceID, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Instance ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}
		userID, err := strconv.Atoi(c.Param("userID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("User ID is not a number: %s", c.Param("userID"))).SetInternal(err)
		}

		instanceUserFind := &api.InstanceUserFind{
			ID:         &userID,
			InstanceID: &instanceID,
		}
		instanceUser, err := s.store.GetInstanceUser(ctx, instanceUserFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance user with instanceID: %v and userID: %v", instanceID, userID)).SetInternal(err)
		}
		if instanceUser == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("instanceUser not found with instanceID: %v and userID: %v", instanceID, userID))
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, instanceUser); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance user with instanceID: %v and userID: %v", instanceID, userID)).SetInternal(err)
		}
		return nil
	})

	g.POST("/instance/:instanceID/migration", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		instance, err := s.store.GetInstanceByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
		}

		resultSet := &api.SQLResultSet{}
		db, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, "" /* databaseName */)
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
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		instance, err := s.store.GetInstanceByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
		}

		instanceMigration := &api.InstanceMigration{}
		db, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, "" /* databaseName */)
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
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Instance ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		historyID, err := strconv.Atoi(c.Param("historyID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("History ID is not a number: %s", c.Param("historyID"))).SetInternal(err)
		}

		instance, err := s.store.GetInstanceByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
		}

		find := &db.MigrationHistoryFind{ID: &historyID}
		driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, "" /* databaseName */)
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
			ID:                    entry.ID,
			Creator:               entry.Creator,
			CreatedTs:             entry.CreatedTs,
			Updater:               entry.Updater,
			UpdatedTs:             entry.UpdatedTs,
			ReleaseVersion:        entry.ReleaseVersion,
			Database:              entry.Namespace,
			Source:                entry.Source,
			Type:                  entry.Type,
			Status:                entry.Status,
			Version:               entry.Version,
			UseSemanticVersion:    entry.UseSemanticVersion,
			SemanticVersionSuffix: entry.SemanticVersionSuffix,
			Description:           entry.Description,
			Statement:             entry.Statement,
			Schema:                entry.Schema,
			SchemaPrev:            entry.SchemaPrev,
			ExecutionDurationNs:   entry.ExecutionDurationNs,
			IssueID:               entry.IssueID,
			Payload:               entry.Payload,
		}); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration history response for instance: %v", instance.Name)).SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceID/migration/history", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		instance, err := s.store.GetInstanceByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
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
		driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, "" /* databaseName */)
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
				ID:                    entry.ID,
				Creator:               entry.Creator,
				CreatedTs:             entry.CreatedTs,
				Updater:               entry.Updater,
				UpdatedTs:             entry.UpdatedTs,
				ReleaseVersion:        entry.ReleaseVersion,
				Database:              entry.Namespace,
				Source:                entry.Source,
				Type:                  entry.Type,
				Status:                entry.Status,
				Version:               entry.Version,
				UseSemanticVersion:    entry.UseSemanticVersion,
				SemanticVersionSuffix: entry.SemanticVersionSuffix,
				Description:           entry.Description,
				Statement:             entry.Statement,
				Schema:                entry.Schema,
				SchemaPrev:            entry.SchemaPrev,
				ExecutionDurationNs:   entry.ExecutionDurationNs,
				IssueID:               entry.IssueID,
				Payload:               entry.Payload,
			})
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, historyList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration history response for instance: %v", instance.Name)).SetInternal(err)
		}
		return nil
	})

	// Initial and start an embedded postgres instance and there should be only one currently.
	// Its port and dataDir are fixed values.
	g.POST("/instance/new-embedded-pg", func(c echo.Context) error {
		pgUser := "postgres"
		port := 23333
		dataDir := fmt.Sprintf("%s/%s", s.profile.DataDir, "tmp-pgdata")

		// If the data dir does not exist, then we will start a PostgreSQL instance with a fixed port temporarily.
		if _, err := os.Stat(dataDir); os.IsNotExist(err) {
			if err := postgres.InitDB(s.pgBinDir, dataDir, pgUser); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to init embedded postgres database").SetInternal(err)
			}

			if err := postgres.Start(port, s.pgBinDir, dataDir); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to start embedded postgres instance").SetInternal(err)
			}
		}

		return c.JSON(http.StatusOK, pgConnectionInfo{
			Host:     common.GetPostgresSocketDir(),
			Port:     port,
			Username: pgUser,
		})
	})
}

// instanceCountGuard is a feature guard for instance count.
// We only count instances with NORMAL status since users cannot make any operations for ARCHIVED one.
func (s *Server) instanceCountGuard(ctx context.Context) error {
	status := api.Normal
	count, err := s.store.CountInstance(ctx, &api.InstanceFind{
		RowStatus: &status,
	})

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count instance").SetInternal(err)
	}
	subscription := s.licenseService.LoadSubscription(ctx)
	if count >= subscription.InstanceCount {
		return echo.NewHTTPError(http.StatusForbidden, fmt.Sprintf("You have reached the maximum instance count %d.", subscription.InstanceCount))
	}
	return nil
}

// disallowBytebaseStore prevents users adding Bytebase's own Postgres database.
// Otherwise, users can take control of the database which is a security issue.
func (s *Server) disallowBytebaseStore(engine db.Type, host, port string) error {
	// Even when Postgres opens Unix domain socket only for connection, it still requires a port as socket file extension to differentiate different Postgres instances.
	if engine == db.Postgres && port == fmt.Sprintf("%v", s.profile.DatastorePort) && host == common.GetPostgresSocketDir() {
		return errors.Errorf("instance doesn't exist for host %q and port %q", host, port)
	}
	return nil
}

func (s *Server) createInstance(ctx context.Context, create *store.InstanceCreate) (*api.Instance, error) {
	if err := s.instanceCountGuard(ctx); err != nil {
		return nil, err
	}
	if err := s.validateInstanceName(ctx, create.Name); err != nil {
		return nil, err
	}
	if err := s.validateDataSourceList(create.DataSourceList); err != nil {
		return nil, err
	}

	if err := s.disallowBytebaseStore(create.Engine, create.Host, create.Port); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}
	if create.Engine != db.Postgres && create.Database != "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "database parameter is only allowed for Postgres")
	}

	instance, err := s.store.CreateInstance(ctx, create)
	if err != nil {
		if common.ErrorCode(err) == common.Conflict {
			return nil, echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Instance name already exists: %s", create.Name))
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to create instance").SetInternal(err)
	}

	// Try creating the "bytebase" db in the added instance if needed.
	// Since we allow user to add new instance upfront even providing the incorrect username/password,
	// thus it's OK if it fails. Frontend will surface relevant info suggesting the "bytebase" db hasn't created yet.
	db, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, "" /* databaseName */)
	if err == nil {
		defer db.Close(ctx)
		if err := db.SetupMigrationIfNeeded(ctx); err != nil {
			log.Warn("Failed to setup migration schema on instance creation",
				zap.String("instance_name", instance.Name),
				zap.String("engine", string(instance.Engine)),
				zap.Error(err))
		}
		if _, err := s.SchemaSyncer.SyncInstance(ctx, instance); err != nil {
			log.Warn("Failed to sync instance",
				zap.Int("instance_id", instance.ID),
				zap.Error(err))
		}
		// Sync all databases in the instance asynchronously.
		s.stateCfg.InstanceDatabaseSyncChan <- instance
	}

	return instance, nil
}

func (s *Server) updateInstance(ctx context.Context, patch *store.InstancePatch) (*api.Instance, error) {
	instance, err := s.store.GetInstanceByID(ctx, patch.ID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get instance ID: %v", patch.ID)).SetInternal(err)
	}
	if instance == nil {
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", patch.ID))
	}

	if v := patch.Name; v != nil {
		if err := s.validateInstanceName(ctx, *v); err != nil {
			return nil, err
		}
	}
	if v := patch.DataSourceList; v != nil {
		if err := s.validateDataSourceList(v); err != nil {
			return nil, err
		}
	}

	host, port, database := instance.Host, instance.Port, instance.Database
	if patch.Host != nil {
		host = *patch.Host
	}
	if patch.Port != nil {
		port = *patch.Port
	}
	if patch.Database != nil {
		database = *patch.Database
	}
	if err := s.disallowBytebaseStore(instance.Engine, host, port); err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
	}
	if instance.Engine != db.Postgres && database != "" {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "database parameter is only allowed for Postgres")
	}

	instancePatched := instance
	if patch.RowStatus != nil || patch.Name != nil || patch.ExternalLink != nil || patch.Host != nil || patch.Port != nil || patch.Database != nil || patch.DataSourceList != nil {
		// Users can switch instance status from ARCHIVED to NORMAL.
		// So we need to check the current instance count with NORMAL status for quota limitation.
		if patch.RowStatus != nil && *patch.RowStatus == string(api.Normal) {
			if err := s.instanceCountGuard(ctx); err != nil {
				return nil, err
			}
		}
		// Ensure all databases belong to this instance are under the default project before instance is archived.
		if v := patch.RowStatus; v != nil && *v == string(api.Archived) {
			databases, err := s.store.FindDatabase(ctx, &api.DatabaseFind{InstanceID: &patch.ID})
			if err != nil {
				return nil, echo.NewHTTPError(http.StatusInternalServerError,
					errors.Errorf("failed to find databases in the instance %d", patch.ID)).SetInternal(err)
			}
			var databaseNameList []string
			for _, database := range databases {
				if database.ProjectID != api.DefaultProjectID {
					databaseNameList = append(databaseNameList, database.Name)
				}
			}
			if len(databaseNameList) > 0 {
				return nil, echo.NewHTTPError(http.StatusBadRequest,
					fmt.Sprintf("You should transfer these databases to the default project before archiving the instance: %s.", strings.Join(databaseNameList, ", ")))
			}
		}
		instancePatched, err = s.store.PatchInstance(ctx, patch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", patch.ID))
			}
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch instance ID: %v", patch.ID)).SetInternal(err)
		}
	}

	// Try immediately setup the migration schema, sync the engine version and schema after updating any connection related info.
	if patch.Host != nil || patch.Port != nil || patch.Database != nil {
		db, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instancePatched, "" /* databaseName */)
		if err == nil {
			defer db.Close(ctx)
			if err := db.SetupMigrationIfNeeded(ctx); err != nil {
				log.Warn("Failed to setup migration schema on instance update",
					zap.String("instance_name", instancePatched.Name),
					zap.String("engine", string(instancePatched.Engine)),
					zap.Error(err))
			}
			if _, err := s.SchemaSyncer.SyncInstance(ctx, instancePatched); err != nil {
				log.Warn("Failed to sync instance",
					zap.Int("instance_id", instancePatched.ID),
					zap.Error(err))
			}
			// Sync all databases in the instance asynchronously.
			s.stateCfg.InstanceDatabaseSyncChan <- instancePatched
		}
	}

	return instancePatched, nil
}

func (s *Server) validateInstanceName(ctx context.Context, instanceName string) error {
	if len(instanceName) < instanceNameMinLength || len(instanceName) > instanceNameMaxLength {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid length for the instance name %s, it should in %d-%d length", instanceName, instanceNameMinLength, instanceNameMaxLength))
	}

	pattern := regexp.MustCompile(instanceNamePattern)
	if !pattern.MatchString(instanceName) {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("The instance name \"%s\" doesn't matches the naming pattern, it should only contains lowercase and number, and a single dash (\"-\") can be used as the word separator", instanceName))
	}

	count, err := s.store.CountInstance(ctx, &api.InstanceFind{
		Name: &instanceName,
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count the instance by name").SetInternal(err)
	}

	if count > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Duplicate instance name %s", instanceName))
	}

	return nil
}

func (*Server) validateDataSourceList(dataSourceList []*api.DataSourceCreate) error {
	dataSourceMap := map[api.DataSourceType]bool{}
	for _, dataSource := range dataSourceList {
		if dataSourceMap[dataSource.Type] {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Duplicate data source type %s", dataSource.Type))
		}
		dataSourceMap[dataSource.Type] = true
	}
	if !dataSourceMap[api.Admin] {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Missing required data source type %s", api.Admin))
	}

	return nil
}
