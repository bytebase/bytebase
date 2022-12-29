package v1

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
	"github.com/bytebase/bytebase/store"
)

const projectNamePrefix = "projects/"

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
	projectID, err := getProjectID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	project, err := s.store.GetProjectV2(ctx, projectID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.InvalidArgument, "project %q not found", projectID)
	}

	return convertProject(project), nil
}

// ListProjects lists all projects.
func (s *ProjectService) ListProjects(ctx context.Context, request *v1pb.ListProjectsRequest) (*v1pb.ListProjectsResponse, error) {
	projects, err := s.store.ListProjectV2(ctx, request.ShowDeleted)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	response := &v1pb.ListProjectsResponse{}
	for _, project := range projects {
		response.Projects = append(response.Projects, convertProject(project))
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
	return convertProject(project), nil
}

// UpdateProject updates a project.
func (s *ProjectService) UpdateProject(ctx context.Context, request *v1pb.UpdateProjectRequest) (*v1pb.Project, error) {
	if request.Project == nil {
		return nil, status.Errorf(codes.InvalidArgument, "project must be set")
	}

	projectID, err := getProjectID(request.Project.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	project, err := s.store.GetProjectV2(ctx, projectID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.InvalidArgument, "project %q not found", projectID)
	}

	patch := &store.UpdateProjectMessage{
		UpdaterID:  ctx.Value(common.PrincipalIDContextKey).(int),
		ResourceID: projectID,
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
		case "project.role_provider":
			roleProvider, err := convertToProjectRoleProvider(request.Project.RoleProvider)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, err.Error())
			}
			patch.RoleProvider = &roleProvider
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

	projectMsg, err := s.store.UpdateProjectV2(ctx, patch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return convertProject(projectMsg), nil
}

// DeleteProject deletes a project.
func (s *ProjectService) DeleteProject(ctx context.Context, request *v1pb.DeleteProjectRequest) (*emptypb.Empty, error) {
	projectID, err := getProjectID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	project, err := s.store.GetProjectV2(ctx, projectID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.InvalidArgument, "project %q not found", projectID)
	}

	rowStatus := api.Archived
	if _, err := s.store.UpdateProjectV2(ctx, &store.UpdateProjectMessage{
		UpdaterID:  ctx.Value(common.PrincipalIDContextKey).(int),
		ResourceID: projectID,
		RowStatus:  &rowStatus,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

// UndeleteProject undeletes a project.
func (s *ProjectService) UndeleteProject(ctx context.Context, request *v1pb.UndeleteProjectRequest) (*v1pb.Project, error) {
	projectID, err := getProjectID(request.Name)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	project, err := s.store.GetProjectV2(ctx, projectID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	if project == nil {
		return nil, status.Errorf(codes.InvalidArgument, "project %q not found", projectID)
	}

	rowStatus := api.Normal
	projectMsg, err := s.store.UpdateProjectV2(ctx, &store.UpdateProjectMessage{
		UpdaterID:  ctx.Value(common.PrincipalIDContextKey).(int),
		ResourceID: projectID,
		RowStatus:  &rowStatus,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}

	return convertProject(projectMsg), nil
}

func getProjectID(name string) (string, error) {
	if !strings.HasPrefix(name, projectNamePrefix) {
		return "", errors.Errorf("invalid project name %q", name)
	}
	projectID := strings.TrimPrefix(name, projectNamePrefix)
	if projectID == "" {
		return "", errors.Errorf("project cannot be empty")
	}
	return projectID, nil
}

func convertProject(projectMessage *store.ProjectMessage) *v1pb.Project {
	workflow := v1pb.Workflow_WORKFLOW_UNSPECIFIED
	switch projectMessage.Workflow {
	case api.UIWorkflow:
		workflow = v1pb.Workflow_WORKFLOW_UI
	case api.VCSWorkflow:
		workflow = v1pb.Workflow_WORKFLOW_VCS
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

	roleProvider := v1pb.RoleProvider_ROLE_PROVIDER_UNSPECIFIED
	switch projectMessage.RoleProvider {
	case api.ProjectRoleProviderBytebase:
		roleProvider = v1pb.RoleProvider_ROLE_PROVIDER_BYTEBASE
	case api.ProjectRoleProviderGitHubCom:
		roleProvider = v1pb.RoleProvider_ROLE_PROVIDER_GITHUB_COM
	case api.ProjectRoleProviderGitLabSelfHost:
		roleProvider = v1pb.RoleProvider_ROLE_PROVIDER_GITLAB_SELF_HOST
	}

	schemaChange := v1pb.SchemaChange_SCHEMA_CHANGE_UNSPECIFIED
	switch projectMessage.SchemaChangeType {
	case api.ProjectSchemaChangeTypeDDL:
		schemaChange = v1pb.SchemaChange_SCHEMA_CHANGE_DDL
	case api.ProjectSchemaChangeTypeSDL:
		schemaChange = v1pb.SchemaChange_SCHEMA_CHANGE_SDL
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
		Title:          projectMessage.Title,
		Key:            projectMessage.Key,
		Workflow:       workflow,
		Visibility:     visibility,
		TenantMode:     tenantMode,
		DbNameTemplate: projectMessage.DBNameTemplate,
		RoleProvider:   roleProvider,
		// TODO: schema_version_type for project.
		SchemaVersion: v1pb.SchemaVersion_SCHEMA_VERSION_UNSPECIFIED,
		SchemaChange:  schemaChange,
		LgtmCheck:     lgtmCheck,
	}
}

func convertToProjectWorkflowType(workflow v1pb.Workflow) (api.ProjectWorkflowType, error) {
	var w api.ProjectWorkflowType
	switch workflow {
	case v1pb.Workflow_WORKFLOW_UI:
		w = api.UIWorkflow
	case v1pb.Workflow_WORKFLOW_VCS:
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

func convertToProjectRoleProvider(roleProvider v1pb.RoleProvider) (api.ProjectRoleProvider, error) {
	var r api.ProjectRoleProvider
	switch roleProvider {
	case v1pb.RoleProvider_ROLE_PROVIDER_BYTEBASE:
		r = api.ProjectRoleProviderBytebase
	case v1pb.RoleProvider_ROLE_PROVIDER_GITHUB_COM:
		r = api.ProjectRoleProviderGitHubCom
	case v1pb.RoleProvider_ROLE_PROVIDER_GITLAB_SELF_HOST:
		r = api.ProjectRoleProviderGitLabSelfHost
	default:
		return r, errors.Errorf("invalid role provider %v", roleProvider)
	}
	return r, nil
}

func convertToProjectSchemaChangeType(schemaChange v1pb.SchemaChange) (api.ProjectSchemaChangeType, error) {
	var s api.ProjectSchemaChangeType
	switch schemaChange {
	case v1pb.SchemaChange_SCHEMA_CHANGE_DDL:
		s = api.ProjectSchemaChangeTypeDDL
	case v1pb.SchemaChange_SCHEMA_CHANGE_SDL:
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

	roleProvider, err := convertToProjectRoleProvider(project.RoleProvider)
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
		ProjectID:        resourceID,
		Title:            project.Title,
		Key:              project.Key,
		Workflow:         workflow,
		Visibility:       visibility,
		TenantMode:       tenantMode,
		DBNameTemplate:   project.DbNameTemplate,
		RoleProvider:     roleProvider,
		SchemaChangeType: schemaChange,
		LGTMCheckSetting: lgtmCheck,
	}, nil
}
