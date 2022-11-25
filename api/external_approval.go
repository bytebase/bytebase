package api

// ExternalApprovalType is the type of the ExternalApproval.
type ExternalApprovalType string

// ExternalApprovalTypeFeishu is the ExternalApproval from feishu.
const ExternalApprovalTypeFeishu = "bb.plugin.app.feishu"

// ExternalApprovalCancelReason describes why the external approval is canceled by Bytebase.
type ExternalApprovalCancelReason string

const (
	// ExternalApprovalCancelReasonGeneral is the general reason, used as a default.
	ExternalApprovalCancelReasonGeneral ExternalApprovalCancelReason = "Canceled because the assignee has been changed, or all tasks of the stage have been approved or the issue is no longer open."
	// ExternalApprovalCancelReasonIssueNotOpen is used if the issue is not open.
	ExternalApprovalCancelReasonIssueNotOpen ExternalApprovalCancelReason = "Canceled because the containing issue is no longer open."
	// ExternalApprovalCancelReasonReassigned is used if the assignee has been changed.
	ExternalApprovalCancelReasonReassigned ExternalApprovalCancelReason = "Canceled because the assignee has changed."
	// ExternalApprovalCancelReasonNoTaskPendingApproval is used if there is no pending approval tasks.
	ExternalApprovalCancelReasonNoTaskPendingApproval ExternalApprovalCancelReason = "Canceled because all tasks have benn approved."
)

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

	// feishu
	InstanceCode string
	RequesterID  string
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
}
