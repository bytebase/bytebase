package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

type githubClient struct {
	apiURL string
	token  string
	client *http.Client
}

type comment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
	User struct {
		ID int64 `json:"id"`
	} `json:"user"`
}

func newClient(apiURL, token string) *githubClient {
	return &githubClient{
		apiURL: apiURL,
		token:  token,
		client: &http.Client{},
	}
}

func (g *githubClient) createComment(repo, pr, msg string) error {
	body := map[string]string{
		"body": msg,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return errors.Wrap(err, "failed to marshal request body")
	}

	url := fmt.Sprintf("%s/repos/%s/issues/%s/comments", g.apiURL, repo, pr)
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.token))
	req.Header.Set("Accept", "application/vnd.github.raw+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := g.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return errors.Errorf("failed to create comment, status code: %d", resp.StatusCode)
	}

	return nil
}

func (g *githubClient) listComments(repo, pr string) ([]comment, error) {
	url := fmt.Sprintf("%s/repos/%s/issues/%s/comments", g.apiURL, repo, pr)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.token))
	req.Header.Set("Accept", "application/vnd.github.raw+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("failed to list comments: %s", string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	var comments []comment
	if err := json.Unmarshal(bodyBytes, &comments); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}

	return comments, nil
}

func (g *githubClient) updateComment(repo string, commentID int64, msg string) error {
	body := map[string]string{
		"body": msg,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return errors.Wrap(err, "failed to marshal request body")
	}

	url := fmt.Sprintf("%s/repos/%s/issues/comments/%d", g.apiURL, repo, commentID)
	req, err := http.NewRequest("PATCH", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.token))
	req.Header.Set("Accept", "application/vnd.github.raw+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := g.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("failed to update comment, status code: %d", resp.StatusCode)
	}

	return nil
}
