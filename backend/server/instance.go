package server

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	parser "github.com/bytebase/bytebase/backend/plugin/parser/sql"

	"github.com/bytebase/bytebase/backend/plugin/parser/sql/transform"
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
		// TODO(p0ny): remove this endpoint because we no longer create migration history table on user instances.
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

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, resultSet); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal migration setup status response for instance %q", instance.Title)).SetInternal(err)
		}
		return nil
	})

	g.GET("/instance/:instanceID/migration/status", func(c echo.Context) error {
		// TODO(p0ny): remove this endpoint because we no longer create migration history table on user instances.
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

		instanceMigration := &api.InstanceMigration{
			Status: api.InstanceMigrationSchemaOK,
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
		find := &db.MigrationHistoryFind{ID: &historyID, InstanceID: &instanceID}
		list, err := s.store.FindInstanceChangeHistoryList(ctx, find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch migration history list").SetInternal(err)
		}
		if len(list) == 0 {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Migration history ID %q not found for instance %q", historyID, instance.Title))
		}
		entry = list[0]

		if isSDL {
			var engineType parser.EngineType
			switch instance.Engine {
			case db.MySQL, db.TiDB, db.MariaDB, db.OceanBase:
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

		find := &db.MigrationHistoryFind{
			InstanceID: &instance.UID,
		}
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

		migrationHistoryList, err := s.store.FindInstanceChangeHistoryList(ctx, find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch migration history list").SetInternal(err)
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

// disallowBytebaseStore prevents users adding Bytebase's own Postgres database.
// Otherwise, users can take control of the database which is a security issue.
func (s *Server) disallowBytebaseStore(engine db.Type, host, port string) error {
	// Even when Postgres opens Unix domain socket only for connection, it still requires a port as socket file extension to differentiate different Postgres instances.
	if engine == db.Postgres && port == fmt.Sprintf("%v", s.profile.DatastorePort) && host == common.GetPostgresSocketDir() {
		return errors.Errorf("instance doesn't exist for host %q and port %q", host, port)
	}
	return nil
}
