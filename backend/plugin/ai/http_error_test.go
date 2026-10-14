package ai

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{name: "absent", value: "", want: 0},
		{name: "seconds", value: "7", want: 7 * time.Second},
		{name: "seconds with spaces", value: " 120 ", want: 2 * time.Minute},
		{name: "fraction of a second", value: "0.5", want: 500 * time.Millisecond},
		{name: "every digit down to the nanosecond", value: "123456789.123456789", want: 123456789123456789},
		{name: "digits past the nanosecond", value: "1.0000000019", want: time.Second + time.Nanosecond},
		{name: "longest count of whole seconds", value: "9223372036", want: 9223372036 * time.Second},
		{name: "longest duration", value: "9223372036.854775807", want: math.MaxInt64},
		{name: "zero", value: "0", want: 0},
		{name: "negative", value: "-3", want: 0},
		{name: "signed", value: "+3", want: 0},
		{name: "not a number", value: "NaN", want: 0},
		{name: "infinite", value: "Inf", want: 0},
		{name: "exponent", value: "1e3", want: 0},
		{name: "hexadecimal", value: "0x10", want: 0},
		{name: "one nanosecond too long for a duration", value: "9223372036.854775808", want: math.MaxInt64},
		{name: "one second too long for a duration", value: "9223372037", want: math.MaxInt64},
		{name: "too long for an integer", value: strings.Repeat("9", 400), want: math.MaxInt64},
		{name: "HTTP date in the future", value: "Wed, 14 Oct 2026 08:00:45 GMT", want: 45 * time.Second},
		{name: "HTTP date in the past", value: "Wed, 14 Oct 2026 07:59:00 GMT", want: 0},
		{name: "obsolete HTTP date format", value: "Wednesday, 14-Oct-26 08:02:00 GMT", want: 2 * time.Minute},
		{name: "malformed", value: "soon", want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, test.want, parseRetryAfter(test.value, now))
		})
	}
}

