package v1

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	"github.com/bytebase/bytebase/backend/utils"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

type ContextProvider struct {
	s *store.Store
}

func NewContextProvider(s *store.Store) *ContextProvider {
	return &ContextProvider{
		s: s,
	}
}

// ContextProvider is the unary interceptor for gRPC API.
func (p *ContextProvider) UnaryInterceptor(ctx context.Context, request any, serverInfo *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	projectIDs, err := p.do(ctx, serverInfo.FullMethod, request)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get project ids for method %q, err: %v", serverInfo.FullMethod, err)
	}

	ctx = common.WithProjectIDs(ctx, projectIDs)

	return handler(ctx, request)
}

// ACLStreamInterceptor is the unary interceptor for gRPC API.
func (p *ContextProvider) StreamInterceptor(request any, ss grpc.ServerStream, serverInfo *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	ctx := ss.Context()
	projectIDs, err := p.do(ctx, serverInfo.FullMethod, request)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get project ids for method %q, err: %v", serverInfo.FullMethod, err)
	}

	ctx = common.WithProjectIDs(ctx, projectIDs)
	ss = overrideStream{ServerStream: ss, childCtx: ctx}

	return handler(request, ss)
}

func (p *ContextProvider) do(ctx context.Context, fullMethod string, req any) ([]string, error) {
	switch fullMethod {
	// skip workspace-level methods
	case
		v1pb.InstanceService_GetInstance_FullMethodName,
		v1pb.InstanceRoleService_GetInstanceRole_FullMethodName,
		v1pb.InstanceRoleService_ListInstanceRoles_FullMethodName,
		v1pb.ProjectService_ListProjects_FullMethodName,
		v1pb.ProjectService_CreateProject_FullMethodName,
		v1pb.ProjectService_DeleteProject_FullMethodName,
		v1pb.ProjectService_UndeleteProject_FullMethodName,
		v1pb.SQLService_AdminExecute_FullMethodName,
		v1pb.InstanceService_ListInstances_FullMethodName,
		v1pb.InstanceService_CreateInstance_FullMethodName,
		v1pb.InstanceService_UpdateInstance_FullMethodName,
		v1pb.InstanceService_DeleteInstance_FullMethodName,
		v1pb.InstanceService_UndeleteInstance_FullMethodName,
		v1pb.InstanceService_SyncInstance_FullMethodName,
		v1pb.InstanceService_BatchSyncInstance_FullMethodName,
		v1pb.InstanceService_AddDataSource_FullMethodName,
		v1pb.InstanceService_RemoveDataSource_FullMethodName,
		v1pb.InstanceService_UpdateDataSource_FullMethodName,
		v1pb.InstanceService_SyncSlowQueries_FullMethodName,
		v1pb.InstanceRoleService_CreateInstanceRole_FullMethodName,
		v1pb.InstanceRoleService_UpdateInstanceRole_FullMethodName,
		v1pb.InstanceRoleService_DeleteInstanceRole_FullMethodName,
		v1pb.InstanceRoleService_UndeleteInstanceRole_FullMethodName,
		v1pb.EnvironmentService_CreateEnvironment_FullMethodName,
		v1pb.EnvironmentService_UpdateEnvironment_FullMethodName,
		v1pb.EnvironmentService_DeleteEnvironment_FullMethodName,
		v1pb.EnvironmentService_UndeleteEnvironment_FullMethodName,
		v1pb.EnvironmentService_GetEnvironment_FullMethodName,
		v1pb.EnvironmentService_ListEnvironments_FullMethodName,
		v1pb.SettingService_ListSettings_FullMethodName,
		v1pb.SettingService_GetSetting_FullMethodName,
		v1pb.SettingService_UpdateSetting_FullMethodName,
		v1pb.OrgPolicyService_ListPolicies_FullMethodName,
		v1pb.OrgPolicyService_GetPolicy_FullMethodName,
		v1pb.OrgPolicyService_CreatePolicy_FullMethodName,
		v1pb.OrgPolicyService_UpdatePolicy_FullMethodName,
		v1pb.OrgPolicyService_DeletePolicy_FullMethodName,
		v1pb.WorkspaceService_GetIamPolicy_FullMethodName,
		v1pb.WorkspaceService_SetIamPolicy_FullMethodName,
		v1pb.ReviewConfigService_CreateReviewConfig_FullMethodName,
		v1pb.ReviewConfigService_ListReviewConfigs_FullMethodName,
		v1pb.ReviewConfigService_GetReviewConfig_FullMethodName,
		v1pb.ReviewConfigService_UpdateReviewConfig_FullMethodName,
		v1pb.ReviewConfigService_DeleteReviewConfig_FullMethodName,
		v1pb.IdentityProviderService_GetIdentityProvider_FullMethodName,
		v1pb.IdentityProviderService_CreateIdentityProvider_FullMethodName,
		v1pb.IdentityProviderService_UpdateIdentityProvider_FullMethodName,
		v1pb.IdentityProviderService_DeleteIdentityProvider_FullMethodName,
		v1pb.IdentityProviderService_UndeleteIdentityProvider_FullMethodName,
		v1pb.IdentityProviderService_TestIdentityProvider_FullMethodName,
		v1pb.VCSProviderService_ListVCSProviders_FullMethodName,
		v1pb.VCSProviderService_GetVCSProvider_FullMethodName,
		v1pb.VCSProviderService_CreateVCSProvider_FullMethodName,
		v1pb.VCSProviderService_UpdateVCSProvider_FullMethodName,
		v1pb.VCSProviderService_DeleteVCSProvider_FullMethodName,
		v1pb.VCSProviderService_SearchVCSProviderRepositories_FullMethodName,
		v1pb.VCSProviderService_ListVCSConnectorsInProvider_FullMethodName,
		v1pb.RiskService_ListRisks_FullMethodName,
		v1pb.RiskService_CreateRisk_FullMethodName,
		v1pb.RiskService_UpdateRisk_FullMethodName,
		v1pb.RiskService_DeleteRisk_FullMethodName,
		v1pb.RoleService_ListRoles_FullMethodName,
		v1pb.RoleService_CreateRole_FullMethodName,
		v1pb.RoleService_UpdateRole_FullMethodName,
		v1pb.RoleService_DeleteRole_FullMethodName,
		v1pb.UserGroupService_CreateUserGroup_FullMethodName,
		v1pb.UserGroupService_GetUserGroup_FullMethodName,
		v1pb.UserGroupService_DeleteUserGroup_FullMethodName,
		v1pb.UserGroupService_UpdateUserGroup_FullMethodName,
		v1pb.UserGroupService_ListUserGroups_FullMethodName:
		return nil, nil

	case
		v1pb.SQLService_Query_FullMethodName,
		v1pb.SQLService_Export_FullMethodName,
		v1pb.DatabaseService_GetDatabase_FullMethodName,
		v1pb.DatabaseService_UpdateDatabase_FullMethodName,
		v1pb.DatabaseService_BatchUpdateDatabases_FullMethodName,
		v1pb.DatabaseService_DiffSchema_FullMethodName,
		v1pb.DatabaseService_SyncDatabase_FullMethodName,
		v1pb.DatabaseService_GetDatabaseMetadata_FullMethodName,
		v1pb.DatabaseService_UpdateDatabaseMetadata_FullMethodName,
		v1pb.DatabaseService_GetDatabaseSchema_FullMethodName,
		v1pb.DatabaseService_ListSecrets_FullMethodName,
		v1pb.DatabaseService_UpdateSecret_FullMethodName,
		v1pb.DatabaseService_DeleteSecret_FullMethodName,
		v1pb.DatabaseService_AdviseIndex_FullMethodName,
		v1pb.DatabaseService_ListChangeHistories_FullMethodName,
		v1pb.DatabaseService_GetChangeHistory_FullMethodName:
		return p.getProjectIDsForDatabase(ctx, req)

	case
		v1pb.VCSConnectorService_CreateVCSConnector_FullMethodName,
		v1pb.VCSConnectorService_GetVCSConnector_FullMethodName,
		v1pb.VCSConnectorService_ListVCSConnectors_FullMethodName,
		v1pb.VCSConnectorService_UpdateVCSConnector_FullMethodName,
		v1pb.VCSConnectorService_DeleteVCSConnector_FullMethodName:
		return p.getProjectIDsForVCSConnectorService(ctx, req)

	case
		v1pb.ChangelistService_CreateChangelist_FullMethodName,
		v1pb.ChangelistService_ListChangelists_FullMethodName,
		v1pb.ChangelistService_GetChangelist_FullMethodName:
		return p.getProjectIDsForChangelistService(ctx, req)

	case
		v1pb.BranchService_ListBranches_FullMethodName,
		v1pb.BranchService_GetBranch_FullMethodName,
		v1pb.BranchService_CreateBranch_FullMethodName:
		return p.getProjectIDsForBranchService(ctx, req)

	case
		v1pb.RolloutService_GetRollout_FullMethodName,
		v1pb.RolloutService_CreateRollout_FullMethodName,
		v1pb.RolloutService_PreviewRollout_FullMethodName,
		v1pb.RolloutService_ListTaskRuns_FullMethodName,
		v1pb.RolloutService_GetTaskRunLog_FullMethodName:
		return p.getProjectIDsForRolloutService(ctx, req)

	case
		v1pb.PlanService_GetPlan_FullMethodName,
		v1pb.PlanService_CreatePlan_FullMethodName,
		v1pb.PlanService_ListPlanCheckRuns_FullMethodName,
		v1pb.PlanService_RunPlanChecks_FullMethodName:
		return p.getProjectIDsForPlanService(ctx, req)

	case
		v1pb.ProjectService_GetProject_FullMethodName,
		v1pb.ProjectService_UpdateProject_FullMethodName,
		v1pb.ProjectService_SetIamPolicy_FullMethodName,
		v1pb.ProjectService_GetIamPolicy_FullMethodName,
		v1pb.ProjectService_BatchGetIamPolicy_FullMethodName,
		v1pb.ProjectService_GetDeploymentConfig_FullMethodName,
		v1pb.ProjectService_UpdateDeploymentConfig_FullMethodName,
		v1pb.ProjectService_AddWebhook_FullMethodName,
		v1pb.ProjectService_UpdateWebhook_FullMethodName,
		v1pb.ProjectService_RemoveWebhook_FullMethodName,
		v1pb.ProjectService_TestWebhook_FullMethodName,

		v1pb.ProjectService_GetDatabaseGroup_FullMethodName,
		v1pb.ProjectService_CreateDatabaseGroup_FullMethodName,
		v1pb.ProjectService_UpdateDatabaseGroup_FullMethodName,
		v1pb.ProjectService_DeleteDatabaseGroup_FullMethodName,
		v1pb.ProjectService_ListDatabaseGroups_FullMethodName,

		v1pb.ProjectService_GetProjectProtectionRules_FullMethodName,
		v1pb.ProjectService_UpdateProjectProtectionRules_FullMethodName:

		return p.getProjectIDsForProjectService(ctx, req)
	}

	return nil, nil
}

