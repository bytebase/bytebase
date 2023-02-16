package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/activity"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
	"github.com/bytebase/bytebase/backend/store"
)

func (s *Server) registerAuthRoutes(g *echo.Group) {
	g.GET("/auth/provider", func(c echo.Context) error {
		ctx := c.Request().Context()
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
				// we do not return secret to the frontend for safety concern
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
		ctx := c.Request().Context()
		var user *store.UserMessage

		authProvider := api.PrincipalAuthProvider(c.Param("auth_provider"))
		switch authProvider {
		case api.PrincipalAuthProviderBytebase:
			{
				login := &api.Login{}
				if err := jsonapi.UnmarshalPayload(c.Request().Body, login); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "Malformed login request").SetInternal(err)
				}

				var err error
				user, err = s.store.GetUser(ctx, &store.FindUserMessage{
					Email:       &login.Email,
					ShowDeleted: true,
				})
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
		case api.PrincipalAuthProviderGitlabSelfHost, api.PrincipalAuthProviderGitHubCom:
			{
				login := &api.VCSLogin{}
				if err := jsonapi.UnmarshalPayload(c.Request().Body, login); err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, "Malformed login request").SetInternal(err)
				}
				vcsFound, err := s.store.GetVCSByID(ctx, login.VCSID)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch vcs, name: %v, ID: %v", login.Name, login.Name)).SetInternal(err)
				}
				if vcsFound == nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("vcs do not exist, name: %v, ID: %v", login.Name, login.Name)).SetInternal(err)
				}

				// Exchange OAuth Token
				oauthToken, err := vcs.Get(vcsFound.Type, vcs.ProviderConfig{}).ExchangeOAuthToken(
					ctx,
					vcsFound.InstanceURL,
					&common.OAuthExchange{
						ClientID:     vcsFound.ApplicationID,
						ClientSecret: vcsFound.Secret,
						Code:         login.Code,
						RedirectURL:  fmt.Sprintf("%s/oauth/callback", s.profile.ExternalURL),
					},
				)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to exchange OAuth token").SetInternal(err)
				}

				userInfo, err := vcs.Get(vcsFound.Type, vcs.ProviderConfig{}).TryLogin(ctx,
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
					return echo.NewHTTPError(http.StatusInternalServerError, "Fail to fetch user info from VCS").SetInternal(err)
				}

				// We only allow active user to login
				if userInfo.State != vcs.StateActive {
					return echo.NewHTTPError(http.StatusUnauthorized, "Fail to login via VCS, user is Archived")
				}

				user, err = s.store.GetUser(ctx, &store.FindUserMessage{
					Email:       &userInfo.PublicEmail,
					ShowDeleted: true,
				})
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, "Failed to authenticate user").SetInternal(err)
				}

				// Create a new user if not exist
				if user == nil {
					if userInfo.PublicEmail == "" {
						profileLink := "https://docs.github.com/en/account-and-profile"
						if authProvider == api.PrincipalAuthProviderGitlabSelfHost {
							profileLink = "https://docs.gitlab.com/ee/user/profile/#set-your-public-email"
						}
						return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Please configure your public email first, %s.", profileLink))
					}
					// If the user logins via VCS for the first time, we will generate a random
					// password. The random password is supposed to be not guessable. If user wants
					// to login via password, they need to set the new password from the profile
					// page.
					password, err := common.RandomString(20)
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate random password").SetInternal(err)
					}
					signUp := &api.SignUp{
						Email:    userInfo.PublicEmail,
						Password: password,
						Name:     userInfo.Name,
					}
					var httpError *echo.HTTPError
					user, httpError = trySignUp(ctx, s, signUp, api.SystemBotID)
					if httpError != nil {
						return httpError
					}
				}
			}
		default:
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unsupported auth provider: %s", authProvider))
		}

		// test the status of this user
		if user == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("user not found: %s", user.Email))
		}
		if user.MemberDeleted {
			return echo.NewHTTPError(http.StatusUnauthorized, "This user has been deactivated by the admin")
		}

		// If password is correct, generate tokens and set cookies.
		if err := GenerateTokensAndSetCookies(c, user, s.profile.Mode, s.secret); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate access token").SetInternal(err)
		}

		composedUser := &api.Principal{
			ID:           user.ID,
			Type:         user.Type,
			Name:         user.Name,
			Email:        user.Email,
			PasswordHash: user.PasswordHash,
			Role:         user.Role,
		}
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, composedUser); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal login response").SetInternal(err)
		}
		return nil
	})
}

func trySignUp(ctx context.Context, s *Server, signUp *api.SignUp, creatorID int) (*store.UserMessage, *echo.HTTPError) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(signUp.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to generate password hash").SetInternal(err)
	}

	users, err := s.store.ListUsers(ctx, &store.FindUserMessage{
		Email:       &signUp.Email,
		ShowDeleted: true,
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to list users").SetInternal(err)
	}
	if len(users) != 0 {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Email %s is already existed", signUp.Email)
	}
	user, err := s.store.CreateUser(ctx, &store.UserMessage{
		Email:        signUp.Email,
		Name:         signUp.Name,
		Type:         api.EndUser,
		PasswordHash: string(passwordHash),
	}, creatorID)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to sign up").SetInternal(err)
	}

	// From now on, the Principal we just created could be composed with a valid Role field.
	bytes, err := json.Marshal(api.ActivityMemberCreatePayload{
		PrincipalID:    user.ID,
		PrincipalName:  user.Name,
		PrincipalEmail: user.Email,
		MemberStatus:   api.Active,
		Role:           user.Role,
	})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to construct activity payload").SetInternal(err)
	}
	activityCreate := &api.ActivityCreate{
		CreatorID:   creatorID,
		ContainerID: user.ID,
		Type:        api.ActivityMemberCreate,
		Level:       api.ActivityInfo,
		Payload:     string(bytes),
	}
	_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &activity.Metadata{})
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to create activity after create user: %d", user.ID)).SetInternal(err)
	}

	return user, nil
}
