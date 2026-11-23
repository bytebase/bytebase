package mcp

import (
	"bufio"
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

// testFormTimeout is shorter than testIdleTimeout, as formAnswerTimeout is
// shorter than mcpSessionIdleTimeout.
const testFormTimeout = testIdleTimeout / 4

// testRequestBound ends any POST a test sends, so a hung one fails the test
// instead of running to the go test timeout.
const testRequestBound = 10 * time.Second

func TestIdleSessionCloses(t *testing.T) {
	_, client := newIdleTimeoutServer(t)
	session := client.initialize(noCapabilities)

	client.reopenGETsUntilClosed(session)
	client.requireClosed(session, 5*time.Second)
	status, _, err := client.post(context.Background(), session, listTools)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, status, "a call to a closed session must get 404")
	client.initialize(noCapabilities)
}

func TestRequestInFlightKeepsItsSessionOpen(t *testing.T) {
	server, client := newIdleTimeoutServer(t)
	hold := addHoldTool(t, server)
	session := client.initialize(noCapabilities)

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
	require.Contains(t, <-result, holdResult, "a call held past both timeouts must still return its result")

	status, body, err := client.post(context.Background(), session, listTools)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status)
	require.Contains(t, body, holdToolName, "the session must outlive a call that outlasted the timeout")

	client.holdGETUntilClosed(session)
	client.requireClosed(session, 5*time.Second)
}

func TestDroppedCallStillLetsItsSessionClose(t *testing.T) {
	server, client := newIdleTimeoutServer(t)
	hold := addHoldTool(t, server)
	session := client.initialize(noCapabilities)

	ctx, drop := context.WithCancel(context.Background())
	dropped := make(chan error, 1)
	go func() {
		_, _, err := client.post(ctx, session, callHold)
		dropped <- err
	}()
	hold.waitStarted(t)
	drop()
	require.Error(t, <-dropped, "the dropped call must not have returned a result")
	time.Sleep(3 * testIdleTimeout)
	hold.release()

	// Well inside one timeout: the timer fired during the hold, so the session
	// closes as soon as the call returns.
	client.requireClosed(session, testIdleTimeout/2)
}

func TestAbandonedFormLetsItsSessionClose(t *testing.T) {
	server, client := newIdleTimeoutServer(t)
	addFormTool(t, server)
	session := client.initialize(formCapabilities)

	ctx, drop := context.WithTimeout(context.Background(), testRequestBound)
	defer drop()
	client.openForm(ctx, session)
	drop()
	time.Sleep(3 * testIdleTimeout)

	status, body, err := client.post(context.Background(), session, listTools)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, status, "the first call of a client back after the timeout must get 404, got %q", body)
}

func TestUnansweredFormEndsItsCall(t *testing.T) {
	server, client := newIdleTimeoutServer(t)
	addFormTool(t, server)
	session := client.initialize(formCapabilities)

	ctx, cancel := context.WithTimeout(context.Background(), testRequestBound)
	defer cancel()
	start := time.Now()
	rest, err := io.ReadAll(client.openForm(ctx, session))
	require.NoError(t, err, "the call must end when the form times out")
	require.Less(t, time.Since(start), testIdleTimeout, "the call must end at the form limit, not the idle timeout")
	require.Contains(t, string(rest), `"error"`, "the call must end with an error")

	status, body, err := client.post(context.Background(), session, listTools)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, status)
	require.Contains(t, body, formToolName, "the session must outlive the form")
}

// TestProductionTimeouts pins the production values, which the other tests
// shorten.
func TestProductionTimeouts(t *testing.T) {
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := newServerWithStore(newTestServerStore(), profile, revalidationSecret, nil)
	require.NoError(t, err)
	require.Equal(t, sessionTimeouts{idle: mcpSessionIdleTimeout, formAnswer: formAnswerTimeout}, s.timeouts)
	require.Greater(t, s.timeouts.formAnswer, confirmationLifetime, "a confirmation accepted late must reach the tool and get STALE_CONFIRMATION")
	require.Less(t, s.timeouts.formAnswer, s.timeouts.idle, "no wait for an abandoned form may outlive the session's idle timer")
}

