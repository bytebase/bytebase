package dynamodb

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
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
			// An explicit scheme wins, as it did in the old endpoint builder.
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
			if (got != nil) != tc.wantTLS {
				t.Fatalf("endpointTLS() = %v, want TLS function %v", got != nil, tc.wantTLS)
			}
			if got == nil {
				return
			}
			config, err := got("")
			if err != nil {
				t.Fatalf("endpointTLS() = %v", err)
			}
			// The empty config verifies the endpoint against the system roots:
			// verify_tls_certificate is ignored and a self-signed endpoint does
			// not pass.
			if config.InsecureSkipVerify {
				t.Error("endpoint TLS config skips verification; TLS must verify")
			}
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
	// Single attempt: the SDK's default retryer would otherwise retry the
	// refused connection with backoff, slowing the failure to several seconds.
	t.Setenv("AWS_MAX_ATTEMPTS", "1")
	// Shield the tests from ambient AWS configuration on the host: a set
	// AWS_PROFILE or shared config file would otherwise leak into
	// config.LoadDefaultConfig.
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
	if err == nil {
		t.Fatal("Open() = nil error; want an assume-role failure, meaning role_arn was ignored")
	}
	if !strings.Contains(err.Error(), "failed to assume role") {
		t.Errorf("Open() error = %q, want it to mention the assume-role attempt", err.Error())
	}
}

func TestOpenWithoutRoleArnSkipsAssumeRole(t *testing.T) {
	setHermeticAWSEnv(t)

	d := newDriver()
	if _, err := d.Open(context.Background(), storepb.Engine_DYNAMODB, db.ConnectionConfig{
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
	}); err != nil {
		t.Fatalf("Open() error = %v, want nil when no role_arn is set", err)
	}
}

// A configured account skips STS and names the pseudo database directly.
func TestSyncInstanceUsesConfiguredAccount(t *testing.T) {
	fake := newFakeDynamoDB(t)
	stsServer, stsLog := newFakeSTS(t, stsAnswers)
	setHermeticAWSEnv(t)
	t.Setenv("AWS_ENDPOINT_URL_STS", stsServer.URL)
	t.Cleanup(fake.server.Close)
	t.Cleanup(stsServer.Close)

	driver := openTestDriver(t, fake, map[string]string{"accountId": "999999999999"})
	instance, err := driver.SyncInstance(context.Background())
	if err != nil {
		t.Fatalf("SyncInstance() = %v", err)
	}
	if len(instance.Databases) != 1 || instance.Databases[0].Name != "999999999999-us-east-1" {
		t.Errorf("SyncInstance() databases = %v, want [999999999999-us-east-1]", instance.Databases)
	}
	if len(stsLog.requests) != 0 {
		t.Errorf("STS requests = %v, want none for a configured account", stsLog.requests)
	}
	if len(fake.requests) != 0 {
		t.Errorf("DynamoDB requests = %v, want none for SyncInstance", fake.requests)
	}
}