func (*ContextProvider) getProjectIDsForVCSConnectorService(_ context.Context, req any) ([]string, error) {
	var projects, vcsConnectors []string
	switch r := req.(type) {
	case *v1pb.CreateVCSConnectorRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.ListVCSConnectorsRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.GetVCSConnectorRequest:
		vcsConnectors = append(vcsConnectors, r.GetName())
	case *v1pb.UpdateVCSConnectorRequest:
		vcsConnectors = append(vcsConnectors, r.GetVcsConnector().GetName())
	case *v1pb.DeleteVCSConnectorRequest:
		vcsConnectors = append(vcsConnectors, r.GetName())
	}

	var projectIDs []string
	for _, vcsConnector := range vcsConnectors {
		projectID, _, err := common.GetProjectVCSConnectorID(vcsConnector)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse vcsConnector %q", vcsConnector)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, project := range projects {
		projectID, err := common.GetProjectID(project)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse project %q", project)
		}
		projectIDs = append(projectIDs, projectID)
	}

	return utils.Uniq(projectIDs), nil
}

func (*ContextProvider) getProjectIDsForChangelistService(_ context.Context, req any) ([]string, error) {
	var projects, changelists []string
	switch r := req.(type) {
	case *v1pb.CreateChangelistRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.ListChangelistsRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.GetChangelistRequest:
		changelists = append(changelists, r.GetName())
	}

	var projectIDs []string
	for _, changelist := range changelists {
		projectID, _, err := common.GetProjectIDChangelistID(changelist)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse changelist %q", changelist)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, project := range projects {
		projectID, err := common.GetProjectID(project)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse project %q", project)
		}
		projectIDs = append(projectIDs, projectID)
	}

	return utils.Uniq(projectIDs), nil
}

