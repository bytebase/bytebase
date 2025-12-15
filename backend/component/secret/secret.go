// Package secret includes the component of getting secrets from external sources.
package secret

import (
	"context"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// ReplaceExternalSecret replaces the secret with external secret.
func ReplaceExternalSecret(ctx context.Context, secret string, externalSecret *storepb.DataSourceExternalSecret) (string, error) {
	if externalSecret != nil {
		// TODO: consider cache?
		switch externalSecret.SecretType {
		case storepb.DataSourceExternalSecret_AWS_SECRETS_MANAGER:
			return getSecretFromAWS(ctx, externalSecret)
		case storepb.DataSourceExternalSecret_VAULT_KV_V2:
			return getSecretFromVault(ctx, externalSecret)
		case storepb.DataSourceExternalSecret_GCP_SECRET_MANAGER:
			return getSecretFromGCP(ctx, externalSecret)
		case storepb.DataSourceExternalSecret_AZURE_KEY_VAULT:
			return getSecretFromAzure(ctx, externalSecret)
		default:
			return "", errors.Errorf("unsupported secret type: %v", externalSecret.SecretType)
		}
	}

	return secret, nil
}
