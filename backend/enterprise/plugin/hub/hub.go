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
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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

var validPlans = map[v1pb.PlanType]bool{
	v1pb.PlanType_TEAM:       true,
	v1pb.PlanType_ENTERPRISE: true,
}

// validateSubscription validates if the subscription is expired or has correct plan type.
func validateSubscription(s *v1pb.Subscription) error {
	if !validPlans[s.Plan] {
		return errors.Errorf("plan %q is not valid, expect %s or %s",
			s.Plan.String(),
			v1pb.PlanType_TEAM.String(),
			v1pb.PlanType_ENTERPRISE.String(),
		)
	}
	if s.ExpiresTime != nil && s.ExpiresTime.AsTime().Before(time.Now()) {
		return errors.Errorf("license has expired at %v", s.ExpiresTime.AsTime())
	}
	return nil
}

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
	license, err := p.findEnterpriseLicense(ctx)
	if err != nil {
		slog.Debug("failed to load enterprise license", log.BBError(err))
	}

	if license == nil {
		return nil
	}

	if err := validateSubscription(license); err != nil {
		slog.Debug("license is invalid", log.BBError(err))
		return nil
	}
	return license
}

func (p *Provider) parseLicense(ctx context.Context, license string) (*v1pb.Subscription, error) {
	claim := &claims{}
	if err := parseJWTToken(license, p.config.Version, p.config.PublicKey, claim); err != nil {
		return nil, common.Wrap(err, common.Invalid)
	}

	return p.parseClaims(ctx, claim)
}

func (p *Provider) findEnterpriseLicense(ctx context.Context) (*v1pb.Subscription, error) {
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
				slog.Time("expiresAt", license.ExpiresTime.AsTime()),
				slog.Int("activeInstances", int(license.ActiveInstances)),
				slog.Int("instances", int(license.Instances)),
				slog.Int("seats", int(license.Seats)),
			)
			return license, nil
		}
	}

	return nil, nil
}

// parseClaims will valid and parse JWT claims to license instance.
func (p *Provider) parseClaims(ctx context.Context, claim *claims) (*v1pb.Subscription, error) {
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

	var expiresTime *timestamppb.Timestamp
	if claim.ExpiresAt != nil && !claim.ExpiresAt.IsZero() {
		expiresTime = timestamppb.New(claim.ExpiresAt.Time)
	}

	license := &v1pb.Subscription{
		ActiveInstances: int32(claim.ActiveInstances),
		Instances:       int32(claim.Instances),
		Seats:           int32(claim.Seats),
		ExpiresTime:     expiresTime,
		Plan:            planType,
		Trialing:        claim.Trialing,
		OrgName:         claim.OrgName,
	}

	return license, nil
}

func parseJWTToken(tokenString, expectVersion, publicKey string, claims jwt.Claims) error {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, common.Errorf(common.Invalid, "unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok || kid != expectVersion {
			return nil, common.Errorf(common.Invalid, "version '%v' is not valid. expect %s", token.Header["kid"], expectVersion)
		}

		key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(publicKey))
		if err != nil {
			return nil, common.Wrap(err, common.Invalid)
		}

		return key, nil
	})
	if err != nil {
		return common.Wrap(err, common.Invalid)
	}

	if !token.Valid {
		return common.Errorf(common.Invalid, "invalid token")
	}

	return nil
}
