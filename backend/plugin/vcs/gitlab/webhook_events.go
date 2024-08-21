package gitlab

// MergeRequestPushEvent is the json message for merge request push event.
// Docs: https://docs.gitlab.com/ee/user/project/integrations/webhook_events.html#merge-request-events
type MergeRequestPushEvent struct {
	ObjectKind       string                `json:"object_kind"`
	User             EventUser             `json:"user"`
	ObjectAttributes EventObjectAttributes `json:"object_attributes"`
}

type EventUser struct {
	Email string `json:"email"`
}

// The available values for object_attributes.action in the payload are:
// open, close, reopen, update, approved, unapproved, approval, unapproval, merge.
// Docs: https://docs.gitlab.com/ee/user/project/integrations/webhook_events.html#merge-request-events
type MergeRequestAction string

const (
	MergeRequestOpen   MergeRequestAction = "open"
	MergeRequestUpdate MergeRequestAction = "update"
	MergeRequestMerge  MergeRequestAction = "merge"
)

type EventObjectAttributes struct {
	IID          int                `json:"iid"`
	URL          string             `json:"url"`
	TargetBranch string             `json:"target_branch"`
	Action       MergeRequestAction `json:"action"`
	Title        string             `json:"title"`
	Description  string             `json:"description"`
	LastCommit   LastCommit         `json:"last_commit"`
}

type LastCommit struct {
	ID string `json:"id"`
}
