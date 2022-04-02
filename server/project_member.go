package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	vcsPlugin "github.com/bytebase/bytebase/plugin/vcs"
	"github.com/google/jsonapi"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func (s *Server) registerProjectMemberRoutes(g *echo.Group) {
	// for now we only support sync project member from privately deployed GitLab
	g.POST("/project/:projectID/syncmember", func(c echo.Context) error {
		ctx := context.Background()
		projectID, err := strconv.Atoi(c.Param("projectID"))
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project ID is not a number: %s", c.Param("projectID"))).SetInternal(err)
		}

		projectFind := &api.ProjectFind{ID: &projectID}
		project, err := s.ProjectService.FindProject(ctx, projectFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Project not found: %s", c.Param("projectID"))).SetInternal(err)
		}
		if project == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project ID not found: %d", projectID))
		}
		if project.WorkflowType != api.VCSWorkflow {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid workflow type: %s, need %s to enable this function", project.WorkflowType, api.VCSWorkflow))
		}

		// fetch project member from VCS
		repoFind := &api.RepositoryFind{ProjectID: &projectID}
		repo, err := s.RepositoryService.FindRepository(ctx, repoFind)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch relevant VCS repo, Project ID: %s", c.Param("projectID"))).SetInternal(err)
		}
		vcs, err := s.store.GetVCSByID(ctx, repo.VCSID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to find VCS for sync project member: %d", repo.VCSID)).SetInternal(err)
		}
		if vcs == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("VCS not found with ID: %d", repo.VCSID))
		}
		vcsProjectMemberList, err := vcsPlugin.Get(vcs.Type, vcsPlugin.ProviderConfig{Logger: s.l}).FetchRepositoryActiveMemberList(ctx,
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
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to fetch repository member from VCS, instance URL: %s", vcs.InstanceURL)).SetInternal(err)
		}

		// The following block will check whether the relevant principal exists in our system.
		// If the principal does not exist, we will try to create one out of the vcs member info.
		var createList []*api.ProjectMemberCreate
		// we declare latSyncTs to ensure that every projectMember would have the same sync time.
		lastSyncTs := time.Now().UTC().Unix()
		for _, projectMember := range vcsProjectMemberList {
			if vcs.Type != projectMember.RoleProvider {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Invalid role provider, expected: %v, got: %v", vcs.Type, projectMember.RoleProvider)).SetInternal(err)
			}

			findPrincipal := &api.PrincipalFind{Email: &projectMember.Email}
			principal, err := s.store.FindPrincipal(ctx, findPrincipal)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch principal info").SetInternal(err)
			}
			if principal == nil { // try to create principal
				signUpInfo := &api.SignUp{
					Name:  projectMember.Name,
					Email: projectMember.Email,
					// Principal created via this method would have no chance to set their password.
					// To prevent potential security issues, we use random string to set up her password.
					// This is another safety measure since we already disallow user login via password
					// if the principal uses external auth provider
					Password: common.RandomString(20),
				}
				createdPrincipal, httpErr := trySignUp(ctx, s, signUpInfo, c.Get(getPrincipalIDContextKey()).(int))
				if httpErr != nil {
					return httpErr
				}
				principal = createdPrincipal
			}

			providerPayload := &api.ProjectRoleProviderPayload{
				VCSRole:    projectMember.VCSRole,
				LastSyncTs: lastSyncTs,
			}
			providerPayloadBytes, err := json.Marshal(providerPayload)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to marshal providerPayload").SetInternal(err)
			}
			createProjectMember := &api.ProjectMemberCreate{
				ProjectID:    projectID,
				CreatorID:    c.Get(getPrincipalIDContextKey()).(int),
				PrincipalID:  principal.ID,
				Role:         projectMember.Role,
				RoleProvider: api.ProjectRoleProvider(projectMember.RoleProvider),
				Payload:      string(providerPayloadBytes),
			}
			createList = append(createList, createProjectMember)
		}

		batchUpdateProjectMember := &api.ProjectMemberBatchUpdate{
			ID:           projectID,
			UpdaterID:    c.Get(getPrincipalIDContextKey()).(int),
			RoleProvider: api.ProjectRoleProviderGitLabSelfHost, /* we only support gitlab for now */
			List:         createList,
		}
		createdMemberRawList, deletedMemberRawList, err := s.ProjectMemberService.BatchUpdateProjectMember(ctx, batchUpdateProjectMember)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to sync project member from VCS").SetInternal(err)
		}
		var createdMemberList []*api.ProjectMember
		var deletedMemberList []*api.ProjectMember
		for _, raw := range createdMemberRawList {
			createdMember, err := s.composeProjectMemberRelationship(ctx, raw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to compose project member relationship with ID %d", raw.ID)).SetInternal(err)
			}
			createdMemberList = append(createdMemberList, createdMember)
		}
		for _, raw := range deletedMemberRawList {
			deletedMember, err := s.composeProjectMemberRelationship(ctx, raw)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to compose project member relationship with ID %d", raw.ID)).SetInternal(err)
			}
			deletedMemberList = append(deletedMemberList, deletedMember)
		}

		createdIDMemberMap := make(map[int]*api.ProjectMember)
		for _, createdMember := range createdMemberList {
			createdIDMemberMap[createdMember.PrincipalID] = createdMember
		}
		deletedIDMemberMap := make(map[int]*api.ProjectMember)
		for _, deletedMember := range deletedMemberList {
			deletedIDMemberMap[deletedMember.PrincipalID] = deletedMember
		}

		// create ROLE CREATE/ MEMBER UPDATE activity
		for id, createdMember := range createdIDMemberMap {
			// if the same member exist before, we will create a ROLE UPDATE activity
			if deletedMember, ok := deletedIDMemberMap[id]; ok {
				// do nothing if nothing changed
				if createdMember.Role == deletedMember.Role && createdMember.RoleProvider == deletedMember.RoleProvider {
					continue
				}
				principal, err := s.store.GetPrincipalByID(ctx, createdMember.PrincipalID)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Fail to create member relation, Principal ID: %v", principal.ID)).SetInternal(err)
				}
				activityUpdateMember := &api.ActivityCreate{
					CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
					ContainerID: projectID,
					Type:        api.ActivityProjectMemberRoleUpdate,
					Level:       api.ActivityInfo,
					Comment: fmt.Sprintf("Changed %s (%s) from %s (provided by %s) to %s (provided by %s).",
						principal.Name, principal.Email, deletedMember.Role, deletedMember.RoleProvider, createdMember.Role, createdMember.RoleProvider),
				}
				if _, err = s.ActivityManager.CreateActivity(ctx, activityUpdateMember, &ActivityMeta{}); err != nil {
					s.l.Warn("Failed to create project activity after updating member role",
						zap.Int("project_id", projectID),
						zap.Int("principal_id", principal.ID),
						zap.String("principal_name", principal.Name),
						zap.String("old_role", deletedMember.Role),
						zap.String("new_role", createdMember.Role),
						zap.Error(err))
				}
			} else {
				// elsewise, we will create a MEMBER CREATE activity
				principalFind := &api.PrincipalFind{ID: &createdMember.PrincipalID}
				principal, err := s.store.FindPrincipal(ctx, principalFind)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Fail to find the relevant principal of the member relation, principal ID: %v", principal.ID)).SetInternal(err)
				}
				activityCreateMember := &api.ActivityCreate{
					CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
					ContainerID: projectID,
					Type:        api.ActivityProjectMemberCreate,
					Level:       api.ActivityInfo,
					Comment: fmt.Sprintf("Granted %s to %s (%s) (synced from VCS).",
						principal.Name, principal.Email, createdMember.Role),
				}
				if _, err = s.ActivityManager.CreateActivity(ctx, activityCreateMember, &ActivityMeta{}); err != nil {
					s.l.Warn("Failed to create project activity after creating member",
						zap.Int("project_id", projectID),
						zap.Int("principal_id", principal.ID),
						zap.String("principal_name", principal.Name),
						zap.String("role", string(createdMember.Role)),
						zap.Error(err))
				}
			}
		}

		// create MEMBER DELETE activity
		for id, deletedMember := range deletedIDMemberMap {
			if _, ok := createdIDMemberMap[id]; ok {
				// if the member does exist in createdMemberList, meaning we need to update this member(already done above).
				continue
			}
			principal, err := s.store.GetPrincipalByID(ctx, deletedMember.PrincipalID)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Fail to create member relation, Principal ID: %v", deletedMember.PrincipalID)).SetInternal(err)
			}
			activityDeleteMember := &api.ActivityCreate{
				CreatorID:   c.Get(getPrincipalIDContextKey()).(int),
				ContainerID: projectID,
				Type:        api.ActivityProjectMemberDelete,
				Level:       api.ActivityInfo,
				Comment: fmt.Sprintf("Revoked %s from %s (%s). Because this member does not belong to the VCS.",
					principal.Name, principal.Email, deletedMember.Role),
			}
			if _, err = s.ActivityManager.CreateActivity(ctx, activityDeleteMember, &ActivityMeta{}); err != nil {
				s.l.Warn("Failed to create project activity after creating member",
					zap.Int("project_id", projectID),
					zap.Int("principal_id", principal.ID),
					zap.String("principal_name", principal.Name),
					zap.String("role", deletedMember.Role),
					zap.Error(err))
			}
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

		projectMemberRaw, err := s.ProjectMemberService.CreateProjectMember(ctx, projectMemberCreate)
		if err != nil {
			if common.ErrorCode(err) == common.Conflict {
				return echo.NewHTTPError(http.StatusConflict, "User is already a project member")
			}
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create project member").SetInternal(err)
		}

		projectMember, err := s.composeProjectMemberRelationship(ctx, projectMemberRaw)
		if err != nil {
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

		projectMemberRaw, err := s.ProjectMemberService.PatchProjectMember(ctx, projectMemberPatch)
		if err != nil {
			if common.ErrorCode(err) == common.NotFound {
				return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project member ID not found: %d", id))
			}
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to change project membership ID: %v", id)).SetInternal(err)
		}

		projectMember, err := s.composeProjectMemberRelationship(ctx, projectMemberRaw)
		if err != nil {
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

		projectMemberRaw, err := s.ProjectMemberService.FindProjectMember(ctx, &api.ProjectMemberFind{ID: &id})
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete project member ID: %v", id)).SetInternal(err)
		}
		if projectMemberRaw == nil {
			return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("Project member ID not found: %d", id))
		}
		projectMember, err := s.composeProjectMemberRelationship(ctx, projectMemberRaw)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to compose project member relationship with ID %d", id)).SetInternal(err)
		}

		projectMemberDelete := &api.ProjectMemberDelete{
			ID:        id,
			DeleterID: c.Get(getPrincipalIDContextKey()).(int),
		}
		if err := s.ProjectMemberService.DeleteProjectMember(ctx, projectMemberDelete); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to delete project member ID: %v", id)).SetInternal(err)
		}

		{
			projectMember.Principal, err = s.store.GetPrincipalByID(ctx, projectMember.PrincipalID)
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
	projectMemberRawList, err := s.ProjectMemberService.FindProjectMemberList(ctx, projectMemberFind)
	if err != nil {
		return nil, err
	}
	var projectMemberList []*api.ProjectMember

	for _, projectMemberRaw := range projectMemberRawList {
		projectMember, err := s.composeProjectMemberRelationship(ctx, projectMemberRaw)
		if err != nil {
			return nil, err
		}
		projectMemberList = append(projectMemberList, projectMember)
	}
	return projectMemberList, nil
}

func (s *Server) composeProjectMemberRelationship(ctx context.Context, raw *api.ProjectMemberRaw) (*api.ProjectMember, error) {
	projectMember := raw.ToProjectMember()

	creator, err := s.store.GetPrincipalByID(ctx, projectMember.CreatorID)
	if err != nil {
		return nil, err
	}
	projectMember.Creator = creator

	updater, err := s.store.GetPrincipalByID(ctx, projectMember.UpdaterID)
	if err != nil {
		return nil, err
	}
	projectMember.Updater = updater

	principal, err := s.store.GetPrincipalByID(ctx, projectMember.PrincipalID)
	if err != nil {
		return nil, err
	}
	projectMember.Principal = principal

	return projectMember, nil
}
