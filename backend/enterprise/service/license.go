// Package service implements the enterprise license service.
package service

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	enterpriseAPI "github.com/bytebase/bytebase/backend/enterprise/api"
	"github.com/bytebase/bytebase/backend/enterprise/config"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/store"
)

// LicenseService is the service for enterprise license.
type LicenseService struct {
	config             *config.Config
	store              *store.Store
	provider           *LicenseProvider
	cachedSubscription *enterpriseAPI.Subscription
}

// Claims creates a struct that will be encoded to a JWT.
// We add jwt.RegisteredClaims as an embedded type, to provide fields such as name.
type Claims struct {
	InstanceCount int    `json:"instanceCount"`
	Trialing      bool   `json:"trialing"`
	Plan          string `json:"plan"`
	OrgName       string `json:"orgName"`
	jwt.RegisteredClaims
}

// NewLicenseService will create a new enterprise license service.
func NewLicenseService(mode common.ReleaseMode, store *store.Store) (*LicenseService, error) {
	config, err := config.NewConfig(mode)
	if err != nil {
		return nil, err
	}

	return &LicenseService{
		store:    store,
		config:   config,
		provider: NewLicenseProvider(config, store),
	}, nil
}

// StoreLicense will store license into file.
func (s *LicenseService) StoreLicense(ctx context.Context, patch *enterpriseAPI.SubscriptionPatch) error {
	if patch.License != "" {
		if _, err := s.parseLicense(patch.License); err != nil {
			return err
		}
	}
	if _, err := s.store.UpsertSettingV2(ctx, &store.SetSettingMessage{
		Name:  api.SettingEnterpriseLicense,
		Value: patch.License,
	}, patch.UpdaterID); err != nil {
		return err
	}

	s.RefreshCache(ctx)
	return nil
}

// LoadSubscription will load subscription.
func (s *LicenseService) LoadSubscription(ctx context.Context) enterpriseAPI.Subscription {
	if s.cachedSubscription != nil && s.cachedSubscription.IsExpired() {
		// refresh expired subscription
		s.cachedSubscription = nil
	}
	if s.cachedSubscription != nil {
		return *s.cachedSubscription
	}

	license := s.loadLicense(ctx)
	if license == nil {
		s.store.RefreshSwap(false)
		return enterpriseAPI.Subscription{
			Plan: api.FREE,
			// -1 means not expire, just for free plan
			ExpiresTs: -1,
			// Instance license count.
			InstanceCount: 0,
		}
	}

	// Cache the subscription.
	s.cachedSubscription = &enterpriseAPI.Subscription{
		Plan:          license.Plan,
		ExpiresTs:     license.ExpiresTs,
		StartedTs:     license.IssuedTs,
		InstanceCount: license.InstanceCount,
		Trialing:      license.Trialing,
		OrgID:         license.OrgID(),
		OrgName:       license.OrgName,
	}
	if !license.Trialing && license.Plan != api.FREE {
		s.store.RefreshSwap(true)
	}
	return *s.cachedSubscription
}

// IsFeatureEnabled returns whether a feature is enabled.
func (s *LicenseService) IsFeatureEnabled(feature api.FeatureType) error {
	if !api.Feature(feature, s.GetEffectivePlan()) {
		return errors.Errorf(feature.AccessErrorMessage())
	}
	return nil
}

// IsFeatureEnabledForInstance returns whether a feature is enabled for the instance.
func (s *LicenseService) IsFeatureEnabledForInstance(feature api.FeatureType, instance *store.InstanceMessage) error {
	plan := s.GetEffectivePlan()
	// DONOT check instance license fo FREE plan.
	if plan == api.FREE {
		return s.IsFeatureEnabled(feature)
	}
	if err := s.IsFeatureEnabled(feature); err != nil {
		return err
	}
	if !api.InstanceLimitFeature[feature] {
		// If the feature not exists in the limit map, we just need to check the feature for current plan.
		return nil
	}
	if !instance.Activation {
		return errors.Errorf(`feature "%s" is not available for instance %s, please assign license to the instance to enable it`, feature.Name(), instance.ResourceID)
	}
	return nil
}

// GetInstanceLicenseCount returns the instance count limit for current subscription.
func (s *LicenseService) GetInstanceLicenseCount(ctx context.Context) int {
	instanceCount := s.LoadSubscription(ctx).InstanceCount
	if instanceCount < 0 {
		return math.MaxInt
	}
	return instanceCount
}

// GetEffectivePlan gets the effective plan.
func (s *LicenseService) GetEffectivePlan() api.PlanType {
	ctx := context.Background()
	subscription := s.LoadSubscription(ctx)
	if expireTime := time.Unix(subscription.ExpiresTs, 0); expireTime.Before(time.Now()) {
		return api.FREE
	}
	return subscription.Plan
}

