package mcp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

// newProbeServer stands up the real /mcp route over an internal API that
// records the credential of every internal call it receives.
func newProbeServer(t *testing.T) (endpoint string, credentials *[]string, mu *sync.Mutex) {
	t.Helper()
	var (
		lock  sync.Mutex
		creds []string
	)
	stub := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lock.Lock()
		creds = append(creds, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		lock.Unlock()
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{}`)
	})
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := newServerWithStore(newTestServerStore(), profile, revalidationSecret, stub)
	require.NoError(t, err)

	e := echo.New()
	s.RegisterRoutes(e)
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	return ts.URL + mcpResourcePath, &creds, &lock
}

// callAPITool drives one tool call that reaches the internal API.
func callAPITool(t *testing.T, session *mcp.ClientSession) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "call_api",
		Arguments: map[string]any{"operationId": "SQLService/Query"},
	})
	require.NoError(t, err)
}

// claim reads one claim out of an internal delegated credential. The signature
// is checked elsewhere; here the credential is only being observed.
func claim(t *testing.T, credential, name string) string {
	t.Helper()
	parsed, _, err := jwt.NewParser().ParseUnverified(credential, jwt.MapClaims{})
	require.NoError(t, err)
	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	value, ok := claims[name].(string)
	require.True(t, ok, "credential must carry a string %q claim", name)
	return value
}

// TestSessionlessProtocolRefused pins that /mcp answers a request announcing
// the sessionless revision with 400 rather than serving it.
//
// The SDK refuses that revision itself for every method but server/discover,
// which it answers on a stateful transport by opening a session the client is
// never told about and can never delete. go-sdk v1.7.0 clients send exactly
// that request on every connect, so without this refusal each connect strands
// one session for the life of the process.
func TestSessionlessProtocolRefused(t *testing.T) {
	endpoint, _, _ := newProbeServer(t)
	token := mintMCPToken(t, "test@example.com", "client-A", "ws-test", "mcp:read-only", time.Hour)

	// Verbatim from a go-sdk v1.7.0 client's first POST. Without the refusal the
	// SDK answers this 200 and keeps the session it opened to do so.
	discover := `{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{"_meta":{` +
		`"io.modelcontextprotocol/clientCapabilities":{"roots":{"listChanged":true}},` +
		`"io.modelcontextprotocol/clientInfo":{"name":"probe","version":"0"},` +
		`"io.modelcontextprotocol/protocolVersion":"2026-07-28"}}}`
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(discover))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("MCP-Protocol-Version", sessionlessProtocolVersion)
	req.Header.Set("Mcp-Method", "server/discover")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, resp.Body.Close())
	require.NoError(t, err)

	require.Equal(t, http.StatusBadRequest, resp.StatusCode,
		"a revision this transport cannot serve must be refused, not served into a stranded session")
	require.Contains(t, string(body), "unsupported protocol version",
		"the refusal must be ours: reaching the SDK means it opened a session to answer")
}

// TestSessionlessProtocolRefusalKeepsClientsWorking pins that the refusal costs
// nothing. A go-sdk v1.7.0 client probes for the sessionless revision first and
// falls back to the initialize handshake on any error, so it still connects and
// still runs tools.
func TestSessionlessProtocolRefusalKeepsClientsWorking(t *testing.T) {
	endpoint, credentials, mu := newProbeServer(t)
	token := mintMCPToken(t, "test@example.com", "client-A", "ws-test", "mcp:read-only", time.Hour)

	session := connectLegacy(t, endpoint, token, nil)
	callAPITool(t, session)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, *credentials, 1, "the tool call must still reach the internal API")
}

// TestRequestBodyCapRejectsBeforeDispatch pins the ingestion bound. A body over
// the cap is refused during the read, so an oversized script never reaches a
// tool — and never reaches the internal API behind it.
func TestRequestBodyCapRejectsBeforeDispatch(t *testing.T) {
	endpoint, credentials, mu := newProbeServer(t)
	token := mintMCPToken(t, "test@example.com", "client-A", "ws-test", "mcp:read-only", time.Hour)

	oversized := strings.Repeat("x", mcp.DefaultMaxRequestBodyBytes+(1<<16))
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":`+
		`{"name":"propose_database_change","arguments":{"database":"db","title":"t","sql":%q}}}`, oversized)
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode)

	mu.Lock()
	defer mu.Unlock()
	require.Empty(t, *credentials, "an over-limit body must be refused before any tool dispatch")
}

