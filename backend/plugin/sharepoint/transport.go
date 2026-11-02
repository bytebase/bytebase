package sharepoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

const (
	maxResponseBytes = 4 << 20
	maxRetryDelay    = 2 * time.Minute
	maxBackoff       = 30 * time.Second
)

type response struct {
	status int
	header http.Header
	body   []byte
}

// unknownOutcomeError means a request may or may not have taken effect.
type unknownOutcomeError struct {
	cause error
}

func (e *unknownOutcomeError) Error() string {
	return "could not reach Microsoft Graph: " + e.cause.Error()
}

func (e *unknownOutcomeError) Unwrap() error {
	return e.cause
}

type graphError struct {
	status  int
	code    string
	message string
}

func newGraphError(resp *response) *graphError {
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(resp.body, &body)
	return &graphError{status: resp.status, code: body.Error.Code, message: body.Error.Message}
}

func (e *graphError) sentinel() error {
	switch e.status {
	case http.StatusUnauthorized:
		return ErrCredential
	case http.StatusForbidden, http.StatusNotFound:
		return ErrNotFoundOrNoAccess
	default:
		return nil
	}
}

func (e *graphError) Error() string {
	detail := fmt.Sprintf("Microsoft Graph returned HTTP %d", e.status)
	if e.code != "" {
		detail += " " + e.code
	}
	if e.message != "" {
		detail += ": " + e.message
	}
	if sentinel := e.sentinel(); sentinel != nil {
		return sentinel.Error() + " (" + detail + ")"
	}
	return detail
}

func (e *graphError) Unwrap() error {
	return e.sentinel()
}

func statusOf(err error) int {
	var graphErr *graphError
	if errors.As(err, &graphErr) {
		return graphErr.status
	}
	return 0
}

// graphJSON calls a Graph API path with an optional JSON body and decodes a
// successful JSON response into out.
func (c *graphClient) graphJSON(ctx context.Context, method, path string, in, out any) error {
	var body []byte
	header := http.Header{}
	if in != nil {
		data, err := json.Marshal(in)
		if err != nil {
			return errors.Wrap(err, "failed to encode Microsoft Graph request")
		}
		body = data
		header.Set("Content-Type", "application/json")
	}
	resp, err := c.send(ctx, method, c.baseURL+path, body, header, true)
	if err != nil {
		return err
	}
	if resp.status < 200 || resp.status >= 300 {
		return newGraphError(resp)
	}
	if out != nil {
		if err := json.Unmarshal(resp.body, out); err != nil {
			return errors.Wrap(err, "failed to parse Microsoft Graph response")
		}
	}
	return nil
}

// graphRaw calls a Graph API path with a binary body and ignores a successful
// response's body.
func (c *graphClient) graphRaw(ctx context.Context, method, path string, body []byte) error {
	header := http.Header{}
	header.Set("Content-Type", "application/octet-stream")
	resp, err := c.send(ctx, method, c.baseURL+path, body, header, true)
	if err != nil {
		return err
	}
	if resp.status < 200 || resp.status >= 300 {
		return newGraphError(resp)
	}
	return nil
}

// send sends a request, retrying throttling, unavailability, and transport
// errors. If any attempt's outcome is unknown and the retries do not produce a
// response, the error is an *unknownOutcomeError.
func (c *graphClient) send(ctx context.Context, method, target string, body []byte, header http.Header, authorize bool) (*response, error) {
	outcomeUnknown := false
	for attempt := 1; ; attempt++ {
		resp, err := c.sendOnce(ctx, method, target, body, header, authorize)
		var lastErr error
		switch {
		case err != nil:
			var unknown *unknownOutcomeError
			if !errors.As(err, &unknown) {
				return nil, err
			}
			outcomeUnknown = true
			lastErr = unknown.cause
			if ctx.Err() != nil {
				return nil, err
			}
		case resp.status == http.StatusTooManyRequests:
			lastErr = newGraphError(resp)
		case resp.status == http.StatusServiceUnavailable || resp.status == http.StatusGatewayTimeout:
			outcomeUnknown = true
			lastErr = newGraphError(resp)
		default:
			return resp, nil
		}
		if attempt >= c.maxAttempts {
			if outcomeUnknown {
				return nil, &unknownOutcomeError{cause: lastErr}
			}
			return nil, lastErr
		}
		if err := c.wait(ctx, retryDelay(resp, attempt)); err != nil {
			return nil, &unknownOutcomeError{cause: err}
		}
	}
}

func (c *graphClient) sendOnce(ctx context.Context, method, target string, body []byte, header http.Header, authorize bool) (*response, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		// The error would include the URL, which may be a pre-authenticated upload URL.
		return nil, errors.New("invalid Microsoft Graph request URL")
	}
	for key, values := range header {
		req.Header[key] = values
	}
	req.Header.Set("Accept", "application/json")
	if authorize {
		token, err := c.token(ctx)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &unknownOutcomeError{cause: transportError(err)}
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, &unknownOutcomeError{cause: transportError(err)}
	}
	return &response{status: resp.StatusCode, header: resp.Header, body: data}, nil
}

// transportError drops the request URL from err.
func transportError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return errors.Errorf("%s request failed: %v", urlErr.Op, urlErr.Err)
	}
	return err
}

func retryDelay(resp *response, attempt int) time.Duration {
	if resp != nil {
		if seconds, err := strconv.Atoi(resp.header.Get("Retry-After")); err == nil && seconds >= 0 {
			return min(time.Duration(seconds)*time.Second, maxRetryDelay)
		}
	}
	return min(time.Second<<(attempt-1), maxBackoff)
}
