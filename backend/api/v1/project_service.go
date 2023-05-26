package v1

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/activity"
	webhookPlugin "github.com/bytebase/bytebase/backend/plugin/webhook"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ProjectService implements the project service.
type ProjectService struct {
	v1pb.UnimplementedProjectServiceServer
	store           *store.Store
	activityManager *activity.Manager
}

// NewProjectService creates a new ProjectService.
func NewProjectService(store *store.Store, activityManager *activity.Manager) *ProjectService {
	return &ProjectService{
		store:           store,
		activityManager: activityManager,
	}
}

// GetProject gets a project.
func (s *ProjectService) GetProject(ctx context.Context, request *v1pb.GetProjectRequest) (*v1pb.Project, error) {
	project, err := s.getProjectMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	return convertToProject(project), nil
}

// ListProjects lists all projects.
func (s *ProjectService) ListProjects(ctx context.Context, request *v1pb.ListProjectsRequest) (*v1pb.ListProjectsResponse, error) {
	projects, err := s.store.ListProjectV2(ctx, &store.FindProjectMessage{ShowDeleted: request.ShowDeleted})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	response := &v1pb.ListProjectsResponse{}
	for _, project := range projects {
		response.Projects = append(response.Projects, convertToProject(project))
	}
	return response, nil
}

// CreateProject creates a project.
func (s *ProjectService) CreateProject(ctx context.Context, request *v1pb.CreateProjectRequest) (*v1pb.Project, error) {
	if !isValidResourceID(request.ProjectId) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid project ID %v", request.ProjectId)
	}

	projectMessage, err := convertToProjectMessage(request.ProjectId, request.Project)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	project, err := s.store.CreateProjectV2(ctx,
		projectMessage,
		principalID,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToProject(project), nil
}

// UpdateProject updates a project.
func (s *ProjectService) UpdateProject(ctx context.Context, request *v1pb.UpdateProjectRequest) (*v1pb.Project, error) {
	if request.Project == nil {
		return nil, status.Errorf(codes.InvalidArgument, "project must be set")
	}
	if request.UpdateMask == nil {
		return nil, status.Errorf(codes.InvalidArgument, "update_mask must be set")
	}

	project, err := s.getProjectMessage(ctx, request.Project.Name)
	if err != nil {
		return nil, err
	}
	if project.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "project %q has been deleted", request.Project.Name)
	}
	if project.ResourceID == api.DefaultProjectID {
		return nil, status.Errorf(codes.InvalidArgument, "default project cannot be updated")
	}

	patch := &store.UpdateProjectMessage{
		UpdaterID:  ctx.Value(common.PrincipalIDContextKey).(int),
		ResourceID: project.ResourceID,
	}

	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Title = &request.Project.Title
		case "key":
			patch.Key = &request.Project.Key
		case "workflow":
			workflow := convertToProjectWorkflowType(request.Project.Workflow)
			patch.Workflow = &workflow
		case "tenant_mode":
			tenantMode := convertToProjectTenantMode(request.Project.TenantMode)
			patch.TenantMode = &tenantMode
		case "db_name_template":
			patch.DBNameTemplate = &request.Project.DbNameTemplate
		case "schema_change":
			schemaChange := convertToProjectSchemaChangeType(request.Project.SchemaChange)
			patch.SchemaChangeType = &schemaChange
		}
	}

	project, err = s.store.UpdateProjectV2(ctx, patch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToProject(project), nil
}

