package gitlab

// MergeRequestPushEvent is the json message for merge request push event.
type MergeRequestPushEvent struct {
	ObjectKind       string                `json:"object_kind"`
	User             EventUser             `json:"user"`
	ObjectAttributes EventObjectAttributes `json:"object_attributes"`
}

type EventUser struct {
	Email string `json:"email"`
}

type EventObjectAttributes struct {
	IID          int    `json:"iid"`
	URL          string `json:"url"`
	TargetBranch string `json:"target_branch"`
	// open, merge.
	Action      string     `json:"action"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	LastCommit  LastCommit `json:"last_commit"`
}

type LastCommit struct {
	ID string `json:"id"`
}
