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
