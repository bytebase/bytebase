package api

import (
	"encoding/json"

	"github.com/bytebase/bytebase/plugin/vcs"
)

// ActivityType is the type for an activity.
type ActivityType string

const (
	// Issue related.

	// ActivityIssueCreate is the type for creating issues.
	ActivityIssueCreate ActivityType = "bb.issue.create"
	// ActivityIssueCommentCreate is the type for creating issue comments.
	ActivityIssueCommentCreate ActivityType = "bb.issue.comment.create"
	// ActivityIssueFieldUpdate is the type for updating issue fields.
	ActivityIssueFieldUpdate ActivityType = "bb.issue.field.update"
	// ActivityIssueStatusUpdate is the type for updating issue status.
	ActivityIssueStatusUpdate ActivityType = "bb.issue.status.update"
	// ActivityPipelineTaskStatusUpdate is the type for updating pipeline task status.
	ActivityPipelineTaskStatusUpdate ActivityType = "bb.pipeline.task.status.update"
	// ActivityPipelineTaskFileCommit is the type for committing pipeline task file.
	ActivityPipelineTaskFileCommit ActivityType = "bb.pipeline.task.file.commit"
	// ActivityPipelineTaskStatementUpdate is the type for updating pipeline task SQL statement.
	ActivityPipelineTaskStatementUpdate ActivityType = "bb.pipeline.task.statement.update"
	// ActivityPipelineTaskEarliestAllowedTimeUpdate is the type for updating pipeline task the earliest allowed time.
	ActivityPipelineTaskEarliestAllowedTimeUpdate ActivityType = "bb.pipeline.task.general.earliest-allowed-time.update"

	// Member related

	// ActivityMemberCreate is the type for creating members.
	ActivityMemberCreate ActivityType = "bb.member.create"
	// ActivityMemberRoleUpdate is the type for updating member roles.
	ActivityMemberRoleUpdate ActivityType = "bb.member.role.update"
	// ActivityMemberActivate is the type for activating members.
	ActivityMemberActivate ActivityType = "bb.member.activate"
	// ActivityMemberDeactivate is the type for deactivating members.
	ActivityMemberDeactivate ActivityType = "bb.member.deactivate"

	// Project related

	// ActivityProjectRepositoryPush is the type for pushing repositories.
	ActivityProjectRepositoryPush ActivityType = "bb.project.repository.push"
	// ActivityProjectDatabaseTransfer is the type for transferring databases.
	ActivityProjectDatabaseTransfer ActivityType = "bb.project.database.transfer"
	// ActivityProjectMemberCreate is the type for creating project members.
	ActivityProjectMemberCreate ActivityType = "bb.project.member.create"
	// ActivityProjectMemberDelete is the type for deleting project members.
	ActivityProjectMemberDelete ActivityType = "bb.project.member.delete"
	// ActivityProjectMemberRoleUpdate is the type for updating project member roles.
	ActivityProjectMemberRoleUpdate ActivityType = "bb.project.member.role.update"

	// SQL Editor related

	// ActivitySQLEditorQuery is the type for executing query.
	ActivitySQLEditorQuery ActivityType = "bb.sql-editor.query"
)

func (e ActivityType) String() string {
	switch e {
	case ActivityIssueCreate:
		return "bb.issue.create"
	case ActivityIssueCommentCreate:
		return "bb.issue.comment.create"
	case ActivityIssueFieldUpdate:
		return "bb.issue.field.update"
	case ActivityIssueStatusUpdate:
		return "bb.issue.status.update"
	case ActivityPipelineTaskStatusUpdate:
		return "bb.pipeline.task.status.update"
	case ActivityPipelineTaskFileCommit:
		return "bb.pipeline.task.file.commit"
	case ActivityPipelineTaskStatementUpdate:
		return "bb.pipeline.task.statement.update"
	case ActivityMemberCreate:
		return "bb.member.create"
	case ActivityMemberRoleUpdate:
		return "bb.member.role.update"
	case ActivityMemberActivate:
		return "bb.member.activate"
	case ActivityMemberDeactivate:
		return "bb.member.deactivate"
	case ActivityProjectRepositoryPush:
		return "bb.project.repository.push"
	case ActivityProjectDatabaseTransfer:
		return "bb.project.database.transfer"
	case ActivityProjectMemberCreate:
		return "bb.project.member.create"
	case ActivityProjectMemberDelete:
		return "bb.project.member.delete"
	case ActivityProjectMemberRoleUpdate:
		return "bb.project.member.role.update"
	case ActivitySQLEditorQuery:
		return "bb.sql-editor.query"
	}
	return "bb.activity.unknown"
}

