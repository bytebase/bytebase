# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [api/v1alpha/annotation.proto](#api_v1alpha_annotation-proto)
    - [AuthMethod](#bytebase-api-v1alpha-AuthMethod)
  
    - [File-level Extensions](#api_v1alpha_annotation-proto-extensions)
    - [File-level Extensions](#api_v1alpha_annotation-proto-extensions)
    - [File-level Extensions](#api_v1alpha_annotation-proto-extensions)
    - [File-level Extensions](#api_v1alpha_annotation-proto-extensions)
  
- [api/v1alpha/common.proto](#api_v1alpha_common-proto)
    - [Position](#bytebase-api-v1alpha-Position)
    - [Range](#bytebase-api-v1alpha-Range)
  
    - [Engine](#bytebase-api-v1alpha-Engine)
    - [ExportFormat](#bytebase-api-v1alpha-ExportFormat)
    - [State](#bytebase-api-v1alpha-State)
    - [VCSType](#bytebase-api-v1alpha-VCSType)
  
- [api/v1alpha/database_catalog_service.proto](#api_v1alpha_database_catalog_service-proto)
    - [ColumnCatalog](#bytebase-api-v1alpha-ColumnCatalog)
    - [ColumnCatalog.LabelsEntry](#bytebase-api-v1alpha-ColumnCatalog-LabelsEntry)
    - [DatabaseCatalog](#bytebase-api-v1alpha-DatabaseCatalog)
    - [GetDatabaseCatalogRequest](#bytebase-api-v1alpha-GetDatabaseCatalogRequest)
    - [ObjectSchema](#bytebase-api-v1alpha-ObjectSchema)
    - [ObjectSchema.ArrayKind](#bytebase-api-v1alpha-ObjectSchema-ArrayKind)
    - [ObjectSchema.StructKind](#bytebase-api-v1alpha-ObjectSchema-StructKind)
    - [ObjectSchema.StructKind.PropertiesEntry](#bytebase-api-v1alpha-ObjectSchema-StructKind-PropertiesEntry)
    - [SchemaCatalog](#bytebase-api-v1alpha-SchemaCatalog)
    - [TableCatalog](#bytebase-api-v1alpha-TableCatalog)
    - [TableCatalog.Columns](#bytebase-api-v1alpha-TableCatalog-Columns)
    - [UpdateDatabaseCatalogRequest](#bytebase-api-v1alpha-UpdateDatabaseCatalogRequest)
  
    - [ObjectSchema.Type](#bytebase-api-v1alpha-ObjectSchema-Type)
  
    - [DatabaseCatalogService](#bytebase-api-v1alpha-DatabaseCatalogService)
  
- [api/v1alpha/instance_role_service.proto](#api_v1alpha_instance_role_service-proto)
    - [GetInstanceRoleRequest](#bytebase-api-v1alpha-GetInstanceRoleRequest)
    - [InstanceRole](#bytebase-api-v1alpha-InstanceRole)
    - [ListInstanceRolesRequest](#bytebase-api-v1alpha-ListInstanceRolesRequest)
    - [ListInstanceRolesResponse](#bytebase-api-v1alpha-ListInstanceRolesResponse)
  
    - [InstanceRoleService](#bytebase-api-v1alpha-InstanceRoleService)
  
- [api/v1alpha/instance_service.proto](#api_v1alpha_instance_service-proto)
    - [AddDataSourceRequest](#bytebase-api-v1alpha-AddDataSourceRequest)
    - [BatchSyncInstancesRequest](#bytebase-api-v1alpha-BatchSyncInstancesRequest)
    - [BatchSyncInstancesResponse](#bytebase-api-v1alpha-BatchSyncInstancesResponse)
    - [CreateInstanceRequest](#bytebase-api-v1alpha-CreateInstanceRequest)
    - [DataSource](#bytebase-api-v1alpha-DataSource)
    - [DataSource.Address](#bytebase-api-v1alpha-DataSource-Address)
    - [DataSource.ClientSecretCredential](#bytebase-api-v1alpha-DataSource-ClientSecretCredential)
    - [DataSource.ExtraConnectionParametersEntry](#bytebase-api-v1alpha-DataSource-ExtraConnectionParametersEntry)
    - [DataSourceExternalSecret](#bytebase-api-v1alpha-DataSourceExternalSecret)
    - [DataSourceExternalSecret.AppRoleAuthOption](#bytebase-api-v1alpha-DataSourceExternalSecret-AppRoleAuthOption)
    - [DeleteInstanceRequest](#bytebase-api-v1alpha-DeleteInstanceRequest)
    - [GetInstanceRequest](#bytebase-api-v1alpha-GetInstanceRequest)
    - [Instance](#bytebase-api-v1alpha-Instance)
    - [InstanceResource](#bytebase-api-v1alpha-InstanceResource)
    - [KerberosConfig](#bytebase-api-v1alpha-KerberosConfig)
    - [ListInstanceDatabaseRequest](#bytebase-api-v1alpha-ListInstanceDatabaseRequest)
    - [ListInstanceDatabaseResponse](#bytebase-api-v1alpha-ListInstanceDatabaseResponse)
    - [ListInstancesRequest](#bytebase-api-v1alpha-ListInstancesRequest)
    - [ListInstancesResponse](#bytebase-api-v1alpha-ListInstancesResponse)
    - [RemoveDataSourceRequest](#bytebase-api-v1alpha-RemoveDataSourceRequest)
    - [SASLConfig](#bytebase-api-v1alpha-SASLConfig)
    - [SyncInstanceRequest](#bytebase-api-v1alpha-SyncInstanceRequest)
    - [SyncInstanceResponse](#bytebase-api-v1alpha-SyncInstanceResponse)
    - [UndeleteInstanceRequest](#bytebase-api-v1alpha-UndeleteInstanceRequest)
    - [UpdateDataSourceRequest](#bytebase-api-v1alpha-UpdateDataSourceRequest)
    - [UpdateInstanceRequest](#bytebase-api-v1alpha-UpdateInstanceRequest)
  
    - [DataSource.AuthenticationType](#bytebase-api-v1alpha-DataSource-AuthenticationType)
    - [DataSource.RedisType](#bytebase-api-v1alpha-DataSource-RedisType)
    - [DataSourceExternalSecret.AppRoleAuthOption.SecretType](#bytebase-api-v1alpha-DataSourceExternalSecret-AppRoleAuthOption-SecretType)
    - [DataSourceExternalSecret.AuthType](#bytebase-api-v1alpha-DataSourceExternalSecret-AuthType)
    - [DataSourceExternalSecret.SecretType](#bytebase-api-v1alpha-DataSourceExternalSecret-SecretType)
    - [DataSourceType](#bytebase-api-v1alpha-DataSourceType)
  
    - [InstanceService](#bytebase-api-v1alpha-InstanceService)
  
- [api/v1alpha/database_service.proto](#api_v1alpha_database_service-proto)
    - [BatchGetDatabasesRequest](#bytebase-api-v1alpha-BatchGetDatabasesRequest)
    - [BatchGetDatabasesResponse](#bytebase-api-v1alpha-BatchGetDatabasesResponse)
    - [BatchSyncDatabasesRequest](#bytebase-api-v1alpha-BatchSyncDatabasesRequest)
    - [BatchSyncDatabasesResponse](#bytebase-api-v1alpha-BatchSyncDatabasesResponse)
    - [BatchUpdateDatabasesRequest](#bytebase-api-v1alpha-BatchUpdateDatabasesRequest)
    - [BatchUpdateDatabasesResponse](#bytebase-api-v1alpha-BatchUpdateDatabasesResponse)
    - [ChangedResourceDatabase](#bytebase-api-v1alpha-ChangedResourceDatabase)
    - [ChangedResourceFunction](#bytebase-api-v1alpha-ChangedResourceFunction)
    - [ChangedResourceProcedure](#bytebase-api-v1alpha-ChangedResourceProcedure)
    - [ChangedResourceSchema](#bytebase-api-v1alpha-ChangedResourceSchema)
    - [ChangedResourceTable](#bytebase-api-v1alpha-ChangedResourceTable)
    - [ChangedResourceView](#bytebase-api-v1alpha-ChangedResourceView)
    - [ChangedResources](#bytebase-api-v1alpha-ChangedResources)
    - [Changelog](#bytebase-api-v1alpha-Changelog)
    - [CheckConstraintMetadata](#bytebase-api-v1alpha-CheckConstraintMetadata)
    - [ColumnMetadata](#bytebase-api-v1alpha-ColumnMetadata)
    - [CreateRevisionRequest](#bytebase-api-v1alpha-CreateRevisionRequest)
    - [Database](#bytebase-api-v1alpha-Database)
    - [Database.LabelsEntry](#bytebase-api-v1alpha-Database-LabelsEntry)
    - [DatabaseMetadata](#bytebase-api-v1alpha-DatabaseMetadata)
    - [DatabaseSchema](#bytebase-api-v1alpha-DatabaseSchema)
    - [DeleteRevisionRequest](#bytebase-api-v1alpha-DeleteRevisionRequest)
    - [DeleteSecretRequest](#bytebase-api-v1alpha-DeleteSecretRequest)
    - [DependencyColumn](#bytebase-api-v1alpha-DependencyColumn)
    - [DependencyTable](#bytebase-api-v1alpha-DependencyTable)
    - [DiffSchemaRequest](#bytebase-api-v1alpha-DiffSchemaRequest)
    - [DiffSchemaResponse](#bytebase-api-v1alpha-DiffSchemaResponse)
    - [EnumTypeMetadata](#bytebase-api-v1alpha-EnumTypeMetadata)
    - [EventMetadata](#bytebase-api-v1alpha-EventMetadata)
    - [ExtensionMetadata](#bytebase-api-v1alpha-ExtensionMetadata)
    - [ExternalTableMetadata](#bytebase-api-v1alpha-ExternalTableMetadata)
    - [ForeignKeyMetadata](#bytebase-api-v1alpha-ForeignKeyMetadata)
    - [FunctionMetadata](#bytebase-api-v1alpha-FunctionMetadata)
    - [GenerationMetadata](#bytebase-api-v1alpha-GenerationMetadata)
    - [GetChangelogRequest](#bytebase-api-v1alpha-GetChangelogRequest)
    - [GetDatabaseMetadataRequest](#bytebase-api-v1alpha-GetDatabaseMetadataRequest)
    - [GetDatabaseRequest](#bytebase-api-v1alpha-GetDatabaseRequest)
    - [GetDatabaseSchemaRequest](#bytebase-api-v1alpha-GetDatabaseSchemaRequest)
    - [GetRevisionRequest](#bytebase-api-v1alpha-GetRevisionRequest)
    - [GetSchemaStringRequest](#bytebase-api-v1alpha-GetSchemaStringRequest)
    - [GetSchemaStringResponse](#bytebase-api-v1alpha-GetSchemaStringResponse)
    - [IndexMetadata](#bytebase-api-v1alpha-IndexMetadata)
    - [ListChangelogsRequest](#bytebase-api-v1alpha-ListChangelogsRequest)
    - [ListChangelogsResponse](#bytebase-api-v1alpha-ListChangelogsResponse)
    - [ListDatabasesRequest](#bytebase-api-v1alpha-ListDatabasesRequest)
    - [ListDatabasesResponse](#bytebase-api-v1alpha-ListDatabasesResponse)
    - [ListRevisionsRequest](#bytebase-api-v1alpha-ListRevisionsRequest)
    - [ListRevisionsResponse](#bytebase-api-v1alpha-ListRevisionsResponse)
    - [ListSecretsRequest](#bytebase-api-v1alpha-ListSecretsRequest)
    - [ListSecretsResponse](#bytebase-api-v1alpha-ListSecretsResponse)
    - [MaterializedViewMetadata](#bytebase-api-v1alpha-MaterializedViewMetadata)
    - [PackageMetadata](#bytebase-api-v1alpha-PackageMetadata)
    - [ProcedureMetadata](#bytebase-api-v1alpha-ProcedureMetadata)
    - [Revision](#bytebase-api-v1alpha-Revision)
    - [SchemaMetadata](#bytebase-api-v1alpha-SchemaMetadata)
    - [Secret](#bytebase-api-v1alpha-Secret)
    - [SequenceMetadata](#bytebase-api-v1alpha-SequenceMetadata)
    - [StreamMetadata](#bytebase-api-v1alpha-StreamMetadata)
    - [SyncDatabaseRequest](#bytebase-api-v1alpha-SyncDatabaseRequest)
    - [SyncDatabaseResponse](#bytebase-api-v1alpha-SyncDatabaseResponse)
    - [TableMetadata](#bytebase-api-v1alpha-TableMetadata)
    - [TablePartitionMetadata](#bytebase-api-v1alpha-TablePartitionMetadata)
    - [TaskMetadata](#bytebase-api-v1alpha-TaskMetadata)
    - [TriggerMetadata](#bytebase-api-v1alpha-TriggerMetadata)
    - [UpdateDatabaseRequest](#bytebase-api-v1alpha-UpdateDatabaseRequest)
    - [UpdateSecretRequest](#bytebase-api-v1alpha-UpdateSecretRequest)
    - [ViewMetadata](#bytebase-api-v1alpha-ViewMetadata)
  
    - [Changelog.Status](#bytebase-api-v1alpha-Changelog-Status)
    - [Changelog.Type](#bytebase-api-v1alpha-Changelog-Type)
    - [ChangelogView](#bytebase-api-v1alpha-ChangelogView)
    - [ColumnMetadata.IdentityGeneration](#bytebase-api-v1alpha-ColumnMetadata-IdentityGeneration)
    - [GenerationMetadata.Type](#bytebase-api-v1alpha-GenerationMetadata-Type)
    - [GetSchemaStringRequest.ObjectType](#bytebase-api-v1alpha-GetSchemaStringRequest-ObjectType)
    - [StreamMetadata.Mode](#bytebase-api-v1alpha-StreamMetadata-Mode)
    - [StreamMetadata.Type](#bytebase-api-v1alpha-StreamMetadata-Type)
    - [TablePartitionMetadata.Type](#bytebase-api-v1alpha-TablePartitionMetadata-Type)
    - [TaskMetadata.State](#bytebase-api-v1alpha-TaskMetadata-State)
  
    - [DatabaseService](#bytebase-api-v1alpha-DatabaseService)
  
- [api/v1alpha/issue_service.proto](#api_v1alpha_issue_service-proto)
    - [ApprovalFlow](#bytebase-api-v1alpha-ApprovalFlow)
    - [ApprovalNode](#bytebase-api-v1alpha-ApprovalNode)
    - [ApprovalStep](#bytebase-api-v1alpha-ApprovalStep)
    - [ApprovalTemplate](#bytebase-api-v1alpha-ApprovalTemplate)
    - [ApproveIssueRequest](#bytebase-api-v1alpha-ApproveIssueRequest)
    - [BatchUpdateIssuesStatusRequest](#bytebase-api-v1alpha-BatchUpdateIssuesStatusRequest)
    - [BatchUpdateIssuesStatusResponse](#bytebase-api-v1alpha-BatchUpdateIssuesStatusResponse)
    - [CreateIssueCommentRequest](#bytebase-api-v1alpha-CreateIssueCommentRequest)
    - [CreateIssueRequest](#bytebase-api-v1alpha-CreateIssueRequest)
    - [GetIssueRequest](#bytebase-api-v1alpha-GetIssueRequest)
    - [GrantRequest](#bytebase-api-v1alpha-GrantRequest)
    - [Issue](#bytebase-api-v1alpha-Issue)
    - [Issue.Approver](#bytebase-api-v1alpha-Issue-Approver)
    - [Issue.TaskStatusCountEntry](#bytebase-api-v1alpha-Issue-TaskStatusCountEntry)
    - [IssueComment](#bytebase-api-v1alpha-IssueComment)
    - [IssueComment.Approval](#bytebase-api-v1alpha-IssueComment-Approval)
    - [IssueComment.IssueUpdate](#bytebase-api-v1alpha-IssueComment-IssueUpdate)
    - [IssueComment.StageEnd](#bytebase-api-v1alpha-IssueComment-StageEnd)
    - [IssueComment.TaskPriorBackup](#bytebase-api-v1alpha-IssueComment-TaskPriorBackup)
    - [IssueComment.TaskPriorBackup.Table](#bytebase-api-v1alpha-IssueComment-TaskPriorBackup-Table)
    - [IssueComment.TaskUpdate](#bytebase-api-v1alpha-IssueComment-TaskUpdate)
    - [ListIssueCommentsRequest](#bytebase-api-v1alpha-ListIssueCommentsRequest)
    - [ListIssueCommentsResponse](#bytebase-api-v1alpha-ListIssueCommentsResponse)
    - [ListIssuesRequest](#bytebase-api-v1alpha-ListIssuesRequest)
    - [ListIssuesResponse](#bytebase-api-v1alpha-ListIssuesResponse)
    - [RejectIssueRequest](#bytebase-api-v1alpha-RejectIssueRequest)
    - [RequestIssueRequest](#bytebase-api-v1alpha-RequestIssueRequest)
    - [SearchIssuesRequest](#bytebase-api-v1alpha-SearchIssuesRequest)
    - [SearchIssuesResponse](#bytebase-api-v1alpha-SearchIssuesResponse)
    - [UpdateIssueCommentRequest](#bytebase-api-v1alpha-UpdateIssueCommentRequest)
    - [UpdateIssueRequest](#bytebase-api-v1alpha-UpdateIssueRequest)
  
    - [ApprovalNode.Type](#bytebase-api-v1alpha-ApprovalNode-Type)
    - [ApprovalStep.Type](#bytebase-api-v1alpha-ApprovalStep-Type)
    - [Issue.Approver.Status](#bytebase-api-v1alpha-Issue-Approver-Status)
    - [Issue.RiskLevel](#bytebase-api-v1alpha-Issue-RiskLevel)
    - [Issue.Type](#bytebase-api-v1alpha-Issue-Type)
    - [IssueComment.Approval.Status](#bytebase-api-v1alpha-IssueComment-Approval-Status)
    - [IssueComment.TaskUpdate.Status](#bytebase-api-v1alpha-IssueComment-TaskUpdate-Status)
    - [IssueStatus](#bytebase-api-v1alpha-IssueStatus)
  
    - [IssueService](#bytebase-api-v1alpha-IssueService)
  
- [api/v1alpha/subscription_service.proto](#api_v1alpha_subscription_service-proto)
    - [Feature](#bytebase-api-v1alpha-Feature)
    - [Feature.MatrixEntry](#bytebase-api-v1alpha-Feature-MatrixEntry)
    - [FeatureMatrix](#bytebase-api-v1alpha-FeatureMatrix)
    - [GetFeatureMatrixRequest](#bytebase-api-v1alpha-GetFeatureMatrixRequest)
    - [GetSubscriptionRequest](#bytebase-api-v1alpha-GetSubscriptionRequest)
    - [PatchSubscription](#bytebase-api-v1alpha-PatchSubscription)
    - [Subscription](#bytebase-api-v1alpha-Subscription)
    - [UpdateSubscriptionRequest](#bytebase-api-v1alpha-UpdateSubscriptionRequest)
  
    - [PlanType](#bytebase-api-v1alpha-PlanType)
  
    - [SubscriptionService](#bytebase-api-v1alpha-SubscriptionService)
  
- [api/v1alpha/setting_service.proto](#api_v1alpha_setting_service-proto)
    - [AISetting](#bytebase-api-v1alpha-AISetting)
    - [AgentPluginSetting](#bytebase-api-v1alpha-AgentPluginSetting)
    - [Algorithm](#bytebase-api-v1alpha-Algorithm)
    - [Algorithm.FullMask](#bytebase-api-v1alpha-Algorithm-FullMask)
    - [Algorithm.InnerOuterMask](#bytebase-api-v1alpha-Algorithm-InnerOuterMask)
    - [Algorithm.MD5Mask](#bytebase-api-v1alpha-Algorithm-MD5Mask)
    - [Algorithm.RangeMask](#bytebase-api-v1alpha-Algorithm-RangeMask)
    - [Algorithm.RangeMask.Slice](#bytebase-api-v1alpha-Algorithm-RangeMask-Slice)
    - [Announcement](#bytebase-api-v1alpha-Announcement)
    - [AppIMSetting](#bytebase-api-v1alpha-AppIMSetting)
    - [AppIMSetting.DingTalk](#bytebase-api-v1alpha-AppIMSetting-DingTalk)
    - [AppIMSetting.Feishu](#bytebase-api-v1alpha-AppIMSetting-Feishu)
    - [AppIMSetting.Lark](#bytebase-api-v1alpha-AppIMSetting-Lark)
    - [AppIMSetting.Slack](#bytebase-api-v1alpha-AppIMSetting-Slack)
    - [AppIMSetting.Wecom](#bytebase-api-v1alpha-AppIMSetting-Wecom)
    - [DataClassificationSetting](#bytebase-api-v1alpha-DataClassificationSetting)
    - [DataClassificationSetting.DataClassificationConfig](#bytebase-api-v1alpha-DataClassificationSetting-DataClassificationConfig)
    - [DataClassificationSetting.DataClassificationConfig.ClassificationEntry](#bytebase-api-v1alpha-DataClassificationSetting-DataClassificationConfig-ClassificationEntry)
    - [DataClassificationSetting.DataClassificationConfig.DataClassification](#bytebase-api-v1alpha-DataClassificationSetting-DataClassificationConfig-DataClassification)
    - [DataClassificationSetting.DataClassificationConfig.Level](#bytebase-api-v1alpha-DataClassificationSetting-DataClassificationConfig-Level)
    - [EnvironmentSetting](#bytebase-api-v1alpha-EnvironmentSetting)
    - [EnvironmentSetting.Environment](#bytebase-api-v1alpha-EnvironmentSetting-Environment)
    - [EnvironmentSetting.Environment.TagsEntry](#bytebase-api-v1alpha-EnvironmentSetting-Environment-TagsEntry)
    - [GetSettingRequest](#bytebase-api-v1alpha-GetSettingRequest)
    - [GetSettingResponse](#bytebase-api-v1alpha-GetSettingResponse)
    - [ListSettingsRequest](#bytebase-api-v1alpha-ListSettingsRequest)
    - [ListSettingsResponse](#bytebase-api-v1alpha-ListSettingsResponse)
    - [MaximumSQLResultSizeSetting](#bytebase-api-v1alpha-MaximumSQLResultSizeSetting)
    - [PasswordRestrictionSetting](#bytebase-api-v1alpha-PasswordRestrictionSetting)
    - [SCIMSetting](#bytebase-api-v1alpha-SCIMSetting)
    - [SMTPMailDeliverySettingValue](#bytebase-api-v1alpha-SMTPMailDeliverySettingValue)
    - [SchemaTemplateSetting](#bytebase-api-v1alpha-SchemaTemplateSetting)
    - [SchemaTemplateSetting.ColumnType](#bytebase-api-v1alpha-SchemaTemplateSetting-ColumnType)
    - [SchemaTemplateSetting.FieldTemplate](#bytebase-api-v1alpha-SchemaTemplateSetting-FieldTemplate)
    - [SchemaTemplateSetting.TableTemplate](#bytebase-api-v1alpha-SchemaTemplateSetting-TableTemplate)
    - [SemanticTypeSetting](#bytebase-api-v1alpha-SemanticTypeSetting)
    - [SemanticTypeSetting.SemanticType](#bytebase-api-v1alpha-SemanticTypeSetting-SemanticType)
    - [Setting](#bytebase-api-v1alpha-Setting)
    - [UpdateSettingRequest](#bytebase-api-v1alpha-UpdateSettingRequest)
    - [Value](#bytebase-api-v1alpha-Value)
    - [WorkspaceApprovalSetting](#bytebase-api-v1alpha-WorkspaceApprovalSetting)
    - [WorkspaceApprovalSetting.Rule](#bytebase-api-v1alpha-WorkspaceApprovalSetting-Rule)
    - [WorkspaceProfileSetting](#bytebase-api-v1alpha-WorkspaceProfileSetting)
    - [WorkspaceTrialSetting](#bytebase-api-v1alpha-WorkspaceTrialSetting)
  
    - [AISetting.Provider](#bytebase-api-v1alpha-AISetting-Provider)
    - [Algorithm.InnerOuterMask.MaskType](#bytebase-api-v1alpha-Algorithm-InnerOuterMask-MaskType)
    - [Announcement.AlertLevel](#bytebase-api-v1alpha-Announcement-AlertLevel)
    - [DatabaseChangeMode](#bytebase-api-v1alpha-DatabaseChangeMode)
    - [SMTPMailDeliverySettingValue.Authentication](#bytebase-api-v1alpha-SMTPMailDeliverySettingValue-Authentication)
    - [SMTPMailDeliverySettingValue.Encryption](#bytebase-api-v1alpha-SMTPMailDeliverySettingValue-Encryption)
  
    - [SettingService](#bytebase-api-v1alpha-SettingService)
  
- [api/v1alpha/user_service.proto](#api_v1alpha_user_service-proto)
    - [CreateUserRequest](#bytebase-api-v1alpha-CreateUserRequest)
    - [DeleteUserRequest](#bytebase-api-v1alpha-DeleteUserRequest)
    - [GetUserRequest](#bytebase-api-v1alpha-GetUserRequest)
    - [ListUsersRequest](#bytebase-api-v1alpha-ListUsersRequest)
    - [ListUsersResponse](#bytebase-api-v1alpha-ListUsersResponse)
    - [UndeleteUserRequest](#bytebase-api-v1alpha-UndeleteUserRequest)
    - [UpdateUserRequest](#bytebase-api-v1alpha-UpdateUserRequest)
    - [User](#bytebase-api-v1alpha-User)
    - [User.Profile](#bytebase-api-v1alpha-User-Profile)
  
    - [UserType](#bytebase-api-v1alpha-UserType)
  
    - [UserService](#bytebase-api-v1alpha-UserService)
  
- [api/v1alpha/actuator_service.proto](#api_v1alpha_actuator_service-proto)
    - [ActuatorInfo](#bytebase-api-v1alpha-ActuatorInfo)
    - [ActuatorInfo.StatUser](#bytebase-api-v1alpha-ActuatorInfo-StatUser)
    - [DeleteCacheRequest](#bytebase-api-v1alpha-DeleteCacheRequest)
    - [GetActuatorInfoRequest](#bytebase-api-v1alpha-GetActuatorInfoRequest)
    - [GetResourcePackageRequest](#bytebase-api-v1alpha-GetResourcePackageRequest)
    - [ResourcePackage](#bytebase-api-v1alpha-ResourcePackage)
    - [UpdateActuatorInfoRequest](#bytebase-api-v1alpha-UpdateActuatorInfoRequest)
  
    - [ActuatorService](#bytebase-api-v1alpha-ActuatorService)
  
- [api/v1alpha/iam_policy.proto](#api_v1alpha_iam_policy-proto)
    - [Binding](#bytebase-api-v1alpha-Binding)
    - [BindingDelta](#bytebase-api-v1alpha-BindingDelta)
    - [GetIamPolicyRequest](#bytebase-api-v1alpha-GetIamPolicyRequest)
    - [IamPolicy](#bytebase-api-v1alpha-IamPolicy)
    - [PolicyDelta](#bytebase-api-v1alpha-PolicyDelta)
    - [SetIamPolicyRequest](#bytebase-api-v1alpha-SetIamPolicyRequest)
  
    - [BindingDelta.Action](#bytebase-api-v1alpha-BindingDelta-Action)
  
- [api/v1alpha/audit_log_service.proto](#api_v1alpha_audit_log_service-proto)
    - [AuditData](#bytebase-api-v1alpha-AuditData)
    - [AuditLog](#bytebase-api-v1alpha-AuditLog)
    - [ExportAuditLogsRequest](#bytebase-api-v1alpha-ExportAuditLogsRequest)
    - [ExportAuditLogsResponse](#bytebase-api-v1alpha-ExportAuditLogsResponse)
    - [RequestMetadata](#bytebase-api-v1alpha-RequestMetadata)
    - [SearchAuditLogsRequest](#bytebase-api-v1alpha-SearchAuditLogsRequest)
    - [SearchAuditLogsResponse](#bytebase-api-v1alpha-SearchAuditLogsResponse)
  
    - [AuditLog.Severity](#bytebase-api-v1alpha-AuditLog-Severity)
  
    - [AuditLogService](#bytebase-api-v1alpha-AuditLogService)
  
- [api/v1alpha/auth_service.proto](#api_v1alpha_auth_service-proto)
    - [IdentityProviderContext](#bytebase-api-v1alpha-IdentityProviderContext)
    - [LoginRequest](#bytebase-api-v1alpha-LoginRequest)
    - [LoginResponse](#bytebase-api-v1alpha-LoginResponse)
    - [LogoutRequest](#bytebase-api-v1alpha-LogoutRequest)
    - [OAuth2IdentityProviderContext](#bytebase-api-v1alpha-OAuth2IdentityProviderContext)
    - [OIDCIdentityProviderContext](#bytebase-api-v1alpha-OIDCIdentityProviderContext)
  
    - [AuthService](#bytebase-api-v1alpha-AuthService)
  
- [api/v1alpha/cel_service.proto](#api_v1alpha_cel_service-proto)
    - [BatchDeparseRequest](#bytebase-api-v1alpha-BatchDeparseRequest)
    - [BatchDeparseResponse](#bytebase-api-v1alpha-BatchDeparseResponse)
    - [BatchParseRequest](#bytebase-api-v1alpha-BatchParseRequest)
    - [BatchParseResponse](#bytebase-api-v1alpha-BatchParseResponse)
  
    - [CelService](#bytebase-api-v1alpha-CelService)
  
- [api/v1alpha/changelist_service.proto](#api_v1alpha_changelist_service-proto)
    - [Changelist](#bytebase-api-v1alpha-Changelist)
    - [Changelist.Change](#bytebase-api-v1alpha-Changelist-Change)
    - [CreateChangelistRequest](#bytebase-api-v1alpha-CreateChangelistRequest)
    - [DeleteChangelistRequest](#bytebase-api-v1alpha-DeleteChangelistRequest)
    - [GetChangelistRequest](#bytebase-api-v1alpha-GetChangelistRequest)
    - [ListChangelistsRequest](#bytebase-api-v1alpha-ListChangelistsRequest)
    - [ListChangelistsResponse](#bytebase-api-v1alpha-ListChangelistsResponse)
    - [UpdateChangelistRequest](#bytebase-api-v1alpha-UpdateChangelistRequest)
  
    - [ChangelistService](#bytebase-api-v1alpha-ChangelistService)
  
- [api/v1alpha/database_group_service.proto](#api_v1alpha_database_group_service-proto)
    - [CreateDatabaseGroupRequest](#bytebase-api-v1alpha-CreateDatabaseGroupRequest)
    - [DatabaseGroup](#bytebase-api-v1alpha-DatabaseGroup)
    - [DatabaseGroup.Database](#bytebase-api-v1alpha-DatabaseGroup-Database)
    - [DeleteDatabaseGroupRequest](#bytebase-api-v1alpha-DeleteDatabaseGroupRequest)
    - [GetDatabaseGroupRequest](#bytebase-api-v1alpha-GetDatabaseGroupRequest)
    - [ListDatabaseGroupsRequest](#bytebase-api-v1alpha-ListDatabaseGroupsRequest)
    - [ListDatabaseGroupsResponse](#bytebase-api-v1alpha-ListDatabaseGroupsResponse)
    - [UpdateDatabaseGroupRequest](#bytebase-api-v1alpha-UpdateDatabaseGroupRequest)
  
    - [DatabaseGroupView](#bytebase-api-v1alpha-DatabaseGroupView)
  
    - [DatabaseGroupService](#bytebase-api-v1alpha-DatabaseGroupService)
  
- [api/v1alpha/group_service.proto](#api_v1alpha_group_service-proto)
    - [CreateGroupRequest](#bytebase-api-v1alpha-CreateGroupRequest)
    - [DeleteGroupRequest](#bytebase-api-v1alpha-DeleteGroupRequest)
    - [GetGroupRequest](#bytebase-api-v1alpha-GetGroupRequest)
    - [Group](#bytebase-api-v1alpha-Group)
    - [GroupMember](#bytebase-api-v1alpha-GroupMember)
    - [ListGroupsRequest](#bytebase-api-v1alpha-ListGroupsRequest)
    - [ListGroupsResponse](#bytebase-api-v1alpha-ListGroupsResponse)
    - [UpdateGroupRequest](#bytebase-api-v1alpha-UpdateGroupRequest)
  
    - [GroupMember.Role](#bytebase-api-v1alpha-GroupMember-Role)
  
    - [GroupService](#bytebase-api-v1alpha-GroupService)
  
- [api/v1alpha/idp_service.proto](#api_v1alpha_idp_service-proto)
    - [CreateIdentityProviderRequest](#bytebase-api-v1alpha-CreateIdentityProviderRequest)
    - [DeleteIdentityProviderRequest](#bytebase-api-v1alpha-DeleteIdentityProviderRequest)
    - [FieldMapping](#bytebase-api-v1alpha-FieldMapping)
    - [GetIdentityProviderRequest](#bytebase-api-v1alpha-GetIdentityProviderRequest)
    - [IdentityProvider](#bytebase-api-v1alpha-IdentityProvider)
    - [IdentityProviderConfig](#bytebase-api-v1alpha-IdentityProviderConfig)
    - [LDAPIdentityProviderConfig](#bytebase-api-v1alpha-LDAPIdentityProviderConfig)
    - [ListIdentityProvidersRequest](#bytebase-api-v1alpha-ListIdentityProvidersRequest)
    - [ListIdentityProvidersResponse](#bytebase-api-v1alpha-ListIdentityProvidersResponse)
    - [OAuth2IdentityProviderConfig](#bytebase-api-v1alpha-OAuth2IdentityProviderConfig)
    - [OAuth2IdentityProviderTestRequestContext](#bytebase-api-v1alpha-OAuth2IdentityProviderTestRequestContext)
    - [OIDCIdentityProviderConfig](#bytebase-api-v1alpha-OIDCIdentityProviderConfig)
    - [TestIdentityProviderRequest](#bytebase-api-v1alpha-TestIdentityProviderRequest)
    - [TestIdentityProviderResponse](#bytebase-api-v1alpha-TestIdentityProviderResponse)
    - [UpdateIdentityProviderRequest](#bytebase-api-v1alpha-UpdateIdentityProviderRequest)
  
    - [IdentityProviderType](#bytebase-api-v1alpha-IdentityProviderType)
    - [OAuth2AuthStyle](#bytebase-api-v1alpha-OAuth2AuthStyle)
  
    - [IdentityProviderService](#bytebase-api-v1alpha-IdentityProviderService)
  
- [api/v1alpha/org_policy_service.proto](#api_v1alpha_org_policy_service-proto)
    - [CreatePolicyRequest](#bytebase-api-v1alpha-CreatePolicyRequest)
    - [DataSourceQueryPolicy](#bytebase-api-v1alpha-DataSourceQueryPolicy)
    - [DeletePolicyRequest](#bytebase-api-v1alpha-DeletePolicyRequest)
    - [DisableCopyDataPolicy](#bytebase-api-v1alpha-DisableCopyDataPolicy)
    - [ExportDataPolicy](#bytebase-api-v1alpha-ExportDataPolicy)
    - [GetPolicyRequest](#bytebase-api-v1alpha-GetPolicyRequest)
    - [ListPoliciesRequest](#bytebase-api-v1alpha-ListPoliciesRequest)
    - [ListPoliciesResponse](#bytebase-api-v1alpha-ListPoliciesResponse)
    - [MaskingExceptionPolicy](#bytebase-api-v1alpha-MaskingExceptionPolicy)
    - [MaskingExceptionPolicy.MaskingException](#bytebase-api-v1alpha-MaskingExceptionPolicy-MaskingException)
    - [MaskingRulePolicy](#bytebase-api-v1alpha-MaskingRulePolicy)
    - [MaskingRulePolicy.MaskingRule](#bytebase-api-v1alpha-MaskingRulePolicy-MaskingRule)
    - [Policy](#bytebase-api-v1alpha-Policy)
    - [QueryDataPolicy](#bytebase-api-v1alpha-QueryDataPolicy)
    - [RestrictIssueCreationForSQLReviewPolicy](#bytebase-api-v1alpha-RestrictIssueCreationForSQLReviewPolicy)
    - [RolloutPolicy](#bytebase-api-v1alpha-RolloutPolicy)
    - [SQLReviewRule](#bytebase-api-v1alpha-SQLReviewRule)
    - [TagPolicy](#bytebase-api-v1alpha-TagPolicy)
    - [TagPolicy.TagsEntry](#bytebase-api-v1alpha-TagPolicy-TagsEntry)
    - [UpdatePolicyRequest](#bytebase-api-v1alpha-UpdatePolicyRequest)
  
    - [DataSourceQueryPolicy.Restriction](#bytebase-api-v1alpha-DataSourceQueryPolicy-Restriction)
    - [MaskingExceptionPolicy.MaskingException.Action](#bytebase-api-v1alpha-MaskingExceptionPolicy-MaskingException-Action)
    - [PolicyResourceType](#bytebase-api-v1alpha-PolicyResourceType)
    - [PolicyType](#bytebase-api-v1alpha-PolicyType)
    - [SQLReviewRuleLevel](#bytebase-api-v1alpha-SQLReviewRuleLevel)
  
    - [OrgPolicyService](#bytebase-api-v1alpha-OrgPolicyService)
  
- [api/v1alpha/plan_service.proto](#api_v1alpha_plan_service-proto)
    - [BatchCancelPlanCheckRunsRequest](#bytebase-api-v1alpha-BatchCancelPlanCheckRunsRequest)
    - [BatchCancelPlanCheckRunsResponse](#bytebase-api-v1alpha-BatchCancelPlanCheckRunsResponse)
    - [CreatePlanRequest](#bytebase-api-v1alpha-CreatePlanRequest)
    - [GetPlanRequest](#bytebase-api-v1alpha-GetPlanRequest)
    - [ListPlanCheckRunsRequest](#bytebase-api-v1alpha-ListPlanCheckRunsRequest)
    - [ListPlanCheckRunsResponse](#bytebase-api-v1alpha-ListPlanCheckRunsResponse)
    - [ListPlansRequest](#bytebase-api-v1alpha-ListPlansRequest)
    - [ListPlansResponse](#bytebase-api-v1alpha-ListPlansResponse)
    - [Plan](#bytebase-api-v1alpha-Plan)
    - [Plan.ChangeDatabaseConfig](#bytebase-api-v1alpha-Plan-ChangeDatabaseConfig)
    - [Plan.ChangeDatabaseConfig.GhostFlagsEntry](#bytebase-api-v1alpha-Plan-ChangeDatabaseConfig-GhostFlagsEntry)
    - [Plan.ChangeDatabaseConfig.PreUpdateBackupDetail](#bytebase-api-v1alpha-Plan-ChangeDatabaseConfig-PreUpdateBackupDetail)
    - [Plan.CreateDatabaseConfig](#bytebase-api-v1alpha-Plan-CreateDatabaseConfig)
    - [Plan.Deployment](#bytebase-api-v1alpha-Plan-Deployment)
    - [Plan.Deployment.DatabaseGroupMapping](#bytebase-api-v1alpha-Plan-Deployment-DatabaseGroupMapping)
    - [Plan.ExportDataConfig](#bytebase-api-v1alpha-Plan-ExportDataConfig)
    - [Plan.PlanCheckRunStatusCountEntry](#bytebase-api-v1alpha-Plan-PlanCheckRunStatusCountEntry)
    - [Plan.ReleaseSource](#bytebase-api-v1alpha-Plan-ReleaseSource)
    - [Plan.Spec](#bytebase-api-v1alpha-Plan-Spec)
    - [Plan.SpecReleaseSource](#bytebase-api-v1alpha-Plan-SpecReleaseSource)
    - [Plan.Step](#bytebase-api-v1alpha-Plan-Step)
    - [PlanCheckRun](#bytebase-api-v1alpha-PlanCheckRun)
    - [PlanCheckRun.Result](#bytebase-api-v1alpha-PlanCheckRun-Result)
    - [PlanCheckRun.Result.SqlReviewReport](#bytebase-api-v1alpha-PlanCheckRun-Result-SqlReviewReport)
    - [PlanCheckRun.Result.SqlSummaryReport](#bytebase-api-v1alpha-PlanCheckRun-Result-SqlSummaryReport)
    - [PreviewPlanRequest](#bytebase-api-v1alpha-PreviewPlanRequest)
    - [PreviewPlanResponse](#bytebase-api-v1alpha-PreviewPlanResponse)
    - [PreviewPlanResponse.DatabaseFiles](#bytebase-api-v1alpha-PreviewPlanResponse-DatabaseFiles)
    - [RunPlanChecksRequest](#bytebase-api-v1alpha-RunPlanChecksRequest)
    - [RunPlanChecksResponse](#bytebase-api-v1alpha-RunPlanChecksResponse)
    - [SearchPlansRequest](#bytebase-api-v1alpha-SearchPlansRequest)
    - [SearchPlansResponse](#bytebase-api-v1alpha-SearchPlansResponse)
    - [UpdatePlanRequest](#bytebase-api-v1alpha-UpdatePlanRequest)
  
    - [Plan.ChangeDatabaseConfig.Type](#bytebase-api-v1alpha-Plan-ChangeDatabaseConfig-Type)
    - [PlanCheckRun.Result.Status](#bytebase-api-v1alpha-PlanCheckRun-Result-Status)
    - [PlanCheckRun.Status](#bytebase-api-v1alpha-PlanCheckRun-Status)
    - [PlanCheckRun.Type](#bytebase-api-v1alpha-PlanCheckRun-Type)
  
    - [PlanService](#bytebase-api-v1alpha-PlanService)
  
- [api/v1alpha/project_service.proto](#api_v1alpha_project_service-proto)
    - [Activity](#bytebase-api-v1alpha-Activity)
    - [AddWebhookRequest](#bytebase-api-v1alpha-AddWebhookRequest)
    - [BatchGetIamPolicyRequest](#bytebase-api-v1alpha-BatchGetIamPolicyRequest)
    - [BatchGetIamPolicyResponse](#bytebase-api-v1alpha-BatchGetIamPolicyResponse)
    - [BatchGetIamPolicyResponse.PolicyResult](#bytebase-api-v1alpha-BatchGetIamPolicyResponse-PolicyResult)
    - [CreateProjectRequest](#bytebase-api-v1alpha-CreateProjectRequest)
    - [DeleteProjectRequest](#bytebase-api-v1alpha-DeleteProjectRequest)
    - [GetProjectRequest](#bytebase-api-v1alpha-GetProjectRequest)
    - [Label](#bytebase-api-v1alpha-Label)
    - [ListProjectsRequest](#bytebase-api-v1alpha-ListProjectsRequest)
    - [ListProjectsResponse](#bytebase-api-v1alpha-ListProjectsResponse)
    - [Project](#bytebase-api-v1alpha-Project)
    - [Project.ExecutionRetryPolicy](#bytebase-api-v1alpha-Project-ExecutionRetryPolicy)
    - [RemoveWebhookRequest](#bytebase-api-v1alpha-RemoveWebhookRequest)
    - [SearchProjectsRequest](#bytebase-api-v1alpha-SearchProjectsRequest)
    - [SearchProjectsResponse](#bytebase-api-v1alpha-SearchProjectsResponse)
    - [TestWebhookRequest](#bytebase-api-v1alpha-TestWebhookRequest)
    - [TestWebhookResponse](#bytebase-api-v1alpha-TestWebhookResponse)
    - [UndeleteProjectRequest](#bytebase-api-v1alpha-UndeleteProjectRequest)
    - [UpdateProjectRequest](#bytebase-api-v1alpha-UpdateProjectRequest)
    - [UpdateWebhookRequest](#bytebase-api-v1alpha-UpdateWebhookRequest)
    - [Webhook](#bytebase-api-v1alpha-Webhook)
  
    - [Activity.Type](#bytebase-api-v1alpha-Activity-Type)
    - [Webhook.Type](#bytebase-api-v1alpha-Webhook-Type)
  
    - [ProjectService](#bytebase-api-v1alpha-ProjectService)
  
- [api/v1alpha/sql_service.proto](#api_v1alpha_sql_service-proto)
    - [AICompletionRequest](#bytebase-api-v1alpha-AICompletionRequest)
    - [AICompletionRequest.Message](#bytebase-api-v1alpha-AICompletionRequest-Message)
    - [AICompletionResponse](#bytebase-api-v1alpha-AICompletionResponse)
    - [AICompletionResponse.Candidate](#bytebase-api-v1alpha-AICompletionResponse-Candidate)
    - [AICompletionResponse.Candidate.Content](#bytebase-api-v1alpha-AICompletionResponse-Candidate-Content)
    - [AICompletionResponse.Candidate.Content.Part](#bytebase-api-v1alpha-AICompletionResponse-Candidate-Content-Part)
    - [AdminExecuteRequest](#bytebase-api-v1alpha-AdminExecuteRequest)
    - [AdminExecuteResponse](#bytebase-api-v1alpha-AdminExecuteResponse)
    - [Advice](#bytebase-api-v1alpha-Advice)
    - [CheckRequest](#bytebase-api-v1alpha-CheckRequest)
    - [CheckResponse](#bytebase-api-v1alpha-CheckResponse)
    - [DiffMetadataRequest](#bytebase-api-v1alpha-DiffMetadataRequest)
    - [DiffMetadataResponse](#bytebase-api-v1alpha-DiffMetadataResponse)
    - [ExportRequest](#bytebase-api-v1alpha-ExportRequest)
    - [ExportResponse](#bytebase-api-v1alpha-ExportResponse)
    - [ParseMyBatisMapperRequest](#bytebase-api-v1alpha-ParseMyBatisMapperRequest)
    - [ParseMyBatisMapperResponse](#bytebase-api-v1alpha-ParseMyBatisMapperResponse)
    - [PrettyRequest](#bytebase-api-v1alpha-PrettyRequest)
    - [PrettyResponse](#bytebase-api-v1alpha-PrettyResponse)
    - [QueryHistory](#bytebase-api-v1alpha-QueryHistory)
    - [QueryOption](#bytebase-api-v1alpha-QueryOption)
    - [QueryRequest](#bytebase-api-v1alpha-QueryRequest)
    - [QueryResponse](#bytebase-api-v1alpha-QueryResponse)
    - [QueryResult](#bytebase-api-v1alpha-QueryResult)
    - [QueryResult.PostgresError](#bytebase-api-v1alpha-QueryResult-PostgresError)
    - [QueryRow](#bytebase-api-v1alpha-QueryRow)
    - [RowValue](#bytebase-api-v1alpha-RowValue)
    - [RowValue.Timestamp](#bytebase-api-v1alpha-RowValue-Timestamp)
    - [RowValue.TimestampTZ](#bytebase-api-v1alpha-RowValue-TimestampTZ)
    - [SearchQueryHistoriesRequest](#bytebase-api-v1alpha-SearchQueryHistoriesRequest)
    - [SearchQueryHistoriesResponse](#bytebase-api-v1alpha-SearchQueryHistoriesResponse)
  
    - [Advice.Status](#bytebase-api-v1alpha-Advice-Status)
    - [CheckRequest.ChangeType](#bytebase-api-v1alpha-CheckRequest-ChangeType)
    - [QueryHistory.Type](#bytebase-api-v1alpha-QueryHistory-Type)
    - [QueryOption.RedisRunCommandsOn](#bytebase-api-v1alpha-QueryOption-RedisRunCommandsOn)
  
    - [SQLService](#bytebase-api-v1alpha-SQLService)
  
- [api/v1alpha/release_service.proto](#api_v1alpha_release_service-proto)
    - [CheckReleaseRequest](#bytebase-api-v1alpha-CheckReleaseRequest)
    - [CheckReleaseResponse](#bytebase-api-v1alpha-CheckReleaseResponse)
    - [CheckReleaseResponse.CheckResult](#bytebase-api-v1alpha-CheckReleaseResponse-CheckResult)
    - [CreateReleaseRequest](#bytebase-api-v1alpha-CreateReleaseRequest)
    - [DeleteReleaseRequest](#bytebase-api-v1alpha-DeleteReleaseRequest)
    - [GetReleaseRequest](#bytebase-api-v1alpha-GetReleaseRequest)
    - [ListReleasesRequest](#bytebase-api-v1alpha-ListReleasesRequest)
    - [ListReleasesResponse](#bytebase-api-v1alpha-ListReleasesResponse)
    - [Release](#bytebase-api-v1alpha-Release)
    - [Release.File](#bytebase-api-v1alpha-Release-File)
    - [Release.VCSSource](#bytebase-api-v1alpha-Release-VCSSource)
    - [UndeleteReleaseRequest](#bytebase-api-v1alpha-UndeleteReleaseRequest)
    - [UpdateReleaseRequest](#bytebase-api-v1alpha-UpdateReleaseRequest)
  
    - [CheckReleaseResponse.RiskLevel](#bytebase-api-v1alpha-CheckReleaseResponse-RiskLevel)
    - [Release.File.ChangeType](#bytebase-api-v1alpha-Release-File-ChangeType)
    - [ReleaseFileType](#bytebase-api-v1alpha-ReleaseFileType)
  
    - [ReleaseService](#bytebase-api-v1alpha-ReleaseService)
  
- [api/v1alpha/review_config_service.proto](#api_v1alpha_review_config_service-proto)
    - [CreateReviewConfigRequest](#bytebase-api-v1alpha-CreateReviewConfigRequest)
    - [DeleteReviewConfigRequest](#bytebase-api-v1alpha-DeleteReviewConfigRequest)
    - [GetReviewConfigRequest](#bytebase-api-v1alpha-GetReviewConfigRequest)
    - [ListReviewConfigsRequest](#bytebase-api-v1alpha-ListReviewConfigsRequest)
    - [ListReviewConfigsResponse](#bytebase-api-v1alpha-ListReviewConfigsResponse)
    - [ReviewConfig](#bytebase-api-v1alpha-ReviewConfig)
    - [UpdateReviewConfigRequest](#bytebase-api-v1alpha-UpdateReviewConfigRequest)
  
    - [ReviewConfigService](#bytebase-api-v1alpha-ReviewConfigService)
  
- [api/v1alpha/risk_service.proto](#api_v1alpha_risk_service-proto)
    - [CreateRiskRequest](#bytebase-api-v1alpha-CreateRiskRequest)
    - [DeleteRiskRequest](#bytebase-api-v1alpha-DeleteRiskRequest)
    - [ListRisksRequest](#bytebase-api-v1alpha-ListRisksRequest)
    - [ListRisksResponse](#bytebase-api-v1alpha-ListRisksResponse)
    - [Risk](#bytebase-api-v1alpha-Risk)
    - [UpdateRiskRequest](#bytebase-api-v1alpha-UpdateRiskRequest)
  
    - [Risk.Source](#bytebase-api-v1alpha-Risk-Source)
  
    - [RiskService](#bytebase-api-v1alpha-RiskService)
  
- [api/v1alpha/role_service.proto](#api_v1alpha_role_service-proto)
    - [CreateRoleRequest](#bytebase-api-v1alpha-CreateRoleRequest)
    - [DeleteRoleRequest](#bytebase-api-v1alpha-DeleteRoleRequest)
    - [GetRoleRequest](#bytebase-api-v1alpha-GetRoleRequest)
    - [ListRolesRequest](#bytebase-api-v1alpha-ListRolesRequest)
    - [ListRolesResponse](#bytebase-api-v1alpha-ListRolesResponse)
    - [Role](#bytebase-api-v1alpha-Role)
    - [UpdateRoleRequest](#bytebase-api-v1alpha-UpdateRoleRequest)
  
    - [Role.Type](#bytebase-api-v1alpha-Role-Type)
  
    - [RoleService](#bytebase-api-v1alpha-RoleService)
  
- [api/v1alpha/rollout_service.proto](#api_v1alpha_rollout_service-proto)
    - [BatchCancelTaskRunsRequest](#bytebase-api-v1alpha-BatchCancelTaskRunsRequest)
    - [BatchCancelTaskRunsResponse](#bytebase-api-v1alpha-BatchCancelTaskRunsResponse)
    - [BatchRunTasksRequest](#bytebase-api-v1alpha-BatchRunTasksRequest)
    - [BatchRunTasksResponse](#bytebase-api-v1alpha-BatchRunTasksResponse)
    - [BatchSkipTasksRequest](#bytebase-api-v1alpha-BatchSkipTasksRequest)
    - [BatchSkipTasksResponse](#bytebase-api-v1alpha-BatchSkipTasksResponse)
    - [CreateRolloutRequest](#bytebase-api-v1alpha-CreateRolloutRequest)
    - [GetRolloutRequest](#bytebase-api-v1alpha-GetRolloutRequest)
    - [GetTaskRunLogRequest](#bytebase-api-v1alpha-GetTaskRunLogRequest)
    - [GetTaskRunRequest](#bytebase-api-v1alpha-GetTaskRunRequest)
    - [GetTaskRunSessionRequest](#bytebase-api-v1alpha-GetTaskRunSessionRequest)
    - [ListRolloutsRequest](#bytebase-api-v1alpha-ListRolloutsRequest)
    - [ListRolloutsResponse](#bytebase-api-v1alpha-ListRolloutsResponse)
    - [ListTaskRunsRequest](#bytebase-api-v1alpha-ListTaskRunsRequest)
    - [ListTaskRunsResponse](#bytebase-api-v1alpha-ListTaskRunsResponse)
    - [PreviewRolloutRequest](#bytebase-api-v1alpha-PreviewRolloutRequest)
    - [PreviewTaskRunRollbackRequest](#bytebase-api-v1alpha-PreviewTaskRunRollbackRequest)
    - [PreviewTaskRunRollbackResponse](#bytebase-api-v1alpha-PreviewTaskRunRollbackResponse)
    - [Rollout](#bytebase-api-v1alpha-Rollout)
    - [Stage](#bytebase-api-v1alpha-Stage)
    - [Task](#bytebase-api-v1alpha-Task)
    - [Task.DatabaseCreate](#bytebase-api-v1alpha-Task-DatabaseCreate)
    - [Task.DatabaseDataExport](#bytebase-api-v1alpha-Task-DatabaseDataExport)
    - [Task.DatabaseDataUpdate](#bytebase-api-v1alpha-Task-DatabaseDataUpdate)
    - [Task.DatabaseSchemaBaseline](#bytebase-api-v1alpha-Task-DatabaseSchemaBaseline)
    - [Task.DatabaseSchemaUpdate](#bytebase-api-v1alpha-Task-DatabaseSchemaUpdate)
    - [TaskRun](#bytebase-api-v1alpha-TaskRun)
    - [TaskRun.PriorBackupDetail](#bytebase-api-v1alpha-TaskRun-PriorBackupDetail)
    - [TaskRun.PriorBackupDetail.Item](#bytebase-api-v1alpha-TaskRun-PriorBackupDetail-Item)
    - [TaskRun.PriorBackupDetail.Item.Table](#bytebase-api-v1alpha-TaskRun-PriorBackupDetail-Item-Table)
    - [TaskRun.SchedulerInfo](#bytebase-api-v1alpha-TaskRun-SchedulerInfo)
    - [TaskRun.SchedulerInfo.WaitingCause](#bytebase-api-v1alpha-TaskRun-SchedulerInfo-WaitingCause)
    - [TaskRun.SchedulerInfo.WaitingCause.Task](#bytebase-api-v1alpha-TaskRun-SchedulerInfo-WaitingCause-Task)
    - [TaskRunLog](#bytebase-api-v1alpha-TaskRunLog)
    - [TaskRunLogEntry](#bytebase-api-v1alpha-TaskRunLogEntry)
    - [TaskRunLogEntry.CommandExecute](#bytebase-api-v1alpha-TaskRunLogEntry-CommandExecute)
    - [TaskRunLogEntry.CommandExecute.CommandResponse](#bytebase-api-v1alpha-TaskRunLogEntry-CommandExecute-CommandResponse)
    - [TaskRunLogEntry.DatabaseSync](#bytebase-api-v1alpha-TaskRunLogEntry-DatabaseSync)
    - [TaskRunLogEntry.PriorBackup](#bytebase-api-v1alpha-TaskRunLogEntry-PriorBackup)
    - [TaskRunLogEntry.RetryInfo](#bytebase-api-v1alpha-TaskRunLogEntry-RetryInfo)
    - [TaskRunLogEntry.SchemaDump](#bytebase-api-v1alpha-TaskRunLogEntry-SchemaDump)
    - [TaskRunLogEntry.TaskRunStatusUpdate](#bytebase-api-v1alpha-TaskRunLogEntry-TaskRunStatusUpdate)
    - [TaskRunLogEntry.TransactionControl](#bytebase-api-v1alpha-TaskRunLogEntry-TransactionControl)
    - [TaskRunSession](#bytebase-api-v1alpha-TaskRunSession)
    - [TaskRunSession.Postgres](#bytebase-api-v1alpha-TaskRunSession-Postgres)
    - [TaskRunSession.Postgres.Session](#bytebase-api-v1alpha-TaskRunSession-Postgres-Session)
  
    - [Task.Status](#bytebase-api-v1alpha-Task-Status)
    - [Task.Type](#bytebase-api-v1alpha-Task-Type)
    - [TaskRun.ExportArchiveStatus](#bytebase-api-v1alpha-TaskRun-ExportArchiveStatus)
    - [TaskRun.Status](#bytebase-api-v1alpha-TaskRun-Status)
    - [TaskRunLogEntry.TaskRunStatusUpdate.Status](#bytebase-api-v1alpha-TaskRunLogEntry-TaskRunStatusUpdate-Status)
    - [TaskRunLogEntry.TransactionControl.Type](#bytebase-api-v1alpha-TaskRunLogEntry-TransactionControl-Type)
    - [TaskRunLogEntry.Type](#bytebase-api-v1alpha-TaskRunLogEntry-Type)
  
    - [RolloutService](#bytebase-api-v1alpha-RolloutService)
  
- [api/v1alpha/sheet_service.proto](#api_v1alpha_sheet_service-proto)
    - [BatchCreateSheetRequest](#bytebase-api-v1alpha-BatchCreateSheetRequest)
    - [BatchCreateSheetResponse](#bytebase-api-v1alpha-BatchCreateSheetResponse)
    - [CreateSheetRequest](#bytebase-api-v1alpha-CreateSheetRequest)
    - [GetSheetRequest](#bytebase-api-v1alpha-GetSheetRequest)
    - [Sheet](#bytebase-api-v1alpha-Sheet)
    - [SheetCommand](#bytebase-api-v1alpha-SheetCommand)
    - [SheetPayload](#bytebase-api-v1alpha-SheetPayload)
    - [UpdateSheetRequest](#bytebase-api-v1alpha-UpdateSheetRequest)
  
    - [SheetPayload.Type](#bytebase-api-v1alpha-SheetPayload-Type)
  
    - [SheetService](#bytebase-api-v1alpha-SheetService)
  
- [api/v1alpha/worksheet_service.proto](#api_v1alpha_worksheet_service-proto)
    - [CreateWorksheetRequest](#bytebase-api-v1alpha-CreateWorksheetRequest)
    - [DeleteWorksheetRequest](#bytebase-api-v1alpha-DeleteWorksheetRequest)
    - [GetWorksheetRequest](#bytebase-api-v1alpha-GetWorksheetRequest)
    - [SearchWorksheetsRequest](#bytebase-api-v1alpha-SearchWorksheetsRequest)
    - [SearchWorksheetsResponse](#bytebase-api-v1alpha-SearchWorksheetsResponse)
    - [UpdateWorksheetOrganizerRequest](#bytebase-api-v1alpha-UpdateWorksheetOrganizerRequest)
    - [UpdateWorksheetRequest](#bytebase-api-v1alpha-UpdateWorksheetRequest)
    - [Worksheet](#bytebase-api-v1alpha-Worksheet)
    - [WorksheetOrganizer](#bytebase-api-v1alpha-WorksheetOrganizer)
  
    - [Worksheet.Visibility](#bytebase-api-v1alpha-Worksheet-Visibility)
  
    - [WorksheetService](#bytebase-api-v1alpha-WorksheetService)
  
- [api/v1alpha/workspace_service.proto](#api_v1alpha_workspace_service-proto)
    - [WorkspaceService](#bytebase-api-v1alpha-WorkspaceService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="api_v1alpha_annotation-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/annotation.proto


 


<a name="bytebase-api-v1alpha-AuthMethod"></a>

### AuthMethod


| Name | Number | Description |
| ---- | ------ | ----------- |
| AUTH_METHOD_UNSPECIFIED | 0 |  |
| IAM | 1 | IAM uses the standard IAM authorization check on the organizational resources. |
| CUSTOM | 2 | Custom authorization method. |


 


<a name="api_v1alpha_annotation-proto-extensions"></a>

### File-level Extensions
| Extension | Type | Base | Number | Description |
| --------- | ---- | ---- | ------ | ----------- |
| allow_without_credential | bool | .google.protobuf.MethodOptions | 100000 |  |
| audit | bool | .google.protobuf.MethodOptions | 100003 |  |
| auth_method | AuthMethod | .google.protobuf.MethodOptions | 100002 |  |
| permission | string | .google.protobuf.MethodOptions | 100001 |  |

 

 



<a name="api_v1alpha_common-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/common.proto



<a name="bytebase-api-v1alpha-Position"></a>

### Position
Position in a text expressed as zero-based line and zero-based column byte
offset.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| line | [int32](#int32) |  | Line position in a text (zero-based). |
| column | [int32](#int32) |  | Column position in a text (zero-based), equivalent to byte offset. |






<a name="bytebase-api-v1alpha-Range"></a>

### Range



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start | [int32](#int32) |  |  |
| end | [int32](#int32) |  |  |





 


<a name="bytebase-api-v1alpha-Engine"></a>

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
| COSMOSDB | 26 |  |
| TRINO | 27 |  |
| CASSANDRA | 28 |  |



<a name="bytebase-api-v1alpha-ExportFormat"></a>

### ExportFormat


| Name | Number | Description |
| ---- | ------ | ----------- |
| FORMAT_UNSPECIFIED | 0 |  |
| CSV | 1 |  |
| JSON | 2 |  |
| SQL | 3 |  |
| XLSX | 4 |  |



<a name="bytebase-api-v1alpha-State"></a>

### State


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATE_UNSPECIFIED | 0 |  |
| ACTIVE | 1 |  |
| DELETED | 2 |  |



<a name="bytebase-api-v1alpha-VCSType"></a>

### VCSType


| Name | Number | Description |
| ---- | ------ | ----------- |
| VCS_TYPE_UNSPECIFIED | 0 |  |
| GITHUB | 1 |  |
| GITLAB | 2 |  |
| BITBUCKET | 3 |  |
| AZURE_DEVOPS | 4 |  |


 

 

 



<a name="api_v1alpha_database_catalog_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/database_catalog_service.proto



<a name="bytebase-api-v1alpha-ColumnCatalog"></a>

### ColumnCatalog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| semantic_type | [string](#string) |  |  |
| labels | [ColumnCatalog.LabelsEntry](#bytebase-api-v1alpha-ColumnCatalog-LabelsEntry) | repeated | The user labels for a column. |
| classification | [string](#string) |  |  |
| object_schema | [ObjectSchema](#bytebase-api-v1alpha-ObjectSchema) | optional |  |






<a name="bytebase-api-v1alpha-ColumnCatalog-LabelsEntry"></a>

### ColumnCatalog.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-DatabaseCatalog"></a>

### DatabaseCatalog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database catalog. Format: instances/{instance}/databases/{database}/catalog |
| schemas | [SchemaCatalog](#bytebase-api-v1alpha-SchemaCatalog) | repeated |  |






<a name="bytebase-api-v1alpha-GetDatabaseCatalogRequest"></a>

### GetDatabaseCatalogRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database catalog to retrieve. Format: instances/{instance}/databases/{database}/catalog |






<a name="bytebase-api-v1alpha-ObjectSchema"></a>

### ObjectSchema



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [ObjectSchema.Type](#bytebase-api-v1alpha-ObjectSchema-Type) |  |  |
| struct_kind | [ObjectSchema.StructKind](#bytebase-api-v1alpha-ObjectSchema-StructKind) |  |  |
| array_kind | [ObjectSchema.ArrayKind](#bytebase-api-v1alpha-ObjectSchema-ArrayKind) |  |  |
| semantic_type | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-ObjectSchema-ArrayKind"></a>

### ObjectSchema.ArrayKind



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| kind | [ObjectSchema](#bytebase-api-v1alpha-ObjectSchema) |  |  |






<a name="bytebase-api-v1alpha-ObjectSchema-StructKind"></a>

### ObjectSchema.StructKind



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| properties | [ObjectSchema.StructKind.PropertiesEntry](#bytebase-api-v1alpha-ObjectSchema-StructKind-PropertiesEntry) | repeated |  |






<a name="bytebase-api-v1alpha-ObjectSchema-StructKind-PropertiesEntry"></a>

### ObjectSchema.StructKind.PropertiesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [ObjectSchema](#bytebase-api-v1alpha-ObjectSchema) |  |  |






<a name="bytebase-api-v1alpha-SchemaCatalog"></a>

### SchemaCatalog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| tables | [TableCatalog](#bytebase-api-v1alpha-TableCatalog) | repeated |  |






<a name="bytebase-api-v1alpha-TableCatalog"></a>

### TableCatalog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| columns | [TableCatalog.Columns](#bytebase-api-v1alpha-TableCatalog-Columns) |  |  |
| object_schema | [ObjectSchema](#bytebase-api-v1alpha-ObjectSchema) |  |  |
| classification | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-TableCatalog-Columns"></a>

### TableCatalog.Columns



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| columns | [ColumnCatalog](#bytebase-api-v1alpha-ColumnCatalog) | repeated |  |






<a name="bytebase-api-v1alpha-UpdateDatabaseCatalogRequest"></a>

### UpdateDatabaseCatalogRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| catalog | [DatabaseCatalog](#bytebase-api-v1alpha-DatabaseCatalog) |  | The database catalog to update.

The catalog&#39;s `name` field is used to identify the database catalog to update. Format: instances/{instance}/databases/{database}/catalog |





 


<a name="bytebase-api-v1alpha-ObjectSchema-Type"></a>

### ObjectSchema.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| STRING | 1 |  |
| NUMBER | 2 |  |
| BOOLEAN | 3 |  |
| OBJECT | 4 |  |
| ARRAY | 5 |  |


 

 


<a name="bytebase-api-v1alpha-DatabaseCatalogService"></a>

### DatabaseCatalogService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetDatabaseCatalog | [GetDatabaseCatalogRequest](#bytebase-api-v1alpha-GetDatabaseCatalogRequest) | [DatabaseCatalog](#bytebase-api-v1alpha-DatabaseCatalog) |  |
| UpdateDatabaseCatalog | [UpdateDatabaseCatalogRequest](#bytebase-api-v1alpha-UpdateDatabaseCatalogRequest) | [DatabaseCatalog](#bytebase-api-v1alpha-DatabaseCatalog) |  |

 



<a name="api_v1alpha_instance_role_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/instance_role_service.proto



<a name="bytebase-api-v1alpha-GetInstanceRoleRequest"></a>

### GetInstanceRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the role to retrieve. Format: instances/{instance}/roles/{role name} The role name is the unique name for the role. |






<a name="bytebase-api-v1alpha-InstanceRole"></a>

### InstanceRole
InstanceRole is the API message for instance role.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the role. Format: instances/{instance}/roles/{role} The role name is the unique name for the role. |
| role_name | [string](#string) |  | The role name. It&#39;s unique within the instance. |
| password | [string](#string) | optional | The role password. |
| connection_limit | [int32](#int32) | optional | The connection count limit for this role. |
| valid_until | [string](#string) | optional | The expiration for the role&#39;s password. |
| attribute | [string](#string) | optional | The role attribute. For PostgreSQL, it containt super_user, no_inherit, create_role, create_db, can_login, replication and bypass_rls. Docs: https://www.postgresql.org/docs/current/role-attributes.html For MySQL, it&#39;s the global privileges as GRANT statements, which means it only contains &#34;GRANT ... ON *.* TO ...&#34;. Docs: https://dev.mysql.com/doc/refman/8.0/en/grant.html |






<a name="bytebase-api-v1alpha-ListInstanceRolesRequest"></a>

### ListInstanceRolesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of roles. Format: instances/{instance} |
| page_size | [int32](#int32) |  | Not used. The maximum number of roles to return. The service may return fewer than this value. If unspecified, at most 10 roles will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListInstanceRoles` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListInstanceRoles` must match the call that provided the page token. |
| refresh | [bool](#bool) |  | Refresh will refresh and return the latest data. |






<a name="bytebase-api-v1alpha-ListInstanceRolesResponse"></a>

### ListInstanceRolesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| roles | [InstanceRole](#bytebase-api-v1alpha-InstanceRole) | repeated | The roles from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |





 

 

 


<a name="bytebase-api-v1alpha-InstanceRoleService"></a>

### InstanceRoleService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetInstanceRole | [GetInstanceRoleRequest](#bytebase-api-v1alpha-GetInstanceRoleRequest) | [InstanceRole](#bytebase-api-v1alpha-InstanceRole) |  |
| ListInstanceRoles | [ListInstanceRolesRequest](#bytebase-api-v1alpha-ListInstanceRolesRequest) | [ListInstanceRolesResponse](#bytebase-api-v1alpha-ListInstanceRolesResponse) |  |

 



<a name="api_v1alpha_instance_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/instance_service.proto



<a name="bytebase-api-v1alpha-AddDataSourceRequest"></a>

### AddDataSourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the instance to add a data source to. Format: instances/{instance} |
| data_source | [DataSource](#bytebase-api-v1alpha-DataSource) |  | Identified by data source ID. Only READ_ONLY data source can be added. |
| validate_only | [bool](#bool) |  | Validate only also tests the data source connection. |






<a name="bytebase-api-v1alpha-BatchSyncInstancesRequest"></a>

### BatchSyncInstancesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| requests | [SyncInstanceRequest](#bytebase-api-v1alpha-SyncInstanceRequest) | repeated | The request message specifying the instances to sync. A maximum of 1000 instances can be synced in a batch. |






<a name="bytebase-api-v1alpha-BatchSyncInstancesResponse"></a>

### BatchSyncInstancesResponse







<a name="bytebase-api-v1alpha-CreateInstanceRequest"></a>

### CreateInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance | [Instance](#bytebase-api-v1alpha-Instance) |  | The instance to create. |
| instance_id | [string](#string) |  | The ID to use for the instance, which will become the final component of the instance&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |
| validate_only | [bool](#bool) |  | Validate only also tests the data source connection. |






<a name="bytebase-api-v1alpha-DataSource"></a>

### DataSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| type | [DataSourceType](#bytebase-api-v1alpha-DataSourceType) |  |  |
| username | [string](#string) |  |  |
| password | [string](#string) |  |  |
| use_ssl | [bool](#bool) |  | Use SSL to connect to the data source. By default, we use system default SSL configuration. |
| ssl_ca | [string](#string) |  |  |
| ssl_cert | [string](#string) |  |  |
| ssl_key | [string](#string) |  |  |
| host | [string](#string) |  |  |
| port | [string](#string) |  |  |
| database | [string](#string) |  |  |
| srv | [bool](#bool) |  | srv, authentication_database and replica_set are used for MongoDB. srv is a boolean flag that indicates whether the host is a DNS SRV record. |
| authentication_database | [string](#string) |  | authentication_database is the database name to authenticate against, which stores the user credentials. |
| replica_set | [string](#string) |  | replica_set is used for MongoDB replica set. |
| sid | [string](#string) |  | sid and service_name are used for Oracle. |
| service_name | [string](#string) |  |  |
| ssh_host | [string](#string) |  | Connection over SSH. The hostname of the SSH server agent. Required. |
| ssh_port | [string](#string) |  | The port of the SSH server agent. It&#39;s 22 typically. Required. |
| ssh_user | [string](#string) |  | The user to login the server. Required. |
| ssh_password | [string](#string) |  | The password to login the server. If it&#39;s empty string, no password is required. |
| ssh_private_key | [string](#string) |  | The private key to login the server. If it&#39;s empty string, we will use the system default private key from os.Getenv(&#34;SSH_AUTH_SOCK&#34;). |
| authentication_private_key | [string](#string) |  | PKCS#8 private key in PEM format. If it&#39;s empty string, no private key is required. Used for authentication when connecting to the data source. |
| external_secret | [DataSourceExternalSecret](#bytebase-api-v1alpha-DataSourceExternalSecret) |  |  |
| authentication_type | [DataSource.AuthenticationType](#bytebase-api-v1alpha-DataSource-AuthenticationType) |  |  |
| client_secret_credential | [DataSource.ClientSecretCredential](#bytebase-api-v1alpha-DataSource-ClientSecretCredential) |  |  |
| sasl_config | [SASLConfig](#bytebase-api-v1alpha-SASLConfig) |  |  |
| additional_addresses | [DataSource.Address](#bytebase-api-v1alpha-DataSource-Address) | repeated | additional_addresses is used for MongoDB replica set. |
| direct_connection | [bool](#bool) |  | direct_connection is used for MongoDB to dispatch all the operations to the node specified in the connection string. |
| region | [string](#string) |  | region is the location of where the DB is, works for AWS RDS. For example, us-east-1. |
| warehouse_id | [string](#string) |  | warehouse_id is used by Databricks. |
| master_name | [string](#string) |  | master_name is the master name used by connecting redis-master via redis sentinel. |
| master_username | [string](#string) |  | master_username and master_password are master credentials used by redis sentinel mode. |
| master_password | [string](#string) |  |  |
| redis_type | [DataSource.RedisType](#bytebase-api-v1alpha-DataSource-RedisType) |  |  |
| cluster | [string](#string) |  | Cluster is the cluster name for the data source. Used by CockroachDB. |
| extra_connection_parameters | [DataSource.ExtraConnectionParametersEntry](#bytebase-api-v1alpha-DataSource-ExtraConnectionParametersEntry) | repeated | Extra connection parameters for the database connection. For PostgreSQL HA, this can be used to set target_session_attrs=read-write |






<a name="bytebase-api-v1alpha-DataSource-Address"></a>

### DataSource.Address



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| host | [string](#string) |  |  |
| port | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-DataSource-ClientSecretCredential"></a>

### DataSource.ClientSecretCredential



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tenant_id | [string](#string) |  |  |
| client_id | [string](#string) |  |  |
| client_secret | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-DataSource-ExtraConnectionParametersEntry"></a>

### DataSource.ExtraConnectionParametersEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-DataSourceExternalSecret"></a>

### DataSourceExternalSecret



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secret_type | [DataSourceExternalSecret.SecretType](#bytebase-api-v1alpha-DataSourceExternalSecret-SecretType) |  |  |
| url | [string](#string) |  |  |
| auth_type | [DataSourceExternalSecret.AuthType](#bytebase-api-v1alpha-DataSourceExternalSecret-AuthType) |  |  |
| app_role | [DataSourceExternalSecret.AppRoleAuthOption](#bytebase-api-v1alpha-DataSourceExternalSecret-AppRoleAuthOption) |  |  |
| token | [string](#string) |  |  |
| engine_name | [string](#string) |  | engine name is the name for secret engine. |
| secret_name | [string](#string) |  | the secret name in the engine to store the password. |
| password_key_name | [string](#string) |  | the key name for the password. |






<a name="bytebase-api-v1alpha-DataSourceExternalSecret-AppRoleAuthOption"></a>

### DataSourceExternalSecret.AppRoleAuthOption



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role_id | [string](#string) |  |  |
| secret_id | [string](#string) |  | the secret id for the role without ttl. |
| type | [DataSourceExternalSecret.AppRoleAuthOption.SecretType](#bytebase-api-v1alpha-DataSourceExternalSecret-AppRoleAuthOption-SecretType) |  |  |
| mount_path | [string](#string) |  | The path where the approle auth method is mounted. |






<a name="bytebase-api-v1alpha-DeleteInstanceRequest"></a>

### DeleteInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the instance to delete. Format: instances/{instance} |
| force | [bool](#bool) |  | If set to true, any databases and sheets from this project will also be moved to default project, and all open issues will be closed. |






<a name="bytebase-api-v1alpha-GetInstanceRequest"></a>

### GetInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the instance to retrieve. Format: instances/{instance} |






<a name="bytebase-api-v1alpha-Instance"></a>

### Instance



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the instance. Format: instances/{instance} |
| state | [State](#bytebase-api-v1alpha-State) |  |  |
| title | [string](#string) |  |  |
| engine | [Engine](#bytebase-api-v1alpha-Engine) |  |  |
| engine_version | [string](#string) |  |  |
| external_link | [string](#string) |  |  |
| data_sources | [DataSource](#bytebase-api-v1alpha-DataSource) | repeated |  |
| environment | [string](#string) |  | The environment resource. Format: environments/prod where prod is the environment resource ID. |
| activation | [bool](#bool) |  |  |
| roles | [InstanceRole](#bytebase-api-v1alpha-InstanceRole) | repeated |  |
| sync_interval | [google.protobuf.Duration](#google-protobuf-Duration) |  | How often the instance is synced. |
| maximum_connections | [int32](#int32) |  | The maximum number of connections. The default is 10 if the value is unset or zero. |
| sync_databases | [string](#string) | repeated | Enable sync for following databases. Default empty, means sync all schemas &amp; databases. |






<a name="bytebase-api-v1alpha-InstanceResource"></a>

### InstanceResource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  |  |
| engine | [Engine](#bytebase-api-v1alpha-Engine) |  |  |
| engine_version | [string](#string) |  |  |
| data_sources | [DataSource](#bytebase-api-v1alpha-DataSource) | repeated |  |
| activation | [bool](#bool) |  |  |
| name | [string](#string) |  | The name of the instance. Format: instances/{instance} |
| environment | [string](#string) |  | The environment resource. Format: environments/prod where prod is the environment resource ID. |






<a name="bytebase-api-v1alpha-KerberosConfig"></a>

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






<a name="bytebase-api-v1alpha-ListInstanceDatabaseRequest"></a>

### ListInstanceDatabaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the instance. Format: instances/{instance} |
| instance | [Instance](#bytebase-api-v1alpha-Instance) | optional | The target instance. We need to set this field if the target instance is not created yet. |






<a name="bytebase-api-v1alpha-ListInstanceDatabaseResponse"></a>

### ListInstanceDatabaseResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [string](#string) | repeated | All database name list in the instance. |






<a name="bytebase-api-v1alpha-ListInstancesRequest"></a>

### ListInstancesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of instances to return. The service may return fewer than this value. If unspecified, at most 10 instances will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListInstances` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListInstances` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Show deleted instances if specified. |
| filter | [string](#string) |  | Filter the instance. Supported filters: - name - resource_id - environment - state - engine - host - port - project

For example: name == &#34;sample instance&#34; name.matches(&#34;sample&#34;) resource_id = &#34;sample-instance&#34; resource_id.matches(&#34;sample&#34;) state == &#34;DELETED&#34; environment == &#34;environments/test&#34; engine == &#34;MYSQL&#34; engine in [&#34;MYSQL&#34;, &#34;POSTGRES&#34;] !(engine in [&#34;MYSQL&#34;, &#34;POSTGRES&#34;]) host == &#34;127.0.0.1&#34; port == &#34;54321&#34; project == &#34;projects/sample-project&#34; You can combine filter conditions like: name.matches(&#34;sample&#34;) &amp;&amp; environment == &#34;environments/test&#34; host == &#34;127.0.0.1&#34; &amp;&amp; port == &#34;54321&#34; |






<a name="bytebase-api-v1alpha-ListInstancesResponse"></a>

### ListInstancesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instances | [Instance](#bytebase-api-v1alpha-Instance) | repeated | The instances from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-RemoveDataSourceRequest"></a>

### RemoveDataSourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the instance to remove a data source from. Format: instances/{instance} |
| data_source | [DataSource](#bytebase-api-v1alpha-DataSource) |  | Identified by data source ID. Only READ_ONLY data source can be removed. |






<a name="bytebase-api-v1alpha-SASLConfig"></a>

### SASLConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| krb_config | [KerberosConfig](#bytebase-api-v1alpha-KerberosConfig) |  |  |






<a name="bytebase-api-v1alpha-SyncInstanceRequest"></a>

### SyncInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of instance. Format: instances/{instance} |
| enable_full_sync | [bool](#bool) |  | When full sync is enabled, all databases in the instance will be synchronized. Otherwise, only the instance metadata (such as the database list) and any newly discovered instances will be synced. |






<a name="bytebase-api-v1alpha-SyncInstanceResponse"></a>

### SyncInstanceResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [string](#string) | repeated | All database name list in the instance. |






<a name="bytebase-api-v1alpha-UndeleteInstanceRequest"></a>

### UndeleteInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the deleted instance. Format: instances/{instance} |






<a name="bytebase-api-v1alpha-UpdateDataSourceRequest"></a>

### UpdateDataSourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the instance to update a data source. Format: instances/{instance} |
| data_source | [DataSource](#bytebase-api-v1alpha-DataSource) |  | Identified by data source ID. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| validate_only | [bool](#bool) |  | Validate only also tests the data source connection. |






<a name="bytebase-api-v1alpha-UpdateInstanceRequest"></a>

### UpdateInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance | [Instance](#bytebase-api-v1alpha-Instance) |  | The instance to update.

The instance&#39;s `name` field is used to identify the instance to update. Format: instances/{instance} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 


<a name="bytebase-api-v1alpha-DataSource-AuthenticationType"></a>

### DataSource.AuthenticationType


| Name | Number | Description |
| ---- | ------ | ----------- |
| AUTHENTICATION_UNSPECIFIED | 0 |  |
| PASSWORD | 1 |  |
| GOOGLE_CLOUD_SQL_IAM | 2 |  |
| AWS_RDS_IAM | 3 |  |
| AZURE_IAM | 4 |  |



<a name="bytebase-api-v1alpha-DataSource-RedisType"></a>

### DataSource.RedisType


| Name | Number | Description |
| ---- | ------ | ----------- |
| REDIS_TYPE_UNSPECIFIED | 0 |  |
| STANDALONE | 1 |  |
| SENTINEL | 2 |  |
| CLUSTER | 3 |  |



<a name="bytebase-api-v1alpha-DataSourceExternalSecret-AppRoleAuthOption-SecretType"></a>

### DataSourceExternalSecret.AppRoleAuthOption.SecretType


| Name | Number | Description |
| ---- | ------ | ----------- |
| SECRET_TYPE_UNSPECIFIED | 0 |  |
| PLAIN | 1 |  |
| ENVIRONMENT | 2 |  |



<a name="bytebase-api-v1alpha-DataSourceExternalSecret-AuthType"></a>

### DataSourceExternalSecret.AuthType


| Name | Number | Description |
| ---- | ------ | ----------- |
| AUTH_TYPE_UNSPECIFIED | 0 |  |
| TOKEN | 1 | ref: https://developer.hashicorp.com/vault/docs/auth/token |
| VAULT_APP_ROLE | 2 | ref: https://developer.hashicorp.com/vault/docs/auth/approle |



<a name="bytebase-api-v1alpha-DataSourceExternalSecret-SecretType"></a>

### DataSourceExternalSecret.SecretType


| Name | Number | Description |
| ---- | ------ | ----------- |
| SAECRET_TYPE_UNSPECIFIED | 0 |  |
| VAULT_KV_V2 | 1 | ref: https://developer.hashicorp.com/vault/api-docs/secret/kv/kv-v2 |
| AWS_SECRETS_MANAGER | 2 | ref: https://docs.aws.amazon.com/secretsmanager/latest/userguide/intro.html |
| GCP_SECRET_MANAGER | 3 | ref: https://cloud.google.com/secret-manager/docs |



<a name="bytebase-api-v1alpha-DataSourceType"></a>

### DataSourceType


| Name | Number | Description |
| ---- | ------ | ----------- |
| DATA_SOURCE_UNSPECIFIED | 0 |  |
| ADMIN | 1 |  |
| READ_ONLY | 2 |  |


 

 


<a name="bytebase-api-v1alpha-InstanceService"></a>

### InstanceService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetInstance | [GetInstanceRequest](#bytebase-api-v1alpha-GetInstanceRequest) | [Instance](#bytebase-api-v1alpha-Instance) |  |
| ListInstances | [ListInstancesRequest](#bytebase-api-v1alpha-ListInstancesRequest) | [ListInstancesResponse](#bytebase-api-v1alpha-ListInstancesResponse) |  |
| CreateInstance | [CreateInstanceRequest](#bytebase-api-v1alpha-CreateInstanceRequest) | [Instance](#bytebase-api-v1alpha-Instance) |  |
| UpdateInstance | [UpdateInstanceRequest](#bytebase-api-v1alpha-UpdateInstanceRequest) | [Instance](#bytebase-api-v1alpha-Instance) |  |
| DeleteInstance | [DeleteInstanceRequest](#bytebase-api-v1alpha-DeleteInstanceRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| UndeleteInstance | [UndeleteInstanceRequest](#bytebase-api-v1alpha-UndeleteInstanceRequest) | [Instance](#bytebase-api-v1alpha-Instance) |  |
| SyncInstance | [SyncInstanceRequest](#bytebase-api-v1alpha-SyncInstanceRequest) | [SyncInstanceResponse](#bytebase-api-v1alpha-SyncInstanceResponse) |  |
| ListInstanceDatabase | [ListInstanceDatabaseRequest](#bytebase-api-v1alpha-ListInstanceDatabaseRequest) | [ListInstanceDatabaseResponse](#bytebase-api-v1alpha-ListInstanceDatabaseResponse) |  |
| BatchSyncInstances | [BatchSyncInstancesRequest](#bytebase-api-v1alpha-BatchSyncInstancesRequest) | [BatchSyncInstancesResponse](#bytebase-api-v1alpha-BatchSyncInstancesResponse) |  |
| AddDataSource | [AddDataSourceRequest](#bytebase-api-v1alpha-AddDataSourceRequest) | [Instance](#bytebase-api-v1alpha-Instance) |  |
| RemoveDataSource | [RemoveDataSourceRequest](#bytebase-api-v1alpha-RemoveDataSourceRequest) | [Instance](#bytebase-api-v1alpha-Instance) |  |
| UpdateDataSource | [UpdateDataSourceRequest](#bytebase-api-v1alpha-UpdateDataSourceRequest) | [Instance](#bytebase-api-v1alpha-Instance) |  |

 



<a name="api_v1alpha_database_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/database_service.proto



<a name="bytebase-api-v1alpha-BatchGetDatabasesRequest"></a>

### BatchGetDatabasesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource shared by all databases being retrieved. - projects/{project}: batch get databases in a project; - instances/{instances}: batch get databases in a instance; Use &#34;-&#34; as wildcard to batch get databases across parent. |
| names | [string](#string) | repeated | The list of database names to retrieve. |






<a name="bytebase-api-v1alpha-BatchGetDatabasesResponse"></a>

### BatchGetDatabasesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [Database](#bytebase-api-v1alpha-Database) | repeated | The databases from the specified request. |






<a name="bytebase-api-v1alpha-BatchSyncDatabasesRequest"></a>

### BatchSyncDatabasesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource shared by all databases being updated. Format: instances/{instance} If the operation spans parents, a dash (-) may be accepted as a wildcard. |
| names | [string](#string) | repeated | The list of database names to sync. |






<a name="bytebase-api-v1alpha-BatchSyncDatabasesResponse"></a>

### BatchSyncDatabasesResponse







<a name="bytebase-api-v1alpha-BatchUpdateDatabasesRequest"></a>

### BatchUpdateDatabasesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource shared by all databases being updated. Format: instances/{instance} If the operation spans parents, a dash (-) may be accepted as a wildcard. We only support updating the project of databases for now. |
| requests | [UpdateDatabaseRequest](#bytebase-api-v1alpha-UpdateDatabaseRequest) | repeated | The request message specifying the resources to update. A maximum of 1000 databases can be modified in a batch. |






<a name="bytebase-api-v1alpha-BatchUpdateDatabasesResponse"></a>

### BatchUpdateDatabasesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [Database](#bytebase-api-v1alpha-Database) | repeated | Databases updated. |






<a name="bytebase-api-v1alpha-ChangedResourceDatabase"></a>

### ChangedResourceDatabase



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| schemas | [ChangedResourceSchema](#bytebase-api-v1alpha-ChangedResourceSchema) | repeated |  |






<a name="bytebase-api-v1alpha-ChangedResourceFunction"></a>

### ChangedResourceFunction



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| ranges | [Range](#bytebase-api-v1alpha-Range) | repeated | The ranges of sub-strings correspond to the statements on the sheet. |






<a name="bytebase-api-v1alpha-ChangedResourceProcedure"></a>

### ChangedResourceProcedure



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| ranges | [Range](#bytebase-api-v1alpha-Range) | repeated | The ranges of sub-strings correspond to the statements on the sheet. |






<a name="bytebase-api-v1alpha-ChangedResourceSchema"></a>

### ChangedResourceSchema



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| tables | [ChangedResourceTable](#bytebase-api-v1alpha-ChangedResourceTable) | repeated |  |
| views | [ChangedResourceView](#bytebase-api-v1alpha-ChangedResourceView) | repeated |  |
| functions | [ChangedResourceFunction](#bytebase-api-v1alpha-ChangedResourceFunction) | repeated |  |
| procedures | [ChangedResourceProcedure](#bytebase-api-v1alpha-ChangedResourceProcedure) | repeated |  |






<a name="bytebase-api-v1alpha-ChangedResourceTable"></a>

### ChangedResourceTable



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| ranges | [Range](#bytebase-api-v1alpha-Range) | repeated | The ranges of sub-strings correspond to the statements on the sheet. |






<a name="bytebase-api-v1alpha-ChangedResourceView"></a>

### ChangedResourceView



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| ranges | [Range](#bytebase-api-v1alpha-Range) | repeated | The ranges of sub-strings correspond to the statements on the sheet. |






<a name="bytebase-api-v1alpha-ChangedResources"></a>

### ChangedResources



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [ChangedResourceDatabase](#bytebase-api-v1alpha-ChangedResourceDatabase) | repeated |  |






<a name="bytebase-api-v1alpha-Changelog"></a>

### Changelog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: instances/{instance}/databases/{database}/changelogs/{changelog} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| status | [Changelog.Status](#bytebase-api-v1alpha-Changelog-Status) |  |  |
| statement | [string](#string) |  | The statement is used for preview purpose. |
| statement_size | [int64](#int64) |  |  |
| statement_sheet | [string](#string) |  | The name of the sheet resource. Format: projects/{project}/sheets/{sheet} |
| schema | [string](#string) |  |  |
| schema_size | [int64](#int64) |  |  |
| prev_schema | [string](#string) |  |  |
| prev_schema_size | [int64](#int64) |  |  |
| issue | [string](#string) |  | Format: projects/{project}/issues/{issue} |
| task_run | [string](#string) |  | Format: projects/{projects}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |
| version | [string](#string) |  | Could be empty |
| revision | [string](#string) |  | Could be empty Or present but not found if deleted |
| changed_resources | [ChangedResources](#bytebase-api-v1alpha-ChangedResources) |  |  |
| type | [Changelog.Type](#bytebase-api-v1alpha-Changelog-Type) |  |  |






<a name="bytebase-api-v1alpha-CheckConstraintMetadata"></a>

### CheckConstraintMetadata
CheckConstraintMetadata is the metadata for check constraints.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a check constraint. |
| expression | [string](#string) |  | The expression is the expression of a check constraint. |






<a name="bytebase-api-v1alpha-ColumnMetadata"></a>

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
| default_on_null | [bool](#bool) |  | Oracle specific metadata. The default_on_null is the default on null of a column. |
| on_update | [string](#string) |  | The on_update is the on update action of a column. For MySQL like databases, it&#39;s only supported for TIMESTAMP columns with CURRENT_TIMESTAMP as on update value. |
| nullable | [bool](#bool) |  | The nullable is the nullable of a column. |
| type | [string](#string) |  | The type is the type of a column. |
| character_set | [string](#string) |  | The character_set is the character_set of a column. |
| collation | [string](#string) |  | The collation is the collation of a column. |
| comment | [string](#string) |  | The comment is the comment of a column. classification and user_comment is parsed from the comment. |
| user_comment | [string](#string) |  | The user_comment is the user comment of a column parsed from the comment. |
| generation | [GenerationMetadata](#bytebase-api-v1alpha-GenerationMetadata) |  | The generation is the generation of a column. |
| is_identity | [bool](#bool) |  |  |
| identity_generation | [ColumnMetadata.IdentityGeneration](#bytebase-api-v1alpha-ColumnMetadata-IdentityGeneration) |  | The identity_generation is for identity columns, PG only. |
| identity_seed | [int64](#int64) |  | The identity_seed is for identity columns, MSSQL only. |
| identity_increment | [int64](#int64) |  | The identity_increment is for identity columns, MSSQL only. |






<a name="bytebase-api-v1alpha-CreateRevisionRequest"></a>

### CreateRevisionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: instances/{instance}/databases/{database} |
| revision | [Revision](#bytebase-api-v1alpha-Revision) |  | The revision to create. |






<a name="bytebase-api-v1alpha-Database"></a>

### Database



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database. Format: instances/{instance}/databases/{database} {database} is the database name in the instance. |
| state | [State](#bytebase-api-v1alpha-State) |  | The existence of a database. |
| successful_sync_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The latest synchronization time. |
| project | [string](#string) |  | The project for a database. Format: projects/{project} |
| schema_version | [string](#string) |  | The version of database schema. |
| environment | [string](#string) |  | The environment resource. Format: environments/prod where prod is the environment resource ID. |
| effective_environment | [string](#string) |  | The effective environment based on environment tag above and environment tag on the instance. Inheritance follows https://cloud.google.com/resource-manager/docs/tags/tags-overview. |
| labels | [Database.LabelsEntry](#bytebase-api-v1alpha-Database-LabelsEntry) | repeated | Labels will be used for deployment and policy control. |
| instance_resource | [InstanceResource](#bytebase-api-v1alpha-InstanceResource) |  | The instance resource. |
| backup_available | [bool](#bool) |  | The database is available for DML prior backup. |
| drifted | [bool](#bool) |  | The schema is drifted from the source of truth. |






<a name="bytebase-api-v1alpha-Database-LabelsEntry"></a>

### Database.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-DatabaseMetadata"></a>

### DatabaseMetadata
DatabaseMetadata is the metadata for databases.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The database metadata name.

Format: instances/{instance}/databases/{database}/metadata |
| schemas | [SchemaMetadata](#bytebase-api-v1alpha-SchemaMetadata) | repeated | The schemas is the list of schemas in a database. |
| character_set | [string](#string) |  | The character_set is the character set of a database. |
| collation | [string](#string) |  | The collation is the collation of a database. |
| extensions | [ExtensionMetadata](#bytebase-api-v1alpha-ExtensionMetadata) | repeated | The extensions is the list of extensions in a database. |
| owner | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-DatabaseSchema"></a>

### DatabaseSchema
DatabaseSchema is the metadata for databases.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  | The schema dump from database. |






<a name="bytebase-api-v1alpha-DeleteRevisionRequest"></a>

### DeleteRevisionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the revision to delete. Format: instances/{instance}/databases/{database}/revisions/{revision} |






<a name="bytebase-api-v1alpha-DeleteSecretRequest"></a>

### DeleteSecretRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the secret to be deleted. Format: instances/{instance}/databases/{database}/secrets/{secret} |






<a name="bytebase-api-v1alpha-DependencyColumn"></a>

### DependencyColumn
DependencyColumn is the metadata for dependency columns.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  | The schema is the schema of a reference column. |
| table | [string](#string) |  | The table is the table of a reference column. |
| column | [string](#string) |  | The column is the name of a reference column. |






<a name="bytebase-api-v1alpha-DependencyTable"></a>

### DependencyTable



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  | The schema is the schema of a reference table. |
| table | [string](#string) |  | The table is the name of a reference table. |






<a name="bytebase-api-v1alpha-DiffSchemaRequest"></a>

### DiffSchemaRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database or changelog. Format: database: instances/{instance}/databases/{database} changelog: instances/{instance}/databases/{database}/changelogs/{changelog} |
| schema | [string](#string) |  | The target schema. |
| changelog | [string](#string) |  | The resource name of the changelog Format: instances/{instance}/databases/{database}/changelogs/{changelog} |
| sdl_format | [bool](#bool) |  | Format the schema dump into SDL format. |






<a name="bytebase-api-v1alpha-DiffSchemaResponse"></a>

### DiffSchemaResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| diff | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-EnumTypeMetadata"></a>

### EnumTypeMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of a type. |
| values | [string](#string) | repeated | The enum values of a type. |
| comment | [string](#string) |  |  |
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-api-v1alpha-EventMetadata"></a>

### EventMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the event. |
| definition | [string](#string) |  | The schedule of the event. |
| time_zone | [string](#string) |  | The time zone of the event. |
| sql_mode | [string](#string) |  |  |
| character_set_client | [string](#string) |  |  |
| collation_connection | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-ExtensionMetadata"></a>

### ExtensionMetadata
ExtensionMetadata is the metadata for extensions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of an extension. |
| schema | [string](#string) |  | The schema is the extension that is installed to. But the extension usage is not limited to the schema. |
| version | [string](#string) |  | The version is the version of an extension. |
| description | [string](#string) |  | The description is the description of an extension. |






<a name="bytebase-api-v1alpha-ExternalTableMetadata"></a>

### ExternalTableMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a external table. |
| external_server_name | [string](#string) |  | The external_server_name is the name of the external server. |
| external_database_name | [string](#string) |  | The external_database_name is the name of the external database. |
| columns | [ColumnMetadata](#bytebase-api-v1alpha-ColumnMetadata) | repeated | The columns is the ordered list of columns in a foreign table. |






<a name="bytebase-api-v1alpha-ForeignKeyMetadata"></a>

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






<a name="bytebase-api-v1alpha-FunctionMetadata"></a>

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
| comment | [string](#string) |  |  |
| dependency_tables | [DependencyTable](#bytebase-api-v1alpha-DependencyTable) | repeated | The dependency_tables is the list of dependency tables of a function. For PostgreSQL, it&#39;s the list of tables that the function depends on the return type definition. |
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-api-v1alpha-GenerationMetadata"></a>

### GenerationMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [GenerationMetadata.Type](#bytebase-api-v1alpha-GenerationMetadata-Type) |  |  |
| expression | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-GetChangelogRequest"></a>

### GetChangelogRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the changelog to retrieve. Format: instances/{instance}/databases/{database}/changelogs/{changelog} |
| view | [ChangelogView](#bytebase-api-v1alpha-ChangelogView) |  |  |
| sdl_format | [bool](#bool) |  | Format the schema dump into SDL format. |






<a name="bytebase-api-v1alpha-GetDatabaseMetadataRequest"></a>

### GetDatabaseMetadataRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to retrieve metadata. Format: instances/{instance}/databases/{database}/metadata |
| filter | [string](#string) |  | Filter is used to filter databases returned in the list. Supported filter: - schema - table

For example: schema == &#34;schema-a&#34; table == &#34;table-a&#34; schema == &#34;schema-a&#34; &amp;&amp; table == &#34;table-a&#34; The filter used for a specific schema object such as &#34;schemas/schema-a/tables/table-a&#34;. The column masking level will only be returned when a table filter is used. |






<a name="bytebase-api-v1alpha-GetDatabaseRequest"></a>

### GetDatabaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to retrieve. Format: instances/{instance}/databases/{database} |






<a name="bytebase-api-v1alpha-GetDatabaseSchemaRequest"></a>

### GetDatabaseSchemaRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to retrieve schema. Format: instances/{instance}/databases/{database}/schema |
| sdl_format | [bool](#bool) |  | Format the schema dump into SDL format. |






<a name="bytebase-api-v1alpha-GetRevisionRequest"></a>

### GetRevisionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the revision. Format: instances/{instance}/databases/{database}/revisions/{revision} |






<a name="bytebase-api-v1alpha-GetSchemaStringRequest"></a>

### GetSchemaStringRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database. Format: instances/{instance}/databases/{database} |
| type | [GetSchemaStringRequest.ObjectType](#bytebase-api-v1alpha-GetSchemaStringRequest-ObjectType) |  |  |
| schema | [string](#string) |  | It&#39;s empty for DATABASE. |
| object | [string](#string) |  | It&#39;s empty for DATABASE and SCHEMA. |
| metadata | [DatabaseMetadata](#bytebase-api-v1alpha-DatabaseMetadata) |  | If use the metadata to generate the schema string, the type is OBJECT_TYPE_UNSPECIFIED. Also the schema and object are empty. |






<a name="bytebase-api-v1alpha-GetSchemaStringResponse"></a>

### GetSchemaStringResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema_string | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-IndexMetadata"></a>

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
| parent_index_schema | [string](#string) |  | The schema name of the parent index. |
| parent_index_name | [string](#string) |  | The index name of the parent index. |
| granularity | [int64](#int64) |  | The number of granules in the block. It&#39;s a ClickHouse specific field. |
| is_constraint | [bool](#bool) |  | It&#39;s a PostgreSQL specific field. The unique constraint and unique index are not the same thing in PostgreSQL. |






<a name="bytebase-api-v1alpha-ListChangelogsRequest"></a>

### ListChangelogsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent of the changelogs. Format: instances/{instance}/databases/{database} |
| page_size | [int32](#int32) |  | The maximum number of changelogs to return. The service may return fewer than this value. If unspecified, at most 10 changelogs will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from the previous call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided must match the call that provided the page token. |
| view | [ChangelogView](#bytebase-api-v1alpha-ChangelogView) |  |  |
| filter | [string](#string) |  | The filter of the changelogs. follow the [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) syntax. Support filter by type, source or table. For example: table = &#34;tableExists(&#39;{database}&#39;, &#39;{schema}&#39;, &#39;{table}&#39;)&#34; table = &#34;tableExists(&#39;db&#39;, &#39;public&#39;, &#39;table1&#39;) || tableExists(&#39;db&#39;, &#39;public&#39;, &#39;table2&#39;)&#34;

The table filter follow the CEL syntax. currently, we have one function for CEL: - tableExists(database, schema, table): return true if the table exists in changed resources.

examples: Use tableExists(&#34;db&#34;, &#34;public&#34;, &#34;table1&#34;) to filter the changelogs which have the table &#34;table1&#34; in the schema &#34;public&#34; of the database &#34;db&#34;. For MySQL, the schema is always &#34;&#34;, such as tableExists(&#34;db&#34;, &#34;&#34;, &#34;table1&#34;).

Combine multiple functions with &#34;&amp;&amp;&#34; and &#34;||&#34;, we MUST use the Disjunctive Normal Form(DNF). In other words, the CEL expression consists of several parts connected by OR operators. For example, the following expression is valid: ( tableExists(&#34;db&#34;, &#34;public&#34;, &#34;table1&#34;) &amp;&amp; tableExists(&#34;db&#34;, &#34;public&#34;, &#34;table2&#34;) ) || ( tableExists(&#34;db&#34;, &#34;public&#34;, &#34;table3&#34;) ) |






<a name="bytebase-api-v1alpha-ListChangelogsResponse"></a>

### ListChangelogsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| changelogs | [Changelog](#bytebase-api-v1alpha-Changelog) | repeated | The list of changelogs. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-ListDatabasesRequest"></a>

### ListDatabasesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | - projects/{project}: list databases in a project, require &#34;bb.projects.get&#34; permission. - workspaces/-: list databases in the workspace, require &#34;bb.databases.list&#34; permission. - instances/{instances}: list databases in a instance, require &#34;bb.instances.get&#34; permission |
| page_size | [int32](#int32) |  | The maximum number of databases to return. The service may return fewer than this value. If unspecified, at most 10 databases will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListDatabases` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListDatabases` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter databases returned in the list. Supported filter: - environment - name - project - instance - engine - label - exclude_unassigned: Not show unassigned databases if specified - drifted

For example: environment == &#34;environments/{environment resource id}&#34; project == &#34;projects/{project resource id}&#34; instance == &#34;instances/{instance resource id}&#34; name.matches(&#34;database name&#34;) engine == &#34;MYSQL&#34; engine in [&#34;MYSQL&#34;, &#34;POSTGRES&#34;] !(engine in [&#34;MYSQL&#34;, &#34;POSTGRES&#34;]) label == &#34;region:asia&#34; label == &#34;tenant:asia,europe&#34; label == &#34;region:asia&#34; &amp;&amp; label == &#34;tenant:bytebase&#34; exclude_unassigned == true drifted == true You can combine filter conditions like: environment == &#34;environments/prod&#34; &amp;&amp; name.matches(&#34;employee&#34;) |
| show_deleted | [bool](#bool) |  | Show deleted database if specified. |






<a name="bytebase-api-v1alpha-ListDatabasesResponse"></a>

### ListDatabasesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [Database](#bytebase-api-v1alpha-Database) | repeated | The databases from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-ListRevisionsRequest"></a>

### ListRevisionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent of the revisions. Format: instances/{instance}/databases/{database} |
| page_size | [int32](#int32) |  | The maximum number of revisions to return. The service may return fewer than this value. If unspecified, at most 10 revisions will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListRevisions` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListRevisions` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  |  |






<a name="bytebase-api-v1alpha-ListRevisionsResponse"></a>

### ListRevisionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revisions | [Revision](#bytebase-api-v1alpha-Revision) | repeated |  |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-ListSecretsRequest"></a>

### ListSecretsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent of the secret. Format: instances/{instance}/databases/{database} |
| page_size | [int32](#int32) |  | Not used. The maximum number of secrets to return. The service may return fewer than this value. If unspecified, at most 10 secrets will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListSecrets` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListSecrets` must match the call that provided the page token. |






<a name="bytebase-api-v1alpha-ListSecretsResponse"></a>

### ListSecretsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secrets | [Secret](#bytebase-api-v1alpha-Secret) | repeated | The list of secrets. |
| next_page_token | [string](#string) |  | Not used. A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-MaterializedViewMetadata"></a>

### MaterializedViewMetadata
MaterializedViewMetadata is the metadata for materialized views.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a materialized view. |
| definition | [string](#string) |  | The definition is the definition of a materialized view. |
| comment | [string](#string) |  | The comment is the comment of a materialized view. |
| dependency_columns | [DependencyColumn](#bytebase-api-v1alpha-DependencyColumn) | repeated | The dependency_columns is the list of dependency columns of a materialized view. |
| triggers | [TriggerMetadata](#bytebase-api-v1alpha-TriggerMetadata) | repeated | The columns is the ordered list of columns in a table. |
| indexes | [IndexMetadata](#bytebase-api-v1alpha-IndexMetadata) | repeated | The indexes is the list of indexes in a table. |
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-api-v1alpha-PackageMetadata"></a>

### PackageMetadata
PackageMetadata is the metadata for packages.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a package. |
| definition | [string](#string) |  | The definition is the definition of a package. |






<a name="bytebase-api-v1alpha-ProcedureMetadata"></a>

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
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-api-v1alpha-Revision"></a>

### Revision



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: instances/{instance}/databases/{database}/revisions/{revision} |
| release | [string](#string) |  | Format: projects/{project}/releases/{release} Can be empty. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| deleter | [string](#string) |  | Format: users/hello@world.com Can be empty. |
| delete_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Can be empty. |
| file | [string](#string) |  | Format: projects/{project}/releases/{release}/files/{id} Can be empty. |
| version | [string](#string) |  |  |
| sheet | [string](#string) |  | The sheet that holds the content. Format: projects/{project}/sheets/{sheet} |
| sheet_sha256 | [string](#string) |  | The SHA256 hash value of the sheet. |
| statement | [string](#string) |  | The statement is used for preview purpose. |
| statement_size | [int64](#int64) |  |  |
| issue | [string](#string) |  | The issue associated with the revision. Can be empty. Format: projects/{project}/issues/{issue} |
| task_run | [string](#string) |  | The task run associated with the revision. Can be empty. Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |






<a name="bytebase-api-v1alpha-SchemaMetadata"></a>

### SchemaMetadata
SchemaMetadata is the metadata for schemas.
This is the concept of schema in Postgres, but it&#39;s a no-op for MySQL.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the schema name. It is an empty string for databases without such concept such as MySQL. |
| tables | [TableMetadata](#bytebase-api-v1alpha-TableMetadata) | repeated | The tables is the list of tables in a schema. |
| external_tables | [ExternalTableMetadata](#bytebase-api-v1alpha-ExternalTableMetadata) | repeated | The external_tables is the list of external tables in a schema. |
| views | [ViewMetadata](#bytebase-api-v1alpha-ViewMetadata) | repeated | The views is the list of views in a schema. |
| functions | [FunctionMetadata](#bytebase-api-v1alpha-FunctionMetadata) | repeated | The functions is the list of functions in a schema. |
| procedures | [ProcedureMetadata](#bytebase-api-v1alpha-ProcedureMetadata) | repeated | The procedures is the list of procedures in a schema. |
| streams | [StreamMetadata](#bytebase-api-v1alpha-StreamMetadata) | repeated | The streams is the list of streams in a schema, currently, only used for Snowflake. |
| tasks | [TaskMetadata](#bytebase-api-v1alpha-TaskMetadata) | repeated | The routines is the list of routines in a schema, currently, only used for Snowflake. |
| materialized_views | [MaterializedViewMetadata](#bytebase-api-v1alpha-MaterializedViewMetadata) | repeated | The materialized_views is the list of materialized views in a schema. |
| packages | [PackageMetadata](#bytebase-api-v1alpha-PackageMetadata) | repeated | The packages is the list of packages in a schema. |
| owner | [string](#string) |  |  |
| sequences | [SequenceMetadata](#bytebase-api-v1alpha-SequenceMetadata) | repeated | The sequences is the list of sequences in a schema, sorted by name. |
| events | [EventMetadata](#bytebase-api-v1alpha-EventMetadata) | repeated |  |
| enum_types | [EnumTypeMetadata](#bytebase-api-v1alpha-EnumTypeMetadata) | repeated |  |
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-api-v1alpha-Secret"></a>

### Secret
Secret is the secret of the database now.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | name is the unique name of the secret, which is specified by the client. Format: instances/{instance}/databases/{database}/secrets/{secret} |
| created_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Not used. The timestamp when the secret resource was created initially. |
| updated_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Not used. The timestamp when the secret resource was updated. |
| value | [string](#string) |  | The value of the secret. |
| description | [string](#string) |  | The description of the secret. |






<a name="bytebase-api-v1alpha-SequenceMetadata"></a>

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
| owner_table | [string](#string) |  | The owner table of the sequence. |
| owner_column | [string](#string) |  | The owner column of the sequence. |
| comment | [string](#string) |  |  |
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-api-v1alpha-StreamMetadata"></a>

### StreamMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a stream. |
| table_name | [string](#string) |  | The table_name is the name of the table/view that the stream is created on. |
| owner | [string](#string) |  | The owner of the stream. |
| comment | [string](#string) |  | The comment of the stream. |
| type | [StreamMetadata.Type](#bytebase-api-v1alpha-StreamMetadata-Type) |  | The type of the stream. |
| stale | [bool](#bool) |  | Indicates whether the stream was last read before the `stale_after` time. |
| mode | [StreamMetadata.Mode](#bytebase-api-v1alpha-StreamMetadata-Mode) |  | The mode of the stream. |
| definition | [string](#string) |  | The definition of the stream. |






<a name="bytebase-api-v1alpha-SyncDatabaseRequest"></a>

### SyncDatabaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to sync. Format: instances/{instance}/databases/{database} |






<a name="bytebase-api-v1alpha-SyncDatabaseResponse"></a>

### SyncDatabaseResponse







<a name="bytebase-api-v1alpha-TableMetadata"></a>

### TableMetadata
TableMetadata is the metadata for tables.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a table. |
| columns | [ColumnMetadata](#bytebase-api-v1alpha-ColumnMetadata) | repeated | The columns is the ordered list of columns in a table. |
| indexes | [IndexMetadata](#bytebase-api-v1alpha-IndexMetadata) | repeated | The indexes is the list of indexes in a table. |
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
| foreign_keys | [ForeignKeyMetadata](#bytebase-api-v1alpha-ForeignKeyMetadata) | repeated | The foreign_keys is the list of foreign keys in a table. |
| partitions | [TablePartitionMetadata](#bytebase-api-v1alpha-TablePartitionMetadata) | repeated | The partitions is the list of partitions in a table. |
| check_constraints | [CheckConstraintMetadata](#bytebase-api-v1alpha-CheckConstraintMetadata) | repeated | The check_constraints is the list of check constraints in a table. |
| owner | [string](#string) |  |  |
| sorting_keys | [string](#string) | repeated | The sorting_keys is a tuple of column names or arbitrary expressions. ClickHouse specific field. Reference: https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/mergetree#order_by |
| triggers | [TriggerMetadata](#bytebase-api-v1alpha-TriggerMetadata) | repeated |  |
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-api-v1alpha-TablePartitionMetadata"></a>

### TablePartitionMetadata
TablePartitionMetadata is the metadata for table partitions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a table partition. |
| type | [TablePartitionMetadata.Type](#bytebase-api-v1alpha-TablePartitionMetadata-Type) |  | The type of a table partition. |
| expression | [string](#string) |  | The expression is the expression of a table partition. For PostgreSQL, the expression is the text of {FOR VALUES partition_bound_spec}, see https://www.postgresql.org/docs/current/sql-createtable.html. For MySQL, the expression is the `expr` or `column_list` of the following syntax. PARTITION BY { [LINEAR] HASH(expr) | [LINEAR] KEY [ALGORITHM={1 | 2}] (column_list) | RANGE{(expr) | COLUMNS(column_list)} | LIST{(expr) | COLUMNS(column_list)} }. |
| value | [string](#string) |  | The value is the value of a table partition. For MySQL, the value is for RANGE and LIST partition types, - For a RANGE partition, it contains the value set in the partition&#39;s VALUES LESS THAN clause, which can be either an integer or MAXVALUE. - For a LIST partition, this column contains the values defined in the partition&#39;s VALUES IN clause, which is a list of comma-separated integer values. - For others, it&#39;s an empty string. |
| use_default | [string](#string) |  | The use_default is whether the users use the default partition, it stores the different value for different database engines. For MySQL, it&#39;s [INT] type, 0 means not use default partition, otherwise, it&#39;s equals to number in syntax [SUB]PARTITION {number}. |
| subpartitions | [TablePartitionMetadata](#bytebase-api-v1alpha-TablePartitionMetadata) | repeated | The subpartitions is the list of subpartitions in a table partition. |
| indexes | [IndexMetadata](#bytebase-api-v1alpha-IndexMetadata) | repeated |  |






<a name="bytebase-api-v1alpha-TaskMetadata"></a>

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
| state | [TaskMetadata.State](#bytebase-api-v1alpha-TaskMetadata-State) |  | The state of the task. |
| condition | [string](#string) |  | The condition of the task. |
| definition | [string](#string) |  | The definition of the task. |






<a name="bytebase-api-v1alpha-TriggerMetadata"></a>

### TriggerMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of the trigger. |
| event | [string](#string) |  | The event is the event of the trigger, such as INSERT, UPDATE, DELETE, TRUNCATE. |
| timing | [string](#string) |  | The timing is the timing of the trigger, such as BEFORE, AFTER. |
| body | [string](#string) |  | The body is the body of the trigger. |
| sql_mode | [string](#string) |  |  |
| character_set_client | [string](#string) |  |  |
| collation_connection | [string](#string) |  |  |
| comment | [string](#string) |  |  |
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-api-v1alpha-UpdateDatabaseRequest"></a>

### UpdateDatabaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [Database](#bytebase-api-v1alpha-Database) |  | The database to update.

The database&#39;s `name` field is used to identify the database to update. Format: instances/{instance}/databases/{database} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






<a name="bytebase-api-v1alpha-UpdateSecretRequest"></a>

### UpdateSecretRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secret | [Secret](#bytebase-api-v1alpha-Secret) |  | The secret to be created or updated. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The mask of the fields to be updated. |
| allow_missing | [bool](#bool) |  | If true, the secret will be created if it does not exist. |






<a name="bytebase-api-v1alpha-ViewMetadata"></a>

### ViewMetadata
ViewMetadata is the metadata for views.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a view. |
| definition | [string](#string) |  | The definition is the definition of a view. |
| comment | [string](#string) |  | The comment is the comment of a view. |
| dependency_columns | [DependencyColumn](#bytebase-api-v1alpha-DependencyColumn) | repeated | The dependency_columns is the list of dependency columns of a view. |
| columns | [ColumnMetadata](#bytebase-api-v1alpha-ColumnMetadata) | repeated | The columns is the ordered list of columns in a table. |
| triggers | [TriggerMetadata](#bytebase-api-v1alpha-TriggerMetadata) | repeated | The triggers is the list of triggers in a view. |
| skip_dump | [bool](#bool) |  |  |





 


<a name="bytebase-api-v1alpha-Changelog-Status"></a>

### Changelog.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| DONE | 2 |  |
| FAILED | 3 |  |



<a name="bytebase-api-v1alpha-Changelog-Type"></a>

### Changelog.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| BASELINE | 1 |  |
| MIGRATE | 2 |  |
| MIGRATE_SDL | 3 |  |
| MIGRATE_GHOST | 4 |  |
| DATA | 6 |  |



<a name="bytebase-api-v1alpha-ChangelogView"></a>

### ChangelogView


| Name | Number | Description |
| ---- | ------ | ----------- |
| CHANGELOG_VIEW_UNSPECIFIED | 0 | The default / unset value. The API will default to the BASIC view. |
| CHANGELOG_VIEW_BASIC | 1 |  |
| CHANGELOG_VIEW_FULL | 2 |  |



<a name="bytebase-api-v1alpha-ColumnMetadata-IdentityGeneration"></a>

### ColumnMetadata.IdentityGeneration


| Name | Number | Description |
| ---- | ------ | ----------- |
| IDENTITY_GENERATION_UNSPECIFIED | 0 |  |
| ALWAYS | 1 |  |
| BY_DEFAULT | 2 |  |



<a name="bytebase-api-v1alpha-GenerationMetadata-Type"></a>

### GenerationMetadata.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| TYPE_VIRTUAL | 1 |  |
| TYPE_STORED | 2 |  |



<a name="bytebase-api-v1alpha-GetSchemaStringRequest-ObjectType"></a>

### GetSchemaStringRequest.ObjectType


| Name | Number | Description |
| ---- | ------ | ----------- |
| OBJECT_TYPE_UNSPECIFIED | 0 |  |
| DATABASE | 1 |  |
| SCHEMA | 2 |  |
| TABLE | 3 |  |
| VIEW | 4 |  |
| MATERIALIZED_VIEW | 5 |  |
| FUNCTION | 6 |  |
| PROCEDURE | 7 |  |
| SEQUENCE | 8 |  |



<a name="bytebase-api-v1alpha-StreamMetadata-Mode"></a>

### StreamMetadata.Mode


| Name | Number | Description |
| ---- | ------ | ----------- |
| MODE_UNSPECIFIED | 0 |  |
| MODE_DEFAULT | 1 |  |
| MODE_APPEND_ONLY | 2 |  |
| MODE_INSERT_ONLY | 3 |  |



<a name="bytebase-api-v1alpha-StreamMetadata-Type"></a>

### StreamMetadata.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| TYPE_DELTA | 1 |  |



<a name="bytebase-api-v1alpha-TablePartitionMetadata-Type"></a>

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



<a name="bytebase-api-v1alpha-TaskMetadata-State"></a>

### TaskMetadata.State


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATE_UNSPECIFIED | 0 |  |
| STATE_STARTED | 1 |  |
| STATE_SUSPENDED | 2 |  |


 

 


<a name="bytebase-api-v1alpha-DatabaseService"></a>

### DatabaseService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetDatabase | [GetDatabaseRequest](#bytebase-api-v1alpha-GetDatabaseRequest) | [Database](#bytebase-api-v1alpha-Database) |  |
| BatchGetDatabases | [BatchGetDatabasesRequest](#bytebase-api-v1alpha-BatchGetDatabasesRequest) | [BatchGetDatabasesResponse](#bytebase-api-v1alpha-BatchGetDatabasesResponse) |  |
| ListDatabases | [ListDatabasesRequest](#bytebase-api-v1alpha-ListDatabasesRequest) | [ListDatabasesResponse](#bytebase-api-v1alpha-ListDatabasesResponse) |  |
| UpdateDatabase | [UpdateDatabaseRequest](#bytebase-api-v1alpha-UpdateDatabaseRequest) | [Database](#bytebase-api-v1alpha-Database) |  |
| BatchUpdateDatabases | [BatchUpdateDatabasesRequest](#bytebase-api-v1alpha-BatchUpdateDatabasesRequest) | [BatchUpdateDatabasesResponse](#bytebase-api-v1alpha-BatchUpdateDatabasesResponse) |  |
| SyncDatabase | [SyncDatabaseRequest](#bytebase-api-v1alpha-SyncDatabaseRequest) | [SyncDatabaseResponse](#bytebase-api-v1alpha-SyncDatabaseResponse) |  |
| BatchSyncDatabases | [BatchSyncDatabasesRequest](#bytebase-api-v1alpha-BatchSyncDatabasesRequest) | [BatchSyncDatabasesResponse](#bytebase-api-v1alpha-BatchSyncDatabasesResponse) |  |
| GetDatabaseMetadata | [GetDatabaseMetadataRequest](#bytebase-api-v1alpha-GetDatabaseMetadataRequest) | [DatabaseMetadata](#bytebase-api-v1alpha-DatabaseMetadata) |  |
| GetDatabaseSchema | [GetDatabaseSchemaRequest](#bytebase-api-v1alpha-GetDatabaseSchemaRequest) | [DatabaseSchema](#bytebase-api-v1alpha-DatabaseSchema) |  |
| DiffSchema | [DiffSchemaRequest](#bytebase-api-v1alpha-DiffSchemaRequest) | [DiffSchemaResponse](#bytebase-api-v1alpha-DiffSchemaResponse) |  |
| ListSecrets | [ListSecretsRequest](#bytebase-api-v1alpha-ListSecretsRequest) | [ListSecretsResponse](#bytebase-api-v1alpha-ListSecretsResponse) |  |
| UpdateSecret | [UpdateSecretRequest](#bytebase-api-v1alpha-UpdateSecretRequest) | [Secret](#bytebase-api-v1alpha-Secret) |  |
| DeleteSecret | [DeleteSecretRequest](#bytebase-api-v1alpha-DeleteSecretRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| ListRevisions | [ListRevisionsRequest](#bytebase-api-v1alpha-ListRevisionsRequest) | [ListRevisionsResponse](#bytebase-api-v1alpha-ListRevisionsResponse) |  |
| GetRevision | [GetRevisionRequest](#bytebase-api-v1alpha-GetRevisionRequest) | [Revision](#bytebase-api-v1alpha-Revision) |  |
| CreateRevision | [CreateRevisionRequest](#bytebase-api-v1alpha-CreateRevisionRequest) | [Revision](#bytebase-api-v1alpha-Revision) |  |
| DeleteRevision | [DeleteRevisionRequest](#bytebase-api-v1alpha-DeleteRevisionRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| ListChangelogs | [ListChangelogsRequest](#bytebase-api-v1alpha-ListChangelogsRequest) | [ListChangelogsResponse](#bytebase-api-v1alpha-ListChangelogsResponse) |  |
| GetChangelog | [GetChangelogRequest](#bytebase-api-v1alpha-GetChangelogRequest) | [Changelog](#bytebase-api-v1alpha-Changelog) |  |
| GetSchemaString | [GetSchemaStringRequest](#bytebase-api-v1alpha-GetSchemaStringRequest) | [GetSchemaStringResponse](#bytebase-api-v1alpha-GetSchemaStringResponse) |  |

 



<a name="api_v1alpha_issue_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/issue_service.proto



<a name="bytebase-api-v1alpha-ApprovalFlow"></a>

### ApprovalFlow



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| steps | [ApprovalStep](#bytebase-api-v1alpha-ApprovalStep) | repeated |  |






<a name="bytebase-api-v1alpha-ApprovalNode"></a>

### ApprovalNode



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [ApprovalNode.Type](#bytebase-api-v1alpha-ApprovalNode-Type) |  |  |
| role | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-ApprovalStep"></a>

### ApprovalStep



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [ApprovalStep.Type](#bytebase-api-v1alpha-ApprovalStep-Type) |  |  |
| nodes | [ApprovalNode](#bytebase-api-v1alpha-ApprovalNode) | repeated |  |






<a name="bytebase-api-v1alpha-ApprovalTemplate"></a>

### ApprovalTemplate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| flow | [ApprovalFlow](#bytebase-api-v1alpha-ApprovalFlow) |  |  |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| creator | [string](#string) |  | The name of the creator in users/{email} format. TODO: we should mark it as OUTPUT_ONLY, but currently the frontend will post the approval setting with creator. |






<a name="bytebase-api-v1alpha-ApproveIssueRequest"></a>

### ApproveIssueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the issue to add an approver. Format: projects/{project}/issues/{issue} |
| comment | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-BatchUpdateIssuesStatusRequest"></a>

### BatchUpdateIssuesStatusRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource shared by all issues being updated. Format: projects/{project} If the operation spans parents, a dash (-) may be accepted as a wildcard. We only support updating the status of databases for now. |
| issues | [string](#string) | repeated | The list of issues to update. Format: projects/{project}/issues/{issue} |
| status | [IssueStatus](#bytebase-api-v1alpha-IssueStatus) |  | The new status. |
| reason | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-BatchUpdateIssuesStatusResponse"></a>

### BatchUpdateIssuesStatusResponse







<a name="bytebase-api-v1alpha-CreateIssueCommentRequest"></a>

### CreateIssueCommentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The issue name Format: projects/{project}/issues/{issue} |
| issue_comment | [IssueComment](#bytebase-api-v1alpha-IssueComment) |  |  |






<a name="bytebase-api-v1alpha-CreateIssueRequest"></a>

### CreateIssueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of issues. Format: projects/{project} |
| issue | [Issue](#bytebase-api-v1alpha-Issue) |  | The issue to create. |






<a name="bytebase-api-v1alpha-GetIssueRequest"></a>

### GetIssueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the issue to retrieve. Format: projects/{project}/issues/{issue} |
| force | [bool](#bool) |  |  |






<a name="bytebase-api-v1alpha-GrantRequest"></a>

### GrantRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [string](#string) |  | The requested role. Format: roles/EXPORTER. |
| user | [string](#string) |  | The user to be granted. Format: users/{email}. |
| condition | [google.type.Expr](#google-type-Expr) |  |  |
| expiration | [google.protobuf.Duration](#google-protobuf-Duration) |  |  |






<a name="bytebase-api-v1alpha-Issue"></a>

### Issue



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the issue. Format: projects/{project}/issues/{issue} |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| type | [Issue.Type](#bytebase-api-v1alpha-Issue-Type) |  |  |
| status | [IssueStatus](#bytebase-api-v1alpha-IssueStatus) |  |  |
| approvers | [Issue.Approver](#bytebase-api-v1alpha-Issue-Approver) | repeated |  |
| approval_templates | [ApprovalTemplate](#bytebase-api-v1alpha-ApprovalTemplate) | repeated |  |
| approval_finding_done | [bool](#bool) |  | If the value is `false`, it means that the backend is still finding matching approval templates. If `true`, approval_templates &amp; approvers &amp; approval_finding_error are available. |
| approval_finding_error | [string](#string) |  |  |
| subscribers | [string](#string) | repeated | The subscribers. Format: users/hello@world.com |
| creator | [string](#string) |  | Format: users/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| plan | [string](#string) |  | The plan associated with the issue. Can be empty. Format: projects/{project}/plans/{plan} |
| rollout | [string](#string) |  | The rollout associated with the issue. Can be empty. Format: projects/{project}/rollouts/{rollout} |
| grant_request | [GrantRequest](#bytebase-api-v1alpha-GrantRequest) |  | Used if the issue type is GRANT_REQUEST. |
| releasers | [string](#string) | repeated | The releasers of the pending stage of the issue rollout, judging from the rollout policy. Format: - roles/workspaceOwner - roles/workspaceDBA - roles/projectOwner - roles/projectReleaser - users/{email} |
| risk_level | [Issue.RiskLevel](#bytebase-api-v1alpha-Issue-RiskLevel) |  |  |
| task_status_count | [Issue.TaskStatusCountEntry](#bytebase-api-v1alpha-Issue-TaskStatusCountEntry) | repeated | The status count of the issue. Keys are the following: - NOT_STARTED - SKIPPED - PENDING - RUNNING - DONE - FAILED - CANCELED |
| labels | [string](#string) | repeated |  |






<a name="bytebase-api-v1alpha-Issue-Approver"></a>

### Issue.Approver



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [Issue.Approver.Status](#bytebase-api-v1alpha-Issue-Approver-Status) |  | The new status. |
| principal | [string](#string) |  | Format: users/hello@world.com |






<a name="bytebase-api-v1alpha-Issue-TaskStatusCountEntry"></a>

### Issue.TaskStatusCountEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [int32](#int32) |  |  |






<a name="bytebase-api-v1alpha-IssueComment"></a>

### IssueComment



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/issues/{issue}/issueComments/{issueComment-uid} |
| comment | [string](#string) |  |  |
| payload | [string](#string) |  | TODO: use struct message instead. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| creator | [string](#string) |  | Format: users/{email} |
| approval | [IssueComment.Approval](#bytebase-api-v1alpha-IssueComment-Approval) |  |  |
| issue_update | [IssueComment.IssueUpdate](#bytebase-api-v1alpha-IssueComment-IssueUpdate) |  |  |
| stage_end | [IssueComment.StageEnd](#bytebase-api-v1alpha-IssueComment-StageEnd) |  |  |
| task_update | [IssueComment.TaskUpdate](#bytebase-api-v1alpha-IssueComment-TaskUpdate) |  |  |
| task_prior_backup | [IssueComment.TaskPriorBackup](#bytebase-api-v1alpha-IssueComment-TaskPriorBackup) |  |  |






<a name="bytebase-api-v1alpha-IssueComment-Approval"></a>

### IssueComment.Approval



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [IssueComment.Approval.Status](#bytebase-api-v1alpha-IssueComment-Approval-Status) |  |  |






<a name="bytebase-api-v1alpha-IssueComment-IssueUpdate"></a>

### IssueComment.IssueUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| from_title | [string](#string) | optional |  |
| to_title | [string](#string) | optional |  |
| from_description | [string](#string) | optional |  |
| to_description | [string](#string) | optional |  |
| from_status | [IssueStatus](#bytebase-api-v1alpha-IssueStatus) | optional |  |
| to_status | [IssueStatus](#bytebase-api-v1alpha-IssueStatus) | optional |  |
| from_labels | [string](#string) | repeated |  |
| to_labels | [string](#string) | repeated |  |






<a name="bytebase-api-v1alpha-IssueComment-StageEnd"></a>

### IssueComment.StageEnd



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| stage | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-IssueComment-TaskPriorBackup"></a>

### IssueComment.TaskPriorBackup



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| task | [string](#string) |  |  |
| tables | [IssueComment.TaskPriorBackup.Table](#bytebase-api-v1alpha-IssueComment-TaskPriorBackup-Table) | repeated |  |
| original_line | [int32](#int32) | optional |  |
| database | [string](#string) |  |  |
| error | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-IssueComment-TaskPriorBackup-Table"></a>

### IssueComment.TaskPriorBackup.Table



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  |  |
| table | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-IssueComment-TaskUpdate"></a>

### IssueComment.TaskUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tasks | [string](#string) | repeated |  |
| from_sheet | [string](#string) | optional | Format: projects/{project}/sheets/{sheet} |
| to_sheet | [string](#string) | optional | Format: projects/{project}/sheets/{sheet} |
| from_earliest_allowed_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional |  |
| to_earliest_allowed_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional |  |
| to_status | [IssueComment.TaskUpdate.Status](#bytebase-api-v1alpha-IssueComment-TaskUpdate-Status) | optional |  |






<a name="bytebase-api-v1alpha-ListIssueCommentsRequest"></a>

### ListIssueCommentsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{projects}/issues/{issue} |
| page_size | [int32](#int32) |  | The maximum number of issue comments to return. The service may return fewer than this value. If unspecified, at most 10 issue comments will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListIssueComments` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListIssueComments` must match the call that provided the page token. |






<a name="bytebase-api-v1alpha-ListIssueCommentsResponse"></a>

### ListIssueCommentsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issue_comments | [IssueComment](#bytebase-api-v1alpha-IssueComment) | repeated |  |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-ListIssuesRequest"></a>

### ListIssuesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of issues. Format: projects/{project} |
| page_size | [int32](#int32) |  | The maximum number of issues to return. The service may return fewer than this value. If unspecified, at most 10 issues will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListIssues` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListIssues` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter issues returned in the list. |
| query | [string](#string) |  | Query is the query statement. |






<a name="bytebase-api-v1alpha-ListIssuesResponse"></a>

### ListIssuesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issues | [Issue](#bytebase-api-v1alpha-Issue) | repeated | The issues from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-RejectIssueRequest"></a>

### RejectIssueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the issue to add an rejection. Format: projects/{project}/issues/{issue} |
| comment | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-RequestIssueRequest"></a>

### RequestIssueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the issue to request a issue. Format: projects/{project}/issues/{issue} |
| comment | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-SearchIssuesRequest"></a>

### SearchIssuesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of issues. Format: projects/{project} Use &#34;projects/-&#34; to list all issues from all projects. |
| page_size | [int32](#int32) |  | The maximum number of issues to return. The service may return fewer than this value. If unspecified, at most 10 issues will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `SearchIssues` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `SearchIssues` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter issues returned in the list. |
| query | [string](#string) |  | Query is the query statement. |






<a name="bytebase-api-v1alpha-SearchIssuesResponse"></a>

### SearchIssuesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issues | [Issue](#bytebase-api-v1alpha-Issue) | repeated | The issues from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-UpdateIssueCommentRequest"></a>

### UpdateIssueCommentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The issue name Format: projects/{project}/issues/{issue} |
| issue_comment | [IssueComment](#bytebase-api-v1alpha-IssueComment) |  |  |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






<a name="bytebase-api-v1alpha-UpdateIssueRequest"></a>

### UpdateIssueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issue | [Issue](#bytebase-api-v1alpha-Issue) |  | The issue to update.

The issue&#39;s `name` field is used to identify the issue to update. Format: projects/{project}/issues/{issue} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 


<a name="bytebase-api-v1alpha-ApprovalNode-Type"></a>

### ApprovalNode.Type
Type of the ApprovalNode.
type determines who should approve this node.
ANY_IN_GROUP means the ApprovalNode can be approved by an user from our predefined user group.
See GroupValue below for the predefined user groups.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| ANY_IN_GROUP | 1 |  |



<a name="bytebase-api-v1alpha-ApprovalStep-Type"></a>

### ApprovalStep.Type
Type of the ApprovalStep
ALL means every node must be approved to proceed.
ANY means approving any node will proceed.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| ALL | 1 |  |
| ANY | 2 |  |



<a name="bytebase-api-v1alpha-Issue-Approver-Status"></a>

### Issue.Approver.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| APPROVED | 2 |  |
| REJECTED | 3 |  |



<a name="bytebase-api-v1alpha-Issue-RiskLevel"></a>

### Issue.RiskLevel


| Name | Number | Description |
| ---- | ------ | ----------- |
| RISK_LEVEL_UNSPECIFIED | 0 |  |
| LOW | 1 |  |
| MODERATE | 2 |  |
| HIGH | 3 |  |



<a name="bytebase-api-v1alpha-Issue-Type"></a>

### Issue.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| DATABASE_CHANGE | 1 |  |
| GRANT_REQUEST | 2 |  |
| DATABASE_DATA_EXPORT | 3 |  |



<a name="bytebase-api-v1alpha-IssueComment-Approval-Status"></a>

### IssueComment.Approval.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| APPROVED | 2 |  |
| REJECTED | 3 |  |



<a name="bytebase-api-v1alpha-IssueComment-TaskUpdate-Status"></a>

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



<a name="bytebase-api-v1alpha-IssueStatus"></a>

### IssueStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| ISSUE_STATUS_UNSPECIFIED | 0 |  |
| OPEN | 1 |  |
| DONE | 2 |  |
| CANCELED | 3 |  |


 

 


<a name="bytebase-api-v1alpha-IssueService"></a>

### IssueService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetIssue | [GetIssueRequest](#bytebase-api-v1alpha-GetIssueRequest) | [Issue](#bytebase-api-v1alpha-Issue) |  |
| CreateIssue | [CreateIssueRequest](#bytebase-api-v1alpha-CreateIssueRequest) | [Issue](#bytebase-api-v1alpha-Issue) |  |
| ListIssues | [ListIssuesRequest](#bytebase-api-v1alpha-ListIssuesRequest) | [ListIssuesResponse](#bytebase-api-v1alpha-ListIssuesResponse) |  |
| SearchIssues | [SearchIssuesRequest](#bytebase-api-v1alpha-SearchIssuesRequest) | [SearchIssuesResponse](#bytebase-api-v1alpha-SearchIssuesResponse) | Search for issues that the caller has the bb.issues.get permission on and also satisfy the specified filter &amp; query. |
| UpdateIssue | [UpdateIssueRequest](#bytebase-api-v1alpha-UpdateIssueRequest) | [Issue](#bytebase-api-v1alpha-Issue) |  |
| ListIssueComments | [ListIssueCommentsRequest](#bytebase-api-v1alpha-ListIssueCommentsRequest) | [ListIssueCommentsResponse](#bytebase-api-v1alpha-ListIssueCommentsResponse) |  |
| CreateIssueComment | [CreateIssueCommentRequest](#bytebase-api-v1alpha-CreateIssueCommentRequest) | [IssueComment](#bytebase-api-v1alpha-IssueComment) |  |
| UpdateIssueComment | [UpdateIssueCommentRequest](#bytebase-api-v1alpha-UpdateIssueCommentRequest) | [IssueComment](#bytebase-api-v1alpha-IssueComment) |  |
| BatchUpdateIssuesStatus | [BatchUpdateIssuesStatusRequest](#bytebase-api-v1alpha-BatchUpdateIssuesStatusRequest) | [BatchUpdateIssuesStatusResponse](#bytebase-api-v1alpha-BatchUpdateIssuesStatusResponse) |  |
| ApproveIssue | [ApproveIssueRequest](#bytebase-api-v1alpha-ApproveIssueRequest) | [Issue](#bytebase-api-v1alpha-Issue) | ApproveIssue approves the issue. The access is based on approval flow. |
| RejectIssue | [RejectIssueRequest](#bytebase-api-v1alpha-RejectIssueRequest) | [Issue](#bytebase-api-v1alpha-Issue) | RejectIssue rejects the issue. The access is based on approval flow. |
| RequestIssue | [RequestIssueRequest](#bytebase-api-v1alpha-RequestIssueRequest) | [Issue](#bytebase-api-v1alpha-Issue) | RequestIssue requests the issue. The access is based on approval flow. |

 



<a name="api_v1alpha_subscription_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/subscription_service.proto



<a name="bytebase-api-v1alpha-Feature"></a>

### Feature



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name is the feature name. |
| matrix | [Feature.MatrixEntry](#bytebase-api-v1alpha-Feature-MatrixEntry) | repeated | Matrix is the feature matrix for different plan. The key is the plan enum in string value. |






<a name="bytebase-api-v1alpha-Feature-MatrixEntry"></a>

### Feature.MatrixEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [bool](#bool) |  |  |






<a name="bytebase-api-v1alpha-FeatureMatrix"></a>

### FeatureMatrix



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| features | [Feature](#bytebase-api-v1alpha-Feature) | repeated |  |






<a name="bytebase-api-v1alpha-GetFeatureMatrixRequest"></a>

### GetFeatureMatrixRequest







<a name="bytebase-api-v1alpha-GetSubscriptionRequest"></a>

### GetSubscriptionRequest







<a name="bytebase-api-v1alpha-PatchSubscription"></a>

### PatchSubscription



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| license | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-Subscription"></a>

### Subscription



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| seat_count | [int32](#int32) |  |  |
| instance_count | [int32](#int32) |  |  |
| expires_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| started_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| plan | [PlanType](#bytebase-api-v1alpha-PlanType) |  |  |
| trialing | [bool](#bool) |  |  |
| org_id | [string](#string) |  |  |
| org_name | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-UpdateSubscriptionRequest"></a>

### UpdateSubscriptionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| patch | [PatchSubscription](#bytebase-api-v1alpha-PatchSubscription) |  |  |





 


<a name="bytebase-api-v1alpha-PlanType"></a>

### PlanType


| Name | Number | Description |
| ---- | ------ | ----------- |
| PLAN_TYPE_UNSPECIFIED | 0 |  |
| FREE | 1 |  |
| TEAM | 2 |  |
| ENTERPRISE | 3 |  |


 

 


<a name="bytebase-api-v1alpha-SubscriptionService"></a>

### SubscriptionService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetSubscription | [GetSubscriptionRequest](#bytebase-api-v1alpha-GetSubscriptionRequest) | [Subscription](#bytebase-api-v1alpha-Subscription) |  |
| GetFeatureMatrix | [GetFeatureMatrixRequest](#bytebase-api-v1alpha-GetFeatureMatrixRequest) | [FeatureMatrix](#bytebase-api-v1alpha-FeatureMatrix) |  |
| UpdateSubscription | [UpdateSubscriptionRequest](#bytebase-api-v1alpha-UpdateSubscriptionRequest) | [Subscription](#bytebase-api-v1alpha-Subscription) |  |

 



<a name="api_v1alpha_setting_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/setting_service.proto



<a name="bytebase-api-v1alpha-AISetting"></a>

### AISetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| provider | [AISetting.Provider](#bytebase-api-v1alpha-AISetting-Provider) |  |  |
| endpoint | [string](#string) |  |  |
| api_key | [string](#string) |  |  |
| model | [string](#string) |  |  |
| version | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-AgentPluginSetting"></a>

### AgentPluginSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| url | [string](#string) |  | The URL for the agent API. |
| token | [string](#string) |  | The token for the agent. |






<a name="bytebase-api-v1alpha-Algorithm"></a>

### Algorithm



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| full_mask | [Algorithm.FullMask](#bytebase-api-v1alpha-Algorithm-FullMask) |  |  |
| range_mask | [Algorithm.RangeMask](#bytebase-api-v1alpha-Algorithm-RangeMask) |  |  |
| md5_mask | [Algorithm.MD5Mask](#bytebase-api-v1alpha-Algorithm-MD5Mask) |  |  |
| inner_outer_mask | [Algorithm.InnerOuterMask](#bytebase-api-v1alpha-Algorithm-InnerOuterMask) |  |  |






<a name="bytebase-api-v1alpha-Algorithm-FullMask"></a>

### Algorithm.FullMask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| substitution | [string](#string) |  | substitution is the string used to replace the original value, the max length of the string is 16 bytes. |






<a name="bytebase-api-v1alpha-Algorithm-InnerOuterMask"></a>

### Algorithm.InnerOuterMask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| prefix_len | [int32](#int32) |  |  |
| suffix_len | [int32](#int32) |  |  |
| type | [Algorithm.InnerOuterMask.MaskType](#bytebase-api-v1alpha-Algorithm-InnerOuterMask-MaskType) |  |  |
| substitution | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-Algorithm-MD5Mask"></a>

### Algorithm.MD5Mask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| salt | [string](#string) |  | salt is the salt value to generate a different hash that with the word alone. |






<a name="bytebase-api-v1alpha-Algorithm-RangeMask"></a>

### Algorithm.RangeMask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| slices | [Algorithm.RangeMask.Slice](#bytebase-api-v1alpha-Algorithm-RangeMask-Slice) | repeated | We store it as a repeated field to face the fact that the original value may have multiple parts should be masked. But frontend can be started with a single rule easily. |






<a name="bytebase-api-v1alpha-Algorithm-RangeMask-Slice"></a>

### Algorithm.RangeMask.Slice



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start | [int32](#int32) |  | start is the start index of the original value, start from 0 and should be less than stop. |
| end | [int32](#int32) |  | stop is the stop index of the original value, should be less than the length of the original value. |
| substitution | [string](#string) |  | substitution is the string used to replace the OriginalValue[start:end). |






<a name="bytebase-api-v1alpha-Announcement"></a>

### Announcement



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| level | [Announcement.AlertLevel](#bytebase-api-v1alpha-Announcement-AlertLevel) |  | The alert level of announcemnt |
| text | [string](#string) |  | The text of announcemnt |
| link | [string](#string) |  | The optional link, user can follow the link to check extra details |






<a name="bytebase-api-v1alpha-AppIMSetting"></a>

### AppIMSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| slack | [AppIMSetting.Slack](#bytebase-api-v1alpha-AppIMSetting-Slack) |  |  |
| feishu | [AppIMSetting.Feishu](#bytebase-api-v1alpha-AppIMSetting-Feishu) |  |  |
| wecom | [AppIMSetting.Wecom](#bytebase-api-v1alpha-AppIMSetting-Wecom) |  |  |
| lark | [AppIMSetting.Lark](#bytebase-api-v1alpha-AppIMSetting-Lark) |  |  |
| dingtalk | [AppIMSetting.DingTalk](#bytebase-api-v1alpha-AppIMSetting-DingTalk) |  |  |






<a name="bytebase-api-v1alpha-AppIMSetting-DingTalk"></a>

### AppIMSetting.DingTalk



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| client_id | [string](#string) |  |  |
| client_secret | [string](#string) |  |  |
| robot_code | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-AppIMSetting-Feishu"></a>

### AppIMSetting.Feishu



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| app_id | [string](#string) |  |  |
| app_secret | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-AppIMSetting-Lark"></a>

### AppIMSetting.Lark



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| app_id | [string](#string) |  |  |
| app_secret | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-AppIMSetting-Slack"></a>

### AppIMSetting.Slack



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| token | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-AppIMSetting-Wecom"></a>

### AppIMSetting.Wecom



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| corp_id | [string](#string) |  |  |
| agent_id | [string](#string) |  |  |
| secret | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-DataClassificationSetting"></a>

### DataClassificationSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| configs | [DataClassificationSetting.DataClassificationConfig](#bytebase-api-v1alpha-DataClassificationSetting-DataClassificationConfig) | repeated |  |






<a name="bytebase-api-v1alpha-DataClassificationSetting-DataClassificationConfig"></a>

### DataClassificationSetting.DataClassificationConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | id is the uuid for classification. Each project can chose one classification config. |
| title | [string](#string) |  |  |
| levels | [DataClassificationSetting.DataClassificationConfig.Level](#bytebase-api-v1alpha-DataClassificationSetting-DataClassificationConfig-Level) | repeated | levels is user defined level list for classification. The order for the level decides its priority. |
| classification | [DataClassificationSetting.DataClassificationConfig.ClassificationEntry](#bytebase-api-v1alpha-DataClassificationSetting-DataClassificationConfig-ClassificationEntry) | repeated | classification is the id - DataClassification map. The id should in [0-9]&#43;-[0-9]&#43;-[0-9]&#43; format. |
| classification_from_config | [bool](#bool) |  | If true, we will only store the classification in the config. Otherwise we will get the classification from table/column comment, and write back to the schema metadata. |






<a name="bytebase-api-v1alpha-DataClassificationSetting-DataClassificationConfig-ClassificationEntry"></a>

### DataClassificationSetting.DataClassificationConfig.ClassificationEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [DataClassificationSetting.DataClassificationConfig.DataClassification](#bytebase-api-v1alpha-DataClassificationSetting-DataClassificationConfig-DataClassification) |  |  |






<a name="bytebase-api-v1alpha-DataClassificationSetting-DataClassificationConfig-DataClassification"></a>

### DataClassificationSetting.DataClassificationConfig.DataClassification



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | id is the classification id in [0-9]&#43;-[0-9]&#43;-[0-9]&#43; format. |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| level_id | [string](#string) | optional |  |






<a name="bytebase-api-v1alpha-DataClassificationSetting-DataClassificationConfig-Level"></a>

### DataClassificationSetting.DataClassificationConfig.Level



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-EnvironmentSetting"></a>

### EnvironmentSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| environments | [EnvironmentSetting.Environment](#bytebase-api-v1alpha-EnvironmentSetting-Environment) | repeated |  |






<a name="bytebase-api-v1alpha-EnvironmentSetting-Environment"></a>

### EnvironmentSetting.Environment



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The resource name of the environment. Format: environments/{environment}. Output only. |
| id | [string](#string) |  | The resource id of the environment. This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |
| title | [string](#string) |  | The display name of the environment. |
| tags | [EnvironmentSetting.Environment.TagsEntry](#bytebase-api-v1alpha-EnvironmentSetting-Environment-TagsEntry) | repeated |  |
| color | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-EnvironmentSetting-Environment-TagsEntry"></a>

### EnvironmentSetting.Environment.TagsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-GetSettingRequest"></a>

### GetSettingRequest
The request message for getting a setting.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The resource name of the setting. |






<a name="bytebase-api-v1alpha-GetSettingResponse"></a>

### GetSettingResponse
The response message for getting a setting.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| setting | [Setting](#bytebase-api-v1alpha-Setting) |  |  |






<a name="bytebase-api-v1alpha-ListSettingsRequest"></a>

### ListSettingsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Not used. The maximum number of settings to return. The service may return fewer than this value. If unspecified, at most 10 settings will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListSettings` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListSettings` must match the call that provided the page token. |






<a name="bytebase-api-v1alpha-ListSettingsResponse"></a>

### ListSettingsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| settings | [Setting](#bytebase-api-v1alpha-Setting) | repeated | The settings from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-MaximumSQLResultSizeSetting"></a>

### MaximumSQLResultSizeSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| limit | [int64](#int64) |  | The limit is in bytes. The default value is 100MB, we will use the default value if the setting not exists, or the limit &lt;= 0. |






<a name="bytebase-api-v1alpha-PasswordRestrictionSetting"></a>

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






<a name="bytebase-api-v1alpha-SCIMSetting"></a>

### SCIMSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-SMTPMailDeliverySettingValue"></a>

### SMTPMailDeliverySettingValue



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| server | [string](#string) |  | The SMTP server address. |
| port | [int32](#int32) |  | The SMTP server port. |
| encryption | [SMTPMailDeliverySettingValue.Encryption](#bytebase-api-v1alpha-SMTPMailDeliverySettingValue-Encryption) |  | The SMTP server encryption. |
| ca | [string](#string) | optional | The CA, KEY, and CERT for the SMTP server. Not used. |
| key | [string](#string) | optional |  |
| cert | [string](#string) | optional |  |
| authentication | [SMTPMailDeliverySettingValue.Authentication](#bytebase-api-v1alpha-SMTPMailDeliverySettingValue-Authentication) |  |  |
| username | [string](#string) |  |  |
| password | [string](#string) | optional | If not specified, server will use the existed password. |
| from | [string](#string) |  | The sender email address. |
| to | [string](#string) |  | The recipient email address, used with validate_only to send test email. |






<a name="bytebase-api-v1alpha-SchemaTemplateSetting"></a>

### SchemaTemplateSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| field_templates | [SchemaTemplateSetting.FieldTemplate](#bytebase-api-v1alpha-SchemaTemplateSetting-FieldTemplate) | repeated |  |
| column_types | [SchemaTemplateSetting.ColumnType](#bytebase-api-v1alpha-SchemaTemplateSetting-ColumnType) | repeated |  |
| table_templates | [SchemaTemplateSetting.TableTemplate](#bytebase-api-v1alpha-SchemaTemplateSetting-TableTemplate) | repeated |  |






<a name="bytebase-api-v1alpha-SchemaTemplateSetting-ColumnType"></a>

### SchemaTemplateSetting.ColumnType



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| engine | [Engine](#bytebase-api-v1alpha-Engine) |  |  |
| enabled | [bool](#bool) |  |  |
| types | [string](#string) | repeated |  |






<a name="bytebase-api-v1alpha-SchemaTemplateSetting-FieldTemplate"></a>

### SchemaTemplateSetting.FieldTemplate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| engine | [Engine](#bytebase-api-v1alpha-Engine) |  |  |
| category | [string](#string) |  |  |
| column | [ColumnMetadata](#bytebase-api-v1alpha-ColumnMetadata) |  |  |
| catalog | [ColumnCatalog](#bytebase-api-v1alpha-ColumnCatalog) |  |  |






<a name="bytebase-api-v1alpha-SchemaTemplateSetting-TableTemplate"></a>

### SchemaTemplateSetting.TableTemplate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| engine | [Engine](#bytebase-api-v1alpha-Engine) |  |  |
| category | [string](#string) |  |  |
| table | [TableMetadata](#bytebase-api-v1alpha-TableMetadata) |  |  |
| catalog | [TableCatalog](#bytebase-api-v1alpha-TableCatalog) |  |  |






<a name="bytebase-api-v1alpha-SemanticTypeSetting"></a>

### SemanticTypeSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| types | [SemanticTypeSetting.SemanticType](#bytebase-api-v1alpha-SemanticTypeSetting-SemanticType) | repeated |  |






<a name="bytebase-api-v1alpha-SemanticTypeSetting-SemanticType"></a>

### SemanticTypeSetting.SemanticType



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | id is the uuid for semantic type. |
| title | [string](#string) |  | the title of the semantic type, it should not be empty. |
| description | [string](#string) |  | the description of the semantic type, it can be empty. |
| algorithm | [Algorithm](#bytebase-api-v1alpha-Algorithm) |  |  |






<a name="bytebase-api-v1alpha-Setting"></a>

### Setting
The schema of setting.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The resource name of the setting. Must be one of the following forms:

- `setting/{setting}` For example, &#34;settings/bb.branding.logo&#34; |
| value | [Value](#bytebase-api-v1alpha-Value) |  | The value of the setting. |






<a name="bytebase-api-v1alpha-UpdateSettingRequest"></a>

### UpdateSettingRequest
The request message for updating or creating a setting.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| setting | [Setting](#bytebase-api-v1alpha-Setting) |  | The setting to update. |
| validate_only | [bool](#bool) |  | validate_only is a flag to indicate whether to validate the setting value, server would not persist the setting value if it is true. |
| allow_missing | [bool](#bool) |  |  |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  |  |






<a name="bytebase-api-v1alpha-Value"></a>

### Value
The data in setting value.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| string_value | [string](#string) |  | Defines this value as being a string value. |
| smtp_mail_delivery_setting_value | [SMTPMailDeliverySettingValue](#bytebase-api-v1alpha-SMTPMailDeliverySettingValue) |  |  |
| app_im_setting_value | [AppIMSetting](#bytebase-api-v1alpha-AppIMSetting) |  |  |
| agent_plugin_setting_value | [AgentPluginSetting](#bytebase-api-v1alpha-AgentPluginSetting) |  |  |
| workspace_profile_setting_value | [WorkspaceProfileSetting](#bytebase-api-v1alpha-WorkspaceProfileSetting) |  |  |
| workspace_approval_setting_value | [WorkspaceApprovalSetting](#bytebase-api-v1alpha-WorkspaceApprovalSetting) |  |  |
| workspace_trial_setting_value | [WorkspaceTrialSetting](#bytebase-api-v1alpha-WorkspaceTrialSetting) |  |  |
| schema_template_setting_value | [SchemaTemplateSetting](#bytebase-api-v1alpha-SchemaTemplateSetting) |  |  |
| data_classification_setting_value | [DataClassificationSetting](#bytebase-api-v1alpha-DataClassificationSetting) |  |  |
| semantic_type_setting_value | [SemanticTypeSetting](#bytebase-api-v1alpha-SemanticTypeSetting) |  |  |
| maximum_sql_result_size_setting | [MaximumSQLResultSizeSetting](#bytebase-api-v1alpha-MaximumSQLResultSizeSetting) |  |  |
| scim_setting | [SCIMSetting](#bytebase-api-v1alpha-SCIMSetting) |  |  |
| password_restriction_setting | [PasswordRestrictionSetting](#bytebase-api-v1alpha-PasswordRestrictionSetting) |  |  |
| ai_setting | [AISetting](#bytebase-api-v1alpha-AISetting) |  |  |
| environment_setting | [EnvironmentSetting](#bytebase-api-v1alpha-EnvironmentSetting) |  |  |






<a name="bytebase-api-v1alpha-WorkspaceApprovalSetting"></a>

### WorkspaceApprovalSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rules | [WorkspaceApprovalSetting.Rule](#bytebase-api-v1alpha-WorkspaceApprovalSetting-Rule) | repeated |  |






<a name="bytebase-api-v1alpha-WorkspaceApprovalSetting-Rule"></a>

### WorkspaceApprovalSetting.Rule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| template | [ApprovalTemplate](#bytebase-api-v1alpha-ApprovalTemplate) |  |  |
| condition | [google.type.Expr](#google-type-Expr) |  |  |






<a name="bytebase-api-v1alpha-WorkspaceProfileSetting"></a>

### WorkspaceProfileSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| external_url | [string](#string) |  | The external URL is used for sso authentication callback. |
| disallow_signup | [bool](#bool) |  | Disallow self-service signup, users can only be invited by the owner. |
| require_2fa | [bool](#bool) |  | Require 2FA for all users. |
| outbound_ip_list | [string](#string) | repeated | outbound_ip_list is the outbound IP for Bytebase instance in SaaS mode. |
| token_duration | [google.protobuf.Duration](#google-protobuf-Duration) |  | The duration for token. |
| announcement | [Announcement](#bytebase-api-v1alpha-Announcement) |  | The setting of custom announcement |
| maximum_role_expiration | [google.protobuf.Duration](#google-protobuf-Duration) |  | The max duration for role expired. |
| domains | [string](#string) | repeated | The workspace domain, e.g. bytebase.com. |
| enforce_identity_domain | [bool](#bool) |  | Only user and group from the domains can be created and login. |
| database_change_mode | [DatabaseChangeMode](#bytebase-api-v1alpha-DatabaseChangeMode) |  | The workspace database change mode. |
| disallow_password_signin | [bool](#bool) |  | Whether to disallow password signin. (Except workspace admins) |






<a name="bytebase-api-v1alpha-WorkspaceTrialSetting"></a>

### WorkspaceTrialSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance_count | [int32](#int32) |  |  |
| expire_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| issued_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| subject | [string](#string) |  |  |
| org_name | [string](#string) |  |  |
| plan | [PlanType](#bytebase-api-v1alpha-PlanType) |  |  |





 


<a name="bytebase-api-v1alpha-AISetting-Provider"></a>

### AISetting.Provider


| Name | Number | Description |
| ---- | ------ | ----------- |
| PROVIDER_UNSPECIFIED | 0 |  |
| OPEN_AI | 1 |  |
| CLAUDE | 2 |  |
| GEMINI | 3 |  |
| AZURE_OPENAI | 4 |  |



<a name="bytebase-api-v1alpha-Algorithm-InnerOuterMask-MaskType"></a>

### Algorithm.InnerOuterMask.MaskType


| Name | Number | Description |
| ---- | ------ | ----------- |
| MASK_TYPE_UNSPECIFIED | 0 |  |
| INNER | 1 |  |
| OUTER | 2 |  |



<a name="bytebase-api-v1alpha-Announcement-AlertLevel"></a>

### Announcement.AlertLevel
We support three levels of AlertLevel: INFO, WARNING, and ERROR.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ALERT_LEVEL_UNSPECIFIED | 0 |  |
| ALERT_LEVEL_INFO | 1 |  |
| ALERT_LEVEL_WARNING | 2 |  |
| ALERT_LEVEL_CRITICAL | 3 |  |



<a name="bytebase-api-v1alpha-DatabaseChangeMode"></a>

### DatabaseChangeMode


| Name | Number | Description |
| ---- | ------ | ----------- |
| DATABASE_CHANGE_MODE_UNSPECIFIED | 0 |  |
| PIPELINE | 1 | A more advanced database change process, including custom approval workflows and other advanced features. Default to this mode. |
| EDITOR | 2 | A simple database change process in SQL editor. Users can execute SQL directly. |



<a name="bytebase-api-v1alpha-SMTPMailDeliverySettingValue-Authentication"></a>

### SMTPMailDeliverySettingValue.Authentication
We support four types of SMTP authentication: NONE, PLAIN, LOGIN, and CRAM-MD5.

| Name | Number | Description |
| ---- | ------ | ----------- |
| AUTHENTICATION_UNSPECIFIED | 0 |  |
| AUTHENTICATION_NONE | 1 |  |
| AUTHENTICATION_PLAIN | 2 |  |
| AUTHENTICATION_LOGIN | 3 |  |
| AUTHENTICATION_CRAM_MD5 | 4 |  |



<a name="bytebase-api-v1alpha-SMTPMailDeliverySettingValue-Encryption"></a>

### SMTPMailDeliverySettingValue.Encryption
We support three types of SMTP encryption: NONE, STARTTLS, and SSL/TLS.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ENCRYPTION_UNSPECIFIED | 0 |  |
| ENCRYPTION_NONE | 1 |  |
| ENCRYPTION_STARTTLS | 2 |  |
| ENCRYPTION_SSL_TLS | 3 |  |


 

 


<a name="bytebase-api-v1alpha-SettingService"></a>

### SettingService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListSettings | [ListSettingsRequest](#bytebase-api-v1alpha-ListSettingsRequest) | [ListSettingsResponse](#bytebase-api-v1alpha-ListSettingsResponse) |  |
| GetSetting | [GetSettingRequest](#bytebase-api-v1alpha-GetSettingRequest) | [Setting](#bytebase-api-v1alpha-Setting) |  |
| UpdateSetting | [UpdateSettingRequest](#bytebase-api-v1alpha-UpdateSettingRequest) | [Setting](#bytebase-api-v1alpha-Setting) |  |

 



<a name="api_v1alpha_user_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/user_service.proto



<a name="bytebase-api-v1alpha-CreateUserRequest"></a>

### CreateUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#bytebase-api-v1alpha-User) |  | The user to create. |






<a name="bytebase-api-v1alpha-DeleteUserRequest"></a>

### DeleteUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the user to delete. Format: users/{user} |






<a name="bytebase-api-v1alpha-GetUserRequest"></a>

### GetUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the user to retrieve. Format: users/{user uid or user email} |






<a name="bytebase-api-v1alpha-ListUsersRequest"></a>

### ListUsersRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of users to return. The service may return fewer than this value. If unspecified, at most 10 users will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListUsers` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListUsers` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Show deleted users if specified. |
| filter | [string](#string) |  | Filter is used to filter users returned in the list. Supported filter: - name - email - user_type - state - project

For example: name == &#34;ed&#34; name.matches(&#34;ed&#34;) email == &#34;ed@bytebase.com&#34; email.matches(&#34;ed&#34;) user_type == &#34;SERVICE_ACCOUNT&#34; user_type in [&#34;SERVICE_ACCOUNT&#34;, &#34;USER&#34;] !(user_type in [&#34;SERVICE_ACCOUNT&#34;, &#34;USER&#34;]) state == &#34;DELETED&#34; project == &#34;projects/sample-project&#34; You can combine filter conditions like: name.matches(&#34;ed&#34;) &amp;&amp; project == &#34;projects/sample-project&#34; (name == &#34;ed&#34; || email == &#34;ed@bytebase.com&#34;) &amp;&amp; project == &#34;projects/sample-project&#34; |






<a name="bytebase-api-v1alpha-ListUsersResponse"></a>

### ListUsersResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| users | [User](#bytebase-api-v1alpha-User) | repeated | The users from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-UndeleteUserRequest"></a>

### UndeleteUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the deleted user. Format: users/{user} |






<a name="bytebase-api-v1alpha-UpdateUserRequest"></a>

### UpdateUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#bytebase-api-v1alpha-User) |  | The user to update.

The user&#39;s `name` field is used to identify the user to update. Format: users/{user} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| otp_code | [string](#string) | optional | The otp_code is used to verify the user&#39;s identity by MFA. |
| regenerate_temp_mfa_secret | [bool](#bool) |  | The regenerate_temp_mfa_secret flag means to regenerate temporary MFA secret for user. This is used for MFA setup. The temporary MFA secret and recovery codes will be returned in the response. |
| regenerate_recovery_codes | [bool](#bool) |  | The regenerate_recovery_codes flag means to regenerate recovery codes for user. |






<a name="bytebase-api-v1alpha-User"></a>

### User



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the user. Format: users/{user}. {user} is a system-generated unique ID. |
| state | [State](#bytebase-api-v1alpha-State) |  |  |
| email | [string](#string) |  |  |
| title | [string](#string) |  |  |
| user_type | [UserType](#bytebase-api-v1alpha-UserType) |  |  |
| password | [string](#string) |  |  |
| service_key | [string](#string) |  |  |
| mfa_enabled | [bool](#bool) |  | The mfa_enabled flag means if the user has enabled MFA. |
| mfa_secret | [string](#string) |  | The mfa_secret is the temporary secret using in two phase verification. |
| recovery_codes | [string](#string) | repeated | The recovery_codes is the temporary recovery codes using in two phase verification. |
| phone | [string](#string) |  | Should be a valid E.164 compliant phone number. Could be empty. |
| profile | [User.Profile](#bytebase-api-v1alpha-User-Profile) |  |  |






<a name="bytebase-api-v1alpha-User-Profile"></a>

### User.Profile



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| last_login_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| last_change_password_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| source | [string](#string) |  | source means where the user comes from. For now we support Entra ID SCIM sync, so the source could be Entra ID. |





 


<a name="bytebase-api-v1alpha-UserType"></a>

### UserType


| Name | Number | Description |
| ---- | ------ | ----------- |
| USER_TYPE_UNSPECIFIED | 0 |  |
| USER | 1 |  |
| SYSTEM_BOT | 2 |  |
| SERVICE_ACCOUNT | 3 |  |


 

 


<a name="bytebase-api-v1alpha-UserService"></a>

### UserService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetUser | [GetUserRequest](#bytebase-api-v1alpha-GetUserRequest) | [User](#bytebase-api-v1alpha-User) | Get the user. Any authenticated user can get the user. |
| GetCurrentUser | [.google.protobuf.Empty](#google-protobuf-Empty) | [User](#bytebase-api-v1alpha-User) | Get the current authenticated user. |
| ListUsers | [ListUsersRequest](#bytebase-api-v1alpha-ListUsersRequest) | [ListUsersResponse](#bytebase-api-v1alpha-ListUsersResponse) | List all users. Any authenticated user can list users. |
| CreateUser | [CreateUserRequest](#bytebase-api-v1alpha-CreateUserRequest) | [User](#bytebase-api-v1alpha-User) | Create a user. When Disallow Signup is enabled, only the caller with bb.users.create on the workspace can create a user. Otherwise, any unauthenticated user can create a user. |
| UpdateUser | [UpdateUserRequest](#bytebase-api-v1alpha-UpdateUserRequest) | [User](#bytebase-api-v1alpha-User) | Only the user itself and the user with bb.users.update permission on the workspace can update the user. |
| DeleteUser | [DeleteUserRequest](#bytebase-api-v1alpha-DeleteUserRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Only the user with bb.users.delete permission on the workspace can delete the user. The last remaining workspace admin cannot be deleted. |
| UndeleteUser | [UndeleteUserRequest](#bytebase-api-v1alpha-UndeleteUserRequest) | [User](#bytebase-api-v1alpha-User) | Only the user with bb.users.undelete permission on the workspace can undelete the user. |

 



<a name="api_v1alpha_actuator_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/actuator_service.proto



<a name="bytebase-api-v1alpha-ActuatorInfo"></a>

### ActuatorInfo
ServerInfo is the API message for server info.
Actuator concept is similar to the Spring Boot Actuator.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [string](#string) |  | version is the bytebase&#39;s server version |
| git_commit | [string](#string) |  | git_commit is the git commit hash of the build |
| readonly | [bool](#bool) |  | readonly flag means if the Bytebase is running in readonly mode. |
| saas | [bool](#bool) |  | saas flag means if the Bytebase is running in SaaS mode, some features are not allowed to edit by users. |
| demo | [bool](#bool) |  | demo flag means if the Bytebase is running in demo mode. |
| host | [string](#string) |  | host is the Bytebase instance host. |
| port | [string](#string) |  | port is the Bytebase instance port. |
| external_url | [string](#string) |  | external_url is the URL where user or webhook callback visits Bytebase. |
| need_admin_setup | [bool](#bool) |  | need_admin_setup flag means the Bytebase instance doesn&#39;t have any end users. |
| disallow_signup | [bool](#bool) |  | disallow_signup is the flag to disable self-service signup. |
| last_active_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | last_active_time is the service last active time in UTC Time Format, any API calls will refresh this value. |
| require_2fa | [bool](#bool) |  | require_2fa is the flag to require 2FA for all users. |
| workspace_id | [string](#string) |  | workspace_id is the identifier for the workspace. |
| debug | [bool](#bool) |  | debug flag means if the debug mode is enabled. |
| unlicensed_features | [string](#string) | repeated |  |
| disallow_password_signin | [bool](#bool) |  | disallow_password_signin is the flag to disallow user signin with email&amp;password. (except workspace admins) |
| password_restriction | [PasswordRestrictionSetting](#bytebase-api-v1alpha-PasswordRestrictionSetting) |  |  |
| docker | [bool](#bool) |  | docker flag means if the Bytebase instance is running in docker. |
| user_stats | [ActuatorInfo.StatUser](#bytebase-api-v1alpha-ActuatorInfo-StatUser) | repeated |  |
| activated_instance_count | [int32](#int32) |  |  |
| total_instance_count | [int32](#int32) |  |  |






<a name="bytebase-api-v1alpha-ActuatorInfo-StatUser"></a>

### ActuatorInfo.StatUser



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user_type | [UserType](#bytebase-api-v1alpha-UserType) |  |  |
| state | [State](#bytebase-api-v1alpha-State) |  |  |
| count | [int32](#int32) |  |  |






<a name="bytebase-api-v1alpha-DeleteCacheRequest"></a>

### DeleteCacheRequest







<a name="bytebase-api-v1alpha-GetActuatorInfoRequest"></a>

### GetActuatorInfoRequest







<a name="bytebase-api-v1alpha-GetResourcePackageRequest"></a>

### GetResourcePackageRequest
The request message for getting the theme resource.






<a name="bytebase-api-v1alpha-ResourcePackage"></a>

### ResourcePackage
The theme resources.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| logo | [bytes](#bytes) |  | The branding logo. |






<a name="bytebase-api-v1alpha-UpdateActuatorInfoRequest"></a>

### UpdateActuatorInfoRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| actuator | [ActuatorInfo](#bytebase-api-v1alpha-ActuatorInfo) |  | The actuator to update. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 

 

 


<a name="bytebase-api-v1alpha-ActuatorService"></a>

### ActuatorService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetActuatorInfo | [GetActuatorInfoRequest](#bytebase-api-v1alpha-GetActuatorInfoRequest) | [ActuatorInfo](#bytebase-api-v1alpha-ActuatorInfo) |  |
| UpdateActuatorInfo | [UpdateActuatorInfoRequest](#bytebase-api-v1alpha-UpdateActuatorInfoRequest) | [ActuatorInfo](#bytebase-api-v1alpha-ActuatorInfo) |  |
| DeleteCache | [DeleteCacheRequest](#bytebase-api-v1alpha-DeleteCacheRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| GetResourcePackage | [GetResourcePackageRequest](#bytebase-api-v1alpha-GetResourcePackageRequest) | [ResourcePackage](#bytebase-api-v1alpha-ResourcePackage) |  |

 



<a name="api_v1alpha_iam_policy-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/iam_policy.proto



<a name="bytebase-api-v1alpha-Binding"></a>

### Binding



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [string](#string) |  | The role that is assigned to the members. Format: roles/{role} |
| members | [string](#string) | repeated | Specifies the principals requesting access for a Bytebase resource. For users, the member should be: user:{email} For groups, the member should be: group:{email} |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this binding. If the condition evaluates to true, then this binding applies to the current request. If the condition evaluates to false, then this binding does not apply to the current request. However, a different role binding might grant the same role to one or more of the principals in this binding. |
| parsed_expr | [google.api.expr.v1alpha1.Expr](#google-api-expr-v1alpha1-Expr) |  | The parsed expression of the condition. |






<a name="bytebase-api-v1alpha-BindingDelta"></a>

### BindingDelta
One delta entry for Binding. Each individual change (only one member in each
entry) to a binding will be a separate entry.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| action | [BindingDelta.Action](#bytebase-api-v1alpha-BindingDelta-Action) |  | The action that was performed on a Binding. |
| role | [string](#string) |  | Role that is assigned to `members`. For example, `roles/projectOwner`. |
| member | [string](#string) |  | Follows the same format of Binding.members. |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this binding. |






<a name="bytebase-api-v1alpha-GetIamPolicyRequest"></a>

### GetIamPolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource | [string](#string) |  | The name of the resource to get the IAM policy. Format: projects/{project} Format: workspaces/{workspace} |






<a name="bytebase-api-v1alpha-IamPolicy"></a>

### IamPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bindings | [Binding](#bytebase-api-v1alpha-Binding) | repeated | Collection of binding. A binding binds one or more project members to a single project role. |
| etag | [string](#string) |  | The current etag of the policy. If an etag is provided and does not match the current etag of the poliy, the call will be blocked and an ABORTED error will be returned. |






<a name="bytebase-api-v1alpha-PolicyDelta"></a>

### PolicyDelta
The difference delta between two policies.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| binding_deltas | [BindingDelta](#bytebase-api-v1alpha-BindingDelta) | repeated | The delta for Bindings between two policies. |






<a name="bytebase-api-v1alpha-SetIamPolicyRequest"></a>

### SetIamPolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource | [string](#string) |  | The name of the resource to set the IAM policy. Format: projects/{project} Format: workspaces/{workspace} |
| policy | [IamPolicy](#bytebase-api-v1alpha-IamPolicy) |  |  |
| etag | [string](#string) |  | The current etag of the policy. |





 


<a name="bytebase-api-v1alpha-BindingDelta-Action"></a>

### BindingDelta.Action
The type of action performed on a Binding in a policy.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ACTION_UNSPECIFIED | 0 |  |
| ADD | 1 |  |
| REMOVE | 2 |  |


 

 

 



<a name="api_v1alpha_audit_log_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/audit_log_service.proto



<a name="bytebase-api-v1alpha-AuditData"></a>

### AuditData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy_delta | [PolicyDelta](#bytebase-api-v1alpha-PolicyDelta) |  |  |






<a name="bytebase-api-v1alpha-AuditLog"></a>

### AuditLog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the log. Formats: - projects/{project}/auditLogs/{uid} - workspaces/{workspace}/auditLogs/{uid} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| user | [string](#string) |  | Format: users/d@d.com |
| method | [string](#string) |  | e.g. `/bytebase.api.v1alphaSQLService/Query`, `bb.project.repository.push` |
| severity | [AuditLog.Severity](#bytebase-api-v1alpha-AuditLog-Severity) |  |  |
| resource | [string](#string) |  | The associated resource. |
| request | [string](#string) |  | JSON-encoded request. |
| response | [string](#string) |  | JSON-encoded response. Some fields are omitted because they are too large or contain sensitive information. |
| status | [google.rpc.Status](#google-rpc-Status) |  |  |
| service_data | [google.protobuf.Any](#google-protobuf-Any) |  | service-specific data about the request, response, and other activities. |
| request_metadata | [RequestMetadata](#bytebase-api-v1alpha-RequestMetadata) |  | Metadata about the operation. |






<a name="bytebase-api-v1alpha-ExportAuditLogsRequest"></a>

### ExportAuditLogsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  |  |
| filter | [string](#string) |  | The filter of the log. It should be a valid CEL expression. For example: - filter = &#34;method == &#39;/bytebase.api.v1alphaSQLService/Query&#39;&#34; - filter = &#34;method == &#39;/bytebase.api.v1alphaSQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39;&#34; - filter = &#34;method == &#39;/bytebase.api.v1alphaSQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39; &amp;&amp; user == &#39;users/bb@bytebase.com&#39;&#34; - filter = &#34;method == &#39;/bytebase.api.v1alphaSQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39; &amp;&amp; create_time &lt;= &#39;2021-01-01T00:00:00Z&#39; &amp;&amp; create_time &gt;= &#39;2020-01-01T00:00:00Z&#39;&#34; |
| order_by | [string](#string) |  | The order by of the log. Only support order by create_time. For example: - order_by = &#34;create_time asc&#34; - order_by = &#34;create_time desc&#34; |
| format | [ExportFormat](#bytebase-api-v1alpha-ExportFormat) |  | The export format. |
| page_size | [int32](#int32) |  | The maximum number of logs to return. The service may return fewer than this value. If unspecified, at most 10 log entries will be returned. The maximum value is 5000; values above 5000 will be coerced to 5000. |
| page_token | [string](#string) |  | A page token, received from a previous `ExportAuditLogs` call. Provide this to retrieve the subsequent page. |






<a name="bytebase-api-v1alpha-ExportAuditLogsResponse"></a>

### ExportAuditLogsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| content | [bytes](#bytes) |  |  |
| next_page_token | [string](#string) |  | A token to retrieve next page of log entities. Pass this value in the page_token field in the subsequent call to retrieve the next page of log entities. |






<a name="bytebase-api-v1alpha-RequestMetadata"></a>

### RequestMetadata
Metadata about the request.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| caller_ip | [string](#string) |  | The IP address of the caller. |
| caller_supplied_user_agent | [string](#string) |  | The user agent of the caller. This information is not authenticated and should be treated accordingly. |






<a name="bytebase-api-v1alpha-SearchAuditLogsRequest"></a>

### SearchAuditLogsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  |  |
| filter | [string](#string) |  | The filter of the log. It should be a valid CEL expression. Supported filter: - method - severity - user - create_time

For example: - filter = &#34;method == &#39;/bytebase.api.v1alphaSQLService/Query&#39;&#34; - filter = &#34;method == &#39;/bytebase.api.v1alphaSQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39;&#34; - filter = &#34;method == &#39;/bytebase.api.v1alphaSQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39; &amp;&amp; user == &#39;users/bb@bytebase.com&#39;&#34; - filter = &#34;method == &#39;/bytebase.api.v1alphaSQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39; &amp;&amp; create_time &lt;= &#39;2021-01-01T00:00:00Z&#39; &amp;&amp; create_time &gt;= &#39;2020-01-01T00:00:00Z&#39;&#34; |
| order_by | [string](#string) |  | The order by of the log. Only support order by create_time. For example: - order_by = &#34;create_time asc&#34; - order_by = &#34;create_time desc&#34; |
| page_size | [int32](#int32) |  | The maximum number of logs to return. The service may return fewer than this value. If unspecified, at most 10 log entries will be returned. The maximum value is 5000; values above 5000 will be coerced to 5000. |
| page_token | [string](#string) |  | A page token, received from a previous `SearchLogs` call. Provide this to retrieve the subsequent page. |






<a name="bytebase-api-v1alpha-SearchAuditLogsResponse"></a>

### SearchAuditLogsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| audit_logs | [AuditLog](#bytebase-api-v1alpha-AuditLog) | repeated |  |
| next_page_token | [string](#string) |  | A token to retrieve next page of log entities. Pass this value in the page_token field in the subsequent call to retrieve the next page of log entities. |





 


<a name="bytebase-api-v1alpha-AuditLog-Severity"></a>

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


 

 


<a name="bytebase-api-v1alpha-AuditLogService"></a>

### AuditLogService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| SearchAuditLogs | [SearchAuditLogsRequest](#bytebase-api-v1alpha-SearchAuditLogsRequest) | [SearchAuditLogsResponse](#bytebase-api-v1alpha-SearchAuditLogsResponse) |  |
| ExportAuditLogs | [ExportAuditLogsRequest](#bytebase-api-v1alpha-ExportAuditLogsRequest) | [ExportAuditLogsResponse](#bytebase-api-v1alpha-ExportAuditLogsResponse) |  |

 



<a name="api_v1alpha_auth_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/auth_service.proto



<a name="bytebase-api-v1alpha-IdentityProviderContext"></a>

### IdentityProviderContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| oauth2_context | [OAuth2IdentityProviderContext](#bytebase-api-v1alpha-OAuth2IdentityProviderContext) |  |  |
| oidc_context | [OIDCIdentityProviderContext](#bytebase-api-v1alpha-OIDCIdentityProviderContext) |  |  |






<a name="bytebase-api-v1alpha-LoginRequest"></a>

### LoginRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| email | [string](#string) |  |  |
| password | [string](#string) |  |  |
| web | [bool](#bool) |  | If web is set, we will set access token, refresh token, and user to the cookie. |
| idp_name | [string](#string) |  | The name of the identity provider. Format: idps/{idp} |
| idp_context | [IdentityProviderContext](#bytebase-api-v1alpha-IdentityProviderContext) |  | The idp_context is using to get the user information from identity provider. |
| otp_code | [string](#string) | optional | The otp_code is used to verify the user&#39;s identity by MFA. |
| recovery_code | [string](#string) | optional | The recovery_code is used to recovery the user&#39;s identity with MFA. |
| mfa_temp_token | [string](#string) | optional | The mfa_temp_token is used to verify the user&#39;s identity by MFA. |






<a name="bytebase-api-v1alpha-LoginResponse"></a>

### LoginResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  |  |
| mfa_temp_token | [string](#string) | optional |  |
| require_reset_password | [bool](#bool) |  |  |
| user | [User](#bytebase-api-v1alpha-User) |  | The user of successful login. |






<a name="bytebase-api-v1alpha-LogoutRequest"></a>

### LogoutRequest







<a name="bytebase-api-v1alpha-OAuth2IdentityProviderContext"></a>

### OAuth2IdentityProviderContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-OIDCIdentityProviderContext"></a>

### OIDCIdentityProviderContext






 

 

 


<a name="bytebase-api-v1alpha-AuthService"></a>

### AuthService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Login | [LoginRequest](#bytebase-api-v1alpha-LoginRequest) | [LoginResponse](#bytebase-api-v1alpha-LoginResponse) |  |
| Logout | [LogoutRequest](#bytebase-api-v1alpha-LogoutRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="api_v1alpha_cel_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/cel_service.proto



<a name="bytebase-api-v1alpha-BatchDeparseRequest"></a>

### BatchDeparseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expressions | [google.api.expr.v1alpha1.Expr](#google-api-expr-v1alpha1-Expr) | repeated |  |






<a name="bytebase-api-v1alpha-BatchDeparseResponse"></a>

### BatchDeparseResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expressions | [string](#string) | repeated |  |






<a name="bytebase-api-v1alpha-BatchParseRequest"></a>

### BatchParseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expressions | [string](#string) | repeated |  |






<a name="bytebase-api-v1alpha-BatchParseResponse"></a>

### BatchParseResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expressions | [google.api.expr.v1alpha1.Expr](#google-api-expr-v1alpha1-Expr) | repeated |  |





 

 

 


<a name="bytebase-api-v1alpha-CelService"></a>

### CelService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| BatchParse | [BatchParseRequest](#bytebase-api-v1alpha-BatchParseRequest) | [BatchParseResponse](#bytebase-api-v1alpha-BatchParseResponse) |  |
| BatchDeparse | [BatchDeparseRequest](#bytebase-api-v1alpha-BatchDeparseRequest) | [BatchDeparseResponse](#bytebase-api-v1alpha-BatchDeparseResponse) |  |

 



<a name="api_v1alpha_changelist_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/changelist_service.proto



<a name="bytebase-api-v1alpha-Changelist"></a>

### Changelist



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the changelist resource. Canonical parent is project. Format: projects/{project}/changelists/{changelist} |
| description | [string](#string) |  |  |
| creator | [string](#string) |  | The creator of the changelist. Format: users/{email} |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The last update time of the changelist. |
| changes | [Changelist.Change](#bytebase-api-v1alpha-Changelist-Change) | repeated |  |






<a name="bytebase-api-v1alpha-Changelist-Change"></a>

### Changelist.Change



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheet | [string](#string) |  | The name of a sheet. |
| source | [string](#string) |  | The source of origin. 1) changelog: instances/{instance}/databases/{database}/changelogs/{changelog}. 2) raw SQL if empty. |
| version | [string](#string) |  | The migration version for a change. |






<a name="bytebase-api-v1alpha-CreateChangelistRequest"></a>

### CreateChangelistRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this changelist will be created. Format: projects/{project} |
| changelist | [Changelist](#bytebase-api-v1alpha-Changelist) |  | The changelist to create. |
| changelist_id | [string](#string) |  | The ID to use for the changelist, which will become the final component of the changelist&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |






<a name="bytebase-api-v1alpha-DeleteChangelistRequest"></a>

### DeleteChangelistRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the changelist to delete. Format: projects/{project}/changelists/{changelist} |






<a name="bytebase-api-v1alpha-GetChangelistRequest"></a>

### GetChangelistRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the changelist to retrieve. Format: projects/{project}/changelists/{changelist} |






<a name="bytebase-api-v1alpha-ListChangelistsRequest"></a>

### ListChangelistsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of changelists. Format: projects/{project} |
| page_size | [int32](#int32) |  | Not used. The maximum number of databases to return. The service may return fewer than this value. If unspecified, at most 50 databases will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListDatabases` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListDatabases` must match the call that provided the page token. |






<a name="bytebase-api-v1alpha-ListChangelistsResponse"></a>

### ListChangelistsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| changelists | [Changelist](#bytebase-api-v1alpha-Changelist) | repeated | The changelists from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-UpdateChangelistRequest"></a>

### UpdateChangelistRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| changelist | [Changelist](#bytebase-api-v1alpha-Changelist) |  | The changelist to update.

The changelist&#39;s `name` field is used to identify the changelist to update. Format: projects/{project}/changelists/{changelist} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to be updated. |





 

 

 


<a name="bytebase-api-v1alpha-ChangelistService"></a>

### ChangelistService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateChangelist | [CreateChangelistRequest](#bytebase-api-v1alpha-CreateChangelistRequest) | [Changelist](#bytebase-api-v1alpha-Changelist) |  |
| GetChangelist | [GetChangelistRequest](#bytebase-api-v1alpha-GetChangelistRequest) | [Changelist](#bytebase-api-v1alpha-Changelist) |  |
| ListChangelists | [ListChangelistsRequest](#bytebase-api-v1alpha-ListChangelistsRequest) | [ListChangelistsResponse](#bytebase-api-v1alpha-ListChangelistsResponse) |  |
| UpdateChangelist | [UpdateChangelistRequest](#bytebase-api-v1alpha-UpdateChangelistRequest) | [Changelist](#bytebase-api-v1alpha-Changelist) |  |
| DeleteChangelist | [DeleteChangelistRequest](#bytebase-api-v1alpha-DeleteChangelistRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="api_v1alpha_database_group_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/database_group_service.proto



<a name="bytebase-api-v1alpha-CreateDatabaseGroupRequest"></a>

### CreateDatabaseGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this database group will be created. Format: projects/{project} |
| database_group | [DatabaseGroup](#bytebase-api-v1alpha-DatabaseGroup) |  | The database group to create. |
| database_group_id | [string](#string) |  | The ID to use for the database group, which will become the final component of the database group&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |
| validate_only | [bool](#bool) |  | If set, validate the create request and preview the full database group response, but do not actually create it. |






<a name="bytebase-api-v1alpha-DatabaseGroup"></a>

### DatabaseGroup



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database group. Format: projects/{project}/databaseGroups/{databaseGroup} |
| database_placeholder | [string](#string) |  | The short name used in actual databases specified by users. For example, the placeholder for db1_2010, db1_2021, db1_2023 will be &#34;db1&#34;. |
| database_expr | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this database group. |
| matched_databases | [DatabaseGroup.Database](#bytebase-api-v1alpha-DatabaseGroup-Database) | repeated | The list of databases that match the database group condition. |
| unmatched_databases | [DatabaseGroup.Database](#bytebase-api-v1alpha-DatabaseGroup-Database) | repeated | The list of databases that match the database group condition. |






<a name="bytebase-api-v1alpha-DatabaseGroup-Database"></a>

### DatabaseGroup.Database



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The resource name of the database. Format: instances/{instance}/databases/{database} |






<a name="bytebase-api-v1alpha-DeleteDatabaseGroupRequest"></a>

### DeleteDatabaseGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database group to delete. Format: projects/{project}/databaseGroups/{databaseGroup} |






<a name="bytebase-api-v1alpha-GetDatabaseGroupRequest"></a>

### GetDatabaseGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database group to retrieve. Format: projects/{project}/databaseGroups/{databaseGroup} |
| view | [DatabaseGroupView](#bytebase-api-v1alpha-DatabaseGroupView) |  | The view to return. Defaults to DATABASE_GROUP_VIEW_BASIC. |






<a name="bytebase-api-v1alpha-ListDatabaseGroupsRequest"></a>

### ListDatabaseGroupsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource whose database groups are to be listed. Format: projects/{project} |
| page_size | [int32](#int32) |  | Not used. The maximum number of database groups to return. The service may return fewer than this value. If unspecified, at most 50 database groups will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListDatabaseGroups` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListDatabaseGroups` must match the call that provided the page token. |






<a name="bytebase-api-v1alpha-ListDatabaseGroupsResponse"></a>

### ListDatabaseGroupsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database_groups | [DatabaseGroup](#bytebase-api-v1alpha-DatabaseGroup) | repeated | database_groups is the list of database groups. |
| next_page_token | [string](#string) |  | Not used. A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-UpdateDatabaseGroupRequest"></a>

### UpdateDatabaseGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database_group | [DatabaseGroup](#bytebase-api-v1alpha-DatabaseGroup) |  | The database group to update.

The database group&#39;s `name` field is used to identify the database group to update. Format: projects/{project}/databaseGroups/{databaseGroup} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 


<a name="bytebase-api-v1alpha-DatabaseGroupView"></a>

### DatabaseGroupView


| Name | Number | Description |
| ---- | ------ | ----------- |
| DATABASE_GROUP_VIEW_UNSPECIFIED | 0 | The default / unset value. The API will default to the BASIC view. |
| DATABASE_GROUP_VIEW_BASIC | 1 | Include basic information about the database group, but exclude the list of matched databases and unmatched databases. |
| DATABASE_GROUP_VIEW_FULL | 2 | Include everything. |


 

 


<a name="bytebase-api-v1alpha-DatabaseGroupService"></a>

### DatabaseGroupService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListDatabaseGroups | [ListDatabaseGroupsRequest](#bytebase-api-v1alpha-ListDatabaseGroupsRequest) | [ListDatabaseGroupsResponse](#bytebase-api-v1alpha-ListDatabaseGroupsResponse) |  |
| GetDatabaseGroup | [GetDatabaseGroupRequest](#bytebase-api-v1alpha-GetDatabaseGroupRequest) | [DatabaseGroup](#bytebase-api-v1alpha-DatabaseGroup) |  |
| CreateDatabaseGroup | [CreateDatabaseGroupRequest](#bytebase-api-v1alpha-CreateDatabaseGroupRequest) | [DatabaseGroup](#bytebase-api-v1alpha-DatabaseGroup) |  |
| UpdateDatabaseGroup | [UpdateDatabaseGroupRequest](#bytebase-api-v1alpha-UpdateDatabaseGroupRequest) | [DatabaseGroup](#bytebase-api-v1alpha-DatabaseGroup) |  |
| DeleteDatabaseGroup | [DeleteDatabaseGroupRequest](#bytebase-api-v1alpha-DeleteDatabaseGroupRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="api_v1alpha_group_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/group_service.proto



<a name="bytebase-api-v1alpha-CreateGroupRequest"></a>

### CreateGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [Group](#bytebase-api-v1alpha-Group) |  | The group to create. |
| group_email | [string](#string) |  | The email to use for the group, which will become the final component of the group&#39;s resource name. |






<a name="bytebase-api-v1alpha-DeleteGroupRequest"></a>

### DeleteGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the group to delete. Format: groups/{email} |






<a name="bytebase-api-v1alpha-GetGroupRequest"></a>

### GetGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the group to retrieve. Format: groups/{email} |






<a name="bytebase-api-v1alpha-Group"></a>

### Group



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the group to retrieve. Format: groups/{group}, group is an email. |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| members | [GroupMember](#bytebase-api-v1alpha-GroupMember) | repeated |  |
| source | [string](#string) |  | source means where the group comes from. For now we support Entra ID SCIM sync, so the source could be Entra ID. |






<a name="bytebase-api-v1alpha-GroupMember"></a>

### GroupMember



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| member | [string](#string) |  | Member is the principal who belong to this group.

Format: users/hello@world.com |
| role | [GroupMember.Role](#bytebase-api-v1alpha-GroupMember-Role) |  |  |






<a name="bytebase-api-v1alpha-ListGroupsRequest"></a>

### ListGroupsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Not used. The maximum number of groups to return. The service may return fewer than this value. If unspecified, at most 10 groups will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListGroups` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListGroups` must match the call that provided the page token. |






<a name="bytebase-api-v1alpha-ListGroupsResponse"></a>

### ListGroupsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| groups | [Group](#bytebase-api-v1alpha-Group) | repeated | The groups from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-UpdateGroupRequest"></a>

### UpdateGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [Group](#bytebase-api-v1alpha-Group) |  | The group to update.

The group&#39;s `name` field is used to identify the group to update. Format: groups/{email} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the group is not found, a new group will be created. In this situation, `update_mask` is ignored. |





 


<a name="bytebase-api-v1alpha-GroupMember-Role"></a>

### GroupMember.Role


| Name | Number | Description |
| ---- | ------ | ----------- |
| ROLE_UNSPECIFIED | 0 |  |
| OWNER | 1 |  |
| MEMBER | 2 |  |


 

 


<a name="bytebase-api-v1alpha-GroupService"></a>

### GroupService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetGroup | [GetGroupRequest](#bytebase-api-v1alpha-GetGroupRequest) | [Group](#bytebase-api-v1alpha-Group) |  |
| ListGroups | [ListGroupsRequest](#bytebase-api-v1alpha-ListGroupsRequest) | [ListGroupsResponse](#bytebase-api-v1alpha-ListGroupsResponse) |  |
| CreateGroup | [CreateGroupRequest](#bytebase-api-v1alpha-CreateGroupRequest) | [Group](#bytebase-api-v1alpha-Group) |  |
| UpdateGroup | [UpdateGroupRequest](#bytebase-api-v1alpha-UpdateGroupRequest) | [Group](#bytebase-api-v1alpha-Group) | UpdateGroup updates the group. Users with &#34;bb.groups.update&#34; permission on the workspace or the group owner can access this method. |
| DeleteGroup | [DeleteGroupRequest](#bytebase-api-v1alpha-DeleteGroupRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="api_v1alpha_idp_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/idp_service.proto



<a name="bytebase-api-v1alpha-CreateIdentityProviderRequest"></a>

### CreateIdentityProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identity_provider | [IdentityProvider](#bytebase-api-v1alpha-IdentityProvider) |  | The identity provider to create. |
| identity_provider_id | [string](#string) |  | The ID to use for the identity provider, which will become the final component of the identity provider&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |
| validate_only | [bool](#bool) |  | If set to true, the request will be validated without actually creating the identity provider. |






<a name="bytebase-api-v1alpha-DeleteIdentityProviderRequest"></a>

### DeleteIdentityProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the identity provider to delete. Format: idps/{identity_provider} |






<a name="bytebase-api-v1alpha-FieldMapping"></a>

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






<a name="bytebase-api-v1alpha-GetIdentityProviderRequest"></a>

### GetIdentityProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-IdentityProvider"></a>

### IdentityProvider



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the identity provider. Format: idps/{idp} |
| title | [string](#string) |  |  |
| domain | [string](#string) |  |  |
| type | [IdentityProviderType](#bytebase-api-v1alpha-IdentityProviderType) |  |  |
| config | [IdentityProviderConfig](#bytebase-api-v1alpha-IdentityProviderConfig) |  |  |






<a name="bytebase-api-v1alpha-IdentityProviderConfig"></a>

### IdentityProviderConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| oauth2_config | [OAuth2IdentityProviderConfig](#bytebase-api-v1alpha-OAuth2IdentityProviderConfig) |  |  |
| oidc_config | [OIDCIdentityProviderConfig](#bytebase-api-v1alpha-OIDCIdentityProviderConfig) |  |  |
| ldap_config | [LDAPIdentityProviderConfig](#bytebase-api-v1alpha-LDAPIdentityProviderConfig) |  |  |






<a name="bytebase-api-v1alpha-LDAPIdentityProviderConfig"></a>

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
| security_protocol | [string](#string) |  | SecurityProtocol is the security protocol to be used for establishing connections with the LDAP server. It must be StartTLS, LDAPS or None. |
| field_mapping | [FieldMapping](#bytebase-api-v1alpha-FieldMapping) |  | FieldMapping is the mapping of the user attributes returned by the LDAP server. |






<a name="bytebase-api-v1alpha-ListIdentityProvidersRequest"></a>

### ListIdentityProvidersRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Not used. The maximum number of identity providers to return. The service may return fewer than this value. If unspecified, at most 10 will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListIdentityProviders` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListIdentityProviders` must match the call that provided the page token. |






<a name="bytebase-api-v1alpha-ListIdentityProvidersResponse"></a>

### ListIdentityProvidersResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identity_providers | [IdentityProvider](#bytebase-api-v1alpha-IdentityProvider) | repeated | The identity providers from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-OAuth2IdentityProviderConfig"></a>

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
| field_mapping | [FieldMapping](#bytebase-api-v1alpha-FieldMapping) |  |  |
| skip_tls_verify | [bool](#bool) |  |  |
| auth_style | [OAuth2AuthStyle](#bytebase-api-v1alpha-OAuth2AuthStyle) |  |  |






<a name="bytebase-api-v1alpha-OAuth2IdentityProviderTestRequestContext"></a>

### OAuth2IdentityProviderTestRequestContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [string](#string) |  | Authorize code from website. |






<a name="bytebase-api-v1alpha-OIDCIdentityProviderConfig"></a>

### OIDCIdentityProviderConfig
OIDCIdentityProviderConfig is the structure for OIDC identity provider config.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issuer | [string](#string) |  |  |
| client_id | [string](#string) |  |  |
| client_secret | [string](#string) |  |  |
| scopes | [string](#string) | repeated | The scopes that the OIDC provider supports. Should be fetched from the well-known configuration file of the OIDC provider. |
| field_mapping | [FieldMapping](#bytebase-api-v1alpha-FieldMapping) |  |  |
| skip_tls_verify | [bool](#bool) |  |  |
| auth_style | [OAuth2AuthStyle](#bytebase-api-v1alpha-OAuth2AuthStyle) |  |  |
| auth_endpoint | [string](#string) |  | The authorization endpoint of the OIDC provider. Should be fetched from the well-known configuration file of the OIDC provider. |






<a name="bytebase-api-v1alpha-TestIdentityProviderRequest"></a>

### TestIdentityProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identity_provider | [IdentityProvider](#bytebase-api-v1alpha-IdentityProvider) |  | The identity provider to test connection including uncreated. |
| oauth2_context | [OAuth2IdentityProviderTestRequestContext](#bytebase-api-v1alpha-OAuth2IdentityProviderTestRequestContext) |  |  |






<a name="bytebase-api-v1alpha-TestIdentityProviderResponse"></a>

### TestIdentityProviderResponse







<a name="bytebase-api-v1alpha-UpdateIdentityProviderRequest"></a>

### UpdateIdentityProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identity_provider | [IdentityProvider](#bytebase-api-v1alpha-IdentityProvider) |  | The identity provider to update.

The identity provider&#39;s `name` field is used to identify the identity provider to update. Format: idps/{identity_provider} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 


<a name="bytebase-api-v1alpha-IdentityProviderType"></a>

### IdentityProviderType


| Name | Number | Description |
| ---- | ------ | ----------- |
| IDENTITY_PROVIDER_TYPE_UNSPECIFIED | 0 |  |
| OAUTH2 | 1 |  |
| OIDC | 2 |  |
| LDAP | 3 |  |



<a name="bytebase-api-v1alpha-OAuth2AuthStyle"></a>

### OAuth2AuthStyle


| Name | Number | Description |
| ---- | ------ | ----------- |
| OAUTH2_AUTH_STYLE_UNSPECIFIED | 0 |  |
| IN_PARAMS | 1 | IN_PARAMS sends the &#34;client_id&#34; and &#34;client_secret&#34; in the POST body as application/x-www-form-urlencoded parameters. |
| IN_HEADER | 2 | IN_HEADER sends the client_id and client_password using HTTP Basic Authorization. This is an optional style described in the OAuth2 RFC 6749 section 2.3.1. |


 

 


<a name="bytebase-api-v1alpha-IdentityProviderService"></a>

### IdentityProviderService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetIdentityProvider | [GetIdentityProviderRequest](#bytebase-api-v1alpha-GetIdentityProviderRequest) | [IdentityProvider](#bytebase-api-v1alpha-IdentityProvider) |  |
| ListIdentityProviders | [ListIdentityProvidersRequest](#bytebase-api-v1alpha-ListIdentityProvidersRequest) | [ListIdentityProvidersResponse](#bytebase-api-v1alpha-ListIdentityProvidersResponse) |  |
| CreateIdentityProvider | [CreateIdentityProviderRequest](#bytebase-api-v1alpha-CreateIdentityProviderRequest) | [IdentityProvider](#bytebase-api-v1alpha-IdentityProvider) |  |
| UpdateIdentityProvider | [UpdateIdentityProviderRequest](#bytebase-api-v1alpha-UpdateIdentityProviderRequest) | [IdentityProvider](#bytebase-api-v1alpha-IdentityProvider) |  |
| DeleteIdentityProvider | [DeleteIdentityProviderRequest](#bytebase-api-v1alpha-DeleteIdentityProviderRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| TestIdentityProvider | [TestIdentityProviderRequest](#bytebase-api-v1alpha-TestIdentityProviderRequest) | [TestIdentityProviderResponse](#bytebase-api-v1alpha-TestIdentityProviderResponse) |  |

 



<a name="api_v1alpha_org_policy_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/org_policy_service.proto



<a name="bytebase-api-v1alpha-CreatePolicyRequest"></a>

### CreatePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this instance will be created. Workspace resource name: &#34;&#34;. Environment resource name: environments/environment-id. Instance resource name: instances/instance-id. Database resource name: instances/instance-id/databases/database-name. |
| policy | [Policy](#bytebase-api-v1alpha-Policy) |  | The policy to create. |
| type | [PolicyType](#bytebase-api-v1alpha-PolicyType) |  |  |






<a name="bytebase-api-v1alpha-DataSourceQueryPolicy"></a>

### DataSourceQueryPolicy
DataSourceQueryPolicy is the policy configuration for running statements in the SQL editor.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| admin_data_source_restriction | [DataSourceQueryPolicy.Restriction](#bytebase-api-v1alpha-DataSourceQueryPolicy-Restriction) |  |  |
| disallow_ddl | [bool](#bool) |  | Disallow running DDL statements in the SQL editor. |
| disallow_dml | [bool](#bool) |  | Disallow running DML statements in the SQL editor. |






<a name="bytebase-api-v1alpha-DeletePolicyRequest"></a>

### DeletePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The policy&#39;s `name` field is used to identify the instance to update. Format: {resource name}/policies/{policy type} Workspace resource name: &#34;&#34;. Environment resource name: environments/environment-id. Instance resource name: instances/instance-id. Database resource name: instances/instance-id/databases/database-name. |






<a name="bytebase-api-v1alpha-DisableCopyDataPolicy"></a>

### DisableCopyDataPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| active | [bool](#bool) |  |  |






<a name="bytebase-api-v1alpha-ExportDataPolicy"></a>

### ExportDataPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| disable | [bool](#bool) |  |  |






<a name="bytebase-api-v1alpha-GetPolicyRequest"></a>

### GetPolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the policy to retrieve. Format: {resource type}/{resource id}/policies/{policy type} |






<a name="bytebase-api-v1alpha-ListPoliciesRequest"></a>

### ListPoliciesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of policies. Format: {resource type}/{resource id} |
| policy_type | [PolicyType](#bytebase-api-v1alpha-PolicyType) | optional |  |
| page_size | [int32](#int32) |  | Not used. The maximum number of policies to return. The service may return fewer than this value. If unspecified, at most 10 policies will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListPolicies` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListPolicies` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Show deleted policies if specified. |






<a name="bytebase-api-v1alpha-ListPoliciesResponse"></a>

### ListPoliciesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policies | [Policy](#bytebase-api-v1alpha-Policy) | repeated | The policies from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-MaskingExceptionPolicy"></a>

### MaskingExceptionPolicy
MaskingExceptionPolicy is the allowlist of users who can access sensitive data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| masking_exceptions | [MaskingExceptionPolicy.MaskingException](#bytebase-api-v1alpha-MaskingExceptionPolicy-MaskingException) | repeated |  |






<a name="bytebase-api-v1alpha-MaskingExceptionPolicy-MaskingException"></a>

### MaskingExceptionPolicy.MaskingException



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| action | [MaskingExceptionPolicy.MaskingException.Action](#bytebase-api-v1alpha-MaskingExceptionPolicy-MaskingException-Action) |  | action is the action that the user can access sensitive data. |
| member | [string](#string) |  | Member is the principal who bind to this exception policy instance.

- `user:{email}`: An email address that represents a specific Bytebase account. For example, `alice@example.com`. - `group:{email}`: An email address for group. |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this exception policy instance. |






<a name="bytebase-api-v1alpha-MaskingRulePolicy"></a>

### MaskingRulePolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rules | [MaskingRulePolicy.MaskingRule](#bytebase-api-v1alpha-MaskingRulePolicy-MaskingRule) | repeated |  |






<a name="bytebase-api-v1alpha-MaskingRulePolicy-MaskingRule"></a>

### MaskingRulePolicy.MaskingRule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | A unique identifier for a node in UUID format. |
| condition | [google.type.Expr](#google-type-Expr) |  |  |
| semantic_type | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-Policy"></a>

### Policy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the policy. Format: {resource name}/policies/{policy type} Workspace resource name: &#34;&#34;. Environment resource name: environments/environment-id. Instance resource name: instances/instance-id. Database resource name: instances/instance-id/databases/database-name. |
| inherit_from_parent | [bool](#bool) |  |  |
| type | [PolicyType](#bytebase-api-v1alpha-PolicyType) |  |  |
| rollout_policy | [RolloutPolicy](#bytebase-api-v1alpha-RolloutPolicy) |  |  |
| disable_copy_data_policy | [DisableCopyDataPolicy](#bytebase-api-v1alpha-DisableCopyDataPolicy) |  |  |
| masking_rule_policy | [MaskingRulePolicy](#bytebase-api-v1alpha-MaskingRulePolicy) |  |  |
| masking_exception_policy | [MaskingExceptionPolicy](#bytebase-api-v1alpha-MaskingExceptionPolicy) |  |  |
| restrict_issue_creation_for_sql_review_policy | [RestrictIssueCreationForSQLReviewPolicy](#bytebase-api-v1alpha-RestrictIssueCreationForSQLReviewPolicy) |  |  |
| tag_policy | [TagPolicy](#bytebase-api-v1alpha-TagPolicy) |  |  |
| data_source_query_policy | [DataSourceQueryPolicy](#bytebase-api-v1alpha-DataSourceQueryPolicy) |  |  |
| export_data_policy | [ExportDataPolicy](#bytebase-api-v1alpha-ExportDataPolicy) |  |  |
| query_data_policy | [QueryDataPolicy](#bytebase-api-v1alpha-QueryDataPolicy) |  |  |
| enforce | [bool](#bool) |  |  |
| resource_type | [PolicyResourceType](#bytebase-api-v1alpha-PolicyResourceType) |  | The resource type for the policy. |






<a name="bytebase-api-v1alpha-QueryDataPolicy"></a>

### QueryDataPolicy
QueryDataPolicy is the policy configuration for querying data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timeout | [google.protobuf.Duration](#google-protobuf-Duration) |  | The query timeout duration. |






<a name="bytebase-api-v1alpha-RestrictIssueCreationForSQLReviewPolicy"></a>

### RestrictIssueCreationForSQLReviewPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| disallow | [bool](#bool) |  |  |






<a name="bytebase-api-v1alpha-RolloutPolicy"></a>

### RolloutPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| automatic | [bool](#bool) |  |  |
| roles | [string](#string) | repeated |  |
| issue_roles | [string](#string) | repeated | roles/LAST_APPROVER roles/CREATOR |






<a name="bytebase-api-v1alpha-SQLReviewRule"></a>

### SQLReviewRule



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  |  |
| level | [SQLReviewRuleLevel](#bytebase-api-v1alpha-SQLReviewRuleLevel) |  |  |
| payload | [string](#string) |  |  |
| engine | [Engine](#bytebase-api-v1alpha-Engine) |  |  |
| comment | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-TagPolicy"></a>

### TagPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tags | [TagPolicy.TagsEntry](#bytebase-api-v1alpha-TagPolicy-TagsEntry) | repeated | tags is the key - value map for resources. for example, the environment resource can have the sql review config tag, like &#34;bb.tag.review_config&#34;: &#34;reviewConfigs/{review config resource id}&#34; |






<a name="bytebase-api-v1alpha-TagPolicy-TagsEntry"></a>

### TagPolicy.TagsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-UpdatePolicyRequest"></a>

### UpdatePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy | [Policy](#bytebase-api-v1alpha-Policy) |  | The policy to update.

The policy&#39;s `name` field is used to identify the instance to update. Format: {resource name}/policies/{policy type} Workspace resource name: &#34;&#34;. Environment resource name: environments/environment-id. Instance resource name: instances/instance-id. Database resource name: instances/instance-id/databases/database-name. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the policy is not found, a new policy will be created. In this situation, `update_mask` is ignored. |





 


<a name="bytebase-api-v1alpha-DataSourceQueryPolicy-Restriction"></a>

### DataSourceQueryPolicy.Restriction


| Name | Number | Description |
| ---- | ------ | ----------- |
| RESTRICTION_UNSPECIFIED | 0 |  |
| FALLBACK | 1 | Allow to query admin data sources when there is no read-only data source. |
| DISALLOW | 2 | Disallow to query admin data sources. |



<a name="bytebase-api-v1alpha-MaskingExceptionPolicy-MaskingException-Action"></a>

### MaskingExceptionPolicy.MaskingException.Action


| Name | Number | Description |
| ---- | ------ | ----------- |
| ACTION_UNSPECIFIED | 0 |  |
| QUERY | 1 |  |
| EXPORT | 2 |  |



<a name="bytebase-api-v1alpha-PolicyResourceType"></a>

### PolicyResourceType


| Name | Number | Description |
| ---- | ------ | ----------- |
| RESOURCE_TYPE_UNSPECIFIED | 0 |  |
| WORKSPACE | 1 |  |
| ENVIRONMENT | 2 |  |
| PROJECT | 3 |  |
| INSTANCE | 4 |  |
| DATABASE | 5 |  |



<a name="bytebase-api-v1alpha-PolicyType"></a>

### PolicyType


| Name | Number | Description |
| ---- | ------ | ----------- |
| POLICY_TYPE_UNSPECIFIED | 0 |  |
| ROLLOUT_POLICY | 11 |  |
| DISABLE_COPY_DATA | 8 |  |
| MASKING_RULE | 9 |  |
| MASKING_EXCEPTION | 10 |  |
| RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW | 12 |  |
| TAG | 13 |  |
| DATA_SOURCE_QUERY | 14 |  |
| DATA_EXPORT | 15 |  |
| DATA_QUERY | 16 |  |



<a name="bytebase-api-v1alpha-SQLReviewRuleLevel"></a>

### SQLReviewRuleLevel


| Name | Number | Description |
| ---- | ------ | ----------- |
| LEVEL_UNSPECIFIED | 0 |  |
| ERROR | 1 |  |
| WARNING | 2 |  |
| DISABLED | 3 |  |


 

 


<a name="bytebase-api-v1alpha-OrgPolicyService"></a>

### OrgPolicyService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetPolicy | [GetPolicyRequest](#bytebase-api-v1alpha-GetPolicyRequest) | [Policy](#bytebase-api-v1alpha-Policy) |  |
| ListPolicies | [ListPoliciesRequest](#bytebase-api-v1alpha-ListPoliciesRequest) | [ListPoliciesResponse](#bytebase-api-v1alpha-ListPoliciesResponse) |  |
| CreatePolicy | [CreatePolicyRequest](#bytebase-api-v1alpha-CreatePolicyRequest) | [Policy](#bytebase-api-v1alpha-Policy) |  |
| UpdatePolicy | [UpdatePolicyRequest](#bytebase-api-v1alpha-UpdatePolicyRequest) | [Policy](#bytebase-api-v1alpha-Policy) |  |
| DeletePolicy | [DeletePolicyRequest](#bytebase-api-v1alpha-DeletePolicyRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="api_v1alpha_plan_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/plan_service.proto



<a name="bytebase-api-v1alpha-BatchCancelPlanCheckRunsRequest"></a>

### BatchCancelPlanCheckRunsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The name of the parent of the planChecks. Format: projects/{project}/plans/{plan} |
| plan_check_runs | [string](#string) | repeated | TODO(d): update this API. The planCheckRuns to cancel. Format: projects/{project}/plans/{plan}/planCheckRuns/{planCheckRun} |






<a name="bytebase-api-v1alpha-BatchCancelPlanCheckRunsResponse"></a>

### BatchCancelPlanCheckRunsResponse







<a name="bytebase-api-v1alpha-CreatePlanRequest"></a>

### CreatePlanRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent project where this plan will be created. Format: projects/{project} |
| plan | [Plan](#bytebase-api-v1alpha-Plan) |  | The plan to create. |






<a name="bytebase-api-v1alpha-GetPlanRequest"></a>

### GetPlanRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the plan to retrieve. Format: projects/{project}/plans/{plan} |






<a name="bytebase-api-v1alpha-ListPlanCheckRunsRequest"></a>

### ListPlanCheckRunsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of plan check runs. Format: projects/{project}/plans/{plan} |
| page_size | [int32](#int32) |  | Not used. The maximum number of plan check runs to return. The service may return fewer than this value. If unspecified, at most 10 plan check runs will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListPlanCheckRuns` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListPlanCheckRuns` must match the call that provided the page token. |
| latest_only | [bool](#bool) |  | If set to true, only the latest plan check run will be returned. |






<a name="bytebase-api-v1alpha-ListPlanCheckRunsResponse"></a>

### ListPlanCheckRunsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| plan_check_runs | [PlanCheckRun](#bytebase-api-v1alpha-PlanCheckRun) | repeated | The plan check runs from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-ListPlansRequest"></a>

### ListPlansRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of plans. Format: projects/{project} Use &#34;projects/-&#34; to list all plans from all projects. |
| page_size | [int32](#int32) |  | The maximum number of plans to return. The service may return fewer than this value. If unspecified, at most 10 plans will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListPlans` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListPlans` must match the call that provided the page token. |






<a name="bytebase-api-v1alpha-ListPlansResponse"></a>

### ListPlansResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| plans | [Plan](#bytebase-api-v1alpha-Plan) | repeated | The plans from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-Plan"></a>

### Plan



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the plan. `plan` is a system generated ID. Format: projects/{project}/plans/{plan} |
| issue | [string](#string) |  | The resource name of the issue associated with this plan. Format: projects/{project}/issues/{issue} |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| steps | [Plan.Step](#bytebase-api-v1alpha-Plan-Step) | repeated |  |
| creator | [string](#string) |  | Format: users/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| plan_check_run_status_count | [Plan.PlanCheckRunStatusCountEntry](#bytebase-api-v1alpha-Plan-PlanCheckRunStatusCountEntry) | repeated | The status count of the latest plan check runs. Keys are: - SUCCESS - WARNING - ERROR |
| release_source | [Plan.ReleaseSource](#bytebase-api-v1alpha-Plan-ReleaseSource) |  |  |
| deployment | [Plan.Deployment](#bytebase-api-v1alpha-Plan-Deployment) |  |  |






<a name="bytebase-api-v1alpha-Plan-ChangeDatabaseConfig"></a>

### Plan.ChangeDatabaseConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| target | [string](#string) |  | The resource name of the target. Format: instances/{instance-id}/databases/{database-name}. Format: projects/{project}/databaseGroups/{databaseGroup}. |
| sheet | [string](#string) |  | The resource name of the sheet. Format: projects/{project}/sheets/{sheet} |
| type | [Plan.ChangeDatabaseConfig.Type](#bytebase-api-v1alpha-Plan-ChangeDatabaseConfig-Type) |  |  |
| schema_version | [string](#string) |  | schema_version is parsed from file name. It is automatically generated in the UI workflow. |
| ghost_flags | [Plan.ChangeDatabaseConfig.GhostFlagsEntry](#bytebase-api-v1alpha-Plan-ChangeDatabaseConfig-GhostFlagsEntry) | repeated |  |
| pre_update_backup_detail | [Plan.ChangeDatabaseConfig.PreUpdateBackupDetail](#bytebase-api-v1alpha-Plan-ChangeDatabaseConfig-PreUpdateBackupDetail) | optional | If set, a backup of the modified data will be created automatically before any changes are applied. |






<a name="bytebase-api-v1alpha-Plan-ChangeDatabaseConfig-GhostFlagsEntry"></a>

### Plan.ChangeDatabaseConfig.GhostFlagsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-Plan-ChangeDatabaseConfig-PreUpdateBackupDetail"></a>

### Plan.ChangeDatabaseConfig.PreUpdateBackupDetail



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [string](#string) |  | The database for keeping the backup data. Format: instances/{instance}/databases/{database} |






<a name="bytebase-api-v1alpha-Plan-CreateDatabaseConfig"></a>

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






<a name="bytebase-api-v1alpha-Plan-Deployment"></a>

### Plan.Deployment



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| environments | [string](#string) | repeated | The environments deploy order. |
| database_group_mappings | [Plan.Deployment.DatabaseGroupMapping](#bytebase-api-v1alpha-Plan-Deployment-DatabaseGroupMapping) | repeated | The database group mapping. |






<a name="bytebase-api-v1alpha-Plan-Deployment-DatabaseGroupMapping"></a>

### Plan.Deployment.DatabaseGroupMapping



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database_group | [string](#string) |  | Format: projects/{project}/databaseGroups/{databaseGroup}. |
| databases | [string](#string) | repeated | Format: instances/{instance-id}/databases/{database-name}. |






<a name="bytebase-api-v1alpha-Plan-ExportDataConfig"></a>

### Plan.ExportDataConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| target | [string](#string) |  | The resource name of the target. Format: instances/{instance-id}/databases/{database-name} |
| sheet | [string](#string) |  | The resource name of the sheet. Format: projects/{project}/sheets/{sheet} |
| format | [ExportFormat](#bytebase-api-v1alpha-ExportFormat) |  | The format of the exported file. |
| password | [string](#string) | optional | The zip password provide by users. Leave it empty if no needs to encrypt the zip file. |






<a name="bytebase-api-v1alpha-Plan-PlanCheckRunStatusCountEntry"></a>

### Plan.PlanCheckRunStatusCountEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [int32](#int32) |  |  |






<a name="bytebase-api-v1alpha-Plan-ReleaseSource"></a>

### Plan.ReleaseSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| release | [string](#string) |  | The release. Format: projects/{project}/releases/{release} |






<a name="bytebase-api-v1alpha-Plan-Spec"></a>

### Plan.Spec



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| earliest_allowed_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | earliest_allowed_time the earliest execution time of the change. |
| id | [string](#string) |  | A UUID4 string that uniquely identifies the Spec. |
| spec_release_source | [Plan.SpecReleaseSource](#bytebase-api-v1alpha-Plan-SpecReleaseSource) |  |  |
| create_database_config | [Plan.CreateDatabaseConfig](#bytebase-api-v1alpha-Plan-CreateDatabaseConfig) |  |  |
| change_database_config | [Plan.ChangeDatabaseConfig](#bytebase-api-v1alpha-Plan-ChangeDatabaseConfig) |  |  |
| export_data_config | [Plan.ExportDataConfig](#bytebase-api-v1alpha-Plan-ExportDataConfig) |  |  |






<a name="bytebase-api-v1alpha-Plan-SpecReleaseSource"></a>

### Plan.SpecReleaseSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file | [string](#string) |  | Format: projects/{project}/releases/{release}/files/{file} {file} is URL path escaped. |






<a name="bytebase-api-v1alpha-Plan-Step"></a>

### Plan.Step



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  |  |
| specs | [Plan.Spec](#bytebase-api-v1alpha-Plan-Spec) | repeated |  |






<a name="bytebase-api-v1alpha-PlanCheckRun"></a>

### PlanCheckRun



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/plans/{plan}/planCheckRuns/{planCheckRun} |
| type | [PlanCheckRun.Type](#bytebase-api-v1alpha-PlanCheckRun-Type) |  |  |
| status | [PlanCheckRun.Status](#bytebase-api-v1alpha-PlanCheckRun-Status) |  |  |
| target | [string](#string) |  | Format: instances/{instance}/databases/{database} |
| sheet | [string](#string) |  | Format: project/{project}/sheets/{sheet} |
| results | [PlanCheckRun.Result](#bytebase-api-v1alpha-PlanCheckRun-Result) | repeated |  |
| error | [string](#string) |  | error is set if the Status is FAILED. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="bytebase-api-v1alpha-PlanCheckRun-Result"></a>

### PlanCheckRun.Result



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [PlanCheckRun.Result.Status](#bytebase-api-v1alpha-PlanCheckRun-Result-Status) |  |  |
| title | [string](#string) |  |  |
| content | [string](#string) |  |  |
| code | [int32](#int32) |  |  |
| sql_summary_report | [PlanCheckRun.Result.SqlSummaryReport](#bytebase-api-v1alpha-PlanCheckRun-Result-SqlSummaryReport) |  |  |
| sql_review_report | [PlanCheckRun.Result.SqlReviewReport](#bytebase-api-v1alpha-PlanCheckRun-Result-SqlReviewReport) |  |  |






<a name="bytebase-api-v1alpha-PlanCheckRun-Result-SqlReviewReport"></a>

### PlanCheckRun.Result.SqlReviewReport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| line | [int32](#int32) |  |  |
| column | [int32](#int32) |  |  |
| start_position | [Position](#bytebase-api-v1alpha-Position) |  | 1-based Position of the SQL statement. To supersede `line` and `column` above. |
| end_position | [Position](#bytebase-api-v1alpha-Position) |  |  |






<a name="bytebase-api-v1alpha-PlanCheckRun-Result-SqlSummaryReport"></a>

### PlanCheckRun.Result.SqlSummaryReport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| statement_types | [string](#string) | repeated | statement_types are the types of statements that are found in the sql. |
| affected_rows | [int32](#int32) |  |  |
| changed_resources | [ChangedResources](#bytebase-api-v1alpha-ChangedResources) |  |  |






<a name="bytebase-api-v1alpha-PreviewPlanRequest"></a>

### PreviewPlanRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The name of the project. Format: projects/{project} |
| release | [string](#string) |  | The release used for preview. |
| targets | [string](#string) | repeated | The targets to deploy. Can be database or databaseGroup. Format: projects/{project}/databaseGroups/{databaseGroup} instances/{instance}/databases/{database} |
| allow_out_of_order | [bool](#bool) |  |  |






<a name="bytebase-api-v1alpha-PreviewPlanResponse"></a>

### PreviewPlanResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| plan | [Plan](#bytebase-api-v1alpha-Plan) |  |  |
| out_of_order_files | [PreviewPlanResponse.DatabaseFiles](#bytebase-api-v1alpha-PreviewPlanResponse-DatabaseFiles) | repeated | The out of order files of each database. |
| applied_but_modified_files | [PreviewPlanResponse.DatabaseFiles](#bytebase-api-v1alpha-PreviewPlanResponse-DatabaseFiles) | repeated | The applied but modified files of each database. |






<a name="bytebase-api-v1alpha-PreviewPlanResponse-DatabaseFiles"></a>

### PreviewPlanResponse.DatabaseFiles



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [string](#string) |  |  |
| files | [string](#string) | repeated | Format: projects/{project}/releases/{release}/files/{file_id} |






<a name="bytebase-api-v1alpha-RunPlanChecksRequest"></a>

### RunPlanChecksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The plan to run plan checks. Format: projects/{project}/plans/{plan} |






<a name="bytebase-api-v1alpha-RunPlanChecksResponse"></a>

### RunPlanChecksResponse







<a name="bytebase-api-v1alpha-SearchPlansRequest"></a>

### SearchPlansRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of plans. Format: projects/{project} Use &#34;projects/-&#34; to list all plans from all projects. |
| page_size | [int32](#int32) |  | The maximum number of plans to return. The service may return fewer than this value. If unspecified, at most 10 plans will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `SearchPlans` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `SearchPlans` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter plans returned in the list. |






<a name="bytebase-api-v1alpha-SearchPlansResponse"></a>

### SearchPlansResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| plans | [Plan](#bytebase-api-v1alpha-Plan) | repeated | The plans from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-UpdatePlanRequest"></a>

### UpdatePlanRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| plan | [Plan](#bytebase-api-v1alpha-Plan) |  | The plan to update.

The plan&#39;s `name` field is used to identify the plan to update. Format: projects/{project}/plans/{plan} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 


<a name="bytebase-api-v1alpha-Plan-ChangeDatabaseConfig-Type"></a>

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



<a name="bytebase-api-v1alpha-PlanCheckRun-Result-Status"></a>

### PlanCheckRun.Result.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| ERROR | 1 |  |
| WARNING | 2 |  |
| SUCCESS | 3 |  |



<a name="bytebase-api-v1alpha-PlanCheckRun-Status"></a>

### PlanCheckRun.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| RUNNING | 1 |  |
| DONE | 2 |  |
| FAILED | 3 |  |
| CANCELED | 4 |  |



<a name="bytebase-api-v1alpha-PlanCheckRun-Type"></a>

### PlanCheckRun.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| DATABASE_STATEMENT_FAKE_ADVISE | 1 |  |
| DATABASE_STATEMENT_ADVISE | 3 |  |
| DATABASE_STATEMENT_SUMMARY_REPORT | 5 |  |
| DATABASE_CONNECT | 6 |  |
| DATABASE_GHOST_SYNC | 7 |  |


 

 


<a name="bytebase-api-v1alpha-PlanService"></a>

### PlanService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetPlan | [GetPlanRequest](#bytebase-api-v1alpha-GetPlanRequest) | [Plan](#bytebase-api-v1alpha-Plan) |  |
| ListPlans | [ListPlansRequest](#bytebase-api-v1alpha-ListPlansRequest) | [ListPlansResponse](#bytebase-api-v1alpha-ListPlansResponse) |  |
| SearchPlans | [SearchPlansRequest](#bytebase-api-v1alpha-SearchPlansRequest) | [SearchPlansResponse](#bytebase-api-v1alpha-SearchPlansResponse) | Search for plans that the caller has the bb.plans.get permission on and also satisfy the specified filter &amp; query. |
| CreatePlan | [CreatePlanRequest](#bytebase-api-v1alpha-CreatePlanRequest) | [Plan](#bytebase-api-v1alpha-Plan) |  |
| UpdatePlan | [UpdatePlanRequest](#bytebase-api-v1alpha-UpdatePlanRequest) | [Plan](#bytebase-api-v1alpha-Plan) | UpdatePlan updates the plan. The plan creator and the user with bb.plans.update permission on the project can update the plan. |
| ListPlanCheckRuns | [ListPlanCheckRunsRequest](#bytebase-api-v1alpha-ListPlanCheckRunsRequest) | [ListPlanCheckRunsResponse](#bytebase-api-v1alpha-ListPlanCheckRunsResponse) |  |
| RunPlanChecks | [RunPlanChecksRequest](#bytebase-api-v1alpha-RunPlanChecksRequest) | [RunPlanChecksResponse](#bytebase-api-v1alpha-RunPlanChecksResponse) |  |
| BatchCancelPlanCheckRuns | [BatchCancelPlanCheckRunsRequest](#bytebase-api-v1alpha-BatchCancelPlanCheckRunsRequest) | [BatchCancelPlanCheckRunsResponse](#bytebase-api-v1alpha-BatchCancelPlanCheckRunsResponse) |  |
| PreviewPlan | [PreviewPlanRequest](#bytebase-api-v1alpha-PreviewPlanRequest) | [PreviewPlanResponse](#bytebase-api-v1alpha-PreviewPlanResponse) |  |

 



<a name="api_v1alpha_project_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/project_service.proto



<a name="bytebase-api-v1alpha-Activity"></a>

### Activity
TODO(zp): move to activity later.






<a name="bytebase-api-v1alpha-AddWebhookRequest"></a>

### AddWebhookRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The name of the project to add the webhook to. Format: projects/{project} |
| webhook | [Webhook](#bytebase-api-v1alpha-Webhook) |  | The webhook to add. |






<a name="bytebase-api-v1alpha-BatchGetIamPolicyRequest"></a>

### BatchGetIamPolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| scope | [string](#string) |  | The scope of the batch get. Typically it&#39;s &#34;projects/-&#34;. |
| names | [string](#string) | repeated |  |






<a name="bytebase-api-v1alpha-BatchGetIamPolicyResponse"></a>

### BatchGetIamPolicyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy_results | [BatchGetIamPolicyResponse.PolicyResult](#bytebase-api-v1alpha-BatchGetIamPolicyResponse-PolicyResult) | repeated |  |






<a name="bytebase-api-v1alpha-BatchGetIamPolicyResponse-PolicyResult"></a>

### BatchGetIamPolicyResponse.PolicyResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  |  |
| policy | [IamPolicy](#bytebase-api-v1alpha-IamPolicy) |  |  |






<a name="bytebase-api-v1alpha-CreateProjectRequest"></a>

### CreateProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [Project](#bytebase-api-v1alpha-Project) |  | The project to create. |
| project_id | [string](#string) |  | The ID to use for the project, which will become the final component of the project&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |






<a name="bytebase-api-v1alpha-DeleteProjectRequest"></a>

### DeleteProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the project to delete. Format: projects/{project} |
| force | [bool](#bool) |  | If set to true, any databases and sheets from this project will also be moved to default project, and all open issues will be closed. |






<a name="bytebase-api-v1alpha-GetProjectRequest"></a>

### GetProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the project to retrieve. Format: projects/{project} |






<a name="bytebase-api-v1alpha-Label"></a>

### Label



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [string](#string) |  |  |
| color | [string](#string) |  |  |
| group | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-ListProjectsRequest"></a>

### ListProjectsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of projects to return. The service may return fewer than this value. If unspecified, at most 10 projects will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListProjects` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListProjects` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Show deleted projects if specified. |
| filter | [string](#string) |  | Filter the project. Check filter for SearchProjectsRequest for details. |






<a name="bytebase-api-v1alpha-ListProjectsResponse"></a>

### ListProjectsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| projects | [Project](#bytebase-api-v1alpha-Project) | repeated | The projects from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-Project"></a>

### Project



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the project. Format: projects/{project} |
| state | [State](#bytebase-api-v1alpha-State) |  |  |
| title | [string](#string) |  | The title or name of a project. It&#39;s not unique within the workspace. |
| webhooks | [Webhook](#bytebase-api-v1alpha-Webhook) | repeated |  |
| data_classification_config_id | [string](#string) |  |  |
| issue_labels | [Label](#bytebase-api-v1alpha-Label) | repeated |  |
| force_issue_labels | [bool](#bool) |  | Force issue labels to be used when creating an issue. |
| allow_modify_statement | [bool](#bool) |  | Allow modifying statement after issue is created. |
| auto_resolve_issue | [bool](#bool) |  | Enable auto resolve issue. |
| enforce_issue_title | [bool](#bool) |  | Enforce issue title created by user instead of generated by Bytebase. |
| auto_enable_backup | [bool](#bool) |  | Whether to automatically enable backup. |
| skip_backup_errors | [bool](#bool) |  | Whether to skip backup errors and continue the data migration. |
| postgres_database_tenant_mode | [bool](#bool) |  | Whether to enable the database tenant mode for PostgreSQL. If enabled, the issue will be created with the pre-appended &#34;set role &lt;db_owner&gt;&#34; statement. |
| allow_self_approval | [bool](#bool) |  | Whether to allow the issue creator to self-approve the issue. |
| execution_retry_policy | [Project.ExecutionRetryPolicy](#bytebase-api-v1alpha-Project-ExecutionRetryPolicy) |  | Execution retry policy for the task run. |
| ci_sampling_size | [int32](#int32) |  | The maximum databases of rows to sample during CI data validation. Without specification, sampling is disabled, resulting in a full validation. |






<a name="bytebase-api-v1alpha-Project-ExecutionRetryPolicy"></a>

### Project.ExecutionRetryPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| maximum_retries | [int32](#int32) |  | The maximum number of retries for the lock timeout issue. |






<a name="bytebase-api-v1alpha-RemoveWebhookRequest"></a>

### RemoveWebhookRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| webhook | [Webhook](#bytebase-api-v1alpha-Webhook) |  | The webhook to remove. Identified by its url. |






<a name="bytebase-api-v1alpha-SearchProjectsRequest"></a>

### SearchProjectsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| show_deleted | [bool](#bool) |  | Show deleted projects if specified. |
| filter | [string](#string) |  | Filter the project. Supported filters: - name - resource_id - exclude_default: if not include the default project. - state

For example: name = &#34;project name&#34; name.matches(&#34;project name&#34;) resource_id = &#34;project id&#34; resource_id.matches(&#34;project id&#34;) exclude_default == true state == &#34;DELETED&#34; You can combine filter conditions like: name = &#34;project name&#34; &amp;&amp; resource_id.matches(&#34;project id&#34;) name.matches(&#34;project name&#34;) || resource_id = &#34;project id&#34; |
| page_size | [int32](#int32) |  | The maximum number of projects to return. The service may return fewer than this value. If unspecified, at most 10 projects will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `SearchProjects` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `SearchProjects` must match the call that provided the page token. |






<a name="bytebase-api-v1alpha-SearchProjectsResponse"></a>

### SearchProjectsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| projects | [Project](#bytebase-api-v1alpha-Project) | repeated | The projects from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-TestWebhookRequest"></a>

### TestWebhookRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The name of the project which owns the webhook to test. Format: projects/{project} |
| webhook | [Webhook](#bytebase-api-v1alpha-Webhook) |  | The webhook to test. Identified by its url. |






<a name="bytebase-api-v1alpha-TestWebhookResponse"></a>

### TestWebhookResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| error | [string](#string) |  | The result of the test, empty if the test is successful. |






<a name="bytebase-api-v1alpha-UndeleteProjectRequest"></a>

### UndeleteProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the deleted project. Format: projects/{project} |






<a name="bytebase-api-v1alpha-UpdateProjectRequest"></a>

### UpdateProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [Project](#bytebase-api-v1alpha-Project) |  | The project to update.

The project&#39;s `name` field is used to identify the project to update. Format: projects/{project} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






<a name="bytebase-api-v1alpha-UpdateWebhookRequest"></a>

### UpdateWebhookRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| webhook | [Webhook](#bytebase-api-v1alpha-Webhook) |  | The webhook to modify. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






<a name="bytebase-api-v1alpha-Webhook"></a>

### Webhook



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | name is the name of the webhook, generated by the server. format: projects/{project}/webhooks/{webhook} |
| type | [Webhook.Type](#bytebase-api-v1alpha-Webhook-Type) |  | type is the type of the webhook. |
| title | [string](#string) |  | title is the title of the webhook. |
| url | [string](#string) |  | url is the url of the webhook, should be unique within the project. |
| direct_message | [bool](#bool) |  | if direct_message is set, the notification is sent directly to the persons and url will be ignored. IM integration setting should be set for this function to work. |
| notification_types | [Activity.Type](#bytebase-api-v1alpha-Activity-Type) | repeated | notification_types is the list of activities types that the webhook is interested in. Bytebase will only send notifications to the webhook if the activity type is in the list. It should not be empty, and should be a subset of the following: - TYPE_ISSUE_CREATED - TYPE_ISSUE_STATUS_UPDATE - TYPE_ISSUE_PIPELINE_STAGE_UPDATE - TYPE_ISSUE_PIPELINE_TASK_STATUS_UPDATE - TYPE_ISSUE_FIELD_UPDATE - TYPE_ISSUE_COMMENT_CREATE |





 


<a name="bytebase-api-v1alpha-Activity-Type"></a>

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
| TYPE_ISSUE_FIELD_UPDATE | 3 | TYPE_ISSUE_FIELD_UPDATE represents updating the issue field, likes title, description, etc. |
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



<a name="bytebase-api-v1alpha-Webhook-Type"></a>

### Webhook.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| SLACK | 1 |  |
| DISCORD | 2 |  |
| TEAMS | 3 |  |
| DINGTALK | 4 |  |
| FEISHU | 5 |  |
| WECOM | 6 |  |
| LARK | 8 |  |


 

 


<a name="bytebase-api-v1alpha-ProjectService"></a>

### ProjectService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetProject | [GetProjectRequest](#bytebase-api-v1alpha-GetProjectRequest) | [Project](#bytebase-api-v1alpha-Project) |  |
| ListProjects | [ListProjectsRequest](#bytebase-api-v1alpha-ListProjectsRequest) | [ListProjectsResponse](#bytebase-api-v1alpha-ListProjectsResponse) |  |
| SearchProjects | [SearchProjectsRequest](#bytebase-api-v1alpha-SearchProjectsRequest) | [SearchProjectsResponse](#bytebase-api-v1alpha-SearchProjectsResponse) |  |
| CreateProject | [CreateProjectRequest](#bytebase-api-v1alpha-CreateProjectRequest) | [Project](#bytebase-api-v1alpha-Project) |  |
| UpdateProject | [UpdateProjectRequest](#bytebase-api-v1alpha-UpdateProjectRequest) | [Project](#bytebase-api-v1alpha-Project) |  |
| DeleteProject | [DeleteProjectRequest](#bytebase-api-v1alpha-DeleteProjectRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| UndeleteProject | [UndeleteProjectRequest](#bytebase-api-v1alpha-UndeleteProjectRequest) | [Project](#bytebase-api-v1alpha-Project) |  |
| GetIamPolicy | [GetIamPolicyRequest](#bytebase-api-v1alpha-GetIamPolicyRequest) | [IamPolicy](#bytebase-api-v1alpha-IamPolicy) |  |
| BatchGetIamPolicy | [BatchGetIamPolicyRequest](#bytebase-api-v1alpha-BatchGetIamPolicyRequest) | [BatchGetIamPolicyResponse](#bytebase-api-v1alpha-BatchGetIamPolicyResponse) | Deprecated. |
| SetIamPolicy | [SetIamPolicyRequest](#bytebase-api-v1alpha-SetIamPolicyRequest) | [IamPolicy](#bytebase-api-v1alpha-IamPolicy) |  |
| AddWebhook | [AddWebhookRequest](#bytebase-api-v1alpha-AddWebhookRequest) | [Project](#bytebase-api-v1alpha-Project) |  |
| UpdateWebhook | [UpdateWebhookRequest](#bytebase-api-v1alpha-UpdateWebhookRequest) | [Project](#bytebase-api-v1alpha-Project) |  |
| RemoveWebhook | [RemoveWebhookRequest](#bytebase-api-v1alpha-RemoveWebhookRequest) | [Project](#bytebase-api-v1alpha-Project) |  |
| TestWebhook | [TestWebhookRequest](#bytebase-api-v1alpha-TestWebhookRequest) | [TestWebhookResponse](#bytebase-api-v1alpha-TestWebhookResponse) |  |

 



<a name="api_v1alpha_sql_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/sql_service.proto



<a name="bytebase-api-v1alpha-AICompletionRequest"></a>

### AICompletionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| messages | [AICompletionRequest.Message](#bytebase-api-v1alpha-AICompletionRequest-Message) | repeated |  |






<a name="bytebase-api-v1alpha-AICompletionRequest-Message"></a>

### AICompletionRequest.Message



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [string](#string) |  |  |
| content | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-AICompletionResponse"></a>

### AICompletionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| candidates | [AICompletionResponse.Candidate](#bytebase-api-v1alpha-AICompletionResponse-Candidate) | repeated | candidates is used for results with multiple choices and candidates. Used for OpenAI and Gemini. |






<a name="bytebase-api-v1alpha-AICompletionResponse-Candidate"></a>

### AICompletionResponse.Candidate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| content | [AICompletionResponse.Candidate.Content](#bytebase-api-v1alpha-AICompletionResponse-Candidate-Content) |  |  |






<a name="bytebase-api-v1alpha-AICompletionResponse-Candidate-Content"></a>

### AICompletionResponse.Candidate.Content



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parts | [AICompletionResponse.Candidate.Content.Part](#bytebase-api-v1alpha-AICompletionResponse-Candidate-Content-Part) | repeated | parts is used for a result content with multiple parts. |






<a name="bytebase-api-v1alpha-AICompletionResponse-Candidate-Content-Part"></a>

### AICompletionResponse.Candidate.Content.Part



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| text | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-AdminExecuteRequest"></a>

### AdminExecuteRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the instance name to execute the query against. Format: instances/{instance}/databases/{databaseName} |
| statement | [string](#string) |  | The SQL statement to execute. |
| limit | [int32](#int32) |  | The maximum number of rows to return. |
| schema | [string](#string) | optional | The default schema to execute the statement. Equals to the current schema in Oracle and search path in Postgres. |
| container | [string](#string) | optional | Container is the container name to execute the query against, used for CosmosDB only. |






<a name="bytebase-api-v1alpha-AdminExecuteResponse"></a>

### AdminExecuteResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [QueryResult](#bytebase-api-v1alpha-QueryResult) | repeated | The query results. |






<a name="bytebase-api-v1alpha-Advice"></a>

### Advice



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [Advice.Status](#bytebase-api-v1alpha-Advice-Status) |  | The advice status. |
| code | [int32](#int32) |  | The advice code. |
| title | [string](#string) |  | The advice title. |
| content | [string](#string) |  | The advice content. |
| start_position | [Position](#bytebase-api-v1alpha-Position) |  | The start_position is inclusive and the end_position is exclusive. TODO: use range instead |
| end_position | [Position](#bytebase-api-v1alpha-Position) |  |  |






<a name="bytebase-api-v1alpha-CheckRequest"></a>

### CheckRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The database name to check against. Format: instances/{instance}/databases/{database} |
| statement | [string](#string) |  |  |
| change_type | [CheckRequest.ChangeType](#bytebase-api-v1alpha-CheckRequest-ChangeType) |  |  |






<a name="bytebase-api-v1alpha-CheckResponse"></a>

### CheckResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| advices | [Advice](#bytebase-api-v1alpha-Advice) | repeated |  |
| affected_rows | [int32](#int32) |  | The count of affected rows of the statement on the target database. |






<a name="bytebase-api-v1alpha-DiffMetadataRequest"></a>

### DiffMetadataRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| source_metadata | [DatabaseMetadata](#bytebase-api-v1alpha-DatabaseMetadata) |  | The metadata of the source schema. |
| target_metadata | [DatabaseMetadata](#bytebase-api-v1alpha-DatabaseMetadata) |  | The metadata of the target schema. |
| source_catalog | [DatabaseCatalog](#bytebase-api-v1alpha-DatabaseCatalog) |  |  |
| target_catalog | [DatabaseCatalog](#bytebase-api-v1alpha-DatabaseCatalog) |  |  |
| engine | [Engine](#bytebase-api-v1alpha-Engine) |  | The database engine of the schema. |
| classification_from_config | [bool](#bool) |  | If false, we will build the raw common by classification in database config. |






<a name="bytebase-api-v1alpha-DiffMetadataResponse"></a>

### DiffMetadataResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| diff | [string](#string) |  | The diff of the metadata. |






<a name="bytebase-api-v1alpha-ExportRequest"></a>

### ExportRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the instance name to execute the query against. Format: instances/{instance}/databases/{databaseName} Format: projects/{project}/issues/{issue} for data export issue. |
| statement | [string](#string) |  | The SQL statement to execute. |
| limit | [int32](#int32) |  | The maximum number of rows to return. |
| format | [ExportFormat](#bytebase-api-v1alpha-ExportFormat) |  | The export format. |
| admin | [bool](#bool) |  | The admin is used for workspace owner and DBA for exporting data from SQL Editor Admin mode. The exported data is not masked. |
| password | [string](#string) |  | The zip password provide by users. |
| data_source_id | [string](#string) |  | The id of data source. It is used for querying admin data source even if the instance has read-only data sources. Or it can be used to query a specific read-only data source. |






<a name="bytebase-api-v1alpha-ExportResponse"></a>

### ExportResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| content | [bytes](#bytes) |  | The export file content. |






<a name="bytebase-api-v1alpha-ParseMyBatisMapperRequest"></a>

### ParseMyBatisMapperRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| content | [bytes](#bytes) |  |  |






<a name="bytebase-api-v1alpha-ParseMyBatisMapperResponse"></a>

### ParseMyBatisMapperResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| statements | [string](#string) | repeated |  |






<a name="bytebase-api-v1alpha-PrettyRequest"></a>

### PrettyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| engine | [Engine](#bytebase-api-v1alpha-Engine) |  |  |
| current_schema | [string](#string) |  | The SDL format SQL schema information that was dumped from a database engine. This information will be sorted to match the order of statements in the userSchema. |
| expected_schema | [string](#string) |  | The expected SDL schema. This schema will be checked for correctness and normalized. |






<a name="bytebase-api-v1alpha-PrettyResponse"></a>

### PrettyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| current_schema | [string](#string) |  | The pretty-formatted version of current schema. |
| expected_schema | [string](#string) |  | The expected SDL schema after normalizing. |






<a name="bytebase-api-v1alpha-QueryHistory"></a>

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
| type | [QueryHistory.Type](#bytebase-api-v1alpha-QueryHistory-Type) |  |  |






<a name="bytebase-api-v1alpha-QueryOption"></a>

### QueryOption



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| redis_run_commands_on | [QueryOption.RedisRunCommandsOn](#bytebase-api-v1alpha-QueryOption-RedisRunCommandsOn) |  |  |






<a name="bytebase-api-v1alpha-QueryRequest"></a>

### QueryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the instance name to execute the query against. Format: instances/{instance}/databases/{databaseName} |
| statement | [string](#string) |  | The SQL statement to execute. |
| limit | [int32](#int32) |  | The maximum number of rows to return. |
| data_source_id | [string](#string) |  | The id of data source. It is used for querying admin data source even if the instance has read-only data sources. Or it can be used to query a specific read-only data source. |
| explain | [bool](#bool) |  | Explain the statement. |
| schema | [string](#string) | optional | The default schema to search objects. Equals to the current schema in Oracle and search path in Postgres. |
| query_option | [QueryOption](#bytebase-api-v1alpha-QueryOption) |  |  |
| container | [string](#string) | optional | Container is the container name to execute the query against, used for CosmosDB only. |






<a name="bytebase-api-v1alpha-QueryResponse"></a>

### QueryResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [QueryResult](#bytebase-api-v1alpha-QueryResult) | repeated | The query results. |






<a name="bytebase-api-v1alpha-QueryResult"></a>

### QueryResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| column_names | [string](#string) | repeated | Column names of the query result. |
| column_type_names | [string](#string) | repeated | Column types of the query result. The types come from the Golang SQL driver. |
| rows | [QueryRow](#bytebase-api-v1alpha-QueryRow) | repeated | Rows of the query result. |
| rows_count | [int64](#int64) |  |  |
| masked | [bool](#bool) | repeated | Columns are masked or not. |
| sensitive | [bool](#bool) | repeated | Columns are sensitive or not. |
| error | [string](#string) |  | The error message if the query failed. |
| latency | [google.protobuf.Duration](#google-protobuf-Duration) |  | The time it takes to execute the query. |
| statement | [string](#string) |  | The query statement for the result. |
| postgres_error | [QueryResult.PostgresError](#bytebase-api-v1alpha-QueryResult-PostgresError) |  |  |
| allow_export | [bool](#bool) |  | The query result is allowed to be exported or not. |






<a name="bytebase-api-v1alpha-QueryResult-PostgresError"></a>

### QueryResult.PostgresError
refer https://www.postgresql.org/docs/11/protocol-error-fields.html
for field description.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| severity | [string](#string) |  |  |
| code | [string](#string) |  |  |
| message | [string](#string) |  |  |
| detail | [string](#string) |  |  |
| hint | [string](#string) |  |  |
| position | [int32](#int32) |  |  |
| internal_position | [int32](#int32) |  |  |
| internal_query | [string](#string) |  |  |
| where | [string](#string) |  |  |
| schema_name | [string](#string) |  |  |
| table_name | [string](#string) |  |  |
| column_name | [string](#string) |  |  |
| data_type_name | [string](#string) |  |  |
| constraint_name | [string](#string) |  |  |
| file | [string](#string) |  |  |
| line | [int32](#int32) |  |  |
| routine | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-QueryRow"></a>

### QueryRow



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| values | [RowValue](#bytebase-api-v1alpha-RowValue) | repeated | Row values of the query result. |






<a name="bytebase-api-v1alpha-RowValue"></a>

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
| timestamp_value | [RowValue.Timestamp](#bytebase-api-v1alpha-RowValue-Timestamp) |  | timestamp_value is used for the timestamp without time zone data type, meaning it only includes the timestamp without any time zone or location info. Although it may be expressed as a UTC value, it should be seen as a timestamp missing location context. |
| timestamp_tz_value | [RowValue.TimestampTZ](#bytebase-api-v1alpha-RowValue-TimestampTZ) |  | timestamp_tz_value is used for the timestamptz data type, which accurately represents the timestamp with location information. |






<a name="bytebase-api-v1alpha-RowValue-Timestamp"></a>

### RowValue.Timestamp



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| google_timestamp | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| accuracy | [int32](#int32) |  | The accuracy is the number of digits after the decimal point. |






<a name="bytebase-api-v1alpha-RowValue-TimestampTZ"></a>

### RowValue.TimestampTZ



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| google_timestamp | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| zone | [string](#string) |  | Zone is the time zone abbreviations in timezone database such as &#34;PDT&#34;, &#34;PST&#34;. https://en.wikipedia.org/wiki/List_of_tz_database_time_zones We retrieve the time zone information from the timestamptz field in the database. A timestamp is in UTC or epoch time, and with zone info, we can convert it to a local time string. Zone and offset are returned by time.Time.Zone() |
| offset | [int32](#int32) |  | The offset is in seconds east of UTC |
| accuracy | [int32](#int32) |  |  |






<a name="bytebase-api-v1alpha-SearchQueryHistoriesRequest"></a>

### SearchQueryHistoriesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of histories to return. The service may return fewer than this value. If unspecified, at most 10 history entries will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListQueryHistory` call. Provide this to retrieve the subsequent page. |
| filter | [string](#string) |  | Filter is the filter to apply on the search query history Supported filter: - project - database - instance - type - statement

For example: project == &#34;projects/{project}&#34; database == &#34;instances/{instance}/databases/{database}&#34; instance == &#34;instances/{instance}&#34; type == &#34;QUERY&#34; type == &#34;EXPORT&#34; statement.matches(&#34;select&#34;) type == &#34;QUERY&#34; &amp;&amp; statement.matches(&#34;select&#34;) |






<a name="bytebase-api-v1alpha-SearchQueryHistoriesResponse"></a>

### SearchQueryHistoriesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| query_histories | [QueryHistory](#bytebase-api-v1alpha-QueryHistory) | repeated | The list of history. |
| next_page_token | [string](#string) |  | A token to retrieve next page of history. Pass this value in the page_token field in the subsequent call to `ListQueryHistory` method to retrieve the next page of history. |





 


<a name="bytebase-api-v1alpha-Advice-Status"></a>

### Advice.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 | Unspecified. |
| SUCCESS | 1 |  |
| WARNING | 2 |  |
| ERROR | 3 |  |



<a name="bytebase-api-v1alpha-CheckRequest-ChangeType"></a>

### CheckRequest.ChangeType


| Name | Number | Description |
| ---- | ------ | ----------- |
| CHANGE_TYPE_UNSPECIFIED | 0 |  |
| DDL | 1 |  |
| DDL_GHOST | 2 |  |
| DML | 3 |  |
| SQL_EDITOR | 4 |  |



<a name="bytebase-api-v1alpha-QueryHistory-Type"></a>

### QueryHistory.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| QUERY | 1 |  |
| EXPORT | 2 |  |



<a name="bytebase-api-v1alpha-QueryOption-RedisRunCommandsOn"></a>

### QueryOption.RedisRunCommandsOn


| Name | Number | Description |
| ---- | ------ | ----------- |
| REDIS_RUN_COMMANDS_ON_UNSPECIFIED | 0 | UNSPECIFIED defaults to SINGLE_NODE. |
| SINGLE_NODE | 1 |  |
| ALL_NODES | 2 |  |


 

 


<a name="bytebase-api-v1alpha-SQLService"></a>

### SQLService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Query | [QueryRequest](#bytebase-api-v1alpha-QueryRequest) | [QueryResponse](#bytebase-api-v1alpha-QueryResponse) |  |
| AdminExecute | [AdminExecuteRequest](#bytebase-api-v1alpha-AdminExecuteRequest) stream | [AdminExecuteResponse](#bytebase-api-v1alpha-AdminExecuteResponse) stream |  |
| SearchQueryHistories | [SearchQueryHistoriesRequest](#bytebase-api-v1alpha-SearchQueryHistoriesRequest) | [SearchQueryHistoriesResponse](#bytebase-api-v1alpha-SearchQueryHistoriesResponse) | SearchQueryHistories searches query histories for the caller. |
| Export | [ExportRequest](#bytebase-api-v1alpha-ExportRequest) | [ExportResponse](#bytebase-api-v1alpha-ExportResponse) |  |
| Check | [CheckRequest](#bytebase-api-v1alpha-CheckRequest) | [CheckResponse](#bytebase-api-v1alpha-CheckResponse) |  |
| ParseMyBatisMapper | [ParseMyBatisMapperRequest](#bytebase-api-v1alpha-ParseMyBatisMapperRequest) | [ParseMyBatisMapperResponse](#bytebase-api-v1alpha-ParseMyBatisMapperResponse) |  |
| Pretty | [PrettyRequest](#bytebase-api-v1alpha-PrettyRequest) | [PrettyResponse](#bytebase-api-v1alpha-PrettyResponse) |  |
| DiffMetadata | [DiffMetadataRequest](#bytebase-api-v1alpha-DiffMetadataRequest) | [DiffMetadataResponse](#bytebase-api-v1alpha-DiffMetadataResponse) |  |
| AICompletion | [AICompletionRequest](#bytebase-api-v1alpha-AICompletionRequest) | [AICompletionResponse](#bytebase-api-v1alpha-AICompletionResponse) |  |

 



<a name="api_v1alpha_release_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/release_service.proto



<a name="bytebase-api-v1alpha-CheckReleaseRequest"></a>

### CheckReleaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{project} |
| release | [Release](#bytebase-api-v1alpha-Release) |  | The release to check. |
| targets | [string](#string) | repeated | The targets to dry-run the release. Can be database or databaseGroup. Format: projects/{project}/databaseGroups/{databaseGroup} instances/{instance}/databases/{database} |






<a name="bytebase-api-v1alpha-CheckReleaseResponse"></a>

### CheckReleaseResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [CheckReleaseResponse.CheckResult](#bytebase-api-v1alpha-CheckReleaseResponse-CheckResult) | repeated |  |
| affected_rows | [int32](#int32) |  | The affected rows of the check. |
| risk_level | [CheckReleaseResponse.RiskLevel](#bytebase-api-v1alpha-CheckReleaseResponse-RiskLevel) |  | The aggregated risk level of the check. |






<a name="bytebase-api-v1alpha-CheckReleaseResponse-CheckResult"></a>

### CheckReleaseResponse.CheckResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file | [string](#string) |  | The file path that is being checked. |
| target | [string](#string) |  | The target that the check is performed on. Should be a database. Format: instances/{instance}/databases/{database} |
| advices | [Advice](#bytebase-api-v1alpha-Advice) | repeated | The list of advice for the file and the target. |
| affected_rows | [int32](#int32) |  | The count of affected rows of the statement on the target. |
| risk_level | [CheckReleaseResponse.RiskLevel](#bytebase-api-v1alpha-CheckReleaseResponse-RiskLevel) |  | The risk level of the statement on the target. |






<a name="bytebase-api-v1alpha-CreateReleaseRequest"></a>

### CreateReleaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{project} |
| release | [Release](#bytebase-api-v1alpha-Release) |  | The release to create. |






<a name="bytebase-api-v1alpha-DeleteReleaseRequest"></a>

### DeleteReleaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the release to delete. Format: projects/{project}/releases/{release} |






<a name="bytebase-api-v1alpha-GetReleaseRequest"></a>

### GetReleaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/releases/{release} |






<a name="bytebase-api-v1alpha-ListReleasesRequest"></a>

### ListReleasesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{project} |
| page_size | [int32](#int32) |  | The maximum number of releases to return. The service may return fewer than this value. If unspecified, at most 10 releases will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListReleases` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListReleases` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Show deleted releases if specified. |






<a name="bytebase-api-v1alpha-ListReleasesResponse"></a>

### ListReleasesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| releases | [Release](#bytebase-api-v1alpha-Release) | repeated |  |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-Release"></a>

### Release



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/releases/{release} |
| title | [string](#string) |  |  |
| files | [Release.File](#bytebase-api-v1alpha-Release-File) | repeated |  |
| vcs_source | [Release.VCSSource](#bytebase-api-v1alpha-Release-VCSSource) |  |  |
| creator | [string](#string) |  | Format: users/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| state | [State](#bytebase-api-v1alpha-State) |  |  |






<a name="bytebase-api-v1alpha-Release-File"></a>

### Release.File



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | The unique identifier for the file. |
| path | [string](#string) |  | The path of the file. e.g. `2.2/V0001_create_table.sql`. |
| sheet | [string](#string) |  | The sheet that holds the content. Format: projects/{project}/sheets/{sheet} |
| sheet_sha256 | [string](#string) |  | The SHA256 hash value of the sheet. |
| type | [ReleaseFileType](#bytebase-api-v1alpha-ReleaseFileType) |  |  |
| version | [string](#string) |  |  |
| change_type | [Release.File.ChangeType](#bytebase-api-v1alpha-Release-File-ChangeType) |  |  |
| statement | [bytes](#bytes) |  | The statement is used for preview or check purpose. |
| statement_size | [int64](#int64) |  |  |






<a name="bytebase-api-v1alpha-Release-VCSSource"></a>

### Release.VCSSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vcs_type | [VCSType](#bytebase-api-v1alpha-VCSType) |  |  |
| url | [string](#string) |  | The url link to the e.g. GitHub commit or pull request. |






<a name="bytebase-api-v1alpha-UndeleteReleaseRequest"></a>

### UndeleteReleaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the deleted release. Format: projects/{project}/releases/{release} |






<a name="bytebase-api-v1alpha-UpdateReleaseRequest"></a>

### UpdateReleaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| release | [Release](#bytebase-api-v1alpha-Release) |  | The release to update. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to be updated. |





 


<a name="bytebase-api-v1alpha-CheckReleaseResponse-RiskLevel"></a>

### CheckReleaseResponse.RiskLevel


| Name | Number | Description |
| ---- | ------ | ----------- |
| RISK_LEVEL_UNSPECIFIED | 0 |  |
| LOW | 1 |  |
| MODERATE | 2 |  |
| HIGH | 3 |  |



<a name="bytebase-api-v1alpha-Release-File-ChangeType"></a>

### Release.File.ChangeType


| Name | Number | Description |
| ---- | ------ | ----------- |
| CHANGE_TYPE_UNSPECIFIED | 0 |  |
| DDL | 1 |  |
| DDL_GHOST | 2 |  |
| DML | 3 |  |



<a name="bytebase-api-v1alpha-ReleaseFileType"></a>

### ReleaseFileType


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| VERSIONED | 1 |  |


 

 


<a name="bytebase-api-v1alpha-ReleaseService"></a>

### ReleaseService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetRelease | [GetReleaseRequest](#bytebase-api-v1alpha-GetReleaseRequest) | [Release](#bytebase-api-v1alpha-Release) |  |
| ListReleases | [ListReleasesRequest](#bytebase-api-v1alpha-ListReleasesRequest) | [ListReleasesResponse](#bytebase-api-v1alpha-ListReleasesResponse) |  |
| CreateRelease | [CreateReleaseRequest](#bytebase-api-v1alpha-CreateReleaseRequest) | [Release](#bytebase-api-v1alpha-Release) |  |
| UpdateRelease | [UpdateReleaseRequest](#bytebase-api-v1alpha-UpdateReleaseRequest) | [Release](#bytebase-api-v1alpha-Release) |  |
| DeleteRelease | [DeleteReleaseRequest](#bytebase-api-v1alpha-DeleteReleaseRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |
| UndeleteRelease | [UndeleteReleaseRequest](#bytebase-api-v1alpha-UndeleteReleaseRequest) | [Release](#bytebase-api-v1alpha-Release) |  |
| CheckRelease | [CheckReleaseRequest](#bytebase-api-v1alpha-CheckReleaseRequest) | [CheckReleaseResponse](#bytebase-api-v1alpha-CheckReleaseResponse) |  |

 



<a name="api_v1alpha_review_config_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/review_config_service.proto



<a name="bytebase-api-v1alpha-CreateReviewConfigRequest"></a>

### CreateReviewConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| review_config | [ReviewConfig](#bytebase-api-v1alpha-ReviewConfig) |  | The sql review to create. |






<a name="bytebase-api-v1alpha-DeleteReviewConfigRequest"></a>

### DeleteReviewConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the sql review to delete. Format: reviewConfigs/{reviewConfig} |






<a name="bytebase-api-v1alpha-GetReviewConfigRequest"></a>

### GetReviewConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the sql review to retrieve. Format: reviewConfigs/{reviewConfig} |






<a name="bytebase-api-v1alpha-ListReviewConfigsRequest"></a>

### ListReviewConfigsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Not used. The maximum number of sql review to return. The service may return fewer than this value. If unspecified, at most 10 sql review will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListReviewConfigs` must match the call that provided the page token. |






<a name="bytebase-api-v1alpha-ListReviewConfigsResponse"></a>

### ListReviewConfigsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| review_configs | [ReviewConfig](#bytebase-api-v1alpha-ReviewConfig) | repeated | The sql review from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-ReviewConfig"></a>

### ReviewConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the sql review to retrieve. Format: reviewConfigs/{reviewConfig} |
| title | [string](#string) |  |  |
| enabled | [bool](#bool) |  |  |
| rules | [SQLReviewRule](#bytebase-api-v1alpha-SQLReviewRule) | repeated |  |
| resources | [string](#string) | repeated | resources using the config. Format: {resurce}/{resource id}, for example, environments/test. |






<a name="bytebase-api-v1alpha-UpdateReviewConfigRequest"></a>

### UpdateReviewConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| review_config | [ReviewConfig](#bytebase-api-v1alpha-ReviewConfig) |  | The sql review to update.

The name field is used to identify the sql review to update. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the config is not found, a new config will be created. In this situation, `update_mask` is ignored. |





 

 

 


<a name="bytebase-api-v1alpha-ReviewConfigService"></a>

### ReviewConfigService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateReviewConfig | [CreateReviewConfigRequest](#bytebase-api-v1alpha-CreateReviewConfigRequest) | [ReviewConfig](#bytebase-api-v1alpha-ReviewConfig) |  |
| ListReviewConfigs | [ListReviewConfigsRequest](#bytebase-api-v1alpha-ListReviewConfigsRequest) | [ListReviewConfigsResponse](#bytebase-api-v1alpha-ListReviewConfigsResponse) |  |
| GetReviewConfig | [GetReviewConfigRequest](#bytebase-api-v1alpha-GetReviewConfigRequest) | [ReviewConfig](#bytebase-api-v1alpha-ReviewConfig) |  |
| UpdateReviewConfig | [UpdateReviewConfigRequest](#bytebase-api-v1alpha-UpdateReviewConfigRequest) | [ReviewConfig](#bytebase-api-v1alpha-ReviewConfig) |  |
| DeleteReviewConfig | [DeleteReviewConfigRequest](#bytebase-api-v1alpha-DeleteReviewConfigRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="api_v1alpha_risk_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/risk_service.proto



<a name="bytebase-api-v1alpha-CreateRiskRequest"></a>

### CreateRiskRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| risk | [Risk](#bytebase-api-v1alpha-Risk) |  | The risk to create. |






<a name="bytebase-api-v1alpha-DeleteRiskRequest"></a>

### DeleteRiskRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the risk to delete. Format: risks/{risk} |






<a name="bytebase-api-v1alpha-ListRisksRequest"></a>

### ListRisksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Not used. The maximum number of risks to return. The service may return fewer than this value. If unspecified, at most 10 risks will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListRisks` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `LiskRisks` must match the call that provided the page token. |






<a name="bytebase-api-v1alpha-ListRisksResponse"></a>

### ListRisksResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| risks | [Risk](#bytebase-api-v1alpha-Risk) | repeated |  |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-Risk"></a>

### Risk



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: risks/{risk} |
| source | [Risk.Source](#bytebase-api-v1alpha-Risk-Source) |  |  |
| title | [string](#string) |  |  |
| level | [int32](#int32) |  |  |
| active | [bool](#bool) |  |  |
| condition | [google.type.Expr](#google-type-Expr) |  |  |






<a name="bytebase-api-v1alpha-UpdateRiskRequest"></a>

### UpdateRiskRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| risk | [Risk](#bytebase-api-v1alpha-Risk) |  | The risk to update.

The risk&#39;s `name` field is used to identify the risk to update. Format: risks/{risk} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 


<a name="bytebase-api-v1alpha-Risk-Source"></a>

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


 

 


<a name="bytebase-api-v1alpha-RiskService"></a>

### RiskService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListRisks | [ListRisksRequest](#bytebase-api-v1alpha-ListRisksRequest) | [ListRisksResponse](#bytebase-api-v1alpha-ListRisksResponse) |  |
| CreateRisk | [CreateRiskRequest](#bytebase-api-v1alpha-CreateRiskRequest) | [Risk](#bytebase-api-v1alpha-Risk) |  |
| UpdateRisk | [UpdateRiskRequest](#bytebase-api-v1alpha-UpdateRiskRequest) | [Risk](#bytebase-api-v1alpha-Risk) |  |
| DeleteRisk | [DeleteRiskRequest](#bytebase-api-v1alpha-DeleteRiskRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="api_v1alpha_role_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/role_service.proto



<a name="bytebase-api-v1alpha-CreateRoleRequest"></a>

### CreateRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [Role](#bytebase-api-v1alpha-Role) |  |  |
| role_id | [string](#string) |  | The ID to use for the role, which will become the final component of the role&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][A-Z][0-9]/. |






<a name="bytebase-api-v1alpha-DeleteRoleRequest"></a>

### DeleteRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: roles/{role} |






<a name="bytebase-api-v1alpha-GetRoleRequest"></a>

### GetRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the role to retrieve. Format: roles/{role} |






<a name="bytebase-api-v1alpha-ListRolesRequest"></a>

### ListRolesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Not used. The maximum number of roles to return. The service may return fewer than this value. If unspecified, at most 10 reviews will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListRoles` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListRoles` must match the call that provided the page token. |






<a name="bytebase-api-v1alpha-ListRolesResponse"></a>

### ListRolesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| roles | [Role](#bytebase-api-v1alpha-Role) | repeated |  |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-Role"></a>

### Role



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: roles/{role} |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| permissions | [string](#string) | repeated |  |
| type | [Role.Type](#bytebase-api-v1alpha-Role-Type) |  |  |






<a name="bytebase-api-v1alpha-UpdateRoleRequest"></a>

### UpdateRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [Role](#bytebase-api-v1alpha-Role) |  |  |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  |  |
| allow_missing | [bool](#bool) |  | If set to true, and the role is not found, a new role will be created. |





 


<a name="bytebase-api-v1alpha-Role-Type"></a>

### Role.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| BUILT_IN | 1 |  |
| CUSTOM | 2 |  |


 

 


<a name="bytebase-api-v1alpha-RoleService"></a>

### RoleService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListRoles | [ListRolesRequest](#bytebase-api-v1alpha-ListRolesRequest) | [ListRolesResponse](#bytebase-api-v1alpha-ListRolesResponse) |  |
| GetRole | [GetRoleRequest](#bytebase-api-v1alpha-GetRoleRequest) | [Role](#bytebase-api-v1alpha-Role) |  |
| CreateRole | [CreateRoleRequest](#bytebase-api-v1alpha-CreateRoleRequest) | [Role](#bytebase-api-v1alpha-Role) |  |
| UpdateRole | [UpdateRoleRequest](#bytebase-api-v1alpha-UpdateRoleRequest) | [Role](#bytebase-api-v1alpha-Role) |  |
| DeleteRole | [DeleteRoleRequest](#bytebase-api-v1alpha-DeleteRoleRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) |  |

 



<a name="api_v1alpha_rollout_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/rollout_service.proto



<a name="bytebase-api-v1alpha-BatchCancelTaskRunsRequest"></a>

### BatchCancelTaskRunsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The name of the parent of the taskRuns. Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} Use `projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/-` to cancel task runs under the same stage. |
| task_runs | [string](#string) | repeated | The taskRuns to cancel. Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |
| reason | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-BatchCancelTaskRunsResponse"></a>

### BatchCancelTaskRunsResponse







<a name="bytebase-api-v1alpha-BatchRunTasksRequest"></a>

### BatchRunTasksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The name of the parent of the tasks. Format: projects/{project}/rollouts/{rollout}/stages/{stage} |
| tasks | [string](#string) | repeated | The tasks to run. Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} |
| reason | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-BatchRunTasksResponse"></a>

### BatchRunTasksResponse







<a name="bytebase-api-v1alpha-BatchSkipTasksRequest"></a>

### BatchSkipTasksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The name of the parent of the tasks. Format: projects/{project}/rollouts/{rollout}/stages/{stage} |
| tasks | [string](#string) | repeated | The tasks to skip. Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} |
| reason | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-BatchSkipTasksResponse"></a>

### BatchSkipTasksResponse







<a name="bytebase-api-v1alpha-CreateRolloutRequest"></a>

### CreateRolloutRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent project where this rollout will be created. Format: projects/{project} |
| rollout | [Rollout](#bytebase-api-v1alpha-Rollout) |  | The rollout to create. |
| target | [string](#string) | optional | Create the rollout and the stages up to the target stage. Format: environments/{environment} If unspecified, all stages are created. If set to &#34;&#34;, no stages are created. |
| validate_only | [bool](#bool) |  | If set, validate the request and preview the rollout, but do not actually create it. |






<a name="bytebase-api-v1alpha-GetRolloutRequest"></a>

### GetRolloutRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the rollout to retrieve. Format: projects/{project}/rollouts/{rollout} |






<a name="bytebase-api-v1alpha-GetTaskRunLogRequest"></a>

### GetTaskRunLogRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} TODO(d): check the resource_reference. |






<a name="bytebase-api-v1alpha-GetTaskRunRequest"></a>

### GetTaskRunRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |






<a name="bytebase-api-v1alpha-GetTaskRunSessionRequest"></a>

### GetTaskRunSessionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |






<a name="bytebase-api-v1alpha-ListRolloutsRequest"></a>

### ListRolloutsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of rollouts. Format: projects/{project} Use &#34;projects/-&#34; to list all rollouts from all projects. |
| page_size | [int32](#int32) |  | The maximum number of rollouts to return. The service may return fewer than this value. If unspecified, at most 10 rollouts will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListRollouts` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListRollouts` must match the call that provided the page token. |






<a name="bytebase-api-v1alpha-ListRolloutsResponse"></a>

### ListRolloutsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rollouts | [Rollout](#bytebase-api-v1alpha-Rollout) | repeated | The rollouts from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-ListTaskRunsRequest"></a>

### ListTaskRunsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of plans. Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} Use &#34;projects/{project}/rollouts/{rollout}/stages/-/tasks/-&#34; to list all taskRuns from a rollout. |
| page_size | [int32](#int32) |  | Not used. The maximum number of taskRuns to return. The service may return fewer than this value. If unspecified, at most 10 taskRuns will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListTaskRuns` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListTaskRuns` must match the call that provided the page token. |






<a name="bytebase-api-v1alpha-ListTaskRunsResponse"></a>

### ListTaskRunsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| task_runs | [TaskRun](#bytebase-api-v1alpha-TaskRun) | repeated | The taskRuns from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-PreviewRolloutRequest"></a>

### PreviewRolloutRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The name of the project. Format: projects/{project} |
| plan | [Plan](#bytebase-api-v1alpha-Plan) |  | The plan used to preview rollout. |






<a name="bytebase-api-v1alpha-PreviewTaskRunRollbackRequest"></a>

### PreviewTaskRunRollbackRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |






<a name="bytebase-api-v1alpha-PreviewTaskRunRollbackResponse"></a>

### PreviewTaskRunRollbackResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| statement | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-Rollout"></a>

### Rollout



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The resource name of the rollout. Format: projects/{project}/rollouts/{rollout} |
| plan | [string](#string) |  | The plan that this rollout is based on. Format: projects/{project}/plans/{plan} |
| title | [string](#string) |  |  |
| stages | [Stage](#bytebase-api-v1alpha-Stage) | repeated | stages and thus tasks of the rollout. |
| creator | [string](#string) |  | Format: users/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| issue | [string](#string) |  | The issue associated with the rollout. Could be empty. Format: projects/{project}/issues/{issue} |






<a name="bytebase-api-v1alpha-Stage"></a>

### Stage



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage} |
| id | [string](#string) |  | The id comes from the deployment config. Format: UUID Empty for legacy stages. |
| environment | [string](#string) |  |  |
| tasks | [Task](#bytebase-api-v1alpha-Task) | repeated |  |






<a name="bytebase-api-v1alpha-Task"></a>

### Task



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} |
| spec_id | [string](#string) |  | A UUID4 string that uniquely identifies the Spec. Could be empty if the rollout of the task does not have an associating plan. |
| status | [Task.Status](#bytebase-api-v1alpha-Task-Status) |  | Status is the status of the task. |
| skipped_reason | [string](#string) |  |  |
| type | [Task.Type](#bytebase-api-v1alpha-Task-Type) |  |  |
| target | [string](#string) |  | Format: instances/{instance} if the task is DatabaseCreate. Format: instances/{instance}/databases/{database} |
| database_create | [Task.DatabaseCreate](#bytebase-api-v1alpha-Task-DatabaseCreate) |  |  |
| database_schema_baseline | [Task.DatabaseSchemaBaseline](#bytebase-api-v1alpha-Task-DatabaseSchemaBaseline) |  |  |
| database_schema_update | [Task.DatabaseSchemaUpdate](#bytebase-api-v1alpha-Task-DatabaseSchemaUpdate) |  |  |
| database_data_update | [Task.DatabaseDataUpdate](#bytebase-api-v1alpha-Task-DatabaseDataUpdate) |  |  |
| database_data_export | [Task.DatabaseDataExport](#bytebase-api-v1alpha-Task-DatabaseDataExport) |  |  |






<a name="bytebase-api-v1alpha-Task-DatabaseCreate"></a>

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






<a name="bytebase-api-v1alpha-Task-DatabaseDataExport"></a>

### Task.DatabaseDataExport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| target | [string](#string) |  | The resource name of the target. Format: instances/{instance-id}/databases/{database-name} |
| sheet | [string](#string) |  | The resource name of the sheet. Format: projects/{project}/sheets/{sheet} |
| format | [ExportFormat](#bytebase-api-v1alpha-ExportFormat) |  | The format of the exported file. |
| password | [string](#string) | optional | The zip password provide by users. Leave it empty if no needs to encrypt the zip file. |






<a name="bytebase-api-v1alpha-Task-DatabaseDataUpdate"></a>

### Task.DatabaseDataUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheet | [string](#string) |  | Format: projects/{project}/sheets/{sheet} |
| schema_version | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-Task-DatabaseSchemaBaseline"></a>

### Task.DatabaseSchemaBaseline



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema_version | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-Task-DatabaseSchemaUpdate"></a>

### Task.DatabaseSchemaUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheet | [string](#string) |  | Format: projects/{project}/sheets/{sheet} |
| schema_version | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-TaskRun"></a>

### TaskRun



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |
| creator | [string](#string) |  | Format: user/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| status | [TaskRun.Status](#bytebase-api-v1alpha-TaskRun-Status) |  |  |
| detail | [string](#string) |  | Below are the results of a task run. |
| changelog | [string](#string) |  | The resource name of the changelog. Format: instances/{instance}/databases/{database}/changelogs/{changelog} |
| schema_version | [string](#string) |  |  |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| export_archive_status | [TaskRun.ExportArchiveStatus](#bytebase-api-v1alpha-TaskRun-ExportArchiveStatus) |  |  |
| prior_backup_detail | [TaskRun.PriorBackupDetail](#bytebase-api-v1alpha-TaskRun-PriorBackupDetail) |  | The prior backup detail that will be used to rollback the task run. |
| scheduler_info | [TaskRun.SchedulerInfo](#bytebase-api-v1alpha-TaskRun-SchedulerInfo) |  |  |
| sheet | [string](#string) |  | Format: projects/{project}/sheets/{sheet} |






<a name="bytebase-api-v1alpha-TaskRun-PriorBackupDetail"></a>

### TaskRun.PriorBackupDetail



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| items | [TaskRun.PriorBackupDetail.Item](#bytebase-api-v1alpha-TaskRun-PriorBackupDetail-Item) | repeated |  |






<a name="bytebase-api-v1alpha-TaskRun-PriorBackupDetail-Item"></a>

### TaskRun.PriorBackupDetail.Item



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| source_table | [TaskRun.PriorBackupDetail.Item.Table](#bytebase-api-v1alpha-TaskRun-PriorBackupDetail-Item-Table) |  | The original table information. |
| target_table | [TaskRun.PriorBackupDetail.Item.Table](#bytebase-api-v1alpha-TaskRun-PriorBackupDetail-Item-Table) |  | The target backup table information. |
| start_position | [Position](#bytebase-api-v1alpha-Position) |  |  |
| end_position | [Position](#bytebase-api-v1alpha-Position) |  |  |






<a name="bytebase-api-v1alpha-TaskRun-PriorBackupDetail-Item-Table"></a>

### TaskRun.PriorBackupDetail.Item.Table



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [string](#string) |  | The database information. Format: instances/{instance}/databases/{database} |
| schema | [string](#string) |  |  |
| table | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-TaskRun-SchedulerInfo"></a>

### TaskRun.SchedulerInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| report_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| waiting_cause | [TaskRun.SchedulerInfo.WaitingCause](#bytebase-api-v1alpha-TaskRun-SchedulerInfo-WaitingCause) |  |  |






<a name="bytebase-api-v1alpha-TaskRun-SchedulerInfo-WaitingCause"></a>

### TaskRun.SchedulerInfo.WaitingCause



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| connection_limit | [bool](#bool) |  |  |
| task | [TaskRun.SchedulerInfo.WaitingCause.Task](#bytebase-api-v1alpha-TaskRun-SchedulerInfo-WaitingCause-Task) |  |  |






<a name="bytebase-api-v1alpha-TaskRun-SchedulerInfo-WaitingCause-Task"></a>

### TaskRun.SchedulerInfo.WaitingCause.Task



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| task | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} |
| issue | [string](#string) |  | Format: projects/{project}/issues/{issue} |






<a name="bytebase-api-v1alpha-TaskRunLog"></a>

### TaskRunLog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun}/log |
| entries | [TaskRunLogEntry](#bytebase-api-v1alpha-TaskRunLogEntry) | repeated |  |






<a name="bytebase-api-v1alpha-TaskRunLogEntry"></a>

### TaskRunLogEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [TaskRunLogEntry.Type](#bytebase-api-v1alpha-TaskRunLogEntry-Type) |  |  |
| log_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| deploy_id | [string](#string) |  |  |
| schema_dump | [TaskRunLogEntry.SchemaDump](#bytebase-api-v1alpha-TaskRunLogEntry-SchemaDump) |  |  |
| command_execute | [TaskRunLogEntry.CommandExecute](#bytebase-api-v1alpha-TaskRunLogEntry-CommandExecute) |  |  |
| database_sync | [TaskRunLogEntry.DatabaseSync](#bytebase-api-v1alpha-TaskRunLogEntry-DatabaseSync) |  |  |
| task_run_status_update | [TaskRunLogEntry.TaskRunStatusUpdate](#bytebase-api-v1alpha-TaskRunLogEntry-TaskRunStatusUpdate) |  |  |
| transaction_control | [TaskRunLogEntry.TransactionControl](#bytebase-api-v1alpha-TaskRunLogEntry-TransactionControl) |  |  |
| prior_backup | [TaskRunLogEntry.PriorBackup](#bytebase-api-v1alpha-TaskRunLogEntry-PriorBackup) |  |  |
| retry_info | [TaskRunLogEntry.RetryInfo](#bytebase-api-v1alpha-TaskRunLogEntry-RetryInfo) |  |  |






<a name="bytebase-api-v1alpha-TaskRunLogEntry-CommandExecute"></a>

### TaskRunLogEntry.CommandExecute



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| log_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| command_indexes | [int32](#int32) | repeated | The indexes of the executed commands. |
| response | [TaskRunLogEntry.CommandExecute.CommandResponse](#bytebase-api-v1alpha-TaskRunLogEntry-CommandExecute-CommandResponse) |  |  |






<a name="bytebase-api-v1alpha-TaskRunLogEntry-CommandExecute-CommandResponse"></a>

### TaskRunLogEntry.CommandExecute.CommandResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| log_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| error | [string](#string) |  |  |
| affected_rows | [int32](#int32) |  |  |
| all_affected_rows | [int32](#int32) | repeated | `all_affected_rows` is the affected rows of each command. `all_affected_rows` may be unavailable if the database driver doesn&#39;t support it. Caller should fallback to `affected_rows` in that case. |






<a name="bytebase-api-v1alpha-TaskRunLogEntry-DatabaseSync"></a>

### TaskRunLogEntry.DatabaseSync



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| end_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| error | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-TaskRunLogEntry-PriorBackup"></a>

### TaskRunLogEntry.PriorBackup



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| end_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| prior_backup_detail | [TaskRun.PriorBackupDetail](#bytebase-api-v1alpha-TaskRun-PriorBackupDetail) |  |  |
| error | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-TaskRunLogEntry-RetryInfo"></a>

### TaskRunLogEntry.RetryInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| error | [string](#string) |  |  |
| retry_count | [int32](#int32) |  |  |
| maximum_retries | [int32](#int32) |  |  |






<a name="bytebase-api-v1alpha-TaskRunLogEntry-SchemaDump"></a>

### TaskRunLogEntry.SchemaDump



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| end_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| error | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-TaskRunLogEntry-TaskRunStatusUpdate"></a>

### TaskRunLogEntry.TaskRunStatusUpdate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [TaskRunLogEntry.TaskRunStatusUpdate.Status](#bytebase-api-v1alpha-TaskRunLogEntry-TaskRunStatusUpdate-Status) |  |  |






<a name="bytebase-api-v1alpha-TaskRunLogEntry-TransactionControl"></a>

### TaskRunLogEntry.TransactionControl



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [TaskRunLogEntry.TransactionControl.Type](#bytebase-api-v1alpha-TaskRunLogEntry-TransactionControl-Type) |  |  |
| error | [string](#string) |  |  |






<a name="bytebase-api-v1alpha-TaskRunSession"></a>

### TaskRunSession



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun}/session |
| postgres | [TaskRunSession.Postgres](#bytebase-api-v1alpha-TaskRunSession-Postgres) |  |  |






<a name="bytebase-api-v1alpha-TaskRunSession-Postgres"></a>

### TaskRunSession.Postgres



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| session | [TaskRunSession.Postgres.Session](#bytebase-api-v1alpha-TaskRunSession-Postgres-Session) |  | `session` is the session of the task run executing commands. |
| blocking_sessions | [TaskRunSession.Postgres.Session](#bytebase-api-v1alpha-TaskRunSession-Postgres-Session) | repeated | `blocking_sessions` block `session`. |
| blocked_sessions | [TaskRunSession.Postgres.Session](#bytebase-api-v1alpha-TaskRunSession-Postgres-Session) | repeated | `blocked_sessions` are blocked by `session`. |






<a name="bytebase-api-v1alpha-TaskRunSession-Postgres-Session"></a>

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





 


<a name="bytebase-api-v1alpha-Task-Status"></a>

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



<a name="bytebase-api-v1alpha-Task-Type"></a>

### Task.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| GENERAL | 1 |  |
| DATABASE_CREATE | 2 | use payload DatabaseCreate |
| DATABASE_SCHEMA_BASELINE | 3 | use payload DatabaseSchemaBaseline |
| DATABASE_SCHEMA_UPDATE | 4 | use payload DatabaseSchemaUpdate |
| DATABASE_SCHEMA_UPDATE_SDL | 5 | use payload DatabaseSchemaUpdate |
| DATABASE_SCHEMA_UPDATE_GHOST | 9 | use payload DatabaseSchemaUpdate |
| DATABASE_DATA_UPDATE | 8 | use payload DatabaseDataUpdate |
| DATABASE_DATA_EXPORT | 12 | use payload DatabaseDataExport |



<a name="bytebase-api-v1alpha-TaskRun-ExportArchiveStatus"></a>

### TaskRun.ExportArchiveStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| EXPORT_ARCHIVE_STATUS_UNSPECIFIED | 0 |  |
| READY | 1 |  |
| EXPORTED | 2 |  |



<a name="bytebase-api-v1alpha-TaskRun-Status"></a>

### TaskRun.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| RUNNING | 2 |  |
| DONE | 3 |  |
| FAILED | 4 |  |
| CANCELED | 5 |  |



<a name="bytebase-api-v1alpha-TaskRunLogEntry-TaskRunStatusUpdate-Status"></a>

### TaskRunLogEntry.TaskRunStatusUpdate.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| RUNNING_WAITING | 1 | the task run is ready to be executed by the scheduler |
| RUNNING_RUNNING | 2 | the task run is being executed by the scheduler |



<a name="bytebase-api-v1alpha-TaskRunLogEntry-TransactionControl-Type"></a>

### TaskRunLogEntry.TransactionControl.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| BEGIN | 1 |  |
| COMMIT | 2 |  |
| ROLLBACK | 3 |  |



<a name="bytebase-api-v1alpha-TaskRunLogEntry-Type"></a>

### TaskRunLogEntry.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| SCHEMA_DUMP | 1 |  |
| COMMAND_EXECUTE | 2 |  |
| DATABASE_SYNC | 3 |  |
| TASK_RUN_STATUS_UPDATE | 4 |  |
| TRANSACTION_CONTROL | 5 |  |
| PRIOR_BACKUP | 6 |  |
| RETRY_INFO | 7 |  |


 

 


<a name="bytebase-api-v1alpha-RolloutService"></a>

### RolloutService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetRollout | [GetRolloutRequest](#bytebase-api-v1alpha-GetRolloutRequest) | [Rollout](#bytebase-api-v1alpha-Rollout) |  |
| ListRollouts | [ListRolloutsRequest](#bytebase-api-v1alpha-ListRolloutsRequest) | [ListRolloutsResponse](#bytebase-api-v1alpha-ListRolloutsResponse) |  |
| CreateRollout | [CreateRolloutRequest](#bytebase-api-v1alpha-CreateRolloutRequest) | [Rollout](#bytebase-api-v1alpha-Rollout) | CreateRollout can be called multiple times with the same rollout.plan but different stage_id to promote rollout stages. |
| PreviewRollout | [PreviewRolloutRequest](#bytebase-api-v1alpha-PreviewRolloutRequest) | [Rollout](#bytebase-api-v1alpha-Rollout) |  |
| ListTaskRuns | [ListTaskRunsRequest](#bytebase-api-v1alpha-ListTaskRunsRequest) | [ListTaskRunsResponse](#bytebase-api-v1alpha-ListTaskRunsResponse) |  |
| GetTaskRun | [GetTaskRunRequest](#bytebase-api-v1alpha-GetTaskRunRequest) | [TaskRun](#bytebase-api-v1alpha-TaskRun) |  |
| GetTaskRunLog | [GetTaskRunLogRequest](#bytebase-api-v1alpha-GetTaskRunLogRequest) | [TaskRunLog](#bytebase-api-v1alpha-TaskRunLog) |  |
| GetTaskRunSession | [GetTaskRunSessionRequest](#bytebase-api-v1alpha-GetTaskRunSessionRequest) | [TaskRunSession](#bytebase-api-v1alpha-TaskRunSession) |  |
| BatchRunTasks | [BatchRunTasksRequest](#bytebase-api-v1alpha-BatchRunTasksRequest) | [BatchRunTasksResponse](#bytebase-api-v1alpha-BatchRunTasksResponse) | BatchRunTasks creates task runs for the specified tasks. DataExport issue only allows the creator to run the task. Users with &#34;bb.taskRuns.create&#34; permission can run the task, e.g. Workspace Admin and DBA. Follow role-based rollout policy for the environment. |
| BatchSkipTasks | [BatchSkipTasksRequest](#bytebase-api-v1alpha-BatchSkipTasksRequest) | [BatchSkipTasksResponse](#bytebase-api-v1alpha-BatchSkipTasksResponse) | BatchSkipTasks skips the specified tasks. The access is the same as BatchRunTasks(). |
| BatchCancelTaskRuns | [BatchCancelTaskRunsRequest](#bytebase-api-v1alpha-BatchCancelTaskRunsRequest) | [BatchCancelTaskRunsResponse](#bytebase-api-v1alpha-BatchCancelTaskRunsResponse) | BatchCancelTaskRuns cancels the specified task runs in batch. The access is the same as BatchRunTasks(). |
| PreviewTaskRunRollback | [PreviewTaskRunRollbackRequest](#bytebase-api-v1alpha-PreviewTaskRunRollbackRequest) | [PreviewTaskRunRollbackResponse](#bytebase-api-v1alpha-PreviewTaskRunRollbackResponse) |  |

 



<a name="api_v1alpha_sheet_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/sheet_service.proto



<a name="bytebase-api-v1alpha-BatchCreateSheetRequest"></a>

### BatchCreateSheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where all sheets will be created. Format: projects/{project} |
| requests | [CreateSheetRequest](#bytebase-api-v1alpha-CreateSheetRequest) | repeated |  |






<a name="bytebase-api-v1alpha-BatchCreateSheetResponse"></a>

### BatchCreateSheetResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheets | [Sheet](#bytebase-api-v1alpha-Sheet) | repeated |  |






<a name="bytebase-api-v1alpha-CreateSheetRequest"></a>

### CreateSheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this sheet will be created. Format: projects/{project} |
| sheet | [Sheet](#bytebase-api-v1alpha-Sheet) |  | The sheet to create. |






<a name="bytebase-api-v1alpha-GetSheetRequest"></a>

### GetSheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the sheet to retrieve. Format: projects/{project}/sheets/{sheet} |
| raw | [bool](#bool) |  | By default, the content of the sheet is cut off, set the `raw` to true to retrieve the full content. |






<a name="bytebase-api-v1alpha-Sheet"></a>

### Sheet



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the sheet resource, generated by the server. Canonical parent is project. Format: projects/{project}/sheets/{sheet} |
| title | [string](#string) |  | The title of the sheet. |
| creator | [string](#string) |  | The creator of the Sheet. Format: users/{email} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The create time of the sheet. |
| content | [bytes](#bytes) |  | The content of the sheet. By default, it will be cut off, if it doesn&#39;t match the `content_size`, you can set the `raw` to true in GetSheet request to retrieve the full content. |
| content_size | [int64](#int64) |  | content_size is the full size of the content, may not match the size of the `content` field. |
| payload | [SheetPayload](#bytebase-api-v1alpha-SheetPayload) |  |  |
| engine | [Engine](#bytebase-api-v1alpha-Engine) |  | The SQL dialect. |






<a name="bytebase-api-v1alpha-SheetCommand"></a>

### SheetCommand



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start | [int32](#int32) |  |  |
| end | [int32](#int32) |  |  |






<a name="bytebase-api-v1alpha-SheetPayload"></a>

### SheetPayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [SheetPayload.Type](#bytebase-api-v1alpha-SheetPayload-Type) |  |  |
| commands | [SheetCommand](#bytebase-api-v1alpha-SheetCommand) | repeated | The start and end position of each command in the sheet statement. |






<a name="bytebase-api-v1alpha-UpdateSheetRequest"></a>

### UpdateSheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheet | [Sheet](#bytebase-api-v1alpha-Sheet) |  | The sheet to update.

The sheet&#39;s `name` field is used to identify the sheet to update. Format: projects/{project}/sheets/{sheet} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to be updated. Fields are specified relative to the sheet. (e.g. `title`, `statement`; *not* `sheet.title` or `sheet.statement`) Only support update the following fields for now: - `title` - `statement` |





 


<a name="bytebase-api-v1alpha-SheetPayload-Type"></a>

### SheetPayload.Type
Type of the SheetPayload.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| SCHEMA_DESIGN | 1 |  |


 

 


<a name="bytebase-api-v1alpha-SheetService"></a>

### SheetService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateSheet | [CreateSheetRequest](#bytebase-api-v1alpha-CreateSheetRequest) | [Sheet](#bytebase-api-v1alpha-Sheet) |  |
| BatchCreateSheet | [BatchCreateSheetRequest](#bytebase-api-v1alpha-BatchCreateSheetRequest) | [BatchCreateSheetResponse](#bytebase-api-v1alpha-BatchCreateSheetResponse) |  |
| GetSheet | [GetSheetRequest](#bytebase-api-v1alpha-GetSheetRequest) | [Sheet](#bytebase-api-v1alpha-Sheet) |  |
| UpdateSheet | [UpdateSheetRequest](#bytebase-api-v1alpha-UpdateSheetRequest) | [Sheet](#bytebase-api-v1alpha-Sheet) |  |

 



<a name="api_v1alpha_worksheet_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/worksheet_service.proto



<a name="bytebase-api-v1alpha-CreateWorksheetRequest"></a>

### CreateWorksheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| worksheet | [Worksheet](#bytebase-api-v1alpha-Worksheet) |  | The worksheet to create. |






<a name="bytebase-api-v1alpha-DeleteWorksheetRequest"></a>

### DeleteWorksheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the worksheet to delete. Format: worksheets/{worksheet} |






<a name="bytebase-api-v1alpha-GetWorksheetRequest"></a>

### GetWorksheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the worksheet to retrieve. Format: worksheets/{worksheet} |






<a name="bytebase-api-v1alpha-SearchWorksheetsRequest"></a>

### SearchWorksheetsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| filter | [string](#string) |  | To filter the search result. Supported filter: - creator - starred - visibility

For example: creator == &#34;users/{email}&#34; creator != &#34;users/{email}&#34; starred == true starred == false visibility in [&#34;VISIBILITY_PRIVATE&#34;, &#34;VISIBILITY_PROJECT_READ&#34;, &#34;VISIBILITY_PROJECT_WRITE&#34;] visibility == &#34;VISIBILITY_PRIVATE&#34; |
| page_size | [int32](#int32) |  | Not used. The maximum number of worksheets to return. The service may return fewer than this value. If unspecified, at most 10 worksheets will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `SearchWorksheets` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `SearchWorksheets` must match the call that provided the page token. |






<a name="bytebase-api-v1alpha-SearchWorksheetsResponse"></a>

### SearchWorksheetsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| worksheets | [Worksheet](#bytebase-api-v1alpha-Worksheet) | repeated | The worksheets that matched the search criteria. |
| next_page_token | [string](#string) |  | Not used. A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-api-v1alpha-UpdateWorksheetOrganizerRequest"></a>

### UpdateWorksheetOrganizerRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organizer | [WorksheetOrganizer](#bytebase-api-v1alpha-WorksheetOrganizer) |  | The organizer to update.

The organizer&#39;s `worksheet` field is used to identify the worksheet. Format: worksheets/{worksheet} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to be updated. Fields are specified relative to the worksheet organizer. Only support update the following fields for now: - `starred` |






<a name="bytebase-api-v1alpha-UpdateWorksheetRequest"></a>

### UpdateWorksheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| worksheet | [Worksheet](#bytebase-api-v1alpha-Worksheet) |  | The worksheet to update.

The worksheet&#39;s `name` field is used to identify the worksheet to update. Format: worksheets/{worksheet} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to be updated. Fields are specified relative to the worksheet. (e.g. `title`, `statement`; *not* `worksheet.title` or `worksheet.statement`) Only support update the following fields for now: - `title` - `statement` - `starred` - `visibility` |






<a name="bytebase-api-v1alpha-Worksheet"></a>

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
| visibility | [Worksheet.Visibility](#bytebase-api-v1alpha-Worksheet-Visibility) |  |  |
| starred | [bool](#bool) |  | starred indicates whether the worksheet is starred by the current authenticated user. |






<a name="bytebase-api-v1alpha-WorksheetOrganizer"></a>

### WorksheetOrganizer



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| worksheet | [string](#string) |  | The name of the worksheet. Format: worksheets/{worksheet} |
| starred | [bool](#bool) |  | starred means if the worksheet is starred. |





 


<a name="bytebase-api-v1alpha-Worksheet-Visibility"></a>

### Worksheet.Visibility


| Name | Number | Description |
| ---- | ------ | ----------- |
| VISIBILITY_UNSPECIFIED | 0 |  |
| VISIBILITY_PROJECT_READ | 1 | Read access in project scope, worksheet OWNER/DBA and project OWNER can read/write, other project members can read. |
| VISIBILITY_PROJECT_WRITE | 2 | Write access in project scope, worksheet OWNER/DBA and all members in the project can write the worksheet. |
| VISIBILITY_PRIVATE | 3 | Private, only worksheet OWNER can read/write. |


 

 


<a name="bytebase-api-v1alpha-WorksheetService"></a>

### WorksheetService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateWorksheet | [CreateWorksheetRequest](#bytebase-api-v1alpha-CreateWorksheetRequest) | [Worksheet](#bytebase-api-v1alpha-Worksheet) | Create a personal worksheet used in SQL Editor. |
| GetWorksheet | [GetWorksheetRequest](#bytebase-api-v1alpha-GetWorksheetRequest) | [Worksheet](#bytebase-api-v1alpha-Worksheet) | Get a worksheet by name. The users can access this method if, - they are the creator of the worksheet; - they have bb.worksheets.get permission on the workspace; - the sheet is shared with them with PROJECT_READ and PROJECT_WRITE visibility, and they have bb.projects.get permission on the project. |
| SearchWorksheets | [SearchWorksheetsRequest](#bytebase-api-v1alpha-SearchWorksheetsRequest) | [SearchWorksheetsResponse](#bytebase-api-v1alpha-SearchWorksheetsResponse) | Search for worksheets. This is used for finding my worksheets or worksheets shared by other people. The sheet accessibility is the same as GetWorksheet(). |
| UpdateWorksheet | [UpdateWorksheetRequest](#bytebase-api-v1alpha-UpdateWorksheetRequest) | [Worksheet](#bytebase-api-v1alpha-Worksheet) | Update a worksheet. The users can access this method if, - they are the creator of the worksheet; - they have bb.worksheets.manage permission on the workspace; - the sheet is shared with them with PROJECT_WRITE visibility, and they have bb.projects.get permission on the project. |
| UpdateWorksheetOrganizer | [UpdateWorksheetOrganizerRequest](#bytebase-api-v1alpha-UpdateWorksheetOrganizerRequest) | [WorksheetOrganizer](#bytebase-api-v1alpha-WorksheetOrganizer) | Update the organizer of a worksheet. The access is the same as UpdateWorksheet method. |
| DeleteWorksheet | [DeleteWorksheetRequest](#bytebase-api-v1alpha-DeleteWorksheetRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Delete a worksheet. The access is the same as UpdateWorksheet method. |

 



<a name="api_v1alpha_workspace_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1alpha/workspace_service.proto


 

 

 


<a name="bytebase-api-v1alpha-WorkspaceService"></a>

### WorkspaceService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetIamPolicy | [GetIamPolicyRequest](#bytebase-api-v1alpha-GetIamPolicyRequest) | [IamPolicy](#bytebase-api-v1alpha-IamPolicy) |  |
| SetIamPolicy | [SetIamPolicyRequest](#bytebase-api-v1alpha-SetIamPolicyRequest) | [IamPolicy](#bytebase-api-v1alpha-IamPolicy) |  |

 



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

