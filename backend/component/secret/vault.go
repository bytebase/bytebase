package secret

import (
	"context"

	"github.com/pkg/errors"

	vault "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/approle"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func getVaultClient(ctx context.Context, externalSecret *storepb.DataSourceExternalSecret) (*vault.Client, error) {
	config := vault.DefaultConfig()
	config.Address = externalSecret.Url

	client, err := vault.NewClient(config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to init vault client: %v", err.Error())
	}

	var token string

	switch externalSecret.AuthType {
	case storepb.DataSourceExternalSecret_TOKEN:
		token = externalSecret.GetToken()
	case storepb.DataSourceExternalSecret_APP_ROLE:
		role := externalSecret.GetAppRole()
		if role == nil {
			return nil, errors.Errorf("app role is invalid")
		}
		appRoleSecret := &approle.SecretID{}
		switch role.Type {
		case storepb.DataSourceExternalSecret_AppRoleAuthOption_PLAIN:
			appRoleSecret.FromString = role.SecretId
		case storepb.DataSourceExternalSecret_AppRoleAuthOption_ENVIRONMENT:
			appRoleSecret.FromEnv = role.SecretId
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
			return nil, errors.Wrapf(err, "failed to login with app role: %v", err.Error())
		}
		token = resp.Auth.ClientToken
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
		return "", errors.Errorf(`failed to get secret value for "%s/%s"`, externalSecret.SecretName, externalSecret.PasswordKeyName)
	}

	return value, nil
}
