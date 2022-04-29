package api

import (
	"encoding/json"
)

// InstanceUser is the API message for instance user.
type InstanceUser struct {
	ID int `jsonapi:"primary,instanceUser"`

	// Related fields
	InstanceID int `jsonapi:"attr,instanceId"`

	// Domain specific fields
	Name  string `jsonapi:"attr,name"`
	Grant string `jsonapi:"attr,grant"`
}

// InstanceUserUpsert is the API message for upserting an instance user.
type InstanceUserUpsert struct {
	// Standard fields
	CreatorID int

	// Related fields
	InstanceID int

	// Domain specific fields
	Name  string `jsonapi:"attr,name"`
	Grant string `jsonapi:"attr,grant"`
}

// InstanceUserFind is the API message for finding instance users.
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

// InstanceUserDelete is the API message for deleting an instance user.
type InstanceUserDelete struct {
	ID int
}
