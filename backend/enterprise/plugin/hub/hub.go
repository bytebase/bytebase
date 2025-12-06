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
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

var _ plugin.LicenseProvider = (*Provider)(nil)

// Provider is the Bytebase Hub license provider.
type Provider struct {
	config *config.Config
	store  *store.Store
}

// claims creates a struct that will be encoded to a JWT.
// We add jwt.RegisteredClaims as an embedded type, to provide fields such as name.
type claims struct {
	ActiveInstances int    `json:"instanceCount"`
	Instances       int    `json:"instance"`
	Seats           int    `json:"seat"`
	Trialing        bool   `json:"trialing"`
	Plan            string `json:"plan"`
	OrgName         string `json:"orgName"`
	WorkspaceID     string `json:"workspaceId"`
	jwt.RegisteredClaims
}

// validateSubscription validates if the subscription is expired or has correct plan type.

// NewProvider will create a new hub license provider.
func NewProvider(providerConfig *plugin.ProviderConfig) (plugin.LicenseProvider, error) {
	config, err := config.NewConfig(providerConfig.Mode)
	if err != nil {
		return nil, err
	}

	return &Provider{
		store:  providerConfig.Store,
		config: config,
	}, nil
}

// StoreLicense will store the hub license.
func (p *Provider) StoreLicense(ctx context.Context, license string) error {
	if license != "" {
		systemSetting, err := p.store.GetSystemSetting(ctx)
		if err != nil {
			return errors.Wrapf(err, "failed to get system setting")
		}
		if _, err := p.parseLicense(license, systemSetting.WorkspaceId); err != nil {
			return err
		}
	}
	return p.store.UpdateLicense(ctx, license)
}

// LoadSubscription will load the hub subscription.
func (p *Provider) LoadSubscription(ctx context.Context) *v1pb.Subscription {
	systemSetting, err := p.store.GetSystemSetting(ctx)
	if err != nil {
		slog.Debug("failed to load enterprise license", log.BBError(errors.Wrapf(err, "failed to get system setting")))
		return nil
	}

	if systemSetting.License == "" {
		return nil
	}

	license, err := p.parseLicense(systemSetting.License, systemSetting.WorkspaceId)
	if err != nil {
		slog.Debug("failed to load enterprise license", log.BBError(errors.Wrapf(err, "failed to parse enterprise license")))
		return nil
	}

	slog.Debug(
		"Load valid license",
		slog.String("plan", license.Plan.String()),
		slog.Time("expiresAt", license.ExpiresTime.AsTime()),
		slog.Int("activeInstances", int(license.ActiveInstances)),
		slog.Int("instances", int(license.Instances)),
		slog.Int("seats", int(license.Seats)),
	)

	return license
}

func (p *Provider) parseLicense(license, workspaceID string) (*v1pb.Subscription, error) {
	claim := &claims{}
	token, err := jwt.ParseWithClaims(license, claim, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, common.Errorf(common.Invalid, "unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok || kid != p.config.Version {
			return nil, common.Errorf(common.Invalid, "version '%v' is not valid. expect %s", token.Header["kid"], p.config.Version)
		}

		return p.config.PublicKey, nil
	})
	if err != nil {
		return nil, common.Wrap(err, common.Invalid)
	}

	if !token.Valid {
		return nil, common.Errorf(common.Invalid, "invalid token")
	}

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
		if workspaceID != claim.WorkspaceID {
			return nil, common.Errorf(common.Invalid, "the workspace id not match")
		}
	}

	switch planType {
	case v1pb.PlanType_FREE, v1pb.PlanType_TEAM, v1pb.PlanType_ENTERPRISE:
	default:
		return nil, errors.Errorf("plan %q is not valid, expect %s or %s",
			planType.String(),
			v1pb.PlanType_TEAM.String(),
			v1pb.PlanType_ENTERPRISE.String(),
		)
	}

	var expiresTime *timestamppb.Timestamp
	if claim.ExpiresAt != nil && !claim.ExpiresAt.IsZero() {
		expiresTime = timestamppb.New(claim.ExpiresAt.Time)
	}
	if expiresTime != nil && expiresTime.AsTime().Before(time.Now()) {
		return nil, errors.Errorf("license has expired at %v", expiresTime.AsTime())
	}

	return &v1pb.Subscription{
		ActiveInstances: int32(claim.ActiveInstances),
		Instances:       int32(claim.Instances),
		Seats:           int32(claim.Seats),
		ExpiresTime:     expiresTime,
		Plan:            planType,
		Trialing:        claim.Trialing,
		OrgName:         claim.OrgName,
	}, nil
}
