package aireview

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/ai"
)

const (
	testAPIKey = "test-key-never-logged"

	geminiOK = `{"candidates": [{"content": {"parts": [{"text": "ok"}]}}], "usageMetadata": {"totalTokenCount": 7}}`
	claudeOK = `{"content": [{"type": "text", "text": "ok"}], "usage": {"input_tokens": 3, "output_tokens": 4}}`
	openAIOK = `{"choices": [{"message": {"role": "assistant", "content": "ok"}}], "usage": {"total_tokens": 7}}`

	geminiUnavailable = `{"error": {"code": 503, "message": "The model is overloaded. Please try again later.", "status": "UNAVAILABLE"}}`
	geminiRateLimit   = `{"error": {"code": 429, "message": "Resource has been exhausted (e.g. check quota).", "status": "RESOURCE_EXHAUSTED", "details": [{"@type": "type.googleapis.com/google.rpc.RetryInfo", "retryDelay": "13s"}]}}`
	claudeRateLimit   = `{"type": "error", "error": {"type": "rate_limit_error", "message": "This request would exceed your organization's rate limit."}}`
	claudeSpendCap    = `{"type": "error", "error": {"type": "rate_limit_error", "message": "You have reached your API usage limits.", "details": {"error_code": "enforced_spend_limit_reached"}}}`
	openAIRateLimit   = `{"error": {"message": "Rate limit reached for requests", "type": "requests", "code": "rate_limit_exceeded"}}`
	openAIQuota       = `{"error": {"message": "You exceeded your current quota, please check your plan and billing details.", "type": "insufficient_quota", "code": "insufficient_quota"}}`
)

// providerReply is one scripted HTTP reply.
type providerReply struct {
	status       int
	retryAfter   string
	retryAfterMs string
	shouldRetry  string
	body         string
	// hangUp drops the connection without a reply.
	hangUp bool
	// handle replaces the reply. It runs on the server's goroutine.
	handle func(r *http.Request)
}

// scriptedProvider serves the replies in order and repeats the last one.
type scriptedProvider struct {
	url      string
	requests atomic.Int32
}

func newScriptedProvider(t *testing.T, replies ...providerReply) *scriptedProvider {
	t.Helper()
	provider := &scriptedProvider{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reply := replies[min(int(provider.requests.Add(1)), len(replies))-1]
		switch {
		case reply.handle != nil:
			reply.handle(r)
		case reply.hangUp:
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
			_ = conn.Close()
		default:
			if reply.retryAfterMs != "" {
				w.Header().Set("Retry-After-Ms", reply.retryAfterMs)
			}
			if reply.shouldRetry != "" {
				w.Header().Set("X-Should-Retry", reply.shouldRetry)
			}
			if reply.retryAfter != "" {
				w.Header().Set("Retry-After", reply.retryAfter)
			}
			w.WriteHeader(reply.status)
			_, _ = w.Write([]byte(reply.body))
		}
	}))
	t.Cleanup(server.Close)
	provider.url = server.URL
	return provider
}

// sleepRecorder stands in for settingModel.sleep. It records each wait and
// returns at once, with the context's error when the context has ended.
type sleepRecorder struct {
	waits []time.Duration
}

func (s *sleepRecorder) sleep(ctx context.Context, wait time.Duration) error {
	s.waits = append(s.waits, wait)
	return ctx.Err()
}

func newRetryModel(provider storepb.AISetting_Provider, endpoint string) (*settingModel, *sleepRecorder) {
	if provider == storepb.AISetting_OPEN_AI {
		endpoint += "/v1/chat/completions"
	}
	recorder := &sleepRecorder{}
	return &settingModel{
		setting: &storepb.AISetting{Provider: provider, Endpoint: endpoint, Model: "test-model", ApiKey: testAPIKey},
		sleep:   recorder.sleep,
	}, recorder
}

func chatRequest() *v1pb.AIChatRequest {
	return &v1pb.AIChatRequest{Messages: []*v1pb.AIChatMessage{textMessage(roleUser, "hello")}}
}

// requireBackoff asserts the jittered waits of the retries, which double from
// a first wait around one second.
func requireBackoff(t *testing.T, waits []time.Duration, retries int) {
	t.Helper()
	require.Len(t, waits, retries)
	for retry, wait := range waits {
		backoff := time.Second << retry
		require.GreaterOrEqual(t, wait, backoff/2, "retry %d", retry)
		require.Less(t, wait, backoff*3/2, "retry %d", retry)
	}
}

