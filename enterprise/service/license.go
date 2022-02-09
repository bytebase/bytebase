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

// LicenseService is the service for enterprise license.
type LicenseService struct {
	l      *zap.Logger
	config *config.Config
}

// NewLicenseService will create a new enterprise license service.
func NewLicenseService(l *zap.Logger, dataDir string, mode string) (*LicenseService, error) {
	config, err := config.NewConfig(l, dataDir, mode)
	if err != nil {
		return nil, err
	}

	return &LicenseService{
		config: config,
		l:      l,
	}, nil
}

// StoreLicense will store license into file.
func (s *LicenseService) StoreLicense(tokenString string) error {
	return s.writeLicense(tokenString)
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

// VerifyLicense will check if license is valid
func (s *LicenseService) VerifyLicense(license string) error {
	_, err := s.parseLicense(license)
	return err
}

func (s *LicenseService) parseLicense(license string) (*enterpriseAPI.License, error) {
	token, err := jwt.Parse(license, func(token *jwt.Token) (interface{}, error) {
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

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, common.Errorf(common.Invalid, fmt.Errorf("invalid token"))
	}

	return s.parseClaims(claims)
}

// parseClaims will valid and parse JWT claims to license instance.
func (s *LicenseService) parseClaims(claims jwt.MapClaims) (*enterpriseAPI.License, error) {
	err := claims.Valid()
	if err != nil {
		return nil, common.Errorf(common.Invalid, err)
	}

	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, common.Errorf(common.Invalid, fmt.Errorf("exp is not valid, found '%v'", claims["exp"]))
	}

	verifyIssuer := claims.VerifyIssuer(s.config.Issuer, true)
	if !verifyIssuer {
		return nil, common.Errorf(common.Invalid, fmt.Errorf("iss is not valid, expect %s but found '%v'", s.config.Issuer, claims["iss"]))
	}

	instance, ok := claims["instance"].(float64)
	if !ok || int(instance) < s.config.MinimumInstance {
		return nil, common.Errorf(common.Invalid, fmt.Errorf("license instance count '%v' is not valid, minimum instance requirement is %d", claims["instance"], s.config.MinimumInstance))
	}

	plan, ok := claims["plan"].(string)
	if !ok {
		return nil, common.Errorf(common.Invalid, fmt.Errorf("plan is not valid, found '%v'", claims["plan"]))
	}

	planType, err := convertPlanType(plan)
	if err != nil {
		return nil, common.Errorf(common.Invalid, fmt.Errorf("plan type %q is not valid", planType))
	}

	aud, ok := claims["aud"].(string)
	if !ok || aud == "" {
		return nil, common.Errorf(common.Invalid, fmt.Errorf("aud is not valid, found '%v'", claims["aud"]))
	}

	license := &enterpriseAPI.License{
		InstanceCount: int(instance),
		ExpiresTs:     int64(exp),
		Plan:          planType,
		Audience:      aud,
	}

	if err := license.Valid(); err != nil {
		return nil, common.Errorf(common.Invalid, err)
	}

	return license, nil
}

func (s *LicenseService) readLicense() (string, error) {
	token, err := ioutil.ReadFile(s.config.StorePath)
	if err != nil {
		return "", common.Errorf(
			common.NotFound,
			fmt.Errorf("cannot read license from %s, error %w", s.config.StorePath, err),
		)
	}

	return string(token), nil
}

func (s *LicenseService) writeLicense(token string) error {
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
