//go:build !release
// +build !release

package config

import (
	"github.com/bytebase/bytebase/backend/common"
)

// NewConfig will create a new enterprise config instance.
func NewConfig(mode common.ReleaseMode) (*Config, error) {
	config, err := getConfig(mode)
	if err != nil {
		return nil, err
	}

	config.HubAPIURL = "https://bytebase-hub-backend-dev.onrender.com"
	return config, nil
}
