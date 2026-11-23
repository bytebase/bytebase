package dynamodb

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	omnidriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func TestEndpointTLS(t *testing.T) {
	testCases := []struct {
		name       string
		dataSource *storepb.DataSource
		wantTLS    bool
	}{
		{
			// No host: the AWS SDK resolves the region's own https endpoint.
			name:       "no_host_uses_the_sdk_transport",
			dataSource: &storepb.DataSource{Region: "ap-northeast-1", UseSsl: true},
		},
		{
			// DynamoDB Local: host + port, no TLS -> http endpoint.
			name:       "local_http",
			dataSource: &storepb.DataSource{Host: "localhost", Port: "18000"},
		},
		{
			name:       "bare_host_with_use_ssl_makes_https",
			dataSource: &storepb.DataSource{Host: "dynamo.internal", Port: "8000", UseSsl: true},
			wantTLS:    true,
		},
		{
			// An explicit scheme wins.
			name:       "scheme_prefixed_host_keeps_its_scheme",
			dataSource: &storepb.DataSource{Host: "http://localhost", Port: "18000", UseSsl: true},
		},
		{
			name:       "scheme_prefixed_full_url",
			dataSource: &storepb.DataSource{Host: "https://dynamodb.example.com:443", UseSsl: true},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := endpointTLS(tc.dataSource)
			require.Equal(t, tc.wantTLS, got != nil)
			if got == nil {
				return
			}
			config, err := got("")
			require.NoError(t, err)
			require.False(t, config.InsecureSkipVerify, "endpoint TLS must verify against system roots")
		})
	}
}

// setHermeticAWSEnv pins base credentials and points STS at an unroutable local
// endpoint so Open never reaches real AWS: an attempted AssumeRole fails fast
// and deterministically instead of depending on ambient credentials or network.
func setHermeticAWSEnv(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_ENDPOINT_URL_STS", "http://127.0.0.1:1")
	t.Setenv("AWS_MAX_ATTEMPTS", "1")
	t.Setenv("AWS_PROFILE", "")
	missing := filepath.Join(t.TempDir(), "nonexistent")
	t.Setenv("AWS_CONFIG_FILE", missing)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", missing)
}

func TestOpenAssumesRoleWhenRoleArnSet(t *testing.T) {
	setHermeticAWSEnv(t)

	d := newDriver()
	_, err := d.Open(context.Background(), storepb.Engine_DYNAMODB, db.ConnectionConfig{
		ConnectionContext: db.ConnectionContext{InstanceID: "dynamodb-test"},
		DataSource: &storepb.DataSource{
			Region: "us-east-1",
			IamExtension: &storepb.DataSource_AwsCredential{
				AwsCredential: &storepb.DataSource_AWSCredential{
					RoleArn: "arn:aws:iam::123456789012:role/bytebase-test-role",
				},
			},
		},
	})
	require.ErrorContains(t, err, "failed to assume role")
}

