package common

type VCSType string

const (
	GITLAB_SELF_HOST VCSType = "GITLAB_SELF_HOST"
)

func (e VCSType) String() string {
	switch e {
	case GITLAB_SELF_HOST:
		return "GITLAB_SELF_HOST"
	}
	return "UNKNOWN"
}

// These payload types are only used when marshalling to the json format for saving into the database.
// So we annotate with json tag using camelCase naming which is consistent with normal
// json naming convention
type VCSFileCommit struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Message    string `json:"message"`
	CreatedTs  int64  `json:"createdTs"`
	URL        string `json:"url"`
	AuthorName string `json:"authorName"`
	Added      string `json:"added"`
}

type VCSPushEvent struct {
	VCSType            VCSType       `json:"vcsType"`
	BaseDirectory      string        `json:"baseDir"`
	Ref                string        `json:"ref"`
	RepositoryID       string        `json:"repositoryID"`
	RepositoryURL      string        `json:"repositoryUrl"`
	RepositoryFullPath string        `json:"repositoryFullPath"`
	AuthorName         string        `json:"authorName"`
	FileCommit         VCSFileCommit `json:"fileCommit"`
}
