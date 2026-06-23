package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

func TestValidateExtraConnectionParametersRejectsTiDBAllowAllFiles(t *testing.T) {
	err := validateExtraConnectionParameters(storepb.Engine_TIDB, map[string]string{
		"allowAllFiles": "true",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "allowAllFiles")
}

func TestValidateExternalSecretForSaaS(t *testing.T) {
	tokenSecret := func(tokenType storepb.DataSourceExternalSecret_TokenType) *storepb.DataSource {
		return &storepb.DataSource{
			ExternalSecret: &storepb.DataSourceExternalSecret{
				AuthType:   storepb.DataSourceExternalSecret_TOKEN,
				TokenType:  tokenType,
				AuthOption: &storepb.DataSourceExternalSecret_Token{Token: "x"},
			},
		}
	}
	appRoleSecret := func(secretType storepb.DataSourceExternalSecret_AppRoleAuthOption_SecretType) *storepb.DataSource {
		return &storepb.DataSource{
			ExternalSecret: &storepb.DataSourceExternalSecret{
				AuthType: storepb.DataSourceExternalSecret_VAULT_APP_ROLE,
				AuthOption: &storepb.DataSourceExternalSecret_AppRole{
					AppRole: &storepb.DataSourceExternalSecret_AppRoleAuthOption{
						RoleId:   "r",
						SecretId: "s",
						Type:     secretType,
					},
				},
			},
		}
	}

	testCases := []struct {
		name       string
		saas       bool
		dataSource *storepb.DataSource
		wantErr    bool
	}{
		{name: "non-saas allows file", saas: false, dataSource: tokenSecret(storepb.DataSourceExternalSecret_FILE), wantErr: false},
		{name: "non-saas allows env", saas: false, dataSource: tokenSecret(storepb.DataSourceExternalSecret_ENVIRONMENT), wantErr: false},
		{name: "saas allows plain", saas: true, dataSource: tokenSecret(storepb.DataSourceExternalSecret_PLAIN), wantErr: false},
		{name: "saas allows unspecified", saas: true, dataSource: tokenSecret(storepb.DataSourceExternalSecret_TOKEN_TYPE_UNSPECIFIED), wantErr: false},
		{name: "saas blocks file", saas: true, dataSource: tokenSecret(storepb.DataSourceExternalSecret_FILE), wantErr: true},
		{name: "saas blocks env", saas: true, dataSource: tokenSecret(storepb.DataSourceExternalSecret_ENVIRONMENT), wantErr: true},
		{name: "non-saas allows approle env", saas: false, dataSource: appRoleSecret(storepb.DataSourceExternalSecret_AppRoleAuthOption_ENVIRONMENT), wantErr: false},
		{name: "saas allows approle plain", saas: true, dataSource: appRoleSecret(storepb.DataSourceExternalSecret_AppRoleAuthOption_PLAIN), wantErr: false},
		{name: "saas blocks approle env", saas: true, dataSource: appRoleSecret(storepb.DataSourceExternalSecret_AppRoleAuthOption_ENVIRONMENT), wantErr: true},
		{name: "saas ignores no external secret", saas: true, dataSource: &storepb.DataSource{}, wantErr: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := &InstanceService{profile: &config.Profile{SaaS: tc.saas}}
			err := s.validateExternalSecretForSaaS(tc.dataSource)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
