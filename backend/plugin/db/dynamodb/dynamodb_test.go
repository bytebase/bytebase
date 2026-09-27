package dynamodb

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestDynamoDBEndpoint(t *testing.T) {
	testCases := []struct {
		name string
		ds   *storepb.DataSource
		want string
	}{
		{
			// No host: fall back to the AWS SDK default (real AWS) endpoint.
			name: "no_host_uses_default_aws",
			ds:   &storepb.DataSource{Region: "ap-northeast-1"},
			want: "",
		},
		{
			// DynamoDB Local: host + port, no TLS -> http endpoint.
			name: "local_http",
			ds:   &storepb.DataSource{Host: "localhost", Port: "18000"},
			want: "http://localhost:18000",
		},
		{
			name: "host_only_no_port",
			ds:   &storepb.DataSource{Host: "127.0.0.1"},
			want: "http://127.0.0.1",
		},
		{
			// useSsl toggles the scheme to https.
			name: "https_via_use_ssl",
			ds:   &storepb.DataSource{Host: "dynamo.internal", Port: "8000", UseSsl: true},
			want: "https://dynamo.internal:8000",
		},
		{
			// Explicit scheme in host is respected; port appended once.
			name: "scheme_prefixed_host_appends_port",
			ds:   &storepb.DataSource{Host: "http://localhost", Port: "18000"},
			want: "http://localhost:18000",
		},
		{
			// Explicit scheme + port already in host: used verbatim.
			name: "scheme_prefixed_full_url",
			ds:   &storepb.DataSource{Host: "https://dynamodb.example.com:443"},
			want: "https://dynamodb.example.com:443",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := dynamoDBEndpoint(tc.ds); got != tc.want {
				t.Errorf("dynamoDBEndpoint() = %q, want %q", got, tc.want)
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
