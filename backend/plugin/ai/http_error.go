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

// providerWait reads the wait a provider asked for in its reply headers.
// Retry-After-Ms, in milliseconds, wins over Retry-After, as it does in the
// official OpenAI and Anthropic Go SDKs.
func providerWait(header http.Header, now time.Time) time.Duration {
	if wait, ok := parseDecimal(strings.TrimSpace(header.Get("Retry-After-Ms")), time.Millisecond); ok {
		return wait
	}
	return parseRetryAfter(header.Get("Retry-After"), now)
}

// parseRetryAfter reads a Retry-After header value, which is a number of
// seconds or an HTTP date. It returns zero for a value that is absent,
// malformed, or not in the future.
func parseRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if wait, ok := parseDecimal(value, time.Second); ok {
		return wait
	}
	if date, err := http.ParseTime(value); err == nil {
		return max(date.Sub(now), 0)
	}
	return 0
}

var decimalNumber = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

// parseDecimal reads a count of units written as digits with an optional
// fraction, to the nanosecond. The unit is a power of ten nanoseconds, such as
// time.Second or time.Millisecond. It saturates instead of overflowing: a wait
// too long for a Duration must not come out short or negative.
func parseDecimal(value string, unit time.Duration) (time.Duration, bool) {
	if !decimalNumber.MatchString(value) {
		return 0, false
	}
	whole, fraction, _ := strings.Cut(value, ".")
	count, err := strconv.ParseInt(whole, 10, 64)
	if err != nil || count > math.MaxInt64/int64(unit) {
		return math.MaxInt64, true
	}
	// Padded or cut to as many digits as the unit has zeros, the fraction
	// counts nanoseconds.
	digits := len(strconv.FormatInt(int64(unit), 10)) - 1
	nanoseconds, _ := strconv.ParseInt((fraction + strings.Repeat("0", digits))[:digits], 10, 64)
	wait := time.Duration(count) * unit
	if wait > math.MaxInt64-time.Duration(nanoseconds) {
		return math.MaxInt64, true
	}
	return wait + time.Duration(nanoseconds), true
}
