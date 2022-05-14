package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/vcs"
)

func (s *Server) registerVCSRoutes(g *echo.Group) {
	g.POST("/vcs", func(c echo.Context) error {
		ctx := c.Request().Context()
		vcsCreate := &api.VCSCreate{
			CreatorID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, vcsCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create VCS request").SetInternal(err)
		}
		// Trim ending "/"
		vcsCreate.InstanceURL = strings.TrimRight(vcsCreate.InstanceURL, "/")
		vcsCreate.APIURL = vcs.Get(vcsCreate.Type, vcs.ProviderConfig{Logger: s.l}).APIURL(vcsCreate.InstanceURL)

		vcs, err := s.store.CreateVCS(ctx, vcsCreate)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create VCS").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, vcs); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create VCS response").SetInternal(err)
		}
		return nil
	})

	g.GET("/vcs", func(c echo.Context) error {
		ctx := c.Request().Context()
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
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("vcsID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("vcsID"))).SetInternal(err)
		}

		vcs, err := s.store.GetVCSByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch vcs ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, vcs); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal vcs ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/vcs/:vcsID", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("vcsID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("VCS ID is not a number: %s", c.Param("vcsID"))).SetInternal(err)
		}

		vcsPatch := &api.VCSPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, vcsPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed change VCS request").SetInternal(err)
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
		ctx := c.Request().Context()
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
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("vcsID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("vcsID"))).SetInternal(err)
		}

		repositoryFind := &api.RepositoryFind{
			VCSID: &id,
		}
		repoList, err := s.store.FindRepository(ctx, repositoryFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository list for vcs ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, repoList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal repository list response for vcs ID: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/vcs/:vcsID/external-repository", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("vcsID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("vcsID"))).SetInternal(err)
		}
		accessToken := c.Request().Header.Get("accessToken")
		refreshToken := c.Request().Header.Get("refreshToken")

		vcsFound, err := s.store.GetVCSByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch VCS, ID: %v", id)).SetInternal(err)
		}
		if vcsFound == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Failed to find VCS, ID: %v", id))
		}

		repoList, err := vcs.Get(vcsFound.Type, vcs.ProviderConfig{Logger: s.l}).FetchRepositoryList(
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

		var externalRepoList []*api.ExternalRepository
		for _, repo := range repoList {
			externalRepoList = append(externalRepoList, &api.ExternalRepository{
				ID:       repo.ID,
				FullPath: repo.FullPath,
				Name:     repo.Name,
				WebURL:   repo.WebURL,
			})
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, externalRepoList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal response").SetInternal(err)
		}

		return nil
	})
}
