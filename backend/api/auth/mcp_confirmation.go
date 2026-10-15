package auth

import (
	"context"
	"time"
)

// MCPConfirmedDecision is one issue decision the user confirmed in their MCP
// client. The MCP gate serves IssueService/ApproveIssue and RejectIssue only to
// a request whose context carries one naming that procedure and issue.
type MCPConfirmedDecision struct {
	// Procedure is the connect procedure confirmed, such as
	// /bytebase.v1.IssueService/ApproveIssue.
	Procedure string
	// Issue is the issue's resource name: projects/{project}/issues/{issue}.
	Issue string
	// IssueUpdateTime is the issue's update time when the user confirmed. The
	// review workflow refuses the decision when it no longer matches.
	IssueUpdateTime time.Time
}

// mcpConfirmedDecisionKey is unexported so that only code in this process can
// set the value; nothing a request carries can.
type mcpConfirmedDecisionKey struct{}

// WithMCPConfirmedDecision marks ctx as carrying a decision the user confirmed.
func WithMCPConfirmedDecision(ctx context.Context, decision MCPConfirmedDecision) context.Context {
	return context.WithValue(ctx, mcpConfirmedDecisionKey{}, decision)
}

// MCPConfirmedDecisionFromContext returns the confirmed decision ctx carries.
func MCPConfirmedDecisionFromContext(ctx context.Context) (MCPConfirmedDecision, bool) {
	decision, ok := ctx.Value(mcpConfirmedDecisionKey{}).(MCPConfirmedDecision)
	return decision, ok
}
