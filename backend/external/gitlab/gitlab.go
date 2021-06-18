package gitlab

import (
	"fmt"
	"io"
	"net/http"
)

const (
	ApiPath = "api/v4"
)

type GitLabWebhookType string

const (
	WebhookPush GitLabWebhookType = "push"
)

func (e GitLabWebhookType) String() string {
	switch e {
	case WebhookPush:
		return "push"
	}
	return "UNKNOWN"
}

type WebhookPut struct {
	URL                    string `json:"url"`
	PushEventsBranchFilter string `json:"push_events_branch_filter"`
}

type WebhookProject struct {
	ID       int    `json:"id"`
	WebURL   string `json:"web_url"`
	FullPath string `json:"path_with_namespace"`
}

type WebhookCommitAuthor struct {
	Name string `json:"name"`
}

type WebhookCommit struct {
	ID        string              `json:"id"`
	Title     string              `json:"title"`
	Message   string              `json:"message"`
	Timestamp string              `json:"timestamp"`
	URL       string              `json:"url"`
	Author    WebhookCommitAuthor `json:"author"`
	AddedList []string            `json:"added"`
}

type WebhookPushEvent struct {
	ObjectKind GitLabWebhookType `json:"object_kind"`
	Ref        string            `json:"ref"`
	AuthorName string            `json:"user_name"`
	Project    WebhookProject    `json:"project"`
	CommitList []WebhookCommit   `json:"commits"`
}

func GET(instanceURL string, resourcePath string, token string) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s/%s", instanceURL, ApiPath, resourcePath)
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

func PUT(instanceURL string, resourcePath string, token string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s/%s", instanceURL, ApiPath, resourcePath)
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

func DELETE(instanceURL string, resourcePath string, token string) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s/%s", instanceURL, ApiPath, resourcePath)
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
