package auth

import (
	"context"
	"time"
)

// MCPConfirmedDecision is the user's decision, confirmed in their MCP client, to
// make one write. The MCP gate serves the procedures that need one only to a
// request whose context carries one naming that procedure and the resource the
// request changes.
type MCPConfirmedDecision struct {
	// Procedure is the connect procedure confirmed, such as
	// /bytebase.v1.IssueService/ApproveIssue.
	Procedure string
	// Resource is the name of the resource the write changes, such as
	// projects/{project}/issues/{issue}.
	Resource string
	// IssueUpdateTime is, for an issue decision, the issue's update time when the
	// user confirmed. The review workflow refuses the decision when it no longer
	// matches.
	IssueUpdateTime time.Time
	// WithoutAsking marks a statement query_database runs without asking the
	// user. SQLService/Query runs a possible write for MCP only with this mark.
	WithoutAsking bool
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
