package v1

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	webhookplugin "github.com/bytebase/bytebase/backend/plugin/webhook"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
)

// ProjectService implements the project service.
type ProjectService struct {
	v1connect.UnimplementedProjectServiceHandler
	store          *store.Store
	profile        *config.Profile
	iamManager     *iam.Manager
	licenseService *enterprise.LicenseService
}

// NewProjectService creates a new ProjectService.
func NewProjectService(
	store *store.Store,
	profile *config.Profile,
	iamManager *iam.Manager,
	licenseService *enterprise.LicenseService,
) *ProjectService {
	return &ProjectService{
		store:          store,
		profile:        profile,
		iamManager:     iamManager,
		licenseService: licenseService,
	}
}

// GetProject gets a project.
func (s *ProjectService) GetProject(ctx context.Context, req *connect.Request[v1pb.GetProjectRequest]) (*connect.Response[v1pb.Project], error) {
	project, err := s.getProjectMessage(ctx, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(convertToProject(project)), nil
}

// ListProjects lists all projects.
func (s *ProjectService) ListProjects(ctx context.Context, req *connect.Request[v1pb.ListProjectsRequest]) (*connect.Response[v1pb.ListProjectsResponse], error) {
	offset, err := parseLimitAndOffset(&pageSize{
		token:   req.Msg.PageToken,
		limit:   int(req.Msg.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	find := &store.FindProjectMessage{
		ShowDeleted: req.Msg.ShowDeleted,
		Limit:       &limitPlusOne,
		Offset:      &offset.offset,
	}
	filter, err := getListProjectFilter(req.Msg.Filter)
	if err != nil {
		return nil, err
	}
	find.Filter = filter
	projects, err := s.store.ListProjectV2(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	nextPageToken := ""
	if len(projects) == limitPlusOne {
		projects = projects[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to marshal next page token"))
		}
	}

	response := &v1pb.ListProjectsResponse{
		NextPageToken: nextPageToken,
	}
	for _, project := range projects {
		response.Projects = append(response.Projects, convertToProject(project))
	}
	return connect.NewResponse(response), nil
}

func getListProjectFilter(filter string) (*store.ListResourceFilter, error) {
	if filter == "" {
		return nil, nil
	}
	e, err := cel.NewEnv()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to create cel env"))
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String()))
	}

	var getFilter func(expr celast.Expr) (string, error)
	var positionalArgs []any

	parseToSQL := func(variable, value any) (string, error) {
		switch variable {
		case "name":
			positionalArgs = append(positionalArgs, value.(string))
			return fmt.Sprintf("project.name = $%d", len(positionalArgs)), nil
		case "resource_id":
			positionalArgs = append(positionalArgs, value.(string))
			return fmt.Sprintf("project.resource_id = $%d", len(positionalArgs)), nil
		case "exclude_default":
			if excludeDefault, ok := value.(bool); excludeDefault && ok {
				positionalArgs = append(positionalArgs, common.DefaultProjectID)
				return fmt.Sprintf("project.resource_id != $%d", len(positionalArgs)), nil
			}
			return "TRUE", nil
		case "state":
			v1State, ok := v1pb.State_value[value.(string)]
			if !ok {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid state filter %q", value))
			}
			positionalArgs = append(positionalArgs, v1pb.State(v1State) == v1pb.State_DELETED)
			return fmt.Sprintf("project.deleted = $%d", len(positionalArgs)), nil
		default:
			// Check if it's a label filter (e.g., "labels.environment" == "production")
			varStr, ok := variable.(string)
			if !ok {
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport variable %q", variable))
			}
			if labelKey, ok := strings.CutPrefix(varStr, "labels."); ok {
				return parseToLabelFilterSQL("project.setting", labelKey, value)
			}
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport variable %q", variable))
		}
	}

	getFilter = func(expr celast.Expr) (string, error) {
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalOr:
				return getSubConditionFromExpr(expr, getFilter, "OR")
			case celoperators.LogicalAnd:
				return getSubConditionFromExpr(expr, getFilter, "AND")
			case celoperators.Equals:
				variable, value := getVariableAndValueFromExpr(expr)
				return parseToSQL(variable, value)
			case celoverloads.Matches:
				variable := expr.AsCall().Target().AsIdent()
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`invalid args for %q`, variable))
				}
				value := args[0].AsLiteral().Value()
				strValue, ok := value.(string)
				if !ok {
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("expect string, got %T, hint: filter literals should be string", value))
				}

				switch variable {
				case "name":
					return "LOWER(project.name) LIKE '%" + strings.ToLower(strValue) + "%'", nil
				case "resource_id":
					return "LOWER(project.resource_id) LIKE '%" + strings.ToLower(strValue) + "%'", nil
				default:
					return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport variable %q", variable))
				}
			case celoperators.In:
				variable, value := getVariableAndValueFromExpr(expr)
				if labelKey, ok := strings.CutPrefix(variable, "labels."); ok {
					return parseToLabelFilterSQL("project.setting", labelKey, value)
				}
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected %v operator for %v", functionName, variable))
			default:
				return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected function %v", functionName))
			}
		default:
			return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unexpected expr kind %v", expr.Kind()))
		}
	}

	where, err := getFilter(ast.NativeRep().Expr())
	if err != nil {
		return nil, err
	}

	return &store.ListResourceFilter{
		Args:  positionalArgs,
		Where: "(" + where + ")",
	}, nil
}

