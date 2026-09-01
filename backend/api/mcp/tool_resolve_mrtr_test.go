package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/component/config"
)

// The ambiguous-database flow is a multi round-trip request (SEP-2322): the
// tool returns an input request instead of an answer, and the client calls
// again carrying the response. These tests drive it end to end over HTTP,
// because the picking code reads the live session's advertised capabilities and
// a *mcp.ServerSession cannot be constructed outside the SDK.
//
// Two client generations reach the same handler:
//
//   - Clients on protocol <= 2025-11-25 never see the input request. The SDK's
//     own server middleware fulfills it by eliciting and re-invoking the
//     handler with the answer. This is every client Bytebase serves today: the
//     production handler is stateful, and the SDK caps the initialize handshake
//     at 2025-11-25 (mcp/shared.go negotiatedVersion), so 2026-07-28 is not
//     reachable there — see mrtrHandler.
//   - Clients on 2026-07-28 receive the input request themselves and retry.

const ambiguousShortName = "employee_db"

// ambiguousDatabases are two databases that share a short name on different
// instances, so resolving by short name alone is ambiguous.
func ambiguousDatabases() []map[string]any {
	return []map[string]any{
		makeDatabase("instances/prod-pg/databases/employee_db", "instances/prod-pg", "projects/hr", "POSTGRES", "ds-1"),
		makeDatabase("instances/staging-pg/databases/employee_db", "instances/staging-pg", "projects/hr", "POSTGRES", "ds-2"),
	}
}

// prodLabel is the elicitation label for the prod-pg candidate, built the same
// way candidateLabels builds it.
const prodLabel = "instances/prod-pg/databases/employee_db (prod-pg, POSTGRES)"

// newAmbiguousServer stands up the real /mcp route over an internal API that
// resolves ambiguously and answers any query. It returns the server plus the
// endpoint and a bearer for it.
func newAmbiguousServer(t *testing.T, databases []map[string]any) (*Server, string, string) {
	t.Helper()
	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	queryResp := makeQueryResponse([]string{"id"}, []string{"int4"}, [][]any{{"1"}}, "0.001s")

	s, err := newServerWithStore(newTestServerStore(), profile, revalidationSecret,
		mockQueryServer(databases, queryResp))
	require.NoError(t, err)

	e := echo.New()
	s.RegisterRoutes(e)
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)

	return s, ts.URL + "/mcp", mintMCPToken(t, "test@example.com", "client-A", "ws-test", "mcp:read-only", time.Hour)
}

// bearerTransport presents a fixed bearer on every request.
type bearerTransport struct{ token string }

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.token)
	return http.DefaultTransport.RoundTrip(req)
}

// connectLegacy opens a session against the production /mcp route.
func connectLegacy(t *testing.T, endpoint, token string, opts *mcp.ClientOptions) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "probe", Version: "0"}, opts)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:   endpoint,
		HTTPClient: &http.Client{Transport: &bearerTransport{token: token}},
		MaxRetries: -1,
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func queryAmbiguous(t *testing.T, session *mcp.ClientSession) (*mcp.CallToolResult, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "query_database",
		Arguments: map[string]any{"database": ambiguousShortName, "statement": "SELECT 1"},
	})
}

func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	require.NotEmpty(t, res.Content, "a completed call must carry content")
	text, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok)
	return text.Text
}

// TestAmbiguousDatabaseElicitsUnderLegacyClient is the pin that the migrated
// handler needs no protocol branch. A client that predates multi round-trip
// still gets a real elicitation, because the SDK's server middleware answers
// the input request on its behalf and re-invokes the handler; the call then
// completes against the database the user picked.
func TestAmbiguousDatabaseElicitsUnderLegacyClient(t *testing.T) {
	_, endpoint, token := newAmbiguousServer(t, ambiguousDatabases())

	var elicited atomic.Int32
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{
		ElicitationHandler: func(_ context.Context, req *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			elicited.Add(1)
			require.Contains(t, req.Params.Message, "Multiple databases match")
			return &mcp.ElicitResult{Action: "accept", Content: map[string]any{"database": prodLabel}}, nil
		},
	})
	require.Less(t, session.InitializeResult().ProtocolVersion, "2026-07-28",
		"the production handler is stateful, so the handshake stays below the multi round-trip revision")

	res, err := queryAmbiguous(t, session)
	require.NoError(t, err)
	require.False(t, res.IsError, resultText(t, res))
	require.Equal(t, int32(1), elicited.Load(), "the legacy client must still be asked")
	require.Contains(t, resultText(t, res), "instances/prod-pg/databases/employee_db")
}