func TestSettingModelWaitsAsLongAsTheProviderAsks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider storepb.AISetting_Provider
		failure  providerReply
		success  string
		want     time.Duration
	}{
		{
			name:     "Claude rate limit with Retry-After",
			provider: storepb.AISetting_CLAUDE,
			failure:  providerReply{status: http.StatusTooManyRequests, retryAfter: "7", body: claudeRateLimit},
			success:  claudeOK,
			want:     7 * time.Second,
		},
		{
			name:     "Claude rate limit with Retry-After-Ms, which wins over Retry-After",
			provider: storepb.AISetting_CLAUDE,
			failure:  providerReply{status: http.StatusTooManyRequests, retryAfterMs: "4500", retryAfter: "5", body: claudeRateLimit},
			success:  claudeOK,
			want:     4500 * time.Millisecond,
		},
		{
			name:     "OpenAI rate limit with Retry-After",
			provider: storepb.AISetting_OPEN_AI,
			failure:  providerReply{status: http.StatusTooManyRequests, retryAfter: "20", body: openAIRateLimit},
			success:  openAIOK,
			want:     20 * time.Second,
		},
		{
			name:     "Gemini rate limit with the delay in the body",
			provider: storepb.AISetting_GEMINI,
			failure:  providerReply{status: http.StatusTooManyRequests, body: geminiRateLimit},
			success:  geminiOK,
			want:     13 * time.Second,
		},
		{
			name:     "Gemini unavailable with Retry-After",
			provider: storepb.AISetting_GEMINI,
			failure:  providerReply{status: http.StatusServiceUnavailable, retryAfter: "2", body: geminiUnavailable},
			success:  geminiOK,
			want:     2 * time.Second,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			provider := newScriptedProvider(t, test.failure, providerReply{status: http.StatusOK, body: test.success})
			model, recorder := newRetryModel(test.provider, provider.url)
			response, err := model.Chat(context.Background(), chatRequest())
			require.NoError(t, err)
			require.Equal(t, "ok", response.GetContent())
			require.Equal(t, int32(2), provider.requests.Load())
			require.Equal(t, []time.Duration{test.want}, recorder.waits)
		})
	}
}

func TestSettingModelBacksOffWhenTheProviderNamesNoWait(t *testing.T) {
	t.Parallel()

	// The 429 here has neither a Retry-After nor the spend cap code, so nothing
	// says it will repeat and it is retried like the other statuses.
	for _, status := range []int{
		http.StatusRequestTimeout,
		http.StatusConflict,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusNotImplemented,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		529,
	} {
		t.Run(fmt.Sprintf("%d then a reply", status), func(t *testing.T) {
			t.Parallel()

			provider := newScriptedProvider(t,
				providerReply{status: status, body: claudeRateLimit},
				providerReply{status: http.StatusOK, body: claudeOK})
			model, recorder := newRetryModel(storepb.AISetting_CLAUDE, provider.url)
			response, err := model.Chat(context.Background(), chatRequest())
			require.NoError(t, err)
			require.Equal(t, "ok", response.GetContent())
			require.Equal(t, int32(2), provider.requests.Load())
			requireBackoff(t, recorder.waits, 1)
		})
	}

	t.Run("x-should-retry true on a 400", func(t *testing.T) {
		t.Parallel()

		provider := newScriptedProvider(t,
			providerReply{status: http.StatusBadRequest, shouldRetry: "true", body: `{"error": {"message": "try again", "type": "invalid_request_error"}}`},
			providerReply{status: http.StatusOK, body: openAIOK})
		model, recorder := newRetryModel(storepb.AISetting_OPEN_AI, provider.url)
		_, err := model.Chat(context.Background(), chatRequest())
		require.NoError(t, err)
		require.Equal(t, int32(2), provider.requests.Load())
		requireBackoff(t, recorder.waits, 1)
	})

	t.Run("a Retry-After of zero", func(t *testing.T) {
		t.Parallel()

		provider := newScriptedProvider(t,
			providerReply{status: http.StatusTooManyRequests, retryAfter: "0", body: claudeRateLimit},
			providerReply{status: http.StatusOK, body: claudeOK})
		model, recorder := newRetryModel(storepb.AISetting_CLAUDE, provider.url)
		_, err := model.Chat(context.Background(), chatRequest())
		require.NoError(t, err)
		require.Equal(t, int32(2), provider.requests.Load())
		requireBackoff(t, recorder.waits, 1)
	})

	t.Run("a quota code on a status other than 429 does not stop the retry", func(t *testing.T) {
		t.Parallel()

		provider := newScriptedProvider(t,
			providerReply{status: http.StatusInternalServerError, body: openAIQuota},
			providerReply{status: http.StatusOK, body: openAIOK})
		model, recorder := newRetryModel(storepb.AISetting_OPEN_AI, provider.url)
		_, err := model.Chat(context.Background(), chatRequest())
		require.NoError(t, err)
		require.Equal(t, int32(2), provider.requests.Load())
		requireBackoff(t, recorder.waits, 1)
	})

	t.Run("gives up after the fourth attempt", func(t *testing.T) {
		t.Parallel()

		provider := newScriptedProvider(t, providerReply{status: http.StatusServiceUnavailable, body: geminiUnavailable})
		model, recorder := newRetryModel(storepb.AISetting_GEMINI, provider.url)
		_, err := model.Chat(context.Background(), chatRequest())

		var httpErr *ai.HTTPError
		require.ErrorAs(t, err, &httpErr)
		require.Equal(t, http.StatusServiceUnavailable, httpErr.StatusCode)
		require.EqualError(t, err, "Gemini API returned status 503: "+geminiUnavailable)
		require.Equal(t, int32(4), provider.requests.Load())
		requireBackoff(t, recorder.waits, 3)
	})
}

