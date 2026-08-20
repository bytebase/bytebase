package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/common/stacktrace"
)

// apiResponse holds the HTTP response from an internal API call.
type apiResponse struct {
	Status  int
	Body    json.RawMessage
	Headers http.Header
}

// internalAPIBaseURL is the nominal URL of the internal API handler chain. The
// in-memory transport never dials it — it exists only to satisfy http.Request.
const internalAPIBaseURL = "https://mcp-internal.bytebase.invalid"

// internalAPIUserAgent identifies MCP-originated calls in audit records.
const internalAPIUserAgent = "bytebase-mcp/internal"

// apiRequest executes a Connect-RPC POST against the internal API handler
// chain. Requests ride the private in-memory transport and authenticate with
// the delegated credential minted at the /mcp boundary — never the inbound
// bearer, which stops at that boundary (P1a PR 4).
func (s *Server) apiRequest(ctx context.Context, path string, body any) (*apiResponse, error) {
	var bodyBytes []byte
	var err error
	if body != nil {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal request body")
		}
	} else {
		bodyBytes = []byte("{}")
	}

	// Per-request bound via the context (not http.Client.Timeout, whose cancel
	// machinery only reaches the standard transport; the in-memory transport
	// honors context cancellation).
	ctx, cancel := context.WithTimeout(ctx, internalAPITimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, internalAPIBaseURL+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTP request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Connect-Protocol-Version", "1")
	// The in-memory transport adds no User-Agent of its own (net/http writes one
	// only when marshaling to the wire), and the audit interceptor records the
	// header. Name the surface explicitly rather than let audit rows for MCP
	// actions go blank.
	httpReq.Header.Set("User-Agent", internalAPIUserAgent)
	// Audit derives the caller IP from X-Real-IP first. An in-memory request has
	// no peer address to fall back to, so without this the origin of every
	// MCP-originated action would be blank.
	if ip := getCallerIP(ctx); ip != "" {
		httpReq.Header.Set(headerRealIP, ip)
	}

	// Mint the credential for this call. Per call, not per session: see
	// delegatedIdentityKey.
	if identity, ok := getDelegatedIdentity(ctx); ok {
		credential, err := auth.GenerateInternalMCPToken(identity, s.secret)
		if err != nil {
			return nil, errors.Wrap(err, "failed to mint the delegated credential")
		}
		httpReq.Header.Set("Authorization", "Bearer "+credential)
	}

	resp, err := s.internalClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute API request")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	return &apiResponse{
		Status:  resp.StatusCode,
		Body:    json.RawMessage(respBody),
		Headers: resp.Header,
	}, nil
}

// internalAPITimeout bounds one internal API call, matching the 30s
// http.Client timeout of the retired loopback transport.
const internalAPITimeout = 30 * time.Second

// newInternalAPIClient builds the HTTP client for the private transport. Its
// RoundTripper dispatches directly to the internal handler chain in memory, so
// internal requests — and the delegated credential they carry — never touch a
// socket.
func newInternalAPIClient(handler http.Handler) *http.Client {
	return &http.Client{
		Transport: &internalRoundTripper{handler: handler},
	}
}

// internalRoundTripper serves each request with the internal handler chain
// in-process instead of dialing anything.
type internalRoundTripper struct {
	handler http.Handler
}

func (t *internalRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.handler == nil {
		return nil, errors.New("internal MCP API transport is not configured")
	}
	if req.Body != nil {
		defer req.Body.Close()
	}
	rec := &responseRecorder{header: make(http.Header), status: http.StatusOK}
	// Serve on a goroutine so cancellation aborts the call like a dropped
	// connection would on a real transport. An abandoned handler keeps running
	// until it observes req.Context() itself — the same semantics a server
	// gives a disconnected client — and nothing reads its recorder afterwards.
	//
	// Recover here as well: on a real listener net/http contains a panic that
	// escapes the handler, and the retired loopback transport had echo's
	// recover middleware on top of that. Without this, such a panic would take
	// the process down. connect's own WithRecover handles panics inside the
	// RPC; this covers everything outside it (routing, unknown paths).
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			if p := recover(); p != nil {
				stack := stacktrace.TakeStacktrace(20 /* n */, 3 /* skip */)
				slog.Error("internal MCP transport panic", "path", req.URL.Path,
					log.BBError(errors.Errorf("error: %v\n%s", p, stack)))
				rec.panicked = true
			}
		}()
		t.handler.ServeHTTP(rec, req)
	}()
	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case <-done:
	}
	if rec.panicked {
		return nil, errors.New("internal API request panicked")
	}
	return &http.Response{
		StatusCode:    rec.status,
		Status:        fmt.Sprintf("%d %s", rec.status, http.StatusText(rec.status)),
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        rec.header,
		Body:          io.NopCloser(bytes.NewReader(rec.body.Bytes())),
		ContentLength: int64(rec.body.Len()),
		Request:       req,
	}, nil
}

// responseRecorder is the minimal in-memory http.ResponseWriter behind the
// internal transport (net/http/httptest is test-only). Connect unary responses
// are fully buffered, which is all the tools need.
type responseRecorder struct {
	header      http.Header
	body        bytes.Buffer
	status      int
	wroteHeader bool
	panicked    bool
}

