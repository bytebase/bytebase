# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [store/common.proto](#store_common-proto)
    - [DatabaseLabel](#bytebase-store-DatabaseLabel)
    - [PageToken](#bytebase-store-PageToken)
    - [Position](#bytebase-store-Position)
    - [Range](#bytebase-store-Range)
  
    - [Engine](#bytebase-store-Engine)
    - [ExportFormat](#bytebase-store-ExportFormat)
    - [MaskingLevel](#bytebase-store-MaskingLevel)
    - [VCSType](#bytebase-store-VCSType)
  
- [store/advice.proto](#store_advice-proto)
    - [Advice](#bytebase-store-Advice)
  
    - [Advice.Status](#bytebase-store-Advice-Status)
  
- [store/anomaly.proto](#store_anomaly-proto)
    - [AnomalyConnectionPayload](#bytebase-store-AnomalyConnectionPayload)
    - [AnomalyDatabaseSchemaDriftPayload](#bytebase-store-AnomalyDatabaseSchemaDriftPayload)
  
- [store/approval.proto](#store_approval-proto)
    - [ApprovalFlow](#bytebase-store-ApprovalFlow)
    - [ApprovalNode](#bytebase-store-ApprovalNode)
    - [ApprovalStep](#bytebase-store-ApprovalStep)
    - [ApprovalTemplate](#bytebase-store-ApprovalTemplate)
    - [IssuePayloadApproval](#bytebase-store-IssuePayloadApproval)
    - [IssuePayloadApproval.Approver](#bytebase-store-IssuePayloadApproval-Approver)
  
    - [ApprovalNode.GroupValue](#bytebase-store-ApprovalNode-GroupValue)
    - [ApprovalNode.Type](#bytebase-store-ApprovalNode-Type)
    - [ApprovalStep.Type](#bytebase-store-ApprovalStep-Type)
    - [IssuePayloadApproval.Approver.Status](#bytebase-store-IssuePayloadApproval-Approver-Status)
    - [IssuePayloadApproval.RiskLevel](#bytebase-store-IssuePayloadApproval-RiskLevel)
  
- [store/audit_log.proto](#store_audit_log-proto)
    - [AuditLog](#bytebase-store-AuditLog)
    - [RequestMetadata](#bytebase-store-RequestMetadata)
  
    - [AuditLog.Severity](#bytebase-store-AuditLog-Severity)
  
- [store/database.proto](#store_database-proto)
    - [CheckConstraintMetadata](#bytebase-store-CheckConstraintMetadata)
    - [ColumnConfig](#bytebase-store-ColumnConfig)
    - [ColumnConfig.LabelsEntry](#bytebase-store-ColumnConfig-LabelsEntry)
    - [ColumnMetadata](#bytebase-store-ColumnMetadata)
    - [DatabaseConfig](#bytebase-store-DatabaseConfig)
    - [DatabaseMetadata](#bytebase-store-DatabaseMetadata)
    - [DatabaseMetadata.LabelsEntry](#bytebase-store-DatabaseMetadata-LabelsEntry)
    - [DatabaseSchemaMetadata](#bytebase-store-DatabaseSchemaMetadata)
    - [DependentColumn](#bytebase-store-DependentColumn)
    - [EventMetadata](#bytebase-store-EventMetadata)
    - [ExtensionMetadata](#bytebase-store-ExtensionMetadata)
    - [ExternalTableMetadata](#bytebase-store-ExternalTableMetadata)
    - [ForeignKeyMetadata](#bytebase-store-ForeignKeyMetadata)
    - [FunctionConfig](#bytebase-store-FunctionConfig)
    - [FunctionMetadata](#bytebase-store-FunctionMetadata)
    - [GenerationMetadata](#bytebase-store-GenerationMetadata)
    - [IndexMetadata](#bytebase-store-IndexMetadata)
    - [InstanceRoleMetadata](#bytebase-store-InstanceRoleMetadata)
    - [LinkedDatabaseMetadata](#bytebase-store-LinkedDatabaseMetadata)
    - [MaterializedViewMetadata](#bytebase-store-MaterializedViewMetadata)
    - [PackageMetadata](#bytebase-store-PackageMetadata)
    - [ProcedureConfig](#bytebase-store-ProcedureConfig)
    - [ProcedureMetadata](#bytebase-store-ProcedureMetadata)
    - [SchemaConfig](#bytebase-store-SchemaConfig)
    - [SchemaMetadata](#bytebase-store-SchemaMetadata)
    - [SecretItem](#bytebase-store-SecretItem)
    - [Secrets](#bytebase-store-Secrets)
    - [SequenceMetadata](#bytebase-store-SequenceMetadata)
    - [StreamMetadata](#bytebase-store-StreamMetadata)
    - [TableConfig](#bytebase-store-TableConfig)
    - [TableMetadata](#bytebase-store-TableMetadata)
    - [TablePartitionMetadata](#bytebase-store-TablePartitionMetadata)
    - [TaskMetadata](#bytebase-store-TaskMetadata)
    - [TriggerMetadata](#bytebase-store-TriggerMetadata)
    - [ViewConfig](#bytebase-store-ViewConfig)
    - [ViewMetadata](#bytebase-store-ViewMetadata)
  
    - [GenerationMetadata.Type](#bytebase-store-GenerationMetadata-Type)
    - [StreamMetadata.Mode](#bytebase-store-StreamMetadata-Mode)
    - [StreamMetadata.Type](#bytebase-store-StreamMetadata-Type)
    - [TablePartitionMetadata.Type](#bytebase-store-TablePartitionMetadata-Type)
    - [TaskMetadata.State](#bytebase-store-TaskMetadata-State)
  
- [store/branch.proto](#store_branch-proto)
    - [BranchConfig](#bytebase-store-BranchConfig)
    - [BranchSnapshot](#bytebase-store-BranchSnapshot)
  
- [store/changelist.proto](#store_changelist-proto)
    - [Changelist](#bytebase-store-Changelist)
    - [Changelist.Change](#bytebase-store-Changelist-Change)
  
- [store/instance_change_history.proto](#store_instance_change_history-proto)
    - [ChangedResourceDatabase](#bytebase-store-ChangedResourceDatabase)
    - [ChangedResourceFunction](#bytebase-store-ChangedResourceFunction)
    - [ChangedResourceProcedure](#bytebase-store-ChangedResourceProcedure)
    - [ChangedResourceSchema](#bytebase-store-ChangedResourceSchema)
    - [ChangedResourceTable](#bytebase-store-ChangedResourceTable)
    - [ChangedResourceView](#bytebase-store-ChangedResourceView)
    - [ChangedResources](#bytebase-store-ChangedResources)
    - [InstanceChangeHistoryPayload](#bytebase-store-InstanceChangeHistoryPayload)
  
- [store/changelog.proto](#store_changelog-proto)
    - [ChangelogPayload](#bytebase-store-ChangelogPayload)
    - [ChangelogRevision](#bytebase-store-ChangelogRevision)
    - [ChangelogTask](#bytebase-store-ChangelogTask)
  
    - [ChangelogRevision.Op](#bytebase-store-ChangelogRevision-Op)
    - [ChangelogTask.Status](#bytebase-store-ChangelogTask-Status)
    - [ChangelogTask.Type](#bytebase-store-ChangelogTask-Type)
  
- [store/data_source.proto](#store_data_source-proto)
    - [DataSourceExternalSecret](#bytebase-store-DataSourceExternalSecret)
    - [DataSourceExternalSecret.AppRoleAuthOption](#bytebase-store-DataSourceExternalSecret-AppRoleAuthOption)
    - [DataSourceOptions](#bytebase-store-DataSourceOptions)
    - [DataSourceOptions.Address](#bytebase-store-DataSourceOptions-Address)
    - [KerberosConfig](#bytebase-store-KerberosConfig)
    - [SASLConfig](#bytebase-store-SASLConfig)
  
    - [DataSourceExternalSecret.AppRoleAuthOption.SecretType](#bytebase-store-DataSourceExternalSecret-AppRoleAuthOption-SecretType)
    - [DataSourceExternalSecret.AuthType](#bytebase-store-DataSourceExternalSecret-AuthType)
    - [DataSourceExternalSecret.SecretType](#bytebase-store-DataSourceExternalSecret-SecretType)
    - [DataSourceOptions.AuthenticationType](#bytebase-store-DataSourceOptions-AuthenticationType)
    - [DataSourceOptions.RedisType](#bytebase-store-DataSourceOptions-RedisType)
  
- [store/db_group.proto](#store_db_group-proto)
    - [DatabaseGroupPayload](#bytebase-store-DatabaseGroupPayload)
  
- [store/export_archive.proto](#store_export_archive-proto)
    - [ExportArchivePayload](#bytebase-store-ExportArchivePayload)
  
- [store/group.proto](#store_group-proto)
    - [GroupMember](#bytebase-store-GroupMember)
    - [GroupPayload](#bytebase-store-GroupPayload)
  
    - [GroupMember.Role](#bytebase-store-GroupMember-Role)
  
- [store/idp.proto](#store_idp-proto)
    - [FieldMapping](#bytebase-store-FieldMapping)
    - [IdentityProviderConfig](#bytebase-store-IdentityProviderConfig)
    - [IdentityProviderUserInfo](#bytebase-store-IdentityProviderUserInfo)
    - [LDAPIdentityProviderConfig](#bytebase-store-LDAPIdentityProviderConfig)
    - [OAuth2IdentityProviderConfig](#bytebase-store-OAuth2IdentityProviderConfig)
    - [OIDCIdentityProviderConfig](#bytebase-store-OIDCIdentityProviderConfig)
  
    - [IdentityProviderType](#bytebase-store-IdentityProviderType)
    - [OAuth2AuthStyle](#bytebase-store-OAuth2AuthStyle)
  
- [store/instance.proto](#store_instance-proto)
    - [InstanceMetadata](#bytebase-store-InstanceMetadata)
    - [InstanceOptions](#bytebase-store-InstanceOptions)
    - [InstanceRole](#bytebase-store-InstanceRole)
  
- [store/issue.proto](#store_issue-proto)
    - [GrantRequest](#bytebase-store-GrantRequest)
    - [IssuePayload](#bytebase-store-IssuePayload)
  
- [store/issue_comment.proto](#store_issue_comment-proto)
    - [IssueCommentPayload](#bytebase-store-IssueCommentPayload)
    - [IssueCommentPayload.Approval](#bytebase-store-IssueCommentPayload-Approval)
    - [IssueCommentPayload.IssueUpdate](#bytebase-store-IssueCommentPayload-IssueUpdate)
    - [IssueCommentPayload.StageEnd](#bytebase-store-IssueCommentPayload-StageEnd)
    - [IssueCommentPayload.TaskPriorBackup](#bytebase-store-IssueCommentPayload-TaskPriorBackup)
    - [IssueCommentPayload.TaskPriorBackup.Table](#bytebase-store-IssueCommentPayload-TaskPriorBackup-Table)
    - [IssueCommentPayload.TaskUpdate](#bytebase-store-IssueCommentPayload-TaskUpdate)
  
    - [IssueCommentPayload.Approval.Status](#bytebase-store-IssueCommentPayload-Approval-Status)
    - [IssueCommentPayload.IssueUpdate.IssueStatus](#bytebase-store-IssueCommentPayload-IssueUpdate-IssueStatus)
    - [IssueCommentPayload.TaskUpdate.Status](#bytebase-store-IssueCommentPayload-TaskUpdate-Status)
  
- [store/plan_check_run.proto](#store_plan_check_run-proto)
    - [PlanCheckRunConfig](#bytebase-store-PlanCheckRunConfig)
    - [PlanCheckRunConfig.GhostFlagsEntry](#bytebase-store-PlanCheckRunConfig-GhostFlagsEntry)
    - [PlanCheckRunResult](#bytebase-store-PlanCheckRunResult)
    - [PlanCheckRunResult.Result](#bytebase-store-PlanCheckRunResult-Result)
    - [PlanCheckRunResult.Result.SqlReviewReport](#bytebase-store-PlanCheckRunResult-Result-SqlReviewReport)
    - [PlanCheckRunResult.Result.SqlSummaryReport](#bytebase-store-PlanCheckRunResult-Result-SqlSummaryReport)
    - [PreUpdateBackupDetail](#bytebase-store-PreUpdateBackupDetail)
  
    - [PlanCheckRunConfig.ChangeDatabaseType](#bytebase-store-PlanCheckRunConfig-ChangeDatabaseType)
    - [PlanCheckRunResult.Result.Status](#bytebase-store-PlanCheckRunResult-Result-Status)
  
- [store/plan.proto](#store_plan-proto)
    - [PlanConfig](#bytebase-store-PlanConfig)
    - [PlanConfig.ChangeDatabaseConfig](#bytebase-store-PlanConfig-ChangeDatabaseConfig)
    - [PlanConfig.ChangeDatabaseConfig.GhostFlagsEntry](#bytebase-store-PlanConfig-ChangeDatabaseConfig-GhostFlagsEntry)
    - [PlanConfig.CreateDatabaseConfig](#bytebase-store-PlanConfig-CreateDatabaseConfig)
    - [PlanConfig.CreateDatabaseConfig.LabelsEntry](#bytebase-store-PlanConfig-CreateDatabaseConfig-LabelsEntry)
    - [PlanConfig.ExportDataConfig](#bytebase-store-PlanConfig-ExportDataConfig)
    - [PlanConfig.ReleaseSource](#bytebase-store-PlanConfig-ReleaseSource)
    - [PlanConfig.Spec](#bytebase-store-PlanConfig-Spec)
    - [PlanConfig.SpecReleaseSource](#bytebase-store-PlanConfig-SpecReleaseSource)
    - [PlanConfig.Step](#bytebase-store-PlanConfig-Step)
    - [PlanConfig.VCSSource](#bytebase-store-PlanConfig-VCSSource)
  
    - [PlanConfig.ChangeDatabaseConfig.Type](#bytebase-store-PlanConfig-ChangeDatabaseConfig-Type)
  
- [store/policy.proto](#store_policy-proto)
    - [Binding](#bytebase-store-Binding)
    - [DataSourceQueryPolicy](#bytebase-store-DataSourceQueryPolicy)
    - [DisableCopyDataPolicy](#bytebase-store-DisableCopyDataPolicy)
    - [EnvironmentTierPolicy](#bytebase-store-EnvironmentTierPolicy)
    - [ExportDataPolicy](#bytebase-store-ExportDataPolicy)
    - [IamPolicy](#bytebase-store-IamPolicy)
    - [MaskData](#bytebase-store-MaskData)
    - [MaskingExceptionPolicy](#bytebase-store-MaskingExceptionPolicy)
    - [MaskingExceptionPolicy.MaskingException](#bytebase-store-MaskingExceptionPolicy-MaskingException)
    - [MaskingPolicy](#bytebase-store-MaskingPolicy)
    - [MaskingRulePolicy](#bytebase-store-MaskingRulePolicy)
    - [MaskingRulePolicy.MaskingRule](#bytebase-store-MaskingRulePolicy-MaskingRule)
    - [RestrictIssueCreationForSQLReviewPolicy](#bytebase-store-RestrictIssueCreationForSQLReviewPolicy)
    - [RolloutPolicy](#bytebase-store-RolloutPolicy)
    - [SQLReviewRule](#bytebase-store-SQLReviewRule)
    - [SlowQueryPolicy](#bytebase-store-SlowQueryPolicy)
    - [TagPolicy](#bytebase-store-TagPolicy)
    - [TagPolicy.TagsEntry](#bytebase-store-TagPolicy-TagsEntry)
  
    - [DataSourceQueryPolicy.Restriction](#bytebase-store-DataSourceQueryPolicy-Restriction)
    - [EnvironmentTierPolicy.EnvironmentTier](#bytebase-store-EnvironmentTierPolicy-EnvironmentTier)
    - [MaskingExceptionPolicy.MaskingException.Action](#bytebase-store-MaskingExceptionPolicy-MaskingException-Action)
    - [SQLReviewRuleLevel](#bytebase-store-SQLReviewRuleLevel)
  
- [store/project.proto](#store_project-proto)
    - [Label](#bytebase-store-Label)
    - [Project](#bytebase-store-Project)
  
- [store/project_webhook.proto](#store_project_webhook-proto)
    - [ProjectWebhookPayload](#bytebase-store-ProjectWebhookPayload)
  
- [store/query_history.proto](#store_query_history-proto)
    - [QueryHistoryPayload](#bytebase-store-QueryHistoryPayload)
  
- [store/release.proto](#store_release-proto)
    - [ReleasePayload](#bytebase-store-ReleasePayload)
    - [ReleasePayload.File](#bytebase-store-ReleasePayload-File)
    - [ReleasePayload.VCSSource](#bytebase-store-ReleasePayload-VCSSource)
  
    - [ReleaseFileType](#bytebase-store-ReleaseFileType)
  
- [store/review_config.proto](#store_review_config-proto)
    - [ReviewConfigPayload](#bytebase-store-ReviewConfigPayload)
  
- [store/revision.proto](#store_revision-proto)
    - [RevisionPayload](#bytebase-store-RevisionPayload)
  
- [store/role.proto](#store_role-proto)
    - [RolePermissions](#bytebase-store-RolePermissions)
  
- [store/setting.proto](#store_setting-proto)
    - [AgentPluginSetting](#bytebase-store-AgentPluginSetting)
    - [Announcement](#bytebase-store-Announcement)
    - [AppIMSetting](#bytebase-store-AppIMSetting)
    - [AppIMSetting.Feishu](#bytebase-store-AppIMSetting-Feishu)
    - [AppIMSetting.Slack](#bytebase-store-AppIMSetting-Slack)
    - [AppIMSetting.Wecom](#bytebase-store-AppIMSetting-Wecom)
    - [DataClassificationSetting](#bytebase-store-DataClassificationSetting)
    - [DataClassificationSetting.DataClassificationConfig](#bytebase-store-DataClassificationSetting-DataClassificationConfig)
    - [DataClassificationSetting.DataClassificationConfig.ClassificationEntry](#bytebase-store-DataClassificationSetting-DataClassificationConfig-ClassificationEntry)
    - [DataClassificationSetting.DataClassificationConfig.DataClassification](#bytebase-store-DataClassificationSetting-DataClassificationConfig-DataClassification)
    - [DataClassificationSetting.DataClassificationConfig.Level](#bytebase-store-DataClassificationSetting-DataClassificationConfig-Level)
    - [ExternalApprovalPayload](#bytebase-store-ExternalApprovalPayload)
    - [ExternalApprovalSetting](#bytebase-store-ExternalApprovalSetting)
    - [ExternalApprovalSetting.Node](#bytebase-store-ExternalApprovalSetting-Node)
    - [MaskingAlgorithmSetting](#bytebase-store-MaskingAlgorithmSetting)
    - [MaskingAlgorithmSetting.Algorithm](#bytebase-store-MaskingAlgorithmSetting-Algorithm)
    - [MaskingAlgorithmSetting.Algorithm.FullMask](#bytebase-store-MaskingAlgorithmSetting-Algorithm-FullMask)
    - [MaskingAlgorithmSetting.Algorithm.InnerOuterMask](#bytebase-store-MaskingAlgorithmSetting-Algorithm-InnerOuterMask)
    - [MaskingAlgorithmSetting.Algorithm.MD5Mask](#bytebase-store-MaskingAlgorithmSetting-Algorithm-MD5Mask)
    - [MaskingAlgorithmSetting.Algorithm.RangeMask](#bytebase-store-MaskingAlgorithmSetting-Algorithm-RangeMask)
    - [MaskingAlgorithmSetting.Algorithm.RangeMask.Slice](#bytebase-store-MaskingAlgorithmSetting-Algorithm-RangeMask-Slice)
    - [MaximumSQLResultSizeSetting](#bytebase-store-MaximumSQLResultSizeSetting)
    - [PasswordRestrictionSetting](#bytebase-store-PasswordRestrictionSetting)
    - [SCIMSetting](#bytebase-store-SCIMSetting)
    - [SMTPMailDeliverySetting](#bytebase-store-SMTPMailDeliverySetting)
    - [SchemaTemplateSetting](#bytebase-store-SchemaTemplateSetting)
    - [SchemaTemplateSetting.ColumnType](#bytebase-store-SchemaTemplateSetting-ColumnType)
    - [SchemaTemplateSetting.FieldTemplate](#bytebase-store-SchemaTemplateSetting-FieldTemplate)
    - [SchemaTemplateSetting.TableTemplate](#bytebase-store-SchemaTemplateSetting-TableTemplate)
    - [SemanticTypeSetting](#bytebase-store-SemanticTypeSetting)
    - [SemanticTypeSetting.SemanticType](#bytebase-store-SemanticTypeSetting-SemanticType)
    - [WorkspaceApprovalSetting](#bytebase-store-WorkspaceApprovalSetting)
    - [WorkspaceApprovalSetting.Rule](#bytebase-store-WorkspaceApprovalSetting-Rule)
    - [WorkspaceProfileSetting](#bytebase-store-WorkspaceProfileSetting)
  
    - [Announcement.AlertLevel](#bytebase-store-Announcement-AlertLevel)
    - [DatabaseChangeMode](#bytebase-store-DatabaseChangeMode)
    - [MaskingAlgorithmSetting.Algorithm.InnerOuterMask.MaskType](#bytebase-store-MaskingAlgorithmSetting-Algorithm-InnerOuterMask-MaskType)
    - [SMTPMailDeliverySetting.Authentication](#bytebase-store-SMTPMailDeliverySetting-Authentication)
    - [SMTPMailDeliverySetting.Encryption](#bytebase-store-SMTPMailDeliverySetting-Encryption)
  
- [store/sheet.proto](#store_sheet-proto)
    - [SheetCommand](#bytebase-store-SheetCommand)
    - [SheetPayload](#bytebase-store-SheetPayload)
  
- [store/slow_query.proto](#store_slow_query-proto)
    - [SlowQueryDetails](#bytebase-store-SlowQueryDetails)
    - [SlowQueryStatistics](#bytebase-store-SlowQueryStatistics)
    - [SlowQueryStatisticsItem](#bytebase-store-SlowQueryStatisticsItem)
  
- [store/task.proto](#store_task-proto)
    - [TaskDatabaseCreatePayload](#bytebase-store-TaskDatabaseCreatePayload)
    - [TaskDatabaseDataExportPayload](#bytebase-store-TaskDatabaseDataExportPayload)
    - [TaskDatabaseUpdatePayload](#bytebase-store-TaskDatabaseUpdatePayload)
    - [TaskDatabaseUpdatePayload.FlagsEntry](#bytebase-store-TaskDatabaseUpdatePayload-FlagsEntry)
    - [TaskReleaseSource](#bytebase-store-TaskReleaseSource)
  
- [store/task_run.proto](#store_task_run-proto)
    - [PriorBackupDetail](#bytebase-store-PriorBackupDetail)
    - [PriorBackupDetail.Item](#bytebase-store-PriorBackupDetail-Item)
    - [PriorBackupDetail.Item.Table](#bytebase-store-PriorBackupDetail-Item-Table)
    - [SchedulerInfo](#bytebase-store-SchedulerInfo)
    - [SchedulerInfo.WaitingCause](#bytebase-store-SchedulerInfo-WaitingCause)
    - [TaskRunResult](#bytebase-store-TaskRunResult)
    - [TaskRunResult.Position](#bytebase-store-TaskRunResult-Position)
  
- [store/task_run_log.proto](#store_task_run_log-proto)
    - [TaskRunLog](#bytebase-store-TaskRunLog)
    - [TaskRunLog.CommandExecute](#bytebase-store-TaskRunLog-CommandExecute)
    - [TaskRunLog.CommandResponse](#bytebase-store-TaskRunLog-CommandResponse)
    - [TaskRunLog.DatabaseSyncEnd](#bytebase-store-TaskRunLog-DatabaseSyncEnd)
    - [TaskRunLog.DatabaseSyncStart](#bytebase-store-TaskRunLog-DatabaseSyncStart)
    - [TaskRunLog.PriorBackupEnd](#bytebase-store-TaskRunLog-PriorBackupEnd)
    - [TaskRunLog.PriorBackupStart](#bytebase-store-TaskRunLog-PriorBackupStart)
    - [TaskRunLog.SchemaDumpEnd](#bytebase-store-TaskRunLog-SchemaDumpEnd)
    - [TaskRunLog.SchemaDumpStart](#bytebase-store-TaskRunLog-SchemaDumpStart)
    - [TaskRunLog.TaskRunStatusUpdate](#bytebase-store-TaskRunLog-TaskRunStatusUpdate)
    - [TaskRunLog.TransactionControl](#bytebase-store-TaskRunLog-TransactionControl)
  
    - [TaskRunLog.TaskRunStatusUpdate.Status](#bytebase-store-TaskRunLog-TaskRunStatusUpdate-Status)
    - [TaskRunLog.TransactionControl.Type](#bytebase-store-TaskRunLog-TransactionControl-Type)
    - [TaskRunLog.Type](#bytebase-store-TaskRunLog-Type)
  
- [store/user.proto](#store_user-proto)
    - [MFAConfig](#bytebase-store-MFAConfig)
    - [UserProfile](#bytebase-store-UserProfile)
  
- [store/vcs.proto](#store_vcs-proto)
    - [VCSConnector](#bytebase-store-VCSConnector)
  
- [Scalar Value Types](#scalar-value-types)



<a name="store_common-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/common.proto



<a name="bytebase-store-DatabaseLabel"></a>

### DatabaseLabel



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-store-PageToken"></a>

### PageToken
Used internally for obfuscating the page token.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| limit | [int32](#int32) |  |  |
| offset | [int32](#int32) |  |  |






<a name="bytebase-store-Position"></a>

### Position



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| line | [int32](#int32) |  |  |
| column | [int32](#int32) |  |  |






<a name="bytebase-store-Range"></a>

### Range



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start | [int32](#int32) |  |  |
| end | [int32](#int32) |  |  |





 


<a name="bytebase-store-Engine"></a>

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
| COCKROACHDB | 25 |  |



<a name="bytebase-store-ExportFormat"></a>

### ExportFormat


| Name | Number | Description |
| ---- | ------ | ----------- |
| FORMAT_UNSPECIFIED | 0 |  |
| CSV | 1 |  |
| JSON | 2 |  |
| SQL | 3 |  |
| XLSX | 4 |  |



<a name="bytebase-store-MaskingLevel"></a>

### MaskingLevel


| Name | Number | Description |
| ---- | ------ | ----------- |
| MASKING_LEVEL_UNSPECIFIED | 0 |  |
| NONE | 1 |  |
| PARTIAL | 2 |  |
| FULL | 3 |  |



<a name="bytebase-store-VCSType"></a>

### VCSType


| Name | Number | Description |
| ---- | ------ | ----------- |
| VCS_TYPE_UNSPECIFIED | 0 |  |
| GITHUB | 1 | GitHub type. Using for GitHub community edition(ce). |
| GITLAB | 2 | GitLab type. Using for GitLab community edition(ce) and enterprise edition(ee). |
| BITBUCKET | 3 | BitBucket type. Using for BitBucket cloud or BitBucket server. |
| AZURE_DEVOPS | 4 | Azure DevOps. Using for Azure DevOps GitOps workflow. |


 

 

 



<a name="store_advice-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/advice.proto



<a name="bytebase-store-Advice"></a>

### Advice



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [Advice.Status](#bytebase-store-Advice-Status) |  | The advice status. |
| code | [int32](#int32) |  | The advice code. |
| title | [string](#string) |  | The advice title. |
| content | [string](#string) |  | The advice content. |
| detail | [string](#string) |  | The advice detail. |
| start_position | [Position](#bytebase-store-Position) |  | 1-based positions of the sql statment. |
| end_position | [Position](#bytebase-store-Position) |  |  |





 


<a name="bytebase-store-Advice-Status"></a>

### Advice.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 | Unspecified. |
| SUCCESS | 1 |  |
| WARNING | 2 |  |
| ERROR | 3 |  |


 

 

 



<a name="store_anomaly-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/anomaly.proto



<a name="bytebase-store-AnomalyConnectionPayload"></a>

### AnomalyConnectionPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| detail | [string](#string) |  | Connection failure detail |






<a name="bytebase-store-AnomalyDatabaseSchemaDriftPayload"></a>

### AnomalyDatabaseSchemaDriftPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [string](#string) |  | The schema version corresponds to the expected schema |
| expect | [string](#string) |  | The expected latest schema stored in the migration history table |
| actual | [string](#string) |  | The actual schema dumped from the database |





 

 

 

 



<a name="store_approval-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/approval.proto



<a name="bytebase-store-ApprovalFlow"></a>

### ApprovalFlow



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| steps | [ApprovalStep](#bytebase-store-ApprovalStep) | repeated |  |






<a name="bytebase-store-ApprovalNode"></a>

### ApprovalNode



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [ApprovalNode.Type](#bytebase-store-ApprovalNode-Type) |  |  |
| group_value | [ApprovalNode.GroupValue](#bytebase-store-ApprovalNode-GroupValue) |  |  |
| role | [string](#string) |  | Format: roles/{role} |
| external_node_id | [string](#string) |  |  |






<a name="bytebase-store-ApprovalStep"></a>

### ApprovalStep



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [ApprovalStep.Type](#bytebase-store-ApprovalStep-Type) |  |  |
| nodes | [ApprovalNode](#bytebase-store-ApprovalNode) | repeated |  |






<a name="bytebase-store-ApprovalTemplate"></a>

### ApprovalTemplate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| flow | [ApprovalFlow](#bytebase-store-ApprovalFlow) |  |  |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| creator_id | [int32](#int32) |  |  |






<a name="bytebase-store-IssuePayloadApproval"></a>

### IssuePayloadApproval
IssuePayloadApproval is a part of the payload of an issue.
IssuePayloadApproval records the approval template used and the approval history.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| approval_templates | [ApprovalTemplate](#bytebase-store-ApprovalTemplate) | repeated |  |
| approvers | [IssuePayloadApproval.Approver](#bytebase-store-IssuePayloadApproval-Approver) | repeated |  |
| approval_finding_done | [bool](#bool) |  | If the value is `false`, it means that the backend is still finding matching approval templates. If `true`, other fields are available. |
| approval_finding_error | [string](#string) |  |  |
| risk_level | [IssuePayloadApproval.RiskLevel](#bytebase-store-IssuePayloadApproval-RiskLevel) |  |  |






<a name="bytebase-store-IssuePayloadApproval-Approver"></a>

### IssuePayloadApproval.Approver



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [IssuePayloadApproval.Approver.Status](#bytebase-store-IssuePayloadApproval-Approver-Status) |  | The new status. |
| principal_id | [int32](#int32) |  | The principal id of the approver. |





 


<a name="bytebase-store-ApprovalNode-GroupValue"></a>

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



<a name="bytebase-store-ApprovalNode-Type"></a>

### ApprovalNode.Type
Type of the ApprovalNode.
type determines who should approve this node.
ANY_IN_GROUP means the ApprovalNode can be approved by an user from our predefined user group.
See GroupValue below for the predefined user groups.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| ANY_IN_GROUP | 1 |  |



<a name="bytebase-store-ApprovalStep-Type"></a>

### ApprovalStep.Type
Type of the ApprovalStep
ALL means every node must be approved to proceed.
ANY means approving any node will proceed.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| ALL | 1 |  |
| ANY | 2 |  |



<a name="bytebase-store-IssuePayloadApproval-Approver-Status"></a>

### IssuePayloadApproval.Approver.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| APPROVED | 2 |  |
| REJECTED | 3 |  |



<a name="bytebase-store-IssuePayloadApproval-RiskLevel"></a>

### IssuePayloadApproval.RiskLevel


| Name | Number | Description |
| ---- | ------ | ----------- |
| RISK_LEVEL_UNSPECIFIED | 0 |  |
| LOW | 1 |  |
| MODERATE | 2 |  |
| HIGH | 3 |  |


 

 

 



<a name="store_audit_log-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/audit_log.proto



<a name="bytebase-store-AuditLog"></a>

### AuditLog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The project or workspace the audit log belongs to. Formats: - projects/{project} - workspaces/{workspace} |
| method | [string](#string) |  | e.g. /bytebase.v1.SQLService/Query |
| resource | [string](#string) |  | resource name projects/{project} |
| user | [string](#string) |  | Format: users/{userUID}. |
| severity | [AuditLog.Severity](#bytebase-store-AuditLog-Severity) |  |  |
| request | [string](#string) |  | Marshalled request. |
| response | [string](#string) |  | Marshalled response. Some fields are omitted because they are too large or contain sensitive information. |
| status | [google.rpc.Status](#google-rpc-Status) |  |  |
| service_data | [google.protobuf.Any](#google-protobuf-Any) |  | service-specific data about the request, response, and other activities. |
| request_metadata | [RequestMetadata](#bytebase-store-RequestMetadata) |  | Metadata about the operation. |






<a name="bytebase-store-RequestMetadata"></a>

### RequestMetadata
Metadata about the request.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| caller_ip | [string](#string) |  | The IP address of the caller. |
| caller_supplied_user_agent | [string](#string) |  | The user agent of the caller. This information is not authenticated and should be treated accordingly. |





 


<a name="bytebase-store-AuditLog-Severity"></a>

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


 

 

 



<a name="store_database-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/database.proto



<a name="bytebase-store-CheckConstraintMetadata"></a>

### CheckConstraintMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a check constraint. |
| expression | [string](#string) |  | The expression is the expression of a check constraint. |






<a name="bytebase-store-ColumnConfig"></a>

### ColumnConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a column. |
| semantic_type_id | [string](#string) |  |  |
| labels | [ColumnConfig.LabelsEntry](#bytebase-store-ColumnConfig-LabelsEntry) | repeated | The user labels for a column. |
| classification_id | [string](#string) |  |  |






<a name="bytebase-store-ColumnConfig-LabelsEntry"></a>

### ColumnConfig.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-store-ColumnMetadata"></a>

### ColumnMetadata
ColumnMetadata is the metadata for columns.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a column. |
| position | [int32](#int32) |  | The position is the position in columns. |
| default | [google.protobuf.StringValue](#google-protobuf-StringValue) |  | The default is the default of a column. Use google.protobuf.StringValue to distinguish between an empty string default value or no default. |
| default_null | [bool](#bool) |  |  |
| default_expression | [string](#string) |  |  |
| on_update | [string](#string) |  | The on_update is the on update action of a column. For MySQL like databases, it&#39;s only supported for TIMESTAMP columns with CURRENT_TIMESTAMP as on update value. |
| nullable | [bool](#bool) |  | The nullable is the nullable of a column. |
| type | [string](#string) |  | The type is the type of a column. |
| character_set | [string](#string) |  | The character_set is the character_set of a column. |
| collation | [string](#string) |  | The collation is the collation of a column. |
| comment | [string](#string) |  | The comment is the comment of a column. classification and user_comment is parsed from the comment. |
| user_comment | [string](#string) |  | The user_comment is the user comment of a table parsed from the comment. |
| generation | [GenerationMetadata](#bytebase-store-GenerationMetadata) |  | The generation is for generated columns. |






<a name="bytebase-store-DatabaseConfig"></a>

### DatabaseConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| schema_configs | [SchemaConfig](#bytebase-store-SchemaConfig) | repeated | The schema_configs is the list of configs for schemas in a database. |






<a name="bytebase-store-DatabaseMetadata"></a>

### DatabaseMetadata
DatabaseMetadata is the metadata for databases.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| labels | [DatabaseMetadata.LabelsEntry](#bytebase-store-DatabaseMetadata-LabelsEntry) | repeated |  |
| last_sync_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| backup_available | [bool](#bool) |  |  |






<a name="bytebase-store-DatabaseMetadata-LabelsEntry"></a>

### DatabaseMetadata.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-store-DatabaseSchemaMetadata"></a>

### DatabaseSchemaMetadata
DatabaseSchemaMetadata is the schema metadata for databases.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| schemas | [SchemaMetadata](#bytebase-store-SchemaMetadata) | repeated | The schemas is the list of schemas in a database. |
| character_set | [string](#string) |  | The character_set is the character set of a database. |
| collation | [string](#string) |  | The collation is the collation of a database. |
| extensions | [ExtensionMetadata](#bytebase-store-ExtensionMetadata) | repeated | The extensions is the list of extensions in a database. |
| datashare | [bool](#bool) |  | The database belongs to a datashare. |
| service_name | [string](#string) |  | The service name of the database. It&#39;s the Oracle specific concept. |
| linked_databases | [LinkedDatabaseMetadata](#bytebase-store-LinkedDatabaseMetadata) | repeated |  |
| owner | [string](#string) |  |  |






<a name="bytebase-store-DependentColumn"></a>

### DependentColumn
DependentColumn is the metadata for dependent columns.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  | The schema is the schema of a reference column. |
| table | [string](#string) |  | The table is the table of a reference column. |
| column | [string](#string) |  | The column is the name of a reference column. |






<a name="bytebase-store-EventMetadata"></a>

### EventMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the event. |
| definition | [string](#string) |  | The schedule of the event. |
| time_zone | [string](#string) |  | The time zone of the event. |
| sql_mode | [string](#string) |  |  |
| character_set_client | [string](#string) |  |  |
| collation_connection | [string](#string) |  |  |






<a name="bytebase-store-ExtensionMetadata"></a>

### ExtensionMetadata
ExtensionMetadata is the metadata for extensions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of an extension. |
| schema | [string](#string) |  | The schema is the extension that is installed to. But the extension usage is not limited to the schema. |
| version | [string](#string) |  | The version is the version of an extension. |
| description | [string](#string) |  | The description is the description of an extension. |






<a name="bytebase-store-ExternalTableMetadata"></a>

### ExternalTableMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a external table. |
| external_server_name | [string](#string) |  | The external_server_name is the name of the external server. |
| external_database_name | [string](#string) |  | The external_database_name is the name of the external database. |
| columns | [ColumnMetadata](#bytebase-store-ColumnMetadata) | repeated | The columns is the ordered list of columns in a foreign table. |






<a name="bytebase-store-ForeignKeyMetadata"></a>

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






<a name="bytebase-store-FunctionConfig"></a>

### FunctionConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a function. |
| updater | [string](#string) |  | The last updater of the function in branch. Format: users/{userUID}. |
| source_branch | [string](#string) |  | The last change come from branch. Format: projcets/{project}/branches/{branch} |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the function is updated in branch. |






<a name="bytebase-store-FunctionMetadata"></a>

### FunctionMetadata
FunctionMetadata is the metadata for functions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a function. |
| definition | [string](#string) |  | The definition is the definition of a function. |
| signature | [string](#string) |  | The signature is the name with the number and type of input arguments the function takes. |
| character_set_client | [string](#string) |  | MySQL specific metadata. |
| collation_connection | [string](#string) |  |  |
| database_collation | [string](#string) |  |  |
| sql_mode | [string](#string) |  |  |






<a name="bytebase-store-GenerationMetadata"></a>

### GenerationMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [GenerationMetadata.Type](#bytebase-store-GenerationMetadata-Type) |  |  |
| expression | [string](#string) |  |  |






<a name="bytebase-store-IndexMetadata"></a>

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






<a name="bytebase-store-InstanceRoleMetadata"></a>

### InstanceRoleMetadata
InstanceRoleMetadata is the message for instance role.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The role name. It&#39;s unique within the instance. |
| grant | [string](#string) |  | The grant display string on the instance. It&#39;s generated by database engine. |






<a name="bytebase-store-LinkedDatabaseMetadata"></a>

### LinkedDatabaseMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| username | [string](#string) |  |  |
| host | [string](#string) |  |  |






<a name="bytebase-store-MaterializedViewMetadata"></a>

### MaterializedViewMetadata
MaterializedViewMetadata is the metadata for materialized views.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a view. |
| definition | [string](#string) |  | The definition is the definition of a view. |
| comment | [string](#string) |  | The comment is the comment of a view. |
| dependent_columns | [DependentColumn](#bytebase-store-DependentColumn) | repeated | The dependent_columns is the list of dependent columns of a view. |






<a name="bytebase-store-PackageMetadata"></a>

### PackageMetadata
PackageMetadata is the metadata for packages.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a package. |
| definition | [string](#string) |  | The definition is the definition of a package. |






<a name="bytebase-store-ProcedureConfig"></a>

### ProcedureConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a procedure. |
| updater | [string](#string) |  | The last updater of the procedure in branch. Format: users/{userUID}. |
| source_branch | [string](#string) |  | The last change come from branch. Format: projcets/{project}/branches/{branch} |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the procedure is updated in branch. |






<a name="bytebase-store-ProcedureMetadata"></a>

### ProcedureMetadata
ProcedureMetadata is the metadata for procedures.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a procedure. |
| definition | [string](#string) |  | The definition is the definition of a procedure. |
| signature | [string](#string) |  | The signature is the name with the number and type of input arguments the function takes. |
| character_set_client | [string](#string) |  | MySQL specific metadata. |
| collation_connection | [string](#string) |  |  |
| database_collation | [string](#string) |  |  |
| sql_mode | [string](#string) |  |  |






<a name="bytebase-store-SchemaConfig"></a>

### SchemaConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the schema name. It is an empty string for databases without such concept such as MySQL. |
| table_configs | [TableConfig](#bytebase-store-TableConfig) | repeated | The table_configs is the list of configs for tables in a schema. |
| function_configs | [FunctionConfig](#bytebase-store-FunctionConfig) | repeated |  |
| procedure_configs | [ProcedureConfig](#bytebase-store-ProcedureConfig) | repeated |  |
| view_configs | [ViewConfig](#bytebase-store-ViewConfig) | repeated |  |






<a name="bytebase-store-SchemaMetadata"></a>

### SchemaMetadata
SchemaMetadata is the metadata for schemas.
This is the concept of schema in Postgres, but it&#39;s a no-op for MySQL.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the schema name. It is an empty string for databases without such concept such as MySQL. |
| tables | [TableMetadata](#bytebase-store-TableMetadata) | repeated | The tables is the list of tables in a schema. |
| external_tables | [ExternalTableMetadata](#bytebase-store-ExternalTableMetadata) | repeated | The external_tables is the list of external tables in a schema. |
| views | [ViewMetadata](#bytebase-store-ViewMetadata) | repeated | The views is the list of views in a schema. |
| functions | [FunctionMetadata](#bytebase-store-FunctionMetadata) | repeated | The functions is the list of functions in a schema. |
| procedures | [ProcedureMetadata](#bytebase-store-ProcedureMetadata) | repeated | The procedures is the list of procedures in a schema. |
| streams | [StreamMetadata](#bytebase-store-StreamMetadata) | repeated | The streams is the list of streams in a schema, currently, only used for Snowflake. |
| tasks | [TaskMetadata](#bytebase-store-TaskMetadata) | repeated | The routines is the list of routines in a schema, currently, only used for Snowflake. |
| materialized_views | [MaterializedViewMetadata](#bytebase-store-MaterializedViewMetadata) | repeated | The materialized_views is the list of materialized views in a schema. |
| sequences | [SequenceMetadata](#bytebase-store-SequenceMetadata) | repeated | The sequences is the list of sequences in a schema. |
| packages | [PackageMetadata](#bytebase-store-PackageMetadata) | repeated | The packages is the list of packages in a schema. |
| owner | [string](#string) |  |  |
| triggers | [TriggerMetadata](#bytebase-store-TriggerMetadata) | repeated | The triggers is the list of triggers in a schema, triggers are sorted by table_name, event, timing, action_order. |
| events | [EventMetadata](#bytebase-store-EventMetadata) | repeated |  |






<a name="bytebase-store-SecretItem"></a>

### SecretItem



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of the secret. |
| value | [string](#string) |  | The value is the value of the secret. |
| description | [string](#string) |  | The description is the description of the secret. |






<a name="bytebase-store-Secrets"></a>

### Secrets



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| items | [SecretItem](#bytebase-store-SecretItem) | repeated | The list of secrets. |






<a name="bytebase-store-SequenceMetadata"></a>

### SequenceMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of a sequence. |
| data_type | [string](#string) |  | The data type of a sequence. |
| start | [string](#string) |  | The start value of a sequence. |
| min_value | [string](#string) |  | The minimum value of a sequence. |
| max_value | [string](#string) |  | The maximum value of a sequence. |
| increment | [string](#string) |  | Increment value of a sequence. |
| cycle | [bool](#bool) |  | Cycle is whether the sequence cycles. |
| cache_size | [string](#string) |  | Cache size of a sequence. |
| last_value | [string](#string) |  | Last value of a sequence. |






<a name="bytebase-store-StreamMetadata"></a>

### StreamMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a stream. |
| table_name | [string](#string) |  | The table_name is the name of the table/view that the stream is created on. |
| owner | [string](#string) |  | The owner of the stream. |
| comment | [string](#string) |  | The comment of the stream. |
| type | [StreamMetadata.Type](#bytebase-store-StreamMetadata-Type) |  | The type of the stream. |
| stale | [bool](#bool) |  | Indicates whether the stream was last read before the `stale_after` time. |
| mode | [StreamMetadata.Mode](#bytebase-store-StreamMetadata-Mode) |  | The mode of the stream. |
| definition | [string](#string) |  | The definition of the stream. |






<a name="bytebase-store-TableConfig"></a>

### TableConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a table. |
| column_configs | [ColumnConfig](#bytebase-store-ColumnConfig) | repeated | The column_configs is the ordered list of configs for columns in a table. |
| classification_id | [string](#string) |  |  |
| updater | [string](#string) |  | The last updater of the table in branch. Format: users/{userUID}. |
| source_branch | [string](#string) |  | The last change come from branch. Format: projcets/{project}/branches/{branch} |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the table is updated in branch. |






<a name="bytebase-store-TableMetadata"></a>

### TableMetadata
TableMetadata is the metadata for tables.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a table. |
| columns | [ColumnMetadata](#bytebase-store-ColumnMetadata) | repeated | The columns is the ordered list of columns in a table. |
| indexes | [IndexMetadata](#bytebase-store-IndexMetadata) | repeated | The indexes is the list of indexes in a table. |
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
| foreign_keys | [ForeignKeyMetadata](#bytebase-store-ForeignKeyMetadata) | repeated | The foreign_keys is the list of foreign keys in a table. |
| partitions | [TablePartitionMetadata](#bytebase-store-TablePartitionMetadata) | repeated | The partitions is the list of partitions in a table. |
| check_constraints | [CheckConstraintMetadata](#bytebase-store-CheckConstraintMetadata) | repeated | The check_constraints is the list of check constraints in a table. |
| owner | [string](#string) |  |  |






<a name="bytebase-store-TablePartitionMetadata"></a>

### TablePartitionMetadata
TablePartitionMetadata is the metadata for table partitions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a table partition. |
| type | [TablePartitionMetadata.Type](#bytebase-store-TablePartitionMetadata-Type) |  | The type of a table partition. |
| expression | [string](#string) |  | The expression is the expression of a table partition. For PostgreSQL, the expression is the text of {FOR VALUES partition_bound_spec}, see https://www.postgresql.org/docs/current/sql-createtable.html. For MySQL, the expression is the `expr` or `column_list` of the following syntax. PARTITION BY { [LINEAR] HASH(expr) | [LINEAR] KEY [ALGORITHM={1 | 2}] (column_list) | RANGE{(expr) | COLUMNS(column_list)} | LIST{(expr) | COLUMNS(column_list)} }. |
| value | [string](#string) |  | The value is the value of a table partition. For MySQL, the value is for RANGE and LIST partition types, - For a RANGE partition, it contains the value set in the partition&#39;s VALUES LESS THAN clause, which can be either an integer or MAXVALUE. - For a LIST partition, this column contains the values defined in the partition&#39;s VALUES IN clause, which is a list of comma-separated integer values. - For others, it&#39;s an empty string. |
| use_default | [string](#string) |  | The use_default is whether the users use the default partition, it stores the different value for different database engines. For MySQL, it&#39;s [INT] type, 0 means not use default partition, otherwise, it&#39;s equals to number in syntax [SUB]PARTITION {number}. |
| subpartitions | [TablePartitionMetadata](#bytebase-store-TablePartitionMetadata) | repeated | The subpartitions is the list of subpartitions in a table partition. |






<a name="bytebase-store-TaskMetadata"></a>

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
| state | [TaskMetadata.State](#bytebase-store-TaskMetadata-State) |  | The state of the task. |
| condition | [string](#string) |  | The condition of the task. |
| definition | [string](#string) |  | The definition of the task. |






<a name="bytebase-store-TriggerMetadata"></a>

### TriggerMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of the trigger. |
| table_name | [string](#string) |  | The table_name is the name of the table/view that the trigger is created on. |
| event | [string](#string) |  | The event is the event of the trigger, such as INSERT, UPDATE, DELETE, TRUNCATE. |
| timing | [string](#string) |  | The timing is the timing of the trigger, such as BEFORE, AFTER. |
| body | [string](#string) |  | The body is the body of the trigger. |
| sql_mode | [string](#string) |  |  |
| character_set_client | [string](#string) |  |  |
| collation_connection | [string](#string) |  |  |






<a name="bytebase-store-ViewConfig"></a>

### ViewConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a view. |
| updater | [string](#string) |  | The last updater of the view in branch. Format: users/{userUID}. |
| source_branch | [string](#string) |  | The last change come from branch. Format: projcets/{project}/branches/{branch} |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the view is updated in branch. |






<a name="bytebase-store-ViewMetadata"></a>

### ViewMetadata
ViewMetadata is the metadata for views.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a view. |
| definition | [string](#string) |  | The definition is the definition of a view. |
| comment | [string](#string) |  | The comment is the comment of a view. |
| dependent_columns | [DependentColumn](#bytebase-store-DependentColumn) | repeated | The dependent_columns is the list of dependent columns of a view. |
| columns | [ColumnMetadata](#bytebase-store-ColumnMetadata) | repeated | The columns is the ordered list of columns in a table. |





 


<a name="bytebase-store-GenerationMetadata-Type"></a>

### GenerationMetadata.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| TYPE_VIRTUAL | 1 |  |
| TYPE_STORED | 2 |  |



<a name="bytebase-store-StreamMetadata-Mode"></a>

### StreamMetadata.Mode


| Name | Number | Description |
| ---- | ------ | ----------- |
| MODE_UNSPECIFIED | 0 |  |
| MODE_DEFAULT | 1 |  |
| MODE_APPEND_ONLY | 2 |  |
| MODE_INSERT_ONLY | 3 |  |



<a name="bytebase-store-StreamMetadata-Type"></a>

### StreamMetadata.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| TYPE_DELTA | 1 |  |



<a name="bytebase-store-TablePartitionMetadata-Type"></a>

### TablePartitionMetadata.Type
Type is the type of a table partition, some database engines may not
support all types. Only avilable for the following database engines now:
MySQL: RANGE, RANGE COLUMNS, LIST, LIST COLUMNS, HASH, LINEAR HASH, KEY,
LINEAR_KEY
(https://dev.mysql.com/doc/refman/8.0/en/partitioning-types.html) TiDB:
RANGE, RANGE COLUMNS, LIST, LIST COLUMNS, HASH, KEY PostgreSQL: RANGE,
LIST, HASH (https://www.postgresql.org/docs/current/ddl-partitioning.html)

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



<a name="bytebase-store-TaskMetadata-State"></a>

### TaskMetadata.State


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATE_UNSPECIFIED | 0 |  |
| STATE_STARTED | 1 |  |
| STATE_SUSPENDED | 2 |  |


 

 

 



<a name="store_branch-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/branch.proto



<a name="bytebase-store-BranchConfig"></a>

### BranchConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| source_database | [string](#string) |  | The name of source database. Optional. Example: instances/instance-id/databases/database-name. |
| source_branch | [string](#string) |  | The name of the source branch. Optional. Example: projects/project-id/branches/branch-id. |






<a name="bytebase-store-BranchSnapshot"></a>

### BranchSnapshot



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| metadata | [DatabaseSchemaMetadata](#bytebase-store-DatabaseSchemaMetadata) |  |  |
| database_config | [DatabaseConfig](#bytebase-store-DatabaseConfig) |  |  |





 

 

 

 



<a name="store_changelist-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/changelist.proto



<a name="bytebase-store-Changelist"></a>

### Changelist



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| description | [string](#string) |  |  |
| changes | [Changelist.Change](#bytebase-store-Changelist-Change) | repeated |  |






<a name="bytebase-store-Changelist-Change"></a>

### Changelist.Change



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheet | [string](#string) |  | The name of a sheet. |
| source | [string](#string) |  | The source of origin. 1) change history: instances/{instance}/databases/{database}/changeHistories/{changeHistory}. 2) branch: projects/{project}/branches/{branch}. 3) raw SQL if empty. |
| version | [string](#string) |  | The migration version for a change. |





 

 

 

 



<a name="store_instance_change_history-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/instance_change_history.proto



<a name="bytebase-store-ChangedResourceDatabase"></a>

### ChangedResourceDatabase



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| schemas | [ChangedResourceSchema](#bytebase-store-ChangedResourceSchema) | repeated |  |






<a name="bytebase-store-ChangedResourceFunction"></a>

### ChangedResourceFunction



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| ranges | [Range](#bytebase-store-Range) | repeated | The ranges of sub-strings correspond to the statements on the sheet. |






<a name="bytebase-store-ChangedResourceProcedure"></a>

### ChangedResourceProcedure



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| ranges | [Range](#bytebase-store-Range) | repeated | The ranges of sub-strings correspond to the statements on the sheet. |






<a name="bytebase-store-ChangedResourceSchema"></a>

### ChangedResourceSchema



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| tables | [ChangedResourceTable](#bytebase-store-ChangedResourceTable) | repeated |  |
| views | [ChangedResourceView](#bytebase-store-ChangedResourceView) | repeated |  |
| functions | [ChangedResourceFunction](#bytebase-store-ChangedResourceFunction) | repeated |  |
| procedures | [ChangedResourceProcedure](#bytebase-store-ChangedResourceProcedure) | repeated |  |






<a name="bytebase-store-ChangedResourceTable"></a>

### ChangedResourceTable



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| table_rows | [int64](#int64) |  | estimated row count of the table |
| ranges | [Range](#bytebase-store-Range) | repeated | The ranges of sub-strings correspond to the statements on the sheet. |






<a name="bytebase-store-ChangedResourceView"></a>

### ChangedResourceView



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| ranges | [Range](#bytebase-store-Range) | repeated | The ranges of sub-strings correspond to the statements on the sheet. |






<a name="bytebase-store-ChangedResources"></a>

### ChangedResources



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [ChangedResourceDatabase](#bytebase-store-ChangedResourceDatabase) | repeated |  |






<a name="bytebase-store-InstanceChangeHistoryPayload"></a>

### InstanceChangeHistoryPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| changed_resources | [ChangedResources](#bytebase-store-ChangedResources) |  |  |





 

 

 

 



<a name="store_changelog-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/changelog.proto



<a name="bytebase-store-ChangelogPayload"></a>

### ChangelogPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| task | [ChangelogTask](#bytebase-store-ChangelogTask) |  |  |
| revision | [ChangelogRevision](#bytebase-store-ChangelogRevision) |  |  |






<a name="bytebase-store-ChangelogRevision"></a>

### ChangelogRevision



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revision | [string](#string) |  | Marshalled revision for display |
| operation | [ChangelogRevision.Op](#bytebase-store-ChangelogRevision-Op) |  |  |






<a name="bytebase-store-ChangelogTask"></a>

### ChangelogTask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| task_run | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskruns/{taskrun} |
| issue | [string](#string) |  | Format: projects/{project}/issues/{issue} |
| revision | [int64](#int64) |  | The revision uid. optional |
| changed_resources | [ChangedResources](#bytebase-store-ChangedResources) |  |  |
| status | [ChangelogTask.Status](#bytebase-store-ChangelogTask-Status) |  |  |
| prev_sync_history_id | [int64](#int64) |  |  |
| sync_history_id | [int64](#int64) |  |  |
| sheet | [string](#string) |  | The sheet that holds the content. Format: projects/{project}/sheets/{sheet} |
| version | [string](#string) |  |  |
| type | [ChangelogTask.Type](#bytebase-store-ChangelogTask-Type) |  |  |





 


<a name="bytebase-store-ChangelogRevision-Op"></a>

### ChangelogRevision.Op


| Name | Number | Description |
| ---- | ------ | ----------- |
| OP_UNSPECIFIED | 0 |  |
| CREATE | 1 |  |
| DELETE | 2 |  |



<a name="bytebase-store-ChangelogTask-Status"></a>

### ChangelogTask.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| DONE | 2 |  |
| FAILED | 3 |  |



<a name="bytebase-store-ChangelogTask-Type"></a>

### ChangelogTask.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| BASELINE | 1 |  |
| MIGRATE | 2 |  |
| MIGRATE_SDL | 3 |  |
| MIGRATE_GHOST | 4 |  |
| DATA | 6 |  |


 

 

 



<a name="store_data_source-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/data_source.proto



<a name="bytebase-store-DataSourceExternalSecret"></a>

### DataSourceExternalSecret



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secret_type | [DataSourceExternalSecret.SecretType](#bytebase-store-DataSourceExternalSecret-SecretType) |  |  |
| url | [string](#string) |  |  |
| auth_type | [DataSourceExternalSecret.AuthType](#bytebase-store-DataSourceExternalSecret-AuthType) |  |  |
| app_role | [DataSourceExternalSecret.AppRoleAuthOption](#bytebase-store-DataSourceExternalSecret-AppRoleAuthOption) |  |  |
| token | [string](#string) |  |  |
| engine_name | [string](#string) |  | engine name is the name for secret engine. |
| secret_name | [string](#string) |  | the secret name in the engine to store the password. |
| password_key_name | [string](#string) |  | the key name for the password. |






<a name="bytebase-store-DataSourceExternalSecret-AppRoleAuthOption"></a>

### DataSourceExternalSecret.AppRoleAuthOption



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role_id | [string](#string) |  |  |
| secret_id | [string](#string) |  | the secret id for the role without ttl. |
| type | [DataSourceExternalSecret.AppRoleAuthOption.SecretType](#bytebase-store-DataSourceExternalSecret-AppRoleAuthOption-SecretType) |  |  |
| mount_path | [string](#string) |  | The path where the approle auth method is mounted. |






<a name="bytebase-store-DataSourceOptions"></a>

### DataSourceOptions



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| srv | [bool](#bool) |  | srv is a boolean flag that indicates whether the host is a DNS SRV record. |
| authentication_database | [string](#string) |  | authentication_database is the database name to authenticate against, which stores the user credentials. |
| sid | [string](#string) |  | sid and service_name are used for Oracle. |
| service_name | [string](#string) |  |  |
| ssh_host | [string](#string) |  | SSH related The hostname of the SSH server agent. |
| ssh_port | [string](#string) |  | The port of the SSH server agent. It&#39;s 22 typically. |
| ssh_user | [string](#string) |  | The user to login the server. |
| ssh_obfuscated_password | [string](#string) |  | The password to login the server. If it&#39;s empty string, no password is required. |
| ssh_obfuscated_private_key | [string](#string) |  | The private key to login the server. If it&#39;s empty string, we will use the system default private key from os.Getenv(&#34;SSH_AUTH_SOCK&#34;). |
| authentication_private_key_obfuscated | [string](#string) |  | PKCS#8 private key in PEM format. If it&#39;s empty string, no private key is required. Used for authentication when connecting to the data source. |
| external_secret | [DataSourceExternalSecret](#bytebase-store-DataSourceExternalSecret) |  |  |
| authentication_type | [DataSourceOptions.AuthenticationType](#bytebase-store-DataSourceOptions-AuthenticationType) |  |  |
| sasl_config | [SASLConfig](#bytebase-store-SASLConfig) |  |  |
| additional_addresses | [DataSourceOptions.Address](#bytebase-store-DataSourceOptions-Address) | repeated | additional_addresses is used for MongoDB replica set. |
| replica_set | [string](#string) |  | replica_set is used for MongoDB replica set. |
| direct_connection | [bool](#bool) |  | direct_connection is used for MongoDB to dispatch all the operations to the node specified in the connection string. |
| region | [string](#string) |  | region is the location of where the DB is, works for AWS RDS. For example, us-east-1. |
| warehouse_id | [string](#string) |  | warehouse_id is used by Databricks. |
| master_name | [string](#string) |  | master_name is the master name used by connecting redis-master via redis sentinel. |
| master_username | [string](#string) |  | master_username and master_obfuscated_password are master credentials used by redis sentinel mode. |
| master_obfuscated_password | [string](#string) |  |  |
| redis_type | [DataSourceOptions.RedisType](#bytebase-store-DataSourceOptions-RedisType) |  |  |
| use_ssl | [bool](#bool) |  | Use SSL to connect to the data source. By default, we use system default SSL configuration. |
| cluster | [string](#string) |  | Cluster is the cluster name for the data source. Used by CockroachDB. |






<a name="bytebase-store-DataSourceOptions-Address"></a>

### DataSourceOptions.Address



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| host | [string](#string) |  |  |
| port | [string](#string) |  |  |






<a name="bytebase-store-KerberosConfig"></a>

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






<a name="bytebase-store-SASLConfig"></a>

### SASLConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| krb_config | [KerberosConfig](#bytebase-store-KerberosConfig) |  |  |





 


<a name="bytebase-store-DataSourceExternalSecret-AppRoleAuthOption-SecretType"></a>

### DataSourceExternalSecret.AppRoleAuthOption.SecretType


| Name | Number | Description |
| ---- | ------ | ----------- |
| SECRET_TYPE_UNSPECIFIED | 0 |  |
| PLAIN | 1 |  |
| ENVIRONMENT | 2 |  |



<a name="bytebase-store-DataSourceExternalSecret-AuthType"></a>

### DataSourceExternalSecret.AuthType


| Name | Number | Description |
| ---- | ------ | ----------- |
| AUTH_TYPE_UNSPECIFIED | 0 |  |
| TOKEN | 1 | ref: https://developer.hashicorp.com/vault/docs/auth/token |
| VAULT_APP_ROLE | 2 | ref: https://developer.hashicorp.com/vault/docs/auth/approle |



<a name="bytebase-store-DataSourceExternalSecret-SecretType"></a>

### DataSourceExternalSecret.SecretType


| Name | Number | Description |
| ---- | ------ | ----------- |
| SAECRET_TYPE_UNSPECIFIED | 0 |  |
| VAULT_KV_V2 | 1 | ref: https://developer.hashicorp.com/vault/api-docs/secret/kv/kv-v2 |
| AWS_SECRETS_MANAGER | 2 | ref: https://docs.aws.amazon.com/secretsmanager/latest/userguide/intro.html |
| GCP_SECRET_MANAGER | 3 | ref: https://cloud.google.com/secret-manager/docs |



<a name="bytebase-store-DataSourceOptions-AuthenticationType"></a>

### DataSourceOptions.AuthenticationType


| Name | Number | Description |
| ---- | ------ | ----------- |
| AUTHENTICATION_UNSPECIFIED | 0 |  |
| PASSWORD | 1 |  |
| GOOGLE_CLOUD_SQL_IAM | 2 |  |
| AWS_RDS_IAM | 3 |  |



<a name="bytebase-store-DataSourceOptions-RedisType"></a>

### DataSourceOptions.RedisType


| Name | Number | Description |
| ---- | ------ | ----------- |
| REDIS_TYPE_UNSPECIFIED | 0 |  |
| STANDALONE | 1 |  |
| SENTINEL | 2 |  |
| CLUSTER | 3 |  |


 

 

 



<a name="store_db_group-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/db_group.proto



<a name="bytebase-store-DatabaseGroupPayload"></a>

### DatabaseGroupPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| multitenancy | [bool](#bool) |  |  |





 

 

 

 



<a name="store_export_archive-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/export_archive.proto



<a name="bytebase-store-ExportArchivePayload"></a>

### ExportArchivePayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file_format | [ExportFormat](#bytebase-store-ExportFormat) |  | The exported file format. e.g. JSON, CSV, SQL |





 

 

 

 



<a name="store_group-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/group.proto



<a name="bytebase-store-GroupMember"></a>

### GroupMember



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| member | [string](#string) |  | Member is the principal who belong to this group.

Format: users/{userUID}. |
| role | [GroupMember.Role](#bytebase-store-GroupMember-Role) |  |  |






<a name="bytebase-store-GroupPayload"></a>

### GroupPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| members | [GroupMember](#bytebase-store-GroupMember) | repeated |  |
| source | [string](#string) |  | source means where the group comes from. For now we support Entra ID SCIM sync, so the source could be Entra ID. |





 


<a name="bytebase-store-GroupMember-Role"></a>

### GroupMember.Role


| Name | Number | Description |
| ---- | ------ | ----------- |
| ROLE_UNSPECIFIED | 0 |  |
| OWNER | 1 |  |
| MEMBER | 2 |  |


 

 

 



<a name="store_idp-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/idp.proto



<a name="bytebase-store-FieldMapping"></a>

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
| display_name | [string](#string) |  | DisplayName is the field name of display name in 3rd-party idp user info. Optional. |
| email | [string](#string) |  | Email is the field name of primary email in 3rd-party idp user info. Optional. |
| phone | [string](#string) |  | Phone is the field name of primary phone in 3rd-party idp user info. Optional. |






<a name="bytebase-store-IdentityProviderConfig"></a>

### IdentityProviderConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| oauth2_config | [OAuth2IdentityProviderConfig](#bytebase-store-OAuth2IdentityProviderConfig) |  |  |
| oidc_config | [OIDCIdentityProviderConfig](#bytebase-store-OIDCIdentityProviderConfig) |  |  |
| ldap_config | [LDAPIdentityProviderConfig](#bytebase-store-LDAPIdentityProviderConfig) |  |  |






<a name="bytebase-store-IdentityProviderUserInfo"></a>

### IdentityProviderUserInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identifier | [string](#string) |  | Identifier is the value of the unique identifier in 3rd-party idp user info. |
| display_name | [string](#string) |  | DisplayName is the value of display name in 3rd-party idp user info. |
| email | [string](#string) |  | Email is the value of primary email in 3rd-party idp user info. |
| phone | [string](#string) |  | Phone is the value of primary phone in 3rd-party idp user info. |






<a name="bytebase-store-LDAPIdentityProviderConfig"></a>

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
| field_mapping | [FieldMapping](#bytebase-store-FieldMapping) |  | FieldMapping is the mapping of the user attributes returned by the LDAP server. |






<a name="bytebase-store-OAuth2IdentityProviderConfig"></a>

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
| field_mapping | [FieldMapping](#bytebase-store-FieldMapping) |  |  |
| skip_tls_verify | [bool](#bool) |  |  |
| auth_style | [OAuth2AuthStyle](#bytebase-store-OAuth2AuthStyle) |  |  |






<a name="bytebase-store-OIDCIdentityProviderConfig"></a>

### OIDCIdentityProviderConfig
OIDCIdentityProviderConfig is the structure for OIDC identity provider config.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issuer | [string](#string) |  |  |
| client_id | [string](#string) |  |  |
| client_secret | [string](#string) |  |  |
| field_mapping | [FieldMapping](#bytebase-store-FieldMapping) |  |  |
| skip_tls_verify | [bool](#bool) |  |  |
| auth_style | [OAuth2AuthStyle](#bytebase-store-OAuth2AuthStyle) |  |  |





 


<a name="bytebase-store-IdentityProviderType"></a>

### IdentityProviderType


| Name | Number | Description |
| ---- | ------ | ----------- |
| IDENTITY_PROVIDER_TYPE_UNSPECIFIED | 0 |  |
| OAUTH2 | 1 |  |
| OIDC | 2 |  |
| LDAP | 3 |  |



<a name="bytebase-store-OAuth2AuthStyle"></a>

### OAuth2AuthStyle


| Name | Number | Description |
| ---- | ------ | ----------- |
| OAUTH2_AUTH_STYLE_UNSPECIFIED | 0 |  |
| IN_PARAMS | 1 | IN_PARAMS sends the &#34;client_id&#34; and &#34;client_secret&#34; in the POST body as application/x-www-form-urlencoded parameters. |
| IN_HEADER | 2 | IN_HEADER sends the client_id and client_password using HTTP Basic Authorization. This is an optional style described in the OAuth2 RFC 6749 section 2.3.1. |


 

 

 



<a name="store_instance-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/instance.proto



<a name="bytebase-store-InstanceMetadata"></a>

### InstanceMetadata
InstanceMetadata is the metadata for instances.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| mysql_lower_case_table_names | [int32](#int32) |  | The lower_case_table_names config for MySQL instances. It is used to determine whether the table names and database names are case sensitive. |
| last_sync_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| roles | [InstanceRole](#bytebase-store-InstanceRole) | repeated |  |






<a name="bytebase-store-InstanceOptions"></a>

### InstanceOptions
InstanceOptions is the option for instances.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sync_interval | [google.protobuf.Duration](#google-protobuf-Duration) |  | How often the instance is synced. |
| maximum_connections | [int32](#int32) |  | The maximum number of connections. The default is 10 if the value is unset or zero. |






<a name="bytebase-store-InstanceRole"></a>

### InstanceRole
InstanceRole is the API message for instance role.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The role name. |
| connection_limit | [int32](#int32) | optional | The connection count limit for this role. |
| valid_until | [string](#string) | optional | The expiration for the role&#39;s password. |
| attribute | [string](#string) | optional | The role attribute. For PostgreSQL, it containt super_user, no_inherit, create_role, create_db, can_login, replication and bypass_rls. Docs: https://www.postgresql.org/docs/current/role-attributes.html For MySQL, it&#39;s the global privileges as GRANT statements, which means it only contains &#34;GRANT ... ON *.* TO ...&#34;. Docs: https://dev.mysql.com/doc/refman/8.0/en/grant.html |





 

 

 

 



<a name="store_issue-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/issue.proto



<a name="bytebase-store-GrantRequest"></a>

### GrantRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [string](#string) |  | The requested role. Format: roles/EXPORTER. |
| user | [string](#string) |  | The user to be granted. Format: users/{userUID}. |
| condition | [google.type.Expr](#google-type-Expr) |  |  |
| expiration | [google.protobuf.Duration](#google-protobuf-Duration) |  |  |






<a name="bytebase-store-IssuePayload"></a>

### IssuePayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| approval | [IssuePayloadApproval](#bytebase-store-IssuePayloadApproval) |  |  |
| grant_request | [GrantRequest](#bytebase-store-GrantRequest) |  |  |
| labels | [string](#string) | repeated |  |





 

 

 

 



<a name="store_issue_comment-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/issue_comment.proto



<a name="bytebase-store-IssueCommentPayload"></a>

### IssueCommentPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| comment | [string](#string) |  |  |
| approval | [IssueCommentPayload.Approval](#bytebase-store-IssueCommentPayload-Approval) |  |  |
| issue_update | [IssueCommentPayload.IssueUpdate](#bytebase-store-IssueCommentPayload-IssueUpdate) |  |  |
| stage_end | [IssueCommentPayload.StageEnd](#bytebase-store-IssueCommentPayload-StageEnd) |  |  |
| task_update | [IssueCommentPayload.TaskUpdate](#bytebase-store-IssueCommentPayload-TaskUpdate) |  |  |
| task_prior_backup | [IssueCommentPayload.TaskPriorBackup](#bytebase-store-IssueCommentPayload-TaskPriorBackup) |  |  |






<a name="bytebase-store-IssueCommentPayload-Approval"></a>

### IssueCommentPayload.Approval



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [IssueCommentPayload.Approval.Status](#bytebase-store-IssueCommentPayload-Approval-Status) |  |  |






<a name="bytebase-store-IssueCommentPayload-IssueUpdate"></a>

### IssueCommentPayload.IssueUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| from_title | [string](#string) | optional |  |
| to_title | [string](#string) | optional |  |
| from_description | [string](#string) | optional |  |
| to_description | [string](#string) | optional |  |
| from_status | [IssueCommentPayload.IssueUpdate.IssueStatus](#bytebase-store-IssueCommentPayload-IssueUpdate-IssueStatus) | optional |  |
| to_status | [IssueCommentPayload.IssueUpdate.IssueStatus](#bytebase-store-IssueCommentPayload-IssueUpdate-IssueStatus) | optional |  |
| from_labels | [string](#string) | repeated |  |
| to_labels | [string](#string) | repeated |  |






<a name="bytebase-store-IssueCommentPayload-StageEnd"></a>

### IssueCommentPayload.StageEnd



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stage | [string](#string) |  |  |






<a name="bytebase-store-IssueCommentPayload-TaskPriorBackup"></a>

### IssueCommentPayload.TaskPriorBackup



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| task | [string](#string) |  |  |
| tables | [IssueCommentPayload.TaskPriorBackup.Table](#bytebase-store-IssueCommentPayload-TaskPriorBackup-Table) | repeated |  |
| original_line | [int32](#int32) | optional |  |
| database | [string](#string) |  |  |
| error | [string](#string) |  |  |






<a name="bytebase-store-IssueCommentPayload-TaskPriorBackup-Table"></a>

### IssueCommentPayload.TaskPriorBackup.Table



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  |  |
| table | [string](#string) |  |  |






<a name="bytebase-store-IssueCommentPayload-TaskUpdate"></a>

### IssueCommentPayload.TaskUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tasks | [string](#string) | repeated |  |
| from_sheet | [string](#string) | optional | Format: projects/{project}/sheets/{sheet} |
| to_sheet | [string](#string) | optional | Format: projects/{project}/sheets/{sheet} |
| from_earliest_allowed_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional |  |
| to_earliest_allowed_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional |  |
| to_status | [IssueCommentPayload.TaskUpdate.Status](#bytebase-store-IssueCommentPayload-TaskUpdate-Status) | optional |  |





 


<a name="bytebase-store-IssueCommentPayload-Approval-Status"></a>

### IssueCommentPayload.Approval.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| APPROVED | 2 |  |
| REJECTED | 3 |  |



<a name="bytebase-store-IssueCommentPayload-IssueUpdate-IssueStatus"></a>

### IssueCommentPayload.IssueUpdate.IssueStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| ISSUE_STATUS_UNSPECIFIED | 0 |  |
| OPEN | 1 |  |
| DONE | 2 |  |
| CANCELED | 3 |  |



<a name="bytebase-store-IssueCommentPayload-TaskUpdate-Status"></a>

### IssueCommentPayload.TaskUpdate.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| RUNNING | 2 |  |
| DONE | 3 |  |
| FAILED | 4 |  |
| SKIPPED | 5 |  |
| CANCELED | 6 |  |


 

 

 



<a name="store_plan_check_run-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/plan_check_run.proto



<a name="bytebase-store-PlanCheckRunConfig"></a>

### PlanCheckRunConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheet_uid | [int32](#int32) |  |  |
| change_database_type | [PlanCheckRunConfig.ChangeDatabaseType](#bytebase-store-PlanCheckRunConfig-ChangeDatabaseType) |  |  |
| instance_uid | [int32](#int32) |  |  |
| database_name | [string](#string) |  |  |
| database_group_uid | [int64](#int64) | optional | **Deprecated.**  |
| ghost_flags | [PlanCheckRunConfig.GhostFlagsEntry](#bytebase-store-PlanCheckRunConfig-GhostFlagsEntry) | repeated |  |
| pre_update_backup_detail | [PreUpdateBackupDetail](#bytebase-store-PreUpdateBackupDetail) | optional | If set, a backup of the modified data will be created automatically before any changes are applied. |






<a name="bytebase-store-PlanCheckRunConfig-GhostFlagsEntry"></a>

### PlanCheckRunConfig.GhostFlagsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-store-PlanCheckRunResult"></a>

### PlanCheckRunResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [PlanCheckRunResult.Result](#bytebase-store-PlanCheckRunResult-Result) | repeated |  |
| error | [string](#string) |  |  |






<a name="bytebase-store-PlanCheckRunResult-Result"></a>

### PlanCheckRunResult.Result



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [PlanCheckRunResult.Result.Status](#bytebase-store-PlanCheckRunResult-Result-Status) |  |  |
| title | [string](#string) |  |  |
| content | [string](#string) |  |  |
| code | [int32](#int32) |  |  |
| sql_summary_report | [PlanCheckRunResult.Result.SqlSummaryReport](#bytebase-store-PlanCheckRunResult-Result-SqlSummaryReport) |  |  |
| sql_review_report | [PlanCheckRunResult.Result.SqlReviewReport](#bytebase-store-PlanCheckRunResult-Result-SqlReviewReport) |  |  |






<a name="bytebase-store-PlanCheckRunResult-Result-SqlReviewReport"></a>

### PlanCheckRunResult.Result.SqlReviewReport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| line | [int32](#int32) |  |  |
| column | [int32](#int32) |  |  |
| detail | [string](#string) |  |  |
| code | [int32](#int32) |  | Code from sql review. |
| start_position | [Position](#bytebase-store-Position) |  | 1-based Position of the SQL statement. To supersede `line` and `column` above. |
| end_position | [Position](#bytebase-store-Position) |  |  |






<a name="bytebase-store-PlanCheckRunResult-Result-SqlSummaryReport"></a>

### PlanCheckRunResult.Result.SqlSummaryReport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [int32](#int32) |  |  |
| statement_types | [string](#string) | repeated | statement_types are the types of statements that are found in the sql. |
| affected_rows | [int32](#int32) |  |  |
| changed_resources | [ChangedResources](#bytebase-store-ChangedResources) |  |  |






<a name="bytebase-store-PreUpdateBackupDetail"></a>

### PreUpdateBackupDetail



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [string](#string) |  | The database for keeping the backup data. Format: instances/{instance}/databases/{database} |





 


<a name="bytebase-store-PlanCheckRunConfig-ChangeDatabaseType"></a>

### PlanCheckRunConfig.ChangeDatabaseType


| Name | Number | Description |
| ---- | ------ | ----------- |
| CHANGE_DATABASE_TYPE_UNSPECIFIED | 0 |  |
| DDL | 1 |  |
| DML | 2 |  |
| SDL | 3 |  |
| DDL_GHOST | 4 |  |
| SQL_EDITOR | 5 |  |



<a name="bytebase-store-PlanCheckRunResult-Result-Status"></a>

### PlanCheckRunResult.Result.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| ERROR | 1 |  |
| WARNING | 2 |  |
| SUCCESS | 3 |  |


 

 

 



<a name="store_plan-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/plan.proto



<a name="bytebase-store-PlanConfig"></a>

### PlanConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| steps | [PlanConfig.Step](#bytebase-store-PlanConfig-Step) | repeated |  |
| vcs_source | [PlanConfig.VCSSource](#bytebase-store-PlanConfig-VCSSource) |  |  |
| release_source | [PlanConfig.ReleaseSource](#bytebase-store-PlanConfig-ReleaseSource) |  |  |






<a name="bytebase-store-PlanConfig-ChangeDatabaseConfig"></a>

### PlanConfig.ChangeDatabaseConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| target | [string](#string) |  | The resource name of the target. Format: instances/{instance-id}/databases/{database-name}. Format: projects/{project}/databaseGroups/{databaseGroup}. |
| sheet | [string](#string) |  | The resource name of the sheet. Format: projects/{project}/sheets/{sheet} |
| type | [PlanConfig.ChangeDatabaseConfig.Type](#bytebase-store-PlanConfig-ChangeDatabaseConfig-Type) |  |  |
| schema_version | [string](#string) |  | schema_version is parsed from VCS file name. It is automatically generated in the UI workflow. |
| ghost_flags | [PlanConfig.ChangeDatabaseConfig.GhostFlagsEntry](#bytebase-store-PlanConfig-ChangeDatabaseConfig-GhostFlagsEntry) | repeated |  |
| pre_update_backup_detail | [PreUpdateBackupDetail](#bytebase-store-PreUpdateBackupDetail) | optional | If set, a backup of the modified data will be created automatically before any changes are applied. |






<a name="bytebase-store-PlanConfig-ChangeDatabaseConfig-GhostFlagsEntry"></a>

### PlanConfig.ChangeDatabaseConfig.GhostFlagsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-store-PlanConfig-CreateDatabaseConfig"></a>

### PlanConfig.CreateDatabaseConfig



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
| environment | [string](#string) |  | The environment resource. Format: environments/prod where prod is the environment resource ID. |
| labels | [PlanConfig.CreateDatabaseConfig.LabelsEntry](#bytebase-store-PlanConfig-CreateDatabaseConfig-LabelsEntry) | repeated | labels of the database. |






<a name="bytebase-store-PlanConfig-CreateDatabaseConfig-LabelsEntry"></a>

### PlanConfig.CreateDatabaseConfig.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-store-PlanConfig-ExportDataConfig"></a>

### PlanConfig.ExportDataConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| target | [string](#string) |  | The resource name of the target. Format: instances/{instance-id}/databases/{database-name} |
| sheet | [string](#string) |  | The resource name of the sheet. Format: projects/{project}/sheets/{sheet} |
| format | [ExportFormat](#bytebase-store-ExportFormat) |  | The format of the exported file. |
| password | [string](#string) | optional | The zip password provide by users. Leave it empty if no needs to encrypt the zip file. |






<a name="bytebase-store-PlanConfig-ReleaseSource"></a>

### PlanConfig.ReleaseSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| release | [string](#string) |  | The release. Format: projects/{project}/releases/{release} |






<a name="bytebase-store-PlanConfig-Spec"></a>

### PlanConfig.Spec



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| earliest_allowed_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | earliest_allowed_time the earliest execution time of the change. |
| id | [string](#string) |  | A UUID4 string that uniquely identifies the Spec. |
| depends_on_specs | [string](#string) | repeated | IDs of the specs that this spec depends on. Must be a subset of the specs in the same step. |
| spec_release_source | [PlanConfig.SpecReleaseSource](#bytebase-store-PlanConfig-SpecReleaseSource) |  |  |
| create_database_config | [PlanConfig.CreateDatabaseConfig](#bytebase-store-PlanConfig-CreateDatabaseConfig) |  |  |
| change_database_config | [PlanConfig.ChangeDatabaseConfig](#bytebase-store-PlanConfig-ChangeDatabaseConfig) |  |  |
| export_data_config | [PlanConfig.ExportDataConfig](#bytebase-store-PlanConfig-ExportDataConfig) |  |  |






<a name="bytebase-store-PlanConfig-SpecReleaseSource"></a>

### PlanConfig.SpecReleaseSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file | [string](#string) |  | Format: projects/{project}/releases/{release}/files/{id} |






<a name="bytebase-store-PlanConfig-Step"></a>

### PlanConfig.Step



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  | Use the title if set. Use a generated title if empty. |
| specs | [PlanConfig.Spec](#bytebase-store-PlanConfig-Spec) | repeated |  |






<a name="bytebase-store-PlanConfig-VCSSource"></a>

### PlanConfig.VCSSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vcs_type | [VCSType](#bytebase-store-VCSType) |  |  |
| vcs_connector | [string](#string) |  | Optional. If present, we will update the pull request for rollout status. Format: projects/{project-ID}/vcsConnectors/{vcs-connector} |
| pull_request_url | [string](#string) |  |  |





 


<a name="bytebase-store-PlanConfig-ChangeDatabaseConfig-Type"></a>

### PlanConfig.ChangeDatabaseConfig.Type
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


 

 

 



<a name="store_policy-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/policy.proto



<a name="bytebase-store-Binding"></a>

### Binding



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [string](#string) |  | The role that is assigned to the members. Format: roles/{role} |
| members | [string](#string) | repeated | Specifies the principals requesting access for a Bytebase resource. For users, the member should be: users/{userUID} For groups, the member should be: groups/{email} |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this binding. If the condition evaluates to true, then this binding applies to the current request. If the condition evaluates to false, then this binding does not apply to the current request. However, a different role binding might grant the same role to one or more of the principals in this binding. |






<a name="bytebase-store-DataSourceQueryPolicy"></a>

### DataSourceQueryPolicy
DataSourceQueryPolicy is the policy configuration for running statements in the SQL editor.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| admin_data_source_restriction | [DataSourceQueryPolicy.Restriction](#bytebase-store-DataSourceQueryPolicy-Restriction) |  |  |
| disallow_ddl | [bool](#bool) |  | Disallow running DDL statements in the SQL editor. |
| disallow_dml | [bool](#bool) |  | Disallow running DML statements in the SQL editor. |






<a name="bytebase-store-DisableCopyDataPolicy"></a>

### DisableCopyDataPolicy
DisableCopyDataPolicy is the policy configuration for disabling copying data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| active | [bool](#bool) |  |  |






<a name="bytebase-store-EnvironmentTierPolicy"></a>

### EnvironmentTierPolicy
EnvironmentTierPolicy is the tier of an environment.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| environment_tier | [EnvironmentTierPolicy.EnvironmentTier](#bytebase-store-EnvironmentTierPolicy-EnvironmentTier) |  |  |
| color | [string](#string) |  |  |






<a name="bytebase-store-ExportDataPolicy"></a>

### ExportDataPolicy
ExportDataPolicy is the policy configuration for export data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| disable | [bool](#bool) |  |  |






<a name="bytebase-store-IamPolicy"></a>

### IamPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bindings | [Binding](#bytebase-store-Binding) | repeated | Collection of binding. A binding binds one or more members or groups to a single role. |






<a name="bytebase-store-MaskData"></a>

### MaskData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  |  |
| table | [string](#string) |  |  |
| column | [string](#string) |  |  |
| masking_level | [MaskingLevel](#bytebase-store-MaskingLevel) |  |  |
| full_masking_algorithm_id | [string](#string) |  |  |
| partial_masking_algorithm_id | [string](#string) |  |  |






<a name="bytebase-store-MaskingExceptionPolicy"></a>

### MaskingExceptionPolicy
MaskingExceptionPolicy is the allowlist of users who can access sensitive data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| masking_exceptions | [MaskingExceptionPolicy.MaskingException](#bytebase-store-MaskingExceptionPolicy-MaskingException) | repeated |  |






<a name="bytebase-store-MaskingExceptionPolicy-MaskingException"></a>

### MaskingExceptionPolicy.MaskingException



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| action | [MaskingExceptionPolicy.MaskingException.Action](#bytebase-store-MaskingExceptionPolicy-MaskingException-Action) |  | action is the action that the user can access sensitive data. |
| masking_level | [MaskingLevel](#bytebase-store-MaskingLevel) |  | Level is the masking level that the user can access sensitive data. |
| member | [string](#string) |  | Member is the principal who bind to this exception policy instance.

Format: users/{userUID} or groups/{group email} |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this exception policy instance. |






<a name="bytebase-store-MaskingPolicy"></a>

### MaskingPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| mask_data | [MaskData](#bytebase-store-MaskData) | repeated |  |






<a name="bytebase-store-MaskingRulePolicy"></a>

### MaskingRulePolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rules | [MaskingRulePolicy.MaskingRule](#bytebase-store-MaskingRulePolicy-MaskingRule) | repeated |  |






<a name="bytebase-store-MaskingRulePolicy-MaskingRule"></a>

### MaskingRulePolicy.MaskingRule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | A unique identifier for a node in UUID format. |
| condition | [google.type.Expr](#google-type-Expr) |  |  |
| masking_level | [MaskingLevel](#bytebase-store-MaskingLevel) |  |  |






<a name="bytebase-store-RestrictIssueCreationForSQLReviewPolicy"></a>

### RestrictIssueCreationForSQLReviewPolicy
RestrictIssueCreationForSQLReviewPolicy is the policy configuration for restricting issue creation for SQL review.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| disallow | [bool](#bool) |  |  |






<a name="bytebase-store-RolloutPolicy"></a>

### RolloutPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| automatic | [bool](#bool) |  |  |
| workspace_roles | [string](#string) | repeated |  |
| project_roles | [string](#string) | repeated |  |
| issue_roles | [string](#string) | repeated | roles/LAST_APPROVER roles/CREATOR |






<a name="bytebase-store-SQLReviewRule"></a>

### SQLReviewRule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  |  |
| level | [SQLReviewRuleLevel](#bytebase-store-SQLReviewRuleLevel) |  |  |
| payload | [string](#string) |  |  |
| engine | [Engine](#bytebase-store-Engine) |  |  |
| comment | [string](#string) |  |  |






<a name="bytebase-store-SlowQueryPolicy"></a>

### SlowQueryPolicy
SlowQueryPolicy is the policy configuration for slow query.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| active | [bool](#bool) |  |  |






<a name="bytebase-store-TagPolicy"></a>

### TagPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tags | [TagPolicy.TagsEntry](#bytebase-store-TagPolicy-TagsEntry) | repeated | tags is the key - value map for resources. for example, the environment resource can have the sql review config tag, like &#34;bb.tag.review_config&#34;: &#34;reviewConfigs/{review config resource id}&#34; |






<a name="bytebase-store-TagPolicy-TagsEntry"></a>

### TagPolicy.TagsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |





 


<a name="bytebase-store-DataSourceQueryPolicy-Restriction"></a>

### DataSourceQueryPolicy.Restriction


| Name | Number | Description |
| ---- | ------ | ----------- |
| RESTRICTION_UNSPECIFIED | 0 |  |
| FALLBACK | 1 | Allow to query admin data sources when there is no read-only data source. |
| DISALLOW | 2 | Disallow to query admin data sources. |



<a name="bytebase-store-EnvironmentTierPolicy-EnvironmentTier"></a>

### EnvironmentTierPolicy.EnvironmentTier


| Name | Number | Description |
| ---- | ------ | ----------- |
| ENVIRONMENT_TIER_UNSPECIFIED | 0 |  |
| PROTECTED | 1 |  |
| UNPROTECTED | 2 |  |



<a name="bytebase-store-MaskingExceptionPolicy-MaskingException-Action"></a>

### MaskingExceptionPolicy.MaskingException.Action


| Name | Number | Description |
| ---- | ------ | ----------- |
| ACTION_UNSPECIFIED | 0 |  |
| QUERY | 1 |  |
| EXPORT | 2 |  |



<a name="bytebase-store-SQLReviewRuleLevel"></a>

### SQLReviewRuleLevel


| Name | Number | Description |
| ---- | ------ | ----------- |
| LEVEL_UNSPECIFIED | 0 |  |
| ERROR | 1 |  |
| WARNING | 2 |  |
| DISABLED | 3 |  |


 

 

 



<a name="store_project-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/project.proto



<a name="bytebase-store-Label"></a>

### Label



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [string](#string) |  |  |
| color | [string](#string) |  |  |
| group | [string](#string) |  |  |






<a name="bytebase-store-Project"></a>

### Project



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issue_labels | [Label](#bytebase-store-Label) | repeated |  |
| force_issue_labels | [bool](#bool) |  | Force issue labels to be used when creating an issue. |
| allow_modify_statement | [bool](#bool) |  | Allow modifying statement after issue is created. |
| auto_resolve_issue | [bool](#bool) |  | Enable auto resolve issue. |
| enforce_issue_title | [bool](#bool) |  | Enforce issue title created by user instead of generated by Bytebase. |
| auto_enable_backup | [bool](#bool) |  | Whether to automatically enable backup. |
| skip_backup_errors | [bool](#bool) |  | Whether to skip backup errors and continue the data migration. |
| postgres_database_tenant_mode | [bool](#bool) |  | Whether to enable the database tenant mode for PostgreSQL. If enabled, the issue will be created with the pre-appended &#34;set role &lt;db_owner&gt;&#34; statement. |





 

 

 

 



<a name="store_project_webhook-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/project_webhook.proto



<a name="bytebase-store-ProjectWebhookPayload"></a>

### ProjectWebhookPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| direct_message | [bool](#bool) |  | if direct_message is set, the notification is sent directly to the persons and url will be ignored. IM integration setting should be set for this function to work. |





 

 

 

 



<a name="store_query_history-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/query_history.proto



<a name="bytebase-store-QueryHistoryPayload"></a>

### QueryHistoryPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| error | [string](#string) | optional |  |
| duration | [google.protobuf.Duration](#google-protobuf-Duration) |  |  |





 

 

 

 



<a name="store_release-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/release.proto



<a name="bytebase-store-ReleasePayload"></a>

### ReleasePayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  |  |
| files | [ReleasePayload.File](#bytebase-store-ReleasePayload-File) | repeated |  |
| vcs_source | [ReleasePayload.VCSSource](#bytebase-store-ReleasePayload-VCSSource) |  |  |






<a name="bytebase-store-ReleasePayload-File"></a>

### ReleasePayload.File



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | The unique identifier for the file. |
| path | [string](#string) |  | The path of the file. e.g. `2.2/V0001_create_table.sql`. |
| sheet | [string](#string) |  | The sheet that holds the content. Format: projects/{project}/sheets/{sheet} |
| sheet_sha256 | [string](#string) |  | The SHA256 hash value of the sheet. |
| type | [ReleaseFileType](#bytebase-store-ReleaseFileType) |  |  |
| version | [string](#string) |  |  |






<a name="bytebase-store-ReleasePayload-VCSSource"></a>

### ReleasePayload.VCSSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vcs_type | [VCSType](#bytebase-store-VCSType) |  |  |
| pull_request_url | [string](#string) |  |  |





 


<a name="bytebase-store-ReleaseFileType"></a>

### ReleaseFileType


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| VERSIONED | 1 |  |


 

 

 



<a name="store_review_config-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/review_config.proto



<a name="bytebase-store-ReviewConfigPayload"></a>

### ReviewConfigPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sql_review_rules | [SQLReviewRule](#bytebase-store-SQLReviewRule) | repeated |  |





 

 

 

 



<a name="store_revision-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/revision.proto



<a name="bytebase-store-RevisionPayload"></a>

### RevisionPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| release | [string](#string) |  | Format: projects/{project}/releases/{release} Can be empty. |
| file | [string](#string) |  | Format: projects/{project}/releases/{release}/files/{id} Can be empty. |
| sheet | [string](#string) |  | The sheet that holds the content. Format: projects/{project}/sheets/{sheet} |
| sheet_sha256 | [string](#string) |  | The SHA256 hash value of the sheet. |
| task_run | [string](#string) |  | The task run associated with the revision. Can be empty. Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |





 

 

 

 



<a name="store_role-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/role.proto



<a name="bytebase-store-RolePermissions"></a>

### RolePermissions



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| permissions | [string](#string) | repeated |  |





 

 

 

 



<a name="store_setting-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/setting.proto



<a name="bytebase-store-AgentPluginSetting"></a>

### AgentPluginSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | [string](#string) |  | The URL for the agent API. |
| token | [string](#string) |  | The token for the agent. |






<a name="bytebase-store-Announcement"></a>

### Announcement



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| level | [Announcement.AlertLevel](#bytebase-store-Announcement-AlertLevel) |  | The alert level of announcemnt |
| text | [string](#string) |  | The text of announcemnt |
| link | [string](#string) |  | The optional link, user can follow the link to check extra details |






<a name="bytebase-store-AppIMSetting"></a>

### AppIMSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| slack | [AppIMSetting.Slack](#bytebase-store-AppIMSetting-Slack) |  |  |
| feishu | [AppIMSetting.Feishu](#bytebase-store-AppIMSetting-Feishu) |  |  |
| wecom | [AppIMSetting.Wecom](#bytebase-store-AppIMSetting-Wecom) |  |  |






<a name="bytebase-store-AppIMSetting-Feishu"></a>

### AppIMSetting.Feishu



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| app_id | [string](#string) |  |  |
| app_secret | [string](#string) |  |  |






<a name="bytebase-store-AppIMSetting-Slack"></a>

### AppIMSetting.Slack



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| token | [string](#string) |  |  |






<a name="bytebase-store-AppIMSetting-Wecom"></a>

### AppIMSetting.Wecom



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| corp_id | [string](#string) |  |  |
| agent_id | [string](#string) |  |  |
| secret | [string](#string) |  |  |






<a name="bytebase-store-DataClassificationSetting"></a>

### DataClassificationSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| configs | [DataClassificationSetting.DataClassificationConfig](#bytebase-store-DataClassificationSetting-DataClassificationConfig) | repeated |  |






<a name="bytebase-store-DataClassificationSetting-DataClassificationConfig"></a>

### DataClassificationSetting.DataClassificationConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | id is the uuid for classification. Each project can chose one classification config. |
| title | [string](#string) |  |  |
| levels | [DataClassificationSetting.DataClassificationConfig.Level](#bytebase-store-DataClassificationSetting-DataClassificationConfig-Level) | repeated | levels is user defined level list for classification. The order for the level decides its priority. |
| classification | [DataClassificationSetting.DataClassificationConfig.ClassificationEntry](#bytebase-store-DataClassificationSetting-DataClassificationConfig-ClassificationEntry) | repeated | classification is the id - DataClassification map. The id should in [0-9]&#43;-[0-9]&#43;-[0-9]&#43; format. |
| classification_from_config | [bool](#bool) |  | If true, we will only store the classification in the config. Otherwise we will get the classification from table/column comment, and write back to the schema metadata. |






<a name="bytebase-store-DataClassificationSetting-DataClassificationConfig-ClassificationEntry"></a>

### DataClassificationSetting.DataClassificationConfig.ClassificationEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [DataClassificationSetting.DataClassificationConfig.DataClassification](#bytebase-store-DataClassificationSetting-DataClassificationConfig-DataClassification) |  |  |






<a name="bytebase-store-DataClassificationSetting-DataClassificationConfig-DataClassification"></a>

### DataClassificationSetting.DataClassificationConfig.DataClassification



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | id is the classification id in [0-9]&#43;-[0-9]&#43;-[0-9]&#43; format. |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| level_id | [string](#string) | optional |  |






<a name="bytebase-store-DataClassificationSetting-DataClassificationConfig-Level"></a>

### DataClassificationSetting.DataClassificationConfig.Level



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |






<a name="bytebase-store-ExternalApprovalPayload"></a>

### ExternalApprovalPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| external_approval_node_id | [string](#string) |  |  |
| id | [string](#string) |  |  |






<a name="bytebase-store-ExternalApprovalSetting"></a>

### ExternalApprovalSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| nodes | [ExternalApprovalSetting.Node](#bytebase-store-ExternalApprovalSetting-Node) | repeated |  |






<a name="bytebase-store-ExternalApprovalSetting-Node"></a>

### ExternalApprovalSetting.Node



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | A unique identifier for a node in UUID format. We will also include the id in the message sending to the external relay service to identify the node. |
| title | [string](#string) |  | The title of the node. |
| endpoint | [string](#string) |  | The external endpoint for the relay service, e.g. &#34;http://hello:1234&#34;. |






<a name="bytebase-store-MaskingAlgorithmSetting"></a>

### MaskingAlgorithmSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| algorithms | [MaskingAlgorithmSetting.Algorithm](#bytebase-store-MaskingAlgorithmSetting-Algorithm) | repeated | algorithms is the list of masking algorithms. |






<a name="bytebase-store-MaskingAlgorithmSetting-Algorithm"></a>

### MaskingAlgorithmSetting.Algorithm



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | id is the uuid for masking algorithm. |
| title | [string](#string) |  | title is the title for masking algorithm. |
| description | [string](#string) |  | description is the description for masking algorithm. |
| category | [string](#string) |  | Category is the category for masking algorithm. Currently, it accepts 2 categories only: MASK and HASH. The range of accepted Payload is decided by the category. MASK: FullMask, RangeMask HASH: MD5Mask |
| full_mask | [MaskingAlgorithmSetting.Algorithm.FullMask](#bytebase-store-MaskingAlgorithmSetting-Algorithm-FullMask) |  |  |
| range_mask | [MaskingAlgorithmSetting.Algorithm.RangeMask](#bytebase-store-MaskingAlgorithmSetting-Algorithm-RangeMask) |  |  |
| md5_mask | [MaskingAlgorithmSetting.Algorithm.MD5Mask](#bytebase-store-MaskingAlgorithmSetting-Algorithm-MD5Mask) |  |  |
| inner_outer_mask | [MaskingAlgorithmSetting.Algorithm.InnerOuterMask](#bytebase-store-MaskingAlgorithmSetting-Algorithm-InnerOuterMask) |  |  |






<a name="bytebase-store-MaskingAlgorithmSetting-Algorithm-FullMask"></a>

### MaskingAlgorithmSetting.Algorithm.FullMask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| substitution | [string](#string) |  | substitution is the string used to replace the original value, the max length of the string is 16 bytes. |






<a name="bytebase-store-MaskingAlgorithmSetting-Algorithm-InnerOuterMask"></a>

### MaskingAlgorithmSetting.Algorithm.InnerOuterMask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| prefix_len | [int32](#int32) |  |  |
| suffix_len | [int32](#int32) |  |  |
| substitution | [string](#string) |  |  |
| type | [MaskingAlgorithmSetting.Algorithm.InnerOuterMask.MaskType](#bytebase-store-MaskingAlgorithmSetting-Algorithm-InnerOuterMask-MaskType) |  |  |






<a name="bytebase-store-MaskingAlgorithmSetting-Algorithm-MD5Mask"></a>

### MaskingAlgorithmSetting.Algorithm.MD5Mask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| salt | [string](#string) |  | salt is the salt value to generate a different hash that with the word alone. |






<a name="bytebase-store-MaskingAlgorithmSetting-Algorithm-RangeMask"></a>

### MaskingAlgorithmSetting.Algorithm.RangeMask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| slices | [MaskingAlgorithmSetting.Algorithm.RangeMask.Slice](#bytebase-store-MaskingAlgorithmSetting-Algorithm-RangeMask-Slice) | repeated | We store it as a repeated field to face the fact that the original value may have multiple parts should be masked. But frontend can be started with a single rule easily. |






<a name="bytebase-store-MaskingAlgorithmSetting-Algorithm-RangeMask-Slice"></a>

### MaskingAlgorithmSetting.Algorithm.RangeMask.Slice



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start | [int32](#int32) |  | start is the start index of the original value, start from 0 and should be less than stop. |
| end | [int32](#int32) |  | stop is the stop index of the original value, should be less than the length of the original value. |
| substitution | [string](#string) |  | OriginalValue[start:end) would be replaced with replace_with. |






<a name="bytebase-store-MaximumSQLResultSizeSetting"></a>

### MaximumSQLResultSizeSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| limit | [int64](#int64) |  | The limit is in bytes. The default value is 100MB, we will use the default value if the setting not exists, or the limit &lt;= 0. |






<a name="bytebase-store-PasswordRestrictionSetting"></a>

### PasswordRestrictionSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| min_length | [int32](#int32) |  | min_length is the minimum length for password, should no less than 8. |
| require_number | [bool](#bool) |  | require_number requires the password must contains at least one number. |
| require_letter | [bool](#bool) |  | require_letter requires the password must contains at least one letter, regardless of upper case or lower case |
| require_uppercase_letter | [bool](#bool) |  | require_uppercase_letter requires the password must contains at least one upper case letter. |
| require_special_character | [bool](#bool) |  | require_uppercase_letter requires the password must contains at least one special character. |
| require_reset_password_for_first_login | [bool](#bool) |  | require_reset_password_for_first_login requires users to reset their password after the 1st login. |
| password_rotation | [google.protobuf.Duration](#google-protobuf-Duration) |  | password_rotation requires users to reset their password after the duration. |






<a name="bytebase-store-SCIMSetting"></a>

### SCIMSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  |  |






<a name="bytebase-store-SMTPMailDeliverySetting"></a>

### SMTPMailDeliverySetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| server | [string](#string) |  | The SMTP server address. |
| port | [int32](#int32) |  | The SMTP server port. |
| encryption | [SMTPMailDeliverySetting.Encryption](#bytebase-store-SMTPMailDeliverySetting-Encryption) |  | The SMTP server encryption. |
| ca | [string](#string) |  | The CA, KEY, and CERT for the SMTP server. |
| key | [string](#string) |  |  |
| cert | [string](#string) |  |  |
| authentication | [SMTPMailDeliverySetting.Authentication](#bytebase-store-SMTPMailDeliverySetting-Authentication) |  |  |
| username | [string](#string) |  |  |
| password | [string](#string) |  |  |
| from | [string](#string) |  | The sender email address. |






<a name="bytebase-store-SchemaTemplateSetting"></a>

### SchemaTemplateSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| field_templates | [SchemaTemplateSetting.FieldTemplate](#bytebase-store-SchemaTemplateSetting-FieldTemplate) | repeated |  |
| column_types | [SchemaTemplateSetting.ColumnType](#bytebase-store-SchemaTemplateSetting-ColumnType) | repeated |  |
| table_templates | [SchemaTemplateSetting.TableTemplate](#bytebase-store-SchemaTemplateSetting-TableTemplate) | repeated |  |






<a name="bytebase-store-SchemaTemplateSetting-ColumnType"></a>

### SchemaTemplateSetting.ColumnType



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| engine | [Engine](#bytebase-store-Engine) |  |  |
| enabled | [bool](#bool) |  |  |
| types | [string](#string) | repeated |  |






<a name="bytebase-store-SchemaTemplateSetting-FieldTemplate"></a>

### SchemaTemplateSetting.FieldTemplate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| engine | [Engine](#bytebase-store-Engine) |  |  |
| category | [string](#string) |  |  |
| column | [ColumnMetadata](#bytebase-store-ColumnMetadata) |  |  |
| config | [ColumnConfig](#bytebase-store-ColumnConfig) |  |  |






<a name="bytebase-store-SchemaTemplateSetting-TableTemplate"></a>

### SchemaTemplateSetting.TableTemplate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| engine | [Engine](#bytebase-store-Engine) |  |  |
| category | [string](#string) |  |  |
| table | [TableMetadata](#bytebase-store-TableMetadata) |  |  |
| config | [TableConfig](#bytebase-store-TableConfig) |  |  |






<a name="bytebase-store-SemanticTypeSetting"></a>

### SemanticTypeSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| types | [SemanticTypeSetting.SemanticType](#bytebase-store-SemanticTypeSetting-SemanticType) | repeated |  |






<a name="bytebase-store-SemanticTypeSetting-SemanticType"></a>

### SemanticTypeSetting.SemanticType



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | id is the uuid for semantic type. |
| title | [string](#string) |  | the title of the semantic type, it should not be empty. |
| description | [string](#string) |  | the description of the semantic type, it can be empty. |
| partial_mask_algorithm_id | [string](#string) |  | the partial mask algorithm id for the semantic type, if it is empty, should use the default partial mask algorithm. |
| full_mask_algorithm_id | [string](#string) |  | the full mask algorithm id for the semantic type, if it is empty, should use the default full mask algorithm. |






<a name="bytebase-store-WorkspaceApprovalSetting"></a>

### WorkspaceApprovalSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rules | [WorkspaceApprovalSetting.Rule](#bytebase-store-WorkspaceApprovalSetting-Rule) | repeated |  |






<a name="bytebase-store-WorkspaceApprovalSetting-Rule"></a>

### WorkspaceApprovalSetting.Rule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expression | [google.api.expr.v1alpha1.Expr](#google-api-expr-v1alpha1-Expr) |  |  |
| template | [ApprovalTemplate](#bytebase-store-ApprovalTemplate) |  |  |
| condition | [google.type.Expr](#google-type-Expr) |  |  |






<a name="bytebase-store-WorkspaceProfileSetting"></a>

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
| announcement | [Announcement](#bytebase-store-Announcement) |  | The setting of custom announcement |
| maximum_role_expiration | [google.protobuf.Duration](#google-protobuf-Duration) |  | The max duration for role expired. |
| domains | [string](#string) | repeated | The workspace domain, e.g. bytebase.com. |
| enforce_identity_domain | [bool](#bool) |  | Only user and group from the domains can be created and login. |
| database_change_mode | [DatabaseChangeMode](#bytebase-store-DatabaseChangeMode) |  | The workspace database change mode. |
| disallow_password_signin | [bool](#bool) |  | Whether to disallow password signin. (Except workspace admins) |





 


<a name="bytebase-store-Announcement-AlertLevel"></a>

### Announcement.AlertLevel
We support three levels of AlertLevel: INFO, WARNING, and ERROR.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ALERT_LEVEL_UNSPECIFIED | 0 |  |
| ALERT_LEVEL_INFO | 1 |  |
| ALERT_LEVEL_WARNING | 2 |  |
| ALERT_LEVEL_CRITICAL | 3 |  |



<a name="bytebase-store-DatabaseChangeMode"></a>

### DatabaseChangeMode


| Name | Number | Description |
| ---- | ------ | ----------- |
| DATABASE_CHANGE_MODE_UNSPECIFIED | 0 |  |
| PIPELINE | 1 | A more advanced database change process, including custom approval workflows and other advanced features. Default to this mode. |
| EDITOR | 2 | A simple database change process in SQL editor. Users can execute SQL directly. |



<a name="bytebase-store-MaskingAlgorithmSetting-Algorithm-InnerOuterMask-MaskType"></a>

### MaskingAlgorithmSetting.Algorithm.InnerOuterMask.MaskType


| Name | Number | Description |
| ---- | ------ | ----------- |
| MASK_TYPE_UNSPECIFIED | 0 |  |
| INNER | 1 |  |
| OUTER | 2 |  |



<a name="bytebase-store-SMTPMailDeliverySetting-Authentication"></a>

### SMTPMailDeliverySetting.Authentication
We support four types of SMTP authentication: NONE, PLAIN, LOGIN, and
CRAM-MD5.

| Name | Number | Description |
| ---- | ------ | ----------- |
| AUTHENTICATION_UNSPECIFIED | 0 |  |
| AUTHENTICATION_NONE | 1 |  |
| AUTHENTICATION_PLAIN | 2 |  |
| AUTHENTICATION_LOGIN | 3 |  |
| AUTHENTICATION_CRAM_MD5 | 4 |  |



<a name="bytebase-store-SMTPMailDeliverySetting-Encryption"></a>

### SMTPMailDeliverySetting.Encryption
We support three types of SMTP encryption: NONE, STARTTLS, and SSL/TLS.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ENCRYPTION_UNSPECIFIED | 0 |  |
| ENCRYPTION_NONE | 1 |  |
| ENCRYPTION_STARTTLS | 2 |  |
| ENCRYPTION_SSL_TLS | 3 |  |


 

 

 



<a name="store_sheet-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/sheet.proto



<a name="bytebase-store-SheetCommand"></a>

### SheetCommand



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start | [int32](#int32) |  |  |
| end | [int32](#int32) |  |  |






<a name="bytebase-store-SheetPayload"></a>

### SheetPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database_config | [DatabaseConfig](#bytebase-store-DatabaseConfig) |  | The snapshot of the database config when creating the sheet, be used to compare with the baseline_database_config and apply the diff to the database. |
| baseline_database_config | [DatabaseConfig](#bytebase-store-DatabaseConfig) |  | The snapshot of the baseline database config when creating the sheet. |
| engine | [Engine](#bytebase-store-Engine) |  | The SQL dialect. |
| commands | [SheetCommand](#bytebase-store-SheetCommand) | repeated | The start and end position of each command in the sheet statement. |





 

 

 

 



<a name="store_slow_query-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/slow_query.proto



<a name="bytebase-store-SlowQueryDetails"></a>

### SlowQueryDetails
SlowQueryDetails is the details of a slow query.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | start_time is the start time of the slow query. |
| query_time | [google.protobuf.Duration](#google-protobuf-Duration) |  | query_time is the query time of the slow query. |
| lock_time | [google.protobuf.Duration](#google-protobuf-Duration) |  | lock_time is the lock time of the slow query. |
| rows_sent | [int32](#int32) |  | rows_sent is the number of rows sent by the slow query. |
| rows_examined | [int32](#int32) |  | rows_examined is the number of rows examined by the slow query. |
| sql_text | [string](#string) |  | sql_text is the SQL text of the slow query. |






<a name="bytebase-store-SlowQueryStatistics"></a>

### SlowQueryStatistics
SlowQueryStatistics is the slow query statistics.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| items | [SlowQueryStatisticsItem](#bytebase-store-SlowQueryStatisticsItem) | repeated | Items is the list of slow query statistics. |






<a name="bytebase-store-SlowQueryStatisticsItem"></a>

### SlowQueryStatisticsItem
SlowQueryStatisticsItem is the item of slow query statistics.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sql_fingerprint | [string](#string) |  | sql_fingerprint is the fingerprint of the slow query. |
| count | [int32](#int32) |  | count is the number of slow queries with the same fingerprint. |
| latest_log_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | latest_log_time is the time of the latest slow query with the same fingerprint. |
| total_query_time | [google.protobuf.Duration](#google-protobuf-Duration) |  | The total query time of the slow query log. |
| maximum_query_time | [google.protobuf.Duration](#google-protobuf-Duration) |  | The maximum query time of the slow query log. |
| total_rows_sent | [int32](#int32) |  | The total rows sent of the slow query log. |
| maximum_rows_sent | [int32](#int32) |  | The maximum rows sent of the slow query log. |
| total_rows_examined | [int32](#int32) |  | The total rows examined of the slow query log. |
| maximum_rows_examined | [int32](#int32) |  | The maximum rows examined of the slow query log. |
| samples | [SlowQueryDetails](#bytebase-store-SlowQueryDetails) | repeated | samples are the details of the sample slow queries with the same fingerprint. |





 

 

 

 



<a name="store_task-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/task.proto



<a name="bytebase-store-TaskDatabaseCreatePayload"></a>

### TaskDatabaseCreatePayload
TaskDatabaseCreatePayload is the task payload for creating databases.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| skipped | [bool](#bool) |  | common fields |
| skipped_reason | [string](#string) |  |  |
| spec_id | [string](#string) |  |  |
| project_id | [int32](#int32) |  |  |
| database_name | [string](#string) |  |  |
| table_name | [string](#string) |  |  |
| sheet_id | [int32](#int32) |  |  |
| character_set | [string](#string) |  |  |
| collation | [string](#string) |  |  |
| environment_id | [string](#string) |  |  |
| labels | [string](#string) |  |  |






<a name="bytebase-store-TaskDatabaseDataExportPayload"></a>

### TaskDatabaseDataExportPayload
TaskDatabaseDataExportPayload is the task payload for database data export.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| spec_id | [string](#string) |  | common fields |
| sheet_id | [int32](#int32) |  |  |
| password | [string](#string) |  |  |
| format | [ExportFormat](#bytebase-store-ExportFormat) |  |  |






<a name="bytebase-store-TaskDatabaseUpdatePayload"></a>

### TaskDatabaseUpdatePayload
TaskDatabaseUpdatePayload is the task payload for updating database (DDL &amp; DML).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| skipped | [bool](#bool) |  | common fields |
| skipped_reason | [string](#string) |  |  |
| spec_id | [string](#string) |  |  |
| schema_version | [string](#string) |  |  |
| sheet_id | [int32](#int32) |  |  |
| pre_update_backup_detail | [PreUpdateBackupDetail](#bytebase-store-PreUpdateBackupDetail) |  |  |
| flags | [TaskDatabaseUpdatePayload.FlagsEntry](#bytebase-store-TaskDatabaseUpdatePayload-FlagsEntry) | repeated | flags is used for ghost sync |
| task_release_source | [TaskReleaseSource](#bytebase-store-TaskReleaseSource) |  |  |






<a name="bytebase-store-TaskDatabaseUpdatePayload-FlagsEntry"></a>

### TaskDatabaseUpdatePayload.FlagsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-store-TaskReleaseSource"></a>

### TaskReleaseSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file | [string](#string) |  | Format: projects/{project}/releases/{release}/files/{id} |





 

 

 

 



<a name="store_task_run-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/task_run.proto



<a name="bytebase-store-PriorBackupDetail"></a>

### PriorBackupDetail



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| items | [PriorBackupDetail.Item](#bytebase-store-PriorBackupDetail-Item) | repeated |  |






<a name="bytebase-store-PriorBackupDetail-Item"></a>

### PriorBackupDetail.Item



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| source_table | [PriorBackupDetail.Item.Table](#bytebase-store-PriorBackupDetail-Item-Table) |  | The original table information. |
| target_table | [PriorBackupDetail.Item.Table](#bytebase-store-PriorBackupDetail-Item-Table) |  | The target backup table information. |
| start_position | [Position](#bytebase-store-Position) |  |  |
| end_position | [Position](#bytebase-store-Position) |  |  |






<a name="bytebase-store-PriorBackupDetail-Item-Table"></a>

### PriorBackupDetail.Item.Table



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [string](#string) |  | The database information. Format: instances/{instance}/databases/{database} |
| schema | [string](#string) |  |  |
| table | [string](#string) |  |  |






<a name="bytebase-store-SchedulerInfo"></a>

### SchedulerInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| report_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| waiting_cause | [SchedulerInfo.WaitingCause](#bytebase-store-SchedulerInfo-WaitingCause) |  |  |






<a name="bytebase-store-SchedulerInfo-WaitingCause"></a>

### SchedulerInfo.WaitingCause



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| connection_limit | [bool](#bool) |  |  |
| task_uid | [int32](#int32) |  |  |






<a name="bytebase-store-TaskRunResult"></a>

### TaskRunResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| detail | [string](#string) |  |  |
| change_history | [string](#string) |  | Format: instances/{instance}/databases/{database}/changeHistories/{changeHistory} |
| version | [string](#string) |  |  |
| start_position | [TaskRunResult.Position](#bytebase-store-TaskRunResult-Position) |  |  |
| end_position | [TaskRunResult.Position](#bytebase-store-TaskRunResult-Position) |  |  |
| export_archive_uid | [int32](#int32) |  | The uid of the export archive. |
| prior_backup_detail | [PriorBackupDetail](#bytebase-store-PriorBackupDetail) |  | The prior backup detail that will be used to rollback the task run. |






<a name="bytebase-store-TaskRunResult-Position"></a>

### TaskRunResult.Position
The following fields are used for error reporting.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| line | [int32](#int32) |  |  |
| column | [int32](#int32) |  |  |





 

 

 

 



<a name="store_task_run_log-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/task_run_log.proto



<a name="bytebase-store-TaskRunLog"></a>

### TaskRunLog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [TaskRunLog.Type](#bytebase-store-TaskRunLog-Type) |  |  |
| deploy_id | [string](#string) |  |  |
| schema_dump_start | [TaskRunLog.SchemaDumpStart](#bytebase-store-TaskRunLog-SchemaDumpStart) |  |  |
| schema_dump_end | [TaskRunLog.SchemaDumpEnd](#bytebase-store-TaskRunLog-SchemaDumpEnd) |  |  |
| command_execute | [TaskRunLog.CommandExecute](#bytebase-store-TaskRunLog-CommandExecute) |  |  |
| command_response | [TaskRunLog.CommandResponse](#bytebase-store-TaskRunLog-CommandResponse) |  |  |
| database_sync_start | [TaskRunLog.DatabaseSyncStart](#bytebase-store-TaskRunLog-DatabaseSyncStart) |  |  |
| database_sync_end | [TaskRunLog.DatabaseSyncEnd](#bytebase-store-TaskRunLog-DatabaseSyncEnd) |  |  |
| task_run_status_update | [TaskRunLog.TaskRunStatusUpdate](#bytebase-store-TaskRunLog-TaskRunStatusUpdate) |  |  |
| transaction_control | [TaskRunLog.TransactionControl](#bytebase-store-TaskRunLog-TransactionControl) |  |  |
| prior_backup_start | [TaskRunLog.PriorBackupStart](#bytebase-store-TaskRunLog-PriorBackupStart) |  |  |
| prior_backup_end | [TaskRunLog.PriorBackupEnd](#bytebase-store-TaskRunLog-PriorBackupEnd) |  |  |






<a name="bytebase-store-TaskRunLog-CommandExecute"></a>

### TaskRunLog.CommandExecute



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| command_indexes | [int32](#int32) | repeated | The indexes of the executed commands. |






<a name="bytebase-store-TaskRunLog-CommandResponse"></a>

### TaskRunLog.CommandResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| command_indexes | [int32](#int32) | repeated | The indexes of the executed commands. |
| error | [string](#string) |  |  |
| affected_rows | [int32](#int32) |  |  |
| all_affected_rows | [int32](#int32) | repeated | `all_affected_rows` is the affected rows of each command. `all_affected_rows` may be unavailable if the database driver doesn&#39;t support it. Caller should fallback to `affected_rows` in that case. |






<a name="bytebase-store-TaskRunLog-DatabaseSyncEnd"></a>

### TaskRunLog.DatabaseSyncEnd



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| error | [string](#string) |  |  |






<a name="bytebase-store-TaskRunLog-DatabaseSyncStart"></a>

### TaskRunLog.DatabaseSyncStart







<a name="bytebase-store-TaskRunLog-PriorBackupEnd"></a>

### TaskRunLog.PriorBackupEnd



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| prior_backup_detail | [PriorBackupDetail](#bytebase-store-PriorBackupDetail) |  |  |
| error | [string](#string) |  |  |






<a name="bytebase-store-TaskRunLog-PriorBackupStart"></a>

### TaskRunLog.PriorBackupStart







<a name="bytebase-store-TaskRunLog-SchemaDumpEnd"></a>

### TaskRunLog.SchemaDumpEnd



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| error | [string](#string) |  |  |






<a name="bytebase-store-TaskRunLog-SchemaDumpStart"></a>

### TaskRunLog.SchemaDumpStart







<a name="bytebase-store-TaskRunLog-TaskRunStatusUpdate"></a>

### TaskRunLog.TaskRunStatusUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [TaskRunLog.TaskRunStatusUpdate.Status](#bytebase-store-TaskRunLog-TaskRunStatusUpdate-Status) |  |  |






<a name="bytebase-store-TaskRunLog-TransactionControl"></a>

### TaskRunLog.TransactionControl



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [TaskRunLog.TransactionControl.Type](#bytebase-store-TaskRunLog-TransactionControl-Type) |  |  |
| error | [string](#string) |  |  |





 


<a name="bytebase-store-TaskRunLog-TaskRunStatusUpdate-Status"></a>

### TaskRunLog.TaskRunStatusUpdate.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| RUNNING_WAITING | 1 | the task run is ready to be executed by the scheduler |
| RUNNING_RUNNING | 2 | the task run is being executed by the scheduler |



<a name="bytebase-store-TaskRunLog-TransactionControl-Type"></a>

### TaskRunLog.TransactionControl.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| BEGIN | 1 |  |
| COMMIT | 2 |  |
| ROLLBACK | 3 |  |



<a name="bytebase-store-TaskRunLog-Type"></a>

### TaskRunLog.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| SCHEMA_DUMP_START | 1 |  |
| SCHEMA_DUMP_END | 2 |  |
| COMMAND_EXECUTE | 3 |  |
| COMMAND_RESPONSE | 4 |  |
| DATABASE_SYNC_START | 5 |  |
| DATABASE_SYNC_END | 6 |  |
| TASK_RUN_STATUS_UPDATE | 7 |  |
| TRANSACTION_CONTROL | 8 |  |
| PRIOR_BACKUP_START | 9 |  |
| PRIOR_BACKUP_END | 10 |  |


 

 

 



<a name="store_user-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/user.proto



<a name="bytebase-store-MFAConfig"></a>

### MFAConfig
MFAConfig is the MFA configuration for a user.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| otp_secret | [string](#string) |  | The otp_secret is the secret key used to validate the OTP code. |
| temp_otp_secret | [string](#string) |  | The temp_otp_secret is the temporary secret key used to validate the OTP code and will replace the otp_secret in two phase commits. |
| recovery_codes | [string](#string) | repeated | The recovery_codes are the codes that can be used to recover the account. |
| temp_recovery_codes | [string](#string) | repeated | The temp_recovery_codes are the temporary codes that will replace the recovery_codes in two phase commits. |






<a name="bytebase-store-UserProfile"></a>

### UserProfile



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| last_login_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| last_change_password_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| source | [string](#string) |  | source means where the user comes from. For now we support Entra ID SCIM sync, so the source could be Entra ID. |





 

 

 

 



<a name="store_vcs-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/vcs.proto



<a name="bytebase-store-VCSConnector"></a>

### VCSConnector



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  | The title or display name of the VCS connector. |
| full_path | [string](#string) |  | Full path from the corresponding VCS provider. For GitLab, this is the project full path. e.g. group1/project-1 |
| web_url | [string](#string) |  | Web url from the corresponding VCS provider. For GitLab, this is the project web url. e.g. https://gitlab.example.com/group1/project-1 |
| branch | [string](#string) |  | Branch to listen to. |
| base_directory | [string](#string) |  | Base working directory we are interested. |
| external_id | [string](#string) |  | Repository id from the corresponding VCS provider. For GitLab, this is the project id. e.g. 123 |
| external_webhook_id | [string](#string) |  | Push webhook id from the corresponding VCS provider. For GitLab, this is the project webhook id. e.g. 123 |
| webhook_secret_token | [string](#string) |  | For GitLab, webhook request contains this in the &#39;X-Gitlab-Token&#34; header and we compare it with the one stored in db to validate it sends to the expected endpoint. |
| database_group | [string](#string) |  | Apply changes to the database group. Optional, if not set, will apply changes to all databases in the project. Format: projects/{project}/databaseGroups/{databaseGroup} |





 

 

 

 



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