// SearchProjects searches all projects on which the user has bb.projects.get permission.
func (s *ProjectService) SearchProjects(ctx context.Context, req *connect.Request[v1pb.SearchProjectsRequest]) (*connect.Response[v1pb.SearchProjectsResponse], error) {
	user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}

	offset, err := parseLimitAndOffset(&pageSize{
		token:   req.Msg.PageToken,
		limit:   int(req.Msg.PageSize),
		maximum: 1000,
	})
	if err != nil {
		return nil, err
	}
	limitPlusOne := offset.limit + 1

	find := &store.FindProjectMessage{
		ShowDeleted: req.Msg.ShowDeleted,
		Limit:       &limitPlusOne,
		Offset:      &offset.offset,
	}
	filter, err := getListProjectFilter(req.Msg.Filter)
	if err != nil {
		return nil, err
	}
	find.Filter = filter

	projects, err := s.store.ListProjectV2(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	nextPageToken := ""
	if len(projects) == limitPlusOne {
		projects = projects[:offset.limit]
		if nextPageToken, err = offset.getNextPageToken(); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to marshal next page token"))
		}
	}

	ok, err = s.iamManager.CheckPermission(ctx, iam.PermissionProjectsGet, user)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check permission"))
	}
	if !ok {
		var ps []*store.ProjectMessage
		for _, project := range projects {
			ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionProjectsGet, user, project.ResourceID)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to check permission for project %q", project.ResourceID))
			}
			if ok {
				ps = append(ps, project)
			}
		}
		projects = ps
	}

	response := &v1pb.SearchProjectsResponse{
		NextPageToken: nextPageToken,
	}
	for _, project := range projects {
		response.Projects = append(response.Projects, convertToProject(project))
	}
	return connect.NewResponse(response), nil
}

// CreateProject creates a project.
func (s *ProjectService) CreateProject(ctx context.Context, req *connect.Request[v1pb.CreateProjectRequest]) (*connect.Response[v1pb.Project], error) {
	if !isValidResourceID(req.Msg.ProjectId) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid project ID %v", req.Msg.ProjectId))
	}

	if req.Msg.Project != nil && req.Msg.Project.Labels != nil {
		if err := validateProjectLabels(req.Msg.Project.Labels); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}

	projectMessage := convertToProjectMessage(req.Msg.ProjectId, req.Msg.Project)

	setting, err := s.store.GetDataClassificationSetting(ctx)
	if err != nil {
		slog.Error("failed to find classification setting", log.BBError(err))
	}
	if setting != nil && len(setting.Configs) != 0 {
		projectMessage.DataClassificationConfigID = setting.Configs[0].Id
	}

	principalID, ok := ctx.Value(common.PrincipalIDContextKey).(int)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("principal ID not found"))
	}
	project, err := s.store.CreateProjectV2(ctx,
		projectMessage,
		principalID,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(convertToProject(project)), nil
}

// UpdateProject updates a project.
func (s *ProjectService) UpdateProject(ctx context.Context, req *connect.Request[v1pb.UpdateProjectRequest]) (*connect.Response[v1pb.Project], error) {
	if req.Msg.Project == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("project must be set"))
	}
	if req.Msg.UpdateMask == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("update_mask must be set"))
	}

	project, err := s.getProjectMessage(ctx, req.Msg.Project.Name)
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound && req.Msg.AllowMissing {
			// When allow_missing is true and project doesn't exist, create a new one
			user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
			if !ok {
				return nil, connect.NewError(connect.CodeInternal, errors.New("user not found"))
			}

			projectID, perr := common.GetProjectID(req.Msg.Project.Name)
			if perr != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, perr)
			}

			ok, err = s.iamManager.CheckPermission(ctx, iam.PermissionProjectsCreate, user)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to check permission"))
			}
			if !ok {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionProjectsCreate))
			}
			return s.CreateProject(ctx, connect.NewRequest(&v1pb.CreateProjectRequest{
				Project:   req.Msg.Project,
				ProjectId: projectID,
			}))
		}
		return nil, err
	}
	if project.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q has been deleted", req.Msg.Project.Name))
	}
	if project.ResourceID == common.DefaultProjectID {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("default project cannot be updated"))
	}

	patch := &store.UpdateProjectMessage{
		ResourceID: project.ResourceID,
	}

	for _, path := range req.Msg.UpdateMask.Paths {
		switch path {
		case "title":
			patch.Title = &req.Msg.Project.Title
		case "data_classification_config_id":
			setting, err := s.store.GetDataClassificationSetting(ctx)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to get data classification setting"))
			}
			existConfig := false
			for _, config := range setting.Configs {
				if config.Id == req.Msg.Project.DataClassificationConfigId {
					existConfig = true
					break
				}
			}
			if !existConfig {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("data classification %s not exists", req.Msg.Project.DataClassificationConfigId))
			}
			patch.DataClassificationConfigID = &req.Msg.Project.DataClassificationConfigId
		case "issue_labels":
			projectSettings := project.Setting
			var issueLabels []*storepb.Label
			for _, label := range req.Msg.Project.IssueLabels {
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
			projectSettings.ForceIssueLabels = req.Msg.Project.ForceIssueLabels
			patch.Setting = projectSettings
		case "allow_modify_statement":
			projectSettings := project.Setting
			projectSettings.AllowModifyStatement = req.Msg.Project.AllowModifyStatement
			patch.Setting = projectSettings
		case "auto_resolve_issue":
			projectSettings := project.Setting
			projectSettings.AutoResolveIssue = req.Msg.Project.AutoResolveIssue
			patch.Setting = projectSettings
		case "enforce_issue_title":
			projectSettings := project.Setting
			projectSettings.EnforceIssueTitle = req.Msg.Project.EnforceIssueTitle
			patch.Setting = projectSettings
		case "auto_enable_backup":
			projectSettings := project.Setting
			projectSettings.AutoEnableBackup = req.Msg.Project.AutoEnableBackup
			patch.Setting = projectSettings
		case "skip_backup_errors":
			projectSettings := project.Setting
			projectSettings.SkipBackupErrors = req.Msg.Project.SkipBackupErrors
			patch.Setting = projectSettings
		case "postgres_database_tenant_mode":
			projectSettings := project.Setting
			projectSettings.PostgresDatabaseTenantMode = req.Msg.Project.PostgresDatabaseTenantMode
			patch.Setting = projectSettings
		case "allow_self_approval":
			projectSettings := project.Setting
			projectSettings.AllowSelfApproval = req.Msg.Project.AllowSelfApproval
			patch.Setting = projectSettings
		case "execution_retry_policy":
			projectSettings := project.Setting
			projectSettings.ExecutionRetryPolicy = convertToStoreExecutionRetryPolicy(req.Msg.Project.ExecutionRetryPolicy)
			patch.Setting = projectSettings
		case "ci_sampling_size":
			projectSettings := project.Setting
			projectSettings.CiSamplingSize = req.Msg.Project.CiSamplingSize
			patch.Setting = projectSettings
		case "parallel_tasks_per_rollout":
			projectSettings := project.Setting
			projectSettings.ParallelTasksPerRollout = req.Msg.Project.ParallelTasksPerRollout
			patch.Setting = projectSettings
		case "labels":
			if err := validateProjectLabels(req.Msg.Project.Labels); err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			projectSettings := project.Setting
			projectSettings.Labels = req.Msg.Project.Labels
			patch.Setting = projectSettings
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`unsupport update_mask "%s"`, path))
		}
	}

	project, err = s.store.UpdateProjectV2(ctx, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(convertToProject(project)), nil
}

