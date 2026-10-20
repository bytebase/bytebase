package pg

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

func TestQueryConnStopsServerQueryWhenContextEnds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sshTunnel bool
		tls       bool
		// timeout is the context deadline; zero cancels the context instead.
		timeout time.Duration
		wantErr string
	}{
		{name: "direct, canceled", wantErr: "context canceled"},
		{name: "direct, deadline exceeded", timeout: 2 * time.Second, wantErr: "context deadline exceeded"},
		{name: "ssh tunnel, canceled", sshTunnel: true, wantErr: "context canceled"},
		{name: "ssh tunnel, deadline exceeded", sshTunnel: true, timeout: 2 * time.Second, wantErr: "context deadline exceeded"},
		{name: "ssh tunnel with TLS, canceled", sshTunnel: true, tls: true, wantErr: "context canceled"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var pgContainer *testcontainer.Container
			var dbName string
			var rawDB *sql.DB
			if tc.tls {
				pgContainer = testcontainer.SharedTLSPgContainer(t)
				dbName, rawDB = testcontainer.NewTLSPgDatabase(t)
			} else {
				pgContainer = testcontainer.SharedPgContainer(t)
				dbName, rawDB = testcontainer.NewPgDatabase(t)
			}
			dataSource := &storepb.DataSource{
				Host:     pgContainer.GetHost(),
				Port:     pgContainer.GetPort(),
				Username: "postgres",
				UseSsl:   tc.tls,
			}
			if tc.sshTunnel {
				sshServer := testcontainer.NewSSHServer(t)
				dataSource.SshHost = sshServer.GetHost()
				dataSource.SshPort = sshServer.GetPort()
				dataSource.SshUser = sshServer.GetUser()
				dataSource.SshPrivateKey = sshServer.GetPrivateKey()
			}

			background := context.Background()
			driver, err := (&Driver{}).Open(background, storepb.Engine_POSTGRES, db.ConnectionConfig{
				DataSource:        dataSource,
				Password:          "root-password",
				ConnectionContext: db.ConnectionContext{DatabaseName: dbName},
			})
			require.NoError(t, err)
			t.Cleanup(func() { _ = driver.Close(background) })

			conn, err := driver.GetDB().Conn(background)
			require.NoError(t, err)
			t.Cleanup(func() { _ = conn.Close() })
			// Runs first, so a query that outlived its context cannot hold up the closes above.
			t.Cleanup(func() {
				_, err := rawDB.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()`, dbName)
				require.NoError(t, err)
			})

			ctx, cancel := context.WithCancel(background)
			if tc.timeout > 0 {
				ctx, cancel = context.WithTimeout(background, tc.timeout)
			}
			defer cancel()

			done := make(chan []*v1pb.QueryResult, 1)
			go func() {
				results, err := driver.QueryConn(ctx, conn, "SELECT pg_sleep(30);", db.QueryContext{MaximumSQLResultSize: 1 << 30})
				if err != nil {
					results = []*v1pb.QueryResult{{Error: err.Error()}}
				}
				done <- results
			}()

			const runningQueries = `state = 'active' AND query LIKE '%pg_sleep%'`
			require.Eventually(t, func() bool { return countSessions(t, rawDB, dbName, runningQueries) == 1 }, 10*time.Second, 20*time.Millisecond,
				"the query never started on the server")
			if tc.timeout == 0 {
				cancel()
			}

			select {
			case results := <-done:
				require.Len(t, results, 1)
				require.Contains(t, results[0].Error, tc.wantErr)
			case <-time.After(10 * time.Second):
				require.FailNow(t, "QueryConn was still blocked 10s after the query started")
			}
			require.Eventually(t, func() bool { return countSessions(t, rawDB, dbName, runningQueries) == 0 }, 10*time.Second, 20*time.Millisecond,
				"the query kept running on the server after its context ended")
			require.Eventually(t, func() bool { return countSessions(t, rawDB, dbName, `application_name = 'bytebase'`) == 0 }, 10*time.Second, 20*time.Millisecond,
				"the canceled connection stayed open on the server")

			require.NoError(t, driver.Ping(background), "the driver must reconnect once the canceled connection is discarded")
		})
	}
}

// countSessions counts the sessions in the database, other than the one asking, that match the pg_stat_activity condition.
func countSessions(t *testing.T, rawDB *sql.DB, dbName, condition string) int {
	t.Helper()
	var count int
	err := rawDB.QueryRow(`SELECT count(*) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid() AND `+condition, dbName).Scan(&count)
	require.NoError(t, err)
	return count
}
