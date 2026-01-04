package v1

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/iam"
	"github.com/bytebase/bytebase/backend/utils"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	webhookplugin "github.com/bytebase/bytebase/backend/plugin/webhook"
	"github.com/bytebase/bytebase/backend/store"
)

// ProjectService implements the project service.
type ProjectService struct {
	v1connect.UnimplementedProjectServiceHandler
	store      *store.Store
	profile    *config.Profile
	iamManager *iam.Manager
}

// NewProjectService creates a new ProjectService.
func NewProjectService(
	store *store.Store,
	profile *config.Profile,
	iamManager *iam.Manager,
) *ProjectService {
	return &ProjectService{
		store:      store,
		profile:    profile,
		iamManager: iamManager,
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

// BatchGetProjects gets projects in batch.
func (s *ProjectService) BatchGetProjects(ctx context.Context, req *connect.Request[v1pb.BatchGetProjectsRequest]) (*connect.Response[v1pb.BatchGetProjectsResponse], error) {
	projects := make([]*v1pb.Project, 0, len(req.Msg.Names))
	for _, name := range req.Msg.Names {
		project, err := s.getProjectMessage(ctx, name)
		if err != nil {
			return nil, err
		}
		if project.Deleted {
			continue
		}
		projects = append(projects, convertToProject(project))
	}
	return connect.NewResponse(&v1pb.BatchGetProjectsResponse{Projects: projects}), nil
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
	filterQ, err := store.GetListProjectFilter(req.Msg.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	find.FilterQ = filterQ

	orderByKeys, err := store.GetProjectOrders(req.Msg.OrderBy)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	find.OrderByKeys = orderByKeys

	projects, err := s.store.ListProjects(ctx, find)
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

// SearchProjects searches all projects on which the user has bb.projects.get permission.
func (s *ProjectService) SearchProjects(ctx context.Context, req *connect.Request[v1pb.SearchProjectsRequest]) (*connect.Response[v1pb.SearchProjectsResponse], error) {
	user, ok := GetUserFromContext(ctx)
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
	filterQ, err := store.GetListProjectFilter(req.Msg.Filter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	find.FilterQ = filterQ

	orderByKeys, err := store.GetProjectOrders(req.Msg.OrderBy)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	find.OrderByKeys = orderByKeys

	projects, err := s.store.ListProjects(ctx, find)
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
		if err := validateLabels(req.Msg.Project.Labels); err != nil {
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

	user, ok := GetUserFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.Errorf("user not found"))
	}
	project, err := s.store.CreateProject(ctx,
		projectMessage,
		user,
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
			projectID, perr := common.GetProjectID(req.Msg.Project.Name)
			if perr != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, perr)
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
		case "enforce_issue_title":
			projectSettings := project.Setting
			projectSettings.EnforceIssueTitle = req.Msg.Project.EnforceIssueTitle
			patch.Setting = projectSettings
		case "enforce_sql_review":
			projectSettings := project.Setting
			projectSettings.EnforceSqlReview = req.Msg.Project.EnforceSqlReview
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
		case "require_issue_approval":
			projectSettings := project.Setting
			projectSettings.RequireIssueApproval = req.Msg.Project.RequireIssueApproval
			patch.Setting = projectSettings
		case "require_plan_check_no_error":
			projectSettings := project.Setting
			projectSettings.RequirePlanCheckNoError = req.Msg.Project.RequirePlanCheckNoError
			patch.Setting = projectSettings
		case "labels":
			if err := validateLabels(req.Msg.Project.Labels); err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			projectSettings := project.Setting
			projectSettings.Labels = req.Msg.Project.Labels
			patch.Setting = projectSettings
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf(`unsupport update_mask "%s"`, path))
		}
	}

	projects, err := s.store.UpdateProjects(ctx, patch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	project = projects[0]
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
		openIssues, err := s.store.ListIssues(ctx, &store.FindIssueMessage{ProjectIDs: &[]string{project.ResourceID}, StatusList: []storepb.Issue_Status{storepb.Issue_OPEN}})
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

	if _, err := s.store.UpdateProjects(ctx, &store.UpdateProjectMessage{
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

	projects, err := s.store.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: project.ResourceID,
		Delete:     &undeletePatch,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	project = projects[0]
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
			openIssues, err := s.store.ListIssues(ctx, &store.FindIssueMessage{
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

	if _, err := s.store.UpdateProjects(ctx, updatePatches...); err != nil {
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
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
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

	iamPolicy, err := convertToV1IamPolicy(policy)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(iamPolicy), nil
}

// SetIamPolicy sets the IAM policy for a project.
func (s *ProjectService) SetIamPolicy(ctx context.Context, req *connect.Request[v1pb.SetIamPolicyRequest]) (*connect.Response[v1pb.IamPolicy], error) {
	projectID, err := common.GetProjectID(req.Msg.Resource)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
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

	policy, err := convertToStoreIamPolicy(req.Msg.Policy)
	if err != nil {
		return nil, err
	}

	policyPayload, err := protojson.Marshal(policy)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if _, err := s.store.CreatePolicy(ctx, &store.PolicyMessage{
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

	iamPolicy, err := convertToV1IamPolicy(iamPolicyMessage)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(iamPolicy), nil
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
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
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

	if _, err := utils.GetEffectiveExternalURL(ctx, s.store, s.profile); err != nil {
		return nil, err
	}

	create, err := convertToStoreProjectWebhookMessage(req.Msg.Webhook)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Validate webhook URL against allowed domains
	if err := webhookplugin.ValidateWebhookURL(create.Payload.GetType(), create.Payload.GetUrl()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid webhook URL"))
	}

	if _, err := s.store.CreateProjectWebhook(ctx, project.ResourceID, create); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	project, err = s.store.GetProject(ctx, &store.FindProjectMessage{
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

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
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

	if _, err := utils.GetEffectiveExternalURL(ctx, s.store, s.profile); err != nil {
		return nil, err
	}

	webhook, err := s.store.GetProjectWebhook(ctx, &store.FindProjectWebhookMessage{
		ProjectID: &project.ResourceID,
		ID:        &webhookIDInt,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if webhook == nil {
		if req.Msg.AllowMissing {
			// When allow_missing is true and webhook doesn't exist, create a new one
			// Call AddWebhook instead since we're creating a new webhook
			return s.AddWebhook(ctx, connect.NewRequest(&v1pb.AddWebhookRequest{
				Project: fmt.Sprintf("projects/%s", project.ResourceID),
				Webhook: req.Msg.Webhook,
			}))
		}
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("webhook %q not found", req.Msg.Webhook.Url))
	}

	// Start with existing webhook payload
	// nolint:revive
	updatedPayload := proto.Clone(webhook.Payload).(*storepb.ProjectWebhook)

	for _, path := range req.Msg.UpdateMask.Paths {
		switch path {
		case "type":
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("type cannot be updated"))
		case "title":
			updatedPayload.Title = req.Msg.Webhook.Title
		case "url":
			updatedPayload.Url = req.Msg.Webhook.Url
			// Validate webhook URL against allowed domains
			if err := webhookplugin.ValidateWebhookURL(updatedPayload.Type, updatedPayload.Url); err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid webhook URL"))
			}
		case "notification_type":
			types, err := convertToStoreActivityTypes(req.Msg.Webhook.NotificationTypes)
			if err != nil {
				return nil, connect.NewError(connect.CodeInvalidArgument, err)
			}
			if len(types) == 0 {
				return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("notification types should not be empty"))
			}
			updatedPayload.Activities = types
		case "direct_message":
			updatedPayload.DirectMessage = req.Msg.Webhook.DirectMessage
		default:
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.Errorf("invalid field %q", path))
		}
	}

	update := &store.UpdateProjectWebhookMessage{
		Payload: updatedPayload,
	}

	if _, err := s.store.UpdateProjectWebhook(ctx, project.ResourceID, webhook.ID, update); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	project, err = s.store.GetProject(ctx, &store.FindProjectMessage{
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

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
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

	webhook, err := s.store.GetProjectWebhook(ctx, &store.FindProjectWebhookMessage{
		ProjectID: &project.ResourceID,
		ID:        &webhookIDInt,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if webhook == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("webhook %q not found", req.Msg.Webhook.Url))
	}

	if err := s.store.DeleteProjectWebhook(ctx, project.ResourceID, webhook.ID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	project, err = s.store.GetProject(ctx, &store.FindProjectMessage{
		ResourceID: &projectID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(convertToProject(project)), nil
}

// TestWebhook tests a webhook.
func (s *ProjectService) TestWebhook(ctx context.Context, req *connect.Request[v1pb.TestWebhookRequest]) (*connect.Response[v1pb.TestWebhookResponse], error) {
	externalURL, err := utils.GetEffectiveExternalURL(ctx, s.store, s.profile)
	if err != nil {
		return nil, err
	}

	projectID, err := common.GetProjectID(req.Msg.Project)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{
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

	// Validate webhook URL against allowed domains
	if err := webhookplugin.ValidateWebhookURL(webhook.Payload.GetType(), webhook.Payload.GetUrl()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Wrapf(err, "invalid webhook URL"))
	}

	resp := &v1pb.TestWebhookResponse{}
	err = webhookplugin.Post(
		webhook.Payload.GetType(),
		webhookplugin.Context{
			URL:         webhook.Payload.GetUrl(),
			Level:       webhookplugin.WebhookInfo,
			EventType:   storepb.Activity_ISSUE_CREATED.String(),
			Title:       fmt.Sprintf("Test webhook %q", webhook.Payload.GetTitle()),
			TitleZh:     fmt.Sprintf("测试 webhook %q", webhook.Payload.GetTitle()),
			Description: "This is a test",
			Link:        fmt.Sprintf("%s/projects/%s/webhooks/%s", externalURL, project.ResourceID, fmt.Sprintf("%s-%d", slug.Make(webhook.Payload.GetTitle()), webhook.ID)),
			ActorID:     common.SystemBotID,
			ActorName:   "Bytebase",
			ActorEmail:  common.SystemBotEmail,
			CreatedTS:   time.Now().Unix(),
			Issue: &webhookplugin.Issue{
				ID:          1,
				Name:        "Test issue",
				Status:      "OPEN",
				Type:        "bb.issue.database.create",
				Description: "This is a test issue",
				Creator: webhookplugin.Creator{
					Name:  "Bytebase",
					Email: common.SystemBotEmail,
				},
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

func (s *ProjectService) getProjectMessage(ctx context.Context, name string) (*store.ProjectMessage, error) {
	projectID, err := common.GetProjectID(name)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	find := &store.FindProjectMessage{
		ResourceID:  &projectID,
		ShowDeleted: true,
	}
	project, err := s.store.GetProject(ctx, find)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if project == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.Errorf("project %q not found", name))
	}

	return project, nil
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

	workspaceProfileSetting, err := stores.GetWorkspaceProfileSetting(ctx)
	if err != nil {
		return false, connect.NewError(connect.CodeInternal, errors.New("failed to get workspace profile setting"))
	}
	var maximumRoleExpiration *durationpb.Duration
	if workspaceProfileSetting != nil {
		maximumRoleExpiration = workspaceProfileSetting.MaximumRoleExpiration
	}

	roleMessages, err := stores.ListRoles(ctx, &store.FindRoleMessage{})
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
