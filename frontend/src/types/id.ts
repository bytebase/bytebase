// Use number as the ID type, though the ID type could be transparent to the client and just
// use string, however, that requires server to do string to int conversion since we store
// ID as auto incremental int in the database.
export type IDType = number;

export type ExecutionID = IDType;

export type PrincipalID = IDType;

export type MemberID = IDType;

export type SettingID = IDType;

export type BookmarkID = IDType;

export type PolicyID = IDType;

export type ProjectID = IDType;

export type ProjectWebhookID = IDType;

export type IssueID = IDType;

export type PipelineID = IDType;

export type StageID = IDType;

export type TaskID = IDType;

export type TaskRunID = IDType;

export type TaskCheckRunID = IDType;

export type ActivityID = IDType;

export type InboxID = IDType;

export type EnvironmentID = IDType;

export type InstanceID = IDType;

export type InstanceUserID = IDType;

export type DataSourceID = IDType;

export type DatabaseID = IDType;

export type TableID = IDType;

export type ViewID = IDType;

export type ColumnID = IDType;

export type TableIndexID = IDType;

export type VCSID = IDType;

export type RepositoryID = IDType;

export type MigrationHistoryID = IDType;

export type BackupID = IDType;

export type BackupSettingID = IDType;

export type AnomalyID = IDType;

export type CommandID = IDType;
export type CommandRegisterID = IDType;

// This references to the object id, which can be used as a container.
// Currently, only issue can be used a container.
// The type is used by Activity and Message
export type ContainerID = IssueID;
