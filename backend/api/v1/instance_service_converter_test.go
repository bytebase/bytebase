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

// krbDataSource builds a data source at the given destination, carrying a
// Kerberos config with the given keytab.
func krbDataSource(id, host, keytab string) *storepb.DataSource {
	return &storepb.DataSource{
		Id:   id,
		Host: host,
		Port: "10000",
		SaslConfig: &storepb.SASLConfig{
			Mechanism: &storepb.SASLConfig_KrbConfig{
				KrbConfig: &storepb.KerberosConfig{
					Primary: "hive",
					Realm:   "EXAMPLE.COM",
					Keytab:  []byte(keytab),
					KdcHost: "kdc.example.com",
					KdcPort: "88",
				},
			},
		},
	}
}

func TestRetainStoredKeytabOnEmptyUpdate(t *testing.T) {
	const storedHost = "hive.internal.example.com"
	stored := krbDataSource("admin", storedHost, "stored-keytab")

	// An empty keytab on update keeps the stored one; other fields still apply.
	updated := krbDataSource("admin", storedHost, "")
	updated.GetSaslConfig().GetKrbConfig().Realm = "NEW.COM"
	require.NoError(t, retainStoredKeytabOnEmptyUpdate(updated, stored))
	require.Equal(t, []byte("stored-keytab"), updated.GetSaslConfig().GetKrbConfig().Keytab)
	require.Equal(t, "NEW.COM", updated.GetSaslConfig().GetKrbConfig().Realm,
		"the principal a keytab claims is not where it is sent — it must stay updatable")

	// A newly uploaded keytab replaces the stored one.
	updated = krbDataSource("admin", storedHost, "new-keytab")
	require.NoError(t, retainStoredKeytabOnEmptyUpdate(updated, stored))
	require.Equal(t, []byte("new-keytab"), updated.GetSaslConfig().GetKrbConfig().Keytab)

	// Nothing stored: the empty keytab stays empty.
	updated = krbDataSource("admin", storedHost, "")
	require.NoError(t, retainStoredKeytabOnEmptyUpdate(updated, krbDataSource("admin", storedHost, "")))
	require.Empty(t, updated.GetSaslConfig().GetKrbConfig().Keytab)

	// Clearing the whole SASL config is a no-op for retention.
	require.NoError(t, retainStoredKeytabOnEmptyUpdate(&storepb.DataSource{Id: "admin"}, stored))
}

