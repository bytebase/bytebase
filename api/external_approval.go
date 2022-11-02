package api

type ExternalApprovalType string

const ExternalApprovalTypeFeishu = "bb.plugin.app.feishu"

type ExternalApproval struct {
	ID int

	// Standard fields
	RowStatus RowStatus
	CreatedTs int64
	UpdatedTs int64

	// Related fields
	IssueID     int
	RequesterID int
	Requester   *Principal
	ApproverID  int
	Approver    *Principal

	// Domain specific fields
	Type    ExternalApprovalType
	Payload string
}

type ExternalApprovalPayloadFeishu struct {
	// feishu
	InstanceCode string
	RequesterID  string

	// bytebase
	StageID    int
	AssigneeID int
}

type ExternalApprovalCreate struct {
	IssueID     int
	RequesterID int
	ApproverID  int
	Type        ExternalApprovalType
	Payload     string
}

type ExternalApprovalFind struct{}

type ExternalApprovalPatch struct {
	ID        int
	RowStatus RowStatus
}
