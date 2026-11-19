package mcp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

// testIdleTimeout is long enough that a request sent right after another one
// lands inside it on a loaded CI runner.
const testIdleTimeout = time.Second

// TestIdleSessionCloses pins that a session nobody deletes is closed once it
// has been idle for the timeout, and that its client can start over. The
// server gains bounded memory: Claude Code and other clients leave without
// DELETE, and before the timeout each such session held a goroutine for the
// life of the process (BYT-10375). The client gains a 404, which the MCP spec
// tells it to answer with a new initialize.
func TestIdleSessionCloses(t *testing.T) {
	_, client := newIdleTimeoutServer(t)
	session := client.initialize()

	client.waitUntilClosed(session)
	client.requireClosed(session)
	require.NotEqual(t, session, client.initialize(), "a fresh initialize must open a new session")
}

// TestRequestInFlightKeepsItsSessionOpen pins that the timeout counts idle
// time only. The user gains a long tool call that is not cut off when it
// outlasts the timeout. A confirmation form is held open the same way: the SDK
// shows it inside the tool call (TestAmbiguousDatabaseElicitsUnderLegacyClient).
// Once the call returns, the session is idle again and closes like any other.
func TestRequestInFlightKeepsItsSessionOpen(t *testing.T) {
	server, client := newIdleTimeoutServer(t)
	hold := addHoldTool(t, server)
	session := client.initialize()

	result := make(chan string, 1)
	go func() {
		status, body, err := client.post(context.Background(), session, callHold)
		if err != nil {
			result <- err.Error()
			return
		}
		result <- http.StatusText(status) + " " + body
	}()
	hold.waitStarted(t)
	time.Sleep(3 * testIdleTimeout)
	hold.release()
	select {
	case got := <-result:
		require.Contains(t, got, holdResult, "a call held past the timeout must still return its result")
	case <-time.After(10 * time.Second):
		require.FailNow(t, "the held call never returned")
	}

	status, _, err := client.post(context.Background(), session, listTools)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status, "the session must outlive a call that outlasted the timeout")

	client.waitUntilClosed(session)
	client.requireClosed(session)
}

// TestDroppedCallStillLetsItsSessionClose pins that a client which goes away
// in the middle of a call does not hold its session open. The server gains
// the same bound for a killed client as for one that left between calls: the
// connection closes with no DELETE and no cancellation sent, and once the call
// returns the session closes after the idle timeout.
func TestDroppedCallStillLetsItsSessionClose(t *testing.T) {
	server, client := newIdleTimeoutServer(t)
	hold := addHoldTool(t, server)
	session := client.initialize()

	ctx, drop := context.WithCancel(context.Background())
	dropped := make(chan error, 1)
	go func() {
		_, _, err := client.post(ctx, session, callHold)
		dropped <- err
	}()
	hold.waitStarted(t)
	drop()
	require.Error(t, <-dropped, "the dropped call must not have returned a result")
	hold.release()

	client.waitUntilClosed(session)
	client.requireClosed(session)
}

// TestProductionConstructionSetsAnIdleTimeout pins that the constructor the
// server uses applies the timeout. The other tests shorten it, so without
// this one the production path could drop it and the leak would come back.
func TestProductionConstructionSetsAnIdleTimeout(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := newServerWithStore(newTestServerStore(), profile, revalidationSecret, nil)
	require.NoError(t, err)
	require.Equal(t, mcpSessionIdleTimeout, s.sessionIdleTimeout)
	require.Positive(t, s.sessionIdleTimeout)
}

const (
	holdToolName = "hold_for_test"
	holdResult   = "released"
	callHold     = `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"` + holdToolName + `","arguments":{}}}`
	listTools    = `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`
	// A notification, because a POST that carries a call to a closing session
	// gets 200 and an empty stream: the SDK writes the stream's headers before
	// it learns the session is gone. A notification gets 404 once the session
	// leaves the SDK's map, so requireClosed retries until it does.
	probeNotification = `{"jsonrpc":"2.0","method":"notifications/roots/list_changed"}`
)

