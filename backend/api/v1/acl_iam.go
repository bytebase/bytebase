package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

func (in *ACLInterceptor) checkIAMPermission(ctx context.Context, fullMethod string, req any, user *store.UserMessage) error {
	if isSkippedMethod(fullMethod) {
		return nil
	}

	p, ok := methodPermissionMap[fullMethod]
	if !ok {
		return errors.Errorf("method %q not found in method-permission map", fullMethod)
	}
	var projectIDsGetter func(context.Context, any) ([]string, error)

	switch fullMethod {
	// below are "workspace-level" permissions.
	// we don't have to go down to the project level.
	case
		v1pb.ProjectService_ListProjects_FullMethodName,
		v1pb.ProjectService_CreateProject_FullMethodName,
		v1pb.ProjectService_DeleteProject_FullMethodName,
		v1pb.ProjectService_UndeleteProject_FullMethodName,
		v1pb.InstanceService_ListInstances_FullMethodName,
		v1pb.InstanceService_GetInstance_FullMethodName,
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
		v1pb.InstanceRoleService_GetInstanceRole_FullMethodName,
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
		v1pb.EnvironmentService_UpdateBackupSetting_FullMethodName,
		v1pb.SettingService_ListSettings_FullMethodName,
		v1pb.SettingService_GetSetting_FullMethodName,
		v1pb.SettingService_SetSetting_FullMethodName,
		v1pb.OrgPolicyService_ListPolicies_FullMethodName,
		v1pb.OrgPolicyService_GetPolicy_FullMethodName,
		v1pb.OrgPolicyService_CreatePolicy_FullMethodName,
		v1pb.OrgPolicyService_UpdatePolicy_FullMethodName,
		v1pb.OrgPolicyService_DeletePolicy_FullMethodName,
		v1pb.IdentityProviderService_GetIdentityProvider_FullMethodName,
		v1pb.IdentityProviderService_CreateIdentityProvider_FullMethodName,
		v1pb.IdentityProviderService_UpdateIdentityProvider_FullMethodName,
		v1pb.IdentityProviderService_DeleteIdentityProvider_FullMethodName,
		v1pb.IdentityProviderService_UndeleteIdentityProvider_FullMethodName,
		v1pb.IdentityProviderService_TestIdentityProvider_FullMethodName,
		v1pb.ExternalVersionControlService_ListExternalVersionControls_FullMethodName,
		v1pb.ExternalVersionControlService_GetExternalVersionControl_FullMethodName,
		v1pb.ExternalVersionControlService_CreateExternalVersionControl_FullMethodName,
		v1pb.ExternalVersionControlService_ExchangeToken_FullMethodName,
		v1pb.ExternalVersionControlService_UpdateExternalVersionControl_FullMethodName,
		v1pb.ExternalVersionControlService_DeleteExternalVersionControl_FullMethodName,
		v1pb.ExternalVersionControlService_SearchExternalVersionControlProjects_FullMethodName,
		v1pb.ExternalVersionControlService_ListProjectGitOpsInfo_FullMethodName,
		v1pb.RiskService_ListRisks_FullMethodName,
		v1pb.RiskService_CreateRisk_FullMethodName,
		v1pb.RiskService_UpdateRisk_FullMethodName,
		v1pb.RiskService_DeleteRisk_FullMethodName,
		v1pb.RoleService_ListRoles_FullMethodName,
		v1pb.RoleService_CreateRole_FullMethodName,
		v1pb.RoleService_UpdateRole_FullMethodName,
		v1pb.RoleService_DeleteRole_FullMethodName:

		projectIDsGetter = func(context.Context, any) ([]string, error) {
			return nil, nil
		}
	case
		v1pb.DatabaseService_GetDatabase_FullMethodName,
		v1pb.DatabaseService_UpdateDatabase_FullMethodName,
		v1pb.DatabaseService_BatchUpdateDatabases_FullMethodName,
		v1pb.DatabaseService_SyncDatabase_FullMethodName,
		v1pb.DatabaseService_GetDatabaseMetadata_FullMethodName,
		v1pb.DatabaseService_UpdateDatabaseMetadata_FullMethodName,
		v1pb.DatabaseService_GetDatabaseSchema_FullMethodName,
		v1pb.DatabaseService_GetBackupSetting_FullMethodName,
		v1pb.DatabaseService_UpdateBackupSetting_FullMethodName,
		v1pb.DatabaseService_CreateBackup_FullMethodName,
		v1pb.DatabaseService_ListBackups_FullMethodName,
		v1pb.DatabaseService_ListSecrets_FullMethodName,
		v1pb.DatabaseService_UpdateSecret_FullMethodName,
		v1pb.DatabaseService_DeleteSecret_FullMethodName,
		v1pb.DatabaseService_AdviseIndex_FullMethodName, // TODO(p0ny): implement.
		v1pb.DatabaseService_ListChangeHistories_FullMethodName,
		v1pb.DatabaseService_GetChangeHistory_FullMethodName:

		projectIDsGetter = in.getProjectIDsForDatabaseService
	case
		v1pb.IssueService_GetIssue_FullMethodName,
		v1pb.IssueService_CreateIssue_FullMethodName,
		v1pb.IssueService_UpdateIssue_FullMethodName,
		v1pb.IssueService_CreateIssueComment_FullMethodName,
		v1pb.IssueService_UpdateIssueComment_FullMethodName,
		v1pb.IssueService_BatchUpdateIssuesStatus_FullMethodName:

		projectIDsGetter = in.getProjectIDsForIssueService
	case
		v1pb.ChangelistService_CreateChangelist_FullMethodName,
		v1pb.ChangelistService_UpdateChangelist_FullMethodName,
		v1pb.ChangelistService_GetChangelist_FullMethodName,
		v1pb.ChangelistService_DeleteChangelist_FullMethodName:

		projectIDsGetter = in.getProjectIDsForChangelistService
	case
		v1pb.BranchService_ListBranches_FullMethodName,
		v1pb.BranchService_GetBranch_FullMethodName,
		v1pb.BranchService_CreateBranch_FullMethodName,
		v1pb.BranchService_UpdateBranch_FullMethodName,
		v1pb.BranchService_DeleteBranch_FullMethodName,
		v1pb.BranchService_MergeBranch_FullMethodName,
		v1pb.BranchService_RebaseBranch_FullMethodName:

		projectIDsGetter = in.getProjectIDsForBranchService
	case
		v1pb.RolloutService_GetRollout_FullMethodName,
		v1pb.RolloutService_CreateRollout_FullMethodName,
		v1pb.RolloutService_PreviewRollout_FullMethodName,
		v1pb.RolloutService_GetPlan_FullMethodName,
		v1pb.RolloutService_CreatePlan_FullMethodName,
		v1pb.RolloutService_UpdatePlan_FullMethodName,
		v1pb.RolloutService_ListTaskRuns_FullMethodName,
		v1pb.RolloutService_ListPlanCheckRuns_FullMethodName,
		v1pb.RolloutService_RunPlanChecks_FullMethodName,
		v1pb.RolloutService_BatchRunTasks_FullMethodName,
		v1pb.RolloutService_BatchSkipTasks_FullMethodName,
		v1pb.RolloutService_BatchCancelTaskRuns_FullMethodName:

		projectIDsGetter = in.getProjectIDsForRolloutService
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
		v1pb.ProjectService_UpdateProjectGitOpsInfo_FullMethodName,
		v1pb.ProjectService_UnsetProjectGitOpsInfo_FullMethodName,
		v1pb.ProjectService_GetProjectGitOpsInfo_FullMethodName,
		v1pb.ProjectService_SetupProjectSQLReviewCI_FullMethodName,

		v1pb.ProjectService_GetDatabaseGroup_FullMethodName,
		v1pb.ProjectService_CreateDatabaseGroup_FullMethodName,
		v1pb.ProjectService_UpdateDatabaseGroup_FullMethodName,
		v1pb.ProjectService_DeleteDatabaseGroup_FullMethodName,

		v1pb.ProjectService_GetSchemaGroup_FullMethodName,
		v1pb.ProjectService_CreateSchemaGroup_FullMethodName,
		v1pb.ProjectService_UpdateSchemaGroup_FullMethodName,
		v1pb.ProjectService_DeleteSchemaGroup_FullMethodName,
		v1pb.ProjectService_GetProjectProtectionRules_FullMethodName,
		v1pb.ProjectService_UpdateProjectProtectionRules_FullMethodName:

		projectIDsGetter = in.getProjectIDsForProjectService
	}

	projectIDs, err := projectIDsGetter(ctx, req)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to check permission, err %v", err)
	}
	ok, err = in.iamManager.CheckPermission(ctx, p, user, projectIDs...)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to check permission for method %q, err: %v", fullMethod, err)
	}
	if !ok {
		return status.Errorf(codes.PermissionDenied, "permission denied for method %q, user does not have permission %q", fullMethod, p)
	}

	return nil
}