// TestAmbiguousDatabaseFallsBackWithoutElicitationCapability pins the guard on
// the client's advertised capability. Most MCP clients advertise no
// elicitation, and returning an input request to one fails the whole tool call
// with a JSON-RPC error the model cannot act on
// (mcp/mrtr.go fulfillServerInputRequests). Such a client must keep getting the
// candidate listing it got before multi round-trip.
func TestAmbiguousDatabaseFallsBackWithoutElicitationCapability(t *testing.T) {
	_, endpoint, token := newAmbiguousServer(t, ambiguousDatabases())
	session := connectLegacy(t, endpoint, token, nil)

	res, err := queryAmbiguous(t, session)
	require.NoError(t, err, "a client that cannot be asked must still get a result, not a protocol error")
	require.True(t, res.IsError)
	text := resultText(t, res)
	require.Contains(t, text, "AMBIGUOUS_TARGET")
	require.Contains(t, text, "prod-pg")
	require.Contains(t, text, "staging-pg")
}

// TestAmbiguousDatabaseFallsBackForURLOnlyElicitation pins the other half of the
// capability guard. The input request asks for a form, and the SDK refuses form
// mode to a client that advertised URL elicitation without it, failing the whole
// tool call. Advertising some elicitation is therefore not enough to be asked.
func TestAmbiguousDatabaseFallsBackForURLOnlyElicitation(t *testing.T) {
	_, endpoint, token := newAmbiguousServer(t, ambiguousDatabases())
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{
		Capabilities: &mcp.ClientCapabilities{
			Elicitation: &mcp.ElicitationCapabilities{URL: &mcp.URLElicitationCapabilities{}},
		},
		ElicitationHandler: func(context.Context, *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			return nil, nil // never reached; a form request is refused before it
		},
	})

	res, err := queryAmbiguous(t, session)
	require.NoError(t, err, "a client that cannot answer a form must still get a result")
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "AMBIGUOUS_TARGET")
}

// TestAmbiguousDatabaseElicitsWhenNeitherModeAdvertised pins the backward-compat
// case: a client that advertises elicitation without naming a mode is assumed to
// support forms, so it must still be asked.
func TestAmbiguousDatabaseElicitsWhenNeitherModeAdvertised(t *testing.T) {
	_, endpoint, token := newAmbiguousServer(t, ambiguousDatabases())
	var elicited atomic.Int32
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{
		Capabilities: &mcp.ClientCapabilities{Elicitation: &mcp.ElicitationCapabilities{}},
		ElicitationHandler: func(context.Context, *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			elicited.Add(1)
			return &mcp.ElicitResult{Action: "accept", Content: map[string]any{"database": prodLabel}}, nil
		},
	})

	res, err := queryAmbiguous(t, session)
	require.NoError(t, err)
	require.False(t, res.IsError, resultText(t, res))
	require.Equal(t, int32(1), elicited.Load())
}

// TestAmbiguousDatabaseDeclinedFallsBack pins that declining still fails the
// call the way it did before the migration: with the candidate listing, not a
// silent pick.
func TestAmbiguousDatabaseDeclinedFallsBack(t *testing.T) {
	_, endpoint, token := newAmbiguousServer(t, ambiguousDatabases())
	session := connectLegacy(t, endpoint, token, &mcp.ClientOptions{
		ElicitationHandler: func(context.Context, *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			return &mcp.ElicitResult{Action: "decline"}, nil
		},
	})

	res, err := queryAmbiguous(t, session)
	require.NoError(t, err)
	require.True(t, res.IsError)
	require.Contains(t, resultText(t, res), "AMBIGUOUS_TARGET")
}