func TestContractBehavior(t *testing.T) {
	t.Run("limit caps the statement", func(t *testing.T) {
		fake := newFakeDynamoDB(t)
		t.Cleanup(fake.server.Close)
		driver := openTestDriver(t, fake, nil)

		results, err := driver.QueryConn(context.Background(), nil, `SELECT * FROM "limit"`, db.QueryContext{Limit: 1})
		if err != nil {
			t.Fatalf("QueryConn() = %v", err)
		}
		if len(results) != 1 || results[0].RowsCount != 1 {
			t.Fatalf("QueryConn() = %d results, rows %v; want one row", len(results), rowCounts(results))
		}
		// Limit is a cap on the statement's rows, so the driver asks for no
		// page after it has them.
		if len(fake.requests) != 1 {
			t.Errorf("requests = %v, want only the first page", fake.requests)
		}
	})

	t.Run("maximum result size truncates", func(t *testing.T) {
		fake := newFakeDynamoDB(t)
		t.Cleanup(fake.server.Close)
		driver := openTestDriver(t, fake, nil)

		results, err := driver.QueryConn(context.Background(), nil, `SELECT * FROM "big"`, db.QueryContext{MaximumSQLResultSize: 1})
		if err != nil {
			t.Fatalf("QueryConn() = %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("QueryConn() = %d results, want 1", len(results))
		}
		// MaximumSQLResultSize truncates the result and reports the limit.
		if !strings.Contains(results[0].GetError(), "max allowed output size") {
			t.Errorf("QueryConn() error = %q, want the result-size message", results[0].GetError())
		}
	})

	t.Run("binary is bytes", func(t *testing.T) {
		fake := newFakeDynamoDB(t)
		t.Cleanup(fake.server.Close)
		driver := openTestDriver(t, fake, nil)

		results, err := driver.QueryConn(context.Background(), nil, `SELECT * FROM "binary"`, db.QueryContext{})
		if err != nil {
			t.Fatalf("QueryConn() = %v", err)
		}
		value := results[0].Rows[0].Values[0]
		// A binary crosses as bytes, so arbitrary bytes need not be valid UTF-8.
		if string(value.GetBytesValue()) != string([]byte{0, 1, 2}) {
			t.Errorf("binary value = %v, want bytes 00 01 02", value)
		}
	})

	t.Run("missing attribute is null", func(t *testing.T) {
		fake := newFakeDynamoDB(t)
		t.Cleanup(fake.server.Close)
		driver := openTestDriver(t, fake, nil)

		results, err := driver.QueryConn(context.Background(), nil, `SELECT * FROM "sparse"`, db.QueryContext{})
		if err != nil {
			t.Fatalf("QueryConn() = %v", err)
		}
		missing := results[0].Rows[1].Values[0]
		// A missing attribute is an explicit NULL, so the grid renders a null
		// cell rather than a dropped value.
		if _, ok := missing.GetKind().(*v1pb.RowValue_NullValue); !ok {
			t.Errorf("missing attribute = %v, want an explicit NULL", missing)
		}
	})

	t.Run("secondary indexes sync", func(t *testing.T) {
		fake := newFakeDynamoDB(t)
		fake.tables = []string{"orders"}
		fake.paged = false
		t.Cleanup(fake.server.Close)
		driver := openTestDriver(t, fake, nil)

		schema, err := driver.SyncDBSchema(context.Background())
		if err != nil {
			t.Fatalf("SyncDBSchema() = %v", err)
		}
		if len(schema.Schemas) != 1 || len(schema.Schemas[0].Tables) != 1 {
			t.Fatalf("SyncDBSchema() = %v, want one table", schema)
		}
		table := schema.Schemas[0].Tables[0]
		var columns []string
		for _, column := range table.Columns {
			columns = append(columns, column.Name)
		}
		if strings.Join(columns, ",") != "created,pk,sku" {
			t.Errorf("columns = %v, want [created pk sku]", columns)
		}
		// Every key attribute of the table and of its secondary indexes is a
		// column; a hash-only table has no RANGE index.
		var indexes []string
		for _, index := range table.Indexes {
			indexes = append(indexes, index.Type+":"+index.Name+"("+strings.Join(index.Expressions, ",")+")")
		}
		want := []string{
			"HASH:pk(pk)",
			"GLOBAL SECONDARY:orders_by_sku(sku)",
			"LOCAL SECONDARY:orders_by_created(pk,created)",
		}
		if strings.Join(indexes, " ") != strings.Join(want, " ") {
			t.Errorf("indexes = %v, want %v", indexes, want)
		}
	})
}

func openTestDriver(t *testing.T, fake *fakeDynamoDB, extraParams map[string]string) *Driver {
	t.Helper()
	setHermeticAWSEnv(t)
	dataSource := &storepb.DataSource{
		Host:                      fake.host,
		Port:                      fake.port,
		Region:                    "us-east-1",
		ExtraConnectionParameters: extraParams,
		IamExtension: &storepb.DataSource_AwsCredential{
			AwsCredential: &storepb.DataSource_AWSCredential{
				AccessKeyId:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
		},
	}
	inner := newDriver()
	opened, err := inner.Open(context.Background(), storepb.Engine_DYNAMODB, db.ConnectionConfig{
		ConnectionContext: db.ConnectionContext{DatabaseName: "123456789012-us-east-1"},
		DataSource:        dataSource,
	})
	if err != nil {
		t.Fatalf("Open() = %v", err)
	}
	driver, ok := opened.(*Driver)
	if !ok {
		t.Fatalf("Open() returned %T, want *Driver", opened)
	}
	t.Cleanup(func() { _ = driver.Close(context.Background()) })
	return driver
}

func rowCounts(results []*v1pb.QueryResult) []int64 {
	counts := make([]int64, 0, len(results))
	for _, result := range results {
		counts = append(counts, result.RowsCount)
	}
	return counts
}
