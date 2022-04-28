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

// Meta is the webhook metadata.
type Meta struct {
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

// EventType is eventType of webhook.
type EventType string

const (
	IssueCreated                   EventType = "issue_created"
	IssueUpdated                   EventType = "issue_updated"
	IssueStatusUpdated             EventType = "issue_status_updated"
	IssueCommentCreated            EventType = "issue_comment_created"
	IssuePipelineTaskStatusUpdated EventType = "issue_pipeline_task_status_updated"
)

// ObjectKind kind of object. for now, only Issue.
type ObjectKind string

// Issue object kind
const Issue ObjectKind = "issue"

// IssueObject object of issue
type IssueObject struct {
	ID          int
	Name        string
	Status      string
	Type        string
	Description string
	AssigneeID  int
	ProjectID   int
}

// Attributes return Attributes map
func (i IssueObject) Attributes() map[string]interface{} {
	return map[string]interface{}{
		"id":          i.ID,
		"name":        i.Name,
		"status":      i.Status,
		"type":        i.Type,
		"description": i.Description,
		"assignee_id": i.AssigneeID,
		"project_id":  i.ProjectID,
	}
}

// Attributer inferface of Attributes
type Attributer interface {
	Attributes() map[string]interface{}
}

// Context is the context of webhook.
type Context struct {
	URL              string
	ObjectKind       ObjectKind
	ObjectAttributes Attributer
	EventType        EventType
	Level            Level
	Title            string
	Description      string
	Link             string
	CreatorName      string
	CreatorEmail     string
	CreatedTs        int64
	MetaList         []Meta
}

// Receiver is the webhook receiver.
type Receiver interface {
	post(context Context) error
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
