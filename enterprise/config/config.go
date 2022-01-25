package config

import (
	"fmt"
	"io/ioutil"

	"go.uber.org/zap"
)

type Config struct {
	// The key we used to decrypt license
	PublicKey string
	// JWT key version
	Version string
	// Should always be "bytebase"
	Issuer string
	// Minimum instance count
	MinimumInstance int
	// File path to store license
	StorePath string
}

const (
	keyID           = "v1"
	issuer          = "bytebase"
	minimumInstance = 5
)

// Create a new enterprise config instance
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