func isSkippedMethod(fullMethod string) bool {
	if auth.IsAuthenticationAllowed(fullMethod) {
		return true
	}

	// Below are the skipped.
	switch fullMethod {
	// skip methods that are not considered to be resource-related.
	case
		v1pb.ActuatorService_GetActuatorInfo_FullMethodName,
		v1pb.ActuatorService_UpdateActuatorInfo_FullMethodName,
		v1pb.ActuatorService_DeleteCache_FullMethodName,
		v1pb.ActuatorService_ListDebugLog_FullMethodName,
		v1pb.AnomalyService_SearchAnomalies_FullMethodName,
		v1pb.AuthService_GetUser_FullMethodName,
		v1pb.AuthService_ListUsers_FullMethodName,
		v1pb.AuthService_CreateUser_FullMethodName,
		v1pb.AuthService_UpdateUser_FullMethodName,
		v1pb.AuthService_DeleteUser_FullMethodName,
		v1pb.AuthService_UndeleteUser_FullMethodName,
		v1pb.AuthService_Login_FullMethodName,
		v1pb.AuthService_Logout_FullMethodName,
		v1pb.CelService_BatchParse_FullMethodName,
		v1pb.CelService_BatchDeparse_FullMethodName,
		v1pb.InboxService_GetInboxSummary_FullMethodName,
		v1pb.InboxService_ListInbox_FullMethodName,
		v1pb.InboxService_UpdateInbox_FullMethodName,
		v1pb.LoggingService_ListLogs_FullMethodName,
		v1pb.LoggingService_GetLog_FullMethodName,
		v1pb.LoggingService_ExportLogs_FullMethodName,
		v1pb.SQLService_Query_FullMethodName,
		v1pb.SQLService_Export_FullMethodName,
		v1pb.SQLService_AdminExecute_FullMethodName,
		v1pb.SQLService_DifferPreview_FullMethodName,
		v1pb.SQLService_Check_FullMethodName,
		v1pb.SQLService_Pretty_FullMethodName,
		v1pb.SQLService_StringifyMetadata_FullMethodName,
		v1pb.SubscriptionService_GetSubscription_FullMethodName,
		v1pb.SubscriptionService_GetFeatureMatrix_FullMethodName,
		v1pb.SubscriptionService_UpdateSubscription_FullMethodName:
		return true
	// skip checking for sheet service because we want to
	// discriminate bytebase artifact sheets and user sheets first.
	// TODO(p0ny): implement
	case
		v1pb.SheetService_CreateSheet_FullMethodName,
		v1pb.SheetService_GetSheet_FullMethodName,
		v1pb.SheetService_SearchSheets_FullMethodName,
		v1pb.SheetService_UpdateSheet_FullMethodName,
		v1pb.SheetService_UpdateSheetOrganizer_FullMethodName,
		v1pb.SheetService_DeleteSheet_FullMethodName:
		return true
	// skip checking for custom approval.
	case
		v1pb.IssueService_ApproveIssue_FullMethodName,
		v1pb.IssueService_RejectIssue_FullMethodName,
		v1pb.IssueService_RequestIssue_FullMethodName:
		return true
	// handled in the method because checking is complex.
	case
		v1pb.DatabaseService_ListSlowQueries_FullMethodName,
		v1pb.DatabaseService_ListDatabases_FullMethodName,     // TODO(p0ny): implement
		v1pb.DatabaseService_DiffSchema_FullMethodName,        // TODO(p0ny): implement
		v1pb.IssueService_ListIssues_FullMethodName,           // TODO(p0ny): implement
		v1pb.ProjectService_ListDatabaseGroups_FullMethodName, // TODO(p0ny): implement
		v1pb.ChangelistService_ListChangelists_FullMethodName, // TODO(p0ny): implement
		v1pb.RolloutService_ListPlans_FullMethodName,          // TODO(p0ny): implement
		v1pb.ProjectService_ListSchemaGroups_FullMethodName:   // TODO(p0ny): implement
		return true
	}
	return false
}

