package pg

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type stubExplainReply struct {
	planJSON string
	errMsg   string
}

func startStubPGExplainServer(t *testing.T, handle func(sql string) stubExplainReply) (string, string, *[]string, *sync.Mutex) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	var mu sync.Mutex
	var queries []string
	done := make(chan struct{})
	t.Cleanup(func() {
		close(done)
		_ = ln.Close()
	})

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveStubPGExplainConn(conn, done, &mu, &queries, handle)
		}
	}()

	host, port, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	return host, port, &queries, &mu
}

func serveStubPGExplainConn(c net.Conn, done <-chan struct{}, mu *sync.Mutex, queries *[]string, handle func(sql string) stubExplainReply) {
	defer func() { _ = c.Close() }()
	go func() {
		<-done
		_ = c.Close()
	}()

	backend := pgproto3.NewBackend(c, c)
	for {
		msg, err := backend.ReceiveStartupMessage()
		if err != nil {
			return
		}
		switch msg.(type) {
		case *pgproto3.SSLRequest:
			if _, err := c.Write([]byte{'N'}); err != nil {
				return
			}
		case *pgproto3.StartupMessage:
			backend.Send(&pgproto3.AuthenticationOk{})
			backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
			if err := backend.Flush(); err != nil {
				return
			}
			serveStubPGExplainLoop(backend, mu, queries, handle)
			return
		default:
			return
		}
	}
}

func serveStubPGExplainLoop(backend *pgproto3.Backend, mu *sync.Mutex, queries *[]string, handle func(sql string) stubExplainReply) {
	var curSQL string
	txStatus := byte('I')
	run := func(sql string) {
		trimmed := strings.TrimSpace(sql)
		mu.Lock()
		*queries = append(*queries, trimmed)
		mu.Unlock()
		switch upper := strings.ToUpper(trimmed); {
		case strings.HasPrefix(upper, "BEGIN"):
			txStatus = 'T'
			backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("BEGIN")})
		case strings.HasPrefix(upper, "ROLLBACK"):
			txStatus = 'I'
			backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("ROLLBACK")})
		case strings.HasPrefix(upper, "SET "):
			backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("SET")})
		default:
			rep := handle(trimmed)
			if rep.errMsg != "" {
				if txStatus == 'T' {
					txStatus = 'E'
				}
				backend.Send(&pgproto3.ErrorResponse{Severity: "ERROR", Code: "42P01", Message: rep.errMsg})
				return
			}
			backend.Send(&pgproto3.RowDescription{Fields: []pgproto3.FieldDescription{{Name: []byte("QUERY PLAN"), DataTypeOID: pgtype.TextOID, TypeModifier: -1}}})
			backend.Send(&pgproto3.DataRow{Values: [][]byte{[]byte(rep.planJSON)}})
			backend.Send(&pgproto3.CommandComplete{CommandTag: []byte("EXPLAIN")})
		}
	}
	for {
		msg, err := backend.Receive()
		if err != nil {
			return
		}
		switch m := msg.(type) {
		case *pgproto3.Terminate:
			return
		case *pgproto3.Query:
			run(m.String)
			backend.Send(&pgproto3.ReadyForQuery{TxStatus: txStatus})
			if err := backend.Flush(); err != nil {
				return
			}
		case *pgproto3.Parse:
			curSQL = m.Query
			backend.Send(&pgproto3.ParseComplete{})
		case *pgproto3.Describe:
			backend.Send(&pgproto3.ParameterDescription{})
			backend.Send(&pgproto3.NoData{})
		case *pgproto3.Bind:
			backend.Send(&pgproto3.BindComplete{})
		case *pgproto3.Execute:
			run(curSQL)
		case *pgproto3.Sync:
			backend.Send(&pgproto3.ReadyForQuery{TxStatus: txStatus})
			if err := backend.Flush(); err != nil {
				return
			}
		default:
		}
	}
}

// TestCountAffectedRows pins the adapter's CountAffectedRows forwarding seam,
// including Bytebase's base.WithSearchPath compatibility with omni/pg/driver.
func TestCountAffectedRows(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	host, port, queries, mu := startStubPGExplainServer(t, func(sql string) stubExplainReply {
		switch {
		case strings.Contains(sql, "SELECT * FROM big"):
			return stubExplainReply{planJSON: `[{"Plan":{"Node Type":"Seq Scan","Plan Rows":10000}}]`}
		case strings.Contains(sql, "missing_table"):
			return stubExplainReply{errMsg: `relation "missing_table" does not exist`}
		case strings.Contains(sql, "only_in_app"):
			return stubExplainReply{planJSON: `[{"Plan":{"Node Type":"ModifyTable","Operation":"Delete","Plans":[{"Node Type":"Seq Scan","Parent Relationship":"Outer","Plan Rows":100}]}}]`}
		default:
			return stubExplainReply{planJSON: `[{"Plan":{"Node Type":"ModifyTable","Operation":"Delete","Plans":[{"Node Type":"Hash Join","Parent Relationship":"Outer","Plan Rows":1000}]}}]`}
		}
	})

	driver, err := (&Driver{}).Open(ctx, storepb.Engine_POSTGRES, db.ConnectionConfig{
		DataSource: &storepb.DataSource{
			Host:                      host,
			Port:                      port,
			Username:                  "dba",
			ExtraConnectionParameters: map[string]string{"sslmode": "disable"},
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, driver.Close(context.Background())) })

	pgDriver, ok := driver.(*Driver)
	require.True(t, ok)

	rows, err := pgDriver.CountAffectedRows(ctx, "DELETE FROM big USING s WHERE big.s_id = s.id AND s.flag;")
	require.NoError(t, err)
	require.Equal(t, int64(1000), rows)

	_, err = pgDriver.CountAffectedRows(ctx, "SELECT * FROM big;")
	require.Error(t, err)

	_, err = pgDriver.CountAffectedRows(ctx, "DELETE FROM missing_table;")
	require.ErrorContains(t, err, "does not exist")

	mu.Lock()
	*queries = nil
	mu.Unlock()

	rows, err = pgDriver.CountAffectedRows(ctx, base.WithSearchPath("DELETE FROM only_in_app;", []string{"app"}))
	require.NoError(t, err)
	require.Equal(t, int64(100), rows)

	mu.Lock()
	recorded := append([]string(nil), (*queries)...)
	mu.Unlock()
	require.Equal(t, []string{
		"begin",
		`SET LOCAL search_path TO "app"`,
		"EXPLAIN (FORMAT JSON) DELETE FROM only_in_app;",
		"rollback",
	}, recorded)
}
