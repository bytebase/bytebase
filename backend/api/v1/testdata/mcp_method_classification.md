# MCP method classification

Rendered from the `bytebase.v1.mcp_method_class` and `mcp_denial_reason`
annotations on the v1 RPCs. The annotations are the source of
truth; this file is a reviewable view of them and nothing reads it at runtime.

Regenerate with:

```
MCP_INVENTORY=write go test ./backend/api/v1/ -run TestMCPClassificationInventory
```

Every class is enforced by the MCP gate. READ and WRITE are the serving classes a
workspace's MCP capability ceiling selects between; EXCLUDED and FORBIDDEN are
served by no ceiling.

| Class | Methods | Meaning |
|---|---|---|
| READ | 56 | served to a read-only session and above |
| WRITE | 41 | served to a read-write session only |
| EXCLUDED | 86 | served by no ceiling this phase ships |
| FORBIDDEN | 35 | never served, whatever the ceiling |
| MCP_METHOD_CLASS_UNSPECIFIED | 0 | unclassified — CI rejects this, and the gate refuses it |
| **total** | **218** | |

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
| AuditLogService/ExportAuditLogs | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.auditLogs.export |
| AuditLogService/SearchAuditLogs | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.auditLogs.search |
| AuthService/ExchangeToken | FORBIDDEN | MINTS_CREDENTIAL | — |
| AuthService/GetAuthenticationInfo | READ | — | — |
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
| IdentityProviderService/ListIdentityProviders | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.identityProviders.list |
| IdentityProviderService/TestIdentityProvider | FORBIDDEN | MINTS_CREDENTIAL_FOR_OTHERS | bb.identityProviders.update |
| IdentityProviderService/UpdateIdentityProvider | FORBIDDEN | MINTS_CREDENTIAL_FOR_OTHERS | bb.identityProviders.update |
| InstanceRoleService/ListInstanceRoles | READ | — | bb.instanceRoles.list |
| InstanceService/AddDataSource | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.instances.update |
| InstanceService/BatchSyncInstances | WRITE | — | bb.instances.sync |
| InstanceService/BatchUpdateInstances | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.instances.update |
| InstanceService/CreateInstance | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.instances.create |
| InstanceService/DeleteInstance | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.instances.delete |
| InstanceService/GetInstance | READ | — | bb.instances.get |
| InstanceService/ListInstanceDatabase | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.instances.get |
| InstanceService/ListInstances | READ | — | bb.instances.list |
| InstanceService/PrepareSampleProjectInstance | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.instances.create |
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
| IssueService/RequestIssue | WRITE | — | — |
| IssueService/RetryIssueApproval | FORBIDDEN | DRIVES_THE_APPROVAL_DECISION | — |
| IssueService/RunReview | WRITE | — | bb.reviewRuns.run |
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
| ProjectService/BatchGetProjects | READ | — | bb.projects.get |
| ProjectService/CreateProject | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.projects.create |
| ProjectService/DeleteProject | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.projects.delete |
| ProjectService/GetIamPolicy | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.projects.getIamPolicy |
| ProjectService/GetProject | READ | — | bb.projects.get |
| ProjectService/ListProjects | READ | — | bb.projects.list |
| ProjectService/RemoveWebhook | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.projects.update |
| ProjectService/SearchProjects | READ | — | — |
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
| SubscriptionService/StartTrial | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.subscription.manage |
| SubscriptionService/UpdatePurchase | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.subscription.manage |
| SubscriptionService/UploadLicense | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.subscription.manage |
| SubscriptionService/VerifyCheckoutSession | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.subscription.manage |
| UserService/BatchGetUsers | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.users.get |
| UserService/ChangePassword | FORBIDDEN | TAKES_OVER_ACCOUNT | — |
| UserService/ConfirmRecoveryCodes | FORBIDDEN | TAKES_OVER_ACCOUNT | — |
| UserService/CreateUser | FORBIDDEN | MINTS_CREDENTIAL_FOR_OTHERS | bb.users.create |
| UserService/DeleteUser | EXCLUDED | ADMINISTERS_THE_WORKSPACE | — |
| UserService/DisableMFA | FORBIDDEN | TAKES_OVER_ACCOUNT | — |
| UserService/EnableMFA | FORBIDDEN | TAKES_OVER_ACCOUNT | — |
| UserService/GetCurrentUser | READ | — | — |
| UserService/GetUser | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.users.get |
| UserService/ListUsers | EXCLUDED | ADMINISTERS_THE_WORKSPACE | bb.users.list |
| UserService/RegenerateRecoveryCodes | FORBIDDEN | MINTS_CREDENTIAL | — |
| UserService/RequestReauthCode | FORBIDDEN | RESETS_CREDENTIAL | — |
| UserService/StartMFAEnrollment | FORBIDDEN | MINTS_CREDENTIAL | — |
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