func (*ContextProvider) getProjectIDsForRolloutService(_ context.Context, req any) ([]string, error) {
	var projects, rollouts, plans, tasks, taskRuns []string
	switch r := req.(type) {
	case *v1pb.GetRolloutRequest:
		rollouts = append(rollouts, r.GetName())
	case *v1pb.CreateRolloutRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.PreviewRolloutRequest:
		projects = append(projects, r.GetProject())
	case *v1pb.GetPlanRequest:
		plans = append(plans, r.GetName())
	case *v1pb.CreatePlanRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.ListTaskRunsRequest:
		tasks = append(tasks, r.GetParent())
	case *v1pb.GetTaskRunLogRequest:
		taskRuns = append(taskRuns, r.GetParent())
	case *v1pb.ListPlanCheckRunsRequest:
		plans = append(plans, r.GetParent())
	case *v1pb.RunPlanChecksRequest:
		plans = append(plans, r.GetName())
	}

	var projectIDs []string
	for _, plan := range plans {
		projectID, _, err := common.GetProjectIDPlanID(plan)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse plan %q", plan)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, project := range projects {
		projectID, err := common.GetProjectID(project)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse project %q", project)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, rollout := range rollouts {
		projectID, _, err := common.GetProjectIDRolloutID(rollout)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse rollout %q", rollout)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, task := range tasks {
		projectID, _, _, _, err := common.GetProjectIDRolloutIDMaybeStageIDMaybeTaskID(task)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse task %q", task)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, taskRun := range taskRuns {
		projectID, _, _, _, _, err := common.GetProjectIDRolloutIDStageIDTaskIDTaskRunID(taskRun)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse taskRun %q", taskRun)
		}
		projectIDs = append(projectIDs, projectID)
	}

	return utils.Uniq(projectIDs), nil
}

