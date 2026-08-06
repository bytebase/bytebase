package v1

import (
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/store"
)

func TestConvertToV1InstanceUsesOwningProjectName(t *testing.T) {
	projectID := "project-a"
	instance := &store.InstanceMessage{
		ResourceID: "instance-a",
		ProjectID:  &projectID,
		Metadata: &storepb.Instance{
			Roles: []*storepb.InstanceRole{{Name: "role-a"}},
		},
	}

	got := convertToV1Instance(instance, false)
	require.Equal(t, "projects/project-a/instances/instance-a", got.Name)
	require.Equal(t, "projects/project-a/instances/instance-a/roles/role-a", got.Roles[0].Name)

	resource := convertToV1InstanceResource(instance, false)
	require.Equal(t, "projects/project-a/instances/instance-a", resource.Name)
}

func TestConvertDataSourceCloudSQLIPType(t *testing.T) {
	tests := []struct {
		name  string
		v1    v1pb.DataSource_CloudSQLIPType
		store storepb.DataSource_CloudSQLIPType
	}{
		{"unspecified", v1pb.DataSource_CLOUD_SQL_IP_TYPE_UNSPECIFIED, storepb.DataSource_CLOUD_SQL_IP_TYPE_UNSPECIFIED},
		{"public", v1pb.DataSource_PUBLIC, storepb.DataSource_PUBLIC},
		{"private", v1pb.DataSource_PRIVATE, storepb.DataSource_PRIVATE},
		{"psc", v1pb.DataSource_PSC, storepb.DataSource_PSC},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// v1 -> store
			storeDS, err := convertV1DataSource(&v1pb.DataSource{
				Type:               v1pb.DataSourceType_ADMIN,
				AuthenticationType: v1pb.DataSource_GOOGLE_CLOUD_SQL_IAM,
				CloudSqlIpType:     tc.v1,
			})
			require.NoError(t, err)
			require.Equal(t, tc.store, storeDS.GetCloudSqlIpType(), "v1->store")

			// store -> v1 round-trip preserves the value
			v1DSs := convertDataSources([]*storepb.DataSource{storeDS})
			require.Len(t, v1DSs, 1)
			require.Equal(t, tc.v1, v1DSs[0].GetCloudSqlIpType(), "store->v1 round-trip")
		})
	}
}

func TestNormalizeGCPDataSources(t *testing.T) {
	tests := []struct {
		name           string
		engine         storepb.Engine
		in             *storepb.DataSource
		wantProjectID  string
		wantInstanceID string
		wantHost       string
	}{
		{
			name:           "spanner legacy host is split into project and instance IDs",
			engine:         storepb.Engine_SPANNER,
			in:             &storepb.DataSource{Host: "projects/my-proj/instances/my-inst"},
			wantProjectID:  "my-proj",
			wantInstanceID: "my-inst",
			wantHost:       "",
		},
		{
			name:   "spanner endpoint host is kept as endpoint",
			engine: storepb.Engine_SPANNER,
			in: &storepb.DataSource{
				ProjectId:  "my-proj",
				InstanceId: "my-inst",
				Host:       "spanner-nonprod.p.googleapis.com",
			},
			wantProjectID:  "my-proj",
			wantInstanceID: "my-inst",
			wantHost:       "spanner-nonprod.p.googleapis.com",
		},
		{
			name:           "spanner new-style without host is untouched",
			engine:         storepb.Engine_SPANNER,
			in:             &storepb.DataSource{ProjectId: "my-proj", InstanceId: "my-inst"},
			wantProjectID:  "my-proj",
			wantInstanceID: "my-inst",
			wantHost:       "",
		},
		{
			name:          "bigquery legacy host becomes project ID",
			engine:        storepb.Engine_BIGQUERY,
			in:            &storepb.DataSource{Host: "my-proj"},
			wantProjectID: "my-proj",
			wantHost:      "",
		},
		{
			name:   "bigquery host is kept as endpoint when project ID is set",
			engine: storepb.Engine_BIGQUERY,
			in: &storepb.DataSource{
				ProjectId: "my-proj",
				Host:      "bigquery-nonprod.p.googleapis.com",
			},
			wantProjectID: "my-proj",
			wantHost:      "bigquery-nonprod.p.googleapis.com",
		},
		{
			name:     "non-GCP engine is untouched",
			engine:   storepb.Engine_POSTGRES,
			in:       &storepb.DataSource{Host: "projects/my-proj/instances/my-inst"},
			wantHost: "projects/my-proj/instances/my-inst",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			normalizeGCPDataSources(tc.engine, []*storepb.DataSource{tc.in})
			require.Equal(t, tc.wantProjectID, tc.in.GetProjectId())
			require.Equal(t, tc.wantInstanceID, tc.in.GetInstanceId())
			require.Equal(t, tc.wantHost, tc.in.GetHost())
		})
	}
}

