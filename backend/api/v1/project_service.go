package v1

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	webhookplugin "github.com/bytebase/bytebase/backend/plugin/webhook"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ProjectService implements the project service.
type ProjectService struct {
	v1pb.UnimplementedProjectServiceServer
	store          *store.Store
	profile        *config.Profile
	iamManager     *iam.Manager
	licenseService enterprise.LicenseService
}

// NewProjectService creates a new ProjectService.
func NewProjectService(store *store.Store, profile *config.Profile, iamManager *iam.Manager, licenseService enterprise.LicenseService) *ProjectService {
	return &ProjectService{
		store:          store,
		profile:        profile,
		iamManager:     iamManager,
		licenseService: licenseService,
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

// SearchProjects searches all projects on which the user has bb.projects.get permission.
func (s *ProjectService) SearchProjects(ctx context.Context, request *v1pb.SearchProjectsRequest) (*v1pb.SearchProjectsResponse, error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, status.Errorf(codes.Internal, "user not found")
	}

	projects, err := s.store.ListProjectV2(ctx, &store.FindProjectMessage{ShowDeleted: request.ShowDeleted})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	ok, err = s.iamManager.CheckPermission(ctx, iam.PermissionProjectsGet, user)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check permission, error %v", err)
	}
	if !ok {
		var ps []*store.ProjectMessage
		for _, project := range projects {
			ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionProjectsGet, user, project.ResourceID)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to check permission for project %q: %v", project.ResourceID, err)
			}
			if ok {
				ps = append(ps, project)
			}
		}
		projects = ps
	}

	response := &v1pb.SearchProjectsResponse{}
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

	setting, err := s.store.GetDataClassificationSetting(ctx)
	if err != nil {
		slog.Error("failed to find classification setting", log.BBError(err))
	}
	if setting != nil && len(setting.Configs) != 0 {
		projectMessage.DataClassificationConfigID = setting.Configs[0].Id
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
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
		return nil, status.Errorf(codes.NotFound, "project %q has been deleted", request.Project.Name)
	}
	if project.ResourceID == api.DefaultProjectID {
		return nil, status.Errorf(codes.InvalidArgument, "default project cannot be updated")
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	patch := &store.UpdateProjectMessage{
		UpdaterID:  principalID,
		ResourceID: project.ResourceID,
	}

	for _, path := range request.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Title = &request.Project.Title
		case "key":
			patch.Key = &request.Project.Key
		case "data_classification_config_id":
			setting, err := s.store.GetDataClassificationSetting(ctx)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to get data classification setting")
			}
			existConfig := false
			for _, config := range setting.Configs {
				if config.Id == request.Project.DataClassificationConfigId {
					existConfig = true
					break
				}
			}
			if !existConfig {
				return nil, status.Errorf(codes.InvalidArgument, "data classification %s not exists", request.Project.DataClassificationConfigId)
			}
			patch.DataClassificationConfigID = &request.Project.DataClassificationConfigId
		case "issue_labels":
			projectSettings := project.Setting
			var issueLabels []*storepb.Label
			for _, label := range request.Project.IssueLabels {
				issueLabels = append(issueLabels, &storepb.Label{
					Value: label.Value,
					Color: label.Color,
					Group: label.Group,
				})
			}
			projectSettings.IssueLabels = issueLabels
			patch.Setting = projectSettings
		case "force_issue_labels":
			projectSettings := project.Setting
			projectSettings.ForceIssueLabels = request.Project.ForceIssueLabels
			patch.Setting = projectSettings
		case "allow_modify_statement":
			projectSettings := project.Setting
			projectSettings.AllowModifyStatement = request.Project.AllowModifyStatement
			patch.Setting = projectSettings
		case "auto_resolve_issue":
			projectSettings := project.Setting
			projectSettings.AutoResolveIssue = request.Project.AutoResolveIssue
			patch.Setting = projectSettings
		default:
			return nil, status.Errorf(codes.InvalidArgument, `unsupport update_mask "%s"`, path)
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
		return nil, status.Errorf(codes.NotFound, "project %q has been deleted", request.Name)
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
		if len(databases) > 0 {
			defaultProject := api.DefaultProjectID
			if _, err := s.store.BatchUpdateDatabaseProject(ctx, databases, defaultProject, api.SystemBotID); err != nil {
				return nil, err
			}
		}
		// We don't close the issues because they might be open still.
	} else {
		// Return the open issue error first because that's more important than transferring out databases.
		openIssues, err := s.store.ListIssueV2(ctx, &store.FindIssueMessage{ProjectIDs: &[]string{project.ResourceID}, StatusList: []api.IssueStatus{api.IssueOpen}})
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

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	if _, err := s.store.UpdateProjectV2(ctx, &store.UpdateProjectMessage{
		UpdaterID:  principalID,
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

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	project, err = s.store.UpdateProjectV2(ctx, &store.UpdateProjectMessage{
		UpdaterID:  principalID,
		ResourceID: project.ResourceID,
		Delete:     &undeletePatch,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToProject(project), nil
}

// GetIamPolicy returns the IAM policy for a project.
func (s *ProjectService) GetIamPolicy(ctx context.Context, request *v1pb.GetIamPolicyRequest) (*v1pb.IamPolicy, error) {
	projectID, err := common.GetProjectID(request.Resource)
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
		return nil, status.Errorf(codes.NotFound, "cannot found project %s", projectID)
	}

	policy, err := s.store.GetProjectIamPolicy(ctx, project.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return convertToV1IamPolicy(ctx, s.store, policy)
}

// BatchGetIamPolicy returns the IAM policy for projects in batch.
func (s *ProjectService) BatchGetIamPolicy(ctx context.Context, request *v1pb.BatchGetIamPolicyRequest) (*v1pb.BatchGetIamPolicyResponse, error) {
	resp := &v1pb.BatchGetIamPolicyResponse{}
	for _, name := range request.Names {
		projectID, err := common.GetProjectID(name)
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
			continue
		}
		policy, err := s.store.GetProjectIamPolicy(ctx, project.UID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, err.Error())
		}

		iamPolicy, err := convertToV1IamPolicy(ctx, s.store, policy)
		if err != nil {
			return nil, err
		}
		resp.PolicyResults = append(resp.PolicyResults, &v1pb.BatchGetIamPolicyResponse_PolicyResult{
			Project: name,
			Policy:  iamPolicy,
		})
	}
	return resp, nil
}

// SetIamPolicy sets the IAM policy for a project.
func (s *ProjectService) SetIamPolicy(ctx context.Context, request *v1pb.SetIamPolicyRequest) (*v1pb.IamPolicy, error) {
	var oldIamPolicyMsg *store.IamPolicyMessage

	projectID, err := common.GetProjectID(request.Resource)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	creatorUID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "cannot get principal ID from context")
	}

	if err := s.validateIAMPolicy(ctx, request.Policy); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", request.Resource)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, "project %q has been deleted", request.Resource)
	}

	policy, err := convertToStoreIamPolicy(ctx, s.store, request.Policy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	policyMessage, err := s.store.GetProjectIamPolicy(ctx, project.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to find project iam policy with error: %v", err.Error())
	}
	if request.Etag != policyMessage.Etag {
		return nil, status.Errorf(codes.Aborted, "there is concurrent update to the project iam policy, please refresh and try again.")
	}
	oldIamPolicyMsg = policyMessage

	policyPayload, err := protojson.Marshal(policy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if _, err := s.store.CreatePolicyV2(ctx, &store.PolicyMessage{
		ResourceUID:       project.UID,
		ResourceType:      api.PolicyResourceTypeProject,
		Payload:           string(policyPayload),
		Type:              api.PolicyTypeIAM,
		InheritFromParent: false,
		// Enforce cannot be false while creating a policy.
		Enforce: true,
	}, creatorUID); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	iamPolicyMessage, err := s.store.GetProjectIamPolicy(ctx, project.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	if setServiceData, ok := common.GetSetServiceDataFromContext(ctx); ok {
		deltas := findIamPolicyDeltas(oldIamPolicyMsg.Policy, iamPolicyMessage.Policy)
		p, err := convertToProtoAny(deltas)
		if err != nil {
			slog.Warn("audit: failed to convert to anypb.Any")
		}
		setServiceData(p)
	}

	return convertToV1IamPolicy(ctx, s.store, iamPolicyMessage)
}

func convertToProtoAny(i any) (*anypb.Any, error) {
	switch deltas := i.(type) {
	case []*v1pb.BindingDelta:
		auditData := v1pb.AuditData{
			PolicyDelta: &v1pb.PolicyDelta{
				BindingDeltas: deltas,
			},
		}
		return anypb.New(&auditData)
	default:
		return &anypb.Any{}, nil
	}
}

type bindMapKey struct {
	User string
	Role string
}

type condMap map[string]bool

func findIamPolicyDeltas(oriIamPolicy *storepb.IamPolicy, newIamPolicy *storepb.IamPolicy) []*v1pb.BindingDelta {
	var deltas []*v1pb.BindingDelta
	oriBindMap := make(map[bindMapKey]condMap)
	newBindMap := make(map[bindMapKey]condMap)

	// build map.
	for _, binding := range oriIamPolicy.Bindings {
		if binding.Condition == nil {
			continue
		}
		for _, mem := range binding.Members {
			key := bindMapKey{
				User: mem,
				Role: binding.Role,
			}

			exprBytes, err := protojson.Marshal(binding.Condition)
			if err != nil {
				return nil
			}
			expr := string(exprBytes)
			if condMap, ok := oriBindMap[key]; ok && condMap != nil {
				if _, ok := condMap[expr]; ok {
					continue
				}
				oriBindMap[key][expr] = true
				continue
			}
			condMap := make(condMap)
			condMap[expr] = true
			oriBindMap[key] = condMap
		}
	}

	// find added items.
	for _, binding := range newIamPolicy.Bindings {
		for _, mem := range binding.Members {
			key := bindMapKey{
				User: mem,
				Role: binding.Role,
			}
			exprBytes, err := protojson.Marshal(binding.Condition)
			if err != nil {
				return nil
			}
			expr := string(exprBytes)

			// ensure the array is unique.
			if condMap, ok := newBindMap[key]; ok && condMap != nil {
				if _, ok := condMap[expr]; ok {
					continue
				}
			}
			tmpCondMap := make(condMap)
			tmpCondMap[expr] = true
			newBindMap[key] = tmpCondMap

			if condMap, ok := oriBindMap[key]; ok && condMap != nil {
				if _, ok := condMap[expr]; ok {
					delete(oriBindMap[key], expr)
					continue
				}
			}
			deltas = append(deltas, &v1pb.BindingDelta{
				Action:    v1pb.BindingDelta_ADD,
				Member:    mem,
				Role:      binding.Role,
				Condition: binding.Condition,
			})
		}
	}

	// find removed items.
	for bindMapKey, condMap := range oriBindMap {
		for cond := range condMap {
			expr := &expr.Expr{}
			if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(cond), expr); err != nil {
				return nil
			}
			deltas = append(deltas, &v1pb.BindingDelta{
				Action:    v1pb.BindingDelta_REMOVE,
				Member:    bindMapKey.User,
				Role:      bindMapKey.Role,
				Condition: expr,
			})
		}
	}

	return deltas
}

// GetDeploymentConfig returns the deployment config for a project.
func (s *ProjectService) GetDeploymentConfig(ctx context.Context, request *v1pb.GetDeploymentConfigRequest) (*v1pb.DeploymentConfig, error) {
	projectID, _, err := common.GetProjectIDDeploymentConfigID(request.Name)
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
	if request.DeploymentConfig == nil {
		return nil, status.Errorf(codes.InvalidArgument, "deployment config is required")
	}
	projectID, _, err := common.GetProjectIDDeploymentConfigID(request.DeploymentConfig.Name)
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
		return nil, status.Errorf(codes.NotFound, "project %q has been deleted", projectID)
	}

	storeDeploymentConfig, err := validateAndConvertToStoreDeploymentSchedule(request.DeploymentConfig)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	deploymentConfig, err := s.store.UpsertDeploymentConfigV2(ctx, project.UID, principalID, storeDeploymentConfig)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	return convertToDeploymentConfig(project.ResourceID, deploymentConfig), nil
}

