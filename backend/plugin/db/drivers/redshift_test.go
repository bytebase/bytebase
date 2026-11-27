package drivers

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	odriver "github.com/bytebase/omni/driver"
	redshiftdriver "github.com/bytebase/omni/redshift/driver"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/util"
)

func TestRedshiftOpenKeepsHostAndPortWhole(t *testing.T) {
	for _, tc := range []struct {
		name, host, port, wantErr string
	}{
		{"a port naming another host", "redshift.example.com", "5439 host=other.example.com", "invalid port"},
		{"a blank host", "  ", "5439", "host must be set"},
		{"a blank port", "redshift.example.com", "", "port must be set"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			driver, err := newDriver(redshiftSpec).Open(context.Background(), storepb.Engine_REDSHIFT, db.ConnectionConfig{
				DataSource: &storepb.DataSource{Username: "dba", Host: tc.host, Port: tc.port},
			})
			if driver != nil {
				_ = driver.Close(context.Background())
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

// TestRedshiftConfig maps a data source onto the contract's Config, which is the
// part of Open that does not reach a server.
func TestRedshiftConfig(t *testing.T) {
	for _, tc := range []struct {
		name      string
		dataStore *storepb.DataSource
		ctx       db.ConnectionContext
		want      struct {
			database  string
			readOnly  bool
			datashare bool
			tls       bool
		}
		wantErr string
	}{
		{
			name:      "the context's database wins",
			dataStore: &storepb.DataSource{Username: "dba", Database: "from-source"},
			ctx:       db.ConnectionContext{DatabaseName: "from-context"},
			want: struct {
				database  string
				readOnly  bool
				datashare bool
				tls       bool
			}{database: "from-context"},
		},
		{
			name:      "the data source's database is the next choice",
			dataStore: &storepb.DataSource{Username: "dba", Database: "from-source"},
			want: struct {
				database  string
				readOnly  bool
				datashare bool
				tls       bool
			}{database: "from-source"},
		},
		{
			name:      "no database leaves the contract's default",
			dataStore: &storepb.DataSource{Username: "dba"},
		},
		{
			name:      "read only and datashare are carried",
			dataStore: &storepb.DataSource{Username: "dba"},
			ctx:       db.ConnectionContext{ReadOnly: true, DataShare: true},
			want: struct {
				database  string
				readOnly  bool
				datashare bool
				tls       bool
			}{readOnly: true, datashare: true},
		},
		{
			name:      "use-ssl alone leaves libpq's negotiation alone",
			dataStore: &storepb.DataSource{Username: "dba", UseSsl: true},
		},
		{
			name:      "a user is required",
			dataStore: &storepb.DataSource{},
			wantErr:   "user must be set",
		},
		{
			name:      "a certificate requires its key",
			dataStore: &storepb.DataSource{Username: "dba", SslCert: "cert"},
			wantErr:   "ssl-cert and ssl-key must be both set or unset",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := redshiftConfig(db.ConnectionConfig{DataSource: tc.dataStore, ConnectionContext: tc.ctx})
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want.database, cfg.Database)
			require.Equal(t, tc.want.readOnly, cfg.ReadOnly)
			require.Equal(t, tc.want.datashare, cfg.Params[redshiftdriver.ParamDatashare] == "true")
			require.Equal(t, tc.want.tls, cfg.TLS != nil)
		})
	}
}

func TestRedshiftDumpWritesNothing(t *testing.T) {
	var out strings.Builder
	require.NoError(t, newDriver(redshiftSpec).Dump(t.Context(), &out, nil))
	require.Empty(t, out.String())
}

type fakeRedshiftDriver struct {
	execScript string
	execOpts   odriver.ExecuteOptions
	countStmt  string
	countRows  int64
	session    odriver.Session
}

func (*fakeRedshiftDriver) Ping(context.Context) error { return nil }
func (*fakeRedshiftDriver) Close() error               { return nil }
func (f *fakeRedshiftDriver) Session(context.Context) (odriver.Session, error) {
	return f.session, nil
}

func (f *fakeRedshiftDriver) Execute(_ context.Context, script string, opts odriver.ExecuteOptions) (int64, error) {
	f.execScript = script
	f.execOpts = opts
	if opts.Log != nil {
		opts.Log.CommandExecute(odriver.Command{Text: script, ByteStart: 0, ByteEnd: len(script)})
		opts.Log.CommandResponse(2, []int64{1, 1}, nil)
	}
	return 2, nil
}

func (f *fakeRedshiftDriver) CountAffectedRows(_ context.Context, stmt string) (int64, error) {
	f.countStmt = stmt
	return f.countRows, nil
}

type fakeRedshiftSession struct {
	gotOpts odriver.QueryOptions
	results []*odriver.Result
}

func (s *fakeRedshiftSession) Query(_ context.Context, _ string, opts odriver.QueryOptions) ([]*odriver.Result, error) {
	s.gotOpts = opts
	return s.results, nil
}

func (*fakeRedshiftSession) Close() error { return nil }

func TestRedshiftAdapterSeam(t *testing.T) {
	ctx := context.Background()
	session := &fakeRedshiftSession{
		results: []*odriver.Result{{
			Statement: "SELECT 1",
			Columns:   []odriver.Column{{Name: "n", TypeName: "int4"}},
			Rows:      [][]odriver.Value{{{Kind: odriver.Int64, Int64: 1}}},
		}},
	}
	fake := &fakeRedshiftDriver{countRows: 42, session: session}
	d := &Driver{spec: redshiftSpec, OmniAdapter: util.OmniAdapter{Driver: fake}}

	results, err := d.QueryConn(ctx, "SELECT 1", db.QueryContext{
		Schema:               "analytics",
		Limit:                10,
		Explain:              true,
		MaximumSQLResultSize: 2048,
		MaskingEnabled:       true,
	})
	require.NoError(t, err)
	require.Equal(t, "analytics", session.gotOpts.Schema)
	require.Equal(t, 10, session.gotOpts.Limit)
	require.True(t, session.gotOpts.Explain)
	require.Equal(t, int64(2048), session.gotOpts.MaxResultBytes)
	require.True(t, session.gotOpts.MaskingEnabled)
	require.Len(t, results, 1)
	require.Equal(t, int64(1), results[0].Rows[0].Values[0].GetInt64Value())

	var logs []*storepb.TaskRunLog
	rows, err := d.Execute(ctx, "-- txn-mode = off\nINSERT INTO t VALUES (1)", db.ExecuteOptions{
		CreateTaskRunLog: func(_ time.Time, entry *storepb.TaskRunLog) error {
			logs = append(logs, entry)
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), rows)
	require.Equal(t, odriver.AutoCommit, fake.execOpts.Transaction)
	require.Equal(t, "INSERT INTO t VALUES (1)", fake.execScript)
	require.Len(t, logs, 2)
	require.Equal(t, int64(2), logs[1].GetCommandResponse().GetAffectedRows())
	require.Nil(t, logs[1].GetCommandResponse().GetAllAffectedRows())

	_, err = d.Execute(ctx, "-- txn-mode = off\nCREATE DATABASE analytics", db.ExecuteOptions{CreateDatabase: true})
	require.NoError(t, err)
	require.True(t, fake.execOpts.CreateDatabase)
	require.Equal(t, "-- txn-mode = off\nCREATE DATABASE analytics", fake.execScript)

	estimator, ok := db.As[db.RowEstimator](d)
	require.True(t, ok)
	estimated, err := estimator.CountAffectedRows(ctx, "DELETE FROM t WHERE id > 1")
	require.NoError(t, err)
	require.Equal(t, int64(42), estimated)
	require.Equal(t, "DELETE FROM t WHERE id > 1", fake.countStmt)
}

func TestRedshiftPingThroughSSHTunnelReturnsWhenContextEnds(t *testing.T) {
	t.Parallel()
	sshServer := testcontainer.NewSSHServer(t)
	// The server accepts the connection and never answers the startup message.
	host, port := testcontainer.NewTCPServer(t, func(net.Conn) {})

	background := context.Background()
	driver, err := newDriver(redshiftSpec).Open(background, storepb.Engine_REDSHIFT, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:          host,
			Port:          port,
			Username:      "bytebase",
			SshHost:       sshServer.GetHost(),
			SshPort:       sshServer.GetPort(),
			SshUser:       sshServer.GetUser(),
			SshPrivateKey: sshServer.GetPrivateKey(),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = driver.Close(background) })

	ctx, cancel := context.WithTimeout(background, 500*time.Millisecond)
	defer cancel()
	pinged := make(chan error, 1)
	go func() { pinged <- driver.Ping(ctx) }()

	select {
	case err := <-pinged:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(10 * time.Second):
		require.FailNow(t, "Ping was still blocked long after its context ended")
	}
}
