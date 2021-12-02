package gitlab

import (
	"fmt"
	"io"
	"net/http"
)

const (
	// APIPath is the API path.
	APIPath = "api/v4"
	// SecretTokenLength is the length of secret token.
	SecretTokenLength = 16
)

// WebhookType is the gitlab webhook type.
type WebhookType string

const (
	// WebhookPush is the webhook type for push.
	WebhookPush WebhookType = "push"
)

func (e WebhookType) String() string {
	switch e {
	case WebhookPush:
		return "push"
	}
	return "UNKNOWN"
}

// WebhookInfo is the API message for webhook info.
type WebhookInfo struct {
	ID int `json:"id"`
}

// WebhookPost is the API message for webhook POST.
type WebhookPost struct {
	URL         string `json:"url"`
	SecretToken string `json:"token"`
	// This is set to true
	PushEvents bool `json:"push_events"`
	// For now, there is no native dry run DDL support in mysql/postgres. One may wonder if we could wrap the DDL
	// in a transaction and just not commit at the end, unfortunately there are side effects which are hard to control.
	// See https://www.postgresql.org/message-id/CAMsr%2BYGiYQ7PYvYR2Voio37YdCpp79j5S%2BcmgVJMOLM2LnRQcA%40mail.gmail.com
	// So we can't possibly display useful info when reviewing a MR, thus we don't enable this event.
	// Saying that, delivering a souding dry run solution would be great and hopefully we can achieve that one day.
	// MergeRequestsEvents  bool   `json:"merge_requests_events"`
	PushEventsBranchFilter string `json:"push_events_branch_filter"`
	// TODO(tianzhou): This is set to false, be lax to not enable_ssl_verification
	EnableSSLVerification bool `json:"enable_ssl_verification"`
}

// WebhookPut is the API message for webhook PUT.
type WebhookPut struct {
	URL                    string `json:"url"`
	PushEventsBranchFilter string `json:"push_events_branch_filter"`
}

// WebhookProject is the API message for webhook project.
type WebhookProject struct {
	ID       int    `json:"id"`
	WebURL   string `json:"web_url"`
	FullPath string `json:"path_with_namespace"`
}

// WebhookCommitAuthor is the API message for webhook commit author.
type WebhookCommitAuthor struct {
	Name string `json:"name"`
}

// WebhookCommit is the API message for webhook commit.
type WebhookCommit struct {
	ID        string              `json:"id"`
	Title     string              `json:"title"`
	Message   string              `json:"message"`
	Timestamp string              `json:"timestamp"`
	URL       string              `json:"url"`
	Author    WebhookCommitAuthor `json:"author"`
	AddedList []string            `json:"added"`
}

// WebhookPushEvent is the API message for webhook push event.
type WebhookPushEvent struct {
	ObjectKind WebhookType     `json:"object_kind"`
	Ref        string          `json:"ref"`
	AuthorName string          `json:"user_name"`
	Project    WebhookProject  `json:"project"`
	CommitList []WebhookCommit `json:"commits"`
}

// FileCommit is the API message for file commit.
type FileCommit struct {
	Branch        string `json:"branch"`
	Content       string `json:"content"`
	CommitMessage string `json:"commit_message"`
	LastCommitID  string `json:"last_commit_id,omitempty"`
}

// File is the API message for file.
type File struct {
	LastCommitID string `json:"last_commit_id"`
}

// POST sends a POST request.
func POST(instanceURL string, resourcePath string, token string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s/%s", instanceURL, APIPath, resourcePath)
	req, err := http.NewRequest("POST",
		url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to construct POST %v (%w)", url, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed POST %v (%w)", url, err)
	}

	return resp, nil
}

// GET sends a GET request.
func GET(instanceURL string, resourcePath string, token string) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s/%s", instanceURL, APIPath, resourcePath)
	req, err := http.NewRequest("GET",
		url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to construct GET %v (%w)", url, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed GET %v (%w)", url, err)
	}

	return resp, nil
}

// PUT sends a PUT request.
func PUT(instanceURL string, resourcePath string, token string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s/%s", instanceURL, APIPath, resourcePath)
	req, err := http.NewRequest("PUT",
		url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to construct PUT %v (%w)", url, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed PUT %v (%w)", url, err)
	}

	return resp, nil
}

// DELETE sends a DELETE request.
func DELETE(instanceURL string, resourcePath string, token string) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s/%s", instanceURL, APIPath, resourcePath)
	req, err := http.NewRequest("DELETE",
		url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to construct DELETE %v (%w)", url, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed DELETE %v (%w)", url, err)
	}

	return resp, nil
}
