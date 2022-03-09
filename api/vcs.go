package api

import (
	"context"
	"encoding/json"

	"github.com/bytebase/bytebase/plugin/vcs"
)

// VCSRaw is the store model for a VCS (Version Control System).
// Fields have exactly the same meanings as VCS.
type VCSRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Domain specific fields
	Name          string
	Type          vcs.Type
	InstanceURL   string
	APIURL        string
	ApplicationID string
	Secret        string
}

// ToVCS creates an instance of VCS based on the VCSRaw.
// This is intended to be called when we need to compose a VCS relationship.
func (raw *VCSRaw) ToVCS() *VCS {
	return &VCS{
		ID: raw.ID,

		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		Name:          raw.Name,
		Type:          raw.Type,
		InstanceURL:   raw.InstanceURL,
		APIURL:        raw.APIURL,
		ApplicationID: raw.ApplicationID,
		Secret:        raw.Secret,
	}
}

// VCS is the API message for a VCS (Version Control System).
type VCS struct {
	ID int `jsonapi:"primary,vcs"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Name          string   `jsonapi:"attr,name"`
	Type          vcs.Type `jsonapi:"attr,type"`
	InstanceURL   string   `jsonapi:"attr,instanceUrl"`
	APIURL        string   `jsonapi:"attr,apiUrl"`
	ApplicationID string   `jsonapi:"attr,applicationId"`
	// Secret will be used for OAuth on the client side when setting up project GitOps workflow.
	// So it should be returned to the response.
	Secret string `jsonapi:"attr,secret"`
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
	CreateVCS(ctx context.Context, create *VCSCreate) (*VCSRaw, error)
	FindVCSList(ctx context.Context, find *VCSFind) ([]*VCSRaw, error)
	FindVCS(ctx context.Context, find *VCSFind) (*VCSRaw, error)
	PatchVCS(ctx context.Context, patch *VCSPatch) (*VCSRaw, error)
	DeleteVCS(ctx context.Context, delete *VCSDelete) error
}