// DeleteProject deletes a project.
func (s *ProjectService) DeleteProject(ctx context.Context, request *v1pb.DeleteProjectRequest) (*emptypb.Empty, error) {
	project, err := s.getProjectMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	if project.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "project %q has been deleted", request.Name)
	}
	if project.ResourceID == api.DefaultProjectID {
		return nil, status.Errorf(codes.InvalidArgument, "default project cannot be deleted")
	}

	// Resources prevent project deletion.
	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID, ShowDeleted: true})
	if err != nil {
		return nil, err
	}
	// We don't move the sheet to default project because BYTEBASE_ARTIFACT sheets belong to the issue and issue project.
	if request.Force {
		defaultProject := api.DefaultProjectID
		if _, err := s.store.BatchUpdateDatabaseProject(ctx, databases, defaultProject, api.SystemBotID); err != nil {
			return nil, err
		}
		// We don't close the issues because they might be open still.
	} else {
		// Return the open issue error first because that's more important than transferring out databases.
		openIssues, err := s.store.ListIssueV2(ctx, &store.FindIssueMessage{ProjectUID: &project.UID, StatusList: []api.IssueStatus{api.IssueOpen}})
		if err != nil {
			return nil, err
		}
		if len(openIssues) > 0 {
			return nil, status.Errorf(codes.FailedPrecondition, "resolve all open issues before deleting the project")
		}
		if len(databases) > 0 {
			return nil, status.Errorf(codes.FailedPrecondition, "transfer all databases to the default project before deleting the project")
		}
	}

	if _, err := s.store.UpdateProjectV2(ctx, &store.UpdateProjectMessage{
		UpdaterID:  ctx.Value(common.PrincipalIDContextKey).(int),
		ResourceID: project.ResourceID,
		Delete:     &deletePatch,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

// UndeleteProject undeletes a project.
func (s *ProjectService) UndeleteProject(ctx context.Context, request *v1pb.UndeleteProjectRequest) (*v1pb.Project, error) {
	project, err := s.getProjectMessage(ctx, request.Name)
	if err != nil {
		return nil, err
	}
	if !project.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "project %q is active", request.Name)
	}

	project, err = s.store.UpdateProjectV2(ctx, &store.UpdateProjectMessage{
		UpdaterID:  ctx.Value(common.PrincipalIDContextKey).(int),
		ResourceID: project.ResourceID,
		Delete:     &undeletePatch,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToProject(project), nil
}

// SearchProjects searches all projects that the caller have permission to.
func (s *ProjectService) SearchProjects(ctx context.Context, _ *v1pb.SearchProjectsRequest) (*v1pb.SearchProjectsResponse, error) {
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	role := ctx.Value(common.RoleContextKey).(api.Role)

	projects, err := s.store.ListProjectV2(ctx, &store.FindProjectMessage{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	response := &v1pb.SearchProjectsResponse{}
	for _, project := range projects {
		policy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{ProjectID: &project.ResourceID})
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		if !isOwnerOrDBA(role) && !isProjectMember(policy, principalID) {
			continue
		}
		response.Projects = append(response.Projects, convertToProject(project))
	}
	return response, nil
}

// GetIamPolicy returns the IAM policy for a project.
func (s *ProjectService) GetIamPolicy(ctx context.Context, request *v1pb.GetIamPolicyRequest) (*v1pb.IamPolicy, error) {
	projectID, err := getProjectID(request.Project)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	iamPolicy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{
		ProjectID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return convertToIamPolicy(iamPolicy), nil
}

// BatchGetIamPolicy returns the IAM policy for projects in batch.
func (s *ProjectService) BatchGetIamPolicy(ctx context.Context, request *v1pb.BatchGetIamPolicyRequest) (*v1pb.BatchGetIamPolicyResponse, error) {
	resp := &v1pb.BatchGetIamPolicyResponse{}
	for _, name := range request.Names {
		projectID, err := getProjectID(name)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}

		iamPolicy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{
			ProjectID: &projectID,
		})
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
		resp.PolicyResults = append(resp.PolicyResults, &v1pb.BatchGetIamPolicyResponse_PolicyResult{
			Project: name,
			Policy:  convertToIamPolicy(iamPolicy),
		})
	}
	return resp, nil
}

// SetIamPolicy sets the IAM policy for a project.
func (s *ProjectService) SetIamPolicy(ctx context.Context, request *v1pb.SetIamPolicyRequest) (*v1pb.IamPolicy, error) {
	projectID, err := getProjectID(request.Project)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	creatorUID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "cannot get principal ID from context")
	}
	roleMessages, err := s.store.ListRoles(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to list roles: %v", err)
	}
	if err := validateIAMPolicy(request.Policy, convertToRoles(roleMessages)); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", request.Project)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "project %q has been deleted", request.Project)
	}

	policy, err := s.convertToIAMPolicyMessage(ctx, request.Policy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	oldPolicy, err := s.store.GetProjectPolicy(ctx, &store.GetProjectPolicyMessage{UID: &project.UID})
	if err != nil {
		return nil, err
	}
	remove, add, err := store.GetIAMPolicyDiff(oldPolicy, policy)
	if err != nil {
		return nil, err
	}
	s.CreateIAMPolicyUpdateActivity(ctx, remove, add, project, creatorUID)

	iamPolicy, err := s.store.SetProjectIAMPolicy(ctx, policy, creatorUID, project.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return convertToIamPolicy(iamPolicy), nil
}

// CreateIAMPolicyUpdateActivity creates project IAM policy change activities.
func (s *ProjectService) CreateIAMPolicyUpdateActivity(ctx context.Context, remove, add *store.IAMPolicyMessage, project *store.ProjectMessage, creatorUID int) {
	var activities []*api.ActivityCreate
	for _, binding := range remove.Bindings {
		for _, member := range binding.Members {
			activities = append(activities, &api.ActivityCreate{
				CreatorID:   creatorUID,
				ContainerID: project.UID,
				Type:        api.ActivityProjectMemberDelete,
				Level:       api.ActivityInfo,
				Comment:     fmt.Sprintf("Revoked %s from %s (%s).", binding.Role, member.Name, member.Email),
			})
		}
	}
	for _, binding := range add.Bindings {
		for _, member := range binding.Members {
			activities = append(activities, &api.ActivityCreate{
				CreatorID:   creatorUID,
				ContainerID: project.UID,
				Type:        api.ActivityProjectMemberCreate,
				Level:       api.ActivityInfo,
				Comment:     fmt.Sprintf("Granted %s to %s (%s).", member.Name, member.Email, binding.Role),
			})
		}
	}
	for _, a := range activities {
		if _, err := s.activityManager.CreateActivity(ctx, a, &activity.Metadata{}); err != nil {
			log.Warn("Failed to create project activity", zap.Error(err))
		}
	}
}

// GetDeploymentConfig returns the deployment config for a project.
func (s *ProjectService) GetDeploymentConfig(ctx context.Context, request *v1pb.GetDeploymentConfigRequest) (*v1pb.DeploymentConfig, error) {
	projectID, err := trimSuffixAndGetProjectID(request.Name, deploymentConfigSuffix)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", request.Name)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "project %q has been deleted", request.Name)
	}

	deploymentConfig, err := s.store.GetDeploymentConfigV2(ctx, project.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if deploymentConfig == nil {
		return nil, status.Errorf(codes.NotFound, "deployment config %q not found", request.Name)
	}

	return convertToDeploymentConfig(project.ResourceID, deploymentConfig), nil
}

// UpdateDeploymentConfig updates the deployment config for a project.
func (s *ProjectService) UpdateDeploymentConfig(ctx context.Context, request *v1pb.UpdateDeploymentConfigRequest) (*v1pb.DeploymentConfig, error) {
	if request.Config == nil {
		return nil, status.Errorf(codes.InvalidArgument, "deployment config is required")
	}
	projectID, err := trimSuffixAndGetProjectID(request.Config.Name, deploymentConfigSuffix)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "project %q has been deleted", projectID)
	}

	storeDeploymentConfig, err := validateAndConvertToStoreDeploymentSchedule(request.Config)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	deploymentConfig, err := s.store.UpsertDeploymentConfigV2(ctx, project.UID, ctx.Value(common.PrincipalIDContextKey).(int), storeDeploymentConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToDeploymentConfig(project.ResourceID, deploymentConfig), nil
}

