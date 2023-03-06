# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [v1/actuator_service.proto](#v1_actuator_service-proto)
    - [ActuatorInfo](#bytebase-v1-ActuatorInfo)
    - [GetActuatorInfoRequest](#bytebase-v1-GetActuatorInfoRequest)
  
    - [ActuatorService](#bytebase-v1-ActuatorService)
  
- [v1/common.proto](#v1_common-proto)
    - [Engine](#bytebase-v1-Engine)
    - [State](#bytebase-v1-State)
  
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
    - [Anomaly.BackupPlanSchedule](#bytebase-v1-Anomaly-BackupPlanSchedule)
  
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
  
- [v1/database_service.proto](#v1_database_service-proto)
    - [Backup](#bytebase-v1-Backup)
    - [BackupSetting](#bytebase-v1-BackupSetting)
    - [BatchUpdateDatabasesRequest](#bytebase-v1-BatchUpdateDatabasesRequest)
    - [BatchUpdateDatabasesResponse](#bytebase-v1-BatchUpdateDatabasesResponse)
    - [ColumnMetadata](#bytebase-v1-ColumnMetadata)
    - [CreateBackupRequest](#bytebase-v1-CreateBackupRequest)
    - [Database](#bytebase-v1-Database)
    - [Database.LabelsEntry](#bytebase-v1-Database-LabelsEntry)
    - [DatabaseMetadata](#bytebase-v1-DatabaseMetadata)
    - [DatabaseSchema](#bytebase-v1-DatabaseSchema)
    - [ExtensionMetadata](#bytebase-v1-ExtensionMetadata)
    - [ForeignKeyMetadata](#bytebase-v1-ForeignKeyMetadata)
    - [GetBackupSettingRequest](#bytebase-v1-GetBackupSettingRequest)
    - [GetDatabaseMetadataRequest](#bytebase-v1-GetDatabaseMetadataRequest)
    - [GetDatabaseRequest](#bytebase-v1-GetDatabaseRequest)
    - [GetDatabaseSchemaRequest](#bytebase-v1-GetDatabaseSchemaRequest)
    - [IndexMetadata](#bytebase-v1-IndexMetadata)
    - [ListBackupRequest](#bytebase-v1-ListBackupRequest)
    - [ListBackupResponse](#bytebase-v1-ListBackupResponse)
    - [ListDatabasesRequest](#bytebase-v1-ListDatabasesRequest)
    - [ListDatabasesResponse](#bytebase-v1-ListDatabasesResponse)
    - [SchemaMetadata](#bytebase-v1-SchemaMetadata)
    - [TableMetadata](#bytebase-v1-TableMetadata)
    - [UpdateBackupSettingRequest](#bytebase-v1-UpdateBackupSettingRequest)
    - [UpdateDatabaseRequest](#bytebase-v1-UpdateDatabaseRequest)
    - [ViewMetadata](#bytebase-v1-ViewMetadata)
  
    - [Backup.BackupState](#bytebase-v1-Backup-BackupState)
    - [Backup.BackupType](#bytebase-v1-Backup-BackupType)
  
    - [DatabaseService](#bytebase-v1-DatabaseService)
  
- [v1/deployment.proto](#v1_deployment-proto)
    - [DeploymentType](#bytebase-v1-DeploymentType)
  
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
  
- [v1/idp_service.proto](#v1_idp_service-proto)
    - [CreateIdentityProviderRequest](#bytebase-v1-CreateIdentityProviderRequest)
    - [DeleteIdentityProviderRequest](#bytebase-v1-DeleteIdentityProviderRequest)
    - [FieldMapping](#bytebase-v1-FieldMapping)
    - [GetIdentityProviderRequest](#bytebase-v1-GetIdentityProviderRequest)
    - [IdentityProvider](#bytebase-v1-IdentityProvider)
    - [IdentityProviderConfig](#bytebase-v1-IdentityProviderConfig)
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
  
    - [IdentityProviderService](#bytebase-v1-IdentityProviderService)
  
- [v1/instance_role_service.proto](#v1_instance_role_service-proto)
    - [CreateRoleRequest](#bytebase-v1-CreateRoleRequest)
    - [DeleteRoleRequest](#bytebase-v1-DeleteRoleRequest)
    - [GetRoleRequest](#bytebase-v1-GetRoleRequest)
    - [InstanceRole](#bytebase-v1-InstanceRole)
    - [ListRolesRequest](#bytebase-v1-ListRolesRequest)
    - [ListRolesResponse](#bytebase-v1-ListRolesResponse)
    - [UndeleteRoleRequest](#bytebase-v1-UndeleteRoleRequest)
    - [UpdateRoleRequest](#bytebase-v1-UpdateRoleRequest)
  
    - [InstanceRoleService](#bytebase-v1-InstanceRoleService)
  
- [v1/instance_service.proto](#v1_instance_service-proto)
    - [AddDataSourceRequest](#bytebase-v1-AddDataSourceRequest)
    - [CreateInstanceRequest](#bytebase-v1-CreateInstanceRequest)
    - [DataSource](#bytebase-v1-DataSource)
    - [DeleteInstanceRequest](#bytebase-v1-DeleteInstanceRequest)
    - [GetInstanceRequest](#bytebase-v1-GetInstanceRequest)
    - [Instance](#bytebase-v1-Instance)
    - [ListInstancesRequest](#bytebase-v1-ListInstancesRequest)
    - [ListInstancesResponse](#bytebase-v1-ListInstancesResponse)
    - [RemoveDataSourceRequest](#bytebase-v1-RemoveDataSourceRequest)
    - [UndeleteInstanceRequest](#bytebase-v1-UndeleteInstanceRequest)
    - [UpdateDataSourceRequest](#bytebase-v1-UpdateDataSourceRequest)
    - [UpdateInstanceRequest](#bytebase-v1-UpdateInstanceRequest)
  
    - [DataSourceType](#bytebase-v1-DataSourceType)
  
    - [InstanceService](#bytebase-v1-InstanceService)
  
- [v1/org_policy_service.proto](#v1_org_policy_service-proto)
    - [AccessControlPolicy](#bytebase-v1-AccessControlPolicy)
    - [AccessControlRule](#bytebase-v1-AccessControlRule)
    - [BackupPlanPolicy](#bytebase-v1-BackupPlanPolicy)
    - [CreatePolicyRequest](#bytebase-v1-CreatePolicyRequest)
    - [DeletePolicyRequest](#bytebase-v1-DeletePolicyRequest)
    - [DeploymentApprovalPolicy](#bytebase-v1-DeploymentApprovalPolicy)
    - [DeploymentApprovalStrategy](#bytebase-v1-DeploymentApprovalStrategy)
    - [GetPolicyRequest](#bytebase-v1-GetPolicyRequest)
    - [ListPoliciesRequest](#bytebase-v1-ListPoliciesRequest)
    - [ListPoliciesResponse](#bytebase-v1-ListPoliciesResponse)
    - [Policy](#bytebase-v1-Policy)
    - [SQLReviewPolicy](#bytebase-v1-SQLReviewPolicy)
    - [SQLReviewRule](#bytebase-v1-SQLReviewRule)
    - [SensitiveData](#bytebase-v1-SensitiveData)
    - [SensitiveDataPolicy](#bytebase-v1-SensitiveDataPolicy)
    - [UpdatePolicyRequest](#bytebase-v1-UpdatePolicyRequest)
  
    - [ApprovalGroup](#bytebase-v1-ApprovalGroup)
    - [ApprovalStrategy](#bytebase-v1-ApprovalStrategy)
    - [BackupPlanSchedule](#bytebase-v1-BackupPlanSchedule)
    - [PolicyType](#bytebase-v1-PolicyType)
    - [SQLReviewRuleLevel](#bytebase-v1-SQLReviewRuleLevel)
    - [SensitiveDataMaskType](#bytebase-v1-SensitiveDataMaskType)
  
    - [OrgPolicyService](#bytebase-v1-OrgPolicyService)
  
- [v1/project_service.proto](#v1_project_service-proto)
    - [Activity](#bytebase-v1-Activity)
    - [AddWebhookRequest](#bytebase-v1-AddWebhookRequest)
    - [BatchUpdateReviewsRequest](#bytebase-v1-BatchUpdateReviewsRequest)
    - [BatchUpdateReviewsResponse](#bytebase-v1-BatchUpdateReviewsResponse)
    - [Binding](#bytebase-v1-Binding)
    - [CreateProjectRequest](#bytebase-v1-CreateProjectRequest)
    - [DeleteProjectRequest](#bytebase-v1-DeleteProjectRequest)
    - [DeploymentConfig](#bytebase-v1-DeploymentConfig)
    - [DeploymentSpec](#bytebase-v1-DeploymentSpec)
    - [GetDeploymentConfigRequest](#bytebase-v1-GetDeploymentConfigRequest)
    - [GetIamPolicyRequest](#bytebase-v1-GetIamPolicyRequest)
    - [GetProjectRequest](#bytebase-v1-GetProjectRequest)
    - [GetReviewRequest](#bytebase-v1-GetReviewRequest)
    - [IamPolicy](#bytebase-v1-IamPolicy)
    - [LabelSelector](#bytebase-v1-LabelSelector)
    - [LabelSelectorRequirement](#bytebase-v1-LabelSelectorRequirement)
    - [ListProjectsRequest](#bytebase-v1-ListProjectsRequest)
    - [ListProjectsResponse](#bytebase-v1-ListProjectsResponse)
    - [ListReviewsRequest](#bytebase-v1-ListReviewsRequest)
    - [ListReviewsResponse](#bytebase-v1-ListReviewsResponse)
    - [Project](#bytebase-v1-Project)
    - [RemoveWebhookRequest](#bytebase-v1-RemoveWebhookRequest)
    - [Review](#bytebase-v1-Review)
    - [Schedule](#bytebase-v1-Schedule)
    - [ScheduleDeployment](#bytebase-v1-ScheduleDeployment)
    - [SetIamPolicyRequest](#bytebase-v1-SetIamPolicyRequest)
    - [TestWebhookRequest](#bytebase-v1-TestWebhookRequest)
    - [TestWebhookResponse](#bytebase-v1-TestWebhookResponse)
    - [UndeleteProjectRequest](#bytebase-v1-UndeleteProjectRequest)
    - [UpdateDeploymentConfigRequest](#bytebase-v1-UpdateDeploymentConfigRequest)
    - [UpdateProjectRequest](#bytebase-v1-UpdateProjectRequest)
    - [UpdateReviewRequest](#bytebase-v1-UpdateReviewRequest)
    - [UpdateWebhookRequest](#bytebase-v1-UpdateWebhookRequest)
    - [Webhook](#bytebase-v1-Webhook)
  
    - [Activity.Type](#bytebase-v1-Activity-Type)
    - [LgtmCheck](#bytebase-v1-LgtmCheck)
    - [OperatorType](#bytebase-v1-OperatorType)
    - [ProjectRole](#bytebase-v1-ProjectRole)
    - [ReviewStatus](#bytebase-v1-ReviewStatus)
    - [SchemaChange](#bytebase-v1-SchemaChange)
    - [SchemaVersion](#bytebase-v1-SchemaVersion)
    - [TenantMode](#bytebase-v1-TenantMode)
    - [Visibility](#bytebase-v1-Visibility)
    - [Webhook.Type](#bytebase-v1-Webhook-Type)
    - [Workflow](#bytebase-v1-Workflow)
  
    - [ProjectService](#bytebase-v1-ProjectService)
  
- [v1/setting_service.proto](#v1_setting_service-proto)
    - [GetSettingRequest](#bytebase-v1-GetSettingRequest)
    - [GetSettingResponse](#bytebase-v1-GetSettingResponse)
    - [SetSettingRequest](#bytebase-v1-SetSettingRequest)
    - [Setting](#bytebase-v1-Setting)
    - [Value](#bytebase-v1-Value)
  
    - [SettingService](#bytebase-v1-SettingService)
  
- [v1/sql_service.proto](#v1_sql_service-proto)
    - [PrettyRequest](#bytebase-v1-PrettyRequest)
    - [PrettyResponse](#bytebase-v1-PrettyResponse)
  
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






<a name="bytebase-v1-GetActuatorInfoRequest"></a>

### GetActuatorInfoRequest






 

 

 


<a name="bytebase-v1-ActuatorService"></a>

### ActuatorService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetActuatorInfo | [GetActuatorInfoRequest](#bytebase-v1-GetActuatorInfoRequest) | [ActuatorInfo](#bytebase-v1-ActuatorInfo) |  |

 



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



<a name="bytebase-v1-State"></a>

### State


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATE_UNSPECIFIED | 0 |  |
| ACTIVE | 1 |  |
| DELETED | 2 |  |


 

 

 



<a name="v1_anomaly_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/anomaly_service.proto



<a name="bytebase-v1-Anomaly"></a>

### Anomaly



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource | [string](#string) |  | The resource that is the target of the operation. Format: - Instance: environments/{environment}/instnaces/{instance} - Database: environments/{environment}/instnaces/{instance}/databases/{database} |
| type | [Anomaly.AnomalyType](#bytebase-v1-Anomaly-AnomalyType) |  | type is the type of the anomaly. |
| severity | [Anomaly.AnomalySeverity](#bytebase-v1-Anomaly-AnomalySeverity) |  | severity is the severity of the anomaly. |
| instance_connection_detail | [Anomaly.InstanceConnectionDetail](#bytebase-v1-Anomaly-InstanceConnectionDetail) |  |  |
| database_connection_detail | [Anomaly.DatabaseConnectionDetail](#bytebase-v1-Anomaly-DatabaseConnectionDetail) |  |  |
| database_backup_policy_violation_detail | [Anomaly.DatabaseBackupPolicyViolationDetail](#bytebase-v1-Anomaly-DatabaseBackupPolicyViolationDetail) |  |  |
| database_backup_missing_detail | [Anomaly.DatabaseBackupMissingDetail](#bytebase-v1-Anomaly-DatabaseBackupMissingDetail) |  |  |
| database_schema_drift_detail | [Anomaly.DatabaseSchemaDriftDetail](#bytebase-v1-Anomaly-DatabaseSchemaDriftDetail) |  |  |






<a name="bytebase-v1-Anomaly-DatabaseBackupMissingDetail"></a>

### Anomaly.DatabaseBackupMissingDetail
DatabaseBackupMissingDetail is the detail for database backup missing anomaly.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expected_schedule | [Anomaly.BackupPlanSchedule](#bytebase-v1-Anomaly-BackupPlanSchedule) |  | expected_schedule is the expected backup plan schedule in the database. |
| latest_backup_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | latest_backup_time is the latest backup time in the database. |






<a name="bytebase-v1-Anomaly-DatabaseBackupPolicyViolationDetail"></a>

### Anomaly.DatabaseBackupPolicyViolationDetail
DatabaseBackupPolicyViolationDetail is the detail for database backup policy violation anomaly.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | parent is the parent of the database. Format: environments/{environment} |
| expected_schedule | [Anomaly.BackupPlanSchedule](#bytebase-v1-Anomaly-BackupPlanSchedule) |  | expected_schedule is the expected backup plan schedule in the parent. |
| actual_schedule | [Anomaly.BackupPlanSchedule](#bytebase-v1-Anomaly-BackupPlanSchedule) |  | actual_schedule is the actual backup plan schedule in the database. |






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
| filter | [string](#string) |  | filter is the filter to apply on the search anomaly request, follow the [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) syntax. Only support filter by resource and type for now. For example: Search the anomalies of a specific resource: &#39;resource=&#34;environments/{environemnt}/instances/{instance}&#34;.&#39; Search the specified types of anomalies: &#39;type=&#34;DATABASE_BACKUP_POLICY_VIOLATION&#34; | &#34;MIGRATION_SCHEMA&#34;.&#39; |
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



<a name="bytebase-v1-Anomaly-BackupPlanSchedule"></a>

### Anomaly.BackupPlanSchedule
BackupPlanSchedule is the backup plan schedule.

| Name | Number | Description |
| ---- | ------ | ----------- |
| BACKUP_PLAN_SCHEDULE_UNSPECIFIED | 0 | Unspecified backup plan schedule. |
| UNSET | 1 | UNSET is the unset backup plan schedule. |
| DAILY | 2 | DAILY is the daily backup plan schedule. |
| WEEKLY | 3 | WEEKLY is the weekly backup plan schedule. |


 

 


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
| mfa_code | [string](#string) | optional | The mfa_code is used to verify the user&#39;s identity by MFA. |
| recovery_code | [string](#string) | optional | The recovery_code is used to recovery the user&#39;s identity with MFA. |






<a name="bytebase-v1-LoginResponse"></a>

### LoginResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  |  |






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
| name | [string](#string) |  | The name of the bookmark. Format: users/{user}/bookmarks/{bookmark}, user and bookmark are server-generated unique IDs. |
| title | [string](#string) |  | The title of the bookmark. |
| link | [string](#string) |  | The resource link of the bookmark. Only support issue link for now. Format: Issue: /issue/slug(issue_name)-{issue_uid} Example: /issue/start-here-add-email-column-to-employee-table-101 |






<a name="bytebase-v1-CreateBookmarkRequest"></a>

### CreateBookmarkRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource of the bookmark. Format: users/{user}, user is a server-generated unique IDs. |
| bookmark | [Bookmark](#bytebase-v1-Bookmark) |  | The bookmark to create. |






<a name="bytebase-v1-DeleteBookmarkRequest"></a>

### DeleteBookmarkRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the bookmark to delete. Format: users/{user}/bookmarks/{bookmark}, user and bookmark are server-generated unique IDs. |






<a name="bytebase-v1-ListBookmarksRequest"></a>

### ListBookmarksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource of the bookmark. Format: users/{user}, user is a server-generated unique ID. |
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
| ListBookmarks | [ListBookmarksRequest](#bytebase-v1-ListBookmarksRequest) | [ListBookmarksResponse](#bytebase-v1-ListBookmarksResponse) | ListBookmark lists bookmarks. |

 



<a name="v1_database_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/database_service.proto



<a name="bytebase-v1-Backup"></a>

### Backup
The message of the backup.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The resource name of the database backup. backup-name is specified by the client. Format: environments/{environment}/instances/{instance}/databases/{database}/backups/{backup-name} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the backup resource was created initally. |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the backup resource was updated. |
| state | [Backup.BackupState](#bytebase-v1-Backup-BackupState) |  | The state of the backup. |
| backup_type | [Backup.BackupType](#bytebase-v1-Backup-BackupType) |  | The type of the backup. |
| comment | [string](#string) |  | The comment of the backup. |






<a name="bytebase-v1-BackupSetting"></a>

### BackupSetting
BackupSetting is the setting for database backup.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database backup setting. Format: environments/{environment}/instances/{instance}/databases/{database}/backupSettings |
| backup_retain_duration | [google.protobuf.Duration](#google-protobuf-Duration) |  | The default maximum age of a Backup created via this BackupPlan. If specified, a Backup will be automatically deleted after its age reaches. If not specified, Backups created under this BackupPlan will be deleted after 7 DAYS. It will be rounded up to the number of days. |
| cron_schedule | [string](#string) |  | Cron(https://wikipedia.com/wiki/cron) string that defines a repeating schedule for creating Backups. Support hour of day, day of week. (UTC time)

Default (empty): Disable automatic backup. |
| hook_url | [string](#string) |  | hook_url(https://www.bytebase.com/docs/administration/webhook-integration/database-webhook) is the URL to send a notification when a backup is created. |






<a name="bytebase-v1-BatchUpdateDatabasesRequest"></a>

### BatchUpdateDatabasesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource shared by all databases being updated. Format: environments/{environment}/instances/{instance} If the operation spans parents, a dash (-) may be accepted as a wildcard. We only support updating the project of databases for now. |
| requests | [UpdateDatabaseRequest](#bytebase-v1-UpdateDatabaseRequest) | repeated | The request message specifying the resources to update. A maximum of 1000 databases can be modified in a batch. |






<a name="bytebase-v1-BatchUpdateDatabasesResponse"></a>

### BatchUpdateDatabasesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [Database](#bytebase-v1-Database) | repeated | Databases updated. |






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
| comment | [string](#string) |  | The comment is the comment of a column. |






<a name="bytebase-v1-CreateBackupRequest"></a>

### CreateBackupRequest
CreateBackupRequest is the request message for CreateBackup.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this backup will be created. Format: environments/{environment}/instances/{instance}/databases/{database} |
| backup | [Backup](#bytebase-v1-Backup) |  |  |






<a name="bytebase-v1-Database"></a>

### Database



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database. Format: environments/{environment}/instances/{instance}/databases/{database} {database} is the database name in the instance. |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| sync_state | [State](#bytebase-v1-State) |  | The existence of a database on latest sync. |
| successful_sync_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The latest synchronization time. |
| project | [string](#string) |  | The project for a database. Format: projects/{project} |
| schema_version | [string](#string) |  | The version of database schema. |
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






<a name="bytebase-v1-GetBackupSettingRequest"></a>

### GetBackupSettingRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to retrieve backup setting. Format: environments/{environment}/instances/{instance}/databases/{database}/backupSetting |






<a name="bytebase-v1-GetDatabaseMetadataRequest"></a>

### GetDatabaseMetadataRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to retrieve metadata. Format: environments/{environment}/instances/{instance}/databases/{database} |






<a name="bytebase-v1-GetDatabaseRequest"></a>

### GetDatabaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to retrieve. Format: environments/{environment}/instances/{instance}/databases/{database} |






<a name="bytebase-v1-GetDatabaseSchemaRequest"></a>

### GetDatabaseSchemaRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to retrieve schema. Format: environments/{environment}/instances/{instance}/databases/{database} |






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






<a name="bytebase-v1-ListBackupRequest"></a>

### ListBackupRequest
ListBackupRequest is the request message for ListBackup.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this backup will be created. Format: environments/{environment}/instances/{instance}/databases/{database} |
| page_size | [int32](#int32) |  | Not used. The maximum number of backups to return. The service may return fewer than this value. If unspecified, at most 50 backups will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListBackup` call. Provide this to retrieve the subsequent page. 

When paginating, all other parameters provided to `ListBackup` must match the call that provided the page token. |






<a name="bytebase-v1-ListBackupResponse"></a>

### ListBackupResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| backups | [Backup](#bytebase-v1-Backup) | repeated | The backups from the specified request. |
| next_page_token | [string](#string) |  | Not used. A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-ListDatabasesRequest"></a>

### ListDatabasesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of databases. Format: environments/{environment}/instances/{instance} Use &#34;environments/-/instances/-&#34; to list all databases from all environments. |
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






<a name="bytebase-v1-SchemaMetadata"></a>

### SchemaMetadata
SchemaMetadata is the metadata for schemas.
This is the concept of schema in Postgres, but it&#39;s a no-op for MySQL.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the schema name. It is an empty string for databases without such concept such as MySQL. |
| tables | [TableMetadata](#bytebase-v1-TableMetadata) | repeated | The tables is the list of tables in a schema. |
| views | [ViewMetadata](#bytebase-v1-ViewMetadata) | repeated | The views is the list of views in a schema. |






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
| comment | [string](#string) |  | The comment is the comment of a table. |
| foreign_keys | [ForeignKeyMetadata](#bytebase-v1-ForeignKeyMetadata) | repeated | The foreign_keys is the list of foreign keys in a table. |






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

The database&#39;s `name` field is used to identify the database to update. Format: environments/{environment}/instances/{instance}/databases/{database} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






<a name="bytebase-v1-ViewMetadata"></a>

### ViewMetadata
ViewMetadata is the metadata for views.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a view. |
| definition | [string](#string) |  | The definition is the definition of a view. |
| comment | [string](#string) |  | The comment is the comment of a view. |





 


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


 

 


<a name="bytebase-v1-DatabaseService"></a>

### DatabaseService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetDatabase | [GetDatabaseRequest](#bytebase-v1-GetDatabaseRequest) | [Database](#bytebase-v1-Database) |  |
| ListDatabases | [ListDatabasesRequest](#bytebase-v1-ListDatabasesRequest) | [ListDatabasesResponse](#bytebase-v1-ListDatabasesResponse) |  |
| UpdateDatabase | [UpdateDatabaseRequest](#bytebase-v1-UpdateDatabaseRequest) | [Database](#bytebase-v1-Database) |  |
| BatchUpdateDatabases | [BatchUpdateDatabasesRequest](#bytebase-v1-BatchUpdateDatabasesRequest) | [BatchUpdateDatabasesResponse](#bytebase-v1-BatchUpdateDatabasesResponse) |  |
| GetDatabaseMetadata | [GetDatabaseMetadataRequest](#bytebase-v1-GetDatabaseMetadataRequest) | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  |
| GetDatabaseSchema | [GetDatabaseSchemaRequest](#bytebase-v1-GetDatabaseSchemaRequest) | [DatabaseSchema](#bytebase-v1-DatabaseSchema) |  |
| GetBackupSetting | [GetBackupSettingRequest](#bytebase-v1-GetBackupSettingRequest) | [BackupSetting](#bytebase-v1-BackupSetting) |  |
| UpdateBackupSetting | [UpdateBackupSettingRequest](#bytebase-v1-UpdateBackupSettingRequest) | [BackupSetting](#bytebase-v1-BackupSetting) |  |
| CreateBackup | [CreateBackupRequest](#bytebase-v1-CreateBackupRequest) | [Backup](#bytebase-v1-Backup) |  |
| ListBackup | [ListBackupRequest](#bytebase-v1-ListBackupRequest) | [ListBackupResponse](#bytebase-v1-ListBackupResponse) |  |

 



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



<a name="bytebase-v1-CreateRoleRequest"></a>

### CreateRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this role will be created. Format: environments/{environment}/instances/{instance} |
| role | [InstanceRole](#bytebase-v1-InstanceRole) |  | The role to create. |






<a name="bytebase-v1-DeleteRoleRequest"></a>

### DeleteRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the role to delete. Format: environments/{environment}/instances/{instance}/roles/{role name} |






<a name="bytebase-v1-GetRoleRequest"></a>

### GetRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the role to retrieve. Format: environments/{environment}/instances/{instance}/roles/{role name} The role name is the unique name for the role. |






<a name="bytebase-v1-InstanceRole"></a>

### InstanceRole
InstanceRole is the API message for instance role.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the role. Format: environments/{environment}/instances/{instance}/roles/{role name} The role name is the unique name for the role. |
| role_name | [string](#string) |  | The role name. It&#39;s unique within the instance. |
| password | [string](#string) | optional | The role password. |
| connection_limit | [int32](#int32) | optional | The connection count limit for this role. |
| valid_until | [string](#string) | optional | The expiration for the role&#39;s password. |
| attribute | [string](#string) | optional | The role attribute. For PostgreSQL, it containt super_user, no_inherit, create_role, create_db, can_login, replication and bypass_rls. Docs: https://www.postgresql.org/docs/current/role-attributes.html For MySQL, it&#39;s the global privileges as GRANT statements, which means it only contains &#34;GRANT ... ON *.* TO ...&#34;. Docs: https://dev.mysql.com/doc/refman/8.0/en/grant.html |






<a name="bytebase-v1-ListRolesRequest"></a>

### ListRolesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of roles. Format: environments/{environment}/instances/{instance} |
| page_size | [int32](#int32) |  | The maximum number of roles to return. The service may return fewer than this value. If unspecified, at most 50 roles will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListRoles` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListRoles` must match the call that provided the page token. |






<a name="bytebase-v1-ListRolesResponse"></a>

### ListRolesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| roles | [InstanceRole](#bytebase-v1-InstanceRole) | repeated | The roles from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-UndeleteRoleRequest"></a>

### UndeleteRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the deleted role. Format: environments/{environment}/instances/{instance}/roles/{role name} |






<a name="bytebase-v1-UpdateRoleRequest"></a>

### UpdateRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [InstanceRole](#bytebase-v1-InstanceRole) |  | The role to update.

The role&#39;s `name`, `environment` and `instance` field is used to identify the role to update. Format: environments/{environment}/instances/{instance}/roles/{role name} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 

 

 


<a name="bytebase-v1-InstanceRoleService"></a>

### InstanceRoleService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetRole | [GetRoleRequest](#bytebase-v1-GetRoleRequest) | [InstanceRole](#bytebase-v1-InstanceRole) |  |
| ListRoles | [ListRolesRequest](#bytebase-v1-ListRolesRequest) | [ListRolesResponse](#bytebase-v1-ListRolesResponse) |  |
| CreateRole | [CreateRoleRequest](#bytebase-v1-CreateRoleRequest) | [InstanceRole](#bytebase-v1-InstanceRole) |  |
| UpdateRole | [UpdateRoleRequest](#bytebase-v1-UpdateRoleRequest) | [InstanceRole](#bytebase-v1-InstanceRole) |  |
| DeleteRole | [DeleteRoleRequest](#bytebase-v1-DeleteRoleRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| UndeleteRole | [UndeleteRoleRequest](#bytebase-v1-UndeleteRoleRequest) | [InstanceRole](#bytebase-v1-InstanceRole) |  |

 



<a name="v1_instance_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/instance_service.proto



<a name="bytebase-v1-AddDataSourceRequest"></a>

### AddDataSourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance | [string](#string) |  | The name of the instance to add a data source to. Format: environments/{environment}/instances/{instance} |
| data_sources | [DataSource](#bytebase-v1-DataSource) |  | Identified by type. Only READ_ONLY data source can be added. |






<a name="bytebase-v1-CreateInstanceRequest"></a>

### CreateInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this instance will be created. Format: environments/{environment} |
| instance | [Instance](#bytebase-v1-Instance) |  | The instance to create. |
| instance_id | [string](#string) |  | The ID to use for the instance, which will become the final component of the instance&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |






<a name="bytebase-v1-DataSource"></a>

### DataSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  |  |
| type | [DataSourceType](#bytebase-v1-DataSourceType) |  |  |
| username | [string](#string) |  |  |
| password | [string](#string) |  |  |
| ssl_ca | [string](#string) |  |  |
| ssl_cert | [string](#string) |  |  |
| ssl_key | [string](#string) |  |  |
| host | [string](#string) |  |  |
| port | [string](#string) |  |  |
| database | [string](#string) |  |  |
| srv | [bool](#bool) |  |  |
| authentication_database | [string](#string) |  |  |






<a name="bytebase-v1-DeleteInstanceRequest"></a>

### DeleteInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the instance to delete. Format: environments/{environment}/instances/{instance} |






<a name="bytebase-v1-GetInstanceRequest"></a>

### GetInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the instance to retrieve. Format: environments/{environment}/instances/{instance} |






<a name="bytebase-v1-Instance"></a>

### Instance



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the instance. Format: environments/{environment}/instances/{instance} |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| state | [State](#bytebase-v1-State) |  |  |
| title | [string](#string) |  |  |
| engine | [Engine](#bytebase-v1-Engine) |  |  |
| external_link | [string](#string) |  |  |
| data_sources | [DataSource](#bytebase-v1-DataSource) | repeated |  |






<a name="bytebase-v1-ListInstancesRequest"></a>

### ListInstancesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of instances. Format: environments/{environment} Use &#34;environments/-&#34; to list all instances from all environments. |
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
| instance | [string](#string) |  | The name of the instance to remove a data source from. Format: environments/{environment}/instances/{instance} |
| data_sources | [DataSource](#bytebase-v1-DataSource) |  | Identified by type. Only READ_ONLY data source can be removed. |






<a name="bytebase-v1-UndeleteInstanceRequest"></a>

### UndeleteInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the deleted instance. Format: environments/{environment}/instances/{instance} |






<a name="bytebase-v1-UpdateDataSourceRequest"></a>

### UpdateDataSourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance | [string](#string) |  | The name of the instance to update a data source. Format: environments/{environment}/instances/{instance} |
| data_sources | [DataSource](#bytebase-v1-DataSource) |  | Identified by type. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






<a name="bytebase-v1-UpdateInstanceRequest"></a>

### UpdateInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance | [Instance](#bytebase-v1-Instance) |  | The instance to update.

The instance&#39;s `name` field is used to identify the instance to update. Format: environments/{environment}/instances/{instance} |
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
| AddDataSource | [AddDataSourceRequest](#bytebase-v1-AddDataSourceRequest) | [Instance](#bytebase-v1-Instance) |  |
| RemoveDataSource | [RemoveDataSourceRequest](#bytebase-v1-RemoveDataSourceRequest) | [Instance](#bytebase-v1-Instance) |  |
| UpdateDataSource | [UpdateDataSourceRequest](#bytebase-v1-UpdateDataSourceRequest) | [Instance](#bytebase-v1-Instance) |  |

 



<a name="v1_org_policy_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/org_policy_service.proto



<a name="bytebase-v1-AccessControlPolicy"></a>

### AccessControlPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| disallow_rules | [AccessControlRule](#bytebase-v1-AccessControlRule) | repeated |  |






<a name="bytebase-v1-AccessControlRule"></a>

### AccessControlRule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| full_database | [bool](#bool) |  |  |






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
| parent | [string](#string) |  | The parent resource where this instance will be created. Workspace resource name: &#34;&#34;. Environment resource name: environments/environment-id. Instance resource name: environments/environment-id/instances/instance-id. Database resource name: environments/environment-id/instances/instance-id/databases/database-name. |
| policy | [Policy](#bytebase-v1-Policy) |  | The policy to create. |
| type | [PolicyType](#bytebase-v1-PolicyType) |  |  |






<a name="bytebase-v1-DeletePolicyRequest"></a>

### DeletePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The policy&#39;s `name` field is used to identify the instance to update. Format: {resource name}/policies/{policy type} Workspace resource name: &#34;&#34;. Environment resource name: environments/environment-id. Instance resource name: environments/environment-id/instances/instance-id. Database resource name: environments/environment-id/instances/instance-id/databases/database-name. |






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
| page_size | [int32](#int32) |  | The maximum number of policies to return. The service may return fewer than this value. If unspecified, at most 50 policies will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `GetPolicies` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `GetPolicies` must match the call that provided the page token. |






<a name="bytebase-v1-ListPoliciesResponse"></a>

### ListPoliciesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policies | [Policy](#bytebase-v1-Policy) | repeated | The policies from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-Policy"></a>

### Policy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the policy. Format: {resource name}/policies/{policy type} Workspace resource name: &#34;&#34;. Environment resource name: environments/environment-id. Instance resource name: environments/environment-id/instances/instance-id. Database resource name: environments/environment-id/instances/instance-id/databases/database-name. |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| inherit_from_parent | [bool](#bool) |  |  |
| type | [PolicyType](#bytebase-v1-PolicyType) |  |  |
| deployment_approval_policy | [DeploymentApprovalPolicy](#bytebase-v1-DeploymentApprovalPolicy) |  |  |
| backup_plan_policy | [BackupPlanPolicy](#bytebase-v1-BackupPlanPolicy) |  |  |
| sensitive_data_policy | [SensitiveDataPolicy](#bytebase-v1-SensitiveDataPolicy) |  |  |
| access_control_policy | [AccessControlPolicy](#bytebase-v1-AccessControlPolicy) |  |  |
| sql_review_policy | [SQLReviewPolicy](#bytebase-v1-SQLReviewPolicy) |  |  |
| enforce | [bool](#bool) |  |  |






<a name="bytebase-v1-SQLReviewPolicy"></a>

### SQLReviewPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  |  |
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






<a name="bytebase-v1-SensitiveData"></a>

### SensitiveData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  |  |
| table | [string](#string) |  |  |
| column | [string](#string) |  |  |
| mask_type | [SensitiveDataMaskType](#bytebase-v1-SensitiveDataMaskType) |  |  |






<a name="bytebase-v1-SensitiveDataPolicy"></a>

### SensitiveDataPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sensitive_data | [SensitiveData](#bytebase-v1-SensitiveData) | repeated |  |






<a name="bytebase-v1-UpdatePolicyRequest"></a>

### UpdatePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [Policy](#bytebase-v1-Policy) |  | The policy to update.

The policy&#39;s `name` field is used to identify the instance to update. Format: {resource name}/policies/{policy type} Workspace resource name: &#34;&#34;. Environment resource name: environments/environment-id. Instance resource name: environments/environment-id/instances/instance-id. Database resource name: environments/environment-id/instances/instance-id/databases/database-name. |
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



<a name="bytebase-v1-PolicyType"></a>

### PolicyType


| Name | Number | Description |
| ---- | ------ | ----------- |
| POLICY_TYPE_UNSPECIFIED | 0 |  |
| DEPLOYMENT_APPROVAL | 1 |  |
| BACKUP_PLAN | 2 |  |
| SQL_REVIEW | 3 |  |
| SENSITIVE_DATA | 4 |  |
| ACCESS_CONTROL | 5 |  |



<a name="bytebase-v1-SQLReviewRuleLevel"></a>

### SQLReviewRuleLevel


| Name | Number | Description |
| ---- | ------ | ----------- |
| LEVEL_UNSPECIFIED | 0 |  |
| ERROR | 1 |  |
| WARNING | 2 |  |
| DISABLED | 3 |  |



<a name="bytebase-v1-SensitiveDataMaskType"></a>

### SensitiveDataMaskType


| Name | Number | Description |
| ---- | ------ | ----------- |
| MASK_TYPE_UNSPECIFIED | 0 |  |
| DEFAULT | 1 |  |


 

 


<a name="bytebase-v1-OrgPolicyService"></a>

### OrgPolicyService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetPolicy | [GetPolicyRequest](#bytebase-v1-GetPolicyRequest) | [Policy](#bytebase-v1-Policy) |  |
| ListPolicies | [ListPoliciesRequest](#bytebase-v1-ListPoliciesRequest) | [ListPoliciesResponse](#bytebase-v1-ListPoliciesResponse) |  |
| CreatePolicy | [CreatePolicyRequest](#bytebase-v1-CreatePolicyRequest) | [Policy](#bytebase-v1-Policy) |  |
| UpdatePolicy | [UpdatePolicyRequest](#bytebase-v1-UpdatePolicyRequest) | [Policy](#bytebase-v1-Policy) |  |
| DeletePolicy | [DeletePolicyRequest](#bytebase-v1-DeletePolicyRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



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






<a name="bytebase-v1-BatchUpdateReviewsRequest"></a>

### BatchUpdateReviewsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource shared by all reviews being updated. Format: projects/{project} If the operation spans parents, a dash (-) may be accepted as a wildcard. We only support updating the status of databases for now. |
| requests | [UpdateReviewRequest](#bytebase-v1-UpdateReviewRequest) | repeated | The request message specifying the resources to update. A maximum of 1000 databases can be modified in a batch. |






<a name="bytebase-v1-BatchUpdateReviewsResponse"></a>

### BatchUpdateReviewsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reviews | [Review](#bytebase-v1-Review) | repeated | Reviews updated. |






<a name="bytebase-v1-Binding"></a>

### Binding



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [ProjectRole](#bytebase-v1-ProjectRole) |  | The project role that is assigned to the members. |
| members | [string](#string) | repeated | Specifies the principals requesting access for a Bytebase resource. `members` can have the following values:

* `user:{emailid}`: An email address that represents a specific Bytebase account. For example, `alice@example.com` . |






<a name="bytebase-v1-CreateProjectRequest"></a>

### CreateProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [Project](#bytebase-v1-Project) |  | The project to create. |
| project_id | [string](#string) |  | The ID to use for the project, which will become the final component of the project&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |






<a name="bytebase-v1-DeleteProjectRequest"></a>

### DeleteProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the project to delete. Format: projects/{project} |






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






<a name="bytebase-v1-GetProjectRequest"></a>

### GetProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the project to retrieve. Format: projects/{project} |






<a name="bytebase-v1-GetReviewRequest"></a>

### GetReviewRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the review to retrieve. Format: projects/{project}/reviews/{review} |






<a name="bytebase-v1-IamPolicy"></a>

### IamPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bindings | [Binding](#bytebase-v1-Binding) | repeated | Collection of binding. A binding binds one or more project members to a single project role. |






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






<a name="bytebase-v1-ListReviewsRequest"></a>

### ListReviewsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of reviews. Format: projects/{project} Use &#34;projects/-&#34; to list all reviews from all projects. |
| page_size | [int32](#int32) |  | The maximum number of reviews to return. The service may return fewer than this value. If unspecified, at most 50 reviews will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListReviews` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListReviews` must match the call that provided the page token. |






<a name="bytebase-v1-ListReviewsResponse"></a>

### ListReviewsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| reviews | [Review](#bytebase-v1-Review) | repeated | The reviews from the specified request. |
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
| visibility | [Visibility](#bytebase-v1-Visibility) |  |  |
| tenant_mode | [TenantMode](#bytebase-v1-TenantMode) |  |  |
| db_name_template | [string](#string) |  |  |
| schema_version | [SchemaVersion](#bytebase-v1-SchemaVersion) |  |  |
| schema_change | [SchemaChange](#bytebase-v1-SchemaChange) |  |  |
| lgtm_check | [LgtmCheck](#bytebase-v1-LgtmCheck) |  |  |
| webhooks | [Webhook](#bytebase-v1-Webhook) | repeated |  |






<a name="bytebase-v1-RemoveWebhookRequest"></a>

### RemoveWebhookRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The name of the project to remove the webhook from. Format: projects/{project} |
| webhook | [Webhook](#bytebase-v1-Webhook) |  | The webhook to remove. Identified by its url. |






<a name="bytebase-v1-Review"></a>

### Review



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the review. `review` is a system generated ID. Format: projects/{project}/reviews/{review} |
| uid | [string](#string) |  | The system-assigned, unique identifier for a resource. |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| status | [ReviewStatus](#bytebase-v1-ReviewStatus) |  |  |
| assignee | [string](#string) |  | Format: user:hello@world.com |
| assignee_attention | [bool](#bool) |  |  |
| subscribers | [string](#string) | repeated | The subscribers. Format: user:hello@world.com |
| creator | [string](#string) |  | Format: user:hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






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






<a name="bytebase-v1-SetIamPolicyRequest"></a>

### SetIamPolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The name of the project to set the IAM policy. Format: projects/{project} |
| policy | [IamPolicy](#bytebase-v1-IamPolicy) |  |  |






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






<a name="bytebase-v1-UpdateDeploymentConfigRequest"></a>

### UpdateDeploymentConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| config | [DeploymentConfig](#bytebase-v1-DeploymentConfig) |  |  |






<a name="bytebase-v1-UpdateProjectRequest"></a>

### UpdateProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [Project](#bytebase-v1-Project) |  | The project to update.

The project&#39;s `name` field is used to identify the project to update. Format: projects/{project} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






<a name="bytebase-v1-UpdateReviewRequest"></a>

### UpdateReviewRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| review | [Review](#bytebase-v1-Review) |  | The review to update.

The review&#39;s `name` field is used to identify the review to update. Format: projects/{project}/reviews/{review} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






<a name="bytebase-v1-UpdateWebhookRequest"></a>

### UpdateWebhookRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The name of the project which owns the webhook to be updated. Format: projects/{project} |
| webhook | [Webhook](#bytebase-v1-Webhook) |  | The webhook to modify. Identified by its url. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






<a name="bytebase-v1-Webhook"></a>

### Webhook



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
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
| TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE | 5 | TYPE_ISSUE_PIPELINE_STAGE_STATUS_UPDATE represents the pipeline stage status change, including BEGIN, END for now. |
| TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE | 6 | TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE represents the pipeline task status change, including PENDING, PENDING_APPROVAL, RUNNING, SUCCESS, FAILURE, CANCELED for now. |
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
| TYPE_PROJECT_MEMBER_ROLE_UPDATE | 18 | TYPE_PROJECT_MEMBER_ROLE_UPDATE represents updating the member role, for example, from ADMIN to MEMBER. |
| TYPE_SQL_EDITOR_QUERY | 19 | SQL Editor related activity types. TYPE_SQL_EDITOR_QUERY represents executing query in SQL Editor. |
| TYPE_DATABASE_RECOVERY_PITR_DONE | 20 | Database related activity types. TYPE_DATABASE_RECOVERY_PITR_DONE represents the database recovery to a point in time is done. |



<a name="bytebase-v1-LgtmCheck"></a>

### LgtmCheck


| Name | Number | Description |
| ---- | ------ | ----------- |
| LGTM_CHECK_UNSPECIFIED | 0 |  |
| LGTM_CHECK_DISABLED | 1 |  |
| LGTM_CHECK_PROJECT_OWNER | 2 |  |
| LGTM_CHECK_PROJECT_MEMBER | 3 |  |



<a name="bytebase-v1-OperatorType"></a>

### OperatorType


| Name | Number | Description |
| ---- | ------ | ----------- |
| OPERATOR_TYPE_UNSPECIFIED | 0 | The operator is not specified. |
| OPERATOR_TYPE_IN | 1 | The operator is &#34;In&#34;. |
| OPERATOR_TYPE_EXISTS | 2 | The operator is &#34;Exists&#34;. |



<a name="bytebase-v1-ProjectRole"></a>

### ProjectRole


| Name | Number | Description |
| ---- | ------ | ----------- |
| PROJECT_ROLE_UNSPECIFIED | 0 |  |
| PROJECT_ROLE_OWNER | 1 |  |
| PROJECT_ROLE_DEVELOPER | 2 |  |



<a name="bytebase-v1-ReviewStatus"></a>

### ReviewStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| REVIEW_STATUS_UNSPECIFIED | 0 |  |
| OPEN | 1 |  |
| DONE | 2 |  |
| CANCELED | 3 |  |



<a name="bytebase-v1-SchemaChange"></a>

### SchemaChange


| Name | Number | Description |
| ---- | ------ | ----------- |
| SCHEMA_CHANGE_UNSPECIFIED | 0 |  |
| DDL | 1 |  |
| SDL | 2 |  |



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
| CreateProject | [CreateProjectRequest](#bytebase-v1-CreateProjectRequest) | [Project](#bytebase-v1-Project) |  |
| UpdateProject | [UpdateProjectRequest](#bytebase-v1-UpdateProjectRequest) | [Project](#bytebase-v1-Project) |  |
| DeleteProject | [DeleteProjectRequest](#bytebase-v1-DeleteProjectRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| UndeleteProject | [UndeleteProjectRequest](#bytebase-v1-UndeleteProjectRequest) | [Project](#bytebase-v1-Project) |  |
| GetIamPolicy | [GetIamPolicyRequest](#bytebase-v1-GetIamPolicyRequest) | [IamPolicy](#bytebase-v1-IamPolicy) |  |
| SetIamPolicy | [SetIamPolicyRequest](#bytebase-v1-SetIamPolicyRequest) | [IamPolicy](#bytebase-v1-IamPolicy) |  |
| GetReview | [GetReviewRequest](#bytebase-v1-GetReviewRequest) | [Review](#bytebase-v1-Review) |  |
| ListReviews | [ListReviewsRequest](#bytebase-v1-ListReviewsRequest) | [ListReviewsResponse](#bytebase-v1-ListReviewsResponse) |  |
| UpdateReview | [UpdateReviewRequest](#bytebase-v1-UpdateReviewRequest) | [Review](#bytebase-v1-Review) |  |
| BatchUpdateReviews | [BatchUpdateReviewsRequest](#bytebase-v1-BatchUpdateReviewsRequest) | [BatchUpdateReviewsResponse](#bytebase-v1-BatchUpdateReviewsResponse) |  |
| GetDeploymentConfig | [GetDeploymentConfigRequest](#bytebase-v1-GetDeploymentConfigRequest) | [DeploymentConfig](#bytebase-v1-DeploymentConfig) |  |
| UpdateDeploymentConfig | [UpdateDeploymentConfigRequest](#bytebase-v1-UpdateDeploymentConfigRequest) | [DeploymentConfig](#bytebase-v1-DeploymentConfig) |  |
| AddWebhook | [AddWebhookRequest](#bytebase-v1-AddWebhookRequest) | [Project](#bytebase-v1-Project) |  |
| UpdateWebhook | [UpdateWebhookRequest](#bytebase-v1-UpdateWebhookRequest) | [Project](#bytebase-v1-Project) |  |
| RemoveWebhook | [RemoveWebhookRequest](#bytebase-v1-RemoveWebhookRequest) | [Project](#bytebase-v1-Project) |  |
| TestWebhook | [TestWebhookRequest](#bytebase-v1-TestWebhookRequest) | [TestWebhookResponse](#bytebase-v1-TestWebhookResponse) |  |

 



<a name="v1_setting_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/setting_service.proto



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






<a name="bytebase-v1-SetSettingRequest"></a>

### SetSettingRequest
The request message for updating a setting.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| setting | [Setting](#bytebase-v1-Setting) |  | The setting to update. |






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





 

 

 


<a name="bytebase-v1-SettingService"></a>

### SettingService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetSetting | [GetSettingRequest](#bytebase-v1-GetSettingRequest) | [Setting](#bytebase-v1-Setting) |  |
| SetSetting | [SetSettingRequest](#bytebase-v1-SetSettingRequest) | [Setting](#bytebase-v1-Setting) |  |

 



<a name="v1_sql_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/sql_service.proto



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





 

 

 


<a name="bytebase-v1-SQLService"></a>

### SQLService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Pretty | [PrettyRequest](#bytebase-v1-PrettyRequest) | [PrettyResponse](#bytebase-v1-PrettyResponse) |  |

 



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