// AddWebhook adds a webhook to a given project.
func (s *ProjectService) AddWebhook(ctx context.Context, request *v1pb.AddWebhookRequest) (*v1pb.Project, error) {
	projectID, err := common.GetProjectID(request.Project)
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
		return nil, status.Errorf(codes.NotFound, "project %q has been deleted", request.Project)
	}

	create, err := convertToStoreProjectWebhookMessage(request.Webhook)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	if _, err := s.store.CreateProjectWebhookV2(ctx, principalID, project.UID, project.ResourceID, create); err != nil {
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
	projectID, webhookID, err := common.GetProjectIDWebhookID(request.Webhook.Name)
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
		return nil, status.Errorf(codes.NotFound, "project %q has been deleted", projectID)
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
			if len(types) == 0 {
				return nil, status.Errorf(codes.InvalidArgument, "notification types should not be empty")
			}
			update.ActivityList = types
		case "direct_message":
			update.Payload = &storepb.ProjectWebhookPayload{
				DirectMessage: request.Webhook.DirectMessage,
			}
		default:
			return nil, status.Errorf(codes.InvalidArgument, "invalid field %q", path)
		}
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	if _, err := s.store.UpdateProjectWebhookV2(ctx, principalID, project.ResourceID, webhook.ID, update); err != nil {
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
	projectID, webhookID, err := common.GetProjectIDWebhookID(request.Webhook.Name)
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
		return nil, status.Errorf(codes.NotFound, "project %q has been deleted", projectID)
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

	if err := s.store.DeleteProjectWebhookV2(ctx, project.ResourceID, webhook.ID); err != nil {
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

	projectID, err := common.GetProjectID(request.Project)
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
		return nil, status.Errorf(codes.NotFound, "project %q has been deleted", request.Project)
	}

	webhook, err := convertToStoreProjectWebhookMessage(request.Webhook)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	resp := &v1pb.TestWebhookResponse{}
	err = webhookplugin.Post(
		webhook.Type,
		webhookplugin.Context{
			URL:          webhook.URL,
			Level:        webhookplugin.WebhookInfo,
			ActivityType: string(api.ActivityIssueCreate),
			Title:        fmt.Sprintf("Test webhook %q", webhook.Title),
			TitleZh:      fmt.Sprintf("测试 webhook %q", webhook.Title),
			Description:  "This is a test",
			Link:         fmt.Sprintf("%s/project/%s/webhook/%s", setting.ExternalUrl, fmt.Sprintf("%s-%d", slug.Make(project.Title), project.UID), fmt.Sprintf("%s-%d", slug.Make(webhook.Title), webhook.ID)),
			CreatorID:    api.SystemBotID,
			CreatorName:  "Bytebase",
			CreatorEmail: s.store.GetSystemBotUser(ctx).Email,
			CreatedTs:    time.Now().Unix(),
			Issue: &webhookplugin.Issue{
				ID:          1,
				Name:        "Test issue",
				Status:      "OPEN",
				Type:        "bb.issue.database.create",
				Description: "This is a test issue",
			},
			Project: &webhookplugin.Project{Name: project.Title},
		},
	)
	if err != nil {
		resp.Error = err.Error()
	}

	return resp, nil
}

