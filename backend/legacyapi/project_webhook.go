package api

// ProjectWebhook is the API message for project webhooks.
type ProjectWebhook struct {
	ID int `jsonapi:"primary,projectWebhookMember"`

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

// ProjectWebhookTestResult is the test result of a project webhook.
type ProjectWebhookTestResult struct {
	Error string `jsonapi:"attr,error"`
}
