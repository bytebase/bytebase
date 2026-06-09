package dynamodb

import (
	"testing"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
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