// GetProjectProtectionRules gets a project protection rules.
func (s *ProjectService) GetProjectProtectionRules(ctx context.Context, request *v1pb.GetProjectProtectionRulesRequest) (*v1pb.ProtectionRules, error) {
	projectName, err := common.TrimSuffix(request.Name, common.ProtectionRulesSuffix)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	projectResourceID, err := common.GetProjectID(projectName)
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
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectResourceID)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, "project %q has been deleted", projectResourceID)
	}

	resp := convertProtectionRules(project)
	return resp, nil
}

// UpdateProjectProtectionRules updates a project protection rules.
func (s *ProjectService) UpdateProjectProtectionRules(ctx context.Context, request *v1pb.UpdateProjectProtectionRulesRequest) (*v1pb.ProtectionRules, error) {
	if request.ProtectionRules == nil {
		return nil, status.Errorf(codes.InvalidArgument, "protection rules must be set")
	}
	projectName, err := common.TrimSuffix(request.ProtectionRules.Name, common.ProtectionRulesSuffix)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	projectResourceID, err := common.GetProjectID(projectName)
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
		return nil, status.Errorf(codes.NotFound, "project %q not found", projectResourceID)
	}
	if project.Deleted {
		return nil, status.Errorf(codes.NotFound, "project %q has been deleted", projectResourceID)
	}

	var rules []*storepb.ProtectionRule
	for _, rule := range request.ProtectionRules.Rules {
		rules = append(rules, &storepb.ProtectionRule{
			Id:           rule.Id,
			Target:       storepb.ProtectionRule_Target(rule.Target),
			NameFilter:   rule.NameFilter,
			BranchSource: storepb.ProtectionRule_BranchSource(rule.BranchSource),
			AllowedRoles: rule.AllowedRoles,
		})
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, status.Errorf(codes.Internal, "principal ID not found")
	}
	setting := project.Setting
	setting.ProtectionRules = rules
	project, err = s.store.UpdateProjectV2(ctx, &store.UpdateProjectMessage{
		UpdaterID:  principalID,
		ResourceID: project.ResourceID,
		Setting:    setting,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	resp := convertProtectionRules(project)
	return resp, nil
}

func convertProtectionRules(project *store.ProjectMessage) *v1pb.ProtectionRules {
	resp := &v1pb.ProtectionRules{
		Name: fmt.Sprintf("%s%s%s", common.ProjectNamePrefix, project.ResourceID, common.ProtectionRulesSuffix),
	}
	if project.Setting != nil {
		for _, rule := range project.Setting.ProtectionRules {
			resp.Rules = append(resp.Rules, &v1pb.ProtectionRule{
				Id:           rule.Id,
				Target:       v1pb.ProtectionRule_Target(rule.Target),
				NameFilter:   rule.NameFilter,
				BranchSource: v1pb.ProtectionRule_BranchSource(rule.GetBranchSource()),
				AllowedRoles: rule.AllowedRoles,
			})
		}
	}
	return resp
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
		Payload: &storepb.ProjectWebhookPayload{
			DirectMessage: webhook.DirectMessage,
		},
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
		case v1pb.Activity_TYPE_ISSUE_APPROVAL_NOTIFY:
			result = append(result, string(api.ActivityIssueApprovalNotify))
		case v1pb.Activity_TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE:
			result = append(result, string(api.ActivityPipelineStageStatusUpdate))
		case v1pb.Activity_TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE:
			result = append(result, string(api.ActivityPipelineTaskStatusUpdate))
		case v1pb.Activity_TYPE_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE:
			result = append(result, string(api.ActivityPipelineTaskRunStatusUpdate))
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
		case v1pb.Activity_TYPE_SQL_EDITOR_QUERY:
			result = append(result, string(api.ActivitySQLQuery))
		case v1pb.Activity_TYPE_NOTIFY_ISSUE_APPROVED:
			result = append(result, string(api.ActivityNotifyIssueApproved))
		case v1pb.Activity_TYPE_NOTIFY_PIPELINE_ROLLOUT:
			result = append(result, string(api.ActivityNotifyPipelineRollout))
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
		case string(api.ActivityIssueApprovalNotify):
			result = append(result, v1pb.Activity_TYPE_ISSUE_APPROVAL_NOTIFY)
		case string(api.ActivityPipelineStageStatusUpdate):
			result = append(result, v1pb.Activity_TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE)
		case string(api.ActivityPipelineTaskStatusUpdate):
			result = append(result, v1pb.Activity_TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE)
		case string(api.ActivityPipelineTaskRunStatusUpdate):
			result = append(result, v1pb.Activity_TYPE_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE)
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
		case string(api.ActivitySQLQuery):
			result = append(result, v1pb.Activity_TYPE_SQL_EDITOR_QUERY)
		case string(api.ActivityNotifyIssueApproved):
			result = append(result, v1pb.Activity_TYPE_NOTIFY_ISSUE_APPROVED)
		case string(api.ActivityNotifyPipelineRollout):
			result = append(result, v1pb.Activity_TYPE_NOTIFY_PIPELINE_ROLLOUT)
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
			case v1pb.OperatorType_OPERATOR_TYPE_IN, v1pb.OperatorType_OPERATOR_TYPE_NOT_IN:
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
	projectID, err := common.GetProjectID(name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	find := &store.FindProjectMessage{
		ResourceID:  &projectID,
		ShowDeleted: true,
	}
	project, err := s.store.GetProjectV2(ctx, find)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", name)
	}

	return project, nil
}

func convertToV1IamPolicy(ctx context.Context, stores *store.Store, iamPolicy *store.IamPolicyMessage) (*v1pb.IamPolicy, error) {
	var bindings []*v1pb.Binding

	for _, binding := range iamPolicy.Policy.Bindings {
		var members []string
		for _, member := range binding.Members {
			if strings.HasPrefix(member, common.UserNamePrefix) {
				userUID, err := common.GetUserID(member)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to parse user id from member %s with error: %v", member, err)
				}
				user, err := stores.GetUserByID(ctx, userUID)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to get user %s with error: %v", member, err)
				}
				if user == nil {
					continue
				}
				members = append(members, fmt.Sprintf("user:%s", user.Email))
			} else if strings.HasPrefix(member, common.GroupPrefix) {
				email, err := common.GetGroupEmail(member)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to parse group email from member %s with error: %v", member, err)
				}
				members = append(members, fmt.Sprintf("group:%s", email))
			} else {
				// handle allUsers.
				members = append(members, member)
			}
		}
		v1pbBinding := &v1pb.Binding{
			Role:      binding.Role,
			Members:   members,
			Condition: binding.Condition,
		}
		if v1pbBinding.Condition == nil {
			v1pbBinding.Condition = &expr.Expr{}
		}
		if v1pbBinding.Condition.Expression != "" {
			e, err := cel.NewEnv(common.IAMPolicyConditionCELAttributes...)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to create cel environment with error: %v", err)
			}
			ast, issues := e.Parse(v1pbBinding.Condition.Expression)
			if issues != nil && issues.Err() != nil {
				return nil, status.Errorf(codes.Internal, "failed to parse expression with error: %v", issues.Err())
			}
			expr, err := cel.AstToParsedExpr(ast)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to convert ast to parsed expression with error: %v", err)
			}
			v1pbBinding.ParsedExpr = expr
		}
		bindings = append(bindings, v1pbBinding)
	}

	return &v1pb.IamPolicy{
		Bindings: bindings,
		Etag:     iamPolicy.Etag,
	}, nil
}

