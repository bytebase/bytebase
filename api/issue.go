package api

import (
	"context"
	"encoding/json"
)

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
	IssueGeneral              IssueType = "bb.issue.general"
	IssueDatabaseCreate       IssueType = "bb.issue.database.create"
	IssueDatabaseGrant        IssueType = "bb.issue.database.grant"
	IssueDatabaseSchemaUpdate IssueType = "bb.issue.database.schema.update"
	IssueDataSourceRequest    IssueType = "bb.issue.data-source.request"
)

func (e IssueType) String() string {
	switch e {
	case IssueGeneral:
		return "bb.issue.general"
	case IssueDatabaseCreate:
		return "bb.issue.database.create"
	case IssueDatabaseGrant:
		return "bb.issue.database.grant"
	case IssueDatabaseSchemaUpdate:
		return "bb.issue.database.schema.update"
	case IssueDataSourceRequest:
		return "bb.issue.data-source.request"
	}
	return "bb.unknown"
}

// It has to be string type because the id for stage field contain multiple parts.
type IssueFieldID string

const (
	IssueFieldName           IssueFieldID = "1"
	IssueFieldStatus         IssueFieldID = "2"
	IssueFieldAssignee       IssueFieldID = "3"
	IssueFieldDescription    IssueFieldID = "4"
	IssueFieldProject        IssueFieldID = "5"
	IssueFieldSubscriberList IssueFieldID = "6"
	IssueFieldSql            IssueFieldID = "7"
	IssueFieldRollbackSql    IssueFieldID = "8"
)

type Issue struct {
	ID int `jsonapi:"primary,issue"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	ProjectID  int
	Project    *Project `jsonapi:"relation,project"`
	PipelineID int
	Pipeline   *Pipeline `jsonapi:"relation,pipeline"`

	// Domain specific fields
	Name             string      `jsonapi:"attr,name"`
	Status           IssueStatus `jsonapi:"attr,status"`
	Type             IssueType   `jsonapi:"attr,type"`
	Description      string      `jsonapi:"attr,description"`
	AssigneeID       int         `jsonapi:"attr,assigneeID"`
	Assignee         *Principal  `jsonapi:"attr,assignee"`
	SubscriberIDList []int       `jsonapi:"attr,subscriberIDList"`
	Payload          string      `jsonapi:"attr,payload"`
}

type IssueCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	ProjectID  int `jsonapi:"attr,projectID"`
	PipelineID int
	Pipeline   PipelineCreate `jsonapi:"attr,pipeline"`

	// Domain specific fields
	Name              string    `jsonapi:"attr,name"`
	Type              IssueType `jsonapi:"attr,type"`
	Description       string    `jsonapi:"attr,description"`
	AssigneeID        int       `jsonapi:"attr,assigneeID"`
	SubscriberIDList  []int     `jsonapi:"attr,subscriberIDList"`
	RollbackIssueID   *int      `jsonapi:"attr,rollbackIssueID"`
	Payload           string    `jsonapi:"attr,payload"`
}

type IssueFind struct {
	ID *int

	// Related fields
	ProjectID *int

	// Domain specific fields
	PipelineID *int
	// Find issue where principalID is either creator, assignee or subscriber
	PrincipalID *int
	StatusList  *[]IssueStatus
	// If specified, then it will only fetch "Limit" most recently updated issues
	Limit *int
}

func (find *IssueFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type IssuePatch struct {
	ID int `jsonapi:"primary,issuePatch"`

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Name *string `jsonapi:"attr,name"`
	// Status is only set manualy via IssueStatusPatch
	Status      *IssueStatus
	Description *string `jsonapi:"attr,description"`
	AssigneeID  *int    `jsonapi:"attr,assigneeID"`
	Payload     *string `jsonapi:"attr,payload"`
}

type IssueStatusPatch struct {
	ID int `jsonapi:"primary,issueStatusPatch"`

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Status  IssueStatus `jsonapi:"attr,status"`
	Comment string      `jsonapi:"attr,comment"`
}

type IssueService interface {
	CreateIssue(ctx context.Context, create *IssueCreate) (*Issue, error)
	FindIssueList(ctx context.Context, find *IssueFind) ([]*Issue, error)
	FindIssue(ctx context.Context, find *IssueFind) (*Issue, error)
	PatchIssue(ctx context.Context, patch *IssuePatch) (*Issue, error)
}
