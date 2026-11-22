package oceanbase

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/component/sheet"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	dbutil "github.com/bytebase/bytebase/backend/plugin/db/util"
)

// columnExtraFake records the catalogue lookup and answers it.
type columnExtraFake struct {
	queries []string
	args    [][]driver.NamedValue
}

func (f *columnExtraFake) Connect(context.Context) (driver.Conn, error) {
	return &columnExtraConn{fake: f}, nil
}

func (f *columnExtraFake) Driver() driver.Driver { return columnExtraDriver{fake: f} }

type columnExtraDriver struct{ fake *columnExtraFake }

func (d columnExtraDriver) Open(string) (driver.Conn, error) {
	return d.fake.Connect(context.Background())
}

type columnExtraConn struct{ fake *columnExtraFake }

func (*columnExtraConn) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (*columnExtraConn) Close() error                        { return nil }
func (*columnExtraConn) Begin() (driver.Tx, error)           { return columnExtraTx{}, nil }

func (c *columnExtraConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.fake.queries = append(c.fake.queries, query)
	c.fake.args = append(c.fake.args, args)
	return &columnExtraRows{}, nil
}

type columnExtraTx struct{}

func (columnExtraTx) Commit() error   { return nil }
func (columnExtraTx) Rollback() error { return nil }

type columnExtraRows struct{ done bool }

func (*columnExtraRows) Columns() []string { return []string{"EXTRA"} }
func (*columnExtraRows) Close() error      { return nil }
func (r *columnExtraRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	dest[0] = "STORED GENERATED"
	r.done = true
	return nil
}

// TestOfflineDDLIssuesTheSameLookup proves the rule still binds the same three
// values into the same INFORMATION_SCHEMA select it ran through the *sql.DB.
func TestOfflineDDLIssuesTheSameLookup(t *testing.T) {
	fake := &columnExtraFake{}
	connection := sql.OpenDB(fake)
	defer connection.Close()

	adviceList, err := advisor.SQLReviewCheck(context.Background(), sheet.NewManager(), "ALTER TABLE tech_book DROP COLUMN name;", []*storepb.SQLReviewRule{{
		Type:   storepb.SQLReviewRule_STATEMENT_DISALLOW_OFFLINE_DDL,
		Level:  storepb.SQLReviewRule_WARNING,
		Engine: storepb.Engine_OCEANBASE,
	}}, advisor.Context{
		DBType:          storepb.Engine_OCEANBASE,
		CurrentDatabase: "test",
		Prober:          dbutil.NewSQLProber(connection),
		NoAppendBuiltin: true,
	})
	require.NoError(t, err)

	require.Equal(t, []string{"SELECT EXTRA FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND COLUMN_NAME = ?"}, fake.queries)
	require.Len(t, fake.args, 1)
	var values []any
	for _, arg := range fake.args[0] {
		values = append(values, arg.Value)
	}
	require.Equal(t, []any{"test", "tech_book", "name"}, values)

	require.Len(t, adviceList, 1)
	require.Equal(t, "Dropping stored generated columns is an offline DDL operation.", adviceList[0].Content)
}
