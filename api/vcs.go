package api

import (
	"encoding/json"

	"github.com/bytebase/bytebase/plugin/vcs"
)

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
	// For safety concern, we will no return secret, and all relevant logic should be dealed in the backend.
	Secret string
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
