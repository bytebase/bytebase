package metric

import (
	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/metric"
)

var (
	// InstanceCountMetricName is the metric name for instance count
	InstanceCountMetricName metric.Name = "bb.instance.count"
	// IssueCountMetricName is the metric name for issue count
	IssueCountMetricName metric.Name = "bb.issue.count"
	// PolicyCountMetricName is the metric name for policy count
	PolicyCountMetricName metric.Name = "bb.policy.count"
	// ProjectCountMetricName is the metric name for project count
	ProjectCountMetricName metric.Name = "bb.project.count"
	// TaskCountMetricName is the metric name for database count
	TaskCountMetricName metric.Name = "bb.task.count"
)

// InstanceCountMetric is the API message for bb.instance.count
type InstanceCountMetric struct {
	Engine        db.Type
	EnvironmentID int
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

// TaskCountMetric is the API message for database count metric
type TaskCountMetric struct {
	Type   api.TaskType
	Status api.TaskStatus
	Count  int
}
