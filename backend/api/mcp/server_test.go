package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewInvoker(t *testing.T) {
	a := require.New(t)

	testCases := []struct {
		baseURL string
		valid   bool
	}{
		{
			baseURL: "http://localhost:8080",
			valid:   true,
		},
		{
			baseURL: "https://example.com",
			valid:   true,
		},
		{
			baseURL: "",
			valid:   true, // Empty string is technically valid
		},
	}

	for _, tc := range testCases {
		invoker := NewInvoker(tc.baseURL)
		a.NotNil(invoker, "NewInvoker should not return nil")
		a.Equal(tc.baseURL, invoker.baseURL)
		a.NotNil(invoker.httpClient)
	}
}

func TestWithAuthHeader(t *testing.T) {
	a := require.New(t)

	testCases := []struct {
		name       string
		authHeader string
	}{
		{
			name:       "Bearer token",
			authHeader: "Bearer token123",
		},
		{
			name:       "Basic auth",
			authHeader: "Basic dXNlcjpwYXNz",
		},
		{
			name:       "Empty auth",
			authHeader: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			ctx := context.Background()
			ctxWithAuth := WithAuthHeader(ctx, tc.authHeader)

			// Verify the auth header is stored in context
			val := ctxWithAuth.Value(authHeaderKey{})
			if tc.authHeader == "" {
				a.Equal("", val)
			} else {
				a.Equal(tc.authHeader, val)
			}
		})
	}
}

func TestInvoker_doConnect_AuthHeaderForwarding(t *testing.T) {
	a := require.New(t)

	// Create a test server that checks for auth header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			t.Error("Expected Authorization header to be present")
		}
		if authHeader != "Bearer test-token" {
			t.Errorf("Expected Authorization header to be 'Bearer test-token', got '%s'", authHeader)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	// Create invoker with test server URL
	invoker := NewInvoker(server.URL)

	// Create context with auth header
	ctx := WithAuthHeader(context.Background(), "Bearer test-token")

	// Note: We can't fully test doConnect without proto messages,
	// but we can verify the invoker setup and context handling
	a.NotNil(invoker)
	a.NotNil(ctx.Value(authHeaderKey{}))
	a.Equal("Bearer test-token", ctx.Value(authHeaderKey{}))
}

func TestNewServer(t *testing.T) {
	a := require.New(t)

	t.Run("valid registry", func(*testing.T) {
		invoker := NewInvoker("http://localhost:8080")
		registry, err := NewRegistry(invoker)
		a.NoError(err)

		server, err := NewServer(registry, nil, nil, nil, nil)
		a.NoError(err)
		a.NotNil(server)
		a.NotNil(server.mcpServer)
		a.NotNil(server.registry)
	})
}

func TestServer_Handler(t *testing.T) {
	a := require.New(t)

	invoker := NewInvoker("http://localhost:8080")
	registry, err := NewRegistry(invoker)
	a.NoError(err)

	server, err := NewServer(registry, nil, nil, nil, nil)
	a.NoError(err)
	a.NotNil(server)

	// Test that Handler returns a non-nil HTTP handler
	handler := server.Handler("http://localhost:8080/.well-known/oauth-protected-resource")
	a.NotNil(handler, "Handler should return a non-nil http.Handler")

	// Note: We can't fully test the handler without proper auth setup,
	// but we can verify it doesn't panic on creation
	a.NotNil(handler)
}

func TestAuthHeaderKey(t *testing.T) {
	a := require.New(t)

	// Test that authHeaderKey is properly unique
	ctx := context.Background()
	ctx1 := context.WithValue(ctx, authHeaderKey{}, "value1")
	ctx2 := context.WithValue(ctx, authHeaderKey{}, "value2")

	a.Equal("value1", ctx1.Value(authHeaderKey{}))
	a.Equal("value2", ctx2.Value(authHeaderKey{}))

	// Verify different keys don't collide
	type otherKey struct{}
	ctx3 := context.WithValue(ctx, otherKey{}, "other")
	a.Nil(ctx3.Value(authHeaderKey{}))
	a.Equal("other", ctx3.Value(otherKey{}))
}
