package service

import (
	"fmt"
	"io/ioutil"
	"time"

	enterpriseAPI "github.com/bytebase/bytebase/enterprise/api"
	"github.com/bytebase/bytebase/enterprise/config"
	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

type licenseService struct {
	l    *zap.Logger
	conf *config.Conf
}

func NewLicenseService(l *zap.Logger, dataDir string) (*licenseService, error) {
	conf, err := config.NewConf(dataDir)
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
			return nil, fmt.Errorf("not valid version")
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
	if !ok || exp <= time.Now().Unix()/1000 {
		return nil, fmt.Errorf("not valid exp")
	}

	iss, ok := claims["iss"].(string)
	if !ok || iss != s.conf.Iss {
		return nil, fmt.Errorf("not valid iss")
	}

	instance, ok := claims["instance"].(int)
	if !ok || instance < s.conf.MinimumInstance {
		return nil, fmt.Errorf("not valid instance count")
	}

	plan, ok := claims["plan"].(string)
	if !ok || plan == "" {
		return nil, fmt.Errorf("not valid plan")
	}

	aud, ok := claims["aud"].(string)
	if !ok || aud == "" {
		return nil, fmt.Errorf("not valid aud")
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
		return "", fmt.Errorf("cannnot read token in %s", s.conf.StorePath)
	}

	return string(token), nil
}

func (s *licenseService) writeLicense(token string) error {
	return ioutil.WriteFile(s.conf.StorePath, []byte(token), 0644)
}
