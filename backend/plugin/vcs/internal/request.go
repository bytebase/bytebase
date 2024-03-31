// Package internal provides library for VCS plugins.
package internal

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

// Post makes a HTTP POST request to the given URL.
func Post(ctx context.Context, url string, authorization string, body []byte) (code int, respBody string, err error) {
	return request(ctx, http.MethodPost, url, authorization, nil, bytes.NewReader(body))
}

// Get makes a HTTP GET request to the given URL.
func Get(ctx context.Context, url string, authorization string) (code int, respBody string, err error) {
	return request(ctx, http.MethodGet, url, authorization, nil, bytes.NewReader(nil))
}

// GetWithHeader makes a HTTP GET request to the given URL wit additional header.
func GetWithHeader(ctx context.Context, url string, authorization string, header map[string]string) (code int, respBody string, err error) {
	return request(ctx, http.MethodGet, url, authorization, header, bytes.NewReader(nil))
}

// Delete makes a HTTP DELETE request to the given URL.
func Delete(ctx context.Context, url string, authorization string) (code int, respBody string, err error) {
	return request(ctx, http.MethodDelete, url, authorization, nil, bytes.NewReader(nil))
}

func request(ctx context.Context, method, url string, authorization string, header map[string]string, requestBody *bytes.Reader) (int, string, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, requestBody)
	if err != nil {
		return 0, "", errors.Wrapf(err, "failed to build delete request for url %s", url)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Authorization", authorization)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", errors.Wrapf(err, "failed to send delete request for url %s", url)
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", errors.Wrapf(err, "read delete %s response body with status code %d", url, resp.StatusCode)
	}
	if err := resp.Body.Close(); err != nil {
		return 0, "", errors.Wrapf(err, "failed to close delete %s response body with status code %d", url, resp.StatusCode)
	}
	return resp.StatusCode, string(respBody), nil
}
