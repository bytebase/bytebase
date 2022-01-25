package service

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/bytebase/bytebase/api"
	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/enterprise/config"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

type licenseService struct {
	l    *zap.Logger
	conf *config.Conf
}

var validPlans = []string{
	api.TEAM.String(),
	api.ENTERPRISE.String(),
}

func NewLicenseService(l *zap.Logger, dataDir string, mode string) (*licenseService, error) {
	conf, err := config.NewConf(l, dataDir, mode)
	if err != nil {
		return nil, err
	}

	return &licenseService{
		conf: conf,
		l:    l,
	}, nil
}

func (s *licenseService) StoreLicense(tokenString string) error {
	return s.writeLicense(tokenString)
}

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
		if !ok || kid != s.conf.Version {
			return nil, fmt.Errorf("version '%v' is not valid. expect %s", token.Header["kid"], s.conf.Version)
		}

		key, err := jwt.ParseRSAPublicKeyFromPEM([]byte(s.conf.PubKey))
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

func (s *licenseService) parseClaims(claims jwt.MapClaims) (*enterpriseAPI.License, error) {
	err := claims.Valid()
	if err != nil {
		return nil, err
	}

	exp, ok := claims["exp"].(int64)
	if !ok {
		return nil, fmt.Errorf("exp is not valid, found '%v'", claims["exp"])
	}
	if exp <= time.Now().Unix()/1000 {
		return nil, fmt.Errorf("license has expired at %v", time.Unix(exp, 0))
	}

	iss, ok := claims["iss"].(string)
	if !ok || iss != s.conf.Iss {
		return nil, fmt.Errorf("iss is not valid, expect %s but found '%v'", s.conf.Iss, claims["iss"])
	}

	instance, ok := claims["instance"].(int)
	if !ok || instance < s.conf.MinimumInstance {
		return nil, fmt.Errorf("instance '%v' is not valid, minimum instance count is %d", claims["instance"], s.conf.MinimumInstance)
	}

	plan, ok := claims["plan"].(string)
	if !ok {
		return nil, fmt.Errorf("plan is not valid, found '%v'", claims["plan"])
	}
	if err := validPlanType(plan); err != nil {
		return nil, err
	}

	aud, ok := claims["aud"].(string)
	if !ok || aud == "" {
		return nil, fmt.Errorf("aud is not valid, found '%v'", claims["aud"])
	}

	return &enterpriseAPI.License{
		InstanceCount: instance,
		ExpiresAt:     exp,
		Plan:          plan,
		Audience:      aud,
	}, nil
}

func (s *licenseService) readLicense() (string, error) {
	token, err := ioutil.ReadFile(s.conf.StorePath)
	if err != nil {
		return "", fmt.Errorf("cannot read license from %s, error %w", s.conf.StorePath, err)
	}

	return string(token), nil
}

func (s *licenseService) writeLicense(token string) error {
	return ioutil.WriteFile(s.conf.StorePath, []byte(token), 0644)
}

func validPlanType(candidate string) error {
	for _, plan := range validPlans {
		if plan == candidate {
			return nil
		}
	}

	return fmt.Errorf("plan '%s' is not valid, expect one of %s",
		candidate,
		strings.Join(validPlans, ", "),
	)
}
