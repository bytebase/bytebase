package api

import (
	"context"
	"encoding/json"

	"github.com/bytebase/bytebase/plugin/vcs"
)

// VCS is the API message for VCS.
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
	Name          string   `jsonapi:"attr,name"`
	Type          vcs.Type `jsonapi:"attr,type"`
	InstanceURL   string   `jsonapi:"attr,instanceUrl"`
	APIURL        string   `jsonapi:"attr,apiUrl"`
	ApplicationID string   `jsonapi:"attr,applicationId"`
	Secret        string   `jsonapi:"attr,secret"`
}

// VCSCreate is the API message for creating a VCS.
type VCSCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Domain specific fields
	Name        string   `jsonapi:"attr,name"`
	Type        vcs.Type `jsonapi:"attr,type"`
	InstanceURL string   `jsonapi:"attr,instanceUrl"`
	// APIURL derives from InstanceURL
	APIURL        string
	ApplicationID string `jsonapi:"attr,applicationId"`
	Secret        string `jsonapi:"attr,secret"`
}

// VCSFind is the API message for finding VCSs.
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

// VCSPatch is the API message for patching a VCS.
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

// VCSDelete is the API message for deleting a VCS.
type VCSDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterID int
}

// VCSService is the service for VCSs.
type VCSService interface {
	CreateVCS(ctx context.Context, create *VCSCreate) (*VCS, error)
	FindVCSList(ctx context.Context, find *VCSFind) ([]*VCS, error)
	FindVCS(ctx context.Context, find *VCSFind) (*VCS, error)
	PatchVCS(ctx context.Context, patch *VCSPatch) (*VCS, error)
	DeleteVCS(ctx context.Context, delete *VCSDelete) error
}