// DeleteProject deletes a project.
func (s *ProjectService) DeleteProject(ctx context.Context, req *connect.Request[v1pb.DeleteProjectRequest]) (*connect.Response[emptypb.Empty], error) {
	project, err := s.getProjectMessage(ctx, req.Msg.Name)
	if err != nil {
		return nil, err
	}

	// Handle purge (hard delete) of soft-deleted project
	if req.Msg.Purge {
		// Following AIP-165, purge only works on already soft-deleted projects
		if !project.Deleted {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("project %q must be soft-deleted before it can be purged", req.Msg.Name))
		}
		if project.ResourceID == common.DefaultProjectID {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("default project cannot be purged"))
		}

		// Permanently delete the project and all related resources
		if err := s.store.DeleteProject(ctx, project.ResourceID); err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to purge project"))
		}

		return connect.NewResponse(&emptypb.Empty{}), nil
	}

	// Regular soft delete flow
	if project.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q has been deleted", req.Msg.Name))
	}
	if project.ResourceID == common.DefaultProjectID {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("default project cannot be deleted"))
	}

	// Resources prevent project deletion.
	databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{ProjectID: &project.ResourceID, ShowDeleted: true})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// We don't move the sheet to default project because BYTEBASE_ARTIFACT sheets belong to the issue and issue project.
	if req.Msg.Force {
		if len(databases) > 0 {
			defaultProject := common.DefaultProjectID
			if _, err := s.store.BatchUpdateDatabases(ctx, databases, &store.BatchUpdateDatabases{ProjectID: &defaultProject}); err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		}
		// We don't close the issues because they might be open still.
	} else {
		// Return the open issue error first because that's more important than transferring out databases.
		openIssues, err := s.store.ListIssueV2(ctx, &store.FindIssueMessage{ProjectIDs: &[]string{project.ResourceID}, StatusList: []storepb.Issue_Status{storepb.Issue_OPEN}})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if len(openIssues) > 0 {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("resolve all open issues before deleting the project"))
		}
		if len(databases) > 0 {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf("transfer all databases to the default project before deleting the project"))
		}
	}

	if _, err := s.store.UpdateProjectV2(ctx, &store.UpdateProjectMessage{
		ResourceID: project.ResourceID,
		Delete:     &deletePatch,
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// UndeleteProject undeletes a project.
func (s *ProjectService) UndeleteProject(ctx context.Context, req *connect.Request[v1pb.UndeleteProjectRequest]) (*connect.Response[v1pb.Project], error) {
	project, err := s.getProjectMessage(ctx, req.Msg.Name)
	if err != nil {
		return nil, err
	}
	if !project.Deleted {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("project %q is active", req.Msg.Name))
	}

	project, err = s.store.UpdateProjectV2(ctx, &store.UpdateProjectMessage{
		ResourceID: project.ResourceID,
		Delete:     &undeletePatch,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(convertToProject(project)), nil
}

// BatchDeleteProjects deletes multiple projects in batch.
func (s *ProjectService) BatchDeleteProjects(ctx context.Context, request *connect.Request[v1pb.BatchDeleteProjectsRequest]) (*connect.Response[emptypb.Empty], error) {
	if len(request.Msg.Names) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("names cannot be empty"))
	}

	// Phase 1: Load all projects and check permissions
	var projects []*store.ProjectMessage
	for _, name := range request.Msg.Names {
		project, err := s.getProjectMessage(ctx, name)
		if err != nil {
			return nil, err
		}
		if project.Deleted {
			return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q has already been deleted", name))
		}
		if project.ResourceID == common.DefaultProjectID {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("default project cannot be deleted"))
		}
		projects = append(projects, project)
	}

	// Phase 2: Check dependencies for all projects if force is false
	if !request.Msg.Force {
		var blockedProjects []string
		for _, project := range projects {
			// Check for open issues
			openIssues, err := s.store.ListIssueV2(ctx, &store.FindIssueMessage{
				ProjectIDs: &[]string{project.ResourceID},
				StatusList: []storepb.Issue_Status{storepb.Issue_OPEN},
			})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			if len(openIssues) > 0 {
				blockedProjects = append(blockedProjects, project.ResourceID)
				continue
			}

			// Check for databases
			databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{
				ProjectID:   &project.ResourceID,
				ShowDeleted: true,
			})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			if len(databases) > 0 {
				blockedProjects = append(blockedProjects, project.ResourceID)
			}
		}

		if len(blockedProjects) > 0 {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				errors.Errorf("the following projects have open issues or databases and cannot be deleted: %v. Use force=true to move databases to default project",
					blockedProjects))
		}
	} else {
		// Phase 3: Execute deletions
		// If force is true, we need to move databases to default project
		var dbs []*store.DatabaseMessage
		for _, project := range projects {
			databases, err := s.store.ListDatabases(ctx, &store.FindDatabaseMessage{
				ProjectID:   &project.ResourceID,
				ShowDeleted: true,
			})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
			dbs = append(dbs, databases...)
		}
		if len(dbs) > 0 {
			defaultProject := common.DefaultProjectID
			// Note: BatchUpdateDatabases already uses transactions internally
			if _, err := s.store.BatchUpdateDatabases(ctx, dbs, &store.BatchUpdateDatabases{
				ProjectID: &defaultProject,
			}); err != nil {
				return nil, connect.NewError(connect.CodeInternal, err)
			}
		}
	}

	// Phase 4: Mark all projects as deleted.
	var updatePatches []*store.UpdateProjectMessage
	for _, project := range projects {
		updatePatches = append(updatePatches, &store.UpdateProjectMessage{
			ResourceID: project.ResourceID,
			Delete:     &deletePatch,
		})
	}

	if _, err := s.store.BatchUpdateProjectsV2(ctx, updatePatches); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&emptypb.Empty{}), nil
}