// AddWebhook adds a webhook to a given project.
func (s *ProjectService) AddWebhook(ctx context.Context, request *v1pb.AddWebhookRequest) (*v1pb.Project, error) {
	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get workspace setting: %v", err)
	}
	if setting.ExternalUrl == "" {
		return nil, status.Errorf(codes.FailedPrecondition, setupExternalURLError)
	}

	projectID, err := getProjectID(request.Project)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", request.Project)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "project %q has been deleted", request.Project)
	}

	create, err := convertToStoreProjectWebhookMessage(request.Webhook)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	if _, err := s.store.CreateProjectWebhookV2(ctx, ctx.Value(common.PrincipalIDContextKey).(int), project.UID, project.ResourceID, create); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	project, err = s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToProject(project), nil
}

// UpdateWebhook updates a webhook.
func (s *ProjectService) UpdateWebhook(ctx context.Context, request *v1pb.UpdateWebhookRequest) (*v1pb.Project, error) {
	projectID, webhookID, err := getProjectIDWebhookID(request.Webhook.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	webhookIDInt, err := strconv.Atoi(webhookID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid webhook id %q", webhookID)
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectID)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "project %q has been deleted", projectID)
	}

	webhook, err := s.store.GetProjectWebhookV2(ctx, &store.FindProjectWebhookMessage{
		ProjectID: &project.UID,
		ID:        &webhookIDInt,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if webhook == nil {
		return nil, status.Errorf(codes.NotFound, "webhook %q not found", request.Webhook.Url)
	}

	update := &store.UpdateProjectWebhookMessage{}
	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "type":
			return nil, status.Errorf(codes.InvalidArgument, "type cannot be updated")
		case "title":
			update.Title = &request.Webhook.Title
		case "url":
			update.URL = &request.Webhook.Url
		case "notification_type":
			types, err := convertToActivityTypeStrings(request.Webhook.NotificationTypes)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
			update.ActivityList = types
		default:
			return nil, status.Errorf(codes.InvalidArgument, "invalid field %q", path)
		}
	}

	if _, err := s.store.UpdateProjectWebhookV2(ctx, ctx.Value(common.PrincipalIDContextKey).(int), project.UID, project.ResourceID, webhook.ID, update); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	project, err = s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToProject(project), nil
}

