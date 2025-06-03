package base

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
