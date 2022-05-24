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
)

var (
	_ metric.Metric = (*InstanceCountMetric)(nil)
	_ metric.Metric = (*IssueCountMetric)(nil)
	_ metric.Metric = (*ProjectCountMetric)(nil)
	_ metric.Metric = (*PolicyCountMetric)(nil)
)

// InstanceCountMetric is the API message for bb.instance.count
type InstanceCountMetric struct {
	Engine        db.Type
	EnvironmentID int
	Count         int
}

// Name returns the metric name.
func (InstanceCountMetric) Name() metric.Name {
	return InstanceCountMetricName
}

// Value returns the metric value.
func (m InstanceCountMetric) Value() int {
	return m.Count
}

// Labels returns the metric labels.
func (m InstanceCountMetric) Labels() map[string]string {
	return map[string]string{
		"engine": string(m.Engine),
		// TODO: let store collector collect env name.
		"environment": string(m.EnvironmentID),
	}
}

// IssueCountMetric is the API message for bb.issue.count
type IssueCountMetric struct {
	Type   api.IssueType
	Status api.IssueStatus
	Count  int
}

// Name returns the metric name.
func (IssueCountMetric) Name() metric.Name {
	return IssueCountMetricName
}

// Value returns the metric value.
func (m IssueCountMetric) Value() int {
	return m.Count
}

// Labels returns the metric labels.
func (m IssueCountMetric) Labels() map[string]string {
	return map[string]string{
		"type":   string(m.Type),
		"status": string(m.Status),
	}
}

// ProjectCountMetric is the API message for project count metric
type ProjectCountMetric struct {
	TenantMode   api.ProjectTenantMode
	WorkflowType api.ProjectWorkflowType
	RowStatus    api.RowStatus
	Count        int
}

// Name returns the metric name.
func (ProjectCountMetric) Name() metric.Name {
	return ProjectCountMetricName
}

// Value returns the metric value.
func (m ProjectCountMetric) Value() int {
	return m.Count
}

// Labels returns the metric labels.
func (m ProjectCountMetric) Labels() map[string]string {
	return map[string]string{
		"tenant_mode":   string(m.TenantMode),
		"workflow_type": m.WorkflowType.String(),
		"row_status":    m.RowStatus.String(),
	}
}

// PolicyCountMetric is the API message for policy count metric
type PolicyCountMetric struct {
	Type            api.PolicyType
	Payload         string
	EnvironmentName string
	Count           int
}

// Name returns the metric name.
func (PolicyCountMetric) Name() metric.Name {
	return PolicyCountMetricName
}

// Value returns the metric value.
func (m PolicyCountMetric) Value() int {
	return m.Count
}

// Labels returns the metric labels.
func (m PolicyCountMetric) Labels() map[string]string {
	return map[string]string{
		"type":        string(m.Type),
		"environment": m.EnvironmentName,
		// TODO: should use 'payload' as label name.
		"value": m.Payload,
	}
}