// RemoveWebhook removes a webhook from a given project.
func (s *ProjectService) RemoveWebhook(ctx context.Context, request *v1pb.RemoveWebhookRequest) (*v1pb.Project, error) {
	projectID, webhookID, err := getProjectIDWebhookID(request.Webhook.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	webhookIDInt, err := strconv.Atoi(webhookID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid webhook id %q", webhookID)
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", webhookID)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "project %q has been deleted", projectID)
	}

	webhook, err := s.store.GetProjectWebhookV2(ctx, &store.FindProjectWebhookMessage{
		ProjectID: &project.UID,
		ID:        &webhookIDInt,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if webhook == nil {
		return nil, status.Errorf(codes.NotFound, "webhook %q not found", request.Webhook.Url)
	}

	if err := s.store.DeleteProjectWebhookV2(ctx, project.UID, project.ResourceID, webhook.ID); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	project, err = s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToProject(project), nil
}

// TestWebhook tests a webhook.
func (s *ProjectService) TestWebhook(ctx context.Context, request *v1pb.TestWebhookRequest) (*v1pb.TestWebhookResponse, error) {
	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get workspace setting: %v", err)
	}
	if setting.ExternalUrl == "" {
		return nil, status.Errorf(codes.FailedPrecondition, setupExternalURLError)
	}

	projectID, err := getProjectID(request.Project)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", request.Project)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "project %q has been deleted", request.Project)
	}

	webhook, err := s.store.GetProjectWebhookV2(ctx, &store.FindProjectWebhookMessage{
		ProjectID: &project.UID,
		URL:       &request.Webhook.Url,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if webhook == nil {
		return nil, status.Errorf(codes.NotFound, "webhook %q not found", request.Webhook.Url)
	}

	err = webhookPlugin.Post(
		webhook.Type,
		webhookPlugin.Context{
			URL:          webhook.URL,
			Level:        webhookPlugin.WebhookInfo,
			ActivityType: string(api.ActivityIssueCreate),
			Title:        fmt.Sprintf("Test webhook %q", webhook.Title),
			Description:  "This is a test",
			Link:         fmt.Sprintf("%s/project/%s/webhook/%s", setting.ExternalUrl, fmt.Sprintf("%s-%d", slug.Make(project.Title), project.UID), fmt.Sprintf("%s-%d", slug.Make(webhook.Title), webhook.ID)),
			CreatorID:    api.SystemBotID,
			CreatorName:  "Bytebase",
			CreatorEmail: api.SystemBotEmail,
			CreatedTs:    time.Now().Unix(),
			Project:      &webhookPlugin.Project{Name: project.Title},
		},
	)

	return &v1pb.TestWebhookResponse{Error: err.Error()}, nil
}

// CreateDatabaseGroup creates a database group.
func (s *ProjectService) CreateDatabaseGroup(ctx context.Context, request *v1pb.CreateDatabaseGroupRequest) (*v1pb.DatabaseGroup, error) {
	projectResourceID, err := getProjectID(request.Parent)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectResourceID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", request.Parent)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.InvalidArgument, "project %q has been deleted", request.Parent)
	}

	// The id value should be 4-63 characters, and valid characters are /[a-z][0-9]-/.
	if len(request.DatabaseGroupId) < 4 || len(request.DatabaseGroupId) > 63 {
		return nil, status.Errorf(codes.InvalidArgument, "database group id %q must be between 4 and 63 characters", request.DatabaseGroupId)
	}
	for _, c := range request.DatabaseGroupId {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			return nil, status.Errorf(codes.InvalidArgument, "invalid database group id %q", request.DatabaseGroupId)
		}
	}
	if request.DatabaseGroup.DatabasePlaceholder == "" {
		return nil, status.Errorf(codes.InvalidArgument, "database group database placeholder is required")
	}

	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	storeDatabaseGroup := &store.DatabaseGroupMessage{
		CreatorID:         principalID,
		ProjectResourceID: project.ResourceID,
		Placeholder:       request.DatabaseGroup.DatabasePlaceholder,
		Expression:        request.DatabaseGroup.DatabaseExpr,
	}
	databaseGroup, err := s.store.CreateDatabaseGroup(ctx, principalID, storeDatabaseGroup)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertStoreToAPIDatabaseGroup(ctx, databaseGroup), nil
}

func convertStoreToAPIDatabaseGroup(ctx context.Context, databaseGroup *store.DatabaseGroupMessage) *v1pb.DatabaseGroup {
	return &v1pb.DatabaseGroup{
		Name:                fmt.Sprintf("%s%s/%s%s", projectNamePrefix, databaseGroup.ProjectResourceID, databaseGroupNamePrefix, databaseGroup.ResourceID),
		DatabasePlaceholder: databaseGroup.Placeholder,
		DatabaseExpr:        databaseGroup.Expression,
	}
}

func convertToStoreProjectWebhookMessage(webhook *v1pb.Webhook) (*store.ProjectWebhookMessage, error) {
	tp, err := convertToAPIWebhookTypeString(webhook.Type)
	if err != nil {
		return nil, err
	}
	activityTypes, err := convertToActivityTypeStrings(webhook.NotificationTypes)
	if err != nil {
		return nil, err
	}
	return &store.ProjectWebhookMessage{
		Type:         tp,
		URL:          webhook.Url,
		Title:        webhook.Title,
		ActivityList: activityTypes,
	}, nil
}

func convertToActivityTypeStrings(types []v1pb.Activity_Type) ([]string, error) {
	var result []string
	for _, tp := range types {
		switch tp {
		case v1pb.Activity_TYPE_UNSPECIFIED:
			return nil, common.Errorf(common.Invalid, "activity type must not be unspecified")
		case v1pb.Activity_TYPE_ISSUE_CREATE:
			result = append(result, string(api.ActivityIssueCreate))
		case v1pb.Activity_TYPE_ISSUE_COMMENT_CREATE:
			result = append(result, string(api.ActivityIssueCommentCreate))
		case v1pb.Activity_TYPE_ISSUE_FIELD_UPDATE:
			result = append(result, string(api.ActivityIssueFieldUpdate))
		case v1pb.Activity_TYPE_ISSUE_STATUS_UPDATE:
			result = append(result, string(api.ActivityIssueStatusUpdate))
		case v1pb.Activity_TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE:
			result = append(result, string(api.ActivityPipelineStageStatusUpdate))
		case v1pb.Activity_TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE:
			result = append(result, string(api.ActivityPipelineTaskStatusUpdate))
		case v1pb.Activity_TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT:
			result = append(result, string(api.ActivityPipelineTaskFileCommit))
		case v1pb.Activity_TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE:
			result = append(result, string(api.ActivityPipelineTaskStatementUpdate))
		case v1pb.Activity_TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE:
			result = append(result, string(api.ActivityPipelineTaskEarliestAllowedTimeUpdate))
		case v1pb.Activity_TYPE_MEMBER_CREATE:
			result = append(result, string(api.ActivityMemberCreate))
		case v1pb.Activity_TYPE_MEMBER_ROLE_UPDATE:
			result = append(result, string(api.ActivityMemberRoleUpdate))
		case v1pb.Activity_TYPE_MEMBER_ACTIVATE:
			result = append(result, string(api.ActivityMemberActivate))
		case v1pb.Activity_TYPE_MEMBER_DEACTIVATE:
			result = append(result, string(api.ActivityMemberDeactivate))
		case v1pb.Activity_TYPE_PROJECT_REPOSITORY_PUSH:
			result = append(result, string(api.ActivityProjectRepositoryPush))
		case v1pb.Activity_TYPE_PROJECT_DATABASE_TRANSFER:
			result = append(result, string(api.ActivityProjectDatabaseTransfer))
		case v1pb.Activity_TYPE_PROJECT_MEMBER_CREATE:
			result = append(result, string(api.ActivityProjectMemberCreate))
		case v1pb.Activity_TYPE_PROJECT_MEMBER_DELETE:
			result = append(result, string(api.ActivityProjectMemberDelete))
		case v1pb.Activity_TYPE_PROJECT_MEMBER_ROLE_UPDATE:
			result = append(result, string(api.ActivityProjectMemberRoleUpdate))
		case v1pb.Activity_TYPE_SQL_EDITOR_QUERY:
			result = append(result, string(api.ActivitySQLEditorQuery))
		case v1pb.Activity_TYPE_DATABASE_RECOVERY_PITR_DONE:
			result = append(result, string(api.ActivityDatabaseRecoveryPITRDone))
		default:
			return nil, common.Errorf(common.Invalid, "unsupported activity type: %v", tp)
		}
	}
	return result, nil
}

func convertNotificationTypeStrings(types []string) []v1pb.Activity_Type {
	var result []v1pb.Activity_Type
	for _, tp := range types {
		switch tp {
		case string(api.ActivityIssueCreate):
			result = append(result, v1pb.Activity_TYPE_ISSUE_CREATE)
		case string(api.ActivityIssueCommentCreate):
			result = append(result, v1pb.Activity_TYPE_ISSUE_COMMENT_CREATE)
		case string(api.ActivityIssueFieldUpdate):
			result = append(result, v1pb.Activity_TYPE_ISSUE_FIELD_UPDATE)
		case string(api.ActivityIssueStatusUpdate):
			result = append(result, v1pb.Activity_TYPE_ISSUE_STATUS_UPDATE)
		case string(api.ActivityPipelineStageStatusUpdate):
			result = append(result, v1pb.Activity_TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE)
		case string(api.ActivityPipelineTaskStatusUpdate):
			result = append(result, v1pb.Activity_TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE)
		case string(api.ActivityPipelineTaskFileCommit):
			result = append(result, v1pb.Activity_TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT)
		case string(api.ActivityPipelineTaskStatementUpdate):
			result = append(result, v1pb.Activity_TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE)
		case string(api.ActivityPipelineTaskEarliestAllowedTimeUpdate):
			result = append(result, v1pb.Activity_TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE)
		case string(api.ActivityMemberCreate):
			result = append(result, v1pb.Activity_TYPE_MEMBER_CREATE)
		case string(api.ActivityMemberRoleUpdate):
			result = append(result, v1pb.Activity_TYPE_MEMBER_ROLE_UPDATE)
		case string(api.ActivityMemberActivate):
			result = append(result, v1pb.Activity_TYPE_MEMBER_ACTIVATE)
		case string(api.ActivityMemberDeactivate):
			result = append(result, v1pb.Activity_TYPE_MEMBER_DEACTIVATE)
		case string(api.ActivityProjectRepositoryPush):
			result = append(result, v1pb.Activity_TYPE_PROJECT_REPOSITORY_PUSH)
		case string(api.ActivityProjectDatabaseTransfer):
			result = append(result, v1pb.Activity_TYPE_PROJECT_DATABASE_TRANSFER)
		case string(api.ActivityProjectMemberCreate):
			result = append(result, v1pb.Activity_TYPE_PROJECT_MEMBER_CREATE)
		case string(api.ActivityProjectMemberDelete):
			result = append(result, v1pb.Activity_TYPE_PROJECT_MEMBER_DELETE)
		case string(api.ActivityProjectMemberRoleUpdate):
			result = append(result, v1pb.Activity_TYPE_PROJECT_MEMBER_ROLE_UPDATE)
		case string(api.ActivitySQLEditorQuery):
			result = append(result, v1pb.Activity_TYPE_SQL_EDITOR_QUERY)
		case string(api.ActivityDatabaseRecoveryPITRDone):
			result = append(result, v1pb.Activity_TYPE_DATABASE_RECOVERY_PITR_DONE)
		default:
			result = append(result, v1pb.Activity_TYPE_UNSPECIFIED)
		}
	}
	return result
}

func convertToAPIWebhookTypeString(tp v1pb.Webhook_Type) (string, error) {
	switch tp {
	case v1pb.Webhook_TYPE_UNSPECIFIED:
		return "", common.Errorf(common.Invalid, "webhook type must not be unspecified")
	// TODO(zp): find a better way to place the "bb.plugin.webhook.*".
	case v1pb.Webhook_TYPE_SLACK:
		return "bb.plugin.webhook.slack", nil
	case v1pb.Webhook_TYPE_DISCORD:
		return "bb.plugin.webhook.discord", nil
	case v1pb.Webhook_TYPE_TEAMS:
		return "bb.plugin.webhook.teams", nil
	case v1pb.Webhook_TYPE_DINGTALK:
		return "bb.plugin.webhook.dingtalk", nil
	case v1pb.Webhook_TYPE_FEISHU:
		return "bb.plugin.webhook.feishu", nil
	case v1pb.Webhook_TYPE_WECOM:
		return "bb.plugin.webhook.wecom", nil
	case v1pb.Webhook_TYPE_CUSTOM:
		return "bb.plugin.webhook.custom", nil
	default:
		return "", common.Errorf(common.Invalid, "webhook type %q is not supported", tp)
	}
}

func convertWebhookTypeString(tp string) v1pb.Webhook_Type {
	switch tp {
	case "bb.plugin.webhook.slack":
		return v1pb.Webhook_TYPE_SLACK
	case "bb.plugin.webhook.discord":
		return v1pb.Webhook_TYPE_DISCORD
	case "bb.plugin.webhook.teams":
		return v1pb.Webhook_TYPE_TEAMS
	case "bb.plugin.webhook.dingtalk":
		return v1pb.Webhook_TYPE_DINGTALK
	case "bb.plugin.webhook.feishu":
		return v1pb.Webhook_TYPE_FEISHU
	case "bb.plugin.webhook.wecom":
		return v1pb.Webhook_TYPE_WECOM
	case "bb.plugin.webhook.custom":
		return v1pb.Webhook_TYPE_CUSTOM
	default:
		return v1pb.Webhook_TYPE_UNSPECIFIED
	}
}

func validateAndConvertToStoreDeploymentSchedule(deployment *v1pb.DeploymentConfig) (*store.DeploymentConfigMessage, error) {
	if deployment.Schedule == nil {
		return nil, common.Errorf(common.Invalid, "schedule must not be empty")
	}
	for _, d := range deployment.Schedule.Deployments {
		if d == nil {
			return nil, common.Errorf(common.Invalid, "deployment must not be empty")
		}
		if d.Title == "" {
			return nil, common.Errorf(common.Invalid, "Deployment name must not be empty")
		}
		hasEnv := false
		for _, e := range d.Spec.LabelSelector.MatchExpressions {
			if e == nil {
				return nil, common.Errorf(common.Invalid, "label selector expression must not be empty")
			}
			switch e.Operator {
			case v1pb.OperatorType_OPERATOR_TYPE_IN:
				if len(e.Values) == 0 {
					return nil, common.Errorf(common.Invalid, "expression key %q with %q operator should have at least one value", e.Key, e.Operator)
				}
			case v1pb.OperatorType_OPERATOR_TYPE_EXISTS:
				if len(e.Values) > 0 {
					return nil, common.Errorf(common.Invalid, "expression key %q with %q operator shouldn't have values", e.Key, e.Operator)
				}
			default:
				return nil, common.Errorf(common.Invalid, "expression key %q has invalid operator %q", e.Key, e.Operator)
			}
			if e.Key == api.EnvironmentLabelKey {
				hasEnv = true
				if e.Operator != v1pb.OperatorType_OPERATOR_TYPE_IN || len(e.Values) != 1 {
					return nil, common.Errorf(common.Invalid, "label %q should must use operator %q with exactly one value", api.EnvironmentLabelKey, v1pb.OperatorType_OPERATOR_TYPE_IN)
				}
			}
		}
		if !hasEnv {
			return nil, common.Errorf(common.Invalid, "deployment should contain %q label", api.EnvironmentLabelKey)
		}
	}
	return convertToStoreDeploymentConfig(deployment)
}

func (s *ProjectService) getProjectMessage(ctx context.Context, name string) (*store.ProjectMessage, error) {
	projectID, err := getProjectID(name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	var project *store.ProjectMessage
	projectUID, isNumber := isNumber(projectID)
	if isNumber {
		project, err = s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			UID:         &projectUID,
			ShowDeleted: true,
		})
	} else {
		project, err = s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID:  &projectID,
			ShowDeleted: true,
		})
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", name)
	}

	return project, nil
}

