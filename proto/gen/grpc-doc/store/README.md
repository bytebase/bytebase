# Protocol Documentation
<a name="top"></a>

## Table of Contents

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
  
- [store/activity.proto](#store_activity-proto)
    - [ActivityIssueApprovalNotifyPayload](#bytebase-store-ActivityIssueApprovalNotifyPayload)
    - [ActivityIssueCommentCreatePayload](#bytebase-store-ActivityIssueCommentCreatePayload)
    - [ActivityIssueCommentCreatePayload.ApprovalEvent](#bytebase-store-ActivityIssueCommentCreatePayload-ApprovalEvent)
    - [ActivityIssueCommentCreatePayload.ExternalApprovalEvent](#bytebase-store-ActivityIssueCommentCreatePayload-ExternalApprovalEvent)
    - [ActivityIssueCommentCreatePayload.TaskRollbackBy](#bytebase-store-ActivityIssueCommentCreatePayload-TaskRollbackBy)
    - [ActivityIssueCreatePayload](#bytebase-store-ActivityIssueCreatePayload)
  
    - [ActivityIssueCommentCreatePayload.ApprovalEvent.Status](#bytebase-store-ActivityIssueCommentCreatePayload-ApprovalEvent-Status)
    - [ActivityIssueCommentCreatePayload.ExternalApprovalEvent.Action](#bytebase-store-ActivityIssueCommentCreatePayload-ExternalApprovalEvent-Action)
    - [ActivityIssueCommentCreatePayload.ExternalApprovalEvent.Type](#bytebase-store-ActivityIssueCommentCreatePayload-ExternalApprovalEvent-Type)
  
- [store/changelist.proto](#store_changelist-proto)
    - [Changelist](#bytebase-store-Changelist)
    - [Changelist.Change](#bytebase-store-Changelist-Change)
  
- [store/common.proto](#store_common-proto)
    - [PageToken](#bytebase-store-PageToken)
  
    - [Engine](#bytebase-store-Engine)
    - [MaskingLevel](#bytebase-store-MaskingLevel)
    - [VcsType](#bytebase-store-VcsType)
  
- [store/data_source.proto](#store_data_source-proto)
    - [DataSourceOptions](#bytebase-store-DataSourceOptions)
  
- [store/database.proto](#store_database-proto)
    - [ColumnConfig](#bytebase-store-ColumnConfig)
    - [ColumnConfig.LabelsEntry](#bytebase-store-ColumnConfig-LabelsEntry)
    - [ColumnMetadata](#bytebase-store-ColumnMetadata)
    - [DatabaseConfig](#bytebase-store-DatabaseConfig)
    - [DatabaseMetadata](#bytebase-store-DatabaseMetadata)
    - [DatabaseMetadata.LabelsEntry](#bytebase-store-DatabaseMetadata-LabelsEntry)
    - [DatabaseSchemaMetadata](#bytebase-store-DatabaseSchemaMetadata)
    - [DependentColumn](#bytebase-store-DependentColumn)
    - [ExtensionMetadata](#bytebase-store-ExtensionMetadata)
    - [ForeignKeyMetadata](#bytebase-store-ForeignKeyMetadata)
    - [FunctionMetadata](#bytebase-store-FunctionMetadata)
    - [IndexMetadata](#bytebase-store-IndexMetadata)
    - [InstanceRoleMetadata](#bytebase-store-InstanceRoleMetadata)
    - [SchemaConfig](#bytebase-store-SchemaConfig)
    - [SchemaMetadata](#bytebase-store-SchemaMetadata)
    - [SecretItem](#bytebase-store-SecretItem)
    - [Secrets](#bytebase-store-Secrets)
    - [StreamMetadata](#bytebase-store-StreamMetadata)
    - [TableConfig](#bytebase-store-TableConfig)
    - [TableMetadata](#bytebase-store-TableMetadata)
    - [TaskMetadata](#bytebase-store-TaskMetadata)
    - [ViewMetadata](#bytebase-store-ViewMetadata)
  
    - [StreamMetadata.Mode](#bytebase-store-StreamMetadata-Mode)
    - [StreamMetadata.Type](#bytebase-store-StreamMetadata-Type)
    - [TaskMetadata.State](#bytebase-store-TaskMetadata-State)
  
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
  
- [store/vcs.proto](#store_vcs-proto)
    - [Commit](#bytebase-store-Commit)
    - [FileCommit](#bytebase-store-FileCommit)
    - [PushEvent](#bytebase-store-PushEvent)
  
- [store/instance_change_history.proto](#store_instance_change_history-proto)
    - [ChangedResourceDatabase](#bytebase-store-ChangedResourceDatabase)
    - [ChangedResourceSchema](#bytebase-store-ChangedResourceSchema)
    - [ChangedResourceTable](#bytebase-store-ChangedResourceTable)
    - [ChangedResources](#bytebase-store-ChangedResources)
    - [InstanceChangeHistoryPayload](#bytebase-store-InstanceChangeHistoryPayload)
  
- [store/issue.proto](#store_issue-proto)
    - [GrantRequest](#bytebase-store-GrantRequest)
    - [IssuePayload](#bytebase-store-IssuePayload)
  
- [store/plan.proto](#store_plan-proto)
    - [PlanConfig](#bytebase-store-PlanConfig)
    - [PlanConfig.ChangeDatabaseConfig](#bytebase-store-PlanConfig-ChangeDatabaseConfig)
    - [PlanConfig.ChangeDatabaseConfig.RollbackDetail](#bytebase-store-PlanConfig-ChangeDatabaseConfig-RollbackDetail)
    - [PlanConfig.CreateDatabaseConfig](#bytebase-store-PlanConfig-CreateDatabaseConfig)
    - [PlanConfig.CreateDatabaseConfig.LabelsEntry](#bytebase-store-PlanConfig-CreateDatabaseConfig-LabelsEntry)
    - [PlanConfig.RestoreDatabaseConfig](#bytebase-store-PlanConfig-RestoreDatabaseConfig)
    - [PlanConfig.Spec](#bytebase-store-PlanConfig-Spec)
    - [PlanConfig.Step](#bytebase-store-PlanConfig-Step)
  
    - [PlanConfig.ChangeDatabaseConfig.Type](#bytebase-store-PlanConfig-ChangeDatabaseConfig-Type)
  
- [store/plan_check_run.proto](#store_plan_check_run-proto)
    - [PlanCheckRunConfig](#bytebase-store-PlanCheckRunConfig)
    - [PlanCheckRunResult](#bytebase-store-PlanCheckRunResult)
    - [PlanCheckRunResult.Result](#bytebase-store-PlanCheckRunResult-Result)
    - [PlanCheckRunResult.Result.SqlReviewReport](#bytebase-store-PlanCheckRunResult-Result-SqlReviewReport)
    - [PlanCheckRunResult.Result.SqlSummaryReport](#bytebase-store-PlanCheckRunResult-Result-SqlSummaryReport)
  
    - [PlanCheckRunConfig.ChangeDatabaseType](#bytebase-store-PlanCheckRunConfig-ChangeDatabaseType)
    - [PlanCheckRunResult.Result.Status](#bytebase-store-PlanCheckRunResult-Result-Status)
  
- [store/policy.proto](#store_policy-proto)
    - [Binding](#bytebase-store-Binding)
    - [IamPolicy](#bytebase-store-IamPolicy)
    - [MaskData](#bytebase-store-MaskData)
    - [MaskingExceptionPolicy](#bytebase-store-MaskingExceptionPolicy)
    - [MaskingExceptionPolicy.MaskingException](#bytebase-store-MaskingExceptionPolicy-MaskingException)
    - [MaskingPolicy](#bytebase-store-MaskingPolicy)
    - [MaskingRulePolicy](#bytebase-store-MaskingRulePolicy)
    - [MaskingRulePolicy.MaskingRule](#bytebase-store-MaskingRulePolicy-MaskingRule)
    - [RolloutPolicy](#bytebase-store-RolloutPolicy)
    - [SQLReviewPolicy](#bytebase-store-SQLReviewPolicy)
    - [SQLReviewRule](#bytebase-store-SQLReviewRule)
  
    - [MaskingExceptionPolicy.MaskingException.Action](#bytebase-store-MaskingExceptionPolicy-MaskingException-Action)
    - [SQLReviewRuleLevel](#bytebase-store-SQLReviewRuleLevel)
  
- [store/project.proto](#store_project-proto)
    - [Project](#bytebase-store-Project)
    - [ProtectionRule](#bytebase-store-ProtectionRule)
  
    - [ProtectionRule.Target](#bytebase-store-ProtectionRule-Target)
  
- [store/setting.proto](#store_setting-proto)
    - [AgentPluginSetting](#bytebase-store-AgentPluginSetting)
    - [Announcement](#bytebase-store-Announcement)
    - [DataClassificationSetting](#bytebase-store-DataClassificationSetting)
    - [DataClassificationSetting.DataClassificationConfig](#bytebase-store-DataClassificationSetting-DataClassificationConfig)
    - [DataClassificationSetting.DataClassificationConfig.ClassificationEntry](#bytebase-store-DataClassificationSetting-DataClassificationConfig-ClassificationEntry)
    - [DataClassificationSetting.DataClassificationConfig.DataClassification](#bytebase-store-DataClassificationSetting-DataClassificationConfig-DataClassification)
    - [DataClassificationSetting.DataClassificationConfig.Level](#bytebase-store-DataClassificationSetting-DataClassificationConfig-Level)
    - [ExternalApprovalSetting](#bytebase-store-ExternalApprovalSetting)
    - [ExternalApprovalSetting.Node](#bytebase-store-ExternalApprovalSetting-Node)
    - [MaskingAlgorithmSetting](#bytebase-store-MaskingAlgorithmSetting)
    - [MaskingAlgorithmSetting.Algorithm](#bytebase-store-MaskingAlgorithmSetting-Algorithm)
    - [MaskingAlgorithmSetting.Algorithm.FullMask](#bytebase-store-MaskingAlgorithmSetting-Algorithm-FullMask)
    - [MaskingAlgorithmSetting.Algorithm.MD5Mask](#bytebase-store-MaskingAlgorithmSetting-Algorithm-MD5Mask)
    - [MaskingAlgorithmSetting.Algorithm.RangeMask](#bytebase-store-MaskingAlgorithmSetting-Algorithm-RangeMask)
    - [MaskingAlgorithmSetting.Algorithm.RangeMask.Slice](#bytebase-store-MaskingAlgorithmSetting-Algorithm-RangeMask-Slice)
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
    - [SMTPMailDeliverySetting.Authentication](#bytebase-store-SMTPMailDeliverySetting-Authentication)
    - [SMTPMailDeliverySetting.Encryption](#bytebase-store-SMTPMailDeliverySetting-Encryption)
  
- [store/sheet.proto](#store_sheet-proto)
    - [SheetPayload](#bytebase-store-SheetPayload)
    - [SheetPayload.SchemaDesign](#bytebase-store-SheetPayload-SchemaDesign)
    - [SheetPayload.SchemaDesign.Protection](#bytebase-store-SheetPayload-SchemaDesign-Protection)
    - [SheetPayload.VCSPayload](#bytebase-store-SheetPayload-VCSPayload)
  
    - [SheetPayload.SchemaDesign.Type](#bytebase-store-SheetPayload-SchemaDesign-Type)
    - [SheetPayload.Type](#bytebase-store-SheetPayload-Type)
  
- [store/slow_query.proto](#store_slow_query-proto)
    - [SlowQueryDetails](#bytebase-store-SlowQueryDetails)
    - [SlowQueryStatistics](#bytebase-store-SlowQueryStatistics)
    - [SlowQueryStatisticsItem](#bytebase-store-SlowQueryStatisticsItem)
  
- [store/task_run.proto](#store_task_run-proto)
    - [TaskRunResult](#bytebase-store-TaskRunResult)
  
- [store/user.proto](#store_user-proto)
    - [MFAConfig](#bytebase-store-MFAConfig)
  
- [Scalar Value Types](#scalar-value-types)



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


 

 

 



<a name="store_activity-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/activity.proto



<a name="bytebase-store-ActivityIssueApprovalNotifyPayload"></a>

### ActivityIssueApprovalNotifyPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| approval_step | [ApprovalStep](#bytebase-store-ApprovalStep) |  |  |






<a name="bytebase-store-ActivityIssueCommentCreatePayload"></a>

### ActivityIssueCommentCreatePayload
ActivityIssueCommentCreatePayload is the payloads for creating issue comments.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| external_approval_event | [ActivityIssueCommentCreatePayload.ExternalApprovalEvent](#bytebase-store-ActivityIssueCommentCreatePayload-ExternalApprovalEvent) |  |  |
| task_rollback_by | [ActivityIssueCommentCreatePayload.TaskRollbackBy](#bytebase-store-ActivityIssueCommentCreatePayload-TaskRollbackBy) |  |  |
| approval_event | [ActivityIssueCommentCreatePayload.ApprovalEvent](#bytebase-store-ActivityIssueCommentCreatePayload-ApprovalEvent) |  |  |
| issue_name | [string](#string) |  | Used by inbox to display info without paying the join cost |






<a name="bytebase-store-ActivityIssueCommentCreatePayload-ApprovalEvent"></a>

### ActivityIssueCommentCreatePayload.ApprovalEvent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [ActivityIssueCommentCreatePayload.ApprovalEvent.Status](#bytebase-store-ActivityIssueCommentCreatePayload-ApprovalEvent-Status) |  | The new status. |






<a name="bytebase-store-ActivityIssueCommentCreatePayload-ExternalApprovalEvent"></a>

### ActivityIssueCommentCreatePayload.ExternalApprovalEvent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [ActivityIssueCommentCreatePayload.ExternalApprovalEvent.Type](#bytebase-store-ActivityIssueCommentCreatePayload-ExternalApprovalEvent-Type) |  |  |
| action | [ActivityIssueCommentCreatePayload.ExternalApprovalEvent.Action](#bytebase-store-ActivityIssueCommentCreatePayload-ExternalApprovalEvent-Action) |  |  |
| stage_name | [string](#string) |  |  |






<a name="bytebase-store-ActivityIssueCommentCreatePayload-TaskRollbackBy"></a>

### ActivityIssueCommentCreatePayload.TaskRollbackBy
TaskRollbackBy records an issue rollback activity.
The task with taskID in IssueID is rollbacked by the task with RollbackByTaskID in RollbackByIssueID.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issue_id | [int64](#int64) |  |  |
| task_id | [int64](#int64) |  |  |
| rollback_by_issue_id | [int64](#int64) |  |  |
| rollback_by_task_id | [int64](#int64) |  |  |






<a name="bytebase-store-ActivityIssueCreatePayload"></a>

### ActivityIssueCreatePayload
ActivityIssueCreatePayload is the payloads for creating issues.
These payload types are only used when marshalling to the json format for saving into the database.
So we annotate with json tag using camelCase naming which is consistent with normal
json naming convention. More importantly, frontend code can simply use JSON.parse to
convert to the expected struct there.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issue_name | [string](#string) |  | Used by inbox to display info without paying the join cost |





 


<a name="bytebase-store-ActivityIssueCommentCreatePayload-ApprovalEvent-Status"></a>

### ActivityIssueCommentCreatePayload.ApprovalEvent.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| APPROVED | 2 |  |
| REJECTED | 3 |  |



<a name="bytebase-store-ActivityIssueCommentCreatePayload-ExternalApprovalEvent-Action"></a>

### ActivityIssueCommentCreatePayload.ExternalApprovalEvent.Action


| Name | Number | Description |
| ---- | ------ | ----------- |
| ACTION_UNSPECIFIED | 0 |  |
| ACTION_APPROVE | 1 |  |
| ACTION_REJECT | 2 |  |



<a name="bytebase-store-ActivityIssueCommentCreatePayload-ExternalApprovalEvent-Type"></a>

### ActivityIssueCommentCreatePayload.ExternalApprovalEvent.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| TYPE_FEISHU | 1 |  |


 

 

 



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
| source | [string](#string) |  | The source of origin. 1) change history: instances/{instance}/databases/{database}/changeHistories/{changeHistory}. 2) branch: projects/{project}/schemaDesigns/{schemaDesign}. 3) raw SQL if empty. |





 

 

 

 



<a name="store_common-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/common.proto



<a name="bytebase-store-PageToken"></a>

### PageToken
Used internally for obfuscating the page token.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| limit | [int32](#int32) |  |  |
| offset | [int32](#int32) |  |  |





 


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



<a name="bytebase-store-MaskingLevel"></a>

### MaskingLevel


| Name | Number | Description |
| ---- | ------ | ----------- |
| MASKING_LEVEL_UNSPECIFIED | 0 |  |
| NONE | 1 |  |
| PARTIAL | 2 |  |
| FULL | 3 |  |



<a name="bytebase-store-VcsType"></a>

### VcsType


| Name | Number | Description |
| ---- | ------ | ----------- |
| VCS_TYPE_UNSPECIFIED | 0 |  |
| GITLAB | 1 |  |
| GITHUB | 2 |  |
| BITBUCKET | 3 |  |


 

 

 



<a name="store_data_source-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/data_source.proto



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





 

 

 

 



<a name="store_database-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/database.proto



<a name="bytebase-store-ColumnConfig"></a>

### ColumnConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a column. |
| semantic_type_id | [string](#string) |  |  |
| labels | [ColumnConfig.LabelsEntry](#bytebase-store-ColumnConfig-LabelsEntry) | repeated | The user labels for a column. |






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
| nullable | [bool](#bool) |  | The nullable is the nullable of a column. |
| type | [string](#string) |  | The type is the type of a column. |
| character_set | [string](#string) |  | The character_set is the character_set of a column. |
| collation | [string](#string) |  | The collation is the collation of a column. |
| comment | [string](#string) |  | The comment is the comment of a column. classification and user_comment is parsed from the comment. |
| classification | [string](#string) |  | The classification is the classification of a table parsed from the comment. |
| user_comment | [string](#string) |  | The user_comment is the user comment of a table parsed from the comment. |






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






<a name="bytebase-store-DependentColumn"></a>

### DependentColumn
DependentColumn is the metadata for dependent columns.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  | The schema is the schema of a reference column. |
| table | [string](#string) |  | The table is the table of a reference column. |
| column | [string](#string) |  | The column is the name of a reference column. |






<a name="bytebase-store-ExtensionMetadata"></a>

### ExtensionMetadata
ExtensionMetadata is the metadata for extensions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of an extension. |
| schema | [string](#string) |  | The schema is the extension that is installed to. But the extension usage is not limited to the schema. |
| version | [string](#string) |  | The version is the version of an extension. |
| description | [string](#string) |  | The description is the description of an extension. |






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






<a name="bytebase-store-FunctionMetadata"></a>

### FunctionMetadata
FunctionMetadata is the metadata for functions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a view. |
| definition | [string](#string) |  | The definition is the definition of a view. |






<a name="bytebase-store-IndexMetadata"></a>

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






<a name="bytebase-store-InstanceRoleMetadata"></a>

### InstanceRoleMetadata
InstanceRoleMetadata is the message for instance role.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The role name. It&#39;s unique within the instance. |
| grant | [string](#string) |  | The grant display string on the instance. It&#39;s generated by database engine. |






<a name="bytebase-store-SchemaConfig"></a>

### SchemaConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the schema name. It is an empty string for databases without such concept such as MySQL. |
| table_configs | [TableConfig](#bytebase-store-TableConfig) | repeated | The table_configs is the list of configs for tables in a schema. |






<a name="bytebase-store-SchemaMetadata"></a>

### SchemaMetadata
SchemaMetadata is the metadata for schemas.
This is the concept of schema in Postgres, but it&#39;s a no-op for MySQL.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the schema name. It is an empty string for databases without such concept such as MySQL. |
| tables | [TableMetadata](#bytebase-store-TableMetadata) | repeated | The tables is the list of tables in a schema. |
| views | [ViewMetadata](#bytebase-store-ViewMetadata) | repeated | The views is the list of views in a schema. |
| functions | [FunctionMetadata](#bytebase-store-FunctionMetadata) | repeated | The functions is the list of functions in a schema. |
| streams | [StreamMetadata](#bytebase-store-StreamMetadata) | repeated | The streams is the list of streams in a schema, currently, only used for Snowflake. |
| tasks | [TaskMetadata](#bytebase-store-TaskMetadata) | repeated | The routines is the list of routines in a schema, currently, only used for Snowflake. |






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
| row_count | [int64](#int64) |  | The row_count is the estimated number of rows of a table. |
| data_size | [int64](#int64) |  | The data_size is the estimated data size of a table. |
| index_size | [int64](#int64) |  | The index_size is the estimated index size of a table. |
| data_free | [int64](#int64) |  | The data_free is the estimated free data size of a table. |
| create_options | [string](#string) |  | The create_options is the create option of a table. |
| comment | [string](#string) |  | The comment is the comment of a table. classification and user_comment is parsed from the comment. |
| classification | [string](#string) |  | The classification is the classification of a table parsed from the comment. |
| user_comment | [string](#string) |  | The user_comment is the user comment of a table parsed from the comment. |
| foreign_keys | [ForeignKeyMetadata](#bytebase-store-ForeignKeyMetadata) | repeated | The foreign_keys is the list of foreign keys in a table. |






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






<a name="bytebase-store-ViewMetadata"></a>

### ViewMetadata
ViewMetadata is the metadata for views.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a view. |
| definition | [string](#string) |  | The definition is the definition of a view. |
| comment | [string](#string) |  | The comment is the comment of a view. |
| dependent_columns | [DependentColumn](#bytebase-store-DependentColumn) | repeated | The dependent_columns is the list of dependent columns of a view. |





 


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



<a name="bytebase-store-TaskMetadata-State"></a>

### TaskMetadata.State


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATE_UNSPECIFIED | 0 |  |
| STATE_STARTED | 1 |  |
| STATE_SUSPENDED | 2 |  |


 

 

 



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






<a name="bytebase-store-InstanceOptions"></a>

### InstanceOptions
InstanceOptions is the option for instances.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema_tenant_mode | [bool](#bool) |  | The schema tenant mode is used to determine whether the instance is in schema tenant mode. For Oracle schema tenant mode, the instance a Oracle database and the database is the Oracle schema. |
| sync_interval | [google.protobuf.Duration](#google-protobuf-Duration) |  | How often the instance is synced. |





 

 

 

 



<a name="store_vcs-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/vcs.proto



<a name="bytebase-store-Commit"></a>

### Commit



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| title | [string](#string) |  |  |
| message | [string](#string) |  |  |
| created_ts | [int64](#int64) |  |  |
| url | [string](#string) |  |  |
| author_name | [string](#string) |  |  |
| author_email | [string](#string) |  |  |
| added_list | [string](#string) | repeated |  |
| modified_list | [string](#string) | repeated |  |






<a name="bytebase-store-FileCommit"></a>

### FileCommit



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| title | [string](#string) |  |  |
| message | [string](#string) |  |  |
| created_ts | [int64](#int64) |  |  |
| url | [string](#string) |  |  |
| author_name | [string](#string) |  |  |
| author_email | [string](#string) |  |  |
| added | [string](#string) |  |  |






<a name="bytebase-store-PushEvent"></a>

### PushEvent



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vcs_type | [VcsType](#bytebase-store-VcsType) |  |  |
| base_dir | [string](#string) |  |  |
| ref | [string](#string) |  |  |
| before | [string](#string) |  |  |
| after | [string](#string) |  |  |
| repository_id | [string](#string) |  |  |
| repository_url | [string](#string) |  |  |
| repository_full_path | [string](#string) |  |  |
| author_name | [string](#string) |  |  |
| commits | [Commit](#bytebase-store-Commit) | repeated |  |
| file_commit | [FileCommit](#bytebase-store-FileCommit) |  |  |





 

 

 

 



<a name="store_instance_change_history-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/instance_change_history.proto



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






<a name="bytebase-store-ChangedResources"></a>

### ChangedResources



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [ChangedResourceDatabase](#bytebase-store-ChangedResourceDatabase) | repeated |  |






<a name="bytebase-store-InstanceChangeHistoryPayload"></a>

### InstanceChangeHistoryPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| push_event | [PushEvent](#bytebase-store-PushEvent) |  |  |
| changed_resources | [ChangedResources](#bytebase-store-ChangedResources) |  |  |





 

 

 

 



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





 

 

 

 



<a name="store_plan-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/plan.proto



<a name="bytebase-store-PlanConfig"></a>

### PlanConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| steps | [PlanConfig.Step](#bytebase-store-PlanConfig-Step) | repeated |  |






<a name="bytebase-store-PlanConfig-ChangeDatabaseConfig"></a>

### PlanConfig.ChangeDatabaseConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| target | [string](#string) |  | The resource name of the target. Format: instances/{instance-id}/databases/{database-name}. Format: projects/{project}/databaseGroups/{databaseGroup}. Format: projects/{project}/deploymentConfigs/default. The plan should have a single step and single spec for the deployment configuration type. |
| sheet | [string](#string) |  | The resource name of the sheet. Format: projects/{project}/sheets/{sheet} |
| type | [PlanConfig.ChangeDatabaseConfig.Type](#bytebase-store-PlanConfig-ChangeDatabaseConfig-Type) |  |  |
| schema_version | [string](#string) |  | schema_version is parsed from VCS file name. It is automatically generated in the UI workflow. |
| rollback_enabled | [bool](#bool) |  | If RollbackEnabled, build the RollbackSheetID of the task. |
| rollback_detail | [PlanConfig.ChangeDatabaseConfig.RollbackDetail](#bytebase-store-PlanConfig-ChangeDatabaseConfig-RollbackDetail) | optional |  |






<a name="bytebase-store-PlanConfig-ChangeDatabaseConfig-RollbackDetail"></a>

### PlanConfig.ChangeDatabaseConfig.RollbackDetail



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rollback_from_task | [string](#string) |  | rollback_from_task is the task from which the rollback SQL statement is generated for this task. Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} |
| rollback_from_issue | [string](#string) |  | rollback_from_issue is the issue containing the original task from which the rollback SQL statement is generated for this task. Format: projects/{project}/issues/{issue} |






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






<a name="bytebase-store-PlanConfig-RestoreDatabaseConfig"></a>

### PlanConfig.RestoreDatabaseConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| target | [string](#string) |  | The resource name of the target to restore. Format: instances/{instance}/databases/{database} |
| create_database_config | [PlanConfig.CreateDatabaseConfig](#bytebase-store-PlanConfig-CreateDatabaseConfig) | optional | create_database_config is present if the user wants to restore to a new database. |
| backup | [string](#string) |  | Restore from a backup. Format: instances/{instance}/databases/{database}/backups/{backup-name} |
| point_in_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | After the PITR operations, the database will be recovered to the state at this time. |






<a name="bytebase-store-PlanConfig-Spec"></a>

### PlanConfig.Spec



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| earliest_allowed_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | earliest_allowed_time the earliest execution time of the change. |
| id | [string](#string) |  | A UUID4 string that uniquely identifies the Spec. |
| create_database_config | [PlanConfig.CreateDatabaseConfig](#bytebase-store-PlanConfig-CreateDatabaseConfig) |  |  |
| change_database_config | [PlanConfig.ChangeDatabaseConfig](#bytebase-store-PlanConfig-ChangeDatabaseConfig) |  |  |
| restore_database_config | [PlanConfig.RestoreDatabaseConfig](#bytebase-store-PlanConfig-RestoreDatabaseConfig) |  |  |






<a name="bytebase-store-PlanConfig-Step"></a>

### PlanConfig.Step



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| specs | [PlanConfig.Spec](#bytebase-store-PlanConfig-Spec) | repeated |  |





 


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
| database_group_uid | [int64](#int64) | optional | database_group_uid is optional. If it&#39;s set, it means the database is part of a database group. |






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
| code | [int64](#int64) |  |  |
| sql_summary_report | [PlanCheckRunResult.Result.SqlSummaryReport](#bytebase-store-PlanCheckRunResult-Result-SqlSummaryReport) |  |  |
| sql_review_report | [PlanCheckRunResult.Result.SqlReviewReport](#bytebase-store-PlanCheckRunResult-Result-SqlReviewReport) |  |  |






<a name="bytebase-store-PlanCheckRunResult-Result-SqlReviewReport"></a>

### PlanCheckRunResult.Result.SqlReviewReport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| line | [int64](#int64) |  |  |
| column | [int64](#int64) |  |  |
| detail | [string](#string) |  |  |
| code | [int64](#int64) |  | Code from sql review. |






<a name="bytebase-store-PlanCheckRunResult-Result-SqlSummaryReport"></a>

### PlanCheckRunResult.Result.SqlSummaryReport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [int64](#int64) |  |  |
| statement_types | [string](#string) | repeated | statement_types are the types of statements that are found in the sql. |
| affected_rows | [int64](#int64) |  |  |
| changed_resources | [ChangedResources](#bytebase-store-ChangedResources) |  |  |





 


<a name="bytebase-store-PlanCheckRunConfig-ChangeDatabaseType"></a>

### PlanCheckRunConfig.ChangeDatabaseType


| Name | Number | Description |
| ---- | ------ | ----------- |
| CHANGE_DATABASE_TYPE_UNSPECIFIED | 0 |  |
| DDL | 1 |  |
| DML | 2 |  |
| SDL | 3 |  |



<a name="bytebase-store-PlanCheckRunResult-Result-Status"></a>

### PlanCheckRunResult.Result.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| ERROR | 1 |  |
| WARNING | 2 |  |
| SUCCESS | 3 |  |


 

 

 



<a name="store_policy-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/policy.proto



<a name="bytebase-store-Binding"></a>

### Binding
Reference: https://cloud.google.com/pubsub/docs/reference/rpc/google.iam.v1#binding


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [string](#string) |  | Role that is assigned to the list of members. Format: roles/{role} |
| members | [string](#string) | repeated | Specifies the principals requesting access for a Bytebase resource. `members` can have the following values:

* `allUsers`: A special identifier that represents anyone. * `user:{emailid}`: An email address that represents a specific Bytebase account. For example, `alice@example.com`. |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this binding. If the condition evaluates to true, then this binding applies to the current request. If the condition evaluates to false, then this binding does not apply to the current request. However, a different role binding might grant the same role to one or more of the principals in this binding. |






<a name="bytebase-store-IamPolicy"></a>

### IamPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bindings | [Binding](#bytebase-store-Binding) | repeated | Collection of binding. |






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

* `user:{emailid}`: An email address that represents a specific Bytebase account. For example, `alice@example.com`. |
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






<a name="bytebase-store-RolloutPolicy"></a>

### RolloutPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| automatic | [bool](#bool) |  |  |
| workspace_roles | [string](#string) | repeated |  |
| project_roles | [string](#string) | repeated |  |
| issue_roles | [string](#string) | repeated | roles/LAST_APPROVER roles/CREATOR |






<a name="bytebase-store-SQLReviewPolicy"></a>

### SQLReviewPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| rule_list | [SQLReviewRule](#bytebase-store-SQLReviewRule) | repeated |  |






<a name="bytebase-store-SQLReviewRule"></a>

### SQLReviewRule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  |  |
| level | [SQLReviewRuleLevel](#bytebase-store-SQLReviewRuleLevel) |  |  |
| payload | [string](#string) |  |  |
| engine | [Engine](#bytebase-store-Engine) |  |  |
| comment | [string](#string) |  |  |





 


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



<a name="bytebase-store-Project"></a>

### Project



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| protection_rules | [ProtectionRule](#bytebase-store-ProtectionRule) | repeated |  |






<a name="bytebase-store-ProtectionRule"></a>

### ProtectionRule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | A unique identifier for a node in UUID format. |
| target | [ProtectionRule.Target](#bytebase-store-ProtectionRule-Target) |  |  |
| name_filter | [string](#string) |  | The name of the branch/changelist or wildcard. |
| create_allowed_roles | [string](#string) | repeated | The roles allowed to create branches or changelists. Format: roles/OWNER. |





 


<a name="bytebase-store-ProtectionRule-Target"></a>

### ProtectionRule.Target
The type of target.

| Name | Number | Description |
| ---- | ------ | ----------- |
| PROTECTION_TARGET_UNSPECIFIED | 0 |  |
| BRANCH | 1 |  |
| CHANGELIST | 2 |  |


 

 

 



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
| category | [string](#string) |  | Category is the category for masking algorithm. Currently, it accepts 2 categories only: MASKING and HASHING. The range of accepted Payload is decided by the category. Mask: FullMask, RangeMask Hash: MD5Mask |
| full_mask | [MaskingAlgorithmSetting.Algorithm.FullMask](#bytebase-store-MaskingAlgorithmSetting-Algorithm-FullMask) |  |  |
| range_mask | [MaskingAlgorithmSetting.Algorithm.RangeMask](#bytebase-store-MaskingAlgorithmSetting-Algorithm-RangeMask) |  |  |
| md5_mask | [MaskingAlgorithmSetting.Algorithm.MD5Mask](#bytebase-store-MaskingAlgorithmSetting-Algorithm-MD5Mask) |  |  |






<a name="bytebase-store-MaskingAlgorithmSetting-Algorithm-FullMask"></a>

### MaskingAlgorithmSetting.Algorithm.FullMask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| substitution | [string](#string) |  | substitution is the string used to replace the original value, the max length of the string is 16 bytes. |






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
| expression | [google.api.expr.v1alpha1.ParsedExpr](#google-api-expr-v1alpha1-ParsedExpr) |  |  |
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





 


<a name="bytebase-store-Announcement-AlertLevel"></a>

### Announcement.AlertLevel
We support three levels of AlertLevel: INFO, WARNING, and ERROR.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ALERT_LEVEL_UNSPECIFIED | 0 |  |
| ALERT_LEVEL_INFO | 1 |  |
| ALERT_LEVEL_WARNING | 2 |  |
| ALERT_LEVEL_CRITICAL | 3 |  |



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



<a name="bytebase-store-SheetPayload"></a>

### SheetPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [SheetPayload.Type](#bytebase-store-SheetPayload-Type) |  |  |
| vcs_payload | [SheetPayload.VCSPayload](#bytebase-store-SheetPayload-VCSPayload) |  |  |
| schema_design | [SheetPayload.SchemaDesign](#bytebase-store-SheetPayload-SchemaDesign) |  |  |
| database_config | [DatabaseConfig](#bytebase-store-DatabaseConfig) |  | The snapshot of the database config when creating the sheet, be used to compare with the baseline_database_config and apply the diff to the database. |
| baseline_database_config | [DatabaseConfig](#bytebase-store-DatabaseConfig) |  | The snapshot of the baseline database config when creating the sheet. |






<a name="bytebase-store-SheetPayload-SchemaDesign"></a>

### SheetPayload.SchemaDesign



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [SheetPayload.SchemaDesign.Type](#bytebase-store-SheetPayload-SchemaDesign-Type) |  | The type of the schema design. |
| engine | [Engine](#bytebase-store-Engine) |  | The database instance engine of the schema design. |
| baseline_sheet_id | [string](#string) |  | The id of the baseline sheet including the baseline full schema. |
| baseline_schema_design_id | [string](#string) |  | The sheet id of the baseline schema design. Only valid when the schema design is a personal draft. |
| baseline_change_history_id | [string](#string) |  | The id of the baseline change history including the baseline change history.(optional) |
| protection | [SheetPayload.SchemaDesign.Protection](#bytebase-store-SheetPayload-SchemaDesign-Protection) |  | The protection of the schema design. |






<a name="bytebase-store-SheetPayload-SchemaDesign-Protection"></a>

### SheetPayload.SchemaDesign.Protection



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| allow_force_pushes | [bool](#bool) |  | Permits force pushes to the branch. |






<a name="bytebase-store-SheetPayload-VCSPayload"></a>

### SheetPayload.VCSPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file_name | [string](#string) |  |  |
| file_path | [string](#string) |  |  |
| size | [int64](#int64) |  |  |
| author | [string](#string) |  |  |
| last_commit_id | [string](#string) |  |  |
| last_sync_ts | [int64](#int64) |  |  |
| push_event | [PushEvent](#bytebase-store-PushEvent) |  |  |





 


<a name="bytebase-store-SheetPayload-SchemaDesign-Type"></a>

### SheetPayload.SchemaDesign.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| MAIN_BRANCH | 1 | Main branch type is the main version of schema design. And only allow to be updated/merged with personal drafts. |
| PERSONAL_DRAFT | 2 | Personal draft type is a copy of the main branch type schema designs. |



<a name="bytebase-store-SheetPayload-Type"></a>

### SheetPayload.Type
Type of the SheetPayload.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| SCHEMA_DESIGN | 1 |  |


 

 

 



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
| rows_sent | [int64](#int64) |  | rows_sent is the number of rows sent by the slow query. |
| rows_examined | [int64](#int64) |  | rows_examined is the number of rows examined by the slow query. |
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
| count | [int64](#int64) |  | count is the number of slow queries with the same fingerprint. |
| latest_log_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | latest_log_time is the time of the latest slow query with the same fingerprint. |
| total_query_time | [google.protobuf.Duration](#google-protobuf-Duration) |  | The total query time of the slow query log. |
| maximum_query_time | [google.protobuf.Duration](#google-protobuf-Duration) |  | The maximum query time of the slow query log. |
| total_rows_sent | [int64](#int64) |  | The total rows sent of the slow query log. |
| maximum_rows_sent | [int64](#int64) |  | The maximum rows sent of the slow query log. |
| total_rows_examined | [int64](#int64) |  | The total rows examined of the slow query log. |
| maximum_rows_examined | [int64](#int64) |  | The maximum rows examined of the slow query log. |
| samples | [SlowQueryDetails](#bytebase-store-SlowQueryDetails) | repeated | samples are the details of the sample slow queries with the same fingerprint. |





 

 

 

 



<a name="store_task_run-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/task_run.proto



<a name="bytebase-store-TaskRunResult"></a>

### TaskRunResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| detail | [string](#string) |  |  |
| change_history | [string](#string) |  | Format: instances/{instance}/databases/{database}/changeHistories/{changeHistory} |
| version | [string](#string) |  |  |





 

 

 

 



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

