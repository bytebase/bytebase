# MCP method classification

Rendered from the `bytebase.v1.mcp_method_class`, `mcp_forbidden_reason` and
`mcp_exclusion_reason` annotations on the v1 RPCs. The annotations are the source of
truth; this file is a reviewable view of them and nothing reads it at runtime.

Regenerate with:

```
MCP_INVENTORY=write go test ./backend/api/v1/ -run TestMCPClassificationInventory
```

Only FORBIDDEN is enforced today. READ, WRITE and EXCLUDED record where a method
belongs; the gate that acts on them is a later change.

| Class | Methods | Meaning |
|---|---|---|
| READ | 47 | served to a read-only session and above |
| WRITE | 39 | served to a read-write session only |
| EXCLUDED | 93 | served by no mode this phase ships |
| FORBIDDEN | 29 | never served, enforced today |
| MCP_METHOD_CLASS_UNSPECIFIED | 0 | unclassified — CI rejects this |
| **total** | **208** | |

| Method | Class | Reason | Permission |
|---|---|---|---|
| AIService/Chat | EXCLUDED | SENDS_DATA_TO_A_THIRD_PARTY | — |
| AccessGrantService/ActivateAccessGrant | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.accessGrants.activate |
| AccessGrantService/CreateAccessGrant | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.accessGrants.create |
| AccessGrantService/GetAccessGrant | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.accessGrants.get |
| AccessGrantService/ListAccessGrants | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.accessGrants.list |
| AccessGrantService/RevokeAccessGrant | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.accessGrants.revoke |
| AccessGrantService/SearchMyAccessGrants | READ | — | bb.accessGrants.get |
| ActuatorService/GetActuatorInfo | READ | — | — |
| ActuatorService/SetupSample | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.projects.create |
| AuditLogService/ExportAuditLogs | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.auditLogs.export |
| AuditLogService/SearchAuditLogs | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.auditLogs.search |
| AuthService/ExchangeToken | FORBIDDEN | MINTS_CREDENTIAL | — |
| AuthService/Login | FORBIDDEN | MINTS_CREDENTIAL | — |
| AuthService/Logout | FORBIDDEN | ENDS_SESSION | — |
| AuthService/Refresh | FORBIDDEN | MINTS_CREDENTIAL | — |
| AuthService/RequestPasswordReset | FORBIDDEN | RESETS_CREDENTIAL | — |
| AuthService/ResetPassword | FORBIDDEN | RESETS_CREDENTIAL | — |
| AuthService/SendEmailLoginCode | FORBIDDEN | RESETS_CREDENTIAL | — |
| AuthService/Signup | FORBIDDEN | MINTS_CREDENTIAL | — |
| AuthService/SwitchWorkspace | FORBIDDEN | MINTS_CREDENTIAL | — |
| CelService/BatchDeparse | READ | — | — |
| CelService/BatchParse | READ | — | — |
| ChangelogService/GetChangelog | READ | — | bb.changelogs.get |
| ChangelogService/ListChangelogs | READ | — | bb.changelogs.list |
| DatabaseCatalogService/GetDatabaseCatalog | READ | — | bb.databaseCatalogs.get |
| DatabaseCatalogService/UpdateDatabaseCatalog | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.databaseCatalogs.update |
| DatabaseGroupService/CreateDatabaseGroup | WRITE | — | bb.databaseGroups.create |
| DatabaseGroupService/DeleteDatabaseGroup | WRITE | — | bb.databaseGroups.delete |
| DatabaseGroupService/GetDatabaseGroup | READ | — | bb.databaseGroups.get |
| DatabaseGroupService/ListDatabaseGroups | READ | — | bb.databaseGroups.list |
| DatabaseGroupService/UpdateDatabaseGroup | WRITE | — | bb.databaseGroups.update |
| DatabaseService/BatchGetDatabases | READ | — | bb.databases.get |
| DatabaseService/BatchSyncDatabases | WRITE | — | bb.databases.sync |
| DatabaseService/BatchUpdateDatabases | WRITE | — | bb.databases.update |
| DatabaseService/DiffMetadata | WRITE | — | bb.databases.diffMetadata |
| DatabaseService/DiffSchema | WRITE | — | bb.databases.get |
| DatabaseService/GetDatabase | READ | — | bb.databases.get |
| DatabaseService/GetDatabaseMetadata | READ | — | bb.databases.getSchema |
| DatabaseService/GetDatabaseSDLSchema | READ | — | bb.databases.getSchema |
| DatabaseService/GetDatabaseSchema | READ | — | bb.databases.getSchema |
| DatabaseService/GetSchemaString | READ | — | bb.databases.getSchema |
| DatabaseService/ListDatabases | READ | — | bb.databases.list |
| DatabaseService/SyncDatabase | WRITE | — | bb.databases.sync |
| DatabaseService/UpdateDatabase | WRITE | — | bb.databases.update |
| GroupService/BatchGetGroups | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.groups.get |
| GroupService/CreateGroup | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.groups.create |
| GroupService/DeleteGroup | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.groups.delete |
| GroupService/GetGroup | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.groups.get |
| GroupService/ListGroups | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.groups.list |
| GroupService/UpdateGroup | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.groups.update |
| IdentityProviderService/CreateIdentityProvider | FORBIDDEN | MINTS_CREDENTIAL_FOR_OTHERS | bb.identityProviders.create |
| IdentityProviderService/DeleteIdentityProvider | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.identityProviders.delete |
| IdentityProviderService/GetIdentityProvider | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.identityProviders.get |
| IdentityProviderService/ListIdentityProviders | EXCLUDED | ADMINISTERS_THE_WORKSPACE | — |
| IdentityProviderService/TestIdentityProvider | FORBIDDEN | MINTS_CREDENTIAL_FOR_OTHERS | bb.identityProviders.update |
| IdentityProviderService/UpdateIdentityProvider | FORBIDDEN | MINTS_CREDENTIAL_FOR_OTHERS | bb.identityProviders.update |
| InstanceRoleService/ListInstanceRoles | EXCLUDED | RETURNS_A_STORED_SECRET | bb.instanceRoles.list |
| InstanceService/AddDataSource | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.instances.update |
| InstanceService/BatchSyncInstances | WRITE | — | bb.instances.sync |
| InstanceService/BatchUpdateInstances | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.instances.update |
| InstanceService/CreateInstance | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.instances.create |
| InstanceService/DeleteInstance | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.instances.delete |
| InstanceService/GetInstance | EXCLUDED | RETURNS_A_STORED_SECRET | bb.instances.get |
| InstanceService/ListInstanceDatabase | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.instances.get |
| InstanceService/ListInstances | EXCLUDED | RETURNS_A_STORED_SECRET | bb.instances.list |
| InstanceService/RemoveDataSource | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.instances.update |
| InstanceService/SyncInstance | WRITE | — | bb.instances.sync |
| InstanceService/UndeleteInstance | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.instances.undelete |
| InstanceService/UpdateDataSource | FORBIDDEN | MINTS_CREDENTIAL_FOR_OTHERS | bb.instances.update |
| InstanceService/UpdateInstance | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.instances.update |
| IssueService/ApproveIssue | FORBIDDEN | DRIVES_THE_APPROVAL_DECISION | — |
| IssueService/BatchUpdateIssuesStatus | WRITE | — | bb.issues.update |
| IssueService/CreateIssue | WRITE | — | bb.issues.create |
| IssueService/CreateIssueComment | WRITE | — | bb.issueComments.create |
| IssueService/GetIssue | READ | — | bb.issues.get |
| IssueService/ListIssueComments | READ | — | bb.issueComments.list |
| IssueService/ListIssues | READ | — | bb.issues.list |
| IssueService/RejectIssue | FORBIDDEN | DRIVES_THE_APPROVAL_DECISION | — |
| IssueService/RequestIssue | FORBIDDEN | DRIVES_THE_APPROVAL_DECISION | — |
| IssueService/RetryIssueApproval | FORBIDDEN | DRIVES_THE_APPROVAL_DECISION | — |
| IssueService/SearchIssues | READ | — | bb.issues.get |
| IssueService/UpdateIssue | WRITE | — | bb.issues.update |
| IssueService/UpdateIssueComment | WRITE | — | bb.issueComments.update |
| OrgPolicyService/CreatePolicy | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.policies.create |
| OrgPolicyService/DeletePolicy | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.policies.delete |
| OrgPolicyService/GetPolicy | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.policies.get |
| OrgPolicyService/ListPolicies | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.policies.list |
| OrgPolicyService/UpdatePolicy | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.policies.update |
| PlanService/CancelPlanCheckRun | WRITE | — | bb.planCheckRuns.run |
| PlanService/CreatePlan | WRITE | — | bb.plans.create |
| PlanService/GetPlan | READ | — | bb.plans.get |
| PlanService/GetPlanCheckRun | READ | — | bb.planCheckRuns.get |
| PlanService/ListPlans | READ | — | bb.plans.list |
| PlanService/RunPlanChecks | WRITE | — | bb.planCheckRuns.run |
| PlanService/UpdatePlan | WRITE | — | bb.plans.update |
| ProjectService/AddWebhook | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.projects.update |
| ProjectService/BatchDeleteProjects | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.projects.delete |
| ProjectService/BatchGetProjects | EXCLUDED | RETURNS_A_STORED_SECRET | bb.projects.get |
| ProjectService/CreateProject | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.projects.create |
| ProjectService/DeleteProject | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.projects.delete |
| ProjectService/GetIamPolicy | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.projects.getIamPolicy |
| ProjectService/GetProject | EXCLUDED | RETURNS_A_STORED_SECRET | bb.projects.get |
| ProjectService/ListProjects | EXCLUDED | RETURNS_A_STORED_SECRET | bb.projects.list |
| ProjectService/RemoveWebhook | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.projects.update |
| ProjectService/SearchProjects | EXCLUDED | RETURNS_A_STORED_SECRET | — |
| ProjectService/SetIamPolicy | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.projects.setIamPolicy |
| ProjectService/TestWebhook | EXCLUDED | SENDS_DATA_TO_A_THIRD_PARTY | bb.projects.update |
| ProjectService/UndeleteProject | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.projects.undelete |
| ProjectService/UpdateProject | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.projects.update |
| ProjectService/UpdateWebhook | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.projects.update |
| QueryHistoryService/GetQueryHistory | READ | — | — |
| QueryHistoryService/ListQueryHistories | EXCLUDED | READS_OTHER_USERS_SQL | bb.queryHistories.list |
| QueryHistoryService/SearchQueryHistories | READ | — | — |
| ReleaseService/CheckRelease | WRITE | — | bb.releases.check |
| ReleaseService/CreateRelease | WRITE | — | bb.releases.create |
| ReleaseService/DeleteRelease | WRITE | — | bb.releases.delete |
| ReleaseService/GetRelease | READ | — | bb.releases.get |
| ReleaseService/ListReleaseCategories | READ | — | bb.releases.list |
| ReleaseService/ListReleases | READ | — | bb.releases.list |
| ReleaseService/UndeleteRelease | WRITE | — | bb.releases.undelete |
| ReleaseService/UpdateRelease | WRITE | — | bb.releases.update |
| ReviewConfigService/CreateReviewConfig | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.reviewConfigs.create |
| ReviewConfigService/DeleteReviewConfig | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.reviewConfigs.delete |
| ReviewConfigService/GetReviewConfig | READ | — | bb.reviewConfigs.get |
| ReviewConfigService/ListReviewConfigs | READ | — | bb.reviewConfigs.list |
| ReviewConfigService/UpdateReviewConfig | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.reviewConfigs.update |
| RevisionService/BatchCreateRevisions | WRITE | — | bb.revisions.create |
| RevisionService/DeleteRevision | WRITE | — | bb.revisions.delete |
| RevisionService/GetRevision | READ | — | bb.revisions.get |
| RevisionService/ListRevisions | READ | — | bb.revisions.list |
| RoleService/CreateRole | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.roles.create |
| RoleService/DeleteRole | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.roles.delete |
| RoleService/GetRole | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.roles.get |
| RoleService/ListRoles | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.roles.list |
| RoleService/UpdateRole | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.roles.update |
| RolloutService/BatchCancelTaskRuns | WRITE | — | — |
| RolloutService/BatchRunTasks | WRITE | — | — |
| RolloutService/BatchSkipTasks | WRITE | — | — |
| RolloutService/CreateRollout | WRITE | — | — |
| RolloutService/GetRollout | READ | — | bb.rollouts.get |
| RolloutService/GetTaskRun | READ | — | bb.taskRuns.list |
| RolloutService/GetTaskRunLog | READ | — | bb.taskRuns.list |
| RolloutService/GetTaskRunSession | EXCLUDED | OPENS_AN_ADMIN_CONNECTION | bb.taskRuns.list |
| RolloutService/ListRollouts | READ | — | bb.rollouts.list |
| RolloutService/ListTaskRuns | READ | — | bb.taskRuns.list |
| RolloutService/PreviewTaskRunRollback | READ | — | bb.taskRuns.list |
| SQLService/AdminExecute | EXCLUDED | OPENS_AN_ADMIN_CONNECTION | bb.sql.admin |
| SQLService/Export | WRITE | — | bb.databases.get |
| SQLService/GetQueryHistory | READ | — | — |
| SQLService/ListQueryHistories | EXCLUDED | READS_OTHER_USERS_SQL | bb.queryHistories.list |
| SQLService/Query | READ | — | bb.databases.get |
| SQLService/SearchQueryHistories | READ | — | — |
| SavedQueryService/CreateSavedQuery | WRITE | — | bb.savedQueries.create |
| SavedQueryService/DeleteSavedQuery | WRITE | — | — |
| SavedQueryService/GetSavedQuery | READ | — | — |
| SavedQueryService/GetSavedQueryPolicy | EXCLUDED | ADMINISTERS_THE_WORKSPACE | — |
| SavedQueryService/ListSavedQueries | EXCLUDED | READS_OTHER_USERS_SQL | bb.savedQueries.list |
| SavedQueryService/MoveMySavedQueries | WRITE | — | — |
| SavedQueryService/SearchSavedQueries | READ | — | — |
| SavedQueryService/SearchSavedQueryFolders | READ | — | — |
| SavedQueryService/SetSavedQueryPolicy | EXCLUDED | ADMINISTERS_THE_WORKSPACE | — |
| SavedQueryService/UpdateSavedQuery | WRITE | — | — |
| SavedQueryService/UpdateSavedQueryStar | WRITE | — | — |
| ServiceAccountService/CreateServiceAccount | FORBIDDEN | MINTS_CREDENTIAL_FOR_OTHERS | bb.serviceAccounts.create |
| ServiceAccountService/DeleteServiceAccount | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.serviceAccounts.delete |
| ServiceAccountService/GetServiceAccount | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.serviceAccounts.get |
| ServiceAccountService/ListServiceAccounts | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.serviceAccounts.list |
| ServiceAccountService/UndeleteServiceAccount | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.serviceAccounts.undelete |
| ServiceAccountService/UpdateServiceAccount | FORBIDDEN | MINTS_CREDENTIAL_FOR_OTHERS | bb.serviceAccounts.update |
| SettingService/GetSetting | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.settings.get |
| SettingService/ListSettings | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.settings.list |
| SettingService/TestEmailSetting | FORBIDDEN | MINTS_CREDENTIAL_FOR_OTHERS | bb.settings.set |
| SettingService/UpdateSetting | FORBIDDEN | REWRITES_SESSION_BOUNDARY | bb.settings.set |
| SheetService/BatchCreateSheets | WRITE | — | bb.sheets.create |
| SheetService/CreateSheet | WRITE | — | bb.sheets.create |
| SheetService/GetSheet | READ | — | bb.sheets.get |
| SubscriptionService/CancelPurchase | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.subscription.manage |
| SubscriptionService/CreatePurchase | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.subscription.manage |
| SubscriptionService/ExportVCSProviderUsers | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.subscription.manage |
| SubscriptionService/GetPaymentInfo | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.subscription.manage |
| SubscriptionService/GetSubscription | EXCLUDED | ADMINISTERS_THE_WORKSPACE | — |
| SubscriptionService/ListPurchasePlans | EXCLUDED | ADMINISTERS_THE_WORKSPACE | — |
| SubscriptionService/UpdatePurchase | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.subscription.manage |
| SubscriptionService/UploadLicense | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.subscription.manage |
| SubscriptionService/VerifyCheckoutSession | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.subscription.manage |
| UserService/BatchGetUsers | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.users.get |
| UserService/CreateUser | FORBIDDEN | MINTS_CREDENTIAL_FOR_OTHERS | bb.users.create |
| UserService/DeleteUser | EXCLUDED | ADMINISTERS_THE_WORKSPACE | — |
| UserService/GetCurrentUser | EXCLUDED | RETURNS_A_STORED_SECRET | — |
| UserService/GetUser | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.users.get |
| UserService/ListUsers | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.users.list |
| UserService/UndeleteUser | EXCLUDED | ADMINISTERS_THE_WORKSPACE | — |
| UserService/UpdateEmail | FORBIDDEN | MINTS_CREDENTIAL_FOR_OTHERS | bb.users.updateEmail |
| UserService/UpdateUser | FORBIDDEN | TAKES_OVER_ACCOUNT | — |
| WorkloadIdentityService/CreateWorkloadIdentity | FORBIDDEN | MINTS_CREDENTIAL_FOR_OTHERS | bb.workloadIdentities.create |
| WorkloadIdentityService/DeleteWorkloadIdentity | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.workloadIdentities.delete |
| WorkloadIdentityService/GetWorkloadIdentity | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.workloadIdentities.get |
| WorkloadIdentityService/ListWorkloadIdentities | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.workloadIdentities.list |
| WorkloadIdentityService/UndeleteWorkloadIdentity | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.workloadIdentities.undelete |
| WorkloadIdentityService/UpdateWorkloadIdentity | FORBIDDEN | MINTS_CREDENTIAL_FOR_OTHERS | bb.workloadIdentities.update |
| WorkspaceService/DeleteWorkspace | FORBIDDEN | ENDS_MEMBERSHIP | bb.workspaces.delete |
| WorkspaceService/GetIamPolicy | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.workspaces.getIamPolicy |
| WorkspaceService/GetWorkspace | READ | — | — |
| WorkspaceService/LeaveWorkspace | FORBIDDEN | ENDS_MEMBERSHIP | — |
| WorkspaceService/ListWorkspaces | READ | — | — |
| WorkspaceService/RotateDirectorySyncToken | FORBIDDEN | MINTS_CREDENTIAL_FOR_OTHERS | bb.workspaces.rotateDirectorySyncToken |
| WorkspaceService/SetIamPolicy | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.workspaces.setIamPolicy |
| WorkspaceService/UpdateWorkspace | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.workspaces.update |