func convertToIamPolicy(iamPolicy *store.IAMPolicyMessage) *v1pb.IamPolicy {
	var bindings []*v1pb.Binding

	for _, binding := range iamPolicy.Bindings {
		var members []string
		for _, member := range binding.Members {
			members = append(members, fmt.Sprintf("user:%s", member.Email))
		}
		bindings = append(bindings, &v1pb.Binding{
			Role:      convertToProjectRole(binding.Role),
			Members:   members,
			Condition: binding.Condition,
		})
	}
	return &v1pb.IamPolicy{
		Bindings: bindings,
	}
}

// convertToIAMPolicyMessage will convert the IAM policy to IAM policy message.
func (s *ProjectService) convertToIAMPolicyMessage(ctx context.Context, iamPolicy *v1pb.IamPolicy) (*store.IAMPolicyMessage, error) {
	var bindings []*store.PolicyBinding
	for _, binding := range iamPolicy.Bindings {
		var users []*store.UserMessage
		role, err := convertProjectRole(binding.Role)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, err.Error())
		}
		for _, member := range binding.Members {
			email := strings.TrimPrefix(member, "user:")
			user, err := s.store.GetUser(ctx, &store.FindUserMessage{
				Email:       &email,
				ShowDeleted: true,
			})
			if err != nil {
				return nil, status.Errorf(codes.Internal, err.Error())
			}
			if user == nil {
				return nil, status.Errorf(codes.NotFound, "user %q not found", member)
			}
			users = append(users, user)
		}

		bindings = append(bindings, &store.PolicyBinding{
			Role:      role,
			Members:   users,
			Condition: binding.Condition,
		})
	}
	return &store.IAMPolicyMessage{
		Bindings: bindings,
	}, nil
}

