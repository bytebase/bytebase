// Package webhook provides the webhook implementations for various messaging platforms.
package webhook

import (
	"sync"
	"time"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
)

var (
	receiverMu sync.RWMutex
	receivers  = make(map[string]Receiver)
	// Based on the local test, Teams sometimes cannot finish the request in 1 second, so use 3s.
	timeout = 3 * time.Second
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

// Issue object of issue.
type Issue struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// TaskResult is the latest result of a task.
// The `detail` field is only present if the status is TaskFailed.
// The `SkippedReason` field is only present if the task is skipped.
type TaskResult struct {
	Name          string `json:"name"`
	Status        string `json:"status"`
	Detail        string `json:"detail"`
	SkippedReason string `json:"skippedReason"`
}

// Project object of project.
type Project struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Context is the context of webhook.
type Context struct {
	URL                 string
	Level               Level
	ActivityType        string
	Title               string
	TitleZh             string
	Description         string
	Link                string
	CreatorID           int
	CreatorName         string
	CreatorEmail        string
	CreatedTs           int64
	Issue               *Issue
	Project             *Project
	TaskResult          *TaskResult
	MentionUsers        []*store.UserMessage
	MentionUsersByPhone []string

	DirectMessage bool
	IMSetting     *storepb.AppIMSetting
}

// Receiver is the webhook receiver.
type Receiver interface {
	Post(context Context) error
}

func (c *Context) GetMetaList() []Meta {
	m := []Meta{}

	if c.Project != nil {
		m = append(m, Meta{
			Name:  "Project",
			Value: c.Project.Name,
		})
	}

	if c.Issue != nil {
		m = append(m, Meta{
			Name:  "Issue",
			Value: c.Issue.Name,
		})
		// For VCS workflow, the generated issue description is composed of file names in the push event.
		// So the description could be long, which is hard to display if merged into the issue name.
		// We also trim it to 200 bytes to limit the message size in the webhook body, so that users can
		// view it easily in the corresponding webhook client.
		m = append(m, Meta{
			Name:  "Issue Description",
			Value: common.TruncateStringWithDescription(c.Issue.Description),
		})
	}

	if c.TaskResult != nil {
		m = append(m, Meta{
			Name:  "Task",
			Value: c.TaskResult.Name,
		})
		m = append(m, Meta{
			Name:  "Status",
			Value: c.TaskResult.Status,
		})
		if c.TaskResult.Detail != "" {
			m = append(m, Meta{
				Name:  "Result Detail",
				Value: common.TruncateStringWithDescription(c.TaskResult.Detail),
			})
		}
		if c.TaskResult.SkippedReason != "" {
			m = append(m, Meta{
				Name:  "Skipped Reason",
				Value: c.TaskResult.SkippedReason,
			})
		}
	}

	return m
}

func (c *Context) GetMetaListZh() []Meta {
	m := []Meta{}

	if c.Project != nil {
		m = append(m, Meta{
			Name:  "项目",
			Value: c.Project.Name,
		})
	}

	if c.Issue != nil {
		m = append(m, Meta{
			Name:  "工单",
			Value: c.Issue.Name,
		})
		// For VCS workflow, the generated issue description is composed of file names in the push event.
		// So the description could be long, which is hard to display if merged into the issue name.
		// We also trim it to 200 bytes to limit the message size in the webhook body, so that users can
		// view it easily in the corresponding webhook client.
		m = append(m, Meta{
			Name:  "工单描述",
			Value: common.TruncateStringWithDescription(c.Issue.Description),
		})
	}

	if c.TaskResult != nil {
		m = append(m, Meta{
			Name:  "任务",
			Value: c.TaskResult.Name,
		})
		m = append(m, Meta{
			Name:  "状态",
			Value: c.TaskResult.Status,
		})
		if c.TaskResult.Detail != "" {
			m = append(m, Meta{
				Name:  "结果详情",
				Value: common.TruncateStringWithDescription(c.TaskResult.Detail),
			})
		}
		if c.TaskResult.SkippedReason != "" {
			m = append(m, Meta{
				Name:  "跳过原因",
				Value: c.TaskResult.SkippedReason,
			})
		}
	}

	return m
}

// Register makes a receiver available by the url host
// If Register is called twice with the same url host or if receiver is nil,
// it panics.
func Register(host string, r Receiver) {
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
	return r.Post(context)
}
