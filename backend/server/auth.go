package server

import (
	"context"
	"net/http"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) registerAuthRoutes(g *echo.Group) {
	g.POST("/auth/login", func(c echo.Context) error {
		login := &api.Login{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, login); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "malformatted login request").SetInternal(err)
		}

		user, err := s.PrincipalService.FindPrincipalByEmail(context.Background(), login.Email)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusUnauthorized).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to authenticate user").SetInternal(err)
		}

		// Compare the stored hashed password, with the hashed version of the password that was received.
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(login.Password)); err != nil {
			// If the two passwords don't match, return a 401 status.
			return echo.NewHTTPError(http.StatusUnauthorized, "incorrect password").SetInternal(err)
		}

		// If password is correct, generate tokens and set cookies.
		if err := GenerateTokensAndSetCookies(user, c); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate access token").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		if err := jsonapi.MarshalPayload(c.Response().Writer, user); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to marshal login response").SetInternal(err)
		}

		return nil
	})

	g.POST("/auth/signup", func(c echo.Context) error {
		signup := &api.Signup{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, signup); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "malformatted signup request").SetInternal(err)
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(signup.Password), bcrypt.DefaultCost)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate password hash").SetInternal(err)
		}

		principalCreate := &api.PrincipalCreate{
			CreatorId:    api.SYSTEM_BOT_ID,
			Status:       api.Active,
			Type:         api.EndUser,
			Name:         signup.Name,
			Email:        signup.Email,
			PasswordHash: string(passwordHash),
		}

		user, err := s.PrincipalService.CreatePrincipal(context.Background(), principalCreate)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict).SetInternal(err)
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to signup user").SetInternal(err)
		}

		if err := GenerateTokensAndSetCookies(user, c); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate access token").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		if err := jsonapi.MarshalPayload(c.Response().Writer, user); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to marshal signup response").SetInternal(err)
		}

		return nil
	})
}
