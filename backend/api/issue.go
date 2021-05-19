package api

import "context"

// Issue status
type IssueStatus string

const (
	Issue_Open     IssueStatus = "OPEN"
	Issue_Done     IssueStatus = "DONE"
	Issue_Canceled IssueStatus = "CANCELED"
)

func (e IssueStatus) String() string {
	switch e {
	case Issue_Open:
		return "OPEN"
	case Issue_Done:
		return "DONE"
	case Issue_Canceled:
		return "CANCELED"
	}
	return ""
}

// Issue type
type IssueType string

const (
	IssueGeneral              IssueType = "bb.general"
	IssueDatabaseCreate       IssueType = "bb.database.create"
	IssueDatabaseGrant        IssueType = "bb.database.grant"
	IssueDatabaseSchemaUpdate IssueType = "bb.database.schema.update"
	IssueDataSourceRequest    IssueType = "bb.data-source.request"
)

func (e IssueType) String() string {
	switch e {
	case IssueGeneral:
		return "bb.general"
	case IssueDatabaseCreate:
		return "bb.database.create"
	case IssueDatabaseGrant:
		return "bb.database.grant"
	case IssueDatabaseSchemaUpdate:
		return "bb.database.schema.update"
	case IssueDataSourceRequest:
		return "bb.data-source.request"
	}
	return "bb.unknown"
}

type IssuePayload struct {
}

type Issue struct {
	ID int `jsonapi:"primary,issue"`

	// Standard fields
	CreatorId   int
	Creator     *Principal `jsonapi:"attr,creator"`
	CreatedTs   int64      `jsonapi:"attr,createdTs"`
	UpdaterId   int
	Updater     *Principal `jsonapi:"attr,updater"`
	UpdatedTs   int64      `jsonapi:"attr,updatedTs"`
	WorkspaceId int

	// Related fields
	ProjectId  int
	Project    *Project `jsonapi:"relation,project"`
	PipelineId int
	Pipeline   *Pipeline `jsonapi:"relation,pipeline"`

	// Domain specific fields
	Name             string       `jsonapi:"attr,name"`
	Status           IssueStatus  `jsonapi:"attr,status"`
	Type             IssueType    `jsonapi:"attr,type"`
	Description      string       `jsonapi:"attr,description"`
	AssigneeId       int          `jsonapi:"attr,assigneeId"`
	Assignee         *Principal   `jsonapi:"attr,assignee"`
	SubscriberIdList []int        `jsonapi:"attr,subscriberIdList"`
	Sql              string       `jsonapi:"attr,sql"`
	RollbackSql      string       `jsonapi:"attr,rollbackSql"`
	Payload          IssuePayload `jsonapi:"attr,payload"`
}

type IssueCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId   int
	WorkspaceId int

	// Related fields
	ProjectId  int `jsonapi:"attr,projectId"`
	PipelineId int
	Pipeline   PipelineCreate `jsonapi:"attr,pipeline"`

	// Domain specific fields
	Name             string       `jsonapi:"attr,name"`
	Type             IssueType    `jsonapi:"attr,type"`
	Description      string       `jsonapi:"attr,description"`
	AssigneeId       int          `jsonapi:"attr,assigneeId"`
	SubscriberIdList []int        `jsonapi:"attr,subscriberIdList"`
	Sql              string       `jsonapi:"attr,sql"`
	RollbackSql      string       `jsonapi:"attr,rollbackSql"`
	Payload          IssuePayload `jsonapi:"attr,payload"`
}

type IssueFind struct {
	ID *int

	// Standard fields
	WorkspaceId *int
}

type IssuePatch struct {
	ID int `jsonapi:"primary,issuePatch"`

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId   int
	WorkspaceId int

	// Related fields
	ProjectId *int `jsonapi:"attr,projectId"`

	// Domain specific fields
	Name *string `jsonapi:"attr,name"`
	// Status is only set manualy via IssueStatusPatch
	Status           *IssueStatus
	Description      *string       `jsonapi:"attr,description"`
	AssigneeId       *int          `jsonapi:"attr,assigneeId"`
	SubscriberIdList *[]int        `jsonapi:"attr,subscriberIdList"`
	Payload          *IssuePayload `jsonapi:"attr,payload"`
}

type IssueStatusPatch struct {
	ID int `jsonapi:"primary,issueStatusPatch"`

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId   int
	WorkspaceId int

	// Domain specific fields
	Status  *string `jsonapi:"attr,status"`
	Comment *string `jsonapi:"attr,comment"`
}

type IssueService interface {
	CreateIssue(ctx context.Context, create *IssueCreate) (*Issue, error)
	FindIssueList(ctx context.Context, find *IssueFind) ([]*Issue, error)
	FindIssue(ctx context.Context, find *IssueFind) (*Issue, error)
	PatchIssue(ctx context.Context, patch *IssuePatch) (*Issue, error)
}
