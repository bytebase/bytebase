package api

import (
	"context"
	"encoding/json"
)

type VCSType string

const (
	GITLAB_SELF_HOST VCSType = "GITLAB_SELF_HOST"
)

func (e VCSType) String() string {
	switch e {
	case GITLAB_SELF_HOST:
		return "GITLAB_SELF_HOST"
	}
	return "UNKNOWN"
}

type VCS struct {
	ID int `jsonapi:"primary,vcs"`

	// Standard fields
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Domain specific fields
	Name          string  `jsonapi:"attr,name"`
	Type          VCSType `jsonapi:"attr,type"`
	InstanceURL   string  `jsonapi:"attr,instanceURL"`
	ApiURL        string  `jsonapi:"attr,apiURL"`
	ApplicationId string  `jsonapi:"attr,applicationId"`
	Secret        string  `jsonapi:"attr,secret"`
	// These will be exclusively used on the server side (e.g. clean up orphaned webhooks) and we don't return it to the client.
	AccessToken  string
	ExpiresTs    int64
	RefreshToken string
}

type VCSCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Domain specific fields
	Name        string  `jsonapi:"attr,name"`
	Type        VCSType `jsonapi:"attr,type"`
	InstanceURL string  `jsonapi:"attr,instanceURL"`
	// ApiURL derives from InstanceURL
	ApiURL        string
	ApplicationId string `jsonapi:"attr,applicationId"`
	Secret        string `jsonapi:"attr,secret"`
	AccessToken   string `jsonapi:"attr,accessToken"`
	ExpiresTs     int64  `jsonapi:"attr,expiresTs"`
	RefreshToken  string `jsonapi:"attr,refreshToken"`
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
	UpdaterId int

	// Domain specific fields
	Name          *string `jsonapi:"attr,name"`
	ApplicationId *string `jsonapi:"attr,applicationId"`
	Secret        *string `jsonapi:"attr,secret"`
	AccessToken   *string `jsonapi:"attr,accessToken"`
	ExpiresTs     *int64  `jsonapi:"attr,expiresTs"`
	RefreshToken  *string `jsonapi:"attr,refreshToken"`
}

type VCSDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterId int
}

// These payload types are only used when marshalling to the json format for saving into the database.
// So we annotate with json tag using camelCase naming which is consistent with normal
// json naming convention
type VCSFileCommit struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Message    string `json:"message"`
	CreatedTs  int64  `json:"createdTs"`
	URL        string `json:"url"`
	AuthorName string `json:"authorName"`
	Added      string `json:"added"`
}

type VCSPushEvent struct {
	Ref                string        `json:"ref"`
	RepositoryID       string        `json:"repoId"`
	RepositoryURL      string        `json:"repoUrl"`
	RepositoryFullPath string        `json:"repoFullPath"`
	AuthorName         string        `json:"authorName"`
	FileCommit         VCSFileCommit `json:"fileCommit"`
}

type VCSService interface {
	CreateVCS(ctx context.Context, create *VCSCreate) (*VCS, error)
	FindVCSList(ctx context.Context, find *VCSFind) ([]*VCS, error)
	FindVCS(ctx context.Context, find *VCSFind) (*VCS, error)
	PatchVCS(ctx context.Context, patch *VCSPatch) (*VCS, error)
	DeleteVCS(ctx context.Context, delete *VCSDelete) error
}
