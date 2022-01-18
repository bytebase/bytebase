package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	vcsPlugin "github.com/bytebase/bytebase/plugin/vcs"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func (s *Server) registerProjectMemberRoutes(g *echo.Group) {
	// for now we only support sync project member from privately deployed GitLab
	g.POST("/project/:projectID/sync/:roleProvider", func(c echo.Context) error {
		ctx := context.Background()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}
		roleProvider := api.ProjectRoleProvider(c.Param("roleProvider"))

		switch roleProvider {
		case api.ProjectRoleProviderGitlabSelfHost:
			{
				projectFind := &api.ProjectFind{ID: &projectID}
				project, err := s.ProjectService.FindProject(ctx, projectFind)
				if err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project not found: %s", c.Param("projectID"))).SetInternal(err)
				}
				if project.WorkflowType == api.UIWorkflow {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid workflow type: %s", project.WorkflowType))
				}

				repoFind := &api.RepositoryFind{ProjectID: &projectID}
				repo, err := s.RepositoryService.FindRepository(ctx, repoFind)
				if err != nil {
					return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Fail to fetch relevant VCS repo, Project ID: %s", c.Param("projectID"))).SetInternal(err)
				}

				vcsFind := &api.VCSFind{
					ID: &repo.VCSID,
				}
				vcs, err := s.VCSService.FindVCS(ctx, vcsFind)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find VCS for sync project member: %d", repo.VCSID)).SetInternal(err)
				}
				if vcs == nil {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS ID not found: %d", repo.VCSID))
				}

				gitlabProjectMemberList, err := vcsPlugin.Get("GITLAB_SELF_HOST", vcsPlugin.ProviderConfig{Logger: s.l}).FetchProjectMemberList(ctx,
					common.OauthContext{
						ClientID:     vcs.ApplicationID,
						ClientSecret: vcs.Secret,
						AccessToken:  repo.AccessToken,
						RefreshToken: repo.RefreshToken,
						Refresher:    s.refreshToken(ctx, repo.ID),
					},
					vcs.InstanceURL,
					repo.ExternalID,
				)
				if err != nil {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("fail to fetch project member from GitLab, instance URL: %s", vcs.InstanceURL)).SetInternal(err)
				}

				// create or patch project member
				findProjectMember := &api.ProjectMemberFind{ProjectID: &projectID}
				bytebaseProjectMemberList, err := s.ProjectMemberService.FindProjectMemberList(ctx, findProjectMember)
				if err != nil {
					return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("fail to fetch project member in bytebase: %d", projectID)).SetInternal(err)
				}

				// check whether principal exists in our system. if not exist, create one.
				for _, projectMember := range gitlabProjectMemberList {
					findPrincipal := &api.PrincipalFind{Email: &projectMember.Email}
					principal, err := s.PrincipalService.FindPrincipal(ctx, findPrincipal)
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch principal info").SetInternal(err)
					}

					if projectMember.State != vcsPlugin.UserStateActive && projectMember.MembershipState != vcsPlugin.UserStateActive {
						continue
					}

					if principal == nil {
						signupInfo := &api.Signup{
							Name:     projectMember.Name,
							Email:    projectMember.Email,
							Password: common.RandomString(20),
						}
						createdPrincipal, httpErr := TrySignup(ctx, s, signupInfo, api.PrincipalAuthProviderGitlabSelfHost, c.Get(getPrincipalIDContextKey()).(int))
						if httpErr != nil {
							return httpErr
						}
						principal = createdPrincipal
					}

					isProjectMemberExist := false
					for _, bytebaseProjectMember := range bytebaseProjectMemberList {
						if bytebaseProjectMember.PrincipalID == principal.ID {
							payload, _ := json.Marshal(projectMember)
							// try update status
							patchProjectMember := &api.ProjectMemberPatch{
								ID:           bytebaseProjectMember.ID,
								RoleProvider: api.ProjectRoleProviderGitlabSelfHost,
								Payload:      string(payload),
							}
							_, err := s.ProjectMemberService.PatchProjectMember(ctx, patchProjectMember)
							if err != nil {
								return echo.NewHTTPError(http.StatusInternalServerError, "Failed to sync member from GitLab").SetInternal(err)
							}
							isProjectMemberExist = true
						}
					}

					if !isProjectMemberExist {
						var role api.ProjectRole
						switch projectMember.AccessLevel { // see https://docs.gitlab.com/ee/api/members.html
						case 50, /* Owner */
							40 /* Maintainer */ :
							role = api.ProjectOwner
						case 30, /* Developer */
							20, /* Reporter */
							10, /* Guest */
							5,  /* Minimal access */
							0 /* No access */ :
							role = api.ProjectDeveloper
						}

						payload, _ := json.Marshal(projectMember)
						createProjectMember := &api.ProjectMemberCreate{
							ProjectID:    projectID,
							CreatorID:    c.Get(getPrincipalIDContextKey()).(int),
							RoleProvider: api.ProjectRoleProviderGitlabSelfHost,
							Payload:      string(payload),
							PrincipalID:  principal.ID,
							Role:         role,
						}
						_, err := s.ProjectMemberService.CreateProjectMember(ctx, createProjectMember)
						if err != nil {
							return echo.NewHTTPError(http.StatusInternalServerError, "Failed to mapping project member from GitLab").SetInternal(err)
						}
					}
				}
			}
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to sync project member, invalid provider type: %s", roleProvider))
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		return nil
	})

	g.POST("/project/:projectID/member", func(c echo.Context) error {
		ctx := context.Background()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		projectMemberCreate := &api.ProjectMemberCreate{
			ProjectID: projectID,
			CreatorID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, projectMemberCreate); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted create project membership request").SetInternal(err)
		}

		projectMember, err := s.ProjectMemberService.CreateProjectMember(ctx, projectMemberCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, "User is already a project member")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create project member").SetInternal(err)
		}

		if err := s.composeProjectMemberRelationship(ctx, projectMember); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch created project membership relationship").SetInternal(err)
		}

		{
			activityCreate := &api.ActivityCreate{
				CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
				ContainerID: projectID,
				Type:        api.ActivityProjectMemberCreate,
				Level:       api.ActivityInfo,
				Comment: fmt.Sprintf("Granted %s to %s (%s).",
					projectMember.Principal.Name, projectMember.Principal.Email, projectMember.Role),
			}
			_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
			if err != nil {
				s.l.Warn("Failed to create project activity after creating member",
					zap.Int("project_id", projectID),
					zap.Int("principal_id", projectMember.Principal.ID),
					zap.String("principal_name", projectMember.Principal.Name),
					zap.String("role", projectMember.Role),
					zap.Error(err))
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, projectMember); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal create projectMember response").SetInternal(err)
		}
		return nil
	})

	g.PATCH("/project/:projectID/member/:memberID", func(c echo.Context) error {
		ctx := context.Background()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		id, err := strconv.Atoi(c.Param("memberID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("memberID"))).SetInternal(err)
		}

		existingProjectMember, err := s.ProjectMemberService.FindProjectMember(ctx, &api.ProjectMemberFind{ID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete project member ID: %v", id)).SetInternal(err)
		}
		if existingProjectMember == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project member ID not found: %d", id))
		}

		projectMemberPatch := &api.ProjectMemberPatch{
			ID:        id,
			UpdaterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := jsonapi.UnmarshalPayload(c.Request().Body, projectMemberPatch); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Malformatted change project membership").SetInternal(err)
		}

		projectMember, err := s.ProjectMemberService.PatchProjectMember(ctx, projectMemberPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project member ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change project membership ID: %v", id)).SetInternal(err)
		}

		if err := s.composeProjectMemberRelationship(ctx, projectMember); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch updated project membership relationship").SetInternal(err)
		}

		{
			activityCreate := &api.ActivityCreate{
				CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
				ContainerID: projectID,
				Type:        api.ActivityProjectMemberRoleUpdate,
				Level:       api.ActivityInfo,
				Comment: fmt.Sprintf("Changed %s (%s) from %s to %s.",
					projectMember.Principal.Name, projectMember.Principal.Email, existingProjectMember.Role, projectMember.Role),
			}
			_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
			if err != nil {
				s.l.Warn("Failed to create project activity after updating member role",
					zap.Int("project_id", projectID),
					zap.Int("principal_id", projectMember.Principal.ID),
					zap.String("principal_name", projectMember.Principal.Name),
					zap.String("old_role", existingProjectMember.Role),
					zap.String("new_role", projectMember.Role),
					zap.Error(err))
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		if err := jsonapi.MarshalPayload(c.Response().Writer, projectMember); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to marshal project membership change response: %v", id)).SetInternal(err)
		}
		return nil
	})

	g.DELETE("/project/:projectID/member/:memberID", func(c echo.Context) error {
		ctx := context.Background()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		id, err := strconv.Atoi(c.Param("memberID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("ID is not a number: %s", c.Param("memberID"))).SetInternal(err)
		}

		projectMember, err := s.ProjectMemberService.FindProjectMember(ctx, &api.ProjectMemberFind{ID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete project member ID: %v", id)).SetInternal(err)
		}
		if projectMember == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project member ID not found: %d", id))
		}

		projectMemberDelete := &api.ProjectMemberDelete{
			ID:        id,
			DeleterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := s.ProjectMemberService.DeleteProjectMember(ctx, projectMemberDelete); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete project member ID: %v", id)).SetInternal(err)
		}

		{
			projectMember.Principal, err = s.composePrincipalByID(ctx, projectMember.PrincipalID)
			if err == nil {
				activityCreate := &api.ActivityCreate{
					CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
					ContainerID: projectID,
					Type:        api.ActivityProjectMemberDelete,
					Level:       api.ActivityInfo,
					Comment: fmt.Sprintf("Revoked %s from %s (%s).",
						projectMember.Role, projectMember.Principal.Name, projectMember.Principal.Email),
				}
				_, err = s.ActivityManager.CreateActivity(ctx, activityCreate, &ActivityMeta{})
			}
			if err != nil {
				s.l.Warn("Failed to create project activity after deleting member",
					zap.Int("project_id", projectID),
					zap.Int("principal_id", projectMember.Principal.ID),
					zap.String("principal_name", projectMember.Principal.Name),
					zap.String("role", projectMember.Role),
					zap.Error(err))
			}
		}

		c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSONCharsetUTF8)
		c.Response().WriteHeader(http.StatusOK)
		return nil
	})
}

func (s *Server) composeProjectMemberListByProjectID(ctx context.Context, projectID int) ([]*api.ProjectMember, error) {
	projectMemberFind := &api.ProjectMemberFind{
		ProjectID: &projectID,
	}
	projectMemberList, err := s.ProjectMemberService.FindProjectMemberList(ctx, projectMemberFind)
	if err != nil {
		return nil, err
	}

	for _, projectMember := range projectMemberList {
		if err := s.composeProjectMemberRelationship(ctx, projectMember); err != nil {
			return nil, err
		}
	}
	return projectMemberList, nil
}

func (s *Server) composeProjectMemberRelationship(ctx context.Context, projectMember *api.ProjectMember) error {
	var err error

	projectMember.Creator, err = s.composePrincipalByID(ctx, projectMember.CreatorID)
	if err != nil {
		return err
	}

	projectMember.Updater, err = s.composePrincipalByID(ctx, projectMember.UpdaterID)
	if err != nil {
		return err
	}

	projectMember.Principal, err = s.composePrincipalByID(ctx, projectMember.PrincipalID)
	if err != nil {
		return err
	}

	return nil
}
