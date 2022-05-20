package api

import (
	"github.com/bytebase/bytebase/plugin/db"
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