// TestRetainStoredKeytabRefusesANewDestination is the guard: a keytab is
// inherited to keep a read-modify-write client from wiping it, never to carry
// it somewhere the caller chose. Each case moves exactly one destination field
// and leaves the keytab out.
func TestRetainStoredKeytabRefusesANewDestination(t *testing.T) {
	const storedHost = "hive.internal.example.com"

	moved := map[string]func(ds *storepb.DataSource){
		"host": func(ds *storepb.DataSource) { ds.Host = "attacker.example.com" },
		"port": func(ds *storepb.DataSource) { ds.Port = "10001" },
		"additional_addresses": func(ds *storepb.DataSource) {
			ds.AdditionalAddresses = []*storepb.DataSource_Address{{Host: "attacker.example.com", Port: "10000"}}
		},
		"ssh_host": func(ds *storepb.DataSource) { ds.SshHost = "attacker.example.com" },
		"ssh_port": func(ds *storepb.DataSource) { ds.SshPort = "2222" },
		// Compared wholesale rather than by address-bearing key: the key
		// names differ per driver, and a name this code does not know would
		// be a hole. Over-refusing costs nothing — Hive, the only engine that
		// reads a keytab, does not read this map at all.
		"extra_connection_parameters": func(ds *storepb.DataSource) {
			ds.ExtraConnectionParameters = map[string]string{"host": "attacker.example.com"}
		},
		"sasl_config.kdc_host": func(ds *storepb.DataSource) {
			ds.GetSaslConfig().GetKrbConfig().KdcHost = "kdc.attacker.example.com"
		},
		"sasl_config.kdc_port": func(ds *storepb.DataSource) {
			ds.GetSaslConfig().GetKrbConfig().KdcPort = "8888"
		},
	}

	for field, move := range moved {
		t.Run(field+" moved, keytab omitted", func(t *testing.T) {
			stored := krbDataSource("admin", storedHost, "stored-keytab")
			updated := krbDataSource("admin", storedHost, "")
			move(updated)

			err := retainStoredKeytabOnEmptyUpdate(updated, stored)
			require.Error(t, err, "moving %s must not carry the stored keytab along", field)
			require.Contains(t, err.Error(), "keytab")
			require.Empty(t, updated.GetSaslConfig().GetKrbConfig().Keytab,
				"the keytab must not reach the new destination")
		})

		t.Run(field+" moved, keytab re-supplied", func(t *testing.T) {
			stored := krbDataSource("admin", storedHost, "stored-keytab")
			updated := krbDataSource("admin", storedHost, "re-supplied-keytab")
			move(updated)

			require.NoError(t, retainStoredKeytabOnEmptyUpdate(updated, stored),
				"re-supplying the keytab is what the guard asks for; %s may then move", field)
			require.Equal(t, []byte("re-supplied-keytab"), updated.GetSaslConfig().GetKrbConfig().Keytab)
		})
	}

	// Fields the caller cannot name an address through keep inheriting. Some
	// of them do move the peer — a replica set follows what the operator's own
	// server advertises, srv resolves the operator's own hostname through the
	// operator's DNS, cloud_sql_ip_type picks among the addresses Google
	// resolves for the instance already named — but none of them lets the
	// caller choose where it lands, and treating them as a move would fire the
	// guard on ordinary edits.
	kept := map[string]func(ds *storepb.DataSource){
		"username":                   func(ds *storepb.DataSource) { ds.Username = "someone-else" },
		"database":                   func(ds *storepb.DataSource) { ds.Database = "other" },
		"srv":                        func(ds *storepb.DataSource) { ds.Srv = true },
		"replica_set":                func(ds *storepb.DataSource) { ds.ReplicaSet = "rs1" },
		"direct_connection":          func(ds *storepb.DataSource) { ds.DirectConnection = true },
		"region":                     func(ds *storepb.DataSource) { ds.Region = "us-east-1" },
		"cloud_sql_ip_type":          func(ds *storepb.DataSource) { ds.CloudSqlIpType = storepb.DataSource_PRIVATE },
		"authentication_type":        func(ds *storepb.DataSource) { ds.AuthenticationType = storepb.DataSource_AWS_RDS_IAM },
		"sasl_config.realm":          func(ds *storepb.DataSource) { ds.GetSaslConfig().GetKrbConfig().Realm = "OTHER.COM" },
		"sasl_config.kdc_transport":  func(ds *storepb.DataSource) { ds.GetSaslConfig().GetKrbConfig().KdcTransportProtocol = "tcp" },
		"sasl_config.primary":        func(ds *storepb.DataSource) { ds.GetSaslConfig().GetKrbConfig().Primary = "impala" },
		"verify_tls_certificate":     func(ds *storepb.DataSource) { ds.VerifyTlsCertificate = true },
		"authentication_private_key": func(ds *storepb.DataSource) { ds.AuthenticationPrivateKey = "pem" },
	}

	for field, edit := range kept {
		t.Run(field+" edited, keytab still inherited", func(t *testing.T) {
			stored := krbDataSource("admin", storedHost, "stored-keytab")
			updated := krbDataSource("admin", storedHost, "")
			edit(updated)

			require.NoError(t, retainStoredKeytabOnEmptyUpdate(updated, stored))
			require.Equal(t, []byte("stored-keytab"), updated.GetSaslConfig().GetKrbConfig().Keytab,
				"editing %s connects nowhere new, so a read-modify-write client must not be forced to re-upload", field)
		})
	}
}