// failingProvider serves one reply to every request.
func failingProvider(t *testing.T, status int, header map[string]string, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		for name, value := range header {
			w.Header().Set(name, value)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func chatOnce(ctx context.Context, provider storepb.AISetting_Provider, endpoint string) error {
	content := "hello"
	_, err := Chat(ctx, &storepb.AISetting{Provider: provider, Endpoint: endpoint, Model: "test-model", ApiKey: "test-key"}, &v1pb.AIChatRequest{
		Messages: []*v1pb.AIChatMessage{{Role: v1pb.AIChatMessageRole_AI_CHAT_MESSAGE_ROLE_USER, Content: &content}},
	})
	return err
}

func TestChatReturnsAnHTTPErrorForAFailedReply(t *testing.T) {
	t.Parallel()

	const (
		openAIRateLimit = `{"error": {"message": "Rate limit reached for requests", "type": "requests", "param": null, "code": "rate_limit_exceeded"}}`
		openAIQuota     = `{"error": {"message": "You exceeded your current quota, please check your plan and billing details.", "type": "insufficient_quota", "param": null, "code": "insufficient_quota"}}`
		azureRateLimit  = `{"error": {"code": "429", "message": "Requests have exceeded the token rate limit of your current pricing tier."}}`
		numericCode     = `{"error": {"message": "insufficient quota", "type": "insufficient_quota", "code": 429}}`
		claudeRateLimit = `{"type": "error", "error": {"type": "rate_limit_error", "message": "This request would exceed your organization's rate limit of 1,000 requests per minute."}, "request_id": "req_1"}`
		claudeSpendCap  = `{"type": "error", "error": {"type": "rate_limit_error", "message": "You have reached your API usage limits.", "details": {"error_code": "enforced_spend_limit_reached"}}, "request_id": "req_2"}`
		claudeOverload  = `{"type": "error", "error": {"type": "overloaded_error", "message": "Overloaded"}, "request_id": "req_3"}`
		geminiRateLimit = `{"error": {"code": 429, "message": "Resource has been exhausted (e.g. check quota).", "status": "RESOURCE_EXHAUSTED", "details": [{"@type": "type.googleapis.com/google.rpc.QuotaFailure", "violations": [{"quotaId": "GenerateRequestsPerMinutePerProjectPerModel"}]}, {"@type": "type.googleapis.com/google.rpc.RetryInfo", "retryDelay": "34.5s"}]}}`
		geminiNoDetails = `{"error": {"code": 503, "message": "The model is overloaded. Please try again later.", "status": "UNAVAILABLE"}}`
		geminiLongDelay = `{"error": {"code": 429, "status": "RESOURCE_EXHAUSTED", "details": [{"@type": "type.googleapis.com/google.rpc.RetryInfo", "retryDelay": "99999999999999999999s"}]}}`
		geminiBadDelay  = `{"error": {"code": 429, "status": "RESOURCE_EXHAUSTED", "details": [{"@type": "type.googleapis.com/google.rpc.RetryInfo", "retryDelay": "-5s"}, {"@type": "type.googleapis.com/google.rpc.RetryInfo", "retryDelay": "Infs"}, {"@type": "type.googleapis.com/google.rpc.RetryInfo", "retryDelay": "7"}, {"@type": "type.googleapis.com/google.rpc.Help", "retryDelay": "9s"}]}}`
		notJSON         = `upstream connect error or disconnect/reset before headers`
	)
	tests := []struct {
		name     string
		provider storepb.AISetting_Provider
		path     string
		status   int
		header   map[string]string
		body     string
		want     HTTPError
	}{
		{
			name:     "OpenAI rate limit with the wait in the header",
			provider: storepb.AISetting_OPEN_AI,
			path:     "/v1/chat/completions",
			status:   http.StatusTooManyRequests,
			header:   map[string]string{"Retry-After": "20"},
			body:     openAIRateLimit,
			want:     HTTPError{Provider: "OpenAI", RetryAfter: 20 * time.Second},
		},
		{
			name:     "OpenAI used-up quota",
			provider: storepb.AISetting_OPEN_AI,
			path:     "/v1/chat/completions",
			status:   http.StatusTooManyRequests,
			body:     openAIQuota,
			want:     HTTPError{Provider: "OpenAI", QuotaExhausted: true},
		},
		{
			name:     "OpenAI used-up quota on the responses endpoint",
			provider: storepb.AISetting_OPEN_AI,
			path:     "/v1/responses",
			status:   http.StatusTooManyRequests,
			body:     openAIQuota,
			want:     HTTPError{Provider: "OpenAI", QuotaExhausted: true},
		},
		{
			name:     "Azure OpenAI rate limit",
			provider: storepb.AISetting_AZURE_OPENAI,
			path:     "/openai/deployments/test/chat/completions",
			status:   http.StatusTooManyRequests,
			header:   map[string]string{"Retry-After": "3"},
			body:     azureRateLimit,
			want:     HTTPError{Provider: "OpenAI", RetryAfter: 3 * time.Second},
		},
		{
			name:     "used-up quota from a service that sends a numeric code",
			provider: storepb.AISetting_OPEN_AI,
			path:     "/v1/chat/completions",
			status:   http.StatusTooManyRequests,
			body:     numericCode,
			want:     HTTPError{Provider: "OpenAI", QuotaExhausted: true},
		},
		{
			name:     "OpenAI bad key",
			provider: storepb.AISetting_OPEN_AI,
			path:     "/v1/chat/completions",
			status:   http.StatusUnauthorized,
			body:     `{"error": {"message": "Incorrect API key provided", "type": "invalid_request_error", "code": "invalid_api_key"}}`,
			want:     HTTPError{Provider: "OpenAI"},
		},
		{
			name:     "Claude rate limit with the wait in the header",
			provider: storepb.AISetting_CLAUDE,
			status:   http.StatusTooManyRequests,
			header:   map[string]string{"Retry-After": "12"},
			body:     claudeRateLimit,
			want:     HTTPError{Provider: "Claude", RetryAfter: 12 * time.Second},
		},
		{
			name:     "Claude spend cap",
			provider: storepb.AISetting_CLAUDE,
			status:   http.StatusTooManyRequests,
			body:     claudeSpendCap,
			want:     HTTPError{Provider: "Claude", QuotaExhausted: true},
		},
		{
			name:     "Claude rate limit without a wait",
			provider: storepb.AISetting_CLAUDE,
			status:   http.StatusTooManyRequests,
			body:     claudeRateLimit,
			want:     HTTPError{Provider: "Claude"},
		},
		{
			name:     "Claude overloaded",
			provider: storepb.AISetting_CLAUDE,
			status:   529,
			body:     claudeOverload,
			want:     HTTPError{Provider: "Claude"},
		},
		{
			name:     "Gemini rate limit with the wait in the body",
			provider: storepb.AISetting_GEMINI,
			status:   http.StatusTooManyRequests,
			body:     geminiRateLimit,
			want:     HTTPError{Provider: "Gemini", RetryAfter: 34500 * time.Millisecond},
		},
		{
			name:     "Gemini takes the longer of the header and the body",
			provider: storepb.AISetting_GEMINI,
			status:   http.StatusTooManyRequests,
			header:   map[string]string{"Retry-After": "60"},
			body:     geminiRateLimit,
			want:     HTTPError{Provider: "Gemini", RetryAfter: time.Minute},
		},
		{
			name:     "Gemini delay too long for a duration",
			provider: storepb.AISetting_GEMINI,
			status:   http.StatusTooManyRequests,
			body:     geminiLongDelay,
			want:     HTTPError{Provider: "Gemini", RetryAfter: math.MaxInt64},
		},
		{
			name:     "Gemini delays that are negative, malformed, or of another detail type",
			provider: storepb.AISetting_GEMINI,
			status:   http.StatusTooManyRequests,
			body:     geminiBadDelay,
			want:     HTTPError{Provider: "Gemini"},
		},
		{
			name:     "Gemini unavailable without details",
			provider: storepb.AISetting_GEMINI,
			status:   http.StatusServiceUnavailable,
			body:     geminiNoDetails,
			want:     HTTPError{Provider: "Gemini"},
		},
		{
			name:     "OpenAI reply that is not JSON",
			provider: storepb.AISetting_OPEN_AI,
			path:     "/v1/chat/completions",
			status:   http.StatusBadGateway,
			body:     notJSON,
			want:     HTTPError{Provider: "OpenAI"},
		},
		{
			name:     "Claude reply that is not JSON",
			provider: storepb.AISetting_CLAUDE,
			status:   http.StatusBadGateway,
			body:     notJSON,
			want:     HTTPError{Provider: "Claude"},
		},
		{
			name:     "Gemini reply that is not JSON",
			provider: storepb.AISetting_GEMINI,
			status:   http.StatusBadGateway,
			body:     notJSON,
			want:     HTTPError{Provider: "Gemini"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := failingProvider(t, test.status, test.header, test.body)
			err := chatOnce(context.Background(), test.provider, server.URL+test.path)

			var httpErr *HTTPError
			require.ErrorAs(t, err, &httpErr)
			want := test.want
			want.StatusCode = test.status
			want.Body = test.body
			require.Equal(t, want, *httpErr)
			require.EqualError(t, err, fmt.Sprintf("%s API returned status %d: %s", want.Provider, test.status, test.body))
		})
	}
}

func TestChatReturnsATransportErrorForACallWithNoReply(t *testing.T) {
	t.Parallel()

	t.Run("canceled context", func(t *testing.T) {
		t.Parallel()

		server := failingProvider(t, http.StatusOK, nil, `{}`)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := chatOnce(ctx, storepb.AISetting_GEMINI, server.URL)

		var transportErr *TransportError
		require.ErrorAs(t, err, &transportErr)
		require.NotErrorIs(t, err, context.Canceled, "the caller asks its context, the error does not unwrap")
		require.ErrorContains(t, err, "failed to send HTTP request to Gemini: Post ")
		require.ErrorContains(t, err, ": context canceled")
	})

	t.Run("expired context", func(t *testing.T) {
		t.Parallel()

		server := failingProvider(t, http.StatusOK, nil, `{}`)
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()
		err := chatOnce(ctx, storepb.AISetting_CLAUDE, server.URL)

		var transportErr *TransportError
		require.ErrorAs(t, err, &transportErr)
		require.NotErrorIs(t, err, context.DeadlineExceeded)
		require.ErrorContains(t, err, "failed to send HTTP request to Claude: Post ")
	})

	t.Run("connection closed before the reply", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			closeConnection(t, w, "")
		}))
		defer server.Close()
		err := chatOnce(context.Background(), storepb.AISetting_OPEN_AI, server.URL+"/v1/chat/completions")

		var transportErr *TransportError
		require.ErrorAs(t, err, &transportErr)
		require.NotErrorIs(t, err, context.Canceled)
		require.NotErrorIs(t, err, context.DeadlineExceeded)
		require.ErrorContains(t, err, "failed to send HTTP request to OpenAI: Post ")
	})

	t.Run("connection closed in the middle of the reply", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			closeConnection(t, w, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 100\r\n\r\n{\"candidates\": [")
		}))
		defer server.Close()
		err := chatOnce(context.Background(), storepb.AISetting_GEMINI, server.URL)

		var transportErr *TransportError
		require.ErrorAs(t, err, &transportErr)
		require.EqualError(t, err, "failed to read Gemini response body: unexpected EOF")
	})
}

// closeConnection writes the raw bytes, if any, and drops the connection.
func closeConnection(t *testing.T, w http.ResponseWriter, raw string) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		t.Error("the response writer cannot be hijacked")
		return
	}
	conn, _, err := hijacker.Hijack()
	if err != nil {
		t.Errorf("failed to hijack the connection: %v", err)
		return
	}
	_, _ = conn.Write([]byte(raw))
	_ = conn.Close()
}