func convertToStoreIamPolicy(ctx context.Context, stores *store.Store, iamPolicy *v1pb.IamPolicy) (*storepb.IamPolicy, error) {
	var bindings []*storepb.Binding

	for _, binding := range iamPolicy.Bindings {
		var members []string
		for _, member := range binding.Members {
			storeMember, err := convertToStoreIamPolicyMember(ctx, stores, member)
			if err != nil {
				return nil, err
			}
			members = append(members, storeMember)
		}

		storeBinding := &storepb.Binding{
			Role:      binding.Role,
			Members:   members,
			Condition: binding.Condition,
		}
		if storeBinding.Condition == nil {
			storeBinding.Condition = &expr.Expr{}
		}
		bindings = append(bindings, storeBinding)
	}

	return &storepb.IamPolicy{
		Bindings: bindings,
	}, nil
}

func convertToStoreIamPolicyMember(ctx context.Context, stores *store.Store, member string) (string, error) {
	if strings.HasPrefix(member, "user:") {
		email := strings.TrimPrefix(member, "user:")
		user, err := stores.GetUserByEmail(ctx, email)
		if err != nil {
			return "", status.Errorf(codes.Internal, err.Error())
		}
		if user == nil {
			return "", status.Errorf(codes.NotFound, "user %q not found", member)
		}
		return common.FormatUserUID(user.ID), nil
	} else if strings.HasPrefix(member, "group:") {
		email := strings.TrimPrefix(member, "group:")
		return common.FormatGroupEmail(email), nil
	} else if member == api.AllUsers {
		return member, nil
	}
	return "", status.Errorf(codes.InvalidArgument, "unsupport member %s", member)
}

