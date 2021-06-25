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
	Username     string  `jsonapi:"attr,username"`
	// Password is not returned to the client
	Password string
}

type InstanceCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Related fields
	EnvironmentId int `jsonapi:"attr,environmentId"`

	// Domain specific fields
	Name         string  `jsonapi:"attr,name"`
	Engine       db.Type `jsonapi:"attr,engine"`
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

// Instance migration status
type InstanceMigrationStatus string

const (
	InstanceMigrationUnknown  InstanceMigrationStatus = "UNKNOWN"
	InstanceMigrationOK       InstanceMigrationStatus = "OK"
	InstanceMigrationNotExist InstanceMigrationStatus = "NOT_EXIST"
)

func (e InstanceMigrationStatus) String() string {
	switch e {
	case InstanceMigrationUnknown:
		return "UNKNOWN"
	case InstanceMigrationOK:
		return "OK"
	case InstanceMigrationNotExist:
		return "NOT_EXIST"
	}
	return "UNKNOWN"
}

type InstanceMigration struct {
	Status InstanceMigrationStatus `jsonapi:"attr,status"`
	Error  string                  `jsonapi:"attr,error"`
}

type InstanceService interface {
	// CreateInstance should also create the * database and the admin data source.
	CreateInstance(ctx context.Context, create *InstanceCreate) (*Instance, error)
	FindInstanceList(ctx context.Context, find *InstanceFind) ([]*Instance, error)
	FindInstance(ctx context.Context, find *InstanceFind) (*Instance, error)
	PatchInstance(ctx context.Context, patch *InstancePatch) (*Instance, error)
}