func (*ContextProvider) getProjectIDsForPlanService(_ context.Context, req any) ([]string, error) {
	var projects, plans []string
	switch r := req.(type) {
	case *v1pb.GetPlanRequest:
		plans = append(plans, r.GetName())
	case *v1pb.CreatePlanRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.ListPlanCheckRunsRequest:
		plans = append(plans, r.GetParent())
	case *v1pb.RunPlanChecksRequest:
		plans = append(plans, r.GetName())
	}

	var projectIDs []string
	for _, plan := range plans {
		projectID, _, err := common.GetProjectIDPlanID(plan)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse plan %q", plan)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, project := range projects {
		projectID, err := common.GetProjectID(project)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse project %q", project)
		}
		projectIDs = append(projectIDs, projectID)
	}

	return utils.Uniq(projectIDs), nil
}

func (*ContextProvider) getProjectIDsForBranchService(_ context.Context, req any) ([]string, error) {
	var projects, branches []string
	switch r := req.(type) {
	case *v1pb.GetBranchRequest:
		branches = append(branches, r.GetName())
	case *v1pb.ListBranchesRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.CreateBranchRequest:
		projects = append(projects, r.GetParent())
	}

	var projectIDs []string
	for _, branch := range branches {
		projectID, _, err := common.GetProjectAndBranchID(branch)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse branch %q", branch)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, project := range projects {
		projectID, err := common.GetProjectID(project)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse project %q", project)
		}
		projectIDs = append(projectIDs, projectID)
	}
	return utils.Uniq(projectIDs), nil
}

