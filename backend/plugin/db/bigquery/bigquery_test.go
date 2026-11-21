package bigquery

import (
	"context"
	"errors"
	"testing"

	odriver "github.com/bytebase/omni/driver"
	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func TestBigQueryConfig(t *testing.T) {
	for _, tc := range []struct {
		name             string
		dataSource       *storepb.DataSource
		connectionConfig db.ConnectionConfig
		want             odriver.Config
	}{
		{
			name: "the project is a parameter and the dataset is the database",
			dataSource: &storepb.DataSource{
				ProjectId: "example-project",
				IamExtension: &storepb.DataSource_GcpCredential{
					GcpCredential: &storepb.DataSource_GCPCredential{Content: `{"type":"service_account"}`},
				},
			},
			connectionConfig: db.ConnectionConfig{
				ConnectionContext: db.ConnectionContext{DatabaseName: "analytics"},
			},
			want: odriver.Config{
				Database: "analytics",
				Password: `{"type":"service_account"}`,
				Params:   map[string]string{"project": "example-project"},
			},
		},
		{
			name: "the data source's database field is not the dataset",
			dataSource: &storepb.DataSource{
				ProjectId: "example-project",
				Database:  "from_data_source",
			},
			want: odriver.Config{
				Params: map[string]string{"project": "example-project"},
			},
		},
		{
			name: "a resolved credential wins over the data source's",
			dataSource: &storepb.DataSource{
				ProjectId: "example-project",
				IamExtension: &storepb.DataSource_GcpCredential{
					GcpCredential: &storepb.DataSource_GCPCredential{Content: `{"type":"service_account"}`},
				},
			},
			connectionConfig: db.ConnectionConfig{Password: `{"type":"authorized_user"}`},
			want: odriver.Config{
				Password: `{"type":"authorized_user"}`,
				Params:   map[string]string{"project": "example-project"},
			},
		},
		{
			name: "the endpoint override is the host and port",
			dataSource: &storepb.DataSource{
				ProjectId: "example-project",
				Host:      "bigquery.p.googleapis.com",
				Port:      "443",
			},
			want: odriver.Config{
				Host:   "bigquery.p.googleapis.com",
				Port:   "443",
				Params: map[string]string{"project": "example-project"},
			},
		},
		{
			name: "an SSL data source sets no TLS and an SSH one no dialer",
			dataSource: &storepb.DataSource{
				ProjectId:            "example-project",
				UseSsl:               true,
				VerifyTlsCertificate: true,
				SshHost:              "bastion.example.com",
				SshPort:              "22",
				SshUser:              "user",
			},
			want: odriver.Config{
				Params: map[string]string{"project": "example-project"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := tc.connectionConfig
			config.DataSource = tc.dataSource
			require.Equal(t, tc.want, bigQueryConfig(config))
		})
	}
}

func TestOpenRequiresAProject(t *testing.T) {
	_, err := newDriver().Open(context.Background(), storepb.Engine_BIGQUERY, db.ConnectionConfig{
		DataSource: &storepb.DataSource{Database: "analytics"},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "project")
}

type fakeBigQueryDriver struct {
	execScript string
	execOpts   odriver.ExecuteOptions
	databases  []*metadatapb.DatabaseSchemaMetadata
}

func (*fakeBigQueryDriver) Ping(context.Context) error { return nil }
func (*fakeBigQueryDriver) Close() error               { return nil }

func (f *fakeBigQueryDriver) Execute(_ context.Context, script string, opts odriver.ExecuteOptions) (int64, error) {
	f.execScript = script
	f.execOpts = opts
	return 2, nil
}

func (f *fakeBigQueryDriver) SyncInstance(context.Context) (*odriver.Instance, error) {
	return &odriver.Instance{Version: "ignored", Databases: f.databases}, nil
}

func (*fakeBigQueryDriver) SyncDatabase(context.Context) (*metadatapb.DatabaseSchemaMetadata, error) {
	return nil, nil
}

type stubSession struct {
	results []*odriver.Result
}

func (s *stubSession) Query(context.Context, string, odriver.QueryOptions) ([]*odriver.Result, error) {
	return s.results, nil
}

func (*stubSession) Close() error { return nil }

func TestAdapterSeam(t *testing.T) {
	ctx := context.Background()
	affected := int64(3)
	const maxResultBytes = 1 << 20

	fake := &fakeBigQueryDriver{
		databases: []*metadatapb.DatabaseSchemaMetadata{{Name: "analytics"}},
	}
	d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}
	d.SetSession(&stubSession{
		results: []*odriver.Result{
			{
				Statement: "SELEKT 1",
				Err:       errors.New("syntax error"),
			},
			{
				Statement:    "INSERT INTO t VALUES (1), (2), (3)",
				AffectedRows: &affected,
			},
			{
				Statement: "SELECT * FROM t",
				Columns:   []odriver.Column{{Name: "n", TypeName: "NUMERIC"}},
				Rows:      [][]odriver.Value{{{Kind: odriver.String, String: "12.5"}}},
				Truncated: true,
				Plan:      &odriver.Plan{Format: "JSON", Executed: true, Content: `[{"id":"0"}]`},
			},
		},
	})

	results, err := d.QueryConn(ctx, "SELECT 1", db.QueryContext{MaximumSQLResultSize: maxResultBytes})
	require.NoError(t, err)
	require.Len(t, results, 3)

	require.Equal(t, "syntax error", results[0].Error)
	require.Empty(t, results[0].ColumnNames)

	require.Equal(t, []string{"Affected Rows"}, results[1].ColumnNames)
	require.Equal(t, int64(3), results[1].Rows[0].Values[0].GetInt64Value())

	require.Equal(t, "12.5", results[2].Rows[0].Values[0].GetStringValue())
	require.Equal(t, util.FormatMaximumSQLResultSizeMessage(maxResultBytes), results[2].Error)
	require.Equal(t, v1pb.QueryOption_JSON, results[2].QueryPlan.GetFormat())
	require.True(t, results[2].QueryPlan.GetExecuted())
	require.Equal(t, `[{"id":"0"}]`, results[2].QueryPlan.GetContent())

	rows, err := d.Execute(ctx, "-- txn-mode=off\nINSERT INTO t VALUES (1)", db.ExecuteOptions{MaximumRetries: 3})
	require.NoError(t, err)
	require.Equal(t, int64(2), rows)
	require.Equal(t, "-- txn-mode=off\nINSERT INTO t VALUES (1)", fake.execScript)
	require.Equal(t, 3, fake.execOpts.MaxRetries)

	instance, err := d.SyncInstance(ctx)
	require.NoError(t, err)
	require.Empty(t, instance.Version)
	require.Nil(t, instance.Metadata)
	require.Len(t, instance.Databases, 1)
	require.Equal(t, "analytics", instance.Databases[0].Name)
}
