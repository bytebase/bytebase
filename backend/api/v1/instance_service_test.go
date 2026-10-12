package v1

import (
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	"github.com/bytebase/bytebase/backend/component/config"
	"github.com/bytebase/bytebase/backend/component/sample"
	"github.com/bytebase/bytebase/backend/enterprise"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/store"
)

func TestSampleInstanceLifecycleHooksWrapMetadataUpdate(t *testing.T) {
	t.Parallel()
	ctx, stores, projectID, instanceID, _ := setupProjectInstanceLifecycleAPITest(t)
	_, err := stores.UpdateInstance(ctx, &store.UpdateInstanceMessage{
		ResourceID: &instanceID,
		Workspace:  "default",
		Metadata: &storepb.Instance{
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	manager := &sampleManagerStub{}
	manager.validate = func(ctx context.Context, workspaceID, gotInstanceID string) error {
		require.Equal(t, "default", workspaceID)
		require.Equal(t, instanceID, gotInstanceID)
		instance, err := stores.GetInstance(ctx, &store.FindInstanceMessage{Workspace: workspaceID, ResourceID: &gotInstanceID, ShowDeleted: true})
		require.NoError(t, err)
		require.True(t, instance.Deleted)
		return nil
	}
	manager.lifecycle = func(ctx context.Context, workspaceID, gotInstanceID string, deleted bool) error {
		instance, err := stores.GetInstance(ctx, &store.FindInstanceMessage{Workspace: workspaceID, ResourceID: &gotInstanceID, ShowDeleted: true})
		require.NoError(t, err)
		require.Equal(t, deleted, instance.Deleted)
		return nil
	}
	service := &InstanceService{
		store:          stores,
		licenseService: newInstanceServiceTestLicenseService(t, stores),
		sampleManager:  manager,
	}
	instanceName := common.FormatProjectInstance(projectID, instanceID)

	_, err = service.DeleteInstance(ctx, connect.NewRequest(&v1pb.DeleteInstanceRequest{Name: instanceName}))
	require.NoError(t, err)
	require.Equal(t, []bool{true}, manager.lifecycleCalls)

	_, err = service.UndeleteInstance(ctx, connect.NewRequest(&v1pb.UndeleteInstanceRequest{Name: instanceName}))
	require.NoError(t, err)
	require.Equal(t, 1, manager.validateCalls)
	require.Equal(t, []bool{true, false}, manager.lifecycleCalls)
}

func TestSampleProjectInstanceConnectErrorMapsUnknownFailure(t *testing.T) {
	t.Parallel()
	err := sampleProjectInstanceConnectError(errors.New("unexpected manager failure"))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}

type sampleManagerStub struct {
	instance            *store.InstanceMessage
	prepareErr          error
	prepareRequests     []sample.PrepareRequest
	checkAvailableErr   error
	checkAvailableCalls int
	listInstancesErr    error
	projectPurge        func(context.Context, string, string) error
	projectPurgeCalls   []string
	validate            func(context.Context, string, string) error
	lifecycle           func(context.Context, string, string, bool) error
	validateCalls       int
	lifecycleCalls      []bool
}

func (s *sampleManagerStub) CheckAvailable(context.Context) error {
	s.checkAvailableCalls++
	return s.checkAvailableErr
}

func (s *sampleManagerStub) ListInstances(context.Context, string) ([]*sample.Instance, error) {
	return nil, s.listInstancesErr
}

func (s *sampleManagerStub) HandleProjectPurge(ctx context.Context, workspaceID, projectID string) error {
	s.projectPurgeCalls = append(s.projectPurgeCalls, projectID)
	if s.projectPurge != nil {
		return s.projectPurge(ctx, workspaceID, projectID)
	}
	return nil
}

func (*sampleManagerStub) Start(context.Context, string) error { return nil }

func (*sampleManagerStub) Cleanup(context.Context) error { return nil }

func (*sampleManagerStub) Stop() {}

func (s *sampleManagerStub) PrepareSampleProjectInstance(_ context.Context, request sample.PrepareRequest) (*store.InstanceMessage, error) {
	s.prepareRequests = append(s.prepareRequests, request)
	return s.instance, s.prepareErr
}

func (s *sampleManagerStub) ValidateInstanceRestore(ctx context.Context, workspaceID, instanceID string) error {
	s.validateCalls++
	if s.validate != nil {
		return s.validate(ctx, workspaceID, instanceID)
	}
	return nil
}

func (s *sampleManagerStub) HandleInstanceLifecycle(ctx context.Context, workspaceID, instanceID string, deleted bool) error {
	s.lifecycleCalls = append(s.lifecycleCalls, deleted)
	if s.lifecycle != nil {
		return s.lifecycle(ctx, workspaceID, instanceID, deleted)
	}
	return nil
}

func TestValidateIAMCredentialForSaaS(t *testing.T) {
	saas := &InstanceService{profile: &config.Profile{SaaS: true}}
	selfHosted := &InstanceService{profile: &config.Profile{SaaS: false}}

	awsDS := func(credential *storepb.DataSource_AWSCredential) *storepb.DataSource {
		ds := &storepb.DataSource{AuthenticationType: storepb.DataSource_AWS_RDS_IAM}
		if credential != nil {
			ds.IamExtension = &storepb.DataSource_AwsCredential{AwsCredential: credential}
		}
		return ds
	}

	testCases := []struct {
		name    string
		service *InstanceService
		ds      *storepb.DataSource
		wantErr bool
	}{
		{
			name:    "saas rejects nil iam extension",
			service: saas,
			ds:      awsDS(nil),
			wantErr: true,
		},
		{
			// An all-empty credential behaves exactly like the default chain
			// at connect time: the host's own AWS identity.
			name:    "saas rejects empty aws credential",
			service: saas,
			ds:      awsDS(&storepb.DataSource_AWSCredential{}),
			wantErr: true,
		},
		{
			name:    "saas accepts aws access key",
			service: saas,
			ds:      awsDS(&storepb.DataSource_AWSCredential{AccessKeyId: "AKIAIOSFODNN7EXAMPLE", SecretAccessKey: "secret"}),
			wantErr: false,
		},
		{
			// Role-only is the legitimate SaaS cross-account pattern: the
			// tenant's role trusts the host identity as the base principal.
			name:    "saas accepts aws role arn without keys",
			service: saas,
			ds:      awsDS(&storepb.DataSource_AWSCredential{RoleArn: "arn:aws:iam::123456789012:role/tenant"}),
			wantErr: false,
		},
		{
			name:    "saas rejects empty gcp credential",
			service: saas,
			ds: &storepb.DataSource{
				AuthenticationType: storepb.DataSource_GOOGLE_CLOUD_SQL_IAM,
				IamExtension:       &storepb.DataSource_GcpCredential{GcpCredential: &storepb.DataSource_GCPCredential{}},
			},
			wantErr: true,
		},
		{
			name:    "saas accepts gcp credential content",
			service: saas,
			ds: &storepb.DataSource{
				AuthenticationType: storepb.DataSource_GOOGLE_CLOUD_SQL_IAM,
				IamExtension:       &storepb.DataSource_GcpCredential{GcpCredential: &storepb.DataSource_GCPCredential{Content: `{"type":"service_account"}`}},
			},
			wantErr: false,
		},
		{
			name:    "saas rejects incomplete azure credential",
			service: saas,
			ds: &storepb.DataSource{
				AuthenticationType: storepb.DataSource_AZURE_IAM,
				IamExtension:       &storepb.DataSource_AzureCredential_{AzureCredential: &storepb.DataSource_AzureCredential{TenantId: "tenant"}},
			},
			wantErr: true,
		},
		{
			name:    "saas accepts complete azure credential",
			service: saas,
			ds: &storepb.DataSource{
				AuthenticationType: storepb.DataSource_AZURE_IAM,
				IamExtension:       &storepb.DataSource_AzureCredential_{AzureCredential: &storepb.DataSource_AzureCredential{TenantId: "tenant", ClientId: "client", ClientSecret: "secret"}},
			},
			wantErr: false,
		},
		{
			name:    "saas ignores password authentication",
			service: saas,
			ds:      &storepb.DataSource{AuthenticationType: storepb.DataSource_PASSWORD},
			wantErr: false,
		},
		{
			// Self-hosted deployments may use the default credential chain.
			name:    "self-hosted accepts empty aws credential",
			service: selfHosted,
			ds:      awsDS(&storepb.DataSource_AWSCredential{}),
			wantErr: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.service.validateIAMCredentialForSaaS(tc.ds)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateExtraConnectionParametersRejectsTiDBAllowAllFiles(t *testing.T) {
	t.Parallel()
	err := validateExtraConnectionParameters(storepb.Engine_TIDB, map[string]string{
		"allowAllFiles": "true",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "allowAllFiles")
}

func TestValidateProjectInstanceListFilter(t *testing.T) {
	t.Parallel()
	projectID := "project-a"
	tests := []struct {
		name    string
		parent  *string
		filter  string
		wantErr bool
	}{
		{name: "project parent accepts omitted filter", parent: &projectID},
		{name: "project parent accepts matching project filter", parent: &projectID, filter: `project == "projects/project-a"`},
		{name: "project parent accepts matching project filter in conjunction", parent: &projectID, filter: `project == "projects/project-a" && engine == "POSTGRES"`},
		{name: "project parent rejects another project filter", parent: &projectID, filter: `project == "projects/project-b"`, wantErr: true},
		{name: "project parent rejects reversed another project filter", parent: &projectID, filter: `"projects/project-b" == project`, wantErr: true},
		{name: "project parent rejects another project in disjunction", parent: &projectID, filter: `project == "projects/project-a" || project == "projects/project-b"`, wantErr: true},
		{name: "project parent rejects another project under negation", parent: &projectID, filter: `!(project == "projects/project-b")`, wantErr: true},
		{name: "workspace parent retains project filter behavior", filter: `project == "projects/project-b"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := validateProjectInstanceListFilter(test.parent, test.filter)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestClassifyConnectionFailure(t *testing.T) {
	t.Parallel()
	connectErr := connect.NewError(connect.CodeInvalidArgument, errors.New("generic connect error"))
	connectErr.Meta().Set(connectionCategoryHeader, connectionCategoryAuthFailed)
	var typedNilConnectErr *connect.Error

	testCases := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: connectionCategorySuccess},
		{name: "typed nil connect error", err: typedNilConnectErr, want: connectionCategorySuccess},
		{name: "category already on the connect error", err: connectErr, want: connectionCategoryAuthFailed},

		{name: "dial timeout", err: errors.New("dial tcp 203.0.113.5:5432: i/o timeout"), want: connectionCategoryTimeout},
		{name: "context deadline", err: errors.New("failed to connect: context deadline exceeded"), want: connectionCategoryTimeout},
		{name: "context canceled", err: errors.New("failed to connect to database: context canceled"), want: connectionCategoryTimeout},
		{name: "operation canceled", err: errors.New("dial tcp 203.0.113.5:3306: operation was canceled"), want: connectionCategoryTimeout},

		{name: "PostgreSQL password", err: errors.New("failed SASL auth: FATAL: password authentication failed for user \"bytebase\" (SQLSTATE 28P01)"), want: connectionCategoryAuthFailed},
		{name: "MariaDB access denied", err: errors.New("Error 1045 (28000): Access denied for user 'bytebase'@'203.0.113.9' (using password: YES)"), want: connectionCategoryAuthFailed},
		{name: "SQL Server logon trigger", err: errors.New("mssql: login error: Logon failed for login 'bytebase' due to trigger execution"), want: connectionCategoryAuthFailed},
		{name: "SQL Server login", err: errors.New("mssql: login error: Login failed for user 'bytebase'"), want: connectionCategoryAuthFailed},
		{name: "MongoDB handshake", err: errors.New("failed to ping MongoDB: connection() error occurred during connection handshake: auth error: sasl conversation error: unable to authenticate using mechanism \"SCRAM-SHA-256\""), want: connectionCategoryAuthFailed},

		{name: "PostgreSQL unknown role after a plaintext fallback", err: errors.New("failed to connect to `user=bytebase database=postgres`:\n\t203.0.113.5:5432 (db.example.com): tls error: server refused TLS connection\n\t203.0.113.5:5432 (db.example.com): server error: FATAL: role \"bytebase\" does not exist (SQLSTATE 28000)"), want: connectionCategoryAuthFailed},
		{name: "PostgreSQL unknown database after a plaintext fallback", err: errors.New("failed to connect to `user=bytebase database=app`:\n\t203.0.113.5:5432 (db.example.com): tls error: server refused TLS connection\n\t203.0.113.5:5432 (db.example.com): server error: FATAL: database \"app\" does not exist (SQLSTATE 3D000)"), want: connectionCategoryUnknown},

		{name: "permission denied", err: errors.New("permission denied for schema public"), want: connectionCategoryPermissionDenied},

		{name: "unknown certificate authority", err: errors.New("tls: failed to verify certificate: x509: certificate signed by unknown authority"), want: connectionCategorySSLTLSFailed},
		{name: "server without TLS", err: errors.New("pq: SSL is not enabled on the server"), want: connectionCategorySSLTLSFailed},

		{name: "lookup failure", err: errors.New("dial tcp: lookup db.corp.example: no such host"), want: connectionCategoryHostNotFound},
		{name: "hostname resolving error", err: errors.New("failed to connect to `user=bytebase database=postgres`: hostname resolving error: lookup db.corp.example: no such host"), want: connectionCategoryHostNotFound},
		{name: "libpq translate host", err: errors.New("could not translate host name \"db.corp.example\" to address"), want: connectionCategoryHostNotFound},

		{name: "connection refused", err: errors.New("dial tcp 203.0.113.5:5432: connect: connection refused"), want: connectionCategoryNetworkUnreachable},
		{name: "no route", err: errors.New("dial tcp 203.0.113.5:5432: connect: no route to host"), want: connectionCategoryNetworkUnreachable},
		{name: "network keyword", err: errors.New("a network-related error occurred"), want: connectionCategoryNetworkUnreachable},

		{name: "unsupported engine", err: errors.New("unsupported engine"), want: connectionCategoryUnsupportedEngine},
		{name: "endpoint that is not the selected engine", err: errors.New("failed to list DynamoDB tables: operation error DynamoDB: ListTables, https response error StatusCode: 200, RequestID: , deserialization failed, failed to decode response body, invalid character '<' looking for beginning of value"), want: connectionCategoryUnsupportedEngine},
		{name: "unrecognized", err: errors.New("driver returned an unexpected error"), want: connectionCategoryUnknown},

		// A driver that retries until its deadline reports the deadline and
		// the last cause in one message; the cause is what the user fixes.
		{name: "timeout wrapping a lookup failure", err: errors.New("server selection error: context deadline exceeded, current topology: { Type: Unknown, Servers: [{ Addr: db.corp.example:27017, Type: Unknown, Last error: dial tcp: lookup db.corp.example: no such host }, ] }"), want: connectionCategoryHostNotFound},
		{name: "timeout wrapping a refusal", err: errors.New("server selection error: server selection timeout, current topology: { Type: Unknown, Servers: [{ Addr: 203.0.113.5:27017, Type: Unknown, Last error: dial tcp 203.0.113.5:27017: connect: connection refused }, ] }"), want: connectionCategoryNetworkUnreachable},
		{name: "timeout wrapping a certificate failure", err: errors.New("server selection error: context deadline exceeded, current topology: { Last error: connection() error occurred during connection handshake: x509: certificate signed by unknown authority }"), want: connectionCategorySSLTLSFailed},
		{name: "timeout wrapping an auth failure", err: errors.New("server selection error: context deadline exceeded, current topology: { Last error: connection() error occurred during connection handshake: auth error: sasl conversation error }"), want: connectionCategoryAuthFailed},
		{name: "timeout wrapping a dial timeout", err: errors.New("server selection error: context deadline exceeded, current topology: { Last error: dial tcp 203.0.113.5:27017: i/o timeout }"), want: connectionCategoryTimeout},
		{name: "TLS handshake that never completes", err: errors.New("net/http: TLS handshake timeout"), want: connectionCategoryTimeout},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, classifyConnectionFailure(tc.err))
		})
	}
}

func TestIsPrivateOrLocalHost(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		engine     storepb.Engine
		dataSource *storepb.DataSource
		want       bool
	}{
		{name: "10/8", dataSource: &storepb.DataSource{Host: "10.0.0.5"}, want: true},
		{name: "172.16/12 lower bound", dataSource: &storepb.DataSource{Host: "172.16.0.1"}, want: true},
		{name: "172.16/12 upper bound", dataSource: &storepb.DataSource{Host: "172.31.255.254"}, want: true},
		{name: "just outside 172.16/12", dataSource: &storepb.DataSource{Host: "172.32.0.1"}, want: false},
		{name: "192.168/16", dataSource: &storepb.DataSource{Host: "192.168.1.10"}, want: true},
		{name: "loopback", dataSource: &storepb.DataSource{Host: "127.1.2.3"}, want: true},
		{name: "unspecified", dataSource: &storepb.DataSource{Host: "0.0.0.0"}, want: true},
		{name: "link-local", dataSource: &storepb.DataSource{Host: "169.254.169.254"}, want: true},
		{name: "carrier-grade NAT", dataSource: &storepb.DataSource{Host: "100.64.0.7"}, want: true},
		{name: "just outside carrier-grade NAT", dataSource: &storepb.DataSource{Host: "100.128.0.7"}, want: false},
		{name: "IPv6 loopback", dataSource: &storepb.DataSource{Host: "::1"}, want: true},
		{name: "bracketed IPv6 loopback", dataSource: &storepb.DataSource{Host: "[::1]"}, want: true},
		{name: "IPv6 unique local", dataSource: &storepb.DataSource{Host: "fd12:3456:789a::1"}, want: true},
		{name: "IPv6 link-local", dataSource: &storepb.DataSource{Host: "fe80::1"}, want: true},
		{name: "IPv4-mapped private", dataSource: &storepb.DataSource{Host: "::ffff:10.0.0.5"}, want: true},
		{name: "public IPv4", dataSource: &storepb.DataSource{Host: "203.0.113.5"}, want: false},
		{name: "public IPv6", dataSource: &storepb.DataSource{Host: "2001:db8::5"}, want: false},
		{name: "private IP with a port", dataSource: &storepb.DataSource{Host: "10.0.0.5:5432"}, want: true},
		{name: "private IP in an endpoint URL", engine: storepb.Engine_ELASTICSEARCH, dataSource: &storepb.DataSource{Host: "https://10.0.0.5:9200"}, want: true},
		{name: "surrounding spaces and case", dataSource: &storepb.DataSource{Host: "  LocalHost "}, want: true},

		{name: "localhost", dataSource: &storepb.DataSource{Host: "localhost"}, want: true},
		{name: "Docker host alias", dataSource: &storepb.DataSource{Host: "host.docker.internal"}, want: true},
		{name: "single-label name", dataSource: &storepb.DataSource{Host: "postgres"}, want: true},
		{name: "reserved .internal suffix", dataSource: &storepb.DataSource{Host: "ip-10-0-0-5.ec2.internal"}, want: true},
		{name: "reserved .local suffix", dataSource: &storepb.DataSource{Host: "db.local"}, want: true},
		{name: "trailing dot", dataSource: &storepb.DataSource{Host: "db.local."}, want: true},
		{name: "public hostname", dataSource: &storepb.DataSource{Host: "db.example.com"}, want: false},
		{name: "public hostname that only contains a reserved word", dataSource: &storepb.DataSource{Host: "internal.example.com"}, want: false},
		{name: "socket path", dataSource: &storepb.DataSource{Host: "/var/run/postgresql"}, want: false},
		{name: "empty host", dataSource: &storepb.DataSource{}, want: false},

		{name: "SSH tunnel reaches a private IP", dataSource: &storepb.DataSource{Host: "10.0.0.5", SshHost: "bastion.example.com"}, want: false},
		{name: "SSH tunnel reaches a local name", dataSource: &storepb.DataSource{Host: "postgres", SshHost: "bastion.example.com"}, want: false},

		// The host is an account locator or a connection name here, not a
		// name Bytebase resolves.
		{name: "Snowflake account locator", engine: storepb.Engine_SNOWFLAKE, dataSource: &storepb.DataSource{Host: "xy12345"}, want: false},
		{name: "Cloud SQL connection name", dataSource: &storepb.DataSource{Host: "project:region:instance", AuthenticationType: storepb.DataSource_GOOGLE_CLOUD_SQL_IAM}, want: false},
		{name: "BigQuery without an endpoint", engine: storepb.Engine_BIGQUERY, dataSource: &storepb.DataSource{}, want: false},
		{name: "DynamoDB local endpoint IP", engine: storepb.Engine_DYNAMODB, dataSource: &storepb.DataSource{Host: "http://127.0.0.1:8000"}, want: true},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			engine := tc.engine
			if engine == storepb.Engine_ENGINE_UNSPECIFIED {
				engine = storepb.Engine_POSTGRES
			}
			require.Equal(t, tc.want, isPrivateOrLocalHost(engine, tc.dataSource))
		})
	}
}

func TestConnectionTestRefusesPrivateHostOnCloud(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	private := &storepb.DataSource{Host: "10.0.0.5", Port: "5432"}
	instance := &store.InstanceMessage{Metadata: &storepb.Instance{Engine: storepb.Engine_POSTGRES}}
	lookupOf := func(addrs ...string) func(context.Context, string) ([]netip.Addr, error) {
		return func(context.Context, string) ([]netip.Addr, error) {
			var resolved []netip.Addr
			for _, addr := range addrs {
				resolved = append(resolved, netip.MustParseAddr(addr))
			}
			return resolved, nil
		}
	}

	t.Run("Cloud answers without dialing", func(t *testing.T) {
		t.Parallel()
		// No dbFactory: reaching the dial would panic.
		service := &InstanceService{profile: &config.Profile{SaaS: true}}
		err := service.checkAndLogInstanceConnection(ctx, "test", instance, private)
		require.NotNil(t, err)
		require.Equal(t, connect.CodeInvalidArgument, err.Code())
		require.Equal(t, connectionCategoryPrivateAddress, err.Meta().Get(connectionCategoryHeader))
		require.Equal(t, `"10.0.0.5" is a private or local address that Bytebase Cloud cannot reach`, err.Message())
	})

	t.Run("self-hosted dials private hosts", func(t *testing.T) {
		t.Parallel()
		service := &InstanceService{profile: &config.Profile{}, lookupHost: lookupOf("10.0.0.5")}
		require.Empty(t, service.privateHostReason(ctx, storepb.Engine_POSTGRES, private))
		require.Empty(t, service.privateHostReason(ctx, storepb.Engine_POSTGRES, &storepb.DataSource{Host: "db.example.com"}))
	})

	t.Run("Cloud dials its own sample database", func(t *testing.T) {
		t.Parallel()
		service := &InstanceService{profile: &config.Profile{
			SaaS:                       true,
			SampleProjectInstancePgURL: "postgres://sample:secret@10.0.0.5:5432/postgres",
		}}
		require.Empty(t, service.privateHostReason(ctx, storepb.Engine_POSTGRES, private))
		require.NotEmpty(t, service.privateHostReason(ctx, storepb.Engine_POSTGRES, &storepb.DataSource{Host: "10.0.0.6"}))
	})

	t.Run("a hostname that resolves only to private addresses", func(t *testing.T) {
		t.Parallel()
		service := &InstanceService{profile: &config.Profile{SaaS: true}, lookupHost: lookupOf("10.1.2.3", "fd12::1")}
		require.Equal(t,
			`"db.example.com" resolves to a private address that Bytebase Cloud cannot reach`,
			service.privateHostReason(ctx, storepb.Engine_POSTGRES, &storepb.DataSource{Host: "db.example.com"}))
	})

	t.Run("a hostname with one public address is dialed", func(t *testing.T) {
		t.Parallel()
		service := &InstanceService{profile: &config.Profile{SaaS: true}, lookupHost: lookupOf("10.1.2.3", "203.0.113.5")}
		require.Empty(t, service.privateHostReason(ctx, storepb.Engine_POSTGRES, &storepb.DataSource{Host: "db.example.com"}))
	})

	t.Run("a lookup that fails leaves the reason to the driver", func(t *testing.T) {
		t.Parallel()
		service := &InstanceService{profile: &config.Profile{SaaS: true}, lookupHost: func(context.Context, string) ([]netip.Addr, error) {
			return nil, errors.New("no such host")
		}}
		require.Empty(t, service.privateHostReason(ctx, storepb.Engine_POSTGRES, &storepb.DataSource{Host: "db.example.com"}))
	})

	t.Run("names Bytebase does not resolve itself are not looked up", func(t *testing.T) {
		t.Parallel()
		service := &InstanceService{profile: &config.Profile{SaaS: true}, lookupHost: func(context.Context, string) ([]netip.Addr, error) {
			t.Error("unexpected lookup")
			return nil, nil
		}}
		for name, dataSource := range map[string]*storepb.DataSource{
			"through an SSH tunnel":   {Host: "db.example.com", SshHost: "bastion.example.com"},
			"SRV record":              {Host: "cluster.example.com", Srv: true},
			"public IP literal":       {Host: "203.0.113.5"},
			"several hosts":           {Host: "a.example.com,b.example.com"},
			"Cloud SQL connection":    {Host: "project:region:instance", AuthenticationType: storepb.DataSource_GOOGLE_CLOUD_SQL_IAM},
			"host with nothing in it": {},
		} {
			require.Empty(t, service.privateHostReason(ctx, storepb.Engine_MONGODB, dataSource), name)
		}
		require.Empty(t, service.privateHostReason(ctx, storepb.Engine_SNOWFLAKE, &storepb.DataSource{Host: "xy12345.eu-west-1"}))
	})
}

// probeDriver is a driver whose Ping and Close are observable.
type probeDriver struct {
	db.Driver
	ping        func(context.Context) error
	closed      chan struct{}
	closeCtxErr error
}

func newProbeDriver(ping func(context.Context) error) *probeDriver {
	return &probeDriver{ping: ping, closed: make(chan struct{})}
}

func (d *probeDriver) Ping(ctx context.Context) error { return d.ping(ctx) }

func (d *probeDriver) Close(ctx context.Context) error {
	d.closeCtxErr = ctx.Err()
	close(d.closed)
	return nil
}

func (d *probeDriver) requireClosed(t *testing.T) {
	t.Helper()
	select {
	case <-d.closed:
	case <-time.After(5 * time.Second):
		require.Fail(t, "the driver was never closed")
	}
}

func TestProbeConnection(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	untilDone := func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}

	t.Run("a driver that answers", func(t *testing.T) {
		t.Parallel()
		driver := newProbeDriver(func(context.Context) error { return nil })
		probe := probeConnection(ctx, time.Minute, func(context.Context) (db.Driver, error) { return driver, nil })
		require.Equal(t, connectionProbe{}, probe)
		driver.requireClosed(t)
	})

	t.Run("a ping that waits for its context is cut off at the bound", func(t *testing.T) {
		t.Parallel()
		driver := newProbeDriver(untilDone)
		probe := probeConnection(ctx, 20*time.Millisecond, func(context.Context) (db.Driver, error) { return driver, nil })
		require.ErrorIs(t, probe.err, context.DeadlineExceeded)
		require.False(t, probe.openFailed)
		require.True(t, probe.timedOut)
		driver.requireClosed(t)
	})

	t.Run("an open that ignores its context is abandoned at the bound", func(t *testing.T) {
		t.Parallel()
		release := make(chan struct{})
		driver := newProbeDriver(func(ctx context.Context) error { return ctx.Err() })
		start := time.Now()
		probe := probeConnection(ctx, 20*time.Millisecond, func(context.Context) (db.Driver, error) {
			<-release
			return driver, nil
		})
		require.Less(t, time.Since(start), 5*time.Second)
		require.ErrorIs(t, probe.err, context.DeadlineExceeded)
		require.True(t, probe.timedOut)

		// The abandoned attempt still closes the driver it ends up with, and
		// not under the probe's expired context.
		close(release)
		driver.requireClosed(t)
		require.NoError(t, driver.closeCtxErr)
	})

	t.Run("a driver that cannot be built", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("ssl-cert and ssl-key must be both set or unset")
		probe := probeConnection(ctx, time.Minute, func(context.Context) (db.Driver, error) { return nil, cause })
		require.Equal(t, connectionProbe{err: cause, openFailed: true}, probe)
	})

	t.Run("a ping the database refuses", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("password authentication failed")
		driver := newProbeDriver(func(context.Context) error { return cause })
		probe := probeConnection(ctx, time.Minute, func(context.Context) (db.Driver, error) { return driver, nil })
		require.Equal(t, connectionProbe{err: cause}, probe)
		driver.requireClosed(t)
	})

	t.Run("the caller giving up is not the probe's bound", func(t *testing.T) {
		t.Parallel()
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		driver := newProbeDriver(untilDone)
		probe := probeConnection(canceled, time.Minute, func(context.Context) (db.Driver, error) { return driver, nil })
		require.ErrorIs(t, probe.err, context.Canceled)
		require.False(t, probe.timedOut)
		// The driver is closed although the request is over.
		driver.requireClosed(t)
		require.NoError(t, driver.closeCtxErr)
	})
}

func TestDescribeConnectionTimeout(t *testing.T) {
	t.Parallel()
	dataSource := &storepb.DataSource{Host: "db.example.com", Port: "5432"}

	t.Run("names the host and port that did not answer", func(t *testing.T) {
		t.Parallel()
		err := describeConnectionTimeout(true, dataSource, context.DeadlineExceeded)
		require.EqualError(t, err, "connection to db.example.com:5432 timed out after 20s: context deadline exceeded")
		require.Equal(t, connectionCategoryTimeout, classifyConnectionFailure(err))
	})

	t.Run("names the host alone when there is no port", func(t *testing.T) {
		t.Parallel()
		err := describeConnectionTimeout(true, &storepb.DataSource{Host: "db.example.com"}, errors.New("driver gave up"))
		require.EqualError(t, err, "connection to db.example.com timed out after 20s: driver gave up")
	})

	t.Run("names both hops of an SSH tunnel", func(t *testing.T) {
		t.Parallel()
		tunneled := &storepb.DataSource{Host: "10.0.0.5", Port: "5432", SshHost: "bastion.example.com", SshPort: "22"}
		err := describeConnectionTimeout(true, tunneled, context.DeadlineExceeded)
		require.EqualError(t, err, "connection to 10.0.0.5:5432 through the SSH tunnel at bastion.example.com:22 timed out after 20s: context deadline exceeded")
	})

	t.Run("keeps a definite cause returned after the deadline", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("password authentication failed for user bytebase")
		require.Equal(t, cause, describeConnectionTimeout(true, dataSource, cause))
	})

	t.Run("leaves errors alone before the deadline", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("driver returned an unexpected error")
		require.Equal(t, cause, describeConnectionTimeout(false, dataSource, cause))
	})
}

func TestBuildInstanceConnectionLogAttrs(t *testing.T) {
	t.Parallel()
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

// TestApplyDataSourceUpdateMaskCredentials covers the T15 headline: the mask
// path decides which IAM credential is written. The branch used to dispatch on
// the request's authentication_type, which an AIP-134 client leaves unset when
// it masks only the credential — so rotating a leaked key wrote nothing and
// returned 200.
func TestApplyDataSourceUpdateMaskCredentials(t *testing.T) {
	gcpStored := func() *storepb.DataSource {
		return &storepb.DataSource{
			Id:                 "admin",
			AuthenticationType: storepb.DataSource_GOOGLE_CLOUD_SQL_IAM,
			IamExtension: &storepb.DataSource_GcpCredential{
				GcpCredential: &storepb.DataSource_GCPCredential{Content: "leaked"},
			},
		}
	}

	t.Run("rotation without authentication_type in the body", func(t *testing.T) {
		dataSource := gcpStored()
		err := applyDataSourceUpdateMask(dataSource, &v1pb.DataSource{
			IamExtension: &v1pb.DataSource_GcpCredential{
				GcpCredential: &v1pb.DataSource_GCPCredential{Content: "rotated"},
			},
		}, []string{"gcp_credential"})
		require.NoError(t, err)
		require.Equal(t, "rotated", dataSource.GetGcpCredential().GetContent())
	})

	t.Run("credential path disagreeing with the stored type", func(t *testing.T) {
		dataSource := gcpStored()
		err := applyDataSourceUpdateMask(dataSource, &v1pb.DataSource{
			AuthenticationType: v1pb.DataSource_AWS_RDS_IAM,
			IamExtension: &v1pb.DataSource_AwsCredential{
				AwsCredential: &v1pb.DataSource_AWSCredential{AccessKeyId: "AKIA"},
			},
		}, []string{"aws_credential"})
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		require.Equal(t, "leaked", dataSource.GetGcpCredential().GetContent())
		require.Equal(t, storepb.DataSource_GOOGLE_CLOUD_SQL_IAM, dataSource.GetAuthenticationType())
	})

	t.Run("switching type and credential together, either path order", func(t *testing.T) {
		for _, paths := range [][]string{
			{"authentication_type", "aws_credential"},
			{"aws_credential", "authentication_type"},
		} {
			dataSource := gcpStored()
			err := applyDataSourceUpdateMask(dataSource, &v1pb.DataSource{
				AuthenticationType: v1pb.DataSource_AWS_RDS_IAM,
				IamExtension: &v1pb.DataSource_AwsCredential{
					AwsCredential: &v1pb.DataSource_AWSCredential{AccessKeyId: "AKIA"},
				},
			}, paths)
			require.NoError(t, err, paths)
			require.Equal(t, "AKIA", dataSource.GetAwsCredential().GetAccessKeyId())
		}
	})

	t.Run("a masked credential the body omits is cleared", func(t *testing.T) {
		dataSource := gcpStored()
		require.NoError(t, applyDataSourceUpdateMask(dataSource, &v1pb.DataSource{}, []string{"gcp_credential"}))
		require.Nil(t, dataSource.GetIamExtension())
	})

	t.Run("unknown path", func(t *testing.T) {
		err := applyDataSourceUpdateMask(gcpStored(), &v1pb.DataSource{}, []string{"nope"})
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
}

func newInstanceServiceTestLicenseService(t *testing.T, stores *store.Store) *enterprise.LicenseService {
	t.Helper()
	licenseService, err := enterprise.NewLicenseService(common.ReleaseModeDev, stores, false, "")
	require.NoError(t, err)
	return licenseService
}

func setupProjectInstanceLifecycleAPITest(t *testing.T) (context.Context, *store.Store, string, string, string) {
	t.Helper()
	ctx := context.WithValue(context.Background(), common.WorkspaceIDContextKey, "default")
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('default');
		INSERT INTO project (resource_id, workspace, name) VALUES
			('project-a', 'default', 'Project A'),
			('project-b', 'default', 'Project B');
	`)
	require.NoError(t, err)

	projectID := "project-a"
	instanceID := "project-instance"
	_, err = stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: instanceID,
		Workspace:  "default",
		ProjectID:  &projectID,
		Metadata: &storepb.Instance{
			Activation: true,
			DataSources: []*storepb.DataSource{{
				Id:   "admin",
				Type: storepb.DataSourceType_ADMIN,
			}},
		},
	})
	require.NoError(t, err)

	databaseName := "app"
	_, err = stores.UpsertDatabase(ctx, &store.DatabaseMessage{
		InstanceID:   instanceID,
		DatabaseName: databaseName,
		ProjectID:    projectID,
		Metadata:     &storepb.DatabaseMetadata{},
	})
	require.NoError(t, err)
	return ctx, stores, projectID, instanceID, databaseName
}