// GetIamPolicy returns the IAM policy for a project.
func (s *ProjectService) GetIamPolicy(ctx context.Context, req *connect.Request[v1pb.GetIamPolicyRequest]) (*connect.Response[v1pb.IamPolicy], error) {
	projectID, err := common.GetProjectID(req.Msg.Resource)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("cannot found project %s", projectID))
	}

	policy, err := s.store.GetProjectIamPolicy(ctx, project.ResourceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	iamPolicy, err := convertToV1IamPolicy(ctx, s.store, policy)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(iamPolicy), nil
}

// BatchGetIamPolicy returns the IAM policy for projects in batch.
func (s *ProjectService) BatchGetIamPolicy(ctx context.Context, req *connect.Request[v1pb.BatchGetIamPolicyRequest]) (*connect.Response[v1pb.BatchGetIamPolicyResponse], error) {
	resp := &v1pb.BatchGetIamPolicyResponse{}
	for _, name := range req.Msg.Names {
		projectID, err := common.GetProjectID(name)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
			ResourceID: &projectID,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		if project == nil {
			continue
		}
		policy, err := s.store.GetProjectIamPolicy(ctx, project.ResourceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
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
	return connect.NewResponse(resp), nil
}

// SetIamPolicy sets the IAM policy for a project.
func (s *ProjectService) SetIamPolicy(ctx context.Context, req *connect.Request[v1pb.SetIamPolicyRequest]) (*connect.Response[v1pb.IamPolicy], error) {
	projectID, err := common.GetProjectID(req.Msg.Resource)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", req.Msg.Resource))
	}
	if project.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q has been deleted", req.Msg.Resource))
	}

	oldIamPolicyMsg, err := s.store.GetProjectIamPolicy(ctx, project.ResourceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to find project iam policy with error"))
	}
	if req.Msg.Etag != "" && req.Msg.Etag != oldIamPolicyMsg.Etag {
		return nil, connect.NewError(connect.CodeAborted, errors.Errorf("there is concurrent update to the project iam policy, please refresh and try again"))
	}

	existProjectOwner, err := validateIAMPolicy(ctx, s.store, s.iamManager, req.Msg.Policy, oldIamPolicyMsg)
	if err != nil {
		return nil, err
	}
	// Must contain one owner binding.
	if !existProjectOwner {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("IAM Policy must have at least one binding with %s", common.ProjectOwner))
	}

	policy, err := convertToStoreIamPolicy(ctx, s.store, req.Msg.Policy)
	if err != nil {
		return nil, err
	}

	policyPayload, err := protojson.Marshal(policy)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if _, err := s.store.CreatePolicyV2(ctx, &store.PolicyMessage{
		Resource:          common.FormatProject(project.ResourceID),
		ResourceType:      storepb.Policy_PROJECT,
		Payload:           string(policyPayload),
		Type:              storepb.Policy_IAM,
		InheritFromParent: false,
		// Enforce cannot be false while creating a policy.
		Enforce: true,
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	iamPolicyMessage, err := s.store.GetProjectIamPolicy(ctx, project.ResourceID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if setServiceData, ok := common.GetSetServiceDataFromContext(ctx); ok {
		deltas := findIamPolicyDeltas(oldIamPolicyMsg.Policy, iamPolicyMessage.Policy)
		p, err := convertToProtoAny(deltas)
		if err != nil {
			slog.Warn("audit: failed to convert to anypb.Any")
		}
		setServiceData(p)
	}

	iamPolicy, err := convertToV1IamPolicy(ctx, s.store, iamPolicyMessage)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(iamPolicy), nil
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

// AddWebhook adds a webhook to a given project.
func (s *ProjectService) AddWebhook(ctx context.Context, req *connect.Request[v1pb.AddWebhookRequest]) (*connect.Response[v1pb.Project], error) {
	projectID, err := common.GetProjectID(req.Msg.Project)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", req.Msg.Project))
	}
	if project.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q has been deleted", req.Msg.Project))
	}

	create, err := convertToStoreProjectWebhookMessage(req.Msg.Webhook)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if _, err := s.store.CreateProjectWebhookV2(ctx, project.ResourceID, create); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	project, err = s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(convertToProject(project)), nil
}

// UpdateWebhook updates a webhook.
func (s *ProjectService) UpdateWebhook(ctx context.Context, req *connect.Request[v1pb.UpdateWebhookRequest]) (*connect.Response[v1pb.Project], error) {
	projectID, webhookID, err := common.GetProjectIDWebhookID(req.Msg.Webhook.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	webhookIDInt, err := strconv.Atoi(webhookID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid webhook id %q", webhookID))
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", projectID))
	}
	if project.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q has been deleted", projectID))
	}

	webhook, err := s.store.GetProjectWebhookV2(ctx, &store.FindProjectWebhookMessage{
		ProjectID: &project.ResourceID,
		ID:        &webhookIDInt,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if webhook == nil {
		if req.Msg.AllowMissing {
			// When allow_missing is true and webhook doesn't exist, create a new one
			user, ok := ctx.Value(common.UserContextKey).(*store.UserMessage)
			if !ok {
				return nil, connect.NewError(connect.CodeInternal, errors.New("user not found"))
			}
			// Check if user has permission to update project (which includes adding webhooks)
			ok, err := s.iamManager.CheckPermission(ctx, iam.PermissionProjectsUpdate, user, project.ResourceID)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrap(err, "failed to check permission"))
			}
			if !ok {
				return nil, connect.NewError(connect.CodePermissionDenied, errors.Errorf("user does not have permission %q", iam.PermissionProjectsUpdate))
			}
			// Call AddWebhook instead since we're creating a new webhook
			return s.AddWebhook(ctx, connect.NewRequest(&v1pb.AddWebhookRequest{
				Project: fmt.Sprintf("projects/%s", project.ResourceID),
				Webhook: req.Msg.Webhook,
			}))
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("webhook %q not found", req.Msg.Webhook.Url))
	}

	update := &store.UpdateProjectWebhookMessage{}
	for _, path := range req.Msg.UpdateMask.Paths {
		switch path {
		case "type":
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("type cannot be updated"))
		case "title":
			update.Title = &req.Msg.Webhook.Title
		case "url":
			update.URL = &req.Msg.Webhook.Url
		case "notification_type":
			types, err := convertToActivityTypeStrings(req.Msg.Webhook.NotificationTypes)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			if len(types) == 0 {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("notification types should not be empty"))
			}
			update.Events = types
		case "direct_message":
			update.Payload = &storepb.ProjectWebhookPayload{
				DirectMessage: req.Msg.Webhook.DirectMessage,
			}
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid field %q", path))
		}
	}

	if _, err := s.store.UpdateProjectWebhookV2(ctx, project.ResourceID, webhook.ID, update); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	project, err = s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(convertToProject(project)), nil
}