// TestRequestBodyCapEndsTheSession records what tripping the cap costs. The
// transport answers a bare 413, which the SDK client does not count as
// transient, so it fails the connection: the caller loses the session, not just
// the oversized call. v1.6.1 capped nothing, so this is new with the bump. If a
// later SDK makes 413 recoverable, this test is what notices.
func TestRequestBodyCapEndsTheSession(t *testing.T) {
	endpoint, _, _ := newProbeServer(t)
	token := mintMCPToken(t, "test@example.com", "client-A", "ws-test", "mcp:read-only", time.Hour)
	session := connectLegacy(t, endpoint, token, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	call := func(size int) error {
		_, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name:      "propose_database_change",
			Arguments: map[string]any{"database": "db", "title": "t", "sql": strings.Repeat("x", size)},
		})
		return err
	}
	require.NoError(t, call(1024), "a small call works before the oversized one")
	require.Error(t, call(maxMCPRequestBodyBytes+(1<<16)))
	require.Error(t, call(1024), "the session does not survive an over-limit body")
}

// TestCorrelationIDIsSessionScoped is the regression pin behind the
// mcp_correlation_id filter and the MCPDelegation.correlation_id comment: every
// tool call on one session must write one correlation ID, so an operator can
// pull an MCP session's audit rows with a single filter value.
//
// The ID is minted per HTTP request in authMiddleware, and stays session-scoped
// only because the SDK runs tool handlers on the context of the request that
// opened the session. Any SDK change to that plumbing would silently scatter a
// session's rows across as many IDs as it made calls, and this is what would
// catch it.
func TestCorrelationIDIsSessionScoped(t *testing.T) {
	endpoint, credentials, mu := newProbeServer(t)
	token := mintMCPToken(t, "test@example.com", "client-A", "ws-test", "mcp:read-only", time.Hour)

	session := connectLegacy(t, endpoint, token, nil)
	callAPITool(t, session)
	callAPITool(t, session)

	mu.Lock()
	seen := append([]string(nil), *credentials...)
	mu.Unlock()

	require.Len(t, seen, 2)
	first, second := claim(t, seen[0], "correlation_id"), claim(t, seen[1], "correlation_id")
	require.NotEmpty(t, first)
	require.Equal(t, first, second, "two tool calls on one session must share one correlation ID")
}

// TestSeparateSessionsGetSeparateCorrelationIDs is the other half: the ID
// identifies a session, so a second session must not reuse the first one's.
func TestSeparateSessionsGetSeparateCorrelationIDs(t *testing.T) {
	endpoint, credentials, mu := newProbeServer(t)
	token := mintMCPToken(t, "test@example.com", "client-A", "ws-test", "mcp:read-only", time.Hour)

	callAPITool(t, connectLegacy(t, endpoint, token, nil))
	callAPITool(t, connectLegacy(t, endpoint, token, nil))

	mu.Lock()
	seen := append([]string(nil), *credentials...)
	mu.Unlock()

	require.Len(t, seen, 2)
	require.NotEqual(t, claim(t, seen[0], "correlation_id"), claim(t, seen[1], "correlation_id"))
}

// TestInterleavedSessionsExecuteAsTheirOwnPrincipal pins that identity is
// per request, not per process: two sessions held open by different bearers,
// used in turn, each execute as their own principal. Nothing that a session
// carries may leak into a request that belongs to the other.
func TestInterleavedSessionsExecuteAsTheirOwnPrincipal(t *testing.T) {
	endpoint, credentials, mu := newProbeServer(t)

	alice := connectLegacy(t, endpoint,
		mintMCPToken(t, "alice@example.com", "client-A", "ws-test", "mcp:read-only", time.Hour), nil)
	bob := connectLegacy(t, endpoint,
		mintMCPToken(t, "bob@example.com", "client-B", "ws-test", "mcp:read-only", time.Hour), nil)

	for _, session := range []*mcp.ClientSession{alice, bob, alice, bob} {
		callAPITool(t, session)
	}

	mu.Lock()
	seen := append([]string(nil), *credentials...)
	mu.Unlock()

	require.Len(t, seen, 4)
	want := []string{"alice@example.com", "bob@example.com", "alice@example.com", "bob@example.com"}
	for i, principal := range want {
		require.Equal(t, principal, claim(t, seen[i], "sub"),
			"call %d must execute as its own bearer's principal", i)
	}
}
