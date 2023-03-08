package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser"
	"github.com/bytebase/bytebase/backend/plugin/parser/transform"
	"github.com/bytebase/bytebase/backend/resources/postgres"
	"github.com/bytebase/bytebase/backend/store"
)

// pgConnectionInfo represents the embedded postgres instance connection info.
type pgConnectionInfo struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
}

func (s *Server) registerInstanceRoutes(g *echo.Group) {
	// Besides adding the instance to Bytebase, it will also try to create a "bytebase" db in the newly added instance.
	g.POST("/instance", func(c echo.Context) error {
		ctx := c.Request().Context()

		if err := s.instanceCountGuard(ctx); err != nil {
			return err
		}

		instanceCreate := &api.InstanceCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, instanceCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create instance request").SetInternal(err)
		}

		if err := s.disallowBytebaseStore(instanceCreate.Engine, instanceCreate.Host, instanceCreate.Port); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error()).SetInternal(err)
		}
		if !isValidResourceID(instanceCreate.ResourceID) {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid instance id %s", instanceCreate.ResourceID))
		}
		if instanceCreate.Engine != db.Postgres && instanceCreate.Engine != db.MongoDB && instanceCreate.Database != "" {
			return echo.NewHTTPError(http.StatusBadRequest, "database parameter is only allowed for Postgres and MongoDB")
		}
		environment, err := s.store.GetEnvironmentByID(ctx, instanceCreate.EnvironmentID)
		if err != nil {
			return err
		}
		if environment == nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("environment %v not found", instanceCreate.EnvironmentID))
		}
		creator := c.Get(getPrincipalIDContextKey()).(int)
		instance, err := s.store.CreateInstanceV2(ctx, environment.ResourceID, &store.InstanceMessage{
			ResourceID:   instanceCreate.ResourceID,
			Title:        instanceCreate.Name,
			Engine:       instanceCreate.Engine,
			ExternalLink: instanceCreate.ExternalLink,
			DataSources: []*store.DataSourceMessage{
				{
					Title:                  api.AdminDataSourceName,
					Type:                   api.Admin,
					Username:               instanceCreate.Username,
					ObfuscatedPassword:     common.Obfuscate(instanceCreate.Password, s.secret),
					ObfuscatedSslCa:        common.Obfuscate(instanceCreate.SslCa, s.secret),
					ObfuscatedSslCert:      common.Obfuscate(instanceCreate.SslCert, s.secret),
					ObfuscatedSslKey:       common.Obfuscate(instanceCreate.SslKey, s.secret),
					Host:                   instanceCreate.Host,
					Port:                   instanceCreate.Port,
					Database:               instanceCreate.Database,
					SRV:                    instanceCreate.SRV,
					AuthenticationDatabase: instanceCreate.AuthenticationDatabase,
				},
			},
		}, creator)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create instance").SetInternal(err)
		}
		composedInstance, err := s.store.GetInstanceByID(ctx, instance.UID)
		if err != nil {
			return err
		}
		// Try creating the "bytebase" db in the added instance if needed.
		// Since we allow user to add new instance upfront even providing the incorrect username/password,
		// thus it's OK if it fails. Frontend will surface relevant info suggesting the "bytebase" db hasn't created yet.
		driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, "" /* databaseName */)
		if err == nil {
			defer driver.Close(ctx)
			if err := driver.SetupMigrationIfNeeded(ctx); err != nil {
				log.Warn("Failed to setup migration schema on instance creation",
					zap.String("instance", instance.ResourceID),
					zap.String("engine", string(instance.Engine)),
					zap.Error(err))
			}
			if _, err := s.SchemaSyncer.SyncInstance(ctx, instance); err != nil {
				log.Warn("Failed to sync instance",
					zap.String("instance", instance.ResourceID),
					zap.Error(err))
			}
			// Sync all databases in the instance asynchronously.
			s.stateCfg.InstanceDatabaseSyncChan <- composedInstance
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedInstance); err != nil {
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
		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &id})
		if err != nil {
			return err
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("instance %v not found", id))
		}

		patch := &api.InstancePatch{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, patch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch instance request").SetInternal(err)
		}

		var deletes *bool
		if patch.RowStatus != nil {
			if *patch.RowStatus == string(api.Normal) {
				if err := s.instanceCountGuard(ctx); err != nil {
					return err
				}
				f := false
				deletes = &f
			} else if *patch.RowStatus == string(api.Archived) {
				databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{InstanceID: &instance.ResourceID})
				if err != nil {
					return err
				}
				var databaseNameList []string
				for _, database := range databases {
					if database.ProjectID != api.DefaultProjectID {
						databaseNameList = append(databaseNameList, database.DatabaseName)
					}
				}
				if len(databaseNameList) > 0 {
					return echo.NewHTTPError(http.StatusBadRequest,
						fmt.Sprintf("You should transfer these databases to the unassigned project before archiving the instance: %s.", strings.Join(databaseNameList, ", ")))
				}
				f := true
				deletes = &f
			}
		}

		updateMessage := &store.UpdateInstanceMessage{
			Title:         patch.Name,
			ExternalLink:  patch.ExternalLink,
			Delete:        deletes,
			UpdaterID:     c.Get(getPrincipalIDContextKey()).(int),
			EnvironmentID: instance.EnvironmentID,
			ResourceID:    instance.ResourceID,
		}
		if _, err := s.store.UpdateInstanceV2(ctx, updateMessage); err != nil {
			return err
		}

		composedInstance, err := s.store.GetInstanceByID(ctx, instance.UID)
		if err != nil {
			return err
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedInstance); err != nil {
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

		instanceUsers, err := s.store.ListInstanceUsers(ctx, &store.FindInstanceUserMessage{InstanceUID: id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch user list for instance: %v", id)).SetInternal(err)
		}
		var composedInstanceUsers []*api.InstanceUser
		for _, instanceUser := range instanceUsers {
			composedInstanceUsers = append(composedInstanceUsers, &api.InstanceUser{
				ID:         instanceUser.Name,
				InstanceID: id,
				Name:       instanceUser.Name,
				Grant:      instanceUser.Grant,
			})
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedInstanceUsers); err != nil {
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
		userID := c.Param("userID")

		instanceUser, err := s.store.GetInstanceUser(ctx, &store.FindInstanceUserMessage{InstanceUID: instanceID, Name: &userID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance user with instanceID: %v and userID: %v", instanceID, userID)).SetInternal(err)
		}
		if instanceUser == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("instanceUser not found with instanceID: %v and userID: %v", instanceID, userID))
		}
		composedInstanceUser := &api.InstanceUser{
			ID:         instanceUser.Name,
			InstanceID: instanceID,
			Name:       instanceUser.Name,
			Grant:      instanceUser.Grant,
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedInstanceUser); err != nil {
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

		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &id})
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
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration setup status response for instance %q", instance.Title)).SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceID/migration/status", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &id})
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
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to check migration setup status for instance %q", instance.Title)).SetInternal(err)
			}
			if setup {
				instanceMigration.Status = api.InstanceMigrationSchemaNotExist
			} else {
				instanceMigration.Status = api.InstanceMigrationSchemaOK
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, instanceMigration); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration setup status response for instance %q", instance.Title)).SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceID/migration/history/:historyID", func(c echo.Context) error {
		ctx := c.Request().Context()
		instanceID, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Instance ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		historyID := c.Param("historyID")
		isSDL := c.QueryParam("sdl") == "true"

		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &instanceID})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", instanceID)).SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", instanceID))
		}

		var entry *db.MigrationHistory
		if instance.Engine == db.Redis {
			list, err := s.store.FindInstanceChangeHistoryList(ctx, &db.MigrationHistoryFind{
				InstanceID: instanceID,
				ID:         &historyID,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch migration history list").SetInternal(err)
			}
			if len(list) == 0 {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Migration history ID %q not found for instance %q", historyID, instance.Title))
			}
			entry = list[0]
		} else {
			find := &db.MigrationHistoryFind{ID: &historyID}
			driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, "" /* databaseName */)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch migration history ID %v for instance %q", historyID, instance.Title)).SetInternal(err)
			}
			defer driver.Close(ctx)
			list, err := driver.FindMigrationHistoryList(ctx, find)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch migration history list").SetInternal(err)
			}
			if len(list) == 0 {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Migration history ID %q not found for instance %q", historyID, instance.Title))
			}
			entry = list[0]
		}

		if isSDL {
			var engineType parser.EngineType
			switch instance.Engine {
			case db.MySQL:
				engineType = parser.MySQL
			default:
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Not support SDL format for %s instance", instance.Engine))
			}
			entry.Schema, err = transform.SchemaTransform(engineType, entry.Schema)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to transform Schema to SDL format for instance %q", instance.Title)).SetInternal(err)
			}
			entry.SchemaPrev, err = transform.SchemaTransform(engineType, entry.SchemaPrev)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to transform SchemaPrev to SDL format for instance %q", instance.Title)).SetInternal(err)
			}
		}

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
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration history response for instance: %v", instance.Title)).SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceID/migration/history", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("instanceID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("instanceID"))).SetInternal(err)
		}

		instance, err := s.store.GetInstanceV2(ctx, &store.FindInstanceMessage{UID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch instance ID: %v", id)).SetInternal(err)
		}
		if instance == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Instance ID not found: %d", id))
		}

		var migrationHistoryList []*db.MigrationHistory
		if instance.Engine == db.Redis {
			find := &db.MigrationHistoryFind{}
			if databaseStr := c.QueryParams().Get("database"); databaseStr != "" {
				database, err := s.store.GetDatabaseV2(ctx, &store.FindDatabaseMessage{
					InstanceID:   &instance.ResourceID,
					DatabaseName: &databaseStr,
				})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get database %q from instance ID %v", databaseStr, instance.ResourceID)).SetInternal(err)
				}
				if database == nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Cannot find database %q from instance ID %v", databaseStr, instance.ResourceID)).SetInternal(err)
				}
				find.Database = &databaseStr
				find.DatabaseID = &database.UID
			}
			if versionStr := c.QueryParams().Get("version"); versionStr != "" {
				find.Version = &versionStr
			}
			if limitStr := c.QueryParam("limit"); limitStr != "" {
				limit, err := strconv.Atoi(limitStr)
				if err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("limit query parameter is not a number: %s", limitStr)).SetInternal(err)
				}
				find.Limit = &limit
			}

			list, err := s.store.FindInstanceChangeHistoryList(ctx, find)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch migration history list").SetInternal(err)
			}
			migrationHistoryList = list
		} else {
			find := &db.MigrationHistoryFind{}
			if databaseStr := c.QueryParams().Get("database"); databaseStr != "" {
				find.Database = &databaseStr
			}
			if versionStr := c.QueryParams().Get("version"); versionStr != "" {
				find.Version = &versionStr
			}
			if limitStr := c.QueryParam("limit"); limitStr != "" {
				limit, err := strconv.Atoi(limitStr)
				if err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("limit query parameter is not a number: %s", limitStr)).SetInternal(err)
				}
				find.Limit = &limit
			}

			driver, err := s.dbFactory.GetAdminDatabaseDriver(ctx, instance, "" /* databaseName */)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch migration history for instance %q", instance.Title)).SetInternal(err)
			}
			defer driver.Close(ctx)
			list, err := driver.FindMigrationHistoryList(ctx, find)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch migration history list").SetInternal(err)
			}
			migrationHistoryList = list
		}

		historyList := []*api.MigrationHistory{}
		for _, entry := range migrationHistoryList {
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
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration history response for instance: %v", instance.Title)).SetInternal(err)
		}
		return nil
	})

	// Returns the sample embedded postgres instance connection info.
	g.GET("/instance/sample-pg", func(c echo.Context) error {
		return c.JSON(http.StatusOK, pgConnectionInfo{
			Host:     common.GetPostgresSocketDir(),
			Port:     s.profile.SampleDatabasePort,
			Username: postgres.SampleUser,
		})
	})
}

// instanceCountGuard is a feature guard for instance count.
// We only count instances with NORMAL status since users cannot make any operations for ARCHIVED one.
func (s *Server) instanceCountGuard(ctx context.Context) error {
	subscription := s.licenseService.LoadSubscription(ctx)
	if subscription.InstanceCount == -1 {
		return nil
	}

	count, err := s.store.CountInstance(ctx, &store.CountInstanceMessage{})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count instance").SetInternal(err)
	}
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
