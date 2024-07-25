package github

type PullRequestEventType string

const (
	// A pull request was created.
	PullRequestEventOpened PullRequestEventType = "opened"
	// A pull request was closed. If merged is false in the webhook payload, the pull request was closed with unmerged commits. If merged is true in the webhook payload, the pull request was merged.
	PullRequestEventClosed PullRequestEventType = "closed"
	// A pull request's head branch was updated. For example, the head branch was updated from the base branch or new commits were pushed to the head branch.
	PullRequestEventSynchronize PullRequestEventType = "synchronize"
)

// PullRequestPushEvent is the json message for pull request push event.
// Docs: https://docs.github.com/en/webhooks/webhook-events-and-payloads#pull_request
type PullRequestPushEvent struct {
	Action PullRequestEventType `json:"action"`
	Number int                  `json:"number"`
	// PR close will also send webhook event with "closed" action, so we need to check the "merged" field in "pull_request".
	PullRequest EventPullRequest `json:"pull_request"`
}

type EventPullRequest struct {
	HTMLURL string      `json:"html_url"`
	Title   string      `json:"title"`
	Body    string      `json:"body"`
	Base    EventBranch `json:"base"`
	Head    EventBranch `json:"head"`
	Merged  bool        `json:"merged"`
}

type EventBranch struct {
	// The branch name, e.g. main.
	Ref string `json:"ref"`
	SHA string `json:"sha"`
}