// TestAmbiguousDatabaseReturnsInputRequestOnProductionRoute is the pin that this
// migration matters in production. refuseSessionlessProtocol reads the
// MCP-Protocol-Version header, which an initialize request does not carry, so a
// client that asks for the multi round-trip revision in its initialize params is
// admitted and its session records that revision — even though the handshake
// answers 2025-11-25. Every later tool call on it must take the input-request
// path, because the SDK refuses to elicit on such a session.
func TestAmbiguousDatabaseReturnsInputRequestOnProductionRoute(t *testing.T) {
	_, endpoint, token := newAmbiguousServer(t, ambiguousDatabases())

	// status, the returned session ID, and the SSE data line.
	post := func(sessionID, body string) (int, string, string) {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Authorization", "Bearer "+token)
		if sessionID != "" {
			req.Header.Set("Mcp-Session-Id", sessionID)
		}
		resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		raw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		payload := string(raw)
		for _, line := range strings.Split(payload, "\n") {
			if after, found := strings.CutPrefix(line, "data: "); found {
				payload = after
				break
			}
		}
		return resp.StatusCode, resp.Header.Get("Mcp-Session-Id"), payload
	}

	status, sessionID, payload := post("", `{"jsonrpc":"2.0","id":1,"method":"initialize","params":`+
		`{"protocolVersion":"`+sessionlessProtocolVersion+`","capabilities":{"elicitation":{}},`+
		`"clientInfo":{"name":"probe","version":"0"}}}`)
	require.Equal(t, http.StatusOK, status)
	require.NotEmpty(t, sessionID)
	require.Contains(t, payload, `"protocolVersion":"2025-11-25"`,
		"the handshake answers the legacy revision even though the session records the newer one")

	post(sessionID, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)

	_, _, payload = post(sessionID, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":`+
		`{"name":"query_database","arguments":{"database":"`+ambiguousShortName+`","statement":"SELECT 1"}}}`)
	require.Contains(t, payload, `"resultType":"input_required"`)
	require.Contains(t, payload, `"inputRequests"`)
	require.Contains(t, payload, "Multiple databases match")
}

// shrinkingDatabases serves the full set on the first ListDatabases call and the
// reduced one after, so a tool call's two resolves disagree.
func shrinkingDatabases(t *testing.T, first, rest []map[string]any) (http.Handler, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	queryResp := makeQueryResponse([]string{"id"}, []string{"int4"}, [][]any{{"1"}}, "0.001s")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if !strings.Contains(r.URL.Path, "ListDatabases") {
			_ = json.NewEncoder(w).Encode(queryResp)
			return
		}
		set := first
		if calls.Add(1) > 1 {
			set = rest
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"databases": set})
	}), &calls
}

// TestAmbiguousDatabaseNeverRunsAnUnchosenDatabase is the correctness pin behind
// the whole round trip. The handler resolves once to ask and once to act, so the
// two resolves can disagree: here the database the user picked stops matching
// between them, leaving exactly one other match. The answer still decides, so
// the call must refuse rather than run the statement against the database that
// happens to be left.
func TestAmbiguousDatabaseNeverRunsAnUnchosenDatabase(t *testing.T) {
	both := ambiguousDatabases()
	handler, listCalls := shrinkingDatabases(t, both, both[1:]) // staging survives, prod does not

	profile := &config.Profile{Mode: common.ReleaseModeDev, ExternalURL: "https://bb.example.com"}
	s, err := newServerWithStore(newTestServerStore(), profile, revalidationSecret, handler)
	require.NoError(t, err)
	e := echo.New()
	s.RegisterRoutes(e)
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)

	token := mintMCPToken(t, "test@example.com", "client-A", "ws-test", "mcp:read-only", time.Hour)
	session := connectLegacy(t, ts.URL+mcpResourcePath, token, &mcp.ClientOptions{
		ElicitationHandler: func(context.Context, *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			return &mcp.ElicitResult{Action: "accept", Content: map[string]any{"database": prodLabel}}, nil
		},
	})

	res, err := queryAmbiguous(t, session)
	require.NoError(t, err)
	require.Equal(t, int32(2), listCalls.Load(), "the retry must resolve again")
	require.True(t, res.IsError, "picking a database that no longer matches must not run somewhere else")
	text := resultText(t, res)
	require.Contains(t, text, "STALE_TARGET")
	require.NotContains(t, text, "Query:", "no statement may run")
	require.NotContains(t, text, "Specify instance or project to narrow",
		"the ambiguous remedy would send the caller straight back to the database it did not pick")
}