const (
	noCapabilities   = `{}`
	formCapabilities = `{"elicitation":{"form":{}}}`

	holdToolName = "hold_for_test"
	holdResult   = "released"
	callHold     = `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"` + holdToolName + `","arguments":{}}}`
	listTools    = `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`
	formToolName = "form_for_test"
	callForm     = `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"` + formToolName + `","arguments":{}}}`
	// A notification, because a call to a closing session gets 200 and an
	// empty stream, and its refused reply cancels the calls still in flight.
	// A notification gets 404 once the session leaves the SDK's map.
	probeNotification = `{"jsonrpc":"2.0","method":"notifications/roots/list_changed"}`
)

// newIdleTimeoutServer serves the real /mcp route with testIdleTimeout and
// testFormTimeout. The hold tests' tools run past testFormTimeout, which pins
// that the form limit bounds only the wait for an answer.
func newIdleTimeoutServer(t *testing.T) (*Server, *rawMCPClient) {
	t.Helper()
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := newServerWithTimeouts(newTestServerStore(), profile, revalidationSecret, nil, sessionTimeouts{idle: testIdleTimeout, formAnswer: testFormTimeout})
	require.NoError(t, err)
	e := echo.New()
	s.RegisterRoutes(e)
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	return s, &rawMCPClient{
		t:          t,
		httpClient: ts.Client(),
		endpoint:   ts.URL + mcpResourcePath,
		token:      mintMCPToken(t, "test@example.com", "client-A", "ws-test", "mcp:read-only", time.Hour),
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
	case <-time.After(testRequestBound):
		require.FailNow(t, "the held tool never started")
	}
}

// formTestWrite is the write the form tool asks the user to confirm.
type formTestWrite struct{}

func (formTestWrite) confirmedProcedure() string { return "/bytebase.v1.TestService/Write" }
func (formTestWrite) confirmedResource() string  { return "test" }

// addFormTool adds a tool that asks to confirm a write the way the write tools
// do. The tests never answer it.
func addFormTool(t *testing.T, s *Server) {
	t.Helper()
	mcp.AddTool(s.mcpServer, &mcp.Tool{Name: formToolName, Description: "Asks to confirm a write."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
			return s.askToConfirm(ctx, formTestWrite{}, []byte("digest"), "Make the test write?",
				confirmChoice{question: "Make the test write?", action: "Make it"}), nil, nil
		})
}

// rawMCPClient speaks the streamable HTTP transport by hand. The SDK client
// would send DELETE when it closes and a cancellation when a call is dropped,
// which are the two things these tests need a client not to do.
type rawMCPClient struct {
	t          *testing.T
	httpClient *http.Client
	endpoint   string
	token      string
}

func (c *rawMCPClient) request(ctx context.Context, method, session, body string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.endpoint, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json, text/event-stream")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if session != "" {
		req.Header.Set("Mcp-Session-Id", session)
		req.Header.Set("MCP-Protocol-Version", legacyProtocolCeiling)
	}
	return req, nil
}

// post can run outside the test goroutine, so it returns errors instead of
// failing the test.
func (c *rawMCPClient) post(ctx context.Context, session, body string) (int, string, error) {
	ctx, cancel := context.WithTimeout(ctx, testRequestBound)
	defer cancel()
	req, err := c.request(ctx, http.MethodPost, session, body)
	if err != nil {
		return 0, "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return resp.StatusCode, string(data), err
}

// initialize runs the handshake, declaring capabilities, and returns the new
// session's ID.
func (c *rawMCPClient) initialize(capabilities string) string {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), testRequestBound)
	defer cancel()
	req, err := c.request(ctx, http.MethodPost, "",
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"`+legacyProtocolCeiling+
			`","capabilities":`+capabilities+`,"clientInfo":{"name":"probe","version":"0"}}}`)
	require.NoError(c.t, err)
	resp, err := c.httpClient.Do(req)
	require.NoError(c.t, err)
	_, err = io.Copy(io.Discard, resp.Body)
	require.NoError(c.t, err)
	require.NoError(c.t, resp.Body.Close())
	require.Equal(c.t, http.StatusOK, resp.StatusCode)
	session := resp.Header.Get("Mcp-Session-Id")
	require.NotEmpty(c.t, session)

	status, _, err := c.post(ctx, session, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	require.NoError(c.t, err)
	require.Equal(c.t, http.StatusAccepted, status)
	return session
}

