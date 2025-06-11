//go:build aws

package enterprise

import (
	"github.com/bytebase/bytebase/backend/enterprise/plugin"
	"github.com/bytebase/bytebase/backend/enterprise/plugin/aws"
)

func getLicenseProvider(providerConfig *plugin.ProviderConfig) (plugin.LicenseProvider, error) {
	return aws.NewProvider(providerConfig)
}

