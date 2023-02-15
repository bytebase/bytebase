package v1

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

// ProjectService implements the project service.
type ProjectService struct {
	v1pb.UnimplementedProjectServiceServer
	store *store.Store
}

// NewProjectService creates a new ProjectService.
func NewProjectService(store *store.Store) *ProjectService {
	return &ProjectService{
		store: store,
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
	principalID := ctx.Value(common.PrincipalIDContextKey).(int)
	role := ctx.Value(common.RoleContextKey).(api.Role)
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
		case "project.title":
			patch.Title = &request.Project.Title
		case "project.key":
			patch.Key = &request.Project.Key
		case "project.workflow":
			workflow, err := convertToProjectWorkflowType(request.Project.Workflow)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
			patch.Workflow = &workflow
		case "project.tenant_mode":
			tenantMode, err := convertToProjectTenantMode(request.Project.TenantMode)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
			patch.TenantMode = &tenantMode
		case "project.db_name_template":
			patch.DBNameTemplate = &request.Project.DbNameTemplate
		case "project.schema_change":
			schemaChange, err := convertToProjectSchemaChangeType(request.Project.SchemaChange)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
			patch.SchemaChangeType = &schemaChange
		case "project.lgtm_check":
			lgtm, err := convertToLGTMCheckSetting(request.Project.LgtmCheck)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
			patch.LGTMCheckSetting = &lgtm
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
	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID})
	if err != nil {
		return nil, err
	}
	if len(databases) > 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "transfer all databases under the project before deleting the project")
	}
	openIssues, err := s.store.ListIssueV2(ctx, &store.FindIssueMessage{ProjectUID: &project.UID, StatusList: []api.IssueStatus{api.IssueOpen}})
	if err != nil {
		return nil, err
	}
	if len(openIssues) > 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "resolve all open issues before deleting the project")
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
	if err := validateIAMPolicy(request.Policy); err != nil {
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

	iamPolicy, err := s.store.SetProjectIAMPolicy(ctx, policy, creatorUID, project.UID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return convertToIamPolicy(iamPolicy), nil
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

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID:  &projectID,
		ShowDeleted: true,
	})
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
			members = append(members, getUserIdentifier(member.Email))
		}
		bindings = append(bindings, &v1pb.Binding{
			Role:    convertToProjectRole(binding.Role),
			Members: members,
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
			email := getUserEmailFromIdentifier(member)
			user, err := s.store.GetUser(ctx, &store.FindUserMessage{
				Email: &email,
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
			Role:    role,
			Members: users,
		})
	}
	return &store.IAMPolicyMessage{
		Bindings: bindings,
	}, nil
}

// getUserIdentifier returns the user identifier.
// See more details in project_service.proto.
func getUserIdentifier(email string) string {
	return "user:" + email
}

func convertToProjectRole(role api.Role) v1pb.ProjectRole {
	switch role {
	case api.Owner:
		return v1pb.ProjectRole_PROJECT_ROLE_OWNER
	case api.Developer:
		return v1pb.ProjectRole_PROJECT_ROLE_DEVELOPER
	default:
		return v1pb.ProjectRole_PROJECT_ROLE_UNSPECIFIED
	}
}

func convertProjectRole(role v1pb.ProjectRole) (api.Role, error) {
	switch role {
	case v1pb.ProjectRole_PROJECT_ROLE_OWNER:
		return api.Owner, nil
	case v1pb.ProjectRole_PROJECT_ROLE_DEVELOPER:
		return api.Developer, nil
	}
	return api.Role(""), errors.Errorf("invalid project role %q", role)
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

	lgtmCheck := v1pb.LgtmCheck_LGTM_CHECK_UNSPECIFIED
	switch projectMessage.LGTMCheckSetting.Value {
	case api.LGTMValueDisabled:
		lgtmCheck = v1pb.LgtmCheck_LGTM_CHECK_DISABLED
	case api.LGTMValueProjectMember:
		lgtmCheck = v1pb.LgtmCheck_LGTM_CHECK_PROJECT_MEMBER
	case api.LGTMValueProjectOwner:
		lgtmCheck = v1pb.LgtmCheck_LGTM_CHECK_PROJECT_OWNER
	}

	return &v1pb.Project{
		Name:           fmt.Sprintf("%s%s", projectNamePrefix, projectMessage.ResourceID),
		Uid:            fmt.Sprintf("%d", projectMessage.UID),
		Title:          projectMessage.Title,
		Key:            projectMessage.Key,
		Workflow:       workflow,
		Visibility:     visibility,
		TenantMode:     tenantMode,
		DbNameTemplate: projectMessage.DBNameTemplate,
		// TODO(d): schema_version_type for project.
		SchemaVersion: v1pb.SchemaVersion_SCHEMA_VERSION_UNSPECIFIED,
		SchemaChange:  schemaChange,
		LgtmCheck:     lgtmCheck,
	}
}

func convertToProjectWorkflowType(workflow v1pb.Workflow) (api.ProjectWorkflowType, error) {
	var w api.ProjectWorkflowType
	switch workflow {
	case v1pb.Workflow_UI:
		w = api.UIWorkflow
	case v1pb.Workflow_VCS:
		w = api.VCSWorkflow
	default:
		return w, errors.Errorf("invalid workflow %v", workflow)
	}
	return w, nil
}

