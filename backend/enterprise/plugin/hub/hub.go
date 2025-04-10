package hub

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/base"
	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	enterprise "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/enterprise/config"
	"github.com/bytebase/bytebase/backend/enterprise/plugin"
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
	Seat          int    `json:"seat"`
	Trialing      bool   `json:"trialing"`
	Plan          string `json:"plan"`
	OrgName       string `json:"orgName"`
	WorkspaceID   string `json:"workspaceId"`
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
		if _, err := p.parseLicense(ctx, patch.License); err != nil {
			return err
		}
	}
	if _, err := p.store.UpsertSettingV2(ctx, &store.SetSettingMessage{
		Name:  base.SettingEnterpriseLicense,
		Value: patch.License,
	}); err != nil {
		return err
	}

	return nil
}

// LoadSubscription will load the hub subscription.
func (p *Provider) LoadSubscription(ctx context.Context) *enterprise.Subscription {
	license := p.loadLicense(ctx)
	if license == nil {
		return &enterprise.Subscription{
			Plan: base.FREE,
			// -1 means not expire, just for free plan
			ExpiresTS: -1,
			// Instance license count.
			InstanceCount: 0,
			Seat:          0,
		}
	}

	return &enterprise.Subscription{
		Plan:          license.Plan,
		ExpiresTS:     license.ExpiresTS,
		StartedTS:     license.IssuedTS,
		InstanceCount: license.InstanceCount,
		Seat:          license.Seat,
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
	result, err := p.parseLicense(ctx, license)
	if err != nil {
		return nil, err
	}

	if _, err := p.store.UpsertSettingV2(ctx, &store.SetSettingMessage{
		Name:  base.SettingEnterpriseLicense,
		Value: license,
	}); err != nil {
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

func (p *Provider) parseLicense(ctx context.Context, license string) (*enterprise.License, error) {
	claim := &claims{}
	if err := parseJWTToken(license, p.config.Version, p.config.PublicKey, claim); err != nil {
		return nil, common.Wrap(err, common.Invalid)
	}

	return p.parseClaims(ctx, claim)
}

func (p *Provider) findEnterpriseLicense(ctx context.Context) (*enterprise.License, error) {
	// Find enterprise license.
	setting, err := p.store.GetSettingV2(ctx, base.SettingEnterpriseLicense)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load enterprise license from settings")
	}
	if setting != nil && setting.Value != "" {
		license, err := p.parseLicense(ctx, setting.Value)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse enterprise license")
		}
		if license != nil {
			slog.Debug(
				"Load valid license",
				slog.String("plan", license.Plan.String()),
				slog.Time("expiresAt", time.Unix(license.ExpiresTS, 0)),
				slog.Int("instanceCount", license.InstanceCount),
			)
			return license, nil
		}
	}

	return nil, nil
}

// parseClaims will valid and parse JWT claims to license instance.
func (p *Provider) parseClaims(ctx context.Context, claim *claims) (*enterprise.License, error) {
	if p.config.Issuer != claim.Issuer {
		return nil, common.Errorf(common.Invalid, "iss is not valid, expect %s but found '%v'", p.config.Issuer, claim.Issuer)
	}
	if !slices.Contains(claim.Audience, p.config.Audience) {
		return nil, common.Errorf(common.Invalid, "aud is not valid, expect %s but found '%v'", p.config.Audience, claim.Audience)
	}

	planType, err := convertPlanType(claim.Plan)
	if err != nil {
		return nil, common.Errorf(common.Invalid, "plan type %q is not valid", planType)
	}

	if claim.WorkspaceID != "" && planType == base.ENTERPRISE && !claim.Trialing {
		workspaceID, err := p.store.GetWorkspaceID(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get workspace id from setting")
		}
		if workspaceID != claim.WorkspaceID {
			return nil, common.Errorf(common.Invalid, "the workspace id not match")
		}
	}

	license := &enterprise.License{
		InstanceCount: claim.InstanceCount,
		Seat:          claim.Seat,
		ExpiresTS:     claim.ExpiresAt.Unix(),
		IssuedTS:      claim.IssuedAt.Unix(),
		Plan:          planType,
		Subject:       claim.Subject,
		Trialing:      claim.Trialing,
		OrgName:       claim.OrgName,
	}

	return license, nil
}

func convertPlanType(candidate string) (base.PlanType, error) {
	switch candidate {
	case base.TEAM.String():
		return base.TEAM, nil
	case base.ENTERPRISE.String():
		return base.ENTERPRISE, nil
	case base.FREE.String():
		return base.FREE, nil
	default:
		return base.FREE, errors.Errorf("cannot conver plan type %q", candidate)
	}
}
