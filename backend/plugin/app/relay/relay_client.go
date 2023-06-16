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
	// TODO(d): content TBD.
}

// CreateResponse is the response message to create external approval.
type CreateResponse struct {
	URI string `json:"uri"`
}

// Create sends a message to create external approval.
func (c *Client) Create(relayEndpoint string, payload CreatePayload) (string, error) {
	out, err := json.Marshal(&payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/approvals", relayEndpoint), bytes.NewBuffer(out))
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
	responseBody := &CreateResponse{}
	err = json.NewDecoder(resp.Body).Decode(responseBody)
	if err != nil {
		return "", err
	}
	return responseBody.URI, nil
}

// UpdatePayload is the message to update external approval.
type UpdatePayload struct {
	URI string `json:"uri"`
}

// UpdateStatus sends a message to update the status of an external approval.
func (c *Client) UpdateStatus(relayEndpoint string, uri string) error {
	payload := &UpdatePayload{URI: uri}
	out, err := json.Marshal(&payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/approvals", relayEndpoint), bytes.NewBuffer(out))
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

type Status string

// GetStatus is the response message to get the status of an external approval.
type GetStatus struct {
	Status Status `json:"status"`
}

const (
	StatusApproved Status = "APPROVED"
	StatusRejected Status = "REJECTED"
)

// GetStatus gets the status of an external approval.
func (c *Client) GetStatus(relayEndpoint string, uri string) (Status, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/approvals/status", relayEndpoint), strings.NewReader(""))
	if err != nil {
		return "", err
	}
	q := req.URL.Query()
	q.Add("uri", uri)
	req.URL.RawQuery = q.Encode()
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("failed to get external approval status with status %v", resp.Status)
	}
	responseBody := &GetStatus{}
	err = json.NewDecoder(resp.Body).Decode(responseBody)
	if err != nil {
		return "", err
	}
	return responseBody.Status, nil
}
