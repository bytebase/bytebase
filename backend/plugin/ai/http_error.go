package ai

import (
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// HTTPError is a provider reply with a status other than 200.
type HTTPError struct {
	// Provider is the adapter's name for the vendor: "OpenAI", "Claude", or "Gemini".
	Provider   string
	StatusCode int
	// RetryAfter is how long the provider asked the caller to wait before the
	// next call. It is zero when the provider named no wait, or a wait of zero.
	RetryAfter time.Duration
	// QuotaExhausted means the provider refused the call for a quota that a
	// wait does not restore, such as a spend limit. The same call keeps failing.
	QuotaExhausted bool
	Body           string
}

// Error keeps a text that callers read: the frontend agent loop looks for the
// status code in it to decide whether to call again.
func (e *HTTPError) Error() string {
	return fmt.Sprintf("%s API returned status %d: %s", e.Provider, e.StatusCode, e.Body)
}

// TransportError is a provider call that got no complete reply: the request
// was not sent, or the reply could not be read. It does not unwrap to its
// cause: connect codes an uncoded error by a context error in its chain, and a
// dial timeout to the provider would then read as the caller's own deadline.
type TransportError struct {
	message string
}

func (e *TransportError) Error() string {
	return e.message
}

// parseRetryAfter reads a Retry-After header value, which is a number of
// seconds or an HTTP date. It returns zero for a value that is absent,
// malformed, or not in the future.
func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if wait, ok := parseDecimalSeconds(value); ok {
		return wait
	}
	if date, err := http.ParseTime(value); err == nil {
		return max(date.Sub(now), 0)
	}
	return 0
}

var decimalSeconds = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

// parseDecimalSeconds reads a count of seconds written as digits with an
// optional fraction, to the nanosecond. It saturates instead of overflowing: a
// wait too long for a Duration must not come out short or negative.
func parseDecimalSeconds(value string) (time.Duration, bool) {
	if !decimalSeconds.MatchString(value) {
		return 0, false
	}
	whole, fraction, _ := strings.Cut(value, ".")
	seconds, err := strconv.ParseInt(whole, 10, 64)
	if err != nil || seconds > math.MaxInt64/int64(time.Second) {
		return math.MaxInt64, true
	}
	// Nine digits always parse.
	nanoseconds, _ := strconv.Atoi((fraction + "000000000")[:9])
	wait := time.Duration(seconds) * time.Second
	if wait > math.MaxInt64-time.Duration(nanoseconds) {
		return math.MaxInt64, true
	}
	return wait + time.Duration(nanoseconds), true
}
