package api

import (
	"context"
	"encoding/json"
)

// Approval policy only controls updating schema on the existing database.
// For creating new database, Developer always requires manual approval, while Owner and DBA doesn't
type ApprovalPolicy string

const (
	// We name this way because we may add approval policy lying in between. (e.g. only DDL change requires manual approval)
	ManualApprovalNever  ApprovalPolicy = "MANUAL_APPROVAL_NEVER"
	ManualApprovalAlways ApprovalPolicy = "MANUAL_APPROVAL_ALWAYS"
)

func (e ApprovalPolicy) String() string {
	switch e {
	case ManualApprovalNever:
		return "MANUAL_APPROVAL_NEVER"
	case ManualApprovalAlways:
		return "MANUAL_APPROVAL_ALWAYS"
	}
	return "UNKNOWN"
}

type Environment struct {
	ID int `jsonapi:"primary,environment"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Name           string         `jsonapi:"attr,name"`
	Order          int            `jsonapi:"attr,order"`
	ApprovalPolicy ApprovalPolicy `jsonapi:"attr,approvalPolicy"`
}

type EnvironmentCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Domain specific fields
	Name           string         `jsonapi:"attr,name"`
	ApprovalPolicy ApprovalPolicy `jsonapi:"attr,approvalPolicy"`
}

type EnvironmentFind struct {
	ID *int

	// Standard fields
	RowStatus *RowStatus
}

func (find *EnvironmentFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type EnvironmentPatch struct {
	ID int `jsonapi:"primary,environmentPatch"`

	// Standard fields
	RowStatus *string `jsonapi:"attr,rowStatus"`
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int

	// Domain specific fields
	Name           *string `jsonapi:"attr,name"`
	Order          *int    `jsonapi:"attr,order"`
	ApprovalPolicy *string `jsonapi:"attr,approvalPolicy"`
}

type EnvironmentDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterId int
}

type EnvironmentService interface {
	CreateEnvironment(ctx context.Context, create *EnvironmentCreate) (*Environment, error)
	FindEnvironmentList(ctx context.Context, find *EnvironmentFind) ([]*Environment, error)
	FindEnvironment(ctx context.Context, find *EnvironmentFind) (*Environment, error)
	PatchEnvironment(ctx context.Context, patch *EnvironmentPatch) (*Environment, error)
}
