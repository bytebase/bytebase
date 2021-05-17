package api

import "context"

// Activity type
type ActivityType string

const (
	ActivityIssueCreate          ActivityType = "bb.issue.create"
	ActivityIssueCommentCreate   ActivityType = "bb.issue.comment.create"
	ActivityIssueFieldUpdate     ActivityType = "bb.issue.field.update"
	ActivityIssueStatusUpdate    ActivityType = "bb.issue.status.update"
	ActivityPipelineStatusUpdate ActivityType = "bb.pipeline.task.status.update"
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
	case ActivityPipelineStatusUpdate:
		return "bb.pipeline.task.status.update"
	}
	return "bb.activity.unknown"
}

type Activity struct {
	ID int `jsonapi:"primary,activity"`

	// Standard fields
	CreatorId   int
	Creator     *Principal `jsonapi:"attr,creator"`
	CreatedTs   int64      `jsonapi:"attr,createdTs"`
	UpdaterId   int
	Updater     *Principal `jsonapi:"attr,updater"`
	UpdatedTs   int64      `jsonapi:"attr,updatedTs"`
	WorkspaceId int

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
	CreatorId   int
	WorkspaceId int

	// Domain specific fields
	ContainerId int          `jsonapi:"attr,containerId"`
	Type        ActivityType `jsonapi:"attr,actionType"`
	Comment     string       `jsonapi:"attr,comment"`
	Payload     string       `jsonapi:"attr,payload"`
}

type ActivityFind struct {
	ID *int

	// Standard fields
	WorkspaceId *int

	// Domain specific fields
	ContainerId *int
}

type ActivityPatch struct {
	ID int `jsonapi:"primary,activityPatch"`

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId   int
	WorkspaceId int

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
