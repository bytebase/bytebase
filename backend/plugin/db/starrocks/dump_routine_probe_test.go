package starrocks

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// BYT-10211: neither Doris nor StarRocks has MySQL stored routines or events, and Doris
// kernels that dropped the PL/SQL grammar (SelectDB 4.0.12+/4.1.8+/5.0.5+, apache/doris#58700)
// reject `SHOW FUNCTION STATUS` outright, which aborted the whole schema sync. The dump must
// complete without ever sending the MySQL routine or event probes.
func TestDumpSendsNoRoutineOrEventProbe(t *testing.T) {
	for _, engine := range []storepb.Engine{storepb.Engine_DORIS, storepb.Engine_STARROCKS} {
		t.Run(engine.String(), func(t *testing.T) {
			dumpRecorder.reset(plsqlStrippedCatalog(engine))
			sqlDB, err := sql.Open(dumpRecorderDriverName, "")
			require.NoError(t, err)
			defer sqlDB.Close()

			d := &Driver{dbType: engine, db: sqlDB, databaseName: "prod_tg"}
			var out bytes.Buffer
			require.NoError(t, d.Dump(context.Background(), &out, nil))

			require.Contains(t, out.String(), "CREATE TABLE `orders`")
			require.Contains(t, out.String(), "CREATE VIEW `v_orders` AS SELECT `id` FROM `orders`")
			for _, q := range dumpRecorder.recorded() {
				require.NotRegexp(t, forbiddenProbe, q)
			}
		})
	}
}

var (
	routineStatusProbe = regexp.MustCompile(`(?i)^\s*SHOW\s+(FUNCTION|PROCEDURE)\s+STATUS`)
	forbiddenProbe     = regexp.MustCompile(`(?i)SHOW\s+((FUNCTION|PROCEDURE)\s+STATUS|EVENTS|CREATE\s+(FUNCTION|PROCEDURE|EVENT))`)
)

// plsqlStrippedCatalog answers the dump's table and view queries with the column shapes a
// live Doris 4.1.3 returns (and the StarRocks materialized-view catalog in its own shape),
// and rejects the routine probes with the exact error a PL/SQL-stripped kernel returns.
func plsqlStrippedCatalog(engine storepb.Engine) func(string) ([]string, [][]driver.Value, error) {
	return func(query string) ([]string, [][]driver.Value, error) {
		switch {
		case routineStatusProbe.MatchString(query):
			return nil, nil, &mysql.MySQLError{Number: 1105, SQLState: [5]byte{'H', 'Y', '0', '0', '0'}, Message: "errCode = 2, detailMessage = no viable alternative at input 'SHOW FUNCTION'(line 1, pos 5)"}
		case strings.Contains(query, "information_schema.TABLES"):
			return []string{"TABLE_NAME", "TABLE_TYPE"}, [][]driver.Value{{"orders", "BASE TABLE"}, {"v_orders", "VIEW"}}, nil
		case strings.Contains(query, "mv_infos("):
			return []string{"Name"}, nil, nil
		case strings.Contains(query, "information_schema.materialized_views"):
			return []string{"TABLE_NAME", "REFRESH_TYPE"}, nil, nil
		case strings.HasPrefix(query, "SHOW CREATE TABLE"):
			return []string{"Table", "Create Table"}, [][]driver.Value{{"orders", "CREATE TABLE `orders` (\n  `id` int NULL\n) ENGINE=OLAP"}}, nil
		case strings.HasPrefix(query, "SHOW CREATE VIEW"):
			return []string{"View", "Create View", "character_set_client", "collation_connection"}, [][]driver.Value{{"v_orders", "CREATE VIEW `v_orders` AS SELECT `id` FROM `orders`", "utf8", "utf8_general_ci"}}, nil
		case strings.HasPrefix(query, "SHOW COLUMNS FROM"):
			return []string{"Field", "Type", "Null", "Key", "Default", "Extra"}, [][]driver.Value{{"id", "int", "YES", "", nil, ""}}, nil
		}
		return nil, nil, errors.Errorf("unexpected statement for %s: %q", engine, query)
	}
}

// dumpRecorder is registered once: database/sql panics on a repeated driver name, which
// would take the whole package down under `go test -count=2`. Each subtest resets it.
const dumpRecorderDriverName = "starrocks-dump-recorder"

var dumpRecorder recordingDriver

func init() {
	sql.Register(dumpRecorderDriverName, &dumpRecorder)
}

// recordingDriver is a database/sql driver that records every statement and answers it
// from a canned function, so Dump runs against a scripted server without a network.
type recordingDriver struct {
	mu      sync.Mutex
	queries []string
	answer  func(query string) ([]string, [][]driver.Value, error)
}

func (d *recordingDriver) Open(string) (driver.Conn, error) { return &recordingConn{d: d}, nil }

func (d *recordingDriver) reset(answer func(string) ([]string, [][]driver.Value, error)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.queries = nil
	d.answer = answer
}

func (d *recordingDriver) recorded() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.queries...)
}

type recordingConn struct{ d *recordingDriver }

func (*recordingConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("prepare is not supported")
}
func (*recordingConn) Close() error              { return nil }
func (*recordingConn) Begin() (driver.Tx, error) { return nil, errors.New("tx is not supported") }

// QueryContext makes database/sql skip Prepare for both db.Query(q) and db.Query(q, arg);
// the driver interpolates the argument itself, as the real DSN (interpolateParams=true) does.
func (c *recordingConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	for _, a := range args {
		query = strings.Replace(query, "?", fmt.Sprintf("'%v'", a.Value), 1)
	}
	c.d.mu.Lock()
	c.d.queries = append(c.d.queries, query)
	c.d.mu.Unlock()
	cols, rows, err := c.d.answer(query)
	if err != nil {
		return nil, err
	}
	return &sliceRows{cols: cols, rows: rows}, nil
}

type sliceRows struct {
	cols []string
	rows [][]driver.Value
	i    int
}

func (r *sliceRows) Columns() []string { return r.cols }
func (*sliceRows) Close() error        { return nil }
func (r *sliceRows) Next(dest []driver.Value) error {
	if r.i >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.i])
	r.i++
	return nil
}
