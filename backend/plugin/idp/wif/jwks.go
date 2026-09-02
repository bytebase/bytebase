package wif

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/pkg/errors"
)

var (
	jwksCache     = make(map[string]*cachedJWKS)
	jwksCacheLock sync.RWMutex
	cacheDuration = 15 * time.Minute

	// httpClient with timeout for JWKS fetching.
	httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}
)

type cachedJWKS struct {
	jwks      *jose.JSONWebKeySet
	fetchedAt time.Time
}

type oidcConfig struct {
	JwksURI string `json:"jwks_uri"`
}

// FetchJWKS fetches the JWKS from an OIDC issuer or a direct JWKS URL with caching.
func FetchJWKS(ctx context.Context, issuerURL, jwksURL string) (*jose.JSONWebKeySet, error) {
	// Validate issuer URL format
	if err := validateIssuerURL(issuerURL); err != nil {
		return nil, err
	}

	effectiveJWKSURL := jwksURL
	if effectiveJWKSURL == "" {
		configURL := fmt.Sprintf("%s/.well-known/openid-configuration", issuerURL)
		config, err := fetchOIDCConfig(ctx, configURL)
		if err != nil {
			return nil, err
		}
		effectiveJWKSURL = config.JwksURI
	}
	if err := validateJWKSURL(effectiveJWKSURL); err != nil {
		return nil, err
	}

	// Check cache by the endpoint that actually serves the signing keys.
	jwksCacheLock.RLock()
	if cached, ok := jwksCache[effectiveJWKSURL]; ok {
		if time.Since(cached.fetchedAt) < cacheDuration {
			jwksCacheLock.RUnlock()
			return cached.jwks, nil
		}
	}
	jwksCacheLock.RUnlock()

	// Fetch JWKS
	jwks, err := fetchJWKSFromURL(ctx, effectiveJWKSURL)
	if err != nil {
		return nil, err
	}

	// Update cache
	jwksCacheLock.Lock()
	jwksCache[effectiveJWKSURL] = &cachedJWKS{
		jwks:      jwks,
		fetchedAt: time.Now(),
	}
	jwksCacheLock.Unlock()

	return jwks, nil
}

// validateIssuerURL validates that the issuer URL is a valid HTTPS URL.
func validateIssuerURL(issuerURL string) error {
	return validateRemoteURL(issuerURL, "issuer URL")
}

func validateJWKSURL(jwksURL string) error {
	return validateRemoteURL(jwksURL, "JWKS URL")
}

func validateRemoteURL(rawURL, label string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return errors.Wrapf(err, "invalid %s", label)
	}
	if parsed.Scheme != "https" {
		return errors.Errorf("%s must use HTTPS: %s", label, rawURL)
	}
	if parsed.Host == "" {
		return errors.Errorf("%s must have a host: %s", label, rawURL)
	}
	// Prevent localhost and private IPs in production (basic SSRF prevention)
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasPrefix(host, "127.") || strings.HasPrefix(host, "10.") ||
		strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "172.") {
		return errors.Errorf("%s cannot be a private address: %s", label, rawURL)
	}
	return nil
}

func fetchOIDCConfig(ctx context.Context, configURL string) (*oidcConfig, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, configURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch OIDC config from %s", configURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("failed to fetch OIDC config: HTTP %d", resp.StatusCode)
	}

	var config oidcConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, errors.Wrap(err, "failed to decode OIDC config")
	}

	return &config, nil
}

func fetchJWKSFromURL(ctx context.Context, jwksURL string) (*jose.JSONWebKeySet, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch JWKS from %s", jwksURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("failed to fetch JWKS: HTTP %d", resp.StatusCode)
	}

	var jwks jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, errors.Wrap(err, "failed to decode JWKS")
	}

	return &jwks, nil
}
