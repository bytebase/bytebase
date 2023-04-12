package server

import (
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	metricAPI "github.com/bytebase/bytebase/backend/metric"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *Server) registerPrincipalRoutes(g *echo.Group) {
	g.POST("/principal", func(c echo.Context) error {
		ctx := c.Request().Context()
		principalCreate := &api.PrincipalCreate{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, principalCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed create principal request").SetInternal(err)
		}
		if err := validateEmail(principalCreate.Email); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid email %q format: %v", principalCreate.Email, err))
		}

		creatorID := c.Get(getPrincipalIDContextKey()).(int)
		password := principalCreate.Password
		if principalCreate.Type == api.ServiceAccount {
			pwd, err := common.RandomString(20)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate access key for service account.").SetInternal(err)
			}
			password = fmt.Sprintf("%s%s", api.ServiceAccountAccessKeyPrefix, pwd)
		}
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate password hash").SetInternal(err)
		}

		users, err := s.store.ListUsers(ctx, &store.FindUserMessage{
			Email:       &principalCreate.Email,
			ShowDeleted: true,
		})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to list users").SetInternal(err)
		}
		if len(users) != 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "Email %s already been occupied", principalCreate.Email)
		}
		user, err := s.store.CreateUser(ctx, &store.UserMessage{
			Email:        principalCreate.Email,
			Name:         principalCreate.Name,
			Type:         principalCreate.Type,
			PasswordHash: string(passwordHash),
		}, creatorID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create principal").SetInternal(err)
		}
		principal := &api.Principal{
			ID:           user.ID,
			Type:         user.Type,
			Name:         user.Name,
			Email:        user.Email,
			PasswordHash: user.PasswordHash,
			Role:         user.Role,
		}
		// Only return the token if the user is ServiceAccount
		if principal.Type == api.ServiceAccount {
			principal.ServiceKey = password
		}

		count, err := s.store.CountUsers(ctx, api.EndUser)
		if err != nil {
			// it's okay to ignore the error to avoid workflow broken.
			log.Debug("failed to count end users", zap.Error(err))
		}
		s.MetricReporter.Report(ctx, &metric.Metric{
			Name:  metricAPI.PrincipalCreateMetricName,
			Value: 1,
			Labels: map[string]any{
				"type": principal.Type,
				"role": principal.Role,
				"rank": count,
			},
		})

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

		principalPatch := &api.PrincipalPatch{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, principalPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformed patch principal request").SetInternal(err)
		}

		if principalPatch.Email != nil {
			if err := validateEmail(*principalPatch.Email); err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("invalid email %q format: %v", *principalPatch.Email, err))
			}
			currentUser, err := s.store.GetUserByID(ctx, id)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to get user by id %d", id)).SetInternal(err)
			}
			if currentUser.Email != *principalPatch.Email {
				users, err := s.store.ListUsers(ctx, &store.FindUserMessage{
					Email:       principalPatch.Email,
					ShowDeleted: true,
				})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get principal list").SetInternal(err)
				}
				if len(users) != 0 {
					return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Email %s is already existed", *principalPatch.Email))
				}
			} else {
				principalPatch.Email = nil
			}
		}

		update := &store.UpdateUserMessage{
			Name:  principalPatch.Name,
			Email: principalPatch.Email,
		}
		newPassword := principalPatch.Password
		if principalPatch.Type == api.ServiceAccount && principalPatch.RefreshKey {
			val, err := common.RandomString(20)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate access key for service account.").SetInternal(err)
			}
			password := fmt.Sprintf("%s%s", api.ServiceAccountAccessKeyPrefix, val)
			newPassword = &password
		}
		if newPassword != nil && *newPassword != "" {
			passwordHash, err := bcrypt.GenerateFromPassword([]byte(*newPassword), bcrypt.DefaultCost)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate password hash").SetInternal(err)
			}
			passwordHashStr := string(passwordHash)
			update.PasswordHash = &passwordHashStr
		}

		updaterID := c.Get(getPrincipalIDContextKey()).(int)
		user, err := s.store.UpdateUser(ctx, id, update, updaterID)
		if err != nil {
			return err
		}
		principal := &api.Principal{
			ID:           user.ID,
			Type:         user.Type,
			Name:         user.Name,
			Email:        user.Email,
			PasswordHash: user.PasswordHash,
			Role:         user.Role,
		}
		// Only return the token if the user is ServiceAccount
		if user.Type == api.ServiceAccount && newPassword != nil {
			principal.ServiceKey = *newPassword
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, principal); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal principal ID response: %v", id)).SetInternal(err)
		}
		return nil
	})
}

func validateEmail(email string) error {
	formatedEmail := strings.ToLower(email)
	if email != formatedEmail {
		return errors.New("email should be lowercase")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return err
	}
	return nil
}
