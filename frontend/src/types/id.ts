// These ID format may change in the future, so we encapsulate with a type.
// Also good for readability.

export type ExecutionId = string;

export type UserId = string;

// For now, Principal is equal to UserId, in the future it may contain other id such as application, bot etc.
export type PrincipalId = UserId;

export type MemberId = string;

export type BookmarkId = string;

export type ProjectId = string;

export type IssueId = string;

export type PipelineId = string;

export type StageId = string;

export type TaskId = string;

export type ActivityId = string;

export type MessageId = string;

export type EnvironmentId = string;

export type InstanceId = string;

export type DataSourceId = string;

export type DatabaseId = string;

export type CommandId = string;
export type CommandRegisterId = string;

// This references to the object id, which can be used as a container.
// Currently, only issue can be used a container.
// The type is used by Activity and Message
export type ContainerId = IssueId;
