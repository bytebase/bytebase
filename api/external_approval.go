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
