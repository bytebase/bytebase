// Package service implements the enterprise license service.
package service

import (
	"context"
	"encoding/json"
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
	config *config.Config
	store  *store.Store

	cachedSubscription *enterpriseAPI.Subscription
}

// Claims creates a struct that will be encoded to a JWT.
// We add jwt.RegisteredClaims as an embedded type, to provide fields such as name.
type Claims struct {
	Seat          int    `json:"seat"`
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
		store:  store,
		config: config,
	}, nil
}

// StoreLicense will store license into file.
func (s *LicenseService) StoreLicense(ctx context.Context, patch *enterpriseAPI.SubscriptionPatch) error {
	if patch.License != "" {
		if _, err := s.parseLicense(patch.License); err != nil {
			return err
		}
	}
	if _, err := s.store.PatchSetting(ctx, &api.SettingPatch{
		UpdaterID: patch.UpdaterID,
		Name:      api.SettingEnterpriseLicense,
		Value:     patch.License,
	}); err != nil {
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

	license, err := s.loadLicense(ctx)
	if err != nil {
		log.Error("failed to load license", zap.Error(err))
	}
	if license == nil {
		return enterpriseAPI.Subscription{
			Plan: api.FREE,
			// -1 means not expire, just for free plan
			ExpiresTs:     -1,
			InstanceCount: config.MaximumInstanceForFreePlan,
			Seat:          config.MaximumSeatForFreePlan,
		}
	}

	seat := license.Seat
	if seat == 0 {
		seat = -1
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
		Seat:          seat,
	}
	return *s.cachedSubscription
}

// IsFeatureEnabled returns whether a feature is enabled.
func (s *LicenseService) IsFeatureEnabled(feature api.FeatureType) bool {
	return api.Feature(feature, s.GetEffectivePlan())
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
func (s *LicenseService) GetPlanLimitValue(name api.PlanLimit) int64 {
	v, ok := api.PlanLimitValues[name]
	if !ok {
		return 0
	}
	return v[s.GetEffectivePlan()]
}

// RefreshCache will invalidate and refresh the subscription cache.
func (s *LicenseService) RefreshCache(ctx context.Context) {
	s.cachedSubscription = nil
	s.LoadSubscription(ctx)
}

// loadLicense will load license and validate it.
<<<<<<< HEAD
func (s *LicenseService) loadLicense(ctx context.Context) (*enterpriseAPI.License, error) {
=======
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
>>>>>>> 82369aac6 (fix: refresh expired license (#5546))
	// Find enterprise license.
	settingName := api.SettingEnterpriseLicense
	settings, err := s.store.FindSetting(ctx, &api.SettingFind{
		Name: &settingName,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load enterprise license from settings")
	}
	tokenString := ""
	if len(settings) > 0 {
		tokenString = settings[0].Value
	}
	if tokenString != "" {
		license, err := s.parseLicense(tokenString)
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

	// Find free trial license.
	settingName = api.SettingEnterpriseTrial
	settings, err = s.store.FindSetting(ctx, &api.SettingFind{
		Name: &settingName,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load trial license from settings")
	}
	if len(settings) != 0 {
		var data enterpriseAPI.License
		if err := json.Unmarshal([]byte(settings[0].Value), &data); err != nil {
			return nil, errors.Wrapf(err, "failed to parse trial license")
		}
		return &data, nil
	}

	// No license or trial license found.
	return nil, nil
}

func (s *LicenseService) parseLicense(license string) (*enterpriseAPI.License, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(license, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, common.Errorf(common.Invalid, "unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok || kid != s.config.Version {
			return nil, common.Errorf(common.Invalid, "version '%v' is not valid. expect %s", token.Header["kid"], s.config.Version)
		}

		key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(s.config.PublicKey))
		if err != nil {
			return nil, common.Wrap(err, common.Invalid)
		}

		return key, nil
	})
	if err != nil {
		return nil, common.Wrap(err, common.Invalid)
	}

	if !token.Valid {
		return nil, common.Errorf(common.Invalid, "invalid token")
	}

	return s.parseClaims(claims)
}

// parseClaims will valid and parse JWT claims to license instance.
func (s *LicenseService) parseClaims(claims *Claims) (*enterpriseAPI.License, error) {
	err := claims.Valid()
	if err != nil {
		return nil, common.Wrap(err, common.Invalid)
	}

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
		Seat:          claims.Seat,
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