func convertToProjectVisibility(visibility v1pb.Visibility) (api.ProjectVisibility, error) {
	var v api.ProjectVisibility
	switch visibility {
	case v1pb.Visibility_VISIBILITY_PRIVATE:
		v = api.Private
	case v1pb.Visibility_VISIBILITY_PUBLIC:
		v = api.Public
	default:
		return v, errors.Errorf("invalid visibility %v", visibility)
	}
	return v, nil
}

func convertToProjectTenantMode(tenantMode v1pb.TenantMode) (api.ProjectTenantMode, error) {
	var t api.ProjectTenantMode
	switch tenantMode {
	case v1pb.TenantMode_TENANT_MODE_DISABLED:
		t = api.TenantModeDisabled
	case v1pb.TenantMode_TENANT_MODE_ENABLED:
		t = api.TenantModeTenant
	default:
		return t, errors.Errorf("invalid tenant mode %v", tenantMode)
	}
	return t, nil
}

func convertToProjectSchemaChangeType(schemaChange v1pb.SchemaChange) (api.ProjectSchemaChangeType, error) {
	var s api.ProjectSchemaChangeType
	switch schemaChange {
	case v1pb.SchemaChange_DDL:
		s = api.ProjectSchemaChangeTypeDDL
	case v1pb.SchemaChange_SDL:
		s = api.ProjectSchemaChangeTypeSDL
	default:
		return s, errors.Errorf("invalid schema change type %v", schemaChange)
	}
	return s, nil
}

func convertToLGTMCheckSetting(lgtmCheck v1pb.LgtmCheck) (api.LGTMCheckSetting, error) {
	var lgtm api.LGTMCheckSetting
	switch lgtmCheck {
	case v1pb.LgtmCheck_LGTM_CHECK_DISABLED:
		lgtm = api.LGTMCheckSetting{
			Value: api.LGTMValueDisabled,
		}
	case v1pb.LgtmCheck_LGTM_CHECK_PROJECT_MEMBER:
		lgtm = api.LGTMCheckSetting{
			Value: api.LGTMValueProjectMember,
		}
	case v1pb.LgtmCheck_LGTM_CHECK_PROJECT_OWNER:
		lgtm = api.LGTMCheckSetting{
			Value: api.LGTMValueProjectOwner,
		}
	default:
		return lgtm, errors.Errorf("invalid LGTM check %v", lgtmCheck)
	}
	return lgtm, nil
}

func convertToProjectMessage(resourceID string, project *v1pb.Project) (*store.ProjectMessage, error) {
	workflow, err := convertToProjectWorkflowType(project.Workflow)
	if err != nil {
		return nil, err
	}

	visibility, err := convertToProjectVisibility(project.Visibility)
	if err != nil {
		return nil, err
	}

	tenantMode, err := convertToProjectTenantMode(project.TenantMode)
	if err != nil {
		return nil, err
	}

	schemaChange, err := convertToProjectSchemaChangeType(project.SchemaChange)
	if err != nil {
		return nil, err
	}

	lgtmCheck, err := convertToLGTMCheckSetting(project.LgtmCheck)
	if err != nil {
		return nil, err
	}

	return &store.ProjectMessage{
		ResourceID:       resourceID,
		Title:            project.Title,
		Key:              project.Key,
		Workflow:         workflow,
		Visibility:       visibility,
		TenantMode:       tenantMode,
		DBNameTemplate:   project.DbNameTemplate,
		SchemaChangeType: schemaChange,
		LGTMCheckSetting: lgtmCheck,
	}, nil
}

func convertToDeploymentConfig(projectID string, deploymentConfig *store.DeploymentConfigMessage) *v1pb.DeploymentConfig {
	resourceName := fmt.Sprintf("projects/%s/deploymentConfig)", projectID)
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

func validateIAMPolicy(policy *v1pb.IamPolicy) error {
	if policy == nil {
		return errors.Errorf("IAM Policy is required")
	}
	return validateBindings(policy.Bindings)
}

func getUserEmailFromIdentifier(ident string) string {
	return strings.TrimPrefix(ident, "user:")
}

func validateBindings(bindings []*v1pb.Binding) error {
	if len(bindings) == 0 {
		return errors.Errorf("IAM Binding is required")
	}
	userMap := make(map[string]bool)
	projectRoleMap := make(map[v1pb.ProjectRole]bool)
	for _, binding := range bindings {
		if binding.Role == v1pb.ProjectRole_PROJECT_ROLE_UNSPECIFIED {
			return errors.Errorf("IAM Binding role is required")
		}
		// Each of the bindings must contain at least one member.
		if len(binding.Members) == 0 {
			return errors.Errorf("Each IAM binding must have at least one member")
		}
		// We have not merge the binding by the same role yet, so the roles in each binding must be unique.
		if _, ok := projectRoleMap[binding.Role]; ok {
			return errors.Errorf("Each IAM binding must have a unique role")
		}

		for _, member := range binding.Members {
			if _, ok := userMap[member]; ok {
				return errors.Errorf("duplicate user %s", member)
			}
			userMap[member] = true
			if err := validateMember(member); err != nil {
				return err
			}
		}
		projectRoleMap[binding.Role] = true
	}
	// Must contain one owner binding.
	if _, ok := projectRoleMap[v1pb.ProjectRole_PROJECT_ROLE_OWNER]; !ok {
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
