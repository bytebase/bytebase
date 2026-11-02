package wif

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"testing"

	"github.com/go-jose/go-jose/v4"
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

func TestFetchJWKSReusesCachedDiscoveryResult(t *testing.T) {
	requestCount := map[string]int{}
	setJWKSHTTPClientForTest(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestCount[request.URL.String()]++
		switch request.URL.String() {
		case "https://issuer.example.com/.well-known/openid-configuration":
			return jsonHTTPResponse(http.StatusOK, `{"jwks_uri":"https://keys.example.com/discovered.json"}`), nil
		case "https://keys.example.com/discovered.json":
			return jsonHTTPResponse(http.StatusOK, `{"keys":[]}`), nil
		default:
			return jsonHTTPResponse(http.StatusNotFound, `{}`), nil
		}
	}))

	for range 2 {
		_, err := FetchJWKS(context.Background(), "https://issuer.example.com", "")
		require.NoError(t, err)
	}

	require.Equal(t, map[string]int{
		"https://issuer.example.com/.well-known/openid-configuration": 1,
		"https://keys.example.com/discovered.json":                    1,
	}, requestCount)
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

func TestFetchJWKSKeepsIssuerAndDirectURLCacheKeysSeparate(t *testing.T) {
	requestCount := map[string]int{}
	setJWKSHTTPClientForTest(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestCount[request.URL.String()]++
		switch request.URL.String() {
		case "https://issuer.example.com/.well-known/openid-configuration":
			return jsonHTTPResponse(http.StatusOK, `{"jwks_uri":"https://keys.example.com/discovered.json"}`), nil
		default:
			return jsonHTTPResponse(http.StatusOK, `{"keys":[]}`), nil
		}
	}))

	_, err := FetchJWKS(context.Background(), "https://issuer.example.com", "")
	require.NoError(t, err)
	_, err = FetchJWKS(
		context.Background(),
		"https://other-issuer.example.com",
		"https://issuer.example.com",
	)
	require.NoError(t, err)

	require.Equal(t, 1, requestCount["https://issuer.example.com"])
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

func TestFetchJWKSKeepsUsableKeysBesideOneItCannotParse(t *testing.T) {
	usable := rsaJWKJSON(t, "usable", 2048)
	oversized := rsaJWKJSON(t, "oversized", jose.MaxRSAModulusBits+1)
	setJWKSHTTPClientForTest(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonHTTPResponse(http.StatusOK, `{"keys":[`+oversized+`,`+usable+`]}`), nil
	}))

	jwks, err := FetchJWKS(
		context.Background(),
		"https://issuer.example.com",
		"https://keys.example.com/direct.json",
	)
	require.NoError(t, err)
	require.Len(t, jwks.Keys, 1)
	require.Equal(t, "usable", jwks.Keys[0].KeyID)
	require.Len(t, jwks.Key("usable"), 1)
}

func TestFetchJWKSErrorsWhenNoPublishedKeyParses(t *testing.T) {
	oversized := rsaJWKJSON(t, "oversized", jose.MaxRSAModulusBits+1)
	requestCount := 0
	setJWKSHTTPClientForTest(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		requestCount++
		return jsonHTTPResponse(http.StatusOK, `{"keys":[`+oversized+`]}`), nil
	}))

	for range 2 {
		_, err := FetchJWKS(
			context.Background(),
			"https://issuer.example.com",
			"https://keys.example.com/direct.json",
		)
		require.Error(t, err)
	}
	// The emptied set must not reach the cache, or a fixed issuer would stay
	// unreachable for cacheDuration.
	require.Equal(t, 2, requestCount)
}

// rsaJWKJSON renders a public RSA JWK whose modulus has the given bit length.
// The modulus is fabricated, not generated: JWK parsing reads only n and e, and
// generating a key above jose.MaxRSAModulusBits would dominate the test.
func rsaJWKJSON(t *testing.T, kid string, modulusBits int) string {
	t.Helper()
	modulus := new(big.Int).SetBit(new(big.Int).Lsh(big.NewInt(1), uint(modulusBits-1)), 0, 1)
	require.Equal(t, modulusBits, modulus.BitLen())
	return fmt.Sprintf(`{"kty":"RSA","kid":%q,"alg":"RS256","use":"sig","n":%q,"e":%q}`,
		kid,
		base64.RawURLEncoding.EncodeToString(modulus.Bytes()),
		base64.RawURLEncoding.EncodeToString(big.NewInt(65537).Bytes()),
	)
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
