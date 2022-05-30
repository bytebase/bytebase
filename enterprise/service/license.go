package service

import (
	"context"
	"fmt"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/enterprise/config"
	"github.com/bytebase/bytebase/store"
	"github.com/golang-jwt/jwt/v4"
)

// LicenseService is the service for enterprise license.
type LicenseService struct {
	config *config.Config
	store  *store.Store
}

// Claims creates a struct that will be encoded to a JWT.
// We add jwt.StandardClaims as an embedded type, to provide fields like name.
type Claims struct {
	InstanceCount int    `json:"instanceCount"`
	Trialing      bool   `json:"trialing"`
	Plan          string `json:"plan"`
	jwt.StandardClaims
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
func (s *LicenseService) StoreLicense(patch *enterpriseAPI.SubscriptionPatch) error {
	if patch.License != "" {
		if _, err := s.parseLicense(patch.License); err != nil {
			return nil
		}
	}
	return s.writeLicense(patch)
}

// LoadLicense will load license from file and validate it.
func (s *LicenseService) LoadLicense() (*enterpriseAPI.License, error) {
	tokenString, err := s.readLicense()
	if err != nil {
		return nil, err
	}
	if tokenString == "" {
		return nil, common.Errorf(common.NotFound, fmt.Errorf("cannot find license"))
	}

	return s.parseLicense(tokenString)
}

func (s *LicenseService) parseLicense(license string) (*enterpriseAPI.License, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(license, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, common.Errorf(common.Invalid, fmt.Errorf("unexpected signing method: %v", token.Header["alg"]))
		}

		kid, ok := token.Header["kid"].(string)
		if !ok || kid != s.config.Version {
			return nil, common.Errorf(common.Invalid, fmt.Errorf("version '%v' is not valid. expect %s", token.Header["kid"], s.config.Version))
		}

		key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(s.config.PublicKey))
		if err != nil {
			return nil, common.Errorf(common.Invalid, err)
		}

		return key, nil
	})
	if err != nil {
		return nil, common.Errorf(common.Invalid, err)
	}

	if !token.Valid {
		return nil, common.Errorf(common.Invalid, fmt.Errorf("invalid token"))
	}

	return s.parseClaims(claims)
}

// parseClaims will valid and parse JWT claims to license instance.
func (s *LicenseService) parseClaims(claims *Claims) (*enterpriseAPI.License, error) {
	err := claims.Valid()
	if err != nil {
		return nil, common.Errorf(common.Invalid, err)
	}

	verifyIssuer := claims.VerifyIssuer(s.config.Issuer, true)
	if !verifyIssuer {
		return nil, common.Errorf(common.Invalid, fmt.Errorf("iss is not valid, expect %s but found '%v'", s.config.Issuer, claims.Issuer))
	}

	verifyAudience := claims.VerifyAudience(s.config.Audience, true)
	if !verifyAudience {
		return nil, common.Errorf(common.Invalid, fmt.Errorf("aud is not valid, expect %s but found '%v'", s.config.Audience, claims.Audience))
	}

	instanceCount := claims.InstanceCount
	if instanceCount < s.config.MinimumInstance {
		return nil, common.Errorf(common.Invalid, fmt.Errorf("license instance count '%v' is not valid, minimum instance requirement is %d", instanceCount, s.config.MinimumInstance))
	}

	planType, err := convertPlanType(claims.Plan)
	if err != nil {
		return nil, common.Errorf(common.Invalid, fmt.Errorf("plan type %q is not valid", planType))
	}

	license := &enterpriseAPI.License{
		InstanceCount: instanceCount,
		ExpiresTs:     claims.ExpiresAt,
		IssuedTs:      claims.IssuedAt,
		Plan:          planType,
		Subject:       claims.Subject,
		Trialing:      claims.Trialing,
	}

	if err := license.Valid(); err != nil {
		return nil, common.Errorf(common.Invalid, err)
	}

	return license, nil
}

func (s *LicenseService) readLicense() (string, error) {
	settingName := api.SettingEnterpriseLicense
	ctx := context.Background()
	settings, err := s.store.FindSetting(ctx, &api.SettingFind{
		Name: &settingName,
	})

	if len(settings) == 0 {
		return "", common.Errorf(
			common.NotFound,
			fmt.Errorf("cannot find license with error %w", err),
		)
	}

	return settings[0].Value, nil
}

func (s *LicenseService) writeLicense(patch *enterpriseAPI.SubscriptionPatch) error {
	ctx := context.Background()
	_, err := s.store.PatchSetting(ctx, &api.SettingPatch{
		UpdaterID: patch.UpdaterID,
		Name:      api.SettingEnterpriseLicense,
		Value:     patch.License,
	})
	return err
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
		return api.FREE, fmt.Errorf("cannot conver plan type %q", candidate)
	}
}
