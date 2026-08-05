package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/component/config"
)

// newTestServerWithMock creates a *Server whose internal transport dispatches
// to the given handler in memory — the same shape production uses, no socket.
func newTestServerWithMock(t *testing.T, handler http.Handler) *Server {
	t.Helper()
	return &Server{
		profile:        &config.Profile{},
		internalClient: newInternalAPIClient(handler),
	}
}

// TestApiRequest_DelegatedCredentialReplacesInboundBearer is the PR 4
// bearer-elimination pin, replacing the retired TestApiRequest_AuthForwarding
// (which asserted the loopback design: the inbound bearer forwarded verbatim).
// Internal requests carry the minted delegated credential, and the inbound
// bearer string must not appear anywhere in the request.
func TestApiRequest_DelegatedCredentialReplacesInboundBearer(t *testing.T) {
	const inboundBearer = "inbound-public-bearer-token"
	var capturedAuth string
	var capturedHeaders http.Header
	s := newTestServerWithMock(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))

	s.secret = "test-secret-key"
	ctx := withAccessToken(context.Background(), inboundBearer)
	ctx = withDelegatedIdentity(ctx, auth.DelegatedMCPCredential{
		Principal:     "test@example.com",
		WorkspaceID:   "ws-test",
		CorrelationID: "corr-1",
	})
	resp, err := s.apiRequest(ctx, "/api/test", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Status)

	// Assert on the test goroutine: the handler ran on the transport's.
	for name, values := range capturedHeaders {
		for _, v := range values {
			require.NotContains(t, v, inboundBearer,
				"inbound bearer leaked into internal request header %s", name)
		}
	}

	credential := strings.TrimPrefix(capturedAuth, "Bearer ")
	require.NotEqual(t, capturedAuth, credential, "internal requests must carry a bearer credential")
	cred, err := auth.VerifyInternalMCPToken(credential, s.secret)
	require.NoError(t, err)
	require.Equal(t, "test@example.com", cred.Principal)
}

// TestApiRequest_InMemoryDispatch pins the transport shape: the request
// reaches the internal handler with its connect headers and body intact, and
// the response round-trips — all without a listener.
func TestApiRequest_InMemoryDispatch(t *testing.T) {
	var capturedPath, capturedProto, capturedContentType string
	var capturedBody []byte
	s := newTestServerWithMock(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedProto = r.Header.Get("Connect-Protocol-Version")
		capturedContentType = r.Header.Get("Content-Type")
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok":true}`)
	}))

	resp, err := s.apiRequest(context.Background(), "/bytebase.v1.SQLService/Query", map[string]any{"statement": "SELECT 1"})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Status)
	require.Equal(t, "/bytebase.v1.SQLService/Query", capturedPath)
	require.Equal(t, "1", capturedProto)
	require.Equal(t, "application/json", capturedContentType)
	require.JSONEq(t, `{"statement":"SELECT 1"}`, string(capturedBody))
	require.JSONEq(t, `{"ok":true}`, string(resp.Body))
	require.Equal(t, "application/json", resp.Headers.Get("Content-Type"))
}

func TestApiRequest_ErrorParsing(t *testing.T) {
	s := newTestServerWithMock(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"message": "permission denied", "code": "PERMISSION_DENIED"}`)
	}))

	ctx := context.Background()
	resp, err := s.apiRequest(ctx, "/api/test", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, resp.Status)

	errMsg := parseError(resp.Body)
	require.Equal(t, "permission denied", errMsg)
}

func TestApiRequest_RawMessage(t *testing.T) {
	const payload = `{"id":12345,"count":9999999999}`
	s := newTestServerWithMock(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, payload)
	}))

	ctx := context.Background()
	resp, err := s.apiRequest(ctx, "/api/test", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Status)

	// Verify raw JSON is preserved.
	require.JSONEq(t, payload, string(resp.Body))

	// Decode into typed struct with int64 fields to verify integers are preserved.
	type result struct {
		ID    int64 `json:"id"`
		Count int64 `json:"count"`
	}
	var r result
	err = json.Unmarshal(resp.Body, &r)
	require.NoError(t, err)
	require.Equal(t, int64(12345), r.ID)
	require.Equal(t, int64(9999999999), r.Count)
}

// testContext returns a context with a test workspace ID.
func testContext() context.Context {
	return withWorkspaceID(context.Background(), "wk-test")
}

func TestWorkspaceIDContext(t *testing.T) {
	ctx := context.Background()

	// Empty context returns empty string.
	require.Equal(t, "", getWorkspaceID(ctx))

	// Round-trips through context.
	ctx = withWorkspaceID(ctx, "wk-test-123")
	require.Equal(t, "wk-test-123", getWorkspaceID(ctx))
}
