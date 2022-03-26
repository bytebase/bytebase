package server

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/vcs"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

func (s *Server) registerVCSRoutes(g *echo.Group) {
	g.POST("/vcs", func(c echo.Context) error {
		ctx := context.Background()
		vcsCreate := &api.VCSCreate{
			CreatorID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, vcsCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create VCS request").SetInternal(err)
		}
		// Trim ending "/"
		vcsCreate.InstanceURL = strings.TrimRight(vcsCreate.InstanceURL, "/")
		vcsCreate.APIURL = vcs.Get(vcs.GitLabSelfHost, vcs.ProviderConfig{Logger: s.l}).APIURL(vcsCreate.InstanceURL)

		vcsRaw, err := s.VCSService.CreateVCS(ctx, vcsCreate)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create VCS").SetInternal(err)
		}

		vcs, err := s.composeVCSRelationship(ctx, vcsRaw)
		if err != nil {
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
		vcsRawList, err := s.VCSService.FindVCSList(ctx, vcsFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch vcs list").SetInternal(err)
		}

		var vcsList []*api.VCS
		for _, vcsRaw := range vcsRawList {
			vcs, err := s.composeVCSRelationship(ctx, vcsRaw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch vcs relationship: %v", vcs.ID)).SetInternal(err)
			}
			vcsList = append(vcsList, vcs)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, vcsList); err != nil {
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

		vcs, err := s.composeVCSByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch vcs ID: %v", id)).SetInternal(err)
		}
		if vcs == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS ID not found: %d", id))
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
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, vcsPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted change VCS request").SetInternal(err)
		}

		vcsRaw, err := s.VCSService.PatchVCS(ctx, vcsPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change VCS ID: %v", id)).SetInternal(err)
		}

		vcs, err := s.composeVCSRelationship(ctx, vcsRaw)
		if err != nil {
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
			DeleterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := s.VCSService.DeleteVCS(ctx, vcsDelete); err != nil {
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
		repoRawList, err := s.RepositoryService.FindRepositoryList(ctx, repositoryFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository list for vcs ID: %v", id)).SetInternal(err)
		}

		var repoList []*api.Repository
		for _, repoRaw := range repoRawList {
			repo, err := s.composeRepositoryRelationship(ctx, repoRaw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository relationship: %v", repoRaw.ID)).SetInternal(err)
			}
			repoList = append(repoList, repo)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, repoList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal repository list response for vcs ID: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func (s *Server) composeVCSByID(ctx context.Context, id int) (*api.VCS, error) {
	vcsFind := &api.VCSFind{
		ID: &id,
	}
	vcsRaw, err := s.VCSService.FindVCS(ctx, vcsFind)
	if err != nil {
		return nil, err
	}
	if vcsRaw == nil {
		return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("VCS not found with ID %v", id)}
	}

	vcs, err := s.composeVCSRelationship(ctx, vcsRaw)
	if err != nil {
		return nil, err
	}
	return vcs, nil
}

func (s *Server) composeVCSRelationship(ctx context.Context, raw *api.VCSRaw) (*api.VCS, error) {
	vcs := raw.ToVCS()

	creator, err := s.store.GetPrincipalByID(ctx, vcs.CreatorID)
	if err != nil {
		return nil, err
	}
	vcs.Creator = creator

	updater, err := s.store.GetPrincipalByID(ctx, vcs.UpdaterID)
	if err != nil {
		return nil, err
	}
	vcs.Updater = updater

	return vcs, nil
}
