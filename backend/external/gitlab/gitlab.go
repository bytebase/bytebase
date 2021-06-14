package gitlab

import (
	"fmt"
	"io"
	"net/http"
)

const (
	ApiPath = "/api/v4"
)

type WebhookPut struct {
	URL                    string `json:"url"`
	PushEventsBranchFilter string `json:"push_events_branch_filter"`
}

func PUT(instanceURL string, resourcePath string, token string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s%s%s", instanceURL, ApiPath, resourcePath)
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
	url := fmt.Sprintf("%s%s%s", instanceURL, ApiPath, resourcePath)
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