func convertToProjectRole(role api.Role) string {
	return fmt.Sprintf("%s%s", rolePrefix, role)
}

func convertProjectRole(role string) (api.Role, error) {
	roleID, err := getRoleID(role)
	if err != nil {
		return api.Role(""), errors.Wrapf(err, "invalid project role %q", role)
	}
	return api.Role(roleID), nil
}

func convertToProject(projectMessage *store.ProjectMessage) *v1pb.Project {
	workflow := v1pb.Workflow_WORKFLOW_UNSPECIFIED
	switch projectMessage.Workflow {
	case api.UIWorkflow:
		workflow = v1pb.Workflow_UI
	case api.VCSWorkflow:
		workflow = v1pb.Workflow_VCS
	}

	visibility := v1pb.Visibility_VISIBILITY_UNSPECIFIED
	switch projectMessage.Visibility {
	case api.Private:
		visibility = v1pb.Visibility_VISIBILITY_PRIVATE
	case api.Public:
		visibility = v1pb.Visibility_VISIBILITY_PUBLIC
	}

	tenantMode := v1pb.TenantMode_TENANT_MODE_UNSPECIFIED
	switch projectMessage.TenantMode {
	case api.TenantModeDisabled:
		tenantMode = v1pb.TenantMode_TENANT_MODE_DISABLED
	case api.TenantModeTenant:
		tenantMode = v1pb.TenantMode_TENANT_MODE_ENABLED
	}

	schemaChange := v1pb.SchemaChange_SCHEMA_CHANGE_UNSPECIFIED
	switch projectMessage.SchemaChangeType {
	case api.ProjectSchemaChangeTypeDDL:
		schemaChange = v1pb.SchemaChange_DDL
	case api.ProjectSchemaChangeTypeSDL:
		schemaChange = v1pb.SchemaChange_SDL
	}

	var projectWebhooks []*v1pb.Webhook
	for _, webhook := range projectMessage.Webhooks {
		projectWebhooks = append(projectWebhooks, &v1pb.Webhook{
			Name:              fmt.Sprintf("%s%s/%s%d", projectNamePrefix, projectMessage.ResourceID, webhookIDPrefix, webhook.ID),
			Type:              convertWebhookTypeString(webhook.Type),
			Title:             webhook.Title,
			Url:               webhook.URL,
			NotificationTypes: convertNotificationTypeStrings(webhook.ActivityList),
		})
	}

	return &v1pb.Project{
		Name:           fmt.Sprintf("%s%s", projectNamePrefix, projectMessage.ResourceID),
		Uid:            fmt.Sprintf("%d", projectMessage.UID),
		State:          convertDeletedToState(projectMessage.Deleted),
		Title:          projectMessage.Title,
		Key:            projectMessage.Key,
		Workflow:       workflow,
		Visibility:     visibility,
		TenantMode:     tenantMode,
		DbNameTemplate: projectMessage.DBNameTemplate,
		// TODO(d): schema_version_type for project.
		SchemaVersion: v1pb.SchemaVersion_SCHEMA_VERSION_UNSPECIFIED,
		SchemaChange:  schemaChange,
		Webhooks:      projectWebhooks,
	}
}

