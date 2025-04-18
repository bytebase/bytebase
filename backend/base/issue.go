package base

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

func (t IssueStatus) String() string {
	return string(t)
}

// IssueType is the type of an issue.
type IssueType string

const (
	// IssueGrantRequest is the issue type for requesting grant.
	IssueGrantRequest IssueType = "bb.issue.grant.request"

	// IssueDatabaseGeneral is the issue type for general database issues.
	IssueDatabaseGeneral IssueType = "bb.issue.database.general"

	// IssueDatabaseDataExport is the issue type for requesting data export.
	IssueDatabaseDataExport IssueType = "bb.issue.database.data-export"
)

func (t IssueType) String() string {
	return string(t)
}
