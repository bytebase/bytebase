package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/vcs"
	vcsPlugin "github.com/bytebase/bytebase/plugin/vcs"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) registerAuthRoutes(g *echo.Group) {

	// for now, we only support Gitlab
	g.GET("/auth/provider", func(c echo.Context) error {
		ctx := context.Background()
		vcsFind := &api.VCSFind{}
		vcsList, err := s.store.FindVCS(ctx, vcsFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch vcs list").SetInternal(err)
		}

		var authProviderList []*api.AuthProvider
		for _, vcs := range vcsList {
			newProvider := &api.AuthProvider{
				ID:            vcs.ID,
				Type:          vcs.Type,
				Name:          vcs.Name,
				InstanceURL:   vcs.InstanceURL,
				ApplicationID: vcs.ApplicationID,
				// we do not return secret to the frontend for safety
				Secret: "",
			}
			authProviderList = append(authProviderList, newProvider)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, authProviderList); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal auth provider").SetInternal(err)
		}
		return nil
	})

	g.POST("/auth/login/:auth_provider", func(c echo.Context) error {
		ctx := context.Background()
		var user *api.Principal

		authProvider := api.PrincipalAuthProvider(c.Param("auth_provider"))
		switch authProvider {
		case api.PrincipalAuthProviderBytebase:
			{
				login := &api.Login{}
				if err := jsonapi.UnmarshalPayload(c.Request().Body, login); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "Malformatted login request").SetInternal(err)
				}

				principalFind := &api.PrincipalFind{
					Email: &login.Email,
				}
				var err error
				user, err = s.store.FindPrincipal(ctx, principalFind)
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
			}
		case api.PrincipalAuthProviderGitlabSelfHost:
			{
				gitlabLogin := &api.GitlabLogin{}
				if err := jsonapi.UnmarshalPayload(c.Request().Body, gitlabLogin); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "Malformatted gitlab login request").SetInternal(err)
				}
				vcsFound, err := s.store.GetVCSByID(ctx, gitlabLogin.ID)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch vcs, name: %v, ID: %v", gitlabLogin.Name, gitlabLogin.Name)).SetInternal(err)
				}
				if vcsFound == nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("vcs do not exist, name: %v, ID: %v", gitlabLogin.Name, gitlabLogin.Name)).SetInternal(err)
				}

				// exchange Oauth Token
				oauthToken, err := vcsPlugin.Get(vcsFound.Type, vcsPlugin.ProviderConfig{Logger: s.l}).ExchangeOAuthToken(
					ctx,
					vcsFound.InstanceURL,
					common.OAuthExchange{
						ClientID:     vcsFound.ApplicationID,
						ClientSecret: vcsFound.Secret,
					},
					gitlabLogin.Code,
					fmt.Sprintf("%s:%d/oauth/callback", s.frontendHost, s.frontendPort),
				)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to exchange OAuth token").SetInternal(err)
				}

				gitlabUserInfo, err := vcsPlugin.Get(vcs.GitLabSelfHost, vcsPlugin.ProviderConfig{Logger: s.l}).TryLogin(ctx,
					common.OauthContext{
						ClientID:     vcsFound.ApplicationID,
						ClientSecret: vcsFound.Secret,
						AccessToken:  oauthToken.AccessToken,
						RefreshToken: "",
						Refresher:    nil,
					},
					vcsFound.InstanceURL,
				)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Fail to fetch user info from gitlab").SetInternal(err)
				}

				// we only allow active user to login via gitlab
				if gitlabUserInfo.State != vcsPlugin.StateActive {
					return echo.NewHTTPError(http.StatusUnauthorized, "Fail to login via Gitlab, user is Archived")
				}

				principalFind := &api.PrincipalFind{
					Email: &gitlabUserInfo.PublicEmail,
				}
				user, err = s.store.FindPrincipal(ctx, principalFind)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to authenticate user").SetInternal(err)
				}

				// create a new user if not exist
				if user == nil {
					if gitlabUserInfo.PublicEmail == "" {
						return echo.NewHTTPError(http.StatusNotFound, "Please configure your public email first, https://docs.gitlab.com/ee/user/profile/")
					}
					// if user login via gitlab at the first time, we will generate a random password.
					// The random password is supposed to be not guessable. If user wants to login
					// via password, she needs to set the new password from the profile page.
					signUp := &api.SignUp{
						Email:    gitlabUserInfo.PublicEmail,
						Password: common.RandomString(20),
						Name:     gitlabUserInfo.Name,
					}
					var httpError *echo.HTTPError
					user, httpError = trySignUp(ctx, s, signUp, api.SystemBotID)
					if httpError != nil {
						return httpError
					}
				}
			}
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
		ctx := context.Background()
		signUp := &api.SignUp{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, signUp); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted sign up request").SetInternal(err)
		}

		user, err := trySignUp(ctx, s, signUp, api.SystemBotID)
		if err != nil {
			return err
		}

		if err := GenerateTokensAndSetCookies(c, user, s.mode, s.secret); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate access token").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, user); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal sign up response").SetInternal(err)
		}
		return nil
	})

	// TODO(zilong): we may not need to return access token back to the frontend
	g.GET("/auth/exchange-oauth-token/:vcsID", func(c echo.Context) error {
		ctx := context.Background()

		vcsID64, err := strconv.ParseInt(c.Param("vcsID"), 10, 32)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Failed to marshal oauth provider's ID: %v", c.Param("id"))).SetInternal(err)
		}
		vcsID := int(vcsID64)
		code := c.Request().Header.Get("code")

		findVCS := &api.VCSFind{ID: &vcsID}
		vcs, err := s.VCSService.FindVCS(ctx, findVCS)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		if vcs == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Failed to find VCS, ID: %v", vcsID)).SetInternal(err)
		}

		oauthToken, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{Logger: s.l}).ExchangeOAuthToken(
			ctx,
			vcs.InstanceURL,
			common.OAuthExchange{
				ClientID:     vcs.ApplicationID,
				ClientSecret: vcs.Secret,
			},
			code,
			fmt.Sprintf("%s:%d/oauth/callback", s.frontendHost, s.frontendPort),
		)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to exchange OAuth token").SetInternal(err)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, oauthToken); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal exchange OAuth token response").SetInternal(err)
		}

		return nil
	})

}

