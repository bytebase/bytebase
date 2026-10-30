package auth

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"time"
)

// MCPConfirmedDecision is the user's decision, confirmed in their MCP client, to
// make one write. The MCP gate serves the procedures that need one only to a
// request whose context carries one naming that procedure and the resource the
// request changes; SQLService/Query checks its own. query_database also
// attaches one, without asking, when the client cannot show the form.
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
	// Digest is, for a SQL request, MCPStatementDigest of the request the user
	// confirmed.
	Digest []byte
}

// MCPConfirmationReason and MCPConfirmationDomain name the ErrorInfo
// SQLService/Query returns for an MCP request that may write and that the user
// has not confirmed. query_database asks the user when it receives one.
const (
	MCPConfirmationReason = "CONFIRMATION_REQUIRED"
	MCPConfirmationDomain = "bytebase.com"
)

// MCPStatementDigest identifies the SQL request a user confirms: the statement
// exactly as sent, the database, the default schema and the CosmosDB container.
// Each field is length prefixed, so no two requests share a digest by moving
// text between fields.
func MCPStatementDigest(statement, database, schema, container string) []byte {
	h := sha256.New()
	for _, field := range []string{statement, database, schema, container} {
		_, _ = h.Write(binary.BigEndian.AppendUint64(nil, uint64(len(field))))
		_, _ = h.Write([]byte(field))
	}
	return h.Sum(nil)
}

// mcpConfirmedDecisionKey is unexported so that only code in this process can
// set the value; nothing a request carries can.
type mcpConfirmedDecisionKey struct{}

// WithMCPConfirmedDecision marks ctx as carrying decision.
func WithMCPConfirmedDecision(ctx context.Context, decision MCPConfirmedDecision) context.Context {
	return context.WithValue(ctx, mcpConfirmedDecisionKey{}, decision)
}

// MCPConfirmedDecisionFromContext returns the confirmed decision ctx carries.
func MCPConfirmedDecisionFromContext(ctx context.Context) (MCPConfirmedDecision, bool) {
	decision, ok := ctx.Value(mcpConfirmedDecisionKey{}).(MCPConfirmedDecision)
	return decision, ok
}
