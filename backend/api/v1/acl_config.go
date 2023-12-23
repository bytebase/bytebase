package v1

import (
	"github.com/bytebase/bytebase/backend/component/iam"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
)

var ownerAndDBAMethods = map[string]bool{
	v1pb.EnvironmentService_CreateEnvironment_FullMethodName:   true,
	v1pb.EnvironmentService_UpdateEnvironment_FullMethodName:   true,
	v1pb.EnvironmentService_DeleteEnvironment_FullMethodName:   true,
	v1pb.EnvironmentService_UndeleteEnvironment_FullMethodName: true,
	v1pb.EnvironmentService_UpdateBackupSetting_FullMethodName: true,
	v1pb.InstanceService_CreateInstance_FullMethodName:         true,
	v1pb.InstanceService_UpdateInstance_FullMethodName:         true,
	v1pb.InstanceService_DeleteInstance_FullMethodName:         true,
	v1pb.InstanceService_UndeleteInstance_FullMethodName:       true,
	v1pb.InstanceService_AddDataSource_FullMethodName:          true,
	v1pb.InstanceService_RemoveDataSource_FullMethodName:       true,
	v1pb.InstanceService_UpdateDataSource_FullMethodName:       true,
	v1pb.RiskService_CreateRisk_FullMethodName:                 true,
	v1pb.RiskService_UpdateRisk_FullMethodName:                 true,
	v1pb.RiskService_DeleteRisk_FullMethodName:                 true,
	v1pb.SettingService_SetSetting_FullMethodName:              true,
	v1pb.RoleService_CreateRole_FullMethodName:                 true,
	v1pb.RoleService_UpdateRole_FullMethodName:                 true,
	v1pb.RoleService_DeleteRole_FullMethodName:                 true,
	v1pb.ActuatorService_UpdateActuatorInfo_FullMethodName:     true,
	v1pb.ActuatorService_ListDebugLog_FullMethodName:           true,
}

var projectOwnerMethods = map[string]bool{
	v1pb.ProjectService_UpdateProject_FullMethodName:           true,
	v1pb.ProjectService_DeleteProject_FullMethodName:           true,
	v1pb.ProjectService_UndeleteProject_FullMethodName:         true,
	v1pb.ProjectService_SetIamPolicy_FullMethodName:            true,
	v1pb.SubscriptionService_UpdateSubscription_FullMethodName: true,
}

var transferDatabaseMethods = map[string]bool{
	v1pb.DatabaseService_UpdateDatabase_FullMethodName:       true,
	v1pb.DatabaseService_BatchUpdateDatabases_FullMethodName: true,
}

