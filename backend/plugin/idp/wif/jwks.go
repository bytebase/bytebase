package wif

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/pkg/errors"
)

var (
	jwksCache     = make(map[string]*cachedJWKS)
	jwksCacheLock sync.RWMutex
	cacheDuration = 15 * time.Minute
)

type cachedJWKS struct {
	jwks      *jose.JSONWebKeySet
	fetchedAt time.Time
}

type oidcConfig struct {
	JwksURI string `json:"jwks_uri"`
}

// FetchJWKS fetches the JWKS from an OIDC issuer with caching.
func FetchJWKS(_ context.Context, issuerURL string) (*jose.JSONWebKeySet, error) {
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
	resp, err := http.Get(configURL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch OIDC config from %s", configURL)
	}
	defer resp.Body.Close()

	var config oidcConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, errors.Wrap(err, "failed to decode OIDC config")
	}

	// Fetch JWKS
	jwksResp, err := http.Get(config.JwksURI)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch JWKS from %s", config.JwksURI)
	}
	defer jwksResp.Body.Close()

	var jwks jose.JSONWebKeySet
	if err := json.NewDecoder(jwksResp.Body).Decode(&jwks); err != nil {
		return nil, errors.Wrap(err, "failed to decode JWKS")
	}

	// Update cache
	jwksCacheLock.Lock()
	jwksCache[issuerURL] = &cachedJWKS{
		jwks:      &jwks,
		fetchedAt: time.Now(),
	}
	jwksCacheLock.Unlock()

	return &jwks, nil
}