func getDatabaseMessage(ctx context.Context, s *store.Store, databaseResourceName string) (*store.DatabaseMessage, error) {
	instanceID, databaseName, err := common.GetInstanceDatabaseID(databaseResourceName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %q", databaseResourceName)
	}
	find := &store.FindDatabaseMessage{
		ShowDeleted: true,
	}
	databaseUID, isNumber := isNumber(databaseName)
	if isNumber {
		// Expected format: "instances/{ignored_value}/database/{uid}"
		find.UID = &databaseUID
	} else {
		// Expected format: "instances/{instance}/database/{database}"
		find.InstanceID = &instanceID
		find.DatabaseName = &databaseName
		instance, err := s.GetInstanceV2(ctx, &store.FindInstanceMessage{ResourceID: &instanceID})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get instance %s", instanceID)
		}
		if instance == nil {
			return nil, errors.Wrapf(err, "instance not found")
		}
		find.IgnoreCaseSensitive = store.IgnoreDatabaseAndTableCaseSensitive(instance)
	}
	database, err := s.GetDatabaseV2(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database")
	}
	if database == nil {
		return nil, errors.Wrapf(err, "database %q not found", databaseResourceName)
	}
	return database, nil
}

func (*ACLInterceptor) getProjectIDsForChangelistService(_ context.Context, req any) ([]string, error) {
	var projects, changelists []string
	switch r := req.(type) {
	case *v1pb.CreateChangelistRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.UpdateChangelistRequest:
		changelists = append(changelists, r.GetChangelist().GetName())
	case *v1pb.GetChangelistRequest:
		changelists = append(changelists, r.GetName())
	case *v1pb.DeleteChangelistRequest:
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

	return uniq(projectIDs), nil
}

func (*ACLInterceptor) getProjectIDsForRolloutService(_ context.Context, req any) ([]string, error) {
	var projects, rollouts, plans, tasks, stages []string
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
	case *v1pb.UpdatePlanRequest:
		plans = append(plans, r.GetPlan().GetName())
	case *v1pb.ListTaskRunsRequest:
		tasks = append(tasks, r.GetParent())
	case *v1pb.ListPlanCheckRunsRequest:
		plans = append(plans, r.GetParent())
	case *v1pb.RunPlanChecksRequest:
		plans = append(plans, r.GetName())
	case *v1pb.BatchRunTasksRequest:
		stages = append(stages, r.GetParent())
	case *v1pb.BatchSkipTasksRequest:
		stages = append(stages, r.GetParent())
	case *v1pb.BatchCancelTaskRunsRequest:
		tasks = append(tasks, r.GetParent())
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
	for _, stage := range stages {
		projectID, _, _, err := common.GetProjectIDRolloutIDMaybeStageID(stage)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse stage %q", stage)
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

	return uniq(projectIDs), nil
}

func (*ACLInterceptor) getProjectIDsForBranchService(_ context.Context, req any) ([]string, error) {
	var projects, branches []string
	switch r := req.(type) {
	case *v1pb.GetBranchRequest:
		branches = append(branches, r.GetName())
	case *v1pb.ListBranchesRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.CreateBranchRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.UpdateBranchRequest:
		branches = append(branches, r.GetBranch().GetName())
	case *v1pb.DeleteBranchRequest:
		branches = append(branches, r.GetName())
	case *v1pb.MergeBranchRequest:
		branches = append(branches, r.GetName())
	case *v1pb.RebaseBranchRequest:
		branches = append(branches, r.GetName())
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
	return uniq(projectIDs), nil
}

func (*ACLInterceptor) getProjectIDsForProjectService(_ context.Context, req any) ([]string, error) {
	var projects, projectDeploymentConfigs, projectWebhooks, projectGitopsInfos, databaseGroups, schemaGroups, protectionRules []string

	switch r := req.(type) {
	case *v1pb.GetProjectRequest:
		projects = append(projects, r.GetName())
	case *v1pb.UpdateProjectRequest:
		projects = append(projects, r.GetProject().GetName())
	case *v1pb.SetIamPolicyRequest:
		projects = append(projects, r.GetProject())
	case *v1pb.GetIamPolicyRequest:
		projects = append(projects, r.GetProject())
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
	case *v1pb.UpdateProjectGitOpsInfoRequest:
		projectGitopsInfos = append(projectGitopsInfos, r.GetProjectGitopsInfo().GetName())
	case *v1pb.UnsetProjectGitOpsInfoRequest:
		projectGitopsInfos = append(projectGitopsInfos, r.GetName())
	case *v1pb.SetupSQLReviewCIRequest:
		projectGitopsInfos = append(projectGitopsInfos, r.GetName())
	case *v1pb.GetProjectGitOpsInfoRequest:
		projectGitopsInfos = append(projectGitopsInfos, r.GetName())
	case *v1pb.GetDatabaseGroupRequest:
		databaseGroups = append(databaseGroups, r.GetName())
	case *v1pb.CreateDatabaseGroupRequest:
		projects = append(projects, r.GetParent())
	case *v1pb.UpdateDatabaseGroupRequest:
		databaseGroups = append(databaseGroups, r.GetDatabaseGroup().GetName())
	case *v1pb.DeleteDatabaseGroupRequest:
		databaseGroups = append(databaseGroups, r.GetName())
	case *v1pb.GetSchemaGroupRequest:
		schemaGroups = append(schemaGroups, r.GetName())
	case *v1pb.CreateSchemaGroupRequest:
		databaseGroups = append(databaseGroups, r.GetParent())
	case *v1pb.UpdateSchemaGroupRequest:
		schemaGroups = append(schemaGroups, r.GetSchemaGroup().GetName())
	case *v1pb.DeleteSchemaGroupRequest:
		schemaGroups = append(schemaGroups, r.GetName())
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
	for _, projectGitopsInfo := range projectGitopsInfos {
		projectID, err := common.TrimSuffixAndGetProjectID(projectGitopsInfo, common.GitOpsInfoSuffix)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %q", projectGitopsInfo)
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
	for _, schemaGroup := range schemaGroups {
		projectID, _, _, err := common.GetProjectIDDatabaseGroupIDSchemaGroupID(schemaGroup)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse %q", schemaGroup)
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
	return uniq(projectIDs), nil
}

func (*ACLInterceptor) getProjectIDsForIssueService(_ context.Context, req any) ([]string, error) {
	var issueNames []string

	switch r := req.(type) {
	case *v1pb.GetIssueRequest:
		issueNames = append(issueNames, r.GetName())
	case *v1pb.CreateIssueRequest:
		issueNames = append(issueNames, r.GetIssue().GetName())
	case *v1pb.UpdateIssueRequest:
		issueNames = append(issueNames, r.GetIssue().GetName())
	case *v1pb.CreateIssueCommentRequest:
		issueNames = append(issueNames, r.GetParent())
	case *v1pb.UpdateIssueCommentRequest:
		issueNames = append(issueNames, r.GetParent())
	case *v1pb.BatchUpdateIssuesStatusRequest:
		issueNames = append(issueNames, r.GetIssues()...)
	}

	var projectIDs []string
	for _, issueName := range issueNames {
		projectID, _, err := common.GetProjectIDIssueID(issueName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get projectID from %q", issueName)
		}
		projectIDs = append(projectIDs, projectID)
	}

	return uniq(projectIDs), nil
}

func (in *ACLInterceptor) getProjectIDsForDatabaseService(ctx context.Context, req any) ([]string, error) {
	var projectIDs []string

	var databaseNames []string
	switch r := req.(type) {
	case *v1pb.GetDatabaseRequest:
		databaseNames = append(databaseNames, r.GetName())
	case *v1pb.SyncDatabaseRequest:
		databaseNames = append(databaseNames, r.GetName())
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
			projectIDs = append(projectIDs, projectID)
		}
	case *v1pb.BatchUpdateDatabasesRequest:
		for _, request := range r.Requests {
			databaseNames = append(databaseNames, request.GetDatabase().GetName())
			if hasPath(request.GetUpdateMask(), "project") {
				projectID, err := common.GetProjectID(request.GetDatabase().GetProject())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get projectID from %q", request.GetDatabase().GetProject())
				}
				projectIDs = append(projectIDs, projectID)
			}
		}
	case *v1pb.GetDatabaseSchemaRequest:
		databaseName, err := common.TrimSuffix(r.GetName(), "/schema")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetName())
		}
		databaseNames = append(databaseNames, databaseName)
	case *v1pb.GetBackupSettingRequest:
		databaseName, err := common.TrimSuffix(r.GetName(), "/backupSetting")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetName())
		}
		databaseNames = append(databaseNames, databaseName)
	case *v1pb.UpdateBackupSettingRequest:
		databaseName, err := common.TrimSuffix(r.GetSetting().GetName(), "/backupSetting")
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get databaseName from %q", r.GetSetting().GetName())
		}
		databaseNames = append(databaseNames, databaseName)
	case *v1pb.CreateBackupRequest:
		databaseNames = append(databaseNames, r.GetParent())
	case *v1pb.ListBackupsRequest:
		databaseNames = append(databaseNames, r.GetParent())
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
	}

	for _, databaseName := range databaseNames {
		database, err := getDatabaseMessage(ctx, in.store, databaseName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get database %q", databaseName)
		}
		projectIDs = append(projectIDs, database.ProjectID)
	}

	return uniq(projectIDs), nil
}

func uniq[T comparable](array []T) []T {
	res := make([]T, 0, len(array))
	seen := make(map[T]struct{}, len(array))

	for _, e := range array {
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		res = append(res, e)
	}

	return res
}
