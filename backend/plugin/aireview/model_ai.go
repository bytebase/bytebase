package aireview

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/ai"
)

const (
	// maxChatRetries is how many times one model call is sent again.
	maxChatRetries = 3
	// firstRetryBackoff is the middle of the first backoff. Each retry doubles it.
	firstRetryBackoff = time.Second
	// maxOutputTokens holds Claude's thinking, which counts against the limit,
	// and a long findings list. claude-sonnet-5 allows 128K, but Anthropic
	// advises streaming above about 16K and ai.Chat does not stream. The Claude
	// adapter lowers it for an older model whose maximum is smaller.
	maxOutputTokens = 16000
)

type settingModel struct {
	setting *storepb.AISetting
	// sleep waits for the duration, or returns the context's error as soon as
	// the context ends. Tests replace it so that a retry takes no time.
	sleep func(ctx context.Context, wait time.Duration) error
}

// NewModel returns the Model behind the workspace AI setting.
func NewModel(setting *storepb.AISetting) Model {
	return &settingModel{setting: setting, sleep: sleepContext}
}

// Chat sends the request again after a transient provider failure, so the
// review loop sees one model call however many attempts it takes.
func (m *settingModel) Chat(ctx context.Context, request *v1pb.AIChatRequest) (*v1pb.AIChatResponse, error) {
	for retry := 0; ; retry++ {
		response, err := ai.Chat(ctx, m.setting, request, ai.WithMaxOutputTokens(maxOutputTokens))
		if err == nil {
			return response, nil
		}
		// A context that ended is reported as such, whatever the provider
		// replied. The context is asked, not the error: a dial timeout also
		// matches context.DeadlineExceeded while the context is alive.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		if retry == maxChatRetries || !isTransientFailure(err) {
			return nil, err
		}
		wait := retryWait(err, retry)
		// A sleep that ends after the deadline would trade the provider's
		// error for a timeout.
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= wait {
			return nil, err
		}
		slog.Warn("AI review model call failed, retrying",
			slog.String("provider", m.setting.GetProvider().String()),
			slog.Int("status", httpStatus(err)),
			slog.Int("attempt", retry+1),
			slog.Duration("wait", wait))
		if err := m.sleep(ctx, wait); err != nil {
			return nil, err
		}
	}
}

// isTransientFailure reports whether the failed call should be sent again.
func isTransientFailure(err error) bool {
	var httpErr *ai.HTTPError
	if errors.As(err, &httpErr) {
		if httpErr.StatusCode == http.StatusTooManyRequests && httpErr.QuotaExhausted {
			return false
		}
		// The rest follows the official OpenAI and Anthropic Go SDKs.
		if httpErr.ShouldRetry != nil {
			return *httpErr.ShouldRetry
		}
		switch httpErr.StatusCode {
		case http.StatusRequestTimeout, http.StatusConflict, http.StatusTooManyRequests:
			return true
		default:
			return httpErr.StatusCode >= http.StatusInternalServerError
		}
	}
	var transportErr *ai.TransportError
	return errors.As(err, &transportErr)
}

// retryWait is the wait the provider asked for, or else a backoff. A wait of
// zero gets the backoff too, so a failed call is never repeated at once. The
// jitter keeps the concurrent reviews of one run from retrying at the same moment.
func retryWait(err error, retry int) time.Duration {
	var httpErr *ai.HTTPError
	if errors.As(err, &httpErr) && httpErr.RetryAfter > 0 {
		return httpErr.RetryAfter
	}
	backoff := firstRetryBackoff << retry
	return backoff/2 + rand.N(backoff)
}

// httpStatus is zero for a call that got no reply.
func httpStatus(err error) int {
	var httpErr *ai.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode
	}
	return 0
}

func sleepContext(ctx context.Context, wait time.Duration) error {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
