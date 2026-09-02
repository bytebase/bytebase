package wif

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestFetchJWKSUsesDiscoveryWhenDirectURLIsEmpty(t *testing.T) {
	var requestedURLs []string
	setJWKSHTTPClientForTest(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestedURLs = append(requestedURLs, request.URL.String())
		switch request.URL.String() {
		case "https://issuer.example.com/.well-known/openid-configuration":
			return jsonHTTPResponse(http.StatusOK, `{"jwks_uri":"https://keys.example.com/discovered.json"}`), nil
		case "https://keys.example.com/discovered.json":
			return jsonHTTPResponse(http.StatusOK, `{"keys":[]}`), nil
		default:
			return jsonHTTPResponse(http.StatusNotFound, `{}`), nil
		}
	}))

	_, err := FetchJWKS(context.Background(), "https://issuer.example.com", "")
	require.NoError(t, err)
	require.Equal(t, []string{
		"https://issuer.example.com/.well-known/openid-configuration",
		"https://keys.example.com/discovered.json",
	}, requestedURLs)
}

func TestFetchJWKSUsesDirectURLWithoutDiscovery(t *testing.T) {
	var requestedURLs []string
	setJWKSHTTPClientForTest(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestedURLs = append(requestedURLs, request.URL.String())
		return jsonHTTPResponse(http.StatusOK, `{"keys":[]}`), nil
	}))

	_, err := FetchJWKS(
		context.Background(),
		"https://issuer.example.com",
		"https://keys.example.com/direct.json",
	)
	require.NoError(t, err)
	require.Equal(t, []string{"https://keys.example.com/direct.json"}, requestedURLs)
}

func TestFetchJWKSUsesDiscoveredURLAsCacheKey(t *testing.T) {
	requestCount := map[string]int{}
	setJWKSHTTPClientForTest(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestCount[request.URL.String()]++
		if strings.HasSuffix(request.URL.String(), "/.well-known/openid-configuration") {
			return jsonHTTPResponse(http.StatusOK, `{"jwks_uri":"https://keys.example.com/shared.json"}`), nil
		}
		return jsonHTTPResponse(http.StatusOK, `{"keys":[]}`), nil
	}))

	for _, issuerURL := range []string{
		"https://first-issuer.example.com",
		"https://second-issuer.example.com",
	} {
		_, err := FetchJWKS(context.Background(), issuerURL, "")
		require.NoError(t, err)
	}

	require.Equal(t, 1, requestCount["https://keys.example.com/shared.json"])
}

func TestFetchJWKSSeparatesDirectURLCacheEntries(t *testing.T) {
	requestCount := map[string]int{}
	setJWKSHTTPClientForTest(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestCount[request.URL.String()]++
		return jsonHTTPResponse(http.StatusOK, `{"keys":[]}`), nil
	}))

	for _, keyURL := range []string{
		"https://keys.example.com/first.json",
		"https://keys.example.com/second.json",
		"https://keys.example.com/first.json",
	} {
		_, err := FetchJWKS(context.Background(), "https://issuer.example.com", keyURL)
		require.NoError(t, err)
	}

	require.Equal(t, map[string]int{
		"https://keys.example.com/first.json":  1,
		"https://keys.example.com/second.json": 1,
	}, requestCount)
}

func TestFetchJWKSRejectsInvalidDirectURL(t *testing.T) {
	setJWKSHTTPClientForTest(t, roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		t.Fatal("HTTP client must not be called for an invalid URL")
		return nil, nil
	}))

	for _, keyURL := range []string{
		"http://keys.example.com/jwks.json",
		"https://localhost/jwks.json",
		"https://127.0.0.1/jwks.json",
		"https://10.0.0.1/jwks.json",
		"https://192.168.1.1/jwks.json",
	} {
		t.Run(keyURL, func(t *testing.T) {
			_, err := FetchJWKS(context.Background(), "https://issuer.example.com", keyURL)
			require.Error(t, err)
		})
	}
}

func TestFetchJWKSErrorDoesNotIncludeResponseBody(t *testing.T) {
	const secretBody = "sensitive-upstream-response"
	setJWKSHTTPClientForTest(t, roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusInternalServerError, secretBody), nil
	}))

	_, err := FetchJWKS(
		context.Background(),
		"https://issuer.example.com",
		"https://keys.example.com/direct.json",
	)
	require.Error(t, err)
	require.NotContains(t, err.Error(), secretBody)
}

func setJWKSHTTPClientForTest(t *testing.T, transport http.RoundTripper) {
	t.Helper()

	previousClient := httpClient
	previousCache := jwksCache
	httpClient = &http.Client{Transport: transport}
	jwksCache = make(map[string]*cachedJWKS)
	t.Cleanup(func() {
		httpClient = previousClient
		jwksCache = previousCache
	})
}

func jsonHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