func TestSettingModelDoesNotRetryAFailureThatWillRepeat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider storepb.AISetting_Provider
		reply    providerReply
		wantText string
	}{
		{
			name:     "bad request",
			provider: storepb.AISetting_GEMINI,
			reply:    providerReply{status: http.StatusBadRequest, body: `{"error": {"code": 400, "status": "INVALID_ARGUMENT"}}`},
			wantText: "Gemini API returned status 400",
		},
		{
			name:     "bad key",
			provider: storepb.AISetting_OPEN_AI,
			reply:    providerReply{status: http.StatusUnauthorized, body: `{"error": {"message": "Incorrect API key provided", "code": "invalid_api_key"}}`},
			wantText: "OpenAI API returned status 401",
		},
		{
			name:     "forbidden",
			provider: storepb.AISetting_CLAUDE,
			reply:    providerReply{status: http.StatusForbidden, body: `{"type": "error", "error": {"type": "permission_error"}}`},
			wantText: "Claude API returned status 403",
		},
		{
			name:     "not found",
			provider: storepb.AISetting_GEMINI,
			reply:    providerReply{status: http.StatusNotFound, body: `{"error": {"code": 404, "status": "NOT_FOUND"}}`},
			wantText: "Gemini API returned status 404",
		},
		{
			name:     "a status outside the retry list, even with Retry-After",
			provider: storepb.AISetting_CLAUDE,
			reply:    providerReply{status: http.StatusUnprocessableEntity, retryAfter: "1", body: `{}`},
			wantText: "Claude API returned status 422",
		},
		{
			name:     "x-should-retry false on a 503",
			provider: storepb.AISetting_CLAUDE,
			reply:    providerReply{status: http.StatusServiceUnavailable, shouldRetry: "false", body: `{"type": "error", "error": {"type": "api_error"}}`},
			wantText: "Claude API returned status 503",
		},
		{
			name:     "OpenAI used-up quota",
			provider: storepb.AISetting_OPEN_AI,
			reply:    providerReply{status: http.StatusTooManyRequests, body: openAIQuota},
			wantText: "OpenAI API returned status 429",
		},
		{
			name:     "OpenAI used-up quota, even with x-should-retry true",
			provider: storepb.AISetting_OPEN_AI,
			reply:    providerReply{status: http.StatusTooManyRequests, shouldRetry: "true", body: openAIQuota},
			wantText: "OpenAI API returned status 429",
		},
		{
			name:     "Claude spend cap",
			provider: storepb.AISetting_CLAUDE,
			reply:    providerReply{status: http.StatusTooManyRequests, body: claudeSpendCap},
			wantText: "Claude API returned status 429",
		},
		{
			name:     "a reply that cannot be decoded",
			provider: storepb.AISetting_GEMINI,
			reply:    providerReply{status: http.StatusOK, body: `not JSON`},
			wantText: "failed to unmarshal Gemini response",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			provider := newScriptedProvider(t, test.reply)
			model, recorder := newRetryModel(test.provider, provider.url)
			_, err := model.Chat(context.Background(), chatRequest())
			require.ErrorContains(t, err, test.wantText)
			require.Equal(t, int32(1), provider.requests.Load())
			require.Empty(t, recorder.waits)
		})
	}
}

