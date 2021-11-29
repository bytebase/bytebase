package api

import (
	"context"
	"encoding/json"

	"github.com/bytebase/bytebase/common"
)

type VCS struct {
	ID int `jsonapi:"primary,vcs"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Name          string         `jsonapi:"attr,name"`
	Type          common.VCSType `jsonapi:"attr,type"`
	InstanceURL   string         `jsonapi:"attr,instanceURL"`
	ApiURL        string         `jsonapi:"attr,apiURL"`
	ApplicationID string         `jsonapi:"attr,applicationId"`
	Secret        string         `jsonapi:"attr,secret"`
}

type VCSCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Domain specific fields
	Name        string         `jsonapi:"attr,name"`
	Type        common.VCSType `jsonapi:"attr,type"`
	InstanceURL string         `jsonapi:"attr,instanceURL"`
	// ApiURL derives from InstanceURL
	ApiURL        string
	ApplicationID string `jsonapi:"attr,applicationId"`
	Secret        string `jsonapi:"attr,secret"`
}

type VCSFind struct {
	ID *int
}

func (find *VCSFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type VCSPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Name          *string `jsonapi:"attr,name"`
	ApplicationID *string `jsonapi:"attr,applicationId"`
	Secret        *string `jsonapi:"attr,secret"`
}

type VCSDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterID int
}

type VCSService interface {
	CreateVCS(ctx context.Context, create *VCSCreate) (*VCS, error)
	FindVCSList(ctx context.Context, find *VCSFind) ([]*VCS, error)
	FindVCS(ctx context.Context, find *VCSFind) (*VCS, error)
	PatchVCS(ctx context.Context, patch *VCSPatch) (*VCS, error)
	DeleteVCS(ctx context.Context, delete *VCSDelete) error
}
