package bitbucket

// Header X-Event-Key: pullrequest:created, pullrequest:updated, pullrequest:fulfilled.

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
