package auth

import (
	"context"
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
