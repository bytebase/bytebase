package command

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/action/world"
)

func TestNewClientWithOptionsUsesServiceAccountAuth(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{}
	client, err := newClient("https://example.com", "", "sa@example.com", "secret", clientOptions{
		httpClient: httpClient,
		pageSize:   -1,
	})
	require.NoError(t, err)

	t.Cleanup(client.close)

	require.Same(t, httpClient, client.httpClient)
	require.Equal(t, int32(100), client.options.pageSize)
	require.Equal(t, "sa@example.com", client.serviceAccount)
	require.Equal(t, "secret", client.serviceAccountSecret)
}

func TestCustomHeaderFlag(t *testing.T) {
	t.Parallel()

	headers := http.Header{}
	var headerErr error
	flag := newCustomHeaderFlag(&headers, &headerErr)

	require.NoError(t, flag.Set("Cookie: CF_Authorization=token"))
	require.NoError(t, flag.Set("X-Request-Id: run-123"))
	require.NoError(t, headerErr)
	require.Equal(t, "CF_Authorization=token", headers.Get("Cookie"))
	require.Equal(t, "run-123", headers.Get("X-Request-Id"))
}

func TestCustomHeaderFlagRejectsInvalidHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "missing separator",
			value: "Cookie",
		},
		{
			name:  "empty name",
			value: ": value",
		},
		{
			name:  "newline in value",
			value: "Cookie: value\nX-Other: injected",
		},
		{
			name:  "authorization is managed by bytebase action",
			value: "Authorization: Bearer token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			headers := http.Header{}
			var headerErr error
			require.NoError(t, newCustomHeaderFlag(&headers, &headerErr).Set(tt.value))
			require.Error(t, headerErr)
		})
	}
}

func TestCustomHeaderFlagDoesNotLeakHeaderValueInFormatError(t *testing.T) {
	t.Parallel()

	headers := http.Header{}
	var headerErr error
	secret := "Cookie CF_Authorization=secret-token"
	require.NoError(t, newCustomHeaderFlag(&headers, &headerErr).Set(secret))
	require.Error(t, headerErr)
	require.NotContains(t, headerErr.Error(), "secret-token")
	require.NotContains(t, headerErr.Error(), secret)
}

func TestCustomHeadersAreSentOnLoginRequest(t *testing.T) {
	t.Parallel()

	customHeaders := http.Header{}
	customHeaders.Set("Cookie", "CF_Authorization=token")
	customHeaders.Set("X-Request-Id", "run-123")

	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "CF_Authorization=token", req.Header.Get("Cookie"))
		require.Equal(t, "run-123", req.Header.Get("X-Request-Id"))
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("unauthorized")),
			Request:    req,
		}, nil
	})

	httpClient := &http.Client{Transport: transport}
	client, err := newClient("https://example.com", "", "sa@example.com", "secret", clientOptions{
		httpClient:    httpClient,
		customHeaders: customHeaders,
	})
	require.NoError(t, err)
	t.Cleanup(client.close)
	refreshToken := getTokenRefresher(client.httpClient, "sa@example.com", "secret", "https://example.com")

	_, err = refreshToken(t.Context())
	require.Error(t, err)
}

func TestNewClientDoesNotMutateProvidedHTTPClientTransport(t *testing.T) {
	t.Parallel()

	customHeaders := http.Header{}
	customHeaders.Set("Cookie", "CF_Authorization=token")
	transport := &closeIdleRoundTripper{}
	httpClient := &http.Client{Transport: transport}

	client, err := newClient("https://example.com", "token", "", "", clientOptions{
		httpClient:    httpClient,
		customHeaders: customHeaders,
	})
	require.NoError(t, err)
	t.Cleanup(client.close)

	require.Same(t, transport, httpClient.Transport)
	require.NotSame(t, httpClient, client.httpClient)
}

func TestCustomHeaderTransportAddsCustomHeaders(t *testing.T) {
	t.Parallel()

	customHeaders := http.Header{}
	customHeaders.Set("Cookie", "CF_Authorization=token")
	customHeaders.Set("X-Request-Id", "run-123")

	transport := newCustomHeaderTransport(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "CF_Authorization=token", req.Header.Get("Cookie"))
		require.Equal(t, "run-123", req.Header.Get("X-Request-Id"))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("ok")),
			Request:    req,
		}, nil
	}), customHeaders)

	resp, err := transport.RoundTrip(httptest.NewRequest(http.MethodGet, "https://example.com", nil))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.NoError(t, resp.Body.Close())
}

func TestCustomHeaderTransportClosesIdleConnections(t *testing.T) {
	t.Parallel()

	base := &closeIdleRoundTripper{}
	transport := newCustomHeaderTransport(base, http.Header{})
	transport.(interface{ CloseIdleConnections() }).CloseIdleConnections()
	require.True(t, base.closed)
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type closeIdleRoundTripper struct {
	closed bool
}

func (*closeIdleRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("ok")),
		Request:    req,
	}, nil
}

func (t *closeIdleRoundTripper) CloseIdleConnections() {
	t.closed = true
}

func TestNewClientWithAccessTokenUsesAccessTokenAuth(t *testing.T) {
	t.Parallel()

	client, err := newClient("https://example.com", "token", "", "", defaultClientOptions())
	require.NoError(t, err)

	t.Cleanup(client.close)

	require.Equal(t, "", client.serviceAccount)
	require.Equal(t, "", client.serviceAccountSecret)
	require.Equal(t, 120*time.Second, client.httpClient.Timeout)
}

func TestNewClientFromWorldPrefersAccessToken(t *testing.T) {
	t.Parallel()

	w := &world.World{
		URL:                  "https://example.com",
		Timeout:              5 * time.Second,
		AccessToken:          "token",
		ServiceAccount:       "sa@example.com",
		ServiceAccountSecret: "secret",
	}

	client, err := newClientFromWorld(w)
	require.NoError(t, err)

	t.Cleanup(client.close)

	require.Equal(t, 5*time.Second, client.httpClient.Timeout)
	require.Equal(t, "", client.serviceAccount)
	require.Equal(t, "", client.serviceAccountSecret)
}
