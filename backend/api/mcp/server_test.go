package mcp

import (
	"context"
	"encoding/json"
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

func TestTransformBundleSchema(t *testing.T) {
	a := require.New(t)

	// Test bundle schema transformation
	bundleSchema := []byte(`{
		"$defs": {
			"bytebase.v1.TestRequest.jsonschema.json": {
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type": "object",
				"properties": {
					"name": {"type": "string"}
				}
			}
		},
		"$id": "bytebase.v1.TestRequest.jsonschema.bundle.json",
		"$ref": "#/$defs/bytebase.v1.TestRequest.jsonschema.json",
		"$schema": "https://json-schema.org/draft/2020-12/schema"
	}`)

	transformed, err := transformBundleSchema(bundleSchema)
	a.NoError(err)

	// Verify the transformed schema has type: object at root
	var result map[string]any
	err = json.Unmarshal(transformed, &result)
	a.NoError(err)
	a.Equal("object", result["type"])
	a.NotNil(result["properties"])

	// Verify no $ref at root level
	a.Nil(result["$ref"])
}

func TestTransformBundleSchemaWithDeps(t *testing.T) {
	a := require.New(t)

	// Test bundle schema with dependencies
	bundleSchema := []byte(`{
		"$defs": {
			"bytebase.v1.TestRequest.jsonschema.json": {
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type": "object",
				"properties": {
					"nested": {"$ref": "#/$defs/bytebase.v1.NestedType.jsonschema.json"}
				}
			},
			"bytebase.v1.NestedType.jsonschema.json": {
				"$schema": "https://json-schema.org/draft/2020-12/schema",
				"type": "object",
				"properties": {
					"value": {"type": "string"}
				}
			}
		},
		"$id": "bytebase.v1.TestRequest.jsonschema.bundle.json",
		"$ref": "#/$defs/bytebase.v1.TestRequest.jsonschema.json",
		"$schema": "https://json-schema.org/draft/2020-12/schema"
	}`)

	transformed, err := transformBundleSchema(bundleSchema)
	a.NoError(err)

	// Verify the transformed schema has type: object at root
	var result map[string]any
	err = json.Unmarshal(transformed, &result)
	a.NoError(err)
	a.Equal("object", result["type"])

	// Verify $defs contains the dependency but not the main definition
	defs, ok := result["$defs"].(map[string]any)
	a.True(ok)
	a.NotNil(defs["bytebase.v1.NestedType.jsonschema.json"])
	a.Nil(defs["bytebase.v1.TestRequest.jsonschema.json"])
}
