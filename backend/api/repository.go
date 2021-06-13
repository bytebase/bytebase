package api

import (
	"context"
	"encoding/json"
)

type Repository struct {
	ID int `jsonapi:"primary,repository"`

	// Standard fields
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	VCSId     int
	VCS       *VCS `jsonapi:"relation,vcs"`
	ProjectId int
	Project   *Project `jsonapi:"relation,project"`

	// Domain specific fields
	Name          string `jsonapi:"attr,name"`
	FullPath      string `jsonapi:"attr,fullPath"`
	WebURL        string `jsonapi:"attr,webURL"`
	BaseDirectory string `jsonapi:"attr,baseDirectory"`
	BranchFilter  string `jsonapi:"attr,branchFilter"`
	ExternalId    string `jsonapi:"attr,externalId"`
	WebhookId     string
	WebhookURL    string
}

type RepositoryCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Related fields
	VCSId     int `jsonapi:"attr,vcsId"`
	ProjectId int `jsonapi:"attr,projectId"`

	// Domain specific fields
	Name          string `jsonapi:"attr,name"`
	FullPath      string `jsonapi:"attr,fullPath"`
	WebURL        string `jsonapi:"attr,webURL"`
	BaseDirectory string `jsonapi:"attr,baseDirectory"`
	BranchFilter  string `jsonapi:"attr,branchFilter"`
	ExternalId    string `jsonapi:"attr,externalId"`
	WebhookId     string `jsonapi:"attr,webhookId"`
	WebhookURL    string `jsonapi:"attr,webhookURL"`
}

type RepositoryFind struct {
	ID *int

	// Related fields
	VCSId     *int
	ProjectId *int
}

func (find *RepositoryFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type RepositoryPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int

	// Domain specific fields
	BaseDirectory *string `jsonapi:"attr,baseDirectory"`
	BranchFilter  *string `jsonapi:"attr,branchFilter"`
}

type RepositoryDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterId int
}

type RepositoryService interface {
	CreateRepository(ctx context.Context, create *RepositoryCreate) (*Repository, error)
	FindRepositoryList(ctx context.Context, find *RepositoryFind) ([]*Repository, error)
	FindRepository(ctx context.Context, find *RepositoryFind) (*Repository, error)
	PatchRepository(ctx context.Context, patch *RepositoryPatch) (*Repository, error)
	DeleteRepository(ctx context.Context, delete *RepositoryDelete) error
}