// RemoveWebhook removes a webhook from a given project.
func (s *ProjectService) RemoveWebhook(ctx context.Context, req *connect.Request[v1pb.RemoveWebhookRequest]) (*connect.Response[v1pb.Project], error) {
	projectID, webhookID, err := common.GetProjectIDWebhookID(req.Msg.Webhook.Name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	webhookIDInt, err := strconv.Atoi(webhookID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid webhook id %q", webhookID))
	}

	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", webhookID))
	}
	if project.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q has been deleted", projectID))
	}

	webhook, err := s.store.GetProjectWebhookV2(ctx, &store.FindProjectWebhookMessage{
		ProjectID: &project.ResourceID,
		ID:        &webhookIDInt,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if webhook == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("webhook %q not found", req.Msg.Webhook.Url))
	}

	if err := s.store.DeleteProjectWebhookV2(ctx, project.ResourceID, webhook.ID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	project, err = s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(convertToProject(project)), nil
}

// TestWebhook tests a webhook.
func (s *ProjectService) TestWebhook(ctx context.Context, req *connect.Request[v1pb.TestWebhookRequest]) (*connect.Response[v1pb.TestWebhookResponse], error) {
	setting, err := s.store.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to get workspace setting"))
	}
	if setting.ExternalUrl == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.Errorf(setupExternalURLError))
	}

	projectID, err := common.GetProjectID(req.Msg.Project)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProjectV2(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", req.Msg.Project))
	}
	if project.Deleted {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q has been deleted", req.Msg.Project))
	}

	webhook, err := convertToStoreProjectWebhookMessage(req.Msg.Webhook)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	resp := &v1pb.TestWebhookResponse{}
	err = webhookplugin.Post(
		webhook.Type,
		webhookplugin.Context{
			URL:         webhook.URL,
			Level:       webhookplugin.WebhookInfo,
			EventType:   string(common.EventTypeIssueCreate),
			Title:       fmt.Sprintf("Test webhook %q", webhook.Title),
			TitleZh:     fmt.Sprintf("测试 webhook %q", webhook.Title),
			Description: "This is a test",
			Link:        fmt.Sprintf("%s/projects/%s/webhooks/%s", setting.ExternalUrl, project.ResourceID, fmt.Sprintf("%s-%d", slug.Make(webhook.Title), webhook.ID)),
			ActorID:     common.SystemBotID,
			ActorName:   "Bytebase",
			ActorEmail:  s.store.GetSystemBotUser(ctx).Email,
			CreatedTS:   time.Now().Unix(),
			Issue: &webhookplugin.Issue{
				ID:          1,
				Name:        "Test issue",
				Status:      "OPEN",
				Type:        "bb.issue.database.create",
				Description: "This is a test issue",
				Creator:     s.store.GetSystemBotUser(ctx),
			},

			Project: &webhookplugin.Project{
				Name:  common.FormatProject(project.ResourceID),
				Title: project.Title,
			},
		},
	)
	if err != nil {
		resp.Error = err.Error()
	}

	return connect.NewResponse(resp), nil
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
		Type:   tp,
		URL:    webhook.Url,
		Title:  webhook.Title,
		Events: activityTypes,
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
		case v1pb.Activity_ISSUE_CREATE:
			result = append(result, string(common.EventTypeIssueCreate))
		case v1pb.Activity_ISSUE_COMMENT_CREATE:
			result = append(result, string(common.EventTypeIssueCommentCreate))
		case v1pb.Activity_ISSUE_FIELD_UPDATE:
			result = append(result, string(common.EventTypeIssueUpdate))
		case v1pb.Activity_ISSUE_STATUS_UPDATE:
			result = append(result, string(common.EventTypeIssueStatusUpdate))
		case v1pb.Activity_ISSUE_APPROVAL_NOTIFY:
			result = append(result, string(common.EventTypeIssueApprovalCreate))
		case v1pb.Activity_ISSUE_PIPELINE_STAGE_STATUS_UPDATE:
			result = append(result, string(common.EventTypeStageStatusUpdate))
		case v1pb.Activity_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE:
			result = append(result, string(common.EventTypeTaskRunStatusUpdate))
		case v1pb.Activity_NOTIFY_ISSUE_APPROVED:
			result = append(result, string(common.EventTypeIssueApprovalPass))
		case v1pb.Activity_NOTIFY_PIPELINE_ROLLOUT:
			result = append(result, string(common.EventTypeIssueRolloutReady))
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
		case string(common.EventTypeIssueCreate):
			result = append(result, v1pb.Activity_ISSUE_CREATE)
		case string(common.EventTypeIssueCommentCreate):
			result = append(result, v1pb.Activity_ISSUE_COMMENT_CREATE)
		case string(common.EventTypeIssueUpdate):
			result = append(result, v1pb.Activity_ISSUE_FIELD_UPDATE)
		case string(common.EventTypeIssueStatusUpdate):
			result = append(result, v1pb.Activity_ISSUE_STATUS_UPDATE)
		case string(common.EventTypeIssueApprovalCreate):
			result = append(result, v1pb.Activity_ISSUE_APPROVAL_NOTIFY)
		case string(common.EventTypeStageStatusUpdate):
			result = append(result, v1pb.Activity_ISSUE_PIPELINE_STAGE_STATUS_UPDATE)
		case string(common.EventTypeTaskRunStatusUpdate):
			result = append(result, v1pb.Activity_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE)
		case string(common.EventTypeIssueApprovalPass):
			result = append(result, v1pb.Activity_NOTIFY_ISSUE_APPROVED)
		case string(common.EventTypeIssueRolloutReady):
			result = append(result, v1pb.Activity_NOTIFY_PIPELINE_ROLLOUT)
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
	case v1pb.Webhook_SLACK:
		return "bb.plugin.webhook.slack", nil
	case v1pb.Webhook_DISCORD:
		return "bb.plugin.webhook.discord", nil
	case v1pb.Webhook_TEAMS:
		return "bb.plugin.webhook.teams", nil
	case v1pb.Webhook_DINGTALK:
		return "bb.plugin.webhook.dingtalk", nil
	case v1pb.Webhook_FEISHU:
		return "bb.plugin.webhook.feishu", nil
	case v1pb.Webhook_WECOM:
		return "bb.plugin.webhook.wecom", nil
	case v1pb.Webhook_LARK:
		return "bb.plugin.webhook.lark", nil
	default:
		return "", common.Errorf(common.Invalid, "webhook type %q is not supported", tp)
	}
}

