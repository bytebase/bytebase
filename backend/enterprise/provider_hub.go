//go:build !aws

package enterprise

import (
	"github.com/bytebase/bytebase/backend/enterprise/plugin"
	"github.com/bytebase/bytebase/backend/enterprise/plugin/hub"
)

func getLicenseProvider(providerConfig *plugin.ProviderConfig) (plugin.LicenseProvider, error) {
	return hub.NewProvider(providerConfig)
}
