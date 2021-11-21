package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/external/gitlab"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerVCSRoutes(g *echo.Group) {
	g.POST("/vcs", func(c echo.Context) error {
		ctx := context.Background()
		vcsCreate := &api.VCSCreate{
			CreatorID: c.Get(GetPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, vcsCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create VCS request").SetInternal(err)
		}
		// Trim ending "/"
		vcsCreate.InstanceURL = strings.TrimRight(vcsCreate.InstanceURL, "/")
		vcsCreate.ApiURL = fmt.Sprintf("%s/%s", vcsCreate.InstanceURL, gitlab.ApiPath)

		vcs, err := s.VCSService.CreateVCS(ctx, vcsCreate)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create VCS").SetInternal(err)
		}

		if err := s.ComposeVCSRelationship(ctx, vcs); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch created VCS relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, vcs); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create VCS response").SetInternal(err)
		}
		return nil
	})

	g.GET("/vcs", func(c echo.Context) error {
		ctx := context.Background()
		vcsFind := &api.VCSFind{}
		list, err := s.VCSService.FindVCSList(ctx, vcsFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch vcs list").SetInternal(err)
		}

		for _, vcs := range list {
			if err := s.ComposeVCSRelationship(ctx, vcs); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch vcs relationship: %v", vcs.ID)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal vcs list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/vcs/:vcsID", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("vcsID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("vcsID"))).SetInternal(err)
		}

		vcs, err := s.ComposeVCSByID(ctx, id)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch vcs ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, vcs); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal vcs ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/vcs/:vcsID", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("vcsID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("VCS ID is not a number: %s", c.Param("vcsID"))).SetInternal(err)
		}

		vcsPatch := &api.VCSPatch{
			ID:        id,
			UpdaterID: c.Get(GetPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, vcsPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted change VCS request").SetInternal(err)
		}

		vcs, err := s.VCSService.PatchVCS(ctx, vcsPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change VCS ID: %v", id)).SetInternal(err)
		}

		if err := s.ComposeVCSRelationship(ctx, vcs); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch updated VCS relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, vcs); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal VCS change response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.DELETE("/vcs/:vcsID", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("vcsID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("VCS is not a number: %s", c.Param("vcsID"))).SetInternal(err)
		}

		vcsDelete := &api.VCSDelete{
			ID:        id,
			DeleterID: c.Get(GetPrincipalIDContextKey()).(int),
		}
		err = s.VCSService.DeleteVCS(ctx, vcsDelete)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete VCS ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	g.GET("/vcs/:vcsID/repository", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("vcsID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("vcsID"))).SetInternal(err)
		}

		repositoryFind := &api.RepositoryFind{
			VCSID: &id,
		}
		list, err := s.RepositoryService.FindRepositoryList(ctx, repositoryFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository list for vcs ID: %v", id)).SetInternal(err)
		}

		for _, repository := range list {
			if err := s.ComposeRepositoryRelationship(ctx, repository); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository relationship: %v", repository.ID)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal repository list response for vcs ID: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) ComposeVCSByID(ctx context.Context, id int) (*api.VCS, error) {
	vcsFind := &api.VCSFind{
		ID: &id,
	}
	vcs, err := s.VCSService.FindVCS(ctx, vcsFind)
	if err != nil {
		return nil, err
	}

	if err := s.ComposeVCSRelationship(ctx, vcs); err != nil {
		return nil, err
	}

	return vcs, nil
}

func (s *Server) ComposeVCSRelationship(ctx context.Context, vcs *api.VCS) error {
	var err error

	vcs.Creator, err = s.ComposePrincipalByID(ctx, vcs.CreatorID)
	if err != nil {
		return err
	}

	vcs.Updater, err = s.ComposePrincipalByID(ctx, vcs.UpdaterID)
	if err != nil {
		return err
	}

	return nil
}
