// Package metric provides the API definition for metrics.
package metric

import (
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/plugin/metric"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	// InstanceCountMetricName is the metric name for instance count.
	InstanceCountMetricName metric.Name = "bb.instance.count"
	// IssueCountMetricName is the metric name for issue count.
	IssueCountMetricName metric.Name = "bb.issue.count"
	// ProjectCountMetricName is the metric name for project count.
	ProjectCountMetricName metric.Name = "bb.project.count"
	// UserCountMetricName is the metric name for user count.
	UserCountMetricName metric.Name = "bb.user.count"
	// OpenAPIMetricName is the metric name for OpenAPI.
	OpenAPIMetricName metric.Name = "bb.api.call"
	// SQLAdviseAPIMetricName is the metric name for SQL check API.
	SQLAdviseAPIMetricName metric.Name = "bb.api.sql.advise"
	// SubscriptionTrialMetricName is the metric name for trial.
	SubscriptionTrialMetricName metric.Name = "bb.subscription.trial"
	// PrincipalRegistrationMetricName is the metric name for the principal registration event.
	PrincipalRegistrationMetricName metric.Name = "bb.principal.registration"
	// PrincipalLoginMetricName is the metric name for principal login event.
	PrincipalLoginMetricName metric.Name = "bb.principal.login"
	// IssueCreateMetricName is the metric name for issue creation event.
	IssueCreateMetricName metric.Name = "bb.issue.create"
	// SQLEditorExecutionMetricName is the metric name for SQL Editor execution event.
	SQLEditorExecutionMetricName metric.Name = "bb.sql-editor.execute"
	// PrincipalCreateMetricName is the metric name for principal creation event.
	PrincipalCreateMetricName metric.Name = "bb.principal.create"
	// APIRequestMetricName is the metric name for api request.
	APIRequestMetricName metric.Name = "bb.api.request"
	// InstanceCreateMetricName is the metric name for instance creation event.
	InstanceCreateMetricName metric.Name = "bb.instance.create"
)

// InstanceCountMetric is the API message for bb.instance.count.
type InstanceCountMetric struct {
	Engine        storepb.Engine
	EnvironmentID string
	Count         int
}

// IssueCountMetric is the API message for bb.issue.count.
type IssueCountMetric struct {
	Count int
}

// ProjectCountMetric is the API message for project count metric.
type ProjectCountMetric struct {
	Count int
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

// SheetCountMetric is the API message for sheet count metric.
type SheetCountMetric struct {
	Visibility string
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