func convertToProject(projectMessage *store.ProjectMessage) *v1pb.Project {
	workflow := v1pb.Workflow_UI
	if projectMessage.VCSConnectorsCount > 0 {
		workflow = v1pb.Workflow_VCS
	}
	var projectWebhooks []*v1pb.Webhook
	for _, webhook := range projectMessage.Webhooks {
		projectWebhooks = append(projectWebhooks, &v1pb.Webhook{
			Name:              fmt.Sprintf("%s%s/%s%d", common.ProjectNamePrefix, projectMessage.ResourceID, common.WebhookIDPrefix, webhook.ID),
			Type:              convertWebhookTypeString(webhook.Type),
			Title:             webhook.Title,
			Url:               webhook.URL,
			NotificationTypes: convertNotificationTypeStrings(webhook.ActivityList),
			DirectMessage:     webhook.Payload.GetDirectMessage(),
		})
	}

	var issueLabels []*v1pb.Label
	for _, label := range projectMessage.Setting.IssueLabels {
		issueLabels = append(issueLabels, &v1pb.Label{
			Value: label.Value,
			Color: label.Color,
			Group: label.Group,
		})
	}

	return &v1pb.Project{
		Name:                       fmt.Sprintf("%s%s", common.ProjectNamePrefix, projectMessage.ResourceID),
		Uid:                        fmt.Sprintf("%d", projectMessage.UID),
		State:                      convertDeletedToState(projectMessage.Deleted),
		Title:                      projectMessage.Title,
		Key:                        projectMessage.Key,
		Workflow:                   workflow,
		Webhooks:                   projectWebhooks,
		DataClassificationConfigId: projectMessage.DataClassificationConfigID,
		IssueLabels:                issueLabels,
		ForceIssueLabels:           projectMessage.Setting.ForceIssueLabels,
		AllowModifyStatement:       projectMessage.Setting.AllowModifyStatement,
		AutoResolveIssue:           projectMessage.Setting.AutoResolveIssue,
	}
}

