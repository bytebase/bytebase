// Package relay is the client sending requests to relay server for external approvals.
package relay

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Client is the client for relay.
type Client struct {
	client *http.Client
}

// NewClient returns a client.
func NewClient() *Client {
	return &Client{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// CreatePayload is the message to create external approval.
type CreatePayload struct {
	IssueID     string    `json:"issueId"`
	Creator     string    `json:"creator"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Project     string    `json:"project"`
	Assignee    string    `json:"assignee"`
	Statement   string    `json:"statement"`
	CreateTime  time.Time `json:"createTime"`
}

// Status is the status of the external approval.
type Status string

const (
	// StatusApproved means that the external approval is approved.
	StatusApproved Status = "APPROVED"
	// StatusRejected means that the external approval is rejected.
	StatusRejected Status = "REJECTED"
)

// ResponsePayload is the response message to for external approval.
type ResponsePayload struct {
	ID     string `json:"id"`
	Status Status `json:"status"`
}

// Create sends a message to create external approval.
func (c *Client) Create(relayEndpoint string, payload *CreatePayload) (string, error) {
	out, err := json.Marshal(&payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/approval", relayEndpoint), bytes.NewBuffer(out))
	if err != nil {
		return "", err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("failed to create external approval with status %v", resp.Status)
	}
	responseBody := &ResponsePayload{}
	err = json.NewDecoder(resp.Body).Decode(responseBody)
	if err != nil {
		return "", err
	}
	return responseBody.ID, nil
}

// UpdatePayload is the message to update external approval.
type UpdatePayload struct {
	Title     string `json:"title"`
	Statement string `json:"statement"`
}

// UpdateApproval sends a message to update the external approval.
func (c *Client) UpdateApproval(relayEndpoint string, id string, payload *UpdatePayload) error {
	out, err := json.Marshal(&payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/approval/%s", relayEndpoint, id), bytes.NewBuffer(out))
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("failed to create external approval with status %v", resp.Status)
	}
	return nil
}

// GetApproval gets the approval of an external approval.
func (c *Client) GetApproval(relayEndpoint string, id string) (*ResponsePayload, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/approval/%s", relayEndpoint, id), strings.NewReader(""))
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("failed to get external approval status with status %v", resp.Status)
	}
	responseBody := &ResponsePayload{}
	err = json.NewDecoder(resp.Body).Decode(responseBody)
	if err != nil {
		return nil, err
	}
	return responseBody, nil
}
