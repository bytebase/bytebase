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

// TestBoundaryRevalidatesEveryRequest pins the invariant the delegated
// credential rests on: authMiddleware guards the whole /mcp route, so EVERY
// JSON-RPC POST — not just initialize — re-validates the live bearer before any
// tool runs. Minting internal credentials from the session identity is
// therefore not a way around token expiry or revocation: a session whose
// credential has gone bad cannot reach a tool at all.
func TestBoundaryRevalidatesEveryRequest(t *testing.T) {
	const secret = "test-secret-key"
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}

	newSession := func(t *testing.T) (*mcp.ClientSession, *swappingTransport, *int32, *Server) {
		t.Helper()
		var toolHits int32
		stub := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			atomic.AddInt32(&toolHits, 1)
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `{}`)
		})
		s, err := NewServer(nil, profile, secret, stub)
		require.NoError(t, err)

		e := echo.New()
		s.RegisterRoutes(e)
		ts := httptest.NewServer(e)
		t.Cleanup(ts.Close)

		valid, err := auth.GenerateOAuth2AccessToken("test@example.com", "client-A", "ws-test", "https://bb.example.com/mcp", "mcp:read-only", secret, time.Hour)
		require.NoError(t, err)
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

		expired, err := auth.GenerateOAuth2AccessToken("test@example.com", "client-A", "ws-test", "https://bb.example.com/mcp", "mcp:read-only", secret, -time.Minute)
		require.NoError(t, err)
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
}
