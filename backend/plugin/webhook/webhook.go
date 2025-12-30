// Package webhook provides the webhook implementations for various messaging platforms.
package webhook

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
)

var (
	receiverMu sync.RWMutex
	receivers  = make(map[storepb.WebhookType]Receiver)
	// Based on the local test, Teams sometimes cannot finish the request in 1 second, so use 3s.
	Timeout = 3 * time.Second
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
	ID          int
	Name        string
	Status      string
	Type        string
	Description string
	Creator     Creator
}

type Creator struct {
	Name  string
	Email string
}

type Rollout struct {
	UID   int
	Title string
}

type Stage struct {
	Name string
}

// TaskResult is the latest result of a task.
// The `detail` field is only present if the status is TaskFailed.
// The `SkippedReason` field is only present if the task is skipped.
type TaskResult struct {
	Name          string
	Status        string
	Detail        string
	SkippedReason string
}

// Project object of project.
type Project struct {
	Name  string
	Title string
}

// Context is the context of webhook.
type Context struct {
	URL         string
	Level       Level
	EventType   string
	Title       string
	TitleZh     string
	Description string
	Link        string
	ActorID     int
	ActorName   string
	ActorEmail  string
	CreatedTS   int64
	Issue       *Issue
	Rollout     *Rollout
	Stage       *Stage
	Project     *Project
	TaskResult  *TaskResult
	// End users that should be mentioned.
	MentionEndUsers []*store.UserMessage

	DirectMessage bool
	IMSetting     *storepb.AppIMSetting

	// New event data
	ApprovalRole     string
	ApprovalRequired int
	FailedTasks      []FailedTaskInfo
	PipelineMetrics  *PipelineMetrics
}

// FailedTaskInfo contains information about a failed task.
type FailedTaskInfo struct {
	Name         string
	Instance     string
	Database     string
	ErrorMessage string
	FailedAt     string
}

// PipelineMetrics contains metrics about a pipeline execution.
type PipelineMetrics struct {
	TotalTasks   int
	StartedAt    string
	CompletedAt  string
	DurationSecs int64
}

// Receiver is the webhook receiver.
type Receiver interface {
	Post(context Context) error
}

func (c *Context) GetMetaList() []Meta {
	m := []Meta{}

	if c.Project != nil {
		m = append(m, Meta{
			Name:  "Project Title",
			Value: c.Project.Title,
		},
			Meta{
				Name:  "Project ID",
				Value: c.Project.Name,
			})
	}

	if c.Issue != nil {
		m = append(m, Meta{
			Name:  "Issue",
			Value: c.Issue.Name,
		}, Meta{
			Name:  "Issue Creator",
			Value: fmt.Sprintf("%s (%s)", c.Issue.Creator.Name, c.Issue.Creator.Email),
		})
		m = append(m, Meta{
			Name:  "Issue Description",
			Value: common.TruncateStringWithDescription(c.Issue.Description),
		})
	} else if c.Rollout != nil {
		if c.Rollout.Title != "" {
			m = append(m, Meta{
				Name:  "Rollout",
				Value: c.Rollout.Title,
			})
		}
	}

	if c.Stage != nil {
		m = append(m, Meta{
			Name:  "Stage",
			Value: c.Stage.Name,
		})
	}

	if c.TaskResult != nil {
		if c.TaskResult.Name != "" {
			m = append(m, Meta{
				Name:  "Task",
				Value: c.TaskResult.Name,
			})
		}
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
			Name:  "项目名称",
			Value: c.Project.Title,
		}, Meta{
			Name:  "项目 ID",
			Value: c.Project.Name,
		})
	}

	if c.Issue != nil {
		m = append(m, Meta{
			Name:  "工单",
			Value: c.Issue.Name,
		})
		m = append(m, Meta{
			Name:  "工单创建者",
			Value: fmt.Sprintf("%s (%s)", c.Issue.Creator.Name, c.Issue.Creator.Email),
		})
		m = append(m, Meta{
			Name:  "工单描述",
			Value: common.TruncateStringWithDescription(c.Issue.Description),
		})
	} else if c.Rollout != nil {
		if c.Rollout.Title != "" {
			m = append(m, Meta{
				Name:  "发布",
				Value: c.Rollout.Title,
			})
		}
	}

	if c.Stage != nil {
		m = append(m, Meta{
			Name:  "阶段",
			Value: c.Stage.Name,
		})
	}

	if c.TaskResult != nil {
		if c.TaskResult.Name != "" {
			m = append(m, Meta{
				Name:  "任务",
				Value: c.TaskResult.Name,
			})
		}
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

// Register makes a receiver available by the webhook type
// If Register is called twice with the same type or if receiver is nil,
// it panics.
func Register(webhookType storepb.WebhookType, r Receiver) {
	receiverMu.Lock()
	defer receiverMu.Unlock()
	if r == nil {
		panic("webhook: Register receiver is nil")
	}
	if _, dup := receivers[webhookType]; dup {
		panic("webhook: Register called twice for type " + webhookType.String())
	}
	receivers[webhookType] = r
}

// Post posts the message to webhook.
func Post(webhookType storepb.WebhookType, context Context) error {
	receiverMu.RLock()
	r, ok := receivers[webhookType]
	receiverMu.RUnlock()
	if !ok {
		return errors.Errorf("webhook: no applicable receiver for webhook type: %v", webhookType)
	}
	return r.Post(context)
}