func convertToProjectWorkflowType(workflow v1pb.Workflow) api.ProjectWorkflowType {
	switch workflow {
	case v1pb.Workflow_UI:
		return api.UIWorkflow
	case v1pb.Workflow_VCS:
		return api.VCSWorkflow
	default:
		// Default is UI workflow.
		return api.UIWorkflow
	}
}

func convertToProjectVisibility(visibility v1pb.Visibility) api.ProjectVisibility {
	switch visibility {
	case v1pb.Visibility_VISIBILITY_PRIVATE:
		return api.Private
	case v1pb.Visibility_VISIBILITY_PUBLIC:
		return api.Public
	default:
		// Default is public.
		return api.Public
	}
}

func convertToProjectTenantMode(tenantMode v1pb.TenantMode) api.ProjectTenantMode {
	switch tenantMode {
	case v1pb.TenantMode_TENANT_MODE_DISABLED:
		return api.TenantModeDisabled
	case v1pb.TenantMode_TENANT_MODE_ENABLED:
		return api.TenantModeTenant
	default:
		return api.TenantModeDisabled
	}
}

func convertToProjectSchemaChangeType(schemaChange v1pb.SchemaChange) api.ProjectSchemaChangeType {
	switch schemaChange {
	case v1pb.SchemaChange_DDL:
		return api.ProjectSchemaChangeTypeDDL
	case v1pb.SchemaChange_SDL:
		return api.ProjectSchemaChangeTypeSDL
	default:
		return api.ProjectSchemaChangeTypeDDL
	}
}

func convertToProjectMessage(resourceID string, project *v1pb.Project) (*store.ProjectMessage, error) {
	workflow := convertToProjectWorkflowType(project.Workflow)
	visibility := convertToProjectVisibility(project.Visibility)
	tenantMode := convertToProjectTenantMode(project.TenantMode)
	schemaChange := convertToProjectSchemaChangeType(project.SchemaChange)

	return &store.ProjectMessage{
		ResourceID:       resourceID,
		Title:            project.Title,
		Key:              project.Key,
		Workflow:         workflow,
		Visibility:       visibility,
		TenantMode:       tenantMode,
		DBNameTemplate:   project.DbNameTemplate,
		SchemaChangeType: schemaChange,
	}, nil
}

func convertToDeploymentConfig(projectID string, deploymentConfig *store.DeploymentConfigMessage) *v1pb.DeploymentConfig {
	resourceName := fmt.Sprintf("projects/%s/deploymentConfig", projectID)
	return &v1pb.DeploymentConfig{
		Name:     resourceName,
		Title:    deploymentConfig.Name,
		Schedule: convertToSchedule(deploymentConfig.Schedule),
	}
}

func convertToStoreDeploymentConfig(deploymentConfig *v1pb.DeploymentConfig) (*store.DeploymentConfigMessage, error) {
	schedule, err := convertToStoreSchedule(deploymentConfig.Schedule)
	if err != nil {
		return nil, err
	}

	return &store.DeploymentConfigMessage{
		Name:     deploymentConfig.Title,
		Schedule: schedule,
	}, nil
}

func convertToSchedule(schedule *store.Schedule) *v1pb.Schedule {
	var ds []*v1pb.ScheduleDeployment
	for _, d := range schedule.Deployments {
		ds = append(ds, convertToDeployment(d))
	}
	return &v1pb.Schedule{
		Deployments: ds,
	}
}

func convertToStoreSchedule(schedule *v1pb.Schedule) (*store.Schedule, error) {
	var ds []*store.Deployment
	for _, d := range schedule.Deployments {
		deployment, err := convertToStoreDeployment(d)
		if err != nil {
			return nil, err
		}
		ds = append(ds, deployment)
	}
	return &store.Schedule{
		Deployments: ds,
	}, nil
}

