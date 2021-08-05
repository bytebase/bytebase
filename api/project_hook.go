package api

import (
	"context"
	"encoding/json"
)

type ProjectHook struct {
	ID int `jsonapi:"primary,projectWebhookMember"`

	// Standard fields
	CreatorId int
	Creator   *Principal `jsonapi:"attr,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterId int
	Updater   *Principal `jsonapi:"attr,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	// Just returns ProjectId since it always operates within the project context
	ProjectId int `jsonapi:"attr,projecId"`

	// Domain specific fields
	Name         string   `jsonapi:"attr,name"`
	URL          string   `jsonapi:"attr,url"`
	ActivityList []string `jsonapi:"attr,activityList"`
}

type ProjectHookCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorId int

	// Related fields
	ProjectId int

	// Domain specific fields
	Name         string   `jsonapi:"attr,name"`
	URL          string   `jsonapi:"attr,url"`
	ActivityList []string `jsonapi:"attr,activityList"`
}

type ProjectHookFind struct {
	ID *int

	// Related fields
	ProjectId    *int
	ActivityType *ActivityType
}

func (find *ProjectHookFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

type ProjectHookPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterId int

	// Domain specific fields
	Name         *string   `jsonapi:"attr,name"`
	URL          *string   `jsonapi:"attr,url"`
	ActivityList *[]string `jsonapi:"attr,activityList"`
}

type ProjectHookDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterId int
}

type ProjectHookService interface {
	CreateProjectHook(ctx context.Context, create *ProjectHookCreate) (*ProjectHook, error)
	FindProjectHookList(ctx context.Context, find *ProjectHookFind) ([]*ProjectHook, error)
	FindProjectHook(ctx context.Context, find *ProjectHookFind) (*ProjectHook, error)
	PatchProjectHook(ctx context.Context, patch *ProjectHookPatch) (*ProjectHook, error)
	DeleteProjectHook(ctx context.Context, delete *ProjectHookDelete) error
}
