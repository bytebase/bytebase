package config

import (
	"fmt"
	"io/ioutil"

	"go.uber.org/zap"
)

// Config is the API message for exterprise config.
type Config struct {
	// PublicKey is the key we used to decrypt license.
	PublicKey string
	// Version is the JWT key version.
	Version string
	// Issuer is the license issuer, it should always be "bytebase".
	Issuer string
	// MinimumInstance is the minimum instance count in each plan.
	MinimumInstance int
	// StorePath is the file path to store license.
	StorePath string
}

const (
	// keyID is the license key version.
	keyID = "v1"
	// issuer is the license issuer.
	issuer = "bytebase"
	// minimumInstance is the minimum instance count in subscribed plan.
	minimumInstance = 5
)

// NewConfig will create a new enterprise config instance.
func NewConfig(l *zap.Logger, dataDir string, mode string) (*Config, error) {
	l.Info("get project env", zap.String("env", mode))

	licensePubKey, err := ioutil.ReadFile(fmt.Sprintf("enterprise/keys/%s.pub.pem", mode))
	if err != nil {
		return nil, fmt.Errorf("cannot read license public key for env %s", mode)
	}

	storefile := "license"
	if mode != "release" {
		storefile = fmt.Sprintf("license_%s", mode)
	}

	return &Config{
		PublicKey:       string(licensePubKey),
		Version:         keyID,
		Issuer:          issuer,
		MinimumInstance: minimumInstance,
		StorePath:       fmt.Sprintf("file:%s/%s", dataDir, storefile),
	}, nil
}
