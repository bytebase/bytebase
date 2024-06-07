package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/webhook"
)

type provider struct {
	c     *http.Client
	token string
}

func newProvider(token string) *provider {
	return &provider{
		c:     &http.Client{},
		token: token,
	}
}

func ValidateToken(ctx context.Context, token string) error {
	return newProvider(token).listUsers(ctx, 1)
}

type listUsersResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

type lookupByEmailResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
	User  struct {
		ID string `json:"id"`
	} `json:"user"`
}

type conversationsOpenResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error"`
	Channel struct {
		ID string `json:"id"`
	} `json:"channel"`
}

type chatPostMessageResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

// https://api.slack.com/methods/users.list
func (p *provider) listUsers(ctx context.Context, limit int) error {
	q := url.Values{}
	q.Set("limit", fmt.Sprintf("%d", limit))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://slack.com/api/users.list", nil)
	if err != nil {
		return errors.Wrapf(err, "failed to new request")
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.c.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to send GET request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("received non-200 status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to read body")
	}
	var res listUsersResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return errors.Wrapf(err, "failed to unmarshal")
	}
	if !res.OK {
		return errors.Errorf("failed to list users, error: %v", res.Error)
	}

	return nil
}

// https://api.slack.com/methods/users.lookupByEmail
func (p *provider) lookupByEmail(ctx context.Context, email string) (string, error) {
	q := url.Values{}
	q.Set("email", email)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://slack.com/api/users.lookupByEmail", nil)
	if err != nil {
		return "", errors.Wrapf(err, "failed to new request")
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Add("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.c.Do(req)
	if err != nil {
		return "", errors.Wrapf(err, "failed to send GET request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("received non-200 status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read body")
	}
	var res lookupByEmailResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal")
	}
	if !res.OK {
		return "", errors.Errorf("failed to get user, error: %v", res.Error)
	}

	return res.User.ID, nil
}

// https://api.slack.com/methods/conversations.open
func (p *provider) openConversation(ctx context.Context, userID string) (string, error) {
	data := url.Values{}
	data.Set("users", userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/conversations.open", strings.NewReader(data.Encode()))
	if err != nil {
		return "", errors.Wrapf(err, "failed to new request")
	}
	req.Header.Add("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.c.Do(req)
	if err != nil {
		return "", errors.Wrapf(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("received non-200 status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read body")
	}
	var res conversationsOpenResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return "", errors.Wrapf(err, "failed to unmarshal")
	}
	if !res.OK {
		return "", errors.Errorf("failed to open conversation, error: %v", res.Error)
	}

	return res.Channel.ID, nil
}

// https://api.slack.com/methods/chat.postMessage
func (p *provider) chatPostMessage(ctx context.Context, channelID string, webhookContext webhook.Context) error {
	blocks := GetBlocks(webhookContext)
	blocksJSON, err := json.Marshal(blocks)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal blocks")
	}

	data := url.Values{}
	data.Set("channel", channelID)
	data.Set("blocks", string(blocksJSON))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://slack.com/api/chat.postMessage", strings.NewReader(data.Encode()))
	if err != nil {
		return errors.Wrapf(err, "failed to new request")
	}
	req.Header.Add("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.c.Do(req)
	if err != nil {
		return errors.Wrapf(err, "failed to send request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("received non-200 status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to read body")
	}
	var res chatPostMessageResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return errors.Wrapf(err, "failed to unmarshal")
	}
	if !res.OK {
		return errors.Errorf("failed to post message, error: %v", res.Error)
	}

	return nil
}
