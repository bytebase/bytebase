package api

import (
	"context"
)

type Policy struct {
	ID int `jsonapi:"primary,environment"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	EnvironmentId int `jsonapi:"attr,environmentId"`
	// Do not return this to the client since the client always has the database context and fetching the
	// database object and all its own related objects is a bit expensive.
	Environment *Environment

	// Domain specific fields
	Type    string `jsonapi:"attr,type"`
	Payload string `jsonapi:"attr,payload"`
}

// PolicyFind is the message to get a policy.
type PolicyFind struct {
	ID *int

	// Related fields
	EnvironmentId *int

	// Domain specific fields
	Type *string `jsonapi:"attr,type"`
}

// PolicyUpsert is the message to upsert a policy.
// NOTE: We use PATCH for Upsert, this is inspired by https://google.aip.dev/134#patch-and-put
type PolicyUpsert struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	// CreatorId is the ID of the creator.
	UpdaterId int

	// Related fields
	EnvironmentId int `jsonapi:"attr,environmentId"`

	// Domain specific fields
	Type    string `jsonapi:"attr,type"`
	Payload string `jsonapi:"attr,payload"`
}

// PolicyService is the backend for policies.
type PolicyService interface {
	FindPolicy(ctx context.Context, find *PolicyFind) (*Policy, error)
	UpsertPolicy(ctx context.Context, upsert *PolicyUpsert) (*Policy, error)
}
