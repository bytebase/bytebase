package secret

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func getSecretFromAzure(ctx context.Context, externalSecret *storepb.DataSourceExternalSecret) (string, error) {
	// Use default Azure credentials.
	// This supports:
	// - Environment variables (AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, AZURE_TENANT_ID)
	// - Managed Identity (when running in Azure)
	// - Azure CLI credentials
	// ref: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#DefaultAzureCredential
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get Azure credentials")
	}

	// The URL should be the Key Vault URL (e.g., https://myvault.vault.azure.net/)
	vaultURL := externalSecret.Url
	if vaultURL == "" {
		return "", errors.New("missing Azure Key Vault URL")
	}

	client, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create Azure Key Vault client")
	}

	// Get the secret using the secret name.
	// Empty version string means get the latest version.
	resp, err := client.GetSecret(ctx, externalSecret.SecretName, "", nil)
	if err != nil {
		if strings.Contains(err.Error(), "SecretNotFound") {
			return "", errors.Wrapf(err, "cannot find secret %s", externalSecret.SecretName)
		}
		return "", errors.Wrapf(err, "failed to get Azure Key Vault secret %s", externalSecret.SecretName)
	}

	if resp.Value == nil {
		return "", errors.Errorf("empty secret value for %s", externalSecret.SecretName)
	}

	return *resp.Value, nil
}
