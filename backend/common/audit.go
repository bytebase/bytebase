package common

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// Audit method names for the two MCP doors that are not v1 RPCs. The
// interceptor names a row by the connect procedure it wrapped, so every
// consumer of AuditLog.method is shaped for "/<package>.<Service>/<Method>";
// these keep that shape without claiming a name in bytebase.v1, which is the
// surface the protos define and where the next reader would go looking for a
// descriptor. component/recovery already names its rows the same way, under
// bytebase.cli. The method part names what was ATTEMPTED — the row's status
// carries the verdict, exactly as for an RPC that ACL refused.
const (
	// AuditMethodMCPSessionAuthorize is a bearer token presented at /mcp and
	// held against the workspace ceiling. Per REQUEST, not per session: /mcp is
	// Streamable HTTP, so a client polling a workspace that refuses it writes a
	// row per poll. Nothing collapses repeats. Retention bounds the volume; if
	// that stops being enough, collapse on the session fingerprint the
	// middleware already computes for an admitted request.
	AuditMethodMCPSessionAuthorize = "/bytebase.mcp.Session/Authorize"
	// AuditMethodMCPConsentApprove is a user approving an MCP OAuth2 consent.
	AuditMethodMCPConsentApprove = "/bytebase.mcp.Consent/Approve"
)

// AuditLogWriter is the one store method a door outside the connect chains
// needs to record its own denial.
type AuditLogWriter interface {
	CreateAuditLog(ctx context.Context, workspace string, payload *storepb.AuditLog) error
}

// RecordOutOfBandAudit writes a row for a refusal no interceptor sees, and
// mirrors it to stdout when that is enabled.
//
// Best effort by design: an audit row that cannot be written must never turn a
// refusal into an admission, so the caller gets no error to act on. The stdout
// mirror does NOT depend on the insert — a metadata-database failure is when
// losing the row from both surfaces would matter most.
func RecordOutOfBandAudit(ctx context.Context, writer AuditLogWriter, mirrorToStdout bool, workspace string, row *storepb.AuditLog) {
	// WithoutCancel: the row must survive the client hanging up on its own
	// refusal, which is what a client does when it sees one.
	ctx = context.WithoutCancel(ctx)
	if err := writer.CreateAuditLog(ctx, workspace, row); err != nil {
		slog.Warn("failed to record a policy denial",
			slog.String("workspace", workspace), slog.String("method", row.GetMethod()), log.BBError(err))
	}
	if mirrorToStdout {
		LogAuditToStdout(ctx, row)
	}
}

// maxAuditPayloadChars is the maximum characters for request/response payloads in stdout logs.
// Set to 100KB (102400 chars) to match AWS CloudTrail industry standard for audit logs.
const maxAuditPayloadChars = 102400

