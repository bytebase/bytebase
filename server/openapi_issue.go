package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (s *Server) registerOpenAPIRoutesForIssue(g *echo.Group) {
	g.POST("issue", s.createIssueByOpenAPI)
}

func (s *Server) createIssueByOpenAPI(c echo.Context) error {
	ctx := c.Request().Context()

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body").SetInternal(err)
	}

	var create v1pb.IssueCreate
	if err := protojson.Unmarshal(body, &create); err != nil {
		return err
	}

	project, err := s.store.GetProjectByKey(ctx, create.ProjectKey)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find project with key %s", create.ProjectKey)).SetInternal(err)
	}

	migrationList := []*api.MigrationDetail{}
	for _, mi := range create.MigrationList {
		dbList, err := s.findProjectDatabases(ctx, project.ID, mi.DatabaseName, mi.EnvironmentName)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to list database").SetInternal(err)
		}

		for _, database := range dbList {
			migrationList = append(migrationList, &api.MigrationDetail{
				DatabaseID:    database.ID,
				MigrationType: db.MigrationType(mi.MigrationType),
				Statement:     mi.Statement,
				SchemaVersion: mi.SchemaVersion,
			})
		}
	}

	createContext, err := json.Marshal(
		&api.MigrationContext{
			DetailList: migrationList,
		},
	)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal update schema context").SetInternal(err)
	}

	issueCreate := &api.IssueCreate{
		CreatorID:             c.Get(getPrincipalIDContextKey()).(int),
		ProjectID:             project.ID,
		Name:                  create.Name,
		Type:                  api.IssueType(create.Type),
		Description:           create.Description,
		AssigneeID:            api.SystemBotID,
		AssigneeNeedAttention: true,
		CreateContext:         string(createContext),
	}

	issue, err := s.createIssue(ctx, issueCreate)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create the issue").SetInternal(err)
	}

	return c.JSON(http.StatusOK, issue)
}