func convertToDeployment(deployment *store.Deployment) *v1pb.ScheduleDeployment {
	return &v1pb.ScheduleDeployment{
		Title: deployment.Name,
		Spec:  convertToSpec(deployment.Spec),
	}
}

func convertToStoreDeployment(deployment *v1pb.ScheduleDeployment) (*store.Deployment, error) {
	spec, err := convertToStoreSpec(deployment.Spec)
	if err != nil {
		return nil, err
	}

	return &store.Deployment{
		Name: deployment.Title,
		Spec: spec,
	}, nil
}

func convertToSpec(spec *store.DeploymentSpec) *v1pb.DeploymentSpec {
	return &v1pb.DeploymentSpec{
		LabelSelector: convertToLabelSelector(spec.Selector),
	}
}

func convertToStoreSpec(spec *v1pb.DeploymentSpec) (*store.DeploymentSpec, error) {
	selector, err := convertToStoreLabelSelector(spec.LabelSelector)
	if err != nil {
		return nil, err
	}
	return &store.DeploymentSpec{
		Selector: selector,
	}, nil
}

func convertToLabelSelector(selector *store.LabelSelector) *v1pb.LabelSelector {
	var exprs []*v1pb.LabelSelectorRequirement
	for _, expr := range selector.MatchExpressions {
		exprs = append(exprs, convertToLabelSelectorRequirement(expr))
	}

	return &v1pb.LabelSelector{
		MatchExpressions: exprs,
	}
}

func convertToStoreLabelSelector(selector *v1pb.LabelSelector) (*store.LabelSelector, error) {
	var exprs []*store.LabelSelectorRequirement
	for _, expr := range selector.MatchExpressions {
		requirement, err := convertToStoreLabelSelectorRequirement(expr)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, requirement)
	}
	return &store.LabelSelector{
		MatchExpressions: exprs,
	}, nil
}

func convertToLabelSelectorRequirement(requirements *store.LabelSelectorRequirement) *v1pb.LabelSelectorRequirement {
	return &v1pb.LabelSelectorRequirement{
		Key:      requirements.Key,
		Operator: convertToLabelSelectorOperator(requirements.Operator),
		Values:   requirements.Values,
	}
}

func convertToStoreLabelSelectorRequirement(requirements *v1pb.LabelSelectorRequirement) (*store.LabelSelectorRequirement, error) {
	op, err := convertToStoreLabelSelectorOperator(requirements.Operator)
	if err != nil {
		return nil, err
	}
	return &store.LabelSelectorRequirement{
		Key:      requirements.Key,
		Operator: op,
		Values:   requirements.Values,
	}, nil
}

func convertToLabelSelectorOperator(operator store.OperatorType) v1pb.OperatorType {
	switch operator {
	case store.InOperatorType:
		return v1pb.OperatorType_OPERATOR_TYPE_IN
	case store.ExistsOperatorType:
		return v1pb.OperatorType_OPERATOR_TYPE_EXISTS
	}
	return v1pb.OperatorType_OPERATOR_TYPE_UNSPECIFIED
}

func convertToStoreLabelSelectorOperator(operator v1pb.OperatorType) (store.OperatorType, error) {
	switch operator {
	case v1pb.OperatorType_OPERATOR_TYPE_IN:
		return store.InOperatorType, nil
	case v1pb.OperatorType_OPERATOR_TYPE_EXISTS:
		return store.ExistsOperatorType, nil
	}
	return store.OperatorType(""), errors.Errorf("invalid operator type: %v", operator)
}

func validateIAMPolicy(policy *v1pb.IamPolicy, roles []*v1pb.Role) error {
	if policy == nil {
		return errors.Errorf("IAM Policy is required")
	}
	return validateBindings(policy.Bindings, roles)
}

func validateBindings(bindings []*v1pb.Binding, roles []*v1pb.Role) error {
	if len(bindings) == 0 {
		return errors.Errorf("IAM Binding is required")
	}
	projectRoleMap := make(map[string]bool)
	existingRoles := make(map[string]bool)
	for _, role := range roles {
		existingRoles[role.Name] = true
	}
	for _, binding := range bindings {
		if binding.Role == "" {
			return errors.Errorf("IAM Binding role is required")
		}
		if !existingRoles[binding.Role] {
			return errors.Errorf("IAM Binding role %s does not exist", binding.Role)
		}
		// Each of the bindings must contain at least one member.
		if len(binding.Members) == 0 {
			return errors.Errorf("Each IAM binding must have at least one member")
		}

		// Users within each binding must be unique.
		userMap := make(map[string]bool)
		for _, member := range binding.Members {
			if _, ok := userMap[member]; ok {
				return errors.Errorf("duplicate user %s in role %s", member, binding.Role)
			}
			userMap[member] = true
			if err := validateMember(member); err != nil {
				return err
			}
		}
		projectRoleMap[binding.Role] = true
	}
	// Must contain one owner binding.
	if _, ok := projectRoleMap["roles/OWNER"]; !ok {
		return errors.Errorf("IAM Policy must have at least one binding with role PROJECT_OWNER")
	}
	return nil
}

func validateMember(member string) error {
	userIdentifierMap := map[string]bool{
		"user:": true,
	}
	for prefix := range userIdentifierMap {
		if strings.HasPrefix(member, prefix) && len(member[len(prefix):]) > 0 {
			return nil
		}
	}
	return errors.Errorf("invalid user %s", member)
}

func isProjectMember(policy *store.IAMPolicyMessage, userID int) bool {
	for _, binding := range policy.Bindings {
		for _, member := range binding.Members {
			if member.ID == userID {
				return true
			}
		}
	}
	return false
}
