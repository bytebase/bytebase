package databricks

import (
	"context"
	"testing"
	"time"

	odbdriver "github.com/bytebase/omni/driver"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func TestDatabricksConfig(t *testing.T) {
	for _, tc := range []struct {
		name     string
		config   db.ConnectionConfig
		want     string
		wantCata string
		wantWrhs string
	}{
		{
			name: "the token is the authentication private key and the catalog is the database",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:                     "https://dbc-00000000-0000.cloud.databricks.com",
					AuthenticationPrivateKey: "token",
					WarehouseId:              "warehouse",
				},
				ConnectionContext: db.ConnectionContext{DatabaseName: "workspace"},
			},
			want:     "token",
			wantCata: "workspace",
			wantWrhs: "warehouse",
		},
		{
			name: "a resolved token stands in when the data source carries none",
			config: db.ConnectionConfig{
				DataSource:        &storepb.DataSource{Host: "https://dbc.example.com", WarehouseId: "warehouse"},
				ConnectionContext: db.ConnectionContext{DatabaseName: "workspace"},
				Password:          "resolved",
			},
			want:     "resolved",
			wantCata: "workspace",
			wantWrhs: "warehouse",
		},
		{
			name: "an SSL or SSH data source adds no TLS and no dialer",
			config: db.ConnectionConfig{
				DataSource: &storepb.DataSource{
					Host:                     "https://dbc.example.com",
					AuthenticationPrivateKey: "token",
					WarehouseId:              "warehouse",
					UseSsl:                   true,
					VerifyTlsCertificate:     true,
					SshHost:                  "bastion.example.com",
					SshPort:                  "22",
					SshUser:                  "user",
				},
				ConnectionContext: db.ConnectionContext{DatabaseName: "workspace"},
			},
			want:     "token",
			wantCata: "workspace",
			wantWrhs: "warehouse",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := databricksConfig(tc.config)
			require.Equal(t, tc.want, got.Password)
			require.Equal(t, tc.wantCata, got.Database)
			require.Equal(t, tc.wantWrhs, got.Params["warehouseId"])
			require.Equal(t, tc.config.DataSource.GetHost(), got.Host)
			require.Nil(t, got.TLS)
			require.Nil(t, got.Dial)
		})
	}
}

func TestOpenRefusesAnIncompleteConfig(t *testing.T) {
	for _, tc := range []struct {
		name   string
		config db.ConnectionConfig
		want   string
	}{
		{
			name:   "no host",
			config: db.ConnectionConfig{DataSource: &storepb.DataSource{AuthenticationPrivateKey: "token", WarehouseId: "warehouse"}},
			want:   "host must be set",
		},
		{
			name:   "no token",
			config: db.ConnectionConfig{DataSource: &storepb.DataSource{Host: "https://dbc.example.com", WarehouseId: "warehouse"}},
			want:   "token must be set",
		},
		{
			name:   "no warehouse",
			config: db.ConnectionConfig{DataSource: &storepb.DataSource{Host: "https://dbc.example.com", AuthenticationPrivateKey: "token"}},
			want:   "warehouse ID must be set",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newDriver().Open(context.Background(), storepb.Engine_DATABRICKS, tc.config)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

type fakeDatabricksDriver struct {
	execScript string
	execOpts   odbdriver.ExecuteOptions
}

func (*fakeDatabricksDriver) Ping(context.Context) error { return nil }
func (*fakeDatabricksDriver) Close() error               { return nil }

func (f *fakeDatabricksDriver) Execute(_ context.Context, script string, opts odbdriver.ExecuteOptions) (int64, error) {
	f.execScript = script
	f.execOpts = opts
	if opts.Log != nil {
		opts.Log.CommandExecute(odbdriver.Command{Text: script, ByteStart: 0, ByteEnd: len(script)})
		opts.Log.CommandResponse(2, []int64{2}, nil)
	}
	return 2, nil
}

type fakeDatabricksSession struct {
	gotOpts odbdriver.QueryOptions
	results []*odbdriver.Result
}

func (s *fakeDatabricksSession) Query(_ context.Context, _ string, opts odbdriver.QueryOptions) ([]*odbdriver.Result, error) {
	s.gotOpts = opts
	return s.results, nil
}

func (*fakeDatabricksSession) Close() error { return nil }

// TestExecuteLogsRangesInTheCallersScript pins that Execute strips transaction
// directives without setting a transaction mode and shifts TaskRunLog ranges
// back into the caller's original script.
func TestExecuteLogsRangesInTheCallersScript(t *testing.T) {
	fake := &fakeDatabricksDriver{}
	d := &Driver{OmniAdapter: util.OmniAdapter{Driver: fake}}

	script := "-- txn-mode=off\nINSERT INTO workspace.analytics.lookup (id, label) VALUES (1, 'x'), (2, 'y')"
	var logs []*storepb.TaskRunLog
	affected, err := d.Execute(context.Background(), script, db.ExecuteOptions{
		CreateDatabase: true,
		MaximumRetries: 2,
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), affected)
	require.Equal(t, "INSERT INTO workspace.analytics.lookup (id, label) VALUES (1, 'x'), (2, 'y')", fake.execScript)
	require.Zero(t, fake.execOpts.Transaction)
	require.True(t, fake.execOpts.CreateDatabase)
	require.Equal(t, 2, fake.execOpts.MaxRetries)

	require.NotEmpty(t, logs)
	command := logs[0].GetCommandExecute()
	require.NotNil(t, command)
	got := script[command.GetRange().GetStart():command.GetRange().GetEnd()]
	require.NotContains(t, got, "txn-mode")
	require.Equal(t, "INSERT INTO workspace.analytics.lookup (id, label) VALUES (1, 'x'), (2, 'y')", got)
}

func TestQueryConnSeam(t *testing.T) {
	session := &fakeDatabricksSession{
		results: []*odbdriver.Result{{
			Statement: "SELECT 1",
			Columns:   []odbdriver.Column{{Name: "n", TypeName: "INT"}},
			Rows:      [][]odbdriver.Value{{{Kind: odbdriver.Int64, Int64: 1}}},
			Plan:      &odbdriver.Plan{Format: "TEXT"},
		}},
	}
	d := &Driver{}
	d.SetSession(session)

	results, err := d.QueryConn(context.Background(), "SELECT 1", db.QueryContext{
		Schema:               "analytics",
		Limit:                50,
		Explain:              true,
		Option:               &v1pb.QueryOption{ExplainFormat: v1pb.QueryOption_TEXT},
		MaximumSQLResultSize: 4096,
		MaskingEnabled:       true,
	})
	require.NoError(t, err)
	require.Equal(t, "analytics", session.gotOpts.Schema)
	require.Equal(t, 50, session.gotOpts.Limit)
	require.True(t, session.gotOpts.Explain)
	require.Equal(t, "TEXT", session.gotOpts.ExplainFormat)
	require.Equal(t, int64(4096), session.gotOpts.MaxResultBytes)
	require.True(t, session.gotOpts.MaskingEnabled)
	require.Len(t, results, 1)
	require.Equal(t, int64(1), results[0].Rows[0].Values[0].GetInt64Value())
	require.Equal(t, v1pb.QueryOption_TEXT, results[0].QueryPlan.GetFormat())
}
