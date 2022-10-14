// Package webhook provides the webhook implementations for various messaging platforms.
package webhook

import (
	"sync"
	"time"

	"github.com/pkg/errors"
)

var (
	receiverMu sync.RWMutex
	receivers  = make(map[string]Receiver)
	// Based on the local test, Teams sometimes cannot finish the request in 1 second, so use 3s.
	timeout = 3 * time.Second
)

// meta is the webhook metadata.
type meta struct {
	Name  string
	Value string
}

// Level is the level of webhook.
type Level string

const (
	// WebhookInfo is the webhook level for INFO.
	WebhookInfo Level = "INFO"
	// WebhookSuccess is the webhook level for SUCCESS.
	WebhookSuccess Level = "SUCCESS"
	// WebhookWarn is the webhook level for WARN.
	WebhookWarn Level = "WARN"
	// WebhookError is the webhook level for ERROR.
	WebhookError Level = "ERROR"
)

// Issue object of issue.
type Issue struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Project object of project.
type Project struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Context is the context of webhook.
type Context struct {
	URL          string
	Level        Level
	ActivityType string
	Title        string
	Description  string
	Link         string
	CreatorID    int
	CreatorName  string
	CreatorEmail string
	CreatedTs    int64
	Issue        *Issue
	Project      *Project
}

// Receiver is the webhook receiver.
type Receiver interface {
	post(context Context) error
}

func (c *Context) getMetaList() []meta {
	m := []meta{}

	if c.Project != nil {
		m = append(m, meta{
			Name:  "Project",
			Value: c.Project.Name,
		})
	}

	if c.Issue != nil {
		m = append(m, meta{
			Name:  "Issue",
			Value: c.Issue.Name,
		})
		// For VCS workflow, the generated issue description is composed of file names in the push event.
		// So the description could be long, which is hard to display if merged into the issue name.
		// We also trim it to 200 bytes to limit the message size in the webhook body, so that users can
		// view it easily in the corresponding webhook client.
		description := c.Issue.Description
		if len(description) > 200 {
			description = description[:200] + "... (view details in Bytebase)"
		}
		m = append(m, meta{
			Name:  "Issue Description",
			Value: description,
		})
	}

	return m
}

// Register makes a receiver available by the url host
// If Register is called twice with the same url host or if receiver is nil,
// it panics.
func register(host string, r Receiver) {
	receiverMu.Lock()
	defer receiverMu.Unlock()
	if r == nil {
		panic("webhook: Register receiver is nil")
	}
	if _, dup := receivers[host]; dup {
		panic("webhook: Register called twice for host " + host)
	}
	receivers[host] = r
}

// Post posts the message to webhook.
func Post(webhookType string, context Context) error {
	receiverMu.RLock()
	r, ok := receivers[webhookType]
	receiverMu.RUnlock()
	if !ok {
		return errors.Errorf("webhook: no applicable receiver for webhook type: %v", webhookType)
	}

	return r.post(context)
}
