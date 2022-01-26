package service

import (
	"fmt"
	"io/ioutil"

	"github.com/bytebase/bytebase/api"
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

// ParseLicense will valid and parse license from file.
func (s *licenseService) ParseLicense() (*enterpriseAPI.License, error) {
	tokenString, err := s.readLicense()
	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok || kid != s.config.Version {
			return nil, fmt.Errorf("version '%v' is not valid. expect %s", token.Header["kid"], s.config.Version)
		}

		key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(s.config.PublicKey))
		if err != nil {
			return nil, err
		}

		return key, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return s.parseClaims(claims)
}

// parseClaims will valid and parse JWT claims to license instance.
func (s *licenseService) parseClaims(claims jwt.MapClaims) (*enterpriseAPI.License, error) {
	err := claims.Valid()
	if err != nil {
		return nil, err
	}

	exp, ok := claims["exp"].(int64)
	if !ok {
		return nil, fmt.Errorf("exp is not valid, found '%v'", claims["exp"])
	}

	iss, ok := claims["iss"].(string)
	if !ok || iss != s.config.Issuer {
		return nil, fmt.Errorf("iss is not valid, expect %s but found '%v'", s.config.Issuer, claims["iss"])
	}

	instance, ok := claims["instance"].(int)
	if !ok || instance < s.config.MinimumInstance {
		return nil, fmt.Errorf("license instance count '%v' is not valid, minimum instance requirement is %d", claims["instance"], s.config.MinimumInstance)
	}

	plan, ok := claims["plan"].(string)
	if !ok {
		return nil, fmt.Errorf("plan is not valid, found '%v'", claims["plan"])
	}

	aud, ok := claims["aud"].(string)
	if !ok || aud == "" {
		return nil, fmt.Errorf("aud is not valid, found '%v'", claims["aud"])
	}

	license := &enterpriseAPI.License{
		InstanceCount: instance,
		ExpiresTs:     exp,
		Plan:          convertPlanType(plan),
		Audience:      aud,
	}

	if err := license.Valid(); err != nil {
		return nil, err
	}

	return license, nil
}

func (s *licenseService) readLicense() (string, error) {
	token, err := ioutil.ReadFile(s.config.StorePath)
	if err != nil {
		return "", fmt.Errorf("cannot read license from %s, error %w", s.config.StorePath, err)
	}

	return string(token), nil
}

func (s *licenseService) writeLicense(token string) error {
	s.l.Info("Write license to", zap.String("path", s.config.StorePath), zap.String("token", token))
	return ioutil.WriteFile(s.config.StorePath, []byte(token), 0644)
}

func convertPlanType(candidate string) api.PlanType {
	switch candidate {
	case api.TEAM.String():
		return api.TEAM
	case api.ENTERPRISE.String():
		return api.ENTERPRISE
	default:
		return api.FREE
	}
}