func convertWebhookTypeString(tp string) v1pb.Webhook_Type {
	switch tp {
	case "bb.plugin.webhook.slack":
		return v1pb.Webhook_SLACK
	case "bb.plugin.webhook.discord":
		return v1pb.Webhook_DISCORD
	case "bb.plugin.webhook.teams":
		return v1pb.Webhook_TEAMS
	case "bb.plugin.webhook.dingtalk":
		return v1pb.Webhook_DINGTALK
	case "bb.plugin.webhook.feishu":
		return v1pb.Webhook_FEISHU
	case "bb.plugin.webhook.wecom":
		return v1pb.Webhook_WECOM
	case "bb.plugin.webhook.lark":
		return v1pb.Webhook_LARK
	default:
		return v1pb.Webhook_TYPE_UNSPECIFIED
	}
}

func (s *ProjectService) getProjectMessage(ctx context.Context, name string) (*store.ProjectMessage, error) {
	projectID, err := common.GetProjectID(name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	find := &store.FindProjectMessage{
		ResourceID:  &projectID,
		ShowDeleted: true,
	}
	project, err := s.store.GetProjectV2(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", name))
	}

	return project, nil
}

func convertToV1MemberInBinding(ctx context.Context, stores *store.Store, member string) string {
	if strings.HasPrefix(member, common.UserNamePrefix) {
		userUID, err := common.GetUserID(member)
		if err != nil {
			slog.Error("failed to user id from member", slog.String("member", member), log.BBError(err))
			return ""
		}
		user, err := stores.GetUserByID(ctx, userUID)
		if err != nil {
			slog.Error("failed to get user", slog.String("member", member), log.BBError(err))
			return ""
		}
		if user == nil {
			return ""
		}
		return fmt.Sprintf("%s%s", common.UserBindingPrefix, user.Email)
	} else if strings.HasPrefix(member, common.GroupPrefix) {
		email, err := common.GetGroupEmail(member)
		if err != nil {
			slog.Error("failed to parse group email from member", slog.String("member", member), log.BBError(err))
			return ""
		}
		return fmt.Sprintf("%s%s", common.GroupBindingPrefix, email)
	}
	// handle allUsers.
	return member
}

func convertToV1IamPolicy(ctx context.Context, stores *store.Store, iamPolicy *store.IamPolicyMessage) (*v1pb.IamPolicy, error) {
	var bindings []*v1pb.Binding

	for _, binding := range iamPolicy.Policy.Bindings {
		var members []string
		for _, member := range binding.Members {
			memberInBinding := convertToV1MemberInBinding(ctx, stores, member)
			if memberInBinding == "" {
				continue
			}
			members = append(members, memberInBinding)
		}
		if len(members) == 0 {
			continue
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
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to create cel environment"))
			}
			ast, issues := e.Parse(v1pbBinding.Condition.Expression)
			if issues != nil && issues.Err() != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Errorf("failed to parse expression with error: %v", issues.Err()))
			}
			expr, err := cel.AstToParsedExpr(ast)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert ast to parsed expression with error"))
			}
			v1pbBinding.ParsedExpr = expr.Expr
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
		for _, member := range utils.Uniq(binding.Members) {
			storeMember, err := convertToStoreIamPolicyMember(ctx, stores, member)
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to convert iam member with error"))
			}
			members = append(members, storeMember)
		}
		if len(members) == 0 {
			continue
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

	if len(bindings) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("policy binding is empty"))
	}

	return &storepb.IamPolicy{
		Bindings: bindings,
	}, nil
}

