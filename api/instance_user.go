package api

import (
	"context"
	"encoding/json"
)

type InstanceUser struct {
	ID int `jsonapi:"primary,instanceUser"`

	// Related fields
	InstanceID int `jsonapi:"attr,instanceId"`

	// Domain specific fields
	Name  string `jsonapi:"attr,name"`
	Grant string `jsonapi:"attr,grant"`
}

type InstanceUserUpsert struct {
	// Standard fields
	CreatorID int

	// Related fields
	InstanceID int

	// Domain specific fields
	Name  string `jsonapi:"attr,name"`
	Grant string `jsonapi:"attr,grant"`
}

type InstanceUserFind struct {
	InstanceID int
}

func (find *InstanceUserFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type InstanceUserDelete struct {
	ID int
}

type InstanceUserService interface {
	// UpsertInstanceUser would update the existing user if name matches.
	UpsertInstanceUser(ctx context.Context, upsert *InstanceUserUpsert) (*InstanceUser, error)
	FindInstanceUserList(ctx context.Context, find *InstanceUserFind) ([]*InstanceUser, error)
	DeleteInstanceUser(ctx context.Context, delete *InstanceUserDelete) error
}
