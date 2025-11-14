package secret

import (
	"context"

	"github.com/pkg/errors"

	vault "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/approle"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func getVaultClient(ctx context.Context, externalSecret *storepb.DataSourceExternalSecret) (*vault.Client, error) {
	config := vault.DefaultConfig()
	config.Address = externalSecret.Url

	// Configure TLS based on Vault-specific TLS settings
	tlsConfig := &vault.TLSConfig{
		// Use skip_vault_tls_verification from ExternalSecret
		// Default is false (verification enabled) for security
		Insecure: externalSecret.SkipVaultTlsVerification,
	}

	// If custom CA certificate is provided for Vault, use it
	if externalSecret.VaultSslCa != "" {
		tlsConfig.CACertBytes = []byte(externalSecret.VaultSslCa)
	}

	// Configure client certificate for Vault if provided
	if externalSecret.VaultSslCert != "" && externalSecret.VaultSslKey != "" {
		tlsConfig.ClientCert = externalSecret.VaultSslCert
		tlsConfig.ClientKey = externalSecret.VaultSslKey
	}

	if err := config.ConfigureTLS(tlsConfig); err != nil {
		return nil, errors.Wrapf(err, "failed to configure TLS")
	}

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to init vault client: %v", err.Error())
	}

	var token string

	switch externalSecret.AuthType {
	case storepb.DataSourceExternalSecret_TOKEN:
		token = externalSecret.GetToken()
	case storepb.DataSourceExternalSecret_VAULT_APP_ROLE:
		role := externalSecret.GetAppRole()
		if role == nil {
			return nil, errors.Errorf("approle is invalid")
		}
		appRoleSecret := &approle.SecretID{}
		switch role.Type {
		case storepb.DataSourceExternalSecret_AppRoleAuthOption_PLAIN:
			appRoleSecret.FromString = role.SecretId
		case storepb.DataSourceExternalSecret_AppRoleAuthOption_ENVIRONMENT:
			appRoleSecret.FromEnv = role.SecretId
		default:
			return nil, errors.Errorf("unsupported app role auth type: %v", role.Type)
		}

		opts := []approle.LoginOption{}
		if role.MountPath != "" {
			opts = append(opts, approle.WithMountPath(role.MountPath))
		}
		appRoleAuth, err := approle.NewAppRoleAuth(
			role.RoleId,
			appRoleSecret,
			opts...,
		)
		if err != nil {
			return nil, err
		}
		resp, err := client.Auth().Login(
			ctx,
			appRoleAuth,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to login with approle: %v", err.Error())
		}
		token = resp.Auth.ClientToken
	default:
		return nil, errors.Errorf("unsupported auth type: %v", externalSecret.AuthType)
	}

	client.SetToken(token)

	return client, nil
}

func getSecretFromVault(ctx context.Context, externalSecret *storepb.DataSourceExternalSecret) (string, error) {
	client, err := getVaultClient(ctx, externalSecret)
	if err != nil {
		return "", err
	}

	secret, err := client.KVv2(externalSecret.EngineName).Get(ctx, externalSecret.SecretName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get vault secret: %v", err.Error())
	}

	value, ok := secret.Data[externalSecret.PasswordKeyName].(string)
	if !ok {
		return "", errors.Errorf(`failed to get vault secret value for "%s/%s"`, externalSecret.SecretName, externalSecret.PasswordKeyName)
	}

	return value, nil
}
