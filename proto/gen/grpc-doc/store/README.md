# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [store/common.proto](#store_common-proto)
    - [PageToken](#bytebase-store-PageToken)
    - [Position](#bytebase-store-Position)
    - [Range](#bytebase-store-Range)
  
    - [Engine](#bytebase-store-Engine)
    - [ExportFormat](#bytebase-store-ExportFormat)
    - [RiskLevel](#bytebase-store-RiskLevel)
    - [SchemaChangeType](#bytebase-store-SchemaChangeType)
    - [VCSType](#bytebase-store-VCSType)
    - [WebhookType](#bytebase-store-WebhookType)
  
- [store/advice.proto](#store_advice-proto)
    - [Advice](#bytebase-store-Advice)
  
    - [Advice.Status](#bytebase-store-Advice-Status)
  
- [store/approval.proto](#store_approval-proto)
    - [ApprovalFlow](#bytebase-store-ApprovalFlow)
    - [ApprovalTemplate](#bytebase-store-ApprovalTemplate)
    - [IssuePayloadApproval](#bytebase-store-IssuePayloadApproval)
    - [IssuePayloadApproval.Approver](#bytebase-store-IssuePayloadApproval-Approver)
  
    - [IssuePayloadApproval.Approver.Status](#bytebase-store-IssuePayloadApproval-Approver-Status)
  
- [store/audit_log.proto](#store_audit_log-proto)
    - [AuditLog](#bytebase-store-AuditLog)
    - [RequestMetadata](#bytebase-store-RequestMetadata)
  
    - [AuditLog.Severity](#bytebase-store-AuditLog-Severity)
  
- [store/changelog.proto](#store_changelog-proto)
    - [ChangelogPayload](#bytebase-store-ChangelogPayload)
  
    - [ChangelogPayload.Type](#bytebase-store-ChangelogPayload-Type)
  
- [store/database.proto](#store_database-proto)
    - [BoundingBox](#bytebase-store-BoundingBox)
    - [CheckConstraintMetadata](#bytebase-store-CheckConstraintMetadata)
    - [ColumnCatalog](#bytebase-store-ColumnCatalog)
    - [ColumnCatalog.LabelsEntry](#bytebase-store-ColumnCatalog-LabelsEntry)
    - [ColumnMetadata](#bytebase-store-ColumnMetadata)
    - [DatabaseConfig](#bytebase-store-DatabaseConfig)
    - [DatabaseMetadata](#bytebase-store-DatabaseMetadata)
    - [DatabaseMetadata.LabelsEntry](#bytebase-store-DatabaseMetadata-LabelsEntry)
    - [DatabaseSchemaMetadata](#bytebase-store-DatabaseSchemaMetadata)
    - [DependencyColumn](#bytebase-store-DependencyColumn)
    - [DependencyTable](#bytebase-store-DependencyTable)
    - [DimensionalConfig](#bytebase-store-DimensionalConfig)
    - [EnumTypeMetadata](#bytebase-store-EnumTypeMetadata)
    - [EventMetadata](#bytebase-store-EventMetadata)
    - [EventTriggerMetadata](#bytebase-store-EventTriggerMetadata)
    - [ExcludeConstraintMetadata](#bytebase-store-ExcludeConstraintMetadata)
    - [ExtensionMetadata](#bytebase-store-ExtensionMetadata)
    - [ExternalTableMetadata](#bytebase-store-ExternalTableMetadata)
    - [ForeignKeyMetadata](#bytebase-store-ForeignKeyMetadata)
    - [FunctionMetadata](#bytebase-store-FunctionMetadata)
    - [GenerationMetadata](#bytebase-store-GenerationMetadata)
    - [GridLevel](#bytebase-store-GridLevel)
    - [IndexMetadata](#bytebase-store-IndexMetadata)
    - [InstanceRoleMetadata](#bytebase-store-InstanceRoleMetadata)
    - [LinkedDatabaseMetadata](#bytebase-store-LinkedDatabaseMetadata)
    - [MaterializedViewMetadata](#bytebase-store-MaterializedViewMetadata)
    - [ObjectSchema](#bytebase-store-ObjectSchema)
    - [ObjectSchema.ArrayKind](#bytebase-store-ObjectSchema-ArrayKind)
    - [ObjectSchema.StructKind](#bytebase-store-ObjectSchema-StructKind)
    - [ObjectSchema.StructKind.PropertiesEntry](#bytebase-store-ObjectSchema-StructKind-PropertiesEntry)
    - [PackageMetadata](#bytebase-store-PackageMetadata)
    - [ProcedureMetadata](#bytebase-store-ProcedureMetadata)
    - [RuleMetadata](#bytebase-store-RuleMetadata)
    - [SchemaCatalog](#bytebase-store-SchemaCatalog)
    - [SchemaMetadata](#bytebase-store-SchemaMetadata)
    - [SequenceMetadata](#bytebase-store-SequenceMetadata)
    - [SpatialIndexConfig](#bytebase-store-SpatialIndexConfig)
    - [SpatialIndexConfig.EngineSpecificEntry](#bytebase-store-SpatialIndexConfig-EngineSpecificEntry)
    - [StorageConfig](#bytebase-store-StorageConfig)
    - [StreamMetadata](#bytebase-store-StreamMetadata)
    - [TableCatalog](#bytebase-store-TableCatalog)
    - [TableMetadata](#bytebase-store-TableMetadata)
    - [TablePartitionMetadata](#bytebase-store-TablePartitionMetadata)
    - [TaskMetadata](#bytebase-store-TaskMetadata)
    - [TessellationConfig](#bytebase-store-TessellationConfig)
    - [TriggerMetadata](#bytebase-store-TriggerMetadata)
    - [ViewMetadata](#bytebase-store-ViewMetadata)
  
    - [ColumnMetadata.IdentityGeneration](#bytebase-store-ColumnMetadata-IdentityGeneration)
    - [GenerationMetadata.Type](#bytebase-store-GenerationMetadata-Type)
    - [ObjectSchema.Type](#bytebase-store-ObjectSchema-Type)
    - [StreamMetadata.Mode](#bytebase-store-StreamMetadata-Mode)
    - [StreamMetadata.Type](#bytebase-store-StreamMetadata-Type)
    - [TablePartitionMetadata.Type](#bytebase-store-TablePartitionMetadata-Type)
    - [TaskMetadata.State](#bytebase-store-TaskMetadata-State)
  
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
    - [LDAPIdentityProviderConfig.SecurityProtocol](#bytebase-store-LDAPIdentityProviderConfig-SecurityProtocol)
    - [OAuth2AuthStyle](#bytebase-store-OAuth2AuthStyle)
  
- [store/instance.proto](#store_instance-proto)
    - [DataSource](#bytebase-store-DataSource)
    - [DataSource.AWSCredential](#bytebase-store-DataSource-AWSCredential)
    - [DataSource.Address](#bytebase-store-DataSource-Address)
    - [DataSource.AzureCredential](#bytebase-store-DataSource-AzureCredential)
    - [DataSource.ExtraConnectionParametersEntry](#bytebase-store-DataSource-ExtraConnectionParametersEntry)
    - [DataSource.GCPCredential](#bytebase-store-DataSource-GCPCredential)
    - [DataSourceExternalSecret](#bytebase-store-DataSourceExternalSecret)
    - [DataSourceExternalSecret.AppRoleAuthOption](#bytebase-store-DataSourceExternalSecret-AppRoleAuthOption)
    - [Instance](#bytebase-store-Instance)
    - [Instance.LabelsEntry](#bytebase-store-Instance-LabelsEntry)
    - [InstanceRole](#bytebase-store-InstanceRole)
    - [KerberosConfig](#bytebase-store-KerberosConfig)
    - [SASLConfig](#bytebase-store-SASLConfig)
  
    - [DataSource.AuthenticationType](#bytebase-store-DataSource-AuthenticationType)
    - [DataSource.RedisType](#bytebase-store-DataSource-RedisType)
    - [DataSourceExternalSecret.AppRoleAuthOption.SecretType](#bytebase-store-DataSourceExternalSecret-AppRoleAuthOption-SecretType)
    - [DataSourceExternalSecret.AuthType](#bytebase-store-DataSourceExternalSecret-AuthType)
    - [DataSourceExternalSecret.SecretType](#bytebase-store-DataSourceExternalSecret-SecretType)
    - [DataSourceType](#bytebase-store-DataSourceType)
  
- [store/issue.proto](#store_issue-proto)
    - [GrantRequest](#bytebase-store-GrantRequest)
    - [Issue](#bytebase-store-Issue)
  
    - [Issue.Status](#bytebase-store-Issue-Status)
    - [Issue.Type](#bytebase-store-Issue-Type)
  
- [store/issue_comment.proto](#store_issue_comment-proto)
    - [IssueCommentPayload](#bytebase-store-IssueCommentPayload)
    - [IssueCommentPayload.Approval](#bytebase-store-IssueCommentPayload-Approval)
    - [IssueCommentPayload.IssueUpdate](#bytebase-store-IssueCommentPayload-IssueUpdate)
    - [IssueCommentPayload.PlanSpecUpdate](#bytebase-store-IssueCommentPayload-PlanSpecUpdate)
  
- [store/oauth2.proto](#store_oauth2-proto)
    - [OAuth2AuthorizationCodeConfig](#bytebase-store-OAuth2AuthorizationCodeConfig)
    - [OAuth2ClientConfig](#bytebase-store-OAuth2ClientConfig)
  
- [store/plan.proto](#store_plan-proto)
    - [PlanConfig](#bytebase-store-PlanConfig)
    - [PlanConfig.ChangeDatabaseConfig](#bytebase-store-PlanConfig-ChangeDatabaseConfig)
    - [PlanConfig.ChangeDatabaseConfig.GhostFlagsEntry](#bytebase-store-PlanConfig-ChangeDatabaseConfig-GhostFlagsEntry)
    - [PlanConfig.CreateDatabaseConfig](#bytebase-store-PlanConfig-CreateDatabaseConfig)
    - [PlanConfig.ExportDataConfig](#bytebase-store-PlanConfig-ExportDataConfig)
    - [PlanConfig.Spec](#bytebase-store-PlanConfig-Spec)
  
- [store/plan_check_run.proto](#store_plan_check_run-proto)
    - [ChangedResourceDatabase](#bytebase-store-ChangedResourceDatabase)
    - [ChangedResourceSchema](#bytebase-store-ChangedResourceSchema)
    - [ChangedResourceTable](#bytebase-store-ChangedResourceTable)
    - [ChangedResources](#bytebase-store-ChangedResources)
    - [PlanCheckRunResult](#bytebase-store-PlanCheckRunResult)
    - [PlanCheckRunResult.Result](#bytebase-store-PlanCheckRunResult-Result)
    - [PlanCheckRunResult.Result.SqlReviewReport](#bytebase-store-PlanCheckRunResult-Result-SqlReviewReport)
    - [PlanCheckRunResult.Result.SqlSummaryReport](#bytebase-store-PlanCheckRunResult-Result-SqlSummaryReport)
  
    - [PlanCheckType](#bytebase-store-PlanCheckType)
  
- [store/policy.proto](#store_policy-proto)
    - [Binding](#bytebase-store-Binding)
    - [DataSourceQueryPolicy](#bytebase-store-DataSourceQueryPolicy)
    - [EnvironmentTierPolicy](#bytebase-store-EnvironmentTierPolicy)
    - [IamPolicy](#bytebase-store-IamPolicy)
    - [MaskingExemptionPolicy](#bytebase-store-MaskingExemptionPolicy)
    - [MaskingExemptionPolicy.Exemption](#bytebase-store-MaskingExemptionPolicy-Exemption)
    - [MaskingRulePolicy](#bytebase-store-MaskingRulePolicy)
    - [MaskingRulePolicy.MaskingRule](#bytebase-store-MaskingRulePolicy-MaskingRule)
    - [Policy](#bytebase-store-Policy)
    - [QueryDataPolicy](#bytebase-store-QueryDataPolicy)
    - [RolloutPolicy](#bytebase-store-RolloutPolicy)
    - [TagPolicy](#bytebase-store-TagPolicy)
    - [TagPolicy.TagsEntry](#bytebase-store-TagPolicy-TagsEntry)
  
    - [DataSourceQueryPolicy.Restriction](#bytebase-store-DataSourceQueryPolicy-Restriction)
    - [EnvironmentTierPolicy.EnvironmentTier](#bytebase-store-EnvironmentTierPolicy-EnvironmentTier)
    - [Policy.Resource](#bytebase-store-Policy-Resource)
    - [Policy.Type](#bytebase-store-Policy-Type)
  
- [store/project.proto](#store_project-proto)
    - [Label](#bytebase-store-Label)
    - [Project](#bytebase-store-Project)
    - [Project.ExecutionRetryPolicy](#bytebase-store-Project-ExecutionRetryPolicy)
    - [Project.LabelsEntry](#bytebase-store-Project-LabelsEntry)
  
- [store/project_webhook.proto](#store_project_webhook-proto)
    - [Activity](#bytebase-store-Activity)
    - [ProjectWebhook](#bytebase-store-ProjectWebhook)
  
    - [Activity.Type](#bytebase-store-Activity-Type)
  
- [store/query_history.proto](#store_query_history-proto)
    - [QueryHistoryPayload](#bytebase-store-QueryHistoryPayload)
  
- [store/release.proto](#store_release-proto)
    - [ReleasePayload](#bytebase-store-ReleasePayload)
    - [ReleasePayload.File](#bytebase-store-ReleasePayload-File)
    - [ReleasePayload.VCSSource](#bytebase-store-ReleasePayload-VCSSource)
  
- [store/review_config.proto](#store_review_config-proto)
    - [ReviewConfigPayload](#bytebase-store-ReviewConfigPayload)
    - [SQLReviewRule](#bytebase-store-SQLReviewRule)
    - [SQLReviewRule.CommentConventionRulePayload](#bytebase-store-SQLReviewRule-CommentConventionRulePayload)
    - [SQLReviewRule.NamingCaseRulePayload](#bytebase-store-SQLReviewRule-NamingCaseRulePayload)
    - [SQLReviewRule.NamingRulePayload](#bytebase-store-SQLReviewRule-NamingRulePayload)
    - [SQLReviewRule.NumberRulePayload](#bytebase-store-SQLReviewRule-NumberRulePayload)
    - [SQLReviewRule.StringArrayRulePayload](#bytebase-store-SQLReviewRule-StringArrayRulePayload)
    - [SQLReviewRule.StringRulePayload](#bytebase-store-SQLReviewRule-StringRulePayload)
  
    - [SQLReviewRule.Level](#bytebase-store-SQLReviewRule-Level)
    - [SQLReviewRule.Type](#bytebase-store-SQLReviewRule-Type)
  
- [store/revision.proto](#store_revision-proto)
    - [RevisionPayload](#bytebase-store-RevisionPayload)
  
- [store/role.proto](#store_role-proto)
    - [RolePermissions](#bytebase-store-RolePermissions)
  
- [store/setting.proto](#store_setting-proto)
    - [AISetting](#bytebase-store-AISetting)
    - [Algorithm](#bytebase-store-Algorithm)
    - [Algorithm.FullMask](#bytebase-store-Algorithm-FullMask)
    - [Algorithm.InnerOuterMask](#bytebase-store-Algorithm-InnerOuterMask)
    - [Algorithm.MD5Mask](#bytebase-store-Algorithm-MD5Mask)
    - [Algorithm.RangeMask](#bytebase-store-Algorithm-RangeMask)
    - [Algorithm.RangeMask.Slice](#bytebase-store-Algorithm-RangeMask-Slice)
    - [AppIMSetting](#bytebase-store-AppIMSetting)
    - [AppIMSetting.DingTalk](#bytebase-store-AppIMSetting-DingTalk)
    - [AppIMSetting.Feishu](#bytebase-store-AppIMSetting-Feishu)
    - [AppIMSetting.IMSetting](#bytebase-store-AppIMSetting-IMSetting)
    - [AppIMSetting.Lark](#bytebase-store-AppIMSetting-Lark)
    - [AppIMSetting.Slack](#bytebase-store-AppIMSetting-Slack)
    - [AppIMSetting.Teams](#bytebase-store-AppIMSetting-Teams)
    - [AppIMSetting.Wecom](#bytebase-store-AppIMSetting-Wecom)
    - [DataClassificationSetting](#bytebase-store-DataClassificationSetting)
    - [DataClassificationSetting.DataClassificationConfig](#bytebase-store-DataClassificationSetting-DataClassificationConfig)
    - [DataClassificationSetting.DataClassificationConfig.ClassificationEntry](#bytebase-store-DataClassificationSetting-DataClassificationConfig-ClassificationEntry)
    - [DataClassificationSetting.DataClassificationConfig.DataClassification](#bytebase-store-DataClassificationSetting-DataClassificationConfig-DataClassification)
    - [DataClassificationSetting.DataClassificationConfig.Level](#bytebase-store-DataClassificationSetting-DataClassificationConfig-Level)
    - [EnvironmentSetting](#bytebase-store-EnvironmentSetting)
    - [EnvironmentSetting.Environment](#bytebase-store-EnvironmentSetting-Environment)
    - [EnvironmentSetting.Environment.TagsEntry](#bytebase-store-EnvironmentSetting-Environment-TagsEntry)
    - [SemanticTypeSetting](#bytebase-store-SemanticTypeSetting)
    - [SemanticTypeSetting.SemanticType](#bytebase-store-SemanticTypeSetting-SemanticType)
    - [SystemSetting](#bytebase-store-SystemSetting)
    - [WorkspaceApprovalSetting](#bytebase-store-WorkspaceApprovalSetting)
    - [WorkspaceApprovalSetting.Rule](#bytebase-store-WorkspaceApprovalSetting-Rule)
    - [WorkspaceProfileSetting](#bytebase-store-WorkspaceProfileSetting)
    - [WorkspaceProfileSetting.Announcement](#bytebase-store-WorkspaceProfileSetting-Announcement)
    - [WorkspaceProfileSetting.PasswordRestriction](#bytebase-store-WorkspaceProfileSetting-PasswordRestriction)
  
    - [AISetting.Provider](#bytebase-store-AISetting-Provider)
    - [Algorithm.InnerOuterMask.MaskType](#bytebase-store-Algorithm-InnerOuterMask-MaskType)
    - [SettingName](#bytebase-store-SettingName)
    - [WorkspaceApprovalSetting.Rule.Source](#bytebase-store-WorkspaceApprovalSetting-Rule-Source)
    - [WorkspaceProfileSetting.Announcement.AlertLevel](#bytebase-store-WorkspaceProfileSetting-Announcement-AlertLevel)
    - [WorkspaceProfileSetting.DatabaseChangeMode](#bytebase-store-WorkspaceProfileSetting-DatabaseChangeMode)
  
- [store/signal.proto](#store_signal-proto)
    - [Signal](#bytebase-store-Signal)
  
    - [Signal.Type](#bytebase-store-Signal-Type)
  
- [store/task.proto](#store_task-proto)
    - [Task](#bytebase-store-Task)
    - [Task.FlagsEntry](#bytebase-store-Task-FlagsEntry)
  
    - [Task.Type](#bytebase-store-Task-Type)
  
- [store/task_run.proto](#store_task_run-proto)
    - [SchedulerInfo](#bytebase-store-SchedulerInfo)
    - [SchedulerInfo.WaitingCause](#bytebase-store-SchedulerInfo-WaitingCause)
    - [TaskRun](#bytebase-store-TaskRun)
    - [TaskRunResult](#bytebase-store-TaskRunResult)
  
    - [TaskRun.Status](#bytebase-store-TaskRun-Status)
  
- [store/task_run_log.proto](#store_task_run_log-proto)
    - [PriorBackupDetail](#bytebase-store-PriorBackupDetail)
    - [PriorBackupDetail.Item](#bytebase-store-PriorBackupDetail-Item)
    - [PriorBackupDetail.Item.Table](#bytebase-store-PriorBackupDetail-Item-Table)
    - [TaskRunLog](#bytebase-store-TaskRunLog)
    - [TaskRunLog.CommandExecute](#bytebase-store-TaskRunLog-CommandExecute)
    - [TaskRunLog.CommandResponse](#bytebase-store-TaskRunLog-CommandResponse)
    - [TaskRunLog.ComputeDiffEnd](#bytebase-store-TaskRunLog-ComputeDiffEnd)
    - [TaskRunLog.ComputeDiffStart](#bytebase-store-TaskRunLog-ComputeDiffStart)
    - [TaskRunLog.DatabaseSyncEnd](#bytebase-store-TaskRunLog-DatabaseSyncEnd)
    - [TaskRunLog.DatabaseSyncStart](#bytebase-store-TaskRunLog-DatabaseSyncStart)
    - [TaskRunLog.PriorBackupEnd](#bytebase-store-TaskRunLog-PriorBackupEnd)
    - [TaskRunLog.PriorBackupStart](#bytebase-store-TaskRunLog-PriorBackupStart)
    - [TaskRunLog.ReleaseFileExecute](#bytebase-store-TaskRunLog-ReleaseFileExecute)
    - [TaskRunLog.RetryInfo](#bytebase-store-TaskRunLog-RetryInfo)
    - [TaskRunLog.SchemaDumpEnd](#bytebase-store-TaskRunLog-SchemaDumpEnd)
    - [TaskRunLog.SchemaDumpStart](#bytebase-store-TaskRunLog-SchemaDumpStart)
    - [TaskRunLog.TransactionControl](#bytebase-store-TaskRunLog-TransactionControl)
  
    - [TaskRunLog.TransactionControl.Type](#bytebase-store-TaskRunLog-TransactionControl-Type)
    - [TaskRunLog.Type](#bytebase-store-TaskRunLog-Type)
  
- [store/user.proto](#store_user-proto)
    - [MFAConfig](#bytebase-store-MFAConfig)
    - [UserProfile](#bytebase-store-UserProfile)
    - [WorkloadIdentityConfig](#bytebase-store-WorkloadIdentityConfig)
  
    - [PrincipalType](#bytebase-store-PrincipalType)
    - [WorkloadIdentityConfig.ProviderType](#bytebase-store-WorkloadIdentityConfig-ProviderType)
  
- [store/worksheet.proto](#store_worksheet-proto)
    - [WorkSheetOrganizerPayload](#bytebase-store-WorkSheetOrganizerPayload)
  
- [Scalar Value Types](#scalar-value-types)



<a name="store_common-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/common.proto



<a name="bytebase-store-PageToken"></a>

### PageToken
PageToken is used internally for obfuscating pagination tokens.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| limit | [int32](#int32) |  | Maximum number of items to return. |
| offset | [int32](#int32) |  | Number of items to skip before starting to return results. |






<a name="bytebase-store-Position"></a>

### Position
Position in a text expressed as one-based line and one-based column.
We use 1-based numbering to match the majority of industry standards:
- Monaco Editor uses 1-based (https://microsoft.github.io/monaco-editor/typedoc/interfaces/IPosition.html)
- GitHub Actions uses 1-based (https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#setting-an-error-message)
- Most text editors display 1-based positions to users
Note: LSP uses 0-based (https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#position),
but we follow the canonical user-facing standards.

Handling unknown positions:
- If the entire position is unknown, leave this field as nil/undefined
- If only line is known, set line and leave column as 0 (e.g., line=5, column=0)
- If only column is known (rare), set column and leave line as 0
Frontends should check for nil/undefined/zero values and handle them appropriately.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| line | [int32](#int32) |  | Line position in a text (one-based). First line of the text is line 1. A value of 0 indicates the line information is unknown. |
| column | [int32](#int32) |  | Column position in a text (one-based). Column is measured in Unicode code points (characters/runes), not bytes or grapheme clusters. First character of the line is column 1. A value of 0 indicates the column information is unknown.

Examples: - &#34;SELECT * FROM t&#34; - column 8 is &#39;*&#39; - &#34;SELECT 你好 FROM t&#34; - column 8 is &#39;你&#39; (even though it&#39;s at byte offset 7) - &#34;SELECT 😀 FROM t&#34; - column 8 is &#39;😀&#39; (even though it&#39;s 4 bytes in UTF-8) |






<a name="bytebase-store-Range"></a>

### Range
Range represents a span within a text or sequence.
Whether the indices are byte offsets or character indices depends on the context.
Check the documentation of the field using Range for specific semantics.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start | [int32](#int32) |  | Start index (inclusive). |
| end | [int32](#int32) |  | End index (exclusive). |





 


<a name="bytebase-store-Engine"></a>

### Engine
Engine represents the type of database system.

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
| STARROCKS | 15 |  |
| DORIS | 16 |  |
| HIVE | 17 |  |
| ELASTICSEARCH | 18 |  |
| BIGQUERY | 19 |  |
| DYNAMODB | 20 |  |
| DATABRICKS | 21 |  |
| COCKROACHDB | 22 |  |
| COSMOSDB | 23 |  |
| TRINO | 24 |  |
| CASSANDRA | 25 |  |



<a name="bytebase-store-ExportFormat"></a>

### ExportFormat
ExportFormat represents the file format for exported data.

| Name | Number | Description |
| ---- | ------ | ----------- |
| FORMAT_UNSPECIFIED | 0 |  |
| CSV | 1 |  |
| JSON | 2 |  |
| SQL | 3 |  |
| XLSX | 4 |  |



<a name="bytebase-store-RiskLevel"></a>

### RiskLevel
RiskLevel represents the assessed risk level of a database operation.

| Name | Number | Description |
| ---- | ------ | ----------- |
| RISK_LEVEL_UNSPECIFIED | 0 |  |
| LOW | 1 |  |
| MODERATE | 2 |  |
| HIGH | 3 |  |



<a name="bytebase-store-SchemaChangeType"></a>

### SchemaChangeType
SchemaChangeType represents the strategy for schema changes.

| Name | Number | Description |
| ---- | ------ | ----------- |
| SCHEMA_CHANGE_TYPE_UNSPECIFIED | 0 |  |
| VERSIONED | 1 | Versioned migration with explicit migration scripts. |
| DECLARATIVE | 2 | Declarative schema definition (state-based). |



<a name="bytebase-store-VCSType"></a>

### VCSType
VCSType represents the type of version control system.

| Name | Number | Description |
| ---- | ------ | ----------- |
| VCS_TYPE_UNSPECIFIED | 0 |  |
| GITHUB | 1 |  |
| GITLAB | 2 |  |
| BITBUCKET | 3 |  |
| AZURE_DEVOPS | 4 |  |



<a name="bytebase-store-WebhookType"></a>

### WebhookType


| Name | Number | Description |
| ---- | ------ | ----------- |
| WEBHOOK_TYPE_UNSPECIFIED | 0 | Unspecified type. |
| SLACK | 1 | Slack integration. |
| DISCORD | 2 | Discord integration. |
| TEAMS | 3 | Microsoft Teams integration. |
| DINGTALK | 4 | DingTalk integration. |
| FEISHU | 5 | Feishu integration. |
| WECOM | 6 | WeCom (WeChat Work) integration. |
| LARK | 7 | Lark integration. |


 

 

 



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
| start_position | [Position](#bytebase-store-Position) |  | The start_position is inclusive and the end_position is exclusive. TODO: use range instead. |
| end_position | [Position](#bytebase-store-Position) |  |  |





 


<a name="bytebase-store-Advice-Status"></a>

### Advice.Status
Status represents the result status of the advice.

| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| SUCCESS | 1 |  |
| WARNING | 2 |  |
| ERROR | 3 |  |


 

 

 



<a name="store_approval-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/approval.proto



<a name="bytebase-store-ApprovalFlow"></a>

### ApprovalFlow
ApprovalFlow defines the sequence of approvals required.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| roles | [string](#string) | repeated | List of role names that must approve, in order. |






<a name="bytebase-store-ApprovalTemplate"></a>

### ApprovalTemplate
ApprovalTemplate defines the approval workflow and requirements for an issue.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| flow | [ApprovalFlow](#bytebase-store-ApprovalFlow) |  | The approval workflow specification. |
| title | [string](#string) |  | Human-readable title of the approval template. |
| description | [string](#string) |  | Detailed description of when this template applies. |






<a name="bytebase-store-IssuePayloadApproval"></a>

### IssuePayloadApproval
IssuePayloadApproval records the approval template used and approval history for an issue.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| approval_template | [ApprovalTemplate](#bytebase-store-ApprovalTemplate) |  | The approval template being used for this issue. |
| approvers | [IssuePayloadApproval.Approver](#bytebase-store-IssuePayloadApproval-Approver) | repeated | List of approvers and their current status. |
| approval_finding_done | [bool](#bool) |  | Whether the system has finished finding a matching approval template. False means the backend is still searching for matching templates. |






<a name="bytebase-store-IssuePayloadApproval-Approver"></a>

### IssuePayloadApproval.Approver
Approver represents a user who can approve or reject an issue.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [IssuePayloadApproval.Approver.Status](#bytebase-store-IssuePayloadApproval-Approver-Status) |  | The current approval status. |
| principal | [string](#string) |  | The principal who is the approver. Format: users/{email}. |





 


<a name="bytebase-store-IssuePayloadApproval-Approver-Status"></a>

### IssuePayloadApproval.Approver.Status
Status represents the approver&#39;s decision state.

| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 | Approval is pending from this approver. |
| APPROVED | 2 | Approver has approved the issue. |
| REJECTED | 3 | Approver has rejected the issue. |


 

 

 



<a name="store_audit_log-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/audit_log.proto



<a name="bytebase-store-AuditLog"></a>

### AuditLog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The project or workspace the audit log belongs to. Formats: - projects/{project} - workspaces/{workspace} |
| method | [string](#string) |  | Example: /bytebase.v1.SQLService/Query |
| resource | [string](#string) |  | The resource name. Example: projects/{project} |
| user | [string](#string) |  | Format: users/{email}. |
| severity | [AuditLog.Severity](#bytebase-store-AuditLog-Severity) |  |  |
| request | [string](#string) |  | Marshalled request. |
| response | [string](#string) |  | Marshalled response. Some fields are omitted because they are too large or contain sensitive information. |
| status | [google.rpc.Status](#google-rpc-Status) |  |  |
| latency | [google.protobuf.Duration](#google-protobuf-Duration) |  | The latency of the RPC. |
| service_data | [google.protobuf.Any](#google-protobuf-Any) |  | The service-specific data about the request, response, and other activities. |
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
| SEVERITY_UNSPECIFIED | 0 |  |
| DEBUG | 1 |  |
| INFO | 2 |  |
| NOTICE | 3 |  |
| WARNING | 4 |  |
| ERROR | 5 |  |
| CRITICAL | 6 |  |
| ALERT | 7 |  |
| EMERGENCY | 8 |  |


 

 

 



<a name="store_changelog-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/changelog.proto



<a name="bytebase-store-ChangelogPayload"></a>

### ChangelogPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| task_run | [string](#string) |  | Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |
| type | [ChangelogPayload.Type](#bytebase-store-ChangelogPayload-Type) |  |  |
| git_commit | [string](#string) |  |  |
| dump_version | [int32](#int32) |  | Dump format version for drift detection reliability. Engine-specific version stored when baseline/migration is created. 0 = legacy changelog (pre-versioning). |





 


<a name="bytebase-store-ChangelogPayload-Type"></a>

### ChangelogPayload.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| BASELINE | 1 |  |
| MIGRATE | 2 |  |
| SDL | 3 |  |


 

 

 



<a name="store_database-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/database.proto



<a name="bytebase-store-BoundingBox"></a>

### BoundingBox
BoundingBox defines the bounding box for spatial indexes.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| xmin | [double](#double) |  |  |
| ymin | [double](#double) |  |  |
| xmax | [double](#double) |  |  |
| ymax | [double](#double) |  |  |






<a name="bytebase-store-CheckConstraintMetadata"></a>

### CheckConstraintMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the check constraint. |
| expression | [string](#string) |  | The expression is the expression of a check constraint. |






<a name="bytebase-store-ColumnCatalog"></a>

### ColumnCatalog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the column. |
| semantic_type | [string](#string) |  |  |
| labels | [ColumnCatalog.LabelsEntry](#bytebase-store-ColumnCatalog-LabelsEntry) | repeated | The user labels for a column. |
| classification | [string](#string) |  |  |
| object_schema | [ObjectSchema](#bytebase-store-ObjectSchema) | optional |  |






<a name="bytebase-store-ColumnCatalog-LabelsEntry"></a>

### ColumnCatalog.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-store-ColumnMetadata"></a>

### ColumnMetadata
ColumnMetadata is the metadata for columns.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the column. |
| position | [int32](#int32) |  | The position is the position in columns. |
| default | [string](#string) |  | The default value of the column. |
| default_on_null | [bool](#bool) |  | Oracle specific metadata. The default_on_null is the default on null of a column. |
| on_update | [string](#string) |  | The on_update is the on update action of a column. For MySQL like databases, it&#39;s only supported for TIMESTAMP columns with CURRENT_TIMESTAMP as on update value. |
| nullable | [bool](#bool) |  | The nullable is the nullable of a column. |
| type | [string](#string) |  | The type is the type of a column. |
| character_set | [string](#string) |  | The character_set is the character_set of a column. |
| collation | [string](#string) |  | The collation is the collation of a column. |
| comment | [string](#string) |  | The comment is the comment of a column. |
| generation | [GenerationMetadata](#bytebase-store-GenerationMetadata) |  | The generation is for generated columns. |
| is_identity | [bool](#bool) |  |  |
| identity_generation | [ColumnMetadata.IdentityGeneration](#bytebase-store-ColumnMetadata-IdentityGeneration) |  | The identity_generation is for identity columns, PG only. |
| identity_seed | [int64](#int64) |  | The identity_seed is for identity columns, MSSQL only. |
| identity_increment | [int64](#int64) |  | The identity_increment is for identity columns, MSSQL only. |
| default_constraint_name | [string](#string) |  | The default_constraint_name is the name of the default constraint, MSSQL only. In MSSQL, default values are implemented as named constraints. When modifying or dropping a column&#39;s default value, you must reference the constraint by name. This field stores the actual constraint name from the database.

Example: A column definition like: CREATE TABLE employees ( status NVARCHAR(20) DEFAULT &#39;active&#39; )

Will create a constraint with an auto-generated name like &#39;DF__employees__statu__3B75D760&#39; or a user-defined name if specified: ALTER TABLE employees ADD CONSTRAINT DF_employees_status DEFAULT &#39;active&#39; FOR status

To modify the default, you must first drop the existing constraint by name: ALTER TABLE employees DROP CONSTRAINT DF__employees__statu__3B75D760 ALTER TABLE employees ADD CONSTRAINT DF_employees_status DEFAULT &#39;inactive&#39; FOR status

This field is populated when syncing from the database. When empty (e.g., when parsing from SQL files), the system cannot automatically drop the constraint. |






<a name="bytebase-store-DatabaseConfig"></a>

### DatabaseConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| schemas | [SchemaCatalog](#bytebase-store-SchemaCatalog) | repeated | The schema_configs is the list of configs for schemas in a database. |






<a name="bytebase-store-DatabaseMetadata"></a>

### DatabaseMetadata
DatabaseMetadata is the metadata for databases.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| labels | [DatabaseMetadata.LabelsEntry](#bytebase-store-DatabaseMetadata-LabelsEntry) | repeated |  |
| last_sync_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| backup_available | [bool](#bool) |  |  |
| datashare | [bool](#bool) |  |  |
| drifted | [bool](#bool) |  | The schema has drifted from the source of truth. |
| version | [string](#string) |  | The version of database schema. |






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
| schemas | [SchemaMetadata](#bytebase-store-SchemaMetadata) | repeated | The list of schemas in a database. |
| character_set | [string](#string) |  | The character set of the database. |
| collation | [string](#string) |  | The collation of the database. |
| extensions | [ExtensionMetadata](#bytebase-store-ExtensionMetadata) | repeated | The list of extensions in a database. |
| datashare | [bool](#bool) |  | The database belongs to a datashare. |
| service_name | [string](#string) |  | The service name of the database. It&#39;s an Oracle-specific concept. |
| linked_databases | [LinkedDatabaseMetadata](#bytebase-store-LinkedDatabaseMetadata) | repeated |  |
| owner | [string](#string) |  |  |
| search_path | [string](#string) |  | The search_path is the search path of a PostgreSQL database. |
| event_triggers | [EventTriggerMetadata](#bytebase-store-EventTriggerMetadata) | repeated | The list of event triggers in a database (PostgreSQL specific). Event triggers are database-level objects, not schema-scoped. |






<a name="bytebase-store-DependencyColumn"></a>

### DependencyColumn
DependencyColumn is the metadata for dependency columns.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  | The schema is the schema of a reference column. |
| table | [string](#string) |  | The table is the table of a reference column. |
| column | [string](#string) |  | The column is the name of a reference column. |






<a name="bytebase-store-DependencyTable"></a>

### DependencyTable



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  | The schema is the schema of a reference table. |
| table | [string](#string) |  | The table is the name of a reference table. |






<a name="bytebase-store-DimensionalConfig"></a>

### DimensionalConfig
DimensionalConfig defines dimensional and constraint parameters for spatial indexes.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| dimensions | [int32](#int32) |  | Number of dimensions (2-4, default 2) |
| data_type | [string](#string) |  | Spatial data type Examples: GEOMETRY, GEOGRAPHY, POINT, POLYGON, etc. |
| operator_class | [string](#string) |  | PostgreSQL operator class Examples: gist_geometry_ops_2d, gist_geometry_ops_nd, etc. |
| layer_gtype | [string](#string) |  | Oracle geometry type constraint Examples: POINT, LINE, POLYGON, COLLECTION |
| parallel_build | [bool](#bool) |  | Parallel index creation |






<a name="bytebase-store-EnumTypeMetadata"></a>

### EnumTypeMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the enum type. |
| values | [string](#string) | repeated | The enum values of the type. |
| comment | [string](#string) |  |  |
| skip_dump | [bool](#bool) |  |  |






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
| comment | [string](#string) |  |  |






<a name="bytebase-store-EventTriggerMetadata"></a>

### EventTriggerMetadata
EventTriggerMetadata is the metadata for PostgreSQL event triggers.
Event triggers are database-level objects that fire on DDL events.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the event trigger. |
| event | [string](#string) |  | The event type: DDL_COMMAND_START, DDL_COMMAND_END, SQL_DROP, TABLE_REWRITE. |
| tags | [string](#string) | repeated | The tags filter (e.g., [&#39;CREATE TABLE&#39;, &#39;DROP TABLE&#39;]). |
| function_schema | [string](#string) |  | The schema of the function to execute. |
| function_name | [string](#string) |  | The name of the function to execute. |
| enabled | [bool](#bool) |  | Whether the trigger is enabled. |
| definition | [string](#string) |  | The full CREATE EVENT TRIGGER definition from pg_get_event_trigger_def(). SDL output should prefer using this field. |
| comment | [string](#string) |  | The comment on the event trigger. |
| skip_dump | [bool](#bool) |  | Skip dump flag (for extension-owned triggers). |






<a name="bytebase-store-ExcludeConstraintMetadata"></a>

### ExcludeConstraintMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the EXCLUDE constraint. |
| expression | [string](#string) |  | The expression is the full EXCLUDE constraint definition including &#34;EXCLUDE&#34; keyword. Example: &#34;EXCLUDE USING gist (room_id WITH =, during WITH &amp;&amp;)&#34; |






<a name="bytebase-store-ExtensionMetadata"></a>

### ExtensionMetadata
ExtensionMetadata is the metadata for extensions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the extension. |
| schema | [string](#string) |  | The schema where the extension is installed. However, the extension usage is not limited to the schema. |
| version | [string](#string) |  | The version is the version of an extension. |
| description | [string](#string) |  | The description is the description of an extension. |






<a name="bytebase-store-ExternalTableMetadata"></a>

### ExternalTableMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the external table. |
| external_server_name | [string](#string) |  | The external_server_name is the name of the external server. |
| external_database_name | [string](#string) |  | The external_database_name is the name of the external database. |
| columns | [ColumnMetadata](#bytebase-store-ColumnMetadata) | repeated | The columns is the ordered list of columns in a foreign table. |






<a name="bytebase-store-ForeignKeyMetadata"></a>

### ForeignKeyMetadata
ForeignKeyMetadata is the metadata for foreign keys.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the foreign key. |
| columns | [string](#string) | repeated | The columns are the ordered referencing columns of a foreign key. |
| referenced_schema | [string](#string) |  | The referenced_schema is the referenced schema name of a foreign key. It is an empty string for databases without such concept such as MySQL. |
| referenced_table | [string](#string) |  | The referenced_table is the referenced table name of a foreign key. |
| referenced_columns | [string](#string) | repeated | The referenced_columns are the ordered referenced columns of a foreign key. |
| on_delete | [string](#string) |  | The on_delete is the on delete action of a foreign key. |
| on_update | [string](#string) |  | The on_update is the on update action of a foreign key. |
| match_type | [string](#string) |  | The match_type is the match type of a foreign key. The match_type is the PostgreSQL specific field. It&#39;s empty string for other databases. |






<a name="bytebase-store-FunctionMetadata"></a>

### FunctionMetadata
FunctionMetadata is the metadata for functions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the function. |
| definition | [string](#string) |  | The definition is the definition of a function. |
| signature | [string](#string) |  | The signature is the name with the number and type of input arguments the function takes. |
| character_set_client | [string](#string) |  | MySQL specific metadata. |
| collation_connection | [string](#string) |  |  |
| database_collation | [string](#string) |  |  |
| sql_mode | [string](#string) |  |  |
| comment | [string](#string) |  |  |
| dependency_tables | [DependencyTable](#bytebase-store-DependencyTable) | repeated | The dependency_tables is the list of dependency tables of a function. For PostgreSQL, it&#39;s the list of tables that the function depends on the return type definition. |
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-store-GenerationMetadata"></a>

### GenerationMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [GenerationMetadata.Type](#bytebase-store-GenerationMetadata-Type) |  |  |
| expression | [string](#string) |  |  |






<a name="bytebase-store-GridLevel"></a>

### GridLevel
GridLevel defines a grid level for spatial tessellation.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| level | [int32](#int32) |  | 1-4 for SQL Server |
| density | [string](#string) |  | LOW, MEDIUM, HIGH |






<a name="bytebase-store-IndexMetadata"></a>

### IndexMetadata
IndexMetadata is the metadata for indexes.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the index. |
| expressions | [string](#string) | repeated | The expressions are the ordered columns or expressions of an index. This could refer to a column or an expression. |
| key_length | [int64](#int64) | repeated | The ordered list of key lengths for the index. If the key length is not specified, it is -1. |
| descending | [bool](#bool) | repeated | The ordered list of descending flags for the index columns. |
| type | [string](#string) |  | The type is the type of an index. |
| unique | [bool](#bool) |  | The unique is whether the index is unique. |
| primary | [bool](#bool) |  | The primary is whether the index is a primary key index. |
| visible | [bool](#bool) |  | The visible is whether the index is visible. |
| comment | [string](#string) |  | The comment is the comment of an index. |
| definition | [string](#string) |  | The definition of an index. |
| parent_index_schema | [string](#string) |  | The schema name of the parent index. |
| parent_index_name | [string](#string) |  | The index name of the parent index. |
| granularity | [int64](#int64) |  | The number of granules in the block. It&#39;s a ClickHouse specific field. |
| is_constraint | [bool](#bool) |  | It&#39;s a PostgreSQL specific field. The unique constraint and unique index are not the same thing in PostgreSQL. |
| spatial_config | [SpatialIndexConfig](#bytebase-store-SpatialIndexConfig) |  | Spatial index specific configuration |
| opclass_names | [string](#string) | repeated | https://www.postgresql.org/docs/current/catalog-pg-opclass.html Name of the operator class for each column. (PostgreSQL specific). |
| opclass_defaults | [bool](#bool) | repeated | True if the operator class is the default. (PostgreSQL specific). |






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
| name | [string](#string) |  | The name of the materialized view. |
| definition | [string](#string) |  | The definition is the definition of a view. |
| comment | [string](#string) |  | The comment is the comment of a view. |
| dependency_columns | [DependencyColumn](#bytebase-store-DependencyColumn) | repeated | The list of dependency columns of the view. |
| triggers | [TriggerMetadata](#bytebase-store-TriggerMetadata) | repeated | The ordered list of columns in the materialized view. |
| indexes | [IndexMetadata](#bytebase-store-IndexMetadata) | repeated | The list of indexes in the materialized view. |
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-store-ObjectSchema"></a>

### ObjectSchema



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [ObjectSchema.Type](#bytebase-store-ObjectSchema-Type) |  |  |
| struct_kind | [ObjectSchema.StructKind](#bytebase-store-ObjectSchema-StructKind) |  |  |
| array_kind | [ObjectSchema.ArrayKind](#bytebase-store-ObjectSchema-ArrayKind) |  |  |
| semantic_type | [string](#string) |  |  |






<a name="bytebase-store-ObjectSchema-ArrayKind"></a>

### ObjectSchema.ArrayKind



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| kind | [ObjectSchema](#bytebase-store-ObjectSchema) |  |  |






<a name="bytebase-store-ObjectSchema-StructKind"></a>

### ObjectSchema.StructKind



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| properties | [ObjectSchema.StructKind.PropertiesEntry](#bytebase-store-ObjectSchema-StructKind-PropertiesEntry) | repeated |  |






<a name="bytebase-store-ObjectSchema-StructKind-PropertiesEntry"></a>

### ObjectSchema.StructKind.PropertiesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [ObjectSchema](#bytebase-store-ObjectSchema) |  |  |






<a name="bytebase-store-PackageMetadata"></a>

### PackageMetadata
PackageMetadata is the metadata for packages.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the package. |
| definition | [string](#string) |  | The definition is the definition of a package. |






<a name="bytebase-store-ProcedureMetadata"></a>

### ProcedureMetadata
ProcedureMetadata is the metadata for procedures.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the procedure. |
| definition | [string](#string) |  | The definition is the definition of a procedure. |
| signature | [string](#string) |  | The signature is the name with the number and type of input arguments the function takes. |
| character_set_client | [string](#string) |  | MySQL specific metadata. |
| collation_connection | [string](#string) |  |  |
| database_collation | [string](#string) |  |  |
| sql_mode | [string](#string) |  |  |
| comment | [string](#string) |  |  |
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-store-RuleMetadata"></a>

### RuleMetadata
RuleMetadata is the metadata for PostgreSQL rules.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the rule. |
| event | [string](#string) |  | The event type of the rule: SELECT, INSERT, UPDATE, or DELETE. |
| condition | [string](#string) |  | The WHERE condition of the rule (optional). |
| action | [string](#string) |  | The command(s) to execute when the rule fires. |
| is_instead | [bool](#bool) |  | The is_instead indicates whether this is an INSTEAD rule. |
| is_enabled | [bool](#bool) |  | The is_enabled indicates whether the rule is enabled. |
| definition | [string](#string) |  | The full CREATE RULE statement. |






<a name="bytebase-store-SchemaCatalog"></a>

### SchemaCatalog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The schema name. It is an empty string for databases without such concept such as MySQL. |
| tables | [TableCatalog](#bytebase-store-TableCatalog) | repeated | The table_configs is the list of configs for tables in a schema. |






<a name="bytebase-store-SchemaMetadata"></a>

### SchemaMetadata
SchemaMetadata is the metadata for schemas.
This is the concept of schema in Postgres, but it&#39;s a no-op for MySQL.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The schema name. It is an empty string for databases without such concept such as MySQL. |
| tables | [TableMetadata](#bytebase-store-TableMetadata) | repeated | The list of tables in a schema. |
| external_tables | [ExternalTableMetadata](#bytebase-store-ExternalTableMetadata) | repeated | The list of external tables in a schema. |
| views | [ViewMetadata](#bytebase-store-ViewMetadata) | repeated | The list of views in a schema. |
| functions | [FunctionMetadata](#bytebase-store-FunctionMetadata) | repeated | The list of functions in a schema. |
| procedures | [ProcedureMetadata](#bytebase-store-ProcedureMetadata) | repeated | The list of procedures in a schema. |
| streams | [StreamMetadata](#bytebase-store-StreamMetadata) | repeated | The list of streams in a schema, currently only used for Snowflake. |
| tasks | [TaskMetadata](#bytebase-store-TaskMetadata) | repeated | The list of tasks in a schema, currently only used for Snowflake. |
| materialized_views | [MaterializedViewMetadata](#bytebase-store-MaterializedViewMetadata) | repeated | The list of materialized views in a schema. |
| sequences | [SequenceMetadata](#bytebase-store-SequenceMetadata) | repeated | The list of sequences in a schema. |
| packages | [PackageMetadata](#bytebase-store-PackageMetadata) | repeated | The list of packages in a schema. |
| owner | [string](#string) |  |  |
| comment | [string](#string) |  |  |
| events | [EventMetadata](#bytebase-store-EventMetadata) | repeated |  |
| enum_types | [EnumTypeMetadata](#bytebase-store-EnumTypeMetadata) | repeated |  |
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-store-SequenceMetadata"></a>

### SequenceMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of a sequence. |
| data_type | [string](#string) |  | The data type of a sequence. |
| start | [string](#string) |  | The start value of a sequence. |
| min_value | [string](#string) |  | The minimum value of a sequence. |
| max_value | [string](#string) |  | The maximum value of a sequence. |
| increment | [string](#string) |  | The increment value of a sequence. |
| cycle | [bool](#bool) |  | Whether the sequence cycles. |
| cache_size | [string](#string) |  | Cache size of a sequence. |
| last_value | [string](#string) |  | The last value of a sequence. |
| owner_table | [string](#string) |  | The table that owns the sequence. |
| owner_column | [string](#string) |  | The column that owns the sequence. |
| comment | [string](#string) |  |  |
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-store-SpatialIndexConfig"></a>

### SpatialIndexConfig
SpatialIndexConfig is the configuration for spatial indexes across different database engines.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| method | [string](#string) |  | Index method/type (database-specific) Examples: &#34;SPATIAL&#34; (MySQL/SQL Server), &#34;GIST&#34;/&#34;SPGIST&#34; (PostgreSQL), &#34;MDSYS.SPATIAL_INDEX_V2&#34; (Oracle) |
| tessellation | [TessellationConfig](#bytebase-store-TessellationConfig) |  | Tessellation configuration (primarily SQL Server) |
| storage | [StorageConfig](#bytebase-store-StorageConfig) |  | Storage and performance parameters |
| dimensional | [DimensionalConfig](#bytebase-store-DimensionalConfig) |  | Dimensional and constraint parameters |
| engine_specific | [SpatialIndexConfig.EngineSpecificEntry](#bytebase-store-SpatialIndexConfig-EngineSpecificEntry) | repeated | Database-specific parameters (stored as key-value pairs for extensibility) |






<a name="bytebase-store-SpatialIndexConfig-EngineSpecificEntry"></a>

### SpatialIndexConfig.EngineSpecificEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-store-StorageConfig"></a>

### StorageConfig
StorageConfig defines storage and performance parameters for spatial indexes.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fillfactor | [int32](#int32) |  | PostgreSQL parameters

10-100 |
| buffering | [string](#string) |  | auto, on, off |
| tablespace | [string](#string) |  | Oracle parameters |
| work_tablespace | [string](#string) |  |  |
| sdo_level | [int32](#int32) |  |  |
| commit_interval | [int32](#int32) |  |  |
| pad_index | [bool](#bool) |  | SQL Server parameters |
| sort_in_tempdb | [string](#string) |  | ON, OFF |
| drop_existing | [bool](#bool) |  |  |
| online | [bool](#bool) |  |  |
| allow_row_locks | [bool](#bool) |  |  |
| allow_page_locks | [bool](#bool) |  |  |
| maxdop | [int32](#int32) |  |  |
| data_compression | [string](#string) |  | NONE, ROW, PAGE |






<a name="bytebase-store-StreamMetadata"></a>

### StreamMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the stream. |
| table_name | [string](#string) |  | The table_name is the name of the table/view that the stream is created on. |
| owner | [string](#string) |  | The owner of the stream. |
| comment | [string](#string) |  | The comment of the stream. |
| type | [StreamMetadata.Type](#bytebase-store-StreamMetadata-Type) |  | The type of the stream. |
| stale | [bool](#bool) |  | Indicates whether the stream was last read before the `stale_after` time. |
| mode | [StreamMetadata.Mode](#bytebase-store-StreamMetadata-Mode) |  | The mode of the stream. |
| definition | [string](#string) |  | The definition of the stream. |






<a name="bytebase-store-TableCatalog"></a>

### TableCatalog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the table. |
| columns | [ColumnCatalog](#bytebase-store-ColumnCatalog) | repeated | The column_configs is the ordered list of configs for columns in a table. |
| object_schema | [ObjectSchema](#bytebase-store-ObjectSchema) | optional |  |
| classification | [string](#string) |  |  |






<a name="bytebase-store-TableMetadata"></a>

### TableMetadata
TableMetadata is the metadata for tables.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the table. |
| columns | [ColumnMetadata](#bytebase-store-ColumnMetadata) | repeated | The columns is the ordered list of columns in a table. |
| indexes | [IndexMetadata](#bytebase-store-IndexMetadata) | repeated | The indexes is the list of indexes in a table. |
| engine | [string](#string) |  | The engine is the engine of a table. |
| collation | [string](#string) |  | The collation is the collation of a table. |
| charset | [string](#string) |  | The character set of the table. |
| row_count | [int64](#int64) |  | The row_count is the estimated number of rows of a table. |
| data_size | [int64](#int64) |  | The data_size is the estimated data size of a table. |
| index_size | [int64](#int64) |  | The index_size is the estimated index size of a table. |
| data_free | [int64](#int64) |  | The data_free is the estimated free data size of a table. |
| create_options | [string](#string) |  | The create_options is the create option of a table. |
| comment | [string](#string) |  | The comment is the comment of a table. |
| foreign_keys | [ForeignKeyMetadata](#bytebase-store-ForeignKeyMetadata) | repeated | The foreign_keys is the list of foreign keys in a table. |
| partitions | [TablePartitionMetadata](#bytebase-store-TablePartitionMetadata) | repeated | The partitions is the list of partitions in a table. |
| check_constraints | [CheckConstraintMetadata](#bytebase-store-CheckConstraintMetadata) | repeated | The check_constraints is the list of check constraints in a table. |
| owner | [string](#string) |  |  |
| sorting_keys | [string](#string) | repeated | The sorting_keys is a tuple of column names or arbitrary expressions. ClickHouse specific field. Reference: https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/mergetree#order_by |
| triggers | [TriggerMetadata](#bytebase-store-TriggerMetadata) | repeated |  |
| skip_dump | [bool](#bool) |  |  |
| rules | [RuleMetadata](#bytebase-store-RuleMetadata) | repeated | The rules is the list of rules in a table (PostgreSQL specific). |
| sharding_info | [string](#string) |  | https://docs.pingcap.com/tidb/stable/information-schema-tables/ |
| primary_key_type | [string](#string) |  | https://docs.pingcap.com/tidb/stable/clustered-indexes/#clustered-indexes CLUSTERED or NONCLUSTERED. |
| exclude_constraints | [ExcludeConstraintMetadata](#bytebase-store-ExcludeConstraintMetadata) | repeated | The exclude_constraints is the list of EXCLUDE constraints in a table (PostgreSQL specific). |






<a name="bytebase-store-TablePartitionMetadata"></a>

### TablePartitionMetadata
TablePartitionMetadata is the metadata for table partitions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the table partition. |
| type | [TablePartitionMetadata.Type](#bytebase-store-TablePartitionMetadata-Type) |  | The type of a table partition. |
| expression | [string](#string) |  | The expression is the expression of a table partition. For PostgreSQL, the expression is the text of {FOR VALUES partition_bound_spec}, see https://www.postgresql.org/docs/current/sql-createtable.html. For MySQL, the expression is the `expr` or `column_list` of the following syntax. PARTITION BY { [LINEAR] HASH(expr) | [LINEAR] KEY [ALGORITHM={1 | 2}] (column_list) | RANGE{(expr) | COLUMNS(column_list)} | LIST{(expr) | COLUMNS(column_list)} }. |
| value | [string](#string) |  | The value is the value of a table partition. For MySQL, the value is for RANGE and LIST partition types, - For a RANGE partition, it contains the value set in the partition&#39;s VALUES LESS THAN clause, which can be either an integer or MAXVALUE. - For a LIST partition, this column contains the values defined in the partition&#39;s VALUES IN clause, which is a list of comma-separated integer values. - For others, it&#39;s an empty string. |
| use_default | [string](#string) |  | The use_default is whether the users use the default partition, it stores the different value for different database engines. For MySQL, it&#39;s [INT] type, 0 means not use default partition, otherwise, it&#39;s equals to number in syntax [SUB]PARTITION {number}. |
| subpartitions | [TablePartitionMetadata](#bytebase-store-TablePartitionMetadata) | repeated | The subpartitions is the list of subpartitions in a table partition. |
| indexes | [IndexMetadata](#bytebase-store-IndexMetadata) | repeated |  |
| check_constraints | [CheckConstraintMetadata](#bytebase-store-CheckConstraintMetadata) | repeated |  |
| exclude_constraints | [ExcludeConstraintMetadata](#bytebase-store-ExcludeConstraintMetadata) | repeated |  |






<a name="bytebase-store-TaskMetadata"></a>

### TaskMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the task. |
| id | [string](#string) |  | The Snowflake-generated ID of the task. Example: 01ad32a0-1bb6-5e93-0000-000000000001. |
| owner | [string](#string) |  | The owner of the task. |
| comment | [string](#string) |  | The comment of the task. |
| warehouse | [string](#string) |  | The warehouse of the task. |
| schedule | [string](#string) |  | The schedule interval of the task. |
| predecessors | [string](#string) | repeated | The predecessor tasks of the task. |
| state | [TaskMetadata.State](#bytebase-store-TaskMetadata-State) |  | The state of the task. |
| condition | [string](#string) |  | The condition of the task. |
| definition | [string](#string) |  | The definition of the task. |






<a name="bytebase-store-TessellationConfig"></a>

### TessellationConfig
TessellationConfig defines tessellation parameters for spatial indexes.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| scheme | [string](#string) |  | Tessellation scheme Examples: GEOMETRY_GRID, GEOGRAPHY_GRID, GEOMETRY_AUTO_GRID, GEOGRAPHY_AUTO_GRID |
| bounding_box | [BoundingBox](#bytebase-store-BoundingBox) |  | Bounding box for GEOMETRY indexes (SQL Server) |
| grid_levels | [GridLevel](#bytebase-store-GridLevel) | repeated | Grid level configuration (SQL Server) |
| cells_per_object | [int32](#int32) |  | Cells per object (SQL Server) |






<a name="bytebase-store-TriggerMetadata"></a>

### TriggerMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the trigger. |
| event | [string](#string) |  | The event that triggers this action, such as INSERT, UPDATE, DELETE, or TRUNCATE. |
| timing | [string](#string) |  | The timing of when the trigger fires, such as BEFORE or AFTER. |
| body | [string](#string) |  | The body of the trigger. |
| sql_mode | [string](#string) |  |  |
| character_set_client | [string](#string) |  |  |
| collation_connection | [string](#string) |  |  |
| comment | [string](#string) |  |  |
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-store-ViewMetadata"></a>

### ViewMetadata
ViewMetadata is the metadata for views.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the view. |
| definition | [string](#string) |  | The definition is the definition of a view. |
| comment | [string](#string) |  | The comment is the comment of a view. |
| dependency_columns | [DependencyColumn](#bytebase-store-DependencyColumn) | repeated | The list of dependency columns of a view. |
| columns | [ColumnMetadata](#bytebase-store-ColumnMetadata) | repeated | The ordered list of columns in the view. |
| triggers | [TriggerMetadata](#bytebase-store-TriggerMetadata) | repeated | The list of triggers in the view. |
| skip_dump | [bool](#bool) |  |  |
| rules | [RuleMetadata](#bytebase-store-RuleMetadata) | repeated | The rules is the list of rules in a view (PostgreSQL specific). |





 


<a name="bytebase-store-ColumnMetadata-IdentityGeneration"></a>

### ColumnMetadata.IdentityGeneration


| Name | Number | Description |
| ---- | ------ | ----------- |
| IDENTITY_GENERATION_UNSPECIFIED | 0 |  |
| ALWAYS | 1 |  |
| BY_DEFAULT | 2 |  |



<a name="bytebase-store-GenerationMetadata-Type"></a>

### GenerationMetadata.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| TYPE_VIRTUAL | 1 |  |
| TYPE_STORED | 2 |  |



<a name="bytebase-store-ObjectSchema-Type"></a>

### ObjectSchema.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| STRING | 1 |  |
| NUMBER | 2 |  |
| BOOLEAN | 3 |  |
| OBJECT | 4 |  |
| ARRAY | 5 |  |



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
The type is the type of a table partition. Some database engines may not
support all types. Only available for the following database engines now:
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
| member | [string](#string) |  | Member is the principal who belongs to this group.

Format: users/{email}. |
| role | [GroupMember.Role](#bytebase-store-GroupMember-Role) |  |  |






<a name="bytebase-store-GroupPayload"></a>

### GroupPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| members | [GroupMember](#bytebase-store-GroupMember) | repeated |  |
| source | [string](#string) |  | The source indicates where the group comes from. For now we support Entra ID SCIM sync, so the source could be Entra ID. |





 


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


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identifier | [string](#string) |  | Identifier is the field name of the unique identifier in 3rd-party idp user info. Required. |
| display_name | [string](#string) |  | DisplayName is the field name of display name in 3rd-party idp user info. Optional. |
| phone | [string](#string) |  | Phone is the field name of primary phone in 3rd-party idp user info. Optional. |
| groups | [string](#string) |  | Groups is the field name of groups in 3rd-party idp user info. Optional. Mainly used for OIDC: https://developer.okta.com/docs/guides/customize-tokens-groups-claim/main/ |






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
| phone | [string](#string) |  | Phone is the value of primary phone in 3rd-party idp user info. |
| groups | [string](#string) | repeated | Groups is the value of groups in 3rd-party idp user info. Mainly used for OIDC: https://developer.okta.com/docs/guides/customize-tokens-groups-claim/main/ |
| has_groups | [bool](#bool) |  |  |






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
| security_protocol | [LDAPIdentityProviderConfig.SecurityProtocol](#bytebase-store-LDAPIdentityProviderConfig-SecurityProtocol) |  | SecurityProtocol is the security protocol to be used for establishing connections with the LDAP server. |
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
| scopes | [string](#string) | repeated |  |
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



<a name="bytebase-store-LDAPIdentityProviderConfig-SecurityProtocol"></a>

### LDAPIdentityProviderConfig.SecurityProtocol


| Name | Number | Description |
| ---- | ------ | ----------- |
| SECURITY_PROTOCOL_UNSPECIFIED | 0 |  |
| START_TLS | 1 | StartTLS is the security protocol that starts with an unencrypted connection and then upgrades to TLS. |
| LDAPS | 2 | LDAPS is the security protocol that uses TLS from the beginning. |



<a name="bytebase-store-OAuth2AuthStyle"></a>

### OAuth2AuthStyle


| Name | Number | Description |
| ---- | ------ | ----------- |
| OAUTH2_AUTH_STYLE_UNSPECIFIED | 0 |  |
| IN_PARAMS | 1 | IN_PARAMS sends the &#34;client_id&#34; and &#34;client_secret&#34; in the POST body as application/x-www-form-urlencoded parameters. |
| IN_HEADER | 2 | IN_HEADER sends the client_id and client_secret using HTTP Basic Authorization. This is an optional style described in the OAuth2 RFC 6749 section 2.3.1. |


 

 

 



<a name="store_instance-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/instance.proto



<a name="bytebase-store-DataSource"></a>

### DataSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| type | [DataSourceType](#bytebase-store-DataSourceType) |  |  |
| username | [string](#string) |  |  |
| password | [string](#string) |  |  |
| obfuscated_password | [string](#string) |  |  |
| use_ssl | [bool](#bool) |  | Use SSL to connect to the data source. By default, we use the system&#39;s SSL configuration. |
| ssl_ca | [string](#string) |  |  |
| obfuscated_ssl_ca | [string](#string) |  |  |
| ssl_cert | [string](#string) |  |  |
| obfuscated_ssl_cert | [string](#string) |  |  |
| ssl_key | [string](#string) |  |  |
| obfuscated_ssl_key | [string](#string) |  |  |
| verify_tls_certificate | [bool](#bool) |  | verify_tls_certificate enables TLS certificate verification for SSL connections. Default is false (no verification) for backward compatibility. Set to true for secure connections (recommended for production). Only set to false for development or when certificates cannot be properly validated (e.g., self-signed certs, VPN environments). |
| host | [string](#string) |  |  |
| port | [string](#string) |  |  |
| database | [string](#string) |  |  |
| srv | [bool](#bool) |  | srv, authentication_database, and replica_set are used for MongoDB. srv is a boolean flag that indicates whether the host is a DNS SRV record. |
| authentication_database | [string](#string) |  | authentication_database is the database name to authenticate against, which stores the user credentials. |
| replica_set | [string](#string) |  | replica_set is used for MongoDB replica set. |
| sid | [string](#string) |  | sid and service_name are used for Oracle. |
| service_name | [string](#string) |  |  |
| ssh_host | [string](#string) |  | SSH related The hostname of the SSH server agent. |
| ssh_port | [string](#string) |  | The port of the SSH server agent. It&#39;s 22 typically. |
| ssh_user | [string](#string) |  | The user to login the server. |
| ssh_password | [string](#string) |  | The password to login the server. If it&#39;s empty string, no password is required. |
| obfuscated_ssh_password | [string](#string) |  |  |
| ssh_private_key | [string](#string) |  | The private key to login the server. If it&#39;s empty string, we will use the system default private key from os.Getenv(&#34;SSH_AUTH_SOCK&#34;). |
| obfuscated_ssh_private_key | [string](#string) |  |  |
| authentication_private_key | [string](#string) |  | PKCS#8 private key in PEM format. If it&#39;s empty string, no private key is required. Used for authentication when connecting to the data source. |
| obfuscated_authentication_private_key | [string](#string) |  |  |
| authentication_private_key_passphrase | [string](#string) |  | Passphrase for the encrypted PKCS#8 private key. Only used when the private key is encrypted. |
| obfuscated_authentication_private_key_passphrase | [string](#string) |  |  |
| external_secret | [DataSourceExternalSecret](#bytebase-store-DataSourceExternalSecret) |  |  |
| authentication_type | [DataSource.AuthenticationType](#bytebase-store-DataSource-AuthenticationType) |  |  |
| azure_credential | [DataSource.AzureCredential](#bytebase-store-DataSource-AzureCredential) |  |  |
| aws_credential | [DataSource.AWSCredential](#bytebase-store-DataSource-AWSCredential) |  |  |
| gcp_credential | [DataSource.GCPCredential](#bytebase-store-DataSource-GCPCredential) |  |  |
| sasl_config | [SASLConfig](#bytebase-store-SASLConfig) |  |  |
| additional_addresses | [DataSource.Address](#bytebase-store-DataSource-Address) | repeated | additional_addresses is used for MongoDB replica set. |
| direct_connection | [bool](#bool) |  | direct_connection is used for MongoDB to dispatch all the operations to the node specified in the connection string. |
| region | [string](#string) |  | Region is the location of the database, used for AWS RDS. For example, us-east-1. |
| warehouse_id | [string](#string) |  | warehouse_id is used by Databricks. |
| master_name | [string](#string) |  | master_name is the master name used by connecting redis-master via redis sentinel. |
| master_username | [string](#string) |  | master_username and master_obfuscated_password are master credentials used by redis sentinel mode. |
| master_password | [string](#string) |  |  |
| obfuscated_master_password | [string](#string) |  |  |
| redis_type | [DataSource.RedisType](#bytebase-store-DataSource-RedisType) |  |  |
| cluster | [string](#string) |  | Cluster is the cluster name for the data source. Used by CockroachDB. |
| extra_connection_parameters | [DataSource.ExtraConnectionParametersEntry](#bytebase-store-DataSource-ExtraConnectionParametersEntry) | repeated | Extra connection parameters for the database connection. For PostgreSQL HA, this can be used to set target_session_attrs=read-write |






<a name="bytebase-store-DataSource-AWSCredential"></a>

### DataSource.AWSCredential



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| access_key_id | [string](#string) |  |  |
| obfuscated_access_key_id | [string](#string) |  |  |
| secret_access_key | [string](#string) |  |  |
| obfuscated_secret_access_key | [string](#string) |  |  |
| session_token | [string](#string) |  |  |
| obfuscated_session_token | [string](#string) |  |  |
| role_arn | [string](#string) |  | ARN of IAM role to assume for cross-account access. See: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use.html |
| external_id | [string](#string) |  | Optional external ID for additional security when assuming role. See: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create_for-user_externalid.html |






<a name="bytebase-store-DataSource-Address"></a>

### DataSource.Address



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| host | [string](#string) |  |  |
| port | [string](#string) |  |  |






<a name="bytebase-store-DataSource-AzureCredential"></a>

### DataSource.AzureCredential



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tenant_id | [string](#string) |  |  |
| client_id | [string](#string) |  |  |
| client_secret | [string](#string) |  |  |
| obfuscated_client_secret | [string](#string) |  |  |






<a name="bytebase-store-DataSource-ExtraConnectionParametersEntry"></a>

### DataSource.ExtraConnectionParametersEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-store-DataSource-GCPCredential"></a>

### DataSource.GCPCredential



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| content | [string](#string) |  |  |
| obfuscated_content | [string](#string) |  |  |






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
| skip_vault_tls_verification | [bool](#bool) |  | TLS configuration for connecting to Vault server. These fields are separate from the database TLS configuration in DataSource. skip_vault_tls_verification disables TLS certificate verification for Vault connections. Default is false (verification enabled) for security. Only set to true for development or when certificates cannot be properly validated. |
| vault_ssl_ca | [string](#string) |  | CA certificate for Vault server verification. |
| obfuscated_vault_ssl_ca | [string](#string) |  |  |
| vault_ssl_cert | [string](#string) |  | Client certificate for mutual TLS authentication with Vault. |
| obfuscated_vault_ssl_cert | [string](#string) |  |  |
| vault_ssl_key | [string](#string) |  | Client private key for mutual TLS authentication with Vault. |
| obfuscated_vault_ssl_key | [string](#string) |  |  |






<a name="bytebase-store-DataSourceExternalSecret-AppRoleAuthOption"></a>

### DataSourceExternalSecret.AppRoleAuthOption



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role_id | [string](#string) |  |  |
| secret_id | [string](#string) |  | The secret ID for the role without TTL. |
| type | [DataSourceExternalSecret.AppRoleAuthOption.SecretType](#bytebase-store-DataSourceExternalSecret-AppRoleAuthOption-SecretType) |  |  |
| mount_path | [string](#string) |  | The path where the approle auth method is mounted. |






<a name="bytebase-store-Instance"></a>

### Instance
Instance is the proto for instances.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  |  |
| engine | [Engine](#bytebase-store-Engine) |  |  |
| activation | [bool](#bool) |  |  |
| version | [string](#string) |  |  |
| external_link | [string](#string) |  |  |
| data_sources | [DataSource](#bytebase-store-DataSource) | repeated |  |
| sync_interval | [google.protobuf.Duration](#google-protobuf-Duration) |  | The interval between automatic instance synchronizations. |
| sync_databases | [string](#string) | repeated | Enable sync for the following databases. Default empty, means sync all schemas &amp; databases. |
| mysql_lower_case_table_names | [int32](#int32) |  | The lower_case_table_names config for MySQL instances. It is used to determine whether the table names and database names are case sensitive. |
| last_sync_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| roles | [InstanceRole](#bytebase-store-InstanceRole) | repeated |  |
| labels | [Instance.LabelsEntry](#bytebase-store-Instance-LabelsEntry) | repeated | Labels are key-value pairs that can be attached to the instance. For example, { &#34;org_group&#34;: &#34;infrastructure&#34;, &#34;environment&#34;: &#34;production&#34; } |






<a name="bytebase-store-Instance-LabelsEntry"></a>

### Instance.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-store-InstanceRole"></a>

### InstanceRole
InstanceRole is the API message for instance role.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The role name. |
| connection_limit | [int32](#int32) | optional | The connection count limit for this role. |
| valid_until | [string](#string) | optional | The expiration for the role&#39;s password. |
| attribute | [string](#string) | optional | The role attribute. For PostgreSQL, it contains super_user, no_inherit, create_role, create_db, can_login, replication and bypass_rls. Docs: https://www.postgresql.org/docs/current/role-attributes.html For MySQL, it is the global privileges as GRANT statements, which means it only contains &#34;GRANT ... ON *.* TO ...&#34;. Docs: https://dev.mysql.com/doc/refman/8.0/en/grant.html |






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





 


<a name="bytebase-store-DataSource-AuthenticationType"></a>

### DataSource.AuthenticationType


| Name | Number | Description |
| ---- | ------ | ----------- |
| AUTHENTICATION_UNSPECIFIED | 0 |  |
| PASSWORD | 1 |  |
| GOOGLE_CLOUD_SQL_IAM | 2 |  |
| AWS_RDS_IAM | 3 |  |
| AZURE_IAM | 4 |  |



<a name="bytebase-store-DataSource-RedisType"></a>

### DataSource.RedisType


| Name | Number | Description |
| ---- | ------ | ----------- |
| REDIS_TYPE_UNSPECIFIED | 0 |  |
| STANDALONE | 1 |  |
| SENTINEL | 2 |  |
| CLUSTER | 3 |  |



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
| SECRET_TYPE_UNSPECIFIED | 0 |  |
| VAULT_KV_V2 | 1 | ref: https://developer.hashicorp.com/vault/api-docs/secret/kv/kv-v2 |
| AWS_SECRETS_MANAGER | 2 | ref: https://docs.aws.amazon.com/secretsmanager/latest/userguide/intro.html |
| GCP_SECRET_MANAGER | 3 | ref: https://cloud.google.com/secret-manager/docs |
| AZURE_KEY_VAULT | 4 | ref: https://learn.microsoft.com/en-us/azure/key-vault/secrets/about-secrets |



<a name="bytebase-store-DataSourceType"></a>

### DataSourceType


| Name | Number | Description |
| ---- | ------ | ----------- |
| DATA_SOURCE_UNSPECIFIED | 0 |  |
| ADMIN | 1 |  |
| READ_ONLY | 2 |  |


 

 

 



<a name="store_issue-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/issue.proto



<a name="bytebase-store-GrantRequest"></a>

### GrantRequest
GrantRequest contains details for requesting database access permissions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [string](#string) |  | The role being requested for the user. Format: roles/EXPORTER. |
| user | [string](#string) |  | The user who will receive the role. Format: users/{email}. |
| condition | [google.type.Expr](#google-type-Expr) |  | Optional conditional expression that limits when the grant applies. |
| expiration | [google.protobuf.Duration](#google-protobuf-Duration) |  | Duration after which the grant automatically expires. |






<a name="bytebase-store-Issue"></a>

### Issue
Issue is the metadata for issues that track database operations and access requests.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| approval | [IssuePayloadApproval](#bytebase-store-IssuePayloadApproval) |  | Approval information for the issue workflow. |
| grant_request | [GrantRequest](#bytebase-store-GrantRequest) |  | Access grant request details if this is a grant request issue. |
| labels | [string](#string) | repeated | Labels attached to categorize and filter the issue. |
| risk_level | [RiskLevel](#bytebase-store-RiskLevel) |  | Risk level for the issue, calculated from statement types. |





 


<a name="bytebase-store-Issue-Status"></a>

### Issue.Status
Status represents the current state of the issue.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ISSUE_STATUS_UNSPECIFIED | 0 |  |
| OPEN | 1 | Issue is open and pending action. |
| DONE | 2 | Issue has been completed successfully. |
| CANCELED | 3 | Issue was canceled and will not be completed. |



<a name="bytebase-store-Issue-Type"></a>

### Issue.Type
Type represents the category of issue.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ISSUE_TYPE_UNSPECIFIED | 0 |  |
| DATABASE_CHANGE | 1 | Issue for database schema or data changes. |
| GRANT_REQUEST | 2 | Issue requesting database access permissions. |
| DATABASE_EXPORT | 3 | Issue for exporting data from databases. |


 

 

 



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
| plan_spec_update | [IssueCommentPayload.PlanSpecUpdate](#bytebase-store-IssueCommentPayload-PlanSpecUpdate) |  |  |






<a name="bytebase-store-IssueCommentPayload-Approval"></a>

### IssueCommentPayload.Approval



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [IssuePayloadApproval.Approver.Status](#bytebase-store-IssuePayloadApproval-Approver-Status) |  |  |






<a name="bytebase-store-IssueCommentPayload-IssueUpdate"></a>

### IssueCommentPayload.IssueUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| from_title | [string](#string) | optional |  |
| to_title | [string](#string) | optional |  |
| from_description | [string](#string) | optional |  |
| to_description | [string](#string) | optional |  |
| from_status | [Issue.Status](#bytebase-store-Issue-Status) | optional |  |
| to_status | [Issue.Status](#bytebase-store-Issue-Status) | optional |  |
| from_labels | [string](#string) | repeated |  |
| to_labels | [string](#string) | repeated |  |






<a name="bytebase-store-IssueCommentPayload-PlanSpecUpdate"></a>

### IssueCommentPayload.PlanSpecUpdate
Plan spec update event (tracks sheet changes to plan specs)


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| spec | [string](#string) |  | The spec that was updated Format: projects/{project}/plans/{plan}/specs/{spec} |
| from_sheet_sha256 | [string](#string) | optional | The SHA256 hash of the previous sheet content (hex-encoded). |
| to_sheet_sha256 | [string](#string) | optional | The SHA256 hash of the new sheet content (hex-encoded). |





 

 

 

 



<a name="store_oauth2-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/oauth2.proto



<a name="bytebase-store-OAuth2AuthorizationCodeConfig"></a>

### OAuth2AuthorizationCodeConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| redirect_uri | [string](#string) |  |  |
| code_challenge | [string](#string) |  |  |
| code_challenge_method | [string](#string) |  |  |






<a name="bytebase-store-OAuth2ClientConfig"></a>

### OAuth2ClientConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| client_name | [string](#string) |  |  |
| redirect_uris | [string](#string) | repeated |  |
| grant_types | [string](#string) | repeated |  |
| token_endpoint_auth_method | [string](#string) |  |  |





 

 

 

 



<a name="store_plan-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/plan.proto



<a name="bytebase-store-PlanConfig"></a>

### PlanConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| specs | [PlanConfig.Spec](#bytebase-store-PlanConfig-Spec) | repeated |  |
| has_rollout | [bool](#bool) |  | Whether the plan has started the rollout. |






<a name="bytebase-store-PlanConfig-ChangeDatabaseConfig"></a>

### PlanConfig.ChangeDatabaseConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| targets | [string](#string) | repeated | The list of targets. Multi-database format: [instances/{instance-id}/databases/{database-name}]. Single database group format: [projects/{project}/databaseGroups/{databaseGroup}]. |
| sheet_sha256 | [string](#string) |  | The SHA256 hash of the sheet content (hex-encoded). |
| release | [string](#string) |  | The resource name of the release. Format: projects/{project}/releases/{release} |
| ghost_flags | [PlanConfig.ChangeDatabaseConfig.GhostFlagsEntry](#bytebase-store-PlanConfig-ChangeDatabaseConfig-GhostFlagsEntry) | repeated |  |
| enable_prior_backup | [bool](#bool) |  | If set, a backup of the modified data will be created automatically before any changes are applied. |
| enable_ghost | [bool](#bool) |  | Whether to use gh-ost for online schema migration. |






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
| environment | [string](#string) |  | The environment resource. Format: environments/prod where prod is the environment resource ID. |






<a name="bytebase-store-PlanConfig-ExportDataConfig"></a>

### PlanConfig.ExportDataConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| targets | [string](#string) | repeated | The list of targets. Multi-database format: [instances/{instance-id}/databases/{database-name}]. Single database group format: [projects/{project}/databaseGroups/{databaseGroup}]. |
| sheet_sha256 | [string](#string) |  | The SHA256 hash of the sheet content (hex-encoded). |
| format | [ExportFormat](#bytebase-store-ExportFormat) |  | The format of the exported file. |
| password | [string](#string) | optional | The zip password provided by users. Leave it empty if there is no need to encrypt the zip file. |






<a name="bytebase-store-PlanConfig-Spec"></a>

### PlanConfig.Spec



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | A UUID4 string that uniquely identifies the Spec. |
| create_database_config | [PlanConfig.CreateDatabaseConfig](#bytebase-store-PlanConfig-CreateDatabaseConfig) |  |  |
| change_database_config | [PlanConfig.ChangeDatabaseConfig](#bytebase-store-PlanConfig-ChangeDatabaseConfig) |  |  |
| export_data_config | [PlanConfig.ExportDataConfig](#bytebase-store-PlanConfig-ExportDataConfig) |  |  |





 

 

 

 



<a name="store_plan_check_run-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/plan_check_run.proto



<a name="bytebase-store-ChangedResourceDatabase"></a>

### ChangedResourceDatabase



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| schemas | [ChangedResourceSchema](#bytebase-store-ChangedResourceSchema) | repeated |  |






<a name="bytebase-store-ChangedResourceSchema"></a>

### ChangedResourceSchema



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| tables | [ChangedResourceTable](#bytebase-store-ChangedResourceTable) | repeated |  |






<a name="bytebase-store-ChangedResourceTable"></a>

### ChangedResourceTable



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| table_rows | [int64](#int64) |  | The estimated row count of the table. |






<a name="bytebase-store-ChangedResources"></a>

### ChangedResources



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [ChangedResourceDatabase](#bytebase-store-ChangedResourceDatabase) | repeated |  |






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
| status | [Advice.Status](#bytebase-store-Advice-Status) |  |  |
| title | [string](#string) |  |  |
| content | [string](#string) |  |  |
| code | [int32](#int32) |  |  |
| target | [string](#string) |  | Target identification for consolidated results Format: instances/{instance}/databases/{database} |
| type | [PlanCheckType](#bytebase-store-PlanCheckType) |  |  |
| sql_summary_report | [PlanCheckRunResult.Result.SqlSummaryReport](#bytebase-store-PlanCheckRunResult-Result-SqlSummaryReport) |  |  |
| sql_review_report | [PlanCheckRunResult.Result.SqlReviewReport](#bytebase-store-PlanCheckRunResult-Result-SqlReviewReport) |  |  |






<a name="bytebase-store-PlanCheckRunResult-Result-SqlReviewReport"></a>

### PlanCheckRunResult.Result.SqlReviewReport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start_position | [Position](#bytebase-store-Position) |  | Position of the SQL statement. |
| end_position | [Position](#bytebase-store-Position) |  |  |






<a name="bytebase-store-PlanCheckRunResult-Result-SqlSummaryReport"></a>

### PlanCheckRunResult.Result.SqlSummaryReport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| statement_types | [string](#string) | repeated | statement_types are the types of statements found in the SQL. |
| affected_rows | [int64](#int64) |  |  |
| changed_resources | [ChangedResources](#bytebase-store-ChangedResources) |  |  |





 


<a name="bytebase-store-PlanCheckType"></a>

### PlanCheckType


| Name | Number | Description |
| ---- | ------ | ----------- |
| PLAN_CHECK_TYPE_UNSPECIFIED | 0 |  |
| PLAN_CHECK_TYPE_STATEMENT_ADVISE | 1 |  |
| PLAN_CHECK_TYPE_STATEMENT_SUMMARY_REPORT | 2 |  |
| PLAN_CHECK_TYPE_GHOST_SYNC | 3 |  |


 

 

 



<a name="store_policy-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/policy.proto



<a name="bytebase-store-Binding"></a>

### Binding



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [string](#string) |  | The role that is assigned to the members. Format: roles/{role} |
| members | [string](#string) | repeated | Specifies the principals requesting access for a Bytebase resource. For users, the member should be: users/{email} For groups, the member should be: groups/{email} |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this binding. If the condition evaluates to true, then this binding applies to the current request. If the condition evaluates to false, then this binding does not apply to the current request. However, a different role binding might grant the same role to one or more of the principals in this binding. |






<a name="bytebase-store-DataSourceQueryPolicy"></a>

### DataSourceQueryPolicy
DataSourceQueryPolicy is the policy configuration for running statements in the SQL editor.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| admin_data_source_restriction | [DataSourceQueryPolicy.Restriction](#bytebase-store-DataSourceQueryPolicy-Restriction) |  |  |
| disallow_ddl | [bool](#bool) |  | Disallow running DDL statements in the SQL editor. |
| disallow_dml | [bool](#bool) |  | Disallow running DML statements in the SQL editor. |






<a name="bytebase-store-EnvironmentTierPolicy"></a>

### EnvironmentTierPolicy
EnvironmentTierPolicy is the tier of an environment.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| environment_tier | [EnvironmentTierPolicy.EnvironmentTier](#bytebase-store-EnvironmentTierPolicy-EnvironmentTier) |  |  |
| color | [string](#string) |  |  |






<a name="bytebase-store-IamPolicy"></a>

### IamPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bindings | [Binding](#bytebase-store-Binding) | repeated | Collection of binding. A binding binds one or more members or groups to a single role. |






<a name="bytebase-store-MaskingExemptionPolicy"></a>

### MaskingExemptionPolicy
MaskingExemptionPolicy is the allowlist of users who can access sensitive data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| exemptions | [MaskingExemptionPolicy.Exemption](#bytebase-store-MaskingExemptionPolicy-Exemption) | repeated |  |






<a name="bytebase-store-MaskingExemptionPolicy-Exemption"></a>

### MaskingExemptionPolicy.Exemption



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| members | [string](#string) | repeated | Members who bind to this exemption.

Format: users/{email} or groups/{group email} |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this exception policy instance. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec If the condition is empty, means the user can access all databases without expiration.

Support variables: resource.instance_id: the instance resource id. Only support &#34;==&#34; operation. resource.database_name: the database name. Only support &#34;==&#34; operation. resource.schema_name: the schema name. Only support &#34;==&#34; operation. resource.table_name: the table name. Only support &#34;==&#34; operation. resource.column_name: the column name. Only support &#34;==&#34; operation. request.time: the expiration. Only support &#34;&lt;&#34; operation in `request.time &lt; timestamp(&#34;{ISO datetime string format}&#34;)` All variables should join with &#34;&amp;&amp;&#34; condition.

For example: resource.instance_id == &#34;local&#34; &amp;&amp; resource.database_name == &#34;employee&#34; &amp;&amp; request.time &lt; timestamp(&#34;2025-04-30T11:10:39.000Z&#34;) resource.instance_id == &#34;local&#34; &amp;&amp; resource.database_name == &#34;employee&#34; |






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
| semantic_type | [string](#string) |  |  |






<a name="bytebase-store-Policy"></a>

### Policy







<a name="bytebase-store-QueryDataPolicy"></a>

### QueryDataPolicy
QueryDataPolicy is the policy configuration for querying data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timeout | [google.protobuf.Duration](#google-protobuf-Duration) |  | The query timeout duration. |
| disable_export | [bool](#bool) |  | Disable exporting data in the SQL editor. |
| maximum_result_size | [int64](#int64) |  | The size limit in bytes. The default value is 100MB, we will use the default value if the setting not exists, or the limit &lt;= 0. |
| maximum_result_rows | [int32](#int32) |  | The return rows limit. The default value is -1, means no limit. |
| disable_copy_data | [bool](#bool) |  | Disable copying data. |






<a name="bytebase-store-RolloutPolicy"></a>

### RolloutPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| automatic | [bool](#bool) |  |  |
| roles | [string](#string) | repeated |  |






<a name="bytebase-store-TagPolicy"></a>

### TagPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tags | [TagPolicy.TagsEntry](#bytebase-store-TagPolicy-TagsEntry) | repeated | tags is the key-value map for resources. For example, the environment resource can have the SQL review config tag, such as &#34;bb.tag.review_config&#34;: &#34;reviewConfigs/{review config resource id}&#34;. |






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



<a name="bytebase-store-Policy-Resource"></a>

### Policy.Resource


| Name | Number | Description |
| ---- | ------ | ----------- |
| RESOURCE_UNSPECIFIED | 0 |  |
| WORKSPACE | 1 |  |
| ENVIRONMENT | 2 |  |
| PROJECT | 3 |  |



<a name="bytebase-store-Policy-Type"></a>

### Policy.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| ROLLOUT | 1 |  |
| MASKING_EXEMPTION | 2 |  |
| QUERY_DATA | 3 |  |
| MASKING_RULE | 4 |  |
| IAM | 5 |  |
| TAG | 6 |  |
| DATA_SOURCE_QUERY | 7 |  |


 

 

 



<a name="store_project-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/project.proto



<a name="bytebase-store-Label"></a>

### Label
Label represents a categorization tag that can be applied to issues.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [string](#string) |  | The display text of the label. |
| color | [string](#string) |  | The color for the label in hex format (e.g., &#34;#FF0000&#34;). |
| group | [string](#string) |  | Optional group name for organizing related labels. |






<a name="bytebase-store-Project"></a>

### Project
Project contains settings and configuration for a Bytebase project.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issue_labels | [Label](#bytebase-store-Label) | repeated | Available labels that can be applied to issues in this project. |
| force_issue_labels | [bool](#bool) |  | Force issue labels to be used when creating an issue. |
| enforce_issue_title | [bool](#bool) |  | Enforce issue title created by user instead of generated by Bytebase. |
| auto_enable_backup | [bool](#bool) |  | Whether to automatically enable backup. |
| skip_backup_errors | [bool](#bool) |  | Whether to skip backup errors and continue the data migration. |
| postgres_database_tenant_mode | [bool](#bool) |  | Whether to enable the database tenant mode for PostgreSQL. If enabled, the issue will be created with the prepend &#34;set role &lt;db_owner&gt;&#34; statement. |
| allow_self_approval | [bool](#bool) |  | Whether to allow the issue creator to self-approve the issue. |
| execution_retry_policy | [Project.ExecutionRetryPolicy](#bytebase-store-Project-ExecutionRetryPolicy) |  | Configuration for automatic retry on task execution failures. |
| ci_sampling_size | [int32](#int32) |  | The maximum number of databases to sample during CI data validation. If not specified, sampling is disabled, resulting in a full validation. |
| parallel_tasks_per_rollout | [int32](#int32) |  | The maximum number of parallel tasks to run during the rollout. |
| labels | [Project.LabelsEntry](#bytebase-store-Project-LabelsEntry) | repeated | Labels are key-value pairs that can be attached to the project. For example, { &#34;environment&#34;: &#34;production&#34;, &#34;team&#34;: &#34;backend&#34; } |
| enforce_sql_review | [bool](#bool) |  | Whether to enforce SQL review checks to pass before issue creation. If enabled, issues cannot be created when SQL review finds errors. |
| require_issue_approval | [bool](#bool) |  | Whether issue approval is required before proceeding with rollout. |
| require_plan_check_no_error | [bool](#bool) |  | Whether to block rollout when plan check finds errors. |






<a name="bytebase-store-Project-ExecutionRetryPolicy"></a>

### Project.ExecutionRetryPolicy
ExecutionRetryPolicy defines retry behavior for failed task executions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| maximum_retries | [int32](#int32) |  | The maximum number of retry attempts for lock timeout errors. |






<a name="bytebase-store-Project-LabelsEntry"></a>

### Project.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |





 

 

 

 



<a name="store_project_webhook-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/project_webhook.proto



<a name="bytebase-store-Activity"></a>

### Activity
Activity types for webhook notifications.






<a name="bytebase-store-ProjectWebhook"></a>

### ProjectWebhook



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [WebhookType](#bytebase-store-WebhookType) |  | Webhook type. |
| title | [string](#string) |  | Webhook title. |
| url | [string](#string) |  | Webhook URL. |
| activities | [Activity.Type](#bytebase-store-Activity-Type) | repeated | List of activities that trigger this webhook. |
| direct_message | [bool](#bool) |  | If direct_message is set, the notification is sent directly to the persons and url will be ignored. IM integration setting should be set for this function to work. |





 


<a name="bytebase-store-Activity-Type"></a>

### Activity.Type
Activity type enumeration.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 | Unspecified type. |
| ISSUE_CREATED | 10 | ISSUE_CREATED represents a new issue creation event. |
| ISSUE_APPROVAL_REQUESTED | 11 | ISSUE_APPROVAL_REQUESTED represents an approval request event. |
| ISSUE_SENT_BACK | 12 | ISSUE_SENT_BACK represents an issue being sent back by an approver. |
| PIPELINE_FAILED | 13 | PIPELINE_FAILED represents a pipeline failure event. |
| PIPELINE_COMPLETED | 14 | PIPELINE_COMPLETED represents a pipeline completion event. |


 

 

 



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
| type | [SchemaChangeType](#bytebase-store-SchemaChangeType) |  |  |






<a name="bytebase-store-ReleasePayload-File"></a>

### ReleasePayload.File



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) |  | The path of the file, e.g., `2.2/V0001_create_table.sql`. |
| sheet_sha256 | [string](#string) |  | The SHA256 hash of the sheet content (hex-encoded). |
| version | [string](#string) |  |  |
| enable_ghost | [bool](#bool) |  | Whether to use gh-ost for online schema migration. |






<a name="bytebase-store-ReleasePayload-VCSSource"></a>

### ReleasePayload.VCSSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vcs_type | [VCSType](#bytebase-store-VCSType) |  |  |
| url | [string](#string) |  |  |





 

 

 

 



<a name="store_review_config-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/review_config.proto



<a name="bytebase-store-ReviewConfigPayload"></a>

### ReviewConfigPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sql_review_rules | [SQLReviewRule](#bytebase-store-SQLReviewRule) | repeated |  |






<a name="bytebase-store-SQLReviewRule"></a>

### SQLReviewRule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [SQLReviewRule.Type](#bytebase-store-SQLReviewRule-Type) |  |  |
| level | [SQLReviewRule.Level](#bytebase-store-SQLReviewRule-Level) |  |  |
| naming_payload | [SQLReviewRule.NamingRulePayload](#bytebase-store-SQLReviewRule-NamingRulePayload) |  |  |
| number_payload | [SQLReviewRule.NumberRulePayload](#bytebase-store-SQLReviewRule-NumberRulePayload) |  |  |
| string_array_payload | [SQLReviewRule.StringArrayRulePayload](#bytebase-store-SQLReviewRule-StringArrayRulePayload) |  |  |
| comment_convention_payload | [SQLReviewRule.CommentConventionRulePayload](#bytebase-store-SQLReviewRule-CommentConventionRulePayload) |  |  |
| string_payload | [SQLReviewRule.StringRulePayload](#bytebase-store-SQLReviewRule-StringRulePayload) |  |  |
| naming_case_payload | [SQLReviewRule.NamingCaseRulePayload](#bytebase-store-SQLReviewRule-NamingCaseRulePayload) |  |  |
| engine | [Engine](#bytebase-store-Engine) |  |  |






<a name="bytebase-store-SQLReviewRule-CommentConventionRulePayload"></a>

### SQLReviewRule.CommentConventionRulePayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| required | [bool](#bool) |  |  |
| max_length | [int32](#int32) |  |  |






<a name="bytebase-store-SQLReviewRule-NamingCaseRulePayload"></a>

### SQLReviewRule.NamingCaseRulePayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| upper | [bool](#bool) |  |  |






<a name="bytebase-store-SQLReviewRule-NamingRulePayload"></a>

### SQLReviewRule.NamingRulePayload
Payload message types for SQL review rules


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| max_length | [int32](#int32) |  |  |
| format | [string](#string) |  |  |






<a name="bytebase-store-SQLReviewRule-NumberRulePayload"></a>

### SQLReviewRule.NumberRulePayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| number | [int32](#int32) |  |  |






<a name="bytebase-store-SQLReviewRule-StringArrayRulePayload"></a>

### SQLReviewRule.StringArrayRulePayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| list | [string](#string) | repeated |  |






<a name="bytebase-store-SQLReviewRule-StringRulePayload"></a>

### SQLReviewRule.StringRulePayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [string](#string) |  |  |





 


<a name="bytebase-store-SQLReviewRule-Level"></a>

### SQLReviewRule.Level
The severity level for SQL review rules.

| Name | Number | Description |
| ---- | ------ | ----------- |
| LEVEL_UNSPECIFIED | 0 | Unspecified level. |
| ERROR | 1 | Rule violation is an error. |
| WARNING | 2 | Rule violation is a warning. |



<a name="bytebase-store-SQLReviewRule-Type"></a>

### SQLReviewRule.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| ENGINE_MYSQL_USE_INNODB | 1 |  |
| NAMING_FULLY_QUALIFIED | 2 |  |
| NAMING_TABLE | 3 |  |
| NAMING_COLUMN | 4 |  |
| NAMING_INDEX_PK | 5 |  |
| NAMING_INDEX_UK | 6 |  |
| NAMING_INDEX_FK | 7 |  |
| NAMING_INDEX_IDX | 8 |  |
| NAMING_COLUMN_AUTO_INCREMENT | 9 |  |
| NAMING_TABLE_NO_KEYWORD | 10 |  |
| NAMING_IDENTIFIER_NO_KEYWORD | 11 |  |
| NAMING_IDENTIFIER_CASE | 12 |  |
| STATEMENT_SELECT_NO_SELECT_ALL | 13 |  |
| STATEMENT_WHERE_REQUIRE_SELECT | 14 |  |
| STATEMENT_WHERE_REQUIRE_UPDATE_DELETE | 15 |  |
| STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE | 16 |  |
| STATEMENT_DISALLOW_ON_DEL_CASCADE | 17 |  |
| STATEMENT_DISALLOW_RM_TBL_CASCADE | 18 |  |
| STATEMENT_DISALLOW_COMMIT | 19 |  |
| STATEMENT_DISALLOW_LIMIT | 20 |  |
| STATEMENT_DISALLOW_ORDER_BY | 21 |  |
| STATEMENT_MERGE_ALTER_TABLE | 22 |  |
| STATEMENT_INSERT_ROW_LIMIT | 23 |  |
| STATEMENT_INSERT_MUST_SPECIFY_COLUMN | 24 |  |
| STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND | 25 |  |
| STATEMENT_AFFECTED_ROW_LIMIT | 26 |  |
| STATEMENT_DML_DRY_RUN | 27 |  |
| STATEMENT_DISALLOW_ADD_COLUMN_WITH_DEFAULT | 28 |  |
| STATEMENT_ADD_CHECK_NOT_VALID | 29 |  |
| STATEMENT_ADD_FOREIGN_KEY_NOT_VALID | 30 |  |
| STATEMENT_DISALLOW_ADD_NOT_NULL | 31 |  |
| STATEMENT_SELECT_FULL_TABLE_SCAN | 32 |  |
| STATEMENT_CREATE_SPECIFY_SCHEMA | 33 |  |
| STATEMENT_CHECK_SET_ROLE_VARIABLE | 34 |  |
| STATEMENT_DISALLOW_USING_FILESORT | 35 |  |
| STATEMENT_DISALLOW_USING_TEMPORARY | 36 |  |
| STATEMENT_WHERE_NO_EQUAL_NULL | 37 |  |
| STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS | 38 |  |
| STATEMENT_QUERY_MINIMUM_PLAN_LEVEL | 39 |  |
| STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT | 40 |  |
| STATEMENT_MAXIMUM_LIMIT_VALUE | 41 |  |
| STATEMENT_MAXIMUM_JOIN_TABLE_COUNT | 42 |  |
| STATEMENT_MAXIMUM_STATEMENTS_IN_TRANSACTION | 43 |  |
| STATEMENT_JOIN_STRICT_COLUMN_ATTRS | 44 |  |
| STATEMENT_NON_TRANSACTIONAL | 45 |  |
| STATEMENT_ADD_COLUMN_WITHOUT_POSITION | 46 |  |
| STATEMENT_DISALLOW_OFFLINE_DDL | 47 |  |
| STATEMENT_DISALLOW_CROSS_DB_QUERIES | 48 |  |
| STATEMENT_MAX_EXECUTION_TIME | 49 |  |
| STATEMENT_REQUIRE_ALGORITHM_OPTION | 50 |  |
| STATEMENT_REQUIRE_LOCK_OPTION | 51 |  |
| STATEMENT_OBJECT_OWNER_CHECK | 52 |  |
| TABLE_REQUIRE_PK | 53 |  |
| TABLE_NO_FOREIGN_KEY | 54 |  |
| TABLE_DROP_NAMING_CONVENTION | 55 |  |
| TABLE_COMMENT | 56 |  |
| TABLE_DISALLOW_PARTITION | 57 |  |
| TABLE_DISALLOW_TRIGGER | 58 |  |
| TABLE_NO_DUPLICATE_INDEX | 59 |  |
| TABLE_TEXT_FIELDS_TOTAL_LENGTH | 60 |  |
| TABLE_DISALLOW_SET_CHARSET | 61 |  |
| TABLE_DISALLOW_DDL | 62 |  |
| TABLE_DISALLOW_DML | 63 |  |
| TABLE_LIMIT_SIZE | 64 |  |
| TABLE_REQUIRE_CHARSET | 65 |  |
| TABLE_REQUIRE_COLLATION | 66 |  |
| COLUMN_REQUIRED | 67 |  |
| COLUMN_NO_NULL | 68 |  |
| COLUMN_DISALLOW_CHANGE_TYPE | 69 |  |
| COLUMN_SET_DEFAULT_FOR_NOT_NULL | 70 |  |
| COLUMN_DISALLOW_CHANGE | 71 |  |
| COLUMN_DISALLOW_CHANGING_ORDER | 72 |  |
| COLUMN_DISALLOW_DROP | 73 |  |
| COLUMN_DISALLOW_DROP_IN_INDEX | 74 |  |
| COLUMN_COMMENT | 75 |  |
| COLUMN_AUTO_INCREMENT_MUST_INTEGER | 76 |  |
| COLUMN_TYPE_DISALLOW_LIST | 77 |  |
| COLUMN_DISALLOW_SET_CHARSET | 78 |  |
| COLUMN_MAXIMUM_CHARACTER_LENGTH | 79 |  |
| COLUMN_MAXIMUM_VARCHAR_LENGTH | 80 |  |
| COLUMN_AUTO_INCREMENT_INITIAL_VALUE | 81 |  |
| COLUMN_AUTO_INCREMENT_MUST_UNSIGNED | 82 |  |
| COLUMN_CURRENT_TIME_COUNT_LIMIT | 83 |  |
| COLUMN_REQUIRE_DEFAULT | 84 |  |
| COLUMN_DEFAULT_DISALLOW_VOLATILE | 85 |  |
| COLUMN_ADD_NOT_NULL_REQUIRE_DEFAULT | 86 |  |
| COLUMN_REQUIRE_CHARSET | 87 |  |
| COLUMN_REQUIRE_COLLATION | 88 |  |
| SCHEMA_BACKWARD_COMPATIBILITY | 89 |  |
| DATABASE_DROP_EMPTY_DATABASE | 90 |  |
| INDEX_NO_DUPLICATE_COLUMN | 91 |  |
| INDEX_KEY_NUMBER_LIMIT | 92 |  |
| INDEX_PK_TYPE_LIMIT | 93 |  |
| INDEX_TYPE_NO_BLOB | 94 |  |
| INDEX_TOTAL_NUMBER_LIMIT | 95 |  |
| INDEX_PRIMARY_KEY_TYPE_ALLOWLIST | 96 |  |
| INDEX_CREATE_CONCURRENTLY | 97 |  |
| INDEX_TYPE_ALLOW_LIST | 98 |  |
| INDEX_NOT_REDUNDANT | 99 |  |
| SYSTEM_CHARSET_ALLOWLIST | 100 |  |
| SYSTEM_COLLATION_ALLOWLIST | 101 |  |
| SYSTEM_COMMENT_LENGTH | 102 |  |
| SYSTEM_PROCEDURE_DISALLOW_CREATE | 103 |  |
| SYSTEM_EVENT_DISALLOW_CREATE | 104 |  |
| SYSTEM_VIEW_DISALLOW_CREATE | 105 |  |
| SYSTEM_FUNCTION_DISALLOW_CREATE | 106 |  |
| SYSTEM_FUNCTION_DISALLOWED_LIST | 107 |  |
| ADVICE_ONLINE_MIGRATION | 108 |  |
| BUILTIN_PRIOR_BACKUP_CHECK | 109 |  |


 

 

 



<a name="store_revision-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/revision.proto



<a name="bytebase-store-RevisionPayload"></a>

### RevisionPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| release | [string](#string) |  | Format: projects/{project}/releases/{release} Can be empty. |
| file | [string](#string) |  | Format: projects/{project}/releases/{release}/files/{id} Can be empty. |
| sheet_sha256 | [string](#string) |  | The SHA256 hash of the sheet content (hex-encoded). |
| task_run | [string](#string) |  | The task run associated with the revision. Can be empty. Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |
| type | [SchemaChangeType](#bytebase-store-SchemaChangeType) |  | The type of the revision. |





 

 

 

 



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



<a name="bytebase-store-AISetting"></a>

### AISetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| provider | [AISetting.Provider](#bytebase-store-AISetting-Provider) |  |  |
| endpoint | [string](#string) |  |  |
| api_key | [string](#string) |  |  |
| model | [string](#string) |  |  |
| version | [string](#string) |  |  |






<a name="bytebase-store-Algorithm"></a>

### Algorithm



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| full_mask | [Algorithm.FullMask](#bytebase-store-Algorithm-FullMask) |  |  |
| range_mask | [Algorithm.RangeMask](#bytebase-store-Algorithm-RangeMask) |  |  |
| md5_mask | [Algorithm.MD5Mask](#bytebase-store-Algorithm-MD5Mask) |  |  |
| inner_outer_mask | [Algorithm.InnerOuterMask](#bytebase-store-Algorithm-InnerOuterMask) |  |  |






<a name="bytebase-store-Algorithm-FullMask"></a>

### Algorithm.FullMask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| substitution | [string](#string) |  | substitution is the string used to replace the original value, the max length of the string is 16 bytes. |






<a name="bytebase-store-Algorithm-InnerOuterMask"></a>

### Algorithm.InnerOuterMask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| prefix_len | [int32](#int32) |  |  |
| suffix_len | [int32](#int32) |  |  |
| type | [Algorithm.InnerOuterMask.MaskType](#bytebase-store-Algorithm-InnerOuterMask-MaskType) |  |  |
| substitution | [string](#string) |  |  |






<a name="bytebase-store-Algorithm-MD5Mask"></a>

### Algorithm.MD5Mask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| salt | [string](#string) |  | salt is the salt value to generate a different hash that with the word alone. |






<a name="bytebase-store-Algorithm-RangeMask"></a>

### Algorithm.RangeMask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| slices | [Algorithm.RangeMask.Slice](#bytebase-store-Algorithm-RangeMask-Slice) | repeated | We store it as a repeated field to face the fact that the original value may have multiple parts should be masked. But frontend can be started with a single rule easily. |






<a name="bytebase-store-Algorithm-RangeMask-Slice"></a>

### Algorithm.RangeMask.Slice



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start | [int32](#int32) |  | start is the start character index (0-based) of the original value, should be less than end. Uses character indices (not byte offsets) for display-oriented masking. Example: For &#34;你好world&#34;, character index 2 refers to &#39;w&#39; (the 3rd character). |
| end | [int32](#int32) |  | end is the end character index (exclusive) of the original value. Uses character indices (not byte offsets) for display-oriented masking. |
| substitution | [string](#string) |  | OriginalValue[start:end) would be replaced with substitution. |






<a name="bytebase-store-AppIMSetting"></a>

### AppIMSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| settings | [AppIMSetting.IMSetting](#bytebase-store-AppIMSetting-IMSetting) | repeated |  |






<a name="bytebase-store-AppIMSetting-DingTalk"></a>

### AppIMSetting.DingTalk



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| client_id | [string](#string) |  |  |
| client_secret | [string](#string) |  |  |
| robot_code | [string](#string) |  |  |






<a name="bytebase-store-AppIMSetting-Feishu"></a>

### AppIMSetting.Feishu



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_id | [string](#string) |  |  |
| app_secret | [string](#string) |  |  |






<a name="bytebase-store-AppIMSetting-IMSetting"></a>

### AppIMSetting.IMSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [WebhookType](#bytebase-store-WebhookType) |  |  |
| slack | [AppIMSetting.Slack](#bytebase-store-AppIMSetting-Slack) |  |  |
| feishu | [AppIMSetting.Feishu](#bytebase-store-AppIMSetting-Feishu) |  |  |
| wecom | [AppIMSetting.Wecom](#bytebase-store-AppIMSetting-Wecom) |  |  |
| lark | [AppIMSetting.Lark](#bytebase-store-AppIMSetting-Lark) |  |  |
| dingtalk | [AppIMSetting.DingTalk](#bytebase-store-AppIMSetting-DingTalk) |  |  |
| teams | [AppIMSetting.Teams](#bytebase-store-AppIMSetting-Teams) |  |  |






<a name="bytebase-store-AppIMSetting-Lark"></a>

### AppIMSetting.Lark



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_id | [string](#string) |  |  |
| app_secret | [string](#string) |  |  |






<a name="bytebase-store-AppIMSetting-Slack"></a>

### AppIMSetting.Slack



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  |  |






<a name="bytebase-store-AppIMSetting-Teams"></a>

### AppIMSetting.Teams



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tenant_id | [string](#string) |  | Azure AD tenant ID (Directory ID). |
| client_id | [string](#string) |  | Azure AD application (client) ID. |
| client_secret | [string](#string) |  | Azure AD client secret. |






<a name="bytebase-store-AppIMSetting-Wecom"></a>

### AppIMSetting.Wecom



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
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






<a name="bytebase-store-EnvironmentSetting"></a>

### EnvironmentSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| environments | [EnvironmentSetting.Environment](#bytebase-store-EnvironmentSetting-Environment) | repeated |  |






<a name="bytebase-store-EnvironmentSetting-Environment"></a>

### EnvironmentSetting.Environment



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| id | [string](#string) |  | The resource id of the environment. This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |
| title | [string](#string) |  | The display name of the environment. |
| tags | [EnvironmentSetting.Environment.TagsEntry](#bytebase-store-EnvironmentSetting-Environment-TagsEntry) | repeated |  |
| color | [string](#string) |  |  |






<a name="bytebase-store-EnvironmentSetting-Environment-TagsEntry"></a>

### EnvironmentSetting.Environment.TagsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






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
| algorithm | [Algorithm](#bytebase-store-Algorithm) |  |  |
| icon | [string](#string) |  | icon is the icon for semantic type, it can be emoji or base64 encoded image. |






<a name="bytebase-store-SystemSetting"></a>

### SystemSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| auth_secret | [string](#string) |  | Authentication secret for token signing (32-character random string). |
| workspace_id | [string](#string) |  | Unique workspace identifier (UUID). |
| license | [string](#string) |  | Enterprise license JWT token. |






<a name="bytebase-store-WorkspaceApprovalSetting"></a>

### WorkspaceApprovalSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rules | [WorkspaceApprovalSetting.Rule](#bytebase-store-WorkspaceApprovalSetting-Rule) | repeated |  |






<a name="bytebase-store-WorkspaceApprovalSetting-Rule"></a>

### WorkspaceApprovalSetting.Rule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| template | [ApprovalTemplate](#bytebase-store-ApprovalTemplate) |  |  |
| condition | [google.type.Expr](#google-type-Expr) |  |  |
| source | [WorkspaceApprovalSetting.Rule.Source](#bytebase-store-WorkspaceApprovalSetting-Rule-Source) |  |  |






<a name="bytebase-store-WorkspaceProfileSetting"></a>

### WorkspaceProfileSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| external_url | [string](#string) |  | The external URL is used for sso authentication callback. |
| disallow_signup | [bool](#bool) |  | Disallow self-service signup, users can only be invited by the owner. |
| require_2fa | [bool](#bool) |  | Require 2FA for all users. |
| refresh_token_duration | [google.protobuf.Duration](#google-protobuf-Duration) |  | The duration for refresh token. Default is 7 days. |
| access_token_duration | [google.protobuf.Duration](#google-protobuf-Duration) |  | The duration for access token. Default is 1 hour. |
| announcement | [WorkspaceProfileSetting.Announcement](#bytebase-store-WorkspaceProfileSetting-Announcement) |  | The setting of custom announcement |
| maximum_role_expiration | [google.protobuf.Duration](#google-protobuf-Duration) |  | The max duration for role expired. |
| domains | [string](#string) | repeated | The workspace domain, e.g., bytebase.com. |
| enforce_identity_domain | [bool](#bool) |  | Only user and group from the domains can be created and login. |
| database_change_mode | [WorkspaceProfileSetting.DatabaseChangeMode](#bytebase-store-WorkspaceProfileSetting-DatabaseChangeMode) |  | The workspace database change mode. |
| disallow_password_signin | [bool](#bool) |  | Whether to disallow password signin. (Except workspace admins) |
| enable_metric_collection | [bool](#bool) |  | Whether to enable metric collection for the workspace. |
| inactive_session_timeout | [google.protobuf.Duration](#google-protobuf-Duration) |  | The session expiration time if not activity detected for the user. Value &lt;= 0 means no limit. |
| enable_audit_log_stdout | [bool](#bool) |  | Whether to enable audit logging to stdout in structured JSON format. Requires TEAM or ENTERPRISE license. |
| watermark | [bool](#bool) |  | Whether to display watermark on pages. Requires ENTERPRISE license. |
| directory_sync_token | [string](#string) |  | The token for directory sync authentication. |
| branding_logo | [string](#string) |  | The branding logo as a data URI (e.g. data:image/png;base64,...). |
| password_restriction | [WorkspaceProfileSetting.PasswordRestriction](#bytebase-store-WorkspaceProfileSetting-PasswordRestriction) |  | Password restriction settings. |






<a name="bytebase-store-WorkspaceProfileSetting-Announcement"></a>

### WorkspaceProfileSetting.Announcement



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| level | [WorkspaceProfileSetting.Announcement.AlertLevel](#bytebase-store-WorkspaceProfileSetting-Announcement-AlertLevel) |  | The alert level of the announcement. |
| text | [string](#string) |  | The text of the announcement. |
| link | [string](#string) |  | The optional link, user can follow the link to check extra details |






<a name="bytebase-store-WorkspaceProfileSetting-PasswordRestriction"></a>

### WorkspaceProfileSetting.PasswordRestriction



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| min_length | [int32](#int32) |  | min_length is the minimum length for password, should be no less than 8. |
| require_number | [bool](#bool) |  | require_number requires the password must contain at least one number. |
| require_letter | [bool](#bool) |  | require_letter requires the password must contain at least one letter, regardless of upper case or lower case |
| require_uppercase_letter | [bool](#bool) |  | require_uppercase_letter requires the password must contain at least one upper case letter. |
| require_special_character | [bool](#bool) |  | require_special_character requires the password must contain at least one special character. |
| require_reset_password_for_first_login | [bool](#bool) |  | require_reset_password_for_first_login requires users to reset their password after the 1st login. |
| password_rotation | [google.protobuf.Duration](#google-protobuf-Duration) |  | password_rotation requires users to reset their password after the duration. |





 


<a name="bytebase-store-AISetting-Provider"></a>

### AISetting.Provider


| Name | Number | Description |
| ---- | ------ | ----------- |
| PROVIDER_UNSPECIFIED | 0 |  |
| OPEN_AI | 1 |  |
| CLAUDE | 2 |  |
| GEMINI | 3 |  |
| AZURE_OPENAI | 4 |  |



<a name="bytebase-store-Algorithm-InnerOuterMask-MaskType"></a>

### Algorithm.InnerOuterMask.MaskType


| Name | Number | Description |
| ---- | ------ | ----------- |
| MASK_TYPE_UNSPECIFIED | 0 |  |
| INNER | 1 |  |
| OUTER | 2 |  |



<a name="bytebase-store-SettingName"></a>

### SettingName


| Name | Number | Description |
| ---- | ------ | ----------- |
| SETTING_NAME_UNSPECIFIED | 0 |  |
| SYSTEM | 1 |  |
| WORKSPACE_PROFILE | 2 |  |
| WORKSPACE_APPROVAL | 3 |  |
| APP_IM | 4 |  |
| AI | 5 |  |
| DATA_CLASSIFICATION | 6 |  |
| SEMANTIC_TYPES | 7 |  |
| ENVIRONMENT | 8 |  |



<a name="bytebase-store-WorkspaceApprovalSetting-Rule-Source"></a>

### WorkspaceApprovalSetting.Rule.Source


| Name | Number | Description |
| ---- | ------ | ----------- |
| SOURCE_UNSPECIFIED | 0 |  |
| CHANGE_DATABASE | 1 |  |
| CREATE_DATABASE | 2 |  |
| EXPORT_DATA | 3 |  |
| REQUEST_ROLE | 4 |  |



<a name="bytebase-store-WorkspaceProfileSetting-Announcement-AlertLevel"></a>

### WorkspaceProfileSetting.Announcement.AlertLevel
We support three levels of AlertLevel: INFO, WARNING, and ERROR.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ALERT_LEVEL_UNSPECIFIED | 0 |  |
| INFO | 1 |  |
| WARNING | 2 |  |
| CRITICAL | 3 |  |



<a name="bytebase-store-WorkspaceProfileSetting-DatabaseChangeMode"></a>

### WorkspaceProfileSetting.DatabaseChangeMode


| Name | Number | Description |
| ---- | ------ | ----------- |
| DATABASE_CHANGE_MODE_UNSPECIFIED | 0 |  |
| PIPELINE | 1 | A more advanced database change process, including custom approval workflows and other advanced features. Default to this mode. |
| EDITOR | 2 | A simple database change process in SQL editor. Users can execute SQL directly. |


 

 

 



<a name="store_signal-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/signal.proto



<a name="bytebase-store-Signal"></a>

### Signal
Signal represents a notification payload sent via PostgreSQL NOTIFY for HA coordination.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [Signal.Type](#bytebase-store-Signal-Type) |  |  |
| uid | [int32](#int32) |  |  |





 


<a name="bytebase-store-Signal-Type"></a>

### Signal.Type
Type represents the type of signal.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| CANCEL_PLAN_CHECK_RUN | 1 |  |
| CANCEL_TASK_RUN | 2 |  |


 

 

 



<a name="store_task-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/task.proto



<a name="bytebase-store-Task"></a>

### Task
Task is the metadata for database operation tasks.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| skipped | [bool](#bool) |  | Whether the task was skipped during execution. |
| skipped_reason | [string](#string) |  | Reason why the task was skipped. |
| spec_id | [string](#string) |  | UUID that identifies the spec this task implements. |
| sheet_sha256 | [string](#string) |  | The SHA256 hash of a single sheet content (hex-encoded). Used for non-release tasks. |
| release | [string](#string) |  | The release resource name: projects/{project}/releases/{release}. Used for GitOps release-based tasks that execute multiple files. |
| enable_prior_backup | [bool](#bool) |  | Whether to create an automatic backup before applying changes. |
| flags | [Task.FlagsEntry](#bytebase-store-Task-FlagsEntry) | repeated | Configuration flags for gh-ost migration tool. |
| enable_ghost | [bool](#bool) |  | Whether to use gh-ost for online schema migration. |






<a name="bytebase-store-Task-FlagsEntry"></a>

### Task.FlagsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |





 


<a name="bytebase-store-Task-Type"></a>

### Task.Type
Type represents the type of database operation to perform.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TASK_TYPE_UNSPECIFIED | 0 |  |
| DATABASE_CREATE | 1 | Create a new database. |
| DATABASE_MIGRATE | 2 | Apply schema/data migrations to an existing database. Execution strategy is determined by release type (VERSIONED/DECLARATIVE) or sheet content for non-release tasks. |
| DATABASE_EXPORT | 3 | Export data from a database. |


 

 

 



<a name="store_task_run-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/task_run.proto



<a name="bytebase-store-SchedulerInfo"></a>

### SchedulerInfo
SchedulerInfo contains information about task scheduling and execution delays.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| report_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Timestamp when the scheduler reported this information. |
| waiting_cause | [SchedulerInfo.WaitingCause](#bytebase-store-SchedulerInfo-WaitingCause) |  | Reason why the task run is currently waiting. |






<a name="bytebase-store-SchedulerInfo-WaitingCause"></a>

### SchedulerInfo.WaitingCause
WaitingCause indicates why a task run is waiting to execute.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parallel_tasks_limit | [bool](#bool) |  | Task is waiting due to parallel execution limit. |






<a name="bytebase-store-TaskRun"></a>

### TaskRun
TaskRun represents an execution attempt of a task.






<a name="bytebase-store-TaskRunResult"></a>

### TaskRunResult
TaskRunResult contains the outcome and metadata from a task run execution.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| detail | [string](#string) |  | Error message for failed task runs. Empty for successful or canceled runs. |
| export_archive_uid | [int32](#int32) |  | UID of the export archive generated for export tasks. |
| has_prior_backup | [bool](#bool) |  | Indicates whether a prior backup was created for this task run. When true, the task run can be rolled back using the backup tables. Backup details are available in the task run logs (PRIOR_BACKUP log entries). |





 


<a name="bytebase-store-TaskRun-Status"></a>

### TaskRun.Status
Status represents the current execution state of a task run.

| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 | Task run is queued and waiting to execute. |
| RUNNING | 2 | Task run is currently executing. |
| DONE | 3 | Task run completed successfully. |
| FAILED | 4 | Task run encountered an error and failed. |
| CANCELED | 5 | Task run was canceled by user or system. |
| NOT_STARTED | 6 | Task run has not started yet. |
| SKIPPED | 7 | Task run was skipped and will not execute. |
| AVAILABLE | 8 | Task run is ready for immediate execution. |


 

 

 



<a name="store_task_run_log-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/task_run_log.proto



<a name="bytebase-store-PriorBackupDetail"></a>

### PriorBackupDetail
PriorBackupDetail contains information about automatic backups created before migration.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| items | [PriorBackupDetail.Item](#bytebase-store-PriorBackupDetail-Item) | repeated | List of backup operations performed. |






<a name="bytebase-store-PriorBackupDetail-Item"></a>

### PriorBackupDetail.Item
Item represents a single backup operation for a table.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| source_table | [PriorBackupDetail.Item.Table](#bytebase-store-PriorBackupDetail-Item-Table) |  | The original table that was backed up. |
| target_table | [PriorBackupDetail.Item.Table](#bytebase-store-PriorBackupDetail-Item-Table) |  | The backup table where data was copied. |
| start_position | [Position](#bytebase-store-Position) |  | Starting position in SQL for this backup operation. |
| end_position | [Position](#bytebase-store-Position) |  | Ending position in SQL for this backup operation. |






<a name="bytebase-store-PriorBackupDetail-Item-Table"></a>

### PriorBackupDetail.Item.Table
Table identifies a database table.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [string](#string) |  | The database containing the table. Format: instances/{instance}/databases/{database} |
| schema | [string](#string) |  | Schema name (for databases that support schemas). |
| table | [string](#string) |  | Table name. |






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
| transaction_control | [TaskRunLog.TransactionControl](#bytebase-store-TaskRunLog-TransactionControl) |  |  |
| prior_backup_start | [TaskRunLog.PriorBackupStart](#bytebase-store-TaskRunLog-PriorBackupStart) |  |  |
| prior_backup_end | [TaskRunLog.PriorBackupEnd](#bytebase-store-TaskRunLog-PriorBackupEnd) |  |  |
| retry_info | [TaskRunLog.RetryInfo](#bytebase-store-TaskRunLog-RetryInfo) |  |  |
| compute_diff_start | [TaskRunLog.ComputeDiffStart](#bytebase-store-TaskRunLog-ComputeDiffStart) |  |  |
| compute_diff_end | [TaskRunLog.ComputeDiffEnd](#bytebase-store-TaskRunLog-ComputeDiffEnd) |  |  |
| release_file_execute | [TaskRunLog.ReleaseFileExecute](#bytebase-store-TaskRunLog-ReleaseFileExecute) |  |  |






<a name="bytebase-store-TaskRunLog-CommandExecute"></a>

### TaskRunLog.CommandExecute



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| range | [Range](#bytebase-store-Range) |  | The byte offset range of the executed command in the sheet. Uses byte offsets (not character indices) for efficient slicing of sheet content bytes. Example: For &#34;SELECT 你好;&#34; in a UTF-8 sheet, range [0, 13) represents all 13 bytes. |
| statement | [string](#string) |  | The statement to be executed. |






<a name="bytebase-store-TaskRunLog-CommandResponse"></a>

### TaskRunLog.CommandResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| error | [string](#string) |  |  |
| affected_rows | [int64](#int64) |  |  |
| all_affected_rows | [int64](#int64) | repeated | `all_affected_rows` is the affected rows of each command. `all_affected_rows` may be unavailable if the database driver doesn&#39;t support it. Caller should fallback to `affected_rows` in that case. |






<a name="bytebase-store-TaskRunLog-ComputeDiffEnd"></a>

### TaskRunLog.ComputeDiffEnd



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| error | [string](#string) |  |  |






<a name="bytebase-store-TaskRunLog-ComputeDiffStart"></a>

### TaskRunLog.ComputeDiffStart







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







<a name="bytebase-store-TaskRunLog-ReleaseFileExecute"></a>

### TaskRunLog.ReleaseFileExecute



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [string](#string) |  | The version of the file being executed (e.g., &#34;0001&#34;). |
| file_path | [string](#string) |  | The file path within the release (e.g., &#34;2.2/V0001_create_table.sql&#34;). |






<a name="bytebase-store-TaskRunLog-RetryInfo"></a>

### TaskRunLog.RetryInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| error | [string](#string) |  |  |
| retry_count | [int32](#int32) |  |  |
| maximum_retries | [int32](#int32) |  |  |






<a name="bytebase-store-TaskRunLog-SchemaDumpEnd"></a>

### TaskRunLog.SchemaDumpEnd



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| error | [string](#string) |  |  |






<a name="bytebase-store-TaskRunLog-SchemaDumpStart"></a>

### TaskRunLog.SchemaDumpStart







<a name="bytebase-store-TaskRunLog-TransactionControl"></a>

### TaskRunLog.TransactionControl



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [TaskRunLog.TransactionControl.Type](#bytebase-store-TaskRunLog-TransactionControl-Type) |  |  |
| error | [string](#string) |  |  |





 


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
| TRANSACTION_CONTROL | 8 |  |
| PRIOR_BACKUP_START | 9 |  |
| PRIOR_BACKUP_END | 10 |  |
| RETRY_INFO | 11 |  |
| COMPUTE_DIFF_START | 12 |  |
| COMPUTE_DIFF_END | 13 |  |
| RELEASE_FILE_EXECUTE | 14 |  |


 

 

 



<a name="store_user-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/user.proto



<a name="bytebase-store-MFAConfig"></a>

### MFAConfig
MFAConfig is the MFA configuration for a user.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| otp_secret | [string](#string) |  | The otp_secret is the secret key used to validate the OTP code. |
| temp_otp_secret | [string](#string) |  | The temp_otp_secret is the temporary secret key used to validate the OTP code and will replace the otp_secret in two-phase commits. |
| recovery_codes | [string](#string) | repeated | The recovery_codes are the codes that can be used to recover the account. |
| temp_recovery_codes | [string](#string) | repeated | The temp_recovery_codes are the temporary codes that will replace the recovery_codes in two-phase commits. |
| temp_otp_secret_created_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The temp_otp_secret_created_time is the timestamp when temp_otp_secret was created. Used to enforce expiration. |






<a name="bytebase-store-UserProfile"></a>

### UserProfile



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| last_login_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| last_change_password_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| source | [string](#string) |  | The source indicates where the user comes from. For now we support Entra ID SCIM sync, so the source could be Entra ID. |
| workload_identity_config | [WorkloadIdentityConfig](#bytebase-store-WorkloadIdentityConfig) |  | Workload identity configuration (only for WORKLOAD_IDENTITY type) |






<a name="bytebase-store-WorkloadIdentityConfig"></a>

### WorkloadIdentityConfig
WorkloadIdentityConfig stores OIDC configuration for workload identity.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider_type | [WorkloadIdentityConfig.ProviderType](#bytebase-store-WorkloadIdentityConfig-ProviderType) |  | Provider type (currently only GITHUB is supported) |
| issuer_url | [string](#string) |  | OIDC issuer URL |
| allowed_audiences | [string](#string) | repeated | Allowed audiences for token validation |
| subject_pattern | [string](#string) |  | Subject pattern to match against token subject claim |





 


<a name="bytebase-store-PrincipalType"></a>

### PrincipalType
PrincipalType is the type of a principal.

| Name | Number | Description |
| ---- | ------ | ----------- |
| PRINCIPAL_TYPE_UNSPECIFIED | 0 |  |
| END_USER | 1 | END_USER represents the human being using Bytebase. |
| SERVICE_ACCOUNT | 2 | SERVICE_ACCOUNT represents the external service calling Bytebase OpenAPI. |
| SYSTEM_BOT | 3 | SYSTEM_BOT represents the internal system bot performing operations. |
| WORKLOAD_IDENTITY | 4 | WORKLOAD_IDENTITY represents external CI/CD workload identity. |



<a name="bytebase-store-WorkloadIdentityConfig-ProviderType"></a>

### WorkloadIdentityConfig.ProviderType
ProviderType identifies the CI/CD platform.

| Name | Number | Description |
| ---- | ------ | ----------- |
| PROVIDER_TYPE_UNSPECIFIED | 0 |  |
| GITHUB | 1 |  |
| GITLAB | 2 |  |


 

 

 



<a name="store_worksheet-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/worksheet.proto



<a name="bytebase-store-WorkSheetOrganizerPayload"></a>

### WorkSheetOrganizerPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| starred | [bool](#bool) |  |  |
| folders | [string](#string) | repeated | The folder path for a worksheet. For example, if the folders is [A, B, C], means the worksheet is in the A/B/C subfolder. |





 

 

 

 



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

