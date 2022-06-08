package metric

import (
	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/metric"
)

const (
	// InstanceCountMetricName is the metric name for instance count
	InstanceCountMetricName metric.Name = "bb.instance.count"
	// IssueCountMetricName is the metric name for issue count
	IssueCountMetricName metric.Name = "bb.issue.count"
	// PolicyCountMetricName is the metric name for policy count
	PolicyCountMetricName metric.Name = "bb.policy.count"
	// ProjectCountMetricName is the metric name for project count
	ProjectCountMetricName metric.Name = "bb.project.count"
	// TaskCountMetricName is the metric name for task count
	TaskCountMetricName metric.Name = "bb.task.count"
	// DatabaseCountMetricName is the metric name for database count
	DatabaseCountMetricName metric.Name = "bb.database.count"
	// SheetCountMetricName is the metric name for sheet count
	SheetCountMetricName metric.Name = "bb.sheet.count"
)

// InstanceCountMetric is the API message for bb.instance.count
type InstanceCountMetric struct {
	Engine        db.Type
	EnvironmentID int
	RowStatus     api.RowStatus
	Count         int
}

// IssueCountMetric is the API message for bb.issue.count
type IssueCountMetric struct {
	Type   api.IssueType
	Status api.IssueStatus
	Count  int
}

// ProjectCountMetric is the API message for project count metric
type ProjectCountMetric struct {
	TenantMode   api.ProjectTenantMode
	WorkflowType api.ProjectWorkflowType
	RowStatus    api.RowStatus
	Count        int
}

// PolicyCountMetric is the API message for policy count metric
type PolicyCountMetric struct {
	Type            api.PolicyType
	Value           string
	EnvironmentName string
	Count           int
}

// TaskCountMetric is the API message for task count metric
type TaskCountMetric struct {
	Type   api.TaskType
	Status api.TaskStatus
	Count  int
}

// DatabaseCountMetric is the API message for database count metric
type DatabaseCountMetric struct {
	BackupPlanPolicySchedule *api.BackupPlanPolicySchedule
	BackupSettingEnabled     *bool // nil if BackupPlanPolicyScheduleUnset
	Count                    int
}

// SheetCountMetric is the API message for sheet count metric
type SheetCountMetric struct {
	RowStatus  api.RowStatus
	Visibility api.SheetVisibility
	Source     api.SheetSource
	Type       api.SheetType
	Count      int
}
