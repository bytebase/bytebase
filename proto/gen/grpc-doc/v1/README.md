# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [v1/actuator_service.proto](#v1_actuator_service-proto)
    - [ActuatorInfo](#bytebase-v1-ActuatorInfo)
    - [DebugLog](#bytebase-v1-DebugLog)
    - [DeleteCacheRequest](#bytebase-v1-DeleteCacheRequest)
    - [GetActuatorInfoRequest](#bytebase-v1-GetActuatorInfoRequest)
    - [ListDebugLogRequest](#bytebase-v1-ListDebugLogRequest)
    - [ListDebugLogResponse](#bytebase-v1-ListDebugLogResponse)
    - [UpdateActuatorInfoRequest](#bytebase-v1-UpdateActuatorInfoRequest)
  
    - [ActuatorService](#bytebase-v1-ActuatorService)
  
- [v1/common.proto](#v1_common-proto)
    - [Engine](#bytebase-v1-Engine)
    - [MaskingLevel](#bytebase-v1-MaskingLevel)
    - [State](#bytebase-v1-State)
  
- [v1/deployment.proto](#v1_deployment-proto)
    - [DeploymentType](#bytebase-v1-DeploymentType)
  
- [v1/iam_policy.proto](#v1_iam_policy-proto)
    - [Binding](#bytebase-v1-Binding)
    - [IamPolicy](#bytebase-v1-IamPolicy)
  
- [v1/org_policy_service.proto](#v1_org_policy_service-proto)
    - [BackupPlanPolicy](#bytebase-v1-BackupPlanPolicy)
    - [CreatePolicyRequest](#bytebase-v1-CreatePolicyRequest)
    - [DeletePolicyRequest](#bytebase-v1-DeletePolicyRequest)
    - [DeploymentApprovalPolicy](#bytebase-v1-DeploymentApprovalPolicy)
    - [DeploymentApprovalStrategy](#bytebase-v1-DeploymentApprovalStrategy)
    - [DisableCopyDataPolicy](#bytebase-v1-DisableCopyDataPolicy)
    - [GetPolicyRequest](#bytebase-v1-GetPolicyRequest)
    - [ListPoliciesRequest](#bytebase-v1-ListPoliciesRequest)
    - [ListPoliciesResponse](#bytebase-v1-ListPoliciesResponse)
    - [MaskData](#bytebase-v1-MaskData)
    - [MaskingExceptionPolicy](#bytebase-v1-MaskingExceptionPolicy)
    - [MaskingExceptionPolicy.MaskingException](#bytebase-v1-MaskingExceptionPolicy-MaskingException)
    - [MaskingPolicy](#bytebase-v1-MaskingPolicy)
    - [MaskingRulePolicy](#bytebase-v1-MaskingRulePolicy)
    - [MaskingRulePolicy.MaskingRule](#bytebase-v1-MaskingRulePolicy-MaskingRule)
    - [Policy](#bytebase-v1-Policy)
    - [SQLReviewPolicy](#bytebase-v1-SQLReviewPolicy)
    - [SQLReviewRule](#bytebase-v1-SQLReviewRule)
    - [SlowQueryPolicy](#bytebase-v1-SlowQueryPolicy)
    - [UpdatePolicyRequest](#bytebase-v1-UpdatePolicyRequest)
  
    - [ApprovalGroup](#bytebase-v1-ApprovalGroup)
    - [ApprovalStrategy](#bytebase-v1-ApprovalStrategy)
    - [BackupPlanSchedule](#bytebase-v1-BackupPlanSchedule)
    - [MaskingExceptionPolicy.MaskingException.Action](#bytebase-v1-MaskingExceptionPolicy-MaskingException-Action)
    - [PolicyResourceType](#bytebase-v1-PolicyResourceType)
    - [PolicyType](#bytebase-v1-PolicyType)
    - [SQLReviewRuleLevel](#bytebase-v1-SQLReviewRuleLevel)
  
    - [OrgPolicyService](#bytebase-v1-OrgPolicyService)
  
- [v1/anomaly_service.proto](#v1_anomaly_service-proto)
    - [Anomaly](#bytebase-v1-Anomaly)
    - [Anomaly.DatabaseBackupMissingDetail](#bytebase-v1-Anomaly-DatabaseBackupMissingDetail)
    - [Anomaly.DatabaseBackupPolicyViolationDetail](#bytebase-v1-Anomaly-DatabaseBackupPolicyViolationDetail)
    - [Anomaly.DatabaseConnectionDetail](#bytebase-v1-Anomaly-DatabaseConnectionDetail)
    - [Anomaly.DatabaseSchemaDriftDetail](#bytebase-v1-Anomaly-DatabaseSchemaDriftDetail)
    - [Anomaly.InstanceConnectionDetail](#bytebase-v1-Anomaly-InstanceConnectionDetail)
    - [SearchAnomaliesRequest](#bytebase-v1-SearchAnomaliesRequest)
    - [SearchAnomaliesResponse](#bytebase-v1-SearchAnomaliesResponse)
  
    - [Anomaly.AnomalySeverity](#bytebase-v1-Anomaly-AnomalySeverity)
    - [Anomaly.AnomalyType](#bytebase-v1-Anomaly-AnomalyType)
  
    - [AnomalyService](#bytebase-v1-AnomalyService)
  
- [v1/auth_service.proto](#v1_auth_service-proto)
    - [CreateUserRequest](#bytebase-v1-CreateUserRequest)
    - [DeleteUserRequest](#bytebase-v1-DeleteUserRequest)
    - [GetUserRequest](#bytebase-v1-GetUserRequest)
    - [IdentityProviderContext](#bytebase-v1-IdentityProviderContext)
    - [ListUsersRequest](#bytebase-v1-ListUsersRequest)
    - [ListUsersResponse](#bytebase-v1-ListUsersResponse)
    - [LoginRequest](#bytebase-v1-LoginRequest)
    - [LoginResponse](#bytebase-v1-LoginResponse)
    - [LogoutRequest](#bytebase-v1-LogoutRequest)
    - [OAuth2IdentityProviderContext](#bytebase-v1-OAuth2IdentityProviderContext)
    - [OIDCIdentityProviderContext](#bytebase-v1-OIDCIdentityProviderContext)
    - [UndeleteUserRequest](#bytebase-v1-UndeleteUserRequest)
    - [UpdateUserRequest](#bytebase-v1-UpdateUserRequest)
    - [User](#bytebase-v1-User)
  
    - [UserRole](#bytebase-v1-UserRole)
    - [UserType](#bytebase-v1-UserType)
  
    - [AuthService](#bytebase-v1-AuthService)
  
- [v1/bookmark_service.proto](#v1_bookmark_service-proto)
    - [Bookmark](#bytebase-v1-Bookmark)
    - [CreateBookmarkRequest](#bytebase-v1-CreateBookmarkRequest)
    - [DeleteBookmarkRequest](#bytebase-v1-DeleteBookmarkRequest)
    - [ListBookmarksRequest](#bytebase-v1-ListBookmarksRequest)
    - [ListBookmarksResponse](#bytebase-v1-ListBookmarksResponse)
  
    - [BookmarkService](#bytebase-v1-BookmarkService)
  
- [v1/cel_service.proto](#v1_cel_service-proto)
    - [DeparseRequest](#bytebase-v1-DeparseRequest)
    - [DeparseResponse](#bytebase-v1-DeparseResponse)
    - [ParseRequest](#bytebase-v1-ParseRequest)
    - [ParseResponse](#bytebase-v1-ParseResponse)
  
    - [CelService](#bytebase-v1-CelService)
  
- [v1/database_edit.proto](#v1_database_edit-proto)
    - [AddColumnContext](#bytebase-v1-AddColumnContext)
    - [AddForeignKeyContext](#bytebase-v1-AddForeignKeyContext)
    - [AlterColumnContext](#bytebase-v1-AlterColumnContext)
    - [AlterTableContext](#bytebase-v1-AlterTableContext)
    - [CreateSchemaContext](#bytebase-v1-CreateSchemaContext)
    - [CreateTableContext](#bytebase-v1-CreateTableContext)
    - [DatabaseEdit](#bytebase-v1-DatabaseEdit)
    - [DropColumnContext](#bytebase-v1-DropColumnContext)
    - [DropSchemaContext](#bytebase-v1-DropSchemaContext)
    - [DropTableContext](#bytebase-v1-DropTableContext)
    - [RenameSchemaContext](#bytebase-v1-RenameSchemaContext)
    - [RenameTableContext](#bytebase-v1-RenameTableContext)
  
- [v1/vcs.proto](#v1_vcs-proto)
    - [Commit](#bytebase-v1-Commit)
    - [FileCommit](#bytebase-v1-FileCommit)
    - [PushEvent](#bytebase-v1-PushEvent)
  
    - [VcsType](#bytebase-v1-VcsType)
  
- [v1/database_service.proto](#v1_database_service-proto)
    - [AdviseIndexRequest](#bytebase-v1-AdviseIndexRequest)
    - [AdviseIndexResponse](#bytebase-v1-AdviseIndexResponse)
    - [Backup](#bytebase-v1-Backup)
    - [BackupSetting](#bytebase-v1-BackupSetting)
    - [BatchUpdateDatabasesRequest](#bytebase-v1-BatchUpdateDatabasesRequest)
    - [BatchUpdateDatabasesResponse](#bytebase-v1-BatchUpdateDatabasesResponse)
    - [ChangeHistory](#bytebase-v1-ChangeHistory)
    - [ChangedResourceDatabase](#bytebase-v1-ChangedResourceDatabase)
    - [ChangedResourceSchema](#bytebase-v1-ChangedResourceSchema)
    - [ChangedResourceTable](#bytebase-v1-ChangedResourceTable)
    - [ChangedResources](#bytebase-v1-ChangedResources)
    - [ColumnMetadata](#bytebase-v1-ColumnMetadata)
    - [CreateBackupRequest](#bytebase-v1-CreateBackupRequest)
    - [Database](#bytebase-v1-Database)
    - [Database.LabelsEntry](#bytebase-v1-Database-LabelsEntry)
    - [DatabaseMetadata](#bytebase-v1-DatabaseMetadata)
    - [DatabaseSchema](#bytebase-v1-DatabaseSchema)
    - [DeleteSecretRequest](#bytebase-v1-DeleteSecretRequest)
    - [DependentColumn](#bytebase-v1-DependentColumn)
    - [ExtensionMetadata](#bytebase-v1-ExtensionMetadata)
    - [ForeignKeyMetadata](#bytebase-v1-ForeignKeyMetadata)
    - [FunctionMetadata](#bytebase-v1-FunctionMetadata)
    - [GetBackupSettingRequest](#bytebase-v1-GetBackupSettingRequest)
    - [GetChangeHistoryRequest](#bytebase-v1-GetChangeHistoryRequest)
    - [GetDatabaseMetadataRequest](#bytebase-v1-GetDatabaseMetadataRequest)
    - [GetDatabaseRequest](#bytebase-v1-GetDatabaseRequest)
    - [GetDatabaseSchemaRequest](#bytebase-v1-GetDatabaseSchemaRequest)
    - [IndexMetadata](#bytebase-v1-IndexMetadata)
    - [ListBackupsRequest](#bytebase-v1-ListBackupsRequest)
    - [ListBackupsResponse](#bytebase-v1-ListBackupsResponse)
    - [ListChangeHistoriesRequest](#bytebase-v1-ListChangeHistoriesRequest)
    - [ListChangeHistoriesResponse](#bytebase-v1-ListChangeHistoriesResponse)
    - [ListDatabasesRequest](#bytebase-v1-ListDatabasesRequest)
    - [ListDatabasesResponse](#bytebase-v1-ListDatabasesResponse)
    - [ListSecretsRequest](#bytebase-v1-ListSecretsRequest)
    - [ListSecretsResponse](#bytebase-v1-ListSecretsResponse)
    - [ListSlowQueriesRequest](#bytebase-v1-ListSlowQueriesRequest)
    - [ListSlowQueriesResponse](#bytebase-v1-ListSlowQueriesResponse)
    - [SchemaMetadata](#bytebase-v1-SchemaMetadata)
    - [SearchDatabasesRequest](#bytebase-v1-SearchDatabasesRequest)
    - [SearchDatabasesResponse](#bytebase-v1-SearchDatabasesResponse)
    - [Secret](#bytebase-v1-Secret)
    - [SlowQueryDetails](#bytebase-v1-SlowQueryDetails)
    - [SlowQueryLog](#bytebase-v1-SlowQueryLog)
    - [SlowQueryStatistics](#bytebase-v1-SlowQueryStatistics)
    - [StreamMetadata](#bytebase-v1-StreamMetadata)
    - [SyncDatabaseRequest](#bytebase-v1-SyncDatabaseRequest)
    - [SyncDatabaseResponse](#bytebase-v1-SyncDatabaseResponse)
    - [TableMetadata](#bytebase-v1-TableMetadata)
    - [TaskMetadata](#bytebase-v1-TaskMetadata)
    - [UpdateBackupSettingRequest](#bytebase-v1-UpdateBackupSettingRequest)
    - [UpdateDatabaseRequest](#bytebase-v1-UpdateDatabaseRequest)
    - [UpdateSecretRequest](#bytebase-v1-UpdateSecretRequest)
    - [ViewMetadata](#bytebase-v1-ViewMetadata)
  
    - [Backup.BackupState](#bytebase-v1-Backup-BackupState)
    - [Backup.BackupType](#bytebase-v1-Backup-BackupType)
    - [ChangeHistory.Source](#bytebase-v1-ChangeHistory-Source)
    - [ChangeHistory.Status](#bytebase-v1-ChangeHistory-Status)
    - [ChangeHistory.Type](#bytebase-v1-ChangeHistory-Type)
    - [ChangeHistoryView](#bytebase-v1-ChangeHistoryView)
    - [StreamMetadata.Mode](#bytebase-v1-StreamMetadata-Mode)
    - [StreamMetadata.Type](#bytebase-v1-StreamMetadata-Type)
    - [TaskMetadata.State](#bytebase-v1-TaskMetadata-State)
  
    - [DatabaseService](#bytebase-v1-DatabaseService)
  
- [v1/environment_service.proto](#v1_environment_service-proto)
    - [CreateEnvironmentRequest](#bytebase-v1-CreateEnvironmentRequest)
    - [DeleteEnvironmentRequest](#bytebase-v1-DeleteEnvironmentRequest)
    - [Environment](#bytebase-v1-Environment)
    - [EnvironmentBackupSetting](#bytebase-v1-EnvironmentBackupSetting)
    - [GetEnvironmentRequest](#bytebase-v1-GetEnvironmentRequest)
    - [ListEnvironmentsRequest](#bytebase-v1-ListEnvironmentsRequest)
    - [ListEnvironmentsResponse](#bytebase-v1-ListEnvironmentsResponse)
    - [UndeleteEnvironmentRequest](#bytebase-v1-UndeleteEnvironmentRequest)
    - [UpdateEnvironmentBackupSettingRequest](#bytebase-v1-UpdateEnvironmentBackupSettingRequest)
    - [UpdateEnvironmentRequest](#bytebase-v1-UpdateEnvironmentRequest)
  
    - [EnvironmentTier](#bytebase-v1-EnvironmentTier)
  
    - [EnvironmentService](#bytebase-v1-EnvironmentService)
  
- [v1/externalvs_service.proto](#v1_externalvs_service-proto)
    - [CreateExternalVersionControlRequest](#bytebase-v1-CreateExternalVersionControlRequest)
    - [DeleteExternalVersionControlRequest](#bytebase-v1-DeleteExternalVersionControlRequest)
    - [ExchangeToken](#bytebase-v1-ExchangeToken)
    - [ExchangeTokenRequest](#bytebase-v1-ExchangeTokenRequest)
    - [ExternalVersionControl](#bytebase-v1-ExternalVersionControl)
    - [GetExternalVersionControlRequest](#bytebase-v1-GetExternalVersionControlRequest)
    - [ListExternalVersionControlsRequest](#bytebase-v1-ListExternalVersionControlsRequest)
    - [ListExternalVersionControlsResponse](#bytebase-v1-ListExternalVersionControlsResponse)
    - [ListProjectGitOpsInfoRequest](#bytebase-v1-ListProjectGitOpsInfoRequest)
    - [ListProjectGitOpsInfoResponse](#bytebase-v1-ListProjectGitOpsInfoResponse)
    - [OAuthToken](#bytebase-v1-OAuthToken)
    - [ProjectGitOpsInfo](#bytebase-v1-ProjectGitOpsInfo)
    - [SearchExternalVersionControlProjectsRequest](#bytebase-v1-SearchExternalVersionControlProjectsRequest)
    - [SearchExternalVersionControlProjectsResponse](#bytebase-v1-SearchExternalVersionControlProjectsResponse)
    - [SearchExternalVersionControlProjectsResponse.Project](#bytebase-v1-SearchExternalVersionControlProjectsResponse-Project)
    - [UpdateExternalVersionControlRequest](#bytebase-v1-UpdateExternalVersionControlRequest)
  
    - [ExternalVersionControl.Type](#bytebase-v1-ExternalVersionControl-Type)
  
    - [ExternalVersionControlService](#bytebase-v1-ExternalVersionControlService)
  
- [v1/idp_service.proto](#v1_idp_service-proto)
    - [CreateIdentityProviderRequest](#bytebase-v1-CreateIdentityProviderRequest)
    - [DeleteIdentityProviderRequest](#bytebase-v1-DeleteIdentityProviderRequest)
    - [FieldMapping](#bytebase-v1-FieldMapping)
    - [GetIdentityProviderRequest](#bytebase-v1-GetIdentityProviderRequest)
    - [IdentityProvider](#bytebase-v1-IdentityProvider)
    - [IdentityProviderConfig](#bytebase-v1-IdentityProviderConfig)
    - [LDAPIdentityProviderConfig](#bytebase-v1-LDAPIdentityProviderConfig)
    - [ListIdentityProvidersRequest](#bytebase-v1-ListIdentityProvidersRequest)
    - [ListIdentityProvidersResponse](#bytebase-v1-ListIdentityProvidersResponse)
    - [OAuth2IdentityProviderConfig](#bytebase-v1-OAuth2IdentityProviderConfig)
    - [OAuth2IdentityProviderTestRequestContext](#bytebase-v1-OAuth2IdentityProviderTestRequestContext)
    - [OIDCIdentityProviderConfig](#bytebase-v1-OIDCIdentityProviderConfig)
    - [TestIdentityProviderRequest](#bytebase-v1-TestIdentityProviderRequest)
    - [TestIdentityProviderResponse](#bytebase-v1-TestIdentityProviderResponse)
    - [UndeleteIdentityProviderRequest](#bytebase-v1-UndeleteIdentityProviderRequest)
    - [UpdateIdentityProviderRequest](#bytebase-v1-UpdateIdentityProviderRequest)
  
    - [IdentityProviderType](#bytebase-v1-IdentityProviderType)
    - [OAuth2AuthStyle](#bytebase-v1-OAuth2AuthStyle)
  
    - [IdentityProviderService](#bytebase-v1-IdentityProviderService)
  
- [v1/logging_service.proto](#v1_logging_service-proto)
    - [GetLogRequest](#bytebase-v1-GetLogRequest)
    - [ListLogsRequest](#bytebase-v1-ListLogsRequest)
    - [ListLogsResponse](#bytebase-v1-ListLogsResponse)
    - [LogEntity](#bytebase-v1-LogEntity)
  
    - [LogEntity.Action](#bytebase-v1-LogEntity-Action)
    - [LogEntity.Level](#bytebase-v1-LogEntity-Level)
  
    - [LoggingService](#bytebase-v1-LoggingService)
  
- [v1/inbox_service.proto](#v1_inbox_service-proto)
    - [GetInboxSummaryRequest](#bytebase-v1-GetInboxSummaryRequest)
    - [InboxMessage](#bytebase-v1-InboxMessage)
    - [InboxSummary](#bytebase-v1-InboxSummary)
    - [ListInboxRequest](#bytebase-v1-ListInboxRequest)
    - [ListInboxResponse](#bytebase-v1-ListInboxResponse)
    - [UpdateInboxRequest](#bytebase-v1-UpdateInboxRequest)
  
    - [InboxMessage.Status](#bytebase-v1-InboxMessage-Status)
  
    - [InboxService](#bytebase-v1-InboxService)
  
- [v1/instance_role_service.proto](#v1_instance_role_service-proto)
    - [CreateInstanceRoleRequest](#bytebase-v1-CreateInstanceRoleRequest)
    - [DeleteInstanceRoleRequest](#bytebase-v1-DeleteInstanceRoleRequest)
    - [GetInstanceRoleRequest](#bytebase-v1-GetInstanceRoleRequest)
    - [InstanceRole](#bytebase-v1-InstanceRole)
    - [ListInstanceRolesRequest](#bytebase-v1-ListInstanceRolesRequest)
    - [ListInstanceRolesResponse](#bytebase-v1-ListInstanceRolesResponse)
    - [UndeleteInstanceRoleRequest](#bytebase-v1-UndeleteInstanceRoleRequest)
    - [UpdateInstanceRoleRequest](#bytebase-v1-UpdateInstanceRoleRequest)
  
    - [InstanceRoleService](#bytebase-v1-InstanceRoleService)
  
- [v1/instance_service.proto](#v1_instance_service-proto)
    - [AddDataSourceRequest](#bytebase-v1-AddDataSourceRequest)
    - [CreateInstanceRequest](#bytebase-v1-CreateInstanceRequest)
    - [DataSource](#bytebase-v1-DataSource)
    - [DeleteInstanceRequest](#bytebase-v1-DeleteInstanceRequest)
    - [GetInstanceRequest](#bytebase-v1-GetInstanceRequest)
    - [Instance](#bytebase-v1-Instance)
    - [InstanceOptions](#bytebase-v1-InstanceOptions)
    - [ListInstancesRequest](#bytebase-v1-ListInstancesRequest)
    - [ListInstancesResponse](#bytebase-v1-ListInstancesResponse)
    - [RemoveDataSourceRequest](#bytebase-v1-RemoveDataSourceRequest)
    - [SyncInstanceRequest](#bytebase-v1-SyncInstanceRequest)
    - [SyncInstanceResponse](#bytebase-v1-SyncInstanceResponse)
    - [SyncSlowQueriesRequest](#bytebase-v1-SyncSlowQueriesRequest)
    - [UndeleteInstanceRequest](#bytebase-v1-UndeleteInstanceRequest)
    - [UpdateDataSourceRequest](#bytebase-v1-UpdateDataSourceRequest)
    - [UpdateInstanceRequest](#bytebase-v1-UpdateInstanceRequest)
  
    - [DataSourceType](#bytebase-v1-DataSourceType)
  
    - [InstanceService](#bytebase-v1-InstanceService)
  
- [v1/issue_service.proto](#v1_issue_service-proto)
    - [ApprovalFlow](#bytebase-v1-ApprovalFlow)
    - [ApprovalNode](#bytebase-v1-ApprovalNode)
    - [ApprovalStep](#bytebase-v1-ApprovalStep)
    - [ApprovalTemplate](#bytebase-v1-ApprovalTemplate)
    - [ApproveIssueRequest](#bytebase-v1-ApproveIssueRequest)
    - [BatchUpdateIssuesStatusRequest](#bytebase-v1-BatchUpdateIssuesStatusRequest)
    - [BatchUpdateIssuesStatusResponse](#bytebase-v1-BatchUpdateIssuesStatusResponse)
    - [CreateIssueCommentRequest](#bytebase-v1-CreateIssueCommentRequest)
    - [CreateIssueRequest](#bytebase-v1-CreateIssueRequest)
    - [GetIssueRequest](#bytebase-v1-GetIssueRequest)
    - [Issue](#bytebase-v1-Issue)
    - [Issue.Approver](#bytebase-v1-Issue-Approver)
    - [IssueComment](#bytebase-v1-IssueComment)
    - [ListIssuesRequest](#bytebase-v1-ListIssuesRequest)
    - [ListIssuesResponse](#bytebase-v1-ListIssuesResponse)
    - [RejectIssueRequest](#bytebase-v1-RejectIssueRequest)
    - [RequestIssueRequest](#bytebase-v1-RequestIssueRequest)
    - [SearchIssuesRequest](#bytebase-v1-SearchIssuesRequest)
    - [SearchIssuesResponse](#bytebase-v1-SearchIssuesResponse)
    - [UpdateIssueCommentRequest](#bytebase-v1-UpdateIssueCommentRequest)
    - [UpdateIssueRequest](#bytebase-v1-UpdateIssueRequest)
  
    - [ApprovalNode.GroupValue](#bytebase-v1-ApprovalNode-GroupValue)
    - [ApprovalNode.Type](#bytebase-v1-ApprovalNode-Type)
    - [ApprovalStep.Type](#bytebase-v1-ApprovalStep-Type)
    - [Issue.Approver.Status](#bytebase-v1-Issue-Approver-Status)
    - [Issue.Type](#bytebase-v1-Issue-Type)
    - [IssueStatus](#bytebase-v1-IssueStatus)
  
    - [IssueService](#bytebase-v1-IssueService)
  
- [v1/project_service.proto](#v1_project_service-proto)
    - [Activity](#bytebase-v1-Activity)
    - [AddWebhookRequest](#bytebase-v1-AddWebhookRequest)
    - [BatchGetIamPolicyRequest](#bytebase-v1-BatchGetIamPolicyRequest)
    - [BatchGetIamPolicyResponse](#bytebase-v1-BatchGetIamPolicyResponse)
    - [BatchGetIamPolicyResponse.PolicyResult](#bytebase-v1-BatchGetIamPolicyResponse-PolicyResult)
    - [CreateDatabaseGroupRequest](#bytebase-v1-CreateDatabaseGroupRequest)
    - [CreateProjectRequest](#bytebase-v1-CreateProjectRequest)
    - [CreateSchemaGroupRequest](#bytebase-v1-CreateSchemaGroupRequest)
    - [DatabaseGroup](#bytebase-v1-DatabaseGroup)
    - [DatabaseGroup.Database](#bytebase-v1-DatabaseGroup-Database)
    - [DeleteDatabaseGroupRequest](#bytebase-v1-DeleteDatabaseGroupRequest)
    - [DeleteProjectRequest](#bytebase-v1-DeleteProjectRequest)
    - [DeleteSchemaGroupRequest](#bytebase-v1-DeleteSchemaGroupRequest)
    - [DeploymentConfig](#bytebase-v1-DeploymentConfig)
    - [DeploymentSpec](#bytebase-v1-DeploymentSpec)
    - [GetDatabaseGroupRequest](#bytebase-v1-GetDatabaseGroupRequest)
    - [GetDeploymentConfigRequest](#bytebase-v1-GetDeploymentConfigRequest)
    - [GetIamPolicyRequest](#bytebase-v1-GetIamPolicyRequest)
    - [GetProjectGitOpsInfoRequest](#bytebase-v1-GetProjectGitOpsInfoRequest)
    - [GetProjectRequest](#bytebase-v1-GetProjectRequest)
    - [GetSchemaGroupRequest](#bytebase-v1-GetSchemaGroupRequest)
    - [LabelSelector](#bytebase-v1-LabelSelector)
    - [LabelSelectorRequirement](#bytebase-v1-LabelSelectorRequirement)
    - [ListDatabaseGroupsRequest](#bytebase-v1-ListDatabaseGroupsRequest)
    - [ListDatabaseGroupsResponse](#bytebase-v1-ListDatabaseGroupsResponse)
    - [ListProjectsRequest](#bytebase-v1-ListProjectsRequest)
    - [ListProjectsResponse](#bytebase-v1-ListProjectsResponse)
    - [ListSchemaGroupsRequest](#bytebase-v1-ListSchemaGroupsRequest)
    - [ListSchemaGroupsResponse](#bytebase-v1-ListSchemaGroupsResponse)
    - [Project](#bytebase-v1-Project)
    - [RemoveWebhookRequest](#bytebase-v1-RemoveWebhookRequest)
    - [Schedule](#bytebase-v1-Schedule)
    - [ScheduleDeployment](#bytebase-v1-ScheduleDeployment)
    - [SchemaGroup](#bytebase-v1-SchemaGroup)
    - [SchemaGroup.Table](#bytebase-v1-SchemaGroup-Table)
    - [SearchProjectsRequest](#bytebase-v1-SearchProjectsRequest)
    - [SearchProjectsResponse](#bytebase-v1-SearchProjectsResponse)
    - [SetIamPolicyRequest](#bytebase-v1-SetIamPolicyRequest)
    - [SetupSQLReviewCIRequest](#bytebase-v1-SetupSQLReviewCIRequest)
    - [SetupSQLReviewCIResponse](#bytebase-v1-SetupSQLReviewCIResponse)
    - [TestWebhookRequest](#bytebase-v1-TestWebhookRequest)
    - [TestWebhookResponse](#bytebase-v1-TestWebhookResponse)
    - [UndeleteProjectRequest](#bytebase-v1-UndeleteProjectRequest)
    - [UnsetProjectGitOpsInfoRequest](#bytebase-v1-UnsetProjectGitOpsInfoRequest)
    - [UpdateDatabaseGroupRequest](#bytebase-v1-UpdateDatabaseGroupRequest)
    - [UpdateDeploymentConfigRequest](#bytebase-v1-UpdateDeploymentConfigRequest)
    - [UpdateProjectGitOpsInfoRequest](#bytebase-v1-UpdateProjectGitOpsInfoRequest)
    - [UpdateProjectRequest](#bytebase-v1-UpdateProjectRequest)
    - [UpdateSchemaGroupRequest](#bytebase-v1-UpdateSchemaGroupRequest)
    - [UpdateWebhookRequest](#bytebase-v1-UpdateWebhookRequest)
    - [Webhook](#bytebase-v1-Webhook)
  
    - [Activity.Type](#bytebase-v1-Activity-Type)
    - [DatabaseGroupView](#bytebase-v1-DatabaseGroupView)
    - [OperatorType](#bytebase-v1-OperatorType)
    - [SchemaChange](#bytebase-v1-SchemaChange)
    - [SchemaGroupView](#bytebase-v1-SchemaGroupView)
    - [SchemaVersion](#bytebase-v1-SchemaVersion)
    - [TenantMode](#bytebase-v1-TenantMode)
    - [Visibility](#bytebase-v1-Visibility)
    - [Webhook.Type](#bytebase-v1-Webhook-Type)
    - [Workflow](#bytebase-v1-Workflow)
  
    - [ProjectService](#bytebase-v1-ProjectService)
  
- [v1/risk_service.proto](#v1_risk_service-proto)
    - [CreateRiskRequest](#bytebase-v1-CreateRiskRequest)
    - [DeleteRiskRequest](#bytebase-v1-DeleteRiskRequest)
    - [ListRisksRequest](#bytebase-v1-ListRisksRequest)
    - [ListRisksResponse](#bytebase-v1-ListRisksResponse)
    - [Risk](#bytebase-v1-Risk)
    - [UpdateRiskRequest](#bytebase-v1-UpdateRiskRequest)
  
    - [Risk.Source](#bytebase-v1-Risk-Source)
  
    - [RiskService](#bytebase-v1-RiskService)
  
- [v1/role_service.proto](#v1_role_service-proto)
    - [CreateRoleRequest](#bytebase-v1-CreateRoleRequest)
    - [DeleteRoleRequest](#bytebase-v1-DeleteRoleRequest)
    - [ListRolesRequest](#bytebase-v1-ListRolesRequest)
    - [ListRolesResponse](#bytebase-v1-ListRolesResponse)
    - [Role](#bytebase-v1-Role)
    - [UpdateRoleRequest](#bytebase-v1-UpdateRoleRequest)
  
    - [RoleService](#bytebase-v1-RoleService)
  
- [v1/rollout_service.proto](#v1_rollout_service-proto)
    - [BatchCancelTaskRunsRequest](#bytebase-v1-BatchCancelTaskRunsRequest)
    - [BatchCancelTaskRunsResponse](#bytebase-v1-BatchCancelTaskRunsResponse)
    - [BatchRunTasksRequest](#bytebase-v1-BatchRunTasksRequest)
    - [BatchRunTasksResponse](#bytebase-v1-BatchRunTasksResponse)
    - [BatchSkipTasksRequest](#bytebase-v1-BatchSkipTasksRequest)
    - [BatchSkipTasksResponse](#bytebase-v1-BatchSkipTasksResponse)
    - [CreatePlanRequest](#bytebase-v1-CreatePlanRequest)
    - [CreateRolloutRequest](#bytebase-v1-CreateRolloutRequest)
    - [GetPlanRequest](#bytebase-v1-GetPlanRequest)
    - [GetRolloutRequest](#bytebase-v1-GetRolloutRequest)
    - [ListPlanCheckRunsRequest](#bytebase-v1-ListPlanCheckRunsRequest)
    - [ListPlanCheckRunsResponse](#bytebase-v1-ListPlanCheckRunsResponse)
    - [ListPlansRequest](#bytebase-v1-ListPlansRequest)
    - [ListPlansResponse](#bytebase-v1-ListPlansResponse)
    - [ListTaskRunsRequest](#bytebase-v1-ListTaskRunsRequest)
    - [ListTaskRunsResponse](#bytebase-v1-ListTaskRunsResponse)
    - [Plan](#bytebase-v1-Plan)
    - [Plan.ChangeDatabaseConfig](#bytebase-v1-Plan-ChangeDatabaseConfig)
    - [Plan.ChangeDatabaseConfig.RollbackDetail](#bytebase-v1-Plan-ChangeDatabaseConfig-RollbackDetail)
    - [Plan.CreateDatabaseConfig](#bytebase-v1-Plan-CreateDatabaseConfig)
    - [Plan.CreateDatabaseConfig.LabelsEntry](#bytebase-v1-Plan-CreateDatabaseConfig-LabelsEntry)
    - [Plan.RestoreDatabaseConfig](#bytebase-v1-Plan-RestoreDatabaseConfig)
    - [Plan.Spec](#bytebase-v1-Plan-Spec)
    - [Plan.Step](#bytebase-v1-Plan-Step)
    - [PlanCheckRun](#bytebase-v1-PlanCheckRun)
    - [PlanCheckRun.Result](#bytebase-v1-PlanCheckRun-Result)
    - [PlanCheckRun.Result.SqlReviewReport](#bytebase-v1-PlanCheckRun-Result-SqlReviewReport)
    - [PlanCheckRun.Result.SqlSummaryReport](#bytebase-v1-PlanCheckRun-Result-SqlSummaryReport)
    - [PreviewRolloutRequest](#bytebase-v1-PreviewRolloutRequest)
    - [Rollout](#bytebase-v1-Rollout)
    - [RunPlanChecksRequest](#bytebase-v1-RunPlanChecksRequest)
    - [RunPlanChecksResponse](#bytebase-v1-RunPlanChecksResponse)
    - [Stage](#bytebase-v1-Stage)
    - [Task](#bytebase-v1-Task)
    - [Task.DatabaseBackup](#bytebase-v1-Task-DatabaseBackup)
    - [Task.DatabaseCreate](#bytebase-v1-Task-DatabaseCreate)
    - [Task.DatabaseCreate.LabelsEntry](#bytebase-v1-Task-DatabaseCreate-LabelsEntry)
    - [Task.DatabaseDataUpdate](#bytebase-v1-Task-DatabaseDataUpdate)
    - [Task.DatabaseRestoreRestore](#bytebase-v1-Task-DatabaseRestoreRestore)
    - [Task.DatabaseSchemaBaseline](#bytebase-v1-Task-DatabaseSchemaBaseline)
    - [Task.DatabaseSchemaUpdate](#bytebase-v1-Task-DatabaseSchemaUpdate)
    - [TaskRun](#bytebase-v1-TaskRun)
    - [UpdatePlanRequest](#bytebase-v1-UpdatePlanRequest)
  
    - [Plan.ChangeDatabaseConfig.Type](#bytebase-v1-Plan-ChangeDatabaseConfig-Type)
    - [PlanCheckRun.Result.Status](#bytebase-v1-PlanCheckRun-Result-Status)
    - [PlanCheckRun.Status](#bytebase-v1-PlanCheckRun-Status)
    - [PlanCheckRun.Type](#bytebase-v1-PlanCheckRun-Type)
    - [Task.DatabaseDataUpdate.RollbackSqlStatus](#bytebase-v1-Task-DatabaseDataUpdate-RollbackSqlStatus)
    - [Task.Status](#bytebase-v1-Task-Status)
    - [Task.Type](#bytebase-v1-Task-Type)
    - [TaskRun.Status](#bytebase-v1-TaskRun-Status)
  
    - [RolloutService](#bytebase-v1-RolloutService)
  
- [v1/schema_design_service.proto](#v1_schema_design_service-proto)
    - [CreateSchemaDesignRequest](#bytebase-v1-CreateSchemaDesignRequest)
    - [DeleteSchemaDesignRequest](#bytebase-v1-DeleteSchemaDesignRequest)
    - [GetSchemaDesignRequest](#bytebase-v1-GetSchemaDesignRequest)
    - [ListSchemaDesignsRequest](#bytebase-v1-ListSchemaDesignsRequest)
    - [ListSchemaDesignsResponse](#bytebase-v1-ListSchemaDesignsResponse)
    - [MergeSchemaDesignRequest](#bytebase-v1-MergeSchemaDesignRequest)
    - [ParseSchemaStringRequest](#bytebase-v1-ParseSchemaStringRequest)
    - [ParseSchemaStringResponse](#bytebase-v1-ParseSchemaStringResponse)
    - [SchemaDesign](#bytebase-v1-SchemaDesign)
    - [UpdateSchemaDesignRequest](#bytebase-v1-UpdateSchemaDesignRequest)
  
    - [SchemaDesign.Type](#bytebase-v1-SchemaDesign-Type)
  
    - [SchemaDesignService](#bytebase-v1-SchemaDesignService)
  
- [v1/subscription_service.proto](#v1_subscription_service-proto)
    - [Feature](#bytebase-v1-Feature)
    - [Feature.MatrixEntry](#bytebase-v1-Feature-MatrixEntry)
    - [FeatureMatrix](#bytebase-v1-FeatureMatrix)
    - [GetFeatureMatrixRequest](#bytebase-v1-GetFeatureMatrixRequest)
    - [GetSubscriptionRequest](#bytebase-v1-GetSubscriptionRequest)
    - [PatchSubscription](#bytebase-v1-PatchSubscription)
    - [Subscription](#bytebase-v1-Subscription)
    - [TrialSubscription](#bytebase-v1-TrialSubscription)
    - [TrialSubscriptionRequest](#bytebase-v1-TrialSubscriptionRequest)
    - [UpdateSubscriptionRequest](#bytebase-v1-UpdateSubscriptionRequest)
  
    - [PlanType](#bytebase-v1-PlanType)
  
    - [SubscriptionService](#bytebase-v1-SubscriptionService)
  
- [v1/setting_service.proto](#v1_setting_service-proto)
    - [AgentPluginSetting](#bytebase-v1-AgentPluginSetting)
    - [AppIMSetting](#bytebase-v1-AppIMSetting)
    - [AppIMSetting.ExternalApproval](#bytebase-v1-AppIMSetting-ExternalApproval)
    - [DataClassificationSetting](#bytebase-v1-DataClassificationSetting)
    - [DataClassificationSetting.DataClassificationConfig](#bytebase-v1-DataClassificationSetting-DataClassificationConfig)
    - [DataClassificationSetting.DataClassificationConfig.ClassificationEntry](#bytebase-v1-DataClassificationSetting-DataClassificationConfig-ClassificationEntry)
    - [DataClassificationSetting.DataClassificationConfig.DataClassification](#bytebase-v1-DataClassificationSetting-DataClassificationConfig-DataClassification)
    - [DataClassificationSetting.DataClassificationConfig.Level](#bytebase-v1-DataClassificationSetting-DataClassificationConfig-Level)
    - [ExternalApprovalSetting](#bytebase-v1-ExternalApprovalSetting)
    - [ExternalApprovalSetting.Node](#bytebase-v1-ExternalApprovalSetting-Node)
    - [GetSettingRequest](#bytebase-v1-GetSettingRequest)
    - [GetSettingResponse](#bytebase-v1-GetSettingResponse)
    - [ListSettingsRequest](#bytebase-v1-ListSettingsRequest)
    - [ListSettingsResponse](#bytebase-v1-ListSettingsResponse)
    - [SMTPMailDeliverySettingValue](#bytebase-v1-SMTPMailDeliverySettingValue)
    - [SchemaTemplateSetting](#bytebase-v1-SchemaTemplateSetting)
    - [SchemaTemplateSetting.ColumnType](#bytebase-v1-SchemaTemplateSetting-ColumnType)
    - [SchemaTemplateSetting.FieldTemplate](#bytebase-v1-SchemaTemplateSetting-FieldTemplate)
    - [SemanticCategorySetting](#bytebase-v1-SemanticCategorySetting)
    - [SemanticCategorySetting.SemanticCategory](#bytebase-v1-SemanticCategorySetting-SemanticCategory)
    - [SetSettingRequest](#bytebase-v1-SetSettingRequest)
    - [Setting](#bytebase-v1-Setting)
    - [Value](#bytebase-v1-Value)
    - [WorkspaceApprovalSetting](#bytebase-v1-WorkspaceApprovalSetting)
    - [WorkspaceApprovalSetting.Rule](#bytebase-v1-WorkspaceApprovalSetting-Rule)
    - [WorkspaceProfileSetting](#bytebase-v1-WorkspaceProfileSetting)
    - [WorkspaceTrialSetting](#bytebase-v1-WorkspaceTrialSetting)
  
    - [AppIMSetting.IMType](#bytebase-v1-AppIMSetting-IMType)
    - [SMTPMailDeliverySettingValue.Authentication](#bytebase-v1-SMTPMailDeliverySettingValue-Authentication)
    - [SMTPMailDeliverySettingValue.Encryption](#bytebase-v1-SMTPMailDeliverySettingValue-Encryption)
  
    - [SettingService](#bytebase-v1-SettingService)
  
- [v1/sheet_service.proto](#v1_sheet_service-proto)
    - [CreateSheetRequest](#bytebase-v1-CreateSheetRequest)
    - [DeleteSheetRequest](#bytebase-v1-DeleteSheetRequest)
    - [GetSheetRequest](#bytebase-v1-GetSheetRequest)
    - [SearchSheetsRequest](#bytebase-v1-SearchSheetsRequest)
    - [SearchSheetsResponse](#bytebase-v1-SearchSheetsResponse)
    - [Sheet](#bytebase-v1-Sheet)
    - [SheetOrganizer](#bytebase-v1-SheetOrganizer)
    - [SyncSheetsRequest](#bytebase-v1-SyncSheetsRequest)
    - [UpdateSheetOrganizerRequest](#bytebase-v1-UpdateSheetOrganizerRequest)
    - [UpdateSheetRequest](#bytebase-v1-UpdateSheetRequest)
  
    - [Sheet.Source](#bytebase-v1-Sheet-Source)
    - [Sheet.Type](#bytebase-v1-Sheet-Type)
    - [Sheet.Visibility](#bytebase-v1-Sheet-Visibility)
  
    - [SheetService](#bytebase-v1-SheetService)
  
- [v1/sql_service.proto](#v1_sql_service-proto)
    - [AdminExecuteRequest](#bytebase-v1-AdminExecuteRequest)
    - [AdminExecuteResponse](#bytebase-v1-AdminExecuteResponse)
    - [Advice](#bytebase-v1-Advice)
    - [DifferPreviewRequest](#bytebase-v1-DifferPreviewRequest)
    - [DifferPreviewResponse](#bytebase-v1-DifferPreviewResponse)
    - [ExportRequest](#bytebase-v1-ExportRequest)
    - [ExportResponse](#bytebase-v1-ExportResponse)
    - [PrettyRequest](#bytebase-v1-PrettyRequest)
    - [PrettyResponse](#bytebase-v1-PrettyResponse)
    - [QueryRequest](#bytebase-v1-QueryRequest)
    - [QueryResponse](#bytebase-v1-QueryResponse)
    - [QueryResult](#bytebase-v1-QueryResult)
    - [QueryRow](#bytebase-v1-QueryRow)
    - [RowValue](#bytebase-v1-RowValue)
  
    - [Advice.Status](#bytebase-v1-Advice-Status)
    - [ExportRequest.Format](#bytebase-v1-ExportRequest-Format)
  
    - [SQLService](#bytebase-v1-SQLService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="v1_actuator_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/actuator_service.proto



<a name="bytebase-v1-ActuatorInfo"></a>

### ActuatorInfo
ServerInfo is the API message for server info.
Actuator concept is similar to the Spring Boot Actuator.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [string](#string) |  | version is the bytebase&#39;s server version |
| git_commit | [string](#string) |  | git_commit is the git commit hash of the build |
| readonly | [bool](#bool) |  | readonly flag means if the Bytebase is running in readonly mode. |
| saas | [bool](#bool) |  | saas flag means if the Bytebase is running in SaaS mode, some features are not allowed to edit by users. |
| demo_name | [string](#string) |  | demo_name specifies the demo name, empty string means no demo. |
| host | [string](#string) |  | host is the Bytebase instance host. |
| port | [string](#string) |  | port is the Bytebase instance port. |
| external_url | [string](#string) |  | external_url is the URL where user or webhook callback visits Bytebase. |
| need_admin_setup | [bool](#bool) |  | need_admin_setup flag means the Bytebase instance doesn&#39;t have any end users. |
| disallow_signup | [bool](#bool) |  | disallow_signup is the flag to disable self-service signup. |
| last_active_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | last_active_time is the service last active time in UTC Time Format, any API calls will refresh this value. |
| require_2fa | [bool](#bool) |  | require_2fa is the flag to require 2FA for all users. |
| workspace_id | [string](#string) |  | workspace_id is the identifier for the workspace. |
| gitops_webhook_url | [string](#string) |  | gitops_webhook_url is the webhook URL for GitOps. |
| debug | [bool](#bool) |  | debug flag means if the debug mode is enabled. |
| development_use_v2_scheduler | [bool](#bool) |  | development_use_v2_scheduler flag means if the server uses the v2 task run scheduler. this flag is only used for development purpose and will be removed once we switch to the v2 scheduler. |






<a name="bytebase-v1-DebugLog"></a>

### DebugLog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| record_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| request_path | [string](#string) |  |  |
| role | [string](#string) |  |  |
| error | [string](#string) |  |  |
| stack_trace | [string](#string) |  |  |






<a name="bytebase-v1-DeleteCacheRequest"></a>

### DeleteCacheRequest







<a name="bytebase-v1-GetActuatorInfoRequest"></a>

### GetActuatorInfoRequest







<a name="bytebase-v1-ListDebugLogRequest"></a>

### ListDebugLogRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of logs to return. The service may return fewer than this value. If unspecified, at most 50 logs will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListDebugLog` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListDebugLog` must match the call that provided the page token. |






<a name="bytebase-v1-ListDebugLogResponse"></a>

### ListDebugLogResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| logs | [DebugLog](#bytebase-v1-DebugLog) | repeated | The logs from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-UpdateActuatorInfoRequest"></a>

### UpdateActuatorInfoRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| actuator | [ActuatorInfo](#bytebase-v1-ActuatorInfo) |  | The actuator to update. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 

 

 


<a name="bytebase-v1-ActuatorService"></a>

### ActuatorService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetActuatorInfo | [GetActuatorInfoRequest](#bytebase-v1-GetActuatorInfoRequest) | [ActuatorInfo](#bytebase-v1-ActuatorInfo) |  |
| UpdateActuatorInfo | [UpdateActuatorInfoRequest](#bytebase-v1-UpdateActuatorInfoRequest) | [ActuatorInfo](#bytebase-v1-ActuatorInfo) |  |
| DeleteCache | [DeleteCacheRequest](#bytebase-v1-DeleteCacheRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| ListDebugLog | [ListDebugLogRequest](#bytebase-v1-ListDebugLogRequest) | [ListDebugLogResponse](#bytebase-v1-ListDebugLogResponse) |  |

 



<a name="v1_common-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/common.proto


 


<a name="bytebase-v1-Engine"></a>

### Engine


| Name | Number | Description |
| ---- | ------ | ----------- |
| ENGINE_UNSPECIFIED | 0 |  |
| CLICKHOUSE | 1 |  |
| MYSQL | 2 |  |
| POSTGRES | 3 |  |
| SNOWFLAKE | 4 |  |
| SQLITE | 5 |  |
| TIDB | 6 |  |
| MONGODB | 7 |  |
| REDIS | 8 |  |
| ORACLE | 9 |  |
| SPANNER | 10 |  |
| MSSQL | 11 |  |
| REDSHIFT | 12 |  |
| MARIADB | 13 |  |
| OCEANBASE | 14 |  |
| DM | 15 |  |
| RISINGWAVE | 16 |  |



<a name="bytebase-v1-MaskingLevel"></a>

### MaskingLevel


| Name | Number | Description |
| ---- | ------ | ----------- |
| MASKING_LEVEL_UNSPECIFIED | 0 |  |
| NONE | 1 |  |
| PARTIAL | 2 |  |
| FULL | 3 |  |



<a name="bytebase-v1-State"></a>

### State


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATE_UNSPECIFIED | 0 |  |
| ACTIVE | 1 |  |
| DELETED | 2 |  |


 

 

 



<a name="v1_deployment-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/deployment.proto


 


<a name="bytebase-v1-DeploymentType"></a>

### DeploymentType


| Name | Number | Description |
| ---- | ------ | ----------- |
| DEPLOYMENT_TYPE_UNSPECIFIED | 0 |  |
| DATABASE_CREATE | 1 |  |
| DATABASE_DDL | 2 |  |
| DATABASE_DDL_GHOST | 3 |  |
| DATABASE_DML | 4 |  |
| DATABASE_RESTORE_PITR | 5 |  |


 

 

 



<a name="v1_iam_policy-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/iam_policy.proto



<a name="bytebase-v1-Binding"></a>

### Binding



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [string](#string) |  | The role that is assigned to the members. Format: roles/{role} |
| members | [string](#string) | repeated | Specifies the principals requesting access for a Bytebase resource. |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this binding. If the condition evaluates to true, then this binding applies to the current request. If the condition evaluates to false, then this binding does not apply to the current request. However, a different role binding might grant the same role to one or more of the principals in this binding. |
| parsed_expr | [google.api.expr.v1alpha1.ParsedExpr](#google-api-expr-v1alpha1-ParsedExpr) |  | The parsed expression of the condition. |






<a name="bytebase-v1-IamPolicy"></a>

### IamPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bindings | [Binding](#bytebase-v1-Binding) | repeated | Collection of binding. A binding binds one or more project members to a single project role. |





 

 

 

 



<a name="v1_org_policy_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/org_policy_service.proto



<a name="bytebase-v1-BackupPlanPolicy"></a>

### BackupPlanPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schedule | [BackupPlanSchedule](#bytebase-v1-BackupPlanSchedule) |  |  |
| retention_duration | [google.protobuf.Duration](#google-protobuf-Duration) |  |  |






<a name="bytebase-v1-CreatePolicyRequest"></a>

### CreatePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this instance will be created. Workspace resource name: &#34;&#34;. Environment resource name: environments/environment-id. Instance resource name: instances/instance-id. Database resource name: instances/instance-id/databases/database-name. |
| policy | [Policy](#bytebase-v1-Policy) |  | The policy to create. |
| type | [PolicyType](#bytebase-v1-PolicyType) |  |  |






<a name="bytebase-v1-DeletePolicyRequest"></a>

### DeletePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The policy&#39;s `name` field is used to identify the instance to update. Format: {resource name}/policies/{policy type} Workspace resource name: &#34;&#34;. Environment resource name: environments/environment-id. Instance resource name: instances/instance-id. Database resource name: instances/instance-id/databases/database-name. |






<a name="bytebase-v1-DeploymentApprovalPolicy"></a>

### DeploymentApprovalPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| default_strategy | [ApprovalStrategy](#bytebase-v1-ApprovalStrategy) |  |  |
| deployment_approval_strategies | [DeploymentApprovalStrategy](#bytebase-v1-DeploymentApprovalStrategy) | repeated |  |






<a name="bytebase-v1-DeploymentApprovalStrategy"></a>

### DeploymentApprovalStrategy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| deployment_type | [DeploymentType](#bytebase-v1-DeploymentType) |  |  |
| approval_group | [ApprovalGroup](#bytebase-v1-ApprovalGroup) |  |  |
| approval_strategy | [ApprovalStrategy](#bytebase-v1-ApprovalStrategy) |  |  |






<a name="bytebase-v1-DisableCopyDataPolicy"></a>

### DisableCopyDataPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| active | [bool](#bool) |  |  |






<a name="bytebase-v1-GetPolicyRequest"></a>

### GetPolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the policy to retrieve. Format: {resource type}/{resource id}/policies/{policy type} |






<a name="bytebase-v1-ListPoliciesRequest"></a>

### ListPoliciesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of policies. Format: {resource type}/{resource id}/policies/{policy type} |
| policy_type | [PolicyType](#bytebase-v1-PolicyType) | optional |  |
| page_size | [int32](#int32) |  | The maximum number of policies to return. The service may return fewer than this value. If unspecified, at most 50 policies will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `GetPolicies` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `GetPolicies` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Show deleted policies if specified. |






<a name="bytebase-v1-ListPoliciesResponse"></a>

### ListPoliciesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policies | [Policy](#bytebase-v1-Policy) | repeated | The policies from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-MaskData"></a>

### MaskData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  |  |
| table | [string](#string) |  |  |
| column | [string](#string) |  |  |
| semantic_category_id | [string](#string) |  |  |
| masking_level | [MaskingLevel](#bytebase-v1-MaskingLevel) |  |  |






<a name="bytebase-v1-MaskingExceptionPolicy"></a>

### MaskingExceptionPolicy
MaskingExceptionPolicy is the allowlist of users who can access sensitive data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| masking_exceptions | [MaskingExceptionPolicy.MaskingException](#bytebase-v1-MaskingExceptionPolicy-MaskingException) | repeated |  |






<a name="bytebase-v1-MaskingExceptionPolicy-MaskingException"></a>

### MaskingExceptionPolicy.MaskingException



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| action | [MaskingExceptionPolicy.MaskingException.Action](#bytebase-v1-MaskingExceptionPolicy-MaskingException-Action) |  | action is the action that the user can access sensitive data. |
| masking_level | [MaskingLevel](#bytebase-v1-MaskingLevel) |  | Level is the masking level that the user can access sensitive data. |
| members | [string](#string) | repeated | Members is the list of principals who bind to this exception policy instance.

* `user:{emailid}`: An email address that represents a specific Bytebase account. For example, `alice@example.com`. |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this exception policy instance. |






<a name="bytebase-v1-MaskingPolicy"></a>

### MaskingPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| mask_data | [MaskData](#bytebase-v1-MaskData) | repeated |  |






<a name="bytebase-v1-MaskingRulePolicy"></a>

### MaskingRulePolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rules | [MaskingRulePolicy.MaskingRule](#bytebase-v1-MaskingRulePolicy-MaskingRule) | repeated |  |






<a name="bytebase-v1-MaskingRulePolicy-MaskingRule"></a>

### MaskingRulePolicy.MaskingRule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | A unique identifier for a node in UUID format. |
| condition | [google.type.Expr](#google-type-Expr) |  |  |
| masking_level | [MaskingLevel](#bytebase-v1-MaskingLevel) |  |  |






<a name="bytebase-v1-Policy"></a>

### Policy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the policy. Format: {resource name}/policies/{policy type} Workspace resource name: &#34;&#34;. Environment resource name: environments/environment-id. Instance resource name: instances/instance-id. Database resource name: instances/instance-id/databases/database-name. |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| inherit_from_parent | [bool](#bool) |  |  |
| type | [PolicyType](#bytebase-v1-PolicyType) |  |  |
| workspace_iam_policy | [IamPolicy](#bytebase-v1-IamPolicy) |  |  |
| deployment_approval_policy | [DeploymentApprovalPolicy](#bytebase-v1-DeploymentApprovalPolicy) |  |  |
| backup_plan_policy | [BackupPlanPolicy](#bytebase-v1-BackupPlanPolicy) |  |  |
| masking_policy | [MaskingPolicy](#bytebase-v1-MaskingPolicy) |  |  |
| sql_review_policy | [SQLReviewPolicy](#bytebase-v1-SQLReviewPolicy) |  |  |
| slow_query_policy | [SlowQueryPolicy](#bytebase-v1-SlowQueryPolicy) |  |  |
| disable_copy_data_policy | [DisableCopyDataPolicy](#bytebase-v1-DisableCopyDataPolicy) |  |  |
| masking_rule_policy | [MaskingRulePolicy](#bytebase-v1-MaskingRulePolicy) |  |  |
| masking_exception_policy | [MaskingExceptionPolicy](#bytebase-v1-MaskingExceptionPolicy) |  |  |
| enforce | [bool](#bool) |  |  |
| resource_type | [PolicyResourceType](#bytebase-v1-PolicyResourceType) |  | The resource type for the policy. |
| resource_uid | [string](#string) |  | The system-assigned, unique identifier for the resource. |






<a name="bytebase-v1-SQLReviewPolicy"></a>

### SQLReviewPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| rules | [SQLReviewRule](#bytebase-v1-SQLReviewRule) | repeated |  |






<a name="bytebase-v1-SQLReviewRule"></a>

### SQLReviewRule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  |  |
| level | [SQLReviewRuleLevel](#bytebase-v1-SQLReviewRuleLevel) |  |  |
| payload | [string](#string) |  |  |
| engine | [Engine](#bytebase-v1-Engine) |  |  |
| comment | [string](#string) |  |  |






<a name="bytebase-v1-SlowQueryPolicy"></a>

### SlowQueryPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| active | [bool](#bool) |  |  |






<a name="bytebase-v1-UpdatePolicyRequest"></a>

### UpdatePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [Policy](#bytebase-v1-Policy) |  | The policy to update.

The policy&#39;s `name` field is used to identify the instance to update. Format: {resource name}/policies/{policy type} Workspace resource name: &#34;&#34;. Environment resource name: environments/environment-id. Instance resource name: instances/instance-id. Database resource name: instances/instance-id/databases/database-name. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the policy is not found, a new policy will be created. In this situation, `update_mask` is ignored. |





 


<a name="bytebase-v1-ApprovalGroup"></a>

### ApprovalGroup


| Name | Number | Description |
| ---- | ------ | ----------- |
| ASSIGNEE_GROUP_UNSPECIFIED | 0 |  |
| APPROVAL_GROUP_DBA | 1 |  |
| APPROVAL_GROUP_PROJECT_OWNER | 2 |  |



<a name="bytebase-v1-ApprovalStrategy"></a>

### ApprovalStrategy


| Name | Number | Description |
| ---- | ------ | ----------- |
| APPROVAL_STRATEGY_UNSPECIFIED | 0 |  |
| AUTOMATIC | 1 |  |
| MANUAL | 2 |  |



<a name="bytebase-v1-BackupPlanSchedule"></a>

### BackupPlanSchedule


| Name | Number | Description |
| ---- | ------ | ----------- |
| SCHEDULE_UNSPECIFIED | 0 |  |
| UNSET | 1 |  |
| DAILY | 2 |  |
| WEEKLY | 3 |  |



<a name="bytebase-v1-MaskingExceptionPolicy-MaskingException-Action"></a>

### MaskingExceptionPolicy.MaskingException.Action


| Name | Number | Description |
| ---- | ------ | ----------- |
| ACTION_UNSPECIFIED | 0 |  |
| QUERY | 1 |  |
| EXPORT | 2 |  |



<a name="bytebase-v1-PolicyResourceType"></a>

### PolicyResourceType


| Name | Number | Description |
| ---- | ------ | ----------- |
| RESOURCE_TYPE_UNSPECIFIED | 0 |  |
| WORKSPACE | 1 |  |
| ENVIRONMENT | 2 |  |
| PROJECT | 3 |  |
| INSTANCE | 4 |  |
| DATABASE | 5 |  |



<a name="bytebase-v1-PolicyType"></a>

### PolicyType


| Name | Number | Description |
| ---- | ------ | ----------- |
| POLICY_TYPE_UNSPECIFIED | 0 |  |
| WORKSPACE_IAM | 1 |  |
| DEPLOYMENT_APPROVAL | 2 |  |
| BACKUP_PLAN | 3 |  |
| SQL_REVIEW | 4 |  |
| MASKING | 5 |  |
| SLOW_QUERY | 7 |  |
| DISABLE_COPY_DATA | 8 |  |
| MASKING_RULE | 9 |  |
| MASKING_EXCEPTION | 10 |  |



<a name="bytebase-v1-SQLReviewRuleLevel"></a>

### SQLReviewRuleLevel


| Name | Number | Description |
| ---- | ------ | ----------- |
| LEVEL_UNSPECIFIED | 0 |  |
| ERROR | 1 |  |
| WARNING | 2 |  |
| DISABLED | 3 |  |


 

 


<a name="bytebase-v1-OrgPolicyService"></a>

### OrgPolicyService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetPolicy | [GetPolicyRequest](#bytebase-v1-GetPolicyRequest) | [Policy](#bytebase-v1-Policy) |  |
| ListPolicies | [ListPoliciesRequest](#bytebase-v1-ListPoliciesRequest) | [ListPoliciesResponse](#bytebase-v1-ListPoliciesResponse) |  |
| CreatePolicy | [CreatePolicyRequest](#bytebase-v1-CreatePolicyRequest) | [Policy](#bytebase-v1-Policy) |  |
| UpdatePolicy | [UpdatePolicyRequest](#bytebase-v1-UpdatePolicyRequest) | [Policy](#bytebase-v1-Policy) |  |
| DeletePolicy | [DeletePolicyRequest](#bytebase-v1-DeletePolicyRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="v1_anomaly_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/anomaly_service.proto



<a name="bytebase-v1-Anomaly"></a>

### Anomaly



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource | [string](#string) |  | The resource that is the target of the operation. Format: - Instance: instnaces/{instance} - Database: instnaces/{instance}/databases/{database} |
| type | [Anomaly.AnomalyType](#bytebase-v1-Anomaly-AnomalyType) |  | type is the type of the anomaly. |
| severity | [Anomaly.AnomalySeverity](#bytebase-v1-Anomaly-AnomalySeverity) |  | severity is the severity of the anomaly. |
| instance_connection_detail | [Anomaly.InstanceConnectionDetail](#bytebase-v1-Anomaly-InstanceConnectionDetail) |  |  |
| database_connection_detail | [Anomaly.DatabaseConnectionDetail](#bytebase-v1-Anomaly-DatabaseConnectionDetail) |  |  |
| database_backup_policy_violation_detail | [Anomaly.DatabaseBackupPolicyViolationDetail](#bytebase-v1-Anomaly-DatabaseBackupPolicyViolationDetail) |  |  |
| database_backup_missing_detail | [Anomaly.DatabaseBackupMissingDetail](#bytebase-v1-Anomaly-DatabaseBackupMissingDetail) |  |  |
| database_schema_drift_detail | [Anomaly.DatabaseSchemaDriftDetail](#bytebase-v1-Anomaly-DatabaseSchemaDriftDetail) |  |  |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="bytebase-v1-Anomaly-DatabaseBackupMissingDetail"></a>

### Anomaly.DatabaseBackupMissingDetail
DatabaseBackupMissingDetail is the detail for database backup missing anomaly.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expected_schedule | [BackupPlanSchedule](#bytebase-v1-BackupPlanSchedule) |  | expected_schedule is the expected backup plan schedule in the database. |
| latest_backup_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | latest_backup_time is the latest backup time in the database. |






<a name="bytebase-v1-Anomaly-DatabaseBackupPolicyViolationDetail"></a>

### Anomaly.DatabaseBackupPolicyViolationDetail
DatabaseBackupPolicyViolationDetail is the detail for database backup policy violation anomaly.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | parent is the parent of the database. Format: environments/{environment} |
| expected_schedule | [BackupPlanSchedule](#bytebase-v1-BackupPlanSchedule) |  | expected_schedule is the expected backup plan schedule in the parent. |
| actual_schedule | [BackupPlanSchedule](#bytebase-v1-BackupPlanSchedule) |  | actual_schedule is the actual backup plan schedule in the database. |






<a name="bytebase-v1-Anomaly-DatabaseConnectionDetail"></a>

### Anomaly.DatabaseConnectionDetail
Database level anomaly detial.

DatbaaseConnectionDetail is the detail for database connection anomaly.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| detail | [string](#string) |  | detail is the detail of the database connection failure. |






<a name="bytebase-v1-Anomaly-DatabaseSchemaDriftDetail"></a>

### Anomaly.DatabaseSchemaDriftDetail
DatabaseSchemaDriftDetail is the detail for database schema drift anomaly.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| record_version | [string](#string) |  | record_version is the record version of the database schema drift. |
| expected_schema | [string](#string) |  | expected_schema is the expected schema in the database. |
| actual_schema | [string](#string) |  | actual_schema is the actual schema in the database. |






<a name="bytebase-v1-Anomaly-InstanceConnectionDetail"></a>

### Anomaly.InstanceConnectionDetail
Instance level anomaly detail.

InstanceConnectionDetail is the detail for instance connection anomaly.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| detail | [string](#string) |  | detail is the detail of the instance connection failure. |






<a name="bytebase-v1-SearchAnomaliesRequest"></a>

### SearchAnomaliesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filter | [string](#string) |  | filter is the filter to apply on the search anomaly request, follow the [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) syntax. Only support filter by resource and type for now. For example: Search the anomalies of a specific resource: &#39;resource=&#34;instances/{instance}&#34;.&#39; Search the specified types of anomalies: &#39;type=&#34;DATABASE_BACKUP_POLICY_VIOLATION&#34; | &#34;MIGRATION_SCHEMA&#34;.&#39; |
| page_size | [int32](#int32) |  | Not used. The maximum number of anomalies to return. The service may return fewer than this value. If unspecified, at most 50 anomalies will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `SearchAnomalies` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `SearchAnomalies` must match the call that provided the page token. |






<a name="bytebase-v1-SearchAnomaliesResponse"></a>

### SearchAnomaliesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| anomalies | [Anomaly](#bytebase-v1-Anomaly) | repeated | anomalies is the list of anomalies. |
| next_page_token | [string](#string) |  | Not used. A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |





 


<a name="bytebase-v1-Anomaly-AnomalySeverity"></a>

### Anomaly.AnomalySeverity
AnomalySeverity is the severity of the anomaly.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ANOMALY_SEVERITY_UNSPECIFIED | 0 | Unspecified anomaly severity. |
| MEDIUM | 1 | MEDIUM is the info level anomaly severity. |
| HIGH | 2 | HIGH is the warning level anomaly severity. |
| CRITICAL | 3 | CRITICAL is the critical level anomaly severity. |



<a name="bytebase-v1-Anomaly-AnomalyType"></a>

### Anomaly.AnomalyType
AnomalyType is the type of the anomaly.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ANOMALY_TYPE_UNSPECIFIED | 0 | Unspecified anomaly type. |
| INSTANCE_CONNECTION | 1 | Instance level anomaly.

INSTANCE_CONNECTION is the anomaly type for instance connection, e.g. the instance is down. |
| MIGRATION_SCHEMA | 2 | MIGRATION_SCHEMA is the anomaly type for migration schema, e.g. the migration schema in the instance is missing. |
| DATABASE_BACKUP_POLICY_VIOLATION | 3 | Database level anomaly.

DATABASE_BACKUP_POLICY_VIOLATION is the anomaly type for database backup policy violation, e.g. the database backup policy is not meet the environment backup policy. |
| DATABASE_BACKUP_MISSING | 4 | DATABASE_BACKUP_MISSING is the anomaly type for the backup missing, e.g. the backup is missing. |
| DATABASE_CONNECTION | 5 | DATABASE_CONNECTION is the anomaly type for database connection, e.g. the database had been deleted. |
| DATABASE_SCHEMA_DRIFT | 6 | DATABASE_SCHEMA_DRIFT is the anomaly type for database schema drift, e.g. the database schema had been changed without bytebase migration. |


 

 


<a name="bytebase-v1-AnomalyService"></a>

### AnomalyService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| SearchAnomalies | [SearchAnomaliesRequest](#bytebase-v1-SearchAnomaliesRequest) | [SearchAnomaliesResponse](#bytebase-v1-SearchAnomaliesResponse) |  |

 



<a name="v1_auth_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/auth_service.proto



<a name="bytebase-v1-CreateUserRequest"></a>

### CreateUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#bytebase-v1-User) |  | The user to create. |






<a name="bytebase-v1-DeleteUserRequest"></a>

### DeleteUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the user to delete. Format: users/{user} |






<a name="bytebase-v1-GetUserRequest"></a>

### GetUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the user to retrieve. Format: users/{user} |






<a name="bytebase-v1-IdentityProviderContext"></a>

### IdentityProviderContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| oauth2_context | [OAuth2IdentityProviderContext](#bytebase-v1-OAuth2IdentityProviderContext) |  |  |
| oidc_context | [OIDCIdentityProviderContext](#bytebase-v1-OIDCIdentityProviderContext) |  |  |






<a name="bytebase-v1-ListUsersRequest"></a>

### ListUsersRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of users to return. The service may return fewer than this value. If unspecified, at most 50 users will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListUsers` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListUsers` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Show deleted users if specified. |






<a name="bytebase-v1-ListUsersResponse"></a>

### ListUsersResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| users | [User](#bytebase-v1-User) | repeated | The users from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-LoginRequest"></a>

### LoginRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| email | [string](#string) |  |  |
| password | [string](#string) |  |  |
| web | [bool](#bool) |  | If web is set, we will set access token, refresh token, and user to the cookie. |
| idp_name | [string](#string) |  | The name of the identity provider. Format: idps/{idp} |
| idp_context | [IdentityProviderContext](#bytebase-v1-IdentityProviderContext) |  | The idp_context is using to get the user information from identity provider. |
| otp_code | [string](#string) | optional | The otp_code is used to verify the user&#39;s identity by MFA. |
| recovery_code | [string](#string) | optional | The recovery_code is used to recovery the user&#39;s identity with MFA. |
| mfa_temp_token | [string](#string) | optional | The mfa_temp_token is used to verify the user&#39;s identity by MFA. |






<a name="bytebase-v1-LoginResponse"></a>

### LoginResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  |  |
| mfa_temp_token | [string](#string) | optional |  |






<a name="bytebase-v1-LogoutRequest"></a>

### LogoutRequest







<a name="bytebase-v1-OAuth2IdentityProviderContext"></a>

### OAuth2IdentityProviderContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [string](#string) |  |  |






<a name="bytebase-v1-OIDCIdentityProviderContext"></a>

### OIDCIdentityProviderContext







<a name="bytebase-v1-UndeleteUserRequest"></a>

### UndeleteUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the deleted user. Format: users/{user} |






<a name="bytebase-v1-UpdateUserRequest"></a>

### UpdateUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#bytebase-v1-User) |  | The user to update.

The user&#39;s `name` field is used to identify the user to update. Format: users/{user} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| otp_code | [string](#string) | optional | The otp_code is used to verify the user&#39;s identity by MFA. |
| regenerate_temp_mfa_secret | [bool](#bool) |  | The regenerate_temp_mfa_secret flag means to regenerate temporary MFA secret for user. This is used for MFA setup. The temporary MFA secret and recovery codes will be returned in the response. |
| regenerate_recovery_codes | [bool](#bool) |  | The regenerate_recovery_codes flag means to regenerate recovery codes for user. |






<a name="bytebase-v1-User"></a>

### User



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the user. Format: users/{user}. {user} is a system-generated unique ID. |
| state | [State](#bytebase-v1-State) |  |  |
| email | [string](#string) |  |  |
| title | [string](#string) |  |  |
| user_type | [UserType](#bytebase-v1-UserType) |  |  |
| user_role | [UserRole](#bytebase-v1-UserRole) |  | The user role will not be respected in the create user request, because the role is controlled by workspace owner. |
| password | [string](#string) |  |  |
| service_key | [string](#string) |  |  |
| mfa_enabled | [bool](#bool) |  | The mfa_enabled flag means if the user has enabled MFA. |
| mfa_secret | [string](#string) |  | The mfa_secret is the temporary secret using in two phase verification. |
| recovery_codes | [string](#string) | repeated | The recovery_codes is the temporary recovery codes using in two phase verification. |
| phone | [string](#string) |  | Should be a valid E.164 compliant phone number. Could be empty. |





 


<a name="bytebase-v1-UserRole"></a>

### UserRole


| Name | Number | Description |
| ---- | ------ | ----------- |
| USER_ROLE_UNSPECIFIED | 0 |  |
| OWNER | 1 |  |
| DBA | 2 |  |
| DEVELOPER | 3 |  |



<a name="bytebase-v1-UserType"></a>

### UserType


| Name | Number | Description |
| ---- | ------ | ----------- |
| USER_TYPE_UNSPECIFIED | 0 |  |
| USER | 1 |  |
| SYSTEM_BOT | 2 |  |
| SERVICE_ACCOUNT | 3 |  |


 

 


<a name="bytebase-v1-AuthService"></a>

### AuthService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetUser | [GetUserRequest](#bytebase-v1-GetUserRequest) | [User](#bytebase-v1-User) |  |
| ListUsers | [ListUsersRequest](#bytebase-v1-ListUsersRequest) | [ListUsersResponse](#bytebase-v1-ListUsersResponse) |  |
| CreateUser | [CreateUserRequest](#bytebase-v1-CreateUserRequest) | [User](#bytebase-v1-User) |  |
| UpdateUser | [UpdateUserRequest](#bytebase-v1-UpdateUserRequest) | [User](#bytebase-v1-User) |  |
| DeleteUser | [DeleteUserRequest](#bytebase-v1-DeleteUserRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| UndeleteUser | [UndeleteUserRequest](#bytebase-v1-UndeleteUserRequest) | [User](#bytebase-v1-User) |  |
| Login | [LoginRequest](#bytebase-v1-LoginRequest) | [LoginResponse](#bytebase-v1-LoginResponse) |  |
| Logout | [LogoutRequest](#bytebase-v1-LogoutRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="v1_bookmark_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/bookmark_service.proto



<a name="bytebase-v1-Bookmark"></a>

### Bookmark



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the bookmark. Format: bookmarks/{bookmark}, user and bookmark are server-generated unique IDs. |
| title | [string](#string) |  | The title of the bookmark. |
| link | [string](#string) |  | The resource link of the bookmark. Only support issue link for now. Format: Issue: /issue/slug(issue_name)-{issue_uid} Example: /issue/start-here-add-email-column-to-employee-table-101 |






<a name="bytebase-v1-CreateBookmarkRequest"></a>

### CreateBookmarkRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bookmark | [Bookmark](#bytebase-v1-Bookmark) |  | The bookmark to create. |






<a name="bytebase-v1-DeleteBookmarkRequest"></a>

### DeleteBookmarkRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the bookmark to delete. Format: bookmarks/{bookmark} |






<a name="bytebase-v1-ListBookmarksRequest"></a>

### ListBookmarksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Not used. The maximum number of bookmarks to return. The service may return fewer than this value. If unspecified, at most 50 bookmarks will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListBookmarks` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListBookmarks` must match the call that provided the page token. |






<a name="bytebase-v1-ListBookmarksResponse"></a>

### ListBookmarksResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bookmarks | [Bookmark](#bytebase-v1-Bookmark) | repeated | The list of bookmarks. |
| next_page_token | [string](#string) |  | Not used. A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |





 

 

 


<a name="bytebase-v1-BookmarkService"></a>

### BookmarkService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateBookmark | [CreateBookmarkRequest](#bytebase-v1-CreateBookmarkRequest) | [Bookmark](#bytebase-v1-Bookmark) | CreateBookmark creates a new bookmark. |
| DeleteBookmark | [DeleteBookmarkRequest](#bytebase-v1-DeleteBookmarkRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | DeleteBookmark deletes a bookmark. |
| ListBookmarks | [ListBookmarksRequest](#bytebase-v1-ListBookmarksRequest) | [ListBookmarksResponse](#bytebase-v1-ListBookmarksResponse) | ListBookmarks lists bookmarks. |

 



<a name="v1_cel_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/cel_service.proto



<a name="bytebase-v1-DeparseRequest"></a>

### DeparseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expression | [google.api.expr.v1alpha1.ParsedExpr](#google-api-expr-v1alpha1-ParsedExpr) |  |  |






<a name="bytebase-v1-DeparseResponse"></a>

### DeparseResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expression | [string](#string) |  |  |






<a name="bytebase-v1-ParseRequest"></a>

### ParseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expression | [string](#string) |  |  |






<a name="bytebase-v1-ParseResponse"></a>

### ParseResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expression | [google.api.expr.v1alpha1.ParsedExpr](#google-api-expr-v1alpha1-ParsedExpr) |  |  |





 

 

 


<a name="bytebase-v1-CelService"></a>

### CelService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Parse | [ParseRequest](#bytebase-v1-ParseRequest) | [ParseResponse](#bytebase-v1-ParseResponse) |  |
| Deparse | [DeparseRequest](#bytebase-v1-DeparseRequest) | [DeparseResponse](#bytebase-v1-DeparseResponse) |  |

 



<a name="v1_database_edit-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/database_edit.proto



<a name="bytebase-v1-AddColumnContext"></a>

### AddColumnContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the column to add. |
| type | [string](#string) |  | The type of the column. |
| character_set | [string](#string) |  | The character set of the column. |
| collation | [string](#string) |  | The collation of the column. |
| comment | [string](#string) |  | The comment of the column. |
| nullable | [bool](#bool) |  | Whether the column is nullable. |
| default_value | [string](#string) |  | The default value of the column. |
| has_default_value | [bool](#bool) |  | Whether the column has a default value. |






<a name="bytebase-v1-AddForeignKeyContext"></a>

### AddForeignKeyContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| column | [string](#string) |  | The column of the foreign key. |
| referenced_schema | [string](#string) |  | The referenced schema of the foreign key. |
| referenced_table | [string](#string) |  | The referenced table of the foreign key. |
| referenced_column | [string](#string) |  | The referenced column of the foreign key. |






<a name="bytebase-v1-AlterColumnContext"></a>

### AlterColumnContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| old_name | [string](#string) |  | The old name of the column. |
| new_name | [string](#string) |  | The new name of the column. |
| type | [string](#string) |  | The type of the column. |
| character_set | [string](#string) |  | The character set of the column. |
| collation | [string](#string) |  | The collation of the column. |
| comment | [string](#string) |  | The comment of the column. |
| nullable | [bool](#bool) |  | Whether the column is nullable. |
| default_value | [string](#string) |  | The default value of the column. |
| is_default_value_changed | [bool](#bool) |  | Whether the default value of the column has changed. |






<a name="bytebase-v1-AlterTableContext"></a>

### AlterTableContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the table to alter. |
| schema | [string](#string) |  | The schema of the table. |
| add_column_contexts | [AddColumnContext](#bytebase-v1-AddColumnContext) | repeated | List of column addition contexts. |
| alter_column_contexts | [AlterColumnContext](#bytebase-v1-AlterColumnContext) | repeated | List of column alteration contexts. |
| drop_column_contexts | [DropColumnContext](#bytebase-v1-DropColumnContext) | repeated | List of column dropping contexts. |
| drop_primary_keys | [string](#string) | repeated | List of primary key columns to be dropped. |
| primary_keys | [string](#string) | repeated | List of primary key columns. |
| drop_foreign_keys | [string](#string) | repeated | List of foreign key columns to be dropped. |
| add_foreign_key_contexts | [AddForeignKeyContext](#bytebase-v1-AddForeignKeyContext) | repeated | List of foreign key addition contexts. |






<a name="bytebase-v1-CreateSchemaContext"></a>

### CreateSchemaContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the schema to create. |






<a name="bytebase-v1-CreateTableContext"></a>

### CreateTableContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the table to create. |
| schema | [string](#string) |  | The schema of the table. |
| type | [string](#string) |  | The type of the table. |
| engine | [string](#string) |  | The engine of the table. |
| character_set | [string](#string) |  | The character set of the table. |
| collation | [string](#string) |  | The collation of the table. |
| comment | [string](#string) |  | The comment of the table. |
| add_column_contexts | [AddColumnContext](#bytebase-v1-AddColumnContext) | repeated | List of column addition contexts. |
| primary_keys | [string](#string) | repeated | List of primary key columns. |
| add_foreign_key_contexts | [AddForeignKeyContext](#bytebase-v1-AddForeignKeyContext) | repeated | List of foreign key addition contexts. |






<a name="bytebase-v1-DatabaseEdit"></a>

### DatabaseEdit



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| create_schema_contexts | [CreateSchemaContext](#bytebase-v1-CreateSchemaContext) | repeated | List of schema creation contexts. |
| rename_schema_contexts | [RenameSchemaContext](#bytebase-v1-RenameSchemaContext) | repeated | List of schema renaming contexts. |
| drop_schema_contexts | [DropSchemaContext](#bytebase-v1-DropSchemaContext) | repeated | List of schema dropping contexts. |
| create_table_contexts | [CreateTableContext](#bytebase-v1-CreateTableContext) | repeated | List of table creation contexts. |
| alter_table_contexts | [AlterTableContext](#bytebase-v1-AlterTableContext) | repeated | List of table alteration contexts. |
| rename_table_contexts | [RenameTableContext](#bytebase-v1-RenameTableContext) | repeated | List of table renaming contexts. |
| drop_table_contexts | [DropTableContext](#bytebase-v1-DropTableContext) | repeated | List of table dropping contexts. |






<a name="bytebase-v1-DropColumnContext"></a>

### DropColumnContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the column to drop. |






<a name="bytebase-v1-DropSchemaContext"></a>

### DropSchemaContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the schema to drop. |






<a name="bytebase-v1-DropTableContext"></a>

### DropTableContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the table to drop. |
| schema | [string](#string) |  | The schema of the table. |






<a name="bytebase-v1-RenameSchemaContext"></a>

### RenameSchemaContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| old_name | [string](#string) |  | The old name of the schema. |
| new_name | [string](#string) |  | The new name of the schema. |






<a name="bytebase-v1-RenameTableContext"></a>

### RenameTableContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  | The schema of the table. |
| old_name | [string](#string) |  | The old name of the table. |
| new_name | [string](#string) |  | The new name of the table. |





 

 

 

 



<a name="v1_vcs-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/vcs.proto



<a name="bytebase-v1-Commit"></a>

### Commit



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| title | [string](#string) |  |  |
| message | [string](#string) |  |  |
| created_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| url | [string](#string) |  |  |
| author_name | [string](#string) |  |  |
| author_email | [string](#string) |  |  |
| added_list | [string](#string) | repeated |  |
| modified_list | [string](#string) | repeated |  |






<a name="bytebase-v1-FileCommit"></a>

### FileCommit



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| title | [string](#string) |  |  |
| message | [string](#string) |  |  |
| created_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| url | [string](#string) |  |  |
| author_name | [string](#string) |  |  |
| author_email | [string](#string) |  |  |
| added | [string](#string) |  |  |






<a name="bytebase-v1-PushEvent"></a>

### PushEvent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vcs_type | [VcsType](#bytebase-v1-VcsType) |  |  |
| base_dir | [string](#string) |  |  |
| ref | [string](#string) |  |  |
| before | [string](#string) |  |  |
| after | [string](#string) |  |  |
| repository_id | [string](#string) |  |  |
| repository_url | [string](#string) |  |  |
| repository_full_path | [string](#string) |  |  |
| author_name | [string](#string) |  |  |
| commits | [Commit](#bytebase-v1-Commit) | repeated |  |
| file_commit | [FileCommit](#bytebase-v1-FileCommit) |  |  |





 


<a name="bytebase-v1-VcsType"></a>

### VcsType


| Name | Number | Description |
| ---- | ------ | ----------- |
| VCS_TYPE_UNSPECIFIED | 0 |  |
| GITLAB | 1 |  |
| GITHUB | 2 |  |
| BITBUCKET | 3 |  |


 

 

 



<a name="v1_database_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/database_service.proto



<a name="bytebase-v1-AdviseIndexRequest"></a>

### AdviseIndexRequest
AdviseIndexRequest is the request of advising index.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: instances/{instance}/databases/{database} |
| statement | [string](#string) |  | The statement to be advised. |






<a name="bytebase-v1-AdviseIndexResponse"></a>

### AdviseIndexResponse
AdviseIndexResponse is the response of advising index.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| current_index | [string](#string) |  | The current index of the statement used. |
| suggestion | [string](#string) |  | The suggested index of the statement. |
| create_index_statement | [string](#string) |  | The create index statement of the suggested index. |






<a name="bytebase-v1-Backup"></a>

### Backup
The message of the backup.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The resource name of the database backup. backup-name is specified by the client. Format: instances/{instance}/databases/{database}/backups/{backup-name} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the backup resource was created initially. |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the backup resource was updated. |
| state | [Backup.BackupState](#bytebase-v1-Backup-BackupState) |  | The state of the backup. |
| backup_type | [Backup.BackupType](#bytebase-v1-Backup-BackupType) |  | The type of the backup. |
| comment | [string](#string) |  | The comment of the backup. |
| uid | [string](#string) |  |  |






<a name="bytebase-v1-BackupSetting"></a>

### BackupSetting
BackupSetting is the setting for database backup.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database backup setting. Format: instances/{instance}/databases/{database}/backupSetting |
| backup_retain_duration | [google.protobuf.Duration](#google-protobuf-Duration) |  | The default maximum age of a Backup created via this BackupPlan. If specified, a Backup will be automatically deleted after its age reaches. If not specified, Backups created under this BackupPlan will be deleted after 7 DAYS. It will be rounded up to the number of days. |
| cron_schedule | [string](#string) |  | Cron(https://wikipedia.com/wiki/cron) string that defines a repeating schedule for creating Backups. Support hour of day, day of week. (UTC time)

Default (empty): Disable automatic backup. |
| hook_url | [string](#string) |  | hook_url(https://www.bytebase.com/docs/disaster-recovery/backup/#post-backup-webhook) is the URL to send a notification when a backup is created. |






<a name="bytebase-v1-BatchUpdateDatabasesRequest"></a>

### BatchUpdateDatabasesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource shared by all databases being updated. Format: instances/{instance} If the operation spans parents, a dash (-) may be accepted as a wildcard. We only support updating the project of databases for now. |
| requests | [UpdateDatabaseRequest](#bytebase-v1-UpdateDatabaseRequest) | repeated | The request message specifying the resources to update. A maximum of 1000 databases can be modified in a batch. |






<a name="bytebase-v1-BatchUpdateDatabasesResponse"></a>

### BatchUpdateDatabasesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [Database](#bytebase-v1-Database) | repeated | Databases updated. |






<a name="bytebase-v1-ChangeHistory"></a>

### ChangeHistory



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: instances/{instance}/databases/{database}/changeHistories/{changeHistory} |
| uid | [string](#string) |  |  |
| creator | [string](#string) |  | Format: users/hello@world.com |
| updater | [string](#string) |  | Format: users/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| release_version | [string](#string) |  | release version of Bytebase |
| source | [ChangeHistory.Source](#bytebase-v1-ChangeHistory-Source) |  |  |
| type | [ChangeHistory.Type](#bytebase-v1-ChangeHistory-Type) |  |  |
| status | [ChangeHistory.Status](#bytebase-v1-ChangeHistory-Status) |  |  |
| version | [string](#string) |  |  |
| description | [string](#string) |  |  |
| statement | [string](#string) |  |  |
| schema | [string](#string) |  |  |
| prev_schema | [string](#string) |  |  |
| execution_duration | [google.protobuf.Duration](#google-protobuf-Duration) |  |  |
| issue | [string](#string) |  | Format: projects/{project}/issues/{issue} |
| push_event | [PushEvent](#bytebase-v1-PushEvent) |  |  |
| changed_resources | [ChangedResources](#bytebase-v1-ChangedResources) |  |  |






<a name="bytebase-v1-ChangedResourceDatabase"></a>

### ChangedResourceDatabase



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| schemas | [ChangedResourceSchema](#bytebase-v1-ChangedResourceSchema) | repeated |  |






<a name="bytebase-v1-ChangedResourceSchema"></a>

### ChangedResourceSchema



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| tables | [ChangedResourceTable](#bytebase-v1-ChangedResourceTable) | repeated |  |






<a name="bytebase-v1-ChangedResourceTable"></a>

### ChangedResourceTable



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |






<a name="bytebase-v1-ChangedResources"></a>

### ChangedResources



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [ChangedResourceDatabase](#bytebase-v1-ChangedResourceDatabase) | repeated |  |






<a name="bytebase-v1-ColumnMetadata"></a>

### ColumnMetadata
ColumnMetadata is the metadata for columns.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a column. |
| position | [int32](#int32) |  | The position is the position in columns. |
| default | [google.protobuf.StringValue](#google-protobuf-StringValue) |  | The default is the default of a column. Use google.protobuf.StringValue to distinguish between an empty string default value or no default. |
| nullable | [bool](#bool) |  | The nullable is the nullable of a column. |
| type | [string](#string) |  | The type is the type of a column. |
| character_set | [string](#string) |  | The character_set is the character_set of a column. |
| collation | [string](#string) |  | The collation is the collation of a column. |
| comment | [string](#string) |  | The comment is the comment of a column. classification and user_comment is parsed from the comment. |
| classification | [string](#string) |  | The classification is the classification of a column parsed from the comment. |
| user_comment | [string](#string) |  | The user_comment is the user comment of a column parsed from the comment. |






<a name="bytebase-v1-CreateBackupRequest"></a>

### CreateBackupRequest
CreateBackupRequest is the request message for CreateBackup.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this backup will be created. Format: instances/{instance}/databases/{database} |
| backup | [Backup](#bytebase-v1-Backup) |  |  |






<a name="bytebase-v1-Database"></a>

### Database



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database. Format: instances/{instance}/databases/{database} {database} is the database name in the instance. |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| sync_state | [State](#bytebase-v1-State) |  | The existence of a database on latest sync. |
| successful_sync_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The latest synchronization time. |
| project | [string](#string) |  | The project for a database. Format: projects/{project} |
| schema_version | [string](#string) |  | The version of database schema. |
| environment | [string](#string) |  | The environment resource. Format: environments/prod where prod is the environment resource ID. |
| effective_environment | [string](#string) |  | The effective environment based on environment tag above and environment tag on the instance. Inheritance follows https://cloud.google.com/resource-manager/docs/tags/tags-overview. |
| labels | [Database.LabelsEntry](#bytebase-v1-Database-LabelsEntry) | repeated | Labels will be used for deployment and policy control. |






<a name="bytebase-v1-Database-LabelsEntry"></a>

### Database.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-v1-DatabaseMetadata"></a>

### DatabaseMetadata
DatabaseMetadata is the metadata for databases.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| schemas | [SchemaMetadata](#bytebase-v1-SchemaMetadata) | repeated | The schemas is the list of schemas in a database. |
| character_set | [string](#string) |  | The character_set is the character set of a database. |
| collation | [string](#string) |  | The collation is the collation of a database. |
| extensions | [ExtensionMetadata](#bytebase-v1-ExtensionMetadata) | repeated | The extensions is the list of extensions in a database. |






<a name="bytebase-v1-DatabaseSchema"></a>

### DatabaseSchema
DatabaseMetadata is the metadata for databases.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  | The schema dump from database. |






<a name="bytebase-v1-DeleteSecretRequest"></a>

### DeleteSecretRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the secret to be deleted. Format: instances/{instance}/databases/{database}/secrets/{secret} |






<a name="bytebase-v1-DependentColumn"></a>

### DependentColumn
DependentColumn is the metadata for dependent columns.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  | The schema is the schema of a reference column. |
| table | [string](#string) |  | The table is the table of a reference column. |
| column | [string](#string) |  | The column is the name of a reference column. |






<a name="bytebase-v1-ExtensionMetadata"></a>

### ExtensionMetadata
ExtensionMetadata is the metadata for extensions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of an extension. |
| schema | [string](#string) |  | The schema is the extension that is installed to. But the extension usage is not limited to the schema. |
| version | [string](#string) |  | The version is the version of an extension. |
| description | [string](#string) |  | The description is the description of an extension. |






<a name="bytebase-v1-ForeignKeyMetadata"></a>

### ForeignKeyMetadata
ForeignKeyMetadata is the metadata for foreign keys.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a foreign key. |
| columns | [string](#string) | repeated | The columns are the ordered referencing columns of a foreign key. |
| referenced_schema | [string](#string) |  | The referenced_schema is the referenced schema name of a foreign key. It is an empty string for databases without such concept such as MySQL. |
| referenced_table | [string](#string) |  | The referenced_table is the referenced table name of a foreign key. |
| referenced_columns | [string](#string) | repeated | The referenced_columns are the ordered referenced columns of a foreign key. |
| on_delete | [string](#string) |  | The on_delete is the on delete action of a foreign key. |
| on_update | [string](#string) |  | The on_update is the on update action of a foreign key. |
| match_type | [string](#string) |  | The match_type is the match type of a foreign key. The match_type is the PostgreSQL specific field. It&#39;s empty string for other databases. |






<a name="bytebase-v1-FunctionMetadata"></a>

### FunctionMetadata
FunctionMetadata is the metadata for functions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a view. |
| definition | [string](#string) |  | The definition is the definition of a view. |






<a name="bytebase-v1-GetBackupSettingRequest"></a>

### GetBackupSettingRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to retrieve backup setting. Format: instances/{instance}/databases/{database}/backupSetting |






<a name="bytebase-v1-GetChangeHistoryRequest"></a>

### GetChangeHistoryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the change history to retrieve. Format: instances/{instance}/databases/{database}/changeHistories/{changeHistory} |
| view | [ChangeHistoryView](#bytebase-v1-ChangeHistoryView) |  |  |
| sdl_format | [bool](#bool) |  | Format the schema dump into SDL format. |






<a name="bytebase-v1-GetDatabaseMetadataRequest"></a>

### GetDatabaseMetadataRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to retrieve metadata. Format: instances/{instance}/databases/{database}/metadata |






<a name="bytebase-v1-GetDatabaseRequest"></a>

### GetDatabaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to retrieve. Format: instances/{instance}/databases/{database} |






<a name="bytebase-v1-GetDatabaseSchemaRequest"></a>

### GetDatabaseSchemaRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to retrieve schema. Format: instances/{instance}/databases/{database}/schema |
| sdl_format | [bool](#bool) |  | Format the schema dump into SDL format. |






<a name="bytebase-v1-IndexMetadata"></a>

### IndexMetadata
IndexMetadata is the metadata for indexes.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of an index. |
| expressions | [string](#string) | repeated | The expressions are the ordered columns or expressions of an index. This could refer to a column or an expression. |
| type | [string](#string) |  | The type is the type of an index. |
| unique | [bool](#bool) |  | The unique is whether the index is unique. |
| primary | [bool](#bool) |  | The primary is whether the index is a primary key index. |
| visible | [bool](#bool) |  | The visible is whether the index is visible. |
| comment | [string](#string) |  | The comment is the comment of an index. |






<a name="bytebase-v1-ListBackupsRequest"></a>

### ListBackupsRequest
ListBackupsRequest is the request message for ListBackup.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this backup will be created. Format: instances/{instance}/databases/{database} |
| page_size | [int32](#int32) |  | Not used. The maximum number of backups to return. The service may return fewer than this value. If unspecified, at most 50 backups will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListBackup` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListBackup` must match the call that provided the page token. |






<a name="bytebase-v1-ListBackupsResponse"></a>

### ListBackupsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| backups | [Backup](#bytebase-v1-Backup) | repeated | The backups from the specified request. |
| next_page_token | [string](#string) |  | Not used. A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-ListChangeHistoriesRequest"></a>

### ListChangeHistoriesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent of the change histories. Format: instances/{instance}/databases/{database} |
| page_size | [int32](#int32) |  | The maximum number of change histories to return. The service may return fewer than this value. If unspecified, at most 10 change histories will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListChangeHistories` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListChangeHistories` must match the call that provided the page token. |
| view | [ChangeHistoryView](#bytebase-v1-ChangeHistoryView) |  |  |
| filter | [string](#string) |  | The filter of the change histories. Follow the CEL syntax. currently, we have one function for CEL: - tableExists(database, schema, table): return true if the table exists in changed resources.

examples: Use tableExists(&#34;db&#34;, &#34;public&#34;, &#34;table1&#34;) to filter the change histories which have the table &#34;table1&#34; in the schema &#34;public&#34; of the database &#34;db&#34;. For MySQL, the schema is always &#34;&#34;, such as tableExists(&#34;db&#34;, &#34;&#34;, &#34;table1&#34;).

Combine multiple functions with &#34;&amp;&amp;&#34; and &#34;||&#34;, we MUST use the Disjunctive Normal Form(DNF). In other words, the CEL expression consists of several parts connected by OR operators. For example, the following expression is valid: ( tableExists(&#34;db&#34;, &#34;public&#34;, &#34;table1&#34;) &amp;&amp; tableExists(&#34;db&#34;, &#34;public&#34;, &#34;table2&#34;) ) || ( tableExists(&#34;db&#34;, &#34;public&#34;, &#34;table3&#34;) ) |






<a name="bytebase-v1-ListChangeHistoriesResponse"></a>

### ListChangeHistoriesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| change_histories | [ChangeHistory](#bytebase-v1-ChangeHistory) | repeated | The list of change histories. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-ListDatabasesRequest"></a>

### ListDatabasesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of databases. Format: instances/{instance} Use &#34;instances/-&#34; to list all databases. |
| page_size | [int32](#int32) |  | The maximum number of databases to return. The service may return fewer than this value. If unspecified, at most 50 databases will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListDatabases` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListDatabases` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter databases returned in the list. For example, &#34;project = projects/{project}&#34; can be used to list databases in a project. |






<a name="bytebase-v1-ListDatabasesResponse"></a>

### ListDatabasesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [Database](#bytebase-v1-Database) | repeated | The databases from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-ListSecretsRequest"></a>

### ListSecretsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent of the secret. Format: instances/{instance}/databases/{database} |
| page_size | [int32](#int32) |  | Not used. The maximum number of databases to return. The service may return fewer than this value. If unspecified, at most 50 databases will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListSecrets` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListSecrets` must match the call that provided the page token. |






<a name="bytebase-v1-ListSecretsResponse"></a>

### ListSecretsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secrets | [Secret](#bytebase-v1-Secret) | repeated | The list of secrets. |
| next_page_token | [string](#string) |  | Not used. A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-ListSlowQueriesRequest"></a>

### ListSlowQueriesRequest
ListSlowQueriesRequest is the request of listing slow query.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: instances/{instance}/databases/{database} |
| filter | [string](#string) |  | The filter of the slow query log. follow the [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) syntax. Support filter by project and start_time in SlowQueryDetails for now. For example: Search the slow query log of the specific project: - the specific project: project = &#34;projects/{project}&#34; Search the slow query log that start_time after 2022-01-01T12:00:00.000Z: - start_time &gt; &#34;2022-01-01T12:00:00.000Z&#34; - Should use [RFC-3339 format](https://www.rfc-editor.org/rfc/rfc3339). - Currently we only support filtering down to date granularity. |
| order_by | [string](#string) |  | The order by of the slow query log. Support order by count, latest_log_time, average_query_time, maximum_query_time, average_rows_sent, maximum_rows_sent, average_rows_examined, maximum_rows_examined for now. For example: - order by count: order_by = &#34;count&#34; - order by latest_log_time desc: order_by = &#34;latest_log_time desc&#34; Default: order by average_query_time desc. |






<a name="bytebase-v1-ListSlowQueriesResponse"></a>

### ListSlowQueriesResponse
ListSlowQueriesResponse is the response of listing slow query.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| slow_query_logs | [SlowQueryLog](#bytebase-v1-SlowQueryLog) | repeated | The slow query logs. |






<a name="bytebase-v1-SchemaMetadata"></a>

### SchemaMetadata
SchemaMetadata is the metadata for schemas.
This is the concept of schema in Postgres, but it&#39;s a no-op for MySQL.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the schema name. It is an empty string for databases without such concept such as MySQL. |
| tables | [TableMetadata](#bytebase-v1-TableMetadata) | repeated | The tables is the list of tables in a schema. |
| views | [ViewMetadata](#bytebase-v1-ViewMetadata) | repeated | The views is the list of views in a schema. |
| functions | [FunctionMetadata](#bytebase-v1-FunctionMetadata) | repeated | The functions is the list of functions in a schema. |
| streams | [StreamMetadata](#bytebase-v1-StreamMetadata) | repeated | The streams is the list of streams in a schema, currently, only used for Snowflake. |
| tasks | [TaskMetadata](#bytebase-v1-TaskMetadata) | repeated | The routines is the list of routines in a schema, currently, only used for Snowflake. |






<a name="bytebase-v1-SearchDatabasesRequest"></a>

### SearchDatabasesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of databases. Format: instances/{instance} Use &#34;instances/-&#34; to list all databases. |
| page_size | [int32](#int32) |  | The maximum number of databases to return. The service may return fewer than this value. If unspecified, at most 50 databases will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListDatabases` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListDatabases` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter databases returned in the list. For example, &#34;project = projects/{project}&#34; can be used to list databases in a project. |






<a name="bytebase-v1-SearchDatabasesResponse"></a>

### SearchDatabasesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [Database](#bytebase-v1-Database) | repeated | The databases from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-Secret"></a>

### Secret
Secret is the secret of the database now.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | name is the unique name of the secret, which is specified by the client. Format: instances/{instance}/databases/{database}/secrets/{secret} |
| created_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Not used. The timestamp when the secret resource was created initially. |
| updated_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Not used. The timestamp when the secret resource was updated. |
| value | [string](#string) |  | The value of the secret. |
| description | [string](#string) |  | The description of the secret. |






<a name="bytebase-v1-SlowQueryDetails"></a>

### SlowQueryDetails
SlowQueryDetails is the details of the slow query log.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The start time of the slow query log. |
| query_time | [google.protobuf.Duration](#google-protobuf-Duration) |  | The query time of the slow query log. |
| lock_time | [google.protobuf.Duration](#google-protobuf-Duration) |  | The lock time of the slow query log. |
| rows_sent | [int64](#int64) |  | The rows sent of the slow query log. |
| rows_examined | [int64](#int64) |  | The rows examined of the slow query log. |
| sql_text | [string](#string) |  | The sql text of the slow query log. |






<a name="bytebase-v1-SlowQueryLog"></a>

### SlowQueryLog
SlowQueryLog is the slow query log.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource | [string](#string) |  | The resource of the slow query log. The format is &#34;instances/{instance}/databases/{database}&#34;. |
| project | [string](#string) |  | The project of the slow query log. The format is &#34;projects/{project}&#34;. |
| statistics | [SlowQueryStatistics](#bytebase-v1-SlowQueryStatistics) |  | The statistics of the slow query log. |






<a name="bytebase-v1-SlowQueryStatistics"></a>

### SlowQueryStatistics
SlowQueryStatistics is the statistics of the slow query log.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sql_fingerprint | [string](#string) |  | The fingerprint of the slow query log. |
| count | [int64](#int64) |  | The count of the slow query log. |
| latest_log_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The latest log time of the slow query log. |
| average_query_time | [google.protobuf.Duration](#google-protobuf-Duration) |  | The average query time of the slow query log. |
| maximum_query_time | [google.protobuf.Duration](#google-protobuf-Duration) |  | The maximum query time of the slow query log. |
| average_rows_sent | [int64](#int64) |  | The average rows sent of the slow query log. |
| maximum_rows_sent | [int64](#int64) |  | The maximum rows sent of the slow query log. |
| average_rows_examined | [int64](#int64) |  | The average rows examined of the slow query log. |
| maximum_rows_examined | [int64](#int64) |  | The maximum rows examined of the slow query log. |
| query_time_percent | [double](#double) |  | The percentage of the query time. |
| count_percent | [double](#double) |  | The percentage of the count. |
| samples | [SlowQueryDetails](#bytebase-v1-SlowQueryDetails) | repeated | Samples are details of the sample slow query logs with the same fingerprint. |






<a name="bytebase-v1-StreamMetadata"></a>

### StreamMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a stream. |
| table_name | [string](#string) |  | The table_name is the name of the table/view that the stream is created on. |
| owner | [string](#string) |  | The owner of the stream. |
| comment | [string](#string) |  | The comment of the stream. |
| type | [StreamMetadata.Type](#bytebase-v1-StreamMetadata-Type) |  | The type of the stream. |
| stale | [bool](#bool) |  | Indicates whether the stream was last read before the `stale_after` time. |
| mode | [StreamMetadata.Mode](#bytebase-v1-StreamMetadata-Mode) |  | The mode of the stream. |
| definition | [string](#string) |  | The definition of the stream. |






<a name="bytebase-v1-SyncDatabaseRequest"></a>

### SyncDatabaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to sync. Format: instances/{instance}/databases/{database} |






<a name="bytebase-v1-SyncDatabaseResponse"></a>

### SyncDatabaseResponse







<a name="bytebase-v1-TableMetadata"></a>

### TableMetadata
TableMetadata is the metadata for tables.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a table. |
| columns | [ColumnMetadata](#bytebase-v1-ColumnMetadata) | repeated | The columns is the ordered list of columns in a table. |
| indexes | [IndexMetadata](#bytebase-v1-IndexMetadata) | repeated | The indexes is the list of indexes in a table. |
| engine | [string](#string) |  | The engine is the engine of a table. |
| collation | [string](#string) |  | The collation is the collation of a table. |
| row_count | [int64](#int64) |  | The row_count is the estimated number of rows of a table. |
| data_size | [int64](#int64) |  | The data_size is the estimated data size of a table. |
| index_size | [int64](#int64) |  | The index_size is the estimated index size of a table. |
| data_free | [int64](#int64) |  | The data_free is the estimated free data size of a table. |
| create_options | [string](#string) |  | The create_options is the create option of a table. |
| comment | [string](#string) |  | The comment is the comment of a table. classification and user_comment is parsed from the comment. |
| classification | [string](#string) |  | The classification is the classification of a table parsed from the comment. |
| user_comment | [string](#string) |  | The user_comment is the user comment of a table parsed from the comment. |
| foreign_keys | [ForeignKeyMetadata](#bytebase-v1-ForeignKeyMetadata) | repeated | The foreign_keys is the list of foreign keys in a table. |






<a name="bytebase-v1-TaskMetadata"></a>

### TaskMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a task. |
| id | [string](#string) |  | The id is the snowflake-generated id of a task. Example: 01ad32a0-1bb6-5e93-0000-000000000001 |
| owner | [string](#string) |  | The owner of the task. |
| comment | [string](#string) |  | The comment of the task. |
| warehouse | [string](#string) |  | The warehouse of the task. |
| schedule | [string](#string) |  | The schedule interval of the task. |
| predecessors | [string](#string) | repeated | The predecessor tasks of the task. |
| state | [TaskMetadata.State](#bytebase-v1-TaskMetadata-State) |  | The state of the task. |
| condition | [string](#string) |  | The condition of the task. |
| definition | [string](#string) |  | The definition of the task. |






<a name="bytebase-v1-UpdateBackupSettingRequest"></a>

### UpdateBackupSettingRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| setting | [BackupSetting](#bytebase-v1-BackupSetting) |  | The database backup setting to update. |






<a name="bytebase-v1-UpdateDatabaseRequest"></a>

### UpdateDatabaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [Database](#bytebase-v1-Database) |  | The database to update.

The database&#39;s `name` field is used to identify the database to update. Format: instances/{instance}/databases/{database} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






<a name="bytebase-v1-UpdateSecretRequest"></a>

### UpdateSecretRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secret | [Secret](#bytebase-v1-Secret) |  | The secret to be created or updated. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The mask of the fields to be updated. |
| allow_missing | [bool](#bool) |  | If true, the secret will be created if it does not exist. |






<a name="bytebase-v1-ViewMetadata"></a>

### ViewMetadata
ViewMetadata is the metadata for views.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a view. |
| definition | [string](#string) |  | The definition is the definition of a view. |
| comment | [string](#string) |  | The comment is the comment of a view. |
| dependent_columns | [DependentColumn](#bytebase-v1-DependentColumn) | repeated | The dependent_columns is the list of dependent columns of a view. |





 


<a name="bytebase-v1-Backup-BackupState"></a>

### Backup.BackupState
The state of the backup.

| Name | Number | Description |
| ---- | ------ | ----------- |
| BACKUP_STATE_UNSPECIFIED | 0 | The state of the backup is unknown. |
| PENDING_CREATE | 1 | The backup is being pending to create. |
| DONE | 2 | The backup is ready to use. |
| FAILED | 3 | The backup is being deleted. |



<a name="bytebase-v1-Backup-BackupType"></a>

### Backup.BackupType
The type of the backup.

| Name | Number | Description |
| ---- | ------ | ----------- |
| BACKUP_TYPE_UNSPECIFIED | 0 | The type of the backup is unknown. |
| MANUAL | 1 | The backup is created by user. |
| AUTOMATIC | 2 | The backup is created by automatic backup. |
| PITR | 3 | The backup is created automatically after doing PITR. |



<a name="bytebase-v1-ChangeHistory-Source"></a>

### ChangeHistory.Source


| Name | Number | Description |
| ---- | ------ | ----------- |
| SOURCE_UNSPECIFIED | 0 |  |
| UI | 1 |  |
| VCS | 2 |  |
| LIBRARY | 3 |  |



<a name="bytebase-v1-ChangeHistory-Status"></a>

### ChangeHistory.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| DONE | 2 |  |
| FAILED | 3 |  |



<a name="bytebase-v1-ChangeHistory-Type"></a>

### ChangeHistory.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| BASELINE | 1 |  |
| MIGRATE | 2 |  |
| MIGRATE_SDL | 3 |  |
| MIGRATE_GHOST | 4 |  |
| BRANCH | 5 |  |
| DATA | 6 |  |



<a name="bytebase-v1-ChangeHistoryView"></a>

### ChangeHistoryView


| Name | Number | Description |
| ---- | ------ | ----------- |
| CHANGE_HISTORY_VIEW_UNSPECIFIED | 0 | The default / unset value. The API will default to the BASIC view. |
| CHANGE_HISTORY_VIEW_BASIC | 1 |  |
| CHANGE_HISTORY_VIEW_FULL | 2 |  |



<a name="bytebase-v1-StreamMetadata-Mode"></a>

### StreamMetadata.Mode


| Name | Number | Description |
| ---- | ------ | ----------- |
| MODE_UNSPECIFIED | 0 |  |
| MODE_DEFAULT | 1 |  |
| MODE_APPEND_ONLY | 2 |  |
| MODE_INSERT_ONLY | 3 |  |



<a name="bytebase-v1-StreamMetadata-Type"></a>

### StreamMetadata.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| TYPE_DELTA | 1 |  |



<a name="bytebase-v1-TaskMetadata-State"></a>

### TaskMetadata.State


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATE_UNSPECIFIED | 0 |  |
| STATE_STARTED | 1 |  |
| STATE_SUSPENDED | 2 |  |


 

 


<a name="bytebase-v1-DatabaseService"></a>

### DatabaseService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetDatabase | [GetDatabaseRequest](#bytebase-v1-GetDatabaseRequest) | [Database](#bytebase-v1-Database) |  |
| ListDatabases | [ListDatabasesRequest](#bytebase-v1-ListDatabasesRequest) | [ListDatabasesResponse](#bytebase-v1-ListDatabasesResponse) |  |
| SearchDatabases | [SearchDatabasesRequest](#bytebase-v1-SearchDatabasesRequest) | [SearchDatabasesResponse](#bytebase-v1-SearchDatabasesResponse) | Search for databases that the caller has both projects.get permission on, and also satisfy the specified query. |
| UpdateDatabase | [UpdateDatabaseRequest](#bytebase-v1-UpdateDatabaseRequest) | [Database](#bytebase-v1-Database) |  |
| BatchUpdateDatabases | [BatchUpdateDatabasesRequest](#bytebase-v1-BatchUpdateDatabasesRequest) | [BatchUpdateDatabasesResponse](#bytebase-v1-BatchUpdateDatabasesResponse) |  |
| SyncDatabase | [SyncDatabaseRequest](#bytebase-v1-SyncDatabaseRequest) | [SyncDatabaseResponse](#bytebase-v1-SyncDatabaseResponse) |  |
| GetDatabaseMetadata | [GetDatabaseMetadataRequest](#bytebase-v1-GetDatabaseMetadataRequest) | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  |
| GetDatabaseSchema | [GetDatabaseSchemaRequest](#bytebase-v1-GetDatabaseSchemaRequest) | [DatabaseSchema](#bytebase-v1-DatabaseSchema) |  |
| GetBackupSetting | [GetBackupSettingRequest](#bytebase-v1-GetBackupSettingRequest) | [BackupSetting](#bytebase-v1-BackupSetting) |  |
| UpdateBackupSetting | [UpdateBackupSettingRequest](#bytebase-v1-UpdateBackupSettingRequest) | [BackupSetting](#bytebase-v1-BackupSetting) |  |
| CreateBackup | [CreateBackupRequest](#bytebase-v1-CreateBackupRequest) | [Backup](#bytebase-v1-Backup) |  |
| ListBackups | [ListBackupsRequest](#bytebase-v1-ListBackupsRequest) | [ListBackupsResponse](#bytebase-v1-ListBackupsResponse) |  |
| ListSlowQueries | [ListSlowQueriesRequest](#bytebase-v1-ListSlowQueriesRequest) | [ListSlowQueriesResponse](#bytebase-v1-ListSlowQueriesResponse) |  |
| ListSecrets | [ListSecretsRequest](#bytebase-v1-ListSecretsRequest) | [ListSecretsResponse](#bytebase-v1-ListSecretsResponse) |  |
| UpdateSecret | [UpdateSecretRequest](#bytebase-v1-UpdateSecretRequest) | [Secret](#bytebase-v1-Secret) |  |
| DeleteSecret | [DeleteSecretRequest](#bytebase-v1-DeleteSecretRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| AdviseIndex | [AdviseIndexRequest](#bytebase-v1-AdviseIndexRequest) | [AdviseIndexResponse](#bytebase-v1-AdviseIndexResponse) |  |
| ListChangeHistories | [ListChangeHistoriesRequest](#bytebase-v1-ListChangeHistoriesRequest) | [ListChangeHistoriesResponse](#bytebase-v1-ListChangeHistoriesResponse) |  |
| GetChangeHistory | [GetChangeHistoryRequest](#bytebase-v1-GetChangeHistoryRequest) | [ChangeHistory](#bytebase-v1-ChangeHistory) |  |

 



<a name="v1_environment_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/environment_service.proto



<a name="bytebase-v1-CreateEnvironmentRequest"></a>

### CreateEnvironmentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| environment | [Environment](#bytebase-v1-Environment) |  | The environment to create. |
| environment_id | [string](#string) |  | The ID to use for the environment, which will become the final component of the environment&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |






<a name="bytebase-v1-DeleteEnvironmentRequest"></a>

### DeleteEnvironmentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the environment to delete. Format: environments/{environment} |






<a name="bytebase-v1-Environment"></a>

### Environment



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the environment. Format: environments/{environment} |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| state | [State](#bytebase-v1-State) |  |  |
| title | [string](#string) |  |  |
| order | [int32](#int32) |  |  |
| tier | [EnvironmentTier](#bytebase-v1-EnvironmentTier) |  |  |






<a name="bytebase-v1-EnvironmentBackupSetting"></a>

### EnvironmentBackupSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the environment backup setting. Format: environments/{environment}/backupSetting |
| enabled | [bool](#bool) |  |  |






<a name="bytebase-v1-GetEnvironmentRequest"></a>

### GetEnvironmentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the environment to retrieve. Format: environments/{environment} |






<a name="bytebase-v1-ListEnvironmentsRequest"></a>

### ListEnvironmentsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of environments to return. The service may return fewer than this value. If unspecified, at most 50 environments will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListEnvironments` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListEnvironments` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Show deleted environments if specified. |






<a name="bytebase-v1-ListEnvironmentsResponse"></a>

### ListEnvironmentsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| environments | [Environment](#bytebase-v1-Environment) | repeated | The environments from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-UndeleteEnvironmentRequest"></a>

### UndeleteEnvironmentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the deleted environment. Format: environments/{environment} |






<a name="bytebase-v1-UpdateEnvironmentBackupSettingRequest"></a>

### UpdateEnvironmentBackupSettingRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| setting | [EnvironmentBackupSetting](#bytebase-v1-EnvironmentBackupSetting) |  | The environment backup setting to update. |






<a name="bytebase-v1-UpdateEnvironmentRequest"></a>

### UpdateEnvironmentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| environment | [Environment](#bytebase-v1-Environment) |  | The environment to update.

The environment&#39;s `name` field is used to identify the environment to update. Format: environments/{environment} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 


<a name="bytebase-v1-EnvironmentTier"></a>

### EnvironmentTier


| Name | Number | Description |
| ---- | ------ | ----------- |
| ENVIRONMENT_TIER_UNSPECIFIED | 0 |  |
| PROTECTED | 1 |  |
| UNPROTECTED | 2 |  |


 

 


<a name="bytebase-v1-EnvironmentService"></a>

### EnvironmentService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetEnvironment | [GetEnvironmentRequest](#bytebase-v1-GetEnvironmentRequest) | [Environment](#bytebase-v1-Environment) |  |
| ListEnvironments | [ListEnvironmentsRequest](#bytebase-v1-ListEnvironmentsRequest) | [ListEnvironmentsResponse](#bytebase-v1-ListEnvironmentsResponse) |  |
| CreateEnvironment | [CreateEnvironmentRequest](#bytebase-v1-CreateEnvironmentRequest) | [Environment](#bytebase-v1-Environment) |  |
| UpdateEnvironment | [UpdateEnvironmentRequest](#bytebase-v1-UpdateEnvironmentRequest) | [Environment](#bytebase-v1-Environment) |  |
| DeleteEnvironment | [DeleteEnvironmentRequest](#bytebase-v1-DeleteEnvironmentRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| UndeleteEnvironment | [UndeleteEnvironmentRequest](#bytebase-v1-UndeleteEnvironmentRequest) | [Environment](#bytebase-v1-Environment) |  |
| UpdateBackupSetting | [UpdateEnvironmentBackupSettingRequest](#bytebase-v1-UpdateEnvironmentBackupSettingRequest) | [EnvironmentBackupSetting](#bytebase-v1-EnvironmentBackupSetting) |  |

 



<a name="v1_externalvs_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/externalvs_service.proto



<a name="bytebase-v1-CreateExternalVersionControlRequest"></a>

### CreateExternalVersionControlRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| external_version_control | [ExternalVersionControl](#bytebase-v1-ExternalVersionControl) |  |  |






<a name="bytebase-v1-DeleteExternalVersionControlRequest"></a>

### DeleteExternalVersionControlRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the external version control to delete. Format: externalVersionControls/{externalVersionControl} |






<a name="bytebase-v1-ExchangeToken"></a>

### ExchangeToken



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the external version control to retrieve the linked projects. Format: externalVersionControls/{externalVersionControl} |
| code | [string](#string) |  |  |
| type | [ExternalVersionControl.Type](#bytebase-v1-ExternalVersionControl-Type) |  |  |
| instance_url | [string](#string) |  |  |
| client_id | [string](#string) |  |  |
| client_secret | [string](#string) |  |  |






<a name="bytebase-v1-ExchangeTokenRequest"></a>

### ExchangeTokenRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| exchange_token | [ExchangeToken](#bytebase-v1-ExchangeToken) |  |  |






<a name="bytebase-v1-ExternalVersionControl"></a>

### ExternalVersionControl



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the external version control, generated by the server, and also is the unique identifier. Format: externalVersionControls/{externalVersionControl} |
| title | [string](#string) |  | The title of the external version control. It is used to display in the UI. Specified by the client. |
| type | [ExternalVersionControl.Type](#bytebase-v1-ExternalVersionControl-Type) |  |  |
| url | [string](#string) |  | The url of the external version control. Specified by the client. For example: github.com, gitlab.com, gitlab.bytebase.com. |
| api_url | [string](#string) |  | The API url of the external version control. Derived from the url. |
| application_id | [string](#string) |  | The application ID of the external version control. Specified by the client. |
| secret | [string](#string) |  | The application secret of the external version control. Specified by the client. |






<a name="bytebase-v1-GetExternalVersionControlRequest"></a>

### GetExternalVersionControlRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the external version control to retrieve. Format: externalVersionControls/{externalVersionControl} |






<a name="bytebase-v1-ListExternalVersionControlsRequest"></a>

### ListExternalVersionControlsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Not used. The maximum number of external version control to return. The service may return fewer than this value. If unspecified, at most 100 external version control will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListExternalVersionControls` call. Provide this to retrieve the subsequent page. |






<a name="bytebase-v1-ListExternalVersionControlsResponse"></a>

### ListExternalVersionControlsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| external_version_controls | [ExternalVersionControl](#bytebase-v1-ExternalVersionControl) | repeated | The list of external version control. |
| next_page_token | [string](#string) |  | Not used. A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-ListProjectGitOpsInfoRequest"></a>

### ListProjectGitOpsInfoRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the external version control to retrieve the linked projects. Format: externalVersionControls/{externalVersionControl} |






<a name="bytebase-v1-ListProjectGitOpsInfoResponse"></a>

### ListProjectGitOpsInfoResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_gitops_info | [ProjectGitOpsInfo](#bytebase-v1-ProjectGitOpsInfo) | repeated | The binding between external version control and project. |






<a name="bytebase-v1-OAuthToken"></a>

### OAuthToken



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| access_token | [string](#string) |  |  |
| refresh_token | [string](#string) |  |  |
| expires_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="bytebase-v1-ProjectGitOpsInfo"></a>

### ProjectGitOpsInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the linked repository, generated by the server, and also is the unique identifier. Format: projects/{project}/gitOpsInfo |
| vcs_uid | [string](#string) |  | The uid for related VCS. |
| title | [string](#string) |  | The title of the repository. For axample: sample. |
| full_path | [string](#string) |  | The full_path of the repository. For example: bytebase/sample. |
| web_url | [string](#string) |  | The web url of the repository. For axample: https://gitlab.bytebase.com/bytebase/sample. |
| branch_filter | [string](#string) |  | The branch Bytebase listens to for changes. For example: main. |
| base_directory | [string](#string) |  | The root directory where Bytebase observes the file change. If empty, then it observes the entire repository. |
| file_path_template | [string](#string) |  | Bytebase only observes the file path name matching the template pattern **relative** to the base directory. Required Placeholder: {{DB_NAME}}, {{VERSION}}, {{TYPE]}. Optional Placeholder: {{ENV_ID}}, {{DESCRIPTION}}. Optional Directory Wildcard: &#39;*&#39;, &#39;**&#39;. |
| schema_path_template | [string](#string) |  | The file path template for storing the latest schema auto-generated by Bytebase after migration. If empty, then Bytebase won&#39;t auto generate it. If specified, required placeholder: {{DB_NAME}}, optional placeholder: {{ENV_ID}}. |
| sheet_path_template | [string](#string) |  | The file path template for matching the sql files for sheet. If specified, required Placeholder: {{NAME}}, optional Placeholder: {{ENV_ID}}, {{DB_NAME}}. |
| external_id | [string](#string) |  | The reposition external id in target VCS. |
| enable_sql_review_ci | [bool](#bool) |  | Set to true to enable SQL review CI for all PR/MRs. |
| webhook_endpoint_id | [string](#string) |  | The webhook endpoint ID of the repository. |
| access_token | [string](#string) |  |  |
| expires_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| refresh_token | [string](#string) |  |  |






<a name="bytebase-v1-SearchExternalVersionControlProjectsRequest"></a>

### SearchExternalVersionControlProjectsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the external version control to retrieve the external version control repositories. Format: externalVersionControls/{externalVersionControl} |
| access_token | [string](#string) |  |  |
| refresh_token | [string](#string) |  |  |






<a name="bytebase-v1-SearchExternalVersionControlProjectsResponse"></a>

### SearchExternalVersionControlProjectsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| projects | [SearchExternalVersionControlProjectsResponse.Project](#bytebase-v1-SearchExternalVersionControlProjectsResponse-Project) | repeated | The list of project in external version control. |






<a name="bytebase-v1-SearchExternalVersionControlProjectsResponse-Project"></a>

### SearchExternalVersionControlProjectsResponse.Project



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | The id of the project in external version control. |
| title | [string](#string) |  | The title of the project in external version control. |
| fullpath | [string](#string) |  | The fullpath of the project in external version control. |
| web_url | [string](#string) |  | Web url of the project in external version control. |






<a name="bytebase-v1-UpdateExternalVersionControlRequest"></a>

### UpdateExternalVersionControlRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| external_version_control | [ExternalVersionControl](#bytebase-v1-ExternalVersionControl) |  |  |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to be updated. |





 


<a name="bytebase-v1-ExternalVersionControl-Type"></a>

### ExternalVersionControl.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| GITHUB | 1 | GitHub type. Using for GitHub community edition(ce). |
| GITLAB | 2 | GitLab type. Using for GitLab community edition(ce) and enterprise edition(ee). |
| BITBUCKET | 3 | BitBucket type. Using for BitBucket cloud or BitBucket server. |
| AZURE_DEVOPS | 4 | Azure DevOps. Using for Azure DevOps GitOps workflow. |


 

 


<a name="bytebase-v1-ExternalVersionControlService"></a>

### ExternalVersionControlService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetExternalVersionControl | [GetExternalVersionControlRequest](#bytebase-v1-GetExternalVersionControlRequest) | [ExternalVersionControl](#bytebase-v1-ExternalVersionControl) |  |
| ListExternalVersionControls | [ListExternalVersionControlsRequest](#bytebase-v1-ListExternalVersionControlsRequest) | [ListExternalVersionControlsResponse](#bytebase-v1-ListExternalVersionControlsResponse) |  |
| CreateExternalVersionControl | [CreateExternalVersionControlRequest](#bytebase-v1-CreateExternalVersionControlRequest) | [ExternalVersionControl](#bytebase-v1-ExternalVersionControl) |  |
| UpdateExternalVersionControl | [UpdateExternalVersionControlRequest](#bytebase-v1-UpdateExternalVersionControlRequest) | [ExternalVersionControl](#bytebase-v1-ExternalVersionControl) |  |
| ExchangeToken | [ExchangeTokenRequest](#bytebase-v1-ExchangeTokenRequest) | [OAuthToken](#bytebase-v1-OAuthToken) |  |
| DeleteExternalVersionControl | [DeleteExternalVersionControlRequest](#bytebase-v1-DeleteExternalVersionControlRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| SearchExternalVersionControlProjects | [SearchExternalVersionControlProjectsRequest](#bytebase-v1-SearchExternalVersionControlProjectsRequest) | [SearchExternalVersionControlProjectsResponse](#bytebase-v1-SearchExternalVersionControlProjectsResponse) |  |
| ListProjectGitOpsInfo | [ListProjectGitOpsInfoRequest](#bytebase-v1-ListProjectGitOpsInfoRequest) | [ListProjectGitOpsInfoResponse](#bytebase-v1-ListProjectGitOpsInfoResponse) |  |

 



<a name="v1_idp_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/idp_service.proto



<a name="bytebase-v1-CreateIdentityProviderRequest"></a>

### CreateIdentityProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identity_provider | [IdentityProvider](#bytebase-v1-IdentityProvider) |  | The identity provider to create. |
| identity_provider_id | [string](#string) |  | The ID to use for the identity provider, which will become the final component of the identity provider&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |






<a name="bytebase-v1-DeleteIdentityProviderRequest"></a>

### DeleteIdentityProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the identity provider to delete. Format: idps/{identity_provider} |






<a name="bytebase-v1-FieldMapping"></a>

### FieldMapping
FieldMapping saves the field names from user info API of identity provider.
As we save all raw json string of user info response data into `principal.idp_user_info`,
we can extract the relevant data based with `FieldMapping`.

e.g. For GitHub authenticated user API, it will return `login`, `name` and `email` in response.
Then the identifier of FieldMapping will be `login`, display_name will be `name`,
and email will be `email`.
reference: https://docs.github.com/en/rest/users/users?apiVersion=2022-11-28#get-the-authenticated-user


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identifier | [string](#string) |  | Identifier is the field name of the unique identifier in 3rd-party idp user info. Required. |
| display_name | [string](#string) |  | DisplayName is the field name of display name in 3rd-party idp user info. |
| email | [string](#string) |  | Email is the field name of primary email in 3rd-party idp user info. |
| phone | [string](#string) |  | Phone is the field name of primary phone in 3rd-party idp user info. |






<a name="bytebase-v1-GetIdentityProviderRequest"></a>

### GetIdentityProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |






<a name="bytebase-v1-IdentityProvider"></a>

### IdentityProvider



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the identity provider. Format: idps/{identity_provider} |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| state | [State](#bytebase-v1-State) |  |  |
| title | [string](#string) |  |  |
| domain | [string](#string) |  |  |
| type | [IdentityProviderType](#bytebase-v1-IdentityProviderType) |  |  |
| config | [IdentityProviderConfig](#bytebase-v1-IdentityProviderConfig) |  |  |






<a name="bytebase-v1-IdentityProviderConfig"></a>

### IdentityProviderConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| oauth2_config | [OAuth2IdentityProviderConfig](#bytebase-v1-OAuth2IdentityProviderConfig) |  |  |
| oidc_config | [OIDCIdentityProviderConfig](#bytebase-v1-OIDCIdentityProviderConfig) |  |  |
| ldap_config | [LDAPIdentityProviderConfig](#bytebase-v1-LDAPIdentityProviderConfig) |  |  |






<a name="bytebase-v1-LDAPIdentityProviderConfig"></a>

### LDAPIdentityProviderConfig
LDAPIdentityProviderConfig is the structure for LDAP identity provider config.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| host | [string](#string) |  | Host is the hostname or IP address of the LDAP server, e.g. &#34;ldap.example.com&#34;. |
| port | [int32](#int32) |  | Port is the port number of the LDAP server, e.g. 389. When not set, the default port of the corresponding security protocol will be used, i.e. 389 for StartTLS and 636 for LDAPS. |
| skip_tls_verify | [bool](#bool) |  | SkipTLSVerify controls whether to skip TLS certificate verification. |
| bind_dn | [string](#string) |  | BindDN is the DN of the user to bind as a service account to perform search requests. |
| bind_password | [string](#string) |  | BindPassword is the password of the user to bind as a service account. |
| base_dn | [string](#string) |  | BaseDN is the base DN to search for users, e.g. &#34;ou=users,dc=example,dc=com&#34;. |
| user_filter | [string](#string) |  | UserFilter is the filter to search for users, e.g. &#34;(uid=%s)&#34;. |
| security_protocol | [string](#string) |  | SecurityProtocol is the security protocol to be used for establishing connections with the LDAP server. It should be either StartTLS or LDAPS, and cannot be empty. |
| field_mapping | [FieldMapping](#bytebase-v1-FieldMapping) |  | FieldMapping is the mapping of the user attributes returned by the LDAP server. |






<a name="bytebase-v1-ListIdentityProvidersRequest"></a>

### ListIdentityProvidersRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of identity providers to return. The service may return fewer than this value. If unspecified, at most 50 will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListIdentityProviders` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListIdentityProviders` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Show deleted identity providers if specified. |






<a name="bytebase-v1-ListIdentityProvidersResponse"></a>

### ListIdentityProvidersResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identity_providers | [IdentityProvider](#bytebase-v1-IdentityProvider) | repeated | The identity providers from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-OAuth2IdentityProviderConfig"></a>

### OAuth2IdentityProviderConfig
OAuth2IdentityProviderConfig is the structure for OAuth2 identity provider config.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| auth_url | [string](#string) |  |  |
| token_url | [string](#string) |  |  |
| user_info_url | [string](#string) |  |  |
| client_id | [string](#string) |  |  |
| client_secret | [string](#string) |  |  |
| scopes | [string](#string) | repeated |  |
| field_mapping | [FieldMapping](#bytebase-v1-FieldMapping) |  |  |
| skip_tls_verify | [bool](#bool) |  |  |
| auth_style | [OAuth2AuthStyle](#bytebase-v1-OAuth2AuthStyle) |  |  |






<a name="bytebase-v1-OAuth2IdentityProviderTestRequestContext"></a>

### OAuth2IdentityProviderTestRequestContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [string](#string) |  | Authorize code from website. |






<a name="bytebase-v1-OIDCIdentityProviderConfig"></a>

### OIDCIdentityProviderConfig
OIDCIdentityProviderConfig is the structure for OIDC identity provider config.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issuer | [string](#string) |  |  |
| client_id | [string](#string) |  |  |
| client_secret | [string](#string) |  |  |
| scopes | [string](#string) | repeated |  |
| field_mapping | [FieldMapping](#bytebase-v1-FieldMapping) |  |  |
| skip_tls_verify | [bool](#bool) |  |  |
| auth_style | [OAuth2AuthStyle](#bytebase-v1-OAuth2AuthStyle) |  |  |






<a name="bytebase-v1-TestIdentityProviderRequest"></a>

### TestIdentityProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identity_provider | [IdentityProvider](#bytebase-v1-IdentityProvider) |  | The identity provider to test connection including uncreated. |
| oauth2_context | [OAuth2IdentityProviderTestRequestContext](#bytebase-v1-OAuth2IdentityProviderTestRequestContext) |  |  |






<a name="bytebase-v1-TestIdentityProviderResponse"></a>

### TestIdentityProviderResponse







<a name="bytebase-v1-UndeleteIdentityProviderRequest"></a>

### UndeleteIdentityProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the deleted identity provider. Format: idps/{identity_provider} |






<a name="bytebase-v1-UpdateIdentityProviderRequest"></a>

### UpdateIdentityProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identity_provider | [IdentityProvider](#bytebase-v1-IdentityProvider) |  | The identity provider to update.

The identity provider&#39;s `name` field is used to identify the identity provider to update. Format: idps/{identity_provider} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 


<a name="bytebase-v1-IdentityProviderType"></a>

### IdentityProviderType


| Name | Number | Description |
| ---- | ------ | ----------- |
| IDENTITY_PROVIDER_TYPE_UNSPECIFIED | 0 |  |
| OAUTH2 | 1 |  |
| OIDC | 2 |  |
| LDAP | 3 |  |



<a name="bytebase-v1-OAuth2AuthStyle"></a>

### OAuth2AuthStyle


| Name | Number | Description |
| ---- | ------ | ----------- |
| OAUTH2_AUTH_STYLE_UNSPECIFIED | 0 |  |
| IN_PARAMS | 1 | IN_PARAMS sends the &#34;client_id&#34; and &#34;client_secret&#34; in the POST body as application/x-www-form-urlencoded parameters. |
| IN_HEADER | 2 | IN_HEADER sends the client_id and client_password using HTTP Basic Authorization. This is an optional style described in the OAuth2 RFC 6749 section 2.3.1. |


 

 


<a name="bytebase-v1-IdentityProviderService"></a>

### IdentityProviderService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetIdentityProvider | [GetIdentityProviderRequest](#bytebase-v1-GetIdentityProviderRequest) | [IdentityProvider](#bytebase-v1-IdentityProvider) |  |
| ListIdentityProviders | [ListIdentityProvidersRequest](#bytebase-v1-ListIdentityProvidersRequest) | [ListIdentityProvidersResponse](#bytebase-v1-ListIdentityProvidersResponse) |  |
| CreateIdentityProvider | [CreateIdentityProviderRequest](#bytebase-v1-CreateIdentityProviderRequest) | [IdentityProvider](#bytebase-v1-IdentityProvider) |  |
| UpdateIdentityProvider | [UpdateIdentityProviderRequest](#bytebase-v1-UpdateIdentityProviderRequest) | [IdentityProvider](#bytebase-v1-IdentityProvider) |  |
| DeleteIdentityProvider | [DeleteIdentityProviderRequest](#bytebase-v1-DeleteIdentityProviderRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| UndeleteIdentityProvider | [UndeleteIdentityProviderRequest](#bytebase-v1-UndeleteIdentityProviderRequest) | [IdentityProvider](#bytebase-v1-IdentityProvider) |  |
| TestIdentityProvider | [TestIdentityProviderRequest](#bytebase-v1-TestIdentityProviderRequest) | [TestIdentityProviderResponse](#bytebase-v1-TestIdentityProviderResponse) |  |

 



<a name="v1_logging_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/logging_service.proto



<a name="bytebase-v1-GetLogRequest"></a>

### GetLogRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the log to retrieve. Format: logs/{uid} |






<a name="bytebase-v1-ListLogsRequest"></a>

### ListLogsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filter | [string](#string) |  | filter is the filter to apply on the list logs request, follow the [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) syntax. The field only support in filter: - creator, example: - creator = &#34;users/{email}&#34; - resource, example: - resource = &#34;projects/{project resource id}&#34; - level, example: - level = &#34;INFO&#34; - level = &#34;ERROR | WARN&#34; - action, example: - action = &#34;ACTION_MEMBER_CREATE&#34; | &#34;ACTION_ISSUE_CREATE&#34; - create_time, example: - create_time &lt;= &#34;2022-01-01T12:00:00.000Z&#34; - create_time &gt;= &#34;2022-01-01T12:00:00.000Z&#34; For example: List the logs of type &#39;ACTION_ISSUE_COMMENT_CREATE&#39; in issue/123: &#39;action=&#34;ACTION_ISSUE_COMMENT_CREATE&#34;, resource=&#34;issue/123&#34;&#39; |
| order_by | [string](#string) |  | The order by of the log. Only support order by create_time. For example: - order_by = &#34;create_time asc&#34; - order_by = &#34;create_time desc&#34; |
| page_size | [int32](#int32) |  | Not used. The maximum number of logs to return. The service may return fewer than this value. If unspecified, at most 100 log entries will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListLogs` call. Provide this to retrieve the subsequent page. |






<a name="bytebase-v1-ListLogsResponse"></a>

### ListLogsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| log_entities | [LogEntity](#bytebase-v1-LogEntity) | repeated | The list of log entities. |
| next_page_token | [string](#string) |  | A token to retrieve next page of log entities. Pass this value in the page_token field in the subsequent call to `ListLogs` method to retrieve the next page of log entities. |






<a name="bytebase-v1-LogEntity"></a>

### LogEntity



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the log. Format: logs/{uid} |
| creator | [string](#string) |  | The creator of the log entity. Format: users/{email} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| action | [LogEntity.Action](#bytebase-v1-LogEntity-Action) |  |  |
| level | [LogEntity.Level](#bytebase-v1-LogEntity-Level) |  |  |
| resource | [string](#string) |  | The name of the resource associated with this log entity. For example, the resource user associated with log entity type of &#34;ACTION_MEMBER_CREATE&#34;. Format: For ACTION_MEMBER_*: users/{email} For ACTION_ISSUE_*: issues/{issue uid} For ACTION_PIPELINE_*: pipelines/{pipeline uid} For ACTION_PROJECT_*: projects/{project resource id} For ACTION_DATABASE_*: instances/{instance resource id} |
| payload | [string](#string) |  | The payload of the log entity. TODO: use oneof |
| comment | [string](#string) |  |  |





 


<a name="bytebase-v1-LogEntity-Action"></a>

### LogEntity.Action


| Name | Number | Description |
| ---- | ------ | ----------- |
| ACTION_UNSPECIFIED | 0 | In worksapce resource only. |
| ACTION_MEMBER_CREATE | 1 | Member related activity types. Enum value 1 - 20

ACTION_MEMBER_CREATE is the type for creating a new member. |
| ACTION_MEMBER_ROLE_UPDATE | 2 | ACTION_MEMBER_ROLE_UPDATE is the type for updating a member&#39;s role. |
| ACTION_MEMBER_ACTIVATE | 3 | ACTION_MEMBER_ACTIVATE_UPDATE is the type for activating members. |
| ACTION_MEMBER_DEACTIVE | 4 | ACTION_MEMBER_DEACTIVE is the type for deactiving members. |
| ACTION_ISSUE_CREATE | 21 | Issue related activity types. Enum value 21 - 40

ACTION_ISSUE_CREATE is the type for creating a new issue. |
| ACTION_ISSUE_COMMENT_CREATE | 22 | ACTION_ISSUE_COMMENT_CREATE is the type for creating a new comment on an issue. |
| ACTION_ISSUE_FIELD_UPDATE | 23 | ACTION_ISSUE_FIELD_UPDATE is the type for updating an issue&#39;s field. |
| ACTION_ISSUE_STATUS_UPDATE | 24 | ACTION_ISSUE_STATUS_UPDATE is the type for updating an issue&#39;s status. |
| ACTION_ISSUE_APPROVAL_NOTIFY | 25 | ACTION_ISSUE_APPROVAL_NOTIFY is the type for notifying issue approval. |
| ACTION_PIPELINE_STAGE_STATUS_UPDATE | 31 | ACTION_PIPELINE_STAGE_STATUS_UPDATE represents the pipeline stage status change, including BEGIN, END for now. |
| ACTION_PIPELINE_TASK_STATUS_UPDATE | 32 | ACTION_PIPELINE_TASK_STATUS_UPDATE represents the pipeline task status change, including PENDING, PENDING_APPROVAL, RUNNING, SUCCESS, FAILURE, CANCELED for now. |
| ACTION_PIPELINE_TASK_FILE_COMMIT | 33 | ACTION_PIPELINE_TASK_FILE_COMMIT represents the VCS trigger to commit a file to update the task statement. |
| ACTION_PIPELINE_TASK_STATEMENT_UPDATE | 34 | ACTION_PIPELINE_TASK_STATEMENT_UPDATE represents the manual update of the task statement. |
| ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE | 35 | ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE represents the manual update of the task earliest allowed time. |
| ACTION_PIPELINE_TASK_RUN_STATUS_UPDATE | 36 | ACTION_PIPELINE_TASK_RUN_STATUS_UPDATE represents the pipeline task run status change, including PENDING, RUNNING, SUCCESS, FAILURE, CANCELED for now. |
| ACTION_PROJECT_REPOSITORY_PUSH | 41 | Project related activity types. Enum value 41 - 60

ACTION_PROJECT_REPOSITORY_PUSH represents Bytebase receiving a push event from the project repository. |
| ACTION_PROJECT_MEMBER_CREATE | 42 | ACTION_PROJECT_MEMBER_CREATE represents adding a member to the project. |
| ACTION_PROJECT_MEMBER_DELETE | 43 | ACTION_PROJECT_MEMBER_DELETE represents removing a member from the project. |
| ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE | 44 | ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE is the type for database PITR recovery done. |
| ACTION_PROJECT_DATABASE_TRANSFER | 45 | ACTION_PROJECT_DATABASE_TRANSFER represents transfering the database from one project to another. |
| ACTION_DATABASE_SQL_EDITOR_QUERY | 61 | Database related activity types. Enum value 61 - 80

ACTION_DATABASE_SQL_EDITOR_QUERY is the type for SQL editor query. |
| ACTION_DATABASE_SQL_EXPORT | 62 | ACTION_DATABASE_SQL_EXPORT is the type for exporting SQL. |



<a name="bytebase-v1-LogEntity-Level"></a>

### LogEntity.Level


| Name | Number | Description |
| ---- | ------ | ----------- |
| LEVEL_UNSPECIFIED | 0 |  |
| LEVEL_INFO | 1 | LEVEL_INFO is the type for information. |
| LEVEL_WARNING | 2 | LEVEL_WARNING is the type for warning. |
| LEVEL_ERROR | 3 | LEVEL_ERROR is the type for error. |


 

 


<a name="bytebase-v1-LoggingService"></a>

### LoggingService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListLogs | [ListLogsRequest](#bytebase-v1-ListLogsRequest) | [ListLogsResponse](#bytebase-v1-ListLogsResponse) |  |
| GetLog | [GetLogRequest](#bytebase-v1-GetLogRequest) | [LogEntity](#bytebase-v1-LogEntity) |  |

 



<a name="v1_inbox_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/inbox_service.proto



<a name="bytebase-v1-GetInboxSummaryRequest"></a>

### GetInboxSummaryRequest







<a name="bytebase-v1-InboxMessage"></a>

### InboxMessage



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The message name in inbox/{uid} format. |
| activity_uid | [string](#string) |  |  |
| status | [InboxMessage.Status](#bytebase-v1-InboxMessage-Status) |  |  |
| activity | [LogEntity](#bytebase-v1-LogEntity) |  |  |






<a name="bytebase-v1-InboxSummary"></a>

### InboxSummary



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| unread | [int32](#int32) |  |  |
| unread_error | [int32](#int32) |  |  |






<a name="bytebase-v1-ListInboxRequest"></a>

### ListInboxRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filter | [string](#string) |  | filter is the filter to apply on the list inbox request, follow the [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) syntax. The field only support in filter: - &#34;create_time&#34; with &#34;&gt;=&#34; operator, example: - create_time &gt;= &#34;2022-01-01T12:00:00.000Z&#34; |
| page_size | [int32](#int32) |  | Not used. The maximum number of inbox to return. The service may return fewer than this value. If unspecified, at most 100 log entries will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListInbox` call. Provide this to retrieve the subsequent page. |






<a name="bytebase-v1-ListInboxResponse"></a>

### ListInboxResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| inbox_messages | [InboxMessage](#bytebase-v1-InboxMessage) | repeated | The list of inbox messages. |
| next_page_token | [string](#string) |  | A token to retrieve next page of inbox. Pass this value in the page_token field in the subsequent call to `ListLogs` method to retrieve the next page of log entities. |






<a name="bytebase-v1-UpdateInboxRequest"></a>

### UpdateInboxRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| inbox_message | [InboxMessage](#bytebase-v1-InboxMessage) |  | The inbox message to update. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 


<a name="bytebase-v1-InboxMessage-Status"></a>

### InboxMessage.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| STATUS_UNREAD | 1 |  |
| STATUS_READ | 2 |  |


 

 


<a name="bytebase-v1-InboxService"></a>

### InboxService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListInbox | [ListInboxRequest](#bytebase-v1-ListInboxRequest) | [ListInboxResponse](#bytebase-v1-ListInboxResponse) |  |
| GetInboxSummary | [GetInboxSummaryRequest](#bytebase-v1-GetInboxSummaryRequest) | [InboxSummary](#bytebase-v1-InboxSummary) |  |
| UpdateInbox | [UpdateInboxRequest](#bytebase-v1-UpdateInboxRequest) | [InboxMessage](#bytebase-v1-InboxMessage) |  |

 



<a name="v1_instance_role_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/instance_role_service.proto



<a name="bytebase-v1-CreateInstanceRoleRequest"></a>

### CreateInstanceRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this role will be created. Format: instances/{instance} |
| role | [InstanceRole](#bytebase-v1-InstanceRole) |  | The role to create. |






<a name="bytebase-v1-DeleteInstanceRoleRequest"></a>

### DeleteInstanceRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the role to delete. Format: instances/{instance}/roles/{role name} |






<a name="bytebase-v1-GetInstanceRoleRequest"></a>

### GetInstanceRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the role to retrieve. Format: instances/{instance}/roles/{role name} The role name is the unique name for the role. |






<a name="bytebase-v1-InstanceRole"></a>

### InstanceRole
InstanceRole is the API message for instance role.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the role. Format: instances/{instance}/roles/{role name} The role name is the unique name for the role. |
| role_name | [string](#string) |  | The role name. It&#39;s unique within the instance. |
| password | [string](#string) | optional | The role password. |
| connection_limit | [int32](#int32) | optional | The connection count limit for this role. |
| valid_until | [string](#string) | optional | The expiration for the role&#39;s password. |
| attribute | [string](#string) | optional | The role attribute. For PostgreSQL, it containt super_user, no_inherit, create_role, create_db, can_login, replication and bypass_rls. Docs: https://www.postgresql.org/docs/current/role-attributes.html For MySQL, it&#39;s the global privileges as GRANT statements, which means it only contains &#34;GRANT ... ON *.* TO ...&#34;. Docs: https://dev.mysql.com/doc/refman/8.0/en/grant.html |






<a name="bytebase-v1-ListInstanceRolesRequest"></a>

### ListInstanceRolesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of roles. Format: instances/{instance} |
| page_size | [int32](#int32) |  | The maximum number of roles to return. The service may return fewer than this value. If unspecified, at most 50 roles will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListRoles` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListRoles` must match the call that provided the page token. |
| refresh | [bool](#bool) |  | Refresh will refresh and return the latest data. |






<a name="bytebase-v1-ListInstanceRolesResponse"></a>

### ListInstanceRolesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| roles | [InstanceRole](#bytebase-v1-InstanceRole) | repeated | The roles from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-UndeleteInstanceRoleRequest"></a>

### UndeleteInstanceRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the deleted role. Format: instances/{instance}/roles/{role name} |






<a name="bytebase-v1-UpdateInstanceRoleRequest"></a>

### UpdateInstanceRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [InstanceRole](#bytebase-v1-InstanceRole) |  | The role to update.

The role&#39;s `name` and `instance` field is used to identify the role to update. Format: instances/{instance}/roles/{role name} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 

 

 


<a name="bytebase-v1-InstanceRoleService"></a>

### InstanceRoleService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetInstanceRole | [GetInstanceRoleRequest](#bytebase-v1-GetInstanceRoleRequest) | [InstanceRole](#bytebase-v1-InstanceRole) |  |
| ListInstanceRoles | [ListInstanceRolesRequest](#bytebase-v1-ListInstanceRolesRequest) | [ListInstanceRolesResponse](#bytebase-v1-ListInstanceRolesResponse) |  |
| CreateInstanceRole | [CreateInstanceRoleRequest](#bytebase-v1-CreateInstanceRoleRequest) | [InstanceRole](#bytebase-v1-InstanceRole) |  |
| UpdateInstanceRole | [UpdateInstanceRoleRequest](#bytebase-v1-UpdateInstanceRoleRequest) | [InstanceRole](#bytebase-v1-InstanceRole) |  |
| DeleteInstanceRole | [DeleteInstanceRoleRequest](#bytebase-v1-DeleteInstanceRoleRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| UndeleteInstanceRole | [UndeleteInstanceRoleRequest](#bytebase-v1-UndeleteInstanceRoleRequest) | [InstanceRole](#bytebase-v1-InstanceRole) |  |

 



<a name="v1_instance_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/instance_service.proto



<a name="bytebase-v1-AddDataSourceRequest"></a>

### AddDataSourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance | [string](#string) |  | The name of the instance to add a data source to. Format: instances/{instance} |
| data_source | [DataSource](#bytebase-v1-DataSource) |  | Identified by type. Only READ_ONLY data source can be added. |
| validate_only | [bool](#bool) |  | Validate only also tests the data source connection. |






<a name="bytebase-v1-CreateInstanceRequest"></a>

### CreateInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance | [Instance](#bytebase-v1-Instance) |  | The instance to create. |
| instance_id | [string](#string) |  | The ID to use for the instance, which will become the final component of the instance&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |
| validate_only | [bool](#bool) |  | Validate only also tests the data source connection. |






<a name="bytebase-v1-DataSource"></a>

### DataSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| type | [DataSourceType](#bytebase-v1-DataSourceType) |  |  |
| username | [string](#string) |  |  |
| password | [string](#string) |  |  |
| ssl_ca | [string](#string) |  |  |
| ssl_cert | [string](#string) |  |  |
| ssl_key | [string](#string) |  |  |
| host | [string](#string) |  |  |
| port | [string](#string) |  |  |
| database | [string](#string) |  |  |
| srv | [bool](#bool) |  | srv and authentication_database are used for MongoDB. |
| authentication_database | [string](#string) |  |  |
| sid | [string](#string) |  | sid and service_name are used for Oracle. |
| service_name | [string](#string) |  |  |
| ssh_host | [string](#string) |  | Connection over SSH. The hostname of the SSH server agent. Required. |
| ssh_port | [string](#string) |  | The port of the SSH server agent. It&#39;s 22 typically. Required. |
| ssh_user | [string](#string) |  | The user to login the server. Required. |
| ssh_password | [string](#string) |  | The password to login the server. If it&#39;s empty string, no password is required. |
| ssh_private_key | [string](#string) |  | The private key to login the server. If it&#39;s empty string, we will use the system default private key from os.Getenv(&#34;SSH_AUTH_SOCK&#34;). |






<a name="bytebase-v1-DeleteInstanceRequest"></a>

### DeleteInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the instance to delete. Format: instances/{instance} |
| force | [bool](#bool) |  | If set to true, any databases and sheets from this project will also be moved to default project, and all open issues will be closed. |






<a name="bytebase-v1-GetInstanceRequest"></a>

### GetInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the instance to retrieve. Format: instances/{instance} |






<a name="bytebase-v1-Instance"></a>

### Instance



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the instance. Format: instances/{instance} |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| state | [State](#bytebase-v1-State) |  |  |
| title | [string](#string) |  |  |
| engine | [Engine](#bytebase-v1-Engine) |  |  |
| engine_version | [string](#string) |  |  |
| external_link | [string](#string) |  |  |
| data_sources | [DataSource](#bytebase-v1-DataSource) | repeated |  |
| environment | [string](#string) |  | The environment resource. Format: environments/prod where prod is the environment resource ID. |
| activation | [bool](#bool) |  |  |
| options | [InstanceOptions](#bytebase-v1-InstanceOptions) |  |  |






<a name="bytebase-v1-InstanceOptions"></a>

### InstanceOptions
InstanceOptions is the option for instances.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema_tenant_mode | [bool](#bool) |  | The schema tenant mode is used to determine whether the instance is in schema tenant mode. For Oracle schema tenant mode, the instance a Oracle database and the database is the Oracle schema. |






<a name="bytebase-v1-ListInstancesRequest"></a>

### ListInstancesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of instances to return. The service may return fewer than this value. If unspecified, at most 50 instances will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListInstances` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListInstances` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Show deleted instances if specified. |






<a name="bytebase-v1-ListInstancesResponse"></a>

### ListInstancesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instances | [Instance](#bytebase-v1-Instance) | repeated | The instances from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-RemoveDataSourceRequest"></a>

### RemoveDataSourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance | [string](#string) |  | The name of the instance to remove a data source from. Format: instances/{instance} |
| data_source | [DataSource](#bytebase-v1-DataSource) |  | Identified by type. Only READ_ONLY data source can be removed. |






<a name="bytebase-v1-SyncInstanceRequest"></a>

### SyncInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of instance. Format: instances/{instance} |






<a name="bytebase-v1-SyncInstanceResponse"></a>

### SyncInstanceResponse







<a name="bytebase-v1-SyncSlowQueriesRequest"></a>

### SyncSlowQueriesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The name of the instance to sync slow queries. Format: instances/{instance} for one instance or projects/{project} for one project. |






<a name="bytebase-v1-UndeleteInstanceRequest"></a>

### UndeleteInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the deleted instance. Format: instances/{instance} |






<a name="bytebase-v1-UpdateDataSourceRequest"></a>

### UpdateDataSourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance | [string](#string) |  | The name of the instance to update a data source. Format: instances/{instance} |
| data_source | [DataSource](#bytebase-v1-DataSource) |  | Identified by type. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| validate_only | [bool](#bool) |  | Validate only also tests the data source connection. |






<a name="bytebase-v1-UpdateInstanceRequest"></a>

### UpdateInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance | [Instance](#bytebase-v1-Instance) |  | The instance to update.

The instance&#39;s `name` field is used to identify the instance to update. Format: instances/{instance} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 


<a name="bytebase-v1-DataSourceType"></a>

### DataSourceType


| Name | Number | Description |
| ---- | ------ | ----------- |
| DATA_SOURCE_UNSPECIFIED | 0 |  |
| ADMIN | 1 |  |
| READ_ONLY | 2 |  |


 

 


<a name="bytebase-v1-InstanceService"></a>

### InstanceService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetInstance | [GetInstanceRequest](#bytebase-v1-GetInstanceRequest) | [Instance](#bytebase-v1-Instance) |  |
| ListInstances | [ListInstancesRequest](#bytebase-v1-ListInstancesRequest) | [ListInstancesResponse](#bytebase-v1-ListInstancesResponse) |  |
| CreateInstance | [CreateInstanceRequest](#bytebase-v1-CreateInstanceRequest) | [Instance](#bytebase-v1-Instance) |  |
| UpdateInstance | [UpdateInstanceRequest](#bytebase-v1-UpdateInstanceRequest) | [Instance](#bytebase-v1-Instance) |  |
| DeleteInstance | [DeleteInstanceRequest](#bytebase-v1-DeleteInstanceRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| UndeleteInstance | [UndeleteInstanceRequest](#bytebase-v1-UndeleteInstanceRequest) | [Instance](#bytebase-v1-Instance) |  |
| SyncInstance | [SyncInstanceRequest](#bytebase-v1-SyncInstanceRequest) | [SyncInstanceResponse](#bytebase-v1-SyncInstanceResponse) |  |
| AddDataSource | [AddDataSourceRequest](#bytebase-v1-AddDataSourceRequest) | [Instance](#bytebase-v1-Instance) |  |
| RemoveDataSource | [RemoveDataSourceRequest](#bytebase-v1-RemoveDataSourceRequest) | [Instance](#bytebase-v1-Instance) |  |
| UpdateDataSource | [UpdateDataSourceRequest](#bytebase-v1-UpdateDataSourceRequest) | [Instance](#bytebase-v1-Instance) |  |
| SyncSlowQueries | [SyncSlowQueriesRequest](#bytebase-v1-SyncSlowQueriesRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="v1_issue_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/issue_service.proto



<a name="bytebase-v1-ApprovalFlow"></a>

### ApprovalFlow



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| steps | [ApprovalStep](#bytebase-v1-ApprovalStep) | repeated |  |






<a name="bytebase-v1-ApprovalNode"></a>

### ApprovalNode



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [ApprovalNode.Type](#bytebase-v1-ApprovalNode-Type) |  |  |
| group_value | [ApprovalNode.GroupValue](#bytebase-v1-ApprovalNode-GroupValue) |  |  |
| role | [string](#string) |  | Format: roles/{role} |
| external_node_id | [string](#string) |  |  |






<a name="bytebase-v1-ApprovalStep"></a>

### ApprovalStep



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [ApprovalStep.Type](#bytebase-v1-ApprovalStep-Type) |  |  |
| nodes | [ApprovalNode](#bytebase-v1-ApprovalNode) | repeated |  |






<a name="bytebase-v1-ApprovalTemplate"></a>

### ApprovalTemplate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| flow | [ApprovalFlow](#bytebase-v1-ApprovalFlow) |  |  |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| creator | [string](#string) |  | The name of the creator in users/{email} format. TODO: we should mark it as OUTPUT_ONLY, but currently the frontend will post the approval setting with creator. |






<a name="bytebase-v1-ApproveIssueRequest"></a>

### ApproveIssueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the issue to add an approver. Format: projects/{project}/issues/{issue} |
| comment | [string](#string) |  |  |






<a name="bytebase-v1-BatchUpdateIssuesStatusRequest"></a>

### BatchUpdateIssuesStatusRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource shared by all issues being updated. Format: projects/{project} If the operation spans parents, a dash (-) may be accepted as a wildcard. We only support updating the status of databases for now. |
| issues | [string](#string) | repeated | The list of issues to update. Format: projects/{project}/issues/{issue} |
| status | [IssueStatus](#bytebase-v1-IssueStatus) |  | The new status. |
| reason | [string](#string) |  |  |






<a name="bytebase-v1-BatchUpdateIssuesStatusResponse"></a>

### BatchUpdateIssuesStatusResponse







<a name="bytebase-v1-CreateIssueCommentRequest"></a>

### CreateIssueCommentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The issue name Format: projects/{project}/issues/{issue} |
| issue_comment | [IssueComment](#bytebase-v1-IssueComment) |  |  |






<a name="bytebase-v1-CreateIssueRequest"></a>

### CreateIssueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of issues. Format: projects/{project} |
| issue | [Issue](#bytebase-v1-Issue) |  | The issue to create. |






<a name="bytebase-v1-GetIssueRequest"></a>

### GetIssueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the issue to retrieve. Format: projects/{project}/issues/{issue} |
| force | [bool](#bool) |  |  |






<a name="bytebase-v1-Issue"></a>

### Issue



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the issue. Format: projects/{project}/issues/{issue} |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| type | [Issue.Type](#bytebase-v1-Issue-Type) |  |  |
| status | [IssueStatus](#bytebase-v1-IssueStatus) |  |  |
| assignee | [string](#string) |  | Format: users/hello@world.com |
| assignee_attention | [bool](#bool) |  |  |
| approvers | [Issue.Approver](#bytebase-v1-Issue-Approver) | repeated |  |
| approval_templates | [ApprovalTemplate](#bytebase-v1-ApprovalTemplate) | repeated |  |
| approval_finding_done | [bool](#bool) |  | If the value is `false`, it means that the backend is still finding matching approval templates. If `true`, approval_templates &amp; approvers &amp; approval_finding_error are available. |
| approval_finding_error | [string](#string) |  |  |
| subscribers | [string](#string) | repeated | The subscribers. Format: users/hello@world.com |
| creator | [string](#string) |  | Format: users/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| plan | [string](#string) |  | The plan associated with the issue. Can be empty. Format: projects/{project}/plans/{plan} |
| rollout | [string](#string) |  | The rollout associated with the issue. Can be empty. Format: projects/{project}/rollouts/{rollout} |






<a name="bytebase-v1-Issue-Approver"></a>

### Issue.Approver



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [Issue.Approver.Status](#bytebase-v1-Issue-Approver-Status) |  | The new status. |
| principal | [string](#string) |  | Format: users/hello@world.com |






<a name="bytebase-v1-IssueComment"></a>

### IssueComment



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| uid | [string](#string) |  |  |
| comment | [string](#string) |  |  |
| payload | [string](#string) |  | TODO: use struct message instead. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="bytebase-v1-ListIssuesRequest"></a>

### ListIssuesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of issues. Format: projects/{project} Use &#34;projects/-&#34; to list all issues from all projects. |
| page_size | [int32](#int32) |  | The maximum number of issues to return. The service may return fewer than this value. If unspecified, at most 50 issues will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListIssues` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListIssues` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter issues returned in the list. |






<a name="bytebase-v1-ListIssuesResponse"></a>

### ListIssuesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issues | [Issue](#bytebase-v1-Issue) | repeated | The issues from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-RejectIssueRequest"></a>

### RejectIssueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the issue to add an rejection. Format: projects/{project}/issues/{issue} |
| comment | [string](#string) |  |  |






<a name="bytebase-v1-RequestIssueRequest"></a>

### RequestIssueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the issue to request a issue. Format: projects/{project}/issues/{issue} |
| comment | [string](#string) |  |  |






<a name="bytebase-v1-SearchIssuesRequest"></a>

### SearchIssuesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of issues. Format: projects/{project}. Use &#34;projects/-&#34; to search all issues. |
| page_size | [int32](#int32) |  | The maximum number of issues to return. The service may return fewer than this value. If unspecified, at most 50 issues will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `SearchIssues` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `SearchIssues` must match the call that provided the page token. |
| query | [string](#string) |  | Query is the query statement. |
| filter | [string](#string) |  | Filter is used to filter issues returned in the list. |






<a name="bytebase-v1-SearchIssuesResponse"></a>

### SearchIssuesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issues | [Issue](#bytebase-v1-Issue) | repeated | The issues from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-UpdateIssueCommentRequest"></a>

### UpdateIssueCommentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The issue name Format: projects/{project}/issues/{issue} |
| issue_comment | [IssueComment](#bytebase-v1-IssueComment) |  |  |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






<a name="bytebase-v1-UpdateIssueRequest"></a>

### UpdateIssueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issue | [Issue](#bytebase-v1-Issue) |  | The issue to update.

The issue&#39;s `name` field is used to identify the issue to update. Format: projects/{project}/issues/{issue} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 


<a name="bytebase-v1-ApprovalNode-GroupValue"></a>

### ApprovalNode.GroupValue
The predefined user groups are:
- WORKSPACE_OWNER
- WORKSPACE_DBA
- PROJECT_OWNER
- PROJECT_MEMBER

| Name | Number | Description |
| ---- | ------ | ----------- |
| GROUP_VALUE_UNSPECIFILED | 0 |  |
| WORKSPACE_OWNER | 1 |  |
| WORKSPACE_DBA | 2 |  |
| PROJECT_OWNER | 3 |  |
| PROJECT_MEMBER | 4 |  |



<a name="bytebase-v1-ApprovalNode-Type"></a>

### ApprovalNode.Type
Type of the ApprovalNode.
type determines who should approve this node.
ANY_IN_GROUP means the ApprovalNode can be approved by an user from our predefined user group.
See GroupValue below for the predefined user groups.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| ANY_IN_GROUP | 1 |  |



<a name="bytebase-v1-ApprovalStep-Type"></a>

### ApprovalStep.Type
Type of the ApprovalStep
ALL means every node must be approved to proceed.
ANY means approving any node will proceed.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| ALL | 1 |  |
| ANY | 2 |  |



<a name="bytebase-v1-Issue-Approver-Status"></a>

### Issue.Approver.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| APPROVED | 2 |  |
| REJECTED | 3 |  |



<a name="bytebase-v1-Issue-Type"></a>

### Issue.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| DATABASE_CHANGE | 1 |  |
| GRANT_REQUEST | 2 |  |



<a name="bytebase-v1-IssueStatus"></a>

### IssueStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| ISSUE_STATUS_UNSPECIFIED | 0 |  |
| OPEN | 1 |  |
| DONE | 2 |  |
| CANCELED | 3 |  |


 

 


<a name="bytebase-v1-IssueService"></a>

### IssueService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetIssue | [GetIssueRequest](#bytebase-v1-GetIssueRequest) | [Issue](#bytebase-v1-Issue) |  |
| CreateIssue | [CreateIssueRequest](#bytebase-v1-CreateIssueRequest) | [Issue](#bytebase-v1-Issue) |  |
| ListIssues | [ListIssuesRequest](#bytebase-v1-ListIssuesRequest) | [ListIssuesResponse](#bytebase-v1-ListIssuesResponse) |  |
| UpdateIssue | [UpdateIssueRequest](#bytebase-v1-UpdateIssueRequest) | [Issue](#bytebase-v1-Issue) |  |
| SearchIssues | [SearchIssuesRequest](#bytebase-v1-SearchIssuesRequest) | [SearchIssuesResponse](#bytebase-v1-SearchIssuesResponse) |  |
| CreateIssueComment | [CreateIssueCommentRequest](#bytebase-v1-CreateIssueCommentRequest) | [IssueComment](#bytebase-v1-IssueComment) |  |
| UpdateIssueComment | [UpdateIssueCommentRequest](#bytebase-v1-UpdateIssueCommentRequest) | [IssueComment](#bytebase-v1-IssueComment) |  |
| BatchUpdateIssuesStatus | [BatchUpdateIssuesStatusRequest](#bytebase-v1-BatchUpdateIssuesStatusRequest) | [BatchUpdateIssuesStatusResponse](#bytebase-v1-BatchUpdateIssuesStatusResponse) |  |
| ApproveIssue | [ApproveIssueRequest](#bytebase-v1-ApproveIssueRequest) | [Issue](#bytebase-v1-Issue) |  |
| RejectIssue | [RejectIssueRequest](#bytebase-v1-RejectIssueRequest) | [Issue](#bytebase-v1-Issue) |  |
| RequestIssue | [RequestIssueRequest](#bytebase-v1-RequestIssueRequest) | [Issue](#bytebase-v1-Issue) |  |

 



<a name="v1_project_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/project_service.proto



<a name="bytebase-v1-Activity"></a>

### Activity
TODO(zp): move to activity later.






<a name="bytebase-v1-AddWebhookRequest"></a>

### AddWebhookRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The name of the project to add the webhook to. Format: projects/{project} |
| webhook | [Webhook](#bytebase-v1-Webhook) |  | The webhook to add. |






<a name="bytebase-v1-BatchGetIamPolicyRequest"></a>

### BatchGetIamPolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| scope | [string](#string) |  | The scope of the batch get. Typically it&#39;s &#34;projects/-&#34;. |
| names | [string](#string) | repeated |  |






<a name="bytebase-v1-BatchGetIamPolicyResponse"></a>

### BatchGetIamPolicyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy_results | [BatchGetIamPolicyResponse.PolicyResult](#bytebase-v1-BatchGetIamPolicyResponse-PolicyResult) | repeated |  |






<a name="bytebase-v1-BatchGetIamPolicyResponse-PolicyResult"></a>

### BatchGetIamPolicyResponse.PolicyResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| policy | [IamPolicy](#bytebase-v1-IamPolicy) |  |  |






<a name="bytebase-v1-CreateDatabaseGroupRequest"></a>

### CreateDatabaseGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this database group will be created. Format: projects/{project} |
| database_group | [DatabaseGroup](#bytebase-v1-DatabaseGroup) |  | The database group to create. |
| database_group_id | [string](#string) |  | The ID to use for the database group, which will become the final component of the database group&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |
| validate_only | [bool](#bool) |  | If set, validate the create request and preview the full database group response, but do not actually create it. |






<a name="bytebase-v1-CreateProjectRequest"></a>

### CreateProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [Project](#bytebase-v1-Project) |  | The project to create. |
| project_id | [string](#string) |  | The ID to use for the project, which will become the final component of the project&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |






<a name="bytebase-v1-CreateSchemaGroupRequest"></a>

### CreateSchemaGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this schema group will be created. Format: projects/{project}/databaseGroups/{databaseGroup} |
| schema_group | [SchemaGroup](#bytebase-v1-SchemaGroup) |  | The schema group to create. |
| schema_group_id | [string](#string) |  | The ID to use for the schema group, which will become the final component of the schema group&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |
| validate_only | [bool](#bool) |  | If set, validate the create request and preview the full schema group response, but do not actually create it. |






<a name="bytebase-v1-DatabaseGroup"></a>

### DatabaseGroup



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database group. Format: projects/{project}/databaseGroups/{databaseGroup} |
| database_placeholder | [string](#string) |  | The short name used in actual databases specified by users. For example, the placeholder for db1_2010, db1_2021, db1_2023 will be &#34;db1&#34;. |
| database_expr | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this database group. |
| matched_databases | [DatabaseGroup.Database](#bytebase-v1-DatabaseGroup-Database) | repeated | The list of databases that match the database group condition. |
| unmatched_databases | [DatabaseGroup.Database](#bytebase-v1-DatabaseGroup-Database) | repeated | The list of databases that match the database group condition. |






<a name="bytebase-v1-DatabaseGroup-Database"></a>

### DatabaseGroup.Database



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The resource name of the database. Format: instances/{instance}/databases/{database} |






<a name="bytebase-v1-DeleteDatabaseGroupRequest"></a>

### DeleteDatabaseGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database group to delete. Format: projects/{project}/databaseGroups/{databaseGroup} |






<a name="bytebase-v1-DeleteProjectRequest"></a>

### DeleteProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the project to delete. Format: projects/{project} |
| force | [bool](#bool) |  | If set to true, any databases and sheets from this project will also be moved to default project, and all open issues will be closed. |






<a name="bytebase-v1-DeleteSchemaGroupRequest"></a>

### DeleteSchemaGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the schema group to delete. Format: projects/{project}/databaseGroups/{databaseGroup}/schemaGroups/{schemaGroup} |






<a name="bytebase-v1-DeploymentConfig"></a>

### DeploymentConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the resource. Format: projects/{project}/deploymentConfig |
| title | [string](#string) |  | The title of the deployment config. |
| schedule | [Schedule](#bytebase-v1-Schedule) |  |  |






<a name="bytebase-v1-DeploymentSpec"></a>

### DeploymentSpec



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| label_selector | [LabelSelector](#bytebase-v1-LabelSelector) |  |  |






<a name="bytebase-v1-GetDatabaseGroupRequest"></a>

### GetDatabaseGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database group to retrieve. Format: projects/{project}/databaseGroups/{databaseGroup} |
| view | [DatabaseGroupView](#bytebase-v1-DatabaseGroupView) |  | The view to return. Defaults to DATABASE_GROUP_VIEW_BASIC. |






<a name="bytebase-v1-GetDeploymentConfigRequest"></a>

### GetDeploymentConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the resource. Format: projects/{project}/deploymentConfig |






<a name="bytebase-v1-GetIamPolicyRequest"></a>

### GetIamPolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The name of the project to get the IAM policy. Format: projects/{project} |






<a name="bytebase-v1-GetProjectGitOpsInfoRequest"></a>

### GetProjectGitOpsInfoRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the GitOps info. Format: projects/{project}/gitOpsInfo |






<a name="bytebase-v1-GetProjectRequest"></a>

### GetProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the project to retrieve. Format: projects/{project} |






<a name="bytebase-v1-GetSchemaGroupRequest"></a>

### GetSchemaGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database group to retrieve. Format: projects/{project}/databaseGroups/{databaseGroup}/schemaGroups/{schemaGroup} |
| view | [SchemaGroupView](#bytebase-v1-SchemaGroupView) |  | The view to return. Defaults to SCHEMA_GROUP_VIEW_BASIC. |






<a name="bytebase-v1-LabelSelector"></a>

### LabelSelector



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| match_expressions | [LabelSelectorRequirement](#bytebase-v1-LabelSelectorRequirement) | repeated |  |






<a name="bytebase-v1-LabelSelectorRequirement"></a>

### LabelSelectorRequirement



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| operator | [OperatorType](#bytebase-v1-OperatorType) |  |  |
| values | [string](#string) | repeated |  |






<a name="bytebase-v1-ListDatabaseGroupsRequest"></a>

### ListDatabaseGroupsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource whose database groups are to be listed. Format: projects/{project} Using &#34;projects/-&#34; will list database groups across all projects. |
| page_size | [int32](#int32) |  | Not used. The maximum number of anomalies to return. The service may return fewer than this value. If unspecified, at most 50 anomalies will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListDatabaseGroups` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListDatabaseGroups` must match the call that provided the page token. |






<a name="bytebase-v1-ListDatabaseGroupsResponse"></a>

### ListDatabaseGroupsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database_groups | [DatabaseGroup](#bytebase-v1-DatabaseGroup) | repeated | database_groups is the list of database groups. |
| next_page_token | [string](#string) |  | Not used. A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-ListProjectsRequest"></a>

### ListProjectsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of projects to return. The service may return fewer than this value. If unspecified, at most 50 projects will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListProjects` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListProjects` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Show deleted projects if specified. |






<a name="bytebase-v1-ListProjectsResponse"></a>

### ListProjectsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| projects | [Project](#bytebase-v1-Project) | repeated | The projects from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-ListSchemaGroupsRequest"></a>

### ListSchemaGroupsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource whose schema groups are to be listed. Format: projects/{project}/schemaGroups/{schemaGroup} |
| page_size | [int32](#int32) |  | Not used. The maximum number of anomalies to return. The service may return fewer than this value. If unspecified, at most 50 anomalies will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListSchemaGroups` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListSchemaGroups` must match the call that provided the page token. |






<a name="bytebase-v1-ListSchemaGroupsResponse"></a>

### ListSchemaGroupsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema_groups | [SchemaGroup](#bytebase-v1-SchemaGroup) | repeated | schema_groups is the list of schema groups. |
| next_page_token | [string](#string) |  | Not used. A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-Project"></a>

### Project



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the project. Format: projects/{project} |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| state | [State](#bytebase-v1-State) |  |  |
| title | [string](#string) |  | The title or name of a project. It&#39;s not unique within the workspace. |
| key | [string](#string) |  | The key is a short and upper-case identifier for a project. It&#39;s unique within the workspace. |
| workflow | [Workflow](#bytebase-v1-Workflow) |  |  |
| visibility | [Visibility](#bytebase-v1-Visibility) |  |  |
| tenant_mode | [TenantMode](#bytebase-v1-TenantMode) |  |  |
| db_name_template | [string](#string) |  |  |
| schema_change | [SchemaChange](#bytebase-v1-SchemaChange) |  |  |
| webhooks | [Webhook](#bytebase-v1-Webhook) | repeated |  |
| data_classification_config_id | [string](#string) |  |  |






<a name="bytebase-v1-RemoveWebhookRequest"></a>

### RemoveWebhookRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| webhook | [Webhook](#bytebase-v1-Webhook) |  | The webhook to remove. Identified by its url. |






<a name="bytebase-v1-Schedule"></a>

### Schedule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| deployments | [ScheduleDeployment](#bytebase-v1-ScheduleDeployment) | repeated |  |






<a name="bytebase-v1-ScheduleDeployment"></a>

### ScheduleDeployment



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  | The title of the deployment (stage) in a schedule. |
| spec | [DeploymentSpec](#bytebase-v1-DeploymentSpec) |  |  |






<a name="bytebase-v1-SchemaGroup"></a>

### SchemaGroup



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the schema group. Format: projects/{project}/databaseGroups/{databaseGroup}/schemaGroups/{schemaGroup} |
| table_expr | [google.type.Expr](#google-type-Expr) |  | The table condition that is associated with this schema group. The table_placeholder in the sheet script will be rendered to the actual table name. |
| table_placeholder | [string](#string) |  | The table placeholder used for rendering. For example, if set to &#34;tbl&#34;, all the table name &#34;tbl&#34; in the SQL script will be rendered to the actual table name. |
| matched_tables | [SchemaGroup.Table](#bytebase-v1-SchemaGroup-Table) | repeated | The list of databases that match the database group condition. |
| unmatched_tables | [SchemaGroup.Table](#bytebase-v1-SchemaGroup-Table) | repeated | The list of databases that match the database group condition. |






<a name="bytebase-v1-SchemaGroup-Table"></a>

### SchemaGroup.Table
In the future, we can introduce schema_expr if users use schema (Postgres schema) for groups.
Its keyword will be {{SCHEMA}}.
All the expressions will be used to filter the schema objects in DatabaseSchema.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [string](#string) |  | The resource name of the database. Format: instances/{instance}/databases/{database} |
| schema | [string](#string) |  |  |
| table | [string](#string) |  |  |






<a name="bytebase-v1-SearchProjectsRequest"></a>

### SearchProjectsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of projects to return. The service may return fewer than this value. If unspecified, at most 50 projects will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListProjects` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListProjects` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter projects returned in the list. |






<a name="bytebase-v1-SearchProjectsResponse"></a>

### SearchProjectsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| projects | [Project](#bytebase-v1-Project) | repeated | The projects from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-SetIamPolicyRequest"></a>

### SetIamPolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The name of the project to set the IAM policy. Format: projects/{project} |
| policy | [IamPolicy](#bytebase-v1-IamPolicy) |  |  |






<a name="bytebase-v1-SetupSQLReviewCIRequest"></a>

### SetupSQLReviewCIRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the GitOps info. Format: projects/{project}/gitOpsInfo |






<a name="bytebase-v1-SetupSQLReviewCIResponse"></a>

### SetupSQLReviewCIResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pull_request_url | [string](#string) |  | The CI setup PR URL for the repository. |






<a name="bytebase-v1-TestWebhookRequest"></a>

### TestWebhookRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The name of the project which owns the webhook to test. Format: projects/{project} |
| webhook | [Webhook](#bytebase-v1-Webhook) |  | The webhook to test. Identified by its url. |






<a name="bytebase-v1-TestWebhookResponse"></a>

### TestWebhookResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| error | [string](#string) |  | The result of the test, empty if the test is successful. |






<a name="bytebase-v1-UndeleteProjectRequest"></a>

### UndeleteProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the deleted project. Format: projects/{project} |






<a name="bytebase-v1-UnsetProjectGitOpsInfoRequest"></a>

### UnsetProjectGitOpsInfoRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the GitOps info. Format: projects/{project}/gitOpsInfo |






<a name="bytebase-v1-UpdateDatabaseGroupRequest"></a>

### UpdateDatabaseGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database_group | [DatabaseGroup](#bytebase-v1-DatabaseGroup) |  | The database group to update.

The database group&#39;s `name` field is used to identify the database group to update. Format: projects/{project}/databaseGroups/{databaseGroup} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






<a name="bytebase-v1-UpdateDeploymentConfigRequest"></a>

### UpdateDeploymentConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| config | [DeploymentConfig](#bytebase-v1-DeploymentConfig) |  |  |






<a name="bytebase-v1-UpdateProjectGitOpsInfoRequest"></a>

### UpdateProjectGitOpsInfoRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project_gitops_info | [ProjectGitOpsInfo](#bytebase-v1-ProjectGitOpsInfo) |  | The binding for the project and external version control. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The mask of the fields to be updated. |
| allow_missing | [bool](#bool) |  | If true, the gitops will be created if it does not exist. |






<a name="bytebase-v1-UpdateProjectRequest"></a>

### UpdateProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [Project](#bytebase-v1-Project) |  | The project to update.

The project&#39;s `name` field is used to identify the project to update. Format: projects/{project} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






<a name="bytebase-v1-UpdateSchemaGroupRequest"></a>

### UpdateSchemaGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema_group | [SchemaGroup](#bytebase-v1-SchemaGroup) |  | The schema group to update.

The schema group&#39;s `name` field is used to identify the schema group to update. Format: projects/{project}/databaseGroups/{databaseGroup}/schemaGroups/{schemaGroup} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






<a name="bytebase-v1-UpdateWebhookRequest"></a>

### UpdateWebhookRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| webhook | [Webhook](#bytebase-v1-Webhook) |  | The webhook to modify. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






<a name="bytebase-v1-Webhook"></a>

### Webhook



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | name is the name of the webhook, generated by the server. format: projects/{project}/webhooks/{webhook} |
| type | [Webhook.Type](#bytebase-v1-Webhook-Type) |  | type is the type of the webhook. |
| title | [string](#string) |  | title is the title of the webhook. |
| url | [string](#string) |  | url is the url of the webhook, should be unique within the project. |
| notification_types | [Activity.Type](#bytebase-v1-Activity-Type) | repeated | notification_types is the list of activities types that the webhook is interested in. Bytebase will only send notifications to the webhook if the activity type is in the list. It should not be empty, and shoule be a subset of the following: - TYPE_ISSUE_CREATED - TYPE_ISSUE_STATUS_UPDATE - TYPE_ISSUE_PIPELINE_STAGE_UPDATE - TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE - TYPE_ISSUE_FIELD_UPDATE - TYPE_ISSUE_COMMENT_CREAT |





 


<a name="bytebase-v1-Activity-Type"></a>

### Activity.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| TYPE_ISSUE_CREATE | 1 | Issue related activity types.

TYPE_ISSUE_CREATE represents creating an issue. |
| TYPE_ISSUE_COMMENT_CREATE | 2 | TYPE_ISSUE_COMMENT_CREATE represents commenting on an issue. |
| TYPE_ISSUE_FIELD_UPDATE | 3 | TYPE_ISSUE_FIELD_UPDATE represents updating the issue field, likes title, description, assignee, etc. |
| TYPE_ISSUE_STATUS_UPDATE | 4 | TYPE_ISSUE_STATUS_UPDATE represents the issue status change, including OPEN, CLOSE, CANCEL fow now. |
| TYPE_ISSUE_APPROVAL_NOTIFY | 21 | TYPE_ISSUE_APPROVAL_NOTIFY is the type for notifying issue approval. |
| TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE | 5 | TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE represents the pipeline stage status change, including BEGIN, END for now. |
| TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE | 6 | TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE represents the pipeline task status change, including PENDING, PENDING_APPROVAL, RUNNING, SUCCESS, FAILURE, CANCELED for now. |
| TYPE_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE | 22 | TYPE_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE represents the pipeline task run status change, including PENDING, RUNNING, DONE, FAILED, CANCELED. |
| TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT | 7 | TYPE_ISSUE_PIPELINE_TASK_FILE_COMMIT represents the VCS trigger to commit a file to update the task statement. |
| TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE | 8 | TYPE_ISSUE_PIPELINE_TASK_STATEMENT_UPDATE represents the manual update of the task statement. |
| TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE | 9 | TYPE_ISSUE_PIPELINE_TASK_EARLIEST_ALLOWED_TIME_UPDATE represents the manual update of the task earliest allowed time. |
| TYPE_MEMBER_CREATE | 10 | Member related activity types.

TYPE_MEMBER_CREATE represents creating a members. |
| TYPE_MEMBER_ROLE_UPDATE | 11 | TYPE_MEMBER_ROLE_UPDATE represents updating the member role, for example, from ADMIN to MEMBER. |
| TYPE_MEMBER_ACTIVATE | 12 | TYPE_MEMBER_ACTIVATE represents activating a deactivated member. |
| TYPE_MEMBER_DEACTIVATE | 13 | TYPE_MEMBER_DEACTIVATE represents deactivating an active member. |
| TYPE_PROJECT_REPOSITORY_PUSH | 14 | Project related activity types.

TYPE_PROJECT_REPOSITORY_PUSH represents Bytebase receiving a push event from the project repository. |
| TYPE_PROJECT_DATABASE_TRANSFER | 15 | TYPE_PROJECT_DATABASE_TRANFER represents transfering the database from one project to another. |
| TYPE_PROJECT_MEMBER_CREATE | 16 | TYPE_PROJECT_MEMBER_CREATE represents adding a member to the project. |
| TYPE_PROJECT_MEMBER_DELETE | 17 | TYPE_PROJECT_MEMBER_DELETE represents removing a member from the project. |
| TYPE_SQL_EDITOR_QUERY | 19 | SQL Editor related activity types. TYPE_SQL_EDITOR_QUERY represents executing query in SQL Editor. |
| TYPE_DATABASE_RECOVERY_PITR_DONE | 20 | Database related activity types. TYPE_DATABASE_RECOVERY_PITR_DONE represents the database recovery to a point in time is done. |



<a name="bytebase-v1-DatabaseGroupView"></a>

### DatabaseGroupView


| Name | Number | Description |
| ---- | ------ | ----------- |
| DATABASE_GROUP_VIEW_UNSPECIFIED | 0 | The default / unset value. The API will default to the BASIC view. |
| DATABASE_GROUP_VIEW_BASIC | 1 | Include basic information about the database group, but exclude the list of matched databases and unmatched databases. |
| DATABASE_GROUP_VIEW_FULL | 2 | Include everything. |



<a name="bytebase-v1-OperatorType"></a>

### OperatorType


| Name | Number | Description |
| ---- | ------ | ----------- |
| OPERATOR_TYPE_UNSPECIFIED | 0 | The operator is not specified. |
| OPERATOR_TYPE_IN | 1 | The operator is &#34;In&#34;. |
| OPERATOR_TYPE_EXISTS | 2 | The operator is &#34;Exists&#34;. |



<a name="bytebase-v1-SchemaChange"></a>

### SchemaChange


| Name | Number | Description |
| ---- | ------ | ----------- |
| SCHEMA_CHANGE_UNSPECIFIED | 0 |  |
| DDL | 1 |  |
| SDL | 2 |  |



<a name="bytebase-v1-SchemaGroupView"></a>

### SchemaGroupView


| Name | Number | Description |
| ---- | ------ | ----------- |
| SCHEMA_GROUP_VIEW_UNSPECIFIED | 0 | The default / unset value. The API will default to the BASIC view. |
| SCHEMA_GROUP_VIEW_BASIC | 1 | Include basic information about the schema group, but exclude the list of matched tables and unmatched tables. |
| SCHEMA_GROUP_VIEW_FULL | 2 | Include everything. |



<a name="bytebase-v1-SchemaVersion"></a>

### SchemaVersion


| Name | Number | Description |
| ---- | ------ | ----------- |
| SCHEMA_VERSION_UNSPECIFIED | 0 |  |
| TIMESTAMP | 1 |  |
| SEMANTIC | 2 |  |



<a name="bytebase-v1-TenantMode"></a>

### TenantMode


| Name | Number | Description |
| ---- | ------ | ----------- |
| TENANT_MODE_UNSPECIFIED | 0 |  |
| TENANT_MODE_DISABLED | 1 |  |
| TENANT_MODE_ENABLED | 2 |  |



<a name="bytebase-v1-Visibility"></a>

### Visibility


| Name | Number | Description |
| ---- | ------ | ----------- |
| VISIBILITY_UNSPECIFIED | 0 |  |
| VISIBILITY_PUBLIC | 1 |  |
| VISIBILITY_PRIVATE | 2 |  |



<a name="bytebase-v1-Webhook-Type"></a>

### Webhook.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| TYPE_SLACK | 1 |  |
| TYPE_DISCORD | 2 |  |
| TYPE_TEAMS | 3 |  |
| TYPE_DINGTALK | 4 |  |
| TYPE_FEISHU | 5 |  |
| TYPE_WECOM | 6 |  |
| TYPE_CUSTOM | 7 |  |



<a name="bytebase-v1-Workflow"></a>

### Workflow


| Name | Number | Description |
| ---- | ------ | ----------- |
| WORKFLOW_UNSPECIFIED | 0 |  |
| UI | 1 |  |
| VCS | 2 |  |


 

 


<a name="bytebase-v1-ProjectService"></a>

### ProjectService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetProject | [GetProjectRequest](#bytebase-v1-GetProjectRequest) | [Project](#bytebase-v1-Project) |  |
| ListProjects | [ListProjectsRequest](#bytebase-v1-ListProjectsRequest) | [ListProjectsResponse](#bytebase-v1-ListProjectsResponse) |  |
| SearchProjects | [SearchProjectsRequest](#bytebase-v1-SearchProjectsRequest) | [SearchProjectsResponse](#bytebase-v1-SearchProjectsResponse) | Search for projects that the caller has both projects.get permission on, and also satisfy the specified query. |
| CreateProject | [CreateProjectRequest](#bytebase-v1-CreateProjectRequest) | [Project](#bytebase-v1-Project) |  |
| UpdateProject | [UpdateProjectRequest](#bytebase-v1-UpdateProjectRequest) | [Project](#bytebase-v1-Project) |  |
| DeleteProject | [DeleteProjectRequest](#bytebase-v1-DeleteProjectRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| UndeleteProject | [UndeleteProjectRequest](#bytebase-v1-UndeleteProjectRequest) | [Project](#bytebase-v1-Project) |  |
| GetIamPolicy | [GetIamPolicyRequest](#bytebase-v1-GetIamPolicyRequest) | [IamPolicy](#bytebase-v1-IamPolicy) |  |
| BatchGetIamPolicy | [BatchGetIamPolicyRequest](#bytebase-v1-BatchGetIamPolicyRequest) | [BatchGetIamPolicyResponse](#bytebase-v1-BatchGetIamPolicyResponse) |  |
| SetIamPolicy | [SetIamPolicyRequest](#bytebase-v1-SetIamPolicyRequest) | [IamPolicy](#bytebase-v1-IamPolicy) |  |
| GetDeploymentConfig | [GetDeploymentConfigRequest](#bytebase-v1-GetDeploymentConfigRequest) | [DeploymentConfig](#bytebase-v1-DeploymentConfig) |  |
| UpdateDeploymentConfig | [UpdateDeploymentConfigRequest](#bytebase-v1-UpdateDeploymentConfigRequest) | [DeploymentConfig](#bytebase-v1-DeploymentConfig) |  |
| AddWebhook | [AddWebhookRequest](#bytebase-v1-AddWebhookRequest) | [Project](#bytebase-v1-Project) |  |
| UpdateWebhook | [UpdateWebhookRequest](#bytebase-v1-UpdateWebhookRequest) | [Project](#bytebase-v1-Project) |  |
| RemoveWebhook | [RemoveWebhookRequest](#bytebase-v1-RemoveWebhookRequest) | [Project](#bytebase-v1-Project) |  |
| TestWebhook | [TestWebhookRequest](#bytebase-v1-TestWebhookRequest) | [TestWebhookResponse](#bytebase-v1-TestWebhookResponse) |  |
| UpdateProjectGitOpsInfo | [UpdateProjectGitOpsInfoRequest](#bytebase-v1-UpdateProjectGitOpsInfoRequest) | [ProjectGitOpsInfo](#bytebase-v1-ProjectGitOpsInfo) |  |
| UnsetProjectGitOpsInfo | [UnsetProjectGitOpsInfoRequest](#bytebase-v1-UnsetProjectGitOpsInfoRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| SetupProjectSQLReviewCI | [SetupSQLReviewCIRequest](#bytebase-v1-SetupSQLReviewCIRequest) | [SetupSQLReviewCIResponse](#bytebase-v1-SetupSQLReviewCIResponse) |  |
| GetProjectGitOpsInfo | [GetProjectGitOpsInfoRequest](#bytebase-v1-GetProjectGitOpsInfoRequest) | [ProjectGitOpsInfo](#bytebase-v1-ProjectGitOpsInfo) |  |
| ListDatabaseGroups | [ListDatabaseGroupsRequest](#bytebase-v1-ListDatabaseGroupsRequest) | [ListDatabaseGroupsResponse](#bytebase-v1-ListDatabaseGroupsResponse) |  |
| GetDatabaseGroup | [GetDatabaseGroupRequest](#bytebase-v1-GetDatabaseGroupRequest) | [DatabaseGroup](#bytebase-v1-DatabaseGroup) |  |
| CreateDatabaseGroup | [CreateDatabaseGroupRequest](#bytebase-v1-CreateDatabaseGroupRequest) | [DatabaseGroup](#bytebase-v1-DatabaseGroup) |  |
| UpdateDatabaseGroup | [UpdateDatabaseGroupRequest](#bytebase-v1-UpdateDatabaseGroupRequest) | [DatabaseGroup](#bytebase-v1-DatabaseGroup) |  |
| DeleteDatabaseGroup | [DeleteDatabaseGroupRequest](#bytebase-v1-DeleteDatabaseGroupRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| ListSchemaGroups | [ListSchemaGroupsRequest](#bytebase-v1-ListSchemaGroupsRequest) | [ListSchemaGroupsResponse](#bytebase-v1-ListSchemaGroupsResponse) |  |
| GetSchemaGroup | [GetSchemaGroupRequest](#bytebase-v1-GetSchemaGroupRequest) | [SchemaGroup](#bytebase-v1-SchemaGroup) |  |
| CreateSchemaGroup | [CreateSchemaGroupRequest](#bytebase-v1-CreateSchemaGroupRequest) | [SchemaGroup](#bytebase-v1-SchemaGroup) |  |
| UpdateSchemaGroup | [UpdateSchemaGroupRequest](#bytebase-v1-UpdateSchemaGroupRequest) | [SchemaGroup](#bytebase-v1-SchemaGroup) |  |
| DeleteSchemaGroup | [DeleteSchemaGroupRequest](#bytebase-v1-DeleteSchemaGroupRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="v1_risk_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/risk_service.proto



<a name="bytebase-v1-CreateRiskRequest"></a>

### CreateRiskRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| risk | [Risk](#bytebase-v1-Risk) |  | The risk to create. |






<a name="bytebase-v1-DeleteRiskRequest"></a>

### DeleteRiskRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the risk to delete. Format: risks/{risk} |






<a name="bytebase-v1-ListRisksRequest"></a>

### ListRisksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of risks to return. The service may return fewer than this value. If unspecified, at most 50 risks will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListRisks` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `LiskRisks` must match the call that provided the page token. |






<a name="bytebase-v1-ListRisksResponse"></a>

### ListRisksResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| risks | [Risk](#bytebase-v1-Risk) | repeated |  |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-Risk"></a>

### Risk



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: risks/{risk} |
| uid | [string](#string) |  | system-generated unique identifier. |
| source | [Risk.Source](#bytebase-v1-Risk-Source) |  |  |
| title | [string](#string) |  |  |
| level | [int64](#int64) |  |  |
| active | [bool](#bool) |  |  |
| condition | [google.type.Expr](#google-type-Expr) |  |  |






<a name="bytebase-v1-UpdateRiskRequest"></a>

### UpdateRiskRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| risk | [Risk](#bytebase-v1-Risk) |  | The risk to update.

The risk&#39;s `name` field is used to identify the risk to update. Format: risks/{risk} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 


<a name="bytebase-v1-Risk-Source"></a>

### Risk.Source


| Name | Number | Description |
| ---- | ------ | ----------- |
| SOURCE_UNSPECIFIED | 0 |  |
| DDL | 1 |  |
| DML | 2 |  |
| CREATE_DATABASE | 3 |  |
| QUERY | 4 |  |
| EXPORT | 5 |  |


 

 


<a name="bytebase-v1-RiskService"></a>

### RiskService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListRisks | [ListRisksRequest](#bytebase-v1-ListRisksRequest) | [ListRisksResponse](#bytebase-v1-ListRisksResponse) |  |
| CreateRisk | [CreateRiskRequest](#bytebase-v1-CreateRiskRequest) | [Risk](#bytebase-v1-Risk) |  |
| UpdateRisk | [UpdateRiskRequest](#bytebase-v1-UpdateRiskRequest) | [Risk](#bytebase-v1-Risk) |  |
| DeleteRisk | [DeleteRiskRequest](#bytebase-v1-DeleteRiskRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="v1_role_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/role_service.proto



<a name="bytebase-v1-CreateRoleRequest"></a>

### CreateRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [Role](#bytebase-v1-Role) |  |  |
| role_id | [string](#string) |  | The ID to use for the role, which will become the final component of the role&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][A-Z][0-9]/. |






<a name="bytebase-v1-DeleteRoleRequest"></a>

### DeleteRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: roles/{role} |






<a name="bytebase-v1-ListRolesRequest"></a>

### ListRolesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of roles to return. The service may return fewer than this value. If unspecified, at most 50 reviews will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListRoles` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListRoles` must match the call that provided the page token. |






<a name="bytebase-v1-ListRolesResponse"></a>

### ListRolesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| roles | [Role](#bytebase-v1-Role) | repeated |  |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-Role"></a>

### Role



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: roles/{role} |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |






<a name="bytebase-v1-UpdateRoleRequest"></a>

### UpdateRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [Role](#bytebase-v1-Role) |  |  |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  |  |





 

 

 


<a name="bytebase-v1-RoleService"></a>

### RoleService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListRoles | [ListRolesRequest](#bytebase-v1-ListRolesRequest) | [ListRolesResponse](#bytebase-v1-ListRolesResponse) |  |
| CreateRole | [CreateRoleRequest](#bytebase-v1-CreateRoleRequest) | [Role](#bytebase-v1-Role) |  |
| UpdateRole | [UpdateRoleRequest](#bytebase-v1-UpdateRoleRequest) | [Role](#bytebase-v1-Role) |  |
| DeleteRole | [DeleteRoleRequest](#bytebase-v1-DeleteRoleRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="v1_rollout_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/rollout_service.proto



<a name="bytebase-v1-BatchCancelTaskRunsRequest"></a>

### BatchCancelTaskRunsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The name of the parent of the taskRuns. Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} Use `projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/-` to cancel task runs under the same stage. |
| task_runs | [string](#string) | repeated | The taskRuns to cancel. Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |
| reason | [string](#string) |  |  |






<a name="bytebase-v1-BatchCancelTaskRunsResponse"></a>

### BatchCancelTaskRunsResponse







<a name="bytebase-v1-BatchRunTasksRequest"></a>

### BatchRunTasksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The name of the parent of the tasks. Format: projects/{project}/rollouts/{rollout}/stages/{stage} |
| tasks | [string](#string) | repeated | The tasks to run. Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} |
| reason | [string](#string) |  |  |






<a name="bytebase-v1-BatchRunTasksResponse"></a>

### BatchRunTasksResponse







<a name="bytebase-v1-BatchSkipTasksRequest"></a>

### BatchSkipTasksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The name of the parent of the tasks. Format: projects/{project}/rollouts/{rollout}/stages/{stage} |
| tasks | [string](#string) | repeated | The tasks to skip. Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} |
| reason | [string](#string) |  |  |






<a name="bytebase-v1-BatchSkipTasksResponse"></a>

### BatchSkipTasksResponse







<a name="bytebase-v1-CreatePlanRequest"></a>

### CreatePlanRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent project where this plan will be created. Format: projects/{project} |
| plan | [Plan](#bytebase-v1-Plan) |  | The plan to create. |






<a name="bytebase-v1-CreateRolloutRequest"></a>

### CreateRolloutRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent project where this rollout will be created. Format: projects/{project} |
| plan | [string](#string) |  | The plan used to create rollout. |






<a name="bytebase-v1-GetPlanRequest"></a>

### GetPlanRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the plan to retrieve. Format: projects/{project}/plans/{plan} |






<a name="bytebase-v1-GetRolloutRequest"></a>

### GetRolloutRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the rollout to retrieve. Format: projects/{project}/rollouts/{rollout} |






<a name="bytebase-v1-ListPlanCheckRunsRequest"></a>

### ListPlanCheckRunsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of plan check runs. Format: projects/{project}/plans/{plan} |
| page_size | [int32](#int32) |  | The maximum number of plan check runs to return. The service may return fewer than this value. If unspecified, at most 50 plans will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListPlanCheckRuns` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListPlanCheckRuns` must match the call that provided the page token. |






<a name="bytebase-v1-ListPlanCheckRunsResponse"></a>

### ListPlanCheckRunsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| plan_check_runs | [PlanCheckRun](#bytebase-v1-PlanCheckRun) | repeated | The plan check runs from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-ListPlansRequest"></a>

### ListPlansRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of plans. Format: projects/{project} Use &#34;projects/-&#34; to list all plans from all projects. |
| page_size | [int32](#int32) |  | The maximum number of plans to return. The service may return fewer than this value. If unspecified, at most 50 plans will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListPlans` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListPlans` must match the call that provided the page token. |






<a name="bytebase-v1-ListPlansResponse"></a>

### ListPlansResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| plans | [Plan](#bytebase-v1-Plan) | repeated | The plans from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-ListTaskRunsRequest"></a>

### ListTaskRunsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of plans. Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} Use &#34;projects/{project}/rollouts/{rollout}/stages/-/tasks/-&#34; to list all taskRuns from a rollout. |
| page_size | [int32](#int32) |  | The maximum number of taskRuns to return. The service may return fewer than this value. If unspecified, at most 50 taskRuns will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListRolloutTaskRuns` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListRolloutTaskRuns` must match the call that provided the page token. |






<a name="bytebase-v1-ListTaskRunsResponse"></a>

### ListTaskRunsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| task_runs | [TaskRun](#bytebase-v1-TaskRun) | repeated | The taskRuns from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-Plan"></a>

### Plan



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the plan. `plan` is a system generated ID. Format: projects/{project}/plans/{plan} |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| issue | [string](#string) |  | The resource name of the issue associated with this plan. Format: projects/{project}/issues/{issue} |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| steps | [Plan.Step](#bytebase-v1-Plan-Step) | repeated |  |






<a name="bytebase-v1-Plan-ChangeDatabaseConfig"></a>

### Plan.ChangeDatabaseConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| target | [string](#string) |  | The resource name of the target. Format: instances/{instance-id}/databases/{database-name}. Format: projects/{project}/databaseGroups/{databaseGroup} |
| sheet | [string](#string) |  | The resource name of the sheet. Format: projects/{project}/sheets/{sheet} |
| type | [Plan.ChangeDatabaseConfig.Type](#bytebase-v1-Plan-ChangeDatabaseConfig-Type) |  |  |
| schema_version | [string](#string) |  | schema_version is parsed from VCS file name. It is automatically generated in the UI workflow. |
| rollback_enabled | [bool](#bool) |  | If RollbackEnabled, build the RollbackSheetID of the task. |
| rollback_detail | [Plan.ChangeDatabaseConfig.RollbackDetail](#bytebase-v1-Plan-ChangeDatabaseConfig-RollbackDetail) | optional |  |






<a name="bytebase-v1-Plan-ChangeDatabaseConfig-RollbackDetail"></a>

### Plan.ChangeDatabaseConfig.RollbackDetail



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rollback_from_task | [string](#string) |  | rollback_from_task is the task from which the rollback SQL statement is generated for this task. Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} |
| rollback_from_issue | [string](#string) |  | rollback_from_issue is the issue containing the original task from which the rollback SQL statement is generated for this task. Format: projects/{project}/issues/{issue} |






<a name="bytebase-v1-Plan-CreateDatabaseConfig"></a>

### Plan.CreateDatabaseConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| target | [string](#string) |  | The resource name of the instance on which the database is created. Format: instances/{instance} |
| database | [string](#string) |  | The name of the database to create. |
| table | [string](#string) |  | table is the name of the table, if it is not empty, Bytebase should create a table after creating the database. For example, in MongoDB, it only creates the database when we first store data in that database. |
| character_set | [string](#string) |  | character_set is the character set of the database. |
| collation | [string](#string) |  | collation is the collation of the database. |
| cluster | [string](#string) |  | cluster is the cluster of the database. This is only applicable to ClickHouse for &#34;ON CLUSTER &lt;&lt;cluster&gt;&gt;&#34;. |
| owner | [string](#string) |  | owner is the owner of the database. This is only applicable to Postgres for &#34;WITH OWNER &lt;&lt;owner&gt;&gt;&#34;. |
| backup | [string](#string) |  | backup is the resource name of the backup. Format: instances/{instance}/databases/{database}/backups/{backup-name} |
| labels | [Plan.CreateDatabaseConfig.LabelsEntry](#bytebase-v1-Plan-CreateDatabaseConfig-LabelsEntry) | repeated | labels of the database. |






<a name="bytebase-v1-Plan-CreateDatabaseConfig-LabelsEntry"></a>

### Plan.CreateDatabaseConfig.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-v1-Plan-RestoreDatabaseConfig"></a>

### Plan.RestoreDatabaseConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| target | [string](#string) |  | The resource name of the target to restore. Format: instances/{instance}/databases/{database} |
| create_database_config | [Plan.CreateDatabaseConfig](#bytebase-v1-Plan-CreateDatabaseConfig) | optional | create_database_config is present if the user wants to restore to a new database. |
| backup | [string](#string) |  |  |
| point_in_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | After the PITR operations, the database will be recovered to the state at this time. |






<a name="bytebase-v1-Plan-Spec"></a>

### Plan.Spec



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| earliest_allowed_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | earliest_allowed_time the earliest execution time of the change. |
| id | [string](#string) |  | A UUID4 string that uniquely identifies the Spec. |
| create_database_config | [Plan.CreateDatabaseConfig](#bytebase-v1-Plan-CreateDatabaseConfig) |  |  |
| change_database_config | [Plan.ChangeDatabaseConfig](#bytebase-v1-Plan-ChangeDatabaseConfig) |  |  |
| restore_database_config | [Plan.RestoreDatabaseConfig](#bytebase-v1-Plan-RestoreDatabaseConfig) |  |  |






<a name="bytebase-v1-Plan-Step"></a>

### Plan.Step



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| specs | [Plan.Spec](#bytebase-v1-Plan-Spec) | repeated |  |






<a name="bytebase-v1-PlanCheckRun"></a>

### PlanCheckRun



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/plans/{plan}/planCheckRuns/{planCheckRun} |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| type | [PlanCheckRun.Type](#bytebase-v1-PlanCheckRun-Type) |  |  |
| status | [PlanCheckRun.Status](#bytebase-v1-PlanCheckRun-Status) |  |  |
| target | [string](#string) |  | Format: instances/{instance}/databases/{database} Format: projects/{project}/databaseGroups/{databaseGroup} |
| sheet | [string](#string) |  | Format: projects/{project}/sheets/{sheet} |
| results | [PlanCheckRun.Result](#bytebase-v1-PlanCheckRun-Result) | repeated |  |
| error | [string](#string) |  | error is set if the Status is FAILED. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="bytebase-v1-PlanCheckRun-Result"></a>

### PlanCheckRun.Result



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [PlanCheckRun.Result.Status](#bytebase-v1-PlanCheckRun-Result-Status) |  |  |
| title | [string](#string) |  |  |
| content | [string](#string) |  |  |
| code | [int64](#int64) |  |  |
| sql_summary_report | [PlanCheckRun.Result.SqlSummaryReport](#bytebase-v1-PlanCheckRun-Result-SqlSummaryReport) |  |  |
| sql_review_report | [PlanCheckRun.Result.SqlReviewReport](#bytebase-v1-PlanCheckRun-Result-SqlReviewReport) |  |  |






<a name="bytebase-v1-PlanCheckRun-Result-SqlReviewReport"></a>

### PlanCheckRun.Result.SqlReviewReport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| line | [int64](#int64) |  |  |
| column | [int64](#int64) |  |  |
| detail | [string](#string) |  |  |
| code | [int64](#int64) |  | Code from sql review. |






<a name="bytebase-v1-PlanCheckRun-Result-SqlSummaryReport"></a>

### PlanCheckRun.Result.SqlSummaryReport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [int64](#int64) |  |  |
| statement_types | [string](#string) | repeated | statement_types are the types of statements that are found in the sql. |
| affected_rows | [int64](#int64) |  |  |
| changed_resources | [ChangedResources](#bytebase-v1-ChangedResources) |  |  |






<a name="bytebase-v1-PreviewRolloutRequest"></a>

### PreviewRolloutRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The name of the project. Format: projects/{project} |
| plan | [Plan](#bytebase-v1-Plan) |  | The plan used to preview rollout. |






<a name="bytebase-v1-Rollout"></a>

### Rollout



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The resource name of the rollout. Format: projects/{project}/rollouts/{rollout} |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| plan | [string](#string) |  | The plan that this rollout is based on. Format: projects/{project}/plans/{plan} |
| title | [string](#string) |  |  |
| stages | [Stage](#bytebase-v1-Stage) | repeated | stages and thus tasks of the rollout. |






<a name="bytebase-v1-RunPlanChecksRequest"></a>

### RunPlanChecksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The plan to run plan checks. Format: projects/{project}/plans/{plan} |






<a name="bytebase-v1-RunPlanChecksResponse"></a>

### RunPlanChecksResponse







<a name="bytebase-v1-Stage"></a>

### Stage



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage} |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| environment | [string](#string) |  | Format: environments/{environment} |
| title | [string](#string) |  |  |
| tasks | [Task](#bytebase-v1-Task) | repeated |  |






<a name="bytebase-v1-Task"></a>

### Task



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| title | [string](#string) |  |  |
| spec_id | [string](#string) |  | A UUID4 string that uniquely identifies the Spec. Could be empty if the rollout of the task does not have an associating plan. |
| status | [Task.Status](#bytebase-v1-Task-Status) |  | Status is the status of the task. |
| skipped_reason | [string](#string) |  |  |
| type | [Task.Type](#bytebase-v1-Task-Type) |  |  |
| blocked_by_tasks | [string](#string) | repeated | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} |
| target | [string](#string) |  | Format: instances/{instance} if the task is DatabaseCreate. Format: instances/{instance}/databases/{database} |
| database_create | [Task.DatabaseCreate](#bytebase-v1-Task-DatabaseCreate) |  |  |
| database_schema_baseline | [Task.DatabaseSchemaBaseline](#bytebase-v1-Task-DatabaseSchemaBaseline) |  |  |
| database_schema_update | [Task.DatabaseSchemaUpdate](#bytebase-v1-Task-DatabaseSchemaUpdate) |  |  |
| database_data_update | [Task.DatabaseDataUpdate](#bytebase-v1-Task-DatabaseDataUpdate) |  |  |
| database_backup | [Task.DatabaseBackup](#bytebase-v1-Task-DatabaseBackup) |  |  |
| database_restore_restore | [Task.DatabaseRestoreRestore](#bytebase-v1-Task-DatabaseRestoreRestore) |  |  |






<a name="bytebase-v1-Task-DatabaseBackup"></a>

### Task.DatabaseBackup



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| backup | [string](#string) |  | The resource name of the backup. Format: instances/{instance}/databases/{database}/backups/{backup-name} |






<a name="bytebase-v1-Task-DatabaseCreate"></a>

### Task.DatabaseCreate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The project owning the database. Format: projects/{project} |
| database | [string](#string) |  | database name |
| table | [string](#string) |  | table name |
| sheet | [string](#string) |  | Format: projects/{project}/sheets/{sheet} |
| character_set | [string](#string) |  |  |
| collation | [string](#string) |  |  |
| labels | [Task.DatabaseCreate.LabelsEntry](#bytebase-v1-Task-DatabaseCreate-LabelsEntry) | repeated |  |






<a name="bytebase-v1-Task-DatabaseCreate-LabelsEntry"></a>

### Task.DatabaseCreate.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-v1-Task-DatabaseDataUpdate"></a>

### Task.DatabaseDataUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheet | [string](#string) |  | Format: projects/{project}/sheets/{sheet} |
| schema_version | [string](#string) |  |  |
| rollback_enabled | [bool](#bool) |  | Build the rollback SQL if rollback_enabled. |
| rollback_sql_status | [Task.DatabaseDataUpdate.RollbackSqlStatus](#bytebase-v1-Task-DatabaseDataUpdate-RollbackSqlStatus) |  | The status of the rollback SQL generation. |
| rollback_error | [string](#string) |  |  |
| rollback_sheet | [string](#string) |  | rollback_sheet is the resource name of the sheet that stores the generated rollback SQL statement. Format: projects/{project}/sheets/{sheet} |
| rollback_from_issue | [string](#string) |  | rollback_from_issue is the resource name of the issue that the rollback SQL statement is generated from. Format: projects/{project}/issues/{issue} |
| rollback_from_task | [string](#string) |  | rollback_from_task is the resource name of the task that the rollback SQL statement is generated from. Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} |






<a name="bytebase-v1-Task-DatabaseRestoreRestore"></a>

### Task.DatabaseRestoreRestore



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| target | [string](#string) |  | Target is only used when doing restore to a new database now. It is empty for the case of in-place restore. Target {instance} must be within the same environment as the instance of the original database. {database} is the target database name. Format: instances/{instance}/databases/database |
| backup | [string](#string) |  | Only used when doing restore full backup only. Format: instances/{instance}/databases/{database}/backups/{backup-name} |
| point_in_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | After the PITR operations, the database will be recovered to the state at this time. |






<a name="bytebase-v1-Task-DatabaseSchemaBaseline"></a>

### Task.DatabaseSchemaBaseline



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema_version | [string](#string) |  |  |






<a name="bytebase-v1-Task-DatabaseSchemaUpdate"></a>

### Task.DatabaseSchemaUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheet | [string](#string) |  | Format: projects/{project}/sheets/{sheet} |
| schema_version | [string](#string) |  |  |






<a name="bytebase-v1-TaskRun"></a>

### TaskRun



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| creator | [string](#string) |  | Format: user/hello@world.com |
| updater | [string](#string) |  | Format: user/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| title | [string](#string) |  |  |
| status | [TaskRun.Status](#bytebase-v1-TaskRun-Status) |  |  |
| detail | [string](#string) |  | Below are the results of a task run. |
| change_history | [string](#string) |  | The resource name of the change history Format: instances/{instance}/databases/{database}/changeHistories/{changeHistory} |
| schema_version | [string](#string) |  |  |






<a name="bytebase-v1-UpdatePlanRequest"></a>

### UpdatePlanRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| plan | [Plan](#bytebase-v1-Plan) |  | The plan to update.

The plan&#39;s `name` field is used to identify the plan to update. Format: projects/{project}/plans/{plan} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 


<a name="bytebase-v1-Plan-ChangeDatabaseConfig-Type"></a>

### Plan.ChangeDatabaseConfig.Type
Type is the database change type.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| BASELINE | 1 | Used for establishing schema baseline, this is used when 1. Onboard the database into Bytebase since Bytebase needs to know the current database schema. 2. Had schema drift and need to re-establish the baseline. |
| MIGRATE | 2 | Used for DDL changes including CREATE DATABASE. |
| MIGRATE_SDL | 3 | Used for schema changes via state-based schema migration including CREATE DATABASE. |
| MIGRATE_GHOST | 4 | Used for DDL changes using gh-ost. |
| BRANCH | 5 | Used when restoring from a backup (the restored database branched from the original backup). |
| DATA | 6 | Used for DML change. |



<a name="bytebase-v1-PlanCheckRun-Result-Status"></a>

### PlanCheckRun.Result.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| ERROR | 1 |  |
| WARNING | 2 |  |
| SUCCESS | 3 |  |



<a name="bytebase-v1-PlanCheckRun-Status"></a>

### PlanCheckRun.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| RUNNING | 1 |  |
| DONE | 2 |  |
| FAILED | 3 |  |
| CANCELED | 4 |  |



<a name="bytebase-v1-PlanCheckRun-Type"></a>

### PlanCheckRun.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| DATABASE_STATEMENT_FAKE_ADVISE | 1 |  |
| DATABASE_STATEMENT_COMPATIBILITY | 2 |  |
| DATABASE_STATEMENT_ADVISE | 3 |  |
| DATABASE_STATEMENT_TYPE | 4 |  |
| DATABASE_STATEMENT_SUMMARY_REPORT | 5 |  |
| DATABASE_CONNECT | 6 |  |
| DATABASE_GHOST_SYNC | 7 |  |
| DATABASE_PITR_MYSQL | 8 |  |



<a name="bytebase-v1-Task-DatabaseDataUpdate-RollbackSqlStatus"></a>

### Task.DatabaseDataUpdate.RollbackSqlStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| ROLLBACK_SQL_STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| DONE | 2 |  |
| FAILED | 3 |  |



<a name="bytebase-v1-Task-Status"></a>

### Task.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| NOT_STARTED | 1 |  |
| PENDING | 2 |  |
| RUNNING | 3 |  |
| DONE | 4 |  |
| FAILED | 5 |  |
| CANCELED | 6 |  |
| SKIPPED | 7 |  |



<a name="bytebase-v1-Task-Type"></a>

### Task.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| GENERAL | 1 |  |
| DATABASE_CREATE | 2 | use payload DatabaseCreate |
| DATABASE_SCHEMA_BASELINE | 3 | use payload DatabaseSchemaBaseline |
| DATABASE_SCHEMA_UPDATE | 4 | use payload DatabaseSchemaUpdate |
| DATABASE_SCHEMA_UPDATE_SDL | 5 | use payload DatabaseSchemaUpdate |
| DATABASE_SCHEMA_UPDATE_GHOST_SYNC | 6 | use payload DatabaseSchemaUpdate |
| DATABASE_SCHEMA_UPDATE_GHOST_CUTOVER | 7 | use payload nil |
| DATABASE_DATA_UPDATE | 8 | use payload DatabaseDataUpdate |
| DATABASE_BACKUP | 9 | use payload DatabaseBackup |
| DATABASE_RESTORE_RESTORE | 10 | use payload DatabaseRestoreRestore |
| DATABASE_RESTORE_CUTOVER | 11 | use payload nil |



<a name="bytebase-v1-TaskRun-Status"></a>

### TaskRun.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| RUNNING | 2 |  |
| DONE | 3 |  |
| FAILED | 4 |  |
| CANCELED | 5 |  |


 

 


<a name="bytebase-v1-RolloutService"></a>

### RolloutService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetPlan | [GetPlanRequest](#bytebase-v1-GetPlanRequest) | [Plan](#bytebase-v1-Plan) |  |
| ListPlans | [ListPlansRequest](#bytebase-v1-ListPlansRequest) | [ListPlansResponse](#bytebase-v1-ListPlansResponse) |  |
| CreatePlan | [CreatePlanRequest](#bytebase-v1-CreatePlanRequest) | [Plan](#bytebase-v1-Plan) |  |
| UpdatePlan | [UpdatePlanRequest](#bytebase-v1-UpdatePlanRequest) | [Plan](#bytebase-v1-Plan) |  |
| GetRollout | [GetRolloutRequest](#bytebase-v1-GetRolloutRequest) | [Rollout](#bytebase-v1-Rollout) |  |
| CreateRollout | [CreateRolloutRequest](#bytebase-v1-CreateRolloutRequest) | [Rollout](#bytebase-v1-Rollout) |  |
| PreviewRollout | [PreviewRolloutRequest](#bytebase-v1-PreviewRolloutRequest) | [Rollout](#bytebase-v1-Rollout) |  |
| ListTaskRuns | [ListTaskRunsRequest](#bytebase-v1-ListTaskRunsRequest) | [ListTaskRunsResponse](#bytebase-v1-ListTaskRunsResponse) |  |
| ListPlanCheckRuns | [ListPlanCheckRunsRequest](#bytebase-v1-ListPlanCheckRunsRequest) | [ListPlanCheckRunsResponse](#bytebase-v1-ListPlanCheckRunsResponse) |  |
| RunPlanChecks | [RunPlanChecksRequest](#bytebase-v1-RunPlanChecksRequest) | [RunPlanChecksResponse](#bytebase-v1-RunPlanChecksResponse) |  |
| BatchRunTasks | [BatchRunTasksRequest](#bytebase-v1-BatchRunTasksRequest) | [BatchRunTasksResponse](#bytebase-v1-BatchRunTasksResponse) |  |
| BatchSkipTasks | [BatchSkipTasksRequest](#bytebase-v1-BatchSkipTasksRequest) | [BatchSkipTasksResponse](#bytebase-v1-BatchSkipTasksResponse) |  |
| BatchCancelTaskRuns | [BatchCancelTaskRunsRequest](#bytebase-v1-BatchCancelTaskRunsRequest) | [BatchCancelTaskRunsResponse](#bytebase-v1-BatchCancelTaskRunsResponse) |  |

 



<a name="v1_schema_design_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/schema_design_service.proto



<a name="bytebase-v1-CreateSchemaDesignRequest"></a>

### CreateSchemaDesignRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of schema designs. Format: project/{project} |
| schema_design | [SchemaDesign](#bytebase-v1-SchemaDesign) |  |  |






<a name="bytebase-v1-DeleteSchemaDesignRequest"></a>

### DeleteSchemaDesignRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the schema design to delete. Format: projects/{project}/schemaDesigns/{schemaDesign} |






<a name="bytebase-v1-GetSchemaDesignRequest"></a>

### GetSchemaDesignRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the schema design to retrieve. Format: projects/{project}/schemaDesigns/{schemaDesign} |






<a name="bytebase-v1-ListSchemaDesignsRequest"></a>

### ListSchemaDesignsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource of the schema design. Foramt: projects/{project} |
| filter | [string](#string) |  | To filter the search result. |
| page_size | [int32](#int32) |  | The maximum number of schema designs to return. The service may return fewer than this value. If unspecified, at most 50 schema designs will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListSchemaDesigns` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListSchemaDesigns` must match the call that provided the page token. |






<a name="bytebase-v1-ListSchemaDesignsResponse"></a>

### ListSchemaDesignsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema_designs | [SchemaDesign](#bytebase-v1-SchemaDesign) | repeated | The schema designs from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-MergeSchemaDesignRequest"></a>

### MergeSchemaDesignRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the schema design to merge. Format: projects/{project}/schemaDesigns/{schemaDesign} |
| target_name | [string](#string) |  | The target schema design to merge into. Format: projects/{project}/schemaDesigns/{schemaDesign} |






<a name="bytebase-v1-ParseSchemaStringRequest"></a>

### ParseSchemaStringRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema_string | [string](#string) |  | The schema string to parse. |
| engine | [Engine](#bytebase-v1-Engine) |  | The database engine of the schema string. |






<a name="bytebase-v1-ParseSchemaStringResponse"></a>

### ParseSchemaStringResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema_metadata | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  | The metadata of the parsed schema. |






<a name="bytebase-v1-SchemaDesign"></a>

### SchemaDesign



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the schema design. Format: projects/{project}/schemaDesigns/{schemaDesign} {schemaDesign} should be the id of a sheet. |
| title | [string](#string) |  | The title of schema design. AKA sheet&#39;s name. |
| schema | [string](#string) |  | The schema of schema design. AKA sheet&#39;s statement. |
| schema_metadata | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  | The metadata of the current editing schema. |
| baseline_schema | [string](#string) |  | The baseline schema. |
| baseline_schema_metadata | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  | The metadata of the baseline schema. |
| engine | [Engine](#bytebase-v1-Engine) |  | The database engine of the schema design. |
| baseline_database | [string](#string) |  | The name of the baseline database. Format: instances/{instance}/databases/{database} |
| baseline_sheet_name | [string](#string) |  | The name of the baseline sheet. For main branch, its format will be: projects/{project}/sheets/{sheet} For personal draft, its format will be: projects/{project}/schemaDesigns/{schemaDesign} |
| type | [SchemaDesign.Type](#bytebase-v1-SchemaDesign-Type) |  | The type of the schema design. |
| etag | [string](#string) |  | The etag of the schema design. |
| creator | [string](#string) |  | The creator of the schema design. Format: users/{email} |
| updater | [string](#string) |  | The updater of the schema design. Format: users/{email} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the schema design was created. |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the schema design was last updated. |






<a name="bytebase-v1-UpdateSchemaDesignRequest"></a>

### UpdateSchemaDesignRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema_design | [SchemaDesign](#bytebase-v1-SchemaDesign) |  | The schema design to update.

The schema design&#39;s `name` field is used to identify the schema design to update. Format: projects/{project}/schemaDesigns/{schemaDesign} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 


<a name="bytebase-v1-SchemaDesign-Type"></a>

### SchemaDesign.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| MAIN_BRANCH | 1 | Main branch type is the main version of schema design. And only allow to be updated/merged with personal drafts. |
| PERSONAL_DRAFT | 2 | Personal draft type is a copy of the main branch type schema designs. |


 

 


<a name="bytebase-v1-SchemaDesignService"></a>

### SchemaDesignService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetSchemaDesign | [GetSchemaDesignRequest](#bytebase-v1-GetSchemaDesignRequest) | [SchemaDesign](#bytebase-v1-SchemaDesign) |  |
| ListSchemaDesigns | [ListSchemaDesignsRequest](#bytebase-v1-ListSchemaDesignsRequest) | [ListSchemaDesignsResponse](#bytebase-v1-ListSchemaDesignsResponse) |  |
| CreateSchemaDesign | [CreateSchemaDesignRequest](#bytebase-v1-CreateSchemaDesignRequest) | [SchemaDesign](#bytebase-v1-SchemaDesign) |  |
| UpdateSchemaDesign | [UpdateSchemaDesignRequest](#bytebase-v1-UpdateSchemaDesignRequest) | [SchemaDesign](#bytebase-v1-SchemaDesign) |  |
| MergeSchemaDesign | [MergeSchemaDesignRequest](#bytebase-v1-MergeSchemaDesignRequest) | [SchemaDesign](#bytebase-v1-SchemaDesign) |  |
| ParseSchemaString | [ParseSchemaStringRequest](#bytebase-v1-ParseSchemaStringRequest) | [ParseSchemaStringResponse](#bytebase-v1-ParseSchemaStringResponse) |  |
| DeleteSchemaDesign | [DeleteSchemaDesignRequest](#bytebase-v1-DeleteSchemaDesignRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="v1_subscription_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/subscription_service.proto



<a name="bytebase-v1-Feature"></a>

### Feature



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name is the feature name. |
| matrix | [Feature.MatrixEntry](#bytebase-v1-Feature-MatrixEntry) | repeated | Matrix is the feature matrix for different plan. The key is the plan enum in string value. |






<a name="bytebase-v1-Feature-MatrixEntry"></a>

### Feature.MatrixEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [bool](#bool) |  |  |






<a name="bytebase-v1-FeatureMatrix"></a>

### FeatureMatrix



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| features | [Feature](#bytebase-v1-Feature) | repeated |  |






<a name="bytebase-v1-GetFeatureMatrixRequest"></a>

### GetFeatureMatrixRequest







<a name="bytebase-v1-GetSubscriptionRequest"></a>

### GetSubscriptionRequest







<a name="bytebase-v1-PatchSubscription"></a>

### PatchSubscription



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| license | [string](#string) |  |  |






<a name="bytebase-v1-Subscription"></a>

### Subscription



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_count | [int32](#int32) |  |  |
| expires_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| started_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| plan | [PlanType](#bytebase-v1-PlanType) |  |  |
| trialing | [bool](#bool) |  |  |
| org_id | [string](#string) |  |  |
| org_name | [string](#string) |  |  |






<a name="bytebase-v1-TrialSubscription"></a>

### TrialSubscription



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| plan | [PlanType](#bytebase-v1-PlanType) |  |  |
| days | [int32](#int32) |  |  |
| instance_count | [int32](#int32) |  |  |






<a name="bytebase-v1-TrialSubscriptionRequest"></a>

### TrialSubscriptionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| trial | [TrialSubscription](#bytebase-v1-TrialSubscription) |  |  |






<a name="bytebase-v1-UpdateSubscriptionRequest"></a>

### UpdateSubscriptionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| patch | [PatchSubscription](#bytebase-v1-PatchSubscription) |  |  |





 


<a name="bytebase-v1-PlanType"></a>

### PlanType


| Name | Number | Description |
| ---- | ------ | ----------- |
| PLAN_TYPE_UNSPECIFIED | 0 |  |
| FREE | 1 |  |
| TEAM | 2 |  |
| ENTERPRISE | 3 |  |


 

 


<a name="bytebase-v1-SubscriptionService"></a>

### SubscriptionService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetSubscription | [GetSubscriptionRequest](#bytebase-v1-GetSubscriptionRequest) | [Subscription](#bytebase-v1-Subscription) |  |
| GetFeatureMatrix | [GetFeatureMatrixRequest](#bytebase-v1-GetFeatureMatrixRequest) | [FeatureMatrix](#bytebase-v1-FeatureMatrix) |  |
| UpdateSubscription | [UpdateSubscriptionRequest](#bytebase-v1-UpdateSubscriptionRequest) | [Subscription](#bytebase-v1-Subscription) |  |
| TrialSubscription | [TrialSubscriptionRequest](#bytebase-v1-TrialSubscriptionRequest) | [Subscription](#bytebase-v1-Subscription) |  |

 



<a name="v1_setting_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/setting_service.proto



<a name="bytebase-v1-AgentPluginSetting"></a>

### AgentPluginSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | [string](#string) |  | The URL for the agent API. |
| token | [string](#string) |  | The token for the agent. |






<a name="bytebase-v1-AppIMSetting"></a>

### AppIMSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| im_type | [AppIMSetting.IMType](#bytebase-v1-AppIMSetting-IMType) |  |  |
| app_id | [string](#string) |  |  |
| app_secret | [string](#string) |  |  |
| external_approval | [AppIMSetting.ExternalApproval](#bytebase-v1-AppIMSetting-ExternalApproval) |  |  |






<a name="bytebase-v1-AppIMSetting-ExternalApproval"></a>

### AppIMSetting.ExternalApproval



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| approval_definition_id | [string](#string) |  |  |






<a name="bytebase-v1-DataClassificationSetting"></a>

### DataClassificationSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| configs | [DataClassificationSetting.DataClassificationConfig](#bytebase-v1-DataClassificationSetting-DataClassificationConfig) | repeated |  |






<a name="bytebase-v1-DataClassificationSetting-DataClassificationConfig"></a>

### DataClassificationSetting.DataClassificationConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | id is the uuid for classification. Each project can chose one classification config. |
| title | [string](#string) |  |  |
| levels | [DataClassificationSetting.DataClassificationConfig.Level](#bytebase-v1-DataClassificationSetting-DataClassificationConfig-Level) | repeated | levels is user defined level list for classification. The order for the level decides its priority. |
| classification | [DataClassificationSetting.DataClassificationConfig.ClassificationEntry](#bytebase-v1-DataClassificationSetting-DataClassificationConfig-ClassificationEntry) | repeated | classification is the id - DataClassification map. The id should in [0-9]&#43;-[0-9]&#43;-[0-9]&#43; format. |






<a name="bytebase-v1-DataClassificationSetting-DataClassificationConfig-ClassificationEntry"></a>

### DataClassificationSetting.DataClassificationConfig.ClassificationEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [DataClassificationSetting.DataClassificationConfig.DataClassification](#bytebase-v1-DataClassificationSetting-DataClassificationConfig-DataClassification) |  |  |






<a name="bytebase-v1-DataClassificationSetting-DataClassificationConfig-DataClassification"></a>

### DataClassificationSetting.DataClassificationConfig.DataClassification



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | id is the classification id in [0-9]&#43;-[0-9]&#43;-[0-9]&#43; format. |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| level_id | [string](#string) | optional |  |






<a name="bytebase-v1-DataClassificationSetting-DataClassificationConfig-Level"></a>

### DataClassificationSetting.DataClassificationConfig.Level



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| sensitive | [bool](#bool) |  |  |






<a name="bytebase-v1-ExternalApprovalSetting"></a>

### ExternalApprovalSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| nodes | [ExternalApprovalSetting.Node](#bytebase-v1-ExternalApprovalSetting-Node) | repeated |  |






<a name="bytebase-v1-ExternalApprovalSetting-Node"></a>

### ExternalApprovalSetting.Node



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | A unique identifier for a node in UUID format. We will also include the id in the message sending to the external relay service to identify the node. |
| title | [string](#string) |  | The title of the node. |
| endpoint | [string](#string) |  | The external endpoint for the relay service, e.g. &#34;http://hello:1234&#34;. |






<a name="bytebase-v1-GetSettingRequest"></a>

### GetSettingRequest
The request message for getting a setting.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The resource name of the setting. |






<a name="bytebase-v1-GetSettingResponse"></a>

### GetSettingResponse
The response message for getting a setting.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| setting | [Setting](#bytebase-v1-Setting) |  |  |






<a name="bytebase-v1-ListSettingsRequest"></a>

### ListSettingsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of settings to return. The service may return fewer than this value. If unspecified, at most 50 settings will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListSettings` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListSettings` must match the call that provided the page token. |






<a name="bytebase-v1-ListSettingsResponse"></a>

### ListSettingsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| settings | [Setting](#bytebase-v1-Setting) | repeated | The settings from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-SMTPMailDeliverySettingValue"></a>

### SMTPMailDeliverySettingValue



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| server | [string](#string) |  | The SMTP server address. |
| port | [int32](#int32) |  | The SMTP server port. |
| encryption | [SMTPMailDeliverySettingValue.Encryption](#bytebase-v1-SMTPMailDeliverySettingValue-Encryption) |  | The SMTP server encryption. |
| ca | [string](#string) | optional | The CA, KEY, and CERT for the SMTP server. Not used. |
| key | [string](#string) | optional |  |
| cert | [string](#string) | optional |  |
| authentication | [SMTPMailDeliverySettingValue.Authentication](#bytebase-v1-SMTPMailDeliverySettingValue-Authentication) |  |  |
| username | [string](#string) |  |  |
| password | [string](#string) | optional | If not specified, server will use the existed password. |
| from | [string](#string) |  | The sender email address. |
| to | [string](#string) |  | The recipient email address, used with validate_only to send test email. |






<a name="bytebase-v1-SchemaTemplateSetting"></a>

### SchemaTemplateSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| field_templates | [SchemaTemplateSetting.FieldTemplate](#bytebase-v1-SchemaTemplateSetting-FieldTemplate) | repeated |  |
| column_types | [SchemaTemplateSetting.ColumnType](#bytebase-v1-SchemaTemplateSetting-ColumnType) | repeated |  |






<a name="bytebase-v1-SchemaTemplateSetting-ColumnType"></a>

### SchemaTemplateSetting.ColumnType



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| engine | [Engine](#bytebase-v1-Engine) |  |  |
| enabled | [bool](#bool) |  |  |
| types | [string](#string) | repeated |  |






<a name="bytebase-v1-SchemaTemplateSetting-FieldTemplate"></a>

### SchemaTemplateSetting.FieldTemplate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| engine | [Engine](#bytebase-v1-Engine) |  |  |
| category | [string](#string) |  |  |
| column | [ColumnMetadata](#bytebase-v1-ColumnMetadata) |  |  |






<a name="bytebase-v1-SemanticCategorySetting"></a>

### SemanticCategorySetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| categories | [SemanticCategorySetting.SemanticCategory](#bytebase-v1-SemanticCategorySetting-SemanticCategory) | repeated |  |






<a name="bytebase-v1-SemanticCategorySetting-SemanticCategory"></a>

### SemanticCategorySetting.SemanticCategory



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | id is the uuid for semantic category. The id should in [0-9]&#43;-[0-9]&#43;-[0-9]&#43; format. |
| title | [string](#string) |  | the title of the category item, it should not be empty. |
| description | [string](#string) |  | the description of the category item, it can be empty.

We do not support custom algorithm by now, we only support the default algorithm, so we do not add the algorithm field right now. |






<a name="bytebase-v1-SetSettingRequest"></a>

### SetSettingRequest
The request message for updating a setting.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| setting | [Setting](#bytebase-v1-Setting) |  | The setting to update. |
| validate_only | [bool](#bool) |  | validate_only is a flag to indicate whether to validate the setting value, server would not persist the setting value if it is true. |






<a name="bytebase-v1-Setting"></a>

### Setting
The schema of setting.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The resource name of the setting. Must be one of the following forms:

- `setting/{setting_name}` For example, &#34;settings/bb.branding.logo&#34; |
| value | [Value](#bytebase-v1-Value) |  | The value of the setting. |






<a name="bytebase-v1-Value"></a>

### Value
The data in setting value.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| string_value | [string](#string) |  | Defines this value as being a string value. |
| smtp_mail_delivery_setting_value | [SMTPMailDeliverySettingValue](#bytebase-v1-SMTPMailDeliverySettingValue) |  |  |
| app_im_setting_value | [AppIMSetting](#bytebase-v1-AppIMSetting) |  |  |
| agent_plugin_setting_value | [AgentPluginSetting](#bytebase-v1-AgentPluginSetting) |  |  |
| workspace_profile_setting_value | [WorkspaceProfileSetting](#bytebase-v1-WorkspaceProfileSetting) |  |  |
| workspace_approval_setting_value | [WorkspaceApprovalSetting](#bytebase-v1-WorkspaceApprovalSetting) |  |  |
| workspace_trial_setting_value | [WorkspaceTrialSetting](#bytebase-v1-WorkspaceTrialSetting) |  |  |
| external_approval_setting_value | [ExternalApprovalSetting](#bytebase-v1-ExternalApprovalSetting) |  |  |
| schema_template_setting_value | [SchemaTemplateSetting](#bytebase-v1-SchemaTemplateSetting) |  |  |
| data_classification_setting_value | [DataClassificationSetting](#bytebase-v1-DataClassificationSetting) |  |  |
| semantic_category_setting_value | [SemanticCategorySetting](#bytebase-v1-SemanticCategorySetting) |  |  |






<a name="bytebase-v1-WorkspaceApprovalSetting"></a>

### WorkspaceApprovalSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rules | [WorkspaceApprovalSetting.Rule](#bytebase-v1-WorkspaceApprovalSetting-Rule) | repeated |  |






<a name="bytebase-v1-WorkspaceApprovalSetting-Rule"></a>

### WorkspaceApprovalSetting.Rule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| template | [ApprovalTemplate](#bytebase-v1-ApprovalTemplate) |  |  |
| condition | [google.type.Expr](#google-type-Expr) |  |  |






<a name="bytebase-v1-WorkspaceProfileSetting"></a>

### WorkspaceProfileSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| external_url | [string](#string) |  | The URL user visits Bytebase.

The external URL is used for: 1. Constructing the correct callback URL when configuring the VCS provider. The callback URL points to the frontend. 2. Creating the correct webhook endpoint when configuring the project GitOps workflow. The webhook endpoint points to the backend. |
| disallow_signup | [bool](#bool) |  | Disallow self-service signup, users can only be invited by the owner. |
| require_2fa | [bool](#bool) |  | Require 2FA for all users. |
| outbound_ip_list | [string](#string) | repeated | outbound_ip_list is the outbound IP for Bytebase instance in SaaS mode. |
| gitops_webhook_url | [string](#string) |  | The webhook URL for the GitOps workflow. |
| refresh_token_duration | [google.protobuf.Duration](#google-protobuf-Duration) |  | The duration for refresh token. |






<a name="bytebase-v1-WorkspaceTrialSetting"></a>

### WorkspaceTrialSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_count | [int32](#int32) |  |  |
| expire_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| issued_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| subject | [string](#string) |  |  |
| org_name | [string](#string) |  |  |
| plan | [PlanType](#bytebase-v1-PlanType) |  |  |





 


<a name="bytebase-v1-AppIMSetting-IMType"></a>

### AppIMSetting.IMType


| Name | Number | Description |
| ---- | ------ | ----------- |
| IM_TYPE_UNSPECIFIED | 0 |  |
| FEISHU | 1 |  |



<a name="bytebase-v1-SMTPMailDeliverySettingValue-Authentication"></a>

### SMTPMailDeliverySettingValue.Authentication
We support four types of SMTP authentication: NONE, PLAIN, LOGIN, and CRAM-MD5.

| Name | Number | Description |
| ---- | ------ | ----------- |
| AUTHENTICATION_UNSPECIFIED | 0 |  |
| AUTHENTICATION_NONE | 1 |  |
| AUTHENTICATION_PLAIN | 2 |  |
| AUTHENTICATION_LOGIN | 3 |  |
| AUTHENTICATION_CRAM_MD5 | 4 |  |



<a name="bytebase-v1-SMTPMailDeliverySettingValue-Encryption"></a>

### SMTPMailDeliverySettingValue.Encryption
We support three types of SMTP encryption: NONE, STARTTLS, and SSL/TLS.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ENCRYPTION_UNSPECIFIED | 0 |  |
| ENCRYPTION_NONE | 1 |  |
| ENCRYPTION_STARTTLS | 2 |  |
| ENCRYPTION_SSL_TLS | 3 |  |


 

 


<a name="bytebase-v1-SettingService"></a>

### SettingService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListSettings | [ListSettingsRequest](#bytebase-v1-ListSettingsRequest) | [ListSettingsResponse](#bytebase-v1-ListSettingsResponse) |  |
| GetSetting | [GetSettingRequest](#bytebase-v1-GetSettingRequest) | [Setting](#bytebase-v1-Setting) |  |
| SetSetting | [SetSettingRequest](#bytebase-v1-SetSettingRequest) | [Setting](#bytebase-v1-Setting) |  |

 



<a name="v1_sheet_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/sheet_service.proto



<a name="bytebase-v1-CreateSheetRequest"></a>

### CreateSheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this sheet will be created. Foramt: projects/{project} |
| sheet | [Sheet](#bytebase-v1-Sheet) |  | The sheet to create. |






<a name="bytebase-v1-DeleteSheetRequest"></a>

### DeleteSheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the sheet to delete. Format: projects/{project}/sheets/{sheet} |






<a name="bytebase-v1-GetSheetRequest"></a>

### GetSheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the sheet to retrieve. Format: projects/{project}/sheets/{sheet} |
| raw | [bool](#bool) |  | By default, the content of the sheet is cut off, set the `raw` to true to retrieve the full content. |






<a name="bytebase-v1-SearchSheetsRequest"></a>

### SearchSheetsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource of the sheet. Foramt: projects/{project} |
| filter | [string](#string) |  | To filter the search result. Format: only support the following spec for now: - `creator = users/{email}`, `creator != users/{email}` - `starred = true`, `starred = false`. Not support empty filter for now. |
| page_size | [int32](#int32) |  | Not used. The maximum number of sheets to return. The service may return fewer than this value. If unspecified, at most 50 sheets will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `SearchSheets` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `SearchSheets` must match the call that provided the page token. |






<a name="bytebase-v1-SearchSheetsResponse"></a>

### SearchSheetsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheets | [Sheet](#bytebase-v1-Sheet) | repeated | The sheets that matched the search criteria. |
| next_page_token | [string](#string) |  | Not used. A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-Sheet"></a>

### Sheet



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the sheet resource, generated by the server. Canonical parent is project. Format: projects/{project}/sheets/{sheet} |
| database | [string](#string) |  | The database resource name. Format: instances/{instance}/databases/{database} If the database parent doesn&#39;t exist, the database field is empty. |
| title | [string](#string) |  | The title of the sheet. |
| creator | [string](#string) |  | The creator of the Sheet. Format: users/{email} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The create time of the sheet. |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The last update time of the sheet. |
| content | [bytes](#bytes) |  | The content of the sheet. By default, it will be cut off, if it doesn&#39;t match the `content_size`, you can set the `raw` to true in GetSheet request to retrieve the full content. |
| content_size | [int64](#int64) |  | content_size is the full size of the content, may not match the size of the `content` field. |
| visibility | [Sheet.Visibility](#bytebase-v1-Sheet-Visibility) |  |  |
| source | [Sheet.Source](#bytebase-v1-Sheet-Source) |  | The source of the sheet. |
| type | [Sheet.Type](#bytebase-v1-Sheet-Type) |  | The type of the sheet. |
| starred | [bool](#bool) |  | starred indicates whether the sheet is starred by the current authenticated user. |
| payload | [string](#string) |  | TODO: deprecate this field. |






<a name="bytebase-v1-SheetOrganizer"></a>

### SheetOrganizer



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheet | [string](#string) |  | The name of the sheet. Format: projects/{project}/sheets/{sheet} |
| starred | [bool](#bool) |  | starred means if the sheet is starred. |
| pinned | [bool](#bool) |  | pinned means if the sheet is pinned. |






<a name="bytebase-v1-SyncSheetsRequest"></a>

### SyncSheetsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The name of the project to sync sheets.

Format: projects/{project} |






<a name="bytebase-v1-UpdateSheetOrganizerRequest"></a>

### UpdateSheetOrganizerRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organizer | [SheetOrganizer](#bytebase-v1-SheetOrganizer) |  | The organizer to update.

The organizer&#39;s `sheet` field is used to identify the sheet. Format: projects/{project}/sheets/{sheet} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to be updated. Fields are specified relative to the sheet organizer. Only support update the following fields for now: - `starred` - `pinned` |






<a name="bytebase-v1-UpdateSheetRequest"></a>

### UpdateSheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheet | [Sheet](#bytebase-v1-Sheet) |  | The sheet to update.

The sheet&#39;s `name` field is used to identify the sheet to update. Format: projects/{project}/sheets/{sheet} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to be updated. Fields are specified relative to the sheet. (e.g. `title`, `statement`; *not* `sheet.title` or `sheet.statement`) Only support update the following fields for now: - `title` - `statement` - `starred` - `visibility` |





 


<a name="bytebase-v1-Sheet-Source"></a>

### Sheet.Source


| Name | Number | Description |
| ---- | ------ | ----------- |
| SOURCE_UNSPECIFIED | 0 |  |
| SOURCE_BYTEBASE | 1 | BYTEBASE is the sheet created by Bytebase. e.g. SQL Editor. |
| SOURCE_BYTEBASE_ARTIFACT | 2 | BYTEBASE_ARTIFACT is the artifact sheet. |
| SOURCE_GITLAB | 3 | GITLAB is the sheet synced from GitLab (for both GitLab.com and self-hosted GitLab). |
| SOURCE_GITHUB | 4 | GITHUB is the sheet synced from GitHub (for both GitHub.com and GitHub Enterprise). |
| SOURCE_BITBUCKET | 5 | BITBUCKET is the sheet synced from Bitbucket (for both Bitbucket.org and Bitbucket Server). |



<a name="bytebase-v1-Sheet-Type"></a>

### Sheet.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| TYPE_SQL | 1 |  |



<a name="bytebase-v1-Sheet-Visibility"></a>

### Sheet.Visibility


| Name | Number | Description |
| ---- | ------ | ----------- |
| VISIBILITY_UNSPECIFIED | 0 |  |
| VISIBILITY_PUBLIC | 1 | Public, sheet OWNER can read/write, and all others can read. |
| VISIBILITY_PROJECT | 2 | Project, sheet OWNER and project OWNER can read/write, and project DEVELOPER can read. |
| VISIBILITY_PRIVATE | 3 | Private, only sheet OWNER can read/write. |


 

 


<a name="bytebase-v1-SheetService"></a>

### SheetService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateSheet | [CreateSheetRequest](#bytebase-v1-CreateSheetRequest) | [Sheet](#bytebase-v1-Sheet) |  |
| GetSheet | [GetSheetRequest](#bytebase-v1-GetSheetRequest) | [Sheet](#bytebase-v1-Sheet) |  |
| SearchSheets | [SearchSheetsRequest](#bytebase-v1-SearchSheetsRequest) | [SearchSheetsResponse](#bytebase-v1-SearchSheetsResponse) |  |
| UpdateSheet | [UpdateSheetRequest](#bytebase-v1-UpdateSheetRequest) | [Sheet](#bytebase-v1-Sheet) |  |
| UpdateSheetOrganizer | [UpdateSheetOrganizerRequest](#bytebase-v1-UpdateSheetOrganizerRequest) | [SheetOrganizer](#bytebase-v1-SheetOrganizer) |  |
| DeleteSheet | [DeleteSheetRequest](#bytebase-v1-DeleteSheetRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| SyncSheets | [SyncSheetsRequest](#bytebase-v1-SyncSheetsRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="v1_sql_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/sql_service.proto



<a name="bytebase-v1-AdminExecuteRequest"></a>

### AdminExecuteRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the instance name to execute the query against. Format: instances/{instance} |
| connection_database | [string](#string) |  | The connection database name to execute the query against. For PostgreSQL, it&#39;s required. For other database engines, it&#39;s optional. Use empty string to execute against without specifying a database. |
| statement | [string](#string) |  | The SQL statement to execute. |
| limit | [int32](#int32) |  | The maximum number of rows to return. |
| timeout | [google.protobuf.Duration](#google-protobuf-Duration) |  | The timeout for the request. |






<a name="bytebase-v1-AdminExecuteResponse"></a>

### AdminExecuteResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [QueryResult](#bytebase-v1-QueryResult) | repeated | The query results. |






<a name="bytebase-v1-Advice"></a>

### Advice



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [Advice.Status](#bytebase-v1-Advice-Status) |  | The advice status. |
| code | [int32](#int32) |  | The advice code. |
| title | [string](#string) |  | The advice title. |
| content | [string](#string) |  | The advice content. |
| line | [int32](#int32) |  | The advice line number in the SQL statement. |
| detail | [string](#string) |  | The advice detail. |






<a name="bytebase-v1-DifferPreviewRequest"></a>

### DifferPreviewRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| engine | [Engine](#bytebase-v1-Engine) |  |  |
| old_schema | [string](#string) |  |  |
| new_metadata | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  |  |






<a name="bytebase-v1-DifferPreviewResponse"></a>

### DifferPreviewResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  |  |






<a name="bytebase-v1-ExportRequest"></a>

### ExportRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the instance name to execute the query against. Format: instances/{instance} |
| connection_database | [string](#string) |  | The connection database name to execute the query against. For PostgreSQL, it&#39;s required. For other database engines, it&#39;s optional. Use empty string to execute against without specifying a database. |
| statement | [string](#string) |  | The SQL statement to execute. |
| limit | [int32](#int32) |  | The maximum number of rows to return. |
| format | [ExportRequest.Format](#bytebase-v1-ExportRequest-Format) |  | The export format. |
| admin | [bool](#bool) |  | The admin is used for workspace owner and DBA for exporting data from SQL Editor Admin mode. The exported data is not masked. |






<a name="bytebase-v1-ExportResponse"></a>

### ExportResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| content | [bytes](#bytes) |  | The export file content. |






<a name="bytebase-v1-PrettyRequest"></a>

### PrettyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| engine | [Engine](#bytebase-v1-Engine) |  |  |
| current_schema | [string](#string) |  | The SDL format SQL schema information that was dumped from a database engine. This information will be sorted to match the order of statements in the userSchema. |
| expected_schema | [string](#string) |  | The expected SDL schema. This schema will be checked for correctness and normalized. |






<a name="bytebase-v1-PrettyResponse"></a>

### PrettyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| current_schema | [string](#string) |  | The pretty-formatted version of current schema. |
| expected_schema | [string](#string) |  | The expected SDL schema after normalizing. |






<a name="bytebase-v1-QueryRequest"></a>

### QueryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the instance name to execute the query against. Format: instances/{instance} |
| connection_database | [string](#string) |  | The connection database name to execute the query against. For PostgreSQL, it&#39;s required. For other database engines, it&#39;s optional. Use empty string to execute against without specifying a database. |
| statement | [string](#string) |  | The SQL statement to execute. |
| limit | [int32](#int32) |  | The maximum number of rows to return. |
| timeout | [google.protobuf.Duration](#google-protobuf-Duration) |  | The timeout for the request. |






<a name="bytebase-v1-QueryResponse"></a>

### QueryResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [QueryResult](#bytebase-v1-QueryResult) | repeated | The query results. |
| advices | [Advice](#bytebase-v1-Advice) | repeated | The query advices. |
| allow_export | [bool](#bool) |  | The query is allowed to be exported or not. |






<a name="bytebase-v1-QueryResult"></a>

### QueryResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| column_names | [string](#string) | repeated | Column names of the query result. |
| column_type_names | [string](#string) | repeated | Column types of the query result. The types come from the Golang SQL driver. |
| rows | [QueryRow](#bytebase-v1-QueryRow) | repeated | Rows of the query result. |
| masked | [bool](#bool) | repeated | Columns are masked or not. |
| sensitive | [bool](#bool) | repeated | Columns are sensitive or not. |
| error | [string](#string) |  | The error message if the query failed. |
| latency | [google.protobuf.Duration](#google-protobuf-Duration) |  | The time it takes to execute the query. |
| statement | [string](#string) |  | The query statement for the result. |






<a name="bytebase-v1-QueryRow"></a>

### QueryRow



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| values | [RowValue](#bytebase-v1-RowValue) | repeated | Row values of the query result. |






<a name="bytebase-v1-RowValue"></a>

### RowValue



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| null_value | [google.protobuf.NullValue](#google-protobuf-NullValue) |  |  |
| bool_value | [bool](#bool) |  |  |
| bytes_value | [bytes](#bytes) |  |  |
| double_value | [double](#double) |  |  |
| float_value | [float](#float) |  |  |
| int32_value | [int32](#int32) |  |  |
| int64_value | [int64](#int64) |  |  |
| string_value | [string](#string) |  |  |
| uint32_value | [uint32](#uint32) |  |  |
| uint64_value | [uint64](#uint64) |  |  |
| value_value | [google.protobuf.Value](#google-protobuf-Value) |  | value_value is used for Spanner and TUPLE ARRAY MAP in Clickhouse only. |





 


<a name="bytebase-v1-Advice-Status"></a>

### Advice.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 | Unspecified. |
| SUCCESS | 1 |  |
| WARNING | 2 |  |
| ERROR | 3 |  |



<a name="bytebase-v1-ExportRequest-Format"></a>

### ExportRequest.Format


| Name | Number | Description |
| ---- | ------ | ----------- |
| FORMAT_UNSPECIFIED | 0 |  |
| CSV | 1 |  |
| JSON | 2 |  |
| SQL | 3 |  |
| XLSX | 4 |  |


 

 


<a name="bytebase-v1-SQLService"></a>

### SQLService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Pretty | [PrettyRequest](#bytebase-v1-PrettyRequest) | [PrettyResponse](#bytebase-v1-PrettyResponse) |  |
| Query | [QueryRequest](#bytebase-v1-QueryRequest) | [QueryResponse](#bytebase-v1-QueryResponse) |  |
| Export | [ExportRequest](#bytebase-v1-ExportRequest) | [ExportResponse](#bytebase-v1-ExportResponse) |  |
| AdminExecute | [AdminExecuteRequest](#bytebase-v1-AdminExecuteRequest) stream | [AdminExecuteResponse](#bytebase-v1-AdminExecuteResponse) stream |  |
| DifferPreview | [DifferPreviewRequest](#bytebase-v1-DifferPreviewRequest) | [DifferPreviewResponse](#bytebase-v1-DifferPreviewResponse) |  |

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

