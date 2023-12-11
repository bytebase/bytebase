package hub

import (
	"context"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/enterprise/config"
	"github.com/bytebase/bytebase/backend/enterprise/plugin"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

var _ plugin.LicenseProvider = (*Provider)(nil)

// Provider is the Bytebase Hub license provider.
type Provider struct {
	config         *config.Config
	store          *store.Store
	remoteProvider *remoteLicenseProvider
}

// claims creates a struct that will be encoded to a JWT.
// We add jwt.RegisteredClaims as an embedded type, to provide fields such as name.
type claims struct {
	InstanceCount int    `json:"instanceCount"`
	Trialing      bool   `json:"trialing"`
	Plan          string `json:"plan"`
	OrgName       string `json:"orgName"`
	jwt.RegisteredClaims
}

// NewProvider will create a new hub license provider.
func NewProvider(providerConfig *plugin.ProviderConfig) (plugin.LicenseProvider, error) {
	config, err := config.NewConfig(providerConfig.Mode)
	if err != nil {
		return nil, err
	}

	return &Provider{
		store:          providerConfig.Store,
		config:         config,
		remoteProvider: newRemoteLicenseProvider(config, providerConfig.Store),
	}, nil
}

// StoreLicense will store the hub license.
func (p *Provider) StoreLicense(ctx context.Context, patch *enterprise.SubscriptionPatch) error {
	if patch.License != "" {
		if _, err := p.parseLicense(patch.License); err != nil {
			return err
		}
	}
	if _, err := p.store.UpsertSettingV2(ctx, &store.SetSettingMessage{
		Name:  api.SettingEnterpriseLicense,
		Value: patch.License,
	}, patch.UpdaterID); err != nil {
		return err
	}

	return nil
}

// LoadSubscription will load the hub subscription.
func (p *Provider) LoadSubscription(ctx context.Context) *enterprise.Subscription {
	license := p.loadLicense(ctx)
	if license == nil {
		return &enterprise.Subscription{
			Plan: api.FREE,
			// -1 means not expire, just for free plan
			ExpiresTs: -1,
			// Instance license count.
			InstanceCount: 0,
		}
	}

	return &enterprise.Subscription{
		Plan:          license.Plan,
		ExpiresTs:     license.ExpiresTs,
		StartedTs:     license.IssuedTs,
		InstanceCount: license.InstanceCount,
		Trialing:      license.Trialing,
		OrgID:         license.OrgID(),
		OrgName:       license.OrgName,
	}
}

func (p *Provider) fetchLicense(ctx context.Context) (*enterprise.License, error) {
	license, err := p.remoteProvider.FetchLicense(ctx)
	if err != nil {
		return nil, err
	}
	if license == "" {
		return nil, nil
	}
	result, err := p.parseLicense(license)
	if err != nil {
		return nil, err
	}

	if _, err := p.store.UpsertSettingV2(ctx, &store.SetSettingMessage{
		Name:  api.SettingEnterpriseLicense,
		Value: license,
	}, api.SystemBotID); err != nil {
		return nil, errors.Wrapf(err, "failed to store the license")
	}

	return result, nil
}

// loadLicense will load license and validate it.
func (p *Provider) loadLicense(ctx context.Context) *enterprise.License {
	license, err := p.findEnterpriseLicense(ctx)
	if err != nil {
		slog.Debug("failed to load enterprise license", log.BBError(err))
	}

	if license == nil {
		license, err = p.fetchLicense(ctx)
		if err != nil {
			slog.Debug("failed to fetch license", log.BBError(err))
		}
	}
	if license == nil {
		return nil
	}
	if err := license.Valid(); err != nil {
		slog.Debug("license is invalid", log.BBError(err))
		return nil
	}

	return license
}

func (p *Provider) parseLicense(license string) (*enterprise.License, error) {
	claim := &claims{}
	if err := parseJWTToken(license, p.config.Version, p.config.PublicKey, claim); err != nil {
		return nil, common.Wrap(err, common.Invalid)
	}

	return p.parseClaims(claim)
}

func (p *Provider) findEnterpriseLicense(ctx context.Context) (*enterprise.License, error) {
	// Find enterprise license.
	settingName := api.SettingEnterpriseLicense
	setting, err := p.store.GetSettingV2(ctx, &store.FindSettingMessage{
		Name: &settingName,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load enterprise license from settings")
	}
	if setting != nil && setting.Value != "" {
		license, err := p.parseLicense(setting.Value)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse enterprise license")
		}
		if license != nil {
			slog.Debug(
				"Load valid license",
				slog.String("plan", license.Plan.String()),
				slog.Time("expiresAt", time.Unix(license.ExpiresTs, 0)),
				slog.Int("instanceCount", license.InstanceCount),
			)
			return license, nil
		}
	}

	return nil, nil
}

// parseClaims will valid and parse JWT claims to license instance.
func (p *Provider) parseClaims(claim *claims) (*enterprise.License, error) {
	verifyIssuer := claim.VerifyIssuer(p.config.Issuer, true)
	if !verifyIssuer {
		return nil, common.Errorf(common.Invalid, "iss is not valid, expect %s but found '%v'", p.config.Issuer, claim.Issuer)
	}

	verifyAudience := claim.VerifyAudience(p.config.Audience, true)
	if !verifyAudience {
		return nil, common.Errorf(common.Invalid, "aud is not valid, expect %s but found '%v'", p.config.Audience, claim.Audience)
	}

	planType, err := convertPlanType(claim.Plan)
	if err != nil {
		return nil, common.Errorf(common.Invalid, "plan type %q is not valid", planType)
	}

	license := &enterprise.License{
		InstanceCount: claim.InstanceCount,
		ExpiresTs:     claim.ExpiresAt.Unix(),
		IssuedTs:      claim.IssuedAt.Unix(),
		Plan:          planType,
		Subject:       claim.Subject,
		Trialing:      claim.Trialing,
		OrgName:       claim.OrgName,
	}

	return license, nil
}

func convertPlanType(candidate string) (api.PlanType, error) {
	switch candidate {
	case api.TEAM.String():
		return api.TEAM, nil
	case api.ENTERPRISE.String():
		return api.ENTERPRISE, nil
	case api.FREE.String():
		return api.FREE, nil
	default:
		return api.FREE, errors.Errorf("cannot conver plan type %q", candidate)
	}
}