## What a refused method tells the agent

Rendered by the gate's own code. Each denial must be true of every method listed
under it: the whole method, for every caller, argument and resource owner.

### FORBIDDEN · MINTS_CREDENTIAL

AuthService/ExchangeToken

> `<method>` is not available to MCP sessions, whatever the workspace's MCP access policy, because it returns a sign-in credential (a session token, an MFA secret, or recovery codes) that would keep working after this MCP connection is revoked. Exchange workload identity tokens from your CI/CD pipeline instead.

### FORBIDDEN · MINTS_CREDENTIAL

AuthService/Login, AuthService/Refresh, AuthService/Signup, UserService/RegenerateRecoveryCodes, UserService/StartMFAEnrollment

> `<method>` is not available to MCP sessions, whatever the workspace's MCP access policy, because it returns a sign-in credential (a session token, an MFA secret, or recovery codes) that would keep working after this MCP connection is revoked. If you need this, do it yourself in the Bytebase console.

### FORBIDDEN · MINTS_CREDENTIAL

AuthService/SwitchWorkspace

> `<method>` is not available to MCP sessions, whatever the workspace's MCP access policy, because it returns a sign-in credential (a session token, an MFA secret, or recovery codes) that would keep working after this MCP connection is revoked. To use this MCP connection with another workspace, run the reauthorize tool and choose that workspace when you approve access again.

### FORBIDDEN · RESETS_CREDENTIAL

AuthService/RequestPasswordReset, AuthService/ResetPassword, AuthService/SendEmailLoginCode, UserService/RequestReauthCode

> `<method>` is not available to MCP sessions, whatever the workspace's MCP access policy, because it sends or redeems a one-time code or reset link that can sign in to an account or change its credentials. If you need this, do it yourself in the Bytebase console.

### FORBIDDEN · TAKES_OVER_ACCOUNT

UserService/ChangePassword, UserService/ConfirmRecoveryCodes, UserService/DisableMFA, UserService/EnableMFA, UserService/UpdateUser

> `<method>` is not available to MCP sessions, whatever the workspace's MCP access policy, because it can rewrite an account's credentials, which would let the session take that account over. If you need this, do it yourself in the Bytebase console.

### FORBIDDEN · ENDS_SESSION

AuthService/Logout

> `<method>` is not available to MCP sessions, whatever the workspace's MCP access policy, because it signs the user out of their Bytebase web session. To end this MCP connection instead, run the reauthorize tool or remove Bytebase from your MCP client.

### FORBIDDEN · ENDS_MEMBERSHIP

WorkspaceService/DeleteWorkspace, WorkspaceService/LeaveWorkspace

> `<method>` is not available to MCP sessions, whatever the workspace's MCP access policy, because it deletes the workspace or removes the user from it, and can return a sign-in token for another workspace that this MCP connection's limits would not cover. If you need this, do it yourself in the Bytebase console.

### FORBIDDEN · MINTS_CREDENTIAL_FOR_OTHERS

IdentityProviderService/CreateIdentityProvider, IdentityProviderService/TestIdentityProvider, IdentityProviderService/UpdateIdentityProvider, InstanceService/UpdateDataSource, ServiceAccountService/CreateServiceAccount, ServiceAccountService/UpdateServiceAccount, SettingService/TestEmailSetting, UserService/CreateUser, UserService/UpdateEmail, WorkloadIdentityService/CreateWorkloadIdentity, WorkloadIdentityService/UpdateWorkloadIdentity, WorkspaceService/RotateDirectorySyncToken

> `<method>` is not available to MCP sessions, whatever the workspace's MCP access policy, because it can create, reveal, or redirect a credential for another account, service, or database (a key, token, password, or sign-in trust), and revoking this MCP connection would not take that credential back. If your role allows it, do this in the Bytebase console.

### FORBIDDEN · REWRITES_SESSION_BOUNDARY

SettingService/UpdateSetting

> `<method>` is not available to MCP sessions, whatever the workspace's MCP access policy, because it changes workspace settings, and some of them (the MCP access policy, sign-in and SSO, the mail server, the AI provider) control what this session can do, so AI agents may not change any workspace setting. If your role allows it, do this in the Bytebase console.

### FORBIDDEN · DRIVES_THE_APPROVAL_DECISION

IssueService/ApproveIssue, IssueService/RejectIssue

> `<method>` is not available to MCP sessions, whatever the workspace's MCP access policy, because it approves, rejects, or re-checks an issue's approval, and AI agents may not make approval decisions on any issue, whoever created it. If you are an approver for this issue, approve or reject it in the Bytebase console.

### FORBIDDEN · DRIVES_THE_APPROVAL_DECISION

IssueService/RetryIssueApproval