func TestOpenWithoutRoleArnSkipsAssumeRole(t *testing.T) {
	setHermeticAWSEnv(t)

	d := newDriver()
	opened, err := d.Open(context.Background(), storepb.Engine_DYNAMODB, db.ConnectionConfig{
		ConnectionContext: db.ConnectionContext{InstanceID: "dynamodb-test"},
		DataSource: &storepb.DataSource{
			Region: "us-east-1",
			IamExtension: &storepb.DataSource_AwsCredential{
				AwsCredential: &storepb.DataSource_AWSCredential{
					AccessKeyId:     "AKIAIOSFODNN7EXAMPLE",
					SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				},
			},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = opened.Close(context.Background()) })
}

// A configured accountId skips STS and names the pseudo database directly.
func TestSyncInstanceUsesConfiguredAccount(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	host, port, ok := strings.Cut(strings.TrimPrefix(server.URL, "http://"), ":")
	require.True(t, ok)

	setHermeticAWSEnv(t)
	t.Setenv("AWS_ENDPOINT_URL_STS", server.URL)

	opened, err := newDriver().Open(context.Background(), storepb.Engine_DYNAMODB, db.ConnectionConfig{
		ConnectionContext: db.ConnectionContext{DatabaseName: "123456789012-us-east-1"},
		DataSource: &storepb.DataSource{
			Host:                      host,
			Port:                      port,
			Region:                    "us-east-1",
			ExtraConnectionParameters: map[string]string{"accountId": "999999999999"},
			IamExtension: &storepb.DataSource_AwsCredential{
				AwsCredential: &storepb.DataSource_AWSCredential{
					AccessKeyId:     "AKIAIOSFODNN7EXAMPLE",
					SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
				},
			},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = opened.Close(context.Background()) })

	instance, err := opened.SyncInstance(context.Background())
	require.NoError(t, err)
	require.Len(t, instance.Databases, 1)
	require.Equal(t, "999999999999-us-east-1", instance.Databases[0].Name)
	require.Zero(t, requests, "configured accountId must call neither STS nor DynamoDB")
}

func TestConvertValue(t *testing.T) {
	require.Equal(t, util.NullRowValue, convertValue(omnidriver.Value{Kind: omnidriver.Null}))
	require.True(t, convertValue(omnidriver.Value{Kind: omnidriver.Bool, Bool: true}).GetBoolValue())
	require.Equal(t, []byte{0, 1, 2}, convertValue(omnidriver.Value{Kind: omnidriver.Bytes, Bytes: []byte{0, 1, 2}}).GetBytesValue())
	require.Equal(t, "hello", convertValue(omnidriver.Value{Kind: omnidriver.String, String: "hello"}).GetStringValue())
	require.Equal(t, `{"a":1}`, convertValue(omnidriver.Value{
		Kind:       omnidriver.Structured,
		Structured: map[string]any{"a": float64(1)},
	}).GetStringValue())
}

type fakeDynamoSession struct {
	gotScript string
	gotOpts   omnidriver.QueryOptions
	results   []*omnidriver.Result
	closed    bool
}

func (s *fakeDynamoSession) Query(_ context.Context, script string, opts omnidriver.QueryOptions) ([]*omnidriver.Result, error) {
	s.gotScript = script
	s.gotOpts = opts
	return s.results, nil
}

func (s *fakeDynamoSession) Close() error {
	s.closed = true
	return nil
}

type fakeDynamoDriver struct {
	session       *fakeDynamoSession
	executeScript string
	executeFn     func(script string, opts omnidriver.ExecuteOptions) (int64, error)
	instance      *omnidriver.Instance
	schema        *metadatapb.DatabaseSchemaMetadata
}

func (*fakeDynamoDriver) Ping(context.Context) error { return nil }
func (*fakeDynamoDriver) Close() error               { return nil }
func (f *fakeDynamoDriver) Session(context.Context) (omnidriver.Session, error) {
	return f.session, nil
}

func (f *fakeDynamoDriver) Execute(_ context.Context, script string, opts omnidriver.ExecuteOptions) (int64, error) {
	f.executeScript = script
	if f.executeFn != nil {
		return f.executeFn(script, opts)
	}
	return 0, nil
}

func (f *fakeDynamoDriver) SyncInstance(context.Context) (*omnidriver.Instance, error) {
	return f.instance, nil
}

func (f *fakeDynamoDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return f.schema, nil
}

func (*fakeDynamoDriver) Dump(context.Context, io.Writer, *metadatapb.DatabaseSchemaMetadata) error {
	return nil
}

func TestDynamoDBAdapterSeam(t *testing.T) {
	ctx := context.Background()
	sess := &fakeDynamoSession{
		results: []*omnidriver.Result{
			{
				Statement: `SELECT * FROM "users"`,
				Columns:   []omnidriver.Column{{Name: "pk", TypeName: "String"}, {Name: "meta", TypeName: "Map"}},
				Rows: [][]omnidriver.Value{{
					{Kind: omnidriver.String, String: "a"},
					{Kind: omnidriver.Structured, Structured: map[string]any{"k": "v"}},
				}},
				Truncated: true,
			},
			{
				Statement: `SELECT * FROM "fail"`,
				Err:       errors.New("ValidationException: statement failed"),
			},
		},
	}
	var logs []*storepb.TaskRunLog
	impl := &fakeDynamoDriver{
		session: sess,
		executeFn: func(script string, opts omnidriver.ExecuteOptions) (int64, error) {
			opts.Log.CommandExecute(omnidriver.Command{Text: script, ByteStart: 0, ByteEnd: len(script)})
			opts.Log.CommandResponse(0, []int64{0}, nil)
			return 0, nil
		},
		instance: &omnidriver.Instance{
			Databases: []*metadatapb.DatabaseSchemaMetadata{{Name: "123456789012-us-east-1"}},
		},
		schema: &metadatapb.DatabaseSchemaMetadata{Name: "123456789012-us-east-1"},
	}
	d := &Driver{OmniAdapter: util.OmniAdapter{Driver: impl}}

	got, err := d.QueryConn(ctx, `SELECT * FROM "users"; SELECT * FROM "fail"`, db.QueryContext{
		Limit:                5,
		Explain:              true,
		MaximumSQLResultSize: 64,
	})
	require.NoError(t, err)
	require.True(t, sess.closed)
	require.Equal(t, omnidriver.QueryOptions{
		Limit:          5,
		Explain:        true,
		MaxResultBytes: 64,
	}, sess.gotOpts)
	require.Len(t, got, 2)
	require.Equal(t, util.FormatMaximumSQLResultSizeMessage(64), got[0].Error)
	require.Equal(t, `{"k":"v"}`, got[0].Rows[0].Values[1].GetStringValue())
	require.Equal(t, "ValidationException: statement failed", got[1].Error)

	affected, err := d.Execute(ctx, `INSERT INTO "users" VALUE {'pk': 'a'}`, db.ExecuteOptions{
		LogCommandStatement: true,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	})
	require.NoError(t, err)
	require.Zero(t, affected)
	require.Equal(t, `INSERT INTO "users" VALUE {'pk': 'a'}`, impl.executeScript)
	require.Len(t, logs, 2)
	require.Equal(t, storepb.TaskRunLog_COMMAND_EXECUTE, logs[0].Type)
	require.Equal(t, storepb.TaskRunLog_COMMAND_RESPONSE, logs[1].Type)

	inst, err := d.SyncInstance(ctx)
	require.NoError(t, err)
	require.Len(t, inst.Databases, 1)

	schema, err := d.SyncDBSchema(ctx)
	require.NoError(t, err)
	require.Equal(t, "123456789012-us-east-1", schema.GetName())

	var out bytes.Buffer
	require.NoError(t, d.Dump(ctx, &out, schema))
	require.Empty(t, out.String())
}