func TestSettingModelRetriesATransportFailure(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name     string
		provider storepb.AISetting_Provider
		success  string
	}{
		{name: "OpenAI", provider: storepb.AISetting_OPEN_AI, success: openAIOK},
		{name: "Gemini", provider: storepb.AISetting_GEMINI, success: geminiOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			provider := newScriptedProvider(t, providerReply{hangUp: true}, providerReply{status: http.StatusOK, body: test.success})
			model, recorder := newRetryModel(test.provider, provider.url)
			response, err := model.Chat(context.Background(), chatRequest())
			require.NoError(t, err)
			require.Equal(t, "ok", response.GetContent())
			require.Equal(t, int32(2), provider.requests.Load())
			requireBackoff(t, recorder.waits, 1)
		})
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

// replaceDefaultTransport routes ai.Chat, which sends through
// http.DefaultTransport, to roundTrip. A test that calls it must not be
// parallel. It returns the transport it replaced.
func replaceDefaultTransport(t *testing.T, roundTrip roundTripperFunc) http.RoundTripper {
	t.Helper()
	defaultTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = defaultTransport })
	http.DefaultTransport = roundTrip
	return defaultTransport
}

func TestSettingModelRetriesADialTimeout(t *testing.T) {
	provider := newScriptedProvider(t, providerReply{status: http.StatusOK, body: geminiOK})
	model, recorder := newRetryModel(storepb.AISetting_GEMINI, provider.url)

	// A dial that is given no time fails at once with the error of a real dial timeout.
	dialTimeout := &http.Transport{DialContext: (&net.Dialer{Timeout: time.Nanosecond}).DialContext}
	roundTrips := 0
	var defaultTransport http.RoundTripper
	defaultTransport = replaceDefaultTransport(t, func(request *http.Request) (*http.Response, error) {
		roundTrips++
		if roundTrips > 1 {
			return defaultTransport.RoundTrip(request)
		}
		response, err := dialTimeout.RoundTrip(request)
		if err == nil {
			_ = response.Body.Close()
		}
		// The error matches the context's sentinel while the context is alive.
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.NoError(t, request.Context().Err())
		return nil, err
	})

	response, err := model.Chat(context.Background(), chatRequest())
	require.NoError(t, err)
	require.Equal(t, "ok", response.GetContent())
	require.Equal(t, 2, roundTrips)
	require.Equal(t, int32(1), provider.requests.Load())
	requireBackoff(t, recorder.waits, 1)
}

func TestSettingModelReportsACanceledContextOverTheReply(t *testing.T) {
	for _, status := range []int{http.StatusServiceUnavailable, http.StatusUnauthorized} {
		t.Run(fmt.Sprintf("%d", status), func(t *testing.T) {
			model, recorder := newRetryModel(storepb.AISetting_GEMINI, "http://provider.test")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			roundTrips := 0
			replaceDefaultTransport(t, func(*http.Request) (*http.Response, error) {
				roundTrips++
				cancel()
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(geminiUnavailable))}, nil
			})

			_, err := model.Chat(ctx, chatRequest())
			require.ErrorIs(t, err, context.Canceled)
			require.Equal(t, 1, roundTrips)
			require.Empty(t, recorder.waits)
		})
	}
}

