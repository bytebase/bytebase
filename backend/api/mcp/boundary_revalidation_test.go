package mcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

// swappingTransport presents whatever bearer is currently stored, so a test can
// change the client's credential mid-session.
type swappingTransport struct {
	token atomic.Pointer[string]
}

func (t *swappingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	if token := t.token.Load(); token != nil {
		req.Header.Set("Authorization", "Bearer "+*token)
	}
	return http.DefaultTransport.RoundTrip(req)
}

const revalidationSecret = "test-secret-key"

// TestBoundaryRevalidatesEveryRequest pins the invariant the delegated
// credential rests on: authMiddleware guards the whole /mcp route, so EVERY
// JSON-RPC POST — not just initialize — re-validates the live bearer before any
// tool runs. Minting internal credentials from the session identity is
// therefore not a way around token expiry or revocation: a session whose
// credential has gone bad cannot reach a tool at all.
func TestBoundaryRevalidatesEveryRequest(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}

	newSession := func(t *testing.T) (*mcp.ClientSession, *swappingTransport, *int32, *Server) {
		t.Helper()
		var toolHits int32
		stub := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			atomic.AddInt32(&toolHits, 1)
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{}`)
		})
		s, err := NewServer(nil, profile, revalidationSecret, stub)
		require.NoError(t, err)

		e := echo.New()
		s.RegisterRoutes(e)
		ts := httptest.NewServer(e)
		t.Cleanup(ts.Close)

		valid := mintMCPToken(t, "test@example.com", "client-A", "ws-test", "mcp:read-only", time.Hour)
		transport := &swappingTransport{}
		transport.token.Store(&valid)

		client := mcp.NewClient(&mcp.Implementation{Name: "probe", Version: "0"}, nil)
		session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
			Endpoint:   ts.URL + "/mcp",
			HTTPClient: &http.Client{Transport: transport},
			MaxRetries: -1,
		}, nil)
		require.NoError(t, err)
		t.Cleanup(func() { _ = session.Close() })
		return session, transport, &toolHits, s
	}

	callTool := func(ctx context.Context, session *mcp.ClientSession) error {
		_, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "call_api",
			Arguments: map[string]any{"operationId": "SQLService/Query"},
		})
		return err
	}

	t.Run("expired bearer on an established session cannot reach a tool", func(t *testing.T) {
		session, transport, toolHits, _ := newSession(t)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// The session is live: a tool call works while the bearer is valid.
		require.NoError(t, callTool(ctx, session))
		require.Equal(t, int32(1), atomic.LoadInt32(toolHits))

		expired := mintMCPToken(t, "test@example.com", "client-A", "ws-test", "mcp:read-only", -time.Minute)
		transport.token.Store(&expired)

		require.Error(t, callTool(ctx, session),
			"an expired bearer must be refused at the boundary, session or no session")
		require.Equal(t, int32(1), atomic.LoadInt32(toolHits),
			"no internal API call may be made for a request the boundary refused")
	})

	t.Run("revoked bearer on an established session cannot reach a tool", func(t *testing.T) {
		session, transport, toolHits, s := newSession(t)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		require.NoError(t, callTool(ctx, session))
		require.Equal(t, int32(1), atomic.LoadInt32(toolHits))

		// What the reauthorize tool does to the caller's own token.
		s.revokedAccessTokens.Store(*transport.token.Load(), struct{}{})

		require.Error(t, callTool(ctx, session),
			"a revoked bearer must be refused at the boundary")
		require.Equal(t, int32(1), atomic.LoadInt32(toolHits))
	})

	// The session identity is captured at initialize and reused for the whole
	// session (the SDK gives tool handlers the initialize request's context), so
	// the boundary admitting a DIFFERENT valid bearer mid-session would run the
	// internal call under the original principal. Substituting another user's
	// valid token would then execute as the session's user; substituting a token
	// for a workspace where MCP is still enabled would sidestep the kill switch
	// on the session's own workspace. The session must be bound to the identity
	// it was opened with.
	t.Run("a different principal's valid bearer cannot take over the session", func(t *testing.T) {
		session, transport, toolHits, _ := newSession(t)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		require.NoError(t, callTool(ctx, session))
		require.Equal(t, int32(1), atomic.LoadInt32(toolHits))

		other := mintMCPToken(t, "attacker@example.com", "client-A", "ws-test", "mcp:read-only", time.Hour)
		transport.token.Store(&other)

		require.Error(t, callTool(ctx, session),
			"a bearer for a different principal must not drive an established session")
		require.Equal(t, int32(1), atomic.LoadInt32(toolHits),
			"no internal call may run under the session's original identity for a substituted bearer")
	})

	t.Run("a bearer for a different workspace cannot take over the session", func(t *testing.T) {
		session, transport, toolHits, _ := newSession(t)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		require.NoError(t, callTool(ctx, session))

		otherWorkspace := mintMCPToken(t, "test@example.com", "client-A", "ws-other", "mcp:read-only", time.Hour)
		transport.token.Store(&otherWorkspace)

		require.Error(t, callTool(ctx, session),
			"a bearer bound to another workspace must not drive this session")
		require.Equal(t, int32(1), atomic.LoadInt32(toolHits))
	})

	t.Run("a different OAuth client cannot take over the session", func(t *testing.T) {
		session, transport, toolHits, _ := newSession(t)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		require.NoError(t, callTool(ctx, session))

		// Same user, different registered client: a separate grant, so a
		// separate session.
		otherClient := mintMCPToken(t, "test@example.com", "client-B", "ws-test", "mcp:read-only", time.Hour)
		transport.token.Store(&otherClient)

		require.Error(t, callTool(ctx, session),
			"a bearer issued to another OAuth client must not drive this session")
		require.Equal(t, int32(1), atomic.LoadInt32(toolHits))
	})

	t.Run("a narrowed re-consent cannot ride the old session", func(t *testing.T) {
		session, transport, toolHits, _ := newSession(t)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		require.NoError(t, callTool(ctx, session))

		// Same user, same client: only the consented scope changed. The session
		// still holds the wider grant state, so it must not keep serving.
		reconsented := mintMCPToken(t, "test@example.com", "client-A", "ws-test", "mcp:read-write", time.Hour)
		transport.token.Store(&reconsented)

		require.Error(t, callTool(ctx, session),
			"a re-consented grant must re-initialize rather than inherit the old session's state")
		require.Equal(t, int32(1), atomic.LoadInt32(toolHits))
	})

	// The binding must key on identity, not on the token string: a routine
	// refresh mints a new token with the same principal, workspace, client,
	// resource and scope, and must keep working.
	t.Run("a refreshed bearer for the same identity keeps working", func(t *testing.T) {
		session, transport, toolHits, _ := newSession(t)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		require.NoError(t, callTool(ctx, session))

		refreshed := mintMCPToken(t, "test@example.com", "client-A", "ws-test", "mcp:read-only", 2*time.Hour)
		require.NotEqual(t, *transport.token.Load(), refreshed)
		transport.token.Store(&refreshed)

		require.NoError(t, callTool(ctx, session),
			"refreshing a token must not break an open session")
		require.Equal(t, int32(2), atomic.LoadInt32(toolHits))
	})
}

// forwardedTransport presents a fixed bearer and a settable X-Forwarded-For, so
// a test can move an established session to a different client IP.
type forwardedTransport struct {
	token string
	ip    atomic.Pointer[string]
}

func (t *forwardedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.token)
	if ip := t.ip.Load(); ip != nil {
		req.Header.Set("X-Forwarded-For", *ip)
	}
	return http.DefaultTransport.RoundTrip(req)
}

// TestInternalRequestUsesLiveCallerIP pins that audit sees where a tool call
// came from NOW, not where the session was opened from. Tool handlers run on
// the initialize request's context, so a caller IP read from that context is
// frozen for the session's life: a client that changes network mid-session
// (wifi to cellular, VPN toggle, proxy rotation) would have every later action
// attributed to the address it first connected from.
//
// The session identity is still pinned — a moving IP must not end the session,
// which is what makes taking the live address safe: it cannot be a different
// principal's.
func TestInternalRequestUsesLiveCallerIP(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}

	var capturedIP atomic.Pointer[string]
	stub := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.Header.Get("X-Real-IP")
		capturedIP.Store(&ip)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	})
	s, err := NewServer(nil, profile, revalidationSecret, stub)
	require.NoError(t, err)

	e := echo.New()
	s.RegisterRoutes(e)
	ts := httptest.NewServer(e)
	defer ts.Close()

	transport := &forwardedTransport{
		token: mintMCPToken(t, "test@example.com", "client-A", "ws-test", "mcp:read-only", time.Hour),
	}
	first := "203.0.113.7"
	transport.ip.Store(&first)

	client := mcp.NewClient(&mcp.Implementation{Name: "probe", Version: "0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:   ts.URL + "/mcp",
		HTTPClient: &http.Client{Transport: transport},
		MaxRetries: -1,
	}, nil)
	require.NoError(t, err)
	defer session.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	callTool := func() error {
		_, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "call_api",
			Arguments: map[string]any{"operationId": "SQLService/Query"},
		})
		return err
	}

	require.NoError(t, callTool())
	require.Equal(t, first, *capturedIP.Load())

	// The same client, same credential, now reaching us from a different address.
	second := "198.51.100.22"
	transport.ip.Store(&second)

	require.NoError(t, callTool(), "an IP change must not end the session")
	require.Equal(t, second, *capturedIP.Load(),
		"audit must attribute the call to where it came from now, not to the session's first address")
}

// mintMCPToken mints an MCP OAuth2 access token bound to this deployment's
// canonical resource URI.
func mintMCPToken(t *testing.T, email, clientID, workspaceID, scope string, ttl time.Duration) string {
	t.Helper()
	token, err := auth.GenerateOAuth2AccessToken(email, clientID, workspaceID, "https://bb.example.com/mcp", scope, revalidationSecret, ttl)
	require.NoError(t, err)
	return token
}
