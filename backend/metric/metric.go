// Package metric provides the API definition for metrics.
package metric

import (
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/metric"
)

const (
	// InstanceCountMetricName is the metric name for instance count.
	InstanceCountMetricName metric.Name = "bb.instance.count"
	// IssueCountMetricName is the metric name for issue count.
	IssueCountMetricName metric.Name = "bb.issue.count"
	// PolicyCountMetricName is the metric name for policy count.
	PolicyCountMetricName metric.Name = "bb.policy.count"
	// ProjectCountMetricName is the metric name for project count.
	ProjectCountMetricName metric.Name = "bb.project.count"
	// TaskCountMetricName is the metric name for task count.
	TaskCountMetricName metric.Name = "bb.task.count"
	// DatabaseCountMetricName is the metric name for database count.
	DatabaseCountMetricName metric.Name = "bb.database.count"
	// SheetCountMetricName is the metric name for sheet count.
	SheetCountMetricName metric.Name = "bb.sheet.count"
	// MemberCountMetricName is the metric name for member count.
	MemberCountMetricName metric.Name = "bb.member.count"
	// OpenAPIMetricName is the metric name for OpenAPI.
	OpenAPIMetricName metric.Name = "bb.api.call"
	// SQLAdviseAPIMetricName is the metric name for SQL check API.
	SQLAdviseAPIMetricName metric.Name = "bb.api.sql.advise"
	// SubscriptionTrialMetricName is the metric name for trial.
	SubscriptionTrialMetricName metric.Name = "bb.subscription.trial"
	// PrincipalRegistrationMetricName is the metric name for the principal registration event.
	PrincipalRegistrationMetricName metric.Name = "bb.principal.registration"
	// IssueCreateMetricName is the metric name for issue creation event.
	IssueCreateMetricName metric.Name = "bb.issue.create"
	// SQLEditorExecutionMetricName is the metric name for SQL Editor execution event.
	SQLEditorExecutionMetricName metric.Name = "bb.sql-editor.execute"
	// MemberCreateMetricName is the metric name for member creation event.
	MemberCreateMetricName metric.Name = "bb.member.create"
)

// InstanceCountMetric is the API message for bb.instance.count.
type InstanceCountMetric struct {
	Engine        db.Type
	EnvironmentID int
	RowStatus     api.RowStatus
	Count         int
}

// IssueCountMetric is the API message for bb.issue.count.
type IssueCountMetric struct {
	Type   api.IssueType
	Status api.IssueStatus
	Count  int
}

// ProjectCountMetric is the API message for project count metric.
type ProjectCountMetric struct {
	TenantMode   api.ProjectTenantMode
	WorkflowType api.ProjectWorkflowType
	RowStatus    api.RowStatus
	Count        int
}

// PolicyCountMetric is the API message for policy count metric.
type PolicyCountMetric struct {
	Type            api.PolicyType
	Value           string
	EnvironmentName string
	Count           int
}

// TaskCountMetric is the API message for task count metric.
type TaskCountMetric struct {
	Type   api.TaskType
	Status api.TaskStatus
	Count  int
}

// DatabaseCountMetric is the API message for database count metric.
type DatabaseCountMetric struct {
	BackupPlanPolicySchedule *api.BackupPlanPolicySchedule
	BackupSettingEnabled     *bool // nil if BackupPlanPolicyScheduleUnset
	Count                    int
}

// SheetCountMetric is the API message for sheet count metric.
type SheetCountMetric struct {
	RowStatus  api.RowStatus
	Visibility api.SheetVisibility
	Source     api.SheetSource
	Type       api.SheetType
	Count      int
}

// MemberCountMetric is the API message for member count metric.
type MemberCountMetric struct {
	Count     int
	Role      api.Role
	Status    api.MemberStatus
	RowStatus api.RowStatus
	Type      api.PrincipalType
}