var methodPermissionMap = map[string]iam.Permission{
	v1pb.InstanceService_ListInstances_FullMethodName:     iam.PermissionInstancesList,
	v1pb.InstanceService_GetInstance_FullMethodName:       iam.PermissionInstancesGet,
	v1pb.InstanceService_CreateInstance_FullMethodName:    iam.PermissionInstancesCreate,
	v1pb.InstanceService_UpdateInstance_FullMethodName:    iam.PermissionInstancesUpdate,
	v1pb.InstanceService_DeleteInstance_FullMethodName:    iam.PermissionInstancesDelete,
	v1pb.InstanceService_UndeleteInstance_FullMethodName:  iam.PermissionInstancesUndelete,
	v1pb.InstanceService_SyncInstance_FullMethodName:      iam.PermissionInstancesSync,
	v1pb.InstanceService_BatchSyncInstance_FullMethodName: iam.PermissionInstancesSync,
	v1pb.InstanceService_AddDataSource_FullMethodName:     iam.PermissionInstancesUpdate,
	v1pb.InstanceService_RemoveDataSource_FullMethodName:  iam.PermissionInstancesUpdate,
	v1pb.InstanceService_UpdateDataSource_FullMethodName:  iam.PermissionInstancesUpdate,
	v1pb.InstanceService_SyncSlowQueries_FullMethodName:   iam.PermissionInstancesSync,

	v1pb.DatabaseService_GetDatabase_FullMethodName:            iam.PermissionDatabasesGet,
	v1pb.DatabaseService_ListDatabases_FullMethodName:          iam.PermissionDatabasesList,
	v1pb.DatabaseService_UpdateDatabase_FullMethodName:         iam.PermissionDatabasesUpdate,
	v1pb.DatabaseService_BatchUpdateDatabases_FullMethodName:   iam.PermissionDatabasesUpdate,
	v1pb.DatabaseService_SyncDatabase_FullMethodName:           iam.PermissionDatabasesSync,
	v1pb.DatabaseService_GetDatabaseMetadata_FullMethodName:    iam.PermissionDatabasesGetMetadata,
	v1pb.DatabaseService_UpdateDatabaseMetadata_FullMethodName: iam.PermissionDatabasesUpdateMetadata,
	v1pb.DatabaseService_GetDatabaseSchema_FullMethodName:      iam.PermissionDatabasesGetSchema,
	v1pb.DatabaseService_DiffSchema_FullMethodName:             "", // handled in the method.
	v1pb.DatabaseService_GetBackupSetting_FullMethodName:       iam.PermissionDatabasesGetBackupSetting,
	v1pb.DatabaseService_UpdateBackupSetting_FullMethodName:    iam.PermissionDatabasesUpdateBackupSetting,
	v1pb.DatabaseService_CreateBackup_FullMethodName:           iam.PermissionBackupsCreate,
	v1pb.DatabaseService_ListBackups_FullMethodName:            iam.PermissionBackupsList,
	v1pb.DatabaseService_ListSlowQueries_FullMethodName:        iam.PermissionSlowQueriesList,
	v1pb.DatabaseService_ListSecrets_FullMethodName:            iam.PermissionDatabaseSecretsList,
	v1pb.DatabaseService_UpdateSecret_FullMethodName:           iam.PermissionDatabaseSecretsUpdate,
	v1pb.DatabaseService_DeleteSecret_FullMethodName:           iam.PermissionDatabaseSecretsDelete,
	v1pb.DatabaseService_AdviseIndex_FullMethodName:            iam.PermissionDatabasesAdviseIndex,
	v1pb.DatabaseService_ListChangeHistories_FullMethodName:    iam.PermissionChangeHistoriesList,
	v1pb.DatabaseService_GetChangeHistory_FullMethodName:       iam.PermissionChangeHistoriesGet,
	v1pb.EnvironmentService_CreateEnvironment_FullMethodName:   iam.PermissionEnvironmentsCreate,
	v1pb.EnvironmentService_UpdateEnvironment_FullMethodName:   iam.PermissionEnvironmentsUpdate,
	v1pb.EnvironmentService_DeleteEnvironment_FullMethodName:   iam.PermissionEnvironmentsDelete,
	v1pb.EnvironmentService_UndeleteEnvironment_FullMethodName: iam.PermissionEnvironmentsUndelete,
	v1pb.EnvironmentService_GetEnvironment_FullMethodName:      iam.PermissionEnvironmentsGet,
	v1pb.EnvironmentService_ListEnvironments_FullMethodName:    iam.PermissionEnvironmentsList,
	v1pb.EnvironmentService_UpdateBackupSetting_FullMethodName: iam.PermissionEnvironmentsUpdate,
	v1pb.IssueService_ListIssues_FullMethodName:                iam.PermissionIssuesList,
	v1pb.IssueService_GetIssue_FullMethodName:                  iam.PermissionIssuesGet,
	v1pb.IssueService_UpdateIssue_FullMethodName:               iam.PermissionIssuesUpdate,
	v1pb.IssueService_BatchUpdateIssuesStatus_FullMethodName:   iam.PermissionIssuesUpdate,
	v1pb.IssueService_CreateIssueComment_FullMethodName:        iam.PermissionIssueCommentsCreate,
	v1pb.IssueService_UpdateIssueComment_FullMethodName:        iam.PermissionIssueCommentsUpdate,
	v1pb.IssueService_ApproveIssue_FullMethodName:              "",
	v1pb.IssueService_RejectIssue_FullMethodName:               "",
	v1pb.IssueService_RequestIssue_FullMethodName:              "",

	v1pb.ProjectService_ListProjects_FullMethodName:                 iam.PermissionProjectsList,
	v1pb.ProjectService_GetProject_FullMethodName:                   iam.PermissionProjectsGet,
	v1pb.ProjectService_CreateProject_FullMethodName:                iam.PermissionProjectsCreate,
	v1pb.ProjectService_UpdateProject_FullMethodName:                iam.PermissionProjectsUpdate,
	v1pb.ProjectService_DeleteProject_FullMethodName:                iam.PermissionProjectsDelete,
	v1pb.ProjectService_UndeleteProject_FullMethodName:              iam.PermissionProjectsUndelete,
	v1pb.ProjectService_SetIamPolicy_FullMethodName:                 iam.PermissionProjectsSetIAMPolicy,
	v1pb.ProjectService_GetIamPolicy_FullMethodName:                 iam.PermissionProjectsGetIAMPolicy,
	v1pb.ProjectService_BatchGetIamPolicy_FullMethodName:            iam.PermissionProjectsGetIAMPolicy,
	v1pb.ProjectService_GetDeploymentConfig_FullMethodName:          iam.PermissionProjectsGet,
	v1pb.ProjectService_UpdateDeploymentConfig_FullMethodName:       iam.PermissionProjectsUpdate,
	v1pb.ProjectService_AddWebhook_FullMethodName:                   iam.PermissionProjectsUpdate,
	v1pb.ProjectService_UpdateWebhook_FullMethodName:                iam.PermissionProjectsUpdate,
	v1pb.ProjectService_RemoveWebhook_FullMethodName:                iam.PermissionProjectsUpdate,
	v1pb.ProjectService_TestWebhook_FullMethodName:                  iam.PermissionProjectsUpdate,
	v1pb.ProjectService_UpdateProjectGitOpsInfo_FullMethodName:      iam.PermissionProjectsUpdate,
	v1pb.ProjectService_UnsetProjectGitOpsInfo_FullMethodName:       iam.PermissionProjectsUpdate,
	v1pb.ProjectService_SetupProjectSQLReviewCI_FullMethodName:      iam.PermissionProjectsUpdate,
	v1pb.ProjectService_GetProjectGitOpsInfo_FullMethodName:         iam.PermissionProjectsGet,
	v1pb.ProjectService_ListDatabaseGroups_FullMethodName:           iam.PermissionProjectsGet,
	v1pb.ProjectService_GetDatabaseGroup_FullMethodName:             iam.PermissionProjectsGet,
	v1pb.ProjectService_CreateDatabaseGroup_FullMethodName:          iam.PermissionProjectsUpdate,
	v1pb.ProjectService_UpdateDatabaseGroup_FullMethodName:          iam.PermissionProjectsUpdate,
	v1pb.ProjectService_DeleteDatabaseGroup_FullMethodName:          iam.PermissionProjectsUpdate,
	v1pb.ProjectService_ListSchemaGroups_FullMethodName:             iam.PermissionProjectsGet,
	v1pb.ProjectService_GetSchemaGroup_FullMethodName:               iam.PermissionProjectsGet,
	v1pb.ProjectService_CreateSchemaGroup_FullMethodName:            iam.PermissionProjectsUpdate,
	v1pb.ProjectService_UpdateSchemaGroup_FullMethodName:            iam.PermissionProjectsUpdate,
	v1pb.ProjectService_DeleteSchemaGroup_FullMethodName:            iam.PermissionProjectsUpdate,
	v1pb.ProjectService_GetProjectProtectionRules_FullMethodName:    iam.PermissionProjectsGet,
	v1pb.ProjectService_UpdateProjectProtectionRules_FullMethodName: iam.PermissionProjectsUpdate,

	v1pb.RiskService_ListRisks_FullMethodName:  iam.PermissionRisksList,
	v1pb.RiskService_CreateRisk_FullMethodName: iam.PermissionRisksCreate,
	v1pb.RiskService_UpdateRisk_FullMethodName: iam.PermissionRisksUpdate,
	v1pb.RiskService_DeleteRisk_FullMethodName: iam.PermissionRisksDelete,

	v1pb.RoleService_ListRoles_FullMethodName:  iam.PermissionRolesList,
	v1pb.RoleService_CreateRole_FullMethodName: iam.PermissionRolesCreate,
	v1pb.RoleService_UpdateRole_FullMethodName: iam.PermissionRolesUpdate,
	v1pb.RoleService_DeleteRole_FullMethodName: iam.PermissionRolesDelete,

	v1pb.ChangelistService_ListChangelists_FullMethodName:  iam.PermissionChangelistsList,
	v1pb.ChangelistService_GetChangelist_FullMethodName:    iam.PermissionChangelistsGet,
	v1pb.ChangelistService_CreateChangelist_FullMethodName: iam.PermissionChangelistsCreate,
	v1pb.ChangelistService_UpdateChangelist_FullMethodName: iam.PermissionChangelistsUpdate,
	v1pb.ChangelistService_DeleteChangelist_FullMethodName: iam.PermissionChangelistsDelete,

	v1pb.InstanceRoleService_ListInstanceRoles_FullMethodName:    iam.PermissionInstanceRolesList,
	v1pb.InstanceRoleService_GetInstanceRole_FullMethodName:      iam.PermissionInstanceRolesGet,
	v1pb.InstanceRoleService_CreateInstanceRole_FullMethodName:   iam.PermissionInstanceRolesCreate,
	v1pb.InstanceRoleService_UpdateInstanceRole_FullMethodName:   iam.PermissionInstanceRolesUpdate,
	v1pb.InstanceRoleService_DeleteInstanceRole_FullMethodName:   iam.PermissionInstanceRolesDelete,
	v1pb.InstanceRoleService_UndeleteInstanceRole_FullMethodName: iam.PermissionInstanceRolesUndelete,

	v1pb.ExternalVersionControlService_ListExternalVersionControls_FullMethodName:          iam.PermissionExternalVersionControlsList,
	v1pb.ExternalVersionControlService_GetExternalVersionControl_FullMethodName:            iam.PermissionExternalVersionControlsGet,
	v1pb.ExternalVersionControlService_CreateExternalVersionControl_FullMethodName:         iam.PermissionExternalVersionControlsCreate,
	v1pb.ExternalVersionControlService_ExchangeToken_FullMethodName:                        iam.PermissionExternalVersionControlsUpdate,
	v1pb.ExternalVersionControlService_UpdateExternalVersionControl_FullMethodName:         iam.PermissionExternalVersionControlsUpdate,
	v1pb.ExternalVersionControlService_DeleteExternalVersionControl_FullMethodName:         iam.PermissionExternalVersionControlsDelete,
	v1pb.ExternalVersionControlService_SearchExternalVersionControlProjects_FullMethodName: iam.PermissionExternalVersionControlsSearchProjects,
	v1pb.ExternalVersionControlService_ListProjectGitOpsInfo_FullMethodName:                iam.PermissionExternalVersionControlsListProjects,
	v1pb.RolloutService_ListPlans_FullMethodName:                                           iam.PermissionPlansList,
	v1pb.RolloutService_GetPlan_FullMethodName:                                             iam.PermissionPlansGet,
	v1pb.RolloutService_CreatePlan_FullMethodName:                                          iam.PermissionPlansCreate,
	v1pb.RolloutService_UpdatePlan_FullMethodName:                                          iam.PermissionPlansUpdate,
	v1pb.RolloutService_GetRollout_FullMethodName:                                          iam.PermissionRolloutsGet,
	v1pb.RolloutService_CreateRollout_FullMethodName:                                       iam.PermissionRolloutsCreate,
	v1pb.RolloutService_PreviewRollout_FullMethodName:                                      iam.PermissionRolloutsPreview,
	v1pb.RolloutService_ListTaskRuns_FullMethodName:                                        iam.PermissionTaskRunsList,
	v1pb.RolloutService_ListPlanCheckRuns_FullMethodName:                                   iam.PermissionPlanCheckRunsList,
	v1pb.RolloutService_RunPlanChecks_FullMethodName:                                       iam.PermissionPlanCheckRunsRun,
	v1pb.RolloutService_BatchRunTasks_FullMethodName:                                       iam.PermissionTasksRun,
	v1pb.RolloutService_BatchSkipTasks_FullMethodName:                                      iam.PermissionTasksSkip,
	v1pb.RolloutService_BatchCancelTaskRuns_FullMethodName:                                 iam.PermissionTaskRunsCancel,
	v1pb.SettingService_ListSettings_FullMethodName:                                        iam.PermissionSettingsList,
	v1pb.SettingService_GetSetting_FullMethodName:                                          iam.PermissionSettingsGet,
	v1pb.SettingService_SetSetting_FullMethodName:                                          iam.PermissionSettingsSet,

	v1pb.OrgPolicyService_ListPolicies_FullMethodName: iam.PermissionPoliciesList,
	v1pb.OrgPolicyService_GetPolicy_FullMethodName:    iam.PermissionPoliciesGet,
	v1pb.OrgPolicyService_CreatePolicy_FullMethodName: iam.PermissionPoliciesCreate,
	v1pb.OrgPolicyService_UpdatePolicy_FullMethodName: iam.PermissionPoliciesUpdate,
	v1pb.OrgPolicyService_DeletePolicy_FullMethodName: iam.PermissionPoliciesDelete,

	v1pb.BranchService_GetBranch_FullMethodName:    iam.PermissionBranchesGet,
	v1pb.BranchService_ListBranches_FullMethodName: iam.PermissionBranchesList,
	v1pb.BranchService_CreateBranch_FullMethodName: iam.PermissionBranchesCreate,
	v1pb.BranchService_UpdateBranch_FullMethodName: iam.PermissionBranchesUpdate,
	v1pb.BranchService_DeleteBranch_FullMethodName: iam.PermissionBranchesDelete,
	v1pb.BranchService_MergeBranch_FullMethodName:  iam.PermissionBranchesUpdate,
	v1pb.BranchService_RebaseBranch_FullMethodName: iam.PermissionBranchesUpdate,

	v1pb.IdentityProviderService_GetIdentityProvider_FullMethodName:      iam.PermissionIdentityProvidersGet,
	v1pb.IdentityProviderService_CreateIdentityProvider_FullMethodName:   iam.PermissionIdentityProvidersCreate,
	v1pb.IdentityProviderService_UpdateIdentityProvider_FullMethodName:   iam.PermissionIdentityProvidersUpdate,
	v1pb.IdentityProviderService_DeleteIdentityProvider_FullMethodName:   iam.PermissionIdentityProvidersDelete,
	v1pb.IdentityProviderService_UndeleteIdentityProvider_FullMethodName: iam.PermissionIdentityProvidersUndelete,
	v1pb.IdentityProviderService_TestIdentityProvider_FullMethodName:     iam.PermissionIdentityProvidersUpdate,
}

func isOwnerAndDBAMethod(methodName string) bool {
	return ownerAndDBAMethods[methodName]
}

func isProjectOwnerMethod(methodName string) bool {
	return projectOwnerMethods[methodName]
}

func isTransferDatabaseMethods(methodName string) bool {
	return transferDatabaseMethods[methodName]
}
