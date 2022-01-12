package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
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
		list, err := s.VCSService.FindVCSList(ctx, vcsFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch vcs list").SetInternal(err)
		}

		authProvider := make([]*api.AuthProvider, 0)
		for _, vcs := range list {
			newProvider := &api.AuthProvider{
				Type:          vcs.Type,
				Name:          vcs.Name,
				InstanceURL:   vcs.InstanceURL,
				ApplicationID: vcs.ApplicationID,
				Secret:        vcs.Secret,
			}
			authProvider = append(authProvider, newProvider)
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, authProvider); err != nil {
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
				user, err = s.PrincipalService.FindPrincipal(ctx, principalFind)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to authenticate user").SetInternal(err)
				}
				if user == nil {
					return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("User not found: %s", login.Email))
				}
				// if user had already use third party method logged in, we will forbid she login via Bytebase
				if user.AuthProvider != api.PrincipalAuthProviderBytebase {
					return echo.NewHTTPError(http.StatusUnauthorized, "User has bond to Gitlab")
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
				GitlabUserInfo, err := vcsPlugin.Get("GITLAB_SELF_HOST", vcsPlugin.ProviderConfig{Logger: s.l}).TryLogin(ctx,
					common.OauthContext{
						ClientID:     gitlabLogin.ApplicationID,
						ClientSecret: gitlabLogin.Secret,
						AccessToken:  gitlabLogin.AccessToken,
						RefreshToken: "",
						Refresher:    nil,
					},
					gitlabLogin.InstanceURL,
				)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Fail to fetch user info from gitlab").SetInternal(err)
				}

				// we only allow active user to login via gitlab
				if GitlabUserInfo.State != vcsPlugin.UserStateActive {
					return echo.NewHTTPError(http.StatusInternalServerError, "Fail to login via Gitlab, user is Archived")
				}

				principalFind := &api.PrincipalFind{
					Email: &GitlabUserInfo.Email,
				}
				user, err = s.PrincipalService.FindPrincipal(ctx, principalFind)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to authenticate user").SetInternal(err)
				}

				// create a new user if not exist
				if user == nil {
					// if user login via gitlab at the first time, we will generate a random password.
					// The random password is supposed to be not guessable.
					// But even one can guess the password, she won't be allowed to login.
					// If user signs up via 3rd party login like GitLab, we will disallow bytebase builtin password/email login unless user manually switch the login option later.
					signup := &api.Signup{
						Email:    GitlabUserInfo.Email,
						Password: common.RandomString(20),
						Name:     GitlabUserInfo.Name,
					}
					var httpError *echo.HTTPError
					user, httpError = trySignup(ctx, s, signup, api.PrincipalAuthProviderGitlabSelfHost)
					if httpError != nil {
						return httpError
					}
				}
			}
		}

		// test the status of this user
		memberFind := &api.MemberFind{
			PrincipalID: &user.ID,
		}
		member, err := s.MemberService.FindMember(ctx, memberFind)
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
		signup := &api.Signup{}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, signup); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted signup request").SetInternal(err)
		}

		user, err := trySignup(ctx, s, signup, api.PrincipalAuthProviderBytebase)
		if err != nil {
			return err
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

func trySignup(ctx context.Context, s *Server, signup *api.Signup, authProvider api.PrincipalAuthProvider) (*api.Principal, *echo.HTTPError) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(signup.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate password hash").SetInternal(err)
	}

	principalCreate := &api.PrincipalCreate{
		CreatorID:    api.SystemBotID,
		Type:         api.EndUser,
		AuthProvider: authProvider,
		Name:         signup.Name,
		Email:        signup.Email,
		PasswordHash: string(passwordHash),
	}
	user, err := s.PrincipalService.CreatePrincipal(ctx, principalCreate)
	if err != nil {
		if common.ErrorCode(err) == common.Conflict {
			return nil, echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Email already exists: %s", signup.Email))
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to signup").SetInternal(err)
	}

	findRole := api.Owner
	find := &api.MemberFind{
		Role: &findRole,
	}
	list, err := s.MemberService.FindMemberList(ctx, find)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to signup").SetInternal(err)
	}

	// Grant the member Owner role if there is no existing Owner member.
	role := api.Developer
	if len(list) == 0 {
		role = api.Owner
	}
	memberCreate := &api.MemberCreate{
		CreatorID:   user.ID,
		Status:      api.Active,
		Role:        role,
		PrincipalID: user.ID,
	}

	member, err := s.MemberService.CreateMember(ctx, memberCreate)
	if err != nil {
		if common.ErrorCode(err) == common.Conflict {
			return nil, echo.NewHTTPError(http.StatusConflict, fmt.Sprintf("Member already exists: %s", signup.Email))
		}
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to signup").SetInternal(err)
	}

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
