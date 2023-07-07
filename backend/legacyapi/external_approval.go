package api

// ExternalApprovalType is the type of the ExternalApproval.
type ExternalApprovalType string

const (
	// ExternalApprovalTypeFeishu is the ExternalApproval from feishu.
	ExternalApprovalTypeFeishu ExternalApprovalType = "bb.plugin.app.feishu"
	// ExternalApprovalTypeRelay is the ExternalApproval from relay.
	ExternalApprovalTypeRelay ExternalApprovalType = "bb.plugin.app.relay"
)

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

// ExternalApprovalPayloadRelay is the payload for relay type ExternalApproval.
type ExternalApprovalPayloadRelay struct {
	ExternalApprovalNodeID string `json:"externalApprovalNodeId"`
	ID                     string `json:"id"`
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