func (*ContextProvider) getProjectIDsForProjectService(_ context.Context, req any) ([]string, error) {
	var projects, projectDeploymentConfigs, projectWebhooks, databaseGroups, protectionRules []string

	switch r := req.(type) {
	case *v1pb.GetProjectRequest:
		projects = append(projects, r.GetName())
	case *v1pb.UpdateProjectRequest:
		projects = append(projects, r.GetProject().GetName())
	case *v1pb.SetIamPolicyRequest:
		projects = append(projects, r.GetResource())
	case *v1pb.GetIamPolicyRequest:
		projects = append(projects, r.GetResource())
	case *v1pb.BatchGetIamPolicyRequest:
		projects = append(projects, r.GetNames()...)
	case *v1pb.GetDeploymentConfigRequest:
		projectDeploymentConfigs = append(projectDeploymentConfigs, r.GetName())
	case *v1pb.UpdateDeploymentConfigRequest:
		projectDeploymentConfigs = append(projectDeploymentConfigs, r.GetConfig().GetName())
	case *v1pb.AddWebhookRequest:
		projects = append(projects, r.GetProject())
	case *v1pb.UpdateWebhookRequest:
		projectWebhooks = append(projectWebhooks, r.GetWebhook().GetName())
	case *v1pb.RemoveWebhookRequest:
		projectWebhooks = append(projectWebhooks, r.GetWebhook().GetName())
	case *v1pb.TestWebhookRequest:
		projects = append(projects, r.GetProject())
	case *v1pb.ListDatabaseGroupsRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.GetDatabaseGroupRequest:
		databaseGroups = append(databaseGroups, r.GetName())
	case *v1pb.CreateDatabaseGroupRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.UpdateDatabaseGroupRequest:
		databaseGroups = append(databaseGroups, r.GetDatabaseGroup().GetName())
	case *v1pb.DeleteDatabaseGroupRequest:
		databaseGroups = append(databaseGroups, r.GetName())
	case *v1pb.GetProjectProtectionRulesRequest:
		protectionRules = append(protectionRules, r.GetName())
	case *v1pb.UpdateProjectProtectionRulesRequest:
		protectionRules = append(protectionRules, r.GetProtectionRules().GetName())
	}

	var projectIDs []string

	for _, project := range projects {
		projectID, err := common.GetProjectID(project)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %q", project)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, projectDeploymentConfig := range projectDeploymentConfigs {
		projectID, _, err := common.GetProjectIDDeploymentConfigID(projectDeploymentConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %q", projectDeploymentConfig)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, projectWebhook := range projectWebhooks {
		projectID, _, err := common.GetProjectIDWebhookID(projectWebhook)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %q", projectWebhook)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, databaseGroup := range databaseGroups {
		projectID, _, err := common.GetProjectIDDatabaseGroupID(databaseGroup)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %q", databaseGroup)
		}
		projectIDs = append(projectIDs, projectID)
	}
	for _, protectionRule := range protectionRules {
		projectID, err := common.TrimSuffixAndGetProjectID(protectionRule, common.ProtectionRulesSuffix)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %q", protectionRule)
		}
		projectIDs = append(projectIDs, projectID)
	}
	return utils.Uniq(projectIDs), nil
}

