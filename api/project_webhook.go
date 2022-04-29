package api

import (
	"encoding/json"
)

// ProjectWebhook is the API message for project webhooks.
type ProjectWebhook struct {
	ID int `jsonapi:"primary,projectWebhookMember"`

	// Standard fields
	CreatorID int
	Creator   *Principal `jsonapi:"relation,creator"`
	CreatedTs int64      `jsonapi:"attr,createdTs"`
	UpdaterID int
	Updater   *Principal `jsonapi:"relation,updater"`
	UpdatedTs int64      `jsonapi:"attr,updatedTs"`

	// Related fields
	// Just returns ProjectID since it always operates within the project context
	ProjectID int `jsonapi:"attr,projectId"`

	// Domain specific fields
	Type         string   `jsonapi:"attr,type"`
	Name         string   `jsonapi:"attr,name"`
	URL          string   `jsonapi:"attr,url"`
	ActivityList []string `jsonapi:"attr,activityList"`
}

// ProjectWebhookCreate is the API message for creating a project webhook.
type ProjectWebhookCreate struct {
	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	CreatorID int

	// Related fields
	ProjectID int

	// Domain specific fields
	Type         string   `jsonapi:"attr,type"`
	Name         string   `jsonapi:"attr,name"`
	URL          string   `jsonapi:"attr,url"`
	ActivityList []string `jsonapi:"attr,activityList"`
}

// ProjectWebhookFind is the API message for finding project webhooks.
type ProjectWebhookFind struct {
	ID *int

	// Related fields
	ProjectID    *int
	ActivityType *ActivityType
}

func (find *ProjectWebhookFind) String() string {
	str, err := json.Marshal(*find)
	if err != nil {
		return err.Error()
	}
	return string(str)
}

// ProjectWebhookPatch is the API message for patching a project webhook.
type ProjectWebhookPatch struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	UpdaterID int

	// Domain specific fields
	Name         *string `jsonapi:"attr,name"`
	URL          *string `jsonapi:"attr,url"`
	ActivityList *string `jsonapi:"attr,activityList"`
}

// ProjectWebhookDelete is the API message for deleting a project webhook.
type ProjectWebhookDelete struct {
	ID int

	// Standard fields
	// Value is assigned from the jwt subject field passed by the client.
	DeleterID int
}

// ProjectWebhookTestResult is the test result of a project webhook.
type ProjectWebhookTestResult struct {
	Error string `jsonapi:"attr,error"`
}
