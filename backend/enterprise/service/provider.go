package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"

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
		return "", nil
	}

	settingName := api.SettingHubToken
	setting, err := p.store.GetSetting(ctx, &api.SettingFind{
		Name: &settingName,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to find the hub token from settings")
	}

	url := fmt.Sprintf("%s/api/v1/license", p.config.HubAPIURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", errors.Wrapf(err, "construct GET %s", url)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer: %s", setting.Value))
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

	p.lastFetchTime = time.Now().Unix()
	return response.License, nil
}
