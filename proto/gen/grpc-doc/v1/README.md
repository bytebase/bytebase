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
    - [Position](#bytebase-v1-Position)
    - [Range](#bytebase-v1-Range)
  
    - [Engine](#bytebase-v1-Engine)
    - [ExportFormat](#bytebase-v1-ExportFormat)
    - [State](#bytebase-v1-State)
    - [VCSType](#bytebase-v1-VCSType)
  
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
  
- [v1/instance_role_service.proto](#v1_instance_role_service-proto)
    - [GetInstanceRoleRequest](#bytebase-v1-GetInstanceRoleRequest)
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
    - [ChangedResourceDatabase](#bytebase-v1-ChangedResourceDatabase)
    - [ChangedResourceFunction](#bytebase-v1-ChangedResourceFunction)
    - [ChangedResourceProcedure](#bytebase-v1-ChangedResourceProcedure)
    - [ChangedResourceSchema](#bytebase-v1-ChangedResourceSchema)
    - [ChangedResourceTable](#bytebase-v1-ChangedResourceTable)
    - [ChangedResourceView](#bytebase-v1-ChangedResourceView)
    - [ChangedResources](#bytebase-v1-ChangedResources)
    - [Changelog](#bytebase-v1-Changelog)
    - [CheckConstraintMetadata](#bytebase-v1-CheckConstraintMetadata)
    - [ColumnMetadata](#bytebase-v1-ColumnMetadata)
    - [Database](#bytebase-v1-Database)
    - [Database.LabelsEntry](#bytebase-v1-Database-LabelsEntry)
    - [DatabaseMetadata](#bytebase-v1-DatabaseMetadata)
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
    - [GetSchemaStringRequest.ObjectType](#bytebase-v1-GetSchemaStringRequest-ObjectType)
    - [StreamMetadata.Mode](#bytebase-v1-StreamMetadata-Mode)
    - [StreamMetadata.Type](#bytebase-v1-StreamMetadata-Type)
    - [TablePartitionMetadata.Type](#bytebase-v1-TablePartitionMetadata-Type)
    - [TaskMetadata.State](#bytebase-v1-TaskMetadata-State)
  
    - [DatabaseService](#bytebase-v1-DatabaseService)
  
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
  
    - [ApprovalNode.Type](#bytebase-v1-ApprovalNode-Type)
    - [ApprovalStep.Type](#bytebase-v1-ApprovalStep-Type)
    - [Issue.Approver.Status](#bytebase-v1-Issue-Approver-Status)
    - [Issue.RiskLevel](#bytebase-v1-Issue-RiskLevel)
    - [Issue.Type](#bytebase-v1-Issue-Type)
    - [IssueComment.Approval.Status](#bytebase-v1-IssueComment-Approval-Status)
    - [IssueComment.TaskUpdate.Status](#bytebase-v1-IssueComment-TaskUpdate-Status)
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
    - [AppIMSetting.Lark](#bytebase-v1-AppIMSetting-Lark)
    - [AppIMSetting.Slack](#bytebase-v1-AppIMSetting-Slack)
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
    - [PasswordRestrictionSetting](#bytebase-v1-PasswordRestrictionSetting)
    - [SCIMSetting](#bytebase-v1-SCIMSetting)
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
  
    - [AISetting.Provider](#bytebase-v1-AISetting-Provider)
    - [Algorithm.InnerOuterMask.MaskType](#bytebase-v1-Algorithm-InnerOuterMask-MaskType)
    - [Announcement.AlertLevel](#bytebase-v1-Announcement-AlertLevel)
    - [DatabaseChangeMode](#bytebase-v1-DatabaseChangeMode)
    - [Setting.SettingName](#bytebase-v1-Setting-SettingName)
  
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
    - [UpdateUserRequest](#bytebase-v1-UpdateUserRequest)
    - [User](#bytebase-v1-User)
    - [User.Profile](#bytebase-v1-User-Profile)
  
    - [UserType](#bytebase-v1-UserType)
  
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
    - [IdentityProviderContext](#bytebase-v1-IdentityProviderContext)
    - [LoginRequest](#bytebase-v1-LoginRequest)
    - [LoginResponse](#bytebase-v1-LoginResponse)
    - [LogoutRequest](#bytebase-v1-LogoutRequest)
    - [OAuth2IdentityProviderContext](#bytebase-v1-OAuth2IdentityProviderContext)
    - [OIDCIdentityProviderContext](#bytebase-v1-OIDCIdentityProviderContext)
  
    - [AuthService](#bytebase-v1-AuthService)
  
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
    - [DisableCopyDataPolicy](#bytebase-v1-DisableCopyDataPolicy)
    - [GetPolicyRequest](#bytebase-v1-GetPolicyRequest)
    - [ListPoliciesRequest](#bytebase-v1-ListPoliciesRequest)
    - [ListPoliciesResponse](#bytebase-v1-ListPoliciesResponse)
    - [MaskingExceptionPolicy](#bytebase-v1-MaskingExceptionPolicy)
    - [MaskingExceptionPolicy.MaskingException](#bytebase-v1-MaskingExceptionPolicy-MaskingException)
    - [MaskingRulePolicy](#bytebase-v1-MaskingRulePolicy)
    - [MaskingRulePolicy.MaskingRule](#bytebase-v1-MaskingRulePolicy-MaskingRule)
    - [Policy](#bytebase-v1-Policy)
    - [QueryDataPolicy](#bytebase-v1-QueryDataPolicy)
    - [RestrictIssueCreationForSQLReviewPolicy](#bytebase-v1-RestrictIssueCreationForSQLReviewPolicy)
    - [RolloutPolicy](#bytebase-v1-RolloutPolicy)
    - [SQLReviewRule](#bytebase-v1-SQLReviewRule)
    - [TagPolicy](#bytebase-v1-TagPolicy)
    - [TagPolicy.TagsEntry](#bytebase-v1-TagPolicy-TagsEntry)
    - [UpdatePolicyRequest](#bytebase-v1-UpdatePolicyRequest)
  
    - [DataSourceQueryPolicy.Restriction](#bytebase-v1-DataSourceQueryPolicy-Restriction)
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
    - [Plan.CreateDatabaseConfig](#bytebase-v1-Plan-CreateDatabaseConfig)
    - [Plan.Deployment](#bytebase-v1-Plan-Deployment)
    - [Plan.Deployment.DatabaseGroupMapping](#bytebase-v1-Plan-Deployment-DatabaseGroupMapping)
    - [Plan.ExportDataConfig](#bytebase-v1-Plan-ExportDataConfig)
    - [Plan.PlanCheckRunStatusCountEntry](#bytebase-v1-Plan-PlanCheckRunStatusCountEntry)
    - [Plan.Spec](#bytebase-v1-Plan-Spec)
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
    - [BatchDeleteProjectsRequest](#bytebase-v1-BatchDeleteProjectsRequest)
    - [BatchGetIamPolicyRequest](#bytebase-v1-BatchGetIamPolicyRequest)
    - [BatchGetIamPolicyResponse](#bytebase-v1-BatchGetIamPolicyResponse)
    - [BatchGetIamPolicyResponse.PolicyResult](#bytebase-v1-BatchGetIamPolicyResponse-PolicyResult)
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
    - [Webhook.Type](#bytebase-v1-Webhook-Type)
  
    - [ProjectService](#bytebase-v1-ProjectService)
  
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
    - [CheckRequest](#bytebase-v1-CheckRequest)
    - [CheckResponse](#bytebase-v1-CheckResponse)
    - [DiffMetadataRequest](#bytebase-v1-DiffMetadataRequest)
    - [DiffMetadataResponse](#bytebase-v1-DiffMetadataResponse)
    - [ExportRequest](#bytebase-v1-ExportRequest)
    - [ExportResponse](#bytebase-v1-ExportResponse)
    - [MaskingReason](#bytebase-v1-MaskingReason)
    - [PrettyRequest](#bytebase-v1-PrettyRequest)
    - [PrettyResponse](#bytebase-v1-PrettyResponse)
    - [QueryHistory](#bytebase-v1-QueryHistory)
    - [QueryOption](#bytebase-v1-QueryOption)
    - [QueryRequest](#bytebase-v1-QueryRequest)
    - [QueryResponse](#bytebase-v1-QueryResponse)
    - [QueryResult](#bytebase-v1-QueryResult)
    - [QueryResult.Message](#bytebase-v1-QueryResult-Message)
    - [QueryResult.PostgresError](#bytebase-v1-QueryResult-PostgresError)
    - [QueryRow](#bytebase-v1-QueryRow)
    - [RowValue](#bytebase-v1-RowValue)
    - [RowValue.Timestamp](#bytebase-v1-RowValue-Timestamp)
    - [RowValue.TimestampTZ](#bytebase-v1-RowValue-TimestampTZ)
    - [SearchQueryHistoriesRequest](#bytebase-v1-SearchQueryHistoriesRequest)
    - [SearchQueryHistoriesResponse](#bytebase-v1-SearchQueryHistoriesResponse)
  
    - [Advice.Status](#bytebase-v1-Advice-Status)
    - [CheckRequest.ChangeType](#bytebase-v1-CheckRequest-ChangeType)
    - [QueryHistory.Type](#bytebase-v1-QueryHistory-Type)
    - [QueryOption.RedisRunCommandsOn](#bytebase-v1-QueryOption-RedisRunCommandsOn)
    - [QueryResult.Message.Level](#bytebase-v1-QueryResult-Message-Level)
  
    - [SQLService](#bytebase-v1-SQLService)
  
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
    - [SearchReleasesRequest](#bytebase-v1-SearchReleasesRequest)
    - [SearchReleasesResponse](#bytebase-v1-SearchReleasesResponse)
    - [UndeleteReleaseRequest](#bytebase-v1-UndeleteReleaseRequest)
    - [UpdateReleaseRequest](#bytebase-v1-UpdateReleaseRequest)
  
    - [CheckReleaseResponse.RiskLevel](#bytebase-v1-CheckReleaseResponse-RiskLevel)
    - [Release.File.ChangeType](#bytebase-v1-Release-File-ChangeType)
    - [Release.File.Type](#bytebase-v1-Release-File-Type)
  
    - [ReleaseService](#bytebase-v1-ReleaseService)
  
- [v1/review_config_service.proto](#v1_review_config_service-proto)
    - [CreateReviewConfigRequest](#bytebase-v1-CreateReviewConfigRequest)
    - [DeleteReviewConfigRequest](#bytebase-v1-DeleteReviewConfigRequest)
    - [GetReviewConfigRequest](#bytebase-v1-GetReviewConfigRequest)
    - [ListReviewConfigsRequest](#bytebase-v1-ListReviewConfigsRequest)
    - [ListReviewConfigsResponse](#bytebase-v1-ListReviewConfigsResponse)
    - [ReviewConfig](#bytebase-v1-ReviewConfig)
    - [UpdateReviewConfigRequest](#bytebase-v1-UpdateReviewConfigRequest)
  
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
  
- [v1/risk_service.proto](#v1_risk_service-proto)
    - [CreateRiskRequest](#bytebase-v1-CreateRiskRequest)
    - [DeleteRiskRequest](#bytebase-v1-DeleteRiskRequest)
    - [GetRiskRequest](#bytebase-v1-GetRiskRequest)
    - [ListRisksRequest](#bytebase-v1-ListRisksRequest)
    - [ListRisksResponse](#bytebase-v1-ListRisksResponse)
    - [Risk](#bytebase-v1-Risk)
    - [UpdateRiskRequest](#bytebase-v1-UpdateRiskRequest)
  
    - [Risk.Source](#bytebase-v1-Risk-Source)
  
    - [RiskService](#bytebase-v1-RiskService)
  
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
    - [PreviewRolloutRequest](#bytebase-v1-PreviewRolloutRequest)
    - [PreviewTaskRunRollbackRequest](#bytebase-v1-PreviewTaskRunRollbackRequest)
    - [PreviewTaskRunRollbackResponse](#bytebase-v1-PreviewTaskRunRollbackResponse)
    - [Rollout](#bytebase-v1-Rollout)
    - [Stage](#bytebase-v1-Stage)
    - [Task](#bytebase-v1-Task)
    - [Task.DatabaseCreate](#bytebase-v1-Task-DatabaseCreate)
    - [Task.DatabaseDataExport](#bytebase-v1-Task-DatabaseDataExport)
    - [Task.DatabaseDataUpdate](#bytebase-v1-Task-DatabaseDataUpdate)
    - [Task.DatabaseSchemaUpdate](#bytebase-v1-Task-DatabaseSchemaUpdate)
    - [TaskRun](#bytebase-v1-TaskRun)
    - [TaskRun.PriorBackupDetail](#bytebase-v1-TaskRun-PriorBackupDetail)
    - [TaskRun.PriorBackupDetail.Item](#bytebase-v1-TaskRun-PriorBackupDetail-Item)
    - [TaskRun.PriorBackupDetail.Item.Table](#bytebase-v1-TaskRun-PriorBackupDetail-Item-Table)
    - [TaskRun.SchedulerInfo](#bytebase-v1-TaskRun-SchedulerInfo)
    - [TaskRun.SchedulerInfo.WaitingCause](#bytebase-v1-TaskRun-SchedulerInfo-WaitingCause)
    - [TaskRun.SchedulerInfo.WaitingCause.Task](#bytebase-v1-TaskRun-SchedulerInfo-WaitingCause-Task)
    - [TaskRunLog](#bytebase-v1-TaskRunLog)
    - [TaskRunLogEntry](#bytebase-v1-TaskRunLogEntry)
    - [TaskRunLogEntry.CommandExecute](#bytebase-v1-TaskRunLogEntry-CommandExecute)
    - [TaskRunLogEntry.CommandExecute.CommandResponse](#bytebase-v1-TaskRunLogEntry-CommandExecute-CommandResponse)
    - [TaskRunLogEntry.ComputeDiff](#bytebase-v1-TaskRunLogEntry-ComputeDiff)
    - [TaskRunLogEntry.DatabaseSync](#bytebase-v1-TaskRunLogEntry-DatabaseSync)
    - [TaskRunLogEntry.PriorBackup](#bytebase-v1-TaskRunLogEntry-PriorBackup)
    - [TaskRunLogEntry.RetryInfo](#bytebase-v1-TaskRunLogEntry-RetryInfo)
    - [TaskRunLogEntry.SchemaDump](#bytebase-v1-TaskRunLogEntry-SchemaDump)
    - [TaskRunLogEntry.TaskRunStatusUpdate](#bytebase-v1-TaskRunLogEntry-TaskRunStatusUpdate)
    - [TaskRunLogEntry.TransactionControl](#bytebase-v1-TaskRunLogEntry-TransactionControl)
    - [TaskRunSession](#bytebase-v1-TaskRunSession)
    - [TaskRunSession.Postgres](#bytebase-v1-TaskRunSession-Postgres)
    - [TaskRunSession.Postgres.Session](#bytebase-v1-TaskRunSession-Postgres-Session)
  
    - [Task.Status](#bytebase-v1-Task-Status)
    - [Task.Type](#bytebase-v1-Task-Type)
    - [TaskRun.ExportArchiveStatus](#bytebase-v1-TaskRun-ExportArchiveStatus)
    - [TaskRun.Status](#bytebase-v1-TaskRun-Status)
    - [TaskRunLogEntry.TaskRunStatusUpdate.Status](#bytebase-v1-TaskRunLogEntry-TaskRunStatusUpdate-Status)
    - [TaskRunLogEntry.TransactionControl.Type](#bytebase-v1-TaskRunLogEntry-TransactionControl-Type)
    - [TaskRunLogEntry.Type](#bytebase-v1-TaskRunLogEntry-Type)
  
    - [RolloutService](#bytebase-v1-RolloutService)
  
- [v1/sheet_service.proto](#v1_sheet_service-proto)
    - [BatchCreateSheetsRequest](#bytebase-v1-BatchCreateSheetsRequest)
    - [BatchCreateSheetsResponse](#bytebase-v1-BatchCreateSheetsResponse)
    - [CreateSheetRequest](#bytebase-v1-CreateSheetRequest)
    - [GetSheetRequest](#bytebase-v1-GetSheetRequest)
    - [Sheet](#bytebase-v1-Sheet)
    - [SheetCommand](#bytebase-v1-SheetCommand)
    - [SheetPayload](#bytebase-v1-SheetPayload)
    - [UpdateSheetRequest](#bytebase-v1-UpdateSheetRequest)
  
    - [SheetPayload.Type](#bytebase-v1-SheetPayload-Type)
  
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
| audit | bool | .google.protobuf.MethodOptions | 100003 |  |
| auth_method | AuthMethod | .google.protobuf.MethodOptions | 100002 |  |
| permission | string | .google.protobuf.MethodOptions | 100001 |  |

 

 



<a name="v1_common-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/common.proto



<a name="bytebase-v1-Position"></a>

### Position
Position in a text expressed as zero-based line and zero-based column byte
offset.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| line | [int32](#int32) |  | Line position in a text (zero-based). |
| column | [int32](#int32) |  | Column position in a text (zero-based), equivalent to byte offset. |






<a name="bytebase-v1-Range"></a>

### Range



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start | [int32](#int32) |  |  |
| end | [int32](#int32) |  |  |





 


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



<a name="bytebase-v1-ExportFormat"></a>

### ExportFormat


| Name | Number | Description |
| ---- | ------ | ----------- |
| FORMAT_UNSPECIFIED | 0 |  |
| CSV | 1 |  |
| JSON | 2 |  |
| SQL | 3 |  |
| XLSX | 4 |  |



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
| GITHUB | 1 |  |
| GITLAB | 2 |  |
| BITBUCKET | 3 |  |
| AZURE_DEVOPS | 4 |  |


 

 

 



<a name="v1_database_catalog_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/database_catalog_service.proto



<a name="bytebase-v1-ColumnCatalog"></a>

### ColumnCatalog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| semantic_type | [string](#string) |  |  |
| labels | [ColumnCatalog.LabelsEntry](#bytebase-v1-ColumnCatalog-LabelsEntry) | repeated | The user labels for a column. |
| classification | [string](#string) |  |  |
| object_schema | [ObjectSchema](#bytebase-v1-ObjectSchema) | optional |  |






<a name="bytebase-v1-ColumnCatalog-LabelsEntry"></a>

### ColumnCatalog.LabelsEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="bytebase-v1-DatabaseCatalog"></a>

### DatabaseCatalog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database catalog. Format: instances/{instance}/databases/{database}/catalog |
| schemas | [SchemaCatalog](#bytebase-v1-SchemaCatalog) | repeated |  |






<a name="bytebase-v1-GetDatabaseCatalogRequest"></a>

### GetDatabaseCatalogRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database catalog to retrieve. Format: instances/{instance}/databases/{database}/catalog |






<a name="bytebase-v1-ObjectSchema"></a>

### ObjectSchema



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [ObjectSchema.Type](#bytebase-v1-ObjectSchema-Type) |  |  |
| struct_kind | [ObjectSchema.StructKind](#bytebase-v1-ObjectSchema-StructKind) |  |  |
| array_kind | [ObjectSchema.ArrayKind](#bytebase-v1-ObjectSchema-ArrayKind) |  |  |
| semantic_type | [string](#string) |  |  |






<a name="bytebase-v1-ObjectSchema-ArrayKind"></a>

### ObjectSchema.ArrayKind



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| kind | [ObjectSchema](#bytebase-v1-ObjectSchema) |  |  |






<a name="bytebase-v1-ObjectSchema-StructKind"></a>

### ObjectSchema.StructKind



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| properties | [ObjectSchema.StructKind.PropertiesEntry](#bytebase-v1-ObjectSchema-StructKind-PropertiesEntry) | repeated |  |






<a name="bytebase-v1-ObjectSchema-StructKind-PropertiesEntry"></a>

### ObjectSchema.StructKind.PropertiesEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [ObjectSchema](#bytebase-v1-ObjectSchema) |  |  |






<a name="bytebase-v1-SchemaCatalog"></a>

### SchemaCatalog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| tables | [TableCatalog](#bytebase-v1-TableCatalog) | repeated |  |






<a name="bytebase-v1-TableCatalog"></a>

### TableCatalog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| columns | [TableCatalog.Columns](#bytebase-v1-TableCatalog-Columns) |  |  |
| object_schema | [ObjectSchema](#bytebase-v1-ObjectSchema) |  |  |
| classification | [string](#string) |  |  |






<a name="bytebase-v1-TableCatalog-Columns"></a>

### TableCatalog.Columns



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| columns | [ColumnCatalog](#bytebase-v1-ColumnCatalog) | repeated |  |






<a name="bytebase-v1-UpdateDatabaseCatalogRequest"></a>

### UpdateDatabaseCatalogRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| catalog | [DatabaseCatalog](#bytebase-v1-DatabaseCatalog) |  | The database catalog to update.

The catalog&#39;s `name` field is used to identify the database catalog to update. Format: instances/{instance}/databases/{database}/catalog |





 


<a name="bytebase-v1-ObjectSchema-Type"></a>

### ObjectSchema.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| STRING | 1 |  |
| NUMBER | 2 |  |
| BOOLEAN | 3 |  |
| OBJECT | 4 |  |
| ARRAY | 5 |  |


 

 


<a name="bytebase-v1-DatabaseCatalogService"></a>

### DatabaseCatalogService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetDatabaseCatalog | [GetDatabaseCatalogRequest](#bytebase-v1-GetDatabaseCatalogRequest) | [DatabaseCatalog](#bytebase-v1-DatabaseCatalog) | Permissions required: bb.databaseCatalogs.get |
| UpdateDatabaseCatalog | [UpdateDatabaseCatalogRequest](#bytebase-v1-UpdateDatabaseCatalogRequest) | [DatabaseCatalog](#bytebase-v1-DatabaseCatalog) | Permissions required: bb.databaseCatalogs.update |

 



<a name="v1_instance_role_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/instance_role_service.proto



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
| name | [string](#string) |  | The name of the role. Format: instances/{instance}/roles/{role} The role name is the unique name for the role. |
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


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetInstanceRole | [GetInstanceRoleRequest](#bytebase-v1-GetInstanceRoleRequest) | [InstanceRole](#bytebase-v1-InstanceRole) | Permissions required: bb.instanceRoles.get |
| ListInstanceRoles | [ListInstanceRolesRequest](#bytebase-v1-ListInstanceRolesRequest) | [ListInstanceRolesResponse](#bytebase-v1-ListInstanceRolesResponse) | Permissions required: bb.instanceRoles.list |

 



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
| id | [string](#string) |  |  |
| type | [DataSourceType](#bytebase-v1-DataSourceType) |  |  |
| username | [string](#string) |  |  |
| password | [string](#string) |  |  |
| use_ssl | [bool](#bool) |  | Use SSL to connect to the data source. By default, we use system default SSL configuration. |
| ssl_ca | [string](#string) |  |  |
| ssl_cert | [string](#string) |  |  |
| ssl_key | [string](#string) |  |  |
| verify_tls_certificate | [bool](#bool) |  | verify_tls_certificate enables TLS certificate verification for SSL connections. Default is false (no verification) for backward compatibility. Set to true for secure connections (recommended for production). Only set to false for development or when certificates cannot be properly validated (e.g., self-signed certs, VPN environments). |
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
| state | [State](#bytebase-v1-State) |  |  |
| title | [string](#string) |  |  |
| engine | [Engine](#bytebase-v1-Engine) |  |  |
| engine_version | [string](#string) |  |  |
| external_link | [string](#string) |  |  |
| data_sources | [DataSource](#bytebase-v1-DataSource) | repeated |  |
| environment | [string](#string) | optional | The environment resource. Format: environments/prod where prod is the environment resource ID. |
| activation | [bool](#bool) |  |  |
| roles | [InstanceRole](#bytebase-v1-InstanceRole) | repeated |  |
| sync_interval | [google.protobuf.Duration](#google-protobuf-Duration) |  | How often the instance is synced. |
| maximum_connections | [int32](#int32) |  | The maximum number of connections. The default is 10 if the value is unset or zero. |
| sync_databases | [string](#string) | repeated | Enable sync for following databases. Default empty, means sync all schemas &amp; databases. |
| last_sync_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The last time the instance was synced. |






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
| environment | [string](#string) | optional | The environment resource. Format: environments/prod where prod is the environment resource ID. |






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

Supported filters: - name: the instance name, support &#34;==&#34; and &#34;.matches()&#34; operator. - resource_id: the instance id, support &#34;==&#34; and &#34;.matches()&#34; operator. - environment: the environment full name in &#34;environments/{id}&#34; format, support &#34;==&#34; operator. - state: the instance state, check State enum for values, support &#34;==&#34; operator. - engine: the instance engine, check Engine enum for values. Support &#34;==&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34; operator. - host: the instance host, support &#34;==&#34; and &#34;.matches()&#34; operator. - port: the instance port, support &#34;==&#34; and &#34;.matches()&#34; operator. - project: the project full name in &#34;projects/{id}&#34; format, support &#34;==&#34; operator.

For example: name == &#34;sample instance&#34; name.matches(&#34;sample&#34;) resource_id = &#34;sample-instance&#34; resource_id.matches(&#34;sample&#34;) state == &#34;DELETED&#34; environment == &#34;environments/test&#34; environment == &#34;&#34; (find instances which environment is not set) engine == &#34;MYSQL&#34; engine in [&#34;MYSQL&#34;, &#34;POSTGRES&#34;] !(engine in [&#34;MYSQL&#34;, &#34;POSTGRES&#34;]) host == &#34;127.0.0.1&#34; host.matches(&#34;127.0&#34;) port == &#34;54321&#34; port.matches(&#34;543&#34;) project == &#34;projects/sample-project&#34; You can combine filter conditions like: name.matches(&#34;sample&#34;) &amp;&amp; environment == &#34;environments/test&#34; host == &#34;127.0.0.1&#34; &amp;&amp; port == &#34;54321&#34; |






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
| krb_config | [KerberosConfig](#bytebase-v1-KerberosConfig) |  |  |






<a name="bytebase-v1-SyncInstanceRequest"></a>

### SyncInstanceRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of instance. Format: instances/{instance} |
| enable_full_sync | [bool](#bool) |  | When full sync is enabled, all databases in the instance will be synchronized. Otherwise, only the instance metadata (such as the database list) and any newly discovered instances will be synced. |






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
| GetInstance | [GetInstanceRequest](#bytebase-v1-GetInstanceRequest) | [Instance](#bytebase-v1-Instance) | Permissions required: bb.instances.get |
| ListInstances | [ListInstancesRequest](#bytebase-v1-ListInstancesRequest) | [ListInstancesResponse](#bytebase-v1-ListInstancesResponse) | Permissions required: bb.instances.list |
| CreateInstance | [CreateInstanceRequest](#bytebase-v1-CreateInstanceRequest) | [Instance](#bytebase-v1-Instance) | Permissions required: bb.instances.create |
| UpdateInstance | [UpdateInstanceRequest](#bytebase-v1-UpdateInstanceRequest) | [Instance](#bytebase-v1-Instance) | Permissions required: bb.instances.update |
| DeleteInstance | [DeleteInstanceRequest](#bytebase-v1-DeleteInstanceRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Permissions required: bb.instances.delete |
| UndeleteInstance | [UndeleteInstanceRequest](#bytebase-v1-UndeleteInstanceRequest) | [Instance](#bytebase-v1-Instance) | Permissions required: bb.instances.undelete |
| SyncInstance | [SyncInstanceRequest](#bytebase-v1-SyncInstanceRequest) | [SyncInstanceResponse](#bytebase-v1-SyncInstanceResponse) | Permissions required: bb.instances.sync |
| ListInstanceDatabase | [ListInstanceDatabaseRequest](#bytebase-v1-ListInstanceDatabaseRequest) | [ListInstanceDatabaseResponse](#bytebase-v1-ListInstanceDatabaseResponse) | Permissions required: bb.instances.get |
| BatchSyncInstances | [BatchSyncInstancesRequest](#bytebase-v1-BatchSyncInstancesRequest) | [BatchSyncInstancesResponse](#bytebase-v1-BatchSyncInstancesResponse) | Permissions required: bb.instances.sync |
| BatchUpdateInstances | [BatchUpdateInstancesRequest](#bytebase-v1-BatchUpdateInstancesRequest) | [BatchUpdateInstancesResponse](#bytebase-v1-BatchUpdateInstancesResponse) | Permissions required: bb.instances.update |
| AddDataSource | [AddDataSourceRequest](#bytebase-v1-AddDataSourceRequest) | [Instance](#bytebase-v1-Instance) | Permissions required: bb.instances.update |
| RemoveDataSource | [RemoveDataSourceRequest](#bytebase-v1-RemoveDataSourceRequest) | [Instance](#bytebase-v1-Instance) | Permissions required: bb.instances.update |
| UpdateDataSource | [UpdateDataSourceRequest](#bytebase-v1-UpdateDataSourceRequest) | [Instance](#bytebase-v1-Instance) | Permissions required: bb.instances.update |

 



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






<a name="bytebase-v1-ChangedResourceDatabase"></a>

### ChangedResourceDatabase



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| schemas | [ChangedResourceSchema](#bytebase-v1-ChangedResourceSchema) | repeated |  |






<a name="bytebase-v1-ChangedResourceFunction"></a>

### ChangedResourceFunction



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| ranges | [Range](#bytebase-v1-Range) | repeated | The ranges of sub-strings correspond to the statements on the sheet. |






<a name="bytebase-v1-ChangedResourceProcedure"></a>

### ChangedResourceProcedure



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| ranges | [Range](#bytebase-v1-Range) | repeated | The ranges of sub-strings correspond to the statements on the sheet. |






<a name="bytebase-v1-ChangedResourceSchema"></a>

### ChangedResourceSchema



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| tables | [ChangedResourceTable](#bytebase-v1-ChangedResourceTable) | repeated |  |
| views | [ChangedResourceView](#bytebase-v1-ChangedResourceView) | repeated |  |
| functions | [ChangedResourceFunction](#bytebase-v1-ChangedResourceFunction) | repeated |  |
| procedures | [ChangedResourceProcedure](#bytebase-v1-ChangedResourceProcedure) | repeated |  |






<a name="bytebase-v1-ChangedResourceTable"></a>

### ChangedResourceTable



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| ranges | [Range](#bytebase-v1-Range) | repeated | The ranges of sub-strings correspond to the statements on the sheet. |






<a name="bytebase-v1-ChangedResourceView"></a>

### ChangedResourceView



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| ranges | [Range](#bytebase-v1-Range) | repeated | The ranges of sub-strings correspond to the statements on the sheet. |






<a name="bytebase-v1-ChangedResources"></a>

### ChangedResources



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| databases | [ChangedResourceDatabase](#bytebase-v1-ChangedResourceDatabase) | repeated |  |






<a name="bytebase-v1-Changelog"></a>

### Changelog



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: instances/{instance}/databases/{database}/changelogs/{changelog} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| status | [Changelog.Status](#bytebase-v1-Changelog-Status) |  |  |
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
| changed_resources | [ChangedResources](#bytebase-v1-ChangedResources) |  |  |
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
| comment | [string](#string) |  | The comment is the comment of a column. classification and user_comment is parsed from the comment. |
| user_comment | [string](#string) |  | The user_comment is the user comment of a column parsed from the comment. |
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
| owner | [string](#string) |  |  |
| search_path | [string](#string) |  | The search_path is the search path of a PostgreSQL database. |






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
| sdl_format | [bool](#bool) |  | Format the schema dump into SDL format. |






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
| comment | [string](#string) |  |  |
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-v1-EventMetadata"></a>

### EventMetadata



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the event. |
| definition | [string](#string) |  | The schedule of the event. |
| time_zone | [string](#string) |  | The time zone of the event. |
| sql_mode | [string](#string) |  |  |
| character_set_client | [string](#string) |  |  |
| collation_connection | [string](#string) |  |  |
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
| sdl_format | [bool](#bool) |  | Format the schema dump into SDL format. |






<a name="bytebase-v1-GetDatabaseMetadataRequest"></a>

### GetDatabaseMetadataRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database to retrieve metadata. Format: instances/{instance}/databases/{database}/metadata |
| filter | [string](#string) |  | Filter is used to filter databases returned in the list. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filter: - schema: the schema name, support &#34;==&#34; operator. - table: the table name, support &#34;==&#34; operator.

For example: schema == &#34;schema-a&#34; table == &#34;table-a&#34; schema == &#34;schema-a&#34; &amp;&amp; table == &#34;table-a&#34; The filter used for a specific schema object such as &#34;schemas/schema-a/tables/table-a&#34;. The column masking level will only be returned when a table filter is used. |






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
| filter | [string](#string) |  | The filter of the changelogs. follow the [ebnf](https://en.wikipedia.org/wiki/Extended_Backus%E2%80%93Naur_form) syntax. Support filter by type, source or table. For example: table = &#34;tableExists(&#39;{database}&#39;, &#39;{schema}&#39;, &#39;{table}&#39;)&#34; table = &#34;tableExists(&#39;db&#39;, &#39;public&#39;, &#39;table1&#39;) || tableExists(&#39;db&#39;, &#39;public&#39;, &#39;table2&#39;)&#34;

The table filter follow the CEL syntax. currently, we have one function for CEL: - tableExists(database, schema, table): return true if the table exists in changed resources.

examples: Use tableExists(&#34;db&#34;, &#34;public&#34;, &#34;table1&#34;) to filter the changelogs which have the table &#34;table1&#34; in the schema &#34;public&#34; of the database &#34;db&#34;. For MySQL, the schema is always &#34;&#34;, such as tableExists(&#34;db&#34;, &#34;&#34;, &#34;table1&#34;).

Combine multiple functions with &#34;&amp;&amp;&#34; and &#34;||&#34;, we MUST use the Disjunctive Normal Form(DNF). In other words, the CEL expression consists of several parts connected by OR operators. For example, the following expression is valid: ( tableExists(&#34;db&#34;, &#34;public&#34;, &#34;table1&#34;) &amp;&amp; tableExists(&#34;db&#34;, &#34;public&#34;, &#34;table2&#34;) ) || ( tableExists(&#34;db&#34;, &#34;public&#34;, &#34;table3&#34;) ) |






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

Supported filter: - environment: the environment full name in &#34;environments/{id}&#34; format, support &#34;==&#34; operator. - name: the database name, support &#34;.matches()&#34; operator. - project: the project full name in &#34;projects/{id}&#34; format, support &#34;==&#34; operator. - instance: the instance full name in &#34;instances/{id}&#34; format, support &#34;==&#34; operator. - engine: the database engine, check Engine enum for values. Support &#34;==&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34; operator. - label: the database label in &#34;{key}:{value1},{value2}&#34; format. Support &#34;==&#34; operator. - exclude_unassigned: should be &#34;true&#34; or &#34;false&#34;, will not show unassigned databases if it&#39;s true, support &#34;==&#34; operator. - drifted: should be &#34;true&#34; or &#34;false&#34;, show drifted databases if it&#39;s true, support &#34;==&#34; operator. - table: filter by the database table, support &#34;==&#34; and &#34;.matches()&#34; operator.

For example: environment == &#34;environments/{environment resource id}&#34; environment == &#34;&#34; (find databases which environment is not set) project == &#34;projects/{project resource id}&#34; instance == &#34;instances/{instance resource id}&#34; name.matches(&#34;database name&#34;) engine == &#34;MYSQL&#34; engine in [&#34;MYSQL&#34;, &#34;POSTGRES&#34;] !(engine in [&#34;MYSQL&#34;, &#34;POSTGRES&#34;]) label == &#34;region:asia&#34; label == &#34;tenant:asia,europe&#34; label == &#34;region:asia&#34; &amp;&amp; label == &#34;tenant:bytebase&#34; exclude_unassigned == true drifted == true table == &#34;sample&#34; table.matches(&#34;sam&#34;)

You can combine filter conditions like: environment == &#34;environments/prod&#34; &amp;&amp; name.matches(&#34;employee&#34;) |
| show_deleted | [bool](#bool) |  | Show deleted database if specified. |






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
| signature | [string](#string) |  | The signature is the name with the number and type of input arguments the function takes. |
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
| owner | [string](#string) |  |  |
| sequences | [SequenceMetadata](#bytebase-v1-SequenceMetadata) | repeated | The sequences is the list of sequences in a schema, sorted by name. |
| events | [EventMetadata](#bytebase-v1-EventMetadata) | repeated |  |
| enum_types | [EnumTypeMetadata](#bytebase-v1-EnumTypeMetadata) | repeated |  |
| skip_dump | [bool](#bool) |  |  |
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
| comment | [string](#string) |  |  |
| skip_dump | [bool](#bool) |  |  |






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
| comment | [string](#string) |  | The comment is the comment of a table. classification and user_comment is parsed from the comment. |
| user_comment | [string](#string) |  | The user_comment is the user comment of a table parsed from the comment. |
| foreign_keys | [ForeignKeyMetadata](#bytebase-v1-ForeignKeyMetadata) | repeated | The foreign_keys is the list of foreign keys in a table. |
| partitions | [TablePartitionMetadata](#bytebase-v1-TablePartitionMetadata) | repeated | The partitions is the list of partitions in a table. |
| check_constraints | [CheckConstraintMetadata](#bytebase-v1-CheckConstraintMetadata) | repeated | The check_constraints is the list of check constraints in a table. |
| owner | [string](#string) |  |  |
| sorting_keys | [string](#string) | repeated | The sorting_keys is a tuple of column names or arbitrary expressions. ClickHouse specific field. Reference: https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/mergetree#order_by |
| triggers | [TriggerMetadata](#bytebase-v1-TriggerMetadata) | repeated |  |
| skip_dump | [bool](#bool) |  |  |
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
| sql_mode | [string](#string) |  |  |
| character_set_client | [string](#string) |  |  |
| collation_connection | [string](#string) |  |  |
| comment | [string](#string) |  |  |
| skip_dump | [bool](#bool) |  |  |






<a name="bytebase-v1-UpdateDatabaseRequest"></a>

### UpdateDatabaseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [Database](#bytebase-v1-Database) |  | The database to update.

The database&#39;s `name` field is used to identify the database to update. Format: instances/{instance}/databases/{database} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






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
| MIGRATE_SDL | 3 |  |
| MIGRATE_GHOST | 4 |  |
| DATA | 6 |  |



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


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetDatabase | [GetDatabaseRequest](#bytebase-v1-GetDatabaseRequest) | [Database](#bytebase-v1-Database) | Permissions required: bb.databases.get |
| BatchGetDatabases | [BatchGetDatabasesRequest](#bytebase-v1-BatchGetDatabasesRequest) | [BatchGetDatabasesResponse](#bytebase-v1-BatchGetDatabasesResponse) | Permissions required: bb.databases.get |
| ListDatabases | [ListDatabasesRequest](#bytebase-v1-ListDatabasesRequest) | [ListDatabasesResponse](#bytebase-v1-ListDatabasesResponse) | Permissions required: bb.databases.list |
| UpdateDatabase | [UpdateDatabaseRequest](#bytebase-v1-UpdateDatabaseRequest) | [Database](#bytebase-v1-Database) | Permissions required: bb.databases.update |
| BatchUpdateDatabases | [BatchUpdateDatabasesRequest](#bytebase-v1-BatchUpdateDatabasesRequest) | [BatchUpdateDatabasesResponse](#bytebase-v1-BatchUpdateDatabasesResponse) | Permissions required: bb.databases.update |
| SyncDatabase | [SyncDatabaseRequest](#bytebase-v1-SyncDatabaseRequest) | [SyncDatabaseResponse](#bytebase-v1-SyncDatabaseResponse) | Permissions required: bb.databases.sync |
| BatchSyncDatabases | [BatchSyncDatabasesRequest](#bytebase-v1-BatchSyncDatabasesRequest) | [BatchSyncDatabasesResponse](#bytebase-v1-BatchSyncDatabasesResponse) | Permissions required: bb.databases.sync |
| GetDatabaseMetadata | [GetDatabaseMetadataRequest](#bytebase-v1-GetDatabaseMetadataRequest) | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) | Permissions required: bb.databases.getSchema |
| GetDatabaseSchema | [GetDatabaseSchemaRequest](#bytebase-v1-GetDatabaseSchemaRequest) | [DatabaseSchema](#bytebase-v1-DatabaseSchema) | Permissions required: bb.databases.getSchema |
| DiffSchema | [DiffSchemaRequest](#bytebase-v1-DiffSchemaRequest) | [DiffSchemaResponse](#bytebase-v1-DiffSchemaResponse) | Permissions required: bb.databases.get |
| ListChangelogs | [ListChangelogsRequest](#bytebase-v1-ListChangelogsRequest) | [ListChangelogsResponse](#bytebase-v1-ListChangelogsResponse) | Permissions required: bb.changelogs.list |
| GetChangelog | [GetChangelogRequest](#bytebase-v1-GetChangelogRequest) | [Changelog](#bytebase-v1-Changelog) | Permissions required: changelogs.get |
| GetSchemaString | [GetSchemaStringRequest](#bytebase-v1-GetSchemaStringRequest) | [GetSchemaStringResponse](#bytebase-v1-GetSchemaStringResponse) | Permissions required: databases.getSchema |

 



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
| role | [string](#string) |  |  |






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
| condition | [google.type.Expr](#google-type-Expr) |  | The condition for the role. Same as the condtion in IAM Binding message. |
| expiration | [google.protobuf.Duration](#google-protobuf-Duration) |  |  |






<a name="bytebase-v1-Issue"></a>

### Issue



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the issue. Format: projects/{project}/issues/{issue} |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| type | [Issue.Type](#bytebase-v1-Issue-Type) |  |  |
| status | [IssueStatus](#bytebase-v1-IssueStatus) |  |  |
| approvers | [Issue.Approver](#bytebase-v1-Issue-Approver) | repeated |  |
| approval_templates | [ApprovalTemplate](#bytebase-v1-ApprovalTemplate) | repeated |  |
| approval_finding_done | [bool](#bool) |  | If the value is `false`, it means that the backend is still finding matching approval templates. If `true`, approval_templates &amp; approvers &amp; approval_finding_error are available. |
| approval_finding_error | [string](#string) |  |  |
| creator | [string](#string) |  | Format: users/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| plan | [string](#string) |  | The plan associated with the issue. Can be empty. Format: projects/{project}/plans/{plan} |
| rollout | [string](#string) |  | The rollout associated with the issue. Can be empty. Format: projects/{project}/rollouts/{rollout} |
| grant_request | [GrantRequest](#bytebase-v1-GrantRequest) |  | Used if the issue type is GRANT_REQUEST. |
| releasers | [string](#string) | repeated | The releasers of the pending stage of the issue rollout, judging from the rollout policy. Format: - roles/workspaceOwner - roles/workspaceDBA - roles/projectOwner - roles/projectReleaser - users/{email} |
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
| name | [string](#string) |  | Format: projects/{project}/issues/{issue}/issueComments/{issueComment-uid} |
| comment | [string](#string) |  |  |
| payload | [string](#string) |  | TODO: use struct message instead. |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
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
| from_labels | [string](#string) | repeated |  |
| to_labels | [string](#string) | repeated |  |






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
| error | [string](#string) |  |  |






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
| to_status | [IssueComment.TaskUpdate.Status](#bytebase-v1-IssueComment-TaskUpdate-Status) | optional |  |






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
| issue_comments | [IssueComment](#bytebase-v1-IssueComment) | repeated |  |
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

Supported filters: - creator: issue creator full name in &#34;users/{email or id}&#34; format, support &#34;==&#34; operator. - status: the issue status, support &#34;==&#34; and &#34;in&#34; operator, check the IssueStatus enum for the values. - create_time: issue create time in &#34;2006-01-02T15:04:05Z07:00&#34; format, support &#34;&gt;=&#34; or &#34;&lt;=&#34; operator. - type: the issue type, support &#34;==&#34; and &#34;in&#34; operator, check the Type enum in the Issue message for the values. - task_type: support &#34;==&#34; operator, the value can be &#34;DDL&#34;, &#34;DML&#34; or &#34;DATA_EXPORT&#34; - instance: the instance full name in the &#34;instances/{id}&#34; format, support &#34;==&#34; operator. - database: the database full name in the &#34;instances/{id}/databases/{name}&#34; format, support &#34;==&#34; operator. - labels: the issue labels, support &#34;==&#34; and &#34;in&#34; operator. - has_pipeline: the issue has pipeline or not, support &#34;==&#34; operator, the value should be &#34;true&#34; or &#34;false&#34;.

For example: creator == &#34;users/ed@bytebase.com&#34; &amp;&amp; status in [&#34;OPEN&#34;, &#34;DONE&#34;] status == &#34;CANCELED&#34; &amp;&amp; type == &#34;DATABASE_CHANGE&#34; instance == &#34;instances/sample&#34; &amp;&amp; labels in [&#34;label1&#34;, &#34;label2&#34;] has_pipeline == true &amp;&amp; create_time &gt;= &#34;2025-01-02T15:04:05Z07:00&#34; |
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
| issue_comment | [IssueComment](#bytebase-v1-IssueComment) |  |  |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |






<a name="bytebase-v1-UpdateIssueRequest"></a>

### UpdateIssueRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| issue | [Issue](#bytebase-v1-Issue) |  | The issue to update.

The issue&#39;s `name` field is used to identify the issue to update. Format: projects/{project}/issues/{issue} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 


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
| DATABASE_EXPORT | 3 |  |



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
| GetIssue | [GetIssueRequest](#bytebase-v1-GetIssueRequest) | [Issue](#bytebase-v1-Issue) | Permissions required: bb.issues.get |
| CreateIssue | [CreateIssueRequest](#bytebase-v1-CreateIssueRequest) | [Issue](#bytebase-v1-Issue) | Permissions required: bb.issues.create |
| ListIssues | [ListIssuesRequest](#bytebase-v1-ListIssuesRequest) | [ListIssuesResponse](#bytebase-v1-ListIssuesResponse) | Permissions required: bb.issues.list |
| SearchIssues | [SearchIssuesRequest](#bytebase-v1-SearchIssuesRequest) | [SearchIssuesResponse](#bytebase-v1-SearchIssuesResponse) | Search for issues that the caller has the bb.issues.get permission on and also satisfy the specified filter &amp; query. Permissions required: bb.issues.get |
| UpdateIssue | [UpdateIssueRequest](#bytebase-v1-UpdateIssueRequest) | [Issue](#bytebase-v1-Issue) | Permissions required: bb.issues.update |
| ListIssueComments | [ListIssueCommentsRequest](#bytebase-v1-ListIssueCommentsRequest) | [ListIssueCommentsResponse](#bytebase-v1-ListIssueCommentsResponse) | Permissions required: bb.issueComments.list |
| CreateIssueComment | [CreateIssueCommentRequest](#bytebase-v1-CreateIssueCommentRequest) | [IssueComment](#bytebase-v1-IssueComment) | Permissions required: bb.issueComments.create |
| UpdateIssueComment | [UpdateIssueCommentRequest](#bytebase-v1-UpdateIssueCommentRequest) | [IssueComment](#bytebase-v1-IssueComment) | Permissions required: bb.issueComments.update |
| BatchUpdateIssuesStatus | [BatchUpdateIssuesStatusRequest](#bytebase-v1-BatchUpdateIssuesStatusRequest) | [BatchUpdateIssuesStatusResponse](#bytebase-v1-BatchUpdateIssuesStatusResponse) | Permissions required: bb.issues.update |
| ApproveIssue | [ApproveIssueRequest](#bytebase-v1-ApproveIssueRequest) | [Issue](#bytebase-v1-Issue) | ApproveIssue approves the issue. The access is based on approval flow. Permissions required: None |
| RejectIssue | [RejectIssueRequest](#bytebase-v1-RejectIssueRequest) | [Issue](#bytebase-v1-Issue) | RejectIssue rejects the issue. The access is based on approval flow. Permissions required: None |
| RequestIssue | [RequestIssueRequest](#bytebase-v1-RequestIssueRequest) | [Issue](#bytebase-v1-Issue) | RequestIssue requests the issue. The access is based on approval flow. Permissions required: None |

 



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
| start | [int32](#int32) |  | start is the start index of the original value, start from 0 and should be less than stop. |
| end | [int32](#int32) |  | stop is the stop index of the original value, should be less than the length of the original value. |
| substitution | [string](#string) |  | substitution is the string used to replace the OriginalValue[start:end). |






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
| lark | [AppIMSetting.Lark](#bytebase-v1-AppIMSetting-Lark) |  |  |
| dingtalk | [AppIMSetting.DingTalk](#bytebase-v1-AppIMSetting-DingTalk) |  |  |






<a name="bytebase-v1-AppIMSetting-DingTalk"></a>

### AppIMSetting.DingTalk



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| client_id | [string](#string) |  |  |
| client_secret | [string](#string) |  |  |
| robot_code | [string](#string) |  |  |






<a name="bytebase-v1-AppIMSetting-Feishu"></a>

### AppIMSetting.Feishu



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| enabled | [bool](#bool) |  |  |
| app_id | [string](#string) |  |  |
| app_secret | [string](#string) |  |  |






<a name="bytebase-v1-AppIMSetting-Lark"></a>

### AppIMSetting.Lark



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



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Not used. The maximum number of settings to return. The service may return fewer than this value. If unspecified, at most 10 settings will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListSettings` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListSettings` must match the call that provided the page token. |






<a name="bytebase-v1-ListSettingsResponse"></a>

### ListSettingsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| settings | [Setting](#bytebase-v1-Setting) | repeated | The settings from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-PasswordRestrictionSetting"></a>

### PasswordRestrictionSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| min_length | [int32](#int32) |  | min_length is the minimum length for password, should no less than 8. |
| require_number | [bool](#bool) |  | require_number requires the password must contains at least one number. |
| require_letter | [bool](#bool) |  | require_letter requires the password must contains at least one letter, regardless of upper case or lower case |
| require_uppercase_letter | [bool](#bool) |  | require_uppercase_letter requires the password must contains at least one upper case letter. |
| require_special_character | [bool](#bool) |  | require_special_character requires the password must contains at least one special character. |
| require_reset_password_for_first_login | [bool](#bool) |  | require_reset_password_for_first_login requires users to reset their password after the 1st login. |
| password_rotation | [google.protobuf.Duration](#google-protobuf-Duration) |  | password_rotation requires users to reset their password after the duration. |






<a name="bytebase-v1-SCIMSetting"></a>

### SCIMSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| token | [string](#string) |  |  |






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
| catalog | [ColumnCatalog](#bytebase-v1-ColumnCatalog) |  |  |






<a name="bytebase-v1-SchemaTemplateSetting-TableTemplate"></a>

### SchemaTemplateSetting.TableTemplate



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| engine | [Engine](#bytebase-v1-Engine) |  |  |
| category | [string](#string) |  |  |
| table | [TableMetadata](#bytebase-v1-TableMetadata) |  |  |
| catalog | [TableCatalog](#bytebase-v1-TableCatalog) |  |  |






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
| name | [string](#string) |  | The resource name of the setting. Must be one of the following forms:

- `setting/{setting}` For example, &#34;settings/bb.branding.logo&#34; |
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
| app_im_setting_value | [AppIMSetting](#bytebase-v1-AppIMSetting) |  |  |
| workspace_profile_setting_value | [WorkspaceProfileSetting](#bytebase-v1-WorkspaceProfileSetting) |  | reserved 4; // was AgentPluginSetting agent_plugin_setting_value |
| workspace_approval_setting_value | [WorkspaceApprovalSetting](#bytebase-v1-WorkspaceApprovalSetting) |  |  |
| schema_template_setting_value | [SchemaTemplateSetting](#bytebase-v1-SchemaTemplateSetting) |  |  |
| data_classification_setting_value | [DataClassificationSetting](#bytebase-v1-DataClassificationSetting) |  |  |
| semantic_type_setting_value | [SemanticTypeSetting](#bytebase-v1-SemanticTypeSetting) |  |  |
| scim_setting | [SCIMSetting](#bytebase-v1-SCIMSetting) |  |  |
| password_restriction_setting | [PasswordRestrictionSetting](#bytebase-v1-PasswordRestrictionSetting) |  |  |
| ai_setting | [AISetting](#bytebase-v1-AISetting) |  |  |
| environment_setting | [EnvironmentSetting](#bytebase-v1-EnvironmentSetting) |  |  |






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

Support variables: source: the risk source, check the Source enum in the Risk message for the values, support &#34;==&#34; operator. level: the risk level, support 100 (low), 200 (moderate) and 300 (high), support &#34;==&#34; operator.

For examples: (source == &#34;DML&#34; &amp;&amp; level == 200) || (source == &#34;DDL&#34; &amp;&amp; level == 300) |






<a name="bytebase-v1-WorkspaceProfileSetting"></a>

### WorkspaceProfileSetting



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| external_url | [string](#string) |  | The external URL is used for sso authentication callback. |
| disallow_signup | [bool](#bool) |  | Disallow self-service signup, users can only be invited by the owner. |
| require_2fa | [bool](#bool) |  | Require 2FA for all users. |
| token_duration | [google.protobuf.Duration](#google-protobuf-Duration) |  | The duration for token. |
| announcement | [Announcement](#bytebase-v1-Announcement) |  | The setting of custom announcement |
| maximum_role_expiration | [google.protobuf.Duration](#google-protobuf-Duration) |  | The max duration for role expired. |
| domains | [string](#string) | repeated | The workspace domain, e.g. bytebase.com. |
| enforce_identity_domain | [bool](#bool) |  | Only user and group from the domains can be created and login. |
| database_change_mode | [DatabaseChangeMode](#bytebase-v1-DatabaseChangeMode) |  | The workspace database change mode. |
| disallow_password_signin | [bool](#bool) |  | Whether to disallow password signin. (Except workspace admins) |
| enable_metric_collection | [bool](#bool) |  | Whether to enable metric collection for the workspace. |





 


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
| AUTH_SECRET | 1 |  |
| BRANDING_LOGO | 2 |  |
| WORKSPACE_ID | 3 |  |
| WORKSPACE_PROFILE | 4 |  |
| WORKSPACE_APPROVAL | 5 |  |
| WORKSPACE_EXTERNAL_APPROVAL | 6 |  |
| ENTERPRISE_LICENSE | 7 |  |
| APP_IM | 8 |  |
| WATERMARK | 9 |  |
| AI | 10 |  |
| SCHEMA_TEMPLATE | 13 |  |
| DATA_CLASSIFICATION | 14 |  |
| SEMANTIC_TYPES | 15 |  |
| SCIM | 17 |  |
| PASSWORD_RESTRICTION | 18 |  |
| ENVIRONMENT | 19 |  |


 

 


<a name="bytebase-v1-SettingService"></a>

### SettingService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListSettings | [ListSettingsRequest](#bytebase-v1-ListSettingsRequest) | [ListSettingsResponse](#bytebase-v1-ListSettingsResponse) | Permissions required: bb.settings.list |
| GetSetting | [GetSettingRequest](#bytebase-v1-GetSettingRequest) | [Setting](#bytebase-v1-Setting) | Permissions required: bb.settings.get |
| UpdateSetting | [UpdateSettingRequest](#bytebase-v1-UpdateSettingRequest) | [Setting](#bytebase-v1-Setting) | Permissions required: bb.settings.set |

 



<a name="v1_user_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/user_service.proto



<a name="bytebase-v1-BatchGetUsersRequest"></a>

### BatchGetUsersRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| names | [string](#string) | repeated | The user names to retrieve. Format: users/{user uid or user email} |






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
| name | [string](#string) |  | The name of the user to delete. Format: users/{user} |






<a name="bytebase-v1-GetUserRequest"></a>

### GetUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the user to retrieve. Format: users/{user uid or user email} |






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
| profile | [User.Profile](#bytebase-v1-User-Profile) |  |  |
| groups | [string](#string) | repeated | The groups for the user. Format: groups/{email} |






<a name="bytebase-v1-User-Profile"></a>

### User.Profile



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| last_login_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| last_change_password_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| source | [string](#string) |  | source means where the user comes from. For now we support Entra ID SCIM sync, so the source could be Entra ID. |





 


<a name="bytebase-v1-UserType"></a>

### UserType


| Name | Number | Description |
| ---- | ------ | ----------- |
| USER_TYPE_UNSPECIFIED | 0 |  |
| USER | 1 |  |
| SYSTEM_BOT | 2 |  |
| SERVICE_ACCOUNT | 3 |  |


 

 


<a name="bytebase-v1-UserService"></a>

### UserService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetUser | [GetUserRequest](#bytebase-v1-GetUserRequest) | [User](#bytebase-v1-User) | Get the user. Any authenticated user can get the user. Permissions required: bb.users.get |
| BatchGetUsers | [BatchGetUsersRequest](#bytebase-v1-BatchGetUsersRequest) | [BatchGetUsersResponse](#bytebase-v1-BatchGetUsersResponse) | Get the users in batch. Any authenticated user can batch get users. Permissions required: bb.users.get |
| GetCurrentUser | [.google.protobuf.Empty](#google-protobuf-Empty) | [User](#bytebase-v1-User) | Get the current authenticated user. Permissions required: None |
| ListUsers | [ListUsersRequest](#bytebase-v1-ListUsersRequest) | [ListUsersResponse](#bytebase-v1-ListUsersResponse) | List all users. Any authenticated user can list users. Permissions required: bb.users.list |
| CreateUser | [CreateUserRequest](#bytebase-v1-CreateUserRequest) | [User](#bytebase-v1-User) | Create a user. When Disallow Signup is enabled, only the caller with bb.users.create on the workspace can create a user. Otherwise, any unauthenticated user can create a user. Permissions required: bb.users.create |
| UpdateUser | [UpdateUserRequest](#bytebase-v1-UpdateUserRequest) | [User](#bytebase-v1-User) | Only the user itself and the user with bb.users.update permission on the workspace can update the user. Permissions required: bb.users.update |
| DeleteUser | [DeleteUserRequest](#bytebase-v1-DeleteUserRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Only the user with bb.users.delete permission on the workspace can delete the user. The last remaining workspace admin cannot be deleted. Permissions required: bb.users.delete |
| UndeleteUser | [UndeleteUserRequest](#bytebase-v1-UndeleteUserRequest) | [User](#bytebase-v1-User) | Only the user with bb.users.undelete permission on the workspace can undelete the user. Permissions required: bb.users.undelete |

 



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
| password_restriction | [PasswordRestrictionSetting](#bytebase-v1-PasswordRestrictionSetting) |  |  |
| docker | [bool](#bool) |  | docker flag means if the Bytebase instance is running in docker. |
| user_stats | [ActuatorInfo.StatUser](#bytebase-v1-ActuatorInfo-StatUser) | repeated |  |
| activated_instance_count | [int32](#int32) |  |  |
| total_instance_count | [int32](#int32) |  |  |
| enable_sample | [bool](#bool) |  |  |






<a name="bytebase-v1-ActuatorInfo-StatUser"></a>

### ActuatorInfo.StatUser



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user_type | [UserType](#bytebase-v1-UserType) |  |  |
| state | [State](#bytebase-v1-State) |  |  |
| count | [int32](#int32) |  |  |






<a name="bytebase-v1-DeleteCacheRequest"></a>

### DeleteCacheRequest







<a name="bytebase-v1-GetActuatorInfoRequest"></a>

### GetActuatorInfoRequest







<a name="bytebase-v1-GetResourcePackageRequest"></a>

### GetResourcePackageRequest
The request message for getting the theme resource.






<a name="bytebase-v1-ResourcePackage"></a>

### ResourcePackage
The theme resources.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| logo | [bytes](#bytes) |  | The branding logo. |






<a name="bytebase-v1-SetupSampleRequest"></a>

### SetupSampleRequest







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
| GetActuatorInfo | [GetActuatorInfoRequest](#bytebase-v1-GetActuatorInfoRequest) | [ActuatorInfo](#bytebase-v1-ActuatorInfo) | Permissions required: None |
| UpdateActuatorInfo | [UpdateActuatorInfoRequest](#bytebase-v1-UpdateActuatorInfoRequest) | [ActuatorInfo](#bytebase-v1-ActuatorInfo) | Permissions required: bb.settings.set |
| SetupSample | [SetupSampleRequest](#bytebase-v1-SetupSampleRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Permissions required: bb.projects.create |
| DeleteCache | [DeleteCacheRequest](#bytebase-v1-DeleteCacheRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Permissions required: None |
| GetResourcePackage | [GetResourcePackageRequest](#bytebase-v1-GetResourcePackageRequest) | [ResourcePackage](#bytebase-v1-ResourcePackage) | Permissions required: None |

 



<a name="v1_iam_policy-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/iam_policy.proto



<a name="bytebase-v1-Binding"></a>

### Binding



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [string](#string) |  | The role that is assigned to the members. Format: roles/{role} |
| members | [string](#string) | repeated | Specifies the principals requesting access for a Bytebase resource. For users, the member should be: user:{email} For groups, the member should be: group:{email} |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this binding, only used in the project IAM policy. If the condition evaluates to true, then this binding applies to the current request. If the condition evaluates to false, then this binding does not apply to the current request. However, a different role binding might grant the same role to one or more of the principals in this binding. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Support variables: resource.database: the database full name in &#34;instances/{instance}/databases/{database}&#34; format, used by the &#34;roles/sqlEditorUser&#34; and &#34;roles/projectExporter&#34; role, support &#34;==&#34; operator. resource.schema: the schema name, used by the &#34;roles/sqlEditorUser&#34; and &#34;roles/projectExporter&#34; role, support &#34;==&#34; operator. resource.table: the table name, used by the &#34;roles/sqlEditorUser&#34; and &#34;roles/projectExporter&#34; role, support &#34;==&#34; operator. request.time: the expiration. Only support &#34;&lt;&#34; operation in `request.time &lt; timestamp(&#34;{ISO datetime string format}&#34;)`. request.row_limit: the maximum export rows, used by the &#34;roles/projectExporter&#34; role. Only support &#34;&lt;=&#34; operation.

For example: resource.database == &#34;instances/local-pg/databases/postgres&#34; &amp;&amp; resource.schema in [&#34;public&#34;,&#34;another_schema&#34;] resource.database == &#34;instances/local-pg/databases/bytebase&#34; &amp;&amp; resource.schema == &#34;public&#34; &amp;&amp; resource.table in [&#34;audit_log&#34;] request.time &lt; timestamp(&#34;2025-04-26T11:24:48.655Z&#34;) &amp;&amp; request.row_limit &lt;= 1000 |
| parsed_expr | [google.api.expr.v1alpha1.Expr](#google-api-expr-v1alpha1-Expr) |  | The parsed expression of the condition. |






<a name="bytebase-v1-BindingDelta"></a>

### BindingDelta
One delta entry for Binding. Each individual change (only one member in each
entry) to a binding will be a separate entry.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| action | [BindingDelta.Action](#bytebase-v1-BindingDelta-Action) |  | The action that was performed on a Binding. |
| role | [string](#string) |  | Role that is assigned to `members`. For example, `roles/projectOwner`. |
| member | [string](#string) |  | Follows the same format of Binding.members. |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this binding. |






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






<a name="bytebase-v1-PolicyDelta"></a>

### PolicyDelta
The difference delta between two policies.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| binding_deltas | [BindingDelta](#bytebase-v1-BindingDelta) | repeated | The delta for Bindings between two policies. |






<a name="bytebase-v1-SetIamPolicyRequest"></a>

### SetIamPolicyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| resource | [string](#string) |  | The name of the resource to set the IAM policy. Format: projects/{project} Format: workspaces/{workspace} |
| policy | [IamPolicy](#bytebase-v1-IamPolicy) |  |  |
| etag | [string](#string) |  | The current etag of the policy. |





 


<a name="bytebase-v1-BindingDelta-Action"></a>

### BindingDelta.Action
The type of action performed on a Binding in a policy.

| Name | Number | Description |
| ---- | ------ | ----------- |
| ACTION_UNSPECIFIED | 0 |  |
| ADD | 1 |  |
| REMOVE | 2 |  |


 

 

 



<a name="v1_audit_log_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/audit_log_service.proto



<a name="bytebase-v1-AuditData"></a>

### AuditData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policy_delta | [PolicyDelta](#bytebase-v1-PolicyDelta) |  |  |






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
| latency | [google.protobuf.Duration](#google-protobuf-Duration) |  | The latency of the RPC. |
| service_data | [google.protobuf.Any](#google-protobuf-Any) |  | service-specific data about the request, response, and other activities. |
| request_metadata | [RequestMetadata](#bytebase-v1-RequestMetadata) |  | Metadata about the operation. |






<a name="bytebase-v1-ExportAuditLogsRequest"></a>

### ExportAuditLogsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  |  |
| filter | [string](#string) |  | The filter of the log. It should be a valid CEL expression. Check the filter field in the SearchAuditLogsRequest message. |
| order_by | [string](#string) |  | The order by of the log. Only support order by create_time. For example: - order_by = &#34;create_time asc&#34; - order_by = &#34;create_time desc&#34; |
| format | [ExportFormat](#bytebase-v1-ExportFormat) |  | The export format. |
| page_size | [int32](#int32) |  | The maximum number of logs to return. The service may return fewer than this value. If unspecified, at most 10 log entries will be returned. The maximum value is 5000; values above 5000 will be coerced to 5000. |
| page_token | [string](#string) |  | A page token, received from a previous `ExportAuditLogs` call. Provide this to retrieve the subsequent page. |






<a name="bytebase-v1-ExportAuditLogsResponse"></a>

### ExportAuditLogsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| content | [bytes](#bytes) |  |  |
| next_page_token | [string](#string) |  | A token to retrieve next page of log entities. Pass this value in the page_token field in the subsequent call to retrieve the next page of log entities. |






<a name="bytebase-v1-RequestMetadata"></a>

### RequestMetadata
Metadata about the request.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| caller_ip | [string](#string) |  | The IP address of the caller. |
| caller_supplied_user_agent | [string](#string) |  | The user agent of the caller. This information is not authenticated and should be treated accordingly. |






<a name="bytebase-v1-SearchAuditLogsRequest"></a>

### SearchAuditLogsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  |  |
| filter | [string](#string) |  | The filter of the log. It should be a valid CEL expression. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filter: - method: the API name, can be found in the docs, should start with &#34;/bytebase.v1.&#34; prefix. For example &#34;/bytebase.v1.UserService/CreateUser&#34;. Support &#34;==&#34; operator. - severity: support &#34;==&#34; operator, check Severity enum in AuditLog message for values. - user: the actor, should in &#34;users/{email}&#34; format, support &#34;==&#34; operator. - create_time: support &#34;&gt;=&#34; and &#34;&lt;=&#34; operator.

For example: - filter = &#34;method == &#39;/bytebase.v1.SQLService/Query&#39;&#34; - filter = &#34;method == &#39;/bytebase.v1.SQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39;&#34; - filter = &#34;method == &#39;/bytebase.v1.SQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39; &amp;&amp; user == &#39;users/bb@bytebase.com&#39;&#34; - filter = &#34;method == &#39;/bytebase.v1.SQLService/Query&#39; &amp;&amp; severity == &#39;ERROR&#39; &amp;&amp; create_time &lt;= &#39;2021-01-01T00:00:00Z&#39; &amp;&amp; create_time &gt;= &#39;2020-01-01T00:00:00Z&#39;&#34; |
| order_by | [string](#string) |  | The order by of the log. Only support order by create_time. For example: - order_by = &#34;create_time asc&#34; - order_by = &#34;create_time desc&#34; |
| page_size | [int32](#int32) |  | The maximum number of logs to return. The service may return fewer than this value. If unspecified, at most 10 log entries will be returned. The maximum value is 5000; values above 5000 will be coerced to 5000. |
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
| SearchAuditLogs | [SearchAuditLogsRequest](#bytebase-v1-SearchAuditLogsRequest) | [SearchAuditLogsResponse](#bytebase-v1-SearchAuditLogsResponse) | Permissions required: None |
| ExportAuditLogs | [ExportAuditLogsRequest](#bytebase-v1-ExportAuditLogsRequest) | [ExportAuditLogsResponse](#bytebase-v1-ExportAuditLogsResponse) | Permissions required: bb.auditLogs.export |

 



<a name="v1_auth_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/auth_service.proto



<a name="bytebase-v1-IdentityProviderContext"></a>

### IdentityProviderContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| oauth2_context | [OAuth2IdentityProviderContext](#bytebase-v1-OAuth2IdentityProviderContext) |  |  |
| oidc_context | [OIDCIdentityProviderContext](#bytebase-v1-OIDCIdentityProviderContext) |  |  |






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
| require_reset_password | [bool](#bool) |  |  |
| user | [User](#bytebase-v1-User) |  | The user of successful login. |






<a name="bytebase-v1-LogoutRequest"></a>

### LogoutRequest







<a name="bytebase-v1-OAuth2IdentityProviderContext"></a>

### OAuth2IdentityProviderContext



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| code | [string](#string) |  |  |






<a name="bytebase-v1-OIDCIdentityProviderContext"></a>

### OIDCIdentityProviderContext






 

 

 


<a name="bytebase-v1-AuthService"></a>

### AuthService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Login | [LoginRequest](#bytebase-v1-LoginRequest) | [LoginResponse](#bytebase-v1-LoginResponse) | Permissions required: None |
| Logout | [LogoutRequest](#bytebase-v1-LogoutRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Permissions required: None |

 



<a name="v1_cel_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/cel_service.proto



<a name="bytebase-v1-BatchDeparseRequest"></a>

### BatchDeparseRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| expressions | [google.api.expr.v1alpha1.Expr](#google-api-expr-v1alpha1-Expr) | repeated |  |






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
| expressions | [google.api.expr.v1alpha1.Expr](#google-api-expr-v1alpha1-Expr) | repeated |  |





 

 

 


<a name="bytebase-v1-CelService"></a>

### CelService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| BatchParse | [BatchParseRequest](#bytebase-v1-BatchParseRequest) | [BatchParseResponse](#bytebase-v1-BatchParseResponse) | Permissions required: None |
| BatchDeparse | [BatchDeparseRequest](#bytebase-v1-BatchDeparseRequest) | [BatchDeparseResponse](#bytebase-v1-BatchDeparseResponse) | Permissions required: None |

 



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
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The last update time of the changelist. |
| changes | [Changelist.Change](#bytebase-v1-Changelist-Change) | repeated |  |






<a name="bytebase-v1-Changelist-Change"></a>

### Changelist.Change



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| sheet | [string](#string) |  | The name of a sheet. |
| source | [string](#string) |  | The source of origin. 1) changelog: instances/{instance}/databases/{database}/changelogs/{changelog}. 2) raw SQL if empty. |






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
| page_size | [int32](#int32) |  | Not used. The maximum number of databases to return. The service may return fewer than this value. If unspecified, at most 50 databases will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListDatabases` call. Provide this to retrieve the subsequent page.

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
| CreateChangelist | [CreateChangelistRequest](#bytebase-v1-CreateChangelistRequest) | [Changelist](#bytebase-v1-Changelist) | Permissions required: bb.changelists.create |
| GetChangelist | [GetChangelistRequest](#bytebase-v1-GetChangelistRequest) | [Changelist](#bytebase-v1-Changelist) | Permissions required: bb.changelists.get |
| ListChangelists | [ListChangelistsRequest](#bytebase-v1-ListChangelistsRequest) | [ListChangelistsResponse](#bytebase-v1-ListChangelistsResponse) | Permissions required: bb.changelists.list |
| UpdateChangelist | [UpdateChangelistRequest](#bytebase-v1-UpdateChangelistRequest) | [Changelist](#bytebase-v1-Changelist) | Permissions required: bb.changelists.update |
| DeleteChangelist | [DeleteChangelistRequest](#bytebase-v1-DeleteChangelistRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Permissions required: bb.changelists.delete |

 



<a name="v1_database_group_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/database_group_service.proto



<a name="bytebase-v1-CreateDatabaseGroupRequest"></a>

### CreateDatabaseGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource where this database group will be created. Format: projects/{project} |
| database_group | [DatabaseGroup](#bytebase-v1-DatabaseGroup) |  | The database group to create. |
| database_group_id | [string](#string) |  | The ID to use for the database group, which will become the final component of the database group&#39;s resource name.

This value should be 4-63 characters, and valid characters are /[a-z][0-9]-/. |
| validate_only | [bool](#bool) |  | If set, validate the create request and preview the full database group response, but do not actually create it. |






<a name="bytebase-v1-DatabaseGroup"></a>

### DatabaseGroup



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database group. Format: projects/{project}/databaseGroups/{databaseGroup} |
| title | [string](#string) |  | The short name used in actual databases specified by users. |
| database_expr | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this database group. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Support variables: resource.environment_name: the environment resource id. Support &#34;==&#34;, &#34;!=&#34;, &#34;in [XX]&#34;, &#34;!(in [xx])&#34; operations. resource.instance_id: the instance resource id. Support &#34;==&#34;, &#34;!=&#34;, &#34;in [XX]&#34;, &#34;!(in [xx])&#34;, &#34;contains&#34;, &#34;matches&#34;, &#34;startsWith&#34;, &#34;endsWith&#34; operations. resource.database_name: the database name. Support &#34;==&#34;, &#34;!=&#34;, &#34;in [XX]&#34;, &#34;!(in [xx])&#34;, &#34;contains&#34;, &#34;matches&#34;, &#34;startsWith&#34;, &#34;endsWith&#34; operations. All variables should join with &#34;&amp;&amp;&#34; condition.

For example: resource.environment_name == &#34;test&#34; &amp;&amp; resource.database_name.startsWith(&#34;sample_&#34;) |
| matched_databases | [DatabaseGroup.Database](#bytebase-v1-DatabaseGroup-Database) | repeated | The list of databases that match the database group condition. |






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






<a name="bytebase-v1-GetDatabaseGroupRequest"></a>

### GetDatabaseGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the database group to retrieve. Format: projects/{project}/databaseGroups/{databaseGroup} |
| view | [DatabaseGroupView](#bytebase-v1-DatabaseGroupView) |  | The view to return. Defaults to DATABASE_GROUP_VIEW_BASIC. |






<a name="bytebase-v1-ListDatabaseGroupsRequest"></a>

### ListDatabaseGroupsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent resource whose database groups are to be listed. Format: projects/{project} |
| page_size | [int32](#int32) |  | Not used. The maximum number of database groups to return. The service may return fewer than this value. If unspecified, at most 50 database groups will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListDatabaseGroups` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListDatabaseGroups` must match the call that provided the page token. |
| view | [DatabaseGroupView](#bytebase-v1-DatabaseGroupView) |  | The view to return. Defaults to DATABASE_GROUP_VIEW_BASIC. |






<a name="bytebase-v1-ListDatabaseGroupsResponse"></a>

### ListDatabaseGroupsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database_groups | [DatabaseGroup](#bytebase-v1-DatabaseGroup) | repeated | database_groups is the list of database groups. |
| next_page_token | [string](#string) |  | Not used. A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-UpdateDatabaseGroupRequest"></a>

### UpdateDatabaseGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database_group | [DatabaseGroup](#bytebase-v1-DatabaseGroup) |  | The database group to update.

The database group&#39;s `name` field is used to identify the database group to update. Format: projects/{project}/databaseGroups/{databaseGroup} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |





 


<a name="bytebase-v1-DatabaseGroupView"></a>

### DatabaseGroupView


| Name | Number | Description |
| ---- | ------ | ----------- |
| DATABASE_GROUP_VIEW_UNSPECIFIED | 0 | The default / unset value. The API will default to the BASIC view. |
| DATABASE_GROUP_VIEW_BASIC | 1 | Include basic information about the database group, but exclude the list of matched databases and unmatched databases. |
| DATABASE_GROUP_VIEW_FULL | 2 | Include everything. |


 

 


<a name="bytebase-v1-DatabaseGroupService"></a>

### DatabaseGroupService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListDatabaseGroups | [ListDatabaseGroupsRequest](#bytebase-v1-ListDatabaseGroupsRequest) | [ListDatabaseGroupsResponse](#bytebase-v1-ListDatabaseGroupsResponse) | Permissions required: bb.projects.get |
| GetDatabaseGroup | [GetDatabaseGroupRequest](#bytebase-v1-GetDatabaseGroupRequest) | [DatabaseGroup](#bytebase-v1-DatabaseGroup) | Permissions required: bb.projects.get |
| CreateDatabaseGroup | [CreateDatabaseGroupRequest](#bytebase-v1-CreateDatabaseGroupRequest) | [DatabaseGroup](#bytebase-v1-DatabaseGroup) | Permissions required: bb.projects.update |
| UpdateDatabaseGroup | [UpdateDatabaseGroupRequest](#bytebase-v1-UpdateDatabaseGroupRequest) | [DatabaseGroup](#bytebase-v1-DatabaseGroup) | Permissions required: bb.projects.update |
| DeleteDatabaseGroup | [DeleteDatabaseGroupRequest](#bytebase-v1-DeleteDatabaseGroupRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Permissions required: bb.projects.update |

 



<a name="v1_group_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/group_service.proto



<a name="bytebase-v1-BatchGetGroupsRequest"></a>

### BatchGetGroupsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| names | [string](#string) | repeated | The group names to retrieve. Format: groups/{email} |






<a name="bytebase-v1-BatchGetGroupsResponse"></a>

### BatchGetGroupsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| groups | [Group](#bytebase-v1-Group) | repeated | The groups from the specified request. |






<a name="bytebase-v1-CreateGroupRequest"></a>

### CreateGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [Group](#bytebase-v1-Group) |  | The group to create. |
| group_email | [string](#string) |  | The email to use for the group, which will become the final component of the group&#39;s resource name. |






<a name="bytebase-v1-DeleteGroupRequest"></a>

### DeleteGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the group to delete. Format: groups/{email} |






<a name="bytebase-v1-GetGroupRequest"></a>

### GetGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the group to retrieve. Format: groups/{email} |






<a name="bytebase-v1-Group"></a>

### Group



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the group to retrieve. Format: groups/{group}, group is an email. |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| members | [GroupMember](#bytebase-v1-GroupMember) | repeated |  |
| source | [string](#string) |  | source means where the group comes from. For now we support Entra ID SCIM sync, so the source could be Entra ID. |






<a name="bytebase-v1-GroupMember"></a>

### GroupMember



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| member | [string](#string) |  | Member is the principal who belong to this group.

Format: users/hello@world.com |
| role | [GroupMember.Role](#bytebase-v1-GroupMember-Role) |  |  |






<a name="bytebase-v1-ListGroupsRequest"></a>

### ListGroupsRequest



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



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| groups | [Group](#bytebase-v1-Group) | repeated | The groups from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-UpdateGroupRequest"></a>

### UpdateGroupRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group | [Group](#bytebase-v1-Group) |  | The group to update.

The group&#39;s `name` field is used to identify the group to update. Format: groups/{email} |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the group is not found, a new group will be created. In this situation, `update_mask` is ignored. |





 


<a name="bytebase-v1-GroupMember-Role"></a>

### GroupMember.Role


| Name | Number | Description |
| ---- | ------ | ----------- |
| ROLE_UNSPECIFIED | 0 |  |
| OWNER | 1 |  |
| MEMBER | 2 |  |


 

 


<a name="bytebase-v1-GroupService"></a>

### GroupService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetGroup | [GetGroupRequest](#bytebase-v1-GetGroupRequest) | [Group](#bytebase-v1-Group) | Permissions required: bb.groups.get |
| BatchGetGroups | [BatchGetGroupsRequest](#bytebase-v1-BatchGetGroupsRequest) | [BatchGetGroupsResponse](#bytebase-v1-BatchGetGroupsResponse) | Get the groups in batch. Permissions required: bb.groups.get |
| ListGroups | [ListGroupsRequest](#bytebase-v1-ListGroupsRequest) | [ListGroupsResponse](#bytebase-v1-ListGroupsResponse) | Permissions required: bb.groups.list |
| CreateGroup | [CreateGroupRequest](#bytebase-v1-CreateGroupRequest) | [Group](#bytebase-v1-Group) | Permissions required: bb.groups.create |
| UpdateGroup | [UpdateGroupRequest](#bytebase-v1-UpdateGroupRequest) | [Group](#bytebase-v1-Group) | UpdateGroup updates the group. Users with &#34;bb.groups.update&#34; permission on the workspace or the group owner can access this method. Permissions required: bb.groups.update |
| DeleteGroup | [DeleteGroupRequest](#bytebase-v1-DeleteGroupRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Permissions required: bb.groups.delete |

 



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
| name | [string](#string) |  |  |






<a name="bytebase-v1-IdentityProvider"></a>

### IdentityProvider



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the identity provider. Format: idps/{idp} |
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
| security_protocol | [LDAPIdentityProviderConfig.SecurityProtocol](#bytebase-v1-LDAPIdentityProviderConfig-SecurityProtocol) |  | SecurityProtocol is the security protocol to be used for establishing connections with the LDAP server. |
| field_mapping | [FieldMapping](#bytebase-v1-FieldMapping) |  | FieldMapping is the mapping of the user attributes returned by the LDAP server. |






<a name="bytebase-v1-ListIdentityProvidersRequest"></a>

### ListIdentityProvidersRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Not used. The maximum number of identity providers to return. The service may return fewer than this value. If unspecified, at most 10 will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListIdentityProviders` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListIdentityProviders` must match the call that provided the page token. |






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
| scopes | [string](#string) | repeated | The scopes that the OIDC provider supports. Should be fetched from the well-known configuration file of the OIDC provider. |
| field_mapping | [FieldMapping](#bytebase-v1-FieldMapping) |  |  |
| skip_tls_verify | [bool](#bool) |  |  |
| auth_style | [OAuth2AuthStyle](#bytebase-v1-OAuth2AuthStyle) |  |  |
| auth_endpoint | [string](#string) |  | The authorization endpoint of the OIDC provider. Should be fetched from the well-known configuration file of the OIDC provider. |






<a name="bytebase-v1-TestIdentityProviderRequest"></a>

### TestIdentityProviderRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| identity_provider | [IdentityProvider](#bytebase-v1-IdentityProvider) |  | The identity provider to test connection including uncreated. |
| oauth2_context | [OAuth2IdentityProviderTestRequestContext](#bytebase-v1-OAuth2IdentityProviderTestRequestContext) |  |  |






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





 


<a name="bytebase-v1-IdentityProviderType"></a>

### IdentityProviderType


| Name | Number | Description |
| ---- | ------ | ----------- |
| IDENTITY_PROVIDER_TYPE_UNSPECIFIED | 0 |  |
| OAUTH2 | 1 |  |
| OIDC | 2 |  |
| LDAP | 3 |  |



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


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetIdentityProvider | [GetIdentityProviderRequest](#bytebase-v1-GetIdentityProviderRequest) | [IdentityProvider](#bytebase-v1-IdentityProvider) | Permissions required: bb.identityProviders.get |
| ListIdentityProviders | [ListIdentityProvidersRequest](#bytebase-v1-ListIdentityProvidersRequest) | [ListIdentityProvidersResponse](#bytebase-v1-ListIdentityProvidersResponse) | Permissions required: None |
| CreateIdentityProvider | [CreateIdentityProviderRequest](#bytebase-v1-CreateIdentityProviderRequest) | [IdentityProvider](#bytebase-v1-IdentityProvider) | Permissions required: bb.identityProviders.create |
| UpdateIdentityProvider | [UpdateIdentityProviderRequest](#bytebase-v1-UpdateIdentityProviderRequest) | [IdentityProvider](#bytebase-v1-IdentityProvider) | Permissions required: bb.identityProviders.update |
| DeleteIdentityProvider | [DeleteIdentityProviderRequest](#bytebase-v1-DeleteIdentityProviderRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Permissions required: bb.identityProviders.delete |
| TestIdentityProvider | [TestIdentityProviderRequest](#bytebase-v1-TestIdentityProviderRequest) | [TestIdentityProviderResponse](#bytebase-v1-TestIdentityProviderResponse) | Permissions required: bb.identityProviders.update |

 



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






<a name="bytebase-v1-DataSourceQueryPolicy"></a>

### DataSourceQueryPolicy
DataSourceQueryPolicy is the policy configuration for running statements in the SQL editor.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| admin_data_source_restriction | [DataSourceQueryPolicy.Restriction](#bytebase-v1-DataSourceQueryPolicy-Restriction) |  |  |
| disallow_ddl | [bool](#bool) |  | Disallow running DDL statements in the SQL editor. |
| disallow_dml | [bool](#bool) |  | Disallow running DML statements in the SQL editor. |






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
| page_size | [int32](#int32) |  | Not used. The maximum number of policies to return. The service may return fewer than this value. If unspecified, at most 10 policies will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListPolicies` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListPolicies` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Show deleted policies if specified. |






<a name="bytebase-v1-ListPoliciesResponse"></a>

### ListPoliciesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| policies | [Policy](#bytebase-v1-Policy) | repeated | The policies from the specified request. |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






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
| member | [string](#string) |  | Member is the principal who bind to this exception policy instance.

- `user:{email}`: An email address that represents a specific Bytebase account. For example, `alice@example.com`. - `group:{email}`: An email address for group. |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with this exception policy instance. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec If the condition is empty, means the user can access all databases without expiration.

Support variables: resource.instance_id: the instance resource id. Only support &#34;==&#34; operation. resource.database_name: the database name. Only support &#34;==&#34; operation. resource.schema_name: the schema name. Only support &#34;==&#34; operation. resource.table_name: the table name. Only support &#34;==&#34; operation. resource.column_name: the column name. Only support &#34;==&#34; operation. request.time: the expiration. Only support &#34;&lt;&#34; operation in `request.time &lt; timestamp(&#34;{ISO datetime string format}&#34;)` All variables should join with &#34;&amp;&amp;&#34; condition.

For example: resource.instance_id == &#34;local&#34; &amp;&amp; resource.database_name == &#34;employee&#34; &amp;&amp; request.time &lt; timestamp(&#34;2025-04-30T11:10:39.000Z&#34;) resource.instance_id == &#34;local&#34; &amp;&amp; resource.database_name == &#34;employee&#34; |






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
| condition | [google.type.Expr](#google-type-Expr) |  | The condition for the masking rule. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Support variables: environment_id: the environment resource id. project_id: the project resource id. instance_id: the instance resource id. database_name: the database name. table_name: the table name. column_name: the column name. classification_level: the classification level.

Each variable support following operations: ==: the value equals the target. !=: the value not equals the target. in: the value matches one of the targets. !(in): the value not matches any of the targets.

For example: environment_id == &#34;test&#34; &amp;&amp; project_id == &#34;sample-project&#34; instance_id == &#34;sample-instance&#34; &amp;&amp; database_name == &#34;employee&#34; &amp;&amp; table_name in [&#34;table1&#34;, &#34;table2&#34;] environment_id != &#34;test&#34; || !(project_id in [&#34;poject1&#34;, &#34;prject2&#34;]) instance_id == &#34;sample-instance&#34; &amp;&amp; (database_name == &#34;db1&#34; || database_name == &#34;db2&#34;) |
| semantic_type | [string](#string) |  |  |






<a name="bytebase-v1-Policy"></a>

### Policy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the policy. Format: {resource name}/policies/{policy type} Workspace resource name: &#34;&#34;. Environment resource name: environments/environment-id. Instance resource name: instances/instance-id. Database resource name: instances/instance-id/databases/database-name. |
| inherit_from_parent | [bool](#bool) |  |  |
| type | [PolicyType](#bytebase-v1-PolicyType) |  |  |
| rollout_policy | [RolloutPolicy](#bytebase-v1-RolloutPolicy) |  |  |
| disable_copy_data_policy | [DisableCopyDataPolicy](#bytebase-v1-DisableCopyDataPolicy) |  |  |
| masking_rule_policy | [MaskingRulePolicy](#bytebase-v1-MaskingRulePolicy) |  |  |
| masking_exception_policy | [MaskingExceptionPolicy](#bytebase-v1-MaskingExceptionPolicy) |  |  |
| restrict_issue_creation_for_sql_review_policy | [RestrictIssueCreationForSQLReviewPolicy](#bytebase-v1-RestrictIssueCreationForSQLReviewPolicy) |  |  |
| tag_policy | [TagPolicy](#bytebase-v1-TagPolicy) |  |  |
| data_source_query_policy | [DataSourceQueryPolicy](#bytebase-v1-DataSourceQueryPolicy) |  |  |
| query_data_policy | [QueryDataPolicy](#bytebase-v1-QueryDataPolicy) |  |  |
| enforce | [bool](#bool) |  |  |
| resource_type | [PolicyResourceType](#bytebase-v1-PolicyResourceType) |  | The resource type for the policy. |






<a name="bytebase-v1-QueryDataPolicy"></a>

### QueryDataPolicy
QueryDataPolicy is the policy configuration for querying data.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| timeout | [google.protobuf.Duration](#google-protobuf-Duration) |  | The query timeout duration. |
| disable_export | [bool](#bool) |  | Disable export data in the SQL editor |
| maximum_result_size | [int64](#int64) |  | The size limit in bytes. The default value is 100MB, we will use the default value if the setting not exists, or the limit &lt;= 0. |
| maximum_result_rows | [int32](#int32) |  | The return rows limit. The default value is -1, means no limit. |






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
| roles | [string](#string) | repeated |  |
| issue_roles | [string](#string) | repeated | roles/LAST_APPROVER roles/CREATOR |






<a name="bytebase-v1-SQLReviewRule"></a>

### SQLReviewRule
The SQL review rules. Check the SQL_REVIEW_RULES_DOCUMENTATION.md for details.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| type | [string](#string) |  |  |
| level | [SQLReviewRuleLevel](#bytebase-v1-SQLReviewRuleLevel) |  |  |
| payload | [string](#string) |  | The payload is a JSON string that varies by rule type. |
| engine | [Engine](#bytebase-v1-Engine) |  |  |
| comment | [string](#string) |  |  |






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





 


<a name="bytebase-v1-DataSourceQueryPolicy-Restriction"></a>

### DataSourceQueryPolicy.Restriction


| Name | Number | Description |
| ---- | ------ | ----------- |
| RESTRICTION_UNSPECIFIED | 0 |  |
| FALLBACK | 1 | Allow to query admin data sources when there is no read-only data source. |
| DISALLOW | 2 | Disallow to query admin data sources. |



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



<a name="bytebase-v1-PolicyType"></a>

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
| DATA_QUERY | 16 |  |



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
| GetPolicy | [GetPolicyRequest](#bytebase-v1-GetPolicyRequest) | [Policy](#bytebase-v1-Policy) | Permissions required: bb.policies.get |
| ListPolicies | [ListPoliciesRequest](#bytebase-v1-ListPoliciesRequest) | [ListPoliciesResponse](#bytebase-v1-ListPoliciesResponse) | Permissions required: bb.policies.list |
| CreatePolicy | [CreatePolicyRequest](#bytebase-v1-CreatePolicyRequest) | [Policy](#bytebase-v1-Policy) | Permissions required: bb.policies.create |
| UpdatePolicy | [UpdatePolicyRequest](#bytebase-v1-UpdatePolicyRequest) | [Policy](#bytebase-v1-Policy) | Permissions required: bb.policies.update |
| DeletePolicy | [DeletePolicyRequest](#bytebase-v1-DeletePolicyRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Permissions required: bb.policies.delete |

 



<a name="v1_plan_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/plan_service.proto



<a name="bytebase-v1-BatchCancelPlanCheckRunsRequest"></a>

### BatchCancelPlanCheckRunsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The name of the parent of the planChecks. Format: projects/{project}/plans/{plan} |
| plan_check_runs | [string](#string) | repeated | TODO(d): update this API. The planCheckRuns to cancel. Format: projects/{project}/plans/{plan}/planCheckRuns/{planCheckRun} |






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
| page_size | [int32](#int32) |  | Not used. The maximum number of plan check runs to return. The service may return fewer than this value. If unspecified, at most 10 plan check runs will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListPlanCheckRuns` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListPlanCheckRuns` must match the call that provided the page token. |
| latest_only | [bool](#bool) |  | If set to true, only the latest plan check run will be returned. |
| filter | [string](#string) |  | Filter is used to filter plan check runs returned in the list. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filters: - status: the plan check run status, support &#34;==&#34; and &#34;in&#34; operator, check the Status enum in the PlanCheckRun message for the values. - result_status: the plan check run result status, support &#34;==&#34; and &#34;in&#34; operator, check the Result.Status enum in the PlanCheckRun message for the values.

For example: status in [&#34;DONE&#34;, &#34;FAILED&#34;] status == &#34;RUNNING&#34; result_status in [&#34;SUCCESS&#34;, &#34;ERROR&#34;] result_status == &#34;WARNING&#34; |






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
| page_size | [int32](#int32) |  | The maximum number of plans to return. The service may return fewer than this value. If unspecified, at most 10 plans will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
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
| state | [State](#bytebase-v1-State) |  | The state of the plan. |
| issue | [string](#string) |  | The issue associated with the plan. Can be empty. Format: projects/{project}/issues/{issue} |
| rollout | [string](#string) |  | The rollout associated with the plan. Can be empty. Format: projects/{project}/rollouts/{rollout} |
| title | [string](#string) |  |  |
| description | [string](#string) |  |  |
| specs | [Plan.Spec](#bytebase-v1-Plan-Spec) | repeated |  |
| creator | [string](#string) |  | Format: users/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| plan_check_run_status_count | [Plan.PlanCheckRunStatusCountEntry](#bytebase-v1-Plan-PlanCheckRunStatusCountEntry) | repeated | The status count of the latest plan check runs. Keys are: - SUCCESS - WARNING - ERROR - RUNNING |
| deployment | [Plan.Deployment](#bytebase-v1-Plan-Deployment) |  |  |






<a name="bytebase-v1-Plan-ChangeDatabaseConfig"></a>

### Plan.ChangeDatabaseConfig



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| targets | [string](#string) | repeated | The list of targets. Multi-database format: [instances/{instance-id}/databases/{database-name}]. Single database group format: [projects/{project}/databaseGroups/{databaseGroup}]. |
| sheet | [string](#string) |  | The resource name of the sheet. Format: projects/{project}/sheets/{sheet} |
| release | [string](#string) |  | The resource name of the release. Format: projects/{project}/releases/{release} |
| type | [Plan.ChangeDatabaseConfig.Type](#bytebase-v1-Plan-ChangeDatabaseConfig-Type) |  |  |
| ghost_flags | [Plan.ChangeDatabaseConfig.GhostFlagsEntry](#bytebase-v1-Plan-ChangeDatabaseConfig-GhostFlagsEntry) | repeated |  |
| enable_prior_backup | [bool](#bool) |  | If set, a backup of the modified data will be created automatically before any changes are applied. |






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






<a name="bytebase-v1-Plan-Deployment"></a>

### Plan.Deployment



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| environments | [string](#string) | repeated | The environments deploy order. |
| database_group_mappings | [Plan.Deployment.DatabaseGroupMapping](#bytebase-v1-Plan-Deployment-DatabaseGroupMapping) | repeated | The database group mapping. |






<a name="bytebase-v1-Plan-Deployment-DatabaseGroupMapping"></a>

### Plan.Deployment.DatabaseGroupMapping



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database_group | [string](#string) |  | Format: projects/{project}/databaseGroups/{databaseGroup}. |
| databases | [string](#string) | repeated | Format: instances/{instance-id}/databases/{database-name}. |






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
| name | [string](#string) |  | Format: projects/{project}/plans/{plan}/planCheckRuns/{planCheckRun} |
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
| start_position | [Position](#bytebase-v1-Position) |  | 1-based Position of the SQL statement. To supersede `line` and `column` above. |
| end_position | [Position](#bytebase-v1-Position) |  |  |






<a name="bytebase-v1-PlanCheckRun-Result-SqlSummaryReport"></a>

### PlanCheckRun.Result.SqlSummaryReport



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| statement_types | [string](#string) | repeated | statement_types are the types of statements that are found in the sql. |
| affected_rows | [int64](#int64) |  |  |
| changed_resources | [ChangedResources](#bytebase-v1-ChangedResources) |  |  |






<a name="bytebase-v1-RunPlanChecksRequest"></a>

### RunPlanChecksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The plan to run plan checks. Format: projects/{project}/plans/{plan} |
| spec_id | [string](#string) | optional | The UUID of the specific spec to run plan checks for. This should match the spec.id field in Plan.Spec. If not set, all specs in the plan will be used. |






<a name="bytebase-v1-RunPlanChecksResponse"></a>

### RunPlanChecksResponse







<a name="bytebase-v1-SearchPlansRequest"></a>

### SearchPlansRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of plans. Format: projects/{project} Use &#34;projects/-&#34; to list all plans from all projects. |
| page_size | [int32](#int32) |  | The maximum number of plans to return. The service may return fewer than this value. If unspecified, at most 10 plans will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `SearchPlans` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `SearchPlans` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter plans returned in the list. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filters: - creator: the plan creator full name in &#34;users/{email or id}&#34; format, support &#34;==&#34; operator. - create_time: the plan create time in &#34;2006-01-02T15:04:05Z07:00&#34; format, support &#34;&gt;=&#34; or &#34;&lt;=&#34; operator. - has_pipeline: the plan has pipeline or not, support &#34;==&#34; operator, the value should be &#34;true&#34; or &#34;false&#34;. - has_issue: the plan has issue or not, support &#34;==&#34; operator, the value should be &#34;true&#34; or &#34;false&#34;. - title: the plan title, support &#34;==&#34; operator for exact match and &#34;.matches()&#34; operator for case-insensitive substring match. - spec_type: the plan spec config type, support &#34;==&#34; operator, the value should be &#34;create_database_config&#34;, &#34;change_database_config&#34;, or &#34;export_data_config&#34;. - state: the plan state, support &#34;==&#34; operator, the value should be &#34;ACTIVE&#34; or &#34;DELETED&#34;.

For example: creator == &#34;users/ed@bytebase.com&#34; &amp;&amp; create_time &gt;= &#34;2025-01-02T15:04:05Z07:00&#34; has_pipeline == false &amp;&amp; has_issue == true title == &#34;My Plan&#34; title.matches(&#34;database migration&#34;) spec_type == &#34;change_database_config&#34; state == &#34;ACTIVE&#34; |






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
| GetPlan | [GetPlanRequest](#bytebase-v1-GetPlanRequest) | [Plan](#bytebase-v1-Plan) | Permissions required: bb.plans.get |
| ListPlans | [ListPlansRequest](#bytebase-v1-ListPlansRequest) | [ListPlansResponse](#bytebase-v1-ListPlansResponse) | Permissions required: bb.plans.list |
| SearchPlans | [SearchPlansRequest](#bytebase-v1-SearchPlansRequest) | [SearchPlansResponse](#bytebase-v1-SearchPlansResponse) | Search for plans that the caller has the bb.plans.get permission on and also satisfy the specified filter &amp; query. Permissions required: bb.plans.get |
| CreatePlan | [CreatePlanRequest](#bytebase-v1-CreatePlanRequest) | [Plan](#bytebase-v1-Plan) | Permissions required: bb.plans.create |
| UpdatePlan | [UpdatePlanRequest](#bytebase-v1-UpdatePlanRequest) | [Plan](#bytebase-v1-Plan) | UpdatePlan updates the plan. The plan creator and the user with bb.plans.update permission on the project can update the plan. Permissions required: bb.plans.update |
| ListPlanCheckRuns | [ListPlanCheckRunsRequest](#bytebase-v1-ListPlanCheckRunsRequest) | [ListPlanCheckRunsResponse](#bytebase-v1-ListPlanCheckRunsResponse) | Permissions required: bb.planCheckRuns.list |
| RunPlanChecks | [RunPlanChecksRequest](#bytebase-v1-RunPlanChecksRequest) | [RunPlanChecksResponse](#bytebase-v1-RunPlanChecksResponse) | Permissions required: bb.planCheckRuns.run |
| BatchCancelPlanCheckRuns | [BatchCancelPlanCheckRunsRequest](#bytebase-v1-BatchCancelPlanCheckRunsRequest) | [BatchCancelPlanCheckRunsResponse](#bytebase-v1-BatchCancelPlanCheckRunsResponse) | Permissions required: bb.planCheckRuns.run |

 



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






<a name="bytebase-v1-BatchDeleteProjectsRequest"></a>

### BatchDeleteProjectsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| names | [string](#string) | repeated | The names of the projects to delete. Format: projects/{project} |
| force | [bool](#bool) |  | If set to true, any databases from this project will be moved to default project. Sheets are not moved since BYTEBASE_ARTIFACT sheets belong to the issue and issue project. Open issues will remain open but associated with the deleted project. If set to false, the operation will fail if the project has databases or open issues. |






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



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| value | [string](#string) |  |  |
| color | [string](#string) |  |  |
| group | [string](#string) |  |  |






<a name="bytebase-v1-ListProjectsRequest"></a>

### ListProjectsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | The maximum number of projects to return. The service may return fewer than this value. If unspecified, at most 10 projects will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListProjects` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListProjects` must match the call that provided the page token. |
| show_deleted | [bool](#bool) |  | Show deleted projects if specified. |
| filter | [string](#string) |  | Filter the project. Check filter for SearchProjectsRequest for details. |






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
| state | [State](#bytebase-v1-State) |  |  |
| title | [string](#string) |  | The title or name of a project. It&#39;s not unique within the workspace. |
| webhooks | [Webhook](#bytebase-v1-Webhook) | repeated |  |
| data_classification_config_id | [string](#string) |  |  |
| issue_labels | [Label](#bytebase-v1-Label) | repeated |  |
| force_issue_labels | [bool](#bool) |  | Force issue labels to be used when creating an issue. |
| allow_modify_statement | [bool](#bool) |  | Allow modifying statement after issue is created. |
| auto_resolve_issue | [bool](#bool) |  | Enable auto resolve issue. |
| enforce_issue_title | [bool](#bool) |  | Enforce issue title created by user instead of generated by Bytebase. |
| auto_enable_backup | [bool](#bool) |  | Whether to automatically enable backup. |
| skip_backup_errors | [bool](#bool) |  | Whether to skip backup errors and continue the data migration. |
| postgres_database_tenant_mode | [bool](#bool) |  | Whether to enable the database tenant mode for PostgreSQL. If enabled, the issue will be created with the prepend &#34;set role &lt;db_owner&gt;&#34; statement. |
| allow_self_approval | [bool](#bool) |  | Whether to allow the issue creator to self-approve the issue. |
| execution_retry_policy | [Project.ExecutionRetryPolicy](#bytebase-v1-Project-ExecutionRetryPolicy) |  | Execution retry policy for the task run. |
| ci_sampling_size | [int32](#int32) |  | The maximum databases of rows to sample during CI data validation. Without specification, sampling is disabled, resulting in a full validation. |
| parallel_tasks_per_rollout | [int32](#int32) |  | The maximum number of parallel tasks to run during the rollout. |
| labels | [Project.LabelsEntry](#bytebase-v1-Project-LabelsEntry) | repeated | Labels are key-value pairs that can be attached to the project. For example, { &#34;environment&#34;: &#34;production&#34;, &#34;team&#34;: &#34;backend&#34; } |






<a name="bytebase-v1-Project-ExecutionRetryPolicy"></a>

### Project.ExecutionRetryPolicy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| maximum_retries | [int32](#int32) |  | The maximum number of retries for the lock timeout issue. |






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

Supported filters: - name: the project name, support &#34;==&#34; and &#34;.matches()&#34; operator. - resource_id: the project id, support &#34;==&#34; and &#34;.matches()&#34; operator. - exclude_default: if not include the default project, should be &#34;true&#34; or &#34;false&#34;, support &#34;==&#34; operator. - state: check the State enum for the values, support &#34;==&#34; operator.

For example: name = &#34;project name&#34; name.matches(&#34;project name&#34;) resource_id = &#34;project id&#34; resource_id.matches(&#34;project id&#34;) exclude_default == true state == &#34;DELETED&#34; You can combine filter conditions like: name = &#34;project name&#34; &amp;&amp; resource_id.matches(&#34;project id&#34;) name.matches(&#34;project name&#34;) || resource_id = &#34;project id&#34; |
| page_size | [int32](#int32) |  | The maximum number of projects to return. The service may return fewer than this value. If unspecified, at most 10 projects will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `SearchProjects` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `SearchProjects` must match the call that provided the page token. |






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
| NOTIFY_ISSUE_APPROVED | 23 | Notifications via webhooks.

NOTIFY_ISSUE_APPROVED represents the issue approved notification. |
| NOTIFY_PIPELINE_ROLLOUT | 24 | NOTIFY_PIPELINE_ROLLOUT represents the pipeline rollout notification. |
| ISSUE_CREATE | 1 | Issue related activity types.

ISSUE_CREATE represents creating an issue. |
| ISSUE_COMMENT_CREATE | 2 | ISSUE_COMMENT_CREATE represents commenting on an issue. |
| ISSUE_FIELD_UPDATE | 3 | ISSUE_FIELD_UPDATE represents updating the issue field, likes title, description, etc. |
| ISSUE_STATUS_UPDATE | 4 | ISSUE_STATUS_UPDATE represents the issue status change, including OPEN, CLOSE, CANCEL fow now. |
| ISSUE_APPROVAL_NOTIFY | 21 | ISSUE_APPROVAL_NOTIFY is the type for notifying issue approval. |
| ISSUE_PIPELINE_STAGE_STATUS_UPDATE | 5 | ISSUE_PIPELINE_STAGE_STATUS_UPDATE represents the pipeline stage status change, including BEGIN, END for now. |
| ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE | 22 | ISSUE_PIPELINE_TASK_RUN_STATUS_UPDATE represents the pipeline task run status change, including PENDING, RUNNING, DONE, FAILED, CANCELED. |



<a name="bytebase-v1-Webhook-Type"></a>

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


 

 


<a name="bytebase-v1-ProjectService"></a>

### ProjectService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetProject | [GetProjectRequest](#bytebase-v1-GetProjectRequest) | [Project](#bytebase-v1-Project) | GetProject retrieves a project by name. Users with &#34;bb.projects.get&#34; permission on the workspace or the project owner can access this method. Permissions required: bb.projects.get |
| ListProjects | [ListProjectsRequest](#bytebase-v1-ListProjectsRequest) | [ListProjectsResponse](#bytebase-v1-ListProjectsResponse) | Permissions required: bb.projects.list |
| SearchProjects | [SearchProjectsRequest](#bytebase-v1-SearchProjectsRequest) | [SearchProjectsResponse](#bytebase-v1-SearchProjectsResponse) | Permissions required: None |
| CreateProject | [CreateProjectRequest](#bytebase-v1-CreateProjectRequest) | [Project](#bytebase-v1-Project) | Permissions required: bb.projects.create |
| UpdateProject | [UpdateProjectRequest](#bytebase-v1-UpdateProjectRequest) | [Project](#bytebase-v1-Project) | Permissions required: bb.projects.update |
| DeleteProject | [DeleteProjectRequest](#bytebase-v1-DeleteProjectRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Permissions required: bb.projects.delete |
| UndeleteProject | [UndeleteProjectRequest](#bytebase-v1-UndeleteProjectRequest) | [Project](#bytebase-v1-Project) | Permissions required: bb.projects.undelete |
| BatchDeleteProjects | [BatchDeleteProjectsRequest](#bytebase-v1-BatchDeleteProjectsRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Permissions required: bb.projects.delete |
| GetIamPolicy | [GetIamPolicyRequest](#bytebase-v1-GetIamPolicyRequest) | [IamPolicy](#bytebase-v1-IamPolicy) | Permissions required: bb.projects.getIamPolicy |
| BatchGetIamPolicy | [BatchGetIamPolicyRequest](#bytebase-v1-BatchGetIamPolicyRequest) | [BatchGetIamPolicyResponse](#bytebase-v1-BatchGetIamPolicyResponse) | Deprecated. Permissions required: bb.projects.getIamPolicy |
| SetIamPolicy | [SetIamPolicyRequest](#bytebase-v1-SetIamPolicyRequest) | [IamPolicy](#bytebase-v1-IamPolicy) | Permissions required: bb.projects.setIamPolicy |
| AddWebhook | [AddWebhookRequest](#bytebase-v1-AddWebhookRequest) | [Project](#bytebase-v1-Project) | Permissions required: bb.projects.update |
| UpdateWebhook | [UpdateWebhookRequest](#bytebase-v1-UpdateWebhookRequest) | [Project](#bytebase-v1-Project) | Permissions required: bb.projects.update |
| RemoveWebhook | [RemoveWebhookRequest](#bytebase-v1-RemoveWebhookRequest) | [Project](#bytebase-v1-Project) | Permissions required: bb.projects.update |
| TestWebhook | [TestWebhookRequest](#bytebase-v1-TestWebhookRequest) | [TestWebhookResponse](#bytebase-v1-TestWebhookResponse) | Permissions required: bb.projects.update |

 



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
| status | [Advice.Status](#bytebase-v1-Advice-Status) |  | The advice status. |
| code | [int32](#int32) |  | The advice code. |
| title | [string](#string) |  | The advice title. |
| content | [string](#string) |  | The advice content. |
| start_position | [Position](#bytebase-v1-Position) |  | The start_position is inclusive and the end_position is exclusive. TODO: use range instead |
| end_position | [Position](#bytebase-v1-Position) |  |  |






<a name="bytebase-v1-CheckRequest"></a>

### CheckRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The database name to check against. Format: instances/{instance}/databases/{database} |
| statement | [string](#string) |  |  |
| change_type | [CheckRequest.ChangeType](#bytebase-v1-CheckRequest-ChangeType) |  |  |






<a name="bytebase-v1-CheckResponse"></a>

### CheckResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| advices | [Advice](#bytebase-v1-Advice) | repeated |  |
| affected_rows | [int64](#int64) |  | The count of affected rows of the statement on the target database. |






<a name="bytebase-v1-DiffMetadataRequest"></a>

### DiffMetadataRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| source_metadata | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  | The metadata of the source schema. |
| target_metadata | [DatabaseMetadata](#bytebase-v1-DatabaseMetadata) |  | The metadata of the target schema. |
| source_catalog | [DatabaseCatalog](#bytebase-v1-DatabaseCatalog) |  |  |
| target_catalog | [DatabaseCatalog](#bytebase-v1-DatabaseCatalog) |  |  |
| engine | [Engine](#bytebase-v1-Engine) |  | The database engine of the schema. |
| classification_from_config | [bool](#bool) |  | If false, we will build the raw common by classification in database config. |






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






<a name="bytebase-v1-QueryOption"></a>

### QueryOption



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| redis_run_commands_on | [QueryOption.RedisRunCommandsOn](#bytebase-v1-QueryOption-RedisRunCommandsOn) |  |  |






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
| allow_export | [bool](#bool) |  | The query result is allowed to be exported or not. |
| messages | [QueryResult.Message](#bytebase-v1-QueryResult-Message) | repeated | Informational or debug messages returned by the database engine during query execution. Examples include PostgreSQL&#39;s RAISE NOTICE, MSSQL&#39;s PRINT, or Oracle&#39;s DBMS_OUTPUT.PUT_LINE. |
| masked | [MaskingReason](#bytebase-v1-MaskingReason) | repeated | Masking reasons for each column (empty for non-masked columns). |






<a name="bytebase-v1-QueryResult-Message"></a>

### QueryResult.Message



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| level | [QueryResult.Message.Level](#bytebase-v1-QueryResult-Message-Level) |  |  |
| content | [string](#string) |  |  |






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

Supported filter: - project: the project full name in &#34;projects/{id}&#34; format, support &#34;==&#34; operator. - database: the database full name in &#34;instances/{id}/databases/{name}&#34; format, support &#34;==&#34; operator. - instance: the instance full name in &#34;instances/{id}&#34; format, support &#34;==&#34; operator. - type: the type, should be &#34;QUERY&#34; or &#34;EXPORT&#34;, support &#34;==&#34; operator. - statement: the SQL statemnt, support &#34;.matches()&#34; operator.

For example: project == &#34;projects/{project}&#34; database == &#34;instances/{instance}/databases/{database}&#34; instance == &#34;instances/{instance}&#34; type == &#34;QUERY&#34; type == &#34;EXPORT&#34; statement.matches(&#34;select&#34;) type == &#34;QUERY&#34; &amp;&amp; statement.matches(&#34;select&#34;) |






<a name="bytebase-v1-SearchQueryHistoriesResponse"></a>

### SearchQueryHistoriesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| query_histories | [QueryHistory](#bytebase-v1-QueryHistory) | repeated | The list of history. |
| next_page_token | [string](#string) |  | A token to retrieve next page of history. Pass this value in the page_token field in the subsequent call to `ListQueryHistory` method to retrieve the next page of history. |





 


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
| SQL_EDITOR | 4 |  |



<a name="bytebase-v1-QueryHistory-Type"></a>

### QueryHistory.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| QUERY | 1 |  |
| EXPORT | 2 |  |



<a name="bytebase-v1-QueryOption-RedisRunCommandsOn"></a>

### QueryOption.RedisRunCommandsOn


| Name | Number | Description |
| ---- | ------ | ----------- |
| REDIS_RUN_COMMANDS_ON_UNSPECIFIED | 0 | UNSPECIFIED defaults to SINGLE_NODE. |
| SINGLE_NODE | 1 |  |
| ALL_NODES | 2 |  |



<a name="bytebase-v1-QueryResult-Message-Level"></a>

### QueryResult.Message.Level


| Name | Number | Description |
| ---- | ------ | ----------- |
| LEVEL_UNSPECIFIED | 0 | Unspecified. |
| INFO | 1 |  |
| WARNING | 2 |  |
| DEBUG | 3 |  |
| LOG | 4 |  |
| NOTICE | 5 |  |
| EXCEPTION | 6 |  |


 

 


<a name="bytebase-v1-SQLService"></a>

### SQLService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Query | [QueryRequest](#bytebase-v1-QueryRequest) | [QueryResponse](#bytebase-v1-QueryResponse) | Permissions required: bb.databases.get |
| AdminExecute | [AdminExecuteRequest](#bytebase-v1-AdminExecuteRequest) stream | [AdminExecuteResponse](#bytebase-v1-AdminExecuteResponse) stream | Permissions required: bb.sql.admin |
| SearchQueryHistories | [SearchQueryHistoriesRequest](#bytebase-v1-SearchQueryHistoriesRequest) | [SearchQueryHistoriesResponse](#bytebase-v1-SearchQueryHistoriesResponse) | SearchQueryHistories searches query histories for the caller. Permissions required: None |
| Export | [ExportRequest](#bytebase-v1-ExportRequest) | [ExportResponse](#bytebase-v1-ExportResponse) | Permissions required: bb.databases.get |
| Check | [CheckRequest](#bytebase-v1-CheckRequest) | [CheckResponse](#bytebase-v1-CheckResponse) | Permissions required: bb.databases.check |
| Pretty | [PrettyRequest](#bytebase-v1-PrettyRequest) | [PrettyResponse](#bytebase-v1-PrettyResponse) | Permissions required: None |
| DiffMetadata | [DiffMetadataRequest](#bytebase-v1-DiffMetadataRequest) | [DiffMetadataResponse](#bytebase-v1-DiffMetadataResponse) | Permissions required: None |
| AICompletion | [AICompletionRequest](#bytebase-v1-AICompletionRequest) | [AICompletionResponse](#bytebase-v1-AICompletionResponse) | Permissions required: None |

 



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






<a name="bytebase-v1-CheckReleaseResponse"></a>

### CheckReleaseResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| results | [CheckReleaseResponse.CheckResult](#bytebase-v1-CheckReleaseResponse-CheckResult) | repeated |  |
| affected_rows | [int64](#int64) |  | The affected rows of the check. |
| risk_level | [CheckReleaseResponse.RiskLevel](#bytebase-v1-CheckReleaseResponse-RiskLevel) |  | The aggregated risk level of the check. |






<a name="bytebase-v1-CheckReleaseResponse-CheckResult"></a>

### CheckReleaseResponse.CheckResult



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| file | [string](#string) |  | The file path that is being checked. |
| target | [string](#string) |  | The target that the check is performed on. Should be a database. Format: instances/{instance}/databases/{database} |
| advices | [Advice](#bytebase-v1-Advice) | repeated | The list of advice for the file and the target. |
| affected_rows | [int64](#int64) |  | The count of affected rows of the statement on the target. |
| risk_level | [CheckReleaseResponse.RiskLevel](#bytebase-v1-CheckReleaseResponse-RiskLevel) |  | The risk level of the statement on the target. |






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
| releases | [Release](#bytebase-v1-Release) | repeated |  |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






<a name="bytebase-v1-Release"></a>

### Release



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/releases/{release} |
| title | [string](#string) |  |  |
| files | [Release.File](#bytebase-v1-Release-File) | repeated |  |
| vcs_source | [Release.VCSSource](#bytebase-v1-Release-VCSSource) |  |  |
| creator | [string](#string) |  | Format: users/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| state | [State](#bytebase-v1-State) |  |  |
| digest | [string](#string) |  | The digest of the release. The user can provide the digest of the release. It can be used later to retrieve the release in GetRelease. Whether to provide digest and how to generate it is up to the user. If the digest is not empty, it must be unique in the project. Otherwise, an ALREADY_EXISTS error will be returned. |






<a name="bytebase-v1-Release-File"></a>

### Release.File



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  | The unique identifier for the file. |
| path | [string](#string) |  | The path of the file. e.g. `2.2/V0001_create_table.sql`. |
| type | [Release.File.Type](#bytebase-v1-Release-File-Type) |  | The type of the file. |
| version | [string](#string) |  |  |
| change_type | [Release.File.ChangeType](#bytebase-v1-Release-File-ChangeType) |  | The change type of the file. For versioned files, it is the change type of the file. For declarative files, this field is always DDL, thus meaningless. |
| sheet | [string](#string) |  | For inputs, we must either use `sheet` or `statement`. For outputs, we always use `sheet`. `statement` is the preview of the sheet content.

The sheet that holds the content. Format: projects/{project}/sheets/{sheet} |
| statement | [bytes](#bytes) |  | The raw SQL statement content. |
| sheet_sha256 | [string](#string) |  | The SHA256 hash value of the sheet content or the statement. |
| statement_size | [int64](#int64) |  | The size of the statement in bytes. |






<a name="bytebase-v1-Release-VCSSource"></a>

### Release.VCSSource



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| vcs_type | [VCSType](#bytebase-v1-VCSType) |  |  |
| url | [string](#string) |  | The url link to the e.g. GitHub commit or pull request. |






<a name="bytebase-v1-SearchReleasesRequest"></a>

### SearchReleasesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{project} |
| page_size | [int32](#int32) |  | The maximum number of releases to return. The service may return fewer than this value. If unspecified, at most 10 releases will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListReleases` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListReleases` must match the call that provided the page token. |
| digest | [string](#string) | optional | Search by the digest of the release. |






<a name="bytebase-v1-SearchReleasesResponse"></a>

### SearchReleasesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| releases | [Release](#bytebase-v1-Release) | repeated |  |
| next_page_token | [string](#string) |  | A token, which can be sent as `page_token` to retrieve the next page. If this field is omitted, there are no subsequent pages. |






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





 


<a name="bytebase-v1-CheckReleaseResponse-RiskLevel"></a>

### CheckReleaseResponse.RiskLevel


| Name | Number | Description |
| ---- | ------ | ----------- |
| RISK_LEVEL_UNSPECIFIED | 0 |  |
| LOW | 1 |  |
| MODERATE | 2 |  |
| HIGH | 3 |  |



<a name="bytebase-v1-Release-File-ChangeType"></a>

### Release.File.ChangeType


| Name | Number | Description |
| ---- | ------ | ----------- |
| CHANGE_TYPE_UNSPECIFIED | 0 |  |
| DDL | 1 |  |
| DDL_GHOST | 2 |  |
| DML | 3 |  |



<a name="bytebase-v1-Release-File-Type"></a>

### Release.File.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| VERSIONED | 1 |  |
| DECLARATIVE | 2 |  |


 

 


<a name="bytebase-v1-ReleaseService"></a>

### ReleaseService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetRelease | [GetReleaseRequest](#bytebase-v1-GetReleaseRequest) | [Release](#bytebase-v1-Release) | Permissions required: bb.releases.get |
| ListReleases | [ListReleasesRequest](#bytebase-v1-ListReleasesRequest) | [ListReleasesResponse](#bytebase-v1-ListReleasesResponse) | Permissions required: bb.releases.list |
| SearchReleases | [SearchReleasesRequest](#bytebase-v1-SearchReleasesRequest) | [SearchReleasesResponse](#bytebase-v1-SearchReleasesResponse) | Permissions required: bb.releases.get |
| CreateRelease | [CreateReleaseRequest](#bytebase-v1-CreateReleaseRequest) | [Release](#bytebase-v1-Release) | Permissions required: bb.releases.create |
| UpdateRelease | [UpdateReleaseRequest](#bytebase-v1-UpdateReleaseRequest) | [Release](#bytebase-v1-Release) | Permissions required: bb.releases.update |
| DeleteRelease | [DeleteReleaseRequest](#bytebase-v1-DeleteReleaseRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Permissions required: bb.releases.delete |
| UndeleteRelease | [UndeleteReleaseRequest](#bytebase-v1-UndeleteReleaseRequest) | [Release](#bytebase-v1-Release) | Permissions required: bb.releases.undelete |
| CheckRelease | [CheckReleaseRequest](#bytebase-v1-CheckReleaseRequest) | [CheckReleaseResponse](#bytebase-v1-CheckReleaseResponse) | Permissions required: bb.releases.check |

 



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
| name | [string](#string) |  | The name of the sql review to delete. Format: reviewConfigs/{reviewConfig} |






<a name="bytebase-v1-GetReviewConfigRequest"></a>

### GetReviewConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the sql review to retrieve. Format: reviewConfigs/{reviewConfig} |






<a name="bytebase-v1-ListReviewConfigsRequest"></a>

### ListReviewConfigsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Not used. The maximum number of sql review to return. The service may return fewer than this value. If unspecified, at most 10 sql review will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListReviewConfigs` must match the call that provided the page token. |






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
| name | [string](#string) |  | The name of the sql review to retrieve. Format: reviewConfigs/{reviewConfig} |
| title | [string](#string) |  |  |
| enabled | [bool](#bool) |  |  |
| rules | [SQLReviewRule](#bytebase-v1-SQLReviewRule) | repeated |  |
| resources | [string](#string) | repeated | resources using the config. Format: {resurce}/{resource id}, for example, environments/test. |






<a name="bytebase-v1-UpdateReviewConfigRequest"></a>

### UpdateReviewConfigRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| review_config | [ReviewConfig](#bytebase-v1-ReviewConfig) |  | The sql review to update.

The name field is used to identify the sql review to update. |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  | The list of fields to update. |
| allow_missing | [bool](#bool) |  | If set to true, and the config is not found, a new config will be created. In this situation, `update_mask` is ignored. |





 

 

 


<a name="bytebase-v1-ReviewConfigService"></a>

### ReviewConfigService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateReviewConfig | [CreateReviewConfigRequest](#bytebase-v1-CreateReviewConfigRequest) | [ReviewConfig](#bytebase-v1-ReviewConfig) | Permissions required: bb.reviewConfigs.create |
| ListReviewConfigs | [ListReviewConfigsRequest](#bytebase-v1-ListReviewConfigsRequest) | [ListReviewConfigsResponse](#bytebase-v1-ListReviewConfigsResponse) | Permissions required: bb.reviewConfigs.list |
| GetReviewConfig | [GetReviewConfigRequest](#bytebase-v1-GetReviewConfigRequest) | [ReviewConfig](#bytebase-v1-ReviewConfig) | Permissions required: bb.reviewConfigs.get |
| UpdateReviewConfig | [UpdateReviewConfigRequest](#bytebase-v1-UpdateReviewConfigRequest) | [ReviewConfig](#bytebase-v1-ReviewConfig) | Permissions required: bb.reviewConfigs.update |
| DeleteReviewConfig | [DeleteReviewConfigRequest](#bytebase-v1-DeleteReviewConfigRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Permissions required: bb.reviewConfigs.delete |

 



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
| show_deleted | [bool](#bool) |  |  |






<a name="bytebase-v1-ListRevisionsResponse"></a>

### ListRevisionsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| revisions | [Revision](#bytebase-v1-Revision) | repeated |  |
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
| version | [string](#string) |  |  |
| sheet | [string](#string) |  | The sheet that holds the content. Format: projects/{project}/sheets/{sheet} |
| sheet_sha256 | [string](#string) |  | The SHA256 hash value of the sheet. |
| statement | [string](#string) |  | The statement is used for preview purpose. |
| statement_size | [int64](#int64) |  |  |
| issue | [string](#string) |  | The issue associated with the revision. Can be empty. Format: projects/{project}/issues/{issue} |
| task_run | [string](#string) |  | The task run associated with the revision. Can be empty. Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |
| type | [Revision.Type](#bytebase-v1-Revision-Type) |  | The type of the revision. |





 


<a name="bytebase-v1-Revision-Type"></a>

### Revision.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| VERSIONED | 1 |  |
| DECLARATIVE | 2 |  |


 

 


<a name="bytebase-v1-RevisionService"></a>

### RevisionService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListRevisions | [ListRevisionsRequest](#bytebase-v1-ListRevisionsRequest) | [ListRevisionsResponse](#bytebase-v1-ListRevisionsResponse) | Permissions required: bb.revisions.list |
| GetRevision | [GetRevisionRequest](#bytebase-v1-GetRevisionRequest) | [Revision](#bytebase-v1-Revision) | Permissions required: bb.revisions.get |
| CreateRevision | [CreateRevisionRequest](#bytebase-v1-CreateRevisionRequest) | [Revision](#bytebase-v1-Revision) | Permissions required: bb.revisions.create |
| BatchCreateRevisions | [BatchCreateRevisionsRequest](#bytebase-v1-BatchCreateRevisionsRequest) | [BatchCreateRevisionsResponse](#bytebase-v1-BatchCreateRevisionsResponse) | Permissions required: bb.revisions.create |
| DeleteRevision | [DeleteRevisionRequest](#bytebase-v1-DeleteRevisionRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Permissions required: bb.revisions.delete |

 



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






<a name="bytebase-v1-GetRiskRequest"></a>

### GetRiskRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the risk to retrieve. Format: risks/{risk} |






<a name="bytebase-v1-ListRisksRequest"></a>

### ListRisksRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Not used. The maximum number of risks to return. The service may return fewer than this value. If unspecified, at most 10 risks will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListRisks` call. Provide this to retrieve the subsequent page.

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
| source | [Risk.Source](#bytebase-v1-Risk-Source) |  |  |
| title | [string](#string) |  |  |
| level | [int32](#int32) |  |  |
| active | [bool](#bool) |  |  |
| condition | [google.type.Expr](#google-type-Expr) |  | The condition that is associated with the risk. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

All supported variables: affected_rows: affected row count in the DDL/DML, support &#34;==&#34;, &#34;!=&#34;, &#34;&lt;&#34;, &#34;&lt;=&#34;, &#34;&gt;&#34;, &#34;&gt;=&#34; operations. table_rows: table row count number, support &#34;==&#34;, &#34;!=&#34;, &#34;&lt;&#34;, &#34;&lt;=&#34;, &#34;&gt;&#34;, &#34;&gt;=&#34; operations. environment_id: the environment resource id, support &#34;==&#34;, &#34;!=&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34; operations. project_id: the project resource id, support &#34;==&#34;, &#34;!=&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34;, &#34;contains()&#34;, &#34;matches()&#34;, &#34;startsWith()&#34;, &#34;endsWith()&#34; operations. db_engine: the database engine type, support &#34;==&#34;, &#34;!=&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34; operations. Check the Engine enum for the values. sql_type: the SQL type, support &#34;==&#34;, &#34;!=&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34; operations. when the risk source is DDL, check https://github.com/bytebase/bytebase/blob/main/frontend/src/plugins/cel/types/values.ts#L70 for supported values. when the risk source is DML, check https://github.com/bytebase/bytebase/blob/main/frontend/src/plugins/cel/types/values.ts#L71 for supported values. database_name: the database name, support &#34;==&#34;, &#34;!=&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34;, &#34;contains()&#34;, &#34;matches()&#34;, &#34;startsWith()&#34;, &#34;endsWith()&#34; operations. schema_name: the schema name, support &#34;==&#34;, &#34;!=&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34;, &#34;contains()&#34;, &#34;matches()&#34;, &#34;startsWith()&#34;, &#34;endsWith()&#34; operations. table_name: the table name, support &#34;==&#34;, &#34;!=&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34;, &#34;contains()&#34;, &#34;matches()&#34;, &#34;startsWith()&#34;, &#34;endsWith()&#34; operations. sql_statement: the SQL statement, support &#34;contains()&#34;, &#34;matches()&#34;, &#34;startsWith()&#34;, &#34;endsWith()&#34; operations. export_rows: export data count, support &#34;==&#34;, &#34;!=&#34;, &#34;&lt;&#34;, &#34;&lt;=&#34;, &#34;&gt;&#34;, &#34;&gt;=&#34; operations. expiration_days: the role expiration days for the request, support &#34;==&#34;, &#34;!=&#34;, &#34;&lt;&#34;, &#34;&lt;=&#34;, &#34;&gt;&#34;, &#34;&gt;=&#34; operations. role: the request role full name, support &#34;==&#34;, &#34;!=&#34;, &#34;in [xx]&#34;, &#34;!(in [xx])&#34;, &#34;contains()&#34;, &#34;matches()&#34;, &#34;startsWith()&#34;, &#34;endsWith()&#34; operations.

When the risk source is DDL/DML, support following variables: affected_rows table_rows environment_id project_id db_engine sql_type database_name schema_name table_name sql_statement

When the risk source is CREATE_DATABASE, support following variables: environment_id project_id db_engine database_name

When the risk source is DATA_EXPORT, support following variables: environment_id project_id db_engine database_name schema_name table_name export_rows

When the risk source is REQUEST_QUERY, support following variables: environment_id project_id db_engine database_name schema_name table_name expiration_days

When the risk source is REQUEST_EXPORT, support following variables: environment_id project_id db_engine database_name schema_name table_name expiration_days export_rows

When the risk source is REQUEST_ROLE, support following variables: project_id expiration_days role |






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
| DATA_EXPORT | 6 |  |
| REQUEST_ROLE | 7 |  |


 

 


<a name="bytebase-v1-RiskService"></a>

### RiskService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListRisks | [ListRisksRequest](#bytebase-v1-ListRisksRequest) | [ListRisksResponse](#bytebase-v1-ListRisksResponse) | Permissions required: bb.risks.list |
| CreateRisk | [CreateRiskRequest](#bytebase-v1-CreateRiskRequest) | [Risk](#bytebase-v1-Risk) | Permissions required: bb.risks.create |
| GetRisk | [GetRiskRequest](#bytebase-v1-GetRiskRequest) | [Risk](#bytebase-v1-Risk) | Permissions required: bb.risks.list |
| UpdateRisk | [UpdateRiskRequest](#bytebase-v1-UpdateRiskRequest) | [Risk](#bytebase-v1-Risk) | Permissions required: bb.risks.update |
| DeleteRisk | [DeleteRiskRequest](#bytebase-v1-DeleteRiskRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Permissions required: bb.risks.delete |

 



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






<a name="bytebase-v1-GetRoleRequest"></a>

### GetRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the role to retrieve. Format: roles/{role} |






<a name="bytebase-v1-ListRolesRequest"></a>

### ListRolesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| page_size | [int32](#int32) |  | Not used. The maximum number of roles to return. The service may return fewer than this value. If unspecified, at most 10 reviews will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListRoles` call. Provide this to retrieve the subsequent page.

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
| type | [Role.Type](#bytebase-v1-Role-Type) |  |  |






<a name="bytebase-v1-UpdateRoleRequest"></a>

### UpdateRoleRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| role | [Role](#bytebase-v1-Role) |  |  |
| update_mask | [google.protobuf.FieldMask](#google-protobuf-FieldMask) |  |  |
| allow_missing | [bool](#bool) |  | If set to true, and the role is not found, a new role will be created. |





 


<a name="bytebase-v1-Role-Type"></a>

### Role.Type


| Name | Number | Description |
| ---- | ------ | ----------- |
| TYPE_UNSPECIFIED | 0 |  |
| BUILT_IN | 1 |  |
| CUSTOM | 2 |  |


 

 


<a name="bytebase-v1-RoleService"></a>

### RoleService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ListRoles | [ListRolesRequest](#bytebase-v1-ListRolesRequest) | [ListRolesResponse](#bytebase-v1-ListRolesResponse) | Permissions required: bb.roles.list |
| GetRole | [GetRoleRequest](#bytebase-v1-GetRoleRequest) | [Role](#bytebase-v1-Role) | Permissions required: bb.roles.get |
| CreateRole | [CreateRoleRequest](#bytebase-v1-CreateRoleRequest) | [Role](#bytebase-v1-Role) | Permissions required: bb.roles.create |
| UpdateRole | [UpdateRoleRequest](#bytebase-v1-UpdateRoleRequest) | [Role](#bytebase-v1-Role) | Permissions required: bb.roles.update |
| DeleteRole | [DeleteRoleRequest](#bytebase-v1-DeleteRoleRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Permissions required: bb.roles.delete |

 



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
| run_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional | The task run should run after run_time. |






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
| target | [string](#string) | optional | Create the rollout and the stages up to the target stage. Format: environments/{environment} If unspecified, all stages are created. If set to &#34;&#34;, no stages are created. |
| validate_only | [bool](#bool) |  | If set, validate the request and preview the rollout, but do not actually create it. |






<a name="bytebase-v1-GetRolloutRequest"></a>

### GetRolloutRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The name of the rollout to retrieve. Format: projects/{project}/rollouts/{rollout} |






<a name="bytebase-v1-GetTaskRunLogRequest"></a>

### GetTaskRunLogRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} TODO(d): check the resource_reference. |






<a name="bytebase-v1-GetTaskRunRequest"></a>

### GetTaskRunRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |






<a name="bytebase-v1-GetTaskRunSessionRequest"></a>

### GetTaskRunSessionRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |






<a name="bytebase-v1-ListRolloutsRequest"></a>

### ListRolloutsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| parent | [string](#string) |  | The parent, which owns this collection of rollouts. Format: projects/{project} Use &#34;projects/-&#34; to list all rollouts from all projects. |
| page_size | [int32](#int32) |  | The maximum number of rollouts to return. The service may return fewer than this value. If unspecified, at most 10 rollouts will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | A page token, received from a previous `ListRollouts` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListRollouts` must match the call that provided the page token. |
| filter | [string](#string) |  | Filter is used to filter rollouts returned in the list. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filters: - creator: the rollout creator full name in &#34;users/{email or id}&#34; format, support &#34;==&#34; operator. - update_time: rollout update time in &#34;2006-01-02T15:04:05Z07:00&#34; format, support &#34;&gt;=&#34; or &#34;&lt;=&#34; operator. - task_type: the task type, support &#34;==&#34; and &#34;in&#34; operators, check the Task.Type enum for the values.

For example: creator == &#34;users/ed@bytebase.com&#34; &amp;&amp; update_time &gt;= &#34;2025-01-02T15:04:05Z07:00&#34; task_type == &#34;DATABASE_SCHEMA_UPDATE&#34; task_type in [&#34;DATABASE_SCHEMA_UPDATE&#34;, &#34;DATABASE_DATA_UPDATE&#34;] |






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
| parent | [string](#string) |  | The parent, which owns this collection of plans. Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} Use &#34;projects/{project}/rollouts/{rollout}/stages/-/tasks/-&#34; to list all taskRuns from a rollout. |
| page_size | [int32](#int32) |  | Not used. The maximum number of taskRuns to return. The service may return fewer than this value. If unspecified, at most 10 taskRuns will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
| page_token | [string](#string) |  | Not used. A page token, received from a previous `ListTaskRuns` call. Provide this to retrieve the subsequent page.

When paginating, all other parameters provided to `ListTaskRuns` must match the call that provided the page token. |






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






<a name="bytebase-v1-PreviewTaskRunRollbackRequest"></a>

### PreviewTaskRunRollbackRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task}/taskRuns/{taskRun} |






<a name="bytebase-v1-PreviewTaskRunRollbackResponse"></a>

### PreviewTaskRunRollbackResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| statement | [string](#string) |  |  |






<a name="bytebase-v1-Rollout"></a>

### Rollout



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | The resource name of the rollout. Format: projects/{project}/rollouts/{rollout} |
| plan | [string](#string) |  | The plan that this rollout is based on. Format: projects/{project}/plans/{plan} |
| title | [string](#string) |  | The title of the rollout, inherited from the associated plan. This field is output only and cannot be directly set. |
| stages | [Stage](#bytebase-v1-Stage) | repeated | stages and thus tasks of the rollout. |
| creator | [string](#string) |  | Format: users/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| issue | [string](#string) |  | The issue associated with the rollout. Could be empty. Format: projects/{project}/issues/{issue} |






<a name="bytebase-v1-Stage"></a>

### Stage



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage} Use &#34;-&#34; for {stage} when the stage has no environment or deleted environment. |
| id | [string](#string) |  | id is the environment id of the stage. e.g. &#34;prod&#34;. Use &#34;-&#34; when the stage has no environment or deleted environment. |
| environment | [string](#string) |  | environment is the environment of the stage. Format: environments/{environment} for valid environments, or &#34;environments/-&#34; for stages without environment or with deleted environments. |
| tasks | [Task](#bytebase-v1-Task) | repeated |  |






<a name="bytebase-v1-Task"></a>

### Task



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} |
| spec_id | [string](#string) |  | A UUID4 string that uniquely identifies the Spec. Could be empty if the rollout of the task does not have an associating plan. |
| status | [Task.Status](#bytebase-v1-Task-Status) |  | Status is the status of the task. |
| skipped_reason | [string](#string) |  |  |
| type | [Task.Type](#bytebase-v1-Task-Type) |  |  |
| target | [string](#string) |  | Format: instances/{instance} if the task is DatabaseCreate. Format: instances/{instance}/databases/{database} |
| database_create | [Task.DatabaseCreate](#bytebase-v1-Task-DatabaseCreate) |  |  |
| database_schema_update | [Task.DatabaseSchemaUpdate](#bytebase-v1-Task-DatabaseSchemaUpdate) |  |  |
| database_data_update | [Task.DatabaseDataUpdate](#bytebase-v1-Task-DatabaseDataUpdate) |  |  |
| database_data_export | [Task.DatabaseDataExport](#bytebase-v1-Task-DatabaseDataExport) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional | The update_time is the update time of latest task run. If there are no task runs, it will be empty. |
| run_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional | The run_time is the scheduled run time of latest task run. If there are no task runs or the task run is not scheduled, it will be empty. |






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
| creator | [string](#string) |  | Format: user/hello@world.com |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| update_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| status | [TaskRun.Status](#bytebase-v1-TaskRun-Status) |  |  |
| detail | [string](#string) |  | Below are the results of a task run. |
| changelog | [string](#string) |  | The resource name of the changelog. Format: instances/{instance}/databases/{database}/changelogs/{changelog} |
| schema_version | [string](#string) |  |  |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| export_archive_status | [TaskRun.ExportArchiveStatus](#bytebase-v1-TaskRun-ExportArchiveStatus) |  |  |
| prior_backup_detail | [TaskRun.PriorBackupDetail](#bytebase-v1-TaskRun-PriorBackupDetail) |  | The prior backup detail that will be used to rollback the task run. |
| scheduler_info | [TaskRun.SchedulerInfo](#bytebase-v1-TaskRun-SchedulerInfo) |  |  |
| sheet | [string](#string) |  | Format: projects/{project}/sheets/{sheet} |
| run_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) | optional | The task run should run after run_time. This can only be set when creating the task run calling BatchRunTasks. |






<a name="bytebase-v1-TaskRun-PriorBackupDetail"></a>

### TaskRun.PriorBackupDetail



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| items | [TaskRun.PriorBackupDetail.Item](#bytebase-v1-TaskRun-PriorBackupDetail-Item) | repeated |  |






<a name="bytebase-v1-TaskRun-PriorBackupDetail-Item"></a>

### TaskRun.PriorBackupDetail.Item



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| source_table | [TaskRun.PriorBackupDetail.Item.Table](#bytebase-v1-TaskRun-PriorBackupDetail-Item-Table) |  | The original table information. |
| target_table | [TaskRun.PriorBackupDetail.Item.Table](#bytebase-v1-TaskRun-PriorBackupDetail-Item-Table) |  | The target backup table information. |
| start_position | [Position](#bytebase-v1-Position) |  |  |
| end_position | [Position](#bytebase-v1-Position) |  |  |






<a name="bytebase-v1-TaskRun-PriorBackupDetail-Item-Table"></a>

### TaskRun.PriorBackupDetail.Item.Table



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| database | [string](#string) |  | The database information. Format: instances/{instance}/databases/{database} |
| schema | [string](#string) |  |  |
| table | [string](#string) |  |  |






<a name="bytebase-v1-TaskRun-SchedulerInfo"></a>

### TaskRun.SchedulerInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| report_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| waiting_cause | [TaskRun.SchedulerInfo.WaitingCause](#bytebase-v1-TaskRun-SchedulerInfo-WaitingCause) |  |  |






<a name="bytebase-v1-TaskRun-SchedulerInfo-WaitingCause"></a>

### TaskRun.SchedulerInfo.WaitingCause



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| connection_limit | [bool](#bool) |  |  |
| task | [TaskRun.SchedulerInfo.WaitingCause.Task](#bytebase-v1-TaskRun-SchedulerInfo-WaitingCause-Task) |  |  |
| parallel_tasks_limit | [bool](#bool) |  |  |






<a name="bytebase-v1-TaskRun-SchedulerInfo-WaitingCause-Task"></a>

### TaskRun.SchedulerInfo.WaitingCause.Task



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| task | [string](#string) |  | Format: projects/{project}/rollouts/{rollout}/stages/{stage}/tasks/{task} |
| issue | [string](#string) |  | Format: projects/{project}/issues/{issue} |






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
| deploy_id | [string](#string) |  |  |
| schema_dump | [TaskRunLogEntry.SchemaDump](#bytebase-v1-TaskRunLogEntry-SchemaDump) |  |  |
| command_execute | [TaskRunLogEntry.CommandExecute](#bytebase-v1-TaskRunLogEntry-CommandExecute) |  |  |
| database_sync | [TaskRunLogEntry.DatabaseSync](#bytebase-v1-TaskRunLogEntry-DatabaseSync) |  |  |
| task_run_status_update | [TaskRunLogEntry.TaskRunStatusUpdate](#bytebase-v1-TaskRunLogEntry-TaskRunStatusUpdate) |  |  |
| transaction_control | [TaskRunLogEntry.TransactionControl](#bytebase-v1-TaskRunLogEntry-TransactionControl) |  |  |
| prior_backup | [TaskRunLogEntry.PriorBackup](#bytebase-v1-TaskRunLogEntry-PriorBackup) |  |  |
| retry_info | [TaskRunLogEntry.RetryInfo](#bytebase-v1-TaskRunLogEntry-RetryInfo) |  |  |
| compute_diff | [TaskRunLogEntry.ComputeDiff](#bytebase-v1-TaskRunLogEntry-ComputeDiff) |  |  |






<a name="bytebase-v1-TaskRunLogEntry-CommandExecute"></a>

### TaskRunLogEntry.CommandExecute



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| log_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| command_indexes | [int32](#int32) | repeated | Either `command_indexes` or `statement` is set. The indexes of the executed commands. |
| statement | [string](#string) |  | The executed statement. |
| response | [TaskRunLogEntry.CommandExecute.CommandResponse](#bytebase-v1-TaskRunLogEntry-CommandExecute-CommandResponse) |  |  |






<a name="bytebase-v1-TaskRunLogEntry-CommandExecute-CommandResponse"></a>

### TaskRunLogEntry.CommandExecute.CommandResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| log_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| error | [string](#string) |  |  |
| affected_rows | [int64](#int64) |  |  |
| all_affected_rows | [int64](#int64) | repeated | `all_affected_rows` is the affected rows of each command. `all_affected_rows` may be unavailable if the database driver doesn&#39;t support it. Caller should fallback to `affected_rows` in that case. |






<a name="bytebase-v1-TaskRunLogEntry-ComputeDiff"></a>

### TaskRunLogEntry.ComputeDiff



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| end_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| error | [string](#string) |  |  |






<a name="bytebase-v1-TaskRunLogEntry-DatabaseSync"></a>

### TaskRunLogEntry.DatabaseSync



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| end_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| error | [string](#string) |  |  |






<a name="bytebase-v1-TaskRunLogEntry-PriorBackup"></a>

### TaskRunLogEntry.PriorBackup



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| start_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| end_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  |  |
| prior_backup_detail | [TaskRun.PriorBackupDetail](#bytebase-v1-TaskRun-PriorBackupDetail) |  |  |
| error | [string](#string) |  |  |






<a name="bytebase-v1-TaskRunLogEntry-RetryInfo"></a>

### TaskRunLogEntry.RetryInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| error | [string](#string) |  |  |
| retry_count | [int32](#int32) |  |  |
| maximum_retries | [int32](#int32) |  |  |






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
| DATABASE_SCHEMA_UPDATE | 4 | use payload DatabaseSchemaUpdate |
| DATABASE_SCHEMA_UPDATE_SDL | 5 | use payload DatabaseSchemaUpdate |
| DATABASE_SCHEMA_UPDATE_GHOST | 9 | use payload DatabaseSchemaUpdate |
| DATABASE_DATA_UPDATE | 8 | use payload DatabaseDataUpdate |
| DATABASE_EXPORT | 12 | use payload DatabaseDataExport |



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
| PRIOR_BACKUP | 6 |  |
| RETRY_INFO | 7 |  |
| COMPUTE_DIFF | 8 |  |


 

 


<a name="bytebase-v1-RolloutService"></a>

### RolloutService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetRollout | [GetRolloutRequest](#bytebase-v1-GetRolloutRequest) | [Rollout](#bytebase-v1-Rollout) | Permissions required: bb.rollouts.get |
| ListRollouts | [ListRolloutsRequest](#bytebase-v1-ListRolloutsRequest) | [ListRolloutsResponse](#bytebase-v1-ListRolloutsResponse) | Permissions required: bb.rollouts.list |
| CreateRollout | [CreateRolloutRequest](#bytebase-v1-CreateRolloutRequest) | [Rollout](#bytebase-v1-Rollout) | Permissions required: bb.rollouts.create |
| PreviewRollout | [PreviewRolloutRequest](#bytebase-v1-PreviewRolloutRequest) | [Rollout](#bytebase-v1-Rollout) | Permissions required: bb.rollouts.preview |
| ListTaskRuns | [ListTaskRunsRequest](#bytebase-v1-ListTaskRunsRequest) | [ListTaskRunsResponse](#bytebase-v1-ListTaskRunsResponse) | Permissions required: bb.taskRuns.list |
| GetTaskRun | [GetTaskRunRequest](#bytebase-v1-GetTaskRunRequest) | [TaskRun](#bytebase-v1-TaskRun) | Permissions required: bb.taskRuns.list |
| GetTaskRunLog | [GetTaskRunLogRequest](#bytebase-v1-GetTaskRunLogRequest) | [TaskRunLog](#bytebase-v1-TaskRunLog) | Permissions required: bb.taskRuns.list |
| GetTaskRunSession | [GetTaskRunSessionRequest](#bytebase-v1-GetTaskRunSessionRequest) | [TaskRunSession](#bytebase-v1-TaskRunSession) | Permissions required: bb.taskRuns.list |
| BatchRunTasks | [BatchRunTasksRequest](#bytebase-v1-BatchRunTasksRequest) | [BatchRunTasksResponse](#bytebase-v1-BatchRunTasksResponse) | Permissions required: None |
| BatchSkipTasks | [BatchSkipTasksRequest](#bytebase-v1-BatchSkipTasksRequest) | [BatchSkipTasksResponse](#bytebase-v1-BatchSkipTasksResponse) | Permissions required: None |
| BatchCancelTaskRuns | [BatchCancelTaskRunsRequest](#bytebase-v1-BatchCancelTaskRunsRequest) | [BatchCancelTaskRunsResponse](#bytebase-v1-BatchCancelTaskRunsResponse) | Permissions required: None |
| PreviewTaskRunRollback | [PreviewTaskRunRollbackRequest](#bytebase-v1-PreviewTaskRunRollbackRequest) | [PreviewTaskRunRollbackResponse](#bytebase-v1-PreviewTaskRunRollbackResponse) | Permissions required: bb.taskRuns.list |

 



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
| name | [string](#string) |  | The name of the sheet resource, generated by the server. Canonical parent is project. Format: projects/{project}/sheets/{sheet} |
| title | [string](#string) |  | The title of the sheet. |
| creator | [string](#string) |  | The creator of the Sheet. Format: users/{email} |
| create_time | [google.protobuf.Timestamp](#google-protobuf-Timestamp) |  | The create time of the sheet. |
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
| CreateSheet | [CreateSheetRequest](#bytebase-v1-CreateSheetRequest) | [Sheet](#bytebase-v1-Sheet) | Permissions required: bb.sheets.create |
| BatchCreateSheets | [BatchCreateSheetsRequest](#bytebase-v1-BatchCreateSheetsRequest) | [BatchCreateSheetsResponse](#bytebase-v1-BatchCreateSheetsResponse) | Permissions required: bb.sheets.create |
| GetSheet | [GetSheetRequest](#bytebase-v1-GetSheetRequest) | [Sheet](#bytebase-v1-Sheet) | Permissions required: bb.sheets.get |
| UpdateSheet | [UpdateSheetRequest](#bytebase-v1-UpdateSheetRequest) | [Sheet](#bytebase-v1-Sheet) | Permissions required: bb.sheets.update |

 



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
| FEATURE_CHANGELIST | 14 |  |
| FEATURE_SCHEMA_TEMPLATE | 15 |  |
| FEATURE_ROLLOUT_POLICY | 16 |  |
| FEATURE_WEB_BASED_SQL_EDITOR | 17 | SQL Editor &amp; Development |
| FEATURE_SQL_EDITOR_ADMIN_MODE | 18 |  |
| FEATURE_NATURAL_LANGUAGE_TO_SQL | 19 |  |
| FEATURE_AI_QUERY_EXPLANATION | 20 |  |
| FEATURE_AI_QUERY_SUGGESTIONS | 21 |  |
| FEATURE_AUTO_COMPLETE | 22 |  |
| FEATURE_SCHEMA_DIAGRAM | 23 |  |
| FEATURE_SCHEMA_EDITOR | 24 |  |
| FEATURE_DATA_EXPORT | 25 |  |
| FEATURE_DATA_OFFLINE_EXPORT | 26 |  |
| FEATURE_QUERY_HISTORY | 27 |  |
| FEATURE_SAVED_AND_SHARED_SQL_SCRIPTS | 28 |  |
| FEATURE_BATCH_QUERY | 29 |  |
| FEATURE_INSTANCE_READ_ONLY_CONNECTION | 30 |  |
| FEATURE_QUERY_POLICY | 31 |  |
| FEATURE_RESTRICT_COPYING_DATA | 32 |  |
| FEATURE_IAM | 33 | Security &amp; Compliance |
| FEATURE_INSTANCE_SSL_CONNECTION | 34 |  |
| FEATURE_INSTANCE_CONNECTION_OVER_SSH_TUNNEL | 35 |  |
| FEATURE_INSTANCE_CONNECTION_IAM_AUTHENTICATION | 36 |  |
| FEATURE_GOOGLE_AND_GITHUB_SSO | 37 |  |
| FEATURE_USER_GROUPS | 38 |  |
| FEATURE_DISALLOW_SELF_SERVICE_SIGNUP | 39 |  |
| FEATURE_CUSTOM_INSTANCE_SYNC_TIME | 41 |  |
| FEATURE_CUSTOM_INSTANCE_CONNECTION_LIMIT | 42 |  |
| FEATURE_RISK_ASSESSMENT | 43 |  |
| FEATURE_APPROVAL_WORKFLOW | 44 |  |
| FEATURE_AUDIT_LOG | 45 |  |
| FEATURE_ENTERPRISE_SSO | 46 |  |
| FEATURE_TWO_FA | 47 |  |
| FEATURE_PASSWORD_RESTRICTIONS | 48 |  |
| FEATURE_DISALLOW_PASSWORD_SIGNIN | 49 |  |
| FEATURE_CUSTOM_ROLES | 50 |  |
| FEATURE_REQUEST_ROLE_WORKFLOW | 51 |  |
| FEATURE_DATA_MASKING | 52 |  |
| FEATURE_DATA_CLASSIFICATION | 53 |  |
| FEATURE_SCIM | 54 |  |
| FEATURE_DIRECTORY_SYNC | 55 |  |
| FEATURE_SIGN_IN_FREQUENCY_CONTROL | 56 |  |
| FEATURE_EXTERNAL_SECRET_MANAGER | 57 |  |
| FEATURE_USER_EMAIL_DOMAIN_RESTRICTION | 58 |  |
| FEATURE_PROJECT_MANAGEMENT | 59 | Administration &amp; Support |
| FEATURE_ENVIRONMENT_MANAGEMENT | 60 |  |
| FEATURE_IM_NOTIFICATIONS | 61 |  |
| FEATURE_TERRAFORM_PROVIDER | 62 |  |
| FEATURE_DATABASE_GROUPS | 63 |  |
| FEATURE_ENVIRONMENT_TIERS | 64 |  |
| FEATURE_DASHBOARD_ANNOUNCEMENT | 65 |  |
| FEATURE_API_INTEGRATION_GUIDANCE | 66 |  |
| FEATURE_CUSTOM_LOGO | 67 |  |
| FEATURE_WATERMARK | 68 |  |
| FEATURE_ROADMAP_PRIORITIZATION | 69 |  |
| FEATURE_CUSTOM_MSA | 70 |  |
| FEATURE_COMMUNITY_SUPPORT | 71 |  |
| FEATURE_EMAIL_SUPPORT | 72 |  |
| FEATURE_DEDICATED_SUPPORT_WITH_SLA | 73 |  |



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
| GetSubscription | [GetSubscriptionRequest](#bytebase-v1-GetSubscriptionRequest) | [Subscription](#bytebase-v1-Subscription) | GetSubscription returns the current subscription. If there is no license, we will return a free plan subscription without expiration time. If there is expired license, we will return a free plan subscription with the expiration time of the expired license. Permissions required: None |
| UpdateSubscription | [UpdateSubscriptionRequest](#bytebase-v1-UpdateSubscriptionRequest) | [Subscription](#bytebase-v1-Subscription) | Permissions required: bb.settings.set |

 



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
| filter | [string](#string) |  | To filter the search result. The syntax and semantics of CEL are documented at https://github.com/google/cel-spec

Supported filter: - creator: the worksheet creator in &#34;users/{email}&#34; format, support &#34;==&#34; and &#34;!=&#34; operator. - starred: should be &#34;true&#34; or &#34;false&#34;, filter starred/unstarred sheets, support &#34;==&#34; operator. - visibility: check Visibility enum in the Worksheet message for values, support &#34;==&#34; and &#34;in [xx]&#34; operator.

For example: creator == &#34;users/{email}&#34; creator != &#34;users/{email}&#34; starred == true starred == false visibility in [&#34;PRIVATE&#34;, &#34;PROJECT_READ&#34;, &#34;PROJECT_WRITE&#34;] visibility == &#34;PRIVATE&#34; |
| page_size | [int32](#int32) |  | Not used. The maximum number of worksheets to return. The service may return fewer than this value. If unspecified, at most 10 worksheets will be returned. The maximum value is 1000; values above 1000 will be coerced to 1000. |
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
| PROJECT_READ | 1 | Read access in project scope, worksheet OWNER/DBA and project OWNER can read/write, other project members can read. |
| PROJECT_WRITE | 2 | Write access in project scope, worksheet OWNER/DBA and all members in the project can write the worksheet. |
| PRIVATE | 3 | Private, only worksheet OWNER can read/write. |


 

 


<a name="bytebase-v1-WorksheetService"></a>

### WorksheetService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateWorksheet | [CreateWorksheetRequest](#bytebase-v1-CreateWorksheetRequest) | [Worksheet](#bytebase-v1-Worksheet) | Create a personal worksheet used in SQL Editor. Permissions required: None |
| GetWorksheet | [GetWorksheetRequest](#bytebase-v1-GetWorksheetRequest) | [Worksheet](#bytebase-v1-Worksheet) | Get a worksheet by name. The users can access this method if, - they are the creator of the worksheet; - they have bb.worksheets.get permission on the workspace; - the sheet is shared with them with PROJECT_READ and PROJECT_WRITE visibility, and they have bb.projects.get permission on the project. Permissions required: None |
| SearchWorksheets | [SearchWorksheetsRequest](#bytebase-v1-SearchWorksheetsRequest) | [SearchWorksheetsResponse](#bytebase-v1-SearchWorksheetsResponse) | Search for worksheets. This is used for finding my worksheets or worksheets shared by other people. The sheet accessibility is the same as GetWorksheet(). Permissions required: None |
| UpdateWorksheet | [UpdateWorksheetRequest](#bytebase-v1-UpdateWorksheetRequest) | [Worksheet](#bytebase-v1-Worksheet) | Update a worksheet. The users can access this method if, - they are the creator of the worksheet; - they have bb.worksheets.manage permission on the workspace; - the sheet is shared with them with PROJECT_WRITE visibility, and they have bb.projects.get permission on the project. Permissions required: None |
| UpdateWorksheetOrganizer | [UpdateWorksheetOrganizerRequest](#bytebase-v1-UpdateWorksheetOrganizerRequest) | [WorksheetOrganizer](#bytebase-v1-WorksheetOrganizer) | Update the organizer of a worksheet. The access is the same as UpdateWorksheet method. Permissions required: None |
| DeleteWorksheet | [DeleteWorksheetRequest](#bytebase-v1-DeleteWorksheetRequest) | [.google.protobuf.Empty](#google-protobuf-Empty) | Delete a worksheet. The access is the same as UpdateWorksheet method. Permissions required: None |

 



<a name="v1_workspace_service-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## v1/workspace_service.proto


 

 

 


<a name="bytebase-v1-WorkspaceService"></a>

### WorkspaceService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| GetIamPolicy | [GetIamPolicyRequest](#bytebase-v1-GetIamPolicyRequest) | [IamPolicy](#bytebase-v1-IamPolicy) | Permissions required: bb.workspaces.getIamPolicy |
| SetIamPolicy | [SetIamPolicyRequest](#bytebase-v1-SetIamPolicyRequest) | [IamPolicy](#bytebase-v1-IamPolicy) | Permissions required: bb.workspaces.setIamPolicy |

 



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

