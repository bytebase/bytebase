package api

import "github.com/bytebase/bytebase/plugin/db"

// InstanceCountMetric is the API message for instance count metric
type InstanceCountMetric struct {
	Engine        db.Type
	EnvironmentID int
	Count         int
}

// IssueCountMetric is the API message for issue count metric
type IssueCountMetric struct {
	Type   IssueType
	Status IssueStatus
	Count  int
}

// ProjectCountMetric is the API message for project count metric
type ProjectCountMetric struct {
	TenantMode   ProjectTenantMode
	WorkflowType ProjectWorkflowType
	Count        int
}

// PolicyCountMetric is the API message for policy count metric
type PolicyCountMetric struct {
	Type          PolicyType
	EnvironmentID int
	Count         int
}
