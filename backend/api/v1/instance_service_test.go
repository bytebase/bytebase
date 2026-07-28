package v1

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/config"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

func TestValidateExtraConnectionParametersRejectsTiDBAllowAllFiles(t *testing.T) {
	err := validateExtraConnectionParameters(storepb.Engine_TIDB, map[string]string{
		"allowAllFiles": "true",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "allowAllFiles")
}

func TestClassifyConnectionFailure(t *testing.T) {
	connectErr := connect.NewError(connect.CodeInvalidArgument, errors.New("generic connect error"))
	connectErr.Meta().Set(connectionCategoryHeader, connectionCategoryAuthFailed)
	var typedNilConnectErr *connect.Error

	testCases := []struct {
		err  error
		want string
	}{
		{err: nil, want: connectionCategorySuccess},
		{err: typedNilConnectErr, want: connectionCategorySuccess},
		{err: connectErr, want: connectionCategoryAuthFailed},
		{err: errors.New("dial tcp 10.0.0.5:5432: i/o timeout"), want: connectionCategoryTimeout},
		{err: errors.New("password authentication failed for user bytebase"), want: connectionCategoryAuthFailed},
		{err: errors.New("permission denied for schema public"), want: connectionCategoryPermissionDenied},
		{err: errors.New("tls: failed to verify certificate: x509: certificate signed by unknown authority"), want: connectionCategorySSLTLSFailed},
		{err: errors.New("dial tcp 10.0.0.5:5432: connection refused"), want: connectionCategoryNetworkUnreachable},
		{err: errors.New("unsupported engine"), want: connectionCategoryUnsupportedEngine},
		{err: errors.New("driver returned an unexpected error"), want: connectionCategoryUnknown},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.want, classifyConnectionFailure(tc.err))
	}
}

func TestBuildInstanceConnectionLogAttrs(t *testing.T) {
	instance := &store.InstanceMessage{
		Metadata: &storepb.Instance{
			Engine: storepb.Engine_POSTGRES,
		},
	}
	dataSource := &storepb.DataSource{
		Type:                  storepb.DataSourceType_ADMIN,
		Host:                  "sensitive.example.com",
		Port:                  "5432",
		Username:              "bytebase",
		Password:              "secret",
		Database:              "prod",
		UseSsl:                true,
		SshHost:               "bastion.example.com",
		ObfuscatedSshPassword: "obfuscated",
		ExternalSecret:        &storepb.DataSourceExternalSecret{},
		AdditionalAddresses:   []*storepb.DataSource_Address{{Host: "replica.example.com", Port: "5432"}},
	}

	attrs := buildInstanceConnectionLogAttrs(v1connect.InstanceServiceCreateInstanceProcedure, connectionCategoryAuthFailed, instance, dataSource, 1500*time.Millisecond)
	got := make(map[string]any)
	for _, item := range attrs {
		attr, ok := item.(slog.Attr)
		require.True(t, ok)
		got[attr.Key] = attr.Value.Any()
	}

	require.Equal(t, map[string]any{
		"method":              v1connect.InstanceServiceCreateInstanceProcedure,
		"engine":              storepb.Engine_POSTGRES.String(),
		"data_source_type":    storepb.DataSourceType_ADMIN.String(),
		"category":            connectionCategoryAuthFailed,
		"elapsed_ms":          int64(1500),
		"has_ssl":             true,
		"has_ssh":             true,
		"has_external_secret": true,
	}, got)
	for _, key := range []string{"host", "port", "username", "database", "password", "dsn", "sql"} {
		require.NotContains(t, got, key)
	}
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
