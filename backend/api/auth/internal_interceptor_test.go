package auth

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

// callInternalUnary runs a request through the internal interceptor with a
// recording next, returning the error and whether next was reached.
func callInternalUnary(t *testing.T, authHeader string) (nextCalled bool, err error) {
	t.Helper()
	in := NewInternalMCPAuthInterceptor(nil, testSecret, nil)
	req := connect.NewRequest(&emptypb.Empty{})
	if authHeader != "" {
		req.Header().Set("Authorization", authHeader)
	}
	next := func(_ context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		nextCalled = true
		return nil, nil
	}
	_, err = in.WrapUnary(next)(context.Background(), req)
	return nextCalled, err
}

// TestInternalInterceptorRejectsPublicTokens pins the accept-ONLY-the-internal-
// credential rule: web session tokens and external OAuth2 MCP tokens must never
// reach a handler through the private transport. Rejection happens before any
// store access, so a nil store proves the ordering too.
func TestInternalInterceptorRejectsPublicTokens(t *testing.T) {
	webToken, err := GenerateAccessToken("demo@example.com", "ws-test", testSecret, time.Hour)
	require.NoError(t, err)
	mcpToken, err := GenerateOAuth2AccessToken("demo@example.com", "client-A", "ws-test", testResource, "mcp:read-only", testSecret, time.Hour)
	require.NoError(t, err)

	for name, header := range map[string]string{
		"no credential":            "",
		"malformed header":         "NotBearer x",
		"web session token":        "Bearer " + webToken,
		"external MCP OAuth token": "Bearer " + mcpToken,
		"garbage":                  "Bearer not-a-jwt",
	} {
		t.Run(name, func(t *testing.T) {
			nextCalled, err := callInternalUnary(t, header)
			require.False(t, nextCalled, "handler must not be reached")
			require.Error(t, err)
			require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
		})
	}
}

// TestInternalInterceptorRejectsStreaming pins that the private transport is
// unary-only: no MCP tool streams, so a streaming call through it is a bug and
// fails closed.
func TestInternalInterceptorRejectsStreaming(t *testing.T) {
	in := NewInternalMCPAuthInterceptor(nil, testSecret, nil)
	err := in.WrapStreamingHandler(nil)(context.Background(), nil)
	require.Error(t, err)
}

// TestAuthenticateDelegatedCarriesGrantVerbatim pins the AuthContext contract
// P1b keys on: every internal-chain request carries the verified credential's
// grant state — scope, resource, client, correlation — verbatim, and the two
// empty-scope states stay distinguishable (see common.DelegatedGrant). The
// principal is deliberately not part of this step; it is re-resolved against
// the store on every request, which backend/tests pins on a live server
// (TestMCPMembershipRevocationBitesLiveSession).
func TestAuthenticateDelegatedCarriesGrantVerbatim(t *testing.T) {
	rows := []struct {
		name string
		cred DelegatedMCPCredential
	}{
		{
			name: "a consented grant travels verbatim",
			cred: DelegatedMCPCredential{
				Principal:     "live@example.com",
				WorkspaceID:   "ws-live",
				ClientID:      "client-A",
				CorrelationID: "corr-1",
				Scope:         "mcp:read-only",
				Resource:      testResource,
			},
		},
		{
			name: "legacy pre-grant session: scope and resource both empty",
			cred: DelegatedMCPCredential{
				Principal:     "live@example.com",
				WorkspaceID:   "ws-live",
				CorrelationID: "corr-2",
			},
		},
		{
			// A grant that recorded no scope: a scope-less consent (permanent
			// population) or a PR-3-era mint during a rolling upgrade
			// (transient, and its grant DID record a scope). The populated
			// resource is what keeps it distinguishable from the genuinely
			// pre-grant row above — collapsing the two could widen a consented
			// read-only session to full legacy semantics.
			name: "grant-backed token: resource present, scope empty",
			cred: DelegatedMCPCredential{
				Principal:     "live@example.com",
				WorkspaceID:   "ws-live",
				ClientID:      "client-A",
				CorrelationID: "corr-3",
				Resource:      testResource,
			},
		},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			token, err := GenerateInternalMCPToken(row.cred, testSecret)
			require.NoError(t, err)
			header := http.Header{}
			header.Set("Authorization", "Bearer "+token)

			cred, authContext, err := authenticateDelegated(header, "/bytebase.v1.SQLService/Query", testSecret)
			require.NoError(t, err)
			require.Equal(t, row.cred, *cred)
			require.NotNil(t, authContext.DelegatedGrant,
				"every internal-chain request must carry its delegated grant state")
			require.Equal(t, row.cred.Scope, authContext.DelegatedGrant.Scope)
			require.Equal(t, row.cred.Resource, authContext.DelegatedGrant.Resource)
			require.Equal(t, row.cred.ClientID, authContext.DelegatedGrant.ClientID)
			require.Equal(t, row.cred.CorrelationID, authContext.DelegatedGrant.CorrelationID)
		})
	}

	t.Run("the public chain carries no delegated grant", func(t *testing.T) {
		// The public interceptor builds its AuthContext from the method alone
		// and never touches the grant; only the internal chain sets it.
		authContext, err := getAuthContext("/bytebase.v1.SQLService/Query")
		require.NoError(t, err)
		require.Nil(t, authContext.DelegatedGrant,
			"a public-chain request must leave the delegated grant zero-valued")
	})
}