func convertToStoreIamPolicyMember(ctx context.Context, stores *store.Store, member string) (string, error) {
	if strings.HasPrefix(member, common.UserBindingPrefix) {
		email := strings.TrimPrefix(member, common.UserBindingPrefix)
		user, err := stores.GetUserByEmail(ctx, email)
		if err != nil {
			return "", connect.NewError(connect.CodeInternal, err)
		}
		if user == nil {
			return "", connect.NewError(connect.CodeNotFound, errors.Errorf("user %q not found", member))
		}
		return common.FormatUserUID(user.ID), nil
	} else if strings.HasPrefix(member, common.GroupBindingPrefix) {
		email := strings.TrimPrefix(member, common.GroupBindingPrefix)
		return common.FormatGroupEmail(email), nil
	} else if member == common.AllUsers {
		return member, nil
	}
	return "", connect.NewError(connect.CodeInvalidArgument, errors.Errorf("unsupport member %s", member))
}

func convertToProject(projectMessage *store.ProjectMessage) *v1pb.Project {
	var projectWebhooks []*v1pb.Webhook
	for _, webhook := range projectMessage.Webhooks {
		projectWebhooks = append(projectWebhooks, &v1pb.Webhook{
			Name:              fmt.Sprintf("%s/%s%d", common.FormatProject(projectMessage.ResourceID), common.WebhookIDPrefix, webhook.ID),
			Type:              convertWebhookTypeString(webhook.Type),
			Title:             webhook.Title,
			Url:               webhook.URL,
			NotificationTypes: convertNotificationTypeStrings(webhook.Events),
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
		Name:                       common.FormatProject(projectMessage.ResourceID),
		State:                      convertDeletedToState(projectMessage.Deleted),
		Title:                      projectMessage.Title,
		Webhooks:                   projectWebhooks,
		DataClassificationConfigId: projectMessage.DataClassificationConfigID,
		IssueLabels:                issueLabels,
		ForceIssueLabels:           projectMessage.Setting.ForceIssueLabels,
		AllowModifyStatement:       projectMessage.Setting.AllowModifyStatement,
		AutoResolveIssue:           projectMessage.Setting.AutoResolveIssue,
		EnforceIssueTitle:          projectMessage.Setting.EnforceIssueTitle,
		AutoEnableBackup:           projectMessage.Setting.AutoEnableBackup,
		SkipBackupErrors:           projectMessage.Setting.SkipBackupErrors,
		PostgresDatabaseTenantMode: projectMessage.Setting.PostgresDatabaseTenantMode,
		AllowSelfApproval:          projectMessage.Setting.AllowSelfApproval,
		ExecutionRetryPolicy:       convertToV1ExecutionRetryPolicy(projectMessage.Setting.ExecutionRetryPolicy),
		CiSamplingSize:             projectMessage.Setting.CiSamplingSize,
		ParallelTasksPerRollout:    projectMessage.Setting.ParallelTasksPerRollout,
		Labels:                     projectMessage.Setting.Labels,
	}
}

func convertToV1ExecutionRetryPolicy(policy *storepb.Project_ExecutionRetryPolicy) *v1pb.Project_ExecutionRetryPolicy {
	if policy == nil {
		return &v1pb.Project_ExecutionRetryPolicy{
			MaximumRetries: 0,
		}
	}
	return &v1pb.Project_ExecutionRetryPolicy{
		MaximumRetries: policy.MaximumRetries,
	}
}

func convertToStoreExecutionRetryPolicy(policy *v1pb.Project_ExecutionRetryPolicy) *storepb.Project_ExecutionRetryPolicy {
	if policy == nil {
		return &storepb.Project_ExecutionRetryPolicy{
			MaximumRetries: 0,
		}
	}
	return &storepb.Project_ExecutionRetryPolicy{
		MaximumRetries: policy.MaximumRetries,
	}
}

func convertToProjectMessage(resourceID string, project *v1pb.Project) *store.ProjectMessage {
	setting := &storepb.Project{
		AllowModifyStatement:       project.AllowModifyStatement,
		AutoResolveIssue:           project.AutoResolveIssue,
		EnforceIssueTitle:          project.EnforceIssueTitle,
		AutoEnableBackup:           project.AutoEnableBackup,
		SkipBackupErrors:           project.SkipBackupErrors,
		PostgresDatabaseTenantMode: project.PostgresDatabaseTenantMode,
		AllowSelfApproval:          project.AllowSelfApproval,
		CiSamplingSize:             project.CiSamplingSize,
		ParallelTasksPerRollout:    project.ParallelTasksPerRollout,
		Labels:                     project.Labels,
	}
	return &store.ProjectMessage{
		ResourceID: resourceID,
		Title:      project.Title,
		Setting:    setting,
	}
}

func getBindingIdentifier(role string, condition *expr.Expr) string {
	ids := []string{
		fmt.Sprintf("[role] %s", role),
	}
	if condition != nil {
		ids = append(
			ids,
			fmt.Sprintf("[title] %s", condition.Title),
			fmt.Sprintf("[description] %s", condition.Description),
			fmt.Sprintf("[expression] %s", condition.Expression),
		)
	}
	return strings.Join(ids, ";")
}

func validateIAMPolicy(
	ctx context.Context,
	stores *store.Store,
	iamManager *iam.Manager,
	policy *v1pb.IamPolicy,
	oldPolicyMessage *store.IamPolicyMessage,
) (bool, error) {
	if policy == nil {
		return false, connect.NewError(connect.CodeInvalidArgument, errors.New("IAM Policy is required"))
	}
	if len(policy.Bindings) == 0 {
		return false, connect.NewError(connect.CodeInvalidArgument, errors.New("IAM Binding is empty"))
	}

	generalSetting, err := stores.GetWorkspaceGeneralSetting(ctx)
	if err != nil {
		return false, connect.NewError(connect.CodeInternal, errors.New("failed to get workspace general setting"))
	}
	var maximumRoleExpiration *durationpb.Duration
	if generalSetting != nil {
		maximumRoleExpiration = generalSetting.MaximumRoleExpiration
	}

	roleMessages, err := stores.ListRoles(ctx)
	if err != nil {
		return false, connect.NewError(connect.CodeInternal, errors.Wrapf(err, "failed to list roles"))
	}
	roleMessages = append(roleMessages, iamManager.PredefinedRoles...)

	existingBindings := make(map[string]bool)
	for _, oldBinding := range oldPolicyMessage.Policy.Bindings {
		identifier := getBindingIdentifier(oldBinding.Role, oldBinding.Condition)
		existingBindings[identifier] = true
	}

	existProjectOwner := false
	bindings := []*v1pb.Binding{}
	for _, binding := range policy.Bindings {
		if len(binding.Members) == 0 {
			continue
		}
		if binding.Role == fmt.Sprintf("roles/%s", common.ProjectOwner) {
			existProjectOwner = true
		}

		identifier := getBindingIdentifier(binding.Role, binding.Condition)
		if !existingBindings[identifier] {
			bindings = append(bindings, binding)
		}
	}

	return existProjectOwner, validateBindings(bindings, roleMessages, maximumRoleExpiration)
}

func validateBindings(bindings []*v1pb.Binding, roles []*store.RoleMessage, maximumRoleExpiration *durationpb.Duration) error {
	existingRoles := make(map[string]bool)
	for _, role := range roles {
		existingRoles[common.FormatRole(role.ResourceID)] = true
	}

	for _, binding := range bindings {
		if binding.Role == "" {
			return connect.NewError(connect.CodeInvalidArgument, errors.New("IAM Binding role is required"))
		}
		if !existingRoles[binding.Role] {
			return connect.NewError(connect.CodeInvalidArgument, errors.Errorf("IAM Binding role %s does not exist", binding.Role))
		}

		if _, err := common.ValidateProjectMemberCELExpr(binding.Condition); err != nil {
			return err
		}

		if binding.Role != fmt.Sprintf("roles/%s", common.ProjectOwner) && maximumRoleExpiration != nil {
			// Only validate when maximumRoleExpiration is set and the role is not project owner.
			if err := validateExpirationInExpression(binding.GetCondition().GetExpression(), maximumRoleExpiration); err != nil {
				return connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "failed to validate expiration for binding %v", binding.Role))
			}
		}
	}
	return nil
}

