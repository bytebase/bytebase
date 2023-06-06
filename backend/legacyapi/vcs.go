package api

import (
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/vcs"
)

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
