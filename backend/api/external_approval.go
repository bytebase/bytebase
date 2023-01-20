package api

// ExternalApprovalType is the type of the ExternalApproval.
type ExternalApprovalType string

// ExternalApprovalTypeFeishu is the ExternalApproval from feishu.
const ExternalApprovalTypeFeishu = "bb.plugin.app.feishu"

// ExternalApproval is the API message of ExternalApproval.
// It only lives in the backend.
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

// ExternalApprovalPayloadFeishu is the payload for feishu type ExternalApproval.
type ExternalApprovalPayloadFeishu struct {
	StageID    int
	AssigneeID int

	// Feishu
	InstanceCode string
	RequesterID  string
	// Rejected tells if the approval has been rejected on Feishu.
	Rejected bool
}

// ExternalApprovalCreate is the API message for creating an ExternalApproval.
type ExternalApprovalCreate struct {
	// Related fields
	IssueID     int
	RequesterID int
	ApproverID  int

	// Domain specific fields
	Type    ExternalApprovalType
	Payload string
}

// ExternalApprovalFind is the API message for finding ExternalApprovals.
type ExternalApprovalFind struct {
	IssueID *int
}

// ExternalApprovalPatch is the API message for patching an ExternalApproval.
type ExternalApprovalPatch struct {
	ID        int
	RowStatus RowStatus

	Payload *string
}

// ExternalApprovalEventActionType is the type of the action which the user took.
type ExternalApprovalEventActionType string

const (
	// ExternalApprovalEventActionApprove means that the user approves via the external approval.
	ExternalApprovalEventActionApprove ExternalApprovalEventActionType = "APPROVE"
	// ExternalApprovalEventActionReject means that the user rejects via the external approval.
	ExternalApprovalEventActionReject ExternalApprovalEventActionType = "REJECT"
)

// ExternalApprovalEvent is the API message for describing an ExternalApproval.
type ExternalApprovalEvent struct {
	Type      ExternalApprovalType            `json:"type"`
	Action    ExternalApprovalEventActionType `json:"action"`
	StageName string                          `json:"stageName"`
}

const (
	// ExternalApprovalCancelReasonGeneral is the general reason, used as a default.
	ExternalApprovalCancelReasonGeneral string = "Canceled because the assignee has been changed, or the SQL has been modified, or all tasks of the stage have been approved or the issue is no longer open."
	// ExternalApprovalCancelReasonIssueNotOpen is used if the issue is not open.
	ExternalApprovalCancelReasonIssueNotOpen string = "Canceled because the containing issue is no longer open."
	// ExternalApprovalCancelReasonReassigned is used if the assignee has been changed.
	ExternalApprovalCancelReasonReassigned string = "Canceled because the assignee has changed."
	// ExternalApprovalCancelReasonSQLModified is used if the task SQL statement has been modified.
	ExternalApprovalCancelReasonSQLModified string = "Canceled because the SQL has been modified."
	// ExternalApprovalCancelReasonNoTaskPendingApproval is used if there is no pending approval tasks.
	ExternalApprovalCancelReasonNoTaskPendingApproval string = "Canceled because all tasks have been approved."
)
