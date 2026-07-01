package v1

import (
	"testing"

	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

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