func trySignUp(ctx context.Context, s *Server, signUp *api.SignUp, CreatorID int) (*api.Principal, *echo.HTTPError) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(signUp.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate password hash").SetInternal(err)
	}

	principalCreate := &api.PrincipalCreate{
		CreatorID:    CreatorID,
		Type:         api.EndUser,
		Name:         signUp.Name,
		Email:        signUp.Email,
		PasswordHash: string(passwordHash),
	}
	// The user has an empty field of Role, which corresponds to the Member object created later.
	user, err := s.store.CreatePrincipal(ctx, principalCreate)
	if err != nil {
		if common.ErrorCode(err) == common.Conflict {
			return nil, echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Email already exists: %s", signUp.Email))
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to sign up").SetInternal(err)
	}

	findRole := api.Owner
	find := &api.MemberFind{
		Role: &findRole,
	}
	memberList, err := s.store.FindMember(ctx, find)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to sign up").SetInternal(err)
	}

	// Grant the member Owner role if there is no existing Owner member.
	role := api.Developer
	if len(memberList) == 0 {
		role = api.Owner
	}
	memberCreate := &api.MemberCreate{
		CreatorID:   CreatorID,
		Status:      api.Active,
		Role:        role,
		PrincipalID: user.ID,
	}

	member, err := s.store.CreateMember(ctx, memberCreate)
	if err != nil {
		if common.ErrorCode(err) == common.Conflict {
			return nil, echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Member already exists: %s", signUp.Email))
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to sign up").SetInternal(err)
	}
	// From now on, the Principal we just created could be composed with a valid Role field.

	{
		bytes, err := json.Marshal(api.ActivityMemberCreatePayload{
			PrincipalID:    member.PrincipalID,
			PrincipalName:  user.Name,
			PrincipalEmail: user.Email,
			MemberStatus:   member.Status,
			Role:           member.Role,
		})
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
		}
		activityCreate := &api.ActivityCreate{
			CreatorID:   user.ID,
			ContainerID: member.ID,
			Type:        api.ActivityMemberCreate,
			Level:       api.ActivityInfo,
			Payload:     string(bytes),
		}
		_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
		if err != nil {
			return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after create member: %d", member.ID)).SetInternal(err)
		}
	}

	return user, nil
}