func convertToProjectMessage(resourceID string, project *v1pb.Project) (*store.ProjectMessage, error) {
	setting := &storepb.Project{
		AllowModifyStatement: project.AllowModifyStatement,
		AutoResolveIssue:     project.AutoResolveIssue,
	}
	return &store.ProjectMessage{
		ResourceID: resourceID,
		Title:      project.Title,
		Key:        project.Key,
		Setting:    setting,
	}, nil
}

func convertToDeploymentConfig(projectID string, deploymentConfig *store.DeploymentConfigMessage) *v1pb.DeploymentConfig {
	resourceName := common.FormatDeploymentConfig(common.FormatProject(projectID))
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
	case store.NotInOperatorType:
		return v1pb.OperatorType_OPERATOR_TYPE_NOT_IN
	case store.ExistsOperatorType:
		return v1pb.OperatorType_OPERATOR_TYPE_EXISTS
	}
	return v1pb.OperatorType_OPERATOR_TYPE_UNSPECIFIED
}

func convertToStoreLabelSelectorOperator(operator v1pb.OperatorType) (store.OperatorType, error) {
	switch operator {
	case v1pb.OperatorType_OPERATOR_TYPE_IN:
		return store.InOperatorType, nil
	case v1pb.OperatorType_OPERATOR_TYPE_NOT_IN:
		return store.NotInOperatorType, nil
	case v1pb.OperatorType_OPERATOR_TYPE_EXISTS:
		return store.ExistsOperatorType, nil
	}
	return store.OperatorType(""), errors.Errorf("invalid operator type: %v", operator)
}

