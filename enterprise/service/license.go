package service

import (
	"fmt"
	"io/ioutil"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/enterprise/config"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

// licenseService is the service for enterprise license.
type licenseService struct {
	l      *zap.Logger
	config *config.Config
}

// NewLicenseService will create a new enterprise license service.
func NewLicenseService(l *zap.Logger, dataDir string, mode string) (*licenseService, error) {
	config, err := config.NewConfig(l, dataDir, mode)
	if err != nil {
		return nil, err
	}

	return &licenseService{
		config: config,
		l:      l,
	}, nil
}

// StoreLicense will store license into file.
func (s *licenseService) StoreLicense(tokenString string) error {
	return s.writeLicense(tokenString)
}

// LoadLicense will load license from file and validate it.
func (s *licenseService) LoadLicense() (*enterpriseAPI.License, error) {
	tokenString, err := s.readLicense()
	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, common.Errorf(common.LicenseInvalid, fmt.Errorf("unexpected signing method: %v", token.Header["alg"]))
		}

		kid, ok := token.Header["kid"].(string)
		if !ok || kid != s.config.Version {
			return nil, common.Errorf(common.LicenseInvalid, fmt.Errorf("version '%v' is not valid. expect %s", token.Header["kid"], s.config.Version))
		}

		key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(s.config.PublicKey))
		if err != nil {
			return nil, common.Errorf(common.LicenseInvalid, err)
		}

		return key, nil
	})
	if err != nil {
		return nil, common.Errorf(common.LicenseInvalid, err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, common.Errorf(common.LicenseInvalid, fmt.Errorf("invalid token"))
	}

	return s.parseClaims(claims)
}

// parseClaims will valid and parse JWT claims to license instance.
func (s *licenseService) parseClaims(claims jwt.MapClaims) (*enterpriseAPI.License, error) {
	err := claims.Valid()
	if err != nil {
		return nil, common.Errorf(common.LicenseInvalid, err)
	}

	exp, ok := claims["exp"].(int64)
	if !ok {
		return nil, common.Errorf(common.LicenseInvalid, fmt.Errorf("exp is not valid, found '%v'", claims["exp"]))
	}

	iss, ok := claims["iss"].(string)
	if !ok || iss != s.config.Issuer {
		return nil, common.Errorf(common.LicenseInvalid, fmt.Errorf("iss is not valid, expect %s but found '%v'", s.config.Issuer, claims["iss"]))
	}

	instance, ok := claims["instance"].(int)
	if !ok || instance < s.config.MinimumInstance {
		return nil, common.Errorf(common.LicenseInvalid, fmt.Errorf("license instance count '%v' is not valid, minimum instance requirement is %d", claims["instance"], s.config.MinimumInstance))
	}

	plan, ok := claims["plan"].(string)
	if !ok {
		return nil, common.Errorf(common.LicenseInvalid, fmt.Errorf("plan is not valid, found '%v'", claims["plan"]))
	}

	planType, err := convertPlanType(plan)
	if err != nil {
		return nil, common.Errorf(common.LicenseInvalid, fmt.Errorf("plan type %q is not valid", planType))
	}

	aud, ok := claims["aud"].(string)
	if !ok || aud == "" {
		return nil, common.Errorf(common.LicenseInvalid, fmt.Errorf("aud is not valid, found '%v'", claims["aud"]))
	}

	license := &enterpriseAPI.License{
		InstanceCount: instance,
		ExpiresTs:     exp,
		Plan:          planType,
		Audience:      aud,
	}

	if err := license.Valid(); err != nil {
		return nil, common.Errorf(common.LicenseInvalid, err)
	}

	return license, nil
}

func (s *licenseService) readLicense() (string, error) {
	token, err := ioutil.ReadFile(s.config.StorePath)
	if err != nil {
		return "", common.Errorf(
			common.LicenseNotFound,
			fmt.Errorf("cannot read license from %s, error %w", s.config.StorePath, err),
		)
	}

	return string(token), nil
}

func (s *licenseService) writeLicense(token string) error {
	return ioutil.WriteFile(s.config.StorePath, []byte(token), 0644)
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
