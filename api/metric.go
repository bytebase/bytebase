package api

import (
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/metric/reporter"
)

var (
	// InstanceCountMetricName is the MetricName for instance count
	InstanceCountMetricName reporter.MetricName = "bb.instance.count"
	// IssueCountMetricName is the MetricName for issue count
	IssueCountMetricName reporter.MetricName = "bb.issue.count"
)

// InstanceCountMetric is the API message for bb.instance.count
type InstanceCountMetric struct {
	Engine        db.Type
	EnvironmentID int
	Count         int
}

// IssueCountMetric is the API message for bb.issue.count
type IssueCountMetric struct {
	Type  IssueType
	Count int
}