func (p *ContextProvider) getProjectIDsForDatabase(ctx context.Context, req any) ([]string, error) {
	var projectIDs []string

	var databaseNames []string
	switch r := req.(type) {
	case *v1pb.QueryRequest:
		if strings.Contains(r.GetName(), common.DatabaseIDPrefix) {
			databaseNames = append(databaseNames, r.GetName())
		} else if strings.HasPrefix(r.GetName(), common.InstanceNamePrefix) && r.GetConnectionDatabase() != "" {
			databaseNames = append(databaseNames, fmt.Sprintf("%s/%s%s", r.GetName(), common.DatabaseIDPrefix, r.GetConnectionDatabase()))
		}
	case *v1pb.ExportRequest:
		if strings.HasPrefix(r.GetName(), common.ProjectNamePrefix) {
			projectID, _, err := common.GetProjectIDIssueUID(r.GetName())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get projectID from %q", r.GetName())
			}
			projectIDs = append(projectIDs, projectID)
		} else if strings.Contains(r.GetName(), common.DatabaseIDPrefix) {
			databaseNames = append(databaseNames, r.GetName())
		} else if strings.HasPrefix(r.GetName(), common.InstanceNamePrefix) && r.GetConnectionDatabase() != "" {
			databaseNames = append(databaseNames, fmt.Sprintf("%s/%s%s", r.GetName(), common.DatabaseIDPrefix, r.GetConnectionDatabase()))
		}
	case *v1pb.GetDatabaseRequest:
		databaseNames = append(databaseNames, r.GetName())
	case *v1pb.SyncDatabaseRequest:
		databaseNames = append(databaseNames, r.GetName())
	case *v1pb.AdviseIndexRequest:
		databaseNames = append(databaseNames, r.GetParent())
	case *v1pb.GetDatabaseMetadataRequest:
		databaseName, err := common.TrimSuffix(r.GetName(), "/metadata")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetName())
		}
		databaseNames = append(databaseNames, databaseName)
	case *v1pb.UpdateDatabaseMetadataRequest:
		databaseName, err := common.TrimSuffix(r.GetDatabaseMetadata().GetName(), "/metadata")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetDatabaseMetadata().GetName())
		}
		databaseNames = append(databaseNames, databaseName)
	case *v1pb.UpdateDatabaseRequest:
		databaseNames = append(databaseNames, r.GetDatabase().GetName())
		if hasPath(r.GetUpdateMask(), "project") {
			projectID, err := common.GetProjectID(r.GetDatabase().GetProject())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get projectID from %q", r.GetDatabase().GetProject())
			}
			// allow to transfer databases to the default project.
			if projectID != api.DefaultProjectID {
				projectIDs = append(projectIDs, projectID)
			}
		}
	case *v1pb.BatchUpdateDatabasesRequest:
		for _, request := range r.Requests {
			databaseNames = append(databaseNames, request.GetDatabase().GetName())
			if hasPath(request.GetUpdateMask(), "project") {
				projectID, err := common.GetProjectID(request.GetDatabase().GetProject())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get projectID from %q", request.GetDatabase().GetProject())
				}
				// allow to transfer databases to the default project.
				if projectID != api.DefaultProjectID {
					projectIDs = append(projectIDs, projectID)
				}
			}
		}
	case *v1pb.GetDatabaseSchemaRequest:
		databaseName, err := common.TrimSuffix(r.GetName(), "/schema")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetName())
		}
		databaseNames = append(databaseNames, databaseName)
	case *v1pb.ListSecretsRequest:
		databaseNames = append(databaseNames, r.GetParent())
	case *v1pb.UpdateSecretRequest:
		instance, database, _, err := common.GetInstanceDatabaseIDSecretName(r.GetSecret().GetName())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetSecret().GetName())
		}
		databaseNames = append(databaseNames, common.FormatDatabase(instance, database))
	case *v1pb.DeleteSecretRequest:
		instance, database, _, err := common.GetInstanceDatabaseIDSecretName(r.GetName())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetName())
		}
		databaseNames = append(databaseNames, common.FormatDatabase(instance, database))
	case *v1pb.ListChangeHistoriesRequest:
		databaseNames = append(databaseNames, r.GetParent())
	case *v1pb.GetChangeHistoryRequest:
		instance, database, _, err := common.GetInstanceDatabaseIDChangeHistory(r.GetName())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetName())
		}
		databaseNames = append(databaseNames, common.FormatDatabase(instance, database))
	case *v1pb.DiffSchemaRequest:
		if instance, database, _, err := common.GetInstanceDatabaseIDChangeHistory(r.GetName()); err == nil {
			databaseNames = append(databaseNames, common.FormatDatabase(instance, database))
		} else {
			databaseNames = append(databaseNames, r.GetName())
		}
		if history := r.GetChangeHistory(); history != "" {
			instance, database, _, err := common.GetInstanceDatabaseIDChangeHistory(history)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get databaseName from %q", history)
			}
			databaseNames = append(databaseNames, common.FormatDatabase(instance, database))
		}
	}

	for _, databaseName := range databaseNames {
		database, err := getDatabaseMessage(ctx, p.s, databaseName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database %q", databaseName)
		}
		projectIDs = append(projectIDs, database.ProjectID)
	}

	return utils.Uniq(projectIDs), nil
}

func getDatabaseMessage(ctx context.Context, s *store.Store, databaseResourceName string) (*store.DatabaseMessage, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(databaseResourceName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %q", databaseResourceName)
	}

	instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
	}
	if instance == nil {
		return nil, errors.Errorf("instance not found")
	}

	find := &store.FindDatabaseMessage{
		InstanceID:          &instanceID,
		DatabaseName:        &databaseName,
		IgnoreCaseSensitive: store.IgnoreDatabaseAndTableCaseSensitive(instance),
		ShowDeleted:         true,
	}
	database, err := s.GetDatabaseV2(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Errorf("database %q not found", databaseResourceName)
	}
	return database, nil
}
