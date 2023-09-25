package api

// IssueStatus is the status of an issue.
type IssueStatus string

const (
	// IssueOpen is the issue status for OPEN.
	IssueOpen IssueStatus = "OPEN"
	// IssueDone is the issue status for DONE.
	IssueDone IssueStatus = "DONE"
	// IssueCanceled is the issue status for CANCELED.
	IssueCanceled IssueStatus = "CANCELED"
)

// IssueType is the type of an issue.
type IssueType string

const (
	// IssueGrantRequest is the issue type for requesting grant.
	IssueGrantRequest IssueType = "bb.issue.grant.request"

	// IssueDatabaseGeneral is the issue type for general database issues.
	IssueDatabaseGeneral IssueType = "bb.issue.database.general"
)

// IssueFieldID is the field ID for an issue.
// It has to be string type because the id for stage field contain multiple parts.
type IssueFieldID string

const (
	// IssueFieldName is the field ID for name.
	IssueFieldName IssueFieldID = "1"
	// IssueFieldStatus is the field ID for status.
	IssueFieldStatus IssueFieldID = "2"
	// IssueFieldAssignee is the field ID for assignee.
	IssueFieldAssignee IssueFieldID = "3"
	// IssueFieldDescription is the field ID for description.
	IssueFieldDescription IssueFieldID = "4"
	// IssueFieldProject is the field ID for project.
	IssueFieldProject IssueFieldID = "5"
	// IssueFieldSubscriberList is the field ID for subscriber list.
	IssueFieldSubscriberList IssueFieldID = "6"
	// IssueFieldSQL is the field ID for SQL.
	IssueFieldSQL IssueFieldID = "7"
)
