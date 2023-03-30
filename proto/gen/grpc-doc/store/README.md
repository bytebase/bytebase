# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [store/activity.proto](#store_activity-proto)
    - [ActivityIssueCommentCreatePayload](#bytebase-store-ActivityIssueCommentCreatePayload)
    - [ActivityIssueCommentCreatePayload.ApprovalEvent](#bytebase-store-ActivityIssueCommentCreatePayload-ApprovalEvent)
    - [ActivityIssueCommentCreatePayload.ExternalApprovalEvent](#bytebase-store-ActivityIssueCommentCreatePayload-ExternalApprovalEvent)
    - [ActivityIssueCommentCreatePayload.TaskRollbackBy](#bytebase-store-ActivityIssueCommentCreatePayload-TaskRollbackBy)
    - [ActivityIssueCreatePayload](#bytebase-store-ActivityIssueCreatePayload)
    - [ActivityPayload](#bytebase-store-ActivityPayload)
  
    - [ActivityIssueCommentCreatePayload.ApprovalEvent.Status](#bytebase-store-ActivityIssueCommentCreatePayload-ApprovalEvent-Status)
    - [ActivityIssueCommentCreatePayload.ExternalApprovalEvent.Action](#bytebase-store-ActivityIssueCommentCreatePayload-ExternalApprovalEvent-Action)
    - [ActivityIssueCommentCreatePayload.ExternalApprovalEvent.Type](#bytebase-store-ActivityIssueCommentCreatePayload-ExternalApprovalEvent-Type)
  
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
  
- [store/data_source.proto](#store_data_source-proto)
    - [DataSourceOptions](#bytebase-store-DataSourceOptions)
  
- [store/database.proto](#store_database-proto)
    - [ColumnMetadata](#bytebase-store-ColumnMetadata)
    - [DatabaseMetadata](#bytebase-store-DatabaseMetadata)
    - [DependentColumn](#bytebase-store-DependentColumn)
    - [ExtensionMetadata](#bytebase-store-ExtensionMetadata)
    - [ForeignKeyMetadata](#bytebase-store-ForeignKeyMetadata)
    - [IndexMetadata](#bytebase-store-IndexMetadata)
    - [InstanceRoleMetadata](#bytebase-store-InstanceRoleMetadata)
    - [SchemaMetadata](#bytebase-store-SchemaMetadata)
    - [TableMetadata](#bytebase-store-TableMetadata)
    - [ViewMetadata](#bytebase-store-ViewMetadata)
  
- [store/idp.proto](#store_idp-proto)
    - [FieldMapping](#bytebase-store-FieldMapping)
    - [IdentityProviderConfig](#bytebase-store-IdentityProviderConfig)
    - [IdentityProviderUserInfo](#bytebase-store-IdentityProviderUserInfo)
    - [OAuth2IdentityProviderConfig](#bytebase-store-OAuth2IdentityProviderConfig)
    - [OIDCIdentityProviderConfig](#bytebase-store-OIDCIdentityProviderConfig)
  
    - [IdentityProviderType](#bytebase-store-IdentityProviderType)
  
- [store/issue.proto](#store_issue-proto)
    - [IssuePayload](#bytebase-store-IssuePayload)
  
- [store/setting.proto](#store_setting-proto)
    - [AgentPluginSetting](#bytebase-store-AgentPluginSetting)
    - [WorkspaceApprovalSetting](#bytebase-store-WorkspaceApprovalSetting)
    - [WorkspaceApprovalSetting.Rule](#bytebase-store-WorkspaceApprovalSetting-Rule)
    - [WorkspaceProfileSetting](#bytebase-store-WorkspaceProfileSetting)
  
- [store/user.proto](#store_user-proto)
    - [MFAConfig](#bytebase-store-MFAConfig)
  
- [Scalar Value Types](#scalar-value-types)



<a name="store_activity-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/activity.proto



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






<a name="bytebase-store-ActivityPayload"></a>

### ActivityPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issue_create_payload | [ActivityIssueCreatePayload](#bytebase-store-ActivityIssueCreatePayload) |  |  |
| issue_comment_create_payload | [ActivityIssueCommentCreatePayload](#bytebase-store-ActivityIssueCommentCreatePayload) |  |  |





 


<a name="bytebase-store-ActivityIssueCommentCreatePayload-ApprovalEvent-Status"></a>

### ActivityIssueCommentCreatePayload.ApprovalEvent.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| APPROVED | 2 |  |



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
GroupValue is used if ApprovalNode Type is ANY_IN_GROUP
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





 

 

 

 



<a name="store_database-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/database.proto



<a name="bytebase-store-ColumnMetadata"></a>

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
| comment | [string](#string) |  | The comment is the comment of a column. |






<a name="bytebase-store-DatabaseMetadata"></a>

### DatabaseMetadata
DatabaseMetadata is the metadata for databases.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| schemas | [SchemaMetadata](#bytebase-store-SchemaMetadata) | repeated | The schemas is the list of schemas in a database. |
| character_set | [string](#string) |  | The character_set is the character set of a database. |
| collation | [string](#string) |  | The collation is the collation of a database. |
| extensions | [ExtensionMetadata](#bytebase-store-ExtensionMetadata) | repeated | The extensions is the list of extensions in a database. |






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






<a name="bytebase-store-SchemaMetadata"></a>

### SchemaMetadata
SchemaMetadata is the metadata for schemas.
This is the concept of schema in Postgres, but it&#39;s a no-op for MySQL.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the schema name. It is an empty string for databases without such concept such as MySQL. |
| tables | [TableMetadata](#bytebase-store-TableMetadata) | repeated | The tables is the list of tables in a schema. |
| views | [ViewMetadata](#bytebase-store-ViewMetadata) | repeated | The views is the list of views in a schema. |






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
| comment | [string](#string) |  | The comment is the comment of a table. |
| foreign_keys | [ForeignKeyMetadata](#bytebase-store-ForeignKeyMetadata) | repeated | The foreign_keys is the list of foreign keys in a table. |






<a name="bytebase-store-ViewMetadata"></a>

### ViewMetadata
ViewMetadata is the metadata for views.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a view. |
| definition | [string](#string) |  | The definition is the definition of a view. |
| comment | [string](#string) |  | The comment is the comment of a view. |
| dependent_columns | [DependentColumn](#bytebase-store-DependentColumn) | repeated | The dependent_columns is the list of dependent columns of a view. |





 

 

 

 



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
| display_name | [string](#string) |  | DisplayName is the field name of display name in 3rd-party idp user info. Required. |
| email | [string](#string) |  | Email is the field name of primary email in 3rd-party idp user info. Required. |






<a name="bytebase-store-IdentityProviderConfig"></a>

### IdentityProviderConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| oauth2_config | [OAuth2IdentityProviderConfig](#bytebase-store-OAuth2IdentityProviderConfig) |  |  |
| oidc_config | [OIDCIdentityProviderConfig](#bytebase-store-OIDCIdentityProviderConfig) |  |  |






<a name="bytebase-store-IdentityProviderUserInfo"></a>

### IdentityProviderUserInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identifier | [string](#string) |  | Identifier is the value of the unique identifier in 3rd-party idp user info. |
| display_name | [string](#string) |  | DisplayName is the value of display name in 3rd-party idp user info. |
| email | [string](#string) |  | Email is the value of primary email in 3rd-party idp user info. |






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






<a name="bytebase-store-OIDCIdentityProviderConfig"></a>

### OIDCIdentityProviderConfig
OIDCIdentityProviderConfig is the structure for OIDC identity provider config.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issuer | [string](#string) |  |  |
| client_id | [string](#string) |  |  |
| client_secret | [string](#string) |  |  |
| field_mapping | [FieldMapping](#bytebase-store-FieldMapping) |  |  |





 


<a name="bytebase-store-IdentityProviderType"></a>

### IdentityProviderType


| Name | Number | Description |
| ---- | ------ | ----------- |
| IDENTITY_PROVIDER_TYPE_UNSPECIFIED | 0 |  |
| OAUTH2 | 1 |  |
| OIDC | 2 |  |


 

 

 



<a name="store_issue-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/issue.proto



<a name="bytebase-store-IssuePayload"></a>

### IssuePayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| approval | [IssuePayloadApproval](#bytebase-store-IssuePayloadApproval) |  |  |





 

 

 

 



<a name="store_setting-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## store/setting.proto



<a name="bytebase-store-AgentPluginSetting"></a>

### AgentPluginSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | [string](#string) |  | The URL for the agent API. |
| token | [string](#string) |  | The token for the agent. |






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






<a name="bytebase-store-WorkspaceProfileSetting"></a>

### WorkspaceProfileSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| external_url | [string](#string) |  | The URL user visits Bytebase.

The external URL is used for: 1. Constructing the correct callback URL when configuring the VCS provider. The callback URL points to the frontend. 2. Creating the correct webhook endpoint when configuring the project GitOps workflow. The webhook endpoint points to the backend. |
| disallow_signup | [bool](#bool) |  | Disallow self-service signup, users can only be invited by the owner. |





 

 

 

 



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