// ActivityLevel is the level of activities.
type ActivityLevel string

const (
	// ActivityInfo is the INFO level of activities.
	ActivityInfo ActivityLevel = "INFO"
	// ActivityWarn is the WARN level of activities.
	ActivityWarn ActivityLevel = "WARN"
	// ActivityError is the ERROR level of activities.
	ActivityError ActivityLevel = "ERROR"
)

func (e ActivityLevel) String() string {
	switch e {
	case ActivityInfo:
		return "INFO"
	case ActivityWarn:
		return "WARN"
	case ActivityError:
		return "ERROR"
	}
	return "UNKNOWN"
}

// ActivityIssueCreatePayload is the API message payloads for creating issues.
// These payload types are only used when marshalling to the json format for saving into the database.
// So we annotate with json tag using camelCase naming which is consistent with normal
// json naming convention. More importantly, frontend code can simply use JSON.parse to
// convert to the expected struct there.
type ActivityIssueCreatePayload struct {
	// Used by inbox to display info without paying the join cost
	IssueName string `json:"issueName"`
}

// ActivityIssueCommentCreatePayload is the API message payloads for creating issue comments.
type ActivityIssueCommentCreatePayload struct {
	// Used by inbox to display info without paying the join cost
	IssueName string `json:"issueName"`
}

// ActivityIssueFieldUpdatePayload is the API message payloads for updating issue fields.
type ActivityIssueFieldUpdatePayload struct {
	FieldID  IssueFieldID `json:"fieldId"`
	OldValue string       `json:"oldValue,omitempty"`
	NewValue string       `json:"newValue,omitempty"`
	// Used by inbox to display info without paying the join cost
	IssueName string `json:"issueName"`
}

// ActivityIssueStatusUpdatePayload is the API message payloads for updating issue status.
type ActivityIssueStatusUpdatePayload struct {
	OldStatus IssueStatus `json:"oldStatus,omitempty"`
	NewStatus IssueStatus `json:"newStatus,omitempty"`
	// Used by inbox to display info without paying the join cost
	IssueName string `json:"issueName"`
}

// ActivityPipelineTaskStatusUpdatePayload is the API message payloads for updating pipeline task status.
type ActivityPipelineTaskStatusUpdatePayload struct {
	TaskID    int        `json:"taskId"`
	OldStatus TaskStatus `json:"oldStatus,omitempty"`
	NewStatus TaskStatus `json:"newStatus,omitempty"`
	// Used by inbox to display info without paying the join cost
	IssueName string `json:"issueName"`
	TaskName  string `json:"taskName"`
}

// ActivityPipelineTaskFileCommitPayload is the API message payloads for committing pipeline task files.
type ActivityPipelineTaskFileCommitPayload struct {
	TaskID             int    `json:"taskId"`
	VCSInstanceURL     string `json:"vcsInstanceUrl,omitempty"`
	RepositoryFullPath string `json:"repositoryFullPath,omitempty"`
	Branch             string `json:"branch,omitempty"`
	FilePath           string `json:"filePath,omitempty"`
	CommitID           string `json:"commitId,omitempty"`
}

// ActivityPipelineTaskStatementUpdatePayload is the API message payloads for pipeline task statement updates.
type ActivityPipelineTaskStatementUpdatePayload struct {
	TaskID       int    `json:"taskId"`
	OldStatement string `json:"oldStatement,omitempty"`
	NewStatement string `json:"newStatement,omitempty"`
	// Used by inbox to display info without paying the join cost
	IssueName string `json:"issueName"`
	TaskName  string `json:"taskName"`
}

// ActivityPipelineTaskEarliestAllowedTimeUpdatePayload is the API message payloads for pipeline task the earliest allowed time updates.
type ActivityPipelineTaskEarliestAllowedTimeUpdatePayload struct {
	TaskID               int   `json:"taskId"`
	OldEarliestAllowedTs int64 `json:"oldEarliestAllowedTs,omitempty"`
	NewEarliestAllowedTs int64 `json:"newEarliestAllowedTs,omitempty"`
	// Used by inbox to display info without paying the join cost
	IssueName string `json:"issueName"`
	TaskName  string `json:"taskName"`
}

