package api

import (
	"encoding/json"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
)

// VCS is the API message for a VCS (Version Control System).
type VCS struct {
	ID int `jsonapi:"primary,vcs"`

	// Domain specific fields
	Name          string   `jsonapi:"attr,name"`
	Type          vcs.Type `jsonapi:"attr,type"`
	InstanceURL   string   `jsonapi:"attr,instanceUrl"`
	APIURL        string   `jsonapi:"attr,apiUrl"`
	ApplicationID string   `jsonapi:"attr,applicationId"`
	// For safety concerns, we will not return the secret, and all relevant logic is dealt in the backend.
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

// ExternalRepository is the API message for external repository.
type ExternalRepository struct {
	ID       int64  `jsonapi:"primary,id"`
	Name     string `jsonapi:"attr,name"`
	FullPath string `jsonapi:"attr,fullPath"`
	WebURL   string `jsonapi:"attr,webUrl"`
}

// VCSExchangeToken is the API message of exchanging token for a VCS.
type VCSExchangeToken struct {
	Code         string   `jsonapi:"attr,code"`
	ID           int      `jsonapi:"attr,vcsId"`
	Type         vcs.Type `jsonapi:"attr,vcsType"`
	InstanceURL  string   `jsonapi:"attr,instanceUrl"`
	ClientID     string   `jsonapi:"attr,clientId"`
	ClientSecret string   `jsonapi:"attr,clientSecret"`
}

// VCSSQLReviewResult the is SQL review result in VCS workflow.
type VCSSQLReviewResult struct {
	Status  advisor.Status `json:"status"`
	Content []string       `json:"content"`
}

// VCSSQLReviewRequest is the request from SQL review CI in VCS workflow.
// In the VCS SQL review workflow, the CI will generate the request body then POST /hook/sql-review/:webhook_endpoint_id.
type VCSSQLReviewRequest struct {
	RepositoryID  string `json:"repositoryId"`
	PullRequestID string `json:"pullRequestId"`
	// WebURL is the server URL for GitOps CI.
	// In GitHub, the URL should be "https://github.com". Docs: https://docs.github.com/en/actions/learn-github-actions/environment-variables
	// In GitLab, the URL should be the base URL of the GitLab instance like "https://gitlab.bytebase.com". Docs: https://docs.gitlab.com/ee/ci/variables/predefined_variables.html
	WebURL string `json:"webURL"`
}
