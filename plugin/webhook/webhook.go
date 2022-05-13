package webhook

import (
	"fmt"
	"sync"
	"time"
)

var (
	receiverMu sync.RWMutex
	receivers  = make(map[string]Receiver)
	// Based on the local test, Teams sometimes cannot finish the request in 1 second, so use 3s.
	timeout    = 3 * time.Second
	timeFormat = "2006-01-02 15:04:05"
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

// Issue object of issue
type Issue struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// Project object of project
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

func (c *Context) genMeta() []meta {
	m := []meta{}

	if c.Issue != nil {
		m = append(m, meta{
			Name:  "Isuue",
			Value: c.Issue.Name,
		})
	}

	if c.Project != nil {
		m = append(m, meta{
			Name:  "Project",
			Value: c.Project.Name,
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
		return fmt.Errorf("webhook: no applicable receiver for webhook type: %v", webhookType)
	}

	return r.post(context)
}
