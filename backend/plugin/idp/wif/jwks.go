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

// FetchJWKS fetches the JWKS from an OIDC issuer with caching.
func FetchJWKS(ctx context.Context, issuerURL string) (*jose.JSONWebKeySet, error) {
	// Validate issuer URL format
	if err := validateIssuerURL(issuerURL); err != nil {
		return nil, err
	}

	// Check cache
	jwksCacheLock.RLock()
	if cached, ok := jwksCache[issuerURL]; ok {
		if time.Since(cached.fetchedAt) < cacheDuration {
			jwksCacheLock.RUnlock()
			return cached.jwks, nil
		}
	}
	jwksCacheLock.RUnlock()

	// Fetch OIDC configuration
	configURL := fmt.Sprintf("%s/.well-known/openid-configuration", issuerURL)
	config, err := fetchOIDCConfig(ctx, configURL)
	if err != nil {
		return nil, err
	}

	// Fetch JWKS
	jwks, err := fetchJWKSFromURL(ctx, config.JwksURI)
	if err != nil {
		return nil, err
	}

	// Update cache
	jwksCacheLock.Lock()
	jwksCache[issuerURL] = &cachedJWKS{
		jwks:      jwks,
		fetchedAt: time.Now(),
	}
	jwksCacheLock.Unlock()

	return jwks, nil
}

// validateIssuerURL validates that the issuer URL is a valid HTTPS URL.
func validateIssuerURL(issuerURL string) error {
	parsed, err := url.Parse(issuerURL)
	if err != nil {
		return errors.Wrap(err, "invalid issuer URL")
	}
	if parsed.Scheme != "https" {
		return errors.Errorf("issuer URL must use HTTPS: %s", issuerURL)
	}
	if parsed.Host == "" {
		return errors.Errorf("issuer URL must have a host: %s", issuerURL)
	}
	// Prevent localhost and private IPs in production (basic SSRF prevention)
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasPrefix(host, "127.") || strings.HasPrefix(host, "10.") ||
		strings.HasPrefix(host, "192.168.") || strings.HasPrefix(host, "172.") {
		return errors.Errorf("issuer URL cannot be a private address: %s", issuerURL)
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