func TestConvertDataSourceGCPFields(t *testing.T) {
	storeDS, err := convertV1DataSource(&v1pb.DataSource{
		Type:       v1pb.DataSourceType_ADMIN,
		ProjectId:  "my-proj",
		InstanceId: "my-inst",
		Host:       "spanner-nonprod.p.googleapis.com",
		Port:       "443",
	})
	require.NoError(t, err)
	require.Equal(t, "my-proj", storeDS.GetProjectId(), "v1->store project_id")
	require.Equal(t, "my-inst", storeDS.GetInstanceId(), "v1->store instance_id")

	v1DSs := convertDataSources([]*storepb.DataSource{storeDS})
	require.Len(t, v1DSs, 1)
	require.Equal(t, "my-proj", v1DSs[0].GetProjectId(), "store->v1 project_id")
	require.Equal(t, "my-inst", v1DSs[0].GetInstanceId(), "store->v1 instance_id")
	require.Equal(t, "spanner-nonprod.p.googleapis.com", v1DSs[0].GetHost(), "store->v1 host")
	require.Equal(t, "443", v1DSs[0].GetPort(), "store->v1 port")
}

func TestConvertDataSourceSaslConfigNeverReturnsKeytab(t *testing.T) {
	got := convertDataSourceSaslConfig(&storepb.SASLConfig{
		Mechanism: &storepb.SASLConfig_KrbConfig{
			KrbConfig: &storepb.KerberosConfig{
				Primary:              "hive",
				Instance:             "node1",
				Realm:                "EXAMPLE.COM",
				Keytab:               []byte("secret-keytab"),
				KdcHost:              "kdc.example.com",
				KdcPort:              "88",
				KdcTransportProtocol: "tcp",
			},
		},
	})

	krb := got.GetKrbConfig()
	require.NotNil(t, krb)
	require.Empty(t, krb.Keytab)
	require.Equal(t, "hive", krb.Primary)
	require.Equal(t, "node1", krb.Instance)
	require.Equal(t, "EXAMPLE.COM", krb.Realm)
	require.Equal(t, "kdc.example.com", krb.KdcHost)
	require.Equal(t, "88", krb.KdcPort)
	require.Equal(t, "tcp", krb.KdcTransportProtocol)
}

func TestRetainStoredKeytabOnEmptyUpdate(t *testing.T) {
	stored := &storepb.SASLConfig{
		Mechanism: &storepb.SASLConfig_KrbConfig{
			KrbConfig: &storepb.KerberosConfig{Keytab: []byte("stored-keytab")},
		},
	}

	// An empty keytab on update keeps the stored one; other fields still apply.
	updated := &storepb.SASLConfig{
		Mechanism: &storepb.SASLConfig_KrbConfig{
			KrbConfig: &storepb.KerberosConfig{Realm: "NEW.COM"},
		},
	}
	retainStoredKeytabOnEmptyUpdate(updated, stored)
	require.Equal(t, []byte("stored-keytab"), updated.GetKrbConfig().Keytab)
	require.Equal(t, "NEW.COM", updated.GetKrbConfig().Realm)

	// A newly uploaded keytab replaces the stored one.
	updated = &storepb.SASLConfig{
		Mechanism: &storepb.SASLConfig_KrbConfig{
			KrbConfig: &storepb.KerberosConfig{Keytab: []byte("new-keytab")},
		},
	}
	retainStoredKeytabOnEmptyUpdate(updated, stored)
	require.Equal(t, []byte("new-keytab"), updated.GetKrbConfig().Keytab)

	// Nothing stored: the empty keytab stays empty.
	updated = &storepb.SASLConfig{
		Mechanism: &storepb.SASLConfig_KrbConfig{
			KrbConfig: &storepb.KerberosConfig{},
		},
	}
	retainStoredKeytabOnEmptyUpdate(updated, nil)
	require.Empty(t, updated.GetKrbConfig().Keytab)

	// Clearing the whole SASL config is a no-op for retention.
	retainStoredKeytabOnEmptyUpdate(nil, stored)
}

