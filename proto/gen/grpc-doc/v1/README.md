# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [v1/annotation.proto](#v1_annotation-proto)
    - [AuthMethod](#bytebase-v1-AuthMethod)
  
    - [File-level Extensions](#v1_annotation-proto-extensions)
    - [File-level Extensions](#v1_annotation-proto-extensions)
    - [File-level Extensions](#v1_annotation-proto-extensions)
    - [File-level Extensions](#v1_annotation-proto-extensions)
  
- [v1/common.proto](#v1_common-proto)
    - [PermissionDeniedDetail](#bytebase-v1-PermissionDeniedDetail)
    - [Position](#bytebase-v1-Position)
    - [Range](#bytebase-v1-Range)
  
    - [Engine](#bytebase-v1-Engine)
    - [ExportFormat](#bytebase-v1-ExportFormat)
    - [RiskLevel](#bytebase-v1-RiskLevel)
    - [State](#bytebase-v1-State)
    - [VCSType](#bytebase-v1-VCSType)
    - [WebhookType](#bytebase-v1-WebhookType)
  
- [v1/issue_service.proto](#v1_issue_service-proto)
    - [ApprovalFlow](#bytebase-v1-ApprovalFlow)
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
    - [IssueComment](#bytebase-v1-IssueComment)
    - [IssueComment.Approval](#bytebase-v1-IssueComment-Approval)
    - [IssueComment.IssueUpdate](#bytebase-v1-IssueComment-IssueUpdate)
    - [IssueComment.PlanSpecUpdate](#bytebase-v1-IssueComment-PlanSpecUpdate)
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
  
    - [Issue.ApprovalStatus](#bytebase-v1-Issue-ApprovalStatus)
    - [Issue.Approver.Status](#bytebase-v1-Issue-Approver-Status)
    - [Issue.Type](#bytebase-v1-Issue-Type)
    - [IssueComment.Approval.Status](#bytebase-v1-IssueComment-Approval-Status)
    - [IssueStatus](#bytebase-v1-IssueStatus)
  
    - [IssueService](#bytebase-v1-IssueService)
  
- [v1/setting_service.proto](#v1_setting_service-proto)
    - [AISetting](#bytebase-v1-AISetting)
    - [Algorithm](#bytebase-v1-Algorithm)
    - [Algorithm.FullMask](#bytebase-v1-Algorithm-FullMask)
    - [Algorithm.InnerOuterMask](#bytebase-v1-Algorithm-InnerOuterMask)
    - [Algorithm.MD5Mask](#bytebase-v1-Algorithm-MD5Mask)
    - [Algorithm.RangeMask](#bytebase-v1-Algorithm-RangeMask)
    - [Algorithm.RangeMask.Slice](#bytebase-v1-Algorithm-RangeMask-Slice)
    - [Announcement](#bytebase-v1-Announcement)
    - [AppIMSetting](#bytebase-v1-AppIMSetting)
    - [AppIMSetting.DingTalk](#bytebase-v1-AppIMSetting-DingTalk)
    - [AppIMSetting.Feishu](#bytebase-v1-AppIMSetting-Feishu)
    - [AppIMSetting.IMSetting](#bytebase-v1-AppIMSetting-IMSetting)
    - [AppIMSetting.Lark](#bytebase-v1-AppIMSetting-Lark)
    - [AppIMSetting.Slack](#bytebase-v1-AppIMSetting-Slack)
    - [AppIMSetting.Teams](#bytebase-v1-AppIMSetting-Teams)
    - [AppIMSetting.Wecom](#bytebase-v1-AppIMSetting-Wecom)
    - [DataClassificationSetting](#bytebase-v1-DataClassificationSetting)
    - [DataClassificationSetting.DataClassificationConfig](#bytebase-v1-DataClassificationSetting-DataClassificationConfig)
    - [DataClassificationSetting.DataClassificationConfig.ClassificationEntry](#bytebase-v1-DataClassificationSetting-DataClassificationConfig-ClassificationEntry)
    - [DataClassificationSetting.DataClassificationConfig.DataClassification](#bytebase-v1-DataClassificationSetting-DataClassificationConfig-DataClassification)
    - [DataClassificationSetting.DataClassificationConfig.Level](#bytebase-v1-DataClassificationSetting-DataClassificationConfig-Level)
    - [EnvironmentSetting](#bytebase-v1-EnvironmentSetting)
    - [EnvironmentSetting.Environment](#bytebase-v1-EnvironmentSetting-Environment)
    - [EnvironmentSetting.Environment.TagsEntry](#bytebase-v1-EnvironmentSetting-Environment-TagsEntry)
    - [GetSettingRequest](#bytebase-v1-GetSettingRequest)
    - [GetSettingResponse](#bytebase-v1-GetSettingResponse)
    - [ListSettingsRequest](#bytebase-v1-ListSettingsRequest)
    - [ListSettingsResponse](#bytebase-v1-ListSettingsResponse)
    - [SemanticTypeSetting](#bytebase-v1-SemanticTypeSetting)
    - [SemanticTypeSetting.SemanticType](#bytebase-v1-SemanticTypeSetting-SemanticType)
    - [Setting](#bytebase-v1-Setting)
    - [SettingValue](#bytebase-v1-SettingValue)
    - [UpdateSettingRequest](#bytebase-v1-UpdateSettingRequest)
    - [WorkspaceApprovalSetting](#bytebase-v1-WorkspaceApprovalSetting)
    - [WorkspaceApprovalSetting.Rule](#bytebase-v1-WorkspaceApprovalSetting-Rule)
    - [WorkspaceProfileSetting](#bytebase-v1-WorkspaceProfileSetting)
    - [WorkspaceProfileSetting.PasswordRestriction](#bytebase-v1-WorkspaceProfileSetting-PasswordRestriction)
  
    - [AISetting.Provider](#bytebase-v1-AISetting-Provider)
    - [Algorithm.InnerOuterMask.MaskType](#bytebase-v1-Algorithm-InnerOuterMask-MaskType)
    - [Announcement.AlertLevel](#bytebase-v1-Announcement-AlertLevel)
    - [DatabaseChangeMode](#bytebase-v1-DatabaseChangeMode)
    - [Setting.SettingName](#bytebase-v1-Setting-SettingName)
    - [WorkspaceApprovalSetting.Rule.Source](#bytebase-v1-WorkspaceApprovalSetting-Rule-Source)
  
    - [SettingService](#bytebase-v1-SettingService)
  
- [v1/user_service.proto](#v1_user_service-proto)
    - [BatchGetUsersRequest](#bytebase-v1-BatchGetUsersRequest)
    - [BatchGetUsersResponse](#bytebase-v1-BatchGetUsersResponse)
    - [CreateUserRequest](#bytebase-v1-CreateUserRequest)
    - [DeleteUserRequest](#bytebase-v1-DeleteUserRequest)
    - [GetUserRequest](#bytebase-v1-GetUserRequest)
    - [ListUsersRequest](#bytebase-v1-ListUsersRequest)
    - [ListUsersResponse](#bytebase-v1-ListUsersResponse)
    - [UndeleteUserRequest](#bytebase-v1-UndeleteUserRequest)
    - [UpdateEmailRequest](#bytebase-v1-UpdateEmailRequest)
    - [UpdateUserRequest](#bytebase-v1-UpdateUserRequest)
    - [User](#bytebase-v1-User)
    - [User.Profile](#bytebase-v1-User-Profile)
    - [WorkloadIdentityConfig](#bytebase-v1-WorkloadIdentityConfig)
  
    - [UserType](#bytebase-v1-UserType)
    - [WorkloadIdentityConfig.ProviderType](#bytebase-v1-WorkloadIdentityConfig-ProviderType)
  
    - [UserService](#bytebase-v1-UserService)
  
- [v1/actuator_service.proto](#v1_actuator_service-proto)
    - [ActuatorInfo](#bytebase-v1-ActuatorInfo)
    - [ActuatorInfo.StatUser](#bytebase-v1-ActuatorInfo-StatUser)
    - [DeleteCacheRequest](#bytebase-v1-DeleteCacheRequest)
    - [GetActuatorInfoRequest](#bytebase-v1-GetActuatorInfoRequest)
    - [GetResourcePackageRequest](#bytebase-v1-GetResourcePackageRequest)
    - [ResourcePackage](#bytebase-v1-ResourcePackage)
    - [SetupSampleRequest](#bytebase-v1-SetupSampleRequest)
    - [UpdateActuatorInfoRequest](#bytebase-v1-UpdateActuatorInfoRequest)
  
    - [ActuatorService](#bytebase-v1-ActuatorService)
  
- [v1/iam_policy.proto](#v1_iam_policy-proto)
    - [Binding](#bytebase-v1-Binding)
    - [BindingDelta](#bytebase-v1-BindingDelta)
    - [GetIamPolicyRequest](#bytebase-v1-GetIamPolicyRequest)
    - [IamPolicy](#bytebase-v1-IamPolicy)
    - [PolicyDelta](#bytebase-v1-PolicyDelta)
    - [SetIamPolicyRequest](#bytebase-v1-SetIamPolicyRequest)
  
    - [BindingDelta.Action](#bytebase-v1-BindingDelta-Action)
  
- [v1/audit_log_service.proto](#v1_audit_log_service-proto)
    - [AuditData](#bytebase-v1-AuditData)
    - [AuditLog](#bytebase-v1-AuditLog)
    - [ExportAuditLogsRequest](#bytebase-v1-ExportAuditLogsRequest)
    - [ExportAuditLogsResponse](#bytebase-v1-ExportAuditLogsResponse)
    - [RequestMetadata](#bytebase-v1-RequestMetadata)
    - [SearchAuditLogsRequest](#bytebase-v1-SearchAuditLogsRequest)
    - [SearchAuditLogsResponse](#bytebase-v1-SearchAuditLogsResponse)
  
    - [AuditLog.Severity](#bytebase-v1-AuditLog-Severity)
  
    - [AuditLogService](#bytebase-v1-AuditLogService)
  
- [v1/auth_service.proto](#v1_auth_service-proto)
    - [ExchangeTokenRequest](#bytebase-v1-ExchangeTokenRequest)
    - [ExchangeTokenResponse](#bytebase-v1-ExchangeTokenResponse)
    - [IdentityProviderContext](#bytebase-v1-IdentityProviderContext)
    - [LoginRequest](#bytebase-v1-LoginRequest)
    - [LoginResponse](#bytebase-v1-LoginResponse)
    - [LogoutRequest](#bytebase-v1-LogoutRequest)
    - [OAuth2IdentityProviderContext](#bytebase-v1-OAuth2IdentityProviderContext)
    - [OIDCIdentityProviderContext](#bytebase-v1-OIDCIdentityProviderContext)
    - [RefreshRequest](#bytebase-v1-RefreshRequest)
    - [RefreshResponse](#bytebase-v1-RefreshResponse)
  
    - [AuthService](#bytebase-v1-AuthService)
  
- [v1/cel_service.proto](#v1_cel_service-proto)
    - [BatchDeparseRequest](#bytebase-v1-BatchDeparseRequest)
    - [BatchDeparseResponse](#bytebase-v1-BatchDeparseResponse)
    - [BatchParseRequest](#bytebase-v1-BatchParseRequest)
    - [BatchParseResponse](#bytebase-v1-BatchParseResponse)
  
    - [CelService](#bytebase-v1-CelService)
  
- [v1/database_catalog_service.proto](#v1_database_catalog_service-proto)
    - [ColumnCatalog](#bytebase-v1-ColumnCatalog)
    - [ColumnCatalog.LabelsEntry](#bytebase-v1-ColumnCatalog-LabelsEntry)
    - [DatabaseCatalog](#bytebase-v1-DatabaseCatalog)
    - [GetDatabaseCatalogRequest](#bytebase-v1-GetDatabaseCatalogRequest)
    - [ObjectSchema](#bytebase-v1-ObjectSchema)
    - [ObjectSchema.ArrayKind](#bytebase-v1-ObjectSchema-ArrayKind)
    - [ObjectSchema.StructKind](#bytebase-v1-ObjectSchema-StructKind)
    - [ObjectSchema.StructKind.PropertiesEntry](#bytebase-v1-ObjectSchema-StructKind-PropertiesEntry)
    - [SchemaCatalog](#bytebase-v1-SchemaCatalog)
    - [TableCatalog](#bytebase-v1-TableCatalog)
    - [TableCatalog.Columns](#bytebase-v1-TableCatalog-Columns)
    - [UpdateDatabaseCatalogRequest](#bytebase-v1-UpdateDatabaseCatalogRequest)
  
    - [ObjectSchema.Type](#bytebase-v1-ObjectSchema-Type)
  
    - [DatabaseCatalogService](#bytebase-v1-DatabaseCatalogService)
  
- [v1/database_group_service.proto](#v1_database_group_service-proto)
    - [CreateDatabaseGroupRequest](#bytebase-v1-CreateDatabaseGroupRequest)
    - [DatabaseGroup](#bytebase-v1-DatabaseGroup)
    - [DatabaseGroup.Database](#bytebase-v1-DatabaseGroup-Database)
    - [DeleteDatabaseGroupRequest](#bytebase-v1-DeleteDatabaseGroupRequest)
    - [GetDatabaseGroupRequest](#bytebase-v1-GetDatabaseGroupRequest)
    - [ListDatabaseGroupsRequest](#bytebase-v1-ListDatabaseGroupsRequest)
    - [ListDatabaseGroupsResponse](#bytebase-v1-ListDatabaseGroupsResponse)
    - [UpdateDatabaseGroupRequest](#bytebase-v1-UpdateDatabaseGroupRequest)
  
    - [DatabaseGroupView](#bytebase-v1-DatabaseGroupView)
  
    - [DatabaseGroupService](#bytebase-v1-DatabaseGroupService)
  
- [v1/instance_role_service.proto](#v1_instance_role_service-proto)
    - [InstanceRole](#bytebase-v1-InstanceRole)
    - [ListInstanceRolesRequest](#bytebase-v1-ListInstanceRolesRequest)
    - [ListInstanceRolesResponse](#bytebase-v1-ListInstanceRolesResponse)
  
    - [InstanceRoleService](#bytebase-v1-InstanceRoleService)
  
- [v1/instance_service.proto](#v1_instance_service-proto)
    - [AddDataSourceRequest](#bytebase-v1-AddDataSourceRequest)
    - [BatchSyncInstancesRequest](#bytebase-v1-BatchSyncInstancesRequest)
    - [BatchSyncInstancesResponse](#bytebase-v1-BatchSyncInstancesResponse)
    - [BatchUpdateInstancesRequest](#bytebase-v1-BatchUpdateInstancesRequest)
    - [BatchUpdateInstancesResponse](#bytebase-v1-BatchUpdateInstancesResponse)
    - [CreateInstanceRequest](#bytebase-v1-CreateInstanceRequest)
    - [DataSource](#bytebase-v1-DataSource)
    - [DataSource.AWSCredential](#bytebase-v1-DataSource-AWSCredential)
    - [DataSource.Address](#bytebase-v1-DataSource-Address)
    - [DataSource.AzureCredential](#bytebase-v1-DataSource-AzureCredential)
    - [DataSource.ExtraConnectionParametersEntry](#bytebase-v1-DataSource-ExtraConnectionParametersEntry)
    - [DataSource.GCPCredential](#bytebase-v1-DataSource-GCPCredential)
    - [DataSourceExternalSecret](#bytebase-v1-DataSourceExternalSecret)
    - [DataSourceExternalSecret.AppRoleAuthOption](#bytebase-v1-DataSourceExternalSecret-AppRoleAuthOption)
    - [DeleteInstanceRequest](#bytebase-v1-DeleteInstanceRequest)
    - [GetInstanceRequest](#bytebase-v1-GetInstanceRequest)
    - [Instance](#bytebase-v1-Instance)
    - [Instance.LabelsEntry](#bytebase-v1-Instance-LabelsEntry)
    - [InstanceResource](#bytebase-v1-InstanceResource)
    - [KerberosConfig](#bytebase-v1-KerberosConfig)
    - [ListInstanceDatabaseRequest](#bytebase-v1-ListInstanceDatabaseRequest)
    - [ListInstanceDatabaseResponse](#bytebase-v1-ListInstanceDatabaseResponse)
    - [ListInstancesRequest](#bytebase-v1-ListInstancesRequest)
    - [ListInstancesResponse](#bytebase-v1-ListInstancesResponse)
    - [RemoveDataSourceRequest](#bytebase-v1-RemoveDataSourceRequest)
    - [SASLConfig](#bytebase-v1-SASLConfig)
    - [SyncInstanceRequest](#bytebase-v1-SyncInstanceRequest)
    - [SyncInstanceResponse](#bytebase-v1-SyncInstanceResponse)
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
    - [BatchGetDatabasesRequest](#bytebase-v1-BatchGetDatabasesRequest)
    - [BatchGetDatabasesResponse](#bytebase-v1-BatchGetDatabasesResponse)
    - [BatchSyncDatabasesRequest](#bytebase-v1-BatchSyncDatabasesRequest)
    - [BatchSyncDatabasesResponse](#bytebase-v1-BatchSyncDatabasesResponse)
    - [BatchUpdateDatabasesRequest](#bytebase-v1-BatchUpdateDatabasesRequest)
    - [BatchUpdateDatabasesResponse](#bytebase-v1-BatchUpdateDatabasesResponse)
    - [BoundingBox](#bytebase-v1-BoundingBox)
    - [Changelog](#bytebase-v1-Changelog)
    - [CheckConstraintMetadata](#bytebase-v1-CheckConstraintMetadata)
    - [ColumnMetadata](#bytebase-v1-ColumnMetadata)
    - [Database](#bytebase-v1-Database)
    - [Database.LabelsEntry](#bytebase-v1-Database-LabelsEntry)
    - [DatabaseMetadata](#bytebase-v1-DatabaseMetadata)
    - [DatabaseSDLSchema](#bytebase-v1-DatabaseSDLSchema)
    - [DatabaseSchema](#bytebase-v1-DatabaseSchema)
    - [DependencyColumn](#bytebase-v1-DependencyColumn)
    - [DependencyTable](#bytebase-v1-DependencyTable)
    - [DiffSchemaRequest](#bytebase-v1-DiffSchemaRequest)
    - [DiffSchemaResponse](#bytebase-v1-DiffSchemaResponse)
    - [DimensionConstraint](#bytebase-v1-DimensionConstraint)
    - [DimensionalConfig](#bytebase-v1-DimensionalConfig)
    - [EnumTypeMetadata](#bytebase-v1-EnumTypeMetadata)
    - [EventMetadata](#bytebase-v1-EventMetadata)
    - [ExtensionMetadata](#bytebase-v1-ExtensionMetadata)
    - [ExternalTableMetadata](#bytebase-v1-ExternalTableMetadata)
    - [ForeignKeyMetadata](#bytebase-v1-ForeignKeyMetadata)
    - [FunctionMetadata](#bytebase-v1-FunctionMetadata)
    - [GenerationMetadata](#bytebase-v1-GenerationMetadata)
    - [GetChangelogRequest](#bytebase-v1-GetChangelogRequest)
    - [GetDatabaseMetadataRequest](#bytebase-v1-GetDatabaseMetadataRequest)
    - [GetDatabaseRequest](#bytebase-v1-GetDatabaseRequest)
    - [GetDatabaseSDLSchemaRequest](#bytebase-v1-GetDatabaseSDLSchemaRequest)
    - [GetDatabaseSchemaRequest](#bytebase-v1-GetDatabaseSchemaRequest)
    - [GetSchemaStringRequest](#bytebase-v1-GetSchemaStringRequest)
    - [GetSchemaStringResponse](#bytebase-v1-GetSchemaStringResponse)
    - [GridLevel](#bytebase-v1-GridLevel)
    - [IndexMetadata](#bytebase-v1-IndexMetadata)
    - [ListChangelogsRequest](#bytebase-v1-ListChangelogsRequest)
    - [ListChangelogsResponse](#bytebase-v1-ListChangelogsResponse)
    - [ListDatabasesRequest](#bytebase-v1-ListDatabasesRequest)
    - [ListDatabasesResponse](#bytebase-v1-ListDatabasesResponse)
    - [MaterializedViewMetadata](#bytebase-v1-MaterializedViewMetadata)
    - [PackageMetadata](#bytebase-v1-PackageMetadata)
    - [ProcedureMetadata](#bytebase-v1-ProcedureMetadata)
    - [SchemaMetadata](#bytebase-v1-SchemaMetadata)
    - [SequenceMetadata](#bytebase-v1-SequenceMetadata)
    - [SpatialIndexConfig](#bytebase-v1-SpatialIndexConfig)
    - [StorageConfig](#bytebase-v1-StorageConfig)
    - [StreamMetadata](#bytebase-v1-StreamMetadata)
    - [SyncDatabaseRequest](#bytebase-v1-SyncDatabaseRequest)
    - [SyncDatabaseResponse](#bytebase-v1-SyncDatabaseResponse)
    - [TableMetadata](#bytebase-v1-TableMetadata)
    - [TablePartitionMetadata](#bytebase-v1-TablePartitionMetadata)
    - [TaskMetadata](#bytebase-v1-TaskMetadata)
    - [TessellationConfig](#bytebase-v1-TessellationConfig)
    - [TriggerMetadata](#bytebase-v1-TriggerMetadata)
    - [UpdateDatabaseRequest](#bytebase-v1-UpdateDatabaseRequest)
    - [ViewMetadata](#bytebase-v1-ViewMetadata)
  
    - [Changelog.Status](#bytebase-v1-Changelog-Status)
    - [Changelog.Type](#bytebase-v1-Changelog-Type)
    - [ChangelogView](#bytebase-v1-ChangelogView)
    - [ColumnMetadata.IdentityGeneration](#bytebase-v1-ColumnMetadata-IdentityGeneration)
    - [GenerationMetadata.Type](#bytebase-v1-GenerationMetadata-Type)
    - [GetDatabaseSDLSchemaRequest.SDLFormat](#bytebase-v1-GetDatabaseSDLSchemaRequest-SDLFormat)
    - [GetSchemaStringRequest.ObjectType](#bytebase-v1-GetSchemaStringRequest-ObjectType)
    - [StreamMetadata.Mode](#bytebase-v1-StreamMetadata-Mode)
    - [StreamMetadata.Type](#bytebase-v1-StreamMetadata-Type)
    - [TablePartitionMetadata.Type](#bytebase-v1-TablePartitionMetadata-Type)
    - [TaskMetadata.State](#bytebase-v1-TaskMetadata-State)
  
    - [DatabaseService](#bytebase-v1-DatabaseService)
  
- [v1/group_service.proto](#v1_group_service-proto)
    - [BatchGetGroupsRequest](#bytebase-v1-BatchGetGroupsRequest)
    - [BatchGetGroupsResponse](#bytebase-v1-BatchGetGroupsResponse)
    - [CreateGroupRequest](#bytebase-v1-CreateGroupRequest)
    - [DeleteGroupRequest](#bytebase-v1-DeleteGroupRequest)
    - [GetGroupRequest](#bytebase-v1-GetGroupRequest)
    - [Group](#bytebase-v1-Group)
    - [GroupMember](#bytebase-v1-GroupMember)
    - [ListGroupsRequest](#bytebase-v1-ListGroupsRequest)
    - [ListGroupsResponse](#bytebase-v1-ListGroupsResponse)
    - [UpdateGroupRequest](#bytebase-v1-UpdateGroupRequest)
  
    - [GroupMember.Role](#bytebase-v1-GroupMember-Role)
  
    - [GroupService](#bytebase-v1-GroupService)
  
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
    - [OIDCIdentityProviderTestRequestContext](#bytebase-v1-OIDCIdentityProviderTestRequestContext)
    - [TestIdentityProviderRequest](#bytebase-v1-TestIdentityProviderRequest)
    - [TestIdentityProviderResponse](#bytebase-v1-TestIdentityProviderResponse)
    - [TestIdentityProviderResponse.ClaimsEntry](#bytebase-v1-TestIdentityProviderResponse-ClaimsEntry)
    - [TestIdentityProviderResponse.UserInfoEntry](#bytebase-v1-TestIdentityProviderResponse-UserInfoEntry)
    - [UpdateIdentityProviderRequest](#bytebase-v1-UpdateIdentityProviderRequest)
  
    - [IdentityProviderType](#bytebase-v1-IdentityProviderType)
    - [LDAPIdentityProviderConfig.SecurityProtocol](#bytebase-v1-LDAPIdentityProviderConfig-SecurityProtocol)
    - [OAuth2AuthStyle](#bytebase-v1-OAuth2AuthStyle)
  
    - [IdentityProviderService](#bytebase-v1-IdentityProviderService)
  
- [v1/org_policy_service.proto](#v1_org_policy_service-proto)
    - [CreatePolicyRequest](#bytebase-v1-CreatePolicyRequest)
    - [DataSourceQueryPolicy](#bytebase-v1-DataSourceQueryPolicy)
    - [DeletePolicyRequest](#bytebase-v1-DeletePolicyRequest)
    - [GetPolicyRequest](#bytebase-v1-GetPolicyRequest)
    - [ListPoliciesRequest](#bytebase-v1-ListPoliciesRequest)
    - [ListPoliciesResponse](#bytebase-v1-ListPoliciesResponse)
    - [MaskingExemptionPolicy](#bytebase-v1-MaskingExemptionPolicy)
    - [MaskingExemptionPolicy.Exemption](#bytebase-v1-MaskingExemptionPolicy-Exemption)
    - [MaskingRulePolicy](#bytebase-v1-MaskingRulePolicy)
    - [MaskingRulePolicy.MaskingRule](#bytebase-v1-MaskingRulePolicy-MaskingRule)
    - [Policy](#bytebase-v1-Policy)
    - [QueryDataPolicy](#bytebase-v1-QueryDataPolicy)
    - [RolloutPolicy](#bytebase-v1-RolloutPolicy)
    - [TagPolicy](#bytebase-v1-TagPolicy)
    - [TagPolicy.TagsEntry](#bytebase-v1-TagPolicy-TagsEntry)
    - [UpdatePolicyRequest](#bytebase-v1-UpdatePolicyRequest)
  
    - [DataSourceQueryPolicy.Restriction](#bytebase-v1-DataSourceQueryPolicy-Restriction)
    - [PolicyResourceType](#bytebase-v1-PolicyResourceType)
    - [PolicyType](#bytebase-v1-PolicyType)
  
    - [OrgPolicyService](#bytebase-v1-OrgPolicyService)
  
- [v1/sql_service.proto](#v1_sql_service-proto)
    - [AICompletionRequest](#bytebase-v1-AICompletionRequest)
    - [AICompletionRequest.Message](#bytebase-v1-AICompletionRequest-Message)
    - [AICompletionResponse](#bytebase-v1-AICompletionResponse)
    - [AICompletionResponse.Candidate](#bytebase-v1-AICompletionResponse-Candidate)
    - [AICompletionResponse.Candidate.Content](#bytebase-v1-AICompletionResponse-Candidate-Content)
    - [AICompletionResponse.Candidate.Content.Part](#bytebase-v1-AICompletionResponse-Candidate-Content-Part)
    - [AdminExecuteRequest](#bytebase-v1-AdminExecuteRequest)
    - [AdminExecuteResponse](#bytebase-v1-AdminExecuteResponse)
    - [Advice](#bytebase-v1-Advice)
    - [DiffMetadataRequest](#bytebase-v1-DiffMetadataRequest)
    - [DiffMetadataResponse](#bytebase-v1-DiffMetadataResponse)
    - [ExportRequest](#bytebase-v1-ExportRequest)
    - [ExportResponse](#bytebase-v1-ExportResponse)
    - [MaskingReason](#bytebase-v1-MaskingReason)
    - [QueryHistory](#bytebase-v1-QueryHistory)
    - [QueryOption](#bytebase-v1-QueryOption)
    - [QueryRequest](#bytebase-v1-QueryRequest)
    - [QueryResponse](#bytebase-v1-QueryResponse)
    - [QueryResult](#bytebase-v1-QueryResult)
    - [QueryResult.Message](#bytebase-v1-QueryResult-Message)
    - [QueryResult.PermissionDenied](#bytebase-v1-QueryResult-PermissionDenied)
    - [QueryResult.PostgresError](#bytebase-v1-QueryResult-PostgresError)
    - [QueryResult.SyntaxError](#bytebase-v1-QueryResult-SyntaxError)
    - [QueryRow](#bytebase-v1-QueryRow)
    - [RowValue](#bytebase-v1-RowValue)
    - [RowValue.Timestamp](#bytebase-v1-RowValue-Timestamp)
    - [RowValue.TimestampTZ](#bytebase-v1-RowValue-TimestampTZ)
    - [SearchQueryHistoriesRequest](#bytebase-v1-SearchQueryHistoriesRequest)
    - [SearchQueryHistoriesResponse](#bytebase-v1-SearchQueryHistoriesResponse)
  
    - [Advice.Level](#bytebase-v1-Advice-Level)
    - [Advice.RuleType](#bytebase-v1-Advice-RuleType)
    - [QueryHistory.Type](#bytebase-v1-QueryHistory-Type)
    - [QueryOption.MSSQLExplainFormat](#bytebase-v1-QueryOption-MSSQLExplainFormat)
    - [QueryOption.RedisRunCommandsOn](#bytebase-v1-QueryOption-RedisRunCommandsOn)
    - [QueryResult.Message.Level](#bytebase-v1-QueryResult-Message-Level)
    - [QueryResult.PermissionDenied.CommandType](#bytebase-v1-QueryResult-PermissionDenied-CommandType)
  
    - [SQLService](#bytebase-v1-SQLService)
  
- [v1/plan_service.proto](#v1_plan_service-proto)
    - [CancelPlanCheckRunRequest](#bytebase-v1-CancelPlanCheckRunRequest)
    - [CancelPlanCheckRunResponse](#bytebase-v1-CancelPlanCheckRunResponse)
    - [CreatePlanRequest](#bytebase-v1-CreatePlanRequest)
    - [GetPlanCheckRunRequest](#bytebase-v1-GetPlanCheckRunRequest)
    - [GetPlanRequest](#bytebase-v1-GetPlanRequest)
    - [ListPlansRequest](#bytebase-v1-ListPlansRequest)
    - [ListPlansResponse](#bytebase-v1-ListPlansResponse)
    - [Plan](#bytebase-v1-Plan)
    - [Plan.ChangeDatabaseConfig](#bytebase-v1-Plan-ChangeDatabaseConfig)
    - [Plan.ChangeDatabaseConfig.GhostFlagsEntry](#bytebase-v1-Plan-ChangeDatabaseConfig-GhostFlagsEntry)
    - [Plan.CreateDatabaseConfig](#bytebase-v1-Plan-CreateDatabaseConfig)
    - [Plan.ExportDataConfig](#bytebase-v1-Plan-ExportDataConfig)
    - [Plan.PlanCheckRunStatusCountEntry](#bytebase-v1-Plan-PlanCheckRunStatusCountEntry)
    - [Plan.Spec](#bytebase-v1-Plan-Spec)
    - [PlanCheckRun](#bytebase-v1-PlanCheckRun)
    - [PlanCheckRun.Result](#bytebase-v1-PlanCheckRun-Result)
    - [PlanCheckRun.Result.SqlReviewReport](#bytebase-v1-PlanCheckRun-Result-SqlReviewReport)
    - [PlanCheckRun.Result.SqlSummaryReport](#bytebase-v1-PlanCheckRun-Result-SqlSummaryReport)
    - [RunPlanChecksRequest](#bytebase-v1-RunPlanChecksRequest)
    - [RunPlanChecksResponse](#bytebase-v1-RunPlanChecksResponse)
    - [UpdatePlanRequest](#bytebase-v1-UpdatePlanRequest)
  
    - [PlanCheckRun.Result.Type](#bytebase-v1-PlanCheckRun-Result-Type)
    - [PlanCheckRun.Status](#bytebase-v1-PlanCheckRun-Status)
  
    - [PlanService](#bytebase-v1-PlanService)
  
- [v1/project_service.proto](#v1_project_service-proto)
    - [Activity](#bytebase-v1-Activity)
    - [AddWebhookRequest](#bytebase-v1-AddWebhookRequest)
    - [BatchDeleteProjectsRequest](#bytebase-v1-BatchDeleteProjectsRequest)
    - [BatchGetProjectsRequest](#bytebase-v1-BatchGetProjectsRequest)
    - [BatchGetProjectsResponse](#bytebase-v1-BatchGetProjectsResponse)
    - [CreateProjectRequest](#bytebase-v1-CreateProjectRequest)
    - [DeleteProjectRequest](#bytebase-v1-DeleteProjectRequest)
    - [GetProjectRequest](#bytebase-v1-GetProjectRequest)
    - [Label](#bytebase-v1-Label)
    - [ListProjectsRequest](#bytebase-v1-ListProjectsRequest)
    - [ListProjectsResponse](#bytebase-v1-ListProjectsResponse)
    - [Project](#bytebase-v1-Project)
    - [Project.ExecutionRetryPolicy](#bytebase-v1-Project-ExecutionRetryPolicy)
    - [Project.LabelsEntry](#bytebase-v1-Project-LabelsEntry)
    - [RemoveWebhookRequest](#bytebase-v1-RemoveWebhookRequest)
    - [SearchProjectsRequest](#bytebase-v1-SearchProjectsRequest)
    - [SearchProjectsResponse](#bytebase-v1-SearchProjectsResponse)
    - [TestWebhookRequest](#bytebase-v1-TestWebhookRequest)
    - [TestWebhookResponse](#bytebase-v1-TestWebhookResponse)
    - [UndeleteProjectRequest](#bytebase-v1-UndeleteProjectRequest)
    - [UpdateProjectRequest](#bytebase-v1-UpdateProjectRequest)
    - [UpdateWebhookRequest](#bytebase-v1-UpdateWebhookRequest)
    - [Webhook](#bytebase-v1-Webhook)
  
    - [Activity.Type](#bytebase-v1-Activity-Type)
  
    - [ProjectService](#bytebase-v1-ProjectService)
  
- [v1/release_service.proto](#v1_release_service-proto)
    - [CheckReleaseRequest](#bytebase-v1-CheckReleaseRequest)
    - [CheckReleaseResponse](#bytebase-v1-CheckReleaseResponse)
    - [CheckReleaseResponse.CheckResult](#bytebase-v1-CheckReleaseResponse-CheckResult)
    - [CreateReleaseRequest](#bytebase-v1-CreateReleaseRequest)
    - [DeleteReleaseRequest](#bytebase-v1-DeleteReleaseRequest)
    - [GetReleaseRequest](#bytebase-v1-GetReleaseRequest)
    - [ListReleasesRequest](#bytebase-v1-ListReleasesRequest)
    - [ListReleasesResponse](#bytebase-v1-ListReleasesResponse)
    - [Release](#bytebase-v1-Release)
    - [Release.File](#bytebase-v1-Release-File)
    - [Release.VCSSource](#bytebase-v1-Release-VCSSource)
    - [UndeleteReleaseRequest](#bytebase-v1-UndeleteReleaseRequest)
    - [UpdateReleaseRequest](#bytebase-v1-UpdateReleaseRequest)
  
    - [Release.Type](#bytebase-v1-Release-Type)
  
    - [ReleaseService](#bytebase-v1-ReleaseService)
  
- [v1/review_config_service.proto](#v1_review_config_service-proto)
    - [CreateReviewConfigRequest](#bytebase-v1-CreateReviewConfigRequest)
    - [DeleteReviewConfigRequest](#bytebase-v1-DeleteReviewConfigRequest)
    - [GetReviewConfigRequest](#bytebase-v1-GetReviewConfigRequest)
    - [ListReviewConfigsRequest](#bytebase-v1-ListReviewConfigsRequest)
    - [ListReviewConfigsResponse](#bytebase-v1-ListReviewConfigsResponse)
    - [ReviewConfig](#bytebase-v1-ReviewConfig)
    - [SQLReviewRule](#bytebase-v1-SQLReviewRule)
    - [SQLReviewRule.CommentConventionRulePayload](#bytebase-v1-SQLReviewRule-CommentConventionRulePayload)
    - [SQLReviewRule.NamingCaseRulePayload](#bytebase-v1-SQLReviewRule-NamingCaseRulePayload)
    - [SQLReviewRule.NamingRulePayload](#bytebase-v1-SQLReviewRule-NamingRulePayload)
    - [SQLReviewRule.NumberRulePayload](#bytebase-v1-SQLReviewRule-NumberRulePayload)
    - [SQLReviewRule.StringArrayRulePayload](#bytebase-v1-SQLReviewRule-StringArrayRulePayload)
    - [SQLReviewRule.StringRulePayload](#bytebase-v1-SQLReviewRule-StringRulePayload)
    - [UpdateReviewConfigRequest](#bytebase-v1-UpdateReviewConfigRequest)
  
    - [SQLReviewRule.Level](#bytebase-v1-SQLReviewRule-Level)
    - [SQLReviewRule.Type](#bytebase-v1-SQLReviewRule-Type)
  
    - [ReviewConfigService](#bytebase-v1-ReviewConfigService)
  
- [v1/revision_service.proto](#v1_revision_service-proto)
    - [BatchCreateRevisionsRequest](#bytebase-v1-BatchCreateRevisionsRequest)
    - [BatchCreateRevisionsResponse](#bytebase-v1-BatchCreateRevisionsResponse)
    - [CreateRevisionRequest](#bytebase-v1-CreateRevisionRequest)
    - [DeleteRevisionRequest](#bytebase-v1-DeleteRevisionRequest)
    - [GetRevisionRequest](#bytebase-v1-GetRevisionRequest)
    - [ListRevisionsRequest](#bytebase-v1-ListRevisionsRequest)
    - [ListRevisionsResponse](#bytebase-v1-ListRevisionsResponse)
    - [Revision](#bytebase-v1-Revision)
  
    - [Revision.Type](#bytebase-v1-Revision-Type)
  
    - [RevisionService](#bytebase-v1-RevisionService)
  
- [v1/role_service.proto](#v1_role_service-proto)
    - [CreateRoleRequest](#bytebase-v1-CreateRoleRequest)
    - [DeleteRoleRequest](#bytebase-v1-DeleteRoleRequest)
    - [GetRoleRequest](#bytebase-v1-GetRoleRequest)
    - [ListRolesRequest](#bytebase-v1-ListRolesRequest)
    - [ListRolesResponse](#bytebase-v1-ListRolesResponse)
    - [Role](#bytebase-v1-Role)
    - [UpdateRoleRequest](#bytebase-v1-UpdateRoleRequest)
  
    - [Role.Type](#bytebase-v1-Role-Type)
  
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
    - [GetTaskRunRequest](#bytebase-v1-GetTaskRunRequest)
    - [GetTaskRunSessionRequest](#bytebase-v1-GetTaskRunSessionRequest)
    - [ListRolloutsRequest](#bytebase-v1-ListRolloutsRequest)
    - [ListRolloutsResponse](#bytebase-v1-ListRolloutsResponse)
    - [ListTaskRunsRequest](#bytebase-v1-ListTaskRunsRequest)
    - [ListTaskRunsResponse](#bytebase-v1-ListTaskRunsResponse)
    - [PreviewTaskRunRollbackRequest](#bytebase-v1-PreviewTaskRunRollbackRequest)
    - [PreviewTaskRunRollbackResponse](#bytebase-v1-PreviewTaskRunRollbackResponse)
    - [Rollout](#bytebase-v1-Rollout)
    - [Stage](#bytebase-v1-Stage)
    - [Task](#bytebase-v1-Task)
    - [Task.DatabaseCreate](#bytebase-v1-Task-DatabaseCreate)
    - [Task.DatabaseDataExport](#bytebase-v1-Task-DatabaseDataExport)
    - [Task.DatabaseUpdate](#bytebase-v1-Task-DatabaseUpdate)
    - [TaskRun](#bytebase-v1-TaskRun)
    - [TaskRun.SchedulerInfo](#bytebase-v1-TaskRun-SchedulerInfo)
    - [TaskRun.SchedulerInfo.WaitingCause](#bytebase-v1-TaskRun-SchedulerInfo-WaitingCause)
    - [TaskRunLog](#bytebase-v1-TaskRunLog)
    - [TaskRunLogEntry](#bytebase-v1-TaskRunLogEntry)
    - [TaskRunLogEntry.CommandExecute](#bytebase-v1-TaskRunLogEntry-CommandExecute)
    - [TaskRunLogEntry.CommandExecute.CommandResponse](#bytebase-v1-TaskRunLogEntry-CommandExecute-CommandResponse)
    - [TaskRunLogEntry.ComputeDiff](#bytebase-v1-TaskRunLogEntry-ComputeDiff)
    - [TaskRunLogEntry.DatabaseSync](#bytebase-v1-TaskRunLogEntry-DatabaseSync)
    - [TaskRunLogEntry.PriorBackup](#bytebase-v1-TaskRunLogEntry-PriorBackup)
    - [TaskRunLogEntry.PriorBackup.PriorBackupDetail](#bytebase-v1-TaskRunLogEntry-PriorBackup-PriorBackupDetail)
    - [TaskRunLogEntry.PriorBackup.PriorBackupDetail.Item](#bytebase-v1-TaskRunLogEntry-PriorBackup-PriorBackupDetail-Item)
    - [TaskRunLogEntry.PriorBackup.PriorBackupDetail.Item.Table](#bytebase-v1-TaskRunLogEntry-PriorBackup-PriorBackupDetail-Item-Table)
    - [TaskRunLogEntry.ReleaseFileExecute](#bytebase-v1-TaskRunLogEntry-ReleaseFileExecute)
    - [TaskRunLogEntry.RetryInfo](#bytebase-v1-TaskRunLogEntry-RetryInfo)
    - [TaskRunLogEntry.SchemaDump](#bytebase-v1-TaskRunLogEntry-SchemaDump)
    - [TaskRunLogEntry.TransactionControl](#bytebase-v1-TaskRunLogEntry-TransactionControl)
    - [TaskRunSession](#bytebase-v1-TaskRunSession)
    - [TaskRunSession.Postgres](#bytebase-v1-TaskRunSession-Postgres)
    - [TaskRunSession.Postgres.Session](#bytebase-v1-TaskRunSession-Postgres-Session)
  
    - [Task.Status](#bytebase-v1-Task-Status)
    - [Task.Type](#bytebase-v1-Task-Type)
    - [TaskRun.ExportArchiveStatus](#bytebase-v1-TaskRun-ExportArchiveStatus)
    - [TaskRun.Status](#bytebase-v1-TaskRun-Status)
    - [TaskRunLogEntry.TransactionControl.Type](#bytebase-v1-TaskRunLogEntry-TransactionControl-Type)
    - [TaskRunLogEntry.Type](#bytebase-v1-TaskRunLogEntry-Type)
  
    - [RolloutService](#bytebase-v1-RolloutService)
  
- [v1/sheet_service.proto](#v1_sheet_service-proto)
    - [BatchCreateSheetsRequest](#bytebase-v1-BatchCreateSheetsRequest)
    - [BatchCreateSheetsResponse](#bytebase-v1-BatchCreateSheetsResponse)
    - [CreateSheetRequest](#bytebase-v1-CreateSheetRequest)
    - [GetSheetRequest](#bytebase-v1-GetSheetRequest)
    - [Sheet](#bytebase-v1-Sheet)
  
    - [SheetService](#bytebase-v1-SheetService)
  
- [v1/subscription_service.proto](#v1_subscription_service-proto)
    - [GetSubscriptionRequest](#bytebase-v1-GetSubscriptionRequest)
    - [PlanConfig](#bytebase-v1-PlanConfig)
    - [PlanLimitConfig](#bytebase-v1-PlanLimitConfig)
    - [Subscription](#bytebase-v1-Subscription)
    - [UpdateSubscriptionRequest](#bytebase-v1-UpdateSubscriptionRequest)
  
    - [PlanFeature](#bytebase-v1-PlanFeature)
    - [PlanType](#bytebase-v1-PlanType)
  
    - [SubscriptionService](#bytebase-v1-SubscriptionService)
  
- [v1/worksheet_service.proto](#v1_worksheet_service-proto)
    - [BatchUpdateWorksheetOrganizerRequest](#bytebase-v1-BatchUpdateWorksheetOrganizerRequest)
    - [BatchUpdateWorksheetOrganizerResponse](#bytebase-v1-BatchUpdateWorksheetOrganizerResponse)
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
Authorization method for RPC calls.

| Name | Number | Description |
| ---- | ------ | ----------- |
| AUTH_METHOD_UNSPECIFIED | 0 | Unspecified authorization method. |
| IAM | 1 | Standard IAM authorization check on organizational resources. |
| CUSTOM | 2 | Custom authorization logic. |


 


<a name="v1_annotation-proto-extensions"></a>

### File-level Extensions
| Extension | Type | Base | Number | Description |
| --------- | ---- | ---- | ------ | ----------- |
| allow_without_credential | bool | .google.protobuf.MethodOptions | 100000 | Whether the method allows access without authentication credentials. |
| audit | bool | .google.protobuf.MethodOptions | 100003 | Whether to audit calls to this method. |
| auth_method | AuthMethod | .google.protobuf.MethodOptions | 100002 | The authorization method to use for this RPC. |
| permission | string | .google.protobuf.MethodOptions | 100001 | The permission required to call this method. |

 

 



<a name="v1_common-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/common.proto



<a name="bytebase-v1-PermissionDeniedDetail"></a>

### PermissionDeniedDetail
PermissionDeniedDetail provides structured information about permission failures.
Used as error detail when returning CodePermissionDenied errors.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| method | [string](#string) |  | The API method that was called. |
| required_permissions | [string](#string) | repeated | The permissions required but not granted to the user. |
| resources | [string](#string) | repeated | The resources the user was trying to access. |






<a name="bytebase-v1-Position"></a>

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






<a name="bytebase-v1-Range"></a>

### Range
Range represents a span within a text or sequence.
Whether the indices are byte offsets or character indices depends on the context.
Check the documentation of the field using Range for specific semantics.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start | [int32](#int32) |  | Start index (inclusive). |
| end | [int32](#int32) |  | End index (exclusive). |





 


<a name="bytebase-v1-Engine"></a>

### Engine
Database engine type.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ENGINE_UNSPECIFIED | 0 | Unspecified database engine. |
| CLICKHOUSE | 1 | ClickHouse columnar database. |
| MYSQL | 2 | MySQL relational database. |
| POSTGRES | 3 | PostgreSQL relational database. |
| SNOWFLAKE | 4 | Snowflake cloud data warehouse. |
| SQLITE | 5 | SQLite embedded database. |
| TIDB | 6 | TiDB distributed SQL database. |
| MONGODB | 7 | MongoDB document database. |
| REDIS | 8 | Redis key-value store. |
| ORACLE | 9 | Oracle relational database. |
| SPANNER | 10 | Google Cloud Spanner distributed database. |
| MSSQL | 11 | Microsoft SQL Server relational database. |
| REDSHIFT | 12 | Amazon Redshift data warehouse. |
| MARIADB | 13 | MariaDB relational database. |
| OCEANBASE | 14 | OceanBase distributed database. |
| STARROCKS | 15 | StarRocks analytics database. |
| DORIS | 16 | Apache Doris analytics database. |
| HIVE | 17 | Apache Hive data warehouse. |
| ELASTICSEARCH | 18 | Elasticsearch search engine. |
| BIGQUERY | 19 | Google BigQuery data warehouse. |
| DYNAMODB | 20 | Amazon DynamoDB NoSQL database. |
| DATABRICKS | 21 | Databricks lakehouse platform. |
| COCKROACHDB | 22 | CockroachDB distributed SQL database. |
| COSMOSDB | 23 | Azure Cosmos DB multi-model database. |
| TRINO | 24 | Trino distributed SQL query engine. |
| CASSANDRA | 25 | Apache Cassandra NoSQL database. |



<a name="bytebase-v1-ExportFormat"></a>

### ExportFormat
Data export format.

| Name | Number | Description |
| ---- | ------ | ----------- |
| FORMAT_UNSPECIFIED | 0 | Unspecified export format. |
| CSV | 1 | Comma-separated values format. |
| JSON | 2 | JavaScript Object Notation format. |
| SQL | 3 | SQL statements format. |
| XLSX | 4 | Microsoft Excel spreadsheet format. |



<a name="bytebase-v1-RiskLevel"></a>

### RiskLevel
RiskLevel is the risk level.

| Name | Number | Description |
| ---- | ------ | ----------- |
| RISK_LEVEL_UNSPECIFIED | 0 | Unspecified risk level. |
| LOW | 1 | Low risk operation. |
| MODERATE | 2 | Moderate risk operation. |
| HIGH | 3 | High risk operation. |



<a name="bytebase-v1-State"></a>

### State
Resource lifecycle state.

| Name | Number | Description |
| ---- | ------ | ----------- |
| STATE_UNSPECIFIED | 0 |  |
| ACTIVE | 1 | Resource is active and operational. |
| DELETED | 2 | Resource is soft-deleted but may be recoverable. |



<a name="bytebase-v1-VCSType"></a>

### VCSType
Version control system type.

| Name | Number | Description |
| ---- | ------ | ----------- |
| VCS_TYPE_UNSPECIFIED | 0 | Unspecified VCS type. |
| GITHUB | 1 | GitHub version control platform. |
| GITLAB | 2 | GitLab version control platform. |
| BITBUCKET | 3 | Bitbucket version control platform. |
| AZURE_DEVOPS | 4 | Azure DevOps version control platform. |



<a name="bytebase-v1-WebhookType"></a>

### WebhookType
Webhook integration type.

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


 

 

 



<a name="v1_issue_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/issue_service.proto



<a name="bytebase-v1-ApprovalFlow"></a>

### ApprovalFlow



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| roles | [string](#string) | repeated | The roles required for approval in order. |






<a name="bytebase-v1-ApprovalTemplate"></a>

### ApprovalTemplate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| flow | [ApprovalFlow](#bytebase-v1-ApprovalFlow) |  | The approval flow definition. |
| title | [string](#string) |  | The title of the approval template. |
| description | [string](#string) |  | The description of the approval template. |






<a name="bytebase-v1-ApproveIssueRequest"></a>

### ApproveIssueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the issue to add an approver. Format: projects/{project}/issues/{issue} |
| comment | [string](#string) |  | The comment explaining the approval decision. |






<a name="bytebase-v1-BatchUpdateIssuesStatusRequest"></a>

### BatchUpdateIssuesStatusRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource shared by all issues being updated. Format: projects/{project} If the operation spans parents, a dash (-) may be accepted as a wildcard. We only support updating the status of databases for now. |
| issues | [string](#string) | repeated | The list of issues to update. Format: projects/{project}/issues/{issue} |
| status | [IssueStatus](#bytebase-v1-IssueStatus) |  | The new status. |
| reason | [string](#string) |  | The reason for the status change. |






<a name="bytebase-v1-BatchUpdateIssuesStatusResponse"></a>

### BatchUpdateIssuesStatusResponse







<a name="bytebase-v1-CreateIssueCommentRequest"></a>

### CreateIssueCommentRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The issue name Format: projects/{project}/issues/{issue} |
| issue_comment | [IssueComment](#bytebase-v1-IssueComment) |  | The comment to create. |






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
| force | [bool](#bool) |  | If set to true, bypass cache and fetch the latest data. |






<a name="bytebase-v1-GrantRequest"></a>

### GrantRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [string](#string) |  | The requested role. Format: roles/EXPORTER. |
| user | [string](#string) |  | The user to be granted. Format: users/{email}. |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition for the role. Same as the condition in IAM Binding message. |
| expiration | [google.protobuf.Duration](#google-protobuf-Duration) |  | The duration for which the grant is valid. |






<a name="bytebase-v1-Issue"></a>

### Issue



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the issue. Format: projects/{project}/issues/{issue} |
| title | [string](#string) |  | The title of the issue. |
| description | [string](#string) |  | The description of the issue. |
| type | [Issue.Type](#bytebase-v1-Issue-Type) |  |  |
| status | [IssueStatus](#bytebase-v1-IssueStatus) |  | The status of the issue. |
| approvers | [Issue.Approver](#bytebase-v1-Issue-Approver) | repeated |  |
| approval_template | [ApprovalTemplate](#bytebase-v1-ApprovalTemplate) |  | The approval template for the issue. |
| creator | [string](#string) |  | Format: users/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| plan | [string](#string) |  | The plan associated with the issue. Can be empty. Format: projects/{project}/plans/{plan} |
| grant_request | [GrantRequest](#bytebase-v1-GrantRequest) |  | Used if the issue type is GRANT_REQUEST. |
| risk_level | [RiskLevel](#bytebase-v1-RiskLevel) |  | The risk level of the issue. |
| labels | [string](#string) | repeated | Labels attached to the issue for categorization and filtering. |
| approval_status | [Issue.ApprovalStatus](#bytebase-v1-Issue-ApprovalStatus) |  |  |






<a name="bytebase-v1-Issue-Approver"></a>

### Issue.Approver
Approvers and their approval status for the issue.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [Issue.Approver.Status](#bytebase-v1-Issue-Approver-Status) |  | The new status. |
| principal | [string](#string) |  | Format: users/hello@world.com |






<a name="bytebase-v1-IssueComment"></a>

### IssueComment
A comment on an issue.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/issues/{issue}/issueComments/{issueComment-uid} |
| comment | [string](#string) |  | The text content of the comment. |
| payload | [string](#string) |  | TODO: use struct message instead. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| creator | [string](#string) |  | Format: users/{email} |
| approval | [IssueComment.Approval](#bytebase-v1-IssueComment-Approval) |  | Approval event. |
| issue_update | [IssueComment.IssueUpdate](#bytebase-v1-IssueComment-IssueUpdate) |  | Issue update event. |
| plan_spec_update | [IssueComment.PlanSpecUpdate](#bytebase-v1-IssueComment-PlanSpecUpdate) |  | Plan spec update event. |






<a name="bytebase-v1-IssueComment-Approval"></a>

### IssueComment.Approval
Approval event information.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [IssueComment.Approval.Status](#bytebase-v1-IssueComment-Approval-Status) |  | The approval status. |






<a name="bytebase-v1-IssueComment-IssueUpdate"></a>

### IssueComment.IssueUpdate
Issue update event information.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| from_title | [string](#string) | optional |  |
| to_title | [string](#string) | optional |  |
| from_description | [string](#string) | optional |  |
| to_description | [string](#string) | optional |  |
| from_status | [IssueStatus](#bytebase-v1-IssueStatus) | optional |  |
| to_status | [IssueStatus](#bytebase-v1-IssueStatus) | optional |  |
| from_labels | [string](#string) | repeated |  |
| to_labels | [string](#string) | repeated |  |






<a name="bytebase-v1-IssueComment-PlanSpecUpdate"></a>

### IssueComment.PlanSpecUpdate
Plan spec update event information (tracks sheet changes to plan specs).


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| spec | [string](#string) |  | The spec that was updated. Format: projects/{project}/plans/{plan}/specs/{spec} |
| from_sheet | [string](#string) | optional | The previous sheet. Format: projects/{project}/sheets/{sheet} |
| to_sheet | [string](#string) | optional | The new sheet. Format: projects/{project}/sheets/{sheet} |






<a name="bytebase-v1-ListIssueCommentsRequest"></a>

### ListIssueCommentsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{projects}/issues/{issue} |
| page_size | [int32](#int32) |  | The maximum number of issue comments to return. The service may return fewer than this value. If unspecified, at most 10 issue comments will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListIssueComments` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListIssueComments` must match the call that provided the page token. |






<a name="bytebase-v1-ListIssueCommentsResponse"></a>

### ListIssueCommentsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issue_comments | [IssueComment](#bytebase-v1-IssueComment) | repeated | The issue comments from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-ListIssuesRequest"></a>

### ListIssuesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of issues. Format: projects/{project} |
| page_size | [int32](#int32) |  | The maximum number of issues to return. The service may return fewer than this value. If unspecified, at most 10 issues will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListIssues` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListIssues` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter issues returned in the list. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filters: - creator: issue creator full name in &#34;users/{email or id}&#34; format, support &#34;==&#34; operator. - status: the issue status, support &#34;==&#34; and &#34;in&#34; operator, check the IssueStatus enum for the values. - create_time: issue create time in &#34;2006-01-02T15:04:05Z07:00&#34; format, support &#34;&gt;=&#34; or &#34;&lt;=&#34; operator. - type: the issue type, support &#34;==&#34; and &#34;in&#34; operator, check the Type enum in the Issue message for the values. - labels: the issue labels, support &#34;==&#34; and &#34;in&#34; operator. - approval_status: issue approval status, support &#34;==&#34; operator. - current_approver: the issue approver, should in &#34;users/{email} format&#34;, support &#34;==&#34; operator.

For example: creator == &#34;users/ed@bytebase.com&#34; &amp;&amp; status in [&#34;OPEN&#34;, &#34;DONE&#34;] status == &#34;CANCELED&#34; &amp;&amp; type == &#34;DATABASE_CHANGE&#34; labels in [&#34;label1&#34;, &#34;label2&#34;] create_time &gt;= &#34;2025-01-02T15:04:05Z07:00&#34; |
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
| name | [string](#string) |  | The name of the issue to add a rejection. Format: projects/{project}/issues/{issue} |
| comment | [string](#string) |  | The comment explaining the rejection decision. |






<a name="bytebase-v1-RequestIssueRequest"></a>

### RequestIssueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the issue to request a issue. Format: projects/{project}/issues/{issue} |
| comment | [string](#string) |  | The comment explaining the request. |






<a name="bytebase-v1-SearchIssuesRequest"></a>

### SearchIssuesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of issues. Format: projects/{project} Use &#34;projects/-&#34; to list all issues from all projects. |
| page_size | [int32](#int32) |  | The maximum number of issues to return. The service may return fewer than this value. If unspecified, at most 10 issues will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `SearchIssues` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `SearchIssues` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter issues returned in the list. Check the filter field in the ListIssuesRequest message. |
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
| issue_comment | [IssueComment](#bytebase-v1-IssueComment) |  | The comment to update. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the issue comment is not found, a new issue comment will be created. In this situation, `update_mask` is ignored. |






<a name="bytebase-v1-UpdateIssueRequest"></a>

### UpdateIssueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issue | [Issue](#bytebase-v1-Issue) |  | The issue to update.

The issue&#39;s `name` field is used to identify the issue to update. Format: projects/{project}/issues/{issue} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the issue is not found, a new issue will be created. In this situation, `update_mask` is ignored. |





 


<a name="bytebase-v1-Issue-ApprovalStatus"></a>

### Issue.ApprovalStatus
The overall approval status for the issue.

| Name | Number | Description |
| ---- | ------ | ----------- |
| APPROVAL_STATUS_UNSPECIFIED | 0 | Unspecified approval status. |
| CHECKING | 1 | Approval checks are being evaluated. |
| PENDING | 2 | Approval is pending. |
| APPROVED | 3 | Issue has been approved. |
| REJECTED | 4 | Issue has been rejected. |
| SKIPPED | 5 | Approval was skipped. |



<a name="bytebase-v1-Issue-Approver-Status"></a>

### Issue.Approver.Status
The approval status of an approver.

| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 | Unspecified status. |
| PENDING | 1 | Awaiting approval. |
| APPROVED | 2 | Approved. |
| REJECTED | 3 | Rejected. |



<a name="bytebase-v1-Issue-Type"></a>

### Issue.Type
The type of issue.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 | Unspecified issue type. |
| DATABASE_CHANGE | 1 | Database schema or data change. |
| GRANT_REQUEST | 2 | Database access grant request. |
| DATABASE_EXPORT | 3 | Database data export request. |



<a name="bytebase-v1-IssueComment-Approval-Status"></a>

### IssueComment.Approval.Status
Approval status values.

| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 | Unspecified status. |
| PENDING | 1 | Approval pending. |
| APPROVED | 2 | Approved. |
| REJECTED | 3 | Rejected. |



<a name="bytebase-v1-IssueStatus"></a>

### IssueStatus
The status of an issue.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ISSUE_STATUS_UNSPECIFIED | 0 | Unspecified status. |
| OPEN | 1 | Issue is open and active. |
| DONE | 2 | Issue is completed. |
| CANCELED | 3 | Issue is canceled. |


 

 


<a name="bytebase-v1-IssueService"></a>

### IssueService
IssueService manages issues for tracking database changes and tasks.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetIssue | [GetIssueRequest](#bytebase-v1-GetIssueRequest) | [Issue](#bytebase-v1-Issue) | Retrieves an issue by name. Permissions required: bb.issues.get |
| CreateIssue | [CreateIssueRequest](#bytebase-v1-CreateIssueRequest) | [Issue](#bytebase-v1-Issue) | Creates a new issue for database changes or tasks. Permissions required: bb.issues.create |
| ListIssues | [ListIssuesRequest](#bytebase-v1-ListIssuesRequest) | [ListIssuesResponse](#bytebase-v1-ListIssuesResponse) | Lists issues in a project. Permissions required: bb.issues.list |
| SearchIssues | [SearchIssuesRequest](#bytebase-v1-SearchIssuesRequest) | [SearchIssuesResponse](#bytebase-v1-SearchIssuesResponse) | Search for issues that the caller has the bb.issues.get permission on and also satisfy the specified filter &amp; query. Permissions required: bb.issues.get |
| UpdateIssue | [UpdateIssueRequest](#bytebase-v1-UpdateIssueRequest) | [Issue](#bytebase-v1-Issue) | Updates an issue&#39;s properties such as title, description, or labels. Permissions required: bb.issues.update |
| ListIssueComments | [ListIssueCommentsRequest](#bytebase-v1-ListIssueCommentsRequest) | [ListIssueCommentsResponse](#bytebase-v1-ListIssueCommentsResponse) | Lists comments on an issue. Permissions required: bb.issueComments.list |
| CreateIssueComment | [CreateIssueCommentRequest](#bytebase-v1-CreateIssueCommentRequest) | [IssueComment](#bytebase-v1-IssueComment) | Adds a comment to an issue. Permissions required: bb.issueComments.create |
| UpdateIssueComment | [UpdateIssueCommentRequest](#bytebase-v1-UpdateIssueCommentRequest) | [IssueComment](#bytebase-v1-IssueComment) | Updates an existing issue comment. Permissions required: bb.issueComments.update |
| BatchUpdateIssuesStatus | [BatchUpdateIssuesStatusRequest](#bytebase-v1-BatchUpdateIssuesStatusRequest) | [BatchUpdateIssuesStatusResponse](#bytebase-v1-BatchUpdateIssuesStatusResponse) | Updates the status of multiple issues in a single operation. Permissions required: bb.issues.update |
| ApproveIssue | [ApproveIssueRequest](#bytebase-v1-ApproveIssueRequest) | [Issue](#bytebase-v1-Issue) | Approves an issue. Access determined by approval flow configuration - caller must be a designated approver for the current approval step. Permissions required: None (determined by approval flow) |
| RejectIssue | [RejectIssueRequest](#bytebase-v1-RejectIssueRequest) | [Issue](#bytebase-v1-Issue) | Rejects an issue. Access determined by approval flow configuration - caller must be a designated approver for the current approval step. Permissions required: None (determined by approval flow) |
| RequestIssue | [RequestIssueRequest](#bytebase-v1-RequestIssueRequest) | [Issue](#bytebase-v1-Issue) | Requests changes on an issue. Access determined by approval flow configuration - caller must be a designated approver for the current approval step. Permissions required: None (determined by approval flow) |

 



<a name="v1_setting_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/setting_service.proto



<a name="bytebase-v1-AISetting"></a>

### AISetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| provider | [AISetting.Provider](#bytebase-v1-AISetting-Provider) |  |  |
| endpoint | [string](#string) |  |  |
| api_key | [string](#string) |  |  |
| model | [string](#string) |  |  |
| version | [string](#string) |  |  |






<a name="bytebase-v1-Algorithm"></a>

### Algorithm



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| full_mask | [Algorithm.FullMask](#bytebase-v1-Algorithm-FullMask) |  |  |
| range_mask | [Algorithm.RangeMask](#bytebase-v1-Algorithm-RangeMask) |  |  |
| md5_mask | [Algorithm.MD5Mask](#bytebase-v1-Algorithm-MD5Mask) |  |  |
| inner_outer_mask | [Algorithm.InnerOuterMask](#bytebase-v1-Algorithm-InnerOuterMask) |  |  |






<a name="bytebase-v1-Algorithm-FullMask"></a>

### Algorithm.FullMask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| substitution | [string](#string) |  | substitution is the string used to replace the original value, the max length of the string is 16 bytes. |






<a name="bytebase-v1-Algorithm-InnerOuterMask"></a>

### Algorithm.InnerOuterMask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| prefix_len | [int32](#int32) |  |  |
| suffix_len | [int32](#int32) |  |  |
| type | [Algorithm.InnerOuterMask.MaskType](#bytebase-v1-Algorithm-InnerOuterMask-MaskType) |  |  |
| substitution | [string](#string) |  |  |






<a name="bytebase-v1-Algorithm-MD5Mask"></a>

### Algorithm.MD5Mask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| salt | [string](#string) |  | salt is the salt value to generate a different hash that with the word alone. |






<a name="bytebase-v1-Algorithm-RangeMask"></a>

### Algorithm.RangeMask



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| slices | [Algorithm.RangeMask.Slice](#bytebase-v1-Algorithm-RangeMask-Slice) | repeated | We store it as a repeated field to face the fact that the original value may have multiple parts should be masked. But frontend can be started with a single rule easily. |






<a name="bytebase-v1-Algorithm-RangeMask-Slice"></a>

### Algorithm.RangeMask.Slice



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start | [int32](#int32) |  | start is the start character index (0-based) of the original value, should be less than end. Uses character indices (not byte offsets) for display-oriented masking. Example: For &#34;你好world&#34;, character index 2 refers to &#39;w&#39; (the 3rd character). |
| end | [int32](#int32) |  | end is the end character index (exclusive) of the original value. Uses character indices (not byte offsets) for display-oriented masking. |
| substitution | [string](#string) |  | substitution is the string used to replace the OriginalValue[start:end). |






<a name="bytebase-v1-Announcement"></a>

### Announcement



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| level | [Announcement.AlertLevel](#bytebase-v1-Announcement-AlertLevel) |  | The alert level of announcement |
| text | [string](#string) |  | The text of announcement |
| link | [string](#string) |  | The optional link, user can follow the link to check extra details |






<a name="bytebase-v1-AppIMSetting"></a>

### AppIMSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| settings | [AppIMSetting.IMSetting](#bytebase-v1-AppIMSetting-IMSetting) | repeated |  |






<a name="bytebase-v1-AppIMSetting-DingTalk"></a>

### AppIMSetting.DingTalk



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| client_id | [string](#string) |  |  |
| client_secret | [string](#string) |  |  |
| robot_code | [string](#string) |  |  |






<a name="bytebase-v1-AppIMSetting-Feishu"></a>

### AppIMSetting.Feishu



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_id | [string](#string) |  |  |
| app_secret | [string](#string) |  |  |






<a name="bytebase-v1-AppIMSetting-IMSetting"></a>

### AppIMSetting.IMSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [WebhookType](#bytebase-v1-WebhookType) |  |  |
| slack | [AppIMSetting.Slack](#bytebase-v1-AppIMSetting-Slack) |  |  |
| feishu | [AppIMSetting.Feishu](#bytebase-v1-AppIMSetting-Feishu) |  |  |
| wecom | [AppIMSetting.Wecom](#bytebase-v1-AppIMSetting-Wecom) |  |  |
| lark | [AppIMSetting.Lark](#bytebase-v1-AppIMSetting-Lark) |  |  |
| dingtalk | [AppIMSetting.DingTalk](#bytebase-v1-AppIMSetting-DingTalk) |  |  |
| teams | [AppIMSetting.Teams](#bytebase-v1-AppIMSetting-Teams) |  |  |






<a name="bytebase-v1-AppIMSetting-Lark"></a>

### AppIMSetting.Lark



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_id | [string](#string) |  |  |
| app_secret | [string](#string) |  |  |






<a name="bytebase-v1-AppIMSetting-Slack"></a>

### AppIMSetting.Slack



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  |  |






<a name="bytebase-v1-AppIMSetting-Teams"></a>

### AppIMSetting.Teams



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tenant_id | [string](#string) |  | Azure AD tenant ID (Directory ID). |
| client_id | [string](#string) |  | Azure AD application (client) ID. |
| client_secret | [string](#string) |  | Azure AD client secret. |






<a name="bytebase-v1-AppIMSetting-Wecom"></a>

### AppIMSetting.Wecom



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
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






<a name="bytebase-v1-EnvironmentSetting"></a>

### EnvironmentSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| environments | [EnvironmentSetting.Environment](#bytebase-v1-EnvironmentSetting-Environment) | repeated |  |






<a name="bytebase-v1-EnvironmentSetting-Environment"></a>

### EnvironmentSetting.Environment



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The resource name of the environment. Format: environments/{environment}. Output only. |
| id | [string](#string) |  | The resource id of the environment. This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |
| title | [string](#string) |  | The display name of the environment. |
| tags | [EnvironmentSetting.Environment.TagsEntry](#bytebase-v1-EnvironmentSetting-Environment-TagsEntry) | repeated |  |
| color | [string](#string) |  |  |






<a name="bytebase-v1-EnvironmentSetting-Environment-TagsEntry"></a>

### EnvironmentSetting.Environment.TagsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






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







<a name="bytebase-v1-ListSettingsResponse"></a>

### ListSettingsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| settings | [Setting](#bytebase-v1-Setting) | repeated | The settings from the specified request. |






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
| algorithm | [Algorithm](#bytebase-v1-Algorithm) |  |  |
| icon | [string](#string) |  | icon is the icon for semantic type, it can be emoji or base64 encoded image. |






<a name="bytebase-v1-Setting"></a>

### Setting
The schema of setting.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The resource name of the setting. Format: settings/{setting} Example: &#34;settings/SEMANTIC_TYPES&#34; |
| value | [SettingValue](#bytebase-v1-SettingValue) |  | The configuration value of the setting. |






<a name="bytebase-v1-SettingValue"></a>

### SettingValue
The data in setting value.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| app_im | [AppIMSetting](#bytebase-v1-AppIMSetting) |  |  |
| workspace_profile | [WorkspaceProfileSetting](#bytebase-v1-WorkspaceProfileSetting) |  |  |
| workspace_approval | [WorkspaceApprovalSetting](#bytebase-v1-WorkspaceApprovalSetting) |  |  |
| data_classification | [DataClassificationSetting](#bytebase-v1-DataClassificationSetting) |  |  |
| semantic_type | [SemanticTypeSetting](#bytebase-v1-SemanticTypeSetting) |  |  |
| ai | [AISetting](#bytebase-v1-AISetting) |  |  |
| environment | [EnvironmentSetting](#bytebase-v1-EnvironmentSetting) |  |  |






<a name="bytebase-v1-UpdateSettingRequest"></a>

### UpdateSettingRequest
The request message for updating or creating a setting.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| setting | [Setting](#bytebase-v1-Setting) |  | The setting to update. |
| validate_only | [bool](#bool) |  | validate_only is a flag to indicate whether to validate the setting value, server would not persist the setting value if it is true. |
| allow_missing | [bool](#bool) |  |  |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  |  |






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
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with the rule. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

The `source` field filters which rules apply. The `condition` field then evaluates with full context.

All supported variables: statement.affected_rows: affected row count in the DDL/DML, support &#34;==&#34;, &#34;!=&#34;, &#34;&lt;&#34;, &#34;&lt;=&#34;, &#34;&gt;&#34;, &#34;&gt;=&#34; operations. statement.table_rows: table row count number, support &#34;==&#34;, &#34;!=&#34;, &#34;&lt;&#34;, &#34;&lt;=&#34;, &#34;&gt;&#34;, &#34;&gt;=&#34; operations. resource.environment_id: the environment resource id, support &#34;==&#34;, &#34;!=&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34; operations. resource.project_id: the project resource id, support &#34;==&#34;, &#34;!=&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34;, &#34;contains()&#34;, &#34;matches()&#34;, &#34;startsWith()&#34;, &#34;endsWith()&#34; operations. resource.db_engine: the database engine type, support &#34;==&#34;, &#34;!=&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34; operations. Check the Engine enum for values. statement.sql_type: the SQL type, support &#34;==&#34;, &#34;!=&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34; operations. resource.database_name: the database name, support &#34;==&#34;, &#34;!=&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34;, &#34;contains()&#34;, &#34;matches()&#34;, &#34;startsWith()&#34;, &#34;endsWith()&#34; operations. resource.schema_name: the schema name, support &#34;==&#34;, &#34;!=&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34;, &#34;contains()&#34;, &#34;matches()&#34;, &#34;startsWith()&#34;, &#34;endsWith()&#34; operations. resource.table_name: the table name, support &#34;==&#34;, &#34;!=&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34;, &#34;contains()&#34;, &#34;matches()&#34;, &#34;startsWith()&#34;, &#34;endsWith()&#34; operations. statement.text: the SQL statement, support &#34;contains()&#34;, &#34;matches()&#34;, &#34;startsWith()&#34;, &#34;endsWith()&#34; operations. request.expiration_days: the role expiration days for the request, support &#34;==&#34;, &#34;!=&#34;, &#34;&lt;&#34;, &#34;&lt;=&#34;, &#34;&gt;&#34;, &#34;&gt;=&#34; operations. request.role: the request role full name, support &#34;==&#34;, &#34;!=&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34;, &#34;contains()&#34;, &#34;matches()&#34;, &#34;startsWith()&#34;, &#34;endsWith()&#34; operations.

When source is CHANGE_DATABASE, support: statement.*, resource.* (excluding request.*) When source is CREATE_DATABASE, support: resource.environment_id, resource.project_id, resource.db_engine, resource.database_name When source is EXPORT_DATA, support: resource.environment_id, resource.project_id, resource.db_engine, resource.database_name, resource.schema_name, resource.table_name When source is REQUEST_ROLE, support: resource.project_id, request.expiration_days, request.role

For examples: resource.environment_id == &#34;prod&#34; &amp;&amp; statement.affected_rows &gt;= 100 resource.table_name.matches(&#34;sensitive_.*&#34;) &amp;&amp; resource.db_engine == &#34;MYSQL&#34; |
| source | [WorkspaceApprovalSetting.Rule.Source](#bytebase-v1-WorkspaceApprovalSetting-Rule-Source) |  |  |






<a name="bytebase-v1-WorkspaceProfileSetting"></a>

### WorkspaceProfileSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| external_url | [string](#string) |  | The external URL is used for sso authentication callback. |
| disallow_signup | [bool](#bool) |  | Disallow self-service signup, users can only be invited by the owner. |
| require_2fa | [bool](#bool) |  | Require 2FA for all users. |
| refresh_token_duration | [google.protobuf.Duration](#google-protobuf-Duration) |  | The duration for refresh token. Default is 7 days. |
| access_token_duration | [google.protobuf.Duration](#google-protobuf-Duration) |  | The duration for access token. Default is 1 hour. |
| announcement | [Announcement](#bytebase-v1-Announcement) |  | The setting of custom announcement |
| maximum_role_expiration | [google.protobuf.Duration](#google-protobuf-Duration) |  | The max duration for role expired. |
| domains | [string](#string) | repeated | The workspace domain, e.g., bytebase.com. |
| enforce_identity_domain | [bool](#bool) |  | Only user and group from the domains can be created and login. |
| database_change_mode | [DatabaseChangeMode](#bytebase-v1-DatabaseChangeMode) |  | The workspace database change mode. |
| disallow_password_signin | [bool](#bool) |  | Whether to disallow password signin. (Except workspace admins) |
| enable_metric_collection | [bool](#bool) |  | Whether to enable metric collection for the workspace. |
| inactive_session_timeout | [google.protobuf.Duration](#google-protobuf-Duration) |  | The session expiration time if not activity detected for the user. Value &lt;= 0 means no limit. |
| enable_audit_log_stdout | [bool](#bool) |  | Whether to enable audit logging to stdout in structured JSON format. Requires TEAM or ENTERPRISE license. |
| watermark | [bool](#bool) |  | Whether to display watermark on pages. Requires ENTERPRISE license. |
| directory_sync_token | [string](#string) |  | The token for directory sync authentication. |
| branding_logo | [string](#string) |  | The branding logo as a data URI (e.g. data:image/png;base64,...). |
| password_restriction | [WorkspaceProfileSetting.PasswordRestriction](#bytebase-v1-WorkspaceProfileSetting-PasswordRestriction) |  | Password restriction settings. |






<a name="bytebase-v1-WorkspaceProfileSetting-PasswordRestriction"></a>

### WorkspaceProfileSetting.PasswordRestriction



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| min_length | [int32](#int32) |  | min_length is the minimum length for password, should no less than 8. |
| require_number | [bool](#bool) |  | require_number requires the password must contains at least one number. |
| require_letter | [bool](#bool) |  | require_letter requires the password must contains at least one letter, regardless of upper case or lower case |
| require_uppercase_letter | [bool](#bool) |  | require_uppercase_letter requires the password must contains at least one upper case letter. |
| require_special_character | [bool](#bool) |  | require_special_character requires the password must contains at least one special character. |
| require_reset_password_for_first_login | [bool](#bool) |  | require_reset_password_for_first_login requires users to reset their password after the 1st login. |
| password_rotation | [google.protobuf.Duration](#google-protobuf-Duration) |  | password_rotation requires users to reset their password after the duration. |





 


<a name="bytebase-v1-AISetting-Provider"></a>

### AISetting.Provider


| Name | Number | Description |
| ---- | ------ | ----------- |
| PROVIDER_UNSPECIFIED | 0 |  |
| OPEN_AI | 1 |  |
| CLAUDE | 2 |  |
| GEMINI | 3 |  |
| AZURE_OPENAI | 4 |  |



<a name="bytebase-v1-Algorithm-InnerOuterMask-MaskType"></a>

### Algorithm.InnerOuterMask.MaskType


| Name | Number | Description |
| ---- | ------ | ----------- |
| MASK_TYPE_UNSPECIFIED | 0 |  |
| INNER | 1 |  |
| OUTER | 2 |  |



<a name="bytebase-v1-Announcement-AlertLevel"></a>

### Announcement.AlertLevel
We support three levels of AlertLevel: INFO, WARNING, and ERROR.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ALERT_LEVEL_UNSPECIFIED | 0 |  |
| INFO | 1 |  |
| WARNING | 2 |  |
| CRITICAL | 3 |  |



<a name="bytebase-v1-DatabaseChangeMode"></a>

### DatabaseChangeMode


| Name | Number | Description |
| ---- | ------ | ----------- |
| DATABASE_CHANGE_MODE_UNSPECIFIED | 0 |  |
| PIPELINE | 1 | A more advanced database change process, including custom approval workflows and other advanced features. Default to this mode. |
| EDITOR | 2 | A simple database change process in SQL editor. Users can execute SQL directly. |



<a name="bytebase-v1-Setting-SettingName"></a>

### Setting.SettingName


| Name | Number | Description |
| ---- | ------ | ----------- |
| SETTING_NAME_UNSPECIFIED | 0 |  |
| WORKSPACE_PROFILE | 1 |  |
| WORKSPACE_APPROVAL | 2 |  |
| APP_IM | 3 |  |
| AI | 4 |  |
| DATA_CLASSIFICATION | 5 |  |
| SEMANTIC_TYPES | 6 |  |
| ENVIRONMENT | 7 |  |



<a name="bytebase-v1-WorkspaceApprovalSetting-Rule-Source"></a>

### WorkspaceApprovalSetting.Rule.Source


| Name | Number | Description |
| ---- | ------ | ----------- |
| SOURCE_UNSPECIFIED | 0 |  |
| CHANGE_DATABASE | 1 |  |
| CREATE_DATABASE | 2 |  |
| EXPORT_DATA | 3 |  |
| REQUEST_ROLE | 4 |  |


 

 


<a name="bytebase-v1-SettingService"></a>

### SettingService
SettingService manages workspace-level settings and configurations.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListSettings | [ListSettingsRequest](#bytebase-v1-ListSettingsRequest) | [ListSettingsResponse](#bytebase-v1-ListSettingsResponse) | Lists all workspace settings. Permissions required: bb.settings.list |
| GetSetting | [GetSettingRequest](#bytebase-v1-GetSettingRequest) | [Setting](#bytebase-v1-Setting) | Retrieves a workspace setting by name. Permissions required: bb.settings.get |
| UpdateSetting | [UpdateSettingRequest](#bytebase-v1-UpdateSettingRequest) | [Setting](#bytebase-v1-Setting) | Updates a workspace setting. Permissions required: bb.settings.set |

 



<a name="v1_user_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/user_service.proto



<a name="bytebase-v1-BatchGetUsersRequest"></a>

### BatchGetUsersRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| names | [string](#string) | repeated | The user names to retrieve. Format: users/{email} |






<a name="bytebase-v1-BatchGetUsersResponse"></a>

### BatchGetUsersResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| users | [User](#bytebase-v1-User) | repeated | The users from the specified request. |






<a name="bytebase-v1-CreateUserRequest"></a>

### CreateUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#bytebase-v1-User) |  | The user to create. |






<a name="bytebase-v1-DeleteUserRequest"></a>

### DeleteUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the user to delete. Format: users/{email} |






<a name="bytebase-v1-GetUserRequest"></a>

### GetUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the user to retrieve. Format: users/{email} |






<a name="bytebase-v1-ListUsersRequest"></a>

### ListUsersRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of users to return. The service may return fewer than this value. If unspecified, at most 10 users will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListUsers` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListUsers` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Show deleted users if specified. |
| filter | [string](#string) |  | Filter is used to filter users returned in the list. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filter: - name: the user name, support &#34;==&#34; and &#34;.matches()&#34; operator. - email: the user email, support &#34;==&#34; and &#34;.matches()&#34; operator. - user_type: the type, check UserType enum for values, support &#34;==&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34; operator. - state: check State enum for values, support &#34;==&#34; operator. - project: the project full name in &#34;projects/{id}&#34; format, support &#34;==&#34; operator.

For example: name == &#34;ed&#34; name.matches(&#34;ed&#34;) email == &#34;ed@bytebase.com&#34; email.matches(&#34;ed&#34;) user_type == &#34;SERVICE_ACCOUNT&#34; user_type in [&#34;SERVICE_ACCOUNT&#34;, &#34;USER&#34;] !(user_type in [&#34;SERVICE_ACCOUNT&#34;, &#34;USER&#34;]) state == &#34;DELETED&#34; project == &#34;projects/sample-project&#34; You can combine filter conditions like: name.matches(&#34;ed&#34;) &amp;&amp; project == &#34;projects/sample-project&#34; (name == &#34;ed&#34; || email == &#34;ed@bytebase.com&#34;) &amp;&amp; project == &#34;projects/sample-project&#34; |






<a name="bytebase-v1-ListUsersResponse"></a>

### ListUsersResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| users | [User](#bytebase-v1-User) | repeated | The users from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-UndeleteUserRequest"></a>

### UndeleteUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the deleted user. Format: users/{email} |






<a name="bytebase-v1-UpdateEmailRequest"></a>

### UpdateEmailRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the user whose email to update. Format: users/{email} Note: This is the current (old) email address. The new email is specified in the &#39;email&#39; field. |
| email | [string](#string) |  | The new email address. |






<a name="bytebase-v1-UpdateUserRequest"></a>

### UpdateUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [User](#bytebase-v1-User) |  | The user to update.

The user&#39;s `name` field is used to identify the user to update. Format: users/{email} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| otp_code | [string](#string) | optional | The otp_code is used to verify the user&#39;s identity by MFA. |
| regenerate_temp_mfa_secret | [bool](#bool) |  | The regenerate_temp_mfa_secret flag means to regenerate temporary MFA secret for user. This is used for MFA setup. The temporary MFA secret and recovery codes will be returned in the response. |
| regenerate_recovery_codes | [bool](#bool) |  | The regenerate_recovery_codes flag means to regenerate recovery codes for user. |
| allow_missing | [bool](#bool) |  | If set to true, and the user is not found, a new user will be created. In this situation, `update_mask` is ignored. |






<a name="bytebase-v1-User"></a>

### User



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the user. Format: users/{email} |
| state | [State](#bytebase-v1-State) |  | The lifecycle state of the user account. |
| email | [string](#string) |  | The email address of the user, used for login and notifications. |
| title | [string](#string) |  | The display title or full name of the user. |
| user_type | [UserType](#bytebase-v1-UserType) |  | The type of user account. |
| password | [string](#string) |  | The password for authentication. Only used during user creation or password updates. |
| service_key | [string](#string) |  | The service key for service account authentication. Only used for service accounts. |
| mfa_enabled | [bool](#bool) |  | The mfa_enabled flag means if the user has enabled MFA. |
| temp_otp_secret | [string](#string) |  | Temporary OTP secret used during MFA setup and regeneration. |
| temp_recovery_codes | [string](#string) | repeated | Temporary recovery codes used during MFA setup and regeneration. |
| temp_otp_secret_created_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Timestamp when temp_otp_secret was created. Used by frontend to show countdown timer. |
| phone | [string](#string) |  | Should be a valid E.164 compliant phone number. Could be empty. |
| profile | [User.Profile](#bytebase-v1-User-Profile) |  | User profile metadata. |
| groups | [string](#string) | repeated | The groups for the user. Format: groups/{email} |
| workload_identity_config | [WorkloadIdentityConfig](#bytebase-v1-WorkloadIdentityConfig) |  | Workload Identity configuration (only for WORKLOAD_IDENTITY type) |






<a name="bytebase-v1-User-Profile"></a>

### User.Profile



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| last_login_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The last time the user successfully logged in. |
| last_change_password_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The last time the user changed their password. |
| source | [string](#string) |  | source means where the user comes from. For now we support Entra ID SCIM sync, so the source could be Entra ID. |






<a name="bytebase-v1-WorkloadIdentityConfig"></a>

### WorkloadIdentityConfig
WorkloadIdentityConfig for API layer


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| provider_type | [WorkloadIdentityConfig.ProviderType](#bytebase-v1-WorkloadIdentityConfig-ProviderType) |  | Platform type (currently only GITHUB is supported) |
| issuer_url | [string](#string) |  | OIDC Issuer URL (auto-filled based on provider_type, can be overridden) |
| allowed_audiences | [string](#string) | repeated | Allowed audiences for token validation |
| subject_pattern | [string](#string) |  | Subject pattern to match (e.g., &#34;repo:owner/repo:ref:refs/heads/main&#34;) |





 


<a name="bytebase-v1-UserType"></a>

### UserType


| Name | Number | Description |
| ---- | ------ | ----------- |
| USER_TYPE_UNSPECIFIED | 0 | Unspecified user type. |
| USER | 1 | Regular human user account. |
| SYSTEM_BOT | 2 | System-managed bot account for automated operations. |
| SERVICE_ACCOUNT | 3 | Service account for API integrations. |
| WORKLOAD_IDENTITY | 4 | External CI/CD workload identity. |



<a name="bytebase-v1-WorkloadIdentityConfig-ProviderType"></a>

### WorkloadIdentityConfig.ProviderType
ProviderType identifies the CI/CD platform.

| Name | Number | Description |
| ---- | ------ | ----------- |
| PROVIDER_TYPE_UNSPECIFIED | 0 |  |
| GITHUB | 1 |  |
| GITLAB | 2 |  |


 

 


<a name="bytebase-v1-UserService"></a>

### UserService
UserService manages user accounts and authentication.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetUser | [GetUserRequest](#bytebase-v1-GetUserRequest) | [User](#bytebase-v1-User) | Get the user. Any authenticated user can get the user. Permissions required: bb.users.get |
| BatchGetUsers | [BatchGetUsersRequest](#bytebase-v1-BatchGetUsersRequest) | [BatchGetUsersResponse](#bytebase-v1-BatchGetUsersResponse) | Get the users in batch. Any authenticated user can batch get users. Permissions required: bb.users.get |
| GetCurrentUser | [.google.protobuf.Empty](#google-protobuf-Empty) | [User](#bytebase-v1-User) | Get the current authenticated user. Permissions required: None |
| ListUsers | [ListUsersRequest](#bytebase-v1-ListUsersRequest) | [ListUsersResponse](#bytebase-v1-ListUsersResponse) | List all users. Any authenticated user can list users. Permissions required: bb.users.list |
| CreateUser | [CreateUserRequest](#bytebase-v1-CreateUserRequest) | [User](#bytebase-v1-User) | Creates a user. When Disallow Signup is enabled, requires bb.users.create permission; otherwise any user can sign up. Permissions required: bb.users.create (only when Disallow Signup is enabled) |
| UpdateUser | [UpdateUserRequest](#bytebase-v1-UpdateUserRequest) | [User](#bytebase-v1-User) | Updates a user. Users can update their own profile, or users with bb.users.update permission can update any user. Note: Email updates are not supported through this API. Use UpdateEmail instead. Permissions required: bb.users.update (or self) |
| DeleteUser | [DeleteUserRequest](#bytebase-v1-DeleteUserRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Deletes a user. Requires bb.users.delete permission with additional validation: the last remaining workspace admin cannot be deleted. Permissions required: bb.users.delete |
| UndeleteUser | [UndeleteUserRequest](#bytebase-v1-UndeleteUserRequest) | [User](#bytebase-v1-User) | Restores a deleted user. Permissions required: bb.users.undelete |
| UpdateEmail | [UpdateEmailRequest](#bytebase-v1-UpdateEmailRequest) | [User](#bytebase-v1-User) | Updates a user&#39;s email address. Permissions required: bb.users.updateEmail |

 



<a name="v1_actuator_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/actuator_service.proto



<a name="bytebase-v1-ActuatorInfo"></a>

### ActuatorInfo
System information and configuration for the Bytebase instance.
Actuator concept is similar to the Spring Boot Actuator.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [string](#string) |  | The Bytebase server version. |
| git_commit | [string](#string) |  | The git commit hash of the build. |
| readonly | [bool](#bool) |  | Whether the Bytebase instance is running in read-only mode. |
| saas | [bool](#bool) |  | Whether the Bytebase instance is running in SaaS mode where some features cannot be edited by users. |
| demo | [bool](#bool) |  | Whether the Bytebase instance is running in demo mode. |
| host | [string](#string) |  | The host address of the Bytebase instance. |
| port | [string](#string) |  | The port number of the Bytebase instance. |
| external_url | [string](#string) |  | The external URL where users or webhook callbacks access Bytebase. |
| need_admin_setup | [bool](#bool) |  | Whether the Bytebase instance requires initial admin setup. |
| disallow_signup | [bool](#bool) |  | Whether self-service user signup is disabled. |
| last_active_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The last time any API call was made, refreshed on each request. |
| require_2fa | [bool](#bool) |  | Whether two-factor authentication is required for all users. |
| workspace_id | [string](#string) |  | The unique identifier for the workspace. |
| debug | [bool](#bool) |  | Whether debug mode is enabled. |
| unlicensed_features | [string](#string) | repeated | List of features that are not licensed. |
| disallow_password_signin | [bool](#bool) |  | Whether password-based signin is disabled (except for workspace admins). |
| password_restriction | [WorkspaceProfileSetting.PasswordRestriction](#bytebase-v1-WorkspaceProfileSetting-PasswordRestriction) |  | Password complexity and restriction requirements. |
| docker | [bool](#bool) |  | Whether the Bytebase instance is running in Docker. |
| user_stats | [ActuatorInfo.StatUser](#bytebase-v1-ActuatorInfo-StatUser) | repeated | Statistics about users in the system. |
| activated_instance_count | [int32](#int32) |  | The number of activated database instances. |
| total_instance_count | [int32](#int32) |  | The total number of database instances. |
| enable_sample | [bool](#bool) |  | Whether sample data setup is enabled. |
| external_url_from_flag | [bool](#bool) |  | Whether the external URL is set via command-line flag (and thus cannot be changed via UI). |






<a name="bytebase-v1-ActuatorInfo-StatUser"></a>

### ActuatorInfo.StatUser
User statistics by type and state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user_type | [UserType](#bytebase-v1-UserType) |  | The type of user. |
| state | [State](#bytebase-v1-State) |  | The state of the user. |
| count | [int32](#int32) |  | The count of users matching this type and state. |






<a name="bytebase-v1-DeleteCacheRequest"></a>

### DeleteCacheRequest
Request message for deleting cache.






<a name="bytebase-v1-GetActuatorInfoRequest"></a>

### GetActuatorInfoRequest
Request message for getting actuator information.






<a name="bytebase-v1-GetResourcePackageRequest"></a>

### GetResourcePackageRequest
Request message for getting branding resources.






<a name="bytebase-v1-ResourcePackage"></a>

### ResourcePackage
Custom branding resources for the Bytebase instance.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| logo | [bytes](#bytes) |  | The branding logo. |






<a name="bytebase-v1-SetupSampleRequest"></a>

### SetupSampleRequest
Request message for setting up sample data.






<a name="bytebase-v1-UpdateActuatorInfoRequest"></a>

### UpdateActuatorInfoRequest
Request message for updating actuator information.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| actuator | [ActuatorInfo](#bytebase-v1-ActuatorInfo) |  | The actuator to update. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the actuator is not found, a new actuator will be created. In this situation, `update_mask` is ignored. |





 

 

 


<a name="bytebase-v1-ActuatorService"></a>

### ActuatorService
ActuatorService manages system health and operational information.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetActuatorInfo | [GetActuatorInfoRequest](#bytebase-v1-GetActuatorInfoRequest) | [ActuatorInfo](#bytebase-v1-ActuatorInfo) | Gets system information and health status of the Bytebase instance. Permissions required: None |
| UpdateActuatorInfo | [UpdateActuatorInfoRequest](#bytebase-v1-UpdateActuatorInfoRequest) | [ActuatorInfo](#bytebase-v1-ActuatorInfo) | Updates system configuration settings for the Bytebase instance. Permissions required: bb.settings.set |
| SetupSample | [SetupSampleRequest](#bytebase-v1-SetupSampleRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Sets up sample data for demonstration and testing purposes. Permissions required: bb.projects.create |
| DeleteCache | [DeleteCacheRequest](#bytebase-v1-DeleteCacheRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Clears the system cache to force data refresh. Permissions required: None |
| GetResourcePackage | [GetResourcePackageRequest](#bytebase-v1-GetResourcePackageRequest) | [ResourcePackage](#bytebase-v1-ResourcePackage) | Gets custom branding resources such as logos. Permissions required: None |

 



<a name="v1_iam_policy-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/iam_policy.proto



<a name="bytebase-v1-Binding"></a>

### Binding
Binding associates members with a role and optional conditions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [string](#string) |  | The role that is assigned to the members. Format: roles/{role} |
| members | [string](#string) | repeated | Specifies the principals requesting access for a Bytebase resource. For users, the member should be: user:{email} For groups, the member should be: group:{email} |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this binding, only used in the project IAM policy. If the condition evaluates to true, then this binding applies to the current request. If the condition evaluates to false, then this binding does not apply to the current request. However, a different role binding might grant the same role to one or more of the principals in this binding. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Support variables: resource.database: the database full name in &#34;instances/{instance}/databases/{database}&#34; format, used by the &#34;roles/sqlEditorUser&#34; role, support &#34;==&#34; operator. resource.schema_name: the schema name, used by the &#34;roles/sqlEditorUser&#34; role, support &#34;==&#34; operator. resource.table_name: the table name, used by the &#34;roles/sqlEditorUser&#34; role, support &#34;==&#34; operator. request.time: the expiration. Only support &#34;&lt;&#34; operation in `request.time &lt; timestamp(&#34;{ISO datetime string format}&#34;)`.

For example: resource.database == &#34;instances/local-pg/databases/postgres&#34; &amp;&amp; resource.schema_name in [&#34;public&#34;,&#34;another_schema&#34;] resource.database == &#34;instances/local-pg/databases/bytebase&#34; &amp;&amp; resource.schema_name == &#34;public&#34; &amp;&amp; resource.table_name in [&#34;audit_log&#34;] request.time &lt; timestamp(&#34;2025-04-26T11:24:48.655Z&#34;) |
| parsed_expr | [google.api.expr.v1alpha1.Expr](#google-api-expr-v1alpha1-Expr) |  | The parsed expression of the condition. |






<a name="bytebase-v1-BindingDelta"></a>

### BindingDelta
A single change to a binding.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| action | [BindingDelta.Action](#bytebase-v1-BindingDelta-Action) |  | The action that was performed on a Binding. |
| role | [string](#string) |  | Role that is assigned to `members`. For example, `roles/projectOwner`. |
| member | [string](#string) |  | Follows the same format of Binding.members. |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this binding. |






<a name="bytebase-v1-GetIamPolicyRequest"></a>

### GetIamPolicyRequest
Request message for getting an IAM policy.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource | [string](#string) |  | The name of the resource to get the IAM policy. Format: projects/{project} Format: workspaces/{workspace} |






<a name="bytebase-v1-IamPolicy"></a>

### IamPolicy
IAM policy that binds members to roles.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| bindings | [Binding](#bytebase-v1-Binding) | repeated | Collection of binding. A binding binds one or more project members to a single project role. |
| etag | [string](#string) |  | The current etag of the policy. If an etag is provided and does not match the current etag of the policy, the call will be blocked and an ABORTED error will be returned. |






<a name="bytebase-v1-PolicyDelta"></a>

### PolicyDelta
Describes changes between two IAM policies.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| binding_deltas | [BindingDelta](#bytebase-v1-BindingDelta) | repeated | The delta for Bindings between two policies. |






<a name="bytebase-v1-SetIamPolicyRequest"></a>

### SetIamPolicyRequest
Request message for setting an IAM policy.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource | [string](#string) |  | The name of the resource to set the IAM policy. Format: projects/{project} Format: workspaces/{workspace} |
| policy | [IamPolicy](#bytebase-v1-IamPolicy) |  |  |
| etag | [string](#string) |  | The current etag of the policy. |





 


<a name="bytebase-v1-BindingDelta-Action"></a>

### BindingDelta.Action
Type of action performed on a binding.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ACTION_UNSPECIFIED | 0 | Unspecified action. |
| ADD | 1 | Add a binding. |
| REMOVE | 2 | Remove a binding. |


 

 

 



<a name="v1_audit_log_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/audit_log_service.proto



<a name="bytebase-v1-AuditData"></a>

### AuditData
Additional audit data specific to certain operations.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy_delta | [PolicyDelta](#bytebase-v1-PolicyDelta) |  | Changes to IAM policies. |






<a name="bytebase-v1-AuditLog"></a>

### AuditLog
Audit log entry recording system activity or API call.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the log. Formats: - projects/{project}/auditLogs/{uid} - workspaces/{workspace}/auditLogs/{uid} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The timestamp when the audit log was created. |
| user | [string](#string) |  | The user who performed the action. Format: users/{email} |
| method | [string](#string) |  | The method or action being audited. For example: /bytebase.v1.SQLService/Query or bb.project.repository.push |
| severity | [AuditLog.Severity](#bytebase-v1-AuditLog-Severity) |  | The severity level of this audit log entry. |
| resource | [string](#string) |  | The resource associated with this audit log. |
| request | [string](#string) |  | The request payload in JSON format. |
| response | [string](#string) |  | The response payload in JSON format. Some fields may be omitted if they are too large or contain sensitive information. |
| status | [google.rpc.Status](#google-rpc-Status) |  | The status of the operation. |
| latency | [google.protobuf.Duration](#google-protobuf-Duration) |  | The duration of the operation. |
| service_data | [google.protobuf.Any](#google-protobuf-Any) |  | Service-specific metadata about the request, response, and activities. |
| request_metadata | [RequestMetadata](#bytebase-v1-RequestMetadata) |  | Metadata about the request context. |






<a name="bytebase-v1-ExportAuditLogsRequest"></a>

### ExportAuditLogsRequest
Request message for exporting audit logs.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  |  |
| filter | [string](#string) |  | The filter of the log. It should be a valid CEL expression. Check the filter field in the SearchAuditLogsRequest message. |
| order_by | [string](#string) |  | The order by of the log. Only support order by create_time. The default sorting order is ascending. For example: - order_by = &#34;create_time asc&#34; - order_by = &#34;create_time desc&#34; |
| format | [ExportFormat](#bytebase-v1-ExportFormat) |  | The export format. |
| page_size | [int32](#int32) |  | The maximum number of logs to return. The service may return fewer than this value. If unspecified, at most 10 log entries will be returned. The maximum value is 5000; values above 5000 will be coerced to 5000. |
| page_token | [string](#string) |  | A page token, received from a previous `ExportAuditLogs` call. Provide this to retrieve the subsequent page. |






<a name="bytebase-v1-ExportAuditLogsResponse"></a>

### ExportAuditLogsResponse
Response message for exporting audit logs.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| content | [bytes](#bytes) |  | The exported audit log content in the requested format. |
| next_page_token | [string](#string) |  | A token to retrieve next page of log entities. Pass this value in the page_token field in the subsequent call to retrieve the next page of log entities. |






<a name="bytebase-v1-RequestMetadata"></a>

### RequestMetadata
Metadata about the incoming request.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| caller_ip | [string](#string) |  | The IP address of the request originator. |
| caller_supplied_user_agent | [string](#string) |  | The user agent string provided by the caller. This is supplied by the client and is not authenticated. |






<a name="bytebase-v1-SearchAuditLogsRequest"></a>

### SearchAuditLogsRequest
Request message for searching audit logs.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  |  |
| filter | [string](#string) |  | The filter of the log. It should be a valid CEL expression. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filter: - method: the API name, can be found in the docs, should start with &#34;/bytebase.v1.&#34; prefix. For example &#34;/bytebase.v1.UserService/CreateUser&#34;. Support &#34;==&#34; operator. - severity: support &#34;==&#34; operator, check Severity enum in AuditLog message for values. - user: the actor, should in &#34;users/{email}&#34; format, support &#34;==&#34; operator. - create_time: support &#34;&gt;=&#34; and &#34;&lt;=&#34; operator.

For example: - filter = &#34;method == &#39;/bytebase.v1.SQLService/Query&#39;&#34; - filter = &#34;method == &#39;/bytebase.v1.SQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39;&#34; - filter = &#34;method == &#39;/bytebase.v1.SQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39; &amp;&amp; user == &#39;users/bb@bytebase.com&#39;&#34; - filter = &#34;method == &#39;/bytebase.v1.SQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39; &amp;&amp; create_time &lt;= &#39;2021-01-01T00:00:00Z&#39; &amp;&amp; create_time &gt;= &#39;2020-01-01T00:00:00Z&#39;&#34; |
| order_by | [string](#string) |  | The order by of the log. Only support order by create_time. The default sorting order is ascending. For example: - order_by = &#34;create_time asc&#34; - order_by = &#34;create_time desc&#34; |
| page_size | [int32](#int32) |  | The maximum number of logs to return. The service may return fewer than this value. If unspecified, at most 10 log entries will be returned. The maximum value is 5000; values above 5000 will be coerced to 5000. |
| page_token | [string](#string) |  | A page token, received from a previous `SearchLogs` call. Provide this to retrieve the subsequent page. |






<a name="bytebase-v1-SearchAuditLogsResponse"></a>

### SearchAuditLogsResponse
Response message for searching audit logs.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| audit_logs | [AuditLog](#bytebase-v1-AuditLog) | repeated |  |
| next_page_token | [string](#string) |  | A token to retrieve next page of log entities. Pass this value in the page_token field in the subsequent call to retrieve the next page of log entities. |





 


<a name="bytebase-v1-AuditLog-Severity"></a>

### AuditLog.Severity
Severity level for audit log entries.

| Name | Number | Description |
| ---- | ------ | ----------- |
| SEVERITY_UNSPECIFIED | 0 | Unspecified severity level. |
| DEBUG | 1 | Debug-level information. |
| INFO | 2 | Informational messages. |
| NOTICE | 3 | Notable events. |
| WARNING | 4 | Warning conditions. |
| ERROR | 5 | Error conditions. |
| CRITICAL | 6 | Critical conditions. |
| ALERT | 7 | Action must be taken immediately. |
| EMERGENCY | 8 | System is unusable. |


 

 


<a name="bytebase-v1-AuditLogService"></a>

### AuditLogService
AuditLogService manages audit logs for system activities and API calls.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| SearchAuditLogs | [SearchAuditLogsRequest](#bytebase-v1-SearchAuditLogsRequest) | [SearchAuditLogsResponse](#bytebase-v1-SearchAuditLogsResponse) | Searches audit logs with optional filtering and pagination. Permissions required: bb.auditLogs.search |
| ExportAuditLogs | [ExportAuditLogsRequest](#bytebase-v1-ExportAuditLogsRequest) | [ExportAuditLogsResponse](#bytebase-v1-ExportAuditLogsResponse) | Exports audit logs in a specified format for external analysis. Permissions required: bb.auditLogs.export |

 



<a name="v1_auth_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/auth_service.proto



<a name="bytebase-v1-ExchangeTokenRequest"></a>

### ExchangeTokenRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  | External OIDC token (JWT) from CI/CD platform. |
| email | [string](#string) |  | Workload Identity email for identifying which identity to authenticate as. Format: {name}@workload.bytebase.com |






<a name="bytebase-v1-ExchangeTokenResponse"></a>

### ExchangeTokenResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| access_token | [string](#string) |  | Bytebase access token. |






<a name="bytebase-v1-IdentityProviderContext"></a>

### IdentityProviderContext
Context for identity provider authentication.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| oauth2_context | [OAuth2IdentityProviderContext](#bytebase-v1-OAuth2IdentityProviderContext) |  | OAuth2 authentication context. |
| oidc_context | [OIDCIdentityProviderContext](#bytebase-v1-OIDCIdentityProviderContext) |  | OpenID Connect authentication context. |






<a name="bytebase-v1-LoginRequest"></a>

### LoginRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| email | [string](#string) |  | User&#39;s email address. |
| password | [string](#string) |  | User&#39;s password for authentication. |
| web | [bool](#bool) |  | If true, sets access token and refresh token as HTTP-only cookies instead of returning the token in the response body. Use for browser-based clients. |
| idp_name | [string](#string) |  | The name of the identity provider. Format: idps/{idp} |
| idp_context | [IdentityProviderContext](#bytebase-v1-IdentityProviderContext) |  | The idp_context is used to get the user information from identity provider. |
| otp_code | [string](#string) | optional | The otp_code is used to verify the user&#39;s identity by MFA. |
| recovery_code | [string](#string) | optional | The recovery_code is used to recovery the user&#39;s identity with MFA. |
| mfa_temp_token | [string](#string) | optional | The mfa_temp_token is used to verify the user&#39;s identity by MFA. |






<a name="bytebase-v1-LoginResponse"></a>

### LoginResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  | Access token for authenticated requests. Only returned when web=false. For web=true, the token is set as an HTTP-only cookie. |
| mfa_temp_token | [string](#string) | optional | Temporary token for MFA verification. |
| require_reset_password | [bool](#bool) |  | Whether user must reset password before continuing. |
| user | [User](#bytebase-v1-User) |  | The user from the successful login. |






<a name="bytebase-v1-LogoutRequest"></a>

### LogoutRequest
Request to logout current user session.






<a name="bytebase-v1-OAuth2IdentityProviderContext"></a>

### OAuth2IdentityProviderContext
OAuth2 authentication context.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [string](#string) |  | Authorization code from OAuth2 provider. |






<a name="bytebase-v1-OIDCIdentityProviderContext"></a>

### OIDCIdentityProviderContext
OpenID Connect authentication context.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [string](#string) |  | Authorization code from OIDC provider. |






<a name="bytebase-v1-RefreshRequest"></a>

### RefreshRequest
Request to refresh the access token.






<a name="bytebase-v1-RefreshResponse"></a>

### RefreshResponse
Response from refreshing the access token.





 

 

 


<a name="bytebase-v1-AuthService"></a>

### AuthService
AuthService handles user authentication operations.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Login | [LoginRequest](#bytebase-v1-LoginRequest) | [LoginResponse](#bytebase-v1-LoginResponse) | Authenticates a user and returns access tokens. Permissions required: None |
| Logout | [LogoutRequest](#bytebase-v1-LogoutRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Logs out the current user session. Permissions required: None |
| ExchangeToken | [ExchangeTokenRequest](#bytebase-v1-ExchangeTokenRequest) | [ExchangeTokenResponse](#bytebase-v1-ExchangeTokenResponse) | Exchanges an external OIDC token for a Bytebase access token. Used by CI/CD pipelines with Workload Identity Federation. Permissions required: None (validates via OIDC token) |
| Refresh | [RefreshRequest](#bytebase-v1-RefreshRequest) | [RefreshResponse](#bytebase-v1-RefreshResponse) | Refreshes the access token using the refresh token cookie. Permissions required: None (validates via refresh token cookie) |

 



<a name="v1_cel_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/cel_service.proto



<a name="bytebase-v1-BatchDeparseRequest"></a>

### BatchDeparseRequest
Request message for batch deparsing CEL expressions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expressions | [google.api.expr.v1alpha1.Expr](#google-api-expr-v1alpha1-Expr) | repeated | The CEL expression ASTs to deparse. |






<a name="bytebase-v1-BatchDeparseResponse"></a>

### BatchDeparseResponse
Response message for batch deparsing CEL expressions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expressions | [string](#string) | repeated | The deparsed CEL expressions as strings. |






<a name="bytebase-v1-BatchParseRequest"></a>

### BatchParseRequest
Request message for batch parsing CEL expressions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expressions | [string](#string) | repeated | The CEL expression strings to parse. |






<a name="bytebase-v1-BatchParseResponse"></a>

### BatchParseResponse
Response message for batch parsing CEL expressions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expressions | [google.api.expr.v1alpha1.Expr](#google-api-expr-v1alpha1-Expr) | repeated | The parsed CEL expressions as AST. |





 

 

 


<a name="bytebase-v1-CelService"></a>

### CelService
CelService manages CEL (Common Expression Language) parsing and formatting operations.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| BatchParse | [BatchParseRequest](#bytebase-v1-BatchParseRequest) | [BatchParseResponse](#bytebase-v1-BatchParseResponse) | Parses multiple CEL expression strings into AST representations. Permissions required: None |
| BatchDeparse | [BatchDeparseRequest](#bytebase-v1-BatchDeparseRequest) | [BatchDeparseResponse](#bytebase-v1-BatchDeparseResponse) | Converts multiple CEL AST representations back into expression strings. Permissions required: None |

 



<a name="v1_database_catalog_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/database_catalog_service.proto



<a name="bytebase-v1-ColumnCatalog"></a>

### ColumnCatalog
Column metadata within a table.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The column name. |
| semantic_type | [string](#string) |  | The semantic type describing the data purpose. |
| labels | [ColumnCatalog.LabelsEntry](#bytebase-v1-ColumnCatalog-LabelsEntry) | repeated | User-defined labels for this column. |
| classification | [string](#string) |  | The data classification level for this column. |
| object_schema | [ObjectSchema](#bytebase-v1-ObjectSchema) | optional | Object schema for complex column types like JSON. |






<a name="bytebase-v1-ColumnCatalog-LabelsEntry"></a>

### ColumnCatalog.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-v1-DatabaseCatalog"></a>

### DatabaseCatalog
Catalog metadata for a database including schemas, tables, and columns.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database catalog. Format: instances/{instance}/databases/{database}/catalog |
| schemas | [SchemaCatalog](#bytebase-v1-SchemaCatalog) | repeated | The schemas in the database. |






<a name="bytebase-v1-GetDatabaseCatalogRequest"></a>

### GetDatabaseCatalogRequest
Request message for getting a database catalog.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database catalog to retrieve. Format: instances/{instance}/databases/{database}/catalog |






<a name="bytebase-v1-ObjectSchema"></a>

### ObjectSchema
Schema definition for object-type columns.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [ObjectSchema.Type](#bytebase-v1-ObjectSchema-Type) |  | The data type of this object. |
| struct_kind | [ObjectSchema.StructKind](#bytebase-v1-ObjectSchema-StructKind) |  | Struct schema. |
| array_kind | [ObjectSchema.ArrayKind](#bytebase-v1-ObjectSchema-ArrayKind) |  | Array schema. |
| semantic_type | [string](#string) |  | The semantic type of this object. |






<a name="bytebase-v1-ObjectSchema-ArrayKind"></a>

### ObjectSchema.ArrayKind
Array type with element schema.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| kind | [ObjectSchema](#bytebase-v1-ObjectSchema) |  | The schema of array elements. |






<a name="bytebase-v1-ObjectSchema-StructKind"></a>

### ObjectSchema.StructKind
Structure type with named properties.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| properties | [ObjectSchema.StructKind.PropertiesEntry](#bytebase-v1-ObjectSchema-StructKind-PropertiesEntry) | repeated | Properties of the struct. |






<a name="bytebase-v1-ObjectSchema-StructKind-PropertiesEntry"></a>

### ObjectSchema.StructKind.PropertiesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [ObjectSchema](#bytebase-v1-ObjectSchema) |  |  |






<a name="bytebase-v1-SchemaCatalog"></a>

### SchemaCatalog
Schema metadata within a database.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The schema name. |
| tables | [TableCatalog](#bytebase-v1-TableCatalog) | repeated | The tables in the schema. |






<a name="bytebase-v1-TableCatalog"></a>

### TableCatalog
Table metadata within a schema.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The table name. |
| columns | [TableCatalog.Columns](#bytebase-v1-TableCatalog-Columns) |  | Regular table columns. |
| object_schema | [ObjectSchema](#bytebase-v1-ObjectSchema) |  | Object schema for JSON/XML columns. |
| classification | [string](#string) |  | The data classification level for this table. |






<a name="bytebase-v1-TableCatalog-Columns"></a>

### TableCatalog.Columns
Column list for regular tables.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| columns | [ColumnCatalog](#bytebase-v1-ColumnCatalog) | repeated | The columns in the table. |






<a name="bytebase-v1-UpdateDatabaseCatalogRequest"></a>

### UpdateDatabaseCatalogRequest
Request message for updating a database catalog.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| catalog | [DatabaseCatalog](#bytebase-v1-DatabaseCatalog) |  | The database catalog to update.

The catalog&#39;s `name` field is used to identify the database catalog to update. Format: instances/{instance}/databases/{database}/catalog |
| allow_missing | [bool](#bool) |  | If set to true, and the database catalog is not found, a new database catalog will be created. In this situation, `update_mask` is ignored. |





 


<a name="bytebase-v1-ObjectSchema-Type"></a>

### ObjectSchema.Type
Object schema data types.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 | Unspecified type. |
| STRING | 1 | String type. |
| NUMBER | 2 | Number type. |
| BOOLEAN | 3 | Boolean type. |
| OBJECT | 4 | Object/struct type. |
| ARRAY | 5 | Array type. |


 

 


<a name="bytebase-v1-DatabaseCatalogService"></a>

### DatabaseCatalogService
DatabaseCatalogService manages database schema metadata and classification.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetDatabaseCatalog | [GetDatabaseCatalogRequest](#bytebase-v1-GetDatabaseCatalogRequest) | [DatabaseCatalog](#bytebase-v1-DatabaseCatalog) | Gets the catalog metadata for a database. Permissions required: bb.databaseCatalogs.get |
| UpdateDatabaseCatalog | [UpdateDatabaseCatalogRequest](#bytebase-v1-UpdateDatabaseCatalogRequest) | [DatabaseCatalog](#bytebase-v1-DatabaseCatalog) | Updates catalog metadata such as classifications and labels. Permissions required: bb.databaseCatalogs.update |

 



<a name="v1_database_group_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/database_group_service.proto



<a name="bytebase-v1-CreateDatabaseGroupRequest"></a>

### CreateDatabaseGroupRequest
Request message for creating a database group.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this database group will be created. Format: projects/{project} |
| database_group | [DatabaseGroup](#bytebase-v1-DatabaseGroup) |  | The database group to create. |
| database_group_id | [string](#string) |  | The ID to use for the database group, which will become the final component of the database group&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |
| validate_only | [bool](#bool) |  | If set, validate the create request and preview the full database group response, but do not actually create it. |






<a name="bytebase-v1-DatabaseGroup"></a>

### DatabaseGroup
A group of databases matched by expressions.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database group. Format: projects/{project}/databaseGroups/{databaseGroup} |
| title | [string](#string) |  | The short name used in actual databases specified by users. |
| database_expr | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this database group. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Support variables: resource.environment_id: the environment resource id. Support &#34;==&#34;, &#34;!=&#34;, &#34;in [XX]&#34;, &#34;!(in [xx])&#34; operations. resource.instance_id: the instance resource id. Support &#34;==&#34;, &#34;!=&#34;, &#34;in [XX]&#34;, &#34;!(in [xx])&#34;, &#34;contains&#34;, &#34;matches&#34;, &#34;startsWith&#34;, &#34;endsWith&#34; operations. resource.database_name: the database name. Support &#34;==&#34;, &#34;!=&#34;, &#34;in [XX]&#34;, &#34;!(in [xx])&#34;, &#34;contains&#34;, &#34;matches&#34;, &#34;startsWith&#34;, &#34;endsWith&#34; operations. resource.database_labels: the database labels. Support map access operations. All variables should join with &#34;&amp;&amp;&#34; condition.

For example: resource.environment_id == &#34;test&#34; &amp;&amp; resource.database_name.startsWith(&#34;sample_&#34;) resource.database_labels[&#34;tenant&#34;] == &#34;tenant1&#34; |
| matched_databases | [DatabaseGroup.Database](#bytebase-v1-DatabaseGroup-Database) | repeated | The list of databases that match the database group condition. |






<a name="bytebase-v1-DatabaseGroup-Database"></a>

### DatabaseGroup.Database
A database within a database group.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The resource name of the database. Format: instances/{instance}/databases/{database} |






<a name="bytebase-v1-DeleteDatabaseGroupRequest"></a>

### DeleteDatabaseGroupRequest
Request message for deleting a database group.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database group to delete. Format: projects/{project}/databaseGroups/{databaseGroup} |






<a name="bytebase-v1-GetDatabaseGroupRequest"></a>

### GetDatabaseGroupRequest
Request message for getting a database group.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database group to retrieve. Format: projects/{project}/databaseGroups/{databaseGroup} |
| view | [DatabaseGroupView](#bytebase-v1-DatabaseGroupView) |  | The view to return. Defaults to DATABASE_GROUP_VIEW_BASIC. |






<a name="bytebase-v1-ListDatabaseGroupsRequest"></a>

### ListDatabaseGroupsRequest
Request message for listing database groups.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource whose database groups are to be listed. Format: projects/{project} |
| view | [DatabaseGroupView](#bytebase-v1-DatabaseGroupView) |  | The view to return. Defaults to DATABASE_GROUP_VIEW_BASIC. |






<a name="bytebase-v1-ListDatabaseGroupsResponse"></a>

### ListDatabaseGroupsResponse
Response message for listing database groups.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database_groups | [DatabaseGroup](#bytebase-v1-DatabaseGroup) | repeated | The database groups from the specified request. |






<a name="bytebase-v1-UpdateDatabaseGroupRequest"></a>

### UpdateDatabaseGroupRequest
Request message for updating a database group.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database_group | [DatabaseGroup](#bytebase-v1-DatabaseGroup) |  | The database group to update.

The database group&#39;s `name` field is used to identify the database group to update. Format: projects/{project}/databaseGroups/{databaseGroup} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the database group is not found, a new database group will be created. In this situation, `update_mask` is ignored. |





 


<a name="bytebase-v1-DatabaseGroupView"></a>

### DatabaseGroupView
View options for database group responses.

| Name | Number | Description |
| ---- | ------ | ----------- |
| DATABASE_GROUP_VIEW_UNSPECIFIED | 0 | The default / unset value. The API will default to the BASIC view. |
| DATABASE_GROUP_VIEW_BASIC | 1 | Include basic information about the database group, but exclude the list of matched databases and unmatched databases. |
| DATABASE_GROUP_VIEW_FULL | 2 | Include everything. |


 

 


<a name="bytebase-v1-DatabaseGroupService"></a>

### DatabaseGroupService
DatabaseGroupService manages database groups for organizing databases by criteria.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListDatabaseGroups | [ListDatabaseGroupsRequest](#bytebase-v1-ListDatabaseGroupsRequest) | [ListDatabaseGroupsResponse](#bytebase-v1-ListDatabaseGroupsResponse) | Lists database groups in a project. Permissions required: bb.databaseGroups.list |
| GetDatabaseGroup | [GetDatabaseGroupRequest](#bytebase-v1-GetDatabaseGroupRequest) | [DatabaseGroup](#bytebase-v1-DatabaseGroup) | Gets a database group by name. Permissions required: bb.databaseGroups.get |
| CreateDatabaseGroup | [CreateDatabaseGroupRequest](#bytebase-v1-CreateDatabaseGroupRequest) | [DatabaseGroup](#bytebase-v1-DatabaseGroup) | Creates a new database group. Permissions required: bb.databaseGroups.create |
| UpdateDatabaseGroup | [UpdateDatabaseGroupRequest](#bytebase-v1-UpdateDatabaseGroupRequest) | [DatabaseGroup](#bytebase-v1-DatabaseGroup) | Updates a database group. Permissions required: bb.databaseGroups.update When allow_missing=true, also requires: bb.databaseGroups.create |
| DeleteDatabaseGroup | [DeleteDatabaseGroupRequest](#bytebase-v1-DeleteDatabaseGroupRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Deletes a database group. Permissions required: bb.databaseGroups.delete |

 



<a name="v1_instance_role_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/instance_role_service.proto



<a name="bytebase-v1-InstanceRole"></a>

### InstanceRole
InstanceRole is the API message for instance role.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the role. Format: instances/{instance}/roles/{role} The role name is the unique name for the role. |
| role_name | [string](#string) |  | The role name. It&#39;s unique within the instance. |
| password | [string](#string) | optional | The role password. |
| connection_limit | [int32](#int32) | optional | The connection count limit for this role. |
| valid_until | [string](#string) | optional | The expiration for the role&#39;s password. |
| attribute | [string](#string) | optional | The role attribute. For PostgreSQL, it contains super_user, no_inherit, create_role, create_db, can_login, replication, and bypass_rls. Docs: https://www.postgresql.org/docs/current/role-attributes.html For MySQL, it&#39;s the global privileges as GRANT statements, which means it only contains &#34;GRANT ... ON *.* TO ...&#34;. Docs: https://dev.mysql.com/doc/refman/8.0/en/grant.html |






<a name="bytebase-v1-ListInstanceRolesRequest"></a>

### ListInstanceRolesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of roles. Format: instances/{instance} |
| page_size | [int32](#int32) |  | Not used. The maximum number of roles to return. The service may return fewer than this value. If unspecified, at most 10 roles will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListInstanceRoles` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListInstanceRoles` must match the call that provided the page token. |
| refresh | [bool](#bool) |  | Refresh will refresh and return the latest data. |






<a name="bytebase-v1-ListInstanceRolesResponse"></a>

### ListInstanceRolesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| roles | [InstanceRole](#bytebase-v1-InstanceRole) | repeated | The roles from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |





 

 

 


<a name="bytebase-v1-InstanceRoleService"></a>

### InstanceRoleService
InstanceRoleService manages database roles within instances.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListInstanceRoles | [ListInstanceRolesRequest](#bytebase-v1-ListInstanceRolesRequest) | [ListInstanceRolesResponse](#bytebase-v1-ListInstanceRolesResponse) | Lists all database roles in an instance. Permissions required: bb.instanceRoles.list |

 



<a name="v1_instance_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/instance_service.proto



<a name="bytebase-v1-AddDataSourceRequest"></a>

### AddDataSourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the instance to add a data source to. Format: instances/{instance} |
| data_source | [DataSource](#bytebase-v1-DataSource) |  | Identified by data source ID. Only READ_ONLY data source can be added. |
| validate_only | [bool](#bool) |  | Validate only also tests the data source connection. |






<a name="bytebase-v1-BatchSyncInstancesRequest"></a>

### BatchSyncInstancesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| requests | [SyncInstanceRequest](#bytebase-v1-SyncInstanceRequest) | repeated | The request message specifying the instances to sync. A maximum of 1000 instances can be synced in a batch. |






<a name="bytebase-v1-BatchSyncInstancesResponse"></a>

### BatchSyncInstancesResponse







<a name="bytebase-v1-BatchUpdateInstancesRequest"></a>

### BatchUpdateInstancesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| requests | [UpdateInstanceRequest](#bytebase-v1-UpdateInstanceRequest) | repeated | The request message specifying the resources to update. |






<a name="bytebase-v1-BatchUpdateInstancesResponse"></a>

### BatchUpdateInstancesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instances | [Instance](#bytebase-v1-Instance) | repeated |  |






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
| id | [string](#string) |  | The unique identifier for this data source. |
| type | [DataSourceType](#bytebase-v1-DataSourceType) |  | The type of data source (ADMIN or READ_ONLY). |
| username | [string](#string) |  | The username for database authentication. |
| password | [string](#string) |  | The password for database authentication. |
| use_ssl | [bool](#bool) |  | Use SSL to connect to the data source. By default, we use system default SSL configuration. |
| ssl_ca | [string](#string) |  | The SSL certificate authority certificate. |
| ssl_cert | [string](#string) |  | The SSL client certificate. |
| ssl_key | [string](#string) |  | The SSL client private key. |
| verify_tls_certificate | [bool](#bool) |  | verify_tls_certificate enables TLS certificate verification for SSL connections. Default is false (no verification) for backward compatibility. Set to true for secure connections (recommended for production). Only set to false for development or when certificates cannot be properly validated (e.g., self-signed certs, VPN environments). |
| host | [string](#string) |  | The hostname or IP address of the database server. |
| port | [string](#string) |  | The port number of the database server. |
| database | [string](#string) |  | The name of the database to connect to. |
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
| authentication_private_key_passphrase | [string](#string) |  | Passphrase for the encrypted PKCS#8 private key. Only used when the private key is encrypted. |
| external_secret | [DataSourceExternalSecret](#bytebase-v1-DataSourceExternalSecret) |  |  |
| authentication_type | [DataSource.AuthenticationType](#bytebase-v1-DataSource-AuthenticationType) |  |  |
| azure_credential | [DataSource.AzureCredential](#bytebase-v1-DataSource-AzureCredential) |  |  |
| aws_credential | [DataSource.AWSCredential](#bytebase-v1-DataSource-AWSCredential) |  |  |
| gcp_credential | [DataSource.GCPCredential](#bytebase-v1-DataSource-GCPCredential) |  |  |
| sasl_config | [SASLConfig](#bytebase-v1-SASLConfig) |  |  |
| additional_addresses | [DataSource.Address](#bytebase-v1-DataSource-Address) | repeated | additional_addresses is used for MongoDB replica set. |
| direct_connection | [bool](#bool) |  | direct_connection is used for MongoDB to dispatch all the operations to the node specified in the connection string. |
| region | [string](#string) |  | region is the location of where the DB is, works for AWS RDS. For example, us-east-1. |
| warehouse_id | [string](#string) |  | warehouse_id is used by Databricks. |
| master_name | [string](#string) |  | master_name is the master name used by connecting redis-master via redis sentinel. |
| master_username | [string](#string) |  | master_username and master_password are master credentials used by redis sentinel mode. |
| master_password | [string](#string) |  |  |
| redis_type | [DataSource.RedisType](#bytebase-v1-DataSource-RedisType) |  |  |
| cluster | [string](#string) |  | Cluster is the cluster name for the data source. Used by CockroachDB. |
| extra_connection_parameters | [DataSource.ExtraConnectionParametersEntry](#bytebase-v1-DataSource-ExtraConnectionParametersEntry) | repeated | Extra connection parameters for the database connection. For PostgreSQL HA, this can be used to set target_session_attrs=read-write |






<a name="bytebase-v1-DataSource-AWSCredential"></a>

### DataSource.AWSCredential



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| access_key_id | [string](#string) |  |  |
| secret_access_key | [string](#string) |  |  |
| session_token | [string](#string) |  |  |
| role_arn | [string](#string) |  | ARN of IAM role to assume for cross-account access. See: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use.html |
| external_id | [string](#string) |  | Optional external ID for additional security when assuming role. See: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create_for-user_externalid.html |






<a name="bytebase-v1-DataSource-Address"></a>

### DataSource.Address



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| host | [string](#string) |  |  |
| port | [string](#string) |  |  |






<a name="bytebase-v1-DataSource-AzureCredential"></a>

### DataSource.AzureCredential



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| tenant_id | [string](#string) |  |  |
| client_id | [string](#string) |  |  |
| client_secret | [string](#string) |  |  |






<a name="bytebase-v1-DataSource-ExtraConnectionParametersEntry"></a>

### DataSource.ExtraConnectionParametersEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-v1-DataSource-GCPCredential"></a>

### DataSource.GCPCredential



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| content | [string](#string) |  |  |






<a name="bytebase-v1-DataSourceExternalSecret"></a>

### DataSourceExternalSecret



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| secret_type | [DataSourceExternalSecret.SecretType](#bytebase-v1-DataSourceExternalSecret-SecretType) |  | The type of external secret store. |
| url | [string](#string) |  | The URL of the external secret store. |
| auth_type | [DataSourceExternalSecret.AuthType](#bytebase-v1-DataSourceExternalSecret-AuthType) |  | The authentication method for accessing the secret store. |
| app_role | [DataSourceExternalSecret.AppRoleAuthOption](#bytebase-v1-DataSourceExternalSecret-AppRoleAuthOption) |  | AppRole authentication configuration. |
| token | [string](#string) |  | Token for direct authentication. |
| engine_name | [string](#string) |  | engine name is the name for secret engine. |
| secret_name | [string](#string) |  | the secret name in the engine to store the password. |
| password_key_name | [string](#string) |  | the key name for the password. |
| skip_vault_tls_verification | [bool](#bool) |  | TLS configuration for connecting to Vault server. These fields are separate from the database TLS configuration in DataSource. skip_vault_tls_verification disables TLS certificate verification for Vault connections. Default is false (verification enabled) for security. Only set to true for development or when certificates cannot be properly validated. |
| vault_ssl_ca | [string](#string) |  | CA certificate for Vault server verification. |
| vault_ssl_cert | [string](#string) |  | Client certificate for mutual TLS authentication with Vault. |
| vault_ssl_key | [string](#string) |  | Client private key for mutual TLS authentication with Vault. |






<a name="bytebase-v1-DataSourceExternalSecret-AppRoleAuthOption"></a>

### DataSourceExternalSecret.AppRoleAuthOption



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role_id | [string](#string) |  | The role ID for Vault AppRole authentication. |
| secret_id | [string](#string) |  | the secret id for the role without ttl. |
| type | [DataSourceExternalSecret.AppRoleAuthOption.SecretType](#bytebase-v1-DataSourceExternalSecret-AppRoleAuthOption-SecretType) |  | The type of secret for AppRole authentication. |
| mount_path | [string](#string) |  | The path where the approle auth method is mounted. |






<a name="bytebase-v1-DeleteInstanceRequest"></a>

### DeleteInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the instance to delete. Format: instances/{instance} |
| force | [bool](#bool) |  | If set to true, any databases and sheets from this project will also be moved to default project, and all open issues will be closed. |
| purge | [bool](#bool) |  | If set to true, permanently purge the soft-deleted instance and all related resources. This operation is irreversible. Following AIP-165, this should only be used for administrative cleanup of old soft-deleted instances. The instance must already be soft-deleted for this to work. |






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
| state | [State](#bytebase-v1-State) |  | The lifecycle state of the instance. |
| title | [string](#string) |  | The display title of the instance. |
| engine | [Engine](#bytebase-v1-Engine) |  | The database engine type. |
| engine_version | [string](#string) |  | The version of the database engine. |
| external_link | [string](#string) |  | External URL to the database instance console. |
| data_sources | [DataSource](#bytebase-v1-DataSource) | repeated | Data source configurations for connecting to the instance. |
| environment | [string](#string) | optional | The environment resource. Format: environments/prod where prod is the environment resource ID. |
| activation | [bool](#bool) |  | Whether the instance is activated for use. |
| roles | [InstanceRole](#bytebase-v1-InstanceRole) | repeated | Database roles available in this instance. |
| sync_interval | [google.protobuf.Duration](#google-protobuf-Duration) |  | How often the instance is synced. |
| sync_databases | [string](#string) | repeated | Enable sync for following databases. Default empty, means sync all schemas &amp; databases. |
| last_sync_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The last time the instance was synced. |
| labels | [Instance.LabelsEntry](#bytebase-v1-Instance-LabelsEntry) | repeated | Labels are key-value pairs that can be attached to the instance. For example, { &#34;org_group&#34;: &#34;infrastructure&#34;, &#34;environment&#34;: &#34;production&#34; } |






<a name="bytebase-v1-Instance-LabelsEntry"></a>

### Instance.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-v1-InstanceResource"></a>

### InstanceResource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| title | [string](#string) |  | The display title of the instance. |
| engine | [Engine](#bytebase-v1-Engine) |  | The database engine type. |
| engine_version | [string](#string) |  | The version of the database engine. |
| data_sources | [DataSource](#bytebase-v1-DataSource) | repeated | Data source configurations for the instance. |
| activation | [bool](#bool) |  | Whether the instance is activated. |
| name | [string](#string) |  | The name of the instance. Format: instances/{instance} |
| environment | [string](#string) | optional | The environment resource. Format: environments/prod where prod is the environment resource ID. |






<a name="bytebase-v1-KerberosConfig"></a>

### KerberosConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| primary | [string](#string) |  | The primary component of the Kerberos principal. |
| instance | [string](#string) |  | The instance component of the Kerberos principal. |
| realm | [string](#string) |  | The Kerberos realm. |
| keytab | [bytes](#bytes) |  | The keytab file contents for authentication. |
| kdc_host | [string](#string) |  | The hostname of the Key Distribution Center (KDC). |
| kdc_port | [string](#string) |  | The port of the Key Distribution Center (KDC). |
| kdc_transport_protocol | [string](#string) |  | The transport protocol for KDC communication (tcp or udp). |






<a name="bytebase-v1-ListInstanceDatabaseRequest"></a>

### ListInstanceDatabaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the instance. Format: instances/{instance} |
| instance | [Instance](#bytebase-v1-Instance) | optional | The target instance. We need to set this field if the target instance is not created yet. |






<a name="bytebase-v1-ListInstanceDatabaseResponse"></a>

### ListInstanceDatabaseResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [string](#string) | repeated | All database name list in the instance. |






<a name="bytebase-v1-ListInstancesRequest"></a>

### ListInstancesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of instances to return. The service may return fewer than this value. If unspecified, at most 10 instances will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListInstances` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListInstances` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Show deleted instances if specified. |
| filter | [string](#string) |  | Filter the instance. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filters: - name: the instance name, support &#34;==&#34; and &#34;.matches()&#34; operator. - resource_id: the instance id, support &#34;==&#34; and &#34;.matches()&#34; operator. - environment: the environment full name in &#34;environments/{id}&#34; format, support &#34;==&#34; operator. - state: the instance state, check State enum for values, support &#34;==&#34; operator. - engine: the instance engine, check Engine enum for values. Support &#34;==&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34; operator. - host: the instance host, support &#34;==&#34; and &#34;.matches()&#34; operator. - port: the instance port, support &#34;==&#34; and &#34;.matches()&#34; operator. - project: the project full name in &#34;projects/{id}&#34; format, support &#34;==&#34; operator. - labels.{key}: the instance label, support &#34;==&#34; and &#34;in&#34; operators.

For example: name == &#34;sample instance&#34; name.matches(&#34;sample&#34;) resource_id == &#34;sample-instance&#34; resource_id.matches(&#34;sample&#34;) state == &#34;DELETED&#34; environment == &#34;environments/test&#34; environment == &#34;&#34; (find instances which environment is not set) engine == &#34;MYSQL&#34; engine in [&#34;MYSQL&#34;, &#34;POSTGRES&#34;] !(engine in [&#34;MYSQL&#34;, &#34;POSTGRES&#34;]) host == &#34;127.0.0.1&#34; host.matches(&#34;127.0&#34;) port == &#34;54321&#34; port.matches(&#34;543&#34;) labels.org_group == &#34;infrastructure&#34; labels.environment in [&#34;prod&#34;, &#34;production&#34;] project == &#34;projects/sample-project&#34; You can combine filter conditions like: name.matches(&#34;sample&#34;) &amp;&amp; environment == &#34;environments/test&#34; host == &#34;127.0.0.1&#34; &amp;&amp; port == &#34;54321&#34; |
| order_by | [string](#string) |  | The order by of instances. Support title, environment. The default sorting order is ascending. For example: - order_by = &#34;title&#34; - order_by = &#34;title desc&#34; - order_by = &#34;title desc, environment asc&#34; |






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
| name | [string](#string) |  | The name of the instance to remove a data source from. Format: instances/{instance} |
| data_source | [DataSource](#bytebase-v1-DataSource) |  | Identified by data source ID. Only READ_ONLY data source can be removed. |






<a name="bytebase-v1-SASLConfig"></a>

### SASLConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| krb_config | [KerberosConfig](#bytebase-v1-KerberosConfig) |  | Kerberos authentication configuration. |






<a name="bytebase-v1-SyncInstanceRequest"></a>

### SyncInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of instance. Format: instances/{instance} |
| enable_full_sync | [bool](#bool) |  | When full sync is enabled, all databases in the instance will be synchronized. Otherwise, only the instance metadata (such as the database list) and any newly discovered databases will be synced. |






<a name="bytebase-v1-SyncInstanceResponse"></a>

### SyncInstanceResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [string](#string) | repeated | All database name list in the instance. |






<a name="bytebase-v1-UndeleteInstanceRequest"></a>

### UndeleteInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the deleted instance. Format: instances/{instance} |






<a name="bytebase-v1-UpdateDataSourceRequest"></a>

### UpdateDataSourceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the instance to update a data source. Format: instances/{instance} |
| data_source | [DataSource](#bytebase-v1-DataSource) |  | Identified by data source ID. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| validate_only | [bool](#bool) |  | Validate only also tests the data source connection. |
| allow_missing | [bool](#bool) |  | If set to true, and the data source is not found, a new data source will be created. In this situation, `update_mask` is ignored. |






<a name="bytebase-v1-UpdateInstanceRequest"></a>

### UpdateInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| instance | [Instance](#bytebase-v1-Instance) |  | The instance to update.

The instance&#39;s `name` field is used to identify the instance to update. Format: instances/{instance} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the instance is not found, a new instance will be created. In this situation, `update_mask` is ignored. |





 


<a name="bytebase-v1-DataSource-AuthenticationType"></a>

### DataSource.AuthenticationType


| Name | Number | Description |
| ---- | ------ | ----------- |
| AUTHENTICATION_UNSPECIFIED | 0 |  |
| PASSWORD | 1 |  |
| GOOGLE_CLOUD_SQL_IAM | 2 |  |
| AWS_RDS_IAM | 3 |  |
| AZURE_IAM | 4 |  |



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
| SECRET_TYPE_UNSPECIFIED | 0 | Unspecified secret type. |
| PLAIN | 1 | Plain text secret. |
| ENVIRONMENT | 2 | Secret from environment variable. |



<a name="bytebase-v1-DataSourceExternalSecret-AuthType"></a>

### DataSourceExternalSecret.AuthType


| Name | Number | Description |
| ---- | ------ | ----------- |
| AUTH_TYPE_UNSPECIFIED | 0 | Unspecified authentication type. |
| TOKEN | 1 | ref: https://developer.hashicorp.com/vault/docs/auth/token |
| VAULT_APP_ROLE | 2 | ref: https://developer.hashicorp.com/vault/docs/auth/approle |



<a name="bytebase-v1-DataSourceExternalSecret-SecretType"></a>

### DataSourceExternalSecret.SecretType


| Name | Number | Description |
| ---- | ------ | ----------- |
| SECRET_TYPE_UNSPECIFIED | 0 | Unspecified secret type. |
| VAULT_KV_V2 | 1 | ref: https://developer.hashicorp.com/vault/api-docs/secret/kv/kv-v2 |
| AWS_SECRETS_MANAGER | 2 | ref: https://docs.aws.amazon.com/secretsmanager/latest/userguide/intro.html |
| GCP_SECRET_MANAGER | 3 | ref: https://cloud.google.com/secret-manager/docs |
| AZURE_KEY_VAULT | 4 | ref: https://learn.microsoft.com/en-us/azure/key-vault/secrets/about-secrets |



<a name="bytebase-v1-DataSourceType"></a>

### DataSourceType


| Name | Number | Description |
| ---- | ------ | ----------- |
| DATA_SOURCE_UNSPECIFIED | 0 | Unspecified data source type. |
| ADMIN | 1 | Admin data source with write permissions. |
| READ_ONLY | 2 | Read-only data source for query operations. |


 

 


<a name="bytebase-v1-InstanceService"></a>

### InstanceService
InstanceService manages database instances and their connections.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetInstance | [GetInstanceRequest](#bytebase-v1-GetInstanceRequest) | [Instance](#bytebase-v1-Instance) | Gets a database instance by name. Permissions required: bb.instances.get |
| ListInstances | [ListInstancesRequest](#bytebase-v1-ListInstancesRequest) | [ListInstancesResponse](#bytebase-v1-ListInstancesResponse) | Lists all database instances. Permissions required: bb.instances.list |
| CreateInstance | [CreateInstanceRequest](#bytebase-v1-CreateInstanceRequest) | [Instance](#bytebase-v1-Instance) | Creates a new database instance. Permissions required: bb.instances.create |
| UpdateInstance | [UpdateInstanceRequest](#bytebase-v1-UpdateInstanceRequest) | [Instance](#bytebase-v1-Instance) | Updates a database instance. Permissions required: bb.instances.update |
| DeleteInstance | [DeleteInstanceRequest](#bytebase-v1-DeleteInstanceRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Deletes or soft-deletes a database instance. Permissions required: bb.instances.delete |
| UndeleteInstance | [UndeleteInstanceRequest](#bytebase-v1-UndeleteInstanceRequest) | [Instance](#bytebase-v1-Instance) | Restores a soft-deleted database instance. Permissions required: bb.instances.undelete |
| SyncInstance | [SyncInstanceRequest](#bytebase-v1-SyncInstanceRequest) | [SyncInstanceResponse](#bytebase-v1-SyncInstanceResponse) | Syncs database schemas and metadata from an instance. Permissions required: bb.instances.sync |
| ListInstanceDatabase | [ListInstanceDatabaseRequest](#bytebase-v1-ListInstanceDatabaseRequest) | [ListInstanceDatabaseResponse](#bytebase-v1-ListInstanceDatabaseResponse) | Lists all databases within an instance without creating them. Permissions required: bb.instances.get |
| BatchSyncInstances | [BatchSyncInstancesRequest](#bytebase-v1-BatchSyncInstancesRequest) | [BatchSyncInstancesResponse](#bytebase-v1-BatchSyncInstancesResponse) | Syncs multiple instances in a single request. Permissions required: bb.instances.sync |
| BatchUpdateInstances | [BatchUpdateInstancesRequest](#bytebase-v1-BatchUpdateInstancesRequest) | [BatchUpdateInstancesResponse](#bytebase-v1-BatchUpdateInstancesResponse) | Updates multiple instances in a single request. Permissions required: bb.instances.update |
| AddDataSource | [AddDataSourceRequest](#bytebase-v1-AddDataSourceRequest) | [Instance](#bytebase-v1-Instance) | Adds a read-only data source to an instance. Permissions required: bb.instances.update |
| RemoveDataSource | [RemoveDataSourceRequest](#bytebase-v1-RemoveDataSourceRequest) | [Instance](#bytebase-v1-Instance) | Removes a read-only data source from an instance. Permissions required: bb.instances.update |
| UpdateDataSource | [UpdateDataSourceRequest](#bytebase-v1-UpdateDataSourceRequest) | [Instance](#bytebase-v1-Instance) | Updates a data source configuration. Permissions required: bb.instances.update |

 



<a name="v1_database_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/database_service.proto



<a name="bytebase-v1-BatchGetDatabasesRequest"></a>

### BatchGetDatabasesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource shared by all databases being retrieved. - projects/{project}: batch get databases in a project; - instances/{instances}: batch get databases in a instance; Use &#34;-&#34; as wildcard to batch get databases across parent. |
| names | [string](#string) | repeated | The list of database names to retrieve. |






<a name="bytebase-v1-BatchGetDatabasesResponse"></a>

### BatchGetDatabasesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [Database](#bytebase-v1-Database) | repeated | The databases from the specified request. |






<a name="bytebase-v1-BatchSyncDatabasesRequest"></a>

### BatchSyncDatabasesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource shared by all databases being updated. Format: instances/{instance} If the operation spans parents, a dash (-) may be accepted as a wildcard. |
| names | [string](#string) | repeated | The list of database names to sync. |






<a name="bytebase-v1-BatchSyncDatabasesResponse"></a>

### BatchSyncDatabasesResponse







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






<a name="bytebase-v1-BoundingBox"></a>

### BoundingBox
BoundingBox defines the spatial bounds for GEOMETRY spatial indexes.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| xmin | [double](#double) |  | Minimum X coordinate |
| ymin | [double](#double) |  | Minimum Y coordinate |
| xmax | [double](#double) |  | Maximum X coordinate |
| ymax | [double](#double) |  | Maximum Y coordinate |






<a name="bytebase-v1-Changelog"></a>

### Changelog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: instances/{instance}/databases/{database}/changelogs/{changelog} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| status | [Changelog.Status](#bytebase-v1-Changelog-Status) |  |  |
| schema | [string](#string) |  |  |
| schema_size | [int64](#int64) |  |  |
| task_run | [string](#string) |  | Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |
| plan_title | [string](#string) |  | The title of the plan associated with this changelog&#39;s task run. This field is populated by deriving the plan from task_run for display purposes. |
| type | [Changelog.Type](#bytebase-v1-Changelog-Type) |  |  |






<a name="bytebase-v1-CheckConstraintMetadata"></a>

### CheckConstraintMetadata
CheckConstraintMetadata is the metadata for check constraints.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a check constraint. |
| expression | [string](#string) |  | The expression is the expression of a check constraint. |






<a name="bytebase-v1-ColumnMetadata"></a>

### ColumnMetadata
ColumnMetadata is the metadata for columns.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a column. |
| position | [int32](#int32) |  | The position is the position in columns. |
| has_default | [bool](#bool) |  |  |
| default | [string](#string) |  | The default value of column. |
| default_on_null | [bool](#bool) |  | Oracle specific metadata. The default_on_null is the default on null of a column. |
| on_update | [string](#string) |  | The on_update is the on update action of a column. For MySQL like databases, it&#39;s only supported for TIMESTAMP columns with CURRENT_TIMESTAMP as on update value. |
| nullable | [bool](#bool) |  | The nullable is the nullable of a column. |
| type | [string](#string) |  | The type is the type of a column. |
| character_set | [string](#string) |  | The character_set is the character_set of a column. |
| collation | [string](#string) |  | The collation is the collation of a column. |
| comment | [string](#string) |  | The comment is the comment of a column. |
| generation | [GenerationMetadata](#bytebase-v1-GenerationMetadata) |  | The generation is the generation of a column. |
| is_identity | [bool](#bool) |  |  |
| identity_generation | [ColumnMetadata.IdentityGeneration](#bytebase-v1-ColumnMetadata-IdentityGeneration) |  | The identity_generation is for identity columns, PG only. |
| identity_seed | [int64](#int64) |  | The identity_seed is for identity columns, MSSQL only. |
| identity_increment | [int64](#int64) |  | The identity_increment is for identity columns, MSSQL only. |
| default_constraint_name | [string](#string) |  | The default_constraint_name is the name of the default constraint, MSSQL only. In MSSQL, default values are implemented as named constraints. When modifying or dropping a column&#39;s default value, you must reference the constraint by name. This field stores the actual constraint name from the database.

Example: A column definition like: CREATE TABLE employees ( status NVARCHAR(20) DEFAULT &#39;active&#39; )

Will create a constraint with an auto-generated name like &#39;DF__employees__statu__3B75D760&#39; or a user-defined name if specified: ALTER TABLE employees ADD CONSTRAINT DF_employees_status DEFAULT &#39;active&#39; FOR status

To modify the default, you must first drop the existing constraint by name: ALTER TABLE employees DROP CONSTRAINT DF__employees__statu__3B75D760 ALTER TABLE employees ADD CONSTRAINT DF_employees_status DEFAULT &#39;inactive&#39; FOR status

This field is populated when syncing from the database. When empty (e.g., when parsing from SQL files), the system cannot automatically drop the constraint. |






<a name="bytebase-v1-Database"></a>

### Database



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database. Format: instances/{instance}/databases/{database} {database} is the database name in the instance. |
| state | [State](#bytebase-v1-State) |  | The existence of a database. |
| successful_sync_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The latest synchronization time. |
| project | [string](#string) |  | The project for a database. Format: projects/{project} |
| schema_version | [string](#string) |  | The version of database schema. |
| environment | [string](#string) | optional | The environment resource. Format: environments/prod where prod is the environment resource ID. |
| effective_environment | [string](#string) | optional | The effective environment based on environment tag above and environment tag on the instance. Inheritance follows https://cloud.google.com/resource-manager/docs/tags/tags-overview. |
| labels | [Database.LabelsEntry](#bytebase-v1-Database-LabelsEntry) | repeated | Labels will be used for deployment and policy control. |
| instance_resource | [InstanceResource](#bytebase-v1-InstanceResource) |  | The instance resource. |
| backup_available | [bool](#bool) |  | The database is available for DML prior backup. |
| drifted | [bool](#bool) |  | The schema is drifted from the source of truth. |






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
| name | [string](#string) |  | The database metadata name.

Format: instances/{instance}/databases/{database}/metadata |
| schemas | [SchemaMetadata](#bytebase-v1-SchemaMetadata) | repeated | The schemas is the list of schemas in a database. |
| character_set | [string](#string) |  | The character_set is the character set of a database. |
| collation | [string](#string) |  | The collation is the collation of a database. |
| extensions | [ExtensionMetadata](#bytebase-v1-ExtensionMetadata) | repeated | The extensions is the list of extensions in a database. |
| owner | [string](#string) |  | The owner of the database. |
| search_path | [string](#string) |  | The search_path is the search path of a PostgreSQL database. |






<a name="bytebase-v1-DatabaseSDLSchema"></a>

### DatabaseSDLSchema
DatabaseSDLSchema contains the schema in SDL format.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [bytes](#bytes) |  | The SDL schema content. - For SINGLE_FILE format: contains the complete SDL schema as a text string. - For MULTI_FILE format: contains the ZIP archive as binary data. |
| content_type | [string](#string) |  | The MIME type of the schema content. Indicates how the client should interpret the schema field. Examples: - &#34;text/plain; charset=utf-8&#34; for SINGLE_FILE format - &#34;application/zip&#34; for MULTI_FILE format |






<a name="bytebase-v1-DatabaseSchema"></a>

### DatabaseSchema
DatabaseSchema is the metadata for databases.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  | The schema dump from database. |






<a name="bytebase-v1-DependencyColumn"></a>

### DependencyColumn
DependencyColumn is the metadata for dependency columns.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  | The schema is the schema of a reference column. |
| table | [string](#string) |  | The table is the table of a reference column. |
| column | [string](#string) |  | The column is the name of a reference column. |






<a name="bytebase-v1-DependencyTable"></a>

### DependencyTable



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema | [string](#string) |  | The schema is the schema of a reference table. |
| table | [string](#string) |  | The table is the name of a reference table. |






<a name="bytebase-v1-DiffSchemaRequest"></a>

### DiffSchemaRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database or changelog. Format: database: instances/{instance}/databases/{database} changelog: instances/{instance}/databases/{database}/changelogs/{changelog} |
| schema | [string](#string) |  | The target schema. |
| changelog | [string](#string) |  | The resource name of the changelog Format: instances/{instance}/databases/{database}/changelogs/{changelog} |






<a name="bytebase-v1-DiffSchemaResponse"></a>

### DiffSchemaResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| diff | [string](#string) |  |  |






<a name="bytebase-v1-DimensionConstraint"></a>

### DimensionConstraint
DimensionConstraint defines constraints for a spatial dimension.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| dimension | [string](#string) |  | Dimension name/type (X, Y, Z, M, etc.) |
| min_value | [double](#double) |  | Minimum value for this dimension |
| max_value | [double](#double) |  | Maximum value for this dimension |
| tolerance | [double](#double) |  | Tolerance for this dimension |






<a name="bytebase-v1-DimensionalConfig"></a>

### DimensionalConfig
DimensionalConfig defines dimensional and constraint parameters for spatial indexes.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| dimensions | [int32](#int32) |  | Number of dimensions (2-4, default 2) |
| data_type | [string](#string) |  | Spatial data type (GEOMETRY, GEOGRAPHY, POINT, POLYGON, etc.) |
| srid | [int32](#int32) |  | Spatial reference system identifier (SRID) |
| constraints | [DimensionConstraint](#bytebase-v1-DimensionConstraint) | repeated | Coordinate system constraints |






<a name="bytebase-v1-EnumTypeMetadata"></a>

### EnumTypeMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of a type. |
| values | [string](#string) | repeated | The enum values of a type. |
| comment | [string](#string) |  | The comment describing the enum type. |
| skip_dump | [bool](#bool) |  | Whether to skip this enum type during schema dump operations. |






<a name="bytebase-v1-EventMetadata"></a>

### EventMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the event. |
| definition | [string](#string) |  | The schedule of the event. |
| time_zone | [string](#string) |  | The time zone of the event. |
| sql_mode | [string](#string) |  | The SQL mode setting for the event. |
| character_set_client | [string](#string) |  | The character set used by the client creating the event. |
| collation_connection | [string](#string) |  | The collation used for the connection when creating the event. |
| comment | [string](#string) |  | The comment is the comment of an event. |






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






<a name="bytebase-v1-FunctionMetadata"></a>

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
| dependency_tables | [DependencyTable](#bytebase-v1-DependencyTable) | repeated | The dependency_tables is the list of dependency tables of a function. For PostgreSQL, it&#39;s the list of tables that the function depends on the return type definition. |
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-v1-GenerationMetadata"></a>

### GenerationMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [GenerationMetadata.Type](#bytebase-v1-GenerationMetadata-Type) |  |  |
| expression | [string](#string) |  |  |






<a name="bytebase-v1-GetChangelogRequest"></a>

### GetChangelogRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the changelog to retrieve. Format: instances/{instance}/databases/{database}/changelogs/{changelog} |
| view | [ChangelogView](#bytebase-v1-ChangelogView) |  |  |






<a name="bytebase-v1-GetDatabaseMetadataRequest"></a>

### GetDatabaseMetadataRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to retrieve metadata. Format: instances/{instance}/databases/{database}/metadata |
| filter | [string](#string) |  | Filter is used to filter databases returned in the list. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filter: - schema: the schema name, support &#34;==&#34; operator. - table: the table name, support &#34;==&#34; and &#34;.matches()&#34; operator.

For example: schema == &#34;schema-a&#34; table == &#34;table-a&#34; table.matches(&#34;table-a&#34;) schema == &#34;schema-a&#34; &amp;&amp; table.matches(&#34;sample&#34;) The filter used to search table with wildcard &#34;sample&#34; in the schema &#34;schemas/schema-a&#34;. The column masking level will only be returned when a table filter is used. |
| limit | [int32](#int32) |  | Limit the response size of returned table metadata per schema. For example, if the database has 3 schemas, and each schema has 100 tables, if limit is 20, then only 20 tables will be returned for each schema, total 60 tables. Default 0, means no limit. |






<a name="bytebase-v1-GetDatabaseRequest"></a>

### GetDatabaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to retrieve. Format: instances/{instance}/databases/{database} |






<a name="bytebase-v1-GetDatabaseSDLSchemaRequest"></a>

### GetDatabaseSDLSchemaRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to retrieve SDL schema. Format: instances/{instance}/databases/{database}/sdlSchema |
| format | [GetDatabaseSDLSchemaRequest.SDLFormat](#bytebase-v1-GetDatabaseSDLSchemaRequest-SDLFormat) |  | The format of the SDL schema output. |






<a name="bytebase-v1-GetDatabaseSchemaRequest"></a>

### GetDatabaseSchemaRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to retrieve schema. Format: instances/{instance}/databases/{database}/schema |






<a name="bytebase-v1-GetSchemaStringRequest"></a>

### GetSchemaStringRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database. Format: instances/{instance}/databases/{database} |
| type | [GetSchemaStringRequest.ObjectType](#bytebase-v1-GetSchemaStringRequest-ObjectType) |  |  |
| schema | [string](#string) |  | It&#39;s empty for DATABASE. |
| object | [string](#string) |  | It&#39;s empty for DATABASE and SCHEMA. |
| metadata | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  | If use the metadata to generate the schema string, the type is OBJECT_TYPE_UNSPECIFIED. Also the schema and object are empty. |






<a name="bytebase-v1-GetSchemaStringResponse"></a>

### GetSchemaStringResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| schema_string | [string](#string) |  |  |






<a name="bytebase-v1-GridLevel"></a>

### GridLevel
GridLevel defines a tessellation grid level with its density.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| level | [int32](#int32) |  | Grid level number (1-4 for SQL Server) |
| density | [string](#string) |  | Grid density (LOW, MEDIUM, HIGH) |






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
| parent_index_schema | [string](#string) |  | The schema name of the parent index. |
| parent_index_name | [string](#string) |  | The index name of the parent index. |
| granularity | [int64](#int64) |  | The number of granules in the block. It&#39;s a ClickHouse specific field. |
| is_constraint | [bool](#bool) |  | It&#39;s a PostgreSQL specific field. The unique constraint and unique index are not the same thing in PostgreSQL. |
| spatial_config | [SpatialIndexConfig](#bytebase-v1-SpatialIndexConfig) |  | Spatial index configuration for spatial databases like SQL Server, PostgreSQL with PostGIS, etc. |
| opclass_names | [string](#string) | repeated | https://www.postgresql.org/docs/current/catalog-pg-opclass.html Name of the operator class for each column. (PostgreSQL specific). |
| opclass_defaults | [bool](#bool) | repeated | True if the operator class is the default. (PostgreSQL specific). |






<a name="bytebase-v1-ListChangelogsRequest"></a>

### ListChangelogsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent of the changelogs. Format: instances/{instance}/databases/{database} |
| page_size | [int32](#int32) |  | The maximum number of changelogs to return. The service may return fewer than this value. If unspecified, at most 10 changelogs will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from the previous call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided must match the call that provided the page token. |
| view | [ChangelogView](#bytebase-v1-ChangelogView) |  |  |
| filter | [string](#string) |  | Filter is used to filter changelogs returned in the list. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filter: status: the changelog status, support &#34;==&#34; operation. check Changelog.Status for available values. type: the changelog type, support &#34;in&#34; and &#34;==&#34; operation. check Changelog.Type for available values.

Example: status == &#34;DONE&#34; type in [&#34;BASELINE&#34;, &#34;MIGRATE&#34;] status == &#34;FAILED&#34; &amp;&amp; type == &#34;SDL&#34; |






<a name="bytebase-v1-ListChangelogsResponse"></a>

### ListChangelogsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| changelogs | [Changelog](#bytebase-v1-Changelog) | repeated | The list of changelogs. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-ListDatabasesRequest"></a>

### ListDatabasesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | - projects/{project}: list databases in a project, require &#34;bb.projects.get&#34; permission. - workspaces/-: list databases in the workspace, require &#34;bb.databases.list&#34; permission. - instances/{instances}: list databases in a instance, require &#34;bb.instances.get&#34; permission |
| page_size | [int32](#int32) |  | The maximum number of databases to return. The service may return fewer than this value. If unspecified, at most 10 databases will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListDatabases` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListDatabases` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter databases returned in the list. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filter: - environment: the environment full name in &#34;environments/{id}&#34; format, support &#34;==&#34; operator. - name: the database name, support &#34;.matches()&#34; operator. - project: the project full name in &#34;projects/{id}&#34; format, support &#34;==&#34; operator. - instance: the instance full name in &#34;instances/{id}&#34; format, support &#34;==&#34; operator. - engine: the database engine, check Engine enum for values. Support &#34;==&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34; operator. - exclude_unassigned: should be &#34;true&#34; or &#34;false&#34;, will not show unassigned databases if it&#39;s true, support &#34;==&#34; operator. - drifted: should be &#34;true&#34; or &#34;false&#34;, show drifted databases if it&#39;s true, support &#34;==&#34; operator. - table: filter by the database table, support &#34;==&#34; and &#34;.matches()&#34; operator. - labels.{key}: the database label, support &#34;==&#34; and &#34;in&#34; operators.

For example: environment == &#34;environments/{environment resource id}&#34; environment == &#34;&#34; (find databases which environment is not set) project == &#34;projects/{project resource id}&#34; instance == &#34;instances/{instance resource id}&#34; name.matches(&#34;database name&#34;) engine == &#34;MYSQL&#34; engine in [&#34;MYSQL&#34;, &#34;POSTGRES&#34;] !(engine in [&#34;MYSQL&#34;, &#34;POSTGRES&#34;]) exclude_unassigned == true drifted == true table == &#34;sample&#34; table.matches(&#34;sam&#34;) labels.environment == &#34;production&#34; labels.region == &#34;asia&#34; labels.region in [&#34;asia&#34;, &#34;europe&#34;]

You can combine filter conditions like: environment == &#34;environments/prod&#34; &amp;&amp; name.matches(&#34;employee&#34;) |
| show_deleted | [bool](#bool) |  | Show deleted database if specified. |
| order_by | [string](#string) |  | The order by of databases. Support name, project, instance. The default sorting order is ascending. For example: - order_by = &#34;name&#34; - order by name ascending - order_by = &#34;name desc&#34; - order_by = &#34;name desc, project asc&#34; |






<a name="bytebase-v1-ListDatabasesResponse"></a>

### ListDatabasesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [Database](#bytebase-v1-Database) | repeated | The databases from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-MaterializedViewMetadata"></a>

### MaterializedViewMetadata
MaterializedViewMetadata is the metadata for materialized views.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a materialized view. |
| definition | [string](#string) |  | The definition is the definition of a materialized view. |
| comment | [string](#string) |  | The comment is the comment of a materialized view. |
| dependency_columns | [DependencyColumn](#bytebase-v1-DependencyColumn) | repeated | The dependency_columns is the list of dependency columns of a materialized view. |
| triggers | [TriggerMetadata](#bytebase-v1-TriggerMetadata) | repeated | The columns is the ordered list of columns in a table. |
| indexes | [IndexMetadata](#bytebase-v1-IndexMetadata) | repeated | The indexes is the list of indexes in a table. |
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-v1-PackageMetadata"></a>

### PackageMetadata
PackageMetadata is the metadata for packages.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a package. |
| definition | [string](#string) |  | The definition is the definition of a package. |






<a name="bytebase-v1-ProcedureMetadata"></a>

### ProcedureMetadata
ProcedureMetadata is the metadata for procedures.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a procedure. |
| definition | [string](#string) |  | The definition is the definition of a procedure. |
| signature | [string](#string) |  | The signature is the name with the number and type of input arguments the procedure takes. |
| character_set_client | [string](#string) |  | MySQL specific metadata. |
| collation_connection | [string](#string) |  |  |
| database_collation | [string](#string) |  |  |
| sql_mode | [string](#string) |  |  |
| comment | [string](#string) |  | The comment is the comment of a procedure. |
| skip_dump | [bool](#bool) |  |  |






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
| packages | [PackageMetadata](#bytebase-v1-PackageMetadata) | repeated | The packages is the list of packages in a schema. |
| owner | [string](#string) |  | The owner of the schema. |
| sequences | [SequenceMetadata](#bytebase-v1-SequenceMetadata) | repeated | The sequences is the list of sequences in a schema, sorted by name. |
| events | [EventMetadata](#bytebase-v1-EventMetadata) | repeated | The events is the list of scheduled events in a schema. |
| enum_types | [EnumTypeMetadata](#bytebase-v1-EnumTypeMetadata) | repeated | The enum_types is the list of user-defined enum types in a schema. |
| skip_dump | [bool](#bool) |  | Whether to skip this schema during schema dump operations. |
| comment | [string](#string) |  | The comment is the comment of a schema. |






<a name="bytebase-v1-SequenceMetadata"></a>

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
| comment | [string](#string) |  | The comment describing the sequence. |
| skip_dump | [bool](#bool) |  | Whether to skip this sequence during schema dump operations. |






<a name="bytebase-v1-SpatialIndexConfig"></a>

### SpatialIndexConfig
SpatialIndexConfig defines the spatial index configuration for spatial databases.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| method | [string](#string) |  | Spatial indexing method (e.g., &#34;SPATIAL&#34;, &#34;R-TREE&#34;, &#34;GIST&#34;) |
| tessellation | [TessellationConfig](#bytebase-v1-TessellationConfig) |  | Tessellation configuration for grid-based spatial indexes |
| storage | [StorageConfig](#bytebase-v1-StorageConfig) |  | Storage and performance configuration |
| dimensional | [DimensionalConfig](#bytebase-v1-DimensionalConfig) |  | Dimensional configuration |






<a name="bytebase-v1-StorageConfig"></a>

### StorageConfig
StorageConfig defines storage and performance parameters for spatial indexes.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| fillfactor | [int32](#int32) |  | Fill factor percentage (1-100) |
| buffering | [string](#string) |  | Buffering mode for PostgreSQL (auto, on, off) |
| tablespace | [string](#string) |  | Tablespace configuration for Oracle |
| work_tablespace | [string](#string) |  |  |
| sdo_level | [int32](#int32) |  |  |
| commit_interval | [int32](#int32) |  |  |
| pad_index | [bool](#bool) |  | SQL Server specific parameters |
| sort_in_tempdb | [string](#string) |  | ON, OFF |
| drop_existing | [bool](#bool) |  |  |
| online | [bool](#bool) |  |  |
| allow_row_locks | [bool](#bool) |  |  |
| allow_page_locks | [bool](#bool) |  |  |
| maxdop | [int32](#int32) |  |  |
| data_compression | [string](#string) |  | NONE, ROW, PAGE |






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
| charset | [string](#string) |  | The character set of table. |
| row_count | [int64](#int64) |  | The row_count is the estimated number of rows of a table. |
| data_size | [int64](#int64) |  | The data_size is the estimated data size of a table. |
| index_size | [int64](#int64) |  | The index_size is the estimated index size of a table. |
| data_free | [int64](#int64) |  | The data_free is the estimated free data size of a table. |
| create_options | [string](#string) |  | The create_options is the create option of a table. |
| comment | [string](#string) |  | The comment is the comment of a table. |
| foreign_keys | [ForeignKeyMetadata](#bytebase-v1-ForeignKeyMetadata) | repeated | The foreign_keys is the list of foreign keys in a table. |
| partitions | [TablePartitionMetadata](#bytebase-v1-TablePartitionMetadata) | repeated | The partitions is the list of partitions in a table. |
| check_constraints | [CheckConstraintMetadata](#bytebase-v1-CheckConstraintMetadata) | repeated | The check_constraints is the list of check constraints in a table. |
| owner | [string](#string) |  | The owner of the table. |
| sorting_keys | [string](#string) | repeated | The sorting_keys is a tuple of column names or arbitrary expressions. ClickHouse specific field. Reference: https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/mergetree#order_by |
| triggers | [TriggerMetadata](#bytebase-v1-TriggerMetadata) | repeated | The triggers is the list of triggers associated with the table. |
| skip_dump | [bool](#bool) |  | Whether to skip this table during schema dump operations. |
| sharding_info | [string](#string) |  | https://docs.pingcap.com/tidb/stable/information-schema-tables/ |
| primary_key_type | [string](#string) |  | https://docs.pingcap.com/tidb/stable/clustered-indexes/#clustered-indexes CLUSTERED or NONCLUSTERED. |






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
| indexes | [IndexMetadata](#bytebase-v1-IndexMetadata) | repeated |  |
| check_constraints | [CheckConstraintMetadata](#bytebase-v1-CheckConstraintMetadata) | repeated |  |






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






<a name="bytebase-v1-TessellationConfig"></a>

### TessellationConfig
TessellationConfig defines tessellation parameters for spatial indexes.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| scheme | [string](#string) |  | Tessellation scheme (e.g., &#34;GEOMETRY_GRID&#34;, &#34;GEOGRAPHY_GRID&#34;, &#34;GEOMETRY_AUTO_GRID&#34;) |
| grid_levels | [GridLevel](#bytebase-v1-GridLevel) | repeated | Grid levels and densities for multi-level tessellation |
| cells_per_object | [int32](#int32) |  | Number of cells per object (1-8192 for SQL Server) |
| bounding_box | [BoundingBox](#bytebase-v1-BoundingBox) |  | Bounding box for GEOMETRY tessellation (not used for GEOGRAPHY) |






<a name="bytebase-v1-TriggerMetadata"></a>

### TriggerMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of the trigger. |
| event | [string](#string) |  | The event is the event of the trigger, such as INSERT, UPDATE, DELETE, TRUNCATE. |
| timing | [string](#string) |  | The timing is the timing of the trigger, such as BEFORE, AFTER. |
| body | [string](#string) |  | The body is the body of the trigger. |
| sql_mode | [string](#string) |  | The SQL mode setting for the trigger. |
| character_set_client | [string](#string) |  | The character set used by the client creating the trigger. |
| collation_connection | [string](#string) |  | The collation used for the connection when creating the trigger. |
| comment | [string](#string) |  | The comment describing the trigger. |
| skip_dump | [bool](#bool) |  | Whether to skip this trigger during schema dump operations. |






<a name="bytebase-v1-UpdateDatabaseRequest"></a>

### UpdateDatabaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [Database](#bytebase-v1-Database) |  | The database to update.

The database&#39;s `name` field is used to identify the database to update. Format: instances/{instance}/databases/{database} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the database is not found, a new database will be created. In this situation, `update_mask` is ignored. |






<a name="bytebase-v1-ViewMetadata"></a>

### ViewMetadata
ViewMetadata is the metadata for views.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the name of a view. |
| definition | [string](#string) |  | The definition is the definition of a view. |
| comment | [string](#string) |  | The comment is the comment of a view. |
| dependency_columns | [DependencyColumn](#bytebase-v1-DependencyColumn) | repeated | The dependency_columns is the list of dependency columns of a view. |
| columns | [ColumnMetadata](#bytebase-v1-ColumnMetadata) | repeated | The columns is the ordered list of columns in a table. |
| triggers | [TriggerMetadata](#bytebase-v1-TriggerMetadata) | repeated | The triggers is the list of triggers in a view. |
| skip_dump | [bool](#bool) |  |  |





 


<a name="bytebase-v1-Changelog-Status"></a>

### Changelog.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| PENDING | 1 |  |
| DONE | 2 |  |
| FAILED | 3 |  |



<a name="bytebase-v1-Changelog-Type"></a>

### Changelog.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| BASELINE | 1 |  |
| MIGRATE | 2 |  |
| SDL | 3 |  |



<a name="bytebase-v1-ChangelogView"></a>

### ChangelogView


| Name | Number | Description |
| ---- | ------ | ----------- |
| CHANGELOG_VIEW_UNSPECIFIED | 0 | The default / unset value. The API will default to the BASIC view. |
| CHANGELOG_VIEW_BASIC | 1 |  |
| CHANGELOG_VIEW_FULL | 2 |  |



<a name="bytebase-v1-ColumnMetadata-IdentityGeneration"></a>

### ColumnMetadata.IdentityGeneration


| Name | Number | Description |
| ---- | ------ | ----------- |
| IDENTITY_GENERATION_UNSPECIFIED | 0 |  |
| ALWAYS | 1 |  |
| BY_DEFAULT | 2 |  |



<a name="bytebase-v1-GenerationMetadata-Type"></a>

### GenerationMetadata.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| VIRTUAL | 1 |  |
| STORED | 2 |  |



<a name="bytebase-v1-GetDatabaseSDLSchemaRequest-SDLFormat"></a>

### GetDatabaseSDLSchemaRequest.SDLFormat
SDLFormat specifies the output format for SDL schema.

| Name | Number | Description |
| ---- | ------ | ----------- |
| SDL_FORMAT_UNSPECIFIED | 0 | Unspecified format. Defaults to SINGLE_FILE. |
| SINGLE_FILE | 1 | Single file format: returns the complete SDL schema as a single file. |
| MULTI_FILE | 2 | Multi-file format: returns the SDL schema as a ZIP archive containing multiple files organized by schema objects (tables, views, functions, etc.). |



<a name="bytebase-v1-GetSchemaStringRequest-ObjectType"></a>

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



<a name="bytebase-v1-StreamMetadata-Mode"></a>

### StreamMetadata.Mode


| Name | Number | Description |
| ---- | ------ | ----------- |
| MODE_UNSPECIFIED | 0 |  |
| DEFAULT | 1 |  |
| APPEND_ONLY | 2 |  |
| INSERT_ONLY | 3 |  |



<a name="bytebase-v1-StreamMetadata-Type"></a>

### StreamMetadata.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| DELTA | 1 |  |



<a name="bytebase-v1-TablePartitionMetadata-Type"></a>

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



<a name="bytebase-v1-TaskMetadata-State"></a>

### TaskMetadata.State


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATE_UNSPECIFIED | 0 |  |
| STARTED | 1 |  |
| SUSPENDED | 2 |  |


 

 


<a name="bytebase-v1-DatabaseService"></a>

### DatabaseService
DatabaseService manages databases and their schemas.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetDatabase | [GetDatabaseRequest](#bytebase-v1-GetDatabaseRequest) | [Database](#bytebase-v1-Database) | Retrieves a database by name. Permissions required: bb.databases.get |
| BatchGetDatabases | [BatchGetDatabasesRequest](#bytebase-v1-BatchGetDatabasesRequest) | [BatchGetDatabasesResponse](#bytebase-v1-BatchGetDatabasesResponse) | Retrieves multiple databases by their names. Permissions required: bb.databases.get |
| ListDatabases | [ListDatabasesRequest](#bytebase-v1-ListDatabasesRequest) | [ListDatabasesResponse](#bytebase-v1-ListDatabasesResponse) | Lists databases in a project, instance, or workspace. Permissions required: bb.projects.get (for project parent), bb.databases.list (for workspace parent), or bb.instances.get (for instance parent) |
| UpdateDatabase | [UpdateDatabaseRequest](#bytebase-v1-UpdateDatabaseRequest) | [Database](#bytebase-v1-Database) | Updates database properties such as labels and project assignment. Permissions required: bb.databases.update |
| BatchUpdateDatabases | [BatchUpdateDatabasesRequest](#bytebase-v1-BatchUpdateDatabasesRequest) | [BatchUpdateDatabasesResponse](#bytebase-v1-BatchUpdateDatabasesResponse) | Updates multiple databases in a single batch operation. Permissions required: bb.databases.update |
| SyncDatabase | [SyncDatabaseRequest](#bytebase-v1-SyncDatabaseRequest) | [SyncDatabaseResponse](#bytebase-v1-SyncDatabaseResponse) | Synchronizes database schema from the instance. Permissions required: bb.databases.sync |
| BatchSyncDatabases | [BatchSyncDatabasesRequest](#bytebase-v1-BatchSyncDatabasesRequest) | [BatchSyncDatabasesResponse](#bytebase-v1-BatchSyncDatabasesResponse) | Synchronizes multiple databases in a single batch operation. Permissions required: bb.databases.sync |
| GetDatabaseMetadata | [GetDatabaseMetadataRequest](#bytebase-v1-GetDatabaseMetadataRequest) | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) | Retrieves database metadata including tables, columns, and indexes. Permissions required: bb.databases.getSchema |
| GetDatabaseSchema | [GetDatabaseSchemaRequest](#bytebase-v1-GetDatabaseSchemaRequest) | [DatabaseSchema](#bytebase-v1-DatabaseSchema) | Retrieves database schema as DDL statements. Permissions required: bb.databases.getSchema |
| GetDatabaseSDLSchema | [GetDatabaseSDLSchemaRequest](#bytebase-v1-GetDatabaseSDLSchemaRequest) | [DatabaseSDLSchema](#bytebase-v1-DatabaseSDLSchema) | Retrieves database schema in SDL (Schema Definition Language) format. Permissions required: bb.databases.getSchema |
| DiffSchema | [DiffSchemaRequest](#bytebase-v1-DiffSchemaRequest) | [DiffSchemaResponse](#bytebase-v1-DiffSchemaResponse) | Compares and generates migration statements between two schemas. Permissions required: bb.databases.get |
| ListChangelogs | [ListChangelogsRequest](#bytebase-v1-ListChangelogsRequest) | [ListChangelogsResponse](#bytebase-v1-ListChangelogsResponse) | Lists migration history for a database. Permissions required: bb.changelogs.list |
| GetChangelog | [GetChangelogRequest](#bytebase-v1-GetChangelogRequest) | [Changelog](#bytebase-v1-Changelog) | Retrieves a specific changelog entry. Permissions required: bb.changelogs.get |
| GetSchemaString | [GetSchemaStringRequest](#bytebase-v1-GetSchemaStringRequest) | [GetSchemaStringResponse](#bytebase-v1-GetSchemaStringResponse) | Generates schema DDL for a database object. Permissions required: bb.databases.getSchema |

 



<a name="v1_group_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/group_service.proto



<a name="bytebase-v1-BatchGetGroupsRequest"></a>

### BatchGetGroupsRequest
Request message for batch getting groups.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| names | [string](#string) | repeated | The group names to retrieve. Format: groups/{email} |






<a name="bytebase-v1-BatchGetGroupsResponse"></a>

### BatchGetGroupsResponse
Response message for batch getting groups.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| groups | [Group](#bytebase-v1-Group) | repeated | The groups from the specified request. |






<a name="bytebase-v1-CreateGroupRequest"></a>

### CreateGroupRequest
Request message for creating a group.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [Group](#bytebase-v1-Group) |  | The group to create. |
| group_email | [string](#string) |  | The email to use for the group, which will become the final component of the group&#39;s resource name. |






<a name="bytebase-v1-DeleteGroupRequest"></a>

### DeleteGroupRequest
Request message for deleting a group.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the group to delete. Format: groups/{email} |






<a name="bytebase-v1-GetGroupRequest"></a>

### GetGroupRequest
Request message for getting a group.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the group to retrieve. Format: groups/{email} |






<a name="bytebase-v1-Group"></a>

### Group
A group of users within the workspace.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the group to retrieve. Format: groups/{group}, the group should be email or uuid. |
| title | [string](#string) |  | The display title of the group. |
| description | [string](#string) |  | The description of the group. |
| members | [GroupMember](#bytebase-v1-GroupMember) | repeated | The members of the group. |
| source | [string](#string) |  | The source system where the group originated (e.g., Entra ID for SCIM sync). |
| email | [string](#string) |  | The unique email for the group. |






<a name="bytebase-v1-GroupMember"></a>

### GroupMember
A member of a group with a role.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| member | [string](#string) |  | Member is the principal who belong to this group.

Format: users/hello@world.com |
| role | [GroupMember.Role](#bytebase-v1-GroupMember-Role) |  | The member&#39;s role in the group. |






<a name="bytebase-v1-ListGroupsRequest"></a>

### ListGroupsRequest
Request message for listing groups.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of groups to return. The service may return fewer than this value. If unspecified, at most 10 groups will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListGroups` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListGroups` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter groups returned in the list. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filter: - title: the group title, support &#34;==&#34; and &#34;.matches()&#34; operator. - email: the group email, support &#34;==&#34; and &#34;.matches()&#34; operator. - project: the project full name in &#34;projects/{id}&#34; format, support &#34;==&#34; operator.

For example: title == &#34;dba&#34; email == &#34;dba@bytebase.com&#34; title.matches(&#34;dba&#34;) email.matches(&#34;dba&#34;) project == &#34;projects/sample-project&#34; You can combine filter conditions like: title.matches(&#34;dba&#34;) || email.matches(&#34;dba&#34;) |






<a name="bytebase-v1-ListGroupsResponse"></a>

### ListGroupsResponse
Response message for listing groups.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| groups | [Group](#bytebase-v1-Group) | repeated | The groups from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-UpdateGroupRequest"></a>

### UpdateGroupRequest
Request message for updating a group.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [Group](#bytebase-v1-Group) |  | The group to update.

The group&#39;s `name` field is used to identify the group to update. Format: groups/{email} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the group is not found, a new group will be created. In this situation, `update_mask` is ignored. |





 


<a name="bytebase-v1-GroupMember-Role"></a>

### GroupMember.Role
The role of a group member.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ROLE_UNSPECIFIED | 0 | Unspecified role. |
| OWNER | 1 | Group owner. |
| MEMBER | 2 | Group member. |


 

 


<a name="bytebase-v1-GroupService"></a>

### GroupService
GroupService manages user groups for organizing users and permissions.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetGroup | [GetGroupRequest](#bytebase-v1-GetGroupRequest) | [Group](#bytebase-v1-Group) | Gets a group by name. Permissions required: bb.groups.get |
| BatchGetGroups | [BatchGetGroupsRequest](#bytebase-v1-BatchGetGroupsRequest) | [BatchGetGroupsResponse](#bytebase-v1-BatchGetGroupsResponse) | Gets multiple groups in a single request. Permissions required: bb.groups.get |
| ListGroups | [ListGroupsRequest](#bytebase-v1-ListGroupsRequest) | [ListGroupsResponse](#bytebase-v1-ListGroupsResponse) | Lists all groups in the workspace. Permissions required: bb.groups.list |
| CreateGroup | [CreateGroupRequest](#bytebase-v1-CreateGroupRequest) | [Group](#bytebase-v1-Group) | Creates a new group. Permissions required: bb.groups.create |
| UpdateGroup | [UpdateGroupRequest](#bytebase-v1-UpdateGroupRequest) | [Group](#bytebase-v1-Group) | Updates a group. Group owners or users with bb.groups.update permission can update. Permissions required: bb.groups.update OR caller is group owner When allow_missing=true, also requires: bb.groups.create |
| DeleteGroup | [DeleteGroupRequest](#bytebase-v1-DeleteGroupRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Deletes a group. Group owners or users with bb.groups.delete permission can delete. Permissions required: bb.groups.delete OR caller is group owner |

 



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
| validate_only | [bool](#bool) |  | If set to true, the request will be validated without actually creating the identity provider. |






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


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identifier | [string](#string) |  | Identifier is the field name of the unique identifier in 3rd-party idp user info. Required. |
| display_name | [string](#string) |  | DisplayName is the field name of display name in 3rd-party idp user info. Optional. |
| phone | [string](#string) |  | Phone is the field name of primary phone in 3rd-party idp user info. Optional. |
| groups | [string](#string) |  | Groups is the field name of groups in 3rd-party idp user info. Optional. Mainly used for OIDC: https://developer.okta.com/docs/guides/customize-tokens-groups-claim/main/ |






<a name="bytebase-v1-GetIdentityProviderRequest"></a>

### GetIdentityProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the identity provider to retrieve. Format: idps/{idp} |






<a name="bytebase-v1-IdentityProvider"></a>

### IdentityProvider



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the identity provider. Format: idps/{idp} |
| title | [string](#string) |  | The display title of the identity provider. |
| domain | [string](#string) |  | The domain for email matching when using this identity provider. |
| type | [IdentityProviderType](#bytebase-v1-IdentityProviderType) |  | The type of identity provider protocol. |
| config | [IdentityProviderConfig](#bytebase-v1-IdentityProviderConfig) |  | The configuration details for the identity provider. |






<a name="bytebase-v1-IdentityProviderConfig"></a>

### IdentityProviderConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| oauth2_config | [OAuth2IdentityProviderConfig](#bytebase-v1-OAuth2IdentityProviderConfig) |  | OAuth2 protocol configuration. |
| oidc_config | [OIDCIdentityProviderConfig](#bytebase-v1-OIDCIdentityProviderConfig) |  | OIDC protocol configuration. |
| ldap_config | [LDAPIdentityProviderConfig](#bytebase-v1-LDAPIdentityProviderConfig) |  | LDAP protocol configuration. |






<a name="bytebase-v1-LDAPIdentityProviderConfig"></a>

### LDAPIdentityProviderConfig
LDAPIdentityProviderConfig is the structure for LDAP identity provider config.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| host | [string](#string) |  | Host is the hostname or IP address of the LDAP server, e.g., &#34;ldap.example.com&#34;. |
| port | [int32](#int32) |  | Port is the port number of the LDAP server, e.g., 389. When not set, the default port of the corresponding security protocol will be used, i.e. 389 for StartTLS and 636 for LDAPS. |
| skip_tls_verify | [bool](#bool) |  | SkipTLSVerify controls whether to skip TLS certificate verification. |
| bind_dn | [string](#string) |  | BindDN is the DN of the user to bind as a service account to perform search requests. |
| bind_password | [string](#string) |  | BindPassword is the password of the user to bind as a service account. |
| base_dn | [string](#string) |  | BaseDN is the base DN to search for users, e.g., &#34;ou=users,dc=example,dc=com&#34;. |
| user_filter | [string](#string) |  | UserFilter is the filter to search for users, e.g., &#34;(uid=%s)&#34;. |
| security_protocol | [LDAPIdentityProviderConfig.SecurityProtocol](#bytebase-v1-LDAPIdentityProviderConfig-SecurityProtocol) |  | SecurityProtocol is the security protocol to be used for establishing connections with the LDAP server. |
| field_mapping | [FieldMapping](#bytebase-v1-FieldMapping) |  | FieldMapping is the mapping of the user attributes returned by the LDAP server. |






<a name="bytebase-v1-ListIdentityProvidersRequest"></a>

### ListIdentityProvidersRequest







<a name="bytebase-v1-ListIdentityProvidersResponse"></a>

### ListIdentityProvidersResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identity_providers | [IdentityProvider](#bytebase-v1-IdentityProvider) | repeated | The identity providers from the specified request. |






<a name="bytebase-v1-OAuth2IdentityProviderConfig"></a>

### OAuth2IdentityProviderConfig
OAuth2IdentityProviderConfig is the structure for OAuth2 identity provider config.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| auth_url | [string](#string) |  | The authorization endpoint URL for OAuth2 flow. |
| token_url | [string](#string) |  | The token endpoint URL for exchanging authorization code. |
| user_info_url | [string](#string) |  | The user information endpoint URL. |
| client_id | [string](#string) |  | The OAuth2 client identifier. |
| client_secret | [string](#string) |  | The OAuth2 client secret for authentication. |
| scopes | [string](#string) | repeated | The list of OAuth2 scopes to request. |
| field_mapping | [FieldMapping](#bytebase-v1-FieldMapping) |  | Mapping configuration for user attributes from OAuth2 response. |
| skip_tls_verify | [bool](#bool) |  | Whether to skip TLS certificate verification. |
| auth_style | [OAuth2AuthStyle](#bytebase-v1-OAuth2AuthStyle) |  | The authentication style for client credentials. |






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
| issuer | [string](#string) |  | The OIDC issuer URL for the identity provider. |
| client_id | [string](#string) |  | The OIDC client identifier. |
| client_secret | [string](#string) |  | The OIDC client secret for authentication. |
| scopes | [string](#string) | repeated | The scopes that the OIDC provider supports. Should be fetched from the well-known configuration file of the OIDC provider. |
| field_mapping | [FieldMapping](#bytebase-v1-FieldMapping) |  | Mapping configuration for user attributes from OIDC claims. |
| skip_tls_verify | [bool](#bool) |  | Whether to skip TLS certificate verification. |
| auth_style | [OAuth2AuthStyle](#bytebase-v1-OAuth2AuthStyle) |  | The authentication style for client credentials. |
| auth_endpoint | [string](#string) |  | The authorization endpoint of the OIDC provider. Should be fetched from the well-known configuration file of the OIDC provider. |






<a name="bytebase-v1-OIDCIdentityProviderTestRequestContext"></a>

### OIDCIdentityProviderTestRequestContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [string](#string) |  | Authorize code from OIDC provider. |






<a name="bytebase-v1-TestIdentityProviderRequest"></a>

### TestIdentityProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identity_provider | [IdentityProvider](#bytebase-v1-IdentityProvider) |  | The identity provider to test connection including uncreated. |
| oauth2_context | [OAuth2IdentityProviderTestRequestContext](#bytebase-v1-OAuth2IdentityProviderTestRequestContext) |  |  |
| oidc_context | [OIDCIdentityProviderTestRequestContext](#bytebase-v1-OIDCIdentityProviderTestRequestContext) |  | OIDC authentication context for test connection. |






<a name="bytebase-v1-TestIdentityProviderResponse"></a>

### TestIdentityProviderResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| claims | [TestIdentityProviderResponse.ClaimsEntry](#bytebase-v1-TestIdentityProviderResponse-ClaimsEntry) | repeated | The map of claims returned by the identity provider. |
| user_info | [TestIdentityProviderResponse.UserInfoEntry](#bytebase-v1-TestIdentityProviderResponse-UserInfoEntry) | repeated | The matched user info from the claims. |






<a name="bytebase-v1-TestIdentityProviderResponse-ClaimsEntry"></a>

### TestIdentityProviderResponse.ClaimsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-v1-TestIdentityProviderResponse-UserInfoEntry"></a>

### TestIdentityProviderResponse.UserInfoEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-v1-UpdateIdentityProviderRequest"></a>

### UpdateIdentityProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identity_provider | [IdentityProvider](#bytebase-v1-IdentityProvider) |  | The identity provider to update.

The identity provider&#39;s `name` field is used to identify the identity provider to update. Format: idps/{identity_provider} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the identity provider is not found, a new identity provider will be created. In this situation, `update_mask` is ignored. |





 


<a name="bytebase-v1-IdentityProviderType"></a>

### IdentityProviderType


| Name | Number | Description |
| ---- | ------ | ----------- |
| IDENTITY_PROVIDER_TYPE_UNSPECIFIED | 0 | Unspecified identity provider type. |
| OAUTH2 | 1 | OAuth 2.0 authentication protocol. |
| OIDC | 2 | OpenID Connect authentication protocol. |
| LDAP | 3 | LDAP directory service authentication. |



<a name="bytebase-v1-LDAPIdentityProviderConfig-SecurityProtocol"></a>

### LDAPIdentityProviderConfig.SecurityProtocol


| Name | Number | Description |
| ---- | ------ | ----------- |
| SECURITY_PROTOCOL_UNSPECIFIED | 0 |  |
| START_TLS | 1 | StartTLS is the security protocol that starts with an unencrypted connection and then upgrades to TLS. |
| LDAPS | 2 | LDAPS is the security protocol that uses TLS from the beginning. |



<a name="bytebase-v1-OAuth2AuthStyle"></a>

### OAuth2AuthStyle


| Name | Number | Description |
| ---- | ------ | ----------- |
| OAUTH2_AUTH_STYLE_UNSPECIFIED | 0 |  |
| IN_PARAMS | 1 | IN_PARAMS sends the &#34;client_id&#34; and &#34;client_secret&#34; in the POST body as application/x-www-form-urlencoded parameters. |
| IN_HEADER | 2 | IN_HEADER sends the client_id and client_password using HTTP Basic Authorization. This is an optional style described in the OAuth2 RFC 6749 section 2.3.1. |


 

 


<a name="bytebase-v1-IdentityProviderService"></a>

### IdentityProviderService
IdentityProviderService manages external identity providers for SSO authentication.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetIdentityProvider | [GetIdentityProviderRequest](#bytebase-v1-GetIdentityProviderRequest) | [IdentityProvider](#bytebase-v1-IdentityProvider) | Gets an identity provider by name. Permissions required: bb.identityProviders.get |
| ListIdentityProviders | [ListIdentityProvidersRequest](#bytebase-v1-ListIdentityProvidersRequest) | [ListIdentityProvidersResponse](#bytebase-v1-ListIdentityProvidersResponse) | Lists all configured identity providers (public endpoint for login page). Permissions required: None |
| CreateIdentityProvider | [CreateIdentityProviderRequest](#bytebase-v1-CreateIdentityProviderRequest) | [IdentityProvider](#bytebase-v1-IdentityProvider) | Creates a new identity provider. Permissions required: bb.identityProviders.create |
| UpdateIdentityProvider | [UpdateIdentityProviderRequest](#bytebase-v1-UpdateIdentityProviderRequest) | [IdentityProvider](#bytebase-v1-IdentityProvider) | Updates an identity provider. Permissions required: bb.identityProviders.update When allow_missing=true, also requires: bb.identityProviders.create |
| DeleteIdentityProvider | [DeleteIdentityProviderRequest](#bytebase-v1-DeleteIdentityProviderRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Deletes an identity provider. Permissions required: bb.identityProviders.delete |
| TestIdentityProvider | [TestIdentityProviderRequest](#bytebase-v1-TestIdentityProviderRequest) | [TestIdentityProviderResponse](#bytebase-v1-TestIdentityProviderResponse) | Tests the connection and configuration of an identity provider. Permissions required: bb.identityProviders.update |

 



<a name="v1_org_policy_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/org_policy_service.proto



<a name="bytebase-v1-CreatePolicyRequest"></a>

### CreatePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this instance will be created. Workspace resource name: &#34;&#34;. Environment resource name: environments/environment-id. Instance resource name: instances/instance-id. Database resource name: instances/instance-id/databases/database-name. |
| policy | [Policy](#bytebase-v1-Policy) |  | The policy to create. |
| type | [PolicyType](#bytebase-v1-PolicyType) |  | The type of policy to create. |






<a name="bytebase-v1-DataSourceQueryPolicy"></a>

### DataSourceQueryPolicy
Policy for controlling which data sources can be queried in the SQL editor.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| admin_data_source_restriction | [DataSourceQueryPolicy.Restriction](#bytebase-v1-DataSourceQueryPolicy-Restriction) |  | Restriction for admin data source queries. |
| disallow_ddl | [bool](#bool) |  | Disallow running DDL statements in the SQL editor. |
| disallow_dml | [bool](#bool) |  | Disallow running DML statements in the SQL editor. |






<a name="bytebase-v1-DeletePolicyRequest"></a>

### DeletePolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The policy&#39;s `name` field is used to identify the instance to update. Format: {resource name}/policies/{policy type} Workspace resource name: &#34;&#34;. Environment resource name: environments/environment-id. Instance resource name: instances/instance-id. Database resource name: instances/instance-id/databases/database-name. |






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
| policy_type | [PolicyType](#bytebase-v1-PolicyType) | optional | Filter by specific policy type. |
| show_deleted | [bool](#bool) |  | Show deleted policies if specified. |






<a name="bytebase-v1-ListPoliciesResponse"></a>

### ListPoliciesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policies | [Policy](#bytebase-v1-Policy) | repeated | The policies from the specified request. |






<a name="bytebase-v1-MaskingExemptionPolicy"></a>

### MaskingExemptionPolicy
MaskingExemptionPolicy is the allowlist of users who can access sensitive data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| exemptions | [MaskingExemptionPolicy.Exemption](#bytebase-v1-MaskingExemptionPolicy-Exemption) | repeated |  |






<a name="bytebase-v1-MaskingExemptionPolicy-Exemption"></a>

### MaskingExemptionPolicy.Exemption



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| members | [string](#string) | repeated | Specifies the principals who are exempt from masking. For users, the member should be: user:{email} For groups, the member should be: group:{email} |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this exception policy instance. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec If the condition is empty, means the user can access all databases without expiration.

Support variables: resource.instance_id: the instance resource id. Only support &#34;==&#34; operation. resource.database_name: the database name. Only support &#34;==&#34; operation. resource.schema_name: the schema name. Only support &#34;==&#34; operation. resource.table_name: the table name. Only support &#34;==&#34; operation. resource.column_name: the column name. Only support &#34;==&#34; operation. request.time: the expiration. Only support &#34;&lt;&#34; operation in `request.time &lt; timestamp(&#34;{ISO datetime string format}&#34;)` All variables should join with &#34;&amp;&amp;&#34; condition.

For example: resource.instance_id == &#34;local&#34; &amp;&amp; resource.database_name == &#34;employee&#34; &amp;&amp; request.time &lt; timestamp(&#34;2025-04-30T11:10:39.000Z&#34;) resource.instance_id == &#34;local&#34; &amp;&amp; resource.database_name == &#34;employee&#34; |






<a name="bytebase-v1-MaskingRulePolicy"></a>

### MaskingRulePolicy
Policy for configuring data masking rules.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rules | [MaskingRulePolicy.MaskingRule](#bytebase-v1-MaskingRulePolicy-MaskingRule) | repeated | The list of masking rules. |






<a name="bytebase-v1-MaskingRulePolicy-MaskingRule"></a>

### MaskingRulePolicy.MaskingRule
A rule that defines when and how to mask data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | A unique identifier for the rule in UUID format. |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition for the masking rule. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Support variables: resource.environment_id: the environment resource id. resource.project_id: the project resource id. resource.instance_id: the instance resource id. resource.database_name: the database name. resource.table_name: the table name. resource.column_name: the column name. resource.classification_level: the classification level.

Each variable support following operations: ==: the value equals the target. !=: the value not equals the target. in: the value matches one of the targets. !(in): the value not matches any of the targets.

For example: resource.environment_id == &#34;test&#34; &amp;&amp; resource.project_id == &#34;sample-project&#34; resource.instance_id == &#34;sample-instance&#34; &amp;&amp; resource.database_name == &#34;employee&#34; &amp;&amp; resource.table_name in [&#34;table1&#34;, &#34;table2&#34;] resource.environment_id != &#34;test&#34; || !(resource.project_id in [&#34;poject1&#34;, &#34;prject2&#34;]) resource.instance_id == &#34;sample-instance&#34; &amp;&amp; (resource.database_name == &#34;db1&#34; || resource.database_name == &#34;db2&#34;) |
| semantic_type | [string](#string) |  | The semantic type of data to mask (e.g., &#34;SSN&#34;, &#34;EMAIL&#34;). |






<a name="bytebase-v1-Policy"></a>

### Policy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the policy. Format: {resource name}/policies/{policy type} Workspace resource name: &#34;&#34;. Environment resource name: environments/environment-id. Instance resource name: instances/instance-id. Database resource name: instances/instance-id/databases/database-name. |
| inherit_from_parent | [bool](#bool) |  | Whether this policy inherits from its parent resource. |
| type | [PolicyType](#bytebase-v1-PolicyType) |  | The type of policy. |
| rollout_policy | [RolloutPolicy](#bytebase-v1-RolloutPolicy) |  |  |
| masking_rule_policy | [MaskingRulePolicy](#bytebase-v1-MaskingRulePolicy) |  |  |
| masking_exemption_policy | [MaskingExemptionPolicy](#bytebase-v1-MaskingExemptionPolicy) |  |  |
| tag_policy | [TagPolicy](#bytebase-v1-TagPolicy) |  |  |
| data_source_query_policy | [DataSourceQueryPolicy](#bytebase-v1-DataSourceQueryPolicy) |  |  |
| query_data_policy | [QueryDataPolicy](#bytebase-v1-QueryDataPolicy) |  |  |
| enforce | [bool](#bool) |  | Whether the policy is enforced. |
| resource_type | [PolicyResourceType](#bytebase-v1-PolicyResourceType) |  | The resource type for the policy. |






<a name="bytebase-v1-QueryDataPolicy"></a>

### QueryDataPolicy
QueryDataPolicy is the policy configuration for querying data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timeout | [google.protobuf.Duration](#google-protobuf-Duration) |  | The query timeout duration. |
| disable_export | [bool](#bool) |  | Disable data export in the SQL editor. |
| maximum_result_size | [int64](#int64) |  | The maximum result size limit in bytes. The default value is 100MB, we will use the default value if the setting not exists, or the limit &lt;= 0. |
| maximum_result_rows | [int32](#int32) |  | The maximum number of rows to return. The default value is -1, means no limit. |
| disable_copy_data | [bool](#bool) |  | Disable copying query results. |






<a name="bytebase-v1-RolloutPolicy"></a>

### RolloutPolicy
Rollout policy configuration.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| automatic | [bool](#bool) |  | Whether rollout is automatic without manual approval. |
| roles | [string](#string) | repeated | The roles that can approve rollout execution. |






<a name="bytebase-v1-TagPolicy"></a>

### TagPolicy
Policy for tagging resources with metadata.


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





 


<a name="bytebase-v1-DataSourceQueryPolicy-Restriction"></a>

### DataSourceQueryPolicy.Restriction
Restriction level for admin data source access.

| Name | Number | Description |
| ---- | ------ | ----------- |
| RESTRICTION_UNSPECIFIED | 0 | Unspecified restriction. |
| FALLBACK | 1 | Allow querying admin data sources when there is no read-only data source. |
| DISALLOW | 2 | Disallow querying admin data sources. |



<a name="bytebase-v1-PolicyResourceType"></a>

### PolicyResourceType
The resource type that a policy can be attached to.

| Name | Number | Description |
| ---- | ------ | ----------- |
| RESOURCE_TYPE_UNSPECIFIED | 0 | Unspecified resource type. |
| WORKSPACE | 1 | Workspace-level policy. |
| ENVIRONMENT | 2 | Environment-level policy. |
| PROJECT | 3 | Project-level policy. |



<a name="bytebase-v1-PolicyType"></a>

### PolicyType
The type of organizational policy.

| Name | Number | Description |
| ---- | ------ | ----------- |
| POLICY_TYPE_UNSPECIFIED | 0 | Unspecified policy type. |
| MASKING_RULE | 1 | Data masking rule policy. |
| MASKING_EXEMPTION | 2 | Data masking exemption policy. |
| ROLLOUT_POLICY | 3 | Rollout deployment policy. |
| TAG | 4 | Resource tag policy. |
| DATA_SOURCE_QUERY | 5 | Data source query restrictions policy. |
| DATA_QUERY | 6 | Query data access policy. |


 

 


<a name="bytebase-v1-OrgPolicyService"></a>

### OrgPolicyService
OrgPolicyService manages organizational policies at various resource levels.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetPolicy | [GetPolicyRequest](#bytebase-v1-GetPolicyRequest) | [Policy](#bytebase-v1-Policy) | Retrieves a policy by name. Permissions required: bb.policies.get |
| ListPolicies | [ListPoliciesRequest](#bytebase-v1-ListPoliciesRequest) | [ListPoliciesResponse](#bytebase-v1-ListPoliciesResponse) | Lists policies at a specified resource level. Permissions required: bb.policies.list |
| CreatePolicy | [CreatePolicyRequest](#bytebase-v1-CreatePolicyRequest) | [Policy](#bytebase-v1-Policy) | Creates a new organizational policy. Permissions required: bb.policies.create |
| UpdatePolicy | [UpdatePolicyRequest](#bytebase-v1-UpdatePolicyRequest) | [Policy](#bytebase-v1-Policy) | Updates an existing organizational policy. Permissions required: bb.policies.update |
| DeletePolicy | [DeletePolicyRequest](#bytebase-v1-DeletePolicyRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Deletes an organizational policy. Permissions required: bb.policies.delete |

 



<a name="v1_sql_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/sql_service.proto



<a name="bytebase-v1-AICompletionRequest"></a>

### AICompletionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| messages | [AICompletionRequest.Message](#bytebase-v1-AICompletionRequest-Message) | repeated |  |






<a name="bytebase-v1-AICompletionRequest-Message"></a>

### AICompletionRequest.Message



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [string](#string) |  |  |
| content | [string](#string) |  |  |






<a name="bytebase-v1-AICompletionResponse"></a>

### AICompletionResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| candidates | [AICompletionResponse.Candidate](#bytebase-v1-AICompletionResponse-Candidate) | repeated | candidates is used for results with multiple choices and candidates. Used for OpenAI and Gemini. |






<a name="bytebase-v1-AICompletionResponse-Candidate"></a>

### AICompletionResponse.Candidate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| content | [AICompletionResponse.Candidate.Content](#bytebase-v1-AICompletionResponse-Candidate-Content) |  |  |






<a name="bytebase-v1-AICompletionResponse-Candidate-Content"></a>

### AICompletionResponse.Candidate.Content



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parts | [AICompletionResponse.Candidate.Content.Part](#bytebase-v1-AICompletionResponse-Candidate-Content-Part) | repeated | parts is used for a result content with multiple parts. |






<a name="bytebase-v1-AICompletionResponse-Candidate-Content-Part"></a>

### AICompletionResponse.Candidate.Content.Part



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| text | [string](#string) |  |  |






<a name="bytebase-v1-AdminExecuteRequest"></a>

### AdminExecuteRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the instance name to execute the query against. Format: instances/{instance}/databases/{databaseName} |
| statement | [string](#string) |  | The SQL statement to execute. |
| limit | [int32](#int32) |  | The maximum number of rows to return. |
| schema | [string](#string) | optional | The default schema to execute the statement. Equals to the current schema in Oracle and search path in Postgres. |
| container | [string](#string) | optional | Container is the container name to execute the query against, used for CosmosDB only. |






<a name="bytebase-v1-AdminExecuteResponse"></a>

### AdminExecuteResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [QueryResult](#bytebase-v1-QueryResult) | repeated | The query results. |






<a name="bytebase-v1-Advice"></a>

### Advice



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [Advice.Level](#bytebase-v1-Advice-Level) |  | The advice level. |
| code | [int32](#int32) |  | The advice code. |
| title | [string](#string) |  | The advice title. |
| content | [string](#string) |  | The advice content. |
| start_position | [Position](#bytebase-v1-Position) |  | The start_position is inclusive and the end_position is exclusive. TODO: use range instead |
| end_position | [Position](#bytebase-v1-Position) |  |  |
| rule_type | [Advice.RuleType](#bytebase-v1-Advice-RuleType) |  | The type of linting rule that generated this advice. |






<a name="bytebase-v1-DiffMetadataRequest"></a>

### DiffMetadataRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| source_metadata | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  | The metadata of the source schema. |
| target_metadata | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  | The metadata of the target schema. |
| engine | [Engine](#bytebase-v1-Engine) |  | The database engine of the schema. |






<a name="bytebase-v1-DiffMetadataResponse"></a>

### DiffMetadataResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| diff | [string](#string) |  | The diff of the metadata. |






<a name="bytebase-v1-ExportRequest"></a>

### ExportRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the resource name to execute the export against. Format: instances/{instance}/databases/{database} Format: instances/{instance} Format: projects/{project}/rollouts/{rollout} Format: projects/{project}/rollouts/{rollout}/stages/{stage} |
| statement | [string](#string) |  | The SQL statement to execute. |
| limit | [int32](#int32) |  | The maximum number of rows to return. |
| format | [ExportFormat](#bytebase-v1-ExportFormat) |  | The export format. |
| admin | [bool](#bool) |  | The admin is used for workspace owner and DBA for exporting data from SQL Editor Admin mode. The exported data is not masked. |
| password | [string](#string) |  | The zip password provide by users. |
| data_source_id | [string](#string) |  | The id of data source. It is used for querying admin data source even if the instance has read-only data sources. Or it can be used to query a specific read-only data source. |
| schema | [string](#string) | optional | The default schema to search objects. Equals to the current schema in Oracle and search path in Postgres. |






<a name="bytebase-v1-ExportResponse"></a>

### ExportResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| content | [bytes](#bytes) |  | The export file content. |






<a name="bytebase-v1-MaskingReason"></a>

### MaskingReason



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| semantic_type_id | [string](#string) |  | The semantic type that triggered masking (e.g., &#34;SSN&#34;, &#34;email&#34;, &#34;phone&#34;). |
| semantic_type_title | [string](#string) |  | Human-readable semantic type title. |
| masking_rule_id | [string](#string) |  | The masking rule ID that matched (if applicable). |
| algorithm | [string](#string) |  | The masking algorithm used. |
| context | [string](#string) |  | Additional context (e.g., &#34;Matched global rule: PII Protection&#34;). |
| classification_level | [string](#string) |  | Whether masking was due to classification level. |
| semantic_type_icon | [string](#string) |  | Icon associated with the semantic type (if any). |






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






<a name="bytebase-v1-QueryOption"></a>

### QueryOption



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| redis_run_commands_on | [QueryOption.RedisRunCommandsOn](#bytebase-v1-QueryOption-RedisRunCommandsOn) |  |  |
| mssql_explain_format | [QueryOption.MSSQLExplainFormat](#bytebase-v1-QueryOption-MSSQLExplainFormat) |  |  |






<a name="bytebase-v1-QueryRequest"></a>

### QueryRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name is the instance name to execute the query against. Format: instances/{instance}/databases/{databaseName} |
| statement | [string](#string) |  | The SQL statement to execute. |
| limit | [int32](#int32) |  | The maximum number of rows to return. |
| data_source_id | [string](#string) |  | The id of data source. It is used for querying admin data source even if the instance has read-only data sources. Or it can be used to query a specific read-only data source. |
| explain | [bool](#bool) |  | Explain the statement. |
| schema | [string](#string) | optional | The default schema to search objects. Equals to the current schema in Oracle and search path in Postgres. |
| query_option | [QueryOption](#bytebase-v1-QueryOption) |  |  |
| container | [string](#string) | optional | Container is the container name to execute the query against, used for CosmosDB only. |






<a name="bytebase-v1-QueryResponse"></a>

### QueryResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [QueryResult](#bytebase-v1-QueryResult) | repeated | The query results. |






<a name="bytebase-v1-QueryResult"></a>

### QueryResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| column_names | [string](#string) | repeated | Column names of the query result. |
| column_type_names | [string](#string) | repeated | Column types of the query result. The types come from the Golang SQL driver. |
| rows | [QueryRow](#bytebase-v1-QueryRow) | repeated | Rows of the query result. |
| rows_count | [int64](#int64) |  |  |
| error | [string](#string) |  | The error message if the query failed. |
| latency | [google.protobuf.Duration](#google-protobuf-Duration) |  | The time it takes to execute the query. |
| statement | [string](#string) |  | The query statement for the result. |
| postgres_error | [QueryResult.PostgresError](#bytebase-v1-QueryResult-PostgresError) |  |  |
| syntax_error | [QueryResult.SyntaxError](#bytebase-v1-QueryResult-SyntaxError) |  |  |
| permission_denied | [QueryResult.PermissionDenied](#bytebase-v1-QueryResult-PermissionDenied) |  |  |
| messages | [QueryResult.Message](#bytebase-v1-QueryResult-Message) | repeated | Informational or debug messages returned by the database engine during query execution. Examples include PostgreSQL&#39;s RAISE NOTICE, MSSQL&#39;s PRINT, or Oracle&#39;s DBMS_OUTPUT.PUT_LINE. |
| masked | [MaskingReason](#bytebase-v1-MaskingReason) | repeated | Masking reasons for each column (empty for non-masked columns). |






<a name="bytebase-v1-QueryResult-Message"></a>

### QueryResult.Message



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| level | [QueryResult.Message.Level](#bytebase-v1-QueryResult-Message-Level) |  |  |
| content | [string](#string) |  |  |






<a name="bytebase-v1-QueryResult-PermissionDenied"></a>

### QueryResult.PermissionDenied
Permission denied with resource information or disallowed command_type.
Either resources or command_type is available.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resources | [string](#string) | repeated | Denied to access the resources. Format: instances/{instance}/databases/{database} instances/{instance}/databases/{database}/schemas/{schema} instances/{instance}/databases/{database}/tables/{table} instances/{instance}/databases/{database}/schemas/{schema}/tables/{table} |
| command_type | [QueryResult.PermissionDenied.CommandType](#bytebase-v1-QueryResult-PermissionDenied-CommandType) |  | Disallowed command_type. |






<a name="bytebase-v1-QueryResult-PostgresError"></a>

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






<a name="bytebase-v1-QueryResult-SyntaxError"></a>

### QueryResult.SyntaxError
Syntax error with position information for editor highlighting


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start_position | [Position](#bytebase-v1-Position) |  | Position information for highlighting in editor |






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
| timestamp_value | [RowValue.Timestamp](#bytebase-v1-RowValue-Timestamp) |  | timestamp_value is used for the timestamp without time zone data type, meaning it only includes the timestamp without any time zone or location info. Although it may be expressed as a UTC value, it should be seen as a timestamp missing location context. |
| timestamp_tz_value | [RowValue.TimestampTZ](#bytebase-v1-RowValue-TimestampTZ) |  | timestamp_tz_value is used for the timestamptz data type, which accurately represents the timestamp with location information. |






<a name="bytebase-v1-RowValue-Timestamp"></a>

### RowValue.Timestamp



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| google_timestamp | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| accuracy | [int32](#int32) |  | The accuracy is the number of digits after the decimal point. |






<a name="bytebase-v1-RowValue-TimestampTZ"></a>

### RowValue.TimestampTZ



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| google_timestamp | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| zone | [string](#string) |  | Zone is the time zone abbreviations in timezone database such as &#34;PDT&#34;, &#34;PST&#34;. https://en.wikipedia.org/wiki/List_of_tz_database_time_zones We retrieve the time zone information from the timestamptz field in the database. A timestamp is in UTC or epoch time, and with zone info, we can convert it to a local time string. Zone and offset are returned by time.Time.Zone() |
| offset | [int32](#int32) |  | The offset is in seconds east of UTC |
| accuracy | [int32](#int32) |  |  |






<a name="bytebase-v1-SearchQueryHistoriesRequest"></a>

### SearchQueryHistoriesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of histories to return. The service may return fewer than this value. If unspecified, at most 10 history entries will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListQueryHistory` call. Provide this to retrieve the subsequent page. |
| filter | [string](#string) |  | Filter is the filter to apply on the search query history The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filter: - project: the project full name in &#34;projects/{id}&#34; format, support &#34;==&#34; operator. - database: the database full name in &#34;instances/{id}/databases/{name}&#34; format, support &#34;==&#34; operator. - instance: the instance full name in &#34;instances/{id}&#34; format, support &#34;==&#34; operator. - type: the type, should be &#34;QUERY&#34; or &#34;EXPORT&#34;, support &#34;==&#34; operator. - statement: the SQL statement, support &#34;.matches()&#34; operator.

For example: project == &#34;projects/{project}&#34; database == &#34;instances/{instance}/databases/{database}&#34; instance == &#34;instances/{instance}&#34; type == &#34;QUERY&#34; type == &#34;EXPORT&#34; statement.matches(&#34;select&#34;) type == &#34;QUERY&#34; &amp;&amp; statement.matches(&#34;select&#34;) |






<a name="bytebase-v1-SearchQueryHistoriesResponse"></a>

### SearchQueryHistoriesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| query_histories | [QueryHistory](#bytebase-v1-QueryHistory) | repeated | The list of history. |
| next_page_token | [string](#string) |  | A token to retrieve next page of history. Pass this value in the page_token field in the subsequent call to `ListQueryHistory` method to retrieve the next page of history. |





 


<a name="bytebase-v1-Advice-Level"></a>

### Advice.Level
Level represents the severity level of the advice.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ADVICE_LEVEL_UNSPECIFIED | 0 | Unspecified advice level. |
| SUCCESS | 1 | Success status indicating the check passed without issues. |
| WARNING | 2 | Warning status indicating potential issues that should be reviewed. |
| ERROR | 3 | Error status indicating critical issues that must be addressed. |



<a name="bytebase-v1-Advice-RuleType"></a>

### Advice.RuleType
RuleType indicates the source of the linting rule.

| Name | Number | Description |
| ---- | ------ | ----------- |
| RULE_TYPE_UNSPECIFIED | 0 | Unspecified rule type. |
| PARSER_BASED | 1 | Parser-based rule enforced by the built-in SQL parser. These are non-configurable, mandatory checks for schema health. |
| AI_POWERED | 2 | AI-powered rule defined by user in natural language. These are custom, flexible checks powered by AI models. |



<a name="bytebase-v1-QueryHistory-Type"></a>

### QueryHistory.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 | Unspecified query history type. |
| QUERY | 1 | Query execution for data retrieval. |
| EXPORT | 2 | Data export operation to file. |



<a name="bytebase-v1-QueryOption-MSSQLExplainFormat"></a>

### QueryOption.MSSQLExplainFormat


| Name | Number | Description |
| ---- | ------ | ----------- |
| MSSQL_EXPLAIN_FORMAT_UNSPECIFIED | 0 | defaults to SHOWPLAN_ALL |
| MSSQL_EXPLAIN_FORMAT_ALL | 1 | SHOWPLAN_ALL |
| MSSQL_EXPLAIN_FORMAT_XML | 2 | SHOWPLAN_XML |



<a name="bytebase-v1-QueryOption-RedisRunCommandsOn"></a>

### QueryOption.RedisRunCommandsOn


| Name | Number | Description |
| ---- | ------ | ----------- |
| REDIS_RUN_COMMANDS_ON_UNSPECIFIED | 0 | UNSPECIFIED defaults to SINGLE_NODE. |
| SINGLE_NODE | 1 | Execute Redis commands on a single node in the cluster. |
| ALL_NODES | 2 | Execute Redis commands on all nodes in the cluster for cluster-wide operations. |



<a name="bytebase-v1-QueryResult-Message-Level"></a>

### QueryResult.Message.Level


| Name | Number | Description |
| ---- | ------ | ----------- |
| LEVEL_UNSPECIFIED | 0 | Unspecified message level. |
| INFO | 1 | Informational message. |
| WARNING | 2 | Warning message indicating potential issues. |
| DEBUG | 3 | Debug message for development and troubleshooting. |
| LOG | 4 | General log message. |
| NOTICE | 5 | Notice message for important information. |
| EXCEPTION | 6 | Exception message indicating error conditions. |



<a name="bytebase-v1-QueryResult-PermissionDenied-CommandType"></a>

### QueryResult.PermissionDenied.CommandType


| Name | Number | Description |
| ---- | ------ | ----------- |
| COMMAND_TYPE_UNSPECIFIED | 0 |  |
| DDL | 1 |  |
| DML | 2 |  |
| NON_READ_ONLY | 3 |  |


 

 


<a name="bytebase-v1-SQLService"></a>

### SQLService
SQLService executes SQL queries and manages query operations.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Query | [QueryRequest](#bytebase-v1-QueryRequest) | [QueryResponse](#bytebase-v1-QueryResponse) | Executes a read-only SQL query against a database. Permissions required: bb.databases.get |
| AdminExecute | [AdminExecuteRequest](#bytebase-v1-AdminExecuteRequest) stream | [AdminExecuteResponse](#bytebase-v1-AdminExecuteResponse) stream | Executes SQL with admin privileges via streaming connection. Permissions required: bb.sql.admin |
| SearchQueryHistories | [SearchQueryHistoriesRequest](#bytebase-v1-SearchQueryHistoriesRequest) | [SearchQueryHistoriesResponse](#bytebase-v1-SearchQueryHistoriesResponse) | SearchQueryHistories searches query histories for the caller. Permissions required: None (only returns caller&#39;s own query histories) |
| Export | [ExportRequest](#bytebase-v1-ExportRequest) | [ExportResponse](#bytebase-v1-ExportResponse) | Exports query results to a file format. Permissions required: bb.databases.get |
| DiffMetadata | [DiffMetadataRequest](#bytebase-v1-DiffMetadataRequest) | [DiffMetadataResponse](#bytebase-v1-DiffMetadataResponse) | Computes schema differences between two database metadata. Permissions required: None |
| AICompletion | [AICompletionRequest](#bytebase-v1-AICompletionRequest) | [AICompletionResponse](#bytebase-v1-AICompletionResponse) | Provides AI-powered SQL completion and generation. Permissions required: None (authenticated users only, requires AI to be enabled) |

 



<a name="v1_plan_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/plan_service.proto



<a name="bytebase-v1-CancelPlanCheckRunRequest"></a>

### CancelPlanCheckRunRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the plan check run to cancel. Format: projects/{project}/plans/{plan}/planCheckRun |






<a name="bytebase-v1-CancelPlanCheckRunResponse"></a>

### CancelPlanCheckRunResponse







<a name="bytebase-v1-CreatePlanRequest"></a>

### CreatePlanRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent project where this plan will be created. Format: projects/{project} |
| plan | [Plan](#bytebase-v1-Plan) |  | The plan to create. |






<a name="bytebase-v1-GetPlanCheckRunRequest"></a>

### GetPlanCheckRunRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the plan check run to retrieve. Format: projects/{project}/plans/{plan}/planCheckRun |






<a name="bytebase-v1-GetPlanRequest"></a>

### GetPlanRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the plan to retrieve. Format: projects/{project}/plans/{plan} |






<a name="bytebase-v1-ListPlansRequest"></a>

### ListPlansRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of plans. Format: projects/{project} Use &#34;projects/-&#34; to list all plans from all projects. |
| page_size | [int32](#int32) |  | The maximum number of plans to return. The service may return fewer than this value. If unspecified, at most 10 plans will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListPlans` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListPlans` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter plans returned in the list. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filters: - creator: the plan creator full name in &#34;users/{email or id}&#34; format, support &#34;==&#34; operator. - create_time: the plan create time in &#34;2006-01-02T15:04:05Z07:00&#34; format, support &#34;&gt;=&#34; or &#34;&lt;=&#34; operator. - has_rollout: whether the plan has rollout, support &#34;==&#34; operator, the value should be &#34;true&#34; or &#34;false&#34;. - has_issue: the plan has issue or not, support &#34;==&#34; operator, the value should be &#34;true&#34; or &#34;false&#34;. - title: the plan title, support &#34;==&#34; operator for exact match and &#34;.matches()&#34; operator for case-insensitive substring match. - spec_type: the plan spec config type, support &#34;==&#34; operator, the value should be &#34;create_database_config&#34;, &#34;change_database_config&#34;, or &#34;export_data_config&#34;. - state: the plan state, support &#34;==&#34; operator, the value should be &#34;ACTIVE&#34; or &#34;DELETED&#34;.

For example: creator == &#34;users/ed@bytebase.com&#34; &amp;&amp; create_time &gt;= &#34;2025-01-02T15:04:05Z07:00&#34; has_rollout == false &amp;&amp; has_issue == true title == &#34;My Plan&#34; title.matches(&#34;database migration&#34;) spec_type == &#34;change_database_config&#34; state == &#34;ACTIVE&#34; |






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
| state | [State](#bytebase-v1-State) |  | The state of the plan. |
| issue | [string](#string) |  | The issue associated with the plan. Can be empty. Format: projects/{project}/issues/{issue} |
| title | [string](#string) |  | The title of the plan. |
| description | [string](#string) |  | The description of the plan. |
| specs | [Plan.Spec](#bytebase-v1-Plan-Spec) | repeated | The deployment specs for the plan. |
| creator | [string](#string) |  | Format: users/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| plan_check_run_status_count | [Plan.PlanCheckRunStatusCountEntry](#bytebase-v1-Plan-PlanCheckRunStatusCountEntry) | repeated | The status count of the latest plan check runs. Keys are: - SUCCESS - WARNING - ERROR - RUNNING |
| has_rollout | [bool](#bool) |  | Whether the plan has started the rollout. |






<a name="bytebase-v1-Plan-ChangeDatabaseConfig"></a>

### Plan.ChangeDatabaseConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| targets | [string](#string) | repeated | The list of targets. Multi-database format: [instances/{instance-id}/databases/{database-name}]. Single database group format: [projects/{project}/databaseGroups/{databaseGroup}]. |
| sheet | [string](#string) |  | The resource name of the sheet. Format: projects/{project}/sheets/{sheet} |
| release | [string](#string) |  | The resource name of the release. Format: projects/{project}/releases/{release} |
| ghost_flags | [Plan.ChangeDatabaseConfig.GhostFlagsEntry](#bytebase-v1-Plan-ChangeDatabaseConfig-GhostFlagsEntry) | repeated |  |
| enable_prior_backup | [bool](#bool) |  | If set, a backup of the modified data will be created automatically before any changes are applied. |
| enable_ghost | [bool](#bool) |  | Whether to use gh-ost for online schema migration. |






<a name="bytebase-v1-Plan-ChangeDatabaseConfig-GhostFlagsEntry"></a>

### Plan.ChangeDatabaseConfig.GhostFlagsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






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






<a name="bytebase-v1-Plan-ExportDataConfig"></a>

### Plan.ExportDataConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| targets | [string](#string) | repeated | The list of targets. Multi-database format: [instances/{instance-id}/databases/{database-name}]. Single database group format: [projects/{project}/databaseGroups/{databaseGroup}]. |
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
| id | [string](#string) |  | A UUID4 string that uniquely identifies the Spec. |
| create_database_config | [Plan.CreateDatabaseConfig](#bytebase-v1-Plan-CreateDatabaseConfig) |  |  |
| change_database_config | [Plan.ChangeDatabaseConfig](#bytebase-v1-Plan-ChangeDatabaseConfig) |  |  |
| export_data_config | [Plan.ExportDataConfig](#bytebase-v1-Plan-ExportDataConfig) |  |  |






<a name="bytebase-v1-PlanCheckRun"></a>

### PlanCheckRun



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/plans/{plan}/planCheckRun |
| status | [PlanCheckRun.Status](#bytebase-v1-PlanCheckRun-Status) |  |  |
| results | [PlanCheckRun.Result](#bytebase-v1-PlanCheckRun-Result) | repeated |  |
| error | [string](#string) |  | error is set if the Status is FAILED. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="bytebase-v1-PlanCheckRun-Result"></a>

### PlanCheckRun.Result



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [Advice.Level](#bytebase-v1-Advice-Level) |  |  |
| title | [string](#string) |  |  |
| content | [string](#string) |  |  |
| code | [int32](#int32) |  |  |
| target | [string](#string) |  | Target identification for consolidated results. Format: instances/{instance}/databases/{database} |
| type | [PlanCheckRun.Result.Type](#bytebase-v1-PlanCheckRun-Result-Type) |  |  |
| sql_summary_report | [PlanCheckRun.Result.SqlSummaryReport](#bytebase-v1-PlanCheckRun-Result-SqlSummaryReport) |  |  |
| sql_review_report | [PlanCheckRun.Result.SqlReviewReport](#bytebase-v1-PlanCheckRun-Result-SqlReviewReport) |  |  |






<a name="bytebase-v1-PlanCheckRun-Result-SqlReviewReport"></a>

### PlanCheckRun.Result.SqlReviewReport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start_position | [Position](#bytebase-v1-Position) |  | Position of the SQL statement. |
| end_position | [Position](#bytebase-v1-Position) |  |  |






<a name="bytebase-v1-PlanCheckRun-Result-SqlSummaryReport"></a>

### PlanCheckRun.Result.SqlSummaryReport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| statement_types | [string](#string) | repeated | statement_types are the types of statements that are found in the sql. |
| affected_rows | [int64](#int64) |  |  |






<a name="bytebase-v1-RunPlanChecksRequest"></a>

### RunPlanChecksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The plan to run plan checks. Format: projects/{project}/plans/{plan} |
| spec_id | [string](#string) | optional | The UUID of the specific spec to run plan checks for. This should match the spec.id field in Plan.Spec. If not set, all specs in the plan will be used. |






<a name="bytebase-v1-RunPlanChecksResponse"></a>

### RunPlanChecksResponse







<a name="bytebase-v1-UpdatePlanRequest"></a>

### UpdatePlanRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| plan | [Plan](#bytebase-v1-Plan) |  | The plan to update.

The plan&#39;s `name` field is used to identify the plan to update. Format: projects/{project}/plans/{plan} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the plan is not found, a new plan will be created. In this situation, `update_mask` is ignored. |





 


<a name="bytebase-v1-PlanCheckRun-Result-Type"></a>

### PlanCheckRun.Result.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| STATEMENT_ADVISE | 1 |  |
| STATEMENT_SUMMARY_REPORT | 2 |  |
| GHOST_SYNC | 3 |  |



<a name="bytebase-v1-PlanCheckRun-Status"></a>

### PlanCheckRun.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 |  |
| RUNNING | 1 |  |
| DONE | 2 |  |
| FAILED | 3 |  |
| CANCELED | 4 |  |


 

 


<a name="bytebase-v1-PlanService"></a>

### PlanService
PlanService manages deployment plans for database changes.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetPlan | [GetPlanRequest](#bytebase-v1-GetPlanRequest) | [Plan](#bytebase-v1-Plan) | Retrieves a deployment plan by name. Permissions required: bb.plans.get |
| ListPlans | [ListPlansRequest](#bytebase-v1-ListPlansRequest) | [ListPlansResponse](#bytebase-v1-ListPlansResponse) | Lists deployment plans in a project. Permissions required: bb.plans.list |
| CreatePlan | [CreatePlanRequest](#bytebase-v1-CreatePlanRequest) | [Plan](#bytebase-v1-Plan) | Creates a new deployment plan. Permissions required: bb.plans.create |
| UpdatePlan | [UpdatePlanRequest](#bytebase-v1-UpdatePlanRequest) | [Plan](#bytebase-v1-Plan) | UpdatePlan updates the plan. The plan creator and the user with bb.plans.update permission on the project can update the plan. Permissions required: bb.plans.update (or creator) |
| GetPlanCheckRun | [GetPlanCheckRunRequest](#bytebase-v1-GetPlanCheckRunRequest) | [PlanCheckRun](#bytebase-v1-PlanCheckRun) | Gets the plan check run for a deployment plan. Permissions required: bb.planCheckRuns.get |
| RunPlanChecks | [RunPlanChecksRequest](#bytebase-v1-RunPlanChecksRequest) | [RunPlanChecksResponse](#bytebase-v1-RunPlanChecksResponse) | Executes validation checks on a deployment plan. Permissions required: bb.planCheckRuns.run |
| CancelPlanCheckRun | [CancelPlanCheckRunRequest](#bytebase-v1-CancelPlanCheckRunRequest) | [CancelPlanCheckRunResponse](#bytebase-v1-CancelPlanCheckRunResponse) | Cancels the plan check run for a deployment plan. Permissions required: bb.planCheckRuns.run |

 



<a name="v1_project_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/project_service.proto



<a name="bytebase-v1-Activity"></a>

### Activity
Activity types for webhook notifications.






<a name="bytebase-v1-AddWebhookRequest"></a>

### AddWebhookRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [string](#string) |  | The name of the project to add the webhook to. Format: projects/{project} |
| webhook | [Webhook](#bytebase-v1-Webhook) |  | The webhook to add. |






<a name="bytebase-v1-BatchDeleteProjectsRequest"></a>

### BatchDeleteProjectsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| names | [string](#string) | repeated | The names of the projects to delete. Format: projects/{project} |
| force | [bool](#bool) |  | If set to true, any databases from this project will be moved to default project. Sheets are not moved since BYTEBASE_ARTIFACT sheets belong to the issue and issue project. Open issues will remain open but associated with the deleted project. If set to false, the operation will fail if the project has databases or open issues. |






<a name="bytebase-v1-BatchGetProjectsRequest"></a>

### BatchGetProjectsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| names | [string](#string) | repeated | The names of projects to retrieve. Format: projects/{project} |






<a name="bytebase-v1-BatchGetProjectsResponse"></a>

### BatchGetProjectsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| projects | [Project](#bytebase-v1-Project) | repeated | The projects from the specified request. |






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
| force | [bool](#bool) |  | If set to true, any databases from this project will be moved to default project. Sheets are not moved since BYTEBASE_ARTIFACT sheets belong to the issue and issue project. Open issues will remain open but associated with the deleted project. If set to false, the operation will fail if the project has databases or open issues. |
| purge | [bool](#bool) |  | If set to true, permanently purge the soft-deleted project and all related resources. This operation is irreversible. Following AIP-165, this should only be used for administrative cleanup of old soft-deleted projects. The project must already be soft-deleted for this to work. |






<a name="bytebase-v1-GetProjectRequest"></a>

### GetProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the project to retrieve. Format: projects/{project} |






<a name="bytebase-v1-Label"></a>

### Label
A label for categorizing and organizing issues.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [string](#string) |  | The label value/name. |
| color | [string](#string) |  | The color code for the label (e.g., hex color). |
| group | [string](#string) |  | The group this label belongs to. |






<a name="bytebase-v1-ListProjectsRequest"></a>

### ListProjectsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of projects to return. The service may return fewer than this value. If unspecified, at most 10 projects will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListProjects` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListProjects` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Show deleted projects if specified. |
| filter | [string](#string) |  | Filter the project. Check filter for SearchProjectsRequest for details. Supports filtering by name, resource_id, state, and labels (e.g., labels.environment == &#34;production&#34;). |
| order_by | [string](#string) |  | The order by of projects. Support title. The default sorting order is ascending. For example: - order_by = &#34;title&#34; - order_by = &#34;title desc&#34; - order_by = &#34;title asc&#34; |






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
| state | [State](#bytebase-v1-State) |  | The lifecycle state of the project. |
| title | [string](#string) |  | The title or name of a project. It&#39;s not unique within the workspace. |
| webhooks | [Webhook](#bytebase-v1-Webhook) | repeated | The list of webhooks configured for the project. |
| data_classification_config_id | [string](#string) |  | The data classification configuration ID for the project. |
| issue_labels | [Label](#bytebase-v1-Label) | repeated | Labels available for tagging issues in this project. |
| force_issue_labels | [bool](#bool) |  | Force issue labels to be used when creating an issue. |
| enforce_issue_title | [bool](#bool) |  | Enforce issue title to be created by user instead of generated by Bytebase. |
| auto_enable_backup | [bool](#bool) |  | Whether to automatically enable backup for database changes. |
| skip_backup_errors | [bool](#bool) |  | Whether to skip backup errors and continue with data migration. |
| postgres_database_tenant_mode | [bool](#bool) |  | Whether to enable database tenant mode for PostgreSQL. If enabled, issues will include &#34;set role &lt;db_owner&gt;&#34; statement. |
| allow_self_approval | [bool](#bool) |  | Whether to allow issue creators to self-approve their own issues. |
| execution_retry_policy | [Project.ExecutionRetryPolicy](#bytebase-v1-Project-ExecutionRetryPolicy) |  | Execution retry policy for task runs. |
| ci_sampling_size | [int32](#int32) |  | The maximum number of database rows to sample during CI data validation. Without specification, sampling is disabled, resulting in full validation. |
| parallel_tasks_per_rollout | [int32](#int32) |  | The maximum number of parallel tasks allowed during rollout execution. |
| labels | [Project.LabelsEntry](#bytebase-v1-Project-LabelsEntry) | repeated | Labels are key-value pairs that can be attached to the project. For example, { &#34;environment&#34;: &#34;production&#34;, &#34;team&#34;: &#34;backend&#34; } |
| enforce_sql_review | [bool](#bool) |  | Whether to enforce SQL review checks to pass before issue creation. If enabled, issues cannot be created when SQL review finds errors. |
| require_issue_approval | [bool](#bool) |  | Whether to require issue approval before rollout. |
| require_plan_check_no_error | [bool](#bool) |  | Whether to require plan check to have no error before rollout. |






<a name="bytebase-v1-Project-ExecutionRetryPolicy"></a>

### Project.ExecutionRetryPolicy
Execution retry policy configuration.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| maximum_retries | [int32](#int32) |  | The maximum number of retries for lock timeout errors. |






<a name="bytebase-v1-Project-LabelsEntry"></a>

### Project.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-v1-RemoveWebhookRequest"></a>

### RemoveWebhookRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| webhook | [Webhook](#bytebase-v1-Webhook) |  | The webhook to remove. Identified by its url. |






<a name="bytebase-v1-SearchProjectsRequest"></a>

### SearchProjectsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| show_deleted | [bool](#bool) |  | Show deleted projects if specified. |
| filter | [string](#string) |  | Filter the project. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filters: - name: the project name, support &#34;==&#34; and &#34;.matches()&#34; operator. - resource_id: the project id, support &#34;==&#34; and &#34;.matches()&#34; operator. - exclude_default: if not include the default project, should be &#34;true&#34; or &#34;false&#34;, support &#34;==&#34; operator. - state: check the State enum for the values, support &#34;==&#34; operator. - labels.{key}: the project label, support &#34;==&#34; and &#34;in&#34; operators.

For example: name == &#34;project name&#34; name.matches(&#34;project name&#34;) resource_id == &#34;project id&#34; resource_id.matches(&#34;project id&#34;) exclude_default == true state == &#34;DELETED&#34; labels.environment == &#34;production&#34; labels.tier == &#34;critical&#34; labels.environment in [&#34;staging&#34;, &#34;prod&#34;] You can combine filter conditions like: name == &#34;project name&#34; &amp;&amp; resource_id.matches(&#34;project id&#34;) name.matches(&#34;project name&#34;) || resource_id == &#34;project id&#34; labels.environment == &#34;production&#34; &amp;&amp; labels.tier == &#34;critical&#34; |
| page_size | [int32](#int32) |  | The maximum number of projects to return. The service may return fewer than this value. If unspecified, at most 10 projects will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `SearchProjects` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `SearchProjects` must match the call that provided the page token. |
| order_by | [string](#string) |  | The order by of projects. Support title. The default sorting order is ascending. For example: - order_by = &#34;title&#34; - order_by = &#34;title desc&#34; - order_by = &#34;title asc&#34; |






<a name="bytebase-v1-SearchProjectsResponse"></a>

### SearchProjectsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| projects | [Project](#bytebase-v1-Project) | repeated | The projects from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






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






<a name="bytebase-v1-UpdateProjectRequest"></a>

### UpdateProjectRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| project | [Project](#bytebase-v1-Project) |  | The project to update.

The project&#39;s `name` field is used to identify the project to update. Format: projects/{project} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the project is not found, a new project will be created. In this situation, `update_mask` is ignored. |






<a name="bytebase-v1-UpdateWebhookRequest"></a>

### UpdateWebhookRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| webhook | [Webhook](#bytebase-v1-Webhook) |  | The webhook to modify. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the webhook is not found, a new webhook will be created. In this situation, `update_mask` is ignored. |






<a name="bytebase-v1-Webhook"></a>

### Webhook



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | name is the name of the webhook, generated by the server. format: projects/{project}/webhooks/{webhook} |
| type | [WebhookType](#bytebase-v1-WebhookType) |  | Webhook integration type. type is the type of the webhook. |
| title | [string](#string) |  | title is the title of the webhook. |
| url | [string](#string) |  | url is the url of the webhook, should be unique within the project. |
| direct_message | [bool](#bool) |  | if direct_message is set, the notification is sent directly to the persons and url will be ignored. IM integration setting should be set for this function to work. |
| notification_types | [Activity.Type](#bytebase-v1-Activity-Type) | repeated | notification_types is the list of activities types that the webhook is interested in. Bytebase will only send notifications to the webhook if the activity type is in the list. It should not be empty, and should be a subset of the following: - ISSUE_CREATED - ISSUE_APPROVAL_REQUESTED - ISSUE_SENT_BACK - PIPELINE_FAILED - PIPELINE_COMPLETED |





 


<a name="bytebase-v1-Activity-Type"></a>

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


 

 


<a name="bytebase-v1-ProjectService"></a>

### ProjectService
ProjectService manages projects that group databases and changes.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetProject | [GetProjectRequest](#bytebase-v1-GetProjectRequest) | [Project](#bytebase-v1-Project) | GetProject retrieves a project by name. Users with &#34;bb.projects.get&#34; permission on the workspace or the project owner can access this method. Permissions required: bb.projects.get |
| BatchGetProjects | [BatchGetProjectsRequest](#bytebase-v1-BatchGetProjectsRequest) | [BatchGetProjectsResponse](#bytebase-v1-BatchGetProjectsResponse) | BatchGetProjects retrieves multiple projects by their names. Permissions required: bb.projects.get |
| ListProjects | [ListProjectsRequest](#bytebase-v1-ListProjectsRequest) | [ListProjectsResponse](#bytebase-v1-ListProjectsResponse) | Lists all projects in the workspace with optional filtering. Permissions required: bb.projects.list |
| SearchProjects | [SearchProjectsRequest](#bytebase-v1-SearchProjectsRequest) | [SearchProjectsResponse](#bytebase-v1-SearchProjectsResponse) | Searches for projects with advanced filtering capabilities. Permissions required: bb.projects.get (or project-level bb.projects.get for specific projects) |
| CreateProject | [CreateProjectRequest](#bytebase-v1-CreateProjectRequest) | [Project](#bytebase-v1-Project) | Creates a new project in the workspace. Permissions required: bb.projects.create |
| UpdateProject | [UpdateProjectRequest](#bytebase-v1-UpdateProjectRequest) | [Project](#bytebase-v1-Project) | Updates an existing project&#39;s properties. Permissions required: bb.projects.update |
| DeleteProject | [DeleteProjectRequest](#bytebase-v1-DeleteProjectRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Deletes (soft-delete or purge) a project. Permissions required: bb.projects.delete |
| UndeleteProject | [UndeleteProjectRequest](#bytebase-v1-UndeleteProjectRequest) | [Project](#bytebase-v1-Project) | Restores a soft-deleted project. Permissions required: bb.projects.undelete |
| BatchDeleteProjects | [BatchDeleteProjectsRequest](#bytebase-v1-BatchDeleteProjectsRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Deletes multiple projects in a single operation. Permissions required: bb.projects.delete |
| GetIamPolicy | [GetIamPolicyRequest](#bytebase-v1-GetIamPolicyRequest) | [IamPolicy](#bytebase-v1-IamPolicy) | Retrieves the IAM policy for a project. Permissions required: bb.projects.getIamPolicy |
| SetIamPolicy | [SetIamPolicyRequest](#bytebase-v1-SetIamPolicyRequest) | [IamPolicy](#bytebase-v1-IamPolicy) | Sets the IAM policy for a project. Permissions required: bb.projects.setIamPolicy |
| AddWebhook | [AddWebhookRequest](#bytebase-v1-AddWebhookRequest) | [Project](#bytebase-v1-Project) | Adds a webhook to a project for notifications. Permissions required: bb.projects.update |
| UpdateWebhook | [UpdateWebhookRequest](#bytebase-v1-UpdateWebhookRequest) | [Project](#bytebase-v1-Project) | Updates an existing webhook configuration. Permissions required: bb.projects.update |
| RemoveWebhook | [RemoveWebhookRequest](#bytebase-v1-RemoveWebhookRequest) | [Project](#bytebase-v1-Project) | Removes a webhook from a project. Permissions required: bb.projects.update |
| TestWebhook | [TestWebhookRequest](#bytebase-v1-TestWebhookRequest) | [TestWebhookResponse](#bytebase-v1-TestWebhookResponse) | Tests a webhook by sending a test notification. Permissions required: bb.projects.update |

 



<a name="v1_release_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/release_service.proto



<a name="bytebase-v1-CheckReleaseRequest"></a>

### CheckReleaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{project} |
| release | [Release](#bytebase-v1-Release) |  | The release to check. |
| targets | [string](#string) | repeated | The targets to dry-run the release. Can be database or databaseGroup. Format: projects/{project}/databaseGroups/{databaseGroup} instances/{instance}/databases/{database} |
| custom_rules | [string](#string) |  | Custom linting rules in natural language for AI-powered validation. Each rule should be a clear statement describing the desired schema constraint. Example: &#34;All tables must have a primary key&#34; Example: &#34;VARCHAR columns should specify a maximum length&#34; |






<a name="bytebase-v1-CheckReleaseResponse"></a>

### CheckReleaseResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [CheckReleaseResponse.CheckResult](#bytebase-v1-CheckReleaseResponse-CheckResult) | repeated | The check results for each file and target combination. |
| affected_rows | [int64](#int64) |  | The total affected rows across all checks. |
| risk_level | [RiskLevel](#bytebase-v1-RiskLevel) |  | The aggregated risk level of the check. |






<a name="bytebase-v1-CheckReleaseResponse-CheckResult"></a>

### CheckReleaseResponse.CheckResult
Check result for a single release file on a target database.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file | [string](#string) |  | The file path that is being checked. |
| target | [string](#string) |  | The target that the check is performed on. Should be a database. Format: instances/{instance}/databases/{database} |
| advices | [Advice](#bytebase-v1-Advice) | repeated | The list of advice for the file and the target. |
| affected_rows | [int64](#int64) |  | The count of affected rows of the statement on the target. |
| risk_level | [RiskLevel](#bytebase-v1-RiskLevel) |  | The risk level of the statement on the target. |






<a name="bytebase-v1-CreateReleaseRequest"></a>

### CreateReleaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{project} |
| release | [Release](#bytebase-v1-Release) |  | The release to create. |






<a name="bytebase-v1-DeleteReleaseRequest"></a>

### DeleteReleaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the release to delete. Format: projects/{project}/releases/{release} |






<a name="bytebase-v1-GetReleaseRequest"></a>

### GetReleaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/releases/{release} |






<a name="bytebase-v1-ListReleasesRequest"></a>

### ListReleasesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{project} |
| page_size | [int32](#int32) |  | The maximum number of releases to return. The service may return fewer than this value. If unspecified, at most 10 releases will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListReleases` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListReleases` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Show deleted releases if specified. |






<a name="bytebase-v1-ListReleasesResponse"></a>

### ListReleasesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| releases | [Release](#bytebase-v1-Release) | repeated | The releases from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-Release"></a>

### Release



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/releases/{release} |
| title | [string](#string) |  | The title of the release. |
| files | [Release.File](#bytebase-v1-Release-File) | repeated | The SQL files included in the release. |
| vcs_source | [Release.VCSSource](#bytebase-v1-Release-VCSSource) |  | The version control source of the release. |
| creator | [string](#string) |  | Format: users/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| state | [State](#bytebase-v1-State) |  | The lifecycle state of the release. |
| digest | [string](#string) |  | The digest of the release. The user can provide the digest of the release. It can be used later to retrieve the release in GetRelease. Whether to provide digest and how to generate it is up to the user. If the digest is not empty, it must be unique in the project. Otherwise, an ALREADY_EXISTS error will be returned. |
| type | [Release.Type](#bytebase-v1-Release-Type) |  | The type of schema change for all files in this release. |






<a name="bytebase-v1-Release-File"></a>

### Release.File
A SQL file in a release.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| path | [string](#string) |  | The path of the file. e.g., `2.2/V0001_create_table.sql`. |
| version | [string](#string) |  | The version identifier for the file. |
| enable_ghost | [bool](#bool) |  | Whether to use gh-ost for online schema migration. |
| sheet | [string](#string) |  | For inputs, we must either use `sheet` or `statement`. For outputs, we always use `sheet`. `statement` is the preview of the sheet content.

The sheet that holds the content. Format: projects/{project}/sheets/{sheet} |
| statement | [bytes](#bytes) |  | The raw SQL statement content. |
| sheet_sha256 | [string](#string) |  | The SHA256 hash value of the sheet content or the statement. |






<a name="bytebase-v1-Release-VCSSource"></a>

### Release.VCSSource
Version control system source information.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vcs_type | [VCSType](#bytebase-v1-VCSType) |  | The type of VCS. |
| url | [string](#string) |  | The url link to the e.g., GitHub commit or pull request. |






<a name="bytebase-v1-UndeleteReleaseRequest"></a>

### UndeleteReleaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the deleted release. Format: projects/{project}/releases/{release} |






<a name="bytebase-v1-UpdateReleaseRequest"></a>

### UpdateReleaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| release | [Release](#bytebase-v1-Release) |  | The release to update. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to be updated. |
| allow_missing | [bool](#bool) |  | If set to true, and the release is not found, a new release will be created. In this situation, `update_mask` is ignored. |





 


<a name="bytebase-v1-Release-Type"></a>

### Release.Type
The type of schema change.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 | Unspecified type. |
| VERSIONED | 1 | Versioned schema migration. |
| DECLARATIVE | 2 | Declarative schema definition. |


 

 


<a name="bytebase-v1-ReleaseService"></a>

### ReleaseService
ReleaseService manages releases for coordinating deployments.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetRelease | [GetReleaseRequest](#bytebase-v1-GetReleaseRequest) | [Release](#bytebase-v1-Release) | Retrieves a release by name. Permissions required: bb.releases.get |
| ListReleases | [ListReleasesRequest](#bytebase-v1-ListReleasesRequest) | [ListReleasesResponse](#bytebase-v1-ListReleasesResponse) | Lists releases in a project. Permissions required: bb.releases.list |
| CreateRelease | [CreateReleaseRequest](#bytebase-v1-CreateReleaseRequest) | [Release](#bytebase-v1-Release) | Creates a new release with SQL files. Permissions required: bb.releases.create |
| UpdateRelease | [UpdateReleaseRequest](#bytebase-v1-UpdateReleaseRequest) | [Release](#bytebase-v1-Release) | Updates an existing release. Permissions required: bb.releases.update When allow_missing=true, also requires: bb.releases.create |
| DeleteRelease | [DeleteReleaseRequest](#bytebase-v1-DeleteReleaseRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Deletes a release. Permissions required: bb.releases.delete |
| UndeleteRelease | [UndeleteReleaseRequest](#bytebase-v1-UndeleteReleaseRequest) | [Release](#bytebase-v1-Release) | Restores a deleted release. Permissions required: bb.releases.undelete |
| CheckRelease | [CheckReleaseRequest](#bytebase-v1-CheckReleaseRequest) | [CheckReleaseResponse](#bytebase-v1-CheckReleaseResponse) | Validates a release by dry-running checks on target databases. Permissions required: bb.releases.check |

 



<a name="v1_review_config_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/review_config_service.proto



<a name="bytebase-v1-CreateReviewConfigRequest"></a>

### CreateReviewConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| review_config | [ReviewConfig](#bytebase-v1-ReviewConfig) |  | The SQL review config to create. |






<a name="bytebase-v1-DeleteReviewConfigRequest"></a>

### DeleteReviewConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the SQL review config to delete. Format: reviewConfigs/{reviewConfig} |






<a name="bytebase-v1-GetReviewConfigRequest"></a>

### GetReviewConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the SQL review config to retrieve. Format: reviewConfigs/{reviewConfig} |






<a name="bytebase-v1-ListReviewConfigsRequest"></a>

### ListReviewConfigsRequest







<a name="bytebase-v1-ListReviewConfigsResponse"></a>

### ListReviewConfigsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| review_configs | [ReviewConfig](#bytebase-v1-ReviewConfig) | repeated | The SQL review configs from the specified request. |






<a name="bytebase-v1-ReviewConfig"></a>

### ReviewConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the SQL review config. Format: reviewConfigs/{reviewConfig} |
| title | [string](#string) |  | The title of the review configuration. |
| enabled | [bool](#bool) |  | Whether the review configuration is enabled. |
| rules | [SQLReviewRule](#bytebase-v1-SQLReviewRule) | repeated | The SQL review rules to enforce. |
| resources | [string](#string) | repeated | Resources using the config. Format: {resource}/{resource id}, e.g., environments/test. |






<a name="bytebase-v1-SQLReviewRule"></a>

### SQLReviewRule
SQL review rule configuration. Check the SQL_REVIEW_RULES_DOCUMENTATION.md for details.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [SQLReviewRule.Type](#bytebase-v1-SQLReviewRule-Type) |  | The type of SQL review rule. |
| level | [SQLReviewRule.Level](#bytebase-v1-SQLReviewRule-Level) |  | The severity level of the rule. |
| naming_payload | [SQLReviewRule.NamingRulePayload](#bytebase-v1-SQLReviewRule-NamingRulePayload) |  |  |
| number_payload | [SQLReviewRule.NumberRulePayload](#bytebase-v1-SQLReviewRule-NumberRulePayload) |  |  |
| string_array_payload | [SQLReviewRule.StringArrayRulePayload](#bytebase-v1-SQLReviewRule-StringArrayRulePayload) |  |  |
| comment_convention_payload | [SQLReviewRule.CommentConventionRulePayload](#bytebase-v1-SQLReviewRule-CommentConventionRulePayload) |  |  |
| string_payload | [SQLReviewRule.StringRulePayload](#bytebase-v1-SQLReviewRule-StringRulePayload) |  |  |
| naming_case_payload | [SQLReviewRule.NamingCaseRulePayload](#bytebase-v1-SQLReviewRule-NamingCaseRulePayload) |  |  |
| engine | [Engine](#bytebase-v1-Engine) |  | The database engine this rule applies to. |






<a name="bytebase-v1-SQLReviewRule-CommentConventionRulePayload"></a>

### SQLReviewRule.CommentConventionRulePayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| required | [bool](#bool) |  |  |
| max_length | [int32](#int32) |  |  |






<a name="bytebase-v1-SQLReviewRule-NamingCaseRulePayload"></a>

### SQLReviewRule.NamingCaseRulePayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| upper | [bool](#bool) |  |  |






<a name="bytebase-v1-SQLReviewRule-NamingRulePayload"></a>

### SQLReviewRule.NamingRulePayload
Payload message types for SQL review rules


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| max_length | [int32](#int32) |  |  |
| format | [string](#string) |  |  |






<a name="bytebase-v1-SQLReviewRule-NumberRulePayload"></a>

### SQLReviewRule.NumberRulePayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| number | [int32](#int32) |  |  |






<a name="bytebase-v1-SQLReviewRule-StringArrayRulePayload"></a>

### SQLReviewRule.StringArrayRulePayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| list | [string](#string) | repeated |  |






<a name="bytebase-v1-SQLReviewRule-StringRulePayload"></a>

### SQLReviewRule.StringRulePayload



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [string](#string) |  |  |






<a name="bytebase-v1-UpdateReviewConfigRequest"></a>

### UpdateReviewConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| review_config | [ReviewConfig](#bytebase-v1-ReviewConfig) |  | The SQL review config to update.

The name field is used to identify the SQL review config to update. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the config is not found, a new config will be created. In this situation, `update_mask` is ignored. |





 


<a name="bytebase-v1-SQLReviewRule-Level"></a>

### SQLReviewRule.Level
The severity level for SQL review rules.

| Name | Number | Description |
| ---- | ------ | ----------- |
| LEVEL_UNSPECIFIED | 0 | Unspecified level. |
| ERROR | 1 | Rule violation is an error. |
| WARNING | 2 | Rule violation is a warning. |



<a name="bytebase-v1-SQLReviewRule-Type"></a>

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


 

 


<a name="bytebase-v1-ReviewConfigService"></a>

### ReviewConfigService
ReviewConfigService manages approval flow configurations.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateReviewConfig | [CreateReviewConfigRequest](#bytebase-v1-CreateReviewConfigRequest) | [ReviewConfig](#bytebase-v1-ReviewConfig) | Creates a new SQL review configuration. Permissions required: bb.reviewConfigs.create |
| ListReviewConfigs | [ListReviewConfigsRequest](#bytebase-v1-ListReviewConfigsRequest) | [ListReviewConfigsResponse](#bytebase-v1-ListReviewConfigsResponse) | Lists all SQL review configurations. Permissions required: bb.reviewConfigs.list |
| GetReviewConfig | [GetReviewConfigRequest](#bytebase-v1-GetReviewConfigRequest) | [ReviewConfig](#bytebase-v1-ReviewConfig) | Retrieves a SQL review configuration by name. Permissions required: bb.reviewConfigs.get |
| UpdateReviewConfig | [UpdateReviewConfigRequest](#bytebase-v1-UpdateReviewConfigRequest) | [ReviewConfig](#bytebase-v1-ReviewConfig) | Updates a SQL review configuration. Permissions required: bb.reviewConfigs.update When allow_missing=true, also requires: bb.reviewConfigs.create |
| DeleteReviewConfig | [DeleteReviewConfigRequest](#bytebase-v1-DeleteReviewConfigRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Deletes a SQL review configuration. Permissions required: bb.reviewConfigs.delete |

 



<a name="v1_revision_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/revision_service.proto



<a name="bytebase-v1-BatchCreateRevisionsRequest"></a>

### BatchCreateRevisionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource shared by all revisions being created. Format: instances/{instance}/databases/{database} |
| requests | [CreateRevisionRequest](#bytebase-v1-CreateRevisionRequest) | repeated | The request message specifying the revisions to create. A maximum of 100 revisions can be created in a batch. |






<a name="bytebase-v1-BatchCreateRevisionsResponse"></a>

### BatchCreateRevisionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revisions | [Revision](#bytebase-v1-Revision) | repeated | The created revisions. |






<a name="bytebase-v1-CreateRevisionRequest"></a>

### CreateRevisionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: instances/{instance}/databases/{database} |
| revision | [Revision](#bytebase-v1-Revision) |  | The revision to create. |






<a name="bytebase-v1-DeleteRevisionRequest"></a>

### DeleteRevisionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the revision to delete. Format: instances/{instance}/databases/{database}/revisions/{revision} |






<a name="bytebase-v1-GetRevisionRequest"></a>

### GetRevisionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the revision. Format: instances/{instance}/databases/{database}/revisions/{revision} |






<a name="bytebase-v1-ListRevisionsRequest"></a>

### ListRevisionsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent of the revisions. Format: instances/{instance}/databases/{database} |
| page_size | [int32](#int32) |  | The maximum number of revisions to return. The service may return fewer than this value. If unspecified, at most 10 revisions will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListRevisions` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListRevisions` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Whether to include deleted revisions in the results. |






<a name="bytebase-v1-ListRevisionsResponse"></a>

### ListRevisionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revisions | [Revision](#bytebase-v1-Revision) | repeated | The revisions from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-Revision"></a>

### Revision



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: instances/{instance}/databases/{database}/revisions/{revision} |
| release | [string](#string) |  | Format: projects/{project}/releases/{release} Can be empty. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| deleter | [string](#string) |  | Format: users/hello@world.com Can be empty. |
| delete_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | Can be empty. |
| file | [string](#string) |  | Format: projects/{project}/releases/{release}/files/{id} Can be empty. |
| version | [string](#string) |  | The schema version string for this revision. |
| sheet | [string](#string) |  | The sheet that holds the content. Format: projects/{project}/sheets/{sheet} |
| sheet_sha256 | [string](#string) |  | The SHA256 hash value of the sheet. |
| task_run | [string](#string) |  | The task run associated with the revision. Can be empty. Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |
| type | [Revision.Type](#bytebase-v1-Revision-Type) |  | The type of the revision. |





 


<a name="bytebase-v1-Revision-Type"></a>

### Revision.Type
The type of schema revision.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 | Unspecified type. |
| VERSIONED | 1 | Versioned schema migration. |
| DECLARATIVE | 2 | Declarative schema definition. |


 

 


<a name="bytebase-v1-RevisionService"></a>

### RevisionService
RevisionService manages schema revision history.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListRevisions | [ListRevisionsRequest](#bytebase-v1-ListRevisionsRequest) | [ListRevisionsResponse](#bytebase-v1-ListRevisionsResponse) | Lists schema revisions for a database. Permissions required: bb.revisions.list |
| GetRevision | [GetRevisionRequest](#bytebase-v1-GetRevisionRequest) | [Revision](#bytebase-v1-Revision) | Retrieves a schema revision by name. Permissions required: bb.revisions.get |
| BatchCreateRevisions | [BatchCreateRevisionsRequest](#bytebase-v1-BatchCreateRevisionsRequest) | [BatchCreateRevisionsResponse](#bytebase-v1-BatchCreateRevisionsResponse) | Creates multiple schema revisions in a single operation. Permissions required: bb.revisions.create |
| DeleteRevision | [DeleteRevisionRequest](#bytebase-v1-DeleteRevisionRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Deletes a schema revision. Permissions required: bb.revisions.delete |

 



<a name="v1_role_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/role_service.proto



<a name="bytebase-v1-CreateRoleRequest"></a>

### CreateRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [Role](#bytebase-v1-Role) |  | The role to create. |
| role_id | [string](#string) |  | The ID to use for the role, which will become the final component of the role&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][A-Z][0-9]/. |






<a name="bytebase-v1-DeleteRoleRequest"></a>

### DeleteRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the role to delete. Format: roles/{role} |






<a name="bytebase-v1-GetRoleRequest"></a>

### GetRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the role to retrieve. Format: roles/{role} |






<a name="bytebase-v1-ListRolesRequest"></a>

### ListRolesRequest







<a name="bytebase-v1-ListRolesResponse"></a>

### ListRolesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| roles | [Role](#bytebase-v1-Role) | repeated | The roles from the specified request. |






<a name="bytebase-v1-Role"></a>

### Role
Role defines a set of permissions that can be assigned to users.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Resource name. Format: roles/{role} |
| title | [string](#string) |  | Human-readable title. |
| description | [string](#string) |  | Optional description of the role. |
| permissions | [string](#string) | repeated | List of permission identifiers granted by this role. |
| type | [Role.Type](#bytebase-v1-Role-Type) |  | Role type indicating if it&#39;s built-in or custom. |






<a name="bytebase-v1-UpdateRoleRequest"></a>

### UpdateRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [Role](#bytebase-v1-Role) |  | The role to update. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the role is not found, a new role will be created. In this situation, `update_mask` is ignored. |





 


<a name="bytebase-v1-Role-Type"></a>

### Role.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| BUILT_IN | 1 | System-defined role that cannot be modified. |
| CUSTOM | 2 | User-defined role that can be modified. |


 

 


<a name="bytebase-v1-RoleService"></a>

### RoleService
RoleService manages workspace roles and permissions.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListRoles | [ListRolesRequest](#bytebase-v1-ListRolesRequest) | [ListRolesResponse](#bytebase-v1-ListRolesResponse) | Lists roles in the workspace. Permissions required: bb.roles.list |
| GetRole | [GetRoleRequest](#bytebase-v1-GetRoleRequest) | [Role](#bytebase-v1-Role) | Retrieves a role by name. Permissions required: bb.roles.get |
| CreateRole | [CreateRoleRequest](#bytebase-v1-CreateRoleRequest) | [Role](#bytebase-v1-Role) | Creates a new custom role. Permissions required: bb.roles.create |
| UpdateRole | [UpdateRoleRequest](#bytebase-v1-UpdateRoleRequest) | [Role](#bytebase-v1-Role) | Updates a role&#39;s properties. Permissions required: bb.roles.update When allow_missing=true, also requires: bb.roles.create |
| DeleteRole | [DeleteRoleRequest](#bytebase-v1-DeleteRoleRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Deletes a custom role. Permissions required: bb.roles.delete |

 



<a name="v1_rollout_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/rollout_service.proto



<a name="bytebase-v1-BatchCancelTaskRunsRequest"></a>

### BatchCancelTaskRunsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The task name for the taskRuns. Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task} Use `projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/-` to cancel task runs under the same stage. |
| task_runs | [string](#string) | repeated | The taskRuns to cancel. Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |






<a name="bytebase-v1-BatchCancelTaskRunsResponse"></a>

### BatchCancelTaskRunsResponse







<a name="bytebase-v1-BatchRunTasksRequest"></a>

### BatchRunTasksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The stage name for the tasks. Format: projects/{project}/plans/{plan}/rollout/stages/{stage} |
| tasks | [string](#string) | repeated | The tasks to run. Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task} |
| run_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional | The task run should run after run_time. |






<a name="bytebase-v1-BatchRunTasksResponse"></a>

### BatchRunTasksResponse







<a name="bytebase-v1-BatchSkipTasksRequest"></a>

### BatchSkipTasksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The stage name for the tasks. Format: projects/{project}/plans/{plan}/rollout/stages/{stage} |
| tasks | [string](#string) | repeated | The tasks to skip. Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task} |
| reason | [string](#string) |  | The reason for skipping the tasks. |






<a name="bytebase-v1-BatchSkipTasksResponse"></a>

### BatchSkipTasksResponse







<a name="bytebase-v1-CreateRolloutRequest"></a>

### CreateRolloutRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent plan for which this rollout will be created. Format: projects/{project}/plans/{plan} |
| target | [string](#string) | optional | Create the rollout only for the specified target. Format: environments/{environment} If unspecified, all stages are created. If set to &#34;&#34;, no stages are created. |






<a name="bytebase-v1-GetRolloutRequest"></a>

### GetRolloutRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the rollout to retrieve. This is the rollout resource name, which is the plan name plus /rollout suffix. Format: projects/{project}/plans/{plan}/rollout |






<a name="bytebase-v1-GetTaskRunLogRequest"></a>

### GetTaskRunLogRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |






<a name="bytebase-v1-GetTaskRunRequest"></a>

### GetTaskRunRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |






<a name="bytebase-v1-GetTaskRunSessionRequest"></a>

### GetTaskRunSessionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |






<a name="bytebase-v1-ListRolloutsRequest"></a>

### ListRolloutsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of rollouts. Format: projects/{project} Use &#34;projects/-&#34; to list all rollouts from all projects. |
| page_size | [int32](#int32) |  | The maximum number of rollouts to return. The service may return fewer than this value. If unspecified, at most 10 rollouts will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListRollouts` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListRollouts` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter rollouts returned in the list. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filters: - update_time: rollout update time in &#34;2006-01-02T15:04:05Z07:00&#34; format, support &#34;&gt;=&#34; or &#34;&lt;=&#34; operator. - task_type: the task type, support &#34;in&#34; operator, check the Task.Type enum for the values.

For example: update_time &gt;= &#34;2025-01-02T15:04:05Z07:00&#34; task_type in [&#34;DATABASE_MIGRATE&#34;, &#34;DATABASE_EXPORT&#34;] |






<a name="bytebase-v1-ListRolloutsResponse"></a>

### ListRolloutsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| rollouts | [Rollout](#bytebase-v1-Rollout) | repeated | The rollouts from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-ListTaskRunsRequest"></a>

### ListTaskRunsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of taskRuns. Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task} Use &#34;projects/{project}/plans/{plan}/rollout/stages/-/tasks/-&#34; to list all taskRuns from a rollout. |






<a name="bytebase-v1-ListTaskRunsResponse"></a>

### ListTaskRunsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| task_runs | [TaskRun](#bytebase-v1-TaskRun) | repeated | The taskRuns from the specified request. |






<a name="bytebase-v1-PreviewTaskRunRollbackRequest"></a>

### PreviewTaskRunRollbackRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |






<a name="bytebase-v1-PreviewTaskRunRollbackResponse"></a>

### PreviewTaskRunRollbackResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| statement | [string](#string) |  | The rollback SQL statement that would undo the task run. |






<a name="bytebase-v1-Rollout"></a>

### Rollout



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The resource name of the rollout. Format: projects/{project}/plans/{plan}/rollout |
| title | [string](#string) |  | The title of the rollout, inherited from the associated plan. This field is output only and cannot be directly set. |
| stages | [Stage](#bytebase-v1-Stage) | repeated | Stages and thus tasks of the rollout. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |






<a name="bytebase-v1-Stage"></a>

### Stage



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/plans/{plan}/rollout/stages/{stage} Use &#34;-&#34; for {stage} when the stage has no environment or deleted environment. |
| id | [string](#string) |  | id is the environment id of the stage. e.g., &#34;prod&#34;. Use &#34;-&#34; when the stage has no environment or deleted environment. |
| environment | [string](#string) |  | environment is the environment of the stage. Format: environments/{environment} for valid environments, or &#34;environments/-&#34; for stages without environment or with deleted environments. |
| tasks | [Task](#bytebase-v1-Task) | repeated | The tasks within this stage. |






<a name="bytebase-v1-Task"></a>

### Task



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task} |
| spec_id | [string](#string) |  | A UUID4 string that uniquely identifies the Spec. |
| status | [Task.Status](#bytebase-v1-Task-Status) |  | Status is the status of the task. |
| skipped_reason | [string](#string) |  | The reason why the task was skipped. |
| type | [Task.Type](#bytebase-v1-Task-Type) |  |  |
| target | [string](#string) |  | Format: instances/{instance} if the task is DatabaseCreate. Format: instances/{instance}/databases/{database} |
| database_create | [Task.DatabaseCreate](#bytebase-v1-Task-DatabaseCreate) |  |  |
| database_update | [Task.DatabaseUpdate](#bytebase-v1-Task-DatabaseUpdate) |  |  |
| database_data_export | [Task.DatabaseDataExport](#bytebase-v1-Task-DatabaseDataExport) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional | The update_time is the update time of latest task run. If there are no task runs, it will be empty. |
| run_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional | The run_time is the scheduled run time of latest task run. If there are no task runs or the task run is not scheduled, it will be empty. |






<a name="bytebase-v1-Task-DatabaseCreate"></a>

### Task.DatabaseCreate
Payload for creating a new database.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheet | [string](#string) |  | Format: projects/{project}/sheets/{sheet} |






<a name="bytebase-v1-Task-DatabaseDataExport"></a>

### Task.DatabaseDataExport
Payload for exporting database data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheet | [string](#string) |  | The resource name of the sheet. Format: projects/{project}/sheets/{sheet} |






<a name="bytebase-v1-Task-DatabaseUpdate"></a>

### Task.DatabaseUpdate
Payload for updating a database schema.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheet | [string](#string) |  | Format: projects/{project}/sheets/{sheet} |
| release | [string](#string) |  | Format: projects/{project}/releases/{release} |






<a name="bytebase-v1-TaskRun"></a>

### TaskRun



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |
| creator | [string](#string) |  | Format: users/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| status | [TaskRun.Status](#bytebase-v1-TaskRun-Status) |  | The current execution status of the task run. |
| detail | [string](#string) |  | Below are the results of a task run. Detailed information about the task run result. |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The time when the task run started execution. |
| export_archive_status | [TaskRun.ExportArchiveStatus](#bytebase-v1-TaskRun-ExportArchiveStatus) |  | The export archive status for data export tasks. |
| has_prior_backup | [bool](#bool) |  | Indicates whether a prior backup was created for this task run. When true, rollback SQL can be generated via PreviewTaskRunRollback. Backup details are available in the task run logs. |
| scheduler_info | [TaskRun.SchedulerInfo](#bytebase-v1-TaskRun-SchedulerInfo) |  | Scheduling information about the task run. |
| run_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional | The task run should run after run_time. This can only be set when creating the task run calling BatchRunTasks. |






<a name="bytebase-v1-TaskRun-SchedulerInfo"></a>

### TaskRun.SchedulerInfo
Information about task run scheduling.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| report_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The time when the scheduling info was reported. |
| waiting_cause | [TaskRun.SchedulerInfo.WaitingCause](#bytebase-v1-TaskRun-SchedulerInfo-WaitingCause) |  | The cause for the task run waiting. |






<a name="bytebase-v1-TaskRun-SchedulerInfo-WaitingCause"></a>

### TaskRun.SchedulerInfo.WaitingCause
Information about why a task run is waiting.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parallel_tasks_limit | [bool](#bool) |  | Waiting due to parallel tasks limit. |






<a name="bytebase-v1-TaskRunLog"></a>

### TaskRunLog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun}/log |
| entries | [TaskRunLogEntry](#bytebase-v1-TaskRunLogEntry) | repeated | The log entries for this task run. |






<a name="bytebase-v1-TaskRunLogEntry"></a>

### TaskRunLogEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [TaskRunLogEntry.Type](#bytebase-v1-TaskRunLogEntry-Type) |  | The type of this log entry. |
| log_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The time when the log was recorded. |
| deploy_id | [string](#string) |  | The deployment ID for this log entry. |
| schema_dump | [TaskRunLogEntry.SchemaDump](#bytebase-v1-TaskRunLogEntry-SchemaDump) |  | Schema dump details (if type is SCHEMA_DUMP). |
| command_execute | [TaskRunLogEntry.CommandExecute](#bytebase-v1-TaskRunLogEntry-CommandExecute) |  | Command execution details (if type is COMMAND_EXECUTE). |
| database_sync | [TaskRunLogEntry.DatabaseSync](#bytebase-v1-TaskRunLogEntry-DatabaseSync) |  | Database sync details (if type is DATABASE_SYNC). |
| transaction_control | [TaskRunLogEntry.TransactionControl](#bytebase-v1-TaskRunLogEntry-TransactionControl) |  | Transaction control details (if type is TRANSACTION_CONTROL). |
| prior_backup | [TaskRunLogEntry.PriorBackup](#bytebase-v1-TaskRunLogEntry-PriorBackup) |  | Prior backup details (if type is PRIOR_BACKUP). |
| retry_info | [TaskRunLogEntry.RetryInfo](#bytebase-v1-TaskRunLogEntry-RetryInfo) |  | Retry information details (if type is RETRY_INFO). |
| compute_diff | [TaskRunLogEntry.ComputeDiff](#bytebase-v1-TaskRunLogEntry-ComputeDiff) |  | Compute diff details (if type is COMPUTE_DIFF). |
| release_file_execute | [TaskRunLogEntry.ReleaseFileExecute](#bytebase-v1-TaskRunLogEntry-ReleaseFileExecute) |  | Release file execution details (if type is RELEASE_FILE_EXECUTE). |






<a name="bytebase-v1-TaskRunLogEntry-CommandExecute"></a>

### TaskRunLogEntry.CommandExecute
Command execution details.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| log_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When the command was logged. |
| range | [Range](#bytebase-v1-Range) |  | The byte offset range of the executed command in the sheet. Uses byte offsets (not character indices) for efficient slicing of sheet content bytes. Example: For &#34;SELECT 你好;&#34; in a UTF-8 sheet, range [0, 13) represents all 13 bytes. |
| statement | [string](#string) |  | The executed statement. |
| response | [TaskRunLogEntry.CommandExecute.CommandResponse](#bytebase-v1-TaskRunLogEntry-CommandExecute-CommandResponse) |  | The response from executing the command. |






<a name="bytebase-v1-TaskRunLogEntry-CommandExecute-CommandResponse"></a>

### TaskRunLogEntry.CommandExecute.CommandResponse
Command execution response.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| log_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When the response was logged. |
| error | [string](#string) |  | Error message if command execution failed. |
| affected_rows | [int64](#int64) |  | Total affected rows. |
| all_affected_rows | [int64](#int64) | repeated | `all_affected_rows` is the affected rows of each command. `all_affected_rows` may be unavailable if the database driver doesn&#39;t support it. Caller should fallback to `affected_rows` in that case. |






<a name="bytebase-v1-TaskRunLogEntry-ComputeDiff"></a>

### TaskRunLogEntry.ComputeDiff
Schema diff computation details.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When diff computation started. |
| end_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When diff computation ended. |
| error | [string](#string) |  | Error message if computation failed. |






<a name="bytebase-v1-TaskRunLogEntry-DatabaseSync"></a>

### TaskRunLogEntry.DatabaseSync
Database synchronization details.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When the database sync started. |
| end_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When the database sync ended. |
| error | [string](#string) |  | Error message if sync failed. |






<a name="bytebase-v1-TaskRunLogEntry-PriorBackup"></a>

### TaskRunLogEntry.PriorBackup
Prior backup operation details.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When the backup started. |
| end_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When the backup ended. |
| prior_backup_detail | [TaskRunLogEntry.PriorBackup.PriorBackupDetail](#bytebase-v1-TaskRunLogEntry-PriorBackup-PriorBackupDetail) |  | The backup details. |
| error | [string](#string) |  | Error message if the backup failed. |






<a name="bytebase-v1-TaskRunLogEntry-PriorBackup-PriorBackupDetail"></a>

### TaskRunLogEntry.PriorBackup.PriorBackupDetail
Prior backup detail for rollback purposes.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| items | [TaskRunLogEntry.PriorBackup.PriorBackupDetail.Item](#bytebase-v1-TaskRunLogEntry-PriorBackup-PriorBackupDetail-Item) | repeated | The list of backed up tables. |






<a name="bytebase-v1-TaskRunLogEntry-PriorBackup-PriorBackupDetail-Item"></a>

### TaskRunLogEntry.PriorBackup.PriorBackupDetail.Item
A single backup table mapping.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| source_table | [TaskRunLogEntry.PriorBackup.PriorBackupDetail.Item.Table](#bytebase-v1-TaskRunLogEntry-PriorBackup-PriorBackupDetail-Item-Table) |  | The original table information. |
| target_table | [TaskRunLogEntry.PriorBackup.PriorBackupDetail.Item.Table](#bytebase-v1-TaskRunLogEntry-PriorBackup-PriorBackupDetail-Item-Table) |  | The target backup table information. |
| start_position | [Position](#bytebase-v1-Position) |  | The start position in the SQL statement. |
| end_position | [Position](#bytebase-v1-Position) |  | The end position in the SQL statement. |






<a name="bytebase-v1-TaskRunLogEntry-PriorBackup-PriorBackupDetail-Item-Table"></a>

### TaskRunLogEntry.PriorBackup.PriorBackupDetail.Item.Table
Table information.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [string](#string) |  | The database information. Format: instances/{instance}/databases/{database} |
| schema | [string](#string) |  | The schema name. |
| table | [string](#string) |  | The table name. |






<a name="bytebase-v1-TaskRunLogEntry-ReleaseFileExecute"></a>

### TaskRunLogEntry.ReleaseFileExecute
Release file execution details.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [string](#string) |  | The version of the file being executed (e.g., &#34;0001&#34;). |
| file_path | [string](#string) |  | The file path within the release (e.g., &#34;2.2/V0001_create_table.sql&#34;). |






<a name="bytebase-v1-TaskRunLogEntry-RetryInfo"></a>

### TaskRunLogEntry.RetryInfo
Retry information for failed operations.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| error | [string](#string) |  | The error that triggered the retry. |
| retry_count | [int32](#int32) |  | Current retry attempt number. |
| maximum_retries | [int32](#int32) |  | Maximum number of retries allowed. |






<a name="bytebase-v1-TaskRunLogEntry-SchemaDump"></a>

### TaskRunLogEntry.SchemaDump
Schema dump operation details.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When the schema dump started. |
| end_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When the schema dump ended. |
| error | [string](#string) |  | Error message if the schema dump failed. |






<a name="bytebase-v1-TaskRunLogEntry-TransactionControl"></a>

### TaskRunLogEntry.TransactionControl
Transaction control operation details.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [TaskRunLogEntry.TransactionControl.Type](#bytebase-v1-TaskRunLogEntry-TransactionControl-Type) |  | The type of transaction control. |
| error | [string](#string) |  | Error message if the operation failed. |






<a name="bytebase-v1-TaskRunSession"></a>

### TaskRunSession



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun}/session |
| postgres | [TaskRunSession.Postgres](#bytebase-v1-TaskRunSession-Postgres) |  | PostgreSQL session information. |






<a name="bytebase-v1-TaskRunSession-Postgres"></a>

### TaskRunSession.Postgres
PostgreSQL session information.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| session | [TaskRunSession.Postgres.Session](#bytebase-v1-TaskRunSession-Postgres-Session) |  | `session` is the session of the task run executing commands. |
| blocking_sessions | [TaskRunSession.Postgres.Session](#bytebase-v1-TaskRunSession-Postgres-Session) | repeated | `blocking_sessions` block `session`. |
| blocked_sessions | [TaskRunSession.Postgres.Session](#bytebase-v1-TaskRunSession-Postgres-Session) | repeated | `blocked_sessions` are blocked by `session`. |






<a name="bytebase-v1-TaskRunSession-Postgres-Session"></a>

### TaskRunSession.Postgres.Session
PostgreSQL session information read from `pg_stat_activity`.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| pid | [string](#string) |  | Process ID of the session. |
| blocked_by_pids | [string](#string) | repeated | PIDs of sessions blocking this session. |
| query | [string](#string) |  | Current query being executed. |
| state | [string](#string) | optional | Session state (active, idle, etc.). |
| wait_event_type | [string](#string) | optional | Wait event type if session is waiting. |
| wait_event | [string](#string) | optional | Specific wait event if session is waiting. |
| datname | [string](#string) | optional | Database name. |
| usename | [string](#string) | optional | User name. |
| application_name | [string](#string) |  | Application name. |
| client_addr | [string](#string) | optional | Client IP address. |
| client_port | [string](#string) | optional | Client port number. |
| backend_start | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | When the backend process started. |
| xact_start | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional | When the current transaction started. |
| query_start | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional | When the current query started. |





 


<a name="bytebase-v1-Task-Status"></a>

### Task.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 | Unspecified status. |
| NOT_STARTED | 1 | Task has not started and is awaiting its turn in the deployment sequence. |
| PENDING | 2 | Task is ready to run and waiting for approval or scheduled time. |
| RUNNING | 3 | Task is currently executing. |
| DONE | 4 | Task completed successfully. |
| FAILED | 5 | Task execution failed with errors. |
| CANCELED | 6 | Task execution was canceled by a user. |
| SKIPPED | 7 | Task was skipped and will not be executed. |



<a name="bytebase-v1-Task-Type"></a>

### Task.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 | Unspecified task type. |
| GENERAL | 1 | General task for miscellaneous operations. |
| DATABASE_CREATE | 2 | Database creation task that creates a new database. Use payload DatabaseCreate. |
| DATABASE_MIGRATE | 3 | Database migration task that applies versioned schema changes. Use payload DatabaseUpdate. |
| DATABASE_EXPORT | 4 | Database export task that exports query results or table data. Use payload DatabaseDataExport. |



<a name="bytebase-v1-TaskRun-ExportArchiveStatus"></a>

### TaskRun.ExportArchiveStatus


| Name | Number | Description |
| ---- | ------ | ----------- |
| EXPORT_ARCHIVE_STATUS_UNSPECIFIED | 0 | Unspecified export archive status. |
| READY | 1 | Export archive is ready for download. |
| EXPORTED | 2 | Export archive has been downloaded by the user. |



<a name="bytebase-v1-TaskRun-Status"></a>

### TaskRun.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| STATUS_UNSPECIFIED | 0 | Unspecified status. |
| PENDING | 1 | Task run is queued and waiting to start execution. |
| RUNNING | 2 | Task run is currently executing. |
| DONE | 3 | Task run completed successfully. |
| FAILED | 4 | Task run execution failed with errors. |
| CANCELED | 5 | Task run execution was canceled by a user. |
| AVAILABLE | 6 | Task run is ready for immediate execution. |



<a name="bytebase-v1-TaskRunLogEntry-TransactionControl-Type"></a>

### TaskRunLogEntry.TransactionControl.Type
Transaction control type.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 | Unspecified type. |
| BEGIN | 1 | Begin transaction. |
| COMMIT | 2 | Commit transaction. |
| ROLLBACK | 3 | Rollback transaction. |



<a name="bytebase-v1-TaskRunLogEntry-Type"></a>

### TaskRunLogEntry.Type
The type of log entry.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 | Unspecified type. |
| SCHEMA_DUMP | 1 | Schema dump operation. |
| COMMAND_EXECUTE | 2 | Command execution. |
| DATABASE_SYNC | 3 | Database synchronization. |
| TRANSACTION_CONTROL | 5 | Transaction control (BEGIN, COMMIT, ROLLBACK). |
| PRIOR_BACKUP | 6 | Prior backup operation. |
| RETRY_INFO | 7 | Retry information. |
| COMPUTE_DIFF | 8 | Schema diff computation. |
| RELEASE_FILE_EXECUTE | 9 | Release file execution. |


 

 


<a name="bytebase-v1-RolloutService"></a>

### RolloutService
RolloutService manages the execution of deployment plans.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetRollout | [GetRolloutRequest](#bytebase-v1-GetRolloutRequest) | [Rollout](#bytebase-v1-Rollout) | Retrieves a rollout by its plan name. Permissions required: bb.rollouts.get |
| ListRollouts | [ListRolloutsRequest](#bytebase-v1-ListRolloutsRequest) | [ListRolloutsResponse](#bytebase-v1-ListRolloutsResponse) | Lists rollouts in a project. Permissions required: bb.rollouts.list |
| CreateRollout | [CreateRolloutRequest](#bytebase-v1-CreateRolloutRequest) | [Rollout](#bytebase-v1-Rollout) | Creates a new rollout for a plan. Permissions required: bb.rollouts.create |
| ListTaskRuns | [ListTaskRunsRequest](#bytebase-v1-ListTaskRunsRequest) | [ListTaskRunsResponse](#bytebase-v1-ListTaskRunsResponse) | Lists task run executions for a task. Permissions required: bb.taskRuns.list |
| GetTaskRun | [GetTaskRunRequest](#bytebase-v1-GetTaskRunRequest) | [TaskRun](#bytebase-v1-TaskRun) | Retrieves a task run by name. Permissions required: bb.taskRuns.list |
| GetTaskRunLog | [GetTaskRunLogRequest](#bytebase-v1-GetTaskRunLogRequest) | [TaskRunLog](#bytebase-v1-TaskRunLog) | Retrieves execution logs for a task run. Permissions required: bb.taskRuns.list |
| GetTaskRunSession | [GetTaskRunSessionRequest](#bytebase-v1-GetTaskRunSessionRequest) | [TaskRunSession](#bytebase-v1-TaskRunSession) | Retrieves database session information for a running task. Permissions required: bb.taskRuns.list |
| BatchRunTasks | [BatchRunTasksRequest](#bytebase-v1-BatchRunTasksRequest) | [BatchRunTasksResponse](#bytebase-v1-BatchRunTasksResponse) | Executes multiple tasks in a rollout stage. Permissions required: bb.taskRuns.create (or issue creator for data export issues, or user with rollout policy role for the environment) |
| BatchSkipTasks | [BatchSkipTasksRequest](#bytebase-v1-BatchSkipTasksRequest) | [BatchSkipTasksResponse](#bytebase-v1-BatchSkipTasksResponse) | Skips multiple tasks in a rollout stage. Permissions required: bb.taskRuns.create (or issue creator for data export issues, or user with rollout policy role for the environment) |
| BatchCancelTaskRuns | [BatchCancelTaskRunsRequest](#bytebase-v1-BatchCancelTaskRunsRequest) | [BatchCancelTaskRunsResponse](#bytebase-v1-BatchCancelTaskRunsResponse) | Cancels multiple running task executions. Permissions required: bb.taskRuns.create (or issue creator for data export issues, or user with rollout policy role for the environment) |
| PreviewTaskRunRollback | [PreviewTaskRunRollbackRequest](#bytebase-v1-PreviewTaskRunRollbackRequest) | [PreviewTaskRunRollbackResponse](#bytebase-v1-PreviewTaskRunRollbackResponse) | Generates rollback SQL for a completed task run. Permissions required: bb.taskRuns.list |

 



<a name="v1_sheet_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/sheet_service.proto



<a name="bytebase-v1-BatchCreateSheetsRequest"></a>

### BatchCreateSheetsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where all sheets will be created. Format: projects/{project} |
| requests | [CreateSheetRequest](#bytebase-v1-CreateSheetRequest) | repeated |  |






<a name="bytebase-v1-BatchCreateSheetsResponse"></a>

### BatchCreateSheetsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheets | [Sheet](#bytebase-v1-Sheet) | repeated |  |






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
| name | [string](#string) |  | The name of the sheet resource. Format: projects/{project}/sheets/{sheet} The sheet ID is generated by the server on creation and cannot be changed. |
| content | [bytes](#bytes) |  | The content of the sheet. By default, it will be cut off, if it doesn&#39;t match the `content_size`, you can set the `raw` to true in GetSheet request to retrieve the full content. |
| content_size | [int64](#int64) |  | content_size is the full size of the content, may not match the size of the `content` field. |





 

 

 


<a name="bytebase-v1-SheetService"></a>

### SheetService
SheetService manages SQL scripts and saved queries.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateSheet | [CreateSheetRequest](#bytebase-v1-CreateSheetRequest) | [Sheet](#bytebase-v1-Sheet) | Creates a new SQL sheet. Permissions required: bb.sheets.create |
| BatchCreateSheets | [BatchCreateSheetsRequest](#bytebase-v1-BatchCreateSheetsRequest) | [BatchCreateSheetsResponse](#bytebase-v1-BatchCreateSheetsResponse) | Creates multiple SQL sheets in a single operation. Permissions required: bb.sheets.create |
| GetSheet | [GetSheetRequest](#bytebase-v1-GetSheetRequest) | [Sheet](#bytebase-v1-Sheet) | Retrieves a SQL sheet by name. Permissions required: bb.sheets.get |

 



<a name="v1_subscription_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/subscription_service.proto



<a name="bytebase-v1-GetSubscriptionRequest"></a>

### GetSubscriptionRequest







<a name="bytebase-v1-PlanConfig"></a>

### PlanConfig
PlanConfig represents the configuration for all plans loaded from plan.yaml


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| plans | [PlanLimitConfig](#bytebase-v1-PlanLimitConfig) | repeated |  |
| instance_features | [PlanFeature](#bytebase-v1-PlanFeature) | repeated |  |






<a name="bytebase-v1-PlanLimitConfig"></a>

### PlanLimitConfig
PlanLimitConfig represents a single plan&#39;s configuration


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [PlanType](#bytebase-v1-PlanType) |  |  |
| maximum_instance_count | [int32](#int32) |  |  |
| maximum_seat_count | [int32](#int32) |  |  |
| features | [PlanFeature](#bytebase-v1-PlanFeature) | repeated |  |






<a name="bytebase-v1-Subscription"></a>

### Subscription



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| plan | [PlanType](#bytebase-v1-PlanType) |  |  |
| seats | [int32](#int32) |  |  |
| instances | [int32](#int32) |  |  |
| active_instances | [int32](#int32) |  |  |
| expires_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| trialing | [bool](#bool) |  |  |
| org_name | [string](#string) |  |  |






<a name="bytebase-v1-UpdateSubscriptionRequest"></a>

### UpdateSubscriptionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| license | [string](#string) |  |  |
| allow_missing | [bool](#bool) |  | If set to true, and the subscription is not found, a new subscription will be created. In this situation, `update_mask` is ignored. |





 


<a name="bytebase-v1-PlanFeature"></a>

### PlanFeature
PlanFeature represents the available features in Bytebase

| Name | Number | Description |
| ---- | ------ | ----------- |
| FEATURE_UNSPECIFIED | 0 |  |
| FEATURE_DATABASE_CHANGE | 1 | Database Change Management |
| FEATURE_GIT_BASED_SCHEMA_VERSION_CONTROL | 2 |  |
| FEATURE_DECLARATIVE_SCHEMA_MIGRATION | 3 |  |
| FEATURE_COMPARE_AND_SYNC_SCHEMA | 4 |  |
| FEATURE_ONLINE_SCHEMA_CHANGE | 5 |  |
| FEATURE_PRE_DEPLOYMENT_SQL_REVIEW | 6 |  |
| FEATURE_AUTOMATIC_BACKUP_BEFORE_DATA_CHANGES | 7 |  |
| FEATURE_ONE_CLICK_DATA_ROLLBACK | 8 |  |
| FEATURE_MULTI_DATABASE_BATCH_CHANGES | 9 |  |
| FEATURE_PROGRESSIVE_ENVIRONMENT_DEPLOYMENT | 10 |  |
| FEATURE_SCHEDULED_ROLLOUT_TIME | 11 |  |
| FEATURE_DATABASE_CHANGELOG | 12 |  |
| FEATURE_SCHEMA_DRIFT_DETECTION | 13 |  |
| FEATURE_ROLLOUT_POLICY | 14 |  |
| FEATURE_WEB_BASED_SQL_EDITOR | 15 | SQL Editor &amp; Development |
| FEATURE_SQL_EDITOR_ADMIN_MODE | 16 |  |
| FEATURE_NATURAL_LANGUAGE_TO_SQL | 17 |  |
| FEATURE_AI_QUERY_EXPLANATION | 18 |  |
| FEATURE_AI_QUERY_SUGGESTIONS | 19 |  |
| FEATURE_AUTO_COMPLETE | 20 |  |
| FEATURE_SCHEMA_DIAGRAM | 21 |  |
| FEATURE_SCHEMA_EDITOR | 22 |  |
| FEATURE_DATA_EXPORT | 23 |  |
| FEATURE_DATA_OFFLINE_EXPORT | 24 |  |
| FEATURE_QUERY_HISTORY | 25 |  |
| FEATURE_SAVED_AND_SHARED_SQL_SCRIPTS | 26 |  |
| FEATURE_BATCH_QUERY | 27 |  |
| FEATURE_INSTANCE_READ_ONLY_CONNECTION | 28 |  |
| FEATURE_QUERY_POLICY | 29 |  |
| FEATURE_RESTRICT_COPYING_DATA | 30 |  |
| FEATURE_IAM | 31 | Security &amp; Compliance |
| FEATURE_INSTANCE_SSL_CONNECTION | 32 |  |
| FEATURE_INSTANCE_CONNECTION_OVER_SSH_TUNNEL | 33 |  |
| FEATURE_INSTANCE_CONNECTION_IAM_AUTHENTICATION | 34 |  |
| FEATURE_GOOGLE_AND_GITHUB_SSO | 35 |  |
| FEATURE_USER_GROUPS | 36 |  |
| FEATURE_DISALLOW_SELF_SERVICE_SIGNUP | 37 |  |
| FEATURE_CUSTOM_INSTANCE_SYNC_TIME | 38 |  |
| FEATURE_CUSTOM_INSTANCE_CONNECTION_LIMIT | 39 |  |
| FEATURE_RISK_ASSESSMENT | 40 |  |
| FEATURE_APPROVAL_WORKFLOW | 41 |  |
| FEATURE_AUDIT_LOG | 42 |  |
| FEATURE_ENTERPRISE_SSO | 43 |  |
| FEATURE_TWO_FA | 44 |  |
| FEATURE_PASSWORD_RESTRICTIONS | 45 |  |
| FEATURE_DISALLOW_PASSWORD_SIGNIN | 46 |  |
| FEATURE_CUSTOM_ROLES | 47 |  |
| FEATURE_REQUEST_ROLE_WORKFLOW | 48 |  |
| FEATURE_DATA_MASKING | 49 |  |
| FEATURE_DATA_CLASSIFICATION | 50 |  |
| FEATURE_SCIM | 51 |  |
| FEATURE_DIRECTORY_SYNC | 52 |  |
| FEATURE_TOKEN_DURATION_CONTROL | 53 |  |
| FEATURE_EXTERNAL_SECRET_MANAGER | 54 |  |
| FEATURE_USER_EMAIL_DOMAIN_RESTRICTION | 55 |  |
| FEATURE_PROJECT_MANAGEMENT | 56 | Administration &amp; Support |
| FEATURE_ENVIRONMENT_MANAGEMENT | 57 |  |
| FEATURE_IM_NOTIFICATIONS | 58 |  |
| FEATURE_TERRAFORM_PROVIDER | 59 |  |
| FEATURE_DATABASE_GROUPS | 60 |  |
| FEATURE_ENVIRONMENT_TIERS | 61 |  |
| FEATURE_DASHBOARD_ANNOUNCEMENT | 62 |  |
| FEATURE_API_INTEGRATION_GUIDANCE | 63 |  |
| FEATURE_CUSTOM_LOGO | 64 |  |
| FEATURE_WATERMARK | 65 |  |
| FEATURE_ROADMAP_PRIORITIZATION | 66 |  |
| FEATURE_CUSTOM_MSA | 67 |  |
| FEATURE_COMMUNITY_SUPPORT | 68 |  |
| FEATURE_EMAIL_SUPPORT | 69 |  |
| FEATURE_DEDICATED_SUPPORT_WITH_SLA | 70 |  |



<a name="bytebase-v1-PlanType"></a>

### PlanType


| Name | Number | Description |
| ---- | ------ | ----------- |
| PLAN_TYPE_UNSPECIFIED | 0 | Unspecified plan type. |
| FREE | 1 | Free plan with basic features and no cost. |
| TEAM | 2 | Team plan with collaboration features for small to medium teams. |
| ENTERPRISE | 3 | Enterprise plan with advanced features and dedicated support. |


 

 


<a name="bytebase-v1-SubscriptionService"></a>

### SubscriptionService
SubscriptionService manages enterprise subscriptions and licensing.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetSubscription | [GetSubscriptionRequest](#bytebase-v1-GetSubscriptionRequest) | [Subscription](#bytebase-v1-Subscription) | GetSubscription returns the current subscription. If there is no license, we will return a free plan subscription without expiration time. If there is expired license, we will return a free plan subscription with the expiration time of the expired license. Permissions required: None |
| UpdateSubscription | [UpdateSubscriptionRequest](#bytebase-v1-UpdateSubscriptionRequest) | [Subscription](#bytebase-v1-Subscription) | Updates the enterprise license subscription. Permissions required: bb.settings.set |

 



<a name="v1_worksheet_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/worksheet_service.proto



<a name="bytebase-v1-BatchUpdateWorksheetOrganizerRequest"></a>

### BatchUpdateWorksheetOrganizerRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| requests | [UpdateWorksheetOrganizerRequest](#bytebase-v1-UpdateWorksheetOrganizerRequest) | repeated |  |






<a name="bytebase-v1-BatchUpdateWorksheetOrganizerResponse"></a>

### BatchUpdateWorksheetOrganizerResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| worksheet_organizers | [WorksheetOrganizer](#bytebase-v1-WorksheetOrganizer) | repeated |  |






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
| filter | [string](#string) |  | To filter the search result. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filter: - creator: the worksheet creator in &#34;users/{email}&#34; format, support &#34;==&#34; and &#34;!=&#34; operator. - starred: should be &#34;true&#34; or &#34;false&#34;, filter starred/unstarred sheets, support &#34;==&#34; operator. - visibility: check Visibility enum in the Worksheet message for values, support &#34;==&#34; and &#34;in [xx]&#34; operator. - project: the project full name in &#34;projects/{id}&#34; format, support &#34;==&#34; operator.

For example: creator == &#34;users/{email}&#34; creator != &#34;users/{email}&#34; starred == true starred == false visibility in [&#34;PRIVATE&#34;, &#34;PROJECT_READ&#34;, &#34;PROJECT_WRITE&#34;] visibility == &#34;PRIVATE&#34; project == &#34;projects/{project resource id}&#34; |






<a name="bytebase-v1-SearchWorksheetsResponse"></a>

### SearchWorksheetsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| worksheets | [Worksheet](#bytebase-v1-Worksheet) | repeated | The worksheets that matched the search criteria. |






<a name="bytebase-v1-UpdateWorksheetOrganizerRequest"></a>

### UpdateWorksheetOrganizerRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| organizer | [WorksheetOrganizer](#bytebase-v1-WorksheetOrganizer) |  | The organizer to update.

The organizer&#39;s `worksheet` field is used to identify the worksheet. Format: worksheets/{worksheet} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to be updated. Fields are specified relative to the worksheet organizer. Only support update the following fields for now: - `starred` |
| allow_missing | [bool](#bool) |  | If set to true, and the worksheet organizer is not found, a new worksheet organizer will be created. In this situation, `update_mask` is ignored. |






<a name="bytebase-v1-UpdateWorksheetRequest"></a>

### UpdateWorksheetRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| worksheet | [Worksheet](#bytebase-v1-Worksheet) |  | The worksheet to update.

The worksheet&#39;s `name` field is used to identify the worksheet to update. Format: worksheets/{worksheet} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to be updated. Fields are specified relative to the worksheet. (e.g., `title`, `statement`; *not* `worksheet.title` or `worksheet.statement`) Only support update the following fields for now: - `title` - `statement` - `starred` - `visibility` |






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
| folders | [string](#string) | repeated |  |






<a name="bytebase-v1-WorksheetOrganizer"></a>

### WorksheetOrganizer



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| worksheet | [string](#string) |  | The name of the worksheet. Format: worksheets/{worksheet} |
| starred | [bool](#bool) |  | starred means if the worksheet is starred. |
| folders | [string](#string) | repeated |  |





 


<a name="bytebase-v1-Worksheet-Visibility"></a>

### Worksheet.Visibility


| Name | Number | Description |
| ---- | ------ | ----------- |
| VISIBILITY_UNSPECIFIED | 0 |  |
| PROJECT_READ | 1 | Read access in project scope, worksheet OWNER/DBA and project OWNER can read/write, other project members can read. |
| PROJECT_WRITE | 2 | Write access in project scope, worksheet OWNER/DBA and all members in the project can write the worksheet. |
| PRIVATE | 3 | Private, only worksheet OWNER can read/write. |


 

 


<a name="bytebase-v1-WorksheetService"></a>

### WorksheetService
WorksheetService manages SQL worksheets for query development.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateWorksheet | [CreateWorksheetRequest](#bytebase-v1-CreateWorksheetRequest) | [Worksheet](#bytebase-v1-Worksheet) | Creates a personal worksheet used in SQL Editor. Any authenticated user can create their own worksheets. Permissions required: None (authenticated users only) |
| GetWorksheet | [GetWorksheetRequest](#bytebase-v1-GetWorksheetRequest) | [Worksheet](#bytebase-v1-Worksheet) | Get a worksheet by name. The users can access this method if, - they are the creator of the worksheet; - they have bb.worksheets.get permission on the workspace; - the sheet is shared with them with PROJECT_READ and PROJECT_WRITE visibility, and they have bb.projects.get permission on the project. Permissions required: bb.worksheets.get (or creator, or project member for shared worksheets) |
| SearchWorksheets | [SearchWorksheetsRequest](#bytebase-v1-SearchWorksheetsRequest) | [SearchWorksheetsResponse](#bytebase-v1-SearchWorksheetsResponse) | Search for worksheets. This is used for finding my worksheets or worksheets shared by other people. The sheet accessibility is the same as GetWorksheet(). Permissions required: bb.worksheets.get (or creator, or project member for shared worksheets) |
| UpdateWorksheet | [UpdateWorksheetRequest](#bytebase-v1-UpdateWorksheetRequest) | [Worksheet](#bytebase-v1-Worksheet) | Update a worksheet. The users can access this method if, - they are the creator of the worksheet; - they have bb.worksheets.manage permission on the workspace; - the sheet is shared with them with PROJECT_WRITE visibility, and they have bb.projects.get permission on the project. Permissions required: bb.worksheets.manage (or creator, or project member for PROJECT_WRITE worksheets) |
| UpdateWorksheetOrganizer | [UpdateWorksheetOrganizerRequest](#bytebase-v1-UpdateWorksheetOrganizerRequest) | [WorksheetOrganizer](#bytebase-v1-WorksheetOrganizer) | Update the organizer of a worksheet. The access is the same as UpdateWorksheet method. Permissions required: bb.worksheets.get (or creator, or project member for shared worksheets) |
| BatchUpdateWorksheetOrganizer | [BatchUpdateWorksheetOrganizerRequest](#bytebase-v1-BatchUpdateWorksheetOrganizerRequest) | [BatchUpdateWorksheetOrganizerResponse](#bytebase-v1-BatchUpdateWorksheetOrganizerResponse) | Batch update the organizers of worksheets. The access is the same as UpdateWorksheet method. Permissions required: bb.worksheets.get (or creator, or project member for shared worksheets) |
| DeleteWorksheet | [DeleteWorksheetRequest](#bytebase-v1-DeleteWorksheetRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Delete a worksheet. The access is the same as UpdateWorksheet method. Permissions required: bb.worksheets.manage (or creator, or project member for PROJECT_WRITE worksheets) |

 



<a name="v1_workspace_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/workspace_service.proto


 

 

 


<a name="bytebase-v1-WorkspaceService"></a>

### WorkspaceService
WorkspaceService manages workspace-level operations and profile.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetIamPolicy | [GetIamPolicyRequest](#bytebase-v1-GetIamPolicyRequest) | [IamPolicy](#bytebase-v1-IamPolicy) | Retrieves IAM policy for the workspace. Permissions required: bb.workspaces.getIamPolicy |
| SetIamPolicy | [SetIamPolicyRequest](#bytebase-v1-SetIamPolicyRequest) | [IamPolicy](#bytebase-v1-IamPolicy) | Sets IAM policy for the workspace. Permissions required: bb.workspaces.setIamPolicy |

 



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

