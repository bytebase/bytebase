# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [v1/annotation.proto](#v1_annotation-proto)
    - [AuthMethod](#bytebase-v1-AuthMethod)
  
    - [File-level Extensions](#v1_annotation-proto-extensions)
    - [File-level Extensions](#v1_annotation-proto-extensions)
    - [File-level Extensions](#v1_annotation-proto-extensions)
  
- [v1/actuator_service.proto](#v1_actuator_service-proto)
    - [ActuatorInfo](#bytebase-v1-ActuatorInfo)
    - [DebugLog](#bytebase-v1-DebugLog)
    - [DeleteCacheRequest](#bytebase-v1-DeleteCacheRequest)
    - [GetActuatorInfoRequest](#bytebase-v1-GetActuatorInfoRequest)
    - [GetResourcePackageRequest](#bytebase-v1-GetResourcePackageRequest)
    - [ListDebugLogRequest](#bytebase-v1-ListDebugLogRequest)
    - [ListDebugLogResponse](#bytebase-v1-ListDebugLogResponse)
    - [ResourcePackage](#bytebase-v1-ResourcePackage)
    - [UpdateActuatorInfoRequest](#bytebase-v1-UpdateActuatorInfoRequest)
  
    - [ActuatorService](#bytebase-v1-ActuatorService)
  
- [v1/anomaly_service.proto](#v1_anomaly_service-proto)
    - [Anomaly](#bytebase-v1-Anomaly)
    - [Anomaly.DatabaseConnectionDetail](#bytebase-v1-Anomaly-DatabaseConnectionDetail)
    - [Anomaly.DatabaseSchemaDriftDetail](#bytebase-v1-Anomaly-DatabaseSchemaDriftDetail)
    - [Anomaly.InstanceConnectionDetail](#bytebase-v1-Anomaly-InstanceConnectionDetail)
    - [SearchAnomaliesRequest](#bytebase-v1-SearchAnomaliesRequest)
    - [SearchAnomaliesResponse](#bytebase-v1-SearchAnomaliesResponse)
  
    - [Anomaly.AnomalySeverity](#bytebase-v1-Anomaly-AnomalySeverity)
    - [Anomaly.AnomalyType](#bytebase-v1-Anomaly-AnomalyType)
  
    - [AnomalyService](#bytebase-v1-AnomalyService)
  
- [v1/common.proto](#v1_common-proto)
    - [Position](#bytebase-v1-Position)
  
    - [Engine](#bytebase-v1-Engine)
    - [ExportFormat](#bytebase-v1-ExportFormat)
    - [MaskingLevel](#bytebase-v1-MaskingLevel)
    - [State](#bytebase-v1-State)
    - [VCSType](#bytebase-v1-VCSType)
  
- [v1/audit_log_service.proto](#v1_audit_log_service-proto)
    - [AuditLog](#bytebase-v1-AuditLog)
    - [ExportAuditLogsRequest](#bytebase-v1-ExportAuditLogsRequest)
    - [ExportAuditLogsResponse](#bytebase-v1-ExportAuditLogsResponse)
    - [SearchAuditLogsRequest](#bytebase-v1-SearchAuditLogsRequest)
    - [SearchAuditLogsResponse](#bytebase-v1-SearchAuditLogsResponse)
  
    - [AuditLog.Severity](#bytebase-v1-AuditLog-Severity)
  
    - [AuditLogService](#bytebase-v1-AuditLogService)
  
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
  
    - [UserType](#bytebase-v1-UserType)
  
    - [AuthService](#bytebase-v1-AuthService)
  
- [v1/instance_service.proto](#v1_instance_service-proto)
    - [AddDataSourceRequest](#bytebase-v1-AddDataSourceRequest)
    - [BatchSyncInstanceRequest](#bytebase-v1-BatchSyncInstanceRequest)
    - [BatchSyncInstanceResponse](#bytebase-v1-BatchSyncInstanceResponse)
    - [CreateInstanceRequest](#bytebase-v1-CreateInstanceRequest)
    - [DataSource](#bytebase-v1-DataSource)
    - [DataSource.Address](#bytebase-v1-DataSource-Address)
    - [DataSourceExternalSecret](#bytebase-v1-DataSourceExternalSecret)
    - [DataSourceExternalSecret.AppRoleAuthOption](#bytebase-v1-DataSourceExternalSecret-AppRoleAuthOption)
    - [DeleteInstanceRequest](#bytebase-v1-DeleteInstanceRequest)
    - [GetInstanceRequest](#bytebase-v1-GetInstanceRequest)
    - [Instance](#bytebase-v1-Instance)
    - [InstanceOptions](#bytebase-v1-InstanceOptions)
    - [InstanceResource](#bytebase-v1-InstanceResource)
    - [KerberosConfig](#bytebase-v1-KerberosConfig)
    - [ListInstancesRequest](#bytebase-v1-ListInstancesRequest)
    - [ListInstancesResponse](#bytebase-v1-ListInstancesResponse)
    - [RemoveDataSourceRequest](#bytebase-v1-RemoveDataSourceRequest)
    - [SASLConfig](#bytebase-v1-SASLConfig)
    - [SearchInstancesRequest](#bytebase-v1-SearchInstancesRequest)
    - [SearchInstancesResponse](#bytebase-v1-SearchInstancesResponse)
    - [SyncInstanceRequest](#bytebase-v1-SyncInstanceRequest)
    - [SyncInstanceResponse](#bytebase-v1-SyncInstanceResponse)
    - [SyncSlowQueriesRequest](#bytebase-v1-SyncSlowQueriesRequest)
    - [UndeleteInstanceRequest](#bytebase-v1-UndeleteInstanceRequest)
    - [UpdateDataSourceRequest](#bytebase-v1-UpdateDataSourceRequest)
    - [UpdateInstanceRequest](#bytebase-v1-UpdateInstanceRequest)
  
    - [DataSource.AuthenticationType](#bytebase-v1-DataSource-AuthenticationType)
    - [DataSource.RedisType](#bytebase-v1-DataSource-RedisType)
    - [DataSourceExternalSecret.AppRoleAuthOption.SecretType](#bytebase-v1-DataSourceExternalSecret-AppRoleAuthOption-SecretType)
    - [DataSourceExternalSecret.AuthType](#bytebase-v1-DataSourceExternalSecret-AuthType)
    - [DataSourceExternalSecret.SecretType](#bytebase-v1-DataSourceExternalSecret-SecretType)
    - [DataSourceType](#bytebase-v1-DataSourceType)
  
    - [InstanceService](#bytebase-v1-InstanceService)
  
- [v1/database_service.proto](#v1_database_service-proto)
    - [AdviseIndexRequest](#bytebase-v1-AdviseIndexRequest)
    - [AdviseIndexResponse](#bytebase-v1-AdviseIndexResponse)
    - [BatchUpdateDatabasesRequest](#bytebase-v1-BatchUpdateDatabasesRequest)
    - [BatchUpdateDatabasesResponse](#bytebase-v1-BatchUpdateDatabasesResponse)
    - [ChangeHistory](#bytebase-v1-ChangeHistory)
    - [ChangedResourceDatabase](#bytebase-v1-ChangedResourceDatabase)
    - [ChangedResourceSchema](#bytebase-v1-ChangedResourceSchema)
    - [ChangedResourceTable](#bytebase-v1-ChangedResourceTable)
    - [ChangedResources](#bytebase-v1-ChangedResources)
    - [CheckConstraintMetadata](#bytebase-v1-CheckConstraintMetadata)
    - [ColumnConfig](#bytebase-v1-ColumnConfig)
    - [ColumnConfig.LabelsEntry](#bytebase-v1-ColumnConfig-LabelsEntry)
    - [ColumnMetadata](#bytebase-v1-ColumnMetadata)
    - [Database](#bytebase-v1-Database)
    - [Database.LabelsEntry](#bytebase-v1-Database-LabelsEntry)
    - [DatabaseConfig](#bytebase-v1-DatabaseConfig)
    - [DatabaseMetadata](#bytebase-v1-DatabaseMetadata)
    - [DatabaseSchema](#bytebase-v1-DatabaseSchema)
    - [DeleteSecretRequest](#bytebase-v1-DeleteSecretRequest)
    - [DependentColumn](#bytebase-v1-DependentColumn)
    - [DiffSchemaRequest](#bytebase-v1-DiffSchemaRequest)
    - [DiffSchemaResponse](#bytebase-v1-DiffSchemaResponse)
    - [ExtensionMetadata](#bytebase-v1-ExtensionMetadata)
    - [ExternalTableMetadata](#bytebase-v1-ExternalTableMetadata)
    - [ForeignKeyMetadata](#bytebase-v1-ForeignKeyMetadata)
    - [FunctionConfig](#bytebase-v1-FunctionConfig)
    - [FunctionMetadata](#bytebase-v1-FunctionMetadata)
    - [GenerationMetadata](#bytebase-v1-GenerationMetadata)
    - [GetChangeHistoryRequest](#bytebase-v1-GetChangeHistoryRequest)
    - [GetDatabaseMetadataRequest](#bytebase-v1-GetDatabaseMetadataRequest)
    - [GetDatabaseRequest](#bytebase-v1-GetDatabaseRequest)
    - [GetDatabaseSchemaRequest](#bytebase-v1-GetDatabaseSchemaRequest)
    - [IndexMetadata](#bytebase-v1-IndexMetadata)
    - [ListChangeHistoriesRequest](#bytebase-v1-ListChangeHistoriesRequest)
    - [ListChangeHistoriesResponse](#bytebase-v1-ListChangeHistoriesResponse)
    - [ListDatabasesRequest](#bytebase-v1-ListDatabasesRequest)
    - [ListDatabasesResponse](#bytebase-v1-ListDatabasesResponse)
    - [ListInstanceDatabasesRequest](#bytebase-v1-ListInstanceDatabasesRequest)
    - [ListInstanceDatabasesResponse](#bytebase-v1-ListInstanceDatabasesResponse)
    - [ListSecretsRequest](#bytebase-v1-ListSecretsRequest)
    - [ListSecretsResponse](#bytebase-v1-ListSecretsResponse)
    - [ListSlowQueriesRequest](#bytebase-v1-ListSlowQueriesRequest)
    - [ListSlowQueriesResponse](#bytebase-v1-ListSlowQueriesResponse)
    - [MaterializedViewMetadata](#bytebase-v1-MaterializedViewMetadata)
    - [ProcedureConfig](#bytebase-v1-ProcedureConfig)
    - [ProcedureMetadata](#bytebase-v1-ProcedureMetadata)
    - [SchemaConfig](#bytebase-v1-SchemaConfig)
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
    - [TableConfig](#bytebase-v1-TableConfig)
    - [TableMetadata](#bytebase-v1-TableMetadata)
    - [TablePartitionMetadata](#bytebase-v1-TablePartitionMetadata)
    - [TaskMetadata](#bytebase-v1-TaskMetadata)
    - [UpdateDatabaseMetadataRequest](#bytebase-v1-UpdateDatabaseMetadataRequest)
    - [UpdateDatabaseRequest](#bytebase-v1-UpdateDatabaseRequest)
    - [UpdateSecretRequest](#bytebase-v1-UpdateSecretRequest)
    - [ViewConfig](#bytebase-v1-ViewConfig)
    - [ViewMetadata](#bytebase-v1-ViewMetadata)
  
    - [ChangeHistory.Source](#bytebase-v1-ChangeHistory-Source)
    - [ChangeHistory.Status](#bytebase-v1-ChangeHistory-Status)
    - [ChangeHistory.Type](#bytebase-v1-ChangeHistory-Type)
    - [ChangeHistoryView](#bytebase-v1-ChangeHistoryView)
    - [DatabaseMetadataView](#bytebase-v1-DatabaseMetadataView)
    - [GenerationMetadata.Type](#bytebase-v1-GenerationMetadata-Type)
    - [StreamMetadata.Mode](#bytebase-v1-StreamMetadata-Mode)
    - [StreamMetadata.Type](#bytebase-v1-StreamMetadata-Type)
    - [TablePartitionMetadata.Type](#bytebase-v1-TablePartitionMetadata-Type)
    - [TaskMetadata.State](#bytebase-v1-TaskMetadata-State)
  
    - [DatabaseService](#bytebase-v1-DatabaseService)
  
- [v1/branch_service.proto](#v1_branch_service-proto)
    - [Branch](#bytebase-v1-Branch)
    - [CreateBranchRequest](#bytebase-v1-CreateBranchRequest)
    - [DeleteBranchRequest](#bytebase-v1-DeleteBranchRequest)
    - [DiffDatabaseRequest](#bytebase-v1-DiffDatabaseRequest)
    - [DiffDatabaseResponse](#bytebase-v1-DiffDatabaseResponse)
    - [DiffMetadataRequest](#bytebase-v1-DiffMetadataRequest)
    - [DiffMetadataResponse](#bytebase-v1-DiffMetadataResponse)
    - [GetBranchRequest](#bytebase-v1-GetBranchRequest)
    - [ListBranchesRequest](#bytebase-v1-ListBranchesRequest)
    - [ListBranchesResponse](#bytebase-v1-ListBranchesResponse)
    - [MergeBranchRequest](#bytebase-v1-MergeBranchRequest)
    - [RebaseBranchRequest](#bytebase-v1-RebaseBranchRequest)
    - [RebaseBranchResponse](#bytebase-v1-RebaseBranchResponse)
    - [UpdateBranchRequest](#bytebase-v1-UpdateBranchRequest)
  
    - [BranchView](#bytebase-v1-BranchView)
  
    - [BranchService](#bytebase-v1-BranchService)
  
- [v1/cel_service.proto](#v1_cel_service-proto)
    - [BatchDeparseRequest](#bytebase-v1-BatchDeparseRequest)
    - [BatchDeparseResponse](#bytebase-v1-BatchDeparseResponse)
    - [BatchParseRequest](#bytebase-v1-BatchParseRequest)
    - [BatchParseResponse](#bytebase-v1-BatchParseResponse)
  
    - [CelService](#bytebase-v1-CelService)
  
- [v1/changelist_service.proto](#v1_changelist_service-proto)
    - [Changelist](#bytebase-v1-Changelist)
    - [Changelist.Change](#bytebase-v1-Changelist-Change)
    - [CreateChangelistRequest](#bytebase-v1-CreateChangelistRequest)
    - [DeleteChangelistRequest](#bytebase-v1-DeleteChangelistRequest)
    - [GetChangelistRequest](#bytebase-v1-GetChangelistRequest)
    - [ListChangelistsRequest](#bytebase-v1-ListChangelistsRequest)
    - [ListChangelistsResponse](#bytebase-v1-ListChangelistsResponse)
    - [UpdateChangelistRequest](#bytebase-v1-UpdateChangelistRequest)
  
    - [ChangelistService](#bytebase-v1-ChangelistService)
  
- [v1/environment_service.proto](#v1_environment_service-proto)
    - [CreateEnvironmentRequest](#bytebase-v1-CreateEnvironmentRequest)
    - [DeleteEnvironmentRequest](#bytebase-v1-DeleteEnvironmentRequest)
    - [Environment](#bytebase-v1-Environment)
    - [GetEnvironmentRequest](#bytebase-v1-GetEnvironmentRequest)
    - [ListEnvironmentsRequest](#bytebase-v1-ListEnvironmentsRequest)
    - [ListEnvironmentsResponse](#bytebase-v1-ListEnvironmentsResponse)
    - [UndeleteEnvironmentRequest](#bytebase-v1-UndeleteEnvironmentRequest)
    - [UpdateEnvironmentRequest](#bytebase-v1-UpdateEnvironmentRequest)
  
    - [EnvironmentTier](#bytebase-v1-EnvironmentTier)
  
    - [EnvironmentService](#bytebase-v1-EnvironmentService)
  
- [v1/iam_policy.proto](#v1_iam_policy-proto)
    - [Binding](#bytebase-v1-Binding)
    - [GetIamPolicyRequest](#bytebase-v1-GetIamPolicyRequest)
    - [IamPolicy](#bytebase-v1-IamPolicy)
    - [SetIamPolicyRequest](#bytebase-v1-SetIamPolicyRequest)
  
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
    - [GrantRequest](#bytebase-v1-GrantRequest)
    - [Issue](#bytebase-v1-Issue)
    - [Issue.Approver](#bytebase-v1-Issue-Approver)
    - [Issue.TaskStatusCountEntry](#bytebase-v1-Issue-TaskStatusCountEntry)
    - [IssueComment](#bytebase-v1-IssueComment)
    - [IssueComment.Approval](#bytebase-v1-IssueComment-Approval)
    - [IssueComment.IssueUpdate](#bytebase-v1-IssueComment-IssueUpdate)
    - [IssueComment.StageEnd](#bytebase-v1-IssueComment-StageEnd)
    - [IssueComment.TaskPriorBackup](#bytebase-v1-IssueComment-TaskPriorBackup)
    - [IssueComment.TaskPriorBackup.Table](#bytebase-v1-IssueComment-TaskPriorBackup-Table)
    - [IssueComment.TaskUpdate](#bytebase-v1-IssueComment-TaskUpdate)
    - [ListIssueCommentsRequest](#bytebase-v1-ListIssueCommentsRequest)
    - [ListIssueCommentsResponse](#bytebase-v1-ListIssueCommentsResponse)
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
    - [Issue.RiskLevel](#bytebase-v1-Issue-RiskLevel)
    - [Issue.Type](#bytebase-v1-Issue-Type)
    - [IssueComment.Approval.Status](#bytebase-v1-IssueComment-Approval-Status)
    - [IssueComment.TaskUpdate.Status](#bytebase-v1-IssueComment-TaskUpdate-Status)
    - [IssueStatus](#bytebase-v1-IssueStatus)
  
    - [IssueService](#bytebase-v1-IssueService)
  
- [v1/org_policy_service.proto](#v1_org_policy_service-proto)
    - [CreatePolicyRequest](#bytebase-v1-CreatePolicyRequest)
    - [DeletePolicyRequest](#bytebase-v1-DeletePolicyRequest)
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
    - [RestrictIssueCreationForSQLReviewPolicy](#bytebase-v1-RestrictIssueCreationForSQLReviewPolicy)
    - [RolloutPolicy](#bytebase-v1-RolloutPolicy)
    - [SQLReviewRule](#bytebase-v1-SQLReviewRule)
    - [SlowQueryPolicy](#bytebase-v1-SlowQueryPolicy)
    - [TagPolicy](#bytebase-v1-TagPolicy)
    - [TagPolicy.TagsEntry](#bytebase-v1-TagPolicy-TagsEntry)
    - [UpdatePolicyRequest](#bytebase-v1-UpdatePolicyRequest)
  
    - [MaskingExceptionPolicy.MaskingException.Action](#bytebase-v1-MaskingExceptionPolicy-MaskingException-Action)
    - [PolicyResourceType](#bytebase-v1-PolicyResourceType)
    - [PolicyType](#bytebase-v1-PolicyType)
    - [SQLReviewRuleLevel](#bytebase-v1-SQLReviewRuleLevel)
  
    - [OrgPolicyService](#bytebase-v1-OrgPolicyService)
  
- [v1/plan_service.proto](#v1_plan_service-proto)
    - [BatchCancelPlanCheckRunsRequest](#bytebase-v1-BatchCancelPlanCheckRunsRequest)
    - [BatchCancelPlanCheckRunsResponse](#bytebase-v1-BatchCancelPlanCheckRunsResponse)
    - [CreatePlanRequest](#bytebase-v1-CreatePlanRequest)
    - [GetPlanRequest](#bytebase-v1-GetPlanRequest)
    - [ListPlanCheckRunsRequest](#bytebase-v1-ListPlanCheckRunsRequest)
    - [ListPlanCheckRunsResponse](#bytebase-v1-ListPlanCheckRunsResponse)
    - [ListPlansRequest](#bytebase-v1-ListPlansRequest)
    - [ListPlansResponse](#bytebase-v1-ListPlansResponse)
    - [Plan](#bytebase-v1-Plan)
    - [Plan.ChangeDatabaseConfig](#bytebase-v1-Plan-ChangeDatabaseConfig)
    - [Plan.ChangeDatabaseConfig.GhostFlagsEntry](#bytebase-v1-Plan-ChangeDatabaseConfig-GhostFlagsEntry)
    - [Plan.ChangeDatabaseConfig.PreUpdateBackupDetail](#bytebase-v1-Plan-ChangeDatabaseConfig-PreUpdateBackupDetail)
    - [Plan.CreateDatabaseConfig](#bytebase-v1-Plan-CreateDatabaseConfig)
    - [Plan.CreateDatabaseConfig.LabelsEntry](#bytebase-v1-Plan-CreateDatabaseConfig-LabelsEntry)
    - [Plan.ExportDataConfig](#bytebase-v1-Plan-ExportDataConfig)
    - [Plan.PlanCheckRunStatusCountEntry](#bytebase-v1-Plan-PlanCheckRunStatusCountEntry)
    - [Plan.Spec](#bytebase-v1-Plan-Spec)
    - [Plan.Step](#bytebase-v1-Plan-Step)
    - [Plan.VCSSource](#bytebase-v1-Plan-VCSSource)
    - [PlanCheckRun](#bytebase-v1-PlanCheckRun)
    - [PlanCheckRun.Result](#bytebase-v1-PlanCheckRun-Result)
    - [PlanCheckRun.Result.SqlReviewReport](#bytebase-v1-PlanCheckRun-Result-SqlReviewReport)
    - [PlanCheckRun.Result.SqlSummaryReport](#bytebase-v1-PlanCheckRun-Result-SqlSummaryReport)
    - [RunPlanChecksRequest](#bytebase-v1-RunPlanChecksRequest)
    - [RunPlanChecksResponse](#bytebase-v1-RunPlanChecksResponse)
    - [SearchPlansRequest](#bytebase-v1-SearchPlansRequest)
    - [SearchPlansResponse](#bytebase-v1-SearchPlansResponse)
    - [UpdatePlanRequest](#bytebase-v1-UpdatePlanRequest)
  
    - [Plan.ChangeDatabaseConfig.Type](#bytebase-v1-Plan-ChangeDatabaseConfig-Type)
    - [PlanCheckRun.Result.Status](#bytebase-v1-PlanCheckRun-Result-Status)
    - [PlanCheckRun.Status](#bytebase-v1-PlanCheckRun-Status)
    - [PlanCheckRun.Type](#bytebase-v1-PlanCheckRun-Type)
  
    - [PlanService](#bytebase-v1-PlanService)
  
- [v1/project_service.proto](#v1_project_service-proto)
    - [Activity](#bytebase-v1-Activity)
    - [AddWebhookRequest](#bytebase-v1-AddWebhookRequest)
    - [BatchGetIamPolicyRequest](#bytebase-v1-BatchGetIamPolicyRequest)
    - [BatchGetIamPolicyResponse](#bytebase-v1-BatchGetIamPolicyResponse)
    - [BatchGetIamPolicyResponse.PolicyResult](#bytebase-v1-BatchGetIamPolicyResponse-PolicyResult)
    - [CreateDatabaseGroupRequest](#bytebase-v1-CreateDatabaseGroupRequest)
    - [CreateProjectRequest](#bytebase-v1-CreateProjectRequest)
    - [DatabaseGroup](#bytebase-v1-DatabaseGroup)
    - [DatabaseGroup.Database](#bytebase-v1-DatabaseGroup-Database)
    - [DeleteDatabaseGroupRequest](#bytebase-v1-DeleteDatabaseGroupRequest)
    - [DeleteProjectRequest](#bytebase-v1-DeleteProjectRequest)
    - [DeploymentConfig](#bytebase-v1-DeploymentConfig)
    - [DeploymentSpec](#bytebase-v1-DeploymentSpec)
    - [GetDatabaseGroupRequest](#bytebase-v1-GetDatabaseGroupRequest)
    - [GetDeploymentConfigRequest](#bytebase-v1-GetDeploymentConfigRequest)
    - [GetProjectProtectionRulesRequest](#bytebase-v1-GetProjectProtectionRulesRequest)
    - [GetProjectRequest](#bytebase-v1-GetProjectRequest)
    - [Label](#bytebase-v1-Label)
    - [LabelSelector](#bytebase-v1-LabelSelector)
    - [LabelSelectorRequirement](#bytebase-v1-LabelSelectorRequirement)
    - [ListDatabaseGroupsRequest](#bytebase-v1-ListDatabaseGroupsRequest)
    - [ListDatabaseGroupsResponse](#bytebase-v1-ListDatabaseGroupsResponse)
    - [ListProjectsRequest](#bytebase-v1-ListProjectsRequest)
    - [ListProjectsResponse](#bytebase-v1-ListProjectsResponse)
    - [Project](#bytebase-v1-Project)
    - [ProtectionRule](#bytebase-v1-ProtectionRule)
    - [ProtectionRules](#bytebase-v1-ProtectionRules)
    - [RemoveWebhookRequest](#bytebase-v1-RemoveWebhookRequest)
    - [Schedule](#bytebase-v1-Schedule)
    - [ScheduleDeployment](#bytebase-v1-ScheduleDeployment)
    - [SearchProjectsRequest](#bytebase-v1-SearchProjectsRequest)
    - [SearchProjectsResponse](#bytebase-v1-SearchProjectsResponse)
    - [TestWebhookRequest](#bytebase-v1-TestWebhookRequest)
    - [TestWebhookResponse](#bytebase-v1-TestWebhookResponse)
    - [UndeleteProjectRequest](#bytebase-v1-UndeleteProjectRequest)
    - [UpdateDatabaseGroupRequest](#bytebase-v1-UpdateDatabaseGroupRequest)
    - [UpdateDeploymentConfigRequest](#bytebase-v1-UpdateDeploymentConfigRequest)
    - [UpdateProjectProtectionRulesRequest](#bytebase-v1-UpdateProjectProtectionRulesRequest)
    - [UpdateProjectRequest](#bytebase-v1-UpdateProjectRequest)
    - [UpdateWebhookRequest](#bytebase-v1-UpdateWebhookRequest)
    - [Webhook](#bytebase-v1-Webhook)
  
    - [Activity.Type](#bytebase-v1-Activity-Type)
    - [DatabaseGroupView](#bytebase-v1-DatabaseGroupView)
    - [OperatorType](#bytebase-v1-OperatorType)
    - [ProtectionRule.BranchSource](#bytebase-v1-ProtectionRule-BranchSource)
    - [ProtectionRule.Target](#bytebase-v1-ProtectionRule-Target)
    - [Webhook.Type](#bytebase-v1-Webhook-Type)
    - [Workflow](#bytebase-v1-Workflow)
  
    - [ProjectService](#bytebase-v1-ProjectService)
  
- [v1/review_config_service.proto](#v1_review_config_service-proto)
    - [CreateReviewConfigRequest](#bytebase-v1-CreateReviewConfigRequest)
    - [DeleteReviewConfigRequest](#bytebase-v1-DeleteReviewConfigRequest)
    - [GetReviewConfigRequest](#bytebase-v1-GetReviewConfigRequest)
    - [ListReviewConfigsRequest](#bytebase-v1-ListReviewConfigsRequest)
    - [ListReviewConfigsResponse](#bytebase-v1-ListReviewConfigsResponse)
    - [ReviewConfig](#bytebase-v1-ReviewConfig)
    - [UpdateReviewConfigRequest](#bytebase-v1-UpdateReviewConfigRequest)
  
    - [ReviewConfigService](#bytebase-v1-ReviewConfigService)
  
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
    - [CreateRolloutRequest](#bytebase-v1-CreateRolloutRequest)
    - [GetRolloutRequest](#bytebase-v1-GetRolloutRequest)
    - [GetTaskRunLogRequest](#bytebase-v1-GetTaskRunLogRequest)
    - [GetTaskRunSessionRequest](#bytebase-v1-GetTaskRunSessionRequest)
    - [ListTaskRunsRequest](#bytebase-v1-ListTaskRunsRequest)
    - [ListTaskRunsResponse](#bytebase-v1-ListTaskRunsResponse)
    - [PreviewRolloutRequest](#bytebase-v1-PreviewRolloutRequest)
    - [Rollout](#bytebase-v1-Rollout)
    - [Stage](#bytebase-v1-Stage)
    - [Task](#bytebase-v1-Task)
    - [Task.DatabaseCreate](#bytebase-v1-Task-DatabaseCreate)
    - [Task.DatabaseCreate.LabelsEntry](#bytebase-v1-Task-DatabaseCreate-LabelsEntry)
    - [Task.DatabaseDataExport](#bytebase-v1-Task-DatabaseDataExport)
    - [Task.DatabaseDataUpdate](#bytebase-v1-Task-DatabaseDataUpdate)
    - [Task.DatabaseSchemaBaseline](#bytebase-v1-Task-DatabaseSchemaBaseline)
    - [Task.DatabaseSchemaUpdate](#bytebase-v1-Task-DatabaseSchemaUpdate)
    - [TaskRun](#bytebase-v1-TaskRun)
    - [TaskRun.ExecutionDetail](#bytebase-v1-TaskRun-ExecutionDetail)
    - [TaskRun.ExecutionDetail.Position](#bytebase-v1-TaskRun-ExecutionDetail-Position)
    - [TaskRunLog](#bytebase-v1-TaskRunLog)
    - [TaskRunLogEntry](#bytebase-v1-TaskRunLogEntry)
    - [TaskRunLogEntry.CommandExecute](#bytebase-v1-TaskRunLogEntry-CommandExecute)
    - [TaskRunLogEntry.CommandExecute.CommandResponse](#bytebase-v1-TaskRunLogEntry-CommandExecute-CommandResponse)
    - [TaskRunLogEntry.DatabaseSync](#bytebase-v1-TaskRunLogEntry-DatabaseSync)
    - [TaskRunLogEntry.SchemaDump](#bytebase-v1-TaskRunLogEntry-SchemaDump)
    - [TaskRunLogEntry.TaskRunStatusUpdate](#bytebase-v1-TaskRunLogEntry-TaskRunStatusUpdate)
    - [TaskRunLogEntry.TransactionControl](#bytebase-v1-TaskRunLogEntry-TransactionControl)
    - [TaskRunSession](#bytebase-v1-TaskRunSession)
    - [TaskRunSession.Postgres](#bytebase-v1-TaskRunSession-Postgres)
    - [TaskRunSession.Postgres.Session](#bytebase-v1-TaskRunSession-Postgres-Session)
  
    - [Task.Status](#bytebase-v1-Task-Status)
    - [Task.Type](#bytebase-v1-Task-Type)
    - [TaskRun.ExecutionStatus](#bytebase-v1-TaskRun-ExecutionStatus)
    - [TaskRun.ExportArchiveStatus](#bytebase-v1-TaskRun-ExportArchiveStatus)
    - [TaskRun.Status](#bytebase-v1-TaskRun-Status)
    - [TaskRunLogEntry.TaskRunStatusUpdate.Status](#bytebase-v1-TaskRunLogEntry-TaskRunStatusUpdate-Status)
    - [TaskRunLogEntry.TransactionControl.Type](#bytebase-v1-TaskRunLogEntry-TransactionControl-Type)
    - [TaskRunLogEntry.Type](#bytebase-v1-TaskRunLogEntry-Type)
  
    - [RolloutService](#bytebase-v1-RolloutService)
  
- [v1/subscription_service.proto](#v1_subscription_service-proto)
    - [Feature](#bytebase-v1-Feature)
    - [Feature.MatrixEntry](#bytebase-v1-Feature-MatrixEntry)
    - [FeatureMatrix](#bytebase-v1-FeatureMatrix)
    - [GetFeatureMatrixRequest](#bytebase-v1-GetFeatureMatrixRequest)
    - [GetSubscriptionRequest](#bytebase-v1-GetSubscriptionRequest)
    - [PatchSubscription](#bytebase-v1-PatchSubscription)
    - [Subscription](#bytebase-v1-Subscription)
    - [UpdateSubscriptionRequest](#bytebase-v1-UpdateSubscriptionRequest)
  
    - [PlanType](#bytebase-v1-PlanType)
  
    - [SubscriptionService](#bytebase-v1-SubscriptionService)
  
- [v1/setting_service.proto](#v1_setting_service-proto)
    - [AgentPluginSetting](#bytebase-v1-AgentPluginSetting)
    - [Announcement](#bytebase-v1-Announcement)
    - [AppIMSetting](#bytebase-v1-AppIMSetting)
    - [AppIMSetting.Feishu](#bytebase-v1-AppIMSetting-Feishu)
    - [AppIMSetting.Slack](#bytebase-v1-AppIMSetting-Slack)
    - [AppIMSetting.Wecom](#bytebase-v1-AppIMSetting-Wecom)
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
    - [MaskingAlgorithmSetting](#bytebase-v1-MaskingAlgorithmSetting)
    - [MaskingAlgorithmSetting.Algorithm](#bytebase-v1-MaskingAlgorithmSetting-Algorithm)
    - [MaskingAlgorithmSetting.Algorithm.FullMask](#bytebase-v1-MaskingAlgorithmSetting-Algorithm-FullMask)
    - [MaskingAlgorithmSetting.Algorithm.InnerOuterMask](#bytebase-v1-MaskingAlgorithmSetting-Algorithm-InnerOuterMask)
    - [MaskingAlgorithmSetting.Algorithm.MD5Mask](#bytebase-v1-MaskingAlgorithmSetting-Algorithm-MD5Mask)
    - [MaskingAlgorithmSetting.Algorithm.RangeMask](#bytebase-v1-MaskingAlgorithmSetting-Algorithm-RangeMask)
    - [MaskingAlgorithmSetting.Algorithm.RangeMask.Slice](#bytebase-v1-MaskingAlgorithmSetting-Algorithm-RangeMask-Slice)
    - [SMTPMailDeliverySettingValue](#bytebase-v1-SMTPMailDeliverySettingValue)
    - [SchemaTemplateSetting](#bytebase-v1-SchemaTemplateSetting)
    - [SchemaTemplateSetting.ColumnType](#bytebase-v1-SchemaTemplateSetting-ColumnType)
    - [SchemaTemplateSetting.FieldTemplate](#bytebase-v1-SchemaTemplateSetting-FieldTemplate)
    - [SchemaTemplateSetting.TableTemplate](#bytebase-v1-SchemaTemplateSetting-TableTemplate)
    - [SemanticTypeSetting](#bytebase-v1-SemanticTypeSetting)
    - [SemanticTypeSetting.SemanticType](#bytebase-v1-SemanticTypeSetting-SemanticType)
    - [Setting](#bytebase-v1-Setting)
    - [UpdateSettingRequest](#bytebase-v1-UpdateSettingRequest)
    - [Value](#bytebase-v1-Value)
    - [WorkspaceApprovalSetting](#bytebase-v1-WorkspaceApprovalSetting)
    - [WorkspaceApprovalSetting.Rule](#bytebase-v1-WorkspaceApprovalSetting-Rule)
    - [WorkspaceProfileSetting](#bytebase-v1-WorkspaceProfileSetting)
    - [WorkspaceTrialSetting](#bytebase-v1-WorkspaceTrialSetting)
  
    - [Announcement.AlertLevel](#bytebase-v1-Announcement-AlertLevel)
    - [DatabaseChangeMode](#bytebase-v1-DatabaseChangeMode)
    - [MaskingAlgorithmSetting.Algorithm.InnerOuterMask.MaskType](#bytebase-v1-MaskingAlgorithmSetting-Algorithm-InnerOuterMask-MaskType)
    - [SMTPMailDeliverySettingValue.Authentication](#bytebase-v1-SMTPMailDeliverySettingValue-Authentication)
    - [SMTPMailDeliverySettingValue.Encryption](#bytebase-v1-SMTPMailDeliverySettingValue-Encryption)
  
    - [SettingService](#bytebase-v1-SettingService)
  
- [v1/sheet_service.proto](#v1_sheet_service-proto)
    - [CreateSheetRequest](#bytebase-v1-CreateSheetRequest)
    - [GetSheetRequest](#bytebase-v1-GetSheetRequest)
    - [Sheet](#bytebase-v1-Sheet)
    - [SheetCommand](#bytebase-v1-SheetCommand)
    - [SheetPayload](#bytebase-v1-SheetPayload)
    - [UpdateSheetRequest](#bytebase-v1-UpdateSheetRequest)
  
    - [SheetPayload.Type](#bytebase-v1-SheetPayload-Type)
  
    - [SheetService](#bytebase-v1-SheetService)
  
- [v1/sql_service.proto](#v1_sql_service-proto)
    - [AdminExecuteRequest](#bytebase-v1-AdminExecuteRequest)
    - [AdminExecuteResponse](#bytebase-v1-AdminExecuteResponse)
    - [Advice](#bytebase-v1-Advice)
    - [CheckRequest](#bytebase-v1-CheckRequest)
    - [CheckResponse](#bytebase-v1-CheckResponse)
    - [DifferPreviewRequest](#bytebase-v1-DifferPreviewRequest)
    - [DifferPreviewResponse](#bytebase-v1-DifferPreviewResponse)
    - [ExecuteRequest](#bytebase-v1-ExecuteRequest)
    - [ExecuteResponse](#bytebase-v1-ExecuteResponse)
    - [ExportRequest](#bytebase-v1-ExportRequest)
    - [ExportResponse](#bytebase-v1-ExportResponse)
    - [GenerateRestoreSQLRequest](#bytebase-v1-GenerateRestoreSQLRequest)
    - [GenerateRestoreSQLResponse](#bytebase-v1-GenerateRestoreSQLResponse)
    - [ParseMyBatisMapperRequest](#bytebase-v1-ParseMyBatisMapperRequest)
    - [ParseMyBatisMapperResponse](#bytebase-v1-ParseMyBatisMapperResponse)
    - [PrettyRequest](#bytebase-v1-PrettyRequest)
    - [PrettyResponse](#bytebase-v1-PrettyResponse)
    - [QueryHistory](#bytebase-v1-QueryHistory)
    - [QueryRequest](#bytebase-v1-QueryRequest)
    - [QueryResponse](#bytebase-v1-QueryResponse)
    - [QueryResult](#bytebase-v1-QueryResult)
    - [QueryRow](#bytebase-v1-QueryRow)
    - [RowValue](#bytebase-v1-RowValue)
    - [SearchQueryHistoriesRequest](#bytebase-v1-SearchQueryHistoriesRequest)
    - [SearchQueryHistoriesResponse](#bytebase-v1-SearchQueryHistoriesResponse)
    - [StringifyMetadataRequest](#bytebase-v1-StringifyMetadataRequest)
    - [StringifyMetadataResponse](#bytebase-v1-StringifyMetadataResponse)
  
    - [Advice.Status](#bytebase-v1-Advice-Status)
    - [CheckRequest.ChangeType](#bytebase-v1-CheckRequest-ChangeType)
    - [QueryHistory.Type](#bytebase-v1-QueryHistory-Type)
  
    - [SQLService](#bytebase-v1-SQLService)
  
- [v1/user_group.proto](#v1_user_group-proto)
    - [CreateUserGroupRequest](#bytebase-v1-CreateUserGroupRequest)
    - [DeleteUserGroupRequest](#bytebase-v1-DeleteUserGroupRequest)
    - [GetUserGroupRequest](#bytebase-v1-GetUserGroupRequest)
    - [ListUserGroupsRequest](#bytebase-v1-ListUserGroupsRequest)
    - [ListUserGroupsResponse](#bytebase-v1-ListUserGroupsResponse)
    - [UpdateUserGroupRequest](#bytebase-v1-UpdateUserGroupRequest)
    - [UserGroup](#bytebase-v1-UserGroup)
    - [UserGroupMember](#bytebase-v1-UserGroupMember)
  
    - [UserGroupMember.Role](#bytebase-v1-UserGroupMember-Role)
  
    - [UserGroupService](#bytebase-v1-UserGroupService)
  
- [v1/vcs_connector_service.proto](#v1_vcs_connector_service-proto)
    - [CreateVCSConnectorRequest](#bytebase-v1-CreateVCSConnectorRequest)
    - [DeleteVCSConnectorRequest](#bytebase-v1-DeleteVCSConnectorRequest)
    - [GetVCSConnectorRequest](#bytebase-v1-GetVCSConnectorRequest)
    - [ListVCSConnectorsRequest](#bytebase-v1-ListVCSConnectorsRequest)
    - [ListVCSConnectorsResponse](#bytebase-v1-ListVCSConnectorsResponse)
    - [UpdateVCSConnectorRequest](#bytebase-v1-UpdateVCSConnectorRequest)
    - [VCSConnector](#bytebase-v1-VCSConnector)
  
    - [VCSConnectorService](#bytebase-v1-VCSConnectorService)
  
- [v1/vcs_provider_service.proto](#v1_vcs_provider_service-proto)
    - [CreateVCSProviderRequest](#bytebase-v1-CreateVCSProviderRequest)
    - [DeleteVCSProviderRequest](#bytebase-v1-DeleteVCSProviderRequest)
    - [GetVCSProviderRequest](#bytebase-v1-GetVCSProviderRequest)
    - [ListVCSConnectorsInProviderRequest](#bytebase-v1-ListVCSConnectorsInProviderRequest)
    - [ListVCSConnectorsInProviderResponse](#bytebase-v1-ListVCSConnectorsInProviderResponse)
    - [ListVCSProvidersRequest](#bytebase-v1-ListVCSProvidersRequest)
    - [ListVCSProvidersResponse](#bytebase-v1-ListVCSProvidersResponse)
    - [SearchVCSProviderRepositoriesRequest](#bytebase-v1-SearchVCSProviderRepositoriesRequest)
    - [SearchVCSProviderRepositoriesResponse](#bytebase-v1-SearchVCSProviderRepositoriesResponse)
    - [UpdateVCSProviderRequest](#bytebase-v1-UpdateVCSProviderRequest)
    - [VCSProvider](#bytebase-v1-VCSProvider)
    - [VCSRepository](#bytebase-v1-VCSRepository)
  
    - [VCSProviderService](#bytebase-v1-VCSProviderService)
  
- [v1/worksheet_service.proto](#v1_worksheet_service-proto)
    - [CreateWorksheetRequest](#bytebase-v1-CreateWorksheetRequest)
    - [DeleteWorksheetRequest](#bytebase-v1-DeleteWorksheetRequest)
    - [GetWorksheetRequest](#bytebase-v1-GetWorksheetRequest)
    - [SearchWorksheetsRequest](#bytebase-v1-SearchWorksheetsRequest)
    - [SearchWorksheetsResponse](#bytebase-v1-SearchWorksheetsResponse)
    - [UpdateWorksheetOrganizerRequest](#bytebase-v1-UpdateWorksheetOrganizerRequest)
    - [UpdateWorksheetRequest](#bytebase-v1-UpdateWorksheetRequest)
    - [Worksheet](#bytebase-v1-Worksheet)
    - [WorksheetOrganizer](#bytebase-v1-WorksheetOrganizer)
  
    - [Worksheet.Visibility](#bytebase-v1-Worksheet-Visibility)
  
    - [WorksheetService](#bytebase-v1-WorksheetService)
  
- [v1/workspace_service.proto](#v1_workspace_service-proto)
    - [WorkspaceService](#bytebase-v1-WorkspaceService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="v1_annotation-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/annotation.proto


 


<a name="bytebase-v1-AuthMethod"></a>

### AuthMethod


| Name | Number | Description |
| ---- | ------ | ----------- |
| AUTH_METHOD_UNSPECIFIED | 0 |  |
| IAM | 1 | IAM uses the standard IAM authorization check on the organizational resources. |
| CUSTOM | 2 | Custom authorization method. |


 


<a name="v1_annotation-proto-extensions"></a>

### File-level Extensions
| Extension | Type | Base | Number | Description |
| --------- | ---- | ---- | ------ | ----------- |
| allow_without_credential | bool | .google.protobuf.MethodOptions | 100000 |  |
| auth_method | AuthMethod | .google.protobuf.MethodOptions | 100002 |  |
| permission | string | .google.protobuf.MethodOptions | 100001 |  |

 

 



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
| lsp | [bool](#bool) |  | lsp is the enablement of lsp in SQL Editor. |
| pre_update_backup | [bool](#bool) |  | lsp is the enablement of data backup prior to data update. |
| iam_guard | [bool](#bool) |  | iam_guard is the enablement of IAM checks. |
| unlicensed_features | [string](#string) | repeated |  |






<a name="bytebase-v1-DebugLog"></a>

### DebugLog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| record_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| request_path | [string](#string) |  |  |
| user | [string](#string) |  |  |
| error | [string](#string) |  |  |
| stack_trace | [string](#string) |  |  |






<a name="bytebase-v1-DeleteCacheRequest"></a>

### DeleteCacheRequest







<a name="bytebase-v1-GetActuatorInfoRequest"></a>

### GetActuatorInfoRequest







<a name="bytebase-v1-GetResourcePackageRequest"></a>

### GetResourcePackageRequest
The request message for getting the theme resource.






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






<a name="bytebase-v1-ResourcePackage"></a>

### ResourcePackage
The theme resources.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| logo | [bytes](#bytes) |  | The branding logo. |






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
| GetResourcePackage | [GetResourcePackageRequest](#bytebase-v1-GetResourcePackageRequest) | [ResourcePackage](#bytebase-v1-ResourcePackage) |  |

 



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
| database_schema_drift_detail | [Anomaly.DatabaseSchemaDriftDetail](#bytebase-v1-Anomaly-DatabaseSchemaDriftDetail) |  |  |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






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
| filter | [string](#string) |  | filter is the filter to apply on the search anomaly request, follow the [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) syntax. Only support filter by resource and type for now. For example: Search the anomalies of a specific resource: &#39;resource=&#34;instances/{instance}&#34;.&#39; Search the specified types of anomalies: &#39;type=&#34;MIGRATION_SCHEMA&#34;.&#39; |
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
| DATABASE_CONNECTION | 5 | Database level anomaly.

DATABASE_CONNECTION is the anomaly type for database connection, e.g. the database had been deleted. |
| DATABASE_SCHEMA_DRIFT | 6 | DATABASE_SCHEMA_DRIFT is the anomaly type for database schema drift, e.g. the database schema had been changed without bytebase migration. |


 

 


<a name="bytebase-v1-AnomalyService"></a>

### AnomalyService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| SearchAnomalies | [SearchAnomaliesRequest](#bytebase-v1-SearchAnomaliesRequest) | [SearchAnomaliesResponse](#bytebase-v1-SearchAnomaliesResponse) |  |

 



<a name="v1_common-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/common.proto



<a name="bytebase-v1-Position"></a>

### Position



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| line | [int32](#int32) |  |  |
| column | [int32](#int32) |  |  |





 


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
| OCEANBASE_ORACLE | 17 |  |
| STARROCKS | 18 |  |
| DORIS | 19 |  |
| HIVE | 20 |  |
| ELASTICSEARCH | 21 |  |
| BIGQUERY | 22 |  |
| DYNAMODB | 23 |  |
| DATABRICKS | 24 |  |



<a name="bytebase-v1-ExportFormat"></a>

### ExportFormat


| Name | Number | Description |
| ---- | ------ | ----------- |
| FORMAT_UNSPECIFIED | 0 |  |
| CSV | 1 |  |
| JSON | 2 |  |
| SQL | 3 |  |
| XLSX | 4 |  |



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



<a name="bytebase-v1-VCSType"></a>

### VCSType


| Name | Number | Description |
| ---- | ------ | ----------- |
| VCS_TYPE_UNSPECIFIED | 0 |  |
| GITHUB | 1 | GitHub type. Using for GitHub community edition(ce). |
| GITLAB | 2 | GitLab type. Using for GitLab community edition(ce) and enterprise edition(ee). |
| BITBUCKET | 3 | BitBucket type. Using for BitBucket cloud or BitBucket server. |
| AZURE_DEVOPS | 4 | Azure DevOps. Using for Azure DevOps GitOps workflow. |


 

 

 



<a name="v1_audit_log_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/audit_log_service.proto



<a name="bytebase-v1-AuditLog"></a>

### AuditLog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the log. Formats: - projects/{project}/auditLogs/{uid} - workspaces/{workspace}/auditLogs/{uid} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| user | [string](#string) |  | Format: users/d@d.com |
| method | [string](#string) |  | e.g. `/bytebase.v1.SQLService/Query`, `bb.project.repository.push` |
| severity | [AuditLog.Severity](#bytebase-v1-AuditLog-Severity) |  |  |
| resource | [string](#string) |  | The associated resource. |
| request | [string](#string) |  | JSON-encoded request. |
| response | [string](#string) |  | JSON-encoded response. Some fields are omitted because they are too large or contain sensitive information. |
| status | [google.rpc.Status](#google-rpc-Status) |  |  |






<a name="bytebase-v1-ExportAuditLogsRequest"></a>

### ExportAuditLogsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filter | [string](#string) |  | The filter of the log. It should be a valid CEL expression. For example: - filter = &#34;method == &#39;/bytebase.v1.SQLService/Query&#39;&#34; - filter = &#34;method == &#39;/bytebase.v1.SQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39;&#34; - filter = &#34;method == &#39;/bytebase.v1.SQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39; &amp;&amp; user == &#39;users/bb@bytebase.com&#39;&#34; - filter = &#34;method == &#39;/bytebase.v1.SQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39; &amp;&amp; create_time &lt;= &#39;2021-01-01T00:00:00Z&#39; &amp;&amp; create_time &gt;= &#39;2020-01-01T00:00:00Z&#39;&#34; |
| order_by | [string](#string) |  | The order by of the log. Only support order by create_time. For example: - order_by = &#34;create_time asc&#34; - order_by = &#34;create_time desc&#34; |
| format | [ExportFormat](#bytebase-v1-ExportFormat) |  | The export format. |






<a name="bytebase-v1-ExportAuditLogsResponse"></a>

### ExportAuditLogsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| content | [bytes](#bytes) |  |  |






<a name="bytebase-v1-SearchAuditLogsRequest"></a>

### SearchAuditLogsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filter | [string](#string) |  | The filter of the log. It should be a valid CEL expression. For example: - filter = &#34;method == &#39;/bytebase.v1.SQLService/Query&#39;&#34; - filter = &#34;method == &#39;/bytebase.v1.SQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39;&#34; - filter = &#34;method == &#39;/bytebase.v1.SQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39; &amp;&amp; user == &#39;users/bb@bytebase.com&#39;&#34; - filter = &#34;method == &#39;/bytebase.v1.SQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39; &amp;&amp; create_time &lt;= &#39;2021-01-01T00:00:00Z&#39; &amp;&amp; create_time &gt;= &#39;2020-01-01T00:00:00Z&#39;&#34; |
| order_by | [string](#string) |  | The order by of the log. Only support order by create_time. For example: - order_by = &#34;create_time asc&#34; - order_by = &#34;create_time desc&#34; |
| page_size | [int32](#int32) |  | The maximum number of logs to return. The service may return fewer than this value. If unspecified, at most 100 log entries will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `SearchLogs` call. Provide this to retrieve the subsequent page. |






<a name="bytebase-v1-SearchAuditLogsResponse"></a>

### SearchAuditLogsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| audit_logs | [AuditLog](#bytebase-v1-AuditLog) | repeated |  |
| next_page_token | [string](#string) |  | A token to retrieve next page of log entities. Pass this value in the page_token field in the subsequent call to retrieve the next page of log entities. |





 


<a name="bytebase-v1-AuditLog-Severity"></a>

### AuditLog.Severity


| Name | Number | Description |
| ---- | ------ | ----------- |
| DEFAULT | 0 |  |
| DEBUG | 1 |  |
| INFO | 2 |  |
| NOTICE | 3 |  |
| WARNING | 4 |  |
| ERROR | 5 |  |
| CRITICAL | 6 |  |
| ALERT | 7 |  |
| EMERGENCY | 8 |  |


 

 


<a name="bytebase-v1-AuditLogService"></a>

### AuditLogService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| SearchAuditLogs | [SearchAuditLogsRequest](#bytebase-v1-SearchAuditLogsRequest) | [SearchAuditLogsResponse](#bytebase-v1-SearchAuditLogsResponse) |  |
| ExportAuditLogs | [ExportAuditLogsRequest](#bytebase-v1-ExportAuditLogsRequest) | [ExportAuditLogsResponse](#bytebase-v1-ExportAuditLogsResponse) |  |

 



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
| name | [string](#string) |  | The name of the user to retrieve. Format: users/{user uid or user email} |






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
| password | [string](#string) |  |  |
| service_key | [string](#string) |  |  |
| mfa_enabled | [bool](#bool) |  | The mfa_enabled flag means if the user has enabled MFA. |
| mfa_secret | [string](#string) |  | The mfa_secret is the temporary secret using in two phase verification. |
| recovery_codes | [string](#string) | repeated | The recovery_codes is the temporary recovery codes using in two phase verification. |
| phone | [string](#string) |  | Should be a valid E.164 compliant phone number. Could be empty. |





 


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

 



<a name="v1_instance_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/instance_service.proto



<a name="bytebase-v1-AddDataSourceRequest"></a>

### AddDataSourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance | [string](#string) |  | The name of the instance to add a data source to. Format: instances/{instance} |
| data_source | [DataSource](#bytebase-v1-DataSource) |  | Identified by data source ID. Only READ_ONLY data source can be added. |
| validate_only | [bool](#bool) |  | Validate only also tests the data source connection. |






<a name="bytebase-v1-BatchSyncInstanceRequest"></a>

### BatchSyncInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| requests | [SyncInstanceRequest](#bytebase-v1-SyncInstanceRequest) | repeated | The request message specifying the instances to sync. A maximum of 1000 instances can be synced in a batch. |






<a name="bytebase-v1-BatchSyncInstanceResponse"></a>

### BatchSyncInstanceResponse







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
| use_ssl | [bool](#bool) |  | Use SSL to connect to the data source. By default, we use system default SSL configuration. |
| ssl_ca | [string](#string) |  |  |
| ssl_cert | [string](#string) |  |  |
| ssl_key | [string](#string) |  |  |
| host | [string](#string) |  |  |
| port | [string](#string) |  |  |
| database | [string](#string) |  |  |
| srv | [bool](#bool) |  | srv, authentication_database and replica_set are used for MongoDB. |
| authentication_database | [string](#string) |  |  |
| sid | [string](#string) |  | sid and service_name are used for Oracle. |
| service_name | [string](#string) |  |  |
| ssh_host | [string](#string) |  | Connection over SSH. The hostname of the SSH server agent. Required. |
| ssh_port | [string](#string) |  | The port of the SSH server agent. It&#39;s 22 typically. Required. |
| ssh_user | [string](#string) |  | The user to login the server. Required. |
| ssh_password | [string](#string) |  | The password to login the server. If it&#39;s empty string, no password is required. |
| ssh_private_key | [string](#string) |  | The private key to login the server. If it&#39;s empty string, we will use the system default private key from os.Getenv(&#34;SSH_AUTH_SOCK&#34;). |
| authentication_private_key | [string](#string) |  | PKCS#8 private key in PEM format. If it&#39;s empty string, no private key is required. Used for authentication when connecting to the data source. |
| external_secret | [DataSourceExternalSecret](#bytebase-v1-DataSourceExternalSecret) |  |  |
| authentication_type | [DataSource.AuthenticationType](#bytebase-v1-DataSource-AuthenticationType) |  |  |
| sasl_config | [SASLConfig](#bytebase-v1-SASLConfig) |  |  |
| additional_addresses | [DataSource.Address](#bytebase-v1-DataSource-Address) | repeated | additional_addresses is used for MongoDB replica set. |
| replica_set | [string](#string) |  | replica_set is used for MongoDB replica set. |
| direct_connection | [bool](#bool) |  | direct_connection is used for MongoDB to dispatch all the operations to the node specified in the connection string. |
| region | [string](#string) |  | region is the location of where the DB is, works for AWS RDS. For example, us-east-1. |
| account_id | [string](#string) |  | account_id is used by Databricks. |
| warehouse_id | [string](#string) |  | warehouse_id is used by Databricks. |
| master_name | [string](#string) |  | master_name is the master name used by connecting redis-master via redis sentinel. |
| master_username | [string](#string) |  | master_username and master_password are master credentials used by redis sentinel mode. |
| master_password | [string](#string) |  |  |
| redis_type | [DataSource.RedisType](#bytebase-v1-DataSource-RedisType) |  |  |






<a name="bytebase-v1-DataSource-Address"></a>

### DataSource.Address



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| host | [string](#string) |  |  |
| port | [string](#string) |  |  |






<a name="bytebase-v1-DataSourceExternalSecret"></a>

### DataSourceExternalSecret



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secret_type | [DataSourceExternalSecret.SecretType](#bytebase-v1-DataSourceExternalSecret-SecretType) |  |  |
| url | [string](#string) |  |  |
| auth_type | [DataSourceExternalSecret.AuthType](#bytebase-v1-DataSourceExternalSecret-AuthType) |  |  |
| app_role | [DataSourceExternalSecret.AppRoleAuthOption](#bytebase-v1-DataSourceExternalSecret-AppRoleAuthOption) |  |  |
| token | [string](#string) |  |  |
| engine_name | [string](#string) |  | engine name is the name for secret engine. |
| secret_name | [string](#string) |  | the secret name in the engine to store the password. |
| password_key_name | [string](#string) |  | the key name for the password. |






<a name="bytebase-v1-DataSourceExternalSecret-AppRoleAuthOption"></a>

### DataSourceExternalSecret.AppRoleAuthOption



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role_id | [string](#string) |  |  |
| secret_id | [string](#string) |  | the secret id for the role without ttl. |
| type | [DataSourceExternalSecret.AppRoleAuthOption.SecretType](#bytebase-v1-DataSourceExternalSecret-AppRoleAuthOption-SecretType) |  |  |
| mount_path | [string](#string) |  | The path where the approle auth method is mounted. |






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
| sync_interval | [google.protobuf.Duration](#google-protobuf-Duration) |  | How often the instance is synced. |
| maximum_connections | [int32](#int32) |  | The maximum number of connections. The default is 10 if the value is unset or zero. |






<a name="bytebase-v1-InstanceResource"></a>

### InstanceResource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  |  |
| engine | [Engine](#bytebase-v1-Engine) |  |  |
| engine_version | [string](#string) |  |  |
| data_sources | [DataSource](#bytebase-v1-DataSource) | repeated |  |
| activation | [bool](#bool) |  |  |
| name | [string](#string) |  | The name of the instance. Format: instances/{instance} |
| environment | [string](#string) |  | The environment resource. Format: environments/prod where prod is the environment resource ID. |






<a name="bytebase-v1-KerberosConfig"></a>

### KerberosConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| primary | [string](#string) |  |  |
| instance | [string](#string) |  |  |
| realm | [string](#string) |  |  |
| keytab | [bytes](#bytes) |  |  |
| kdc_host | [string](#string) |  |  |
| kdc_port | [string](#string) |  |  |
| kdc_transport_protocol | [string](#string) |  |  |






<a name="bytebase-v1-ListInstancesRequest"></a>

### ListInstancesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent parameter&#39;s value depends on the target resource for the request. - instances.list(): An empty string. This method doesn&#39;t require a resource; it simply returns all instances the user has access to. - projects.instances.list(): projects/{PROJECT_ID}. This method lists all instances that have databases in the project. |
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
| data_source | [DataSource](#bytebase-v1-DataSource) |  | Identified by data source ID. Only READ_ONLY data source can be removed. |






<a name="bytebase-v1-SASLConfig"></a>

### SASLConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| krb_config | [KerberosConfig](#bytebase-v1-KerberosConfig) |  |  |






<a name="bytebase-v1-SearchInstancesRequest"></a>

### SearchInstancesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent parameter&#39;s value depends on the target resource for the request. - instances.list(): An empty string. This method doesn&#39;t require a resource; it simply returns all instances the user has access to. - projects.instances.list(): projects/{PROJECT_ID}. This method lists all instances that have databases in the project. |
| show_deleted | [bool](#bool) |  | Show deleted instances if specified. |






<a name="bytebase-v1-SearchInstancesResponse"></a>

### SearchInstancesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instances | [Instance](#bytebase-v1-Instance) | repeated | The instances from the specified request. |






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
| data_source | [DataSource](#bytebase-v1-DataSource) |  | Identified by data source ID. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| validate_only | [bool](#bool) |  | Validate only also tests the data source connection. |






<a name="bytebase-v1-UpdateInstanceRequest"></a>

### UpdateInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance | [Instance](#bytebase-v1-Instance) |  | The instance to update.

The instance&#39;s `name` field is used to identify the instance to update. Format: instances/{instance} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 


<a name="bytebase-v1-DataSource-AuthenticationType"></a>

### DataSource.AuthenticationType


| Name | Number | Description |
| ---- | ------ | ----------- |
| AUTHENTICATION_UNSPECIFIED | 0 |  |
| PASSWORD | 1 |  |
| GOOGLE_CLOUD_SQL_IAM | 2 |  |
| AWS_RDS_IAM | 3 |  |



<a name="bytebase-v1-DataSource-RedisType"></a>

### DataSource.RedisType


| Name | Number | Description |
| ---- | ------ | ----------- |
| REDIS_TYPE_UNSPECIFIED | 0 |  |
| STANDALONE | 1 |  |
| SENTINEL | 2 |  |
| CLUSTER | 3 |  |



<a name="bytebase-v1-DataSourceExternalSecret-AppRoleAuthOption-SecretType"></a>

### DataSourceExternalSecret.AppRoleAuthOption.SecretType


| Name | Number | Description |
| ---- | ------ | ----------- |
| SECRET_TYPE_UNSPECIFIED | 0 |  |
| PLAIN | 1 |  |
| ENVIRONMENT | 2 |  |



<a name="bytebase-v1-DataSourceExternalSecret-AuthType"></a>

### DataSourceExternalSecret.AuthType


| Name | Number | Description |
| ---- | ------ | ----------- |
| AUTH_TYPE_UNSPECIFIED | 0 |  |
| TOKEN | 1 | ref: https://developer.hashicorp.com/vault/docs/auth/token |
| VAULT_APP_ROLE | 2 | ref: https://developer.hashicorp.com/vault/docs/auth/approle |



<a name="bytebase-v1-DataSourceExternalSecret-SecretType"></a>

### DataSourceExternalSecret.SecretType


| Name | Number | Description |
| ---- | ------ | ----------- |
| SAECRET_TYPE_UNSPECIFIED | 0 |  |
| VAULT_KV_V2 | 1 | ref: https://developer.hashicorp.com/vault/api-docs/secret/kv/kv-v2 |
| AWS_SECRETS_MANAGER | 2 | ref: https://docs.aws.amazon.com/secretsmanager/latest/userguide/intro.html |
| GCP_SECRET_MANAGER | 3 | ref: https://cloud.google.com/secret-manager/docs |



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
| SearchInstances | [SearchInstancesRequest](#bytebase-v1-SearchInstancesRequest) | [SearchInstancesResponse](#bytebase-v1-SearchInstancesResponse) | Deprecated. |
| CreateInstance | [CreateInstanceRequest](#bytebase-v1-CreateInstanceRequest) | [Instance](#bytebase-v1-Instance) |  |
| UpdateInstance | [UpdateInstanceRequest](#bytebase-v1-UpdateInstanceRequest) | [Instance](#bytebase-v1-Instance) |  |
| DeleteInstance | [DeleteInstanceRequest](#bytebase-v1-DeleteInstanceRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| UndeleteInstance | [UndeleteInstanceRequest](#bytebase-v1-UndeleteInstanceRequest) | [Instance](#bytebase-v1-Instance) |  |
| SyncInstance | [SyncInstanceRequest](#bytebase-v1-SyncInstanceRequest) | [SyncInstanceResponse](#bytebase-v1-SyncInstanceResponse) |  |
| BatchSyncInstance | [BatchSyncInstanceRequest](#bytebase-v1-BatchSyncInstanceRequest) | [BatchSyncInstanceResponse](#bytebase-v1-BatchSyncInstanceResponse) |  |
| AddDataSource | [AddDataSourceRequest](#bytebase-v1-AddDataSourceRequest) | [Instance](#bytebase-v1-Instance) |  |
| RemoveDataSource | [RemoveDataSourceRequest](#bytebase-v1-RemoveDataSourceRequest) | [Instance](#bytebase-v1-Instance) |  |
| UpdateDataSource | [UpdateDataSourceRequest](#bytebase-v1-UpdateDataSourceRequest) | [Instance](#bytebase-v1-Instance) |  |
| SyncSlowQueries | [SyncSlowQueriesRequest](#bytebase-v1-SyncSlowQueriesRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



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
| statement | [string](#string) |  | The statement is used for preview purpose. |
| statement_size | [int64](#int64) |  |  |
| statement_sheet | [string](#string) |  | The name of the sheet resource. Format: projects/{project}/sheets/{sheet} |
| schema | [string](#string) |  |  |
| schema_size | [int64](#int64) |  |  |
| prev_schema | [string](#string) |  |  |
| prev_schema_size | [int64](#int64) |  |  |
| execution_duration | [google.protobuf.Duration](#google-protobuf-Duration) |  |  |
| issue | [string](#string) |  | Format: projects/{project}/issues/{issue} |
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






<a name="bytebase-v1-CheckConstraintMetadata"></a>

### CheckConstraintMetadata
CheckConstraintMetadata is the metadata for check constraints.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a check constraint. |
| expression | [string](#string) |  | The expression is the expression of a check constraint. |






<a name="bytebase-v1-ColumnConfig"></a>

### ColumnConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a column. |
| semantic_type_id | [string](#string) |  |  |
| labels | [ColumnConfig.LabelsEntry](#bytebase-v1-ColumnConfig-LabelsEntry) | repeated | The user labels for a column. |
| classification_id | [string](#string) |  |  |






<a name="bytebase-v1-ColumnConfig-LabelsEntry"></a>

### ColumnConfig.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-v1-ColumnMetadata"></a>

### ColumnMetadata
ColumnMetadata is the metadata for columns.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a column. |
| position | [int32](#int32) |  | The position is the position in columns. |
| has_default | [bool](#bool) |  |  |
| default_null | [bool](#bool) |  |  |
| default_string | [string](#string) |  |  |
| default_expression | [string](#string) |  |  |
| on_update | [string](#string) |  | The on_update is the on update action of a column. For MySQL like databases, it&#39;s only supported for TIMESTAMP columns with CURRENT_TIMESTAMP as on update value. |
| nullable | [bool](#bool) |  | The nullable is the nullable of a column. |
| type | [string](#string) |  | The type is the type of a column. |
| character_set | [string](#string) |  | The character_set is the character_set of a column. |
| collation | [string](#string) |  | The collation is the collation of a column. |
| comment | [string](#string) |  | The comment is the comment of a column. classification and user_comment is parsed from the comment. |
| user_comment | [string](#string) |  | The user_comment is the user comment of a column parsed from the comment. |
| effective_masking_level | [MaskingLevel](#bytebase-v1-MaskingLevel) |  | The effective_masking_level is the effective masking level of the column, evaluate from the column masking data and global masking rules. |
| generation | [GenerationMetadata](#bytebase-v1-GenerationMetadata) |  | The generation is the generation of a column. |






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
| instance_resource | [InstanceResource](#bytebase-v1-InstanceResource) |  | The instance resource. |






<a name="bytebase-v1-Database-LabelsEntry"></a>

### Database.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-v1-DatabaseConfig"></a>

### DatabaseConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| schema_configs | [SchemaConfig](#bytebase-v1-SchemaConfig) | repeated | The schema_configs is the list of configs for schemas in a database. |






<a name="bytebase-v1-DatabaseMetadata"></a>

### DatabaseMetadata
DatabaseMetadata is the metadata for databases.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The database metadata name.

Format: instances/{instance}/databases/{database}/metadata |
| schemas | [SchemaMetadata](#bytebase-v1-SchemaMetadata) | repeated | The schemas is the list of schemas in a database. |
| character_set | [string](#string) |  | The character_set is the character set of a database. |
| collation | [string](#string) |  | The collation is the collation of a database. |
| extensions | [ExtensionMetadata](#bytebase-v1-ExtensionMetadata) | repeated | The extensions is the list of extensions in a database. |
| schema_configs | [SchemaConfig](#bytebase-v1-SchemaConfig) | repeated | The schema_configs is the list of configs for schemas in a database. |






<a name="bytebase-v1-DatabaseSchema"></a>

### DatabaseSchema
DatabaseSchema is the metadata for databases.


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






<a name="bytebase-v1-DiffSchemaRequest"></a>

### DiffSchemaRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database or change history. Format: database: instances/{instance}/databases/{database} change history: instances/{instance}/databases/{database}/changeHistories/{changeHistory} |
| schema | [string](#string) |  | The target schema. |
| change_history | [string](#string) |  | The resource name of the change history Format: instances/{instance}/databases/{database}/changeHistories/{changeHistory} |
| sdl_format | [bool](#bool) |  | Format the schema dump into SDL format. |






<a name="bytebase-v1-DiffSchemaResponse"></a>

### DiffSchemaResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| diff | [string](#string) |  |  |






<a name="bytebase-v1-ExtensionMetadata"></a>

### ExtensionMetadata
ExtensionMetadata is the metadata for extensions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of an extension. |
| schema | [string](#string) |  | The schema is the extension that is installed to. But the extension usage is not limited to the schema. |
| version | [string](#string) |  | The version is the version of an extension. |
| description | [string](#string) |  | The description is the description of an extension. |






<a name="bytebase-v1-ExternalTableMetadata"></a>

### ExternalTableMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a external table. |
| external_server_name | [string](#string) |  | The external_server_name is the name of the external server. |
| external_database_name | [string](#string) |  | The external_database_name is the name of the external database. |
| columns | [ColumnMetadata](#bytebase-v1-ColumnMetadata) | repeated | The columns is the ordered list of columns in a foreign table. |






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






<a name="bytebase-v1-FunctionConfig"></a>

### FunctionConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a function. |
| updater | [string](#string) |  | The last updater of the function in branch. Format: users/{email} |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the function is updated in branch. |






<a name="bytebase-v1-FunctionMetadata"></a>

### FunctionMetadata
FunctionMetadata is the metadata for functions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a function. |
| definition | [string](#string) |  | The definition is the definition of a function. |






<a name="bytebase-v1-GenerationMetadata"></a>

### GenerationMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [GenerationMetadata.Type](#bytebase-v1-GenerationMetadata-Type) |  |  |
| expression | [string](#string) |  |  |






<a name="bytebase-v1-GetChangeHistoryRequest"></a>

### GetChangeHistoryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the change history to retrieve. Format: instances/{instance}/databases/{database}/changeHistories/{changeHistory} |
| view | [ChangeHistoryView](#bytebase-v1-ChangeHistoryView) |  |  |
| sdl_format | [bool](#bool) |  | Format the schema dump into SDL format. |
| concise | [bool](#bool) |  | When true, the schema dump will be concise. |






<a name="bytebase-v1-GetDatabaseMetadataRequest"></a>

### GetDatabaseMetadataRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to retrieve metadata. Format: instances/{instance}/databases/{database}/metadata |
| view | [DatabaseMetadataView](#bytebase-v1-DatabaseMetadataView) |  | The view to return. Defaults to DATABASE_METADATA_VIEW_BASIC. |
| filter | [string](#string) |  | The filter used for a specific schema object such as &#34;schemas/schema-a/tables/table-a&#34;. The column masking level will only be returned when a table filter is used. |






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
| concise | [bool](#bool) |  | When true, the schema dump will be concise. For Oracle, there will be tables and indexes only for Sync Schema. |






<a name="bytebase-v1-IndexMetadata"></a>

### IndexMetadata
IndexMetadata is the metadata for indexes.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of an index. |
| expressions | [string](#string) | repeated | The expressions are the ordered columns or expressions of an index. This could refer to a column or an expression. |
| key_length | [int64](#int64) | repeated | The key_lengths are the ordered key lengths of an index. If the key length is not specified, it&#39;s -1. |
| descending | [bool](#bool) | repeated | The descending is the ordered descending of an index. |
| type | [string](#string) |  | The type is the type of an index. |
| unique | [bool](#bool) |  | The unique is whether the index is unique. |
| primary | [bool](#bool) |  | The primary is whether the index is a primary key index. |
| visible | [bool](#bool) |  | The visible is whether the index is visible. |
| comment | [string](#string) |  | The comment is the comment of an index. |
| definition | [string](#string) |  | The definition of an index. |






<a name="bytebase-v1-ListChangeHistoriesRequest"></a>

### ListChangeHistoriesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent of the change histories. Format: instances/{instance}/databases/{database} |
| page_size | [int32](#int32) |  | The maximum number of change histories to return. The service may return fewer than this value. If unspecified, at most 10 change histories will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListChangeHistories` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListChangeHistories` must match the call that provided the page token. |
| view | [ChangeHistoryView](#bytebase-v1-ChangeHistoryView) |  |  |
| filter | [string](#string) |  | The filter of the change histories. follow the [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) syntax. Support filter by type, source or table. For example: table = &#34;tableExists(&#39;{database}&#39;, &#39;{schema}&#39;, &#39;{table}&#39;)&#34; table = &#34;tableExists(&#39;db&#39;, &#39;public&#39;, &#39;table1&#39;) || tableExists(&#39;db&#39;, &#39;public&#39;, &#39;table2&#39;)&#34; type = &#34;MIGRATE | DATA&#34; source = &#34;UI&#34; source = &#34;VCS&#34;

The table filter follow the CEL syntax. currently, we have one function for CEL: - tableExists(database, schema, table): return true if the table exists in changed resources.

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
| parent | [string](#string) |  | The parent, which owns this collection of databases. - projects/{project}: list all databases in a project. - workspaces/{workspace}: list all databases in a workspace. |
| page_size | [int32](#int32) |  | The maximum number of databases to return. The service may return fewer than this value. If unspecified, at most 50 databases will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListDatabases` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListDatabases` must match the call that provided the page token. |






<a name="bytebase-v1-ListDatabasesResponse"></a>

### ListDatabasesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [Database](#bytebase-v1-Database) | repeated | The databases from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-ListInstanceDatabasesRequest"></a>

### ListInstanceDatabasesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of databases. - instances/{instance}: list all databases for an instance. Use &#34;instances/-&#34; to list all databases. |
| page_size | [int32](#int32) |  | The maximum number of databases to return. The service may return fewer than this value. If unspecified, at most 50 databases will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListDatabases` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListDatabases` must match the call that provided the page token. |
| filter | [string](#string) |  | Deprecated. Filter is used to filter databases returned in the list. For example, `project == &#34;projects/{project}&#34;` can be used to list databases in a project. Note: the project filter will be moved to parent. |






<a name="bytebase-v1-ListInstanceDatabasesResponse"></a>

### ListInstanceDatabasesResponse



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






<a name="bytebase-v1-MaterializedViewMetadata"></a>

### MaterializedViewMetadata
MaterializedViewMetadata is the metadata for materialized views.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a materialized view. |
| definition | [string](#string) |  | The definition is the definition of a materialized view. |
| comment | [string](#string) |  | The comment is the comment of a materialized view. |
| dependent_columns | [DependentColumn](#bytebase-v1-DependentColumn) | repeated | The dependent_columns is the list of dependent columns of a materialized view. |






<a name="bytebase-v1-ProcedureConfig"></a>

### ProcedureConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a procedure. |
| updater | [string](#string) |  | The last updater of the procedure in branch. Format: users/{email} |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the procedure is updated in branch. |






<a name="bytebase-v1-ProcedureMetadata"></a>

### ProcedureMetadata
ProcedureMetadata is the metadata for procedures.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a procedure. |
| definition | [string](#string) |  | The definition is the definition of a procedure. |






<a name="bytebase-v1-SchemaConfig"></a>

### SchemaConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the schema name. It is an empty string for databases without such concept such as MySQL. |
| table_configs | [TableConfig](#bytebase-v1-TableConfig) | repeated | The table_configs is the list of configs for tables in a schema. |
| function_configs | [FunctionConfig](#bytebase-v1-FunctionConfig) | repeated |  |
| procedure_configs | [ProcedureConfig](#bytebase-v1-ProcedureConfig) | repeated |  |
| view_configs | [ViewConfig](#bytebase-v1-ViewConfig) | repeated |  |






<a name="bytebase-v1-SchemaMetadata"></a>

### SchemaMetadata
SchemaMetadata is the metadata for schemas.
This is the concept of schema in Postgres, but it&#39;s a no-op for MySQL.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the schema name. It is an empty string for databases without such concept such as MySQL. |
| tables | [TableMetadata](#bytebase-v1-TableMetadata) | repeated | The tables is the list of tables in a schema. |
| external_tables | [ExternalTableMetadata](#bytebase-v1-ExternalTableMetadata) | repeated | The external_tables is the list of external tables in a schema. |
| views | [ViewMetadata](#bytebase-v1-ViewMetadata) | repeated | The views is the list of views in a schema. |
| functions | [FunctionMetadata](#bytebase-v1-FunctionMetadata) | repeated | The functions is the list of functions in a schema. |
| procedures | [ProcedureMetadata](#bytebase-v1-ProcedureMetadata) | repeated | The procedures is the list of procedures in a schema. |
| streams | [StreamMetadata](#bytebase-v1-StreamMetadata) | repeated | The streams is the list of streams in a schema, currently, only used for Snowflake. |
| tasks | [TaskMetadata](#bytebase-v1-TaskMetadata) | repeated | The routines is the list of routines in a schema, currently, only used for Snowflake. |
| materialized_views | [MaterializedViewMetadata](#bytebase-v1-MaterializedViewMetadata) | repeated | The materialized_views is the list of materialized views in a schema. |






<a name="bytebase-v1-SearchDatabasesRequest"></a>

### SearchDatabasesRequest
Deprecated.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of databases to return. The service may return fewer than this value. If unspecified, at most 50 databases will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListDatabases` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListDatabases` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter databases returned in the list. follow the [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) syntax. The field only support in filter: - project with &#34;=&#34; operator, for example: - project = &#34;projects/sample-project&#34; - project = &#34;projects/-&#34; - instance with &#34;=&#34; operator, for example: - instance = &#34;instances/mysql&#34; - instance = &#34;instances/-&#34; for example, we can use project = &#34;projects/sample&#34; &amp;&amp; instance = &#34;instances/-&#34; to list all databases in the sample project. |






<a name="bytebase-v1-SearchDatabasesResponse"></a>

### SearchDatabasesResponse
Deprecated.


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
| rows_sent | [int32](#int32) |  | The rows sent of the slow query log. |
| rows_examined | [int32](#int32) |  | The rows examined of the slow query log. |
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
| count | [int32](#int32) |  | The count of the slow query log. |
| latest_log_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The latest log time of the slow query log. |
| average_query_time | [google.protobuf.Duration](#google-protobuf-Duration) |  | The average query time of the slow query log. |
| maximum_query_time | [google.protobuf.Duration](#google-protobuf-Duration) |  | The maximum query time of the slow query log. |
| average_rows_sent | [int32](#int32) |  | The average rows sent of the slow query log. |
| maximum_rows_sent | [int32](#int32) |  | The maximum rows sent of the slow query log. |
| average_rows_examined | [int32](#int32) |  | The average rows examined of the slow query log. |
| maximum_rows_examined | [int32](#int32) |  | The maximum rows examined of the slow query log. |
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







<a name="bytebase-v1-TableConfig"></a>

### TableConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a table. |
| column_configs | [ColumnConfig](#bytebase-v1-ColumnConfig) | repeated | The column_configs is the ordered list of configs for columns in a table. |
| classification_id | [string](#string) |  |  |
| updater | [string](#string) |  | The last updater of the table in branch. Format: users/{email} |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the table is updated in branch. |






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
| charset | [string](#string) |  | The character set of table. |
| row_count | [int64](#int64) |  | The row_count is the estimated number of rows of a table. |
| data_size | [int64](#int64) |  | The data_size is the estimated data size of a table. |
| index_size | [int64](#int64) |  | The index_size is the estimated index size of a table. |
| data_free | [int64](#int64) |  | The data_free is the estimated free data size of a table. |
| create_options | [string](#string) |  | The create_options is the create option of a table. |
| comment | [string](#string) |  | The comment is the comment of a table. classification and user_comment is parsed from the comment. |
| user_comment | [string](#string) |  | The user_comment is the user comment of a table parsed from the comment. |
| foreign_keys | [ForeignKeyMetadata](#bytebase-v1-ForeignKeyMetadata) | repeated | The foreign_keys is the list of foreign keys in a table. |
| partitions | [TablePartitionMetadata](#bytebase-v1-TablePartitionMetadata) | repeated | The partitions is the list of partitions in a table. |
| check_constraints | [CheckConstraintMetadata](#bytebase-v1-CheckConstraintMetadata) | repeated | The check_constraints is the list of check constraints in a table. |






<a name="bytebase-v1-TablePartitionMetadata"></a>

### TablePartitionMetadata
TablePartitionMetadata is the metadata for table partitions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a table partition. |
| type | [TablePartitionMetadata.Type](#bytebase-v1-TablePartitionMetadata-Type) |  | The type of a table partition. |
| expression | [string](#string) |  | The expression is the expression of a table partition. For PostgreSQL, the expression is the text of {FOR VALUES partition_bound_spec}, see https://www.postgresql.org/docs/current/sql-createtable.html. For MySQL, the expression is the `expr` or `column_list` of the following syntax. PARTITION BY { [LINEAR] HASH(expr) | [LINEAR] KEY [ALGORITHM={1 | 2}] (column_list) | RANGE{(expr) | COLUMNS(column_list)} | LIST{(expr) | COLUMNS(column_list)} }. |
| value | [string](#string) |  | The value is the value of a table partition. For MySQL, the value is for RANGE and LIST partition types, - For a RANGE partition, it contains the value set in the partition&#39;s VALUES LESS THAN clause, which can be either an integer or MAXVALUE. - For a LIST partition, this column contains the values defined in the partition&#39;s VALUES IN clause, which is a list of comma-separated integer values. - For others, it&#39;s an empty string. |
| use_default | [string](#string) |  | The use_default is whether the users use the default partition, it stores the different value for different database engines. For MySQL, it&#39;s [INT] type, 0 means not use default partition, otherwise, it&#39;s equals to number in syntax [SUB]PARTITION {number}. |
| subpartitions | [TablePartitionMetadata](#bytebase-v1-TablePartitionMetadata) | repeated | The subpartitions is the list of subpartitions in a table partition. |






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






<a name="bytebase-v1-UpdateDatabaseMetadataRequest"></a>

### UpdateDatabaseMetadataRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database_metadata | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  | The database metadata to update.

The database_metadata&#39;s `name` field is used to identify the database metadata to update. Format: instances/{instance}/databases/{database}/metadata |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






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






<a name="bytebase-v1-ViewConfig"></a>

### ViewConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a view. |
| updater | [string](#string) |  | The last updater of the view in branch. Format: users/{email} |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the view is updated in branch. |






<a name="bytebase-v1-ViewMetadata"></a>

### ViewMetadata
ViewMetadata is the metadata for views.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a view. |
| definition | [string](#string) |  | The definition is the definition of a view. |
| comment | [string](#string) |  | The comment is the comment of a view. |
| dependent_columns | [DependentColumn](#bytebase-v1-DependentColumn) | repeated | The dependent_columns is the list of dependent columns of a view. |





 


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



<a name="bytebase-v1-DatabaseMetadataView"></a>

### DatabaseMetadataView


| Name | Number | Description |
| ---- | ------ | ----------- |
| DATABASE_METADATA_VIEW_UNSPECIFIED | 0 | The default and unset value. The API will default to the BASIC view. |
| DATABASE_METADATA_VIEW_BASIC | 1 | Include basic information of schema object names such as schema, table, view, function names. |
| DATABASE_METADATA_VIEW_FULL | 2 | Include everything such as columns and column masking level. |



<a name="bytebase-v1-GenerationMetadata-Type"></a>

### GenerationMetadata.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| TYPE_VIRTUAL | 1 |  |
| TYPE_STORED | 2 |  |



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



<a name="bytebase-v1-TablePartitionMetadata-Type"></a>

### TablePartitionMetadata.Type
Type is the type of a table partition, some database engines may not support all types.
Only avilable for the following database engines now:
MySQL: RANGE, RANGE COLUMNS, LIST, LIST COLUMNS, HASH, LINEAR HASH, KEY, LINEAR_KEY (https://dev.mysql.com/doc/refman/8.0/en/partitioning-types.html)
TiDB: RANGE, RANGE COLUMNS, LIST, LIST COLUMNS, HASH, KEY
PostgreSQL: RANGE, LIST, HASH (https://www.postgresql.org/docs/current/ddl-partitioning.html)

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| RANGE | 1 |  |
| RANGE_COLUMNS | 2 |  |
| LIST | 3 |  |
| LIST_COLUMNS | 4 |  |
| HASH | 5 |  |
| LINEAR_HASH | 6 |  |
| KEY | 7 |  |
| LINEAR_KEY | 8 |  |



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
| ListInstanceDatabases | [ListInstanceDatabasesRequest](#bytebase-v1-ListInstanceDatabasesRequest) | [ListInstanceDatabasesResponse](#bytebase-v1-ListInstanceDatabasesResponse) |  |
| ListDatabases | [ListDatabasesRequest](#bytebase-v1-ListDatabasesRequest) | [ListDatabasesResponse](#bytebase-v1-ListDatabasesResponse) |  |
| SearchDatabases | [SearchDatabasesRequest](#bytebase-v1-SearchDatabasesRequest) | [SearchDatabasesResponse](#bytebase-v1-SearchDatabasesResponse) | Deprecated. This will be removed in the next release. Search for databases that the caller has the bb.databases.get permission on, and also satisfy the specified query. |
| UpdateDatabase | [UpdateDatabaseRequest](#bytebase-v1-UpdateDatabaseRequest) | [Database](#bytebase-v1-Database) |  |
| BatchUpdateDatabases | [BatchUpdateDatabasesRequest](#bytebase-v1-BatchUpdateDatabasesRequest) | [BatchUpdateDatabasesResponse](#bytebase-v1-BatchUpdateDatabasesResponse) |  |
| SyncDatabase | [SyncDatabaseRequest](#bytebase-v1-SyncDatabaseRequest) | [SyncDatabaseResponse](#bytebase-v1-SyncDatabaseResponse) |  |
| GetDatabaseMetadata | [GetDatabaseMetadataRequest](#bytebase-v1-GetDatabaseMetadataRequest) | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  |
| UpdateDatabaseMetadata | [UpdateDatabaseMetadataRequest](#bytebase-v1-UpdateDatabaseMetadataRequest) | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  |
| GetDatabaseSchema | [GetDatabaseSchemaRequest](#bytebase-v1-GetDatabaseSchemaRequest) | [DatabaseSchema](#bytebase-v1-DatabaseSchema) |  |
| DiffSchema | [DiffSchemaRequest](#bytebase-v1-DiffSchemaRequest) | [DiffSchemaResponse](#bytebase-v1-DiffSchemaResponse) |  |
| ListSlowQueries | [ListSlowQueriesRequest](#bytebase-v1-ListSlowQueriesRequest) | [ListSlowQueriesResponse](#bytebase-v1-ListSlowQueriesResponse) |  |
| ListSecrets | [ListSecretsRequest](#bytebase-v1-ListSecretsRequest) | [ListSecretsResponse](#bytebase-v1-ListSecretsResponse) |  |
| UpdateSecret | [UpdateSecretRequest](#bytebase-v1-UpdateSecretRequest) | [Secret](#bytebase-v1-Secret) |  |
| DeleteSecret | [DeleteSecretRequest](#bytebase-v1-DeleteSecretRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| AdviseIndex | [AdviseIndexRequest](#bytebase-v1-AdviseIndexRequest) | [AdviseIndexResponse](#bytebase-v1-AdviseIndexResponse) |  |
| ListChangeHistories | [ListChangeHistoriesRequest](#bytebase-v1-ListChangeHistoriesRequest) | [ListChangeHistoriesResponse](#bytebase-v1-ListChangeHistoriesResponse) |  |
| GetChangeHistory | [GetChangeHistoryRequest](#bytebase-v1-GetChangeHistoryRequest) | [ChangeHistory](#bytebase-v1-ChangeHistory) |  |

 



<a name="v1_branch_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/branch_service.proto



<a name="bytebase-v1-Branch"></a>

### Branch



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the branch. Format: projects/{project}/branches/{branch} {branch} should be the id of a sheet. |
| branch_id | [string](#string) |  | The branch ID. |
| schema | [string](#string) |  | The schema of branch. AKA sheet&#39;s statement. |
| schema_metadata | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  | The metadata of the current editing schema. |
| baseline_schema | [string](#string) |  | The baseline schema. |
| baseline_schema_metadata | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  | The metadata of the baseline schema. |
| engine | [Engine](#bytebase-v1-Engine) |  | The database engine of the branch. |
| baseline_database | [string](#string) |  | The name of the baseline database. Format: instances/{instance}/databases/{database} |
| parent_branch | [string](#string) |  | The name of the parent branch. For main branch, it&#39;s empty. For child branch, its format will be: projects/{project}/branches/{branch} |
| etag | [string](#string) |  | The etag of the branch. |
| creator | [string](#string) |  | The creator of the branch. Format: users/{email} |
| updater | [string](#string) |  | The updater of the branch. Format: users/{email} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the branch was created. |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the branch was last updated. |






<a name="bytebase-v1-CreateBranchRequest"></a>

### CreateBranchRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of branches. Format: project/{project} |
| branch | [Branch](#bytebase-v1-Branch) |  |  |
| branch_id | [string](#string) |  | The ID to use for the branch, which will become the final component of the branch&#39;s resource name. Format: [a-zA-Z][a-zA-Z0-9-_/]&#43;. |






<a name="bytebase-v1-DeleteBranchRequest"></a>

### DeleteBranchRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the branch to delete. Format: projects/{project}/branches/{branch} |
| force | [bool](#bool) |  | By default, server will return `FAILED_PRECONDITION` error if delete the branch that is parent of other branches. If true, server will delete the branch forcely but will not delete its children branches. |






<a name="bytebase-v1-DiffDatabaseRequest"></a>

### DiffDatabaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of branch. |
| database | [string](#string) |  | The name of the databsae to merge the branch to. |






<a name="bytebase-v1-DiffDatabaseResponse"></a>

### DiffDatabaseResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| diff | [string](#string) |  | The schema diff when merge occurs seamlessly. |
| schema | [string](#string) |  | The merged schema if there is no conflict. |
| conflict_schema | [string](#string) |  | The conflict schema when rebase has conflicts. The conflict section is enclosed by the following. &lt;&lt;&lt;&lt;&lt; HEAD ==== &gt;&gt;&gt;&gt;&gt; main |






<a name="bytebase-v1-DiffMetadataRequest"></a>

### DiffMetadataRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| source_metadata | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  | The metadata of the source schema. |
| target_metadata | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  | The metadata of the target schema. |
| engine | [Engine](#bytebase-v1-Engine) |  | The database engine of the schema. |
| classification_from_config | [bool](#bool) |  | If false, we will build the raw common by classification in database config. |






<a name="bytebase-v1-DiffMetadataResponse"></a>

### DiffMetadataResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| diff | [string](#string) |  | The diff of the metadata. |






<a name="bytebase-v1-GetBranchRequest"></a>

### GetBranchRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the branch to retrieve. Format: projects/{project}/branches/{branch} |






<a name="bytebase-v1-ListBranchesRequest"></a>

### ListBranchesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource of the branch. Format: projects/{project} |
| filter | [string](#string) |  | To filter the search result. |
| page_size | [int32](#int32) |  | The maximum number of branches to return. The service may return fewer than this value. If unspecified, at most 50 branches will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListBranches` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListBranches` must match the call that provided the page token. |
| view | [BranchView](#bytebase-v1-BranchView) |  |  |






<a name="bytebase-v1-ListBranchesResponse"></a>

### ListBranchesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| branches | [Branch](#bytebase-v1-Branch) | repeated | The branches from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-MergeBranchRequest"></a>

### MergeBranchRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the base branch to merge to. Format: projects/{project}/branches/{branch} |
| head_branch | [string](#string) |  | The head branch to merge from. Format: projects/{project}/branches/{branch} |
| etag | [string](#string) |  | The current etag of the branch. If an etag is provided and does not match the current etag of the branch, the call will be blocked and an ABORTED error will be returned. The etag should be the etag from named branch. |
| validate_only | [bool](#bool) |  | validate_only determines if the merge can occur seamlessly without any conflicts. |






<a name="bytebase-v1-RebaseBranchRequest"></a>

### RebaseBranchRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the base branch to merge to. Format: projects/{project}/branches/{branch} |
| source_database | [string](#string) |  | The database (remote upstream) used to rebase. We use its schema as baseline and reapply the difference between base and head of the named branch. Format: instances/{instance}/databases/{database} |
| source_branch | [string](#string) |  | The branch (remote upstream) used to rebase. We use its head as baseline. We use its head schema as baseline and reapply the difference between base and head of the named branch. Format: projects/{project}/branches/{branch} |
| merged_schema | [string](#string) |  | For failed merge, we will pass in this addition merged schema and use it for head. This has to be set together with source_database or source_branch. |
| etag | [string](#string) |  | The current etag of the branch. If an etag is provided and does not match the current etag of the branch, the call will be blocked and an ABORTED error will be returned. The etag should be specified for using merged_schema. The etag should be the etag from named branch. |
| validate_only | [bool](#bool) |  | validate_only determines if the rebase can occur seamlessly without any conflicts. |






<a name="bytebase-v1-RebaseBranchResponse"></a>

### RebaseBranchResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| branch | [Branch](#bytebase-v1-Branch) |  | The rebased branch when rebase occurs seamlessly. |
| conflict_schema | [string](#string) |  | The conflict schema when rebase has conflicts. The conflict section is enclosed by the following. &lt;&lt;&lt;&lt;&lt; HEAD ==== &gt;&gt;&gt;&gt;&gt; main |






<a name="bytebase-v1-UpdateBranchRequest"></a>

### UpdateBranchRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| branch | [Branch](#bytebase-v1-Branch) |  | The branch to update.

The branch&#39;s `name` field is used to identify the branch to update. Format: projects/{project}/branches/{branch} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| etag | [string](#string) |  | The current etag of the branch. If an etag is provided and does not match the current etag of the branch, the call will be blocked and an ABORTED error will be returned. The etag should be specified for using merged_schema. The etag should be the etag from named branch. |





 


<a name="bytebase-v1-BranchView"></a>

### BranchView


| Name | Number | Description |
| ---- | ------ | ----------- |
| BRANCH_VIEW_UNSPECIFIED | 0 | The default / unset value. The API will default to the BASIC view. |
| BRANCH_VIEW_BASIC | 1 | Exclude schema, baseline_schema. |
| BRANCH_VIEW_FULL | 2 | Include everything. |


 

 


<a name="bytebase-v1-BranchService"></a>

### BranchService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetBranch | [GetBranchRequest](#bytebase-v1-GetBranchRequest) | [Branch](#bytebase-v1-Branch) |  |
| ListBranches | [ListBranchesRequest](#bytebase-v1-ListBranchesRequest) | [ListBranchesResponse](#bytebase-v1-ListBranchesResponse) |  |
| CreateBranch | [CreateBranchRequest](#bytebase-v1-CreateBranchRequest) | [Branch](#bytebase-v1-Branch) |  |
| UpdateBranch | [UpdateBranchRequest](#bytebase-v1-UpdateBranchRequest) | [Branch](#bytebase-v1-Branch) |  |
| MergeBranch | [MergeBranchRequest](#bytebase-v1-MergeBranchRequest) | [Branch](#bytebase-v1-Branch) |  |
| RebaseBranch | [RebaseBranchRequest](#bytebase-v1-RebaseBranchRequest) | [RebaseBranchResponse](#bytebase-v1-RebaseBranchResponse) |  |
| DeleteBranch | [DeleteBranchRequest](#bytebase-v1-DeleteBranchRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| DiffDatabase | [DiffDatabaseRequest](#bytebase-v1-DiffDatabaseRequest) | [DiffDatabaseResponse](#bytebase-v1-DiffDatabaseResponse) | DiffDatabase works similar to branch rebase. 1) set the base as the schema of a database; 2) apply the changes between base and head of branch to the new base (schema of database); 3) return the diff DDLs similar to DiffSchema in database service. 4) return the conflict schema if conflict needs to be resolved by user. Once resolved, user will call DiffSchema() in database service to get diff DDLs. |
| DiffMetadata | [DiffMetadataRequest](#bytebase-v1-DiffMetadataRequest) | [DiffMetadataResponse](#bytebase-v1-DiffMetadataResponse) |  |

 



<a name="v1_cel_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/cel_service.proto



<a name="bytebase-v1-BatchDeparseRequest"></a>

### BatchDeparseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expressions | [google.api.expr.v1alpha1.ParsedExpr](#google-api-expr-v1alpha1-ParsedExpr) | repeated |  |






<a name="bytebase-v1-BatchDeparseResponse"></a>

### BatchDeparseResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expressions | [string](#string) | repeated |  |






<a name="bytebase-v1-BatchParseRequest"></a>

### BatchParseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expressions | [string](#string) | repeated |  |






<a name="bytebase-v1-BatchParseResponse"></a>

### BatchParseResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expressions | [google.api.expr.v1alpha1.ParsedExpr](#google-api-expr-v1alpha1-ParsedExpr) | repeated |  |





 

 

 


<a name="bytebase-v1-CelService"></a>

### CelService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| BatchParse | [BatchParseRequest](#bytebase-v1-BatchParseRequest) | [BatchParseResponse](#bytebase-v1-BatchParseResponse) |  |
| BatchDeparse | [BatchDeparseRequest](#bytebase-v1-BatchDeparseRequest) | [BatchDeparseResponse](#bytebase-v1-BatchDeparseResponse) |  |

 



<a name="v1_changelist_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/changelist_service.proto



<a name="bytebase-v1-Changelist"></a>

### Changelist



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the changelist resource. Canonical parent is project. Format: projects/{project}/changelists/{changelist} |
| description | [string](#string) |  |  |
| creator | [string](#string) |  | The creator of the changelist. Format: users/{email} |
| updater | [string](#string) |  | The updater of the changelist. Format: users/{email} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The create time of the changelist. |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The last update time of the changelist. |
| changes | [Changelist.Change](#bytebase-v1-Changelist-Change) | repeated |  |






<a name="bytebase-v1-Changelist-Change"></a>

### Changelist.Change



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheet | [string](#string) |  | The name of a sheet. |
| source | [string](#string) |  | The source of origin. 1) change history: instances/{instance}/databases/{database}/changeHistories/{changeHistory}. 2) branch: projects/{project}/branches/{branch}. 3) raw SQL if empty. |
| version | [string](#string) |  | The migration version for a change. |






<a name="bytebase-v1-CreateChangelistRequest"></a>

### CreateChangelistRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this changelist will be created. Format: projects/{project} |
| changelist | [Changelist](#bytebase-v1-Changelist) |  | The changelist to create. |
| changelist_id | [string](#string) |  | The ID to use for the changelist, which will become the final component of the changelist&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |






<a name="bytebase-v1-DeleteChangelistRequest"></a>

### DeleteChangelistRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the changelist to delete. Format: projects/{project}/changelists/{changelist} |






<a name="bytebase-v1-GetChangelistRequest"></a>

### GetChangelistRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the changelist to retrieve. Format: projects/{project}/changelists/{changelist} |






<a name="bytebase-v1-ListChangelistsRequest"></a>

### ListChangelistsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of changelists. Format: projects/{project} |
| page_size | [int32](#int32) |  | The maximum number of databases to return. The service may return fewer than this value. If unspecified, at most 50 databases will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListDatabases` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListDatabases` must match the call that provided the page token. |






<a name="bytebase-v1-ListChangelistsResponse"></a>

### ListChangelistsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| changelists | [Changelist](#bytebase-v1-Changelist) | repeated | The changelists from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-UpdateChangelistRequest"></a>

### UpdateChangelistRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| changelist | [Changelist](#bytebase-v1-Changelist) |  | The changelist to update.

The changelist&#39;s `name` field is used to identify the changelist to update. Format: projects/{project}/changelists/{changelist} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to be updated. |





 

 

 


<a name="bytebase-v1-ChangelistService"></a>

### ChangelistService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateChangelist | [CreateChangelistRequest](#bytebase-v1-CreateChangelistRequest) | [Changelist](#bytebase-v1-Changelist) |  |
| GetChangelist | [GetChangelistRequest](#bytebase-v1-GetChangelistRequest) | [Changelist](#bytebase-v1-Changelist) |  |
| ListChangelists | [ListChangelistsRequest](#bytebase-v1-ListChangelistsRequest) | [ListChangelistsResponse](#bytebase-v1-ListChangelistsResponse) |  |
| UpdateChangelist | [UpdateChangelistRequest](#bytebase-v1-UpdateChangelistRequest) | [Changelist](#bytebase-v1-Changelist) |  |
| DeleteChangelist | [DeleteChangelistRequest](#bytebase-v1-DeleteChangelistRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



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

 



<a name="v1_iam_policy-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/iam_policy.proto



<a name="bytebase-v1-Binding"></a>

### Binding



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [string](#string) |  | The role that is assigned to the members. Format: roles/{role} |
| members | [string](#string) | repeated | Specifies the principals requesting access for a Bytebase resource. For users, the member should be: user:{email} For groups, the member should be: group:{email} |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this binding. If the condition evaluates to true, then this binding applies to the current request. If the condition evaluates to false, then this binding does not apply to the current request. However, a different role binding might grant the same role to one or more of the principals in this binding. |
| parsed_expr | [google.api.expr.v1alpha1.ParsedExpr](#google-api-expr-v1alpha1-ParsedExpr) |  | The parsed expression of the condition. |






<a name="bytebase-v1-GetIamPolicyRequest"></a>

### GetIamPolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource | [string](#string) |  | The name of the resource to get the IAM policy. Format: projects/{project} Format: workspaces/{workspace} |






<a name="bytebase-v1-IamPolicy"></a>

### IamPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bindings | [Binding](#bytebase-v1-Binding) | repeated | Collection of binding. A binding binds one or more project members to a single project role. |
| etag | [string](#string) |  | The current etag of the policy. If an etag is provided and does not match the current etag of the poliy, the call will be blocked and an ABORTED error will be returned. |






<a name="bytebase-v1-SetIamPolicyRequest"></a>

### SetIamPolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource | [string](#string) |  | The name of the resource to set the IAM policy. Format: projects/{project} Format: workspaces/{workspace} |
| policy | [IamPolicy](#bytebase-v1-IamPolicy) |  |  |
| etag | [string](#string) |  | The current etag of the policy. |





 

 

 

 



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






<a name="bytebase-v1-GrantRequest"></a>

### GrantRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [string](#string) |  | The requested role. Format: roles/EXPORTER. |
| user | [string](#string) |  | The user to be granted. Format: users/{email}. |
| condition | [google.type.Expr](#google-type-Expr) |  |  |
| expiration | [google.protobuf.Duration](#google-protobuf-Duration) |  |  |






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
| grant_request | [GrantRequest](#bytebase-v1-GrantRequest) |  | Used if the issue type is GRANT_REQUEST. |
| releasers | [string](#string) | repeated | The releasers of the pending stage of the issue rollout, judging from the rollout policy. If the policy is auto rollout, the releasers are the project owners and the issue creator. Format: - roles/workspaceOwner - roles/workspaceDBA - roles/projectOwner - roles/projectReleaser - users/{email} |
| risk_level | [Issue.RiskLevel](#bytebase-v1-Issue-RiskLevel) |  |  |
| task_status_count | [Issue.TaskStatusCountEntry](#bytebase-v1-Issue-TaskStatusCountEntry) | repeated | The status count of the issue. Keys are the following: - NOT_STARTED - SKIPPED - PENDING - RUNNING - DONE - FAILED - CANCELED |
| labels | [string](#string) | repeated |  |






<a name="bytebase-v1-Issue-Approver"></a>

### Issue.Approver



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [Issue.Approver.Status](#bytebase-v1-Issue-Approver-Status) |  | The new status. |
| principal | [string](#string) |  | Format: users/hello@world.com |






<a name="bytebase-v1-Issue-TaskStatusCountEntry"></a>

### Issue.TaskStatusCountEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [int32](#int32) |  |  |






<a name="bytebase-v1-IssueComment"></a>

### IssueComment



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| uid | [string](#string) |  |  |
| comment | [string](#string) |  |  |
| payload | [string](#string) |  | TODO: use struct message instead. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| name | [string](#string) |  | Format: projects/{project}/issues/{issue}/issueComments/{issueComment-uid} |
| creator | [string](#string) |  | Format: users/{email} |
| approval | [IssueComment.Approval](#bytebase-v1-IssueComment-Approval) |  |  |
| issue_update | [IssueComment.IssueUpdate](#bytebase-v1-IssueComment-IssueUpdate) |  |  |
| stage_end | [IssueComment.StageEnd](#bytebase-v1-IssueComment-StageEnd) |  |  |
| task_update | [IssueComment.TaskUpdate](#bytebase-v1-IssueComment-TaskUpdate) |  |  |
| task_prior_backup | [IssueComment.TaskPriorBackup](#bytebase-v1-IssueComment-TaskPriorBackup) |  |  |






<a name="bytebase-v1-IssueComment-Approval"></a>

### IssueComment.Approval



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [IssueComment.Approval.Status](#bytebase-v1-IssueComment-Approval-Status) |  |  |






<a name="bytebase-v1-IssueComment-IssueUpdate"></a>

### IssueComment.IssueUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| from_title | [string](#string) | optional |  |
| to_title | [string](#string) | optional |  |
| from_description | [string](#string) | optional |  |
| to_description | [string](#string) | optional |  |
| from_status | [IssueStatus](#bytebase-v1-IssueStatus) | optional |  |
| to_status | [IssueStatus](#bytebase-v1-IssueStatus) | optional |  |
| from_assignee | [string](#string) | optional | TODO(d): deprecate from_assignee and to_assignee. Format: users/{email} |
| to_assignee | [string](#string) | optional | Format: users/{email} |






<a name="bytebase-v1-IssueComment-StageEnd"></a>

### IssueComment.StageEnd



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stage | [string](#string) |  |  |






<a name="bytebase-v1-IssueComment-TaskPriorBackup"></a>

### IssueComment.TaskPriorBackup



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| task | [string](#string) |  |  |
| tables | [IssueComment.TaskPriorBackup.Table](#bytebase-v1-IssueComment-TaskPriorBackup-Table) | repeated |  |
| original_line | [int32](#int32) | optional |  |
| database | [string](#string) |  |  |






<a name="bytebase-v1-IssueComment-TaskPriorBackup-Table"></a>

### IssueComment.TaskPriorBackup.Table



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  |  |
| table | [string](#string) |  |  |






<a name="bytebase-v1-IssueComment-TaskUpdate"></a>

### IssueComment.TaskUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tasks | [string](#string) | repeated |  |
| from_sheet | [string](#string) | optional | Format: projects/{project}/sheets/{sheet} |
| to_sheet | [string](#string) | optional | Format: projects/{project}/sheets/{sheet} |
| from_earliest_allowed_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional |  |
| to_earliest_allowed_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional |  |
| to_status | [IssueComment.TaskUpdate.Status](#bytebase-v1-IssueComment-TaskUpdate-Status) | optional |  |






<a name="bytebase-v1-ListIssueCommentsRequest"></a>

### ListIssueCommentsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{projects}/issues/{issue} |
| page_size | [int32](#int32) |  | The maximum number of issues to return. The service may return fewer than this value. If unspecified, at most 50 issues will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListIssues` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListIssues` must match the call that provided the page token. |






<a name="bytebase-v1-ListIssueCommentsResponse"></a>

### ListIssueCommentsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issue_comments | [IssueComment](#bytebase-v1-IssueComment) | repeated |  |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-ListIssuesRequest"></a>

### ListIssuesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of issues. Format: projects/{project} Use &#34;projects/-&#34; to list all issues from all projects. |
| page_size | [int32](#int32) |  | The maximum number of issues to return. The service may return fewer than this value. If unspecified, at most 50 issues will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListIssues` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListIssues` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter issues returned in the list. |
| query | [string](#string) |  | Query is the query statement. |






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
| parent | [string](#string) |  | The parent, which owns this collection of issues. Format: projects/{project} Use &#34;projects/-&#34; to list all issues from all projects. |
| page_size | [int32](#int32) |  | The maximum number of issues to return. The service may return fewer than this value. If unspecified, at most 50 issues will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListIssues` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListIssues` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter issues returned in the list. |
| query | [string](#string) |  | Query is the query statement. |






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



<a name="bytebase-v1-Issue-RiskLevel"></a>

### Issue.RiskLevel


| Name | Number | Description |
| ---- | ------ | ----------- |
| RISK_LEVEL_UNSPECIFIED | 0 |  |
| LOW | 1 |  |
| MODERATE | 2 |  |
| HIGH | 3 |  |



<a name="bytebase-v1-Issue-Type"></a>

### Issue.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| DATABASE_CHANGE | 1 |  |
| GRANT_REQUEST | 2 |  |
| DATABASE_DATA_EXPORT | 3 |  |



<a name="bytebase-v1-IssueComment-Approval-Status"></a>

### IssueComment.Approval.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| APPROVED | 2 |  |
| REJECTED | 3 |  |



<a name="bytebase-v1-IssueComment-TaskUpdate-Status"></a>

### IssueComment.TaskUpdate.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| RUNNING | 2 |  |
| DONE | 3 |  |
| FAILED | 4 |  |
| SKIPPED | 5 |  |
| CANCELED | 6 |  |



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
| SearchIssues | [SearchIssuesRequest](#bytebase-v1-SearchIssuesRequest) | [SearchIssuesResponse](#bytebase-v1-SearchIssuesResponse) | Search for issues that the caller has the bb.issues.get permission on and also satisfy the specified filter &amp; query. |
| UpdateIssue | [UpdateIssueRequest](#bytebase-v1-UpdateIssueRequest) | [Issue](#bytebase-v1-Issue) |  |
| ListIssueComments | [ListIssueCommentsRequest](#bytebase-v1-ListIssueCommentsRequest) | [ListIssueCommentsResponse](#bytebase-v1-ListIssueCommentsResponse) |  |
| CreateIssueComment | [CreateIssueCommentRequest](#bytebase-v1-CreateIssueCommentRequest) | [IssueComment](#bytebase-v1-IssueComment) |  |
| UpdateIssueComment | [UpdateIssueCommentRequest](#bytebase-v1-UpdateIssueCommentRequest) | [IssueComment](#bytebase-v1-IssueComment) |  |
| BatchUpdateIssuesStatus | [BatchUpdateIssuesStatusRequest](#bytebase-v1-BatchUpdateIssuesStatusRequest) | [BatchUpdateIssuesStatusResponse](#bytebase-v1-BatchUpdateIssuesStatusResponse) |  |
| ApproveIssue | [ApproveIssueRequest](#bytebase-v1-ApproveIssueRequest) | [Issue](#bytebase-v1-Issue) |  |
| RejectIssue | [RejectIssueRequest](#bytebase-v1-RejectIssueRequest) | [Issue](#bytebase-v1-Issue) |  |
| RequestIssue | [RequestIssueRequest](#bytebase-v1-RequestIssueRequest) | [Issue](#bytebase-v1-Issue) |  |

 



<a name="v1_org_policy_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/org_policy_service.proto



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
| parent | [string](#string) |  | The parent, which owns this collection of policies. Format: {resource type}/{resource id} |
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
| masking_level | [MaskingLevel](#bytebase-v1-MaskingLevel) |  |  |
| full_masking_algorithm_id | [string](#string) |  |  |
| partial_masking_algorithm_id | [string](#string) |  |  |






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
| member | [string](#string) |  | Member is the principal who bind to this exception policy instance.

- `user:{email}`: An email address that represents a specific Bytebase account. For example, `alice@example.com`. - `group:{email}`: An email address for group. |
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
| rollout_policy | [RolloutPolicy](#bytebase-v1-RolloutPolicy) |  |  |
| masking_policy | [MaskingPolicy](#bytebase-v1-MaskingPolicy) |  |  |
| slow_query_policy | [SlowQueryPolicy](#bytebase-v1-SlowQueryPolicy) |  |  |
| disable_copy_data_policy | [DisableCopyDataPolicy](#bytebase-v1-DisableCopyDataPolicy) |  |  |
| masking_rule_policy | [MaskingRulePolicy](#bytebase-v1-MaskingRulePolicy) |  |  |
| masking_exception_policy | [MaskingExceptionPolicy](#bytebase-v1-MaskingExceptionPolicy) |  |  |
| restrict_issue_creation_for_sql_review_policy | [RestrictIssueCreationForSQLReviewPolicy](#bytebase-v1-RestrictIssueCreationForSQLReviewPolicy) |  |  |
| tag_policy | [TagPolicy](#bytebase-v1-TagPolicy) |  |  |
| enforce | [bool](#bool) |  |  |
| resource_type | [PolicyResourceType](#bytebase-v1-PolicyResourceType) |  | The resource type for the policy. |
| resource_uid | [string](#string) |  | The system-assigned, unique identifier for the resource. |






<a name="bytebase-v1-RestrictIssueCreationForSQLReviewPolicy"></a>

### RestrictIssueCreationForSQLReviewPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| disallow | [bool](#bool) |  |  |






<a name="bytebase-v1-RolloutPolicy"></a>

### RolloutPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| automatic | [bool](#bool) |  |  |
| workspace_roles | [string](#string) | repeated |  |
| project_roles | [string](#string) | repeated |  |
| issue_roles | [string](#string) | repeated | roles/LAST_APPROVER roles/CREATOR |






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






<a name="bytebase-v1-TagPolicy"></a>

### TagPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tags | [TagPolicy.TagsEntry](#bytebase-v1-TagPolicy-TagsEntry) | repeated | tags is the key - value map for resources. for example, the environment resource can have the sql review config tag, like &#34;bb.tag.review_config&#34;: &#34;reviewConfigs/{review config resource id}&#34; |






<a name="bytebase-v1-TagPolicy-TagsEntry"></a>

### TagPolicy.TagsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-v1-UpdatePolicyRequest"></a>

### UpdatePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [Policy](#bytebase-v1-Policy) |  | The policy to update.

The policy&#39;s `name` field is used to identify the instance to update. Format: {resource name}/policies/{policy type} Workspace resource name: &#34;&#34;. Environment resource name: environments/environment-id. Instance resource name: instances/instance-id. Database resource name: instances/instance-id/databases/database-name. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the policy is not found, a new policy will be created. In this situation, `update_mask` is ignored. |





 


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
| ROLLOUT_POLICY | 11 |  |
| MASKING | 5 |  |
| SLOW_QUERY | 7 |  |
| DISABLE_COPY_DATA | 8 |  |
| MASKING_RULE | 9 |  |
| MASKING_EXCEPTION | 10 |  |
| RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW | 12 |  |
| TAG | 13 |  |



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

 



<a name="v1_plan_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/plan_service.proto



<a name="bytebase-v1-BatchCancelPlanCheckRunsRequest"></a>

### BatchCancelPlanCheckRunsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The name of the parent of the planChecks. Format: projects/{project}/plans/{plan} |
| plan_check_runs | [string](#string) | repeated | The planCheckRuns to cancel. Format: projects/{project}/plans/{plan}/planCheckRuns/{planCheckRun} |






<a name="bytebase-v1-BatchCancelPlanCheckRunsResponse"></a>

### BatchCancelPlanCheckRunsResponse







<a name="bytebase-v1-CreatePlanRequest"></a>

### CreatePlanRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent project where this plan will be created. Format: projects/{project} |
| plan | [Plan](#bytebase-v1-Plan) |  | The plan to create. |






<a name="bytebase-v1-GetPlanRequest"></a>

### GetPlanRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the plan to retrieve. Format: projects/{project}/plans/{plan} |






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
| vcs_source | [Plan.VCSSource](#bytebase-v1-Plan-VCSSource) |  |  |
| creator | [string](#string) |  | Format: users/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| plan_check_run_status_count | [Plan.PlanCheckRunStatusCountEntry](#bytebase-v1-Plan-PlanCheckRunStatusCountEntry) | repeated | The status count of the latest plan check runs. Keys are: - SUCCESS - WARNING - ERROR |






<a name="bytebase-v1-Plan-ChangeDatabaseConfig"></a>

### Plan.ChangeDatabaseConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| target | [string](#string) |  | The resource name of the target. Format: instances/{instance-id}/databases/{database-name}. Format: projects/{project}/databaseGroups/{databaseGroup}. |
| sheet | [string](#string) |  | The resource name of the sheet. Format: projects/{project}/sheets/{sheet} |
| type | [Plan.ChangeDatabaseConfig.Type](#bytebase-v1-Plan-ChangeDatabaseConfig-Type) |  |  |
| schema_version | [string](#string) |  | schema_version is parsed from VCS file name. It is automatically generated in the UI workflow. |
| ghost_flags | [Plan.ChangeDatabaseConfig.GhostFlagsEntry](#bytebase-v1-Plan-ChangeDatabaseConfig-GhostFlagsEntry) | repeated |  |
| pre_update_backup_detail | [Plan.ChangeDatabaseConfig.PreUpdateBackupDetail](#bytebase-v1-Plan-ChangeDatabaseConfig-PreUpdateBackupDetail) | optional | If set, a backup of the modified data will be created automatically before any changes are applied. |






<a name="bytebase-v1-Plan-ChangeDatabaseConfig-GhostFlagsEntry"></a>

### Plan.ChangeDatabaseConfig.GhostFlagsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-v1-Plan-ChangeDatabaseConfig-PreUpdateBackupDetail"></a>

### Plan.ChangeDatabaseConfig.PreUpdateBackupDetail



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [string](#string) |  | The database for keeping the backup data. Format: instances/{instance}/databases/{database} |






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
| environment | [string](#string) |  | The environment resource. Format: environments/prod where prod is the environment resource ID. |
| labels | [Plan.CreateDatabaseConfig.LabelsEntry](#bytebase-v1-Plan-CreateDatabaseConfig-LabelsEntry) | repeated | labels of the database. |






<a name="bytebase-v1-Plan-CreateDatabaseConfig-LabelsEntry"></a>

### Plan.CreateDatabaseConfig.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-v1-Plan-ExportDataConfig"></a>

### Plan.ExportDataConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| target | [string](#string) |  | The resource name of the target. Format: instances/{instance-id}/databases/{database-name} |
| sheet | [string](#string) |  | The resource name of the sheet. Format: projects/{project}/sheets/{sheet} |
| format | [ExportFormat](#bytebase-v1-ExportFormat) |  | The format of the exported file. |
| password | [string](#string) | optional | The zip password provide by users. Leave it empty if no needs to encrypt the zip file. |






<a name="bytebase-v1-Plan-PlanCheckRunStatusCountEntry"></a>

### Plan.PlanCheckRunStatusCountEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [int32](#int32) |  |  |






<a name="bytebase-v1-Plan-Spec"></a>

### Plan.Spec



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| earliest_allowed_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | earliest_allowed_time the earliest execution time of the change. |
| id | [string](#string) |  | A UUID4 string that uniquely identifies the Spec. |
| depends_on_specs | [string](#string) | repeated | IDs of the specs that this spec depends on. Must be a subset of the specs in the same step. |
| create_database_config | [Plan.CreateDatabaseConfig](#bytebase-v1-Plan-CreateDatabaseConfig) |  |  |
| change_database_config | [Plan.ChangeDatabaseConfig](#bytebase-v1-Plan-ChangeDatabaseConfig) |  |  |
| export_data_config | [Plan.ExportDataConfig](#bytebase-v1-Plan-ExportDataConfig) |  |  |






<a name="bytebase-v1-Plan-Step"></a>

### Plan.Step



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  |  |
| specs | [Plan.Spec](#bytebase-v1-Plan-Spec) | repeated |  |






<a name="bytebase-v1-Plan-VCSSource"></a>

### Plan.VCSSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vcs_type | [VCSType](#bytebase-v1-VCSType) |  |  |
| vcs_connector | [string](#string) |  | Optional. If present, we will update the pull request for rollout status. Format: projects/{project-ID}/vcsConnectors/{vcs-connector} |
| pull_request_url | [string](#string) |  |  |






<a name="bytebase-v1-PlanCheckRun"></a>

### PlanCheckRun



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/plans/{plan}/planCheckRuns/{planCheckRun} |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| type | [PlanCheckRun.Type](#bytebase-v1-PlanCheckRun-Type) |  |  |
| status | [PlanCheckRun.Status](#bytebase-v1-PlanCheckRun-Status) |  |  |
| target | [string](#string) |  | Format: instances/{instance}/databases/{database} |
| sheet | [string](#string) |  | Format: project/{project}/sheets/{sheet} |
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
| code | [int32](#int32) |  |  |
| sql_summary_report | [PlanCheckRun.Result.SqlSummaryReport](#bytebase-v1-PlanCheckRun-Result-SqlSummaryReport) |  |  |
| sql_review_report | [PlanCheckRun.Result.SqlReviewReport](#bytebase-v1-PlanCheckRun-Result-SqlReviewReport) |  |  |






<a name="bytebase-v1-PlanCheckRun-Result-SqlReviewReport"></a>

### PlanCheckRun.Result.SqlReviewReport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| line | [int32](#int32) |  |  |
| column | [int32](#int32) |  |  |
| detail | [string](#string) |  |  |
| code | [int32](#int32) |  | Code from sql review. |
| start_position | [Position](#bytebase-v1-Position) |  | 1-based Position of the SQL statement. To supersede `line` and `column` above. |
| end_position | [Position](#bytebase-v1-Position) |  |  |






<a name="bytebase-v1-PlanCheckRun-Result-SqlSummaryReport"></a>

### PlanCheckRun.Result.SqlSummaryReport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [int32](#int32) |  |  |
| statement_types | [string](#string) | repeated | statement_types are the types of statements that are found in the sql. |
| affected_rows | [int32](#int32) |  |  |
| changed_resources | [ChangedResources](#bytebase-v1-ChangedResources) |  |  |






<a name="bytebase-v1-RunPlanChecksRequest"></a>

### RunPlanChecksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The plan to run plan checks. Format: projects/{project}/plans/{plan} |






<a name="bytebase-v1-RunPlanChecksResponse"></a>

### RunPlanChecksResponse







<a name="bytebase-v1-SearchPlansRequest"></a>

### SearchPlansRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of plans. Format: projects/{project} Use &#34;projects/-&#34; to list all plans from all projects. |
| page_size | [int32](#int32) |  | The maximum number of plans to return. The service may return fewer than this value. If unspecified, at most 50 plans will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListPlans` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListPlans` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter plans returned in the list. |






<a name="bytebase-v1-SearchPlansResponse"></a>

### SearchPlansResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| plans | [Plan](#bytebase-v1-Plan) | repeated | The plans from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






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
| DATABASE_STATEMENT_ADVISE | 3 |  |
| DATABASE_STATEMENT_SUMMARY_REPORT | 5 |  |
| DATABASE_CONNECT | 6 |  |
| DATABASE_GHOST_SYNC | 7 |  |


 

 


<a name="bytebase-v1-PlanService"></a>

### PlanService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetPlan | [GetPlanRequest](#bytebase-v1-GetPlanRequest) | [Plan](#bytebase-v1-Plan) |  |
| ListPlans | [ListPlansRequest](#bytebase-v1-ListPlansRequest) | [ListPlansResponse](#bytebase-v1-ListPlansResponse) |  |
| SearchPlans | [SearchPlansRequest](#bytebase-v1-SearchPlansRequest) | [SearchPlansResponse](#bytebase-v1-SearchPlansResponse) |  |
| CreatePlan | [CreatePlanRequest](#bytebase-v1-CreatePlanRequest) | [Plan](#bytebase-v1-Plan) |  |
| UpdatePlan | [UpdatePlanRequest](#bytebase-v1-UpdatePlanRequest) | [Plan](#bytebase-v1-Plan) |  |
| ListPlanCheckRuns | [ListPlanCheckRunsRequest](#bytebase-v1-ListPlanCheckRunsRequest) | [ListPlanCheckRunsResponse](#bytebase-v1-ListPlanCheckRunsResponse) |  |
| RunPlanChecks | [RunPlanChecksRequest](#bytebase-v1-RunPlanChecksRequest) | [RunPlanChecksResponse](#bytebase-v1-RunPlanChecksResponse) |  |
| BatchCancelPlanCheckRuns | [BatchCancelPlanCheckRunsRequest](#bytebase-v1-BatchCancelPlanCheckRunsRequest) | [BatchCancelPlanCheckRunsResponse](#bytebase-v1-BatchCancelPlanCheckRunsResponse) |  |

 



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






<a name="bytebase-v1-DatabaseGroup"></a>

### DatabaseGroup



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database group. Format: projects/{project}/databaseGroups/{databaseGroup} |
| database_placeholder | [string](#string) |  | The short name used in actual databases specified by users. For example, the placeholder for db1_2010, db1_2021, db1_2023 will be &#34;db1&#34;. |
| database_expr | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this database group. |
| matched_databases | [DatabaseGroup.Database](#bytebase-v1-DatabaseGroup-Database) | repeated | The list of databases that match the database group condition. |
| unmatched_databases | [DatabaseGroup.Database](#bytebase-v1-DatabaseGroup-Database) | repeated | The list of databases that match the database group condition. |
| multitenancy | [bool](#bool) |  |  |






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






<a name="bytebase-v1-DeploymentConfig"></a>

### DeploymentConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the resource. Format: projects/{project}/deploymentConfigs/default. |
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
| name | [string](#string) |  | The name of the resource. Format: projects/{project}/deploymentConfigs/default. |






<a name="bytebase-v1-GetProjectProtectionRulesRequest"></a>

### GetProjectProtectionRulesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the protection rules. Format: projects/{project}/protectionRules |






<a name="bytebase-v1-GetProjectRequest"></a>

### GetProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the project to retrieve. Format: projects/{project} |






<a name="bytebase-v1-Label"></a>

### Label



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [string](#string) |  |  |
| color | [string](#string) |  |  |
| group | [string](#string) |  |  |






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
| parent | [string](#string) |  | The parent resource whose database groups are to be listed. Format: projects/{project} |
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
| webhooks | [Webhook](#bytebase-v1-Webhook) | repeated |  |
| data_classification_config_id | [string](#string) |  |  |
| issue_labels | [Label](#bytebase-v1-Label) | repeated |  |
| force_issue_labels | [bool](#bool) |  | Force issue labels to be used when creating an issue. |
| allow_modify_statement | [bool](#bool) |  | Allow modifying statement after issue is created. |
| auto_resolve_issue | [bool](#bool) |  | Enable auto resolve issue. |






<a name="bytebase-v1-ProtectionRule"></a>

### ProtectionRule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | A unique identifier for a node in UUID format. |
| target | [ProtectionRule.Target](#bytebase-v1-ProtectionRule-Target) |  |  |
| name_filter | [string](#string) |  | The name of the branch/changelist or wildcard. |
| allowed_roles | [string](#string) | repeated | The roles allowed to create branches or changelists, rebase branches, delete branches. Format: roles/projectOwner. |
| branch_source | [ProtectionRule.BranchSource](#bytebase-v1-ProtectionRule-BranchSource) |  |  |






<a name="bytebase-v1-ProtectionRules"></a>

### ProtectionRules



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the protection rules. Format: projects/{project}/protectionRules |
| rules | [ProtectionRule](#bytebase-v1-ProtectionRule) | repeated |  |






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






<a name="bytebase-v1-SearchProjectsRequest"></a>

### SearchProjectsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| show_deleted | [bool](#bool) |  | Show deleted projects if specified. |






<a name="bytebase-v1-SearchProjectsResponse"></a>

### SearchProjectsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| projects | [Project](#bytebase-v1-Project) | repeated | The projects from the specified request. |






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






<a name="bytebase-v1-UpdateProjectProtectionRulesRequest"></a>

### UpdateProjectProtectionRulesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| protection_rules | [ProtectionRules](#bytebase-v1-ProtectionRules) |  |  |






<a name="bytebase-v1-UpdateProjectRequest"></a>

### UpdateProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [Project](#bytebase-v1-Project) |  | The project to update.

The project&#39;s `name` field is used to identify the project to update. Format: projects/{project} |
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
| direct_message | [bool](#bool) |  | if direct_message is set, the notification is sent directly to the persons and url will be ignored. IM integration setting should be set for this function to work. |
| notification_types | [Activity.Type](#bytebase-v1-Activity-Type) | repeated | notification_types is the list of activities types that the webhook is interested in. Bytebase will only send notifications to the webhook if the activity type is in the list. It should not be empty, and should be a subset of the following: - TYPE_ISSUE_CREATED - TYPE_ISSUE_STATUS_UPDATE - TYPE_ISSUE_PIPELINE_STAGE_UPDATE - TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE - TYPE_ISSUE_FIELD_UPDATE - TYPE_ISSUE_COMMENT_CREATE |





 


<a name="bytebase-v1-Activity-Type"></a>

### Activity.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| TYPE_NOTIFY_ISSUE_APPROVED | 23 | Notifications via webhooks.

TYPE_NOTIFY_ISSUE_APPROVED represents the issue approved notification. |
| TYPE_NOTIFY_PIPELINE_ROLLOUT | 24 | TYPE_NOTIFY_PIPELINE_ROLLOUT represents the pipeline rollout notification. |
| TYPE_ISSUE_CREATE | 1 | Issue related activity types.

TYPE_ISSUE_CREATE represents creating an issue. |
| TYPE_ISSUE_COMMENT_CREATE | 2 | TYPE_ISSUE_COMMENT_CREATE represents commenting on an issue. |
| TYPE_ISSUE_FIELD_UPDATE | 3 | TYPE_ISSUE_FIELD_UPDATE represents updating the issue field, likes title, description, assignee, etc. |
| TYPE_ISSUE_STATUS_UPDATE | 4 | TYPE_ISSUE_STATUS_UPDATE represents the issue status change, including OPEN, CLOSE, CANCEL fow now. |
| TYPE_ISSUE_APPROVAL_NOTIFY | 21 | TYPE_ISSUE_APPROVAL_NOTIFY is the type for notifying issue approval. |
| TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE | 5 | TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE represents the pipeline stage status change, including BEGIN, END for now. |
| TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE | 6 | TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE represents the pipeline task status change, including PENDING, PENDING_APPROVAL, RUNNING, SUCCESS, FAILURE, CANCELED for now. |
| TYPE_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE | 22 | TYPE_ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE represents the pipeline task run status change, including PENDING, RUNNING, DONE, FAILED, CANCELED. |
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
| OPERATOR_TYPE_NOT_IN | 3 | The operator is &#34;Not In&#34;. |



<a name="bytebase-v1-ProtectionRule-BranchSource"></a>

### ProtectionRule.BranchSource


| Name | Number | Description |
| ---- | ------ | ----------- |
| BRANCH_SOURCE_UNSPECIFIED | 0 |  |
| DATABASE | 1 |  |



<a name="bytebase-v1-ProtectionRule-Target"></a>

### ProtectionRule.Target
The type of target.

| Name | Number | Description |
| ---- | ------ | ----------- |
| PROTECTION_TARGET_UNSPECIFIED | 0 |  |
| BRANCH | 1 |  |
| CHANGELIST | 2 |  |



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
| SearchProjects | [SearchProjectsRequest](#bytebase-v1-SearchProjectsRequest) | [SearchProjectsResponse](#bytebase-v1-SearchProjectsResponse) |  |
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
| ListDatabaseGroups | [ListDatabaseGroupsRequest](#bytebase-v1-ListDatabaseGroupsRequest) | [ListDatabaseGroupsResponse](#bytebase-v1-ListDatabaseGroupsResponse) |  |
| GetDatabaseGroup | [GetDatabaseGroupRequest](#bytebase-v1-GetDatabaseGroupRequest) | [DatabaseGroup](#bytebase-v1-DatabaseGroup) |  |
| CreateDatabaseGroup | [CreateDatabaseGroupRequest](#bytebase-v1-CreateDatabaseGroupRequest) | [DatabaseGroup](#bytebase-v1-DatabaseGroup) |  |
| UpdateDatabaseGroup | [UpdateDatabaseGroupRequest](#bytebase-v1-UpdateDatabaseGroupRequest) | [DatabaseGroup](#bytebase-v1-DatabaseGroup) |  |
| DeleteDatabaseGroup | [DeleteDatabaseGroupRequest](#bytebase-v1-DeleteDatabaseGroupRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| GetProjectProtectionRules | [GetProjectProtectionRulesRequest](#bytebase-v1-GetProjectProtectionRulesRequest) | [ProtectionRules](#bytebase-v1-ProtectionRules) |  |
| UpdateProjectProtectionRules | [UpdateProjectProtectionRulesRequest](#bytebase-v1-UpdateProjectProtectionRulesRequest) | [ProtectionRules](#bytebase-v1-ProtectionRules) |  |

 



<a name="v1_review_config_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/review_config_service.proto



<a name="bytebase-v1-CreateReviewConfigRequest"></a>

### CreateReviewConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| review_config | [ReviewConfig](#bytebase-v1-ReviewConfig) |  | The sql review to create. |






<a name="bytebase-v1-DeleteReviewConfigRequest"></a>

### DeleteReviewConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the sql review to delete. Format: reviewConfigs/{uid} |






<a name="bytebase-v1-GetReviewConfigRequest"></a>

### GetReviewConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the sql review to retrieve. Format: reviewConfigs/{uid} |






<a name="bytebase-v1-ListReviewConfigsRequest"></a>

### ListReviewConfigsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of sql review to return. The service may return fewer than this value. If unspecified, at most 50 sql review will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListSQLReviews` must match the call that provided the page token. |






<a name="bytebase-v1-ListReviewConfigsResponse"></a>

### ListReviewConfigsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| review_configs | [ReviewConfig](#bytebase-v1-ReviewConfig) | repeated | The sql review from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-ReviewConfig"></a>

### ReviewConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the sql review to retrieve. Format: reviewConfigs/{uid} |
| title | [string](#string) |  |  |
| enabled | [bool](#bool) |  |  |
| creator | [string](#string) |  | Format: users/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| rules | [SQLReviewRule](#bytebase-v1-SQLReviewRule) | repeated |  |
| resources | [string](#string) | repeated | resources using the config. Format: {resurce}/{resource id}, for example, environments/test. |






<a name="bytebase-v1-UpdateReviewConfigRequest"></a>

### UpdateReviewConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| review_config | [ReviewConfig](#bytebase-v1-ReviewConfig) |  | The sql review toupdate.

The name field is used to identify the sql review to update. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 

 

 


<a name="bytebase-v1-ReviewConfigService"></a>

### ReviewConfigService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateReviewConfig | [CreateReviewConfigRequest](#bytebase-v1-CreateReviewConfigRequest) | [ReviewConfig](#bytebase-v1-ReviewConfig) |  |
| ListReviewConfigs | [ListReviewConfigsRequest](#bytebase-v1-ListReviewConfigsRequest) | [ListReviewConfigsResponse](#bytebase-v1-ListReviewConfigsResponse) |  |
| GetReviewConfig | [GetReviewConfigRequest](#bytebase-v1-GetReviewConfigRequest) | [ReviewConfig](#bytebase-v1-ReviewConfig) |  |
| UpdateReviewConfig | [UpdateReviewConfigRequest](#bytebase-v1-UpdateReviewConfigRequest) | [ReviewConfig](#bytebase-v1-ReviewConfig) |  |
| DeleteReviewConfig | [DeleteReviewConfigRequest](#bytebase-v1-DeleteReviewConfigRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



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
| level | [int32](#int32) |  |  |
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
| REQUEST_QUERY | 4 |  |
| REQUEST_EXPORT | 5 |  |
| DATA_EXPORT | 6 |  |


 

 


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
| permissions | [string](#string) | repeated |  |






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







<a name="bytebase-v1-CreateRolloutRequest"></a>

### CreateRolloutRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent project where this rollout will be created. Format: projects/{project} |
| rollout | [Rollout](#bytebase-v1-Rollout) |  | The rollout to create. |






<a name="bytebase-v1-GetRolloutRequest"></a>

### GetRolloutRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the rollout to retrieve. Format: projects/{project}/rollouts/{rollout} |






<a name="bytebase-v1-GetTaskRunLogRequest"></a>

### GetTaskRunLogRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |






<a name="bytebase-v1-GetTaskRunSessionRequest"></a>

### GetTaskRunSessionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |






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






<a name="bytebase-v1-Stage"></a>

### Stage



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage} |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
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
| depends_on_tasks | [string](#string) | repeated | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} |
| target | [string](#string) |  | Format: instances/{instance} if the task is DatabaseCreate. Format: instances/{instance}/databases/{database} |
| database_create | [Task.DatabaseCreate](#bytebase-v1-Task-DatabaseCreate) |  |  |
| database_schema_baseline | [Task.DatabaseSchemaBaseline](#bytebase-v1-Task-DatabaseSchemaBaseline) |  |  |
| database_schema_update | [Task.DatabaseSchemaUpdate](#bytebase-v1-Task-DatabaseSchemaUpdate) |  |  |
| database_data_update | [Task.DatabaseDataUpdate](#bytebase-v1-Task-DatabaseDataUpdate) |  |  |
| database_data_export | [Task.DatabaseDataExport](#bytebase-v1-Task-DatabaseDataExport) |  |  |






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
| environment | [string](#string) |  |  |
| labels | [Task.DatabaseCreate.LabelsEntry](#bytebase-v1-Task-DatabaseCreate-LabelsEntry) | repeated |  |






<a name="bytebase-v1-Task-DatabaseCreate-LabelsEntry"></a>

### Task.DatabaseCreate.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-v1-Task-DatabaseDataExport"></a>

### Task.DatabaseDataExport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| target | [string](#string) |  | The resource name of the target. Format: instances/{instance-id}/databases/{database-name} |
| sheet | [string](#string) |  | The resource name of the sheet. Format: projects/{project}/sheets/{sheet} |
| format | [ExportFormat](#bytebase-v1-ExportFormat) |  | The format of the exported file. |
| password | [string](#string) | optional | The zip password provide by users. Leave it empty if no needs to encrypt the zip file. |






<a name="bytebase-v1-Task-DatabaseDataUpdate"></a>

### Task.DatabaseDataUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheet | [string](#string) |  | Format: projects/{project}/sheets/{sheet} |
| schema_version | [string](#string) |  |  |






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
| execution_status | [TaskRun.ExecutionStatus](#bytebase-v1-TaskRun-ExecutionStatus) |  |  |
| execution_detail | [TaskRun.ExecutionDetail](#bytebase-v1-TaskRun-ExecutionDetail) |  |  |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| export_archive_status | [TaskRun.ExportArchiveStatus](#bytebase-v1-TaskRun-ExportArchiveStatus) |  |  |






<a name="bytebase-v1-TaskRun-ExecutionDetail"></a>

### TaskRun.ExecutionDetail



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| commands_total | [int32](#int32) |  | Currently, the following fields are only used for EXECUTING status. |
| commands_completed | [int32](#int32) |  |  |
| command_start_position | [TaskRun.ExecutionDetail.Position](#bytebase-v1-TaskRun-ExecutionDetail-Position) |  |  |
| command_end_position | [TaskRun.ExecutionDetail.Position](#bytebase-v1-TaskRun-ExecutionDetail-Position) |  |  |






<a name="bytebase-v1-TaskRun-ExecutionDetail-Position"></a>

### TaskRun.ExecutionDetail.Position



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| line | [int32](#int32) |  | The line number, starting from 0. |
| column | [int32](#int32) |  | The column number, starting from 0. |






<a name="bytebase-v1-TaskRunLog"></a>

### TaskRunLog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun}/log |
| entries | [TaskRunLogEntry](#bytebase-v1-TaskRunLogEntry) | repeated |  |






<a name="bytebase-v1-TaskRunLogEntry"></a>

### TaskRunLogEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [TaskRunLogEntry.Type](#bytebase-v1-TaskRunLogEntry-Type) |  |  |
| log_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| schema_dump | [TaskRunLogEntry.SchemaDump](#bytebase-v1-TaskRunLogEntry-SchemaDump) |  |  |
| command_execute | [TaskRunLogEntry.CommandExecute](#bytebase-v1-TaskRunLogEntry-CommandExecute) |  |  |
| database_sync | [TaskRunLogEntry.DatabaseSync](#bytebase-v1-TaskRunLogEntry-DatabaseSync) |  |  |
| task_run_status_update | [TaskRunLogEntry.TaskRunStatusUpdate](#bytebase-v1-TaskRunLogEntry-TaskRunStatusUpdate) |  |  |
| transaction_control | [TaskRunLogEntry.TransactionControl](#bytebase-v1-TaskRunLogEntry-TransactionControl) |  |  |






<a name="bytebase-v1-TaskRunLogEntry-CommandExecute"></a>

### TaskRunLogEntry.CommandExecute



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| log_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| command_indexes | [int32](#int32) | repeated | The indexes of the executed commands. |
| response | [TaskRunLogEntry.CommandExecute.CommandResponse](#bytebase-v1-TaskRunLogEntry-CommandExecute-CommandResponse) |  |  |






<a name="bytebase-v1-TaskRunLogEntry-CommandExecute-CommandResponse"></a>

### TaskRunLogEntry.CommandExecute.CommandResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| log_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| error | [string](#string) |  |  |
| affected_rows | [int32](#int32) |  |  |
| all_affected_rows | [int32](#int32) | repeated | `all_affected_rows` is the affected rows of each command. `all_affected_rows` may be unavailable if the database driver doesn&#39;t support it. Caller should fallback to `affected_rows` in that case. |






<a name="bytebase-v1-TaskRunLogEntry-DatabaseSync"></a>

### TaskRunLogEntry.DatabaseSync



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| end_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| error | [string](#string) |  |  |






<a name="bytebase-v1-TaskRunLogEntry-SchemaDump"></a>

### TaskRunLogEntry.SchemaDump



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| end_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| error | [string](#string) |  |  |






<a name="bytebase-v1-TaskRunLogEntry-TaskRunStatusUpdate"></a>

### TaskRunLogEntry.TaskRunStatusUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [TaskRunLogEntry.TaskRunStatusUpdate.Status](#bytebase-v1-TaskRunLogEntry-TaskRunStatusUpdate-Status) |  |  |






<a name="bytebase-v1-TaskRunLogEntry-TransactionControl"></a>

### TaskRunLogEntry.TransactionControl



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [TaskRunLogEntry.TransactionControl.Type](#bytebase-v1-TaskRunLogEntry-TransactionControl-Type) |  |  |
| error | [string](#string) |  |  |






<a name="bytebase-v1-TaskRunSession"></a>

### TaskRunSession



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun}/session |
| postgres | [TaskRunSession.Postgres](#bytebase-v1-TaskRunSession-Postgres) |  |  |






<a name="bytebase-v1-TaskRunSession-Postgres"></a>

### TaskRunSession.Postgres



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| session | [TaskRunSession.Postgres.Session](#bytebase-v1-TaskRunSession-Postgres-Session) |  | `session` is the session of the task run executing commands. |
| blocking_sessions | [TaskRunSession.Postgres.Session](#bytebase-v1-TaskRunSession-Postgres-Session) | repeated | `blocking_sessions` block `session`. |
| blocked_sessions | [TaskRunSession.Postgres.Session](#bytebase-v1-TaskRunSession-Postgres-Session) | repeated | `blocked_sessions` are blocked by `session`. |






<a name="bytebase-v1-TaskRunSession-Postgres-Session"></a>

### TaskRunSession.Postgres.Session
Read from `pg_stat_activity`


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pid | [string](#string) |  |  |
| blocked_by_pids | [string](#string) | repeated |  |
| query | [string](#string) |  |  |
| state | [string](#string) | optional |  |
| wait_event_type | [string](#string) | optional |  |
| wait_event | [string](#string) | optional |  |
| datname | [string](#string) | optional |  |
| usename | [string](#string) | optional |  |
| application_name | [string](#string) |  |  |
| client_addr | [string](#string) | optional |  |
| client_port | [string](#string) | optional |  |
| backend_start | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| xact_start | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional |  |
| query_start | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional |  |





 


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
| DATABASE_DATA_EXPORT | 12 | use payload DatabaseDataExport |



<a name="bytebase-v1-TaskRun-ExecutionStatus"></a>

### TaskRun.ExecutionStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| EXECUTION_STATUS_UNSPECIFIED | 0 |  |
| PRE_EXECUTING | 1 |  |
| EXECUTING | 2 |  |
| POST_EXECUTING | 3 |  |



<a name="bytebase-v1-TaskRun-ExportArchiveStatus"></a>

### TaskRun.ExportArchiveStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| EXPORT_ARCHIVE_STATUS_UNSPECIFIED | 0 |  |
| READY | 1 |  |
| EXPORTED | 2 |  |



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



<a name="bytebase-v1-TaskRunLogEntry-TaskRunStatusUpdate-Status"></a>

### TaskRunLogEntry.TaskRunStatusUpdate.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| RUNNING_WAITING | 1 | the task run is ready to be executed by the scheduler |
| RUNNING_RUNNING | 2 | the task run is being executed by the scheduler |



<a name="bytebase-v1-TaskRunLogEntry-TransactionControl-Type"></a>

### TaskRunLogEntry.TransactionControl.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| BEGIN | 1 |  |
| COMMIT | 2 |  |
| ROLLBACK | 3 |  |



<a name="bytebase-v1-TaskRunLogEntry-Type"></a>

### TaskRunLogEntry.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| SCHEMA_DUMP | 1 |  |
| COMMAND_EXECUTE | 2 |  |
| DATABASE_SYNC | 3 |  |
| TASK_RUN_STATUS_UPDATE | 4 |  |
| TRANSACTION_CONTROL | 5 |  |


 

 


<a name="bytebase-v1-RolloutService"></a>

### RolloutService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetRollout | [GetRolloutRequest](#bytebase-v1-GetRolloutRequest) | [Rollout](#bytebase-v1-Rollout) |  |
| CreateRollout | [CreateRolloutRequest](#bytebase-v1-CreateRolloutRequest) | [Rollout](#bytebase-v1-Rollout) |  |
| PreviewRollout | [PreviewRolloutRequest](#bytebase-v1-PreviewRolloutRequest) | [Rollout](#bytebase-v1-Rollout) |  |
| ListTaskRuns | [ListTaskRunsRequest](#bytebase-v1-ListTaskRunsRequest) | [ListTaskRunsResponse](#bytebase-v1-ListTaskRunsResponse) |  |
| GetTaskRunLog | [GetTaskRunLogRequest](#bytebase-v1-GetTaskRunLogRequest) | [TaskRunLog](#bytebase-v1-TaskRunLog) |  |
| GetTaskRunSession | [GetTaskRunSessionRequest](#bytebase-v1-GetTaskRunSessionRequest) | [TaskRunSession](#bytebase-v1-TaskRunSession) |  |
| BatchRunTasks | [BatchRunTasksRequest](#bytebase-v1-BatchRunTasksRequest) | [BatchRunTasksResponse](#bytebase-v1-BatchRunTasksResponse) |  |
| BatchSkipTasks | [BatchSkipTasksRequest](#bytebase-v1-BatchSkipTasksRequest) | [BatchSkipTasksResponse](#bytebase-v1-BatchSkipTasksResponse) |  |
| BatchCancelTaskRuns | [BatchCancelTaskRunsRequest](#bytebase-v1-BatchCancelTaskRunsRequest) | [BatchCancelTaskRunsResponse](#bytebase-v1-BatchCancelTaskRunsResponse) |  |

 



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

 



<a name="v1_setting_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/setting_service.proto



<a name="bytebase-v1-AgentPluginSetting"></a>

### AgentPluginSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | [string](#string) |  | The URL for the agent API. |
| token | [string](#string) |  | The token for the agent. |






<a name="bytebase-v1-Announcement"></a>

### Announcement



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| level | [Announcement.AlertLevel](#bytebase-v1-Announcement-AlertLevel) |  | The alert level of announcemnt |
| text | [string](#string) |  | The text of announcemnt |
| link | [string](#string) |  | The optional link, user can follow the link to check extra details |






<a name="bytebase-v1-AppIMSetting"></a>

### AppIMSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| slack | [AppIMSetting.Slack](#bytebase-v1-AppIMSetting-Slack) |  |  |
| feishu | [AppIMSetting.Feishu](#bytebase-v1-AppIMSetting-Feishu) |  |  |
| wecom | [AppIMSetting.Wecom](#bytebase-v1-AppIMSetting-Wecom) |  |  |






<a name="bytebase-v1-AppIMSetting-Feishu"></a>

### AppIMSetting.Feishu



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| app_id | [string](#string) |  |  |
| app_secret | [string](#string) |  |  |






<a name="bytebase-v1-AppIMSetting-Slack"></a>

### AppIMSetting.Slack



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| token | [string](#string) |  |  |






<a name="bytebase-v1-AppIMSetting-Wecom"></a>

### AppIMSetting.Wecom



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| corp_id | [string](#string) |  |  |
| agent_id | [string](#string) |  |  |
| secret | [string](#string) |  |  |






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
| classification_from_config | [bool](#bool) |  | If true, we will only store the classification in the config. Otherwise we will get the classification from table/column comment, and write back to the schema metadata. |






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






<a name="bytebase-v1-MaskingAlgorithmSetting"></a>

### MaskingAlgorithmSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| algorithms | [MaskingAlgorithmSetting.Algorithm](#bytebase-v1-MaskingAlgorithmSetting-Algorithm) | repeated | algorithms is the list of masking algorithms. |






<a name="bytebase-v1-MaskingAlgorithmSetting-Algorithm"></a>

### MaskingAlgorithmSetting.Algorithm



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | id is the uuid for masking algorithm. |
| title | [string](#string) |  | title is the title for masking algorithm. |
| description | [string](#string) |  | description is the description for masking algorithm. |
| category | [string](#string) |  | Category is the category for masking algorithm. Currently, it accepts 2 categories only: MASK and HASH. The range of accepted Payload is decided by the category. MASK: FullMask, RangeMask HASH: MD5Mask |
| full_mask | [MaskingAlgorithmSetting.Algorithm.FullMask](#bytebase-v1-MaskingAlgorithmSetting-Algorithm-FullMask) |  |  |
| range_mask | [MaskingAlgorithmSetting.Algorithm.RangeMask](#bytebase-v1-MaskingAlgorithmSetting-Algorithm-RangeMask) |  |  |
| md5_mask | [MaskingAlgorithmSetting.Algorithm.MD5Mask](#bytebase-v1-MaskingAlgorithmSetting-Algorithm-MD5Mask) |  |  |
| inner_outer_mask | [MaskingAlgorithmSetting.Algorithm.InnerOuterMask](#bytebase-v1-MaskingAlgorithmSetting-Algorithm-InnerOuterMask) |  |  |






<a name="bytebase-v1-MaskingAlgorithmSetting-Algorithm-FullMask"></a>

### MaskingAlgorithmSetting.Algorithm.FullMask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| substitution | [string](#string) |  | substitution is the string used to replace the original value, the max length of the string is 16 bytes. |






<a name="bytebase-v1-MaskingAlgorithmSetting-Algorithm-InnerOuterMask"></a>

### MaskingAlgorithmSetting.Algorithm.InnerOuterMask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| prefix_len | [int32](#int32) |  |  |
| suffix_len | [int32](#int32) |  |  |
| type | [MaskingAlgorithmSetting.Algorithm.InnerOuterMask.MaskType](#bytebase-v1-MaskingAlgorithmSetting-Algorithm-InnerOuterMask-MaskType) |  |  |
| substitution | [string](#string) |  |  |






<a name="bytebase-v1-MaskingAlgorithmSetting-Algorithm-MD5Mask"></a>

### MaskingAlgorithmSetting.Algorithm.MD5Mask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| salt | [string](#string) |  | salt is the salt value to generate a different hash that with the word alone. |






<a name="bytebase-v1-MaskingAlgorithmSetting-Algorithm-RangeMask"></a>

### MaskingAlgorithmSetting.Algorithm.RangeMask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| slices | [MaskingAlgorithmSetting.Algorithm.RangeMask.Slice](#bytebase-v1-MaskingAlgorithmSetting-Algorithm-RangeMask-Slice) | repeated | We store it as a repeated field to face the fact that the original value may have multiple parts should be masked. But frontend can be started with a single rule easily. |






<a name="bytebase-v1-MaskingAlgorithmSetting-Algorithm-RangeMask-Slice"></a>

### MaskingAlgorithmSetting.Algorithm.RangeMask.Slice



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start | [int32](#int32) |  | start is the start index of the original value, start from 0 and should be less than stop. |
| end | [int32](#int32) |  | stop is the stop index of the original value, should be less than the length of the original value. |
| substitution | [string](#string) |  | substitution is the string used to replace the OriginalValue[start:end). |






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
| table_templates | [SchemaTemplateSetting.TableTemplate](#bytebase-v1-SchemaTemplateSetting-TableTemplate) | repeated |  |






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
| config | [ColumnConfig](#bytebase-v1-ColumnConfig) |  |  |






<a name="bytebase-v1-SchemaTemplateSetting-TableTemplate"></a>

### SchemaTemplateSetting.TableTemplate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| engine | [Engine](#bytebase-v1-Engine) |  |  |
| category | [string](#string) |  |  |
| table | [TableMetadata](#bytebase-v1-TableMetadata) |  |  |
| config | [TableConfig](#bytebase-v1-TableConfig) |  |  |






<a name="bytebase-v1-SemanticTypeSetting"></a>

### SemanticTypeSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| types | [SemanticTypeSetting.SemanticType](#bytebase-v1-SemanticTypeSetting-SemanticType) | repeated |  |






<a name="bytebase-v1-SemanticTypeSetting-SemanticType"></a>

### SemanticTypeSetting.SemanticType



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | id is the uuid for semantic type. |
| title | [string](#string) |  | the title of the semantic type, it should not be empty. |
| description | [string](#string) |  | the description of the semantic type, it can be empty. |
| partial_mask_algorithm_id | [string](#string) |  | the partial mask algorithm id for the semantic type, if it is empty, should use the default partial mask algorithm. |
| full_mask_algorithm_id | [string](#string) |  | the full mask algorithm id for the semantic type, if it is empty, should use the default full mask algorithm. |






<a name="bytebase-v1-Setting"></a>

### Setting
The schema of setting.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The resource name of the setting. Must be one of the following forms:

- `setting/{setting_name}` For example, &#34;settings/bb.branding.logo&#34; |
| value | [Value](#bytebase-v1-Value) |  | The value of the setting. |






<a name="bytebase-v1-UpdateSettingRequest"></a>

### UpdateSettingRequest
The request message for updating or creating a setting.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| setting | [Setting](#bytebase-v1-Setting) |  | The setting to update. |
| validate_only | [bool](#bool) |  | validate_only is a flag to indicate whether to validate the setting value, server would not persist the setting value if it is true. |
| allow_missing | [bool](#bool) |  |  |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  |  |






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
| semantic_type_setting_value | [SemanticTypeSetting](#bytebase-v1-SemanticTypeSetting) |  |  |
| masking_algorithm_setting_value | [MaskingAlgorithmSetting](#bytebase-v1-MaskingAlgorithmSetting) |  |  |






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
| token_duration | [google.protobuf.Duration](#google-protobuf-Duration) |  | The duration for token. |
| announcement | [Announcement](#bytebase-v1-Announcement) |  | The setting of custom announcement |
| maximum_role_expiration | [google.protobuf.Duration](#google-protobuf-Duration) |  | The max duration for role expired. |
| domains | [string](#string) | repeated | The workspace domain, e.g. bytebase.com. |
| enforce_identity_domain | [bool](#bool) |  | Only user and group from the domains can be created and login. |
| database_change_mode | [DatabaseChangeMode](#bytebase-v1-DatabaseChangeMode) |  | The workspace database change mode. |






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





 


<a name="bytebase-v1-Announcement-AlertLevel"></a>

### Announcement.AlertLevel
We support three levels of AlertLevel: INFO, WARNING, and ERROR.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ALERT_LEVEL_UNSPECIFIED | 0 |  |
| ALERT_LEVEL_INFO | 1 |  |
| ALERT_LEVEL_WARNING | 2 |  |
| ALERT_LEVEL_CRITICAL | 3 |  |



<a name="bytebase-v1-DatabaseChangeMode"></a>

### DatabaseChangeMode


| Name | Number | Description |
| ---- | ------ | ----------- |
| DATABASE_CHANGE_MODE_UNSPECIFIED | 0 |  |
| PIPELINE | 1 | A more advanced database change process, including custom approval workflows and other advanced features. Default to this mode. |
| EDITOR | 2 | A simple database change process in SQL editor. Users can execute SQL directly. |



<a name="bytebase-v1-MaskingAlgorithmSetting-Algorithm-InnerOuterMask-MaskType"></a>

### MaskingAlgorithmSetting.Algorithm.InnerOuterMask.MaskType


| Name | Number | Description |
| ---- | ------ | ----------- |
| MASK_TYPE_UNSPECIFIED | 0 |  |
| INNER | 1 |  |
| OUTER | 2 |  |



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
| UpdateSetting | [UpdateSettingRequest](#bytebase-v1-UpdateSettingRequest) | [Setting](#bytebase-v1-Setting) |  |

 



<a name="v1_sheet_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/sheet_service.proto



<a name="bytebase-v1-CreateSheetRequest"></a>

### CreateSheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this sheet will be created. Format: projects/{project} |
| sheet | [Sheet](#bytebase-v1-Sheet) |  | The sheet to create. |






<a name="bytebase-v1-GetSheetRequest"></a>

### GetSheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the sheet to retrieve. Format: projects/{project}/sheets/{sheet} |
| raw | [bool](#bool) |  | By default, the content of the sheet is cut off, set the `raw` to true to retrieve the full content. |






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
| payload | [SheetPayload](#bytebase-v1-SheetPayload) |  |  |
| engine | [Engine](#bytebase-v1-Engine) |  | The SQL dialect. |






<a name="bytebase-v1-SheetCommand"></a>

### SheetCommand



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start | [int32](#int32) |  |  |
| end | [int32](#int32) |  |  |






<a name="bytebase-v1-SheetPayload"></a>

### SheetPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [SheetPayload.Type](#bytebase-v1-SheetPayload-Type) |  |  |
| database_config | [DatabaseConfig](#bytebase-v1-DatabaseConfig) |  | The snapshot of the database config when creating the sheet, be used to compare with the baseline_database_config and apply the diff to the database. |
| baseline_database_config | [DatabaseConfig](#bytebase-v1-DatabaseConfig) |  | The snapshot of the baseline database config when creating the sheet. |
| commands | [SheetCommand](#bytebase-v1-SheetCommand) | repeated | The start and end position of each command in the sheet statement. |






<a name="bytebase-v1-UpdateSheetRequest"></a>

### UpdateSheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheet | [Sheet](#bytebase-v1-Sheet) |  | The sheet to update.

The sheet&#39;s `name` field is used to identify the sheet to update. Format: projects/{project}/sheets/{sheet} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to be updated. Fields are specified relative to the sheet. (e.g. `title`, `statement`; *not* `sheet.title` or `sheet.statement`) Only support update the following fields for now: - `title` - `statement` |





 


<a name="bytebase-v1-SheetPayload-Type"></a>

### SheetPayload.Type
Type of the SheetPayload.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| SCHEMA_DESIGN | 1 |  |


 

 


<a name="bytebase-v1-SheetService"></a>

### SheetService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateSheet | [CreateSheetRequest](#bytebase-v1-CreateSheetRequest) | [Sheet](#bytebase-v1-Sheet) |  |
| GetSheet | [GetSheetRequest](#bytebase-v1-GetSheetRequest) | [Sheet](#bytebase-v1-Sheet) |  |
| UpdateSheet | [UpdateSheetRequest](#bytebase-v1-UpdateSheetRequest) | [Sheet](#bytebase-v1-Sheet) |  |

 



<a name="v1_sql_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/sql_service.proto



<a name="bytebase-v1-AdminExecuteRequest"></a>

### AdminExecuteRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the instance name to execute the query against. Format: instances/{instance}/databases/{databaseName} |
| connection_database | [string](#string) |  | **Deprecated.**  |
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
| column | [int32](#int32) |  | The advice column number in the SQL statement. |
| detail | [string](#string) |  | The advice detail. |
| start_position | [Position](#bytebase-v1-Position) |  | 1-based Position of the SQL statement. To supersede `line` and `column` above. |
| end_position | [Position](#bytebase-v1-Position) |  |  |






<a name="bytebase-v1-CheckRequest"></a>

### CheckRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| statement | [string](#string) |  |  |
| database | [string](#string) |  | The database name to check against. Format: instances/{instance}/databases/{databaseName} |
| metadata | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  | The database metadata to check against. It can be used to check against an uncommitted metadata. If not provided, the database metadata will be fetched from the database. |
| change_type | [CheckRequest.ChangeType](#bytebase-v1-CheckRequest-ChangeType) |  |  |






<a name="bytebase-v1-CheckResponse"></a>

### CheckResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| advices | [Advice](#bytebase-v1-Advice) | repeated |  |






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






<a name="bytebase-v1-ExecuteRequest"></a>

### ExecuteRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the instance name to execute the query against. Format: instances/{instance}/databases/{databaseName} |
| statement | [string](#string) |  | The SQL statement to execute. |
| limit | [int32](#int32) |  | The maximum number of rows to return. |
| timeout | [google.protobuf.Duration](#google-protobuf-Duration) |  | The timeout for the request. |






<a name="bytebase-v1-ExecuteResponse"></a>

### ExecuteResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [QueryResult](#bytebase-v1-QueryResult) | repeated | The execute results. |
| advices | [Advice](#bytebase-v1-Advice) | repeated | The execute advices. |






<a name="bytebase-v1-ExportRequest"></a>

### ExportRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the instance name to execute the query against. Format: instances/{instance}/databases/{databaseName} Format: projects/{project}/issues/{issue} for data export issue. |
| connection_database | [string](#string) |  | **Deprecated.**  |
| statement | [string](#string) |  | The SQL statement to execute. |
| limit | [int32](#int32) |  | The maximum number of rows to return. |
| format | [ExportFormat](#bytebase-v1-ExportFormat) |  | The export format. |
| admin | [bool](#bool) |  | The admin is used for workspace owner and DBA for exporting data from SQL Editor Admin mode. The exported data is not masked. |
| password | [string](#string) |  | The zip password provide by users. |






<a name="bytebase-v1-ExportResponse"></a>

### ExportResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| content | [bytes](#bytes) |  | The export file content. |






<a name="bytebase-v1-GenerateRestoreSQLRequest"></a>

### GenerateRestoreSQLRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the instance name to execute the query against. Format: instances/{instance}/databases/{databaseName} |
| statement | [string](#string) |  | The original SQL statement. |
| backup_data_source | [string](#string) |  | The data source to restore from. Format: instances/{instance}/databases/{databaseName}, for general engines. Or instances/{instance}/databases/{databaseName}/schemas/{schemaName}, for PG only. |
| backup_table | [string](#string) |  | The backup table name. |






<a name="bytebase-v1-GenerateRestoreSQLResponse"></a>

### GenerateRestoreSQLResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| statement | [string](#string) |  | The restore SQL statement. |






<a name="bytebase-v1-ParseMyBatisMapperRequest"></a>

### ParseMyBatisMapperRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| content | [bytes](#bytes) |  |  |






<a name="bytebase-v1-ParseMyBatisMapperResponse"></a>

### ParseMyBatisMapperResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| statements | [string](#string) | repeated |  |






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






<a name="bytebase-v1-QueryHistory"></a>

### QueryHistory



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name for the query history. Format: queryHistories/{uid} |
| database | [string](#string) |  | The database name to execute the query. Format: instances/{instance}/databases/{databaseName} |
| creator | [string](#string) |  |  |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| statement | [string](#string) |  |  |
| error | [string](#string) | optional |  |
| duration | [google.protobuf.Duration](#google-protobuf-Duration) |  |  |
| type | [QueryHistory.Type](#bytebase-v1-QueryHistory-Type) |  |  |






<a name="bytebase-v1-QueryRequest"></a>

### QueryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the instance name to execute the query against. Format: instances/{instance}/databases/{databaseName} |
| connection_database | [string](#string) |  | **Deprecated.**  |
| statement | [string](#string) |  | The SQL statement to execute. |
| limit | [int32](#int32) |  | The maximum number of rows to return. |
| timeout | [google.protobuf.Duration](#google-protobuf-Duration) | optional | The timeout for the request. |
| data_source_id | [string](#string) |  | The id of data source. It is used for querying admin data source even if the instance has read-only data sources. Or it can be used to query a specific read-only data source. |
| explain | [bool](#bool) |  | Explain the statement. |






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






<a name="bytebase-v1-SearchQueryHistoriesRequest"></a>

### SearchQueryHistoriesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Not used. The maximum number of histories to return. The service may return fewer than this value. If unspecified, at most 100 history entries will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListQueryHistory` call. Provide this to retrieve the subsequent page. |
| filter | [string](#string) |  | filter is the filter to apply on the search query history, follow the [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) syntax. Support filter by: - database, for example: database = &#34;instances/{instance}/databases/{database}&#34; - instance, for example: instance = &#34;instance/{instance}&#34; - type, for example: type = &#34;QUERY&#34; |






<a name="bytebase-v1-SearchQueryHistoriesResponse"></a>

### SearchQueryHistoriesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| query_histories | [QueryHistory](#bytebase-v1-QueryHistory) | repeated | The list of history. |
| next_page_token | [string](#string) |  | A token to retrieve next page of history. Pass this value in the page_token field in the subsequent call to `ListQueryHistory` method to retrieve the next page of history. |






<a name="bytebase-v1-StringifyMetadataRequest"></a>

### StringifyMetadataRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  |  |
| engine | [Engine](#bytebase-v1-Engine) |  | The database engine of the schema string. |
| classification_from_config | [bool](#bool) |  | If false, we will build the raw common by classification in database config. |






<a name="bytebase-v1-StringifyMetadataResponse"></a>

### StringifyMetadataResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  |  |





 


<a name="bytebase-v1-Advice-Status"></a>

### Advice.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 | Unspecified. |
| SUCCESS | 1 |  |
| WARNING | 2 |  |
| ERROR | 3 |  |



<a name="bytebase-v1-CheckRequest-ChangeType"></a>

### CheckRequest.ChangeType


| Name | Number | Description |
| ---- | ------ | ----------- |
| CHANGE_TYPE_UNSPECIFIED | 0 |  |
| DDL | 1 |  |
| DDL_GHOST | 2 |  |
| DML | 3 |  |



<a name="bytebase-v1-QueryHistory-Type"></a>

### QueryHistory.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| QUERY | 1 |  |
| EXPORT | 2 |  |


 

 


<a name="bytebase-v1-SQLService"></a>

### SQLService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Query | [QueryRequest](#bytebase-v1-QueryRequest) | [QueryResponse](#bytebase-v1-QueryResponse) |  |
| Execute | [ExecuteRequest](#bytebase-v1-ExecuteRequest) | [ExecuteResponse](#bytebase-v1-ExecuteResponse) |  |
| AdminExecute | [AdminExecuteRequest](#bytebase-v1-AdminExecuteRequest) stream | [AdminExecuteResponse](#bytebase-v1-AdminExecuteResponse) stream |  |
| SearchQueryHistories | [SearchQueryHistoriesRequest](#bytebase-v1-SearchQueryHistoriesRequest) | [SearchQueryHistoriesResponse](#bytebase-v1-SearchQueryHistoriesResponse) |  |
| Export | [ExportRequest](#bytebase-v1-ExportRequest) | [ExportResponse](#bytebase-v1-ExportResponse) |  |
| DifferPreview | [DifferPreviewRequest](#bytebase-v1-DifferPreviewRequest) | [DifferPreviewResponse](#bytebase-v1-DifferPreviewResponse) |  |
| Check | [CheckRequest](#bytebase-v1-CheckRequest) | [CheckResponse](#bytebase-v1-CheckResponse) |  |
| ParseMyBatisMapper | [ParseMyBatisMapperRequest](#bytebase-v1-ParseMyBatisMapperRequest) | [ParseMyBatisMapperResponse](#bytebase-v1-ParseMyBatisMapperResponse) |  |
| Pretty | [PrettyRequest](#bytebase-v1-PrettyRequest) | [PrettyResponse](#bytebase-v1-PrettyResponse) |  |
| StringifyMetadata | [StringifyMetadataRequest](#bytebase-v1-StringifyMetadataRequest) | [StringifyMetadataResponse](#bytebase-v1-StringifyMetadataResponse) |  |
| GenerateRestoreSQL | [GenerateRestoreSQLRequest](#bytebase-v1-GenerateRestoreSQLRequest) | [GenerateRestoreSQLResponse](#bytebase-v1-GenerateRestoreSQLResponse) |  |

 



<a name="v1_user_group-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/user_group.proto



<a name="bytebase-v1-CreateUserGroupRequest"></a>

### CreateUserGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [UserGroup](#bytebase-v1-UserGroup) |  | The group to create. |






<a name="bytebase-v1-DeleteUserGroupRequest"></a>

### DeleteUserGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the group to delete. Format: groups/{email} |






<a name="bytebase-v1-GetUserGroupRequest"></a>

### GetUserGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the group to retrieve. Format: groups/{email} |






<a name="bytebase-v1-ListUserGroupsRequest"></a>

### ListUserGroupsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of groups to return. The service may return fewer than this value. If unspecified, at most 50 groups will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListUsers` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListUsers` must match the call that provided the page token. |






<a name="bytebase-v1-ListUserGroupsResponse"></a>

### ListUserGroupsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| groups | [UserGroup](#bytebase-v1-UserGroup) | repeated | The groups from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-UpdateUserGroupRequest"></a>

### UpdateUserGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [UserGroup](#bytebase-v1-UserGroup) |  | The group to update.

The group&#39;s `name` field is used to identify the group to update. Format: groups/{email} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






<a name="bytebase-v1-UserGroup"></a>

### UserGroup



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the group to retrieve. Format: groups/{email} |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| creator | [string](#string) |  | The name for the creator. Format: users/hello@world.com |
| members | [UserGroupMember](#bytebase-v1-UserGroupMember) | repeated |  |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the group was created. |






<a name="bytebase-v1-UserGroupMember"></a>

### UserGroupMember



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| member | [string](#string) |  | Member is the principal who belong to this user group.

Format: users/hello@world.com |
| role | [UserGroupMember.Role](#bytebase-v1-UserGroupMember-Role) |  |  |





 


<a name="bytebase-v1-UserGroupMember-Role"></a>

### UserGroupMember.Role


| Name | Number | Description |
| ---- | ------ | ----------- |
| ROLE_UNSPECIFIED | 0 |  |
| OWNER | 1 |  |
| MEMBER | 2 |  |


 

 


<a name="bytebase-v1-UserGroupService"></a>

### UserGroupService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetUserGroup | [GetUserGroupRequest](#bytebase-v1-GetUserGroupRequest) | [UserGroup](#bytebase-v1-UserGroup) |  |
| ListUserGroups | [ListUserGroupsRequest](#bytebase-v1-ListUserGroupsRequest) | [ListUserGroupsResponse](#bytebase-v1-ListUserGroupsResponse) |  |
| CreateUserGroup | [CreateUserGroupRequest](#bytebase-v1-CreateUserGroupRequest) | [UserGroup](#bytebase-v1-UserGroup) |  |
| UpdateUserGroup | [UpdateUserGroupRequest](#bytebase-v1-UpdateUserGroupRequest) | [UserGroup](#bytebase-v1-UserGroup) |  |
| DeleteUserGroup | [DeleteUserGroupRequest](#bytebase-v1-DeleteUserGroupRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="v1_vcs_connector_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/vcs_connector_service.proto



<a name="bytebase-v1-CreateVCSConnectorRequest"></a>

### CreateVCSConnectorRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this vcsConnector will be created. Format: projects/{project} |
| vcs_connector | [VCSConnector](#bytebase-v1-VCSConnector) |  | The vcsConnector to create. |
| vcs_connector_id | [string](#string) |  | The ID to use for the vcsConnector, which will become the final component of the vcsConnector&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |






<a name="bytebase-v1-DeleteVCSConnectorRequest"></a>

### DeleteVCSConnectorRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the vcsConnector to delete. Format: projects/{project}/vcsConnectors/{vcsConnector} |






<a name="bytebase-v1-GetVCSConnectorRequest"></a>

### GetVCSConnectorRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the vcsConnector to retrieve. Format: projects/{project}/vcsConnectors/{vcsConnector} |






<a name="bytebase-v1-ListVCSConnectorsRequest"></a>

### ListVCSConnectorsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of vcsConnectors. Format: projects/{project} |
| page_size | [int32](#int32) |  | The maximum number of databases to return. The service may return fewer than this value. If unspecified, at most 50 databases will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListDatabases` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListDatabases` must match the call that provided the page token. |






<a name="bytebase-v1-ListVCSConnectorsResponse"></a>

### ListVCSConnectorsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vcs_connectors | [VCSConnector](#bytebase-v1-VCSConnector) | repeated | The vcsConnectors from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-UpdateVCSConnectorRequest"></a>

### UpdateVCSConnectorRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vcs_connector | [VCSConnector](#bytebase-v1-VCSConnector) |  | The vcsConnector to update.

The vcsConnector&#39;s `name` field is used to identify the vcsConnector to update. Format: projects/{project}/vcsConnectors/{vcsConnector} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to be updated. |






<a name="bytebase-v1-VCSConnector"></a>

### VCSConnector



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the vcsConnector resource. Canonical parent is project. Format: projects/{project}/vcsConnectors/{vcsConnector} |
| title | [string](#string) |  | The title of the vcs connector. |
| creator | [string](#string) |  | The creator of the vcsConnector. Format: users/{email} |
| updater | [string](#string) |  | The updater of the vcsConnector. Format: users/{email} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The create time of the vcsConnector. |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The last update time of the vcsConnector. |
| vcs_provider | [string](#string) |  | The name of the VCS. Format: vcsProviders/{vcsProvider} |
| external_id | [string](#string) |  | The reposition external id in target VCS. |
| base_directory | [string](#string) |  | The root directory where Bytebase observes the file change. If empty, then it observes the entire repository. |
| branch | [string](#string) |  | The branch Bytebase listens to for changes. For example: main. |
| full_path | [string](#string) |  | TODO(d): move these to create VCS connector API. The full_path of the repository. For example: bytebase/sample. |
| web_url | [string](#string) |  | The web url of the repository. For axample: https://gitlab.bytebase.com/bytebase/sample. |
| database_group | [string](#string) |  | Apply changes to the database group. Optional, if not set, will apply changes to all databases in the project. Format: projects/{project}/databaseGroups/{databaseGroup} |





 

 

 


<a name="bytebase-v1-VCSConnectorService"></a>

### VCSConnectorService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateVCSConnector | [CreateVCSConnectorRequest](#bytebase-v1-CreateVCSConnectorRequest) | [VCSConnector](#bytebase-v1-VCSConnector) |  |
| GetVCSConnector | [GetVCSConnectorRequest](#bytebase-v1-GetVCSConnectorRequest) | [VCSConnector](#bytebase-v1-VCSConnector) |  |
| ListVCSConnectors | [ListVCSConnectorsRequest](#bytebase-v1-ListVCSConnectorsRequest) | [ListVCSConnectorsResponse](#bytebase-v1-ListVCSConnectorsResponse) |  |
| UpdateVCSConnector | [UpdateVCSConnectorRequest](#bytebase-v1-UpdateVCSConnectorRequest) | [VCSConnector](#bytebase-v1-VCSConnector) |  |
| DeleteVCSConnector | [DeleteVCSConnectorRequest](#bytebase-v1-DeleteVCSConnectorRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="v1_vcs_provider_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/vcs_provider_service.proto



<a name="bytebase-v1-CreateVCSProviderRequest"></a>

### CreateVCSProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vcs_provider | [VCSProvider](#bytebase-v1-VCSProvider) |  |  |
| vcs_provider_id | [string](#string) |  | The ID to use for the VCS provider, which will become the final component of the VCS provider&#39;s name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |






<a name="bytebase-v1-DeleteVCSProviderRequest"></a>

### DeleteVCSProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the vcs provider to delete. Format: vcsProviders/{vcsProvider} |






<a name="bytebase-v1-GetVCSProviderRequest"></a>

### GetVCSProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the vcs provider to retrieve. Format: vcsProviders/{vcsProvider} |






<a name="bytebase-v1-ListVCSConnectorsInProviderRequest"></a>

### ListVCSConnectorsInProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the vcs provider to retrieve the linked projects. Format: vcsProviders/{vcsProvider} |






<a name="bytebase-v1-ListVCSConnectorsInProviderResponse"></a>

### ListVCSConnectorsInProviderResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vcs_connectors | [VCSConnector](#bytebase-v1-VCSConnector) | repeated | The vcsConnectors from the specified request. |






<a name="bytebase-v1-ListVCSProvidersRequest"></a>

### ListVCSProvidersRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Not used. The maximum number of vcs provider to return. The service may return fewer than this value. If unspecified, at most 100 vcs provider will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListVCSProviders` call. Provide this to retrieve the subsequent page. |






<a name="bytebase-v1-ListVCSProvidersResponse"></a>

### ListVCSProvidersResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vcs_providers | [VCSProvider](#bytebase-v1-VCSProvider) | repeated | The list of vcs providers. |
| next_page_token | [string](#string) |  | Not used. A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-SearchVCSProviderRepositoriesRequest"></a>

### SearchVCSProviderRepositoriesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the vcs provider to retrieve the vcs provider repositories. Format: vcsProviders/{vcsProvider} |






<a name="bytebase-v1-SearchVCSProviderRepositoriesResponse"></a>

### SearchVCSProviderRepositoriesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| repositories | [VCSRepository](#bytebase-v1-VCSRepository) | repeated | The list of repositories in vcs provider. |






<a name="bytebase-v1-UpdateVCSProviderRequest"></a>

### UpdateVCSProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vcs_provider | [VCSProvider](#bytebase-v1-VCSProvider) |  |  |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to be updated. |






<a name="bytebase-v1-VCSProvider"></a>

### VCSProvider



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the vcs provider. Format: vcsProviders/{vcsProvider} |
| title | [string](#string) |  | The title of the vcs provider. It is used to display in the UI. Specified by the client. |
| type | [VCSType](#bytebase-v1-VCSType) |  |  |
| url | [string](#string) |  | The url of the vcs provider. Specified by the client. For example: github.com, gitlab.com, gitlab.bytebase.com. |
| access_token | [string](#string) |  | The access token of the vcs provider. |






<a name="bytebase-v1-VCSRepository"></a>

### VCSRepository



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | The id of the repository in vcs provider. e.g. In GitLab, this is the corresponding project id. e.g. 123 |
| title | [string](#string) |  | The title of the repository in vcs provider. e.g. sample-project |
| full_path | [string](#string) |  | The full_path of the repository in vcs provider. e.g. bytebase/sample-project |
| web_url | [string](#string) |  | Web url of the repository in vcs provider. e.g. http://gitlab.bytebase.com/bytebase/sample-project |





 

 

 


<a name="bytebase-v1-VCSProviderService"></a>

### VCSProviderService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetVCSProvider | [GetVCSProviderRequest](#bytebase-v1-GetVCSProviderRequest) | [VCSProvider](#bytebase-v1-VCSProvider) |  |
| ListVCSProviders | [ListVCSProvidersRequest](#bytebase-v1-ListVCSProvidersRequest) | [ListVCSProvidersResponse](#bytebase-v1-ListVCSProvidersResponse) |  |
| CreateVCSProvider | [CreateVCSProviderRequest](#bytebase-v1-CreateVCSProviderRequest) | [VCSProvider](#bytebase-v1-VCSProvider) |  |
| UpdateVCSProvider | [UpdateVCSProviderRequest](#bytebase-v1-UpdateVCSProviderRequest) | [VCSProvider](#bytebase-v1-VCSProvider) |  |
| DeleteVCSProvider | [DeleteVCSProviderRequest](#bytebase-v1-DeleteVCSProviderRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| SearchVCSProviderRepositories | [SearchVCSProviderRepositoriesRequest](#bytebase-v1-SearchVCSProviderRepositoriesRequest) | [SearchVCSProviderRepositoriesResponse](#bytebase-v1-SearchVCSProviderRepositoriesResponse) |  |
| ListVCSConnectorsInProvider | [ListVCSConnectorsInProviderRequest](#bytebase-v1-ListVCSConnectorsInProviderRequest) | [ListVCSConnectorsInProviderResponse](#bytebase-v1-ListVCSConnectorsInProviderResponse) |  |

 



<a name="v1_worksheet_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/worksheet_service.proto



<a name="bytebase-v1-CreateWorksheetRequest"></a>

### CreateWorksheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| worksheet | [Worksheet](#bytebase-v1-Worksheet) |  | The worksheet to create. |






<a name="bytebase-v1-DeleteWorksheetRequest"></a>

### DeleteWorksheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the worksheet to delete. Format: worksheets/{worksheet} |






<a name="bytebase-v1-GetWorksheetRequest"></a>

### GetWorksheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the worksheet to retrieve. Format: worksheets/{worksheet} |






<a name="bytebase-v1-SearchWorksheetsRequest"></a>

### SearchWorksheetsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filter | [string](#string) |  | To filter the search result. Format: only support the following spec for now: - `creator = users/{email}`, `creator != users/{email}` - `starred = true`, `starred = false`. - `visibility = &#34;VISIBILITY_PRIVATE&#34;`, `visibility = &#34;VISIBILITY_PROJECT_READ | VISIBILITY_PROJECT_WRITE&#34;`, etc. Not support empty filter for now. |
| page_size | [int32](#int32) |  | Not used. The maximum number of worksheets to return. The service may return fewer than this value. If unspecified, at most 50 worksheets will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `SearchWorksheets` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `SearchWorksheets` must match the call that provided the page token. |






<a name="bytebase-v1-SearchWorksheetsResponse"></a>

### SearchWorksheetsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| worksheets | [Worksheet](#bytebase-v1-Worksheet) | repeated | The worksheets that matched the search criteria. |
| next_page_token | [string](#string) |  | Not used. A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-UpdateWorksheetOrganizerRequest"></a>

### UpdateWorksheetOrganizerRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organizer | [WorksheetOrganizer](#bytebase-v1-WorksheetOrganizer) |  | The organizer to update.

The organizer&#39;s `worksheet` field is used to identify the worksheet. Format: worksheets/{worksheet} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to be updated. Fields are specified relative to the worksheet organizer. Only support update the following fields for now: - `starred` |






<a name="bytebase-v1-UpdateWorksheetRequest"></a>

### UpdateWorksheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| worksheet | [Worksheet](#bytebase-v1-Worksheet) |  | The worksheet to update.

The worksheet&#39;s `name` field is used to identify the worksheet to update. Format: worksheets/{worksheet} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to be updated. Fields are specified relative to the worksheet. (e.g. `title`, `statement`; *not* `worksheet.title` or `worksheet.statement`) Only support update the following fields for now: - `title` - `statement` - `starred` - `visibility` |






<a name="bytebase-v1-Worksheet"></a>

### Worksheet



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the worksheet resource, generated by the server. Canonical parent is project. Format: worksheets/{worksheet} |
| project | [string](#string) |  | The project resource name. Format: projects/{project} |
| database | [string](#string) |  | The database resource name. Format: instances/{instance}/databases/{database} If the database parent doesn&#39;t exist, the database field is empty. |
| title | [string](#string) |  | The title of the worksheet. |
| creator | [string](#string) |  | The creator of the Worksheet. Format: users/{email} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The create time of the worksheet. |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The last update time of the worksheet. |
| content | [bytes](#bytes) |  | The content of the worksheet. By default, it will be cut off in SearchWorksheet() method. If it doesn&#39;t match the `content_size`, you can use GetWorksheet() request to retrieve the full content. |
| content_size | [int64](#int64) |  | content_size is the full size of the content, may not match the size of the `content` field. |
| visibility | [Worksheet.Visibility](#bytebase-v1-Worksheet-Visibility) |  |  |
| starred | [bool](#bool) |  | starred indicates whether the worksheet is starred by the current authenticated user. |






<a name="bytebase-v1-WorksheetOrganizer"></a>

### WorksheetOrganizer



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| worksheet | [string](#string) |  | The name of the worksheet. Format: worksheets/{worksheet} |
| starred | [bool](#bool) |  | starred means if the worksheet is starred. |





 


<a name="bytebase-v1-Worksheet-Visibility"></a>

### Worksheet.Visibility


| Name | Number | Description |
| ---- | ------ | ----------- |
| VISIBILITY_UNSPECIFIED | 0 |  |
| VISIBILITY_PROJECT_READ | 1 | Read access in project scope, worksheet OWNER/DBA and project OWNER can read/write, other project members can read. |
| VISIBILITY_PROJECT_WRITE | 2 | Write access in project scope, worksheet OWNER/DBA and all members in the project can write the worksheet. |
| VISIBILITY_PRIVATE | 3 | Private, only worksheet OWNER can read/write. |


 

 


<a name="bytebase-v1-WorksheetService"></a>

### WorksheetService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateWorksheet | [CreateWorksheetRequest](#bytebase-v1-CreateWorksheetRequest) | [Worksheet](#bytebase-v1-Worksheet) |  |
| GetWorksheet | [GetWorksheetRequest](#bytebase-v1-GetWorksheetRequest) | [Worksheet](#bytebase-v1-Worksheet) |  |
| SearchWorksheets | [SearchWorksheetsRequest](#bytebase-v1-SearchWorksheetsRequest) | [SearchWorksheetsResponse](#bytebase-v1-SearchWorksheetsResponse) |  |
| UpdateWorksheet | [UpdateWorksheetRequest](#bytebase-v1-UpdateWorksheetRequest) | [Worksheet](#bytebase-v1-Worksheet) |  |
| UpdateWorksheetOrganizer | [UpdateWorksheetOrganizerRequest](#bytebase-v1-UpdateWorksheetOrganizerRequest) | [WorksheetOrganizer](#bytebase-v1-WorksheetOrganizer) |  |
| DeleteWorksheet | [DeleteWorksheetRequest](#bytebase-v1-DeleteWorksheetRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="v1_workspace_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/workspace_service.proto


 

 

 


<a name="bytebase-v1-WorkspaceService"></a>

### WorkspaceService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetIamPolicy | [GetIamPolicyRequest](#bytebase-v1-GetIamPolicyRequest) | [IamPolicy](#bytebase-v1-IamPolicy) |  |
| SetIamPolicy | [SetIamPolicyRequest](#bytebase-v1-SetIamPolicyRequest) | [IamPolicy](#bytebase-v1-IamPolicy) |  |

 



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

