package server

import (
	"context"
	"encoding/json"
	"fmt"
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
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted login request").SetInternal(err)
		}

		principalFind := &api.PrincipalFind{
			Email: &login.Email,
		}
		user, err := s.PrincipalService.FindPrincipal(context.Background(), principalFind)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("User not found: %s", login.Email))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to authenticate user").SetInternal(err)
		}

		memberFind := &api.MemberFind{
			PrincipalId: &user.ID,
		}
		member, err := s.MemberService.FindMember(context.Background(), memberFind)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ENOTFOUND {
				return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("Member not found: %s", login.Email))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to authenticate user").SetInternal(err)
		}
		if member.RowStatus == api.Archived {
			return echo.NewHTTPError(http.StatusUnauthorized, "This user has been deactivated by the admin")
		}

		// Compare the stored hashed password, with the hashed version of the password that was received.
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(login.Password)); err != nil {
			// If the two passwords don't match, return a 401 status.
			return echo.NewHTTPError(http.StatusUnauthorized, "Incorrect password").SetInternal(err)
		}

		// If password is correct, generate tokens and set cookies.
		if err := GenerateTokensAndSetCookies(c, user, s.mode, s.secret); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate access token").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, user); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal login response").SetInternal(err)
		}
		return nil
	})

	g.POST("/auth/logout", func(c echo.Context) error {
		removeTokenCookie(c, accessTokenCookieName)
		removeTokenCookie(c, refreshTokenCookieName)
		removeUserCookie(c)

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})

	g.POST("/auth/signup", func(c echo.Context) error {
		signup := &api.Signup{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, signup); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted signup request").SetInternal(err)
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(signup.Password), bcrypt.DefaultCost)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate password hash").SetInternal(err)
		}

		principalCreate := &api.PrincipalCreate{
			CreatorId:    api.SYSTEM_BOT_ID,
			Type:         api.EndUser,
			Name:         signup.Name,
			Email:        signup.Email,
			PasswordHash: string(passwordHash),
		}

		user, err := s.PrincipalService.CreatePrincipal(context.Background(), principalCreate)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Email already exists: %s", signup.Email))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to signup").SetInternal(err)
		}

		findRole := api.Owner
		find := &api.MemberFind{
			Role: &findRole,
		}
		list, err := s.MemberService.FindMemberList(context.Background(), find)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to signup").SetInternal(err)
		}

		// Grant the member Owner role if there is no existing Owner member.
		role := api.Developer
		if len(list) == 0 {
			role = api.Owner
		}
		memberCreate := &api.MemberCreate{
			CreatorId:   user.ID,
			Status:      api.Active,
			Role:        role,
			PrincipalId: user.ID,
		}

		member, err := s.MemberService.CreateMember(context.Background(), memberCreate)
		if err != nil {
			if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
				return echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Member already exists: %s", signup.Email))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to signup").SetInternal(err)
		}

		{
			bytes, err := json.Marshal(api.ActivityMemberCreatePayload{
				PrincipalId:    member.PrincipalId,
				PrincipalName:  user.Name,
				PrincipalEmail: user.Email,
				MemberStatus:   member.Status,
				Role:           member.Role,
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
			}
			activity, err := s.ActivityService.CreateActivity(context.Background(), &api.ActivityCreate{
				CreatorId:   user.ID,
				ContainerId: member.ID,
				Type:        api.ActivityMemberCreate,
				Level:       api.ACTIVITY_INFO,
				Payload:     string(bytes),
			})
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after create member: %d", member.ID)).SetInternal(err)
			}

			if err := s.PostInboxMemberActivity(context.Background(), member, activity.ID); err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to post activity to inbox").SetInternal(err)
			}
		}

		if err := GenerateTokensAndSetCookies(c, user, s.mode, s.secret); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate access token").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, user); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal signup response").SetInternal(err)
		}
		return nil
	})
}
