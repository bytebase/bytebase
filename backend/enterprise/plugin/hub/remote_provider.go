package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/enterprise/config"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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

// remoteLicenseProvider is the service to fetch license from the hub.
type remoteLicenseProvider struct {
	config        *config.Config
	store         *store.Store
	client        *http.Client
	lastFetchTime int64
}

// newRemoteLicenseProvider will create a new license provider.
func newRemoteLicenseProvider(config *config.Config, store *store.Store) *remoteLicenseProvider {
	return &remoteLicenseProvider{
		store:  store,
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		lastFetchTime: 0,
	}
}

// FetchLicense will fetch the license from the hub.
func (p *remoteLicenseProvider) FetchLicense(ctx context.Context) (string, error) {
	nextFetchTime := p.lastFetchTime + int64(fetchLicenseInterval.Seconds())
	if time.Now().Unix() < nextFetchTime {
		slog.Debug(fmt.Sprintf("skip fetching license until %d", nextFetchTime))
		return "", nil
	}

	settingName := api.SettingPluginAgent
	setting, err := p.store.GetSettingV2(ctx, &store.FindSettingMessage{
		Name: &settingName,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to find the hub token from settings")
	}
	if setting == nil || setting.Value == "" {
		return "", errors.Wrapf(err, "agent not found")
	}
	payload := new(storepb.AgentPluginSetting)
	if err := protojson.Unmarshal([]byte(setting.Value), payload); err != nil {
		return "", errors.Wrapf(err, "failed to parse agent")
	}
	if _, err := p.parseJWTToken(payload.Token); err != nil {
		return "", errors.Wrapf(err, "invalid internal token")
	}

	defer func() {
		p.lastFetchTime = time.Now().Unix()
	}()

	return p.requestLicense(ctx, payload.Url, payload.Token)
}

func (p *remoteLicenseProvider) requestLicense(ctx context.Context, agentURL, agentToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, agentURL, nil)
	if err != nil {
		return "", errors.Wrapf(err, "construct GET %s", agentURL)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", agentToken))
	resp, err := p.client.Do(req)
	if err != nil {
		return "", errors.Wrapf(err, "GET %s", agentURL)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.Wrapf(err, "read body of GET %s", agentURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("non-200 GET status code %d with body %q", resp.StatusCode, b)
	}

	var response getLicenseResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return "", errors.Wrapf(err, "unmarshal body from GET %s", agentURL)
	}

	return response.License, nil
}

func (p *remoteLicenseProvider) parseJWTToken(tokenStr string) (*internalTokenClaims, error) {
	claims := &internalTokenClaims{}
	if err := parseJWTToken(tokenStr, p.config.Version, p.config.PublicKey, claims); err != nil {
		return nil, common.Wrap(err, common.Invalid)
	}

	return claims, nil
}
