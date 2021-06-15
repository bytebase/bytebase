package api

import (
	"context"
	"encoding/json"

	"github.com/bytebase/bytebase/db"
)

type Instance struct {
	ID int `jsonapi:"primary,instance"`

	// Standard fields
	RowStatus RowStatus `jsonapi:"attr,rowStatus"`
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	EnvironmentId int
	Environment   *Environment `jsonapi:"relation,environment"`

	// Domain specific fields
	Name         string  `jsonapi:"attr,name"`
	Engine       db.Type `jsonapi:"attr,engine"`
	ExternalLink string  `jsonapi:"attr,externalLink"`
	Host         string  `jsonapi:"attr,host"`
	Port         string  `jsonapi:"attr,port"`
	// Only returns username/password if supplying query parameter 'include=secret'
	Username string `jsonapi:"attr,username"`
	Password string `jsonapi:"attr,password"`
}

type InstanceCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Related fields
	EnvironmentId int `jsonapi:"attr,environmentId"`

	// Domain specific fields
	Name         string  `jsonapi:"attr,name"`
	Type         db.Type `jsonapi:"attr,type"`
	ExternalLink string  `jsonapi:"attr,externalLink"`
	Host         string  `jsonapi:"attr,host"`
	Port         string  `jsonapi:"attr,port"`
	Username     string  `jsonapi:"attr,username"`
	Password     string  `jsonapi:"attr,password"`
}

type InstanceFind struct {
	ID *int

	// Standard fields
	RowStatus *RowStatus
}

func (find *InstanceFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type InstancePatch struct {
	ID int `jsonapi:"primary,instancePatch"`

	// Standard fields
	RowStatus *string `jsonapi:"attr,rowStatus"`
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int

	// Domain specific fields
	Name         *string `jsonapi:"attr,name"`
	ExternalLink *string `jsonapi:"attr,externalLink"`
	Host         *string `jsonapi:"attr,host"`
	Port         *string `jsonapi:"attr,port"`
	Username     *string `jsonapi:"attr,username"`
	Password     *string `jsonapi:"attr,password"`
}

type InstanceService interface {
	CreateInstance(ctx context.Context, create *InstanceCreate) (*Instance, error)
	FindInstanceList(ctx context.Context, find *InstanceFind) ([]*Instance, error)
	FindInstance(ctx context.Context, find *InstanceFind) (*Instance, error)
	PatchInstance(ctx context.Context, patch *InstancePatch) (*Instance, error)
}
