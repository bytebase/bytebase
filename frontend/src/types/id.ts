// ResourceId is an unique identifier.
export type ResourceId = string;

// IdType includes auto incremental int in database and resource id.
export type IdType = number | ResourceId;

export type ExecutionId = IdType;

export type PrincipalId = IdType;

export type MemberId = IdType;

export type SettingId = IdType;

export type PolicyId = IdType;

export type ProjectId = IdType;

export type ProjectWebhookId = IdType;

export type IssueId = IdType;

export type PipelineId = IdType;

export type StageId = IdType;

export type TaskId = IdType;

export type TaskRunId = IdType;

export type TaskCheckRunId = IdType;

export type EnvironmentId = IdType;

export type InstanceId = IdType;

export type InstanceUserId = string;

export type DataSourceId = IdType;

export type DatabaseId = IdType;

export type VCSId = IdType;

export type RepositoryId = IdType;

export type MigrationHistoryId = string;

export type BackupId = IdType;

export type BackupSettingId = IdType;

export type CommandId = string;
export type CommandRegisterId = string;

export type SheetId = IdType;

// This references to the object id, which can be used as a container.
// Currently, only issue can be used a container.
// The type is used by Activity and Message
export type ContainerId = IssueId;
