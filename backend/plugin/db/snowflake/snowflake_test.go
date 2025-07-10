// Package snowflake is the plugin for Snowflake driver.
package snowflake

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestBuildSnowflakeDSN(t *testing.T) {
	testCases := []struct {
		input db.ConnectionConfig
		want  string
	}{
		{
			input: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "nb47110.ap-southeast-1",
					Port:     "443",
					Username: "bytebase",
					Password: "pwd",
				},
				Password: "pwd",
			},
			want: "bytebase:pwd@nb47110.ap-southeast-1.snowflakecomputing.com:443?ocspFailOpen=true&region=ap-southeast-1&validateDefaultParameters=true",
		},
		{
			input: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "nb47110.ap-southeast-1",
					Port:     "443",
					Username: "bytebase",
					Password: "pwd",
				},
				Password: "pwd",
				ConnectionContext: db.ConnectionContext{
					DatabaseName: "SAMPLE_DB",
				},
			},
			want: "bytebase:pwd@nb47110.ap-southeast-1.snowflakecomputing.com:443?database=%22SAMPLE_DB%22&ocspFailOpen=true&region=ap-southeast-1&validateDefaultParameters=true",
		},
		{
			input: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "nb47110.ap-southeast-1@10.0.0.1",
					Port:     "4182",
					Username: "bytebase",
					Password: "pwd",
				},
				Password: "pwd",
			},
			want: "bytebase:pwd@10.0.0.1:443?account=nb47110&ocspFailOpen=true&region=ap-southeast-1&validateDefaultParameters=true",
		},
		{
			input: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:     "nb47110.ap-southeast-1@10.0.0.1",
					Port:     "4182",
					Username: "bytebase",
					Password: "pwd",
				},
				Password: "pwd",
				ConnectionContext: db.ConnectionContext{
					DatabaseName: "SAMPLE_DB",
				},
			},
			want: "bytebase:pwd@10.0.0.1:443?account=nb47110&database=%22SAMPLE_DB%22&ocspFailOpen=true&region=ap-southeast-1&validateDefaultParameters=true",
		},
	}
	for _, testCase := range testCases {
		got, _ /* redacted */, err := buildSnowflakeDSN(testCase.input)
		require.NoError(t, err)
		require.Equal(t, testCase.want, got)
	}
}
