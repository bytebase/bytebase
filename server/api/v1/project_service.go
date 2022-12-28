package v1

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/api"
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

// GetProject gets an project.
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
