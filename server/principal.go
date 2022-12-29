package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// serviceAccountAccessKeyPrefix is the prefix for service account access key.
const serviceAccountAccessKeyPrefix = "bbs_"

func (s *Server) registerPrincipalRoutes(g *echo.Group) {
	g.POST("/principal", func(c echo.Context) error {
		ctx := c.Request().Context()
		principalCreate := &api.PrincipalCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, principalCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create principal request").SetInternal(err)
		}

		principalCreate.CreatorID = c.Get(getPrincipalIDContextKey()).(int)
		principalCreate.Email = strings.ToLower(principalCreate.Email)

		if principalCreate.Type == api.ServiceAccount {
			pwd, err := common.RandomString(20)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate access key for service account.").SetInternal(err)
			}
			principalCreate.Password = fmt.Sprintf("%s%s", serviceAccountAccessKeyPrefix, pwd)
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(principalCreate.Password), bcrypt.DefaultCost)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate password hash").SetInternal(err)
		}
		principalCreate.PasswordHash = string(passwordHash)

		principal, err := s.store.CreatePrincipal(ctx, principalCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, "User already exists")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create principal").SetInternal(err)
		}
		// Assign Developer role to the just created principal
		principal.Role = api.Developer

		// Only return the token if the user is ServiceAccount
		if principal.Type == api.ServiceAccount {
			principal.ServiceKey = principalCreate.Password
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, principal); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create principal response").SetInternal(err)
		}
		return nil
	})

	g.GET("/principal", func(c echo.Context) error {
		ctx := c.Request().Context()
		principalList, err := s.store.GetPrincipalList(ctx)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch principal list").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, principalList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal principal list response").SetInternal(err)
		}
		return nil
	})

	g.GET("/principal/:principalID", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("principalID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("principalID"))).SetInternal(err)
		}

		principal, err := s.store.GetPrincipalByID(ctx, id)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch principal ID: %v", id)).SetInternal(err)
		}
		if principal == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("User ID not found: %d", id))
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, principal); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal principal ID response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.PATCH("/principal/:principalID", func(c echo.Context) error {
		ctx := c.Request().Context()
		id, err := strconv.Atoi(c.Param("principalID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("principalID"))).SetInternal(err)
		}

		principalPatch := &api.PrincipalPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, principalPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch principal request").SetInternal(err)
		}

		if principalPatch.Type == api.ServiceAccount && principalPatch.RefreshKey {
			val, err := common.RandomString(20)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate access key for service account.").SetInternal(err)
			}
			password := fmt.Sprintf("%s%s", serviceAccountAccessKeyPrefix, val)
			principalPatch.Password = &password
		}

		if principalPatch.Password != nil && *principalPatch.Password != "" {
			passwordHash, err := bcrypt.GenerateFromPassword([]byte(*principalPatch.Password), bcrypt.DefaultCost)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate password hash").SetInternal(err)
			}
			passwordHashStr := string(passwordHash)
			principalPatch.PasswordHash = &passwordHashStr
		}

		principal, err := s.store.PatchPrincipal(ctx, principalPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("User ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to patch principal ID: %v", id)).SetInternal(err)
		}

		// Only return the token if the user is ServiceAccount
		if principal.Type == api.ServiceAccount && principalPatch.Password != nil {
			principal.ServiceKey = *principalPatch.Password
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, principal); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal principal ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}