func TestRetainStoredKeytabs(t *testing.T) {
	krbDS := func(id string, keytab []byte) *storepb.DataSource {
		return &storepb.DataSource{
			Id: id,
			SaslConfig: &storepb.SASLConfig{
				Mechanism: &storepb.SASLConfig_KrbConfig{
					KrbConfig: &storepb.KerberosConfig{Keytab: keytab},
				},
			},
		}
	}
	stored := []*storepb.DataSource{
		krbDS("admin", []byte("admin-keytab")),
		krbDS("ro", []byte("ro-keytab")),
	}

	updated := []*storepb.DataSource{
		krbDS("admin", nil),                  // unchanged on read-modify-write
		krbDS("ro", []byte("new-ro-keytab")), // freshly uploaded
		krbDS("added", nil),                  // new data source, nothing stored
		{Id: "plain"},                        // no SASL config at all
	}
	retainStoredKeytabs(updated, stored)

	require.Equal(t, []byte("admin-keytab"), updated[0].GetSaslConfig().GetKrbConfig().Keytab)
	require.Equal(t, []byte("new-ro-keytab"), updated[1].GetSaslConfig().GetKrbConfig().Keytab)
	require.Empty(t, updated[2].GetSaslConfig().GetKrbConfig().Keytab)
	require.Nil(t, updated[3].GetSaslConfig())
}

// assertNoInputOnlyValues walks every populated message field and requires
// that fields annotated INPUT_ONLY carry no value. It pins the read-path
// contract: whatever the proto declares write-only must be blanked by the
// store->v1 converters.
func assertNoInputOnlyValues(t *testing.T, m protoreflect.Message, path string) {
	fields := m.Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		fieldPath := fmt.Sprintf("%s.%s", path, fd.Name())
		if opts, ok := fd.Options().(*descriptorpb.FieldOptions); ok && opts != nil {
			behaviors, ok := proto.GetExtension(opts, annotations.E_FieldBehavior).([]annotations.FieldBehavior)
			if ok && slices.Contains(behaviors, annotations.FieldBehavior_INPUT_ONLY) {
				// Oneof members may stay present as an is-configured signal
				// (e.g. the Vault token), so require blank content, not
				// absence.
				if m.Has(fd) {
					blank := false
					switch {
					case fd.IsList():
						blank = m.Get(fd).List().Len() == 0
					case fd.Kind() == protoreflect.BytesKind:
						blank = len(m.Get(fd).Bytes()) == 0
					case fd.Kind() == protoreflect.StringKind:
						blank = m.Get(fd).String() == ""
					default:
					}
					require.True(t, blank, "INPUT_ONLY field %s must be blank on reads", fieldPath)
				}
				continue
			}
		}
		if fd.IsMap() {
			continue
		}
		if fd.Kind() != protoreflect.MessageKind {
			continue
		}
		if fd.IsList() {
			list := m.Get(fd).List()
			for j := 0; j < list.Len(); j++ {
				assertNoInputOnlyValues(t, list.Get(j).Message(), fmt.Sprintf("%s[%d]", fieldPath, j))
			}
			continue
		}
		if m.Has(fd) {
			assertNoInputOnlyValues(t, m.Get(fd).Message(), fieldPath)
		}
	}
}