// GetPlanLimitValue gets the limit value for the plan.
func (s *LicenseService) GetPlanLimitValue(name enterpriseAPI.PlanLimit) int64 {
	v, ok := enterpriseAPI.PlanLimitValues[name]
	if !ok {
		return 0
	}

	ctx := context.Background()
	subscription := s.LoadSubscription(ctx)

	limit := v[subscription.Plan]
	if subscription.Trialing {
		limit = v[api.FREE]
	}

	if limit == -1 {
		return math.MaxInt64
	}
	return limit
}

// RefreshCache will invalidate and refresh the subscription cache.
func (s *LicenseService) RefreshCache(ctx context.Context) {
	s.cachedSubscription = nil
	s.LoadSubscription(ctx)
}

func (s *LicenseService) fetchLicense(ctx context.Context) (*enterpriseAPI.License, error) {
	license, err := s.provider.FetchLicense(ctx)
	if err != nil {
		return nil, err
	}
	if license == "" {
		return nil, nil
	}
	result, err := s.parseLicense(license)
	if err != nil {
		return nil, err
	}

	if _, err := s.store.UpsertSettingV2(ctx, &store.SetSettingMessage{
		Name:  api.SettingEnterpriseLicense,
		Value: license,
	}, api.SystemBotID); err != nil {
		return nil, errors.Wrapf(err, "failed to store the license")
	}

	s.RefreshCache(ctx)

	return result, nil
}

// loadLicense will load license and validate it.
func (s *LicenseService) loadLicense(ctx context.Context) *enterpriseAPI.License {
	license, err := s.findEnterpriseLicense(ctx)
	if err != nil {
		log.Debug("failed to load enterprise license", zap.Error(err))
	}
	if license == nil {
		license, err = s.findTrialingLicense(ctx)
		if err != nil {
			log.Debug("failed to load trialing license", zap.Error(err))
		}
	}

	if license == nil {
		license, err = s.fetchLicense(ctx)
		if err != nil {
			log.Debug("failed to fetch license", zap.Error(err))
		}
	}
	if license == nil {
		return nil
	}
	if err := license.Valid(); err != nil {
		log.Debug("license is invalid", zap.Error(err))
		return nil
	}

	return license
}

func (s *LicenseService) parseLicense(license string) (*enterpriseAPI.License, error) {
	claims := &Claims{}
	if err := parseJWTToken(license, s.config.Version, s.config.PublicKey, claims); err != nil {
		return nil, common.Wrap(err, common.Invalid)
	}

	return s.parseClaims(claims)
}

func (s *LicenseService) findEnterpriseLicense(ctx context.Context) (*enterpriseAPI.License, error) {
	// Find enterprise license.
	settingName := api.SettingEnterpriseLicense
	setting, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{
		Name: &settingName,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load enterprise license from settings")
	}
	if setting != nil && setting.Value != "" {
		license, err := s.parseLicense(setting.Value)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse enterprise license")
		}
		if license != nil {
			log.Debug(
				"Load valid license",
				zap.String("plan", license.Plan.String()),
				zap.Time("expiresAt", time.Unix(license.ExpiresTs, 0)),
				zap.Int("instanceCount", license.InstanceCount),
			)
			return license, nil
		}
	}

	return nil, nil
}

func (s *LicenseService) findTrialingLicense(ctx context.Context) (*enterpriseAPI.License, error) {
	settingName := api.SettingEnterpriseTrial
	setting, err := s.store.GetSettingV2(ctx, &store.FindSettingMessage{
		Name: &settingName,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load trial license from settings")
	}
	if setting != nil && setting.Value != "" {
		var data enterpriseAPI.License
		if err := json.Unmarshal([]byte(setting.Value), &data); err != nil {
			return nil, errors.Wrapf(err, "failed to parse trial license")
		}
		data.InstanceCount = enterpriseAPI.InstanceLimitForTrial
		if time.Now().AddDate(0, 0, -enterpriseAPI.TrialDaysLimit).Unix() >= setting.CreatedTs {
			return nil, nil
		}
		return &data, nil
	}

	return nil, nil
}

// parseClaims will valid and parse JWT claims to license instance.
func (s *LicenseService) parseClaims(claims *Claims) (*enterpriseAPI.License, error) {
	verifyIssuer := claims.VerifyIssuer(s.config.Issuer, true)
	if !verifyIssuer {
		return nil, common.Errorf(common.Invalid, "iss is not valid, expect %s but found '%v'", s.config.Issuer, claims.Issuer)
	}

	verifyAudience := claims.VerifyAudience(s.config.Audience, true)
	if !verifyAudience {
		return nil, common.Errorf(common.Invalid, "aud is not valid, expect %s but found '%v'", s.config.Audience, claims.Audience)
	}

	planType, err := convertPlanType(claims.Plan)
	if err != nil {
		return nil, common.Errorf(common.Invalid, "plan type %q is not valid", planType)
	}

	license := &enterpriseAPI.License{
		InstanceCount: claims.InstanceCount,
		ExpiresTs:     claims.ExpiresAt.Unix(),
		IssuedTs:      claims.IssuedAt.Unix(),
		Plan:          planType,
		Subject:       claims.Subject,
		Trialing:      claims.Trialing,
		OrgName:       claims.OrgName,
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
