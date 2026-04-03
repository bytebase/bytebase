package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/config"
)

// newTestServerWithMock creates a *Server backed by a mock HTTP server.
// The mock server is automatically closed when the test completes.
func newTestServerWithMock(t *testing.T, handler http.Handler) *Server {
	t.Helper()
	mock := httptest.NewServer(handler)
	t.Cleanup(mock.Close)
	// Parse port from URL like "http://127.0.0.1:PORT".
	parts := strings.Split(mock.URL, ":")
	port, _ := strconv.Atoi(parts[len(parts)-1])
	return &Server{
		profile: &config.Profile{Port: port},
	}
}

func TestApiRequest_AuthForwarding(t *testing.T) {
	var capturedAuth string
	s := newTestServerWithMock(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	}))

	ctx := withAccessToken(context.Background(), "test-token-123")
	resp, err := s.apiRequest(ctx, "/api/test", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Status)
	require.Equal(t, "Bearer test-token-123", capturedAuth)
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

func TestWorkspaceIDContext(t *testing.T) {
	ctx := context.Background()

	// Empty context returns empty string.
	require.Equal(t, "", getWorkspaceID(ctx))

	// Round-trips through context.
	ctx = withWorkspaceID(ctx, "wk-test-123")
	require.Equal(t, "wk-test-123", getWorkspaceID(ctx))
}