// validateExpirationInExpression validates the IAM policy expression.
// Currently only validate the following expression:
// * request.time < timestamp("2021-01-01T00:00:00Z")
//
// Other expressions will be ignored.
func validateExpirationInExpression(expr string, maximumRoleExpiration *durationpb.Duration) error {
	if maximumRoleExpiration == nil {
		return nil
	}
	if !strings.Contains(expr, "request.time") {
		return errors.Errorf("request.time is required")
	}
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
					default:
						// Other arg kinds
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

func validateMember(member string) error {
	if member == common.AllUsers {
		return nil
	}

	userIdentifierMap := map[string]bool{
		common.UserBindingPrefix:  true,
		common.GroupBindingPrefix: true,
	}
	for prefix := range userIdentifierMap {
		if strings.HasPrefix(member, prefix) && len(member[len(prefix):]) > 0 {
			return nil
		}
	}
	return errors.Errorf("invalid user %s", member)
}

// validateProjectLabels validates the project labels according to the requirements.
func validateProjectLabels(labels map[string]string) error {
	if len(labels) > 64 {
		return errors.Errorf("maximum 64 labels allowed, got %d", len(labels))
	}

	// Key pattern: must start with lowercase letter, then lowercase letters, numbers, underscores, dashes (max 63 chars)
	keyPattern := `^[a-z][a-z0-9_-]{0,62}$`
	// Value pattern: letters, numbers, underscores, dashes (max 63 chars, can be empty)
	valuePattern := `^[a-zA-Z0-9_-]{0,63}$`

	keyRegex := regexp.MustCompile(keyPattern)
	valueRegex := regexp.MustCompile(valuePattern)

	for key, value := range labels {
		if !keyRegex.MatchString(key) {
			return errors.Errorf("invalid label key %q: must start with lowercase letter and contain only lowercase letters, numbers, underscores, and dashes (max 63 chars)", key)
		}
		if !valueRegex.MatchString(value) {
			return errors.Errorf("invalid label value %q for key %q: must contain only letters, numbers, underscores, and dashes (max 63 chars)", value, key)
		}
	}
	return nil
}
