// Package internal provides library for VCS plugins.
package internal

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
)

// Post makes a HTTP POST request to the given URL using the token. It refreshes
// token and retries the request in the case of the token has expired.
func Post(ctx context.Context, client *http.Client, url string, token string, body []byte) (code int, header http.Header, respBody string, err error) {
	//nolint:bodyclose
	return retry(ctx, requesterWithHeader(ctx, client, http.MethodPost, url, token, bytes.NewReader(body), nil))
}

// Get makes a HTTP GET request to the given URL using the token. It refreshes
// token and retries the request in the case of the token has expired.
func Get(ctx context.Context, client *http.Client, url string, token string) (code int, header http.Header, respBody string, err error) {
	//nolint:bodyclose
	return retry(ctx, requesterWithHeader(ctx, client, http.MethodGet, url, token, nil, nil))
}

// GetWithHeader makes a HTTP GET request to the given URL using the token and
// additional header. It refreshes token and retries the request in the case of
// the token has expired.
func GetWithHeader(ctx context.Context, client *http.Client, url string, token string, header map[string]string) (code int, _ http.Header, respBody string, err error) {
	//nolint:bodyclose
	return retry(ctx, requesterWithHeader(ctx, client, http.MethodGet, url, token, nil, header))
}

// Delete makes a HTTP DELETE request to the given URL using the token. It refreshes
// token and retries the request in the case of the token has expired.
func Delete(ctx context.Context, client *http.Client, url string, token string) (code int, header http.Header, respBody string, err error) {
	//nolint:bodyclose
	return retry(ctx, requesterWithHeader(ctx, client, http.MethodDelete, url, token, nil, nil))
}

func requesterWithHeader(ctx context.Context, client *http.Client, method, url string, token string, body io.Reader, header map[string]string) func() (*http.Response, error) {
	// The body may be read multiple times but io.Reader is meant to be read once,
	// so we read the body first and build the reader every time.
	var bodyBytes []byte
	if body != nil {
		b, err := io.ReadAll(body)
		if err != nil {
			return func() (*http.Response, error) {
				return nil, errors.Wrap(err, "failed to read from body")
			}
		}
		bodyBytes = b
	}
	return func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, errors.Wrapf(err, "construct %s %s", method, url)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
		for k, v := range header {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, errors.Wrapf(err, "%s %s", method, url)
		}
		return resp, nil
	}
}

func retry(ctx context.Context, f func() (*http.Response, error)) (code int, header http.Header, respBody string, err error) {
	// TODO(d): handle timeout retries.
	select {
	case <-ctx.Done():
		return 0, nil, "", ctx.Err()
	default:
	}

	resp, err := f()
	if err != nil {
		return 0, nil, "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, "", errors.Wrapf(err, "read response body with status code %d", resp.StatusCode)
	}
	if err := resp.Body.Close(); err != nil {
		slog.Warn("failed to close resp body", log.BBError(err))
	}
	return resp.StatusCode, resp.Header, string(body), nil
}
