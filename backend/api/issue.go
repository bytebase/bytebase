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
type IssueFieldId string

const (
	IssueFieldName           IssueFieldId = "1"
	IssueFieldStatus         IssueFieldId = "2"
	IssueFieldAssignee       IssueFieldId = "3"
	IssueFieldDescription    IssueFieldId = "4"
	IssueFieldProject        IssueFieldId = "5"
	IssueFieldSubscriberList IssueFieldId = "6"
	IssueFieldSql            IssueFieldId = "7"
	IssueFieldRollbackSql    IssueFieldId = "8"
)

type IssuePayload struct {
}

type Issue struct {
	ID int `jsonapi:"primary,issue"`

	// Standard fields
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

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
	Payload          IssuePayload `jsonapi:"attr,payload"`
}

type IssueCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Related fields
	ProjectId  int `jsonapi:"attr,projectId"`
	PipelineId int
	Pipeline   PipelineCreate `jsonapi:"attr,pipeline"`

	// Domain specific fields
	Name              string       `jsonapi:"attr,name"`
	Type              IssueType    `jsonapi:"attr,type"`
	Description       string       `jsonapi:"attr,description"`
	AssigneeId        int          `jsonapi:"attr,assigneeId"`
	SubscriberIdList  []int        `jsonapi:"attr,subscriberIdList"`
	Statement         string       `jsonapi:"attr,statement"`
	RollbackStatement string       `jsonapi:"attr,rollbackStatement"`
	Payload           IssuePayload `jsonapi:"attr,payload"`
}

type IssueFind struct {
	ID *int

	// Related fields
	ProjectId *int

	// Domain specific fields
	PipelineId *int
	// Find issue where principalId is either creator or assignee
	// TODO: Add subscriber support
	PrincipalId *int
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
	UpdaterId int

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
	UpdaterId int

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
