package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/bytebase/bytebase/api"
)

func (s *Server) registerOpenAPIRoutesForAuth(g *echo.Group) {
	g.POST("/auth/login", s.openAPIUserLogin)
}

func (s *Server) openAPIUserLogin(c echo.Context) error {
	ctx := c.Request().Context()
	login := &api.Login{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Failed to read request body").SetInternal(err)
	}
	if err := json.Unmarshal(body, login); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Malformed login request").SetInternal(err)
	}

	user, err := s.store.GetPrincipalByEmail(ctx, login.Email)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to authenticate user").SetInternal(err)
	}
	if user == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("User not found: %s", login.Email))
	}

	// Compare the stored hashed password, with the hashed version of the password that was received.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(login.Password)); err != nil {
		// If the two passwords don't match, return a 401 status.
		return echo.NewHTTPError(http.StatusUnauthorized, "Incorrect password").SetInternal(err)
	}

	// test the status of this user
	member, err := s.store.GetMemberByPrincipalID(ctx, user.ID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to authenticate user").SetInternal(err)
	}
	if member == nil {
		return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Member not found: %s", user.Email))
	}
	if member.RowStatus == api.Archived {
		return echo.NewHTTPError(http.StatusUnauthorized, "This user has been deactivated by the admin")
	}

	accessToken, err := generateAccessToken(user, s.profile.Mode, s.secret)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate access token").SetInternal(err)
	}

	response := &api.OpenAPIUserLogin{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Token: accessToken,
	}
	return c.JSON(http.StatusOK, response)
}
