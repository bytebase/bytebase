package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/enterprise/config"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	fetchLicenseInterval = time.Duration(1) * time.Minute
)

// getLicenseResponse is the API message for getting license response from the hub.
type getLicenseResponse struct {
	License string `json:"license"`
}

// internalTokenClaims is the token claims for internal API call.
type internalTokenClaims struct {
	PrincipalID int64  `json:"principalId"`
	OrgID       string `json:"orgId"`
	WorkspaceID string `json:"workspaceId"`
	jwt.RegisteredClaims
}

// LicenseProvider is the service to fetch license from the hub.
type LicenseProvider struct {
	config        *config.Config
	store         *store.Store
	client        *http.Client
	lastFetchTime int64
}

// NewLicenseProvider will create a new license provider.
func NewLicenseProvider(config *config.Config, store *store.Store) *LicenseProvider {
	return &LicenseProvider{
		store:  store,
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		lastFetchTime: 0,
	}
}

// FetchLicense will fetch the license from the hub.
func (p *LicenseProvider) FetchLicense(ctx context.Context) (string, error) {
	nextFetchTime := p.lastFetchTime + int64(fetchLicenseInterval.Seconds())
	if time.Now().Unix() < nextFetchTime {
		log.Debug(fmt.Sprintf("skip fetching license until %d", nextFetchTime))
		return "", nil
	}

	settingName := api.SettingHubToken
	setting, err := p.store.GetSetting(ctx, &api.SettingFind{
		Name: &settingName,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to find the hub token from settings")
	}
	if setting == nil || setting.Value == "" {
		return "", errors.Wrapf(err, "hub token not found")
	}
	claims, err := p.parseJWTToken(setting.Value)
	if err != nil {
		return "", errors.Wrapf(err, "invalid internal token")
	}

	defer func() {
		p.lastFetchTime = time.Now().Unix()
	}()

	return p.requestLicense(ctx, setting.Value, claims)
}

func (p *LicenseProvider) requestLicense(ctx context.Context, token string, claims *internalTokenClaims) (string, error) {
	url := fmt.Sprintf("%s/v1/orgs/%s/workspaces/%s/license", p.config.HubAPIURL, claims.OrgID, claims.WorkspaceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", errors.Wrapf(err, "construct GET %s", url)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err := p.client.Do(req)
	if err != nil {
		return "", errors.Wrapf(err, "GET %s", url)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrapf(err, "read body of GET %s", url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("non-200 GET status code %d with body %q", resp.StatusCode, b)
	}

	var response getLicenseResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return "", errors.Wrapf(err, "unmarshal body from GET %s", url)
	}

	return response.License, nil
}

func (p *LicenseProvider) parseJWTToken(tokenStr string) (*internalTokenClaims, error) {
	claims := &internalTokenClaims{}
	if err := parseJWTToken(tokenStr, p.config.Version, p.config.PublicKey, claims); err != nil {
		return nil, common.Wrap(err, common.Invalid)
	}

	return claims, nil
}
