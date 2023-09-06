package api

// ExternalApprovalType is the type of the ExternalApproval.
type ExternalApprovalType string

const (
	// ExternalApprovalTypeFeishu is the ExternalApproval from feishu.
	ExternalApprovalTypeFeishu ExternalApprovalType = "bb.plugin.app.feishu"
	// ExternalApprovalTypeRelay is the ExternalApproval from relay.
	ExternalApprovalTypeRelay ExternalApprovalType = "bb.plugin.app.relay"
)

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