func (s *ProjectService) validateIAMPolicy(ctx context.Context, policy *v1pb.IamPolicy) error {
	if policy == nil {
		return errors.Errorf("IAM Policy is required")
	}

	generalSetting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get workspace general setting")
	}
	var maximumRoleExpiration *durationpb.Duration
	if generalSetting != nil {
		maximumRoleExpiration = generalSetting.MaximumRoleExpiration
	}

	roleMessages, err := s.store.ListRoles(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to list roles: %v", err)
	}
	roleMessages = append(roleMessages, s.iamManager.PredefinedRoles...)
	roles := convertToRoles(roleMessages)

	return s.validateBindings(policy.Bindings, roles, maximumRoleExpiration)
}

func (*ProjectService) validateBindings(bindings []*v1pb.Binding, roles []*v1pb.Role, maximumRoleExpiration *durationpb.Duration) error {
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
		binding.Members = uniqueBindingMembers(binding.Members)
		for _, member := range binding.Members {
			if err := validateMember(member); err != nil {
				return err
			}
		}
		projectRoleMap[binding.Role] = true

		if _, err := common.ValidateProjectMemberCELExpr(binding.Condition); err != nil {
			return err
		}

		// Only validate when maximumRoleExpiration is set and the role is ProjectQuerier or ProjectExporter.
		rolesToValidate := []string{fmt.Sprintf("roles/%s", api.ProjectQuerier), fmt.Sprintf("roles/%s", api.ProjectExporter)}
		if maximumRoleExpiration != nil && binding.Condition != nil && binding.Condition.Expression != "" && slices.Contains(rolesToValidate, binding.Role) {
			if err := validateIAMPolicyExpression(binding.Condition.Expression, maximumRoleExpiration); err != nil {
				return err
			}
		}
	}
	// Must contain one owner binding.
	if _, ok := projectRoleMap[common.FormatRole(api.ProjectOwner.String())]; !ok {
		return errors.Errorf("IAM Policy must have at least one binding with %s", api.ProjectOwner.String())
	}
	return nil
}