// mrtrEndpoint serves the same tools behind the same auth chain, over a
// STATELESS transport. That is the only way to reach a 2026-07-28 client: the
// SDK refuses the revision on a stateful transport
// (mcp/streamable.go SupportsProtocolVersion) and caps the initialize handshake
// at 2025-11-25 regardless (mcp/shared.go negotiatedVersion). Production stays
// stateful, so these tests pin the tool handler under the newer protocol, not
// the /mcp wiring: refuseSessionlessProtocol is deliberately left off here,
// since on the real route it refuses exactly what these tests need to send.
func mrtrEndpoint(t *testing.T, s *Server) string {
	t.Helper()
	streamable := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return s.mcpServer },
		&mcp.StreamableHTTPOptions{
			DisableLocalhostProtection: true,
			MaxRequestBodyBytes:        maxMCPRequestBodyBytes,
			Stateless:                  true,
		})
	e := echo.New()
	e.Any(mcpResourcePath,
		echo.WrapHandler(mcpauth.RequireBearerToken(s.verifySessionBinding, nil)(streamable)),
		s.authMiddleware)
	ts := httptest.NewServer(e)
	t.Cleanup(ts.Close)
	return ts.URL + mcpResourcePath
}

// connectMRTR opens a session that speaks the 2026-07-28 protocol and does not
// auto-answer input requests, so the test sees both round trips.
func connectMRTR(t *testing.T, endpoint, token string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "probe", Version: "0"}, &mcp.ClientOptions{
		MultiRoundTrip: &mcp.MultiRoundTripOptions{Disabled: true},
		ElicitationHandler: func(context.Context, *mcp.ElicitRequest) (*mcp.ElicitResult, error) {
			return nil, nil // never called; declared so the capability is advertised
		},
	})
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:   endpoint,
		HTTPClient: &http.Client{Transport: &bearerTransport{token: token}},
		MaxRetries: -1,
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	require.Equal(t, "2026-07-28", session.InitializeResult().ProtocolVersion)
	return session
}

func callMRTR(t *testing.T, session *mcp.ClientSession, responses mcp.InputResponseMap, state string) *mcp.CallToolResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:           "query_database",
		Arguments:      map[string]any{"database": ambiguousShortName, "statement": "SELECT 1"},
		InputResponses: responses,
		RequestState:   state,
	})
	require.NoError(t, err)
	return res
}

// TestAmbiguousDatabaseReturnsInputRequestUnderMRTRClient pins the shape a
// 2026-07-28 client sees: the first call answers with an input request instead
// of content, and the retry carrying the response completes the query.
func TestAmbiguousDatabaseReturnsInputRequestUnderMRTRClient(t *testing.T) {
	s, _, token := newAmbiguousServer(t, ambiguousDatabases())
	session := connectMRTR(t, mrtrEndpoint(t, s), token)

	first := callMRTR(t, session, nil, "")
	require.True(t, first.NeedsInput(), "an ambiguous match must ask, not guess")
	require.Empty(t, first.Content, "content and input requests are mutually exclusive")
	elicit, ok := first.InputRequests[databaseChoiceRequestID].(*mcp.ElicitParams)
	require.True(t, ok, "the input request must be an elicitation")
	require.Contains(t, elicit.Message, "Multiple databases match")

	second := callMRTR(t, session, mcp.InputResponseMap{
		databaseChoiceRequestID: &mcp.ElicitResult{
			Action:  "accept",
			Content: map[string]any{"database": prodLabel},
		},
	}, first.RequestState)
	require.False(t, second.NeedsInput())
	require.False(t, second.IsError, resultText(t, second))
	require.Contains(t, resultText(t, second), "instances/prod-pg/databases/employee_db")
}

// TestAmbiguousDatabaseStaleSelectionFallsBack pins what happens when the
// candidate set changes between the two halves of a call. The selection is
// matched against the candidates resolved on the retry, so a label that no
// longer names a candidate resolves to nothing.
//
// It answers STALE_TARGET rather than asking again: a handler that returns a
// second input request is retried until the SDK gives up ("multi-round-trip:
// exceeded maximum retries"), which fails the call with a protocol error
// instead of telling the model what it can pick from.
func TestAmbiguousDatabaseStaleSelectionFallsBack(t *testing.T) {
	s, _, token := newAmbiguousServer(t, ambiguousDatabases())
	session := connectMRTR(t, mrtrEndpoint(t, s), token)

	first := callMRTR(t, session, nil, "")
	require.True(t, first.NeedsInput())

	// The retry names a database that is not among the candidates any more.
	second := callMRTR(t, session, mcp.InputResponseMap{
		databaseChoiceRequestID: &mcp.ElicitResult{
			Action:  "accept",
			Content: map[string]any{"database": "instances/gone/databases/employee_db (gone, POSTGRES)"},
		},
	}, first.RequestState)
	require.False(t, second.NeedsInput(), "a stale selection must not start another round trip")
	require.True(t, second.IsError)
	require.Contains(t, resultText(t, second), "STALE_TARGET")
}
