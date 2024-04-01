package azure

// PullRequestEventLastMergeCommit is the API message for PR merge commit.
type PullRequestEventLastMergeCommit struct {
	CommitID string `json:"commitId"`
	URL      string `json:"url"`
}

type PullRequestCreatedBy struct {
	DisplayName string `json:"displayName"`
	UniqueName  string `json:"uniqueName"`
}

type PullRequestWeb struct {
	Href string `json:"href"`
}

type PullRequestLinks struct {
	Web *PullRequestWeb `json:"web"`
}

// PullRequestResource is the API message for pull request.
type PullRequestResource struct {
	Repository    *Repository       `json:"repository"`
	Links         *PullRequestLinks `json:"_links"`
	PullRequestID int               `json:"pullRequestId"`
	// The pull request status, could be active, completed, etc. We only care the "completed" status.
	Status        string `json:"status"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	SourceRefName string `json:"sourceRefName"`
	TargetRefName string `json:"targetRefName"`
	// PR merge status, we only care the "succeeded".
	MergeStatus     string                           `json:"mergeStatus"`
	LastMergeCommit *PullRequestEventLastMergeCommit `json:"lastMergeCommit"`
	CreatedBy       *PullRequestCreatedBy            `json:"createdBy"`
}

type PullRequestMessage struct {
	Text string `json:"text"`
}

// PullRequestEvent is the API message for pull request webhook event.
//
// Docs: https://learn.microsoft.com/en-us/azure/devops/service-hooks/events?view=azure-devops#pull-request-merge-commit-created
type PullRequestEvent struct {
	// ID is the webhook message id.
	ID string `json:"id"`
	// EventType should be "git.pullrequest.merged".
	EventType       string               `json:"eventType"`
	Message         *PullRequestMessage  `json:"message"`
	DetailedMessage *PullRequestMessage  `json:"detailedMessage"`
	Resource        *PullRequestResource `json:"resource"`
}

// WebhookCreateConsumerInputs represents the consumer inputs for creating a webhook.
type WebhookCreateConsumerInputs struct {
	URL                  string `json:"url"`
	AcceptUntrustedCerts bool   `json:"acceptUntrustedCerts"`
	HTTPHeaders          string `json:"httpHeaders"`
}

// WebhookMergeResult is the status for pull request merge result.
type WebhookMergeResult string

// WebhookMergeResultSucceeded is succeeded status for merge result.
const WebhookMergeResultSucceeded WebhookMergeResult = "Succeeded"

// WebhookCreatePublisherInputs represents the publisher inputs for creating a webhook.
type WebhookCreatePublisherInputs struct {
	Repository string `json:"repository"`
	// The target branch for PR without "refs/heads/" prefix.
	Branch string `json:"branch"`
	// The merge result for PR, we only need the "Succeeded".
	MergeResult WebhookMergeResult `json:"mergeResult"`
	ProjectID   string             `json:"projectId"`
}

// WebhookCreateOrUpdate represents a Bitbucket API request for creating or updating a webhook.
type WebhookCreateOrUpdate struct {
	ConsumerActionID string                       `json:"consumerActionId"`
	ConsumerID       string                       `json:"consumerId"`
	ConsumerInputs   WebhookCreateConsumerInputs  `json:"consumerInputs"`
	EventType        string                       `json:"eventType"`
	PublisherID      string                       `json:"publisherId"`
	PublisherInputs  WebhookCreatePublisherInputs `json:"publisherInputs"`
}