func TestSettingModelStopsWhenTheContextIsCanceled(t *testing.T) {
	t.Parallel()

	t.Run("during the wait", func(t *testing.T) {
		t.Parallel()

		provider := newScriptedProvider(t, providerReply{status: http.StatusTooManyRequests, retryAfter: "3600", body: claudeRateLimit})
		model, _ := newRetryModel(storepb.AISetting_CLAUDE, provider.url)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		var waits []time.Duration
		model.sleep = func(waitCtx context.Context, wait time.Duration) error {
			waits = append(waits, wait)
			cancel()
			return sleepContext(waitCtx, wait)
		}

		_, err := model.Chat(ctx, chatRequest())
		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, []time.Duration{time.Hour}, waits)
		require.Equal(t, int32(1), provider.requests.Load())
	})

	t.Run("during the call", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		provider := newScriptedProvider(t, providerReply{handle: func(r *http.Request) {
			// The server notices that the client left only after the body is read.
			_, _ = io.Copy(io.Discard, r.Body)
			cancel()
			// Holding the reply until the client is gone keeps the test from
			// racing the cancellation against a reply.
			<-r.Context().Done()
		}})
		model, recorder := newRetryModel(storepb.AISetting_GEMINI, provider.url)

		_, err := model.Chat(ctx, chatRequest())
		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, int32(1), provider.requests.Load())
		require.Empty(t, recorder.waits)
	})

	t.Run("before the call", func(t *testing.T) {
		t.Parallel()

		provider := newScriptedProvider(t, providerReply{status: http.StatusOK, body: geminiOK})
		model, recorder := newRetryModel(storepb.AISetting_GEMINI, provider.url)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := model.Chat(ctx, chatRequest())
		require.ErrorIs(t, err, context.Canceled)
		require.Zero(t, provider.requests.Load())
		require.Empty(t, recorder.waits)
	})
}

func TestSettingModelDoesNotWaitPastTheDeadline(t *testing.T) {
	t.Parallel()

	t.Run("a wait that ends after the deadline returns the provider's error at once", func(t *testing.T) {
		t.Parallel()

		provider := newScriptedProvider(t, providerReply{status: http.StatusTooManyRequests, retryAfter: "3600", body: claudeRateLimit})
		model, recorder := newRetryModel(storepb.AISetting_CLAUDE, provider.url)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		_, err := model.Chat(ctx, chatRequest())
		var httpErr *ai.HTTPError
		require.ErrorAs(t, err, &httpErr)
		require.Equal(t, http.StatusTooManyRequests, httpErr.StatusCode)
		require.NotErrorIs(t, err, context.DeadlineExceeded)
		require.NoError(t, ctx.Err())
		require.Equal(t, int32(1), provider.requests.Load())
		require.Empty(t, recorder.waits)
	})

	t.Run("a wait that ends before the deadline is taken", func(t *testing.T) {
		t.Parallel()

		provider := newScriptedProvider(t,
			providerReply{status: http.StatusTooManyRequests, retryAfter: "60", body: claudeRateLimit},
			providerReply{status: http.StatusOK, body: claudeOK})
		model, recorder := newRetryModel(storepb.AISetting_CLAUDE, provider.url)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		_, err := model.Chat(ctx, chatRequest())
		require.NoError(t, err)
		require.Equal(t, int32(2), provider.requests.Load())
		require.Equal(t, []time.Duration{time.Minute}, recorder.waits)
	})
}

func TestReviewCountsARetriedModelCallOnce(t *testing.T) {
	t.Parallel()

	emptyFindings := `{"candidates": [{"content": {"parts": [{"text": "{\"findings\": []}"}]}}], "usageMetadata": {"totalTokenCount": 60}}`

	t.Run("a retry that succeeds", func(t *testing.T) {
		t.Parallel()

		provider := newScriptedProvider(t,
			providerReply{status: http.StatusServiceUnavailable, body: geminiUnavailable},
			providerReply{status: http.StatusOK, body: emptyFindings})
		model, recorder := newRetryModel(storepb.AISetting_GEMINI, provider.url)
		result, err := NewReviewer(model).Review(context.Background(), &Request{Statement: threeLineStatement}, ordersCatalog())
		require.NoError(t, err)
		require.Equal(t, 1, result.Calls)
		require.Equal(t, 60, result.TotalTokens)
		require.Equal(t, int32(2), provider.requests.Load())
		requireBackoff(t, recorder.waits, 1)
	})

	t.Run("retries that all fail", func(t *testing.T) {
		t.Parallel()

		provider := newScriptedProvider(t, providerReply{status: http.StatusServiceUnavailable, body: geminiUnavailable})
		model, recorder := newRetryModel(storepb.AISetting_GEMINI, provider.url)
		_, err := NewReviewer(model).Review(context.Background(), &Request{Statement: threeLineStatement}, ordersCatalog())
		require.ErrorContains(t, err, "model call 1 failed: Gemini API returned status 503")
		require.Equal(t, int32(4), provider.requests.Load())
		requireBackoff(t, recorder.waits, 3)
	})
}

