package github

// PullRequestPushEvent is the json message for pull request push event.
type PullRequestPushEvent struct {
	// opened, edited, closed.
	Action      string           `json:"action"`
	Number      int              `json:"number"`
	PullRequest EventPullRequest `json:"pull_request"`
}

type EventPullRequest struct {
	HTMLURL string      `json:"html_url"`
	Title   string      `json:"title"`
	Body    string      `json:"body"`
	Base    EventBranch `json:"base"`
	Head    EventBranch `json:"head"`
}

type EventBranch struct {
	// The branch name, e.g. main.
	Ref string `json:"ref"`
	SHA string `json:"sha"`
}
