// Package metric provides the API definition for metrics.
package metric

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/metric"
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
	// ServiceAccountCountMetricName is the metric name for service account count.
	ServiceAccountCountMetricName metric.Name = "bb.service-account.count"
	// OpenAPIMetricName is the metric name for OpenAPI.
	OpenAPIMetricName metric.Name = "bb.api.call"
	// PrincipalRegistrationMetricName is the metric name for the principal registration event.
	PrincipalRegistrationMetricName metric.Name = "bb.principal.registration"
	// PrincipalLoginMetricName is the metric name for principal login event.
	PrincipalLoginMetricName metric.Name = "bb.principal.login"
	// IssueCreateMetricName is the metric name for issue creation event.
	IssueCreateMetricName metric.Name = "bb.issue.create"
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
