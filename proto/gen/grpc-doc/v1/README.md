# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [v1/actuator_service.proto](#v1_actuator_service-proto)
    - [ActuatorInfo](#bytebase-v1-ActuatorInfo)
    - [DeleteCacheRequest](#bytebase-v1-DeleteCacheRequest)
    - [GetActuatorInfoRequest](#bytebase-v1-GetActuatorInfoRequest)
  
    - [ActuatorService](#bytebase-v1-ActuatorService)
  
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
  
- [v1/common.proto](#v1_common-proto)
    - [Engine](#bytebase-v1-Engine)
    - [State](#bytebase-v1-State)
  
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
    - [AdviseIndexRequest](#bytebase-v1-AdviseIndexRequest)
    - [AdviseIndexResponse](#bytebase-v1-AdviseIndexResponse)
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
    - [DeleteSecretRequest](#bytebase-v1-DeleteSecretRequest)
    - [DependentColumn](#bytebase-v1-DependentColumn)
    - [ExtensionMetadata](#bytebase-v1-ExtensionMetadata)
    - [ForeignKeyMetadata](#bytebase-v1-ForeignKeyMetadata)
    - [FunctionMetadata](#bytebase-v1-FunctionMetadata)
    - [GetBackupSettingRequest](#bytebase-v1-GetBackupSettingRequest)
    - [GetDatabaseMetadataRequest](#bytebase-v1-GetDatabaseMetadataRequest)
    - [GetDatabaseRequest](#bytebase-v1-GetDatabaseRequest)
    - [GetDatabaseSchemaRequest](#bytebase-v1-GetDatabaseSchemaRequest)
    - [IndexMetadata](#bytebase-v1-IndexMetadata)
    - [ListBackupRequest](#bytebase-v1-ListBackupRequest)
    - [ListBackupResponse](#bytebase-v1-ListBackupResponse)
    - [ListDatabasesRequest](#bytebase-v1-ListDatabasesRequest)
    - [ListDatabasesResponse](#bytebase-v1-ListDatabasesResponse)
    - [ListSecretsRequest](#bytebase-v1-ListSecretsRequest)
    - [ListSecretsResponse](#bytebase-v1-ListSecretsResponse)
    - [ListSlowQueriesRequest](#bytebase-v1-ListSlowQueriesRequest)
    - [ListSlowQueriesResponse](#bytebase-v1-ListSlowQueriesResponse)
    - [SchemaMetadata](#bytebase-v1-SchemaMetadata)
    - [Secret](#bytebase-v1-Secret)
    - [SlowQueryDetails](#bytebase-v1-SlowQueryDetails)
    - [SlowQueryLog](#bytebase-v1-SlowQueryLog)
    - [SlowQueryStatistics](#bytebase-v1-SlowQueryStatistics)
    - [TableMetadata](#bytebase-v1-TableMetadata)
    - [UpdateBackupSettingRequest](#bytebase-v1-UpdateBackupSettingRequest)
    - [UpdateDatabaseRequest](#bytebase-v1-UpdateDatabaseRequest)
    - [UpdateSecretRequest](#bytebase-v1-UpdateSecretRequest)
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
  
- [v1/externalvs_service.proto](#v1_externalvs_service-proto)
    - [CreateExternalVersionControlRequest](#bytebase-v1-CreateExternalVersionControlRequest)
    - [DeleteExternalVersionControlRequest](#bytebase-v1-DeleteExternalVersionControlRequest)
    - [ExternalVersionControl](#bytebase-v1-ExternalVersionControl)
    - [GetExternalVersionControlRequest](#bytebase-v1-GetExternalVersionControlRequest)
    - [ListExternalVersionControlsRequest](#bytebase-v1-ListExternalVersionControlsRequest)
    - [ListExternalVersionControlsResponse](#bytebase-v1-ListExternalVersionControlsResponse)
    - [ListProjectGitOpsInfoRequest](#bytebase-v1-ListProjectGitOpsInfoRequest)
    - [ListProjectGitOpsInfoResponse](#bytebase-v1-ListProjectGitOpsInfoResponse)
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
    - [ListInstancesRequest](#bytebase-v1-ListInstancesRequest)
    - [ListInstancesResponse](#bytebase-v1-ListInstancesResponse)
    - [RemoveDataSourceRequest](#bytebase-v1-RemoveDataSourceRequest)
    - [SyncSlowQueriesRequest](#bytebase-v1-SyncSlowQueriesRequest)
    - [UndeleteInstanceRequest](#bytebase-v1-UndeleteInstanceRequest)
    - [UpdateDataSourceRequest](#bytebase-v1-UpdateDataSourceRequest)
    - [UpdateInstanceRequest](#bytebase-v1-UpdateInstanceRequest)
  
    - [DataSourceType](#bytebase-v1-DataSourceType)
  
    - [InstanceService](#bytebase-v1-InstanceService)
  
- [v1/logging_service.proto](#v1_logging_service-proto)
    - [ListLogsRequest](#bytebase-v1-ListLogsRequest)
    - [ListLogsResponse](#bytebase-v1-ListLogsResponse)
    - [LogEntry](#bytebase-v1-LogEntry)
  
    - [LogEntry.Action](#bytebase-v1-LogEntry-Action)
    - [LogEntry.Level](#bytebase-v1-LogEntry-Level)
  
    - [LoggingService](#bytebase-v1-LoggingService)
  
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
    - [PolicyResourceType](#bytebase-v1-PolicyResourceType)
    - [PolicyType](#bytebase-v1-PolicyType)
    - [SQLReviewRuleLevel](#bytebase-v1-SQLReviewRuleLevel)
    - [SensitiveDataMaskType](#bytebase-v1-SensitiveDataMaskType)
  
    - [OrgPolicyService](#bytebase-v1-OrgPolicyService)
  
- [v1/project_service.proto](#v1_project_service-proto)
    - [Activity](#bytebase-v1-Activity)
    - [AddWebhookRequest](#bytebase-v1-AddWebhookRequest)
    - [Binding](#bytebase-v1-Binding)
    - [CreateProjectRequest](#bytebase-v1-CreateProjectRequest)
    - [DeleteProjectRequest](#bytebase-v1-DeleteProjectRequest)
    - [DeploymentConfig](#bytebase-v1-DeploymentConfig)
    - [DeploymentSpec](#bytebase-v1-DeploymentSpec)
    - [GetDeploymentConfigRequest](#bytebase-v1-GetDeploymentConfigRequest)
    - [GetIamPolicyRequest](#bytebase-v1-GetIamPolicyRequest)
    - [GetProjectGitOpsInfoRequest](#bytebase-v1-GetProjectGitOpsInfoRequest)
    - [GetProjectRequest](#bytebase-v1-GetProjectRequest)
    - [IamPolicy](#bytebase-v1-IamPolicy)
    - [LabelSelector](#bytebase-v1-LabelSelector)
    - [LabelSelectorRequirement](#bytebase-v1-LabelSelectorRequirement)
    - [ListProjectsRequest](#bytebase-v1-ListProjectsRequest)
    - [ListProjectsResponse](#bytebase-v1-ListProjectsResponse)
    - [Project](#bytebase-v1-Project)
    - [RemoveWebhookRequest](#bytebase-v1-RemoveWebhookRequest)
    - [Schedule](#bytebase-v1-Schedule)
    - [ScheduleDeployment](#bytebase-v1-ScheduleDeployment)
    - [SetIamPolicyRequest](#bytebase-v1-SetIamPolicyRequest)
    - [SetProjectGitOpsInfoRequest](#bytebase-v1-SetProjectGitOpsInfoRequest)
    - [TestWebhookRequest](#bytebase-v1-TestWebhookRequest)
    - [TestWebhookResponse](#bytebase-v1-TestWebhookResponse)
    - [UndeleteProjectRequest](#bytebase-v1-UndeleteProjectRequest)
    - [UpdateDeploymentConfigRequest](#bytebase-v1-UpdateDeploymentConfigRequest)
    - [UpdateProjectRequest](#bytebase-v1-UpdateProjectRequest)
    - [UpdateWebhookRequest](#bytebase-v1-UpdateWebhookRequest)
    - [Webhook](#bytebase-v1-Webhook)
  
    - [Activity.Type](#bytebase-v1-Activity-Type)
    - [OperatorType](#bytebase-v1-OperatorType)
    - [SchemaChange](#bytebase-v1-SchemaChange)
    - [SchemaVersion](#bytebase-v1-SchemaVersion)
    - [TenantMode](#bytebase-v1-TenantMode)
    - [Visibility](#bytebase-v1-Visibility)
    - [Webhook.Type](#bytebase-v1-Webhook-Type)
    - [Workflow](#bytebase-v1-Workflow)
  
    - [ProjectService](#bytebase-v1-ProjectService)
  
- [v1/review_service.proto](#v1_review_service-proto)
    - [ApprovalFlow](#bytebase-v1-ApprovalFlow)
    - [ApprovalNode](#bytebase-v1-ApprovalNode)
    - [ApprovalStep](#bytebase-v1-ApprovalStep)
    - [ApprovalTemplate](#bytebase-v1-ApprovalTemplate)
    - [ApproveReviewRequest](#bytebase-v1-ApproveReviewRequest)
    - [BatchUpdateReviewsRequest](#bytebase-v1-BatchUpdateReviewsRequest)
    - [BatchUpdateReviewsResponse](#bytebase-v1-BatchUpdateReviewsResponse)
    - [GetReviewRequest](#bytebase-v1-GetReviewRequest)
    - [ListReviewsRequest](#bytebase-v1-ListReviewsRequest)
    - [ListReviewsResponse](#bytebase-v1-ListReviewsResponse)
    - [Review](#bytebase-v1-Review)
    - [Review.Approver](#bytebase-v1-Review-Approver)
    - [UpdateReviewRequest](#bytebase-v1-UpdateReviewRequest)
  
    - [ApprovalNode.GroupValue](#bytebase-v1-ApprovalNode-GroupValue)
    - [ApprovalNode.Type](#bytebase-v1-ApprovalNode-Type)
    - [ApprovalStep.Type](#bytebase-v1-ApprovalStep-Type)
    - [Review.Approver.Status](#bytebase-v1-Review-Approver-Status)
    - [ReviewStatus](#bytebase-v1-ReviewStatus)
  
    - [ReviewService](#bytebase-v1-ReviewService)
  
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
  
- [v1/setting_service.proto](#v1_setting_service-proto)
    - [GetSettingRequest](#bytebase-v1-GetSettingRequest)
    - [GetSettingResponse](#bytebase-v1-GetSettingResponse)
    - [SMTPMailDeliverySettingValue](#bytebase-v1-SMTPMailDeliverySettingValue)
    - [SetSettingRequest](#bytebase-v1-SetSettingRequest)
    - [Setting](#bytebase-v1-Setting)
    - [Value](#bytebase-v1-Value)
  
    - [SMTPMailDeliverySettingValue.Authentication](#bytebase-v1-SMTPMailDeliverySettingValue-Authentication)
    - [SMTPMailDeliverySettingValue.Encryption](#bytebase-v1-SMTPMailDeliverySettingValue-Encryption)
  
    - [SettingService](#bytebase-v1-SettingService)
  
- [v1/sql_service.proto](#v1_sql_service-proto)
    - [PrettyRequest](#bytebase-v1-PrettyRequest)
    - [PrettyResponse](#bytebase-v1-PrettyResponse)
  
    - [SQLService](#bytebase-v1-SQLService)
  
- [v1/subscription_service.proto](#v1_subscription_service-proto)
    - [GetSubscriptionRequest](#bytebase-v1-GetSubscriptionRequest)
    - [PatchSubscription](#bytebase-v1-PatchSubscription)
    - [Subscription](#bytebase-v1-Subscription)
    - [TrialSubscription](#bytebase-v1-TrialSubscription)
    - [TrialSubscriptionRequest](#bytebase-v1-TrialSubscriptionRequest)
    - [UpdateSubscriptionRequest](#bytebase-v1-UpdateSubscriptionRequest)
  
    - [PlanType](#bytebase-v1-PlanType)
  
    - [SubscriptionService](#bytebase-v1-SubscriptionService)
  
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






<a name="bytebase-v1-DeleteCacheRequest"></a>

### DeleteCacheRequest







<a name="bytebase-v1-GetActuatorInfoRequest"></a>

### GetActuatorInfoRequest






 

 

 


<a name="bytebase-v1-ActuatorService"></a>

### ActuatorService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetActuatorInfo | [GetActuatorInfoRequest](#bytebase-v1-GetActuatorInfoRequest) | [ActuatorInfo](#bytebase-v1-ActuatorInfo) |  |
| DeleteCache | [DeleteCacheRequest](#bytebase-v1-DeleteCacheRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



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



<a name="bytebase-v1-State"></a>

### State


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATE_UNSPECIFIED | 0 |  |
| ACTIVE | 1 |  |
| DELETED | 2 |  |


 

 

 



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
| regenerate_temp_mfa_secret | [bool](#bool) |  | The regenerate_temp_mfa_secret flag means to regenerate tempary MFA secret for user. This is used for MFA setup. The tempary MFA secret and recovery codes will be returned in the response. |
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
| mfa_secret | [string](#string) |  | The mfa_secret is the tempary secret using in two phase verification. |
| recovery_codes | [string](#string) | repeated | The recovery_codes is the tempary recovery codes using in two phase verification. |





 


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



<a name="bytebase-v1-AdviseIndexRequest"></a>

### AdviseIndexRequest
AdviseIndexRequest is the request of advising index.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: environments/{environment}/instances/{instance}/databases/{database} |
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






<a name="bytebase-v1-DeleteSecretRequest"></a>

### DeleteSecretRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the secret to be deleted. Format: environments/{environment}/instances/{instance}/databases/{database}/secrets/{secret} |






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






<a name="bytebase-v1-ListSecretsRequest"></a>

### ListSecretsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent of the secret. Format: environments/{environment}/instances/{instance}/databases/{database} |
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
| parent | [string](#string) |  | Format: environments/{environment}/instances/{instance}/databases/{database} |
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






<a name="bytebase-v1-Secret"></a>

### Secret
Secret is the secret of the database now.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | name is the unique name of the secret, which is specified by the client. Format: environments/{environment}/instances/{instance}/databases/{database}/secrets/{secret} |
| created_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Not used. The timestamp when the secret resource was created initally. |
| updated_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Not used. The timestamp when the secret resource was updated. |
| value | [string](#string) |  | The value of the secret. |
| description | [string](#string) |  | The decsription of the secret. |






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
| resource | [string](#string) |  | The resource of the slow query log. The format is &#34;environments/{environment}/instances/{instance}/databases/{database}&#34;. |
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
| ListSlowQueries | [ListSlowQueriesRequest](#bytebase-v1-ListSlowQueriesRequest) | [ListSlowQueriesResponse](#bytebase-v1-ListSlowQueriesResponse) |  |
| ListSecrets | [ListSecretsRequest](#bytebase-v1-ListSecretsRequest) | [ListSecretsResponse](#bytebase-v1-ListSecretsResponse) |  |
| UpdateSecret | [UpdateSecretRequest](#bytebase-v1-UpdateSecretRequest) | [Secret](#bytebase-v1-Secret) |  |
| DeleteSecret | [DeleteSecretRequest](#bytebase-v1-DeleteSecretRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| AdviseIndex | [AdviseIndexRequest](#bytebase-v1-AdviseIndexRequest) | [AdviseIndexResponse](#bytebase-v1-AdviseIndexResponse) |  |

 



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






<a name="bytebase-v1-ProjectGitOpsInfo"></a>

### ProjectGitOpsInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the linked repository, generated by the server, and also is the unique identifier. Format: projects/{project}/gitOpsInfo |
| title | [string](#string) |  | The title of the repository. For axample: sample. |
| full_path | [string](#string) |  | The full_path of the repository. For example: bytebase/sample. |
| web_url | [string](#string) |  | The web url of the repository. For axample: https://gitlab.bytebase.com/bytebase/sample. |
| branch_filter | [string](#string) |  | The branch Bytebase listens to for changes. For example: main. |
| base_directory | [string](#string) |  | The root directory where Bytebase observes the file change. If empty, then it observes the entire repository. |
| file_path_template | [string](#string) |  | Bytebase only observes the file path name matching the template pattern **relative** to the base directory. Required Placeholder: {{DB_NAME}}, {{VERSION}}, {{TYPE]}. Optional Placeholder: {{ENV_ID}}, {{DESCRIPTION}}. Optional Directory Wildcard: &#39;*&#39;, &#39;**&#39;. |
| schema_path_template | [string](#string) |  | The file path template for storing the latest schema auto-generated by Bytebase after migration. If empty, then Bytebase won&#39;t auto generate it. If specified, required placeholder: {{DB_NAME}}, optional placeholder: {{ENV_ID}}. |
| sheet_path_template | [string](#string) |  | The file path template for matching the sql files for sheet. If specified, required Placeholder: {{NAME}}, optional Placeholder: {{ENV_ID}}, {{DB_NAME}}. |
| enable_sql_review_ci | [bool](#bool) |  | Set to true to enable SQL review CI for all PR/MRs. |
| webhook_endpoint_id | [string](#string) |  | The webhook endpoint ID of the repository. |






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
| TYPE_GITHUB | 1 | GitHub type. Using for GitHub community edition(ce). |
| TYPE_GITLAB | 2 | GitLab type. Using for GitLab community edition(ce) and enterprise edition(ee). |
| TYPE_BITBUCKET | 3 | BitBucket type. Using for BitBucket cloud or BitBucket server. |


 

 


<a name="bytebase-v1-ExternalVersionControlService"></a>

### ExternalVersionControlService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetExternalVersionControl | [GetExternalVersionControlRequest](#bytebase-v1-GetExternalVersionControlRequest) | [ExternalVersionControl](#bytebase-v1-ExternalVersionControl) |  |
| ListExternalVersionControls | [ListExternalVersionControlsRequest](#bytebase-v1-ListExternalVersionControlsRequest) | [ListExternalVersionControlsResponse](#bytebase-v1-ListExternalVersionControlsResponse) |  |
| CreateExternalVersionControl | [CreateExternalVersionControlRequest](#bytebase-v1-CreateExternalVersionControlRequest) | [ExternalVersionControl](#bytebase-v1-ExternalVersionControl) |  |
| UpdateExternalVersionControl | [UpdateExternalVersionControlRequest](#bytebase-v1-UpdateExternalVersionControlRequest) | [ExternalVersionControl](#bytebase-v1-ExternalVersionControl) |  |
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
| skip_tls_verify | [bool](#bool) |  |  |






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



<a name="bytebase-v1-CreateInstanceRoleRequest"></a>

### CreateInstanceRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this role will be created. Format: environments/{environment}/instances/{instance} |
| role | [InstanceRole](#bytebase-v1-InstanceRole) |  | The role to create. |






<a name="bytebase-v1-DeleteInstanceRoleRequest"></a>

### DeleteInstanceRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the role to delete. Format: environments/{environment}/instances/{instance}/roles/{role name} |






<a name="bytebase-v1-GetInstanceRoleRequest"></a>

### GetInstanceRoleRequest



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






<a name="bytebase-v1-ListInstanceRolesRequest"></a>

### ListInstanceRolesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of roles. Format: environments/{environment}/instances/{instance} |
| page_size | [int32](#int32) |  | The maximum number of roles to return. The service may return fewer than this value. If unspecified, at most 50 roles will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListRoles` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListRoles` must match the call that provided the page token. |






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
| name | [string](#string) |  | The name of the deleted role. Format: environments/{environment}/instances/{instance}/roles/{role name} |






<a name="bytebase-v1-UpdateInstanceRoleRequest"></a>

### UpdateInstanceRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [InstanceRole](#bytebase-v1-InstanceRole) |  | The role to update.

The role&#39;s `name`, `environment` and `instance` field is used to identify the role to update. Format: environments/{environment}/instances/{instance}/roles/{role name} |
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
| name | [string](#string) |  | The name of the instance to delete. Format: environments/{environment}/instances/{instance} |
| force | [bool](#bool) |  | If set to true, any databases and sheets from this project will also be moved to default project, and all open issues will be closed. |






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






<a name="bytebase-v1-SyncSlowQueriesRequest"></a>

### SyncSlowQueriesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance | [string](#string) |  | The name of the instance to sync slow queries. Format: environments/{environment}/instances/{instance} |






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
| SyncSlowQueries | [SyncSlowQueriesRequest](#bytebase-v1-SyncSlowQueriesRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="v1_logging_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/logging_service.proto



<a name="bytebase-v1-ListLogsRequest"></a>

### ListLogsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource name. Format: projects/{project} workspaces/{workspace} |
| filter | [string](#string) |  | filter is the filter to apply on the list logs request, follow the [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) syntax. The field only support in filter: - creator - container - level - action For example: List the logs of type &#39;ACTION_ISSUE_COMMENT_CREATE&#39; in issue/123: &#39;action=&#34;ACTION_ISSUE_COMMENT_CREATE&#34;, container=&#34;issue/123&#34;&#39; |
| page_size | [int32](#int32) |  | Not used. The maximum number of logs to return. The service may return fewer than this value. If unspecified, at most 100 log entries will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListLogs` call. Provide this to retrieve the subsequent page. |






<a name="bytebase-v1-ListLogsResponse"></a>

### ListLogsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| log_entries | [LogEntry](#bytebase-v1-LogEntry) | repeated | The list of log entries. |
| next_page_token | [string](#string) |  | A token to retrieve next page of log entries. Pass this value in the page_token field in the subsequent call to `ListLogs` method to retrieve the next page of log entries. |






<a name="bytebase-v1-LogEntry"></a>

### LogEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| creator | [string](#string) |  | The creator of the log entry. Format: user:{emailid} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the backup resource was created initally. |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the backup resource was updated. |
| action | [LogEntry.Action](#bytebase-v1-LogEntry-Action) |  |  |
| level | [LogEntry.Level](#bytebase-v1-LogEntry-Level) |  |  |
| resource_name | [string](#string) |  | The name of the resource associated with this log entry. For example, the resource user associated with log entry type of &#34;ACTION_MEMBER_CREATE&#34;. Format: For ACTION_MEMBER_*: user:emailid For ACTION_ISSUE_*: issues/{issue} For ACTION_PIPELINE_*: pipelines/{pipeline} For ACTION_PROJECT_*: projects/{project} For ACTION_SQL_EDITOR_QUERY: workspaces/{workspace} OR projects/{project} |
| json_payload | [google.protobuf.Struct](#google-protobuf-Struct) |  | The payload of the log entry. |





 


<a name="bytebase-v1-LogEntry-Action"></a>

### LogEntry.Action


| Name | Number | Description |
| ---- | ------ | ----------- |
| ACTION_UNSPECIFIED | 0 |  |
| ACTION_MEMBER_CREATE | 1 | In worksapce resource only.

ACTION_MEMBER_CREATE is the type for creating a new member. |
| ACTION_MEMBER_ROLE_UPDATE | 2 | ACTION_MEMBER_ROLE_UPDATE is the type for updating a member&#39;s role. |
| ACTION_MEMBER_ACTIVATE | 3 | ACTION_MEMBER_ACTIVATE_UPDATE is the type for activating members. |
| ACTION_MEMBER_DEACTIVE | 4 | ACTION_MEMBER_DEACTIVE is the type for deactiving members. |
| ACTION_ISSUE_CREATE | 5 | In project resource only.

ACTION_ISSUE_CREATE is the type for creating a new issue. |
| ACTION_ISSUE_COMMENT_CREATE | 6 | ACTION_ISSUE_COMMENT_CREATE is the type for creating a new comment on an issue. |
| ACTION_ISSUE_FIELD_UPDATE | 7 | ACTION_ISSUE_FIELD_UPDATE is the type for updating an issue&#39;s field. |
| ACTION_ISSUE_STATUS_UPDATE | 8 | ACTION_ISSUE_STATUS_UPDATE is the type for updating an issue&#39;s status. |
| ACTION_PIPELINE_STAGE_STATUS_UPDATE | 9 | ACTION_PIPELINE_STAGE_STATUS_UPDATE is the type for stage begins or ends. |
| ACTION_PIPELINE_TASK_STATUS_UPDATE | 10 | ACTION_PIPELINE_TASK_STATUS_UPDATE is the type for updating pipeline task status. |
| ACTION_PIPELINE_TASK_FILE_COMMIT | 11 | ACTION_PIPELINE_TASK_FILE_COMMIT is the type for committing pipeline task files. |
| ACTION_PIPELINE_TASK_STATEMENT_UPDATE | 12 | ACTION_PIPELINE_TASK_STATEMENT_UPDATE is the type for updating pipeline task SQL statement. |
| ACITON_PIPELINE_TASK_EARLIEST_ALLOWED_DATE_UPDATE | 13 | ACTION_PIPELINE_TASK_EARLIEST_ALLOWED_DATE_UPDATE is the type for updating pipeline task the earliest allowed time. |
| ACTION_PROJECT_MEMBER_CREATE | 14 | ACTION_PROJECT_MEMBER_CREATE is the type for creating a new project member. |
| ACTION_PROJECT_MEMBER_ROLE_UPDATE | 15 | ACTION_PROJECT_MEMBER_ROLE_UPDATE is the type for updating a project member&#39;s role. |
| ACTION_PROJECT_MEMBER_DELETE | 16 | ACTION_PROJECT_MEMBER_DELETE is the type for deleting a project member. |
| ACTION_PROJECT_REPOSITORY_PUSH | 17 | ACTION_PROJECT_REPOSITORY_PUSH is the type for pushing to a project repository. |
| ACTION_PROJECT_DTABASE_TRANSFER | 18 | ACTION_PROJECT_DATABASE_TRANSFER is the type for transferring a database to a project. |
| ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE | 19 | ACTION_PROJECT_DATABASE_RECOVERY_PITR_DONE is the type for database PITR recovery done. |
| ACTION_SQL_EDITOR_QUERY | 20 | Both in workspace and project resource.

ACTION_SQL_EDITOR_QUERY is the type for SQL editor query. If user runs SQL in Read-only mode, this action will belong to project resource. If user runs SQL in Read-write mode, this action will belong to workspace resource. |



<a name="bytebase-v1-LogEntry-Level"></a>

### LogEntry.Level


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
| resource_type | [PolicyResourceType](#bytebase-v1-PolicyResourceType) |  | The resource type for the policy. |
| resource_uid | [string](#string) |  | The system-assigned, unique identifier for the resource. |
| state | [State](#bytebase-v1-State) |  |  |






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






<a name="bytebase-v1-Binding"></a>

### Binding



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [string](#string) |  | The project role that is assigned to the members. Format: roles/{role} |
| members | [string](#string) | repeated | Specifies the principals requesting access for a Bytebase resource. `members` can have the following values:

* `user:{emailid}`: An email address that represents a specific Bytebase account. For example, `alice@example.com` . |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this binding. If the condition evaluates to true, then this binding applies to the current request. If the condition evaluates to false, then this binding does not apply to the current request. However, a different role binding might grant the same role to one or more of the principals in this binding. |






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
| force | [bool](#bool) |  | If set to true, any databases and sheets from this project will also be moved to default project, and all open issues will be closed. |






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






<a name="bytebase-v1-GetProjectGitOpsInfoRequest"></a>

### GetProjectGitOpsInfoRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The name of the project. Format: projects/{project} |






<a name="bytebase-v1-GetProjectRequest"></a>

### GetProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the project to retrieve. Format: projects/{project} |






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
| webhooks | [Webhook](#bytebase-v1-Webhook) | repeated |  |






<a name="bytebase-v1-RemoveWebhookRequest"></a>

### RemoveWebhookRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The name of the project to remove the webhook from. Format: projects/{project} |
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






<a name="bytebase-v1-SetIamPolicyRequest"></a>

### SetIamPolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The name of the project to set the IAM policy. Format: projects/{project} |
| policy | [IamPolicy](#bytebase-v1-IamPolicy) |  |  |






<a name="bytebase-v1-SetProjectGitOpsInfoRequest"></a>

### SetProjectGitOpsInfoRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The name of the project. Format: projects/{project} |
| project_gitops_info | [ProjectGitOpsInfo](#bytebase-v1-ProjectGitOpsInfo) |  | The binding for the project and external version control. |






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
| GetDeploymentConfig | [GetDeploymentConfigRequest](#bytebase-v1-GetDeploymentConfigRequest) | [DeploymentConfig](#bytebase-v1-DeploymentConfig) |  |
| UpdateDeploymentConfig | [UpdateDeploymentConfigRequest](#bytebase-v1-UpdateDeploymentConfigRequest) | [DeploymentConfig](#bytebase-v1-DeploymentConfig) |  |
| AddWebhook | [AddWebhookRequest](#bytebase-v1-AddWebhookRequest) | [Project](#bytebase-v1-Project) |  |
| UpdateWebhook | [UpdateWebhookRequest](#bytebase-v1-UpdateWebhookRequest) | [Project](#bytebase-v1-Project) |  |
| RemoveWebhook | [RemoveWebhookRequest](#bytebase-v1-RemoveWebhookRequest) | [Project](#bytebase-v1-Project) |  |
| TestWebhook | [TestWebhookRequest](#bytebase-v1-TestWebhookRequest) | [TestWebhookResponse](#bytebase-v1-TestWebhookResponse) |  |
| SetProjectGitOpsInfo | [SetProjectGitOpsInfoRequest](#bytebase-v1-SetProjectGitOpsInfoRequest) | [ProjectGitOpsInfo](#bytebase-v1-ProjectGitOpsInfo) |  |
| GetProjectGitOpsInfo | [SetProjectGitOpsInfoRequest](#bytebase-v1-SetProjectGitOpsInfoRequest) | [ProjectGitOpsInfo](#bytebase-v1-ProjectGitOpsInfo) |  |

 



<a name="v1_review_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/review_service.proto



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






<a name="bytebase-v1-ApproveReviewRequest"></a>

### ApproveReviewRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the review to add an approver. Format: projects/{project}/reviews/{review} |






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






<a name="bytebase-v1-GetReviewRequest"></a>

### GetReviewRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the review to retrieve. Format: projects/{project}/reviews/{review} |






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
| approval_templates | [ApprovalTemplate](#bytebase-v1-ApprovalTemplate) | repeated |  |
| approvers | [Review.Approver](#bytebase-v1-Review-Approver) | repeated |  |
| approval_finding_done | [bool](#bool) |  | If the value is `false`, it means that the backend is still finding matching approval templates. If `true`, approval_templates &amp; approvers &amp; approval_finding_error are available. |
| approval_finding_error | [string](#string) |  |  |
| subscribers | [string](#string) | repeated | The subscribers. Format: user:hello@world.com |
| creator | [string](#string) |  | Format: user:hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="bytebase-v1-Review-Approver"></a>

### Review.Approver



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [Review.Approver.Status](#bytebase-v1-Review-Approver-Status) |  | The new status. |
| principal | [string](#string) |  | Format: user:hello@world.com |






<a name="bytebase-v1-UpdateReviewRequest"></a>

### UpdateReviewRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| review | [Review](#bytebase-v1-Review) |  | The review to update.

The review&#39;s `name` field is used to identify the review to update. Format: projects/{project}/reviews/{review} |
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



<a name="bytebase-v1-Review-Approver-Status"></a>

### Review.Approver.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| APPROVED | 2 |  |



<a name="bytebase-v1-ReviewStatus"></a>

### ReviewStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| REVIEW_STATUS_UNSPECIFIED | 0 |  |
| OPEN | 1 |  |
| DONE | 2 |  |
| CANCELED | 3 |  |


 

 


<a name="bytebase-v1-ReviewService"></a>

### ReviewService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetReview | [GetReviewRequest](#bytebase-v1-GetReviewRequest) | [Review](#bytebase-v1-Review) |  |
| ListReviews | [ListReviewsRequest](#bytebase-v1-ListReviewsRequest) | [ListReviewsResponse](#bytebase-v1-ListReviewsResponse) |  |
| UpdateReview | [UpdateReviewRequest](#bytebase-v1-UpdateReviewRequest) | [Review](#bytebase-v1-Review) |  |
| BatchUpdateReviews | [BatchUpdateReviewsRequest](#bytebase-v1-BatchUpdateReviewsRequest) | [BatchUpdateReviewsResponse](#bytebase-v1-BatchUpdateReviewsResponse) |  |
| ApproveReview | [ApproveReviewRequest](#bytebase-v1-ApproveReviewRequest) | [Review](#bytebase-v1-Review) |  |

 



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
| expression | [google.api.expr.v1alpha1.ParsedExpr](#google-api-expr-v1alpha1-ParsedExpr) |  |  |
| active | [bool](#bool) |  |  |






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

 



<a name="v1_subscription_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/subscription_service.proto



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
| UpdateSubscription | [UpdateSubscriptionRequest](#bytebase-v1-UpdateSubscriptionRequest) | [Subscription](#bytebase-v1-Subscription) |  |
| TrialSubscription | [TrialSubscriptionRequest](#bytebase-v1-TrialSubscriptionRequest) | [Subscription](#bytebase-v1-Subscription) |  |

 



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