> `<method>` is not available to MCP sessions, whatever the workspace's MCP access policy, because it approves, rejects, or re-checks an issue's approval, and AI agents may not make approval decisions on any issue, whoever created it. The issue's creator can re-run its approval check in the Bytebase console.

### EXCLUDED · ADMINISTERS_THE_WORKSPACE

AccessGrantService/ActivateAccessGrant, AccessGrantService/CreateAccessGrant, AccessGrantService/GetAccessGrant, AccessGrantService/ListAccessGrants, AccessGrantService/RevokeAccessGrant, AuditLogService/ExportAuditLogs, AuditLogService/SearchAuditLogs, DatabaseCatalogService/UpdateDatabaseCatalog, GroupService/BatchGetGroups, GroupService/CreateGroup, GroupService/DeleteGroup, GroupService/GetGroup, GroupService/ListGroups, GroupService/UpdateGroup, IdentityProviderService/DeleteIdentityProvider, IdentityProviderService/GetIdentityProvider, IdentityProviderService/ListIdentityProviders, InstanceService/AddDataSource, InstanceService/BatchUpdateInstances, InstanceService/CreateInstance, InstanceService/DeleteInstance, InstanceService/ListInstanceDatabase, InstanceService/PrepareSampleProjectInstance, InstanceService/RemoveDataSource, InstanceService/UndeleteInstance, InstanceService/UpdateInstance, OrgPolicyService/CreatePolicy, OrgPolicyService/DeletePolicy, OrgPolicyService/GetPolicy, OrgPolicyService/ListPolicies, OrgPolicyService/UpdatePolicy, ProjectService/AddWebhook, ProjectService/BatchDeleteProjects, ProjectService/CreateProject, ProjectService/DeleteProject, ProjectService/GetIamPolicy, ProjectService/RemoveWebhook, ProjectService/SetIamPolicy, ProjectService/UndeleteProject, ProjectService/UpdateProject, ProjectService/UpdateWebhook, ReviewConfigService/CreateReviewConfig, ReviewConfigService/DeleteReviewConfig, ReviewConfigService/UpdateReviewConfig, RoleService/CreateRole, RoleService/DeleteRole, RoleService/GetRole, RoleService/ListRoles, RoleService/UpdateRole, SavedQueryService/GetSavedQueryPolicy, SavedQueryService/SetSavedQueryPolicy, ServiceAccountService/DeleteServiceAccount, ServiceAccountService/GetServiceAccount, ServiceAccountService/ListServiceAccounts, ServiceAccountService/UndeleteServiceAccount, SettingService/GetSetting, SettingService/ListSettings, SubscriptionService/CancelPurchase, SubscriptionService/CreatePurchase, SubscriptionService/ExportVCSProviderUsers, SubscriptionService/GetPaymentInfo, SubscriptionService/GetSubscription, SubscriptionService/ListPurchasePlans, SubscriptionService/StartTrial, SubscriptionService/UpdatePurchase, SubscriptionService/UploadLicense, SubscriptionService/VerifyCheckoutSession, UserService/BatchGetUsers, UserService/DeleteUser, UserService/GetUser, UserService/ListUsers, UserService/UndeleteUser, WorkloadIdentityService/DeleteWorkloadIdentity, WorkloadIdentityService/GetWorkloadIdentity, WorkloadIdentityService/ListWorkloadIdentities, WorkloadIdentityService/UndeleteWorkloadIdentity, WorkspaceService/GetIamPolicy, WorkspaceService/SetIamPolicy, WorkspaceService/UpdateWorkspace

> `<method>` is not available to MCP sessions under any MCP access policy because it belongs to workspace administration: members, roles and access, sign-in, instances and projects, policies and data classification, audit logs, settings, and billing. If your role allows it, do this in the Bytebase console.

### EXCLUDED · READS_OTHER_USERS_SQL

QueryHistoryService/ListQueryHistories, SQLService/ListQueryHistories, SavedQueryService/ListSavedQueries

> `<method>` is not available to MCP sessions under any MCP access policy because it returns SQL that other people wrote, across the workspace or past the sharing that keeps a saved query private. To read your own query history, call QueryHistoryService/SearchQueryHistories; to find saved queries you can open, call SavedQueryService/SearchSavedQueries.

### EXCLUDED · OPENS_AN_ADMIN_CONNECTION

RolloutService/GetTaskRunSession, SQLService/AdminExecute

> `<method>` is not available to MCP sessions under any MCP access policy because it opens an admin-credentialed connection to the database and returns other sessions' live, unmasked SQL. If your role allows it, do this in the Bytebase console.

### EXCLUDED · SENDS_DATA_TO_A_THIRD_PARTY

AIService/Chat, ProjectService/TestWebhook

> `<method>` is not available to MCP sessions under any MCP access policy because it contacts an outside service (the configured AI provider or a webhook endpoint) on the workspace's behalf. If your role allows it, do this in the Bytebase console.
