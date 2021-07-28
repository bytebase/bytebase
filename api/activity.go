package api

import (
	"context"
	"encoding/json"
)

// Activity type
type ActivityType string

const (
	ActivityIssueCreate              ActivityType = "bb.issue.create"
	ActivityIssueCommentCreate       ActivityType = "bb.issue.comment.create"
	ActivityIssueFieldUpdate         ActivityType = "bb.issue.field.update"
	ActivityIssueStatusUpdate        ActivityType = "bb.issue.status.update"
	ActivityPipelineTaskStatusUpdate ActivityType = "bb.pipeline.task.status.update"
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
	}
	return "bb.activity.unknown"
}

// These payload types are only used when marshalling to the json format for saving into the database.
// So we annotate with json tag using camelCase naming which is consistent with normal
// json naming convention. More importantly, frontend code can simply use JSON.parse to
// convert to the expected struct there.
type ActivityIssueCreatePayload struct {
	// This is used by inbox to display the issue name and avoid the join cost to fetch the name via container_id
	IssueName string `json:"issueName"`
}

type ActivityIssueCommentCreatePayload struct {
	// This is used by inbox to display the issue name and avoid the join cost to fetch the name via container_id
	IssueName string `json:"issueName"`
}

type ActivityIssueFieldUpdatePayload struct {
	FieldId  IssueFieldId `json:"fieldId"`
	OldValue string       `json:"oldValue,omitempty"`
	NewValue string       `json:"newValue,omitempty"`
	// This is used by inbox to display the issue name and avoid the join cost to fetch the name via container_id
	IssueName string `json:"issueName"`
}

type ActivityIssueStatusUpdatePayload struct {
	OldStatus IssueStatus `json:"oldStatus,omitempty"`
	NewStatus IssueStatus `json:"newStatus,omitempty"`
	// This is used by inbox to display the issue name and avoid the join cost to fetch the name via container_id
	IssueName string `json:"issueName"`
}

type ActivityPipelineTaskStatusUpdatePayload struct {
	TaskId    int        `json:"taskId"`
	OldStatus TaskStatus `json:"oldStatus,omitempty"`
	NewStatus TaskStatus `json:"newStatus,omitempty"`
	// This is used by inbox to display the issue name and avoid the join cost to fetch the name via container_id
	IssueName string `json:"issueName"`
	TaskName  string `json:"taskName"`
}

type Activity struct {
	ID int `jsonapi:"primary,activity"`

	// Standard fields
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`
	// Related fields
	// The object where this activity belongs
	// e.g if Type is "bb.issue.xxx", then this field refers to the corresponding issue's id.
	ContainerId int `jsonapi:"attr,containerId"`

	// Domain specific fields
	Type    ActivityType `jsonapi:"attr,actionType"`
	Comment string       `jsonapi:"attr,comment"`
	Payload string       `jsonapi:"attr,payload"`
}

type ActivityCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Domain specific fields
	ContainerId int          `jsonapi:"attr,containerId"`
	Type        ActivityType `jsonapi:"attr,actionType"`
	Comment     string       `jsonapi:"attr,comment"`
	Payload     string       `jsonapi:"attr,payload"`
}

type ActivityFind struct {
	ID *int

	// Domain specific fields
	ContainerId *int
}

func (find *ActivityFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type ActivityPatch struct {
	ID int `jsonapi:"primary,activityPatch"`

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int

	// Domain specific fields
	Comment *string `jsonapi:"attr,comment"`
}

type ActivityDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterId int
}

type ActivityService interface {
	CreateActivity(ctx context.Context, create *ActivityCreate) (*Activity, error)
	FindActivityList(ctx context.Context, find *ActivityFind) ([]*Activity, error)
	FindActivity(ctx context.Context, find *ActivityFind) (*Activity, error)
	PatchActivity(ctx context.Context, patch *ActivityPatch) (*Activity, error)
	DeleteActivity(ctx context.Context, delete *ActivityDelete) error
}
