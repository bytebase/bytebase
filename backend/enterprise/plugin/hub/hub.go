package hub

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/enterprise/config"
	"github.com/bytebase/bytebase/backend/enterprise/plugin"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
	v1pb "github.com/bytebase/bytebase/proto/generated-go/v1"
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

// licenseInfo is an internal type for JWT parsing
type licenseInfo struct {
	InstanceCount int
	Seat          int
	ExpiresTS     int64
	IssuedTS      int64
	Plan          v1pb.PlanType
	Trialing      bool
	OrgName       string
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
func (p *Provider) StoreLicense(ctx context.Context, license string) error {
	if license != "" {
		if _, err := p.parseLicense(ctx, license); err != nil {
			return err
		}
	}
	if _, err := p.store.UpsertSettingV2(ctx, &store.SetSettingMessage{
		Name:  storepb.SettingName_ENTERPRISE_LICENSE,
		Value: license,
	}); err != nil {
		return err
	}

	return nil
}

// LoadSubscription will load the hub subscription.
func (p *Provider) LoadSubscription(ctx context.Context) *v1pb.Subscription {
	license := p.loadLicense(ctx)
	if license == nil {
		return &v1pb.Subscription{
			Plan: v1pb.PlanType_FREE,
			// nil ExpiresTime means not expire, just for free plan
			ExpiresTime: nil,
			// Instance license count.
			InstanceCount: 0,
			SeatCount:     0,
		}
	}

	var expiresTime *timestamppb.Timestamp
	if license.ExpiresTS > 0 {
		expiresTime = timestamppb.New(time.Unix(license.ExpiresTS, 0))
	}

	return &v1pb.Subscription{
		Plan:          license.Plan,
		ExpiresTime:   expiresTime,
		InstanceCount: int32(license.InstanceCount),
		SeatCount:     int32(license.Seat),
		Trialing:      license.Trialing,
		OrgName:       license.OrgName,
	}
}

func (p *Provider) fetchLicense(ctx context.Context) (*licenseInfo, error) {
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
		Name:  storepb.SettingName_ENTERPRISE_LICENSE,
		Value: license,
	}); err != nil {
		return nil, errors.Wrapf(err, "failed to store the license")
	}

	return result, nil
}

// loadLicense will load license and validate it.
func (p *Provider) loadLicense(ctx context.Context) *licenseInfo {
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

	if expireTime := time.Unix(license.ExpiresTS, 0); expireTime.Before(time.Now()) {
		slog.Debug("license has expired at %v", expireTime)
	}

	return license
}

func (p *Provider) parseLicense(ctx context.Context, license string) (*licenseInfo, error) {
	claim := &claims{}
	if err := parseJWTToken(license, p.config.Version, p.config.PublicKey, claim); err != nil {
		return nil, common.Wrap(err, common.Invalid)
	}

	return p.parseClaims(ctx, claim)
}

func (p *Provider) findEnterpriseLicense(ctx context.Context) (*licenseInfo, error) {
	// Find enterprise license.
	setting, err := p.store.GetSettingV2(ctx, storepb.SettingName_ENTERPRISE_LICENSE)
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
func (p *Provider) parseClaims(ctx context.Context, claim *claims) (*licenseInfo, error) {
	if p.config.Issuer != claim.Issuer {
		return nil, common.Errorf(common.Invalid, "iss is not valid, expect %s but found '%v'", p.config.Issuer, claim.Issuer)
	}
	if !slices.Contains(claim.Audience, p.config.Audience) {
		return nil, common.Errorf(common.Invalid, "aud is not valid, expect %s but found '%v'", p.config.Audience, claim.Audience)
	}

	v, ok := v1pb.PlanType_value[claim.Plan]
	if !ok {
		return nil, common.Errorf(common.Invalid, "plan type %q is not valid", claim.Plan)
	}
	planType := v1pb.PlanType(v)

	if claim.WorkspaceID != "" && planType == v1pb.PlanType_ENTERPRISE && !claim.Trialing {
		workspaceID, err := p.store.GetWorkspaceID(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get workspace id from setting")
		}
		if workspaceID != claim.WorkspaceID {
			return nil, common.Errorf(common.Invalid, "the workspace id not match")
		}
	}

	license := &licenseInfo{
		InstanceCount: claim.InstanceCount,
		Seat:          claim.Seat,
		ExpiresTS:     claim.ExpiresAt.Unix(),
		IssuedTS:      claim.IssuedAt.Unix(),
		Plan:          planType,
		Trialing:      claim.Trialing,
		OrgName:       claim.OrgName,
	}

	return license, nil
}