// TestDataSourceDestinationClassifiesEveryField forces a decision on every
// DataSource field. dataSourceDestination is a projection, so a field added
// later is silently treated as "not a destination" — and if that field carries
// an address, the keytab guard stops seeing the move. This fails until the new
// field is named in one list or the other, which is the point: the choice gets
// made by whoever adds it rather than defaulted.
func TestDataSourceDestinationClassifiesEveryField(t *testing.T) {
	// Fields the caller supplies an address through. Keep in step with
	// dataSourceDestination.
	inDestination := []string{
		"host", "port", "additional_addresses",
		"ssh_host", "ssh_port",
		"extra_connection_parameters",
		"sasl_config", // krb_config.kdc_host / kdc_port; see the projection
	}
	// Everything else, with the reason recorded beside it in
	// dataSourceDestination's doc comment.
	notDestination := []string{
		"id", "type", "username", "password", "obfuscated_password",
		"use_ssl", "verify_tls_certificate",
		"ssl_ca", "obfuscated_ssl_ca", "ssl_cert", "obfuscated_ssl_cert",
		"ssl_key", "obfuscated_ssl_key",
		"ssl_ca_path", "obfuscated_ssl_ca_path",
		"ssl_cert_path", "obfuscated_ssl_cert_path",
		"ssl_key_path", "obfuscated_ssl_key_path",
		"database", "srv", "authentication_database", "replica_set",
		"sid", "service_name",
		"ssh_user", "ssh_password", "obfuscated_ssh_password",
		"ssh_private_key", "obfuscated_ssh_private_key",
		"authentication_private_key", "obfuscated_authentication_private_key",
		"authentication_private_key_passphrase", "obfuscated_authentication_private_key_passphrase",
		"external_secret", "authentication_type", "cloud_sql_ip_type",
		"azure_credential", "aws_credential", "gcp_credential",
		"direct_connection", "region", "warehouse_id",
		"master_name", "master_username", "master_password", "obfuscated_master_password",
		"redis_type", "project_id", "instance_id",
	}

	classified := make(map[string]bool, len(inDestination)+len(notDestination))
	for _, name := range append(append([]string{}, inDestination...), notDestination...) {
		require.False(t, classified[name], "field %q listed twice", name)
		classified[name] = true
	}

	fields := (&storepb.DataSource{}).ProtoReflect().Descriptor().Fields()
	var unclassified []string
	for i := 0; i < fields.Len(); i++ {
		name := string(fields.Get(i).Name())
		if !classified[name] {
			unclassified = append(unclassified, name)
		}
		delete(classified, name)
	}
	require.Empty(t, unclassified,
		"new DataSource field(s) %v: decide whether a caller supplies an address through them. "+
			"If so, add them to dataSourceDestination — otherwise a keytab will follow them to a new host. "+
			"If not, record the reason in its doc comment and list them here.", unclassified)
	require.Empty(t, classified, "field(s) listed here no longer exist on DataSource: %v", classified)

	// sasl_config is the one field the projection picks apart rather than
	// copying whole, so the walk has to go a level down with it. Otherwise a
	// new address on KerberosConfig defaults to "not a destination" at exactly
	// the message holding the endpoint the keytab itself reaches, and this
	// test stays green because "sasl_config" is already listed above.
	krbClassified := map[string]bool{
		// In the projection.
		"kdc_host": true, "kdc_port": true,
		// Out: the principal kinit claims, the transport to the same KDC, and
		// the secret itself.
		"primary": true, "instance": true, "realm": true,
		"keytab": true, "kdc_transport_protocol": true,
	}
	krbFields := (&storepb.KerberosConfig{}).ProtoReflect().Descriptor().Fields()
	for i := 0; i < krbFields.Len(); i++ {
		name := string(krbFields.Get(i).Name())
		require.True(t, krbClassified[name],
			"new KerberosConfig field %q: decide whether it names an endpoint the keytab reaches, "+
				"and add it to dataSourceDestination if so", name)
		delete(krbClassified, name)
	}
	require.Empty(t, krbClassified, "field(s) listed here no longer exist on KerberosConfig: %v", krbClassified)

	// A second SASL mechanism would carry its own endpoint, and the projection
	// reads only krb_config.
	saslOneofs := (&storepb.SASLConfig{}).ProtoReflect().Descriptor().Oneofs()
	require.Equal(t, 1, saslOneofs.Len())
	require.Equal(t, 1, saslOneofs.Get(0).Fields().Len(),
		"a new SASL mechanism needs its endpoint fields added to dataSourceDestination")
}

func TestRetainStoredKeytabs(t *testing.T) {
	const storedHost = "hive.internal.example.com"
	stored := []*storepb.DataSource{
		krbDataSource("admin", storedHost, "admin-keytab"),
		krbDataSource("ro", storedHost, "ro-keytab"),
	}

	updated := []*storepb.DataSource{
		krbDataSource("admin", storedHost, ""),           // unchanged on read-modify-write
		krbDataSource("ro", storedHost, "new-ro-keytab"), // freshly uploaded
		krbDataSource("added", "new.example.com", ""),    // new data source, nothing stored
		{Id: "plain", Host: storedHost},                  // no SASL config at all
	}
	require.NoError(t, retainStoredKeytabs(updated, stored))

	require.Equal(t, []byte("admin-keytab"), updated[0].GetSaslConfig().GetKrbConfig().Keytab)
	require.Equal(t, []byte("new-ro-keytab"), updated[1].GetSaslConfig().GetKrbConfig().Keytab)
	require.Empty(t, updated[2].GetSaslConfig().GetKrbConfig().Keytab)
	require.Nil(t, updated[3].GetSaslConfig())

	// A full replacement that moves one data source is refused whole. The
	// wholesale rebuild is exactly where a read-modify-write client sends every
	// keytab back empty, so this is the path the inheritance was built for and
	// the path a retarget rides.
	moved := []*storepb.DataSource{
		krbDataSource("admin", storedHost, ""),
		krbDataSource("ro", "attacker.example.com", ""),
	}
	err := retainStoredKeytabs(moved, stored)
	require.Error(t, err)
	require.Contains(t, err.Error(), `"ro"`, "the error must name the data source the caller has to fix")
	require.Empty(t, moved[1].GetSaslConfig().GetKrbConfig().Keytab)
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