// ActivityMemberCreatePayload is the API message payloads for creating members.
type ActivityMemberCreatePayload struct {
	PrincipalID    int          `json:"principalId"`
	PrincipalName  string       `json:"principalName"`
	PrincipalEmail string       `json:"principalEmail"`
	MemberStatus   MemberStatus `json:"memberStatus"`
	Role           Role         `json:"role"`
}

// ActivityMemberRoleUpdatePayload is the API message payloads for updating member roles.
type ActivityMemberRoleUpdatePayload struct {
	PrincipalID    int    `json:"principalId"`
	PrincipalName  string `json:"principalName"`
	PrincipalEmail string `json:"principalEmail"`
	OldRole        Role   `json:"oldRole"`
	NewRole        Role   `json:"newRole"`
}

// ActivityMemberActivateDeactivatePayload is the API message payloads for activating or deactivating members.
type ActivityMemberActivateDeactivatePayload struct {
	PrincipalID    int    `json:"principalId"`
	PrincipalName  string `json:"principalName"`
	PrincipalEmail string `json:"principalEmail"`
	Role           Role   `json:"role"`
}

// ActivityProjectRepositoryPushPayload is the API message payloads for pushing repositories.
type ActivityProjectRepositoryPushPayload struct {
	VCSPushEvent vcs.PushEvent `json:"pushEvent"`
	// Used by activity table to display info without paying the join cost
	// IssueID/IssueName only exist if the push event leads to the issue creation.
	IssueID   int    `json:"issueId,omitempty"`
	IssueName string `json:"issueName,omitempty"`
}

// ActivityProjectDatabaseTransferPayload is the API message payloads for transferring databases.
type ActivityProjectDatabaseTransferPayload struct {
	DatabaseID int `json:"databaseId,omitempty"`
	// Used by activity table to display info without paying the join cost
	DatabaseName string `json:"databaseName,omitempty"`
}

// ActivitySQLEditorQueryPayload is the API message payloads for the executed query info.
type ActivitySQLEditorQueryPayload struct {
	// Used by activity table to display info without paying the join cost
	Statement    string `json:"statement"`
	DurationNs   int64  `json:"durationNs"`
	InstanceName string `json:"instanceName"`
	DatabaseName string `json:"databaseName"`
	Error        string `json:"error"`
}

// Activity is the API message for an activity.
type Activity struct {
	ID int `jsonapi:"primary,activity"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	// The object where this activity belongs
	// e.g if Type is "bb.issue.xxx", then this field refers to the corresponding issue's id.
	ContainerID int `jsonapi:"attr,containerId"`

	// Domain specific fields
	Type    ActivityType  `jsonapi:"attr,type"`
	Level   ActivityLevel `jsonapi:"attr,level"`
	Comment string        `jsonapi:"attr,comment"`
	Payload string        `jsonapi:"attr,payload"`
}

// ActivityCreate is the API message for creating an activity.
type ActivityCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Domain specific fields
	ContainerID int          `jsonapi:"attr,containerId"`
	Type        ActivityType `jsonapi:"attr,type"`
	Level       ActivityLevel
	Comment     string `jsonapi:"attr,comment"`
	Payload     string `jsonapi:"attr,payload"`
}

// ActivityFind is the API message for finding activities.
type ActivityFind struct {
	ID *int

	// Domain specific fields
	CreatorID   *int
	TypePrefix  *string
	Level       *ActivityLevel
	ContainerID *int
	Limit       *int
	// If specified, sorts the returned list by created_ts in <<ORDER>>
	// Different use cases want different orders.
	// e.g. Issue activity list wants ASC, while view recent activity list wants DESC.
	Order *SortOrder
}

func (find *ActivityFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// ActivityPatch is the API message for patching an activity.
type ActivityPatch struct {
	ID int `jsonapi:"primary,activityPatch"`

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Comment *string `jsonapi:"attr,comment"`
}

// ActivityDelete is the API message for deleting an activity.
type ActivityDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterID int
}