// validateIAMPolicyExpression validates the IAM policy expression.
// Currently only validate the following expression:
// * request.time < timestamp("2021-01-01T00:00:00Z")
//
// Other expressions will be ignored.
func validateIAMPolicyExpression(expr string, maximumRoleExpiration *durationpb.Duration) error {
	e, err := cel.NewEnv()
	if err != nil {
		return errors.Wrap(err, "failed to create cel environment")
	}
	ast, iss := e.Parse(expr)
	if iss != nil {
		return errors.Wrap(iss.Err(), "failed to parse expression")
	}

	var validator func(expr celast.Expr) error

	validator = func(expr celast.Expr) error {
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case "_||_":
				for _, arg := range expr.AsCall().Args() {
					err := validator(arg)
					if err != nil {
						return err
					}
				}
				return nil
			case "_&&_":
				for _, arg := range expr.AsCall().Args() {
					err := validator(arg)
					if err != nil {
						return err
					}
				}
				return nil
			// Only handle "request.time < timestamp("2021-01-01T00:00:00Z").
			case "_<_":
				if maximumRoleExpiration == nil {
					return nil
				}

				var value string
				for _, arg := range expr.AsCall().Args() {
					switch arg.Kind() {
					case celast.SelectKind:
						variable := fmt.Sprintf("%s.%s", arg.AsSelect().Operand().AsIdent(), arg.AsSelect().FieldName())
						if variable != "request.time" {
							return errors.Errorf("unexpected variable %v", variable)
						}
					case celast.CallKind:
						functionName := arg.AsCall().FunctionName()
						if functionName != "timestamp" {
							return errors.Errorf("unexpected function %v", functionName)
						}
						if len(arg.AsCall().Args()) != 1 {
							return errors.Errorf("unexpected number of arguments %d", len(arg.AsCall().Args()))
						}
						valueArg := arg.AsCall().Args()[0]
						if valueArg.Kind() != celast.LiteralKind {
							return errors.Errorf("unexpected argument kind %v", valueArg.Kind())
						}
						lit, ok := valueArg.AsLiteral().Value().(string)
						if !ok {
							return errors.Errorf("expect string, got %T, hint: filter literals should be string", arg.AsLiteral().Value())
						}
						value = lit
					}
				}

				t, err := time.Parse(time.RFC3339, value)
				if err != nil {
					return errors.Errorf("failed to parse time %v, error: %v", value, err)
				}
				maxExpirationTime := time.Now().Add(maximumRoleExpiration.AsDuration())
				if t.After(maxExpirationTime) {
					return errors.Errorf("time %s exceeds maximum role expiration %s", t.Format(time.DateTime), maxExpirationTime.Format(time.DateTime))
				}
				return nil
			default:
				// Ignore other functions.
				return nil
			}
		default:
			// Ignore other kinds.
			return nil
		}
	}

	return validator(ast.NativeRep().Expr())
}

func uniqueBindingMembers(members []string) []string {
	temp := []string{}
	flag := make(map[string]bool)
	for _, member := range members {
		if !flag[member] {
			temp = append(temp, member)
			flag[member] = true
		}
	}
	return temp
}

func validateMember(member string) error {
	if member == api.AllUsers {
		return nil
	}

	userIdentifierMap := map[string]bool{
		"user:":  true,
		"group:": true,
	}
	for prefix := range userIdentifierMap {
		if strings.HasPrefix(member, prefix) && len(member[len(prefix):]) > 0 {
			return nil
		}
	}
	return errors.Errorf("invalid user %s", member)
}
