// ResourceId is an unique identifier.
export type ResourceId = string;

// IdType includes auto incremental int in database and resource id.
export type IdType = number | ResourceId;

export type ExecutionId = IdType;

export type PrincipalId = IdType;

export type ProjectId = IdType;

export type EnvironmentId = IdType;

export type InstanceId = IdType;

export type InstanceUserId = string;

export type DataSourceId = IdType;

export type DatabaseId = IdType;

export type VCSId = IdType;

export type RepositoryId = IdType;

export type MigrationHistoryId = string;

export type CommandId = string;
export type CommandRegisterId = string;

export type SheetId = IdType;