// newIdleTimeoutServer serves the real /mcp route with testIdleTimeout.
func newIdleTimeoutServer(t *testing.T) (*Server, *rawMCPClient) {
	t.Helper()
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := newServerWithIdleTimeout(newTestServerStore(), profile, revalidationSecret, nil, testIdleTimeout)
	require.NoError(t, err)
	e := echo.New()
	s.RegisterRoutes(e)
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	return s, &rawMCPClient{
		t:        t,
		endpoint: ts.URL + mcpResourcePath,
		token:    mintMCPToken(t, "test@example.com", "client-A", "ws-test", "mcp:read-only", time.Hour),
	}
}

// holdTool is a tool that runs until the test releases it.
type holdTool struct {
	started     chan struct{}
	releaseOnce sync.Once
	released    chan struct{}
}

func addHoldTool(t *testing.T, s *Server) *holdTool {
	t.Helper()
	h := &holdTool{started: make(chan struct{}, 1), released: make(chan struct{})}
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: holdToolName, Description: "Runs until the test releases it."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
			h.started <- struct{}{}
			select {
			case <-h.released:
				return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: holdResult}}}, nil, nil
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			}
		})
	t.Cleanup(h.release)
	return h
}

func (h *holdTool) release() { h.releaseOnce.Do(func() { close(h.released) }) }

func (h *holdTool) waitStarted(t *testing.T) {
	t.Helper()
	select {
	case <-h.started:
	case <-time.After(10 * time.Second):
		require.FailNow(t, "the held tool never started")
	}
}

// rawMCPClient speaks the streamable HTTP transport by hand. The SDK client
// would send DELETE when it closes and a cancellation when a call is dropped,
// which are the two things these tests need a client not to do.
type rawMCPClient struct {
	t        *testing.T
	endpoint string
	token    string
}

func (c *rawMCPClient) request(ctx context.Context, method, session, body string) *http.Request {
	c.t.Helper()
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint, strings.NewReader(body))
	require.NoError(c.t, err)
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json, text/event-stream")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if session != "" {
		req.Header.Set("Mcp-Session-Id", session)
		req.Header.Set("MCP-Protocol-Version", legacyProtocolCeiling)
	}
	return req
}

func (c *rawMCPClient) post(ctx context.Context, session, body string) (int, string, error) {
	resp, err := http.DefaultClient.Do(c.request(ctx, http.MethodPost, session, body))
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return resp.StatusCode, string(data), err
}

// initialize runs the handshake and returns the new session's ID.
func (c *rawMCPClient) initialize() string {
	c.t.Helper()
	resp, err := http.DefaultClient.Do(c.request(context.Background(), http.MethodPost, "",
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"`+legacyProtocolCeiling+
			`","capabilities":{},"clientInfo":{"name":"probe","version":"0"}}}`))
	require.NoError(c.t, err)
	_, err = io.Copy(io.Discard, resp.Body)
	require.NoError(c.t, err)
	require.NoError(c.t, resp.Body.Close())
	require.Equal(c.t, http.StatusOK, resp.StatusCode)
	session := resp.Header.Get("Mcp-Session-Id")
	require.NotEmpty(c.t, session)

	status, _, err := c.post(context.Background(), session, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	require.NoError(c.t, err)
	require.Equal(c.t, http.StatusAccepted, status)
	return session
}

// waitUntilClosed holds the session's GET stream open until the server ends
// it. The stream does not reset the idle clock, and the SDK ends it when the
// session closes, so its end is the closing signal.
func (c *rawMCPClient) waitUntilClosed(session string) {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*testIdleTimeout)
	defer cancel()
	resp, err := http.DefaultClient.Do(c.request(ctx, http.MethodGet, session, ""))
	require.NoError(c.t, err, "the GET stream must open")
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return
	}
	require.Equal(c.t, http.StatusOK, resp.StatusCode)
	_, err = io.Copy(io.Discard, resp.Body)
	require.NoError(c.t, err, "the server must end the GET stream when the session closes")
}

// requireClosed waits for the session to answer 404. Call it after
// waitUntilClosed: the GET stream ends when the transport closes, a moment
// before the SDK drops the session from its map, and in that moment the
// transport's buffered queue can still accept a notification with 202. A
// request then cannot reopen the session, so retrying is safe.
func (c *rawMCPClient) requireClosed(session string) {
	c.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		status, body, err := c.post(context.Background(), session, probeNotification)
		require.NoError(c.t, err)
		if status == http.StatusNotFound {
			return
		}
		require.True(c.t, time.Now().Before(deadline), "a closed session must answer 404, last got %d %q", status, body)
		time.Sleep(20 * time.Millisecond)
	}
}
