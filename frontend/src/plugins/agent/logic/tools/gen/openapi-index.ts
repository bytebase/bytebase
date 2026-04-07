// Auto-generated from openapi.yaml. DO NOT EDIT manually.
// Run 'pnpm --dir frontend run generate:openapi-index' to regenerate.

export interface EndpointInfo {
  operationId: string;
  path: string;
  service: string;
  method: string;
  summary: string;
  description: string;
  requestSchemaRef: string;
  responseSchemaRef: string;
}

export interface PropertyInfo {
  name: string;
  type: string;
  description?: string;
  required?: boolean;
}

export interface SchemaInfo {
  type: "object" | "enum";
  description: string;
  properties?: PropertyInfo[];
  values?: string[];
}

export const endpoints: EndpointInfo[] = [
  {
    operationId: "bytebase.v1.AIService.Chat",
    path: "/bytebase.v1.AIService/Chat",
    service: "AIService",
    method: "Chat",
    summary: "Chat",
    description:
      "Chat sends a conversation with tool definitions to the configured AI provider\n and returns the AI response.",
    requestSchemaRef: "#/components/schemas/bytebase.v1.AIChatRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.AIChatResponse",
  },
  {
    operationId: "bytebase.v1.AccessGrantService.ActivateAccessGrant",
    path: "/bytebase.v1.AccessGrantService/ActivateAccessGrant",
    service: "AccessGrantService",
    method: "ActivateAccessGrant",
    summary: "ActivateAccessGrant",
    description: "Activates a pending access grant.",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.ActivateAccessGrantRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.AccessGrant",
  },
  {
    operationId: "bytebase.v1.AccessGrantService.CreateAccessGrant",
    path: "/bytebase.v1.AccessGrantService/CreateAccessGrant",
    service: "AccessGrantService",
    method: "CreateAccessGrant",
    summary: "CreateAccessGrant",
    description: "Creates an access grant.",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.CreateAccessGrantRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.AccessGrant",
  },
  {
    operationId: "bytebase.v1.AccessGrantService.GetAccessGrant",
    path: "/bytebase.v1.AccessGrantService/GetAccessGrant",
    service: "AccessGrantService",
    method: "GetAccessGrant",
    summary: "GetAccessGrant",
    description: "Gets an access grant by name.",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetAccessGrantRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.AccessGrant",
  },
  {
    operationId: "bytebase.v1.AccessGrantService.ListAccessGrants",
    path: "/bytebase.v1.AccessGrantService/ListAccessGrants",
    service: "AccessGrantService",
    method: "ListAccessGrants",
    summary: "ListAccessGrants",
    description: "Lists access grants in a project.",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.ListAccessGrantsRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.ListAccessGrantsResponse",
  },
  {
    operationId: "bytebase.v1.AccessGrantService.RevokeAccessGrant",
    path: "/bytebase.v1.AccessGrantService/RevokeAccessGrant",
    service: "AccessGrantService",
    method: "RevokeAccessGrant",
    summary: "RevokeAccessGrant",
    description: "Revokes an active access grant.",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.RevokeAccessGrantRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.AccessGrant",
  },
  {
    operationId: "bytebase.v1.AccessGrantService.SearchMyAccessGrants",
    path: "/bytebase.v1.AccessGrantService/SearchMyAccessGrants",
    service: "AccessGrantService",
    method: "SearchMyAccessGrants",
    summary: "SearchMyAccessGrants",
    description: "Searches access grants created by the caller.",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.SearchMyAccessGrantsRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.SearchMyAccessGrantsResponse",
  },
  {
    operationId: "bytebase.v1.ActuatorService.DeleteCache",
    path: "/bytebase.v1.ActuatorService/DeleteCache",
    service: "ActuatorService",
    method: "DeleteCache",
    summary: "DeleteCache",
    description:
      "Clears the system cache to force data refresh.\n Permissions required: None",
    requestSchemaRef: "#/components/schemas/bytebase.v1.DeleteCacheRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.ActuatorService.GetActuatorInfo",
    path: "/bytebase.v1.ActuatorService/GetActuatorInfo",
    service: "ActuatorService",
    method: "GetActuatorInfo",
    summary: "GetActuatorInfo",
    description:
      "Gets system information and health status of the Bytebase instance.\n Permissions required: None",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetActuatorInfoRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ActuatorInfo",
  },
  {
    operationId: "bytebase.v1.ActuatorService.GetWorkspaceActuatorInfo",
    path: "/bytebase.v1.ActuatorService/GetWorkspaceActuatorInfo",
    service: "ActuatorService",
    method: "GetWorkspaceActuatorInfo",
    summary: "GetWorkspaceActuatorInfo",
    description:
      "Gets workspace-scoped actuator info. Requires authentication.",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.GetWorkspaceActuatorInfoRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ActuatorInfo",
  },
  {
    operationId: "bytebase.v1.ActuatorService.SetupSample",
    path: "/bytebase.v1.ActuatorService/SetupSample",
    service: "ActuatorService",
    method: "SetupSample",
    summary: "SetupSample",
    description:
      "Sets up sample data for demonstration and testing purposes.\n Permissions required: bb.projects.create",
    requestSchemaRef: "#/components/schemas/bytebase.v1.SetupSampleRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.AuditLogService.ExportAuditLogs",
    path: "/bytebase.v1.AuditLogService/ExportAuditLogs",
    service: "AuditLogService",
    method: "ExportAuditLogs",
    summary: "ExportAuditLogs",
    description:
      "Exports audit logs in a specified format for external analysis.\n Permissions required: bb.auditLogs.export",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ExportAuditLogsRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.ExportAuditLogsResponse",
  },
  {
    operationId: "bytebase.v1.AuditLogService.SearchAuditLogs",
    path: "/bytebase.v1.AuditLogService/SearchAuditLogs",
    service: "AuditLogService",
    method: "SearchAuditLogs",
    summary: "SearchAuditLogs",
    description:
      "Searches audit logs with optional filtering and pagination.\n Permissions required: bb.auditLogs.search",
    requestSchemaRef: "#/components/schemas/bytebase.v1.SearchAuditLogsRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.SearchAuditLogsResponse",
  },
  {
    operationId: "bytebase.v1.AuthService.ExchangeToken",
    path: "/bytebase.v1.AuthService/ExchangeToken",
    service: "AuthService",
    method: "ExchangeToken",
    summary: "ExchangeToken",
    description:
      "Exchanges an external OIDC token for a Bytebase access token.\n Used by CI/CD pipelines with Workload Identity Federation.\n Permissions required: None (validates via OIDC token)",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ExchangeTokenRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ExchangeTokenResponse",
  },
  {
    operationId: "bytebase.v1.AuthService.Login",
    path: "/bytebase.v1.AuthService/Login",
    service: "AuthService",
    method: "Login",
    summary: "Login",
    description:
      "Authenticates a user and returns access tokens.\n Permissions required: None",
    requestSchemaRef: "#/components/schemas/bytebase.v1.LoginRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.LoginResponse",
  },
  {
    operationId: "bytebase.v1.AuthService.Logout",
    path: "/bytebase.v1.AuthService/Logout",
    service: "AuthService",
    method: "Logout",
    summary: "Logout",
    description:
      "Logs out the current user session.\n Permissions required: None",
    requestSchemaRef: "#/components/schemas/bytebase.v1.LogoutRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.AuthService.Refresh",
    path: "/bytebase.v1.AuthService/Refresh",
    service: "AuthService",
    method: "Refresh",
    summary: "Refresh",
    description:
      "Refreshes the access token using the refresh token cookie.\n Permissions required: None (validates via refresh token cookie)",
    requestSchemaRef: "#/components/schemas/bytebase.v1.RefreshRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.RefreshResponse",
  },
  {
    operationId: "bytebase.v1.AuthService.Signup",
    path: "/bytebase.v1.AuthService/Signup",
    service: "AuthService",
    method: "Signup",
    summary: "Signup",
    description:
      "Registers a new user account. Creates a principal and assigns a workspace:\n - If the user's email was pre-invited to a workspace, joins that workspace.\n - Otherwise, creates a new workspace with the user as admin.\n Returns access tokens so the user is logged in immediately after signup.",
    requestSchemaRef: "#/components/schemas/bytebase.v1.SignupRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.LoginResponse",
  },
  {
    operationId: "bytebase.v1.AuthService.SwitchWorkspace",
    path: "/bytebase.v1.AuthService/SwitchWorkspace",
    service: "AuthService",
    method: "SwitchWorkspace",
    summary: "SwitchWorkspace",
    description:
      "Switches the current user's active workspace and issues new tokens.\n The user must be a member of the target workspace.",
    requestSchemaRef: "#/components/schemas/bytebase.v1.SwitchWorkspaceRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.LoginResponse",
  },
  {
    operationId: "bytebase.v1.CelService.BatchDeparse",
    path: "/bytebase.v1.CelService/BatchDeparse",
    service: "CelService",
    method: "BatchDeparse",
    summary: "BatchDeparse",
    description:
      "Converts multiple CEL AST representations back into expression strings.\n Permissions required: None",
    requestSchemaRef: "#/components/schemas/bytebase.v1.BatchDeparseRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.BatchDeparseResponse",
  },
  {
    operationId: "bytebase.v1.CelService.BatchParse",
    path: "/bytebase.v1.CelService/BatchParse",
    service: "CelService",
    method: "BatchParse",
    summary: "BatchParse",
    description:
      "Parses multiple CEL expression strings into AST representations.\n Permissions required: None",
    requestSchemaRef: "#/components/schemas/bytebase.v1.BatchParseRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.BatchParseResponse",
  },
  {
    operationId: "bytebase.v1.DatabaseCatalogService.GetDatabaseCatalog",
    path: "/bytebase.v1.DatabaseCatalogService/GetDatabaseCatalog",
    service: "DatabaseCatalogService",
    method: "GetDatabaseCatalog",
    summary: "GetDatabaseCatalog",
    description:
      "Gets the catalog metadata for a database.\n Permissions required: bb.databaseCatalogs.get",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.GetDatabaseCatalogRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.DatabaseCatalog",
  },
  {
    operationId: "bytebase.v1.DatabaseCatalogService.UpdateDatabaseCatalog",
    path: "/bytebase.v1.DatabaseCatalogService/UpdateDatabaseCatalog",
    service: "DatabaseCatalogService",
    method: "UpdateDatabaseCatalog",
    summary: "UpdateDatabaseCatalog",
    description:
      "Updates catalog metadata such as classifications and labels.\n Permissions required: bb.databaseCatalogs.update",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.UpdateDatabaseCatalogRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.DatabaseCatalog",
  },
  {
    operationId: "bytebase.v1.DatabaseGroupService.CreateDatabaseGroup",
    path: "/bytebase.v1.DatabaseGroupService/CreateDatabaseGroup",
    service: "DatabaseGroupService",
    method: "CreateDatabaseGroup",
    summary: "CreateDatabaseGroup",
    description:
      "Creates a new database group.\n Permissions required: bb.databaseGroups.create",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.CreateDatabaseGroupRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.DatabaseGroup",
  },
  {
    operationId: "bytebase.v1.DatabaseGroupService.DeleteDatabaseGroup",
    path: "/bytebase.v1.DatabaseGroupService/DeleteDatabaseGroup",
    service: "DatabaseGroupService",
    method: "DeleteDatabaseGroup",
    summary: "DeleteDatabaseGroup",
    description:
      "Deletes a database group.\n Permissions required: bb.databaseGroups.delete",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.DeleteDatabaseGroupRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.DatabaseGroupService.GetDatabaseGroup",
    path: "/bytebase.v1.DatabaseGroupService/GetDatabaseGroup",
    service: "DatabaseGroupService",
    method: "GetDatabaseGroup",
    summary: "GetDatabaseGroup",
    description:
      "Gets a database group by name.\n Permissions required: bb.databaseGroups.get",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.GetDatabaseGroupRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.DatabaseGroup",
  },
  {
    operationId: "bytebase.v1.DatabaseGroupService.ListDatabaseGroups",
    path: "/bytebase.v1.DatabaseGroupService/ListDatabaseGroups",
    service: "DatabaseGroupService",
    method: "ListDatabaseGroups",
    summary: "ListDatabaseGroups",
    description:
      "Lists database groups in a project.\n Permissions required: bb.databaseGroups.list",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.ListDatabaseGroupsRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.ListDatabaseGroupsResponse",
  },
  {
    operationId: "bytebase.v1.DatabaseGroupService.UpdateDatabaseGroup",
    path: "/bytebase.v1.DatabaseGroupService/UpdateDatabaseGroup",
    service: "DatabaseGroupService",
    method: "UpdateDatabaseGroup",
    summary: "UpdateDatabaseGroup",
    description:
      "Updates a database group.\n Permissions required: bb.databaseGroups.update\n When allow_missing=true, also requires: bb.databaseGroups.create",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.UpdateDatabaseGroupRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.DatabaseGroup",
  },
  {
    operationId: "bytebase.v1.DatabaseService.BatchGetDatabases",
    path: "/bytebase.v1.DatabaseService/BatchGetDatabases",
    service: "DatabaseService",
    method: "BatchGetDatabases",
    summary: "BatchGetDatabases",
    description:
      "Retrieves multiple databases by their names.\n Permissions required: bb.databases.get",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.BatchGetDatabasesRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.BatchGetDatabasesResponse",
  },
  {
    operationId: "bytebase.v1.DatabaseService.BatchSyncDatabases",
    path: "/bytebase.v1.DatabaseService/BatchSyncDatabases",
    service: "DatabaseService",
    method: "BatchSyncDatabases",
    summary: "BatchSyncDatabases",
    description:
      "Synchronizes multiple databases in a single batch operation.\n Permissions required: bb.databases.sync",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.BatchSyncDatabasesRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.BatchSyncDatabasesResponse",
  },
  {
    operationId: "bytebase.v1.DatabaseService.BatchUpdateDatabases",
    path: "/bytebase.v1.DatabaseService/BatchUpdateDatabases",
    service: "DatabaseService",
    method: "BatchUpdateDatabases",
    summary: "BatchUpdateDatabases",
    description:
      "Updates multiple databases in a single batch operation.\n Permissions required: bb.databases.update",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.BatchUpdateDatabasesRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.BatchUpdateDatabasesResponse",
  },
  {
    operationId: "bytebase.v1.DatabaseService.DiffSchema",
    path: "/bytebase.v1.DatabaseService/DiffSchema",
    service: "DatabaseService",
    method: "DiffSchema",
    summary: "DiffSchema",
    description:
      "Compares and generates migration statements between two schemas.\n Permissions required: bb.databases.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.DiffSchemaRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.DiffSchemaResponse",
  },
  {
    operationId: "bytebase.v1.DatabaseService.GetChangelog",
    path: "/bytebase.v1.DatabaseService/GetChangelog",
    service: "DatabaseService",
    method: "GetChangelog",
    summary: "GetChangelog",
    description:
      "Retrieves a specific changelog entry.\n Permissions required: bb.changelogs.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetChangelogRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Changelog",
  },
  {
    operationId: "bytebase.v1.DatabaseService.GetDatabase",
    path: "/bytebase.v1.DatabaseService/GetDatabase",
    service: "DatabaseService",
    method: "GetDatabase",
    summary: "GetDatabase",
    description:
      "Retrieves a database by name.\n Permissions required: bb.databases.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetDatabaseRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Database",
  },
  {
    operationId: "bytebase.v1.DatabaseService.GetDatabaseMetadata",
    path: "/bytebase.v1.DatabaseService/GetDatabaseMetadata",
    service: "DatabaseService",
    method: "GetDatabaseMetadata",
    summary: "GetDatabaseMetadata",
    description:
      "Retrieves database metadata including tables, columns, and indexes.\n Permissions required: bb.databases.getSchema",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.GetDatabaseMetadataRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.DatabaseMetadata",
  },
  {
    operationId: "bytebase.v1.DatabaseService.GetDatabaseSDLSchema",
    path: "/bytebase.v1.DatabaseService/GetDatabaseSDLSchema",
    service: "DatabaseService",
    method: "GetDatabaseSDLSchema",
    summary: "GetDatabaseSDLSchema",
    description:
      "Retrieves database schema in SDL (Schema Definition Language) format.\n Permissions required: bb.databases.getSchema",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.GetDatabaseSDLSchemaRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.DatabaseSDLSchema",
  },
  {
    operationId: "bytebase.v1.DatabaseService.GetDatabaseSchema",
    path: "/bytebase.v1.DatabaseService/GetDatabaseSchema",
    service: "DatabaseService",
    method: "GetDatabaseSchema",
    summary: "GetDatabaseSchema",
    description:
      "Retrieves database schema as DDL statements.\n Permissions required: bb.databases.getSchema",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.GetDatabaseSchemaRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.DatabaseSchema",
  },
  {
    operationId: "bytebase.v1.DatabaseService.GetSchemaString",
    path: "/bytebase.v1.DatabaseService/GetSchemaString",
    service: "DatabaseService",
    method: "GetSchemaString",
    summary: "GetSchemaString",
    description:
      "Generates schema DDL for a database object.\n Permissions required: bb.databases.getSchema",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetSchemaStringRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.GetSchemaStringResponse",
  },
  {
    operationId: "bytebase.v1.DatabaseService.ListChangelogs",
    path: "/bytebase.v1.DatabaseService/ListChangelogs",
    service: "DatabaseService",
    method: "ListChangelogs",
    summary: "ListChangelogs",
    description:
      "Lists migration history for a database.\n Permissions required: bb.changelogs.list",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ListChangelogsRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.ListChangelogsResponse",
  },
  {
    operationId: "bytebase.v1.DatabaseService.ListDatabases",
    path: "/bytebase.v1.DatabaseService/ListDatabases",
    service: "DatabaseService",
    method: "ListDatabases",
    summary: "ListDatabases",
    description:
      "Lists databases in a project, instance, or workspace.\n Permissions required: bb.projects.get (for project parent), bb.databases.list (for workspace parent), or bb.instances.get (for instance parent)",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ListDatabasesRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ListDatabasesResponse",
  },
  {
    operationId: "bytebase.v1.DatabaseService.SyncDatabase",
    path: "/bytebase.v1.DatabaseService/SyncDatabase",
    service: "DatabaseService",
    method: "SyncDatabase",
    summary: "SyncDatabase",
    description:
      "Synchronizes database schema from the instance.\n Permissions required: bb.databases.sync",
    requestSchemaRef: "#/components/schemas/bytebase.v1.SyncDatabaseRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.SyncDatabaseResponse",
  },
  {
    operationId: "bytebase.v1.DatabaseService.UpdateDatabase",
    path: "/bytebase.v1.DatabaseService/UpdateDatabase",
    service: "DatabaseService",
    method: "UpdateDatabase",
    summary: "UpdateDatabase",
    description:
      "Updates database properties such as labels and project assignment.\n Permissions required: bb.databases.update",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UpdateDatabaseRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Database",
  },
  {
    operationId: "bytebase.v1.GroupService.BatchGetGroups",
    path: "/bytebase.v1.GroupService/BatchGetGroups",
    service: "GroupService",
    method: "BatchGetGroups",
    summary: "BatchGetGroups",
    description:
      "Gets multiple groups in a single request.\n Group members or users with bb.groups.get permission can get the group.\n Permissions required: bb.groups.get OR caller is the group member",
    requestSchemaRef: "#/components/schemas/bytebase.v1.BatchGetGroupsRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.BatchGetGroupsResponse",
  },
  {
    operationId: "bytebase.v1.GroupService.CreateGroup",
    path: "/bytebase.v1.GroupService/CreateGroup",
    service: "GroupService",
    method: "CreateGroup",
    summary: "CreateGroup",
    description:
      "Creates a new group.\n Permissions required: bb.groups.create",
    requestSchemaRef: "#/components/schemas/bytebase.v1.CreateGroupRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Group",
  },
  {
    operationId: "bytebase.v1.GroupService.DeleteGroup",
    path: "/bytebase.v1.GroupService/DeleteGroup",
    service: "GroupService",
    method: "DeleteGroup",
    summary: "DeleteGroup",
    description:
      "Deletes a group. Group owners or users with bb.groups.delete permission can delete.\n Permissions required: bb.groups.delete OR caller is group owner",
    requestSchemaRef: "#/components/schemas/bytebase.v1.DeleteGroupRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.GroupService.GetGroup",
    path: "/bytebase.v1.GroupService/GetGroup",
    service: "GroupService",
    method: "GetGroup",
    summary: "GetGroup",
    description:
      "Gets a group by name.\n Group members or users with bb.groups.get permission can get the group.\n Permissions required: bb.groups.get OR caller is the group member",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetGroupRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Group",
  },
  {
    operationId: "bytebase.v1.GroupService.ListGroups",
    path: "/bytebase.v1.GroupService/ListGroups",
    service: "GroupService",
    method: "ListGroups",
    summary: "ListGroups",
    description:
      "Lists all groups in the workspace.\n Permissions required: bb.groups.list",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ListGroupsRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ListGroupsResponse",
  },
  {
    operationId: "bytebase.v1.GroupService.UpdateGroup",
    path: "/bytebase.v1.GroupService/UpdateGroup",
    service: "GroupService",
    method: "UpdateGroup",
    summary: "UpdateGroup",
    description:
      "Updates a group. Group owners or users with bb.groups.update permission can update.\n Permissions required: bb.groups.update OR caller is group owner\n When allow_missing=true, also requires: bb.groups.create",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UpdateGroupRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Group",
  },
  {
    operationId: "bytebase.v1.IdentityProviderService.CreateIdentityProvider",
    path: "/bytebase.v1.IdentityProviderService/CreateIdentityProvider",
    service: "IdentityProviderService",
    method: "CreateIdentityProvider",
    summary: "CreateIdentityProvider",
    description:
      "Creates a new identity provider.\n Permissions required: bb.identityProviders.create",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.CreateIdentityProviderRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.IdentityProvider",
  },
  {
    operationId: "bytebase.v1.IdentityProviderService.DeleteIdentityProvider",
    path: "/bytebase.v1.IdentityProviderService/DeleteIdentityProvider",
    service: "IdentityProviderService",
    method: "DeleteIdentityProvider",
    summary: "DeleteIdentityProvider",
    description:
      "Deletes an identity provider.\n Permissions required: bb.identityProviders.delete",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.DeleteIdentityProviderRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.IdentityProviderService.GetIdentityProvider",
    path: "/bytebase.v1.IdentityProviderService/GetIdentityProvider",
    service: "IdentityProviderService",
    method: "GetIdentityProvider",
    summary: "GetIdentityProvider",
    description:
      "Gets an identity provider by name.\n Permissions required: bb.identityProviders.get",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.GetIdentityProviderRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.IdentityProvider",
  },
  {
    operationId: "bytebase.v1.IdentityProviderService.ListIdentityProviders",
    path: "/bytebase.v1.IdentityProviderService/ListIdentityProviders",
    service: "IdentityProviderService",
    method: "ListIdentityProviders",
    summary: "ListIdentityProviders",
    description:
      "Lists all configured identity providers (public endpoint for login page).\n Permissions required: None",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.ListIdentityProvidersRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.ListIdentityProvidersResponse",
  },
  {
    operationId: "bytebase.v1.IdentityProviderService.TestIdentityProvider",
    path: "/bytebase.v1.IdentityProviderService/TestIdentityProvider",
    service: "IdentityProviderService",
    method: "TestIdentityProvider",
    summary: "TestIdentityProvider",
    description:
      "Tests the connection and configuration of an identity provider.\n Permissions required: bb.identityProviders.update",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.TestIdentityProviderRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.TestIdentityProviderResponse",
  },
  {
    operationId: "bytebase.v1.IdentityProviderService.UpdateIdentityProvider",
    path: "/bytebase.v1.IdentityProviderService/UpdateIdentityProvider",
    service: "IdentityProviderService",
    method: "UpdateIdentityProvider",
    summary: "UpdateIdentityProvider",
    description:
      "Updates an identity provider.\n Permissions required: bb.identityProviders.update\n When allow_missing=true, also requires: bb.identityProviders.create",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.UpdateIdentityProviderRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.IdentityProvider",
  },
  {
    operationId: "bytebase.v1.InstanceRoleService.ListInstanceRoles",
    path: "/bytebase.v1.InstanceRoleService/ListInstanceRoles",
    service: "InstanceRoleService",
    method: "ListInstanceRoles",
    summary: "ListInstanceRoles",
    description:
      "Lists all database roles in an instance.\n Permissions required: bb.instanceRoles.list",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.ListInstanceRolesRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.ListInstanceRolesResponse",
  },
  {
    operationId: "bytebase.v1.InstanceService.AddDataSource",
    path: "/bytebase.v1.InstanceService/AddDataSource",
    service: "InstanceService",
    method: "AddDataSource",
    summary: "AddDataSource",
    description:
      "Adds a read-only data source to an instance.\n Permissions required: bb.instances.update",
    requestSchemaRef: "#/components/schemas/bytebase.v1.AddDataSourceRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Instance",
  },
  {
    operationId: "bytebase.v1.InstanceService.BatchSyncInstances",
    path: "/bytebase.v1.InstanceService/BatchSyncInstances",
    service: "InstanceService",
    method: "BatchSyncInstances",
    summary: "BatchSyncInstances",
    description:
      "Syncs multiple instances in a single request.\n Permissions required: bb.instances.sync",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.BatchSyncInstancesRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.BatchSyncInstancesResponse",
  },
  {
    operationId: "bytebase.v1.InstanceService.BatchUpdateInstances",
    path: "/bytebase.v1.InstanceService/BatchUpdateInstances",
    service: "InstanceService",
    method: "BatchUpdateInstances",
    summary: "BatchUpdateInstances",
    description:
      "Updates multiple instances in a single request.\n Permissions required: bb.instances.update",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.BatchUpdateInstancesRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.BatchUpdateInstancesResponse",
  },
  {
    operationId: "bytebase.v1.InstanceService.CreateInstance",
    path: "/bytebase.v1.InstanceService/CreateInstance",
    service: "InstanceService",
    method: "CreateInstance",
    summary: "CreateInstance",
    description:
      "Creates a new database instance.\n Permissions required: bb.instances.create",
    requestSchemaRef: "#/components/schemas/bytebase.v1.CreateInstanceRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Instance",
  },
  {
    operationId: "bytebase.v1.InstanceService.DeleteInstance",
    path: "/bytebase.v1.InstanceService/DeleteInstance",
    service: "InstanceService",
    method: "DeleteInstance",
    summary: "DeleteInstance",
    description:
      "Deletes or soft-deletes a database instance.\n Permissions required: bb.instances.delete",
    requestSchemaRef: "#/components/schemas/bytebase.v1.DeleteInstanceRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.InstanceService.GetInstance",
    path: "/bytebase.v1.InstanceService/GetInstance",
    service: "InstanceService",
    method: "GetInstance",
    summary: "GetInstance",
    description:
      "Gets a database instance by name.\n Permissions required: bb.instances.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetInstanceRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Instance",
  },
  {
    operationId: "bytebase.v1.InstanceService.ListInstanceDatabase",
    path: "/bytebase.v1.InstanceService/ListInstanceDatabase",
    service: "InstanceService",
    method: "ListInstanceDatabase",
    summary: "ListInstanceDatabase",
    description:
      "Lists all databases within an instance without creating them.\n Permissions required: bb.instances.get",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.ListInstanceDatabaseRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.ListInstanceDatabaseResponse",
  },
  {
    operationId: "bytebase.v1.InstanceService.ListInstances",
    path: "/bytebase.v1.InstanceService/ListInstances",
    service: "InstanceService",
    method: "ListInstances",
    summary: "ListInstances",
    description:
      "Lists all database instances.\n Permissions required: bb.instances.list",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ListInstancesRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ListInstancesResponse",
  },
  {
    operationId: "bytebase.v1.InstanceService.RemoveDataSource",
    path: "/bytebase.v1.InstanceService/RemoveDataSource",
    service: "InstanceService",
    method: "RemoveDataSource",
    summary: "RemoveDataSource",
    description:
      "Removes a read-only data source from an instance.\n Permissions required: bb.instances.update",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.RemoveDataSourceRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Instance",
  },
  {
    operationId: "bytebase.v1.InstanceService.SyncInstance",
    path: "/bytebase.v1.InstanceService/SyncInstance",
    service: "InstanceService",
    method: "SyncInstance",
    summary: "SyncInstance",
    description:
      "Syncs database schemas and metadata from an instance.\n Permissions required: bb.instances.sync",
    requestSchemaRef: "#/components/schemas/bytebase.v1.SyncInstanceRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.SyncInstanceResponse",
  },
  {
    operationId: "bytebase.v1.InstanceService.UndeleteInstance",
    path: "/bytebase.v1.InstanceService/UndeleteInstance",
    service: "InstanceService",
    method: "UndeleteInstance",
    summary: "UndeleteInstance",
    description:
      "Restores a soft-deleted database instance.\n Permissions required: bb.instances.undelete",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.UndeleteInstanceRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Instance",
  },
  {
    operationId: "bytebase.v1.InstanceService.UpdateDataSource",
    path: "/bytebase.v1.InstanceService/UpdateDataSource",
    service: "InstanceService",
    method: "UpdateDataSource",
    summary: "UpdateDataSource",
    description:
      "Updates a data source configuration.\n Permissions required: bb.instances.update",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.UpdateDataSourceRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Instance",
  },
  {
    operationId: "bytebase.v1.InstanceService.UpdateInstance",
    path: "/bytebase.v1.InstanceService/UpdateInstance",
    service: "InstanceService",
    method: "UpdateInstance",
    summary: "UpdateInstance",
    description:
      "Updates a database instance.\n Permissions required: bb.instances.update",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UpdateInstanceRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Instance",
  },
  {
    operationId: "bytebase.v1.IssueService.ApproveIssue",
    path: "/bytebase.v1.IssueService/ApproveIssue",
    service: "IssueService",
    method: "ApproveIssue",
    summary: "ApproveIssue",
    description:
      "Approves an issue. Access determined by approval flow configuration - caller must be a designated approver for the current approval step.\n Permissions required: None (determined by approval flow)",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ApproveIssueRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Issue",
  },
  {
    operationId: "bytebase.v1.IssueService.BatchUpdateIssuesStatus",
    path: "/bytebase.v1.IssueService/BatchUpdateIssuesStatus",
    service: "IssueService",
    method: "BatchUpdateIssuesStatus",
    summary: "BatchUpdateIssuesStatus",
    description:
      "Updates the status of multiple issues in a single operation.\n Permissions required: bb.issues.update",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.BatchUpdateIssuesStatusRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.BatchUpdateIssuesStatusResponse",
  },
  {
    operationId: "bytebase.v1.IssueService.CreateIssue",
    path: "/bytebase.v1.IssueService/CreateIssue",
    service: "IssueService",
    method: "CreateIssue",
    summary: "CreateIssue",
    description:
      "Creates a new issue for database changes or tasks.\n Permissions required: bb.issues.create",
    requestSchemaRef: "#/components/schemas/bytebase.v1.CreateIssueRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Issue",
  },
  {
    operationId: "bytebase.v1.IssueService.CreateIssueComment",
    path: "/bytebase.v1.IssueService/CreateIssueComment",
    service: "IssueService",
    method: "CreateIssueComment",
    summary: "CreateIssueComment",
    description:
      "Adds a comment to an issue.\n Permissions required: bb.issueComments.create",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.CreateIssueCommentRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.IssueComment",
  },
  {
    operationId: "bytebase.v1.IssueService.GetIssue",
    path: "/bytebase.v1.IssueService/GetIssue",
    service: "IssueService",
    method: "GetIssue",
    summary: "GetIssue",
    description:
      "Retrieves an issue by name.\n Permissions required: bb.issues.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetIssueRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Issue",
  },
  {
    operationId: "bytebase.v1.IssueService.ListIssueComments",
    path: "/bytebase.v1.IssueService/ListIssueComments",
    service: "IssueService",
    method: "ListIssueComments",
    summary: "ListIssueComments",
    description:
      "Lists comments on an issue.\n Permissions required: bb.issueComments.list",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.ListIssueCommentsRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.ListIssueCommentsResponse",
  },
  {
    operationId: "bytebase.v1.IssueService.ListIssues",
    path: "/bytebase.v1.IssueService/ListIssues",
    service: "IssueService",
    method: "ListIssues",
    summary: "ListIssues",
    description:
      "Lists issues in a project.\n Permissions required: bb.issues.list",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ListIssuesRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ListIssuesResponse",
  },
  {
    operationId: "bytebase.v1.IssueService.RejectIssue",
    path: "/bytebase.v1.IssueService/RejectIssue",
    service: "IssueService",
    method: "RejectIssue",
    summary: "RejectIssue",
    description:
      "Rejects an issue. Access determined by approval flow configuration - caller must be a designated approver for the current approval step.\n Permissions required: None (determined by approval flow)",
    requestSchemaRef: "#/components/schemas/bytebase.v1.RejectIssueRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Issue",
  },
  {
    operationId: "bytebase.v1.IssueService.RequestIssue",
    path: "/bytebase.v1.IssueService/RequestIssue",
    service: "IssueService",
    method: "RequestIssue",
    summary: "RequestIssue",
    description:
      "Requests changes on an issue. Access determined by approval flow configuration - caller must be a designated approver for the current approval step.\n Permissions required: None (determined by approval flow)",
    requestSchemaRef: "#/components/schemas/bytebase.v1.RequestIssueRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Issue",
  },
  {
    operationId: "bytebase.v1.IssueService.SearchIssues",
    path: "/bytebase.v1.IssueService/SearchIssues",
    service: "IssueService",
    method: "SearchIssues",
    summary: "SearchIssues",
    description:
      "Search for issues that the caller has the bb.issues.get permission on and also satisfy the specified filter & query.\n Permissions required: bb.issues.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.SearchIssuesRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.SearchIssuesResponse",
  },
  {
    operationId: "bytebase.v1.IssueService.UpdateIssue",
    path: "/bytebase.v1.IssueService/UpdateIssue",
    service: "IssueService",
    method: "UpdateIssue",
    summary: "UpdateIssue",
    description:
      "Updates an issue's properties such as title, description, or labels.\n Permissions required: bb.issues.update",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UpdateIssueRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Issue",
  },
  {
    operationId: "bytebase.v1.IssueService.UpdateIssueComment",
    path: "/bytebase.v1.IssueService/UpdateIssueComment",
    service: "IssueService",
    method: "UpdateIssueComment",
    summary: "UpdateIssueComment",
    description:
      "Updates an existing issue comment.\n Permissions required: bb.issueComments.update",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.UpdateIssueCommentRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.IssueComment",
  },
  {
    operationId: "bytebase.v1.OrgPolicyService.CreatePolicy",
    path: "/bytebase.v1.OrgPolicyService/CreatePolicy",
    service: "OrgPolicyService",
    method: "CreatePolicy",
    summary: "CreatePolicy",
    description:
      "Creates a new organizational policy.\n Permissions required: bb.policies.create",
    requestSchemaRef: "#/components/schemas/bytebase.v1.CreatePolicyRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Policy",
  },
  {
    operationId: "bytebase.v1.OrgPolicyService.DeletePolicy",
    path: "/bytebase.v1.OrgPolicyService/DeletePolicy",
    service: "OrgPolicyService",
    method: "DeletePolicy",
    summary: "DeletePolicy",
    description:
      "Deletes an organizational policy.\n Permissions required: bb.policies.delete",
    requestSchemaRef: "#/components/schemas/bytebase.v1.DeletePolicyRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.OrgPolicyService.GetPolicy",
    path: "/bytebase.v1.OrgPolicyService/GetPolicy",
    service: "OrgPolicyService",
    method: "GetPolicy",
    summary: "GetPolicy",
    description:
      "Retrieves a policy by name.\n Permissions required: bb.policies.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetPolicyRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Policy",
  },
  {
    operationId: "bytebase.v1.OrgPolicyService.ListPolicies",
    path: "/bytebase.v1.OrgPolicyService/ListPolicies",
    service: "OrgPolicyService",
    method: "ListPolicies",
    summary: "ListPolicies",
    description:
      "Lists policies at a specified resource level.\n Permissions required: bb.policies.list",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ListPoliciesRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ListPoliciesResponse",
  },
  {
    operationId: "bytebase.v1.OrgPolicyService.UpdatePolicy",
    path: "/bytebase.v1.OrgPolicyService/UpdatePolicy",
    service: "OrgPolicyService",
    method: "UpdatePolicy",
    summary: "UpdatePolicy",
    description:
      "Updates an existing organizational policy.\n Permissions required: bb.policies.update",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UpdatePolicyRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Policy",
  },
  {
    operationId: "bytebase.v1.PlanService.CancelPlanCheckRun",
    path: "/bytebase.v1.PlanService/CancelPlanCheckRun",
    service: "PlanService",
    method: "CancelPlanCheckRun",
    summary: "CancelPlanCheckRun",
    description:
      "Cancels the plan check run for a deployment plan.\n Permissions required: bb.planCheckRuns.run",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.CancelPlanCheckRunRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.CancelPlanCheckRunResponse",
  },
  {
    operationId: "bytebase.v1.PlanService.CreatePlan",
    path: "/bytebase.v1.PlanService/CreatePlan",
    service: "PlanService",
    method: "CreatePlan",
    summary: "CreatePlan",
    description:
      "Creates a new deployment plan.\n Permissions required: bb.plans.create",
    requestSchemaRef: "#/components/schemas/bytebase.v1.CreatePlanRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Plan",
  },
  {
    operationId: "bytebase.v1.PlanService.GetPlan",
    path: "/bytebase.v1.PlanService/GetPlan",
    service: "PlanService",
    method: "GetPlan",
    summary: "GetPlan",
    description:
      "Retrieves a deployment plan by name.\n Permissions required: bb.plans.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetPlanRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Plan",
  },
  {
    operationId: "bytebase.v1.PlanService.GetPlanCheckRun",
    path: "/bytebase.v1.PlanService/GetPlanCheckRun",
    service: "PlanService",
    method: "GetPlanCheckRun",
    summary: "GetPlanCheckRun",
    description:
      "Gets the plan check run for a deployment plan.\n Permissions required: bb.planCheckRuns.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetPlanCheckRunRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.PlanCheckRun",
  },
  {
    operationId: "bytebase.v1.PlanService.ListPlans",
    path: "/bytebase.v1.PlanService/ListPlans",
    service: "PlanService",
    method: "ListPlans",
    summary: "ListPlans",
    description:
      "Lists deployment plans in a project.\n Permissions required: bb.plans.list",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ListPlansRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ListPlansResponse",
  },
  {
    operationId: "bytebase.v1.PlanService.RunPlanChecks",
    path: "/bytebase.v1.PlanService/RunPlanChecks",
    service: "PlanService",
    method: "RunPlanChecks",
    summary: "RunPlanChecks",
    description:
      "Executes validation checks on a deployment plan.\n Permissions required: bb.planCheckRuns.run",
    requestSchemaRef: "#/components/schemas/bytebase.v1.RunPlanChecksRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.RunPlanChecksResponse",
  },
  {
    operationId: "bytebase.v1.PlanService.UpdatePlan",
    path: "/bytebase.v1.PlanService/UpdatePlan",
    service: "PlanService",
    method: "UpdatePlan",
    summary: "UpdatePlan",
    description:
      "UpdatePlan updates the plan.\n The plan creator and the user with bb.plans.update permission on the project can update the plan.\n Permissions required: bb.plans.update (or creator)",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UpdatePlanRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Plan",
  },
  {
    operationId: "bytebase.v1.ProjectService.AddWebhook",
    path: "/bytebase.v1.ProjectService/AddWebhook",
    service: "ProjectService",
    method: "AddWebhook",
    summary: "AddWebhook",
    description:
      "Adds a webhook to a project for notifications.\n Permissions required: bb.projects.update",
    requestSchemaRef: "#/components/schemas/bytebase.v1.AddWebhookRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Project",
  },
  {
    operationId: "bytebase.v1.ProjectService.BatchDeleteProjects",
    path: "/bytebase.v1.ProjectService/BatchDeleteProjects",
    service: "ProjectService",
    method: "BatchDeleteProjects",
    summary: "BatchDeleteProjects",
    description:
      "Deletes multiple projects in a single operation.\n Permissions required: bb.projects.delete",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.BatchDeleteProjectsRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.ProjectService.BatchGetProjects",
    path: "/bytebase.v1.ProjectService/BatchGetProjects",
    service: "ProjectService",
    method: "BatchGetProjects",
    summary: "BatchGetProjects",
    description:
      "BatchGetProjects retrieves multiple projects by their names.\n Permissions required: bb.projects.get",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.BatchGetProjectsRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.BatchGetProjectsResponse",
  },
  {
    operationId: "bytebase.v1.ProjectService.CreateProject",
    path: "/bytebase.v1.ProjectService/CreateProject",
    service: "ProjectService",
    method: "CreateProject",
    summary: "CreateProject",
    description:
      "Creates a new project in the workspace.\n Permissions required: bb.projects.create",
    requestSchemaRef: "#/components/schemas/bytebase.v1.CreateProjectRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Project",
  },
  {
    operationId: "bytebase.v1.ProjectService.DeleteProject",
    path: "/bytebase.v1.ProjectService/DeleteProject",
    service: "ProjectService",
    method: "DeleteProject",
    summary: "DeleteProject",
    description:
      "Deletes (soft-delete or purge) a project.\n Permissions required: bb.projects.delete",
    requestSchemaRef: "#/components/schemas/bytebase.v1.DeleteProjectRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.ProjectService.GetIamPolicy",
    path: "/bytebase.v1.ProjectService/GetIamPolicy",
    service: "ProjectService",
    method: "GetIamPolicy",
    summary: "GetIamPolicy",
    description:
      "Retrieves the IAM policy for a project.\n Permissions required: bb.projects.getIamPolicy",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetIamPolicyRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.IamPolicy",
  },
  {
    operationId: "bytebase.v1.ProjectService.GetProject",
    path: "/bytebase.v1.ProjectService/GetProject",
    service: "ProjectService",
    method: "GetProject",
    summary: "GetProject",
    description:
      'GetProject retrieves a project by name.\n Users with "bb.projects.get" permission on the workspace or the project owner can access this method.\n Permissions required: bb.projects.get',
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetProjectRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Project",
  },
  {
    operationId: "bytebase.v1.ProjectService.ListProjects",
    path: "/bytebase.v1.ProjectService/ListProjects",
    service: "ProjectService",
    method: "ListProjects",
    summary: "ListProjects",
    description:
      "Lists all projects in the workspace with optional filtering.\n Permissions required: bb.projects.list",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ListProjectsRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ListProjectsResponse",
  },
  {
    operationId: "bytebase.v1.ProjectService.RemoveWebhook",
    path: "/bytebase.v1.ProjectService/RemoveWebhook",
    service: "ProjectService",
    method: "RemoveWebhook",
    summary: "RemoveWebhook",
    description:
      "Removes a webhook from a project.\n Permissions required: bb.projects.update",
    requestSchemaRef: "#/components/schemas/bytebase.v1.RemoveWebhookRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Project",
  },
  {
    operationId: "bytebase.v1.ProjectService.SearchProjects",
    path: "/bytebase.v1.ProjectService/SearchProjects",
    service: "ProjectService",
    method: "SearchProjects",
    summary: "SearchProjects",
    description:
      "Searches for projects with advanced filtering capabilities.\n Permissions required: bb.projects.get (or project-level bb.projects.get for specific projects)",
    requestSchemaRef: "#/components/schemas/bytebase.v1.SearchProjectsRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.SearchProjectsResponse",
  },
  {
    operationId: "bytebase.v1.ProjectService.SetIamPolicy",
    path: "/bytebase.v1.ProjectService/SetIamPolicy",
    service: "ProjectService",
    method: "SetIamPolicy",
    summary: "SetIamPolicy",
    description:
      "Sets the IAM policy for a project.\n Permissions required: bb.projects.setIamPolicy",
    requestSchemaRef: "#/components/schemas/bytebase.v1.SetIamPolicyRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.IamPolicy",
  },
  {
    operationId: "bytebase.v1.ProjectService.TestWebhook",
    path: "/bytebase.v1.ProjectService/TestWebhook",
    service: "ProjectService",
    method: "TestWebhook",
    summary: "TestWebhook",
    description:
      "Tests a webhook by sending a test notification.\n Permissions required: bb.projects.update",
    requestSchemaRef: "#/components/schemas/bytebase.v1.TestWebhookRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.TestWebhookResponse",
  },
  {
    operationId: "bytebase.v1.ProjectService.UndeleteProject",
    path: "/bytebase.v1.ProjectService/UndeleteProject",
    service: "ProjectService",
    method: "UndeleteProject",
    summary: "UndeleteProject",
    description:
      "Restores a soft-deleted project.\n Permissions required: bb.projects.undelete",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UndeleteProjectRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Project",
  },
  {
    operationId: "bytebase.v1.ProjectService.UpdateProject",
    path: "/bytebase.v1.ProjectService/UpdateProject",
    service: "ProjectService",
    method: "UpdateProject",
    summary: "UpdateProject",
    description:
      "Updates an existing project's properties.\n Permissions required: bb.projects.update",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UpdateProjectRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Project",
  },
  {
    operationId: "bytebase.v1.ProjectService.UpdateWebhook",
    path: "/bytebase.v1.ProjectService/UpdateWebhook",
    service: "ProjectService",
    method: "UpdateWebhook",
    summary: "UpdateWebhook",
    description:
      "Updates an existing webhook configuration.\n Permissions required: bb.projects.update",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UpdateWebhookRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Project",
  },
  {
    operationId: "bytebase.v1.ReleaseService.CheckRelease",
    path: "/bytebase.v1.ReleaseService/CheckRelease",
    service: "ReleaseService",
    method: "CheckRelease",
    summary: "CheckRelease",
    description:
      "Validates a release by dry-running checks on target databases.\n Permissions required: bb.releases.check",
    requestSchemaRef: "#/components/schemas/bytebase.v1.CheckReleaseRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.CheckReleaseResponse",
  },
  {
    operationId: "bytebase.v1.ReleaseService.CreateRelease",
    path: "/bytebase.v1.ReleaseService/CreateRelease",
    service: "ReleaseService",
    method: "CreateRelease",
    summary: "CreateRelease",
    description:
      "Creates a new release with SQL files.\n Permissions required: bb.releases.create",
    requestSchemaRef: "#/components/schemas/bytebase.v1.CreateReleaseRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Release",
  },
  {
    operationId: "bytebase.v1.ReleaseService.DeleteRelease",
    path: "/bytebase.v1.ReleaseService/DeleteRelease",
    service: "ReleaseService",
    method: "DeleteRelease",
    summary: "DeleteRelease",
    description:
      "Deletes a release.\n Permissions required: bb.releases.delete",
    requestSchemaRef: "#/components/schemas/bytebase.v1.DeleteReleaseRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.ReleaseService.GetRelease",
    path: "/bytebase.v1.ReleaseService/GetRelease",
    service: "ReleaseService",
    method: "GetRelease",
    summary: "GetRelease",
    description:
      "Retrieves a release by name.\n Permissions required: bb.releases.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetReleaseRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Release",
  },
  {
    operationId: "bytebase.v1.ReleaseService.ListReleaseCategories",
    path: "/bytebase.v1.ReleaseService/ListReleaseCategories",
    service: "ReleaseService",
    method: "ListReleaseCategories",
    summary: "ListReleaseCategories",
    description:
      "Lists all unique categories in a project.\n Permissions required: bb.releases.list",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.ListReleaseCategoriesRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.ListReleaseCategoriesResponse",
  },
  {
    operationId: "bytebase.v1.ReleaseService.ListReleases",
    path: "/bytebase.v1.ReleaseService/ListReleases",
    service: "ReleaseService",
    method: "ListReleases",
    summary: "ListReleases",
    description:
      "Lists releases in a project.\n Permissions required: bb.releases.list",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ListReleasesRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ListReleasesResponse",
  },
  {
    operationId: "bytebase.v1.ReleaseService.UndeleteRelease",
    path: "/bytebase.v1.ReleaseService/UndeleteRelease",
    service: "ReleaseService",
    method: "UndeleteRelease",
    summary: "UndeleteRelease",
    description:
      "Restores a deleted release.\n Permissions required: bb.releases.undelete",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UndeleteReleaseRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Release",
  },
  {
    operationId: "bytebase.v1.ReleaseService.UpdateRelease",
    path: "/bytebase.v1.ReleaseService/UpdateRelease",
    service: "ReleaseService",
    method: "UpdateRelease",
    summary: "UpdateRelease",
    description:
      "Updates an existing release.\n Permissions required: bb.releases.update\n When allow_missing=true, also requires: bb.releases.create",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UpdateReleaseRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Release",
  },
  {
    operationId: "bytebase.v1.ReviewConfigService.CreateReviewConfig",
    path: "/bytebase.v1.ReviewConfigService/CreateReviewConfig",
    service: "ReviewConfigService",
    method: "CreateReviewConfig",
    summary: "CreateReviewConfig",
    description:
      "Creates a new SQL review configuration.\n Permissions required: bb.reviewConfigs.create",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.CreateReviewConfigRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ReviewConfig",
  },
  {
    operationId: "bytebase.v1.ReviewConfigService.DeleteReviewConfig",
    path: "/bytebase.v1.ReviewConfigService/DeleteReviewConfig",
    service: "ReviewConfigService",
    method: "DeleteReviewConfig",
    summary: "DeleteReviewConfig",
    description:
      "Deletes a SQL review configuration.\n Permissions required: bb.reviewConfigs.delete",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.DeleteReviewConfigRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.ReviewConfigService.GetReviewConfig",
    path: "/bytebase.v1.ReviewConfigService/GetReviewConfig",
    service: "ReviewConfigService",
    method: "GetReviewConfig",
    summary: "GetReviewConfig",
    description:
      "Retrieves a SQL review configuration by name.\n Permissions required: bb.reviewConfigs.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetReviewConfigRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ReviewConfig",
  },
  {
    operationId: "bytebase.v1.ReviewConfigService.ListReviewConfigs",
    path: "/bytebase.v1.ReviewConfigService/ListReviewConfigs",
    service: "ReviewConfigService",
    method: "ListReviewConfigs",
    summary: "ListReviewConfigs",
    description:
      "Lists all SQL review configurations.\n Permissions required: bb.reviewConfigs.list",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.ListReviewConfigsRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.ListReviewConfigsResponse",
  },
  {
    operationId: "bytebase.v1.ReviewConfigService.UpdateReviewConfig",
    path: "/bytebase.v1.ReviewConfigService/UpdateReviewConfig",
    service: "ReviewConfigService",
    method: "UpdateReviewConfig",
    summary: "UpdateReviewConfig",
    description:
      "Updates a SQL review configuration.\n Permissions required: bb.reviewConfigs.update\n When allow_missing=true, also requires: bb.reviewConfigs.create",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.UpdateReviewConfigRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ReviewConfig",
  },
  {
    operationId: "bytebase.v1.RevisionService.BatchCreateRevisions",
    path: "/bytebase.v1.RevisionService/BatchCreateRevisions",
    service: "RevisionService",
    method: "BatchCreateRevisions",
    summary: "BatchCreateRevisions",
    description:
      "Creates multiple schema revisions in a single operation.\n Permissions required: bb.revisions.create",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.BatchCreateRevisionsRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.BatchCreateRevisionsResponse",
  },
  {
    operationId: "bytebase.v1.RevisionService.DeleteRevision",
    path: "/bytebase.v1.RevisionService/DeleteRevision",
    service: "RevisionService",
    method: "DeleteRevision",
    summary: "DeleteRevision",
    description:
      "Deletes a schema revision.\n Permissions required: bb.revisions.delete",
    requestSchemaRef: "#/components/schemas/bytebase.v1.DeleteRevisionRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.RevisionService.GetRevision",
    path: "/bytebase.v1.RevisionService/GetRevision",
    service: "RevisionService",
    method: "GetRevision",
    summary: "GetRevision",
    description:
      "Retrieves a schema revision by name.\n Permissions required: bb.revisions.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetRevisionRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Revision",
  },
  {
    operationId: "bytebase.v1.RevisionService.ListRevisions",
    path: "/bytebase.v1.RevisionService/ListRevisions",
    service: "RevisionService",
    method: "ListRevisions",
    summary: "ListRevisions",
    description:
      "Lists schema revisions for a database.\n Permissions required: bb.revisions.list",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ListRevisionsRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ListRevisionsResponse",
  },
  {
    operationId: "bytebase.v1.RoleService.CreateRole",
    path: "/bytebase.v1.RoleService/CreateRole",
    service: "RoleService",
    method: "CreateRole",
    summary: "CreateRole",
    description:
      "Creates a new custom role.\n Permissions required: bb.roles.create",
    requestSchemaRef: "#/components/schemas/bytebase.v1.CreateRoleRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Role",
  },
  {
    operationId: "bytebase.v1.RoleService.DeleteRole",
    path: "/bytebase.v1.RoleService/DeleteRole",
    service: "RoleService",
    method: "DeleteRole",
    summary: "DeleteRole",
    description:
      "Deletes a custom role.\n Permissions required: bb.roles.delete",
    requestSchemaRef: "#/components/schemas/bytebase.v1.DeleteRoleRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.RoleService.GetRole",
    path: "/bytebase.v1.RoleService/GetRole",
    service: "RoleService",
    method: "GetRole",
    summary: "GetRole",
    description:
      "Retrieves a role by name.\n Permissions required: bb.roles.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetRoleRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Role",
  },
  {
    operationId: "bytebase.v1.RoleService.ListRoles",
    path: "/bytebase.v1.RoleService/ListRoles",
    service: "RoleService",
    method: "ListRoles",
    summary: "ListRoles",
    description:
      "Lists roles in the workspace.\n Permissions required: bb.roles.list",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ListRolesRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ListRolesResponse",
  },
  {
    operationId: "bytebase.v1.RoleService.UpdateRole",
    path: "/bytebase.v1.RoleService/UpdateRole",
    service: "RoleService",
    method: "UpdateRole",
    summary: "UpdateRole",
    description:
      "Updates a role's properties.\n Permissions required: bb.roles.update\n When allow_missing=true, also requires: bb.roles.create",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UpdateRoleRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Role",
  },
  {
    operationId: "bytebase.v1.RolloutService.BatchCancelTaskRuns",
    path: "/bytebase.v1.RolloutService/BatchCancelTaskRuns",
    service: "RolloutService",
    method: "BatchCancelTaskRuns",
    summary: "BatchCancelTaskRuns",
    description:
      "Cancels multiple running task executions.\n Permissions required: bb.taskRuns.create (or issue creator for data export issues, or user with rollout policy role for the environment)",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.BatchCancelTaskRunsRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.BatchCancelTaskRunsResponse",
  },
  {
    operationId: "bytebase.v1.RolloutService.BatchRunTasks",
    path: "/bytebase.v1.RolloutService/BatchRunTasks",
    service: "RolloutService",
    method: "BatchRunTasks",
    summary: "BatchRunTasks",
    description:
      "Executes multiple tasks in a rollout stage.\n Permissions required: bb.taskRuns.create (or issue creator for data export issues, or user with rollout policy role for the environment)",
    requestSchemaRef: "#/components/schemas/bytebase.v1.BatchRunTasksRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.BatchRunTasksResponse",
  },
  {
    operationId: "bytebase.v1.RolloutService.BatchSkipTasks",
    path: "/bytebase.v1.RolloutService/BatchSkipTasks",
    service: "RolloutService",
    method: "BatchSkipTasks",
    summary: "BatchSkipTasks",
    description:
      "Skips multiple tasks in a rollout stage.\n Permissions required: bb.taskRuns.create (or issue creator for data export issues, or user with rollout policy role for the environment)",
    requestSchemaRef: "#/components/schemas/bytebase.v1.BatchSkipTasksRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.BatchSkipTasksResponse",
  },
  {
    operationId: "bytebase.v1.RolloutService.CreateRollout",
    path: "/bytebase.v1.RolloutService/CreateRollout",
    service: "RolloutService",
    method: "CreateRollout",
    summary: "CreateRollout",
    description:
      "Creates a new rollout for a plan.\n Permissions required: bb.rollouts.create (or issue creator for data export issues)",
    requestSchemaRef: "#/components/schemas/bytebase.v1.CreateRolloutRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Rollout",
  },
  {
    operationId: "bytebase.v1.RolloutService.GetRollout",
    path: "/bytebase.v1.RolloutService/GetRollout",
    service: "RolloutService",
    method: "GetRollout",
    summary: "GetRollout",
    description:
      "Retrieves a rollout by its plan name.\n Permissions required: bb.rollouts.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetRolloutRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Rollout",
  },
  {
    operationId: "bytebase.v1.RolloutService.GetTaskRun",
    path: "/bytebase.v1.RolloutService/GetTaskRun",
    service: "RolloutService",
    method: "GetTaskRun",
    summary: "GetTaskRun",
    description:
      "Retrieves a task run by name.\n Permissions required: bb.taskRuns.list",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetTaskRunRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.TaskRun",
  },
  {
    operationId: "bytebase.v1.RolloutService.GetTaskRunLog",
    path: "/bytebase.v1.RolloutService/GetTaskRunLog",
    service: "RolloutService",
    method: "GetTaskRunLog",
    summary: "GetTaskRunLog",
    description:
      "Retrieves execution logs for a task run.\n Permissions required: bb.taskRuns.list",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetTaskRunLogRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.TaskRunLog",
  },
  {
    operationId: "bytebase.v1.RolloutService.GetTaskRunSession",
    path: "/bytebase.v1.RolloutService/GetTaskRunSession",
    service: "RolloutService",
    method: "GetTaskRunSession",
    summary: "GetTaskRunSession",
    description:
      "Retrieves database session information for a running task.\n Permissions required: bb.taskRuns.list",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.GetTaskRunSessionRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.TaskRunSession",
  },
  {
    operationId: "bytebase.v1.RolloutService.ListRollouts",
    path: "/bytebase.v1.RolloutService/ListRollouts",
    service: "RolloutService",
    method: "ListRollouts",
    summary: "ListRollouts",
    description:
      "Lists rollouts in a project.\n Permissions required: bb.rollouts.list",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ListRolloutsRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ListRolloutsResponse",
  },
  {
    operationId: "bytebase.v1.RolloutService.ListTaskRuns",
    path: "/bytebase.v1.RolloutService/ListTaskRuns",
    service: "RolloutService",
    method: "ListTaskRuns",
    summary: "ListTaskRuns",
    description:
      "Lists task run executions for a task.\n Permissions required: bb.taskRuns.list",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ListTaskRunsRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ListTaskRunsResponse",
  },
  {
    operationId: "bytebase.v1.RolloutService.PreviewTaskRunRollback",
    path: "/bytebase.v1.RolloutService/PreviewTaskRunRollback",
    service: "RolloutService",
    method: "PreviewTaskRunRollback",
    summary: "PreviewTaskRunRollback",
    description:
      "Generates rollback SQL for a completed task run.\n Permissions required: bb.taskRuns.list",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.PreviewTaskRunRollbackRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.PreviewTaskRunRollbackResponse",
  },
  {
    operationId: "bytebase.v1.SQLService.AICompletion",
    path: "/bytebase.v1.SQLService/AICompletion",
    service: "SQLService",
    method: "AICompletion",
    summary: "AICompletion",
    description:
      "Provides AI-powered SQL completion and generation.\n Permissions required: None (authenticated users only, requires AI to be enabled)",
    requestSchemaRef: "#/components/schemas/bytebase.v1.AICompletionRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.AICompletionResponse",
  },
  {
    operationId: "bytebase.v1.SQLService.DiffMetadata",
    path: "/bytebase.v1.SQLService/DiffMetadata",
    service: "SQLService",
    method: "DiffMetadata",
    summary: "DiffMetadata",
    description:
      "Computes schema differences between two database metadata.\n Permissions required: None",
    requestSchemaRef: "#/components/schemas/bytebase.v1.DiffMetadataRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.DiffMetadataResponse",
  },
  {
    operationId: "bytebase.v1.SQLService.Export",
    path: "/bytebase.v1.SQLService/Export",
    service: "SQLService",
    method: "Export",
    summary: "Export",
    description:
      "Exports query results to a file format.\n Permissions required: bb.databases.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ExportRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ExportResponse",
  },
  {
    operationId: "bytebase.v1.SQLService.Query",
    path: "/bytebase.v1.SQLService/Query",
    service: "SQLService",
    method: "Query",
    summary: "Query",
    description:
      "Executes a read-only SQL query against a database.\n Permissions required: bb.databases.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.QueryRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.QueryResponse",
  },
  {
    operationId: "bytebase.v1.SQLService.SearchQueryHistories",
    path: "/bytebase.v1.SQLService/SearchQueryHistories",
    service: "SQLService",
    method: "SearchQueryHistories",
    summary: "SearchQueryHistories",
    description:
      "SearchQueryHistories searches query histories for the caller.\n Permissions required: None (only returns caller's own query histories)",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.SearchQueryHistoriesRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.SearchQueryHistoriesResponse",
  },
  {
    operationId: "bytebase.v1.ServiceAccountService.CreateServiceAccount",
    path: "/bytebase.v1.ServiceAccountService/CreateServiceAccount",
    service: "ServiceAccountService",
    method: "CreateServiceAccount",
    summary: "CreateServiceAccount",
    description:
      "Creates a new service account.\n For workspace-level: parent is workspaces/{id}, permission bb.serviceAccounts.create on workspace.\n For project-level: parent is projects/{project}, permission bb.serviceAccounts.create on project.",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.CreateServiceAccountRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ServiceAccount",
  },
  {
    operationId: "bytebase.v1.ServiceAccountService.DeleteServiceAccount",
    path: "/bytebase.v1.ServiceAccountService/DeleteServiceAccount",
    service: "ServiceAccountService",
    method: "DeleteServiceAccount",
    summary: "DeleteServiceAccount",
    description:
      "Deletes a service account.\n Permissions required: bb.serviceAccounts.delete",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.DeleteServiceAccountRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.ServiceAccountService.GetServiceAccount",
    path: "/bytebase.v1.ServiceAccountService/GetServiceAccount",
    service: "ServiceAccountService",
    method: "GetServiceAccount",
    summary: "GetServiceAccount",
    description:
      "Gets a service account by name.\n Permissions required: bb.serviceAccounts.get",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.GetServiceAccountRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ServiceAccount",
  },
  {
    operationId: "bytebase.v1.ServiceAccountService.ListServiceAccounts",
    path: "/bytebase.v1.ServiceAccountService/ListServiceAccounts",
    service: "ServiceAccountService",
    method: "ListServiceAccounts",
    summary: "ListServiceAccounts",
    description:
      "Lists service accounts.\n For workspace-level: parent is workspaces/{id}, permission bb.serviceAccounts.list on workspace.\n For project-level: parent is projects/{project}, permission bb.serviceAccounts.list on project.",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.ListServiceAccountsRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.ListServiceAccountsResponse",
  },
  {
    operationId: "bytebase.v1.ServiceAccountService.UndeleteServiceAccount",
    path: "/bytebase.v1.ServiceAccountService/UndeleteServiceAccount",
    service: "ServiceAccountService",
    method: "UndeleteServiceAccount",
    summary: "UndeleteServiceAccount",
    description:
      "Restores a deleted service account.\n Permissions required: bb.serviceAccounts.undelete",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.UndeleteServiceAccountRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ServiceAccount",
  },
  {
    operationId: "bytebase.v1.ServiceAccountService.UpdateServiceAccount",
    path: "/bytebase.v1.ServiceAccountService/UpdateServiceAccount",
    service: "ServiceAccountService",
    method: "UpdateServiceAccount",
    summary: "UpdateServiceAccount",
    description:
      "Updates a service account.\n Permissions required: bb.serviceAccounts.update",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.UpdateServiceAccountRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ServiceAccount",
  },
  {
    operationId: "bytebase.v1.SettingService.GetSetting",
    path: "/bytebase.v1.SettingService/GetSetting",
    service: "SettingService",
    method: "GetSetting",
    summary: "GetSetting",
    description:
      "Retrieves a workspace setting by name.\n Permissions required: bb.settings.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetSettingRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Setting",
  },
  {
    operationId: "bytebase.v1.SettingService.ListSettings",
    path: "/bytebase.v1.SettingService/ListSettings",
    service: "SettingService",
    method: "ListSettings",
    summary: "ListSettings",
    description:
      "Lists all workspace settings.\n Permissions required: bb.settings.list",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ListSettingsRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ListSettingsResponse",
  },
  {
    operationId: "bytebase.v1.SettingService.UpdateSetting",
    path: "/bytebase.v1.SettingService/UpdateSetting",
    service: "SettingService",
    method: "UpdateSetting",
    summary: "UpdateSetting",
    description:
      "Updates a workspace setting.\n Permissions required: bb.settings.set",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UpdateSettingRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Setting",
  },
  {
    operationId: "bytebase.v1.SheetService.BatchCreateSheets",
    path: "/bytebase.v1.SheetService/BatchCreateSheets",
    service: "SheetService",
    method: "BatchCreateSheets",
    summary: "BatchCreateSheets",
    description:
      "Creates multiple SQL sheets in a single operation.\n Permissions required: bb.sheets.create",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.BatchCreateSheetsRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.BatchCreateSheetsResponse",
  },
  {
    operationId: "bytebase.v1.SheetService.CreateSheet",
    path: "/bytebase.v1.SheetService/CreateSheet",
    service: "SheetService",
    method: "CreateSheet",
    summary: "CreateSheet",
    description:
      "Creates a new SQL sheet.\n Permissions required: bb.sheets.create",
    requestSchemaRef: "#/components/schemas/bytebase.v1.CreateSheetRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Sheet",
  },
  {
    operationId: "bytebase.v1.SheetService.GetSheet",
    path: "/bytebase.v1.SheetService/GetSheet",
    service: "SheetService",
    method: "GetSheet",
    summary: "GetSheet",
    description:
      "Retrieves a SQL sheet by name.\n Permissions required: bb.sheets.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetSheetRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Sheet",
  },
  {
    operationId: "bytebase.v1.SubscriptionService.CancelPurchase",
    path: "/bytebase.v1.SubscriptionService/CancelPurchase",
    service: "SubscriptionService",
    method: "CancelPurchase",
    summary: "CancelPurchase",
    description: "CancelPurchase cancels an active subscription (SaaS only).",
    requestSchemaRef: "#/components/schemas/bytebase.v1.CancelPurchaseRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.PurchaseResponse",
  },
  {
    operationId: "bytebase.v1.SubscriptionService.CreatePurchase",
    path: "/bytebase.v1.SubscriptionService/CreatePurchase",
    service: "SubscriptionService",
    method: "CreatePurchase",
    summary: "CreatePurchase",
    description:
      "CreatePurchase creates a new subscription purchase (SaaS only).\n Returns a Stripe Checkout URL for the user to complete payment.",
    requestSchemaRef: "#/components/schemas/bytebase.v1.CreatePurchaseRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.PurchaseResponse",
  },
  {
    operationId: "bytebase.v1.SubscriptionService.GetPaymentInfo",
    path: "/bytebase.v1.SubscriptionService/GetPaymentInfo",
    service: "SubscriptionService",
    method: "GetPaymentInfo",
    summary: "GetPaymentInfo",
    description:
      "GetPaymentInfo returns payment details for the current subscription (SaaS only).",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetPaymentInfoRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.PaymentInfo",
  },
  {
    operationId: "bytebase.v1.SubscriptionService.GetSubscription",
    path: "/bytebase.v1.SubscriptionService/GetSubscription",
    service: "SubscriptionService",
    method: "GetSubscription",
    summary: "GetSubscription",
    description:
      "GetSubscription returns the current subscription.\n If there is no license, we will return a free plan subscription without expiration time.\n If there is expired license, we will return a free plan subscription with the expiration time of the expired license.",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetSubscriptionRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Subscription",
  },
  {
    operationId: "bytebase.v1.SubscriptionService.ListPurchasePlans",
    path: "/bytebase.v1.SubscriptionService/ListPurchasePlans",
    service: "SubscriptionService",
    method: "ListPurchasePlans",
    summary: "ListPurchasePlans",
    description:
      "ListPurchasePlans returns available plans for self-service purchase.",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.ListPurchasePlansRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.ListPurchasePlansResponse",
  },
  {
    operationId: "bytebase.v1.SubscriptionService.UpdatePurchase",
    path: "/bytebase.v1.SubscriptionService/UpdatePurchase",
    service: "SubscriptionService",
    method: "UpdatePurchase",
    summary: "UpdatePurchase",
    description:
      "UpdatePurchase updates an existing subscription (SaaS only).\n May return a Stripe Checkout URL if payment method change is needed.",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UpdatePurchaseRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.PurchaseResponse",
  },
  {
    operationId: "bytebase.v1.SubscriptionService.UploadLicense",
    path: "/bytebase.v1.SubscriptionService/UploadLicense",
    service: "SubscriptionService",
    method: "UploadLicense",
    summary: "UploadLicense",
    description: "Uploads an enterprise license (self-hosted only).",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UploadLicenseRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Subscription",
  },
  {
    operationId: "bytebase.v1.SubscriptionService.VerifyCheckoutSession",
    path: "/bytebase.v1.SubscriptionService/VerifyCheckoutSession",
    service: "SubscriptionService",
    method: "VerifyCheckoutSession",
    summary: "VerifyCheckoutSession",
    description:
      "VerifyCheckoutSession verifies a Stripe Checkout Session status (SaaS only).",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.VerifyCheckoutSessionRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.VerifyCheckoutSessionResponse",
  },
  {
    operationId: "bytebase.v1.UserService.BatchGetUsers",
    path: "/bytebase.v1.UserService/BatchGetUsers",
    service: "UserService",
    method: "BatchGetUsers",
    summary: "BatchGetUsers",
    description:
      "Get the users in batch.\n Any authenticated user can batch get users.\n Permissions required: bb.users.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.BatchGetUsersRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.BatchGetUsersResponse",
  },
  {
    operationId: "bytebase.v1.UserService.CreateUser",
    path: "/bytebase.v1.UserService/CreateUser",
    service: "UserService",
    method: "CreateUser",
    summary: "CreateUser",
    description:
      "Creates a user in the caller's workspace (admin action, self-hosted only).\n In SaaS mode, admins should add users via workspace IAM policy instead.\n Permissions required: bb.users.create",
    requestSchemaRef: "#/components/schemas/bytebase.v1.CreateUserRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.User",
  },
  {
    operationId: "bytebase.v1.UserService.DeleteUser",
    path: "/bytebase.v1.UserService/DeleteUser",
    service: "UserService",
    method: "DeleteUser",
    summary: "DeleteUser",
    description:
      "Deletes a user. Requires bb.users.delete permission with additional validation: the last remaining workspace admin cannot be deleted.\n Permissions required: bb.users.delete",
    requestSchemaRef: "#/components/schemas/bytebase.v1.DeleteUserRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.UserService.GetCurrentUser",
    path: "/bytebase.v1.UserService/GetCurrentUser",
    service: "UserService",
    method: "GetCurrentUser",
    summary: "GetCurrentUser",
    description:
      "Get the current authenticated user.\n Permissions required: None",
    requestSchemaRef: "#/components/schemas/google.protobuf.Empty",
    responseSchemaRef: "#/components/schemas/bytebase.v1.User",
  },
  {
    operationId: "bytebase.v1.UserService.GetUser",
    path: "/bytebase.v1.UserService/GetUser",
    service: "UserService",
    method: "GetUser",
    summary: "GetUser",
    description:
      "Get the user.\n Any authenticated user can get the user.\n Permissions required: bb.users.get",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetUserRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.User",
  },
  {
    operationId: "bytebase.v1.UserService.ListUsers",
    path: "/bytebase.v1.UserService/ListUsers",
    service: "UserService",
    method: "ListUsers",
    summary: "ListUsers",
    description:
      "List all users.\n Any authenticated user can list users.\n Permissions required: bb.users.list",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ListUsersRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.ListUsersResponse",
  },
  {
    operationId: "bytebase.v1.UserService.UndeleteUser",
    path: "/bytebase.v1.UserService/UndeleteUser",
    service: "UserService",
    method: "UndeleteUser",
    summary: "UndeleteUser",
    description:
      "Restores a deleted user.\n Permissions required: bb.users.undelete",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UndeleteUserRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.User",
  },
  {
    operationId: "bytebase.v1.UserService.UpdateEmail",
    path: "/bytebase.v1.UserService/UpdateEmail",
    service: "UserService",
    method: "UpdateEmail",
    summary: "UpdateEmail",
    description:
      "Updates a user's email address.\n Permissions required: bb.users.updateEmail",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UpdateEmailRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.User",
  },
  {
    operationId: "bytebase.v1.UserService.UpdateUser",
    path: "/bytebase.v1.UserService/UpdateUser",
    service: "UserService",
    method: "UpdateUser",
    summary: "UpdateUser",
    description:
      "Updates a user. Users can update their own profile, or users with bb.users.update permission can update any user.\n Note: Email updates are not supported through this API. Use UpdateEmail instead.\n Permissions required: bb.users.update (or self)",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UpdateUserRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.User",
  },
  {
    operationId: "bytebase.v1.WorkloadIdentityService.CreateWorkloadIdentity",
    path: "/bytebase.v1.WorkloadIdentityService/CreateWorkloadIdentity",
    service: "WorkloadIdentityService",
    method: "CreateWorkloadIdentity",
    summary: "CreateWorkloadIdentity",
    description:
      "Creates a new workload identity.\n For workspace-level: parent is workspaces/{id}, permission bb.workloadIdentities.create on workspace.\n For project-level: parent is projects/{project}, permission bb.workloadIdentities.create on project.",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.CreateWorkloadIdentityRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.WorkloadIdentity",
  },
  {
    operationId: "bytebase.v1.WorkloadIdentityService.DeleteWorkloadIdentity",
    path: "/bytebase.v1.WorkloadIdentityService/DeleteWorkloadIdentity",
    service: "WorkloadIdentityService",
    method: "DeleteWorkloadIdentity",
    summary: "DeleteWorkloadIdentity",
    description:
      "Deletes a workload identity.\n Permissions required: bb.workloadIdentities.delete",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.DeleteWorkloadIdentityRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.WorkloadIdentityService.GetWorkloadIdentity",
    path: "/bytebase.v1.WorkloadIdentityService/GetWorkloadIdentity",
    service: "WorkloadIdentityService",
    method: "GetWorkloadIdentity",
    summary: "GetWorkloadIdentity",
    description:
      "Gets a workload identity by name.\n Permissions required: bb.workloadIdentities.get",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.GetWorkloadIdentityRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.WorkloadIdentity",
  },
  {
    operationId: "bytebase.v1.WorkloadIdentityService.ListWorkloadIdentities",
    path: "/bytebase.v1.WorkloadIdentityService/ListWorkloadIdentities",
    service: "WorkloadIdentityService",
    method: "ListWorkloadIdentities",
    summary: "ListWorkloadIdentities",
    description:
      "Lists workload identities.\n For workspace-level: parent is workspaces/{id}, permission bb.workloadIdentities.list on workspace.\n For project-level: parent is projects/{project}, permission bb.workloadIdentities.list on project.",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.ListWorkloadIdentitiesRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.ListWorkloadIdentitiesResponse",
  },
  {
    operationId: "bytebase.v1.WorkloadIdentityService.UndeleteWorkloadIdentity",
    path: "/bytebase.v1.WorkloadIdentityService/UndeleteWorkloadIdentity",
    service: "WorkloadIdentityService",
    method: "UndeleteWorkloadIdentity",
    summary: "UndeleteWorkloadIdentity",
    description:
      "Restores a deleted workload identity.\n Permissions required: bb.workloadIdentities.undelete",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.UndeleteWorkloadIdentityRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.WorkloadIdentity",
  },
  {
    operationId: "bytebase.v1.WorkloadIdentityService.UpdateWorkloadIdentity",
    path: "/bytebase.v1.WorkloadIdentityService/UpdateWorkloadIdentity",
    service: "WorkloadIdentityService",
    method: "UpdateWorkloadIdentity",
    summary: "UpdateWorkloadIdentity",
    description:
      "Updates a workload identity.\n Permissions required: bb.workloadIdentities.update",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.UpdateWorkloadIdentityRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.WorkloadIdentity",
  },
  {
    operationId: "bytebase.v1.WorksheetService.BatchUpdateWorksheetOrganizer",
    path: "/bytebase.v1.WorksheetService/BatchUpdateWorksheetOrganizer",
    service: "WorksheetService",
    method: "BatchUpdateWorksheetOrganizer",
    summary: "BatchUpdateWorksheetOrganizer",
    description:
      "Batch update the organizers of worksheets.\n The access is the same as UpdateWorksheet method.\n Permissions required: bb.worksheets.get (or creator, or project member for shared worksheets)",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.BatchUpdateWorksheetOrganizerRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.BatchUpdateWorksheetOrganizerResponse",
  },
  {
    operationId: "bytebase.v1.WorksheetService.CreateWorksheet",
    path: "/bytebase.v1.WorksheetService/CreateWorksheet",
    service: "WorksheetService",
    method: "CreateWorksheet",
    summary: "CreateWorksheet",
    description:
      "Creates a personal worksheet used in SQL Editor. Any authenticated user can create their own worksheets.\n Permissions required: None (authenticated users only)",
    requestSchemaRef: "#/components/schemas/bytebase.v1.CreateWorksheetRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Worksheet",
  },
  {
    operationId: "bytebase.v1.WorksheetService.DeleteWorksheet",
    path: "/bytebase.v1.WorksheetService/DeleteWorksheet",
    service: "WorksheetService",
    method: "DeleteWorksheet",
    summary: "DeleteWorksheet",
    description:
      "Delete a worksheet.\n The access is the same as UpdateWorksheet method.\n Permissions required: bb.worksheets.manage (or creator, or project member for PROJECT_WRITE worksheets)",
    requestSchemaRef: "#/components/schemas/bytebase.v1.DeleteWorksheetRequest",
    responseSchemaRef: "#/components/schemas/google.protobuf.Empty",
  },
  {
    operationId: "bytebase.v1.WorksheetService.GetWorksheet",
    path: "/bytebase.v1.WorksheetService/GetWorksheet",
    service: "WorksheetService",
    method: "GetWorksheet",
    summary: "GetWorksheet",
    description:
      "Get a worksheet by name.\n The users can access this method if,\n - they are the creator of the worksheet;\n - they have bb.worksheets.get permission on the workspace;\n - the sheet is shared with them with PROJECT_READ and PROJECT_WRITE visibility, and they have bb.projects.get permission on the project.\n Permissions required: bb.worksheets.get (or creator, or project member for shared worksheets)",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetWorksheetRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Worksheet",
  },
  {
    operationId: "bytebase.v1.WorksheetService.SearchWorksheets",
    path: "/bytebase.v1.WorksheetService/SearchWorksheets",
    service: "WorksheetService",
    method: "SearchWorksheets",
    summary: "SearchWorksheets",
    description:
      "Search for worksheets.\n This is used for finding my worksheets or worksheets shared by other people.\n The sheet accessibility is the same as GetWorksheet().\n Permissions required: bb.worksheets.get (or creator, or project member for shared worksheets)",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.SearchWorksheetsRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.SearchWorksheetsResponse",
  },
  {
    operationId: "bytebase.v1.WorksheetService.UpdateWorksheet",
    path: "/bytebase.v1.WorksheetService/UpdateWorksheet",
    service: "WorksheetService",
    method: "UpdateWorksheet",
    summary: "UpdateWorksheet",
    description:
      "Update a worksheet.\n The users can access this method if,\n - they are the creator of the worksheet;\n - they have bb.worksheets.manage permission on the workspace;\n - the sheet is shared with them with PROJECT_WRITE visibility, and they have bb.projects.get permission on the project.\n Permissions required: bb.worksheets.manage (or creator, or project member for PROJECT_WRITE worksheets)",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UpdateWorksheetRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Worksheet",
  },
  {
    operationId: "bytebase.v1.WorksheetService.UpdateWorksheetOrganizer",
    path: "/bytebase.v1.WorksheetService/UpdateWorksheetOrganizer",
    service: "WorksheetService",
    method: "UpdateWorksheetOrganizer",
    summary: "UpdateWorksheetOrganizer",
    description:
      "Update the organizer of a worksheet.\n The access is the same as UpdateWorksheet method.\n Permissions required: bb.worksheets.get (or creator, or project member for shared worksheets)",
    requestSchemaRef:
      "#/components/schemas/bytebase.v1.UpdateWorksheetOrganizerRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.WorksheetOrganizer",
  },
  {
    operationId: "bytebase.v1.WorkspaceService.GetIamPolicy",
    path: "/bytebase.v1.WorkspaceService/GetIamPolicy",
    service: "WorkspaceService",
    method: "GetIamPolicy",
    summary: "GetIamPolicy",
    description:
      "Retrieves IAM policy for the workspace.\n Permissions required: bb.workspaces.getIamPolicy",
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetIamPolicyRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.IamPolicy",
  },
  {
    operationId: "bytebase.v1.WorkspaceService.GetWorkspace",
    path: "/bytebase.v1.WorkspaceService/GetWorkspace",
    service: "WorkspaceService",
    method: "GetWorkspace",
    summary: "GetWorkspace",
    description:
      'Gets a workspace by name.\n Supports "workspaces/-" to resolve the current workspace:\n - Authenticated: uses the workspace from JWT context\n - Self-hosted unauthenticated: returns the single workspace\n - SaaS unauthenticated: returns minimal response',
    requestSchemaRef: "#/components/schemas/bytebase.v1.GetWorkspaceRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Workspace",
  },
  {
    operationId: "bytebase.v1.WorkspaceService.ListWorkspaces",
    path: "/bytebase.v1.WorkspaceService/ListWorkspaces",
    service: "WorkspaceService",
    method: "ListWorkspaces",
    summary: "ListWorkspaces",
    description: "Lists all workspaces the current user is a member of.",
    requestSchemaRef: "#/components/schemas/bytebase.v1.ListWorkspacesRequest",
    responseSchemaRef:
      "#/components/schemas/bytebase.v1.ListWorkspacesResponse",
  },
  {
    operationId: "bytebase.v1.WorkspaceService.SetIamPolicy",
    path: "/bytebase.v1.WorkspaceService/SetIamPolicy",
    service: "WorkspaceService",
    method: "SetIamPolicy",
    summary: "SetIamPolicy",
    description:
      "Sets IAM policy for the workspace.\n Permissions required: bb.workspaces.setIamPolicy",
    requestSchemaRef: "#/components/schemas/bytebase.v1.SetIamPolicyRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.IamPolicy",
  },
  {
    operationId: "bytebase.v1.WorkspaceService.UpdateWorkspace",
    path: "/bytebase.v1.WorkspaceService/UpdateWorkspace",
    service: "WorkspaceService",
    method: "UpdateWorkspace",
    summary: "UpdateWorkspace",
    description: "Updates a workspace. Currently only title can be updated.",
    requestSchemaRef: "#/components/schemas/bytebase.v1.UpdateWorkspaceRequest",
    responseSchemaRef: "#/components/schemas/bytebase.v1.Workspace",
  },
];

export const schemas: Record<string, SchemaInfo> = {
  "bytebase.v1.AIChatMessage": {
    type: "object",
    properties: [
      {
        name: "content",
        type: "string",
        description:
          "The text content of the message. Optional for assistant messages that only contain tool calls.",
      },
      {
        name: "role",
        type: "bytebase.v1.AIChatMessageRole",
        description: "The role of the message sender.",
      },
      {
        name: "toolCallId",
        type: "string",
        description:
          "The ID of the tool call this message is responding to. Only present in tool messages.",
      },
      {
        name: "toolCalls",
        type: "array<bytebase.v1.AIChatToolCall>",
        description:
          "Tool calls made by the assistant. Only present in assistant messages.",
      },
    ],
    description: "A single message in the conversation.",
  },
  "bytebase.v1.AIChatMessageRole": {
    type: "enum",
    values: [
      "AI_CHAT_MESSAGE_ROLE_UNSPECIFIED",
      "AI_CHAT_MESSAGE_ROLE_SYSTEM",
      "AI_CHAT_MESSAGE_ROLE_USER",
      "AI_CHAT_MESSAGE_ROLE_ASSISTANT",
      "AI_CHAT_MESSAGE_ROLE_TOOL",
    ],
    description: "Role of a chat message.",
  },
  "bytebase.v1.AIChatRequest": {
    type: "object",
    properties: [
      {
        name: "messages",
        type: "array<bytebase.v1.AIChatMessage>",
        description: "The conversation messages.",
      },
      {
        name: "toolDefinitions",
        type: "array<bytebase.v1.AIChatToolDefinition>",
        description: "The tool definitions available to the AI.",
      },
    ],
    description: "Request message for AIService.Chat.",
  },
  "bytebase.v1.AIChatResponse": {
    type: "object",
    properties: [
      {
        name: "content",
        type: "string",
        description:
          "The text content of the AI response. Optional when the response only contains tool calls.",
      },
      {
        name: "toolCalls",
        type: "array<bytebase.v1.AIChatToolCall>",
        description: "Tool calls the AI wants to make.",
      },
      {
        name: "usage",
        type: "object",
        description: "Token usage for this provider call, when available.",
      },
    ],
    description: "Response message for AIService.Chat.",
  },
  "bytebase.v1.AIChatToolCall": {
    type: "object",
    properties: [
      {
        name: "arguments",
        type: "string",
        description: "The JSON-encoded arguments to pass to the tool.",
      },
      {
        name: "id",
        type: "string",
        description: "The unique ID of this tool call.",
      },
      {
        name: "metadata",
        type: "string",
        description:
          "Opaque provider-specific metadata (e.g., Gemini thought_signature).\n Frontend must echo this back unchanged when sending tool results.",
      },
      {
        name: "name",
        type: "string",
        description: "The name of the tool to call.",
      },
    ],
    description: "A tool call made by the AI.",
  },
  "bytebase.v1.AIChatToolDefinition": {
    type: "object",
    properties: [
      {
        name: "description",
        type: "string",
        description: "A description of what the tool does.",
      },
      {
        name: "name",
        type: "string",
        description: "The name of the tool.",
      },
      {
        name: "parametersSchema",
        type: "string",
        description: "The JSON Schema describing the tool's parameters.",
      },
    ],
    description: "A tool definition that the AI can invoke.",
  },
  "bytebase.v1.AIChatUsage": {
    type: "object",
    properties: [
      {
        name: "totalTokens",
        type: "integer",
        description: "Total tokens used by the provider call.",
      },
    ],
    description: "Token usage for a single AI provider call.",
  },
  "bytebase.v1.AICompletionRequest": {
    type: "object",
    properties: [
      {
        name: "messages",
        type: "array<bytebase.v1.AICompletionRequest.Message>",
      },
    ],
    description: "",
  },
  "bytebase.v1.AICompletionRequest.Message": {
    type: "object",
    properties: [
      {
        name: "content",
        type: "string",
      },
      {
        name: "role",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.AICompletionResponse": {
    type: "object",
    properties: [
      {
        name: "candidates",
        type: "array<bytebase.v1.AICompletionResponse.Candidate>",
        description:
          "candidates is used for results with multiple choices and candidates. Used\n for OpenAI and Gemini.",
      },
    ],
    description: "",
  },
  "bytebase.v1.AICompletionResponse.Candidate": {
    type: "object",
    properties: [
      {
        name: "content",
        type: "bytebase.v1.AICompletionResponse.Candidate.Content",
      },
    ],
    description: "",
  },
  "bytebase.v1.AICompletionResponse.Candidate.Content": {
    type: "object",
    properties: [
      {
        name: "parts",
        type: "array<bytebase.v1.AICompletionResponse.Candidate.Content.Part>",
        description: "parts is used for a result content with multiple parts.",
      },
    ],
    description: "",
  },
  "bytebase.v1.AICompletionResponse.Candidate.Content.Part": {
    type: "object",
    properties: [
      {
        name: "text",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.AISetting": {
    type: "object",
    properties: [
      {
        name: "apiKey",
        type: "string",
      },
      {
        name: "enabled",
        type: "boolean",
      },
      {
        name: "endpoint",
        type: "string",
      },
      {
        name: "model",
        type: "string",
      },
      {
        name: "provider",
        type: "bytebase.v1.AISetting.Provider",
      },
      {
        name: "version",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.AISetting.Provider": {
    type: "enum",
    values: [
      "PROVIDER_UNSPECIFIED",
      "OPEN_AI",
      "CLAUDE",
      "GEMINI",
      "AZURE_OPENAI",
    ],
    description: "",
  },
  "bytebase.v1.AccessGrant": {
    type: "object",
    properties: [
      {
        name: "createTime",
        type: "google.protobuf.Timestamp",
      },
      {
        name: "creator",
        type: "string",
        description: "The creator of the access grant.\n Format: users/{email}",
      },
      {
        name: "issue",
        type: "string",
        description:
          "The issue associated with the access grant.\n Can be empty.\n Format: projects/{project}/issues/{issue}",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the access grant, generated by the server.\n Format: projects/{project}/accessGrants/{access_grant}\n The {access_grant} segment is a server-generated unique ID.",
      },
      {
        name: "query",
        type: "string",
        description: "The query permission granted.",
      },
      {
        name: "reason",
        type: "string",
      },
      {
        name: "status",
        type: "bytebase.v1.AccessGrant.Status",
        description:
          "The status of the access grant.\n An ACTIVE grant with `expire_time` in the past is effectively expired\n and no longer authorizes access. Use `expire_time` to determine\n whether an ACTIVE grant has expired.",
      },
      {
        name: "targets",
        type: "array<string>",
        description:
          "The target databases for this access grant.\n Format: instances/{instance}/databases/{database}",
      },
      {
        name: "unmask",
        type: "boolean",
        description: "Whether the grant allows unmasking sensitive data.",
      },
      {
        name: "updateTime",
        type: "google.protobuf.Timestamp",
      },
    ],
    description: "",
  },
  "bytebase.v1.AccessGrant.Status": {
    type: "enum",
    values: ["STATUS_UNSPECIFIED", "PENDING", "ACTIVE", "REVOKED"],
    description: "The status of the access grant.",
  },
  "bytebase.v1.ActivateAccessGrantRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the access grant to activate.\n Format: projects/{project}/accessGrants/{access_grant}",
      },
    ],
    description: "",
  },
  "bytebase.v1.Activity.Type": {
    type: "enum",
    values: [
      "TYPE_UNSPECIFIED",
      "ISSUE_CREATED",
      "ISSUE_APPROVAL_REQUESTED",
      "ISSUE_SENT_BACK",
      "PIPELINE_FAILED",
      "PIPELINE_COMPLETED",
      "ISSUE_APPROVED",
    ],
    description: "Activity type enumeration.",
  },
  "bytebase.v1.ActuatorInfo": {
    type: "object",
    properties: [
      {
        name: "activatedInstanceCount",
        type: "integer",
        description: "The number of activated database instances.",
      },
      {
        name: "activatedUserCount",
        type: "integer",
        description: "The number of activated users.",
      },
      {
        name: "defaultProject",
        type: "string",
        description:
          "The default project for unassigned databases.\n Format: projects/{id}",
      },
      {
        name: "demo",
        type: "boolean",
        description: "Whether the Bytebase instance is running in demo mode.",
      },
      {
        name: "docker",
        type: "boolean",
        description: "Whether the Bytebase instance is running in Docker.",
      },
      {
        name: "enableSample",
        type: "boolean",
        description: "Whether sample data setup is enabled.",
      },
      {
        name: "externalUrl",
        type: "string",
        description:
          "The external URL where users or webhook callbacks access Bytebase.",
      },
      {
        name: "externalUrlFromFlag",
        type: "boolean",
        description:
          "Whether the external URL is set via command-line flag (and thus cannot be changed via UI).",
      },
      {
        name: "gitCommit",
        type: "string",
        description: "The git commit hash of the build.",
      },
      {
        name: "host",
        type: "string",
        description: "The host address of the Bytebase instance.",
      },
      {
        name: "lastActiveTime",
        type: "google.protobuf.Timestamp",
        description:
          "The last time any API call was made, refreshed on each request.",
      },
      {
        name: "port",
        type: "string",
        description: "The port number of the Bytebase instance.",
      },
      {
        name: "readonly",
        type: "boolean",
        description:
          "Whether the Bytebase instance is running in read-only mode.",
      },
      {
        name: "replicaCount",
        type: "integer",
        description:
          "The number of active replicas (servers sharing the same database).",
      },
      {
        name: "restriction",
        type: "bytebase.v1.Restriction",
      },
      {
        name: "saas",
        type: "boolean",
        description:
          "Whether the Bytebase instance is running in SaaS mode where some features cannot be edited by users.",
      },
      {
        name: "totalInstanceCount",
        type: "integer",
        description: "The total number of database instances.",
      },
      {
        name: "unlicensedFeatures",
        type: "array<string>",
        description: "List of features that are not licensed.",
      },
      {
        name: "version",
        type: "string",
        description: "The Bytebase server version.",
      },
      {
        name: "workspace",
        type: "string",
        description:
          "The unique identifier for the workspace.\n Format: workspaces/{id}",
      },
    ],
    description:
      "System information and configuration for the Bytebase instance.\n Actuator concept is similar to the Spring Boot Actuator.",
  },
  "bytebase.v1.AddDataSourceRequest": {
    type: "object",
    properties: [
      {
        name: "dataSource",
        type: "bytebase.v1.DataSource",
        description:
          "Identified by data source ID.\n Only READ_ONLY data source can be added.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the instance to add a data source to.\n Format: instances/{instance}",
      },
      {
        name: "validateOnly",
        type: "boolean",
        description: "Validate only also tests the data source connection.",
      },
    ],
    description: "",
  },
  "bytebase.v1.AddWebhookRequest": {
    type: "object",
    properties: [
      {
        name: "project",
        type: "string",
        description:
          "The name of the project to add the webhook to.\n Format: projects/{project}",
      },
      {
        name: "webhook",
        type: "bytebase.v1.Webhook",
        description: "The webhook to add.",
      },
    ],
    description: "",
  },
  "bytebase.v1.AdminExecuteRequest": {
    type: "object",
    properties: [
      {
        name: "container",
        type: "string",
        description:
          "Container is the container name to execute the query against, used for\n CosmosDB only.",
      },
      {
        name: "limit",
        type: "integer",
        description: "The maximum number of rows to return.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name is the instance name to execute the query against.\n Format: instances/{instance}/databases/{databaseName}",
      },
      {
        name: "schema",
        type: "string",
        description:
          "The default schema to execute the statement. Equals to the current schema\n in Oracle and search path in Postgres.",
      },
      {
        name: "statement",
        type: "string",
        description: "The SQL statement to execute.",
      },
    ],
    description: "",
  },
  "bytebase.v1.AdminExecuteResponse": {
    type: "object",
    properties: [
      {
        name: "results",
        type: "array<bytebase.v1.QueryResult>",
        description: "The query results.",
      },
    ],
    description: "",
  },
  "bytebase.v1.Advice": {
    type: "object",
    properties: [
      {
        name: "code",
        type: "integer",
        description: "The advice code.",
      },
      {
        name: "content",
        type: "string",
        description: "The advice content.",
      },
      {
        name: "endPosition",
        type: "bytebase.v1.Position",
      },
      {
        name: "ruleType",
        type: "bytebase.v1.Advice.RuleType",
        description: "The type of linting rule that generated this advice.",
      },
      {
        name: "startPosition",
        type: "bytebase.v1.Position",
        description:
          "The start_position is inclusive and the end_position is exclusive.\n TODO: use range instead",
      },
      {
        name: "status",
        type: "bytebase.v1.Advice.Level",
        description: "The advice level.",
      },
      {
        name: "title",
        type: "string",
        description: "The advice title.",
      },
    ],
    description: "",
  },
  "bytebase.v1.Advice.Level": {
    type: "enum",
    values: ["ADVICE_LEVEL_UNSPECIFIED", "SUCCESS", "WARNING", "ERROR"],
    description: "Level represents the severity level of the advice.",
  },
  "bytebase.v1.Advice.RuleType": {
    type: "enum",
    values: ["RULE_TYPE_UNSPECIFIED", "PARSER_BASED", "AI_POWERED"],
    description: "RuleType indicates the source of the linting rule.",
  },
  "bytebase.v1.Algorithm.FullMask": {
    type: "object",
    properties: [
      {
        name: "substitution",
        type: "string",
        description:
          "substitution is the string used to replace the original value, the\n max length of the string is 16 bytes.",
      },
    ],
    description: "",
  },
  "bytebase.v1.Algorithm.InnerOuterMask": {
    type: "object",
    properties: [
      {
        name: "prefixLen",
        type: "integer",
      },
      {
        name: "substitution",
        type: "string",
      },
      {
        name: "suffixLen",
        type: "integer",
      },
      {
        name: "type",
        type: "bytebase.v1.Algorithm.InnerOuterMask.MaskType",
      },
    ],
    description: "",
  },
  "bytebase.v1.Algorithm.InnerOuterMask.MaskType": {
    type: "enum",
    values: ["MASK_TYPE_UNSPECIFIED", "INNER", "OUTER"],
    description: "",
  },
  "bytebase.v1.Algorithm.MD5Mask": {
    type: "object",
    properties: [
      {
        name: "salt",
        type: "string",
        description:
          "salt is the salt value to generate a different hash that with the word alone.",
      },
    ],
    description: "",
  },
  "bytebase.v1.Algorithm.RangeMask": {
    type: "object",
    properties: [
      {
        name: "slices",
        type: "array<bytebase.v1.Algorithm.RangeMask.Slice>",
        description:
          "We store it as a repeated field to face the fact that the original value may have multiple parts should be masked.\n But frontend can be started with a single rule easily.",
      },
    ],
    description: "",
  },
  "bytebase.v1.Algorithm.RangeMask.Slice": {
    type: "object",
    properties: [
      {
        name: "end",
        type: "integer",
        description:
          "end is the end character index (exclusive) of the original value.\n Uses character indices (not byte offsets) for display-oriented masking.",
      },
      {
        name: "start",
        type: "integer",
        description:
          "start is the start character index (0-based) of the original value, should be less than end.\n Uses character indices (not byte offsets) for display-oriented masking.\n Example: For \"你好world\", character index 2 refers to 'w' (the 3rd character).",
      },
      {
        name: "substitution",
        type: "string",
        description:
          "substitution is the string used to replace the OriginalValue[start:end).",
      },
    ],
    description: "",
  },
  "bytebase.v1.Announcement": {
    type: "object",
    properties: [
      {
        name: "level",
        type: "bytebase.v1.Announcement.AlertLevel",
        description: "The alert level of announcement",
      },
      {
        name: "link",
        type: "string",
        description:
          "The optional link, user can follow the link to check extra details",
      },
      {
        name: "text",
        type: "string",
        description: "The text of announcement",
      },
    ],
    description: "",
  },
  "bytebase.v1.Announcement.AlertLevel": {
    type: "enum",
    values: ["ALERT_LEVEL_UNSPECIFIED", "INFO", "WARNING", "CRITICAL"],
    description:
      "We support three levels of AlertLevel: INFO, WARNING, and ERROR.",
  },
  "bytebase.v1.AppIMSetting": {
    type: "object",
    properties: [
      {
        name: "settings",
        type: "array<bytebase.v1.AppIMSetting.IMSetting>",
      },
    ],
    description: "",
  },
  "bytebase.v1.AppIMSetting.DingTalk": {
    type: "object",
    properties: [
      {
        name: "clientId",
        type: "string",
      },
      {
        name: "clientSecret",
        type: "string",
      },
      {
        name: "robotCode",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.AppIMSetting.Feishu": {
    type: "object",
    properties: [
      {
        name: "appId",
        type: "string",
      },
      {
        name: "appSecret",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.AppIMSetting.IMSetting": {
    type: "object",
    properties: [
      {
        name: "type",
        type: "bytebase.v1.WebhookType",
      },
    ],
    description: "",
  },
  "bytebase.v1.AppIMSetting.Lark": {
    type: "object",
    properties: [
      {
        name: "appId",
        type: "string",
      },
      {
        name: "appSecret",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.AppIMSetting.Slack": {
    type: "object",
    properties: [
      {
        name: "token",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.AppIMSetting.Teams": {
    type: "object",
    properties: [
      {
        name: "clientId",
        type: "string",
        description: "Azure AD application (client) ID.",
      },
      {
        name: "clientSecret",
        type: "string",
        description: "Azure AD client secret.",
      },
      {
        name: "tenantId",
        type: "string",
        description: "Azure AD tenant ID (Directory ID).",
      },
    ],
    description: "",
  },
  "bytebase.v1.AppIMSetting.Wecom": {
    type: "object",
    properties: [
      {
        name: "agentId",
        type: "string",
      },
      {
        name: "corpId",
        type: "string",
      },
      {
        name: "secret",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.ApprovalFlow": {
    type: "object",
    properties: [
      {
        name: "roles",
        type: "array<string>",
        description: "The roles required for approval in order.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ApprovalTemplate": {
    type: "object",
    properties: [
      {
        name: "description",
        type: "string",
        description: "The description of the approval template.",
      },
      {
        name: "flow",
        type: "bytebase.v1.ApprovalFlow",
        description: "The approval flow definition.",
      },
      {
        name: "title",
        type: "string",
        description: "The title of the approval template.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ApproveIssueRequest": {
    type: "object",
    properties: [
      {
        name: "comment",
        type: "string",
        description: "The comment explaining the approval decision.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the issue to add an approver.\n Format: projects/{project}/issues/{issue}",
      },
    ],
    description: "",
  },
  "bytebase.v1.AuditData": {
    type: "object",
    properties: [
      {
        name: "policyDelta",
        type: "bytebase.v1.PolicyDelta",
        description: "Changes to IAM policies.",
      },
    ],
    description: "Additional audit data specific to certain operations.",
  },
  "bytebase.v1.AuditLog": {
    type: "object",
    properties: [
      {
        name: "createTime",
        type: "google.protobuf.Timestamp",
        description: "The timestamp when the audit log was created.",
      },
      {
        name: "latency",
        type: "google.protobuf.Duration",
        description: "The duration of the operation.",
      },
      {
        name: "method",
        type: "string",
        description:
          "The method or action being audited.\n For example: /bytebase.v1.SQLService/Query or bb.project.repository.push",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the log.\n Formats:\n - projects/{project}/auditLogs/{uid}\n - workspaces/{workspace}/auditLogs/{uid}",
      },
      {
        name: "request",
        type: "string",
        description: "The request payload in JSON format.",
      },
      {
        name: "requestMetadata",
        type: "bytebase.v1.RequestMetadata",
        description: "Metadata about the request context.",
      },
      {
        name: "resource",
        type: "string",
        description: "The resource associated with this audit log.",
      },
      {
        name: "response",
        type: "string",
        description:
          "The response payload in JSON format.\n Some fields may be omitted if they are too large or contain sensitive information.",
      },
      {
        name: "serviceData",
        type: "google.protobuf.Any",
        description:
          "Service-specific metadata about the request, response, and activities.",
      },
      {
        name: "severity",
        type: "bytebase.v1.AuditLog.Severity",
        description: "The severity level of this audit log entry.",
      },
      {
        name: "status",
        type: "google.rpc.Status",
        description: "The status of the operation.",
      },
      {
        name: "user",
        type: "string",
        description:
          "The user who performed the action.\n Format: users/{email}",
      },
    ],
    description: "Audit log entry recording system activity or API call.",
  },
  "bytebase.v1.AuditLog.Severity": {
    type: "enum",
    values: [
      "SEVERITY_UNSPECIFIED",
      "DEBUG",
      "INFO",
      "NOTICE",
      "WARNING",
      "ERROR",
      "CRITICAL",
      "ALERT",
      "EMERGENCY",
    ],
    description: "Severity level for audit log entries.",
  },
  "bytebase.v1.AuthMethod": {
    type: "enum",
    values: ["AUTH_METHOD_UNSPECIFIED", "IAM", "CUSTOM"],
    description: "Authorization method for RPC calls.",
  },
  "bytebase.v1.BatchCancelTaskRunsRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          "The task name for the taskRuns.\n Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}\n Use `projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/-` to cancel task runs under the same stage.",
      },
      {
        name: "taskRuns",
        type: "array<string>",
        description:
          "The taskRuns to cancel.\n Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun}",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchCreateRevisionsRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          "The parent resource shared by all revisions being created.\n Format: instances/{instance}/databases/{database}",
      },
      {
        name: "requests",
        type: "array<bytebase.v1.CreateRevisionRequest>",
        description:
          "The request message specifying the revisions to create.\n A maximum of 100 revisions can be created in a batch.",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchCreateRevisionsResponse": {
    type: "object",
    properties: [
      {
        name: "revisions",
        type: "array<bytebase.v1.Revision>",
        description: "The created revisions.",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchCreateSheetsRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          "The parent resource where all sheets will be created.\n Format: projects/{project}",
      },
      {
        name: "requests",
        type: "array<bytebase.v1.CreateSheetRequest>",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchCreateSheetsResponse": {
    type: "object",
    properties: [
      {
        name: "sheets",
        type: "array<bytebase.v1.Sheet>",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchDeleteProjectsRequest": {
    type: "object",
    properties: [
      {
        name: "names",
        type: "array<string>",
        description:
          "The names of the projects to delete.\n Format: projects/{project}",
      },
      {
        name: "purge",
        type: "boolean",
        description:
          "If set to true, permanently purge the soft-deleted projects and all related resources.\n This operation is irreversible. Following AIP-165, this should only be used for\n administrative cleanup of old soft-deleted projects.\n All projects must already be soft-deleted for this to work.\n When purge=true, all databases will be moved to the default project before deletion.\n When purge=false (soft delete/archive), the projects and their databases/issues remain unchanged.",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchDeparseRequest": {
    type: "object",
    properties: [
      {
        name: "expressions",
        type: "array<google.api.expr.v1alpha1.Expr>",
        description: "The CEL expression ASTs to deparse.",
      },
    ],
    description: "Request message for batch deparsing CEL expressions.",
  },
  "bytebase.v1.BatchDeparseResponse": {
    type: "object",
    properties: [
      {
        name: "expressions",
        type: "array<string>",
        description: "The deparsed CEL expressions as strings.",
      },
    ],
    description: "Response message for batch deparsing CEL expressions.",
  },
  "bytebase.v1.BatchGetDatabasesRequest": {
    type: "object",
    properties: [
      {
        name: "names",
        type: "array<string>",
        description: "The list of database names to retrieve.",
      },
      {
        name: "parent",
        type: "string",
        description:
          'The parent resource shared by all databases being retrieved.\n - projects/{project}: batch get databases in a project;\n - instances/{instances}: batch get databases in a instance;\n Use "-" as wildcard to batch get databases across parent.',
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchGetDatabasesResponse": {
    type: "object",
    properties: [
      {
        name: "databases",
        type: "array<bytebase.v1.Database>",
        description: "The databases from the specified request.",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchGetGroupsRequest": {
    type: "object",
    properties: [
      {
        name: "names",
        type: "array<string>",
        description: "The group names to retrieve.\n Format: groups/{email}",
      },
    ],
    description: "Request message for batch getting groups.",
  },
  "bytebase.v1.BatchGetGroupsResponse": {
    type: "object",
    properties: [
      {
        name: "groups",
        type: "array<bytebase.v1.Group>",
        description: "The groups from the specified request.",
      },
    ],
    description: "Response message for batch getting groups.",
  },
  "bytebase.v1.BatchGetProjectsRequest": {
    type: "object",
    properties: [
      {
        name: "names",
        type: "array<string>",
        description:
          "The names of projects to retrieve.\n Format: projects/{project}",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchGetProjectsResponse": {
    type: "object",
    properties: [
      {
        name: "projects",
        type: "array<bytebase.v1.Project>",
        description: "The projects from the specified request.",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchGetUsersRequest": {
    type: "object",
    properties: [
      {
        name: "names",
        type: "array<string>",
        description: "The user names to retrieve.\n Format: users/{email}",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchGetUsersResponse": {
    type: "object",
    properties: [
      {
        name: "users",
        type: "array<bytebase.v1.User>",
        description: "The users from the specified request.",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchParseRequest": {
    type: "object",
    properties: [
      {
        name: "expressions",
        type: "array<string>",
        description: "The CEL expression strings to parse.",
      },
    ],
    description: "Request message for batch parsing CEL expressions.",
  },
  "bytebase.v1.BatchParseResponse": {
    type: "object",
    properties: [
      {
        name: "expressions",
        type: "array<google.api.expr.v1alpha1.Expr>",
        description: "The parsed CEL expressions as AST.",
      },
    ],
    description: "Response message for batch parsing CEL expressions.",
  },
  "bytebase.v1.BatchRunTasksRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          "The stage name for the tasks.\n Format: projects/{project}/plans/{plan}/rollout/stages/{stage}",
      },
      {
        name: "runTime",
        type: "object",
        description: "The task run should run after run_time.",
      },
      {
        name: "skipPriorBackup",
        type: "boolean",
        description:
          "If true, skip prior backup for this run even if the task has prior backup enabled.",
      },
      {
        name: "tasks",
        type: "array<string>",
        description:
          "The tasks to run.\n Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchSkipTasksRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          "The stage name for the tasks.\n Format: projects/{project}/plans/{plan}/rollout/stages/{stage}",
      },
      {
        name: "reason",
        type: "string",
        description: "The reason for skipping the tasks.",
      },
      {
        name: "tasks",
        type: "array<string>",
        description:
          "The tasks to skip.\n Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchSyncDatabasesRequest": {
    type: "object",
    properties: [
      {
        name: "names",
        type: "array<string>",
        description: "The list of database names to sync.",
      },
      {
        name: "parent",
        type: "string",
        description:
          "The parent resource shared by all databases being updated.\n Format: instances/{instance}\n If the operation spans parents, a dash (-) may be accepted as a wildcard.",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchSyncInstancesRequest": {
    type: "object",
    properties: [
      {
        name: "requests",
        type: "array<bytebase.v1.SyncInstanceRequest>",
        description:
          "The request message specifying the instances to sync.\n A maximum of 1000 instances can be synced in a batch.",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchUpdateDatabasesRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          "The parent resource shared by all databases being updated.\n Format: instances/{instance}\n If the operation spans parents, a dash (-) may be accepted as a wildcard.\n We only support updating the project of databases for now.",
      },
      {
        name: "requests",
        type: "array<bytebase.v1.UpdateDatabaseRequest>",
        description:
          "The request message specifying the resources to update.\n A maximum of 1000 databases can be modified in a batch.",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchUpdateDatabasesResponse": {
    type: "object",
    properties: [
      {
        name: "databases",
        type: "array<bytebase.v1.Database>",
        description: "Databases updated.",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchUpdateInstancesRequest": {
    type: "object",
    properties: [
      {
        name: "requests",
        type: "array<bytebase.v1.UpdateInstanceRequest>",
        description: "The request message specifying the resources to update.",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchUpdateInstancesResponse": {
    type: "object",
    properties: [
      {
        name: "instances",
        type: "array<bytebase.v1.Instance>",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchUpdateIssuesStatusRequest": {
    type: "object",
    properties: [
      {
        name: "issues",
        type: "array<string>",
        description:
          "The list of issues to update.\n Format: projects/{project}/issues/{issue}",
      },
      {
        name: "parent",
        type: "string",
        description:
          "The parent resource shared by all issues being updated.\n Format: projects/{project}\n If the operation spans parents, a dash (-) may be accepted as a wildcard.\n We only support updating the status of databases for now.",
      },
      {
        name: "reason",
        type: "string",
        description: "The reason for the status change.",
      },
      {
        name: "status",
        type: "bytebase.v1.IssueStatus",
        description: "The new status.",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchUpdateWorksheetOrganizerRequest": {
    type: "object",
    properties: [
      {
        name: "requests",
        type: "array<bytebase.v1.UpdateWorksheetOrganizerRequest>",
      },
    ],
    description: "",
  },
  "bytebase.v1.BatchUpdateWorksheetOrganizerResponse": {
    type: "object",
    properties: [
      {
        name: "worksheetOrganizers",
        type: "array<bytebase.v1.WorksheetOrganizer>",
      },
    ],
    description: "",
  },
  "bytebase.v1.BillingInterval": {
    type: "enum",
    values: ["BILLING_INTERVAL_UNSPECIFIED", "MONTH", "YEAR"],
    description: "",
  },
  "bytebase.v1.Binding": {
    type: "object",
    properties: [
      {
        name: "condition",
        type: "google.type.Expr",
        description:
          'The condition that is associated with this binding, only used in the project IAM policy.\n If the condition evaluates to true, then this binding applies to the current request.\n If the condition evaluates to false, then this binding does not apply to the current request. However, a different role binding might grant the same role to one or more of the principals in this binding.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Support variables:\n resource.database: the database full name in "instances/{instance}/databases/{database}" format, used by the "roles/sqlEditorUser" and "roles/sqlEditorReadUser" roles, support "==" operator.\n resource.schema_name: the schema name, used by the "roles/sqlEditorUser" and "roles/sqlEditorReadUser" roles, support "==" operator.\n resource.table_name: the table name, used by the "roles/sqlEditorUser" and "roles/sqlEditorReadUser" roles, support "==" operator.\n resource.environment_id: the environment to allow the DDL/DML operation in the SQL Editor, only works for the role with bb.sql.ddl or bb.sql.dml permissions. Support "in" operator.\n request.time: the expiration. Only support "<" operation in `request.time < timestamp("{ISO datetime string format}")`.\n\n For example:\n resource.database == "instances/local-pg/databases/postgres" && resource.schema_name in ["public","another_schema"]\n resource.database == "instances/local-pg/databases/bytebase" && resource.schema_name == "public" && resource.table_name in ["audit_log"]\n resource.database == "instances/local-pg/databases/postgres" && resource.environment_id in ["test"]\n request.time < timestamp("2025-04-26T11:24:48.655Z")',
      },
      {
        name: "members",
        type: "array<string>",
        description:
          "Specifies the principals requesting access for a Bytebase resource.\n For users, the member should be: user:{email}\n For groups, the member should be: group:{email}\n For service accounts, the member should be: serviceAccount:{email}\n For workload identities, the member should be: workloadIdentity:{email}",
      },
      {
        name: "parsedExpr",
        type: "google.api.expr.v1alpha1.Expr",
        description: "The parsed expression of the condition.",
      },
      {
        name: "role",
        type: "string",
        description:
          "The role that is assigned to the members.\n Format: roles/{role}",
      },
    ],
    description:
      "Binding associates members with a role and optional conditions.",
  },
  "bytebase.v1.BindingDelta": {
    type: "object",
    properties: [
      {
        name: "action",
        type: "bytebase.v1.BindingDelta.Action",
        description: "The action that was performed on a Binding.",
      },
      {
        name: "condition",
        type: "google.type.Expr",
        description: "The condition that is associated with this binding.",
      },
      {
        name: "member",
        type: "string",
        description: "Follows the same format of Binding.members.",
      },
      {
        name: "role",
        type: "string",
        description:
          "Role that is assigned to `members`.\n For example, `roles/projectOwner`.",
      },
    ],
    description: "A single change to a binding.",
  },
  "bytebase.v1.BindingDelta.Action": {
    type: "enum",
    values: ["ACTION_UNSPECIFIED", "ADD", "REMOVE"],
    description: "Type of action performed on a binding.",
  },
  "bytebase.v1.BoundingBox": {
    type: "object",
    properties: [
      {
        name: "xmax",
        type: "number",
        description: "Maximum X coordinate",
      },
      {
        name: "xmin",
        type: "number",
        description: "Minimum X coordinate",
      },
      {
        name: "ymax",
        type: "number",
        description: "Maximum Y coordinate",
      },
      {
        name: "ymin",
        type: "number",
        description: "Minimum Y coordinate",
      },
    ],
    description:
      "BoundingBox defines the spatial bounds for GEOMETRY spatial indexes.",
  },
  "bytebase.v1.CancelPlanCheckRunRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the plan check run to cancel.\n Format: projects/{project}/plans/{plan}/planCheckRun",
      },
    ],
    description: "",
  },
  "bytebase.v1.Changelog": {
    type: "object",
    properties: [
      {
        name: "createTime",
        type: "google.protobuf.Timestamp",
      },
      {
        name: "name",
        type: "string",
        description:
          "Format: instances/{instance}/databases/{database}/changelogs/{changelog}",
      },
      {
        name: "planTitle",
        type: "string",
        description:
          "The title of the plan associated with this changelog's task run.\n This field is populated by deriving the plan from task_run for display purposes.",
      },
      {
        name: "schema",
        type: "string",
      },
      {
        name: "schemaSize",
        type: "integer",
      },
      {
        name: "status",
        type: "bytebase.v1.Changelog.Status",
      },
      {
        name: "taskRun",
        type: "string",
        description:
          "Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun}",
      },
    ],
    description: "",
  },
  "bytebase.v1.Changelog.Status": {
    type: "enum",
    values: ["STATUS_UNSPECIFIED", "PENDING", "DONE", "FAILED"],
    description: "",
  },
  "bytebase.v1.ChangelogView": {
    type: "enum",
    values: [
      "CHANGELOG_VIEW_UNSPECIFIED",
      "CHANGELOG_VIEW_BASIC",
      "CHANGELOG_VIEW_FULL",
    ],
    description: "",
  },
  "bytebase.v1.CheckConstraintMetadata": {
    type: "object",
    properties: [
      {
        name: "expression",
        type: "string",
        description: "The expression is the expression of a check constraint.",
      },
      {
        name: "name",
        type: "string",
        description: "The name is the name of a check constraint.",
      },
    ],
    description:
      "CheckConstraintMetadata is the metadata for check constraints.",
  },
  "bytebase.v1.CheckReleaseRequest": {
    type: "object",
    properties: [
      {
        name: "customRules",
        type: "string",
        description:
          'Custom linting rules in natural language for AI-powered validation.\n Each rule should be a clear statement describing the desired schema constraint.\n Example: "All tables must have a primary key"\n Example: "VARCHAR columns should specify a maximum length"',
      },
      {
        name: "parent",
        type: "string",
        description: "Format: projects/{project}",
      },
      {
        name: "release",
        type: "bytebase.v1.Release",
        description: "The release to check.",
      },
      {
        name: "targets",
        type: "array<string>",
        description:
          "The targets to dry-run the release.\n Can be database or databaseGroup.\n Format:\n projects/{project}/databaseGroups/{databaseGroup}\n instances/{instance}/databases/{database}",
      },
    ],
    description: "",
  },
  "bytebase.v1.CheckReleaseResponse": {
    type: "object",
    properties: [
      {
        name: "affectedRows",
        type: "integer",
        description: "The total affected rows across all checks.",
      },
      {
        name: "results",
        type: "array<bytebase.v1.CheckReleaseResponse.CheckResult>",
        description: "The check results for each file and target combination.",
      },
      {
        name: "riskLevel",
        type: "bytebase.v1.RiskLevel",
        description: "The aggregated risk level of the check.",
      },
    ],
    description: "",
  },
  "bytebase.v1.CheckReleaseResponse.CheckResult": {
    type: "object",
    properties: [
      {
        name: "advices",
        type: "array<bytebase.v1.Advice>",
        description: "The list of advice for the file and the target.",
      },
      {
        name: "affectedRows",
        type: "integer",
        description:
          "The count of affected rows of the statement on the target.",
      },
      {
        name: "file",
        type: "string",
        description: "The file path that is being checked.",
      },
      {
        name: "riskLevel",
        type: "bytebase.v1.RiskLevel",
        description: "The risk level of the statement on the target.",
      },
      {
        name: "target",
        type: "string",
        description:
          "The target that the check is performed on.\n Should be a database. Format: instances/{instance}/databases/{database}",
      },
    ],
    description: "Check result for a single release file on a target database.",
  },
  "bytebase.v1.ColumnCatalog": {
    type: "object",
    properties: [
      {
        name: "classification",
        type: "string",
        description: "The data classification level for this column.",
      },
      {
        name: "labels",
        type: "object",
        description: "User-defined labels for this column.",
      },
      {
        name: "name",
        type: "string",
        description: "The column name.",
      },
      {
        name: "objectSchema",
        type: "object",
        description: "Object schema for complex column types like JSON.",
      },
      {
        name: "semanticType",
        type: "string",
        description: "The semantic type describing the data purpose.",
      },
    ],
    description: "Column metadata within a table.",
  },
  "bytebase.v1.ColumnCatalog.LabelsEntry": {
    type: "object",
    properties: [
      {
        name: "key",
        type: "string",
      },
      {
        name: "value",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.ColumnMetadata": {
    type: "object",
    properties: [
      {
        name: "characterSet",
        type: "string",
        description: "The character_set is the character_set of a column.",
      },
      {
        name: "collation",
        type: "string",
        description: "The collation is the collation of a column.",
      },
      {
        name: "comment",
        type: "string",
        description: "The comment is the comment of a column.",
      },
      {
        name: "default",
        type: "string",
        description: "The default value of column.",
      },
      {
        name: "defaultConstraintName",
        type: "string",
        description:
          "The default_constraint_name is the name of the default constraint, MSSQL only.\n In MSSQL, default values are implemented as named constraints. When modifying or\n dropping a column's default value, you must reference the constraint by name.\n This field stores the actual constraint name from the database.\n\n Example: A column definition like:\n   CREATE TABLE employees (\n     status NVARCHAR(20) DEFAULT 'active'\n   )\n\n Will create a constraint with an auto-generated name like 'DF__employees__statu__3B75D760'\n or a user-defined name if specified:\n   ALTER TABLE employees ADD CONSTRAINT DF_employees_status DEFAULT 'active' FOR status\n\n To modify the default, you must first drop the existing constraint by name:\n   ALTER TABLE employees DROP CONSTRAINT DF__employees__statu__3B75D760\n   ALTER TABLE employees ADD CONSTRAINT DF_employees_status DEFAULT 'inactive' FOR status\n\n This field is populated when syncing from the database. When empty (e.g., when parsing\n from SQL files), the system cannot automatically drop the constraint.",
      },
      {
        name: "defaultOnNull",
        type: "boolean",
        description:
          "Oracle specific metadata.\n The default_on_null is the default on null of a column.",
      },
      {
        name: "generation",
        type: "bytebase.v1.GenerationMetadata",
        description: "The generation is the generation of a column.",
      },
      {
        name: "hasDefault",
        type: "boolean",
      },
      {
        name: "identityGeneration",
        type: "bytebase.v1.ColumnMetadata.IdentityGeneration",
        description:
          "The identity_generation is for identity columns, PG only.",
      },
      {
        name: "identityIncrement",
        type: "integer",
        description:
          "The identity_increment is for identity columns, MSSQL only.",
      },
      {
        name: "identitySeed",
        type: "integer",
        description: "The identity_seed is for identity columns, MSSQL only.",
      },
      {
        name: "isIdentity",
        type: "boolean",
      },
      {
        name: "name",
        type: "string",
        description: "The name is the name of a column.",
      },
      {
        name: "nullable",
        type: "boolean",
        description: "The nullable is the nullable of a column.",
      },
      {
        name: "onUpdate",
        type: "string",
        description:
          "The on_update is the on update action of a column.\n For MySQL like databases, it's only supported for TIMESTAMP columns with\n CURRENT_TIMESTAMP as on update value.",
      },
      {
        name: "position",
        type: "integer",
        description: "The position is the position in columns.",
      },
      {
        name: "type",
        type: "string",
        description: "The type is the type of a column.",
      },
    ],
    description: "ColumnMetadata is the metadata for columns.",
  },
  "bytebase.v1.ColumnMetadata.IdentityGeneration": {
    type: "enum",
    values: ["IDENTITY_GENERATION_UNSPECIFIED", "ALWAYS", "BY_DEFAULT"],
    description: "",
  },
  "bytebase.v1.CreateAccessGrantRequest": {
    type: "object",
    properties: [
      {
        name: "accessGrant",
        type: "bytebase.v1.AccessGrant",
        description: "The access grant to create.",
      },
      {
        name: "parent",
        type: "string",
        description:
          "The parent project for the access grant.\n Format: projects/{project}",
      },
    ],
    description: "",
  },
  "bytebase.v1.CreateDatabaseGroupRequest": {
    type: "object",
    properties: [
      {
        name: "databaseGroup",
        type: "bytebase.v1.DatabaseGroup",
        description: "The database group to create.",
      },
      {
        name: "databaseGroupId",
        type: "string",
        description:
          "The ID to use for the database group, which will become the final component of\n the database group's resource name.\n\n This value should be 4-63 characters, and valid characters\n are /[a-z][0-9]-/.",
      },
      {
        name: "parent",
        type: "string",
        description:
          "The parent resource where this database group will be created.\n Format: projects/{project}",
      },
      {
        name: "validateOnly",
        type: "boolean",
        description:
          "If set, validate the create request and preview the full database group response, but do not actually create it.",
      },
    ],
    description: "Request message for creating a database group.",
  },
  "bytebase.v1.CreateGroupRequest": {
    type: "object",
    properties: [
      {
        name: "group",
        type: "bytebase.v1.Group",
        description: "The group to create.",
      },
      {
        name: "groupEmail",
        type: "string",
        description:
          "The email to use for the group, which will become the final component\n of the group's resource name.",
      },
    ],
    description: "Request message for creating a group.",
  },
  "bytebase.v1.CreateIdentityProviderRequest": {
    type: "object",
    properties: [
      {
        name: "identityProvider",
        type: "bytebase.v1.IdentityProvider",
        description: "The identity provider to create.",
      },
      {
        name: "identityProviderId",
        type: "string",
        description:
          "The ID to use for the identity provider, which will become the final component of\n the identity provider's resource name.\n\n This value should be 4-63 characters, and valid characters\n are /[a-z][0-9]-/.",
      },
      {
        name: "validateOnly",
        type: "boolean",
        description:
          "If set to true, the request will be validated without actually creating the identity provider.",
      },
    ],
    description: "",
  },
  "bytebase.v1.CreateInstanceRequest": {
    type: "object",
    properties: [
      {
        name: "instance",
        type: "bytebase.v1.Instance",
        description: "The instance to create.",
      },
      {
        name: "instanceId",
        type: "string",
        description:
          "The ID to use for the instance, which will become the final component of\n the instance's resource name.\n\n This value should be 4-63 characters, and valid characters\n are /[a-z][0-9]-/.",
      },
      {
        name: "validateOnly",
        type: "boolean",
        description: "Validate only also tests the data source connection.",
      },
    ],
    description: "",
  },
  "bytebase.v1.CreateIssueCommentRequest": {
    type: "object",
    properties: [
      {
        name: "issueComment",
        type: "bytebase.v1.IssueComment",
        description: "The comment to create.",
      },
      {
        name: "parent",
        type: "string",
        description:
          "The issue name\n Format: projects/{project}/issues/{issue}",
      },
    ],
    description: "",
  },
  "bytebase.v1.CreateIssueRequest": {
    type: "object",
    properties: [
      {
        name: "issue",
        type: "bytebase.v1.Issue",
        description: "The issue to create.",
      },
      {
        name: "parent",
        type: "string",
        description:
          "The parent, which owns this collection of issues.\n Format: projects/{project}",
      },
    ],
    description: "",
  },
  "bytebase.v1.CreatePlanRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          "The parent project where this plan will be created.\n Format: projects/{project}",
      },
      {
        name: "plan",
        type: "bytebase.v1.Plan",
        description: "The plan to create.",
      },
    ],
    description: "",
  },
  "bytebase.v1.CreatePolicyRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          "The parent resource where this instance will be created.\n Workspace resource name: workspaces/{workspace-id}.\n Environment resource name: environments/environment-id.\n Instance resource name: instances/instance-id.\n Database resource name: instances/instance-id/databases/database-name.",
      },
      {
        name: "policy",
        type: "bytebase.v1.Policy",
        description: "The policy to create.",
      },
      {
        name: "type",
        type: "bytebase.v1.PolicyType",
        description: "The type of policy to create.",
      },
    ],
    description: "",
  },
  "bytebase.v1.CreateProjectRequest": {
    type: "object",
    properties: [
      {
        name: "project",
        type: "bytebase.v1.Project",
        description: "The project to create.",
      },
      {
        name: "projectId",
        type: "string",
        description:
          "The ID to use for the project, which will become the final component of\n the project's resource name.\n\n This value should be 4-63 characters, and valid characters\n are /[a-z][0-9]-/.",
      },
    ],
    description: "",
  },
  "bytebase.v1.CreatePurchaseRequest": {
    type: "object",
    properties: [
      {
        name: "interval",
        type: "bytebase.v1.BillingInterval",
      },
      {
        name: "plan",
        type: "bytebase.v1.PlanType",
      },
      {
        name: "seats",
        type: "integer",
      },
    ],
    description: "",
  },
  "bytebase.v1.CreateReleaseRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description: "Format: projects/{project}",
      },
      {
        name: "release",
        type: "bytebase.v1.Release",
        description: "The release to create.",
      },
      {
        name: "releaseIdTemplate",
        type: "string",
        description:
          'Template for release ID generation.\n Available variables: {date}, {time}, {timestamp}, {iteration}.\n Example: "release_{date}-RC{iteration}" generates "release_20260119-RC00".\n Default: "release_{date}-RC{iteration}".',
      },
      {
        name: "releaseIdTimezone",
        type: "string",
        description:
          'Timezone for {date} and {time} variables in the template.\n Must be a valid IANA timezone (e.g., "UTC", "America/Los_Angeles").\n Default: "UTC".',
      },
    ],
    description: "",
  },
  "bytebase.v1.CreateReviewConfigRequest": {
    type: "object",
    properties: [
      {
        name: "reviewConfig",
        type: "bytebase.v1.ReviewConfig",
        description: "The SQL review config to create.",
      },
    ],
    description: "",
  },
  "bytebase.v1.CreateRevisionRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description: "Format: instances/{instance}/databases/{database}",
      },
      {
        name: "revision",
        type: "bytebase.v1.Revision",
        description: "The revision to create.",
      },
    ],
    description: "",
  },
  "bytebase.v1.CreateRoleRequest": {
    type: "object",
    properties: [
      {
        name: "role",
        type: "bytebase.v1.Role",
        description: "The role to create.",
      },
      {
        name: "roleId",
        type: "string",
        description:
          "The ID to use for the role, which will become the final component\n of the role's resource name.\n\n This value should be 4-63 characters, and valid characters\n are /[a-z][A-Z][0-9]/.",
      },
    ],
    description: "",
  },
  "bytebase.v1.CreateRolloutRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          "The parent plan for which this rollout will be created.\n Format: projects/{project}/plans/{plan}",
      },
      {
        name: "target",
        type: "string",
        description:
          'Create the rollout only for the specified target.\n Format: environments/{environment}\n If unspecified, all stages are created.\n If set to "", no stages are created.',
      },
    ],
    description: "",
  },
  "bytebase.v1.CreateServiceAccountRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          "The parent resource where this service account will be created.\n Format: projects/{project} for project-level, workspaces/{id} for workspace-level.",
      },
      {
        name: "serviceAccount",
        type: "bytebase.v1.ServiceAccount",
        description: "The service account to create.",
      },
      {
        name: "serviceAccountId",
        type: "string",
        description:
          "The ID to use for the service account, which will become the final component\n of the service account's email in the format: {service_account_id}@service.bytebase.com\n or {service_account_id}@{project-id}.service.bytebase.com",
      },
    ],
    description: "Request message for creating a service account.",
  },
  "bytebase.v1.CreateSheetRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          "The parent resource where this sheet will be created.\n Format: projects/{project}",
      },
      {
        name: "sheet",
        type: "bytebase.v1.Sheet",
        description: "The sheet to create.",
      },
    ],
    description: "",
  },
  "bytebase.v1.CreateUserRequest": {
    type: "object",
    properties: [
      {
        name: "user",
        type: "bytebase.v1.User",
        description: "The user to create.",
      },
    ],
    description: "",
  },
  "bytebase.v1.CreateWorkloadIdentityRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          "The parent resource where this workload identity will be created.\n Format: projects/{project} for project-level, workspaces/{id} for workspace-level.",
      },
      {
        name: "workloadIdentity",
        type: "bytebase.v1.WorkloadIdentity",
        description: "The workload identity to create.",
      },
      {
        name: "workloadIdentityId",
        type: "string",
        description:
          "The ID to use for the workload identity, which will become the final component\n of the workload identity's email in the format: {workload_identity_id}@workload.bytebase.com\n or {workload_identity_id}@{project-id}.workload.bytebase.com",
      },
    ],
    description: "Request message for creating a workload identity.",
  },
  "bytebase.v1.CreateWorksheetRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          "The parent resource where this worksheet will be created.\n Format: projects/{project}",
      },
      {
        name: "worksheet",
        type: "bytebase.v1.Worksheet",
        description: "The worksheet to create.",
      },
    ],
    description: "",
  },
  "bytebase.v1.DataClassificationSetting": {
    type: "object",
    properties: [
      {
        name: "configs",
        type: "array<bytebase.v1.DataClassificationSetting.DataClassificationConfig>",
      },
    ],
    description: "",
  },
  "bytebase.v1.DataClassificationSetting.DataClassificationConfig": {
    type: "object",
    properties: [
      {
        name: "classification",
        type: "object",
        description:
          "classification is the id - DataClassification map.\n The id should in [0-9]+-[0-9]+-[0-9]+ format.",
      },
      {
        name: "id",
        type: "string",
        description:
          "id is the uuid for classification. Each project can chose one classification config.",
      },
      {
        name: "levels",
        type: "array<bytebase.v1.DataClassificationSetting.DataClassificationConfig.Level>",
        description: "levels is user defined level list for classification.",
      },
      {
        name: "title",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.DataClassificationSetting.DataClassificationConfig.ClassificationEntry":
    {
      type: "object",
      properties: [
        {
          name: "key",
          type: "string",
        },
        {
          name: "value",
          type: "bytebase.v1.DataClassificationSetting.DataClassificationConfig.DataClassification",
        },
      ],
      description: "",
    },
  "bytebase.v1.DataClassificationSetting.DataClassificationConfig.DataClassification":
    {
      type: "object",
      properties: [
        {
          name: "id",
          type: "string",
          description:
            "id is the classification id in [0-9]+-[0-9]+-[0-9]+ format.",
        },
        {
          name: "level",
          type: "integer",
          description: "The sensitivity level. Maps to Level.level.",
        },
        {
          name: "title",
          type: "string",
        },
      ],
      description: "",
    },
  "bytebase.v1.DataClassificationSetting.DataClassificationConfig.Level": {
    type: "object",
    properties: [
      {
        name: "level",
        type: "integer",
        description: "The numeric level for ordering. Higher = more sensitive.",
      },
      {
        name: "title",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.DataSource": {
    type: "object",
    properties: [
      {
        name: "additionalAddresses",
        type: "array<bytebase.v1.DataSource.Address>",
        description: "additional_addresses is used for MongoDB replica set.",
      },
      {
        name: "authenticationDatabase",
        type: "string",
        description:
          "authentication_database is the database name to authenticate against, which stores the user credentials.",
      },
      {
        name: "authenticationPrivateKey",
        type: "string",
        description:
          "PKCS#8 private key in PEM format. If it's empty string, no private key is required.\n Used for authentication when connecting to the data source.",
      },
      {
        name: "authenticationPrivateKeyPassphrase",
        type: "string",
        description:
          "Passphrase for the encrypted PKCS#8 private key. Only used when the private key is encrypted.",
      },
      {
        name: "authenticationType",
        type: "bytebase.v1.DataSource.AuthenticationType",
      },
      {
        name: "cluster",
        type: "string",
        description:
          "Cluster is the cluster name for the data source. Used by CockroachDB.",
      },
      {
        name: "database",
        type: "string",
        description: "The name of the database to connect to.",
      },
      {
        name: "directConnection",
        type: "boolean",
        description:
          "direct_connection is used for MongoDB to dispatch all the operations to the node specified in the connection string.",
      },
      {
        name: "externalSecret",
        type: "bytebase.v1.DataSourceExternalSecret",
      },
      {
        name: "extraConnectionParameters",
        type: "object",
        description:
          "Extra connection parameters for the database connection.\n For PostgreSQL HA, this can be used to set target_session_attrs=read-write",
      },
      {
        name: "host",
        type: "string",
        description: "The hostname or IP address of the database server.",
      },
      {
        name: "id",
        type: "string",
        description: "The unique identifier for this data source.",
      },
      {
        name: "masterName",
        type: "string",
        description:
          "master_name is the master name used by connecting redis-master via redis sentinel.",
      },
      {
        name: "masterPassword",
        type: "string",
      },
      {
        name: "masterUsername",
        type: "string",
        description:
          "master_username and master_password are master credentials used by redis sentinel mode.",
      },
      {
        name: "password",
        type: "string",
        description: "The password for database authentication.",
      },
      {
        name: "port",
        type: "string",
        description: "The port number of the database server.",
      },
      {
        name: "redisType",
        type: "bytebase.v1.DataSource.RedisType",
      },
      {
        name: "region",
        type: "string",
        description:
          "region is the location of where the DB is, works for AWS RDS. For example, us-east-1.",
      },
      {
        name: "replicaSet",
        type: "string",
        description: "replica_set is used for MongoDB replica set.",
      },
      {
        name: "saslConfig",
        type: "bytebase.v1.SASLConfig",
      },
      {
        name: "serviceName",
        type: "string",
      },
      {
        name: "sid",
        type: "string",
        description: "sid and service_name are used for Oracle.",
      },
      {
        name: "srv",
        type: "boolean",
        description:
          "srv, authentication_database and replica_set are used for MongoDB.\n srv is a boolean flag that indicates whether the host is a DNS SRV record.",
      },
      {
        name: "sshHost",
        type: "string",
        description:
          "Connection over SSH.\n The hostname of the SSH server agent.\n Required.",
      },
      {
        name: "sshPassword",
        type: "string",
        description:
          "The password to login the server. If it's empty string, no password is required.",
      },
      {
        name: "sshPort",
        type: "string",
        description:
          "The port of the SSH server agent. It's 22 typically.\n Required.",
      },
      {
        name: "sshPrivateKey",
        type: "string",
        description:
          'The private key to login the server. If it\'s empty string, we will use the system default private key from os.Getenv("SSH_AUTH_SOCK").',
      },
      {
        name: "sshUser",
        type: "string",
        description: "The user to login the server.\n Required.",
      },
      {
        name: "sslCa",
        type: "string",
        description: "The SSL certificate authority certificate.",
      },
      {
        name: "sslCert",
        type: "string",
        description: "The SSL client certificate.",
      },
      {
        name: "sslKey",
        type: "string",
        description: "The SSL client private key.",
      },
      {
        name: "type",
        type: "bytebase.v1.DataSourceType",
        description: "The type of data source (ADMIN or READ_ONLY).",
      },
      {
        name: "username",
        type: "string",
        description: "The username for database authentication.",
      },
      {
        name: "useSsl",
        type: "boolean",
        description:
          "Use SSL to connect to the data source. By default, we use system default SSL configuration.",
      },
      {
        name: "verifyTlsCertificate",
        type: "boolean",
        description:
          "verify_tls_certificate enables TLS certificate verification for SSL connections.\n Default is false (no verification) for backward compatibility.\n Set to true for secure connections (recommended for production).\n Only set to false for development or when certificates cannot be properly\n validated (e.g., self-signed certs, VPN environments).",
      },
      {
        name: "warehouseId",
        type: "string",
        description: "warehouse_id is used by Databricks.",
      },
    ],
    description: "",
  },
  "bytebase.v1.DataSource.AWSCredential": {
    type: "object",
    properties: [
      {
        name: "accessKeyId",
        type: "string",
      },
      {
        name: "externalId",
        type: "string",
        description:
          "Optional external ID for additional security when assuming role.\n See: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create_for-user_externalid.html",
      },
      {
        name: "roleArn",
        type: "string",
        description:
          "ARN of IAM role to assume for cross-account access.\n See: https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use.html",
      },
      {
        name: "secretAccessKey",
        type: "string",
      },
      {
        name: "sessionToken",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.DataSource.Address": {
    type: "object",
    properties: [
      {
        name: "host",
        type: "string",
      },
      {
        name: "port",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.DataSource.AuthenticationType": {
    type: "enum",
    values: [
      "AUTHENTICATION_UNSPECIFIED",
      "PASSWORD",
      "GOOGLE_CLOUD_SQL_IAM",
      "AWS_RDS_IAM",
      "AZURE_IAM",
    ],
    description: "",
  },
  "bytebase.v1.DataSource.AzureCredential": {
    type: "object",
    properties: [
      {
        name: "clientId",
        type: "string",
      },
      {
        name: "clientSecret",
        type: "string",
      },
      {
        name: "tenantId",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.DataSource.ExtraConnectionParametersEntry": {
    type: "object",
    properties: [
      {
        name: "key",
        type: "string",
      },
      {
        name: "value",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.DataSource.GCPCredential": {
    type: "object",
    properties: [
      {
        name: "content",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.DataSource.RedisType": {
    type: "enum",
    values: ["REDIS_TYPE_UNSPECIFIED", "STANDALONE", "SENTINEL", "CLUSTER"],
    description: "",
  },
  "bytebase.v1.DataSourceExternalSecret": {
    type: "object",
    properties: [
      {
        name: "authType",
        type: "bytebase.v1.DataSourceExternalSecret.AuthType",
        description:
          "The authentication method for accessing the secret store.",
      },
      {
        name: "engineName",
        type: "string",
        description: "engine name is the name for secret engine.",
      },
      {
        name: "passwordKeyName",
        type: "string",
        description: "the key name for the password.",
      },
      {
        name: "secretName",
        type: "string",
        description: "the secret name in the engine to store the password.",
      },
      {
        name: "secretType",
        type: "bytebase.v1.DataSourceExternalSecret.SecretType",
        description: "The type of external secret store.",
      },
      {
        name: "skipVaultTlsVerification",
        type: "boolean",
        description:
          "TLS configuration for connecting to Vault server.\n These fields are separate from the database TLS configuration in DataSource.\n skip_vault_tls_verification disables TLS certificate verification for Vault connections.\n Default is false (verification enabled) for security.\n Only set to true for development or when certificates cannot be properly validated.",
      },
      {
        name: "url",
        type: "string",
        description: "The URL of the external secret store.",
      },
      {
        name: "vaultSslCa",
        type: "string",
        description: "CA certificate for Vault server verification.",
      },
      {
        name: "vaultSslCert",
        type: "string",
        description:
          "Client certificate for mutual TLS authentication with Vault.",
      },
      {
        name: "vaultSslKey",
        type: "string",
        description:
          "Client private key for mutual TLS authentication with Vault.",
      },
    ],
    description: "",
  },
  "bytebase.v1.DataSourceExternalSecret.AppRoleAuthOption": {
    type: "object",
    properties: [
      {
        name: "mountPath",
        type: "string",
        description: "The path where the approle auth method is mounted.",
      },
      {
        name: "roleId",
        type: "string",
        description: "The role ID for Vault AppRole authentication.",
      },
      {
        name: "secretId",
        type: "string",
        description: "the secret id for the role without ttl.",
      },
      {
        name: "type",
        type: "bytebase.v1.DataSourceExternalSecret.AppRoleAuthOption.SecretType",
        description: "The type of secret for AppRole authentication.",
      },
    ],
    description: "",
  },
  "bytebase.v1.DataSourceExternalSecret.AppRoleAuthOption.SecretType": {
    type: "enum",
    values: ["SECRET_TYPE_UNSPECIFIED", "PLAIN", "ENVIRONMENT"],
    description: "",
  },
  "bytebase.v1.DataSourceExternalSecret.AuthType": {
    type: "enum",
    values: ["AUTH_TYPE_UNSPECIFIED", "TOKEN", "VAULT_APP_ROLE"],
    description: "",
  },
  "bytebase.v1.DataSourceExternalSecret.SecretType": {
    type: "enum",
    values: [
      "SECRET_TYPE_UNSPECIFIED",
      "VAULT_KV_V2",
      "AWS_SECRETS_MANAGER",
      "GCP_SECRET_MANAGER",
      "AZURE_KEY_VAULT",
    ],
    description: "",
  },
  "bytebase.v1.DataSourceType": {
    type: "enum",
    values: ["DATA_SOURCE_UNSPECIFIED", "ADMIN", "READ_ONLY"],
    description: "",
  },
  "bytebase.v1.Database": {
    type: "object",
    properties: [
      {
        name: "backupAvailable",
        type: "boolean",
        description: "The database is available for DML prior backup.",
      },
      {
        name: "effectiveEnvironment",
        type: "string",
        description:
          "The effective environment based on environment tag above and environment\n tag on the instance. Inheritance follows\n https://cloud.google.com/resource-manager/docs/tags/tags-overview.",
      },
      {
        name: "environment",
        type: "string",
        description:
          "The environment resource.\n Format: environments/prod where prod is the environment resource ID.",
      },
      {
        name: "instanceResource",
        type: "bytebase.v1.InstanceResource",
        description: "The instance resource.",
      },
      {
        name: "labels",
        type: "object",
        description: "Labels will be used for deployment and policy control.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the database.\n Format: instances/{instance}/databases/{database}\n {database} is the database name in the instance.",
      },
      {
        name: "project",
        type: "string",
        description: "The project for a database.\n Format: projects/{project}",
      },
      {
        name: "release",
        type: "string",
        description:
          "The release that was last applied to this database.\n Format: projects/{project}/releases/{release_id}\n Example: projects/my-project/releases/release_20260115-RC00",
      },
      {
        name: "state",
        type: "bytebase.v1.State",
        description: "The existence of a database.",
      },
      {
        name: "successfulSyncTime",
        type: "google.protobuf.Timestamp",
        description: "The latest synchronization time.",
      },
      {
        name: "syncError",
        type: "string",
        description: "The error message if sync failed.",
      },
      {
        name: "syncStatus",
        type: "bytebase.v1.SyncStatus",
        description: "The sync status of the database.",
      },
    ],
    description: "",
  },
  "bytebase.v1.Database.LabelsEntry": {
    type: "object",
    properties: [
      {
        name: "key",
        type: "string",
      },
      {
        name: "value",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.DatabaseCatalog": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the database catalog.\n Format: instances/{instance}/databases/{database}/catalog",
      },
      {
        name: "schemas",
        type: "array<bytebase.v1.SchemaCatalog>",
        description: "The schemas in the database.",
      },
    ],
    description:
      "Catalog metadata for a database including schemas, tables, and columns.",
  },
  "bytebase.v1.DatabaseChangeMode": {
    type: "enum",
    values: ["DATABASE_CHANGE_MODE_UNSPECIFIED", "PIPELINE", "EDITOR"],
    description: "",
  },
  "bytebase.v1.DatabaseGroup": {
    type: "object",
    properties: [
      {
        name: "databaseExpr",
        type: "google.type.Expr",
        description:
          'The condition that is associated with this database group.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Support variables:\n resource.environment_id: the environment resource id. Support "==", "!=", "in [XX]", "!(in [xx])" operations.\n resource.instance_id: the instance resource id. Support "==", "!=", "in [XX]", "!(in [xx])", "contains", "matches", "startsWith", "endsWith" operations.\n resource.database_name: the database name. Support "==", "!=", "in [XX]", "!(in [xx])", "contains", "matches", "startsWith", "endsWith" operations.\n resource.database_labels: the database labels. Support map access operations.\n All variables should join with "&&" condition.\n\n For example:\n resource.environment_id == "test" && resource.database_name.startsWith("sample_")\n resource.database_labels["tenant"] == "tenant1"',
      },
      {
        name: "matchedDatabases",
        type: "array<bytebase.v1.DatabaseGroup.Database>",
        description:
          "The list of databases that match the database group condition.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the database group.\n Format: projects/{project}/databaseGroups/{databaseGroup}",
      },
      {
        name: "title",
        type: "string",
        description:
          "The short name used in actual databases specified by users.",
      },
    ],
    description: "A group of databases matched by expressions.",
  },
  "bytebase.v1.DatabaseGroup.Database": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The resource name of the database.\n Format: instances/{instance}/databases/{database}",
      },
    ],
    description: "A database within a database group.",
  },
  "bytebase.v1.DatabaseGroupView": {
    type: "enum",
    values: [
      "DATABASE_GROUP_VIEW_UNSPECIFIED",
      "DATABASE_GROUP_VIEW_BASIC",
      "DATABASE_GROUP_VIEW_FULL",
    ],
    description: "View options for database group responses.",
  },
  "bytebase.v1.DatabaseMetadata": {
    type: "object",
    properties: [
      {
        name: "characterSet",
        type: "string",
        description: "The character_set is the character set of a database.",
      },
      {
        name: "collation",
        type: "string",
        description: "The collation is the collation of a database.",
      },
      {
        name: "extensions",
        type: "array<bytebase.v1.ExtensionMetadata>",
        description: "The extensions is the list of extensions in a database.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The database metadata name.\n\n Format: instances/{instance}/databases/{database}/metadata",
      },
      {
        name: "owner",
        type: "string",
        description: "The owner of the database.",
      },
      {
        name: "schemas",
        type: "array<bytebase.v1.SchemaMetadata>",
        description: "The schemas is the list of schemas in a database.",
      },
      {
        name: "searchPath",
        type: "string",
        description:
          "The search_path is the search path of a PostgreSQL database.",
      },
    ],
    description: "DatabaseMetadata is the metadata for databases.",
  },
  "bytebase.v1.DatabaseSDLSchema": {
    type: "object",
    properties: [
      {
        name: "contentType",
        type: "string",
        description:
          'The MIME type of the schema content.\n Indicates how the client should interpret the schema field.\n Examples:\n - "text/plain; charset=utf-8" for SINGLE_FILE format\n - "application/zip" for MULTI_FILE format',
      },
      {
        name: "schema",
        type: "string",
        description:
          "The SDL schema content.\n - For SINGLE_FILE format: contains the complete SDL schema as a text string.\n - For MULTI_FILE format: contains the ZIP archive as binary data.",
      },
    ],
    description: "DatabaseSDLSchema contains the schema in SDL format.",
  },
  "bytebase.v1.DatabaseSchema": {
    type: "object",
    properties: [
      {
        name: "schema",
        type: "string",
        description: "The schema dump from database.",
      },
    ],
    description: "DatabaseSchema is the metadata for databases.",
  },
  "bytebase.v1.DeleteDatabaseGroupRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the database group to delete.\n Format: projects/{project}/databaseGroups/{databaseGroup}",
      },
    ],
    description: "Request message for deleting a database group.",
  },
  "bytebase.v1.DeleteGroupRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the group to delete.\n Format: groups/{email}",
      },
    ],
    description: "Request message for deleting a group.",
  },
  "bytebase.v1.DeleteIdentityProviderRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the identity provider to delete.\n Format: idps/{identity_provider}",
      },
    ],
    description: "",
  },
  "bytebase.v1.DeleteInstanceRequest": {
    type: "object",
    properties: [
      {
        name: "force",
        type: "boolean",
        description:
          "If set to true, any databases and sheets from this project will also be moved to default project, and all open issues will be closed.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the instance to delete.\n Format: instances/{instance}",
      },
      {
        name: "purge",
        type: "boolean",
        description:
          "If set to true, permanently purge the soft-deleted instance and all related resources.\n This operation is irreversible. Following AIP-165, this should only be used for\n administrative cleanup of old soft-deleted instances.\n The instance must already be soft-deleted for this to work.",
      },
    ],
    description: "",
  },
  "bytebase.v1.DeletePolicyRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The policy's `name` field is used to identify the instance to update.\n Format: {resource name}/policies/{policy type}\n Workspace resource name: workspaces/{workspace-id}.\n Environment resource name: environments/environment-id.\n Instance resource name: instances/instance-id.\n Database resource name: instances/instance-id/databases/database-name.",
      },
    ],
    description: "",
  },
  "bytebase.v1.DeleteProjectRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the project to delete.\n Format: projects/{project}",
      },
      {
        name: "purge",
        type: "boolean",
        description:
          "If set to true, permanently purge the soft-deleted project and all related resources.\n This operation is irreversible. Following AIP-165, this should only be used for\n administrative cleanup of old soft-deleted projects.\n The project must already be soft-deleted for this to work.\n When purge=true, all databases will be moved to the default project before deletion.\n When purge=false (soft delete/archive), the project and its databases/issues remain unchanged.",
      },
    ],
    description: "",
  },
  "bytebase.v1.DeleteReleaseRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the release to delete.\n Format: projects/{project}/releases/{release}",
      },
    ],
    description: "",
  },
  "bytebase.v1.DeleteReviewConfigRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the SQL review config to delete.\n Format: reviewConfigs/{reviewConfig}",
      },
    ],
    description: "",
  },
  "bytebase.v1.DeleteRevisionRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the revision to delete.\n Format: instances/{instance}/databases/{database}/revisions/{revision}",
      },
    ],
    description: "",
  },
  "bytebase.v1.DeleteRoleRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description: "The name of the role to delete.\n Format: roles/{role}",
      },
    ],
    description: "",
  },
  "bytebase.v1.DeleteServiceAccountRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the service account to delete.\n Format: serviceAccounts/{email}",
      },
    ],
    description: "Request message for deleting a service account.",
  },
  "bytebase.v1.DeleteUserRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description: "The name of the user to delete.\n Format: users/{email}",
      },
    ],
    description: "",
  },
  "bytebase.v1.DeleteWorkloadIdentityRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the workload identity to delete.\n Format: workloadIdentities/{email}",
      },
    ],
    description: "Request message for deleting a workload identity.",
  },
  "bytebase.v1.DeleteWorksheetRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the worksheet to delete.\n Format: projects/{project}/worksheets/{worksheet}",
      },
    ],
    description: "",
  },
  "bytebase.v1.DependencyColumn": {
    type: "object",
    properties: [
      {
        name: "column",
        type: "string",
        description: "The column is the name of a reference column.",
      },
      {
        name: "schema",
        type: "string",
        description: "The schema is the schema of a reference column.",
      },
      {
        name: "table",
        type: "string",
        description: "The table is the table of a reference column.",
      },
    ],
    description: "DependencyColumn is the metadata for dependency columns.",
  },
  "bytebase.v1.DependencyTable": {
    type: "object",
    properties: [
      {
        name: "schema",
        type: "string",
        description: "The schema is the schema of a reference table.",
      },
      {
        name: "table",
        type: "string",
        description: "The table is the name of a reference table.",
      },
    ],
    description: "",
  },
  "bytebase.v1.DiffMetadataRequest": {
    type: "object",
    properties: [
      {
        name: "engine",
        type: "bytebase.v1.Engine",
        description: "The database engine of the schema.",
      },
      {
        name: "sourceMetadata",
        type: "bytebase.v1.DatabaseMetadata",
        description: "The metadata of the source schema.",
      },
      {
        name: "targetMetadata",
        type: "bytebase.v1.DatabaseMetadata",
        description: "The metadata of the target schema.",
      },
    ],
    description: "",
  },
  "bytebase.v1.DiffMetadataResponse": {
    type: "object",
    properties: [
      {
        name: "diff",
        type: "string",
        description: "The diff of the metadata.",
      },
    ],
    description: "",
  },
  "bytebase.v1.DiffSchemaRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the database or changelog.\n Format:\n database: instances/{instance}/databases/{database}\n changelog: instances/{instance}/databases/{database}/changelogs/{changelog}",
      },
    ],
    description: "",
  },
  "bytebase.v1.DiffSchemaResponse": {
    type: "object",
    properties: [
      {
        name: "diff",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.DimensionConstraint": {
    type: "object",
    properties: [
      {
        name: "dimension",
        type: "string",
        description: "Dimension name/type (X, Y, Z, M, etc.)",
      },
      {
        name: "maxValue",
        type: "number",
        description: "Maximum value for this dimension",
      },
      {
        name: "minValue",
        type: "number",
        description: "Minimum value for this dimension",
      },
      {
        name: "tolerance",
        type: "number",
        description: "Tolerance for this dimension",
      },
    ],
    description:
      "DimensionConstraint defines constraints for a spatial dimension.",
  },
  "bytebase.v1.DimensionalConfig": {
    type: "object",
    properties: [
      {
        name: "constraints",
        type: "array<bytebase.v1.DimensionConstraint>",
        description: "Coordinate system constraints",
      },
      {
        name: "dataType",
        type: "string",
        description:
          "Spatial data type (GEOMETRY, GEOGRAPHY, POINT, POLYGON, etc.)",
      },
      {
        name: "dimensions",
        type: "integer",
        description: "Number of dimensions (2-4, default 2)",
      },
      {
        name: "srid",
        type: "integer",
        description: "Spatial reference system identifier (SRID)",
      },
    ],
    description:
      "DimensionalConfig defines dimensional and constraint parameters for spatial indexes.",
  },
  "bytebase.v1.Engine": {
    type: "enum",
    values: [
      "ENGINE_UNSPECIFIED",
      "CLICKHOUSE",
      "MYSQL",
      "POSTGRES",
      "SNOWFLAKE",
      "SQLITE",
      "TIDB",
      "MONGODB",
      "REDIS",
      "ORACLE",
      "SPANNER",
      "MSSQL",
      "REDSHIFT",
      "MARIADB",
      "OCEANBASE",
      "STARROCKS",
      "DORIS",
      "HIVE",
      "ELASTICSEARCH",
      "BIGQUERY",
      "DYNAMODB",
      "DATABRICKS",
      "COCKROACHDB",
      "COSMOSDB",
      "TRINO",
      "CASSANDRA",
    ],
    description: "Database engine type.",
  },
  "bytebase.v1.EnumTypeMetadata": {
    type: "object",
    properties: [
      {
        name: "comment",
        type: "string",
        description: "The comment describing the enum type.",
      },
      {
        name: "name",
        type: "string",
        description: "The name of a type.",
      },
      {
        name: "skipDump",
        type: "boolean",
        description:
          "Whether to skip this enum type during schema dump operations.",
      },
      {
        name: "values",
        type: "array<string>",
        description: "The enum values of a type.",
      },
    ],
    description: "",
  },
  "bytebase.v1.EnvironmentSetting": {
    type: "object",
    properties: [
      {
        name: "environments",
        type: "array<bytebase.v1.EnvironmentSetting.Environment>",
      },
    ],
    description: "",
  },
  "bytebase.v1.EnvironmentSetting.Environment": {
    type: "object",
    properties: [
      {
        name: "color",
        type: "string",
      },
      {
        name: "id",
        type: "string",
        description:
          "The resource id of the environment.\n This value should be 4-63 characters, and valid characters\n are /[a-z][0-9]-/.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The resource name of the environment.\n Format: environments/{environment}.\n Output only.",
      },
      {
        name: "tags",
        type: "object",
      },
      {
        name: "title",
        type: "string",
        description: "The display name of the environment.",
      },
    ],
    description: "",
  },
  "bytebase.v1.EnvironmentSetting.Environment.TagsEntry": {
    type: "object",
    properties: [
      {
        name: "key",
        type: "string",
      },
      {
        name: "value",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.EventMetadata": {
    type: "object",
    properties: [
      {
        name: "characterSetClient",
        type: "string",
        description: "The character set used by the client creating the event.",
      },
      {
        name: "collationConnection",
        type: "string",
        description:
          "The collation used for the connection when creating the event.",
      },
      {
        name: "comment",
        type: "string",
        description: "The comment is the comment of an event.",
      },
      {
        name: "definition",
        type: "string",
        description: "The schedule of the event.",
      },
      {
        name: "name",
        type: "string",
        description: "The name of the event.",
      },
      {
        name: "sqlMode",
        type: "string",
        description: "The SQL mode setting for the event.",
      },
      {
        name: "timeZone",
        type: "string",
        description: "The time zone of the event.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ExchangeTokenRequest": {
    type: "object",
    properties: [
      {
        name: "email",
        type: "string",
        description:
          "Workload Identity email for identifying which identity to authenticate as.\n Format: {name}@workload.bytebase.com",
      },
      {
        name: "token",
        type: "string",
        description: "External OIDC token (JWT) from CI/CD platform.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ExchangeTokenResponse": {
    type: "object",
    properties: [
      {
        name: "accessToken",
        type: "string",
        description: "Bytebase access token.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ExportAuditLogsRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          "The filter of the log. It should be a valid CEL expression.\n Check the filter field in the SearchAuditLogsRequest message.",
      },
      {
        name: "format",
        type: "bytebase.v1.ExportFormat",
        description: "The export format.",
      },
      {
        name: "orderBy",
        type: "string",
        description:
          'The order by of the log.\n Only support order by create_time. The default sorting order is ascending.\n For example:\n  - order_by = "create_time asc"\n  - order_by = "create_time desc"',
      },
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of logs to return.\n The service may return fewer than this value.\n If unspecified, at most 10 log entries will be returned.\n The maximum value is 5000; values above 5000 will be coerced to 5000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `ExportAuditLogs` call.\n Provide this to retrieve the subsequent page.",
      },
      {
        name: "parent",
        type: "string",
      },
    ],
    description: "Request message for exporting audit logs.",
  },
  "bytebase.v1.ExportAuditLogsResponse": {
    type: "object",
    properties: [
      {
        name: "content",
        type: "string",
        description: "The exported audit log content in the requested format.",
      },
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token to retrieve next page of log entities.\n Pass this value in the page_token field in the subsequent call\n to retrieve the next page of log entities.",
      },
    ],
    description: "Response message for exporting audit logs.",
  },
  "bytebase.v1.ExportFormat": {
    type: "enum",
    values: ["FORMAT_UNSPECIFIED", "CSV", "JSON", "SQL", "XLSX"],
    description: "Data export format.",
  },
  "bytebase.v1.ExportRequest": {
    type: "object",
    properties: [
      {
        name: "admin",
        type: "boolean",
        description:
          "The admin is used for workspace owner and DBA for exporting data from SQL\n Editor Admin mode. The exported data is not masked.",
      },
      {
        name: "dataSourceId",
        type: "string",
        description:
          "The id of data source.\n If omitted, Export resolves the data source server-side by using the\n single read-only data source when exactly one exists, or the admin data\n source otherwise. It can also be set explicitly to export from the admin\n data source or a specific read-only data source.",
      },
      {
        name: "format",
        type: "bytebase.v1.ExportFormat",
        description: "The export format.",
      },
      {
        name: "limit",
        type: "integer",
        description: "The maximum number of rows to return.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name is the resource name to execute the export against.\n Format: instances/{instance}/databases/{database}\n Format: instances/{instance}\n Format: projects/{project}/plans/{plan}/rollout\n Format: projects/{project}/plans/{plan}/rollout/stages/{stage}",
      },
      {
        name: "password",
        type: "string",
        description: "The zip password provide by users.",
      },
      {
        name: "schema",
        type: "string",
        description:
          "The default schema to search objects. Equals to the current schema in\n Oracle and search path in Postgres.",
      },
      {
        name: "statement",
        type: "string",
        description: "The SQL statement to execute.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ExportResponse": {
    type: "object",
    properties: [
      {
        name: "content",
        type: "string",
        description: "The export file content.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ExtensionMetadata": {
    type: "object",
    properties: [
      {
        name: "description",
        type: "string",
        description: "The description is the description of an extension.",
      },
      {
        name: "name",
        type: "string",
        description: "The name is the name of an extension.",
      },
      {
        name: "schema",
        type: "string",
        description:
          "The schema is the extension that is installed to. But the extension usage\n is not limited to the schema.",
      },
      {
        name: "version",
        type: "string",
        description: "The version is the version of an extension.",
      },
    ],
    description: "ExtensionMetadata is the metadata for extensions.",
  },
  "bytebase.v1.ExternalTableMetadata": {
    type: "object",
    properties: [
      {
        name: "columns",
        type: "array<bytebase.v1.ColumnMetadata>",
        description:
          "The columns is the ordered list of columns in a foreign table.",
      },
      {
        name: "externalDatabaseName",
        type: "string",
        description:
          "The external_database_name is the name of the external database.",
      },
      {
        name: "externalServerName",
        type: "string",
        description:
          "The external_server_name is the name of the external server.",
      },
      {
        name: "name",
        type: "string",
        description: "The name is the name of a external table.",
      },
    ],
    description: "",
  },
  "bytebase.v1.FieldMapping": {
    type: "object",
    properties: [
      {
        name: "displayName",
        type: "string",
        description:
          "DisplayName is the field name of display name in 3rd-party idp user info. Optional.",
      },
      {
        name: "groups",
        type: "string",
        description:
          "Groups is the field name of groups in 3rd-party idp user info. Optional.\n Mainly used for OIDC: https://developer.okta.com/docs/guides/customize-tokens-groups-claim/main/",
      },
      {
        name: "identifier",
        type: "string",
        description:
          "Identifier is the field name of the unique identifier in 3rd-party idp user info. Required.",
      },
      {
        name: "phone",
        type: "string",
        description:
          "Phone is the field name of primary phone in 3rd-party idp user info. Optional.",
      },
    ],
    description:
      "FieldMapping saves the field names from user info API of identity provider.\n As we save all raw json string of user info response data into `principal.idp_user_info`,\n we can extract the relevant data based with `FieldMapping`.",
  },
  "bytebase.v1.ForeignKeyMetadata": {
    type: "object",
    properties: [
      {
        name: "columns",
        type: "array<string>",
        description:
          "The columns are the ordered referencing columns of a foreign key.",
      },
      {
        name: "matchType",
        type: "string",
        description:
          "The match_type is the match type of a foreign key.\n The match_type is the PostgreSQL specific field.\n It's empty string for other databases.",
      },
      {
        name: "name",
        type: "string",
        description: "The name is the name of a foreign key.",
      },
      {
        name: "onDelete",
        type: "string",
        description: "The on_delete is the on delete action of a foreign key.",
      },
      {
        name: "onUpdate",
        type: "string",
        description: "The on_update is the on update action of a foreign key.",
      },
      {
        name: "referencedColumns",
        type: "array<string>",
        description:
          "The referenced_columns are the ordered referenced columns of a foreign key.",
      },
      {
        name: "referencedSchema",
        type: "string",
        description:
          "The referenced_schema is the referenced schema name of a foreign key.\n It is an empty string for databases without such concept such as MySQL.",
      },
      {
        name: "referencedTable",
        type: "string",
        description:
          "The referenced_table is the referenced table name of a foreign key.",
      },
    ],
    description: "ForeignKeyMetadata is the metadata for foreign keys.",
  },
  "bytebase.v1.FunctionMetadata": {
    type: "object",
    properties: [
      {
        name: "characterSetClient",
        type: "string",
        description: "MySQL specific metadata.",
      },
      {
        name: "collationConnection",
        type: "string",
      },
      {
        name: "comment",
        type: "string",
      },
      {
        name: "databaseCollation",
        type: "string",
      },
      {
        name: "definition",
        type: "string",
        description: "The definition is the definition of a function.",
      },
      {
        name: "dependencyTables",
        type: "array<bytebase.v1.DependencyTable>",
        description:
          "The dependency_tables is the list of dependency tables of a function.\n For PostgreSQL, it's the list of tables that the function depends on the return type definition.",
      },
      {
        name: "name",
        type: "string",
        description: "The name is the name of a function.",
      },
      {
        name: "signature",
        type: "string",
        description:
          "The signature is the name with the number and type of input arguments the\n function takes.",
      },
      {
        name: "skipDump",
        type: "boolean",
      },
      {
        name: "sqlMode",
        type: "string",
      },
    ],
    description: "FunctionMetadata is the metadata for functions.",
  },
  "bytebase.v1.GenerationMetadata": {
    type: "object",
    properties: [
      {
        name: "expression",
        type: "string",
      },
      {
        name: "type",
        type: "bytebase.v1.GenerationMetadata.Type",
      },
    ],
    description: "",
  },
  "bytebase.v1.GenerationMetadata.Type": {
    type: "enum",
    values: ["TYPE_UNSPECIFIED", "VIRTUAL", "STORED"],
    description: "",
  },
  "bytebase.v1.GetAccessGrantRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the access grant to retrieve.\n Format: projects/{project}/accessGrants/{access_grant}",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetChangelogRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the changelog to retrieve.\n Format: instances/{instance}/databases/{database}/changelogs/{changelog}",
      },
      {
        name: "view",
        type: "bytebase.v1.ChangelogView",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetDatabaseCatalogRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the database catalog to retrieve.\n Format: instances/{instance}/databases/{database}/catalog",
      },
    ],
    description: "Request message for getting a database catalog.",
  },
  "bytebase.v1.GetDatabaseGroupRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the database group to retrieve.\n Format: projects/{project}/databaseGroups/{databaseGroup}",
      },
      {
        name: "view",
        type: "bytebase.v1.DatabaseGroupView",
        description:
          "The view to return. Defaults to DATABASE_GROUP_VIEW_BASIC.",
      },
    ],
    description: "Request message for getting a database group.",
  },
  "bytebase.v1.GetDatabaseMetadataRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'Filter is used to filter databases returned in the list.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Supported filter:\n - schema: the schema name, support "==" operator.\n - table: the table name, support "==" and ".contains()" operator.\n\n For example:\n schema == "schema-a"\n table == "table-a"\n table.contains("table-a")\n schema == "schema-a" && table.contains("sample")\n The filter is used to search for tables containing "sample" in the schema "schemas/schema-a".\n The column masking level will only be returned when a table filter is used.',
      },
      {
        name: "limit",
        type: "integer",
        description:
          "Limit the response size of returned table metadata per schema.\n For example, if the database has 3 schemas, and each schema has 100 tables,\n if limit is 20, then only 20 tables will be returned for each schema, total 60 tables.\n Default 0, means no limit.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the database to retrieve metadata.\n Format: instances/{instance}/databases/{database}/metadata",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetDatabaseRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the database to retrieve.\n Format: instances/{instance}/databases/{database}",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetDatabaseSDLSchemaRequest": {
    type: "object",
    properties: [
      {
        name: "format",
        type: "bytebase.v1.GetDatabaseSDLSchemaRequest.SDLFormat",
        description: "The format of the SDL schema output.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the database to retrieve SDL schema.\n Format: instances/{instance}/databases/{database}/sdlSchema",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetDatabaseSDLSchemaRequest.SDLFormat": {
    type: "enum",
    values: ["SDL_FORMAT_UNSPECIFIED", "SINGLE_FILE", "MULTI_FILE"],
    description: "SDLFormat specifies the output format for SDL schema.",
  },
  "bytebase.v1.GetDatabaseSchemaRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the database to retrieve schema.\n Format: instances/{instance}/databases/{database}/schema",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetGroupRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the group to retrieve.\n Format: groups/{email}",
      },
    ],
    description: "Request message for getting a group.",
  },
  "bytebase.v1.GetIamPolicyRequest": {
    type: "object",
    properties: [
      {
        name: "resource",
        type: "string",
        description:
          "The name of the resource to get the IAM policy.\n Format: projects/{project}\n Format: workspaces/{workspace}",
      },
    ],
    description: "Request message for getting an IAM policy.",
  },
  "bytebase.v1.GetIdentityProviderRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the identity provider to retrieve.\n Format: idps/{idp}",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetInstanceRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the instance to retrieve.\n Format: instances/{instance}",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetIssueRequest": {
    type: "object",
    properties: [
      {
        name: "force",
        type: "boolean",
        description: "If set to true, bypass cache and fetch the latest data.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the issue to retrieve.\n Format: projects/{project}/issues/{issue}",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetPlanCheckRunRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the plan check run to retrieve.\n Format: projects/{project}/plans/{plan}/planCheckRun",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetPlanRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the plan to retrieve.\n Format: projects/{project}/plans/{plan}",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetPolicyRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the policy to retrieve.\n Format: {resource type}/{resource id}/policies/{policy type}",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetProjectRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the project to retrieve.\n Format: projects/{project}",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetReleaseRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description: "Format: projects/{project}/releases/{release}",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetReviewConfigRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the SQL review config to retrieve.\n Format: reviewConfigs/{reviewConfig}",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetRevisionRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the revision.\n Format: instances/{instance}/databases/{database}/revisions/{revision}",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetRoleRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description: "The name of the role to retrieve.\n Format: roles/{role}",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetRolloutRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the rollout to retrieve.\n This is the rollout resource name, which is the plan name plus /rollout suffix.\n Format: projects/{project}/plans/{plan}/rollout",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetSchemaStringRequest": {
    type: "object",
    properties: [
      {
        name: "metadata",
        type: "bytebase.v1.DatabaseMetadata",
        description:
          "If use the metadata to generate the schema string, the type is OBJECT_TYPE_UNSPECIFIED.\n Also the schema and object are empty.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the database.\n Format: instances/{instance}/databases/{database}",
      },
      {
        name: "object",
        type: "string",
        description: "It's empty for DATABASE and SCHEMA.",
      },
      {
        name: "schema",
        type: "string",
        description: "It's empty for DATABASE.",
      },
      {
        name: "type",
        type: "bytebase.v1.GetSchemaStringRequest.ObjectType",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetSchemaStringRequest.ObjectType": {
    type: "enum",
    values: [
      "OBJECT_TYPE_UNSPECIFIED",
      "DATABASE",
      "SCHEMA",
      "TABLE",
      "VIEW",
      "MATERIALIZED_VIEW",
      "FUNCTION",
      "PROCEDURE",
      "SEQUENCE",
    ],
    description: "",
  },
  "bytebase.v1.GetSchemaStringResponse": {
    type: "object",
    properties: [
      {
        name: "schemaString",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetServiceAccountRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the service account to retrieve.\n Format: serviceAccounts/{email}",
      },
    ],
    description: "Request message for getting a service account.",
  },
  "bytebase.v1.GetSettingRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description: "The resource name of the setting.",
      },
    ],
    description: "The request message for getting a setting.",
  },
  "bytebase.v1.GetSettingResponse": {
    type: "object",
    properties: [
      {
        name: "setting",
        type: "bytebase.v1.Setting",
      },
    ],
    description: "The response message for getting a setting.",
  },
  "bytebase.v1.GetSheetRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the sheet to retrieve.\n Format: projects/{project}/sheets/{sheet}",
      },
      {
        name: "raw",
        type: "boolean",
        description:
          "By default, the content of the sheet is cut off, set the `raw` to true to retrieve the full content.",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetTaskRunLogRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          "Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun}",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetTaskRunRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun}",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetTaskRunSessionRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          "Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun}",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetUserRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the user to retrieve.\n Format: users/{email}",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetWorkloadIdentityRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the workload identity to retrieve.\n Format: workloadIdentities/{email}",
      },
    ],
    description: "Request message for getting a workload identity.",
  },
  "bytebase.v1.GetWorksheetRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the worksheet to retrieve.\n Format: projects/{project}/worksheets/{worksheet}",
      },
    ],
    description: "",
  },
  "bytebase.v1.GetWorkspaceActuatorInfoRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description: "The workspace name, format: workspaces/{workspace}.",
      },
    ],
    description:
      "Request message for getting workspace-scoped actuator information.",
  },
  "bytebase.v1.GetWorkspaceRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          'The workspace name, format: workspaces/{workspace}.\n Use "workspaces/-" to get the current/default workspace.',
      },
    ],
    description: "",
  },
  "bytebase.v1.GridLevel": {
    type: "object",
    properties: [
      {
        name: "density",
        type: "string",
        description: "Grid density (LOW, MEDIUM, HIGH)",
      },
      {
        name: "level",
        type: "integer",
        description: "Grid level number (1-4 for SQL Server)",
      },
    ],
    description:
      "GridLevel defines a tessellation grid level with its density.",
  },
  "bytebase.v1.Group": {
    type: "object",
    properties: [
      {
        name: "description",
        type: "string",
        description: "The description of the group.",
      },
      {
        name: "email",
        type: "string",
        description: "The unique email for the group.",
      },
      {
        name: "members",
        type: "array<bytebase.v1.GroupMember>",
        description: "The members of the group.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the group to retrieve.\n Format: groups/{group}, the group should be email or uuid.",
      },
      {
        name: "source",
        type: "string",
        description:
          "The source system where the group originated (e.g., Entra ID for SCIM sync).",
      },
      {
        name: "title",
        type: "string",
        description: "The display title of the group.",
      },
    ],
    description: "A group of users within the workspace.",
  },
  "bytebase.v1.GroupMember": {
    type: "object",
    properties: [
      {
        name: "member",
        type: "string",
        description:
          "Member is the principal who belong to this group.\n\n Format: users/hello@world.com",
      },
      {
        name: "role",
        type: "bytebase.v1.GroupMember.Role",
        description: "The member's role in the group.",
      },
    ],
    description: "A member of a group with a role.",
  },
  "bytebase.v1.GroupMember.Role": {
    type: "enum",
    values: ["ROLE_UNSPECIFIED", "OWNER", "MEMBER"],
    description: "The role of a group member.",
  },
  "bytebase.v1.IamPolicy": {
    type: "object",
    properties: [
      {
        name: "bindings",
        type: "array<bytebase.v1.Binding>",
        description:
          "Collection of binding.\n A binding binds one or more project members to a single project role.",
      },
      {
        name: "etag",
        type: "string",
        description:
          "The current etag of the policy.\n If an etag is provided and does not match the current etag of the policy,\n the call will be blocked and an ABORTED error will be returned.",
      },
    ],
    description: "IAM policy that binds members to roles.",
  },
  "bytebase.v1.IdentityProvider": {
    type: "object",
    properties: [
      {
        name: "config",
        type: "bytebase.v1.IdentityProviderConfig",
        description: "The configuration details for the identity provider.",
      },
      {
        name: "domain",
        type: "string",
        description:
          "The domain for email matching when using this identity provider.",
      },
      {
        name: "name",
        type: "string",
        description: "The name of the identity provider.\n Format: idps/{idp}",
      },
      {
        name: "title",
        type: "string",
        description: "The display title of the identity provider.",
      },
      {
        name: "type",
        type: "bytebase.v1.IdentityProviderType",
        description: "The type of identity provider protocol.",
      },
    ],
    description: "",
  },
  "bytebase.v1.IdentityProviderType": {
    type: "enum",
    values: ["IDENTITY_PROVIDER_TYPE_UNSPECIFIED", "OAUTH2", "OIDC", "LDAP"],
    description: "",
  },
  "bytebase.v1.IndexMetadata": {
    type: "object",
    properties: [
      {
        name: "comment",
        type: "string",
        description: "The comment is the comment of an index.",
      },
      {
        name: "definition",
        type: "string",
        description: "The definition of an index.",
      },
      {
        name: "descending",
        type: "array<boolean>",
        description: "The descending is the ordered descending of an index.",
      },
      {
        name: "expressions",
        type: "array<string>",
        description:
          "The expressions are the ordered columns or expressions of an index.\n This could refer to a column or an expression.",
      },
      {
        name: "granularity",
        type: "integer",
        description:
          "The number of granules in the block. It's a ClickHouse specific field.",
      },
      {
        name: "isConstraint",
        type: "boolean",
        description:
          "It's a PostgreSQL specific field.\n The unique constraint and unique index are not the same thing in PostgreSQL.",
      },
      {
        name: "keyLength",
        type: "array<integer>",
        description:
          "The key_lengths are the ordered key lengths of an index.\n If the key length is not specified, it's -1.",
      },
      {
        name: "name",
        type: "string",
        description: "The name is the name of an index.",
      },
      {
        name: "opclassDefaults",
        type: "array<boolean>",
        description:
          "True if the operator class is the default. (PostgreSQL specific).",
      },
      {
        name: "opclassNames",
        type: "array<string>",
        description:
          "https://www.postgresql.org/docs/current/catalog-pg-opclass.html\n Name of the operator class for each column. (PostgreSQL specific).",
      },
      {
        name: "parentIndexName",
        type: "string",
        description: "The index name of the parent index.",
      },
      {
        name: "parentIndexSchema",
        type: "string",
        description: "The schema name of the parent index.",
      },
      {
        name: "primary",
        type: "boolean",
        description: "The primary is whether the index is a primary key index.",
      },
      {
        name: "spatialConfig",
        type: "bytebase.v1.SpatialIndexConfig",
        description:
          "Spatial index configuration for spatial databases like SQL Server, PostgreSQL with PostGIS, etc.",
      },
      {
        name: "type",
        type: "string",
        description: "The type is the type of an index.",
      },
      {
        name: "unique",
        type: "boolean",
        description: "The unique is whether the index is unique.",
      },
      {
        name: "visible",
        type: "boolean",
        description: "The visible is whether the index is visible.",
      },
    ],
    description: "IndexMetadata is the metadata for indexes.",
  },
  "bytebase.v1.Instance": {
    type: "object",
    properties: [
      {
        name: "activation",
        type: "boolean",
        description: "Whether the instance is activated for use.",
      },
      {
        name: "dataSources",
        type: "array<bytebase.v1.DataSource>",
        description:
          "Data source configurations for connecting to the instance.",
      },
      {
        name: "engine",
        type: "bytebase.v1.Engine",
        description: "The database engine type.",
      },
      {
        name: "engineVersion",
        type: "string",
        description: "The version of the database engine.",
      },
      {
        name: "environment",
        type: "string",
        description:
          "The environment resource.\n Format: environments/prod where prod is the environment resource ID.",
      },
      {
        name: "externalLink",
        type: "string",
        description: "External URL to the database instance console.",
      },
      {
        name: "labels",
        type: "object",
        description:
          'Labels are key-value pairs that can be attached to the instance.\n For example, { "org_group": "infrastructure", "environment": "production" }',
      },
      {
        name: "lastSyncTime",
        type: "google.protobuf.Timestamp",
        description: "The last time the instance was synced.",
      },
      {
        name: "name",
        type: "string",
        description: "The name of the instance.\n Format: instances/{instance}",
      },
      {
        name: "roles",
        type: "array<bytebase.v1.InstanceRole>",
        description: "Database roles available in this instance.",
      },
      {
        name: "state",
        type: "bytebase.v1.State",
        description: "The lifecycle state of the instance.",
      },
      {
        name: "syncDatabases",
        type: "array<string>",
        description:
          "Enable sync for following databases.\n Default empty, means sync all schemas & databases.",
      },
      {
        name: "syncInterval",
        type: "google.protobuf.Duration",
        description: "How often the instance is synced.",
      },
      {
        name: "title",
        type: "string",
        description: "The display title of the instance.",
      },
    ],
    description: "",
  },
  "bytebase.v1.Instance.LabelsEntry": {
    type: "object",
    properties: [
      {
        name: "key",
        type: "string",
      },
      {
        name: "value",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.InstanceResource": {
    type: "object",
    properties: [
      {
        name: "activation",
        type: "boolean",
        description: "Whether the instance is activated.",
      },
      {
        name: "dataSources",
        type: "array<bytebase.v1.DataSource>",
        description: "Data source configurations for the instance.",
      },
      {
        name: "engine",
        type: "bytebase.v1.Engine",
        description: "The database engine type.",
      },
      {
        name: "engineVersion",
        type: "string",
        description: "The version of the database engine.",
      },
      {
        name: "environment",
        type: "string",
        description:
          "The environment resource.\n Format: environments/prod where prod is the environment resource ID.",
      },
      {
        name: "name",
        type: "string",
        description: "The name of the instance.\n Format: instances/{instance}",
      },
      {
        name: "title",
        type: "string",
        description: "The display title of the instance.",
      },
    ],
    description: "",
  },
  "bytebase.v1.InstanceRole": {
    type: "object",
    properties: [
      {
        name: "attribute",
        type: "string",
        description:
          'The role attribute.\n For PostgreSQL, it contains super_user, no_inherit, create_role, create_db, can_login, replication, and bypass_rls. Docs: https://www.postgresql.org/docs/current/role-attributes.html\n For MySQL, it\'s the global privileges as GRANT statements, which means it only contains "GRANT ... ON *.* TO ...". Docs: https://dev.mysql.com/doc/refman/8.0/en/grant.html',
      },
      {
        name: "connectionLimit",
        type: "integer",
        description: "The connection count limit for this role.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the role.\n Format: instances/{instance}/roles/{role}\n The role name is the unique name for the role.",
      },
      {
        name: "password",
        type: "string",
        description: "The role password.",
      },
      {
        name: "roleName",
        type: "string",
        description: "The role name. It's unique within the instance.",
      },
      {
        name: "validUntil",
        type: "string",
        description: "The expiration for the role's password.",
      },
    ],
    description: "InstanceRole is the API message for instance role.",
  },
  "bytebase.v1.Issue": {
    type: "object",
    properties: [
      {
        name: "accessGrant",
        type: "string",
        description:
          "The access grant associated with this issue.\n Format: projects/{project}/accessGrants/{access_grant}",
      },
      {
        name: "approvalStatus",
        type: "bytebase.v1.Issue.ApprovalStatus",
      },
      {
        name: "approvalTemplate",
        type: "bytebase.v1.ApprovalTemplate",
        description: "The approval template for the issue.",
      },
      {
        name: "approvers",
        type: "array<bytebase.v1.Issue.Approver>",
      },
      {
        name: "createTime",
        type: "google.protobuf.Timestamp",
      },
      {
        name: "creator",
        type: "string",
        description: "Format: users/hello@world.com",
      },
      {
        name: "description",
        type: "string",
        description: "The description of the issue.",
      },
      {
        name: "labels",
        type: "array<string>",
        description:
          "Labels attached to the issue for categorization and filtering.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the issue.\n Format: projects/{project}/issues/{issue}",
      },
      {
        name: "plan",
        type: "string",
        description:
          "The plan associated with the issue.\n Can be empty.\n Format: projects/{project}/plans/{plan}",
      },
      {
        name: "riskLevel",
        type: "bytebase.v1.RiskLevel",
        description: "The risk level of the issue.",
      },
      {
        name: "roleGrant",
        type: "bytebase.v1.RoleGrant",
        description: "Used if the issue type is ROLE_GRANT.",
      },
      {
        name: "status",
        type: "bytebase.v1.IssueStatus",
        description: "The status of the issue.",
      },
      {
        name: "title",
        type: "string",
        description: "The title of the issue.",
      },
      {
        name: "type",
        type: "bytebase.v1.Issue.Type",
      },
      {
        name: "updateTime",
        type: "google.protobuf.Timestamp",
      },
    ],
    description: "",
  },
  "bytebase.v1.Issue.ApprovalStatus": {
    type: "enum",
    values: [
      "APPROVAL_STATUS_UNSPECIFIED",
      "CHECKING",
      "PENDING",
      "APPROVED",
      "REJECTED",
      "SKIPPED",
    ],
    description: "The overall approval status for the issue.",
  },
  "bytebase.v1.Issue.Approver": {
    type: "object",
    properties: [
      {
        name: "principal",
        type: "string",
        description: "Format: users/hello@world.com",
      },
      {
        name: "status",
        type: "bytebase.v1.Issue.Approver.Status",
        description: "The new status.",
      },
    ],
    description: "Approvers and their approval status for the issue.",
  },
  "bytebase.v1.Issue.Approver.Status": {
    type: "enum",
    values: ["STATUS_UNSPECIFIED", "PENDING", "APPROVED", "REJECTED"],
    description: "The approval status of an approver.",
  },
  "bytebase.v1.Issue.Type": {
    type: "enum",
    values: [
      "TYPE_UNSPECIFIED",
      "DATABASE_CHANGE",
      "ROLE_GRANT",
      "DATABASE_EXPORT",
      "ACCESS_GRANT",
    ],
    description: "The type of issue.",
  },
  "bytebase.v1.IssueComment": {
    type: "object",
    properties: [
      {
        name: "comment",
        type: "string",
        description: "The text content of the comment.",
      },
      {
        name: "createTime",
        type: "google.protobuf.Timestamp",
      },
      {
        name: "creator",
        type: "string",
        description: "Format: users/{email}",
      },
      {
        name: "name",
        type: "string",
        description:
          "Format: projects/{project}/issues/{issue}/issueComments/{issueComment-uid}",
      },
      {
        name: "payload",
        type: "string",
        description: "TODO: use struct message instead.",
      },
      {
        name: "updateTime",
        type: "google.protobuf.Timestamp",
      },
    ],
    description: "A comment on an issue.",
  },
  "bytebase.v1.IssueComment.Approval": {
    type: "object",
    properties: [
      {
        name: "status",
        type: "bytebase.v1.IssueComment.Approval.Status",
        description: "The approval status.",
      },
    ],
    description: "Approval event information.",
  },
  "bytebase.v1.IssueComment.Approval.Status": {
    type: "enum",
    values: ["STATUS_UNSPECIFIED", "PENDING", "APPROVED", "REJECTED"],
    description: "Approval status values.",
  },
  "bytebase.v1.IssueComment.IssueUpdate": {
    type: "object",
    properties: [
      {
        name: "fromDescription",
        type: "string",
      },
      {
        name: "fromLabels",
        type: "array<string>",
      },
      {
        name: "fromStatus",
        type: "object",
      },
      {
        name: "fromTitle",
        type: "string",
      },
      {
        name: "toDescription",
        type: "string",
      },
      {
        name: "toLabels",
        type: "array<string>",
      },
      {
        name: "toStatus",
        type: "object",
      },
      {
        name: "toTitle",
        type: "string",
      },
    ],
    description: "Issue update event information.",
  },
  "bytebase.v1.IssueComment.PlanSpecUpdate": {
    type: "object",
    properties: [
      {
        name: "fromSheet",
        type: "string",
        description:
          "The previous sheet.\n Format: projects/{project}/sheets/{sheet}",
      },
      {
        name: "spec",
        type: "string",
        description:
          "The spec that was updated.\n Format: projects/{project}/plans/{plan}/specs/{spec}",
      },
      {
        name: "toSheet",
        type: "string",
        description:
          "The new sheet.\n Format: projects/{project}/sheets/{sheet}",
      },
    ],
    description:
      "Plan spec update event information (tracks sheet changes to plan specs).",
  },
  "bytebase.v1.IssueStatus": {
    type: "enum",
    values: ["ISSUE_STATUS_UNSPECIFIED", "OPEN", "DONE", "CANCELED"],
    description: "The status of an issue.",
  },
  "bytebase.v1.KerberosConfig": {
    type: "object",
    properties: [
      {
        name: "instance",
        type: "string",
        description: "The instance component of the Kerberos principal.",
      },
      {
        name: "kdcHost",
        type: "string",
        description: "The hostname of the Key Distribution Center (KDC).",
      },
      {
        name: "kdcPort",
        type: "string",
        description: "The port of the Key Distribution Center (KDC).",
      },
      {
        name: "kdcTransportProtocol",
        type: "string",
        description:
          "The transport protocol for KDC communication (tcp or udp).",
      },
      {
        name: "keytab",
        type: "string",
        description: "The keytab file contents for authentication.",
      },
      {
        name: "primary",
        type: "string",
        description: "The primary component of the Kerberos principal.",
      },
      {
        name: "realm",
        type: "string",
        description: "The Kerberos realm.",
      },
    ],
    description: "",
  },
  "bytebase.v1.LDAPIdentityProviderConfig": {
    type: "object",
    properties: [
      {
        name: "baseDn",
        type: "string",
        description:
          'BaseDN is the base DN to search for users, e.g., "ou=users,dc=example,dc=com".',
      },
      {
        name: "bindDn",
        type: "string",
        description:
          "BindDN is the DN of the user to bind as a service account to perform\n search requests.",
      },
      {
        name: "bindPassword",
        type: "string",
        description:
          "BindPassword is the password of the user to bind as a service account.",
      },
      {
        name: "fieldMapping",
        type: "bytebase.v1.FieldMapping",
        description:
          "FieldMapping is the mapping of the user attributes returned by the LDAP\n server.",
      },
      {
        name: "host",
        type: "string",
        description:
          'Host is the hostname or IP address of the LDAP server, e.g.,\n "ldap.example.com".',
      },
      {
        name: "port",
        type: "integer",
        description:
          "Port is the port number of the LDAP server, e.g., 389. When not set, the\n default port of the corresponding security protocol will be used, i.e. 389\n for StartTLS and 636 for LDAPS.",
      },
      {
        name: "securityProtocol",
        type: "bytebase.v1.LDAPIdentityProviderConfig.SecurityProtocol",
        description:
          "SecurityProtocol is the security protocol to be used for establishing\n connections with the LDAP server.",
      },
      {
        name: "skipTlsVerify",
        type: "boolean",
        description:
          "SkipTLSVerify controls whether to skip TLS certificate verification.",
      },
      {
        name: "userFilter",
        type: "string",
        description:
          'UserFilter is the filter to search for users, e.g., "(uid=%s)".',
      },
    ],
    description:
      "LDAPIdentityProviderConfig is the structure for LDAP identity provider config.",
  },
  "bytebase.v1.LDAPIdentityProviderConfig.SecurityProtocol": {
    type: "enum",
    values: ["SECURITY_PROTOCOL_UNSPECIFIED", "START_TLS", "LDAPS"],
    description: "",
  },
  "bytebase.v1.Label": {
    type: "object",
    properties: [
      {
        name: "color",
        type: "string",
        description: "The color code for the label (e.g., hex color).",
      },
      {
        name: "group",
        type: "string",
        description: "The group this label belongs to.",
      },
      {
        name: "value",
        type: "string",
        description: "The label value/name.",
      },
    ],
    description: "A label for categorizing and organizing issues.",
  },
  "bytebase.v1.ListAccessGrantsRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'Filter expression using AIP-160 syntax.\n Supported fields:\n - name: the fullname in "projects/{project}/accessGrants/{access_grant}" format, support "==" operator.\n - creator: the creator name in "users/{email}" format, support "==" operator.\n - status: the access status, support "==" and "in" operator.\n - issue: the access issue fullname, support "==" operator.\n - expire_time: the access expire time in "2006-01-02T15:04:05Z07:00" format, support ">=", ">", "<=" and "<" operator.\n - create_time: the access creation time in "2006-01-02T15:04:05Z07:00" format, support ">=", ">", "<=" and "<" operator.\n - query: the access query, support "==" and ".contains(xx)" operator\n - target: the target database fullname, support "==" operator.\n\n Examples:\n - creator == "users/dev@example.com"\n - status == "ACTIVE"\n - status in ["ACTIVE", "PENDING"]\n - creator == "users/dev@example.com" && status == "ACTIVE"\n - issue == "projects/x/issues/123"\n - status == "ACTIVE" && expire_time > "2024-02-01T00:00:00Z"\n - target == "instances/sample/databases/employee"',
      },
      {
        name: "orderBy",
        type: "string",
        description:
          'The order by of access grants.\n Support creator, expire_time, create_time. The default sorting order is ascending.\n For example:\n - order_by = "creator"\n - order_by = "expire_time desc"\n - order_by = "expire_time asc, create_time desc"',
      },
      {
        name: "pageSize",
        type: "integer",
        description: "The maximum number of access grants to return.",
      },
      {
        name: "pageToken",
        type: "string",
        description: "A page token from a previous ListAccessGrants call.",
      },
      {
        name: "parent",
        type: "string",
        description:
          "The parent project of the access grants.\n Format: projects/{project}",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListAccessGrantsResponse": {
    type: "object",
    properties: [
      {
        name: "accessGrants",
        type: "array<bytebase.v1.AccessGrant>",
        description: "The access grants from the specified request.",
      },
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListChangelogsRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'Filter is used to filter changelogs returned in the list.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Supported filter:\n - status: the changelog status, support "==" operation. check Changelog.Status for available values.\n - create_time: the changelog create time in "2006-01-02T15:04:05Z07:00" format, support ">=" or "<=" operator.\n\n Example:\n status == "DONE"\n status == "FAILED" && type == "SDL"\n create_time >= "2024-01-01T00:00:00Z" && create_time <= "2024-01-02T00:00:00Z"',
      },
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of changelogs to return. The service may return fewer\n than this value. If unspecified, at most 10 changelogs will be returned.\n The maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from the previous call.\n Provide this to retrieve the subsequent page.\n\n When paginating, all other parameters provided must match\n the call that provided the page token.",
      },
      {
        name: "parent",
        type: "string",
        description:
          "The parent of the changelogs.\n Format: instances/{instance}/databases/{database}",
      },
      {
        name: "view",
        type: "bytebase.v1.ChangelogView",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListChangelogsResponse": {
    type: "object",
    properties: [
      {
        name: "changelogs",
        type: "array<bytebase.v1.Changelog>",
        description: "The list of changelogs.",
      },
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListDatabaseGroupsRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          "The parent resource whose database groups are to be listed.\n Format: projects/{project}",
      },
      {
        name: "view",
        type: "bytebase.v1.DatabaseGroupView",
        description:
          "The view to return. Defaults to DATABASE_GROUP_VIEW_BASIC.",
      },
    ],
    description: "Request message for listing database groups.",
  },
  "bytebase.v1.ListDatabaseGroupsResponse": {
    type: "object",
    properties: [
      {
        name: "databaseGroups",
        type: "array<bytebase.v1.DatabaseGroup>",
        description: "The database groups from the specified request.",
      },
    ],
    description: "Response message for listing database groups.",
  },
  "bytebase.v1.ListDatabasesRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'Filter is used to filter databases returned in the list.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Supported filter:\n - environment: the environment full name in "environments/{id}" format, support "==" operator.\n - name: the database name, support ".contains()" operator.\n - project: the project full name in "projects/{id}" format, support "==" operator.\n - instance: the instance full name in "instances/{id}" format, support "==" operator.\n - engine: the database engine, check Engine enum for values. Support "==", "in [xx]", "!(in [xx])" operator.\n - exclude_unassigned: should be "true" or "false", will not show unassigned databases if it\'s true, support "==" operator.\n - table: filter by the database table, support "==" and ".contains()" operator.\n - labels.{key}: the database label, support "==" and "in" operators.\n\n For example:\n environment == "environments/{environment resource id}"\n environment == "" (find databases which environment is not set)\n project == "projects/{project resource id}"\n instance == "instances/{instance resource id}"\n name.contains("database name")\n engine == "MYSQL"\n engine in ["MYSQL", "POSTGRES"]\n !(engine in ["MYSQL", "POSTGRES"])\n exclude_unassigned == true\n table == "sample"\n table.contains("sam")\n labels.environment == "production"\n labels.region == "asia"\n labels.region in ["asia", "europe"]\n\n You can combine filter conditions like:\n environment == "environments/prod" && name.contains("employee")',
      },
      {
        name: "orderBy",
        type: "string",
        description:
          'The order by of databases.\n Support name, project, instance. The default sorting order is ascending.\n For example:\n - order_by = "name" - order by name ascending\n - order_by = "name desc"\n - order_by = "name desc, project asc"',
      },
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of databases to return. The service may return fewer\n than this value.\n If unspecified, at most 10 databases will be returned.\n The maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `ListDatabases` call.\n Provide this to retrieve the subsequent page.\n\n When paginating, all other parameters provided to `ListDatabases` must\n match the call that provided the page token.",
      },
      {
        name: "parent",
        type: "string",
        description:
          '- projects/{project}: list databases in a project, require "bb.projects.get" permission.\n - workspaces/{id}: list databases in the workspace, require "bb.databases.list" permission.\n - instances/{instances}: list databases in a instance, require "bb.instances.get" permission',
      },
      {
        name: "showDeleted",
        type: "boolean",
        description: "Show deleted database if specified.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListDatabasesResponse": {
    type: "object",
    properties: [
      {
        name: "databases",
        type: "array<bytebase.v1.Database>",
        description: "The databases from the specified request.",
      },
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListGroupsRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'Filter is used to filter groups returned in the list.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Supported filter:\n - title: the group title, support "==" and ".contains()" operator.\n - email: the group email, support "==" and ".contains()" operator.\n - project: the project full name in "projects/{id}" format, support "==" operator.\n\n For example:\n title == "dba"\n email == "dba@bytebase.com"\n title.contains("dba")\n email.contains("dba")\n project == "projects/sample-project"\n You can combine filter conditions like:\n title.contains("dba") || email.contains("dba")',
      },
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of groups to return. The service may return fewer than\n this value.\n If unspecified, at most 10 groups will be returned.\n The maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `ListGroups` call.\n Provide this to retrieve the subsequent page.\n\n When paginating, all other parameters provided to `ListGroups` must match\n the call that provided the page token.",
      },
    ],
    description: "Request message for listing groups.",
  },
  "bytebase.v1.ListGroupsResponse": {
    type: "object",
    properties: [
      {
        name: "groups",
        type: "array<bytebase.v1.Group>",
        description: "The groups from the specified request.",
      },
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
    ],
    description: "Response message for listing groups.",
  },
  "bytebase.v1.ListIdentityProvidersResponse": {
    type: "object",
    properties: [
      {
        name: "identityProviders",
        type: "array<bytebase.v1.IdentityProvider>",
        description: "The identity providers from the specified request.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListInstanceDatabaseRequest": {
    type: "object",
    properties: [
      {
        name: "instance",
        type: "object",
        description:
          "The target instance. We need to set this field if the target instance is not created yet.",
      },
      {
        name: "name",
        type: "string",
        description: "The name of the instance.\n Format: instances/{instance}",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListInstanceDatabaseResponse": {
    type: "object",
    properties: [
      {
        name: "databases",
        type: "array<string>",
        description: "All database name list in the instance.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListInstanceRolesRequest": {
    type: "object",
    properties: [
      {
        name: "pageSize",
        type: "integer",
        description:
          "Not used.\n The maximum number of roles to return. The service may return fewer than\n this value.\n If unspecified, at most 10 roles will be returned.\n The maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "Not used.\n A page token, received from a previous `ListInstanceRoles` call.\n Provide this to retrieve the subsequent page.\n\n When paginating, all other parameters provided to `ListInstanceRoles` must match\n the call that provided the page token.",
      },
      {
        name: "parent",
        type: "string",
        description:
          "The parent, which owns this collection of roles.\n Format: instances/{instance}",
      },
      {
        name: "refresh",
        type: "boolean",
        description: "Refresh will refresh and return the latest data.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListInstanceRolesResponse": {
    type: "object",
    properties: [
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
      {
        name: "roles",
        type: "array<bytebase.v1.InstanceRole>",
        description: "The roles from the specified request.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListInstancesRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'Filter the instance.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Supported filters:\n - name: the instance name, support "==" and ".contains()" operator.\n - resource_id: the instance id, support "==" and ".contains()" operator.\n - environment: the environment full name in "environments/{id}" format, support "==" operator.\n - state: the instance state, check State enum for values, support "==" operator.\n - engine: the instance engine, check Engine enum for values. Support "==", "in [xx]", "!(in [xx])" operator.\n - host: the instance host, support "==" and ".contains()" operator.\n - port: the instance port, support "==" and ".contains()" operator.\n - project: the project full name in "projects/{id}" format, support "==" operator.\n - labels.{key}: the instance label, support "==" and "in" operators.\n\n For example:\n name == "sample instance"\n name.contains("sample")\n resource_id == "sample-instance"\n resource_id.contains("sample")\n state == "DELETED"\n environment == "environments/test"\n environment == "" (find instances which environment is not set)\n engine == "MYSQL"\n engine in ["MYSQL", "POSTGRES"]\n !(engine in ["MYSQL", "POSTGRES"])\n host == "127.0.0.1"\n host.contains("127.0")\n port == "54321"\n port.contains("543")\n labels.org_group == "infrastructure"\n labels.environment in ["prod", "production"]\n project == "projects/sample-project"\n You can combine filter conditions like:\n name.contains("sample") && environment == "environments/test"\n host == "127.0.0.1" && port == "54321"',
      },
      {
        name: "orderBy",
        type: "string",
        description:
          'The order by of instances.\n Support title, environment. The default sorting order is ascending.\n For example:\n - order_by = "title"\n - order_by = "title desc"\n - order_by = "title desc, environment asc"',
      },
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of instances to return. The service may return fewer than\n this value.\n If unspecified, at most 10 instances will be returned.\n The maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `ListInstances` call.\n Provide this to retrieve the subsequent page.\n\n When paginating, all other parameters provided to `ListInstances` must match\n the call that provided the page token.",
      },
      {
        name: "showDeleted",
        type: "boolean",
        description: "Show deleted instances if specified.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListInstancesResponse": {
    type: "object",
    properties: [
      {
        name: "instances",
        type: "array<bytebase.v1.Instance>",
        description: "The instances from the specified request.",
      },
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListIssueCommentsRequest": {
    type: "object",
    properties: [
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of issue comments to return. The service may return fewer than\n this value.\n If unspecified, at most 10 issue comments will be returned.\n The maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `ListIssueComments` call.\n Provide this to retrieve the subsequent page.\n\n When paginating, all other parameters provided to `ListIssueComments` must match\n the call that provided the page token.",
      },
      {
        name: "parent",
        type: "string",
        description: "Format: projects/{projects}/issues/{issue}",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListIssueCommentsResponse": {
    type: "object",
    properties: [
      {
        name: "issueComments",
        type: "array<bytebase.v1.IssueComment>",
        description: "The issue comments from the specified request.",
      },
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListIssuesRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'Filter is used to filter issues returned in the list.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Supported filters:\n - creator: issue creator full name in "users/{email or id}" format, support "==" operator.\n - status: the issue status, support "==" and "in" operator, check the IssueStatus enum for the values.\n - create_time: issue create time in "2006-01-02T15:04:05Z07:00" format, support ">=" or "<=" operator.\n - type: the issue type, support "==" and "in" operator, check the Type enum in the Issue message for the values.\n - labels: the issue labels, support "==" and "in" operator.\n - risk_level: the issue risk level, support "in" operator, check the RiskLevel enum for the values.\n - approval_status: issue approval status, support "==" operator.\n - current_approver: the issue approver, should in "users/{email} format", support "==" operator.\n\n For example:\n creator == "users/ed@bytebase.com" && status in ["OPEN", "DONE"]\n status == "CANCELED" && type == "DATABASE_CHANGE"\n labels in ["label1", "label2"]\n risk_level in ["HIGH", "MODERATE"]\n create_time >= "2025-01-02T15:04:05Z07:00"',
      },
      {
        name: "orderBy",
        type: "string",
        description:
          'The order by of issues.\n Support:\n - create_time\n - update_time\n The default sorting order is ascending.\n For example:\n - order_by = "create_time desc"\n - order_by = "update_time asc"',
      },
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of issues to return. The service may return fewer than\n this value.\n If unspecified, at most 10 issues will be returned.\n The maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `ListIssues` call.\n Provide this to retrieve the subsequent page.\n\n When paginating, all other parameters provided to `ListIssues` must match\n the call that provided the page token.",
      },
      {
        name: "parent",
        type: "string",
        description:
          "The parent, which owns this collection of issues.\n Format: projects/{project}",
      },
      {
        name: "query",
        type: "string",
        description: "Query is the query statement.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListIssuesResponse": {
    type: "object",
    properties: [
      {
        name: "issues",
        type: "array<bytebase.v1.Issue>",
        description: "The issues from the specified request.",
      },
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListPlansRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'Filter is used to filter plans returned in the list.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Supported filters:\n - creator: the plan creator full name in "users/{email or id}" format, support "==" operator.\n - create_time: the plan create time in "2006-01-02T15:04:05Z07:00" format, support ">=" or "<=" operator.\n - has_rollout: whether the plan has rollout, support "==" operator, the value should be "true" or "false".\n - has_issue: the plan has issue or not, support "==" operator, the value should be "true" or "false".\n - title: the plan title, support "==" operator for exact match and ".contains()" operator for case-insensitive substring match.\n - spec_type: the plan spec config type, support "==" operator, the value should be "create_database_config", "change_database_config", or "export_data_config".\n - state: the plan state, support "==" operator, the value should be "ACTIVE" or "DELETED".\n\n For example:\n creator == "users/ed@bytebase.com" && create_time >= "2025-01-02T15:04:05Z07:00"\n has_rollout == false && has_issue == true\n title == "My Plan"\n title.contains("database migration")\n spec_type == "change_database_config"\n state == "ACTIVE"',
      },
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of plans to return. The service may return fewer than\n this value.\n If unspecified, at most 10 plans will be returned.\n The maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `ListPlans` call.\n Provide this to retrieve the subsequent page.\n\n When paginating, all other parameters provided to `ListPlans` must match\n the call that provided the page token.",
      },
      {
        name: "parent",
        type: "string",
        description:
          'The parent, which owns this collection of plans.\n Format: projects/{project}\n Use "projects/-" to list all plans from all projects.',
      },
    ],
    description: "",
  },
  "bytebase.v1.ListPlansResponse": {
    type: "object",
    properties: [
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
      {
        name: "plans",
        type: "array<bytebase.v1.Plan>",
        description: "The plans from the specified request.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListPoliciesRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          "The parent, which owns this collection of policies.\n Format: {resource type}/{resource id}",
      },
      {
        name: "policyType",
        type: "object",
        description: "Filter by specific policy type.",
      },
      {
        name: "showDeleted",
        type: "boolean",
        description: "Show deleted policies if specified.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListPoliciesResponse": {
    type: "object",
    properties: [
      {
        name: "policies",
        type: "array<bytebase.v1.Policy>",
        description: "The policies from the specified request.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListProjectsRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'Filter the project.\n Check filter for SearchProjectsRequest for details.\n Supports filtering by name, resource_id, state, and labels (e.g., labels.environment == "production").',
      },
      {
        name: "orderBy",
        type: "string",
        description:
          'The order by of projects.\n Support title. The default sorting order is ascending.\n For example:\n - order_by = "title"\n - order_by = "title desc"\n - order_by = "title asc"',
      },
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of projects to return. The service may return fewer than\n this value.\n If unspecified, at most 10 projects will be returned.\n The maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `ListProjects` call.\n Provide this to retrieve the subsequent page.\n\n When paginating, all other parameters provided to `ListProjects` must match\n the call that provided the page token.",
      },
      {
        name: "showDeleted",
        type: "boolean",
        description: "Show deleted projects if specified.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListProjectsResponse": {
    type: "object",
    properties: [
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
      {
        name: "projects",
        type: "array<bytebase.v1.Project>",
        description: "The projects from the specified request.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListPurchasePlansResponse": {
    type: "object",
    properties: [
      {
        name: "plans",
        type: "array<bytebase.v1.PurchasePlan>",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListReleaseCategoriesRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description: "Format: projects/{project}",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListReleaseCategoriesResponse": {
    type: "object",
    properties: [
      {
        name: "categories",
        type: "array<string>",
        description: "The unique category values in the project.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListReleasesRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'Filter is used to filter releases returned in the list.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Supported filters:\n - category: release category, support "==" operator.\n\n For example:\n category == "webapp"',
      },
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of releases to return. The service may return fewer than this value.\n If unspecified, at most 10 releases will be returned.\n The maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `ListReleases` call.\n Provide this to retrieve the subsequent page.\n\n When paginating, all other parameters provided to `ListReleases` must match\n the call that provided the page token.",
      },
      {
        name: "parent",
        type: "string",
        description: "Format: projects/{project}",
      },
      {
        name: "showDeleted",
        type: "boolean",
        description: "Show deleted releases if specified.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListReleasesResponse": {
    type: "object",
    properties: [
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
      {
        name: "releases",
        type: "array<bytebase.v1.Release>",
        description: "The releases from the specified request.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListReviewConfigsResponse": {
    type: "object",
    properties: [
      {
        name: "reviewConfigs",
        type: "array<bytebase.v1.ReviewConfig>",
        description: "The SQL review configs from the specified request.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListRevisionsRequest": {
    type: "object",
    properties: [
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of revisions to return. The service may return fewer\n than this value. If unspecified, at most 10 revisions will be returned. The\n maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `ListRevisions` call.\n Provide this to retrieve the subsequent page.\n\n When paginating, all other parameters provided to `ListRevisions` must\n match the call that provided the page token.",
      },
      {
        name: "parent",
        type: "string",
        description:
          "The parent of the revisions.\n Format: instances/{instance}/databases/{database}",
      },
      {
        name: "showDeleted",
        type: "boolean",
        description: "Whether to include deleted revisions in the results.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListRevisionsResponse": {
    type: "object",
    properties: [
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
      {
        name: "revisions",
        type: "array<bytebase.v1.Revision>",
        description: "The revisions from the specified request.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListRolesResponse": {
    type: "object",
    properties: [
      {
        name: "roles",
        type: "array<bytebase.v1.Role>",
        description: "The roles from the specified request.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListRolloutsRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'Filter is used to filter rollouts returned in the list.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Supported filters:\n - update_time: rollout update time in "2006-01-02T15:04:05Z07:00" format, support ">=" or "<=" operator.\n - task_type: the task type, support "in" operator, check the Task.Type enum for the values.\n\n For example:\n update_time >= "2025-01-02T15:04:05Z07:00"\n task_type in ["DATABASE_MIGRATE", "DATABASE_EXPORT"]',
      },
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of rollouts to return. The service may return fewer than\n this value.\n If unspecified, at most 10 rollouts will be returned.\n The maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `ListRollouts` call.\n Provide this to retrieve the subsequent page.\n\n When paginating, all other parameters provided to `ListRollouts` must match\n the call that provided the page token.",
      },
      {
        name: "parent",
        type: "string",
        description:
          'The parent, which owns this collection of rollouts.\n Format: projects/{project}\n Use "projects/-" to list all rollouts from all projects.',
      },
    ],
    description: "",
  },
  "bytebase.v1.ListRolloutsResponse": {
    type: "object",
    properties: [
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
      {
        name: "rollouts",
        type: "array<bytebase.v1.Rollout>",
        description: "The rollouts from the specified request.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListServiceAccountsRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'Filter is used to filter service accounts returned in the list.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Supported filter:\n - name: the service account name, support "==" and ".contains()" operator.\n - email: the service account email, support "==" and ".contains()" operator.\n - state: check State enum for values, support "==" operator.\n\n For example:\n name == "ed"\n name.contains("ed")\n state == "DELETED"\n email == "ed@service.bytebase.com"\n email.contains("ed")',
      },
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of service accounts to return. The service may return fewer than\n this value.\n If unspecified, at most 10 service accounts will be returned.\n The maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `ListServiceAccounts` call.\n Provide this to retrieve the subsequent page.\n\n When paginating, all other parameters provided to `ListServiceAccounts` must match\n the call that provided the page token.",
      },
      {
        name: "parent",
        type: "string",
        description:
          "The parent resource.\n Format: projects/{project} for project-level, workspaces/{id} for workspace-level.",
      },
      {
        name: "showDeleted",
        type: "boolean",
        description: "Show deleted service accounts if specified.",
      },
    ],
    description: "Request message for listing service accounts.",
  },
  "bytebase.v1.ListServiceAccountsResponse": {
    type: "object",
    properties: [
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
      {
        name: "serviceAccounts",
        type: "array<bytebase.v1.ServiceAccount>",
        description: "The service accounts from the specified request.",
      },
    ],
    description: "Response message for listing service accounts.",
  },
  "bytebase.v1.ListSettingsResponse": {
    type: "object",
    properties: [
      {
        name: "settings",
        type: "array<bytebase.v1.Setting>",
        description: "The settings from the specified request.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListTaskRunsRequest": {
    type: "object",
    properties: [
      {
        name: "parent",
        type: "string",
        description:
          'The parent, which owns this collection of taskRuns.\n Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}\n Use "projects/{project}/plans/{plan}/rollout/stages/-/tasks/-" to list all taskRuns from a rollout.',
      },
    ],
    description: "",
  },
  "bytebase.v1.ListTaskRunsResponse": {
    type: "object",
    properties: [
      {
        name: "taskRuns",
        type: "array<bytebase.v1.TaskRun>",
        description: "The taskRuns from the specified request.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListUsersRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'Filter is used to filter users returned in the list.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Supported filter:\n - name: the user name, support "==" and ".contains()" operator.\n - email: the user email, support "==" and ".contains()" operator.\n - state: check State enum for values, support "==" operator.\n - project: the project full name in "projects/{id}" format, support "==" operator.\n\n For example:\n name == "ed"\n name.contains("ed")\n email == "ed@bytebase.com"\n email.contains("ed")\n state == "DELETED"\n project == "projects/sample-project"\n You can combine filter conditions like:\n name.contains("ed") && project == "projects/sample-project"\n (name == "ed" || email == "ed@bytebase.com") && project == "projects/sample-project"',
      },
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of users to return. The service may return fewer than\n this value.\n If unspecified, at most 10 users will be returned.\n The maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `ListUsers` call.\n Provide this to retrieve the subsequent page.\n\n When paginating, all other parameters provided to `ListUsers` must match\n the call that provided the page token.",
      },
      {
        name: "showDeleted",
        type: "boolean",
        description: "Show deleted users if specified.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListUsersResponse": {
    type: "object",
    properties: [
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
      {
        name: "users",
        type: "array<bytebase.v1.User>",
        description: "The users from the specified request.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ListWorkloadIdentitiesRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'Filter is used to filter workload identities returned in the list.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Supported filter:\n - name: the workload identity name, support "==" and ".contains()" operator.\n - email: the workload identity email, support "==" and ".contains()" operator.\n - state: check State enum for values, support "==" operator.\n\n For example:\n name == "ed"\n name.contains("ed")\n state == "DELETED"\n email == "ed@workload.bytebase.com"\n email.contains("ed")',
      },
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of workload identities to return. The service may return fewer than\n this value.\n If unspecified, at most 10 workload identities will be returned.\n The maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `ListWorkloadIdentities` call.\n Provide this to retrieve the subsequent page.\n\n When paginating, all other parameters provided to `ListWorkloadIdentities` must match\n the call that provided the page token.",
      },
      {
        name: "parent",
        type: "string",
        description:
          "The parent resource.\n Format: projects/{project} for project-level, workspaces/{id} for workspace-level.",
      },
      {
        name: "showDeleted",
        type: "boolean",
        description: "Show deleted workload identities if specified.",
      },
    ],
    description: "Request message for listing workload identities.",
  },
  "bytebase.v1.ListWorkloadIdentitiesResponse": {
    type: "object",
    properties: [
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
      {
        name: "workloadIdentities",
        type: "array<bytebase.v1.WorkloadIdentity>",
        description: "The workload identities from the specified request.",
      },
    ],
    description: "Response message for listing workload identities.",
  },
  "bytebase.v1.ListWorkspacesResponse": {
    type: "object",
    properties: [
      {
        name: "workspaces",
        type: "array<bytebase.v1.Workspace>",
      },
    ],
    description: "",
  },
  "bytebase.v1.LoginRequest": {
    type: "object",
    properties: [
      {
        name: "email",
        type: "string",
        description: "User's email address.",
      },
      {
        name: "idpContext",
        type: "bytebase.v1.IdentityProviderContext",
        description:
          "The idp_context is used to get the user information from identity provider.",
      },
      {
        name: "idpName",
        type: "string",
        description: "The name of the identity provider.\n Format: idps/{idp}",
      },
      {
        name: "mfaTempToken",
        type: "string",
        description:
          "The mfa_temp_token is used to verify the user's identity by MFA.",
      },
      {
        name: "otpCode",
        type: "string",
        description:
          "The otp_code is used to verify the user's identity by MFA.",
      },
      {
        name: "password",
        type: "string",
        description: "User's password for authentication.",
      },
      {
        name: "recoveryCode",
        type: "string",
        description:
          "The recovery_code is used to recovery the user's identity with MFA.",
      },
      {
        name: "web",
        type: "boolean",
        description:
          "If true, sets access token and refresh token as HTTP-only cookies instead of\n returning the token in the response body. Use for browser-based clients.",
      },
    ],
    description: "",
  },
  "bytebase.v1.LoginResponse": {
    type: "object",
    properties: [
      {
        name: "mfaTempToken",
        type: "string",
        description: "Temporary token for MFA verification.",
      },
      {
        name: "requireResetPassword",
        type: "boolean",
        description: "Whether user must reset password before continuing.",
      },
      {
        name: "token",
        type: "string",
        description:
          "Access token for authenticated requests.\n Only returned when web=false. For web=true, the token is set as an HTTP-only cookie.",
      },
      {
        name: "user",
        type: "bytebase.v1.User",
        description: "The user from the successful login.",
      },
    ],
    description: "",
  },
  "bytebase.v1.MaskingExemptionPolicy": {
    type: "object",
    properties: [
      {
        name: "exemptions",
        type: "array<bytebase.v1.MaskingExemptionPolicy.Exemption>",
      },
    ],
    description:
      "MaskingExemptionPolicy is the allowlist of users who can access sensitive data.",
  },
  "bytebase.v1.MaskingExemptionPolicy.Exemption": {
    type: "object",
    properties: [
      {
        name: "condition",
        type: "google.type.Expr",
        description:
          'The condition that is associated with this exception policy instance.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n If the condition is empty, means the user can access all databases without expiration.\n\n Support variables:\n resource.instance_id: the instance resource id. Only support "==" operation.\n resource.database_name: the database name. Only support "==" operation.\n resource.schema_name: the schema name. Only support "==" operation.\n resource.table_name: the table name. Only support "==" operation.\n resource.column_name: the column name. Only support "==" operation.\n request.time: the expiration. Only support "<" operation in `request.time < timestamp("{ISO datetime string format}")`\n All variables should join with "&&" condition.\n\n For example:\n resource.instance_id == "local" && resource.database_name == "employee" && request.time < timestamp("2025-04-30T11:10:39.000Z")\n resource.instance_id == "local" && resource.database_name == "employee"',
      },
      {
        name: "members",
        type: "array<string>",
        description:
          "Specifies the principals who are exempt from masking.\n For users, the member should be: user:{email}\n For groups, the member should be: group:{email}\n For service accounts, the member should be: serviceAccount:{email}\n For workload identities, the member should be: workloadIdentity:{email}",
      },
    ],
    description: "",
  },
  "bytebase.v1.MaskingReason": {
    type: "object",
    properties: [
      {
        name: "algorithm",
        type: "string",
        description: "The masking algorithm used.",
      },
      {
        name: "classificationLevel",
        type: "integer",
        description: "The classification level that triggered masking.",
      },
      {
        name: "context",
        type: "string",
        description:
          'Additional context (e.g., "Matched global rule: PII Protection").',
      },
      {
        name: "maskingRuleId",
        type: "string",
        description: "The masking rule ID that matched (if applicable).",
      },
      {
        name: "semanticTypeIcon",
        type: "string",
        description: "Icon associated with the semantic type (if any).",
      },
      {
        name: "semanticTypeId",
        type: "string",
        description:
          'The semantic type that triggered masking (e.g., "SSN", "email", "phone").',
      },
      {
        name: "semanticTypeTitle",
        type: "string",
        description: "Human-readable semantic type title.",
      },
    ],
    description: "",
  },
  "bytebase.v1.MaskingRulePolicy": {
    type: "object",
    properties: [
      {
        name: "rules",
        type: "array<bytebase.v1.MaskingRulePolicy.MaskingRule>",
        description: "The list of masking rules.",
      },
    ],
    description: "Policy for configuring data masking rules.",
  },
  "bytebase.v1.MaskingRulePolicy.MaskingRule": {
    type: "object",
    properties: [
      {
        name: "condition",
        type: "google.type.Expr",
        description:
          'The condition for the masking rule.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Support variables:\n resource.environment_id: the environment resource id.\n resource.project_id: the project resource id.\n resource.instance_id: the instance resource id.\n resource.database_name: the database name.\n resource.table_name: the table name.\n resource.column_name: the column name.\n resource.classification_level: the classification level (integer).\n\n Each variable support following operations:\n ==: the value equals the target.\n !=: the value not equals the target.\n in: the value matches one of the targets.\n !(in): the value not matches any of the targets.\n <, <=, >, >=: numeric comparison (classification_level only).\n\n For example:\n resource.environment_id == "test" && resource.project_id == "sample-project"\n resource.instance_id == "sample-instance" && resource.database_name == "employee" && resource.table_name in ["table1", "table2"]\n resource.environment_id != "test" || !(resource.project_id in ["poject1", "prject2"])\n resource.instance_id == "sample-instance" && (resource.database_name == "db1" || resource.database_name == "db2")',
      },
      {
        name: "id",
        type: "string",
        description: "A unique identifier for the rule in UUID format.",
      },
      {
        name: "semanticType",
        type: "string",
        description:
          'The semantic type of data to mask (e.g., "SSN", "EMAIL").',
      },
    ],
    description: "A rule that defines when and how to mask data.",
  },
  "bytebase.v1.MaterializedViewMetadata": {
    type: "object",
    properties: [
      {
        name: "comment",
        type: "string",
        description: "The comment is the comment of a materialized view.",
      },
      {
        name: "definition",
        type: "string",
        description: "The definition is the definition of a materialized view.",
      },
      {
        name: "dependencyColumns",
        type: "array<bytebase.v1.DependencyColumn>",
        description:
          "The dependency_columns is the list of dependency columns of a materialized\n view.",
      },
      {
        name: "indexes",
        type: "array<bytebase.v1.IndexMetadata>",
        description: "The indexes is the list of indexes in a table.",
      },
      {
        name: "name",
        type: "string",
        description: "The name is the name of a materialized view.",
      },
      {
        name: "skipDump",
        type: "boolean",
      },
      {
        name: "triggers",
        type: "array<bytebase.v1.TriggerMetadata>",
        description: "The columns is the ordered list of columns in a table.",
      },
    ],
    description:
      "MaterializedViewMetadata is the metadata for materialized views.",
  },
  "bytebase.v1.OAuth2AuthStyle": {
    type: "enum",
    values: ["OAUTH2_AUTH_STYLE_UNSPECIFIED", "IN_PARAMS", "IN_HEADER"],
    description: "",
  },
  "bytebase.v1.OAuth2IdentityProviderConfig": {
    type: "object",
    properties: [
      {
        name: "authStyle",
        type: "bytebase.v1.OAuth2AuthStyle",
        description: "The authentication style for client credentials.",
      },
      {
        name: "authUrl",
        type: "string",
        description: "The authorization endpoint URL for OAuth2 flow.",
      },
      {
        name: "clientId",
        type: "string",
        description: "The OAuth2 client identifier.",
      },
      {
        name: "clientSecret",
        type: "string",
        description: "The OAuth2 client secret for authentication.",
      },
      {
        name: "fieldMapping",
        type: "bytebase.v1.FieldMapping",
        description:
          "Mapping configuration for user attributes from OAuth2 response.",
      },
      {
        name: "scopes",
        type: "array<string>",
        description: "The list of OAuth2 scopes to request.",
      },
      {
        name: "skipTlsVerify",
        type: "boolean",
        description: "Whether to skip TLS certificate verification.",
      },
      {
        name: "tokenUrl",
        type: "string",
        description:
          "The token endpoint URL for exchanging authorization code.",
      },
      {
        name: "userInfoUrl",
        type: "string",
        description: "The user information endpoint URL.",
      },
    ],
    description:
      "OAuth2IdentityProviderConfig is the structure for OAuth2 identity provider config.",
  },
  "bytebase.v1.OAuth2IdentityProviderContext": {
    type: "object",
    properties: [
      {
        name: "code",
        type: "string",
        description: "Authorization code from OAuth2 provider.",
      },
    ],
    description: "OAuth2 authentication context.",
  },
  "bytebase.v1.OAuth2IdentityProviderTestRequestContext": {
    type: "object",
    properties: [
      {
        name: "code",
        type: "string",
        description: "Authorize code from website.",
      },
    ],
    description: "",
  },
  "bytebase.v1.OIDCIdentityProviderConfig": {
    type: "object",
    properties: [
      {
        name: "authEndpoint",
        type: "string",
        description:
          "The authorization endpoint of the OIDC provider.\n Should be fetched from the well-known configuration file of the OIDC provider.",
      },
      {
        name: "authStyle",
        type: "bytebase.v1.OAuth2AuthStyle",
        description: "The authentication style for client credentials.",
      },
      {
        name: "clientId",
        type: "string",
        description: "The OIDC client identifier.",
      },
      {
        name: "clientSecret",
        type: "string",
        description: "The OIDC client secret for authentication.",
      },
      {
        name: "fieldMapping",
        type: "bytebase.v1.FieldMapping",
        description:
          "Mapping configuration for user attributes from OIDC claims.",
      },
      {
        name: "issuer",
        type: "string",
        description: "The OIDC issuer URL for the identity provider.",
      },
      {
        name: "scopes",
        type: "array<string>",
        description:
          "The scopes that the OIDC provider supports.\n Should be fetched from the well-known configuration file of the OIDC provider.",
      },
      {
        name: "skipTlsVerify",
        type: "boolean",
        description: "Whether to skip TLS certificate verification.",
      },
    ],
    description:
      "OIDCIdentityProviderConfig is the structure for OIDC identity provider config.",
  },
  "bytebase.v1.OIDCIdentityProviderContext": {
    type: "object",
    properties: [
      {
        name: "code",
        type: "string",
        description: "Authorization code from OIDC provider.",
      },
    ],
    description: "OpenID Connect authentication context.",
  },
  "bytebase.v1.OIDCIdentityProviderTestRequestContext": {
    type: "object",
    properties: [
      {
        name: "code",
        type: "string",
        description: "Authorize code from OIDC provider.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ObjectSchema": {
    type: "object",
    properties: [
      {
        name: "semanticType",
        type: "string",
        description: "The semantic type of this object.",
      },
      {
        name: "type",
        type: "bytebase.v1.ObjectSchema.Type",
        description: "The data type of this object.",
      },
    ],
    description: "Schema definition for object-type columns.",
  },
  "bytebase.v1.ObjectSchema.ArrayKind": {
    type: "object",
    properties: [
      {
        name: "kind",
        type: "bytebase.v1.ObjectSchema",
        description: "The schema of array elements.",
      },
    ],
    description: "Array type with element schema.",
  },
  "bytebase.v1.ObjectSchema.StructKind": {
    type: "object",
    properties: [
      {
        name: "properties",
        type: "object",
        description: "Properties of the struct.",
      },
    ],
    description: "Structure type with named properties.",
  },
  "bytebase.v1.ObjectSchema.StructKind.PropertiesEntry": {
    type: "object",
    properties: [
      {
        name: "key",
        type: "string",
      },
      {
        name: "value",
        type: "bytebase.v1.ObjectSchema",
      },
    ],
    description: "",
  },
  "bytebase.v1.ObjectSchema.Type": {
    type: "enum",
    values: [
      "TYPE_UNSPECIFIED",
      "STRING",
      "NUMBER",
      "BOOLEAN",
      "OBJECT",
      "ARRAY",
    ],
    description: "Object schema data types.",
  },
  "bytebase.v1.PackageMetadata": {
    type: "object",
    properties: [
      {
        name: "definition",
        type: "string",
        description: "The definition is the definition of a package.",
      },
      {
        name: "name",
        type: "string",
        description: "The name is the name of a package.",
      },
    ],
    description: "PackageMetadata is the metadata for packages.",
  },
  "bytebase.v1.PaymentInfo": {
    type: "object",
    properties: [
      {
        name: "cancelAtPeriodEnd",
        type: "boolean",
        description:
          "Whether the subscription is scheduled to cancel at the end of the current billing period.",
      },
      {
        name: "currency",
        type: "string",
      },
      {
        name: "invoiceUrl",
        type: "string",
        description: "Stripe Billing Portal URL for invoice management.",
      },
      {
        name: "periodEnd",
        type: "string",
      },
      {
        name: "periodStart",
        type: "string",
      },
      {
        name: "totalPrice",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.PermissionDeniedDetail": {
    type: "object",
    properties: [
      {
        name: "method",
        type: "string",
        description: "The API method that was called.",
      },
      {
        name: "requiredPermissions",
        type: "array<string>",
        description: "The permissions required but not granted to the user.",
      },
      {
        name: "resources",
        type: "array<string>",
        description: "The resources the user was trying to access.",
      },
    ],
    description:
      "PermissionDeniedDetail provides structured information about permission failures.\n Used as error detail when returning CodePermissionDenied errors.",
  },
  "bytebase.v1.Plan": {
    type: "object",
    properties: [
      {
        name: "approvalStatus",
        type: "bytebase.v1.Issue.ApprovalStatus",
        description:
          "The approval status of the linked issue.\n Unspecified when no linked issue exists.",
      },
      {
        name: "createTime",
        type: "google.protobuf.Timestamp",
      },
      {
        name: "creator",
        type: "string",
        description: "Format: users/hello@world.com",
      },
      {
        name: "description",
        type: "string",
        description: "The description of the plan.",
      },
      {
        name: "hasRollout",
        type: "boolean",
        description: "Whether the plan has started the rollout.",
      },
      {
        name: "issue",
        type: "string",
        description:
          "The issue associated with the plan.\n Can be empty.\n Format: projects/{project}/issues/{issue}",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the plan.\n `plan` is a system generated ID.\n Format: projects/{project}/plans/{plan}",
      },
      {
        name: "planCheckRunStatusCount",
        type: "object",
        description:
          "The status count of the latest plan check runs.\n Keys are:\n - SUCCESS\n - WARNING\n - ERROR\n - RUNNING",
      },
      {
        name: "rolloutStageSummaries",
        type: "array<bytebase.v1.Plan.RolloutStageSummary>",
        description:
          "Per-stage rollout status summary.\n Ordered by environment deployment order. Empty when no rollout exists.",
      },
      {
        name: "specs",
        type: "array<bytebase.v1.Plan.Spec>",
        description: "The deployment specs for the plan.",
      },
      {
        name: "state",
        type: "bytebase.v1.State",
        description: "The state of the plan.",
      },
      {
        name: "title",
        type: "string",
        description: "The title of the plan.",
      },
      {
        name: "updateTime",
        type: "google.protobuf.Timestamp",
      },
    ],
    description: "",
  },
  "bytebase.v1.Plan.ChangeDatabaseConfig": {
    type: "object",
    properties: [
      {
        name: "enablePriorBackup",
        type: "boolean",
        description:
          "If set, a backup of the modified data will be created automatically before any changes are applied.",
      },
      {
        name: "release",
        type: "string",
        description:
          "The resource name of the release.\n Format: projects/{project}/releases/{release}",
      },
      {
        name: "sheet",
        type: "string",
        description:
          "The resource name of the sheet.\n Format: projects/{project}/sheets/{sheet}",
      },
      {
        name: "targets",
        type: "array<string>",
        description:
          "The list of targets.\n Multi-database format: [instances/{instance-id}/databases/{database-name}].\n Single database group format: [projects/{project}/databaseGroups/{databaseGroup}].",
      },
    ],
    description: "",
  },
  "bytebase.v1.Plan.CreateDatabaseConfig": {
    type: "object",
    properties: [
      {
        name: "characterSet",
        type: "string",
        description: "character_set is the character set of the database.",
      },
      {
        name: "cluster",
        type: "string",
        description:
          'cluster is the cluster of the database. This is only applicable to ClickHouse for "ON CLUSTER <<cluster>>".',
      },
      {
        name: "collation",
        type: "string",
        description: "collation is the collation of the database.",
      },
      {
        name: "database",
        type: "string",
        description: "The name of the database to create.",
      },
      {
        name: "environment",
        type: "string",
        description:
          "The environment resource.\n Format: environments/prod where prod is the environment resource ID.",
      },
      {
        name: "owner",
        type: "string",
        description:
          'owner is the owner of the database. This is only applicable to Postgres for "WITH OWNER <<owner>>".',
      },
      {
        name: "table",
        type: "string",
        description:
          "table is the name of the table, if it is not empty, Bytebase should create a table after creating the database.\n For example, in MongoDB, it only creates the database when we first store data in that database.",
      },
      {
        name: "target",
        type: "string",
        description:
          "The resource name of the instance on which the database is created.\n Format: instances/{instance}",
      },
    ],
    description: "",
  },
  "bytebase.v1.Plan.ExportDataConfig": {
    type: "object",
    properties: [
      {
        name: "format",
        type: "bytebase.v1.ExportFormat",
        description: "The format of the exported file.",
      },
      {
        name: "password",
        type: "string",
        description:
          "The zip password provide by users.\n Leave it empty if no needs to encrypt the zip file.",
      },
      {
        name: "sheet",
        type: "string",
        description:
          "The resource name of the sheet.\n Format: projects/{project}/sheets/{sheet}",
      },
      {
        name: "targets",
        type: "array<string>",
        description:
          "The list of targets.\n Multi-database format: [instances/{instance-id}/databases/{database-name}].\n Single database group format: [projects/{project}/databaseGroups/{databaseGroup}].",
      },
    ],
    description: "",
  },
  "bytebase.v1.Plan.PlanCheckRunStatusCountEntry": {
    type: "object",
    properties: [
      {
        name: "key",
        type: "string",
      },
      {
        name: "value",
        type: "integer",
      },
    ],
    description: "",
  },
  "bytebase.v1.Plan.RolloutStageSummary": {
    type: "object",
    properties: [
      {
        name: "stage",
        type: "string",
        description:
          "The stage resource name.\n Format: projects/{project}/plans/{plan}/rollout/stages/{stage}",
      },
      {
        name: "taskStatusCounts",
        type: "array<bytebase.v1.Plan.TaskStatusCount>",
        description: "Task status counts for this stage.",
      },
    ],
    description: "",
  },
  "bytebase.v1.Plan.Spec": {
    type: "object",
    properties: [
      {
        name: "id",
        type: "string",
        description: "A UUID4 string that uniquely identifies the Spec.",
      },
    ],
    description: "",
  },
  "bytebase.v1.Plan.TaskStatusCount": {
    type: "object",
    properties: [
      {
        name: "count",
        type: "integer",
        description: "The number of tasks in the status.",
      },
      {
        name: "status",
        type: "bytebase.v1.Task.Status",
        description: "The task status.",
      },
    ],
    description: "",
  },
  "bytebase.v1.PlanCheckRun": {
    type: "object",
    properties: [
      {
        name: "createTime",
        type: "google.protobuf.Timestamp",
      },
      {
        name: "error",
        type: "string",
        description: "error is set if the Status is FAILED.",
      },
      {
        name: "name",
        type: "string",
        description: "Format: projects/{project}/plans/{plan}/planCheckRun",
      },
      {
        name: "results",
        type: "array<bytebase.v1.PlanCheckRun.Result>",
      },
      {
        name: "status",
        type: "bytebase.v1.PlanCheckRun.Status",
      },
    ],
    description: "",
  },
  "bytebase.v1.PlanCheckRun.Result": {
    type: "object",
    properties: [
      {
        name: "code",
        type: "integer",
      },
      {
        name: "content",
        type: "string",
      },
      {
        name: "status",
        type: "bytebase.v1.Advice.Level",
      },
      {
        name: "target",
        type: "string",
        description:
          "Target identification for consolidated results.\n Format: instances/{instance}/databases/{database}",
      },
      {
        name: "title",
        type: "string",
      },
      {
        name: "type",
        type: "bytebase.v1.PlanCheckRun.Result.Type",
      },
    ],
    description: "",
  },
  "bytebase.v1.PlanCheckRun.Result.SqlReviewReport": {
    type: "object",
    properties: [
      {
        name: "endPosition",
        type: "bytebase.v1.Position",
      },
      {
        name: "startPosition",
        type: "bytebase.v1.Position",
        description: "Position of the SQL statement.",
      },
    ],
    description: "",
  },
  "bytebase.v1.PlanCheckRun.Result.SqlSummaryReport": {
    type: "object",
    properties: [
      {
        name: "affectedRows",
        type: "integer",
      },
      {
        name: "statementTypes",
        type: "array<bytebase.v1.StatementType>",
        description:
          "statement_types are the types of statements that are found in the sql.",
      },
    ],
    description: "",
  },
  "bytebase.v1.PlanCheckRun.Result.Type": {
    type: "enum",
    values: [
      "TYPE_UNSPECIFIED",
      "STATEMENT_ADVISE",
      "STATEMENT_SUMMARY_REPORT",
      "GHOST_SYNC",
    ],
    description: "",
  },
  "bytebase.v1.PlanCheckRun.Status": {
    type: "enum",
    values: ["STATUS_UNSPECIFIED", "RUNNING", "DONE", "FAILED", "CANCELED"],
    description: "",
  },
  "bytebase.v1.PlanConfig": {
    type: "object",
    properties: [
      {
        name: "instanceFeatures",
        type: "array<bytebase.v1.PlanFeature>",
      },
      {
        name: "plans",
        type: "array<bytebase.v1.PlanLimitConfig>",
      },
    ],
    description:
      "PlanConfig represents the configuration for all plans loaded from plan.yaml",
  },
  "bytebase.v1.PlanFeature": {
    type: "enum",
    values: [
      "FEATURE_UNSPECIFIED",
      "FEATURE_DATABASE_CHANGE",
      "FEATURE_GIT_BASED_SCHEMA_VERSION_CONTROL",
      "FEATURE_DECLARATIVE_SCHEMA_MIGRATION",
      "FEATURE_COMPARE_AND_SYNC_SCHEMA",
      "FEATURE_ONLINE_SCHEMA_CHANGE",
      "FEATURE_PRE_DEPLOYMENT_SQL_REVIEW",
      "FEATURE_AUTOMATIC_BACKUP_BEFORE_DATA_CHANGES",
      "FEATURE_ONE_CLICK_DATA_ROLLBACK",
      "FEATURE_MULTI_DATABASE_BATCH_CHANGES",
      "FEATURE_PROGRESSIVE_ENVIRONMENT_DEPLOYMENT",
      "FEATURE_SCHEDULED_ROLLOUT_TIME",
      "FEATURE_DATABASE_CHANGELOG",
      "FEATURE_SCHEMA_DRIFT_DETECTION",
      "FEATURE_ROLLOUT_POLICY",
      "FEATURE_WEB_BASED_SQL_EDITOR",
      "FEATURE_SQL_EDITOR_ADMIN_MODE",
      "FEATURE_NATURAL_LANGUAGE_TO_SQL",
      "FEATURE_AI_QUERY_EXPLANATION",
      "FEATURE_AI_QUERY_SUGGESTIONS",
      "FEATURE_AUTO_COMPLETE",
      "FEATURE_SCHEMA_DIAGRAM",
      "FEATURE_SCHEMA_EDITOR",
      "FEATURE_DATA_EXPORT",
      "FEATURE_DATA_OFFLINE_EXPORT",
      "FEATURE_QUERY_HISTORY",
      "FEATURE_SAVED_AND_SHARED_SQL_SCRIPTS",
      "FEATURE_BATCH_QUERY",
      "FEATURE_INSTANCE_READ_ONLY_CONNECTION",
      "FEATURE_QUERY_POLICY",
      "FEATURE_RESTRICT_COPYING_DATA",
      "FEATURE_IAM",
      "FEATURE_INSTANCE_SSL_CONNECTION",
      "FEATURE_INSTANCE_CONNECTION_OVER_SSH_TUNNEL",
      "FEATURE_INSTANCE_CONNECTION_IAM_AUTHENTICATION",
      "FEATURE_GOOGLE_AND_GITHUB_SSO",
      "FEATURE_USER_GROUPS",
      "FEATURE_DISALLOW_SELF_SERVICE_SIGNUP",
      "FEATURE_CUSTOM_INSTANCE_SYNC_TIME",
      "FEATURE_CUSTOM_INSTANCE_CONNECTION_LIMIT",
      "FEATURE_RISK_ASSESSMENT",
      "FEATURE_APPROVAL_WORKFLOW",
      "FEATURE_AUDIT_LOG",
      "FEATURE_ENTERPRISE_SSO",
      "FEATURE_TWO_FA",
      "FEATURE_PASSWORD_RESTRICTIONS",
      "FEATURE_DISALLOW_PASSWORD_SIGNIN",
      "FEATURE_CUSTOM_ROLES",
      "FEATURE_REQUEST_ROLE_WORKFLOW",
      "FEATURE_JIT",
      "FEATURE_DATA_MASKING",
      "FEATURE_DATA_CLASSIFICATION",
      "FEATURE_SCIM",
      "FEATURE_DIRECTORY_SYNC",
      "FEATURE_TOKEN_DURATION_CONTROL",
      "FEATURE_EXTERNAL_SECRET_MANAGER",
      "FEATURE_USER_EMAIL_DOMAIN_RESTRICTION",
      "FEATURE_PROJECT_MANAGEMENT",
      "FEATURE_ENVIRONMENT_MANAGEMENT",
      "FEATURE_IM_NOTIFICATIONS",
      "FEATURE_TERRAFORM_PROVIDER",
      "FEATURE_DATABASE_GROUPS",
      "FEATURE_ENVIRONMENT_TIERS",
      "FEATURE_DASHBOARD_ANNOUNCEMENT",
      "FEATURE_API_INTEGRATION_GUIDANCE",
      "FEATURE_CUSTOM_LOGO",
      "FEATURE_WATERMARK",
      "FEATURE_ROADMAP_PRIORITIZATION",
      "FEATURE_CUSTOM_MSA",
      "FEATURE_COMMUNITY_SUPPORT",
      "FEATURE_EMAIL_SUPPORT",
      "FEATURE_DEDICATED_SUPPORT_WITH_SLA",
    ],
    description: "PlanFeature represents the available features in Bytebase",
  },
  "bytebase.v1.PlanLimitConfig": {
    type: "object",
    properties: [
      {
        name: "features",
        type: "array<bytebase.v1.PlanFeature>",
      },
      {
        name: "maximumInstanceCount",
        type: "integer",
      },
      {
        name: "maximumSeatCount",
        type: "integer",
      },
      {
        name: "type",
        type: "bytebase.v1.PlanType",
      },
    ],
    description: "PlanLimitConfig represents a single plan's configuration",
  },
  "bytebase.v1.PlanType": {
    type: "enum",
    values: ["PLAN_TYPE_UNSPECIFIED", "FREE", "TEAM", "ENTERPRISE"],
    description: "",
  },
  "bytebase.v1.Policy": {
    type: "object",
    properties: [
      {
        name: "enforce",
        type: "boolean",
        description: "Whether the policy is enforced.",
      },
      {
        name: "inheritFromParent",
        type: "boolean",
        description: "Whether this policy inherits from its parent resource.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the policy.\n Format: {resource name}/policies/{policy type}\n Workspace resource name: workspaces/{workspace-id}.\n Environment resource name: environments/environment-id.\n Instance resource name: instances/instance-id.\n Database resource name: instances/instance-id/databases/database-name.",
      },
      {
        name: "resourceType",
        type: "bytebase.v1.PolicyResourceType",
        description: "The resource type for the policy.",
      },
      {
        name: "type",
        type: "bytebase.v1.PolicyType",
        description: "The type of policy.",
      },
    ],
    description: "",
  },
  "bytebase.v1.PolicyDelta": {
    type: "object",
    properties: [
      {
        name: "bindingDeltas",
        type: "array<bytebase.v1.BindingDelta>",
        description: "The delta for Bindings between two policies.",
      },
    ],
    description: "Describes changes between two IAM policies.",
  },
  "bytebase.v1.PolicyResourceType": {
    type: "enum",
    values: [
      "RESOURCE_TYPE_UNSPECIFIED",
      "WORKSPACE",
      "ENVIRONMENT",
      "PROJECT",
    ],
    description: "The resource type that a policy can be attached to.",
  },
  "bytebase.v1.PolicyType": {
    type: "enum",
    values: [
      "POLICY_TYPE_UNSPECIFIED",
      "MASKING_RULE",
      "MASKING_EXEMPTION",
      "ROLLOUT_POLICY",
      "TAG",
      "DATA_QUERY",
    ],
    description: "The type of organizational policy.",
  },
  "bytebase.v1.Position": {
    type: "object",
    properties: [
      {
        name: "column",
        type: "integer",
        description:
          "Column position in a text (one-based).\n Column is measured in Unicode code points (characters/runes), not bytes or grapheme clusters.\n First character of the line is column 1.\n A value of 0 indicates the column information is unknown.\n\n Examples:\n - \"SELECT * FROM t\" - column 8 is '*'\n - \"SELECT 你好 FROM t\" - column 8 is '你' (even though it's at byte offset 7)\n - \"SELECT 😀 FROM t\" - column 8 is '😀' (even though it's 4 bytes in UTF-8)",
      },
      {
        name: "line",
        type: "integer",
        description:
          "Line position in a text (one-based).\n First line of the text is line 1.\n A value of 0 indicates the line information is unknown.",
      },
    ],
    description:
      "Position in a text expressed as one-based line and one-based column.\n We use 1-based numbering to match the majority of industry standards:\n - Monaco Editor uses 1-based (https://microsoft.github.io/monaco-editor/typedoc/interfaces/IPosition.html)\n - GitHub Actions uses 1-based (https://docs.github.com/en/actions/using-workflows/workflow-commands-for-github-actions#setting-an-error-message)\n - Most text editors display 1-based positions to users\n Note: LSP uses 0-based (https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#position),\n but we follow the canonical user-facing standards.\n\n Handling unknown positions:\n - If the entire position is unknown, leave this field as nil/undefined\n - If only line is known, set line and leave column as 0 (e.g., line=5, column=0)\n - If only column is known (rare), set column and leave line as 0\n Frontends should check for nil/undefined/zero values and handle them appropriately.",
  },
  "bytebase.v1.PreviewTaskRunRollbackRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun}",
      },
    ],
    description: "",
  },
  "bytebase.v1.PreviewTaskRunRollbackResponse": {
    type: "object",
    properties: [
      {
        name: "statement",
        type: "string",
        description: "The rollback SQL statement that would undo the task run.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ProcedureMetadata": {
    type: "object",
    properties: [
      {
        name: "characterSetClient",
        type: "string",
        description: "MySQL specific metadata.",
      },
      {
        name: "collationConnection",
        type: "string",
      },
      {
        name: "comment",
        type: "string",
        description: "The comment is the comment of a procedure.",
      },
      {
        name: "databaseCollation",
        type: "string",
      },
      {
        name: "definition",
        type: "string",
        description: "The definition is the definition of a procedure.",
      },
      {
        name: "name",
        type: "string",
        description: "The name is the name of a procedure.",
      },
      {
        name: "signature",
        type: "string",
        description:
          "The signature is the name with the number and type of input arguments the\n procedure takes.",
      },
      {
        name: "skipDump",
        type: "boolean",
      },
      {
        name: "sqlMode",
        type: "string",
      },
    ],
    description: "ProcedureMetadata is the metadata for procedures.",
  },
  "bytebase.v1.Project": {
    type: "object",
    properties: [
      {
        name: "allowJustInTimeAccess",
        type: "boolean",
        description:
          "Once enabled, users can request and use the just-in-time access in the SQL Editor.",
      },
      {
        name: "allowRequestRole",
        type: "boolean",
      },
      {
        name: "allowSelfApproval",
        type: "boolean",
        description:
          "Whether to allow issue creators to self-approve their own issues.",
      },
      {
        name: "ciSamplingSize",
        type: "integer",
        description:
          "The maximum number of database rows to sample during CI data validation.\n Without specification, sampling is disabled, resulting in full validation.",
      },
      {
        name: "dataClassificationConfigId",
        type: "string",
        description:
          "The data classification configuration ID for the project.",
      },
      {
        name: "enforceIssueTitle",
        type: "boolean",
        description:
          "Enforce issue title to be created by user instead of generated by Bytebase.",
      },
      {
        name: "enforceSqlReview",
        type: "boolean",
        description:
          "Whether to enforce SQL review checks to pass before issue creation.\n If enabled, issues cannot be created when SQL review finds errors.",
      },
      {
        name: "executionRetryPolicy",
        type: "bytebase.v1.Project.ExecutionRetryPolicy",
        description: "Execution retry policy for task runs.",
      },
      {
        name: "forceIssueLabels",
        type: "boolean",
        description: "Force issue labels to be used when creating an issue.",
      },
      {
        name: "issueLabels",
        type: "array<bytebase.v1.Label>",
        description: "Labels available for tagging issues in this project.",
      },
      {
        name: "labels",
        type: "object",
        description:
          'Labels are key-value pairs that can be attached to the project.\n For example, { "environment": "production", "team": "backend" }',
      },
      {
        name: "name",
        type: "string",
        description: "The name of the project.\n Format: projects/{project}",
      },
      {
        name: "parallelTasksPerRollout",
        type: "integer",
        description:
          "The maximum number of parallel tasks allowed during rollout execution.",
      },
      {
        name: "postgresDatabaseTenantMode",
        type: "boolean",
        description:
          'Whether to enable database tenant mode for PostgreSQL.\n If enabled, issues will include "set role <db_owner>" statement.',
      },
      {
        name: "requireIssueApproval",
        type: "boolean",
        description: "Whether to require issue approval before rollout.",
      },
      {
        name: "requirePlanCheckNoError",
        type: "boolean",
        description:
          "Whether to require plan check to have no error before rollout.",
      },
      {
        name: "state",
        type: "bytebase.v1.State",
        description: "The lifecycle state of the project.",
      },
      {
        name: "title",
        type: "string",
        description:
          "The title or name of a project. It's not unique within the workspace.",
      },
      {
        name: "webhooks",
        type: "array<bytebase.v1.Webhook>",
        description: "The list of webhooks configured for the project.",
      },
    ],
    description: "",
  },
  "bytebase.v1.Project.ExecutionRetryPolicy": {
    type: "object",
    properties: [
      {
        name: "maximumRetries",
        type: "integer",
        description: "The maximum number of retries for lock timeout errors.",
      },
    ],
    description: "Execution retry policy configuration.",
  },
  "bytebase.v1.Project.LabelsEntry": {
    type: "object",
    properties: [
      {
        name: "key",
        type: "string",
      },
      {
        name: "value",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.PurchaseBillingMethod": {
    type: "object",
    properties: [
      {
        name: "discount",
        type: "bytebase.v1.PurchaseDiscount",
      },
      {
        name: "interval",
        type: "bytebase.v1.BillingInterval",
      },
    ],
    description: "",
  },
  "bytebase.v1.PurchaseDiscount": {
    type: "object",
    properties: [
      {
        name: "type",
        type: "bytebase.v1.PurchaseDiscount.Type",
      },
      {
        name: "value",
        type: "integer",
      },
    ],
    description: "",
  },
  "bytebase.v1.PurchaseDiscount.Type": {
    type: "enum",
    values: [
      "TYPE_UNSPECIFIED",
      "PERCENTAGE_OFF",
      "FIXED_MONTH_OFF",
      "FIXED_PRICE_OFF",
    ],
    description: "",
  },
  "bytebase.v1.PurchasePlan": {
    type: "object",
    properties: [
      {
        name: "additionals",
        type: "array<bytebase.v1.PurchasePlanAdditional>",
      },
      {
        name: "billingMethods",
        type: "array<bytebase.v1.PurchaseBillingMethod>",
      },
      {
        name: "selfServicePurchase",
        type: "boolean",
      },
      {
        name: "type",
        type: "bytebase.v1.PlanType",
      },
    ],
    description: "",
  },
  "bytebase.v1.PurchasePlanAdditional": {
    type: "object",
    properties: [
      {
        name: "freeCount",
        type: "integer",
      },
      {
        name: "maximumCount",
        type: "integer",
        description: "-1 means unlimited.",
      },
      {
        name: "minimumCount",
        type: "integer",
      },
      {
        name: "type",
        type: "bytebase.v1.PurchasePlanAdditional.Type",
      },
      {
        name: "unitPrice",
        type: "integer",
        description: "Price in USD cents per month.",
      },
    ],
    description: "",
  },
  "bytebase.v1.PurchasePlanAdditional.Type": {
    type: "enum",
    values: ["TYPE_UNSPECIFIED", "USER"],
    description: "",
  },
  "bytebase.v1.PurchaseResponse": {
    type: "object",
    properties: [
      {
        name: "paymentUrl",
        type: "string",
        description:
          "If set, redirect to this Stripe Checkout URL.\n If empty, the update was applied directly using the existing payment method.",
      },
      {
        name: "sessionId",
        type: "string",
        description:
          "Stripe Checkout Session ID. Used to verify the session after redirect.",
      },
    ],
    description: "",
  },
  "bytebase.v1.QueryDataPolicy": {
    type: "object",
    properties: [
      {
        name: "allowAdminDataSource",
        type: "boolean",
        description:
          "workspace-level policy\n Allow using the admin data source to query in the SQL editor.\n If true, users can select the admin data source or read-only data source\n If false,\n 1. when read-only data source is configured, users're force to use the read-only data source\n 2. otherwise fallback to use the admin data source.",
      },
      {
        name: "disableCopyData",
        type: "boolean",
        description:
          "workspace-level policy\n Disable copying query results in the SQL editor.",
      },
      {
        name: "disableExport",
        type: "boolean",
        description:
          "workspace-level policy\n Disable data export in the SQL editor.",
      },
      {
        name: "maximumResultRows",
        type: "integer",
        description:
          "Support both project-level and workspace-level.\n The maximum number of rows to return in the SQL editor.\n The default value <= 0, means no limit.",
      },
    ],
    description:
      "QueryDataPolicy is the policy configuration for querying data in the SQL Editor.",
  },
  "bytebase.v1.QueryHistory": {
    type: "object",
    properties: [
      {
        name: "createTime",
        type: "google.protobuf.Timestamp",
      },
      {
        name: "creator",
        type: "string",
      },
      {
        name: "database",
        type: "string",
        description:
          "The database name to execute the query.\n Format: instances/{instance}/databases/{databaseName}",
      },
      {
        name: "duration",
        type: "google.protobuf.Duration",
      },
      {
        name: "error",
        type: "string",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name for the query history.\n Format: projects/{project}/queryHistories/{id}",
      },
      {
        name: "statement",
        type: "string",
      },
      {
        name: "type",
        type: "bytebase.v1.QueryHistory.Type",
      },
    ],
    description: "",
  },
  "bytebase.v1.QueryHistory.Type": {
    type: "enum",
    values: ["TYPE_UNSPECIFIED", "QUERY", "EXPORT"],
    description: "",
  },
  "bytebase.v1.QueryOption": {
    type: "object",
    properties: [
      {
        name: "mssqlExplainFormat",
        type: "bytebase.v1.QueryOption.MSSQLExplainFormat",
      },
      {
        name: "redisRunCommandsOn",
        type: "bytebase.v1.QueryOption.RedisRunCommandsOn",
      },
    ],
    description: "",
  },
  "bytebase.v1.QueryOption.MSSQLExplainFormat": {
    type: "enum",
    values: [
      "MSSQL_EXPLAIN_FORMAT_UNSPECIFIED",
      "MSSQL_EXPLAIN_FORMAT_ALL",
      "MSSQL_EXPLAIN_FORMAT_XML",
    ],
    description: "",
  },
  "bytebase.v1.QueryOption.RedisRunCommandsOn": {
    type: "enum",
    values: ["REDIS_RUN_COMMANDS_ON_UNSPECIFIED", "SINGLE_NODE", "ALL_NODES"],
    description: "",
  },
  "bytebase.v1.QueryRequest": {
    type: "object",
    properties: [
      {
        name: "container",
        type: "string",
        description:
          "Container is the container name to execute the query against, used for\n CosmosDB only.",
      },
      {
        name: "dataSourceId",
        type: "string",
        description:
          "The id of data source.\n If omitted, Query resolves the data source server-side by using the single\n read-only data source when exactly one exists, or the admin data source\n otherwise. It can also be set explicitly to query the admin data source or\n a specific read-only data source.",
      },
      {
        name: "explain",
        type: "boolean",
        description: "Explain the statement.",
      },
      {
        name: "limit",
        type: "integer",
        description: "The maximum number of rows to return.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name is the instance name to execute the query against.\n Format: instances/{instance}/databases/{databaseName}",
      },
      {
        name: "queryOption",
        type: "bytebase.v1.QueryOption",
      },
      {
        name: "schema",
        type: "string",
        description:
          "The default schema to search objects. Equals to the current schema in\n Oracle and search path in Postgres.",
      },
      {
        name: "statement",
        type: "string",
        description: "The SQL statement to execute.",
      },
    ],
    description: "",
  },
  "bytebase.v1.QueryResponse": {
    type: "object",
    properties: [
      {
        name: "results",
        type: "array<bytebase.v1.QueryResult>",
        description: "The query results.",
      },
    ],
    description: "",
  },
  "bytebase.v1.QueryResult": {
    type: "object",
    properties: [
      {
        name: "columnNames",
        type: "array<string>",
        description: "Column names of the query result.",
      },
      {
        name: "columnTypeNames",
        type: "array<string>",
        description:
          "Column types of the query result.\n The types come from the Golang SQL driver.",
      },
      {
        name: "error",
        type: "string",
        description: "The error message if the query failed.",
      },
      {
        name: "latency",
        type: "google.protobuf.Duration",
        description: "The time it takes to execute the query.",
      },
      {
        name: "masked",
        type: "array<bytebase.v1.MaskingReason>",
        description:
          "Masking reasons for each column (empty for non-masked columns).",
      },
      {
        name: "messages",
        type: "array<bytebase.v1.QueryResult.Message>",
        description:
          "Informational or debug messages returned by the database engine during query execution.\n Examples include PostgreSQL's RAISE NOTICE, MSSQL's PRINT, or Oracle's DBMS_OUTPUT.PUT_LINE.",
      },
      {
        name: "rows",
        type: "array<bytebase.v1.QueryRow>",
        description: "Rows of the query result.",
      },
      {
        name: "rowsCount",
        type: "integer",
      },
      {
        name: "statement",
        type: "string",
        description: "The query statement for the result.",
      },
    ],
    description: "",
  },
  "bytebase.v1.QueryResult.CommandError": {
    type: "object",
    properties: [
      {
        name: "commandType",
        type: "bytebase.v1.QueryResult.CommandError.Type",
        description: "Disallowed command_type.",
      },
    ],
    description:
      "Permission denied with resource information or disallowed command_type.\n Either resources or command_type is available.",
  },
  "bytebase.v1.QueryResult.CommandError.Type": {
    type: "enum",
    values: ["TYPE_UNSPECIFIED", "DDL", "DML", "NON_READ_ONLY"],
    description: "",
  },
  "bytebase.v1.QueryResult.Message": {
    type: "object",
    properties: [
      {
        name: "content",
        type: "string",
      },
      {
        name: "level",
        type: "bytebase.v1.QueryResult.Message.Level",
      },
    ],
    description: "",
  },
  "bytebase.v1.QueryResult.Message.Level": {
    type: "enum",
    values: [
      "LEVEL_UNSPECIFIED",
      "INFO",
      "WARNING",
      "DEBUG",
      "LOG",
      "NOTICE",
      "EXCEPTION",
    ],
    description: "",
  },
  "bytebase.v1.QueryResult.PostgresError": {
    type: "object",
    properties: [
      {
        name: "code",
        type: "string",
      },
      {
        name: "columnName",
        type: "string",
      },
      {
        name: "constraintName",
        type: "string",
      },
      {
        name: "dataTypeName",
        type: "string",
      },
      {
        name: "detail",
        type: "string",
      },
      {
        name: "file",
        type: "string",
      },
      {
        name: "hint",
        type: "string",
      },
      {
        name: "internalPosition",
        type: "integer",
      },
      {
        name: "internalQuery",
        type: "string",
      },
      {
        name: "line",
        type: "integer",
      },
      {
        name: "message",
        type: "string",
      },
      {
        name: "position",
        type: "integer",
      },
      {
        name: "routine",
        type: "string",
      },
      {
        name: "schemaName",
        type: "string",
      },
      {
        name: "severity",
        type: "string",
      },
      {
        name: "tableName",
        type: "string",
      },
      {
        name: "where",
        type: "string",
      },
    ],
    description:
      "refer https://www.postgresql.org/docs/11/protocol-error-fields.html\n for field description.",
  },
  "bytebase.v1.QueryResult.SyntaxError": {
    type: "object",
    properties: [
      {
        name: "startPosition",
        type: "bytebase.v1.Position",
        description: "Position information for highlighting in editor",
      },
    ],
    description:
      "Syntax error with position information for editor highlighting",
  },
  "bytebase.v1.QueryRow": {
    type: "object",
    properties: [
      {
        name: "values",
        type: "array<bytebase.v1.RowValue>",
        description: "Row values of the query result.",
      },
    ],
    description: "",
  },
  "bytebase.v1.Range": {
    type: "object",
    properties: [
      {
        name: "end",
        type: "integer",
        description: "End index (exclusive).",
      },
      {
        name: "start",
        type: "integer",
        description: "Start index (inclusive).",
      },
    ],
    description:
      "Range represents a span within a text or sequence.\n Whether the indices are byte offsets or character indices depends on the context.\n Check the documentation of the field using Range for specific semantics.",
  },
  "bytebase.v1.RejectIssueRequest": {
    type: "object",
    properties: [
      {
        name: "comment",
        type: "string",
        description: "The comment explaining the rejection decision.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the issue to add a rejection.\n Format: projects/{project}/issues/{issue}",
      },
    ],
    description: "",
  },
  "bytebase.v1.Release": {
    type: "object",
    properties: [
      {
        name: "category",
        type: "string",
        description:
          'Category extracted from release name (e.g., "webapp", "analytics").\n Set by Bytebase action during release creation.',
      },
      {
        name: "createTime",
        type: "google.protobuf.Timestamp",
      },
      {
        name: "creator",
        type: "string",
        description: "Format: users/hello@world.com",
      },
      {
        name: "files",
        type: "array<bytebase.v1.Release.File>",
        description: "The SQL files included in the release.",
      },
      {
        name: "name",
        type: "string",
        description: "Format: projects/{project}/releases/{release}",
      },
      {
        name: "state",
        type: "bytebase.v1.State",
        description: "The lifecycle state of the release.",
      },
      {
        name: "type",
        type: "bytebase.v1.Release.Type",
        description: "The type of schema change for all files in this release.",
      },
      {
        name: "vcsSource",
        type: "bytebase.v1.Release.VCSSource",
        description: "The version control source of the release.",
      },
    ],
    description: "",
  },
  "bytebase.v1.Release.File": {
    type: "object",
    properties: [
      {
        name: "path",
        type: "string",
        description:
          "The path of the file. e.g., `2.2/V0001_create_table.sql`.",
      },
      {
        name: "sheet",
        type: "string",
        description:
          "For inputs, we must either use `sheet` or `statement`.\n For outputs, we always use `sheet`. `statement` is the preview of the sheet content.\n\n The sheet that holds the content.\n Format: projects/{project}/sheets/{sheet}",
      },
      {
        name: "sheetSha256",
        type: "string",
        description:
          "The SHA256 hash value of the sheet content or the statement.",
      },
      {
        name: "statement",
        type: "string",
        description: "The raw SQL statement content.",
      },
      {
        name: "version",
        type: "string",
        description: "The version identifier for the file.",
      },
    ],
    description: "A SQL file in a release.",
  },
  "bytebase.v1.Release.Type": {
    type: "enum",
    values: ["TYPE_UNSPECIFIED", "VERSIONED", "DECLARATIVE"],
    description: "The type of schema change.",
  },
  "bytebase.v1.Release.VCSSource": {
    type: "object",
    properties: [
      {
        name: "url",
        type: "string",
        description: "The url link to the e.g., GitHub commit or pull request.",
      },
      {
        name: "vcsType",
        type: "bytebase.v1.VCSType",
        description: "The type of VCS.",
      },
    ],
    description: "Version control system source information.",
  },
  "bytebase.v1.RemoveDataSourceRequest": {
    type: "object",
    properties: [
      {
        name: "dataSource",
        type: "bytebase.v1.DataSource",
        description:
          "Identified by data source ID.\n Only READ_ONLY data source can be removed.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the instance to remove a data source from.\n Format: instances/{instance}",
      },
    ],
    description: "",
  },
  "bytebase.v1.RemoveWebhookRequest": {
    type: "object",
    properties: [
      {
        name: "webhook",
        type: "bytebase.v1.Webhook",
        description: "The webhook to remove. Identified by its url.",
      },
    ],
    description: "",
  },
  "bytebase.v1.RequestIssueRequest": {
    type: "object",
    properties: [
      {
        name: "comment",
        type: "string",
        description: "The comment explaining the request.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the issue to request a issue.\n Format: projects/{project}/issues/{issue}",
      },
    ],
    description: "",
  },
  "bytebase.v1.RequestMetadata": {
    type: "object",
    properties: [
      {
        name: "callerIp",
        type: "string",
        description: "The IP address of the request originator.",
      },
      {
        name: "callerSuppliedUserAgent",
        type: "string",
        description:
          "The user agent string provided by the caller.\n This is supplied by the client and is not authenticated.",
      },
    ],
    description: "Metadata about the incoming request.",
  },
  "bytebase.v1.Restriction": {
    type: "object",
    properties: [
      {
        name: "disallowPasswordSignin",
        type: "boolean",
        description:
          "Whether password-based signin is disabled (except for workspace admins).",
      },
      {
        name: "disallowSignup",
        type: "boolean",
        description: "Whether self-service user signup is disabled.",
      },
      {
        name: "passwordRestriction",
        type: "bytebase.v1.WorkspaceProfileSetting.PasswordRestriction",
        description: "Password complexity and restriction requirements.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ReviewConfig": {
    type: "object",
    properties: [
      {
        name: "enabled",
        type: "boolean",
        description: "Whether the review configuration is enabled.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the SQL review config.\n Format: reviewConfigs/{reviewConfig}",
      },
      {
        name: "resources",
        type: "array<string>",
        description:
          "Resources using the config.\n Format: {resource}/{resource id}, e.g., environments/test.",
      },
      {
        name: "rules",
        type: "array<bytebase.v1.SQLReviewRule>",
        description: "The SQL review rules to enforce.",
      },
      {
        name: "title",
        type: "string",
        description: "The title of the review configuration.",
      },
    ],
    description: "",
  },
  "bytebase.v1.Revision": {
    type: "object",
    properties: [
      {
        name: "createTime",
        type: "google.protobuf.Timestamp",
      },
      {
        name: "deleter",
        type: "string",
        description: "Format: users/hello@world.com\n Can be empty.",
      },
      {
        name: "deleteTime",
        type: "google.protobuf.Timestamp",
        description: "Can be empty.",
      },
      {
        name: "file",
        type: "string",
        description:
          "Format: projects/{project}/releases/{release}/files/{id}\n Can be empty.",
      },
      {
        name: "name",
        type: "string",
        description:
          "Format: instances/{instance}/databases/{database}/revisions/{revision}",
      },
      {
        name: "release",
        type: "string",
        description:
          "Format: projects/{project}/releases/{release}\n Can be empty.",
      },
      {
        name: "sheet",
        type: "string",
        description:
          "The sheet that holds the content.\n Format: projects/{project}/sheets/{sheet}",
      },
      {
        name: "sheetSha256",
        type: "string",
        description: "The SHA256 hash value of the sheet.",
      },
      {
        name: "taskRun",
        type: "string",
        description:
          "The task run associated with the revision.\n Can be empty.\n Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun}",
      },
      {
        name: "type",
        type: "bytebase.v1.Revision.Type",
        description: "The type of the revision.",
      },
      {
        name: "version",
        type: "string",
        description: "The schema version string for this revision.",
      },
    ],
    description: "",
  },
  "bytebase.v1.Revision.Type": {
    type: "enum",
    values: ["TYPE_UNSPECIFIED", "VERSIONED", "DECLARATIVE"],
    description: "The type of schema revision.",
  },
  "bytebase.v1.RevokeAccessGrantRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the access grant to revoke.\n Format: projects/{project}/accessGrants/{access_grant}",
      },
    ],
    description: "",
  },
  "bytebase.v1.RiskLevel": {
    type: "enum",
    values: ["RISK_LEVEL_UNSPECIFIED", "LOW", "MODERATE", "HIGH"],
    description: "RiskLevel is the risk level.",
  },
  "bytebase.v1.Role": {
    type: "object",
    properties: [
      {
        name: "description",
        type: "string",
        description: "Optional description of the role.",
      },
      {
        name: "name",
        type: "string",
        description: "Resource name. Format: roles/{role}",
      },
      {
        name: "permissions",
        type: "array<string>",
        description: "List of permission identifiers granted by this role.",
      },
      {
        name: "title",
        type: "string",
        description: "Human-readable title.",
      },
      {
        name: "type",
        type: "bytebase.v1.Role.Type",
        description: "Role type indicating if it's built-in or custom.",
      },
    ],
    description:
      "Role defines a set of permissions that can be assigned to users.",
  },
  "bytebase.v1.Role.Type": {
    type: "enum",
    values: ["TYPE_UNSPECIFIED", "BUILT_IN", "CUSTOM"],
    description: "",
  },
  "bytebase.v1.RoleGrant": {
    type: "object",
    properties: [
      {
        name: "condition",
        type: "google.type.Expr",
        description:
          "The condition for the role. Same as the condition in IAM Binding message.",
      },
      {
        name: "expiration",
        type: "google.protobuf.Duration",
        description: "The duration for which the grant is valid.",
      },
      {
        name: "role",
        type: "string",
        description: "The requested role.\n Format: roles/EXPORTER.",
      },
      {
        name: "user",
        type: "string",
        description: "The user to be granted.\n Format: users/{email}.",
      },
    ],
    description: "",
  },
  "bytebase.v1.Rollout": {
    type: "object",
    properties: [
      {
        name: "createTime",
        type: "google.protobuf.Timestamp",
      },
      {
        name: "name",
        type: "string",
        description:
          "The resource name of the rollout.\n Format: projects/{project}/plans/{plan}/rollout",
      },
      {
        name: "stages",
        type: "array<bytebase.v1.Stage>",
        description: "Stages and thus tasks of the rollout.",
      },
      {
        name: "title",
        type: "string",
        description:
          "The title of the rollout, inherited from the associated plan.\n This field is output only and cannot be directly set.",
      },
      {
        name: "updateTime",
        type: "google.protobuf.Timestamp",
      },
    ],
    description: "",
  },
  "bytebase.v1.RolloutPolicy": {
    type: "object",
    properties: [
      {
        name: "automatic",
        type: "boolean",
        description: "Whether rollout is automatic without manual approval.",
      },
      {
        name: "roles",
        type: "array<string>",
        description: "The roles that can approve rollout execution.",
      },
    ],
    description: "Rollout policy configuration.",
  },
  "bytebase.v1.RowValue.Timestamp": {
    type: "object",
    properties: [
      {
        name: "accuracy",
        type: "integer",
        description:
          "The accuracy is the number of digits after the decimal point.",
      },
      {
        name: "googleTimestamp",
        type: "google.protobuf.Timestamp",
      },
    ],
    description: "",
  },
  "bytebase.v1.RowValue.TimestampTZ": {
    type: "object",
    properties: [
      {
        name: "accuracy",
        type: "integer",
      },
      {
        name: "googleTimestamp",
        type: "google.protobuf.Timestamp",
      },
      {
        name: "offset",
        type: "integer",
        description: "The offset is in seconds east of UTC",
      },
      {
        name: "zone",
        type: "string",
        description:
          'Zone is the time zone abbreviations in timezone database such as "PDT",\n "PST". https://en.wikipedia.org/wiki/List_of_tz_database_time_zones We\n retrieve the time zone information from the timestamptz field in the\n database. A timestamp is in UTC or epoch time, and with zone info, we can\n convert it to a local time string. Zone and offset are returned by\n time.Time.Zone()',
      },
    ],
    description: "",
  },
  "bytebase.v1.RunPlanChecksRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The plan to run plan checks.\n Format: projects/{project}/plans/{plan}",
      },
      {
        name: "specId",
        type: "string",
        description:
          "The UUID of the specific spec to run plan checks for.\n This should match the spec.id field in Plan.Spec.\n If not set, all specs in the plan will be used.",
      },
    ],
    description: "",
  },
  "bytebase.v1.SQLReviewRule": {
    type: "object",
    properties: [
      {
        name: "engine",
        type: "bytebase.v1.Engine",
        description: "The database engine this rule applies to.",
      },
      {
        name: "level",
        type: "bytebase.v1.SQLReviewRule.Level",
        description: "The severity level of the rule.",
      },
      {
        name: "type",
        type: "bytebase.v1.SQLReviewRule.Type",
        description: "The type of SQL review rule.",
      },
    ],
    description:
      "SQL review rule configuration. Check the SQL_REVIEW_RULES_DOCUMENTATION.md for details.",
  },
  "bytebase.v1.SQLReviewRule.CommentConventionRulePayload": {
    type: "object",
    properties: [
      {
        name: "maxLength",
        type: "integer",
      },
      {
        name: "required",
        type: "boolean",
      },
    ],
    description: "",
  },
  "bytebase.v1.SQLReviewRule.Level": {
    type: "enum",
    values: ["LEVEL_UNSPECIFIED", "ERROR", "WARNING"],
    description: "The severity level for SQL review rules.",
  },
  "bytebase.v1.SQLReviewRule.NamingCaseRulePayload": {
    type: "object",
    properties: [
      {
        name: "upper",
        type: "boolean",
      },
    ],
    description: "",
  },
  "bytebase.v1.SQLReviewRule.NamingRulePayload": {
    type: "object",
    properties: [
      {
        name: "format",
        type: "string",
      },
      {
        name: "maxLength",
        type: "integer",
      },
    ],
    description: "Payload message types for SQL review rules",
  },
  "bytebase.v1.SQLReviewRule.NumberRulePayload": {
    type: "object",
    properties: [
      {
        name: "number",
        type: "integer",
      },
    ],
    description: "",
  },
  "bytebase.v1.SQLReviewRule.StringArrayRulePayload": {
    type: "object",
    properties: [
      {
        name: "list",
        type: "array<string>",
      },
    ],
    description: "",
  },
  "bytebase.v1.SQLReviewRule.StringRulePayload": {
    type: "object",
    properties: [
      {
        name: "value",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.SQLReviewRule.Type": {
    type: "enum",
    values: [
      "TYPE_UNSPECIFIED",
      "ENGINE_MYSQL_USE_INNODB",
      "NAMING_FULLY_QUALIFIED",
      "NAMING_TABLE",
      "NAMING_COLUMN",
      "NAMING_INDEX_PK",
      "NAMING_INDEX_UK",
      "NAMING_INDEX_FK",
      "NAMING_INDEX_IDX",
      "NAMING_COLUMN_AUTO_INCREMENT",
      "NAMING_TABLE_NO_KEYWORD",
      "NAMING_IDENTIFIER_NO_KEYWORD",
      "NAMING_IDENTIFIER_CASE",
      "STATEMENT_SELECT_NO_SELECT_ALL",
      "STATEMENT_WHERE_REQUIRE_SELECT",
      "STATEMENT_WHERE_REQUIRE_UPDATE_DELETE",
      "STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE",
      "STATEMENT_DISALLOW_ON_DEL_CASCADE",
      "STATEMENT_DISALLOW_RM_TBL_CASCADE",
      "STATEMENT_DISALLOW_COMMIT",
      "STATEMENT_DISALLOW_LIMIT",
      "STATEMENT_DISALLOW_ORDER_BY",
      "STATEMENT_MERGE_ALTER_TABLE",
      "STATEMENT_INSERT_ROW_LIMIT",
      "STATEMENT_INSERT_MUST_SPECIFY_COLUMN",
      "STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND",
      "STATEMENT_AFFECTED_ROW_LIMIT",
      "STATEMENT_DML_DRY_RUN",
      "STATEMENT_DISALLOW_ADD_COLUMN_WITH_DEFAULT",
      "STATEMENT_ADD_CHECK_NOT_VALID",
      "STATEMENT_ADD_FOREIGN_KEY_NOT_VALID",
      "STATEMENT_DISALLOW_ADD_NOT_NULL",
      "STATEMENT_SELECT_FULL_TABLE_SCAN",
      "STATEMENT_CREATE_SPECIFY_SCHEMA",
      "STATEMENT_CHECK_SET_ROLE_VARIABLE",
      "STATEMENT_DISALLOW_USING_FILESORT",
      "STATEMENT_DISALLOW_USING_TEMPORARY",
      "STATEMENT_WHERE_NO_EQUAL_NULL",
      "STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS",
      "STATEMENT_QUERY_MINIMUM_PLAN_LEVEL",
      "STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT",
      "STATEMENT_MAXIMUM_LIMIT_VALUE",
      "STATEMENT_MAXIMUM_JOIN_TABLE_COUNT",
      "STATEMENT_MAXIMUM_STATEMENTS_IN_TRANSACTION",
      "STATEMENT_JOIN_STRICT_COLUMN_ATTRS",
      "STATEMENT_NON_TRANSACTIONAL",
      "STATEMENT_ADD_COLUMN_WITHOUT_POSITION",
      "STATEMENT_DISALLOW_OFFLINE_DDL",
      "STATEMENT_DISALLOW_CROSS_DB_QUERIES",
      "STATEMENT_MAX_EXECUTION_TIME",
      "STATEMENT_REQUIRE_ALGORITHM_OPTION",
      "STATEMENT_REQUIRE_LOCK_OPTION",
      "STATEMENT_OBJECT_OWNER_CHECK",
      "TABLE_REQUIRE_PK",
      "TABLE_NO_FOREIGN_KEY",
      "TABLE_DROP_NAMING_CONVENTION",
      "TABLE_COMMENT",
      "TABLE_DISALLOW_PARTITION",
      "TABLE_DISALLOW_TRIGGER",
      "TABLE_NO_DUPLICATE_INDEX",
      "TABLE_TEXT_FIELDS_TOTAL_LENGTH",
      "TABLE_DISALLOW_SET_CHARSET",
      "TABLE_DISALLOW_DDL",
      "TABLE_DISALLOW_DML",
      "TABLE_LIMIT_SIZE",
      "TABLE_REQUIRE_CHARSET",
      "TABLE_REQUIRE_COLLATION",
      "COLUMN_REQUIRED",
      "COLUMN_NO_NULL",
      "COLUMN_DISALLOW_CHANGE_TYPE",
      "COLUMN_SET_DEFAULT_FOR_NOT_NULL",
      "COLUMN_DISALLOW_CHANGE",
      "COLUMN_DISALLOW_CHANGING_ORDER",
      "COLUMN_DISALLOW_DROP",
      "COLUMN_DISALLOW_DROP_IN_INDEX",
      "COLUMN_COMMENT",
      "COLUMN_AUTO_INCREMENT_MUST_INTEGER",
      "COLUMN_TYPE_DISALLOW_LIST",
      "COLUMN_DISALLOW_SET_CHARSET",
      "COLUMN_MAXIMUM_CHARACTER_LENGTH",
      "COLUMN_MAXIMUM_VARCHAR_LENGTH",
      "COLUMN_AUTO_INCREMENT_INITIAL_VALUE",
      "COLUMN_AUTO_INCREMENT_MUST_UNSIGNED",
      "COLUMN_CURRENT_TIME_COUNT_LIMIT",
      "COLUMN_REQUIRE_DEFAULT",
      "COLUMN_DEFAULT_DISALLOW_VOLATILE",
      "COLUMN_ADD_NOT_NULL_REQUIRE_DEFAULT",
      "COLUMN_REQUIRE_CHARSET",
      "COLUMN_REQUIRE_COLLATION",
      "SCHEMA_BACKWARD_COMPATIBILITY",
      "DATABASE_DROP_EMPTY_DATABASE",
      "INDEX_NO_DUPLICATE_COLUMN",
      "INDEX_KEY_NUMBER_LIMIT",
      "INDEX_PK_TYPE_LIMIT",
      "INDEX_TYPE_NO_BLOB",
      "INDEX_TOTAL_NUMBER_LIMIT",
      "INDEX_PRIMARY_KEY_TYPE_ALLOWLIST",
      "INDEX_CREATE_CONCURRENTLY",
      "INDEX_TYPE_ALLOW_LIST",
      "INDEX_NOT_REDUNDANT",
      "SYSTEM_CHARSET_ALLOWLIST",
      "SYSTEM_COLLATION_ALLOWLIST",
      "SYSTEM_COMMENT_LENGTH",
      "SYSTEM_PROCEDURE_DISALLOW_CREATE",
      "SYSTEM_EVENT_DISALLOW_CREATE",
      "SYSTEM_VIEW_DISALLOW_CREATE",
      "SYSTEM_FUNCTION_DISALLOW_CREATE",
      "SYSTEM_FUNCTION_DISALLOWED_LIST",
      "ADVICE_ONLINE_MIGRATION",
      "BUILTIN_PRIOR_BACKUP_CHECK",
      "BUILTIN_WALK_THROUGH_CHECK",
    ],
    description: "",
  },
  "bytebase.v1.SchemaCatalog": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description: "The schema name.",
      },
      {
        name: "tables",
        type: "array<bytebase.v1.TableCatalog>",
        description: "The tables in the schema.",
      },
    ],
    description: "Schema metadata within a database.",
  },
  "bytebase.v1.SchemaMetadata": {
    type: "object",
    properties: [
      {
        name: "comment",
        type: "string",
        description: "The comment is the comment of a schema.",
      },
      {
        name: "enumTypes",
        type: "array<bytebase.v1.EnumTypeMetadata>",
        description:
          "The enum_types is the list of user-defined enum types in a schema.",
      },
      {
        name: "events",
        type: "array<bytebase.v1.EventMetadata>",
        description: "The events is the list of scheduled events in a schema.",
      },
      {
        name: "externalTables",
        type: "array<bytebase.v1.ExternalTableMetadata>",
        description:
          "The external_tables is the list of external tables in a schema.",
      },
      {
        name: "functions",
        type: "array<bytebase.v1.FunctionMetadata>",
        description: "The functions is the list of functions in a schema.",
      },
      {
        name: "materializedViews",
        type: "array<bytebase.v1.MaterializedViewMetadata>",
        description:
          "The materialized_views is the list of materialized views in a schema.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name is the schema name.\n It is an empty string for databases without such concept such as MySQL.",
      },
      {
        name: "owner",
        type: "string",
        description: "The owner of the schema.",
      },
      {
        name: "packages",
        type: "array<bytebase.v1.PackageMetadata>",
        description: "The packages is the list of packages in a schema.",
      },
      {
        name: "procedures",
        type: "array<bytebase.v1.ProcedureMetadata>",
        description: "The procedures is the list of procedures in a schema.",
      },
      {
        name: "sequences",
        type: "array<bytebase.v1.SequenceMetadata>",
        description:
          "The sequences is the list of sequences in a schema, sorted by name.",
      },
      {
        name: "skipDump",
        type: "boolean",
        description:
          "Whether to skip this schema during schema dump operations.",
      },
      {
        name: "streams",
        type: "array<bytebase.v1.StreamMetadata>",
        description:
          "The streams is the list of streams in a schema, currently, only used for\n Snowflake.",
      },
      {
        name: "tables",
        type: "array<bytebase.v1.TableMetadata>",
        description: "The tables is the list of tables in a schema.",
      },
      {
        name: "tasks",
        type: "array<bytebase.v1.TaskMetadata>",
        description:
          "The routines is the list of routines in a schema, currently, only used for\n Snowflake.",
      },
      {
        name: "views",
        type: "array<bytebase.v1.ViewMetadata>",
        description: "The views is the list of views in a schema.",
      },
    ],
    description:
      "SchemaMetadata is the metadata for schemas.\n This is the concept of schema in Postgres, but it's a no-op for MySQL.",
  },
  "bytebase.v1.SearchAuditLogsRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'The filter of the log. It should be a valid CEL expression.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Supported filter:\n - method: the API name, can be found in the docs, should start with "/bytebase.v1." prefix. For example "/bytebase.v1.UserService/CreateUser". Support "==" operator.\n - severity: support "==" operator, check Severity enum in AuditLog message for values.\n - user: the actor, should in "users/{email}" format, support "==" operator.\n - create_time: support ">=" and "<=" operator.\n\n For example:\n  - filter = "method == \'/bytebase.v1.SQLService/Query\'"\n  - filter = "method == \'/bytebase.v1.SQLService/Query\' && severity == \'ERROR\'"\n  - filter = "method == \'/bytebase.v1.SQLService/Query\' && severity == \'ERROR\' && user == \'users/bb@bytebase.com\'"\n  - filter = "method == \'/bytebase.v1.SQLService/Query\' && severity == \'ERROR\' && create_time <= \'2021-01-01T00:00:00Z\' && create_time >= \'2020-01-01T00:00:00Z\'"',
      },
      {
        name: "orderBy",
        type: "string",
        description:
          'The order by of the log.\n Only support order by create_time. The default sorting order is ascending.\n For example:\n  - order_by = "create_time asc"\n  - order_by = "create_time desc"',
      },
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of logs to return.\n The service may return fewer than this value.\n If unspecified, at most 10 log entries will be returned.\n The maximum value is 5000; values above 5000 will be coerced to 5000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `SearchLogs` call.\n Provide this to retrieve the subsequent page.",
      },
      {
        name: "parent",
        type: "string",
      },
    ],
    description: "Request message for searching audit logs.",
  },
  "bytebase.v1.SearchAuditLogsResponse": {
    type: "object",
    properties: [
      {
        name: "auditLogs",
        type: "array<bytebase.v1.AuditLog>",
      },
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token to retrieve next page of log entities.\n Pass this value in the page_token field in the subsequent call\n to retrieve the next page of log entities.",
      },
    ],
    description: "Response message for searching audit logs.",
  },
  "bytebase.v1.SearchIssuesRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          "Filter is used to filter issues returned in the list.\n Check the filter field in the ListIssuesRequest message.",
      },
      {
        name: "orderBy",
        type: "string",
        description:
          'The order by of issues.\n Support:\n - create_time\n - update_time\n The default sorting order is ascending.\n For example:\n - order_by = "create_time desc"\n - order_by = "update_time asc"',
      },
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of issues to return. The service may return fewer than\n this value.\n If unspecified, at most 10 issues will be returned.\n The maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `SearchIssues` call.\n Provide this to retrieve the subsequent page.\n\n When paginating, all other parameters provided to `SearchIssues` must match\n the call that provided the page token.",
      },
      {
        name: "parent",
        type: "string",
        description:
          'The parent, which owns this collection of issues.\n Format: projects/{project}\n Use "projects/-" to list all issues from all projects.',
      },
      {
        name: "query",
        type: "string",
        description: "Query is the query statement.",
      },
    ],
    description: "",
  },
  "bytebase.v1.SearchIssuesResponse": {
    type: "object",
    properties: [
      {
        name: "issues",
        type: "array<bytebase.v1.Issue>",
        description: "The issues from the specified request.",
      },
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
    ],
    description: "",
  },
  "bytebase.v1.SearchMyAccessGrantsRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          "Check the filter field in the ListAccessGrantsRequest message.",
      },
      {
        name: "orderBy",
        type: "string",
        description:
          "Check the order_by field in the ListAccessGrantsRequest message.",
      },
      {
        name: "pageSize",
        type: "integer",
        description: "The maximum number of access grants to return.",
      },
      {
        name: "pageToken",
        type: "string",
        description: "A page token from a previous SearchMyAccessGrants call.",
      },
      {
        name: "parent",
        type: "string",
        description:
          "The parent project to search in.\n Format: projects/{project}",
      },
    ],
    description: "",
  },
  "bytebase.v1.SearchMyAccessGrantsResponse": {
    type: "object",
    properties: [
      {
        name: "accessGrants",
        type: "array<bytebase.v1.AccessGrant>",
        description: "The access grants from the specified request.",
      },
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
    ],
    description: "",
  },
  "bytebase.v1.SearchProjectsRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'Filter the project.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Supported filters:\n - name: the project name, support "==" and ".contains()" operator.\n - resource_id: the project id, support "==" and ".contains()" operator.\n - exclude_default: if not include the default project, should be "true" or "false", support "==" operator.\n - state: check the State enum for the values, support "==" operator.\n - labels.{key}: the project label, support "==" and "in" operators.\n\n For example:\n name == "project name"\n name.contains("project name")\n resource_id == "project id"\n resource_id.contains("project id")\n exclude_default == true\n state == "DELETED"\n labels.environment == "production"\n labels.tier == "critical"\n labels.environment in ["staging", "prod"]\n You can combine filter conditions like:\n name == "project name" && resource_id.contains("project id")\n name.contains("project name") || resource_id == "project id"\n labels.environment == "production" && labels.tier == "critical"',
      },
      {
        name: "orderBy",
        type: "string",
        description:
          'The order by of projects.\n Support title. The default sorting order is ascending.\n For example:\n - order_by = "title"\n - order_by = "title desc"\n - order_by = "title asc"',
      },
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of projects to return. The service may return fewer than\n this value.\n If unspecified, at most 10 projects will be returned.\n The maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `SearchProjects` call.\n Provide this to retrieve the subsequent page.\n\n When paginating, all other parameters provided to `SearchProjects` must match\n the call that provided the page token.",
      },
      {
        name: "showDeleted",
        type: "boolean",
        description: "Show deleted projects if specified.",
      },
    ],
    description: "",
  },
  "bytebase.v1.SearchProjectsResponse": {
    type: "object",
    properties: [
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token, which can be sent as `page_token` to retrieve the next page.\n If this field is omitted, there are no subsequent pages.",
      },
      {
        name: "projects",
        type: "array<bytebase.v1.Project>",
        description: "The projects from the specified request.",
      },
    ],
    description: "",
  },
  "bytebase.v1.SearchQueryHistoriesRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'Filter is the filter to apply on the search query history\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Supported filter:\n - project: the project full name in "projects/{id}" format, support "==" operator.\n - database: the database full name in "instances/{id}/databases/{name}" format, support "==" operator.\n - instance: the instance full name in "instances/{id}" format, support "==" operator.\n - type: the type, should be "QUERY" or "EXPORT", support "==" operator.\n - statement: the SQL statement, support ".contains()" operator.\n\n For example:\n project == "projects/{project}"\n database == "instances/{instance}/databases/{database}"\n instance == "instances/{instance}"\n type == "QUERY"\n type == "EXPORT"\n statement.contains("select")\n type == "QUERY" && statement.contains("select")',
      },
      {
        name: "pageSize",
        type: "integer",
        description:
          "The maximum number of histories to return.\n The service may return fewer than this value.\n If unspecified, at most 10 history entries will be returned.\n The maximum value is 1000; values above 1000 will be coerced to 1000.",
      },
      {
        name: "pageToken",
        type: "string",
        description:
          "A page token, received from a previous `ListQueryHistory` call.\n Provide this to retrieve the subsequent page.",
      },
    ],
    description: "",
  },
  "bytebase.v1.SearchQueryHistoriesResponse": {
    type: "object",
    properties: [
      {
        name: "nextPageToken",
        type: "string",
        description:
          "A token to retrieve next page of history.\n Pass this value in the page_token field in the subsequent call to\n `ListQueryHistory` method to retrieve the next page of history.",
      },
      {
        name: "queryHistories",
        type: "array<bytebase.v1.QueryHistory>",
        description: "The list of history.",
      },
    ],
    description: "",
  },
  "bytebase.v1.SearchWorksheetsRequest": {
    type: "object",
    properties: [
      {
        name: "filter",
        type: "string",
        description:
          'To filter the search result.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n Supported filter:\n - creator: the worksheet creator in "users/{email}" format, support "==" and "!=" operator.\n - starred: should be "true" or "false", filter starred/unstarred sheets, support "==" operator.\n - visibility: check Visibility enum in the Worksheet message for values, support "==" and "in [xx]" operator.\n\n For example:\n creator == "users/{email}"\n creator != "users/{email}"\n starred == true\n starred == false\n visibility in ["PRIVATE", "PROJECT_READ", "PROJECT_WRITE"]\n visibility == "PRIVATE"',
      },
      {
        name: "parent",
        type: "string",
        description:
          "The parent resource of the worksheets.\n Format: projects/{project}",
      },
    ],
    description: "",
  },
  "bytebase.v1.SearchWorksheetsResponse": {
    type: "object",
    properties: [
      {
        name: "worksheets",
        type: "array<bytebase.v1.Worksheet>",
        description: "The worksheets that matched the search criteria.",
      },
    ],
    description: "",
  },
  "bytebase.v1.SemanticTypeSetting": {
    type: "object",
    properties: [
      {
        name: "types",
        type: "array<bytebase.v1.SemanticTypeSetting.SemanticType>",
      },
    ],
    description: "",
  },
  "bytebase.v1.SemanticTypeSetting.SemanticType": {
    type: "object",
    properties: [
      {
        name: "algorithm",
        type: "bytebase.v1.Algorithm",
      },
      {
        name: "description",
        type: "string",
        description: "the description of the semantic type, it can be empty.",
      },
      {
        name: "icon",
        type: "string",
        description:
          "icon is the icon for semantic type, it can be emoji or base64 encoded image.",
      },
      {
        name: "id",
        type: "string",
        description: "id is the uuid for semantic type.",
      },
      {
        name: "title",
        type: "string",
        description: "the title of the semantic type, it should not be empty.",
      },
    ],
    description: "",
  },
  "bytebase.v1.SequenceMetadata": {
    type: "object",
    properties: [
      {
        name: "cacheSize",
        type: "string",
        description: "Cache size of a sequence.",
      },
      {
        name: "comment",
        type: "string",
        description: "The comment describing the sequence.",
      },
      {
        name: "cycle",
        type: "boolean",
        description: "Cycle is whether the sequence cycles.",
      },
      {
        name: "dataType",
        type: "string",
        description: "The data type of a sequence.",
      },
      {
        name: "increment",
        type: "string",
        description: "Increment value of a sequence.",
      },
      {
        name: "lastValue",
        type: "string",
        description: "Last value of a sequence.",
      },
      {
        name: "maxValue",
        type: "string",
        description: "The maximum value of a sequence.",
      },
      {
        name: "minValue",
        type: "string",
        description: "The minimum value of a sequence.",
      },
      {
        name: "name",
        type: "string",
        description: "The name of a sequence.",
      },
      {
        name: "ownerColumn",
        type: "string",
        description: "The owner column of the sequence.",
      },
      {
        name: "ownerTable",
        type: "string",
        description: "The owner table of the sequence.",
      },
      {
        name: "skipDump",
        type: "boolean",
        description:
          "Whether to skip this sequence during schema dump operations.",
      },
      {
        name: "start",
        type: "string",
        description: "The start value of a sequence.",
      },
    ],
    description: "",
  },
  "bytebase.v1.ServiceAccount": {
    type: "object",
    properties: [
      {
        name: "createTime",
        type: "google.protobuf.Timestamp",
        description: "The timestamp when the service account was created.",
      },
      {
        name: "email",
        type: "string",
        description:
          "The email of the service account.\n For workspace-level: {name}@service.bytebase.com\n For project-level: {name}@{project-id}.service.bytebase.com",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the service account.\n Format: serviceAccounts/{email}",
      },
      {
        name: "serviceKey",
        type: "string",
        description:
          "The service key for authentication. Only returned on creation or key rotation.",
      },
      {
        name: "state",
        type: "bytebase.v1.State",
        description: "The state of the service account.",
      },
      {
        name: "title",
        type: "string",
        description: "The display title of the service account.",
      },
    ],
    description: "ServiceAccount represents an API integration account.",
  },
  "bytebase.v1.SetIamPolicyRequest": {
    type: "object",
    properties: [
      {
        name: "etag",
        type: "string",
        description: "The current etag of the policy.",
      },
      {
        name: "policy",
        type: "bytebase.v1.IamPolicy",
      },
      {
        name: "resource",
        type: "string",
        description:
          "The name of the resource to set the IAM policy.\n Format: projects/{project}\n Format: workspaces/{workspace}",
      },
    ],
    description: "Request message for setting an IAM policy.",
  },
  "bytebase.v1.Setting": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          'The resource name of the setting.\n Format: settings/{setting}\n Example: "settings/SEMANTIC_TYPES"',
      },
      {
        name: "value",
        type: "bytebase.v1.SettingValue",
        description: "The configuration value of the setting.",
      },
    ],
    description: "The schema of setting.",
  },
  "bytebase.v1.Setting.SettingName": {
    type: "enum",
    values: [
      "SETTING_NAME_UNSPECIFIED",
      "WORKSPACE_PROFILE",
      "WORKSPACE_APPROVAL",
      "APP_IM",
      "AI",
      "DATA_CLASSIFICATION",
      "SEMANTIC_TYPES",
      "ENVIRONMENT",
    ],
    description: "",
  },
  "bytebase.v1.Sheet": {
    type: "object",
    properties: [
      {
        name: "content",
        type: "string",
        description:
          "The content of the sheet.\n By default, it will be cut off, if it doesn't match the `content_size`, you can\n set the `raw` to true in GetSheet request to retrieve the full content.",
      },
      {
        name: "contentSize",
        type: "integer",
        description:
          "content_size is the full size of the content, may not match the size of the `content` field.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the sheet resource.\n Format: projects/{project}/sheets/{sheet}\n The sheet ID is generated by the server on creation and cannot be changed.",
      },
    ],
    description: "",
  },
  "bytebase.v1.SignupRequest": {
    type: "object",
    properties: [
      {
        name: "email",
        type: "string",
        description: "The email for the new account.",
      },
      {
        name: "password",
        type: "string",
        description: "The password for the new account.",
      },
      {
        name: "title",
        type: "string",
        description: "The display name of the user.",
      },
    ],
    description: "",
  },
  "bytebase.v1.SpatialIndexConfig": {
    type: "object",
    properties: [
      {
        name: "dimensional",
        type: "bytebase.v1.DimensionalConfig",
        description: "Dimensional configuration",
      },
      {
        name: "method",
        type: "string",
        description:
          'Spatial indexing method (e.g., "SPATIAL", "R-TREE", "GIST")',
      },
      {
        name: "storage",
        type: "bytebase.v1.StorageConfig",
        description: "Storage and performance configuration",
      },
      {
        name: "tessellation",
        type: "bytebase.v1.TessellationConfig",
        description:
          "Tessellation configuration for grid-based spatial indexes",
      },
    ],
    description:
      "SpatialIndexConfig defines the spatial index configuration for spatial databases.",
  },
  "bytebase.v1.Stage": {
    type: "object",
    properties: [
      {
        name: "environment",
        type: "string",
        description:
          'environment is the environment of the stage.\n Format: environments/{environment} for valid environments, or "environments/-" for stages without environment or with deleted environments.',
      },
      {
        name: "id",
        type: "string",
        description:
          'id is the environment id of the stage.\n e.g., "prod".\n Use "-" when the stage has no environment or deleted environment.',
      },
      {
        name: "name",
        type: "string",
        description:
          'Format: projects/{project}/plans/{plan}/rollout/stages/{stage}\n Use "-" for {stage} when the stage has no environment or deleted environment.',
      },
      {
        name: "tasks",
        type: "array<bytebase.v1.Task>",
        description: "The tasks within this stage.",
      },
    ],
    description: "",
  },
  "bytebase.v1.State": {
    type: "enum",
    values: ["STATE_UNSPECIFIED", "ACTIVE", "DELETED"],
    description: "Resource lifecycle state.",
  },
  "bytebase.v1.StatementType": {
    type: "enum",
    values: [
      "STATEMENT_TYPE_UNSPECIFIED",
      "CREATE_DATABASE",
      "CREATE_TABLE",
      "CREATE_VIEW",
      "CREATE_INDEX",
      "CREATE_SEQUENCE",
      "CREATE_SCHEMA",
      "CREATE_FUNCTION",
      "CREATE_TRIGGER",
      "CREATE_PROCEDURE",
      "CREATE_EVENT",
      "CREATE_EXTENSION",
      "CREATE_TYPE",
      "DROP_DATABASE",
      "DROP_TABLE",
      "DROP_VIEW",
      "DROP_INDEX",
      "DROP_SEQUENCE",
      "DROP_SCHEMA",
      "DROP_FUNCTION",
      "DROP_TRIGGER",
      "DROP_PROCEDURE",
      "DROP_EVENT",
      "DROP_EXTENSION",
      "DROP_TYPE",
      "ALTER_DATABASE",
      "ALTER_TABLE",
      "ALTER_VIEW",
      "ALTER_SEQUENCE",
      "ALTER_EVENT",
      "ALTER_TYPE",
      "ALTER_INDEX",
      "TRUNCATE",
      "RENAME",
      "RENAME_INDEX",
      "RENAME_SCHEMA",
      "RENAME_SEQUENCE",
      "COMMENT",
      "INSERT",
      "UPDATE",
      "DELETE",
    ],
    description: "StatementType represents the type of SQL statement.",
  },
  "bytebase.v1.StorageConfig": {
    type: "object",
    properties: [
      {
        name: "allowPageLocks",
        type: "boolean",
      },
      {
        name: "allowRowLocks",
        type: "boolean",
      },
      {
        name: "buffering",
        type: "string",
        description: "Buffering mode for PostgreSQL (auto, on, off)",
      },
      {
        name: "commitInterval",
        type: "integer",
      },
      {
        name: "dataCompression",
        type: "string",
        description: "NONE, ROW, PAGE",
      },
      {
        name: "dropExisting",
        type: "boolean",
      },
      {
        name: "fillfactor",
        type: "integer",
        description: "Fill factor percentage (1-100)",
      },
      {
        name: "maxdop",
        type: "integer",
      },
      {
        name: "online",
        type: "boolean",
      },
      {
        name: "padIndex",
        type: "boolean",
        description: "SQL Server specific parameters",
      },
      {
        name: "sdoLevel",
        type: "integer",
      },
      {
        name: "sortInTempdb",
        type: "string",
        description: "ON, OFF",
      },
      {
        name: "tablespace",
        type: "string",
        description: "Tablespace configuration for Oracle",
      },
      {
        name: "workTablespace",
        type: "string",
      },
    ],
    description:
      "StorageConfig defines storage and performance parameters for spatial indexes.",
  },
  "bytebase.v1.StreamMetadata": {
    type: "object",
    properties: [
      {
        name: "comment",
        type: "string",
        description: "The comment of the stream.",
      },
      {
        name: "definition",
        type: "string",
        description: "The definition of the stream.",
      },
      {
        name: "mode",
        type: "bytebase.v1.StreamMetadata.Mode",
        description: "The mode of the stream.",
      },
      {
        name: "name",
        type: "string",
        description: "The name is the name of a stream.",
      },
      {
        name: "owner",
        type: "string",
        description: "The owner of the stream.",
      },
      {
        name: "stale",
        type: "boolean",
        description:
          "Indicates whether the stream was last read before the `stale_after` time.",
      },
      {
        name: "tableName",
        type: "string",
        description:
          "The table_name is the name of the table/view that the stream is created on.",
      },
      {
        name: "type",
        type: "bytebase.v1.StreamMetadata.Type",
        description: "The type of the stream.",
      },
    ],
    description: "",
  },
  "bytebase.v1.StreamMetadata.Mode": {
    type: "enum",
    values: ["MODE_UNSPECIFIED", "DEFAULT", "APPEND_ONLY", "INSERT_ONLY"],
    description: "",
  },
  "bytebase.v1.StreamMetadata.Type": {
    type: "enum",
    values: ["TYPE_UNSPECIFIED", "DELTA"],
    description: "",
  },
  "bytebase.v1.Subscription": {
    type: "object",
    properties: [
      {
        name: "activeInstances",
        type: "integer",
      },
      {
        name: "etag",
        type: "string",
        description:
          "Etag for optimistic concurrency on purchase updates. Only set in SaaS mode.",
      },
      {
        name: "expiresTime",
        type: "google.protobuf.Timestamp",
      },
      {
        name: "ha",
        type: "boolean",
        description:
          "Whether high availability (multiple replicas) is enabled.",
      },
      {
        name: "instances",
        type: "integer",
      },
      {
        name: "orgName",
        type: "string",
      },
      {
        name: "plan",
        type: "bytebase.v1.PlanType",
      },
      {
        name: "seats",
        type: "integer",
      },
      {
        name: "trialing",
        type: "boolean",
      },
    ],
    description: "",
  },
  "bytebase.v1.SwitchWorkspaceRequest": {
    type: "object",
    properties: [
      {
        name: "mfaTempToken",
        type: "string",
        description:
          "Temporary MFA token from a previous SwitchWorkspace call that returned mfa_temp_token.",
      },
      {
        name: "otpCode",
        type: "string",
        description:
          "OTP code for MFA verification. Required if the target workspace enforces MFA.",
      },
      {
        name: "recoveryCode",
        type: "string",
        description:
          "Recovery code for MFA verification (alternative to otp_code).",
      },
      {
        name: "web",
        type: "boolean",
        description:
          "If true, sets tokens as HTTP-only cookies (browser clients).",
      },
      {
        name: "workspace",
        type: "string",
        description:
          "The target workspace to switch to.\n Format: workspaces/{workspace}",
      },
    ],
    description: "",
  },
  "bytebase.v1.SyncDatabaseRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the database to sync.\n Format: instances/{instance}/databases/{database}",
      },
    ],
    description: "",
  },
  "bytebase.v1.SyncInstanceRequest": {
    type: "object",
    properties: [
      {
        name: "enableFullSync",
        type: "boolean",
        description:
          "When full sync is enabled, all databases in the instance will be synchronized. Otherwise, only\n the instance metadata (such as the database list) and any newly discovered databases will be synced.",
      },
      {
        name: "name",
        type: "string",
        description: "The name of instance.\n Format: instances/{instance}",
      },
    ],
    description: "",
  },
  "bytebase.v1.SyncInstanceResponse": {
    type: "object",
    properties: [
      {
        name: "databases",
        type: "array<string>",
        description: "All database name list in the instance.",
      },
    ],
    description: "",
  },
  "bytebase.v1.SyncStatus": {
    type: "enum",
    values: ["SYNC_STATUS_UNSPECIFIED", "OK", "FAILED"],
    description: "SyncStatus is the status of the database sync operation.",
  },
  "bytebase.v1.TableCatalog": {
    type: "object",
    properties: [
      {
        name: "classification",
        type: "string",
        description: "The data classification level for this table.",
      },
      {
        name: "name",
        type: "string",
        description: "The table name.",
      },
    ],
    description: "Table metadata within a schema.",
  },
  "bytebase.v1.TableCatalog.Columns": {
    type: "object",
    properties: [
      {
        name: "columns",
        type: "array<bytebase.v1.ColumnCatalog>",
        description: "The columns in the table.",
      },
    ],
    description: "Column list for regular tables.",
  },
  "bytebase.v1.TableMetadata": {
    type: "object",
    properties: [
      {
        name: "charset",
        type: "string",
        description: "The character set of table.",
      },
      {
        name: "checkConstraints",
        type: "array<bytebase.v1.CheckConstraintMetadata>",
        description:
          "The check_constraints is the list of check constraints in a table.",
      },
      {
        name: "collation",
        type: "string",
        description: "The collation is the collation of a table.",
      },
      {
        name: "columns",
        type: "array<bytebase.v1.ColumnMetadata>",
        description: "The columns is the ordered list of columns in a table.",
      },
      {
        name: "comment",
        type: "string",
        description: "The comment is the comment of a table.",
      },
      {
        name: "createOptions",
        type: "string",
        description: "The create_options is the create option of a table.",
      },
      {
        name: "dataFree",
        type: "integer",
        description:
          "The data_free is the estimated free data size of a table.",
      },
      {
        name: "dataSize",
        type: "integer",
        description: "The data_size is the estimated data size of a table.",
      },
      {
        name: "engine",
        type: "string",
        description: "The engine is the engine of a table.",
      },
      {
        name: "foreignKeys",
        type: "array<bytebase.v1.ForeignKeyMetadata>",
        description: "The foreign_keys is the list of foreign keys in a table.",
      },
      {
        name: "indexes",
        type: "array<bytebase.v1.IndexMetadata>",
        description: "The indexes is the list of indexes in a table.",
      },
      {
        name: "indexSize",
        type: "integer",
        description: "The index_size is the estimated index size of a table.",
      },
      {
        name: "name",
        type: "string",
        description: "The name is the name of a table.",
      },
      {
        name: "owner",
        type: "string",
        description: "The owner of the table.",
      },
      {
        name: "partitions",
        type: "array<bytebase.v1.TablePartitionMetadata>",
        description: "The partitions is the list of partitions in a table.",
      },
      {
        name: "primaryKeyType",
        type: "string",
        description:
          "https://docs.pingcap.com/tidb/stable/clustered-indexes/#clustered-indexes\n CLUSTERED or NONCLUSTERED.",
      },
      {
        name: "rowCount",
        type: "integer",
        description:
          "The row_count is the estimated number of rows of a table.",
      },
      {
        name: "shardingInfo",
        type: "string",
        description:
          "https://docs.pingcap.com/tidb/stable/information-schema-tables/",
      },
      {
        name: "skipDump",
        type: "boolean",
        description:
          "Whether to skip this table during schema dump operations.",
      },
      {
        name: "sortingKeys",
        type: "array<string>",
        description:
          "The sorting_keys is a tuple of column names or arbitrary expressions. ClickHouse specific field.\n Reference: https://clickhouse.com/docs/en/engines/table-engines/mergetree-family/mergetree#order_by",
      },
      {
        name: "triggers",
        type: "array<bytebase.v1.TriggerMetadata>",
        description:
          "The triggers is the list of triggers associated with the table.",
      },
    ],
    description: "TableMetadata is the metadata for tables.",
  },
  "bytebase.v1.TablePartitionMetadata": {
    type: "object",
    properties: [
      {
        name: "checkConstraints",
        type: "array<bytebase.v1.CheckConstraintMetadata>",
      },
      {
        name: "expression",
        type: "string",
        description:
          "The expression is the expression of a table partition.\n For PostgreSQL, the expression is the text of {FOR VALUES\n partition_bound_spec}, see\n https://www.postgresql.org/docs/current/sql-createtable.html. For MySQL,\n the expression is the `expr` or `column_list` of the following syntax.\n PARTITION BY\n    { [LINEAR] HASH(expr)\n    | [LINEAR] KEY [ALGORITHM={1 | 2}] (column_list)\n    | RANGE{(expr) | COLUMNS(column_list)}\n    | LIST{(expr) | COLUMNS(column_list)} }.",
      },
      {
        name: "indexes",
        type: "array<bytebase.v1.IndexMetadata>",
      },
      {
        name: "name",
        type: "string",
        description: "The name is the name of a table partition.",
      },
      {
        name: "subpartitions",
        type: "array<bytebase.v1.TablePartitionMetadata>",
        description:
          "The subpartitions is the list of subpartitions in a table partition.",
      },
      {
        name: "type",
        type: "bytebase.v1.TablePartitionMetadata.Type",
        description: "The type of a table partition.",
      },
      {
        name: "useDefault",
        type: "string",
        description:
          "The use_default is whether the users use the default partition, it stores\n the different value for different database engines. For MySQL, it's [INT]\n type, 0 means not use default partition, otherwise, it's equals to number\n in syntax [SUB]PARTITION {number}.",
      },
      {
        name: "value",
        type: "string",
        description:
          "The value is the value of a table partition.\n For MySQL, the value is for RANGE and LIST partition types,\n - For a RANGE partition, it contains the value set in the partition's\n VALUES LESS THAN clause, which can be either an integer or MAXVALUE.\n - For a LIST partition, this column contains the values defined in the\n partition's VALUES IN clause, which is a list of comma-separated integer\n values.\n - For others, it's an empty string.",
      },
    ],
    description: "TablePartitionMetadata is the metadata for table partitions.",
  },
  "bytebase.v1.TablePartitionMetadata.Type": {
    type: "enum",
    values: [
      "TYPE_UNSPECIFIED",
      "RANGE",
      "RANGE_COLUMNS",
      "LIST",
      "LIST_COLUMNS",
      "HASH",
      "LINEAR_HASH",
      "KEY",
      "LINEAR_KEY",
    ],
    description:
      "Type is the type of a table partition, some database engines may not\n support all types. Only avilable for the following database engines now:\n MySQL: RANGE, RANGE COLUMNS, LIST, LIST COLUMNS, HASH, LINEAR HASH, KEY,\n LINEAR_KEY\n (https://dev.mysql.com/doc/refman/8.0/en/partitioning-types.html) TiDB:\n RANGE, RANGE COLUMNS, LIST, LIST COLUMNS, HASH, KEY PostgreSQL: RANGE,\n LIST, HASH (https://www.postgresql.org/docs/current/ddl-partitioning.html)",
  },
  "bytebase.v1.TagPolicy": {
    type: "object",
    properties: [
      {
        name: "tags",
        type: "object",
        description:
          'tags is the key - value map for resources.\n for example, the environment resource can have the sql review config tag, like "bb.tag.review_config": "reviewConfigs/{review config resource id}"',
      },
    ],
    description: "Policy for tagging resources with metadata.",
  },
  "bytebase.v1.TagPolicy.TagsEntry": {
    type: "object",
    properties: [
      {
        name: "key",
        type: "string",
      },
      {
        name: "value",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.Task": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}",
      },
      {
        name: "runTime",
        type: "object",
        description:
          "The run_time is the scheduled run time of latest task run.\n If there are no task runs or the task run is not scheduled, it will be empty.",
      },
      {
        name: "skippedReason",
        type: "string",
        description: "The reason why the task was skipped.",
      },
      {
        name: "specId",
        type: "string",
        description: "A UUID4 string that uniquely identifies the Spec.",
      },
      {
        name: "status",
        type: "bytebase.v1.Task.Status",
        description: "Status is the status of the task.",
      },
      {
        name: "target",
        type: "string",
        description:
          "Format: instances/{instance} if the task is DatabaseCreate.\n Format: instances/{instance}/databases/{database}",
      },
      {
        name: "type",
        type: "bytebase.v1.Task.Type",
      },
      {
        name: "updateTime",
        type: "object",
        description:
          "The update_time is the update time of latest task run.\n If there are no task runs, it will be empty.",
      },
    ],
    description: "",
  },
  "bytebase.v1.Task.DatabaseCreate": {
    type: "object",
    properties: [
      {
        name: "sheet",
        type: "string",
        description: "Format: projects/{project}/sheets/{sheet}",
      },
    ],
    description: "Payload for creating a new database.",
  },
  "bytebase.v1.Task.DatabaseDataExport": {
    type: "object",
    properties: [
      {
        name: "sheet",
        type: "string",
        description:
          "The resource name of the sheet.\n Format: projects/{project}/sheets/{sheet}",
      },
    ],
    description: "Payload for exporting database data.",
  },
  "bytebase.v1.Task.Status": {
    type: "enum",
    values: [
      "STATUS_UNSPECIFIED",
      "NOT_STARTED",
      "PENDING",
      "RUNNING",
      "DONE",
      "FAILED",
      "CANCELED",
      "SKIPPED",
    ],
    description: "",
  },
  "bytebase.v1.Task.Type": {
    type: "enum",
    values: [
      "TYPE_UNSPECIFIED",
      "GENERAL",
      "DATABASE_CREATE",
      "DATABASE_MIGRATE",
      "DATABASE_EXPORT",
    ],
    description: "",
  },
  "bytebase.v1.TaskMetadata": {
    type: "object",
    properties: [
      {
        name: "comment",
        type: "string",
        description: "The comment of the task.",
      },
      {
        name: "condition",
        type: "string",
        description: "The condition of the task.",
      },
      {
        name: "definition",
        type: "string",
        description: "The definition of the task.",
      },
      {
        name: "id",
        type: "string",
        description:
          "The id is the snowflake-generated id of a task.\n Example: 01ad32a0-1bb6-5e93-0000-000000000001",
      },
      {
        name: "name",
        type: "string",
        description: "The name is the name of a task.",
      },
      {
        name: "owner",
        type: "string",
        description: "The owner of the task.",
      },
      {
        name: "predecessors",
        type: "array<string>",
        description: "The predecessor tasks of the task.",
      },
      {
        name: "schedule",
        type: "string",
        description: "The schedule interval of the task.",
      },
      {
        name: "state",
        type: "bytebase.v1.TaskMetadata.State",
        description: "The state of the task.",
      },
      {
        name: "warehouse",
        type: "string",
        description: "The warehouse of the task.",
      },
    ],
    description: "",
  },
  "bytebase.v1.TaskMetadata.State": {
    type: "enum",
    values: ["STATE_UNSPECIFIED", "STARTED", "SUSPENDED"],
    description: "",
  },
  "bytebase.v1.TaskRun": {
    type: "object",
    properties: [
      {
        name: "createTime",
        type: "google.protobuf.Timestamp",
      },
      {
        name: "creator",
        type: "string",
        description: "Format: users/hello@world.com",
      },
      {
        name: "detail",
        type: "string",
        description:
          "Below are the results of a task run.\n Detailed information about the task run result.",
      },
      {
        name: "exportArchiveStatus",
        type: "bytebase.v1.TaskRun.ExportArchiveStatus",
        description: "The export archive status for data export tasks.",
      },
      {
        name: "hasPriorBackup",
        type: "boolean",
        description:
          "Indicates whether a prior backup was created for this task run.\n When true, rollback SQL can be generated via PreviewTaskRunRollback.\n Backup details are available in the task run logs.",
      },
      {
        name: "name",
        type: "string",
        description:
          "Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun}",
      },
      {
        name: "runTime",
        type: "object",
        description:
          "The task run should run after run_time.\n This can only be set when creating the task run calling BatchRunTasks.",
      },
      {
        name: "schedulerInfo",
        type: "bytebase.v1.TaskRun.SchedulerInfo",
        description: "Scheduling information about the task run.",
      },
      {
        name: "startTime",
        type: "google.protobuf.Timestamp",
        description: "The time when the task run started execution.",
      },
      {
        name: "status",
        type: "bytebase.v1.TaskRun.Status",
        description: "The current execution status of the task run.",
      },
      {
        name: "updateTime",
        type: "google.protobuf.Timestamp",
      },
    ],
    description: "",
  },
  "bytebase.v1.TaskRun.ExportArchiveStatus": {
    type: "enum",
    values: ["EXPORT_ARCHIVE_STATUS_UNSPECIFIED", "READY", "EXPORTED"],
    description: "",
  },
  "bytebase.v1.TaskRun.SchedulerInfo": {
    type: "object",
    properties: [
      {
        name: "reportTime",
        type: "google.protobuf.Timestamp",
        description: "The time when the scheduling info was reported.",
      },
      {
        name: "waitingCause",
        type: "bytebase.v1.TaskRun.SchedulerInfo.WaitingCause",
        description: "The cause for the task run waiting.",
      },
    ],
    description: "Information about task run scheduling.",
  },
  "bytebase.v1.TaskRun.Status": {
    type: "enum",
    values: [
      "STATUS_UNSPECIFIED",
      "PENDING",
      "RUNNING",
      "DONE",
      "FAILED",
      "CANCELED",
      "AVAILABLE",
    ],
    description: "",
  },
  "bytebase.v1.TaskRunLog": {
    type: "object",
    properties: [
      {
        name: "entries",
        type: "array<bytebase.v1.TaskRunLogEntry>",
        description: "The log entries for this task run.",
      },
      {
        name: "name",
        type: "string",
        description:
          "Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun}/log",
      },
    ],
    description: "",
  },
  "bytebase.v1.TaskRunLogEntry": {
    type: "object",
    properties: [
      {
        name: "commandExecute",
        type: "bytebase.v1.TaskRunLogEntry.CommandExecute",
        description: "Command execution details (if type is COMMAND_EXECUTE).",
      },
      {
        name: "computeDiff",
        type: "bytebase.v1.TaskRunLogEntry.ComputeDiff",
        description: "Compute diff details (if type is COMPUTE_DIFF).",
      },
      {
        name: "databaseSync",
        type: "bytebase.v1.TaskRunLogEntry.DatabaseSync",
        description: "Database sync details (if type is DATABASE_SYNC).",
      },
      {
        name: "logTime",
        type: "google.protobuf.Timestamp",
        description: "The time when the log was recorded.",
      },
      {
        name: "priorBackup",
        type: "bytebase.v1.TaskRunLogEntry.PriorBackup",
        description: "Prior backup details (if type is PRIOR_BACKUP).",
      },
      {
        name: "releaseFileExecute",
        type: "bytebase.v1.TaskRunLogEntry.ReleaseFileExecute",
        description:
          "Release file execution details (if type is RELEASE_FILE_EXECUTE).",
      },
      {
        name: "replicaId",
        type: "string",
        description: "The replica ID for this log entry.",
      },
      {
        name: "retryInfo",
        type: "bytebase.v1.TaskRunLogEntry.RetryInfo",
        description: "Retry information details (if type is RETRY_INFO).",
      },
      {
        name: "schemaDump",
        type: "bytebase.v1.TaskRunLogEntry.SchemaDump",
        description: "Schema dump details (if type is SCHEMA_DUMP).",
      },
      {
        name: "transactionControl",
        type: "bytebase.v1.TaskRunLogEntry.TransactionControl",
        description:
          "Transaction control details (if type is TRANSACTION_CONTROL).",
      },
      {
        name: "type",
        type: "bytebase.v1.TaskRunLogEntry.Type",
        description: "The type of this log entry.",
      },
    ],
    description: "",
  },
  "bytebase.v1.TaskRunLogEntry.CommandExecute": {
    type: "object",
    properties: [
      {
        name: "logTime",
        type: "google.protobuf.Timestamp",
        description: "When the command was logged.",
      },
      {
        name: "range",
        type: "bytebase.v1.Range",
        description:
          'The byte offset range of the executed command in the sheet.\n Uses byte offsets (not character indices) for efficient slicing of sheet content bytes.\n Example: For "SELECT 你好;" in a UTF-8 sheet, range [0, 13) represents all 13 bytes.',
      },
      {
        name: "response",
        type: "bytebase.v1.TaskRunLogEntry.CommandExecute.CommandResponse",
        description: "The response from executing the command.",
      },
      {
        name: "statement",
        type: "string",
        description: "The executed statement.",
      },
    ],
    description: "Command execution details.",
  },
  "bytebase.v1.TaskRunLogEntry.CommandExecute.CommandResponse": {
    type: "object",
    properties: [
      {
        name: "affectedRows",
        type: "integer",
        description: "Total affected rows.",
      },
      {
        name: "allAffectedRows",
        type: "array<integer>",
        description:
          "`all_affected_rows` is the affected rows of each command.\n `all_affected_rows` may be unavailable if the database driver doesn't support it. Caller should fallback to `affected_rows` in that case.",
      },
      {
        name: "error",
        type: "string",
        description: "Error message if command execution failed.",
      },
      {
        name: "logTime",
        type: "google.protobuf.Timestamp",
        description: "When the response was logged.",
      },
    ],
    description: "Command execution response.",
  },
  "bytebase.v1.TaskRunLogEntry.ComputeDiff": {
    type: "object",
    properties: [
      {
        name: "endTime",
        type: "google.protobuf.Timestamp",
        description: "When diff computation ended.",
      },
      {
        name: "error",
        type: "string",
        description: "Error message if computation failed.",
      },
      {
        name: "startTime",
        type: "google.protobuf.Timestamp",
        description: "When diff computation started.",
      },
    ],
    description: "Schema diff computation details.",
  },
  "bytebase.v1.TaskRunLogEntry.DatabaseSync": {
    type: "object",
    properties: [
      {
        name: "endTime",
        type: "google.protobuf.Timestamp",
        description: "When the database sync ended.",
      },
      {
        name: "error",
        type: "string",
        description: "Error message if sync failed.",
      },
      {
        name: "startTime",
        type: "google.protobuf.Timestamp",
        description: "When the database sync started.",
      },
    ],
    description: "Database synchronization details.",
  },
  "bytebase.v1.TaskRunLogEntry.PriorBackup": {
    type: "object",
    properties: [
      {
        name: "endTime",
        type: "google.protobuf.Timestamp",
        description: "When the backup ended.",
      },
      {
        name: "error",
        type: "string",
        description: "Error message if the backup failed.",
      },
      {
        name: "priorBackupDetail",
        type: "bytebase.v1.TaskRunLogEntry.PriorBackup.PriorBackupDetail",
        description: "The backup details.",
      },
      {
        name: "startTime",
        type: "google.protobuf.Timestamp",
        description: "When the backup started.",
      },
    ],
    description: "Prior backup operation details.",
  },
  "bytebase.v1.TaskRunLogEntry.PriorBackup.PriorBackupDetail": {
    type: "object",
    properties: [
      {
        name: "items",
        type: "array<bytebase.v1.TaskRunLogEntry.PriorBackup.PriorBackupDetail.Item>",
        description: "The list of backed up tables.",
      },
    ],
    description: "Prior backup detail for rollback purposes.",
  },
  "bytebase.v1.TaskRunLogEntry.PriorBackup.PriorBackupDetail.Item": {
    type: "object",
    properties: [
      {
        name: "endPosition",
        type: "bytebase.v1.Position",
        description: "The end position in the SQL statement.",
      },
      {
        name: "sourceTable",
        type: "bytebase.v1.TaskRunLogEntry.PriorBackup.PriorBackupDetail.Item.Table",
        description: "The original table information.",
      },
      {
        name: "startPosition",
        type: "bytebase.v1.Position",
        description: "The start position in the SQL statement.",
      },
      {
        name: "targetTable",
        type: "bytebase.v1.TaskRunLogEntry.PriorBackup.PriorBackupDetail.Item.Table",
        description: "The target backup table information.",
      },
    ],
    description: "A single backup table mapping.",
  },
  "bytebase.v1.TaskRunLogEntry.PriorBackup.PriorBackupDetail.Item.Table": {
    type: "object",
    properties: [
      {
        name: "database",
        type: "string",
        description:
          "The database information.\n Format: instances/{instance}/databases/{database}",
      },
      {
        name: "schema",
        type: "string",
        description: "The schema name.",
      },
      {
        name: "table",
        type: "string",
        description: "The table name.",
      },
    ],
    description: "Table information.",
  },
  "bytebase.v1.TaskRunLogEntry.ReleaseFileExecute": {
    type: "object",
    properties: [
      {
        name: "filePath",
        type: "string",
        description:
          'The file path within the release (e.g., "2.2/V0001_create_table.sql").',
      },
      {
        name: "version",
        type: "string",
        description: 'The version of the file being executed (e.g., "0001").',
      },
    ],
    description: "Release file execution details.",
  },
  "bytebase.v1.TaskRunLogEntry.RetryInfo": {
    type: "object",
    properties: [
      {
        name: "error",
        type: "string",
        description: "The error that triggered the retry.",
      },
      {
        name: "maximumRetries",
        type: "integer",
        description: "Maximum number of retries allowed.",
      },
      {
        name: "retryCount",
        type: "integer",
        description: "Current retry attempt number.",
      },
    ],
    description: "Retry information for failed operations.",
  },
  "bytebase.v1.TaskRunLogEntry.SchemaDump": {
    type: "object",
    properties: [
      {
        name: "endTime",
        type: "google.protobuf.Timestamp",
        description: "When the schema dump ended.",
      },
      {
        name: "error",
        type: "string",
        description: "Error message if the schema dump failed.",
      },
      {
        name: "startTime",
        type: "google.protobuf.Timestamp",
        description: "When the schema dump started.",
      },
    ],
    description: "Schema dump operation details.",
  },
  "bytebase.v1.TaskRunLogEntry.TransactionControl": {
    type: "object",
    properties: [
      {
        name: "error",
        type: "string",
        description: "Error message if the operation failed.",
      },
      {
        name: "type",
        type: "bytebase.v1.TaskRunLogEntry.TransactionControl.Type",
        description: "The type of transaction control.",
      },
    ],
    description: "Transaction control operation details.",
  },
  "bytebase.v1.TaskRunLogEntry.TransactionControl.Type": {
    type: "enum",
    values: ["TYPE_UNSPECIFIED", "BEGIN", "COMMIT", "ROLLBACK"],
    description: "Transaction control type.",
  },
  "bytebase.v1.TaskRunLogEntry.Type": {
    type: "enum",
    values: [
      "TYPE_UNSPECIFIED",
      "SCHEMA_DUMP",
      "COMMAND_EXECUTE",
      "DATABASE_SYNC",
      "TRANSACTION_CONTROL",
      "PRIOR_BACKUP",
      "RETRY_INFO",
      "COMPUTE_DIFF",
      "RELEASE_FILE_EXECUTE",
    ],
    description: "The type of log entry.",
  },
  "bytebase.v1.TaskRunSession": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "Format: projects/{project}/plans/{plan}/rollout/stages/{stage}/tasks/{task}/taskRuns/{taskRun}/session",
      },
    ],
    description: "",
  },
  "bytebase.v1.TaskRunSession.Postgres": {
    type: "object",
    properties: [
      {
        name: "blockedSessions",
        type: "array<bytebase.v1.TaskRunSession.Postgres.Session>",
        description: "`blocked_sessions` are blocked by `session`.",
      },
      {
        name: "blockingSessions",
        type: "array<bytebase.v1.TaskRunSession.Postgres.Session>",
        description: "`blocking_sessions` block `session`.",
      },
      {
        name: "session",
        type: "bytebase.v1.TaskRunSession.Postgres.Session",
        description:
          "`session` is the session of the task run executing commands.",
      },
    ],
    description: "PostgreSQL session information.",
  },
  "bytebase.v1.TaskRunSession.Postgres.Session": {
    type: "object",
    properties: [
      {
        name: "applicationName",
        type: "string",
        description: "Application name.",
      },
      {
        name: "backendStart",
        type: "google.protobuf.Timestamp",
        description: "When the backend process started.",
      },
      {
        name: "blockedByPids",
        type: "array<string>",
        description: "PIDs of sessions blocking this session.",
      },
      {
        name: "clientAddr",
        type: "string",
        description: "Client IP address.",
      },
      {
        name: "clientPort",
        type: "string",
        description: "Client port number.",
      },
      {
        name: "datname",
        type: "string",
        description: "Database name.",
      },
      {
        name: "pid",
        type: "string",
        description: "Process ID of the session.",
      },
      {
        name: "query",
        type: "string",
        description: "Current query being executed.",
      },
      {
        name: "queryStart",
        type: "object",
        description: "When the current query started.",
      },
      {
        name: "state",
        type: "string",
        description: "Session state (active, idle, etc.).",
      },
      {
        name: "usename",
        type: "string",
        description: "User name.",
      },
      {
        name: "waitEvent",
        type: "string",
        description: "Specific wait event if session is waiting.",
      },
      {
        name: "waitEventType",
        type: "string",
        description: "Wait event type if session is waiting.",
      },
      {
        name: "xactStart",
        type: "object",
        description: "When the current transaction started.",
      },
    ],
    description: "PostgreSQL session information read from `pg_stat_activity`.",
  },
  "bytebase.v1.TessellationConfig": {
    type: "object",
    properties: [
      {
        name: "boundingBox",
        type: "bytebase.v1.BoundingBox",
        description:
          "Bounding box for GEOMETRY tessellation (not used for GEOGRAPHY)",
      },
      {
        name: "cellsPerObject",
        type: "integer",
        description: "Number of cells per object (1-8192 for SQL Server)",
      },
      {
        name: "gridLevels",
        type: "array<bytebase.v1.GridLevel>",
        description: "Grid levels and densities for multi-level tessellation",
      },
      {
        name: "scheme",
        type: "string",
        description:
          'Tessellation scheme (e.g., "GEOMETRY_GRID", "GEOGRAPHY_GRID", "GEOMETRY_AUTO_GRID")',
      },
    ],
    description:
      "TessellationConfig defines tessellation parameters for spatial indexes.",
  },
  "bytebase.v1.TestIdentityProviderRequest": {
    type: "object",
    properties: [
      {
        name: "identityProvider",
        type: "bytebase.v1.IdentityProvider",
        description:
          "The identity provider to test connection including uncreated.",
      },
    ],
    description: "",
  },
  "bytebase.v1.TestIdentityProviderResponse": {
    type: "object",
    properties: [
      {
        name: "claims",
        type: "object",
        description: "The map of claims returned by the identity provider.",
      },
      {
        name: "userInfo",
        type: "object",
        description: "The matched user info from the claims.",
      },
    ],
    description: "",
  },
  "bytebase.v1.TestIdentityProviderResponse.ClaimsEntry": {
    type: "object",
    properties: [
      {
        name: "key",
        type: "string",
      },
      {
        name: "value",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.TestIdentityProviderResponse.UserInfoEntry": {
    type: "object",
    properties: [
      {
        name: "key",
        type: "string",
      },
      {
        name: "value",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.TestWebhookRequest": {
    type: "object",
    properties: [
      {
        name: "project",
        type: "string",
        description:
          "The name of the project which owns the webhook to test.\n Format: projects/{project}",
      },
      {
        name: "webhook",
        type: "bytebase.v1.Webhook",
        description: "The webhook to test. Identified by its url.",
      },
    ],
    description: "",
  },
  "bytebase.v1.TestWebhookResponse": {
    type: "object",
    properties: [
      {
        name: "error",
        type: "string",
        description: "The result of the test, empty if the test is successful.",
      },
    ],
    description: "",
  },
  "bytebase.v1.TriggerMetadata": {
    type: "object",
    properties: [
      {
        name: "body",
        type: "string",
        description: "The body is the body of the trigger.",
      },
      {
        name: "characterSetClient",
        type: "string",
        description:
          "The character set used by the client creating the trigger.",
      },
      {
        name: "collationConnection",
        type: "string",
        description:
          "The collation used for the connection when creating the trigger.",
      },
      {
        name: "comment",
        type: "string",
        description: "The comment describing the trigger.",
      },
      {
        name: "event",
        type: "string",
        description:
          "The event is the event of the trigger, such as INSERT, UPDATE, DELETE,\n TRUNCATE.",
      },
      {
        name: "name",
        type: "string",
        description: "The name is the name of the trigger.",
      },
      {
        name: "skipDump",
        type: "boolean",
        description:
          "Whether to skip this trigger during schema dump operations.",
      },
      {
        name: "sqlMode",
        type: "string",
        description: "The SQL mode setting for the trigger.",
      },
      {
        name: "timing",
        type: "string",
        description:
          "The timing is the timing of the trigger, such as BEFORE, AFTER.",
      },
    ],
    description: "",
  },
  "bytebase.v1.UndeleteInstanceRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the deleted instance.\n Format: instances/{instance}",
      },
    ],
    description: "",
  },
  "bytebase.v1.UndeleteProjectRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the deleted project.\n Format: projects/{project}",
      },
    ],
    description: "",
  },
  "bytebase.v1.UndeleteReleaseRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the deleted release.\n Format: projects/{project}/releases/{release}",
      },
    ],
    description: "",
  },
  "bytebase.v1.UndeleteServiceAccountRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the deleted service account.\n Format: serviceAccounts/{email}",
      },
    ],
    description: "Request message for restoring a deleted service account.",
  },
  "bytebase.v1.UndeleteUserRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description: "The name of the deleted user.\n Format: users/{email}",
      },
    ],
    description: "",
  },
  "bytebase.v1.UndeleteWorkloadIdentityRequest": {
    type: "object",
    properties: [
      {
        name: "name",
        type: "string",
        description:
          "The name of the deleted workload identity.\n Format: workloadIdentities/{email}",
      },
    ],
    description: "Request message for restoring a deleted workload identity.",
  },
  "bytebase.v1.UpdateDataSourceRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
        description:
          "If set to true, and the data source is not found, a new data source will be created.\n In this situation, `update_mask` is ignored.",
      },
      {
        name: "dataSource",
        type: "bytebase.v1.DataSource",
        description: "Identified by data source ID.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the instance to update a data source.\n Format: instances/{instance}",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description: "The list of fields to update.",
      },
      {
        name: "validateOnly",
        type: "boolean",
        description: "Validate only also tests the data source connection.",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdateDatabaseCatalogRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
        description:
          "If set to true, and the database catalog is not found, a new database catalog will be created.\n In this situation, `update_mask` is ignored.",
      },
      {
        name: "catalog",
        type: "bytebase.v1.DatabaseCatalog",
        description:
          "The database catalog to update.\n\n The catalog's `name` field is used to identify the database catalog to update.\n Format: instances/{instance}/databases/{database}/catalog",
      },
    ],
    description: "Request message for updating a database catalog.",
  },
  "bytebase.v1.UpdateDatabaseGroupRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
        description:
          "If set to true, and the database group is not found, a new database group will be created.\n In this situation, `update_mask` is ignored.",
      },
      {
        name: "databaseGroup",
        type: "bytebase.v1.DatabaseGroup",
        description:
          "The database group to update.\n\n The database group's `name` field is used to identify the database group to update.\n Format: projects/{project}/databaseGroups/{databaseGroup}",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description: "The list of fields to update.",
      },
    ],
    description: "Request message for updating a database group.",
  },
  "bytebase.v1.UpdateDatabaseRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
        description:
          "If set to true, and the database is not found, a new database will be created.\n In this situation, `update_mask` is ignored.",
      },
      {
        name: "database",
        type: "bytebase.v1.Database",
        description:
          "The database to update.\n\n The database's `name` field is used to identify the database to update.\n Format: instances/{instance}/databases/{database}",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description: "The list of fields to update.",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdateEmailRequest": {
    type: "object",
    properties: [
      {
        name: "email",
        type: "string",
        description: "The new email address.",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the user whose email to update.\n Format: users/{email}\n Note: This is the current (old) email address. The new email is specified in the 'email' field.",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdateGroupRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
        description:
          "If set to true, and the group is not found, a new group will be created.\n In this situation, `update_mask` is ignored.",
      },
      {
        name: "group",
        type: "bytebase.v1.Group",
        description:
          "The group to update.\n\n The group's `name` field is used to identify the group to update.\n Format: groups/{email}",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description: "The list of fields to update.",
      },
    ],
    description: "Request message for updating a group.",
  },
  "bytebase.v1.UpdateIdentityProviderRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
        description:
          "If set to true, and the identity provider is not found, a new identity provider will be created.\n In this situation, `update_mask` is ignored.",
      },
      {
        name: "identityProvider",
        type: "bytebase.v1.IdentityProvider",
        description:
          "The identity provider to update.\n\n The identity provider's `name` field is used to identify the identity provider to update.\n Format: idps/{identity_provider}",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description: "The list of fields to update.",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdateInstanceRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
        description:
          "If set to true, and the instance is not found, a new instance will be created.\n In this situation, `update_mask` is ignored.",
      },
      {
        name: "instance",
        type: "bytebase.v1.Instance",
        description:
          "The instance to update.\n\n The instance's `name` field is used to identify the instance to update.\n Format: instances/{instance}",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description: "The list of fields to update.",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdateIssueCommentRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
        description:
          "If set to true, and the issue comment is not found, a new issue comment will be created.\n In this situation, `update_mask` is ignored.",
      },
      {
        name: "issueComment",
        type: "bytebase.v1.IssueComment",
        description: "The comment to update.",
      },
      {
        name: "parent",
        type: "string",
        description:
          "The issue name\n Format: projects/{project}/issues/{issue}",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description: "The list of fields to update.",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdateIssueRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
        description:
          "If set to true, and the issue is not found, a new issue will be created.\n In this situation, `update_mask` is ignored.",
      },
      {
        name: "issue",
        type: "bytebase.v1.Issue",
        description:
          "The issue to update.\n\n The issue's `name` field is used to identify the issue to update.\n Format: projects/{project}/issues/{issue}",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description: "The list of fields to update.",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdatePlanRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
        description:
          "If set to true, and the plan is not found, a new plan will be created.\n In this situation, `update_mask` is ignored.",
      },
      {
        name: "plan",
        type: "bytebase.v1.Plan",
        description:
          "The plan to update.\n\n The plan's `name` field is used to identify the plan to update.\n Format: projects/{project}/plans/{plan}",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description: "The list of fields to update.",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdatePolicyRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
        description:
          "If set to true, and the policy is not found, a new policy will be created.\n In this situation, `update_mask` is ignored.",
      },
      {
        name: "policy",
        type: "bytebase.v1.Policy",
        description:
          "The policy to update.\n\n The policy's `name` field is used to identify the instance to update.\n Format: {resource name}/policies/{policy type}\n Workspace resource name: workspaces/{workspace-id}.\n Environment resource name: environments/environment-id.\n Instance resource name: instances/instance-id.\n Database resource name: instances/instance-id/databases/database-name.",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description: "The list of fields to update.",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdateProjectRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
        description:
          "If set to true, and the project is not found, a new project will be created.\n In this situation, `update_mask` is ignored.",
      },
      {
        name: "project",
        type: "bytebase.v1.Project",
        description:
          "The project to update.\n\n The project's `name` field is used to identify the project to update.\n Format: projects/{project}",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description: "The list of fields to update.",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdatePurchaseRequest": {
    type: "object",
    properties: [
      {
        name: "etag",
        type: "string",
      },
      {
        name: "interval",
        type: "bytebase.v1.BillingInterval",
      },
      {
        name: "plan",
        type: "bytebase.v1.PlanType",
      },
      {
        name: "seats",
        type: "integer",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdateReleaseRequest": {
    type: "object",
    properties: [
      {
        name: "release",
        type: "bytebase.v1.Release",
        description: "The release to update.",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description: "The list of fields to be updated.",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdateReviewConfigRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
        description:
          "If set to true, and the config is not found, a new config will be created.\n In this situation, `update_mask` is ignored.",
      },
      {
        name: "reviewConfig",
        type: "bytebase.v1.ReviewConfig",
        description:
          "The SQL review config to update.\n\n The name field is used to identify the SQL review config to update.",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description: "The list of fields to update.",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdateRoleRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
        description:
          "If set to true, and the role is not found, a new role will be created.\n In this situation, `update_mask` is ignored.",
      },
      {
        name: "role",
        type: "bytebase.v1.Role",
        description: "The role to update.",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description: "The list of fields to update.",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdateServiceAccountRequest": {
    type: "object",
    properties: [
      {
        name: "serviceAccount",
        type: "bytebase.v1.ServiceAccount",
        description:
          "The service account to update.\n\n The service account's `name` field is used to identify the service account to update.\n Format: serviceAccounts/{email}",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description:
          "The list of fields to update.\n Supported fields: title, service_key (triggers key rotation)",
      },
    ],
    description: "Request message for updating a service account.",
  },
  "bytebase.v1.UpdateSettingRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
      },
      {
        name: "setting",
        type: "bytebase.v1.Setting",
        description: "The setting to update.",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
      },
      {
        name: "validateOnly",
        type: "boolean",
        description:
          "validate_only is a flag to indicate whether to validate the setting value,\n server would not persist the setting value if it is true.",
      },
    ],
    description: "The request message for updating or creating a setting.",
  },
  "bytebase.v1.UpdateUserRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
        description:
          "If set to true, and the user is not found, a new user will be created.\n In this situation, `update_mask` is ignored.",
      },
      {
        name: "otpCode",
        type: "string",
        description:
          "The otp_code is used to verify the user's identity by MFA.",
      },
      {
        name: "regenerateRecoveryCodes",
        type: "boolean",
        description:
          "The regenerate_recovery_codes flag means to regenerate recovery codes for user.",
      },
      {
        name: "regenerateTempMfaSecret",
        type: "boolean",
        description:
          "The regenerate_temp_mfa_secret flag means to regenerate temporary MFA secret for user.\n This is used for MFA setup. The temporary MFA secret and recovery codes will be returned in the response.",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description: "The list of fields to update.",
      },
      {
        name: "user",
        type: "bytebase.v1.User",
        description:
          "The user to update.\n\n The user's `name` field is used to identify the user to update.\n Format: users/{email}",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdateWebhookRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
        description:
          "If set to true, and the webhook is not found, a new webhook will be created.\n In this situation, `update_mask` is ignored.",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description: "The list of fields to update.",
      },
      {
        name: "webhook",
        type: "bytebase.v1.Webhook",
        description: "The webhook to modify.",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdateWorkloadIdentityRequest": {
    type: "object",
    properties: [
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description:
          "The list of fields to update.\n Supported fields: title, workload_identity_config",
      },
      {
        name: "workloadIdentity",
        type: "bytebase.v1.WorkloadIdentity",
        description:
          "The workload identity to update.\n\n The workload identity's `name` field is used to identify the workload identity to update.\n Format: workloadIdentities/{email}",
      },
    ],
    description: "Request message for updating a workload identity.",
  },
  "bytebase.v1.UpdateWorksheetOrganizerRequest": {
    type: "object",
    properties: [
      {
        name: "allowMissing",
        type: "boolean",
        description:
          "If set to true, and the worksheet organizer is not found, a new worksheet organizer will be created.\n In this situation, `update_mask` is ignored.",
      },
      {
        name: "organizer",
        type: "bytebase.v1.WorksheetOrganizer",
        description:
          "The organizer to update.\n\n The organizer's `worksheet` field is used to identify the worksheet.\n Format: projects/{project}/worksheets/{worksheet}",
      },
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description:
          "The list of fields to be updated.\n Fields are specified relative to the worksheet organizer.\n Only support update the following fields for now:\n - `starred`",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdateWorksheetRequest": {
    type: "object",
    properties: [
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
        description:
          "The list of fields to be updated.\n Fields are specified relative to the worksheet.\n (e.g., `title`, `statement`; *not* `worksheet.title` or `worksheet.statement`)\n Only support update the following fields for now:\n - `title`\n - `statement`\n - `starred`\n - `visibility`",
      },
      {
        name: "worksheet",
        type: "bytebase.v1.Worksheet",
        description:
          "The worksheet to update.\n\n The worksheet's `name` field is used to identify the worksheet to update.\n Format: projects/{project}/worksheets/{worksheet}",
      },
    ],
    description: "",
  },
  "bytebase.v1.UpdateWorkspaceRequest": {
    type: "object",
    properties: [
      {
        name: "updateMask",
        type: "google.protobuf.FieldMask",
      },
      {
        name: "workspace",
        type: "bytebase.v1.Workspace",
      },
    ],
    description: "",
  },
  "bytebase.v1.UploadLicenseRequest": {
    type: "object",
    properties: [
      {
        name: "license",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.User": {
    type: "object",
    properties: [
      {
        name: "email",
        type: "string",
        description:
          "The email address of the user, used for login and notifications.",
      },
      {
        name: "groups",
        type: "array<string>",
        description: "The groups for the user.\n Format: groups/{email}",
      },
      {
        name: "mfaEnabled",
        type: "boolean",
        description: "The mfa_enabled flag means if the user has enabled MFA.",
      },
      {
        name: "name",
        type: "string",
        description: "The name of the user.\n Format: users/{email}",
      },
      {
        name: "password",
        type: "string",
        description:
          "The password for authentication. Only used during user creation or password updates.",
      },
      {
        name: "phone",
        type: "string",
        description:
          "Should be a valid E.164 compliant phone number.\n Could be empty.",
      },
      {
        name: "profile",
        type: "bytebase.v1.User.Profile",
        description: "User profile metadata.",
      },
      {
        name: "serviceKey",
        type: "string",
        description: "The service key for service account authentication.",
      },
      {
        name: "state",
        type: "bytebase.v1.State",
        description: "The lifecycle state of the user account.",
      },
      {
        name: "tempOtpSecret",
        type: "string",
        description:
          "Temporary OTP secret used during MFA setup and regeneration.",
      },
      {
        name: "tempOtpSecretCreatedTime",
        type: "google.protobuf.Timestamp",
        description:
          "Timestamp when temp_otp_secret was created. Used by frontend to show countdown timer.",
      },
      {
        name: "tempRecoveryCodes",
        type: "array<string>",
        description:
          "Temporary recovery codes used during MFA setup and regeneration.",
      },
      {
        name: "title",
        type: "string",
        description: "The display title or full name of the user.",
      },
      {
        name: "workspace",
        type: "string",
        description: "The current workspace.\n Format: workspaces/{id}",
      },
    ],
    description: "",
  },
  "bytebase.v1.User.Profile": {
    type: "object",
    properties: [
      {
        name: "lastChangePasswordTime",
        type: "google.protobuf.Timestamp",
        description: "The last time the user changed their password.",
      },
      {
        name: "lastLoginTime",
        type: "google.protobuf.Timestamp",
        description: "The last time the user successfully logged in.",
      },
      {
        name: "source",
        type: "string",
        description:
          "source means where the user comes from. For now we support Entra ID SCIM sync, so the source could be Entra ID.",
      },
    ],
    description: "",
  },
  "bytebase.v1.VCSType": {
    type: "enum",
    values: [
      "VCS_TYPE_UNSPECIFIED",
      "GITHUB",
      "GITLAB",
      "BITBUCKET",
      "AZURE_DEVOPS",
    ],
    description: "Version control system type.",
  },
  "bytebase.v1.VerifyCheckoutSessionRequest": {
    type: "object",
    properties: [
      {
        name: "sessionId",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.VerifyCheckoutSessionResponse": {
    type: "object",
    properties: [
      {
        name: "status",
        type: "string",
        description:
          'Stripe Checkout Session status: "complete", "expired", or "open".',
      },
    ],
    description: "",
  },
  "bytebase.v1.ViewMetadata": {
    type: "object",
    properties: [
      {
        name: "columns",
        type: "array<bytebase.v1.ColumnMetadata>",
        description: "The columns is the ordered list of columns in a table.",
      },
      {
        name: "comment",
        type: "string",
        description: "The comment is the comment of a view.",
      },
      {
        name: "definition",
        type: "string",
        description: "The definition is the definition of a view.",
      },
      {
        name: "dependencyColumns",
        type: "array<bytebase.v1.DependencyColumn>",
        description:
          "The dependency_columns is the list of dependency columns of a view.",
      },
      {
        name: "name",
        type: "string",
        description: "The name is the name of a view.",
      },
      {
        name: "skipDump",
        type: "boolean",
      },
      {
        name: "triggers",
        type: "array<bytebase.v1.TriggerMetadata>",
        description: "The triggers is the list of triggers in a view.",
      },
    ],
    description: "ViewMetadata is the metadata for views.",
  },
  "bytebase.v1.Webhook": {
    type: "object",
    properties: [
      {
        name: "directMessage",
        type: "boolean",
        description:
          "if direct_message is set, the notification is sent directly\n to the persons and url will be ignored.\n IM integration setting should be set for this function to work.",
      },
      {
        name: "name",
        type: "string",
        description:
          "name is the name of the webhook, generated by the server.\n format: projects/{project}/webhooks/{webhook}",
      },
      {
        name: "notificationTypes",
        type: "array<bytebase.v1.Activity.Type>",
        description:
          "notification_types is the list of activities types that the webhook is interested in.\n Bytebase will only send notifications to the webhook if the activity type is in the list.\n It should not be empty, and should be a subset of the following:\n - ISSUE_CREATED\n - ISSUE_APPROVAL_REQUESTED\n - ISSUE_SENT_BACK\n - ISSUE_APPROVED\n - PIPELINE_FAILED\n - PIPELINE_COMPLETED",
      },
      {
        name: "title",
        type: "string",
        description: "title is the title of the webhook.",
      },
      {
        name: "type",
        type: "bytebase.v1.WebhookType",
        description:
          "Webhook integration type.\n type is the type of the webhook.",
      },
      {
        name: "url",
        type: "string",
        description:
          "url is the url of the webhook, should be unique within the project.",
      },
    ],
    description: "",
  },
  "bytebase.v1.WebhookType": {
    type: "enum",
    values: [
      "WEBHOOK_TYPE_UNSPECIFIED",
      "SLACK",
      "DISCORD",
      "TEAMS",
      "DINGTALK",
      "FEISHU",
      "WECOM",
      "LARK",
    ],
    description: "Webhook integration type.",
  },
  "bytebase.v1.WorkloadIdentity": {
    type: "object",
    properties: [
      {
        name: "createTime",
        type: "google.protobuf.Timestamp",
        description: "The timestamp when the workload identity was created.",
      },
      {
        name: "email",
        type: "string",
        description:
          "The email of the workload identity.\n For workspace-level: {name}@workload.bytebase.com\n For project-level: {name}@{project-id}.workload.bytebase.com",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the workload identity.\n Format: workloadIdentities/{email}",
      },
      {
        name: "state",
        type: "bytebase.v1.State",
        description: "The state of the workload identity.",
      },
      {
        name: "title",
        type: "string",
        description: "The display title of the workload identity.",
      },
      {
        name: "workloadIdentityConfig",
        type: "bytebase.v1.WorkloadIdentityConfig",
        description:
          "The workload identity configuration for OIDC token validation.",
      },
    ],
    description:
      "WorkloadIdentity represents an external CI/CD workload identity.",
  },
  "bytebase.v1.WorkloadIdentityConfig": {
    type: "object",
    properties: [
      {
        name: "allowedAudiences",
        type: "array<string>",
        description: "Allowed audiences for token validation",
      },
      {
        name: "issuerUrl",
        type: "string",
        description:
          "OIDC Issuer URL (auto-filled based on provider_type, can be overridden)",
      },
      {
        name: "providerType",
        type: "bytebase.v1.WorkloadIdentityConfig.ProviderType",
        description: "Platform type (currently only GITHUB is supported)",
      },
      {
        name: "subjectPattern",
        type: "string",
        description:
          'Subject pattern to match (e.g., "repo:owner/repo:ref:refs/heads/main")',
      },
    ],
    description: "WorkloadIdentityConfig for API layer",
  },
  "bytebase.v1.WorkloadIdentityConfig.ProviderType": {
    type: "enum",
    values: ["PROVIDER_TYPE_UNSPECIFIED", "GITHUB", "GITLAB"],
    description: "ProviderType identifies the CI/CD platform.",
  },
  "bytebase.v1.Worksheet": {
    type: "object",
    properties: [
      {
        name: "content",
        type: "string",
        description:
          "The content of the worksheet.\n By default, it will be cut off in SearchWorksheet() method. If it doesn't match the `content_size`, you can\n use GetWorksheet() request to retrieve the full content.",
      },
      {
        name: "contentSize",
        type: "integer",
        description:
          "content_size is the full size of the content, may not match the size of the `content` field.",
      },
      {
        name: "createTime",
        type: "google.protobuf.Timestamp",
        description: "The create time of the worksheet.",
      },
      {
        name: "creator",
        type: "string",
        description: "The creator of the Worksheet.\n Format: users/{email}",
      },
      {
        name: "database",
        type: "string",
        description:
          "The database resource name.\n Format: instances/{instance}/databases/{database}\n If the database parent doesn't exist, the database field is empty.",
      },
      {
        name: "folders",
        type: "array<string>",
      },
      {
        name: "name",
        type: "string",
        description:
          "The name of the worksheet resource, generated by the server.\n Canonical parent is project.\n Format: projects/{project}/worksheets/{worksheet}",
      },
      {
        name: "project",
        type: "string",
        description: "The project resource name.\n Format: projects/{project}",
      },
      {
        name: "starred",
        type: "boolean",
        description:
          "starred indicates whether the worksheet is starred by the current authenticated user.",
      },
      {
        name: "title",
        type: "string",
        description: "The title of the worksheet.",
      },
      {
        name: "updateTime",
        type: "google.protobuf.Timestamp",
        description: "The last update time of the worksheet.",
      },
      {
        name: "visibility",
        type: "bytebase.v1.Worksheet.Visibility",
      },
    ],
    description: "",
  },
  "bytebase.v1.Worksheet.Visibility": {
    type: "enum",
    values: [
      "VISIBILITY_UNSPECIFIED",
      "PROJECT_READ",
      "PROJECT_WRITE",
      "PRIVATE",
    ],
    description: "",
  },
  "bytebase.v1.WorksheetOrganizer": {
    type: "object",
    properties: [
      {
        name: "folders",
        type: "array<string>",
      },
      {
        name: "starred",
        type: "boolean",
        description: "starred means if the worksheet is starred.",
      },
      {
        name: "worksheet",
        type: "string",
        description:
          "The name of the worksheet.\n Format: projects/{project}/worksheets/{worksheet}",
      },
    ],
    description: "",
  },
  "bytebase.v1.Workspace": {
    type: "object",
    properties: [
      {
        name: "logo",
        type: "string",
        description: "The branding logo.",
      },
      {
        name: "name",
        type: "string",
        description: "Format: workspaces/{workspace}",
      },
      {
        name: "title",
        type: "string",
      },
    ],
    description: "",
  },
  "bytebase.v1.WorkspaceApprovalSetting": {
    type: "object",
    properties: [
      {
        name: "rules",
        type: "array<bytebase.v1.WorkspaceApprovalSetting.Rule>",
      },
    ],
    description: "",
  },
  "bytebase.v1.WorkspaceApprovalSetting.Rule": {
    type: "object",
    properties: [
      {
        name: "condition",
        type: "google.type.Expr",
        description:
          'The condition that is associated with the rule.\n The syntax and semantics of CEL are documented at https://github.com/google/cel-spec\n\n The `source` field filters which rules apply. The `condition` field then evaluates with full context.\n\n All supported variables:\n statement.affected_rows: affected row count in the DDL/DML, support "==", "!=", "<", "<=", ">", ">=" operations.\n statement.table_rows: table row count number, support "==", "!=", "<", "<=", ">", ">=" operations.\n resource.environment_id: the environment resource id, support "==", "!=", "in [xx]", "!(in [xx])" operations.\n resource.project_id: the project resource id, support "==", "!=", "in [xx]", "!(in [xx])", "contains()", "matches()", "startsWith()", "endsWith()" operations.\n resource.db_engine: the database engine type, support "==", "!=", "in [xx]", "!(in [xx])" operations. Check the Engine enum for values.\n statement.sql_type: the SQL type, support "==", "!=", "in [xx]", "!(in [xx])" operations.\n resource.database_name: the database name, support "==", "!=", "in [xx]", "!(in [xx])", "contains()", "matches()", "startsWith()", "endsWith()" operations.\n resource.schema_name: the schema name, support "==", "!=", "in [xx]", "!(in [xx])", "contains()", "matches()", "startsWith()", "endsWith()" operations.\n resource.table_name: the table name, support "==", "!=", "in [xx]", "!(in [xx])", "contains()", "matches()", "startsWith()", "endsWith()" operations.\n statement.text: the SQL statement, support "contains()", "matches()", "startsWith()", "endsWith()" operations.\n request.expiration_days: the role expiration days for the request, support "==", "!=", "<", "<=", ">", ">=" operations.\n request.role: the request role full name, support "==", "!=", "in [xx]", "!(in [xx])", "contains()", "matches()", "startsWith()", "endsWith()" operations.\n\n When source is CHANGE_DATABASE, support: statement.*, resource.* (excluding request.*)\n When source is CREATE_DATABASE, support: resource.environment_id, resource.project_id, resource.db_engine, resource.database_name\n When source is EXPORT_DATA, support: resource.environment_id, resource.project_id, resource.db_engine, resource.database_name, resource.schema_name, resource.table_name\n When source is REQUEST_ROLE, support: resource.project_id, request.expiration_days, request.role\n When source is REQUEST_ACCESS, support: resource.environment_id, resource.project_id, request.unmask\n\n For examples:\n resource.environment_id == "prod" && statement.affected_rows >= 100\n resource.table_name.matches("sensitive_.*") && resource.db_engine == "MYSQL"',
      },
      {
        name: "source",
        type: "bytebase.v1.WorkspaceApprovalSetting.Rule.Source",
      },
      {
        name: "template",
        type: "bytebase.v1.ApprovalTemplate",
      },
    ],
    description: "",
  },
  "bytebase.v1.WorkspaceApprovalSetting.Rule.Source": {
    type: "enum",
    values: [
      "SOURCE_UNSPECIFIED",
      "CHANGE_DATABASE",
      "CREATE_DATABASE",
      "EXPORT_DATA",
      "REQUEST_ROLE",
      "REQUEST_ACCESS",
    ],
    description: "",
  },
  "bytebase.v1.WorkspaceProfileSetting": {
    type: "object",
    properties: [
      {
        name: "accessTokenDuration",
        type: "google.protobuf.Duration",
        description: "The duration for access token. Default is 1 hour.",
      },
      {
        name: "announcement",
        type: "bytebase.v1.Announcement",
        description: "The setting of custom announcement",
      },
      {
        name: "databaseChangeMode",
        type: "bytebase.v1.DatabaseChangeMode",
        description: "The workspace database change mode.",
      },
      {
        name: "directorySyncToken",
        type: "string",
        description: "The token for directory sync authentication.",
      },
      {
        name: "disallowPasswordSignin",
        type: "boolean",
        description:
          "Whether to disallow password signin. (Except workspace admins)",
      },
      {
        name: "disallowSignup",
        type: "boolean",
        description:
          "Disallow self-service signup, users can only be invited by the owner.",
      },
      {
        name: "domains",
        type: "array<string>",
        description: "The workspace domain, e.g., bytebase.com.",
      },
      {
        name: "enableAuditLogStdout",
        type: "boolean",
        description:
          "Whether to enable audit logging to stdout in structured JSON format.\n Requires TEAM or ENTERPRISE license.",
      },
      {
        name: "enableDebug",
        type: "boolean",
        description: "Whether debug mode is enabled.",
      },
      {
        name: "enableMetricCollection",
        type: "boolean",
        description: "Whether to enable metric collection for the workspace.",
      },
      {
        name: "enforceIdentityDomain",
        type: "boolean",
        description:
          "Only user and group from the domains can be created and login.",
      },
      {
        name: "externalUrl",
        type: "string",
        description:
          "The external URL is used for sso authentication callback.",
      },
      {
        name: "inactiveSessionTimeout",
        type: "google.protobuf.Duration",
        description:
          "The session expiration time if not activity detected for the user. Value <= 0 means no limit.",
      },
      {
        name: "maximumRoleExpiration",
        type: "google.protobuf.Duration",
        description: "The max duration for role expired.",
      },
      {
        name: "passwordRestriction",
        type: "bytebase.v1.WorkspaceProfileSetting.PasswordRestriction",
        description: "Password restriction settings.",
      },
      {
        name: "queryTimeout",
        type: "google.protobuf.Duration",
        description:
          "The query timeout duration for query and export, works for the SQL Editor and Export Center.",
      },
      {
        name: "refreshTokenDuration",
        type: "google.protobuf.Duration",
        description: "The duration for refresh token. Default is 7 days.",
      },
      {
        name: "require2fa",
        type: "boolean",
        description: "Require 2FA for all users.",
      },
      {
        name: "sqlResultSize",
        type: "integer",
        description:
          "The maximum result size limit in bytes for query and export, works for the SQL Editor and Export Center.\n The default value is 100MB, we will use the default value if the setting not exists, or the limit <= 0.",
      },
      {
        name: "watermark",
        type: "boolean",
        description:
          "Whether to display watermark on pages.\n Requires ENTERPRISE license.",
      },
    ],
    description: "",
  },
  "bytebase.v1.WorkspaceProfileSetting.PasswordRestriction": {
    type: "object",
    properties: [
      {
        name: "minLength",
        type: "integer",
        description:
          "min_length is the minimum length for password, should no less than 8.",
      },
      {
        name: "passwordRotation",
        type: "google.protobuf.Duration",
        description:
          "password_rotation requires users to reset their password after the duration.",
      },
      {
        name: "requireLetter",
        type: "boolean",
        description:
          "require_letter requires the password must contains at least one letter, regardless of upper case or lower case",
      },
      {
        name: "requireNumber",
        type: "boolean",
        description:
          "require_number requires the password must contains at least one number.",
      },
      {
        name: "requireResetPasswordForFirstLogin",
        type: "boolean",
        description:
          "require_reset_password_for_first_login requires users to reset their password after the 1st login.",
      },
      {
        name: "requireSpecialCharacter",
        type: "boolean",
        description:
          "require_special_character requires the password must contains at least one special character.",
      },
      {
        name: "requireUppercaseLetter",
        type: "boolean",
        description:
          "require_uppercase_letter requires the password must contains at least one upper case letter.",
      },
    ],
    description: "",
  },
};
