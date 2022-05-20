package api

import (
	"github.com/bytebase/bytebase/plugin/db"
	"github.com/bytebase/bytebase/plugin/metric"
)

var (
	// InstanceCountMetricName is the metric name for instance count
	InstanceCountMetricName metric.Name = "bb.instance.count"
	// IssueCountMetricName is the metric name for issue count
	IssueCountMetricName metric.Name = "bb.issue.count"
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