// lockedBuffer lets the test read the log while the handler may still write.
type lockedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

// Not parallel: it replaces the default logger.
func TestSettingModelLogsEachRetryWithoutTheBodyOrTheKey(t *testing.T) {
	defaultLogger := slog.Default()
	t.Cleanup(func() { slog.SetDefault(defaultLogger) })
	var log lockedBuffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&log, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return attr
		},
	})))

	provider := newScriptedProvider(t,
		providerReply{status: http.StatusTooManyRequests, retryAfter: "7", body: claudeRateLimit},
		providerReply{hangUp: true},
		providerReply{status: http.StatusOK, body: claudeOK})
	model, _ := newRetryModel(storepb.AISetting_CLAUDE, provider.url)
	_, err := model.Chat(context.Background(), chatRequest())
	require.NoError(t, err)

	lines := bytes.Split(bytes.TrimSpace([]byte(log.String())), []byte("\n"))
	require.Len(t, lines, 2)
	require.Equal(t, `level=WARN msg="AI review model call failed, retrying" provider=CLAUDE status=429 attempt=1 wait=7s`, string(lines[0]))
	require.Regexp(t, `^level=WARN msg="AI review model call failed, retrying" provider=CLAUDE status=0 attempt=2 wait=[0-9.]+m?s$`, string(lines[1]))
	require.NotContains(t, log.String(), testAPIKey)
	require.NotContains(t, log.String(), "rate_limit_error")
}

func TestSleepContext(t *testing.T) {
	t.Parallel()

	t.Run("waits for the whole duration", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			start := time.Now()
			require.NoError(t, sleepContext(context.Background(), time.Hour))
			require.Equal(t, time.Hour, time.Since(start))
		})
	})

	t.Run("stops as soon as the context ends", func(t *testing.T) {
		t.Parallel()

		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			time.AfterFunc(time.Minute, cancel)
			start := time.Now()
			require.ErrorIs(t, sleepContext(ctx, time.Hour), context.Canceled)
			require.Equal(t, time.Minute, time.Since(start))
		})
	})
}

func TestSettingModelChecksBeforeEveryAttempt(t *testing.T) {
	t.Parallel()

	refusal := errors.New("AI SQL review is not enabled in the workspace AI setting")

	t.Run("a refusal before the first call", func(t *testing.T) {
		t.Parallel()
		provider := newScriptedProvider(t, providerReply{status: http.StatusOK, body: geminiOK})
		model := NewModel(
			&storepb.AISetting{Provider: storepb.AISetting_GEMINI, Endpoint: provider.url, Model: "test-model", ApiKey: testAPIKey},
			CheckBeforeEachCall(func(context.Context) error { return refusal }),
		)
		_, err := model.Chat(context.Background(), chatRequest())
		require.ErrorIs(t, err, refusal)
		require.Zero(t, provider.requests.Load())
	})

	t.Run("a refusal while a failed call waits to retry", func(t *testing.T) {
		t.Parallel()
		provider := newScriptedProvider(t,
			providerReply{status: http.StatusServiceUnavailable, body: geminiUnavailable},
			providerReply{status: http.StatusOK, body: geminiOK},
		)
		model, recorder := newRetryModel(storepb.AISetting_GEMINI, provider.url)
		var refused atomic.Bool
		CheckBeforeEachCall(func(context.Context) error {
			if refused.Load() {
				return refusal
			}
			return nil
		})(model)
		model.sleep = func(ctx context.Context, wait time.Duration) error {
			refused.Store(true)
			return recorder.sleep(ctx, wait)
		}

		_, err := model.Chat(context.Background(), chatRequest())
		require.ErrorIs(t, err, refusal)
		require.Equal(t, int32(1), provider.requests.Load(), "no retry reaches the provider after the check refuses")
	})
}
