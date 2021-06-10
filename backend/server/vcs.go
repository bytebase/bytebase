package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
)

const (
	apiPath = "api/v4"
)

func (s *Server) registerVCSRoutes(g *echo.Group) {
	g.POST("/vcs", func(c echo.Context) error {
		vcsCreate := &api.VCSCreate{
			CreatorId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, vcsCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create VCS request").SetInternal(err)
		}
		vcsCreate.ApiURL = fmt.Sprintf("%s/%s", vcsCreate.InstanceURL, apiPath)

		vcs, err := s.VCSService.CreateVCS(context.Background(), vcsCreate)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create VCS").SetInternal(err)
		}

		if err := s.ComposeVCSRelationship(context.Background(), vcs, c.Get(getIncludeKey()).([]string)); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch created VCS relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, vcs); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create VCS response").SetInternal(err)
		}
		return nil
	})

	g.GET("/vcs", func(c echo.Context) error {
		vcsFind := &api.VCSFind{}
		list, err := s.VCSService.FindVCSList(context.Background(), vcsFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch vcs list").SetInternal(err)
		}

		for _, vcs := range list {
			if err := s.ComposeVCSRelationship(context.Background(), vcs, c.Get(getIncludeKey()).([]string)); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch vcs relationship: %v", vcs.ID)).SetInternal(err)
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, list); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal vcs list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/vcs/:vcsId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("vcsId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		vcsFind := &api.VCSFind{
			ID: &id,
		}
		vcs, err := s.VCSService.FindVCS(context.Background(), vcsFind)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch vcs ID: %v", id)).SetInternal(err)
		}

		if err := s.ComposeVCSRelationship(context.Background(), vcs, c.Get(getIncludeKey()).([]string)); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch created VCS relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, vcs); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal vcs ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/vcs/:vcsId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("vcsId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("VCS ID is not a number: %s", c.Param("vcsId"))).SetInternal(err)
		}

		vcsPatch := &api.VCSPatch{
			ID:        id,
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, vcsPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted change VCS request").SetInternal(err)
		}

		vcs, err := s.VCSService.PatchVCS(context.Background(), vcsPatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change VCS ID: %v", id)).SetInternal(err)
		}

		if err := s.ComposeVCSRelationship(context.Background(), vcs, c.Get(getIncludeKey()).([]string)); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch updated VCS relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, vcs); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal VCS change response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.DELETE("/vcs/:vcsId", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("vcsId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("VCS is not a number: %s", c.Param("vcsId"))).SetInternal(err)
		}

		vcsDelete := &api.VCSDelete{
			ID:        id,
			DeleterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		err = s.VCSService.DeleteVCS(context.Background(), vcsDelete)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete VCS ID: %v", id)).SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	g.POST("/vcs/:vcsId/token", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("vcsId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		vcsTokenCreate := &api.VCSTokenCreate{
			CreatorId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, vcsTokenCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create VCS token request").SetInternal(err)
		}

		vcsFind := &api.VCSFind{
			ID: &id,
		}
		vcs, err := s.VCSService.FindVCS(context.Background(), vcsFind)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch vcs ID: %v", id)).SetInternal(err)
		}

		oauthToken := &api.OAuthToken{}
		switch vcs.Type {
		case "GITLAB_SELF_HOST":
			url := fmt.Sprintf("%s/oauth/token?client_id=%s&client_secret=%s&code=%s&redirect_uri=%s&grant_type=authorization_code",
				vcs.InstanceURL,
				vcs.ApplicationId,
				vcs.Secret,
				vcsTokenCreate.Code,
				vcsTokenCreate.RedirectUrl,
			)
			req, err := http.NewRequest("POST",
				url, new(bytes.Buffer))
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to construct request to fetch token for vcs ID: %v", id)).SetInternal(err)
			}

			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{}
			res, err := client.Do(req)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch token for vcs ID: %v", id)).SetInternal(err)
			}
			defer res.Body.Close()

			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to read token response for vcs ID: %v", id)).SetInternal(err)
			}

			if err := json.Unmarshal(body, oauthToken); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to unmarshal token response for vcs ID: %v", id)).SetInternal(err)
			}
		}

		vcsPatch := &api.VCSPatch{
			ID:        id,
			UpdaterId: c.Get(GetPrincipalIdContextKey()).(int),
		}
		if oauthToken.AccessToken != "" {
			vcsPatch.AccessToken = &oauthToken.AccessToken
		}
		// For GitLab, as of 13.12, the default config won't expire the access token, thus this field is 0.
		// see https://gitlab.com/gitlab-org/gitlab/-/issues/21745.
		if oauthToken.ExpiresIn != 0 {
			ts := oauthToken.CreatedAt + oauthToken.ExpiresIn
			vcsPatch.ExpireTs = &ts
		}
		if oauthToken.RefreshToken != "" {
			vcsPatch.RefreshToken = &oauthToken.RefreshToken
		}

		updatedVCS, err := s.VCSService.PatchVCS(context.Background(), vcsPatch)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change VCS ID: %v", id)).SetInternal(err)
		}

		if err := s.ComposeVCSRelationship(context.Background(), updatedVCS, c.Get(getIncludeKey()).([]string)); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch updated VCS relationship").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, updatedVCS); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal vcs ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.GET("/vcs/:vcsId/repository", func(c echo.Context) error {
		id, err := strconv.Atoi(c.Param("vcsId"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("id"))).SetInternal(err)
		}

		repositoryFind := &api.RepositoryFind{
			VCSId: &id,
		}
		list, err := s.RepositoryService.FindRepositoryList(context.Background(), repositoryFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository for vcs ID: %v", id)).SetInternal(err)
		}

		for _, repository := range list {
			if err := s.ComposeRepositoryRelationship(context.Background(), repository, c.Get(getIncludeKey()).([]string)); err != nil {
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

func (s *Server) ComposeVCSRelationship(ctx context.Context, vcs *api.VCS, includeList []string) error {
	var err error

	vcs.Creator, err = s.ComposePrincipalById(context.Background(), vcs.CreatorId, includeList)
	if err != nil {
		return err
	}

	vcs.Updater, err = s.ComposePrincipalById(context.Background(), vcs.UpdaterId, includeList)
	if err != nil {
		return err
	}

	return nil
}
