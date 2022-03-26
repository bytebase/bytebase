package api

import (
	"encoding/json"
)

// SystemBotID is the ID of the system robot.
const SystemBotID = 1

// PrincipalType is the type of a principal.
type PrincipalType string

const (
	// EndUser is the principal type for END_USER.
	EndUser PrincipalType = "END_USER"
	// BOT is the principal type for BOT.
	BOT PrincipalType = "BOT"
)

func (e PrincipalType) String() string {
	switch e {
	case EndUser:
		return "END_USER"
	case BOT:
		return "BOT"
	}
	return ""
}

// PrincipalAuthProvider is the type of an authentication provider.
type PrincipalAuthProvider string

const (
	// PrincipalAuthProviderBytebase is the Bytebase's own authentication provider.
	PrincipalAuthProviderBytebase PrincipalAuthProvider = "BYTEBASE"
	// PrincipalAuthProviderGitlabSelfHost is the self-hosted GitLab authentication provider.
	PrincipalAuthProviderGitlabSelfHost PrincipalAuthProvider = "GITLAB_SELF_HOST"
)

// Principal is the API message for principals.
type Principal struct {
	ID int `jsonapi:"primary,principal"`

	// Standard fields
	CreatorID int   `jsonapi:"attr,creatorId"`
	CreatedTs int64 `jsonapi:"attr,createdTs"`
	UpdaterID int   `jsonapi:"attr,updaterId"`
	UpdatedTs int64 `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Type  PrincipalType `jsonapi:"attr,type"`
	Name  string        `jsonapi:"attr,name"`
	Email string        `jsonapi:"attr,email"`
	// Do not return to the client
	PasswordHash string
	// Role is stored in the member table, but we include it when returning the principal.
	// This simplifies the client code where it won't require order dependency to fetch the related member info first.
	Role Role `jsonapi:"attr,role"`
}

// MarshalJSON customizes the Principal Marshal method so the returned object
// can map directly to the frontend Principal object without any conversion.
func (p *Principal) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID        int           `json:"id"`
		CreatorID int           `json:"creatorId"`
		CreatedTs int64         `json:"createdTs"`
		UpdaterID int           `json:"updaterId"`
		UpdatedTs int64         `json:"updatedTs"`
		Type      PrincipalType `json:"type"`
		Name      string        `json:"name"`
		Email     string        `json:"email"`
		Role      Role          `json:"role"`
	}{
		ID:        p.ID,
		CreatorID: p.CreatorID,
		CreatedTs: p.CreatedTs,
		UpdaterID: p.UpdaterID,
		UpdatedTs: p.UpdatedTs,
		Type:      p.Type,
		Name:      p.Name,
		Email:     p.Email,
		Role:      p.Role,
	})
}

// PrincipalCreate is the API message for creating a principal.
type PrincipalCreate struct {
	// Standard fields
	// For signup, value is SYSTEM_BOT_ID
	// For invite, value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Domain specific fields
	Type         PrincipalType
	Name         string `jsonapi:"attr,name"`
	Email        string `jsonapi:"attr,email"`
	Password     string `jsonapi:"attr,password"`
	PasswordHash string
}

// PrincipalFind is the API message for finding principals.
type PrincipalFind struct {
	ID *int

	// Domain specific fields
	Email *string
}

func (find *PrincipalFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// PrincipalPatch is the API message for patching a principal.
type PrincipalPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Name         *string `jsonapi:"attr,name"`
	Password     *string `jsonapi:"attr,password"`
	PasswordHash *string
}
