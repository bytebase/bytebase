// Use number as the ID type, though the ID type could be transparent to the client and just
// use string, however, that requires server to do string to int conversion since we store
// ID as auto incremental int in the database.
export type IdType = number;

export type ExecutionId = IdType;

export type PrincipalId = IdType;

export type MemberId = IdType;

export type BookmarkId = IdType;

export type ProjectId = IdType;

export type IssueId = IdType;

export type PipelineId = IdType;

export type StageId = IdType;

export type TaskId = IdType;

export type TaskRunId = IdType;

export type ActivityId = IdType;

export type MessageId = IdType;

export type EnvironmentId = IdType;

export type InstanceId = IdType;

export type DataSourceId = IdType;

export type DatabaseId = IdType;

export type TableId = IdType;

export type VCSId = IdType;

export type CommandId = IdType;
export type CommandRegisterId = IdType;

// This references to the object id, which can be used as a container.
// Currently, only issue can be used a container.
// The type is used by Activity and Message
export type ContainerId = IssueId;
