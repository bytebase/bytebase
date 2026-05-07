package store

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsFilePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{
			name: "postgres URL",
			in:   "postgres://bb@example.com:5432/bytebase",
			want: false,
		},
		{
			name: "postgresql URL",
			in:   "postgresql://bb@example.com:5432/bytebase",
			want: false,
		},
		{
			name: "keyword value DSN",
			in:   "host=example.com port=5432 user=bb dbname=bytebase",
			want: false,
		},
		{
			name: "keyword value IAM DSN",
			in:   "host=example.com port=5432 user=bb dbname=bytebase bytebase_aws_rds_iam=true bytebase_aws_region=us-east-1 sslmode=verify-full",
			want: false,
		},
		{
			name: "keyword value DSN with runtime parameter first",
			in:   "application_name=bytebase host=example.com port=5432 user=bb dbname=bytebase",
			want: false,
		},
		{
			name: "unix socket keyword value DSN",
			in:   "host=/tmp port=5432 user=bb dbname=bytebase",
			want: false,
		},
		{
			name: "file path",
			in:   "/var/run/bytebase/pg-url",
			want: true,
		},
		{
			name: "file path with equals",
			in:   "/var/run/bytebase/pg-url=prod",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isFilePath(tt.in))
		})
	}
}