func TestConvertDataSourcesBlanksEveryInputOnlyField(t *testing.T) {
	// One store data source per authentication/secret shape, each with every
	// secret populated, so the INPUT_ONLY walk covers all converter branches.
	dataSources := []*storepb.DataSource{
		{
			Id:                                 "password-auth",
			Type:                               storepb.DataSourceType_ADMIN,
			Username:                           "admin",
			Password:                           "password",
			SslCa:                              "ca",
			SslCert:                            "cert",
			SslKey:                             "key",
			SslCaPath:                          "/ca",
			SslCertPath:                        "/cert",
			SslKeyPath:                         "/key",
			SshPassword:                        "ssh-password",
			SshPrivateKey:                      "ssh-private-key",
			AuthenticationPrivateKey:           "auth-private-key",
			AuthenticationPrivateKeyPassphrase: "passphrase",
			MasterPassword:                     "master-password",
			SaslConfig: &storepb.SASLConfig{
				Mechanism: &storepb.SASLConfig_KrbConfig{
					KrbConfig: &storepb.KerberosConfig{
						Primary: "hive",
						Realm:   "EXAMPLE.COM",
						Keytab:  []byte("keytab"),
					},
				},
			},
		},
		{
			Id:   "vault-token",
			Type: storepb.DataSourceType_READ_ONLY,
			ExternalSecret: &storepb.DataSourceExternalSecret{
				SecretType:   storepb.DataSourceExternalSecret_VAULT_KV_V2,
				Url:          "https://vault.example.com",
				AuthType:     storepb.DataSourceExternalSecret_TOKEN,
				VaultSslCa:   "vault-ca",
				VaultSslCert: "vault-cert",
				VaultSslKey:  "vault-key",
				AuthOption: &storepb.DataSourceExternalSecret_Token{
					Token: "vault-token",
				},
			},
		},
		{
			Id:   "vault-app-role",
			Type: storepb.DataSourceType_READ_ONLY,
			ExternalSecret: &storepb.DataSourceExternalSecret{
				SecretType: storepb.DataSourceExternalSecret_VAULT_KV_V2,
				Url:        "https://vault.example.com",
				AuthType:   storepb.DataSourceExternalSecret_VAULT_APP_ROLE,
				AuthOption: &storepb.DataSourceExternalSecret_AppRole{
					AppRole: &storepb.DataSourceExternalSecret_AppRoleAuthOption{
						RoleId:   "role-id",
						SecretId: "secret-id",
					},
				},
			},
		},
		{
			Id:                 "aws-iam",
			Type:               storepb.DataSourceType_ADMIN,
			AuthenticationType: storepb.DataSource_AWS_RDS_IAM,
			IamExtension: &storepb.DataSource_AwsCredential{
				AwsCredential: &storepb.DataSource_AWSCredential{
					AccessKeyId:     "AKIA123",
					SecretAccessKey: "secret-access-key",
					SessionToken:    "session-token",
					RoleArn:         "arn:aws:iam::123456789012:role/bytebase",
					ExternalId:      "external-id",
				},
			},
		},
		{
			Id:                 "azure-iam",
			Type:               storepb.DataSourceType_ADMIN,
			AuthenticationType: storepb.DataSource_AZURE_IAM,
			IamExtension: &storepb.DataSource_AzureCredential_{
				AzureCredential: &storepb.DataSource_AzureCredential{
					TenantId:     "tenant",
					ClientId:     "client",
					ClientSecret: "client-secret",
				},
			},
		},
		{
			Id:                 "gcp-iam",
			Type:               storepb.DataSourceType_ADMIN,
			AuthenticationType: storepb.DataSource_GOOGLE_CLOUD_SQL_IAM,
			IamExtension: &storepb.DataSource_GcpCredential{
				GcpCredential: &storepb.DataSource_GCPCredential{
					Content: "service-account-json",
				},
			},
		},
	}

	got := convertDataSources(dataSources)
	require.Len(t, got, len(dataSources))
	for _, ds := range got {
		assertNoInputOnlyValues(t, ds.ProtoReflect(), fmt.Sprintf("data_source(%s)", ds.GetId()))
	}

	// The AWS credential keeps signaling presence without content.
	var awsDS *v1pb.DataSource
	for _, ds := range got {
		if ds.GetId() == "aws-iam" {
			awsDS = ds
		}
	}
	require.NotNil(t, awsDS)
	require.NotNil(t, awsDS.GetAwsCredential())
}