func (r *responseRecorder) Header() http.Header {
	return r.header
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.wroteHeader = true
	return r.body.Write(b)
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.wroteHeader = true
	r.status = status
}

// parseError extracts the error message from an API error response body.
func parseError(body json.RawMessage) string {
	var errMap map[string]any
	if json.Unmarshal(body, &errMap) != nil {
		return ""
	}
	if msg, ok := errMap["message"].(string); ok {
		return msg
	}
	if code, ok := errMap["code"].(string); ok {
		return code
	}
	return ""
}

// toolError is a structured error returned by MCP tools.
type toolError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
	// ReadOnlyEnforcement is what held the failed request to reads, when the
	// session is capped at read-only. Empty otherwise. A statement the
	// classifier admitted can still be refused by a read-only database
	// session, and that is when the depth is most worth knowing.
	ReadOnlyEnforcement string `json:"readOnlyEnforcement,omitempty"`
}

func (e *toolError) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Suggestion)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Context key for storing the access token.
type accessTokenKey struct{}

// withAccessToken adds the access token to the context.
func withAccessToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, accessTokenKey{}, token)
}

// getAccessToken retrieves the access token from the context.
func getAccessToken(ctx context.Context) string {
	if token, ok := ctx.Value(accessTokenKey{}).(string); ok {
		return token
	}
	return ""
}

// Context key for the delegated identity established at the /mcp boundary.
// Internal API requests mint their credential from this — never from the
// inbound access token, which stops at the boundary.
//
// The context carries the identity, not a minted credential, because the MCP
// SDK hands every tool handler the context of the request that CREATED the
// session. A credential minted at the boundary would therefore be frozen at
// session start, and its request-scale TTL would expire mid-session while the
// session stayed open.
type delegatedIdentityKey struct{}

// withDelegatedIdentity adds the delegated identity to the context.
func withDelegatedIdentity(ctx context.Context, identity auth.DelegatedMCPCredential) context.Context {
	return context.WithValue(ctx, delegatedIdentityKey{}, identity)
}

// getDelegatedIdentity retrieves the delegated identity from the context.
func getDelegatedIdentity(ctx context.Context) (auth.DelegatedMCPCredential, bool) {
	identity, ok := ctx.Value(delegatedIdentityKey{}).(auth.DelegatedMCPCredential)
	return identity, ok
}

// Context key for the inbound caller's IP, carried onto internal requests so
// audit records where an MCP-originated action actually came from.
type callerIPKey struct{}

// withCallerIP adds the inbound caller's IP to the context.
func withCallerIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, callerIPKey{}, ip)
}

// getCallerIP retrieves the inbound caller's IP from the context.
func getCallerIP(ctx context.Context) string {
	if ip, ok := ctx.Value(callerIPKey{}).(string); ok {
		return ip
	}
	return ""
}

// Context key for the session binding: the identity fingerprint the /mcp
// session is pinned to, plus the bearer's own expiry. Established per request
// by authMiddleware and read by the SDK session-ownership check.
type sessionBindingKey struct{}

type sessionBinding struct {
	fingerprint string
	expiry      time.Time
}

// withSessionBinding adds the session binding to the context.
func withSessionBinding(ctx context.Context, binding sessionBinding) context.Context {
	return context.WithValue(ctx, sessionBindingKey{}, binding)
}

// getSessionBinding retrieves the session binding from the context.
func getSessionBinding(ctx context.Context) (sessionBinding, bool) {
	binding, ok := ctx.Value(sessionBindingKey{}).(sessionBinding)
	return binding, ok
}

// Context key for storing the authenticated user email.
type userEmailKey struct{}

// withUserEmail adds the authenticated user email to the context.
func withUserEmail(ctx context.Context, email string) context.Context {
	return context.WithValue(ctx, userEmailKey{}, email)
}

// getUserEmail retrieves the authenticated user email from the context.
func getUserEmail(ctx context.Context) string {
	if email, ok := ctx.Value(userEmailKey{}).(string); ok {
		return email
	}
	return ""
}

// Context key for storing the OAuth2 client ID.
type oauth2ClientIDKey struct{}

// withOAuth2ClientID adds the OAuth2 client ID to the context.
func withOAuth2ClientID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, oauth2ClientIDKey{}, id)
}

// getOAuth2ClientID retrieves the OAuth2 client ID from the context.
func getOAuth2ClientID(ctx context.Context) string {
	if id, ok := ctx.Value(oauth2ClientIDKey{}).(string); ok {
		return id
	}
	return ""
}

// Context key for storing the workspace ID.
type workspaceIDKey struct{}

// withWorkspaceID adds the workspace ID to the context.
func withWorkspaceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, workspaceIDKey{}, id)
}

// getWorkspaceID retrieves the workspace ID from the context.
func getWorkspaceID(ctx context.Context) string {
	if id, ok := ctx.Value(workspaceIDKey{}).(string); ok {
		return id
	}
	return ""
}