// openForm calls the form tool and reads its stream until the form arrives.
// It returns the rest of the stream, which ends with the call.
func (c *rawMCPClient) openForm(ctx context.Context, session string) io.Reader {
	c.t.Helper()
	req, err := c.request(ctx, http.MethodPost, session, callForm)
	require.NoError(c.t, err)
	resp, err := c.httpClient.Do(req)
	require.NoError(c.t, err)
	c.t.Cleanup(func() { _ = resp.Body.Close() })
	require.Equal(c.t, http.StatusOK, resp.StatusCode)
	stream := bufio.NewReader(resp.Body)
	for {
		line, err := stream.ReadString('\n')
		require.NoError(c.t, err, "the form must reach the client")
		if strings.Contains(line, `"method":"elicitation/create"`) {
			return stream
		}
	}
}

// holdGETUntilClosed holds one GET stream on the session until the server
// ends it. The stream does not reset the idle clock, and the SDK ends it when
// the session closes, so its end is the closing signal.
func (c *rawMCPClient) holdGETUntilClosed(session string) {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*testIdleTimeout)
	defer cancel()
	req, err := c.request(ctx, http.MethodGet, session, "")
	require.NoError(c.t, err)
	resp, err := c.httpClient.Do(req)
	require.NoError(c.t, err, "the GET stream must open")
	defer resp.Body.Close()
	require.Equal(c.t, http.StatusOK, resp.StatusCode, "the GET stream must open on a live session")
	_, err = io.Copy(io.Discard, resp.Body)
	require.NoError(c.t, err, "the server must end the GET stream when the session closes")
}

// reopenGETsUntilClosed opens a new GET stream on the session every quarter of
// the timeout, holding each until the next, and returns once the server ends
// one or answers 404. Neither a new GET nor a held one resets the idle clock,
// so the session must close anyway. A 404 counts only after some GET opened:
// a session that closes between two GETs shows only as a 404 on the next.
func (c *rawMCPClient) reopenGETsUntilClosed(session string) {
	c.t.Helper()
	deadline := time.Now().Add(10 * testIdleTimeout)
	opened := false
	for {
		status, ended := c.holdGET(session, testIdleTimeout/4)
		opened = opened || status == http.StatusOK
		if ended || status == http.StatusNotFound {
			require.True(c.t, opened, "a GET stream must open while the session is live")
			return
		}
		require.True(c.t, time.Now().Before(deadline), "a session must close while new GET streams keep arriving")
		if status != http.StatusOK {
			// 0: the GET ran out before its headers. 409: the server has not
			// yet let go of the stream before.
			require.Contains(c.t, []int{0, http.StatusConflict}, status, "unexpected GET status")
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// holdGET holds a GET stream on the session for up to d. It returns the status,
// or 0 when d ran out before it, and whether the server ended the stream first,
// as it does when the session closes.
func (c *rawMCPClient) holdGET(session string, d time.Duration) (int, bool) {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	req, err := c.request(ctx, http.MethodGet, session, "")
	require.NoError(c.t, err)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		require.ErrorIs(c.t, err, context.DeadlineExceeded)
		return 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, false
	}
	_, err = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, err == nil
}

// requireClosed waits up to within for the session to answer 404. The GET
// stream ends when the transport closes, a moment before the SDK drops the
// session from its map, and in that moment the transport's buffered queue can
// still accept a notification with 202.
func (c *rawMCPClient) requireClosed(session string, within time.Duration) {
	c.t.Helper()
	deadline := time.Now().Add(within)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	for {
		status, body, err := c.post(ctx, session, probeNotification)
		require.NoError(c.t, err, "a closed session must answer 404 within %v", within)
		if status == http.StatusNotFound {
			return
		}
		require.True(c.t, time.Now().Before(deadline), "a closed session must answer 404 within %v, last got %d %q", within, status, body)
		time.Sleep(20 * time.Millisecond)
	}
}
