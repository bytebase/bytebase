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
	vcsPlugin "github.com/bytebase/bytebase/plugin/vcs"
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

		vcs, err := s.store.CreateVCS(ctx, vcsCreate)
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
		vcsList, err := s.store.FindVCS(ctx, vcsFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch vcs list").SetInternal(err)
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

		vcs, err := s.store.GetVCSByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch vcs ID: %v", id)).SetInternal(err)
		}

		// we do not return secret to the frontend for safety
		vcs.Secret = ""
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

		vcs, err := s.store.PatchVCS(ctx, vcsPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change VCS ID: %v", id)).SetInternal(err)
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
		if err := s.store.DeleteVCS(ctx, vcsDelete); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete VCS with ID[%d]", id)).SetInternal(err)
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

	g.GET("/vcs/:vcsID/external-repository", func(c echo.Context) error {
		ctx := context.Background()
		id, err := strconv.Atoi(c.Param("vcsID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("vcsID"))).SetInternal(err)
		}
		accessToken := c.Request().Header.Get("accessToken")
		refreshToken := c.Request().Header.Get("refreshToken")

		vcsFind := &api.VCSFind{ID: &id}
		vcsFound, err := s.VCSService.FindVCS(ctx, vcsFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch VCS, ID: %v", id)).SetInternal(err)
		}
		if vcsFound == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Failed to find VCS, ID: %v", id))
		}

		externalRepoListByted, err := vcsPlugin.Get(vcsFound.Type, vcsPlugin.ProviderConfig{Logger: s.l}).FetchRepositoryList(
			ctx,
			common.OauthContext{
				ClientID:     vcsFound.ApplicationID,
				ClientSecret: vcsFound.Secret,
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
				Refresher:    nil,
			},
			vcsFound.InstanceURL,
		)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find external repository, instance URL: %s", vcsFound.InstanceURL)).SetInternal(err)
		}
		if externalRepoListByted == nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to find external repository, instance URL: %s", vcsFound.InstanceURL))
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)

		if _, err := c.Response().Writer.Write(externalRepoListByted); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to write response, instance URL: %s", vcsFound.InstanceURL)).SetInternal(err)
		}

		return nil
	})
}
