package api

// ExternalApprovalType is the type of the ExternalApproval.
type ExternalApprovalType string

const (
	// ExternalApprovalTypeFeishu is the ExternalApproval from feishu.
	ExternalApprovalTypeFeishu ExternalApprovalType = "bb.plugin.app.feishu"
	// ExternalApprovalTypeRelay is the ExternalApproval from relay.
	ExternalApprovalTypeRelay ExternalApprovalType = "bb.plugin.app.relay"
)
