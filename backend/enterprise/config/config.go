// Package config provides configuration interface for enterprise licensing.
package config

import (
	"embed"
	"fmt"
	"io/fs"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

//go:embed keys
var keysFS embed.FS

// Config is the API message for enterprise config.
type Config struct {
	// PublicKey is the key we used to decrypt license.
	PublicKey string
	// Version is the JWT key version.
	Version string
	// Issuer is the license issuer, it should always be "bytebase".
	Issuer string
	// Audience is the license audience, it should always be "bb.license".
	Audience string
	// Mode can be "prod" or "dev"
	Mode common.ReleaseMode
}

const (
	// keyID is the license key version.
	keyID = "v1"
	// issuer is the license issuer.
	issuer = "bytebase"
	// audience is the license token audience.
	audience = "bb.license"
	// MaximumInstanceForFreePlan is the maximum instance limit for the FREE plan.
	MaximumInstanceForFreePlan = -1
)

// NewConfig will create a new enterprise config instance.
func NewConfig(mode common.ReleaseMode) (*Config, error) {
	licensePubKey, err := fs.ReadFile(keysFS, fmt.Sprintf("keys/%s.pub.pem", mode))
	if err != nil {
		return nil, errors.Errorf("cannot read license public key for env %s", mode)
	}

	return &Config{
		PublicKey: string(licensePubKey),
		Version:   keyID,
		Issuer:    issuer,
		Audience:  audience,
		Mode:      mode,
	}, nil
}
