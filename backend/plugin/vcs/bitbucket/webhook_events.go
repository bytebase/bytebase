package bitbucket

// Header X-Event-Key: pullrequest:created, pullrequest:updated, pullrequest:fulfilled.
//
// https://support.atlassian.com/bitbucket-cloud/docs/event-payloads
type PullRequestEventType string

const (
	// A pull request is created in a Git repository.
	PullRequestEventCreated PullRequestEventType = "pullrequest:created"
	// A pull request is updated.
	PullRequestEventUpdated PullRequestEventType = "pullrequest:updated"
	// A pull request is merged.
	PullRequestEventFulfilled PullRequestEventType = "pullrequest:fulfilled"
)

// PullRequestPushEvent is the json message for pull request push event.
type PullRequestPushEvent struct {
	// Actor: does not include email.
	PullRequest EventPullRequest `json:"pullrequest"`
}

type EventPullRequest struct {
	ID          int         `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Destination EventBranch `json:"destination"`
	Source      EventBranch `json:"source"`
	Links       EventLinks  `json:"links"`
}

type EventBranch struct {
	Branch EventBranchName `json:"branch"`
	Commit EventCommit     `json:"commit"`
}

type EventBranchName struct {
	Name string `json:"name"`
}

type EventCommit struct {
	Hash string `json:"hash"`
}

type EventLinks struct {
	HTML EventHTML `json:"html"`
}

type EventHTML struct {
	Href string `json:"href"`
}