// LogAuditToStdout writes audit log events to stdout using Go's standard slog library.
// Output format is controlled by the global slog handler (JSON in production, text in dev).
// Logs include a "log_type": "audit" field to distinguish from application logs.
// This is a best-effort operation - errors are not returned to avoid failing the audit flow.
//
// Every writer in the server process calls this when stdout audit is enabled:
// the stream is a mirror of the table, and a row only one of them carries is a
// row an operator reading the other cannot see. The recovery CLI
// (component/recovery) is the exception — it writes without a profile to read
// the flag from.
func LogAuditToStdout(ctx context.Context, p *storepb.AuditLog) {
	attrs := []slog.Attr{
		slog.String("log_type", "audit"),
		slog.String("parent", p.Parent),
		slog.String("method", p.Method),
	}

	if p.Resource != "" {
		attrs = append(attrs, slog.String("resource", p.Resource))
	}
	if p.User != "" {
		attrs = append(attrs, slog.String("user", p.User))
	}

	if p.Status != nil {
		attrs = append(attrs, slog.Int("status_code", int(p.Status.Code)))
		if p.Status.Message != "" {
			attrs = append(attrs, slog.String("status_message", p.Status.Message))
		}
	}

	if p.Latency != nil {
		attrs = append(attrs,
			slog.Int64("latency_ms", p.Latency.AsDuration().Milliseconds()),
		)
	}

	if p.RequestMetadata != nil {
		if p.RequestMetadata.CallerIp != "" {
			attrs = append(attrs, slog.String("client_ip", p.RequestMetadata.CallerIp))
		}
		if p.RequestMetadata.CallerSuppliedUserAgent != "" {
			attrs = append(attrs, slog.String("user_agent", p.RequestMetadata.CallerSuppliedUserAgent))
		}
	}

	// Include audit severity as an attribute (not as slog level)
	// Audit logs are always logged at INFO level - they represent business events, not system health
	// The severity field helps categorize the audit event itself
	if p.Severity != storepb.AuditLog_SEVERITY_UNSPECIFIED {
		attrs = append(attrs, slog.String("severity", p.Severity.String()))
	}

	attrs = append(attrs, mcpDelegationAttrs(p.McpDelegation)...)

	// Include request payload (truncated to 100KB for log manageability)
	// Request is already redacted for sensitive data by getRequestString()
	if p.Request != "" {
		request := p.Request
		if truncated, wasTruncated := TruncateString(p.Request, maxAuditPayloadChars); wasTruncated {
			request = truncated + "...[truncated]"
		}
		attrs = append(attrs, slog.String("request", request))
	}

	// Include response payload (truncated to 100KB for log manageability)
	// Response is already redacted for sensitive data by getResponseString()
	if p.Response != "" {
		response := p.Response
		if truncated, wasTruncated := TruncateString(p.Response, maxAuditPayloadChars); wasTruncated {
			response = truncated + "...[truncated]"
		}
		attrs = append(attrs, slog.String("response", response))
	}

	slog.LogAttrs(ctx, slog.LevelInfo, p.Method, attrs...)
}

// mcpDelegationAttrs renders the MCP provenance for stdout audit lines:
// "mcp": true marks the row as MCP-originated even when the grant fields are
// empty (legacy sessions); the correlation ID is what operators pivot on to
// reassemble an agent session.
func mcpDelegationAttrs(d *storepb.MCPDelegation) []slog.Attr {
	if d == nil {
		return nil
	}
	attrs := []slog.Attr{
		slog.Bool("mcp", true),
		// Minted for every session, legacy included. Empty only on a consent
		// entry, which never reached the /mcp boundary that mints it.
		slog.String("mcp_correlation_id", d.CorrelationId),
	}
	if d.Scope != "" {
		attrs = append(attrs, slog.String("mcp_scope", d.Scope))
	}
	if d.Resource != "" {
		attrs = append(attrs, slog.String("mcp_resource", d.Resource))
	}
	if d.ClientId != "" {
		attrs = append(attrs, slog.String("mcp_client_id", d.ClientId))
	}
	return attrs
}

// Caller-IP headers, in the precedence every audit path reads them: X-Real-IP
// is what a reverse proxy sets and is a single value, X-Forwarded-For is the
// standard one and can carry a client-supplied list. Both are
// client-controllable on a direct connection.
const (
	HeaderRealIP       = "X-Real-IP"
	headerForwardedFor = "X-Forwarded-For"
)

// CallerIPFromHeaders reads the caller IP out of the forwarding headers, or ""
// when neither is set.
func CallerIPFromHeaders(header http.Header) string {
	if ip := header.Get(HeaderRealIP); ip != "" {
		return ip
	}
	return header.Get(headerForwardedFor)
}

// CallerIP resolves who made an HTTP request: the forwarding headers if
// present, otherwise the peer address.
func CallerIP(r *http.Request) string {
	if ip := CallerIPFromHeaders(r.Header); ip != "" {
		return ip
	}
	return StripPort(r.RemoteAddr)
}

// StripPort drops the port from a peer address so one audit column holds one
// shape. A forwarding header carries a bare IP; a peer address carries
// host:port, and storing both spellings of "who called" makes the column
// unusable for grouping.
func StripPort(addr string) string {
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

// RequestMetadataFromHTTP renders the audit row's request metadata for a caller
// that arrived over plain HTTP rather than over a connect chain.
func RequestMetadataFromHTTP(r *http.Request) *storepb.RequestMetadata {
	return &storepb.RequestMetadata{
		CallerIp:                CallerIP(r),
		CallerSuppliedUserAgent: r.Header.Get("User-Agent"),
	}
}
